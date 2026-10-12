# Graph Report - spd_msp  (2026-10-12)

## Corpus Check
- 55 files · ~157,479 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 24 file(s) not represented in the graph (top: .css 5, .ttf 3, .eot 3)

## Summary
- 5670 nodes · 22786 edges · 105 communities (86 shown, 19 thin omitted)
- Extraction: 77% EXTRACTED · 23% INFERRED · 0% AMBIGUOUS · INFERRED: 5211 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `80f0701b`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- packageFolderName
- generate_pdf.cjs
- _$
- _$
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
- $_
- Reglas funcionales de estructura ACFSS
- Estado real de implementación: SPD MSP / Folio
- 07_reglas_estructura_acfss.md
- hA
- R028: Workspace sincroniza documentos Oracle [IMPLEMENTADO parcialmente]
- Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?
- Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?
- N031: Coberturas [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- t
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
- $$
- go_pkg_encoding_xml
- testing.T
- clasificacion.go
- WorkspacePicker.vue
- ingesta.go
- sameObjectionPeriod
- se
- jI
- n
- e
- n
- .push
- .push
- .push
- .push
- standardCodeForFilename
- Vp
- t
- hd
- server/main.go
- he
- coberturas.go
- Va
- uI
- Q: Where should Archivo histórico appear in the Folio sidebar?
- e
- oe
- Manual de operación de Folio
- query_live.cjs
- Ea
- cV
- t
- n
- runtime_config.cjs
- n
- svg_layout.cjs
- $l
- dependencies
- wt
- t
- Ea
- $i
- kS
- Ht
- coverageDateLabel
- Nv
- id
- stagedJob
- bg
- qt
- rt
- Wn
- z_

## God Nodes (most connected - your core abstractions)
1. `_$` - 758 edges
2. `$_` - 757 edges
3. `$$` - 724 edges
4. `_$` - 722 edges
5. `n()` - 133 edges
6. `n()` - 130 edges
7. `t()` - 124 edges
8. `n()` - 124 edges
9. `n()` - 123 edges
10. `t()` - 122 edges

## Surprising Connections (you probably didn't know these)
- `iF` --indirect_call--> `ct()`  [INFERRED]
  dist.pre-sync-deploy-20261007/assets/index--Uh1Xf-Z.js → .dist-before-live-individual-coverage-20261012/assets/index-C51sf43P.js
- `pa()` --indirect_call--> `Py()`  [INFERRED]
  dist.pre-sync-deploy-20261007/assets/index--Uh1Xf-Z.js → .dist-before-live-individual-coverage-20261012/assets/index-C51sf43P.js
- `$$` --indirect_call--> `pT()`  [INFERRED]
  .dist-before-live-historic-archive-20261008/assets/index-0fzV9O3-.js → .dist-before-live-individual-coverage-20261012/assets/index-C51sf43P.js
- `Am` --indirect_call--> `Qv()`  [INFERRED]
  .dist-before-live-individual-coverage-20261012/assets/index-C51sf43P.js → .dist-before-live-historic-archive-20261008/assets/index-0fzV9O3-.js
- `Dl()` --indirect_call--> `yi()`  [INFERRED]
  dist.pre-sync-deploy-20261007/assets/index--Uh1Xf-Z.js → .dist-before-live-historic-archive-20261008/assets/index-0fzV9O3-.js

## Import Cycles
- None detected.

## Communities (105 total, 19 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.01
Nodes (247): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, canDeleteWorkspaces, completionFields (+239 more)

### Community 1 - "packageFolderName"
Cohesion: 0.12
Nodes (30): server, historicalArchiveMonthLabel(), historicalArchiveServiceLabel(), normalizeHistoricalArchiveSearch(), server, server, jobResponse(), missingHeaders() (+22 more)

### Community 2 - "generate_pdf.cjs"
Cohesion: 0.14
Nodes (27): buildFooterImages(), buildSvgPage(), ensureDir(), escapeXml(), estimateRowHeight(), formatCoverageDate(), { formatDateTimeInTimezone }, fs (+19 more)

### Community 3 - "_$"
Cohesion: 0.01
Nodes (247): _$, _0, _1(), a(), a0, a1, a2, Aa (+239 more)

### Community 4 - "_$"
Cohesion: 0.01
Nodes (261): _$, _1(), _3, a0, a1(), a3, aE, aj (+253 more)

### Community 5 - "package.json"
Cohesion: 0.09
Nodes (23): devDependencies, vite, @vitejs/plugin-vue, name, private, scripts, build, dev (+15 more)

### Community 6 - "loadSavedWorkspaces"
Cohesion: 0.09
Nodes (39): addMissingDocuments(), checkSession(), classifyIngest(), clearIngestTramiteMapping(), confirmMergeAllPending(), confirmWorkspaceFusion(), coverageDocumentSequence(), deleteWholeWorkspace() (+31 more)

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
Cohesion: 0.12
Nodes (16): **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**, 1. Stack Tecnológico y Dependencias, **2. Compilar Paquete Final ISO / ZIP**, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, **3. Obtener Metadatos para Rotulado Físico de CD/DVD**, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos (+8 more)

### Community 11 - "Documento de Diseño Técnico"
Cohesion: 0.13
Nodes (14): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs), Decisiones Pendientes, Documento de Diseño Técnico (+6 more)

