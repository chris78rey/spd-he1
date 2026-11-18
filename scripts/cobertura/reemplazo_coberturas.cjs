#!/usr/bin/env node
// Prepara el reemplazo de hojas de cobertura generadas localmente por el PDF real del portal MSP.
//
// Fase A (este script): NO modifica expedientes, job.json ni Oracle. Para un período:
//   1. Encuentra cada C_COBERTURA*.pdf del expediente y detecta el origen de cada página por su
//      texto: el generador local escribía la fecha como "July 15, 2026" y el pie "Amaru Nan" con
//      "3814400 |"; el portal escribe "15 July 2026" y "Amaru Ñan". En archivos sin fusionar se
//      contrasta con el productor (PDFKit = local, react-pdf = portal). Las fuentes NO sirven:
//      el portal también usa Helvetica/Times. Si las señales se contradicen, la página se revisa a mano.
//   2. Lee de cada página local su cédula y su fecha de cobertura.
//   3. Descarga del portal la hoja de cada página (download_site_pdf.cjs), en el mismo orden.
//   4. Compara, seguro por seguro, "si/no registra cobertura" entre la hoja local y la del portal.
//   5. Arma la hoja candidata (misma cantidad y orden de páginas) y escribe un reporte.
//
// Uso:
//   Fase A (preparar):  node scripts/cobertura/reemplazo_coberturas.cjs --periodo HOSPITALIZACION_SEPTIEMBRE_2026 [--salida DIR]
//   Fase B (aplicar):   node scripts/cobertura/reemplazo_coberturas.cjs --aplicar DIR_PREPARADO
// Reutilizar --salida reanuda: las páginas ya descargadas y válidas no se vuelven a consultar.
// --aplicar envía al servidor (cola <data>/reemplazo_coberturas/solicitudes) las hojas LISTA y REVISAR
// del reporte. El servidor las reemplaza con su propio candado, respalda la hoja anterior y
// sincroniza Oracle; este script espera y muestra el resultado.
// Códigos de salida: 0 terminado · 1 error · 3 el portal aplicó su límite de consultas (reanudar luego).

"use strict";

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execFileSync, spawnSync } = require("child_process");
const { loadEnvFile } = require("./runtime_config.cjs");

loadEnvFile();

const INSURERS = ["IESS", "ISSFA", "ISSPOL"];
const MONTHS = ["january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"];
// El portal escribe el mes en español cuando la fecha se elige en su calendario ("01 octubre 2026").
const MESES = ["enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"];
const DOWNLOADER = path.join(__dirname, "download_site_pdf.cjs");

function parseArgs(argv) {
  const args = {};
  for (let i = 2; i < argv.length; i += 2) {
    if (argv[i].startsWith("--")) {
      args[argv[i].slice(2)] = argv[i + 1];
    }
  }
  return args;
}

function run(cmd, args) {
  return execFileSync(cmd, args, { encoding: "utf8", maxBuffer: 32 << 20, stdio: ["ignore", "pipe", "ignore"] });
}

function pdfInfo(file) {
  const out = run("pdfinfo", [file]);
  const producer = (out.match(/^Producer:\s*(.*)$/m) || [])[1] || "";
  const pages = Number((out.match(/^Pages:\s*(\d+)/m) || [])[1] || 0);
  return { producer: producer.trim(), pages };
}

const MONTH_NAMES = "January|February|March|April|May|June|July|August|September|October|November|December";
const MES_NAMES = "enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|setiembre|octubre|noviembre|diciembre";
const LOCAL_DATE = new RegExp(`Salud:\\s*(${MONTH_NAMES})\\s+\\d{1,2},\\s*\\d{4}`, "i");
const PORTAL_DATE = new RegExp(`Salud:\\s*\\d{1,2}\\s+(${MONTH_NAMES}|${MES_NAMES})\\s+\\d{4}`, "i");

// Origen de una página según su texto y, si el archivo no está fusionado, su productor.
function detectOrigin(text, producer = "") {
  const local = LOCAL_DATE.test(text) || /Amaru Nan\b/.test(text) || /3814400\s*\|/.test(text);
  const portal = PORTAL_DATE.test(text) || /Amaru Ñan/.test(text);
  let origin = local && !portal ? "local" : portal && !local ? "portal" : "desconocido";
  const hint = /PDFKit/i.test(producer) ? "local" : /react-pdf/i.test(producer) ? "portal" : "";
  if (hint && origin !== hint) origin = "desconocido";
  return origin;
}

function pageText(file, page) {
  return run("pdftotext", ["-layout", "-f", String(page), "-l", String(page), file, "-"]);
}

