package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const objectionUploadLimit int64 = 64 << 20

type objectionRecord struct {
	Tramite             string `json:"pdi_tramite"`
	Cedula              string `json:"pdi_cedula,omitempty"`
	Patient             string `json:"paciente,omitempty"`
	PatientFolder       string `json:"carpeta_paciente,omitempty"`
	SourcePatientFolder string `json:"-"`
	ObjectedValue       string `json:"valor_objetado,omitempty"`
	Code                string `json:"codigo_objecion,omitempty"`
	Motive              string `json:"motivo_objecion"`
	PlanillaID          int64  `json:"pdi_id,omitempty"`
	Matched             bool   `json:"coincide_expediente"`
	Posture             string `json:"postura,omitempty"`
	CoverageAttached    bool   `json:"cobertura_adjunta"`
}

func editableObjectionWorkspace(job stagedJob) bool {
	return job.IsObjections && (job.Status == "PROCESSED" || job.Status == "INCOMPLETE")
}

type objectionPackagePDF struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size_bytes"`
	Included bool   `json:"incluido_en_zip"`
	Required bool   `json:"obligatorio"`
}

func normalizedWorkspaceRelativePath(path string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
}

func requiredObjectionPDF(path string) bool {
	name := strings.ToUpper(filepath.Base(filepath.FromSlash(path)))
	return name == "P_INDIVIDUAL.PDF" || name == "C_COBERTURA.PDF"
}

func objectionPDFIncluded(job stagedJob, path string) bool {
	if !job.IsObjections {
		return true
	}
	path = normalizedWorkspaceRelativePath(path)
	if requiredObjectionPDF(path) {
		return true
	}
	return job.ObjectionPDFSelection[path]
}

func objectionZIPIncludesFile(job stagedJob, relative string) bool {
	relative = normalizedWorkspaceRelativePath(relative)
	if !job.IsObjections {
		return true
	}
	if relative == "3. MATRIZ_RESPUESTA.xlsx" {
		return false
	}
	if strings.HasPrefix(relative, "4. EXPEDIENTES/") && strings.EqualFold(filepath.Ext(relative), ".pdf") {
		return objectionPDFIncluded(job, relative)
	}
	return true
}

func objectionPDFRecord(job stagedJob, path string) (*objectionRecord, bool) {
	path = normalizedWorkspaceRelativePath(path)
	if !strings.HasPrefix(path, "4. EXPEDIENTES/") || !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return nil, false
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, false
	}
	planillaID := planillaForPath(job, path)
	for index := range job.ObjectionRows {
		row := &job.ObjectionRows[index]
		if row.PatientFolder == parts[1] && row.Matched && row.PlanillaID > 0 && row.PlanillaID == planillaID {
			return row, true
		}
	}
	return nil, false
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func (s *server) objectionPDFSelection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/pdfs/seleccion/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	if r.Method == http.MethodPost {
		s.ingestMu.Lock()
		defer s.ingestMu.Unlock()
	}
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio preparado de objeciones.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	if r.Method == http.MethodPost {
		var input struct {
			Path     string `json:"path"`
			Included *bool  `json:"incluir"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Included == nil {
			writeError(w, http.StatusBadRequest, "Indica el PDF y si debe incluirse en el ZIP.")
			return
		}
		path := normalizedWorkspaceRelativePath(input.Path)
		_, allowed := objectionPDFRecord(job, path)
		if !allowed {
			writeError(w, http.StatusBadRequest, "Selecciona un PDF asociado a un trámite objetado.")
			return
		}
		if !*input.Included && requiredObjectionPDF(path) {
			writeError(w, http.StatusConflict, "P_INDIVIDUAL.pdf y C_COBERTURA.pdf son obligatorios para cada trámite.")
			return
		}
		filePath, err := safeWorkspacePath(packageRoot, path)
		if err != nil {
			writeError(w, http.StatusBadRequest, "La ruta del PDF no es válida.")
			return
		}
		info, err := os.Stat(filePath)
		if err != nil || !info.Mode().IsRegular() {
			writeError(w, http.StatusNotFound, "No se encontró el PDF seleccionado.")
			return
		}
		previousSelection := cloneBoolMap(job.ObjectionPDFSelection)
		if *input.Included {
			if job.ObjectionPDFSelection == nil {
				job.ObjectionPDFSelection = make(map[string]bool)
			}
			job.ObjectionPDFSelection[path] = true
		} else {
			delete(job.ObjectionPDFSelection, path)
		}
		if err := s.saveStagedJob(job); err != nil {
			job.ObjectionPDFSelection = previousSelection
			writeError(w, http.StatusInternalServerError, "No se pudo guardar la selección de PDFs.")
			return
		}
	}
	documents := make([]workspacePDF, 0)
	if _, statErr := os.Stat(packageRoot); statErr == nil {
		documents, _, err = listWorkspacePDFs(packageRoot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudieron listar los PDFs de objeciones.")
			return
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "No se pudieron listar los PDFs de objeciones.")
		return
	}
	result := make([]objectionPackagePDF, 0, len(documents))
	for _, document := range documents {
		if _, allowed := objectionPDFRecord(job, document.Path); !allowed {
			continue
		}
		result = append(result, objectionPackagePDF{
			Path: document.Path, Name: document.Name, Size: document.Size,
			Included: objectionPDFIncluded(job, document.Path), Required: requiredObjectionPDF(document.Path),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": result})
}

func (s *server) setObjectionPosture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/postura/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	var input struct {
		Tramite string `json:"pdi_tramite"`
		Posture string `json:"postura"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Indica el trámite y su postura.")
		return
	}
	tramite, err := normalizeTramiteValue(input.Tramite)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Selecciona un trámite objetado.")
		return
	}
	posture := strings.ToUpper(strings.TrimSpace(input.Posture))
	if posture != "ACEPTA" && posture != "RECHAZA" {
		writeError(w, http.StatusBadRequest, "Selecciona ACEPTA o RECHAZA como postura.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de objeciones.")
		return
	}
	var record *objectionRecord
	for index := range job.ObjectionRows {
		if job.ObjectionRows[index].Tramite == tramite {
			record = &job.ObjectionRows[index]
			break
		}
	}
	if record == nil {
		writeError(w, http.StatusNotFound, "El trámite no pertenece a este espacio de objeciones.")
		return
	}
	previous := record.Posture
	record.Posture = posture
	if err := s.saveStagedJob(job); err != nil {
		record.Posture = previous
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la postura.")
		return
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, "", nil))
}

