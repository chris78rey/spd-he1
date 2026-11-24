# Graph Report - spd_msp  (2026-11-24)

## Corpus Check
- 78 files · ~137,666 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 7 file(s) not represented in the graph (top: (none) 3, .example 1, .zip 1)

## Summary
- 1507 nodes · 3911 edges · 62 communities (51 shown, 11 thin omitted)
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 934 edges (avg confidence: 0.85)
- Token cost: 98,403 input · 0 output

## Graph Freshness
- Built from commit: `7de6c2fc`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- App.vue Main UI State
- Server Core Handlers
- objeciones.go
- PDF Review Skill & Rules
- Coverage Records & Indexing
- Demo Lot & Coverage Marking
- Workspace PDF Actions UI
- Workspace Rename & ZIP
- Workspace Picker & Search
- testing.T
- PDF Classification Engine
- Objections & Team Import UI
- Runtime Deploy Skill
- stagedJob
- Coverage Replacement Script
- Frontend Dependencies
- PDF Review Python Helper
- ingesta.go
- Buscar por Planilla
- go_pkg_path_filepath
- loadIngestPreview
- MSP Portal Downloader
- Product Principles
- Coverage Page UI
- objeciones_sync.go
- Implementation Status Audit
- Runtime Config & Timezone
- Oracle Identity Requirements
- Objections & Closure Reqs
- OCR & Oracle Schema Reqs
- Manual Classification Reqs
- Dual Ingest & Objections Reqs
- oracle_documents.go
- workspace_auditor_import.go
- Implementation Evidence
- Package Structure Rules
- Chrome DevTools Client
- Historical Archive Tests
- planilla_folder_link_test.go
- Find and recover documents (top user job)
- Requirements Documents
- reemplazo_coberturas.go
- PDF Merge Limits
- Decision: coverage query with chosen date
- README SPD MSP
- Delete Permissions
- Decision: Objeciones in separate derived workspace
- Ingest Frontend Reqs
- Replacement Script Tests
- Agent Interface Config
- Last-Attention Coverage UI
- Cross-Trámite Coverage UI
- Objection Period Spaces
- Library Upload UI
- SPA Entry Point
- System Vision & Stack
- Deploy Access Installer
- Service Installer
- Fusion Drag & Drop
- Auditor Folder Mapping
- Go Module

## God Nodes (most connected - your core abstractions)
1. `server` - 125 edges
2. `stagedJob` - 112 edges
3. `packageFolderName()` - 63 edges
4. `main()` - 51 edges
5. `writeError()` - 50 edges
6. `writeJSON()` - 43 edges
7. `signOut()` - 33 edges
8. `Principios de producto de Folio` - 33 edges
9. `validJobID()` - 29 edges
10. `safeWorkspacePath()` - 28 edges

## Surprising Connections (you probably didn't know these)
- `Replacing executable or restarting service is a deployment requiring explicit authorization` --semantically_similar_to--> `Deploy only with explicit user authorization`  [INFERRED] [semantically similar]
  AGENTS.md → .agents/skills/desplegar-spd/SKILL.md
- `Query: de qué trata esta app` --references--> `Principios de producto de Folio`  [INFERRED]
  graphify-out/memory/query_20261023_185439_02b394f2_revisa_en_graphify_de_que_se_trata_este_app.md → PRINCIPIOS_DE_PRODUCTO.md
- `Biblioteca IndexedDB (N040)` --semantically_similar_to--> `Mis documentos library in browser IndexedDB`  [INFERRED] [semantically similar]
  requerimientos/09_estado_real_implementacion.md → README.md
- `Coberturas y PDI_COBERTURA` --semantically_similar_to--> `Descargar hojas de cobertura (C_COBERTURA.pdf, up to 10 planillas, PDI_COBERTURA='S')`  [INFERRED] [semantically similar]
  requerimientos/09_estado_real_implementacion.md → README.md
