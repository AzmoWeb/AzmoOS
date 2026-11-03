// Package system collecte les informations matérielles (optimisé Raspberry Pi).
package system

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Disk struct {
	Path  string `json:"path"`
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type Iface struct {
	Name string   `json:"name"`
	MAC  string   `json:"mac"`
	IPs  []string `json:"ips"`
}

type Stats struct {
	Hostname   string    `json:"hostname"`
	Model      string    `json:"model"`
	OS         string    `json:"os"`
	Kernel     string    `json:"kernel"`
	Arch       string    `json:"arch"`
	Cores      int       `json:"cores"`
	CPU        float64   `json:"cpu"`      // %
	CPUFreqMHz int       `json:"cpuFreq"`  // MHz
	Load       []float64 `json:"load"`
	MemTotal   uint64    `json:"memTotal"`
	MemUsed    uint64    `json:"memUsed"`
	SwapTotal  uint64    `json:"swapTotal"`
	SwapUsed   uint64    `json:"swapUsed"`
	TempC      float64   `json:"temp"`
	Throttled  string    `json:"throttled,omitempty"`
	Uptime     int64     `json:"uptime"`
	NetRx      uint64    `json:"netRx"` // octets/s
	NetTx      uint64    `json:"netTx"`
	Disks      []Disk    `json:"disks"`
	Interfaces []Iface   `json:"interfaces"`
	Time       int64     `json:"time"`
}

// Monitor échantillonne CPU et réseau en tâche de fond.
type Monitor struct {
	mu        sync.RWMutex
	cpu       float64
	rx, tx    uint64
	dataDir   string
	model, os string
}

func NewMonitor(dataDir string) *Monitor {
	m := &Monitor{dataDir: dataDir, model: readModel(), os: readOS()}
	go m.loop()
	return m
}

func (m *Monitor) loop() {
	idle0, total0 := readCPU()
	rx0, tx0 := readNet()
	t0 := time.Now()
	for {
		time.Sleep(2 * time.Second)
		idle1, total1 := readCPU()
		rx1, tx1 := readNet()
		dt := time.Since(t0).Seconds()
		m.mu.Lock()
		if total1 > total0 {
			m.cpu = 100 * (1 - float64(idle1-idle0)/float64(total1-total0))
		}
		if dt > 0 && rx1 >= rx0 && tx1 >= tx0 {
			m.rx = uint64(float64(rx1-rx0) / dt)
			m.tx = uint64(float64(tx1-tx0) / dt)
		}
		m.mu.Unlock()
		idle0, total0, rx0, tx0, t0 = idle1, total1, rx1, tx1, time.Now()
	}
}

// Snapshot renvoie l'état actuel.
func (m *Monitor) Snapshot() Stats {
	s := Stats{Model: m.model, OS: m.os, Arch: runtime.GOARCH, Cores: runtime.NumCPU(), Time: time.Now().Unix()}
	s.Hostname, _ = os.Hostname()
	s.Kernel = strings.TrimSpace(readFile("/proc/sys/kernel/osrelease"))
	m.mu.RLock()
	s.CPU, s.NetRx, s.NetTx = m.cpu, m.rx, m.tx
	m.mu.RUnlock()
	if f, err := strconv.Atoi(strings.TrimSpace(readFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq"))); err == nil {
		s.CPUFreqMHz = f / 1000
	}
	for _, f := range strings.Fields(readFile("/proc/loadavg")) {
		if v, err := strconv.ParseFloat(f, 64); err == nil && len(s.Load) < 3 {
			s.Load = append(s.Load, v)
		}
	}
	mem := readMeminfo()
	s.MemTotal = mem["MemTotal"]
	s.MemUsed = mem["MemTotal"] - mem["MemAvailable"]
	s.SwapTotal = mem["SwapTotal"]
	s.SwapUsed = mem["SwapTotal"] - mem["SwapFree"]
	if t, err := strconv.ParseFloat(strings.TrimSpace(readFile("/sys/class/thermal/thermal_zone0/temp")), 64); err == nil {
		s.TempC = t / 1000
	}
	if out, err := exec.Command("vcgencmd", "get_throttled").Output(); err == nil {
		s.Throttled = strings.TrimPrefix(strings.TrimSpace(string(out)), "throttled=")
	}
	if f := strings.Fields(readFile("/proc/uptime")); len(f) > 0 {
		v, _ := strconv.ParseFloat(f[0], 64)
		s.Uptime = int64(v)
	}
	seen := map[uint64]bool{}
	for _, p := range []string{"/", m.dataDir} {
		var st syscall.Statfs_t
		if syscall.Statfs(p, &st) != nil {
			continue
		}
		if seen[st.Blocks] && p != "/" {
			continue
		}
		seen[st.Blocks] = true
		total := st.Blocks * uint64(st.Bsize)
		free := st.Bavail * uint64(st.Bsize)
		s.Disks = append(s.Disks, Disk{Path: p, Total: total, Free: free, Used: total - st.Bfree*uint64(st.Bsize)})
	}
	s.Interfaces = interfaces()
	return s
}

func readFile(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

func readModel() string {
	m := strings.Trim(readFile("/proc/device-tree/model"), "\x00\n ")
	if m == "" {
		m = "Ordinateur générique (" + runtime.GOARCH + ")"
	}
	return m
}

func readOS() string {
	for _, l := range strings.Split(readFile("/etc/os-release"), "\n") {
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(l, "PRETTY_NAME="), `"`)
		}
	}
	return runtime.GOOS
}

func readCPU() (idle, total uint64) {
	line := strings.SplitN(readFile("/proc/stat"), "\n", 2)[0]
	f := strings.Fields(line)
	for i := 1; i < len(f); i++ {
		v, _ := strconv.ParseUint(f[i], 10, 64)
		total += v
		if i == 4 || i == 5 { // idle + iowait
			idle += v
		}
	}
	return
}

func readNet() (rx, tx uint64) {
	sc := bufio.NewScanner(strings.NewReader(readFile("/proc/net/dev")))
	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "lo" || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") {
			continue
		}
		f := strings.Fields(rest)
		if len(f) >= 9 {
			r, _ := strconv.ParseUint(f[0], 10, 64)
			t, _ := strconv.ParseUint(f[8], 10, 64)
			rx += r
			tx += t
		}
	}
	return
}

