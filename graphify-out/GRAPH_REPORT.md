# Graph Report - spd_msp  (2026-11-18)

## Corpus Check
- 80 files · ~119,255 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 7 file(s) not represented in the graph (top: (none) 3, .example 1, .zip 1)

## Summary
- 1359 nodes · 3376 edges · 61 communities (51 shown, 10 thin omitted)
- Extraction: 77% EXTRACTED · 23% INFERRED · 0% AMBIGUOUS · INFERRED: 784 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- App.vue Main Component State
- PDF Ingest Handlers
- PDF Generation Scripts
- Objection Workspace UI
- Saved Workspace Picker
- Workspace PDF Renaming
- NPM Dependencies
- Workspace PDF Actions UI
- Objection Rows Builder
- Folio PDF Review Skill
- PDF Review Script
- Demo Batch Generator
- Coverage Logic
- Product Principles
- Objection Sync Tests
- PDF Classifier
- Coverage Tests
- Objection Sync Copy
- Operations Manual & Queries
- Ingest File Utilities
- Planilla Provisioning
- Deploy Skill
- Coverage & Archive Filters UI
- Oracle Document Sync
- Oracle Requirements
- Objection & Closeout Requirements
- Hybrid Classification Requirements
- Oracle Date Parsing
- Classification Rules Config
- Dual Ingest & Objections Reqs
- Hybrid OCR Classifier
- Ingest Frontend Loaders
- Historical Archive
- Historical Archive Tests
- Package Structure Rules
- Auditor ZIP Import
- Merge Limit Decisions
- Institutional Logos
- Oracle Schema & Repository
- Real Implementation Status
- Objection Annex Naming
- Classification Confidence Rules
- MSP Catalog Decisions
- Implementation Status Queries
- Dual Ingest Frontend
- Workspace ZIP Download
- Classification Rule Keys
- OpenAI Skill Config
- Coverage Bot Overview
- Objection Period Spaces
- Upload Dropzone
- SPA Entry Point
- Tech Stack & Vision
- Runtime Deploy Installer
- Service Installer
- Coverage Labels
- Fusion Drag Handlers
- Auditor Folder Planilla
- Go Module

## God Nodes (most connected - your core abstractions)
1. `server` - 105 edges
2. `stagedJob` - 87 edges
3. `packageFolderName()` - 53 edges
4. `writeError()` - 49 edges
5. `main()` - 48 edges
6. `writeJSON()` - 42 edges
7. `signOut()` - 31 edges
8. `validJobID()` - 28 edges
9. `Principios de producto de Folio` - 27 edges
10. `loadSavedWorkspaces()` - 25 edges

## Surprising Connections (you probably didn't know these)
- `Decision: classify by document header` --semantically_similar_to--> `Classification precedence: filename alias, header phrases, priority phrases, regular rules`  [INFERRED] [semantically similar]
  PRINCIPIOS_DE_PRODUCTO.md → .agents/skills/folio-pdf-review/SKILL.md
- `Replacing executable or restarting service is a deployment requiring explicit authorization` --semantically_similar_to--> `Deploy only with explicit user authorization`  [INFERRED] [semantically similar]
  AGENTS.md → .agents/skills/desplegar-spd/SKILL.md
- `Tie broken by longest specific phrase (conflicts with current rule: equal counts mean pending)` --references--> `Classification precedence: filename alias, header phrases, priority phrases, regular rules`  [AMBIGUOUS]
  graphify-out/memory/query_20261023_202221_a0eeb6ab_que_nombre_le_deberias_dar_a__home_spd_he1_recurso.md → .agents/skills/folio-pdf-review/SKILL.md
- `OCR/classification pipeline (pdftoppm, Tesseract spa, PENDIENTE_tmp_)` --semantically_similar_to--> `OCR fallback: Tesseract Spanish on first page only when no text`  [INFERRED] [semantically similar]
  README.md → .agents/skills/folio-pdf-review/SKILL.md
