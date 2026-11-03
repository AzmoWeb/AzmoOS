package server

import "testing"

func TestWidgetsAndNote(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	a := newClient(t, ts)
	a.do("POST", "/api/setup", map[string]string{"username": "azmo", "display": "Azmo", "password": "motdepasse1"}, true)

	if code, _ := a.do("PUT", "/api/me/widgets", map[string]any{"widgets": []string{"note", "clock", "note"}}, true); code != 200 {
		t.Fatalf("widgets : %d", code)
	}
	_, me := a.do("GET", "/api/me", nil, false)
	u, _ := me["user"].(map[string]any)
	w, _ := u["widgets"].([]any)
	if len(w) != 2 || w[0] != "note" || w[1] != "clock" {
		t.Fatalf("ordre ou doublons mal gérés : %v", u["widgets"])
	}
	if code, _ := a.do("PUT", "/api/me/widgets", map[string]any{"widgets": []string{"meteo"}}, true); code != 400 {
		t.Fatalf("un widget inconnu doit être refusé : %d", code)
	}
	a.do("PUT", "/api/me/widgets", map[string]any{"widgets": nil}, true)
	if _, me := a.do("GET", "/api/me", nil, false); me["user"].(map[string]any)["widgets"] != nil {
		t.Fatalf("retour à la disposition par défaut attendu : %v", me["user"])
	}

	a.do("PUT", "/api/me/note", map[string]string{"text": "Penser au lait 🥛"}, true)
	if _, n := a.do("GET", "/api/me/note", nil, false); n["text"] != "Penser au lait 🥛" {
		t.Fatalf("pense-bête : %v", n)
	}
	long := make([]byte, 4001)
	for i := range long {
		long[i] = 'a'
	}
	if code, _ := a.do("PUT", "/api/me/note", map[string]string{"text": string(long)}, true); code != 400 {
		t.Fatalf("pense-bête trop long accepté : %d", code)
	}
}

func TestAppUpdatesSettings(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	a := newClient(t, ts)
	a.do("POST", "/api/setup", map[string]string{"username": "azmo", "display": "Azmo", "password": "motdepasse1"}, true)

	_, g := a.do("GET", "/api/app-updates", nil, false)
	st, _ := g["settings"].(map[string]any)
	if st["enabled"] != false || st["hour"] != float64(4) {
		t.Fatalf("réglages par défaut : %v", g)
	}
	if code, _ := a.do("PUT", "/api/app-updates", map[string]any{"enabled": true, "frequency": "yearly", "hour": 3}, true); code != 400 {
		t.Fatalf("fréquence invalide acceptée : %d", code)
	}
	if code, _ := a.do("PUT", "/api/app-updates", map[string]any{"enabled": true, "frequency": "weekly", "hour": 3, "weekday": 6, "exclude": []string{"jellyfin"}}, true); code != 200 {
		t.Fatalf("enregistrement : %d", code)
	}
	_, g = a.do("GET", "/api/app-updates", nil, false)
	st = g["settings"].(map[string]any)
	if st["enabled"] != true || st["frequency"] != "weekly" || g["next"].(float64) == 0 {
		t.Fatalf("réglages relus : %v", g)
	}
	if ex := st["exclude"].([]any); len(ex) != 1 || ex[0] != "jellyfin" {
		t.Fatalf("exclusions : %v", ex)
	}
	if code, _ := a.do("POST", "/api/app-updates/run", nil, true); code != 200 {
		t.Fatalf("lancement manuel : %d", code)
	}

	// Un membre sans la permission « apps.manage » ne voit pas ces réglages.
	a.do("POST", "/api/users", map[string]any{"username": "leo", "display": "Léo", "password": "motdepasse2", "role": "member"}, true)
	b := newClient(t, ts)
	b.do("POST", "/api/login", map[string]string{"username": "leo", "password": "motdepasse2"}, true)
	if code, _ := b.do("GET", "/api/app-updates", nil, false); code != 403 {
		t.Fatalf("membre : %d", code)
	}
}
