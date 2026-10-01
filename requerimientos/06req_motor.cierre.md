# Documento de Diseño Técnico

**Proyecto:** Módulo Automatizado de Ingesta, Clasificación Híbrida y Subsanación Manual para la ACFSS (DPSP - MSP)  
**Componente:** **Motor de Validación, Guardián de Cierre y Empaquetado Normativo**  
**Rol:** Arquitecto de Software & Tech Lead

---

### 1. Stack Tecnológico y Dependencias

Basado estrictamente en las especificaciones normativas de la Dirección Provincial de Salud de Pichincha (DPSP - MSP) y en la arquitectura técnica definida en las fuentes:

- **Backend y Orquestador de Cierre:**
    - **Lenguaje:** **Go (Golang) 1.22+**, encargado de ejecutar las reglas de validación de completitud, la auditoría técnica de archivos y la compilación del árbol final de directorios.
    - **Auditoría de PDFs y Firmas:** Biblioteca **`pdfcpu`** en Go para la verificación del catálogo de streams PDF, inspección de dimensiones/resolución (72–300 DPI) y validación de contenedores de firma electrónica PAdES (ISO 32000-1 / SHA-256).
    - **Empaquetador y Generador de Medios:** Herramientas de sistema en Ubuntu Linux (`mkisofs` / `genisoimage` o el paquete nativo `archive/zip` de Go) para empaquetar el lote final en un archivo comprimido o imagen ISO lista para grabación en disco óptico.
- **Frontend (Interfaz de Control de Cierre):**
    - **Framework UI:** **Vuetify** (Vue.js), para la vista de checklist de completitud, semáforo de aprobación del lote y generación de la plantilla de rotulado térmico de CD/DVD.
    - **Cliente HTTP:** **Axios**, para invocar los servicios REST de auditoría pre-cierre y descarga del paquete empaquetado.
- **Base de Datos y Persistencia:**
    - **Motor BD:** **Oracle Database 11gR2** (vía driver `godror` con Oracle Instant Client).
    - **Persistencia:** Mapeo y actualización de los campos `PDI_CERRADO`, `PDI_ESTADO_DIGITALIZACION` y `PDI_PROCESADO` en la tabla `DIGITALIZACION.PLANILLA_DIGITAL`.

---

### 2. Contexto y Flujo de Datos

#### **Problema que resuelve:**

Para que un trámite de Auditoría de la Calidad de la Facturación de los Servicios de Salud (ACFSS) sea admitido en la Ventanilla Única de la DPSP (Plataforma Gubernamental Sur de Quito), debe cumplir de forma estricta con la estructura oficial de directorios, la presencia de los tres documentos habilitantes de cabecera, la resolución técnica de los escaneos (72 a 300 DPI) y la validez de los expedientes. Si un CD presenta carpetas incompletas, archivos corruptos o nombres fuera de norma, el personal de recepción rechaza el ingreso.

Este módulo actúa como el **Guardián de Entrega y Cierre (_Gatekeeper_)**:

1. **Auditoría Previa al Cierre:** Impide la generación del paquete final si existe al menos un expediente incompleto, un archivo sin clasificar (`PENDIENTE_`) o si faltan los documentos de cabecera.
2. **Validación Técnica de Formatos:** Verifica que el 100% de los documentos internos sean formato PDF válido y que la resolución de imagen esté dentro del rango normativo de 72 a 300 DPI (\(595 \times 842\) px a \(2480 \times 3508\) px en A4).
3. **Compilador del Árbol Oficial:** Construye la jerarquía definitiva para **Primer Ingreso** o para **Levantamiento de Objeciones** (incorporando la carpeta `5. ANEXOS/`).
4. **Generador de Medios y Rotulado:** Produce la imagen ISO o paquete `.zip` para ser quemado en **dos (2) copias idénticas en CD/DVD**, emitiendo la ficha de rotulado térmico estándar exigida por la DPSP.

