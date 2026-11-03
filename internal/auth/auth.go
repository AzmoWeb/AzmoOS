// Package auth gère les comptes, les rôles/permissions et les sessions.
package auth

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/totp"
)

const (
	CookieName = "azmo_session"
	iterations = 210000
	sessionTTL = 30 * 24 * time.Hour
)

// Permissions accordables à un membre (un admin les a toutes).
const (
	PermAppsManage = "apps.manage"
	PermFilesRead  = "files.read"
	PermFilesWrite = "files.write"
	PermAdmin      = "admin" // pseudo-permission : rôle admin requis
)

// AllPerms liste les permissions configurables.
var AllPerms = []string{PermAppsManage, PermFilesRead, PermFilesWrite}

var (
	ErrInvalid = errors.New("nom d'utilisateur ou mot de passe incorrect")
	ErrLocked  = errors.New("trop de tentatives : réessaie dans quelques minutes")
	nameRe     = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)
)

// Auth fournit l'authentification.
type Auth struct {
	cfg  *config.Store
	mu   sync.Mutex
	fail map[string]*attempts
}

type attempts struct {
	n     int
	until time.Time
}

func New(cfg *config.Store) *Auth { return &Auth{cfg: cfg, fail: map[string]*attempts{}} }

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hash(password, salt string) string {
	k, _ := pbkdf2.Key(sha256.New, password, []byte(salt), iterations, 32)
	return hex.EncodeToString(k)
}

// PublicUser est la vue d'un compte sans secrets.
type PublicUser struct {
	Username     string   `json:"username"`
	Display      string   `json:"display"`
	Role         string   `json:"role"`
	Perms        []string `json:"perms"`
	Created      int64    `json:"created"`
	LastSeen     int64    `json:"lastSeen"`
	Lang         string   `json:"lang"`
	TOTP         bool     `json:"totp"`
	RecoveryLeft int      `json:"recoveryLeft"`
}

func public(u *config.User) PublicUser {
	p := PublicUser{Username: u.Username, Display: u.Display, Role: u.Role, Perms: u.Perms, Created: u.Created, LastSeen: u.LastSeen,
		Lang: u.Lang, TOTP: u.TOTPSecret != "", RecoveryLeft: len(u.Recovery)}
	if p.Display == "" {
		p.Display = u.Username
	}
	if u.Role == config.RoleAdmin {
		p.Perms = append([]string{PermAdmin}, AllPerms...)
	}
	if p.Perms == nil {
		p.Perms = []string{}
	}
	return p
}

// Initialized indique si au moins un compte existe.
func (a *Auth) Initialized() (ok bool) {
	a.cfg.View(func(c *config.Config) { ok = len(c.Users) > 0 })
	return
}

// ValidateName vérifie un nom d'utilisateur.
func ValidateName(n string) error {
	if !nameRe.MatchString(n) {
		return errors.New("nom d'utilisateur : 2 à 32 caractères, minuscules, chiffres, - ou _, commençant par une lettre")
	}
	return nil
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return errors.New("le mot de passe doit contenir au moins 8 caractères")
	}
	return nil
}

func cleanPerms(p []string) []string {
	out := []string{}
	for _, x := range p {
		if slices.Contains(AllPerms, x) && !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

// CreateUser ajoute un compte.
func (a *Auth) CreateUser(username, display, password, role string, perms []string) error {
	username = strings.ToLower(strings.TrimSpace(username))
	if err := ValidateName(username); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	if role != config.RoleAdmin {
		role = config.RoleMember
	}
	salt := randHex(16)
	return a.cfg.Update(func(c *config.Config) error {
		if c.FindUser(username) != nil {
			return errors.New("ce nom d'utilisateur existe déjà")
		}
		c.Users = append(c.Users, &config.User{Username: username, Display: strings.TrimSpace(display), Role: role,
			Perms: cleanPerms(perms), Salt: salt, Hash: hash(password, salt), Created: time.Now().Unix()})
		return nil
	})
}

// UpdateUser modifie le nom affiché, le rôle et les permissions.
func (a *Auth) UpdateUser(username, display, role string, perms []string) error {
	return a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil {
			return errors.New("utilisateur introuvable")
		}
		if role != config.RoleAdmin {
			role = config.RoleMember
		}
		if u.Role == config.RoleAdmin && role != config.RoleAdmin && countAdmins(c) == 1 {
			return errors.New("il faut garder au moins un administrateur")
		}
		u.Display, u.Role, u.Perms = strings.TrimSpace(display), role, cleanPerms(perms)
		return nil
	})
}

