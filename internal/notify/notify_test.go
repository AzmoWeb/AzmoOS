package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/azmoweb/azmoos/internal/config"
)

func TestDedupeResolveAndPush(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		mu.Lock()
		got = append(got, m)
		mu.Unlock()
	}))
	defer srv.Close()

	cfg, _ := config.Load(t.TempDir())
	_ = cfg.Update(func(c *config.Config) error {
		c.Notify.Ntfy = config.NtfyChannel{Enabled: true, Server: srv.URL, Topic: "azmo-test"}
		return nil
	})
	c := New(cfg)
	if !c.Raise("temp", "temperature", Warning, "Chaud", "78 °C") {
		t.Fatal("première alerte non créée")
	}
	if c.Raise("temp", "temperature", Warning, "Chaud", "79 °C") {
		t.Fatal("alerte répétée alors qu'elle est toujours en cours")
	}
	c.Resolve("temp")
	if !c.Raise("temp", "temperature", Warning, "Chaud", "80 °C") {
		t.Fatal("l'alerte doit pouvoir revenir après résolution")
	}
	c.Raise("", "backups", Info, "Sauvegarde réussie", "ok") // info : pas envoyée (seuil warning)

	items, unread := c.List()
	if len(items) != 3 || unread != 3 {
		t.Fatalf("liste : %d éléments, %d non lus", len(items), unread)
	}
	c.MarkRead(0)
	if _, unread := c.List(); unread != 0 {
		t.Fatal("tout marquer comme lu n'a pas fonctionné")
	}
	time.Sleep(300 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("2 envois ntfy attendus (avertissements seulement), reçu %d", len(got))
	}
	if got[0]["topic"] != "azmo-test" || got[0]["priority"].(float64) != 4 {
		t.Fatalf("message ntfy incorrect : %v", got[0])
	}
	// Persistance : un nouveau centre relit l'historique et les alertes actives.
	c2 := New(cfg)
	if items, _ := c2.List(); len(items) != 3 || !c2.Active("temp") {
		t.Fatal("historique ou alertes actives non conservés")
	}
}