- `folio.service systemd unit` --references--> `Compare active service process executable, bin/folio-server, dist/ and HTTP frontend with fresh build`  [INFERRED]
  MANUAL_OPERACION_FOLIO.md → .agents/skills/runtime-freshness/SKILL.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **PDF classification by header, rules and OCR** — _agents_skills_folio_pdf_review_skill_clasificacion_go, reglas_clasificacion, _agents_skills_folio_pdf_review_skill_ocr_fallback, _agents_skills_folio_pdf_review_skill_header_first_precedence, principios_de_producto_clasificar_encabezado [INFERRED 0.85]
- **Runtime freshness and deploy policy** — agents, _agents_skills_runtime_freshness_skill, _agents_skills_desplegar_spd_skill, _agents_skills_runtime_freshness_skill_verify_runtime_helper, _agents_skills_desplegar_spd_skill_deploy_runtime_helper [EXTRACTED 1.00]
- **HCU_010 merge limit evolution 50 to 1000** — graphify_out_memory_query_20261023_190126_a94b640d_no_sera_porque_son_muchos_merge_limit_50, graphify_out_memory_query_20261023_190553_5d173669_pongamos_el_limite_a_100_por_grupo_merge_limit_100, graphify_out_memory_query_20261023_194431_69b11d43_yerovi_tampoco_fusiona__tiene_277_pdfs_en_hcu_010_merge_limit_300, graphify_out_memory_query_20261023_194716_9eedc058_yerovi_tampoco_fusiona__tiene_277_pdfs_en_hcu_010_merge_limit_1000, principios_de_producto_fusion_1000 [EXTRACTED 1.00]
- **Pipeline de clasificacion documental (vectorial, OCR, YAML, pendiente, fusion)** — requerimientos_04ocr_go_clasificador_hibrido, requerimientos_04ocr_go_ocr_tesseract_spa, requerimientos_04ocr_go_reglas_yaml, requerimientos_04ocr_go_pendiente_tmp, requerimientos_04ocr_go_fusion_pdfcpu_merge [INFERRED 0.85]
- **Componentes del Guardian de cierre** — requerimientos_06req_motor_cierre_auditoria_preflight, requerimientos_06req_motor_cierre_auditoria_tecnica_pdf_dpi, requerimientos_06req_motor_cierre_compilador_iso_zip, requerimientos_06req_motor_cierre_rotulado_cd, requerimientos_06req_motor_cierre_cierre_lote_oracle [EXTRACTED 1.00]
- **Frontera pendiente del cierre formal (guardian, checklist, estado)** — requerimientos_09_estado_real_implementacion_guardian_no_implementado, requerimientos_09_estado_real_implementacion_checklist_no_implementado, requerimientos_09_estado_real_implementacion_estado_digitalizacion_no_integrado [EXTRACTED 1.00]
- **Coverage institution logos (IESS, ISSFA, ISSPOL, MSP, RPIS)** — assets_cobertura_iess, assets_cobertura_issfa, assets_cobertura_isspol, assets_cobertura_logomsp, assets_cobertura_logorpis [INFERRED 0.85]

## Communities (61 total, 10 thin omitted)

### Community 0 - "App.vue Main Component State"
Cohesion: 0.01
Nodes (295): activeDoc, activeFolder, activePage, auditorImportAllFilesMapped, auditorImportBusy, auditorImportError, auditorImportFile, auditorImportFolderDefaults (+287 more)

### Community 1 - "PDF Ingest Handlers"
Cohesion: 0.10
Nodes (42): extractPDFText(), validateCoveragePDF(), copyPrivateFile(), jobResponse(), missingHeaders(), packageFolderName(), periodWorkspaceID(), safeOriginalFilename() (+34 more)

### Community 2 - "PDF Generation Scripts"
Cohesion: 0.06
Nodes (62): applyChromeToAllPages(), buildFooterImages(), buildSvgPage(), drawFooter(), drawHeader(), ensureDir(), escapeXml(), estimateRowHeight() (+54 more)

### Community 3 - "Objection Workspace UI"
Cohesion: 0.08
Nodes (40): addObjectionPatients(), applyAuditorFolderImport(), chooseObjectionWorkspace(), clearAuditorImportPreview(), closeHistoricalArchiveDelete(), createObjectionWorkspace(), createOrOpenAuditorPeriod(), deleteHistoricalArchive() (+32 more)

