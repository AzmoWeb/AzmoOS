package server

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/backupsched"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/terminal"
)

// Réglages stockés dans config.Settings.
const (
	SettingHTTPS         = "https"          // « on » : serveur HTTPS actif
	SettingHTTPSRedirect = "https_redirect" // « on » : redirige HTTP vers HTTPS
	SettingTerminal      = "terminal"       // « off » : terminal web désactivé
)

func (s *Server) setting(k string) string { return s.Config.Setting(k) }

func (s *Server) setSetting(k, v string) error {
	return s.Config.Update(func(c *config.Config) error { c.Settings[k] = v; return nil })
}

// HTTPSPort renvoie le port HTTPS configuré.
func (s *Server) HTTPSPort() int {
	_, p, _ := net.SplitHostPort(s.HTTPSAddr)
	n, _ := strconv.Atoi(p)
	if n == 0 {
		n = 443
	}
	return n
}

// StartHTTPS démarre le serveur HTTPS si le réglage est actif.
func (s *Server) StartHTTPS(h http.Handler) error {
	if s.setting(SettingHTTPS) != "on" {
		return nil
	}
	return s.TLS.Start(s.HTTPSAddr, h)
}

// Redirect enveloppe le serveur HTTP : redirection vers HTTPS si demandé.
// Le certificat de l'autorité reste téléchargeable en HTTP (pour l'installer avant).
func (s *Server) Redirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil && s.TLS.Running() && s.setting(SettingHTTPSRedirect) == "on" && r.URL.Path != "/api/https/ca.crt" {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if p := s.HTTPSPort(); p != 443 {
				host = net.JoinHostPort(host, strconv.Itoa(p))
			}
			http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusPermanentRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) routesV3(handle routeFn) {
	// ---------- Terminal web ----------
	handle("GET /api/terminal/info", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"enabled": s.setting(SettingTerminal) != "off", "account": terminal.DefaultAccount()})
	})
	handle("POST /api/terminal/settings", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Enabled bool }
		if !decode(w, r, &req) {
			return
		}
		v := "on"
		if !req.Enabled {
			v = "off"
		}
		if err := s.setSetting(SettingTerminal, v); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Warning, "security", map[bool]string{true: "Terminal web activé", false: "Terminal web désactivé"}[req.Enabled])
		ok(w)
	})
	handle("GET /api/terminal", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if s.setting(SettingTerminal) == "off" {
			fail(w, 403, errString("le terminal web est désactivé (Sécurité)"))
			return
		}
		if !terminal.SameOrigin(r) {
			fail(w, 403, errString("origine refusée"))
			return
		}
		ws, err := terminal.Upgrade(w, r)
		if err != nil {
			return
		}
		acc := terminal.DefaultAccount()
		s.log(r, events.Warning, "security", "Terminal web ouvert (utilisateur système "+acc.Name+")")
		_ = s.Terminal.Serve(ws, acc)
		s.log(r, events.Info, "security", "Terminal web fermé")
	})

	// ---------- HTTPS local ----------
	handle("GET /api/https/ca.crt", "public", func(w http.ResponseWriter, r *http.Request) {
		pem, err := s.TLS.CAPEM()
		if err != nil {
			fail(w, 500, err)
			return
		}
		host, _ := os.Hostname()
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
		w.Header().Set("Content-Disposition", `attachment; filename="azmoos-`+host+`.crt"`)
		_, _ = w.Write(pem)
	})
	handle("GET /api/https", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if s.TLS.Info().CAFingerprint == "" {
			_ = s.TLS.Ensure()
		}
		writeJSON(w, 200, map[string]any{
			"enabled": s.setting(SettingHTTPS) == "on", "redirect": s.setting(SettingHTTPSRedirect) == "on",
			"port": s.HTTPSPort(), "secure": r.TLS != nil, "cert": s.TLS.Info(),
		})
	})
	handle("POST /api/https", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Enabled, Redirect bool }
		if !decode(w, r, &req) {
			return
		}
		if req.Enabled {
			if err := s.TLS.Start(s.HTTPSAddr, s.HTTPSHandler); err != nil {
				fail(w, 400, err)
				return
			}
		} else {
			req.Redirect = false
			// Arrêt différé : la réponse doit partir si on est connecté en HTTPS.
			defer func() { go s.TLS.Stop() }()
		}
		_ = s.setSetting(SettingHTTPS, map[bool]string{true: "on", false: ""}[req.Enabled])
		_ = s.setSetting(SettingHTTPSRedirect, map[bool]string{true: "on", false: ""}[req.Redirect])
		s.log(r, events.Warning, "security", map[bool]string{true: "HTTPS local activé", false: "HTTPS local désactivé"}[req.Enabled])
		ok(w)
	})
	handle("POST /api/https/regenerate", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.TLS.Regenerate(); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Info, "security", "Certificat HTTPS régénéré")
		writeJSON(w, 200, s.TLS.Info())
	})

	// ---------- Notifications ----------
	handle("GET /api/notifications", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		items, unread := s.Notify.List()
		writeJSON(w, 200, map[string]any{"items": items, "unread": unread})
	})
	handle("POST /api/notifications/read", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ ID int64 }
		if !decode(w, r, &req) {
			return
		}
		s.Notify.MarkRead(req.ID)
		ok(w)
	})
	handle("POST /api/notifications/clear", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		s.Notify.Clear()
		ok(w)
	})
	handle("GET /api/notifications/settings", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, s.Notify.Settings())
	})
	handle("POST /api/notifications/settings", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var n config.Notify
		if !decode(w, r, &n) {
			return
		}
		if n.TempWarn < 50 || n.TempWarn > 90 || n.DiskWarn < 50 || n.DiskWarn > 99 || n.MemWarn < 50 || n.MemWarn > 99 {
			fail(w, 400, errString("seuil d'alerte hors limites"))
			return
		}
		if n.PushLevel != "info" && n.PushLevel != "warning" && n.PushLevel != "critical" {
			n.PushLevel = "warning"
		}
		if n.Lang != "en" {
			n.Lang = "fr"
		}
		n.Ntfy.Server = strings.TrimSpace(n.Ntfy.Server)
		if n.Ntfy.Server != "" && !strings.HasPrefix(n.Ntfy.Server, "https://") && !strings.HasPrefix(n.Ntfy.Server, "http://") {
			fail(w, 400, errString("adresse du serveur ntfy invalide"))
			return
		}
		if err := s.Config.Update(func(c *config.Config) error { c.Notify = &n; return nil }); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Info, "system", "Réglages des notifications modifiés")
		ok(w)
	})
	handle("POST /api/notifications/test", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, s.Notify.Test())
	})

	// ---------- Sauvegardes planifiées ----------
	handle("GET /api/backups/schedules", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		installed, remotes := s.Sched.RcloneInfo()
		type view struct {
			config.Backup
			Next int64 `json:"next"`
		}
		out := []view{}
		for _, b := range s.Sched.List() {
			v := view{Backup: b}
			if b.Enabled {
				v.Next = backupsched.Next(b, timeNow()).Unix()
			}
			out = append(out, v)
		}
		writeJSON(w, 200, map[string]any{"schedules": out, "rclone": map[string]any{"installed": installed, "remotes": remotes, "config": s.Sched.RcloneConfig}})
	})
	handle("POST /api/backups/schedules", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var b config.Backup
		if !decode(w, r, &b) {
			return
		}
		saved, err := s.Sched.Save(b)
		if err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "system", "Sauvegarde planifiée « "+saved.Name+" » enregistrée")
		writeJSON(w, 200, saved)
	})
	handle("DELETE /api/backups/schedules/{id}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Sched.Delete(r.PathValue("id")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "system", "Sauvegarde planifiée supprimée")
		ok(w)
	})
	handle("POST /api/backups/schedules/{id}/run", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Sched.Run(r.PathValue("id"), auth.From(r.Context()).Username); err != nil {
			fail(w, 409, err)
			return
		}
		ok(w)
	})
	handle("GET /api/backups/schedules/{id}/files", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		files, err := s.Sched.Files(r.PathValue("id"))
		if err != nil {
			fail(w, 400, err)
			return
		}
		writeJSON(w, 200, files)
	})
	handle("POST /api/backups/schedules/{id}/restore", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ File string }
		if !decode(w, r, &req) {
			return
		}
		f, cleanup, err := s.Sched.Open(r.PathValue("id"), req.File)
		if err != nil {
			fail(w, 400, err)
			return
		}
		defer cleanup()
		defer f.Close()
		man, err := s.Backup.Restore(f)
		if err != nil {
			fail(w, 400, err)
			return
		}
		if err := s.Config.Reload(); err != nil {
			fail(w, 500, err)
			return
		}
		s.Store.Redeploy()
		_ = s.Shares.Apply()
		s.log(r, events.Warning, "system", "Sauvegarde restaurée : "+req.File)
		writeJSON(w, 200, man)
	})
}
