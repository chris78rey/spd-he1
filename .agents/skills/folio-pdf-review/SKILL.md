---
name: folio-pdf-review
description: Review one or more Folio patient PDFs and recommend their official MSP filenames from the project's current classification rules. Use when asked what a PDF is, what name/code it should receive, or to review a batch; do not rename or modify files during review.
---

# Review Folio PDFs

Use this skill for one or more PDFs that need a document-type recommendation such as `HCU_006.pdf` or `HCU_053.pdf`.

## Sources of truth

Read the current project rules and classifier before interpreting a result:

- `reglas_clasificacion.yaml` defines confirmed filename aliases, header phrases, priority phrases, and regular phrase rules.
- `cmd/server/clasificacion.go` defines page limits, OCR fallback, filename handling, normalization, scoring, and ambiguity behavior.
- `catalogos/codigos_msp.json` provides the official display names for output codes.

Do not maintain a second hard-coded catalog in this skill. Use the bundled helper, which reads the YAML and JSON files on each run:

```bash
python3 .agents/skills/folio-pdf-review/scripts/review_pdfs.py "/path/one.pdf" "/path/two.pdf"
```

To review a directory, opt into recursive discovery:

```bash
python3 .agents/skills/folio-pdf-review/scripts/review_pdfs.py --recursive "/path/to/pdfs"
```

The helper prints basenames, recommended codes, matched phrases, competing rule scores, and extraction method. Use `--json` for structured output. It never writes to, renames, moves, or deletes the PDFs.

## How to interpret the result

- The Folio classifier reads text from at most the first two pages. If that extraction yields no text, it tries Tesseract in Spanish on the first page only. If text extraction returns some text, it does not OCR later pages just because no rule matched.
- Text normalization uppercases and removes common Spanish accents and punctuation before matching phrases.
- A confirmed filename alias from the YAML can override content. Do not assume a name such as `HCU_006 (1).pdf` is a confirmed alias; only the aliases currently listed in `entrada.codigos_nombre_archivo_confirmados` are used by the classifier.
- Otherwise, phrases configured as document-title signals are checked in the first 12 non-empty lines of page 1. A unique header match takes precedence over mentions elsewhere in the PDF; conflicting header types mean **pending review**.
- If the header has no configured title signal, priority phrases are checked in the extracted text. Regular rules then compare the number of distinct matching phrases. Equal counts across codes mean **pending review**; phrase length does not break ties.
- For multiple files, report each recommendation separately. Flag files that receive the same code as a possible duplicate group; the correct numbered suffix depends on the destination folder's existing files, so do not invent one.
- The helper uses Poppler's `pdftotext` for searchable PDFs while the app uses its Go PDF parser. Treat borderline results as recommendations and mention that parser differences can change a borderline result. The OCR fallback follows the app's first-page Spanish OCR settings.

## User-facing report

Give a concise table with the input basename, recommended code, matching phrases, competing matches, and whether the result is clear or needs visual confirmation. Say whether the result came from the first-page header, a priority phrase, or regular content matching. Do not paste extracted document text or repeat patient identifiers; mention only the phrases needed to justify the classification.

Review is read-only. If the user separately asks to rename or move files, handle that as a distinct action and preserve originals or avoid overwriting existing files.
