# Graph Report - spd_msp  (2026-10-02)

## Corpus Check
- 32 files · ~55,993 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 2, .example 1, .zip 1)

## Summary
- 682 nodes · 1300 edges · 29 communities (25 shown, 4 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 117 edges (avg confidence: 0.84)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `64d13a01`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- writeError
- 3. Requisitos Funcionales Generales (RF)
- package.json
- 2. Contexto y Flujo de Datos
- stagedJob
- Documento de Diseño Técnico
- ingesta.go
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- loadSavedWorkspaces
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Find and recover documents
- standardCodeForFilename
- generate_pdf.cjs
- AGENTS.md
- Reglas OCR, clasificación y checklist ACFSS
- loadCoveragePlanillas
- upload
- install-service.sh
- 07_reglas_estructura_acfss.md
- folio
- Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?
- Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?
- Reglas funcionales de estructura ACFSS
- 2. Contexto y Flujo de Datos
- dropWorkspaceFusion

## God Nodes (most connected - your core abstractions)
1. `stagedJob` - 30 edges
2. `writeError()` - 28 edges
3. `writeJSON()` - 24 edges
4. `server` - 20 edges
5. `packageFolderName()` - 20 edges
6. `validJobID()` - 15 edges
7. `generateSvgFromResult()` - 14 edges
8. `loadSavedWorkspaces()` - 13 edges
9. `jobResponse()` - 12 edges
10. `classifyPDF()` - 11 edges

## Surprising Connections (you probably didn't know these)
- `pendingPDFName()` --calls--> `safeOriginalFilename()`  [INFERRED]
  cmd/server/clasificacion.go → cmd/server/ingesta.go
- `validateCoveragePDF()` --calls--> `extractPDFText()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/clasificacion.go
- `validateCoveragePDF()` --calls--> `normalizeOCRText()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/clasificacion.go
- `validateCoveragePDF()` --calls--> `validateStagedFile()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/ingesta.go
- `summarizeClassification()` --references--> `classificationResult`  [EXTRACTED]
  cmd/server/ingesta.go → cmd/server/clasificacion.go

## Import Cycles
- None detected.

## Communities (29 total, 4 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.01
Nodes (126): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, completionFields, completionFiles (+118 more)

### Community 1 - "writeError"
Cohesion: 0.13
Nodes (29): server, jobResponse(), missingHeaders(), packageFolderName(), periodWorkspaceID(), safeOriginalFilename(), saveUploadPart(), sha256Path() (+21 more)

### Community 2 - "3. Requisitos Funcionales Generales (RF)"
Cohesion: 0.15
Nodes (12): 1. Visión General y Objetivo del Sistema, 2. Arquitectura y Stack Tecnológico Aprobado, 3. Requisitos Funcionales Generales (RF), 4. Requisitos No Funcionales (RNF), 5. Resumen de Módulos Técnicos para el Equipo de Desarrollo, Documento de Requerimientos Generales del Sistema (SRS), **RF-01: Ingesta Dual de Información**, **RF-02: Resolución de Identidades y Normalización de Nombres** (+4 more)

### Community 3 - "package.json"
Cohesion: 0.06
Nodes (31): dependencies, crypto-js, idb, @mdi/font, pdfkit, svg-to-pdfkit, vue, vuetify (+23 more)

### Community 4 - "2. Contexto y Flujo de Datos"
Cohesion: 0.50
Nodes (4): 2. Contexto y Flujo de Datos, **Flujos de Entrada (Inputs):**, **Flujos de Salida (Outputs):**, **Problema que resuelve:**

### Community 5 - "stagedJob"
Cohesion: 0.07
Nodes (46): addDateCandidateForPeriod(), classifyPDF(), copyWithLimit(), datesOutsideBilledPeriod(), extractPDFText(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules() (+38 more)

### Community 6 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Métodos Go), **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**, **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)** (+11 more)

### Community 7 - "ingesta.go"
Cohesion: 0.06
Nodes (65): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+57 more)

### Community 8 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 9 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 10 - "loadSavedWorkspaces"
Cohesion: 0.16
Nodes (22): addMissingDocuments(), checkSession(), classifyIngest(), clearIngestTramiteMapping(), confirmMergeAllPending(), confirmWorkspaceFusion(), deleteWholeWorkspace(), forgetUnavailableIngest() (+14 more)

### Community 12 - "Documento de Diseño Técnico"
Cohesion: 0.17
Nodes (12): **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**, 1. Stack Tecnológico y Dependencias, **2. Compilar Paquete Final ISO / ZIP**, 3. Estructura de Directorios Sugerida, **3. Obtener Metadatos para Rotulado Físico de CD/DVD**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs REST) (+4 more)

### Community 13 - "Documento de Diseño Técnico"
Cohesion: 0.12
Nodes (16): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Eventos Internos), **A. Archivo de Reglas YAML (`reglas_clasificacion.yaml`)**, **A. Interfaz Go para Invocación del Pipeline (`pipeline/processor.go`)** (+8 more)

### Community 14 - "Documento de Diseño Técnico"
Cohesion: 0.20
Nodes (10): 1. Stack Tecnológico y Dependencias, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs), Decisiones Pendientes, Documento de Diseño Técnico, **Endpoint Principal: Envío de Carga Dual de Lote Mensual** (+2 more)

### Community 16 - "Find and recover documents"
Cohesion: 0.24
Nodes (12): Accessible responsive use, Batch organization, Complete action feedback, Recover from mistakes, Find and recover documents, Folio product, Measure and validate, Primary user outcome (+4 more)

### Community 17 - "standardCodeForFilename"
Cohesion: 0.22
Nodes (9): addWorkspacePDFs(), deleteWorkspacePDF(), openWorkspaceFusion(), queuedWorkspaceDuplicates, reviewPendingFusion(), selectWorkspacePatient(), selectWorkspacePDF(), selectWorkspacePDFs() (+1 more)

### Community 18 - "generate_pdf.cjs"
Cohesion: 0.05
Nodes (70): ref_crypto, ref_fs, ref_https, ref_path, applyChromeToAllPages(), buildFooterImages(), buildSvgPage(), drawFooter() (+62 more)

### Community 20 - "Reglas OCR, clasificación y checklist ACFSS"
Cohesion: 0.18
Nodes (11): 1. Entrada, extracción y clasificación, 2. Catálogo de señales confirmadas, 3. Catálogo de nombres de archivo conocidos, 4. Matriz de documentos requeridos por servicio, 5. Duplicados y documentos de varias páginas, 6. Banco de pruebas anonimizado, 7. Objeciones y límites de implementación, 8. Añadir, renombrar y reemplazar PDFs desde la bandeja (+3 more)

### Community 21 - "loadCoveragePlanillas"
Cohesion: 0.28
Nodes (9): coverageMonthLabel(), coverageServiceLabel(), generateCoverageSheets(), loadCoveragePlanillas(), loadPlanilla(), openCoverageDownloads(), openPlanilla(), signOut() (+1 more)

### Community 22 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 25 - "07_reglas_estructura_acfss.md"
Cohesion: 0.28
Nodes (3): Desarrollo local, Inicio automático en Linux, SPD MSP

### Community 27 - "Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?, Source Nodes

### Community 28 - "Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?, Source Nodes

### Community 29 - "Reglas funcionales de estructura ACFSS"
Cohesion: 0.25
Nodes (8): 1. Tipos de servicio y paquetes, 2. Identidad y nombres, 3. Primer ingreso, 4. Levantamiento de objeciones, 5. Guardián de cierre, 6. Estado de implementación, Documentos del expediente, Reglas funcionales de estructura ACFSS

### Community 31 - "2. Contexto y Flujo de Datos"
Cohesion: 0.50
Nodes (4): 2. Contexto y Flujo de Datos, **Flujos de Entrada (Inputs):**, **Flujos de Salida (Outputs):**, **Problema que resuelve:**

## Knowledge Gaps
- **275 isolated node(s):** `coverageFailureItem`, `coverageGenerateRequest`, `loginRequest`, `oracleDocument`, `mspPDFCode` (+270 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 333 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Why does `idb` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.045) - this node is a cross-community bridge._
- **Why does `crypto-js` connect `package.json` to `generate_pdf.cjs`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `writeError()` (e.g. with `.addWorkspacePDFs()` and `.completeStagedJob()`) actually correct?**
  _`writeError()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **Are the 18 inferred relationships involving `writeJSON()` (e.g. with `.addWorkspacePDFs()` and `.completeStagedJob()`) actually correct?**
  _`writeJSON()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **What connects `coverageFailureItem`, `coverageGenerateRequest`, `loginRequest` to the rest of the system?**
  _275 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.012658227848101266 - nodes in this community are weakly interconnected._