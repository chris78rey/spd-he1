package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	pdf "github.com/ledongthuc/pdf"
	"gopkg.in/yaml.v3"
)

type classificationRules struct {
	Rules []struct {
		Code     string   `yaml:"codigo"`
		Keywords []string `yaml:"palabras_clave"`
	} `yaml:"reglas"`
}

type classificationResult struct {
	PlanillaID         int64    `json:"pdi_id,omitempty"`
	Tramite            string   `json:"pdi_tramite,omitempty"`
	ZIPTramite         string   `json:"tramite_carpeta_zip,omitempty"`
	Code               string   `json:"codigo"`
	Method             string   `json:"metodo"`
	Reason             string   `json:"motivo,omitempty"`
	Matches            int      `json:"coincidencias,omitempty"`
	Original           string   `json:"nombre_original"`
	Output             string   `json:"salida,omitempty"`
	DatesOutsidePeriod []string `json:"fechas_fuera_periodo,omitempty"`
	BilledPeriod       string   `json:"periodo_facturado_oracle,omitempty"`
}

type periodDateWarning struct {
	Document string   `json:"documento"`
	Dates    []string `json:"fechas_detectadas"`
	Period   string   `json:"periodo_facturado_oracle,omitempty"`
}

type classificationSummary struct {
	Total            int                 `json:"total"`
	Classified       int                 `json:"clasificados"`
	Pending          int                 `json:"pendientes"`
	FusionPending    int                 `json:"fusiones_pendientes"`
	Vector           int                 `json:"texto_vectorial"`
	OCR              int                 `json:"ocr"`
	Unreadable       int                 `json:"sin_texto_legible"`
	PeriodAlertFiles []periodDateWarning `json:"documentos_fecha_fuera_periodo,omitempty"`
}

func loadClassificationRules(path string) (classificationRules, error) {
	var rules classificationRules
	b, err := os.ReadFile(path)
	if err != nil {
		return rules, fmt.Errorf("no se pudo leer el catálogo OCR: %w", err)
	}
	if err := yaml.Unmarshal(b, &rules); err != nil {
		return rules, fmt.Errorf("el catálogo OCR no es YAML válido: %w", err)
	}
	if len(rules.Rules) == 0 {
		return rules, fmt.Errorf("el catálogo OCR no contiene reglas")
	}
	return rules, nil
}

func classifyPDF(ctx context.Context, filePath, originalName string, rules classificationRules, periodMonth, periodYear string) classificationResult {
	result := classificationResult{Original: originalName, BilledPeriod: formatBilledPeriod(periodMonth, periodYear)}
	text, err := extractPDFText(filePath)
	if err == nil && strings.TrimSpace(text) != "" {
		result.Method = "VECTORIAL"
	} else {
		ocrText, ocrErr := runOCR(ctx, filePath)
		if ocrErr == nil && strings.TrimSpace(ocrText) != "" {
			text = ocrText
			result.Method = "OCR_TESSERACT"
		} else {
			result.Code = pendingPDFName(originalName)
			if ocrErr != nil {
				result.Reason = "OCR_NO_DISPONIBLE_O_FALLIDO"
			} else {
				result.Reason = "SIN_TEXTO_LEGIBLE"
			}
			return result
		}
	}
	result.DatesOutsidePeriod = datesOutsideBilledPeriod(text, periodMonth, periodYear)
	if code, matches, tie := matchClassification(text, rules); code != "" && !tie {
		result.Code, result.Matches = code, matches
		return result
	}
	result.Code = pendingPDFName(originalName)
	result.Reason = "SIN_COINCIDENCIA_O_AMBIGUO"
	return result
}

func formatBilledPeriod(month, year string) string {
	monthNumber, err := strconv.Atoi(strings.TrimSpace(month))
	if err != nil || monthNumber < 1 || monthNumber > 12 || len(strings.TrimSpace(year)) != 4 {
		return ""
	}
	return fmt.Sprintf("%02d/%s", monthNumber, strings.TrimSpace(year))
}

