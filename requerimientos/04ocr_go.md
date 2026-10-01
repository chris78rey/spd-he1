# Documento de Diseño Técnico

**Proyecto:** Módulo Automatizado de Ingesta, Clasificación Híbrida y Subsanación Manual para la ACFSS (DPSP - MSP)  
**Componente:** **Pipeline Backend Concurrente y Clasificador Híbrido (Go + OCR + YAML)**  
**Rol:** Arquitecto de Software & Tech Lead

---

### 1. Stack Tecnológico y Dependencias

Con base en las especificaciones técnicas y operativas del proyecto, las tecnologías aprobadas para este módulo son:

- **Lenguaje y Runtimes Backend:** **Go (Golang)** versión **1.22 o superior**.
- **Patrón de Concurrencia:** **Worker Pool** nativo en Go mediante _goroutines_ y canales (_channels_) amortiguados.
- **Driver de Conexión a Base de Datos:** **`godror`** para Go con soporte nativo para **Oracle Instant Client** (conectando a Oracle Database 11gR2).
- **Motor OCR de Respaldo:** **Tesseract OCR** (invocado vía librería `gosseract` o llamada al binario del sistema optimizado para idioma español `spa`).
- **Motor de Manipulación y Fusión de PDFs:** Biblioteca **`pdfcpu`** en Go (o utilidades nativas de parsing de streams PDF).
- **Motor de Reglas Desacoplado:** Parser de archivos **YAML** (`gopkg.in/yaml.v3`) para consumir `reglas_clasificacion.yaml`.
- **Plataforma de Despliegue:** Sistema Operativo **Ubuntu Linux**, con configuraciones de rutas en archivos externos (`/etc/app/config.json` o variables de entorno) fuera del directorio del código compilado.

---

### 2. Contexto y Flujo de Datos

#### **Problema que resuelve:**

El sistema de gestión hospitalaria del prestador exporta mensualmente miles de expedientes médicos agrupados únicamente por carpetas numéricas de planilla (`1234560/`, `1234561/`), conteniendo PDFs con nombres arbitrarios o temporales (`tmp_a1.pdf`, `doc_99.pdf`) sin estandarización.

Este módulo resuelve la **transformación, clasificación masiva y estructuración normativa**:

1. Recorre concurrentemente las carpetas numéricas sin saturar la memoria ni los descriptores de archivos del sistema operativo.
2. Resuelve la identidad del paciente en tiempo constante \(O(1)\) mediante la caché en memoria cargada desde Oracle.
3. Analiza el contenido de cada PDF (vectorial o escaneado vía OCR).
4. Clasifica y renombra los archivos según la nomenclatura oficial del Ministerio de Salud Pública (DPSP - MSP).
5. Consolida (_merge_) documentos repetidos del mismo tipo en un solo archivo cronológico por paciente.

#### **Flujos de Entrada (Inputs):**

- **Directorio de Entrada (_Staging_):** Ruta configurada externamente en Ubuntu (ej. `/staging/`) con carpetas numéricas por planilla (`1234560/`).
- **Caché de Datos en Memoria (`map[string]DatosPaciente`):** Registros previamente precargados desde Oracle DB (`PDI_TRAMITE`, `PDI_PACIENTE`, `PDI_CEDULA`, `PDI_COD_SERVICIO`).
- **Catálogo de Reglas YAML (`reglas_clasificacion.yaml`):** Archivo externo con palabras clave asociadas a códigos normativos (`HCU_008.pdf`, `HCU_053.pdf`, etc.).
- **Fuente de reglas y checklist:** Usar [`08_reglas_ocr_y_checklist.md`](08_reglas_ocr_y_checklist.md) y el catálogo raíz [`reglas_clasificacion.yaml`](../reglas_clasificacion.yaml); no inferir patrones OCR faltantes.

#### **Flujos de Salida (Outputs):**

- **Árbol de Directorios Oficial:** Archivos clasificados dentro de `[SERVICIO]_[MES]_[AÑO]/4. EXPEDIENTES/[PDI_PACIENTE_NORMALIZADO]/`, según [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md).
- **PDFs Fusionados:** Unificación de formularios múltiples mediante `pdfcpu.Merge`.
- **Aislamiento de Pendientes:** PDFs no reconocidos etiquetados como `PENDIENTE_tmp_a1.pdf` para la bandeja de revisión manual.
- **Actualización de Oracle:** Cambios de estado en la tabla `PLANILLA_DIGITAL` (`PDI_ESTADO_DIGITALIZACION = 'COMPLETO' | 'REQUIERE_VALIDACION' | 'INCOMPLETO'`).

