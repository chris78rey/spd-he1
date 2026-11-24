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
	ManualCedula     bool                   `json:"cedula_manual_permitida,omitempty"`
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
	AttentionDates   []string               `json:"fechas_atencion,omitempty"`
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
	Cedula      string  `json:"cedula,omitempty"`
}

type individualCoverageRequest struct {
	Cedula    string `json:"cedula"`
	QueryDate string `json:"fecha_consulta"`
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
	// The attention days only guide the chosen-date query; without them the
	// listing still works and the screen falls back to PDI_FECHA_HASTA.
	if err := s.addCoverageAttentionDates(ctx, job, rows); err != nil {
		log.Printf("coberturas %s: no se pudieron leer los días de atención en SIS: %v", id, err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"job_id": job.ID, "tipo_servicio": job.Service, "mes": job.Month, "anio": job.Year,
		"planillas": rows,
	})
}

// coverageQueryMode says which date each selected planilla is consulted with.
type coverageQueryMode int

const (
	// coverageModeTramite uses PDI_FECHA_HASTA and skips planillas already covered.
	coverageModeTramite coverageQueryMode = iota
	// coverageModeChosenDate applies one date chosen by the user to every planilla.
	coverageModeChosenDate
	// coverageModeLastAttention uses each planilla's own last attention day in SIS.
	coverageModeLastAttention
)

func (s *server) generateCoverageSheets(w http.ResponseWriter, r *http.Request) {
	s.generateCoverageSheetsWithDate(w, r, coverageModeTramite)
}

func (s *server) generateCoverageSheetsAtDate(w http.ResponseWriter, r *http.Request) {
	s.generateCoverageSheetsWithDate(w, r, coverageModeChosenDate)
}

func (s *server) generateCoverageSheetsAtLastAttention(w http.ResponseWriter, r *http.Request) {
	s.generateCoverageSheetsWithDate(w, r, coverageModeLastAttention)
}

// lastAttentionDate is the day a planilla's coverage sheet is consulted with
// by default: its last non-annulled SIS account, else PDI_FECHA_HASTA.
func lastAttentionDate(row coveragePlanilla) string {
	if len(row.AttentionDates) > 0 {
		return row.AttentionDates[len(row.AttentionDates)-1]
	}
	date, _ := normalizeCoverageDate(row.CareUntil)
	return date
}

func (s *server) generateIndividualCoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var input individualCoverageRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Ingresa una cédula y una fecha de consulta válidas.")
		return
	}
	input.Cedula = strings.TrimSpace(input.Cedula)
	queryDate, ok := normalizeCoverageDate(input.QueryDate)
	if !coverageCedulaPattern.MatchString(input.Cedula) || !ok {
		writeError(w, http.StatusBadRequest, "Ingresa una cédula de 10 dígitos y una fecha válida.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
	defer cancel()
	outputDir, err := os.MkdirTemp("", "folio-cobertura-individual-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la descarga de cobertura.")
		return
	}
	defer os.RemoveAll(outputDir)
	outputName := fmt.Sprintf("COBERTURA_%s_%s", input.Cedula, strings.ReplaceAll(queryDate, "-", ""))
	cmd := exec.CommandContext(ctx, envOr("NODE_BIN", "node"), coverageScript, "--cedula", input.Cedula, "--fecha", queryDate, "--output_name", outputName, "--output_dir", outputDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		tag := coverageScriptFailureTag(output)
		log.Printf("cobertura individual: consulta MSP fallida (%T, %s)", err, tag)
		if tag == coverageTagSiteLimit {
			writeError(w, http.StatusTooManyRequests, "El portal MSP alcanzó su límite de consultas. Espera unos minutos y vuelve a intentar.")
			return
		}
		writeError(w, http.StatusBadGateway, "No se pudo obtener la cobertura del portal MSP. Revisa la cédula y vuelve a intentar.")
		return
	}
	pdfPath := filepath.Join(outputDir, outputName+".pdf")
	if err := validateCoveragePDF(pdfPath, input.Cedula, queryDate); err != nil {
		log.Printf("cobertura individual: el PDF devuelto no pasó la validación (%T)", err)
		writeError(w, http.StatusBadGateway, "El portal MSP no devolvió un PDF válido para la cédula y fecha ingresadas.")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, outputName))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, pdfPath)
}

