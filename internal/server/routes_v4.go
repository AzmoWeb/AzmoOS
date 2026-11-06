package server

import (
	"net/http"
	"slices"
	"time"

	"github.com/azmoweb/azmoos/internal/appupdate"
	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/config"
)

// Widgets du bureau et mises à jour automatiques des applications.
func (s *Server) routesV4(handle routeFn) {
	handle("PUT /api/me/widgets", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Widgets []string `json:"widgets"`
		}
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.SetWidgets(auth.From(r.Context()).Username, req.Widgets); err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})
	handle("GET /api/me/note", "", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"text": s.Auth.Note(auth.From(r.Context()).Username)})
	})
	handle("PUT /api/me/note", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.SetNote(auth.From(r.Context()).Username, req.Text); err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})

	if s.AppUpd == nil {
		return
	}
	handle("GET /api/app-updates", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		cfg := s.AppUpd.Settings()
		running, current := s.AppUpd.Status()
		type appRow struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Icon    string `json:"icon"`
			Enabled bool   `json:"enabled"`
		}
		apps := []appRow{}
		for _, a := range s.Store.Installed() {
			apps = append(apps, appRow{ID: a.ID, Name: a.Name, Icon: a.Icon, Enabled: !slices.Contains(cfg.Exclude, a.ID)})
		}
		var next int64
		if cfg.Enabled {
			next = appupdate.Next(cfg, time.Now()).Unix()
		}
		writeJSON(w, 200, map[string]any{"settings": cfg, "apps": apps, "next": next, "running": running, "current": current})
	})
	handle("PUT /api/app-updates", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		var req config.AppUpdates
		if !decode(w, r, &req) {
			return
		}
		if err := s.AppUpd.Save(req); err != nil {
			fail(w, 400, err)
			return
		}
		if req.Enabled {
			s.log(r, "info", "apps", "Mises à jour automatiques des applications activées")
		} else {
			s.log(r, "info", "apps", "Mises à jour automatiques des applications désactivées")
		}
		ok(w)
	})
	handle("POST /api/app-updates/run", auth.PermAppsManage, func(w http.ResponseWriter, r *http.Request) {
		if err := s.AppUpd.Start(auth.From(r.Context()).Username); err != nil {
			fail(w, 409, err)
			return
		}
		ok(w)
	})
}
