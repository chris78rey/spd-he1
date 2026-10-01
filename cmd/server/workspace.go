package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type workspacePDF struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size_bytes"`
}

type mspPDFCode struct {
	Title    string `json:"title"`
	Value    string `json:"value"`
	Category string `json:"categoria"`
}

func loadMSPPDFCodeSet() (map[string]struct{}, error) {
	data, err := os.ReadFile(filepath.Join("catalogos", "codigos_msp.json"))
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar el catálogo MSP: %w", err)
	}
	var codes []mspPDFCode
	if err := json.Unmarshal(data, &codes); err != nil {
		return nil, fmt.Errorf("el catálogo MSP no es válido: %w", err)
	}
	allowed := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if code.Value == "" || filepath.Ext(code.Value) != ".pdf" || filepath.Base(code.Value) != code.Value {
			return nil, errors.New("el catálogo contiene un nombre PDF inválido")
		}
		allowed[code.Value] = struct{}{}
	}
	return allowed, nil
}

func nextMSPPDFName(dir, code, ignoredName string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	used := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == ignoredName {
			continue
		}
		used[entry.Name()] = true
	}
	if !used[code] {
		return code, nil
	}
	base := strings.TrimSuffix(code, ".pdf")
	for suffix := 1; ; suffix++ {
		candidate := fmt.Sprintf("%s_%d.pdf", base, suffix)
		if !used[candidate] {
			return candidate, nil
		}
	}
}

var workspaceDeleteCodePattern = regexp.MustCompile(`^[0-9]{8}$`)

func (s *server) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/eliminar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	var input struct {
		Confirmation string `json:"confirmacion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || !workspaceDeleteCodePattern.MatchString(input.Confirmation) {
		writeError(w, http.StatusBadRequest, "Escribe manualmente el código de confirmación de 8 dígitos.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	root := filepath.Clean(s.jobRoot(job.ID))
	base := filepath.Clean(s.workspacesDir)
	relative, err := filepath.Rel(base, root)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		writeError(w, http.StatusBadRequest, "La ruta del espacio no es válida.")
		return
	}
	info, err := os.Lstat(root)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró la carpeta completa del período.")
		return
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		writeError(w, http.StatusBadRequest, "La ruta del espacio no es una carpeta válida.")
		return
	}
	if err := os.RemoveAll(root); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo eliminar por completo la carpeta del período.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "DELETED", "job_id": job.ID, "message": "Se eliminó la carpeta completa del período con sus fuentes, reportes y documentos preparados."})
}

func (s *server) workspaceDocuments(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/documentos/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
		return
	}
	if r.Method == http.MethodPost {
		s.ingestMu.Lock()
		defer s.ingestMu.Unlock()
		// Reload after taking the lock so concurrent additions reserve distinct suffixes.
		job, err = s.loadStagedJob(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
			return
		}
	}
	if r.Method == http.MethodPost {
		s.addWorkspacePDFs(w, r, job)
		return
	}
	if r.Method == http.MethodDelete {
		s.deleteWorkspacePDF(w, r, job)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero los expedientes para poder revisar sus PDFs.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron listar los PDFs del expediente.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients})
}

func (s *server) deleteWorkspacePDF(w http.ResponseWriter, r *http.Request, job stagedJob) {
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero el expediente antes de quitar PDFs.")
		return
	}
	var input struct {
		Path string `json:"path"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Selecciona el PDF que quieres quitar.")
		return
	}
	cleanRelative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(input.Path)))
	if !strings.HasPrefix(cleanRelative, "4. EXPEDIENTES/") || !strings.EqualFold(filepath.Ext(cleanRelative), ".pdf") {
		writeError(w, http.StatusBadRequest, "Solo se pueden quitar PDFs de una carpeta de paciente.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	filePath, err := safeWorkspacePath(packageRoot, cleanRelative)
	if err != nil {
		writeError(w, http.StatusBadRequest, "La ruta del PDF no es válida.")
		return
	}
	info, err := os.Stat(filePath)
	if err != nil || !info.Mode().IsRegular() {
		writeError(w, http.StatusNotFound, "No se encontró el PDF seleccionado.")
		return
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el cambio.")
		return
	}
	backupPath := fmt.Sprintf("%s.folio-delete-%x", filePath, token[:])
	if err := os.Rename(filePath, backupPath); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo quitar el PDF.")
		return
	}
	previousDeleted := make(map[string]bool, len(job.DeletedPDFs))
	for path, deleted := range job.DeletedPDFs {
		previousDeleted[path] = deleted
	}
	if job.DeletedPDFs == nil {
		job.DeletedPDFs = make(map[string]bool)
	}
	job.DeletedPDFs[cleanRelative] = true
	if err := s.saveStagedJob(job); err != nil {
		job.DeletedPDFs = previousDeleted
		_ = os.Rename(backupPath, filePath)
		writeError(w, http.StatusInternalServerError, "No se pudo registrar el cambio; el PDF fue restaurado.")
		return
	}
	if err := os.Remove(backupPath); err != nil {
		writeError(w, http.StatusInternalServerError, "El PDF se quitó, pero no se pudo limpiar el respaldo temporal.")
		return
	}
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "El PDF se quitó, pero no se pudo actualizar la lista.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients, "removed": cleanRelative})
}