#### **Flujos de Entrada (Inputs):**

- **Directorio de Trabajo (`/workspace/`):** Carpetas de expedientes procesados en `4. EXPEDIENTES/[NOMBRE_PACIENTE]/` y, si aplica, `5. ANEXOS/[NOMBRE_PACIENTE]/`.
- **Archivos Habilitantes de Cabecera:**
    - `1. OFICIO DE PAGO.pdf` (`M_PAGO.pdf`).
    - `2. PLANILLA CONSOLIDADA.pdf` (`C_CONSOLIDADA.pdf` o `P_CONSOLIDADA.pdf`).
    - `3. MATRIZ_[TIPO_SERVICIO]_[MES_AÑO].xlsm` (Matriz de planillaje oficial con macros).
- **Consulta de Estado en Oracle DB:** Verificación de que todos los registros del mes en `PLANILLA_DIGITAL` se encuentren en estado `COMPLETO` o `VALIDADO`.

#### **Flujos de Salida (Outputs):**

- **Paquete Final (ISO / ZIP):** Archivo descargable con la estructura oficial exacta del trámite para su grabación en 2 CD/DVDs.
- **Metadatos para Rotulado Térmico:** Generación del diseño de etiqueta física para el disco con los datos institucionales (Prestador, RUC, Servicio, Período, N° de Disco).
- **Cierre de Lote en Oracle DB:** Actualización masiva de los registros con `PDI_CERRADO = 'S'` y marca de tiempo.

---

### 3. Estructura de Directorios Sugerida

El árbol entregable debe cumplir [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md), incluida la diferencia entre primer ingreso y objeciones, los nombres de cabecera y la inclusión exclusiva de pacientes observados en las carpetas 4 y 5 del paquete de objeciones.

Organización limpia por capas separando la lógica del backend en Go y los componentes de Vuetify:

```
guardian-cierre-empaquetado/
├── frontend-vuetify/
│   ├── src/
│   │   ├── components/
│   │   │   ├── ChecklistCierreCard.vue     # Checklist con semáforos verde/rojo de validación pre-cierre
│   │   │   ├── DpiAuditTable.vue          # Tabla de reportes técnicos de resolución e integridad
│   │   │   └── CdRotuladoPreview.vue       # Vista previa de la carátula/etiqueta para rotulado del CD
│   │   ├── views/
│   │   │   └── CierreEmpaquetadoView.vue   # Pantalla principal del Guardián de Cierre
│   │   └── services/
│   │       └── cierreApiService.js        # Cliente Axios para auditar y solicitar empaquetado
├── backend-go/
│   ├── internal/
│   │   ├── domain/
│   │   │   └── empaquetado.go             # DTOs y estructuras de auditoría y empaquetado
│   │   ├── handler/
│   │   │   └── empaquetado_handler.go     # Handlers HTTP REST
│   │   ├── service/
│   │   │   ├── auditor_compleitud_service.go # Servicio de validación pre-flight (Cabecera + Expedientes)
│   │   │   ├── auditor_pdf_tech_service.go   # Servicio de inspección de PDF, DPI y firmas PAdES
│   │   │   ├── iso_compiler_service.go       # Compilador de la imagen ISO / .zip final
│   │   │   └── rotulado_label_service.go     # Generador de metadatos/etiqueta para impresión de CD
│   │   └── repository/
│   │       └── oracle_cierre_repo.go         # Actualización del campo PDI_CERRADO en Oracle
```

---

### 4. Modelos de Datos y Entidades

#### **A. Estructuras de Datos en Go (`domain/empaquetado.go`)**