function isoFromParts(year, monthName, day) {
  const name = monthName.toLowerCase();
  const month = (MONTHS.includes(name) ? MONTHS.indexOf(name) : MESES.indexOf(name.replace("setiembre", "septiembre"))) + 1;
  if (month < 1) return null;
  return `${year}-${String(month).padStart(2, "0")}-${String(Number(day)).padStart(2, "0")}`;
}

// Extrae lo que importa de una hoja: cédula, fecha de cobertura, titular y resultado por seguro.
function parseSheet(text) {
  const cedula = (text.match(/Identificaci[oó]n:\s*(\d{10})/) || [])[1] || null;
  let fecha = null;
  const portalDate = text.match(/Salud:\s*(\d{1,2})\s+([A-Za-záéíóú]+)\s+(\d{4})/); // 18 November 2026 · 01 octubre 2026
  const localDate = text.match(/Salud:\s*([A-Za-z]+)\s+(\d{1,2}),\s*(\d{4})/); // July 15, 2026
  if (portalDate) fecha = isoFromParts(portalDate[3], portalDate[2], portalDate[1]);
  else if (localDate) fecha = isoFromParts(localDate[3], localDate[1], localDate[2]);
  const lines = text.split(/\r?\n/);
  const titleIndex = lines.findIndex((line) => /CONSULTA DE COBERTURA DE SALUD/i.test(line));
  const titular = titleIndex >= 0 ? (lines.slice(titleIndex + 1).find((line) => line.trim()) || "").trim() : "";
  const seguros = {};
  const noDisponibles = [];
  for (const line of lines) {
    const match = line.match(/^\s*(IESS|ISSFA|ISSPOL)\s{2,}/);
    if (!match) continue;
    if (/servicio\s+no\s+disponible/i.test(line)) noDisponibles.push(match[1]);
    const results = [...line.matchAll(/\b(s[ií]|no)\s+registra\s+cobertura/gi)];
    if (results.length) {
      seguros[match[1]] = results[results.length - 1][1].toLowerCase().startsWith("s") ? "si" : "no";
    }
  }
  const redPrivada = /NO EXISTEN RESULTADOS/i.test(text) ? "sin resultados" : "con resultados";
  return { cedula, fecha, titular, seguros, redPrivada, noDisponibles };
}

function normalizeName(value) {
  return value.normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/\s+/g, " ").trim().toUpperCase();
}

function compareSheets(local, portal) {
  const diffs = [];
  for (const insurer of INSURERS) {
    if (local.seguros[insurer] !== portal.seguros[insurer]) {
      diffs.push(`${insurer}: local=${local.seguros[insurer] || "?"} portal=${portal.seguros[insurer] || "?"}`);
    }
  }
  if (local.redPrivada !== portal.redPrivada) {
    diffs.push(`Red privada: local=${local.redPrivada} portal=${portal.redPrivada}`);
  }
  const warnings = [];
  if (local.titular && portal.titular && normalizeName(local.titular) !== normalizeName(portal.titular)) {
    warnings.push(`Titular distinto: local="${local.titular}" portal="${portal.titular}"`);
  }
  return { diffs, warnings };
}

function validPortalPdf(file, cedula) {
  if (!fs.existsSync(file)) return false;
  try {
    const { producer, pages } = pdfInfo(file);
    return /react-pdf/i.test(producer) && pages >= 1 && pageText(file, 1).includes(cedula);
  } catch {
    return false;
  }
}

function findCoverageFiles(periodRoot) {
  const found = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(full);
      else if (/^C_COBERTURA(_\d+)?\.pdf$/i.test(entry.name) && full.includes(`${path.sep}4. EXPEDIENTES${path.sep}`)) found.push(full);
    }
  };
  walk(periodRoot);
  return found.sort();
}

function normalizeText(text) {
  return text.replace(/\s+/g, " ").trim();
}

// PDFs de una página del portal guardados en fuentes/ del período (fuentes previas a fusiones,
// cargas manuales, etc.), indexados por su texto normalizado. Sirven para conservar tal cual las
// páginas del portal de una hoja mixta: nunca se vuelven a consultar, porque el portal cambia.
function portalSourceIndex(periodRoot) {
  const index = new Map();
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(full);
        continue;
      }
      if (!entry.name.toLowerCase().endsWith(".pdf")) continue;
      try {
        const { pages, producer } = pdfInfo(full);
        if (pages !== 1) continue;
        const text = pageText(full, 1);
        if (!/CONSULTA DE COBERTURA/i.test(text) || detectOrigin(text, /pdfcpu/i.test(producer) ? "" : producer) !== "portal") continue;
        const key = normalizeText(text);
        if (!index.has(key)) index.set(key, full);
      } catch {
        // PDF ilegible: no es una fuente útil
      }
    }
  };
  const sources = path.join(periodRoot, "fuentes");
  if (fs.existsSync(sources)) walk(sources);
  return index;
}

