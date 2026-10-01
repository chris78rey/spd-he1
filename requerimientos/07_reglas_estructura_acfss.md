# Reglas funcionales de estructura ACFSS

**Estado:** reglas de negocio confirmadas por el usuario el 2026-09-30. Este documento es la referencia funcional del árbol de salida para futuras implementaciones. No afirma por sí mismo que cada detalle haya sido contrastado con una publicación normativa vigente del MSP/DPSP.

Las reglas detalladas de clasificación OCR, coincidencias, formularios obligatorios y fixtures están en [08_reglas_ocr_y_checklist.md](08_reglas_ocr_y_checklist.md); ese documento amplía y prevalece sobre las listas resumidas de formularios de esta página.

## 1. Tipos de servicio

El sistema debe poder identificar estos tipos de prestación:

1. Hospitalización / Internación / Hospital del Día.
2. Emergencia.
3. Ambulatorio / Laboratorio Clínico.
4. Ambulatorio / Procedimientos.
5. Ambulatorio / Consulta Externa.
6. Hemodiálisis.
7. Diálisis Peritoneal.
8. Componentes Sanguíneos.
9. Transporte Sanitario (prehospitalario y secundario: terrestre, aéreo o acuático).
10. Trasplante.
11. Coberturas Compartidas.

Los valores canónicos enviados por la interfaz y usados en nombres de carpeta/matriz son:

| Opción visible | Token canónico |
| --- | --- |
| Hospitalización / Internación / Hospital del Día | `HOSPITALIZACION` |
| Emergencia | `EMERGENCIA` |
| Ambulatorio / Laboratorio Clínico | `AMBULATORIO_LABORATORIO_CLINICO` |
| Ambulatorio / Procedimientos | `AMBULATORIO_PROCEDIMIENTOS` |
| Ambulatorio / Consulta Externa | `AMBULATORIO_CONSULTA_EXTERNA` |
| Hemodiálisis | `HEMODIALISIS` |
| Diálisis Peritoneal | `DIALISIS_PERITONEAL` |
| Componentes Sanguíneos | `COMPONENTES_SANGUINEOS` |
| Transporte Sanitario | `TRANSPORTE_SANITARIO` |
| Trasplante | `TRASPLANTE` |
| Coberturas Compartidas | `COBERTURAS_COMPARTIDAS` |

No inventar formularios obligatorios para servicios cuya lista no esté definida.

## 2. Identidad y nombres

- El ZIP contiene carpetas numéricas identificadas por `PDI_TRAMITE` (ejemplo: `9999999/`).
- Para el mes y año seleccionados, Go resuelve `PDI_TRAMITE` a `PDI_PACIENTE` mediante Oracle y filtra `PDI_PLANILLADO = 'S'`.
- El directorio del paciente se construye únicamente con `PDI_PACIENTE`: mayúsculas, puntuación inicial eliminada, diacríticos convertidos a ASCII (`Ñ` → `N`) y espacios sustituidos por `_`.
- Ruta: `4. EXPEDIENTES/<PDI_PACIENTE_NORMALIZADO>/`.
- No agregar `PDI_TRAMITE`, cédula, pasaporte, prefijos u otros datos al nombre de la carpeta. No agregar marcadores artificiales para pacientes con un solo apellido.
- Si falta la identidad, el nombre está vacío o hay colisión tras normalizar, informar el caso para revisión; no inventar ni fusionar identidades silenciosamente.

## 3. Primer ingreso

La carpeta madre usa el formato `[TIPO_DE_SERVICIO]_[MES]_[AÑO]/`, por ejemplo `EMERGENCIA_AGOSTO_2026/` o `HOSPITALIZACION_AGOSTO_2026/`.

```text
[TIPO_DE_SERVICIO]_[MES]_[AÑO]/
├── 1. OFICIO DE PAGO.pdf
├── 2. PLANILLA CONSOLIDADA.pdf
├── 3. MATRIZ_[TIPO_SERVICIO]_[MES_AÑO].xlsm
└── 4. EXPEDIENTES/
    └── <PDI_PACIENTE_NORMALIZADO>/
```

| Posición | Nombre en paquete | Código documental |
| --- | --- | --- |
| Oficio | `1. OFICIO DE PAGO.pdf` | `M_PAGO.pdf` |
| Consolidada | `2. PLANILLA CONSOLIDADA.pdf` | `C_CONSOLIDADA.pdf` para primer ingreso |
| Matriz | `3. MATRIZ_[TIPO_SERVICIO]_[MES_AÑO].xlsm` | Matriz oficial habilitada para macros |

El nombre original de carga no determina el rol: la salida usa el nombre canónico. El cierre exige los tres habilitantes.

Los archivos se colocan dentro de la carpeta madre nombrada por servicio y mes prestacional; el identificador `JOB-...` es solo interno y no debe ser la raíz del paquete que se comprime o entrega. **Todo documento PDF del paquete debe llevar la extensión explícita `.pdf`** (incluidos los códigos de clasificación, por ejemplo `HCU_008.pdf`). La extensión se conserva en minúsculas aunque el nombre de entrada use `.PDF` o no tenga extensión al renombrarse por clasificación.

