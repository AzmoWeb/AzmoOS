#!/usr/bin/env bash
# Copie le binaire arm64, l'interface et les services dans le stage pi-gen.
set -euo pipefail
cd "$(dirname "$0")/.."
F=image/stage-azmo/01-azmoos/files
cp dist/azmod-linux-arm64 "$F/azmod"
cp scripts/azmoos.service scripts/azmoos-hotspot.service scripts/azmoos-hotspot.sh "$F/"
rm -rf "$F/web" && cp -r web/dist "$F/web" && rm -f "$F/web/.keep"
chmod +x image/stage-azmo/*.sh image/stage-azmo/*/*.sh
echo "Stage Azmo OS prêt."
