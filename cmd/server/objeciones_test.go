package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestObjectionZIPSelectionOnlyFiltersObjectionPatientPDFs(t *testing.T) {
	job := stagedJob{
		IsObjections: true,
		ObjectionPDFSelection: map[string]bool{
			"4. EXPEDIENTES/PACIENTE/HCU_008.pdf": false,
		},
	}
	if objectionZIPIncludesFile(job, "4. EXPEDIENTES/PACIENTE/HCU_008.pdf") {
		t.Fatal("an objection PDF marked out should not enter its ZIP")
	}
	for _, path := range []string{
		"4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL.pdf",
		"1. OFICIO DE PAGO.pdf",
		"OFICIO_LIQUIDACION_MSP.pdf",
		"5. ANEXOS/PACIENTE/FACTURA.pdf",
	} {
		if !objectionZIPIncludesFile(job, path) {
			t.Fatalf("required or non-selectable package path %q was filtered", path)
		}
	}
	if objectionZIPIncludesFile(job, "3. MATRIZ_RESPUESTA.xlsx") {
		t.Fatal("internal response matrix should remain excluded from objection ZIP")
	}
	if !objectionZIPIncludesFile(stagedJob{}, "4. EXPEDIENTES/PACIENTE/HCU_008.pdf") {
		t.Fatal("the normal workspace ZIP must keep including patient PDFs")
	}
}

