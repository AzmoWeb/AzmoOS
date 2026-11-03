package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/azmoweb/azmoos/internal/jobs"
)

func TestCatalogIsValid(t *testing.T) {
	s, err := New(t.TempDir(), jobs.NewRunner())
	if err != nil {
		t.Fatal(err)
	}
	cat := s.Catalog()
	if len(cat) < 20 {
		t.Fatalf("catalogue trop petit : %d", len(cat))
	}
	ports := map[int]string{}
	for _, a := range cat {
		if a.ID == "" || a.Name == "" || a.Tagline == "" || a.Description == "" || a.Category == "" {
			t.Errorf("%s : champ obligatoire manquant", a.ID)
		}
		if !idRe.MatchString(a.ID) {
			t.Errorf("%s : identifiant invalide", a.ID)
		}
		if !strings.Contains(a.Compose, "services:") || strings.Contains(a.Compose, "{{") {
			t.Errorf("%s : compose invalide ou variable non remplacée", a.ID)
		}
		if en := a.I18n["en"]; en["tagline"] == "" || en["description"] == "" || (a.Notes != "" && en["notes"] == "") {
			t.Errorf("%s : traduction anglaise incomplète", a.ID)
		}
		if len(a.Arch) == 0 {
			t.Errorf("%s : architectures non déclarées", a.ID)
		}
		if a.Port > 0 {
			if other, dup := ports[a.Port]; dup {
				t.Errorf("port %d utilisé par %s et %s", a.Port, other, a.ID)
			}
			ports[a.Port] = a.ID
		}
		if a.Port == 80 {
			t.Errorf("%s : le port 80 est réservé à Azmo OS", a.ID)
		}
	}
}

func TestPrepareDirs(t *testing.T) {
	data := t.TempDir()
	s, _ := New(data, jobs.NewRunner())
	compose := s.Render("services:\n  x:\n    volumes:\n      - {{APPDATA}}/config:/config\n      - \"{{APPDATA}}/db:/db\"\n      - /etc/localtime:/etc/localtime:ro\n", "demo")
	s.prepareDirs("demo", compose)
	for _, d := range []string{"config", "db"} {
		if st, err := os.Stat(filepath.Join(data, "AppData", "demo", d)); err != nil || !st.IsDir() {
			t.Errorf("dossier %s non créé", d)
		}
	}
	if _, err := os.Stat(filepath.Join(data, "etc")); err == nil {
		t.Error("un chemin hors AppData a été créé")
	}
}
