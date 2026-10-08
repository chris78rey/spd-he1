package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
)

const sessionDuration = 8 * time.Hour
const requiredOracleRole = "SPD_EXTERNOS"
const deleteWorkspaceOracleRole = "SPD_BORRA_EXPEDIENTE"
const planillaPageSize = 10

var oracleIdentifierPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_$#]{0,127}$`)

type session struct {
	username            string
	canDeleteWorkspaces bool
	expiresAt           time.Time
}

type server struct {
	host          string
	port          string
	service       string
	schema        string
	serviceDB     *sql.DB
	dataDir       string
	stagingDir    string
	workspacesDir string
	maxIngest     int64
	cookieSecure  bool
	ingestMu      sync.Mutex
	mu            sync.Mutex
	sessions      map[string]session
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	_ = godotenv.Load()

	dataDir := envOr("FOLIO_DATA_DIR", "data")
	s := &server{
		host:          os.Getenv("ORACLE_HOST"),
		port:          os.Getenv("ORACLE_PORT"),
		service:       os.Getenv("ORACLE_SERVICE"),
		schema:        strings.ToUpper(strings.TrimSpace(os.Getenv("ORACLE_SCHEMA"))),
		dataDir:       dataDir,
		stagingDir:    filepath.Join(dataDir, "staging"),
		workspacesDir: filepath.Join(dataDir, "expedientes"),
		maxIngest:     envInt64("MAX_INGEST_UPLOAD_BYTES", 2<<30),
		cookieSecure:  strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
		sessions:      make(map[string]session),
	}
	if s.host == "" || s.port == "" || s.service == "" {
		log.Fatal("ORACLE_HOST, ORACLE_PORT y ORACLE_SERVICE son obligatorios")
	}
	if s.schema != "" && !oracleIdentifierPattern.MatchString(s.schema) {
		log.Fatal("ORACLE_SCHEMA debe ser un identificador Oracle válido")
	}
	serviceUser, servicePassword := os.Getenv("ORACLE_USER"), os.Getenv("ORACLE_PASSWORD")
	if serviceUser == "" || servicePassword == "" {
		log.Fatal("ORACLE_USER y ORACLE_PASSWORD son obligatorios para consultar planilla_digital")
	}
	var err error
	s.serviceDB, err = s.openOracle(serviceUser, servicePassword)
	if err != nil {
		log.Fatalf("No fue posible conectar con la cuenta de servicio Oracle (%T): %v", err, err)
	}
	defer s.serviceDB.Close()
	go s.cleanExpiredSessions()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", s.login)
	mux.HandleFunc("/api/session", s.currentSession)
	mux.HandleFunc("/api/logout", s.logout)
	mux.HandleFunc("/api/planilla-digital", s.listPlanillaDigital)
	mux.HandleFunc("/api/v1/coberturas/planillas/", s.listCoveragePlanillas)
	mux.HandleFunc("/api/v1/coberturas/generar/", s.generateCoverageSheets)
	mux.HandleFunc("/api/v1/coberturas/generar-fecha/", s.generateCoverageSheetsAtDate)
	mux.HandleFunc("/api/v1/coberturas/descargar/", s.downloadCoverageSheets)
	mux.HandleFunc("/api/v1/coberturas/manual/", s.uploadManualCoverageSheets)
	mux.HandleFunc("/api/v1/ingesta/lote-dual", s.receiveDualUpload)
	mux.HandleFunc("/api/v1/ingesta/reemplazar-zip/", s.replaceStagedZIP)
	mux.HandleFunc("/api/v1/ingesta/procesar/", s.processStagedJob)
	mux.HandleFunc("/api/v1/ingesta/clasificar/", s.reclassifyStagedJob)
	mux.HandleFunc("/api/v1/ingesta/completar/", s.completeStagedJob)
	mux.HandleFunc("/api/v1/ingesta/estado/", s.getStagedJobStatus)
	mux.HandleFunc("/api/v1/ingesta/previsualizar/", s.getIngestPreview)
	mux.HandleFunc("/api/v1/ingesta/vincular-tramite/", s.setIngestTramiteMapping)
	mux.HandleFunc("/api/v1/expedientes", s.listWorkspaces)
	mux.HandleFunc("/api/v1/expedientes/eliminar/", s.deleteWorkspace)
	mux.HandleFunc("/api/v1/expedientes/documentos/renombrar/", s.renameWorkspacePDF)
	mux.HandleFunc("/api/v1/expedientes/documentos/fusionar/", s.mergeWorkspacePDFs)
	mux.HandleFunc("/api/v1/expedientes/documentos/archivo/", s.serveWorkspacePDF)
	mux.HandleFunc("/api/v1/expedientes/documentos/", s.workspaceDocuments)
	mux.HandleFunc("/api/v1/expedientes/descargar/", s.downloadWorkspaceZIP)
	mux.HandleFunc("/api/v1/objeciones/previsualizar", s.previewObjections)
	mux.HandleFunc("/api/v1/objeciones/crear", s.createObjectionWorkspace)
	mux.HandleFunc("/api/v1/objeciones/agregar/", s.addObjectionPatients)
	mux.HandleFunc("/api/v1/objeciones/sincronizar/", s.syncObjectionDocuments)
	mux.HandleFunc("/api/v1/objeciones/cabeceras/", s.uploadObjectionHeader)
	mux.HandleFunc("/api/v1/objeciones/documentos/", s.uploadObjectionDocument)
	mux.HandleFunc("/api/v1/objeciones/pdfs/seleccion/", s.objectionPDFSelection)
	mux.HandleFunc("/api/v1/objeciones/postura/", s.setObjectionPosture)
	mux.Handle("/", http.FileServer(http.Dir("dist")))

	addr := net.JoinHostPort(envOr("FOLIO_LISTEN_HOST", "127.0.0.1"), envOr("API_PORT", "8080"))
	log.Printf("API de Folio escuchando en %s", addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var input loginRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Username) == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "Escribe tu usuario y contraseña de Oracle.")
		return
	}

	db, err := s.openOracle(input.Username, input.Password)
	if err != nil {
		if oracleCredentialError(err) {
			writeError(w, http.StatusUnauthorized, "Oracle rechazó el usuario o la contraseña.")
			return
		}
		log.Printf("No fue posible validar un inicio de sesión contra Oracle (%T)", err)
		writeError(w, http.StatusServiceUnavailable, "No se pudo validar con Oracle. Inténtalo de nuevo más tarde.")
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	hasRequiredRole, err := oracleHasRole(ctx, db, requiredOracleRole)
	if err != nil {
		log.Printf("No fue posible comprobar el rol requerido de Oracle (%T)", err)
		writeError(w, http.StatusServiceUnavailable, "No se pudo comprobar el rol de Oracle. Inténtalo de nuevo más tarde.")
		return
	}
	if !hasRequiredRole {
		writeError(w, http.StatusForbidden, "Tu usuario Oracle no tiene asignado el rol SPD_EXTERNOS.")
		return
	}
	canDeleteWorkspaces, permissionErr := oracleHasRole(ctx, db, deleteWorkspaceOracleRole)
	if permissionErr != nil {
		log.Printf("No fue posible comprobar el permiso de borrado de períodos en Oracle (%T); se denegará ese permiso", permissionErr)
		canDeleteWorkspaces = false
	}

	sid, err := newSessionID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo iniciar la sesión.")
		return
	}

	s.mu.Lock()
	s.sessions[sid] = session{username: strings.TrimSpace(input.Username), canDeleteWorkspaces: canDeleteWorkspaces, expiresAt: time.Now().Add(sessionDuration)}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: "folio_session", Value: sid, Path: "/api", HttpOnly: true,
		Secure: s.cookieSecure || r.TLS != nil, SameSite: http.SameSiteStrictMode,
		Expires: time.Now().Add(sessionDuration), MaxAge: int(sessionDuration.Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"username": strings.TrimSpace(input.Username), "can_delete_workspaces": canDeleteWorkspaces})
}

