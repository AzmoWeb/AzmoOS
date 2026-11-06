// Package network gère les interfaces et le Wi-Fi via NetworkManager (nmcli).
package network

import (
	"errors"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/cmdutil"
)

// Available indique si NetworkManager est utilisable.
func Available() bool {
	if !cmdutil.Has("nmcli") {
		return false
	}
	_, err := cmdutil.Run("nmcli", "-t", "general", "status")
	return err == nil
}

type Interface struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"` // ethernet, wifi, other
	State      string   `json:"state"`
	MAC        string   `json:"mac"`
	IPs        []string `json:"ips"`
	Gateway    string   `json:"gateway"`
	DNS        []string `json:"dns"`
	Connection string   `json:"connection"`
	Method     string   `json:"method"` // auto (DHCP) ou manual
	SpeedMbps  int      `json:"speed"`
}

// split découpe une ligne nmcli -t (les ':' échappés en '\:').
func split(l string) []string {
	var out []string
	var cur strings.Builder
	for i := 0; i < len(l); i++ {
		if l[i] == '\\' && i+1 < len(l) {
			cur.WriteByte(l[i+1])
			i++
			continue
		}
		if l[i] == ':' {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(l[i])
	}
	return append(out, cur.String())
}

func ignored(name string) bool {
	for _, p := range []string{"lo", "veth", "docker", "br-", "tailscale", "wg", "p2p"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// Interfaces liste les interfaces réseau physiques.
func Interfaces() []Interface {
	out := []Interface{}
	if Available() {
		s, _ := cmdutil.Run("nmcli", "-t", "-f", "DEVICE,TYPE,STATE,CONNECTION", "device")
		for _, l := range strings.Split(s, "\n") {
			f := split(l)
			if len(f) < 4 || ignored(f[0]) || (f[1] != "ethernet" && f[1] != "wifi") {
				continue
			}
			it := Interface{Name: f[0], Type: f[1], State: f[2], Connection: f[3], IPs: []string{}, DNS: []string{}}
			d, _ := cmdutil.Run("nmcli", "-t", "-f", "GENERAL.HWADDR,IP4.ADDRESS,IP4.GATEWAY,IP4.DNS", "device", "show", it.Name)
			for _, dl := range strings.Split(d, "\n") {
				k, v, _ := strings.Cut(dl, ":")
				v = strings.ReplaceAll(v, `\:`, ":")
				switch {
				case k == "GENERAL.HWADDR":
					it.MAC = v
				case strings.HasPrefix(k, "IP4.ADDRESS"):
					it.IPs = append(it.IPs, v)
				case k == "IP4.GATEWAY":
					it.Gateway = v
				case strings.HasPrefix(k, "IP4.DNS"):
					it.DNS = append(it.DNS, v)
				}
			}
			if it.Connection != "" && it.Connection != "--" {
				m, _ := cmdutil.Run("nmcli", "-g", "ipv4.method", "connection", "show", it.Connection)
				it.Method = m
			} else {
				it.Connection = ""
			}
			it.SpeedMbps = speed(it.Name)
			out = append(out, it)
		}
		return out
	}
	// Repli sans NetworkManager : lecture simple.
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if ignored(i.Name) {
			continue
		}
		it := Interface{Name: i.Name, Type: "ethernet", MAC: i.HardwareAddr.String(), IPs: []string{}, DNS: []string{}, State: "déconnecté"}
		if _, err := os.Stat("/sys/class/net/" + i.Name + "/wireless"); err == nil {
			it.Type = "wifi"
		}
		if i.Flags&net.FlagUp != 0 {
			it.State = "connecté"
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
				ones, _ := ipn.Mask.Size()
				it.IPs = append(it.IPs, ipn.IP.String()+"/"+strconv.Itoa(ones))
			}
		}
		it.SpeedMbps = speed(i.Name)
		out = append(out, it)
	}
	return out
}

func speed(name string) int {
	b, err := os.ReadFile("/sys/class/net/" + name + "/speed")
	if err != nil {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if v < 0 {
		return 0
	}
	return v
}

type WiFi struct {
	SSID     string `json:"ssid"`
	Signal   int    `json:"signal"`
	Security string `json:"security"`
	Active   bool   `json:"active"`
	Known    bool   `json:"known"`
}

// ScanWiFi liste les réseaux Wi-Fi à portée.
func ScanWiFi() ([]WiFi, error) {
	if !Available() {
		return nil, errors.New("NetworkManager n'est pas disponible")
	}
	_, _ = cmdutil.RunTimeout(15*time.Second, "nmcli", "device", "wifi", "rescan")
	s, err := cmdutil.Run("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL,SECURITY", "device", "wifi", "list")
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	k, _ := cmdutil.Run("nmcli", "-t", "-f", "NAME,TYPE", "connection", "show")
	for _, l := range strings.Split(k, "\n") {
		f := split(l)
		if len(f) == 2 && strings.Contains(f[1], "wireless") {
			known[f[0]] = true
		}
	}
	seen := map[string]int{}
	out := []WiFi{}
	for _, l := range strings.Split(s, "\n") {
		f := split(l)
		if len(f) < 4 || f[1] == "" {
			continue
		}
		sig, _ := strconv.Atoi(f[2])
		w := WiFi{SSID: f[1], Signal: sig, Security: f[3], Active: f[0] == "yes", Known: known[f[1]]}
		if i, ok := seen[w.SSID]; ok {
			if w.Signal > out[i].Signal || w.Active {
				w.Active = w.Active || out[i].Active
				out[i] = w
			}
			continue
		}
		seen[w.SSID] = len(out)
		out = append(out, w)
	}
	return out, nil
}

// ConnectWiFi se connecte à un réseau Wi-Fi.
func ConnectWiFi(ssid, password string) error {
	if ssid == "" {
		return errors.New("nom du réseau requis")
	}
	args := []string{"device", "wifi", "connect", ssid}
	if password != "" {
		args = append(args, "password", password)
	}
	_, err := cmdutil.RunTimeout(45*time.Second, "nmcli", args...)
	return err
}

// Forget supprime un réseau enregistré.
func Forget(name string) error {
	_, err := cmdutil.Run("nmcli", "connection", "delete", "id", name)
	return err
}

var (
	cidrRe = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}/\d{1,2}$`)
	ipRe   = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
)

// IPConfig configure une connexion en DHCP (method=auto) ou IP fixe (manual).
type IPConfig struct {
	Connection string   `json:"connection"`
	Method     string   `json:"method"`
	Address    string   `json:"address"` // 192.168.1.50/24
	Gateway    string   `json:"gateway"`
	DNS        []string `json:"dns"`
}

// Configure applique une configuration IPv4 et réactive la connexion.
func Configure(c IPConfig) error {
	if c.Connection == "" {
		return errors.New("connexion requise")
	}
	args := []string{"connection", "modify", c.Connection}
	switch c.Method {
	case "auto":
		args = append(args, "ipv4.method", "auto", "ipv4.addresses", "", "ipv4.gateway", "", "ipv4.dns", "")
	case "manual":
		if !cidrRe.MatchString(c.Address) || (c.Gateway != "" && !ipRe.MatchString(c.Gateway)) {
			return errors.New("adresse invalide (ex : 192.168.1.50/24)")
		}
		for _, d := range c.DNS {
			if !ipRe.MatchString(d) {
				return errors.New("serveur DNS invalide : " + d)
			}
		}
		args = append(args, "ipv4.method", "manual", "ipv4.addresses", c.Address, "ipv4.gateway", c.Gateway, "ipv4.dns", strings.Join(c.DNS, " "))
	default:
		return errors.New("méthode inconnue")
	}
	if _, err := cmdutil.Run("nmcli", args...); err != nil {
		return err
	}
	// Réactivation en arrière-plan : la connexion HTTP en cours peut être coupée.
	go func() {
		time.Sleep(time.Second)
		_, _ = cmdutil.Run("nmcli", "connection", "up", c.Connection)
	}()
	return nil
}
