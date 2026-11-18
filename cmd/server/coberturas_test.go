package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateIndividualCoverageRequiresSession(t *testing.T) {
	s := &server{sessions: make(map[string]session)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/coberturas/individual", strings.NewReader(`{"cedula":"1720000000","fecha_consulta":"2026-10-12"}`))
	response := httptest.NewRecorder()

	s.generateIndividualCoverage(response, req)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("generateIndividualCoverage() status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestGenerateIndividualCoverageRejectsInvalidInputsBeforePortal(t *testing.T) {
	s := &server{sessions: map[string]session{"test-session": {username: "TEST", expiresAt: time.Now().Add(time.Hour)}}}
	for _, body := range []string{
		`{"cedula":"123","fecha_consulta":"2026-10-12"}`,
		`{"cedula":"1720000000","fecha_consulta":"2026-02-30"}`,
		`not-json`,
	} {
		t.Run(body, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/coberturas/individual", strings.NewReader(body))
			req.AddCookie(&http.Cookie{Name: "folio_session", Value: "test-session"})
			response := httptest.NewRecorder()

			s.generateIndividualCoverage(response, req)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("generateIndividualCoverage() status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
		})
	}
}

func TestNormalizeCoverageDateRequiresRealISODate(t *testing.T) {
	if got, ok := normalizeCoverageDate("2026-02-28"); !ok || got != "2026-02-28" {
		t.Fatalf("normalizeCoverageDate(valid) = (%q, %t)", got, ok)
	}
	for _, value := range []string{"2026-02-30", "2026/02/28", "28-02-2026", ""} {
		if got, ok := normalizeCoverageDate(value); ok {
			t.Errorf("normalizeCoverageDate(%q) = (%q, true), want invalid date", value, got)
		}
	}
}

func TestCoverageMembersAtUsesChosenDateForMinorAndReferences(t *testing.T) {
	row := coveragePlanilla{
		CareUntil:  "2026-08-31",
		Cedula:     "1720000000",
		Minor:      "S",
		Dependent1: "1711111111",
		Dependent2: "1720000000",
	}

	got := coverageMembersAt(row, "2026-07-14")
	if len(got) != 2 {
		t.Fatalf("coverageMembersAt() returned %d members, want 2: %#v", len(got), got)
	}
	if got[0].Cedula != row.Cedula || got[1].Cedula != row.Dependent1 {
		t.Fatalf("coverageMembersAt() cédulas = %#v, want patient and unique referente", got)
	}
	for _, member := range got {
		if member.Fecha != "2026-07-14" {
			t.Errorf("member %s has date %q, want selected date", member.Cedula, member.Fecha)
		}
	}
}

func TestMissingCoverageMembersForDateOnlyReturnsMembersWithoutThatDate(t *testing.T) {
	const chosenDate = "2026-07-14"
	row := coveragePlanilla{
		ID:         42,
		CareUntil:  "2026-08-31",
		Cedula:     "1720000000",
		Minor:      "S",
		Dependent1: "1711111111",
		Dependent2: "1700000000",
	}
	job := stagedJob{
		ID: "JOB-20261008T120000-abcdef0123456789",
		ExternalPDFs: []workspaceDocument{
			{PlanillaID: row.ID, RelativePath: filepath.Join("4. EXPEDIENTES", "PACIENTE", "C_COBERTURA.pdf"), CoverageCedula: row.Cedula, CoverageDate: chosenDate},
			{PlanillaID: row.ID, RelativePath: filepath.Join("4. EXPEDIENTES", "PACIENTE", "C_COBERTURA_1.pdf"), CoverageCedula: row.Dependent1, CoverageDate: chosenDate},
			{PlanillaID: row.ID, RelativePath: filepath.Join("4. EXPEDIENTES", "PACIENTE", "C_COBERTURA_2.pdf"), CoverageCedula: row.Dependent2, CoverageDate: row.CareUntil},
		},
	}
	s := &server{workspacesDir: t.TempDir()}

	got := missingCoverageMembersForDate(s, job, row, chosenDate)
	if len(got) != 1 || got[0].Cedula != row.Dependent2 || got[0].Fecha != chosenDate {
		t.Fatalf("missingCoverageMembersForDate() = %#v, want only referente %s at %s", got, row.Dependent2, chosenDate)
	}
}

func TestCoverageHasDateUsesDateAssociatedWithSavedPDFs(t *testing.T) {
	row := coveragePlanilla{
		CareUntil:     "2026-08-31",
		CoverageDates: []string{"2026-07-14", "2026-08-31"},
	}
	if !coverageHasDate(row, "2026-07-14") || !coverageHasDate(row, row.CareUntil) {
		t.Fatalf("coverageHasDate() should find both selected and normal dates: %#v", row.CoverageDates)
	}
	if coverageHasDate(row, "2026-07-15") {
		t.Fatalf("coverageHasDate() matched an unsaved date: %#v", row.CoverageDates)
	}
}

func TestCoverageScriptFailureTag(t *testing.T) {
	cases := map[string]string{
		"LIMITE_SITIO: Ha superado el límite de consultas\n": coverageTagSiteLimit,
		"MENSAJE_SITIO: otro aviso\n":                        coverageTagSiteMsg,
		"Fallo técnico: Chrome no abrió el puerto\n":         "TECNICO",
		"": "TECNICO",
	}
	for output, want := range cases {
		if got := coverageScriptFailureTag([]byte(output)); got != want {
			t.Fatalf("coverageScriptFailureTag(%q) = %q, want %q", output, got, want)
		}
	}
}

// Prueba de integración contra el portal real del MSP. Se omite salvo que se defina
// COBERTURA_LIVE_CEDULA (10 dígitos); COBERTURA_LIVE_FECHA (YYYY-MM-DD) es opcional y usa hoy.
// Ejemplo:
//
//	COBERTURA_LIVE_CEDULA=1712345678 COBERTURA_CHROME_NO_SANDBOX=true go test ./cmd/server -run LivePortal -v
func TestGenerateIndividualCoverageLivePortal(t *testing.T) {
	cedula := strings.TrimSpace(os.Getenv("COBERTURA_LIVE_CEDULA"))
	if cedula == "" {
		t.Skip("define COBERTURA_LIVE_CEDULA para consultar el portal real del MSP")
	}
	fecha := strings.TrimSpace(os.Getenv("COBERTURA_LIVE_FECHA"))
	if fecha == "" {
		fecha = time.Now().Format("2006-01-02")
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil { // el script se resuelve desde la raíz del proyecto
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	s := &server{sessions: map[string]session{"live": {username: "TEST", expiresAt: time.Now().Add(time.Hour)}}}
	body := `{"cedula":"` + cedula + `","fecha_consulta":"` + fecha + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/coberturas/individual", strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "folio_session", Value: "live"})
	response := httptest.NewRecorder()

	s.generateIndividualCoverage(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type = %q, want application/pdf", got)
	}
	if !bytes.HasPrefix(response.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("la respuesta no empieza con %%PDF-")
	}
	t.Logf("PDF del portal recibido: %d bytes", response.Body.Len())
}

func TestCoverageSheetHasDateRequiresRequestedDate(t *testing.T) {
	cases := []struct {
		text, date string
		want       bool
	}{
		{"Fecha de Cobertura de Seguro de Salud: 01 octubre 2026", "2026-10-01", true},
		{"Fecha de Cobertura de Seguro de Salud: 1 October 2026", "2026-10-01", true},
		{"Fecha de Cobertura de Seguro de Salud:   October 01, 2026", "2026-10-01", false}, // formato del generador local retirado
		{"Fecha de Cobertura de Seguro de Salud: October 1, 2026", "2026-10-01", true},
		{"Fecha de Cobertura de Seguro de Salud: 18 November 2026", "2026-10-01", false},
		{"Fecha de Cobertura de Seguro de Salud: 11 octubre 2026", "2026-10-01", false},
		{"Fecha de Cobertura de Seguro de Salud: 01 octubre 2025", "2026-10-01", false},
		{"cualquier texto", "fecha-invalida", false},
	}
	for _, c := range cases {
		if got := coverageSheetHasDate(c.text, c.date); got != c.want {
			t.Errorf("coverageSheetHasDate(%q, %q) = %v, want %v", c.text, c.date, got, c.want)
		}
	}
}
