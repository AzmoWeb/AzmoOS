package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/events"
)

func (s *Server) routesAdmin(handle routeFn) {
	handle("GET /api/users", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"users": s.Auth.Users(), "perms": auth.AllPerms})
	})
	handle("POST /api/users", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username, Display, Password, Role string
			Perms                             []string
		}
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.CreateUser(req.Username, req.Display, req.Password, req.Role, req.Perms); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Compte créé : "+normalize(req.Username)+" ("+req.Role+")")
		ok(w)
	})
	handle("PUT /api/users/{name}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Display, Role string
			Perms         []string
		}
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.UpdateUser(r.PathValue("name"), req.Display, req.Role, req.Perms); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Compte modifié : "+r.PathValue("name"))
		ok(w)
	})
	handle("POST /api/users/{name}/password", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Password string }
		if !decode(w, r, &req) {
			return
		}
		if err := s.Auth.SetPassword(r.PathValue("name"), req.Password, ""); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "security", "Mot de passe réinitialisé pour "+r.PathValue("name"))
		ok(w)
	})
	handle("DELETE /api/users/{name}", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		if auth.From(r.Context()).Username == r.PathValue("name") {
			fail(w, 400, errString("tu ne peux pas supprimer ton propre compte"))
			return
		}
		if err := s.Auth.DeleteUser(r.PathValue("name")); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Warning, "security", "Compte supprimé : "+r.PathValue("name"))
		ok(w)
	})

	handle("GET /api/events", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 2000 {
			limit = 300
		}
		writeJSON(w, 200, s.Events.List(r.URL.Query().Get("category"), limit))
	})

	handle("GET /api/backup", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		withData := r.URL.Query().Get("appdata") == "1"
		name := fmt.Sprintf("azmoos-sauvegarde-%s.tar.gz", time.Now().Format("2006-01-02-1504"))
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		s.log(r, events.Info, "system", map[bool]string{true: "Sauvegarde complète téléchargée", false: "Sauvegarde de la configuration téléchargée"}[withData])
		if err := s.Backup.Write(w, withData); err != nil {
			s.Events.Add(events.Error, "system", "", "", "Échec de la sauvegarde : "+err.Error())
		}
	})
	handle("POST /api/backup/restore", auth.PermAdmin, func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			fail(w, 400, err)
			return
		}
		part, err := mr.NextPart()
		if err != nil {
			fail(w, 400, errString("fichier manquant"))
			return
		}
		man, err := s.Backup.Restore(part)
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
		s.log(r, events.Warning, "system", fmt.Sprintf("Sauvegarde du %s restaurée (%d apps)", time.Unix(man.Created, 0).Format("02/01/2006 15:04"), len(man.Apps)))
		writeJSON(w, 200, man)
	})
}