func (s *server) previewObjections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el período seleccionado.")
		return
	}
	sourceID := strings.TrimSpace(r.FormValue("expediente_origen"))
	if sourceID == "" || !validJobID(sourceID) {
		writeError(w, http.StatusBadRequest, "Selecciona un expediente de primer ingreso válido.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	candidates, source, err := s.objectionCandidates(ctx, sourceID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"expediente_origen": source.ID, "mes": source.Month, "anio": source.Year,
		"tipo_servicio": source.Service, "candidatos": candidates,
		"message": "Selecciona manualmente los pacientes objetados. La aplicación no lee el informe de liquidación.",
	})
}

type objectionRequest struct {
	sourceID         string
	reuseID          string
	selectedTramites []string
}

func (s *server) readObjectionRequest(w http.ResponseWriter, r *http.Request) (objectionRequest, error) {
	var request objectionRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
		return request, errors.New("No se pudieron leer los documentos cargados. Comprueba que cada archivo no supere 64 MiB.")
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	request.sourceID = strings.TrimSpace(r.FormValue("expediente_origen"))
	request.reuseID = strings.TrimSpace(r.FormValue("reutilizar_espacio"))
	if request.reuseID != "" && !validJobID(request.reuseID) {
		return request, errors.New("El identificador del espacio de objeciones que quieres reutilizar no es válido.")
	}
	var selectionErr error
	request.selectedTramites, selectionErr = parseSelectedObjectionTramites(r.MultipartForm.Value["tramites_objetados"])
	if selectionErr != nil {
		return request, selectionErr
	}
	if request.sourceID == "" || !validJobID(request.sourceID) {
		return request, errors.New("Selecciona un expediente de primer ingreso válido.")
	}
	return request, nil
}

func parseSelectedObjectionTramites(values []string) ([]string, error) {
	if len(values) > 5000 {
		return nil, errors.New("No se pueden seleccionar más de 5.000 planillas.")
	}
	seen := make(map[string]bool, len(values))
	selected := make([]string, 0, len(values))
	for _, value := range values {
		tramite, err := normalizeTramiteValue(value)
		if err != nil {
			return nil, errors.New("La selección contiene un trámite no válido.")
		}
		if !seen[tramite] {
			seen[tramite] = true
			selected = append(selected, tramite)
		}
	}
	return selected, nil
}

func multipartHeader(r *http.Request, field string) *multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	files := r.MultipartForm.File[field]
	if len(files) != 1 {
		return nil
	}
	return files[0]
}

func readMultipartBytes(header *multipart.FileHeader, limit int64) ([]byte, error) {
	input, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer input.Close()
	data, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("archivo demasiado grande")
	}
	return data, nil
}

func pdfSignature(data []byte) bool { return len(data) >= 5 && string(data[:5]) == "%PDF-" }

func normalizeTramiteValue(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, " ", ""))
	if strings.ContainsAny(value, "eE") {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || number < 0 || number != float64(int64(number)) {
			return "", errors.New("trámite científico inválido")
		}
		value = strconv.FormatInt(int64(number), 10)
	}
	if strings.Contains(value, ",") {
		value = strings.ReplaceAll(value, ",", "")
	}
	if strings.Contains(value, ".") {
		if strings.Trim(value[strings.LastIndex(value, ".")+1:], "0") == "" {
			value = value[:strings.LastIndex(value, ".")]
		} else {
			return "", errors.New("trámite no entero")
		}
	}
	if !tramitePattern.MatchString(value) {
		return "", errors.New("trámite no numérico")
	}
	return value, nil
}

func normalizeCedulaValue(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, " ", ""))
	if strings.ContainsAny(value, "eE") {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || number < 0 || number != float64(int64(number)) {
			return "", errors.New("cédula científica inválida")
		}
		value = strconv.FormatInt(int64(number), 10)
	}
	if index := strings.LastIndex(value, "."); index >= 0 {
		if strings.Trim(value[index+1:], "0") != "" {
			return "", errors.New("cédula decimal")
		}
		value = value[:index]
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", errors.New("cédula no numérica")
		}
	}
	if len(value) < 6 || len(value) > 13 {
		return "", errors.New("longitud de cédula no válida")
	}
	if len(value) < 10 {
		value = strings.Repeat("0", 10-len(value)) + value
	}
	return value, nil
}

