package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStaysInRoot(t *testing.T) {
	root := t.TempDir()
	fs := New(root)
	for _, p := range []string{"../../etc/passwd", "/../..", "a/../../b", "/"} {
		got, err := fs.Resolve(p)
		if err != nil {
			continue
		}
		if got != root && filepath.Dir(got) != root && len(got) < len(root) {
			t.Fatalf("%q résolu hors racine : %s", p, got)
		}
	}
	// lien symbolique sortant
	_ = os.Symlink("/etc", filepath.Join(root, "evil"))
	if _, err := fs.Resolve("/evil"); err == nil {
		t.Fatal("le lien symbolique vers /etc devrait être refusé")
	}
	if err := fs.Delete("/"); err == nil {
		t.Fatal("la suppression de la racine devrait être refusée")
	}
	if err := fs.Mkdir("/", "../x"); err == nil {
		t.Fatal("nom invalide accepté")
	}
}