```
package domain

import "time"

// Informe de Auditoría Previa al Cierre (Pre-flight Check)
type ReporteAuditCierreDTO struct {
	PeríodoMes         string   `json:"mes"`
	PeríodoAnio        string   `json:"anio"`
	TipoServicio       string   `json:"tipo_servicio"`
	EsObjecion         bool     `json:"es_objecion"`
	CabeceraValida     bool     `json:"cabecera_valida"`
	OficioPresente     bool     `json:"oficio_presente"`      // 1. OFICIO DE PAGO.pdf
	ConsolidadaPresente bool    `json:"consolidada_presente"` // 2. PLANILLA CONSOLIDADA.pdf
	MatrizXlsmPresente bool     `json:"matriz_xlsm_presente"` // 3. MATRIZ_[...].xlsm
	TotalExpedientes   int      `json:"total_expedientes"`
	ExpedientesOk      int      `json:"expedientes_ok"`
	ArchivosPendientes int      `json:"archivos_pendientes"`  // Con prefijo PENDIENTE_
	ErroresTecnicos    []string `json:"errores_tecnicos"`     // Violaciones de DPI o PDF corrupto
	AptoParaCierre     bool     `json:"apto_para_cierre"`     // true solo si cumple 100%
}

// Datos obligatorios para el Rotulado Físico del CD/DVD según normativa DPSP
type CDRotuladoMetadataDTO struct {
	InstitucionDestino string `json:"institucion_destino"` // "MSP - Dirección Provincial de Salud de Pichincha"
	RazonSocial        string `json:"razon_social"`        // Nombre del Prestador
	RucPrestador       string `json:"ruc_prestador"`       // RUC institucional
	TipoServicio       string `json:"tipo_servicio"`       // Ej: "HOSPITALIZACIÓN", "EMERGENCIA"
	PeriodoPrestacional string `json:"periodo_prestacional"`// Ej: "AGOSTO 2026"
	EsObjecion         bool   `json:"es_objecion"`
	NumeroDisco        string `json:"numero_disco"`        // "Disco 1 de 1"
	NumeroTramiteOficio string `json:"numero_tramite_oficio"`
}

// Solicitud de Compilación y Generación del Lote
type CompilarLoteRequest struct {
	Anio         string `json:"anio" binding:"required"`
	Mes          string `json:"mes" binding:"required"`
	TipoServicio string `json:"tipo_servicio" binding:"required"`
	EsObjecion   bool   `json:"es_objecion"`
	FormatoSalida string `json:"formato_salida"` // "ISO" o "ZIP"
}
```

---

### 5. Lógica de Negocio y Algoritmos

```
[Usuario presiona "Auditar Lote para Cierre"]
       │
       ▼
[Paso 1: Auditoría Pre-flight de Cabecera y Expedientes]
 ├── ¿Existen 1. OFICIO DE PAGO.pdf, 2. PLANILLA CONSOLIDADA.pdf y 3. MATRIZ.xlsm en la raíz?
 ├── ¿Existen 0 archivos con prefijo PENDIENTE_ en 4. EXPEDIENTES/?
 └── ¿Todos los registros en Oracle están en PDI_ESTADO_DIGITALIZACION = 'COMPLETO'?
       │
       ├────► (NO) ──► Bloquear compilación / Desplegar alertas rojas en checklist
       │
       ▼ (SÍ)
[Paso 2: Inspección Técnica de Archivos PDF en Go (pdfcpu)]
 ├── Validar que todo archivo dentro de expedientes tenga formato PDF estricto
 ├── Comprobar resolución de imagen: 72 DPI <= Resolucion <= 300 DPI
 └── Verificar integridad del contenedor PDF (sin corrupción de streams)
       │
       ├────► (RECHAZO) ──► Listar archivos fuera de rango DPI en el reporte técnico
       │
       ▼ (Aprobado)
[Paso 3: Construcción del Árbol de Directorios Definitivo]
 ├── Trámite Primer Ingreso:
 │   └── [SERVICIO]_[MES]_[AÑO]/ ──► Cabecera + 4. EXPEDIENTES/[PACIENTE]/
 └── Trámite Levantamiento de Objeciones:
     └── [SERVICIO]_[MES]_[AÑO]_OBJECIONES/ ──► Cabecera + 4. EXPEDIENTES/ + 5. ANEXOS/
       │
       ▼
[Paso 4: Compilación de Imagen ISO / ZIP & Cálculo de Checksum]
       │
       ▼
[Paso 5: Actualizar Oracle DB (UPDATE PLANILLA_DIGITAL SET PDI_CERRADO = 'S')]
       │
       ▼
[Habilitar Botón "Descargar Imagen Oficial ISO" & Generar Etiqueta de Rotulado CD]
```

