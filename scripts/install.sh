#!/usr/bin/env bash
# Azmo OS — installation sur Raspberry Pi OS / Debian / Ubuntu — Made by Azmo
# Usage : curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo bash
#
# Variables optionnelles :
#   AZMO_VERSION=v0.2.0     version à installer (défaut : la dernière)
#   AZMO_BIN=/chemin/azmod  binaire local (développement)
#   AZMO_WEB=/chemin/web    dossier de l'interface compilée (développement)
#   AZMO_MINIMAL=1          ne pas installer Samba, ufw ni le point d'accès
#   AZMO_USER=azmo AZMO_PASSWORD=… [AZMO_NAME="Azmo"] [AZMO_HOSTNAME=azmoos]
#                           installation sans questions (automatisation)
set -euo pipefail

REPO="${AZMO_REPO:-AzmoWeb/AzmoOS}"
VERSION="${AZMO_VERSION:-latest}"
RAW="https://raw.githubusercontent.com/$REPO/main/scripts"

c_ok="\033[1;32m"; c_info="\033[1;35m"; c_warn="\033[1;33m"; c_err="\033[1;31m"; c_off="\033[0m"
step() { echo -e "\n${c_info}●${c_off} $*"; }
ok()   { echo -e "  ${c_ok}✔${c_off} $*"; }
warn() { echo -e "  ${c_warn}!${c_off} $*"; }
die()  { echo -e "\n${c_err}✘ $*${c_off}" >&2; exit 1; }

cat <<'BANNER'

     _                         ___  ____
    / \    _____ __ ___   ___ / _ \/ ___|
   / _ \  |_  / '_ ` _ \ / _ \ | | \___ \
  / ___ \  / /| | | | | | (_) | |_| |___) |
 /_/   \_\/___|_| |_| |_|\___/ \___/|____/
                              Made by Azmo

BANNER

[ "$(id -u)" -eq 0 ] || die "Lance ce script avec sudo."
command -v systemctl >/dev/null || die "systemd est requis."
command -v apt-get >/dev/null || die "Ce script nécessite une distribution Debian (Raspberry Pi OS, Debian, Ubuntu)."

case "$(uname -m)" in
  aarch64|arm64) ARCH=arm64 ;;
  armv6l|armv7l) ARCH=armv6 ;;
  x86_64|amd64)  ARCH=amd64 ;;
  *) die "Processeur non supporté : $(uname -m)" ;;
esac
MODEL="$(tr -d '\0' < /proc/device-tree/model 2>/dev/null || echo 'Ordinateur générique')"
RAM_MB=$(( $(awk '/MemTotal/{print $2}' /proc/meminfo) / 1024 ))

step "Matériel détecté"
ok "$MODEL — $ARCH — ${RAM_MB} Mo de RAM"
[ "$ARCH" = armv6 ] && warn "Système 32 bits : une partie de l'App Store sera indisponible. Raspberry Pi OS 64 bits est conseillé (Pi 3, 4, 5, Zero 2 W)."
[ "$RAM_MB" -lt 1536 ] && warn "Peu de mémoire : Azmo OS activera le mode léger."
FREE_MB=$(df -Pm / | awk 'NR==2{print $4}')
[ "$FREE_MB" -lt 2048 ] && warn "Moins de 2 Go libres sur la carte SD : prévois un disque externe pour les applications."

step "Installation des dépendances"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
PKGS="curl ca-certificates avahi-daemon"
[ "${AZMO_MINIMAL:-0}" = 1 ] || PKGS="$PKGS samba ufw exfatprogs parted rclone"
# shellcheck disable=SC2086
apt-get install -y -qq $PKGS >/dev/null
ok "Paquets système installés"
if ! command -v nmcli >/dev/null; then
  warn "NetworkManager absent : la gestion Wi-Fi depuis l'interface sera limitée."
fi

if ! command -v docker >/dev/null; then
  step "Installation de Docker"
  curl -fsSL https://get.docker.com | sh >/dev/null
fi
if ! docker compose version >/dev/null 2>&1; then
  apt-get install -y -qq docker-compose-plugin >/dev/null || die "Impossible d'installer docker compose."
fi
systemctl enable --now docker >/dev/null 2>&1
ok "Docker $(docker --version | awk '{print $3}' | tr -d ,) prêt"
if id -nu 1000 >/dev/null 2>&1; then usermod -aG docker "$(id -nu 1000)" || true; fi