func (s *server) downloadWorkspaceZIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/descargar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero los expedientes antes de descargar el ZIP.")
		return
	}
	root := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() {
		writeError(w, http.StatusNotFound, "No se encontró la carpeta preparada del expediente.")
		return
	}
	fusionGroups, err := workspaceFusionGroups(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo revisar si hay documentos pendientes de fusionar.")
		return
	}
	if len(fusionGroups) > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("Hay %d grupo(s) de PDFs reconocidos pendientes de fusionar. Fusiónalos desde la carpeta del paciente antes de descargar el ZIP final.", len(fusionGroups)))
		return
	}
	archiveName := packageFolderName(&job) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", archiveName))
	w.Header().Set("Cache-Control", "no-store")
	archive := zip.NewWriter(w)
	writeDirectory := func(name string, info os.FileInfo) error {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(name) + "/"
		header.Method = zip.Store
		_, err = archive.CreateHeader(header)
		return err
	}
	if err := writeDirectory(packageFolderName(&job), rootInfo); err != nil {
		_ = archive.Close()
		return
	}
	walkErr := filepath.WalkDir(root, func(filePath string, file os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if filePath == root {
			return nil
		}
		if strings.HasPrefix(file.Name(), ".") {
			if file.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		archivePath := filepath.ToSlash(filepath.Join(packageFolderName(&job), relative))
		if file.IsDir() {
			info, err := file.Info()
			if err != nil {
				return err
			}
			return writeDirectory(archivePath, info)
		}
		if !file.Type().IsRegular() {
			return nil
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = archivePath
		header.Method = zip.Deflate
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		input, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	closeErr := archive.Close()
	if walkErr != nil || closeErr != nil {
		log.Printf("expediente %s: falló exportación ZIP (walk=%v close=%v)", job.ID, walkErr, closeErr)
	}
}

func listWorkspacePDFs(packageRoot string) ([]workspacePDF, []string, error) {
	documents := make([]workspacePDF, 0)
	patients := make([]string, 0)
	expedientesRoot := filepath.Join(packageRoot, "4. EXPEDIENTES")
	patientEntries, err := os.ReadDir(expedientesRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	for _, entry := range patientEntries {
		if entry.IsDir() {
			patients = append(patients, entry.Name())
		}
	}
	if err := filepath.WalkDir(packageRoot, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(packageRoot, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		documents = append(documents, workspacePDF{Path: relative, Name: entry.Name(), Size: info.Size()})
		return nil
	}); err != nil {
		return nil, nil, err
	}
	sort.Slice(documents, func(i, j int) bool { return strings.ToLower(documents[i].Path) < strings.ToLower(documents[j].Path) })
	sort.Strings(patients)
	return documents, patients, nil
}

type workspaceFusionGroup struct {
	Code          string   `json:"codigo"`
	CanonicalPath string   `json:"ruta_canonica"`
	Paths         []string `json:"rutas"`
}

var numberedMSPName = regexp.MustCompile(`^(.*)_([1-9][0-9]*)\.pdf$`)

func workspaceFusionGroups(packageRoot string) ([]workspaceFusionGroup, error) {
	codes, err := loadMSPPDFCodeSet()
	if err != nil {
		return nil, err
	}
	documents, _, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		return nil, err
	}
	groups := make(map[string]*workspaceFusionGroup)
	for _, document := range documents {
		if !strings.HasPrefix(document.Path, "4. EXPEDIENTES/") {
			continue
		}
		code := document.Name
		if _, ok := codes[code]; !ok {
			match := numberedMSPName.FindStringSubmatch(code)
			if match == nil {
				continue
			}
			code = match[1] + ".pdf"
			if _, ok := codes[code]; !ok {
				continue
			}
		}
		directory := filepath.ToSlash(filepath.Dir(filepath.FromSlash(document.Path)))
		key := directory + "/" + code
		group := groups[key]
		if group == nil {
			group = &workspaceFusionGroup{Code: code, CanonicalPath: filepath.ToSlash(filepath.Join(directory, code))}
			groups[key] = group
		}
		group.Paths = append(group.Paths, document.Path)
	}
	result := make([]workspaceFusionGroup, 0)
	for _, group := range groups {
		if len(group.Paths) < 2 {
			continue
		}
		sort.Slice(group.Paths, func(i, j int) bool {
			if group.Paths[i] == group.CanonicalPath {
				return group.Paths[j] != group.CanonicalPath
			}
			if group.Paths[j] == group.CanonicalPath {
				return false
			}
			return group.Paths[i] < group.Paths[j]
		})
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CanonicalPath < result[j].CanonicalPath })
	return result, nil
}

func safeWorkspacePath(root, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" || strings.ContainsRune(relative, '\x00') {
		return "", errors.New("ruta vacía")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("ruta fuera del espacio")
	}
	absolute := filepath.Join(root, clean)
	check, err := filepath.Rel(root, absolute)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) {
		return "", errors.New("ruta fuera del espacio")
	}
	return absolute, nil
}

func copyReplacePrivateFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".replace-*.pdf")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, input); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func (s *server) serveWorkspacePDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/documentos/archivo/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero los expedientes para poder ver sus PDFs.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	filePath, err := safeWorkspacePath(packageRoot, r.URL.Query().Get("path"))
	if err != nil || !strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		writeError(w, http.StatusBadRequest, "Selecciona un PDF válido del expediente.")
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(filePath)))
	http.ServeContent(w, r, filepath.Base(filePath), info.ModTime(), file)
}

func (s *server) renameWorkspacePDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/documentos/renombrar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	var input struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Indica el PDF y su nuevo nombre.")
		return
	}
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero los expedientes para renombrar sus PDFs.")
		return
	}
	if !strings.HasPrefix(filepath.ToSlash(input.Path), "4. EXPEDIENTES/") {
		writeError(w, http.StatusBadRequest, "Solo se pueden renombrar PDFs dentro de la carpeta de expedientes.")
		return
	}
	if !strings.EqualFold(filepath.Ext(input.Path), ".pdf") {
		writeError(w, http.StatusBadRequest, "Solo se pueden renombrar archivos PDF.")
		return
	}
	allowedCodes, err := loadMSPPDFCodeSet()
	if err != nil {
		log.Printf("expediente %s: %v", job.ID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo cargar el catálogo de nombres MSP.")
		return
	}
	if _, ok := allowedCodes[input.Name]; !ok {
		writeError(w, http.StatusBadRequest, "Elige un nombre PDF del catálogo MSP.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	oldPath, err := safeWorkspacePath(packageRoot, input.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "La ruta del PDF no es válida.")
		return
	}
	if _, err := os.Stat(oldPath); err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el PDF seleccionado.")
		return
	}
	newName, err := nextMSPPDFName(filepath.Dir(oldPath), input.Name, filepath.Base(oldPath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo comprobar el nombre disponible en la carpeta.")
		return
	}
	newRelative := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(input.Path)), newName))
	newPath, err := safeWorkspacePath(packageRoot, newRelative)
	if err != nil {
		writeError(w, http.StatusBadRequest, "El nuevo nombre no es válido.")
		return
	}
	if oldPath == newPath {
		writeJSON(w, http.StatusOK, map[string]string{"path": newRelative, "name": newName})
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo renombrar el PDF.")
		return
	}
	oldRelative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(input.Path)))
	if job.Renames == nil {
		job.Renames = make(map[string]string)
	}
	original := oldRelative
	for source, target := range job.Renames {
		if target == oldRelative {
			original = source
			break
		}
	}
	if original == newRelative {
		delete(job.Renames, original)
	} else {
		job.Renames[original] = newRelative
	}
	for index := range job.ExternalPDFs {
		if filepath.ToSlash(job.ExternalPDFs[index].RelativePath) == oldRelative {
			job.ExternalPDFs[index].RelativePath = newRelative
		}
	}
	if storedName, exists := job.Replacements[oldRelative]; exists {
		delete(job.Replacements, oldRelative)
		if job.Replacements == nil {
			job.Replacements = make(map[string]string)
		}
		job.Replacements[newRelative] = storedName
	}
	if err := s.saveStagedJob(job); err != nil {
		_ = os.Rename(newPath, oldPath)
		writeError(w, http.StatusInternalServerError, "El PDF se renombró, pero no se pudo guardar el cambio.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": newRelative, "name": newName})
}

