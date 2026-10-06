---
type: "query"
date: "2026-10-05T19:31:47.443625+00:00"
question: "¿Cuál es el estado real de implementación y verificación de SPD MSP / Folio?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["SPD MSP", "Clasificador híbrido", ".mergeWorkspacePDFs()", "5. Guardián de cierre"]
---

# Q: ¿Cuál es el estado real de implementación y verificación de SPD MSP / Folio?

## Answer

# Estado real de implementación: SPD MSP / Folio

> Registro aportado por el usuario para conservar en Graphify. Mantiene los IDs N001–N043 y separa requisitos, código y evidencia de ejecución. Este texto es una evaluación de estado basada en el análisis del usuario y en referencias a `repomix-output`; las referencias no se re-auditaron al registrar esta nota.

## Criterio de evidencia

- **REQUERIDO**: definido por reglas/requisitos del repositorio.
- **IMPLEMENTADO**: existe código concreto que realiza el comportamiento.
- **PARCIAL**: hay código, pero no satisface por completo el requisito documentado.
- **NO IMPLEMENTADO**: está requerido y figura pendiente o no aparece integrado en el código actual.
- **VERIFICADO**: hay evidencia de ejecución/prueba, no solo código.
- **NO VERIFICADO**: existe implementación, pero no evidencia funcional suficiente.
- **CONFLICTO**: el código y la documentación/estado histórico discrepan.

Que exista código demuestra que algo está programado, no que funcione correctamente en producción. La evidencia explícita de ejecución identificada en el análisis es: compilación correcta de backend y frontend, y respuesta HTTP 200 de `folio.service`; varias pruebas funcionales seguían pendientes. No se halló una batería `go test` ni evidencia suficiente para declarar completos los flujos de negocio.

## Nodos N001–N043

| ID | Nodo | Requisito | Código | Verificación | Conclusión |
|---|---|---|---|---|---|
| N001 | SPD MSP / Folio | REQUERIDO | IMPLEMENTADO | VERIFICADO básica | Sistema existente |
| N002 | ACFSS | REQUERIDO | — | — | Dominio del sistema |
| N003 | Frontend Vue/Vuetify | REQUERIDO | IMPLEMENTADO | VERIFICADO build | Implementado |
| N004 | Backend Go | REQUERIDO | IMPLEMENTADO | VERIFICADO build + HTTP 200 | Implementado |
| N005 | `PLANILLA_DIGITAL` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Consultada por código |
| N006 | Autenticación Oracle | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementada |
| N007 | Rol `SPD_EXTERNOS` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N008 | Período + servicio | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N009 | Workspace del período | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N010 | ZIP clínico | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N011 | Previsualización de ingesta | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementada |
| N012 | `PDI_TRAMITE` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N013 | `PDI_PACIENTE` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N014 | Vinculación manual trámite | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementada |
| N015 | Clasificador híbrido | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N016 | `reglas_clasificacion.yaml` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N017 | Extracción vectorial | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementada |
| N018 | OCR Tesseract | REQUERIDO | IMPLEMENTADO | NO VERIFICADO ambiental | Depende del SO |
| N019 | Alertas de fecha | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementadas |
| N020 | `PENDIENTE_tmp_*` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N021 | Catálogo MSP | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N022 | Expediente del paciente | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N023 | `fuentes/` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N024 | `trabajo/` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N025 | Reportes clasificación | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N026 | Detección de duplicados | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementada |
| N027 | Fusión de PDFs | REQUERIDO automática/cronológica | PARCIAL manual | NO VERIFICADO | Parcial respecto al requisito |
| N028 | Sincronización Oracle documental | REQUERIDO | IMPLEMENTADO parcialmente | NO VERIFICADO | Parcialmente implementada |
| N029 | `PDI_DOCUMENTO_DIGITAL` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Integrada |
| N030 | `PDI_PATH` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Integrado |
| N031 | Coberturas | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementadas |
| N032 | Portal cobertura MSP | REQUERIDO | Cliente implementado | NO VERIFICADO externo | Dependencia externa |
| N033 | `C_COBERTURA.pdf` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Implementado |
| N034 | `PDI_COBERTURA` | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | Escritura Oracle programada |
| N035 | Tres habilitantes cabecera | REQUERIDO obligatorios | PARCIAL | NO VERIFICADO | No bloquean toda la ingesta actual |
| N036 | Primer ingreso | REQUERIDO | PARCIAL | NO VERIFICADO | Estructura sí; cierre integral no |
| N037 | Objeciones | REQUERIDO | NO IMPLEMENTADO | — | No integrado |
| N038 | Guardián de cierre | REQUERIDO | NO IMPLEMENTADO | — | No integrado |
| N039 | `PDI_ESTADO_DIGITALIZACION` | REQUERIDO | NO IMPLEMENTADO/PARCIAL | — | No integrado como flujo completo |
| N040 | Biblioteca IndexedDB | REQUERIDO de producto | IMPLEMENTADO | NO VERIFICADO | Subsistema separado |
| N041 | Graphify | — | Artefacto implementado | — | Herramienta de análisis |
| N042 | Oficio MSP citado | REQUERIDO referencia | — | Fuente no disponible | Norma externa no verificada |
| N043 | Exportación ZIP | REQUERIDO | IMPLEMENTADO | NO VERIFICADO | No equivale al cierre oficial |