#### **Paso a Paso del Algoritmo:**

1. **Auditoría de Cabecera y Estado Global (`auditor_compleitud_service.go`):**
    
    - Go verifica la presencia física de los tres archivos habilitantes en la raíz de trabajo:
        - `1. OFICIO DE PAGO.pdf` (`M_PAGO.pdf`)
        - `2. PLANILLA CONSOLIDADA.pdf` (`C_CONSOLIDADA.pdf` o `P_CONSOLIDADA.pdf`)
        - `3. MATRIZ_[TIPO_SERVICIO]_[MES_AÑO].xlsm`
    - Consulta a la base de datos Oracle para verificar que el 100% de los expedientes del lote tengan `PDI_ESTADO_DIGITALIZACION = 'COMPLETO'`.
    - Escanea el sistema de archivos asegurando que no existan carpetas marcadas como `INCOMPLETO` ni archivos temporales etiquetados con `PENDIENTE_`.
2. **Inspección Técnica de Formato y Resolución (`auditor_pdf_tech_service.go`):**
    
    - Utilizando la librería `pdfcpu`, el backend recorre recursivamente todos los PDFs dentro de `4. EXPEDIENTES/` y `5. ANEXOS/`.
    - **Control de Formato y Calidad:** Confirma que la extensión sea estrictamente `.pdf` y que el archivo no presente corrupción de encabezados `%PDF-`.
    - **Control de Resolución DPI:** Mide el tamaño en píxeles y las dimensiones en puntos del PDF. Para una página A4 estándar (\(8.27 \times 11.69\) pulgadas), verifica que la resolución esté comprendida entre:
        - **Mínimo:** 72 DPI (\(595 \times 842\) px).
        - **Máximo:** 300 DPI (\(2480 \times 3508\) px).
    - Si un archivo está fuera de este rango de píxeles, lo reporta en el JSON de auditoría como un hallazgo técnico que debe corregirse.
3. **Estructuración del Paquete Oficial para CD/DVD (`iso_compiler_service.go`):**
    
    - Con las validaciones aprobadas, Go genera el directorio final respetando la nomenclatura oficial normada:
        - **Primer Ingreso:** `[TIPO_SERVICIO]_[MES]_[AÑO]/`.
        - **Objeciones:** `[TIPO_SERVICIO]_[MES]_[AÑO]_OBJECIONES/` (incluyendo la carpeta `5. ANEXOS/`).
    - Ejecuta el empaquetado creando una imagen ISO o archivo `.zip` utilizando la herramienta del sistema operativo (`mkisofs -V "MSP_DPSP" -o salida.iso /directorio_origen/`).
4. **Generación de Etiqueta de Rotulado Térmico (`rotulado_label_service.go`):**
    
    - Emite un conjunto de datos formateados para que la interfaz web renderice la carátula o etiqueta física del CD/DVD.
    - Incluye los campos normativos obligatorios: _Ministerio de Salud Pública - Dirección Provincial de Salud de Pichincha_, Razón Social y RUC del Prestador, Tipo de Servicio, Período Prestacional (Mes/Año), Indicación de volumen (ej. _Disco 1 de 1_) y Número de Trámite/Oficio.
