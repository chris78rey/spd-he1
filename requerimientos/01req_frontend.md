---
base: "[[00_Inbox/Notas de tareas/ejecucion del primer prompt del spd MSP.md|ejecucion del primer prompt del spd MSP]]"
tags:
  - msp
  - spd
  - desarrollo_sw
  - prompt
---
# Documento de Diseño Técnico

**Proyecto:** Módulo Automatizado de Ingesta, Clasificación Híbrida y Subsanación Manual para la ACFSS (DPSP - MSP)  
**Componente:** **Frontend de Ingesta y Carga Dual**  
**Rol:** Arquitecto de Software & Tech Lead

---

### 1. Stack Tecnológico y Dependencias

Basado estrictamente en las fuentes del proyecto, el stack tecnológico aprobado e identificado para la capa frontend comprende:

- **Framework de Interfaz de Usuario:** **Vuetify** (basado en Vue.js), especificado como la biblioteca de componentes UI para la construcción de la interfaz.
- **Cliente HTTP:** **Axios** o API Fetch nativa de JavaScript para la transmisión asíncrona de archivos pesados mediante peticiones `multipart/form-data`.
- **Gestor de Estado (State Management):** **Pinia** (o Vuex), para mantener el estado reactivo del formulario de carga dual, el porcentaje de progreso de subida y los tokens de estado del lote.
- **Componentes de Captura de Archivos:** Componentes de Vuetify (`v-file-input`, zonas de arrastre/drag-and-drop HTML5) para la ingesta del contenedor masivo `.zip` y los tres casilleros independientes de cabecera.
- **Validación en Cliente:** JavaScript nativo (ES6+) para la inspección previa de extensiones (`.zip`, `.pdf`, `.xlsm`) y tipos MIME (`application/zip`, `application/pdf`, `application/vnd.ms-excel.sheet.macroEnabled.12`).

---

### 2. Contexto y Flujo de Datos

#### **Problema que resuelve:**

El sistema hospitalario emisor del prestador únicamente exporta la documentación médica/clínica agrupada en carpetas numéricas por planilla con nombres de PDF temporales o aleatorios (`tmp_a1.pdf`, `doc_99.pdf`), pero **no genera los documentos habilitantes macro, financieros ni administrativos** (`M_PAGO.pdf`, `C_CONSOLIDADA.pdf`, ni la matriz oficial Excel `.xlsm`).

Este módulo resuelve esa desconexión mediante una pantalla de **Carga Dual** que permite al operador subir el lote masivo `.zip` enviado por el sistema hospitalario y, en la misma interfaz, adjuntar los 3 documentos de cabecera indispensables para la aprobación en Ventanilla Única de la Dirección Provincial de Salud de Pichincha (DPSP).

#### **Flujos de Entrada (Inputs):**

- **Lote Clínico Masivo:** Archivo comprimido `.zip` exportado mensualmente con las carpetas numéricas de las planillas.
- **Matriz de Planillaje Oficial:** Archivo `.xlsm` con macros descargado del MSP con el desglose de prestaciones y tarifas.
- **Planilla Consolidada:** Archivo PDF (`C_CONSOLIDADA.pdf` en primer ingreso o `P_CONSOLIDADA.pdf` en objeciones) firmado por los responsables médico y financiero.
- **Oficio de Solicitud de Pago:** Archivo PDF (`M_PAGO.pdf`) firmado por la Máxima Autoridad o Representante Legal.
- **Parámetros del Período:** Selección manual del tipo de servicio (Emergencia, Hospitalización, Ambulatorio, etc.), mes y año de prestación.
- **Catálogo de servicios:** Incluir los once tipos de [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md) y mantener su selección para generar la carpeta madre y la matriz. No limitar el catálogo a las opciones actuales del prototipo.

#### **Flujos de Salida (Outputs):**

- **Transmisión al Backend:** Envío de la carga empaquetada `multipart/form-data` hacia el servicio en Go.
- **Respuesta de Servidor:** Recepción del identificador del proceso (`job_id`) e inicialización de los indicadores visuales/barras de progreso para el monitoreo del procesamiento.

---

### 3. Estructura de Directorios Sugerida

Aplicando las mejores prácticas de la arquitectura Vue / Vuetify, la estructura del módulo se organiza de la siguiente manera:

