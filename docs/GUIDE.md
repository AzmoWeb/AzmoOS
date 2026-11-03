# Guide d'utilisation

## Le tableau de bord

La page d'accueil te dit en une phrase comment va ton Pi : *« Tout roule »*, *« Il commence à
faire chaud »*, *« Le stockage est presque plein »*… En dessous : processeur, mémoire,
température, réseau et stockage en temps réel, puis tes applications.

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
tableau de bord et ses apps, et peut recevoir ces permissions :

| Permission | Effet |
|---|---|
| Gérer les applications | Installer, démarrer, arrêter, configurer les apps |
| Voir les fichiers | Parcourir et télécharger |
| Modifier les fichiers | Envoyer, renommer, supprimer |

## Sauvegardes

**Télécharger la sauvegarde** crée un fichier `.tar.gz` avec comptes, réglages, partages et la
configuration de chaque app (option : données internes des apps). **Restaurer** remet tout en
place et redéploie les apps. Garde une copie hors du Pi !

## Mises à jour

- **Azmo OS** : vérifié automatiquement toutes les 12 h ; installation en un clic, fichier vérifié par SHA-256.
- **Système** : mises à jour Raspberry Pi OS / Debian, avec indication quand un redémarrage est nécessaire.

## Journal

Tout ce qui se passe est tracé : connexions (réussies ou non), installations, changements de
réglages, disques, réseau. Filtrable par catégorie.
