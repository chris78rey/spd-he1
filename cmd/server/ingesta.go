package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type stagedUpload struct {
	Field        string `json:"field"`
	StoredName   string `json:"stored_name"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size_bytes"`
	SHA256       string `json:"sha256,omitempty"`
}

type workspaceDocument struct {
	ID             string `json:"id"`
	StoredName     string `json:"stored_name"`
	OriginalName   string `json:"original_name"`
	RelativePath   string `json:"relative_path"`
	Size           int64  `json:"size_bytes"`
	PlanillaID     int64  `json:"pdi_id,omitempty"`
	CoverageCedula string `json:"coverage_cedula,omitempty"`
}

type stagedJob struct {
	ID                    string                    `json:"job_id"`
	Status                string                    `json:"status"`
	Username              string                    `json:"username"`
	Month                 string                    `json:"mes"`
	Year                  string                    `json:"anio"`
	Service               string                    `json:"tipo_servicio"`
	ReceivedAt            time.Time                 `json:"received_at"`
	Files                 []stagedUpload            `json:"files"`
	Aliases               []string                  `json:"legacy_ids,omitempty"`
	ExternalPDFs          []workspaceDocument       `json:"external_pdfs,omitempty"`
	Renames               map[string]string         `json:"renames,omitempty"`
	Replacements          map[string]string         `json:"replacements,omitempty"`
	MergedDuplicates      map[string][]string       `json:"merged_duplicates,omitempty"`
	DeletedPDFs           map[string]bool           `json:"deleted_pdfs,omitempty"`
	DocumentPlanillas     map[string]int64          `json:"document_planillas,omitempty"`
	TramiteMappings       map[string]string         `json:"tramite_mappings,omitempty"`
	CoverageFailures      map[int64]coverageFailure `json:"coverage_failures,omitempty"`
	IsObjections          bool                      `json:"es_objeciones,omitempty"`
	ObjectionSourceID     string                    `json:"expediente_origen,omitempty"`
	ObjectionRows         []objectionRecord         `json:"objeciones,omitempty"`
	ObjectionPDFSelection map[string]bool           `json:"objecion_pdf_selection,omitempty"`
	StatusDetail          string                    `json:"status_detail"`
}

var headerParts = []uploadPart{
	{field: "matriz_file", storedName: "matriz.xlsm", extension: ".xlsm"},
	{field: "consolidada_file", storedName: "consolidada.pdf", extension: ".pdf"},
	{field: "oficio_file", storedName: "oficio.pdf", extension: ".pdf"},
}

func missingHeaders(job stagedJob) []string {
	if job.IsObjections {
		return nil
	}
	present := make(map[string]bool, len(job.Files))
	for _, file := range job.Files {
		present[file.Field] = true
	}
	labels := map[string]string{"matriz_file": "Matriz de planillaje (.xlsm)", "consolidada_file": "Planilla consolidada (.pdf)", "oficio_file": "Oficio de pago (.pdf)"}
	missing := make([]string, 0, 3)
	for _, part := range headerParts {
		if !present[part.field] {
			missing = append(missing, labels[part.field])
		}
	}
	return missing
}

func jobResponse(s *server, job stagedJob, output string, summary any) map[string]any {
	missing := missingHeaders(job)
	status := job.Status
	if (status == "PROCESSED" || status == "INCOMPLETE") && len(missing) > 0 {
		status = "INCOMPLETE"
	}
	result := map[string]any{"status": status, "job_id": job.ID, "message": job.StatusDetail, "workspace": s.jobRoot(job.ID), "missing_documents": missing, "files": job.Files, "mes": job.Month, "anio": job.Year, "tipo_servicio": job.Service, "creado_por": job.Username}
	if job.IsObjections {
		result["es_objeciones"] = true
		result["expediente_origen"] = job.ObjectionSourceID
		rows := append([]objectionRecord(nil), job.ObjectionRows...)
		packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
		for index := range rows {
			coverage := filepath.Join(packageRoot, "4. EXPEDIENTES", rows[index].PatientFolder, "C_COBERTURA.pdf")
			rows[index].CoverageAttached = validPDFFile(coverage)
		}
		result["objeciones"] = rows
		result["cabeceras_objeciones"] = objectionHeaderStatuses(s, job)
	}
	if output != "" {
		result["output"] = output
	}
	if summary != nil {
		result["clasificacion"] = summary
	}
	return result
}

var legacyJobIDPattern = regexp.MustCompile(`^JOB-[0-9]{8}T[0-9]{6}-[a-f0-9]{16}$`)
var stableWorkspaceIDPattern = regexp.MustCompile(`^WORK-([A-Z0-9_]+)-([0-9]{6})$`)
var tramitePattern = regexp.MustCompile(`^[0-9]+$`)

func validJobID(id string) bool {
	return legacyJobIDPattern.MatchString(id) || stableWorkspaceIDPattern.MatchString(id)
}

func periodWorkspaceID(service, month, year string) string {
	return "WORK-" + service + "-" + year + month
}

type planillaIdentity struct {
	PlanillaID int64
	Tramite    string
	Cedula     string
	Patient    string
	Service    string
	CareFrom   time.Time
	CareUntil  time.Time
}

type ingestFolderPreview struct {
	Tramite         string `json:"tramite"`
	OracleTramite   string `json:"pdi_tramite_oracle,omitempty"`
	ManualMapping   bool   `json:"vinculo_manual,omitempty"`
	MappingConflict bool   `json:"conflicto_vinculo,omitempty"`
	PDFs            int    `json:"pdfs"`
	OracleMatch     bool   `json:"coincide_oracle"`
	PatientMissing  bool   `json:"paciente_faltante_oracle,omitempty"`
	Patient         string `json:"paciente,omitempty"`
	Service         string `json:"servicio_oracle,omitempty"`
	CareFrom        string `json:"fecha_desde,omitempty"`
	CareUntil       string `json:"fecha_hasta,omitempty"`
}

type ingestOracleCandidate struct {
	Tramite string `json:"tramite"`
	Title   string `json:"titulo"`
}

type ingestPreview struct {
	Service               string                  `json:"tipo_servicio"`
	Month                 string                  `json:"mes"`
	Year                  string                  `json:"anio"`
	OraclePlanillas       int                     `json:"planillas_oracle"`
	TramiteFolders        int                     `json:"carpetas_tramite"`
	PDFs                  int                     `json:"pdfs"`
	MatchedFolders        int                     `json:"tramites_en_oracle"`
	PDFsMatched           int                     `json:"pdfs_con_tramite_oracle"`
	UnmatchedFolders      int                     `json:"tramites_sin_oracle"`
	OracleWithoutZIP      int                     `json:"tramites_oracle_sin_zip"`
	OracleMissingTramites []string                `json:"pdi_tramite_oracle_sin_zip,omitempty"`
	OracleCandidates      []ingestOracleCandidate `json:"planillas_oracle_disponibles,omitempty"`
	FoldersWithoutPatient int                     `json:"tramites_sin_paciente_oracle"`
	InvalidEntries        int                     `json:"entradas_invalidas"`
	InvalidPaths          []string                `json:"rutas_invalidas,omitempty"`
	Folders               []ingestFolderPreview   `json:"carpetas"`
}