func (s *server) objectionCandidates(ctx context.Context, sourceID string) ([]objectionRecord, stagedJob, error) {
	source, err := s.loadStagedJob(sourceID)
	if err != nil || source.IsObjections || (source.Status != "PROCESSED" && source.Status != "INCOMPLETE") || !s.hasClinicalSource(source) {
		return nil, stagedJob{}, errors.New("El expediente de origen debe ser un primer ingreso preparado que conserve su ZIP.")
	}
	identities, err := s.loadPlanillaIdentities(ctx, source)
	if err != nil {
		return nil, source, err
	}
	packageRoot := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source))
	patientRoot := filepath.Join(packageRoot, "4. EXPEDIENTES")
	available := make(map[int64]map[string]bool)
	for relative, planillaID := range source.DocumentPlanillas {
		relative = filepath.ToSlash(relative)
		parts := strings.Split(relative, "/")
		if planillaID <= 0 || len(parts) < 3 || parts[0] != "4. EXPEDIENTES" || !strings.EqualFold(filepath.Ext(relative), ".pdf") {
			continue
		}
		filename, pathErr := safeWorkspacePath(packageRoot, relative)
		if pathErr != nil {
			continue
		}
		info, statErr := os.Stat(filename)
		if statErr == nil && info.Mode().IsRegular() && info.Size() > 0 {
			if available[planillaID] == nil {
				available[planillaID] = make(map[string]bool)
			}
			available[planillaID][parts[1]] = true
		}
	}
	result := make([]objectionRecord, 0, len(available))
	folderNames := make(map[string]string)
	for _, identity := range identities {
		folder := normalizePatientFolder(identity.Patient)
		if !available[identity.PlanillaID][folder] {
			continue
		}
		if previous, exists := folderNames[folder]; exists && !strings.EqualFold(strings.TrimSpace(previous), strings.TrimSpace(identity.Patient)) {
			return nil, source, fmt.Errorf("Dos nombres de Oracle colisionan en la carpeta %s. Corrige la identidad antes de seleccionar objeciones.", folder)
		}
		folderNames[folder] = identity.Patient
		info, statErr := os.Stat(filepath.Join(patientRoot, folder))
		if folder == "" || statErr != nil || !info.IsDir() {
			continue
		}
		candidate := objectionRecord{
			Tramite: identity.Tramite, Cedula: identity.Cedula, Patient: identity.Patient,
			SourcePatientFolder: folder, PlanillaID: identity.PlanillaID, Matched: true,
		}
		if cedula, normalizeErr := normalizeCedulaValue(candidate.Cedula); normalizeErr == nil {
			candidate.Cedula = cedula
		}
		candidate.PatientFolder = objectionPatientFolder(candidate.Cedula, candidate.Patient, candidate.Tramite)
		result = append(result, candidate)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Patient == result[j].Patient {
			return result[i].Tramite < result[j].Tramite
		}
		return result[i].Patient < result[j].Patient
	})
	return result, source, nil
}

func buildSelectedObjectionRows(selected []string, candidates []objectionRecord) ([]objectionRecord, error) {
	if len(selected) == 0 {
		return nil, errors.New("Selecciona al menos un paciente objetado.")
	}
	candidatesByTramite := make(map[string]objectionRecord, len(candidates))
	for _, candidate := range candidates {
		candidatesByTramite[candidate.Tramite] = candidate
	}
	result := make([]objectionRecord, 0, len(selected))
	seen := make(map[string]bool, len(selected))
	for _, tramite := range selected {
		if seen[tramite] {
			continue
		}
		seen[tramite] = true
		candidate, ok := candidatesByTramite[tramite]
		if !ok || !candidate.Matched {
			return nil, fmt.Errorf("El trámite %s no tiene un expediente disponible para seleccionar.", tramite)
		}
		result = append(result, candidate)
	}
	if len(result) == 0 {
		return nil, errors.New("No hay filas de objeción para los pacientes seleccionados.")
	}
	if len(result) > 10000 {
		return nil, errors.New("La selección supera el límite de 10.000 filas de respuesta.")
	}
	return result, nil
}

func (s *server) objectionWorkspacesForPeriod(service, month, year string) ([]stagedJob, error) {
	directories, err := os.ReadDir(s.workspacesDir)
	if errors.Is(err, os.ErrNotExist) {
		return []stagedJob{}, nil
	}
	if err != nil {
		return nil, err
	}
	workspaces := make([]stagedJob, 0)
	for _, directory := range directories {
		if !directory.IsDir() || strings.HasPrefix(directory.Name(), ".") {
			continue
		}
		job, loadErr := s.loadStagedJob(directory.Name())
		if loadErr != nil || !job.IsObjections || !sameObjectionPeriod(job.Service, job.Month, job.Year, service, month, year) {
			continue
		}
		workspaces = append(workspaces, job)
	}
	sort.Slice(workspaces, func(i, j int) bool {
		if workspaces[i].ReceivedAt.Equal(workspaces[j].ReceivedAt) {
			return workspaces[i].ID < workspaces[j].ID
		}
		return workspaces[i].ReceivedAt.Before(workspaces[j].ReceivedAt)
	})
	return workspaces, nil
}

func sameObjectionPeriod(serviceA, monthA, yearA, serviceB, monthB, yearB string) bool {
	parsedMonthA, errA := strconv.Atoi(strings.TrimSpace(monthA))
	parsedMonthB, errB := strconv.Atoi(strings.TrimSpace(monthB))
	monthsMatch := strings.EqualFold(strings.TrimSpace(monthA), strings.TrimSpace(monthB))
	if errA == nil && errB == nil {
		monthsMatch = parsedMonthA == parsedMonthB
	}
	return strings.EqualFold(strings.TrimSpace(serviceA), strings.TrimSpace(serviceB)) &&
		monthsMatch && strings.EqualFold(strings.TrimSpace(yearA), strings.TrimSpace(yearB))
}

