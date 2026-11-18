package main

// Reemplazo de hojas de cobertura generadas localmente por el PDF real del portal MSP (fase B).
//
// La fase A (scripts/cobertura/reemplazo_coberturas.cjs) descarga del portal cada página, arma
// la hoja candidata y deja una solicitud JSON en <data>/reemplazo_coberturas/solicitudes/.
// El servidor procesa esas solicitudes dentro de su propio proceso, con s.ingestMu tomado, para
// no pisarse con usuarios que trabajen en el mismo período. Por cada hoja:
//   - verifica que el PDF activo no cambió desde la preparación (SHA-256);
//   - valida por separado cada página descargada del portal (SHA-256, una página, contiene su
//     cédula) y las fusiona en el mismo orden con pdfcpu, como la fusión de la aplicación;
//   - guarda la hoja anterior en fuentes/reemplazo-coberturas/<marca>/ como respaldo de auditoría;
//   - guarda el candidato en fuentes/externos/, lo coloca en el expediente y actualiza
//     replacements y external_pdfs como lo hace la fusión;
// y al final guarda el período y sincroniza Oracle. El resultado queda en resultados/.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

var (
	coverageReplacementRequestName = regexp.MustCompile(`^[A-Za-z0-9_.-]+\.json$`)
	coverageReplacementSheetName   = regexp.MustCompile(`^C_COBERTURA(_\d+)?\.pdf$`)
)

type coverageReplacementPage struct {
	File   string `json:"pdf"`
	SHA256 string `json:"sha256"`
	Cedula string `json:"cedula"`
}

type coverageReplacementSheet struct {
	Path           string                    `json:"ruta"`
	OriginalSHA256 string                    `json:"sha256_original"`
	OriginalPages  int                       `json:"paginas_originales"`
	Pages          []coverageReplacementPage `json:"paginas"`
}

type coverageReplacementRequest struct {
	JobID    string                     `json:"job_id"`
	Prepared string                     `json:"preparado"`
	Sheets   []coverageReplacementSheet `json:"hojas"`
}

type coverageReplacementSheetResult struct {
	Path   string `json:"ruta"`
	Status string `json:"estado"`
	Detail string `json:"detalle,omitempty"`
	Backup string `json:"respaldo,omitempty"`
	Stored string `json:"almacenado,omitempty"`
}

type coverageReplacementResult struct {
	JobID       string                           `json:"job_id"`
	ProcessedAt string                           `json:"procesado"`
	Error       string                           `json:"error,omitempty"`
	OracleSync  string                           `json:"oracle"`
	Sheets      []coverageReplacementSheetResult `json:"hojas"`
}

func (s *server) coverageReplacementRoot() string {
	return filepath.Join(s.dataDir, "reemplazo_coberturas")
}