step "Installation d'Azmo OS"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
if [ "$VERSION" = latest ]; then BASE="https://github.com/$REPO/releases/latest/download"
else BASE="https://github.com/$REPO/releases/download/$VERSION"; fi

if [ -n "${AZMO_BIN:-}" ]; then
  cp "$AZMO_BIN" "$TMP/azmod"
else
  curl -fsSL "$BASE/azmod-linux-$ARCH" -o "$TMP/azmod" || die "Téléchargement impossible : $BASE/azmod-linux-$ARCH"
  if curl -fsSL "$BASE/SHA256SUMS" -o "$TMP/SHA256SUMS" 2>/dev/null; then
    want=$(awk -v f="azmod-linux-$ARCH" '$2==f || $2=="*"f {print $1}' "$TMP/SHA256SUMS")
    got=$(sha256sum "$TMP/azmod" | awk '{print $1}')
    [ -n "$want" ] && [ "$want" != "$got" ] && die "Somme de contrôle incorrecte : fichier corrompu."
    ok "Intégrité vérifiée (SHA-256)"
  fi
fi
install -m 0755 "$TMP/azmod" /usr/local/bin/azmod

# Interface web (optionnelle : une copie est embarquée dans le binaire).
mkdir -p /usr/share/azmoos
if [ -n "${AZMO_WEB:-}" ]; then
  rm -rf /usr/share/azmoos/web && cp -r "$AZMO_WEB" /usr/share/azmoos/web
elif curl -fsSL "$BASE/azmoos-web.tar.gz" -o "$TMP/web.tar.gz" 2>/dev/null; then
  rm -rf /usr/share/azmoos/web && mkdir -p /usr/share/azmoos/web && tar -xzf "$TMP/web.tar.gz" -C /usr/share/azmoos/web
fi

mkdir -p /DATA /etc/azmoos /usr/local/lib/azmoos
chmod 700 /etc/azmoos

fetch() { # fichier du dépôt : local si on lance le script depuis un clone, sinon téléchargé
  local here; here="$(cd "$(dirname "${BASH_SOURCE[0]:-.}")" 2>/dev/null && pwd || true)"
  if [ -n "$here" ] && [ -f "$here/$1" ]; then cp "$here/$1" "$2"; else curl -fsSL "$RAW/$1" -o "$2"; fi
}
fetch azmoos.service /etc/systemd/system/azmoos.service
if [ "${AZMO_MINIMAL:-0}" != 1 ] && command -v nmcli >/dev/null; then
  fetch azmoos-hotspot.sh /usr/local/lib/azmoos/azmoos-hotspot.sh
  chmod 755 /usr/local/lib/azmoos/azmoos-hotspot.sh
  fetch azmoos-hotspot.service /etc/systemd/system/azmoos-hotspot.service
  systemctl enable azmoos-hotspot >/dev/null 2>&1 || true
fi

# ---------------------------------------------------------------------------
# Compte administrateur : on le crée maintenant, comme ça il suffit d'ouvrir
# l'adresse du Pi et de se connecter.
# ---------------------------------------------------------------------------
TTY=""
if [ -r /dev/tty ] && [ -w /dev/tty ] && { : < /dev/tty; } 2>/dev/null; then TTY=/dev/tty; fi
ask() { # ask "Question" "défaut" → $REPLY
  local q="$1" def="${2:-}"
  if [ -n "$def" ]; then printf "  %s ${c_info}[%s]${c_off} : " "$q" "$def" > "$TTY"; else printf "  %s : " "$q" > "$TTY"; fi
  IFS= read -r REPLY < "$TTY" || REPLY=""
  [ -z "$REPLY" ] && REPLY="$def"
  return 0
}
ask_secret() { # ask_secret "Question" → $REPLY
  printf "  %s : " "$1" > "$TTY"
  IFS= read -rs REPLY < "$TTY" || REPLY=""
  printf "\n" > "$TTY"
}

