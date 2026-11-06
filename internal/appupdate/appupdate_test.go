package appupdate

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/notify"
	"github.com/azmoweb/azmoos/internal/store"
)

type fakeApps struct {
	results map[string]string
	errs    map[string]error
	called  []string
}

func (f *fakeApps) Installed() []store.App {
	return []store.App{{ID: "jellyfin", Name: "Jellyfin"}, {ID: "immich", Name: "Immich"}, {ID: "pihole", Name: "Pi-hole"},
		{ID: "kuma", Name: "Uptime Kuma"}, {ID: "nextcloud", Name: "Nextcloud"}}
}

func (f *fakeApps) AutoUpdate(id string) (string, error) {
	f.called = append(f.called, id)
	return f.results[id], f.errs[id]
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	apps := &fakeApps{
		results: map[string]string{"jellyfin": store.UpdUpdated, "immich": store.UpdUpToDate, "kuma": store.UpdSkipped, "nextcloud": store.UpdUpdated},
		errs:    map[string]error{"pihole": errors.New("pull access denied")},
	}
	n := notify.New(cfg)
	s := &Scheduler{Config: cfg, Apps: apps, Events: events.Open(dir), Notify: n}
	if err := s.Save(config.AppUpdates{Enabled: true, Frequency: "daily", Hour: 4, Exclude: []string{"nextcloud"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Run("azmo"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(apps.called, ",") != "jellyfin,immich,pihole,kuma" {
		t.Fatalf("une app exclue ne doit pas être vérifiée : %v", apps.called)
	}
	st := s.Settings()
	if st.LastCheck != 3 || st.LastCount != 1 || st.LastFail != 1 {
		t.Fatalf("bilan : vérifiées %d, mises à jour %d, échecs %d", st.LastCheck, st.LastCount, st.LastFail)
	}
	if len(st.History) != 2 {
		t.Fatalf("seules les mises à jour et les échecs sont gardés : %v", st.History)
	}
	items, _ := n.List()
	var titles []string
	for _, it := range items {
		titles = append(titles, it.Title+" : "+it.Message)
	}
	all := strings.Join(titles, "\n")
	if !strings.Contains(all, "Jellyfin") || !strings.Contains(all, "Pi-hole") || strings.Contains(all, "Immich") {
		t.Fatalf("notifications :\n%s", all)
	}
	if Due(st, time.Now()) {
		t.Fatal("rien ne doit être relancé juste après un passage")
	}
}

func TestSchedule(t *testing.T) {
	cfg, _ := config.Load(t.TempDir())
	s := &Scheduler{Config: cfg}
	if s.Settings().Enabled {
		t.Fatal("désactivées par défaut")
	}
	if err := s.Save(config.AppUpdates{Frequency: "monthly"}); err == nil {
		t.Fatal("fréquence invalide acceptée")
	}
	if err := s.Save(config.AppUpdates{Frequency: "daily", Hour: 24}); err == nil {
		t.Fatal("heure invalide acceptée")
	}
	// À l'activation, on attend la prochaine échéance au lieu de tout mettre à jour d'un coup.
	_ = s.Save(config.AppUpdates{Enabled: true, Frequency: "daily", Hour: 4})
	if Due(s.Settings(), time.Now()) {
		t.Fatal("ne doit pas se lancer dès l'activation")
	}
	loc := time.Local
	a := config.AppUpdates{Enabled: true, Frequency: "weekly", Hour: 3, Weekday: 0} // dimanche 3 h
	mon := time.Date(2026, 11, 2, 12, 0, 0, 0, loc)                                 // lundi
	if got := Next(a, mon); got.Weekday() != time.Sunday || got.Hour() != 3 || !got.After(mon) {
		t.Fatalf("prochaine échéance : %v", got)
	}
	a.LastRun = time.Date(2026, 10, 25, 3, 0, 0, 0, loc).Unix() // dimanche précédent
	if !Due(a, mon) {
		t.Fatal("échéance de dimanche manquée (Pi éteint) : à rattraper")
	}
}
