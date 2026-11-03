# Développer Azmo OS

## Prérequis

- Go 1.24+
- Node.js 22+
- Docker (pour tester les applications)

## Démarrage rapide

```bash
make run     # compile l'interface + le backend, lance sur http://localhost:8080 (données dans .dev/)
make dev     # dans un 2e terminal : interface avec rechargement à chaud sur http://localhost:5173
make test    # go vet + tests Go + vérification TypeScript
```

`make dev` relaie `/api` vers `localhost:8080`.

## Organisation du code

```
cmd/azmod/              point d'entrée : lecture des options, assemblage des modules
internal/
  auth/                 comptes, rôles, permissions, sessions, anti force brute
  backup/               sauvegarde / restauration .tar.gz
  cmdutil/              exécution de commandes système avec délai
  config/               config.json (écriture atomique, migration v0.1 → v0.2)
  docker/               client Docker minimal (socket Unix, sans SDK)
  events/               journal JSONL avec rotation
  files/                explorateur confiné à /DATA
  firewall/             ufw
  jobs/                 tâches de fond avec journal (installations, formatage, apt…)
  network/              NetworkManager (nmcli) : interfaces, Wi-Fi, IP fixe
  server/               API HTTP : une route = une permission (routes_*.go)
  shares/               Samba
  storage/              disques : lsblk, mount, mkfs, fstab
  store/                App Store : catalogue apps/*.json + Docker Compose
  system/               stats, détection matériel, services, apt
  updater/              mise à jour depuis GitHub Releases
frontend/               interface React 19 + TypeScript + Vite
  src/api/              client HTTP et types partagés avec le backend
  src/components/       design system (ui.tsx), modales/menus/notifications, layout
  src/lib/              formatage et messages « humains »
  src/pages/            une page par section
web/                    embarque frontend compilé (web/dist) dans le binaire
scripts/                install.sh, uninstall.sh, services systemd, point d'accès
image/                  stage pi-gen pour l'image .img.xz
docs/                   documentation
```

## Principes

- **Aucune dépendance Go externe** : bibliothèque standard uniquement. Plus simple à auditer et à compiler pour tous les Pi.
- **Le backend ne fait confiance à rien** : chaque route vérifie sa permission, chaque chemin et chaque argument de commande est validé.
- **Pas de shell** : les commandes sont lancées avec des arguments séparés (`exec.Command`), jamais via `sh -c`.
- **Interface humaine** : on tutoie, on explique, on rassure. Les erreurs techniques sont traduites (`lib/human.ts`).

## Ajouter une application au catalogue

Crée `internal/store/apps/<id>.json` :

```json
{
  "id": "monapp",
  "name": "Mon App",
  "tagline": "Une phrase courte",
  "description": "Ce que fait l'app, pour qui.",
  "category": "Outils",
  "icon": "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/monapp.png",
  "port": 8123,
  "arch": ["arm64", "amd64"],
  "minRam": 512,
  "notes": "Identifiants par défaut, conseils…",
  "compose": "services:\n  monapp:\n    image: …\n    restart: unless-stopped\n    ports:\n      - \"8123:80\"\n    volumes:\n      - {{APPDATA}}/data:/data\n"
}
```

Variables : `{{APPDATA}}` (= `/DATA/AppData/<id>`), `{{DATA}}`, `{{TZ}}`, `{{PUID}}`, `{{PGID}}`.
`go test ./internal/store` vérifie automatiquement le catalogue (champs, ports en double, port 80 réservé…).

## Ajouter une route API

1. Écris la logique dans le module concerné (`internal/<module>`).
2. Déclare la route dans `internal/server/routes_<domaine>.go` avec sa permission :
   `handle("POST /api/truc", auth.PermAdmin, func(w, r) { … })`.
3. Journalise les actions importantes : `s.log(r, events.Info, "system", "…")`.
4. Ajoute le type dans `frontend/src/api/types.ts`.

## Publier une version

```bash
git tag v0.2.0
git push origin v0.2.0
```

GitHub Actions : tests → binaires (arm64, armv6, amd64) → archive de l'interface → `SHA256SUMS` →
Release → image Raspberry Pi `.img.xz` (≈ 45 min).

## Construire l'image en local

```bash
make image   # nécessite Docker et ~20 Go
```
