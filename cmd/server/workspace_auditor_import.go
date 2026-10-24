package main

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type auditorImportFile struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

type auditorImportFolder struct {
	Name  string              `json:"name"`
	Files []auditorImportFile `json:"files"`
}

type auditorImportPlanilla struct {
	PlanillaID int64  `json:"pdi_id"`
	Tramite    string `json:"pdi_tramite"`
	Service    string `json:"pdi_servicio"`
}

type auditorReceipt struct {
	ID      string `json:"id"`
	Patient string `json:"paciente"`
	Service string `json:"servicio"`
	Month   string `json:"mes"`
	Year    string `json:"anio"`
	Status  string `json:"estado"`
}

type auditorImportStage struct {
	JobID     string                  `json:"job_id"`
	Username  string                  `json:"username"`
	CreatedAt time.Time               `json:"created_at"`
	Folders   []auditorImportFolder   `json:"folders"`
	Planillas []auditorImportPlanilla `json:"planillas"`
	StripRoot bool                    `json:"strip_root"`
}

type auditorImportPreviewResponse struct {
	PreviewID string                  `json:"preview_id"`
	Folders   []auditorImportFolder   `json:"folders"`
	Planillas []auditorImportPlanilla `json:"planillas"`
	TotalPDFs int                     `json:"total_pdfs"`
}

type auditorImportApplyRequest struct {
	PreviewID    string           `json:"preview_id"`
	FileMappings map[string]int64 `json:"file_mappings"`
}

func isTeamImportedPDF(doc workspaceDocument) bool {
	if doc.TeamImport {
		return true
	}
	// ZIP imports made before the origin flag was added used 12 random bytes;
	// manual uploads use 8. Keep older team imports distinguishable on upgrade.
	if doc.ReceiptID == "" {
		return false
	}
	id := strings.TrimSuffix(strings.TrimSpace(doc.StoredName), filepath.Ext(doc.StoredName))
	if id == "" {
		id = strings.TrimSpace(doc.ID)
	}
	decoded, err := hex.DecodeString(id)
	return err == nil && len(decoded) == 12
}