func readMeminfo() map[string]uint64 {
	m := map[string]uint64{}
	for _, l := range strings.Split(readFile("/proc/meminfo"), "\n") {
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		f := strings.Fields(v)
		if len(f) > 0 {
			n, _ := strconv.ParseUint(f[0], 10, 64)
			m[k] = n * 1024
		}
	}
	return m
}

func interfaces() []Iface {
	var out []Iface
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagLoopback != 0 || i.Flags&net.FlagUp == 0 ||
			strings.HasPrefix(i.Name, "veth") || strings.HasPrefix(i.Name, "docker") || strings.HasPrefix(i.Name, "br-") {
			continue
		}
		it := Iface{Name: i.Name, MAC: i.HardwareAddr.String()}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
				it.IPs = append(it.IPs, ipn.IP.String())
			}
		}
		out = append(out, it)
	}
	return out
}

// Power redémarre ou éteint la machine.
func Power(action string) error {
	switch action {
	case "reboot", "poweroff":
		return exec.Command("systemctl", action).Start()
	}
	return os.ErrInvalid
}

// SetHostname change le nom d'hôte.
func SetHostname(name string) error {
	out, err := exec.Command("hostnamectl", "set-hostname", name).CombinedOutput()
	if err != nil {
		return &cmdErr{err, string(out)}
	}
	return nil
}

type cmdErr struct {
	err error
	out string
}

func (e *cmdErr) Error() string { return strings.TrimSpace(e.err.Error() + ": " + e.out) }