El clasificador híbrido se describe directamente en código: `classifyPDF()` intenta extraer texto del PDF, pasa a OCR si no hay texto, aplica reglas YAML, produce `PENDIENTE_tmp_*` ante fallo o falta de coincidencia y calcula alertas de fechas. Está implementado según el análisis; no está demostrado E2E.

## Relaciones R001–R040

| ID | Relación | Estado | Interpretación |
|---|---|---|---|
| R001 | Folio → AUTOMATIZA → ACFSS | REQUERIDO/IMPLEMENTADO | Finalidad general parcial |
| R002 | Folio → USA → Vue/Vuetify | IMPLEMENTADO | Código existente |
| R003 | Folio → USA → Go | IMPLEMENTADO | Código existente |
| R004 | Backend → CONSULTA → `PLANILLA_DIGITAL` | IMPLEMENTADO | SQL implementado |
| R005 | Login → REQUIERE → `SPD_EXTERNOS` | IMPLEMENTADO | Control presente |
| R007 | Período + servicio → IDENTIFICA → workspace | IMPLEMENTADO | — |
| R012 | Previsualización → INSPECCIONA → ZIP | IMPLEMENTADO | — |
| R013 | Previsualización → CONTRASTA_CON → Oracle | IMPLEMENTADO | — |
| R014 | Vínculo manual → CORRIGE → trámite | IMPLEMENTADO | — |
| R017 | Clasificador → USA → YAML | IMPLEMENTADO | — |
| R018 | Clasificador → PRIORIZA → vectorial | IMPLEMENTADO | — |
| R019 | Clasificador → FALLBACK → OCR | IMPLEMENTADO | — |
| R020 | Clasificador → GENERA → pendiente | IMPLEMENTADO | — |
| R021 | Clasificador → DETECTA → fechas discordantes | IMPLEMENTADO | — |
| R022 | Catálogo → RESTRINGE → renombrado | IMPLEMENTADO | — |
| R024 | Expediente → PUEDE_CONTENER → duplicados | IMPLEMENTADO | Detectado por código |
| R025 | Fusión → RESUELVE → duplicados | PARCIAL | Manual; no cronológica automática |
| R027 | Duplicados → BLOQUEAN → ZIP | IMPLEMENTADO | — |
| R028 | Workspace → SINCRONIZA → documentos Oracle | IMPLEMENTADO | — |
| R033 | Cobertura → GENERA → `C_COBERTURA` | IMPLEMENTADO | — |
| R035 | Cobertura → ACTUALIZA → `PDI_COBERTURA` | IMPLEMENTADO | SQL implementado |
| R037 | Guardián → VALIDA → habilitantes | NO IMPLEMENTADO | Requerido, no integrado |
| R038 | Guardián → VALIDA → estado digitalización | NO IMPLEMENTADO | Requerido, no integrado |
| R039 | Guardián → PRODUCE → paquete final | NO IMPLEMENTADO | Requerido, no integrado |
| R040 | Guardián → PRODUCE → objeciones | NO IMPLEMENTADO | Requerido, no integrado |

Las rutas del servidor citadas incluyen coberturas, ingesta, clasificación, previsualización, workspaces, renombrado, fusión, visor y ZIP. El análisis no encuentra APIs `/objeciones/...` ni `/empaquetado/auditar`, `/empaquetado/generar`, aunque aparecen en documentos de diseño. Revalidar contra el código actual.

## Clústeres funcionales

