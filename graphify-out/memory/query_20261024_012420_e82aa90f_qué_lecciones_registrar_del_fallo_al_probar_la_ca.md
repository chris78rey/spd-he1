---
type: "query"
date: "2026-10-24T01:24:20.939623+00:00"
question: "¿Qué lecciones registrar del fallo al probar la carga de carpetas de auditoría con un período sintético y un ZIP de prueba?"
contributor: "graphify"
outcome: "corrected"
correction: "No subir el ZIP a un período operativo ni dejar carpetas sobrantes al regenerarlo. Crear un período sintético separado, usar solo datos ficticios, verificar exactamente 3 carpetas y sus PDFs, renombrar todos los binds Oracle con prefijo b_, desplegar y verificar; después pedir que el usuario vuelva a seleccionar el ZIP actualizado."
source_nodes: ["applyAuditorFolders", "syncOracleAuditorReceipts", "inspectAuditorZIP", "periodWorkspaceID"]
---

# Q: ¿Qué lecciones registrar del fallo al probar la carga de carpetas de auditoría con un período sintético y un ZIP de prueba?

## Answer

Lecciones: crear un período de prueba dedicado y marcado incompleto, sin planillas reales; no reutilizar un período operativo porque la asociación persiste el período elegido. Antes de presentar un ZIP como prueba, reconstruirlo desde una lista explícita de carpetas y verificar cantidad de carpetas únicas, cantidad de PDFs y nombres de entradas; al actualizar un ZIP existente, borrar carpetas antiguas para no arrastrarlas. Para pruebas sin PDI, cargar mediante el flujo de carpetas de auditoría y dejar que el backend cree registros externos Oracle; no insertar PDI ficticios. En Oracle, usar nombres de bind seguros con prefijo b_ y evitar palabras reservadas como month/year/user, que pueden producir ORA-01745 en MERGE. Si la sincronización falla, la transacción debe revertirse; revisar el log del servidor antes de reintentar y volver a seleccionar el ZIP actualizado porque la vista previa del navegador conserva el archivo anterior. Confirmar el despliegue con FRESHNESS_OK antes de pedir otro intento.

## Outcome

- Signal: corrected
- Correction: No subir el ZIP a un período operativo ni dejar carpetas sobrantes al regenerarlo. Crear un período sintético separado, usar solo datos ficticios, verificar exactamente 3 carpetas y sus PDFs, renombrar todos los binds Oracle con prefijo b_, desplegar y verificar; después pedir que el usuario vuelva a seleccionar el ZIP actualizado.

## Source Nodes

- applyAuditorFolders
- syncOracleAuditorReceipts
- inspectAuditorZIP
- periodWorkspaceID