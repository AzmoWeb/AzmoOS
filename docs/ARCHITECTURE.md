# Architecture d'Azmo OS

> Azmo OS — Made by Azmo. Ce document décrit comment le système est construit et pourquoi.

## Vue d'ensemble

```
┌──────────────────────────────── Raspberry Pi ────────────────────────────────┐
│                                                                              │
│  Navigateur ──HTTP──▶  azmod (démon Go, root, port 80)                       │
│  (PC, tablette,        ├── API REST /api/*  (auth, permissions, journal)     │
│   smartphone)          ├── Interface React servie depuis                     │
│                        │     /usr/share/azmoos/web  (si présente)            │
│                        │     ou embarquée dans le binaire (secours)          │
│                        └── Modules                                           │
│                             system · hardware · docker · store · files       │
│                             storage · shares · network · firewall            │
│                             backup · updater · users · events · jobs         │
│                                 │                                            │
│          ┌──────────────────────┼─────────────────────────────┐              │
│          ▼                      ▼                             ▼              │
│   Docker Engine          Outils système Debian          Fichiers            │
│   + Compose v2           lsblk mount mkfs nmcli         /etc/azmoos          │
│   (apps)                 ufw samba systemctl apt        /DATA                │
│                                                                              │
│  Raspberry Pi OS Lite 64 bits (Debian) — NetworkManager, avahi (.local)      │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Choix techniques

| Couche | Technologie | Pourquoi |
|---|---|---|
| Base | Raspberry Pi OS Lite 64 bits (Debian) | Support officiel de tous les Pi, firmware, NetworkManager |
| Backend | **Go**, binaire statique sans dépendance | ~10 Mo de RAM, démarre en 50 ms, tourne même sur un Pi Zero 2 W |
| Frontend | **React 19 + TypeScript + Vite** | Moderne, typé, rapide à charger (~180 Ko gzip, terminal compris) |
| Langues | Français (source) et anglais, `t("texte français")` | Code lisible, traductions vérifiées en CI |
| Apps | **Docker + Docker Compose v2** | Isolation, catalogue facile, mises à jour simples |
| Données | Fichiers JSON (config) + JSONL (journal) | Pas de serveur de base de données : léger, lisible, sauvegardable |
| Partage | Samba (SMB) | Compatible Windows, macOS, Linux, Android, iOS |
| Pare-feu | ufw | Simple, standard sur Debian |
| Image | pi-gen (outil officiel Raspberry Pi) | Même processus que Raspberry Pi OS |

## Séparation interface / services

- Le **backend** (`azmod`) n'expose qu'une API JSON. Il ne connaît rien de l'interface.
- L'**interface** est un site statique (React compilé). Elle est cherchée dans
  `/usr/share/azmoos/web` ; si ce dossier est absent, la copie embarquée dans le binaire est utilisée.
  On peut donc mettre à jour l'interface seule, sans toucher au backend.
- Les **applications** tournent dans Docker, totalement indépendantes d'Azmo OS :
  si `azmod` s'arrête, Jellyfin, Home Assistant, etc. continuent de tourner.

## Modules backend (`internal/`)

| Module | Rôle |
|---|---|
| `config` | Lecture/écriture atomique de `/etc/azmoos/config.json` (comptes, sessions, partages, réglages) |
| `auth` | Comptes, rôles et permissions, sessions, limitation des tentatives de connexion |
| `events` | Journal des événements (`/etc/azmoos/events.jsonl`, rotation automatique) |
| `jobs` | Exécution de tâches longues en arrière-plan avec journal (installations, mises à jour, formatage) |
| `system` | Statistiques temps réel, **détection du matériel** et profil de performance, alimentation, SSH, mises à jour APT |
| `docker` | Client Docker via le socket Unix : conteneurs, ressources, logs, images, volumes |
| `store` | Catalogue d'applications, installation via Docker Compose, apps personnalisées |
| `files` | Explorateur de fichiers confiné à `/DATA` |
| `storage` | Disques et clés USB : détection, montage, démontage, formatage, montage au démarrage |
| `shares` | Partages réseau Samba (NAS) |
| `network` | Interfaces, Wi-Fi, IP fixe / DHCP via NetworkManager |
| `firewall` | Pare-feu ufw |
| `backup` | Sauvegarde/restauration de la configuration et des apps |
| `updater` | Mise à jour d'Azmo OS depuis les Releases GitHub (vérification SHA-256) |
| `totp` | Double authentification (RFC 6238) et codes de secours |
| `tlsmgr` | HTTPS local : autorité de certification à contrainte de nom, certificat serveur renouvelé automatiquement |
| `terminal` | Terminal web : pseudo-terminal Linux (PTY) et WebSocket (RFC 6455), sans dépendance |
| `notify` | Centre de notifications et envoi vers ntfy, Discord, Telegram |
| `watch` | Surveillance continue (toutes les minutes) qui déclenche les alertes |
| `backupsched` | Sauvegardes planifiées vers un dossier ou le cloud (rclone) |

## Adaptation au matériel

Au démarrage, `system.DetectHardware()` lit `/proc/device-tree/model` et la RAM, puis choisit un **profil** :

| Profil | Exemples | Effets |
|---|---|---|
| `leger` (< 1,5 Go) | Pi Zero 2 W, Pi 3 (1 Go) | Rafraîchissement plus lent, apps gourmandes signalées et bloquées par défaut |
| `standard` (< 3,5 Go) | Pi 4 2 Go, Pi 5 2 Go | Avertissement sur les apps très gourmandes |
| `performance` | Pi 4/5 4–8 Go | Tout est disponible |

Chaque app du catalogue déclare `minRam` (Mo) et les architectures supportées. L'interface compare avec le matériel détecté.

## Sécurité

- Mots de passe : PBKDF2-SHA256 (210 000 itérations) + sel unique.
- Sessions : jeton aléatoire 256 bits, cookie `HttpOnly` + `SameSite=Strict`, durée 30 jours.
- CSRF : en-tête `X-Requested-With: AzmoOS` obligatoire sur toute requête qui modifie l'état.
- Force brute : blocage d'une adresse IP pendant 5 min après 8 échecs.
- Rôles : `admin` (tout) et `membre` avec permissions fines (`apps.manage`, `files.read`, `files.write`).
- Fichiers : chemins confinés à `/DATA`, liens symboliques sortants refusés.
- Disques : le disque système ne peut ni être formaté ni démonté.
- Journal : connexions, installations, changements de configuration sont tracés.

## Arborescence sur le Pi

| Chemin | Contenu |
|---|---|
| `/usr/local/bin/azmod` | Backend |
| `/usr/share/azmoos/web` | Interface web |
| `/etc/azmoos/config.json` | Comptes, sessions, partages, réglages |
| `/etc/azmoos/events.jsonl` | Journal |
| `/DATA` | Données utilisateur (Media, Documents, Downloads, Backups) |
| `/DATA/AppData/<app>` | Données des applications |
| `/DATA/AppData/.azmoos/apps/<app>/docker-compose.yml` | Définition des apps installées |
| `/DATA/Disks/<nom>` | Disques externes montés |
| `/etc/samba/azmoos-shares.conf` | Partages réseau générés |
| `/etc/azmoos/tls/` | Autorité locale et certificat HTTPS |
| `/etc/azmoos/notifications.json` | Historique des notifications |
| `/etc/azmoos/rclone.conf` | Comptes cloud des sauvegardes (rclone) |

## Premier démarrage (image)

1. Le Pi démarre, obtient une IP par DHCP (Ethernet ou Wi-Fi configuré dans Raspberry Pi Imager).
2. Sans réseau après 90 s, `azmoos-hotspot` crée un point d'accès Wi-Fi **« AzmoOS-Setup »**
   (mot de passe `azmoos-setup`) ; l'interface est alors sur `http://10.42.0.1`.
3. L'utilisateur ouvre `http://azmoos.local` : l'**assistant** lui fait créer son compte,
   nommer son Pi et choisir son Wi-Fi.

## Feuille de route

- **v0.1** : tableau de bord, App Store, fichiers, conteneurs ✅
- **v0.2** : interface React/TS, multi-utilisateurs, disques USB, NAS Samba, réseau/Wi-Fi, pare-feu,
  sauvegardes, journal, mises à jour, assistant de premier démarrage, adaptation au matériel ✅
- **v0.3** : terminal web, sauvegardes planifiées, notifications, 2FA, HTTPS local, interface en anglais ✅
- **v0.4** : widgets personnalisables, mises à jour automatiques des apps, catalogue communautaire, autres langues
