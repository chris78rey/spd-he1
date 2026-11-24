#!/usr/bin/env node
// Descarga el PDF de cobertura que genera la propia página del MSP.
//
// La página arma el PDF en el navegador y lo muestra en <embed src="blob:...">.
// Este script maneja Chrome por CDP (WebSocket nativo de Node 22, sin dependencias),
// llena el formulario, pulsa "Consultar" y guarda los bytes del blob tal cual.
//
// Por normativa la hoja de cobertura debe ser la del sitio del MSP: no se genera localmente
// ni se consulta su API directamente. Uso:
//   node download_site_pdf.cjs --cedula 1712345678 --fecha 2026-10-05 \
//        --output_name COBERTURA_x --output_dir /ruta
// Escribe <output_dir>/<output_name>.pdf
//
// Códigos de salida: 0 ok · 1 fallo técnico · 2 mensaje del sitio · 3 límite de consultas del sitio.
// Las líneas LIMITE_SITIO / MENSAJE_SITIO de stderr las lee el servidor Go. Nunca imprimen la cédula.
//
// Variables: CHROME_PATH, COBERTURA_CHROME_NO_SANDBOX, COBERTURA_SITE_URL (solo pruebas).

"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawn } = require("child_process");
const { loadEnvFile } = require("./runtime_config.cjs");

loadEnvFile();

const SITE_URL = (process.env.COBERTURA_SITE_URL || "https://coberturasalud.msp.gob.ec/").trim();
const HARD_TIMEOUT_MS = 150000;
const MAX_ATTEMPTS = 3;

const BTN_NUEVA = "[...document.querySelectorAll('button')].find(b => b.innerText.trim().toLowerCase() === 'nueva consulta')";
const BTN_CONSULTAR = "[...document.querySelectorAll('button')].find(b => b.innerText.trim().toLowerCase() === 'consultar')";
const INPUT_CEDULA = "document.querySelector('#cedula')";
const INPUT_FECHA =
  "([...document.querySelectorAll('input')].find(i => [...(i.labels ?? [])].some(l => /fecha de consulta/i.test(l.textContent))) ?? document.querySelectorAll('input')[1])";

class SiteMessage extends Error {
  constructor(message) {
    super(message);
    this.siteMessage = message;
  }
}

function parseArgs(argv) {
  const args = {};
  for (let i = 2; i < argv.length; i += 2) {
    if (argv[i].startsWith("--")) {
      args[argv[i].slice(2)] = argv[i + 1];
    }
  }
  return args;
}

function toPortalDate(fechaIso) {
  const [year, month, day] = fechaIso.split("-");
  return `${day}-${month}-${year}`;
}

function isTrue(value) {
  return ["1", "true", "yes"].includes(String(value || "").trim().toLowerCase());
}