func (s *server) manageTeamPatientFolders(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/carpetas-equipo/pacientes/")
	if !validJobID(jobID) {
		writeError(w, http.StatusBadRequest, "El identificador del período no es válido.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(jobID)
	if err != nil || job.IsObjections || (job.Status != "PROCESSED" && job.Status != "INCOMPLETE") {
		writeError(w, http.StatusNotFound, "No se encontró un período disponible para administrar sus carpetas.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	switch r.Method {
	case http.MethodGet:
		documents, patients, err := listWorkspacePDFs(packageRoot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudieron leer las carpetas del período.")
			return
		}
		teamPaths := make(map[string]bool)
		latestByPath := make(map[string]workspaceDocument)
		for _, doc := range job.ExternalPDFs {
			path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))
			latestByPath[path] = doc
		}
		for path, doc := range latestByPath {
			if isTeamImportedPDF(doc) {
				teamPaths[path] = true
			}
		}
		teamFolders := make(map[string]bool, len(job.AuditorReceipts))
		for _, receipt := range job.AuditorReceipts {
			teamFolders[receipt.Patient] = true
		}
		folders := make([]map[string]any, 0, len(patients))
		for _, patient := range patients {
			count, teamCount := 0, 0
			prefix := "4. EXPEDIENTES/" + patient + "/"
			for _, document := range documents {
				if strings.HasPrefix(document.Path, prefix) {
					count++
					if teamPaths[document.Path] {
						teamCount++
					}
				}
			}
			if count == 0 {
				continue
			}
			origin := "Carga normal"
			hasTeamFolder := teamFolders[patient] || teamCount > 0
			if hasTeamFolder && teamCount < count {
				origin = "Mixta · equipo y carga normal"
			} else if hasTeamFolder {
				origin = "Carpetas del equipo"
			}
			folders = append(folders, map[string]any{"nombre": patient, "pdfs": count, "origen": origin})
		}
		writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "carpetas": folders})
	case http.MethodDelete:
		r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
		var input struct {
			Folders      []string `json:"carpetas"`
			Confirmation string   `json:"confirmacion"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil || len(input.Folders) == 0 || len(input.Folders) > 100 {
			writeError(w, http.StatusBadRequest, "Selecciona las carpetas de pacientes que quieres quitar.")
			return
		}
		confirmationWord := "CARPETAS"
		if len(input.Folders) == 1 {
			confirmationWord = "CARPETA"
		}
		if strings.TrimSpace(input.Confirmation) != fmt.Sprintf("QUITAR %d %s", len(input.Folders), confirmationWord) {
			writeError(w, http.StatusBadRequest, "Completa la tercera confirmación escribiendo el texto indicado.")
			return
		}
		selected := make(map[string]bool, len(input.Folders))
		for _, name := range input.Folders {
			name = strings.TrimSpace(name)
			if name == "" || selected[name] {
				writeError(w, http.StatusBadRequest, "La selección de carpetas no es válida.")
				return
			}
			selected[name] = true
		}
		documents, patients, err := listWorkspacePDFs(packageRoot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudieron comprobar las carpetas seleccionadas.")
			return
		}
		available := make(map[string]bool, len(patients))
		for _, patient := range patients {
			available[patient] = true
		}
		moving := make([]struct{ source, backup, patient string }, 0)
		keptEmptyOracleFolders := make([]string, 0)
		for patient := range selected {
			if !available[patient] {
				writeError(w, http.StatusConflict, "Una de las carpetas seleccionadas ya no existe en este período.")
				return
			}
			prefix := "4. EXPEDIENTES/" + patient + "/"
			for _, document := range documents {
				if !strings.HasPrefix(document.Path, prefix) {
					continue
				}
				parts := strings.Split(document.Path, "/")
				if len(parts) != 3 || parts[0] != "4. EXPEDIENTES" || parts[1] != patient {
					writeError(w, http.StatusConflict, "No se pudo validar una ruta de PDF de la carpeta seleccionada.")
					return
				}
				full, pathErr := safeWorkspacePath(packageRoot, document.Path)
				if pathErr != nil {
					writeError(w, http.StatusBadRequest, "La ruta de un PDF no es válida.")
					return
				}
				moving = append(moving, struct{ source, backup, patient string }{source: full, patient: patient})
			}
		}
		if len(moving) == 0 {
			writeError(w, http.StatusConflict, "Las carpetas seleccionadas ya no contienen PDFs.")
			return
		}
		previousJob, err := cloneStagedJob(job)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo preparar el retiro de las carpetas.")
			return
		}
		backupRoot, err := os.MkdirTemp(s.jobRoot(job.ID), ".quitar-carpetas-")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo preparar la operación reversible.")
			return
		}
		restore := func() {
			for _, item := range moving {
				if item.backup == "" {
					continue
				}
				_ = os.MkdirAll(filepath.Dir(item.source), 0700)
				_ = os.Rename(item.backup, item.source)
			}
			_ = os.RemoveAll(backupRoot)
		}
		for index := range moving {
			moving[index].backup = filepath.Join(backupRoot, fmt.Sprintf("%04d.pdf", index))
			if err := os.Rename(moving[index].source, moving[index].backup); err != nil {
				restore()
				writeError(w, http.StatusInternalServerError, "No se pudieron apartar los PDFs; se restauraron los archivos movidos.")
				return
			}
			relative, _ := filepath.Rel(packageRoot, moving[index].source)
			if job.DeletedPDFs == nil {
				job.DeletedPDFs = make(map[string]bool)
			}
			job.DeletedPDFs[filepath.ToSlash(relative)] = true
		}
		if err := s.saveStagedJob(job); err != nil {
			restore()
			writeError(w, http.StatusInternalServerError, "No se pudo registrar el retiro; las carpetas y PDFs se restauraron.")
			return
		}
		if err := s.syncOracleWorkspace(r.Context(), &job, packageRoot); err != nil {
			restore()
			_ = s.saveStagedJob(previousJob)
			log.Printf("expediente %s: no se pudieron quitar carpetas del equipo: %v", job.ID, err)
			writeError(w, http.StatusBadGateway, "Oracle no confirmó el cambio; las carpetas y PDFs se restauraron.")
			return
		}
		if err := os.RemoveAll(backupRoot); err != nil {
			log.Printf("expediente %s: PDFs quitados; quedó respaldo recuperable %s: %v", job.ID, backupRoot, err)
		}
		for patient := range selected {
			hasPlanilla := false
			for _, planilla := range job.OraclePlanillas {
				if planilla.PatientFolder == patient {
					hasPlanilla = true
					break
				}
			}
			if !hasPlanilla {
				for _, doc := range job.ExternalPDFs {
					if doc.PlanillaID > 0 && strings.HasPrefix(filepath.ToSlash(doc.RelativePath), "4. EXPEDIENTES/"+patient+"/") {
						hasPlanilla = true
						break
					}
				}
			}
			if !hasPlanilla {
				for path, planillaID := range job.DocumentPlanillas {
					if planillaID > 0 && strings.HasPrefix(filepath.ToSlash(path), "4. EXPEDIENTES/"+patient+"/") {
						hasPlanilla = true
						break
					}
				}
			}
			if !hasPlanilla {
				if reportBytes, readErr := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json")); readErr == nil {
					var report classificationReport
					if json.Unmarshal(reportBytes, &report) == nil {
						for _, file := range report.Files {
							if file.PlanillaID > 0 && strings.HasPrefix(filepath.ToSlash(file.Output), "4. EXPEDIENTES/"+patient+"/") {
								hasPlanilla = true
								break
							}
						}
					}
				}
			}
			if !hasPlanilla {
				_ = os.Remove(filepath.Join(packageRoot, "4. EXPEDIENTES", patient))
			} else {
				keptEmptyOracleFolders = append(keptEmptyOracleFolders, patient)
			}
		}
		removed := make([]string, 0, len(selected))
		for patient := range selected {
			removed = append(removed, patient)
		}
		sort.Strings(removed)
		sort.Strings(keptEmptyOracleFolders)
		message := fmt.Sprintf("Se quitaron %d PDFs de %d carpetas. Las fuentes originales se conservaron en el historial.", len(moving), len(removed))
		if len(keptEmptyOracleFolders) > 0 {
			message += fmt.Sprintf(" %d carpetas asociadas a trámites Oracle se conservaron vacías.", len(keptEmptyOracleFolders))
		}
		writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "carpetas_quitadas": removed, "carpetas_oracle_conservadas_vacias": keptEmptyOracleFolders, "pdfs_quitados": len(moving), "message": message})
	default:
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
	}
}

func (s *server) createTeamFoldersPeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	session, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var input struct {
		Service string `json:"tipo_servicio"`
		Month   string `json:"mes"`
		Year    string `json:"anio"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, http.StatusBadRequest, "Indica servicio, mes y año.")
		return
	}
	input.Service = strings.ToUpper(strings.TrimSpace(input.Service))
	input.Month = strings.TrimSpace(input.Month)
	input.Year = strings.TrimSpace(input.Year)
	month, monthErr := strconv.Atoi(input.Month)
	year, yearErr := strconv.Atoi(input.Year)
	if !validServiceCode(input.Service) || monthErr != nil || month < 1 || month > 12 || len(input.Month) != 2 || yearErr != nil || year < 2000 || year > 2100 || len(input.Year) != 4 {
		writeError(w, http.StatusBadRequest, "Indica un servicio, mes y año válidos.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	if err := os.MkdirAll(s.workspacesDir, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el período.")
		return
	}
	// Reuse any saved workspace for this exact service and period, including a
	// legacy job ID. The stable ID below is shared with normal planilla intake.
	entries, err := os.ReadDir(s.workspacesDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron revisar los períodos existentes.")
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(s.workspacesDir, entry.Name(), "job.json"))
		if readErr != nil {
			continue
		}
		var existing stagedJob
		if json.Unmarshal(data, &existing) != nil || existing.Service != input.Service || existing.Month != input.Month || existing.Year != input.Year || existing.IsObjections {
			continue
		}
		if existing.Status != "PROCESSED" && existing.Status != "INCOMPLETE" {
			writeError(w, http.StatusConflict, "Ya existe este período, pero todavía no está preparado para recibir carpetas.")
			return
		}
		if existing.TeamFoldersPeriod || s.hasClinicalSource(existing) || len(existing.ExternalPDFs) > 0 {
			output := filepath.Join(s.jobRoot(existing.ID), "trabajo", packageFolderName(&existing))
			writeJSON(w, http.StatusOK, map[string]any{"period": jobResponse(s, existing, output, nil), "reused": true})
			return
		}
	}
	jobID := periodWorkspaceID(input.Service, input.Month, input.Year)
	destination := s.jobRoot(jobID)
	if _, err := os.Stat(filepath.Join(destination, "job.json")); err == nil {
		writeError(w, http.StatusConflict, "Ya existe un espacio para este período que no se puede abrir. Actualiza la lista o revisa el período guardado.")
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "No se pudo revisar el período existente.")
		return
	}
	temporary, err := os.MkdirTemp(s.workspacesDir, ".team-period-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo crear el período.")
		return
	}
	defer os.RemoveAll(temporary)
	job := stagedJob{ID: jobID, Status: "INCOMPLETE", TeamFoldersPeriod: true, Username: session.username, Month: input.Month, Year: input.Year, Service: input.Service, ReceivedAt: time.Now().UTC(), Files: []stagedUpload{}, StatusDetail: "Período creado para recibir carpetas del equipo."}
	packageRoot := filepath.Join(temporary, "trabajo", packageFolderName(&job))
	if err := os.MkdirAll(filepath.Join(packageRoot, "4. EXPEDIENTES"), 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carpeta del período.")
		return
	}
	if err := os.Mkdir(filepath.Join(temporary, "fuentes"), 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el historial del período.")
		return
	}
	metadata, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el período.")
		return
	}
	if err := os.WriteFile(filepath.Join(temporary, "job.json"), metadata, 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el período.")
		return
	}
	if err := os.Rename(temporary, destination); err != nil {
		writeError(w, http.StatusConflict, "No se pudo crear el período porque ya existe otro espacio para esa combinación.")
		return
	}
	output := filepath.Join(destination, "trabajo", packageFolderName(&job))
	writeJSON(w, http.StatusCreated, map[string]any{"period": jobResponse(s, job, output, nil), "reused": false})
}

