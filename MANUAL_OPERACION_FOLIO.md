# Manual de operación de Folio

Guía para arrancar, detener, revisar y recuperar el servicio Folio en el servidor Linux.

## Instalación actual

| Elemento | Configuración |
|---|---|
| Servicio systemd | `folio.service` |
| Usuario del servicio | `spd-he1` |
| Directorio de trabajo | `/home/spd-he1/codex_projects/spd_msp` |
| Ejecutable | `/home/spd-he1/codex_projects/spd_msp/bin/folio-server` |
| Configuración privada | `/home/spd-he1/codex_projects/spd_msp/.env` |
| URL actual en la red interna | `http://172.16.60.41:5174/` |
| Datos de expedientes | `/home/spd-he1/codex_projects/spd_msp/data/` |

El servicio está habilitado para iniciar con Linux. Systemd también intenta volver a iniciarlo si el proceso falla, con una pausa de cinco segundos. La configuración se encuentra en `/etc/systemd/system/folio.service`.

El archivo `.env` contiene credenciales y otros datos sensibles. No lo publiques, no lo adjuntes a tickets y no copies su contenido en mensajes o logs.

## Arrancar Folio

```bash
sudo systemctl start folio.service
sudo systemctl status folio.service --no-pager
```

El estado esperado es `active (running)`. Para una comprobación breve:

```bash
systemctl is-active folio.service
curl --max-time 5 -sS -o /dev/null -w 'HTTP %{http_code}\n' http://172.16.60.41:5174/
```

La respuesta HTTP `200` confirma que la página principal responde. No comprueba por sí sola que Oracle esté disponible ni que los flujos de trabajo estén operativos; revisa los logs e inicia sesión para validar esas partes.

## Detener Folio

Antes de detenerlo, avisa a las personas usuarias y confirma que no haya una carga, clasificación, sincronización, fusión o descarga en curso. Detener el servicio puede interrumpir solicitudes activas.

```bash
sudo systemctl stop folio.service
sudo systemctl status folio.service --no-pager
```

Tras un `stop` manual, el servicio permanece detenido. Se puede volver a iniciar con el comando de la sección anterior. Un reinicio posterior del servidor lo arrancará porque está habilitado.

## Reiniciar Folio

Usa el reinicio después de cambiar la configuración o instalar una versión nueva. Coordínalo como una interrupción breve y verifica antes que no haya operaciones en curso.

```bash
sudo systemctl restart folio.service
sudo systemctl status folio.service --no-pager
```

Si el estado es `failed` o no llega a `active (running)`, consulta los logs antes de repetir el reinicio.

## Revisar disponibilidad y logs

```bash
# Estado actual
systemctl status folio.service --no-pager

# Confirmar arranque automático
systemctl is-enabled folio.service

# Últimos 100 mensajes
sudo journalctl -u folio.service -n 100 --no-pager

# Seguir los mensajes nuevos
sudo journalctl -u folio.service -f

# Confirmar que el puerto 5174 está escuchando
sudo ss -ltnp | grep ':5174'

# Revisar la página desde el propio servidor
curl --max-time 5 -sS -o /dev/null -w 'HTTP %{http_code}\n' http://172.16.60.41:5174/
```

Al compartir logs, revisa primero que no contengan datos personales, rutas sensibles ni información de conexión.

## Si no arranca o no abre la página

1. Revisa `systemctl status folio.service --no-pager` y `sudo journalctl -u folio.service -n 100 --no-pager`.
2. Confirma que `/home/spd-he1/codex_projects/spd_msp/.env` exista y sea legible por `spd-he1`, sin mostrar su contenido:

   ```bash
   sudo -u spd-he1 test -r /home/spd-he1/codex_projects/spd_msp/.env && echo 'Configuración legible'
   ```

3. Comprueba si otro proceso ocupa el puerto:

   ```bash
   sudo ss -ltnp | grep ':5174'
   ```

4. Si los logs indican un problema Oracle, confirma con el DBA que la red y el servicio Oracle estén disponibles y que las variables Oracle requeridas estén configuradas en `.env`. No pegues las credenciales en el ticket.
5. Cuando se corrija la causa, ejecuta `sudo systemctl restart folio.service` y comprueba el estado y la URL.

El servicio necesita conectar con Oracle al arrancar. Si Oracle no está disponible, systemd intentará reiniciar Folio al fallar el proceso.