func (s *server) openOracle(username, password string) (*sql.DB, error) {
	dsn := (&url.URL{
		Scheme: "oracle",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(s.host, s.port),
		Path:   "/" + s.service,
	}).String()
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(sessionDuration)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func oracleHasRole(ctx context.Context, db *sql.DB, role string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM USER_ROLE_PRIVS WHERE GRANTED_ROLE = :role",
		sql.Named("role", role),
	).Scan(&count)
	return count > 0, err
}

func (s *server) listPlanillaDigital(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	_, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	page := 1
	if rawPage := r.URL.Query().Get("page"); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed < 1 || parsed > 1_000_000 {
			writeError(w, http.StatusBadRequest, "El número de página no es válido.")
			return
		}
		page = parsed
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	tableName := oracleTableName(s.schema)
	var total int64
	if err := s.serviceDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tableName+" WHERE PDI_ASEGURADORA = 'MSP'").Scan(&total); err != nil {
		log.Printf("No fue posible contar %s (%T): %v", tableName, err, err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("No se pudo consultar %s. Comprueba el esquema y el permiso SELECT de ORACLE_USER.", tableName))
		return
	}
	totalPages := int((total + planillaPageSize - 1) / planillaPageSize)
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	firstRow := int64((page-1)*planillaPageSize + 1)
	lastRow := int64(page * planillaPageSize)
	query := fmt.Sprintf(`SELECT * FROM (
		SELECT t.*, ROW_NUMBER() OVER (ORDER BY t.ROWID) AS "__FOLIO_PAGE_ROW"
		FROM %s t
		WHERE t.PDI_ASEGURADORA = 'MSP'
	) WHERE "__FOLIO_PAGE_ROW" BETWEEN :first_row AND :last_row
	ORDER BY "__FOLIO_PAGE_ROW"`, tableName)
	rows, err := s.serviceDB.QueryContext(ctx, query,
		sql.Named("first_row", firstRow), sql.Named("last_row", lastRow),
	)
	if err != nil {
		log.Printf("No fue posible leer %s (%T): %v", tableName, err, err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("No se pudo consultar %s. Comprueba el esquema y el permiso SELECT de ORACLE_USER.", tableName))
		return
	}
	defer rows.Close()
	allColumns, err := rows.Columns()
	if err != nil || len(allColumns) == 0 {
		writeError(w, http.StatusInternalServerError, "No se pudieron leer las columnas de planilla_digital.")
		return
	}
	columns := allColumns[:len(allColumns)-1]
	resultRows := make([][]any, 0, planillaPageSize)
	for rows.Next() {
		values := make([]any, len(allColumns))
		scanTargets := make([]any, len(allColumns))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			writeError(w, http.StatusInternalServerError, "No se pudo leer una fila de planilla_digital.")
			return
		}
		resultRows = append(resultRows, values[:len(columns)])
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron leer las filas de planilla_digital.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"columns": columns, "rows": resultRows, "page": page,
		"pageSize": planillaPageSize, "total": total, "totalPages": totalPages,
	})
}

