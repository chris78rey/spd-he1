Este documento de **Requerimientos Generales del Sistema** compila la arquitectura, reglas de negocio y estándares normativos exigidos por la Dirección Provincial de Salud de Pichincha (DPSP - MSP). Está diseñado para entregarse al líder de desarrollo o equipo de programación como marco sombrilla previo a la ejecución de cada módulo específico.

---

# Documento de Requerimientos Generales del Sistema (SRS)

**Proyecto:** Plataforma Automatizada de Ingesta, Clasificación Híbrida, Subsanación y Empaquetado de Expedientes Digitales para la ACFSS (DPSP - MSP)  
**Entidad Destino:** Dirección Provincial de Salud de Pichincha (Plataforma Gubernamental Sur, Quito).

> **Referencia funcional vigente:** Consultar [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md) para el catálogo de servicios, el árbol de primer ingreso, la estructura de objeciones y los nombres canónicos. En caso de diferencia con un resumen de este documento, seguir esa referencia detallada y confirmar cambios con el usuario.

---

### 1. Visión General y Objetivo del Sistema
El objetivo del sistema es transformar los lotes masivos de planillas facturadas exportados por sistemas hospitalarios externos (los cuales contienen únicamente carpetas numéricas con archivos PDF de nombres aleatorios) en expedientes digitales estructurados, validados y normados bajo el estándar oficial de la DPSP. 

El aplicativo debe orquestar la resolución de identidades vía Oracle DB, clasificar automáticamente los formularios médicos mediante lectura vectorial y OCR, gestionar la subsanación manual asistida, tramitar el levantamiento de objeciones y auditar la completitud del paquete antes de generar los medios ópticos finales (CD/DVD).

---

### 2. Arquitectura y Stack Tecnológico Aprobado

* **Frontend:** **Vuetify (Vue.js)** + **Axios** para la interfaz de usuario reactiva, la carga dual de archivos, la bandeja de control documental con visor dividido y la visualización de semáforos de completitud.
* **Backend:** **Go (Golang) 1.22+** como motor principal de procesamiento paralelo, utilizando un patrón **Worker Pool** concurrente.
* **Base de Datos:** **Oracle Database 11gR2**, accediendo a la tabla `DIGITALIZACION.PLANILLA_DIGITAL` mediante el driver **`godror`** y **Oracle Instant Client**.
* **Motor OCR & Herramientas PDF:** **Tesseract OCR** en español (`spa`) para escaneos planos (200–300 DPI) y biblioteca **`pdfcpu`** en Go para inspección, renombrado y concatenación (*merge*) de PDFs.
* **Procesamiento de Reglas & Hojas de Cálculo:** Archivo desacoplado **`reglas_clasificacion.yaml`** para detección de formularios y biblioteca **`excelize`** para la lectura/escritura de matrices de planillaje (`.xlsm`) e informes de liquidación (`I_LIQUIDACION`).

---

### 3. Requisitos Funcionales Generales (RF)

#### **RF-01: Ingesta Dual de Información**
* **Lote Clínico Masivo:** El sistema debe permitir la carga en bloque (archivo comprimido `.zip` o directorio de *staging*) con las carpetas numéricas por planilla (`1234560/`, `1234561/`) exportadas por el sistema hospitalario.
* **Documentos Habilitantes Macro:** En la misma pantalla, el sistema debe disponer de tres casilleros independientes obligatorios para adjuntar los documentos de cabecera:
  1. `1. OFICIO DE PAGO.pdf` (`M_PAGO.pdf`) firmado por el Representante Legal.
  2. `2. PLANILLA CONSOLIDADA.pdf` (`C_CONSOLIDADA.pdf` o `P_CONSOLIDADA.pdf`) con doble firma de responsabilidad (Médico Auditor y Financiero).
  3. `3. MATRIZ_[TIPO_SERVICIO]_[MES_AÑO].xlsm` (Matriz oficial Excel habilitada para macros).

