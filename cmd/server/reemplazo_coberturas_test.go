package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func coverageReplacementTestPDF(t *testing.T, text string) []byte {
	t.Helper()
	stream := []byte(fmt.Sprintf("BT /F1 12 Tf 72 700 Td (%s) Tj ET", text))
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
		append(append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(stream))), stream...), []byte("endstream")...),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", index+1)
		out.Write(object)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

type coverageReplacementFixture struct {
	s        *server
	job      stagedJob
	active   string
	relative string
	prepared string
	request  coverageReplacementRequest
}

func newCoverageReplacementFixture(t *testing.T) coverageReplacementFixture {
	t.Helper()
	dataDir := t.TempDir()
	s := &server{dataDir: dataDir, workspacesDir: filepath.Join(dataDir, "expedientes"), stagingDir: filepath.Join(dataDir, "staging")}
	job := stagedJob{ID: "WORK-HOSPITALIZACION-202609", Service: "HOSPITALIZACION", Month: "09", Year: "2026"}
	relative := "4. EXPEDIENTES/PACIENTE_PRUEBA/C_COBERTURA.pdf"
	job.ExternalPDFs = []workspaceDocument{{ID: "fusion-old", StoredName: "fusion-old.pdf", OriginalName: "C_COBERTURA.pdf", RelativePath: relative, PlanillaID: 77,
		CoverageItems: []coverageDocumentMetadata{{PlanillaID: 77, Cedula: "1712345678", Date: "2026-09-02"}}}}
	job.Replacements = map[string]string{relative: "fusion-old.pdf"}
	root := s.jobRoot(job.ID)
	packageRoot := filepath.Join(root, "trabajo", packageFolderName(&job))
	if err := os.MkdirAll(filepath.Join(packageRoot, "4. EXPEDIENTES", "PACIENTE_PRUEBA"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fuentes"), 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(job)
	if err := os.WriteFile(filepath.Join(root, "job.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(packageRoot, filepath.FromSlash(relative))
	if err := os.WriteFile(active, coverageReplacementTestPDF(t, "HOJA LOCAL 1712345678"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared := filepath.Join(dataDir, "reemplazo_coberturas", "HOSPITALIZACION_SEPTIEMBRE_2026", "20261118T000000Z")
	if err := os.MkdirAll(filepath.Join(prepared, "paginas"), 0700); err != nil {
		t.Fatal(err)
	}
	pages := make([]coverageReplacementPage, 0, 2)
	for index, date := range []string{"20260902", "20260908"} {
		name := "paginas/1712345678_" + date + ".pdf"
		file := filepath.Join(prepared, filepath.FromSlash(name))
		if err := os.WriteFile(file, coverageReplacementTestPDF(t, fmt.Sprintf("HOJA PORTAL %d 1712345678", index+1)), 0600); err != nil {
			t.Fatal(err)
		}
		sum, _ := fileSHA256(file)
		pages = append(pages, coverageReplacementPage{File: name, SHA256: sum, Cedula: "1712345678"})
	}
	originalSum, _ := fileSHA256(active)
	request := coverageReplacementRequest{JobID: job.ID, Prepared: prepared, Sheets: []coverageReplacementSheet{{
		Path: relative, OriginalSHA256: originalSum, OriginalPages: 2, Pages: pages,
	}}}
	return coverageReplacementFixture{s: s, job: job, active: active, relative: relative, prepared: prepared, request: request}
}

func TestCoverageReplacementPlacesPortalPDFAndKeepsBackup(t *testing.T) {
	f := newCoverageReplacementFixture(t)
	before, _ := os.ReadFile(f.active)

	result := f.s.applyCoverageReplacement(f.request, coverageReplacementResult{})

	if result.Error != "" || len(result.Sheets) != 1 || result.Sheets[0].Status != "REEMPLAZADA" {
		t.Fatalf("result = %+v", result)
	}
	if !strings.HasPrefix(result.OracleSync, "falló") { // sin Oracle en pruebas
		t.Fatalf("OracleSync = %q", result.OracleSync)
	}
	after, _ := os.ReadFile(f.active)
	if pages, err := api.PageCountFile(f.active); err != nil || pages != 2 || bytes.Equal(after, before) {
		t.Fatalf("el expediente no quedó con la hoja del portal de 2 páginas: %d %v", pages, err)
	}
	text, _ := extractPDFText(f.active)
	if !strings.Contains(text, "HOJA PORTAL 1") || !strings.Contains(text, "HOJA PORTAL 2") || strings.Index(text, "HOJA PORTAL 1") > strings.Index(text, "HOJA PORTAL 2") {
		t.Fatalf("páginas del portal ausentes o fuera de orden: %q", text)
	}
	if backup, err := os.ReadFile(result.Sheets[0].Backup); err != nil || !bytes.Equal(backup, before) {
		t.Fatalf("respaldo de la hoja anterior ausente o distinto: %v", err)
	}
	if _, err := os.Stat(f.active + ".folio-reemplazo-backup"); !os.IsNotExist(err) {
		t.Fatal("quedó el respaldo temporal en el expediente")
	}
	job, err := f.s.loadStagedJob(f.job.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored := result.Sheets[0].Stored
	if job.Replacements[f.relative] != stored || job.ExternalPDFs[0].StoredName != stored || job.ExternalPDFs[0].ID != "fusion-old" || len(job.ExternalPDFs[0].ID) > 24 {
		t.Fatalf("job no actualizado: replacements=%v external=%+v", job.Replacements, job.ExternalPDFs[0])
	}
	if len(job.ExternalPDFs[0].CoverageItems) != 1 || job.ExternalPDFs[0].PlanillaID != 77 {
		t.Fatal("se perdió la metadata de cobertura de la hoja")
	}
	if storedBytes, err := os.ReadFile(filepath.Join(f.s.jobSourcesDir(f.job.ID), "externos", stored)); err != nil || !bytes.Equal(storedBytes, after) {
		t.Fatalf("PDF del portal no guardado en externos: %v", err)
	}
}

func TestCoverageReplacementSkipsSheetChangedSincePreparation(t *testing.T) {
	f := newCoverageReplacementFixture(t)
	if err := os.WriteFile(f.active, coverageReplacementTestPDF(t, "EDITADA 1712345678"), 0600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(f.active)

	result := f.s.applyCoverageReplacement(f.request, coverageReplacementResult{})

	if len(result.Sheets) != 1 || result.Sheets[0].Status != "OMITIDA" || result.OracleSync != "no necesaria" {
		t.Fatalf("result = %+v", result)
	}
	if after, _ := os.ReadFile(f.active); !bytes.Equal(after, before) {
		t.Fatal("se modificó una hoja que debía omitirse")
	}
}

func TestCoverageReplacementRejectsPathsOutsidePreparation(t *testing.T) {
	f := newCoverageReplacementFixture(t)
	escaped := f.request
	escaped.Sheets = append([]coverageReplacementSheet(nil), f.request.Sheets...)
	escaped.Sheets[0].Pages = append([]coverageReplacementPage(nil), f.request.Sheets[0].Pages...)
	escaped.Sheets[0].Pages[1].File = "../../../../expedientes/otro.pdf"
	if result := f.s.applyCoverageReplacement(escaped, coverageReplacementResult{}); result.Sheets[0].Status != "OMITIDA" {
		t.Fatalf("página fuera de la preparación aceptada: %+v", result)
	}
	wrongCedula := f.request
	wrongCedula.Sheets = append([]coverageReplacementSheet(nil), f.request.Sheets...)
	wrongCedula.Sheets[0].Pages = append([]coverageReplacementPage(nil), f.request.Sheets[0].Pages...)
	wrongCedula.Sheets[0].Pages[0].Cedula = "1799999999"
	if result := f.s.applyCoverageReplacement(wrongCedula, coverageReplacementResult{}); result.Sheets[0].Status != "OMITIDA" {
		t.Fatalf("página con otra cédula aceptada: %+v", result)
	}
	badPath := f.request
	badPath.Sheets = append([]coverageReplacementSheet(nil), f.request.Sheets...)
	badPath.Sheets[0].Path = "4. EXPEDIENTES/PACIENTE_PRUEBA/HCU_007.pdf"
	if result := f.s.applyCoverageReplacement(badPath, coverageReplacementResult{}); result.Sheets[0].Status != "OMITIDA" {
		t.Fatalf("ruta que no es C_COBERTURA aceptada: %+v", result)
	}
	outside := f.request
	outside.Prepared = t.TempDir()
	if result := f.s.applyCoverageReplacement(outside, coverageReplacementResult{}); result.Error == "" {
		t.Fatal("carpeta preparada fuera de reemplazo_coberturas aceptada")
	}
}

func TestCoverageReplacementQueueWritesResultAndRemovesRequest(t *testing.T) {
	f := newCoverageReplacementFixture(t)
	requests := filepath.Join(f.s.coverageReplacementRoot(), "solicitudes")
	if err := os.MkdirAll(requests, 0700); err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(requests, "piloto.json")
	raw, _ := json.Marshal(f.request)
	if err := os.WriteFile(requestPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	f.s.processCoverageReplacementRequest(requestPath)

	if _, err := os.Stat(requestPath); !os.IsNotExist(err) {
		t.Fatal("la solicitud no se retiró de la cola")
	}
	var result coverageReplacementResult
	data, err := os.ReadFile(filepath.Join(f.s.coverageReplacementRoot(), "resultados", "piloto.json"))
	if err != nil || json.Unmarshal(data, &result) != nil {
		t.Fatalf("resultado no escrito: %v", err)
	}
	if result.JobID != f.job.ID || len(result.Sheets) != 1 || result.Sheets[0].Status != "REEMPLAZADA" {
		t.Fatalf("resultado = %+v", result)
	}
}

func TestCoverageReplacementRetryRestoresIDsOnlyForToolReplacedSheets(t *testing.T) {
	f := newCoverageReplacementFixture(t)
	if result := f.s.applyCoverageReplacement(f.request, coverageReplacementResult{}); result.Sheets[0].Status != "REEMPLAZADA" {
		t.Fatalf("apply = %+v", result)
	}
	job, _ := f.s.loadStagedJob(f.job.ID)
	job.ExternalPDFs[0].ID = "cobertura-portal-0123456789abcdef" // estado de la primera versión con ID largo
	job.ExternalPDFs = append(job.ExternalPDFs, workspaceDocument{ID: "otro", StoredName: "fusion-x.pdf", RelativePath: "4. EXPEDIENTES/OTRO/C_COBERTURA.pdf"})
	if err := f.s.saveStagedJob(job); err != nil {
		t.Fatal(err)
	}
	request := coverageReplacementRequest{JobID: f.job.ID, RetryOracle: true, RestoreIDs: map[string]string{
		f.relative:                            "fusion-old",
		"4. EXPEDIENTES/OTRO/C_COBERTURA.pdf": "fusion-nuevo",
		"4. EXPEDIENTES/X/HCU_007.pdf":        "fusion-y",
	}}

	result := f.s.retryCoverageReplacementOracle(request, coverageReplacementResult{})

	statuses := map[string]string{}
	for _, sheet := range result.Sheets {
		statuses[sheet.Path] = sheet.Status
	}
	if statuses[f.relative] != "ID_RESTAURADO" || statuses["4. EXPEDIENTES/OTRO/C_COBERTURA.pdf"] != "OMITIDA" || statuses["4. EXPEDIENTES/X/HCU_007.pdf"] != "OMITIDA" {
		t.Fatalf("statuses = %v", statuses)
	}
	job, _ = f.s.loadStagedJob(f.job.ID)
	if job.ExternalPDFs[0].ID != "fusion-old" || job.ExternalPDFs[1].ID != "otro" {
		t.Fatalf("IDs = %q, %q", job.ExternalPDFs[0].ID, job.ExternalPDFs[1].ID)
	}
	if !strings.HasPrefix(result.OracleSync, "falló") { // sin Oracle en pruebas
		t.Fatalf("OracleSync = %q", result.OracleSync)
	}
}
