# Contribuer à Azmo OS

Merci de vouloir aider ! 🙌 Toute contribution est la bienvenue : correction de bug, nouvelle
application dans le catalogue, traduction, documentation, idée…

## Signaler un problème

Ouvre une [issue](https://github.com/AzmoWeb/AzmoOS/issues/new/choose) en précisant :
- ton modèle de Raspberry Pi et sa mémoire ;
- la version d'Azmo OS (*Réglages → À propos*) ;
- ce que tu as fait, ce que tu attendais, ce qui s'est passé ;
- si possible, les journaux : `sudo journalctl -u azmoos -n 100`.

## Proposer une modification

1. Fork le dépôt et crée une branche : `git checkout -b ma-fonctionnalite`
2. Installe l'environnement : Go 1.24+, Node 22+ (voir [docs/DEVELOPPEMENT.md](docs/DEVELOPPEMENT.md))
3. Développe : `make run` (serveur) et `make dev` (interface en direct)
4. Vérifie : `make test` doit passer
5. Ouvre une pull request en expliquant le *pourquoi* de ta modification

## Ajouter une application au catalogue

C'est la contribution la plus simple : un seul fichier JSON dans `internal/store/apps/`.
Le format est décrit dans [docs/DEVELOPPEMENT.md](docs/DEVELOPPEMENT.md#ajouter-une-application-au-catalogue).
Vérifie que l'image Docker existe pour **arm64**, choisis un port libre, puis lance
`go test ./internal/store` qui contrôle le catalogue automatiquement.

## Style

- **Go** : `gofmt`, bibliothèque standard uniquement, commentaires en français.
- **TypeScript/React** : composants fonctionnels, types stricts, réutiliser `components/ui.tsx`.
- **Textes de l'interface** : en français, on tutoie, on explique simplement, on rassure.
  Pas de jargon sans explication.
- **Sécurité** : jamais de `sh -c` avec une donnée utilisateur ; chaque route API déclare sa permission.

## Code de conduite

Sois bienveillant·e et respectueux·se. On est là pour construire quelque chose de chouette ensemble.