- `Clasificador hibrido implementado (N015)` --semantically_similar_to--> `OCR/classification pipeline (pdftoppm, Tesseract spa, PENDIENTE_tmp_)`  [INFERRED] [semantically similar]
  requerimientos/09_estado_real_implementacion.md → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Pending frontier of the formal closure** — requerimientos_09_estado_real_implementacion_checklist_no_implementado, requerimientos_09_estado_real_implementacion_objeciones_parcial, requerimientos_09_estado_real_implementacion_estado_digitalizacion_no_integrado, requerimientos_09_estado_real_implementacion_guardian_no_implementado [EXTRACTED 1.00]
- **Componentes del Guardian de cierre** — requerimientos_06req_motor_cierre_auditoria_preflight, requerimientos_06req_motor_cierre_auditoria_tecnica_pdf_dpi, requerimientos_06req_motor_cierre_compilador_iso_zip, requerimientos_06req_motor_cierre_rotulado_cd, requerimientos_06req_motor_cierre_cierre_lote_oracle [EXTRACTED 1.00]
- **Runtime freshness and deploy policy** — agents, _agents_skills_runtime_freshness_skill, _agents_skills_desplegar_spd_skill, _agents_skills_runtime_freshness_skill_verify_runtime_helper, _agents_skills_desplegar_spd_skill_deploy_runtime_helper [EXTRACTED 1.00]
- **Coverage sheet flows sourced from the MSP portal** — principios_de_producto_cobertura_portal_msp_normativa, principios_de_producto_cobertura_fecha_elegida, principios_de_producto_cobertura_ultima_atencion_lote, principios_de_producto_fecha_propuesta_cobertura, principios_de_producto_cobertura_individual, readme_cobertura_download [INFERRED 0.85]
- **Objeciones derived workspace lifecycle** — principios_de_producto_objeciones_espacio_separado, principios_de_producto_objeciones_reutilizar, principios_de_producto_objeciones_sincronizar, principios_de_producto_objeciones_zip_requisitos, principios_de_producto_objeciones_paginacion, requerimientos_09_estado_real_implementacion_objeciones_parcial [INFERRED 0.85]
- **PDF classification by header, rules and OCR** — _agents_skills_folio_pdf_review_skill_clasificacion_go, reglas_clasificacion, _agents_skills_folio_pdf_review_skill_ocr_fallback, _agents_skills_folio_pdf_review_skill_header_first_precedence, principios_de_producto_clasificar_encabezado [INFERRED 0.85]
- **Pipeline de clasificacion documental (vectorial, OCR, YAML, pendiente, fusion)** — requerimientos_04ocr_go_clasificador_hibrido, requerimientos_04ocr_go_ocr_tesseract_spa, requerimientos_04ocr_go_reglas_yaml, requerimientos_04ocr_go_pendiente_tmp, requerimientos_04ocr_go_fusion_pdfcpu_merge [INFERRED 0.85]

## Communities (62 total, 11 thin omitted)

### Community 0 - "App.vue Main UI State"
Cohesion: 0.01
Nodes (310): activeDoc, activeFolder, activePage, auditorImportAllFilesMapped, auditorImportBusy, auditorImportError, auditorImportFile, auditorImportFolderDefaults (+302 more)

### Community 1 - "Server Core Handlers"
Cohesion: 0.09
Nodes (48): atomicWritePrivateFile(), copyPrivateFile(), headerOutputFilename(), jobResponse(), matrixFilename(), missingHeaders(), packageFolderName(), periodWorkspaceID() (+40 more)

### Community 2 - "objeciones.go"
Cohesion: 0.14
Nodes (22): cleanupObjectionRows(), nextSafeAnnexName(), normalizeAnnexFilename(), normalizeCedulaValue(), normalizedWorkspaceRelativePath(), objectionAnnexBase(), objectionCloseoutMissing(), objectionHeaderStatuses() (+14 more)

