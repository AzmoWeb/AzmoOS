#!/usr/bin/env bash
# AzmoOS — script d'installation — made by Azmo
# Usage : curl -fsSL https://raw.githubusercontent.com/azmoweb/azmoos/main/scripts/install.sh | sudo bash
set -euo pipefail

REPO="${AZMO_REPO:-azmoweb/azmoos}"
VERSION="${AZMO_VERSION:-latest}"
BIN_SRC="${AZMO_BIN:-}"   # chemin vers un binaire local (optionnel)

c_ok="\033[1;32m"; c_info="\033[1;36m"; c_err="\033[1;31m"; c_off="\033[0m"
info() { echo -e "${c_info}==>${c_off} $*"; }
ok()   { echo -e "${c_ok}✔${c_off} $*"; }
die()  { echo -e "${c_err}✘ $*${c_off}" >&2; exit 1; }

cat <<'BANNER'

     _                         ___  ____
    / \    _____ __ ___   ___ / _ \/ ___|
   / _ \  |_  / '_ ` _ \ / _ \ | | \___ \
  / ___ \  / /| | | | | | (_) | |_| |___) |
 /_/   \_\/___|_| |_| |_|\___/ \___/|____/
                              made by Azmo

BANNER

[ "$(id -u)" -eq 0 ] || die "Lancez ce script avec sudo."
command -v systemctl >/dev/null || die "systemd est requis."

case "$(uname -m)" in
  aarch64|arm64) ARCH=arm64 ;;
  armv6l|armv7l) ARCH=armv6 ;;
  x86_64|amd64)  ARCH=amd64 ;;
  *) die "Architecture non supportée : $(uname -m)" ;;
esac
MODEL="$(tr -d '\0' < /proc/device-tree/model 2>/dev/null || echo 'machine générique')"
info "Matériel détecté : $MODEL ($ARCH)"
[ "$ARCH" = armv6 ] && echo "   ⚠️  OS 32 bits : beaucoup d'apps de l'App Store n'existent qu'en 64 bits. Raspberry Pi OS 64 bits recommandé (Pi 3/4/5/Zero 2)."

info "Installation des dépendances…"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq curl ca-certificates avahi-daemon >/dev/null
ok "Dépendances installées (avahi : accès via http://$(hostname).local)"

if ! command -v docker >/dev/null; then
  info "Installation de Docker…"
  curl -fsSL https://get.docker.com | sh
fi
if ! docker compose version >/dev/null 2>&1; then
  apt-get install -y -qq docker-compose-plugin >/dev/null || die "Impossible d'installer docker compose."
fi
systemctl enable --now docker >/dev/null
ok "Docker $(docker --version | awk '{print $3}' | tr -d ,) prêt"

info "Installation d'AzmoOS…"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
if [ -n "$BIN_SRC" ]; then
  cp "$BIN_SRC" "$TMP/azmod"
else
  if [ "$VERSION" = latest ]; then URL="https://github.com/$REPO/releases/latest/download/azmod-linux-$ARCH"
  else URL="https://github.com/$REPO/releases/download/$VERSION/azmod-linux-$ARCH"; fi
  curl -fL --progress-bar "$URL" -o "$TMP/azmod" || die "Téléchargement impossible : $URL"
fi
install -m 0755 "$TMP/azmod" /usr/local/bin/azmod
mkdir -p /DATA /etc/azmoos
chmod 700 /etc/azmoos

if ss -ltnH 2>/dev/null | awk '{print $4}' | grep -qE '(^|:)80$' && ! systemctl is-active -q azmoos; then
  echo "   ⚠️  Le port 80 est déjà utilisé. Modifiez -addr dans /etc/systemd/system/azmoos.service si besoin."
fi

cat > /etc/systemd/system/azmoos.service <<'UNIT'
[Unit]
Description=AzmoOS — tableau de bord serveur (made by Azmo)
After=network-online.target docker.service
Wants=network-online.target docker.service

[Service]
Type=simple
ExecStart=/usr/local/bin/azmod -addr :80 -data /DATA -config /etc/azmoos
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable azmoos >/dev/null
systemctl restart azmoos
ok "$(/usr/local/bin/azmod -version) installé et démarré"

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
echo -e "${c_ok}🎉 AzmoOS est prêt !${c_off}"
echo "   Ouvrez : http://${IP:-<ip-du-pi>}   ou   http://$(hostname).local"
echo "   Créez votre compte administrateur au premier accès."
echo
