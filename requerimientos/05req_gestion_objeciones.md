
# Documento de Diseño Técnico

Las reglas vigentes de la carpeta madre y de inclusión exclusiva de pacientes objetados están definidas en [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md).

**Proyecto:** Módulo Automatizado de Ingesta, Clasificación Híbrida y Subsanación Manual para la ACFSS (DPSP - MSP)  
**Componente:** **Bandeja de Control Documental, Subsanación e Inyección Puntual**  
**Rol:** Arquitecto de Software & Tech Lead

---

### 1. Stack Tecnológico y Dependencias

Basado estrictamente en las fuentes del proyecto, el stack tecnológico aprobado para este módulo comprende:

- **Frontend (Interfaz de Usuario):**
    - **Framework UI:** **Vuetify** (Vue.js), para la pantalla de control documental, tablas de estado, filtros y visor dividido.
    - **Visualizador PDF:** Componente de renderizado nativo HTML5 (`<embed>` / `<iframe>`) o integración con `pdf.js` para la vista previa fluida del PDF no identificado en el panel izquierdo.
    - **Cliente HTTP:** **Axios**, para el envío de archivos individuales (`multipart/form-data`) y llamadas asíncronas REST.
- **Backend (Servicios de Subsanación):**
    - **Lenguaje:** **Go (Golang) 1.22+**, para el manejo de archivos en disco, renombrado normativo y manipulación de streams.
    - **Motor de Fusión de PDFs:** Biblioteca **`pdfcpu`** en Go, para la concatenación automática cuando se inyecta un documento que complementa a uno existente.
- **Base de Datos y Persistencia:**
    - **Motor BD:** **Oracle Database 11gR2** (vía driver `godror` / Oracle Instant Client).
    - **Estructura de Persistencia:** Tabla `DIGITALIZACION.PLANILLA_DIGITAL` y el índice optimizado `IDX_PDI_ESTADO` sobre `(PDI_ESTADO_DIGITALIZACION, PDI_OBJETADO)`.

---

### 2. Contexto y Flujo de Datos

#### **Problema que resuelve:**

Durante el procesamiento masivo, existen archivos escaneados que no pueden ser catalogados automáticamente por el motor OCR o el evaluador YAML debido a manchas en la imagen, baja legibilidad o tipografías atípicas. Asimismo, puede ocurrir que un expediente presente la omisión de un formulario obligatorio según su tipo de servicio (por ejemplo, ausencia del `HCU_008.pdf` en Emergencia).

Este módulo resuelve **la resiliencia operativa y la corrección puntual**:

1. **No interrumpe el procesamiento del lote mensual:** Aísla los archivos no reconocidos bajo la etiqueta `PENDIENTE_tmp_[nombre].pdf` y marca el registro como `REQUIERE_VALIDACION` o `INCOMPLETO`.
2. **Proporciona una interfaz visual de selección asistida (Visor Dividido):** Permite al operador examinar el PDF a la izquierda y asignarle el código normativo oficial del MSP desde un desplegable a la derecha con un solo clic.
3. **Permite la inyección directa de documentos faltantes:** Permite buscar a un paciente específico y subir únicamente el PDF que faltaba, el cual es renombrado e inyectado en su carpeta oficial sin necesidad de reprocesar el lote completo.

#### **Flujos de Entrada (Inputs):**

- **Archivos Aislados en Disco:** PDFs ubicados en `/workspace/4. EXPEDIENTES/[PACIENTE]/` nombrados con el prefijo `PENDIENTE_`.
- **Registros de Excepción desde Oracle DB:** Consultas filtradas por `PDI_ESTADO_DIGITALIZACION IN ('REQUIERE_VALIDACION', 'INCOMPLETO')`.
- **Cargas Puntuales del Usuario:** Subida individual de un PDF corregido desde la interfaz.

#### **Flujos de Salida (Outputs):**

- **Inyección y Renombrado en Disco:** Reemplazo o renombrado al código MSP oficial (`HCU_008.pdf`, `HCU_053.pdf`, `C_COBERTURA.pdf`, etc.) dentro de la subcarpeta del paciente.
- **Fusión de Documentos:** PDF unificado mediante `pdfcpu.Merge` si el archivo subido corresponde a una segunda hoja de un formulario existente.
- **Actualización en Oracle DB:** Sincronización del estado de digitalización a `COMPLETO` / `VALIDADO` y `PDI_PROCESADO = 'S'`.

---

### 3. Estructura de Directorios Sugerida

Sugerencia de arquitectura separada por capas entre Vuetify y Go:

