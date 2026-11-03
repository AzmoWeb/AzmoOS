<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="branding/logotype-sombre.svg">
  <img src="branding/logotype-clair.svg" width="420" alt="Azmo OS">
</picture>

<br>

**Ton serveur personnel sur Raspberry Pi. Simple, beau, et à toi.**

Films, photos, fichiers, domotique, VPN, NAS… pilotés depuis une interface web qui te parle comme un humain.

[![Licence MIT](https://img.shields.io/badge/licence-MIT-8b7bff)](LICENSE)
[![Raspberry Pi](https://img.shields.io/badge/Raspberry%20Pi-5%20%C2%B7%204%20%C2%B7%203%20%C2%B7%20Zero%202-c51a4a?logo=raspberrypi&logoColor=white)](docs/MATERIEL.md)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](go.mod)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black)](frontend/package.json)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?logo=typescript&logoColor=white)](frontend/tsconfig.json)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](internal/store/apps)

[**Installation**](#-installation) · [**Aperçu**](#-aperçu) · [**Fonctionnalités**](#-fonctionnalités) · [**Applications**](#-les-applications) · [**Documentation**](#-documentation) · [**Logo**](branding/)

<br>

<img src="docs/images/tableau-de-bord.webp" alt="Tableau de bord d'Azmo OS" width="100%">

</div>

---

## 🚀 Installation

### En une commande, par SSH

Sur un Raspberry Pi avec **Raspberry Pi OS** (64 bits conseillé) :

```bash
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo bash
```

L'installateur détecte ton Pi, installe tout ce qu'il faut, puis **te pose quelques questions** :

```text
● Matériel détecté
  ✔ Raspberry Pi 5 Model B Rev 1.0 — arm64 — 8064 Mo de RAM

● Installation des dépendances
  ✔ Paquets système installés
  ✔ Docker 28.5.1 prêt

● Installation d'Azmo OS
  ✔ Intégrité vérifiée (SHA-256)

● Création de ton compte
  Ces identifiants te serviront à te connecter à l'interface web.

  Comment tu t'appelles ? [Azmo] : Azmo
  Nom d'utilisateur [azmo] :
  Mot de passe (8 caractères minimum) :
  Confirme le mot de passe :
  Nom de ton Pi sur le réseau [raspberrypi] : azmoos
  ✔ Compte administrateur « azmo » créé
  ✔ Ton Pi s'appelle maintenant « azmoos » (avant : raspberrypi)
  ✔ Azmo OS v0.2.0 installé et démarré

🎉 Azmo OS est prêt !

   Ouvre ton navigateur sur :  http://192.168.1.42
                          ou :  http://azmoos.local

   Connecte-toi avec l'utilisateur « azmo » et ton mot de passe.
```

Ouvre l'adresse affichée depuis n'importe quel appareil de la maison, connecte-toi : c'est prêt. ✨

### Pas à pas, depuis zéro

**1. Prépare la carte microSD (ou le SSD)**

1. Installe [Raspberry Pi Imager](https://www.raspberrypi.com/software/) sur ton ordinateur.
2. Choisis ton modèle de Pi, puis **Raspberry Pi OS Lite (64-bit)**.
3. Dans les réglages (⚙️) : donne un nom d'utilisateur et un mot de passe, **active SSH**,
   et renseigne ton Wi-Fi si tu n'utilises pas de câble Ethernet.
4. Écris la carte, insère-la dans le Pi et allume-le. Attends 2 minutes.

**2. Connecte-toi au Pi en SSH**

Depuis le Terminal (Mac / Linux) ou PowerShell (Windows) :

```bash
ssh pi@raspberrypi.local
```

> Remplace `pi` par le nom d'utilisateur choisi dans Imager. Si `raspberrypi.local` ne répond pas,
> utilise l'adresse IP du Pi (visible dans l'interface de ta box).

**3. Installe Azmo OS**

```bash
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo bash
```

**4. Ouvre ton navigateur** sur `http://<ip-du-pi>` (affichée à la fin de l'installation) et connecte-toi. 🎉

### Ou avec l'image prête à flasher

Télécharge `AzmoOS-….img.xz` dans les [Releases](https://github.com/AzmoWeb/AzmoOS/releases), flashe-la avec
Raspberry Pi Imager (*Utiliser une image personnalisée*), démarre, puis ouvre **http://azmoos.local** :
un assistant te guide pour créer ton compte. Sans réseau, le Pi crée son propre Wi-Fi
**« AzmoOS-Setup »** (mot de passe `azmoos-setup`), l'interface est alors sur `http://10.42.0.1`.

### Commandes utiles

| Action | Commande |
|---|---|
| Mettre à jour Azmo OS | relancer la commande d'installation, ou *Mises à jour* dans l'interface |
| Lister les comptes | `sudo azmod user list` |
| Ajouter un compte | `sudo azmod user add camille --member` |
| Mot de passe oublié | `sudo azmod user passwd azmo` |
| État du service | `sudo systemctl status azmoos` |
| Journaux en direct | `sudo journalctl -u azmoos -f` |
| Désinstaller (garde tes données) | `curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/uninstall.sh \| sudo bash` |

Installation sans questions (automatisation) :

```bash
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh \
  | sudo AZMO_USER=azmo AZMO_PASSWORD='MonMotDePasse' AZMO_HOSTNAME=azmoos bash
```

Toutes les options et le dépannage : **[docs/INSTALLATION.md](docs/INSTALLATION.md)**.

---

## 👀 Aperçu

| | |
|:---:|:---:|
| <img src="docs/images/assistant-1-bienvenue.webp" alt="Assistant de premier démarrage"> | <img src="docs/images/connexion.webp" alt="Connexion"> |
| **Assistant de premier démarrage** | **Connexion** |
| <img src="docs/images/app-store.webp" alt="App Store"> | <img src="docs/images/app-store-fiche.webp" alt="Fiche d'une application"> |
| **App Store** : 25 applications prêtes à l'emploi | **Fiche d'une app** : mémoire conseillée, conseils, configuration |
| <img src="docs/images/mes-applications.webp" alt="Mes applications"> | <img src="docs/images/details-application.webp" alt="Détails d'une application"> |
| **Mes applications** | **Ressources, ports, volumes, journaux** |
| <img src="docs/images/fichiers-films.webp" alt="Fichiers"> | <img src="docs/images/conteneurs.webp" alt="Conteneurs"> |
| **Fichiers** : glisser-déposer, aperçu, raccourcis | **Conteneurs, images et volumes Docker** |
| <img src="docs/images/utilisateurs.webp" alt="Utilisateurs"> | <img src="docs/images/securite.webp" alt="Sécurité"> |
| **Utilisateurs et permissions** | **Sécurité** : pare-feu, SSH, sessions |
| <img src="docs/images/journal.webp" alt="Journal"> | <img src="docs/images/sauvegardes.webp" alt="Sauvegardes"> |
| **Journal des événements** | **Sauvegarde et restauration** |
| <img src="docs/images/theme-clair.webp" alt="Thème clair"> | <img src="docs/images/reglages.webp" alt="Réglages"> |
| **Thème clair** | **Réglages** |

<p align="center">
  <img src="docs/images/mobile-accueil.webp" width="23%" alt="Accueil sur mobile">
  <img src="docs/images/mobile-applications.webp" width="23%" alt="Applications sur mobile">
  <img src="docs/images/mobile-menu.webp" width="23%" alt="Menu sur mobile">
  <img src="docs/images/mobile-store.webp" width="23%" alt="App Store sur mobile">
  <br><b>Pensé aussi pour le téléphone</b>
</p>

> Captures réalisées sur une machine de test x86 avec des données de démonstration : sur un Raspberry Pi,
> le tableau de bord affiche le modèle (« Raspberry Pi 5 Model B »), la température et le bridage.
> Toutes les captures : **[docs/GALERIE.md](docs/GALERIE.md)**.

---

## ✨ Fonctionnalités

### 🏠 Un tableau de bord qui parle comme un humain
L'état de ton Pi en une phrase : *« Tout roule ✨ »*, *« Il commence à faire chaud »*,
*« Le stockage est presque plein »*. Processeur, mémoire, température, réseau et stockage en temps réel.

### 🧠 Adapté à ton matériel
Azmo OS reconnaît ton Raspberry Pi et choisit un profil **léger**, **standard** ou **performance**.
L'App Store te prévient (ou te protège) si une application est trop gourmande pour ton modèle.

### 🛍️ App Store en un clic
25 applications prêtes à l'emploi, chacune isolée dans Docker. Une application personnalisée ?
Colle son `docker-compose.yml`. Si une installation échoue, Azmo OS **t'explique pourquoi**
en français et propose de réessayer.

### 📦 Gestion complète des applications
Démarrer, arrêter, redémarrer, mettre à jour, désinstaller (les données sont conservées),
consulter les journaux, la consommation CPU/RAM, les ports et les volumes, modifier la configuration.

### 📁 Fichiers et 💽 NAS
Explorateur de fichiers avec glisser-déposer. Disques et clés USB : utilisation, éjection, formatage,
montage au démarrage. Partages réseau **Samba** pour Windows, Mac, Linux, Android et iOS.

### 📶 Réseau
Ethernet et Wi-Fi, connexion à un nouveau réseau, IP fixe ou automatique.

### 🛡️ Sécurité
Pare-feu en un interrupteur, contrôle de SSH, sessions par appareil avec déconnexion à distance,
blocage anti force brute, journal de toutes les actions, conseils pour l'accès à distance (Tailscale, WireGuard).

### 👨‍👩‍👧 Multi-utilisateurs
Administrateurs et membres, avec des permissions précises : gérer les apps, voir ou modifier les fichiers.

### 💾 Sauvegardes et 🔄 mises à jour
Sauvegarde et restauration de toute la configuration en un clic. Mises à jour d'Azmo OS
(vérifiées par SHA-256) et du système, avec signalement quand un redémarrage est nécessaire.

### 🧭 Premier démarrage sans prise de tête
Questions posées pendant l'installation, ou assistant dans le navigateur avec l'image.
Point d'accès Wi-Fi de secours si le Pi n'a pas de réseau.

---

## 🍓 Compatibilité

| Modèle | Profil | Conseillé pour |
|---|---|---|
| **Raspberry Pi 5** (4 / 8 Go) | Performance | Tout : Immich, Nextcloud, Jellyfin, Home Assistant ensemble |
| Raspberry Pi 5 (2 Go) | Standard | 4 à 6 applications |
| Raspberry Pi 4 B / 400 (4 Go et +) | Performance | Très polyvalent |
| Raspberry Pi 4 B (1–2 Go) | Léger / Standard | Domotique, VPN, petit NAS |
| Raspberry Pi 3 B / B+ | Léger | Applications légères |
| Raspberry Pi Zero 2 W | Léger | 1 ou 2 apps (AdGuard, Tailscale) |
| PC / mini-PC x86_64 | selon la RAM | Tout |

Détails, conseils d'alimentation et de stockage : **[docs/MATERIEL.md](docs/MATERIEL.md)**.

---

## 🧩 Les applications

| Catégorie | Applications |
|---|---|
| 🎬 Multimédia | Jellyfin, Plex, Immich, Navidrome |
| ☁️ Cloud | Nextcloud, Syncthing |
| 🏡 Domotique | Home Assistant, Node-RED, Mosquitto |
| 🌐 Réseau | Pi-hole, AdGuard Home, Tailscale, WireGuard Easy, Nginx Proxy Manager |
| 🔒 Sécurité | Vaultwarden |
| 🧰 Outils | Uptime Kuma, Portainer, Grafana, Homepage, Paperless-ngx |
| 💻 Développement | Gitea, Code Server |
| ⬇️ Téléchargement | qBittorrent, Transmission |
| 💾 Sauvegarde | Duplicati |

Envie d'en ajouter une ? Un simple fichier JSON suffit, voir [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 🏗️ Comment c'est construit

```text
 Navigateur (PC, tablette, téléphone)
        │  HTTP :80
        ▼
 ┌──────────────────────────────────────────────┐
 │ azmod — binaire Go unique, sans dépendance   │
 │  ├─ API REST sécurisée (/api, permissions)   │
 │  ├─ Interface React (/usr/share/azmoos/web)  │
 │  └─ Modules : système · matériel · apps ·    │
 │     fichiers · disques · partages · réseau · │
 │     pare-feu · sauvegardes · mises à jour ·  │
 │     utilisateurs · journal                   │
 └───────┬──────────────────┬───────────────────┘
         ▼                  ▼
  Docker + Compose     Raspberry Pi OS (Debian)
  (tes applications)   NetworkManager · Samba · ufw · avahi
```

| | |
|---|---|
| **Backend** | Go 1.24, bibliothèque standard uniquement. ~7,5 Mo, ~15 Mo de RAM, tourne même sur un Pi Zero 2 W |
| **Interface** | React 19, TypeScript, Vite. ~110 Ko compressés, mise à jour indépendamment du backend |
| **Applications** | Docker et Docker Compose v2 |
| **Données** | JSON pour la configuration, JSONL pour le journal : léger, lisible, facile à sauvegarder |
| **Image** | pi-gen, l'outil officiel de la fondation Raspberry Pi |

Les choix et le fonctionnement interne : **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)**.

---

## 🛠️ Développement

```bash
git clone https://github.com/AzmoWeb/AzmoOS.git && cd AzmoOS
make run     # Azmo OS en local sur http://localhost:8080
make dev     # interface avec rechargement à chaud (dans un 2e terminal)
make test    # tests Go + vérification TypeScript
make help    # toutes les commandes
```

```text
cmd/azmod/        point d'entrée et commandes « azmod user »
internal/         modules du backend (auth, docker, store, storage, network…)
frontend/         interface React + TypeScript
scripts/          installation, désinstallation, services systemd
image/            construction de l'image Raspberry Pi
docs/             documentation
```

Publier une version : `git tag v0.2.0 && git push origin v0.2.0`. GitHub Actions compile les binaires
(arm64, armv6, amd64), publie la Release avec `SHA256SUMS`, puis construit l'image `.img.xz`.

---

## 📚 Documentation

| | |
|---|---|
| [Installation](docs/INSTALLATION.md) | SSH, image, options, comptes, dépannage |
| [Guide d'utilisation](docs/GUIDE.md) | Toutes les fonctionnalités, pas à pas |
| [Matériel](docs/MATERIEL.md) | Modèles supportés, profils, conseils |
| [Sécurité](docs/SECURITE.md) | Protections en place et bonnes pratiques |
| [Architecture](docs/ARCHITECTURE.md) | Comment c'est construit |
| [Développement](docs/DEVELOPPEMENT.md) | Contribuer, ajouter une app, publier |
| [API](docs/API.md) | Référence de l'API REST |
| [Galerie](docs/GALERIE.md) | Toutes les captures d'écran |
| [Identité visuelle](branding/) | Logo, couleurs, typographie, fichiers à télécharger |
| [Versions](CHANGELOG.md) | Nouveautés de chaque version |

---

## 🗺️ Feuille de route

- [x] Tableau de bord, App Store, fichiers, conteneurs
- [x] Interface React, multi-utilisateurs, NAS, réseau, pare-feu, sauvegardes, journal
- [x] Installation interactive en une commande, image prête à flasher
- [ ] Terminal web intégré
- [ ] Sauvegardes planifiées vers un disque ou le cloud
- [ ] Double authentification et HTTPS local
- [ ] Notifications (température, disque plein, mises à jour)
- [ ] Interface en anglais

## 🤝 Contribuer

Les contributions sont les bienvenues : bugs, idées, nouvelles applications, documentation.
Lis [CONTRIBUTING.md](CONTRIBUTING.md) pour démarrer. Pour une faille de sécurité, voir [SECURITY.md](SECURITY.md).

## 📄 Licence

[MIT](LICENSE) © 2026 Azmo. Azmo OS n'est affilié ni à la fondation Raspberry Pi, ni à CasaOS,
ni aux éditeurs des applications proposées. Les icônes d'applications proviennent du projet
[dashboard-icons](https://github.com/homarr-labs/dashboard-icons).

<div align="center">
<br>
Fait avec ❤️, <b>Made by Azmo</b>
</div>
