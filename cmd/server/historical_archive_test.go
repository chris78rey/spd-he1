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
