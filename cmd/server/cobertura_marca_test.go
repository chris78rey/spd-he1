package main

import "testing"

func TestCoveragePageIsPortalDistinguishesPortalFromLocalSheets(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"portal en inglés", "Fecha de Consulta: 15 July 2026\nCédula 1725557977", true},
		{"portal en español", "Fecha de Consulta: 01 octubre 2026", true},
		{"local por fecha", "Fecha de Consulta: July 15, 2026", false},
		{"local por pie", "15 July 2026 Teléfono: 593-2 3814400 | www.salud.gob.ec", false},
		{"sin fecha de consulta", "Documento escaneado sin texto útil", false},
	}
	for _, tc := range cases {
		if got := coveragePageIsPortal(tc.text); got != tc.want {
			t.Errorf("%s: coveragePageIsPortal = %v, se esperaba %v", tc.name, got, tc.want)
		}
	}
}

func TestCoveragePortalCedulasRejectsUnreadableFile(t *testing.T) {
	if got := coveragePortalCedulas("no-existe.pdf", []coverageMember{{Cedula: "1725557977"}}); got != nil {
		t.Fatalf("un PDF ilegible no debe contar como hoja del portal: %v", got)
	}
}