---

### 3. Estructura de Directorios Sugerida

Aplicando las mejores prácticas de arquitectura limpia en Go (Clean Architecture / Domain-Driven Layout):

```
pipeline-backend-go/
├── cmd/
│   └── pipeline/
│       └── main.go                 # Punto de entrada y orquestador del servicio
├── config/
│   ├── config.go                   # Lector de configuración JSON/Ubuntu
│   └── reglas_clasificacion.yaml   # Reglas desacopladas de palabras clave MSP
├── internal/
│   ├── domain/
│   │   ├── expediente.go           # Modelos del dominio (Paciente, Documento, Estado)
│   │   └── clasificador.go         # Interfaces del clasificador
│   ├── pipeline/
│   │   ├── worker_pool.go          # Implementación del Worker Pool concurrente
│   │   ├── job.go                  # Estructura del Job de procesamiento por planilla
│   │   └── processor.go            # Pipeline paso a paso por expediente
│   ├── classifier/
│   │   ├── pdf_extractor.go        # Extracción de texto vectorial en memoria
│   │   ├── ocr_engine.go           # Motor Tesseract OCR para imágenes (200-300 DPI)
│   │   └── evaluator_yaml.go       # Evaluador de patrones contra reglas_clasificacion.yaml
│   ├── pdfservice/
│   │   └── merger.go               # Wrapper de pdfcpu para concatenación de PDFs
│   └── repository/
│       └── oracle_updater.go       # Sincronización de estados con Oracle DB
├── go.mod
└── go.sum
```

---

### 4. Modelos de Datos y Entidades

#### **A. Archivo de Reglas YAML (`reglas_clasificacion.yaml`)**

```
formularios:
  - codigo_msp: "HCU_008.pdf"
    descripcion: "Formulario de Emergencia"
    palabras_clave:
      - "FORMULARIO 008"
      - "EMERGENCIA"
      - "MOTIVO DE CONSULTA"
  - codigo_msp: "HCU_053.pdf"
    descripcion: "Referencia y Derivación"
    palabras_clave:
      - "FORMULARIO 053"
      - "REFERENCIA"
      - "CONTRARREFERENCIA"
  - codigo_msp: "HCU_006.pdf"
    descripcion: "Epicrisis"
    palabras_clave:
      - "FORMULARIO 006"
      - "EPICRISIS"
      - "RESUMEN DE ALTA"
  - codigo_msp: "HCU_017.pdf"
    descripcion: "Protocolo Quirúrgico"
    palabras_clave:
      - "FORMULARIO 017"
      - "PROTOCOLO QUIRÚRGICO"
      - "CIRUGÍA"
  - codigo_msp: "HCU_018A.pdf"
    descripcion: "Protocolo Transanestésico"
    palabras_clave:
      - "FORMULARIO 018A"
      - "TRANSANESTÉSICO"
      - "ANESTESIA"
  - codigo_msp: "C_COBERTURA.pdf"
    descripcion: "Consulta de Cobertura en Línea"
    palabras_clave:
      - "COBERTURA DE SALUD"
      - "CERTIFICADO DE AFILIACIÓN"
      - "COMPROBANTE DE DERECHO"
  - codigo_msp: "P_INDIVIDUAL.pdf"
    descripcion: "Planilla Individual"
    palabras_clave:
      - "PLANILLA INDIVIDUAL"
      - "VALOR FACTURADO"
  - codigo_msp: "A_ENTREGA.pdf"
    descripcion: "Acta de Entrega-Recepción"
    palabras_clave:
      - "ACTA DE ENTREGA"
      - "RECEPCIÓN DEL SERVICIO"
```

#### **B. Estructuras Internas en Go (`internal/domain/expediente.go`)**

```
package domain

import "time"

type DatosPaciente struct {
	Tramite         int64  `json:"tramite"`          // PDI_TRAMITE
	PacienteRaw     string `json:"paciente_raw"`     // PDI_PACIENTE original
	PacienteSaneado string `json:"paciente_saneado"` // Nombres limpios para el directorio
	Cedula          string `json:"cedula"`           // PDI_CEDULA / Pasaporte
	CodServicio     string `json:"cod_servicio"`     // URG, HSP, EMR, etc.
}

type DocumentoProcesado struct {
	RutaOriginal string `json:"ruta_original"` // Ej: /staging/1234560/tmp_a1.pdf
	CodigoMSP    string `json:"codigo_msp"`    // Ej: HCU_008.pdf o PENDIENTE_tmp_a1.pdf
	Identificado bool   `json:"identificado"`  // true si coincidió con YAML
	MetodoLectura string `json:"metodo"`       // "VECTORIAL" o "OCR_TESSERACT"
}

type JobPlanilla struct {
	RutaCarpeta string       `json:"ruta_carpeta"` // /staging/1234560
	NumTramite  int64        `json:"num_tramite"`  // 1234560
	Paciente    DatosPaciente `json:"paciente"`
}
```