5. **Cierre de Lote en Oracle DB (`oracle_cierre_repo.go`):**
    
    - Una vez empaquetado con éxito, ejecuta la sentencia de actualización masiva: `UPDATE DIGITALIZACION.PLANILLA_DIGITAL SET PDI_CERRADO = 'S' WHERE PDI_MES = :1 AND PDI_ANIO = :2 AND PDI_PLANILLADO = 'S'`.

---

### 6. Interfaz de Comunicación (APIs REST)

#### **1. Solicitar Auditoría Pre-Cierre (Pre-flight Check)**

- **Método / Ruta:** `GET /api/v1/empaquetado/auditar`
- **Query Params:** `anio=2026&mes=09&tipo_servicio=EMERGENCIA&es_objecion=false`
- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "periodo_mes": "09",
  "periodo_anio": "2026",
  "tipo_servicio": "EMERGENCIA",
  "es_objecion": false,
  "cabecera_valida": true,
  "oficio_presente": true,
  "consolidada_presente": true,
  "matriz_xlsm_presente": true,
  "total_expedientes": 150,
  "expedientes_ok": 150,
  "archivos_pendientes": 0,
  "errores_tecnicos": [],
  "apto_para_cierre": true
}
```

#### **2. Compilar Paquete Final ISO / ZIP**

- **Método / Ruta:** `POST /api/v1/empaquetado/compilar`
- **Payload de Entrada (JSON):**

```
{
  "anio": "2026",
  "mes": "09",
  "tipo_servicio": "EMERGENCIA",
  "es_objecion": false,
  "formato_salida": "ISO"
}
```

- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "status": "COMPLETED",
  "archivo_salida": "EMERGENCIA_SEPTIEMBRE_2026.iso",
  "tamanio_bytes": 412589000,
  "download_url": "/api/v1/downloads/EMERGENCIA_SEPTIEMBRE_2026.iso",
  "sha256_checksum": "a8f5f167f44f4964e6c998dee827110c",
  "mensaje": "Lote empaquetado exitosamente. Listo para grabación en 2 CD/DVDs."
}
```

#### **3. Obtener Metadatos para Rotulado Físico de CD/DVD**

- **Método / Ruta:** `GET /api/v1/empaquetado/rotulado-cd`
- **Query Params:** `anio=2026&mes=09&tipo_servicio=EMERGENCIA`
- **Respuesta Esperada (HTTP 200 OK):**

```
{
  "institucion_destino": "MSP - DIRECCIÓN PROVINCIAL DE SALUD DE PICHINCHA",
  "razon_social": "HOSPITAL CLINICA ESPECIALIZADA S.A.",
  "ruc_prestador": "1791234567001",
  "tipo_servicio": "SERVICIO DE EMERGENCIA",
  "periodo_prestacional": "SEPTIEMBRE 2026",
  "es_objecion": false,
  "numero_disco": "DISCO 1 DE 1",
  "numero_tramite_oficio": "OFICIO-HCE-2026-0941"
}
```

---

### Decisiones Pendientes

Para garantizar que el desarrollador pueda implementar la compilación y la entrega sin reuniones adicionales, se deben verificar los siguientes puntos con el equipo de infraestructura/sistemas:

1. **Ubicación y Permisos de la Herramienta de Creación ISO en Ubuntu:** Las fuentes establecen la generación de la imagen ISO para los CD/DVDs. _El desarrollador debe confirmar si el servidor Ubuntu cuenta con la utilidad `mkisofs` / `genisoimage` instalada en la ruta de ejecución `/usr/bin/mkisofs` o si debe utilizarse una librería pura de Go para compilar el archivo empaquetado `.zip` estructurado._
2. **Impresión de Rotulado Físico (Térmico Directo vs. PDF Imprimible):** La normativa prohíbe el uso de etiquetas de papel con cinta o goma húmeda sobre los discos. _Se debe definir si la interfaz web simplemente exportará una plantilla PDF con el troquel circular exacto para ser enviada a una impresora térmica de discos o si solo desplegará los campos de texto en pantalla._
