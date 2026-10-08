package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const objectionSyncPDFLimit int64 = 256 << 20

type objectionSourcePDF struct {
	TargetPath string `json:"destino"`
	SHA256     string `json:"sha256"`
}

type objectionSyncConflict struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Name       string `json:"name"`
	Reason     string `json:"reason"`
}

type objectionSourcePDFFile struct {
	SourcePath        string
	SourceFile        string
	TargetPath        string
	CurrentTargetPath string
	PlanillaID        int64
	Size              int64
	SHA256            string
}

func cloneObjectionSourcePDFs(source map[string]objectionSourcePDF) map[string]objectionSourcePDF {
	copy := make(map[string]objectionSourcePDF, len(source))
	for path, document := range source {
		copy[path] = document
	}
	return copy
}

func objectionSourceDocumentHash(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > objectionSyncPDFLimit {
		return "", 0, errors.New("PDF de origen inválido o demasiado grande")
	}
	header := make([]byte, 5)
	if _, err := io.ReadFull(file, header); err != nil || string(header) != "%PDF-" {
		return "", 0, errors.New("el archivo de origen no contiene una firma PDF válida")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func objectionTargetRelativePath(sourceRelative, sourceFolder, targetFolder string) (string, error) {
	cleanSource := normalizedWorkspaceRelativePath(sourceRelative)
	prefix := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", sourceFolder)) + "/"
	if !strings.HasPrefix(cleanSource, prefix) {
		return "", errors.New("el PDF no pertenece a la carpeta del trámite de origen")
	}
	relative := strings.TrimPrefix(cleanSource, prefix)
	if relative == "" || relative == "." {
		return "", errors.New("la ruta del PDF de origen no es válida")
	}
	if strings.HasPrefix(strings.ToUpper(filepath.Base(relative)), "C_COBERTURA") && strings.EqualFold(filepath.Ext(relative), ".pdf") {
		relative = "C_COBERTURA.pdf"
	}
	return filepath.ToSlash(filepath.Join("4. EXPEDIENTES", targetFolder, filepath.FromSlash(relative))), nil
}

func objectionSourceIdentityPath(source stagedJob, currentPath string) string {
	currentPath = normalizedWorkspaceRelativePath(currentPath)
	for original, renamed := range source.Renames {
		if normalizedWorkspaceRelativePath(renamed) == currentPath {
			return normalizedWorkspaceRelativePath(original)
		}
	}
	return currentPath
}

func objectionWorkspaceTargetPath(job stagedJob, initialPath string) string {
	current := normalizedWorkspaceRelativePath(initialPath)
	seen := make(map[string]bool)
	for current != "" && !seen[current] {
		seen[current] = true
		next, ok := job.Renames[current]
		if !ok {
			break
		}
		current = normalizedWorkspaceRelativePath(next)
	}
	return current
}

func objectionDefaultTargetPath(job stagedJob, file objectionSourcePDFFile, packageRoot string) string {
	initial := objectionWorkspaceTargetPath(job, file.TargetPath)
	current := objectionWorkspaceTargetPath(job, file.CurrentTargetPath)
	if current == "" || initial == current || job.DeletedPDFs[initial] {
		return initial
	}
	path, err := safeWorkspacePath(packageRoot, initial)
	if err != nil {
		return initial
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return initial
	}
	return current
}

func collectObjectionSourcePDFs(packageRoot string, source stagedJob, rows []objectionRecord) ([]objectionSourcePDFFile, error) {
	files := make([]objectionSourcePDFFile, 0)
	seen := make(map[string]bool)
	for _, row := range rows {
		if !row.Matched || row.PlanillaID <= 0 {
			continue
		}
		sourceFolder := row.SourcePatientFolder
		if sourceFolder == "" {
			sourceFolder = normalizePatientFolder(row.Patient)
		}
		patientRoot, err := safeWorkspacePath(packageRoot, filepath.ToSlash(filepath.Join("4. EXPEDIENTES", sourceFolder)))
		if err != nil {
			return nil, err
		}
		walkErr := filepath.WalkDir(patientRoot, func(current string, entry os.DirEntry, entryErr error) error {
			if entryErr != nil {
				if errors.Is(entryErr, os.ErrNotExist) && current == patientRoot {
					return filepath.SkipDir
				}
				return entryErr
			}
			if current == patientRoot {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink no permitido en los documentos del primer ingreso")
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			relative, err := filepath.Rel(packageRoot, current)
			if err != nil {
				return err
			}
			sourceRelative := filepath.ToSlash(relative)
			if planillaForPath(source, sourceRelative) != row.PlanillaID {
				return nil
			}
			sourceIdentity := objectionSourceIdentityPath(source, sourceRelative)
			targetRelative, err := objectionTargetRelativePath(sourceIdentity, sourceFolder, row.PatientFolder)
			if err != nil {
				return err
			}
			currentTargetRelative, err := objectionTargetRelativePath(sourceRelative, sourceFolder, row.PatientFolder)
			if err != nil {
				return err
			}
			hash, size, err := objectionSourceDocumentHash(current)
			if err != nil {
				return fmt.Errorf("no se pudo revisar %s: %w", entry.Name(), err)
			}
			if seen[sourceIdentity] {
				return nil
			}
			seen[sourceIdentity] = true
			files = append(files, objectionSourcePDFFile{
				SourcePath:        sourceIdentity,
				SourceFile:        current,
				TargetPath:        targetRelative,
				CurrentTargetPath: currentTargetRelative,
				PlanillaID:        row.PlanillaID,
				Size:              size,
				SHA256:            hash,
			})
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("no se pudieron revisar los PDFs del trámite %s: %w", row.Tramite, walkErr)
		}
	}
	return files, nil
}

func (s *server) syncObjectionDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/objeciones/sincronizar/")
	if !validJobID(id) {
		writeError(w, http.StatusBadRequest, "El identificador del espacio no es válido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input struct {
		SourcePath string `json:"source_path"`
		Update     bool   `json:"actualizar_version"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "No se pudo leer la acción de sincronización.")
			return
		}
	}
	if input.Update != (input.SourcePath != "") {
		writeError(w, http.StatusBadRequest, "Para actualizar una versión indica el PDF de origen y confirma la actualización.")
		return
	}
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()
	job, err := s.loadStagedJob(id)
	if err != nil || !editableObjectionWorkspace(job) {
		writeError(w, http.StatusNotFound, "No se encontró el espacio de objeciones disponible para sincronizar.")
		return
	}
	source, err := s.loadStagedJob(job.ObjectionSourceID)
	if err != nil || source.IsObjections || (source.Status != "PROCESSED" && source.Status != "INCOMPLETE") ||
		!sameObjectionPeriod(source.Service, source.Month, source.Year, job.Service, job.Month, job.Year) {
		writeError(w, http.StatusConflict, "No se encontró el período de primer ingreso asociado a este espacio.")
		return
	}
	sourcePackageRoot := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source))
	files, err := collectObjectionSourcePDFs(sourcePackageRoot, source, job.ObjectionRows)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if input.Update {
		updated, err := s.updateObjectionSourcePDF(&job, files, input.SourcePath)
		if err != nil {
			status := http.StatusUnprocessableEntity
			if errors.Is(err, errObjectionSourceVersionNotPending) {
				status = http.StatusConflict
			}
			writeError(w, status, err.Error())
			return
		}
		result := jobResponse(s, job, filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job)), nil)
		result["documentos_actualizados"] = updated
		result["documentos_agregados"] = 0
		result["conflictos"] = []objectionSyncConflict{}
		result["message"] = "Se actualizó el PDF desde el primer ingreso. La versión anterior quedó guardada como respaldo; la postura y las selecciones se conservaron."
		writeJSON(w, http.StatusOK, result)
		return
	}
	added, conflicts, err := s.importMissingObjectionSourcePDFs(&job, files)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := jobResponse(s, job, filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job)), nil)
	result["documentos_agregados"] = added
	result["documentos_actualizados"] = 0
	result["conflictos"] = conflicts
	switch {
	case added > 0 && len(conflicts) > 0:
		result["message"] = fmt.Sprintf("Se agregaron %d PDF nuevos. %d documento(s) cambiaron o tienen otro contenido; se conservaron en Objeciones hasta que elijas si quieres actualizarlos.", added, len(conflicts))
	case added > 0:
		result["message"] = fmt.Sprintf("Se agregaron %d PDF nuevos desde el primer ingreso. Las posturas, anexos y selecciones existentes se conservaron.", added)
	case len(conflicts) > 0:
		result["message"] = fmt.Sprintf("No había PDFs nuevos. %d documento(s) cambiaron o tienen otro contenido; se conservaron en Objeciones hasta que elijas si quieres actualizarlos.", len(conflicts))
	default:
		result["message"] = "Los documentos de Objeciones ya están al día con el primer ingreso. No se modificaron posturas, anexos ni selecciones."
	}
	writeJSON(w, http.StatusOK, result)
}

var errObjectionSourceVersionNotPending = errors.New("la versión del primer ingreso ya no requiere actualización")

func (s *server) importMissingObjectionSourcePDFs(job *stagedJob, files []objectionSourcePDFFile) (int, []objectionSyncConflict, error) {
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(job))
	previousMappings := clonePlanillaMappings(job.DocumentPlanillas)
	previousSourcePDFs := cloneObjectionSourcePDFs(job.ObjectionSourcePDFs)
	previousDeleted := cloneDeletedPDFs(job.DeletedPDFs)
	addedPaths := make([]string, 0)
	conflicts := make([]objectionSyncConflict, 0)
	added := 0
	changed := false
	rollback := func() {
		for _, relative := range addedPaths {
			if path, err := safeWorkspacePath(packageRoot, relative); err == nil {
				_ = os.Remove(path)
			}
		}
		job.DocumentPlanillas = previousMappings
		job.ObjectionSourcePDFs = previousSourcePDFs
		job.DeletedPDFs = previousDeleted
	}
	for _, file := range files {
		tracked, hasBaseline := job.ObjectionSourcePDFs[file.SourcePath]
		targetRelative := objectionDefaultTargetPath(*job, file, packageRoot)
		if hasBaseline {
			targetRelative = normalizedWorkspaceRelativePath(tracked.TargetPath)
		}
		target, err := safeWorkspacePath(packageRoot, targetRelative)
		if err != nil {
			rollback()
			return 0, nil, errors.New("la ruta de un PDF de Objeciones no es válida")
		}
		info, statErr := os.Lstat(target)
		if errors.Is(statErr, os.ErrNotExist) {
			if job.DeletedPDFs[targetRelative] {
				if hasBaseline && tracked.SHA256 != file.SHA256 {
					conflicts = append(conflicts, objectionSyncConflict{
						SourcePath: file.SourcePath, TargetPath: targetRelative, Name: filepath.Base(targetRelative),
						Reason: "El PDF fue quitado de Objeciones y cambió en el primer ingreso; puedes decidir si incorporas esta versión.",
					})
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				rollback()
				return 0, nil, errors.New("no se pudo preparar la carpeta del PDF sincronizado")
			}
			if err := copyReplacePrivateFile(file.SourceFile, target); err != nil {
				rollback()
				return 0, nil, errors.New("no se pudo copiar un PDF nuevo desde el primer ingreso")
			}
			addedPaths = append(addedPaths, targetRelative)
			added++
			changed = true
			job.ObjectionSourcePDFs = ensureObjectionSourcePDFMap(job.ObjectionSourcePDFs)
			job.ObjectionSourcePDFs[file.SourcePath] = objectionSourcePDF{TargetPath: targetRelative, SHA256: file.SHA256}
			job.DocumentPlanillas = ensurePlanillaMappingMap(job.DocumentPlanillas)
			if job.DocumentPlanillas[targetRelative] != file.PlanillaID {
				job.DocumentPlanillas[targetRelative] = file.PlanillaID
				changed = true
			}
			delete(job.DeletedPDFs, targetRelative)
			continue
		}
		if statErr != nil || !info.Mode().IsRegular() {
			rollback()
			return 0, nil, errors.New("no se pudo revisar un PDF existente en Objeciones")
		}
		targetHash, _, err := objectionSourceDocumentHash(target)
		if err != nil {
			rollback()
			return 0, nil, errors.New("el PDF existente en Objeciones no se pudo comparar de forma segura")
		}
		if hasBaseline && tracked.SHA256 != file.SHA256 {
			reason := "El PDF del primer ingreso cambió después de crear Objeciones."
			if targetHash != tracked.SHA256 {
				reason = "El PDF cambió tanto en el primer ingreso como en Objeciones; se conservaron ambas versiones."
			}
			conflicts = append(conflicts, objectionSyncConflict{SourcePath: file.SourcePath, TargetPath: targetRelative, Name: filepath.Base(targetRelative), Reason: reason})
			continue
		}
		if !hasBaseline && targetHash != file.SHA256 {
			conflicts = append(conflicts, objectionSyncConflict{
				SourcePath: file.SourcePath, TargetPath: targetRelative, Name: filepath.Base(targetRelative),
				Reason: "Ya hay un PDF con ese nombre y otro contenido en Objeciones; se conservó.",
			})
			continue
		}
		if !hasBaseline {
			changed = true
			job.ObjectionSourcePDFs = ensureObjectionSourcePDFMap(job.ObjectionSourcePDFs)
			job.ObjectionSourcePDFs[file.SourcePath] = objectionSourcePDF{TargetPath: targetRelative, SHA256: file.SHA256}
		}
		job.DocumentPlanillas = ensurePlanillaMappingMap(job.DocumentPlanillas)
		if job.DocumentPlanillas[targetRelative] != file.PlanillaID {
			job.DocumentPlanillas[targetRelative] = file.PlanillaID
			changed = true
		}
		if job.DeletedPDFs[targetRelative] {
			delete(job.DeletedPDFs, targetRelative)
			changed = true
		}
	}
	if !changed && added == 0 {
		return 0, conflicts, nil
	}
	if err := s.saveStagedJob(*job); err != nil {
		rollback()
		return 0, nil, errors.New("no se pudo guardar la sincronización; se retiraron los PDFs nuevos")
	}
	return added, conflicts, nil
}

func (s *server) updateObjectionSourcePDF(job *stagedJob, files []objectionSourcePDFFile, sourcePath string) (int, error) {
	var sourceFile *objectionSourcePDFFile
	for index := range files {
		if files[index].SourcePath == normalizedWorkspaceRelativePath(sourcePath) {
			sourceFile = &files[index]
			break
		}
	}
	if sourceFile == nil {
		return 0, errors.New("el PDF ya no está disponible en el expediente de origen")
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(job))
	tracked, hasBaseline := job.ObjectionSourcePDFs[sourceFile.SourcePath]
	targetRelative := objectionDefaultTargetPath(*job, *sourceFile, packageRoot)
	if hasBaseline {
		targetRelative = normalizedWorkspaceRelativePath(tracked.TargetPath)
	}
	target, err := safeWorkspacePath(packageRoot, targetRelative)
	if err != nil {
		return 0, errors.New("la ruta del PDF de Objeciones no es válida")
	}
	info, statErr := os.Lstat(target)
	targetExists := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return 0, errors.New("no se pudo revisar la copia actual del PDF")
	}
	if targetExists && !info.Mode().IsRegular() {
		return 0, errors.New("la copia actual del PDF no es un archivo regular")
	}
	var previous []byte
	if targetExists {
		targetHash, _, err := objectionSourceDocumentHash(target)
		if err != nil {
			return 0, errors.New("la copia actual del PDF no se pudo comparar de forma segura")
		}
		if targetHash == sourceFile.SHA256 {
			return 0, errObjectionSourceVersionNotPending
		}
		if hasBaseline && sourceFile.SHA256 == tracked.SHA256 {
			return 0, errObjectionSourceVersionNotPending
		}
		previous, err = os.ReadFile(target)
		if err != nil || int64(len(previous)) > objectionSyncPDFLimit {
			return 0, errors.New("no se pudo conservar la versión actual de Objeciones")
		}
		if err := saveObjectionVersion(s.jobRoot(job.ID), filepath.Base(targetRelative), previous); err != nil {
			return 0, errors.New("no se pudo guardar una copia de respaldo antes de actualizar el PDF")
		}
	}
	if err := copyReplacePrivateFile(sourceFile.SourceFile, target); err != nil {
		return 0, errors.New("no se pudo actualizar el PDF desde el primer ingreso")
	}
	previousMappings := clonePlanillaMappings(job.DocumentPlanillas)
	previousSourcePDFs := cloneObjectionSourcePDFs(job.ObjectionSourcePDFs)
	previousDeleted := cloneDeletedPDFs(job.DeletedPDFs)
	job.DocumentPlanillas = ensurePlanillaMappingMap(job.DocumentPlanillas)
	job.DocumentPlanillas[targetRelative] = sourceFile.PlanillaID
	job.ObjectionSourcePDFs = ensureObjectionSourcePDFMap(job.ObjectionSourcePDFs)
	job.ObjectionSourcePDFs[sourceFile.SourcePath] = objectionSourcePDF{TargetPath: targetRelative, SHA256: sourceFile.SHA256}
	delete(job.DeletedPDFs, targetRelative)
	if err := s.saveStagedJob(*job); err != nil {
		if targetExists {
			_ = atomicWritePrivateFile(target, previous)
		} else {
			_ = os.Remove(target)
		}
		job.DocumentPlanillas = previousMappings
		job.ObjectionSourcePDFs = previousSourcePDFs
		job.DeletedPDFs = previousDeleted
		return 0, errors.New("no se pudo registrar el cambio; se restauró la versión anterior")
	}
	return 1, nil
}

func ensureObjectionSourcePDFMap(source map[string]objectionSourcePDF) map[string]objectionSourcePDF {
	if source == nil {
		return make(map[string]objectionSourcePDF)
	}
	return source
}

func clonePlanillaMappings(source map[string]int64) map[string]int64 {
	copy := make(map[string]int64, len(source))
	for path, id := range source {
		copy[path] = id
	}
	return copy
}

func ensurePlanillaMappingMap(source map[string]int64) map[string]int64 {
	if source == nil {
		return make(map[string]int64)
	}
	return source
}

func cloneDeletedPDFs(source map[string]bool) map[string]bool {
	copy := make(map[string]bool, len(source))
	for path, deleted := range source {
		copy[path] = deleted
	}
	return copy
}