func countAdmins(c *config.Config) (n int) {
	for _, u := range c.Users {
		if u.Role == config.RoleAdmin {
			n++
		}
	}
	return
}

// DeleteUser supprime un compte et ses sessions.
func (a *Auth) DeleteUser(username string) error {
	return a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil {
			return errors.New("utilisateur introuvable")
		}
		if u.Role == config.RoleAdmin && countAdmins(c) == 1 {
			return errors.New("impossible de supprimer le dernier administrateur")
		}
		c.Users = slices.DeleteFunc(c.Users, func(x *config.User) bool { return x.Username == username })
		for k, s := range c.Sessions {
			if s.User == username {
				delete(c.Sessions, k)
			}
		}
		return nil
	})
}

// SetPassword change le mot de passe et ferme les autres sessions.
func (a *Auth) SetPassword(username, password, keepToken string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	salt := randHex(16)
	return a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil {
			return errors.New("utilisateur introuvable")
		}
		u.Salt, u.Hash = salt, hash(password, salt)
		for k, s := range c.Sessions {
			if s.User == username && k != keepToken {
				delete(c.Sessions, k)
			}
		}
		return nil
	})
}

// Users liste les comptes.
func (a *Auth) Users() []PublicUser {
	out := []PublicUser{}
	a.cfg.View(func(c *config.Config) {
		for _, u := range c.Users {
			out = append(out, public(u))
		}
	})
	return out
}

// Check vérifie les identifiants (temps constant).
func (a *Auth) Check(username, password string) bool {
	var salt, want string
	a.cfg.View(func(c *config.Config) {
		if u := c.FindUser(strings.ToLower(strings.TrimSpace(username))); u != nil {
			salt, want = u.Salt, u.Hash
		}
	})
	if want == "" {
		hash(password, "leurre") // même coût que le vrai calcul
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(hash(password, salt))) == 1
}

// ClientIP renvoie l'IP du client.
func ClientIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

// Allowed vérifie le blocage anti force brute.
func (a *Auth) Allowed(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fail[ip]
	return f == nil || time.Now().After(f.until)
}

// Failed enregistre un échec ; renvoie true si l'IP vient d'être bloquée.
func (a *Auth) Failed(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fail[ip]
	if f == nil {
		f = &attempts{}
		a.fail[ip] = f
	}
	f.n++
	if f.n >= 8 {
		f.n, f.until = 0, time.Now().Add(5*time.Minute)
		return true
	}
	return false
}

// Succeeded remet le compteur à zéro.
func (a *Auth) Succeeded(ip string) {
	a.mu.Lock()
	delete(a.fail, ip)
	a.mu.Unlock()
}

// NewSession crée une session et pose le cookie.
func (a *Auth) NewSession(w http.ResponseWriter, r *http.Request, username string) error {
	tok := randHex(32)
	exp := time.Now().Add(sessionTTL)
	err := a.cfg.Update(func(c *config.Config) error {
		now := time.Now().Unix()
		for k, v := range c.Sessions {
			if v.Expires < now {
				delete(c.Sessions, k)
			}
		}
		c.Sessions[tok] = &config.Session{User: username, Expires: exp.Unix(), IP: ClientIP(r), Agent: trunc(r.UserAgent(), 200)}
		if u := c.FindUser(username); u != nil {
			u.LastSeen = now
		}
		return nil
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

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Logout supprime la session courante.
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_ = a.cfg.Update(func(cfg *config.Config) error { delete(cfg.Sessions, c.Value); return nil })
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1})
}

type ctxKey struct{}

// Identity est l'utilisateur authentifié d'une requête.
type Identity struct {
	PublicUser
	Token string
}

// Can indique si l'utilisateur possède une permission.
func (i *Identity) Can(perm string) bool {
	if i == nil {
		return false
	}
	if i.Role == config.RoleAdmin {
		return true
	}
	return perm != PermAdmin && slices.Contains(i.Perms, perm)
}

// Identify renvoie l'identité associée à la requête, ou nil.
func (a *Auth) Identify(r *http.Request) *Identity {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil
	}
	var id *Identity
	a.cfg.View(func(cfg *config.Config) {
		s, ok := cfg.Sessions[c.Value]
		if !ok || s.Expires < time.Now().Unix() {
			return
		}
		if u := cfg.FindUser(s.User); u != nil {
			id = &Identity{PublicUser: public(u), Token: c.Value}
		}
	})
	return id
}

// From renvoie l'identité stockée dans le contexte.
func From(ctx context.Context) *Identity {
	id, _ := ctx.Value(ctxKey{}).(*Identity)
	return id
}

