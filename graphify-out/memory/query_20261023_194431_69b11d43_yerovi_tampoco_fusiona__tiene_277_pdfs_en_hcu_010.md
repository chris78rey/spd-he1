---
type: "query"
date: "2026-10-23T19:44:31.427867+00:00"
question: "YEROVI tampoco fusiona; tiene 277 PDFs en HCU_010"
contributor: "graphify"
outcome: "useful"
source_nodes: ["mergeWorkspacePDFs()", "confirmWorkspaceFusion()", "confirmMergeAllPending()"]
---

# Q: YEROVI tampoco fusiona; tiene 277 PDFs en HCU_010

## Answer

Expansión desde el vocabulario del grafo: [planilla, digital, paciente, nombres, pdf, identidad, ingesta, ocr, faltante, expediente, tramite]. El grupo YEROVI HCU_010 tiene 277 PDFs, por encima del máximo 100 que estaba desplegado, así que el endpoint rechazaba la petición. El límite se elevó a 300 por grupo y la espera de la interfaz para fusiones pasó de 60 a 300 segundos. El endpoint sigue exigiendo el grupo completo y conserva el orden y las fuentes originales.

## Outcome

- Signal: useful

## Source Nodes

- mergeWorkspacePDFs()
- confirmWorkspaceFusion()
- confirmMergeAllPending()