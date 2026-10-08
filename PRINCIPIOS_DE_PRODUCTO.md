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

Antes de descargar el ZIP, se exigen en la raíz `I_LIQUIDACION.pdf`, `1. OFICIO DE PAGO.pdf`, `2. PLANILLA CONSOLIDADA.pdf` y la matriz oficial `3. MATRIZ_OBJECIONES_<SERVICIO>_<MES>_<AÑO>.xlsm`. El sistema conserva esos archivos y no interpreta el informe ni la matriz. La matriz cargada debe detallar por trámite los valores objetados, códigos, motivos y respuesta técnica. `P_INDIVIDUAL.pdf` y `C_COBERTURA.pdf` se reutilizan desde la copia del primer ingreso y no se vuelven a cargar en la pantalla de Objeciones; el operador puede corregirlos en esa copia desde Abrir documentos del paciente. La postura se registra por trámite. Antes de descargar, el operador marca qué PDFs clínicos incluir; solo los marcados se exportan, y `P_INDIVIDUAL.pdf` y `C_COBERTURA.pdf` siempre se incluyen. Los anexos son opcionales y, cuando existen, se cargan en `5. ANEXOS/<PACIENTE_TRAMITE>/` y se incluyen en el ZIP.

Cada carpeta dentro de `4. EXPEDIENTES/` y `5. ANEXOS/` identifica una atención con cédula, paciente y trámite para que un mismo paciente pueda tener varios `P_INDIVIDUAL.pdf` sin colisiones. La selección de PDFs afecta solo al ZIP de Objeciones; las descargas de los períodos normales conservan su comportamiento actual. Renombrar, reemplazar o añadir PDFs se hace en la copia del espacio derivado, nunca altera el primer ingreso.

El flujo no modifica `PDI_OBJETADO`, `PDI_ESTADO_DIGITALIZACION` ni otros estados Oracle hasta confirmar el esquema y la política transaccional. La interfaz debe distinguir el espacio derivado del primer ingreso y no presentarlo como cierre oficial del MSP.

## Evidencia de uso: búsqueda de períodos guardados

El usuario necesita localizar espacios acumulados de varios años y distinguir de inmediato Recepción de planillas de Objeciones del mismo servicio y mes. La lista conserva estado, documentos faltantes o mensaje, fecha, usuario e ID; permite combinar búsqueda parcial sin distinguir tildes o mayúsculas con filtros de tipo, servicio, año y estado, y ordena por período reciente primero sin ocultar años anteriores. Estos filtros solo afectan la vista: no cambian archivos ni el comportamiento de los espacios.

La misma búsqueda, normalización y orden por período se reutiliza al elegir espacios para Hojas de cobertura, Objeciones, documentos del paciente y descarga ZIP. Cada selector mantiene la elegibilidad propia de su flujo; se oculta el filtro de tipo cuando el tipo ya está fijado por la pantalla. Los filtros empiezan sin selección, muestran el total coincidente, permiten limpiarse y explican cómo recuperar años antiguos si no hay resultados. En estas opciones el texto principal usa al menos 16 px y los datos secundarios 14 px, con filas que admiten varias líneas y controles operables con teclado y en pantallas pequeñas.

## Decisión de navegación: expedientes primero

El menú prioriza las tareas con expedientes en este orden: recibir planillas, revisar documentos del paciente, hojas de cobertura, subsanar objeciones y descargar el expediente ZIP. El orden orienta, pero no establece pasos obligatorios para todos los lotes. Objeciones es una ruta separada que parte de un período de primer ingreso preparado. La biblioteca «Mis documentos» sigue accesible en una sección propia y se identifica como almacenamiento de este navegador; «Planilla digital», una consulta de registros, queda bajo «Consultas» al final del menú.

## Decisión de producto: descarga de avance y preparación para entrega

La descarga ZIP puede entregar los archivos disponibles mientras falten requisitos del paquete. El sistema muestra todos los requisitos pendientes y nombra el archivo `*_AVANCE_INCOMPLETO.zip`. Solo muestra «Listo para entrega» cuando están presentes los documentos obligatorios válidos del paquete y no quedan PDFs reconocidos pendientes de fusionar; para Objeciones también exige la matriz oficial, la postura y los documentos obligatorios de cada trámite. Los anexos de Objeciones siguen siendo opcionales. La descarga es de solo lectura: no modifica el expediente ni cambia su estado. El ZIP sin sufijo se reserva para paquetes que cumplen esas comprobaciones actuales.

En «Descargar expediente ZIP», Recepción de planillas y Objeciones se eligen por separado. La pantalla abre Recepción de planillas por defecto para que el ZIP ordinario sea visible de inmediato; el selector de Objeciones conserva la selección de PDFs propia de ese flujo. Cada selector muestra cuántos espacios preparados hay disponibles.

## Decisión de producto: reutilizar Objeciones por servicio y período

Solo puede haber un espacio nuevo de Objeciones por servicio, mes y año. Si ya existe uno, el flujo lo reutiliza y agrega únicamente los trámites nuevos, sin duplicar filas ni alterar documentos, posturas o selecciones ya guardadas. Si hay duplicados históricos, se muestran con sus IDs para que la persona elija uno; el sistema no los elimina ni fusiona automáticamente. Agregar trámites invalida la matriz oficial existente, conserva una copia y solicita volver a cargarla. Los documentos de cabecera siguen pendientes hasta preparar el ZIP final. Una lista vacía de PDFs es un resultado normal; los fallos de consulta se informan dentro de esa sección con una opción de reintento y sin cambiar el estado de creación del espacio.

## Evidencia de uso: períodos de Objeciones con muchos pacientes

Al crecer un período, evita renderizar todas las tarjetas completas de pacientes de una vez. Permite localizar por nombre, trámite o cédula y muestra diez expedientes por página. Agrupa la lista de PDFs una vez por carpeta para que consultar cada tarjeta no vuelva a recorrer todos los archivos. Conserva accesibles todos los trámites y sus selecciones, posturas y anexos.

## Decisión de producto: consultar coberturas con fecha elegida

«Hojas de cobertura» conserva la consulta habitual con la fecha del trámite y ofrece una segunda modalidad para elegir otra fecha. La persona selecciona un período de Recepción preparado, una o varias planillas y una fecha; la fecha elegida se aplica a todas las seleccionadas y sus PDFs se guardan en los expedientes de ese período. En menores se consultan también las cédulas registradas como referentes con la misma fecha. La interfaz muestra las fechas guardadas y permite descargar un ZIP filtrado por la fecha consultada. Esta modalidad registra el documento en el expediente y en su asociación documental Oracle, pero deja intacto `PDI_COBERTURA`; ese indicador continúa representando el resultado del flujo habitual.
