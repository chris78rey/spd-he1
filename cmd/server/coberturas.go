package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxCoverageBatch = 10
const maxCoverageDownload = 500

var coverageCedulaPattern = regexp.MustCompile(`^\d{10}$`)

type coveragePlanilla struct {
	ID               int64                  `json:"pdi_id"`
	Tramite          string                 `json:"pdi_tramite"`
	Patient          string                 `json:"paciente"`
	PatientFolder    string                 `json:"carpeta_paciente"`
	CareUntil        string                 `json:"fecha_hasta"`
	Cedula           string                 `json:"cedula"`
	Minor            string                 `json:"menor_edad"`
	Dependent1       string                 `json:"dependiente_01"`
	Dependent2       string                 `json:"dependiente_02"`
	CoverageStatus   string                 `json:"pdi_cobertura"`
	HasPDF           bool                   `json:"hoja_generada"`
	ManualRequired   bool                   `json:"descarga_manual"`
	ManualReason     string                 `json:"motivo_manual,omitempty"`
	ManualQueryDate  string                 `json:"fecha_consulta_manual,omitempty"`
	ManualCustomDate bool                   `json:"fecha_personalizada_manual,omitempty"`
	CoverageDates    []string               `json:"fechas_cobertura,omitempty"`
	ManualMembers    []coverageManualMember `json:"coberturas_manual,omitempty"`
}

type coverageFailure struct {
	Reason         string   `json:"motivo"`
	Manual         bool     `json:"manual"`
	QueryDate      string   `json:"fecha_consulta,omitempty"`
	CustomDate     bool     `json:"fecha_personalizada,omitempty"`
	PendingCedulas []string `json:"cedulas_pendientes,omitempty"`
}

type coverageFailureItem struct {
	PlanillaID int64  `json:"pdi_id"`
	Tramite    string `json:"pdi_tramite"`
	Reason     string `json:"motivo"`
	Manual     bool   `json:"manual"`
}

type coverageManualMember struct {
	Cedula  string `json:"cedula"`
	Fecha   string `json:"fecha"`
	Adjunta bool   `json:"adjunta"`
}

type coverageGenerateRequest struct {
	PlanillaIDs []int64 `json:"pdi_ids"`
	QueryDate   string  `json:"fecha_consulta,omitempty"`
}

type coverageMember struct {
	Cedula string
	Fecha  string
}

type coverageDateIndex struct {
	HasPDF     bool
	HasUndated bool
	Dates      map[string]bool
}

func (s *server) listCoveragePlanillas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/coberturas/planillas/")
	job, err := s.coverageJob(id)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	rows, err := s.coverageRows(ctx, job)
	if err != nil {
		log.Printf("coberturas %s: consulta Oracle fallida (%T)", id, err)
		writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar las planillas de cobertura de este lote en Oracle.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"job_id": job.ID, "tipo_servicio": job.Service, "mes": job.Month, "anio": job.Year,
		"planillas": rows,
	})
}

func (s *server) generateCoverageSheets(w http.ResponseWriter, r *http.Request) {
	s.generateCoverageSheetsWithDate(w, r, false)
}

func (s *server) generateCoverageSheetsAtDate(w http.ResponseWriter, r *http.Request) {
	s.generateCoverageSheetsWithDate(w, r, true)
}

