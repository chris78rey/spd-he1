package main

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const historicalArchiveUploadField = "zip_file"

var historicalArchiveIDPattern = regexp.MustCompile(`^HIST-[a-f0-9]{32}$`)

type historicalArchiveRecord struct {
	ID           string    `json:"id"`
	Service      string    `json:"tipo_servicio"`
	Month        string    `json:"mes"`
	Year         string    `json:"anio"`
	OriginalName string    `json:"nombre_original"`
	Size         int64     `json:"size_bytes"`
	SHA256       string    `json:"sha256"`
	UploadedBy   string    `json:"subido_por"`
	UploadedAt   time.Time `json:"subido_en"`
}

func (s *server) historicalArchives(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.listHistoricalArchives(w, r)
	case http.MethodPost:
		s.uploadHistoricalArchive(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
	}
}

func (s *server) listHistoricalArchives(w http.ResponseWriter, r *http.Request) {
	records, err := s.readHistoricalArchiveRecords()
	if err != nil {
		log.Printf("No se pudo leer el archivo histórico (%T): %v", err, err)
		writeError(w, http.StatusInternalServerError, "No se pudo cargar el archivo histórico.")
		return
	}

	serviceFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("servicio")))
	monthFilter := strings.TrimSpace(r.URL.Query().Get("mes"))
	yearFilter := strings.TrimSpace(r.URL.Query().Get("anio"))
	queryTerms := strings.Fields(normalizeHistoricalArchiveSearch(r.URL.Query().Get("q")))
	filtered := make([]historicalArchiveRecord, 0, len(records))
	for _, record := range records {
		if serviceFilter != "" && serviceFilter != "ALL" && record.Service != serviceFilter {
			continue
		}
		if monthFilter != "" && monthFilter != "ALL" && record.Month != monthFilter {
			continue
		}
		if yearFilter != "" && yearFilter != "ALL" && record.Year != yearFilter {
			continue
		}
		searchable := normalizeHistoricalArchiveSearch(strings.Join([]string{
			record.ID, record.Service, historicalArchiveServiceLabel(record.Service),
			record.Month, historicalArchiveMonthLabel(record.Month), record.Year,
			record.OriginalName, record.UploadedBy,
		}, " "))
		match := true
		for _, term := range queryTerms {
			if !strings.Contains(searchable, term) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, record)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"archives": filtered})
}

func (s *server) uploadHistoricalArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	user, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	limit := s.maxIngest
	if limit <= 0 {
		limit = 2 << 30
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))
	parseErr := r.ParseMultipartForm(32 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if parseErr != nil {
		var maxErr *http.MaxBytesError
		if errors.As(parseErr, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "El ZIP supera el tamaño máximo permitido.")
			return
		}
		writeError(w, http.StatusBadRequest, "No se pudo leer la carga del ZIP.")
		return
	}

	service := strings.ToUpper(strings.TrimSpace(r.FormValue("tipo_servicio")))
	monthValue := strings.TrimSpace(r.FormValue("mes"))
	monthNumber, monthErr := strconv.Atoi(monthValue)
	year := strings.TrimSpace(r.FormValue("anio"))
	yearNumber, yearErr := strconv.Atoi(year)
	if !validServiceCode(service) {
		writeError(w, http.StatusBadRequest, "Selecciona un servicio válido.")
		return
	}
	if monthErr != nil || monthNumber < 1 || monthNumber > 12 {
		writeError(w, http.StatusBadRequest, "Selecciona un mes válido.")
		return
	}
	if yearErr != nil || yearNumber < 1900 || yearNumber > 9999 || len(year) != 4 {
		writeError(w, http.StatusBadRequest, "Escribe un año de cuatro dígitos válido.")
		return
	}
	files := r.MultipartForm.File[historicalArchiveUploadField]
	if len(files) != 1 || files[0].Size <= 0 || files[0].Size > limit || !strings.EqualFold(filepath.Ext(files[0].Filename), ".zip") {
		writeError(w, http.StatusBadRequest, "Selecciona un archivo ZIP no vacío dentro del tamaño permitido.")
		return
	}

	if err := os.MkdirAll(s.historicalArchivesDir, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el repositorio histórico.")
		return
	}
	temporaryDir, err := os.MkdirTemp(s.historicalArchivesDir, ".carga-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo preparar el almacenamiento del ZIP.")
		return
	}
	defer os.RemoveAll(temporaryDir)

	temporaryZIP := filepath.Join(temporaryDir, "original.zip")
	file, err := files[0].Open()
	if err != nil {
		writeError(w, http.StatusBadRequest, "No se pudo abrir el ZIP cargado.")
		return
	}
	output, err := os.OpenFile(temporaryZIP, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = file.Close()
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el ZIP cargado.")
		return
	}
	hasher := sha256.New()
	storedBytes, copyErr := io.Copy(io.MultiWriter(output, hasher), io.LimitReader(file, limit+1))
	fileCloseErr := file.Close()
	syncErr := output.Sync()
	outputCloseErr := output.Close()
	if copyErr != nil || fileCloseErr != nil || syncErr != nil || outputCloseErr != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo guardar el ZIP completo.")
		return
	}
	if storedBytes == 0 || storedBytes > limit || storedBytes != files[0].Size {
		writeError(w, http.StatusBadRequest, "El tamaño del ZIP cargado no es válido.")
		return
	}
	archive, err := zip.OpenReader(temporaryZIP)
	if err != nil {
		writeError(w, http.StatusBadRequest, "El archivo seleccionado no es un ZIP válido.")
		return
	}
	entryCount := len(archive.File)
	_ = archive.Close()
	if entryCount == 0 {
		writeError(w, http.StatusBadRequest, "El ZIP no contiene archivos.")
		return
	}

	id, err := newHistoricalArchiveID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo asignar un identificador al ZIP.")
		return
	}
	record := historicalArchiveRecord{
		ID: id, Service: service, Month: fmt.Sprintf("%02d", monthNumber), Year: year,
		OriginalName: safeOriginalFilename(files[0].Filename), Size: storedBytes,
		SHA256: hex.EncodeToString(hasher.Sum(nil)), UploadedBy: user.username, UploadedAt: time.Now().UTC(),
	}
	metadata, err := json.Marshal(record)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la información del ZIP.")
		return
	}
	metadataFile, err := os.OpenFile(filepath.Join(temporaryDir, "metadata.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la información del ZIP.")
		return
	}
	_, metadataErr := metadataFile.Write(metadata)
	metadataSyncErr := metadataFile.Sync()
	metadataCloseErr := metadataFile.Close()
	if metadataErr != nil || metadataSyncErr != nil || metadataCloseErr != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo registrar la información del ZIP.")
		return
	}
	if err := os.Rename(temporaryDir, filepath.Join(s.historicalArchivesDir, id)); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo conservar el ZIP en el archivo histórico.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"archive": record})
}

