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
**Componente:** **Servicio de Persistencia e Integración Oracle DB 11gR2**  
**Rol:** Arquitecto de Software & Tech Lead

---

### 1. Stack Tecnológico y Dependencias

Basado estrictamente en las fuentes del proyecto, el stack de persistencia e integración identificado comprende:

- **Motor de Base de Datos:** **Oracle Database 11g Release 2 (11gR2)**.
- **Lenguaje de Programación Backend:** **Go (Golang)** versión 1.22 o superior.
- **Driver de Conexión a Base de Datos:** **`godror`** (Go DRiver for ORacle), utilizando la biblioteca nativa **Oracle Instant Client** instalada en el sistema operativo Ubuntu.
- **Abstracción SQL:** Paquete estándar `database/sql` de Go combinado con consultas nativas parametrizadas para Oracle.
- **Componentes BD de Soporte:** Secuencias (`SEQUENCE`), Triggers PL/SQL (`BEFORE INSERT`), Restricciones (`CHECK CONSTRAINTS`) e Índices B-Tree de rendimiento.

---

### 2. Contexto y Flujo de Datos

#### **Problema que resuelve:**

El sistema hospitalario emisor del prestador exporta carpetas numéricas asociadas al número de trámite o planilla (`PDI_TRAMITE`), pero con archivos PDF internos cuyos nombres son aleatorios o temporales (`tmp_a1.pdf`, `doc_99.pdf`).

Este módulo resuelve la **resolución de identidades y la persistencia del estado documental**:

1. Consulta en la tabla `DIGITALIZACION.PLANILLA_DIGITAL` los datos del paciente (`PDI_PACIENTE`, `PDI_CEDULA`, `PDI_HC`), la especialidad (`PDI_SERVICIO` / `PDI_COD_SERVICIO`) y los flags de cobertura.
2. Resuelve el cuello de botella de rendimiento mediante una **precarga masiva en memoria (`map[int]PlanillaDTO`)** filtrada por mes, año, bandera de planillado activo (`PDI_PLANILLADO = 'S'`) y aseguradora MSP (`PDI_ASEGURADORA = 'MSP'`). Esto permite al motor en Go consultar datos en tiempo constante \(O(1)\) sin saturar la base de datos con miles de conexiones concurrentes.
3. Persiste y actualiza las transiciones de estado de digitalización (`PDI_ESTADO_DIGITALIZACION`, `PDI_PROCESADO`, `PDI_FECHA_PROCESO`, `PDI_OBJETADO`) a lo largo del pipeline.

#### **Flujos de Entrada (Inputs):**

- **Parámetros de Período:** Año (`PDI_ANIO`) y Mes (`PDI_MES`) recibidos desde el controlador de ingesta.
- **Identificador de Trámite:** Número de carpeta numérica (`PDI_TRAMITE`) procesada por los _workers_ en Go.
- **Actualizaciones de Pipeline:** Cambios de estado generados tras la clasificación automática, la subsanación manual o la ingesta de objeciones.

#### **Flujos de Salida (Outputs):**

- **Diccionario en Memoria (`CacheMap`):** Mapa indexado por `PDI_TRAMITE` retornado a los _workers_ para resolver nombres de carpetas (`4. EXPEDIENTES/[PACIENTE_SANEADO]/`).
- **Nombre de salida:** Resolver `PDI_TRAMITE` a `PDI_PACIENTE`; usar solo el nombre normalizado de este último, según [07_reglas_estructura_acfss.md](07_reglas_estructura_acfss.md). No añadir cédula, pasaporte ni trámite al nombre de carpeta.
- **Persistencia en Oracle:** Registros actualizados en la tabla `DIGITALIZACION.PLANILLA_DIGITAL`.

---

### 3. Estructura de Directorios Sugerida

Siguiente las mejores prácticas de arquitectura en Go y separación de capas:

```
service-persistence-oracle/
├── config/
│   └── oracle_config.go          # Configuración de DSN y Pool de Conexiones (godror)
├── internal/
│   ├── domain/
│   │   ├── planilla_dto.go       # DTO y estructura del modelo de datos de Oracle
│   │   └── errors.go             # Definición de errores del dominio de datos
│   ├── repository/
│   │   ├── oracle_repository.go  # Repositorio SQL con consultas y actualizaciones
│   │   └── mapper.go             # Mapeo de hileras SQL a DTO y saneamiento de texto
│   └── service/
│       └── cache_service.go      # Servicio de precarga masiva en memoria (Sync Map)
├── scripts/
│   ├── 01_ddl_planilla_digital.sql # Definición de la tabla
│   ├── 02_pk_sequence_trigger.sql  # Autonumérico para Oracle 11gR2
│   ├── 03_constraints_check.sql    # Restricciones de integridad
│   └── 04_indices.sql              # Índices para optimización de consultas
├── go.mod
└── go.sum
```

