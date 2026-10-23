#!/usr/bin/env python3
"""Review Folio PDFs against the live project classification rules; never edits inputs."""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
import tempfile
import unicodedata
from pathlib import Path

try:
    import yaml
except ImportError:
    yaml = None


ACCENT_MAP = str.maketrans(
    {
        "Á": "A", "À": "A", "Â": "A", "Ä": "A",
        "É": "E", "È": "E", "Ê": "E", "Ë": "E",
        "Í": "I", "Ì": "I", "Î": "I", "Ï": "I",
        "Ó": "O", "Ò": "O", "Ô": "O", "Ö": "O",
        "Ú": "U", "Ù": "U", "Û": "U", "Ü": "U", "Ñ": "N",
    }
)
HEADER_LINE_LIMIT = 12


def project_root() -> Path:
    for parent in Path(__file__).resolve().parents:
        if (parent / "reglas_clasificacion.yaml").is_file():
            return parent
    raise FileNotFoundError("No encuentro reglas_clasificacion.yaml en el proyecto.")


def normalize(text: str) -> str:
    """Mirror normalizeOCRText: uppercase, normalize Spanish accents, keep words/digits."""
    text = unicodedata.normalize("NFC", text.upper()).translate(ACCENT_MAP)
    chars = [char if char.isalpha() or char.isdigit() else " " for char in text]
    return " ".join("".join(chars).split())


def run(command: list[str], timeout: int) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        capture_output=True,
        text=True,
        errors="replace",
        timeout=timeout,
        check=False,
    )


def extract_text(path: Path) -> tuple[str, str, str]:
    """Read pages 1-2 as text, then use Spanish OCR on page 1 only if text is empty."""
    pdftotext = shutil.which("pdftotext")
    if pdftotext:
        try:
            result = run([pdftotext, "-f", "1", "-l", "2", str(path), "-"], 60)
            if result.returncode == 0 and result.stdout.strip():
                return result.stdout, "VECTORIAL (págs. 1–2)", ""
        except subprocess.TimeoutExpired:
            pass

    pdftoppm = shutil.which("pdftoppm")
    tesseract = shutil.which("tesseract")
    if not pdftoppm or not tesseract:
        if not pdftotext:
            return "", "SIN EXTRACCIÓN", "Falta pdftotext; no se pudo leer texto vectorial."
        return "", "SIN TEXTO/OCR", "No hubo texto extraíble y falta pdftoppm o tesseract."

    try:
        with tempfile.TemporaryDirectory(prefix="folio-pdf-review-") as temp_dir:
            prefix = str(Path(temp_dir) / "page")
            render = subprocess.run(
                [pdftoppm, "-f", "1", "-l", "1", "-gray", "-r", "250", "-singlefile", str(path), prefix],
                capture_output=True,
                text=True,
                errors="replace",
                timeout=90,
                check=False,
            )
            if render.returncode != 0:
                return "", "OCR FALLÓ", "No se pudo renderizar la primera página."
            image = Path(prefix + ".pgm")
            if not image.is_file():
                image = Path(prefix + ".png")
            ocr = run([tesseract, str(image), "stdout", "-l", "spa"], 90)
            if ocr.returncode == 0 and ocr.stdout.strip():
                return ocr.stdout, "OCR español (pág. 1)", ""
            return "", "OCR FALLÓ", "Tesseract no devolvió texto legible."
    except subprocess.TimeoutExpired:
        return "", "OCR FALLÓ", "La extracción excedió el tiempo permitido."


def filename_code(path: Path, rules: dict) -> str:
    base = path.stem.upper().strip()
    if base.startswith("PENDIENTE_TMP_"):
        base = base[len("PENDIENTE_TMP_") :]
    if "_" in base:
        stem, suffix = base.rsplit("_", 1)
        if suffix.isdigit() and stem:
            base = stem
    normalized_base = normalize(base)
    for hint in rules.get("entrada", {}).get("codigos_nombre_archivo_confirmados", []):
        if normalized_base and normalized_base == normalize(str(hint.get("base", ""))):
            return str(hint.get("codigo", "")).strip()
    return ""


def phrase_matches(text: str, phrases: list[str]) -> list[str]:
    found: list[str] = []
    seen: set[str] = set()
    for phrase in phrases:
        normalized = normalize(str(phrase))
        if normalized and normalized not in seen:
            seen.add(normalized)
            if normalized in text:
                found.append(str(phrase))
    return found


def first_page_header(text: str) -> str:
    first_page = text.split("\f", 1)[0]
    lines = [line.strip() for line in first_page.splitlines() if line.strip()]
    return normalize(" ".join(lines[:HEADER_LINE_LIMIT]))


def score_rules(text: str, header: str, rule_list: list[dict]) -> tuple[list[dict], list[dict], list[dict]]:
    title_matches: list[dict] = []
    priority_matches: list[dict] = []
    regular_scores: list[dict] = []
    for rule in rule_list:
        code = str(rule.get("codigo", "")).strip()
        title = phrase_matches(header, rule.get("palabras_clave_titulo", []))
        if title:
            title_matches.append({"code": code, "count": len(title), "matches": title})
        priority = phrase_matches(text, rule.get("palabras_clave_prioritarias", []))
        if priority:
            priority_matches.append({"code": code, "matches": priority})
        matches = phrase_matches(text, rule.get("palabras_clave", []))
        if matches:
            regular_scores.append(
                {
                    "code": code,
                    "count": len(matches),
                    "specificity": sum(len(normalize(match)) for match in matches),
                    "matches": matches,
                }
            )
    regular_scores.sort(key=lambda row: (row["count"], row["specificity"]), reverse=True)
    return title_matches, priority_matches, regular_scores


