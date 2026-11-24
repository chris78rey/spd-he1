/*
  PLANILLA_DIGITAL: job automático + sincronización de fechas desde SIS
  ---------------------------------------------------------------------
  Para revisión y ejecución del DBA. Oracle 11.2.0.4.
  Ejecutar conectado como DIGITALIZACION (el job queda a nombre de ese usuario,
  igual que los jobs 5343, 5364, 5406 y 5416 existentes).

  Contexto
  - Ningún job ni objeto llama hoy a DIGITALIZACION.PCK_GENERAR_DIGITALIZACION;
    se ha ejecutado a mano (cargas del 08/10, 13/10, 23/10 y 24/11/2026).
  - El paquete (y la creación de planillas de la aplicación Folio) toma
    PDI_FECHA_DESDE / PDI_FECHA_HASTA de MIN(ROWID) / MAX(ROWID) de SIS.CUENTAS.
    El ROWID no sigue el orden cronológico, así que hay fechas equivocadas
    (p. ej. trámite 6093842: HASTA = 07/07/2026, última atención real 24/07/2026;
    trámite 6093852: DESDE 31/07 y HASTA 01/07). Al 24/11/2026: 147 registros.
  - La hoja de cobertura se consulta con PDI_FECHA_HASTA.
  - Este script replica para PLANILLA_DIGITAL lo que PR_SINCRONIZA_FECHAS_SIS ya
    hace para DIGITALIZACION.DIGITALIZACION: recalcula fechas con MIN/MAX(fecha).

  Alcance de la sincronización
  - Cambia solo PDI_FECHA_DESDE y PDI_FECHA_HASTA.
  - NO cambia PDI_MES / PDI_ANIO (mover un trámite de período requiere revisión).
  - Solo registros con PDI_CERRADO = 'N'.
  - Usa las cuentas no anuladas (ESTADO NOT LIKE 'AN%') de la misma promoción
    (PRM_CODIGO = PDI_COD_ASEGURADORA), igual que el paquete.

  Secciones
    1. Vista previa (solo lectura)
    2. Respaldo de las fechas actuales
    3. Procedimiento PR_SINCRONIZA_FECHAS_PDI
    4. Procedimiento PR_EJECUTA_PLANILLA_DIGITAL (paquete + sincronización)
    5. Primera ejecución manual y verificación
    6. Programar el job
    7. Verificar el job
    8. Reversión
*/

------------------------------------------------------------------------------
-- 1. VISTA PREVIA (solo lectura): registros que cambiarían
------------------------------------------------------------------------------
SELECT p.pdi_id,
       p.pdi_tramite,
       p.pdi_cod_aseguradora,
       p.pdi_cod_servicio,
       p.pdi_anio || '-' || p.pdi_mes         periodo,
       p.pdi_fecha_desde                      desde_actual,
       s.sis_desde                            desde_nueva,
       p.pdi_fecha_hasta                      hasta_actual,
       s.sis_hasta                            hasta_nueva,
       p.pdi_cobertura,
       CASE
          WHEN TO_CHAR (s.sis_desde, 'YYYYMM') <> p.pdi_anio || p.pdi_mes
            OR TO_CHAR (s.sis_hasta, 'YYYYMM') <> p.pdi_anio || p.pdi_mes
          THEN 'REVISAR: fechas fuera del mes del registro'
       END                                    observacion,
       p.pdi_paciente
  FROM digitalizacion.planilla_digital p
       JOIN (  SELECT cta.pla_numero_planilla,
                      cta.prm_codigo,
                      MIN (TRUNC (cta.fecha)) sis_desde,
                      MAX (TRUNC (cta.fecha)) sis_hasta
                 FROM sis.cuentas cta
                WHERE cta.estado NOT LIKE 'AN%'
                  AND cta.pla_numero_planilla IN
                         (SELECT pdi_tramite
                            FROM digitalizacion.planilla_digital
                           WHERE pdi_cerrado = 'N')
             GROUP BY cta.pla_numero_planilla, cta.prm_codigo) s
          ON s.pla_numero_planilla = p.pdi_tramite
         AND s.prm_codigo = p.pdi_cod_aseguradora
 WHERE p.pdi_cerrado = 'N'
   AND (   NVL (p.pdi_fecha_desde, DATE '1900-01-01') <> s.sis_desde
        OR NVL (p.pdi_fecha_hasta, DATE '1900-01-01') <> s.sis_hasta)
 ORDER BY p.pdi_anio, p.pdi_mes, p.pdi_tramite;