func (s *server) downloadHistoricalArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if _, ok := s.getSession(r); !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/archivo-historico/descargar/")
	if !historicalArchiveIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "El identificador del archivo histórico no es válido.")
		return
	}
	record, err := s.readHistoricalArchiveRecord(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "No se encontró el ZIP histórico.")
			return
		}
		log.Printf("No se pudo leer ZIP histórico %s (%T): %v", id, err, err)
		writeError(w, http.StatusInternalServerError, "No se pudo abrir el ZIP histórico.")
		return
	}
	zipPath := filepath.Join(s.historicalArchivesDir, id, "original.zip")
	file, err := os.Open(zipPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "No se encontró el ZIP histórico.")
			return
		}
		writeError(w, http.StatusInternalServerError, "No se pudo abrir el ZIP histórico.")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo verificar el ZIP histórico.")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": record.OriginalName}))
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("No se pudo completar la descarga del ZIP histórico %s: %v", id, err)
	}
}

func (s *server) readHistoricalArchiveRecords() ([]historicalArchiveRecord, error) {
	entries, err := os.ReadDir(s.historicalArchivesDir)
	if errors.Is(err, os.ErrNotExist) {
		return []historicalArchiveRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	records := make([]historicalArchiveRecord, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !historicalArchiveIDPattern.MatchString(entry.Name()) {
			continue
		}
		record, err := s.readHistoricalArchiveRecord(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("registro %s: %w", entry.Name(), err)
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Year != records[j].Year {
			return records[i].Year > records[j].Year
		}
		if records[i].Month != records[j].Month {
			return records[i].Month > records[j].Month
		}
		if !records[i].UploadedAt.Equal(records[j].UploadedAt) {
			return records[i].UploadedAt.After(records[j].UploadedAt)
		}
		return records[i].ID > records[j].ID
	})
	return records, nil
}

func (s *server) readHistoricalArchiveRecord(id string) (historicalArchiveRecord, error) {
	if !historicalArchiveIDPattern.MatchString(id) {
		return historicalArchiveRecord{}, os.ErrNotExist
	}
	contents, err := os.ReadFile(filepath.Join(s.historicalArchivesDir, id, "metadata.json"))
	if err != nil {
		return historicalArchiveRecord{}, err
	}
	var record historicalArchiveRecord
	if err := json.Unmarshal(contents, &record); err != nil {
		return historicalArchiveRecord{}, err
	}
	if record.ID != id || !validServiceCode(record.Service) || len(record.Month) != 2 || len(record.Year) != 4 || record.OriginalName == "" {
		return historicalArchiveRecord{}, fmt.Errorf("metadatos inválidos")
	}
	return record, nil
}

func newHistoricalArchiveID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "HIST-" + hex.EncodeToString(random[:]), nil
}

func normalizeHistoricalArchiveSearch(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
	).Replace(value)
}

func historicalArchiveServiceLabel(service string) string {
	switch service {
	case "AMBULATORIO":
		return "Ambulatorio"
	case "EMERGENCIA":
		return "Emergencia"
	case "HOSPITALIZACION":
		return "Hospitalización"
	default:
		return service
	}
}

func historicalArchiveMonthLabel(month string) string {
	months := [...]string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
	value, err := strconv.Atoi(month)
	if err != nil || value < 1 || value > 12 {
		return month
	}
	return months[value]
}