func auditorImportRoot(s *server) string {
	return filepath.Join(s.stagingDir, "auditoria-carpetas")
}

func cleanupAuditorImportStages(root string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err == nil && time.Since(info.ModTime()) > 2*time.Hour {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
}

func inspectAuditorZIP(filename string, maxBytes int64) ([]auditorImportFolder, bool, error) {
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return nil, false, errors.New("El archivo no es un ZIP válido.")
	}
	defer archive.Close()
	type source struct {
		file *zip.File
		path string
	}
	files := make([]source, 0)
	var total uint64
	for _, entry := range archive.File {
		name := strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/")
		if name == "" || strings.HasPrefix(name, "__MACOSX/") || filepath.Base(name) == ".DS_Store" {
			continue
		}
		if entry.Mode()&os.ModeSymlink != 0 || strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
			return nil, false, errors.New("El ZIP contiene una ruta no permitida.")
		}
		for _, segment := range strings.Split(name, "/") {
			if segment == ".." || segment == "." || segment == "" {
				return nil, false, errors.New("El ZIP contiene una ruta no permitida.")
			}
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".pdf") || entry.UncompressedSize64 == 0 || entry.UncompressedSize64 > 256<<20 {
			return nil, false, errors.New("El ZIP debe contener únicamente PDFs de hasta 256 MiB.")
		}
		total += entry.UncompressedSize64
		if total > uint64(maxBytes) || len(files) >= 1000 {
			return nil, false, errors.New("El ZIP supera el límite de tamaño o contiene más de 1000 PDFs.")
		}
		reader, openErr := entry.Open()
		if openErr != nil {
			return nil, false, errors.New("No se pudo leer uno de los PDFs del ZIP.")
		}
		var signature [5]byte
		_, readErr := io.ReadFull(reader, signature[:])
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || string(signature[:]) != "%PDF-" {
			return nil, false, errors.New("Uno de los archivos no tiene un encabezado PDF válido.")
		}
		files = append(files, source{file: entry, path: name})
	}
	if len(files) == 0 {
		return nil, false, errors.New("El ZIP no contiene PDFs.")
	}
	// Strip a single enclosing directory when the ZIP contains a whole parent folder.
	commonRoot := ""
	stripRoot := true
	for _, item := range files {
		parts := strings.Split(item.path, "/")
		if len(parts) < 3 {
			stripRoot = false
			break
		}
		if commonRoot == "" {
			commonRoot = parts[0]
		} else if commonRoot != parts[0] {
			stripRoot = false
			break
		}
	}
	folders := make(map[string]*auditorImportFolder)
	for _, item := range files {
		parts := strings.Split(item.path, "/")
		if stripRoot {
			parts = parts[1:]
		}
		if len(parts) != 2 {
			return nil, false, errors.New("Organiza el ZIP con una carpeta por paciente y los PDFs directamente dentro, sin subcarpetas.")
		}
		folder := normalizePatientFolder(parts[0])
		filename := safeOriginalFilename(parts[1])
		if folder == "" || filename == "" || !strings.EqualFold(filepath.Ext(filename), ".pdf") {
			return nil, false, errors.New("El ZIP contiene un nombre de carpeta o PDF no válido.")
		}
		if folders[folder] == nil {
			folders[folder] = &auditorImportFolder{Name: folder, Files: []auditorImportFile{}}
		}
		for _, existing := range folders[folder].Files {
			if strings.EqualFold(existing.Name, filename) {
				return nil, false, fmt.Errorf("La carpeta %s tiene dos PDFs con el mismo nombre.", folder)
			}
		}
		folders[folder].Files = append(folders[folder].Files, auditorImportFile{Name: filename, Size: item.file.UncompressedSize64})
	}
	result := make([]auditorImportFolder, 0, len(folders))
	for _, folder := range folders {
		sort.Slice(folder.Files, func(i, j int) bool {
			return strings.ToLower(folder.Files[i].Name) < strings.ToLower(folder.Files[j].Name)
		})
		result = append(result, *folder)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, stripRoot, nil
}

