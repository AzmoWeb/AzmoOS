# Guide d'utilisation

## Le bureau

En te connectant, tu arrives sur ton **bureau** :

- **à gauche**, les widgets : l'heure, l'état de ton Pi en une phrase (*« Tout roule ✨ »*, *« Il commence à faire chaud »*…),
  les jauges du processeur et de la mémoire, la température, le stockage et le réseau ;
- **à droite**, tes applications sous forme de grandes icônes : un clic ouvre l'application, le bouton **⋮** (au survol)
  propose arrêter, redémarrer, mettre à jour, journaux… La pastille verte ou rouge indique si elle tourne ;
- **en haut**, la barre de recherche (raccourci `/` ou `Ctrl+K`) trouve une app, une section ou une app de l'App Store ;
  le bouton ▦ ouvre le **lanceur** avec toutes les sections ; la cloche 🔔 affiche les notifications ; ton avatar
  donne accès aux réglages, au fond d'écran, au thème et à la déconnexion.

Chaque section (Fichiers, App Store, Sécurité…) s'ouvre comme une **fenêtre** par-dessus le bureau : « ‹ Bureau » ou ✕ pour revenir.

**Changer le fond d'écran** : avatar → *Fond d'écran*, puis choisis un des 8 fonds Azmo ou envoie ta propre photo
(JPEG, PNG ou WebP, 12 Mo maximum). Ton choix suit ton compte sur tous tes appareils.

## Installer une application

1. Va dans **App Store**.
2. Choisis une catégorie ou cherche (« films », « photos », « VPN »…).
3. Clique sur une app : description, mémoire conseillée, port, conseils.
4. **Installer**. Le téléchargement peut prendre quelques minutes ; une notification te prévient.
5. Clique ensuite sur la carte de l'app dans **Mes applications** pour l'ouvrir.

> 💡 *Configuration avancée* permet de modifier ports, dossiers ou mots de passe avant l'installation.

### Gérer une application

Le menu **⋮** d'une carte : Ouvrir, Détails et ressources (CPU, mémoire, ports, volumes),
Arrêter/Démarrer, Redémarrer, Mettre à jour, Désinstaller.
L'onglet **Journaux** affiche les messages de l'app ; **Configuration** permet d'éditer son
`docker-compose.yml`.

La désinstallation **conserve les données** dans `/DATA/AppData/<app>` : réinstaller l'app la
retrouve telle quelle.

### Application personnalisée

*Mes applications → App personnalisée* : colle n'importe quel `docker-compose.yml`.

## Fichiers

Explorateur de `/DATA` : envoi par glisser-déposer (avec barre de progression), téléchargement,
aperçu, renommage, déplacement, suppression. Raccourcis vers Films, Séries, Musique, Photos, Documents…

## Disques & partages (NAS)

### Utiliser un disque USB
1. Branche le disque : il apparaît dans **Disques** en quelques secondes.
2. **Utiliser ce disque** : il est monté dans `/DATA/Disks/<nom>`.
3. Active **Monter automatiquement au démarrage** pour le retrouver après un redémarrage.
4. **Éjecter** avant de le débrancher.

Un disque neuf peut être **formaté** (ext4 conseillé, exFAT pour le relire sur Windows/Mac).
Le disque système ne peut jamais être formaté ni éjecté.

### Partager des dossiers sur le réseau
1. Onglet **Partages réseau** → *Comptes réseau* → **Activer l'accès** pour ton compte et choisis un mot de passe réseau.
2. **Partager un dossier** : nom, dossier, lecture seule ou non, accès invité ou réservé.
3. Sur ton ordinateur :
   - Windows : explorateur → `\\azmoos`
   - Mac : Finder → *Aller* → *Se connecter au serveur* → `smb://azmoos.local`
   - Android/iOS : app Fichiers → serveur SMB

## Réseau

État de l'Ethernet et du Wi-Fi, connexion à un nouveau réseau, **IP fixe** ou automatique.

## Sécurité

- **Pare-feu** : un interrupteur. L'interface et SSH restent toujours autorisés à l'activation.
- **SSH** : à désactiver si tu ne t'en sers pas.
- **Accès à distance** : utilise Tailscale ou WireGuard (App Store) plutôt que d'ouvrir des ports sur ta box.
- **Sessions** : la liste des appareils connectés à ton compte, avec déconnexion à distance.

## Utilisateurs

Ajoute ta famille ou tes colocs. Un **administrateur** a tous les droits ; un **membre** voit le
bureau et ses apps, et peut recevoir ces permissions :