func TestObjectionPDFSelectionPersistsAndProtectsRequiredDocuments(t *testing.T) {
	id := "JOB-20261006T130000-abcdef0123456789"
	patientFolder := "PACIENTE_9900601"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true,
		ObjectionRows: []objectionRecord{{Tramite: "9900601", PlanillaID: 701, Patient: "PACIENTE", PatientFolder: patientFolder, Matched: true}},
		DocumentPlanillas: map[string]int64{
			"4. EXPEDIENTES/" + patientFolder + "/P_INDIVIDUAL.pdf": 701,
			"4. EXPEDIENTES/" + patientFolder + "/C_COBERTURA.pdf":  701,
			"4. EXPEDIENTES/" + patientFolder + "/HCU_008.pdf":      701,
		},
	}
	if _, ok := objectionPDFRecord(job, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf"); !ok {
		t.Fatalf("fixture objection PDF did not match the row: %+v", job.ObjectionRows)
	}
	if err := os.MkdirAll(s.jobRoot(id), 0700); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job), "4. EXPEDIENTES", patientFolder)
	if err := os.MkdirAll(packageRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"P_INDIVIDUAL.pdf", "C_COBERTURA.pdf", "HCU_008.pdf"} {
		if err := os.WriteFile(filepath.Join(packageRoot, name), []byte("%PDF-1.4\nficticio\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/objeciones/pdfs/seleccion/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.objectionPDFSelection(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("initial objectionPDFSelection() status = %d, body = %s", response.Code, response.Body.String())
	}
	var initial struct {
		Documents []objectionPackagePDF `json:"documents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	for _, document := range initial.Documents {
		if document.Required && !document.Included {
			t.Fatalf("required PDF %q was not selected by default", document.Path)
		}
		if document.Name == "HCU_008.pdf" && document.Included {
			t.Fatal("optional clinical PDFs should require explicit selection")
		}
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/pdfs/seleccion/"+id,
		bytes.NewBufferString(`{"path":"4. EXPEDIENTES/`+patientFolder+`/HCU_008.pdf","incluir":false}`))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response = httptest.NewRecorder()
	s.objectionPDFSelection(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("objectionPDFSelection() status = %d, body = %s", response.Code, response.Body.String())
	}
	updated, err := s.loadStagedJob(id)
	if err != nil {
		t.Fatal(err)
	}
	if got := planillaForPath(updated, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf"); got != 701 {
		t.Fatalf("planillaForPath() = %d; job mapping = %#v", got, updated.DocumentPlanillas)
	}
	if _, ok := objectionPDFRecord(updated, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf"); !ok {
		t.Fatalf("objectionPDFRecord() did not match row %+v for path %q", updated.ObjectionRows, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf")
	}
	if objectionPDFIncluded(updated, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf") {
		t.Fatal("the saved objection PDF exclusion was not persisted")
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/pdfs/seleccion/"+id,
		bytes.NewBufferString(`{"path":"4. EXPEDIENTES/`+patientFolder+`/HCU_008.pdf","incluir":true}`))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response = httptest.NewRecorder()
	s.objectionPDFSelection(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("including optional objection PDF status = %d, body = %s", response.Code, response.Body.String())
	}
	updated, err = s.loadStagedJob(id)
	if err != nil || !objectionPDFIncluded(updated, "4. EXPEDIENTES/"+patientFolder+"/HCU_008.pdf") {
		t.Fatalf("the saved objection PDF inclusion was not persisted: job=%+v error=%v", updated, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/pdfs/seleccion/"+id,
		bytes.NewBufferString(`{"path":"4. EXPEDIENTES/`+patientFolder+`/P_INDIVIDUAL.pdf","incluir":false}`))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response = httptest.NewRecorder()
	s.objectionPDFSelection(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unselecting required P_INDIVIDUAL.pdf status = %d, body = %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/postura/"+id,
		bytes.NewBufferString(`{"pdi_tramite":"9900601","postura":"RECHAZA"}`))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response = httptest.NewRecorder()
	s.setObjectionPosture(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("saving objection posture without uploading a PDF status = %d, body = %s", response.Code, response.Body.String())
	}
	updated, err = s.loadStagedJob(id)
	if err != nil || len(updated.ObjectionRows) != 1 || updated.ObjectionRows[0].Posture != "RECHAZA" {
		t.Fatalf("objection posture was not persisted: job=%+v error=%v", updated, err)
	}
}

func TestObjectionUploadAcceptsOnlyNewAnnexes(t *testing.T) {
	id := "JOB-20261006T130100-abcdef0123456789"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{ID: id, Status: "PROCESSED", IsObjections: true}
	if err := os.MkdirAll(s.jobRoot(id), 0700); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"respuesta", "cobertura"} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("tipo", kind); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/documentos/"+id, &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
		response := httptest.NewRecorder()
		s.uploadObjectionDocument(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "solo se cargan anexos") {
			t.Fatalf("uploadObjectionDocument(%q) = %d, %s; want rejection for duplicate uploads", kind, response.Code, response.Body.String())
		}
	}
}

func TestDownloadObjectionZIPIncludesOnlyMarkedClinicalPDFs(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Clean(filepath.Join(workingDirectory, "..", ".."))); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(workingDirectory) }()

	id := "JOB-20261006T130200-abcdef0123456789"
	patientFolder := "PACIENTE_9900602"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{
		ID: id, Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true,
		ObjectionRows:         []objectionRecord{{Tramite: "9900602", PlanillaID: 702, Patient: "PACIENTE", PatientFolder: patientFolder, Matched: true, Posture: "RECHAZA"}},
		ObjectionPDFSelection: map[string]bool{"4. EXPEDIENTES/" + patientFolder + "/HCU_006.pdf": true},
		DocumentPlanillas: map[string]int64{
			"4. EXPEDIENTES/" + patientFolder + "/P_INDIVIDUAL.pdf": 702,
			"4. EXPEDIENTES/" + patientFolder + "/C_COBERTURA.pdf":  702,
			"4. EXPEDIENTES/" + patientFolder + "/HCU_006.pdf":      702,
			"4. EXPEDIENTES/" + patientFolder + "/HCU_008.pdf":      702,
		},
	}
	if err := os.MkdirAll(s.jobRoot(id), 0700); err != nil {
		t.Fatal(err)
	}
	metadata, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(id), "job.json"), metadata, 0600); err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(s.jobRoot(id), "trabajo", packageFolderName(&job))
	patientRoot := filepath.Join(packageRoot, "4. EXPEDIENTES", patientFolder)
	annexRoot := filepath.Join(packageRoot, "5. ANEXOS", patientFolder)
	for _, directory := range []string{patientRoot, annexRoot} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"P_INDIVIDUAL.pdf", "C_COBERTURA.pdf", "HCU_006.pdf", "HCU_008.pdf"} {
		if err := os.WriteFile(filepath.Join(patientRoot, name), []byte("%PDF-1.4\nficticio\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"OFICIO_LIQUIDACION_MSP.pdf", "I_LIQUIDACION.pdf", "1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf"} {
		if err := os.WriteFile(filepath.Join(packageRoot, name), []byte("%PDF-1.4\ncabecera\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	matrixFile, err := os.Create(filepath.Join(packageRoot, objectionMatrixFilename(job)))
	if err != nil {
		t.Fatal(err)
	}
	matrixZip := zip.NewWriter(matrixFile)
	contentTypes, err := matrixZip.Create("[Content_Types].xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contentTypes.Write([]byte("macroEnabled.main+xml")); err != nil {
		t.Fatal(err)
	}
	if _, err := matrixZip.Create("xl/workbook.xml"); err != nil {
		t.Fatal(err)
	}
	if err := matrixZip.Close(); err != nil {
		t.Fatal(err)
	}
	if err := matrixFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(annexRoot, "FACTURA.pdf"), []byte("%PDF-1.4\nanexo\n"), 0600); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/expedientes/descargar/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.downloadWorkspaceZIP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("downloadWorkspaceZIP() status = %d, body = %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("download did not produce a ZIP: %v", err)
	}
	entries := make(map[string]bool, len(archive.File))
	for _, file := range archive.File {
		entries[file.Name] = true
	}
	root := packageFolderName(&job) + "/"
	for _, path := range []string{
		root + "OFICIO_LIQUIDACION_MSP.pdf",
		root + "4. EXPEDIENTES/1. PACIENTE/P_INDIVIDUAL.pdf",
		root + "4. EXPEDIENTES/1. PACIENTE/C_COBERTURA.pdf",
		root + "4. EXPEDIENTES/1. PACIENTE/HCU_006.pdf",
		root + "5. ANEXOS/1. PACIENTE/FACTURA.pdf",
	} {
		if !entries[path] {
			t.Errorf("expected ZIP entry %q", path)
		}
	}
	if entries[root+"4. EXPEDIENTES/1. PACIENTE/HCU_008.pdf"] {
		t.Fatal("unmarked clinical PDF was included in objection ZIP")
	}
}

func TestObjectionHeaderNameSeparatesMSPOfficeFromHospitalResponse(t *testing.T) {
	job := stagedJob{Service: "AMBULATORIO", Month: "08", Year: "2026"}
	cases := []struct {
		kind, wantName, wantField string
	}{
		{"oficio_liquidacion_msp", "OFICIO_LIQUIDACION_MSP.pdf", "objection_oficio_liquidacion_msp"},
		{"liquidacion", "I_LIQUIDACION.pdf", "objection_liquidacion"},
		{"oficio", "1. OFICIO DE PAGO.pdf", "objection_oficio"},
	}
	for _, test := range cases {
		name, field, err := objectionHeaderName(job, test.kind)
		if err != nil {
			t.Fatalf("objectionHeaderName(%q): %v", test.kind, err)
		}
		if name != test.wantName || field != test.wantField {
			t.Errorf("objectionHeaderName(%q) = (%q, %q), want (%q, %q)", test.kind, name, field, test.wantName, test.wantField)
		}
	}
}

func TestObjectionHeaderStatusesShowsMSPOfficeFirstAndDistinguishesHospitalResponse(t *testing.T) {
	s := &server{workspacesDir: t.TempDir()}
	job := stagedJob{ID: "JOB-20261007T140000-abcdef0123456789", Service: "AMBULATORIO", Month: "08", Year: "2026"}
	statuses := objectionHeaderStatuses(s, job)
	if len(statuses) < 3 {
		t.Fatalf("header statuses returned %d entries, want at least 3", len(statuses))
	}
	want := []struct{ kind, label string }{
		{"oficio_liquidacion_msp", "Oficio de liquidación del MSP"},
		{"liquidacion", "Informe de liquidación del MSP"},
		{"oficio", "Oficio de pago del hospital"},
	}
	for index, expected := range want {
		if got := statuses[index]["tipo"]; got != expected.kind {
			t.Errorf("header %d type = %v, want %q", index, got, expected.kind)
		}
		if got := statuses[index]["etiqueta"]; got != expected.label {
			t.Errorf("header %d label = %v, want %q", index, got, expected.label)
		}
	}
}

func TestBuildSelectedObjectionRowsIncludesOnlyChosenCandidates(t *testing.T) {
	candidates := []objectionRecord{
		{Tramite: "9900401", Cedula: "0000000001", Patient: "PACIENTE FICTICIO UNO", PatientFolder: "PACIENTE_FICTICIO_UNO", PlanillaID: 801, Matched: true},
		{Tramite: "9900402", Cedula: "0000000002", Patient: "PACIENTE FICTICIO DOS", PatientFolder: "PACIENTE_FICTICIO_DOS", PlanillaID: 802, Matched: true},
	}
	rows, err := buildSelectedObjectionRows([]string{"9900402"}, candidates)
	if err != nil {
		t.Fatalf("buildSelectedObjectionRows() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Tramite != "9900402" || rows[0].Patient != "PACIENTE FICTICIO DOS" {
		t.Fatalf("got selected rows %+v; want only the second patient", rows)
	}
	if _, err := buildSelectedObjectionRows(nil, candidates); err == nil {
		t.Fatal("expected an empty selection to be rejected")
	}
	if _, err := buildSelectedObjectionRows([]string{"9900499"}, candidates); err == nil {
		t.Fatal("expected an unknown trámite to be rejected")
	}
}

func TestObjectionFolderIsUniquePerTransaction(t *testing.T) {
	first := objectionPatientFolder("1712345678", "GARCIA LUIS", "9900101")
	second := objectionPatientFolder("1712345678", "GARCIA LUIS", "9900102")
	if first == second || !strings.HasSuffix(first, "_9900101") || !strings.HasSuffix(second, "_9900102") {
		t.Fatalf("transaction folders are not distinct: %q and %q", first, second)
	}
	job := stagedJob{Service: "AMBULATORIO", Month: "08", Year: "2026"}
	if got := objectionMatrixFilename(job); got != "3. MATRIZ_OBJECIONES_AMBULATORIO_AGOSTO_2026.xlsm" {
		t.Fatalf("objection matrix name = %q", got)
	}
}

func TestReadObjectionRequestNeedsOnlySourceAndSelection(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("expediente_origen", "JOB-20261005T120000-abcdef0123456789"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("tramites_objetados", "9900501"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/crear", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	input, err := (&server{}).readObjectionRequest(httptest.NewRecorder(), request)
	if err != nil {
		t.Fatalf("readObjectionRequest() with only selection: %v", err)
	}
	if len(input.selectedTramites) != 1 || input.selectedTramites[0] != "9900501" {
		t.Fatalf("unexpected selected trámites: %v", input.selectedTramites)
	}
}

func TestObjectionWorkspacesForPeriodReturnsAllMatchingDuplicates(t *testing.T) {
	s := &server{workspacesDir: filepath.Join(t.TempDir(), "expedientes")}
	jobs := []stagedJob{
		{ID: "JOB-20261006T130001-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true},
		{ID: "JOB-20261006T130002-abcdef0123456789", Status: "INCOMPLETE", Service: "ambulatorio", Month: "8", Year: "2026", IsObjections: true},
		{ID: "JOB-20261006T130003-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "09", Year: "2026", IsObjections: true},
		{ID: "JOB-20261006T130004-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026"},
	}
	for _, job := range jobs {
		persistTestObjectionJob(t, s, job)
	}

	got, err := s.objectionWorkspacesForPeriod("Ambulatorio", "8", "2026")
	if err != nil {
		t.Fatalf("objectionWorkspacesForPeriod() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != jobs[0].ID || got[1].ID != jobs[1].ID {
		t.Fatalf("matching workspaces = %#v; want both duplicate IDs in received order", got)
	}
}

func TestCreateObjectionWorkspaceReportsExistingDuplicatesWithoutCreatingAnother(t *testing.T) {
	root := t.TempDir()
	s := &server{
		workspacesDir: filepath.Join(root, "expedientes"),
		stagingDir:    filepath.Join(root, "staging"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	source := stagedJob{
		ID: "JOB-20261006T130000-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026",
		Files: []stagedUpload{{Field: "zip_file", StoredName: "lote.zip"}},
	}
	persistTestObjectionJob(t, s, source)
	if err := os.MkdirAll(s.jobSourcesDir(source.ID), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobSourcesDir(source.ID), "lote.zip"), []byte("zip ficticio"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, job := range []stagedJob{
		{ID: "JOB-20261006T130001-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026", IsObjections: true},
		{ID: "JOB-20261006T130002-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "8", Year: "2026", IsObjections: true},
	} {
		persistTestObjectionJob(t, s, job)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for field, value := range map[string]string{"expediente_origen": source.ID, "tramites_objetados": "9900501"} {
		if err := writer.WriteField(field, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/crear", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.createObjectionWorkspace(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("createObjectionWorkspace() status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Spaces []struct {
			ID string `json:"job_id"`
		} `json:"espacios_existentes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Spaces) != 2 {
		t.Fatalf("server returned %d existing spaces, want 2: %s", len(result.Spaces), response.Body.String())
	}
	if _, err := os.Stat(s.jobRoot("JOB-20261006T130003-abcdef0123456789")); !os.IsNotExist(err) {
		t.Fatalf("server created a third workspace while duplicates exist: %v", err)
	}
}

func TestAppendObjectionRowsPreservesWorkAndInvalidatesExistingMatrix(t *testing.T) {
	root := t.TempDir()
	s := &server{workspacesDir: filepath.Join(root, "expedientes")}
	source := stagedJob{ID: "JOB-20261006T130000-abcdef0123456789", Service: "AMBULATORIO", Month: "08", Year: "2026"}
	job := stagedJob{
		ID: "JOB-20261006T130001-abcdef0123456789", Status: "PROCESSED", Service: source.Service, Month: source.Month, Year: source.Year,
		IsObjections: true, ObjectionSourceID: source.ID,
		ObjectionRows:         []objectionRecord{{Tramite: "9900501", Cedula: "0000000501", Patient: "PACIENTE FICTICIO UNO", PatientFolder: "PACIENTE_FICTICIO_UNO_9900501", SourcePatientFolder: "PACIENTE_FICTICIO_UNO", PlanillaID: 501, Matched: true, Posture: "RECHAZA"}},
		DocumentPlanillas:     map[string]int64{"4. EXPEDIENTES/PACIENTE_FICTICIO_UNO_9900501/HCU_008.pdf": 501},
		ObjectionPDFSelection: map[string]bool{"4. EXPEDIENTES/PACIENTE_FICTICIO_UNO_9900501/HCU_008.pdf": true},
		Files:                 []stagedUpload{{Field: "objection_matriz", StoredName: objectionMatrixFilename(stagedJob{Service: source.Service, Month: source.Month, Year: source.Year})}, {Field: "objection_oficio", StoredName: "1. OFICIO DE PAGO.pdf"}},
	}
	persistTestObjectionJob(t, s, job)
	packageRoot := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	oldPDF := filepath.Join(packageRoot, "4. EXPEDIENTES", job.ObjectionRows[0].PatientFolder, "HCU_008.pdf")
	if err := os.MkdirAll(filepath.Dir(oldPDF), 0700); err != nil {
		t.Fatal(err)
	}
	oldContents := []byte("PDF clínico ficticio existente")
	if err := os.WriteFile(oldPDF, oldContents, 0600); err != nil {
		t.Fatal(err)
	}
	matrixPath := filepath.Join(packageRoot, objectionMatrixFilename(job))
	matrixContents := testMacroEnabledWorkbook(t)
	if err := os.WriteFile(matrixPath, matrixContents, 0600); err != nil {
		t.Fatal(err)
	}
	newSourceFolder := "PACIENTE_FICTICIO_DOS"
	newSourcePDF := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source), "4. EXPEDIENTES", newSourceFolder, "HCU_006.pdf")
	if err := os.MkdirAll(filepath.Dir(newSourcePDF), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSourcePDF, []byte("%PDF-1.4\nPDF nuevo ficticio\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source.DocumentPlanillas = map[string]int64{"4. EXPEDIENTES/PACIENTE_FICTICIO_DOS/HCU_006.pdf": 502}
	newRow := objectionRecord{Tramite: "9900502", Cedula: "0000000502", Patient: "PACIENTE FICTICIO DOS", PatientFolder: "PACIENTE_FICTICIO_DOS_9900502", SourcePatientFolder: newSourceFolder, PlanillaID: 502, Matched: true}
	loaded, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	added, invalidated, err := s.appendObjectionRows(&loaded, source, []objectionRecord{newRow})
	if err != nil {
		t.Fatalf("appendObjectionRows() error = %v", err)
	}
	if added != 1 || !invalidated {
		t.Fatalf("appendObjectionRows() = added %d, invalidated %t; want 1, true", added, invalidated)
	}
	updated, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.ObjectionRows) != 2 || updated.ObjectionRows[0].Posture != "RECHAZA" {
		t.Fatalf("existing rows/posture were not preserved: %+v", updated.ObjectionRows)
	}
	if !updated.ObjectionPDFSelection["4. EXPEDIENTES/PACIENTE_FICTICIO_UNO_9900501/HCU_008.pdf"] {
		t.Fatal("existing PDF selection was not preserved")
	}
	if got, err := os.ReadFile(oldPDF); err != nil || !bytes.Equal(got, oldContents) {
		t.Fatalf("existing clinical PDF changed: contents=%q error=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", newRow.PatientFolder, "HCU_006.pdf")); err != nil {
		t.Fatalf("newly selected transaction was not copied: %v", err)
	}
	if _, err := os.Stat(matrixPath); !os.IsNotExist(err) {
		t.Fatalf("stale matrix should be removed after adding a transaction: %v", err)
	}
	if len(updated.Files) != 1 || updated.Files[0].Field != "objection_oficio" {
		t.Fatalf("matrix metadata should be invalidated while other header metadata remains: %+v", updated.Files)
	}
	versions, err := os.ReadDir(filepath.Join(s.jobRoot(job.ID), "fuentes", "versiones"))
	if err != nil || len(versions) != 1 {
		t.Fatalf("old matrix should be kept in versions: entries=%v error=%v", versions, err)
	}
	archivedMatrix, err := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "fuentes", "versiones", versions[0].Name()))
	if err != nil || !bytes.Equal(archivedMatrix, matrixContents) {
		t.Fatalf("archived matrix changed: error=%v", err)
	}
	added, invalidated, err = s.appendObjectionRows(&updated, source, []objectionRecord{newRow})
	if err != nil || added != 0 || invalidated {
		t.Fatalf("readding an existing transaction should be a no-op: added=%d invalidated=%t error=%v", added, invalidated, err)
	}
}

