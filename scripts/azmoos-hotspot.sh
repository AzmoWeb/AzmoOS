#!/usr/bin/env bash
# Azmo OS — point d'accès de configuration.
# Si le Pi n'a aucune connexion réseau 90 s après le démarrage, il crée le Wi-Fi
# « AzmoOS-Setup » : on s'y connecte avec un téléphone et on ouvre http://10.42.0.1
# pour choisir son vrai Wi-Fi dans l'interface. Made by Azmo.
set -u
SSID="AzmoOS-Setup"
PASS="azmoos-setup"
CON="AzmoOS-Setup"

command -v nmcli >/dev/null || exit 0
sleep "${AZMO_HOTSPOT_DELAY:-90}"

state="$(nmcli -t networking connectivity 2>/dev/null || echo none)"
if [ "$state" != "none" ]; then
  # Déjà connecté : on s'assure que le point d'accès n'est pas actif.
  nmcli -t -f NAME connection show --active | grep -qx "$CON" && nmcli connection down "$CON" >/dev/null 2>&1
  exit 0
fi

IFACE="$(nmcli -t -f DEVICE,TYPE device | awk -F: '$2=="wifi"{print $1; exit}')"
[ -n "$IFACE" ] || exit 0

echo "Aucun réseau : création du point d'accès $SSID sur $IFACE"
nmcli connection delete "$CON" >/dev/null 2>&1 || true
nmcli device wifi hotspot ifname "$IFACE" con-name "$CON" ssid "$SSID" password "$PASS"
nmcli connection modify "$CON" connection.autoconnect no