```
subsanacion-control-documental/
├── frontend-vuetify/
│   ├── src/
│   │   ├── components/
│   │   │   ├── SplitPdfViewer.vue       # Visor dividido (PDF a la izq, Formulario a la der)
│   │   │   ├── MissingDocCard.vue       # Tarjeta de alerta con casillero rojo para documentos faltantes
│   │   │   └── SubsanacionFilter.vue    # Filtros por Cédula, Planilla, Nombre o Estado
│   │   ├── views/
│   │   │   ├── ControlDocumentalView.vue # Pantalla de revisión de PDFs no identificados
│   │   │   └── InyeccionFaltantesView.vue # Pantalla de inyección de formularios pendientes
│   │   └── services/
│   │       └── subsanacionService.js    # Cliente Axios para APIs de clasificación e inyección
├── backend-go/
│   ├── internal/
│   │   ├── domain/
│   │   │   └── subsanacion.go           # DTOs y estructuras de validación
│   │   ├── handler/
│   │   │   └── subsanacion_handler.go   # Handlers HTTP REST
│   │   ├── service/
│   │   │   ├── classification_service.go # Servicio de renombrado, reemplazo y fusión pdfcpu
│   │   │   └── semaforo_service.go       # Recálculo del estado de completitud del paciente
│   │   └── repository/
│   │       └── oracle_subsanacion_repo.go # Actualización de estados en PLANILLA_DIGITAL
```

---

### 4. Modelos de Datos y Entidades

#### **A. Estructuras de Datos en Go (`domain/subsanacion.go`)**

```
package domain

// Petición para clasificar manualmente un PDF no reconocido desde el visor dividido
type ClasificacionManualRequest struct {
	PdiID           int64  `json:"pdi_id" binding:"required"`
	TramiteID       int64  `json:"tramite_id" binding:"required"`
	NombrePaciente  string `json:"nombre_paciente" binding:"required"`
	ArchivoOriginal string `json:"archivo_original" binding:"required"` // Ej: PENDIENTE_tmp_x892.pdf
	CodigoMspElegido string `json:"codigo_msp_elegido" binding:"required"` // Ej: HCU_008.pdf
}

// Resumen del semáforo de completitud por paciente
type SemaforoPacienteDTO struct {
	PdiID                int64    `json:"pdi_id"`
	Tramite              int64    `json:"tramite"`
	Paciente             string   `json:"paciente"`
	Cedula               string   `json:"cedula"`
	TipoServicio         string   `json:"tipo_servicio"`
	EstadoDigitalizacion string   `json:"estado_digitalizacion"` // REQUIERE_VALIDACION, INCOMPLETO, COMPLETO
	ArchivosPresentes    []string `json:"archivos_presentes"`    // ["P_INDIVIDUAL.pdf", "C_COBERTURA.pdf"]
	ArchivosFaltantes    []string `json:"archivos_faltantes"`    // ["HCU_008.pdf"]
}
```

#### **B. Catálogo Oficial MSP para Menú Desplegable (Configuración Estática en Frontend/Backend)**

Los nombres normativos permitidos en la selección manual abarcan:

- `HCU_008.pdf` (Emergencia)
- `HCU_053.pdf` (Referencia y Derivación)
- `HCU_006.pdf` (Epicrisis)
- `HCU_017.pdf` (Protocolo Quirúrgico)
- `HCU_018A.pdf` (Transanestésico)
- `C_COBERTURA.pdf` (Consulta de Cobertura en Línea)
- `P_INDIVIDUAL.pdf` (Planilla Individual)
- `A_ENTREGA.pdf` (Acta Entrega-Recepción)
- `R_EXAMENES D.pdf` (Resultados de Exámenes)

---

### 5. Lógica de Negocio y Algoritmos

#### **Algoritmo 1: Clasificación Manual desde el Visor Dividido (PDFs No Identificados)**