## Arranque después de apagar o reiniciar el servidor

Cuando el equipo vuelva a encenderse, systemd inicia Folio automáticamente. Confirma el estado con:

```bash
systemctl is-enabled folio.service
systemctl is-active folio.service
```

Los resultados esperados son `enabled` y `active`. Si alguien detuvo Folio manualmente antes del apagado, comprueba igualmente el estado tras el siguiente arranque. El encendido físico del servidor y los reinicios del sistema operativo deben coordinarse con DTIC.

## Acceso por HTTP y alcance de red

La URL actual usa HTTP, no HTTPS. El inicio de sesión transmite credenciales Oracle y la aplicación intercambia cookies de sesión y documentos. En esta configuración esos datos no están cifrados durante el tránsito.

Mientras no exista HTTPS, limita el acceso a la red interna autorizada o a una VPN administrada. No publiques el puerto `5174` en Internet ni lo redirijas desde un router. Solicita a DTIC una regla de firewall que permita solo los equipos o segmentos autorizados. El número del puerto no sustituye el cifrado.

La aplicación usa actualmente `FOLIO_LISTEN_HOST=172.16.60.41`, `API_PORT=5174` y `COOKIE_SECURE=false`. No cambies estos valores sin coordinar la URL, las reglas de red y la compatibilidad de las cookies con DTIC.

## Datos y respaldos

La configuración actual usa `FOLIO_DATA_DIR=data`. Los expedientes y archivos de trabajo se guardan bajo `data/`, incluidos `data/expedientes/` y `data/staging/`. Oracle mantiene sus propios datos; su respaldo corresponde al proceso administrado por el DBA.

Antes de respaldar archivos locales, coordina una ventana, avisa a las personas usuarias y detén Folio para evitar una copia inconsistente:

```bash
sudo systemctl stop folio.service
systemctl is-active folio.service  # Debe responder inactive antes de copiar.
sudo tar -czf "/RUTA_DE_RESPALDO_APROBADA/folio-data-$(date +%Y%m%d-%H%M%S).tar.gz" \
  -C /home/spd-he1/codex_projects/spd_msp data
sudo systemctl start folio.service
sudo systemctl status folio.service --no-pager
```

Reemplaza `/RUTA_DE_RESPALDO_APROBADA` por un destino de respaldo autorizado, disponible y con permisos restringidos. Si `is-active` devuelve `active`, detén el procedimiento y revisa por qué Folio no se detuvo. Si `tar` falla, vuelve a iniciar Folio y revisa el error. Verifica que el archivo se creó y que el respaldo se puede leer. No guardes el respaldo únicamente dentro de `data/` ni en el mismo disco si se busca proteger frente a una falla del servidor.

La unidad systemd no configura respaldos automáticos. DTIC debe definir la frecuencia, retención, ubicación protegida y prueba periódica de restauración. Para restaurar, detén Folio, conserva una copia del estado actual, restaura el respaldo validado con el propietario y permisos correctos, y comprueba la aplicación antes de reabrir el servicio.

## Permisos para eliminar períodos

Todo usuario que ingrese necesita el rol Oracle `SPD_EXTERNOS`. Para eliminar un período completo, además necesita `SPD_BORRA_EXPEDIENTE` en su propia cuenta Oracle. La aplicación consulta los roles de la cuenta personal al iniciar sesión y no guarda su contraseña. Después de que el DBA otorgue el rol, la persona debe cerrar sesión e ingresar de nuevo para actualizar el permiso de esa sesión.

Si la persona no tiene el rol adicional, no verá la acción de borrar y el servidor rechazará una petición directa con HTTP `403`. El permiso solo cubre eliminar un período completo; quitar un PDF y administrar «Mis documentos» son acciones separadas.

## Lista breve de operación

- Antes de detener o reiniciar: avisar, revisar que no haya tareas en curso y coordinar la ventana.
- Después de arrancar: comprobar `active`, el puerto `5174`, respuesta HTTP e inicio de sesión con Oracle.
- Para problemas: conservar el mensaje de error y los logs pertinentes, ocultando secretos y datos personales.
- Para cambios de versión o configuración: tener respaldo y una ruta de reversión antes de reemplazar archivos.
- Mantener el acceso limitado a la red aprobada mientras Folio funcione por HTTP.