ACCOUNT_USER=""
if /usr/local/bin/azmod user list 2>/dev/null | grep -q "^Aucun compte"; then
  if [ -n "${AZMO_USER:-}" ] && [ -n "${AZMO_PASSWORD:-}" ]; then
    ACCOUNT_USER="$(echo "$AZMO_USER" | tr 'A-Z' 'a-z')"
    printf '%s\n' "$AZMO_PASSWORD" | /usr/local/bin/azmod user add "$ACCOUNT_USER" --display "${AZMO_NAME:-$AZMO_USER}" >/dev/null \
      || die "Impossible de créer le compte $ACCOUNT_USER (mot de passe de 8 caractères minimum)."
    [ -n "${AZMO_HOSTNAME:-}" ] && hostnamectl set-hostname "$AZMO_HOSTNAME"
    ok "Compte administrateur « $ACCOUNT_USER » créé"
  elif [ -n "$TTY" ]; then
    step "Création de ton compte"
    echo "  Ces identifiants te serviront à te connecter à l'interface web." > "$TTY"
    echo > "$TTY"
    ask "Comment tu t'appelles ?" "Azmo"; NAME="$REPLY"
    DEF_USER="$(echo "$NAME" | iconv -f utf-8 -t ascii//TRANSLIT 2>/dev/null | tr 'A-Z' 'a-z' | tr -cd 'a-z0-9_-' | cut -c1-32 || true)"
    case "$DEF_USER" in [a-z]*) ;; *) DEF_USER="azmo" ;; esac
    while :; do
      ask "Nom d'utilisateur" "$DEF_USER"; USERNAME="$(echo "$REPLY" | tr 'A-Z' 'a-z')"
      if echo "$USERNAME" | grep -Eq '^[a-z][a-z0-9_-]{1,31}$'; then break; fi
      warn "2 à 32 caractères : minuscules, chiffres, - ou _, en commençant par une lettre."
    done
    while :; do
      ask_secret "Mot de passe (8 caractères minimum)"; PW1="$REPLY"
      if [ "${#PW1}" -lt 8 ]; then warn "Trop court, encore un effort 🙂"; continue; fi
      ask_secret "Confirme le mot de passe"; PW2="$REPLY"
      [ "$PW1" = "$PW2" ] && break
      warn "Les deux mots de passe ne sont pas identiques, on recommence."
    done
    ask "Nom de ton Pi sur le réseau" "$(hostname)"; NEWHOST="$(echo "$REPLY" | tr -cd 'a-zA-Z0-9-' | cut -c1-63)"
    printf '%s\n' "$PW1" | /usr/local/bin/azmod user add "$USERNAME" --display "$NAME" >/dev/null || die "Impossible de créer le compte."
    unset PW1 PW2 REPLY
    ACCOUNT_USER="$USERNAME"
    ok "Compte administrateur « $USERNAME » créé"
    if [ -n "$NEWHOST" ] && [ "$NEWHOST" != "$(hostname)" ]; then
      OLDHOST="$(hostname)"
      hostnamectl set-hostname "$NEWHOST"
      sed -i "s/^127\.0\.1\.1.*/127.0.1.1\t$NEWHOST/" /etc/hosts 2>/dev/null || true
      systemctl restart avahi-daemon >/dev/null 2>&1 || true
      ok "Ton Pi s'appelle maintenant « $NEWHOST » (avant : $OLDHOST)"
    fi
  else
    warn "Pas de terminal interactif : tu créeras ton compte depuis le navigateur."
  fi
else
  ok "Comptes existants conservés ($(/usr/local/bin/azmod user list | wc -l) compte(s))"
fi

if ss -ltnH 2>/dev/null | awk '{print $4}' | grep -qE '(^|:)80$' && ! systemctl is-active -q azmoos; then
  warn "Le port 80 est déjà utilisé par un autre programme. Change -addr dans /etc/systemd/system/azmoos.service."
fi

systemctl daemon-reload
systemctl enable avahi-daemon >/dev/null 2>&1 || true
systemctl enable azmoos >/dev/null
systemctl restart azmoos
sleep 1
systemctl is-active -q azmoos || die "Le service ne démarre pas. Regarde : journalctl -u azmoos -n 50"
ok "$(/usr/local/bin/azmod -version) installé et démarré"

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
echo
echo -e "${c_ok}🎉 Azmo OS est prêt !${c_off}"
echo
echo -e "   Ouvre ton navigateur sur :  ${c_info}http://${IP:-<ip-du-pi>}${c_off}"
echo -e "                          ou :  ${c_info}http://$(hostname).local${c_off}"
echo
if [ -n "$ACCOUNT_USER" ]; then
  echo "   Connecte-toi avec l'utilisateur « $ACCOUNT_USER » et ton mot de passe."
else
  echo "   L'assistant va te guider pour créer ton compte."
fi
echo
echo "   Commandes utiles :  sudo azmod user help        (comptes, 2FA)"
echo "                       sudo journalctl -u azmoos -f (journaux)"
echo