func (s *server) generateCoverageSheetsWithDate(w http.ResponseWriter, r *http.Request, useChosenDate bool) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	prefix := "/api/v1/coberturas/generar/"
	if useChosenDate {
		prefix = "/api/v1/coberturas/generar-fecha/"
	}
	id := strings.TrimPrefix(r.URL.Path, prefix)
	job, err := s.coverageJob(id)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var input coverageGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.PlanillaIDs) == 0 || len(input.PlanillaIDs) > maxCoverageBatch {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Selecciona entre 1 y %d planillas.", maxCoverageBatch))
		return
	}
	queryDate := ""
	if useChosenDate {
		var ok bool
		queryDate, ok = normalizeCoverageDate(input.QueryDate)
		if !ok {
			writeError(w, http.StatusBadRequest, "Selecciona una fecha válida para consultar las coberturas.")
			return
		}
	}
	selected := make(map[int64]bool, len(input.PlanillaIDs))
	for _, pdiID := range input.PlanillaIDs {
		if pdiID <= 0 || selected[pdiID] {
			writeError(w, http.StatusBadRequest, "La selección contiene identificadores de planilla inválidos o repetidos.")
			return
		}
		selected[pdiID] = true
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	rows, err := s.coverageRows(ctx, job)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar las planillas de cobertura de este lote en Oracle.")
		return
	}
	byID := make(map[int64]coveragePlanilla, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	chosen := make([]coveragePlanilla, 0, len(selected))
	for pdiID := range selected {
		row, ok := byID[pdiID]
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "Una planilla seleccionada no pertenece al año, mes, servicio y ZIP de este lote.")
			return
		}
		if !useChosenDate && strings.EqualFold(row.CoverageStatus, "S") && !coverageHasDate(row, row.CareUntil) {
			writeError(w, http.StatusConflict, "Oracle ya marca una de las planillas como cubierta, pero el lote no contiene su PDF. Revisa el expediente antes de continuar.")
			return
		}
		chosen = append(chosen, row)
	}
	sort.Slice(chosen, func(i, j int) bool { return chosen[i].ID < chosen[j].ID })

	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES")); err != nil {
		writeError(w, http.StatusConflict, "El expediente del ZIP no tiene carpetas de pacientes preparadas.")
		return
	}
	sourceRoot := filepath.Join(s.jobSourcesDir(job.ID), "externos")
	if err := os.MkdirAll(sourceRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carpeta privada de generación.")
		return
	}
	newDocuments := make([]workspaceDocument, 0)
	createdTargets := make([]string, 0)
	createdSources := make([]string, 0)
	rollback := func() {
		for _, path := range createdTargets {
			_ = os.Remove(path)
		}
		for _, path := range createdSources {
			_ = os.Remove(path)
		}
	}
	generatedIDs := make([]int64, 0, len(chosen))
	failures := make([]coverageFailureItem, 0)
	if job.CoverageFailures == nil {
		job.CoverageFailures = make(map[int64]coverageFailure)
	}
	for _, row := range chosen {
		rowQueryDate := row.CareUntil
		members := coverageMembersAt(row, rowQueryDate)
		if useChosenDate {
			rowQueryDate = queryDate
			members = missingCoverageMembersForDate(s, job, row, queryDate)
			if len(members) == 0 && len(coverageMembersAt(row, queryDate)) > 0 {
				generatedIDs = append(generatedIDs, row.ID)
				delete(job.CoverageFailures, row.ID)
				continue
			}
		} else if coverageHasDate(row, row.CareUntil) {
			generatedIDs = append(generatedIDs, row.ID)
			delete(job.CoverageFailures, row.ID)
			continue
		}
		documents, targets, sources, failure := s.generateCoveragePlanilla(ctx, job, packageRoot, sourceRoot, row, rowQueryDate, members)
		if failure != nil {
			failure.QueryDate, failure.CustomDate = rowQueryDate, useChosenDate
			failure.PendingCedulas = coverageMemberCedulas(members)
			job.CoverageFailures[row.ID] = *failure
			failures = append(failures, coverageFailureItem{PlanillaID: row.ID, Tramite: row.Tramite, Reason: failure.Reason, Manual: failure.Manual})
			continue
		}
		newDocuments = append(newDocuments, documents...)
		createdTargets = append(createdTargets, targets...)
		createdSources = append(createdSources, sources...)
		generatedIDs = append(generatedIDs, row.ID)
		delete(job.CoverageFailures, row.ID)
	}
	previousExternal := append([]workspaceDocument(nil), job.ExternalPDFs...)
	job.ExternalPDFs = append(job.ExternalPDFs, newDocuments...)
	if err := s.saveStagedJob(job); err != nil {
		job.ExternalPDFs = previousExternal
		rollback()
		writeError(w, http.StatusInternalServerError, "Las hojas se generaron, pero no se pudo guardar su registro en el lote.")
		return
	}
	if len(newDocuments) > 0 {
		if err := s.syncOracleWorkspace(ctx, &job, packageRoot); err != nil {
			job.ExternalPDFs = previousExternal
			_ = s.saveStagedJob(job)
			rollback()
			log.Printf("coberturas %s: sincronización Oracle fallida (%T): %v", job.ID, err, err)
			writeError(w, http.StatusInternalServerError, "Las hojas se generaron, pero no se pudo registrar el expediente en Oracle.")
			return
		}
	}
	if !useChosenDate {
		tx, err := s.serviceDB.BeginTx(ctx, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo iniciar la actualización de cobertura en Oracle.")
			return
		}
		for _, row := range chosen {
			if !containsInt64(generatedIDs, row.ID) || strings.EqualFold(row.CoverageStatus, "S") {
				continue
			}
			result, updateErr := tx.ExecContext(ctx, "UPDATE "+oracleTableName(s.schema)+" SET PDI_COBERTURA = 'S' WHERE PDI_ID = :id AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP'", sql.Named("id", row.ID))
			if updateErr != nil {
				_ = tx.Rollback()
				log.Printf("coberturas %s PDI_ID=%d: no se pudo marcar PDI_COBERTURA (%T)", job.ID, row.ID, updateErr)
				writeError(w, http.StatusInternalServerError, "Las hojas quedaron guardadas en el expediente, pero Oracle no confirmó el estado de cobertura.")
				return
			}
			affected, rowsErr := result.RowsAffected()
			if rowsErr != nil || affected != 1 {
				_ = tx.Rollback()
				writeError(w, http.StatusConflict, "Oracle no confirmó exactamente una planilla al actualizar su estado de cobertura.")
				return
			}
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "Oracle no confirmó la actualización de cobertura.")
			return
		}
	}
	response := map[string]any{"job_id": job.ID, "generadas": generatedIDs, "errores": failures, "oracle_actualizado": !useChosenDate}
	if useChosenDate {
		response["fecha_consulta"] = queryDate
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) generateCoveragePlanilla(ctx context.Context, job stagedJob, packageRoot, sourceRoot string, row coveragePlanilla, queryDate string, members []coverageMember) ([]workspaceDocument, []string, []string, *coverageFailure) {
	if len(members) == 0 {
		return nil, nil, nil, &coverageFailure{Reason: "La planilla no tiene una cédula válida de 10 dígitos o una fecha de atención.", Manual: false}
	}
	patientFolder := normalizePatientFolder(row.Patient)
	if patientFolder == "" {
		return nil, nil, nil, &coverageFailure{Reason: "Oracle no devuelve un nombre de paciente utilizable para este trámite.", Manual: false}
	}
	patientDir := filepath.Join(packageRoot, "4. EXPEDIENTES", patientFolder)
	if info, err := os.Stat(patientDir); err != nil || !info.IsDir() {
		return nil, nil, nil, &coverageFailure{Reason: "El ZIP no produjo la carpeta de expediente de este trámite.", Manual: false}
	}
	tempRoot, err := os.MkdirTemp(s.jobRoot(job.ID), ".coverage-")
	if err != nil {
		return nil, nil, nil, &coverageFailure{Reason: "No se pudo preparar la generación temporal del PDF.", Manual: false}
	}
	defer os.RemoveAll(tempRoot)
	generated := make([]string, 0, len(members))
	for index, member := range members {
		outputDir := filepath.Join(tempRoot, strconv.Itoa(index+1))
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			return nil, nil, nil, &coverageFailure{Reason: "No se pudo preparar una salida temporal.", Manual: false}
		}
		outputName := fmt.Sprintf("cobertura_%d_%d", row.ID, index+1)
		cmd := exec.CommandContext(ctx, envOr("NODE_BIN", "node"), "scripts/cobertura/generate_pdf.cjs", "--cedula", member.Cedula, "--fecha", member.Fecha, "--output_name", outputName, "--output_dir", outputDir)
		cmd.Dir = "."
		if _, err := cmd.CombinedOutput(); err != nil {
			log.Printf("coberturas %s PDI_ID=%d: consulta al portal fallida (%T)", job.ID, row.ID, err)
			return nil, nil, nil, &coverageFailure{Reason: "El portal MSP no respondió correctamente. Descarga manualmente las hojas de esta planilla.", Manual: true}
		}
		pdfPath := filepath.Join(outputDir, outputName+".pdf")
		if err := validateCoveragePDF(pdfPath, member.Cedula); err != nil {
			log.Printf("coberturas %s PDI_ID=%d: PDF automático rechazado (%T)", job.ID, row.ID, err)
			return nil, nil, nil, &coverageFailure{Reason: "La descarga no produjo un PDF legible o no corresponde a la cédula consultada. Descarga manualmente las hojas de esta planilla.", Manual: true}
		}
		generated = append(generated, pdfPath)
	}
	documents := make([]workspaceDocument, 0, len(members))
	targets := make([]string, 0, len(members))
	sources := make([]string, 0, len(members))
	rollback := func() {
		for _, path := range targets {
			_ = os.Remove(path)
		}
		for _, path := range sources {
			_ = os.Remove(path)
		}
	}
	for index, member := range members {
		name, err := nextMSPPDFName(patientDir, "C_COBERTURA.pdf", "")
		if err != nil {
			rollback()
			return nil, nil, nil, &coverageFailure{Reason: "No se pudo asignar un nombre al PDF dentro del expediente.", Manual: false}
		}
		relative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", patientFolder, name))
		target, err := safeWorkspacePath(packageRoot, relative)
		if err != nil {
			rollback()
			return nil, nil, nil, &coverageFailure{Reason: "La ruta de salida de la cobertura no es válida.", Manual: false}
		}
		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			rollback()
			return nil, nil, nil, &coverageFailure{Reason: "No se pudo identificar el PDF generado.", Manual: false}
		}
		storedName := fmt.Sprintf("cobertura-%x.pdf", idBytes)
		source := filepath.Join(sourceRoot, storedName)
		if err := copyPrivateFile(generated[index], source); err != nil {
			rollback()
			return nil, nil, nil, &coverageFailure{Reason: "No se pudo guardar el PDF generado.", Manual: false}
		}
		sources = append(sources, source)
		if err := copyPrivateFile(source, target); err != nil {
			rollback()
			return nil, nil, nil, &coverageFailure{Reason: "No se pudo añadir el PDF al expediente.", Manual: false}
		}
		targets = append(targets, target)
		info, _ := os.Stat(target)
		documents = append(documents, workspaceDocument{ID: strings.TrimSuffix(storedName, ".pdf"), StoredName: storedName, OriginalName: "C_COBERTURA.pdf", RelativePath: relative, Size: info.Size(), PlanillaID: row.ID, CoverageCedula: member.Cedula, CoverageDate: queryDate})
	}
	return documents, targets, sources, nil
}

