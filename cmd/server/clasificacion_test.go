package main

import (
	"path/filepath"
	"testing"
)

func TestMatchFilenameClassificationUsesConfirmedNumericBases(t *testing.T) {
	rules, err := loadClassificationRules(filepath.Join("..", "..", "reglas_clasificacion.yaml"))
	if err != nil {
		t.Fatalf("loadClassificationRules() error = %v", err)
	}

	tests := []struct {
		name string
		want string
	}{
		{name: "08.pdf", want: "HCU_008.pdf"},
		{name: "008.pdf", want: "HCU_008.pdf"},
		{name: "PENDIENTE_tmp_08.pdf", want: "HCU_008.pdf"},
		{name: "08_1.pdf", want: "HCU_008.pdf"},
		{name: "PENDIENTE_tmp_08_1.pdf", want: "HCU_008.pdf"},
		{name: "PENDIENTE_tmp_008.pdf", want: "HCU_008.pdf"},
		{name: "007.pdf", want: "HCU_007.pdf"},
		{name: "PENDIENTE_tmp_007.pdf", want: "HCU_007.pdf"},
		{name: "007_2.pdf", want: "HCU_007.pdf"},
		{name: "PENDIENTE_tmp_007_2.pdf", want: "HCU_007.pdf"},
		{name: "documento_08.pdf", want: ""},
		{name: "documento_007_2.pdf", want: ""},
		{name: "007_final.pdf", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchFilenameClassification(test.name, rules); got != test.want {
				t.Errorf("matchFilenameClassification(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

func TestMatchClassificationRecognizesHCUForm008Keyword(t *testing.T) {
	rules, err := loadClassificationRules(filepath.Join("..", "..", "reglas_clasificacion.yaml"))
	if err != nil {
		t.Fatalf("loadClassificationRules() error = %v", err)
	}

	code, matches, tied := matchClassification("HCU FORM.008", rules)
	if code != "HCU_008.pdf" || matches == 0 || tied {
		t.Fatalf("matchClassification() = (%q, %d, %t), want HCU_008.pdf and a unique match", code, matches, tied)
	}
}

func TestMatchClassificationPrioritizesHCU007InterconsultaPhrase(t *testing.T) {
	rules, err := loadClassificationRules(filepath.Join("..", "..", "reglas_clasificacion.yaml"))
	if err != nil {
		t.Fatalf("loadClassificationRules() error = %v", err)
	}

	text := "FORMULARIO 7 – INTERCONSULTA\nFORMULARIO 017\nPROTOCOLO QUIRURGICO\nCIRUGIA"
	code, matches, tied := matchClassification(text, rules)
	if code != "HCU_007.pdf" || matches != 1 || tied {
		t.Fatalf("matchClassification() = (%q, %d, %t), want prioritized HCU_007.pdf", code, matches, tied)
	}
}

func TestResolvePDFClassificationPrefersConfirmedFilenameOverGenericContent(t *testing.T) {
	rules, err := loadClassificationRules(filepath.Join("..", "..", "reglas_clasificacion.yaml"))
	if err != nil {
		t.Fatalf("loadClassificationRules() error = %v", err)
	}

	filenameCode := matchFilenameClassification("007_2.pdf", rules)
	code, matches, reason := resolvePDFClassification("FORMULARIO 017 PROTOCOLO QUIRURGICO CIRUGIA", filenameCode, rules)
	if code != "HCU_007.pdf" || matches != 0 || reason != "CODIGO_RECONOCIDO_POR_NOMBRE" {
		t.Fatalf("resolvePDFClassification() = (%q, %d, %q), want filename HCU_007.pdf", code, matches, reason)
	}
}
