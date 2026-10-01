# Graph Report - documentos  (2026-10-01)

## Corpus Check
- 24 files · ~38,298 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 2, .example 1, .css 1)

## Summary
- 499 nodes · 861 edges · 25 communities (22 shown, 3 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 54 edges (avg confidence: 0.83)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `21aa458f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- package.json
- Find and recover documents
- 07_reglas_estructura_acfss.md
- demo-lote/main.go
- upload
- AGENTS.md
- writeError
- ingesta.go
- loadSavedWorkspaces
- folio
- loadPlanilla
- install-service.sh
- Documento de Diseño Técnico
- clasificacion.go
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- server/main.go
- server
- .buildPatientFolders
- standardCodeForFilename
- Reglas OCR, clasificación y checklist ACFSS
- .loadPlanillaIdentities

## God Nodes (most connected - your core abstractions)
1. `writeError()` - 21 edges
2. `stagedJob` - 20 edges
3. `server` - 18 edges
4. `writeJSON()` - 18 edges
5. `packageFolderName()` - 15 edges
6. `classifyPDF()` - 12 edges
7. `jobResponse()` - 11 edges
8. `validJobID()` - 11 edges
9. `server` - 11 edges
10. `loadWorkspaceDocuments()` - 9 edges

## Surprising Connections (you probably didn't know these)
- `pendingPDFName()` --calls--> `safeOriginalFilename()`  [INFERRED]
  cmd/server/clasificacion.go → cmd/server/ingesta.go
- `summarizeClassification()` --references--> `classificationResult`  [EXTRACTED]
  cmd/server/ingesta.go → cmd/server/clasificacion.go
- `classificationReport` --references--> `classificationResult`  [EXTRACTED]
  cmd/server/ingesta.go → cmd/server/clasificacion.go
- `summarizeClassification()` --references--> `classificationSummary`  [EXTRACTED]
  cmd/server/ingesta.go → cmd/server/clasificacion.go
- `classificationReport` --references--> `classificationSummary`  [EXTRACTED]
  cmd/server/ingesta.go → cmd/server/clasificacion.go

## Import Cycles
- None detected.

## Communities (25 total, 3 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.02
Nodes (94): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, completionFields, completionFiles (+86 more)

### Community 1 - "package.json"
Cohesion: 0.08
Nodes (25): dependencies, idb, @mdi/font, vue, vuetify, devDependencies, vite, @vitejs/plugin-vue (+17 more)

### Community 2 - "Find and recover documents"
Cohesion: 0.24
Nodes (12): Accessible responsive use, Batch organization, Complete action feedback, Recover from mistakes, Find and recover documents, Folio product, Measure and validate, Primary user outcome (+4 more)

### Community 3 - "07_reglas_estructura_acfss.md"
Cohesion: 0.05
Nodes (37): Desarrollo local, Folio, Inicio automático en Linux, 1. Visión General y Objetivo del Sistema, 2. Arquitectura y Stack Tecnológico Aprobado, 3. Requisitos Funcionales Generales (RF), 4. Requisitos No Funcionales (RNF), 5. Resumen de Módulos Técnicos para el Equipo de Desarrollo (+29 more)

### Community 4 - "demo-lote/main.go"
Cohesion: 0.15
Nodes (18): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+10 more)

### Community 5 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 7 - "writeError"
Cohesion: 0.17
Nodes (23): copyPrivateFile(), server, headerOutputFilename(), jobResponse(), matrixFilename(), missingHeaders(), packageFolderName(), safeOriginalFilename() (+15 more)

### Community 8 - "ingesta.go"
Cohesion: 0.13
Nodes (14): periodWorkspaceID(), sha256Upload(), validServiceCode(), go_pkg_crypto_sha256, go_pkg_encoding_hex, go_pkg_mime_multipart, go_pkg_path, go_pkg_time (+6 more)

### Community 9 - "loadSavedWorkspaces"
Cohesion: 0.25
Nodes (15): addMissingDocuments(), checkSession(), classifyIngest(), deleteWholeWorkspace(), forgetUnavailableIngest(), loadDocuments(), loadIngestPreview(), loadSavedWorkspaces() (+7 more)

### Community 11 - "loadPlanilla"
Cohesion: 0.67
Nodes (3): loadPlanilla(), openPlanilla(), signOut()

### Community 13 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 14 - "clasificacion.go"
Cohesion: 0.24
Nodes (16): addDateCandidate(), classifyPDF(), datesOutsideCareInterval(), extractPDFText(), formatCareInterval(), loadClassificationRules(), matchClassification(), normalizeOCRText() (+8 more)

### Community 15 - "Documento de Diseño Técnico"
Cohesion: 0.12
Nodes (16): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Eventos Internos), **A. Archivo de Reglas YAML (`reglas_clasificacion.yaml`)**, **A. Interfaz Go para Invocación del Pipeline (`pipeline/processor.go`)** (+8 more)

### Community 16 - "Documento de Diseño Técnico"
Cohesion: 0.12
Nodes (16): **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**, 1. Stack Tecnológico y Dependencias, **2. Compilar Paquete Final ISO / ZIP**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Obtener Metadatos para Rotulado Físico de CD/DVD**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+8 more)

### Community 17 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Métodos Go), **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**, **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)** (+11 more)

### Community 18 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 19 - "server/main.go"
Cohesion: 0.12
Nodes (21): go_pkg_archive_zip, go_pkg_context, go_pkg_crypto_rand, go_pkg_encoding_base64, go_pkg_encoding_json, go_pkg_errors, go_pkg_fmt, go_pkg_github_com_joho_godotenv (+13 more)

### Community 20 - "server"
Cohesion: 0.19
Nodes (11): envInt64(), envOr(), server, main(), newSessionID(), oracleCredentialError(), oracleHasRole(), securityHeaders() (+3 more)

### Community 21 - ".buildPatientFolders"
Cohesion: 0.18
Nodes (12): copyWithLimit(), normalizePatientFolder(), replaceWorkspaceOutputs(), summarizeClassification(), writePrivateJSON(), copyReplacePrivateFile(), io.Reader, io.Writer (+4 more)

### Community 22 - "standardCodeForFilename"
Cohesion: 0.33
Nodes (6): addWorkspacePDFs(), deleteWorkspacePDF(), selectWorkspacePatient(), selectWorkspacePDF(), selectWorkspacePDFs(), standardCodeForFilename()

### Community 23 - "Reglas OCR, clasificación y checklist ACFSS"
Cohesion: 0.18
Nodes (11): 1. Entrada, extracción y clasificación, 2. Catálogo de señales confirmadas, 3. Catálogo de nombres de archivo conocidos, 4. Matriz de documentos requeridos por servicio, 5. Duplicados y documentos de varias páginas, 6. Banco de pruebas anonimizado, 7. Objeciones y límites de implementación, 8. Añadir, renombrar y reemplazar PDFs desde la bandeja (+3 more)

### Community 24 - ".loadPlanillaIdentities"
Cohesion: 0.40
Nodes (4): parseOracleDate(), oracleTableName(), database/sql.NullString, planillaIdentity

## Knowledge Gaps
- **215 isolated node(s):** `loginRequest`, `mspPDFCode`, `folio`, `name`, `version` (+210 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 266 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Documento de Diseño Técnico` connect `Documento de Diseño Técnico` to `07_reglas_estructura_acfss.md`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Are the 14 inferred relationships involving `writeError()` (e.g. with `.addWorkspacePDFs()` and `.completeStagedJob()`) actually correct?**
  _`writeError()` has 14 INFERRED edges - model-reasoned connections that need verification._
- **What connects `loginRequest`, `mspPDFCode`, `folio` to the rest of the system?**
  _215 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.016260162601626018 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.07936507936507936 - nodes in this community are weakly interconnected._
- **Should `07_reglas_estructura_acfss.md` be split into smaller, more focused modules?**
  _Cohesion score 0.048726467331118496 - nodes in this community are weakly interconnected._
- **Should `ingesta.go` be split into smaller, more focused modules?**
  _Cohesion score 0.1323529411764706 - nodes in this community are weakly interconnected._