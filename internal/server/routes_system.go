package server

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/system"
)

var (
	hostRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	tzRe   = regexp.MustCompile(`^[A-Za-z_]+(/[A-Za-z0-9_+-]+){0,2}$`)
)

func (s *Server) routesSystem(handle routeFn) {
	handle("GET /api/system", "", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Monitor.Snapshot()) })
	handle("GET /api/system/history", "", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Monitor.History()) })
	handle("GET /api/hardware", "", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Hardware) })
	handle("GET /api/jobs", "", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Jobs.List()) })

	handle("POST /api/system/power", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Action string }
		if !decode(w, r, &req) {
			return
		}
		if err := system.Power(req.Action); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "system", map[string]string{"reboot": "Redémarrage demandé", "poweroff": "Arrêt demandé"}[req.Action])
		ok(w)
	})
	handle("POST /api/system/hostname", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Hostname string }
		if !decode(w, r, &req) {
			return
		}
		if !hostRe.MatchString(req.Hostname) {
			fail(w, 400, errString("nom invalide : lettres, chiffres et tirets uniquement"))
			return
		}
		if err := system.SetHostname(req.Hostname); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Info, "system", "Nom de la machine changé en « "+req.Hostname+" »")
		ok(w)
	})
	handle("GET /api/system/settings", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"timezone": system.Timezone(), "ssh": system.Service(system.SSHService())})
	})
	handle("POST /api/system/timezone", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Timezone string }
		if !decode(w, r, &req) {
			return
		}
		if !tzRe.MatchString(req.Timezone) {
			fail(w, 400, errString("fuseau horaire invalide"))
			return
		}
		if err := system.SetTimezone(req.Timezone); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Info, "system", "Fuseau horaire : "+req.Timezone)
		ok(w)
	})
	handle("POST /api/system/ssh", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Enabled bool }
		if !decode(w, r, &req) {
			return
		}
		if err := system.SetService(system.SSHService(), req.Enabled); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Warning, "security", map[bool]string{true: "Accès SSH activé", false: "Accès SSH désactivé"}[req.Enabled])
		ok(w)
	})

	// Mises à jour
	handle("GET /api/updates", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"azmoos": s.Updater.Info(), "packages": system.Upgradable(), "rebootRequired": system.RebootRequired(),
		})
	})
	handle("POST /api/updates/check", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		info := s.Updater.Check()
		_ = s.Jobs.Start("system", "apt", "refresh", system.AptRefreshSteps(), nil)
		writeJSON(w, 200, info)
	})
	handle("POST /api/updates/system", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Jobs.Start("system", "apt", "upgrade", system.AptUpgradeSteps(), nil); err != nil {
			fail(w, 409, err)
			return
		}
		s.log(r, events.Info, "system", "Mise à jour du système lancée")
		ok(w)
	})
	handle("POST /api/updates/azmoos", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Updater.Apply(restartSoon); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "system", fmt.Sprintf("Mise à jour d'Azmo OS lancée (depuis %s)", s.Version))
		ok(w)
	})
}
