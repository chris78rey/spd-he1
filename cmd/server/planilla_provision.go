package main

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// pdiProvisioningReport is stored with the staged job so the operator can see
// whether missing PDI rows were created when the ZIP was received.
type pdiProvisioningReport struct {
	Created    []string `json:"creados,omitempty"`
	Unresolved []string `json:"pendientes,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func oracleServiceCodesForIngest(service string) []string {
	switch strings.ToUpper(strings.TrimSpace(service)) {
	case "AMBULATORIO":
		return []string{"CEX"}
	case "EMERGENCIA":
		return []string{"EMR", "OBE", "URG"}
	case "HOSPITALIZACION":
		return []string{"HSP"}
	default:
		return nil
	}
}

// collectZIPPlanillaNumbers validates the ZIP structure before it can cause
// Oracle writes. Only numeric trámite folders containing valid PDFs are used.
func collectZIPPlanillaNumbers(zipPath string) ([]string, error) {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, errors.New("No se pudo abrir el ZIP para identificar sus trámites.")
	}
	defer archive.Close()

	tramites := make(map[string]bool)
	validPDFs := 0
	for _, entry := range archive.File {
		clean := strings.Trim(strings.ReplaceAll(entry.Name, `\`, "/"), "/")
		if clean == "" {
			continue
		}
		parts := strings.Split(clean, "/")
		if entry.FileInfo().IsDir() {
			if len(parts) != 1 || !tramitePattern.MatchString(parts[0]) {
				return nil, fmt.Errorf("La carpeta %q del ZIP no corresponde a un número de planilla.", entry.Name)
			}
			continue
		}
		if len(parts) != 2 || !tramitePattern.MatchString(parts[0]) || !strings.EqualFold(filepath.Ext(parts[1]), ".pdf") || filepath.Base(parts[1]) != parts[1] || entry.UncompressedSize64 > 256<<20 || entry.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("La ruta %q del ZIP no es un PDF válido dentro de una carpeta numérica.", entry.Name)
		}
		file, openErr := entry.Open()
		if openErr != nil {
			return nil, errors.New("No se pudo revisar un PDF del ZIP antes de completar Oracle.")
		}
		var signature [5]byte
		_, readErr := io.ReadFull(file, signature[:])
		closeErr := file.Close()
		if readErr != nil || string(signature[:]) != "%PDF-" || closeErr != nil {
			return nil, fmt.Errorf("El archivo %q no tiene una firma PDF válida.", parts[1])
		}
		tramites[parts[0]] = true
		validPDFs++
	}
	if validPDFs == 0 || len(tramites) == 0 {
		return nil, errors.New("El ZIP no contiene PDFs válidos dentro de carpetas de planilla.")
	}
	result := make([]string, 0, len(tramites))
	for tramite := range tramites {
		result = append(result, tramite)
	}
	sort.Strings(result)
	return result, nil
}

// loadPlanillaIdentitiesByTramites looks up records using PDI_TRAMITE alone.
// It deliberately does not constrain insurer, period, service, or status.
func (s *server) loadPlanillaIdentitiesByTramites(ctx context.Context, tramites []string) (map[string]planillaIdentity, error) {
	identities := make(map[string]planillaIdentity, len(tramites))
	unique := make([]string, 0, len(tramites))
	seen := make(map[string]bool, len(tramites))
	for _, tramite := range tramites {
		tramite = strings.TrimSpace(tramite)
		if !tramitePattern.MatchString(tramite) || seen[tramite] {
			continue
		}
		seen[tramite] = true
		unique = append(unique, tramite)
	}
	for offset := 0; offset < len(unique); offset += 500 {
		end := offset + 500
		if end > len(unique) {
			end = len(unique)
		}
		chunk := unique[offset:end]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for index, tramite := range chunk {
			name := fmt.Sprintf("tramite_%d", index)
			placeholders[index] = ":" + name
			args[index] = sql.Named(name, tramite)
		}
		query := "SELECT TO_CHAR(PDI_ID), TO_CHAR(PDI_TRAMITE), PDI_PACIENTE, TO_CHAR(PDI_FECHA_DESDE, 'YYYY-MM-DD'), TO_CHAR(PDI_FECHA_HASTA, 'YYYY-MM-DD'), PDI_SERVICIO, PDI_CEDULA FROM " + oracleTableName(s.schema) + " WHERE PDI_TRAMITE IN (" + strings.Join(placeholders, ",") + ")"
		rows, err := s.serviceDB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, errors.New("No se pudieron consultar en Oracle los números de planilla incluidos en el ZIP.")
		}
		for rows.Next() {
			var rawPlanillaID, rawTramite, patient, rawCareFrom, rawCareUntil, service, cedula sql.NullString
			if err := rows.Scan(&rawPlanillaID, &rawTramite, &patient, &rawCareFrom, &rawCareUntil, &service, &cedula); err != nil {
				rows.Close()
				return nil, errors.New("No se pudo leer una planilla encontrada en Oracle.")
			}
			tramite := strings.TrimSpace(rawTramite.String)
			planillaID, parseErr := parseOraclePlanillaID(rawPlanillaID)
			if !rawTramite.Valid || !tramitePattern.MatchString(tramite) || parseErr != nil || planillaID <= 0 {
				rows.Close()
				return nil, errors.New("Oracle devolvió un identificador de planilla no válido.")
			}
			if _, duplicate := identities[tramite]; duplicate {
				rows.Close()
				return nil, fmt.Errorf("Oracle contiene más de una fila para la planilla %s; se requiere resolver ese duplicado antes de continuar.", tramite)
			}
			identities[tramite] = planillaIdentity{PlanillaID: planillaID, Tramite: tramite, Cedula: strings.TrimSpace(cedula.String), Patient: strings.TrimSpace(patient.String), Service: strings.TrimSpace(service.String), CareFrom: parseOracleDate(rawCareFrom), CareUntil: parseOracleDate(rawCareUntil)}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, errors.New("Falló la lectura de planillas desde Oracle.")
		}
		rows.Close()
	}
	return identities, nil
}

func parseOraclePlanillaID(value sql.NullString) (int64, error) {
	if !value.Valid {
		return 0, errors.New("Oracle devolvió un PDI_ID vacío.")
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value.String), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, errors.New("Oracle devolvió un PDI_ID no válido.")
	}
	return parsed, nil
}

// provisionMissingPlanillas inserts only ZIP trámites that do not already have
// a PDI_TRAMITE row and whose source data identifies the selected MSP period
// and service. The write is scoped to this ZIP and is all-or-nothing on SQL errors.
func (s *server) provisionMissingPlanillas(ctx context.Context, job stagedJob) (pdiProvisioningReport, error) {
	report := pdiProvisioningReport{Created: []string{}, Unresolved: []string{}}
	serviceCodes := oracleServiceCodesForIngest(job.Service)
	if len(serviceCodes) == 0 {
		return report, errors.New("El servicio seleccionado no tiene un mapeo Oracle autorizado.")
	}
	zipPath := filepath.Join(s.jobSourcesDir(job.ID), "lote.zip")
	tramites, err := collectZIPPlanillaNumbers(zipPath)
	if err != nil {
		return report, err
	}
	existing, err := s.loadPlanillaIdentitiesByTramites(ctx, tramites)
	if err != nil {
		return report, err
	}
	missing := make([]string, 0, len(tramites))
	for _, tramite := range tramites {
		if _, found := existing[tramite]; !found {
			missing = append(missing, tramite)
		}
	}
	if len(missing) == 0 {
		return report, nil
	}

	tx, err := s.serviceDB.BeginTx(ctx, nil)
	if err != nil {
		return report, errors.New("No se pudo iniciar la operación Oracle para completar los trámites faltantes.")
	}
	defer tx.Rollback()
	rollback := func(cause error) (pdiProvisioningReport, error) {
		_ = tx.Rollback()
		report.Created = []string{}
		report.Unresolved = missing
		return report, cause
	}
	insertSQL := missingPlanillaInsertSQL(oracleTableName(s.schema), serviceCodes)
	for _, tramite := range missing {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+oracleTableName(s.schema)+" WHERE PDI_TRAMITE = :tramite", sql.Named("tramite", tramite)).Scan(&count); err != nil {
			return rollback(errors.New("No se pudo comprobar si un trámite faltante ya apareció en Oracle."))
		}
		if count > 1 {
			return rollback(fmt.Errorf("Oracle contiene más de una fila para la planilla %s; no se hicieron cambios.", tramite))
		}
		if count == 1 {
			continue
		}
		args := []any{
			sql.Named("period_start", job.Year+"-"+job.Month+"-01"),
			sql.Named("zip_tramite", tramite),
			sql.Named("zip_tramite_guard", tramite),
		}
		for index, code := range serviceCodes {
			args = append(args, sql.Named(fmt.Sprintf("service_%d", index), code))
		}
		result, err := tx.ExecContext(ctx, insertSQL, args...)
		if err != nil {
			return rollback(errors.New("Oracle no pudo crear uno de los registros faltantes; se revirtieron las inserciones de este ZIP."))
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return rollback(errors.New("Oracle no confirmó cuántos registros se crearon; se revirtieron las inserciones de este ZIP."))
		}
		if inserted == 0 {
			report.Unresolved = append(report.Unresolved, tramite)
			continue
		}
		if inserted != 1 {
			return rollback(fmt.Errorf("La fuente Oracle devolvió %d filas para la planilla %s; se revirtieron las inserciones de este ZIP.", inserted, tramite))
		}
		if err := finalizeNewPlanilla(ctx, tx, oracleTableName(s.schema), tramite); err != nil {
			return rollback(errors.New("No se pudo completar la información complementaria de un nuevo registro; se revirtieron las inserciones de este ZIP."))
		}
		report.Created = append(report.Created, tramite)
	}
	if err := tx.Commit(); err != nil {
		report.Created = []string{}
		report.Unresolved = missing
		return report, errors.New("Oracle no confirmó la operación; el ZIP quedó guardado y no se informaron registros como creados.")
	}
	return report, nil
}

func missingPlanillaInsertSQL(table string, serviceCodes []string) string {
	servicePlaceholders := make([]string, len(serviceCodes))
	for index := range serviceCodes {
		servicePlaceholders[index] = fmt.Sprintf(":service_%d", index)
	}
	return `INSERT INTO ` + table + ` (
		PDI_TRAMITE, PDI_COD_ASEGURADORA, PDI_ASEGURADORA, PDI_FECHA_DESDE, PDI_FECHA_HASTA,
		PDI_COD_SERVICIO, PDI_SERVICIO, PDI_MES, PDI_ANIO, PDI_HC, PDI_PACIENTE,
		PDI_CEDULA, PDI_FECHA_NACIMIENTO, PDI_CODIGO_USUARIO, PDI_NUMERO_PERMANENCIA,
		PDI_USUARIO, PDI_FECHA_PROCESO, PDI_FECHA_ALTA, PDI_MENOR_EDAD, PDI_CERRADO,
		PDI_PLANILLADO, PDI_COBERTURA, PDI_PROCESADO
	)
	WITH period AS (
		SELECT TO_DATE(:period_start, 'YYYY-MM-DD') AS period_start FROM DUAL
	), source_groups AS (
		SELECT MIN(cta.ROWID) AS first_rowid, MAX(cta.ROWID) AS last_rowid,
			cta.pla_numero_planilla AS tramite, cta.prm_codigo AS promocion,
			DECODE(cta.prm_codigo, 14, cta.pcn_numero_hc, cta.pcn_numero_hc_migrado) AS hc,
			MIN(cta.numero_permanencia) AS numero_permanencia,
			COUNT(DISTINCT NVL(TO_CHAR(cta.numero_permanencia), '#NULL#')) AS permanence_count,
			COUNT(*) OVER () AS identity_group_count
		FROM SIS.CUENTAS cta CROSS JOIN period
		WHERE cta.pla_numero_planilla = :zip_tramite
			AND cta.prm_codigo = '04'
			AND cta.fecha >= period.period_start
			AND cta.fecha < ADD_MONTHS(period.period_start, 1)
			AND cta.pla_numero_planilla IS NOT NULL
			AND cta.estado NOT LIKE 'AN%'
		GROUP BY cta.pla_numero_planilla, cta.prm_codigo,
			cta.pcn_numero_hc, cta.pcn_numero_hc_migrado
	)
	SELECT source.tramite, source.promocion, promo.descripcion,
		TRUNC(first_row.fecha), TRUNC(last_row.fecha), first_row.servicio, UPPER(codes.rv_meaning),
		TO_CHAR(period.period_start, 'MM'), TO_CHAR(period.period_start, 'YYYY'), source.hc,
		pct.apellido_paterno || ' ' || pct.apellido_materno || ' ' || pct.primer_nombre || ' ' || pct.segundo_nombre,
		pct.cedula, pct.fecha_nacimiento,
		(SELECT pe.codigo FROM SIS.PERSONAL pe JOIN SIS.CUENTAS c ON pe.usuario = c.actualizado_por WHERE c.ROWID = source.last_rowid),
		source.numero_permanencia, last_row.actualizado_por, SYSDATE,
		(SELECT TRUNC(pya.fecha_alta) FROM SIS.PERMANENCIAS_Y_ATENCIONES pya WHERE pya.numero = source.numero_permanencia),
		'N', 'N', 'S', 'N', 'N'
	FROM source_groups source
	JOIN period ON 1 = 1
	JOIN SIS.CUENTAS first_row ON first_row.ROWID = source.first_rowid
	JOIN SIS.CUENTAS last_row ON last_row.ROWID = source.last_rowid
	JOIN SIS.PROMOCIONES promo ON promo.codigo = source.promocion AND UPPER(TRIM(promo.descripcion)) = 'MSP'
	JOIN SIS.CG_REF_CODES codes ON codes.rv_domain = 'SERVICIO PERMANENCIA' AND codes.rv_low_value = first_row.servicio
	JOIN SIS.PACIENTES pct ON pct.numero_hc = source.hc
	WHERE source.tramite = :zip_tramite_guard
		AND source.hc IS NOT NULL
		AND source.identity_group_count = 1
		AND source.permanence_count = 1
		AND first_row.servicio IN (` + strings.Join(servicePlaceholders, ",") + `)
		AND NOT EXISTS (SELECT 1 FROM ` + table + ` existing WHERE existing.PDI_TRAMITE = source.tramite)`
}

func finalizeNewPlanilla(ctx context.Context, tx *sql.Tx, table, tramite string) error {
	minorUpdate := `UPDATE ` + table + ` SET PDI_MENOR_EDAD = 'S'
		WHERE PDI_TRAMITE = :tramite AND PDI_MENOR_EDAD = 'N' AND PDI_FECHA_DESDE IS NOT NULL
		AND NVL(FLOOR(MONTHS_BETWEEN(LAST_DAY(PDI_FECHA_DESDE), LAST_DAY(PDI_FECHA_NACIMIENTO)) / 12), -1) < 18`
	if _, err := tx.ExecContext(ctx, minorUpdate, sql.Named("tramite", tramite)); err != nil {
		return err
	}
	var hc sql.NullInt64
	var minor string
	if err := tx.QueryRowContext(ctx, "SELECT PDI_HC, PDI_MENOR_EDAD FROM "+table+" WHERE PDI_TRAMITE = :tramite", sql.Named("tramite", tramite)).Scan(&hc, &minor); err != nil {
		return err
	}
	if !hc.Valid || minor != "S" {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT CEDULA FROM SIS.REFERENTES WHERE RELACION = 1 AND PCN_NUMERO_HC = :hc AND CEDULA IS NOT NULL ORDER BY CEDULA`, sql.Named("hc", hc.Int64))
	if err != nil {
		return err
	}
	dependents := make([]string, 0, 2)
	for rows.Next() {
		var cedula string
		if err := rows.Scan(&cedula); err != nil {
			rows.Close()
			return err
		}
		dependents = append(dependents, cedula)
		if len(dependents) == 2 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(dependents) == 0 {
		return nil
	}
	var first, second any
	first = dependents[0]
	if len(dependents) > 1 {
		second = dependents[1]
	}
	_, err = tx.ExecContext(ctx, `UPDATE `+table+` SET PDI_DEPENDIENTE_01 = :first, PDI_DEPENDIENTE_02 = :second WHERE PDI_TRAMITE = :tramite AND PDI_MENOR_EDAD = 'S'`, sql.Named("first", first), sql.Named("second", second), sql.Named("tramite", tramite))
	return err
}

func (s *server) applyPlanillaProvisioning(ctx context.Context, job *stagedJob) {
	if job == nil || job.IsObjections || (job.Status != "STAGED" && job.Status != "REQUIERE_REVISION") {
		return
	}
	report, err := s.provisionMissingPlanillas(ctx, *job)
	if err != nil {
		report.Error = err.Error()
		log.Printf("ingesta %s: no se completaron registros faltantes en Oracle (%T)", job.ID, err)
	}
	if job.PlanillaProvisioning != nil {
		created := make(map[string]bool, len(report.Created)+len(job.PlanillaProvisioning.Created))
		for _, tramite := range report.Created {
			created[tramite] = true
		}
		for _, tramite := range job.PlanillaProvisioning.Created {
			created[tramite] = true
		}
		report.Created = report.Created[:0]
		for tramite := range created {
			report.Created = append(report.Created, tramite)
		}
		sort.Strings(report.Created)
	}
	job.PlanillaProvisioning = &report
	if err := s.saveStagedJob(*job); err != nil {
		log.Printf("ingesta %s: no se pudo guardar el resultado del cruce Oracle (%T)", job.ID, err)
	}
}
