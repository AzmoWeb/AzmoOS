# Sécurité d'Azmo OS

## Modèle

Azmo OS est conçu pour un **réseau domestique**. L'interface écoute en HTTP sur le port 80 du réseau
local, et en HTTPS sur le port 443 si tu actives le HTTPS local (conseillé). Pour un accès depuis internet, utilise un VPN (Tailscale ou WireGuard dans l'App Store), ou
un reverse proxy HTTPS (Nginx Proxy Manager). **N'expose jamais directement le port 80 sur internet.**

## Mesures en place

| Domaine | Mesure |
|---|---|
| Mots de passe | PBKDF2-SHA256, 210 000 itérations, sel aléatoire de 128 bits, 8 caractères minimum |
| Sessions | Jeton de 256 bits, cookie `HttpOnly` + `SameSite=Strict` (+ `Secure` en HTTPS), 30 jours |
| Changement de mot de passe | Ferme toutes les autres sessions |
| CSRF | En-tête `X-Requested-With: AzmoOS` obligatoire pour toute modification |
| Force brute | Blocage de l'IP 5 minutes après 8 échecs ; temps de vérification constant ; alerte de sécurité |
| Double authentification | TOTP (RFC 6238) : fenêtre de ±30 s, **un code ne peut servir qu'une fois**, 8 codes de secours stockés sous forme d'empreinte SHA-256 et consommés à l'usage |
| HTTPS local | Autorité de certification ECDSA P-256 propre au Pi avec **contrainte de nom** (`.local`, `home.arpa`, nom du Pi, réseaux privés uniquement) : même volée, elle ne peut pas usurper un site public. Certificat serveur renouvelé automatiquement (397 jours max, changement d'IP). TLS 1.2 minimum |
| Terminal web | Réservé aux administrateurs, désactivable, **vérification de l'origine** du WebSocket (anti cross-site WebSocket hijacking), 5 sessions max, shell lancé avec l'utilisateur système principal (pas root si possible), chaque session journalisée |
| Permissions | Vérifiées côté serveur sur chaque route (`admin`, `apps.manage`, `files.read`, `files.write`) |
| Fichiers | Confinés à `/DATA`, liens symboliques sortants refusés, téléchargements servis avec `CSP: sandbox` |
| Disques | Disque système protégé ; formatage soumis à une double confirmation |
| Pare-feu | Active toujours d'abord l'interface et SSH pour ne jamais te bloquer dehors |
| Mises à jour | Binaire vérifié par SHA-256 et testé avant remplacement ; ancienne version conservée (`azmod.old`) |
| Sauvegardes | Restauration refusant les chemins absolus, `..` et liens symboliques externes |
| Secrets | Les variables contenant PASS/SECRET/TOKEN/KEY sont masquées dans l'interface |
| En-têtes HTTP | `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` |
| Journal | Connexions, échecs, blocages, changements de droits et de réglages |

## Limites connues (feuille de route)

- `azmod` tourne en root (nécessaire pour disques, réseau, Docker). Le service systemd limite
  `/home` en lecture seule et isole `/tmp`.
- Docker contourne ufw pour les ports publiés par les conteneurs : le pare-feu protège le Pi,
  mais les ports des apps restent accessibles sur le réseau local.

## Signaler une faille

Ouvre une *Security Advisory* privée sur le dépôt GitHub plutôt qu'une issue publique.
