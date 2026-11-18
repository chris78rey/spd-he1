package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
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

func crossTramiteCoverageFixture(t *testing.T) (stagedJob, string) {
	t.Helper()
	root := t.TempDir()
	patient := filepath.Join(root, "4. EXPEDIENTES", "PACIENTE")
	if err := os.MkdirAll(patient, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"C_COBERTURA.pdf", "C_COBERTURA_1.pdf", "P_INDIVIDUAL.pdf", "P_INDIVIDUAL_1.pdf"} {
		if err := os.WriteFile(filepath.Join(patient, name), []byte("pdf"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// El trámite 502 se atendió antes (5 de septiembre) que el 501 (20 de septiembre).
	job := stagedJob{
		DocumentPlanillas: map[string]int64{
			"4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL.pdf":   501,
			"4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL_1.pdf": 502,
		},
		ExternalPDFs: []workspaceDocument{
			{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", PlanillaID: 501, CoverageCedula: "1712345678", CoverageDate: "2026-09-20"},
			{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf", PlanillaID: 502, CoverageCedula: "1712345678", CoverageDate: "2026-09-05"},
		},
	}
	return job, root
}

func TestCrossTramiteCoverageGroupIsOptionalAndNeverPending(t *testing.T) {
	job, root := crossTramiteCoverageFixture(t)

	pending, err := workspaceFusionGroupsWithCodes(job, root, map[string]struct{}{"C_COBERTURA.pdf": {}, "P_INDIVIDUAL.pdf": {}})
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range pending {
		if group.Code == "C_COBERTURA.pdf" {
			t.Fatalf("una hoja por trámite no debe quedar como fusión pendiente (bloquearía la entrega): %#v", group)
		}
	}
	cross, err := workspaceCrossTramiteCoverageGroups(job, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cross) != 1 || !cross[0].CrossTramite || cross[0].CanonicalPath != "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf" {
		t.Fatalf("cross groups = %#v", cross)
	}
	// Orden por fecha de atención: primero la del 5 de septiembre.
	want := []string{"4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf", "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf"}
	if len(cross[0].Paths) != 2 || cross[0].Paths[0] != want[0] || cross[0].Paths[1] != want[1] {
		t.Fatalf("paths = %v, want %v", cross[0].Paths, want)
	}
}

func TestCrossTramiteCoverageGroupNotOfferedForSingleTramite(t *testing.T) {
	job, root := crossTramiteCoverageFixture(t)
	job.ExternalPDFs[1].PlanillaID = 501
	cross, err := workspaceCrossTramiteCoverageGroups(job, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cross) != 0 {
		t.Fatalf("con un solo trámite la fusión normal ya cubre el caso: %#v", cross)
	}
}

func TestMergedCrossTramiteCoverageBelongsToEarliestAndBacksEveryTramite(t *testing.T) {
	job, _ := crossTramiteCoverageFixture(t)
	items := workspaceCoverageMetadataForPaths(job, []string{"4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", "4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf"})
	if primary := primaryCoveragePlanilla(items); primary != 502 {
		t.Fatalf("primaryCoveragePlanilla() = %d, want 502 (fecha más antigua)", primary)
	}
	if primary := primaryCoveragePlanilla([]coverageDocumentMetadata{{PlanillaID: 9, Date: "2026-09-05"}, {PlanillaID: 4, Date: "2026-09-05"}, {PlanillaID: 2}}); primary != 4 {
		t.Fatalf("a igual fecha gana el menor PDI_ID y las páginas sin fecha van al final; got %d", primary)
	}

	// Estado tras fusionar: una sola hoja asociada al trámite 502 con páginas de ambos.
	merged := stagedJob{
		DocumentPlanillas: job.DocumentPlanillas,
		ExternalPDFs:      []workspaceDocument{{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", PlanillaID: 502, CoverageItems: items}},
	}
	relative := "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf"
	if ids := coveragePlanillaIDs(merged, relative); len(ids) != 2 || ids[0] != 501 || ids[1] != 502 {
		t.Fatalf("coveragePlanillaIDs() = %v", ids)
	}
	if !pdfBelongsToPlanilla(merged, relative, 501) || !pdfBelongsToPlanilla(merged, relative, 502) || pdfBelongsToPlanilla(merged, relative, 503) {
		t.Fatal("la hoja reunida debe respaldar a los trámites 501 y 502, y solo a ellos")
	}
	if pdfBelongsToPlanilla(merged, "4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL.pdf", 502) {
		t.Fatal("un PDF que no es cobertura solo pertenece a su trámite")
	}
	// Ambos trámites siguen viéndose cubiertos, cada uno con su fecha.
	s := &server{workspacesDir: t.TempDir()}
	index := s.indexCoverageDocuments(merged)
	if index[501] == nil || !index[501].Dates["2026-09-20"] || index[502] == nil || !index[502].Dates["2026-09-05"] {
		t.Fatalf("índice de coberturas = %#v", index)
	}

	// Oracle: la hoja se registra también en el trámite que no es el principal.
	planillaByPath := map[string]int64{relative: 502}
	planillas := map[int64]string{501: "/x/PACIENTE", 502: "/x/PACIENTE"}
	extras := coverageExtraPlanillas(&merged, planillaByPath, planillas)
	if len(extras[relative]) != 1 || extras[relative][0] != 501 {
		t.Fatalf("coverageExtraPlanillas() = %v, want 501 como trámite adicional", extras)
	}
	delete(planillas, 501)
	if extras := coverageExtraPlanillas(&merged, planillaByPath, planillas); len(extras) != 0 {
		t.Fatalf("un trámite sin carpeta conocida no se registra: %v", extras)
	}
}

func TestMergeEndpointJoinsCoverageOfTwoTramitesUnderTheEarliest(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil { // el catálogo MSP se lee desde la raíz del proyecto
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	dataDir := t.TempDir()
	s := &server{dataDir: dataDir, workspacesDir: filepath.Join(dataDir, "expedientes"), stagingDir: filepath.Join(dataDir, "staging"),
		sessions: map[string]session{"t": {username: "TEST", expiresAt: time.Now().Add(time.Hour)}}}
	job := stagedJob{ID: "WORK-EMERGENCIA-202609", Service: "EMERGENCIA", Month: "09", Year: "2026", Status: "PROCESSED",
		DocumentPlanillas: map[string]int64{},
		ExternalPDFs: []workspaceDocument{
			{ID: "a", StoredName: "a.pdf", OriginalName: "C_COBERTURA.pdf", RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", PlanillaID: 501, CoverageCedula: "1712345678", CoverageDate: "2026-09-20"},
			{ID: "b", StoredName: "b.pdf", OriginalName: "C_COBERTURA.pdf", RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf", PlanillaID: 502, CoverageCedula: "1712345678", CoverageDate: "2026-09-05"},
		}}
	root := s.jobRoot(job.ID)
	packageRoot := filepath.Join(root, "trabajo", packageFolderName(&job))
	patient := filepath.Join(packageRoot, "4. EXPEDIENTES", "PACIENTE")
	if err := os.MkdirAll(patient, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fuentes"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(patient, "C_COBERTURA.pdf"), coverageReplacementTestPDF(t, "ATENCION 20 SEPTIEMBRE"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(patient, "C_COBERTURA_1.pdf"), coverageReplacementTestPDF(t, "ATENCION 05 SEPTIEMBRE"), 0600); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(job)
	if err := os.WriteFile(filepath.Join(root, "job.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}

	// La pantalla envía las hojas en orden de atención: primero la del 5 de septiembre.
	body := `{"rutas":["4. EXPEDIENTES/PACIENTE/C_COBERTURA_1.pdf","4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expedientes/documentos/fusionar/"+job.ID, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "folio_session", Value: "t"})
	response := httptest.NewRecorder()

	s.mergeWorkspacePDFs(response, req)

	// Sin Oracle en pruebas, la fusión se guarda y la sincronización informa el fallo.
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "Oracle") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	saved, err := s.loadStagedJob(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	var merged workspaceDocument
	for _, document := range saved.ExternalPDFs {
		if document.RelativePath == "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf" {
			merged = document
		}
	}
	if merged.PlanillaID != 502 || len(merged.CoverageItems) != 2 {
		t.Fatalf("hoja reunida = %#v, want asociada al trámite 502 (atención más antigua) con 2 páginas", merged)
	}
	if !pdfBelongsToPlanilla(saved, merged.RelativePath, 501) || !pdfBelongsToPlanilla(saved, merged.RelativePath, 502) {
		t.Fatal("la hoja reunida debe respaldar ambos trámites")
	}
	text, _ := extractPDFText(filepath.Join(patient, "C_COBERTURA.pdf"))
	if first, second := strings.Index(text, "05 SEPTIEMBRE"), strings.Index(text, "20 SEPTIEMBRE"); first < 0 || second < 0 || first > second {
		t.Fatalf("páginas fuera de orden: %q", text)
	}
	if pages, err := api.PageCountFile(filepath.Join(patient, "C_COBERTURA.pdf")); err != nil || pages != 2 {
		t.Fatalf("páginas = %d, %v", pages, err)
	}
	if _, err := os.Stat(filepath.Join(patient, "C_COBERTURA_1.pdf")); !os.IsNotExist(err) {
		t.Fatal("la hoja numerada debía consumirse en la fusión")
	}
}

func TestObjectionCopyIncludesJoinedCoverageForNonPrimaryTramite(t *testing.T) {
	source := filepath.Join(t.TempDir(), "PACIENTE")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"C_COBERTURA.pdf", "P_INDIVIDUAL.pdf", "P_INDIVIDUAL_1.pdf"} {
		if err := os.WriteFile(filepath.Join(source, name), coverageReplacementTestPDF(t, name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Hoja reunida: asociada al trámite 502 (principal), con una página del 501.
	job := stagedJob{
		DocumentPlanillas: map[string]int64{
			"4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL.pdf":   501,
			"4. EXPEDIENTES/PACIENTE/P_INDIVIDUAL_1.pdf": 502,
		},
		ExternalPDFs: []workspaceDocument{{RelativePath: "4. EXPEDIENTES/PACIENTE/C_COBERTURA.pdf", PlanillaID: 502, CoverageItems: []coverageDocumentMetadata{
			{PlanillaID: 502, Cedula: "1712345678", Date: "2026-09-05"},
			{PlanillaID: 501, Cedula: "1712345678", Date: "2026-09-20"},
		}}},
	}
	destination := filepath.Join(t.TempDir(), "PACIENTE_501")

	// Se objeta solo el trámite 501, que no es el principal de la hoja.
	copied, mappings, _, err := copyObjectionPDFTree(source, destination, "PACIENTE", "PACIENTE_501", job, map[int64]bool{501: true})
	if err != nil {
		t.Fatal(err)
	}
	if copied != 2 {
		t.Fatalf("copied = %d, want P_INDIVIDUAL.pdf y la hoja de cobertura reunida", copied)
	}
	if _, err := os.Stat(filepath.Join(destination, "C_COBERTURA.pdf")); err != nil {
		t.Fatalf("la cobertura reunida no se copió para el trámite 501: %v", err)
	}
	if mappings["4. EXPEDIENTES/PACIENTE_501/C_COBERTURA.pdf"] != 501 {
		t.Fatalf("mappings = %v, want la cobertura vinculada al trámite objetado 501", mappings)
	}
	if _, err := os.Stat(filepath.Join(destination, "P_INDIVIDUAL_1.pdf")); !os.IsNotExist(err) {
		t.Fatal("no debe copiarse la planilla del trámite 502, que no está objetado")
	}
}