func objectionWorkspaceSummaries(workspaces []stagedJob) []map[string]any {
	result := make([]map[string]any, 0, len(workspaces))
	for _, job := range workspaces {
		result = append(result, map[string]any{
			"job_id": job.ID, "status": job.Status, "received_at": job.ReceivedAt,
			"creado_por": job.Username, "tipo_servicio": job.Service, "mes": job.Month, "anio": job.Year,
		})
	}
	return result
}

func (s *server) appendObjectionRows(job *stagedJob, source stagedJob, rows []objectionRecord) (int, bool, error) {
	existing := make(map[string]bool, len(job.ObjectionRows))
	for _, row := range job.ObjectionRows {
		existing[row.Tramite] = true
	}
	newRows := make([]objectionRecord, 0, len(rows))
	for _, row := range rows {
		if existing[row.Tramite] {
			continue
		}
		existing[row.Tramite] = true
		newRows = append(newRows, row)
	}
	if len(newRows) == 0 {
		return 0, false, nil
	}

	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(job))
	patientRoot := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source), "4. EXPEDIENTES")
	previousRows := append([]objectionRecord(nil), job.ObjectionRows...)
	previousFiles := append([]stagedUpload(nil), job.Files...)
	previousStatusDetail := job.StatusDetail
	previousMappings := make(map[string]int64, len(job.DocumentPlanillas))
	for path, planillaID := range job.DocumentPlanillas {
		previousMappings[path] = planillaID
	}
	matrixName := objectionMatrixFilename(*job)
	matrixPath := filepath.Join(packageRoot, matrixName)
	previousMatrix, matrixErr := os.ReadFile(matrixPath)
	matrixExisted := matrixErr == nil
	if matrixErr != nil && !errors.Is(matrixErr, os.ErrNotExist) {
		return 0, false, errors.New("No se pudo revisar la matriz anterior.")
	}
	if err := installObjectionRows(packageRoot, patientRoot, source, newRows, job); err != nil {
		job.DocumentPlanillas = previousMappings
		return 0, false, err
	}
	job.ObjectionRows = append(job.ObjectionRows, newRows...)
	if matrixExisted {
		if err := saveObjectionVersion(s.jobRoot(job.ID), matrixName, previousMatrix); err != nil {
			cleanupObjectionRows(packageRoot, newRows)
			job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
			return 0, false, errors.New("No se pudo conservar la matriz anterior antes de agregar los trámites.")
		}
		if err := os.Remove(matrixPath); err != nil {
			cleanupObjectionRows(packageRoot, newRows)
			job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
			return 0, false, errors.New("No se pudo invalidar la matriz anterior.")
		}
	}
	job.Files = removeObjectionFile(job.Files, "objection_matriz")
	job.StatusDetail = fmt.Sprintf("Se agregaron %d trámites al espacio de objeciones. Los documentos y posturas existentes se conservaron.", len(newRows))
	if matrixExisted {
		job.StatusDetail += " La matriz anterior quedó archivada e invalidada; vuelve a cargarla antes de preparar la descarga final."
	}
	job.StatusDetail += " Los documentos de cabecera se solicitan al preparar el ZIP. Oracle permanece sin cambios."
	if err := s.saveStagedJob(*job); err != nil {
		cleanupObjectionRows(packageRoot, newRows)
		job.ObjectionRows, job.Files, job.DocumentPlanillas, job.StatusDetail = previousRows, previousFiles, previousMappings, previousStatusDetail
		if matrixExisted {
			_ = atomicWritePrivateFile(matrixPath, previousMatrix)
		}
		return 0, false, errors.New("No se pudo registrar la selección; se restauraron la matriz y el contenido anterior.")
	}
	return len(newRows), matrixExisted, nil
}