var numericDatePattern = regexp.MustCompile(`\b(\d{1,2})[/-](\d{1,2})[/-](\d{4})\b`)
var isoDatePattern = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
var spanishDatePattern = regexp.MustCompile(`\b(\d{1,2}) (?:DE )?(ENERO|FEBRERO|MARZO|ABRIL|MAYO|JUNIO|JULIO|AGOSTO|SEPTIEMBRE|OCTUBRE|NOVIEMBRE|DICIEMBRE) (?:DE )?(\d{4})\b`)
var labelledDateFieldPattern = regexp.MustCompile(`\bFECHA(?:\s+DE)?\s*[:\-]?\s*(?:\d{1,2}\b|ENERO|FEBRERO|MARZO|ABRIL|MAYO|JUNIO|JULIO|AGOSTO|SEPTIEMBRE|OCTUBRE|NOVIEMBRE|DICIEMBRE)`)

func datesOutsideBilledPeriod(text, periodMonth, periodYear string) []string {
	monthNumber, err := strconv.Atoi(strings.TrimSpace(periodMonth))
	yearNumber, yearErr := strconv.Atoi(strings.TrimSpace(periodYear))
	if err != nil || yearErr != nil || monthNumber < 1 || monthNumber > 12 || yearNumber < 1 {
		return nil
	}
	monthNumbers := map[string]int{"ENERO": 1, "FEBRERO": 2, "MARZO": 3, "ABRIL": 4, "MAYO": 5, "JUNIO": 6, "JULIO": 7, "AGOSTO": 8, "SEPTIEMBRE": 9, "OCTUBRE": 10, "NOVIEMBRE": 11, "DICIEMBRE": 12}
	unique := make(map[string]time.Time)
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		contextText := strings.ToUpper(line)
		if index > 0 {
			contextText = strings.ToUpper(lines[index-1]) + " " + contextText
		}
		contextText = normalizeOCRText(contextText)
		// Solo comparar fechas presentadas como campos del formulario. Omitir historia,
		// límites para realizar exámenes, y fechas de impresión/planillación automática.
		if !isRelevantDocumentDateContext(contextText) {
			continue
		}
		for _, match := range numericDatePattern.FindAllStringSubmatch(line, -1) {
			day, _ := strconv.Atoi(match[1])
			month, _ := strconv.Atoi(match[2])
			year, _ := strconv.Atoi(match[3])
			addDateCandidateForPeriod(unique, year, month, day, monthNumber, yearNumber)
		}
		for _, match := range isoDatePattern.FindAllStringSubmatch(line, -1) {
			year, _ := strconv.Atoi(match[1])
			month, _ := strconv.Atoi(match[2])
			day, _ := strconv.Atoi(match[3])
			addDateCandidateForPeriod(unique, year, month, day, monthNumber, yearNumber)
		}
		normalizedLine := normalizeOCRText(line)
		for _, match := range spanishDatePattern.FindAllStringSubmatch(normalizedLine, -1) {
			day, _ := strconv.Atoi(match[1])
			month := monthNumbers[match[2]]
			year, _ := strconv.Atoi(match[3])
			addDateCandidateForPeriod(unique, year, month, day, monthNumber, yearNumber)
		}
	}
	result := make([]string, 0, len(unique))
	for date := range unique {
		result = append(result, date)
	}
	sort.Strings(result)
	return result
}

func isRelevantDocumentDateContext(context string) bool {
	if strings.Contains(context, "NACIMIENTO") || strings.Contains(context, "FECHA PROCESO") || strings.Contains(context, "FECHA DE PROCESO") || strings.Contains(context, "FECHA PLANILLACION") || strings.Contains(context, "PLANILLAJE AUTOMATICO") || strings.Contains(context, "GENERADO AUTOMATICAMENTE") || strings.Contains(context, "FECHA MAXIMA") || strings.Contains(context, "FECHA LIMITE") || strings.Contains(context, "VENCIMIENTO") {
		return false
	}
	for _, label := range []string{"FECHA DE ATENCION", "FECHA DE CONSULTA", "FECHA DE SOLICITUD", "FECHA DE TOMA", "FECHA DE PROCEDIMIENTO", "FECHA DE INGRESO", "FECHA DE EGRESO", "FECHA DE EVOLUCION", "FECHA DE INTERCONSULTA", "FECHA DE VISITA"} {
		if strings.Contains(context, label) {
			return true
		}
	}
	return labelledDateFieldPattern.MatchString(context)
}

