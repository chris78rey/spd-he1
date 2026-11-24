package main

import (
	"strings"
	"testing"
)

func folderLinkTestJob() stagedJob {
	return stagedJob{ID: "WORK-AMBULATORIO-202607", Service: "AMBULATORIO", Month: "07", Year: "2026", Status: "INCOMPLETE"}
}

func validFolderLinkPlanilla() folderLinkPlanilla {
	return folderLinkPlanilla{PlanillaID: 101, Tramite: "1000001", Patient: "PEREZ LOPEZ JUAN CARLOS", InsurerCode: "04", Insurer: "MSP", ServiceCode: "CEX", Month: "07", Year: "2026", Planillado: "S"}
}

func TestFolderLinkValidationAcceptsPlanillaOfTheSamePeriodAndService(t *testing.T) {
	if reason := folderLinkValidation(folderLinkTestJob(), validFolderLinkPlanilla()); reason != "" {
		t.Fatalf("expected planilla to be accepted, got %q", reason)
	}
}

func TestFolderLinkValidationRejectsOtherPeriodServiceInsurerOrUnplanillado(t *testing.T) {
	cases := map[string]struct {
		change func(*folderLinkPlanilla)
		want   string
	}{
		"otro mes":         {func(p *folderLinkPlanilla) { p.Month = "08" }, "agosto 2026"},
		"otro servicio":    {func(p *folderLinkPlanilla) { p.ServiceCode = "EMR" }, "Emergencia"},
		"otra aseguradora": {func(p *folderLinkPlanilla) { p.Insurer = "SOAT/SPPAT"; p.InsurerCode = "14" }, "SOAT/SPPAT"},
		"no planillada":    {func(p *folderLinkPlanilla) { p.Planillado = "N" }, "ya no figura como planillada"},
	}
	for name, tc := range cases {
		planilla := validFolderLinkPlanilla()
		tc.change(&planilla)
		reason := folderLinkValidation(folderLinkTestJob(), planilla)
		if reason == "" || !strings.Contains(reason, tc.want) {
			t.Errorf("%s: reason = %q; want it to mention %q", name, reason, tc.want)
		}
	}
}

func TestFolderLinkNameWarningIgnoresAccentsAndCopySuffix(t *testing.T) {
	if warning := folderLinkNameWarning("MUNOZ_PENA_ANA_MARIA", "MUÑOZ PEÑA ANA MARIA"); warning != "" {
		t.Fatalf("unexpected warning for matching name: %q", warning)
	}
	if warning := folderLinkNameWarning("TORRES_VEGA_LUIS_ALBERTO_2", "TORRES VEGA LUIS ALBERTO"); warning != "" {
		t.Fatalf("unexpected warning for copy suffix: %q", warning)
	}
	if warning := folderLinkNameWarning("RAMOS_SILVA_ROSA", "PEREZ LOPEZ JUAN CARLOS"); warning == "" {
		t.Fatal("expected a warning when the folder belongs to another name")
	}
}

func TestApplyPlanillaFolderLinkLinksPendingPDFsOfThatFolderOnly(t *testing.T) {
	job := folderLinkTestJob()
	job.ExternalPDFs = []workspaceDocument{
		{RelativePath: "4. EXPEDIENTES/GOMEZ_RUIZ_PEDRO_PABLO/HCU_008.pdf", OraclePending: true, ReceiptID: "r1"},
		{RelativePath: "4. EXPEDIENTES/GOMEZ_RUIZ_PEDRO_PABLO/P_INDIVIDUAL.pdf", OraclePending: true, ReceiptID: "r1"},
		{RelativePath: "4. EXPEDIENTES/OTRO_PACIENTE/HCU_008.pdf", OraclePending: true, ReceiptID: "r2"},
	}
	linked := applyPlanillaFolderLink(&job, "GOMEZ_RUIZ_PEDRO_PABLO", workspaceOraclePlanilla{PlanillaID: 102, Tramite: "1000002"})
	if linked != 2 {
		t.Fatalf("linked = %d; want 2", linked)
	}
	for _, doc := range job.ExternalPDFs[:2] {
		if doc.OraclePending || doc.PlanillaID != 102 {
			t.Fatalf("pending PDF of the folder was not linked: %+v", doc)
		}
	}
	if !job.ExternalPDFs[2].OraclePending || job.ExternalPDFs[2].PlanillaID != 0 {
		t.Fatalf("PDF of another folder must stay pending: %+v", job.ExternalPDFs[2])
	}
	if len(job.OraclePlanillas) != 1 || job.OraclePlanillas[0].PatientFolder != "GOMEZ_RUIZ_PEDRO_PABLO" || job.OraclePlanillas[0].PlanillaID != 102 {
		t.Fatalf("folder ownership not recorded: %+v", job.OraclePlanillas)
	}
}

