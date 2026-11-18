---
type: "query"
date: "2026-10-23T19:47:16.307119+00:00"
question: "YEROVI tampoco fusiona; tiene 277 PDFs en HCU_010"
contributor: "graphify"
outcome: "corrected"
correction: "El grupo YEROVI HCU_010 tiene 277 PDFs. El límite correcto solicitado por el usuario es 1000 por grupo, con cuerpo JSON de 1 MiB y timeout de interfaz de 10 minutos; no 300 ni 5 minutos."
source_nodes: ["mergeWorkspacePDFs()", "confirmWorkspaceFusion()", "confirmMergeAllPending()"]
---

# Q: YEROVI tampoco fusiona; tiene 277 PDFs en HCU_010

## Answer

Expansión desde el vocabulario del grafo: [planilla, digital, paciente, nombres, pdf, identidad, ingesta, ocr, faltante, expediente, tramite]. El grupo tiene 277 PDFs y el límite 100 lo rechaza; la decisión final es permitir hasta 1000 PDFs por grupo, aumentar el límite del cuerpo JSON a 1 MiB y esperar hasta diez minutos en la interfaz.

## Outcome

- Signal: corrected
- Correction: El grupo YEROVI HCU_010 tiene 277 PDFs. El límite correcto solicitado por el usuario es 1000 por grupo, con cuerpo JSON de 1 MiB y timeout de interfaz de 10 minutos; no 300 ni 5 minutos.

## Source Nodes

- mergeWorkspacePDFs()
- confirmWorkspaceFusion()
- confirmMergeAllPending()