### Community 3 - "PDF Review Skill & Rules"
Cohesion: 0.05
Nodes (46): Skill folio-pdf-review, cmd/server/clasificacion.go classifier, OCR fallback: Tesseract Spanish on first page only when no text, pdftotext vs Go PDF parser difference, review_pdfs.py helper (read-only PDF review), Query: cómo se clasifica una epicrisis, HCU_006 Epicrisis rule: four phrases, hierarchical scoring, pending on tie, Query: nombre sugerido HCU_006 (1).pdf (+38 more)

### Community 4 - "Coverage Records & Indexing"
Cohesion: 0.12
Nodes (37): adoptRemovedPaths(), containsInt64(), coverageDocumentDate(), coverageDocumentMetadataItems(), coverageHasDate(), coverageMemberCedulas(), coverageMembers(), coverageMembersAt() (+29 more)

### Community 5 - "Demo Lot & Coverage Marking"
Cohesion: 0.11
Nodes (14): buildJPEGImagePDF(), buildPDF(), demoPDF(), escapePDF(), generate(), main(), rasterOCRDemoPDF(), writeDemoPackage() (+6 more)

### Community 6 - "Workspace PDF Actions UI"
Cohesion: 0.08
Nodes (36): addMissingDocuments(), addWorkspacePDFs(), applyWorkspaceOracleSync(), applyWorkspacePDFNameSuggestions(), associatePendingWorkspacePDF(), confirmMergeAllPending(), confirmWorkspaceFusion(), coverageDateLabel() (+28 more)

### Community 7 - "Workspace Rename & ZIP"
Cohesion: 0.09
Nodes (39): coverageExtraPlanillas(), applyWorkspacePDFFileRenames(), applyWorkspacePDFRenameMetadata(), archiveRelativePatientPath(), assignWorkspaceNameTargets(), canonicalMSPPDFCode(), compareArchivePatientTieBreak(), TestCrossTramiteCoverageGroupIsOptionalAndNeverPending() (+31 more)

### Community 8 - "Workspace Picker & Search"
Cohesion: 0.09
Nodes (34): filteredObjectionPatients, filteredSavedWorkspaces, orderedSavedWorkspaces, savedWorkspaceYearFilters, workspaceReceivedAt(), displayedModelValue, emit, filteredWorkspaces (+26 more)

### Community 9 - "testing.T"
Cohesion: 0.11
Nodes (38): TestCoverageHasDateUsesDateAssociatedWithSavedPDFs(), TestCoverageMembersAtUsesChosenDateForMinorAndReferences(), TestCoverageScriptFailureTag(), TestCoverageSheetHasDateRequiresRequestedDate(), TestGenerateIndividualCoverageLivePortal(), TestGenerateIndividualCoverageRejectsInvalidInputsBeforePortal(), TestGenerateIndividualCoverageRequiresSession(), TestKeepMissingMembersDropsSheetsSavedMeanwhile() (+30 more)

### Community 10 - "PDF Classification Engine"
Cohesion: 0.13
Nodes (27): addDateCandidateForPeriod(), classifyPDF(), copyWithLimit(), datesOutsideBilledPeriod(), extractPDFText(), formatBilledPeriod(), isRelevantDocumentDateContext(), loadClassificationRules() (+19 more)

### Community 11 - "Objections & Team Import UI"
Cohesion: 0.08
Nodes (34): addObjectionPatients(), applyAuditorFolderImport(), chooseObjectionWorkspace(), clearAuditorImportPreview(), closeHistoricalArchiveDelete(), createObjectionWorkspace(), createOrOpenAuditorPeriod(), deleteHistoricalArchive() (+26 more)

