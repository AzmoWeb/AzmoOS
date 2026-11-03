# Journal des versions

## v0.3.0 — « Sérénité »

- **Nouvelle interface « bureau »** : fond d'écran (8 fonds Azmo ou ta propre photo), barre du haut en verre dépoli
  avec recherche instantanée (`/` ou Ctrl+K), lanceur de sections, widgets (horloge, santé, jauges, stockage, réseau),
  grandes icônes d'applications, et chaque section qui s'ouvre comme une fenêtre par-dessus le bureau.

- **Terminal web** : la ligne de commande dans le navigateur (PTY + WebSocket implémentés sans dépendance),
  barre de touches pour mobile, réservé aux administrateurs, désactivable, chaque session journalisée.
- **Double authentification (TOTP)** : QR code, 8 codes de secours à usage unique, protection contre la réutilisation
  d'un code, réinitialisation par un administrateur ou `azmod user 2fa-off`.
- **HTTPS local** : autorité de certification propre au Pi, **limitée aux noms et réseaux locaux** (contrainte de nom),
  certificat renouvelé automatiquement quand l'IP change, redirection HTTP → HTTPS, guide d'installation par système.
- **Notifications** : surveillance de la température, de l'alimentation, du stockage, de la mémoire, des apps qui plantent,
  des mises à jour, des tentatives d'intrusion et des sauvegardes ; envoi sur téléphone via ntfy, Discord ou Telegram ;
  pas de répétition tant que le problème dure.
- **Sauvegardes planifiées** : quotidiennes ou hebdomadaires, vers un dossier (disque USB, partage) ou le cloud avec rclone,
  rattrapage si le Pi était éteint, conservation des N dernières, restauration depuis la liste.
- **Interface en anglais** : ~800 textes et 114 messages serveur traduits, catalogue d'apps bilingue, détection automatique,
  langue enregistrée par compte ; `npm run i18n:check` (vérifié en CI) empêche d'oublier une traduction.
- Corrigé : la pastille de la cloche bloquait le clic.

## v0.2.0 — « Humain »

- **Nouveau logo « le A-serveur »** et identité visuelle complète (`branding/`) : logotypes, monochromes, favicon, image de partage, fond d'écran, icônes pour l'écran d'accueil des téléphones.

- **Installation en une commande par SSH** qui pose les questions (nom, utilisateur, mot de passe, nom du Pi) : on se connecte directement sur l'IP locale.
- Commandes `azmod user list|add|passwd|delete`, prises en compte à chaud par le service.

- Nouvelle interface **React + TypeScript**, design chaleureux, thème sombre/clair, mobile.
- **Assistant de premier démarrage** (compte, nom du Pi, fuseau, Wi-Fi) et point d'accès Wi-Fi de secours.
- **Multi-utilisateurs** avec rôles et permissions, sessions par appareil.
- **Disques & clés USB** : détection, montage, éjection, formatage, montage au démarrage.
- **NAS** : partages Samba avec comptes réseau.
- **Réseau** : Wi-Fi, IP fixe / DHCP.
- **Sécurité** : pare-feu ufw, SSH, anti force brute.
- **Sauvegarde / restauration** de la configuration et des apps.
- **Mises à jour** d'Azmo OS (vérifiées SHA-256) et du système.
- **Journal** des événements.
- **Adaptation au matériel** : profils léger / standard / performance, compatibilité des apps.
- Conteneurs : ressources, volumes, images, ménage.
- 25 applications au catalogue (ajout de Tailscale, Immich, Paperless-ngx ; retrait de File Browser, archivé par ses auteurs).
- Icônes du catalogue : passage au dépôt maintenu homarr-labs/dashboard-icons.
- Correctif : les dossiers des apps sont créés pour l'utilisateur 1000 (plantages de certaines apps).

## v0.1.0

- Première version : tableau de bord, App Store, fichiers, conteneurs.
