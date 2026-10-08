# Graph Report - spd_msp  (2026-10-08)

## Corpus Check
- 49 files · ~108,634 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 12 file(s) not represented in the graph (top: (none) 2, .zip 2, .css 2)

## Summary
- 2251 nodes · 7314 edges · 122 communities (96 shown, 26 thin omitted)
- Extraction: 82% EXTRACTED · 18% INFERRED · 0% AMBIGUOUS · INFERRED: 1312 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `81c69f02`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue
- packageFolderName
- generate_pdf.cjs
- _$
- Mt
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
- context.Context
- R028: Workspace sincroniza documentos Oracle [IMPLEMENTADO parcialmente]
- Q: Why couldn't the user delete saved period WORK-AMBULATORIO-202609?
- Q: Does Oracle-only planilla absence block processing when ZIP is the source of truth?
- N031: Coberturas [REQUERIDO; IMPLEMENTADO; NO VERIFICADO]
- e
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
- Vl
- go_pkg_encoding_xml
- testing.T
- classifyPDF
- WorkspacePicker.vue
- ingesta.go
- sameObjectionPeriod
- get
- i
- a
- qe
- Ml
- t
- rt
- El
- I_
- Bd
- n
- oe
- Cs
- d
- he
- coberturas.go
- .push
- um
- server
- workspaceSavedOption
- $l
- Manual de operación de Folio
- query_live.cjs
- Se
- cV
- jv
- we
- runtime_config.cjs
- bt
- svg_layout.cjs
- l
- dependencies
- pw
- ep
- b
- m_
- b_
- j
- coverageDateLabel
- uo
- Jb
- stagedJob
- Yt
- Xt
- Ud
- yI
- fg
- Vc
- jx
- iD
- te
- Gc
- fm
- Gv
- Lu
- qf
- Kd
- Vx
- copyWithLimit
- Gl
- hk
- Bh
- Db
- Ru

## God Nodes (most connected - your core abstractions)
1. `_$` - 720 edges
2. `n()` - 133 edges
3. `t()` - 122 edges
4. `he()` - 105 edges
5. `a()` - 104 edges
6. `oe()` - 99 edges
7. `Me()` - 92 edges
8. `l()` - 84 edges
9. `e()` - 72 edges
10. `i()` - 71 edges