### Community 12 - "Runtime Deploy Skill"
Cohesion: 0.09
Nodes (25): Skill desplegar-spd, Recovery copies and rollback on failed deploy, deploy-runtime.sh --confirm-deploy helper, install-runtime-deploy-access.sh one-time setup, Wrapper /usr/local/sbin/spd-deploy-runtime (sudo NOPASSWD), catalogos/codigos_msp.json, Skill runtime-freshness, FRESHNESS_OK / FRESHNESS_STALE / FRESHNESS_UNVERIFIED (+17 more)

### Community 13 - "stagedJob"
Cohesion: 0.16
Nodes (15): normalizePatientFolder(), parseOracleDate(), oracleTableName(), removeObjectionFile(), upsertObjectionFile(), finalizeNewPlanilla(), insertPlanillaFromSource(), parseOraclePlanillaID() (+7 more)

### Community 14 - "Coverage Replacement Script"
Cohesion: 0.12
Nodes (28): apply(), compareSheets(), crypto, detectOrigin(), DOWNLOADER, { execFileSync, spawnSync }, findCoverageFiles(), fs (+20 more)

### Community 15 - "Frontend Dependencies"
Cohesion: 0.10
Nodes (24): dependencies, idb@^8.0.0, @mdi/font@^7.4.47, vue@^3.5.0, vuetify@^4.2.2, devDependencies, vite@^7.0.0, @vitejs/plugin-vue@^6.0.0 (+16 more)

### Community 16 - "PDF Review Python Helper"
Cohesion: 0.15
Nodes (12): collect_inputs(), decide(), extract_text(), filename_code(), first_page_header(), main(), normalize(), phrase_matches() (+4 more)

### Community 17 - "ingesta.go"
Cohesion: 0.10
Nodes (19): historicalArchiveDuplicateMessage(), historicalArchiveMonthLabel(), historicalArchiveServiceLabel(), newHistoricalArchiveID(), normalizeHistoricalArchiveSearch(), backupAndReplace(), newUploadJobID(), replaceWorkspaceOutputs() (+11 more)

### Community 18 - "Buscar por Planilla"
Cohesion: 0.28
Nodes (11): classifySISPlanilla(), containsString(), folderLinkMonthYear(), folderLinkNameWarning(), folderLinkOwner(), folderLinkValidation(), serviceCodeLabel(), oracleServiceCodesForIngest() (+3 more)

### Community 19 - "go_pkg_path_filepath"
Cohesion: 0.14
Nodes (11): TestCoveragePageIsPortalDistinguishesPortalFromLocalSheets(), TestCoveragePortalCedulasRejectsUnreadableFile(), TestCurrentSessionExposesDeleteWorkspacePermission(), crossTramiteCoverageFixture(), TestCrossTramiteCoverageGroupNotOfferedForSingleTramite(), TestObjectionCopyIncludesJoinedCoverageForNonPrimaryTramite(), TestWorkspaceCoverageFusionPreservesDateAndCedulaMetadata(), TestListWorkspacesSkipsHiddenDeletionRecoveryDirectories() (+3 more)

### Community 20 - "loadIngestPreview"
Cohesion: 0.13
Nodes (22): checkSession(), classifyIngest(), clearIngestTramiteMapping(), deleteWholeWorkspace(), ensurePatientDocumentsWorkspaceSelection(), forgetUnavailableIngest(), loadDocuments(), loadIngestPreview() (+14 more)

### Community 21 - "MSP Portal Downloader"
Cohesion: 0.14
Nodes (19): closeChrome(), consult(), findChrome(), fs, isTrue(), launchChrome(), main(), mark() (+11 more)

### Community 22 - "Product Principles"
Cohesion: 0.14
Nodes (19): Principios de producto de Folio, Decision: search planilla by PDI_TRAMITE from patient folder, Decision: import team shared folders (Carpetas del equipo), Remove all PDFs of selected team folders with three confirmations, Ranked user jobs (find, save, recognize, batch, recover), Decision: classify by document header, Manual 10-digit cédula for team-folder PDFs, Decision: batch re-query with last attention of each trámite (+11 more)

