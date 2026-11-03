// Package store gère l'App Store AzmoOS (applications Docker Compose).
package store

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
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
	Installed   bool     `json:"installed"`
	Custom      bool     `json:"custom,omitempty"`
}

type Job struct {
	App    string    `json:"app"`
	Action string    `json:"action"`
	Status string    `json:"status"` // running, done, error
	Log    string    `json:"log"`
	Start  time.Time `json:"start"`
}

type Store struct {
	dataDir string // /DATA
	appsDir string // /DATA/AppData/.azmoos/apps
	tz      string
	catalog map[string]App
	mu      sync.Mutex
	jobs    map[string]*Job
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,40}$`)

func New(dataDir string) (*Store, error) {
	s := &Store{
		dataDir: dataDir,
		appsDir: filepath.Join(dataDir, "AppData", ".azmoos", "apps"),
		catalog: map[string]App{},
		jobs:    map[string]*Job{},
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

// Jobs renvoie l'état des opérations en cours/récentes.
func (s *Store) Jobs() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Job{}
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Start.After(out[k].Start) })
	return out
}

func (s *Store) startJob(id, action string, steps [][]string) error {
	s.mu.Lock()
	if j, ok := s.jobs[id]; ok && j.Status == "running" {
		s.mu.Unlock()
		return errors.New("une opération est déjà en cours pour cette app")
	}
	j := &Job{App: id, Action: action, Status: "running", Start: time.Now()}
	s.jobs[id] = j
	s.mu.Unlock()

	go func() {
		for _, args := range steps {
			cmd := exec.Command("docker", args...)
			var buf bytes.Buffer
			cmd.Stdout, cmd.Stderr = &buf, &buf
			err := cmd.Run()
			s.mu.Lock()
			j.Log += "$ docker " + strings.Join(args, " ") + "\n" + buf.String()
			if err != nil {
				j.Status = "error"
				j.Log += "\nErreur: " + err.Error()
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
		}
		s.mu.Lock()
		j.Status = "done"
		s.mu.Unlock()
		if action == "uninstall" {
			_ = os.RemoveAll(filepath.Join(s.appsDir, id))
		}
	}()
	return nil
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
	_ = os.MkdirAll(filepath.Join(s.dataDir, "AppData", id), 0o755)
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