### Community 12 - "Documento de Diseño Técnico"
Cohesion: 0.10
Nodes (19): 1. Stack Tecnológico y Dependencias, 2. Contexto y Flujo de Datos, 3. Estructura de Directorios Sugerida, 4. Modelos de Datos y Entidades, 5. Lógica de Negocio y Algoritmos, 6. Interfaz de Comunicación (APIs / Métodos Go), **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**, **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)** (+11 more)

### Community 13 - "3. Requisitos Funcionales Generales (RF)"
Cohesion: 0.15
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
Cohesion: 0.08
Nodes (32): addObjectionPatients(), chooseObjectionWorkspace(), closeHistoricalArchiveDelete(), createObjectionWorkspace(), deleteHistoricalArchive(), downloadHistoricalArchive(), downloadSelectedCoverageSheets(), generateCoverageSheets() (+24 more)

### Community 18 - "Reglas OCR, clasificación y checklist ACFSS"
Cohesion: 0.18
Nodes (11): 1. Entrada, extracción y clasificación, 2. Catálogo de señales confirmadas, 3. Catálogo de nombres de archivo conocidos, 4. Matriz de documentos requeridos por servicio, 5. Duplicados y documentos de varias páginas, 6. Banco de pruebas anonimizado, 7. Objeciones y límites de implementación, 8. Añadir, renombrar y reemplazar PDFs desde la bandeja (+3 more)

