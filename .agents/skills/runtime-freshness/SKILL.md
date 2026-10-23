---
name: spd-msp-runtime-freshness
description: Comprueba si la versión actual de SPD MSP coincide con el backend cargado por systemd y el frontend servido por HTTP.
---

# Verificar la versión activa de SPD MSP

Usa esta skill cuando el usuario pregunte si un cambio ya está ejecutándose, si la última compilación es la que ve en el navegador o si el servicio está actualizado.

## Procedimiento

1. Desde la raíz del repositorio `/home/spd-he1/codex_projects/spd_msp`, ejecuta el verificador:

   ```bash
   "${CODEX_HOME:-$HOME/.codex}/skills/folio-runtime-freshness/scripts/verify-runtime.sh"
   ```

2. El verificador compila el backend Go y el frontend Vue en un directorio temporal y compara los resultados con el ejecutable cargado por `folio.service`, `bin/folio-server`, `dist/` y los recursos que entrega HTTP. La verificación no modifica archivos desplegados ni reinicia servicios.

3. Si el servidor usa otro nombre de unidad o URL, configura `FOLIO_SERVICE_NAME` y `FOLIO_BASE_URL`. El valor predeterminado de esta instalación es `folio.service` y `http://172.16.60.41:5174`.

4. Informa el resultado y los componentes que difieren:
   - `FRESHNESS_OK`: el backend activo, el binario instalado, el build local y el frontend servido corresponden al código fuente actual.
   - `FRESHNESS_STALE`: indica qué artefactos no coinciden. La aplicación activa no debe presentarse como lista para probar ese cambio.
   - `FRESHNESS_UNVERIFIED`: alguna comprobación no pudo completarse. La versión activa queda sin confirmar.

5. Si la compilación falla, informa el error como verificación incompleta; no concluyas que el runtime está actualizado.

## Límites

- Esta skill solo verifica. No despliega, reemplaza binarios, modifica `dist/` ni reinicia el servicio.
- Desplegar requiere que el usuario lo solicite explícitamente. Sigue `folio-runtime-freshness` y las reglas de recuperación de `AGENTS.md`; inspecciona antes los cambios pendientes y usa su helper de despliegue.
- El helper está preparado para `sudo -n` mediante el wrapper acotado `/usr/local/sbin/spd-deploy-runtime`. Para instalarlo una vez, el usuario ejecuta `bash scripts/install-runtime-deploy-access.sh` desde la raíz del repo e ingresa su contraseña de `sudo`. Después, un despliegue explícitamente solicitado no pide contraseña.
- No determines qué versión corre el servicio usando solo fechas de archivos. Comprueba el proceso activo y su ejecutable cargado mediante el helper.
