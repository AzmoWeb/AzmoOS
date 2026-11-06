# API REST

Toutes les routes sont sous `/api`, échangent du JSON et utilisent le cookie de session `azmo_session`.
Toute requête autre que `GET` doit porter l'en-tête `X-Requested-With: AzmoOS`.
Les erreurs renvoient `{"error": "message lisible"}` avec un code HTTP adapté
(400, 401 non connecté, 403 interdit, 409 conflit, 429 trop de tentatives).

## Publiques

| Méthode | Route | Description |
|---|---|---|
| GET | `/api/status` | Version, nom d'hôte, état d'initialisation, utilisateur connecté, matériel |
| POST | `/api/setup` | `{username, display, password}`, création du premier compte (une seule fois) |
| POST | `/api/login` | `{username, password, code?}`. Avec la double authentification, une première réponse `{ok:false, totp:true}` demande le code (TOTP ou code de secours) |
| GET | `/api/https/ca.crt` | Certificat de l'autorité locale, à installer sur les appareils |
| POST | `/api/logout` | |

## Compte connecté

| Méthode | Route | Description |
|---|---|---|
| GET | `/api/me` | Profil + sessions |
| POST | `/api/me/profile` | `{display}` |
| POST | `/api/me/password` | `{current, password}` |
| POST | `/api/me/sessions/revoke` | Ferme les autres sessions |
| POST | `/api/me/lang` | `{lang: "fr" \| "en"}` |
| POST | `/api/me/2fa/begin` | Démarre l'activation : `{secret, uri}` (lien otpauth:// pour le QR code) |
| POST | `/api/me/2fa/confirm` | `{code}` → `{recovery: [8 codes]}` |
| POST | `/api/me/2fa/disable` | `{password, code}` |
| GET | `/api/system` | Statistiques temps réel |
| GET | `/api/system/history` | Historique (1 point / 10 s, 1 h) |
| GET | `/api/hardware` | Matériel détecté et profil |
| GET | `/api/jobs` | Tâches en cours et récentes |
| GET | `/api/apps` | Apps installées et leur état |
| GET | `/api/store/catalog` | Catalogue avec compatibilité |
| GET | `/api/docker/stats` | CPU/RAM par conteneur |

## `apps.manage`

| Méthode | Route | Description |
|---|---|---|
| POST | `/api/apps/{id}/install` | `{compose?, name?, icon?, port?}`. Sans compose : modèle du catalogue |
| POST | `/api/apps/{id}/{start,stop,restart,update,uninstall}` | |
| GET | `/api/apps/{id}/compose` | Fichier compose actuel |
| GET | `/api/docker/containers` · `/{id}` · `/{id}/logs?tail=` | |
| POST | `/api/docker/containers/{id}/{start,stop,restart,remove}` | |
| GET / DELETE | `/api/docker/images` · `/{id}` | |
| POST | `/api/docker/prune` | Supprime les images inutilisées |
| GET / DELETE | `/api/docker/volumes` · `/{name}` | |

## `files.read` / `files.write`

| Méthode | Route | Permission |
|---|---|---|
| GET | `/api/files?path=` | read |
| GET | `/api/files/download?path=&inline=1` | read |
| POST | `/api/files/upload?path=` (multipart `files`) | write |
| POST | `/api/files/{mkdir,rename,move,delete}` `{path, name?, to?}` | write |

## Administrateur

| Domaine | Routes |
|---|---|
| Système | `POST /api/system/{power,hostname,timezone,ssh}`, `GET /api/system/settings` |
| Mises à jour | `GET /api/updates`, `POST /api/updates/{check,system,azmoos}` |
| Utilisateurs | `GET/POST /api/users`, `PUT/DELETE /api/users/{name}`, `POST /api/users/{name}/password` |
| Disques | `GET /api/storage/disks`, `POST /api/storage/{mount,unmount,automount,format}` |
| Partages | `GET/POST /api/shares`, `DELETE /api/shares/{name}`, `POST /api/shares/users`, `DELETE /api/shares/users/{name}` |
| Réseau | `GET /api/network`, `GET /api/network/wifi`, `POST /api/network/wifi/{connect,forget}`, `POST /api/network/ip` |
| Pare-feu | `GET /api/firewall`, `POST /api/firewall/{enable,disable}`, `POST /api/firewall/rules`, `DELETE /api/firewall/rules/{n}` |
| Sauvegardes | `GET /api/backup?appdata=1`, `POST /api/backup/restore` (multipart) |
| Journal | `GET /api/events?category=&limit=` |
| Double authentification | `DELETE /api/users/{name}/2fa` (réinitialisation) |
| Terminal | `GET /api/terminal/info`, `POST /api/terminal/settings` `{enabled}`, `GET /api/terminal` (WebSocket : le client envoie `0`+saisie ou `1`+`{"cols","rows"}`, le serveur renvoie la sortie en binaire) |
| HTTPS | `GET /api/https`, `POST /api/https` `{enabled, redirect}`, `POST /api/https/regenerate` |
| Notifications | `GET /api/notifications`, `POST /api/notifications/read` `{id}` (0 = toutes), `POST /api/notifications/clear`, `GET/POST /api/notifications/settings`, `POST /api/notifications/test` |
| Sauvegardes planifiées | `GET/POST /api/backups/schedules`, `DELETE /api/backups/schedules/{id}`, `POST …/{id}/run`, `GET …/{id}/files`, `POST …/{id}/restore` `{file}` |