def decide(path: Path, rules: dict) -> dict:
    code_by_name = filename_code(path, rules)
    try:
        text, method, warning = extract_text(path)
    except OSError:
        text, method, warning = "", "ERROR", "No se pudo abrir el PDF o ejecutar las herramientas."
    normalized_text = normalize(text)
    normalized_header = first_page_header(text) if normalized_text else ""
    title_matches, priority, scores = (
        score_rules(normalized_text, normalized_header, rules.get("reglas", []))
        if normalized_text else ([], [], [])
    )

    result = {
        "file": path.name,
        "recommended": "PENDIENTE DE REVISIÓN",
        "method": method,
        "reason": "",
        "matches": [],
        "competing_rules": scores,
        "warning": warning,
        "batch_collision": False,
    }

    if code_by_name:
        result.update(recommended=code_by_name, reason="alias confirmado en el nombre de entrada", matches=[])
        return result

    if title_matches:
        codes = {row["code"] for row in title_matches}
        result["matches"] = title_matches
        if len(codes) == 1:
            result.update(recommended=next(iter(codes)), reason="coincidencia en el encabezado de la primera página")
        else:
            result["reason"] = "empate entre tipos documentales en el encabezado; requiere revisión"
        return result

    if priority:
        codes = {row["code"] for row in priority}
        result["matches"] = priority
        if len(codes) == 1:
            result.update(recommended=next(iter(codes)), reason="frase prioritaria")
        else:
            result["reason"] = "empate entre reglas prioritarias"
        return result

    if not scores:
        result["reason"] = "sin coincidencias de contenido"
        return result

    best_count = scores[0]["count"]
    winners = [row for row in scores if row["count"] == best_count]
    winning_codes = {row["code"] for row in winners}
    result["matches"] = winners
    if len(winning_codes) == 1:
        result["recommended"] = next(iter(winning_codes))
        result["reason"] = "más frases coincidentes" if best_count > 1 else "coincidencia única"
    else:
        result["reason"] = "empate entre reglas; requiere revisión"
    return result


def collect_inputs(paths: list[Path], recursive: bool) -> tuple[list[Path], list[str]]:
    files: list[Path] = []
    errors: list[str] = []
    seen: set[Path] = set()
    for path in paths:
        if not path.exists():
            errors.append(f"No existe: {path.name}")
            continue
        if path.is_dir():
            if not recursive:
                errors.append(f"Es una carpeta; usa --recursive: {path.name}")
                continue
            candidates = sorted(item for item in path.rglob("*") if item.is_file() and item.suffix.lower() == ".pdf")
        else:
            candidates = [path]
        for item in candidates:
            if item.suffix.lower() != ".pdf":
                errors.append(f"No es PDF: {item.name}")
                continue
            resolved = item.resolve()
            if resolved not in seen:
                seen.add(resolved)
                files.append(resolved)
    return files, errors


def render_table(rows: list[dict], errors: list[str]) -> None:
    print("# | ARCHIVO | NOMBRE PROPUESTO | MÉTODO | COINCIDENCIAS | RESULTADO")
    for index, row in enumerate(rows, 1):
        matches = []
        for candidate in row["matches"]:
            matches.extend(candidate.get("matches", []))
        for candidate in row["competing_rules"]:
            if candidate["code"] != row["recommended"]:
                matches.extend(f"{candidate['code']}: {phrase}" for phrase in candidate["matches"])
        match_text = ", ".join(matches) or "—"
        note = row["reason"]
        if row["warning"]:
            note = f"{note}; {row['warning']}".strip("; ")
        if row["batch_collision"]:
            note += "; mismo código que otro PDF del lote"
        print(f"{index} | {row['file']} | {row['recommended']} | {row['method']} | {match_text} | {note}")
    for error in errors:
        print(f"ERROR | {error}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Recomienda códigos de nombre para PDFs según las reglas actuales de Folio.")
    parser.add_argument("paths", nargs="+", type=Path, help="PDFs; admite carpetas con --recursive")
    parser.add_argument("--recursive", action="store_true", help="buscar PDFs dentro de carpetas")
    parser.add_argument("--json", action="store_true", help="emitir resultado JSON en vez de tabla")
    args = parser.parse_args()

    if yaml is None:
        print("Error: instala PyYAML en el entorno Python para leer reglas_clasificacion.yaml.", file=sys.stderr)
        return 2
    try:
        root = project_root()
        rules = yaml.safe_load((root / "reglas_clasificacion.yaml").read_text(encoding="utf-8"))
        catalog = json.loads((root / "catalogos" / "codigos_msp.json").read_text(encoding="utf-8"))
    except (OSError, ValueError, yaml.YAMLError) as exc:
        print(f"Error al cargar las reglas del proyecto: {exc}", file=sys.stderr)
        return 2

    titles = {str(item.get("value", "")): str(item.get("title", "")) for item in catalog}
    files, errors = collect_inputs(args.paths, args.recursive)
    rows = [decide(path, rules) for path in files]
    counts: dict[str, int] = {}
    for row in rows:
        code = row["recommended"]
        if code != "PENDIENTE DE REVISIÓN":
            counts[code] = counts.get(code, 0) + 1
    for row in rows:
        row["batch_collision"] = counts.get(row["recommended"], 0) > 1
        row["official_title"] = titles.get(row["recommended"], "")

    if args.json:
        print(json.dumps({"results": rows, "errors": errors}, ensure_ascii=False, indent=2))
    else:
        render_table(rows, errors)
    return 2 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
