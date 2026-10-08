# Reglas OCR, clasificación y checklist ACFSS

**Estado:** reglas operativas aportadas por el usuario a partir de la respuesta del experto. Se indicó como referencia el Oficio Nro. MSP-DP17-2026-6548-O; el documento fuente no está en este repositorio, por lo que la referencia aún no se ha contrastado aquí.

## 1. Entrada, extracción y clasificación

- Ignorar el nombre del PDF como señal general: puede ser aleatorio (`tmp_a1.pdf`, `doc_99.pdf`). Excepción confirmada: las bases exactas `08` y `008` identifican `HCU_008.pdf`, y `007` identifica `HCU_007.pdf`; se admite un sufijo numérico de duplicado (`007_2.pdf`, `08_1.pdf`) y el prefijo generado `PENDIENTE_tmp_`. Estos códigos de archivo prevalecen sobre coincidencias genéricas del contenido. Para nombres no confirmados, clasificar por el texto del PDF y conservar como pendiente las coincidencias ambiguas.
- Analizar primero el texto vectorial de las páginas 1 y 2.
- Si no hay texto digital utilizable, renderizar la página 1 en escala de grises a 200–300 DPI y aplicar Tesseract en español (`spa`).
- Texto vectorial con coincidencia exacta de una regla: confianza operativa 100% según el criterio recibido; no representa una probabilidad calibrada estadísticamente.
- En OCR, una coincidencia positiva de palabra/frase del catálogo habilita clasificación. Si varias reglas coinciden, gana la de mayor cantidad de frases exactas coincidentes en el encabezado.
- Si hay empate entre reglas, coincidencia parcial, buffer vacío o ilegible, conservar el archivo para revisión; el lote continúa.
- Nombre pendiente: `PENDIENTE_tmp_[nombre_original].pdf`, saneando el nombre para que sea seguro como archivo. Estado Oracle previsto: `PDI_ESTADO_DIGITALIZACION='REQUIERE_VALIDACION'`.
- Normalizar texto de comparación a mayúsculas y ASCII para comparar tildes con tolerancia. Fuera de las bases numéricas confirmadas arriba, el nombre de entrada no participa como señal.
- Toda salida PDF lleva `.pdf` en minúsculas.
- Los alias de entrada/salida `HCU_53` y `HCU_053` se normalizan al nombre canónico `HCU_053.pdf`.

### Comparación de fechas clínicas con Oracle

- Relacionar cada carpeta numérica con su fila por `PDI_TRAMITE`, usando el conjunto filtrado por `PDI_MES`, `PDI_ANIO`, `PDI_PLANILLADO = 'S'` y `PDI_ASEGURADORA = 'MSP'`.
- Para la alerta de fechas del PDF, comparar el mes y año reconocidos con el período facturado Oracle `PDI_MES/PDI_ANIO` de la fila del lote. `PDI_FECHA_DESDE/HASTA` describen la atención clínica y se pueden mostrar como referencia, pero no determinan esta alerta.
- Durante la clasificación, buscar fechas reconocibles junto a campos de fecha del formulario en el texto vectorial de las páginas 1 y 2 o en el texto OCR de la página 1. Se aceptan `DD/MM/YYYY`, `DD-MM-YYYY`, `YYYY-MM-DD` y fechas en español con el nombre del mes. No tomar fechas sueltas de la narrativa clínica como fecha del documento.
- Omitir fechas de datos personales, generación o vencimiento: `NACIMIENTO`, `FECHA DE PROCESO`, `FECHA PLANILLACIÓN`, `PLANILLAJE AUTOMÁTICO`, `GENERADO AUTOMÁTICAMENTE`, `FECHA MÁXIMA`, `FECHA LÍMITE` y `VENCIMIENTO`. Las fechas de campos del formulario fuera del período facturado se muestran asociadas al PDF como alerta para revisión humana. No bloquean, renombran ni eliminan el documento.
- Si el texto no permite reconocer fechas, no afirmar que pasó la comprobación. Las fechas que Folio reconoce son indicativas; hay que revisar el PDF original antes de resolver la alerta.
- La lectura de fechas por OCR es indicativa, no una verificación clínica: revisar el PDF original antes de resolver la alerta. Los documentos añadidos manualmente después de la clasificación no están cubiertos por esta comprobación automática.

**Implementación:** la consulta Oracle filtra las planillas por `PDI_MES/PDI_ANIO`, `PDI_PLANILLADO = 'S'` y `PDI_ASEGURADORA = 'MSP'`. Las fechas extraídas del texto PDF u OCR se comparan con ese mismo mes/año. La UI lista los documentos y fechas fuera del período como alerta de revisión; no bloquea ni modifica la preparación.