### Community 4 - "Saved Workspace Picker"
Cohesion: 0.09
Nodes (34): filteredSavedWorkspaces, orderedSavedWorkspaces, savedWorkspaceStatusFilters, savedWorkspaceYearFilters, workspaceReceivedAt(), workspaceStatusLabel(), displayedModelValue, emit (+26 more)

### Community 5 - "Workspace PDF Renaming"
Cohesion: 0.10
Nodes (30): applyWorkspacePDFFileRenames(), applyWorkspacePDFRenameMetadata(), archiveRelativePatientPath(), assignWorkspaceNameTargets(), canonicalMSPPDFCode(), compareArchivePatientTieBreak(), TestWorkspaceCoverageFusionPreservesDateAndCedulaMetadata(), TestWorkspaceFusionGroupsIncludeCoverageAndSeparatePlanillas() (+22 more)

### Community 6 - "NPM Dependencies"
Cohesion: 0.08
Nodes (30): dependencies, crypto-js@^4.2.0, idb@^8.0.0, @mdi/font@^7.4.47, pdfkit@^0.17.2, svg-to-pdfkit@^0.1.8, vue@^3.5.0, vuetify@^4.2.2 (+22 more)

### Community 7 - "Workspace PDF Actions UI"
Cohesion: 0.08
Nodes (34): addMissingDocuments(), addWorkspacePDFs(), applyWorkspaceOracleSync(), applyWorkspacePDFNameSuggestions(), associatePendingWorkspacePDF(), confirmMergeAllPending(), confirmWorkspaceFusion(), coverageDocumentSequence() (+26 more)

### Community 8 - "Objection Rows Builder"
Cohesion: 0.13
Nodes (27): buildSelectedObjectionRows(), cleanupObjectionRows(), normalizeCedulaValue(), normalizedWorkspaceRelativePath(), objectionCloseoutMissing(), objectionHeaderStatuses(), objectionPDFIncluded(), objectionPDFRecord() (+19 more)

### Community 9 - "Folio PDF Review Skill"
Cohesion: 0.10
Nodes (24): Skill folio-pdf-review, cmd/server/clasificacion.go classifier, OCR fallback: Tesseract Spanish on first page only when no text, pdftotext vs Go PDF parser difference, review_pdfs.py helper (read-only PDF review), Fusión de PDFs manual; cronológica automática no demostrada, Query: cómo se clasifica una epicrisis, HCU_006 Epicrisis rule: four phrases, hierarchical scoring, pending on tie (+16 more)

### Community 10 - "PDF Review Script"
Cohesion: 0.15
Nodes (12): collect_inputs(), decide(), extract_text(), filename_code(), first_page_header(), main(), normalize(), phrase_matches() (+4 more)

### Community 11 - "Demo Batch Generator"
Cohesion: 0.11
Nodes (14): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+6 more)

### Community 12 - "Coverage Logic"
Cohesion: 0.17
Nodes (23): containsInt64(), coverageDocumentDate(), coverageDocumentMetadataItems(), coverageHasDate(), coverageMemberCedulas(), coverageMembers(), coverageMembersAt(), filterCoverageMembers() (+15 more)

### Community 13 - "Product Principles"
Cohesion: 0.12
Nodes (26): Product decision: Archivo histórico en el menú, Principios de producto de Folio, Evidence: search and filters for saved periods, Decision: import team shared folders (Carpetas del equipo), Remove all PDFs of selected team folders with three confirmations, Ranked user jobs (find, save, recognize, batch, recover), Manual 10-digit cédula for team-folder PDFs, Decision: coverage query with chosen date (+18 more)

### Community 14 - "Objection Sync Tests"
Cohesion: 0.20
Nodes (22): objectionHeaderName(), objectionMatrixFilename(), callObjectionSync(), objectionSyncFixture(), persistTestObjectionJob(), TestAppendObjectionRowsPreservesWorkAndInvalidatesExistingMatrix(), TestBuildSelectedObjectionRowsIncludesOnlyChosenCandidates(), TestCreateObjectionWorkspaceReportsExistingDuplicatesWithoutCreatingAnother() (+14 more)