func (s *server) generateCoverageSheetsWithDate(w http.ResponseWriter, r *http.Request, mode coverageQueryMode) {
	useChosenDate := mode == coverageModeChosenDate
	customDate := mode != coverageModeTramite
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	prefix := "/api/v1/coberturas/generar/"
	switch mode {
	case coverageModeChosenDate:
		prefix = "/api/v1/coberturas/generar-fecha/"
	case coverageModeLastAttention:
		prefix = "/api/v1/coberturas/generar-ultima-atencion/"
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
	input.Cedula = strings.TrimSpace(input.Cedula)
	if input.Cedula != "" {
		if !useChosenDate {
			writeError(w, http.StatusBadRequest, "La cédula manual solo se admite al consultar con otra fecha.")
			return
		}
		if !coverageCedulaPattern.MatchString(input.Cedula) {
			writeError(w, http.StatusBadRequest, "Ingresa una cédula manual de exactamente 10 dígitos.")
			return
		}
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
	if mode == coverageModeLastAttention {
		if err := s.addCoverageAttentionDates(ctx, job, rows); err != nil {
			log.Printf("coberturas %s: no se pudieron leer los días de atención en SIS: %v", job.ID, err)
			writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar en SIS los días de atención de las planillas.")
			return
		}
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
		if input.Cedula != "" && !row.ManualCedula {
			writeError(w, http.StatusBadRequest, "La cédula manual solo está disponible para trámites con PDFs importados desde Carpetas del equipo.")
			return
		}
		if mode == coverageModeTramite && strings.EqualFold(row.CoverageStatus, "S") && !coverageHasDate(row, row.CareUntil) {
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
	queryDates := make(map[int64]string, len(chosen))
	failures := make([]coverageFailureItem, 0)
	if job.CoverageFailures == nil {
		job.CoverageFailures = make(map[int64]coverageFailure)
	}
	for _, row := range chosen {
		rowQueryDate := row.CareUntil
		members := coverageMembersAt(row, rowQueryDate)
		if customDate {
			rowQueryDate = queryDate
			if mode == coverageModeLastAttention {
				rowQueryDate = lastAttentionDate(row)
				if rowQueryDate == "" {
					reason := "SIS no registra días de atención para esta planilla ni Oracle tiene su fecha hasta."
					job.CoverageFailures[row.ID] = coverageFailure{Reason: reason, Manual: false}
					failures = append(failures, coverageFailureItem{PlanillaID: row.ID, Tramite: row.Tramite, Reason: reason, Manual: false})
					continue
				}
			}
			queryDates[row.ID] = rowQueryDate
			expectedMembers := coverageMembersAt(row, rowQueryDate)
			if input.Cedula != "" {
				expectedMembers = []coverageMember{{Cedula: input.Cedula, Fecha: rowQueryDate}}
				members = missingCoverageMemberForDate(s, job, row, rowQueryDate, input.Cedula)
			} else {
				members = missingCoverageMembersForDate(s, job, row, rowQueryDate)
			}
			if len(members) == 0 && len(expectedMembers) > 0 {
				generatedIDs = append(generatedIDs, row.ID)
				delete(job.CoverageFailures, row.ID)
				continue
			}
			if len(members) == 0 {
				reason := "Oracle no registra una cédula válida de 10 dígitos para esta planilla."
				if row.ManualCedula {
					reason += " Como sus PDFs vienen de Carpetas del equipo, escribe la cédula que quieres consultar."
				}
				failure := coverageFailure{Reason: reason, QueryDate: rowQueryDate, CustomDate: customDate, Manual: false}
				job.CoverageFailures[row.ID] = failure
				failures = append(failures, coverageFailureItem{PlanillaID: row.ID, Tramite: row.Tramite, Reason: reason, Manual: false})
				continue
			}
		} else if coverageHasDate(row, row.CareUntil) {
			generatedIDs = append(generatedIDs, row.ID)
			delete(job.CoverageFailures, row.ID)
			continue
		}
		documents, targets, sources, failure := s.generateCoveragePlanilla(ctx, job, packageRoot, sourceRoot, row, rowQueryDate, members)
		if failure != nil {
			failure.QueryDate, failure.CustomDate = rowQueryDate, customDate
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
	previousDeleted := cloneBoolMap(job.DeletedPDFs)
	adoptRemovedPaths(&job, newDocuments)
	job.ExternalPDFs = append(job.ExternalPDFs, newDocuments...)
	if err := s.saveStagedJob(job); err != nil {
		job.ExternalPDFs = previousExternal
		job.DeletedPDFs = previousDeleted
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
	if mode == coverageModeLastAttention {
		response["fechas_consulta"] = queryDates
	}
	writeJSON(w, http.StatusOK, response)
}

// coverageScript descarga el PDF que genera la propia página del portal MSP. Por normativa la hoja de
// cobertura debe ser la del sitio del MSP: no se genera localmente ni se consulta su API directamente.
const (
	coverageScript       = "scripts/cobertura/download_site_pdf.cjs"
	coverageTagSiteLimit = "LIMITE_SITIO"
	coverageTagSiteMsg   = "MENSAJE_SITIO"
)

// coverageScriptFailureTag resume por qué falló el script sin copiar su salida (puede traer avisos del sitio).
func coverageScriptFailureTag(output []byte) string {
	text := string(output)
	switch {
	case strings.Contains(text, coverageTagSiteLimit):
		return coverageTagSiteLimit
	case strings.Contains(text, coverageTagSiteMsg):
		return coverageTagSiteMsg
	default:
		return "TECNICO"
	}
}

func (s *server) generateCoveragePlanilla(ctx context.Context, job stagedJob, packageRoot, sourceRoot string, row coveragePlanilla, queryDate string, members []coverageMember) ([]workspaceDocument, []string, []string, *coverageFailure) {
	if len(members) == 0 {
		return nil, nil, nil, &coverageFailure{Reason: "La planilla no tiene una cédula válida de 10 dígitos o una fecha de atención.", Manual: false}
	}
	patientFolder := strings.TrimSpace(row.PatientFolder)
	if patientFolder == "" {
		patientFolder = normalizePatientFolder(row.Patient)
	}
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
		cmd := exec.CommandContext(ctx, envOr("NODE_BIN", "node"), coverageScript, "--cedula", member.Cedula, "--fecha", member.Fecha, "--output_name", outputName, "--output_dir", outputDir)
		cmd.Dir = "."
		if output, err := cmd.CombinedOutput(); err != nil {
			tag := coverageScriptFailureTag(output)
			log.Printf("coberturas %s PDI_ID=%d: consulta al portal fallida (%T, %s)", job.ID, row.ID, err, tag)
			if tag == coverageTagSiteLimit {
				return nil, nil, nil, &coverageFailure{Reason: "El portal MSP alcanzó su límite de consultas. Espera unos minutos y reintenta, o descarga manualmente las hojas de esta planilla.", Manual: true}
			}
			return nil, nil, nil, &coverageFailure{Reason: "El portal MSP no respondió correctamente. Descarga manualmente las hojas de esta planilla.", Manual: true}
		}
		pdfPath := filepath.Join(outputDir, outputName+".pdf")
		if err := validateCoveragePDF(pdfPath, member.Cedula, member.Fecha); err != nil {
			log.Printf("coberturas %s PDI_ID=%d: PDF automático rechazado (%T)", job.ID, row.ID, err)
			return nil, nil, nil, &coverageFailure{Reason: "La descarga no produjo un PDF legible o no corresponde a la cédula y fecha consultadas. Descarga manualmente las hojas de esta planilla.", Manual: true}
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

func validateCoveragePDF(path, expectedCedula, expectedDate string) error {
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
	if !coverageSheetHasDate(text, expectedDate) {
		return errors.New("la fecha de cobertura no coincide")
	}
	return nil
}

var (
	coverageMonthsES = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
	coverageMonthsEN = []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}
)

// coverageSheetHasDate comprueba que la hoja del portal diga la fecha de cobertura pedida
// (YYYY-MM-DD). El portal la escribe como "01 octubre 2026" o "1 October 2026". Sin esta
// comprobación, una consulta que el portal respondió con otra fecha pasaba como válida.
func coverageSheetHasDate(text, expectedDate string) bool {
	date, err := time.Parse("2006-01-02", expectedDate)
	if err != nil {
		return false
	}
	normalized := strings.ToLower(strings.Join(strings.Fields(text), " "))
	month := int(date.Month()) - 1
	for _, name := range []string{coverageMonthsES[month], coverageMonthsEN[month]} {
		for _, day := range []string{fmt.Sprintf("%02d", date.Day()), strconv.Itoa(date.Day())} {
			pattern := regexp.MustCompile(`(^|[^0-9])` + day + ` ` + name + ` ` + strconv.Itoa(date.Year()) + `\b`)
			if pattern.MatchString(normalized) {
				return true
			}
		}
	}
	if name := coverageMonthsEN[month]; strings.Contains(normalized, name+" "+strconv.Itoa(date.Day())+", "+strconv.Itoa(date.Year())) {
		return true
	}
	return false
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
	lastAttention := r.URL.Query().Get("ultima_atencion") == "1"
	if lastAttention && queryDate != "" {
		writeError(w, http.StatusBadRequest, "Elige una fecha o la última atención, no ambas.")
		return
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
	if lastAttention {
		if err := s.addCoverageAttentionDates(r.Context(), job, rows); err != nil {
			log.Printf("coberturas %s: no se pudieron leer los días de atención en SIS: %v", job.ID, err)
			writeError(w, http.StatusServiceUnavailable, "No se pudieron consultar en SIS los días de atención de las planillas.")
			return
		}
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
		if !ok || (queryDate == "" && !lastAttention && !row.HasPDF) {
			_ = writer.Close()
			writeError(w, http.StatusConflict, "Una de las planillas no tiene todas sus hojas guardadas en el expediente.")
			return
		}
		rowDate := queryDate
		if lastAttention {
			rowDate = lastAttentionDate(row)
		}
		for _, doc := range s.coverageDocumentsForPlanilla(job, id) {
			if rowDate != "" && coverageDocumentDate(doc, row) != rowDate {
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
	} else if lastAttention {
		filename += "_ULTIMA_ATENCION"
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
	previousDeleted := cloneBoolMap(job.DeletedPDFs)
	adoptRemovedPaths(&job, documents)
	job.ExternalPDFs = append(job.ExternalPDFs, documents...)
	delete(job.CoverageFailures, row.ID)
	if err := s.saveStagedJob(job); err != nil {
		job.ExternalPDFs = previousExternal
		job.DeletedPDFs = previousDeleted
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
	if !s.hasClinicalSource(job) && !job.TeamFoldersPeriod {
		return stagedJob{}, errors.New("Este período ya no conserva el ZIP fuente.")
	}
	return job, nil
}

func (s *server) coverageRows(ctx context.Context, job stagedJob) ([]coveragePlanilla, error) {
	tramites := make(map[string]bool)
	planillaIDs := make(map[int64]bool)
	patientFolders := make(map[int64]string)
	manualCedulaPlanillas := make(map[int64]bool)
	if s.hasClinicalSource(job) {
		zipPath := filepath.Join(s.jobSourcesDir(job.ID), "lote.zip")
		archive, err := zip.OpenReader(zipPath)
		if err != nil {
			return nil, err
		}
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
	}
	for _, planilla := range job.OraclePlanillas {
		if planilla.PlanillaID > 0 {
			planillaIDs[planilla.PlanillaID] = true
			if planilla.PatientFolder != "" {
				patientFolders[planilla.PlanillaID] = planilla.PatientFolder
			}
		}
	}
	for _, document := range job.ExternalPDFs {
		if document.PlanillaID > 0 && !document.OraclePending {
			planillaIDs[document.PlanillaID] = true
			if isTeamImportedPDF(document) {
				manualCedulaPlanillas[document.PlanillaID] = true
			}
			parts := strings.Split(filepath.ToSlash(filepath.Clean(filepath.FromSlash(document.RelativePath))), "/")
			if len(parts) >= 3 && parts[0] == "4. EXPEDIENTES" && patientFolders[document.PlanillaID] == "" {
				patientFolders[document.PlanillaID] = parts[1]
			}
		}
	}
	if len(tramites) == 0 && len(planillaIDs) == 0 {
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
		if !tramites[row.Tramite] && !planillaIDs[row.ID] {
			continue
		}
		row.Patient, row.CareUntil = strings.TrimSpace(patient.String), strings.TrimSpace(date.String)
		row.PatientFolder = normalizePatientFolder(row.Patient)
		if folder := patientFolders[row.ID]; folder != "" {
			row.PatientFolder = folder
		}
		row.ManualCedula = manualCedulaPlanillas[row.ID]
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

func missingCoverageMemberForDate(s *server, job stagedJob, row coveragePlanilla, queryDate, cedula string) []coverageMember {
	for _, doc := range s.coverageDocumentsForPlanilla(job, row.ID) {
		if coverageDocumentDate(doc, row) == queryDate && strings.TrimSpace(doc.CoverageCedula) == cedula {
			return nil
		}
	}
	return []coverageMember{{Cedula: cedula, Fecha: queryDate}}
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
	current := currentRecordIndexByPath(job.ExternalPDFs)
	for index, doc := range job.ExternalPDFs {
		if !isCoveragePDFName(filepath.Base(doc.RelativePath)) || !isCurrentRecord(current, index, doc) || s.coverageDocumentRemoved(job, doc.RelativePath) {
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
				if s.coverageDocumentRemoved(job, item.Output) {
					continue
				}
				add(item.PlanillaID, item.Output, "")
			}
		}
	}
	return indexed
}

func (s *server) coverageDocumentsForPlanilla(job stagedJob, pdiID int64) []workspaceDocument {
	documents := make([]workspaceDocument, 0)
	current := currentRecordIndexByPath(job.ExternalPDFs)
	for index, doc := range job.ExternalPDFs {
		if !isCoveragePDFName(filepath.Base(doc.RelativePath)) || !isCurrentRecord(current, index, doc) || s.coverageDocumentRemoved(job, doc.RelativePath) {
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

// addCoverageAttentionDates fills each row with the distinct days of its
// non-annulled SIS accounts. The coverage sheet is consulted on one of these
// days; by default the last one.
func (s *server) addCoverageAttentionDates(ctx context.Context, job stagedJob, rows []coveragePlanilla) error {
	if len(rows) == 0 || s.serviceDB == nil {
		return nil
	}
	index := make(map[int64]int, len(rows))
	for i := range rows {
		index[rows[i].ID] = i
	}
	query := "SELECT TO_CHAR(p.PDI_ID), TO_CHAR(TRUNC(cta.fecha), 'YYYY-MM-DD') FROM " + oracleTableName(s.schema) + " p " +
		"JOIN SIS.CUENTAS cta ON cta.pla_numero_planilla = p.PDI_TRAMITE AND cta.prm_codigo = p.PDI_COD_ASEGURADORA AND cta.estado NOT LIKE 'AN%' " +
		"WHERE p.PDI_MES = :mes AND p.PDI_ANIO = :anio AND p.PDI_PLANILLADO = 'S' AND p.PDI_ASEGURADORA = 'MSP' " +
		"GROUP BY p.PDI_ID, TRUNC(cta.fecha)"
	result, err := s.serviceDB.QueryContext(ctx, query, sql.Named("mes", job.Month), sql.Named("anio", job.Year))
	if err != nil {
		return err
	}
	defer result.Close()
	dates := make(map[int64][]string)
	for result.Next() {
		var rawID, rawDate sql.NullString
		if err := result.Scan(&rawID, &rawDate); err != nil {
			return err
		}
		id, err := strconv.ParseInt(strings.TrimSpace(rawID.String), 10, 64)
		if err != nil {
			continue
		}
		if date, ok := normalizeCoverageDate(rawDate.String); ok {
			if _, wanted := index[id]; wanted {
				dates[id] = append(dates[id], date)
			}
		}
	}
	if err := result.Err(); err != nil {
		return err
	}
	for id, values := range dates {
		sort.Strings(values)
		rows[index[id]].AttentionDates = values
	}
	return nil
}

// coverageDocumentRemoved reports a coverage record whose PDF was removed from
// the patient folder. Such a sheet no longer counts as saved for its date, so
// the same date can be consulted again. A file re-created at that path still
// counts even when an older removal mark remains.
//
// A sheet merged into another file also stops counting by itself when the
// merged file records its sheets: the merged file's record already lists
// them, and an earlier record at its path must not add phantom dates.
func (s *server) coverageDocumentRemoved(job stagedJob, relative string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
	removed := job.DeletedPDFs[clean]
	if !removed {
		canonical := mergedCanonicalPath(job, clean)
		if canonical == "" {
			return false
		}
		removed = job.DeletedPDFs[canonical] || currentRecordHasCoverageItems(job, canonical)
		if !removed {
			return false
		}
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	path, err := safeWorkspacePath(packageRoot, clean)
	if err != nil {
		return true
	}
	_, err = os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}

// mergedCanonicalPath returns the file a merged duplicate was folded into.
func mergedCanonicalPath(job stagedJob, relative string) string {
	for canonical, duplicates := range job.MergedDuplicates {
		for _, duplicate := range duplicates {
			if filepath.ToSlash(filepath.Clean(filepath.FromSlash(duplicate))) == relative {
				return filepath.ToSlash(filepath.Clean(filepath.FromSlash(canonical)))
			}
		}
	}
	return ""
}

func currentRecordHasCoverageItems(job stagedJob, relative string) bool {
	index, ok := currentRecordIndexByPath(job.ExternalPDFs)[relative]
	return ok && len(job.ExternalPDFs[index].CoverageItems) > 0
}

// adoptRemovedPaths prepares paths that will hold a new PDF. When the path
// was marked as removed, its older records describe the removed file and are
// dropped, so they do not count again as saved sheets; the mark is cleared so
// a later reclassification does not remove the new file. Call it before
// appending the new records.
func adoptRemovedPaths(job *stagedJob, documents []workspaceDocument) {
	for _, doc := range documents {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))
		if !job.DeletedPDFs[path] {
			continue
		}
		job.ExternalPDFs = withoutWorkspaceDocumentsAt(job.ExternalPDFs, path)
		delete(job.DeletedPDFs, path)
	}
}

func withoutWorkspaceDocumentsAt(documents []workspaceDocument, relative string) []workspaceDocument {
	kept := make([]workspaceDocument, 0, len(documents))
	for _, doc := range documents {
		if filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath))) != relative {
			kept = append(kept, doc)
		}
	}
	return kept
}

// currentRecordIndexByPath maps each PDF path to its latest record. A path
// holds one file: a merge leaves a single record with every sheet in its
// coverage items, and a new file at the path is recorded after the old one.
// Older records at the same path describe a file that was removed or
// replaced, and must not count as saved sheets.
func currentRecordIndexByPath(documents []workspaceDocument) map[string]int {
	current := make(map[string]int, len(documents))
	for index, doc := range documents {
		current[filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))] = index
	}
	return current
}

func isCurrentRecord(current map[string]int, index int, doc workspaceDocument) bool {
	return current[filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))] == index
}