### Vista previa del lote antes de preparar

- Después de guardar el ZIP en staging y antes de ejecutar OCR/organización, mostrar el total de carpetas de trámites y PDFs.
- Por carpeta numérica mostrar `PDI_TRAMITE`, cantidad de PDFs, paciente, `PDI_SERVICIO`, `PDI_FECHA_DESDE/HASTA` y si hubo cruce con una planilla de Oracle filtrada por `PDI_MES`, `PDI_ANIO`, `PDI_PLANILLADO = 'S'` y `PDI_ASEGURADORA = 'MSP'`. El aviso de fechas en los PDFs compara con el período `PDI_MES/PDI_ANIO`, no con la ventana clínica.
- El ZIP define los trámites incluidos en el lote. Comparar en ambas direcciones: listar carpetas del ZIP sin planilla MSP coincidente en Oracle y planillas MSP del período sin carpeta en el ZIP. La ausencia de una carpeta en el ZIP se informa para revisión y no añade esa planilla al lote.
- Para una carpeta del ZIP sin coincidencia exacta, permitir asociarla manualmente a un `PDI_TRAMITE` Oracle MSP disponible que no tenga otra carpeta asignada. Guardar la asociación con el lote, conservar intacto el ZIP fuente y mostrar ambos números de trámite. La asociación se aplica al preparar o reanalizar los expedientes.
- Contabilizar y señalar carpetas no encontradas en Oracle, trámites sin nombre de paciente y rutas/archivos que no siguen `[PDI_TRAMITE]/archivo.pdf`. La tabla se pagina de diez en diez.
- La vista previa inspecciona el ZIP ya subido y no vuelve a transferirlo ni ejecuta OCR. No descarta registros por servicio o fecha; expone esos campos para revisión. Solo habilita preparar cuando hay PDFs y no hay carpetas/trámites sin correspondencia ni entradas incompatibles con la estructura aceptada.

**Implementación:** `GET /api/v1/ingesta/previsualizar/{id}` inspecciona el ZIP guardado y consulta Oracle. La interfaz permite recorrer las carpetas antes de preparar.

## 2. Catálogo de señales confirmadas

La fuente ejecutable del catálogo es [`../reglas_clasificacion.yaml`](../reglas_clasificacion.yaml). Las frases de cada regla se cuentan como coincidencias exactas distintas; varias apariciones de la misma frase cuentan una sola vez. Gana la regla con más frases coincidentes; si hay empate, gana la que tenga mayor especificidad total de frases (longitud normalizada). Si también empata la especificidad, el documento queda pendiente de revisión.

| Código canónico | Señales discriminatorias y patrones OCR confirmados |
| --- | --- |
| `HCU_008.pdf` | `FORMULARIO 008`, `HCU FORM.008` (señales específicas; no usar `EMERGENCIA` ni `MOTIVO DE CONSULTA` por sí solas) |
| `HCU_053.pdf` | `FORMULARIO 053`, `REFERENCIA`, `DERIVACION`, `CONTRARREFERENCIA`; discriminador adicional `ESTABLECIMIENTO QUE DERIVA` |
| `HCU_006.pdf` | `FORMULARIO 006`, `EPICRISIS`, `RESUMEN DE ALTA`, `CUADRO CLINICO DE EGRESO` |
| `HCU_017.pdf` | `FORMULARIO 017`, `PROTOCOLO QUIRURGICO`, `CIRUGIA` |
| `HCU_007.pdf` | `FORMULARIO 7 – INTERCONSULTA` (señal prioritaria frente a palabras genéricas como `CIRUGIA`) |
| `HCU_018A.pdf` | `FORMULARIO 018A`, `TRANSANESTESICO`, `ANESTESIA` |
| `HCU_012.pdf` | `FORMULARIO 12 IMAGENOLOGIA`, `RESULTADOS EXAMENES DE IMAGEN` |
| `HCU_010.pdf` | `FORMULARIO 10 LABORATORIO CLINICO`, `RESULTADOS EXAMENES` |
| `HCU_013.pdf` | `FORMULARIO 13`; `FORMULARIO 10 PATOLOGIA`, `RESULTADOS PAPANICOLAOUS`, `RESULTADOS PAP`, `ESTUDIO CITOLOGICO` (resultado de citología / 013B observado en expediente) |
| `C_COBERTURA.pdf` | `COBERTURA DE SALUD`, `CERTIFICADO DE AFILIACION`, `COMPROBANTE DE DERECHO` |
| `P_INDIVIDUAL.pdf` | `PLANILLA INDIVIDUAL`, `VALOR FACTURADO`, `LIQUIDACION INDIVIDUAL` |
| `A_ENTREGA.pdf` | `ACTA DE ENTREGA`, `RECEPCION DEL SERVICIO` |
| `I_PROCEDIMIENTO.pdf` | `PROCEDIMIENTO MENOR` |