func auditorImportPlanillas(ctx context.Context, s *server, job stagedJob) ([]auditorImportPlanilla, error) {
	if s.serviceDB == nil {
		return nil, errors.New("Oracle no está disponible.")
	}
	rows, err := s.workspaceOraclePlanillas(ctx, job)
	if err != nil {
		return nil, err
	}
	result := make([]auditorImportPlanilla, 0, len(rows))
	for _, row := range rows {
		result = append(result, auditorImportPlanilla{PlanillaID: row.PlanillaID, Tramite: row.Tramite, Service: row.Service})
	}
	return result, nil
}

func (s *server) importAuditorFolders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	session, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/importar-carpetas/")
	if !validJobID(jobID) {
		writeError(w, http.StatusBadRequest, "El identificador del período no es válido.")
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
		writeError(w, http.StatusConflict, "Selecciona un período de recepción ya preparado.")
		return
	}
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		s.applyAuditorFolders(w, r, job, session)
		return
	}
	if s.maxIngest < 1 {
		writeError(w, http.StatusServiceUnavailable, "La carga de archivos no está configurada.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxIngest)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer el ZIP de carpetas.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	archives := r.MultipartForm.File["zip_file"]
	if len(archives) != 1 || archives[0].Size == 0 || !strings.EqualFold(filepath.Ext(archives[0].Filename), ".zip") {
		writeError(w, http.StatusBadRequest, "Selecciona un ZIP con carpetas de pacientes.")
		return
	}
	if err := os.MkdirAll(auditorImportRoot(s), 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la carga temporal.")
		return
	}
	cleanupAuditorImportStages(auditorImportRoot(s))
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la vista previa.")
		return
	}
	token := hex.EncodeToString(tokenBytes)
	stageDir := filepath.Join(auditorImportRoot(s), token)
	if err := os.Mkdir(stageDir, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la vista previa.")
		return
	}
	zipPath := filepath.Join(stageDir, "carpetas.zip")
	source, err := archives[0].Open()
	if err != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusBadRequest, "No se pudo abrir el ZIP seleccionado.")
		return
	}
	target, err := os.OpenFile(zipPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = source.Close()
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la carga temporal.")
		return
	}
	_, copyErr := io.Copy(target, source)
	closeSourceErr, closeTargetErr := source.Close(), target.Close()
	if copyErr != nil || closeSourceErr != nil || closeTargetErr != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el ZIP temporal.")
		return
	}
	folders, stripRoot, err := inspectAuditorZIP(zipPath, s.maxIngest)
	if err != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	planillas, err := auditorImportPlanillas(r.Context(), s, job)
	if err != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	stage := auditorImportStage{JobID: job.ID, Username: session.username, CreatedAt: time.Now().UTC(), Folders: folders, Planillas: planillas, StripRoot: stripRoot}
	metadata, err := json.Marshal(stage)
	if err != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la vista previa.")
		return
	}
	if err := os.WriteFile(filepath.Join(stageDir, "preview.json"), metadata, 0600); err != nil {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la vista previa.")
		return
	}
	total := 0
	for _, folder := range folders {
		total += len(folder.Files)
	}
	writeJSON(w, http.StatusOK, auditorImportPreviewResponse{PreviewID: token, Folders: folders, Planillas: planillas, TotalPDFs: total})
}

