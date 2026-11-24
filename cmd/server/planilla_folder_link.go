package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Buscar por planilla: a patient folder without a usable Oracle trámite can be
// linked to the PDI_TRAMITE the user already has. When that planilla is not in
// PLANILLA_DIGITAL yet, it can be created only from SIS data, with the same
// rules Recepción uses. The user never types patient data.

const (
	folderLinkLinked    = "asociada"
	folderLinkAvailable = "disponible"
	folderLinkCreatable = "crear"
	folderLinkRejected  = "rechazada"
)

var folderCopySuffix = regexp.MustCompile(`_[0-9]+$`)

type folderLinkPlanilla struct {
	PlanillaID  int64  `json:"pdi_id,omitempty"`
	Tramite     string `json:"pdi_tramite"`
	Patient     string `json:"paciente,omitempty"`
	Cedula      string `json:"cedula,omitempty"`
	InsurerCode string `json:"cod_aseguradora,omitempty"`
	Insurer     string `json:"aseguradora,omitempty"`
	ServiceCode string `json:"cod_servicio,omitempty"`
	Service     string `json:"servicio,omitempty"`
	Month       string `json:"mes,omitempty"`
	Year        string `json:"anio,omitempty"`
	CareFrom    string `json:"fecha_desde,omitempty"`
	CareUntil   string `json:"fecha_hasta,omitempty"`
	Planillado  string `json:"planillado,omitempty"`
	Coverage    string `json:"cobertura,omitempty"`
	Minor       string `json:"menor_edad,omitempty"`
}

type folderLinkPreview struct {
	Status   string              `json:"estado"`
	Reason   string              `json:"motivo,omitempty"`
	Warnings []string            `json:"avisos,omitempty"`
	Source   string              `json:"origen,omitempty"`
	Folder   string              `json:"carpeta"`
	Planilla *folderLinkPlanilla `json:"planilla,omitempty"`
}

// sisPlanillaSummary describes the SIS accounts of one planilla number, so a
// missing PDI row can be explained before anything is written.
type sisPlanillaSummary struct {
	Total        int
	Annulled     int
	MSPActive    int
	PeriodActive int
	Months       []string
	Services     []string
	Insurers     []string
}

func folderLinkMonthYear(month, year string) string {
	return strings.ToLower(historicalArchiveMonthLabel(month)) + " " + year
}

func serviceCodeLabel(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "CEX":
		return "Ambulatorio (consulta externa)"
	case "EMR", "OBE", "URG":
		return "Emergencia"
	case "HSP":
		return "Hospitalización"
	default:
		return code
	}
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

// folderLinkValidation applies the rules of a period to an existing PDI row:
// same insurer, month, year and service, and still planillado. It returns the
// reason a link is not allowed, or "" when the row belongs to this period.
func folderLinkValidation(job stagedJob, planilla folderLinkPlanilla) string {
	if !strings.EqualFold(strings.TrimSpace(planilla.Insurer), "MSP") {
		insurer := strings.TrimSpace(planilla.Insurer)
		if insurer == "" {
			insurer = "otra aseguradora"
		}
		return fmt.Sprintf("La planilla %s es de %s, no de MSP. Este período solo admite planillas MSP.", planilla.Tramite, insurer)
	}
	if planilla.Month != job.Month || planilla.Year != job.Year {
		return fmt.Sprintf("La planilla %s corresponde a %s. Ábrela en el período de ese mes.", planilla.Tramite, folderLinkMonthYear(planilla.Month, planilla.Year))
	}
	if !containsString(oracleServiceCodesForIngest(job.Service), planilla.ServiceCode) {
		return fmt.Sprintf("La planilla %s es de %s. Cada servicio tiene su propio período: ábrela en el período de ese servicio.", planilla.Tramite, serviceCodeLabel(planilla.ServiceCode))
	}
	if !strings.EqualFold(strings.TrimSpace(planilla.Planillado), "S") {
		return fmt.Sprintf("La planilla %s ya no figura como planillada en Oracle: sus cuentas no existen en SIS. Puede haberse anulado o replanillado con otro número.", planilla.Tramite)
	}
	return ""
}