La separación HCU 006 / HCU 053 se basa especialmente en EPICRISIS, RESUMEN DE ALTA y CUADRO CLINICO DE EGRESO frente a REFERENCIA, DERIVACION, CONTRARREFERENCIA y ESTABLECIMIENTO QUE DERIVA.

Para `HCU_007.pdf`, la frase `FORMULARIO 7 – INTERCONSULTA` es una señal prioritaria de contenido. Para los demás códigos conocidos de nombres de archivo sin frases OCR confirmadas, enviarlos a revisión manual.

## 3. Catálogo de nombres de archivo conocidos

Estos nombres siempre incluyen la extensión PDF en minúsculas cuando son PDF. Esta lista no implica que todos tengan reglas automáticas OCR; solo los ocho del apartado 2 las tienen.

| Código de salida | Documento |
| --- | --- |
| `P_INDIVIDUAL.pdf` | Planilla individual firmada por el prestador |
| `C_COBERTURA.pdf` | Consulta de cobertura de salud al inicio y fin de atención |
| `A_ENTREGA.pdf` | Acta de entrega-recepción del servicio (RPC; puede acreditarse con firma del paciente/acompañante en planilla individual física según regla local) |
| `C_VALIDACION.pdf` | Código de validación interinstitucional (RPC) |
| `HCU_008.pdf` | Formulario 008 - Emergencia |
| `HCU_053.pdf` | Formulario 053 - Referencia, derivación o contrarreferencia; normaliza alias `HCU_53.pdf` |
| `HCU_006.pdf` | Formulario 006 - Epicrisis |
| `HCU_017.pdf` | Formulario 017 - Protocolo quirúrgico/operatorio |
| `HCU_018A.pdf` | Formulario 018A - Protocolo transanestésico |
| `HCU_007.pdf` | Formulario 007 - Interconsultas |
| `HCU_002.pdf` | Formulario 002 - Consulta externa |
| `HCU_010.pdf` | Formulario 010A solicitud / 010B resultados de laboratorio |
| `HCU_012.pdf` | Formulario 012A solicitud / 012B informe de imagenología |
| `HCU_013.pdf` | Formulario 013A solicitud / 013B informe de anatomía patológica |
| `HCU_113.pdf`, `HCU_114.pdf`, `HCU_115.pdf` | Registros UCI adulto, neonatal o pediátrico |
| `HCU_119.pdf` | Transporte secundario |
| `HCU_122.pdf` | Solicitud y despacho de componentes sanguíneos |
| `F_ANEXO 002.pdf` | Anexo 002 - Atención prehospitalaria |
| `H_RUTA.pdf` | Hoja de ruta de movilización de ambulancia/transporte aéreo |
| `I_PROCEDIMIENTO.pdf` | Informe detallado del procedimiento realizado |
| `T_REHABILITACION.pdf` | Pedido y registro de terapias de rehabilitación |
| `M_MULTIPLES.pdf` | Listado de muestras enviadas y procesadas |
| `R_DIARIO D.pdf` | Reporte diario de sesiones de hemodiálisis |
| `R_ASISTENCIA D.pdf` | Registro mensual de asistencia a hemodiálisis |
| `R_EXAMENES D.pdf` | Resultados de exámenes complementarios en diálisis |
| `R_CONSULTA D.pdf` | Registro de consulta externa/soporte clínico en hemodiálisis |
| `I_TRIMESTRAL D.pdf` | Informe trimestral de evolución clínica en hemodiálisis |
| `I_SOCIAL D.pdf` | Informe semestral de trabajo social |
| `R.VISITA D.pdf` | Registro mensual de visita domiciliaria en diálisis peritoneal |
| `I_CAPACITACION D.pdf` | Informe semestral de capacitación en diálisis peritoneal |
| `D_JURADA.pdf` | Declaración juramentada anual de no reutilización de insumos |
| `I_LIQUIDACION.pdf` | Informe de liquidación de auditoría previa |
| `B_VIAJE.pdf` | Respaldo de viaje acuático indicado para transporte sanitario; confirmar código oficial y señales OCR |

