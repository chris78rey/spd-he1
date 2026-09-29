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
