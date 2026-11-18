package main

import (
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
)

// La marca PDI_COBERTURA dice si el expediente ya tiene su hoja de cobertura del portal
// MSP, sin importar cómo llegó: descargada desde «Hojas de cobertura», subida en la
// carpeta del equipo, adjuntada a mano o reemplazada por la del portal.

// Las hojas que Folio generaba antes localmente escriben la fecha como "July 15, 2026" y
// el pie "3814400 |". El portal escribe "15 July 2026" o "15 julio 2026".
var (
	coverageLocalDatePattern   = regexp.MustCompile(`(?i)\b(january|february|march|april|may|june|july|august|september|october|november|december) \d{1,2}, \d{4}\b`)
	coverageLocalFooterPattern = regexp.MustCompile(`3814400 ?\|`)
	coveragePortalDatePattern  = regexp.MustCompile(`(?i)\b\d{1,2} (january|february|march|april|may|june|july|august|september|october|november|december|enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|octubre|noviembre|diciembre) \d{4}\b`)
)

// coveragePageTexts devuelve el texto de cada página. extractPDFText solo lee dos, y una
// hoja reunida de un menor con dos referentes ya tiene tres.
func coveragePageTexts(path string) ([]string, error) {
	f, reader, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	pages := make([]string, 0, reader.NumPage())
	for pageNo := 1; pageNo <= reader.NumPage(); pageNo++ {
		text, err := reader.Page(pageNo).GetPlainText(nil)
		if err != nil {
			return nil, err
		}
		pages = append(pages, text)
	}
	return pages, nil
}

func coveragePageIsPortal(text string) bool {
	normalized := strings.Join(strings.Fields(text), " ")
	if coverageLocalDatePattern.MatchString(normalized) || coverageLocalFooterPattern.MatchString(normalized) {
		return false
	}
	return coveragePortalDatePattern.MatchString(normalized)
}

// coveragePortalCedulas lee una hoja y devuelve las cédulas de members que aparecen en
// páginas del portal. Si alguna página no es del portal, la hoja entera no cuenta.
func coveragePortalCedulas(path string, members []coverageMember) map[string]bool {
	pages, err := coveragePageTexts(path)
	if err != nil || len(pages) == 0 {
		return nil
	}
	found := make(map[string]bool)
	for _, text := range pages {
		if !coveragePageIsPortal(text) {
			return nil
		}
		compact := strings.Join(strings.Fields(normalizeOCRText(text)), "")
		for _, member := range members {
			if strings.Contains(compact, member.Cedula) {
				found[member.Cedula] = true
			}
		}
	}
	return found
}

// coverageFolderComplete indica si las hojas del expediente cubren, con páginas del
// portal, al titular y —en menores— a sus referentes.
func (s *server) coverageFolderComplete(job stagedJob, packageRoot string, row coveragePlanilla) bool {
	members := coverageMembers(row)
	if len(members) == 0 {
		return false
	}
	found := make(map[string]bool)
	seen := make(map[string]bool)
	for _, doc := range s.coverageDocumentsForPlanilla(job, row.ID) {
		relative := filepath.ToSlash(doc.RelativePath)
		if seen[relative] {
			continue
		}
		seen[relative] = true
		path, err := safeWorkspacePath(packageRoot, relative)
		if err != nil {
			continue
		}
		for cedula := range coveragePortalCedulas(path, members) {
			found[cedula] = true
		}
	}
	for _, member := range members {
		if !found[member.Cedula] {
			return false
		}
	}
	return true
}

// markCoverageFromFolders pone PDI_COBERTURA = 'S' en las planillas del período cuyo
// expediente ya tiene su hoja del portal. Nunca la baja a 'N' y un fallo aquí no revierte
// la operación que lo llamó: solo se registra.
func (s *server) markCoverageFromFolders(ctx context.Context, job stagedJob, packageRoot string) {
	if s.serviceDB == nil || job.IsObjections {
		return
	}
	rows, err := s.coverageRows(ctx, job)
	if err != nil {
		log.Printf("coberturas %s: no se pudo revisar la marca de cobertura (%T)", job.ID, err)
		return
	}
	for _, row := range rows {
		if row.CoverageStatus == "S" || !row.HasPDF || !s.coverageFolderComplete(job, packageRoot, row) {
			continue
		}
		if _, err := s.serviceDB.ExecContext(ctx, "UPDATE "+oracleTableName(s.schema)+" SET PDI_COBERTURA = 'S' WHERE PDI_ID = :id AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP'", sql.Named("id", row.ID)); err != nil {
			log.Printf("coberturas %s PDI_ID=%d: no se pudo marcar PDI_COBERTURA (%T)", job.ID, row.ID, err)
			continue
		}
		log.Printf("coberturas %s PDI_ID=%d: PDI_COBERTURA = 'S' por hoja del portal en el expediente", job.ID, row.ID)
	}
}