// watchCoverageReplacements revisa la cola de solicitudes de reemplazo cada pocos segundos.
func (s *server) watchCoverageReplacements() {
	requests := filepath.Join(s.coverageReplacementRoot(), "solicitudes")
	for {
		time.Sleep(5 * time.Second)
		entries, err := os.ReadDir(requests)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.Type().IsRegular() && coverageReplacementRequestName.MatchString(entry.Name()) {
				names = append(names, entry.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			s.processCoverageReplacementRequest(filepath.Join(requests, name))
		}
	}
}

func (s *server) processCoverageReplacementRequest(requestPath string) {
	name := filepath.Base(requestPath)
	result := coverageReplacementResult{ProcessedAt: time.Now().UTC().Format(time.RFC3339), OracleSync: "no ejecutada"}
	raw, err := os.ReadFile(requestPath)
	var request coverageReplacementRequest
	if err == nil {
		err = json.Unmarshal(raw, &request)
	}
	if err != nil {
		result.Error = "Solicitud ilegible."
	} else {
		result.JobID = request.JobID
		result = s.applyCoverageReplacement(request, result)
	}
	resultsDir := filepath.Join(s.coverageReplacementRoot(), "resultados")
	if mkErr := os.MkdirAll(resultsDir, 0700); mkErr != nil || writePrivateJSON(filepath.Join(resultsDir, name), result) != nil {
		log.Printf("reemplazo coberturas %s: no se pudo escribir el resultado", name)
		return // se reintenta en la siguiente vuelta
	}
	_ = os.Remove(requestPath)
	replaced := 0
	for _, sheet := range result.Sheets {
		if sheet.Status == "REEMPLAZADA" {
			replaced++
		}
	}
	log.Printf("reemplazo coberturas %s: %d de %d hojas reemplazadas, oracle=%s%s", request.JobID, replaced, len(result.Sheets), result.OracleSync, map[bool]string{true: ", error=" + result.Error, false: ""}[result.Error != ""])
}

func (s *server) applyCoverageReplacement(request coverageReplacementRequest, result coverageReplacementResult) coverageReplacementResult {
	preparedRoot, err := canonicalPath(request.Prepared)
	replacementRoot, rootErr := canonicalPath(s.coverageReplacementRoot())
	if err != nil || rootErr != nil || !strings.HasPrefix(preparedRoot, replacementRoot+string(os.PathSeparator)) {
		result.Error = "La carpeta preparada debe estar dentro de reemplazo_coberturas."
		return result
	}

	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()

	job, err := s.loadStagedJob(request.JobID)
	if err != nil || job.ID != request.JobID {
		result.Error = "No se encontró el período indicado."
		return result
	}
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	sourceRoot := s.jobSourcesDir(job.ID)
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	historyDir := filepath.Join(sourceRoot, "reemplazo-coberturas", stamp)
	externalRoot := filepath.Join(sourceRoot, "externos")
	if os.MkdirAll(historyDir, 0700) != nil || os.MkdirAll(externalRoot, 0700) != nil {
		result.Error = "No se pudo preparar el historial del reemplazo."
		return result
	}

	previousReplacements := make(map[string]string, len(job.Replacements))
	for path, stored := range job.Replacements {
		previousReplacements[path] = stored
	}
	previousExternal := append([]workspaceDocument(nil), job.ExternalPDFs...)
	type applied struct{ active, backup, stored string }
	done := make([]applied, 0, len(request.Sheets))
	restore := func() {
		for _, item := range done {
			_ = os.Remove(item.active)
			_ = os.Rename(item.backup, item.active)
			_ = os.Remove(item.stored)
		}
	}

	for index, sheet := range request.Sheets {
		entry := coverageReplacementSheetResult{Path: sheet.Path}
		active, stored, backup, archive, detail := s.replaceCoverageSheet(packageRoot, preparedRoot, historyDir, externalRoot, index, sheet)
		if detail != "" {
			entry.Status, entry.Detail = "OMITIDA", detail
			result.Sheets = append(result.Sheets, entry)
			continue
		}
		done = append(done, applied{active: active, backup: backup, stored: stored})
		relative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(sheet.Path)))
		storedName := filepath.Base(stored)
		if job.Replacements == nil {
			job.Replacements = make(map[string]string)
		}
		job.Replacements[relative] = storedName
		for i := range job.ExternalPDFs {
			if filepath.ToSlash(filepath.Clean(filepath.FromSlash(job.ExternalPDFs[i].RelativePath))) == relative {
				info, _ := os.Stat(active)
				job.ExternalPDFs[i].ID = strings.TrimSuffix(storedName, ".pdf")
				job.ExternalPDFs[i].StoredName = storedName
				if info != nil {
					job.ExternalPDFs[i].Size = info.Size()
				}
			}
		}
		entry.Status, entry.Backup, entry.Stored = "REEMPLAZADA", archive, storedName
		result.Sheets = append(result.Sheets, entry)
	}

	if len(done) == 0 {
		result.OracleSync = "no necesaria"
		return result
	}
	if err := s.saveStagedJob(job); err != nil {
		restore()
		job.Replacements, job.ExternalPDFs = previousReplacements, previousExternal
		for i := range result.Sheets {
			if result.Sheets[i].Status == "REEMPLAZADA" {
				result.Sheets[i].Status, result.Sheets[i].Detail = "REVERTIDA", "No se pudo guardar el período; se restauró la hoja anterior."
			}
		}
		result.Error = "No se pudo guardar el período."
		return result
	}
	for _, item := range done {
		_ = os.Remove(item.backup)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := s.syncOracleWorkspace(ctx, &job, packageRoot); err != nil {
		log.Printf("reemplazo coberturas %s: falló la sincronización Oracle: %v", job.ID, err)
		result.OracleSync = "falló: " + err.Error()
		return result
	}
	result.OracleSync = "ok"
	return result
}

