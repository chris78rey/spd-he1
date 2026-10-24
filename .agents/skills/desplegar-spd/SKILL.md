---
name: desplegar-spd
description: Compila y despliega la versión actual de SPD MSP cuando el usuario solicita explícitamente publicarla en el servicio activo.
---

# Desplegar SPD MSP

Usa esta skill para compilar y desplegar SPD MSP en `folio.service`. No la uses para una consulta de estado o para verificar si la aplicación está actualizada; en esos casos usa `runtime-freshness`.

## Antes de desplegar

- Despliega únicamente cuando el usuario haya pedido claramente desplegar los cambios actuales. La invocación de esta skill para consultar o explicar el proceso no autoriza por sí sola un despliegue.
- Desde `/home/spd-he1/codex_projects/spd_msp`, revisa `git status --short` y los diffs de código que entrarán en la compilación. El helper compila el código actual, incluidos los cambios locales sin commit. Si hay cambios de código ajenos al objetivo solicitado, detente y explica cuáles entrarían.
- No hagas commit ni push como parte del despliegue.

## Despliegue

La autorización administrativa sin contraseña se instala una sola vez. Antes del primer despliegue, comprueba si el wrapper limitado ya está autorizado:

```bash
sudo -n -l /usr/local/sbin/spd-deploy-runtime /tmp/folio-runtime-deploy.ABC123
```

Si el wrapper aún no está autorizado, indícale al usuario que ejecute una vez desde una terminal del servidor:

```bash
cd /home/spd-he1/codex_projects/spd_msp && bash scripts/install-runtime-deploy-access.sh
```

La instalación pide la contraseña de `sudo` para crear una regla `NOPASSWD` limitada al wrapper root de SPD MSP. No guarda la contraseña. Cuando termine, los siguientes despliegues no requieren ingresarla.

Con autorización explícita del usuario y con `folio.service` activo, ejecuta el helper directamente, sin `sudo -v` ni `sudo systemctl`:

```bash
cd /home/spd-he1/codex_projects/spd_msp && /home/spd-he1/.codex/skills/folio-runtime-freshness/scripts/deploy-runtime.sh --confirm-deploy
```

El helper compila Go y Vue, verifica que el proceso del servicio no haya cambiado ni tenga trabajos hijos activos, guarda una copia de recuperación, instala los candidatos mediante el wrapper limitado y valida el ejecutable cargado y los recursos HTTP. Conserva las copias de recuperación que informa. Si el wrapper indica que el servicio no está activo, detente e informa el estado; no invoques `systemctl` directamente.

Si la instalación inicial requiere contraseña, el usuario debe ejecutarla en una terminal interactiva. Nunca solicites ni aceptes la contraseña por chat. No reintentes automáticamente si el helper aborta por trabajos activos, por un cambio de proceso o por otro fallo.

Después de `DEPLOY_OK`, ejecuta el verificador de solo lectura:

```bash
/home/spd-he1/.codex/skills/folio-runtime-freshness/scripts/verify-runtime.sh
```

Informa si terminó con `FRESHNESS_OK`, `FRESHNESS_STALE` o `FRESHNESS_UNVERIFIED`, e incluye el PID, hash y rutas de recuperación que imprima el despliegue. Si hubo rollback o la verificación no confirma la versión, no digas que el cambio está activo.