### Documentos del expediente

Comunes obligatorios para cada paciente: `P_INDIVIDUAL.pdf` y `C_COBERTURA.pdf`. Para Red Privada se agregan `A_ENTREGA.pdf` y `C_VALIDACION.pdf` como obligatorios.

| Tipo de servicio | Formularios conocidos | Condición indicada |
| --- | --- | --- |
| Hospitalización / Internación | `HCU_006.pdf` (epicrisis), `HCU_053.pdf` (referencia/derivación), `HCU_017.pdf` (protocolo quirúrgico), `HCU_018A.pdf` (transanestésico), `HCU_007.pdf` (interconsultas), `HCU_113.pdf`, `HCU_114.pdf`, `HCU_115.pdf` (UCI) | `HCU_006` es clave; otros según cirugía, derivación, interconsulta o UCI. |
| Emergencia | `HCU_008.pdf`, `HCU_053.pdf` | `HCU_008` corresponde a emergencia; `HCU_053` si hubo derivación. |
| Ambulatorio / Laboratorio Clínico | `HCU_010.pdf` (010A/010B), `HCU_013.pdf`, `M_MULTIPLES.pdf` | `HCU_013` si aplica; `M_MULTIPLES` para muestras múltiples. |
| Ambulatorio / Procedimientos | `I_PROCEDIMIENTO.pdf`, `HCU_018A.pdf`, `HCU_053.pdf` | Según procedimiento, anestesia o derivación. |
| Ambulatorio / Consulta Externa | `HCU_002.pdf`, `T_REHABILITACION.pdf` | Rehabilitación si aplica. |
| Hemodiálisis | `R_DIARIO D.pdf`, `R_ASISTENCIA D.pdf`, `R_EXAMENES D.pdf`, `I_TRIMESTRAL D.pdf`, `I_SOCIAL D.pdf` | Reportes diarios, mensuales, trimestrales o semestrales según el documento. |
| Diálisis Peritoneal | `R.VISITA D.pdf`, `I_CAPACITACION D.pdf`, `D_JURADA.pdf` | Visita mensual, capacitación semestral y declaración anual según corresponda. |
| Componentes Sanguíneos | Pendiente de especificación | No inferir formularios. |
| Transporte Sanitario | Pendiente de especificación | No inferir formularios. |
| Trasplante | Pendiente de especificación | No inferir formularios. |
| Coberturas Compartidas | Pendiente de especificación | No inferir formularios. |

Los documentos condicionales se validan según datos y reglas del servicio, no solo por la existencia del archivo. Ante clasificación ambigua, enviar a revisión.

## 4. Levantamiento de objeciones

La carpeta madre se nombra `[TIPO_DE_SERVICIO]_[MES]_[AÑO]_OBJECIONES/`.

```text
[TIPO_DE_SERVICIO]_[MES]_[AÑO]_OBJECIONES/
├── 1. OFICIO DE PAGO.pdf
├── 2. PLANILLA CONSOLIDADA.pdf
├── 3. MATRIZ_OBJECIONES_[TIPO_SERVICIO]_[MES_AÑO].xlsm
├── 4. EXPEDIENTES/
│   └── <PACIENTE_OBJETADO_NORMALIZADO>/
│       ├── P_INDIVIDUAL.pdf
│       └── <formularios corregidos o ratificados>
└── 5. ANEXOS/
    └── <PACIENTE_OBJETADO_NORMALIZADO>/
        └── <respaldos de objeción>.pdf
```

- `4. EXPEDIENTES/` y `5. ANEXOS/` incluyen únicamente pacientes objetados.
- `P_INDIVIDUAL.pdf` indica expresamente si se acepta o rechaza cada objeción.
- La consolidada de respuesta usa código `P_CONSOLIDADA.pdf`; el nombre externo de posición sigue siendo `2. PLANILLA CONSOLIDADA.pdf`.
- `5. ANEXOS/` contiene sustentos como facturas, fichas técnicas y protocolos.
- No incluir pacientes aprobados ni sus anexos en este paquete.

## 5. Guardián de cierre

Antes de producir el ZIP/ISO de entrega, validar nombre de carpeta madre, servicio y período; los tres habilitantes; los nombres de paciente derivados de Oracle; documentos comunes y formularios aplicables; pacientes/anexos restringidos a objetados en el flujo de objeciones; y ausencia de archivos sin clasificar o conflictos de identidad. Si falla una condición, no presentar el paquete como final y detallar qué debe corregirse.

## 6. Estado de implementación

Estas reglas describen el resultado deseado. La ingesta recibe los archivos y los guarda en staging. El procesamiento inicial prepara una carpeta madre nombrada por servicio/período, copia los habilitantes con nombres canónicos y crea carpetas de paciente desde Oracle; aún no clasifica formularios, no valida el checklist por servicio y no implementa objeciones ni anexos. Mantener explícita esta diferencia en la interfaz y al informar avances.
