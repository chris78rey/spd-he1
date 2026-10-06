package main

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	for _, name := range []string{"I_LIQUIDACION.pdf", "1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf", objectionMatrixFilename(job)} {
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
	if err := validateObjectionCloseout(job, root); err == nil || !strings.Contains(err.Error(), "I_LIQUIDACION.pdf") {
		t.Fatalf("expected missing header to block ZIP; got %v", err)
	}
	for _, name := range []string{"I_LIQUIDACION.pdf", "1. OFICIO DE PAGO.pdf", "2. PLANILLA CONSOLIDADA.pdf"} {
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
