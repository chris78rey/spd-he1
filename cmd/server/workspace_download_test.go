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
	wantEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", "1. PACIENTE 9900888", "HCU_008.pdf"))
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
	if err := os.MkdirAll(filepath.Join(packageRoot, "5. ANEXOS", patientFolder), 0700); err != nil {
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
	markedEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", "1. PACIENTE FICTICIO", "HCU_008.pdf"))
	unmarkedEntry := filepath.ToSlash(filepath.Join(packageFolderName(&job), "4. EXPEDIENTES", "1. PACIENTE FICTICIO", "HCU_006.pdf"))
	annexRoot := filepath.ToSlash(filepath.Join(packageFolderName(&job), "5. ANEXOS")) + "/"
	foundMarked, foundUnmarked, foundAnnex := false, false, false
	for _, entry := range archive.File {
		foundMarked = foundMarked || entry.Name == markedEntry
		foundUnmarked = foundUnmarked || entry.Name == unmarkedEntry
		foundAnnex = foundAnnex || strings.HasPrefix(entry.Name, annexRoot)
	}
	if !foundMarked || foundUnmarked || foundAnnex {
		t.Fatalf("objection progress ZIP selection/optional annexes: marked=%v unmarked=%v annexEntries=%v", foundMarked, foundUnmarked, foundAnnex)
	}
	if after, err := os.ReadFile(filepath.Join(s.jobRoot(id), "job.json")); err != nil || !bytes.Equal(after, metadata) {
		t.Fatalf("progress download changed objection workspace metadata: err=%v", err)
	}
}

func TestNormalizeArchivePatientName(t *testing.T) {
	got := normalizeArchivePatientName("  Peña-Ávila, María José_2  ")
	if got != "PENAAVILA MARIA JOSE 2" {
		t.Fatalf("normalizeArchivePatientName() = %q, want %q", got, "PENAAVILA MARIA JOSE 2")
	}
}