// folderLinkNameWarning compares the folder with the Oracle patient. A
// mismatch does not block the link because folder names come from people.
func folderLinkNameWarning(folder, patient string) string {
	expected := normalizePatientFolder(patient)
	if expected == "" {
		return ""
	}
	current := folderCopySuffix.ReplaceAllString(folder, "")
	if current == expected {
		return ""
	}
	return fmt.Sprintf("El nombre de la carpeta (%s) no coincide con el paciente de la planilla (%s). Confirma que es la misma persona antes de asociarla.", folder, strings.Join(strings.Fields(patient), " "))
}

// folderLinkOwner returns the folder that already owns the planilla in this
// period, from PDFs, the classification report or a previous folder link.
func folderLinkOwner(s *server, job stagedJob, planillaID int64) (string, error) {
	folder, err := workspacePlanillaDocumentFolder(s, job, planillaID)
	if err != nil {
		return "", err
	}
	for _, item := range job.OraclePlanillas {
		if item.PlanillaID != planillaID || item.PatientFolder == "" {
			continue
		}
		if folder != "" && folder != item.PatientFolder {
			return "", errors.New("El trámite ya aparece asociado a varias carpetas; revisa el expediente antes de continuar.")
		}
		folder = item.PatientFolder
	}
	return folder, nil
}

// folderOtherPlanillas lists the trámites, other than planillaID, already
// linked to the folder. Pending PDFs are linked automatically only when the
// folder has no other trámite, so a second trámite never takes them silently.
func folderOtherPlanillas(job stagedJob, folder string, planillaID int64) map[int64]bool {
	others := make(map[int64]bool)
	inFolder := func(raw string) bool {
		parts := strings.Split(filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw))), "/")
		return len(parts) >= 3 && parts[0] == "4. EXPEDIENTES" && parts[1] == folder
	}
	for _, item := range job.OraclePlanillas {
		if item.PatientFolder == folder && item.PlanillaID > 0 && item.PlanillaID != planillaID {
			others[item.PlanillaID] = true
		}
	}
	for path, id := range job.DocumentPlanillas {
		if id > 0 && id != planillaID && inFolder(path) {
			others[id] = true
		}
	}
	for _, doc := range job.ExternalPDFs {
		if doc.PlanillaID > 0 && doc.PlanillaID != planillaID && !doc.OraclePending && inFolder(doc.RelativePath) {
			others[doc.PlanillaID] = true
		}
	}
	return others
}

// applyPlanillaFolderLink records that the folder owns the planilla and, when
// the folder has no other trámite, links its PDFs still pending association.
func applyPlanillaFolderLink(job *stagedJob, folder string, planilla workspaceOraclePlanilla) int {
	linked := 0
	if len(folderOtherPlanillas(*job, folder, planilla.PlanillaID)) == 0 {
		for index := range job.ExternalPDFs {
			doc := &job.ExternalPDFs[index]
			parts := strings.Split(filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath))), "/")
			if doc.OraclePending && len(parts) >= 3 && parts[0] == "4. EXPEDIENTES" && parts[1] == folder {
				doc.PlanillaID = planilla.PlanillaID
				doc.OraclePending = false
				linked++
			}
		}
	}
	planilla.PatientFolder = folder
	for index := range job.OraclePlanillas {
		if job.OraclePlanillas[index].PlanillaID == planilla.PlanillaID {
			job.OraclePlanillas[index].PatientFolder = folder
			return linked
		}
	}
	job.OraclePlanillas = append(job.OraclePlanillas, planilla)
	return linked
}

// classifySISPlanilla explains why SIS cannot produce a PDI row for this
// period, or returns "" when the read-only source query should be tried.
func classifySISPlanilla(job stagedJob, tramite string, summary sisPlanillaSummary) string {
	if summary.Total == 0 {
		return fmt.Sprintf("La planilla %s no existe en PLANILLA_DIGITAL ni en las cuentas de SIS. Revisa el número.", tramite)
	}
	if summary.Total == summary.Annulled {
		return fmt.Sprintf("Todas las cuentas de la planilla %s están anuladas en SIS.", tramite)
	}
	if summary.MSPActive == 0 {
		insurer := strings.Join(summary.Insurers, ", ")
		if insurer == "" {
			insurer = "otra aseguradora"
		}
		return fmt.Sprintf("La planilla %s es de %s, no de MSP.", tramite, insurer)
	}
	if summary.PeriodActive == 0 {
		months := make([]string, 0, len(summary.Months))
		for _, value := range summary.Months {
			if len(value) == 7 {
				months = append(months, folderLinkMonthYear(value[5:7], value[:4]))
			}
		}
		return fmt.Sprintf("Las cuentas MSP de la planilla %s son de %s, no de %s. Ábrela en el período que corresponde.", tramite, strings.Join(months, ", "), folderLinkMonthYear(job.Month, job.Year))
	}
	codes := oracleServiceCodesForIngest(job.Service)
	for _, service := range summary.Services {
		if containsString(codes, service) {
			return ""
		}
	}
	labels := make([]string, 0, len(summary.Services))
	for _, service := range summary.Services {
		label := serviceCodeLabel(service)
		if !containsString(labels, label) {
			labels = append(labels, label)
		}
	}
	return fmt.Sprintf("La planilla %s es de %s. Cada servicio tiene su propio período: ábrela en el período de ese servicio.", tramite, strings.Join(labels, ", "))
}

