#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
SKILL_HELPER="${CODEX_HOME:-$HOME/.codex}/skills/folio-runtime-freshness/scripts/deploy-runtime.sh"
WRAPPER_SOURCE="$ROOT/scripts/deploy-runtime-root-wrapper"
WRAPPER="/usr/local/sbin/spd-deploy-runtime"
ENGINE="/usr/local/libexec/spd-deploy-runtime-engine"
SUDOERS="/etc/sudoers.d/spd-msp-runtime-deploy"
TEMP_RULE="$(mktemp "${TMPDIR:-/tmp}/spd-msp-sudoers.XXXXXX")"
trap 'rm -f -- "$TEMP_RULE"' EXIT

[[ "$ROOT" == "/home/spd-he1/codex_projects/spd_msp" ]] || { echo "Raíz inesperada: $ROOT" >&2; exit 2; }
[[ -f "$SKILL_HELPER" && -x "$WRAPPER_SOURCE" ]] || { echo "Falta el helper o wrapper requerido." >&2; exit 2; }
grep -Fq 'sudo -n /usr/local/sbin/spd-deploy-runtime "$tmp_dir"' "$SKILL_HELPER" || {
  echo "El helper de despliegue aún no está configurado para usar el wrapper limitado." >&2
  exit 2
}

sudo -v
sudo install -d -o root -g root -m 0755 /usr/local/libexec /usr/local/sbin
sudo install -o root -g root -m 0755 "$SKILL_HELPER" "$ENGINE"
sudo install -o root -g root -m 0755 "$WRAPPER_SOURCE" "$WRAPPER"

printf '%s ALL=(root) NOPASSWD: %s *\n' "$(id -un)" "$WRAPPER" > "$TEMP_RULE"
sudo visudo -cf "$TEMP_RULE"
sudo install -o root -g root -m 0440 "$TEMP_RULE" "$SUDOERS"
sudo visudo -c
sudo -n -l "$WRAPPER" /tmp/folio-runtime-deploy.ABC123

printf '\nConfigurado. Los siguientes despliegues autorizados por el usuario usarán sudo sin pedir contraseña.\n'