```
frontend-ingesta/
├── src/
│   ├── assets/                      # Logotipos institucionales (MSP/DPSP) y estilos globales
│   ├── components/                  # Componentes reutilizables de Vuetify
│   │   ├── FileDropZone.vue         # Componente drag-and-drop para el lote masivo .zip
│   │   ├── HeaderUploadCard.vue     # Casilleros independientes para M_PAGO, C_CONSOLIDADA y .xlsm
│   │   ├── PeriodSelector.vue       # Desplegables para Mes, Año y Tipo de Servicio
│   │   └── UploadProgressBar.vue    # Barra de estado y progreso de la transmisión
│   ├── views/
│   │   └── IngestaDualView.vue      # Vista principal que integra la Carga Dual
│   ├── services/
│   │   └── ingestaApiService.js     # Cliente Axios con interceptores y llamadas Multipart
│   ├── stores/
│   │   └── ingestaStore.js          # Store de Pinia para estado reactivo del formulario
│   ├── utils/
│   │   └── fileValidators.js        # Funciones puras de validación de extensión y tamaño
│   ├── router/
│   │   └── index.js                 # Definición de rutas de la SPA
│   ├── App.vue
│   └── main.js                      # Configuración de inicialización de Vuetify y Plugins
├── package.json
└── vite.config.js
```

---

### 4. Modelos de Datos y Entidades

#### **Estado del Formulario Frontend (`IngestaState` in PiniaStore)**

```
// store/ingestaStore.js
export const useIngestaStore = defineStore('ingesta', {
  state: () => ({
    periodo: {
      mes: '',             // Ej: "09"
      anio: '',            // Ej: "2026"
      tipoServicio: ''     // Ej: "EMERGENCIA", "HOSPITALIZACION", "AMBULATORIO"
    },
    archivos: {
      zipLote: null,       // File Object: .zip con carpetas numéricas
      matrizXlsm: null,    // File Object: MATRIZ_[SERVICIO]_[MES_AÑO].xlsm
      consolidadaPdf: null,// File Object: C_CONSOLIDADA.pdf / P_CONSOLIDADA.pdf
      oficioPdf: null      // File Object: M_PAGO.pdf
    },
    ui: {
      isUploading: false,
      uploadProgress: 0,   // Porcentaje (0 - 100)
      errorMessage: null,
      jobId: null
    }
  })
});
```

---

### 5. Lógica de Negocio y Algoritmos

```
[Inicio de Carga Dual]
       │
       ▼
[Selección de Período (Mes/Año/Servicio)]
       │
       ▼
[Arrastre/Selección de Archivos]
 ├── 1. Lote Masivo (.zip)
 ├── 2. Matriz de Planillaje (.xlsm)
 ├── 3. Planilla Consolidada (.pdf)
 └── 4. Oficio de Pago (.pdf)
       │
       ▼
[Validaciones Client-Side (Pre-flight)]
 ├── ¿Extensiones correctas? (.zip, .xlsm, .pdf)
 ├── ¿Archivos no vacíos? (size > 0)
 └── ¿Todos los casilleros obligatorios llenos?
       │
       ├────► (NO) ──► Bloquear botón "Iniciar Procesamiento" / Mostrar alerta
       │
       ▼ (SÍ)
[Habilitar Botón "Iniciar Procesamiento"]
       │
       ▼
[Construir Multipart FormData & POST /api/v1/ingesta/lote-dual]
       │
       ▼
[Monitorear Evento onUploadProgress (Axios)]
       │
       ▼
[Recibir HTTP 202 Accepted + job_id] ──► Transición a Pantalla de Progreso/Pipeline
```

#### **Paso a Paso del Algoritmo:**

1. **Pre-validación de Archivos (`fileValidators.js`):**
    - Al seleccionar o soltar un archivo en cualquier casillero de Vuetify, el sistema intercepta el evento `@change` / `@drop`.
    - Se comprueba la extensión contra una lista blanca explícita (`.zip` para la carga masiva; `.xlsm` para la matriz; `.pdf` para Oficio y Consolidada).
    - Se valida que `file.size > 0` para prevenir la carga de documentos corruptos de 0 bytes.
2. **Control de Habilitación Reactiva:**
    - El botón `[ Iniciar Procesamiento / Cargar Lote ]` se mantiene deshabilitado mediante una propiedad computada (`isFormValid`) hasta que los cuatro casilleros contengan archivos válidos y se hayan seleccionado mes, año y tipo de servicio.