func (s *server) createObjectionWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	session, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	input, err := s.readObjectionRequest(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.MultipartForm.RemoveAll()
	if len(input.selectedTramites) == 0 {
		writeError(w, http.StatusBadRequest, "Selecciona al menos un paciente objetado.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	source, err := s.loadStagedJob(input.sourceID)
	if err != nil || source.IsObjections || (source.Status != "PROCESSED" && source.Status != "INCOMPLETE") || !s.hasClinicalSource(source) {
		writeError(w, http.StatusUnprocessableEntity, "El expediente de origen debe ser un primer ingreso preparado que conserve su ZIP.")
		return
	}
	existing, err := s.objectionWorkspacesForPeriod(source.Service, source.Month, source.Year)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron revisar los espacios de objeciones existentes.")
		return
	}
	var reuse *stagedJob
	if len(existing) > 1 {
		for index := range existing {
			if existing[index].ID == input.reuseID {
				reuse = &existing[index]
				break
			}
		}
		if reuse == nil {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":               "Ya existen varios espacios de objeciones para este servicio y período. Elige uno para continuar; no se crearán, borrarán ni fusionarán espacios automáticamente.",
				"espacios_existentes": objectionWorkspaceSummaries(existing),
			})
			return
		}
	} else if len(existing) == 1 {
		reuse = &existing[0]
		if input.reuseID != "" && input.reuseID != reuse.ID {
			writeError(w, http.StatusConflict, "El espacio elegido no corresponde al período seleccionado.")
			return
		}
	} else if input.reuseID != "" {
		writeError(w, http.StatusConflict, "El espacio de objeciones elegido ya no existe para este período. Actualiza la lista e inténtalo de nuevo.")
		return
	}
	candidates, source, err := s.objectionCandidates(ctx, input.sourceID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	rows, err := buildSelectedObjectionRows(input.selectedTramites, candidates)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if reuse != nil {
		if !editableObjectionWorkspace(*reuse) {
			writeError(w, http.StatusConflict, "El espacio de objeciones existente no está disponible para continuar.")
			return
		}
		added, invalidatedMatrix, appendErr := s.appendObjectionRows(reuse, source, rows)
		if appendErr != nil {
			writeError(w, http.StatusUnprocessableEntity, appendErr.Error())
			return
		}
		message := fmt.Sprintf("Se reutilizó el espacio %s para %s · %s/%s.", reuse.ID, reuse.Service, reuse.Month, reuse.Year)
		if added == 0 {
			message += " Los trámites seleccionados ya estaban incluidos; no se duplicaron y no se creó otro ID."
		} else {
			message += fmt.Sprintf(" Se agregaron %d trámites nuevos; los documentos y posturas existentes se conservaron.", added)
			if invalidatedMatrix {
				message += " La matriz anterior quedó archivada e invalidada; vuelve a cargarla antes de preparar la descarga final."
			}
		}
		message += " Los documentos de cabecera se solicitan al preparar el ZIP. Oracle permanece sin cambios."
		result := jobResponse(s, *reuse, filepath.Join(s.jobRoot(reuse.ID), "trabajo", packageFolderName(reuse)), nil)
		result["message"] = message
		result["reutilizado"] = true
		result["tramites_agregados"] = added
		writeJSON(w, http.StatusOK, result)
		return
	}
	id, err := newUploadJobID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo crear el identificador del espacio de objeciones.")
		return
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Username: session.username,
		Month: source.Month, Year: source.Year, Service: source.Service, ReceivedAt: time.Now().UTC(),
		IsObjections: true, ObjectionSourceID: source.ID, ObjectionRows: rows,
		StatusDetail: fmt.Sprintf("Espacio de objeciones separado con %d trámites seleccionados. Los documentos de cabecera se solicitan al preparar la descarga final del ZIP. Oracle permanece sin cambios.", len(rows)),
	}
	if err := s.installObjectionWorkspace(&job, source, rows); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	output := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	writeJSON(w, http.StatusCreated, jobResponse(s, job, output, nil))
}

func objectionMatrixFilename(job stagedJob) string {
	monthNames := [...]string{"", "ENERO", "FEBRERO", "MARZO", "ABRIL", "MAYO", "JUNIO", "JULIO", "AGOSTO", "SEPTIEMBRE", "OCTUBRE", "NOVIEMBRE", "DICIEMBRE"}
	month, err := strconv.Atoi(job.Month)
	if err != nil || month < 1 || month >= len(monthNames) {
		return "3. MATRIZ_OBJECIONES_" + normalizeAnnexFilename(job.Service) + "_PERIODO_INVALIDO_" + job.Year + ".xlsm"
	}
	return "3. MATRIZ_OBJECIONES_" + normalizeAnnexFilename(job.Service) + "_" + monthNames[month] + "_" + job.Year + ".xlsm"
}

func objectionHeaderName(job stagedJob, kind string) (string, string, error) {
	switch kind {
	case "liquidacion":
		return "I_LIQUIDACION.pdf", "objection_liquidacion", nil
	case "oficio":
		return "1. OFICIO DE PAGO.pdf", "objection_oficio", nil
	case "consolidada":
		return "2. PLANILLA CONSOLIDADA.pdf", "objection_consolidada", nil
	case "matriz":
		return objectionMatrixFilename(job), "objection_matriz", nil
	default:
		return "", "", errors.New("Selecciona un documento de cabecera válido.")
	}
}

func (s *server) uploadObjectionHeader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/cabeceras/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de objeciones.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, objectionUploadLimit+64*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el documento seleccionado.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	kind := strings.TrimSpace(r.FormValue("tipo"))
	name, field, err := objectionHeaderName(job, kind)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	file := multipartHeader(r, "archivo")
	extension := ".pdf"
	if field == "objection_matriz" {
		extension = ".xlsm"
	}
	if file == nil || file.Size <= 0 || file.Size > objectionUploadLimit || !strings.EqualFold(filepath.Ext(file.Filename), extension) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Carga un archivo %s válido de hasta 64 MiB.", extension))
		return
	}
	data, err := readMultipartBytes(file, objectionUploadLimit)
	if err != nil || (extension == ".pdf" && !pdfSignature(data)) || (extension == ".xlsm" && !validXLSMBytes(data)) {
		writeError(w, http.StatusBadRequest, "El archivo no corresponde a un documento válido del tipo seleccionado.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	target := filepath.Join(packageRoot, name)
	previous, readErr := os.ReadFile(target)
	existed := readErr == nil
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "No se pudo leer la versión anterior del documento.")
		return
	}
	if existed {
		if err := saveObjectionVersion(s.jobRoot(job.ID), name, previous); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo conservar la versión anterior del documento.")
			return
		}
	}
	if err := atomicWritePrivateFile(target, data); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el documento de cabecera.")
		return
	}
	previousFiles := append([]stagedUpload(nil), job.Files...)
	upsertObjectionFile(&job, stagedUpload{Field: field, StoredName: name, OriginalName: safeOriginalFilename(file.Filename), Size: int64(len(data)), SHA256: sha256Hex(data)})
	if err := s.saveStagedJob(job); err != nil {
		if existed {
			_ = atomicWritePrivateFile(target, previous)
		} else {
			_ = os.Remove(target)
		}
		job.Files = previousFiles
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el documento; se restauró su versión anterior.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "tipo": kind, "nombre_archivo": name, "cabeceras_objeciones": objectionHeaderStatuses(s, job), "message": "Documento de cabecera guardado. Oracle no fue modificado."})
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func upsertObjectionFile(job *stagedJob, upload stagedUpload) {
	for index := range job.Files {
		if job.Files[index].Field == upload.Field {
			job.Files[index] = upload
			return
		}
	}
	job.Files = append(job.Files, upload)
}

