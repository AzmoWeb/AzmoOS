#!/usr/bin/env bash
# Désinstalle AzmoOS (Docker, vos apps et /DATA sont conservés).
set -euo pipefail
[ "$(id -u)" -eq 0 ] || { echo "Lancez avec sudo."; exit 1; }
systemctl disable --now azmoos 2>/dev/null || true
rm -f /etc/systemd/system/azmoos.service /usr/local/bin/azmod
systemctl daemon-reload
echo "AzmoOS désinstallé. Conservés : /DATA, /etc/azmoos, Docker et les conteneurs."
echo "Pour tout effacer : sudo rm -rf /etc/azmoos /DATA"
