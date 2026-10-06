package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
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
	for _, tramite := range selected {
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
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
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
	id, err := newUploadJobID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo crear el identificador del espacio de objeciones.")
		return
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Username: session.username,
		Month: source.Month, Year: source.Year, Service: source.Service, ReceivedAt: time.Now().UTC(),
		IsObjections: true, ObjectionSourceID: source.ID, ObjectionRows: rows,
		StatusDetail: fmt.Sprintf("Espacio de objeciones separado con %d trámites seleccionados. Completa los documentos de cabecera antes de descargar el ZIP. Oracle permanece sin cambios.", len(rows)),
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
	if err != nil || !job.IsObjections || job.Status != "PROCESSED" {
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
	if err != nil || !job.IsObjections || job.Status != "PROCESSED" {
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
	existing := make(map[string]bool, len(job.ObjectionRows))
	for _, row := range job.ObjectionRows {
		existing[row.Tramite] = true
	}
	for _, row := range rows {
		if existing[row.Tramite] {
			writeError(w, http.StatusConflict, fmt.Sprintf("El trámite %s ya está en este espacio de objeciones.", row.Tramite))
			return
		}
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	patientRoot := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source), "4. EXPEDIENTES")
	previousRows := append([]objectionRecord(nil), job.ObjectionRows...)
	previousFiles := append([]stagedUpload(nil), job.Files...)
	previousMappings := make(map[string]int64, len(job.DocumentPlanillas))
	for path, planillaID := range job.DocumentPlanillas {
		previousMappings[path] = planillaID
	}
	if err := installObjectionRows(packageRoot, patientRoot, source, rows, &job); err != nil {
		job.DocumentPlanillas = previousMappings
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	job.ObjectionRows = append(job.ObjectionRows, rows...)
	matrixName := objectionMatrixFilename(job)
	matrixPath := filepath.Join(packageRoot, matrixName)
	previousMatrix, matrixErr := os.ReadFile(matrixPath)
	matrixExisted := matrixErr == nil
	if matrixErr != nil && !errors.Is(matrixErr, os.ErrNotExist) {
		cleanupObjectionRows(packageRoot, rows)
		job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
		writeError(w, http.StatusInternalServerError, "No se pudo revisar la matriz anterior.")
		return
	}
	if matrixExisted {
		if err := saveObjectionVersion(s.jobRoot(job.ID), matrixName, previousMatrix); err != nil {
			cleanupObjectionRows(packageRoot, rows)
			job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
			writeError(w, http.StatusInternalServerError, "No se pudo conservar la matriz anterior antes de agregar los trámites.")
			return
		}
		if err := os.Remove(matrixPath); err != nil {
			cleanupObjectionRows(packageRoot, rows)
			job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
			writeError(w, http.StatusInternalServerError, "No se pudo invalidar la matriz anterior.")
			return
		}
	}
	job.Files = removeObjectionFile(job.Files, "objection_matriz")
	job.StatusDetail = fmt.Sprintf("Espacio separado con %d trámites objetados. Puedes seguir agregando pacientes; vuelve a cargar la matriz oficial después de cada cambio. Oracle permanece sin cambios.", len(job.ObjectionRows))
	if err := s.saveStagedJob(job); err != nil {
		cleanupObjectionRows(packageRoot, rows)
		job.ObjectionRows, job.Files, job.DocumentPlanillas = previousRows, previousFiles, previousMappings
		if matrixExisted {
			_ = atomicWritePrivateFile(matrixPath, previousMatrix)
		}
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la selección; se restauró la matriz anterior.")
		return
	}
	writeJSON(w, http.StatusOK, jobResponse(s, job, filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job)), nil))
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
	if err != nil || !job.IsObjections || job.Status != "PROCESSED" {
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
	savedFilename := ""
	switch kind {
	case "respuesta":
		posture := strings.ToUpper(strings.TrimSpace(r.FormValue("postura")))
		if posture != "ACEPTA" && posture != "RECHAZA" {
			writeError(w, http.StatusBadRequest, "Selecciona ACEPTA o RECHAZA como postura.")
			return
		}
		responsePath := filepath.Join(packageRoot, "4. EXPEDIENTES", record.PatientFolder, "P_INDIVIDUAL.pdf")
		previous, readErr := os.ReadFile(responsePath)
		existed := readErr == nil
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			writeError(w, http.StatusInternalServerError, "No se pudo leer la versión anterior de P_INDIVIDUAL.pdf.")
			return
		}
		if existed {
			if err := saveObjectionVersion(s.jobRoot(job.ID), "P_INDIVIDUAL_"+record.Tramite, previous); err != nil {
				writeError(w, http.StatusInternalServerError, "No se pudo conservar la versión anterior de P_INDIVIDUAL.pdf.")
				return
			}
		}
		previousRows := append([]objectionRecord(nil), job.ObjectionRows...)
		previousMappings := make(map[string]int64, len(job.DocumentPlanillas))
		for path, planillaID := range job.DocumentPlanillas {
			previousMappings[path] = planillaID
		}
		if job.DocumentPlanillas == nil {
			job.DocumentPlanillas = make(map[string]int64)
		}
		relative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", record.PatientFolder, "P_INDIVIDUAL.pdf"))
		job.DocumentPlanillas[relative] = record.PlanillaID
		record.Posture = posture
		if err := atomicWritePrivateFile(responsePath, data); err != nil {
			job.ObjectionRows = previousRows
			job.DocumentPlanillas = previousMappings
			writeError(w, http.StatusInternalServerError, "No se pudo guardar P_INDIVIDUAL.pdf.")
			return
		}
		if err := s.saveStagedJob(job); err != nil {
			job.ObjectionRows = previousRows
			job.DocumentPlanillas = previousMappings
			if existed {
				_ = atomicWritePrivateFile(responsePath, previous)
			} else {
				_ = os.Remove(responsePath)
			}
			writeError(w, http.StatusInternalServerError, "No se pudo registrar la postura; se restauró la versión anterior de P_INDIVIDUAL.pdf.")
			return
		}
		savedFilename = "P_INDIVIDUAL.pdf"
	case "cobertura":
		coveragePath := filepath.Join(packageRoot, "4. EXPEDIENTES", record.PatientFolder, "C_COBERTURA.pdf")
		previous, readErr := os.ReadFile(coveragePath)
		existed := readErr == nil
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			writeError(w, http.StatusInternalServerError, "No se pudo leer la cobertura anterior.")
			return
		}
		if existed {
			if err := saveObjectionVersion(s.jobRoot(job.ID), "C_COBERTURA_"+record.Tramite, previous); err != nil {
				writeError(w, http.StatusInternalServerError, "No se pudo conservar la cobertura anterior.")
				return
			}
		}
		previousMappings := make(map[string]int64, len(job.DocumentPlanillas))
		for path, planillaID := range job.DocumentPlanillas {
			previousMappings[path] = planillaID
		}
		if job.DocumentPlanillas == nil {
			job.DocumentPlanillas = make(map[string]int64)
		}
		relative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", record.PatientFolder, "C_COBERTURA.pdf"))
		job.DocumentPlanillas[relative] = record.PlanillaID
		if err := atomicWritePrivateFile(coveragePath, data); err != nil {
			job.DocumentPlanillas = previousMappings
			writeError(w, http.StatusInternalServerError, "No se pudo guardar C_COBERTURA.pdf.")
			return
		}
		if err := s.saveStagedJob(job); err != nil {
			job.DocumentPlanillas = previousMappings
			if existed {
				_ = atomicWritePrivateFile(coveragePath, previous)
			} else {
				_ = os.Remove(coveragePath)
			}
			writeError(w, http.StatusInternalServerError, "No se pudo registrar la cobertura; se restauró el archivo anterior.")
			return
		}
		savedFilename = "C_COBERTURA.pdf"
	case "anexo":
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
		savedFilename = name
	default:
		writeError(w, http.StatusBadRequest, "El tipo debe ser respuesta, cobertura o anexo.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "pdi_tramite": tramite, "tipo": kind, "nombre_archivo": savedFilename, "postura": record.Posture, "message": "Documento guardado en el espacio aislado de objeciones. Oracle no fue modificado."})
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
	requiredPDFs := []struct{ name, label string }{
		{"I_LIQUIDACION.pdf", "I_LIQUIDACION.pdf"},
		{"1. OFICIO DE PAGO.pdf", "1. OFICIO DE PAGO.pdf"},
		{"2. PLANILLA CONSOLIDADA.pdf", "2. PLANILLA CONSOLIDADA.pdf"},
	}
	for _, item := range requiredPDFs {
		if !validPDFFile(filepath.Join(packageRoot, item.name)) {
			return fmt.Errorf("Falta cargar %s en la raíz del paquete.", item.label)
		}
	}
	matrixName := objectionMatrixFilename(job)
	if !validXLSMFile(filepath.Join(packageRoot, matrixName)) {
		return fmt.Errorf("Falta cargar la matriz oficial %s en la raíz del paquete.", matrixName)
	}
	seen := make(map[string]objectionRecord)
	for _, row := range job.ObjectionRows {
		seen[row.Tramite] = row
	}
	for tramite, row := range seen {
		if row.Posture != "ACEPTA" && row.Posture != "RECHAZA" {
			return fmt.Errorf("Falta registrar ACEPTA o RECHAZA para el trámite %s.", tramite)
		}
		path := filepath.Join(packageRoot, "4. EXPEDIENTES", row.PatientFolder, "P_INDIVIDUAL.pdf")
		if !validPDFFile(path) {
			return fmt.Errorf("Falta P_INDIVIDUAL.pdf válido para el trámite %s de %s.", tramite, row.Patient)
		}
		coverage := filepath.Join(packageRoot, "4. EXPEDIENTES", row.PatientFolder, "C_COBERTURA.pdf")
		if !validPDFFile(coverage) {
			return fmt.Errorf("Falta C_COBERTURA.pdf válido para el trámite %s de %s.", tramite, row.Patient)
		}
	}
	return nil
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
