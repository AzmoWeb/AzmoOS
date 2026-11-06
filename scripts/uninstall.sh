#!/usr/bin/env bash
# Désinstalle Azmo OS. Docker, tes applications et /DATA sont conservés.
set -euo pipefail
[ "$(id -u)" -eq 0 ] || { echo "Lance ce script avec sudo."; exit 1; }
systemctl disable --now azmoos azmoos-hotspot 2>/dev/null || true
rm -f /etc/systemd/system/azmoos.service /etc/systemd/system/azmoos-hotspot.service /usr/local/bin/azmod
rm -rf /usr/share/azmoos /usr/local/lib/azmoos
systemctl daemon-reload
echo "Azmo OS est désinstallé."
echo "Conservés : /DATA (tes fichiers), /etc/azmoos (comptes), Docker et les conteneurs."
echo "Pour tout effacer : sudo rm -rf /etc/azmoos /DATA"