func validateCoveragePDF(path, expectedCedula string) error {
	if err := validateStagedFile(path, ".pdf"); err != nil {
		return err
	}
	text, err := extractPDFText(path)
	if err != nil {
		return err
	}
	compact := strings.Join(strings.Fields(normalizeOCRText(text)), "")
	if !strings.Contains(compact, expectedCedula) {
		return errors.New("la identificación no coincide")
	}
	return nil
}

func containsInt64(values []int64, value int64) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func (s *server) downloadCoverageSheets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	job, err := s.coverageJob(strings.TrimPrefix(r.URL.Path, "/api/v1/coberturas/descargar/"))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	queryDate := ""
	if rawDate := r.URL.Query().Get("fecha_consulta"); rawDate != "" {
		var ok bool
		queryDate, ok = normalizeCoverageDate(rawDate)
		if !ok {
			writeError(w, http.StatusBadRequest, "La fecha para descargar las coberturas no es válida.")
			return
		}
	}
	ids := strings.Split(r.URL.Query().Get("pdi_ids"), ",")
	if len(ids) == 0 || len(ids) > maxCoverageDownload {
		writeError(w, http.StatusBadRequest, "La descarga no contiene una selección válida.")
		return
	}
	rows, err := s.coverageRows(r.Context(), job)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar las planillas de este lote.")
		return
	}
	byID := make(map[int64]coveragePlanilla, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	added := 0
	for _, rawID := range ids {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil {
			_ = writer.Close()
			writeError(w, http.StatusBadRequest, "La descarga contiene un identificador inválido.")
			return
		}
		row, ok := byID[id]
		if !ok || (queryDate == "" && !row.HasPDF) {
			_ = writer.Close()
			writeError(w, http.StatusConflict, "Una de las planillas no tiene todas sus hojas guardadas en el expediente.")
			return
		}
		for _, doc := range s.coverageDocumentsForPlanilla(job, id) {
			if queryDate != "" && coverageDocumentDate(doc, row) != queryDate {
				continue
			}
			path, err := safeWorkspacePath(packageRoot, doc.RelativePath)
			if err != nil {
				continue
			}
			if err := validateStagedFile(path, ".pdf"); err != nil {
				continue
			}
			file, err := os.Open(path)
			if err != nil {
				continue
			}
			entry, err := writer.Create(filepath.ToSlash(doc.RelativePath))
			if err == nil {
				if _, err = io.Copy(entry, file); err == nil {
					added++
				}
			}
			_ = file.Close()
		}
	}
	if err := writer.Close(); err != nil || added == 0 {
		writeError(w, http.StatusConflict, "No se encontraron hojas válidas para descargar.")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	filename := fmt.Sprintf("COBERTURAS_%s_%s_%s", job.Service, job.Month, job.Year)
	if queryDate != "" {
		filename += "_FECHA_" + strings.ReplaceAll(queryDate, "-", "")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive.Bytes())
}

func (s *server) uploadManualCoverageSheets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	job, err := s.coverageJob(strings.TrimPrefix(r.URL.Path, "/api/v1/coberturas/manual/"))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer la carga manual de PDFs.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	pdiID, err := strconv.ParseInt(r.FormValue("pdi_id"), 10, 64)
	if err != nil || pdiID <= 0 {
		writeError(w, http.StatusBadRequest, "Selecciona una planilla válida.")
		return
	}
	rows, err := s.coverageRows(r.Context(), job)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar las planillas de este lote.")
		return
	}
	var row *coveragePlanilla
	for index := range rows {
		if rows[index].ID == pdiID {
			row = &rows[index]
			break
		}
	}
	failure, hasFailure := job.CoverageFailures[pdiID]
	customDate := hasFailure && failure.CustomDate
	if row == nil || (strings.EqualFold(row.CoverageStatus, "S") && !customDate) {
		writeError(w, http.StatusConflict, "La planilla no pertenece al ZIP o ya está marcada como cubierta.")
		return
	}
	queryDate := row.CareUntil
	if hasFailure && failure.QueryDate != "" {
		queryDate = failure.QueryDate
	}
	members := filterCoverageMembers(coverageMembersAt(*row, queryDate), failure.PendingCedulas)
	files := r.MultipartForm.File["pdf_files"]
	if len(members) == 0 || len(files) != len(members) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Adjunta un PDF del portal MSP para cada persona de la planilla (%d requeridos).", len(members)))
		return
	}
	memberByID := make(map[string]coverageMember, len(members))
	for _, member := range members {
		memberByID[member.Cedula] = member
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	patientFolder := normalizePatientFolder(row.Patient)
	patientDir := filepath.Join(packageRoot, "4. EXPEDIENTES", patientFolder)
	sourceRoot := filepath.Join(s.jobSourcesDir(job.ID), "externos")
	if err := os.MkdirAll(sourceRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el almacenamiento del PDF.")
		return
	}
	stage, err := os.MkdirTemp(s.jobRoot(job.ID), ".coverage-manual-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la validación del PDF.")
		return
	}
	defer os.RemoveAll(stage)
	documents := make([]workspaceDocument, 0, len(files))
	sources := make([]string, 0, len(files))
	targets := make([]string, 0, len(files))
	rollback := func() {
		for _, path := range sources {
			_ = os.Remove(path)
		}
		for _, path := range targets {
			_ = os.Remove(path)
		}
	}
	seen := make(map[string]bool)
	for index, header := range files {
		staged := filepath.Join(stage, fmt.Sprintf("manual-%02d.pdf", index+1))
		if _, err := saveUploadPart(header, staged); err != nil {
			rollback()
			writeError(w, http.StatusBadRequest, "No se pudo guardar uno de los PDFs seleccionados.")
			return
		}
		if err := validateStagedFile(staged, ".pdf"); err != nil {
			rollback()
			writeError(w, http.StatusBadRequest, "Uno de los archivos no es un PDF válido descargado desde el portal.")
			return
		}
		text, err := extractPDFText(staged)
		if err != nil {
			rollback()
			writeError(w, http.StatusBadRequest, "No se pudo leer la identificación de uno de los PDFs.")
			return
		}
		compact := strings.Join(strings.Fields(normalizeOCRText(text)), "")
		matched := ""
		for cedula := range memberByID {
			if strings.Contains(compact, cedula) {
				if matched != "" {
					rollback()
					writeError(w, http.StatusBadRequest, "El PDF coincide con más de una persona; revisa los archivos descargados.")
					return
				}
				matched = cedula
			}
		}
		if matched == "" || seen[matched] {
			rollback()
			writeError(w, http.StatusBadRequest, "Un PDF no corresponde a una cédula pendiente de esta planilla o está repetido.")
			return
		}
		seen[matched] = true
		name, err := nextMSPPDFName(patientDir, "C_COBERTURA.pdf", "")
		if err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo elegir un nombre para el PDF manual.")
			return
		}
		relative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", patientFolder, name))
		target, err := safeWorkspacePath(packageRoot, relative)
		if err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "La ruta de salida no es válida.")
			return
		}
		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo identificar el PDF.")
			return
		}
		stored := fmt.Sprintf("cobertura-manual-%x.pdf", idBytes)
		source := filepath.Join(sourceRoot, stored)
		if err := copyPrivateFile(staged, source); err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo guardar el PDF manual.")
			return
		}
		sources = append(sources, source)
		if err := copyPrivateFile(source, target); err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo añadir el PDF al expediente.")
			return
		}
		targets = append(targets, target)
		info, _ := os.Stat(target)
		documents = append(documents, workspaceDocument{ID: strings.TrimSuffix(stored, ".pdf"), StoredName: stored, OriginalName: "C_COBERTURA.pdf", RelativePath: relative, Size: info.Size(), PlanillaID: row.ID, CoverageCedula: matched, CoverageDate: queryDate})
	}
	if len(seen) != len(members) {
		rollback()
		writeError(w, http.StatusBadRequest, "Falta al menos una cobertura manual para un miembro de la planilla.")
		return
	}
	previousExternal := append([]workspaceDocument(nil), job.ExternalPDFs...)
	job.ExternalPDFs = append(job.ExternalPDFs, documents...)
	delete(job.CoverageFailures, row.ID)
	if err := s.saveStagedJob(job); err != nil {
		job.ExternalPDFs = previousExternal
		rollback()
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el registro de la carga manual.")
		return
	}
	if err := s.syncOracleWorkspace(r.Context(), &job, packageRoot); err != nil {
		job.ExternalPDFs = previousExternal
		_ = s.saveStagedJob(job)
		rollback()
		writeError(w, http.StatusInternalServerError, "Los PDFs se guardaron, pero no se pudieron sincronizar con Oracle.")
		return
	}
	if !customDate {
		result, err := s.serviceDB.ExecContext(r.Context(), "UPDATE "+oracleTableName(s.schema)+" SET PDI_COBERTURA = 'S' WHERE PDI_ID = :id AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP'", sql.Named("id", row.ID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Los PDFs quedaron en el expediente, pero Oracle no confirmó PDI_COBERTURA.")
			return
		}
		if affected, err := result.RowsAffected(); err != nil || affected != 1 {
			writeError(w, http.StatusConflict, "Oracle no confirmó una planilla al actualizar PDI_COBERTURA.")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "pdi_id": row.ID, "adjuntadas": len(documents), "fecha_consulta": queryDate, "oracle_actualizado": !customDate})
}

