// Package server expose l'API HTTP et l'interface web d'AzmoOS.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/files"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
)

type Server struct {
	Version string
	Auth    *auth.Auth
	Docker  *docker.Client
	Store   *store.Store
	Files   *files.FS
	Monitor *system.Monitor
	Web     fs.FS
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func ok(w http.ResponseWriter) { writeJSON(w, 200, map[string]bool{"ok": true}) }

func decode(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(v)
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

func (s *Server) Handler() http.Handler {
	pub := http.NewServeMux()
	api := http.NewServeMux()

	pub.HandleFunc("GET /api/status", s.status)
	pub.HandleFunc("POST /api/setup", s.setup)
	pub.HandleFunc("POST /api/login", s.login)
	pub.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) { s.Auth.Logout(w, r); ok(w) })

	api.HandleFunc("GET /api/system", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Monitor.Snapshot()) })
	api.HandleFunc("POST /api/system/power", s.power)
	api.HandleFunc("POST /api/system/hostname", s.hostname)
	api.HandleFunc("POST /api/account", s.account)

	api.HandleFunc("GET /api/docker/containers", s.containers)
	api.HandleFunc("POST /api/docker/containers/{id}/{action}", s.containerAction)
	api.HandleFunc("GET /api/docker/containers/{id}/logs", s.containerLogs)

	api.HandleFunc("GET /api/store/catalog", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Store.Catalog()) })
	api.HandleFunc("GET /api/apps", s.apps)
	api.HandleFunc("GET /api/apps/{id}/compose", s.appCompose)
	api.HandleFunc("POST /api/apps/{id}/{action}", s.appAction)
	api.HandleFunc("GET /api/jobs", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Store.Jobs()) })

	api.HandleFunc("GET /api/files", s.fileList)
	api.HandleFunc("GET /api/files/download", s.fileDownload)
	api.HandleFunc("POST /api/files/upload", s.fileUpload)
	api.HandleFunc("POST /api/files/{op}", s.fileOp)

	pub.Handle("/api/", s.Auth.Require(api))
	static := http.FileServer(http.FS(s.Web))
	pub.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	}))
	return csrf(pub)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	authed := s.Auth.Valid(r)
	res := map[string]any{"version": s.Version, "initialized": s.Auth.Initialized(), "authenticated": authed}
	if authed {
		res["username"] = s.Auth.Username()
		res["docker"] = s.Docker.Available()
	}
	writeJSON(w, 200, res)
}

type creds struct{ Username, Password string }

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if s.Auth.Initialized() {
		fail(w, 409, errString("AzmoOS est déjà configuré"))
		return
	}
	var c creds
	if err := decode(r, &c); err != nil {
		fail(w, 400, err)
		return
	}
	if err := s.Auth.SetUser(c.Username, c.Password); err != nil {
		fail(w, 400, err)
		return
	}
	_ = s.Auth.NewSession(w, r)
	ok(w)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var c creds
	if err := decode(r, &c); err != nil {
		fail(w, 400, err)
		return
	}
	if !s.Auth.Check(c.Username, c.Password) {
		time.Sleep(time.Second) // ralentit la force brute
		fail(w, 401, auth.ErrInvalid)
		return
	}
	if err := s.Auth.NewSession(w, r); err != nil {
		fail(w, 500, err)
		return
	}
	ok(w)
}

func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	var req struct{ Username, Current, Password string }
	if err := decode(r, &req); err != nil {
		fail(w, 400, err)
		return
	}
	if !s.Auth.Check(s.Auth.Username(), req.Current) {
		fail(w, 403, errString("mot de passe actuel incorrect"))
		return
	}
	if req.Username == "" {
		req.Username = s.Auth.Username()
	}
	if err := s.Auth.SetUser(req.Username, req.Password); err != nil {
		fail(w, 400, err)
		return
	}
	_ = s.Auth.NewSession(w, r)
	ok(w)
}

func (s *Server) power(w http.ResponseWriter, r *http.Request) {
	var req struct{ Action string }
	if err := decode(r, &req); err != nil {
		fail(w, 400, err)
		return
	}
	if err := system.Power(req.Action); err != nil {
		fail(w, 400, err)
		return
	}
	ok(w)
}

var hostRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func (s *Server) hostname(w http.ResponseWriter, r *http.Request) {
	var req struct{ Hostname string }
	if err := decode(r, &req); err != nil || !hostRe.MatchString(req.Hostname) {
		fail(w, 400, errString("nom d'hôte invalide"))
		return
	}
	if err := system.SetHostname(req.Hostname); err != nil {
		fail(w, 500, err)
		return
	}
	ok(w)
}

