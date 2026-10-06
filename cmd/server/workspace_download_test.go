package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadWorkspaceZIPMarksIncompleteProgressWithoutChangingWorkspace(t *testing.T) {
	withRepositoryWorkingDirectory(t)

	const id = "JOB-20261006T140000-abcdef0123456789"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{ID: id, Status: "INCOMPLETE", Service: "AMBULATORIO", Month: "08", Year: "2026"}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	documentPath := filepath.Join(packageRoot, "4. EXPEDIENTES", "PACIENTE_9900888", "HCU_008.pdf")
	if err := os.MkdirAll(filepath.Dir(documentPath), 0700); err != nil {
		t.Fatal(err)
	}
	document := []byte("%PDF-1.4\ncontenido clínico disponible\n")
	if err := os.WriteFile(documentPath, document, 0600); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/expedientes/descargar/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.downloadWorkspaceZIP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("download returned %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Folio-Archive-State"); got != "INCOMPLETE" {
		t.Fatalf("archive state = %q, want INCOMPLETE", got)
	}
	if got := response.Header().Get("Content-Disposition"); !strings.Contains(got, "_AVANCE_INCOMPLETO.zip") {
		t.Fatalf("content disposition = %q, want incomplete-progress filename", got)
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("download was not a valid ZIP: %v", err)
	}
	wantEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", "PACIENTE_9900888", "HCU_008.pdf"))
	foundDocument := false
	for _, entry := range archive.File {
		if entry.Name != wantEntry {
			continue
		}
		foundDocument = true
		file, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("could not read ZIP entry: read=%v close=%v", readErr, closeErr)
		}
		if !bytes.Equal(contents, document) {
			t.Fatal("ZIP changed the available PDF contents")
		}
	}
	if !foundDocument {
		t.Fatalf("ZIP omitted available document %q", wantEntry)
	}
	missing, ready := s.workspaceDeliveryMissing(job)
	if ready || len(missing) != 3 {
		t.Fatalf("readiness = %v, missing = %#v; want incomplete with three headers", ready, missing)
	}
	for _, name := range []string{"1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf", matrixFilename(&job)} {
		if !strings.Contains(strings.Join(missing, "\n"), name) {
			t.Errorf("missing list %#v does not name %q", missing, name)
		}
	}
	afterMetadata, err := os.ReadFile(filepath.Join(s.jobRoot(id), "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterMetadata, metadata) {
		t.Fatal("downloading a progress ZIP changed the saved workspace state")
	}
	afterDocument, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterDocument, document) {
		t.Fatal("downloading a progress ZIP changed the saved PDF")
	}
}

func TestDownloadObjectionProgressZIPAllowsMissingCloseoutDocuments(t *testing.T) {
	withRepositoryWorkingDirectory(t)

	const id = "JOB-20261006T140050-abcdef0123456789"
	patientFolder := "PACIENTE_9900890"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true,
		ObjectionRows:         []objectionRecord{{Tramite: "9900890", Patient: "PACIENTE FICTICIO", PatientFolder: patientFolder}},
		ObjectionPDFSelection: map[string]bool{"4. EXPEDIENTES/" + patientFolder + "/HCU_008.pdf": true},
	}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	patientRoot := filepath.Join(packageRoot, "4. EXPEDIENTES", patientFolder)
	if err := os.MkdirAll(patientRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string][]byte{
		"HCU_008.pdf": []byte("%PDF-1.4\nmarcado\n"),
		"HCU_006.pdf": []byte("%PDF-1.4\nno marcado\n"),
	} {
		if err := os.WriteFile(filepath.Join(patientRoot, name), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/expedientes/descargar/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.downloadWorkspaceZIP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("download returned %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Folio-Archive-State"); got != "INCOMPLETE" {
		t.Fatalf("archive state = %q, want INCOMPLETE", got)
	}
	if got := response.Header().Get("Content-Disposition"); !strings.Contains(got, "_AVANCE_INCOMPLETO.zip") {
		t.Fatalf("content disposition = %q, want incomplete-progress filename", got)
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("download was not a valid ZIP: %v", err)
	}
	markedEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", patientFolder, "HCU_008.pdf"))
	unmarkedEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", patientFolder, "HCU_006.pdf"))
	foundMarked, foundUnmarked := false, false
	for _, entry := range archive.File {
		foundMarked = foundMarked || entry.Name == markedEntry
		foundUnmarked = foundUnmarked || entry.Name == unmarkedEntry
	}
	if !foundMarked || foundUnmarked {
		t.Fatalf("objection progress ZIP selection: marked=%v unmarked=%v", foundMarked, foundUnmarked)
	}
	if after, err := os.ReadFile(filepath.Join(s.jobRoot(id), "job.json")); err != nil || !bytes.Equal(after, metadata) {
		t.Fatalf("progress download changed objection workspace metadata: err=%v", err)
	}
}

func TestDownloadWorkspaceZIPNamesCompletePackageForDelivery(t *testing.T) {
	withRepositoryWorkingDirectory(t)

	const id = "JOB-20261006T140100-abcdef0123456789"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026",
		Files: []stagedUpload{{Field: "matriz_file"}, {Field: "consolidada_file"}, {Field: "oficio_file"}},
	}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	for _, name := range []string{"1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf"} {
		if err := os.MkdirAll(packageRoot, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(packageRoot, name), []byte("%PDF-1.4\nhabilitante\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(packageRoot, matrixFilename(&job)), testMacroEnabledWorkbook(t), 0600); err != nil {
		t.Fatal(err)
	}
	clinicalPDF := filepath.Join(packageRoot, "4. EXPEDIENTES", "PACIENTE_9900889", "HCU_008.pdf")
	if err := os.MkdirAll(filepath.Dir(clinicalPDF), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(clinicalPDF, []byte("%PDF-1.4\ncontenido clínico\n"), 0600); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/expedientes/descargar/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.downloadWorkspaceZIP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("download returned %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Folio-Archive-State"); got != "READY" {
		t.Fatalf("archive state = %q, want READY", got)
	}
	if got := response.Header().Get("Content-Disposition"); strings.Contains(got, "AVANCE_INCOMPLETO") {
		t.Fatalf("complete package filename incorrectly marks an incomplete advance: %q", got)
	}
	missing, ready := s.workspaceDeliveryMissing(job)
	if !ready || len(missing) != 0 {
		t.Fatalf("readiness = %v, missing = %#v; want ready with no requirements", ready, missing)
	}
}

func withRepositoryWorkingDirectory(t *testing.T) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repositoryRoot := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	if err := os.Chdir(repositoryRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