const T0 = Date.now();
function mark(label) {
  if (isTrue(process.env.COBERTURA_DEBUG_TIMING)) {
    process.stderr.write(`[tiempo] ${label}: ${((Date.now() - T0) / 1000).toFixed(1)}s\n`);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function findChrome() {
  const fromEnv = (process.env.CHROME_PATH || "").trim();
  if (fromEnv) {
    if (!fs.existsSync(fromEnv)) {
      throw new Error("CHROME_PATH no existe.");
    }
    return fromEnv;
  }
  const fixed = [
    "/usr/bin/google-chrome",
    "/usr/bin/google-chrome-stable",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/snap/bin/chromium",
  ];
  for (const candidate of fixed) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  const roots = [process.env.PLAYWRIGHT_BROWSERS_PATH, path.join(os.homedir(), ".cache", "ms-playwright")].filter(Boolean);
  for (const root of roots) {
    let dirs = [];
    try {
      dirs = fs.readdirSync(root).filter((name) => /^chromium-\d+$/.test(name)).sort().reverse();
    } catch {
      continue;
    }
    for (const dir of dirs) {
      for (const sub of ["chrome-linux64", "chrome-linux"]) {
        const candidate = path.join(root, dir, sub, "chrome");
        if (fs.existsSync(candidate)) {
          return candidate;
        }
      }
    }
  }
  throw new Error("No se encontró Chrome/Chromium. Define CHROME_PATH.");
}

async function launchChrome() {
  const exe = findChrome();
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), "folio-cobertura-chrome-"));
  const args = [
    "--headless=new",
    "--remote-debugging-port=0",
    `--user-data-dir=${profile}`,
    "--no-first-run",
    "--no-default-browser-check",
    "--disable-gpu",
    "--disable-dev-shm-usage",
  ];
  if ((process.getuid && process.getuid() === 0) || isTrue(process.env.COBERTURA_CHROME_NO_SANDBOX)) {
    args.push("--no-sandbox");
  }
  // El portal escribe "Fecha de consulta" con la hora del navegador. El servidor está en UTC,
  // así que Chrome usa la zona horaria de Ecuador (o COBERTURA_TZ) para que la hoja muestre la hora real.
  const env = { ...process.env, TZ: (process.env.COBERTURA_TZ || "").trim() || "America/Guayaquil" };
  // detached: Chrome crea procesos auxiliares; se cierran todos matando el grupo.
  const child = spawn(exe, args, { stdio: ["ignore", "ignore", "pipe"], detached: true, env });
  let exited = false;
  let stderrTail = "";
  child.stderr.on("data", (chunk) => {
    if (stderrTail.length < 4000) {
      stderrTail += chunk.toString(); // el motivo del fallo de Chrome está al inicio, no al final
    }
  });
  child.on("exit", () => {
    exited = true;
  });
  child.on("error", () => {
    exited = true;
  });
  const handle = { child, profile };
  const portFile = path.join(profile, "DevToolsActivePort");
  for (let i = 0; i < 80; i += 1) {
    if (exited) {
      await sleep(150); // deja leer el último stderr antes de armar el mensaje
      const hint = /sandbox/i.test(stderrTail)
        ? " Chrome no puede usar su sandbox en este equipo: define COBERTURA_CHROME_NO_SANDBOX=true en .env si aceptas ejecutarlo sin sandbox."
        : "";
      throw Object.assign(new Error(`Chrome terminó antes de abrir el puerto de depuración.${hint}`), { chrome: handle });
    }
    if (fs.existsSync(portFile)) {
      const [port, wsPath] = fs.readFileSync(portFile, "utf8").split(/\r?\n/);
      if (port && wsPath) {
        handle.wsUrl = `ws://127.0.0.1:${port}${wsPath}`;
        return handle;
      }
    }
    await sleep(250);
  }
  throw Object.assign(new Error("Chrome no abrió el puerto de depuración a tiempo."), { chrome: handle });
}

function closeChrome(handle) {
  if (!handle) {
    return;
  }
  try {
    process.kill(-handle.child.pid, "SIGKILL");
  } catch {
    try {
      handle.child.kill("SIGKILL");
    } catch {
      // ya terminó
    }
  }
  try {
    fs.rmSync(handle.profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
  } catch {
    // el perfil temporal lo limpia el sistema
  }
}

class Cdp {
  constructor(ws) {
    this.ws = ws;
    this.nextId = 1;
    this.pending = new Map();
    this.sessionId = null;
    ws.addEventListener("message", (event) => {
      let message;
      try {
        message = JSON.parse(event.data);
      } catch {
        return;
      }
      if (message.id && this.pending.has(message.id)) {
        const { resolve, reject, timer } = this.pending.get(message.id);
        this.pending.delete(message.id);
        clearTimeout(timer);
        if (message.error) {
          reject(new Error(`CDP ${message.error.message || "error"}`));
        } else {
          resolve(message.result || {});
        }
      }
    });
    ws.addEventListener("close", () => {
      for (const { reject, timer } of this.pending.values()) {
        clearTimeout(timer);
        reject(new Error("Conexión con Chrome cerrada."));
      }
      this.pending.clear();
    });
  }

  static connect(url) {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url);
      ws.addEventListener("open", () => resolve(new Cdp(ws)), { once: true });
      ws.addEventListener("error", () => reject(new Error("No se pudo conectar con Chrome (CDP).")), { once: true });
    });
  }

  call(method, params = {}, timeoutMs = 60000) {
    const id = this.nextId;
    this.nextId += 1;
    const message = { id, method, params };
    if (this.sessionId) {
      message.sessionId = this.sessionId;
    }
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`CDP ${method}: sin respuesta.`));
      }, timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      this.ws.send(JSON.stringify(message));
    });
  }

  async evaluate(expression) {
    const result = await this.call("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
    if (result.exceptionDetails) {
      const description = result.exceptionDetails.exception?.description || "excepción JS";
      throw new Error(`JS: ${String(description).split("\n")[0]}`);
    }
    return result.result ? result.result.value : undefined;
  }

  async waitFor(expression, timeoutMs, stepMs = 150) {
    for (let elapsed = 0; elapsed < timeoutMs; elapsed += stepMs) {
      if ((await this.evaluate(expression)) === true) {
        return;
      }
      await sleep(stepMs);
    }
    throw new Error("Tiempo agotado esperando la página.");
  }

  async openPage() {
    const target = await this.call("Target.createTarget", { url: "about:blank" });
    const attached = await this.call("Target.attachToTarget", { targetId: target.targetId, flatten: true });
    this.sessionId = attached.sessionId;
    await this.call("Page.enable");
  }

  async navigate() {
    await this.call("Page.navigate", { url: SITE_URL });
    await sleep(1500);
    await this.waitFor(`!!${INPUT_CEDULA} && !!${INPUT_FECHA}`, 30000);
  }

  async type(selectorExpr, text) {
    await this.evaluate(`(() => { const e = ${selectorExpr}; e.focus(); e.select(); return true; })()`);
    for (const type of ["keyDown", "keyUp"]) {
      await this.call("Input.dispatchKeyEvent", { type, key: "Backspace", code: "Backspace", windowsVirtualKeyCode: 8 });
    }
    await this.call("Input.insertText", { text });
  }
}

