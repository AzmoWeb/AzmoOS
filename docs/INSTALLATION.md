# Installer Azmo OS

Deux méthodes : **l'image prête à flasher** (la plus simple) ou **l'installation sur un Raspberry Pi OS existant**.

## Ce qu'il te faut

| | Minimum | Conseillé |
|---|---|---|
| Raspberry Pi | Pi 3 B / Zero 2 W | **Pi 5 (4 ou 8 Go)** ou Pi 4 (4 Go+) |
| Stockage | microSD 16 Go (classe A1) | microSD 32 Go A2, ou **SSD** (USB 3 ou NVMe sur Pi 5) |
| Alimentation | officielle 5 V / 3 A | **officielle 27 W** pour le Pi 5 |
| Réseau | Wi-Fi | **câble Ethernet** |
| Refroidissement | — | ventilateur actif (Pi 5) |

## Méthode 1 — Image Azmo OS (recommandée)

1. Télécharge `AzmoOS-….img.xz` depuis la page [Releases](https://github.com/azmoweb/azmoos/releases).
2. Ouvre [Raspberry Pi Imager](https://www.raspberrypi.com/software/) → **Choisir l'OS** → *Utiliser une image personnalisée* → sélectionne le fichier.
3. *(Optionnel)* Dans les réglages d'Imager, renseigne ton Wi-Fi si tu n'as pas de câble Ethernet.
4. Flashe la carte (ou le SSD), insère-la dans le Pi et allume-le.
5. Patiente 2 à 3 minutes au premier démarrage (agrandissement de la partition, premiers réglages).
6. Depuis un ordinateur ou un téléphone **sur le même réseau**, ouvre **http://azmoos.local**.
7. L'assistant te guide : création de ton compte, nom du Pi, Wi-Fi. C'est fini !

### Pas de réseau ? Le point d'accès de secours

Si ton Pi n'est ni branché en Ethernet ni configuré en Wi-Fi, il crée au bout de 90 secondes
son propre réseau Wi-Fi :

- Nom : **AzmoOS-Setup**
- Mot de passe : **azmoos-setup**

Connecte ton téléphone à ce réseau puis ouvre **http://10.42.0.1** : l'assistant te propose
de choisir ton Wi-Fi. Une fois connecté, le point d'accès disparaît.

### Accès SSH de l'image

Utilisateur `azmo`, mot de passe `azmoos`. **Change-le** dès la première connexion : `passwd`.
Tu peux aussi désactiver SSH depuis *Sécurité* dans l'interface.

## Méthode 2 — Sur un Raspberry Pi OS existant

Fonctionne sur Raspberry Pi OS Lite/Desktop (Bookworm ou plus récent), Debian 12+ et Ubuntu 22.04+.

```bash
curl -fsSL https://raw.githubusercontent.com/azmoweb/azmoos/main/scripts/install.sh | sudo bash
```

Le script :
1. détecte ton matériel (modèle, processeur, mémoire) ;
2. installe Docker, avahi (`.local`), Samba, ufw et les outils disques ;
3. télécharge le binaire Azmo OS **vérifié par SHA-256** et l'interface ;
4. installe les services `azmoos` et `azmoos-hotspot`, puis démarre tout.

Options :

```bash
# Version précise
curl -fsSL …/install.sh | sudo AZMO_VERSION=v0.2.0 bash
# Installation minimale (sans Samba, pare-feu ni point d'accès)
curl -fsSL …/install.sh | sudo AZMO_MINIMAL=1 bash
```

## Après l'installation

- Interface : `http://<nom-du-pi>.local` ou `http://<ip-du-pi>`
- État du service : `sudo systemctl status azmoos`
- Journaux en direct : `sudo journalctl -u azmoos -f`

## Mot de passe oublié

```bash
sudo systemctl stop azmoos
sudo python3 - <<'PY'
import json; p = "/etc/azmoos/config.json"; c = json.load(open(p))
c["users"] = []; c["sessions"] = {}; json.dump(c, open(p, "w"), indent=2)
PY
sudo systemctl start azmoos
```

L'assistant de premier démarrage réapparaît : recrée ton compte. Tes applications et fichiers ne sont pas touchés.

## Désinstaller

```bash
curl -fsSL https://raw.githubusercontent.com/azmoweb/azmoos/main/scripts/uninstall.sh | sudo bash
```

Docker, tes applications et `/DATA` sont conservés.

## Problèmes fréquents

| Symptôme | Solution |
|---|---|
| `azmoos.local` ne s'ouvre pas | Utilise l'adresse IP (visible dans l'interface de ta box). Sur Windows ancien, installe *Bonjour*. |
| « Le port 80 est déjà utilisé » | Un autre serveur web tourne (Apache, nginx, Pi-hole…). Modifie `-addr :8080` dans `/etc/systemd/system/azmoos.service`. |
| Une app ne s'installe pas | Ouvre sa fiche : Azmo OS explique l'erreur en clair et propose de réessayer. |
| Ton Pi chauffe / ralentit | Le tableau de bord le signale. Ajoute un ventilateur ou un dissipateur. |
| Docker Hub « 429 Too Many Requests » | Limite de téléchargements atteinte, réessaie plus tard. |