------------------------------------------------------------------------------
-- 2. RESPALDO de las fechas actuales (antes de la primera sincronización)
------------------------------------------------------------------------------
CREATE TABLE digitalizacion.pdi_fechas_resp_20261124
AS
   SELECT pdi_id,
          pdi_tramite,
          pdi_fecha_desde,
          pdi_fecha_hasta,
          SYSDATE fecha_respaldo
     FROM digitalizacion.planilla_digital;

SELECT COUNT (*) FROM digitalizacion.pdi_fechas_resp_20261124;

------------------------------------------------------------------------------
-- 3. SINCRONIZACIÓN DE FECHAS
------------------------------------------------------------------------------
CREATE OR REPLACE PROCEDURE digitalizacion.pr_sincroniza_fechas_pdi
AS
   v_registros   NUMBER := 0;
BEGIN
   MERGE INTO digitalizacion.planilla_digital d
        USING (  SELECT p.pdi_id,
                        MIN (TRUNC (cta.fecha)) sis_desde,
                        MAX (TRUNC (cta.fecha)) sis_hasta
                   FROM digitalizacion.planilla_digital p
                        JOIN sis.cuentas cta
                           ON cta.pla_numero_planilla = p.pdi_tramite
                          AND cta.prm_codigo = p.pdi_cod_aseguradora
                          AND cta.estado NOT LIKE 'AN%'
                  WHERE p.pdi_cerrado = 'N'
               GROUP BY p.pdi_id) s
           ON (d.pdi_id = s.pdi_id)
   WHEN MATCHED
   THEN
      UPDATE SET d.pdi_fecha_desde = s.sis_desde,
                 d.pdi_fecha_hasta = s.sis_hasta
         WHERE d.pdi_cerrado = 'N'
           AND (   NVL (d.pdi_fecha_desde, DATE '1900-01-01') <> s.sis_desde
                OR NVL (d.pdi_fecha_hasta, DATE '1900-01-01') <> s.sis_hasta);

   v_registros := SQL%ROWCOUNT;
   COMMIT;
   DBMS_OUTPUT.put_line ('PLANILLA_DIGITAL: fechas sincronizadas desde SIS: ' || v_registros);
EXCEPTION
   WHEN OTHERS
   THEN
      ROLLBACK;
      raise_application_error (-20011, 'Error en PR_SINCRONIZA_FECHAS_PDI: ' || SQLERRM);
END pr_sincroniza_fechas_pdi;
/

SHOW ERRORS PROCEDURE digitalizacion.pr_sincroniza_fechas_pdi

------------------------------------------------------------------------------
-- 4. ORQUESTADOR: carga del paquete + sincronización de fechas
--    p_genera_planilla no hace COMMIT propio y los COMMIT posteriores del
--    paquete solo ocurren si sus cursores encuentran filas; por eso se confirma
--    aquí de forma explícita.
--    Las llamadas van SIN prefijo de esquema: dentro de DIGITALIZACION existe la
--    tabla DIGITALIZACION.DIGITALIZACION y PL/SQL resolvería "digitalizacion.x"
--    como esa tabla (PLS-00302). Los jobs existentes también llaman sin prefijo.
------------------------------------------------------------------------------
CREATE OR REPLACE PROCEDURE digitalizacion.pr_ejecuta_planilla_digital
AS
BEGIN
   pck_generar_digitalizacion.p_ejecutar;
   COMMIT;

   pr_sincroniza_fechas_pdi;
EXCEPTION
   WHEN OTHERS
   THEN
      ROLLBACK;
      raise_application_error (-20012, 'Error en PR_EJECUTA_PLANILLA_DIGITAL: ' || SQLERRM);
END pr_ejecuta_planilla_digital;
/

SHOW ERRORS PROCEDURE digitalizacion.pr_ejecuta_planilla_digital

