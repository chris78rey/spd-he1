#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
run_user="$(id -un)"

if ! command -v systemctl >/dev/null 2>&1; then
  echo "Este instalador requiere Linux con systemd." >&2
  exit 1
fi
if [[ ! -f "$project_dir/.env" ]]; then
  echo "Falta $project_dir/.env. Configura primero los datos de conexión Oracle." >&2
  exit 1
fi

cd "$project_dir"
npm run build
gofmt -w cmd/server/main.go
mkdir -p "$project_dir/bin"
go build -o "$project_dir/bin/folio-server" ./cmd/server

sudo tee /etc/systemd/system/folio.service >/dev/null <<UNIT
[Unit]
Description=Folio documents and Oracle service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$run_user
WorkingDirectory=$project_dir
EnvironmentFile=$project_dir/.env
ExecStart=$project_dir/bin/folio-server
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
UMask=0077

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable folio.service
sudo systemctl restart folio.service || sudo systemctl start folio.service

echo "Folio quedó habilitado para iniciar al arrancar el equipo."
echo "Abre http://127.0.0.1:8080"
echo "Estado: sudo systemctl status folio.service"