### Community 23 - "Coverage Page UI"
Cohesion: 0.12
Nodes (19): auditorImportWorkspaceItems, coverageMonthLabel(), coverageServiceLabel(), filteredHistoricalArchives, generateCoverageSheets(), generateCoverageSheetsAtChosenDate(), generateCoverageSheetsAtLastAttention(), generateCoverageSheetsForDate() (+11 more)

### Community 24 - "objeciones_sync.go"
Cohesion: 0.18
Nodes (19): copyObjectionPDFTree(), installObjectionRows(), objectionPDFRecord(), cloneDeletedPDFs(), cloneObjectionSourcePDFs(), clonePlanillaMappings(), collectObjectionSourcePDFs(), ensureObjectionSourcePDFMap() (+11 more)

### Community 25 - "Implementation Status Audit"
Cohesion: 0.17
Nodes (14): Query: estado real de implementación N001-N043, Conflicts CFL-01..CFL-08 between docs and code, Fusión de PDFs manual; cronológica automática no demostrada, Guardián de cierre (requerido, no implementado), Evidence levels REQUERIDO/IMPLEMENTADO/PARCIAL/VERIFICADO/NO VERIFICADO, Query: de qué trata esta app, Not yet integrated: duplicate merge validation, full required-doc validation, partial upload, Oracle digitization state, Estado de implementacion de la estructura (+6 more)

### Community 26 - "Runtime Config & Timezone"
Cohesion: 0.20
Nodes (11): formatDateTimeInTimezone(), formatTimestampSlugInTimezone(), fs, getConfiguredTimezone(), getDateTimeParts(), getHttpsAgent(), https, isTrue() (+3 more)

### Community 27 - "Oracle Identity Requirements"
Cohesion: 0.14
Nodes (13): RF-02 Resolucion de identidades y normalizacion de nombres, FetchMonthlyCache precarga masiva map[int64]PlanillaDTO, godror y Oracle Instant Client pool, PDI_TRAMITE (clave de carpeta numerica), PlanillaDTO, PlanillaRepositoryInterface, Saneamiento de nombres de paciente, UpdateObjetadosBatch (+5 more)

### Community 28 - "Objections & Closure Reqs"
Cohesion: 0.14
Nodes (12): RF-05 Modulo de gestion de objeciones, RF-06 Guardian de cierre y empaquetado, Visor dividido SplitPdfViewer, Archivo PENDIENTE_tmp_*.pdf, Limites: sin DPI, sin checklist integral, sin escritura Oracle, APIs /api/v1/empaquetado (auditar, compilar, rotulado-cd), Auditoria pre-flight de cierre, Auditoria tecnica PDF 72-300 DPI y firmas PAdES (+4 more)

### Community 29 - "OCR & Oracle Schema Reqs"
Cohesion: 0.18
Nodes (12): Compatibilidad Oracle 11gR2 (secuencia, trigger, CHECK, indices), Tabla DIGITALIZACION.PLANILLA_DIGITAL, Clasificador hibrido vectorial + OCR, OCR Tesseract espanol (pagina 1, 200-300 DPI), reglas_clasificacion.yaml evaluador, Cierre de lote PDI_CERRADO, Alerta de fechas fuera del periodo facturado, Banco de pruebas anonimizado (+4 more)

### Community 30 - "Manual Classification Reqs"
Cohesion: 0.23
Nodes (13): RF-03 Clasificacion hibrida y fusion PDF, RF-04 Subsanacion manual y control de faltantes, Estados PDI_ESTADO_DIGITALIZACION, APIs REST /api/v1/subsanacion, Catalogo oficial MSP para desplegable, Clasificacion manual desde visor dividido, Inyeccion puntual de documento faltante, Semaforo de completitud por paciente (+5 more)