func validXLSMBytes(data []byte) bool {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return false
	}
	contentTypes, workbook := false, false
	for _, file := range archive.File {
		switch file.Name {
		case "[Content_Types].xml":
			input, err := file.Open()
			if err != nil {
				return false
			}
			contents, readErr := io.ReadAll(io.LimitReader(input, 1<<20))
			_ = input.Close()
			contentTypes = readErr == nil && bytes.Contains(contents, []byte("macroEnabled.main+xml"))
		case "xl/workbook.xml":
			workbook = true
		}
	}
	return contentTypes && workbook
}

func validXLSMFile(filename string) bool {
	data, err := os.ReadFile(filename)
	return err == nil && validXLSMBytes(data)
}

func objectionHeaderStatuses(s *server, job stagedJob) []map[string]any {
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	headers := []struct{ kind, name string }{
		{"liquidacion", "I_LIQUIDACION.pdf"},
		{"oficio", "1. OFICIO DE PAGO.pdf"},
		{"consolidada", "2. PLANILLA CONSOLIDADA.pdf"},
		{"matriz", objectionMatrixFilename(job)},
	}
	result := make([]map[string]any, 0, len(headers))
	for _, header := range headers {
		filename := filepath.Join(packageRoot, header.name)
		present := validPDFFile(filename)
		if header.kind == "matriz" {
			present = validXLSMFile(filename)
		}
		result = append(result, map[string]any{"tipo": header.kind, "nombre": header.name, "cargado": present})
	}
	return result
}

func (s *server) addObjectionPatients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/agregar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024*1024)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer la selección de trámites.")
		return
	}
	selected, err := parseSelectedObjectionTramites(r.Form["tramites_objetados"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de objeciones.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	candidates, source, err := s.objectionCandidates(ctx, job.ObjectionSourceID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	rows, err := buildSelectedObjectionRows(selected, candidates)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	added, invalidatedMatrix, err := s.appendObjectionRows(&job, source, rows)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	result := jobResponse(s, job, filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job)), nil)
	if added == 0 {
		result["message"] = "Los trámites seleccionados ya estaban en este espacio; no se duplicaron ni se modificaron sus documentos, posturas o matriz."
	} else if invalidatedMatrix {
		result["message"] = job.StatusDetail
	}
	writeJSON(w, http.StatusOK, result)
}

func removeObjectionFile(files []stagedUpload, field string) []stagedUpload {
	result := make([]stagedUpload, 0, len(files))
	for _, file := range files {
		if file.Field != field {
			result = append(result, file)
		}
	}
	return result
}

func cleanupObjectionRows(packageRoot string, rows []objectionRecord) {
	for _, row := range rows {
		_ = os.RemoveAll(filepath.Join(packageRoot, "4. EXPEDIENTES", row.PatientFolder))
		_ = os.RemoveAll(filepath.Join(packageRoot, "5. ANEXOS", row.PatientFolder))
	}
}

func (s *server) installObjectionWorkspace(job *stagedJob, source stagedJob, rows []objectionRecord) error {
	if err := os.MkdirAll(s.workspacesDir, 0700); err != nil {
		return errors.New("No se pudo preparar el directorio privado de expedientes.")
	}
	root := s.jobRoot(job.ID)
	if _, err := os.Lstat(root); err == nil {
		return errors.New("Ya existe un espacio con ese identificador.")
	}
	tempRoot, err := os.MkdirTemp(s.workspacesDir, ".objeciones-")
	if err != nil {
		return errors.New("No se pudo crear el espacio aislado.")
	}
	defer os.RemoveAll(tempRoot)
	packageRoot := filepath.Join(tempRoot, "trabajo", packageFolderName(job))
	for _, relative := range []string{"fuentes", "trabajo", filepath.Join("trabajo", packageFolderName(job), "4. EXPEDIENTES")} {
		if err := os.MkdirAll(filepath.Join(tempRoot, relative), 0700); err != nil {
			return errors.New("No se pudo preparar la estructura del paquete de objeciones.")
		}
	}
	patientRoot := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source), "4. EXPEDIENTES")
	if err := installObjectionRows(packageRoot, patientRoot, source, rows, job); err != nil {
		return err
	}
	if err := writePrivateJSON(filepath.Join(tempRoot, "job.json"), job); err != nil {
		return errors.New("No se pudo guardar el registro del espacio de objeciones.")
	}
	if err := os.Rename(tempRoot, root); err != nil {
		return errors.New("No se pudo publicar el espacio aislado de objeciones.")
	}
	return nil
}

func objectionPatientFolder(cedula, patient, tramite string) string {
	identity := normalizePatientFolder(strings.TrimSpace(cedula) + "_" + patient)
	if identity == "" {
		identity = "PACIENTE"
	}
	if len(identity) > 110 {
		identity = strings.Trim(identity[:110], "_")
	}
	return identity + "_" + tramite
}

