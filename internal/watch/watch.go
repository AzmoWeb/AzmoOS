// Package watch surveille le Pi en continu et déclenche les alertes.
package watch

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/docker"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/notify"
	"github.com/azmoweb/azmoos/internal/store"
	"github.com/azmoweb/azmoos/internal/system"
	"github.com/azmoweb/azmoos/internal/updater"
)

type Watcher struct {
	Notify   *notify.Center
	Monitor  *system.Monitor
	Docker   *docker.Client
	Store    *store.Store
	Jobs     *jobs.Runner
	Updater  *updater.Updater
	Interval time.Duration

	prevState map[string]string
	restarts  map[string]int
	memHigh   int
}

// Run lance la surveillance (bloquant).
func (w *Watcher) Run() {
	if w.Interval == 0 {
		w.Interval = time.Minute
	}
	w.prevState, w.restarts = map[string]string{}, map[string]int{}
	time.Sleep(30 * time.Second) // laisse le système démarrer
	for {
		w.Check()
		time.Sleep(w.Interval)
	}
}

// Check effectue une passe de vérifications (exposé pour les tests).
func (w *Watcher) Check() {
	if w.prevState == nil {
		w.prevState, w.restarts = map[string]string{}, map[string]int{}
	}
	cfg := w.Notify.Settings()
	L := func(fr, en string) string { return notify.T(cfg.Lang, fr, en) }
	s := w.Monitor.Snapshot()

	// Température
	if cfg.Temperature && s.TempC > 0 {
		switch {
		case s.TempC >= float64(cfg.TempWarn+7):
			w.Notify.Raise("temp-critical", "temperature", notify.Critical, L("Ton Pi surchauffe", "Your Pi is overheating"),
				fmt.Sprintf(L("%.0f °C : il ralentit pour se protéger. Vérifie la ventilation dès que possible.", "%.0f °C: it is slowing down to protect itself. Check the cooling as soon as possible."), s.TempC))
		case s.TempC >= float64(cfg.TempWarn):
			w.Notify.Raise("temp", "temperature", notify.Warning, L("Il fait chaud dans ton Pi", "Your Pi is getting hot"),
				fmt.Sprintf(L("%.0f °C. Rien de grave pour l'instant, mais un ventilateur l'aiderait.", "%.0f °C. Nothing serious yet, but a fan would help."), s.TempC))
		case s.TempC < float64(cfg.TempWarn-5):
			w.Notify.Resolve("temp")
			w.Notify.Resolve("temp-critical")
		}
	}
	// Alimentation (vcgencmd get_throttled : bit 0 = sous-tension en ce moment)
	if cfg.Temperature && s.Throttled != "" {
		v, _ := strconv.ParseUint(strings.TrimPrefix(s.Throttled, "0x"), 16, 32)
		if v&0x1 != 0 {
			w.Notify.Raise("power", "temperature", notify.Warning, L("Alimentation trop faible", "Power supply too weak"),
				L("Ton Pi manque de courant : utilise l'alimentation officielle (27 W pour un Pi 5) pour éviter les plantages et les corruptions de données.",
					"Your Pi is under-powered: use the official power supply (27 W for a Pi 5) to avoid crashes and data corruption."))
		} else {
			w.Notify.Resolve("power")
		}
	}
	// Stockage
	if cfg.Disk {
		for _, d := range s.Disks {
			if d.Total == 0 {
				continue
			}
			p := int(d.Used * 100 / d.Total)
			key := "disk:" + d.Path
			switch {
			case p >= 98:
				w.Notify.Raise(key+":full", "disk", notify.Critical, L("Stockage plein", "Storage full"),
					fmt.Sprintf(L("%s est plein à %d %% : les applications risquent de ne plus fonctionner.", "%s is %d%% full: apps may stop working."), d.Path, p))
			case p >= cfg.DiskWarn:
				w.Notify.Raise(key, "disk", notify.Warning, L("Stockage presque plein", "Storage almost full"),
					fmt.Sprintf(L("%s est rempli à %d %%. Libère de la place ou ajoute un disque.", "%s is %d%% full. Free some space or add a disk."), d.Path, p))
			case p < cfg.DiskWarn-2:
				w.Notify.Resolve(key)
				w.Notify.Resolve(key + ":full")
			}
		}
	}
	// Mémoire (alerte seulement si ça dure 3 vérifications)
	if cfg.Memory && s.MemTotal > 0 {
		p := int(s.MemUsed * 100 / s.MemTotal)
		if p >= cfg.MemWarn {
			w.memHigh++
		} else {
			w.memHigh = 0
		}
		if w.memHigh >= 3 {
			w.Notify.Raise("memory", "memory", notify.Warning, L("Mémoire saturée", "Memory full"),
				fmt.Sprintf(L("La mémoire est utilisée à %d %% depuis plusieurs minutes. Arrête une application gourmande pour soulager ton Pi.", "Memory has been %d%% used for several minutes. Stop a heavy app to relieve your Pi."), p))
		} else if p < cfg.MemWarn-5 {
			w.Notify.Resolve("memory")
		}
	}
	// Applications arrêtées sans qu'on l'ait demandé
	if cfg.Apps && w.Store != nil {
		w.checkApps(L)
	}
	// Mise à jour d'Azmo OS disponible
	if cfg.Updates && w.Updater != nil {
		if info := w.Updater.Info(); info.Available && info.Latest != nil {
			w.Notify.Raise("update:"+info.Latest.Version, "updates", notify.Info, L("Mise à jour disponible", "Update available"),
				fmt.Sprintf(L("Azmo OS %s est disponible. Installe-la depuis Mises à jour.", "Azmo OS %s is available. Install it from Updates."), info.Latest.Version))
		}
	}
}