func addDateCandidateForPeriod(unique map[string]time.Time, year, month, day, billedMonth, billedYear int) {
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return
	}
	if month != billedMonth || year != billedYear {
		unique[date.Format("2006-01-02")] = date
	}
}

func extractPDFText(filePath string) (string, error) {
	f, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var text strings.Builder
	pageCount := reader.NumPage()
	if pageCount > 2 {
		pageCount = 2
	}
	for pageNo := 1; pageNo <= pageCount; pageNo++ {
		page := reader.Page(pageNo)
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(pageText)
		text.WriteByte('\n')
	}
	return text.String(), nil
}

func runOCR(ctx context.Context, filePath string) (string, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return "", err
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "folio-ocr-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)
	outputPrefix := filepath.Join(tempDir, "page")
	render := exec.CommandContext(ctx, "pdftoppm", "-f", "1", "-l", "1", "-gray", "-r", "250", "-singlefile", filePath, outputPrefix)
	if output, err := render.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm: %w: %s", err, strings.TrimSpace(string(output)))
	}
	imagePath := outputPrefix + ".pgm"
	if _, err := os.Stat(imagePath); err != nil {
		imagePath = outputPrefix + ".png"
	}
	ocr := exec.CommandContext(ctx, "tesseract", imagePath, "stdout", "-l", "spa")
	output, err := ocr.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func matchClassification(text string, rules classificationRules) (string, int, bool) {
	normalized := normalizeOCRText(text)
	bestCode, bestCount, bestSpecificity := "", 0, 0
	tied := false
	for _, rule := range rules.Rules {
		count, specificity := 0, 0
		seen := make(map[string]struct{}, len(rule.Keywords))
		for _, keyword := range rule.Keywords {
			normalizedKeyword := normalizeOCRText(keyword)
			if normalizedKeyword == "" {
				continue
			}
			if _, duplicate := seen[normalizedKeyword]; duplicate {
				continue
			}
			seen[normalizedKeyword] = struct{}{}
			if strings.Contains(normalized, normalizedKeyword) {
				count++
				specificity += len(normalizedKeyword)
			}
		}
		if count > bestCount {
			bestCode, bestCount, bestSpecificity, tied = rule.Code, count, specificity, false
		} else if count > 0 && count == bestCount {
			if specificity > bestSpecificity {
				bestCode, bestSpecificity, tied = rule.Code, specificity, false
			} else if specificity == bestSpecificity && rule.Code != bestCode {
				tied = true
			}
		}
	}
	return bestCode, bestCount, tied
}

func normalizeOCRText(text string) string {
	var out strings.Builder
	for _, r := range strings.ToUpper(text) {
		switch r {
		case 'Á', 'À', 'Â', 'Ä':
			r = 'A'
		case 'É', 'È', 'Ê', 'Ë':
			r = 'E'
		case 'Í', 'Ì', 'Î', 'Ï':
			r = 'I'
		case 'Ó', 'Ò', 'Ô', 'Ö':
			r = 'O'
		case 'Ú', 'Ù', 'Û', 'Ü':
			r = 'U'
		case 'Ñ':
			r = 'N'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		} else {
			out.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func pendingPDFName(originalName string) string {
	base := strings.TrimSuffix(filepath.Base(originalName), filepath.Ext(originalName))
	base = safeOriginalFilename(base)
	if base == "" {
		base = "documento"
	}
	return "PENDIENTE_tmp_" + base + ".pdf"
}

func copyWithLimit(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return n, err
	}
	if n > limit {
		return n, fmt.Errorf("el documento supera el límite de %d bytes", limit)
	}
	return n, nil
}
