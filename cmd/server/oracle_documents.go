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

// syncOracleWorkspace records the active PDFs and version history after the
// workspace and report have been atomically published on disk.
func (s *server) syncOracleWorkspace(ctx context.Context, job *stagedJob, packageRoot string) error {
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
	if err != nil {
		return fmt.Errorf("no se pudo leer el reporte de PDFs: %w", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("reporte de PDFs inválido: %w", err)
	}
	identities, err := s.loadPlanillaIdentities(ctx, *job)
	if err != nil {
		return err
	}
	patientIDs := make(map[string]map[int64]bool)
	for _, identity := range identities {
		folder := normalizePatientFolder(identity.Patient)
		if folder == "" {
			continue
		}
		if patientIDs[folder] == nil {
			patientIDs[folder] = make(map[int64]bool)
		}
		patientIDs[folder][identity.PlanillaID] = true
	}
	planillaByPath := make(map[string]int64)
	originalByPath := make(map[string]string)
	for _, item := range report.Files {
		if item.Output == "" || item.PlanillaID <= 0 {
			continue
		}
		planillaByPath[filepath.ToSlash(item.Output)] = item.PlanillaID
		originalByPath[filepath.ToSlash(item.Output)] = item.Original
	}
	for path, id := range job.DocumentPlanillas {
		if id > 0 {
			planillaByPath[filepath.ToSlash(path)] = id
		}
	}
	for _, doc := range job.ExternalPDFs {
		path := filepath.ToSlash(doc.RelativePath)
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
	documents, _, err := listWorkspacePDFs(packageRoot)
	if err != nil {
		return err
	}
	active := make([]oracleDocument, 0, len(documents))
	for _, doc := range documents {
		if !strings.HasPrefix(doc.Path, "4. EXPEDIENTES/") {
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
		active = append(active, oracleDocument{path: full, name: doc.Name, code: strings.TrimSuffix(doc.Name, ".pdf"), status: "VIGENTE", planillaID: id, original: originalByPath[doc.Path], hash: hash})
	}
	tx, err := s.serviceDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar la transacción Oracle: %w", err)
	}
	defer tx.Rollback()
	planillas := make(map[int64]string)
	for _, identity := range identities {
		patientFolder := normalizePatientFolder(identity.Patient)
		if patientFolder == "" {
			continue
		}
		planillas[identity.PlanillaID] = filepath.Join(packageRoot, "4. EXPEDIENTES", patientFolder)
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