func (s *Server) containers(w http.ResponseWriter, r *http.Request) {
	list, err := s.Docker.List()
	if err != nil {
		fail(w, 503, err)
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) containerAction(w http.ResponseWriter, r *http.Request) {
	if err := s.Docker.Action(r.PathValue("id"), r.PathValue("action")); err != nil {
		fail(w, 400, err)
		return
	}
	ok(w)
}

func (s *Server) containerLogs(w http.ResponseWriter, r *http.Request) {
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	if tail <= 0 || tail > 5000 {
		tail = 300
	}
	logs, err := s.Docker.Logs(r.PathValue("id"), tail)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]string{"logs": logs})
}

type appView struct {
	store.App
	State      string             `json:"state"` // running, stopped, partial, unknown
	Containers []docker.Container `json:"containers"`
}

func (s *Server) apps(w http.ResponseWriter, r *http.Request) {
	list, _ := s.Docker.List()
	byProject := map[string][]docker.Container{}
	for _, c := range list {
		byProject[c.Project] = append(byProject[c.Project], c)
	}
	out := []appView{}
	for _, a := range s.Store.Installed() {
		v := appView{App: a, Containers: byProject[store.ProjectPrefix+a.ID], State: "stopped"}
		running := 0
		for _, c := range v.Containers {
			if c.State == "running" {
				running++
			}
		}
		switch {
		case len(v.Containers) == 0 && list == nil:
			v.State = "unknown"
		case running > 0 && running == len(v.Containers):
			v.State = "running"
		case running > 0:
			v.State = "partial"
		}
		if v.Containers == nil {
			v.Containers = []docker.Container{}
		}
		out = append(out, v)
	}
	// Conteneurs non gérés par AzmoOS
	others := []docker.Container{}
	for _, c := range list {
		if !strings.HasPrefix(c.Project, store.ProjectPrefix) {
			others = append(others, c)
		}
	}
	writeJSON(w, 200, map[string]any{"apps": out, "others": others})
}

func (s *Server) appCompose(w http.ResponseWriter, r *http.Request) {
	c, err := s.Store.Compose(r.PathValue("id"))
	if err != nil {
		fail(w, 404, err)
		return
	}
	writeJSON(w, 200, map[string]string{"compose": c})
}

func (s *Server) appAction(w http.ResponseWriter, r *http.Request) {
	id, action := r.PathValue("id"), r.PathValue("action")
	var err error
	switch action {
	case "install":
		var req struct {
			Compose string
			Name    string
			Icon    string
			Port    int
			Path    string
		}
		if r.ContentLength != 0 {
			if e := decode(r, &req); e != nil {
				fail(w, 400, e)
				return
			}
		}
		err = s.Store.Install(id, req.Compose, &store.App{Name: req.Name, Icon: req.Icon, Port: req.Port, Path: req.Path, Category: "Perso"})
	case "update":
		err = s.Store.Update(id)
	case "uninstall":
		err = s.Store.Uninstall(id)
	default:
		err = s.Store.Control(id, action)
	}
	if err != nil {
		fail(w, 400, err)
		return
	}
	ok(w)
}

func (s *Server) fileList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Files.List(r.URL.Query().Get("path"))
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, entries)
}

func (s *Server) fileDownload(w http.ResponseWriter, r *http.Request) {
	p, err := s.Files.Resolve(r.URL.Query().Get("path"))
	if err != nil {
		fail(w, 400, err)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		fail(w, 404, err)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		fail(w, 400, errString("pas un fichier"))
		return
	}
	disp := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", disp+`; filename*=UTF-8''`+urlEscape(filepath.Base(p)))
	w.Header().Set("Content-Security-Policy", "sandbox")
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}

func (s *Server) fileUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		fail(w, 400, err)
		return
	}
	defer r.MultipartForm.RemoveAll()
	dir := r.URL.Query().Get("path")
	for _, fh := range r.MultipartForm.File["files"] {
		if err := s.Files.Save(dir, fh); err != nil {
			fail(w, 400, err)
			return
		}
	}
	ok(w)
}

func (s *Server) fileOp(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path, Name string }
	if err := decode(r, &req); err != nil {
		fail(w, 400, err)
		return
	}
	var err error
	switch r.PathValue("op") {
	case "mkdir":
		err = s.Files.Mkdir(req.Path, req.Name)
	case "rename":
		err = s.Files.Rename(req.Path, req.Name)
	case "delete":
		err = s.Files.Delete(req.Path)
	default:
		err = errString("opération inconnue")
	}
	if err != nil {
		fail(w, 400, err)
		return
	}
	ok(w)
}

type errString string

func (e errString) Error() string { return string(e) }

func urlEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.IndexByte("-._~", c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(c)|0x100, 16)[1:]))
		}
	}
	return b.String()
}