#### **RF-02: Resolución de Identidades y Normalización de Nombres**
* **Precarga Masiva en Memoria:** Al iniciar el procesamiento mensual, el backend en Go debe realizar una consulta filtrada a Oracle DB (`WHERE PDI_ANIO = :1 AND PDI_MES = :2 AND PDI_PLANILLADO = 'S'`) y precargar un mapa en memoria (`map[int64]DatosPaciente`) para resolver identidades en tiempo constante \\(O(1)\\).
* **Saneamiento Normativo de Nombres:** Debe limpiar los nombres de los pacientes obtenidos de `PDI_PACIENTE` convirtiendo a mayúsculas, eliminando caracteres de puntuación iniciales (ej. `'. PRUEBA PACIENTE JUAN CARLOS'`), convirtiendo caracteres diacríticos a ASCII (`Á` \\(\rightarrow\\) `A`, `Ñ` \\(\rightarrow\\) `N`) y reemplazando espacios por guion bajo (`_`) para la creación de subcarpetas en `4. EXPEDIENTES/`.
* **Identidad de salida:** El nombre de carpeta usa únicamente `PDI_PACIENTE` normalizado. No agregar trámite, cédula/pasaporte ni etiquetas artificiales, incluidos pacientes con un solo apellido. Si falta el nombre o hay colisión, enviar a revisión.

#### **RF-03: Pipeline de Clasificación Híbrida y Fusión PDF**
* **Estrategia Híbrida (Vectorial vs. OCR):**
  * *Nivel 1:* Leer directamente el texto vectorial de las páginas 1 y 2 del PDF en memoria.
  * *Nivel 2 (Respaldo):* Si es un escaneo plano sin capa de texto, renderizar la página 1 a escala de grises (200–300 DPI) y aplicar Tesseract OCR (`spa`).
* **Evaluador YAML:** Contrastar el texto extraído contra las palabras clave definidas en `reglas_clasificacion.yaml` para asignar los códigos oficiales (`HCU_008.pdf`, `HCU_053.pdf`, `HCU_006.pdf`, `C_COBERTURA.pdf`, `P_INDIVIDUAL.pdf`, etc.).
* **Extensión obligatoria:** Todo PDF de salida debe terminar explícitamente en `.pdf`, incluidos los documentos clasificados, pendientes, habilitantes y anexos (por ejemplo, `HCU_008.pdf`).
* **Fusión de Documentos (*Merge*):** Si en la misma planilla de un paciente existen dos o más archivos asignados al mismo código normativo, Go debe unificarlos cronológicamente en un único PDF con `pdfcpu.Merge`.

#### **RF-04: Subsanación Manual y Control de Faltantes**
* **Archivos No Reconocidos:** Si un PDF no coincide con ninguna regla por problemas de legibilidad, Go no debe detener el lote; lo guardará como `PENDIENTE_tmp_[nombre].pdf` y actualizará el registro en Oracle a `PDI_ESTADO_DIGITALIZACION = 'REQUIERE_VALIDACION'`.
* **Visor Dividido:** El operador podrá examinar el PDF a la izquierda y asignarle el código normativo oficial desde un desplegable a la derecha para actualizar el expediente a `COMPLETO`.
* **Inyección Puntual:** Permitirá buscar a un paciente con estado `INCOMPLETO` y subir únicamente el formulario que le faltaba, renombrándolo e inyectándolo en su carpeta sin procesar todo el mes de nuevo.

#### **RF-05: Módulo de Gestión de Objeciones**
* **Ingesta del Reporte del MSP:** Leer la matriz Excel de liquidación (`I_LIQUIDACION`) enviada por los auditores de la DPSP.
* **Aislamiento Exclusivo:** Filtrar y bloquear automáticamente a los pacientes aprobados, desplegando en pantalla únicamente a los expedientes en controversia.
* **Tratamiento Documental:**
  * Marcar la posición explícita (**Acepta** o **Rechaza**) en la planilla individual (`P_INDIVIDUAL.pdf`).
  * Reemplazar PDFs corregidos en `4. EXPEDIENTES/`.
  * Ubicar justificativos técnicos/financieros (facturas, fichas técnicas, protocolos) en la carpeta exclusiva **`5. ANEXOS/[PACIENTE]/`**.