### Community 31 - "Dual Ingest & Objections Reqs"
Cohesion: 0.22
Nodes (11): RF-01 Ingesta dual de informacion, 5. ANEXOS justificativos por tramite, C_COBERTURA.pdf por tramite en objeciones, Condiciones de cierre del ZIP de objeciones, Espacio derivado de objeciones, Flujo manual de Objeciones en Folio, I_LIQUIDACION informe de liquidacion MSP, Matriz de objeciones .xlsm (no interpretada) (+3 more)

### Community 32 - "oracle_documents.go"
Cohesion: 0.18
Nodes (10): coveragePageIsPortal(), coveragePageTexts(), coveragePortalCedulas(), fileSHA256(), oracleQualified(), receiptsHas(), syncOracleAuditorReceipts(), workspaceFolderPlanillas() (+2 more)

### Community 33 - "workspace_auditor_import.go"
Cohesion: 0.11
Nodes (18): collectZIPPlanillaNumbers(), missingPlanillaInsertSQL(), missingPlanillaSourceSQL(), TestCollectZIPPlanillaNumbers(), TestCollectZIPPlanillaNumbersRejectsInvalidPDFPaths(), TestMissingPlanillaInsertSQLIsScopedToSelectedMSPPeriodAndService(), TestOracleServiceCodesForIngest(), TestPlanillaForPatientUsesPersistedZIPAssociation() (+10 more)

### Community 34 - "Implementation Evidence"
Cohesion: 0.20
Nodes (9): Query: Oracle-only planilla absence vs ZIP, ingestPreviewReadyToPrepare: ZIP is source of truth, Oracle-only rows skipped, Permanent job workspace: fuentes/, trabajo/, reportes/, Criterio de evidencia (REQUERIDO/IMPLEMENTADO/VERIFICADO), Gestion documental mayormente implementada, Ingesta y cruce Oracle implementado, Mapa global: núcleo programado vs frontera del cierre formal, Relaciones R001-R040 (+1 more)

### Community 35 - "Package Structure Rules"
Cohesion: 0.22
Nodes (9): Catalogo de servicios del selector, Control de completitud por servicio, Compilador ISO/ZIP del paquete final, Documentos del expediente por servicio, Estructura de primer ingreso, Red Privada (RPC) A_ENTREGA y C_VALIDACION, Tipos de servicio y tokens canonicos, Catalogo de nombres de archivo conocidos (+1 more)

### Community 37 - "Historical Archive Tests"
Cohesion: 0.64
Nodes (8): historicalArchiveTestServer(), historicalUploadRequest(), makeHistoricalTestZIP(), TestConcurrentHistoricalArchiveUploadsKeepOnePerServicePeriod(), TestDeleteHistoricalArchiveRequiresRoleAndExactIDConfirmation(), TestHistoricalArchiveRejectsDuplicateServicePeriod(), TestHistoricalArchiveRequiresSessionAndRejectsInvalidZIP(), TestHistoricalArchiveUploadPreservesOriginalZIPAndCanSearchAndDownload()

### Community 38 - "planilla_folder_link_test.go"
Cohesion: 0.32
Nodes (11): applyPlanillaFolderLink(), folderOtherPlanillas(), folderLinkTestJob(), TestApplyPlanillaFolderLinkKeepsPendingPDFsWhenFolderHasAnotherTramite(), TestApplyPlanillaFolderLinkLinksPendingPDFsOfThatFolderOnly(), TestApplyPlanillaFolderLinkUpdatesExistingEntryInsteadOfDuplicating(), TestClassifySISPlanillaExplainsWhyItCannotBeCreated(), TestFolderLinkNameWarningIgnoresAccentsAndCopySuffix() (+3 more)

### Community 39 - "Find and recover documents (top user job)"
Cohesion: 0.29
Nodes (7): Product decision: Archivo histórico en el menú, Evidence: search and filters for saved periods, Find and recover documents (top user job), Decision: Archivo histórico first in sidebar, Decision: expedientes-first navigation order, Mis documentos library in browser IndexedDB, Biblioteca IndexedDB (N040)

