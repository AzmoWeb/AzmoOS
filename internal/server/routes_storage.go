package server

import (
	"net/http"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
)

func (s *Server) routesStorage(handle routeFn) {
	handle("GET /api/storage/disks", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		d, err := s.Storage.List()
		if err != nil {
			fail(w, 500, err)
			return
		}
		writeJSON(w, 200, d)
	})
	handle("POST /api/storage/{op}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path, FSType, Label string
			Enabled             bool
		}
		if !decode(w, r, &req) {
			return
		}
		var err error
		switch r.PathValue("op") {
		case "mount":
			var mp string
			if mp, err = s.Storage.Mount(req.Path); err == nil {
				s.log(r, events.Info, "storage", req.Path+" monté dans "+mp)
			}
		case "unmount":
			if err = s.Storage.Unmount(req.Path); err == nil {
				s.log(r, events.Info, "storage", req.Path+" démonté")
			}
		case "automount":
			if err = s.Storage.SetAutoMount(req.Path, req.Enabled); err == nil {
				s.log(r, events.Info, "storage", req.Path+map[bool]string{true: " : montage automatique activé", false: " : montage automatique désactivé"}[req.Enabled])
			}
		case "format":
			if err = s.Storage.Format(req.Path, req.FSType, req.Label); err == nil {
				s.log(r, events.Warning, "storage", "Formatage de "+req.Path+" en "+req.FSType)
			}
		default:
			err = errString("opération inconnue")
		}
		if err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})

	handle("GET /api/shares", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": s.Shares.Status(), "shares": s.Shares.List()})
	})
	handle("POST /api/shares", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var sh config.Share
		if !decode(w, r, &sh) {
			return
		}
		if err := s.Shares.Save(sh); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "storage", "Partage réseau « "+sh.Name+" » enregistré")
		ok(w)
	})
	handle("DELETE /api/shares/{name}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Shares.Delete(r.PathValue("name")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "storage", "Partage réseau « "+r.PathValue("name")+" » supprimé")
		ok(w)
	})
	handle("POST /api/shares/users", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Username, Password string }
		if !decode(w, r, &req) {
			return
		}
		if err := auth.ValidateName(req.Username); err != nil {
			fail(w, 400, err)
			return
		}
		if err := s.Shares.SetPassword(req.Username, req.Password); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Accès réseau (SMB) activé pour "+req.Username)
		ok(w)
	})
	handle("DELETE /api/shares/users/{name}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if err := s.Shares.RemoveUser(r.PathValue("name")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Accès réseau (SMB) retiré pour "+r.PathValue("name"))
		ok(w)
	})
}
