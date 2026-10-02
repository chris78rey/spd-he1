package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
)

var identifier = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_$#]{0,127}$`)

func main() {
	month := flag.String("mes", "09", "Valor de PDI_MES en Oracle")
	year := flag.String("anio", "2026", "Valor de PDI_ANIO en Oracle")
	service := flag.String("servicio", "EMERGENCIA", "Tipo de servicio que llevará la matriz de demostración")
	output := flag.String("salida", "samples/carga-demo-oracle-09-2026", "Carpeta de salida de los archivos de demostración")
	flag.Parse()
	if _, err := strconv.Atoi(*month); err != nil || len(*month) != 2 {
		log.Fatal("-mes debe tener dos dígitos, por ejemplo 09")
	}
	if _, err := strconv.Atoi(*year); err != nil || len(*year) != 4 {
		log.Fatal("-anio debe tener cuatro dígitos, por ejemplo 2026")
	}
	if !identifier.MatchString(*service) {
		log.Fatal("-servicio solo puede contener letras, números y guion bajo")
	}
	if err := generate(*month, *year, strings.ToUpper(*service), *output); err != nil {
		log.Fatal(err)
	}
}

func generate(month, year, service, output string) error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("no se pudo cargar .env: %w", err)
	}
	user, password := os.Getenv("ORACLE_USER"), os.Getenv("ORACLE_PASSWORD")
	host, port, serviceName := os.Getenv("ORACLE_HOST"), os.Getenv("ORACLE_PORT"), os.Getenv("ORACLE_SERVICE")
	schema := strings.ToUpper(strings.TrimSpace(os.Getenv("ORACLE_SCHEMA")))
	if schema == "" {
		schema = "DIGITALIZACION"
	}
	if user == "" || password == "" || host == "" || port == "" || serviceName == "" {
		return errors.New("ORACLE_USER, ORACLE_PASSWORD, ORACLE_HOST, ORACLE_PORT y ORACLE_SERVICE deben estar definidos en .env")
	}
	if !identifier.MatchString(schema) {
		return errors.New("ORACLE_SCHEMA no es un identificador Oracle válido")
	}
	dsn := (&url.URL{
		Scheme: "oracle",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + serviceName,
	}).String()
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return fmt.Errorf("no se pudo abrir Oracle: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("no se pudo conectar con Oracle: %w", err)
	}
	query := fmt.Sprintf(`SELECT DISTINCT PDI_TRAMITE
		FROM "%s"."PLANILLA_DIGITAL"
		WHERE PDI_MES = :mes AND PDI_ANIO = :anio
		  AND PDI_PLANILLADO = 'S' AND PDI_TRAMITE IS NOT NULL
		ORDER BY PDI_TRAMITE`, schema)
	rows, err := db.QueryContext(ctx, query, sql.Named("mes", month), sql.Named("anio", year))
	if err != nil {
		return fmt.Errorf("no se pudieron consultar los trámites del período: %w", err)
	}
	var tramites []int64
	for rows.Next() {
		var tramite int64
		if err := rows.Scan(&tramite); err != nil {
			_ = rows.Close()
			return fmt.Errorf("PDI_TRAMITE debe contener números enteros: %w", err)
		}
		if tramite > 0 {
			tramites = append(tramites, tramite)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("falló la lectura de PDI_TRAMITE: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("falló el cierre de la consulta Oracle: %w", err)
	}
	if len(tramites) == 0 {
		return fmt.Errorf("Oracle no devolvió planillas planilladas para PDI_MES=%q y PDI_ANIO=%q", month, year)
	}

	parent := filepath.Dir(output)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("no se pudo crear la carpeta de salida: %w", err)
	}
	if _, err := os.Stat(output); err == nil {
		return fmt.Errorf("ya existe %s; especifica otra ruta con -salida para no sobrescribir datos", output)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no se pudo revisar la carpeta de salida: %w", err)
	}
	tempDir, err := os.MkdirTemp(parent, ".demo-lote-")
	if err != nil {
		return fmt.Errorf("no se pudo preparar la carpeta temporal: %w", err)
	}
	defer os.RemoveAll(tempDir)
	if err := os.Chmod(tempDir, 0700); err != nil {
		return fmt.Errorf("no se pudo restringir la carpeta temporal: %w", err)
	}
	if err := writeDemoPackage(tempDir, month, year, service, tramites); err != nil {
		return err
	}
	if err := os.Rename(tempDir, output); err != nil {
		return fmt.Errorf("no se pudo finalizar la carpeta de demostración: %w", err)
	}
	fmt.Printf("Generados %d trámites para PDI_MES=%s, PDI_ANIO=%s y PDI_PLANILLADO='S' en %s\n", len(tramites), month, year, output)
	fmt.Println("Solo se leyó PDI_TRAMITE. Los PDFs y documentos de cabecera son ficticios; no se modificó Oracle.")
	return nil
}