### Community 15 - "PDF Classifier"
Cohesion: 0.20
Nodes (19): addDateCandidateForPeriod(), classifyPDF(), datesOutsideBilledPeriod(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules(), matchClassification(), matchClassificationWithReason() (+11 more)

### Community 16 - "Coverage Tests"
Cohesion: 0.15
Nodes (11): TestCoverageHasDateUsesDateAssociatedWithSavedPDFs(), TestCoverageMembersAtUsesChosenDateForMinorAndReferences(), TestGenerateIndividualCoverageRejectsInvalidInputsBeforePortal(), TestGenerateIndividualCoverageRequiresSession(), TestMissingCoverageMembersForDateOnlyReturnsMembersWithoutThatDate(), TestNormalizeCoverageDateRequiresRealISODate(), TestCurrentSessionExposesDeleteWorkspacePermission(), TestListWorkspacesSkipsHiddenDeletionRecoveryDirectories() (+3 more)

### Community 17 - "Objection Sync Copy"
Cohesion: 0.21
Nodes (20): normalizePatientFolder(), copyObjectionPDFTree(), installObjectionRows(), objectionPatientFolder(), cloneDeletedPDFs(), cloneObjectionSourcePDFs(), clonePlanillaMappings(), collectObjectionSourcePDFs() (+12 more)

### Community 18 - "Operations Manual & Queries"
Cohesion: 0.12
Nodes (21): Query: why delete saved period failed, deleteWorkspaceDialog confirm by exact workspace ID, Query: Oracle-only planilla absence vs ZIP, ingestPreviewReadyToPrepare: ZIP is source of truth, Oracle-only rows skipped, Manual de operación de Folio, Data directory and offline backup procedure, Private .env configuration, folio.service systemd unit (+13 more)

### Community 19 - "Ingest File Utilities"
Cohesion: 0.14
Nodes (13): copyWithLimit(), backupAndReplace(), newUploadJobID(), replaceWorkspaceOutputs(), summarizeClassification(), writePrivateJSON(), classificationReport, classificationResult (+5 more)

### Community 20 - "Planilla Provisioning"
Cohesion: 0.15
Nodes (10): collectZIPPlanillaNumbers(), finalizeNewPlanilla(), missingPlanillaInsertSQL(), oracleServiceCodesForIngest(), TestCollectZIPPlanillaNumbers(), TestCollectZIPPlanillaNumbersRejectsInvalidPDFPaths(), TestMissingPlanillaInsertSQLIsScopedToSelectedMSPPeriodAndService(), TestOracleServiceCodesForIngest() (+2 more)

### Community 21 - "Deploy Skill"
Cohesion: 0.15
Nodes (14): Skill desplegar-spd, Recovery copies and rollback on failed deploy, deploy-runtime.sh --confirm-deploy helper, install-runtime-deploy-access.sh one-time setup, Wrapper /usr/local/sbin/spd-deploy-runtime (sudo NOPASSWD), Skill runtime-freshness, FRESHNESS_OK / FRESHNESS_STALE / FRESHNESS_UNVERIFIED, Compare active service process executable, bin/folio-server, dist/ and HTTP frontend with fresh build (+6 more)

### Community 22 - "Coverage & Archive Filters UI"
Cohesion: 0.13
Nodes (18): auditorImportWorkspaceItems, coverageMonthLabel(), coverageServiceLabel(), filteredHistoricalArchives, filteredObjectionPatients, generateCoverageSheets(), generateCoverageSheetsAtChosenDate(), generateCoverageSheetsForDate() (+10 more)

### Community 23 - "Oracle Document Sync"
Cohesion: 0.17
Nodes (11): fileSHA256(), oracleQualified(), receiptsHas(), syncOracleAuditorReceipts(), workspacePlanillaOptions(), auditorReceipt, oracleDocument, workspaceOraclePlanilla (+3 more)

### Community 24 - "Oracle Requirements"
Cohesion: 0.15
Nodes (12): RF-02 Resolucion de identidades y normalizacion de nombres, FetchMonthlyCache precarga masiva map[int64]PlanillaDTO, godror y Oracle Instant Client pool, PDI_TRAMITE (clave de carpeta numerica), PlanillaDTO, Saneamiento de nombres de paciente, PipelineRunner interfaz Go, Worker Pool concurrente de goroutines (+4 more)

### Community 25 - "Objection & Closeout Requirements"
Cohesion: 0.14
Nodes (12): RF-05 Modulo de gestion de objeciones, RF-06 Guardian de cierre y empaquetado, Visor dividido SplitPdfViewer, Archivo PENDIENTE_tmp_*.pdf, Limites: sin DPI, sin checklist integral, sin escritura Oracle, APIs /api/v1/empaquetado (auditar, compilar, rotulado-cd), Auditoria pre-flight de cierre, Auditoria tecnica PDF 72-300 DPI y firmas PAdES (+4 more)

### Community 26 - "Hybrid Classification Requirements"
Cohesion: 0.23
Nodes (13): RF-03 Clasificacion hibrida y fusion PDF, RF-04 Subsanacion manual y control de faltantes, APIs REST /api/v1/subsanacion, Catalogo oficial MSP para desplegable, Clasificacion manual desde visor dividido, Inyeccion puntual de documento faltante, Fusion de duplicados con pdfcpu.Merge, Extension .pdf explicita en minusculas (+5 more)

### Community 27 - "Oracle Date Parsing"
Cohesion: 0.27
Nodes (5): parseOracleDate(), parseOraclePlanillaID(), sortedPlanillaIDs(), workspaceAuditorReceiptID(), planillaIdentity

### Community 28 - "Classification Rules Config"
Cohesion: 0.17
Nodes (12): base, codigo, entrada, codigos_nombre_archivo_confirmados, ignorar_nombre_archivo, ocr_respaldo, paginas_texto_vectorial, dpi_max (+4 more)

### Community 29 - "Dual Ingest & Objections Reqs"
Cohesion: 0.20
Nodes (12): RF-01 Ingesta dual de informacion, 5. ANEXOS justificativos por tramite, C_COBERTURA.pdf por tramite en objeciones, Condiciones de cierre del ZIP de objeciones, Espacio derivado de objeciones, Flujo manual de Objeciones en Folio, I_LIQUIDACION informe de liquidacion MSP, Matriz de objeciones .xlsm (no interpretada) (+4 more)

### Community 30 - "Hybrid OCR Classifier"
Cohesion: 0.21
Nodes (10): Clasificador hibrido vectorial + OCR, OCR Tesseract espanol (pagina 1, 200-300 DPI), reglas_clasificacion.yaml evaluador, Alerta de fechas fuera del periodo facturado, Banco de pruebas anonimizado, Catalogo de senales confirmadas por codigo, Reglas de entrada, extraccion y clasificacion, Oficio MSP-DP17-2026-6548-O (fuente no incluida) (+2 more)

### Community 31 - "Ingest Frontend Loaders"
Cohesion: 0.21
Nodes (12): checkSession(), classifyIngest(), clearIngestTramiteMapping(), forgetUnavailableIngest(), loadDocuments(), loadIngestPreview(), processIngest(), replaceIngestZIP() (+4 more)

### Community 32 - "Historical Archive"
Cohesion: 0.27
Nodes (6): historicalArchiveDuplicateMessage(), historicalArchiveMonthLabel(), historicalArchiveServiceLabel(), newHistoricalArchiveID(), normalizeHistoricalArchiveSearch(), historicalArchiveRecord

### Community 33 - "Historical Archive Tests"
Cohesion: 0.53
Nodes (8): historicalArchiveTestServer(), historicalUploadRequest(), makeHistoricalTestZIP(), TestConcurrentHistoricalArchiveUploadsKeepOnePerServicePeriod(), TestDeleteHistoricalArchiveRequiresRoleAndExactIDConfirmation(), TestHistoricalArchiveRejectsDuplicateServicePeriod(), TestHistoricalArchiveRequiresSessionAndRejectsInvalidZIP(), TestHistoricalArchiveUploadPreservesOriginalZIPAndCanSearchAndDownload()

### Community 35 - "Package Structure Rules"
Cohesion: 0.22
Nodes (9): Catalogo de servicios del selector, Control de completitud por servicio, Compilador ISO/ZIP del paquete final, Documentos del expediente por servicio, Estructura de primer ingreso, Red Privada (RPC) A_ENTREGA y C_VALIDACION, Tipos de servicio y tokens canonicos, Matriz de documentos requeridos por servicio (+1 more)

### Community 36 - "Auditor ZIP Import"
Cohesion: 0.39
Nodes (8): cleanupAuditorImportStages(), inspectAuditorZIP(), auditorImportApplyRequest, auditorImportFile, auditorImportFolder, auditorImportPlanilla, auditorImportPreviewResponse, auditorImportStage

### Community 37 - "Merge Limit Decisions"
Cohesion: 0.28
Nodes (8): Query: fusión HCU_010 límite 50, Original merge endpoint limit 2-50 PDFs; HCU_010 group of 56 rejected, Query: límite 100 por grupo, maxWorkspaceFusionPDFs raised to 100, Query: YEROVI 277 PDFs límite 300, Interim merge limit 300 with 300s UI timeout (superseded), Query (corrected): límite 1000 por grupo, Corrected limit 1000 per group, 1 MiB JSON body, 10-minute UI timeout

### Community 38 - "Institutional Logos"
Cohesion: 0.25
Nodes (8): Escudo del Ecuador (grayscale coat of arms with condor), IESS logo (Instituto Ecuatoriano de Seguridad Social; white-on-transparent, renders blank), ISSFA logo (Instituto de Seguridad Social de las Fuerzas Armadas; white-on-transparent, renders blank), ISSPOL logo (Instituto de Seguridad Social de la Policia Nacional; white-on-transparent, renders blank), Ministerio de Salud Publica wordmark (dark blue text), Red Publica Integral de Salud (RPIS) logo with yellow-dot Ecuador map, Ministerio de Defensa logo (white-on-transparent, renders blank), Ministerio del Interior logo (white-on-transparent, renders blank)

### Community 39 - "Oracle Schema & Repository"
Cohesion: 0.29
Nodes (8): Compatibilidad Oracle 11gR2 (secuencia, trigger, CHECK, indices), Estados PDI_ESTADO_DIGITALIZACION, Tabla DIGITALIZACION.PLANILLA_DIGITAL, PlanillaRepositoryInterface, UpdateObjetadosBatch, Semaforo de completitud por paciente, Cierre de lote PDI_CERRADO, PDI_ESTADO_DIGITALIZACION no integrado (N039)

### Community 40 - "Real Implementation Status"
Cohesion: 0.25
Nodes (8): Estado de implementacion de la estructura, Biblioteca IndexedDB (N040), Conflictos CFL-01 a CFL-08, Criterio de evidencia (REQUERIDO/IMPLEMENTADO/VERIFICADO), Guardian de cierre no implementado (N038), Nodos N001-N043 de estado real, Relaciones R001-R040, Sin verificacion E2E: solo build y HTTP 200

### Community 42 - "Objection Annex Naming"
Cohesion: 0.38
Nodes (6): atomicWritePrivateFile(), nextSafeAnnexName(), normalizeAnnexFilename(), objectionAnnexBase(), saveObjectionVersion(), TestObjectionAnnexNamesAreCanonicalAndCollisionSafe()

### Community 43 - "Classification Confidence Rules"
Cohesion: 0.29
Nodes (7): coincidencias, confianza_ocr, confianza_vectorial_con_coincidencia, empate, precedencia, sin_coincidencia, texto_vacio_o_ilegible

### Community 44 - "MSP Catalog Decisions"
Cohesion: 0.40
Nodes (6): catalogos/codigos_msp.json, Query: Anexos.pdf en catálogo MSP, Add MSP name by editing catalogos/codigos_msp.json (title, value, categoria) and rebuild frontend, renameWorkspacePDF / loadMSPPDFCodeSet validate allowed names, Decision: classify by document header, Decision: apply suggested names in batch (Actualizar nombres)

### Community 45 - "Implementation Status Queries"
Cohesion: 0.40
Nodes (6): Query: estado real de implementación N001-N043, Conflicts CFL-01..CFL-08 between docs and code, Guardián de cierre (requerido, no implementado), Evidence levels REQUERIDO/IMPLEMENTADO/PARCIAL/VERIFICADO/NO VERIFICADO, Query: de qué trata esta app, Not yet integrated: duplicate merge validation, full required-doc validation, partial upload, Oracle digitization state

### Community 46 - "Dual Ingest Frontend"
Cohesion: 0.40
Nodes (5): Frontend de ingesta y carga dual (Vuetify), POST /api/v1/ingesta/lote-dual, fileValidators pre-validacion cliente, ingestaStore (Pinia) estado del formulario, Respuesta HTTP 202 con job_id

### Community 47 - "Workspace ZIP Download"
Cohesion: 0.70
Nodes (4): headerOutputFilename(), matrixFilename(), TestDownloadWorkspaceZIPMarksIncompleteProgressWithoutChangingWorkspace(), TestDownloadWorkspaceZIPNamesCompletePackageForDelivery()

### Community 48 - "Classification Rule Keys"
Cohesion: 0.40
Nodes (5): reglas, codigo, palabras_clave, palabras_clave_prioritarias, palabras_clave_titulo

### Community 49 - "OpenAI Skill Config"
Cohesion: 0.50
Nodes (3): interface, display_name, short_description

### Community 50 - "Coverage Bot Overview"
Cohesion: 0.67
Nodes (3): Bot autonomo de consulta de coberturas (IESS/MSP), Siete modulos tecnicos del proyecto, Coberturas y PDI_COBERTURA

### Community 51 - "Objection Period Spaces"
Cohesion: 0.67
Nodes (3): objectionPeriodSpaces, objectionSourcePeriodSpaces, sameObjectionPeriod()

### Community 52 - "Upload Dropzone"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

## Ambiguous Edges - Review These
- `Classification precedence: filename alias, header phrases, priority phrases, regular rules` → `Tie broken by longest specific phrase (conflicts with current rule: equal counts mean pending)`  [AMBIGUOUS]
  graphify-out/memory/query_20261023_202221_a0eeb6ab_que_nombre_le_deberias_dar_a__home_spd_he1_recurso.md · relation: references

## Knowledge Gaps
- **54 isolated node(s):** `folio`, `Recovery copies and rollback on failed deploy`, `Skill folio-pdf-review`, `pdftotext vs Go PDF parser difference`, `Private .env configuration` (+49 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 530 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `NPM Dependencies` to `App.vue Main Component State`, `Saved Workspace Picker`?**
  _High betweenness centrality (0.023) - this node is a cross-community bridge._
- **Are the 42 inferred relationships involving `packageFolderName()` (e.g. with `objectionHeaderStatuses()` and `objectionSyncFixture()`) actually correct?**
  _`packageFolderName()` has 42 INFERRED edges - model-reasoned connections that need verification._
- **What connects `folio`, `Recovery copies and rollback on failed deploy`, `Skill folio-pdf-review` to the rest of the system?**
  _54 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue Main Component State` be split into smaller, more focused modules?**
  _Cohesion score 0.0057306590257879654 - nodes in this community are weakly interconnected._
- **Why does `idb` connect `NPM Dependencies` to `App.vue Main Component State`?**
  _High betweenness centrality (0.021) - this node is a cross-community bridge._
- **Should `PDF Ingest Handlers` be split into smaller, more focused modules?**
  _Cohesion score 0.10270896711574677 - nodes in this community are weakly interconnected._

### Low-confidence Hints
_AMBIGUOUS edges — the extractor was unsure. Verify before acting on these._

- **What is the exact relationship between `Classification precedence: filename alias, header phrases, priority phrases, regular rules` and `Tie broken by longest specific phrase (conflicts with current rule: equal counts mean pending)`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._