function sha256(file) {
  return crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");
}

function sleepSync(ms) {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
}

function writeReports(outDir, report) {
  fs.writeFileSync(path.join(outDir, "reporte.json"), JSON.stringify(report, null, 2), { mode: 0o600 });
  const lines = [
    `# Reemplazo de coberturas — ${report.periodo}`,
    "",
    `Preparado: ${report.preparado} · Fase A (sin cambios en expedientes ni Oracle)`,
    "",
    `Hojas: ${report.hojas.length} · listas: ${report.resumen.LISTA} · revisar: ${report.resumen.REVISAR} · pendientes: ${report.resumen.PENDIENTE} · ya del portal: ${report.resumen.YA_PORTAL}`,
    "",
  ];
  for (const hoja of report.hojas) {
    lines.push(`## ${hoja.estado} · ${hoja.ruta}`, "");
    if (hoja.candidato) lines.push(`Candidato: \`${hoja.candidato}\``, "");
    if (hoja.motivo) lines.push(`Motivo: ${hoja.motivo}`, "");
    lines.push("| Pág. | Origen | Cédula | Fecha | Titular | IESS | ISSFA | ISSPOL | Diferencias |", "|---|---|---|---|---|---|---|---|---|");
    for (const p of hoja.paginas) {
      const s = p.local || {};
      const seg = (k) => `${(s.seguros || {})[k] || "?"}${p.portal ? ` → ${(p.portal.seguros || {})[k] || "?"}` : ""}`;
      const notes = [...(p.diferencias || []), ...(p.avisos || []), p.error || ""].filter(Boolean).join("; ") || "—";
      lines.push(`| ${p.pagina} | ${p.origen} | ${s.cedula || "?"} | ${s.fecha || "?"} | ${s.titular || ""} | ${seg("IESS")} | ${seg("ISSFA")} | ${seg("ISSPOL")} | ${notes} |`);
    }
    lines.push("");
  }
  fs.writeFileSync(path.join(outDir, "reporte.md"), lines.join("\n"), { mode: 0o600 });
}

// Fase B: envía al servidor las hojas preparadas y espera su resultado.
function apply(preparedArg) {
  const dataDir = path.resolve(process.env.FOLIO_DATA_DIR || "data");
  const prepared = fs.realpathSync(path.resolve(preparedArg));
  const report = JSON.parse(fs.readFileSync(path.join(prepared, "reporte.json"), "utf8"));
  const job = JSON.parse(fs.readFileSync(path.join(dataDir, "expedientes", report.periodo, "job.json"), "utf8"));
  // El servidor valida cada página por separado y las fusiona en este orden; el candidato armado
  // con pdfunite queda solo para revisión humana.
  const hojas = report.hojas
    .filter((h) => (h.estado === "LISTA" || h.estado === "REVISAR") && h.sha256_original && h.paginas.every((p) => p.portal_pdf && p.sha256 && p.local && p.local.cedula))
    .map((h) => ({
      ruta: h.ruta_expediente,
      sha256_original: h.sha256_original,
      paginas_originales: h.paginas_originales,
      paginas: h.paginas.map((p) => ({ pdf: p.portal_pdf.split(path.sep).join("/"), sha256: p.sha256, cedula: p.local.cedula })),
    }));
  if (!hojas.length) {
    process.stdout.write("No hay hojas LISTA o REVISAR con candidato para aplicar.\n");
    return 0;
  }
  const root = path.join(dataDir, "reemplazo_coberturas");
  const requests = path.join(root, "solicitudes");
  fs.mkdirSync(requests, { recursive: true, mode: 0o700 });
  const name = `${report.periodo}-${Date.now()}.json`;
  const tmp = path.join(requests, `.${name}.tmp`);
  fs.writeFileSync(tmp, JSON.stringify({ job_id: job.job_id, preparado: prepared, hojas }, null, 2), { mode: 0o600 });
  fs.renameSync(tmp, path.join(requests, name)); // el servidor solo toma archivos completos *.json
  process.stdout.write(`Solicitud enviada (${hojas.length} hojas). Esperando al servidor…\n`);
  const resultPath = path.join(root, "resultados", name);
  for (let waited = 0; waited < 300000; waited += 2000) {
    if (fs.existsSync(resultPath)) {
      const result = JSON.parse(fs.readFileSync(resultPath, "utf8"));
      fs.writeFileSync(path.join(prepared, "resultado_aplicacion.json"), JSON.stringify(result, null, 2), { mode: 0o600 });
      for (const h of result.hojas || []) process.stdout.write(`${h.estado.padEnd(11)} ${h.ruta}${h.detalle ? ` — ${h.detalle}` : ""}\n`);
      process.stdout.write(`Oracle: ${result.oracle}${result.error ? ` · Error: ${result.error}` : ""}\n`);
      return result.error || result.oracle !== "ok" || !(result.hojas || []).every((h) => h.estado === "REEMPLAZADA") ? 1 : 0;
    }
    sleepSync(2000);
  }
  process.stderr.write("El servidor no respondió en 5 minutos. ¿Está desplegada la versión con la cola de reemplazos?\n");
  return 1;
}

