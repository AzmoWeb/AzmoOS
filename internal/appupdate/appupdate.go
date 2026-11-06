// Package appupdate met à jour automatiquement les applications installées,
// la nuit, sans toucher à celles qui sont arrêtées ou exclues.
package appupdate

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/azmoweb/azmoos/internal/backupsched"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/notify"
	"github.com/azmoweb/azmoos/internal/store"
)

const maxHistory = 50

// Apps est la partie de l'App Store utilisée ici (remplaçable dans les tests).
type Apps interface {
	Installed() []store.App
	AutoUpdate(id string) (string, error)
}

type Scheduler struct {
	Config *config.Store
	Apps   Apps
	Events *events.Log
	Notify *notify.Center

	mu      sync.Mutex
	running bool
	current string
}

var ErrBusy = errors.New("une vérification des mises à jour est déjà en cours")

// Settings renvoie les réglages (valeurs par défaut si jamais configurés).
func (s *Scheduler) Settings() config.AppUpdates {
	out := config.DefaultAppUpdates()
	s.Config.View(func(c *config.Config) {
		if c.AppUpd != nil {
			out = *c.AppUpd
			out.Exclude = slices.Clone(c.AppUpd.Exclude)
			out.History = slices.Clone(c.AppUpd.History)
		}
	})
	if out.Exclude == nil {
		out.Exclude = []string{}
	}
	if out.History == nil {
		out.History = []config.AppUpdateEntry{}
	}
	return out
}

// Save enregistre les réglages modifiables (le reste est conservé).
func (s *Scheduler) Save(in config.AppUpdates) error {
	if in.Frequency != "daily" && in.Frequency != "weekly" {
		return errors.New("fréquence invalide")
	}
	if in.Hour < 0 || in.Hour > 23 || in.Weekday < 0 || in.Weekday > 6 {
		return errors.New("heure ou jour invalide")
	}
	ex := []string{}
	for _, id := range in.Exclude {
		if id != "" && len(id) <= 41 && !slices.Contains(ex, id) {
			ex = append(ex, id)
		}
	}
	return s.Config.Update(func(c *config.Config) error {
		cur := config.DefaultAppUpdates()
		if c.AppUpd != nil {
			cur = *c.AppUpd
		}
		wasOff := !cur.Enabled
		cur.Enabled, cur.Frequency, cur.Hour, cur.Weekday, cur.Exclude = in.Enabled, in.Frequency, in.Hour, in.Weekday, ex
		if wasOff && in.Enabled && cur.LastRun == 0 {
			// Activation : on attend la prochaine échéance au lieu de tout lancer tout de suite.
			cur.LastRun = time.Now().Unix()
		}
		c.AppUpd = &cur
		return nil
	})
}

func asBackup(a config.AppUpdates) config.Backup {
	return config.Backup{Enabled: a.Enabled, Frequency: a.Frequency, Hour: a.Hour, Weekday: a.Weekday, LastRun: a.LastRun}
}

// Due indique si un passage doit être lancé (y compris une échéance manquée).
func Due(a config.AppUpdates, now time.Time) bool { return backupsched.Due(asBackup(a), now) }

// Next renvoie la prochaine échéance.
func Next(a config.AppUpdates, now time.Time) time.Time { return backupsched.Next(asBackup(a), now) }

// Status indique si un passage est en cours et l'app en train d'être vérifiée.
func (s *Scheduler) Status() (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running, s.current
}

// Loop vérifie chaque minute s'il faut lancer les mises à jour.
func (s *Scheduler) Loop() {
	time.Sleep(2 * time.Minute)
	for {
		if Due(s.Settings(), time.Now()) {
			_ = s.Run("")
		}
		time.Sleep(time.Minute)
	}
}

// Start lance un passage en arrière-plan.
func (s *Scheduler) Start(user string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrBusy
	}
	s.running = true
	s.mu.Unlock()
	go s.run(user)
	return nil
}

// Run lance un passage et attend la fin.
func (s *Scheduler) Run(user string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrBusy
	}
	s.running = true
	s.mu.Unlock()
	s.run(user)
	return nil
}

func (s *Scheduler) run(user string) {
	defer func() {
		s.mu.Lock()
		s.running, s.current = false, ""
		s.mu.Unlock()
	}()
	cfg := s.Settings()
	var updated, failed []string
	var entries []config.AppUpdateEntry
	checked := 0
	for _, a := range s.Apps.Installed() {
		if slices.Contains(cfg.Exclude, a.ID) {
			continue
		}
		s.mu.Lock()
		s.current = a.ID
		s.mu.Unlock()
		name := a.Name
		if name == "" {
			name = a.ID
		}
		res, err := s.Apps.AutoUpdate(a.ID)
		if res == store.UpdSkipped {
			continue
		}
		checked++
		e := config.AppUpdateEntry{App: a.ID, Name: name, Time: time.Now().Unix()}
		switch {
		case err != nil:
			e.Result, e.Error = "error", err.Error()
			failed = append(failed, name)
			s.Events.Add(events.Warning, "apps", user, "", fmt.Sprintf("Mise à jour automatique de « %s » impossible : %s", name, err))
		case res == store.UpdUpdated:
			e.Result = "updated"
			updated = append(updated, name)
			s.Events.Add(events.Info, "apps", user, "", fmt.Sprintf("« %s » mise à jour automatiquement", name))
		default:
			continue
		}
		entries = append(entries, e)
	}
	_ = s.Config.Update(func(c *config.Config) error {
		cur := config.DefaultAppUpdates()
		if c.AppUpd != nil {
			cur = *c.AppUpd
		}
		cur.LastRun, cur.LastCheck, cur.LastCount, cur.LastFail = time.Now().Unix(), checked, len(updated), len(failed)
		slices.Reverse(entries)
		cur.History = append(entries, cur.History...)
		if len(cur.History) > maxHistory {
			cur.History = cur.History[:maxHistory]
		}
		c.AppUpd = &cur
		return nil
	})
	if s.Notify == nil || !s.Notify.Settings().Updates {
		return
	}
	lang := s.Notify.Settings().Lang
	L := func(fr, en string) string { return notify.T(lang, fr, en) }
	if len(updated) > 0 {
		title := L("Applications mises à jour", "Apps updated")
		if len(updated) == 1 {
			title = L("Application mise à jour", "App updated")
		}
		s.Notify.Raise("", "updates", notify.Info, title, fmt.Sprintf(L("Nouvelle version installée pour : %s.", "New version installed for: %s."), strings.Join(updated, ", ")))
	}
	if len(failed) > 0 {
		s.Notify.Raise("", "updates", notify.Warning, L("Mise à jour d'application impossible", "App update failed"),
			fmt.Sprintf(L("Ces applications n'ont pas pu être mises à jour : %s. Elles continuent avec leur version actuelle.", "These apps could not be updated: %s. They keep running their current version."), strings.Join(failed, ", ")))
	}
}
