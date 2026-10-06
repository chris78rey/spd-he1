# Graph Report - spd_msp  (2026-10-06)

## Corpus Check
- 37 files · ~76,669 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 2, .example 1, .zip 1)

## Summary
- 960 nodes · 1871 edges · 58 communities (40 shown, 18 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 195 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `25536a56`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- stagedJob
- generate_pdf.cjs
- ingesta.go
- workspace_delete_test.go
- package.json
- loadSavedWorkspaces
- N038: Guardián de cierre [REQUERIDO; NO IMPLEMENTADO; no integrado]
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- Documento de Diseño Técnico
- 3. Requisitos Funcionales Generales (RF)
- Find and recover documents
- Estado real de implementación: SPD MSP / Folio
- Documento de Diseño Técnico
- signOut
- Reglas OCR, clasificación y checklist ACFSS
- N015: Clasificador híbrido [REQUERIDO; IMPLEMENTADO; NO VERIFICADO E2E]
- N001: SPD MSP / Folio [REQUERIDO; IMPLEMENTADO; VERIFICADO básica]
- standardCodeForFilename
- Reglas funcionales de estructura ACFSS
- Estado real de implementación: SPD MSP / Folio
- 07_reglas_estructura_acfss.md
- objeciones.go
- R028: Workspace sincroniza documentos Oracle [IMPLEMENTADO parcialmente]
- Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?
- Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?
- N031: Coberturas [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- server/main.go
- N011: Previsualización de ingesta [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- AGENTS.md
- R014: Vínculo manual corrige trámite [IMPLEMENTADO]
- upload
- CFL_08: Oficio MSP citado, pero no incluido [CONFLICTO/PARCIAL]
- N007: Rol SPD_EXTERNOS [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N021: Catálogo MSP [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- install-service.sh
- dropWorkspaceFusion
- folio
- N006: Autenticación Oracle [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N013: PDI_PACIENTE [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N023: fuentes/ [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N024: trabajo/ [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N025: Reportes clasificación [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- N028: Sincronización Oracle documental [REQUERIDO; IMPLEMENTADO parcialmente; NO VERIFICADO]
- N032: Portal cobertura MSP [REQUERIDO; cliente implementado; NO VERIFICADO externo]
- N036: Primer ingreso [REQUERIDO; PARCIAL; estructura sí, cierre integral no]
- N040: Biblioteca IndexedDB [REQUERIDO de producto; IMPLEMENTADO; NO VERIFICADO; subsistema separado]
- N041: Graphify [artefacto implementado; herramienta de análisis]
- server
- go_pkg_encoding_xml
- objeciones_test.go
- clasificacion.go
- filteredSavedWorkspaces
- 6. Interfaz de Comunicación (APIs REST)
- sameObjectionPeriod

## God Nodes (most connected - your core abstractions)
1. `stagedJob` - 49 edges
2. `packageFolderName()` - 35 edges
3. `writeError()` - 35 edges
4. `writeJSON()` - 31 edges
5. `validJobID()` - 22 edges
6. `server` - 21 edges
7. `loadSavedWorkspaces()` - 20 edges
8. `jobResponse()` - 18 edges
9. `signOut()` - 17 edges
10. `generateSvgFromResult()` - 14 edges

## Surprising Connections (you probably didn't know these)
- `pendingPDFName()` --calls--> `safeOriginalFilename()`  [INFERRED]
  cmd/server/clasificacion.go → cmd/server/ingesta.go
- `validateCoveragePDF()` --calls--> `validateStagedFile()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/ingesta.go
- `jobResponse()` --calls--> `validPDFFile()`  [INFERRED]
  cmd/server/ingesta.go → cmd/server/objeciones.go
- `TestAppendObjectionRowsPreservesWorkAndInvalidatesExistingMatrix()` --calls--> `packageFolderName()`  [INFERRED]
  cmd/server/objeciones_test.go → cmd/server/ingesta.go
- `TestDownloadObjectionZIPIncludesOnlyMarkedClinicalPDFs()` --calls--> `packageFolderName()`  [INFERRED]
  cmd/server/objeciones_test.go → cmd/server/ingesta.go

## Import Cycles
- None detected.

## Communities (58 total, 18 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.01
Nodes (180): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, completionFields, completionFiles (+172 more)

### Community 1 - "stagedJob"
Cohesion: 0.08
Nodes (54): coverageMembers(), server, atomicWritePrivateFile(), copyPrivateFile(), server, server, headerOutputFilename(), jobResponse() (+46 more)

### Community 2 - "generate_pdf.cjs"
Cohesion: 0.05
Nodes (70): ref_crypto, ref_fs, ref_https, ref_path, applyChromeToAllPages(), buildFooterImages(), buildSvgPage(), drawFooter() (+62 more)

### Community 3 - "ingesta.go"
Cohesion: 0.08
Nodes (36): containsInt64(), backupAndReplace(), newUploadJobID(), periodWorkspaceID(), replaceWorkspaceOutputs(), sha256Upload(), validServiceCode(), readMultipartBytes() (+28 more)

### Community 4 - "workspace_delete_test.go"
Cohesion: 0.29
Nodes (6): TestDeleteObjectionWorkspaceLeavesSourceUntouched(), TestListWorkspacesSkipsHiddenDeletionRecoveryDirectories(), go_pkg_bytes, go_pkg_net_http_httptest, go_pkg_strings, go_pkg_testing

### Community 5 - "package.json"
Cohesion: 0.06
Nodes (31): dependencies, crypto-js, idb, @mdi/font, pdfkit, svg-to-pdfkit, vue, vuetify (+23 more)

### Community 6 - "loadSavedWorkspaces"
Cohesion: 0.13
Nodes (26): addMissingDocuments(), checkSession(), classifyIngest(), clearIngestTramiteMapping(), confirmMergeAllPending(), confirmWorkspaceFusion(), deleteWholeWorkspace(), forgetUnavailableIngest() (+18 more)

### Community 7 - "N038: Guardián de cierre [REQUERIDO; NO IMPLEMENTADO; no integrado]"
Cohesion: 0.09
Nodes (22): CFL_01: Documentación dice fusión pendiente, pero se describe handler actual con fusión; cronología automática no demostrada [CONFLICTO/PARCIAL], CFL_02: Requisito de fusión cronológica frente a implementación con orden humano [CONFLICTO/PARCIAL], CFL_03: SRS propone actualizar PDI_ESTADO_DIGITALIZACION; estado operativo difiere [CONFLICTO/PARCIAL], CFL_04: Especificación del Guardián describe endpoints ausentes en main.go [CONFLICTO/PARCIAL], CFL_05: Diseño de objeciones sin flujo en rutas [CONFLICTO/PARCIAL], CFL_06: SRS requiere habilitantes al ingreso, pero el flujo permite trabajar antes del cierre [CONFLICTO/PARCIAL], CFL_07: ZIP llamado final puede originarse en workspace INCOMPLETE y no ser paquete formal [CONFLICTO/PARCIAL], N022: Expediente del paciente [REQUERIDO; IMPLEMENTADO; NO VERIFICADO] (+14 more)

### Community 8 - "Documento de Diseño Técnico"
Cohesion: 0.11
Nodes (18): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+10 more)

### Community 9 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): **1. Obtener Expedientes Pendientes de Subsanación**, 1. Stack Tecnológico y Dependencias, **2. Clasificación Manual de PDF No Identificado**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Inyección Puntual de Documento Faltante**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+11 more)

### Community 10 - "Documento de Diseño Técnico"
Cohesion: 0.17
Nodes (12): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, **A. Estructuras de Datos en Go (`domain/empaquetado.go`)**, Decisiones Pendientes, Documento de Diseño Técnico (+4 more)

### Community 11 - "Documento de Diseño Técnico"
Cohesion: 0.13
Nodes (14): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs), Decisiones Pendientes, Documento de Diseño Técnico (+6 more)

### Community 12 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Métodos Go), **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**, **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)** (+11 more)

### Community 13 - "3. Requisitos Funcionales Generales (RF)"
Cohesion: 0.17
Nodes (12): 1. Visión General y Objetivo del Sistema, 2. Arquitectura y Stack Tecnológico Aprobado, 3. Requisitos Funcionales Generales (RF), 4. Requisitos No Funcionales (RNF), 5. Resumen de Módulos Técnicos para el Equipo de Desarrollo, Documento de Requerimientos Generales del Sistema (SRS), **RF-01: Ingesta Dual de Información**, **RF-02: Resolución de Identidades y Normalización de Nombres** (+4 more)

### Community 14 - "Find and recover documents"
Cohesion: 0.24
Nodes (12): Accessible responsive use, Batch organization, Complete action feedback, Recover from mistakes, Find and recover documents, Folio product, Measure and validate, Primary user outcome (+4 more)

### Community 15 - "Estado real de implementación: SPD MSP / Folio"
Cohesion: 0.17
Nodes (11): Answer, Clústeres funcionales, Conflictos CFL-01–CFL-08, Criterio de evidencia, Estado real de implementación: SPD MSP / Folio, Mapa global y conclusión, Nodos N001–N043, Outcome (+3 more)

### Community 16 - "Documento de Diseño Técnico"
Cohesion: 0.12
Nodes (16): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Eventos Internos), **A. Archivo de Reglas YAML (`reglas_clasificacion.yaml`)**, **A. Interfaz Go para Invocación del Pipeline (`pipeline/processor.go`)** (+8 more)

### Community 17 - "signOut"
Cohesion: 0.13
Nodes (21): addObjectionPatients(), chooseObjectionWorkspace(), createObjectionWorkspace(), downloadSelectedCoverageSheets(), generateCoverageSheets(), loadCoveragePlanillas(), loadObjectionPDFSelection(), loadObjectionWorkspace() (+13 more)

### Community 18 - "Reglas OCR, clasificación y checklist ACFSS"
Cohesion: 0.18
Nodes (11): 1. Entrada, extracción y clasificación, 2. Catálogo de señales confirmadas, 3. Catálogo de nombres de archivo conocidos, 4. Matriz de documentos requeridos por servicio, 5. Duplicados y documentos de varias páginas, 6. Banco de pruebas anonimizado, 7. Objeciones y límites de implementación, 8. Añadir, renombrar y reemplazar PDFs desde la bandeja (+3 more)

### Community 19 - "N015: Clasificador híbrido [REQUERIDO; IMPLEMENTADO; NO VERIFICADO E2E]"
Cohesion: 0.18
Nodes (11): N015: Clasificador híbrido [REQUERIDO; IMPLEMENTADO; NO VERIFICADO E2E], N016: reglas_clasificacion.yaml [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N017: Extracción vectorial [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N018: OCR Tesseract [REQUERIDO; IMPLEMENTADO; NO VERIFICADO ambiental; depende del SO], N019: Alertas de fecha [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N020: PENDIENTE_tmp_* [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R017: Clasificador usa YAML [IMPLEMENTADO], R018: Clasificador prioriza vectorial [IMPLEMENTADO] (+3 more)

### Community 20 - "N001: SPD MSP / Folio [REQUERIDO; IMPLEMENTADO; VERIFICADO básica]"
Cohesion: 0.22
Nodes (9): N001: SPD MSP / Folio [REQUERIDO; IMPLEMENTADO; VERIFICADO básica], N002: ACFSS [REQUERIDO; dominio del sistema], N003: Frontend Vue/Vuetify [REQUERIDO; IMPLEMENTADO; VERIFICADO build], N004: Backend Go [REQUERIDO; IMPLEMENTADO; VERIFICADO build + HTTP 200], N005: PLANILLA_DIGITAL [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R001: Folio automatiza ACFSS [REQUERIDO/IMPLEMENTADO; finalidad general parcial], R002: Folio usa Vue/Vuetify [IMPLEMENTADO], R003: Folio usa Go [IMPLEMENTADO] (+1 more)

### Community 21 - "standardCodeForFilename"
Cohesion: 0.18
Nodes (11): addWorkspacePDFs(), deleteWorkspacePDF(), openObjectionPatientDocuments(), openPatientDocumentsWorkspace(), openWorkspaceFusion(), queuedWorkspaceDuplicates, reviewPendingFusion(), selectWorkspacePatient() (+3 more)

### Community 22 - "Reglas funcionales de estructura ACFSS"
Cohesion: 0.25
Nodes (8): 1. Tipos de servicio y paquetes, 2. Identidad y nombres, 3. Primer ingreso, 4. Levantamiento de objeciones, 5. Guardián de cierre, 6. Estado de implementación, Documentos del expediente, Reglas funcionales de estructura ACFSS

### Community 23 - "Estado real de implementación: SPD MSP / Folio"
Cohesion: 0.25
Nodes (7): Clústeres funcionales, Conflictos CFL-01–CFL-08, Criterio de evidencia, Estado real de implementación: SPD MSP / Folio, Mapa global y conclusión, Nodos N001–N043, Relaciones R001–R040

### Community 24 - "07_reglas_estructura_acfss.md"
Cohesion: 0.28
Nodes (3): Desarrollo local, Inicio automático en Linux, SPD MSP

### Community 25 - "objeciones.go"
Cohesion: 0.12
Nodes (25): buildSelectedObjectionRows(), copyObjectionPDFTree(), nextSafeAnnexName(), normalizeAnnexFilename(), normalizeCedulaValue(), normalizedWorkspaceRelativePath(), objectionAnnexBase(), objectionHeaderName() (+17 more)

### Community 26 - "R028: Workspace sincroniza documentos Oracle [IMPLEMENTADO parcialmente]"
Cohesion: 0.33
Nodes (6): N008: Período + servicio [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N009: Workspace del período [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N029: PDI_DOCUMENTO_DIGITAL [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N030: PDI_PATH [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R007: Período y servicio identifican workspace [IMPLEMENTADO], R028: Workspace sincroniza documentos Oracle [IMPLEMENTADO parcialmente]

### Community 27 - "Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?, Source Nodes

### Community 28 - "Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?, Source Nodes

### Community 29 - "N031: Coberturas [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]"
Cohesion: 0.40
Nodes (5): N031: Coberturas [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N033: C_COBERTURA.pdf [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N034: PDI_COBERTURA [REQUERIDO; IMPLEMENTADO; NO VERIFICADO; escritura Oracle programada], R033: Cobertura genera C_COBERTURA [IMPLEMENTADO], R035: Cobertura actualiza PDI_COBERTURA [IMPLEMENTADO]

### Community 30 - "server/main.go"
Cohesion: 0.13
Nodes (24): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+16 more)

### Community 31 - "N011: Previsualización de ingesta [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]"
Cohesion: 0.50
Nodes (4): N010: ZIP clínico [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N011: Previsualización de ingesta [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R012: Previsualización inspecciona ZIP [IMPLEMENTADO], R013: Previsualización contrasta con Oracle [IMPLEMENTADO]

### Community 33 - "R014: Vínculo manual corrige trámite [IMPLEMENTADO]"
Cohesion: 0.67
Nodes (3): N012: PDI_TRAMITE [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N014: Vinculación manual trámite [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R014: Vínculo manual corrige trámite [IMPLEMENTADO]

### Community 34 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 51 - "server"
Cohesion: 0.13
Nodes (16): parseOracleDate(), envInt64(), envOr(), server, main(), newSessionID(), oracleCredentialError(), oracleHasRole() (+8 more)

### Community 53 - "objeciones_test.go"
Cohesion: 0.22
Nodes (20): installObjectionRows(), objectionMatrixFilename(), objectionPatientFolder(), server, persistTestObjectionJob(), TestAppendObjectionRowsPreservesWorkAndInvalidatesExistingMatrix(), TestBuildSelectedObjectionRowsIncludesOnlyChosenCandidates(), TestCreateObjectionWorkspaceReportsExistingDuplicatesWithoutCreatingAnother() (+12 more)

### Community 54 - "clasificacion.go"
Cohesion: 0.14
Nodes (24): addDateCandidateForPeriod(), classifyPDF(), copyWithLimit(), datesOutsideBilledPeriod(), extractPDFText(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules() (+16 more)

### Community 55 - "filteredSavedWorkspaces"
Cohesion: 0.19
Nodes (16): coverageMonthLabel(), coverageServiceLabel(), filteredSavedWorkspaces, normalizeWorkspaceSearch(), objectionSourceItems, objectionWorkspaceItems, orderedSavedWorkspaces, savedWorkspaceItems (+8 more)

### Community 56 - "6. Interfaz de Comunicación (APIs REST)"
Cohesion: 0.50
Nodes (4): **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**, **2. Compilar Paquete Final ISO / ZIP**, **3. Obtener Metadatos para Rotulado Físico de CD/DVD**, 6. Interfaz de Comunicación (APIs REST)

### Community 57 - "sameObjectionPeriod"
Cohesion: 0.67
Nodes (3): objectionPeriodSpaces, objectionSourcePeriodSpaces, sameObjectionPeriod()

## Knowledge Gaps
- **387 isolated node(s):** `coverageFailureItem`, `coverageGenerateRequest`, `loginRequest`, `objectionPackagePDF`, `oracleDocument` (+382 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 455 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **18 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **Why does `idb` connect `package.json` to `App.vue`?**
  _High betweenness centrality (0.034) - this node is a cross-community bridge._
- **Why does `crypto-js` connect `package.json` to `generate_pdf.cjs`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Are the 24 inferred relationships involving `packageFolderName()` (e.g. with `objectionHeaderStatuses()` and `TestAppendObjectionRowsPreservesWorkAndInvalidatesExistingMatrix()`) actually correct?**
  _`packageFolderName()` has 24 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `writeError()` (e.g. with `.addObjectionPatients()` and `.addWorkspacePDFs()`) actually correct?**
  _`writeError()` has 28 INFERRED edges - model-reasoned connections that need verification._
- **What connects `coverageFailureItem`, `coverageGenerateRequest`, `loginRequest` to the rest of the system?**
  _387 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.00904977375565611 - nodes in this community are weakly interconnected._