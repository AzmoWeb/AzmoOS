package backupsched

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/azmoweb/azmoos/internal/backup"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/notify"
)

func at(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	return t
}

func TestSchedule(t *testing.T) {
	daily := config.Backup{Enabled: true, Frequency: "daily", Hour: 3}
	if got := LastDue(daily, at("2026-11-04 10:00")); !got.Equal(at("2026-11-04 03:00")) {
		t.Errorf("dernière échéance quotidienne : %v", got)
	}
	if got := LastDue(daily, at("2026-11-04 02:59")); !got.Equal(at("2026-11-03 03:00")) {
		t.Errorf("échéance de la veille : %v", got)
	}
	if got := Next(daily, at("2026-11-04 10:00")); !got.Equal(at("2026-11-05 03:00")) {
		t.Errorf("prochaine échéance : %v", got)
	}
	weekly := config.Backup{Enabled: true, Frequency: "weekly", Hour: 4, Weekday: 0} // dimanche 4 h
	// Le 4 novembre 2026 est un mercredi : dernière échéance = dimanche 1er novembre.
	if got := LastDue(weekly, at("2026-11-04 12:00")); !got.Equal(at("2026-11-01 04:00")) {
		t.Errorf("échéance hebdomadaire : %v", got)
	}
	// Une sauvegarde manquée (Pi éteint) est rattrapée…
	daily.LastRun = at("2026-11-02 03:00").Unix()
	if !Due(daily, at("2026-11-04 10:00")) {
		t.Error("une échéance manquée doit être rattrapée")
	}
	// … mais pas relancée une fois faite.
	daily.LastRun = at("2026-11-04 03:01").Unix()
	if Due(daily, at("2026-11-04 10:00")) {
		t.Error("sauvegarde relancée alors qu'elle est déjà faite")
	}
	daily.Enabled = false
	daily.LastRun = 0
	if Due(daily, at("2026-11-04 10:00")) {
		t.Error("une planification désactivée ne doit pas tourner")
	}
}

func TestRunAndPrune(t *testing.T) {
	conf, data := t.TempDir(), t.TempDir()
	cfg, _ := config.Load(conf)
	_ = cfg.Update(func(c *config.Config) error { c.Settings["x"] = "y"; return nil })
	apps := filepath.Join(data, "AppData", ".azmoos", "apps")
	_ = os.MkdirAll(apps, 0o755)
	r := jobs.NewRunner()
	s := &Scheduler{Config: cfg, Backup: &backup.Manager{ConfDir: conf, AppsDir: apps, DataDir: data}, Jobs: r,
		Events: events.Open(conf), Notify: notify.New(cfg), DataDir: data}
	b, err := s.Save(config.Backup{Name: "Test", Enabled: true, Frequency: "daily", Hour: 3, Target: "/Backups/Auto", Keep: 2})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(data, "Backups", "Auto")
	// Anciennes sauvegardes déjà présentes
	for _, n := range []string{"2026-01-01-0300", "2026-01-02-0300", "2026-01-03-0300"} {
		_ = os.WriteFile(filepath.Join(dir, prefix+n+".tar.gz"), []byte("x"), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "autre-fichier.txt"), []byte("x"), 0o644)
	if err := s.Run(b.ID, "test"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100 && r.Running("backup:"+b.ID); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	files, _ := s.Files(b.ID)
	if len(files) != 2 {
		t.Fatalf("2 sauvegardes devraient être gardées, trouvé %d : %v", len(files), files)
	}
	if files[0].Name < prefix+"2026-11" {
		t.Fatalf("la nouvelle sauvegarde devrait être la plus récente : %v", files)
	}
	if _, err := os.Stat(filepath.Join(dir, "autre-fichier.txt")); err != nil {
		t.Fatal("un fichier qui n'est pas une sauvegarde a été supprimé")
	}
	got, _ := s.get(b.ID)
	if !got.LastOK || got.LastFile == "" {
		t.Fatalf("le résultat n'a pas été enregistré : %+v", got)
	}
	// Une destination hors de /DATA est refusée.
	if _, err := s.Save(config.Backup{Frequency: "daily", Target: "../../etc", Keep: 3}); err == nil {
		if d, _ := s.localDir("../../etc"); !filepath.HasPrefix(d, data) {
			t.Fatal("destination hors de /DATA acceptée")
		}
	}
	if _, err := s.Save(config.Backup{Frequency: "daily", Target: "rclone:../x", Keep: 3}); err == nil {
		t.Fatal("destination cloud invalide acceptée")
	}
}
