// Package config gère la configuration persistante d'Azmo OS (/etc/azmoos/config.json).
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Rôles disponibles.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User est un compte Azmo OS.
type User struct {
	Username string   `json:"username"`
	Display  string   `json:"display,omitempty"`
	Role     string   `json:"role"`
	Perms    []string `json:"perms,omitempty"`
	Hash     string   `json:"hash"`
	Salt     string   `json:"salt"`
	Created  int64    `json:"created"`
	LastSeen int64    `json:"lastSeen,omitempty"`
	Lang     string   `json:"lang,omitempty"`

	// Double authentification (TOTP)
	TOTPSecret   string   `json:"totpSecret,omitempty"`
	TOTPPending  string   `json:"totpPending,omitempty"` // secret en cours d'activation
	TOTPLastStep int64    `json:"totpLastStep,omitempty"`
	Recovery     []string `json:"recovery,omitempty"` // empreintes SHA-256 des codes de secours
}

// Session associe un jeton à un utilisateur.
type Session struct {
	User    string `json:"user"`
	Expires int64  `json:"expires"`
	IP      string `json:"ip,omitempty"`
	Agent   string `json:"agent,omitempty"`
}

// Share est un partage réseau Samba.
type Share struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"` // relatif à /DATA
	ReadOnly bool     `json:"readOnly"`
	Guest    bool     `json:"guest"`
	Users    []string `json:"users,omitempty"`
}

// Notify configure les alertes et leurs canaux d'envoi.
type Notify struct {
	Lang        string `json:"lang"`        // langue des messages envoyés (fr, en)
	TempWarn    int    `json:"tempWarn"`    // °C
	DiskWarn    int    `json:"diskWarn"`    // %
	MemWarn     int    `json:"memWarn"`     // %
	Temperature bool   `json:"temperature"` // surveillances actives
	Disk        bool   `json:"disk"`
	Memory      bool   `json:"memory"`
	Apps        bool   `json:"apps"`
	Updates     bool   `json:"updates"`
	Security    bool   `json:"security"`
	Backups     bool   `json:"backups"`
	PushLevel   string `json:"pushLevel"` // niveau minimal envoyé sur le téléphone : info, warning, critical

	Ntfy     NtfyChannel     `json:"ntfy"`
	Discord  DiscordChannel  `json:"discord"`
	Telegram TelegramChannel `json:"telegram"`
}

type NtfyChannel struct {
	Enabled bool   `json:"enabled"`
	Server  string `json:"server"`
	Topic   string `json:"topic"`
	Token   string `json:"token,omitempty"`
}

type DiscordChannel struct {
	Enabled bool   `json:"enabled"`
	Webhook string `json:"webhook"`
}

type TelegramChannel struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
	ChatID  string `json:"chatId"`
}

// DefaultNotify renvoie les réglages par défaut des alertes.
func DefaultNotify() Notify {
	return Notify{Lang: "fr", TempWarn: 75, DiskWarn: 90, MemWarn: 95, Temperature: true, Disk: true, Memory: true,
		Apps: true, Updates: true, Security: true, Backups: true, PushLevel: "warning", Ntfy: NtfyChannel{Server: "https://ntfy.sh"}}
}

// Backup est une sauvegarde planifiée.
type Backup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Frequency string `json:"frequency"` // daily, weekly
	Hour      int    `json:"hour"`      // 0-23
	Weekday   int    `json:"weekday"`   // 0 = dimanche (hebdomadaire)
	Target    string `json:"target"`    // dossier dans /DATA, ou « rclone:distant:chemin »
	AppData   bool   `json:"appData"`
	Keep      int    `json:"keep"` // nombre de sauvegardes conservées
	LastRun   int64  `json:"lastRun,omitempty"`
	LastOK    bool   `json:"lastOk"`
	LastError string `json:"lastError,omitempty"`
	LastFile  string `json:"lastFile,omitempty"`
	LastSize  int64  `json:"lastSize,omitempty"`
}

// Config est le contenu de config.json.
type Config struct {
	Users    []*User             `json:"users"`
	Sessions map[string]*Session `json:"sessions"`
	Shares   []Share             `json:"shares,omitempty"`
	Settings map[string]string   `json:"settings"`
	Notify   *Notify             `json:"notify,omitempty"`
	Backups  []Backup            `json:"backups,omitempty"`

	// Ancien format v0.1 (un seul compte) — migré automatiquement.
	LegacyUser *User `json:"user,omitempty"`
}

// FindUser renvoie un utilisateur par son nom.
func (c *Config) FindUser(name string) *User {
	for _, u := range c.Users {
		if u.Username == name {
			return u
		}
	}
	return nil
}

// Store protège la config et la sauvegarde sur disque.
type Store struct {
	mu      sync.RWMutex
	dir     string
	path    string
	c       Config
	modTime time.Time // date du fichier lors de la dernière lecture/écriture
	size    int64     // taille du fichier lors de la dernière lecture/écriture
}

// Load lit (ou crée) la configuration.
func Load(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, path: filepath.Join(dir, "config.json")}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) reload() error {
	var c Config
	b, err := os.ReadFile(s.path)
	if err == nil {
		// L'ancien format stockait les sessions sous forme token -> expiration.
		var probe struct {
			Sessions map[string]json.RawMessage `json:"sessions"`
		}
		_ = json.Unmarshal(b, &probe)
		for _, v := range probe.Sessions {
			if len(v) > 0 && v[0] != '{' {
				b = stripSessions(b)
			}
			break
		}
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if c.LegacyUser != nil {
		c.LegacyUser.Role = RoleAdmin
		c.Users = append(c.Users, c.LegacyUser)
		c.LegacyUser = nil
	}
	if c.Sessions == nil {
		c.Sessions = map[string]*Session{}
	}
	if c.Settings == nil {
		c.Settings = map[string]string{}
	}
	if c.Notify == nil {
		n := DefaultNotify()
		c.Notify = &n
	}
	var mt time.Time
	var size int64
	if st, err := os.Stat(s.path); err == nil {
		mt, size = st.ModTime(), st.Size()
	}
	s.mu.Lock()
	s.c, s.modTime, s.size = c, mt, size
	s.mu.Unlock()
	return nil
}

// syncExternal relit le fichier s'il a été modifié par un autre processus
// (par exemple « azmod user add » pendant que le service tourne). Un simple stat : ~1 µs.
func (s *Store) syncExternal() {
	st, err := os.Stat(s.path)
	if err != nil {
		return
	}
	s.mu.RLock()
	changed := !st.ModTime().Equal(s.modTime) || st.Size() != s.size
	s.mu.RUnlock()
	if changed {
		_ = s.reload()
	}
}

func stripSessions(b []byte) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return b
	}
	delete(m, "sessions")
	out, _ := json.Marshal(m)
	return out
}

// Dir renvoie le dossier de configuration.
func (s *Store) Dir() string { return s.dir }

// Reload relit le fichier (après une restauration).
func (s *Store) Reload() error { return s.reload() }

// View exécute fn en lecture.
func (s *Store) View(fn func(c *Config)) {
	s.syncExternal()
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(&s.c)
}

// Update exécute fn puis sauvegarde de façon atomique.
func (s *Store) Update(fn func(c *Config) error) error {
	s.syncExternal()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.c); err != nil {
		return err
	}
	b, err := json.MarshalIndent(&s.c, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if st, err := os.Stat(s.path); err == nil {
		s.modTime, s.size = st.ModTime(), st.Size()
	}
	return nil
}

// Setting lit un réglage.
func (s *Store) Setting(k string) (v string) {
	s.View(func(c *Config) { v = c.Settings[k] })
	return
}