| Permission | Effet |
|---|---|
| Gérer les applications | Installer, démarrer, arrêter, configurer les apps |
| Voir les fichiers | Parcourir et télécharger |
| Modifier les fichiers | Envoyer, renommer, supprimer |

## Terminal web

*Système → Terminal* (administrateurs) : la ligne de commande de ton Pi dans le navigateur.
Tu es connecté avec l'utilisateur principal du Pi (celui créé dans Raspberry Pi Imager) : utilise `sudo`
pour les commandes d'administration. Sur téléphone, une barre propose Échap, Tab, Ctrl+C et les flèches.
Le terminal peut être désactivé dans *Sécurité* ; chaque ouverture est enregistrée dans le *Journal*.

## Double authentification

*Sécurité → Double authentification → Activer* :
1. installe une app d'authentification (Google Authenticator, Authy, Microsoft Authenticator, 1Password, Bitwarden) ;
2. scanne le QR code, saisis le code à 6 chiffres ;
3. **mets les 8 codes de secours à l'abri** (téléchargement ou papier).

À chaque connexion, le code est demandé après le mot de passe. Téléphone perdu ? Utilise un code de secours,
demande à un administrateur de réinitialiser ta double authentification (*Utilisateurs*), ou en SSH :
`sudo azmod user 2fa-off <nom>`.

## HTTPS local

*Sécurité → HTTPS local → Activer le HTTPS* : l'interface est aussi disponible en `https://`.
Pour ne plus voir l'avertissement du navigateur, **télécharge le certificat** et installe-le une fois sur chaque
appareil (le guide pas à pas pour iPhone, Android, Windows, Mac et Linux est sur la page). Ensuite, active
*Toujours utiliser HTTPS*. Le certificat de l'autorité ne peut servir que pour des adresses locales.

## Notifications

La cloche 🔔 affiche les alertes : température, alimentation, stockage, mémoire, app arrêtée ou qui plante,
mise à jour disponible, tentatives d'intrusion, sauvegardes. *Système → Notifications* permet de choisir ce qui est
surveillé, les seuils, et de les recevoir **sur ton téléphone** :

| Service | Mise en place |
|---|---|
| **ntfy** (le plus simple) | Installe l'app ntfy, abonne-toi à un sujet secret, saisis-le dans Azmo OS |
| **Discord** | Paramètres du salon → Intégrations → Webhooks → copie l'URL |
| **Telegram** | Crée un bot avec @BotFather, récupère son jeton et ton identifiant de discussion |

Le bouton *Envoyer un test* vérifie que tout fonctionne. Une alerte n'est envoyée qu'une fois tant que le problème dure.

## Sauvegardes

### Sauvegardes automatiques

*Sauvegardes → Planifier* : choisis la fréquence (chaque jour ou chaque semaine), l'heure, la destination et
le nombre de sauvegardes à garder.

- **Disque ou dossier** : par exemple `/Disks/MonDisque/Sauvegardes` (branche le disque et clique *Utiliser ce disque*
  dans *Disques & partages*, puis active *Monter au démarrage*).
- **Cloud** : Google Drive, Dropbox, OneDrive, S3, et des dizaines d'autres grâce à rclone. Ouvre le *Terminal* et lance
  `sudo rclone config --config /etc/azmoos/rclone.conf` pour connecter ton compte, puis choisis-le dans Azmo OS.

Si le Pi était éteint à l'heure prévue, la sauvegarde est faite au démarrage suivant. Le bouton 🕘 liste les sauvegardes
présentes dans la destination et permet d'en restaurer une.

### Sauvegarde manuelle

**Télécharger la sauvegarde** crée un fichier `.tar.gz` avec comptes, réglages, partages et la
configuration de chaque app (option : données internes des apps). **Restaurer** remet tout en
place et redéploie les apps. Garde une copie hors du Pi !

## Mises à jour

- **Azmo OS** : vérifié automatiquement toutes les 12 h ; installation en un clic, fichier vérifié par SHA-256.
- **Système** : mises à jour Raspberry Pi OS / Debian, avec indication quand un redémarrage est nécessaire.

## Langue

*Réglages → Langue* : français ou English. Le choix est enregistré dans ton compte ; au premier accès, la langue
du navigateur est utilisée.

## Journal

Tout ce qui se passe est tracé : connexions (réussies ou non), installations, changements de
réglages, disques, réseau. Filtrable par catégorie.
