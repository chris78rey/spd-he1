---
type: "query"
date: "2026-10-22T22:19:40.842963+00:00"
question: "si ya salio Anexos.pdf y si le cambio graba esto en graphify para si toca aumentar mas de estos"
contributor: "graphify"
outcome: "useful"
source_nodes: ["catalogos_codigos_msp", "renameWorkspacePDF", "mspPDFCode", "loadMSPPDFCodeSet", "workspace.go", "App.vue", "nextMSPPDFName"]
---

# Q: si ya salio Anexos.pdf y si le cambio graba esto en graphify para si toca aumentar mas de estos

## Answer

Expansion de consulta usada: [catalogos, codigos, msp, rename, workspace, pdf, documento].

La lista de nombres estándar vive en catalogos/codigos_msp.json. Cada opción contiene title (texto que ve la persona), value (nombre exacto del archivo .pdf) y categoria. La interfaz importa este catálogo en src/App.vue como mspPdfCodes y lo presenta en el autocomplete mspPDFOptions. El backend en cmd/server/workspace.go, mediante loadMSPPDFCodeSet(), lee el mismo JSON y valida los nombres permitidos al renombrar.

Para agregar otro nombre: añadir una entrada al JSON con un title legible, un value con basename .pdf válido (sin directorios) y una categoria adecuada. La interfaz necesita una compilación frontend nueva porque el JSON queda incluido en el bundle; reiniciar el servicio por sí solo no actualiza la lista del navegador. Desplegar según scripts/deploy-runtime.sh --confirm-deploy y comprobar frontend servido y proceso activo con scripts/verify-runtime.sh.

En la ventana Cambiar nombre del documento, seleccionar el PDF, buscar la opción en Nombre estándar MSP (o Nombre MSP), elegirla en el autocomplete y pulsar Guardar nombre. Escribir texto solo filtra sugerencias; no guarda por sí mismo.

Anexos.pdf ya se agregó, desplegó, verificó en el bundle servido y quedó en el commit ff4b896 (Add Anexos.pdf to MSP catalog), enviado a origin/main.

## Outcome

- Signal: useful

## Source Nodes

- catalogos_codigos_msp
- renameWorkspacePDF
- mspPDFCode
- loadMSPPDFCodeSet
- workspace.go
- App.vue
- nextMSPPDFName