func (s *server) coverageJob(id string) (stagedJob, error) {
	if !validJobID(id) {
		return stagedJob{}, errors.New("El identificador del lote no es válido.")
	}
	job, err := s.loadStagedJob(id)
	if err != nil || (job.Status != "PROCESSED" && job.Status != "INCOMPLETE") {
		return stagedJob{}, errors.New("Abre un período cuyo ZIP ya se haya preparado antes de generar coberturas.")
	}
	if job.IsObjections {
		return stagedJob{}, errors.New("Los espacios de objeciones no admiten generar coberturas del primer ingreso.")
	}
	if !s.hasClinicalSource(job) {
		return stagedJob{}, errors.New("Este período ya no conserva el ZIP fuente.")
	}
	return job, nil
}

func (s *server) coverageRows(ctx context.Context, job stagedJob) ([]coveragePlanilla, error) {
	zipPath := filepath.Join(s.jobSourcesDir(job.ID), "lote.zip")
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	tramites := make(map[string]bool)
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		parts := strings.Split(strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/"), "/")
		if len(parts) != 2 || !tramitePattern.MatchString(parts[0]) {
			continue
		}
		mapped := strings.TrimSpace(job.TramiteMappings[parts[0]])
		if mapped == "" {
			mapped = parts[0]
		}
		tramites[mapped] = true
	}
	_ = archive.Close()
	if len(tramites) == 0 {
		return []coveragePlanilla{}, nil
	}
	table := oracleTableName(s.schema)
	docTable := oracleQualified(s.schema, pdiDocumentTable)
	query := "SELECT p.PDI_ID, TO_CHAR(p.PDI_TRAMITE), p.PDI_PACIENTE, TO_CHAR(p.PDI_FECHA_HASTA, 'YYYY-MM-DD'), p.PDI_CEDULA, p.PDI_MENOR_EDAD, p.PDI_DEPENDIENTE_01, p.PDI_DEPENDIENTE_02, " +
		"CASE WHEN p.PDI_COBERTURA = 'S' AND p.PDI_PATH IS NULL " +
		"AND EXISTS (SELECT 1 FROM " + docTable + " d WHERE d.PDI_ID = p.PDI_ID AND d.PDD_ESTADO = 'ELIMINADO') " +
		"AND NOT EXISTS (SELECT 1 FROM " + docTable + " a WHERE a.PDI_ID = p.PDI_ID AND a.PDD_ESTADO IN ('VIGENTE', 'PENDIENTE')) " +
		"THEN 'N' ELSE p.PDI_COBERTURA END " +
		"FROM " + table + " p WHERE p.PDI_MES = :mes AND p.PDI_ANIO = :anio AND p.PDI_PLANILLADO = 'S' AND p.PDI_ASEGURADORA = 'MSP'"
	rows, err := s.serviceDB.QueryContext(ctx, query, sql.Named("mes", job.Month), sql.Named("anio", job.Year))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	coverageIndex := s.indexCoverageDocuments(job)
	result := make([]coveragePlanilla, 0)
	for rows.Next() {
		var row coveragePlanilla
		var rawID, rawTramite, patient, date, cedula, minor, dep1, dep2, coverage sql.NullString
		if err := rows.Scan(&rawID, &rawTramite, &patient, &date, &cedula, &minor, &dep1, &dep2, &coverage); err != nil {
			return nil, err
		}
		row.ID, err = strconv.ParseInt(strings.TrimSpace(rawID.String), 10, 64)
		if err != nil || row.ID <= 0 {
			continue
		}
		row.Tramite = strings.TrimSpace(rawTramite.String)
		if !tramites[row.Tramite] {
			continue
		}
		row.Patient, row.CareUntil = strings.TrimSpace(patient.String), strings.TrimSpace(date.String)
		row.PatientFolder = normalizePatientFolder(row.Patient)
		row.Cedula, row.Minor = strings.TrimSpace(cedula.String), strings.ToUpper(strings.TrimSpace(minor.String))
		row.Dependent1, row.Dependent2 = strings.TrimSpace(dep1.String), strings.TrimSpace(dep2.String)
		row.CoverageStatus = strings.ToUpper(strings.TrimSpace(coverage.String))
		if indexed := coverageIndex[row.ID]; indexed != nil {
			row.HasPDF = indexed.HasPDF
			if indexed.HasUndated {
				if date, ok := normalizeCoverageDate(row.CareUntil); ok {
					indexed.Dates[date] = true
				}
			}
			row.CoverageDates = make([]string, 0, len(indexed.Dates))
			for date := range indexed.Dates {
				row.CoverageDates = append(row.CoverageDates, date)
			}
			sort.Strings(row.CoverageDates)
		}
		if failure, ok := job.CoverageFailures[row.ID]; ok {
			row.ManualRequired, row.ManualReason = failure.Manual, failure.Reason
			row.ManualQueryDate, row.ManualCustomDate = failure.QueryDate, failure.CustomDate
		}
		if row.ManualRequired {
			attached := make(map[string]bool)
			for _, doc := range job.ExternalPDFs {
				if doc.PlanillaID == row.ID && coverageDocumentDate(doc, row) == row.ManualQueryDate {
					attached[doc.CoverageCedula] = true
				}
			}
			members := filterCoverageMembers(coverageMembersAt(row, row.ManualQueryDate), job.CoverageFailures[row.ID].PendingCedulas)
			for _, member := range members {
				row.ManualMembers = append(row.ManualMembers, coverageManualMember{Cedula: member.Cedula, Fecha: member.Fecha, Adjunta: attached[member.Cedula]})
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Tramite < result[j].Tramite })
	return result, nil
}

func normalizeCoverageDate(value string) (string, bool) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		return "", false
	}
	return value, true
}