## Surprising Connections (you probably didn't know these)
- `validateCoveragePDF()` --calls--> `extractPDFText()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/clasificacion.go
- `validateCoveragePDF()` --calls--> `normalizeOCRText()`  [INFERRED]
  cmd/server/coberturas.go → cmd/server/clasificacion.go
- `pendingPDFName()` --calls--> `safeOriginalFilename()`  [INFERRED]
  cmd/server/clasificacion.go → cmd/server/ingesta.go
- `TestNormalizeCoverageDateRequiresRealISODate()` --calls--> `normalizeCoverageDate()`  [INFERRED]
  cmd/server/coberturas_test.go → cmd/server/coberturas.go
- `TestCoverageMembersAtUsesChosenDateForMinorAndReferences()` --calls--> `coverageMembersAt()`  [INFERRED]
  cmd/server/coberturas_test.go → cmd/server/coberturas.go

## Import Cycles
- None detected.

## Communities (122 total, 26 thin omitted)

### Community 0 - "App.vue"
Cohesion: 0.01
Nodes (215): catalogos_codigos_msp, activeDoc, activeFolder, activePage, authError, authStatus, canDeleteWorkspaces, completionFields (+207 more)

### Community 1 - "packageFolderName"
Cohesion: 0.14
Nodes (29): copyPrivateFile(), server, server, jobResponse(), missingHeaders(), packageFolderName(), safeOriginalFilename(), saveUploadPart() (+21 more)

### Community 2 - "generate_pdf.cjs"
Cohesion: 0.14
Nodes (27): buildFooterImages(), buildSvgPage(), ensureDir(), escapeXml(), estimateRowHeight(), formatCoverageDate(), { formatDateTimeInTimezone }, fs (+19 more)

### Community 3 - "_$"
Cohesion: 0.01
Nodes (139): _$, _0, a1, a2, Aa, ag, aT, Au (+131 more)

### Community 4 - "Mt"
Cohesion: 0.17
Nodes (11): cb(), Fd, Gi, u(), gy(), Mt(), ok(), ow() (+3 more)

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
Cohesion: 0.15
Nodes (19): addObjectionPatients(), chooseObjectionWorkspace(), createObjectionWorkspace(), downloadSelectedCoverageSheets(), loadObjectionPDFSelection(), loadObjectionWorkspace(), loadPlanilla(), objectionFormData() (+11 more)

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
Cohesion: 0.33
Nodes (6): addWorkspacePDFs(), openWorkspaceFusion(), queuedWorkspaceDuplicates, reviewPendingFusion(), selectWorkspacePDFs(), standardCodeForFilename()

### Community 22 - "Reglas funcionales de estructura ACFSS"
Cohesion: 0.25
Nodes (8): 1. Tipos de servicio y paquetes, 2. Identidad y nombres, 3. Primer ingreso, 4. Levantamiento de objeciones, 5. Guardián de cierre, 6. Estado de implementación, Documentos del expediente, Reglas funcionales de estructura ACFSS

### Community 23 - "Estado real de implementación: SPD MSP / Folio"
Cohesion: 0.25
Nodes (7): Clústeres funcionales, Conflictos CFL-01–CFL-08, Criterio de evidencia, Estado real de implementación: SPD MSP / Folio, Mapa global y conclusión, Nodos N001–N043, Relaciones R001–R040

### Community 24 - "07_reglas_estructura_acfss.md"
Cohesion: 0.38
Nodes (3): Desarrollo local, Inicio automático en Linux, SPD MSP

### Community 25 - "context.Context"
Cohesion: 0.09
Nodes (27): parseOracleDate(), oracleTableName(), canonicalPath(), fileSHA256(), server, oracleQualified(), collectZIPPlanillaNumbers(), finalizeNewPlanilla() (+19 more)

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

### Community 30 - "e"
Cohesion: 0.14
Nodes (30): an(), ao, b(), d(), f(), g(), p(), S() (+22 more)

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

### Community 51 - "Vl"
Cohesion: 0.09
Nodes (30): a0, _c(), $d, Ec(), fp, Hn, ii(), k_() (+22 more)

### Community 53 - "testing.T"
Cohesion: 0.10
Nodes (44): TestCoverageHasDateUsesDateAssociatedWithSavedPDFs(), TestCoverageMembersAtUsesChosenDateForMinorAndReferences(), TestMissingCoverageMembersForDateOnlyReturnsMembersWithoutThatDate(), TestNormalizeCoverageDateRequiresRealISODate(), TestCurrentSessionExposesDeleteWorkspacePermission(), nextSafeAnnexName(), normalizeAnnexFilename(), objectionAnnexBase() (+36 more)

### Community 54 - "classifyPDF"
Cohesion: 0.18
Nodes (18): addDateCandidateForPeriod(), classifyPDF(), datesOutsideBilledPeriod(), extractPDFText(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules(), matchClassification() (+10 more)

### Community 55 - "WorkspacePicker.vue"
Cohesion: 0.10
Nodes (32): filteredObjectionPatients, orderedSavedWorkspaces, savedWorkspaceYearFilters, displayedModelValue, emit, filteredWorkspaces, hasFilters, orderedWorkspaces (+24 more)

### Community 56 - "ingesta.go"
Cohesion: 0.07
Nodes (65): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+57 more)

### Community 57 - "sameObjectionPeriod"
Cohesion: 0.67
Nodes (3): objectionPeriodSpaces, objectionSourcePeriodSpaces, sameObjectionPeriod()

### Community 58 - "get"
Cohesion: 0.12
Nodes (13): c2, i(), r(), eV(), get(), _h(), jt, nn() (+5 more)

### Community 59 - "i"
Cohesion: 0.16
Nodes (20): cF, Ea, h2, mo(), g(), m(), pL, i() (+12 more)

### Community 60 - "a"
Cohesion: 0.13
Nodes (19): _2(), a(), i(), l(), aD(), a(), i(), l() (+11 more)

### Community 61 - "qe"
Cohesion: 0.25
Nodes (11): P(), w(), dm(), has(), Hd, lw, Pu(), qe() (+3 more)

### Community 62 - "Ml"
Cohesion: 0.19
Nodes (16): _1(), Ex(), Gu(), Kb(), kh(), Ml(), Mx(), Q() (+8 more)

### Community 63 - "t"
Cohesion: 0.15
Nodes (15): cI(), dk(), i(), o(), t(), io, Jo(), ll() (+7 more)

### Community 64 - "rt"
Cohesion: 0.24
Nodes (10): ak(), b1(), b(), eD, jd(), nk(), rt(), xE (+2 more)

### Community 65 - "El"
Cohesion: 0.23
Nodes (8): cp, El(), Ix, Kn(), Ld(), Oa(), px(), wc()

### Community 66 - "I_"
Cohesion: 0.17
Nodes (8): I_(), r(), u(), Ne(), Qa(), r(), rg(), Sc()

### Community 67 - "Bd"
Cohesion: 0.15
Nodes (9): Bd(), k(), by(), cD(), Gr(), jn, b(), S() (+1 more)

### Community 68 - "n"
Cohesion: 0.07
Nodes (34): aV(), o(), be(), cg, eo(), n(), n(), gS() (+26 more)

### Community 69 - "oe"
Cohesion: 0.10
Nodes (55): m(), V(), Ba(), bf, bo(), Ca(), Da(), y() (+47 more)

### Community 70 - "Cs"
Cohesion: 0.07
Nodes (48): Ac(), Bl(), Cn(), Cs(), cw(), Dc, dp, dr() (+40 more)

### Community 71 - "d"
Cohesion: 0.10
Nodes (43): I(), l(), b2, bI(), c1, I(), S(), er() (+35 more)

### Community 72 - "he"
Cohesion: 0.06
Nodes (46): Af(), am(), Bk, bm, Bv(), cl, Co, cy() (+38 more)

### Community 73 - "coberturas.go"
Cohesion: 0.13
Nodes (28): containsInt64(), coverageDocumentDate(), coverageDocumentMetadataItems(), coverageHasDate(), coverageMemberCedulas(), coverageMembers(), coverageMembersAt(), filterCoverageMembers() (+20 more)

### Community 74 - ".push"
Cohesion: 0.06
Nodes (43): ab(), cm(), Ct(), w(), eb(), es(), Ff(), a() (+35 more)

### Community 75 - "um"
Cohesion: 0.33
Nodes (5): bp, um(), a(), i(), l()

### Community 76 - "server"
Cohesion: 0.17
Nodes (12): envInt64(), envOr(), server, main(), newSessionID(), oracleCredentialError(), oracleHasRole(), securityHeaders() (+4 more)

### Community 77 - "workspaceSavedOption"
Cohesion: 0.14
Nodes (17): coverageMonthLabel(), coverageServiceLabel(), filteredSavedWorkspaces, generateCoverageSheets(), generateCoverageSheetsAtChosenDate(), generateCoverageSheetsForDate(), loadCoveragePlanillas(), objectionSourceItems (+9 more)

### Community 78 - "$l"
Cohesion: 0.15
Nodes (57): dO, e_(), C(), D(), P(), T(), V(), fe() (+49 more)

### Community 79 - "Manual de operación de Folio"
Cohesion: 0.16
Nodes (12): Acceso por HTTP y alcance de red, Arrancar Folio, Arranque después de apagar o reiniciar el servidor, Datos y respaldos, Detener Folio, Instalación actual, Lista breve de operación, Manual de operación de Folio (+4 more)

### Community 80 - "query_live.cjs"
Cohesion: 0.16
Nodes (18): ref_crypto, crypto, CryptoJS, csconsulta(), extractCookie(), fs, { getHttpsAgent, loadEnvFile }, https (+10 more)

### Community 81 - "Se"
Cohesion: 0.18
Nodes (14): dn, dw, D(), P(), V(), fw(), gA(), D() (+6 more)

### Community 82 - "cV"
Cohesion: 0.06
Nodes (19): as(), cV(), gm(), Hl(), jm(), S(), mV(), tV() (+11 more)

### Community 83 - "jv"
Cohesion: 0.20
Nodes (5): jv, np(), rp(), ux(), Vh()

### Community 84 - "we"
Cohesion: 0.12
Nodes (17): E(), k(), M(), ek(), s(), u(), mA, mD() (+9 more)

### Community 85 - "runtime_config.cjs"
Cohesion: 0.22
Nodes (13): ref_https, postText(), formatDateTimeInTimezone(), formatTimestampSlugInTimezone(), fs, getConfiguredTimezone(), getDateTimeParts(), getHttpsAgent() (+5 more)

### Community 86 - "bt"
Cohesion: 0.12
Nodes (16): S(), M(), bt(), ob(), Pk(), C(), D(), I() (+8 more)

### Community 87 - "svg_layout.cjs"
Cohesion: 0.22
Nodes (12): ref_fs, ref_path, applyChromeToAllPages(), drawFooter(), drawHeader(), fs, loadSvgLayout(), parseHeaderLineY() (+4 more)

### Community 88 - "l"
Cohesion: 0.29
Nodes (10): ce(), Hf, it(), kf(), a(), l(), Le(), l() (+2 more)

### Community 89 - "dependencies"
Cohesion: 0.25
Nodes (8): dependencies, crypto-js, idb, @mdi/font, pdfkit, svg-to-pdfkit, vue, vuetify

### Community 90 - "pw"
Cohesion: 0.43
Nodes (7): bw(), gw(), hw(), nA, pw(), sw, yw()

### Community 91 - "ep"
Cohesion: 0.25
Nodes (6): ap(), m(), r(), ep, ip(), yf()

### Community 92 - "b"
Cohesion: 0.12
Nodes (21): cr(), Df(), dV(), l(), f1, Hr(), Mf(), Nv() (+13 more)

### Community 93 - "m_"
Cohesion: 0.10
Nodes (24): a(), dy(), fo(), fy(), a(), gp(), hp(), hS() (+16 more)

### Community 94 - "b_"
Cohesion: 0.33
Nodes (6): b_(), k1(), Lb(), ts(), w1(), xf()

### Community 95 - "j"
Cohesion: 0.22
Nodes (10): en(), o(), j(), ka(), lr(), Qm, sp, tw() (+2 more)

### Community 97 - "uo"
Cohesion: 0.15
Nodes (10): al(), Dl(), ho, Nh(), ro, sl(), uo, S() (+2 more)

### Community 99 - "stagedJob"
Cohesion: 0.07
Nodes (57): atomicWritePrivateFile(), headerOutputFilename(), matrixFilename(), writePrivateJSON(), buildSelectedObjectionRows(), cleanupObjectionRows(), copyObjectionPDFTree(), installObjectionRows() (+49 more)

### Community 100 - "Yt"
Cohesion: 0.33
Nodes (9): br, bS(), k2, lo, no(), pD(), rw, sa() (+1 more)

### Community 101 - "Xt"
Cohesion: 0.28
Nodes (8): Ih(), mn(), a(), S(), sD, Xt(), zd(), zw

### Community 102 - "Ud"
Cohesion: 0.22
Nodes (7): fx(), tI(), Ud(), f(), o(), s(), u()

### Community 103 - "yI"
Cohesion: 0.36
Nodes (8): aI(), ey(), oI(), qb(), qt(), sI(), ty(), yI()

### Community 104 - "fg"
Cohesion: 0.33
Nodes (5): b0(), dg(), fg(), h0, w0()

### Community 105 - "Vc"
Cohesion: 0.33
Nodes (6): ck(), cu(), gT(), kT, Vc(), Zu()

### Community 107 - "iD"
Cohesion: 0.40
Nodes (5): iD, d(), g(), s(), u()

### Community 109 - "Gc"
Cohesion: 0.50
Nodes (3): Gc(), i(), l()

### Community 110 - "fm"
Cohesion: 0.50
Nodes (4): fm, op, Tp(), vm

### Community 111 - "Gv"
Cohesion: 0.83
Nodes (4): Gv(), ks(), qx(), Wv()

### Community 112 - "Lu"
Cohesion: 0.50
Nodes (4): hy(), Lu(), sr, vr()

### Community 113 - "qf"
Cohesion: 0.50
Nodes (4): jf(), Ky(), qf(), Wu()

### Community 114 - "Kd"
Cohesion: 0.50
Nodes (4): Kd(), i(), l(), o()

### Community 116 - "copyWithLimit"
Cohesion: 0.67
Nodes (3): copyWithLimit(), io.Reader, io.Writer

### Community 118 - "hk"
Cohesion: 0.67
Nodes (3): hk, jk(), Ps()

## Knowledge Gaps
- **569 isolated node(s):** `coverageFailureItem`, `coverageGenerateRequest`, `loginRequest`, `objectionPackagePDF`, `oracleDocument` (+564 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 743 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `_$` connect `_$` to `Mt`, `e`, `Vl`, `get`, `i`, `a`, `qe`, `Ml`, `t`, `rt`, `El`, `I_`, `Bd`, `n`, `oe`, `Cs`, `d`, `he`, `.push`, `um`, `$l`, `Se`, `cV`, `jv`, `we`, `bt`, `l`, `pw`, `ep`, `b`, `m_`, `b_`, `j`, `uo`, `Jb`, `Yt`, `Xt`, `Ud`, `yI`, `fg`, `Vc`, `jx`, `iD`, `te`, `Gc`, `fm`, `Gv`, `Lu`, `qf`, `Kd`, `Vx`, `Gl`, `hk`, `Bh`, `Db`, `Ru`?**
  _High betweenness centrality (0.215) - this node is a cross-community bridge._
- **Why does `Pk()` connect `bt` to `Bd`, `_$`, `oe`, `d`, `$l`, `Se`, `b`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Why does `vue` connect `package.json` to `App.vue`, `WorkspacePicker.vue`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Are the 22 inferred relationships involving `_$` (e.g. with `_1()` and `aE()`) actually correct?**
  _`_$` has 22 INFERRED edges - model-reasoned connections that need verification._
- **Are the 76 inferred relationships involving `n()` (e.g. with `a()` and `aD()`) actually correct?**
  _`n()` has 76 INFERRED edges - model-reasoned connections that need verification._
- **Are the 82 inferred relationships involving `t()` (e.g. with `aD()` and `m()`) actually correct?**
  _`t()` has 82 INFERRED edges - model-reasoned connections that need verification._
- **Are the 56 inferred relationships involving `a()` (e.g. with `_2()` and `n()`) actually correct?**
  _`a()` has 56 INFERRED edges - model-reasoned connections that need verification._