// busy indique si une opération récente explique l'arrêt de l'app (arrêt, mise à jour…).
func (w *Watcher) busy(app string) bool {
	for _, j := range w.Jobs.List() {
		if j.Kind == "app" && j.Target == app && (j.Status == "running" || time.Since(j.End) < 3*time.Minute) {
			return true
		}
	}
	return false
}

func (w *Watcher) checkApps(L func(string, string) string) {
	list, err := w.Docker.List()
	if err != nil {
		return
	}
	names := map[string]string{}
	for _, a := range w.Store.Installed() {
		names[a.ID] = a.Name
	}
	seen := map[string]bool{}
	for _, c := range list {
		if !strings.HasPrefix(c.Project, store.ProjectPrefix) {
			continue
		}
		app := strings.TrimPrefix(c.Project, store.ProjectPrefix)
		name := names[app]
		if name == "" {
			name = app
		}
		seen[c.ID] = true
		prev := w.prevState[c.ID]
		w.prevState[c.ID] = c.State
		key := "app:" + app
		switch {
		case c.State == "running":
			w.restarts[c.ID] = 0
			w.Notify.Resolve(key)
			w.Notify.Resolve(key + ":loop")
		case c.State == "restarting":
			w.restarts[c.ID]++
			if w.restarts[c.ID] >= 2 && !w.busy(app) {
				w.Notify.Raise(key+":loop", "apps", notify.Critical, fmt.Sprintf(L("« %s » plante en boucle", "“%s” keeps crashing"), name),
					L("L'application redémarre sans arrêt. Ouvre sa fiche dans Mes applications pour voir ses journaux.", "The app keeps restarting. Open it in My apps to see its logs."))
			}
		case prev == "running" && !w.busy(app):
			w.Notify.Raise(key, "apps", notify.Warning, fmt.Sprintf(L("« %s » s'est arrêtée", "“%s” has stopped"), name),
				fmt.Sprintf(L("Le conteneur %s s'est arrêté tout seul (%s). Redémarre-la depuis Mes applications.", "Container %s stopped on its own (%s). Restart it from My apps."), c.Name, c.Status))
		}
	}
	for id := range w.prevState {
		if !seen[id] {
			delete(w.prevState, id)
			delete(w.restarts, id)
		}
	}
}