func TestWorkspaceArchivePatientFoldersSortsAndKeepsTransactionsSeparate(t *testing.T) {
	job := stagedJob{IsObjections: true, ObjectionRows: []objectionRecord{
		{Tramite: "20", Patient: "PÉREZ GÓMEZ JUAN CARLOS", PatientFolder: "ID_PEREZ_20"},
		{Tramite: "10", Patient: "PÉREZ GÓMEZ JUAN CARLOS", PatientFolder: "ID_PEREZ_10"},
		{Tramite: "2", Patient: "PÉREZ GÓMEZ JUAN CARLOS", PatientFolder: "ID_PEREZ_2"},
		{Tramite: "30", Patient: "ÁLVAREZ NÚÑEZ ANA", PatientFolder: "ID_ALVAREZ_30"},
	}}
	bySource, ordered, err := workspaceArchivePatientFolders(job, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ID_ALVAREZ_30": "1. ALVAREZ NUNEZ ANA",
		"ID_PEREZ_2":    "2. PEREZ GOMEZ JUAN CARLOS",
		"ID_PEREZ_10":   "3. PEREZ GOMEZ JUAN CARLOS",
		"ID_PEREZ_20":   "4. PEREZ GOMEZ JUAN CARLOS",
	}
	if len(bySource) != len(want) {
		t.Fatalf("folder map has %d entries, want %d: %#v", len(bySource), len(want), bySource)
	}
	for source, expected := range want {
		if got := bySource[source]; got != expected {
			t.Errorf("folder %q maps to %q, want %q", source, got, expected)
		}
	}
	wantOrder := []string{"1. ALVAREZ NUNEZ ANA", "2. PEREZ GOMEZ JUAN CARLOS", "3. PEREZ GOMEZ JUAN CARLOS", "4. PEREZ GOMEZ JUAN CARLOS"}
	if strings.Join(ordered, "\n") != strings.Join(wantOrder, "\n") {
		t.Fatalf("ordered folder names = %#v, want %#v", ordered, wantOrder)
	}

	packageRoot := t.TempDir()
	patientRoot := filepath.Join(packageRoot, "4. EXPEDIENTES")
	for _, folder := range []string{"ZAMBRANO_NUNEZ_CARLOS", "ALVAREZ_GOMEZ_ANA"} {
		if err := os.MkdirAll(filepath.Join(patientRoot, folder), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(patientRoot, "ÁVILA_GARCIA_MARIA"), 0700); err != nil {
		t.Fatal(err)
	}
	normalNames, normalOrder, err := workspaceArchivePatientFolders(stagedJob{}, packageRoot)
	if err != nil {
		t.Fatal(err)
	}
	wantNormalOrder := []string{"1. ALVAREZ GOMEZ ANA", "2. AVILA GARCIA MARIA", "3. ZAMBRANO NUNEZ CARLOS"}
	if strings.Join(normalOrder, "\n") != strings.Join(wantNormalOrder, "\n") {
		t.Fatalf("normal workspace order = %#v, want %#v", normalOrder, wantNormalOrder)
	}
	if normalNames["ÁVILA_GARCIA_MARIA"] != "2. AVILA GARCIA MARIA" {
		t.Fatalf("normal workspace map did not normalize the accented name: %#v", normalNames)
	}
}

func TestDownloadObjectionZIPKeepsRepeatedPatientTransactionsInSeparateNumberedFolders(t *testing.T) {
	withRepositoryWorkingDirectory(t)

	const id = "JOB-20261007T140000-abcdef0123456789"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	firstFolder := "1720000000_PEREZ_GOMEZ_JUAN_2"
	secondFolder := "1720000000_PEREZ_GOMEZ_JUAN_10"
	job := stagedJob{
		ID: id, Status: "INCOMPLETE", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true,
		ObjectionRows: []objectionRecord{
			{Tramite: "10", Patient: "PÉREZ GÓMEZ JUAN", PatientFolder: secondFolder},
			{Tramite: "2", Patient: "PÉREZ GÓMEZ JUAN", PatientFolder: firstFolder},
		},
		ObjectionPDFSelection: map[string]bool{
			"4. EXPEDIENTES/" + firstFolder + "/HCU_008.pdf":  true,
			"4. EXPEDIENTES/" + secondFolder + "/HCU_008.pdf": true,
		},
	}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	for _, folder := range []string{firstFolder, secondFolder} {
		pdfPath := filepath.Join(packageRoot, "4. EXPEDIENTES", folder, "HCU_008.pdf")
		if err := os.MkdirAll(filepath.Dir(pdfPath), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"+folder+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	annexPath := filepath.Join(packageRoot, "5. ANEXOS", secondFolder, "FACTURA_COMPRA.pdf")
	if err := os.MkdirAll(filepath.Dir(annexPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(annexPath, []byte("%PDF-1.4\nfactura de respaldo\n"), 0600); err != nil {
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
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("download was not a valid ZIP: %v", err)
	}
	entries := make(map[string]bool, len(archive.File))
	for _, entry := range archive.File {
		entries[entry.Name] = true
	}
	root := packageFolderName(&job) + "/"
	for _, name := range []string{"1. PEREZ GOMEZ JUAN", "2. PEREZ GOMEZ JUAN"} {
		if !entries[root+"4. EXPEDIENTES/"+name+"/HCU_008.pdf"] {
			t.Errorf("ZIP omitted transaction folder %q", name)
		}
	}
	if entries[root+"5. ANEXOS/1. PEREZ GOMEZ JUAN/"] {
		t.Error("ZIP included an empty annex folder for the transaction without a justification")
	}
	if !entries[root+"5. ANEXOS/2. PEREZ GOMEZ JUAN/FACTURA_COMPRA.pdf"] {
		t.Error("ZIP omitted the annex or failed to preserve the matching patient/trámite number")
	}
	for _, folder := range []string{firstFolder, secondFolder} {
		if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", folder, "HCU_008.pdf")); err != nil {
			t.Errorf("ZIP export changed the stored transaction path %q: %v", folder, err)
		}
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