func (s *server) loadPlanillaIdentities(ctx context.Context, job stagedJob) (map[string]planillaIdentity, error) {
	table := oracleTableName(s.schema)
	query := "SELECT TO_CHAR(PDI_ID), PDI_TRAMITE, PDI_PACIENTE, TO_CHAR(PDI_FECHA_DESDE, 'YYYY-MM-DD'), TO_CHAR(PDI_FECHA_HASTA, 'YYYY-MM-DD'), PDI_SERVICIO, PDI_CEDULA FROM " + table + " WHERE PDI_MES = :mes AND PDI_ANIO = :anio AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP'"
	rows, err := s.serviceDB.QueryContext(ctx, query, sql.Named("mes", job.Month), sql.Named("anio", job.Year))
	if err != nil {
		return nil, fmt.Errorf("No se pudo consultar Oracle para el período %s/%s.", job.Month, job.Year)
	}
	defer rows.Close()
	identities := make(map[string]planillaIdentity)
	for rows.Next() {
		var rawPlanillaID, rawID, patient, rawCareFrom, rawCareUntil, service, cedula sql.NullString
		if err := rows.Scan(&rawPlanillaID, &rawID, &patient, &rawCareFrom, &rawCareUntil, &service, &cedula); err != nil {
			return nil, errors.New("No se pudo leer la identidad del paciente desde Oracle.")
		}
		tramite := strings.TrimSpace(rawID.String)
		if !rawID.Valid || !tramitePattern.MatchString(tramite) {
			continue
		}
		planillaID, parseErr := strconv.ParseInt(strings.TrimSpace(rawPlanillaID.String), 10, 64)
		if !rawPlanillaID.Valid || parseErr != nil || planillaID <= 0 {
			return nil, errors.New("Oracle devolvió un PDI_ID inválido para una planilla MSP.")
		}
		identities[tramite] = planillaIdentity{PlanillaID: planillaID, Tramite: tramite, Cedula: strings.TrimSpace(cedula.String), Patient: strings.TrimSpace(patient.String), Service: strings.TrimSpace(service.String), CareFrom: parseOracleDate(rawCareFrom), CareUntil: parseOracleDate(rawCareUntil)}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("Falló la lectura de pacientes desde Oracle.")
	}
	return identities, nil
}

func (s *server) inspectIngestZIP(ctx context.Context, job stagedJob) (ingestPreview, error) {
	preview := ingestPreview{Service: job.Service, Month: job.Month, Year: job.Year, Folders: []ingestFolderPreview{}}
	identities, err := s.loadPlanillaIdentities(ctx, job)
	if err != nil {
		return preview, err
	}
	preview.OraclePlanillas = len(identities)
	archive, err := zip.OpenReader(filepath.Join(s.jobSourcesDir(job.ID), "lote.zip"))
	if err != nil {
		return preview, errors.New("No se pudo abrir el ZIP guardado para generar la vista previa.")
	}
	defer archive.Close()
	folders := make(map[string]*ingestFolderPreview)
	addInvalid := func(name string) {
		preview.InvalidEntries++
		if len(preview.InvalidPaths) < 12 {
			preview.InvalidPaths = append(preview.InvalidPaths, name)
		}
	}
	for _, entry := range archive.File {
		clean := strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/")
		if clean == "" {
			continue
		}
		parts := strings.Split(clean, "/")
		isPDF := strings.EqualFold(filepath.Ext(parts[len(parts)-1]), ".pdf")
		if !entry.FileInfo().IsDir() && isPDF {
			preview.PDFs++
		}
		if len(parts) > 0 && tramitePattern.MatchString(parts[0]) {
			if _, exists := folders[parts[0]]; !exists {
				folders[parts[0]] = &ingestFolderPreview{Tramite: parts[0]}
			}
			if !entry.FileInfo().IsDir() && isPDF {
				folders[parts[0]].PDFs++
			}
		}
		if entry.FileInfo().IsDir() {
			if len(parts) != 1 || !tramitePattern.MatchString(parts[0]) {
				addInvalid(entry.Name)
			}
			continue
		}
		if len(parts) != 2 || !tramitePattern.MatchString(parts[0]) || !isPDF || path.Base(parts[1]) != parts[1] {
			addInvalid(entry.Name)
			continue
		}
		file, openErr := entry.Open()
		if openErr != nil {
			addInvalid(entry.Name)
			continue
		}
		var signature [5]byte
		_, readErr := io.ReadFull(file, signature[:])
		_ = file.Close()
		if readErr != nil || string(signature[:]) != "%PDF-" {
			addInvalid(entry.Name)
		}
	}
	preview.TramiteFolders = len(folders)
	usedOracleTramites := make(map[string]string, len(folders))
	for source, folder := range folders {
		target := strings.TrimSpace(job.TramiteMappings[source])
		if target == "" {
			target = source
		}
		folder.OracleTramite = target
		folder.ManualMapping = target != source
		if identity, found := identities[target]; found {
			if usedBy, exists := usedOracleTramites[target]; exists && usedBy != source {
				folder.MappingConflict = true
				preview.UnmatchedFolders++
				preview.Folders = append(preview.Folders, *folder)
				continue
			}
			usedOracleTramites[target] = source
			folder.OracleMatch = true
			folder.Patient = identity.Patient
			folder.PatientMissing = identity.Patient == ""
			folder.Service = identity.Service
			if !identity.CareFrom.IsZero() {
				folder.CareFrom = identity.CareFrom.Format("2006-01-02")
			}
			if !identity.CareUntil.IsZero() {
				folder.CareUntil = identity.CareUntil.Format("2006-01-02")
			}
			preview.MatchedFolders++
			preview.PDFsMatched += folder.PDFs
			if folder.PatientMissing {
				preview.FoldersWithoutPatient++
			}
		} else {
			preview.UnmatchedFolders++
		}
		preview.Folders = append(preview.Folders, *folder)
	}
	for tramite, identity := range identities {
		if _, included := usedOracleTramites[tramite]; included {
			continue
		}
		preview.OracleMissingTramites = append(preview.OracleMissingTramites, tramite)
		patient := normalizePatientFolder(identity.Patient)
		title := tramite
		if patient != "" {
			title += " · " + patient
		}
		preview.OracleCandidates = append(preview.OracleCandidates, ingestOracleCandidate{Tramite: tramite, Title: title})
	}
	sort.Strings(preview.OracleMissingTramites)
	sort.Slice(preview.OracleCandidates, func(i, j int) bool { return preview.OracleCandidates[i].Tramite < preview.OracleCandidates[j].Tramite })
	preview.OracleWithoutZIP = len(preview.OracleMissingTramites)
	sort.Slice(preview.Folders, func(i, j int) bool { return preview.Folders[i].Tramite < preview.Folders[j].Tramite })
	return preview, nil
}

func (s *server) getIngestPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/previsualizar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el lote para previsualizar.")
		return
	}
	if !s.hasClinicalSource(job) {
		writeError(w, http.StatusGone, "Este lote ya no conserva su ZIP fuente.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	preview, err := s.inspectIngestZIP(ctx, job)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vista_previa": preview})
}