### Community 19 - "N015: Clasificador híbrido [REQUERIDO; IMPLEMENTADO; NO VERIFICADO E2E]"
Cohesion: 0.18
Nodes (11): N015: Clasificador híbrido [REQUERIDO; IMPLEMENTADO; NO VERIFICADO E2E], N016: reglas_clasificacion.yaml [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N017: Extracción vectorial [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N018: OCR Tesseract [REQUERIDO; IMPLEMENTADO; NO VERIFICADO ambiental; depende del SO], N019: Alertas de fecha [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N020: PENDIENTE_tmp_* [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R017: Clasificador usa YAML [IMPLEMENTADO], R018: Clasificador prioriza vectorial [IMPLEMENTADO] (+3 more)

### Community 20 - "N001: SPD MSP / Folio [REQUERIDO; IMPLEMENTADO; VERIFICADO básica]"
Cohesion: 0.22
Nodes (9): N001: SPD MSP / Folio [REQUERIDO; IMPLEMENTADO; VERIFICADO básica], N002: ACFSS [REQUERIDO; dominio del sistema], N003: Frontend Vue/Vuetify [REQUERIDO; IMPLEMENTADO; VERIFICADO build], N004: Backend Go [REQUERIDO; IMPLEMENTADO; VERIFICADO build + HTTP 200], N005: PLANILLA_DIGITAL [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R001: Folio automatiza ACFSS [REQUERIDO/IMPLEMENTADO; finalidad general parcial], R002: Folio usa Vue/Vuetify [IMPLEMENTADO], R003: Folio usa Go [IMPLEMENTADO] (+1 more)

### Community 21 - "$_"
Cohesion: 0.01
Nodes (260): p0, $_, _2(), _3, a0, a1(), a3, Aa() (+252 more)

### Community 22 - "Reglas funcionales de estructura ACFSS"
Cohesion: 0.25
Nodes (8): 1. Tipos de servicio y paquetes, 2. Identidad y nombres, 3. Primer ingreso, 4. Levantamiento de objeciones, 5. Guardián de cierre, 6. Estado de implementación, Documentos del expediente, Reglas funcionales de estructura ACFSS

### Community 23 - "Estado real de implementación: SPD MSP / Folio"
Cohesion: 0.25
Nodes (7): Clústeres funcionales, Conflictos CFL-01–CFL-08, Criterio de evidencia, Estado real de implementación: SPD MSP / Folio, Mapa global y conclusión, Nodos N001–N043, Relaciones R001–R040

### Community 24 - "07_reglas_estructura_acfss.md"
Cohesion: 0.38
Nodes (3): Desarrollo local, Inicio automático en Linux, SPD MSP

### Community 25 - "hA"
Cohesion: 0.09
Nodes (21): Ar(), cb(), dA(), a(), fV(), hA(), b(), D() (+13 more)

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

### Community 30 - "t"
Cohesion: 0.04
Nodes (79): Ac(), f(), ap(), m(), r(), Bl(), cr(), Cs() (+71 more)

### Community 31 - "N011: Previsualización de ingesta [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]"
Cohesion: 0.50
Nodes (4): N010: ZIP clínico [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N011: Previsualización de ingesta [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R012: Previsualización inspecciona ZIP [IMPLEMENTADO], R013: Previsualización contrasta con Oracle [IMPLEMENTADO]

### Community 32 - "AGENTS.md"
Cohesion: 0.50
Nodes (3): graphify, Product decisions, Runtime build freshness

### Community 33 - "R014: Vínculo manual corrige trámite [IMPLEMENTADO]"
Cohesion: 0.67
Nodes (3): N012: PDI_TRAMITE [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], N014: Vinculación manual trámite [REQUERIDO; IMPLEMENTADO; NO VERIFICADO], R014: Vínculo manual corrige trámite [IMPLEMENTADO]

### Community 34 - "upload"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

### Community 51 - "$$"
Cohesion: 0.01
Nodes (265): $$, _1, aa(), Ac(), aj, ak, an(), As() (+257 more)

### Community 53 - "testing.T"
Cohesion: 0.07
Nodes (65): TestCoverageHasDateUsesDateAssociatedWithSavedPDFs(), TestCoverageMembersAtUsesChosenDateForMinorAndReferences(), TestGenerateIndividualCoverageRejectsInvalidInputsBeforePortal(), TestGenerateIndividualCoverageRequiresSession(), TestMissingCoverageMembersForDateOnlyReturnsMembersWithoutThatDate(), TestNormalizeCoverageDateRequiresRealISODate(), server, historicalArchiveTestServer() (+57 more)

### Community 54 - "clasificacion.go"
Cohesion: 0.07
Nodes (48): addDateCandidateForPeriod(), classifyPDF(), copyWithLimit(), datesOutsideBilledPeriod(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules(), matchClassification() (+40 more)

### Community 55 - "WorkspacePicker.vue"
Cohesion: 0.07
Nodes (45): coverageMonthLabel(), coverageServiceLabel(), filteredHistoricalArchives, filteredObjectionPatients, filteredSavedWorkspaces, objectionSourceItems, objectionWorkspaceItems, orderedSavedWorkspaces (+37 more)

### Community 56 - "ingesta.go"
Cohesion: 0.07
Nodes (42): newHistoricalArchiveID(), backupAndReplace(), newUploadJobID(), periodWorkspaceID(), sha256Upload(), readMultipartBytes(), archiveRelativePatientPath(), compareArchivePatientTieBreak() (+34 more)

### Community 57 - "sameObjectionPeriod"
Cohesion: 0.67
Nodes (3): objectionPeriodSpaces, objectionSourcePeriodSpaces, sameObjectionPeriod()

### Community 58 - "se"
Cohesion: 0.05
Nodes (156): _2(), Ac(), ad(), i(), l(), ah(), b(), h() (+148 more)

### Community 59 - "jI"
Cohesion: 0.05
Nodes (126): ae(), Ar(), cb(), V(), Cm(), D2(), C(), D() (+118 more)

### Community 60 - "n"
Cohesion: 0.03
Nodes (141): _2(), a(), i(), l(), ab(), aD(), a(), i() (+133 more)

### Community 61 - "e"
Cohesion: 0.04
Nodes (92): b0, n(), c0(), c2(), f(), g(), d0(), Eb() (+84 more)

### Community 62 - "n"
Cohesion: 0.05
Nodes (131): Ac, aD, n(), s(), Bf(), b(), d(), f() (+123 more)

### Community 63 - ".push"
Cohesion: 0.03
Nodes (126): _2(), a(), ad(), ag(), al(), ao, aw(), bo (+118 more)

### Community 64 - ".push"
Cohesion: 0.03
Nodes (135): Bd(), k(), P(), w(), br, bS(), bw(), _c() (+127 more)

### Community 65 - ".push"
Cohesion: 0.03
Nodes (119): bE(), bo, bv, Ce(), Cm(), ct(), cu(), db() (+111 more)

### Community 66 - ".push"
Cohesion: 0.03
Nodes (153): _0, Af, ak(), An, Ao, Ay(), b0, bE() (+145 more)

### Community 67 - "standardCodeForFilename"
Cohesion: 0.33
Nodes (6): addWorkspacePDFs(), openWorkspaceFusion(), queuedWorkspaceDuplicates, reviewPendingFusion(), selectWorkspacePDFs(), standardCodeForFilename()

### Community 68 - "Vp"
Cohesion: 0.08
Nodes (89): Ap(), bm(), co(), cS(), cy(), ef(), V(), Eo() (+81 more)

### Community 69 - "t"
Cohesion: 0.05
Nodes (124): _0, a0, a1(), b2, Bc(), bD, Bf(), Bl() (+116 more)

### Community 70 - "hd"
Cohesion: 0.06
Nodes (33): bP(), Bt(), cv(), Er(), fI(), fP(), Fu(), Gp() (+25 more)

### Community 71 - "server/main.go"
Cohesion: 0.09
Nodes (31): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+23 more)

### Community 72 - "he"
Cohesion: 0.04
Nodes (101): au(), cf, Af(), am(), Bk, bm, bo(), Bv() (+93 more)

### Community 73 - "coberturas.go"
Cohesion: 0.12
Nodes (31): extractPDFText(), normalizeOCRText(), containsInt64(), coverageDocumentDate(), coverageDocumentMetadataItems(), coverageHasDate(), coverageMemberCedulas(), coverageMembers() (+23 more)

### Community 74 - "Va"
Cohesion: 0.04
Nodes (42): a(), a2(), Aa(), Ba, Br(), cA(), eg(), fA() (+34 more)

### Community 75 - "uI"
Cohesion: 0.06
Nodes (100): Fo(), Mo, Lo(), al, Am, bA(), x(), Dm() (+92 more)

### Community 76 - "Q: Where should Archivo histórico appear in the Folio sidebar?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Where should Archivo histórico appear in the Folio sidebar?, Source Nodes

### Community 77 - "e"
Cohesion: 0.04
Nodes (98): ae(), At(), h(), bl(), c1, D(), w(), x() (+90 more)

### Community 78 - "oe"
Cohesion: 0.06
Nodes (129): al(), b2, Ba(), M(), bf, bt(), Ca(), Co (+121 more)

### Community 79 - "Manual de operación de Folio"
Cohesion: 0.15
Nodes (12): Acceso por HTTP y alcance de red, Arrancar Folio, Arranque después de apagar o reiniciar el servidor, Datos y respaldos, Detener Folio, Instalación actual, Lista breve de operación, Manual de operación de Folio (+4 more)

### Community 80 - "query_live.cjs"
Cohesion: 0.16
Nodes (18): ref_crypto, crypto, CryptoJS, csconsulta(), extractCookie(), fs, { getHttpsAgent, loadEnvFile }, https (+10 more)

### Community 81 - "Ea"
Cohesion: 0.04
Nodes (48): ab(), aV(), b_, cV, cw(), Dr(), dw(), Ea() (+40 more)

### Community 82 - "cV"
Cohesion: 0.04
Nodes (43): as(), aV(), o(), cu(), cV(), eV(), fo(), fs (+35 more)

### Community 83 - "t"
Cohesion: 0.04
Nodes (102): ag, Am(), aS, av, B_, Bc(), bd(), Bf() (+94 more)

### Community 84 - "n"
Cohesion: 0.05
Nodes (75): a2, b1(), bu(), cD(), a(), c(), i(), l() (+67 more)

### Community 85 - "runtime_config.cjs"
Cohesion: 0.22
Nodes (13): ref_https, postText(), formatDateTimeInTimezone(), formatTimestampSlugInTimezone(), fs, getConfiguredTimezone(), getDateTimeParts(), getHttpsAgent() (+5 more)

### Community 86 - "n"
Cohesion: 0.04
Nodes (49): a(), bv(), bw(), by(), CP(), f0(), P(), Fn() (+41 more)

### Community 87 - "svg_layout.cjs"
Cohesion: 0.22
Nodes (12): ref_fs, ref_path, applyChromeToAllPages(), drawFooter(), drawHeader(), fs, loadSvgLayout(), parseHeaderLineY() (+4 more)

### Community 88 - "$l"
Cohesion: 0.05
Nodes (60): af(), Ai, am(), bI(), r(), Bt(), by(), dl() (+52 more)

### Community 89 - "dependencies"
Cohesion: 0.25
Nodes (8): dependencies, crypto-js, idb, @mdi/font, pdfkit, svg-to-pdfkit, vue, vuetify

### Community 90 - "wt"
Cohesion: 0.08
Nodes (24): ab(), cw(), db(), j1(), i(), kE, kg(), d() (+16 more)

### Community 91 - "t"
Cohesion: 0.05
Nodes (101): _1(), a2(), aI(), an(), Au, B_, b0(), Bn (+93 more)

### Community 92 - "Ea"
Cohesion: 0.04
Nodes (48): a(), ag(), S(), Bu(), cA(), Cy(), o(), dd() (+40 more)

### Community 93 - "$i"
Cohesion: 0.08
Nodes (21): _0, CP(), dw(), Dy(), eS(), fn, Gn, Gp() (+13 more)

### Community 94 - "kS"
Cohesion: 0.09
Nodes (24): Af(), aS(), Ay(), bd(), BS(), fd(), fv(), gS() (+16 more)

### Community 95 - "Ht"
Cohesion: 0.09
Nodes (25): by(), c0(), cd(), D(), P(), V(), Eu(), H1 (+17 more)

### Community 97 - "Nv"
Cohesion: 0.11
Nodes (15): S(), by(), Gr(), r(), hy(), jn, Nv(), b() (+7 more)

### Community 98 - "id"
Cohesion: 0.11
Nodes (18): Do, a(), gb(), gT(), S(), T(), id(), c() (+10 more)

### Community 99 - "stagedJob"
Cohesion: 0.09
Nodes (53): atomicWritePrivateFile(), headerOutputFilename(), matrixFilename(), buildSelectedObjectionRows(), cleanupObjectionRows(), copyObjectionPDFTree(), installObjectionRows(), nextSafeAnnexName() (+45 more)

### Community 100 - "bg"
Cohesion: 0.16
Nodes (14): b1(), bg(), V(), w(), y(), e2(), ED, f2() (+6 more)

### Community 101 - "qt"
Cohesion: 0.15
Nodes (17): ay(), bI(), E(), k(), M(), fx(), ks(), oI() (+9 more)

### Community 102 - "rt"
Cohesion: 0.15
Nodes (15): ak(), P(), b(), i0(), ig(), it(), jd(), Le() (+7 more)

### Community 103 - "Wn"
Cohesion: 0.21
Nodes (9): S(), j2(), jb(), y(), Kr(), mb(), qm(), vw() (+1 more)

## Knowledge Gaps
- **1047 isolated node(s):** `tf`, `dc`, `zp`, `fi`, `gu` (+1042 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1527 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **19 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `$_` connect `$_` to `.push`, `Vp`, `t`, `bg`, `Wn`, `z_`, `uI`, `e`, `$i`, `$l`, `kS`, `se`, `t`, `Ea`, `e`, `n`, `Ht`?**
  _High betweenness centrality (0.165) - this node is a cross-community bridge._
- **Why does `$$` connect `$$` to `id`, `Vp`, `t`, `he`, `t`, `uI`, `Ea`, `n`, `$l`, `hA`, `se`, `jI`, `e`, `.push`?**
  _High betweenness centrality (0.158) - this node is a cross-community bridge._
- **Why does `_$` connect `_$` to `.push`, `.push`, `t`, `wt`, `hd`, `Va`, `uI`, `$$`, `t`, `$_`, `n`, `se`, `jI`, `e`, `n`, `.push`?**
  _High betweenness centrality (0.151) - this node is a cross-community bridge._
- **Are the 32 inferred relationships involving `_$` (e.g. with `cm()` and `dm()`) actually correct?**
  _`_$` has 32 INFERRED edges - model-reasoned connections that need verification._
- **Are the 25 inferred relationships involving `$_` (e.g. with `GE()` and `JE()`) actually correct?**
  _`$_` has 25 INFERRED edges - model-reasoned connections that need verification._
- **Are the 28 inferred relationships involving `$$` (e.g. with `aa()` and `f1()`) actually correct?**
  _`$$` has 28 INFERRED edges - model-reasoned connections that need verification._
- **Are the 24 inferred relationships involving `_$` (e.g. with `aa()` and `au()`) actually correct?**
  _`_$` has 24 INFERRED edges - model-reasoned connections that need verification._