```
[Usuario selecciona expediente en REQUIERE_VALIDACION]
       │
       ▼
[Cargar Visor Dividido Vuetify]
 ├── Lado Izquierdo: Visualizar /workspace/4. EXPEDIENTES/[PACIENTE]/PENDIENTE_tmp_x892.pdf
 └── Lado Derecho: Desplegar lista con Catálogo Oficial MSP (HCU_008.pdf, HCU_053.pdf, etc.)
       │
       ▼
[Usuario selecciona Código MSP (ej. HCU_008.pdf) y presiona "Guardar y Aplicar"]
       │
       ▼
[Backend Go recibe solicitud POST /api/v1/subsanacion/clasificar-manual]
       │
       ▼
¿Existe ya un HCU_008.pdf en la carpeta del paciente?
 ├── (SÍ) ──► Exec pdfcpu.Merge(HCU_008.pdf, PENDIENTE_tmp_x892.pdf) ──► Reemplazar
 └── (NO) ──► os.Rename(PENDIENTE_tmp_x892.pdf, HCU_008.pdf)
       │
       ▼
[Eliminar archivo PENDIENTE_ original]
       │
       ▼
[Recalcular Semáforo del Paciente] ──► ¿Están todos los obligatorios presentes?
 ├── (SÍ) ──► UPDATE PLANILLA_DIGITAL SET PDI_ESTADO_DIGITALIZACION = 'COMPLETO'
 └── (NO) ──► UPDATE PLANILLA_DIGITAL SET PDI_ESTADO_DIGITALIZACION = 'INCOMPLETO'
```

#### **Algoritmo 2: Inyección Directa de Documentos Faltantes**

```
[Usuario busca expediente INCOMPLETO (por Cédula, Planilla o Nombre)]
       │
       ▼
[Mostrar tarjeta de paciente con casillero en rojo: "Falta HCU_008.pdf"]
       │
       ▼
[Botón: Subir Documento Faltante ──► Adjuntar PDF (ej. scan_nuevo.pdf)]
       │
       ▼
[Backend Go recibe POST /api/v1/subsanacion/inyectado-puntual (Multipart)]
       │
       ▼
[Validar que el archivo sea un PDF válido (size > 0, header %PDF)]
       │
       ▼
[Renombrar binario directamente al Código MSP exigido: HCU_008.pdf]
       │
       ▼
[Copiar a /workspace/4. EXPEDIENTES/[NOMBRE_PACIENTE_SANEADO]/HCU_008.pdf]
       │
       ▼
¿Existía previamente una primera página de ese formulario?
 ├── (SÍ) ──► Aplicar fusión con pdfcpu.Merge para consolidar en 1 solo archivo
 └── (NO) ──► Guardar archivo directo
       │
       ▼
[Actualizar Oracle DB: PDI_ESTADO_DIGITALIZACION = 'COMPLETO']
```

---

### 6. Interfaz de Comunicación (APIs REST)

#### **1. Obtener Expedientes Pendientes de Subsanación**

- **Método / Ruta:** `GET /api/v1/subsanacion/pendientes`
- **Query Params:** `anio=2026&mes=09&estado=REQUIERE_VALIDACION|INCOMPLETO`
- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "total_pendientes": 2,
  "expedientes": [
    {
      "pdi_id": 293,
      "tramite": 6151828,
      "paciente": "PRUEBA PACIENTE JUAN CARLOS",
      "cedula": "1800808311",
      "tipo_servicio": "HOSPITALIZACIÓN",
      "estado_digitalizacion": "REQUIERE_VALIDACION",
      "archivos_no_identificados": ["PENDIENTE_tmp_x892.pdf"],
      "archivos_faltantes": []
    },
    {
      "pdi_id": 292,
      "tramite": 9999999,
      "paciente": "PRUEBA PACIENTE JUAN CARLOS",
      "cedula": "1700000000",
      "tipo_servicio": "EMERGENCIA",
      "estado_digitalizacion": "INCOMPLETO",
      "archivos_no_identificados": [],
      "archivos_faltantes": ["HCU_008.pdf"]
    }
  ]
}
```

#### **2. Clasificación Manual de PDF No Identificado**

- **Método / Ruta:** `POST /api/v1/subsanacion/clasificar-manual`
- **Payload de Entrada (JSON):**

```
{
  "pdi_id": 293,
  "tramite_id": 6151828,
  "nombre_paciente": "PRUEBA PACIENTE JUAN CARLOS",
  "archivo_original": "PENDIENTE_tmp_x892.pdf",
  "codigo_msp_elegido": "HCU_006.pdf"
}
```

- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "status": "SUCCESS",
  "message": "Archivo renombrado exitosamente a HCU_006.pdf.",
  "nuevo_estado_paciente": "COMPLETO"
}
```

#### **3. Inyección Puntual de Documento Faltante**

- **Método / Ruta:** `POST /api/v1/subsanacion/inyectar-faltante`
- **Headers:** `Content-Type: multipart/form-data`
- **FormData Params:**

|Campo|Tipo|Descripción|
|---|---|---|
|`pdi_id`|Number|ID único del registro en Oracle.|
|`codigo_msp`|String|Nombre oficial del formulario faltante (ej. `"HCU_008.pdf"`).|
|`pdf_file`|File|Binario del archivo corregido/escaneado.|

- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "status": "SUCCESS",
  "message": "Documento HCU_008.pdf inyectado correctamente en la carpeta del paciente.",
  "nuevo_estado_paciente": "COMPLETO"
}
```

---

### Decisiones Pendientes

1. **Servicio de Entrega de PDFs para el Visor (Stream vs. Static URL):** El desarrollador debe confirmar la ruta técnica para que la interfaz Vuetify renderice el PDF temporal no identificado en el navegador (por ejemplo, endpoint seguro `/api/v1/subsanacion/pdf-preview?path=...` que retorne el stream con header `Content-Type: application/pdf`).
2. **Estrategia ante Archivos Inyectados que superen la Resolución Permitida:** La normativa de la DPSP exige resoluciones entre **72 DPI y 300 DPI**. _El desarrollador debe definir si el endpoint de inyección ejecutará una comprobación previa mediante `pdfcpu` / ImageMagick para rechazar PDFs que no cumplan la resolución antes de copiarlos al expediente._

---

## Estado de implementación en Folio (2026-10-06)

Esta sección describe el código actual y no certifica ejecución contra la base Oracle institucional ni aprobación normativa.

- El menú **Objeciones** permite elegir un primer ingreso preparado y marcar manualmente los trámites observados. El espacio separado se crea al confirmar al menos un trámite; la carga de documentos de cabecera no bloquea su creación. El sistema no interpreta el informe de liquidación ni la matriz.
- Solo se copian los trámites seleccionados que tienen PDFs vinculados en el primer ingreso. Cada trámite se guarda en una carpeta de atención única, formada con cédula, paciente y número de trámite, bajo `4. EXPEDIENTES/`; la subcarpeta correspondiente bajo `5. ANEXOS/` aparece solo cuando se agrega un justificativo. El primer ingreso no se modifica. El operador puede agregar trámites omitidos desde el espacio ya creado; al hacerlo se invalida la matriz cargada para que se vuelva a subir actualizada.
- Los documentos de cabecera se cargan después de crear el espacio: `I_LIQUIDACION.pdf`, `1. OFICIO DE PAGO.pdf`, `2. PLANILLA CONSOLIDADA.pdf` y la matriz oficial `3. MATRIZ_OBJECIONES_<SERVICIO>_<MES>_<AÑO>.xlsm`. La matriz no se genera ni interpreta Folio; la plantilla oficial debe detallar por trámite el valor objetado, el código, el motivo y la respuesta técnica.
- Por cada trámite, el operador guarda `P_INDIVIDUAL.pdf` con postura `ACEPTA` o `RECHAZA`. También debe estar presente `C_COBERTURA.pdf`; se copia desde el primer ingreso si existe y se puede cargar o reemplazar en el espacio de objeciones. Los anexos PDF en `5. ANEXOS/<CEDULA>_<PACIENTE>_<TRAMITE>/` son opcionales. Se pueden nombrar `FICHA_TECNICA.pdf`, `FACTURA_DISPOSITIVO.pdf`, `FACTURAS_RESPALDO.pdf`, `FACTURA_COMPRA.pdf`, `PROTOCOLOS_MEDICOS.pdf`, `PROTOCOLO_DETALLADO.pdf` o `EXAMEN_ADICIONAL.pdf`; otros nombres descriptivos se normalizan a mayúsculas, sin tildes, con guiones bajos y extensión `.pdf`. Los duplicados reciben un sufijo numérico.
- La descarga ZIP exige los cuatro documentos de cabecera con extensión/firma correspondiente, postura y `P_INDIVIDUAL.pdf` por cada trámite, además de `C_COBERTURA.pdf` válido por trámite. Los anexos no forman parte de las condiciones de cierre. Se mantienen los controles generales de documentos pendientes de fusión.
- La creación y edición del espacio derivado no escriben `PDI_OBJETADO`, `PDI_ESTADO_DIGITALIZACION`, rutas documentales ni coberturas en Oracle. El cruce consulta identidades del período.
- El sistema valida la firma PDF de los tres documentos PDF y que la matriz sea un contenedor Excel macro-enabled `.xlsm`; no inspecciona sus celdas ni certifica que incluya los datos de respuesta. Tampoco valida resolución DPI, checklist documental integral ni certifica el cierre formal del MSP.
- Las pruebas automatizadas usan nombres y números ficticios. La vista previa y el flujo completo requieren una cuenta con acceso a Oracle y un período preparado real; no se usaron datos de pacientes reales como fixtures.

---