func (s *server) mergeWorkspacePDFs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/expedientes/documentos/fusionar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	var input struct {
		Paths []string `json:"rutas"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Paths) < 2 || len(input.Paths) > 50 {
		writeError(w, http.StatusBadRequest, "Selecciona entre 2 y 50 PDFs repetidos para fusionar.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de trabajo.")
		return
	}
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero el expediente antes de fusionar documentos.")
		return
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	groups, err := workspaceFusionGroups(packageRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron revisar los PDFs repetidos.")
		return
	}
	selected := make(map[string]bool, len(input.Paths))
	for _, path := range input.Paths {
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if !strings.HasPrefix(clean, "4. EXPEDIENTES/") || selected[clean] {
			writeError(w, http.StatusBadRequest, "La selección de PDFs repetidos no es válida.")
			return
		}
		selected[clean] = true
	}
	var group *workspaceFusionGroup
	for index := range groups {
		if len(groups[index].Paths) != len(selected) {
			continue
		}
		all := true
		for _, path := range groups[index].Paths {
			if !selected[path] {
				all = false
				break
			}
		}
		if all {
			group = &groups[index]
			break
		}
	}
	if group == nil {
		writeError(w, http.StatusBadRequest, "Selecciona todos los PDFs del mismo tipo y paciente para fusionarlos.")
		return
	}
	// Keep the user-selected order; it determines the page order in the merged PDF.
	ordered := make([]string, 0, len(input.Paths))
	for _, path := range input.Paths {
		ordered = append(ordered, filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))))
	}
	sourceRoot := s.jobSourcesDir(job.ID)
	historyDir := filepath.Join(sourceRoot, "fusiones", time.Now().UTC().Format("20060102T150405.000000000"))
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el historial de la fusión.")
		return
	}
	inputs := make([]string, 0, len(ordered))
	for index, relative := range ordered {
		active, pathErr := safeWorkspacePath(packageRoot, relative)
		archive := filepath.Join(historyDir, fmt.Sprintf("documento-%02d.pdf", index+1))
		if pathErr != nil || copyPrivateFile(active, archive) != nil {
			writeError(w, http.StatusInternalServerError, "No se pudieron conservar las fuentes antes de fusionar.")
			return
		}
		inputs = append(inputs, archive)
	}
	externalRoot := filepath.Join(sourceRoot, "externos")
	if err := os.MkdirAll(externalRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar la salida de la fusión.")
		return
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo generar un identificador de fusión.")
		return
	}
	storedName := fmt.Sprintf("fusion-%x.pdf", idBytes)
	mergedSource := filepath.Join(externalRoot, storedName)
	if err := api.MergeCreateFile(inputs, mergedSource, false, nil); err != nil {
		_ = os.Remove(mergedSource)
		writeError(w, http.StatusUnprocessableEntity, "No se pudieron fusionar los PDFs seleccionados; se conservaron sin cambios.")
		return
	}
	if err := validateStagedFile(mergedSource, ".pdf"); err != nil {
		_ = os.Remove(mergedSource)
		writeError(w, http.StatusUnprocessableEntity, "La fusión no produjo un PDF válido; se conservaron los originales.")
		return
	}
	canonical, err := safeWorkspacePath(packageRoot, group.CanonicalPath)
	if err != nil {
		_ = os.Remove(mergedSource)
		writeError(w, http.StatusBadRequest, "La ruta del PDF consolidado no es válida.")
		return
	}
	backupPaths := make(map[string]string)
	for index, relative := range group.Paths {
		active, pathErr := safeWorkspacePath(packageRoot, relative)
		if pathErr != nil {
			writeError(w, http.StatusBadRequest, "Una ruta de PDF no es válida.")
			return
		}
		backup := active + fmt.Sprintf(".folio-fusion-backup-%d", index)
		if err := os.Rename(active, backup); err != nil {
			for path, saved := range backupPaths {
				_ = os.Rename(saved, path)
			}
			writeError(w, http.StatusInternalServerError, "No se pudieron resguardar temporalmente los PDFs originales.")
			return
		}
		backupPaths[active] = backup
	}
	if err := copyPrivateFile(mergedSource, canonical); err != nil {
		for path, saved := range backupPaths {
			_ = os.Rename(saved, path)
		}
		_ = os.Remove(mergedSource)
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el PDF fusionado; se restauraron los originales.")
		return
	}
	previousReplacements := make(map[string]string, len(job.Replacements))
	for path, stored := range job.Replacements {
		previousReplacements[path] = stored
	}
	previousExternal := append([]workspaceDocument(nil), job.ExternalPDFs...)
	previousMerged := make(map[string][]string, len(job.MergedDuplicates))
	for path, duplicates := range job.MergedDuplicates {
		previousMerged[path] = append([]string(nil), duplicates...)
	}
	canonicalRelative := group.CanonicalPath
	if job.Replacements == nil {
		job.Replacements = make(map[string]string)
	}
	if job.MergedDuplicates == nil {
		job.MergedDuplicates = make(map[string][]string)
	}
	job.Replacements[canonicalRelative] = storedName
	duplicates := make([]string, 0, len(group.Paths)-1)
	for _, path := range group.Paths {
		if path != canonicalRelative {
			duplicates = append(duplicates, path)
		}
	}
	job.MergedDuplicates[canonicalRelative] = duplicates
	job.ExternalPDFs = append(job.ExternalPDFs, workspaceDocument{ID: strings.TrimSuffix(storedName, ".pdf"), StoredName: storedName, OriginalName: group.Code, RelativePath: canonicalRelative})
	if err := s.saveStagedJob(job); err != nil {
		_ = os.Remove(canonical)
		for path, saved := range backupPaths {
			_ = os.Rename(saved, path)
		}
		job.Replacements, job.ExternalPDFs, job.MergedDuplicates = previousReplacements, previousExternal, previousMerged
		_ = os.Remove(mergedSource)
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la fusión; se restauraron los originales.")
		return
	}
	for _, saved := range backupPaths {
		_ = os.Remove(saved)
	}
	// Update the stored report so duplicate entries no longer appear as separate active outputs.
	reportPath := filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json")
	if raw, readErr := os.ReadFile(reportPath); readErr == nil {
		var report classificationReport
		if json.Unmarshal(raw, &report) == nil {
			removed := make(map[string]bool, len(duplicates))
			for _, path := range duplicates {
				removed[path] = true
			}
			files := report.Files[:0]
			for _, item := range report.Files {
				if removed[item.Output] {
					continue
				}
				if item.Output == canonicalRelative {
					item.Reason = "FUSION_APLICADA"
				}
				files = append(files, item)
			}
			report.Files = files
			report.Summary = summarizeClassification(report.Files)
			if writeErr := writePrivateJSON(reportPath, report); writeErr != nil {
				log.Printf("expediente %s: no se pudo actualizar el reporte tras fusionar: %v", job.ID, writeErr)
			}
		}
	}
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Se fusionaron los PDFs, pero no se pudo actualizar la lista.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients, "fused": canonicalRelative, "merged_count": len(group.Paths)})
}

func (s *server) addWorkspacePDFs(w http.ResponseWriter, r *http.Request, job stagedJob) {
	if job.Status != "PROCESSED" && job.Status != "INCOMPLETE" {
		writeError(w, http.StatusConflict, "Prepara primero el expediente antes de añadir PDFs.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxIngest)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo leer la carga de PDFs.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	patient := r.FormValue("paciente")
	if patient == "" || normalizePatientFolder(patient) != patient {
		writeError(w, http.StatusBadRequest, "Selecciona una carpeta de paciente válida.")
		return
	}
	files := r.MultipartForm.File["pdf_files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "Selecciona al menos un PDF para añadir.")
		return
	}
	codes := r.MultipartForm.Value["pdf_codes"]
	if len(codes) != len(files) && r.FormValue("modo") != "reemplazar" {
		writeError(w, http.StatusBadRequest, "Asigna un nombre MSP del catálogo a cada PDF.")
		return
	}
	allowedCodes, err := loadMSPPDFCodeSet()
	if err != nil {
		log.Printf("expediente %s: %v", job.ID, err)
		writeError(w, http.StatusInternalServerError, "No se pudo cargar el catálogo de nombres MSP.")
		return
	}
	if r.FormValue("modo") != "reemplazar" {
		for _, code := range codes {
			if _, ok := allowedCodes[code]; !ok {
				writeError(w, http.StatusBadRequest, "Cada PDF debe tener un nombre del catálogo MSP.")
				return
			}
		}
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	patientRelative := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", patient))
	patientRoot, err := safeWorkspacePath(packageRoot, patientRelative)
	if err != nil {
		writeError(w, http.StatusBadRequest, "La carpeta de paciente no es válida.")
		return
	}
	if info, err := os.Stat(patientRoot); err != nil || !info.IsDir() {
		writeError(w, http.StatusNotFound, "No se encontró la carpeta de paciente seleccionada.")
		return
	}
	sourceRoot := filepath.Join(s.jobSourcesDir(job.ID), "externos")
	mode := r.FormValue("modo")
	replaceRelative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(r.FormValue("ruta"))))
	if mode != "" && mode != "reemplazar" {
		writeError(w, http.StatusBadRequest, "La acción de carga no es válida.")
		return
	}
	if mode == "reemplazar" {
		patientPrefix := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", patient)) + "/"
		if len(files) != 1 || !strings.HasPrefix(replaceRelative, patientPrefix) || !strings.EqualFold(filepath.Ext(replaceRelative), ".pdf") {
			writeError(w, http.StatusBadRequest, "Selecciona un solo PDF y elige un archivo nuevo para reemplazarlo.")
			return
		}
	}
	for _, file := range files {
		if file.Size == 0 || file.Size > 256<<20 || !strings.EqualFold(filepath.Ext(file.Filename), ".pdf") {
			writeError(w, http.StatusBadRequest, "Cada archivo debe ser un PDF válido de hasta 256 MiB.")
			return
		}
	}
	if mode == "reemplazar" {
		destination, err := safeWorkspacePath(packageRoot, replaceRelative)
		if err != nil {
			writeError(w, http.StatusBadRequest, "La ruta del PDF seleccionado no es válida.")
			return
		}
		if _, err := os.Stat(destination); err != nil {
			writeError(w, http.StatusNotFound, "No se encontró el PDF que quieres reemplazar.")
			return
		}
		if err := os.MkdirAll(sourceRoot, 0700); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo preparar el almacenamiento de PDFs fuente.")
			return
		}
		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo generar un identificador de documento.")
			return
		}
		storedName := fmt.Sprintf("%x.pdf", idBytes)
		sourcePath := filepath.Join(sourceRoot, storedName)
		size, err := saveUploadPart(files[0], sourcePath)
		if err != nil {
			_ = os.Remove(sourcePath)
			writeError(w, http.StatusInternalServerError, "No se pudo guardar el PDF nuevo.")
			return
		}
		if err := validateStagedFile(sourcePath, ".pdf"); err != nil {
			_ = os.Remove(sourcePath)
			writeError(w, http.StatusBadRequest, "El archivo nuevo no contiene un PDF válido.")
			return
		}
		backupPath := destination + ".folio-backup"
		if _, err := os.Stat(backupPath); err == nil {
			_ = os.Remove(sourcePath)
			writeError(w, http.StatusConflict, "Hay otro reemplazo en curso para este PDF.")
			return
		}
		if err := os.Rename(destination, backupPath); err != nil {
			_ = os.Remove(sourcePath)
			writeError(w, http.StatusInternalServerError, "No se pudo resguardar el PDF anterior durante el reemplazo.")
			return
		}
		if err := copyPrivateFile(sourcePath, destination); err != nil {
			_ = os.Rename(backupPath, destination)
			_ = os.Remove(sourcePath)
			writeError(w, http.StatusInternalServerError, "No se pudo guardar el PDF reemplazado.")
			return
		}
		previousReplacements := make(map[string]string, len(job.Replacements))
		for path, stored := range job.Replacements {
			previousReplacements[path] = stored
		}
		previousExternal := append([]workspaceDocument(nil), job.ExternalPDFs...)
		if job.Replacements == nil {
			job.Replacements = make(map[string]string)
		}
		job.Replacements[replaceRelative] = storedName
		job.ExternalPDFs = append(job.ExternalPDFs, workspaceDocument{ID: strings.TrimSuffix(storedName, ".pdf"), StoredName: storedName, OriginalName: safeOriginalFilename(files[0].Filename), RelativePath: replaceRelative, Size: size})
		if err := s.saveStagedJob(job); err != nil {
			_ = os.Remove(destination)
			_ = os.Rename(backupPath, destination)
			_ = os.Remove(sourcePath)
			job.Replacements = previousReplacements
			job.ExternalPDFs = previousExternal
			writeError(w, http.StatusInternalServerError, "No se pudo registrar el reemplazo; se restauró el PDF anterior.")
			return
		}
		_ = os.Remove(backupPath)
		documents, patients, err := listWorkspacePDFs(packageRoot)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "El reemplazo se guardó, pero no se pudo actualizar la lista.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients, "replaced": replaceRelative})
		return
	}
	if err := os.MkdirAll(sourceRoot, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el almacenamiento de PDFs fuente.")
		return
	}
	added := make([]workspaceDocument, 0, len(files))
	newTargets := make(map[string]bool)
	rollback := func() {
		for _, document := range added {
			_ = os.Remove(filepath.Join(sourceRoot, document.StoredName))
		}
		for relative := range newTargets {
			if target, pathErr := safeWorkspacePath(packageRoot, relative); pathErr == nil {
				_ = os.Remove(target)
			}
		}
	}
	for index, file := range files {
		idBytes := make([]byte, 8)
		if _, err := rand.Read(idBytes); err != nil {
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo generar un identificador de documento.")
			return
		}
		storedName := fmt.Sprintf("%x.pdf", idBytes)
		sourcePath := filepath.Join(sourceRoot, storedName)
		size, err := saveUploadPart(file, sourcePath)
		if err != nil {
			_ = os.Remove(sourcePath)
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo guardar uno de los PDFs.")
			return
		}
		if err := validateStagedFile(sourcePath, ".pdf"); err != nil {
			_ = os.Remove(sourcePath)
			rollback()
			writeError(w, http.StatusBadRequest, "Uno de los archivos no contiene un PDF válido.")
			return
		}
		// Adding a document never replaces another one. Repeated MSP codes get
		// the next free suffix; replacement is handled by the explicit replace mode above.
		filename, err := nextMSPPDFName(patientRoot, codes[index], "")
		if err != nil {
			_ = os.Remove(sourcePath)
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo comprobar un nombre PDF disponible.")
			return
		}
		relative := filepath.ToSlash(filepath.Join(patientRelative, filename))
		destination, err := safeWorkspacePath(packageRoot, relative)
		if err != nil {
			_ = os.Remove(sourcePath)
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo incorporar uno de los PDFs al expediente.")
			return
		}
		if err := copyPrivateFile(sourcePath, destination); err != nil {
			_ = os.Remove(sourcePath)
			rollback()
			writeError(w, http.StatusInternalServerError, "No se pudo incorporar uno de los PDFs al expediente.")
			return
		}
		newTargets[relative] = true
		added = append(added, workspaceDocument{ID: strings.TrimSuffix(storedName, ".pdf"), StoredName: storedName, OriginalName: safeOriginalFilename(file.Filename), RelativePath: relative, Size: size})
	}
	for _, document := range added {
		path := filepath.ToSlash(document.RelativePath)
		delete(job.DeletedPDFs, path)
	}
	// Keep only the currently active source in the external-document manifest;
	// earlier versions remain untouched as private files in fuentes/externos.
	activeExternal := make(map[string]bool, len(added))
	for _, document := range added {
		activeExternal[filepath.ToSlash(document.RelativePath)] = true
	}
	keptExternal := job.ExternalPDFs[:0]
	for _, previous := range job.ExternalPDFs {
		if !activeExternal[filepath.ToSlash(previous.RelativePath)] {
			keptExternal = append(keptExternal, previous)
		}
	}
	job.ExternalPDFs = keptExternal
	job.ExternalPDFs = append(job.ExternalPDFs, added...)
	if err := s.saveStagedJob(job); err != nil {
		rollback()
		writeError(w, http.StatusInternalServerError, "Se cargaron los PDFs, pero no se pudo guardar su registro.")
		return
	}
	documents, patients, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Los PDFs se guardaron, pero no se pudo actualizar su lista.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": job.ID, "documents": documents, "patients": patients, "added": added, "added_count": len(added), "replaced_count": 0})
}
