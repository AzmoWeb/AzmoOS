package system

import (
	"os"
	"runtime"
	"strings"
)

// Profils de performance.
const (
	ProfileLight       = "leger"
	ProfileStandard    = "standard"
	ProfilePerformance = "performance"
)

// Hardware décrit la machine détectée et ses capacités.
type Hardware struct {
	Model       string `json:"model"`
	Family      string `json:"family"` // pi5, pi4, pi3, pizero2, pi, generic
	IsPi        bool   `json:"isPi"`
	RAMMB       int    `json:"ramMB"`
	Cores       int    `json:"cores"`
	Arch        string `json:"arch"`    // arm64, arm, amd64
	Profile     string `json:"profile"` // leger, standard, performance
	ProfileHint string `json:"profileHint"`
	WiFi        bool   `json:"wifi"`
	Bluetooth   bool   `json:"bluetooth"`
	PCIe        bool   `json:"pcie"`    // Pi 5 : SSD NVMe possible
	Refresh     int    `json:"refresh"` // intervalle conseillé (ms) pour l'interface
}

// DetectHardware identifie le Raspberry Pi (ou la machine) et choisit un profil.
func DetectHardware() Hardware {
	h := Hardware{Model: readModel(), Cores: runtime.NumCPU(), Arch: runtime.GOARCH}
	m := strings.ToLower(h.Model)
	h.IsPi = strings.Contains(m, "raspberry pi")
	switch {
	case strings.Contains(m, "raspberry pi 5"), strings.Contains(m, "raspberry pi 500"), strings.Contains(m, "compute module 5"):
		h.Family, h.PCIe = "pi5", true
	case strings.Contains(m, "raspberry pi 4"), strings.Contains(m, "raspberry pi 400"), strings.Contains(m, "compute module 4"):
		h.Family = "pi4"
	case strings.Contains(m, "raspberry pi 3"):
		h.Family = "pi3"
	case strings.Contains(m, "zero 2"):
		h.Family = "pizero2"
	case h.IsPi:
		h.Family = "pi"
	default:
		h.Family = "generic"
	}
	h.RAMMB = int(readMeminfo()["MemTotal"] / (1 << 20))
	switch {
	case h.RAMMB > 0 && h.RAMMB < 1536:
		h.Profile, h.Refresh = ProfileLight, 6000
		h.ProfileHint = "Mode léger : les applications gourmandes sont déconseillées pour garder ton Pi réactif."
	case h.RAMMB > 0 && h.RAMMB < 3584:
		h.Profile, h.Refresh = ProfileStandard, 4000
		h.ProfileHint = "Mode standard : la plupart des applications fonctionnent, évite d'en lancer trop de grosses en même temps."
	default:
		h.Profile, h.Refresh = ProfilePerformance, 2500
		h.ProfileHint = "Mode performance : ton matériel peut faire tourner toutes les applications du catalogue."
	}
	h.WiFi = exists("/sys/class/net/wlan0") || globExists("/sys/class/net/*/wireless")
	h.Bluetooth = exists("/sys/class/bluetooth/hci0")
	return h
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func globExists(pattern string) bool {
	entries, _ := os.ReadDir("/sys/class/net")
	for _, e := range entries {
		if exists("/sys/class/net/" + e.Name() + "/wireless") {
			return true
		}
	}
	_ = pattern
	return false
}

// CatalogArch renvoie le nom d'architecture utilisé dans le catalogue d'apps.
func CatalogArch() string {
	if runtime.GOARCH == "arm" {
		return "arm"
	}
	return runtime.GOARCH
}
