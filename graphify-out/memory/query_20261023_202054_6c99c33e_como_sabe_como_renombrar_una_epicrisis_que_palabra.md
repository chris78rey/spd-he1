---
type: "query"
date: "2026-10-23T20:20:54.823197+00:00"
question: "como sabe como renombrar una epicrisis que palabra clave y que determina"
contributor: "graphify"
outcome: "useful"
source_nodes: ["classifyPDF()", "matchClassification()", "resolvePDFClassification()", "HCU_006.pdf", "Reglas OCR, clasificación y checklist ACFSS"]
---

# Q: como sabe como renombrar una epicrisis que palabra clave y que determina

## Answer

Expansión desde el vocabulario del grafo: [classification, classify, clasificacion, keyword, ocr, pdf]. La regla de HCU_006.pdf (Epicrisis/alta hospitalaria) busca cuatro frases: FORMULARIO 006, EPICRISIS, RESUMEN DE ALTA y CUADRO CLINICO DE EGRESO. El clasificador extrae texto de hasta las dos primeras páginas; si no puede, aplica Tesseract en español a la primera página. Normaliza mayúsculas y tildes, cuenta frases distintas coincidentes y elige la regla ganadora; un empate o ninguna coincidencia deja el archivo como pendiente. HCU_006 no está entre los códigos de nombre de archivo confirmados; por tanto, su nombre de entrada no basta para asignarlo. El nombre final se toma del código elegido, HCU_006.pdf.

## Outcome

- Signal: useful

## Source Nodes

- classifyPDF()
- matchClassification()
- resolvePDFClassification()
- HCU_006.pdf
- Reglas OCR, clasificación y checklist ACFSS