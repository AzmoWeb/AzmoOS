package server

import (
	"net/http"
	"os"
	"strings"

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
	handle("POST /api/me/lang", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Lang string }
		if !decode(w, r, &req) {
			return
		}
		id := auth.From(r.Context())
		if err := s.Auth.SetLang(id.Username, req.Lang); err != nil {
			fail(w, 400, err)
			return
		}
		// Les alertes envoyées sur le téléphone suivent la langue de l'administrateur.
		if id.Can(auth.PermAdmin) {
			_ = s.Config.Update(func(c *config.Config) error { c.Notify.Lang = req.Lang; return nil })
		}
		ok(w)
	})
	handle("POST /api/me/2fa/begin", "", func(w http.ResponseWriter, r *http.Request) {
		host, _ := os.Hostname()
		secret, uri, err := s.Auth.BeginTOTP(auth.From(r.Context()).Username, host)
		if err != nil {
			fail(w, 500, err)
			return
		}
		writeJSON(w, 200, map[string]string{"secret": secret, "uri": uri})
	})
	handle("POST /api/me/2fa/confirm", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Code string }
		if !decode(w, r, &req) {
			return
		}
		codes, err := s.Auth.ConfirmTOTP(auth.From(r.Context()).Username, req.Code)
		if err != nil {
			fail(w, 400, err)
			return
		}
		s.log(r, events.Info, "security", "Double authentification activée")
		writeJSON(w, 200, map[string]any{"recovery": codes})
	})
	handle("POST /api/me/2fa/disable", "", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Password, Code string }
		if !decode(w, r, &req) {
			return
		}
		id := auth.From(r.Context())
		if !s.Auth.Check(id.Username, req.Password) {
			fail(w, 403, errString("mot de passe incorrect"))
			return
		}
		if ok, _ := s.Auth.VerifySecondFactor(id.Username, req.Code); !ok {
			fail(w, 403, errString("code incorrect ou expiré"))
			return
		}
		_ = s.Auth.DisableTOTP(id.Username)
		s.log(r, events.Warning, "security", "Double authentification désactivée")
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
	Username, Password, Display, Code string
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
			s.securityAlert("lock:"+ip, "Tentatives de connexion suspectes", "Suspicious login attempts",
				"L'adresse "+ip+" a été bloquée 5 minutes après 8 mots de passe incorrects (compte « "+c.Username+" »).",
				"Address "+ip+" was blocked for 5 minutes after 8 wrong passwords (account “"+c.Username+"”).")
		} else {
			s.Events.Add(events.Warning, "auth", c.Username, ip, "Échec de connexion")
		}
		fail(w, 401, auth.ErrInvalid)
		return
	}
	user := normalize(c.Username)
	if s.Auth.TOTPEnabled(user) {
		if strings.TrimSpace(c.Code) == "" {
			// Mot de passe correct : on demande le second facteur.
			writeJSON(w, 200, map[string]any{"ok": false, "totp": true})
			return
		}
		ok, recovery := s.Auth.VerifySecondFactor(user, c.Code)
		if !ok {
			s.Auth.Failed(ip)
			s.Events.Add(events.Warning, "auth", user, ip, "Code de double authentification incorrect")
			fail(w, 401, errString("code incorrect ou expiré"))
			return
		}
		if recovery {
			s.Events.Add(events.Warning, "security", user, ip, "Connexion avec un code de secours")
			s.securityAlert("", "Code de secours utilisé", "Recovery code used",
				"Le compte « "+user+" » s'est connecté avec un code de secours depuis "+ip+". Si ce n'est pas toi, change ton mot de passe.",
				"Account “"+user+"” signed in with a recovery code from "+ip+". If it wasn't you, change your password.")
		}
	}
	s.Auth.Succeeded(ip)
	if s.Notify != nil {
		s.Notify.Resolve("lock:" + ip)
	}
	if err := s.Auth.NewSession(w, r, user); err != nil {
		fail(w, 500, err)
		return
	}
	s.Events.Add(events.Info, "auth", user, ip, "Connexion réussie")
	ok(w)
}

// securityAlert envoie une alerte de sécurité si cette surveillance est active.
func (s *Server) securityAlert(key, titleFR, titleEN, msgFR, msgEN string) {
	if s.Notify == nil {
		return
	}
	st := s.Notify.Settings()
	if !st.Security {
		return
	}
	title, msg := titleFR, msgFR
	if st.Lang == "en" {
		title, msg = titleEN, msgEN
	}
	s.Notify.Raise(key, "security", "warning", title, msg)
}
