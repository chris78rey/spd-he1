package main

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOracleServiceCodesForIngest(t *testing.T) {
	tests := []struct {
		service string
		want    []string
	}{
		{service: "AMBULATORIO", want: []string{"CEX"}},
		{service: "EMERGENCIA", want: []string{"EMR", "OBE", "URG"}},
		{service: "HOSPITALIZACION", want: []string{"HSP"}},
		{service: "DESCONOCIDO", want: nil},
	}
	for _, test := range tests {
		if got := oracleServiceCodesForIngest(test.service); !reflect.DeepEqual(got, test.want) {
			t.Errorf("oracleServiceCodesForIngest(%q) = %v; want %v", test.service, got, test.want)
		}
	}
}

func TestCollectZIPPlanillaNumbers(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "lote.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	for _, name := range []string{"6141270/HCU_008.pdf", "6151828/HCU_006.pdf"} {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("%PDF-1.4\ncontenido de prueba")); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := collectZIPPlanillaNumbers(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"6141270", "6151828"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectZIPPlanillaNumbers() = %v; want %v", got, want)
	}
}

func TestCollectZIPPlanillaNumbersRejectsInvalidPDFPaths(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "lote-invalido.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("no-es-planilla/documento.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("%PDF-1.4\ncontenido")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := collectZIPPlanillaNumbers(zipPath); err == nil {
		t.Fatal("expected invalid folder to be rejected")
	}
}

func TestMissingPlanillaInsertSQLIsScopedToSelectedMSPPeriodAndService(t *testing.T) {
	query := missingPlanillaInsertSQL("DIGITALIZACION.PLANILLA_DIGITAL", []string{"EMR", "OBE", "URG"})
	for _, required := range []string{
		"PDI_TRAMITE", "cta.pla_numero_planilla = :zip_tramite", "cta.prm_codigo = '04'",
		"cta.fecha >= period.period_start", "ADD_MONTHS(period.period_start, 1)",
		"promo.descripcion)) = 'MSP'", "first_row.servicio IN (:service_0,:service_1,:service_2)",
		"MIN(cta.ROWID) AS first_rowid", "codes.rv_low_value = first_row.servicio",
		"existing.PDI_TRAMITE = source.tramite", "PDI_PLANILLADO, PDI_COBERTURA, PDI_PROCESADO",
	} {
		if !strings.Contains(query, required) {
			t.Errorf("scoped insert query missing %q", required)
		}
	}
	if strings.Contains(query, "source.service_count = 1") {
		t.Error("insert query must follow the stored procedure's service choice from the first account row")
	}
}

func TestPlanillaForPatientUsesPersistedZIPAssociation(t *testing.T) {
	job := stagedJob{DocumentPlanillas: map[string]int64{
		"4. EXPEDIENTES/ACOSTA BRIONES ANA/008.pdf": 123,
		"4. EXPEDIENTES/OTRO PACIENTE/006.pdf":      456,
	}}
	got, err := (&server{}).planillaForPatient(context.Background(), job, "ACOSTA BRIONES ANA")
	if err != nil {
		t.Fatal(err)
	}
	if got != 123 {
		t.Fatalf("planillaForPatient() = %d; want 123", got)
	}
}