func coverageMembers(row coveragePlanilla) []coverageMember {
	return coverageMembersAt(row, row.CareUntil)
}

func coverageMembersAt(row coveragePlanilla, queryDate string) []coverageMember {
	queryDate, ok := normalizeCoverageDate(queryDate)
	if !ok {
		return nil
	}
	members := make([]coverageMember, 0, 3)
	seen := make(map[string]bool)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if coverageCedulaPattern.MatchString(value) && !seen[value] {
			seen[value] = true
			members = append(members, coverageMember{Cedula: value, Fecha: queryDate})
		}
	}
	add(row.Cedula)
	if row.Minor == "S" {
		add(row.Dependent1)
		add(row.Dependent2)
	}
	return members
}

func coverageMemberCedulas(members []coverageMember) []string {
	cedulas := make([]string, 0, len(members))
	for _, member := range members {
		cedulas = append(cedulas, member.Cedula)
	}
	return cedulas
}

func filterCoverageMembers(members []coverageMember, selectedCedulas []string) []coverageMember {
	if len(selectedCedulas) == 0 {
		return members
	}
	selected := make(map[string]bool, len(selectedCedulas))
	for _, cedula := range selectedCedulas {
		selected[cedula] = true
	}
	filtered := make([]coverageMember, 0, len(selectedCedulas))
	for _, member := range members {
		if selected[member.Cedula] {
			filtered = append(filtered, member)
		}
	}
	return filtered
}

