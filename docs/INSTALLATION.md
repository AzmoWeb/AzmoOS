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

1. Télécharge `AzmoOS-….img.xz` depuis la page [Releases](https://github.com/AzmoWeb/AzmoOS/releases).
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

## Méthode 2 — Sur un Raspberry Pi OS existant (en SSH)

Fonctionne sur Raspberry Pi OS Lite/Desktop (Bookworm ou plus récent), Debian 12+ et Ubuntu 22.04+.

**1. Connecte-toi à ton Pi en SSH** depuis ton ordinateur (Terminal sur Mac/Linux, PowerShell sur Windows) :

```bash
ssh pi@raspberrypi.local        # remplace « pi » par l'utilisateur choisi dans Raspberry Pi Imager
```

**2. Lance l'installation** :

```bash
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo bash
```

**3. Réponds aux questions** :

```
● Création de ton compte
  Ces identifiants te serviront à te connecter à l'interface web.

  Comment tu t'appelles ? [Azmo] : Azmo
  Nom d'utilisateur [azmo] :
  Mot de passe (8 caractères minimum) : ••••••••••
  Confirme le mot de passe : ••••••••••
  Nom de ton Pi sur le réseau [raspberrypi] : azmoos
  ✔ Compte administrateur « azmo » créé
```

**4. Ouvre l'adresse affichée** à la fin, par exemple `http://192.168.1.42`, et connecte-toi. C'est tout !

Le script :
1. détecte ton matériel (modèle, processeur, mémoire, place libre) ;
2. installe Docker, avahi (`.local`), Samba, ufw, rclone et les outils disques ;
3. télécharge le binaire Azmo OS **vérifié par SHA-256** et l'interface ;
4. te demande ton nom, ton nom d'utilisateur, ton mot de passe et le nom du Pi ;
5. installe et démarre les services `azmoos` et `azmoos-hotspot`.

Relancer le script plus tard **met à jour** Azmo OS sans toucher à tes comptes ni à tes apps.

### Options

```bash
# Une version précise
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo AZMO_VERSION=v0.2.0 bash

# Sans questions (automatisation, Ansible…)
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh \
  | sudo AZMO_USER=azmo AZMO_PASSWORD='MonMotDePasse' AZMO_NAME="Azmo" AZMO_HOSTNAME=azmoos bash

# Installation minimale (sans Samba, pare-feu ni point d'accès Wi-Fi)
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/install.sh | sudo AZMO_MINIMAL=1 bash
```

## Après l'installation

- Interface : `http://<nom-du-pi>.local` ou `http://<ip-du-pi>`
- État du service : `sudo systemctl status azmoos`
- Journaux en direct : `sudo journalctl -u azmoos -f`

## Gérer les comptes en ligne de commande

Depuis une session SSH sur le Pi :

```bash
sudo azmod user list                     # liste les comptes
sudo azmod user add camille --member     # ajoute un membre (le mot de passe est demandé)
sudo azmod user add azmo2                # ajoute un administrateur
sudo azmod user passwd azmo              # change un mot de passe
sudo azmod user delete camille           # supprime un compte
sudo azmod user 2fa-off azmo             # désactive la double authentification (téléphone perdu)
```

Les changements sont pris en compte immédiatement, sans redémarrer le service.

## Mot de passe oublié

```bash
sudo azmod user passwd azmo
```

## Téléphone perdu (double authentification)

Utilise un de tes codes de secours, ou en SSH :

```bash
sudo azmod user 2fa-off azmo
```

## Désinstaller

```bash
curl -fsSL https://raw.githubusercontent.com/AzmoWeb/AzmoOS/main/scripts/uninstall.sh | sudo bash
```

Docker, tes applications et `/DATA` sont conservés.

## Problèmes fréquents

| Symptôme | Solution |
|---|---|
| `azmoos.local` ne s'ouvre pas | Utilise l'adresse IP (visible dans l'interface de ta box). Sur Windows ancien, installe *Bonjour*. |
| « Le port 80 est déjà utilisé » | Un autre serveur web tourne (Apache, nginx, Pi-hole…). Modifie `-addr :8080` dans `/etc/systemd/system/azmoos.service`. |
| Une app ne s'installe pas | Ouvre sa fiche : Azmo OS explique l'erreur en clair et propose de réessayer. |
| Ton Pi chauffe / ralentit | Le bureau le signale (et une notification te prévient). Ajoute un ventilateur ou un dissipateur. |
| Docker Hub « 429 Too Many Requests » | Limite de téléchargements atteinte, réessaie plus tard. |