La matriz conserva `.xlsm`: `3. MATRIZ_[TIPO_DE_SERVICIO]_[MES]_[AÑO].xlsm`. Todo elemento del catálogo cuya extensión sea `.pdf` debe escribirse con esa extensión explícita y en minúsculas.

## 4. Matriz de documentos requeridos por servicio

| Servicio | Obligatorios siempre | Condicionales / criterio |
| --- | --- | --- |
| Hospitalización / Internación / Hospital del Día | `HCU_006.pdf`, `C_COBERTURA.pdf` | `HCU_053.pdf` si referencia/derivación; `HCU_017.pdf` en cirugía; `HCU_018A.pdf` si anestesia; `HCU_007.pdf` si interconsulta; `HCU_113/114/115.pdf` según UCI; `C_VALIDACION.pdf` para RPC. |
| Emergencia | `HCU_008.pdf`, `C_COBERTURA.pdf` (inicio y fin) | `HCU_053.pdf` si deriva desde otro centro; `C_VALIDACION.pdf` para RPC. |
| Ambulatorio / Laboratorio Clínico | `HCU_010.pdf` (010A/010B), `C_COBERTURA.pdf` | `HCU_053.pdf`; `HCU_013.pdf` si examen histopatológico; `M_MULTIPLES.pdf` en procesamiento masivo de muestras; `C_VALIDACION.pdf` para RPC. |
| Ambulatorio / Procedimientos | `I_PROCEDIMIENTO.pdf`, `C_COBERTURA.pdf` | `HCU_053.pdf`; `HCU_018A.pdf` si anestesia; `C_VALIDACION.pdf` para RPC. |
| Ambulatorio / Consulta Externa | `HCU_002.pdf`, `C_COBERTURA.pdf` | `HCU_053.pdf`; `T_REHABILITACION.pdf` en terapia/rehabilitación; `C_VALIDACION.pdf` para RPC. |
| Hemodiálisis | `R_DIARIO D.pdf`, `R_ASISTENCIA D.pdf`, `R_EXAMENES D.pdf`, `R_CONSULTA D.pdf`, `C_COBERTURA.pdf` | `I_TRIMESTRAL D.pdf` cada 3 meses; `I_SOCIAL D.pdf` semestral; `C_VALIDACION.pdf` para RPC. |
| Diálisis Peritoneal | `R.VISITA D.pdf`, `R_EXAMENES D.pdf`, `C_COBERTURA.pdf` | `I_CAPACITACION D.pdf` semestral; `D_JURADA.pdf` anual; `C_VALIDACION.pdf` para RPC. |
| Componentes Sanguíneos | `HCU_122.pdf`, `C_COBERTURA.pdf` | `C_VALIDACION.pdf` para RPC. |
| Transporte Sanitario | `F_ANEXO 002.pdf` en transporte primario o `HCU_119.pdf` en secundario; `H_RUTA.pdf`; `C_COBERTURA.pdf` | `HCU_053.pdf` en secundario excepto destino domicilio; `B_VIAJE.pdf` en transporte acuático; `C_VALIDACION.pdf` para RPC. `B_VIAJE.pdf` requiere confirmar su código/patrones porque no estaba en el catálogo inicial. |
| Trasplante | `HCU_006.pdf`, `HCU_053.pdf`, `C_COBERTURA.pdf` | `HCU_008.pdf` si pasó por Emergencia; `HCU_017.pdf`/`HCU_018A.pdf` si hubo cirugía; `C_VALIDACION.pdf` para RPC. |
| Coberturas Compartidas | `I_LIQUIDACION.pdf`, `M_PAGO.pdf`, `C_CONSOLIDADA.pdf`, `C_COBERTURA.pdf` | Informe de liquidación del subsistema que ejecutó la primera auditoría. Confirmar cómo debe ubicarse/clasificarse `I_LIQUIDACION.pdf` dentro del árbol final. |

Documentos comunes: `P_INDIVIDUAL.pdf` por expediente. `C_COBERTURA.pdf` es común obligatorio en todos los servicios. Para RPC, `C_VALIDACION.pdf` es obligatorio. `A_ENTREGA.pdf` aplica a RPC; si la planilla individual física tiene firma de paciente o acompañante, esa firma constituye el acta según la regla recibida.

## 5. Duplicados y documentos de varias páginas