func installObjectionRows(packageRoot, patientRoot string, source stagedJob, rows []objectionRecord, job *stagedJob) error {
	if job.DocumentPlanillas == nil {
		job.DocumentPlanillas = make(map[string]int64)
	}
	installed := make([]string, 0, len(rows))
	rollback := func() {
		for _, folder := range installed {
			_ = os.RemoveAll(folder)
			_ = os.RemoveAll(filepath.Join(packageRoot, "5. ANEXOS", filepath.Base(folder)))
		}
	}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if seen[row.Tramite] {
			continue
		}
		seen[row.Tramite] = true
		sourceFolder := row.SourcePatientFolder
		if sourceFolder == "" {
			sourceFolder = normalizePatientFolder(row.Patient)
		}
		destinationFolder := row.PatientFolder
		if destinationFolder == "" || destinationFolder == sourceFolder {
			destinationFolder = objectionPatientFolder(row.Cedula, row.Patient, row.Tramite)
		}
		destination := filepath.Join(packageRoot, "4. EXPEDIENTES", destinationFolder)
		annex := filepath.Join(packageRoot, "5. ANEXOS", destinationFolder)
		if _, err := os.Lstat(destination); err == nil {
			rollback()
			return fmt.Errorf("Ya existe una carpeta para el trámite %s.", row.Tramite)
		}
		if _, err := os.Lstat(annex); err == nil {
			rollback()
			return fmt.Errorf("Ya existe una carpeta de anexos para el trámite %s.", row.Tramite)
		}
		if err := os.MkdirAll(destination, 0700); err != nil {
			rollback()
			return errors.New("No se pudo crear el expediente objetado.")
		}
		installed = append(installed, destination)
		copied, mappings, err := copyObjectionPDFTree(filepath.Join(patientRoot, sourceFolder), destination, sourceFolder, destinationFolder, source, map[int64]bool{row.PlanillaID: true})
		if err != nil || copied == 0 {
			rollback()
			return fmt.Errorf("No se pudieron copiar los PDFs asociados al trámite %s desde el expediente de origen.", row.Tramite)
		}
		for relative, planillaID := range mappings {
			job.DocumentPlanillas[relative] = planillaID
		}
	}
	return nil
}

func copyObjectionPDFTree(source, destination, sourcePatientFolder, destinationPatientFolder string, sourceJob stagedJob, allowedPlanillas map[int64]bool) (int, map[string]int64, error) {
	copied := 0
	mappings := make(map[string]int64)
	err := filepath.WalkDir(source, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == source {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink no permitido")
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			return nil
		}
		sourceRelative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", sourcePatientFolder, relative))
		planillaID := planillaForPath(sourceJob, sourceRelative)
		if planillaID <= 0 {
			return fmt.Errorf("el PDF %s no tiene un vínculo PDI_ID verificable", entry.Name())
		}
		if !allowedPlanillas[planillaID] {
			return nil
		}
		if strings.HasPrefix(strings.ToUpper(entry.Name()), "C_COBERTURA") && strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			target = filepath.Join(destination, "C_COBERTURA.pdf")
			if _, err := os.Lstat(target); err == nil {
				return nil
			}
		}
		info, err := entry.Info()
		if err != nil || info.Size() <= 0 || info.Size() > 256<<20 {
			return errors.New("PDF de origen inválido o demasiado grande")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := copyPrivateFile(current, target); err != nil {
			return err
		}
		destinationPath := relative
		if filepath.Base(target) == "C_COBERTURA.pdf" {
			destinationPath = "C_COBERTURA.pdf"
		}
		destinationRelative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", destinationPatientFolder, destinationPath))
		mappings[destinationRelative] = planillaID
		copied++
		return nil
	})
	return copied, mappings, err
}