func coverageDocumentDate(doc workspaceDocument, row coveragePlanilla) string {
	if date, ok := normalizeCoverageDate(doc.CoverageDate); ok {
		return date
	}
	date, _ := normalizeCoverageDate(row.CareUntil)
	return date
}

func coverageDocumentMetadataItems(doc workspaceDocument) []coverageDocumentMetadata {
	if len(doc.CoverageItems) > 0 {
		return doc.CoverageItems
	}
	if doc.PlanillaID > 0 || doc.CoverageCedula != "" || doc.CoverageDate != "" {
		return []coverageDocumentMetadata{{PlanillaID: doc.PlanillaID, Cedula: doc.CoverageCedula, Date: doc.CoverageDate}}
	}
	return nil
}

func coverageHasDate(row coveragePlanilla, date string) bool {
	date, ok := normalizeCoverageDate(date)
	if !ok {
		return false
	}
	for _, storedDate := range row.CoverageDates {
		if storedDate == date {
			return true
		}
	}
	return false
}

func missingCoverageMembersForDate(s *server, job stagedJob, row coveragePlanilla, queryDate string) []coverageMember {
	existing := make(map[string]bool)
	for _, doc := range s.coverageDocumentsForPlanilla(job, row.ID) {
		if coverageDocumentDate(doc, row) == queryDate && coverageCedulaPattern.MatchString(doc.CoverageCedula) {
			existing[doc.CoverageCedula] = true
		}
	}
	missing := make([]coverageMember, 0, 3)
	for _, member := range coverageMembersAt(row, queryDate) {
		if !existing[member.Cedula] {
			missing = append(missing, member)
		}
	}
	return missing
}

