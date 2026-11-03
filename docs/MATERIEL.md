# Compatibilité matérielle

Azmo OS détecte automatiquement ton Raspberry Pi au démarrage et adapte son comportement.

| Modèle | RAM | OS conseillé | Profil Azmo OS | Usage conseillé |
|---|---|---|---|---|
| **Raspberry Pi 5** | 8 Go | 64 bits | Performance | Tout : Immich, Nextcloud, Jellyfin, Home Assistant en parallèle |
| Raspberry Pi 5 | 4 Go | 64 bits | Performance | Quasiment tout |
| Raspberry Pi 5 | 2 Go | 64 bits | Standard | 4 à 6 apps légères ou moyennes |
| Raspberry Pi 4 B / 400 | 4–8 Go | 64 bits | Performance | Très polyvalent |
| Raspberry Pi 4 B | 2 Go | 64 bits | Standard | Domotique, VPN, petit NAS |
| Raspberry Pi 4 B | 1 Go | 64 bits | Léger | Pi-hole, WireGuard, Mosquitto |
| Raspberry Pi 3 B / B+ | 1 Go | 64 bits | Léger | Apps légères uniquement |
| Raspberry Pi Zero 2 W | 512 Mo | 64 bits | Léger | 1 à 2 apps très légères (AdGuard, Tailscale) |
| Pi 1 / 2 / Zero / Zero W | ≤ 1 Go | 32 bits | Léger | Interface et fichiers ; App Store très limité |
| PC / mini-PC x86_64 | — | Debian 12+ | selon RAM | Tout |

## Ce que fait la détection

- **Modèle** : lu dans `/proc/device-tree/model`, affiché dans le tableau de bord et l'assistant.
- **Profil** :
  - *Léger* (< 1,5 Go) : rafraîchissement de l'interface toutes les 6 s, apps gourmandes **bloquées**
    (on peut forcer l'installation en connaissance de cause).
  - *Standard* (< 3,5 Go) : avertissement sur les apps qui demandent plus de RAM que disponible.
  - *Performance* : tout est disponible, rafraîchissement toutes les 2,5 s.
- **Architecture** : chaque app déclare les processeurs supportés ; une app sans version 32 bits
  est signalée « Incompatible » sur un OS 32 bits.
- **Température et bridage** : `vcgencmd get_throttled` signale une sous-alimentation ou une surchauffe.
- **Wi-Fi, Bluetooth, PCIe** (SSD NVMe sur Pi 5) sont détectés.

## Conseils

- **Pi 5** : utilise l'alimentation officielle 27 W, sinon les ports USB sont limités et les disques
  externes peuvent se déconnecter.
- **SSD** : pour Nextcloud, Immich ou Jellyfin, un SSD est bien plus rapide et durable qu'une microSD.
- **Disques USB sur Pi 4** : branche-les sur les ports **bleus** (USB 3).
