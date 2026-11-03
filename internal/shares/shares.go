// Package shares gère les partages réseau Samba (NAS).
package shares

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/azmoweb/azmoos/internal/cmdutil"
	"github.com/azmoweb/azmoos/internal/config"
)

const (
	confPath = "/etc/samba/azmoos-shares.conf"
	mainConf = "/etc/samba/smb.conf"
)

type Manager struct {
	cfg     *config.Store
	dataDir string
}

func New(cfg *config.Store, dataDir string) *Manager { return &Manager{cfg: cfg, dataDir: dataDir} }

// Status décrit l'état de Samba.
type Status struct {
	Installed bool     `json:"installed"`
	Running   bool     `json:"running"`
	SMBUsers  []string `json:"smbUsers"`
}

func (m *Manager) Status() Status {
	s := Status{Installed: cmdutil.Has("smbd"), SMBUsers: []string{}}
	if out, err := cmdutil.Run("systemctl", "is-active", "smbd"); err == nil && out == "active" {
		s.Running = true
	}
	if out, err := cmdutil.Run("pdbedit", "-L"); err == nil {
		for _, l := range strings.Split(out, "\n") {
			if n, _, ok := strings.Cut(l, ":"); ok && n != "" {
				s.SMBUsers = append(s.SMBUsers, n)
			}
		}
	}
	return s
}

func (m *Manager) List() []config.Share {
	var out []config.Share
	m.cfg.View(func(c *config.Config) { out = append(out, c.Shares...) })
	if out == nil {
		out = []config.Share{}
	}
	return out
}

var shareName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _-]{0,31}$`)

// Save crée ou remplace un partage.
func (m *Manager) Save(s config.Share) error {
	s.Name = strings.TrimSpace(s.Name)
	if !shareName.MatchString(s.Name) || strings.EqualFold(s.Name, "global") || strings.EqualFold(s.Name, "homes") {
		return errors.New("nom de partage invalide (lettres, chiffres, espaces, - et _)")
	}
	rel := filepath.Clean("/" + s.Path)
	abs := filepath.Join(m.dataDir, rel)
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return errors.New("le dossier à partager n'existe pas")
	}
	s.Path = rel
	err := m.cfg.Update(func(c *config.Config) error {
		c.Shares = slices.DeleteFunc(c.Shares, func(x config.Share) bool { return strings.EqualFold(x.Name, s.Name) })
		c.Shares = append(c.Shares, s)
		return nil
	})
	if err != nil {
		return err
	}
	return m.Apply()
}

// Delete supprime un partage.
func (m *Manager) Delete(name string) error {
	err := m.cfg.Update(func(c *config.Config) error {
		c.Shares = slices.DeleteFunc(c.Shares, func(x config.Share) bool { return x.Name == name })
		return nil
	})
	if err != nil {
		return err
	}
	return m.Apply()
}

func owner() string {
	if u, err := user.LookupId("1000"); err == nil {
		return u.Username
	}
	return "root"
}

// Apply régénère la configuration Samba et recharge le service.
func (m *Manager) Apply() error {
	if !cmdutil.Has("smbd") {
		return errors.New("Samba n'est pas installé (sudo apt install samba)")
	}
	var b strings.Builder
	b.WriteString("# Généré par Azmo OS — ne pas modifier à la main\n")
	force := owner()
	for _, s := range m.List() {
		fmt.Fprintf(&b, "\n[%s]\n   path = %s\n   browseable = yes\n   read only = %s\n   force user = %s\n   create mask = 0664\n   directory mask = 0775\n",
			s.Name, filepath.Join(m.dataDir, s.Path), yesno(s.ReadOnly), force)
		if s.Guest {
			b.WriteString("   guest ok = yes\n")
		} else if len(s.Users) > 0 {
			fmt.Fprintf(&b, "   valid users = %s\n", strings.Join(s.Users, " "))
		}
	}
	if err := os.WriteFile(confPath, []byte(b.String()), 0o644); err != nil {
		return err
	}
	if err := ensureInclude(); err != nil {
		return err
	}
	_, err := cmdutil.Run("systemctl", "reload-or-restart", "smbd")
	return err
}

func yesno(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func ensureInclude() error {
	b, err := os.ReadFile(mainConf)
	if err != nil {
		return err
	}
	line := "include = " + confPath
	if strings.Contains(string(b), line) {
		return nil
	}
	s := string(b)
	if !strings.Contains(s, "map to guest") {
		s = strings.Replace(s, "[global]", "[global]\n   map to guest = Bad User", 1)
	}
	return os.WriteFile(mainConf, []byte(strings.TrimRight(s, "\n")+"\n\n# Partages Azmo OS\n"+line+"\n"), 0o644)
}

// SetPassword active l'accès réseau d'un utilisateur Azmo OS (compte Unix sans shell + Samba).
func (m *Manager) SetPassword(username, password string) error {
	if len(password) < 8 {
		return errors.New("mot de passe réseau : 8 caractères minimum")
	}
	if !cmdutil.Has("smbpasswd") {
		return errors.New("Samba n'est pas installé")
	}
	if _, err := user.Lookup(username); err != nil {
		if _, err := cmdutil.Run("useradd", "--no-create-home", "--shell", "/usr/sbin/nologin", "-G", groupOf(owner()), username); err != nil {
			return err
		}
	}
	cmd := exec.Command("smbpasswd", "-a", "-s", username)
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("smbpasswd : %s", strings.TrimSpace(string(out)))
	}
	_, _ = cmdutil.Run("smbpasswd", "-e", username)
	return nil
}

func groupOf(u string) string {
	if usr, err := user.Lookup(u); err == nil {
		if g, err := user.LookupGroupId(usr.Gid); err == nil {
			return g.Name
		}
	}
	return "users"
}

// RemoveUser retire l'accès réseau.
func (m *Manager) RemoveUser(username string) error {
	_, err := cmdutil.Run("smbpasswd", "-x", username)
	return err
}