3. **Empaquetado FormData y Transmisión:**
    - Se instancia un objeto `FormData`.
    - Se adjuntan los binarios: `zip_file`, `matriz_file`, `consolidada_file`, `oficio_file`.
    - Se adjunta la metadata: `mes`, `anio`, `tipo_servicio`.
    - Se ejecuta la petición HTTP POST invocando `ingestaApiService.postLoteDual()` configurando la opción `onUploadProgress` de Axios para actualizar `uploadProgress` en la UI en tiempo real.
4. **Manejo de Respuestas e Inconvenientes:**
    - **Éxito (HTTP 202 Accepted):** Se almacena el `job_id` devuelto por el backend y se redirige a la vista de seguimiento del pipeline en Go.
    - **Error de Red o Validación (HTTP 400/500):** Se detiene la animación de subida, se captura el mensaje estructurado de error devuelto por la API y se despliega una alerta de Vuetify (`v-alert`) indicando la falla.

---

### 6. Interfaz de Comunicación (APIs)

#### **Endpoint Principal: Envío de Carga Dual de Lote Mensual**

- **Método:** `POST`
- **Ruta:** `/api/v1/ingesta/lote-dual`
- **Headers:** `Content-Type: multipart/form-data`
- **Payload de Entrada (FormData):**

|Campo|Tipo|Requerido|Descripción|
|---|---|---|---|
|`zip_file`|File (`.zip`)|Sí|Archivo comprimido con subcarpetas numéricas por planilla.|
|`matriz_file`|File (`.xlsm`)|Sí|Matriz Excel oficial del MSP con macros.|
|`consolidada_file`|File (`.pdf`)|Sí|Planilla Consolidada firmada (`C_CONSOLIDADA.pdf`).|
|`oficio_file`|File (`.pdf`)|Sí|Oficio de solicitud de pago firmado (`M_PAGO.pdf`).|
|`mes`|String|Sí|Período prestacional mes (ej. `"09"`).|
|`anio`|String|Sí|Período prestacional año (ej. `"2026"`).|
|`tipo_servicio`|String|Sí|Categoría de servicio (ej. `"EMERGENCIA"`, `"HOSPITALIZACION"`).|

- **Respuesta de Exito Esperada (HTTP 202 Accepted):**

```
{
  "status": "ACCEPTED",
  "job_id": "JOB-202609-001",
  "message": "Carga dual recibida correctamente. Archivos depositados en el área de staging.",
  "timestamp": "2026-09-30T06:50:00Z"
}
```

- **Respuesta de Error de Validación (HTTP 400 Bad Request):**

```
{
  "status": "ERROR",
  "code": "INVALID_FILE_TYPE",
  "message": "El archivo proporcionado como matriz debe tener extension .xlsm habilitada para macros.",
  "timestamp": "2026-09-30T06:50:00Z"
}
```

---

### Decisiones Pendientes

Para que el desarrollador frontend pueda iniciar la codificación sin ambigüedades, se deben definir formalmente los siguientes puntos operativos con el área de infraestructura/backend:

1. **Límite Máximo de Tamaño de Archivo (_Max Upload Size_):** Las fuentes no especifican el tamaño tope en megabytes (MB) para el archivo `.zip` que agrupa cientos o miles de expedientes clínicos del mes. _El programador debe confirmar el límite configurado en el servidor web (Nginx/Ubuntu) para ajustar la regla de validación en el cliente (`file.size < MAX_SIZE`)._
2. **Mecanismo de Seguimiento del Pipeline en Tiempo Real:** Las fuentes indican que el backend ejecuta el procesamiento en segundo plano tras recibir los archivos. _El programador debe consultar si el frontend debe consultar el estado mediante sondeo periódico HTTP (polling a `/api/v1/ingesta/status/:jobId`) o si existe un canal de WebSockets / Server-Sent Events (SSE) para actualizar la barra de avance._
3. **Estrategia ante Interrupción de Transmisión (_Chunked Uploads_):** No se especifica si en redes inestables se requerirá el reintento por fragmentos (_chunking_) para el archivo `.zip` masivo o si se mantendrá la transmisión simple Multipart en un solo paquete.