func workspacePlanillaDocumentFolder(s *server, job stagedJob, planillaID int64) (string, error) {
	folders := make(map[string]bool)
	addPath := func(raw string) {
		parts := strings.Split(filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw))), "/")
		if len(parts) >= 3 && parts[0] == "4. EXPEDIENTES" {
			folders[parts[1]] = true
		}
	}
	for relative, id := range job.DocumentPlanillas {
		if id == planillaID {
			addPath(relative)
		}
	}
	for _, doc := range job.ExternalPDFs {
		if doc.PlanillaID == planillaID && !doc.OraclePending {
			addPath(doc.RelativePath)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("No se pudo comprobar la carpeta asociada al trámite.")
	}
	if err == nil {
		var report classificationReport
		if json.Unmarshal(data, &report) != nil {
			return "", errors.New("El reporte de clasificación no es válido.")
		}
		for _, item := range report.Files {
			if item.PlanillaID == planillaID {
				addPath(item.Output)
			}
		}
	}
	if len(folders) > 1 {
		return "", errors.New("El trámite ya aparece asociado a varias carpetas; revisa el expediente antes de continuar.")
	}
	for folder := range folders {
		return folder, nil
	}
	return "", nil
}

func (s *server) associateAuditorPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/documentos/asociar-oracle/")
	if !validJobID(jobID) {
		writeError(w, http.StatusBadRequest, "El identificador del período no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input struct {
		Path       string `json:"path"`
		PlanillaID int64  `json:"pdi_id"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || input.PlanillaID <= 0 {
		writeError(w, http.StatusBadRequest, "Selecciona un trámite Oracle válido.")
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
	relative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(input.Path)))
	parts := strings.Split(relative, "/")
	if relative != input.Path || len(parts) != 3 || parts[0] != "4. EXPEDIENTES" || !strings.EqualFold(filepath.Ext(parts[2]), ".pdf") {
		writeError(w, http.StatusBadRequest, "Selecciona un PDF de una carpeta de paciente.")
		return
	}
	index := -1
	for i := range job.ExternalPDFs {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(job.ExternalPDFs[i].RelativePath)))
		if path == relative && job.ExternalPDFs[i].OraclePending {
			index = i
			break
		}
	}
	if index < 0 {
		writeError(w, http.StatusNotFound, "Ese PDF no está pendiente de asociación Oracle.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	absolute, err := safeWorkspacePath(packageRoot, relative)
	if err != nil {
		writeError(w, http.StatusBadRequest, "La ruta del PDF no es válida.")
		return
	}
	if info, err := os.Stat(absolute); err != nil || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "No se encontró el PDF seleccionado.")
		return
	}
	rows, err := s.workspaceOraclePlanillas(r.Context(), job)
	if err != nil {
		writeError(w, http.StatusBadGateway, "No se pudieron consultar los trámites Oracle del período.")
		return
	}
	var selected *workspaceOraclePlanilla
	for i := range rows {
		if rows[i].PlanillaID == input.PlanillaID {
			selected = &rows[i]
			break
		}
	}
	if selected == nil {
		writeError(w, http.StatusConflict, "El trámite no está planillado en Oracle para este período y servicio.")
		return
	}
	otherFolder, err := workspacePlanillaDocumentFolder(s, job, input.PlanillaID)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if otherFolder != "" && otherFolder != parts[1] {
		writeError(w, http.StatusConflict, "Ese trámite ya tiene PDFs asociados a otra carpeta de paciente.")
		return
	}
	previousJob, err := cloneStagedJob(job)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la asociación.")
		return
	}
	job.ExternalPDFs[index].PlanillaID = input.PlanillaID
	job.ExternalPDFs[index].OraclePending = false
	found := false
	for i := range job.OraclePlanillas {
		if job.OraclePlanillas[i].PlanillaID == input.PlanillaID {
			job.OraclePlanillas[i].PatientFolder = parts[1]
			found = true
		}
	}
	if !found {
		row := *selected
		row.PatientFolder = parts[1]
		job.OraclePlanillas = append(job.OraclePlanillas, row)
	}
	if err := s.saveStagedJob(job); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar la asociación.")
		return
	}
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		_ = s.saveStagedJob(previousJob)
		writeError(w, http.StatusInternalServerError, "No se pudo actualizar la lista de PDFs; la asociación no se completó.")
		return
	}
	if err := s.syncOracleWorkspace(r.Context(), &job, packageRoot); err != nil {
		if restoreErr := s.saveStagedJob(previousJob); restoreErr != nil {
			log.Printf("expediente %s: no se pudo restaurar la asociación pendiente: %v", job.ID, restoreErr)
			writeError(w, http.StatusInternalServerError, "Falló Oracle y no se pudo restaurar el estado pendiente; revisa el expediente antes de continuar.")
			return
		}
		log.Printf("expediente %s: falló la asociación Oracle de %s: %v", job.ID, relative, err)
		writeError(w, http.StatusBadGateway, "No se confirmó la asociación con Oracle; se conservó el PDF como pendiente.")
		return
	}
	for i := range documents {
		for _, doc := range job.ExternalPDFs {
			if filepath.ToSlash(doc.RelativePath) == documents[i].Path {
				documents[i].PlanillaID = doc.PlanillaID
				documents[i].PendingOracle = doc.OraclePending
			}
		}
	}
	available := make([]auditorImportPlanilla, 0, len(rows))
	for _, row := range rows {
		available = append(available, auditorImportPlanilla{PlanillaID: row.PlanillaID, Tramite: row.Tramite, Service: row.Service})
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients, "planillas": workspacePlanillaOptions(job, documents), "planillas_disponibles": available, "message": "El PDF quedó asociado al trámite Oracle."})
}

func (s *server) applyAuditorFolders(w http.ResponseWriter, r *http.Request, job stagedJob, currentSession session) {
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	var input auditorImportApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer la confirmación de importación.")
		return
	}
	decodedToken, tokenErr := hex.DecodeString(input.PreviewID)
	if tokenErr != nil || len(decodedToken) != 24 || len(input.PreviewID) != 48 {
		writeError(w, http.StatusBadRequest, "La vista previa de carga expiró; vuelve a seleccionar el ZIP.")
		return
	}
	stageDir := filepath.Join(auditorImportRoot(s), input.PreviewID)
	metadata, err := os.ReadFile(filepath.Join(stageDir, "preview.json"))
	if err != nil {
		writeError(w, http.StatusNotFound, "La vista previa expiró; vuelve a seleccionar el ZIP.")
		return
	}
	var stage auditorImportStage
	if json.Unmarshal(metadata, &stage) != nil || stage.JobID != job.ID || stage.Username != currentSession.username || time.Since(stage.CreatedAt) > 2*time.Hour {
		_ = os.RemoveAll(stageDir)
		writeError(w, http.StatusConflict, "La vista previa ya no es válida; vuelve a seleccionar el ZIP.")
		return
	}
	oracleRows, err := s.workspaceOraclePlanillas(r.Context(), job)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	validIDs := make(map[int64]workspaceOraclePlanilla, len(oracleRows))
	for _, row := range oracleRows {
		validIDs[row.PlanillaID] = row
	}
	totalFiles := 0
	planillaFolders := make(map[int64]string)
	for _, folder := range stage.Folders {
		for _, file := range folder.Files {
			totalFiles++
			id, exists := input.FileMappings[folder.Name+"/"+file.Name]
			if !exists || id < 0 {
				writeError(w, http.StatusBadRequest, "Asocia cada PDF a un trámite o márcalo como pendiente de asociación.")
				return
			}
			if id == 0 {
				continue
			}
			if _, ok := validIDs[id]; !ok {
				writeError(w, http.StatusConflict, "Uno de los trámites cambió en Oracle; vuelve a cargar el ZIP y revisa la asociación.")
				return
			}
			if otherFolder, used := planillaFolders[id]; used && otherFolder != folder.Name {
				writeError(w, http.StatusBadRequest, "Un trámite Oracle no puede asociarse a carpetas de pacientes distintas.")
				return
			}
			planillaFolders[id] = folder.Name
		}
	}
	if len(input.FileMappings) != totalFiles {
		writeError(w, http.StatusBadRequest, "La asociación de PDFs no coincide con la vista previa.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	expRoot := filepath.Join(packageRoot, "4. EXPEDIENTES")
	if err := os.MkdirAll(expRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron preparar las carpetas del período.")
		return
	}
	previousJob, err := cloneStagedJob(job)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la importación.")
		return
	}
	archive, err := zip.OpenReader(filepath.Join(stageDir, "carpetas.zip"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "El ZIP temporal dejó de estar disponible; vuelve a cargarlo.")
		return
	}
	defer archive.Close()
	entries := make(map[string]*zip.File)
	for _, entry := range archive.File {
		name := strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/")
		if entry.FileInfo().IsDir() || strings.HasPrefix(name, "__MACOSX/") || filepath.Base(name) == ".DS_Store" {
			continue
		}
		parts := strings.Split(name, "/")
		if stage.StripRoot && len(parts) > 2 {
			parts = parts[1:]
		}
		if len(parts) == 2 {
			entries[normalizePatientFolder(parts[0])+"/"+safeOriginalFilename(parts[1])] = entry
		}
	}
	sourceRoot := filepath.Join(s.jobSourcesDir(job.ID), "externos")
	if err := os.MkdirAll(sourceRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el historial de archivos originales.")
		return
	}
	createdFiles := make([]string, 0)
	createdDirs := make(map[string]bool)
	added := make([]workspaceDocument, 0)
	receiptByFolder := make(map[string]string, len(stage.Folders))
	for _, folder := range stage.Folders {
		for _, receipt := range job.AuditorReceipts {
			if receipt.Patient == folder.Name {
				receiptByFolder[folder.Name] = receipt.ID
				break
			}
		}
		if receiptByFolder[folder.Name] == "" {
			idBytes := make([]byte, 20)
			if _, err := rand.Read(idBytes); err != nil {
				writeError(w, http.StatusInternalServerError, "No se pudo crear el registro Oracle de una carpeta.")
				return
			}
			receiptByFolder[folder.Name] = hex.EncodeToString(idBytes)
			job.AuditorReceipts = append(job.AuditorReceipts, auditorReceipt{ID: receiptByFolder[folder.Name], Patient: folder.Name, Service: job.Service, Month: job.Month, Year: job.Year, Status: "PENDIENTE_PLANILLA"})
		}
	}
	rollback := func() {
		for _, name := range createdFiles {
			_ = os.Remove(name)
		}
		for dir := range createdDirs {
			_ = os.Remove(dir)
		}
		_ = s.saveStagedJob(previousJob)
	}
	for _, folder := range stage.Folders {
		patientPath, pathErr := safeWorkspacePath(packageRoot, filepath.ToSlash(filepath.Join("4. EXPEDIENTES", folder.Name)))
		if pathErr != nil {
			rollback()
			writeError(w, http.StatusBadRequest, "El nombre de una carpeta no es válido.")
			return
		}
		info, statErr := os.Stat(patientPath)
		if errors.Is(statErr, os.ErrNotExist) {
			if err := os.Mkdir(patientPath, 0700); err != nil {
				rollback()
				writeError(w, http.StatusInternalServerError, "No se pudo crear una carpeta de paciente.")
				return
			}
			createdDirs[patientPath] = true
		} else if statErr != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo comprobar una carpeta de paciente.")
			return
		} else if !info.IsDir() {
			rollback()
			writeError(w, http.StatusConflict, "El nombre de una carpeta de paciente coincide con un archivo existente.")
			return
		}
		for _, fileInfo := range folder.Files {
			entry := entries[folder.Name+"/"+fileInfo.Name]
			if entry == nil {
				rollback()
				writeError(w, http.StatusConflict, "El ZIP no coincide con la vista previa; vuelve a cargarlo.")
				return
			}
			planillaID := input.FileMappings[folder.Name+"/"+fileInfo.Name]
			randomID := make([]byte, 12)
			if _, err := rand.Read(randomID); err != nil {
				rollback()
				writeError(w, http.StatusInternalServerError, "No se pudo preparar un PDF.")
				return
			}
			storedName := hex.EncodeToString(randomID) + ".pdf"
			sourcePath := filepath.Join(sourceRoot, storedName)
			out, err := os.OpenFile(sourcePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				rollback()
				writeError(w, http.StatusInternalServerError, "No se pudo guardar un PDF original.")
				return
			}
			in, err := entry.Open()
			var copied int64
			if err == nil {
				copied, err = io.Copy(out, io.LimitReader(in, int64(fileInfo.Size)+1))
			}
			if in != nil {
				_ = in.Close()
			}
			closeErr := out.Close()
			if err != nil || closeErr != nil || copied != int64(fileInfo.Size) {
				createdFiles = append(createdFiles, sourcePath)
				rollback()
				writeError(w, http.StatusInternalServerError, "No se pudo extraer un PDF del ZIP.")
				return
			}
			createdFiles = append(createdFiles, sourcePath)
			if err := validateStagedFile(sourcePath, ".pdf"); err != nil {
				rollback()
				writeError(w, http.StatusBadRequest, "Uno de los archivos no contiene un PDF válido.")
				return
			}
			filename, err := nextPendingWorkspacePDFName(patientPath, fileInfo.Name)
			if err != nil {
				rollback()
				writeError(w, http.StatusInternalServerError, "No se pudo generar un nombre temporal para un PDF.")
				return
			}
			relative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", folder.Name, filename))
			destination, err := safeWorkspacePath(packageRoot, relative)
			if err != nil {
				rollback()
				writeError(w, http.StatusBadRequest, "La ruta de un PDF no es válida.")
				return
			}
			if err := copyPrivateFile(sourcePath, destination); err != nil {
				_ = os.Remove(destination)
				rollback()
				writeError(w, http.StatusConflict, "No se pudo incorporar un PDF sin sobrescribir archivos existentes.")
				return
			}
			createdFiles = append(createdFiles, destination)
			added = append(added, workspaceDocument{ID: strings.TrimSuffix(storedName, ".pdf"), StoredName: storedName, OriginalName: fileInfo.Name, RelativePath: relative, Size: int64(fileInfo.Size), PlanillaID: planillaID, OraclePending: planillaID == 0, ReceiptID: receiptByFolder[folder.Name], TeamImport: true})
		}
	}
	for _, item := range added {
		job.ExternalPDFs = append(job.ExternalPDFs, item)
	}
	for index := range job.OraclePlanillas {
		if patient, ok := planillaFolders[job.OraclePlanillas[index].PlanillaID]; ok {
			job.OraclePlanillas[index].PatientFolder = patient
		}
	}
	for id, patient := range planillaFolders {
		found := false
		for _, existing := range job.OraclePlanillas {
			if existing.PlanillaID == id {
				found = true
				break
			}
		}
		if !found {
			row := validIDs[id]
			row.PatientFolder = patient
			job.OraclePlanillas = append(job.OraclePlanillas, row)
		}
	}
	if err := s.saveStagedJob(job); err != nil {
		rollback()
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la importación.")
		return
	}
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		rollback()
		writeError(w, http.StatusInternalServerError, "No se pudo actualizar la lista de PDFs.")
		return
	}
	if err := s.syncOracleWorkspace(r.Context(), &job, packageRoot); err != nil {
		log.Printf("expediente %s: falló sincronización Oracle tras importar carpetas: %v", job.ID, err)
		rollback()
		writeError(w, http.StatusBadGateway, "Los PDFs no se incorporaron porque no se pudo sincronizar Oracle.")
		return
	}
	_ = os.RemoveAll(stageDir)
	pendingCount := 0
	for _, item := range added {
		if item.OraclePending {
			pendingCount++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "added_count": len(added), "folder_count": len(stage.Folders), "pendientes_asociacion": pendingCount, "documents": documents, "patients": patients, "message": fmt.Sprintf("Se importaron %d PDFs en %d carpetas; %d quedaron pendientes de asociación Oracle.", len(added), len(stage.Folders), pendingCount)})
}