const MESES = ["enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"];

// El campo "Fecha de Consulta" es un selector de MUI de solo lectura: no acepta texto y por
// defecto queda con la fecha de hoy. Se abre el calendario, se navega al mes pedido, se pulsa
// el día y "OK", y se comprueba que el campo quedó con la fecha pedida.
async function selectDate(cdp, fechaPortal) {
  const [day, month, year] = fechaPortal.split("-").map(Number);
  const target = year * 12 + (month - 1);
  await cdp.evaluate(`${INPUT_FECHA}.click()`);
  await cdp.waitFor("!!document.querySelector('[role=dialog] [class*=CalendarHeader-label]')", 15000, 120);
  for (let step = 0; step < 240; step += 1) {
    const label = String((await cdp.evaluate("document.querySelector('[role=dialog] [class*=CalendarHeader-label]').textContent")) || "")
      .trim()
      .toLowerCase();
    const match = label.match(/([a-záéíóú]+)\s+(\d{4})/);
    const shownMonth = match ? MESES.indexOf(match[1].replace("setiembre", "septiembre")) : -1;
    if (shownMonth < 0) throw new Error("No se reconoce el mes del calendario del portal.");
    const shown = Number(match[2]) * 12 + shownMonth;
    if (shown === target) break;
    const button = shown > target ? "Previous month" : "Next month";
    const clicked = await cdp.evaluate(`(() => { const b = document.querySelector('[role=dialog] button[aria-label="${button}"]'); if (!b || b.disabled) return false; b.click(); return true; })()`);
    if (!clicked) throw new Error("El calendario del portal no permite llegar a la fecha pedida.");
    await cdp.waitFor(`(document.querySelector('[role=dialog] [class*=CalendarHeader-label]')?.textContent || '').trim().toLowerCase() !== ${JSON.stringify(label)}`, 5000, 80);
    if (step === 239) throw new Error("No se pudo navegar al mes pedido en el calendario.");
  }
  const picked = await cdp.evaluate(`(() => {
    const b = [...document.querySelectorAll('[role=dialog] button')].find((x) => /PickersDay/.test(x.className) && !/dayOutsideMonth/.test(x.className) && x.innerText.trim() === "${day}");
    if (!b || b.disabled) return false;
    b.click();
    return true;
  })()`);
  if (!picked) throw new Error("El día pedido no está disponible en el calendario del portal.");
  await cdp.evaluate("[...document.querySelectorAll('[role=dialog] button')].find((b) => b.innerText.trim().toUpperCase() === 'OK')?.click()");
  await cdp.waitFor("!document.querySelector('[role=dialog] [class*=CalendarHeader-label]')", 5000, 80);
  const value = await cdp.evaluate(`${INPUT_FECHA}.value`);
  if (value !== fechaPortal) throw new Error(`El portal no aceptó la fecha (quedó ${value}).`);
}

