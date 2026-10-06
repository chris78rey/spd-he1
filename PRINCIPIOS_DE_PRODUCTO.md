# Principios de producto de Folio

Estos criterios son la base permanente para diseñar y priorizar las próximas iteraciones de Folio. Priorizan trabajos reales de gestión documental por encima de sumar pantallas o controles decorativos.

## Resultado principal

Una persona puede guardar un documento, encontrarlo cuando lo necesita y estar segura de qué va a pasar al abrirlo, moverlo o eliminarlo. Cada mejora debe reducir el tiempo, el esfuerzo o la incertidumbre de una de esas tareas.

## Casos de uso que guían el producto

1. **Encontrar un documento rápidamente.** Buscar por nombre, filtrar por formato y ver cuándo se añadió. Los resultados deben ser claros, inmediatos y útiles en pantallas pequeñas. Prioridad máxima.
2. **Guardar documentos sin fricción.** Admitir selector de archivos y arrastrar y soltar; indicar destino y resultado de la carga; conservar los documentos al recargar.
3. **Reconocer el documento antes de actuar.** Mostrar formato, tamaño y fecha; ofrecer una vista previa cuando el formato lo permita.
4. **Ordenar varios documentos de una vez.** Permitir selección múltiple y acciones en lote con el destino a la vista.
5. **Recuperarse de errores.** Explicar acciones destructivas, dar confirmación cuando haga falta y permitir deshacer cuando sea posible. No borrar ni mover datos de demostración como si fueran documentos reales.

## Reglas para decisiones de diseño

- Orden de prioridad: encontrar y recuperar; guardar con confianza; organizar; colaboración; personalización visual.
- Usar componentes Vuetify para interacciones consistentes: `v-menu` para acciones contextuales, `v-dialog` para decisiones o formularios, `v-chip` para filtros y estado, `v-tooltip` para iconos sin etiqueta y `v-snackbar` para confirmación breve.
- Una acción tiene que actualizar los datos persistidos, informar qué ocurrió y reflejar el resultado en la pantalla. No presentar acciones simuladas como funcionales.
- Mantener atajos de teclado y diseño adaptable, sin hacer que dependan del ratón.
- Proteger el contenido y ser explícito sobre el almacenamiento. La persistencia local actual pertenece al navegador; la sincronización entre personas o dispositivos requiere un servicio y no se debe insinuar hasta tenerlo.
- Probar los flujos con documentos reales de distintos formatos. Los datos de ejemplo deben estar identificados y no deben aparentar ser archivos del usuario.

## Próxima mejora de mayor impacto

Los archivos de una biblioteca crecen más rápido que la capacidad de recordar dónde se guardaron. Por eso, búsqueda inmediata más filtros por formato y fecha aporta más valor que añadir nuevas páginas administrativas. Después, la prioridad es la carga fiable y la organización en lote. La colaboración multiusuario es una etapa posterior porque requiere identidad, permisos y almacenamiento remoto.

## Qué medir

- Tiempo y número de acciones para localizar un documento por nombre o formato.
- Porcentaje de cargas que sobreviven a una recarga y se abren correctamente.
- Errores al mover o eliminar documentos y facilidad para recuperarse de ellos.
- Uso de filtros y acciones en lote frente a abrir documentos uno por uno.

## Evidencia de uso: documentos del paciente

El usuario indicó que ampliar el visor dejaba poco espacio para distinguir los nombres de los PDFs junto al botón «Quitar». El visor y las acciones del paciente se reúnen en un modal amplio: la lista conserva espacio propio, muestra los nombres en varias líneas y sitúa «Quitar» debajo de cada nombre. Cerrar y reabrir el modal conserva la selección del paciente y del PDF dentro del mismo período.

## Evidencia de uso: regenerar coberturas después de eliminar un período

El usuario indicó que, al eliminar un período, debe poder volver a generar y descargar sus hojas de cobertura. El estado de cobertura debe liberarse junto con las rutas de documentos del período eliminado; las planillas que conservan una ruta activa en otro expediente no se deben alterar. Los registros históricos de períodos eliminados deben permitir recuperar las planillas que quedaron marcadas como cubiertas sin un expediente activo.

## Decisión de producto: subsanar objeciones en un espacio separado

El usuario pidió un flujo de Objeciones independiente de la recepción del primer ingreso. El operador selecciona directamente en una lista a los pacientes objetados del período preparado; el sistema crea un espacio derivado solo con los seleccionados. El lote de primer ingreso debe permanecer intacto.

El espacio derivado se crea en cuanto el operador selecciona trámites objetados; no depende de que haya cargado documentos de cabecera. Solo copia los trámites seleccionados y deja intacto el primer ingreso. Se puede agregar un trámite omitido más tarde desde el mismo espacio; si ya se cargó la matriz oficial, se invalida para que se vuelva a presentar con la nueva selección.

Antes de descargar el ZIP, se exigen en la raíz `I_LIQUIDACION.pdf`, `1. OFICIO DE PAGO.pdf`, `2. PLANILLA CONSOLIDADA.pdf` y la matriz oficial `3. MATRIZ_OBJECIONES_<SERVICIO>_<MES>_<AÑO>.xlsm`. El sistema conserva esos archivos y no interpreta el informe ni la matriz. La matriz cargada debe detallar por trámite los valores objetados, códigos, motivos y respuesta técnica. El ZIP requiere además `C_COBERTURA.pdf`, postura y un `P_INDIVIDUAL.pdf` por cada trámite; no exige anexos, que se guardan en `5. ANEXOS/<PACIENTE_TRAMITE>/` cuando existan.

Cada carpeta dentro de `4. EXPEDIENTES/` y `5. ANEXOS/` identifica una atención con cédula, paciente y trámite para que un mismo paciente pueda tener varios `P_INDIVIDUAL.pdf` sin colisiones. Si falta `C_COBERTURA.pdf` en el primer ingreso, el operador puede cargarlo en el espacio derivado.

El flujo no modifica `PDI_OBJETADO`, `PDI_ESTADO_DIGITALIZACION` ni otros estados Oracle hasta confirmar el esquema y la política transaccional. La interfaz debe distinguir el espacio derivado del primer ingreso y no presentarlo como cierre oficial del MSP.
