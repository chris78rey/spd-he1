package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const pdiDocumentTable = "PDI_DOCUMENTO_DIGITAL"
const pdiDocumentSequence = "SEQ_PDI_DOCUMENTO"
const auditorReceiptTable = "SPD_AUDITORIA_RECEPCION"
const auditorReceiptPDFTable = "SPD_AUDITORIA_RECEPCION_PDF"

type oracleDocument struct {
	path       string
	name       string
	code       string
	status     string
	planillaID int64
	original   string
	version    int
	hash       string
}

func oracleQualified(schema, object string) string {
	if schema == "" {
		return `"` + object + `"`
	}
	return `"` + schema + `"."` + object + `"`
}

func fileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func canonicalPath(filePath string) (string, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(abs); resolveErr == nil {
		return resolved, nil
	}
	return abs, nil
}

func syncOracleAuditorReceipts(ctx context.Context, tx *sql.Tx, schema string, job *stagedJob, packageRoot string) error {
	if len(job.AuditorReceipts) == 0 {
		return nil
	}
	receiptTable := oracleQualified(schema, auditorReceiptTable)
	documentTable := oracleQualified(schema, auditorReceiptPDFTable)
	receipts := make(map[string]auditorReceipt, len(job.AuditorReceipts))
	for _, receipt := range job.AuditorReceipts {
		if len(receipt.ID) != 40 || receipt.Patient == "" {
			return errors.New("el registro externo de auditoría tiene una identidad inválida")
		}
		receipts[receipt.ID] = receipt
		merge := "MERGE INTO " + receiptTable + " t USING (SELECT :b_receipt_id AER_ID FROM DUAL) s ON (t.AER_ID = s.AER_ID) " +
			"WHEN MATCHED THEN UPDATE SET t.AER_JOB_ID=:b_job_id, t.AER_CARPETA=:b_folder, t.AER_SERVICIO=:b_service, t.AER_MES=:b_month, t.AER_ANIO=:b_year, t.AER_ESTADO=:b_status, t.AER_FECHA_ACTUALIZACION=SYSDATE " +
			"WHEN NOT MATCHED THEN INSERT (AER_ID,AER_JOB_ID,AER_CARPETA,AER_SERVICIO,AER_MES,AER_ANIO,AER_ESTADO,AER_USUARIO,AER_FECHA_REGISTRO,AER_FECHA_ACTUALIZACION) VALUES (:b_receipt_id,:b_job_id,:b_folder,:b_service,:b_month,:b_year,:b_status,:b_username,SYSDATE,SYSDATE)"
		if _, err := tx.ExecContext(ctx, merge,
			sql.Named("b_receipt_id", receipt.ID), sql.Named("b_job_id", job.ID), sql.Named("b_folder", receipt.Patient),
			sql.Named("b_service", receipt.Service), sql.Named("b_month", receipt.Month), sql.Named("b_year", receipt.Year),
			sql.Named("b_status", receipt.Status), sql.Named("b_username", job.Username)); err != nil {
			return fmt.Errorf("no se pudo guardar el registro Oracle de la carpeta %s: %w", receipt.Patient, err)
		}
	}
	activeByReceipt := make(map[string]map[string]bool, len(receipts))
	latestByPath := make(map[string]workspaceDocument)
	for _, doc := range job.ExternalPDFs {
		if doc.ReceiptID != "" && receiptsHas(receipts, doc.ReceiptID) {
			path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))
			latestByPath[path] = doc
		}
	}
	for _, doc := range latestByPath {
		if doc.ReceiptID == "" || !receiptsHas(receipts, doc.ReceiptID) {
			continue
		}
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(doc.RelativePath)))
		fullPath, err := safeWorkspacePath(packageRoot, path)
		if err != nil {
			return fmt.Errorf("ruta inválida para PDF de auditoría: %s", path)
		}
		if info, statErr := os.Stat(fullPath); statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		hash, err := fileSHA256(fullPath)
		if err != nil {
			return fmt.Errorf("no se pudo calcular la huella del PDF de auditoría %s: %w", doc.OriginalName, err)
		}
		if activeByReceipt[doc.ReceiptID] == nil {
			activeByReceipt[doc.ReceiptID] = make(map[string]bool)
		}
		activeByReceipt[doc.ReceiptID][doc.ID] = true
		var pdi any
		if doc.PlanillaID > 0 && !doc.OraclePending {
			pdi = doc.PlanillaID
		}
		var currentVersion int
		var currentHash, currentPath string
		lookup := "SELECT AED_VERSION, NVL(AED_SHA256,'-'), AED_RUTA FROM " + documentTable + " WHERE AED_UUID=:b_uuid AND AED_ESTADO='VIGENTE'"
		err = tx.QueryRowContext(ctx, lookup, sql.Named("b_uuid", doc.ID)).Scan(&currentVersion, &currentHash, &currentPath)
		if err == nil && strings.EqualFold(currentHash, hash) && currentPath == fullPath {
			update := "UPDATE " + documentTable + " SET AED_PDI_ID=:b_pdi_id, AED_NOMBRE_ARCHIVO=:b_name, AED_NOMBRE_ORIGEN=:b_original, AED_USUARIO=:b_username WHERE AED_UUID=:b_uuid AND AED_VERSION=:b_version"
			if _, err := tx.ExecContext(ctx, update, sql.Named("b_pdi_id", pdi), sql.Named("b_name", filepath.Base(fullPath)), sql.Named("b_original", doc.OriginalName), sql.Named("b_username", job.Username), sql.Named("b_uuid", doc.ID), sql.Named("b_version", currentVersion)); err != nil {
				return fmt.Errorf("no se pudo actualizar el registro del PDF %s: %w", doc.OriginalName, err)
			}
			continue
		}
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("no se pudo consultar el historial del PDF %s: %w", doc.OriginalName, err)
		}
		if err == nil {
			if _, err := tx.ExecContext(ctx, "UPDATE "+documentTable+" SET AED_ESTADO='REEMPLAZADO' WHERE AED_UUID=:b_uuid AND AED_VERSION=:b_version", sql.Named("b_uuid", doc.ID), sql.Named("b_version", currentVersion)); err != nil {
				return fmt.Errorf("no se pudo conservar la versión previa del PDF %s: %w", doc.OriginalName, err)
			}
		}
		var maxVersion int
		if err := tx.QueryRowContext(ctx, "SELECT NVL(MAX(AED_VERSION),0) FROM "+documentTable+" WHERE AED_UUID=:b_uuid", sql.Named("b_uuid", doc.ID)).Scan(&maxVersion); err != nil {
			return fmt.Errorf("no se pudo calcular la versión del PDF %s: %w", doc.OriginalName, err)
		}
		insert := "INSERT INTO " + documentTable + " (AED_UUID,AER_ID,AED_VERSION,AED_PDI_ID,AED_NOMBRE_ARCHIVO,AED_NOMBRE_ORIGEN,AED_RUTA,AED_SHA256,AED_ESTADO,AED_USUARIO,AED_FECHA_REGISTRO) VALUES (:b_uuid,:b_receipt_id,:b_version,:b_pdi_id,:b_name,:b_original,:b_path,:b_sha,'VIGENTE',:b_username,SYSDATE)"
		if _, err := tx.ExecContext(ctx, insert, sql.Named("b_uuid", doc.ID), sql.Named("b_receipt_id", doc.ReceiptID), sql.Named("b_version", maxVersion+1), sql.Named("b_pdi_id", pdi), sql.Named("b_name", filepath.Base(fullPath)), sql.Named("b_original", doc.OriginalName), sql.Named("b_path", fullPath), sql.Named("b_sha", hash), sql.Named("b_username", job.Username)); err != nil {
			return fmt.Errorf("no se pudo registrar en Oracle el PDF %s: %w", doc.OriginalName, err)
		}
	}
	for id, receipt := range receipts {
		rows, err := tx.QueryContext(ctx, "SELECT AED_UUID,AED_VERSION FROM "+documentTable+" WHERE AER_ID=:b_receipt_id AND AED_ESTADO='VIGENTE'", sql.Named("b_receipt_id", id))
		if err != nil {
			return fmt.Errorf("no se pudo revisar el historial Oracle de la carpeta %s: %w", receipt.Patient, err)
		}
		type stale struct {
			uuid    string
			version int
		}
		var staleRows []stale
		for rows.Next() {
			var item stale
			if err := rows.Scan(&item.uuid, &item.version); err != nil {
				rows.Close()
				return err
			}
			if !activeByReceipt[id][item.uuid] {
				staleRows = append(staleRows, item)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, item := range staleRows {
			if _, err := tx.ExecContext(ctx, "UPDATE "+documentTable+" SET AED_ESTADO='ELIMINADO' WHERE AED_UUID=:b_uuid AND AED_VERSION=:b_version", sql.Named("b_uuid", item.uuid), sql.Named("b_version", item.version)); err != nil {
				return fmt.Errorf("no se pudo actualizar el historial de la carpeta %s: %w", receipt.Patient, err)
			}
		}
		var pending int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+documentTable+" WHERE AER_ID=:b_receipt_id AND AED_ESTADO='VIGENTE' AND AED_PDI_ID IS NULL", sql.Named("b_receipt_id", id)).Scan(&pending); err != nil {
			return fmt.Errorf("no se pudo consultar el estado de asociación de la carpeta %s: %w", receipt.Patient, err)
		}
		status := "VINCULADO"
		if pending > 0 {
			status = "PENDIENTE_PLANILLA"
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+receiptTable+" SET AER_ESTADO=:b_status,AER_FECHA_ACTUALIZACION=SYSDATE WHERE AER_ID=:b_receipt_id", sql.Named("b_status", status), sql.Named("b_receipt_id", id)); err != nil {
			return fmt.Errorf("no se pudo actualizar el estado Oracle de la carpeta %s: %w", receipt.Patient, err)
		}
		receipt.Status = status
	}
	return nil
}