// replaceCoverageSheet valida y coloca una hoja. Devuelve detail != "" si la hoja se omite sin cambios.
func (s *server) replaceCoverageSheet(packageRoot, preparedRoot, historyDir, externalRoot string, index int, sheet coverageReplacementSheet) (active, stored, backup, archive, detail string) {
	relative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(sheet.Path)))
	parts := strings.Split(relative, "/")
	if len(parts) != 3 || parts[0] != "4. EXPEDIENTES" || !coverageReplacementSheetName.MatchString(parts[2]) {
		return "", "", "", "", "La ruta no corresponde a una hoja C_COBERTURA de un expediente."
	}
	active, err := safeWorkspacePath(packageRoot, relative)
	if err != nil {
		return "", "", "", "", "Ruta de expediente inválida."
	}
	current, err := fileSHA256(active)
	if err != nil {
		return "", "", "", "", "La hoja ya no existe en el expediente."
	}
	if !strings.EqualFold(current, sheet.OriginalSHA256) {
		return "", "", "", "", "La hoja cambió desde la preparación; vuelve a preparar este período."
	}
	if len(sheet.Pages) == 0 || len(sheet.Pages) != sheet.OriginalPages {
		return "", "", "", "", "La cantidad de páginas del portal no coincide con la hoja original."
	}
	pages := make([]string, 0, len(sheet.Pages))
	for number, page := range sheet.Pages {
		file, err := canonicalPath(filepath.Join(preparedRoot, filepath.FromSlash(page.File)))
		if err != nil || !strings.HasPrefix(file, preparedRoot+string(os.PathSeparator)) {
			return "", "", "", "", fmt.Sprintf("Página %d fuera de la carpeta preparada.", number+1)
		}
		if sum, err := fileSHA256(file); err != nil || !strings.EqualFold(sum, page.SHA256) {
			return "", "", "", "", fmt.Sprintf("Página %d no coincide con la preparada.", number+1)
		}
		if err := validateStagedFile(file, ".pdf"); err != nil {
			return "", "", "", "", fmt.Sprintf("Página %d no es un PDF válido.", number+1)
		}
		if count, err := api.PageCountFile(file); err != nil || count != 1 {
			return "", "", "", "", fmt.Sprintf("Página %d no tiene exactamente una página.", number+1)
		}
		text, err := extractPDFText(file)
		compact := strings.Join(strings.Fields(normalizeOCRText(text)), "")
		if err != nil || !coverageCedulaPattern.MatchString(page.Cedula) || !strings.Contains(compact, page.Cedula) {
			return "", "", "", "", fmt.Sprintf("Página %d no contiene la cédula esperada.", number+1)
		}
		pages = append(pages, file)
	}

	archive = filepath.Join(historyDir, fmt.Sprintf("%02d-%s__%s", index+1, parts[1], parts[2]))
	if err := copyPrivateFile(active, archive); err != nil {
		return "", "", "", "", "No se pudo respaldar la hoja anterior."
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", "", "", "No se pudo identificar el PDF del portal."
	}
	stored = filepath.Join(externalRoot, fmt.Sprintf("cobertura-portal-%x.pdf", idBytes))
	if len(pages) == 1 {
		if err := copyPrivateFile(pages[0], stored); err != nil {
			return "", "", "", "", "No se pudo guardar el PDF del portal."
		}
	} else if err := api.MergeCreateFile(pages, stored, false, nil); err != nil {
		_ = os.Remove(stored)
		return "", "", "", "", "No se pudieron fusionar las páginas del portal."
	}
	if count, err := api.PageCountFile(stored); err != nil || count != len(pages) || validateStagedFile(stored, ".pdf") != nil {
		_ = os.Remove(stored)
		return "", "", "", "", "La hoja armada con el portal no es válida."
	}
	_ = os.Chmod(stored, 0600)
	backup = active + ".folio-reemplazo-backup"
	if err := os.Rename(active, backup); err != nil {
		_ = os.Remove(stored)
		return "", "", "", "", "No se pudo resguardar temporalmente la hoja anterior."
	}
	if err := copyPrivateFile(stored, active); err != nil {
		_ = os.Remove(active)
		_ = os.Rename(backup, active)
		_ = os.Remove(stored)
		return "", "", "", "", "No se pudo colocar el PDF del portal en el expediente."
	}
	return active, stored, backup, archive, ""
}