------------------------------------------------------------------------------
-- 5. PRIMERA EJECUCIÓN MANUAL Y VERIFICACIÓN
------------------------------------------------------------------------------
-- Antes de ejecutar, debe estar VALID (si no, revisar DBA_ERRORS):
SELECT object_name, status
  FROM dba_objects
 WHERE owner = 'DIGITALIZACION'
   AND object_name IN ('PR_SINCRONIZA_FECHAS_PDI', 'PR_EJECUTA_PLANILLA_DIGITAL');

-- Conectado como DIGITALIZACION, sin prefijo de esquema (ver nota de la sección 4):
SET SERVEROUTPUT ON
EXEC pr_ejecuta_planilla_digital;

-- Debe devolver 0 filas (todas las fechas coinciden con SIS):
SELECT p.pdi_tramite, p.pdi_fecha_desde, s.sis_desde, p.pdi_fecha_hasta, s.sis_hasta
  FROM digitalizacion.planilla_digital p
       JOIN (  SELECT pla_numero_planilla, prm_codigo,
                      MIN (TRUNC (fecha)) sis_desde, MAX (TRUNC (fecha)) sis_hasta
                 FROM sis.cuentas
                WHERE estado NOT LIKE 'AN%'
             GROUP BY pla_numero_planilla, prm_codigo) s
          ON s.pla_numero_planilla = p.pdi_tramite
         AND s.prm_codigo = p.pdi_cod_aseguradora
 WHERE p.pdi_cerrado = 'N'
   AND (p.pdi_fecha_desde <> s.sis_desde OR p.pdi_fecha_hasta <> s.sis_hasta);

-- Casos de control (esperado: 6093842 HASTA 24/07/2026; 6093852 DESDE 01/07 y HASTA 31/07):
SELECT pdi_tramite, pdi_fecha_desde, pdi_fecha_hasta
  FROM digitalizacion.planilla_digital
 WHERE pdi_tramite IN (6093842, 6093852, 6093871, 6106155, 6093848);

------------------------------------------------------------------------------
-- 6. PROGRAMAR EL JOB (cada 30 minutos, mismo estilo que los jobs existentes)
--    El paquete recorre SIS.CUENTAS desde el 01/07/2026 con TRUNC(fecha), que
--    no usa índice; si preocupa la carga, cambiar a 'SYSDATE + 1/24' (cada hora).
------------------------------------------------------------------------------
VARIABLE v_job NUMBER
BEGIN
   DBMS_JOB.submit (
      job         => :v_job,
      what        => 'PR_EJECUTA_PLANILLA_DIGITAL;',
      next_date   => SYSDATE + 5 / 1440,
      interval    => 'SYSDATE + 30/1440');
   COMMIT;
END;
/
PRINT v_job

------------------------------------------------------------------------------
-- 7. VERIFICAR EL JOB
------------------------------------------------------------------------------
SELECT job, log_user, what, last_date, next_date, interval, broken, failures
  FROM dba_jobs
 WHERE UPPER (what) LIKE '%PR_EJECUTA_PLANILLA_DIGITAL%';

-- Cargas recientes del paquete:
  SELECT TRUNC (pdi_fecha_proceso) dia, pdi_cod_aseguradora, COUNT (*)
    FROM digitalizacion.planilla_digital
   WHERE pdi_fecha_proceso >= TRUNC (SYSDATE) - 7
GROUP BY TRUNC (pdi_fecha_proceso), pdi_cod_aseguradora
ORDER BY 1, 2;

------------------------------------------------------------------------------
-- 8. REVERSIÓN (solo si hace falta)
------------------------------------------------------------------------------
-- 8.1 Quitar el job (reemplazar <JOB> por el número de la sección 7):
-- EXEC DBMS_JOB.remove (<JOB>); COMMIT;

-- 8.2 Restaurar las fechas anteriores:
-- MERGE INTO digitalizacion.planilla_digital d
--      USING digitalizacion.pdi_fechas_resp_20261124 r
--         ON (d.pdi_id = r.pdi_id)
-- WHEN MATCHED THEN
--    UPDATE SET d.pdi_fecha_desde = r.pdi_fecha_desde,
--               d.pdi_fecha_hasta = r.pdi_fecha_hasta;
-- COMMIT;

-- 8.3 Eliminar los procedimientos:
-- DROP PROCEDURE digitalizacion.pr_ejecuta_planilla_digital;
-- DROP PROCEDURE digitalizacion.pr_sincroniza_fechas_pdi;