- Si varios archivos de una misma carpeta numérica se reconocen con el mismo código, unirlos con `pdfcpu.Merge` en orden cronológico según fechas clínicas.
- La pertenencia se establece por carpeta numérica de entrada (`PDI_TRAMITE`) o por resolución a la misma identidad en la caché Oracle. Atenciones dobles del mes se consolidan en el PDF mensual.
- Si no se pueden extraer fechas o resolver pertenencia, dejar el caso pendiente en vez de asumir el orden.

## 6. Banco de pruebas anonimizado

Fixtures sintéticos deben usar nombres ficticios (`PRUEBA_PACIENTE_JUAN_CARLOS`), cédulas de prueba válidas módulo 10 cuando haga falta y HCU ficticia `999999`; no incluir datos clínicos ni nombres reales.

Preparar: PDF vectorial con texto seleccionable, PDF escaneado gris 200–300 DPI, PDF de baja calidad/blanco para derivación a pendiente, casos HCU 006/053 con palabras discriminatorias, formularios de los 8 patrones configurados, empate/múltiples coincidencias y documentos sin palabras clave. Los tests de zip que requieren cruce Oracle deben usar identificadores de una base de prueba; el corpus standalone de OCR no debe contener pacientes Oracle reales.

## 7. Objeciones y límites de implementación

En objeciones se excluyen pacientes aprobados, `P_INDIVIDUAL.pdf` contiene explícitamente `ACEPTA` o `RECHAZA`, `4. EXPEDIENTES/` contiene documentos corregidos/ratificados y `5. ANEXOS/` los sustentos exclusivos por paciente objetado. La salida conserva extensión PDF explícita.

Estado de implementación al 2026-09-30: el backend carga `reglas_clasificacion.yaml`, extrae texto vectorial de páginas 1 y 2 y resuelve coincidencias por cantidad de frases exactas distintas; empate o ausencia de señales se conserva como `PENDIENTE_tmp_*.pdf`. El OCR de respaldo se ejecuta con Poppler (`pdftoppm`) y Tesseract (`spa`) cuando están instalados; si faltan o no hay texto reconocible, el escaneado queda pendiente. Cada lote tiene un espacio fijo `FOLIO_DATA_DIR/expedientes/<JOB-ID>/`: `fuentes/`, `trabajo/`, `reportes/` y `job.json`. Reclasificar reemplaza atómicamente `trabajo/` y `reportes/` desde las fuentes originales; no crea otra carpeta de salida. Los lotes antiguos se migran copiando sus fuentes la primera vez que se procesan en esta versión. Siguen pendientes el checklist automático, la fusión cronológica de duplicados con `pdfcpu`, la carga parcial y las actualizaciones Oracle de `PDI_ESTADO_DIGITALIZACION`; no cambiar ese campo hasta confirmar el esquema y la política de escritura.

## 8. Añadir, renombrar y reemplazar PDFs desde la bandeja

- El catálogo compartido de nombres de expedientes está en [`../catalogos/codigos_msp.json`](../catalogos/codigos_msp.json). La interfaz ofrece ese catálogo al renombrar y, de forma independiente para cada archivo local, al añadir PDFs a una carpeta de paciente.
- El backend valida el código contra el mismo catálogo; no confía en que la validación de la interfaz sea suficiente. Todos los códigos de este catálogo terminan en `.pdf` minúsculas. La lista refleja los nombres aportados para el flujo; no constituye por sí sola una certificación normativa independiente.
- Si el código elegido ya está ocupado, el nuevo archivo recibe el primer sufijo libre (`HCU_008_1.pdf`, `HCU_008_2.pdf`, etc.) para no sobrescribir el existente. Esos duplicados siguen necesitando la consolidación posterior prevista en la sección 5.
- Al reemplazar, el archivo local sustituye el contenido de la ruta seleccionada y adopta su nombre existente. El nombre externo se conserva como referencia del original; el contenido anterior queda en `fuentes/`.
- Si varios PDFs de un paciente coinciden con el mismo código, el primero usa el nombre canónico y los demás reciben sufijos numéricos; quedan reconocidos y se marcan para revisión/fusión posterior. La interfaz permite ordenarlos y fusionarlos bajo el nombre canónico, conservando fuentes. No se permite descargar el ZIP final mientras haya grupos sin resolver.
- Esta asignación manual de código no inspecciona el contenido ni comprueba que coincida con el tipo documental elegido. Los PDFs añadidos manualmente después de clasificar tampoco reciben la comparación automática de fechas clínicas descrita en la sección 1.
