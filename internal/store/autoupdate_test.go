package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azmoweb/azmoos/internal/jobs"
)

// fakeDocker remplace docker par un script qui simule une app « demo » :
// state/running = conteneur en marche, state/current = image utilisée,
// state/latest = image disponible après « pull », state/fail = pull en échec.
const fakeDocker = `#!/bin/sh
S="$FAKE_STATE"
echo "$*" >> "$S/calls"
case "$*" in
  *" ps -q") [ -f "$S/running" ] && echo c1; exit 0 ;;
  *" ps -aq") echo c1; exit 0 ;;
  "inspect --format {{.Image}} c1") cat "$S/current"; exit 0 ;;
  *" pull --quiet") if [ -f "$S/fail" ]; then echo "pull access denied for demo"; exit 1; fi; exit 0 ;;
  *" config --images") echo demo:latest; exit 0 ;;
  "image inspect --format {{.Id}} demo:latest") cat "$S/latest"; exit 0 ;;
  *" up -d --remove-orphans") cp "$S/latest" "$S/current"; exit 0 ;;
  "image prune -f") exit 0 ;;
esac
echo "commande inattendue : $*" >&2
exit 2
`

func setupFake(t *testing.T) (*Store, string) {
	t.Helper()
	bin, state := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(fakeDocker), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_STATE", state)
	s, err := New(t.TempDir(), jobs.NewRunner())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.appsDir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(s.composePath("demo"), []byte("services:\n  demo:\n    image: demo:latest\n"), 0o644)
	w := func(n, v string) { _ = os.WriteFile(filepath.Join(state, n), []byte(v), 0o644) }
	w("running", "")
	w("current", "sha256:aaa\n")
	w("latest", "sha256:aaa\n")
	return s, state
}

func calls(state string) string {
	b, _ := os.ReadFile(filepath.Join(state, "calls"))
	_ = os.Remove(filepath.Join(state, "calls"))
	return string(b)
}

func TestAutoUpdate(t *testing.T) {
	s, state := setupFake(t)

	// Déjà à jour : pas de redémarrage.
	res, err := s.AutoUpdate("demo")
	if err != nil || res != UpdUpToDate {
		t.Fatalf("à jour : %q %v", res, err)
	}
	if c := calls(state); strings.Contains(c, "up -d") {
		t.Fatalf("une app à jour ne doit pas être recréée :\n%s", c)
	}

	// Nouvelle image : conteneurs recréés.
	_ = os.WriteFile(filepath.Join(state, "latest"), []byte("sha256:bbb\n"), 0o644)
	res, err = s.AutoUpdate("demo")
	if err != nil || res != UpdUpdated {
		t.Fatalf("mise à jour : %q %v", res, err)
	}
	if c := calls(state); !strings.Contains(c, "up -d --remove-orphans") || !strings.Contains(c, "image prune -f") {
		t.Fatalf("étapes manquantes :\n%s", c)
	}

	// App arrêtée : on n'y touche pas (pas de pull, pas de démarrage).
	_ = os.Remove(filepath.Join(state, "running"))
	_ = os.WriteFile(filepath.Join(state, "latest"), []byte("sha256:ccc\n"), 0o644)
	res, err = s.AutoUpdate("demo")
	if err != nil || res != UpdSkipped {
		t.Fatalf("arrêtée : %q %v", res, err)
	}
	if c := calls(state); strings.Contains(c, "pull") || strings.Contains(c, "up -d") {
		t.Fatalf("une app arrêtée ne doit pas être touchée :\n%s", c)
	}

	// Échec du téléchargement : erreur lisible, l'app reste comme elle est.
	_ = os.WriteFile(filepath.Join(state, "running"), nil, 0o644)
	_ = os.WriteFile(filepath.Join(state, "fail"), nil, 0o644)
	_, err = s.AutoUpdate("demo")
	if err == nil || !strings.Contains(err.Error(), "pull access denied") {
		t.Fatalf("erreur attendue, obtenu %v", err)
	}
	if c := calls(state); strings.Contains(c, "up -d") {
		t.Fatalf("pas de redémarrage après un échec :\n%s", c)
	}

	if _, err := s.AutoUpdate("inconnue"); err == nil {
		t.Fatal("une app non installée doit être refusée")
	}
}

func TestFriendlyErrors(t *testing.T) {
	for in, want := range map[string]string{
		`failed to resolve reference "docker.io/library/nginx:alpine": 429 Too Many Requests`: "Docker Hub limite",
		"Cannot connect to the Docker daemon at unix:///var/run/docker.sock":                  "Docker ne répond pas",
		"write /var/lib/docker/x: no space left on device":                                    "plus assez de place",
		"pull access denied for demo":                                                         "pull access denied",
	} {
		if got := friendly(errors.New(in)).Error(); !strings.Contains(got, want) {
			t.Errorf("%q → %q", in, got)
		}
	}
}
