package server

import (
	"net/http"
	"os"

	"github.com/azmoweb/azmoos/internal/auth"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
)

func (s *Server) routesAuth(handle routeFn) {
	handle("GET /api/status", "public", s.status)
	handle("POST /api/setup", "public", s.setup)
	handle("POST /api/login", "public", s.login)
	handle("POST /api/logout", "public", func(w http.ResponseWriter, r *http.Request) {
		if id := s.Auth.Identify(r); id != nil {
			s.Events.Add(events.Info, "auth", id.Username, auth.ClientIP(r), "Déconnexion")
		}
		s.Auth.Logout(w, r)
		ok(w)
	})

	handle("GET /api/me", "", func(w http.ResponseWriter, r *http.Request) {
		id := auth.From(r.Context())
		writeJSON(w, 200, map[string]any{"user": id.PublicUser, "sessions": s.Auth.Sessions(id.Username, id.Token)})
	})
	handle("POST /api/me/profile", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Display string }
		if !decode(w, r, &req) {
			return
		}
		id := auth.From(r.Context())
		if err := s.Auth.UpdateUser(id.Username, req.Display, id.Role, id.Perms); err != nil {
			fail(w, 400, err)
			return
		}
		ok(w)
	})
	handle("POST /api/me/password", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Current, Password string }
		if !decode(w, r, &req) {
			return
		}
		id := auth.From(r.Context())
		if !s.Auth.Check(id.Username, req.Current) {
			fail(w, 403, errString("ton mot de passe actuel est incorrect"))
			return
		}
		if err := s.Auth.SetPassword(id.Username, req.Password, id.Token); err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Mot de passe modifié")
		ok(w)
	})
	handle("POST /api/me/sessions/revoke", "", func(w http.ResponseWriter, r *http.Request) {
		id := auth.From(r.Context())
		if err := s.Auth.RevokeOthers(id.Username, id.Token); err != nil {
			fail(w, 500, err)
			return
		}
		s.log(r, events.Info, "security", "Autres sessions fermées")
		ok(w)
	})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	res := map[string]any{
		"product": "Azmo OS", "version": s.Version, "hostname": host,
		"initialized": s.Auth.Initialized(), "authenticated": false,
	}
	if id := s.Auth.Identify(r); id != nil {
		res["authenticated"] = true
		res["user"] = id.PublicUser
		res["docker"] = s.Docker.Available()
		res["hardware"] = s.Hardware
	}
	writeJSON(w, 200, res)
}

type creds struct {
	Username, Password, Display string
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if s.Auth.Initialized() {
		fail(w, 409, errString("Azmo OS est déjà configuré"))
		return
	}
	var c creds
	if !decode(w, r, &c) {
		return
	}
	if err := s.Auth.CreateUser(c.Username, c.Display, c.Password, config.RoleAdmin, nil); err != nil {
		fail(w, 400, err)
		return
	}
	_ = s.Auth.NewSession(w, r, c.Username)
	s.Events.Add(events.Info, "auth", c.Username, auth.ClientIP(r), "Premier démarrage : compte administrateur créé")
	ok(w)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	ip := auth.ClientIP(r)
	if !s.Auth.Allowed(ip) {
		fail(w, 429, auth.ErrLocked)
		return
	}
	var c creds
	if !decode(w, r, &c) {
		return
	}
	if !s.Auth.Check(c.Username, c.Password) {
		if s.Auth.Failed(ip) {
			s.Events.Add(events.Warning, "security", c.Username, ip, "Adresse bloquée 5 minutes après trop d'échecs de connexion")
		} else {
			s.Events.Add(events.Warning, "auth", c.Username, ip, "Échec de connexion")
		}
		fail(w, 401, auth.ErrInvalid)
		return
	}
	s.Auth.Succeeded(ip)
	if err := s.Auth.NewSession(w, r, normalize(c.Username)); err != nil {
		fail(w, 500, err)
		return
	}
	s.Events.Add(events.Info, "auth", normalize(c.Username), ip, "Connexion réussie")
	ok(w)
}
