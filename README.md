<p align="center"><img src="web/static/logo.svg" width="96" alt="AzmoOS"></p>
<h1 align="center">AzmoOS</h1>
<p align="center"><b>Votre cloud personnel sur Raspberry Pi — made by Azmo</b></p>

AzmoOS transforme un Raspberry Pi (ou n'importe quel PC Linux) en **serveur domestique** piloté depuis une belle interface web : tableau de bord matériel, App Store en un clic, gestionnaire de fichiers, gestion Docker et réglages système.

## ✨ Fonctionnalités

| | |
|---|---|
| 📊 **Tableau de bord** | CPU, RAM, **température**, bridage (`vcgencmd`), stockage, réseau en temps réel, modèle du Pi |
| 🛍️ **App Store** | 23 apps prêtes : Jellyfin, Plex, Nextcloud, Home Assistant, Pi-hole, AdGuard, Vaultwarden, WireGuard, Syncthing, Gitea, Uptime Kuma, Portainer… |
| 🧩 **Apps perso** | Installez n'importe quel `docker-compose.yml`, modifiez-le et redéployez depuis l'interface |
| 📁 **Fichiers** | Parcourir, envoyer (glisser-déposer), télécharger, renommer, supprimer dans `/DATA` |
| 🐳 **Conteneurs** | Démarrer / arrêter / redémarrer / supprimer, logs en direct |
| ⚙️ **Paramètres** | Nom d'hôte, compte admin, redémarrer / éteindre |
| 🔒 **Sécurité** | Mot de passe PBKDF2-SHA256, sessions HttpOnly, protection CSRF, fichiers confinés à `/DATA` |
| 🌗 **Thème** | Sombre / clair, responsive mobile |

Un **seul binaire Go statique (~7 Mo)**, sans aucune dépendance externe, interface web embarquée.

## 🍓 Compatibilité

| Matériel | Binaire | App Store |
|---|---|---|
| Raspberry Pi 5, 4, 400, 500, 3B/3B+, Zero 2 W, CM4/CM5 — **OS 64 bits** (recommandé) | `arm64` | ✅ toutes les apps |
| Raspberry Pi 1, 2, Zero / Zero W, ou OS 32 bits | `armv6` | ⚠️ apps marquées `arm` uniquement |
| PC / mini-PC x86_64 (Debian, Ubuntu…) | `amd64` | ✅ |

## 🚀 Installation

### Option 1 — Sur un Raspberry Pi OS existant (le plus simple)

1. Flashez **Raspberry Pi OS Lite (64-bit)** avec [Raspberry Pi Imager](https://www.raspberrypi.com/software/) (activez SSH).
2. Connectez-vous en SSH puis :

```bash
curl -fsSL https://raw.githubusercontent.com/azmoweb/azmoos/main/scripts/install.sh | sudo bash
```

3. Ouvrez **http://\<ip-du-pi\>** ou **http://\<nom\>.local** et créez votre compte.

Le script installe Docker, avahi (`.local`), le binaire `azmod` et le service systemd `azmoos`.

### Option 2 — Image AzmoOS prête à flasher

Téléchargez `AzmoOS-*.img.xz` depuis les [Releases](https://github.com/azmoweb/azmoos/releases), flashez-la avec Raspberry Pi Imager (« Use custom »), démarrez, puis ouvrez **http://azmoos.local**.

- Utilisateur SSH : `azmo` / mot de passe `azmoos` → **changez-le** (`passwd`)
- Clavier/locale : français, fuseau Europe/Paris

## 🛠️ Développement

Prérequis : Go 1.24+.

```bash
make run          # http://localhost:8080 (données dans .dev/)
make test         # vet + tests
make build-all    # dist/azmod-linux-{arm64,armv6,amd64}
make image        # image Raspberry Pi via pi-gen (Docker, ~20 Go, 30-60 min)
```

Déployer un binaire compilé sur un Pi :

```bash
make build-arm64
scp dist/azmod-linux-arm64 azmo@azmoos.local:/tmp/azmod
ssh azmo@azmoos.local 'sudo AZMO_BIN=/tmp/azmod bash -s' < scripts/install.sh
```

### Publier une version

```bash
git tag v0.1.0 && git push origin v0.1.0
```

GitHub Actions compile les 3 binaires, crée la Release **et** construit l'image `.img.xz`.
(L'image peut aussi être lancée à la main : *Actions → Build AzmoOS → Run workflow → image ✓*.)

## 📂 Structure

```
cmd/azmod/            point d'entrée du démon
internal/auth/        compte admin, sessions
internal/config/      configuration persistante (/etc/azmoos/config.json)
internal/system/      stats matérielles (Pi : modèle, température, bridage)
internal/docker/      client Docker via le socket Unix (sans SDK)
internal/store/       App Store + apps/*.json (catalogue)
internal/files/       gestionnaire de fichiers sécurisé
internal/server/      API REST + service de l'interface
web/static/           interface web (HTML/CSS/JS sans framework)
scripts/              install.sh, uninstall.sh, service systemd
image/                stage pi-gen pour construire l'image AzmoOS
```

## ➕ Ajouter une app au catalogue

Créez `internal/store/apps/<id>.json` :

```json
{
  "id": "monapp",
  "name": "Mon App",
  "tagline": "Une phrase",
  "description": "Description complète",
  "category": "Outils",
  "icon": "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/monapp.png",
  "port": 8080,
  "arch": ["arm64", "amd64"],
  "compose": "services:\n  monapp:\n    image: ...\n    ports:\n      - \"8080:80\"\n    volumes:\n      - {{APPDATA}}/data:/data\n"
}
```

Variables disponibles : `{{APPDATA}}` (`/DATA/AppData/<id>`), `{{DATA}}` (`/DATA`), `{{TZ}}`, `{{PUID}}`, `{{PGID}}`.

## 📍 Emplacements sur le Pi

| Chemin | Contenu |
|---|---|
| `/usr/local/bin/azmod` | binaire |
| `/etc/azmoos/config.json` | compte et sessions |
| `/DATA` | vos fichiers (Media, Documents, Downloads, Backups) |
| `/DATA/AppData/<app>` | données des applications |
| `/DATA/AppData/.azmoos/apps/<app>/docker-compose.yml` | définition des apps installées |

Logs : `journalctl -u azmoos -f` · Désinstaller : `sudo bash scripts/uninstall.sh` (vos données sont conservées).

**Mot de passe oublié ?** `sudo systemctl stop azmoos && sudo python3 -c "import json;p='/etc/azmoos/config.json';c=json.load(open(p));c.pop('user',None);json.dump(c,open(p,'w'))" && sudo systemctl start azmoos` puis recréez le compte dans l'interface.

## 🗺️ Feuille de route

- [ ] Terminal web intégré
- [ ] Mises à jour d'AzmoOS depuis l'interface
- [ ] Gestion des disques USB (montage, formatage)
- [ ] Partages réseau Samba
- [ ] Sauvegarde / restauration de la configuration
- [ ] Multi-utilisateurs

---

<p align="center">Fait avec ❤️ — <b>made by Azmo</b></p>
