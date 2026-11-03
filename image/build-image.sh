#!/usr/bin/env bash
# Construit l'image AzmoOS (.img.xz) avec pi-gen dans Docker.
# Prérequis : Docker, ~20 Go libres. Durée : 30 à 60 min.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$PWD"
make build-arm64
cp dist/azmod-linux-arm64 image/stage-azmo/01-azmoos/files/azmod
cp scripts/azmoos.service image/stage-azmo/01-azmoos/files/azmoos.service

WORK="${PIGEN_DIR:-$ROOT/.pi-gen}"
if [ ! -d "$WORK" ]; then
  git clone --depth 1 --branch arm64 https://github.com/RPi-Distro/pi-gen.git "$WORK"
fi
rm -rf "$WORK/stage-azmo"
cp -r image/stage-azmo "$WORK/stage-azmo"
cp image/config "$WORK/config"
touch "$WORK/stage2/SKIP_IMAGES"   # on n'exporte que l'image AzmoOS
chmod +x "$WORK"/stage-azmo/*.sh "$WORK"/stage-azmo/*/*.sh
cd "$WORK"
./build-docker.sh
echo "Image disponible dans : $WORK/deploy/"