// setIngestTramiteMapping lets an operator associate an unmatched ZIP folder
// with an Oracle PDI_TRAMITE that has no folder in this ZIP.
func (s *server) setIngestTramiteMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/vincular-tramite/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var input struct {
		ZIPTramite    string `json:"tramite_zip"`
		OracleTramite string `json:"tramite_oracle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || !tramitePattern.MatchString(strings.TrimSpace(input.ZIPTramite)) {
		writeError(w, http.StatusBadRequest, "El trámite de la carpeta ZIP no es válido.")
		return
	}
	input.ZIPTramite = strings.TrimSpace(input.ZIPTramite)
	input.OracleTramite = strings.TrimSpace(input.OracleTramite)
	if input.OracleTramite != "" && !tramitePattern.MatchString(input.OracleTramite) {
		writeError(w, http.StatusBadRequest, "El trámite Oracle seleccionado no es válido.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !s.hasClinicalSource(job) {
		writeError(w, http.StatusNotFound, "No se encontró el ZIP guardado para este período.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	preview, err := s.inspectIngestZIP(ctx, job)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	var source *ingestFolderPreview
	for index := range preview.Folders {
		if preview.Folders[index].Tramite == input.ZIPTramite {
			source = &preview.Folders[index]
			break
		}
	}
	if source == nil {
		writeError(w, http.StatusNotFound, "La carpeta indicada ya no está dentro del ZIP.")
		return
	}
	if input.OracleTramite == "" {
		if job.TramiteMappings == nil || job.TramiteMappings[input.ZIPTramite] == "" {
			writeError(w, http.StatusConflict, "La carpeta no tiene una corrección manual que quitar.")
			return
		}
		delete(job.TramiteMappings, input.ZIPTramite)
		if len(job.TramiteMappings) == 0 {
			job.TramiteMappings = nil
		}
		if err := s.saveStagedJob(job); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo guardar la corrección del trámite.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Se quitó la corrección manual. La carpeta volverá a usar el PDI_TRAMITE de su nombre original."})
		return
	}
	if source.OracleMatch && !source.ManualMapping {
		writeError(w, http.StatusConflict, "Esta carpeta ya coincide con Oracle. Solo puedes corregir carpetas sin coincidencia o cambiar una corrección manual existente.")
		return
	}
	available := false
	for _, candidate := range preview.OracleMissingTramites {
		if candidate == input.OracleTramite {
			available = true
			break
		}
	}
	if !available {
		writeError(w, http.StatusConflict, "El trámite Oracle seleccionado ya tiene otra carpeta asociada o no pertenece a este período MSP.")
		return
	}
	if job.TramiteMappings == nil {
		job.TramiteMappings = make(map[string]string)
	}
	job.TramiteMappings[input.ZIPTramite] = input.OracleTramite
	if err := s.saveStagedJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la corrección del trámite.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Se guardó la asociación manual. Revisa la vista previa; si el lote ya estaba preparado, vuelve a analizar los PDFs para aplicar el cambio."})
}

// replaceStagedZIP lets an operator correct the clinical ZIP after Oracle preview
// has exposed mismatched routes or trámites, without losing the saved period.
func (s *server) replaceStagedZIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/reemplazar-zip/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxIngest)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el ZIP corregido.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	files := r.MultipartForm.File["zip_file"]
	if len(files) != 1 || files[0].Size == 0 || !strings.EqualFold(filepath.Ext(files[0].Filename), ".zip") {
		writeError(w, http.StatusBadRequest, "Selecciona un archivo ZIP no vacío.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el expediente guardado.")
		return
	}
	if job.Status != "STAGED" && job.Status != "REQUIERE_REVISION" {
		writeError(w, http.StatusConflict, "Solo puedes cambiar el ZIP antes de preparar el expediente.")
		return
	}
	sourceRoot := s.jobSourcesDir(id)
	if err := os.MkdirAll(sourceRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el almacenamiento privado del ZIP.")
		return
	}
	temp, err := os.CreateTemp(sourceRoot, ".lote-nuevo-*.zip")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carga del ZIP.")
		return
	}
	tempPath := temp.Name()
	_ = temp.Close()
	if err := os.Remove(tempPath); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carga del ZIP.")
		return
	}
	defer os.Remove(tempPath)
	if _, err := saveUploadPart(files[0], tempPath); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el ZIP corregido.")
		return
	}
	archive, err := zip.OpenReader(tempPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "El archivo seleccionado no es un ZIP válido.")
		return
	}
	_ = archive.Close()
	digest, err := sha256Path(tempPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo verificar el ZIP corregido.")
		return
	}
	currentPath := filepath.Join(sourceRoot, "lote.zip")
	backupPath := filepath.Join(sourceRoot, fmt.Sprintf("lote-anterior-%s.zip", time.Now().UTC().Format("20060102T150405.000000000")))
	if _, err := os.Stat(currentPath); err == nil {
		if err := os.Rename(currentPath, backupPath); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo conservar la versión anterior del ZIP.")
			return
		}
	}
	if err := os.Rename(tempPath, currentPath); err != nil {
		_ = os.Rename(backupPath, currentPath)
		writeError(w, http.StatusInternalServerError, "No se pudo activar el ZIP corregido.")
		return
	}
	found := false
	for index := range job.Files {
		if job.Files[index].Field == "zip_file" {
			job.Files[index].StoredName = "lote.zip"
			job.Files[index].OriginalName = safeOriginalFilename(files[0].Filename)
			job.Files[index].Size = files[0].Size
			job.Files[index].SHA256 = digest
			found = true
			break
		}
	}
	if !found {
		_ = os.Remove(currentPath)
		_ = os.Rename(backupPath, currentPath)
		writeError(w, http.StatusConflict, "El expediente guardado no tiene un ZIP fuente registrado.")
		return
	}
	job.Status = "STAGED"
	job.StatusDetail = "ZIP actualizado en el mismo espacio. Revisa el cruce con Oracle antes de preparar los expedientes."
	if err := s.saveStagedJob(job); err != nil {
		_ = os.Remove(currentPath)
		_ = os.Rename(backupPath, currentPath)
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el ZIP corregido; se restauró el anterior.")
		return
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, "", nil))
}

func parseOracleDate(value sql.NullString) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value.String))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func summarizeClassification(files []classificationResult) classificationSummary {
	var summary classificationSummary
	for _, file := range files {
		summary.Total++
		if file.Reason == "FUSION_PENDIENTE" {
			summary.FusionPending++
		}
		if strings.HasPrefix(file.Code, "PENDIENTE_") {
			summary.Pending++
		} else {
			summary.Classified++
		}
		switch file.Method {
		case "VECTORIAL":
			summary.Vector++
		case "OCR_TESSERACT":
			summary.OCR++
		default:
			summary.Unreadable++
		}
		if len(file.DatesOutsidePeriod) > 0 {
			summary.PeriodAlertFiles = append(summary.PeriodAlertFiles, periodDateWarning{Document: file.Output, Dates: file.DatesOutsidePeriod, Period: file.BilledPeriod})
		}
	}
	return summary
}

type classificationReport struct {
	Summary classificationSummary  `json:"resumen"`
	Files   []classificationResult `json:"archivos"`
}

// processStagedJob resolves numeric ZIP folders using the selected period in Oracle,
// then writes the official patient-name folder layout into a private processed job.
func (s *server) processStagedJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/procesar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el expediente por ese ID.")
		return
	}
	if job.Status != "STAGED" {
		writeError(w, http.StatusConflict, "El lote ya fue procesado o está en otro estado.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	if err := s.ensureJobWorkspace(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el espacio permanente del lote.")
		return
	}
	outputRoot := filepath.Join(s.jobRoot(job.ID), "trabajo")
	report, err := s.buildPatientFolders(ctx, &job, outputRoot)
	if err != nil {
		job.Status = "REQUIERE_REVISION"
		job.StatusDetail = err.Error()
		_ = s.saveStagedJob(job)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	job.Status = "PROCESSED"
	if len(missingHeaders(job)) > 0 {
		job.Status = "INCOMPLETE"
	}
	job.StatusDetail = fmt.Sprintf("Lote preparado para revisión. %d PDF clasificados y %d pendientes. Los documentos habilitantes pueden añadirse después.", report.Summary.Classified, report.Summary.Pending)
	if err := s.saveStagedJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "Se procesó el lote, pero no se pudo guardar su estado.")
		return
	}
	packageRoot := filepath.Join(outputRoot, packageFolderName(&job))
	if err := s.syncOracleWorkspace(ctx, &job, packageRoot); err != nil {
		log.Printf("ingesta %s: paquete guardado localmente; falló índice Oracle: %v", job.ID, err)
		job.StatusDetail = "Los archivos se prepararon en el espacio privado, pero no se pudo terminar el registro en Oracle. Vuelve a preparar el expediente cuando Oracle esté disponible."
		_ = s.saveStagedJob(job)
		writeError(w, http.StatusInternalServerError, job.StatusDetail)
		return
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, filepath.Join(outputRoot, packageFolderName(&job)), report.Summary))
}

func (s *server) reclassifyStagedJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/clasificar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró este lote en el espacio permanente. Si se borraron los datos de prueba, vuelve a recibir el ZIP clínico para crear un expediente nuevo.")
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Solo se puede actualizar un expediente procesado que conserve sus archivos fuente.")
		return
	}
	if !s.hasClinicalSource(job) {
		writeError(w, http.StatusGone, "El expediente no conserva el ZIP clínico fuente. Vuelve a recibir el ZIP para crear un expediente nuevo.")
		return
	}
	if err := s.ensureJobWorkspace(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el espacio permanente del lote.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	outputRoot := filepath.Join(s.jobRoot(job.ID), "trabajo")
	report, err := s.buildPatientFolders(ctx, &job, outputRoot)
	if err != nil {
		log.Printf("ingesta %s: falló reclasificación OCR: %v", job.ID, err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := s.syncOracleWorkspace(ctx, &job, filepath.Join(outputRoot, packageFolderName(&job))); err != nil {
		log.Printf("ingesta %s: falló sincronización Oracle tras reclasificar: %v", job.ID, err)
		writeError(w, http.StatusInternalServerError, "Los archivos se actualizaron, pero no se pudo sincronizar el registro Oracle.")
		return
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, filepath.Join(outputRoot, packageFolderName(&job)), report.Summary))
}

func (s *server) hasClinicalSource(job stagedJob) bool {
	if job.IsObjections {
		return job.Status == "PROCESSED" && len(job.ObjectionRows) > 0
	}
	for _, file := range job.Files {
		if file.Field != "zip_file" || file.StoredName != "lote.zip" {
			continue
		}
		info, err := os.Stat(filepath.Join(s.jobSourcesDir(job.ID), file.StoredName))
		return err == nil && info.Mode().IsRegular()
	}
	return false
}

func (s *server) completeStagedJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/completar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el expediente por ese ID.")
		return
	}
	id = job.ID
	if job.Status != "STAGED" && job.Status != "PROCESSED" && job.Status != "INCOMPLETE" && job.Status != "REQUIERE_REVISION" {
		writeError(w, http.StatusConflict, "Este lote no admite documentos adicionales en su estado actual.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxIngest)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "Los documentos superan el límite de carga configurado.")
			return
		}
		writeError(w, http.StatusBadRequest, "No se pudo leer la carga multipart.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err := s.ensureJobWorkspace(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo abrir el espacio permanente del lote.")
		return
	}
	added := 0
	for _, part := range headerParts {
		files := r.MultipartForm.File[part.field]
		if len(files) == 0 {
			continue
		}
		if len(files) != 1 || files[0].Size == 0 || !strings.EqualFold(filepath.Ext(files[0].Filename), part.extension) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Adjunta un archivo válido en %s (%s, no vacío).", part.field, part.extension))
			return
		}
		part.file = files[0]
		tempFile, err := os.CreateTemp(s.jobSourcesDir(id), ".habilitante-nuevo-")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo preparar el reemplazo del documento habilitante.")
			return
		}
		tempPath := tempFile.Name()
		source, openErr := part.file.Open()
		var size int64
		if openErr == nil {
			size, err = io.Copy(tempFile, source)
			_ = source.Close()
		} else {
			err = openErr
		}
		if closeErr := tempFile.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(tempPath)
			writeError(w, http.StatusInternalServerError, "No se pudo guardar el documento habilitante.")
			return
		}
		if err := validateStagedFile(tempPath, part.extension); err != nil {
			_ = os.Remove(tempPath)
			writeError(w, http.StatusBadRequest, fmt.Sprintf("El archivo %s no tiene contenido válido.", part.field))
			return
		}
		staged := stagedUpload{Field: part.field, StoredName: part.storedName, OriginalName: safeOriginalFilename(part.file.Filename), Size: size}
		if err := s.replaceHeaderFile(&job, staged, tempPath); err != nil {
			_ = os.Remove(tempPath)
			writeError(w, http.StatusInternalServerError, "No se pudo reemplazar el documento habilitante.")
			return
		}
		added++
	}
	if added == 0 {
		writeError(w, http.StatusBadRequest, "Selecciona al menos uno de los documentos habilitantes pendientes.")
		return
	}
	missing := missingHeaders(job)
	if job.Status == "REQUIERE_REVISION" {
		job.StatusDetail = "Documentos habilitantes guardados. El lote sigue requiriendo revisión."
	} else if len(missing) == 0 && job.Status != "STAGED" {
		job.Status = "PROCESSED"
		job.StatusDetail = "Lote y documentos habilitantes guardados en el mismo espacio permanente."
	} else if job.Status != "STAGED" {
		job.Status = "INCOMPLETE"
		job.StatusDetail = "Documentos guardados. Aún faltan documentos habilitantes para completar el lote."
	}
	if err := s.saveStagedJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "Se guardaron los archivos, pero no se pudo actualizar el expediente.")
		return
	}
	output := ""
	if job.Status == "PROCESSED" || job.Status == "INCOMPLETE" {
		output = filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, output, nil))
}

// replaceHeaderFile swaps a validated source and, when the expediente is already
// prepared, its copy in the work folder. Temporary backups let us restore both
// copies if the metadata update fails.
func (s *server) replaceHeaderFile(job *stagedJob, staged stagedUpload, tempSource string) error {
	sourcePath := filepath.Join(s.jobSourcesDir(job.ID), staged.StoredName)
	outputPath := ""
	var tempOutput string
	if job.Status == "PROCESSED" || job.Status == "INCOMPLETE" {
		outputPath = filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(job), headerOutputFilename(job, staged.Field))
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			return err
		}
		file, err := os.CreateTemp(outputDir, ".habilitante-nuevo-")
		if err != nil {
			return err
		}
		tempOutput = file.Name()
		input, err := os.Open(tempSource)
		if err == nil {
			_, err = io.Copy(file, input)
			_ = input.Close()
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(tempOutput)
			return err
		}
	}
	defer func() {
		if tempOutput != "" {
			_ = os.Remove(tempOutput)
		}
	}()

	sourceBackup, err := backupAndReplace(sourcePath, tempSource)
	if err != nil {
		return err
	}
	var outputBackup string
	if outputPath != "" {
		outputBackup, err = backupAndReplace(outputPath, tempOutput)
		if err != nil {
			if sourceBackup != "" {
				_ = os.Remove(sourcePath)
				_ = os.Rename(sourceBackup, sourcePath)
			}
			return err
		}
	}
	oldFiles := append([]stagedUpload(nil), job.Files...)
	found := false
	for index := range job.Files {
		if job.Files[index].Field == staged.Field {
			job.Files[index] = staged
			found = true
			break
		}
	}
	if !found {
		job.Files = append(job.Files, staged)
	}
	if err := s.saveStagedJob(*job); err != nil {
		job.Files = oldFiles
		_ = os.Remove(sourcePath)
		if sourceBackup != "" {
			_ = os.Rename(sourceBackup, sourcePath)
		}
		if outputPath != "" {
			_ = os.Remove(outputPath)
			if outputBackup != "" {
				_ = os.Rename(outputBackup, outputPath)
			}
		}
		return err
	}
	if sourceBackup != "" {
		_ = os.Remove(sourceBackup)
	}
	if outputBackup != "" {
		_ = os.Remove(outputBackup)
	}
	return nil
}

func backupAndReplace(destination, replacement string) (string, error) {
	backup := ""
	if _, err := os.Stat(destination); err == nil {
		file, createErr := os.CreateTemp(filepath.Dir(destination), ".habilitante-anterior-")
		if createErr != nil {
			return "", createErr
		}
		backup = file.Name()
		if err := file.Close(); err != nil {
			_ = os.Remove(backup)
			return "", err
		}
		if err := os.Remove(backup); err != nil {
			return "", err
		}
		if err := os.Rename(destination, backup); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.Rename(replacement, destination); err != nil {
		if backup != "" {
			_ = os.Rename(backup, destination)
		}
		return "", err
	}
	return backup, nil
}

func (s *server) getStagedJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/ingesta/estado/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del lote no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró este lote en el espacio permanente.")
		return
	}
	if !s.hasClinicalSource(job) {
		writeError(w, http.StatusGone, "Este lote ya no conserva el ZIP clínico fuente.")
		return
	}
	output := ""
	var summary any
	if job.Status == "PROCESSED" || job.Status == "INCOMPLETE" {
		output = filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
		if !job.IsObjections {
			if reportData, readErr := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json")); readErr == nil {
				var report classificationReport
				if json.Unmarshal(reportData, &report) == nil {
					summary = report.Summary
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, output, summary))
}

func (s *server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	directories, err := os.ReadDir(s.workspacesDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "No se pudieron leer los espacios guardados.")
		return
	}
	items := make([]map[string]any, 0, len(directories))
	for _, directory := range directories {
		if !directory.IsDir() || strings.HasPrefix(directory.Name(), ".") {
			continue
		}
		job, loadErr := s.loadStagedJob(directory.Name())
		if loadErr != nil || !s.hasClinicalSource(job) {
			continue
		}
		response := jobResponse(s, job, "", nil)
		response["period"] = job.Year + "-" + job.Month
		response["received_at"] = job.ReceivedAt
		items = append(items, response)
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["period"]) > fmt.Sprint(items[j]["period"])
	})
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (s *server) buildPatientFolders(ctx context.Context, job *stagedJob, outputRoot string) (classificationReport, error) {
	var report classificationReport
	identities, err := s.loadPlanillaIdentities(ctx, *job)
	if err != nil {
		return report, err
	}
	if len(identities) == 0 {
		return report, errors.New("Oracle no devolvió planillas MSP del período seleccionado con PDI_PLANILLADO = 'S'.")
	}

	stage := s.jobSourcesDir(job.ID)
	zipPath := filepath.Join(stage, "lote.zip")
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return report, errors.New("No se pudo abrir el ZIP del lote.")
	}
	defer archive.Close()
	jobRoot := s.jobRoot(job.ID)
	if err := os.MkdirAll(jobRoot, 0700); err != nil {
		return report, errors.New("No se pudo preparar el espacio privado del lote.")
	}
	tempOutput, err := os.MkdirTemp(jobRoot, ".processing-")
	if err != nil {
		return report, errors.New("No se pudo preparar el espacio privado del lote.")
	}
	defer os.RemoveAll(tempOutput)
	if err := os.MkdirAll(filepath.Join(tempOutput, packageFolderName(job), "4. EXPEDIENTES"), 0700); err != nil {
		return report, errors.New("No se pudo preparar la salida privada del lote.")
	}
	tempPackageRoot := filepath.Join(tempOutput, packageFolderName(job))
	for _, upload := range job.Files {
		canonicalName := ""
		switch upload.Field {
		case "oficio_file":
			canonicalName = "1. OFICIO DE PAGO.pdf"
		case "consolidada_file":
			canonicalName = "2. PLANILLA CONSOLIDADA.pdf"
		case "matriz_file":
			canonicalName = matrixFilename(job)
		}
		if canonicalName == "" {
			continue
		}
		if err := copyPrivateFile(filepath.Join(stage, upload.StoredName), filepath.Join(tempPackageRoot, canonicalName)); err != nil {
			return report, errors.New("No se pudo copiar uno de los documentos habilitantes.")
		}
	}
	rules, err := loadClassificationRules("reglas_clasificacion.yaml")
	if err != nil {
		return report, err
	}
	report.Files = make([]classificationResult, 0, len(archive.File))
	seen := 0
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if entry.UncompressedSize64 > 256<<20 {
			return report, errors.New("Un PDF del ZIP excede el límite de 256 MiB por archivo.")
		}
		clean := strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/")
		parts := strings.Split(clean, "/")
		if len(parts) != 2 || !tramitePattern.MatchString(parts[0]) || !strings.EqualFold(filepath.Ext(parts[1]), ".pdf") || path.Base(parts[1]) != parts[1] {
			return report, errors.New("El ZIP debe contener solo archivos PDF dentro de carpetas cuyo nombre sea el PDI_TRAMITE numérico.")
		}
		oracleTramite := strings.TrimSpace(job.TramiteMappings[parts[0]])
		if oracleTramite == "" {
			oracleTramite = parts[0]
		}
		identity, found := identities[oracleTramite]
		if !found {
			return report, fmt.Errorf("El trámite %s de la carpeta %s no aparece como planillado en Oracle para %s/%s.", oracleTramite, parts[0], job.Month, job.Year)
		}
		patientFolder := normalizePatientFolder(identity.Patient)
		if patientFolder == "" {
			return report, fmt.Errorf("Oracle no tiene un nombre de paciente utilizable para el trámite %s.", parts[0])
		}
		folder := filepath.Join(tempPackageRoot, "4. EXPEDIENTES", patientFolder)
		if err := os.MkdirAll(folder, 0700); err != nil {
			return report, errors.New("No se pudo crear una carpeta de expediente.")
		}
		file, err := entry.Open()
		if err != nil {
			return report, errors.New("No se pudo leer un PDF del ZIP.")
		}
		var signature [5]byte
		if _, err := io.ReadFull(file, signature[:]); err != nil || string(signature[:]) != "%PDF-" {
			_ = file.Close()
			return report, fmt.Errorf("El archivo %s no tiene contenido PDF válido.", parts[1])
		}
		workFile, err := os.CreateTemp(tempOutput, ".document-*.pdf")
		if err == nil {
			_, err = copyWithLimit(workFile, io.MultiReader(bytes.NewReader(signature[:]), file), 256<<20)
			closeErr := workFile.Close()
			if err == nil {
				err = closeErr
			}
		}
		_ = file.Close()
		if err != nil {
			if workFile != nil {
				_ = os.Remove(workFile.Name())
			}
			return report, errors.New("No se pudo guardar un PDF del expediente.")
		}
		originalName := filepath.Base(parts[1])
		classified := classifyPDF(ctx, workFile.Name(), originalName, rules, job.Month, job.Year)
		classified.PlanillaID = identity.PlanillaID
		classified.Tramite = oracleTramite
		classified.ZIPTramite = parts[0]
		classifiedName := classified.Code
		if classifiedName == "" {
			classifiedName = pendingPDFName(originalName)
			classified.Reason = "SIN_COINCIDENCIA"
		}
		destination := filepath.Join(folder, classifiedName)
		if classified.Code != "" {
			if _, statErr := os.Stat(destination); statErr == nil {
				uniqueName, nameErr := nextMSPPDFName(folder, classifiedName, "")
				if nameErr != nil {
					_ = os.Remove(workFile.Name())
					return report, errors.New("No se pudo reservar un nombre para el PDF duplicado.")
				}
				classified.Code = uniqueName
				classified.Reason = "FUSION_PENDIENTE"
				destination = filepath.Join(folder, uniqueName)
			}
		}
		if err := os.Rename(workFile.Name(), destination); err != nil {
			_ = os.Remove(workFile.Name())
			return report, errors.New("No se pudo guardar el PDF clasificado.")
		}
		classified.Output = filepath.ToSlash(filepath.Join("4. EXPEDIENTES", patientFolder, classified.Code))
		report.Files = append(report.Files, classified)
		report.Summary.Total++
		seen++
	}
	if seen == 0 {
		return report, errors.New("El ZIP no contiene PDFs de pacientes.")
	}
	for _, external := range job.ExternalPDFs {
		if _, replaced := job.Replacements[filepath.ToSlash(external.RelativePath)]; replaced {
			continue
		}
		source, sourceErr := safeWorkspacePath(filepath.Join(stage, "externos"), external.StoredName)
		destination, destinationErr := safeWorkspacePath(tempPackageRoot, external.RelativePath)
		if sourceErr != nil || destinationErr != nil {
			return report, errors.New("Un PDF externo tiene una ruta inválida.")
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			return report, errors.New("No se pudo crear la carpeta para un PDF externo.")
		}
		if err := copyPrivateFile(source, destination); err != nil {
			return report, errors.New("No se pudo conservar un PDF externo en la reclasificación.")
		}
	}
	for original, renamed := range job.Renames {
		source, sourceErr := safeWorkspacePath(tempPackageRoot, original)
		destination, destinationErr := safeWorkspacePath(tempPackageRoot, renamed)
		if sourceErr != nil || destinationErr != nil {
			return report, errors.New("Un cambio manual de nombre tiene una ruta inválida.")
		}
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return report, errors.New("No se pudo comprobar un PDF renombrado.")
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			return report, errors.New("No se pudo preparar la carpeta del PDF renombrado.")
		}
		if _, err := os.Stat(destination); err == nil {
			return report, errors.New("Un nombre corregido coincide con otro PDF; resuelve el duplicado antes de reclasificar.")
		}
		if err := os.Rename(source, destination); err != nil {
			return report, errors.New("No se pudo conservar un nombre corregido durante la reclasificación.")
		}
		for index := range report.Files {
			if report.Files[index].Output == filepath.ToSlash(original) {
				report.Files[index].Output = filepath.ToSlash(renamed)
				report.Files[index].Code = filepath.Base(renamed)
			}
		}
	}
	for relative, storedName := range job.Replacements {
		source, sourceErr := safeWorkspacePath(filepath.Join(stage, "externos"), storedName)
		destination, destinationErr := safeWorkspacePath(tempPackageRoot, relative)
		if sourceErr != nil || destinationErr != nil {
			return report, errors.New("Un reemplazo de PDF tiene una ruta inválida.")
		}
		if err := copyReplacePrivateFile(source, destination); err != nil {
			return report, errors.New("No se pudo conservar un PDF reemplazado durante la reclasificación.")
		}
	}
	for canonical, duplicates := range job.MergedDuplicates {
		for index := range report.Files {
			if report.Files[index].Output == canonical {
				report.Files[index].Reason = "FUSION_APLICADA"
			}
		}
		for _, relative := range duplicates {
			destination, pathErr := safeWorkspacePath(tempPackageRoot, relative)
			if pathErr != nil {
				return report, errors.New("Un PDF de una fusión guardada tiene una ruta inválida.")
			}
			if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
				return report, errors.New("No se pudo aplicar una fusión guardada durante la reclasificación.")
			}
		}
	}
	for relative, deleted := range job.DeletedPDFs {
		if !deleted {
			continue
		}
		destination, pathErr := safeWorkspacePath(tempPackageRoot, relative)
		if pathErr != nil {
			return report, errors.New("Un PDF marcado para quitar tiene una ruta inválida.")
		}
		if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
			return report, errors.New("No se pudo quitar un PDF durante la reclasificación.")
		}
	}
	filtered := report.Files[:0]
	for _, result := range report.Files {
		mergedAway := false
		for _, paths := range job.MergedDuplicates {
			for _, path := range paths {
				if result.Output == path {
					mergedAway = true
					break
				}
			}
			if mergedAway {
				break
			}
		}
		if !job.DeletedPDFs[result.Output] && !mergedAway {
			filtered = append(filtered, result)
		}
	}
	report.Files = filtered
	report.Summary = summarizeClassification(report.Files)
	if job.DocumentPlanillas == nil {
		job.DocumentPlanillas = make(map[string]int64)
	}
	for _, item := range report.Files {
		if item.Output != "" && item.PlanillaID > 0 {
			job.DocumentPlanillas[filepath.ToSlash(item.Output)] = item.PlanillaID
		}
	}
	tempReports, err := os.MkdirTemp(jobRoot, ".reportes-")
	if err != nil {
		return report, errors.New("No se pudo preparar el directorio de reportes.")
	}
	defer os.RemoveAll(tempReports)
	if err := writePrivateJSON(filepath.Join(tempReports, "classification_report.json"), report); err != nil {
		return report, errors.New("No se pudo guardar el reporte de clasificación.")
	}
	if err := replaceWorkspaceOutputs(tempOutput, tempReports, outputRoot, filepath.Join(jobRoot, "reportes")); err != nil {
		return report, errors.New("No se pudo publicar la salida del lote.")
	}
	return report, nil
}

func writePrivateJSON(filePath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

func atomicWritePrivateFile(destination string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".folio-write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func (s *server) jobRoot(id string) string {
	if match := stableWorkspaceIDPattern.FindStringSubmatch(id); match != nil {
		period := match[2]
		job := stagedJob{Service: match[1], Year: period[:4], Month: period[4:]}
		return filepath.Join(s.workspacesDir, packageFolderName(&job))
	}
	return filepath.Join(s.workspacesDir, id)
}

func (s *server) jobSourcesDir(id string) string {
	current := filepath.Join(s.jobRoot(id), "fuentes")
	if info, err := os.Stat(current); err == nil && info.IsDir() {
		return current
	}
	return filepath.Join(s.stagingDir, id)
}

// ensureJobWorkspace imports legacy staging inputs by copying them. Existing staging
// and processed folders remain untouched so a failed migration can always be retried.
func (s *server) ensureJobWorkspace(job stagedJob) error {
	root := s.jobRoot(job.ID)
	if _, err := os.Stat(filepath.Join(root, "job.json")); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	legacyRoot := filepath.Join(s.stagingDir, job.ID)
	legacyMetadata, err := os.ReadFile(filepath.Join(legacyRoot, "job.json"))
	if err != nil {
		return fmt.Errorf("no se encontró metadata del lote legado: %w", err)
	}
	if err := os.MkdirAll(s.workspacesDir, 0700); err != nil {
		return err
	}
	tempRoot, err := os.MkdirTemp(s.workspacesDir, ".migrating-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempRoot)
	sourceRoot := filepath.Join(tempRoot, "fuentes")
	if err := os.Mkdir(sourceRoot, 0700); err != nil {
		return err
	}
	for _, upload := range job.Files {
		if err := copyPrivateFile(filepath.Join(legacyRoot, upload.StoredName), filepath.Join(sourceRoot, upload.StoredName)); err != nil {
			return fmt.Errorf("no se pudo copiar una fuente del lote: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(tempRoot, "job.json"), legacyMetadata, 0600); err != nil {
		return err
	}
	if err := os.Rename(tempRoot, root); err != nil {
		if _, statErr := os.Stat(filepath.Join(root, "job.json")); statErr == nil {
			return nil
		}
		return err
	}
	return nil
}

func headerOutputFilename(job *stagedJob, field string) string {
	switch field {
	case "oficio_file":
		return "1. OFICIO DE PAGO.pdf"
	case "consolidada_file":
		return "2. PLANILLA CONSOLIDADA.pdf"
	case "matriz_file":
		return matrixFilename(job)
	default:
		return ""
	}
}

func replaceWorkspaceOutputs(workSource, reportSource, workDestination, reportDestination string) error {
	root := filepath.Dir(workDestination)
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	workBackup := filepath.Join(root, ".trabajo-prev-"+suffix)
	reportBackup := filepath.Join(root, ".reportes-prev-"+suffix)
	hadWork, hadReports := false, false
	if _, err := os.Stat(workDestination); err == nil {
		if err := os.Rename(workDestination, workBackup); err != nil {
			return err
		}
		hadWork = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(reportDestination); err == nil {
		if err := os.Rename(reportDestination, reportBackup); err != nil {
			if hadWork {
				_ = os.Rename(workBackup, workDestination)
			}
			return err
		}
		hadReports = true
	} else if !errors.Is(err, os.ErrNotExist) {
		if hadWork {
			_ = os.Rename(workBackup, workDestination)
		}
		return err
	}
	if err := os.Rename(workSource, workDestination); err != nil {
		if hadWork {
			_ = os.Rename(workBackup, workDestination)
		}
		if hadReports {
			_ = os.Rename(reportBackup, reportDestination)
		}
		return err
	}
	if err := os.Rename(reportSource, reportDestination); err != nil {
		_ = os.RemoveAll(workDestination)
		if hadWork {
			_ = os.Rename(workBackup, workDestination)
		}
		if hadReports {
			_ = os.Rename(reportBackup, reportDestination)
		}
		return err
	}
	if hadWork {
		if err := os.RemoveAll(workBackup); err != nil {
			log.Printf("ingesta: no se pudo quitar copia temporal de trabajo: %v", err)
		}
	}
	if hadReports {
		if err := os.RemoveAll(reportBackup); err != nil {
			log.Printf("ingesta: no se pudo quitar copia temporal de reportes: %v", err)
		}
	}
	return nil
}

func packageFolderName(job *stagedJob) string {
	monthNames := [...]string{"", "ENERO", "FEBRERO", "MARZO", "ABRIL", "MAYO", "JUNIO", "JULIO", "AGOSTO", "SEPTIEMBRE", "OCTUBRE", "NOVIEMBRE", "DICIEMBRE"}
	month, err := strconv.Atoi(job.Month)
	if err != nil || month < 1 || month >= len(monthNames) {
		return "LOTE_PERIODO_INVALIDO"
	}
	name := job.Service + "_" + monthNames[month] + "_" + job.Year
	if job.IsObjections {
		name += "_OBJECIONES"
	}
	return name
}

func matrixFilename(job *stagedJob) string {
	return "3. MATRIZ_" + job.Service + "_" + job.Month + "_" + job.Year + ".xlsm"
}

func pdfFilenameWithExtension(name string) string {
	name = strings.TrimSpace(name)
	if strings.EqualFold(filepath.Ext(name), ".pdf") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		name = "documento"
	}
	return name + ".pdf"
}

func (s *server) loadStagedJob(id string) (stagedJob, error) {
	var job stagedJob
	b, err := os.ReadFile(filepath.Join(s.jobRoot(id), "job.json"))
	if errors.Is(err, os.ErrNotExist) {
		b, err = os.ReadFile(filepath.Join(s.stagingDir, id, "job.json"))
	}
	if errors.Is(err, os.ErrNotExist) && legacyJobIDPattern.MatchString(id) {
		entries, readErr := os.ReadDir(s.workspacesDir)
		if readErr == nil {
			for _, entry := range entries {
				if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				candidate, candidateErr := os.ReadFile(filepath.Join(s.workspacesDir, entry.Name(), "job.json"))
				if candidateErr != nil || json.Unmarshal(candidate, &job) != nil {
					continue
				}
				for _, alias := range job.Aliases {
					if alias == id {
						return job, nil
					}
				}
			}
		}
	}
	if err != nil {
		return job, err
	}
	err = json.Unmarshal(b, &job)
	return job, err
}

func (s *server) saveStagedJob(job stagedJob) error {
	p := filepath.Join(s.jobRoot(job.ID), "job.json")
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		p = filepath.Join(s.stagingDir, job.ID, "job.json")
	}
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return atomicWritePrivateFile(p, b)
}

func copyPrivateFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func normalizePatientFolder(patient string) string {
	patient = strings.ToUpper(strings.TrimSpace(patient))
	patient = strings.TrimLeftFunc(patient, func(r rune) bool {
		return !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) && r != 'Á' && r != 'É' && r != 'Í' && r != 'Ó' && r != 'Ú' && r != 'Ñ'
	})
	replacer := strings.NewReplacer("Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N", "À", "A", "È", "E", "Ì", "I", "Ò", "O", "Ù", "U")
	patient = replacer.Replace(patient)
	var b strings.Builder
	underscore := false
	for _, r := range patient {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			underscore = false
		} else if !underscore && b.Len() > 0 {
			b.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

type uploadPart struct {
	field      string
	storedName string
	extension  string
	file       *multipart.FileHeader
}

func (s *server) receiveDualUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	entry, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	if s.maxIngest < 1 {
		writeError(w, http.StatusServiceUnavailable, "La carga de lotes no está configurada en el servidor.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxIngest)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			writeError(w, http.StatusRequestEntityTooLarge, "El lote supera el límite de carga configurado en el servidor.")
		} else {
			writeError(w, http.StatusBadRequest, "No se pudo leer la carga multipart.")
		}
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	month := strings.TrimSpace(r.FormValue("mes"))
	year := strings.TrimSpace(r.FormValue("anio"))
	service := strings.ToUpper(strings.TrimSpace(r.FormValue("tipo_servicio")))
	monthNumber, monthErr := strconv.Atoi(month)
	yearNumber, yearErr := strconv.Atoi(year)
	if monthErr != nil || monthNumber < 1 || monthNumber > 12 || len(month) != 2 || yearErr != nil || yearNumber < 2000 || yearNumber > 2100 || len(year) != 4 || !validServiceCode(service) {
		writeError(w, http.StatusBadRequest, "Indica un mes, año y tipo de servicio válidos.")
		return
	}
	parts := []uploadPart{
		{field: "zip_file", storedName: "lote.zip", extension: ".zip"},
	}
	for i := range parts {
		files := r.MultipartForm.File[parts[i].field]
		if len(files) != 1 || files[0].Size == 0 || !strings.EqualFold(filepath.Ext(files[0].Filename), parts[i].extension) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Adjunta un archivo válido en el campo %s (%s, no vacío).", parts[i].field, parts[i].extension))
			return
		}
		parts[i].file = files[0]
	}
	for _, optional := range headerParts {
		files := r.MultipartForm.File[optional.field]
		if len(files) == 0 {
			continue
		}
		if len(files) != 1 || files[0].Size == 0 || !strings.EqualFold(filepath.Ext(files[0].Filename), optional.extension) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Adjunta un archivo válido en el campo %s (%s, no vacío).", optional.field, optional.extension))
			return
		}
		optional.file = files[0]
		parts = append(parts, optional)
	}

	incomingDigest, err := sha256Upload(parts[0].file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el ZIP de planillas.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	jobID := periodWorkspaceID(service, month, year)
	if err := os.MkdirAll(s.workspacesDir, 0700); err != nil {
		log.Printf("ingesta %s: no se pudo crear el directorio de expedientes: %v", jobID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el área privada del lote.")
		return
	}
	var existing *stagedJob
	stableMetadataPath := filepath.Join(s.jobRoot(jobID), "job.json")
	if _, statErr := os.Stat(stableMetadataPath); statErr == nil {
		loaded, loadErr := s.loadStagedJob(jobID)
		if loadErr != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo leer el espacio existente para este período.")
			return
		}
		existing = &loaded
	} else if !errors.Is(statErr, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "No se pudo revisar el espacio existente para este período.")
		return
	} else {
		candidate, legacyRoot, findErr := s.findLegacyPeriodWorkspace(service, month, year, incomingDigest)
		if findErr != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo revisar expedientes anteriores de este período.")
			return
		}
		if legacyRoot != "" {
			migrated, migrateErr := s.migrateToPeriodWorkspace(candidate, legacyRoot, jobID)
			if migrateErr != nil {
				log.Printf("ingesta %s: no se pudo reutilizar espacio anterior: %v", jobID, migrateErr)
				writeError(w, http.StatusInternalServerError, "No se pudo reutilizar el expediente existente de este período.")
				return
			}
			existing = &migrated
		}
	}
	if existing != nil {
		storedZip := ""
		storedDigest := ""
		for _, file := range existing.Files {
			if file.Field == "zip_file" {
				storedZip, storedDigest = file.StoredName, file.SHA256
				break
			}
		}
		if storedZip != "" && storedDigest == "" {
			storedDigest, _ = sha256Path(filepath.Join(s.jobSourcesDir(existing.ID), storedZip))
		}
		if storedDigest == incomingDigest {
			output := ""
			if existing.Status == "PROCESSED" || existing.Status == "INCOMPLETE" {
				output = filepath.Join(s.jobRoot(existing.ID), "trabajo", packageFolderName(existing))
			}
			writeJSON(w, http.StatusOK, jobResponse(s, *existing, output, nil))
			return
		}
		response := jobResponse(s, *existing, filepath.Join(s.jobRoot(existing.ID), "trabajo", packageFolderName(existing)), nil)
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Ya existe un espacio de trabajo para este período y servicio. Se abrió el mismo expediente; añade los PDFs nuevos desde la sección de documentos.", "existing_job": response})
		return
	}
	if err := os.Chmod(s.workspacesDir, 0700); err != nil {
		log.Printf("ingesta %s: no se pudieron asegurar permisos privados: %v", jobID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo asegurar el acceso privado al espacio del lote.")
		return
	}
	tempDir, err := os.MkdirTemp(s.workspacesDir, ".upload-")
	if err != nil {
		log.Printf("ingesta %s: no se pudo crear espacio temporal: %v", jobID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el espacio privado del lote.")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(tempDir)
		}
	}()
	sourceDir := filepath.Join(tempDir, "fuentes")
	if err := os.Mkdir(sourceDir, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el espacio privado del lote.")
		return
	}

	job := stagedJob{
		ID: jobID, Status: "STAGED", Username: entry.username,
		Month: month, Year: year, Service: service, ReceivedAt: time.Now().UTC(),
		Files:        make([]stagedUpload, 0, len(parts)),
		StatusDetail: "Lote guardado en el espacio permanente. Los documentos habilitantes pueden añadirse después.",
	}
	for _, part := range parts {
		storedPath := filepath.Join(sourceDir, part.storedName)
		size, err := saveUploadPart(part.file, storedPath)
		if err != nil {
			log.Printf("ingesta %s: no se pudo guardar el campo %s: %v", jobID, part.field, err)
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("No se pudo guardar el archivo %s.", part.field))
			return
		}
		if err := validateStagedFile(storedPath, part.extension); err != nil {
			log.Printf("ingesta %s: validación fallida para %s: %v", jobID, part.field, err)
			writeError(w, http.StatusBadRequest, fmt.Sprintf("El archivo %s no tiene un contenido %s válido.", part.field, part.extension))
			return
		}
		digest := incomingDigest
		if part.field != "zip_file" {
			digest, _ = sha256Path(storedPath)
		}
		job.Files = append(job.Files, stagedUpload{
			Field: part.field, StoredName: part.storedName,
			OriginalName: safeOriginalFilename(part.file.Filename), Size: size, SHA256: digest,
		})
	}
	metadata, err := os.OpenFile(filepath.Join(tempDir, "job.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Printf("ingesta %s: no se pudo crear job.json: %v", jobID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el lote recibido.")
		return
	}
	encodeErr := json.NewEncoder(metadata).Encode(job)
	closeErr := metadata.Close()
	if encodeErr != nil || closeErr != nil {
		log.Printf("ingesta %s: no se pudo escribir job.json (encode=%v close=%v)", jobID, encodeErr, closeErr)
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el lote recibido.")
		return
	}
	if err := os.Rename(tempDir, s.jobRoot(jobID)); err != nil {
		log.Printf("ingesta %s: no se pudo publicar espacio de expediente: %v", jobID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo confirmar el espacio del lote.")
		return
	}
	committed = true
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": job.Status, "job_id": job.ID, "message": job.StatusDetail,
		"timestamp": job.ReceivedAt, "workspace": s.jobRoot(job.ID), "missing_documents": missingHeaders(job), "files": job.Files,
	})
}

func saveUploadPart(file *multipart.FileHeader, destination string) (int64, error) {
	source, err := file.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return 0, err
	}
	size, copyErr := io.Copy(output, source)
	closeErr := output.Close()
	if copyErr != nil {
		return size, copyErr
	}
	if closeErr != nil {
		return size, closeErr
	}
	return size, nil
}

func validateStagedFile(filename, extension string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var header [8]byte
	n, err := io.ReadFull(file, header[:])
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	if n < 5 {
		return fmt.Errorf("archivo demasiado pequeño")
	}
	if extension == ".pdf" {
		if string(header[:5]) != "%PDF-" {
			return fmt.Errorf("encabezado PDF inválido")
		}
		return nil
	}
	if n < 4 || header[0] != 'P' || header[1] != 'K' || header[2] != 3 && header[2] != 5 && header[2] != 7 || header[3] != 4 && header[3] != 6 && header[3] != 8 {
		return fmt.Errorf("encabezado ZIP inválido")
	}
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) == 0 {
		return fmt.Errorf("archivo ZIP vacío")
	}
	if extension == ".xlsm" {
		contentTypes, workbook := false, false
		for _, item := range archive.File {
			switch item.Name {
			case "[Content_Types].xml":
				contentTypes = true
			case "xl/workbook.xml":
				workbook = true
			}
		}
		if !contentTypes || !workbook {
			return fmt.Errorf("paquete Excel incompleto")
		}
	}
	return nil
}

func safeOriginalFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	name := path.Base(filename)
	if name == "." || name == ".." || name == "/" || strings.TrimSpace(name) == "" {
		return "archivo"
	}
	return name
}

func validServiceCode(value string) bool {
	switch value {
	case "HOSPITALIZACION", "EMERGENCIA", "AMBULATORIO":
		return true
	default:
		return false
	}
}

func (s *server) findLegacyPeriodWorkspace(service, month, year, digest string) (stagedJob, string, error) {
	entries, err := os.ReadDir(s.workspacesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return stagedJob{}, "", nil
		}
		return stagedJob{}, "", err
	}
	var best stagedJob
	bestRoot := ""
	bestMatchesUpload := false
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || stableWorkspaceIDPattern.MatchString(entry.Name()) {
			continue
		}
		root := filepath.Join(s.workspacesDir, entry.Name())
		data, readErr := os.ReadFile(filepath.Join(root, "job.json"))
		if readErr != nil {
			continue
		}
		var candidate stagedJob
		if json.Unmarshal(data, &candidate) != nil || candidate.Service != service || candidate.Month != month || candidate.Year != year || !s.hasClinicalSource(candidate) {
			continue
		}
		candidateDigest := ""
		for _, file := range candidate.Files {
			if file.Field == "zip_file" {
				candidateDigest = file.SHA256
				if candidateDigest == "" {
					candidateDigest, _ = sha256Path(filepath.Join(root, "fuentes", file.StoredName))
				}
				break
			}
		}
		matchesUpload := candidateDigest != "" && candidateDigest == digest
		if bestRoot == "" || matchesUpload && !bestMatchesUpload || matchesUpload == bestMatchesUpload && candidate.ReceivedAt.After(best.ReceivedAt) {
			best, bestRoot, bestMatchesUpload = candidate, root, matchesUpload
		}
	}
	return best, bestRoot, nil
}

func (s *server) migrateToPeriodWorkspace(job stagedJob, oldRoot, stableID string) (stagedJob, error) {
	targetRoot := s.jobRoot(stableID)
	if _, err := os.Stat(filepath.Join(targetRoot, "job.json")); err == nil {
		return s.loadStagedJob(stableID)
	}
	oldID := job.ID
	if legacyJobIDPattern.MatchString(oldID) {
		found := false
		for _, alias := range job.Aliases {
			if alias == oldID {
				found = true
				break
			}
		}
		if !found {
			job.Aliases = append(job.Aliases, oldID)
		}
	}
	job.ID = stableID
	metadata, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return stagedJob{}, err
	}
	oldMetadataPath := filepath.Join(oldRoot, "job.json")
	original, err := os.ReadFile(oldMetadataPath)
	if err != nil {
		return stagedJob{}, err
	}
	if err := os.WriteFile(oldMetadataPath, metadata, 0600); err != nil {
		return stagedJob{}, err
	}
	if err := os.Rename(oldRoot, targetRoot); err != nil {
		_ = os.WriteFile(oldMetadataPath, original, 0600)
		return stagedJob{}, err
	}
	return job, nil
}

func sha256Upload(file *multipart.FileHeader) (string, error) {
	input, err := file.Open()
	if err != nil {
		return "", err
	}
	defer input.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sha256Path(path string) (string, error) {
	input, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer input.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func newUploadJobID() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("JOB-%s-%s", time.Now().UTC().Format("20060102T150405"), hex.EncodeToString(random[:])), nil
}