---

### 5. Lógica de Negocio y Algoritmos

```
[Inicio Pipeline en Go]
       │
       ▼
[Generar Canal de Trabajos: chan JobPlanilla]
       │
       ▼
[Lanzar Worker Pool (Goroutines 4-8)] ──► [Escuchar Canal de Trabajos]
                                                │
                                                ▼
                                [Paso 1: Resolver Identidad desde Cache Oracle]
                                 ├── Saneamiento de Nombres (ASCII, Trim, ToUpper)
                                 └── Crear Dir: 4. EXPEDIENTES/[PACIENTE_SANEADO]/
                                                │
                                                ▼
                                [Paso 2: Recorrer PDFs Internos en /staging/1234560/]
                                                │
                                                ▼
                                [Paso 3: Extracción de Texto (Nivel 1 Vectorial)]
                                 ¿Contiene texto digital en Pág 1-2?
                                 ├── (SÍ) ──► Enviar Buffer a Evaluador YAML
                                 └── (NO) ──► [Nivel 2: OCR Tesseract Pág 1 (200-300 DPI)]
                                                │
                                                ▼
                                [Paso 4: Evaluador de Reglas YAML]
                                 ¿Coincide palabra clave?
                                 ├── (SÍ) ──► Asignar Código MSP (ej. HCU_008.pdf)
                                 └── (NO) ──► Renombrar PENDIENTE_[orig].pdf & Estado = REQUIERE_VALIDACION
                                                │
                                                ▼
                                [Paso 5: Regla de Fusión (pdfcpu.Merge)]
                                 ¿Existe más de un archivo para el mismo Código MSP?
                                 ├── (SÍ) ──► Concatenar cronológicamente en 1 solo PDF
                                 └── (NO) ──► Copiar a subcarpeta del paciente
                                                │
                                                ▼
                                [Paso 6: Sincronizar Estado en Oracle DB]
```

#### **Paso a Paso del Algoritmo:**

1. **Inicialización y Concurrencia (`worker_pool.go`):**
    
    - El orquestador lee los directorios contenidos en `/staging/` (ej. `1234560/`, `1234561/`).
    - Se inicia un grupo de _workers_ asignando un número de _goroutines_ proporcional a los núcleos del CPU del servidor Ubuntu (ej. 4 u 8 _workers_).
    - Cada _worker_ toma un `JobPlanilla` del canal.
2. **Resolución de Identidad y Saneamiento (`processor.go`):**
    
    - El _worker_ consulta el número de trámite en el `map[string]DatosPaciente` cargado en memoria.
    - Se aplica la función de limpieza y normalización sobre `NOMBRES_PACIENTE`:
        - Convertir todo el texto a mayúsculas.
        - Eliminar puntos, comas o caracteres iniciales (ej. `'. PRUEBA PACIENTE JUAN CARLOS'` \(\rightarrow\) `'PRUEBA PACIENTE JUAN CARLOS'`).
        - Reemplazar diacríticos (`Á` \(\rightarrow\) `A`, `Ñ` \(\rightarrow\) `N`, `Ü` \(\rightarrow\) `U`).
        - Normalizar espacios o sustituir por guion bajo (`_`).
        - **Identidad de carpeta:** usar exclusivamente `PDI_PACIENTE` normalizado; no adjuntar pasaporte, cédula, trámite ni marcadores para apellidos ausentes. Si la identidad es inválida o colisiona, derivar a revisión según `07_reglas_estructura_acfss.md`.
    - Se crea la subcarpeta oficial de destino: `/workspace/4. EXPEDIENTES/PRUEBA_PACIENTE_JUAN_CARLOS/`.
3. **Estrategia Híbrida de Lectura PDF vs. OCR (`pdf_extractor.go` & `ocr_engine.go`):**
    
    - **Nivel 1 (Extracción Vectorial):** Intenta extraer el texto nativo de las páginas 1 y 2 del PDF directamente en memoria. Si devuelve texto estructurado, omite el OCR.
    - **Nivel 2 (OCR Tesseract de Respaldo):** Si el PDF es un escaneo plano sin capa de texto, renderiza la **Página 1** a escala de grises con una resolución estricta entre **200 DPI y 300 DPI**. Ejecuta Tesseract OCR en español (`spa`) para leer los encabezados del documento.
