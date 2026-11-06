#!/usr/bin/env bash
# Construit l'image Azmo OS (.img.xz) avec pi-gen dans Docker.
# Prérequis : Docker, ~20 Go libres. Durée : 30 à 60 min.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$PWD"
make web build-arm64
./image/prepare-stage.sh
WORK="${PIGEN_DIR:-$ROOT/.pi-gen}"
[ -d "$WORK" ] || git clone --depth 1 --branch arm64 https://github.com/RPi-Distro/pi-gen.git "$WORK"
rm -rf "$WORK/stage-azmo"
cp -r image/stage-azmo "$WORK/stage-azmo"
cp image/config "$WORK/config"
touch "$WORK/stage2/SKIP_IMAGES"
cd "$WORK" && ./build-docker.sh
echo "Image disponible dans : $WORK/deploy/"