func TestObjectionPDFSelectionTreatsMissingPackageAsEmptyList(t *testing.T) {
	const id = "JOB-20261006T130000-abcdef0123456789"
	s := &server{
		workspacesDir: filepath.Join(t.TempDir(), "expedientes"),
		sessions:      map[string]session{"test-session": {username: "tester", expiresAt: time.Now().Add(time.Hour)}},
	}
	job := stagedJob{ID: id, Status: "PROCESSED", IsObjections: true, ObjectionRows: []objectionRecord{{Tramite: "9900501", Patient: "PACIENTE FICTICIO", PatientFolder: "PACIENTE_FICTICIO_9900501", PlanillaID: 501, Matched: true}}}
	persistTestObjectionJob(t, s, job)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/objeciones/pdfs/seleccion/"+id, nil)
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
	response := httptest.NewRecorder()
	s.objectionPDFSelection(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("empty PDF list status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Documents []objectionPackagePDF `json:"documents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Documents == nil || len(result.Documents) != 0 {
		t.Fatalf("empty PDF list = %#v; want a non-nil empty list", result.Documents)
	}
}

func persistTestObjectionJob(t *testing.T, s *server, job stagedJob) {
	t.Helper()
	if err := os.MkdirAll(s.jobRoot(job.ID), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.jobRoot(job.ID), "job.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestObjectionAnnexNamesAreCanonicalAndCollisionSafe(t *testing.T) {
	base, err := objectionAnnexBase("FICHA_TECNICA", "")
	if err != nil || base != "FICHA_TECNICA" {
		t.Fatalf("objectionAnnexBase() = %q, %v", base, err)
	}
	custom, err := objectionAnnexBase("OTRO", "Factura técnica Ñandú")
	if err != nil || custom != "FACTURA_TECNICA_NANDU" {
		t.Fatalf("custom annex base = %q, %v", custom, err)
	}
	directory := t.TempDir()
	first, err := nextSafeAnnexName(directory, custom)
	if err != nil || first != "FACTURA_TECNICA_NANDU.pdf" {
		t.Fatalf("first annex name = %q, %v", first, err)
	}
	if err := os.WriteFile(filepath.Join(directory, first), []byte("pdf ficticio"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := nextSafeAnnexName(directory, custom)
	if err != nil || second != "FACTURA_TECNICA_NANDU_1.pdf" {
		t.Fatalf("collision-safe annex name = %q, %v", second, err)
	}
}

func TestInstallObjectionWorkspaceCopiesOnlyObjectedPlanillaFiles(t *testing.T) {
	root := t.TempDir()
	server := &server{workspacesDir: filepath.Join(root, "expedientes")}
	source := stagedJob{ID: "JOB-20261005T120000-abcdef0123456789", Status: "PROCESSED", Month: "09", Year: "2026", Service: "EMERGENCIA"}
	patient := "PACIENTE_FICTICIO"
	sourcePackage := filepath.Join(server.jobRoot(source.ID), "trabajo", packageFolderName(&source))
	sourcePatient := filepath.Join(sourcePackage, "4. EXPEDIENTES", patient)
	if err := os.MkdirAll(sourcePatient, 0700); err != nil {
		t.Fatal(err)
	}
	objectedPDF := []byte("%PDF-1.4\ncontenido objetado ficticio\n")
	approvedPDF := []byte("%PDF-1.4\ncontenido aprobado ficticio\n")
	for name, content := range map[string][]byte{
		"objetado.pdf": objectedPDF, "aprobado.pdf": approvedPDF, "C_COBERTURA.pdf": []byte("%PDF-1.4\ncobertura ficticia\n"),
	} {
		if err := os.WriteFile(filepath.Join(sourcePatient, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	source.DocumentPlanillas = map[string]int64{
		"4. EXPEDIENTES/PACIENTE_FICTICIO/objetado.pdf":    501,
		"4. EXPEDIENTES/PACIENTE_FICTICIO/aprobado.pdf":    502,
		"4. EXPEDIENTES/PACIENTE_FICTICIO/C_COBERTURA.pdf": 501,
	}
	folder := objectionPatientFolder("0000000501", "PACIENTE FICTICIO", "9900001")
	job := stagedJob{
		ID: "JOB-20261005T120001-abcdef0123456789", Status: "PROCESSED", Month: source.Month, Year: source.Year, Service: source.Service,
		IsObjections: true, ObjectionSourceID: source.ID,
		ObjectionRows: []objectionRecord{{Tramite: "9900001", PlanillaID: 501, Cedula: "0000000501", Patient: "PACIENTE FICTICIO", PatientFolder: folder, SourcePatientFolder: patient, Matched: true}},
	}
	if err := server.installObjectionWorkspace(&job, source, job.ObjectionRows); err != nil {
		t.Fatalf("installObjectionWorkspace() error = %v", err)
	}
	packageRoot := filepath.Join(server.jobRoot(job.ID), "trabajo", packageFolderName(&job))
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", folder, "objetado.pdf")); err != nil {
		t.Fatal("expected objected PDF in derived workspace:", err)
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", folder, "aprobado.pdf")); !os.IsNotExist(err) {
		t.Fatalf("approved PDF was copied into derived workspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", folder, "C_COBERTURA.pdf")); err != nil {
		t.Fatal("expected copied coverage PDF:", err)
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "5. ANEXOS")); !os.IsNotExist(err) {
		t.Fatalf("optional annex directory should not be created before an attachment: %v", err)
	}
	for _, name := range []string{"OFICIO_LIQUIDACION_MSP.pdf", "I_LIQUIDACION.pdf", "1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf", objectionMatrixFilename(job)} {
		if _, err := os.Stat(filepath.Join(packageRoot, name)); !os.IsNotExist(err) {
			t.Fatalf("header %s should be uploaded after creation: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(server.jobRoot(source.ID), "trabajo", packageFolderName(&source), "4. EXPEDIENTES", patient, "aprobado.pdf")); err != nil {
		t.Fatalf("source workspace was changed: %v", err)
	}
	if job.DocumentPlanillas[filepath.ToSlash(filepath.Join("4. EXPEDIENTES", folder, "objetado.pdf"))] != 501 {
		t.Fatal("derived document lost its PDI_ID mapping")
	}
}

func TestSyncObjectionDocumentsAddsNewPDFsAndPreservesObjectionWork(t *testing.T) {
	root := t.TempDir()
	s := &server{workspacesDir: filepath.Join(root, "expedientes"), sessions: map[string]session{"sync-test": {expiresAt: time.Now().Add(time.Hour)}}}
	source, job, sourcePatient, targetPatient := objectionSyncFixture(t, s, "9900711")
	newSourcePDF := filepath.Join(sourcePatient, "HCU_010.pdf")
	currentSourcePDF := filepath.Join(sourcePatient, "HCU_006.pdf")
	newContents := []byte("%PDF-1.4\nformulario cargado después\n")
	if err := os.WriteFile(newSourcePDF, newContents, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(newSourcePDF, currentSourcePDF); err != nil {
		t.Fatal(err)
	}
	delete(source.DocumentPlanillas, "4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_010.pdf")
	source.DocumentPlanillas["4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_006.pdf"] = 711
	source.Renames = map[string]string{"4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_010.pdf": "4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_006.pdf"}
	persistTestObjectionJob(t, s, source)

	job, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	job.ObjectionRows[0].Posture = "RECHAZA"
	selectedPath := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", targetPatient, "HCU_008.pdf"))
	job.ObjectionPDFSelection = map[string]bool{selectedPath: true}
	annex := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job), "5. ANEXOS", targetPatient, "FACTURA_COMPRA.pdf")
	annexContents := []byte("anexo guardado en Objeciones")
	if err := atomicWritePrivateFile(annex, annexContents); err != nil {
		t.Fatal(err)
	}
	if err := s.saveStagedJob(job); err != nil {
		t.Fatal(err)
	}

	response := callObjectionSync(t, s, job.ID, "{}")
	if response.Code != http.StatusOK {
		t.Fatalf("sync status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Added     int                     `json:"documentos_agregados"`
		Conflicts []objectionSyncConflict `json:"conflictos"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || len(result.Conflicts) != 0 {
		t.Fatalf("sync result = %+v; want one added PDF and no conflicts", result)
	}
	updated, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ObjectionRows[0].Posture != "RECHAZA" || !updated.ObjectionPDFSelection[selectedPath] {
		t.Fatalf("sync changed saved posture or PDF selection: %+v", updated.ObjectionRows[0])
	}
	newTargetPDF := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&updated), "4. EXPEDIENTES", targetPatient, "HCU_006.pdf")
	got, err := os.ReadFile(newTargetPDF)
	if err != nil || !bytes.Equal(got, newContents) {
		t.Fatalf("new PDF was not copied correctly: contents=%q error=%v", got, err)
	}
	if objectionZIPIncludesFile(updated, filepath.ToSlash(filepath.Join("4. EXPEDIENTES", targetPatient, "HCU_010.pdf"))) {
		t.Fatal("new optional clinical PDF should remain unselected for the objection ZIP")
	}
	if got, err := os.ReadFile(annex); err != nil || !bytes.Equal(got, annexContents) {
		t.Fatalf("sync changed the objection annex: contents=%q error=%v", got, err)
	}
	if got, err := os.ReadFile(currentSourcePDF); err != nil || !bytes.Equal(got, newContents) {
		t.Fatalf("sync changed the first-ingress source: contents=%q error=%v", got, err)
	}
}

func TestSyncObjectionDocumentsRequiresChoiceBeforeUpdatingChangedPDF(t *testing.T) {
	root := t.TempDir()
	s := &server{workspacesDir: filepath.Join(root, "expedientes"), sessions: map[string]session{"sync-test": {expiresAt: time.Now().Add(time.Hour)}}}
	source, job, sourcePatient, targetPatient := objectionSyncFixture(t, s, "9900712")
	sourcePDF := filepath.Join(sourcePatient, "HCU_008.pdf")
	newContents := []byte("%PDF-1.4\nversión corregida en primer ingreso\n")
	if err := os.WriteFile(sourcePDF, newContents, 0600); err != nil {
		t.Fatal(err)
	}
	persistTestObjectionJob(t, s, source)

	targetPDF := filepath.Join(s.jobRoot(job.ID), "trabajo", packageFolderName(&job), "4. EXPEDIENTES", targetPatient, "HCU_008.pdf")
	oldContents, err := os.ReadFile(targetPDF)
	if err != nil {
		t.Fatal(err)
	}
	job, err = s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	selectedPath := filepath.ToSlash(filepath.Join("4. EXPEDIENTES", targetPatient, "HCU_008.pdf"))
	job.ObjectionRows[0].Posture = "ACEPTA"
	job.ObjectionPDFSelection = map[string]bool{selectedPath: true}
	if err := s.saveStagedJob(job); err != nil {
		t.Fatal(err)
	}

	response := callObjectionSync(t, s, job.ID, "{}")
	if response.Code != http.StatusOK {
		t.Fatalf("detect changed source status = %d, body = %s", response.Code, response.Body.String())
	}
	var conflicts struct {
		Items []objectionSyncConflict `json:"conflictos"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &conflicts); err != nil {
		t.Fatal(err)
	}
	if len(conflicts.Items) != 1 || conflicts.Items[0].SourcePath != "4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_008.pdf" {
		t.Fatalf("sync conflicts = %+v; want the changed HCU_008.pdf", conflicts.Items)
	}
	if got, err := os.ReadFile(targetPDF); err != nil || !bytes.Equal(got, oldContents) {
		t.Fatalf("detection overwrote the objection copy: contents=%q error=%v", got, err)
	}

	body, _ := json.Marshal(map[string]any{"source_path": conflicts.Items[0].SourcePath, "actualizar_version": true})
	response = callObjectionSync(t, s, job.ID, string(body))
	if response.Code != http.StatusOK {
		t.Fatalf("explicit update status = %d, body = %s", response.Code, response.Body.String())
	}
	if got, err := os.ReadFile(targetPDF); err != nil || !bytes.Equal(got, newContents) {
		t.Fatalf("explicit update did not copy the new source: contents=%q error=%v", got, err)
	}
	updated, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ObjectionRows[0].Posture != "ACEPTA" || !updated.ObjectionPDFSelection[selectedPath] {
		t.Fatalf("explicit update changed saved posture or selection: %+v", updated.ObjectionRows[0])
	}
	versions, err := os.ReadDir(filepath.Join(s.jobRoot(job.ID), "fuentes", "versiones"))
	if err != nil || len(versions) != 1 {
		t.Fatalf("previous objection PDF was not backed up: entries=%v error=%v", versions, err)
	}
	backup, err := os.ReadFile(filepath.Join(s.jobRoot(job.ID), "fuentes", "versiones", versions[0].Name()))
	if err != nil || !bytes.Equal(backup, oldContents) {
		t.Fatalf("backup does not contain previous objection PDF: contents=%q error=%v", backup, err)
	}
}

func objectionSyncFixture(t *testing.T, s *server, tramite string) (stagedJob, stagedJob, string, string) {
	t.Helper()
	source := stagedJob{
		ID: "JOB-20261007T100000-abcdef0123456789", Status: "PROCESSED", Service: "AMBULATORIO", Month: "08", Year: "2026",
		DocumentPlanillas: map[string]int64{
			"4. EXPEDIENTES/PACIENTE_SINCRONIZACION/HCU_008.pdf":     711,
			"4. EXPEDIENTES/PACIENTE_SINCRONIZACION/C_COBERTURA.pdf": 711,
		},
	}
	job := stagedJob{
		ID: "JOB-20261007T100001-abcdef0123456789", Status: "PROCESSED", Service: source.Service, Month: source.Month, Year: source.Year,
		IsObjections: true, ObjectionSourceID: source.ID,
		ObjectionRows: []objectionRecord{{Tramite: tramite, PlanillaID: 711, Cedula: "0000000711", Patient: "PACIENTE SINCRONIZACION", PatientFolder: objectionPatientFolder("0000000711", "PACIENTE SINCRONIZACION", tramite), SourcePatientFolder: "PACIENTE_SINCRONIZACION", Matched: true}},
	}
	sourcePackage := filepath.Join(s.jobRoot(source.ID), "trabajo", packageFolderName(&source))
	sourcePatient := filepath.Join(sourcePackage, "4. EXPEDIENTES", "PACIENTE_SINCRONIZACION")
	if err := os.MkdirAll(sourcePatient, 0700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{
		"HCU_008.pdf":     []byte("%PDF-1.4\nversión inicial\n"),
		"C_COBERTURA.pdf": []byte("%PDF-1.4\ncobertura\n"),
	} {
		if err := os.WriteFile(filepath.Join(sourcePatient, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	persistTestObjectionJob(t, s, source)
	if err := s.installObjectionWorkspace(&job, source, job.ObjectionRows); err != nil {
		t.Fatalf("installObjectionWorkspace() error = %v", err)
	}
	return source, job, sourcePatient, job.ObjectionRows[0].PatientFolder
}

func callObjectionSync(t *testing.T, s *server, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/objeciones/sincronizar/"+id, strings.NewReader(body))
	request.AddCookie(&http.Cookie{Name: "folio_session", Value: "sync-test"})
	response := httptest.NewRecorder()
	s.syncObjectionDocuments(response, request)
	return response
}

func TestInstallObjectionRowsAppendsForgottenTransactionWithoutChangingExistingRows(t *testing.T) {
	root := t.TempDir()
	packageRoot := filepath.Join(root, "trabajo", "EMERGENCIA_AGOSTO_2026_OBJECIONES")
	patientRoot := filepath.Join(root, "origen", "4. EXPEDIENTES")
	existingFolder := objectionPatientFolder("0000000501", "PACIENTE FICTICIO", "9900001")
	newSourceFolder := "PACIENTE_FICTICIO_DOS"
	newFolder := objectionPatientFolder("0000000502", "PACIENTE FICTICIO DOS", "9900002")
	existingPDF := filepath.Join(packageRoot, "4. EXPEDIENTES", existingFolder, "existente.pdf")
	if err := os.MkdirAll(filepath.Dir(existingPDF), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingPDF, []byte("%PDF-1.4\nseleccionado anteriormente\n"), 0600); err != nil {
		t.Fatal(err)
	}
	newSourcePDF := filepath.Join(patientRoot, newSourceFolder, "segundo.pdf")
	if err := os.MkdirAll(filepath.Dir(newSourcePDF), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSourcePDF, []byte("%PDF-1.4\ntrámite agregado\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := stagedJob{DocumentPlanillas: map[string]int64{"4. EXPEDIENTES/PACIENTE_FICTICIO_DOS/segundo.pdf": 502}}
	job := stagedJob{DocumentPlanillas: map[string]int64{"4. EXPEDIENTES/" + existingFolder + "/existente.pdf": 501}}
	row := objectionRecord{Tramite: "9900002", Cedula: "0000000502", Patient: "PACIENTE FICTICIO DOS", PatientFolder: newFolder, SourcePatientFolder: newSourceFolder, PlanillaID: 502, Matched: true}
	if err := installObjectionRows(packageRoot, patientRoot, source, []objectionRecord{row}, &job); err != nil {
		t.Fatalf("installObjectionRows() append error = %v", err)
	}
	if _, err := os.Stat(existingPDF); err != nil {
		t.Fatalf("existing selected transaction changed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "4. EXPEDIENTES", newFolder, "segundo.pdf")); err != nil {
		t.Fatalf("forgotten transaction was not appended: %v", err)
	}
	if job.DocumentPlanillas[filepath.ToSlash(filepath.Join("4. EXPEDIENTES", existingFolder, "existente.pdf"))] != 501 {
		t.Fatal("existing transaction mapping changed")
	}
	if job.DocumentPlanillas[filepath.ToSlash(filepath.Join("4. EXPEDIENTES", newFolder, "segundo.pdf"))] != 502 {
		t.Fatal("appended transaction mapping was not saved")
	}
}

func testMacroEnabledWorkbook(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for name, content := range map[string]string{
		"[Content_Types].xml": "<?xml version=\"1.0\"?><Types><Override PartName=\"/xl/workbook.xml\" ContentType=\"application/vnd.ms-excel.sheet.macroEnabled.main+xml\"/></Types>",
		"xl/workbook.xml":     "<?xml version=\"1.0\"?><workbook/>",
	} {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestObjectionCloseoutRequiresHeadersResponseAndCoverageButNotAnnex(t *testing.T) {
	root := t.TempDir()
	job := stagedJob{
		IsObjections: true, Service: "AMBULATORIO", Month: "08", Year: "2026",
		ObjectionRows: []objectionRecord{{Tramite: "9900201", Patient: "PACIENTE FICTICIO", PatientFolder: "PACIENTE_FICTICIO_9900201", Matched: true}},
	}
	if err := validateObjectionCloseout(job, root); err == nil || !strings.Contains(err.Error(), "OFICIO_LIQUIDACION_MSP.pdf") {
		t.Fatalf("expected missing header to block ZIP; got %v", err)
	}
	for _, name := range []string{"OFICIO_LIQUIDACION_MSP.pdf", "I_LIQUIDACION.pdf", "1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("%PDF-1.4\nficticio\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, objectionMatrixFilename(job)), testMacroEnabledWorkbook(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateObjectionCloseout(job, root); err == nil || !strings.Contains(err.Error(), "ACEPTA o RECHAZA") {
		t.Fatalf("expected missing posture to block ZIP; got %v", err)
	}
	job.ObjectionRows[0].Posture = "ACEPTA"
	patientRoot := filepath.Join(root, "4. EXPEDIENTES", job.ObjectionRows[0].PatientFolder)
	if err := os.MkdirAll(patientRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(patientRoot, "P_INDIVIDUAL.pdf"), []byte("%PDF-1.4\nficticio\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateObjectionCloseout(job, root); err == nil || !strings.Contains(err.Error(), "C_COBERTURA.pdf") {
		t.Fatalf("expected missing coverage to block ZIP; got %v", err)
	}
	if err := os.WriteFile(filepath.Join(patientRoot, "C_COBERTURA.pdf"), []byte("%PDF-1.4\ncobertura ficticia\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateObjectionCloseout(job, root); err != nil {
		t.Fatalf("package without optional annexes should pass closeout: %v", err)
	}
}
