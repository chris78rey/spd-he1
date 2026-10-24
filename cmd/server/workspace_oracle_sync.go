package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type workspaceOraclePlanilla struct {
	PlanillaID    int64  `json:"pdi_id"`
	Tramite       string `json:"pdi_tramite"`
	Patient       string `json:"paciente"`
	PatientFolder string `json:"carpeta_paciente,omitempty"`
	Service       string `json:"pdi_servicio"`
}

type workspacePlanillaOption struct {
	PatientFolder string `json:"patient_folder"`
	PlanillaID    int64  `json:"pdi_id"`
	Tramite       string `json:"pdi_tramite,omitempty"`
}

func workspacePlanillaOptions(job stagedJob, documents []workspacePDF) []workspacePlanillaOption {
	byKey := make(map[string]workspacePlanillaOption)
	add := func(patient string, id int64, tramite string) {
		if patient == "" || id <= 0 {
			return
		}
		key := fmt.Sprintf("%s|%d", patient, id)
		option := byKey[key]
		option.PatientFolder = patient
		option.PlanillaID = id
		if tramite != "" {
			option.Tramite = tramite
		}
		byKey[key] = option
	}
	for _, document := range documents {
		if document.PlanillaID <= 0 {
			continue
		}
		parts := strings.Split(filepath.ToSlash(document.Path), "/")
		if len(parts) >= 3 && parts[0] == "4. EXPEDIENTES" {
			add(parts[1], document.PlanillaID, "")
		}
	}
	for _, planilla := range job.OraclePlanillas {
		add(planilla.PatientFolder, planilla.PlanillaID, planilla.Tramite)
	}
	for _, row := range job.ObjectionRows {
		add(row.PatientFolder, row.PlanillaID, row.Tramite)
	}
	options := make([]workspacePlanillaOption, 0, len(byKey))
	for _, option := range byKey {
		options = append(options, option)
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].PatientFolder != options[j].PatientFolder {
			return options[i].PatientFolder < options[j].PatientFolder
		}
		return options[i].PlanillaID < options[j].PlanillaID
	})
	return options
}

type workspaceOracleSyncPreview struct {
	PreviewID      string                    `json:"preview_id"`
	BaselineReady  bool                      `json:"baseline_ready"`
	Total          int                       `json:"total"`
	Importable     int                       `json:"importables"`
	MissingPatient int                       `json:"sin_paciente"`
	Items          []workspaceOraclePlanilla `json:"items"`
}

type workspaceOracleSyncRequest struct {
	Apply     bool   `json:"apply"`
	PreviewID string `json:"preview_id"`
}