### Community 41 - "reemplazo_coberturas.go"
Cohesion: 0.18
Nodes (14): hasAnyPrefix(), coverageReplacementTestPDF(), newCoverageReplacementFixture(), TestCoverageReplacementPlacesPortalPDFAndKeepsBackup(), TestCoverageReplacementQueueWritesResultAndRemovesRequest(), TestCoverageReplacementRejectsPathsOutsidePreparation(), TestCoverageReplacementRetryRestoresIDsOnlyForToolReplacedSheets(), TestCoverageReplacementSkipsSheetChangedSincePreparation() (+6 more)

### Community 42 - "PDF Merge Limits"
Cohesion: 0.29
Nodes (8): Query: fusión HCU_010 límite 50, Original merge endpoint limit 2-50 PDFs; HCU_010 group of 56 rejected, Query: límite 100 por grupo, maxWorkspaceFusionPDFs raised to 100, Query: YEROVI 277 PDFs límite 300, Interim merge limit 300 with 300s UI timeout (superseded), Query (corrected): límite 1000 por grupo, Corrected limit 1000 per group, 1 MiB JSON body, 10-minute UI timeout

### Community 43 - "Decision: coverage query with chosen date"
Cohesion: 0.32
Nodes (7): Decision: coverage query with chosen date, Decision: Cobertura individual without expediente, Decision: optionally combine coverage sheets of several trámites, Descargar hojas de cobertura (C_COBERTURA.pdf, up to 10 planillas, PDI_COBERTURA='S'), Bot autonomo de consulta de coberturas (IESS/MSP), Siete modulos tecnicos del proyecto, Coberturas y PDI_COBERTURA

### Community 44 - "README SPD MSP"
Cohesion: 0.32
Nodes (8): README SPD MSP, OCR/classification pipeline (pdftoppm, Tesseract spa, PENDIENTE_tmp_), cmd/demo-lote synthetic OCR demo package generator, Local development (npm run dev:api, npm run dev, Vite proxy on 8080), ORACLE_USER technical account for planilla_digital (SELECT, UPDATE PDI_COBERTURA), Planilla digital (PLANILLA_DIGITAL table view), install-service.sh systemd autostart, Clasificador hibrido implementado (N015)

### Community 46 - "Delete Permissions"
Cohesion: 0.40
Nodes (5): Query: why delete saved period failed, deleteWorkspaceDialog confirm by exact workspace ID, Decision: delete historical ZIP by role, Oracle login validation with SPD_EXTERNOS role, 8h session, Acceso y arquitectura implementado (login Oracle + SPD_EXTERNOS)

### Community 47 - "Decision: Objeciones in separate derived workspace"
Cohesion: 0.33
Nodes (6): Decision: progress ZIP download (*_AVANCE_INCOMPLETO.zip) and ready-for-delivery, Download ZIP: Recepción and Objeciones chosen separately, Decision: Objeciones in separate derived workspace, Decision: explicit sync of later documents into Objeciones, Objeciones ZIP required files (oficio, I_LIQUIDACION, oficio de pago, planilla consolidada, matriz xlsm), ZIP patient folder numbering 4. EXPEDIENTES / 5. ANEXOS

### Community 48 - "Ingest Frontend Reqs"
Cohesion: 0.40
Nodes (5): Frontend de ingesta y carga dual (Vuetify), POST /api/v1/ingesta/lote-dual, fileValidators pre-validacion cliente, ingestaStore (Pinia) estado del formulario, Respuesta HTTP 202 con job_id

### Community 50 - "Agent Interface Config"
Cohesion: 0.50
Nodes (3): interface, display_name, short_description

### Community 51 - "Last-Attention Coverage UI"
Cohesion: 0.67
Nodes (4): coverageDownloadableSelectedIds, coverageHasLastAttentionSheet(), coverageLastAttention(), coverageLastAttentionPending

