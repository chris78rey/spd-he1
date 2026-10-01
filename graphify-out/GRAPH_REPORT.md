# Graph Report - spd_msp  (2026-10-01)

## Corpus Check
- 24 files · ~37,935 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 4 file(s) not represented in the graph (top: (none) 2, .example 1, .css 1)

## Summary
- 502 nodes · 866 edges · 20 communities (17 shown, 3 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 54 edges (avg confidence: 0.83)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `fece9ba9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- package.json
- Find and recover documents
- Documento de Diseño Técnico
- Reglas OCR, clasificación y checklist ACFSS
- upload
- AGENTS.md
- .buildPatientFolders
- loadSavedWorkspaces
- folio
- loadPlanilla
- install-service.sh
- Documento de Diseño Técnico
- ingesta.go
- server/main.go
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- 07_reglas_estructura_acfss.md
- standardCodeForFilename

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

## Communities (20 total, 3 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.02
Nodes (96): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, completionFields, completionFiles (+88 more)

### Community 1 - "package.json"
Cohesion: 0.08
Nodes (25): dependencies, idb, @mdi/font, vue, vuetify, devDependencies, vite, @vitejs/plugin-vue (+17 more)

### Community 2 - "Find and recover documents"
Cohesion: 0.24
Nodes (12): Accessible responsive use, Batch organization, Complete action feedback, Recover from mistakes, Find and recover documents, Folio product, Measure and validate, Primary user outcome (+4 more)

### Community 3 - "Documento de Diseño Técnico"
Cohesion: 0.13
Nodes (14): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs), Decisiones Pendientes, Documento de Diseño Técnico (+6 more)

### Community 4 - "Reglas OCR, clasificación y checklist ACFSS"
Cohesion: 0.18
Nodes (11): 1. Entrada, extracción y clasificación, 2. Catálogo de señales confirmadas, 3. Catálogo de nombres de archivo conocidos, 4. Matriz de documentos requeridos por servicio, 5. Duplicados y documentos de varias páginas, 6. Banco de pruebas anonimizado, 7. Objeciones y límites de implementación, 8. Añadir, renombrar y reemplazar PDFs desde la bandeja (+3 more)

### Community 5 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 7 - ".buildPatientFolders"
Cohesion: 0.17
Nodes (23): copyPrivateFile(), server, headerOutputFilename(), jobResponse(), matrixFilename(), missingHeaders(), normalizePatientFolder(), packageFolderName() (+15 more)

### Community 9 - "loadSavedWorkspaces"
Cohesion: 0.21
Nodes (17): addMissingDocuments(), checkSession(), classifyIngest(), deleteWholeWorkspace(), forgetUnavailableIngest(), loadDocuments(), loadIngestPreview(), loadSavedWorkspaces() (+9 more)

### Community 11 - "loadPlanilla"
Cohesion: 0.67
Nodes (3): loadPlanilla(), openPlanilla(), signOut()

### Community 13 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 14 - "ingesta.go"
Cohesion: 0.06
Nodes (48): addDateCandidate(), classifyPDF(), copyWithLimit(), datesOutsideCareInterval(), extractPDFText(), formatCareInterval(), loadClassificationRules(), matchClassification() (+40 more)

### Community 15 - "server/main.go"
Cohesion: 0.06
Nodes (48): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+40 more)

### Community 16 - "Documento de Diseño Técnico"
Cohesion: 0.12
Nodes (16): **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**, 1. Stack Tecnológico y Dependencias, **2. Compilar Paquete Final ISO / ZIP**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Obtener Metadatos para Rotulado Físico de CD/DVD**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+8 more)

### Community 17 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Métodos Go), **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**, **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)** (+11 more)

### Community 18 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 19 - "07_reglas_estructura_acfss.md"
Cohesion: 0.05
Nodes (39): Desarrollo local, Folio, Inicio automático en Linux, 1. Visión General y Objetivo del Sistema, 2. Arquitectura y Stack Tecnológico Aprobado, 3. Requisitos Funcionales Generales (RF), 4. Requisitos No Funcionales (RNF), 5. Resumen de Módulos Técnicos para el Equipo de Desarrollo (+31 more)

### Community 22 - "standardCodeForFilename"
Cohesion: 0.33
Nodes (6): addWorkspacePDFs(), deleteWorkspacePDF(), selectWorkspacePatient(), selectWorkspacePDF(), selectWorkspacePDFs(), standardCodeForFilename()

## Knowledge Gaps
- **217 isolated node(s):** `loginRequest`, `mspPDFCode`, `folio`, `name`, `version` (+212 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 267 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Are the 14 inferred relationships involving `writeError()` (e.g. with `.addWorkspacePDFs()` and `.completeStagedJob()`) actually correct?**
  _`writeError()` has 14 INFERRED edges - model-reasoned connections that need verification._
- **What connects `loginRequest`, `mspPDFCode`, `folio` to the rest of the system?**
  _217 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.016129032258064516 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.07936507936507936 - nodes in this community are weakly interconnected._
- **Should `Documento de Diseño Técnico` be split into smaller, more focused modules?**
  _Cohesion score 0.13333333333333333 - nodes in this community are weakly interconnected._
- **Should `Documento de Diseño Técnico` be split into smaller, more focused modules?**
  _Cohesion score 0.10526315789473684 - nodes in this community are weakly interconnected._
- **Should `ingesta.go` be split into smaller, more focused modules?**
  _Cohesion score 0.06095791001451379 - nodes in this community are weakly interconnected._