---

### 4. Modelos de Datos y Entidades

#### **A. Definición DDL de la Tabla (`DIGITALIZACION.PLANILLA_DIGITAL`)**

```
CREATE TABLE DIGITALIZACION.PLANILLA_DIGITAL
(
  PDI_ID                     NUMBER,
  PDI_TRAMITE                NUMBER,
  PDI_COD_ASEGURADORA        VARCHAR2(100 BYTE),
  PDI_ASEGURADORA            VARCHAR2(100 BYTE),
  PDI_FECHA_DESDE            DATE,
  PDI_FECHA_HASTA            DATE,
  PDI_HC                     NUMBER,
  PDI_PACIENTE               VARCHAR2(1000 BYTE),
  PDI_CEDULA                 VARCHAR2(20 BYTE),
  PDI_MENOR_EDAD             CHAR(1 BYTE)       DEFAULT 'N',
  PDI_DEPENDIENTE_01         VARCHAR2(20 BYTE),
  PDI_DEPENDIENTE_02         VARCHAR2(20 BYTE),
  PDI_COD_SERVICIO           VARCHAR2(10 BYTE),
  PDI_SERVICIO               VARCHAR2(100 BYTE),
  PDI_MES                    VARCHAR2(30 BYTE),
  PDI_ANIO                   VARCHAR2(4 BYTE),
  PDI_CERRADO                VARCHAR2(10 BYTE)  DEFAULT 'N',
  PDI_PLANILLADO             CHAR(1 BYTE)       DEFAULT 'S',
  PDI_COBERTURA              VARCHAR2(1 BYTE)   DEFAULT 'N',
  PDI_ID_GENERACION          NUMBER,
  PDI_USUARIO                VARCHAR2(100 BYTE),
  PDI_CODIGO_USUARIO         VARCHAR2(5 BYTE),
  PDI_FECHA_ALTA             DATE,
  PDI_NUMERO_PERMANENCIA     NUMBER,
  PDI_PROCESADO              CHAR(1 BYTE)       DEFAULT 'N',
  PDI_FECHA_PROCESO          DATE,
  PDI_FECHA_NACIMIENTO       DATE,
  PDI_ESTADO_DIGITALIZACION  VARCHAR2(100 BYTE),
  PDI_OBJETADO               VARCHAR2(1 BYTE)
);
```

#### **B. Implementación de Autonumérico (Compatibilidad Oracle 11gR2)**

Puesto que Oracle 11gR2 no soporta columnas `IDENTITY`, se implementa una secuencia y un trigger `BEFORE INSERT`:

```
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL
  ADD CONSTRAINT PK_PLANILLA_DIGITAL PRIMARY KEY (PDI_ID);

CREATE SEQUENCE DIGITALIZACION.SEQ_PLANILLA_DIGITAL
  START WITH 1
  INCREMENT BY 1
  NOCACHE
  NOCYCLE;

CREATE OR REPLACE TRIGGER DIGITALIZACION.TRG_PLANILLA_DIGITAL_BI
BEFORE INSERT ON DIGITALIZACION.PLANILLA_DIGITAL
FOR EACH ROW
BEGIN
  IF :NEW.PDI_ID IS NULL THEN
    SELECT DIGITALIZACION.SEQ_PLANILLA_DIGITAL.NEXTVAL INTO :NEW.PDI_ID FROM DUAL;
  END IF;
END;
/
```

#### **C. Restricciones de Integridad (`CHECK CONSTRAINTS`)**

```
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_MENOR_EDAD CHECK (PDI_MENOR_EDAD IN ('S', 'N'));
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_PLANILLADO CHECK (PDI_PLANILLADO IN ('S', 'N'));
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_COBERTURA CHECK (PDI_COBERTURA IN ('S', 'N'));
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_PROCESADO CHECK (PDI_PROCESADO IN ('S', 'N'));
ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_OBJETADO CHECK (PDI_OBJETADO IN ('S', 'N') OR PDI_OBJETADO IS NULL);

ALTER TABLE DIGITALIZACION.PLANILLA_DIGITAL ADD CONSTRAINT CHK_PDI_ESTADO CHECK (
  PDI_ESTADO_DIGITALIZACION IN (
    'PENDIENTE', 'PROCESANDO', 'REQUIERE_VALIDACION', 'INCOMPLETO', 'COMPLETO', 'OBJETADO'
  ) OR PDI_ESTADO_DIGITALIZACION IS NULL
);
```

#### **D. Índices de Rendimiento**

