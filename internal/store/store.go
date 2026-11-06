// Package store gère l'App Store AzmoOS (applications Docker Compose).
package store

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/azmoweb/azmoos/internal/jobs"
)

//go:embed apps/*.json
var catalogFS embed.FS

// ProjectPrefix préfixe les projets compose installés par AzmoOS.
const ProjectPrefix = "azmo-"

type App struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Port        int      `json:"port"`           // port de l'interface web
	Path        string   `json:"path,omitempty"` // chemin à ouvrir (ex: /admin)
	Scheme      string   `json:"scheme,omitempty"`
	Arch        []string `json:"arch"`
	Compose     string   `json:"compose"`
	Notes       string   `json:"notes,omitempty"`
	MinRAM      int      `json:"minRam,omitempty"` // Mo conseillés
	Website     string   `json:"website,omitempty"`
	Installed   bool     `json:"installed"`
	Custom      bool     `json:"custom,omitempty"`
	// Traductions des textes (tagline, description, notes) par langue.
	I18n map[string]map[string]string `json:"i18n,omitempty"`
}

type Store struct {
	dataDir string // /DATA
	appsDir string // /DATA/AppData/.azmoos/apps
	tz      string
	catalog map[string]App
	jobs    *jobs.Runner
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,40}$`)

func New(dataDir string, runner *jobs.Runner) (*Store, error) {
	s := &Store{
		dataDir: dataDir,
		appsDir: filepath.Join(dataDir, "AppData", ".azmoos", "apps"),
		catalog: map[string]App{},
		jobs:    runner,
		tz:      detectTZ(),
	}
	entries, err := catalogFS.ReadDir("apps")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		b, _ := catalogFS.ReadFile("apps/" + e.Name())
		var a App
		if err := json.Unmarshal(b, &a); err != nil {
			return nil, fmt.Errorf("catalogue %s: %w", e.Name(), err)
		}
		s.catalog[a.ID] = a
	}
	return s, os.MkdirAll(s.appsDir, 0o755)
}

func detectTZ() string {
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		if tz := strings.TrimSpace(string(b)); tz != "" {
			return tz
		}
	}
	if l, err := os.Readlink("/etc/localtime"); err == nil {
		if i := strings.Index(l, "zoneinfo/"); i >= 0 {
			return l[i+9:]
		}
	}
	return "Europe/Paris"
}

func (s *Store) composePath(id string) string {
	return filepath.Join(s.appsDir, id, "docker-compose.yml")
}

func (s *Store) isInstalled(id string) bool {
	_, err := os.Stat(s.composePath(id))
	return err == nil
}

// Catalog renvoie le catalogue complet, trié.
func (s *Store) Catalog() []App {
	out := make([]App, 0, len(s.catalog))
	for _, a := range s.catalog {
		a.Installed = s.isInstalled(a.ID)
		a.Compose = s.Render(a.Compose, a.ID)
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// Installed renvoie les apps installées (catalogue + perso).
func (s *Store) Installed() []App {
	var out []App
	entries, _ := os.ReadDir(s.appsDir)
	for _, e := range entries {
		if !e.IsDir() || !s.isInstalled(e.Name()) {
			continue
		}
		a, ok := s.catalog[e.Name()]
		if !ok {
			a = App{ID: e.Name(), Name: e.Name(), Category: "Perso", Custom: true}
			if b, err := os.ReadFile(filepath.Join(s.appsDir, e.Name(), "app.json")); err == nil {
				_ = json.Unmarshal(b, &a)
				a.Custom = true
			}
		}
		a.Installed = true
		a.Compose = ""
		out = append(out, a)
	}
	return out
}

// Render remplace les variables du modèle compose.
func (s *Store) Render(tpl, id string) string {
	return strings.NewReplacer(
		"{{APPDATA}}", filepath.Join(s.dataDir, "AppData", id),
		"{{DATA}}", s.dataDir,
		"{{TZ}}", s.tz,
		"{{PUID}}", "1000",
		"{{PGID}}", "1000",
	).Replace(tpl)
}

// Compose lit le fichier compose d'une app installée.
func (s *Store) Compose(id string) (string, error) {
	if !idRe.MatchString(id) {
		return "", errors.New("identifiant invalide")
	}
	b, err := os.ReadFile(s.composePath(id))
	return string(b), err
}

// AppsDir renvoie le dossier des définitions d'apps installées.
func (s *Store) AppsDir() string { return s.appsDir }

func (s *Store) startJob(id, action string, steps [][]string) error {
	st := make([]jobs.Step, len(steps))
	for i, a := range steps {
		st[i] = jobs.Step{Cmd: append([]string{"docker"}, a...)}
	}
	return s.jobs.Start("app", id, action, st, func(ok bool) {
		if ok && action == "uninstall" {
			_ = os.RemoveAll(filepath.Join(s.appsDir, id))
		}
	})
}

// Redeploy relance toutes les apps installées (après une restauration).
func (s *Store) Redeploy() {
	for _, a := range s.Installed() {
		_ = s.startJob(a.ID, "install", [][]string{s.composeArgs(a.ID, "up", "-d", "--remove-orphans")})
	}
}

func (s *Store) composeArgs(id string, extra ...string) []string {
	return append([]string{"compose", "-p", ProjectPrefix + id, "-f", s.composePath(id)}, extra...)
}

// Install installe une app. compose peut remplacer le modèle (édition avancée).
// Pour une app perso, meta donne le nom, l'icône et le port.
func (s *Store) Install(id, compose string, meta *App) error {
	if !idRe.MatchString(id) {
		return errors.New("identifiant invalide (minuscules, chiffres, tirets)")
	}
	a, inCatalog := s.catalog[id]
	if compose == "" {
		if !inCatalog {
			return errors.New("application inconnue")
		}
		compose = s.Render(a.Compose, id)
	}
	if !strings.Contains(compose, "services:") {
		return errors.New("fichier compose invalide (section services: manquante)")
	}
	dir := filepath.Join(s.appsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	s.prepareDirs(id, compose)
	if err := os.WriteFile(s.composePath(id), []byte(compose), 0o644); err != nil {
		return err
	}
	if !inCatalog && meta != nil {
		meta.ID, meta.Custom, meta.Compose = id, true, ""
		b, _ := json.MarshalIndent(meta, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, "app.json"), b, 0o644)
	}
	return s.startJob(id, "install", [][]string{
		s.composeArgs(id, "pull"),
		s.composeArgs(id, "up", "-d", "--remove-orphans"),
	})
}

// AppUID/AppGID sont l'utilisateur sous lequel tournent les apps (PUID/PGID).
const AppUID, AppGID = 1000, 1000

var bindRe = regexp.MustCompile(`(?m)^\s*-\s*"?(/[^:"\s]+):`)

// prepareDirs crée les dossiers montés par l'app avant Docker. Sinon Docker les crée
// en root et les apps qui tournent avec l'utilisateur 1000 ne peuvent pas y écrire.
func (s *Store) prepareDirs(id, compose string) {
	appData := filepath.Join(s.dataDir, "AppData", id)
	dirs := []string{appData}
	for _, m := range bindRe.FindAllStringSubmatch(compose, -1) {
		p := filepath.Clean(m[1])
		if strings.HasPrefix(p, appData+"/") && filepath.Ext(p) == "" {
			dirs = append(dirs, p)
		}
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			continue // dossier existant : on ne touche pas aux droits
		}
		if os.MkdirAll(d, 0o755) == nil {
			_ = os.Chown(d, AppUID, AppGID)
		}
	}
}

// Update télécharge les nouvelles images et recrée les conteneurs.
func (s *Store) Update(id string) error {
	if !idRe.MatchString(id) || !s.isInstalled(id) {
		return errors.New("application non installée")
	}
	return s.startJob(id, "update", [][]string{
		s.composeArgs(id, "pull"),
		s.composeArgs(id, "up", "-d", "--remove-orphans"),
		{"image", "prune", "-f"},
	})
}

// Control démarre/arrête/redémarre tous les conteneurs d'une app.
func (s *Store) Control(id, action string) error {
	if !idRe.MatchString(id) || !s.isInstalled(id) {
		return errors.New("application non installée")
	}
	switch action {
	case "start":
		return s.startJob(id, action, [][]string{s.composeArgs(id, "up", "-d")})
	case "stop", "restart":
		return s.startJob(id, action, [][]string{s.composeArgs(id, action)})
	}
	return errors.New("action inconnue")
}

// Uninstall supprime une app (les données dans AppData sont conservées).
func (s *Store) Uninstall(id string) error {
	if !idRe.MatchString(id) || !s.isInstalled(id) {
		return errors.New("application non installée")
	}
	return s.startJob(id, "uninstall", [][]string{s.composeArgs(id, "down", "--remove-orphans")})
}
