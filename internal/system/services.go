package system

import (
	"os"
	"strings"

	"github.com/azmoweb/azmoos/internal/cmdutil"
	"github.com/azmoweb/azmoos/internal/jobs"
)

// ServiceState décrit un service systemd.
type ServiceState struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Active    bool   `json:"active"`
	Enabled   bool   `json:"enabled"`
}

// Service renvoie l'état d'un service systemd.
func Service(name string) ServiceState {
	s := ServiceState{Name: name}
	if !cmdutil.Has("systemctl") {
		return s
	}
	out, _ := cmdutil.Run("systemctl", "show", name, "--property=LoadState,ActiveState,UnitFileState")
	for _, l := range strings.Split(out, "\n") {
		k, v, _ := strings.Cut(l, "=")
		switch k {
		case "LoadState":
			s.Installed = v == "loaded"
		case "ActiveState":
			s.Active = v == "active"
		case "UnitFileState":
			s.Enabled = v == "enabled"
		}
	}
	return s
}

// SetService active (et démarre) ou désactive (et arrête) un service.
func SetService(name string, on bool) error {
	if on {
		_, err := cmdutil.Run("systemctl", "enable", "--now", name)
		return err
	}
	_, err := cmdutil.Run("systemctl", "disable", "--now", name)
	return err
}

// SSHService renvoie le nom du service SSH (ssh sur Debian, sshd ailleurs).
func SSHService() string {
	if Service("ssh").Installed {
		return "ssh"
	}
	return "sshd"
}

// Package est une mise à jour disponible.
type Package struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	New     string `json:"new"`
}

// Upgradable liste les paquets APT à mettre à jour (selon le dernier apt update).
func Upgradable() []Package {
	out := []Package{}
	if !cmdutil.Has("apt") {
		return out
	}
	s, _ := cmdutil.Run("apt", "list", "--upgradable")
	for _, l := range strings.Split(s, "\n") {
		// nom/suite version arch [upgradable from: ancienne]
		if !strings.Contains(l, "[upgradable from:") {
			continue
		}
		f := strings.Fields(l)
		if len(f) < 6 {
			continue
		}
		name, _, _ := strings.Cut(f[0], "/")
		out = append(out, Package{Name: name, New: f[1], Current: strings.TrimSuffix(f[len(f)-1], "]")})
	}
	return out
}

// RebootRequired indique si un redémarrage est nécessaire après une mise à jour.
func RebootRequired() bool { return exists("/var/run/reboot-required") }

var aptEnv = []string{"DEBIAN_FRONTEND=noninteractive"}

// AptRefreshSteps rafraîchit la liste des paquets.
func AptRefreshSteps() []jobs.Step {
	return []jobs.Step{{Cmd: []string{"apt-get", "update"}, Env: aptEnv}}
}

// AptUpgradeSteps installe toutes les mises à jour.
func AptUpgradeSteps() []jobs.Step {
	return []jobs.Step{
		{Cmd: []string{"apt-get", "update"}, Env: aptEnv},
		{Cmd: []string{"apt-get", "-y", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", "full-upgrade"}, Env: aptEnv},
		{Cmd: []string{"apt-get", "-y", "autoremove"}, Env: aptEnv},
	}
}

// Timezone renvoie le fuseau horaire.
func Timezone() string {
	if b, err := os.ReadFile("/etc/timezone"); err == nil {
		return strings.TrimSpace(string(b))
	}
	if l, err := os.Readlink("/etc/localtime"); err == nil {
		if i := strings.Index(l, "zoneinfo/"); i >= 0 {
			return l[i+9:]
		}
	}
	return "UTC"
}

// SetTimezone change le fuseau horaire.
func SetTimezone(tz string) error {
	_, err := cmdutil.Run("timedatectl", "set-timezone", tz)
	return err
}