func TestApplyPlanillaFolderLinkKeepsPendingPDFsWhenFolderHasAnotherTramite(t *testing.T) {
	job := folderLinkTestJob()
	job.OraclePlanillas = []workspaceOraclePlanilla{{PlanillaID: 100, Tramite: "6000001", PatientFolder: "PACIENTE_UNO"}}
	job.ExternalPDFs = []workspaceDocument{
		{RelativePath: "4. EXPEDIENTES/PACIENTE_UNO/HCU_002.pdf", OraclePending: true, ReceiptID: "r1"},
	}
	linked := applyPlanillaFolderLink(&job, "PACIENTE_UNO", workspaceOraclePlanilla{PlanillaID: 200, Tramite: "6000002"})
	if linked != 0 || !job.ExternalPDFs[0].OraclePending {
		t.Fatalf("a second trámite must not take pending PDFs silently: linked=%d doc=%+v", linked, job.ExternalPDFs[0])
	}
	if len(job.OraclePlanillas) != 2 || job.OraclePlanillas[1].PatientFolder != "PACIENTE_UNO" {
		t.Fatalf("second trámite not added to the same folder: %+v", job.OraclePlanillas)
	}
}

func TestApplyPlanillaFolderLinkUpdatesExistingEntryInsteadOfDuplicating(t *testing.T) {
	job := folderLinkTestJob()
	job.OraclePlanillas = []workspaceOraclePlanilla{{PlanillaID: 300, Tramite: "6000003"}}
	applyPlanillaFolderLink(&job, "PACIENTE_TRES", workspaceOraclePlanilla{PlanillaID: 300, Tramite: "6000003"})
	if len(job.OraclePlanillas) != 1 || job.OraclePlanillas[0].PatientFolder != "PACIENTE_TRES" {
		t.Fatalf("existing entry should be reused: %+v", job.OraclePlanillas)
	}
}

func TestClassifySISPlanillaExplainsWhyItCannotBeCreated(t *testing.T) {
	job := folderLinkTestJob()
	cases := map[string]struct {
		summary sisPlanillaSummary
		want    string
	}{
		"no existe":        {sisPlanillaSummary{}, "no existe"},
		"anulada":          {sisPlanillaSummary{Total: 3, Annulled: 3}, "anuladas"},
		"otra aseguradora": {sisPlanillaSummary{Total: 1, Insurers: []string{"IESS"}}, "IESS"},
		"otro mes":         {sisPlanillaSummary{Total: 2, MSPActive: 2, Months: []string{"2026-08"}}, "agosto 2026"},
		"otro servicio":    {sisPlanillaSummary{Total: 2, MSPActive: 2, PeriodActive: 2, Months: []string{"2026-07"}, Services: []string{"EMR"}}, "Emergencia"},
	}
	for name, tc := range cases {
		reason := classifySISPlanilla(job, "6000009", tc.summary)
		if reason == "" || !strings.Contains(reason, tc.want) {
			t.Errorf("%s: reason = %q; want it to mention %q", name, reason, tc.want)
		}
	}
	ok := sisPlanillaSummary{Total: 2, MSPActive: 2, PeriodActive: 2, Months: []string{"2026-07"}, Services: []string{"CEX"}}
	if reason := classifySISPlanilla(job, "6000009", ok); reason != "" {
		t.Fatalf("valid SIS planilla should go to the source query, got %q", reason)
	}
}
