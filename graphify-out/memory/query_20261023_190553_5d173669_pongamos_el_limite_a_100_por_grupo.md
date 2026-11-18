---
type: "query"
date: "2026-10-23T19:05:53.935864+00:00"
question: "pongamos el limite a 100 por grupo"
contributor: "graphify"
outcome: "useful"
source_nodes: ["maxWorkspaceFusionPDFs", "mergeWorkspacePDFs()", "workspaceFusionGroups()"]
---

# Q: pongamos el limite a 100 por grupo

## Answer

Expansión desde el vocabulario del grafo: [fusion, pdf, merge, workspace, pendientes, manual, duplicados, rutas]. Se implementó un máximo de 100 PDFs por grupo en el endpoint de fusión y el mensaje de validación toma el mismo límite. La fusión sigue exigiendo el grupo completo, mantiene el orden seleccionado y conserva las fuentes originales. Se documentó la decisión en PRINCIPIOS_DE_PRODUCTO.md: el grupo HCU_010 reportado tenía 56 PDFs y excedía el límite previo de 50.

## Outcome

- Signal: useful

## Source Nodes

- maxWorkspaceFusionPDFs
- mergeWorkspacePDFs()
- workspaceFusionGroups()