```
CREATE INDEX DIGITALIZACION.IDX_PDI_PERIODO ON DIGITALIZACION.PLANILLA_DIGITAL (PDI_ANIO, PDI_MES, PDI_PLANILLADO);
CREATE INDEX DIGITALIZACION.IDX_PDI_TRAMITE ON DIGITALIZACION.PLANILLA_DIGITAL (PDI_TRAMITE);
CREATE INDEX DIGITALIZACION.IDX_PDI_CEDULA ON DIGITALIZACION.PLANILLA_DIGITAL (PDI_CEDULA);
CREATE INDEX DIGITALIZACION.IDX_PDI_HC ON DIGITALIZACION.PLANILLA_DIGITAL (PDI_HC);
CREATE INDEX DIGITALIZACION.IDX_PDI_ESTADO ON DIGITALIZACION.PLANILLA_DIGITAL (PDI_ESTADO_DIGITALIZACION, PDI_OBJETADO);
```

#### **E. Estructura de Datos en Go (`domain/planilla_dto.go`)**

```
package domain

import "time"

type PlanillaDTO struct {
	ID                   int64     `json:"pdi_id"`
	Tramite              int64     `json:"pdi_tramite"`
	Aseguradora          string    `json:"pdi_aseguradora"`
	HC                   int64     `json:"pdi_hc"`
	PacienteRaw          string    `json:"pdi_paciente_raw"`
	PacienteSaneado      string    `json:"pdi_paciente_saneado"`
	Cedula               string    `json:"pdi_cedula"`
	MenorEdad            string    `json:"pdi_menor_edad"`
	Dependiente01        string    `json:"pdi_dependiente_01"`
	CodServicio          string    `json:"pdi_cod_servicio"`
	Servicio             string    `json:"pdi_servicio"`
	Mes                  string    `json:"pdi_mes"`
	Anio                 string    `json:"pdi_anio"`
	Planillado           string    `json:"pdi_planillado"`
	Procesado            string    `json:"pdi_procesado"`
	FechaProceso         time.Time `json:"pdi_fecha_proceso"`
	EstadoDigitalizacion string    `json:"pdi_estado_digitalizacion"`
	Objetado             string    `json:"pdi_objetado"`
}
```

---

### 5. Lógica de Negocio y Algoritmos

```
[Inicializar Pool godror]
       │
       ▼
[Petición de Precarga Masiva: FetchMonthlyCache(anio, mes)]
       │
       ▼
[SQL Query]
SELECT PDI_ID, PDI_TRAMITE, PDI_PACIENTE, PDI_CEDULA, PDI_HC,
       PDI_COD_SERVICIO, PDI_SERVICIO, PDI_MENOR_EDAD, PDI_DEPENDIENTE_01, PDI_OBJETADO
FROM DIGITALIZACION.PLANILLA_DIGITAL
WHERE PDI_ANIO = :1 AND PDI_MES = :2 AND PDI_PLANILLADO = 'S' AND PDI_ASEGURADORA = 'MSP'
       │
       ▼
[Iterar Hileras SQL & Aplicar Sanitización de Nombres en Go]
 ├── ToUpper()
 ├── Eliminar puntos/caracteres iniciales (TrimLeft)
 ├── Normalizar diacríticos (Á->A, Ñ->N)
 ├── Conflicto o identidad sin nombre útil: derivar a revisión; no alterar PDI_PACIENTE con datos adicionales
 └── Sustituir espacios por '_'
       │
       ▼
[Poblar Map Sync en Memoria: map[int64]PlanillaDTO]
       │
       ▼
[Worker Pool consulta en O(1) por PDI_TRAMITE] ──► Devuelve Ruta Normada
       │
       ▼
[Actualizar Estado en Oracle tras procesar archivo]
UPDATE PLANILLA_DIGITAL
SET PDI_ESTADO_DIGITALIZACION = :1, PDI_PROCESADO = 'S', PDI_FECHA_PROCESO = SYSDATE
WHERE PDI_ID = :2
```

#### **Paso a Paso del Algoritmo:**

1. **Inicialización y Conexión (`oracle_config.go`):**
    
    - Configura la cadena de conexión DSN para Oracle utilizando `godror` (ej. `user/password@host:1521/service_name`).
    - Define los límites del Pool de Conexiones (`SetMaxOpenConns(20)`, `SetMaxIdleConns(5)`, `SetConnMaxLifetime(15 * time.Minute)`).