func writeDemoPackage(output, month, year, service string, tramites []int64) error {
	zipPath := filepath.Join(output, "lote_pacientes_oracle_demo.zip")
	archiveFile, err := os.OpenFile(zipPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("no se pudo crear el ZIP: %w", err)
	}
	archive := zip.NewWriter(archiveFile)
	ocrScan, err := rasterOCRDemoPDF([]string{"FORMULARIO 008", "EMERGENCIA", "MOTIVO DE CONSULTA", "DOCUMENTO FICTICIO PARA OCR"})
	if err != nil {
		_ = archive.Close()
		_ = archiveFile.Close()
		return fmt.Errorf("no se pudo generar el PDF escaneado de prueba (se requiere ffmpeg): %w", err)
	}
	for _, tramite := range tramites {
		folder := strconv.FormatInt(tramite, 10)
		documents := []struct {
			name  string
			lines []string
		}{
			{"tmp_a1.pdf", []string{"FORMULARIO 008", "EMERGENCIA", "MOTIVO DE CONSULTA"}},
			{"doc_99.pdf", []string{"FORMULARIO 053", "REFERENCIA", "DERIVACION", "CONTRARREFERENCIA", "ESTABLECIMIENTO QUE DERIVA"}},
			{"epicrisis_02.pdf", []string{"FORMULARIO 006", "EPICRISIS", "RESUMEN DE ALTA", "CUADRO CLINICO DE EGRESO"}},
			{"protocolo_03.pdf", []string{"FORMULARIO 017", "PROTOCOLO QUIRURGICO", "CIRUGIA"}},
			{"anestesia_04.pdf", []string{"FORMULARIO 018A", "TRANSANESTESICO", "ANESTESIA"}},
			{"coverage_05.pdf", []string{"COBERTURA DE SALUD", "CERTIFICADO DE AFILIACION", "COMPROBANTE DE DERECHO"}},
			{"export_01.pdf", []string{"PLANILLA INDIVIDUAL", "VALOR FACTURADO", "LIQUIDACION INDIVIDUAL", "CASO FICTICIO " + folder}},
			{"acta_06.pdf", []string{"ACTA DE ENTREGA", "RECEPCION DEL SERVICIO"}},
			{"ambiguous_07.pdf", []string{"FORMULARIO 008", "EMERGENCIA", "MOTIVO DE CONSULTA", "FORMULARIO 053", "REFERENCIA"}},
			{"ambiguous_tie_08.pdf", []string{"FORMULARIO 008", "EMERGENCIA", "FORMULARIO 053", "REFERENCIA"}},
			{"unrecognized_09.pdf", []string{"DOCUMENTO ADMINISTRATIVO SINTETICO SIN FORMULARIO CLINICO"}},
			{"scan_ilegible.pdf", nil},
		}
		for _, document := range documents {
			writer, err := archive.Create(folder + "/" + document.name)
			if err != nil {
				_ = archive.Close()
				_ = archiveFile.Close()
				return fmt.Errorf("no se pudo agregar un PDF al ZIP: %w", err)
			}
			content := demoPDF(document.lines, month, year)
			if _, err := writer.Write(content); err != nil {
				_ = archive.Close()
				_ = archiveFile.Close()
				return fmt.Errorf("no se pudo escribir un PDF de demostración: %w", err)
			}
		}
		if tramite == tramites[0] {
			writer, err := archive.Create(folder + "/image_ocr_008.pdf")
			if err != nil {
				_ = archive.Close()
				_ = archiveFile.Close()
				return fmt.Errorf("no se pudo agregar el PDF escaneado: %w", err)
			}
			if _, err := writer.Write(ocrScan); err != nil {
				_ = archive.Close()
				_ = archiveFile.Close()
				return fmt.Errorf("no se pudo guardar el PDF escaneado: %w", err)
			}
		}
	}
	archiveErr := archive.Close()
	fileErr := archiveFile.Close()
	if archiveErr != nil {
		return fmt.Errorf("no se pudo cerrar el ZIP: %w", archiveErr)
	}
	if fileErr != nil {
		return fmt.Errorf("no se pudo cerrar el archivo ZIP: %w", fileErr)
	}

	for _, document := range []struct {
		name  string
		lines []string
	}{
		{"1. OFICIO DE PAGO.pdf", []string{"OFICIO DE PAGO - DEMOSTRACION", "PERIODO " + month + "/" + year, "FIRMAS FICTICIAS - SIN VALIDEZ OFICIAL"}},
		{"2. PLANILLA CONSOLIDADA.pdf", []string{"PLANILLA CONSOLIDADA - DEMOSTRACION", "SERVICIO " + service, "PERIODO " + month + "/" + year, "FIRMAS FICTICIAS - SIN VALIDEZ OFICIAL"}},
	} {
		if err := os.WriteFile(filepath.Join(output, document.name), demoPDF(document.lines, month, year), 0600); err != nil {
			return fmt.Errorf("no se pudo crear %s: %w", document.name, err)
		}
	}
	if err := writeMacroEnabledWorkbook(filepath.Join(output, "3. MATRIZ_"+service+"_"+month+"_"+year+".xlsm"), tramites); err != nil {
		return err
	}
	readme := fmt.Sprintf(`ARCHIVOS DE DEMOSTRACION PARA FOLIO

Periodo Oracle: PDI_MES=%s, PDI_ANIO=%s
Filtro: PDI_PLANILLADO='S'
Tramites consultados: %d (solo se leyó PDI_TRAMITE; no se leyeron nombres ni cédulas)
Servicio de los documentos de ejemplo: %s

Seleccione en Carga dual:
- lote_pacientes_oracle_demo.zip como lote
- 3. MATRIZ_%s_%s_%s.xlsm como matriz
- 2. PLANILLA CONSOLIDADA.pdf como planilla consolidada
- 1. OFICIO DE PAGO.pdf como oficio

Los PDFs son ficticios y están marcados como demostración. Cada carpeta numérica
incluye fixtures vectoriales para HCU 008, HCU 053, HCU 006, HCU 017, HCU 018A,
C_COBERTURA, P_INDIVIDUAL y A_ENTREGA; ambiguous_07.pdf activa HCU 008 y HCU 053
con más frases para HCU 008; ambiguous_tie_08.pdf empata, y unrecognized_09.pdf
no coincide. scan_ilegible.pdf es una página vacía; image_ocr_008.pdf es una imagen
escaneada sin capa de texto, generada a ~200 DPI, dentro de la primera planilla.
Los nombres aleatorios de entrada no deben usarse para clasificar. No son expedientes
clínicos reales. La API aún no conecta OCR ni reglas YAML: el procesamiento actual
solo valida PDFs y organiza carpetas.
`, month, year, len(tramites), service, service, month, year)
	if err := os.WriteFile(filepath.Join(output, "LEEME.txt"), []byte(readme), 0600); err != nil {
		return fmt.Errorf("no se pudo crear LEEME.txt: %w", err)
	}
	manifest, err := os.Create(filepath.Join(output, "OCR_EXPECTED.csv"))
	if err != nil {
		return fmt.Errorf("no se pudo crear el manifiesto OCR: %w", err)
	}
	csvWriter := csv.NewWriter(manifest)
	for _, row := range [][]string{
		{"archivo", "resultado_esperado", "escenario"},
		{"tmp_a1.pdf", "HCU_008.pdf", "Coincidencia vectorial directa"},
		{"doc_99.pdf", "HCU_053.pdf", "Referencia y derivacion"},
		{"epicrisis_02.pdf", "HCU_006.pdf", "Epicrisis diferenciada de referencia"},
		{"protocolo_03.pdf", "HCU_017.pdf", "Protocolo quirurgico"},
		{"anestesia_04.pdf", "HCU_018A.pdf", "Protocolo transanestesico"},
		{"coverage_05.pdf", "C_COBERTURA.pdf", "Certificado de cobertura"},
		{"export_01.pdf", "P_INDIVIDUAL.pdf", "Planilla individual"},
		{"acta_06.pdf", "A_ENTREGA.pdf", "Acta de entrega"},
		{"ambiguous_07.pdf", "HCU_008.pdf", "Dos reglas; HCU 008 tiene mas frases coincidentes"},
		{"ambiguous_tie_08.pdf", "PENDIENTE_REVISION", "Empate entre reglas"},
		{"unrecognized_09.pdf", "PENDIENTE_REVISION", "Sin coincidencia"},
		{"scan_ilegible.pdf", "PENDIENTE_REVISION", "PDF en blanco sin texto"},
		{"image_ocr_008.pdf", "HCU_008.pdf", "Escaneo en escala de grises a aproximadamente 200 DPI sin capa de texto"},
	} {
		if err := csvWriter.Write(row); err != nil {
			_ = manifest.Close()
			return fmt.Errorf("no se pudo escribir el manifiesto OCR: %w", err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		_ = manifest.Close()
		return fmt.Errorf("no se pudo finalizar el manifiesto OCR: %w", err)
	}
	if err := manifest.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el manifiesto OCR: %w", err)
	}
	return nil
}

func demoPDF(lines []string, month, year string) []byte {
	var content bytes.Buffer
	if lines == nil {
		content.WriteString("q Q\n")
	} else {
		content.WriteString("BT\n/F1 15 Tf\n54 740 Td\n(DOCUMENTO DE PRUEBA FOLIO) Tj\n/F1 11 Tf\n")
		for _, line := range append(lines, "PERIODO "+month+"/"+year, "NO CONTIENE DATOS CLINICOS REALES") {
			content.WriteString("0 -28 Td\n(")
			content.WriteString(escapePDF(line))
			content.WriteString(") Tj\n")
		}
		content.WriteString("ET\n")
	}
	return buildPDF(content.Bytes())
}

func escapePDF(value string) string {
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(value)
}

func buildPDF(stream []byte) []byte {
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>"),
		[]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"),
		append(append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(stream))), stream...), []byte("endstream")...),
	}
	var result bytes.Buffer
	result.Write([]byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n"))
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = result.Len()
		fmt.Fprintf(&result, "%d 0 obj\n", index+1)
		result.Write(object)
		result.WriteString("\nendobj\n")
	}
	xref := result.Len()
	fmt.Fprintf(&result, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&result, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&result, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return result.Bytes()
}