func (s *server) indexCoverageDocuments(job stagedJob) map[int64]*coverageDateIndex {
	indexed := make(map[int64]*coverageDateIndex)
	add := func(pdiID int64, relativePath, date string) {
		if pdiID <= 0 || !strings.HasPrefix(filepath.Base(relativePath), "C_COBERTURA") {
			return
		}
		entry := indexed[pdiID]
		if entry == nil {
			entry = &coverageDateIndex{Dates: make(map[string]bool)}
			indexed[pdiID] = entry
		}
		entry.HasPDF = true
		if normalized, ok := normalizeCoverageDate(date); ok {
			entry.Dates[normalized] = true
		} else {
			entry.HasUndated = true
		}
	}
	for _, doc := range job.ExternalPDFs {
		if !isCoveragePDFName(filepath.Base(doc.RelativePath)) {
			continue
		}
		items := coverageDocumentMetadataItems(doc)
		if len(items) == 0 {
			add(doc.PlanillaID, doc.RelativePath, doc.CoverageDate)
			continue
		}
		for _, item := range items {
			add(item.PlanillaID, doc.RelativePath, item.Date)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json"))
	if err == nil {
		var report classificationReport
		if json.Unmarshal(data, &report) == nil {
			for _, item := range report.Files {
				add(item.PlanillaID, item.Output, "")
			}
		}
	}
	return indexed
}

func (s *server) coverageDocumentsForPlanilla(job stagedJob, pdiID int64) []workspaceDocument {
	documents := make([]workspaceDocument, 0)
	for _, doc := range job.ExternalPDFs {
		if !isCoveragePDFName(filepath.Base(doc.RelativePath)) {
			continue
		}
		items := coverageDocumentMetadataItems(doc)
		if len(items) == 0 {
			if doc.PlanillaID == pdiID {
				documents = append(documents, doc)
			}
			continue
		}
		for _, item := range items {
			if item.PlanillaID != pdiID {
				continue
			}
			copy := doc
			copy.PlanillaID = item.PlanillaID
			copy.CoverageCedula = item.Cedula
			copy.CoverageDate = item.Date
			copy.CoverageItems = nil
			documents = append(documents, copy)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json"))
	if err == nil {
		var report classificationReport
		if json.Unmarshal(data, &report) == nil {
			for _, item := range report.Files {
				if item.PlanillaID == pdiID && strings.HasPrefix(filepath.Base(item.Output), "C_COBERTURA") {
					documents = append(documents, workspaceDocument{OriginalName: item.Code, RelativePath: item.Output, PlanillaID: pdiID})
				}
			}
		}
	}
	return documents
}
