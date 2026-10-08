package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceFusionGroupsIncludeCoverageAndSeparatePlanillas(t *testing.T) {
	root := t.TempDir()
	patient := filepath.Join(root, "4. EXPEDIENTES", "PACIENTE")
	if err := os.MkdirAll(patient, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]int64{
		"C_COBERTURA.pdf":   501,
		"C_COBERTURA_1.pdf": 501,
		"C_COBERTURA_2.pdf": 501,
		"C_COBERTURA_3.pdf": 502,
		"C_COBERTURA_4.pdf": 502,
		"HCU_008.pdf":       501,
		"HCU_008_1.pdf":     501,
	}
	job := stagedJob{DocumentPlanillas: make(map[string]int64)}
	for name, planillaID := range files {
		if err := os.WriteFile(filepath.Join(patient, name), []byte("pdf"), 0600); err != nil {
			t.Fatal(err)
		}
		if isCoveragePDFName(name) {
			job.ExternalPDFs = append(job.ExternalPDFs, workspaceDocument{
				RelativePath: filepath.ToSlash(filepath.Join("4. EXPEDIENTES", "PACIENTE", name)),
				PlanillaID:   planillaID,
			})
		} else {
			job.DocumentPlanillas[filepath.ToSlash(filepath.Join("4. EXPEDIENTES", "PACIENTE", name))] = planillaID
		}
	}

	codes := map[string]struct{}{"C_COBERTURA.pdf": {}, "HCU_008.pdf": {}}
	groups, err := workspaceFusionGroupsWithCodes(job, root, codes)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 3 {
		t.Fatalf("got %d fusion groups, want coverage PDI_ID 501, coverage PDI_ID 502 and HCU_008: %#v", len(groups), groups)
	}
	coverage501, coverage502, clinical := false, false, false
	for _, group := range groups {
		switch {
		case group.Code == "C_COBERTURA.pdf" && group.PlanillaID == 501:
			coverage501 = len(group.Paths) == 3 && group.CanonicalPath == "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf"
		case group.Code == "C_COBERTURA.pdf" && group.PlanillaID == 502:
			coverage502 = len(group.Paths) == 2 && group.CanonicalPath == "4. EXPEDIENTES/PACIENTE/C_COBERTURA_3.pdf"
		case group.Code == "HCU_008.pdf":
			clinical = len(group.Paths) == 2 && group.PlanillaID == 0
		}
	}
	if !coverage501 || !coverage502 || !clinical {
		t.Fatalf("coverage and clinical groups were not separated as expected: %#v", groups)
	}
}

func TestWorkspaceCoverageFusionPreservesDateAndCedulaMetadata(t *testing.T) {
	job := stagedJob{ExternalPDFs: []workspaceDocument{
		{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", PlanillaID: 501, CoverageCedula: "1722250980", CoverageDate: "2026-10-05"},
		{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf", PlanillaID: 501, CoverageCedula: "1712345678", CoverageDate: "2026-10-05"},
		{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA_2.pdf", PlanillaID: 501, CoverageCedula: "1722250980", CoverageDate: "2026-10-06"},
	}}
	items := workspaceCoverageMetadataForPaths(job, []string{
		"4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf",
		"4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf",
		"4. EXPEDIENTES/PACIENTE/C_COBERTURA_2.pdf",
	})
	if len(items) != 3 {
		t.Fatalf("got %d metadata entries, want 3: %#v", len(items), items)
	}
	merged := workspaceDocument{CoverageItems: items}
	if got := coverageDocumentMetadataItems(merged); len(got) != 3 {
		t.Fatalf("merged coverage metadata lost entries: %#v", got)
	}
}