function main() {
  const args = parseArgs(process.argv);
  if (args.aplicar) {
    return apply(args.aplicar);
  }
  const periodo = args.periodo || "";
  if (!/^[A-Za-z0-9_.-]+$/.test(periodo)) {
    throw new Error("Indica --periodo con el nombre de la carpeta del período (data/expedientes/<periodo>).");
  }
  const dataDir = path.resolve(process.env.FOLIO_DATA_DIR || "data");
  const periodRoot = path.join(dataDir, "expedientes", periodo);
  if (!fs.existsSync(path.join(periodRoot, "job.json"))) {
    throw new Error(`No existe el período ${periodo} en ${path.join(dataDir, "expedientes")}.`);
  }
  const stamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d+Z$/, "Z");
  const outDir = path.resolve(args.salida || path.join(dataDir, "reemplazo_coberturas", periodo, stamp));
  const pagesDir = path.join(outDir, "paginas");
  const candidatesDir = path.join(outDir, "candidatos");
  fs.mkdirSync(pagesDir, { recursive: true, mode: 0o700 });
  fs.mkdirSync(candidatesDir, { recursive: true, mode: 0o700 });

  const report = { periodo, preparado: new Date().toISOString(), salida: outDir, hojas: [], resumen: { LISTA: 0, REVISAR: 0, PENDIENTE: 0, YA_PORTAL: 0 } };
  let limitReached = false;
  let sourceIndex = null;

  for (const file of findCoverageFiles(periodRoot)) {
    const ruta = path.relative(periodRoot, file);
    const { pages, producer } = pdfInfo(file);
    const producerHint = /pdfcpu/i.test(producer) ? "" : producer; // en fusiones el productor es pdfcpu
    const marker = `${path.sep}4. EXPEDIENTES${path.sep}`;
    const rutaExpediente = file.slice(file.indexOf(marker) + 1).split(path.sep).join("/");
    const hoja = { ruta, ruta_expediente: rutaExpediente, sha256_original: sha256(file), paginas_originales: pages, paginas: [], estado: "", candidato: null, motivo: "" };
    const pagePdfs = [];
    const texts = Array.from({ length: pages }, (_, i) => pageText(file, i + 1));
    const origins = texts.map((text) => detectOrigin(text, producerHint));
    hoja.mixta = origins.includes("portal") && origins.includes("local");
    if (hoja.mixta && !sourceIndex) sourceIndex = portalSourceIndex(periodRoot);
    for (let page = 1; page <= pages; page += 1) {
      const text = texts[page - 1];
      const origen = origins[page - 1];
      const local = parseSheet(text);
      const item = { pagina: page, origen, local };
      hoja.paginas.push(item);
      if (origen === "portal") {
        // Página ya obtenida del portal: se conserva tal cual, tomando su PDF original de fuentes/.
        // Separarla de la hoja no sirve: el lector de PDF del servidor no lee esa salida.
        const source = sourceIndex && sourceIndex.get(normalizeText(text));
        if (!source) {
          item.error = "Página del portal sin su PDF original exacto en fuentes/; resolver a mano.";
          continue;
        }
        const kept = path.join(pagesDir, `${path.basename(path.dirname(file))}__${path.basename(file, ".pdf")}__p${page}_original.pdf`);
        fs.copyFileSync(source, kept);
        fs.chmodSync(kept, 0o600);
        item.fuente_original = path.relative(periodRoot, source);
        item.portal_pdf = path.relative(outDir, kept);
        item.sha256 = sha256(kept);
        pagePdfs.push(kept);
        continue;
      }
      if (origen === "desconocido") {
        item.error = "No se pudo determinar si la página viene del portal o del generador local; revisar a mano.";
        continue;
      }
      if (!local.cedula || !local.fecha) {
        item.error = "No se pudo leer la cédula o la fecha de esta página; revisar a mano.";
        continue;
      }
      const name = `${local.cedula}_${local.fecha.replace(/-/g, "")}`;
      const portalFile = path.join(pagesDir, `${name}.pdf`);
      if (!limitReached && !validPortalPdf(portalFile, local.cedula)) {
        const result = spawnSync(process.execPath, [DOWNLOADER, "--cedula", local.cedula, "--fecha", local.fecha, "--output_name", name, "--output_dir", pagesDir], {
          encoding: "utf8",
          timeout: 180000,
        });
        if (result.status === 3) {
          limitReached = true;
        } else if (result.status !== 0) {
          item.error = `Portal: ${(result.stderr || "sin respuesta").trim().split("\n").pop()}`;
        }
      }
      if (limitReached && !validPortalPdf(portalFile, local.cedula)) {
        item.error = "El portal aplicó su límite de consultas; reanudar con --salida.";
        continue;
      }
      if (!item.error && !validPortalPdf(portalFile, local.cedula)) {
        item.error = "El PDF del portal no es válido o no contiene la cédula consultada.";
      }
      if (item.error) continue;
      item.portal = parseSheet(pageText(portalFile, 1));
      if (item.portal.noDisponibles.length) {
        // El portal no respondió por algún seguro: la hoja saldría incompleta. Se reintenta luego.
        item.error = `El portal respondió "servicio no disponible" para ${item.portal.noDisponibles.join(", ")}; reintentar.`;
        fs.renameSync(portalFile, `${portalFile}.servicio-no-disponible`);
        continue;
      }
      if (item.portal.fecha !== local.fecha) {
        // Protección: el portal respondió con otra fecha de cobertura (p. ej. la de hoy).
        item.error = `El portal devolvió la fecha ${item.portal.fecha || "?"} en vez de ${local.fecha}.`;
        fs.renameSync(portalFile, `${portalFile}.fecha-incorrecta`);
        continue;
      }
      item.portal_pdf = path.relative(outDir, portalFile);
      item.sha256 = sha256(portalFile);
      const { diffs, warnings } = compareSheets(local, item.portal);
      item.diferencias = diffs;
      item.avisos = warnings;
      if (pdfInfo(portalFile).pages !== 1) item.avisos.push(`El portal devolvió ${pdfInfo(portalFile).pages} páginas.`);
      pagePdfs.push(portalFile);
    }

    if (hoja.paginas.every((p) => p.origen === "portal")) {
      hoja.estado = "YA_PORTAL";
    } else if (hoja.paginas.some((p) => p.error)) {
      hoja.estado = "PENDIENTE";
      hoja.motivo = "Al menos una página no se pudo obtener del portal; la hoja no se reemplaza parcialmente.";
    } else {
      const candidate = path.join(candidatesDir, ruta);
      fs.mkdirSync(path.dirname(candidate), { recursive: true, mode: 0o700 });
      if (pagePdfs.length === 1) fs.copyFileSync(pagePdfs[0], candidate);
      else run("pdfunite", [...pagePdfs, candidate]);
      const candidatePages = pdfInfo(candidate).pages;
      fs.chmodSync(candidate, 0o600);
      hoja.candidato = path.relative(outDir, candidate);
      hoja.sha256_candidato = sha256(candidate);
      hoja.cedulas = [...new Set(hoja.paginas.map((p) => p.local.cedula).filter(Boolean))];
      const sameShape = candidatePages === pages;
      const anyDiff = hoja.paginas.some((p) => (p.diferencias || []).length || (p.avisos || []).length);
      hoja.estado = sameShape && !anyDiff ? "LISTA" : "REVISAR";
      if (!sameShape) hoja.motivo = `El candidato tiene ${candidatePages} páginas y el original ${pages}.`;
      else if (anyDiff) hoja.motivo = "El portal no coincide con la hoja local en al menos una página.";
    }
    report.resumen[hoja.estado] += 1;
    report.hojas.push(hoja);
    writeReports(outDir, report);
  }
  writeReports(outDir, report);
  process.stdout.write(`${JSON.stringify(report.resumen)}\nReporte: ${path.join(outDir, "reporte.md")}\n`);
  return limitReached ? 3 : 0;
}

if (require.main === module) {
  try {
    process.exit(main());
  } catch (error) {
    process.stderr.write(`${error.message}\n`);
    process.exit(1);
  }
}

module.exports = { parseSheet, compareSheets, detectOrigin };