1. **Acceso y arquitectura — implementado, verificación básica.** Se describen rutas login/sesión/logout; `login()` conecta a Oracle, comprueba `SPD_EXTERNOS` y crea la sesión. Build + HTTP 200 no demuestran un E2E de login.
2. **Ingesta y cruce Oracle — implementado, no E2E verificado.** Se describen validación del ZIP, carpetas/trámites, PDFs y encabezado `%PDF-`, cruce Oracle y mappings manuales. El ZIP define el universo operativo: planillas solo en Oracle se muestran ausentes y no entran automáticamente al procesamiento.
3. **Clasificación vectorial + OCR — implementado, no E2E verificado.** El análisis cita `pdftoppm`, `tesseract`, primera página a 250 DPI, idioma `spa`, reglas YAML y pendientes.
4. **Gestión documental — mayormente implementada.** Ver, renombrar, añadir/reemplazar/quitar, detectar duplicados, fusionar y sincronizar Oracle. Renombrado validado contra catálogo y con sufijos para evitar sobrescritura.
5. **Fusión — parcial frente al requisito.** Se citan `mergeWorkspacePDFs` y `/api/v1/expedientes/documentos/fusionar/`; el usuario escoge orden en vez de fusión cronológica automática. El análisis también indica que exige todos los archivos del grupo, restringe paciente/tipo, conserva fuentes, genera PDF canónico y bloquea descarga con grupos pendientes.
6. **Coberturas — SQL implementado, ejecución no verificada.** Se cita `UPDATE ... SET PDI_COBERTURA = 'S'`, comprobando que se actualice exactamente una fila.
7. **Estado de digitalización — no integrado como flujo completo.** Diseño propone `PENDIENTE → PROCESANDO → REQUIERE_VALIDACION → INCOMPLETO → COMPLETO → OBJETADO`; documento operativo citado deja pendiente escribir `PDI_ESTADO_DIGITALIZACION` hasta confirmar esquema y política.
8. **Checklist automático — no implementado** según estado declarado en `08_reglas_ocr_y_checklist.md`. Reconocer un PDF, por ejemplo `HCU_008.pdf`, no prueba que el expediente contenga automáticamente todo lo requerido.
9. **Objeciones — no integrado.** Diseño describe `[SERVICIO]_[MES]_[AÑO]_OBJECIONES/`, `4. EXPEDIENTES/`, `5. ANEXOS/`, `P_INDIVIDUAL.pdf` con ACEPTA/RECHAZA y solo pacientes objetados; no hay API/flujo equivalente según el análisis.
10. **Guardián de cierre — no implementado integralmente.** La especificación pide tres habilitantes, cero `PENDIENTE_`, 100 % expedientes COMPLETO, formato/resolución/estructura, objeciones y paquete final. `06req_motor.cierre.md` es una especificación, no descripción del backend integrado.

## Conflictos CFL-01–CFL-08

| ID | Conflicto | Conclusión |
|---|---|---|
| CFL-01 | Documento dice fusión pendiente; handler actual tiene fusión | El código prevalece para lo programado; cronología automática sigue sin demostrarse |
| CFL-02 | Requisito cronológico; implementación usa orden humano | Parcial |
| CFL-03 | SRS propone actualizar `PDI_ESTADO_DIGITALIZACION`; estado operativo dice no hacerlo aún | Pendiente hasta confirmar esquema/política |
| CFL-04 | Guardián describe endpoints ausentes en `main.go` | No integrado |
| CFL-05 | Diseño de objeciones sin flujo en rutas | No integrado |
| CFL-06 | SRS requiere habilitantes al ingreso; flujo permite trabajar antes del cierre | Flujo distinto/parcial |
| CFL-07 | ZIP llamado “final” puede originarse en workspace `INCOMPLETE` | No equivale a paquete formal |
| CFL-08 | Oficio MSP citado pero no incluido | Norma externa no verificada |

CFL-01 se apoya en el contraste descrito entre un registro operativo fechado el 30 de septiembre (fusión cronológica pendiente) y la presencia posterior de `mergeWorkspacePDFs` y la ruta de fusión. Conclusión: fusión manual general programada; fusión cronológica automática no demostrada.

## Mapa global y conclusión

```text
SPD MSP / FOLIO
├─ Ingesta → Oracle → clasificación vectorial/OCR: IMPLEMENTADO según código
├─ Expediente → renombrado / cobertura / duplicados: IMPLEMENTADO según código
├─ Duplicados → fusión manual → ZIP workspace: PARCIAL / IMPLEMENTADO
└─ Frontera pendiente del cierre formal
   ├─ Checklist completo: NO IMPLEMENTADO
   ├─ Objeciones: NO IMPLEMENTADO
   ├─ Estado de digitalización: NO INTEGRADO
   └─ Guardián de cierre → paquete formal/ISO: NO IMPLEMENTADO
```

El núcleo programado que identifica el análisis es: autenticación, consulta Oracle, workspace, inspección ZIP, vínculo trámite/paciente, clasificación vectorial/OCR, pendientes, edición documental, detección y fusión manual de duplicados, sincronización documental Oracle, coberturas y exportación ZIP.

No inferir implementación completa solo desde documentos de diseño para: checklist integral, flujo completo de `PDI_ESTADO_DIGITALIZACION`, subsanación basada en ese estado, objeciones completas, Guardián de Cierre, validación técnica integral, ISO/rotulado y paquete formal definitivo. Niveles: REQUERIDO = lo deseado; IMPLEMENTADO = código programado; VERIFICADO = ejecución probada bajo condiciones explícitas.

## Outcome

- Signal: useful

## Source Nodes

- SPD MSP
- Clasificador híbrido
- .mergeWorkspacePDFs()
- 5. Guardián de cierre