package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/azmoweb/azmoos/internal/totp"
)

func TestTwoFactorLogin(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	a := newClient(t, ts)
	a.do("POST", "/api/setup", map[string]string{"username": "azmo", "display": "Azmo", "password": "motdepasse1"}, true)

	_, begin := a.do("POST", "/api/me/2fa/begin", nil, true)
	secret, _ := begin["secret"].(string)
	if secret == "" || !strings.HasPrefix(begin["uri"].(string), "otpauth://totp/") {
		t.Fatalf("activation : %v", begin)
	}
	if code, _ := a.do("POST", "/api/me/2fa/confirm", map[string]string{"code": "000000"}, true); code != 400 {
		t.Fatalf("un mauvais code doit être refusé à l'activation, obtenu %d", code)
	}
	good, _ := totp.Code(secret, totp.Step(time.Now()))
	code, conf := a.do("POST", "/api/me/2fa/confirm", map[string]string{"code": good}, true)
	rec, _ := conf["recovery"].([]any)
	if code != 200 || len(rec) != 8 {
		t.Fatalf("confirmation : %d %v", code, conf)
	}

	// Connexion : le mot de passe seul ne suffit plus.
	b := newClient(t, ts)
	_, r := b.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1"}, true)
	if r["totp"] != true {
		t.Fatalf("le second facteur aurait dû être demandé : %v", r)
	}
	if _, st := b.do("GET", "/api/status", nil, false); st["authenticated"] == true {
		t.Fatal("connecté sans le second facteur")
	}
	if code, _ := b.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1", "code": "123456"}, true); code != 401 {
		t.Fatalf("mauvais code accepté : %d", code)
	}
	// Le code déjà utilisé pour l'activation ne peut pas resservir.
	if code, _ := b.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1", "code": good}, true); code != 401 {
		t.Fatalf("code rejoué accepté : %d", code)
	}
	// Code de secours : accepté une seule fois.
	rc := rec[0].(string)
	if code, _ := b.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1", "code": rc}, true); code != 200 {
		t.Fatalf("code de secours refusé : %d", code)
	}
	c := newClient(t, ts)
	if code, _ := c.do("POST", "/api/login", map[string]string{"username": "azmo", "password": "motdepasse1", "code": rc}, true); code != 401 {
		t.Fatalf("code de secours réutilisé : %d", code)
	}
	_, me := b.do("GET", "/api/me", nil, false)
	if u := me["user"].(map[string]any); u["totp"] != true || u["recoveryLeft"].(float64) != 7 {
		t.Fatalf("état 2FA incorrect : %v", u)
	}
	// Une alerte de sécurité a été créée pour l'utilisation du code de secours.
	_, n := a.do("GET", "/api/notifications", nil, false)
	if n["unread"].(float64) < 1 {
		t.Fatalf("pas d'alerte après l'usage d'un code de secours : %v", n)
	}
}

func TestTerminalRejectsForeignOrigin(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	a := newClient(t, ts)
	a.do("POST", "/api/setup", map[string]string{"username": "azmo", "password": "motdepasse1"}, true)
	req, _ := http.NewRequest("GET", ts.URL+"/api/terminal", nil)
	req.Header.Set("Origin", "http://pirate.example")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	res, err := a.hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("WebSocket depuis une origine étrangère : %d (403 attendu)", res.StatusCode)
	}
	// Désactivé : refusé même depuis la bonne origine.
	a.do("POST", "/api/terminal/settings", map[string]bool{"enabled": false}, true)
	req.Header.Set("Origin", ts.URL)
	res, _ = a.hc.Do(req)
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("terminal désactivé mais accessible : %d", res.StatusCode)
	}
}

func TestNotificationsAndSettings(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	a := newClient(t, ts)
	a.do("POST", "/api/setup", map[string]string{"username": "azmo", "password": "motdepasse1"}, true)
	_, st := a.do("GET", "/api/notifications/settings", nil, false)
	if st["tempWarn"].(float64) != 75 {
		t.Fatalf("réglages par défaut : %v", st)
	}
	st["tempWarn"] = 200
	if code, _ := a.do("POST", "/api/notifications/settings", st, true); code != 400 {
		t.Fatalf("seuil absurde accepté : %d", code)
	}
	st["tempWarn"] = 70
	if code, _ := a.do("POST", "/api/notifications/settings", st, true); code != 200 {
		t.Fatalf("réglages valides refusés : %d", code)
	}
	if code, _ := a.do("GET", "/api/https", nil, false); code != 200 {
		t.Fatalf("état HTTPS : %d", code)
	}
	res, _ := http.Get(ts.URL + "/api/https/ca.crt")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "application/x-x509-ca-cert" {
		t.Fatalf("certificat de l'autorité : %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	res.Body.Close()
}
