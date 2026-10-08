package main

import (
	"path/filepath"
	"testing"
)

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
