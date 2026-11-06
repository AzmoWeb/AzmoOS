// Package server expose l'API HTTP d'Azmo OS et sert l'interface web.
package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/appupdate"
	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/backup"
	"github.com/azmoweb/azmoos/internal/backupsched"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/files"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/notify"
	"github.com/azmoweb/azmoos/internal/shares"
	"github.com/azmoweb/azmoos/internal/storage"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
	"github.com/azmoweb/azmoos/internal/terminal"
	"github.com/azmoweb/azmoos/internal/tlsmgr"
	"github.com/azmoweb/azmoos/internal/updater"
)

var timeNow = time.Now

// Server regroupe tous les modules.
type Server struct {
	Version  string
	WebPort  int
	Config   *config.Store
	Auth     *auth.Auth
	Events   *events.Log
	Jobs     *jobs.Runner
	Docker   *docker.Client
	Store    *store.Store
	Files    *files.FS
	Storage  *storage.Manager
	Shares   *shares.Manager
	Backup   *backup.Manager
	Updater  *updater.Updater
	Monitor  *system.Monitor
	Hardware system.Hardware
	Web      fs.FS

	// v0.3
	Notify       *notify.Center
	Terminal     *terminal.Manager
	TLS          *tlsmgr.Manager
	Sched        *backupsched.Scheduler
	AppUpd       *appupdate.Scheduler
	HTTPSAddr    string
	HTTPSHandler http.Handler // gestionnaire servi en HTTPS (le même que HTTP, sans redirection)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func ok(w http.ResponseWriter) { writeJSON(w, 200, map[string]bool{"ok": true}) }

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		fail(w, 400, errors.New("requête invalide"))
		return false
	}
	return true
}

// log ajoute un événement au journal au nom de l'utilisateur courant.
func (s *Server) log(r *http.Request, level, cat, msg string) {
	user := ""
	if id := auth.From(r.Context()); id != nil {
		user = id.Username
	}
	s.Events.Add(level, cat, user, auth.ClientIP(r), msg)
}

// csrf exige un en-tête personnalisé sur les requêtes qui modifient l'état.
func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("X-Requested-With") != "AzmoOS" {
			http.Error(w, `{"error":"requête refusée"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type routeFn func(pattern, perm string, h http.HandlerFunc)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	handle := func(pattern, perm string, h http.HandlerFunc) {
		if perm == "public" {
			mux.HandleFunc(pattern, h)
			return
		}
		mux.Handle(pattern, s.Auth.Require(perm, h))
	}
	s.routesAuth(handle)
	s.routesSystem(handle)
	s.routesApps(handle)
	s.routesFiles(handle)
	s.routesStorage(handle)
	s.routesNetwork(handle)
	s.routesAdmin(handle)
	s.routesV3(handle)
	s.routesWallpaper(handle)
	s.routesV4(handle)
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail(w, 404, errors.New("route inconnue"))
	}))
	mux.Handle("/", s.static())
	return csrf(securityHeaders(mux))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// static sert l'interface (SPA : toute route inconnue renvoie index.html).
func (s *Server) static() http.Handler {
	fsrv := http.FileServer(http.FS(s.Web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(s.Web, p); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			p = "index.html"
		}
		if p == "index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fsrv.ServeHTTP(w, r)
	})
}

type errString string

func (e errString) Error() string { return string(e) }

// restartSoon arrête le processus : systemd le relance (Restart=always).
func restartSoon() {
	go func() {
		time.Sleep(1500 * time.Millisecond)
		os.Exit(0)
	}()
}
