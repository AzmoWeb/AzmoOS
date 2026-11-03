# Journal des versions

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
