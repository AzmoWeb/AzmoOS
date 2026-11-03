package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/backup"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/files"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/shares"
	"github.com/azmoweb/azmoos/internal/storage"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
	"github.com/azmoweb/azmoos/internal/updater"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	conf, data := t.TempDir(), t.TempDir()
	cfg, err := config.Load(conf)
	if err != nil {
		t.Fatal(err)
	}
	r := jobs.NewRunner()
	st, err := store.New(data, r)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{
		Version: "test", WebPort: 80, Config: cfg, Auth: auth.New(cfg), Events: events.Open(conf), Jobs: r,
		Docker: docker.New("/nonexistent.sock"), Store: st, Files: files.New(data), Storage: storage.New(data, r),
		Shares: shares.New(cfg, data), Backup: &backup.Manager{ConfDir: conf, AppsDir: st.AppsDir(), DataDir: data},
		Updater: updater.New("x/y", "test", "", r), Monitor: system.NewMonitor(data), Hardware: system.DetectHardware(),
		Web: fstest.MapFS{"index.html": {Data: []byte("<html>azmo</html>")}},
	}
	return httptest.NewServer(s.Handler())
}

type client struct {
	t  *testing.T
	hc *http.Client
	u  string
}

func newClient(t *testing.T, ts *httptest.Server) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, hc: &http.Client{Jar: jar}, u: ts.URL}
}

func (c *client) do(method, path string, body any, csrf bool) (int, map[string]any) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.u+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if csrf {
		req.Header.Set("X-Requested-With", "AzmoOS")
	}
	res, err := c.hc.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestAuthFlowAndPermissions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	admin := newClient(t, ts)

	if code, s := admin.do("GET", "/api/status", nil, false); code != 200 || s["initialized"] != false {
		t.Fatalf("status initial : %d %v", code, s)
	}
	if code, _ := admin.do("POST", "/api/setup", map[string]string{"username": "azmo", "password": "motdepasse1"}, false); code != 403 {
		t.Fatalf("CSRF absent devrait être refusé, obtenu %d", code)
	}
	if code, r := admin.do("POST", "/api/setup", map[string]string{"username": "azmo", "display": "Azmo", "password": "court"}, true); code != 400 {
		t.Fatalf("mot de passe court accepté : %d %v", code, r)
	}
	if code, r := admin.do("POST", "/api/setup", map[string]string{"username": "azmo", "display": "Azmo", "password": "motdepasse1"}, true); code != 200 {
		t.Fatalf("setup : %d %v", code, r)
	}
	if code, _ := admin.do("POST", "/api/setup", map[string]string{"username": "pirate", "password": "motdepasse1"}, true); code != 409 {
		t.Fatalf("un second setup doit être refusé, obtenu %d", code)
	}
	if _, s := admin.do("GET", "/api/status", nil, false); s["authenticated"] != true {
		t.Fatalf("l'admin devrait être connecté après le setup : %v", s)
	}

	// Création d'un membre avec lecture des fichiers seulement.
	if code, r := admin.do("POST", "/api/users", map[string]any{"username": "camille", "display": "Camille", "password": "motdepasse2", "role": "member", "perms": []string{"files.read"}}, true); code != 200 {
		t.Fatalf("création membre : %d %v", code, r)
	}
	member := newClient(t, ts)
	if code, _ := member.do("GET", "/api/files", nil, false); code != 401 {
		t.Fatalf("non connecté devrait recevoir 401, obtenu %d", code)
	}
	if code, r := member.do("POST", "/api/login", map[string]string{"username": "camille", "password": "motdepasse2"}, true); code != 200 {
		t.Fatalf("login membre : %d %v", code, r)
	}
	if code, _ := member.do("GET", "/api/files?path=/", nil, false); code != 200 {
		t.Fatalf("le membre doit pouvoir lire les fichiers, obtenu %d", code)
	}
	if code, _ := member.do("POST", "/api/files/mkdir", map[string]string{"path": "/", "name": "x"}, true); code != 403 {
		t.Fatalf("le membre ne doit pas écrire, obtenu %d", code)
	}
	for _, p := range []string{"/api/users", "/api/events", "/api/network", "/api/storage/disks", "/api/updates"} {
		if code, _ := member.do("GET", p, nil, false); code != 403 {
			t.Fatalf("%s devrait être réservé à l'admin, obtenu %d", p, code)
		}
	}
	if code, _ := admin.do("DELETE", "/api/users/azmo", nil, true); code != 400 {
		t.Fatalf("l'admin ne doit pas pouvoir se supprimer lui-même, obtenu %d", code)
	}

	// Anti force brute : 8 échecs → blocage.
	bad := newClient(t, ts)
	for i := 0; i < 8; i++ {
		bad.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "faux"}, true)
	}
	if code, _ := bad.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1"}, true); code != 429 {
		t.Fatalf("l'IP devrait être bloquée, obtenu %d", code)
	}

	// Le journal a bien tracé les événements.
	req, _ := http.NewRequest("GET", ts.URL+"/api/events", nil)
	res, _ := admin.hc.Do(req)
	var evs []events.Event
	_ = json.NewDecoder(res.Body).Decode(&evs)
	res.Body.Close()
	if len(evs) < 3 {
		t.Fatalf("journal trop court : %v", evs)
	}

	// SPA : une route inconnue renvoie index.html.
	res, _ = http.Get(ts.URL + "/apps")
	if res.StatusCode != 200 {
		t.Fatalf("fallback SPA : %d", res.StatusCode)
	}
	res.Body.Close()
}
