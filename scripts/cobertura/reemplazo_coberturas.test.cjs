// node --test scripts/cobertura/reemplazo_coberturas.test.cjs
"use strict";

const test = require("node:test");
const assert = require("node:assert");
const { detectOrigin, parseSheet, compareSheets } = require("./reemplazo_coberturas.cjs");

// Fragmentos con la forma real del texto de cada origen (pdftotext -layout), sin datos personales.
const LOCAL = `CONSULTA DE COBERTURA DE SALUD
   PACIENTE DE PRUEBA
   Número de documento de Identificación:          1712345678     Fecha de Cobertura de Seguro de Salud:          July 15, 2026
     IESS        tipo de afiliación no definida     afiliado no se encuentra activo     no registra cobertura
     ISSFA       no registra cobertura              cedula o pasaporte no registrado    no registra cobertura
     ISSPOL      no registra cobertura. ciudadano   no                                  no registra cobertura
  NO EXISTEN RESULTADOS PARA LOS PARÁMETROS INGRESADOS
                                    Av. Quitumbe Nan y Amaru Nan
                             Telf: 593 (2) 3814400 | www.msp.gob.ec`;

const PORTAL = `CONSULTA DE COBERTURA DE SALUD
PACIENTE DE PRUEBA
  Número de documento de Identificación: 1712345678        Fecha de Cobertura de Seguro de Salud: 15 July 2026
     IESS        afiliado seguro general tiempo completo   cédula encontrada en archivo hl   si registra cobertura
     ISSFA       no registra cobertura                     cedula o pasaporte no registrado  no registra cobertura
     ISSPOL      no registra cobertura. ciudadano no encon-    no                            no registra cobertura
  NO EXISTEN RESULTADOS PARA LOS PARÁMETROS INGRESADOS
                                    Av. Quitumbe Ñan y Amaru Ñan
                             Telf: 593 (2) 3814400 www.msp.gob.ec`;

test("detecta el origen por el texto, no por las fuentes", () => {
  assert.strictEqual(detectOrigin(LOCAL), "local");
  assert.strictEqual(detectOrigin(PORTAL), "portal");
});

test("el productor de un archivo sin fusionar debe coincidir con el texto", () => {
  assert.strictEqual(detectOrigin(LOCAL, "PDFKit"), "local");
  assert.strictEqual(detectOrigin(PORTAL, "react-pdf"), "portal");
  assert.strictEqual(detectOrigin(PORTAL, "PDFKit"), "desconocido");
  assert.strictEqual(detectOrigin(LOCAL, "react-pdf"), "desconocido");
});

test("sin señales o con señales contradictorias queda para revisión manual", () => {
  assert.strictEqual(detectOrigin("CONSULTA DE COBERTURA DE SALUD"), "desconocido");
  assert.strictEqual(detectOrigin(`${LOCAL}\nAmaru Ñan`), "desconocido");
});

test("lee cédula, fecha y resultado por seguro en ambos formatos", () => {
  const local = parseSheet(LOCAL);
  const portal = parseSheet(PORTAL);
  assert.deepStrictEqual([local.cedula, local.fecha], ["1712345678", "2026-07-15"]);
  assert.deepStrictEqual([portal.cedula, portal.fecha], ["1712345678", "2026-07-15"]);
  assert.deepStrictEqual(local.seguros, { IESS: "no", ISSFA: "no", ISSPOL: "no" });
  assert.deepStrictEqual(portal.seguros, { IESS: "si", ISSFA: "no", ISSPOL: "no" });
  assert.deepStrictEqual(compareSheets(local, portal).diffs, ["IESS: local=no portal=si"]);
});