### Community 53 - "Cross-Trámite Coverage UI"
Cohesion: 0.67
Nodes (3): coverageEarliestDate(), coveragePlanillaKey(), workspaceCrossCoverageGroups

### Community 54 - "Objection Period Spaces"
Cohesion: 0.67
Nodes (3): objectionPeriodSpaces, objectionSourcePeriodSpaces, sameObjectionPeriod()

### Community 55 - "Library Upload UI"
Cohesion: 0.67
Nodes (3): onDrop(), prettySize(), upload()

## Ambiguous Edges - Review These
- `Not yet integrated: duplicate merge validation, full required-doc validation, partial upload, Oracle digitization state` → `Fusion de PDFs parcial (manual)`  [AMBIGUOUS]
  requerimientos/09_estado_real_implementacion.md · relation: conceptually_related_to
- `Tie broken by longest specific phrase (conflicts with current rule: equal counts mean pending)` → `Classification precedence: filename alias, header phrases, priority phrases, regular rules`  [AMBIGUOUS]
  graphify-out/memory/query_20261023_202221_a0eeb6ab_que_nombre_le_deberias_dar_a__home_spd_he1_recurso.md · relation: references

## Knowledge Gaps
- **54 isolated node(s):** `folio`, `Recovery copies and rollback on failed deploy`, `Data directory and offline backup procedure`, `Private .env configuration`, `SPD_EXTERNOS and SPD_BORRA_EXPEDIENTE roles` (+49 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 564 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **11 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Work-memory lessons

**Preferred sources** — corroborated by past sessions; start here.
- `classifyPDF()` (2× useful, score=1.990308954)
- `matchClassification()` (2× useful, score=1.990308954)
- `N027: Fusión de PDFs [REQUERIDO cronológica automática; PARCIAL manual; NO VERIFICADO]` (2× useful, score=1.98764134) _(code changed — re-verify)_

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `stagedJob` connect `stagedJob` to `oracle_documents.go`, `Server Core Handlers`, `objeciones.go`, `Coverage Records & Indexing`, `planilla_folder_link_test.go`, `Workspace Rename & ZIP`, `testing.T`, `reemplazo_coberturas.go`, `ingesta.go`, `Buscar por Planilla`, `go_pkg_path_filepath`, `objeciones_sync.go`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._
- **Are the 52 inferred relationships involving `packageFolderName()` (e.g. with `TestMergedDuplicateCountsOnlyWhenTheMergedFileDoesNotListItsSheets()` and `TestOnlyTheLatestRecordOfAPathCountsAsSavedSheet()`) actually correct?**
  _`packageFolderName()` has 52 INFERRED edges - model-reasoned connections that need verification._
- **What connects `folio`, `Recovery copies and rollback on failed deploy`, `Data directory and offline backup procedure` to the rest of the system?**
  _54 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `App.vue Main UI State` be split into smaller, more focused modules?**
  _Cohesion score 0.005420054200542005 - nodes in this community are weakly interconnected._
- **Why does `server` connect `Server Core Handlers` to `objeciones.go`, `Coverage Records & Indexing`, `Demo Lot & Coverage Marking`, `Historical Archive Tests`, `Workspace Rename & ZIP`, `testing.T`, `reemplazo_coberturas.go`, `stagedJob`, `ingesta.go`, `Buscar por Planilla`, `objeciones_sync.go`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._

### Low-confidence Hints
_AMBIGUOUS edges — the extractor was unsure. Verify before acting on these._

- **What is the exact relationship between `Not yet integrated: duplicate merge validation, full required-doc validation, partial upload, Oracle digitization state` and `Fusion de PDFs parcial (manual)`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Tie broken by longest specific phrase (conflicts with current rule: equal counts mean pending)` and `Classification precedence: filename alias, header phrases, priority phrases, regular rules`?**
  _Edge tagged AMBIGUOUS (relation: references) - confidence is low._