4. **Clasificación y Renombrado (`evaluator_yaml.go`):**
    
    - Compara la cadena de texto obtenida contra las listas de `palabras_clave` definidas en `reglas_clasificacion.yaml`.
    - Al encontrar la primera coincidencia, asigna el `codigo_msp` correspondiente (ej. `HCU_008.pdf`, `HCU_053.pdf`, `C_COBERTURA.pdf`).
    - **Sin Coincidencia (No Identificado):** Si el texto no coincide por manchas o ilicitud en la imagen, el sistema **no detiene el lote**. Copia el archivo como `PENDIENTE_tmp_a1.pdf` dentro de la carpeta del paciente y actualiza el registro en Oracle con `PDI_ESTADO_DIGITALIZACION = 'REQUIERE_VALIDACION'` para su resolución en el visor manual.
5. **Documentos reconocidos repetidos:**

    - El primer documento ocupa el nombre MSP canónico (`HCU_008.pdf`); los siguientes reciben sufijos (`HCU_008_1.pdf`, `HCU_008_2.pdf`) y se marcan como pendientes de fusión, sin quedar como documentos no reconocidos.
    - Desde la carpeta del paciente se revisan, ordenan y fusionan explícitamente. El resultado usa el nombre canónico; las fuentes anteriores se conservan en el expediente privado.
    - La descarga del ZIP final se habilita cuando se resolvieron todos los grupos pendientes de fusión. El sistema no infiere el orden cronológico de páginas: lo confirma el usuario en la vista de fusión.
6. **Control de Completitud por Servicio:**
    
    - Compara los formularios encontrados contra la lista de obligatorios traída de Oracle según el `PDI_COD_SERVICIO` (ej. si es Emergencia exige `HCU_008.pdf`; si es Hospitalización exige `HCU_006.pdf`).
    - Si falta algún formulario, registra el expediente con `PDI_ESTADO_DIGITALIZACION = 'INCOMPLETO'`. Si todo está correcto, marca `PDI_ESTADO_DIGITALIZACION = 'COMPLETO'`, `PDI_PROCESADO = 'S'` y `PDI_FECHA_PROCESO = SYSDATE`.

---

### 6. Interfaz de Comunicación (APIs / Eventos Internos)

#### **A. Interfaz Go para Invocación del Pipeline (`pipeline/processor.go`)**

```
package pipeline

import (
	"context"
	"pipeline-backend-go/internal/domain"
)

type PipelineRunner interface {
	// Ejecuta el lote completo para un año y mes específico
	EjecutarLoteMensual(ctx context.Context, req domain.LoteRequest) (*domain.LoteResponse, error)

	// Reprocesa un expediente puntual tras la inyección de un documento faltante
	ReprocesarExpediente(ctx context.Context, tramiteID int64) error
}
```

#### **B. Contratos de Datos (Request / Response JSON)**

- **Estructura de Entrada `LoteRequest`:**

```
{
  "anio": "2026",
  "mes": "09",
  "tipo_servicio": "EMERGENCIA",
  "ruta_staging": "/staging/staging_mensual_2026_09/",
  "num_workers": 6
}
```

- **Estructura de Salida `LoteResponse`:**

```
{
  "job_id": "JOB-202609-001",
  "total_planillas": 200,
  "exitosos_completos": 192,
  "requieren_validacion": 5,
  "incompletos": 3,
  "tiempo_ejecucion_segundos": 38.4,
  "timestamp_fin": "2026-09-30T06:58:00Z"
}
```

---

### Decisiones Pendientes

Para que el desarrollador backend pueda iniciar la implementación sin bloqueos, se deben confirmar los siguientes aspectos con el equipo de infraestructura/sistemas:

1. **Ruta Absoluta de Instalación de Tesseract OCR en Ubuntu:** Las fuentes especifican el uso de Tesseract OCR en español (`spa`), pero el desarrollador debe verificar si el binario se encuentra en la ruta estándar del sistema (`/usr/bin/tesseract`) o si debe configurarse en el archivo `config.json` la variable `TESSDATA_PREFIX` para las plantillas de entrenamiento en español.
2. **Estrategia de Almacenamiento Temporal para Renderizado de Páginas:** Para ejecutar el OCR en imágenes escaneadas, la página 1 del PDF debe renderizarse temporalmente a PNG/JPEG (200–300 DPI). _El desarrollador debe confirmar la ruta del directorio temporal en disco (ej. `/tmp/pdf_render/`) para asegurar permisos de escritura y una rutina de limpieza automática (`defer os.Remove`) tras el análisis OCR._