func (s *server) workspaceOraclePlanillas(ctx context.Context, job stagedJob) ([]workspaceOraclePlanilla, error) {
	serviceCodes := oracleServiceCodesForIngest(job.Service)
	if len(serviceCodes) == 0 {
		return nil, errors.New("No se reconoce el servicio del expediente para consultar Oracle.")
	}
	placeholders := make([]string, len(serviceCodes))
	args := []any{sql.Named("mes", job.Month), sql.Named("anio", job.Year)}
	for index, code := range serviceCodes {
		name := fmt.Sprintf("servicio_%d", index)
		placeholders[index] = ":" + name
		args = append(args, sql.Named(name, code))
	}
	query := "SELECT TO_CHAR(PDI_ID), TO_CHAR(PDI_TRAMITE), PDI_PACIENTE, PDI_SERVICIO FROM " + oracleTableName(s.schema) +
		" WHERE PDI_MES = :mes AND PDI_ANIO = :anio AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP' AND PDI_COD_SERVICIO IN (" + strings.Join(placeholders, ",") + ") ORDER BY PDI_ID"
	rows, err := s.serviceDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.New("No se pudieron consultar en Oracle los trámites del servicio y período.")
	}
	defer rows.Close()
	result := make([]workspaceOraclePlanilla, 0)
	seen := make(map[int64]bool)
	for rows.Next() {
		var rawID, rawTramite, rawPatient, rawService sql.NullString
		if err := rows.Scan(&rawID, &rawTramite, &rawPatient, &rawService); err != nil {
			return nil, errors.New("No se pudieron leer los trámites del período desde Oracle.")
		}
		id, err := parseOraclePlanillaID(rawID)
		if err != nil {
			return nil, errors.New("Oracle devolvió un PDI_ID inválido para el período.")
		}
		if seen[id] {
			return nil, fmt.Errorf("Oracle devolvió repetido el PDI_ID %d.", id)
		}
		seen[id] = true
		tramite := strings.TrimSpace(rawTramite.String)
		if !rawTramite.Valid || !tramitePattern.MatchString(tramite) {
			continue
		}
		patient := strings.TrimSpace(rawPatient.String)
		result = append(result, workspaceOraclePlanilla{
			PlanillaID:    id,
			Tramite:       tramite,
			Patient:       patient,
			PatientFolder: normalizePatientFolder(patient),
			Service:       strings.TrimSpace(rawService.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("Falló la lectura de trámites de Oracle.")
	}
	return result, nil
}

func workspaceKnownOraclePlanillas(s *server, job stagedJob) (map[int64]bool, error) {
	known := make(map[int64]bool)
	for _, planillaID := range job.OracleBaselineIDs {
		if planillaID > 0 {
			known[planillaID] = true
		}
	}
	for _, planilla := range job.OraclePlanillas {
		if planilla.PlanillaID > 0 {
			known[planilla.PlanillaID] = true
		}
	}
	for _, planillaID := range job.DocumentPlanillas {
		if planillaID > 0 {
			known[planillaID] = true
		}
	}
	for _, document := range job.ExternalPDFs {
		if document.PlanillaID > 0 {
			known[document.PlanillaID] = true
		}
	}
	reportPath := filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json")
	data, err := os.ReadFile(reportPath)
	if errors.Is(err, os.ErrNotExist) {
		return known, nil
	}
	if err != nil {
		return nil, errors.New("No se pudo comprobar qué trámites ya están en el expediente.")
	}
	var report classificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, errors.New("El reporte de clasificación no permite comprobar los trámites ya incorporados.")
	}
	for _, file := range report.Files {
		if file.PlanillaID > 0 {
			known[file.PlanillaID] = true
		}
	}
	return known, nil
}

func buildWorkspaceOracleSyncPreview(ctx context.Context, s *server, job stagedJob) (workspaceOracleSyncPreview, error) {
	preview := workspaceOracleSyncPreview{BaselineReady: job.OracleBaselineIDs != nil, Items: []workspaceOraclePlanilla{}}
	if s.serviceDB == nil {
		return preview, errors.New("Oracle no está disponible.")
	}
	known, err := workspaceKnownOraclePlanillas(s, job)
	if err != nil {
		return preview, err
	}
	all, err := s.workspaceOraclePlanillas(ctx, job)
	if err != nil {
		return preview, err
	}
	for _, item := range all {
		if err := ctx.Err(); err != nil {
			return preview, err
		}
		if known[item.PlanillaID] {
			continue
		}
		preview.Items = append(preview.Items, item)
		if item.PatientFolder == "" {
			preview.MissingPatient++
		} else {
			preview.Importable++
		}
	}
	preview.Total = len(preview.Items)
	fingerprintInput, err := json.Marshal(struct {
		JobID   string                    `json:"job_id"`
		Service string                    `json:"service"`
		Month   string                    `json:"month"`
		Year    string                    `json:"year"`
		Items   []workspaceOraclePlanilla `json:"items"`
	}{job.ID, job.Service, job.Month, job.Year, preview.Items})
	if err != nil {
		return preview, errors.New("No se pudo preparar la vista previa de Oracle.")
	}
	fingerprint := sha256.Sum256(fingerprintInput)
	preview.PreviewID = hex.EncodeToString(fingerprint[:])
	return preview, nil
}

func (s *server) syncWorkspaceOracleRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/oracle-nuevos/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input workspaceOracleSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer la solicitud de sincronización.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el expediente guardado.")
		return
	}
	if (job.Status != "PROCESSED" && job.Status != "INCOMPLETE") || job.IsObjections {
		writeError(w, http.StatusConflict, "Abre un período de recepción ya preparado para buscar trámites nuevos.")
		return
	}
	preview, err := buildWorkspaceOracleSyncPreview(r.Context(), s, job)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if !input.Apply {
		writeJSON(w, http.StatusOK, preview)
		return
	}
	if input.PreviewID == "" || input.PreviewID != preview.PreviewID {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Oracle o el expediente cambiaron desde la vista previa. Revisa la lista actualizada antes de incorporar.", "preview": preview})
		return
	}
	if preview.Importable == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"applied": true, "imported": 0, "preview": preview})
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	expedientesRoot := filepath.Join(packageRoot, "4. EXPEDIENTES")
	if err := os.MkdirAll(expedientesRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carpeta de expedientes.")
		return
	}
	createdDirs := make([]string, 0)
	existing := make(map[int64]bool, len(job.OraclePlanillas))
	for _, item := range job.OraclePlanillas {
		existing[item.PlanillaID] = true
	}
	for _, item := range preview.Items {
		if item.PatientFolder == "" {
			continue
		}
		patientPath, pathErr := safeWorkspacePath(packageRoot, filepath.ToSlash(filepath.Join("4. EXPEDIENTES", item.PatientFolder)))
		if pathErr != nil {
			for _, created := range createdDirs {
				_ = os.Remove(created)
			}
			writeError(w, http.StatusBadRequest, "Oracle devolvió un nombre de paciente no válido.")
			return
		}
		if info, statErr := os.Stat(patientPath); errors.Is(statErr, os.ErrNotExist) {
			if err := os.Mkdir(patientPath, 0700); err != nil {
				for _, created := range createdDirs {
					_ = os.Remove(created)
				}
				writeError(w, http.StatusInternalServerError, "No se pudo crear una carpeta para un paciente nuevo.")
				return
			}
			createdDirs = append(createdDirs, patientPath)
		} else if statErr != nil {
			for _, created := range createdDirs {
				_ = os.Remove(created)
			}
			writeError(w, http.StatusInternalServerError, "No se pudo comprobar una carpeta de paciente.")
			return
		} else if !info.IsDir() {
			for _, created := range createdDirs {
				_ = os.Remove(created)
			}
			writeError(w, http.StatusConflict, "La ruta de un paciente nuevo ya existe y no es una carpeta.")
			return
		}
		if !existing[item.PlanillaID] {
			job.OraclePlanillas = append(job.OraclePlanillas, item)
			existing[item.PlanillaID] = true
		}
	}
	if err := s.saveStagedJob(job); err != nil {
		for _, created := range createdDirs {
			_ = os.Remove(created)
		}
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la lista de trámites incorporados.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": true, "imported": preview.Importable, "items": preview.Items, "message": fmt.Sprintf("Se incorporaron %d trámite(s) nuevos al período.", preview.Importable)})
}