// Require protège un handler par une permission ("" = simplement connecté).
func (a *Auth) Require(perm string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := a.Identify(r)
		if id == nil {
			http.Error(w, `{"error":"tu n'es pas connecté"}`, http.StatusUnauthorized)
			return
		}
		if perm != "" && !id.Can(perm) {
			http.Error(w, `{"error":"tu n'as pas la permission de faire ça"}`, http.StatusForbidden)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

// Sessions liste les sessions actives d'un utilisateur.
func (a *Auth) Sessions(username, current string) []map[string]any {
	out := []map[string]any{}
	a.cfg.View(func(c *config.Config) {
		now := time.Now().Unix()
		for k, s := range c.Sessions {
			if s.User == username && s.Expires > now {
				out = append(out, map[string]any{"id": k[:12], "ip": s.IP, "agent": s.Agent, "expires": s.Expires, "current": k == current})
			}
		}
	})
	return out
}

// RevokeOthers ferme toutes les autres sessions de l'utilisateur.
func (a *Auth) RevokeOthers(username, current string) error {
	return a.cfg.Update(func(c *config.Config) error {
		for k, s := range c.Sessions {
			if s.User == username && k != current {
				delete(c.Sessions, k)
			}
		}
		return nil
	})
}

// ---------- Double authentification ----------

const issuer = "Azmo OS"

// TOTPEnabled indique si le compte a la double authentification.
func (a *Auth) TOTPEnabled(username string) (on bool) {
	a.cfg.View(func(c *config.Config) {
		if u := c.FindUser(username); u != nil {
			on = u.TOTPSecret != ""
		}
	})
	return
}

// BeginTOTP prépare l'activation : renvoie le secret et le lien otpauth:// (QR code).
func (a *Auth) BeginTOTP(username, host string) (secret, uri string, err error) {
	secret = totp.NewSecret()
	err = a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil {
			return errors.New("utilisateur introuvable")
		}
		u.TOTPPending = secret
		return nil
	})
	account := username
	if host != "" {
		account = username + "@" + host
	}
	return secret, totp.URI(secret, account, issuer), err
}

func hashRecovery(code string) string {
	code = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(code))
	h := sha256.Sum256([]byte("azmoos-recovery:" + code))
	return hex.EncodeToString(h[:])
}

// ConfirmTOTP active la double authentification si le code est bon, et renvoie les codes de secours.
func (a *Auth) ConfirmTOTP(username, code string) ([]string, error) {
	codes := totp.RecoveryCodes(8)
	err := a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil || u.TOTPPending == "" {
			return errors.New("aucune activation en cours, recommence depuis le début")
		}
		step, ok := totp.Verify(u.TOTPPending, code, time.Now(), 0)
		if !ok {
			return errors.New("code incorrect : vérifie l'heure de ton téléphone et réessaie")
		}
		u.TOTPSecret, u.TOTPPending, u.TOTPLastStep = u.TOTPPending, "", step
		u.Recovery = nil
		for _, rc := range codes {
			u.Recovery = append(u.Recovery, hashRecovery(rc))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// DisableTOTP désactive la double authentification.
func (a *Auth) DisableTOTP(username string) error {
	return a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil {
			return errors.New("utilisateur introuvable")
		}
		u.TOTPSecret, u.TOTPPending, u.TOTPLastStep, u.Recovery = "", "", 0, nil
		return nil
	})
}

// VerifySecondFactor vérifie un code TOTP ou un code de secours (consommé).
// Renvoie (ok, codeDeSecoursUtilisé).
func (a *Auth) VerifySecondFactor(username, code string) (ok, recovery bool) {
	_ = a.cfg.Update(func(c *config.Config) error {
		u := c.FindUser(username)
		if u == nil || u.TOTPSecret == "" {
			return nil
		}
		if step, good := totp.Verify(u.TOTPSecret, code, time.Now(), u.TOTPLastStep); good {
			u.TOTPLastStep, ok = step, true
			return nil
		}
		h := hashRecovery(code)
		for i, r := range u.Recovery {
			if subtle.ConstantTimeCompare([]byte(r), []byte(h)) == 1 {
				u.Recovery = append(u.Recovery[:i], u.Recovery[i+1:]...)
				ok, recovery = true, true
				return nil
			}
		}
		return nil
	})
	return
}

// SetLang enregistre la langue préférée de l'utilisateur.
func (a *Auth) SetLang(username, lang string) error {
	if lang != "fr" && lang != "en" {
		return fmt.Errorf("langue non prise en charge : %s", lang)
	}
	return a.cfg.Update(func(c *config.Config) error {
		if u := c.FindUser(username); u != nil {
			u.Lang = lang
		}
		return nil
	})
}