2. **Precarga Masiva en Memoria (`cache_service.go`):**
    
    - Al iniciar el lote de un mes prestacional, ejecuta la consulta SQL filtrada por `PDI_ANIO`, `PDI_MES`, `PDI_PLANILLADO = 'S'` y `PDI_ASEGURADORA = 'MSP'`.
    - Lee secuencialmente las filas y procesa el campo `PDI_PACIENTE` con la función de saneamiento:
        - Convierte a mayúsculas.
        - Elimina puntos o espacios iniciales (ej. `'. PRUEBA PACIENTE JUAN CARLOS'` \(\rightarrow\) `'PRUEBA PACIENTE JUAN CARLOS'`).
        - Convierte caracteres diacríticos a ASCII (`Á` \(\rightarrow\) `A`, `Ñ` \(\rightarrow\) `N`).
        - El nombre del directorio se deriva únicamente de `PDI_PACIENTE`; los identificadores personales no se añaden al nombre de salida (ver `07_reglas_estructura_acfss.md`).
    - Construye el DTO y lo inserta en la estructura `map[int64]PlanillaDTO` indexada por `PDI_TRAMITE`.
3. **Consulta Instantánea de Identidad \(O(1)\):**
    
    - Cuando un _worker_ en Go procesa la carpeta `/staging/1234560/`, consulta `cacheMap`.
    - Obtiene de forma inmediata `PacienteSaneado` para construir la ruta `/workspace/4. EXPEDIENTES/PRUEBA_PACIENTE_JUAN_CARLOS/`.
4. **Sincronización de Estado y Transacciones (`oracle_repository.go`):**
    
    - Conforme los expedientes avanzan o requieren validación manual, ejecuta actualizaciones directas a la base de datos:
        - `UPDATE PLANILLA_DIGITAL SET PDI_ESTADO_DIGITALIZACION = 'COMPLETO', PDI_PROCESADO = 'S', PDI_FECHA_PROCESO = SYSDATE WHERE PDI_ID = :1`.

---

### 6. Interfaz de Comunicación (APIs / Métodos Go)

Este módulo expone una interfaz nativa en Go (`PlanillaRepositoryInterface`) para ser consumida por los demás componentes del backend:

#### **A. Firma de la Interfaz Go (`repository/oracle_repository.go`)**

```
package repository

import (
	"context"
	"service-persistence-oracle/internal/domain"
)

type PlanillaRepositoryInterface interface {
	// Carga masiva en memoria de las planillas del mes
	FetchMonthlyCache(ctx context.Context, anio string, mes string) (map[int64]domain.PlanillaDTO, error)

	// Consulta puntual por trámite (fallback fuera de memoria)
	GetByTramite(ctx context.Context, tramite int64) (*domain.PlanillaDTO, error)

	// Actualización de estado del pipeline de digitalización
	UpdateEstadoDigitalizacion(ctx context.Context, pdiID int64, estado string, procesado string) error

	// Marca masiva de pacientes objetados tras cargar el Excel del MSP
	UpdateObjetadosBatch(ctx context.Context, tramiteIDs []int64) error
}
```

#### **B. Contratos de Datos Internos**

- **Entrada `FetchMonthlyCache`:** `anio` (`"2026"`), `mes` (`"09"`).
    
- **Retorno `FetchMonthlyCache`:** `map[int64]domain.PlanillaDTO` indexado por `PDI_TRAMITE`.
    
- **Entrada `UpdateEstadoDigitalizacion`:**
    
    - `pdiID`: Identificador único (`PDI_ID`).
    - `estado`: Uno de `'PENDIENTE'`, `'PROCESANDO'`, `'REQUIERE_VALIDACION'`, `'INCOMPLETO'`, `'COMPLETO'`, `'OBJETADO'`.
    - `procesado`: `'S'` o `'N'`.
- **Sentencia SQL Ejecutada en `UpdateObjetadosBatch`:**
    

```
UPDATE DIGITALIZACION.PLANILLA_DIGITAL
SET PDI_OBJETADO = 'S', PDI_ESTADO_DIGITALIZACION = 'OBJETADO'
WHERE PDI_TRAMITE = :1;
```

---

### Decisiones Pendientes

1. **Configuración de la Variable `LD_LIBRARY_PATH` para Oracle Instant Client:** El desarrollador debe verificar que en la imagen Docker o servidor Ubuntu donde corra el binario compilado en Go se encuentre exportada la ruta de bibliotecas nativas de C para `godror` (ej. `export LD_LIBRARY_PATH=/usr/lib/oracle/19.21/client64/lib`).
2. **Estrategia de Re-sincronización de Caché (_Cache Invalidation_):** Si mientras el lote se procesa el área de Facturación inserta nuevos registros en la tabla `PLANILLA_DIGITAL`, se debe acordar si el backend debe refrescar el mapa en memoria de forma periódica o mediante un evento de re-sincronización manual.