func receiptsHas(receipts map[string]auditorReceipt, id string) bool {
	_, ok := receipts[id]
	return ok
}

// syncOracleWorkspace records the active PDFs and version history after the
// workspace and report have been atomically published on disk.
func (s *server) syncOracleWorkspace(ctx context.Context, job *stagedJob, packageRoot string) error {
	// Derived objection workspaces remain isolated from Oracle until the business
	// policy for PDI_OBJETADO and PDI_ESTADO_DIGITALIZACION is confirmed.
	if job.IsObjections {
		return nil
	}
	if s.serviceDB == nil {
		return errors.New("Oracle no está disponible")
	}
	absoluteRoot, err := canonicalPath(packageRoot)
	if err != nil {
		return fmt.Errorf("no se pudo resolver la carpeta del expediente: %w", err)
	}
	packageRoot = absoluteRoot
	var report classificationReport
	reportPath := filepath.Join(s.jobRoot(job.ID), "reportes", "classification_report.json")
	data, err := os.ReadFile(reportPath)
	if errors.Is(err, os.ErrNotExist) && job.TeamFoldersPeriod {
		// Team-created periods have no clinical ZIP to classify. Their Oracle
		// document index is built from ExternalPDFs below.
		err = nil
	} else if err != nil {
		return fmt.Errorf("no se pudo leer el reporte de PDFs: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("reporte de PDFs inválido: %w", err)
		}
	}
	patientIDs := make(map[string]map[int64]bool)
	planillas := make(map[int64]string)
	planillaByPath := make(map[string]int64)
	originalByPath := make(map[string]string)
	pendingOracleByPath := make(map[string]bool)
	for _, item := range report.Files {
		if item.Output == "" || item.PlanillaID <= 0 {
			continue
		}
		path := filepath.ToSlash(item.Output)
		parts := strings.Split(path, "/")
		if len(parts) < 3 || parts[0] != "4. EXPEDIENTES" {
			continue
		}
		folder := parts[1]
		if patientIDs[folder] == nil {
			patientIDs[folder] = make(map[int64]bool)
		}
		patientIDs[folder][item.PlanillaID] = true
		planillas[item.PlanillaID] = filepath.Join(packageRoot, "4. EXPEDIENTES", folder)
		planillaByPath[path] = item.PlanillaID
		originalByPath[path] = item.Original
	}
	for _, item := range job.OraclePlanillas {
		if item.PlanillaID > 0 && item.PatientFolder != "" {
			planillas[item.PlanillaID] = filepath.Join(packageRoot, "4. EXPEDIENTES", item.PatientFolder)
		}
	}
	for path, id := range job.DocumentPlanillas {
		if id > 0 {
			planillaByPath[filepath.ToSlash(path)] = id
		}
	}
	for _, doc := range job.ExternalPDFs {
		path := filepath.ToSlash(doc.RelativePath)
		if doc.OraclePending {
			pendingOracleByPath[path] = true
			originalByPath[path] = doc.OriginalName
			continue
		}
		if doc.PlanillaID > 0 {
			planillaByPath[path] = doc.PlanillaID
			originalByPath[path] = doc.OriginalName
			continue
		}
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			continue
		}
		ids := patientIDs[parts[1]]
		if len(ids) == 1 {
			for id := range ids {
				planillaByPath[path] = id
				originalByPath[path] = doc.OriginalName
			}
		}
	}
	for original, renamed := range job.Renames {
		original = filepath.ToSlash(filepath.Clean(filepath.FromSlash(original)))
		renamed = filepath.ToSlash(filepath.Clean(filepath.FromSlash(renamed)))
		if name := originalByPath[original]; name != "" {
			originalByPath[renamed] = name
		}
		if id := planillaByPath[original]; id > 0 {
			planillaByPath[renamed] = id
		}
	}
	documents, _, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		return err
	}
	active := make([]oracleDocument, 0, len(documents))
	for _, doc := range documents {
		if !strings.HasPrefix(doc.Path, "4. EXPEDIENTES/") {
			continue
		}
		if pendingOracleByPath[doc.Path] {
			continue
		}
		id := planillaByPath[doc.Path]
		if id <= 0 {
			return fmt.Errorf("no se pudo asociar %s a una planilla Oracle; seleccione un expediente con PDI_TRAMITE inequívoco", doc.Path)
		}
		full := filepath.Join(packageRoot, filepath.FromSlash(doc.Path))
		hash, err := fileSHA256(full)
		if err != nil {
			return fmt.Errorf("no se pudo calcular la huella de %s: %w", doc.Path, err)
		}
		code := strings.TrimSuffix(doc.Name, ".pdf")
		if strings.HasPrefix(strings.ToUpper(code), "PENDIENTE_TMP_") {
			code = ""
		}
		active = append(active, oracleDocument{path: full, name: doc.Name, code: code, status: "VIGENTE", planillaID: id, original: originalByPath[doc.Path], hash: hash})
	}
	tx, err := s.serviceDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción Oracle: %w", err)
	}
	defer tx.Rollback()
	if err := syncOracleAuditorReceipts(ctx, tx, s.schema, job, packageRoot); err != nil {
		return err
	}
	touchedPlanillas := make(map[int64]bool)
	for _, id := range planillaByPath {
		touchedPlanillas[id] = true
	}
	pdiTable := oracleTableName(s.schema)
	for id := range touchedPlanillas {
		patientPath := planillas[id]
		if patientPath == "" {
			return fmt.Errorf("no se encontró la carpeta del paciente para PDI_ID %d", id)
		}
		if _, err := tx.ExecContext(ctx, "UPDATE "+pdiTable+" SET PDI_PATH = :ruta WHERE PDI_ID = :id", sql.Named("ruta", patientPath), sql.Named("id", id)); err != nil {
			return fmt.Errorf("no se pudo registrar la carpeta del paciente en Oracle (PDI_ID %d): %w", id, err)
		}
	}
	docTable := oracleQualified(s.schema, pdiDocumentTable)
	for _, doc := range active {
		var existingID int64
		var existingHash string
		lookup := "SELECT PDD_ID, NVL(PDD_SHA256, '-') FROM (SELECT PDD_ID, PDD_SHA256 FROM " + docTable + " WHERE PDI_ID = :pid AND PDD_RUTA = :ruta AND PDD_ESTADO = 'VIGENTE' ORDER BY PDD_VERSION DESC, PDD_ID DESC) WHERE ROWNUM = 1"
		err := tx.QueryRowContext(ctx, lookup, sql.Named("pid", doc.planillaID), sql.Named("ruta", doc.path)).Scan(&existingID, &existingHash)
		if err == nil && strings.EqualFold(existingHash, doc.hash) {
			continue
		}
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("no se pudo consultar el historial Oracle de %s: %w", doc.name, err)
		}
		if err == nil {
			if _, err := tx.ExecContext(ctx, "UPDATE "+docTable+" SET PDD_ESTADO = 'REEMPLAZADO' WHERE PDD_ID = :id", sql.Named("id", existingID)); err != nil {
				return err
			}
		}
		var maxVersion int
		versionQuery := "SELECT NVL(MAX(PDD_VERSION), 0) FROM " + docTable + " WHERE PDI_ID = :pid AND PDD_RUTA = :ruta"
		if err := tx.QueryRowContext(ctx, versionQuery, sql.Named("pid", doc.planillaID), sql.Named("ruta", doc.path)).Scan(&maxVersion); err != nil {
			return fmt.Errorf("no se pudo determinar la versión del PDF %s: %w", doc.name, err)
		}
		version := maxVersion + 1
		original := doc.original
		if original == "" {
			original = doc.name
		}
		insert := "INSERT INTO " + docTable + " (PDD_ID, PDI_ID, PDD_CODIGO_MSP, PDD_NOMBRE_ARCHIVO, PDD_RUTA, PDD_NOMBRE_ORIGEN, PDD_VERSION, PDD_ESTADO, PDD_SHA256, PDD_FECHA_REGISTRO, PDD_USUARIO) VALUES (" + oracleQualified(s.schema, pdiDocumentSequence) + ".NEXTVAL, :pid, :codigo, :nombre, :ruta, :origen, :version, 'VIGENTE', :sha, SYSDATE, :usuario)"
		if _, err := tx.ExecContext(ctx, insert, sql.Named("pid", doc.planillaID), sql.Named("codigo", doc.code), sql.Named("nombre", doc.name), sql.Named("ruta", doc.path), sql.Named("origen", original), sql.Named("version", version), sql.Named("sha", doc.hash), sql.Named("usuario", job.Username)); err != nil {
			return fmt.Errorf("no se pudo guardar en Oracle el PDF %s: %w", doc.name, err)
		}
	}
	// Close any previously active rows no longer present in the workspace. A
	// duplicate consumed by a confirmed merge gets FUSIONADO; removed files get
	// ELIMINADO. Replaced versions are handled above and retain their history.
	current := make(map[string]bool, len(active))
	for _, doc := range active {
		current[fmt.Sprintf("%d|%s", doc.planillaID, doc.path)] = true
	}
	for id := range touchedPlanillas {
		rows, err := tx.QueryContext(ctx, "SELECT PDD_ID, PDD_RUTA FROM "+docTable+" WHERE PDI_ID = :pid AND PDD_ESTADO = 'VIGENTE'", sql.Named("pid", id))
		if err != nil {
			return fmt.Errorf("no se pudo consultar el estado de documentos de PDI_ID %d: %w", id, err)
		}
		type staleRow struct {
			id   int64
			path string
		}
		stale := []staleRow{}
		for rows.Next() {
			var row staleRow
			if err := rows.Scan(&row.id, &row.path); err != nil {
				rows.Close()
				return err
			}
			if !current[fmt.Sprintf("%d|%s", id, row.path)] {
				stale = append(stale, row)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		for _, row := range stale {
			status := "ELIMINADO"
			for _, duplicates := range job.MergedDuplicates {
				for _, duplicate := range duplicates {
					if filepath.Clean(row.path) == filepath.Clean(filepath.Join(packageRoot, filepath.FromSlash(duplicate))) {
						status = "FUSIONADO"
					}
				}
			}
			if _, err := tx.ExecContext(ctx, "UPDATE "+docTable+" SET PDD_ESTADO = :estado WHERE PDD_ID = :id", sql.Named("estado", status), sql.Named("id", row.id)); err != nil {
				return fmt.Errorf("no se pudo actualizar el historial Oracle del PDF %s: %w", row.path, err)
			}
		}
	}
	return tx.Commit()
}

// clearOracleWorkspaceIndex marks the documents in a workspace as removed and
// resets coverage only for planillas still indexed to this workspace.
func (s *server) clearOracleWorkspaceIndex(ctx context.Context, job stagedJob, packageRoot string) error {
	if s.serviceDB == nil {
		return errors.New("Oracle no está disponible")
	}
	root, err := filepath.Abs(packageRoot)
	if err != nil {
		return fmt.Errorf("no se pudo resolver la carpeta del expediente: %w", err)
	}
	expedientesPrefix := filepath.Join(root, "4. EXPEDIENTES") + string(filepath.Separator)
	docTable := oracleQualified(s.schema, pdiDocumentTable)
	pdiTable := oracleTableName(s.schema)
	tx, err := s.serviceDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción Oracle: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT DISTINCT PDI_ID FROM "+docTable+" WHERE INSTR(PDD_RUTA, :prefix) = 1", sql.Named("prefix", expedientesPrefix))
	if err != nil {
		return fmt.Errorf("no se pudo localizar el historial Oracle de este expediente: %w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE "+docTable+" SET PDD_ESTADO = 'ELIMINADO' WHERE INSTR(PDD_RUTA, :prefix) = 1 AND PDD_ESTADO IN ('VIGENTE', 'PENDIENTE')", sql.Named("prefix", expedientesPrefix)); err != nil {
		return fmt.Errorf("no se pudo marcar como eliminados los PDFs de Oracle: %w", err)
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE "+pdiTable+" SET PDI_PATH = NULL, PDI_COBERTURA = 'N' WHERE PDI_ID = :id AND INSTR(PDI_PATH, :prefix) = 1", sql.Named("id", id), sql.Named("prefix", expedientesPrefix)); err != nil {
			return fmt.Errorf("no se pudo limpiar la ruta y restablecer la cobertura para PDI_ID %d: %w", id, err)
		}
	}
	return tx.Commit()
}