func (s *server) uploadObjectionDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/documentos/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de objeciones.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, objectionUploadLimit+64*1024)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el PDF.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	kind := strings.TrimSpace(r.FormValue("tipo"))
	if kind != "anexo" {
		writeError(w, http.StatusBadRequest, "En Objeciones solo se cargan anexos nuevos. Revisa o reemplaza los demás PDFs en Abrir documentos del paciente.")
		return
	}
	tramite, err := normalizeTramiteValue(r.FormValue("pdi_tramite"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Selecciona un trámite objetado.")
		return
	}
	var record *objectionRecord
	for index := range job.ObjectionRows {
		if job.ObjectionRows[index].Tramite == tramite {
			record = &job.ObjectionRows[index]
			break
		}
	}
	if record == nil || !record.Matched {
		writeError(w, http.StatusNotFound, "El trámite no pertenece a este espacio de objeciones.")
		return
	}
	file := multipartHeader(r, "pdf_file")
	if file == nil || file.Size <= 0 || file.Size > objectionUploadLimit || !strings.EqualFold(filepath.Ext(file.Filename), ".pdf") {
		writeError(w, http.StatusBadRequest, "Carga un PDF de hasta 64 MiB.")
		return
	}
	data, err := readMultipartBytes(file, objectionUploadLimit)
	if err != nil || !pdfSignature(data) {
		writeError(w, http.StatusBadRequest, "El archivo cargado no contiene un PDF válido.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	directory := filepath.Join(packageRoot, "5. ANEXOS", record.PatientFolder)
	if err := os.MkdirAll(directory, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carpeta de anexos.")
		return
	}
	name, err := objectionAnnexBase(r.FormValue("tipo_anexo"), r.FormValue("nombre_anexo"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	name, err = nextSafeAnnexName(directory, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar un nombre seguro para el anexo.")
		return
	}
	if err := atomicWritePrivateFile(filepath.Join(directory, name), data); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el anexo.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "pdi_tramite": tramite, "tipo": kind, "nombre_archivo": name, "postura": record.Posture, "message": "Anexo guardado en el espacio aislado de objeciones. Oracle no fue modificado."})
}

func saveObjectionVersion(root, tramite string, contents []byte) error {
	directory := filepath.Join(root, "fuentes", "versiones")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	extension := filepath.Ext(tramite)
	base := strings.TrimSuffix(tramite, extension)
	if extension == "" {
		extension = ".pdf"
	}
	base = normalizeAnnexFilename(base)
	if base == "" {
		base = "DOCUMENTO"
	}
	name := fmt.Sprintf("%s_%s%s", base, time.Now().UTC().Format("20060102T150405.000000000"), extension)
	return atomicWritePrivateFile(filepath.Join(directory, name), contents)
}

func objectionAnnexBase(kind, custom string) (string, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	allowed := map[string]bool{
		"FICHA_TECNICA": true, "FACTURA_DISPOSITIVO": true, "FACTURAS_RESPALDO": true,
		"FACTURA_COMPRA": true, "PROTOCOLOS_MEDICOS": true, "PROTOCOLO_DETALLADO": true,
		"EXAMEN_ADICIONAL": true,
	}
	if allowed[kind] {
		return kind, nil
	}
	if kind != "OTRO" {
		return "", errors.New("Selecciona una categoría válida para el anexo.")
	}
	base := normalizeAnnexFilename(custom)
	if base == "" {
		return "", errors.New("Escribe un nombre descriptivo para el otro justificativo.")
	}
	return base, nil
}

func normalizeAnnexFilename(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(filepath.Ext(value), ".pdf") {
		value = strings.TrimSuffix(value, filepath.Ext(value))
	}
	value = norm.NFD.String(strings.ToUpper(value))
	var out strings.Builder
	separator := false
	for _, r := range value {
		if unicode.Is(unicode.M, r) {
			continue
		}
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
			separator = false
		} else if out.Len() > 0 && !separator {
			out.WriteByte('_')
			separator = true
		}
	}
	base := strings.Trim(out.String(), "_")
	if len(base) > 100 {
		base = strings.Trim(base[:100], "_")
	}
	return base
}

func nextSafeAnnexName(directory, base string) (string, error) {
	base = normalizeAnnexFilename(base)
	if base == "" {
		return "", errors.New("nombre de anexo vacío")
	}
	name := base + ".pdf"
	for index := 1; ; index++ {
		if _, err := os.Lstat(filepath.Join(directory, name)); errors.Is(err, os.ErrNotExist) {
			return name, nil
		} else if err != nil {
			return "", err
		}
		name = fmt.Sprintf("%s_%d.pdf", base, index)
	}
}

func validateObjectionCloseout(job stagedJob, packageRoot string) error {
	missing := objectionCloseoutMissing(job, packageRoot)
	if len(missing) > 0 {
		return errors.New(missing[0])
	}
	return nil
}

func objectionCloseoutMissing(job stagedJob, packageRoot string) []string {
	missing := make([]string, 0)
	requiredPDFs := []struct{ name, label string }{
		{"I_LIQUIDACION.pdf", "I_LIQUIDACION.pdf"},
		{"1. OFICIO DE PAGO.pdf", "1. OFICIO DE PAGO.pdf"},
		{"2. PLANILLA CONSOLIDADA.pdf", "2. PLANILLA CONSOLIDADA.pdf"},
	}
	for _, item := range requiredPDFs {
		if !validPDFFile(filepath.Join(packageRoot, item.name)) {
			missing = append(missing, fmt.Sprintf("Falta cargar %s en la raíz del paquete.", item.label))
		}
	}
	matrixName := objectionMatrixFilename(job)
	if !validXLSMFile(filepath.Join(packageRoot, matrixName)) {
		missing = append(missing, fmt.Sprintf("Falta cargar la matriz oficial %s en la raíz del paquete.", matrixName))
	}
	seen := make(map[string]objectionRecord, len(job.ObjectionRows))
	for _, row := range job.ObjectionRows {
		seen[row.Tramite] = row
	}
	tramites := make([]string, 0, len(seen))
	for tramite := range seen {
		tramites = append(tramites, tramite)
	}
	sort.Strings(tramites)
	if len(tramites) == 0 {
		missing = append(missing, "Selecciona al menos un trámite objetado.")
	}
	for _, tramite := range tramites {
		row := seen[tramite]
		if row.Posture != "ACEPTA" && row.Posture != "RECHAZA" {
			missing = append(missing, fmt.Sprintf("Falta registrar ACEPTA o RECHAZA para el trámite %s.", tramite))
		}
		path := filepath.Join(packageRoot, "4. EXPEDIENTES", row.PatientFolder, "P_INDIVIDUAL.pdf")
		if !validPDFFile(path) {
			missing = append(missing, fmt.Sprintf("Falta P_INDIVIDUAL.pdf válido para el trámite %s de %s.", tramite, row.Patient))
		}
		relativeResponse := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", row.PatientFolder, "P_INDIVIDUAL.pdf"))
		if !objectionPDFIncluded(job, relativeResponse) {
			missing = append(missing, fmt.Sprintf("P_INDIVIDUAL.pdf del trámite %s debe incluirse en el ZIP.", tramite))
		}
		coverage := filepath.Join(packageRoot, "4. EXPEDIENTES", row.PatientFolder, "C_COBERTURA.pdf")
		if !validPDFFile(coverage) {
			missing = append(missing, fmt.Sprintf("Falta C_COBERTURA.pdf válido para el trámite %s de %s.", tramite, row.Patient))
		}
		relativeCoverage := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", row.PatientFolder, "C_COBERTURA.pdf"))
		if !objectionPDFIncluded(job, relativeCoverage) {
			missing = append(missing, fmt.Sprintf("C_COBERTURA.pdf del trámite %s debe incluirse en el ZIP.", tramite))
		}
	}
	return missing
}

func validPDFFile(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer file.Close()
	var signature [5]byte
	_, err = io.ReadFull(file, signature[:])
	return err == nil && string(signature[:]) == "%PDF-"
}
