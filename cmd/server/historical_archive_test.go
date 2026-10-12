package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func historicalArchiveTestServer(t *testing.T) *server {
	t.Helper()
	return &server{
		historicalArchivesDir: filepath.Join(t.TempDir(), "archivo-historico"),
		maxIngest:             1 << 20,
		sessions: map[string]session{
			"test-session": {username: "operador", expiresAt: time.Now().Add(time.Hour)},
		},
	}
}

func makeHistoricalTestZIP(t *testing.T, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("archivo/original.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return append([]byte(nil), buffer.Bytes()...)
}

func historicalUploadRequest(t *testing.T, zipBytes []byte, filename, service, month, year string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("tipo_servicio", service)
	_ = writer.WriteField("mes", month)
	_ = writer.WriteField("anio", year)
	file, err := writer.CreateFormFile(historicalArchiveUploadField, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(zipBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/archivo-historico", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	return request
}

func TestHistoricalArchiveUploadPreservesOriginalZIPAndCanSearchAndDownload(t *testing.T) {
	s := historicalArchiveTestServer(t)
	original := makeHistoricalTestZIP(t, "preserve these exact bytes")
	response := httptest.NewRecorder()
	s.historicalArchives(response, historicalUploadRequest(t, original, "EMGagosto2021.zip", "EMERGENCIA", "08", "2021"))
	if response.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, body = %s", response.Code, response.Body.String())
	}
	var uploaded struct {
		Archive historicalArchiveRecord `json:"archive"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	if uploaded.Archive.ID == "" || uploaded.Archive.OriginalName != "EMGagosto2021.zip" || uploaded.Archive.UploadedBy != "operador" {
		t.Fatalf("unexpected saved metadata: %+v", uploaded.Archive)
	}
	stored, err := os.ReadFile(filepath.Join(s.historicalArchivesDir, uploaded.Archive.ID, "original.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, original) {
		t.Fatal("stored ZIP bytes differ from the uploaded original")
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/archivo-historico?q="+url.QueryEscape("EMERGÉNCIA agosto 2021"), nil)
	listRequest.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	listResponse := httptest.NewRecorder()
	s.historicalArchives(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}
	var listed struct {
		Archives []historicalArchiveRecord `json:"archives"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Archives) != 1 || listed.Archives[0].ID != uploaded.Archive.ID {
		t.Fatalf("accent-insensitive search returned %+v", listed.Archives)
	}

	downloadRequest := httptest.NewRequest(http.MethodGet, "/api/v1/archivo-historico/descargar/"+uploaded.Archive.ID, nil)
	downloadRequest.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	downloadResponse := httptest.NewRecorder()
	s.downloadHistoricalArchive(downloadResponse, downloadRequest)
	if downloadResponse.Code != http.StatusOK {
		t.Fatalf("download status = %d, body = %s", downloadResponse.Code, downloadResponse.Body.String())
	}
	if !bytes.Equal(downloadResponse.Body.Bytes(), original) {
		t.Fatal("downloaded ZIP bytes differ from the uploaded original")
	}
	if !strings.Contains(downloadResponse.Header().Get("Content-Disposition"), "EMGagosto2021.zip") {
		t.Fatalf("unexpected download filename header: %q", downloadResponse.Header().Get("Content-Disposition"))
	}
}

func TestHistoricalArchiveRejectsDuplicateServicePeriod(t *testing.T) {
	s := historicalArchiveTestServer(t)
	original := makeHistoricalTestZIP(t, "first archive")
	first := httptest.NewRecorder()
	s.historicalArchives(first, historicalUploadRequest(t, original, "EMGagosto2021.zip", "EMERGENCIA", "08", "2021"))
	if first.Code != http.StatusCreated {
		t.Fatalf("first upload status = %d, body = %s", first.Code, first.Body.String())
	}

	duplicate := httptest.NewRecorder()
	s.historicalArchives(duplicate, historicalUploadRequest(t, makeHistoricalTestZIP(t, "duplicate archive"), "otro.zip", "EMERGENCIA", "8", "2021"))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate upload status = %d, want %d, body = %s", duplicate.Code, http.StatusConflict, duplicate.Body.String())
	}
	var conflict struct {
		Error    string                    `json:"error"`
		Archives []historicalArchiveRecord `json:"archives"`
	}
	if err := json.Unmarshal(duplicate.Body.Bytes(), &conflict); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(conflict.Error, "Emergencia · Agosto 2021") || len(conflict.Archives) != 1 {
		t.Fatalf("duplicate response did not explain the existing period: %+v", conflict)
	}

	stored, err := os.ReadFile(filepath.Join(s.historicalArchivesDir, conflict.Archives[0].ID, "original.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, original) {
		t.Fatal("rejecting a duplicate changed the original ZIP")
	}

	// Each part of the service/month/year key remains independent.
	for _, period := range []struct {
		service string
		month   string
		year    string
	}{
		{service: "EMERGENCIA", month: "09", year: "2021"},
		{service: "AMBULATORIO", month: "08", year: "2021"},
		{service: "EMERGENCIA", month: "08", year: "2022"},
	} {
		response := httptest.NewRecorder()
		s.historicalArchives(response, historicalUploadRequest(t, makeHistoricalTestZIP(t, "different period"), "periodo.zip", period.service, period.month, period.year))
		if response.Code != http.StatusCreated {
			t.Fatalf("distinct period %s/%s/%s upload status = %d, body = %s", period.service, period.month, period.year, response.Code, response.Body.String())
		}
	}

	entries, err := os.ReadDir(s.historicalArchivesDir)
	if err != nil {
		t.Fatal(err)
	}
	archiveCount := 0
	for _, entry := range entries {
		if entry.IsDir() && historicalArchiveIDPattern.MatchString(entry.Name()) {
			archiveCount++
		}
	}
	if archiveCount != 4 {
		t.Fatalf("saved archive count = %d, want 4 (one for each distinct period)", archiveCount)
	}
}

func TestConcurrentHistoricalArchiveUploadsKeepOnePerServicePeriod(t *testing.T) {
	s := historicalArchiveTestServer(t)
	statuses := make(chan int, 2)
	var wait sync.WaitGroup
	for _, filename := range []string{"primero.zip", "segundo.zip"} {
		filename := filename
		wait.Add(1)
		go func() {
			defer wait.Done()
			response := httptest.NewRecorder()
			s.historicalArchives(response, historicalUploadRequest(t, makeHistoricalTestZIP(t, filename), filename, "EMERGENCIA", "08", "2021"))
			statuses <- response.Code
		}()
	}
	wait.Wait()
	close(statuses)
	created, conflicts := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("concurrent upload status = %d, want 201 or 409", status)
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("concurrent upload outcomes = %d created, %d conflicts; want 1 each", created, conflicts)
	}
	entries, err := os.ReadDir(s.historicalArchivesDir)
	if err != nil {
		t.Fatal(err)
	}
	archiveCount := 0
	for _, entry := range entries {
		if entry.IsDir() && historicalArchiveIDPattern.MatchString(entry.Name()) {
			archiveCount++
		}
	}
	if archiveCount != 1 {
		t.Fatalf("saved archive count after concurrent uploads = %d, want 1", archiveCount)
	}
}

func TestHistoricalArchiveRequiresSessionAndRejectsInvalidZIP(t *testing.T) {
	s := historicalArchiveTestServer(t)
	unauthenticatedUpload := historicalUploadRequest(t, makeHistoricalTestZIP(t, "private"), "historico.zip", "EMERGENCIA", "08", "2021")
	unauthenticatedUpload.Header.Del("Cookie")
	unauthenticatedUploadResponse := httptest.NewRecorder()
	s.historicalArchives(unauthenticatedUploadResponse, unauthenticatedUpload)
	if unauthenticatedUploadResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated upload status = %d", unauthenticatedUploadResponse.Code)
	}

	unauthenticated := httptest.NewRequest(http.MethodGet, "/api/v1/archivo-historico", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	s.historicalArchives(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d", unauthenticatedResponse.Code)
	}

	response := httptest.NewRecorder()
	s.historicalArchives(response, historicalUploadRequest(t, []byte("not a ZIP"), "historico.zip", "EMERGENCIA", "08", "2021"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid ZIP upload status = %d, body = %s", response.Code, response.Body.String())
	}
	entries, err := os.ReadDir(s.historicalArchivesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("invalid upload created saved archive entries: %+v", entries)
	}
}

func TestDeleteHistoricalArchiveRequiresRoleAndExactIDConfirmation(t *testing.T) {
	s := historicalArchiveTestServer(t)
	uploadResponse := httptest.NewRecorder()
	s.historicalArchives(uploadResponse, historicalUploadRequest(t, makeHistoricalTestZIP(t, "keep until authorized"), "historico.zip", "EMERGENCIA", "08", "2021"))
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, body = %s", uploadResponse.Code, uploadResponse.Body.String())
	}
	var uploaded struct {
		Archive historicalArchiveRecord `json:"archive"`
	}
	if err := json.Unmarshal(uploadResponse.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	archiveDir := filepath.Join(s.historicalArchivesDir, uploaded.Archive.ID)
	zipPath := filepath.Join(archiveDir, "original.zip")

	deleteRequest := func(confirmation string) *http.Request {
		body, err := json.Marshal(map[string]string{"confirmacion": confirmation})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodDelete, "/api/v1/archivo-historico/eliminar/"+uploaded.Archive.ID, bytes.NewReader(body))
		request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
		return request
	}

	denied := httptest.NewRecorder()
	s.deleteHistoricalArchive(denied, deleteRequest(uploaded.Archive.ID))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unauthorized delete status = %d, want 403, body = %s", denied.Code, denied.Body.String())
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("ZIP was changed without the delete role: %v", err)
	}

	s.mu.Lock()
	s.sessions["test-session"] = session{username: "supervisor", canDeleteWorkspaces: true, expiresAt: time.Now().Add(time.Hour)}
	s.mu.Unlock()

	wrongConfirmation := httptest.NewRecorder()
	s.deleteHistoricalArchive(wrongConfirmation, deleteRequest("HIST-incorrecto"))
	if wrongConfirmation.Code != http.StatusBadRequest {
		t.Fatalf("wrong confirmation status = %d, want 400, body = %s", wrongConfirmation.Code, wrongConfirmation.Body.String())
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("ZIP was changed without the exact ID confirmation: %v", err)
	}

	deleted := httptest.NewRecorder()
	s.deleteHistoricalArchive(deleted, deleteRequest(uploaded.Archive.ID))
	if deleted.Code != http.StatusOK {
		t.Fatalf("authorized delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}
	if _, err := os.Stat(archiveDir); !os.IsNotExist(err) {
		t.Fatalf("archive directory still exists after authorized delete, stat error = %v", err)
	}
}