#### **RF-06: Guardián de Cierre y Empaquetado Normativo**
* **Auditoría Pre-flight:** Bloquear la compilación final si falta alguno de los 3 documentos de cabecera o si existen expedientes pendientes o incompletos en Oracle DB.
* **Control Técnico de PDF:** Verificar que el 100% de los archivos sean PDF válidos y que sus dimensiones correspondan a una resolución comprendida entre **72 DPI y 300 DPI**.
* **Generación de Paquete:** Compilar la estructura oficial para **Primer Ingreso** (`[SERVICIO]_[MES]_[AÑO]/`) o para **Objeciones** (`[SERVICIO]_[MES]_[AÑO]_OBJECIONES/` con la carpeta `5. ANEXOS/`), generando la imagen ISO / paquete `.zip` para la grabación en **dos (2) copias idénticas de CD/DVD**.
* **Rotulado Físico Térmico:** Generar los metadatos para la impresión térmica directa o carátula normalizada sobre el disco (prohibido el uso de etiquetas de papel adheridas con cinta/goma).

---

### 4. Requisitos No Funcionales (RNF)

* **RNF-01 (Rendimiento y Escalabilidad):** El motor en Go debe procesar miles de archivos en paralelo mediante un *Worker Pool* ajustable (4–8 *workers*), manteniendo un tiempo de procesamiento inferior a 60 segundos por cada lote de 200 planillas.
* **RNF-02 (Compatibilidad Oracle 11gR2):** El esquema DDL debe ser 100% compatible con Oracle 11gR2, utilizando secuencias (`SEQ_PLANILLA_DIGITAL`) y triggers PL/SQL (`BEFORE INSERT`) para autonuméricos, restricciones `CHECK CONSTRAINTS` e índices B-Tree estratégicos (`IDX_PDI_PERIODO`, `IDX_PDI_TRAMITE`, `IDX_PDI_ESTADO`).
* **RNF-03 (Validez de Firma Electrónica):** Soporte para la verificación de firmas electrónicas PAdES (ISO 32000-1 / SHA-256) emitidas por entidades acreditadas en Ecuador (FirmaEC / PKI).
* **RNF-04 (Resiliencia Documental):** Ante fallas de lectura física en ventanilla o corrupción del soporte óptico, el sistema debe permitir regenerar la imagen ISO de forma inmediata para cumplir con el plazo de subsanación de **10 días hábiles** establecido por el COA.
* **RNF-05 (Cumplimiento de Términos Legales):** Garantizar el flujo de control para que el levantamiento de objeciones se compile y radique dentro del término preclusivo improrrogable de **45 días hábiles** tras la notificación del MSP (Acuerdo Ministerial 00140-2023).

---

### 5. Resumen de Módulos Técnicos para el Equipo de Desarrollo

Para facilitar la asignación de tareas, el proyecto se encuentra dividido en **7 módulos independientes**:

1. **Frontend de Ingesta y Carga Dual:** Pantalla en Vuetify para recibir el `.zip` masivo y los casilleros para `M_PAGO.pdf`, `C_CONSOLIDADA.pdf` y la matriz `.xlsm`.
2. **Servicio de Persistencia Oracle DB 11gR2:** Conexión `godror`, caché masiva en memoria (`map[int64]PlanillaDTO`) y scripts DDL/Triggers/Índices.
3. **Bot Autónomo de Consulta de Coberturas:** Servicio desasistido para consultar portales institucionales (IESS/MSP) y descargar `C_COBERTURA.pdf`.
4. **Pipeline Backend Concurrente y Clasificador Híbrido:** Motor en Go con *Worker Pool*, Tesseract OCR (`spa`), evaluador YAML y fusión de PDFs (`pdfcpu`).
5. **Bandeja de Control Documental y Subsanación:** Visor dividido Vuetify para clasificación manual e inyección directa de formularios faltantes.
6. **Módulo de Gestión de Objeciones:** Parser de `I_LIQUIDACION`, aislamiento de expedientes observados, generación de `P_INDIVIDUAL.pdf` (Acepta/Rechaza) e inyección en `5. ANEXOS/`.
7. **Motor de Validación y Guardián de Empaquetado:** Auditoría pre-cierre (completitud, 72–300 DPI), compilador de imagen ISO/ZIP para 2 CD/DVDs y plantilla de rotulado térmico.
