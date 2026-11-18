---
type: "query"
date: "2026-10-23T19:01:26.557595+00:00"
question: "no sera porque son muchos?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["N027: Fusión de PDFs [REQUERIDO cronológica automática; PARCIAL manual; NO VERIFICADO]", "R025: Fusión resuelve duplicados [PARCIAL; manual, no cronológica automática]", "mergeWorkspacePDFs()", "confirmWorkspaceFusion()", "confirmMergeAllPending()"]
---

# Q: no sera porque son muchos?

## Answer

Expansión desde el vocabulario del grafo: [pdf, fusion, duplicados, pendientes, paciente, hcu, manual, revisar, orden, fuentes]. Sí: el grupo HCU_010 contiene 56 PDFs y el endpoint de fusión acepta entre 2 y 50 rutas. Al enviar 56 devuelve HTTP 400 con el mensaje Selecciona entre 2 y 50 PDFs repetidos para fusionar. La fusión individual y Fusionar todos envían el grupo completo al mismo endpoint; Fusionar todos se detiene al fallar un grupo. El backend además exige que la selección coincida con un grupo completo, así que dividirlo en dos desde la interfaz tampoco funciona.

## Outcome

- Signal: useful

## Source Nodes

- N027: Fusión de PDFs [REQUERIDO cronológica automática; PARCIAL manual; NO VERIFICADO]
- R025: Fusión resuelve duplicados [PARCIAL; manual, no cronológica automática]
- mergeWorkspacePDFs()
- confirmWorkspaceFusion()
- confirmMergeAllPending()