func oracleTableName(schema string) string {
	if schema == "" {
		return `"PLANILLA_DIGITAL"`
	}
	return `"` + strings.ReplaceAll(schema, `"`, `""`) + `"."PLANILLA_DIGITAL"`
}

func (s *server) currentSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	entry, ok := s.getSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Inicia sesión para continuar.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": entry.username, "can_delete_workspaces": entry.canDeleteWorkspaces})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Método no permitido.")
		return
	}
	if cookie, err := r.Cookie("folio_session"); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "folio_session", Value: "", Path: "/api", HttpOnly: true, Secure: s.cookieSecure || r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getSession(r *http.Request) (session, bool) {
	cookie, err := r.Cookie("folio_session")
	if err != nil {
		return session{}, false
	}
	s.mu.Lock()
	entry, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(s.sessions, cookie.Value)
		}
		s.mu.Unlock()
		return session{}, false
	}
	s.mu.Unlock()
	return entry, true
}

func (s *server) cleanExpiredSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, entry := range s.sessions {
			if now.After(entry.expiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func oracleCredentialError(err error) bool {
	var codeError interface{ Code() int }
	if errors.As(err, &codeError) {
		code := codeError.Code()
		return code == 1017 || code == 28000 || code == 28001 || code == 1045
	}
	message := strings.ToUpper(err.Error())
	return strings.Contains(message, "ORA-01017") || strings.Contains(message, "ORA-28000") || strings.Contains(message, "ORA-28001") || strings.Contains(message, "ORA-01045")
}

func newSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		log.Printf("%s no es válido; se usará el valor predeterminado", key)
		return fallback
	}
	return parsed
}
