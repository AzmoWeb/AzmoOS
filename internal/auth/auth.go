// Package auth gère le compte admin et les sessions.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/config"
)

const (
	CookieName = "azmo_session"
	iterations = 210000
	sessionTTL = 30 * 24 * time.Hour
)

var ErrInvalid = errors.New("identifiants invalides")

// Auth fournit l'authentification.
type Auth struct{ cfg *config.Store }

func New(cfg *config.Store) *Auth { return &Auth{cfg: cfg} }

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hash(password, salt string) string {
	k, _ := pbkdf2.Key(sha256.New, password, []byte(salt), iterations, 32)
	return hex.EncodeToString(k)
}

// Initialized indique si un compte existe.
func (a *Auth) Initialized() (ok bool) {
	a.cfg.View(func(c *config.Config) { ok = c.User != nil })
	return
}

// SetUser crée ou remplace le compte admin.
func (a *Auth) SetUser(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < 6 {
		return errors.New("nom requis et mot de passe de 6 caractères minimum")
	}
	salt := randHex(16)
	return a.cfg.Update(func(c *config.Config) {
		c.User = &config.User{Username: username, Salt: salt, Hash: hash(password, salt)}
		c.Sessions = map[string]int64{} // invalide les anciennes sessions
	})
}

// Check vérifie les identifiants.
func (a *Auth) Check(username, password string) bool {
	var u *config.User
	a.cfg.View(func(c *config.Config) {
		if c.User != nil {
			cp := *c.User
			u = &cp
		}
	})
	if u == nil {
		return false
	}
	h := hash(password, u.Salt)
	return subtle.ConstantTimeCompare([]byte(u.Username), []byte(username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(u.Hash), []byte(h)) == 1
}

// Username renvoie le nom du compte.
func (a *Auth) Username() (n string) {
	a.cfg.View(func(c *config.Config) {
		if c.User != nil {
			n = c.User.Username
		}
	})
	return
}

// NewSession crée une session et pose le cookie.
func (a *Auth) NewSession(w http.ResponseWriter, r *http.Request) error {
	tok := randHex(32)
	exp := time.Now().Add(sessionTTL)
	err := a.cfg.Update(func(c *config.Config) {
		now := time.Now().Unix()
		for k, v := range c.Sessions {
			if v < now {
				delete(c.Sessions, k)
			}
		}
		c.Sessions[tok] = exp.Unix()
	})
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: tok, Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil,
	})
	return nil
}

// Logout supprime la session courante.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_ = a.cfg.Update(func(cfg *config.Config) { delete(cfg.Sessions, c.Value) })
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1})
}

// Valid indique si la requête porte une session valide.
func (a *Auth) Valid(r *http.Request) (ok bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}
	a.cfg.View(func(cfg *config.Config) {
		exp, found := cfg.Sessions[c.Value]
		ok = found && exp > time.Now().Unix()
	})
	return
}

// Require protège un handler.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Valid(r) {
			http.Error(w, `{"error":"non authentifié"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