func rasterOCRDemoPDF(lines []string) ([]byte, error) {
	temp, err := os.MkdirTemp("", "folio-ocr-scan-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)
	textFile := filepath.Join(temp, "ocr.txt")
	if err := os.WriteFile(textFile, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		return nil, err
	}
	jpegFile := filepath.Join(temp, "scan.jpg")
	filter := "format=gray,drawtext=fontfile=/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf:textfile=" + textFile + ":fontcolor=black:fontsize=78:line_spacing=48:x=140:y=380"
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=white:s=1654x2338", "-vf", filter, "-frames:v", "1", "-pix_fmt", "gray", "-q:v", "2", "-y", jpegFile)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	jpeg, err := os.ReadFile(jpegFile)
	if err != nil {
		return nil, err
	}
	return buildJPEGImagePDF(jpeg), nil
}

func buildJPEGImagePDF(jpeg []byte) []byte {
	content := []byte("q 595 0 0 842 0 0 cm /Im0 Do Q\n")
	var imageObject bytes.Buffer
	fmt.Fprintf(&imageObject, "<< /Type /XObject /Subtype /Image /Width 1654 /Height 2338 /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n", len(jpeg))
	imageObject.Write(jpeg)
	imageObject.WriteString("\nendstream")
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /XObject << /Im0 5 0 R >> >> /Contents 4 0 R >>"),
		append(append([]byte(fmt.Sprintf("<< /Length %d >>\nstream\n", len(content))), content...), []byte("endstream")...),
		imageObject.Bytes(),
	}
	var result bytes.Buffer
	result.Write([]byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n"))
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = result.Len()
		fmt.Fprintf(&result, "%d 0 obj\n", index+1)
		result.Write(object)
		result.WriteString("\nendobj\n")
	}
	xref := result.Len()
	fmt.Fprintf(&result, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&result, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&result, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return result.Bytes()
}

func writeMacroEnabledWorkbook(filename string, tramites []int64) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("no se pudo crear la matriz XLSM: %w", err)
	}
	book := zip.NewWriter(file)
	files := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.ms-excel.sheet.macroEnabled.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Planillas DEMO" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	for name, body := range files {
		writer, err := book.Create(name)
		if err != nil {
			_ = book.Close()
			_ = file.Close()
			return fmt.Errorf("no se pudo construir la matriz XLSM: %w", err)
		}
		if _, err := writer.Write([]byte(body)); err != nil {
			_ = book.Close()
			_ = file.Close()
			return fmt.Errorf("no se pudo escribir la matriz XLSM: %w", err)
		}
	}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="inlineStr"><is><t>PDI_TRAMITE</t></is></c></row>`)
	for index, tramite := range tramites {
		row := index + 2
		fmt.Fprintf(&sheet, `<row r="%d"><c r="A%d"><v>%d</v></c></row>`, row, row, tramite)
	}
	sheet.WriteString("</sheetData></worksheet>")
	writer, err := book.Create("xl/worksheets/sheet1.xml")
	if err == nil {
		_, err = writer.Write([]byte(sheet.String()))
	}
	bookErr := book.Close()
	fileErr := file.Close()
	if err != nil {
		return fmt.Errorf("no se pudo escribir las planillas en la matriz: %w", err)
	}
	if bookErr != nil {
		return fmt.Errorf("no se pudo cerrar la matriz XLSM: %w", bookErr)
	}
	if fileErr != nil {
		return fmt.Errorf("no se pudo cerrar el archivo de la matriz: %w", fileErr)
	}
	return nil
}
