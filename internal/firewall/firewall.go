// Package firewall pilote le pare-feu ufw.
package firewall

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/azmoweb/azmoos/internal/cmdutil"
)

type Rule struct {
	Number int    `json:"number"`
	To     string `json:"to"`
	Action string `json:"action"`
	From   string `json:"from"`
	V6     bool   `json:"v6"`
}

type Status struct {
	Installed bool   `json:"installed"`
	Active    bool   `json:"active"`
	Rules     []Rule `json:"rules"`
}

var ruleRe = regexp.MustCompile(`^\[\s*(\d+)\]\s+(.+?)\s{2,}(ALLOW|DENY|REJECT|LIMIT)( IN| OUT)?\s+(.+)$`)

// Get renvoie l'état du pare-feu.
func Get() Status {
	s := Status{Installed: cmdutil.Has("ufw"), Rules: []Rule{}}
	if !s.Installed {
		return s
	}
	out, err := cmdutil.Run("ufw", "status", "numbered")
	if err != nil {
		return s
	}
	s.Active = strings.Contains(out, "Status: active")
	for _, l := range strings.Split(out, "\n") {
		m := ruleRe.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		r := Rule{Number: n, To: strings.TrimSpace(m[2]), Action: m[3], From: strings.TrimSpace(m[5])}
		r.V6 = strings.Contains(r.To, "(v6)")
		s.Rules = append(s.Rules, r)
	}
	return s
}

// Enable active le pare-feu en autorisant d'abord l'interface web et SSH.
func Enable(webPort int) error {
	if !cmdutil.Has("ufw") {
		return errors.New("ufw n'est pas installé (sudo apt install ufw)")
	}
	for _, p := range []string{strconv.Itoa(webPort) + "/tcp", "22/tcp", "5353/udp"} {
		if _, err := cmdutil.Run("ufw", "allow", p); err != nil {
			return err
		}
	}
	_, err := cmdutil.Run("ufw", "--force", "enable")
	return err
}

// Disable désactive le pare-feu.
func Disable() error {
	_, err := cmdutil.Run("ufw", "disable")
	return err
}

var portRe = regexp.MustCompile(`^\d{1,5}(:\d{1,5})?$`)
var srcRe = regexp.MustCompile(`^[0-9a-fA-F.:/]+$`)

// Add ajoute une règle : action allow/deny, port (ex: 8080 ou 6000:6010), proto tcp/udp/any, source optionnelle.
func Add(action, port, proto, from string) error {
	if action != "allow" && action != "deny" {
		return errors.New("action invalide")
	}
	if !portRe.MatchString(port) {
		return errors.New("port invalide")
	}
	if proto != "tcp" && proto != "udp" && proto != "any" && proto != "" {
		return errors.New("protocole invalide")
	}
	if strings.Contains(port, ":") && (proto == "any" || proto == "") {
		return errors.New("une plage de ports exige tcp ou udp")
	}
	args := []string{action}
	if from != "" {
		if !srcRe.MatchString(from) {
			return errors.New("source invalide (ex : 192.168.1.0/24)")
		}
		args = append(args, "from", from, "to", "any", "port", port)
		if proto == "tcp" || proto == "udp" {
			args = append(args, "proto", proto)
		}
	} else {
		p := port
		if proto == "tcp" || proto == "udp" {
			p += "/" + proto
		}
		args = append(args, p)
	}
	_, err := cmdutil.Run("ufw", args...)
	return err
}

// Delete supprime la règle numéro n.
func Delete(n int) error {
	_, err := cmdutil.Run("ufw", "--force", "delete", strconv.Itoa(n))
	return err
}