// Una consulta completa. Devuelve los bytes del PDF o lanza SiteMessage con el aviso del sitio.
async function consult(cdp, cedula, fechaPortal) {
  await cdp.evaluate(`${BTN_NUEVA}?.click()`);
  await cdp.waitFor("!document.querySelector('embed[src^=\"blob:\"]') && !document.querySelector('[role=dialog]')", 15000, 120);
  await cdp.type(INPUT_CEDULA, cedula);
  await selectDate(cdp, fechaPortal);
  await cdp.waitFor(`(() => { const b = ${BTN_CONSULTAR}; return !!b && !b.disabled; })()`, 15000, 120);
  mark("formulario lleno");
  await cdp.evaluate(`${BTN_CONSULTAR}.click()`);
  await cdp.waitFor("!!document.querySelector('embed[src^=\"blob:\"]') || !!document.querySelector('[role=dialog]')", 60000);

  mark("respuesta del sitio");
  if ((await cdp.evaluate("!!document.querySelector('[role=dialog]')")) === true) {
    const raw = (await cdp.evaluate("document.querySelector('[role=dialog]').innerText")) || "";
    await cdp.evaluate("document.querySelector('[role=dialog] button')?.click()");
    let message = raw.replace(/[\r\n]+/g, " ").trim();
    message = message.replace(/^Mensaje\s+/i, "").replace(/\s+ok$/i, "").trim();
    throw new SiteMessage(message || "El sitio devolvió un aviso sin texto.");
  }

  const b64 = await cdp.evaluate(`(async () => {
    const buf = new Uint8Array(await (await fetch(document.querySelector('embed').src)).arrayBuffer());
    let s = ''; for (let i = 0; i < buf.length; i += 0x8000) s += String.fromCharCode.apply(null, buf.subarray(i, i + 0x8000));
    return btoa(s); })()`);
  const pdf = Buffer.from(String(b64 || ""), "base64");
  if (pdf.length < 1000 || pdf.subarray(0, 5).toString("latin1") !== "%PDF-") {
    throw new SiteMessage("El blob descargado no es un PDF.");
  }
  return pdf;
}

function oneLine(text) {
  return String(text).replace(/[\r\n]+/g, " ").slice(0, 400);
}

async function main() {
  const args = parseArgs(process.argv);
  const { cedula, fecha, output_name: outputName, output_dir: outputDir } = args;
  if (!/^\d{10}$/.test(cedula || "")) {
    throw new Error("La cédula debe tener 10 dígitos.");
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(fecha || "")) {
    throw new Error("La fecha debe usar formato YYYY-MM-DD.");
  }
  if (!/^[A-Za-z0-9_.-]+$/.test(outputName || "") || !outputDir) {
    throw new Error("output_name u output_dir inválido.");
  }

  let chrome = null;
  const killer = setTimeout(() => {
    closeChrome(chrome);
    process.stderr.write("Tiempo máximo excedido descargando la cobertura.\n");
    process.exit(1);
  }, HARD_TIMEOUT_MS);

  try {
    chrome = await launchChrome();
    mark("chrome listo");
    const cdp = await Cdp.connect(chrome.wsUrl);
    await cdp.openPage();
    await cdp.navigate();
    mark("página cargada");

    let lastError = null;
    for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt += 1) {
      try {
        const pdf = await consult(cdp, cedula, toPortalDate(fecha));
        fs.mkdirSync(outputDir, { recursive: true, mode: 0o700 });
        fs.writeFileSync(path.join(outputDir, `${outputName}.pdf`), pdf, { mode: 0o600 });
        return 0;
      } catch (error) {
        lastError = error;
        if (error instanceof SiteMessage) {
          if (/mite de consultas/i.test(error.siteMessage)) {
            // El sitio pide esperar (minutos u hora). No se insiste: se informa para que el operador reintente.
            process.stderr.write(`LIMITE_SITIO: ${oneLine(error.siteMessage)}\n`);
            return 3;
          }
          // Aviso genérico con cédula ya validada: suele ser pasajero.
          if (!/mero de identificaci/i.test(error.siteMessage) || attempt === MAX_ATTEMPTS) {
            process.stderr.write(`MENSAJE_SITIO: ${oneLine(error.siteMessage)}\n`);
            return 2;
          }
        }
        if (attempt < MAX_ATTEMPTS) {
          await sleep(2000 * attempt);
          try {
            await cdp.navigate();
          } catch {
            // el siguiente intento vuelve a fallar con su propio error
          }
        }
      }
    }
    process.stderr.write(`Fallo técnico: ${oneLine(lastError && lastError.message)}\n`);
    return 1;
  } catch (error) {
    if (error && error.chrome) {
      chrome = error.chrome;
    }
    process.stderr.write(`Fallo técnico: ${oneLine(error.message)}\n`);
    return 1;
  } finally {
    clearTimeout(killer);
    closeChrome(chrome);
  }
}

if (require.main === module) {
  main().then(
    (code) => process.exit(code),
    (error) => {
      process.stderr.write(`Fallo técnico: ${oneLine(error.message)}\n`);
      process.exit(1);
    }
  );
}

module.exports = { toPortalDate, consult };