func (s *server) loadFolderLinkPlanilla(ctx context.Context, tramite string) (*folderLinkPlanilla, error) {
	query := "SELECT TO_CHAR(PDI_ID), TO_CHAR(PDI_TRAMITE), PDI_PACIENTE, PDI_CEDULA, PDI_COD_ASEGURADORA, PDI_ASEGURADORA, PDI_COD_SERVICIO, PDI_SERVICIO, PDI_MES, PDI_ANIO, " +
		"TO_CHAR(PDI_FECHA_DESDE, 'YYYY-MM-DD'), TO_CHAR(PDI_FECHA_HASTA, 'YYYY-MM-DD'), PDI_PLANILLADO, PDI_COBERTURA, PDI_MENOR_EDAD FROM " + oracleTableName(s.schema) + " WHERE PDI_TRAMITE = :tramite"
	rows, err := s.serviceDB.QueryContext(ctx, query, sql.Named("tramite", tramite))
	if err != nil {
		return nil, errors.New("No se pudo consultar la planilla en Oracle.")
	}
	defer rows.Close()
	var found *folderLinkPlanilla
	for rows.Next() {
		if found != nil {
			return nil, fmt.Errorf("Oracle contiene más de una fila para la planilla %s; se requiere resolver ese duplicado antes de continuar.", tramite)
		}
		var raw [15]sql.NullString
		pointers := make([]any, len(raw))
		for index := range raw {
			pointers[index] = &raw[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, errors.New("No se pudo leer la planilla encontrada en Oracle.")
		}
		id, err := parseOraclePlanillaID(raw[0])
		if err != nil {
			return nil, err
		}
		value := func(index int) string { return strings.TrimSpace(raw[index].String) }
		found = &folderLinkPlanilla{
			PlanillaID: id, Tramite: value(1), Patient: strings.Join(strings.Fields(value(2)), " "), Cedula: value(3),
			InsurerCode: value(4), Insurer: value(5), ServiceCode: value(6), Service: value(7), Month: value(8), Year: value(9),
			CareFrom: value(10), CareUntil: value(11), Planillado: strings.ToUpper(value(12)), Coverage: strings.ToUpper(value(13)), Minor: strings.ToUpper(value(14)),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("Falló la lectura de la planilla en Oracle.")
	}
	return found, nil
}

func (s *server) summarizeSISPlanilla(ctx context.Context, job stagedJob, tramite string) (sisPlanillaSummary, error) {
	summary := sisPlanillaSummary{}
	query := `SELECT cta.prm_codigo, NVL(promo.descripcion, cta.prm_codigo), cta.servicio,
		CASE WHEN cta.estado LIKE 'AN%' THEN 'S' ELSE 'N' END,
		TO_CHAR(cta.fecha, 'YYYY-MM'), COUNT(*)
		FROM SIS.CUENTAS cta LEFT JOIN SIS.PROMOCIONES promo ON promo.codigo = cta.prm_codigo
		WHERE cta.pla_numero_planilla = :tramite
		GROUP BY cta.prm_codigo, NVL(promo.descripcion, cta.prm_codigo), cta.servicio,
			CASE WHEN cta.estado LIKE 'AN%' THEN 'S' ELSE 'N' END, TO_CHAR(cta.fecha, 'YYYY-MM')`
	rows, err := s.serviceDB.QueryContext(ctx, query, sql.Named("tramite", tramite))
	if err != nil {
		return summary, errors.New("No se pudieron consultar las cuentas de la planilla en SIS.")
	}
	defer rows.Close()
	period := job.Year + "-" + job.Month
	for rows.Next() {
		var promotion, insurer, service, annulled, month sql.NullString
		var count int
		if err := rows.Scan(&promotion, &insurer, &service, &annulled, &month, &count); err != nil {
			return summary, errors.New("No se pudieron leer las cuentas de la planilla en SIS.")
		}
		summary.Total += count
		if annulled.String == "S" {
			summary.Annulled += count
			continue
		}
		isMSP := strings.TrimSpace(promotion.String) == "04" && strings.EqualFold(strings.TrimSpace(insurer.String), "MSP")
		if !isMSP {
			if label := strings.TrimSpace(insurer.String); label != "" && !containsString(summary.Insurers, label) {
				summary.Insurers = append(summary.Insurers, label)
			}
			continue
		}
		summary.MSPActive += count
		if value := strings.TrimSpace(month.String); value != "" && !containsString(summary.Months, value) {
			summary.Months = append(summary.Months, value)
		}
		if strings.TrimSpace(month.String) != period {
			continue
		}
		summary.PeriodActive += count
		if value := strings.TrimSpace(service.String); value != "" && !containsString(summary.Services, value) {
			summary.Services = append(summary.Services, value)
		}
	}
	if err := rows.Err(); err != nil {
		return summary, errors.New("Falló la lectura de las cuentas de la planilla en SIS.")
	}
	return summary, nil
}

// sourcePlanillaPreview runs the insert's own SELECT without writing, so the
// preview shows exactly the row that would be created.
func (s *server) sourcePlanillaPreview(ctx context.Context, job stagedJob, tramite string) (*folderLinkPlanilla, error) {
	serviceCodes := oracleServiceCodesForIngest(job.Service)
	query := "SELECT TO_CHAR(source.tramite), source.paciente, source.cedula, source.promocion, source.aseguradora, source.cod_servicio, source.servicio, " +
		"source.mes, source.anio, TO_CHAR(source.fecha_desde, 'YYYY-MM-DD'), TO_CHAR(source.fecha_hasta, 'YYYY-MM-DD') FROM (" +
		missingPlanillaSourceSQL(oracleTableName(s.schema), serviceCodes) + ") source"
	rows, err := s.serviceDB.QueryContext(ctx, query, planillaSourceArgs(job, serviceCodes, tramite)...)
	if err != nil {
		return nil, errors.New("No se pudo preparar la planilla desde SIS.")
	}
	defer rows.Close()
	var found *folderLinkPlanilla
	for rows.Next() {
		if found != nil {
			return nil, fmt.Errorf("SIS devuelve más de un paciente o permanencia para la planilla %s; no se puede crear automáticamente.", tramite)
		}
		var raw [11]sql.NullString
		pointers := make([]any, len(raw))
		for index := range raw {
			pointers[index] = &raw[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, errors.New("No se pudo leer la planilla desde SIS.")
		}
		value := func(index int) string { return strings.TrimSpace(raw[index].String) }
		found = &folderLinkPlanilla{
			Tramite: value(0), Patient: strings.Join(strings.Fields(value(1)), " "), Cedula: value(2), InsurerCode: value(3), Insurer: value(4),
			ServiceCode: value(5), Service: value(6), Month: value(7), Year: value(8), CareFrom: value(9), CareUntil: value(10), Planillado: "S", Coverage: "N",
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("Falló la lectura de la planilla desde SIS.")
	}
	return found, nil
}

// buildFolderLinkPreview decides, read-only, what linking the trámite to the
// folder would do.
func (s *server) buildFolderLinkPreview(ctx context.Context, job stagedJob, folder, tramite string) (folderLinkPreview, error) {
	preview := folderLinkPreview{Folder: folder}
	planilla, err := s.loadFolderLinkPlanilla(ctx, tramite)
	if err != nil {
		return preview, err
	}
	if planilla != nil {
		preview.Source = "planilla_digital"
		preview.Planilla = planilla
		if reason := folderLinkValidation(job, *planilla); reason != "" {
			preview.Status, preview.Reason = folderLinkRejected, reason
			return preview, nil
		}
		owner, err := folderLinkOwner(s, job, planilla.PlanillaID)
		if err != nil {
			return preview, err
		}
		if owner != "" && owner != folder {
			preview.Status = folderLinkRejected
			preview.Reason = fmt.Sprintf("La planilla %s ya está asociada a la carpeta %s de este período. Si es el mismo paciente, sus documentos deben quedar en una sola carpeta: revisa esa carpeta en lugar de crear otra asociación.", tramite, owner)
			return preview, nil
		}
		if warning := folderLinkNameWarning(folder, planilla.Patient); warning != "" {
			preview.Warnings = append(preview.Warnings, warning)
		}
		if owner == folder {
			preview.Status = folderLinkLinked
			return preview, nil
		}
		preview.Status = folderLinkAvailable
		return preview, nil
	}
	preview.Source = "sis"
	summary, err := s.summarizeSISPlanilla(ctx, job, tramite)
	if err != nil {
		return preview, err
	}
	if reason := classifySISPlanilla(job, tramite, summary); reason != "" {
		preview.Status, preview.Reason = folderLinkRejected, reason
		preview.Planilla = &folderLinkPlanilla{Tramite: tramite}
		return preview, nil
	}
	source, err := s.sourcePlanillaPreview(ctx, job, tramite)
	if err != nil {
		preview.Status, preview.Reason = folderLinkRejected, err.Error()
		preview.Planilla = &folderLinkPlanilla{Tramite: tramite}
		return preview, nil
	}
	if source == nil {
		preview.Status = folderLinkRejected
		preview.Reason = fmt.Sprintf("SIS tiene datos incompletos o inconsistentes para la planilla %s (por ejemplo, varias historias clínicas o permanencias en la misma planilla, o un paciente o servicio sin registrar). No se puede crear automáticamente: debe corregirse en SIS.", tramite)
		preview.Planilla = &folderLinkPlanilla{Tramite: tramite}
		return preview, nil
	}
	preview.Planilla = source
	preview.Status = folderLinkCreatable
	if warning := folderLinkNameWarning(folder, source.Patient); warning != "" {
		preview.Warnings = append(preview.Warnings, warning)
	}
	return preview, nil
}

func (s *server) createFolderLinkPlanilla(ctx context.Context, job stagedJob, tramite string) error {
	serviceCodes := oracleServiceCodesForIngest(job.Service)
	tx, err := s.serviceDB.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("No se pudo iniciar la operación en Oracle.")
	}
	defer tx.Rollback()
	outcome, err := insertPlanillaFromSource(ctx, tx, oracleTableName(s.schema), job, serviceCodes, tramite)
	if err != nil {
		log.Printf("expediente %s: no se creó la planilla %s desde SIS: %v", job.ID, tramite, err)
		return errors.New("Oracle no pudo crear la planilla desde SIS; no se hicieron cambios.")
	}
	if outcome == planillaNotInSource {
		return errors.New("SIS ya no permite crear esta planilla para el período; vuelve a buscarla.")
	}
	if err := tx.Commit(); err != nil {
		return errors.New("Oracle no confirmó la creación de la planilla; no se hicieron cambios.")
	}
	if outcome == planillaInserted {
		log.Printf("expediente %s: planilla %s creada en PLANILLA_DIGITAL desde SIS", job.ID, tramite)
	}
	return nil
}

func (s *server) planillaFolderLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/planilla-carpeta/")
	if !validJobID(jobID) {
		writeError(w, http.StatusBadRequest, "El identificador del período no es válido.")
		return
	}
	var input struct {
		Folder  string `json:"carpeta"`
		Tramite string `json:"pdi_tramite"`
		Create  bool   `json:"crear"`
	}
	if r.Method == http.MethodGet {
		input.Folder, input.Tramite = r.URL.Query().Get("carpeta"), r.URL.Query().Get("pdi_tramite")
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			writeError(w, http.StatusBadRequest, "La solicitud no es válida.")
			return
		}
	}
	input.Tramite = strings.TrimSpace(input.Tramite)
	if !tramitePattern.MatchString(input.Tramite) || len(input.Tramite) > 12 {
		writeError(w, http.StatusBadRequest, "Escribe el número de planilla (PDI_TRAMITE) solo con dígitos.")
		return
	}
	if input.Folder == "" || input.Folder != filepath.Base(input.Folder) || input.Folder == "." || input.Folder == ".." || strings.ContainsAny(input.Folder, `/\`) {
		writeError(w, http.StatusBadRequest, "Selecciona una carpeta de paciente válida.")
		return
	}
	if s.serviceDB == nil {
		writeError(w, http.StatusServiceUnavailable, "Oracle no está disponible.")
		return
	}

	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el período guardado.")
		return
	}
	if (job.Status != "PROCESSED" && job.Status != "INCOMPLETE") || job.IsObjections {
		writeError(w, http.StatusConflict, "Selecciona un período de recepción preparado.")
		return
	}
	if len(oracleServiceCodesForIngest(job.Service)) == 0 {
		writeError(w, http.StatusConflict, "No se reconoce el servicio del período para consultar Oracle.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	folderPath, err := safeWorkspacePath(packageRoot, filepath.Join("4. EXPEDIENTES", input.Folder))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Selecciona una carpeta de paciente válida.")
		return
	}
	if info, err := os.Stat(folderPath); err != nil || !info.IsDir() {
		writeError(w, http.StatusNotFound, "No se encontró la carpeta del paciente en este período.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	preview, err := s.buildFolderLinkPreview(ctx, job, input.Folder, input.Tramite)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, preview)
		return
	}

	created := false
	switch preview.Status {
	case folderLinkRejected:
		writeError(w, http.StatusConflict, preview.Reason)
		return
	case folderLinkLinked:
		writeJSON(w, http.StatusOK, map[string]any{"vista": preview, "message": fmt.Sprintf("La planilla %s ya estaba asociada a esta carpeta.", input.Tramite)})
		return
	case folderLinkCreatable:
		if !input.Create {
			writeError(w, http.StatusConflict, "La planilla no existe en PLANILLA_DIGITAL; confirma que deseas crearla desde SIS.")
			return
		}
		if err := s.createFolderLinkPlanilla(ctx, job, input.Tramite); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		created = true
		preview, err = s.buildFolderLinkPreview(ctx, job, input.Folder, input.Tramite)
		if err != nil {
			writeError(w, http.StatusBadGateway, "La planilla se creó en Oracle, pero no se pudo volver a leer: "+err.Error())
			return
		}
		if preview.Status == folderLinkLinked {
			writeJSON(w, http.StatusOK, map[string]any{"vista": preview, "message": fmt.Sprintf("La planilla %s ya estaba asociada a esta carpeta.", input.Tramite)})
			return
		}
		if preview.Status != folderLinkAvailable {
			writeError(w, http.StatusConflict, "La planilla se creó en Oracle, pero no se pudo asociar: "+preview.Reason)
			return
		}
	}

	planilla := preview.Planilla
	previousJob, err := cloneStagedJob(job)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la asociación.")
		return
	}
	linkedPDFs := applyPlanillaFolderLink(&job, input.Folder, workspaceOraclePlanilla{
		PlanillaID: planilla.PlanillaID, Tramite: planilla.Tramite, Patient: planilla.Patient, Service: planilla.Service,
	})
	if err := s.saveStagedJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la asociación.")
		return
	}
	if err := s.syncOracleWorkspace(ctx, &job, packageRoot); err != nil {
		if restoreErr := s.saveStagedJob(previousJob); restoreErr != nil {
			log.Printf("expediente %s: no se pudo restaurar la carpeta %s tras fallar Oracle: %v", job.ID, input.Folder, restoreErr)
			writeError(w, http.StatusInternalServerError, "Falló Oracle y no se pudo restaurar el estado anterior; revisa el expediente antes de continuar.")
			return
		}
		log.Printf("expediente %s: falló la asociación Oracle de la planilla %s a %s: %v", job.ID, input.Tramite, input.Folder, err)
		writeError(w, http.StatusBadGateway, "No se confirmó la asociación con Oracle; la carpeta quedó como estaba.")
		return
	}
	log.Printf("expediente %s: planilla %s (PDI_ID=%d) asociada a la carpeta %s; %d PDF(s) pendientes vinculados", job.ID, planilla.Tramite, planilla.PlanillaID, input.Folder, linkedPDFs)
	message := fmt.Sprintf("La planilla %s quedó asociada a esta carpeta.", planilla.Tramite)
	if created {
		message = fmt.Sprintf("La planilla %s se creó desde SIS y quedó asociada a esta carpeta.", planilla.Tramite)
	}
	if linkedPDFs > 0 {
		message += fmt.Sprintf(" También se vincularon %d PDF(s) que estaban pendientes de planilla.", linkedPDFs)
	}
	preview.Status = folderLinkLinked
	writeJSON(w, http.StatusOK, map[string]any{"vista": preview, "message": message, "pdfs_vinculados": linkedPDFs})
}
