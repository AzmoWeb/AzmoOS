// Package storage gère les disques : détection, montage, formatage, montage au démarrage.
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/azmoweb/azmoos/internal/cmdutil"
	"github.com/azmoweb/azmoos/internal/jobs"
)

// Partition est un système de fichiers (ou une partition vide).
type Partition struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       uint64 `json:"size"`
	FSType     string `json:"fstype"`
	Label      string `json:"label"`
	UUID       string `json:"uuid"`
	Mountpoint string `json:"mountpoint"`
	FSUsed     uint64 `json:"fsUsed"`
	FSAvail    uint64 `json:"fsAvail"`
	AutoMount  bool   `json:"autoMount"`
	System     bool   `json:"system"`
}

// Disk est un disque physique.
type Disk struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Model      string      `json:"model"`
	Size       uint64      `json:"size"`
	Transport  string      `json:"transport"` // usb, nvme, mmc, sata
	Removable  bool        `json:"removable"`
	System     bool        `json:"system"`
	Partitions []Partition `json:"partitions"`
}

type Manager struct {
	disksDir string // /DATA/Disks
	jobs     *jobs.Runner
}

func New(dataDir string, r *jobs.Runner) *Manager {
	return &Manager{disksDir: filepath.Join(dataDir, "Disks"), jobs: r}
}

// num accepte les nombres JSON ou les chaînes (selon la version de lsblk).
type num uint64

func (n *num) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	*n = num(v)
	return err
}

type flag bool

func (f *flag) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	*f = flag(s == "true" || s == "1")
	return nil
}

type lsblkDev struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Size       num        `json:"size"`
	Type       string     `json:"type"`
	FSType     *string    `json:"fstype"`
	Label      *string    `json:"label"`
	UUID       *string    `json:"uuid"`
	Mountpoint *string    `json:"mountpoint"`
	Model      *string    `json:"model"`
	Tran       *string    `json:"tran"`
	RM         flag       `json:"rm"`
	FSUsed     num        `json:"fsused"`
	FSAvail    num        `json:"fsavail"`
	Children   []lsblkDev `json:"children"`
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

var systemMounts = map[string]bool{"/": true, "/boot": true, "/boot/firmware": true, "[SWAP]": true}

// List renvoie les disques détectés.
func (m *Manager) List() ([]Disk, error) {
	if !cmdutil.Has("lsblk") {
		return nil, errors.New("lsblk introuvable")
	}
	out, err := exec.Command("lsblk", "-J", "-b", "-o", "NAME,PATH,SIZE,TYPE,FSTYPE,LABEL,UUID,MOUNTPOINT,MODEL,TRAN,RM,FSUSED,FSAVAIL").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk : %w", err)
	}
	var res struct {
		Blockdevices []lsblkDev `json:"blockdevices"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}
	fstab := readFstab()
	disks := []Disk{}
	for _, d := range res.Blockdevices {
		if d.Type != "disk" || strings.HasPrefix(d.Name, "zram") || strings.HasPrefix(d.Name, "loop") || d.Size == 0 {
			continue
		}
		dk := Disk{Name: d.Name, Path: d.Path, Model: str(d.Model), Size: uint64(d.Size), Transport: str(d.Tran), Removable: bool(d.RM), Partitions: []Partition{}}
		if dk.Path == "" {
			dk.Path = "/dev/" + d.Name
		}
		if dk.Transport == "" && strings.HasPrefix(d.Name, "mmcblk") {
			dk.Transport = "mmc"
		}
		if dk.Model == "" {
			dk.Model = map[string]string{"mmc": "Carte microSD", "nvme": "SSD NVMe", "usb": "Disque USB"}[dk.Transport]
		}
		parts := d.Children
		if len(parts) == 0 && str(d.FSType) != "" { // disque formaté sans table de partitions
			parts = []lsblkDev{d}
		}
		for _, p := range parts {
			pt := Partition{Name: p.Name, Path: p.Path, Size: uint64(p.Size), FSType: str(p.FSType), Label: str(p.Label),
				UUID: str(p.UUID), Mountpoint: str(p.Mountpoint), FSUsed: uint64(p.FSUsed), FSAvail: uint64(p.FSAvail)}
			if pt.Path == "" {
				pt.Path = "/dev/" + p.Name
			}
			pt.System = systemMounts[pt.Mountpoint]
			pt.AutoMount = pt.UUID != "" && fstab[pt.UUID]
			if pt.System {
				dk.System = true
			}
			dk.Partitions = append(dk.Partitions, pt)
		}
		disks = append(disks, dk)
	}
	return disks, nil
}

func (m *Manager) find(path string) (*Disk, *Partition, error) {
	disks, err := m.List()
	if err != nil {
		return nil, nil, err
	}
	for i := range disks {
		if disks[i].Path == path {
			return &disks[i], nil, nil
		}
		for j := range disks[i].Partitions {
			if disks[i].Partitions[j].Path == path {
				return &disks[i], &disks[i].Partitions[j], nil
			}
		}
	}
	return nil, nil, errors.New("périphérique introuvable (débranché ?)")
}

var nameRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func mountName(p *Partition) string {
	n := nameRe.ReplaceAllString(p.Label, "-")
	n = strings.Trim(n, "-.")
	if n == "" {
		n = p.Name
	}
	return n
}

func mountOptions(fstype string) []string {
	switch fstype {
	case "vfat", "exfat", "ntfs", "ntfs3":
		// Pas de droits Unix : on donne les fichiers à l'utilisateur 1000 (apps Docker).
		return []string{"-o", "uid=1000,gid=1000,umask=0002"}
	}
	return nil
}

// Mount monte une partition dans /DATA/Disks/<nom>.
func (m *Manager) Mount(path string) (string, error) {
	_, p, err := m.find(path)
	if err != nil {
		return "", err
	}
	if p == nil || p.FSType == "" {
		return "", errors.New("cette partition n'est pas formatée")
	}
	if p.Mountpoint != "" {
		return p.Mountpoint, nil
	}
	mp := filepath.Join(m.disksDir, mountName(p))
	if err := os.MkdirAll(mp, 0o755); err != nil {
		return "", err
	}
	fstype := p.FSType
	if fstype == "ntfs" {
		fstype = "ntfs3"
	}
	args := append([]string{"-t", fstype}, mountOptions(p.FSType)...)
	args = append(args, p.Path, mp)
	if _, err := cmdutil.Run("mount", args...); err != nil {
		_ = os.Remove(mp)
		return "", err
	}
	return mp, nil
}

// Unmount démonte une partition.
func (m *Manager) Unmount(path string) error {
	_, p, err := m.find(path)
	if err != nil {
		return err
	}
	if p == nil || p.Mountpoint == "" {
		return nil
	}
	if p.System {
		return errors.New("impossible de démonter le disque système")
	}
	if _, err := cmdutil.Run("umount", p.Path); err != nil {
		return fmt.Errorf("démontage impossible (fichiers en cours d'utilisation ?) : %w", err)
	}
	if strings.HasPrefix(p.Mountpoint, m.disksDir+"/") {
		_ = os.Remove(p.Mountpoint)
	}
	return nil
}

// Format efface un disque entier et crée une partition unique.
func (m *Manager) Format(diskPath, fstype, label string) error {
	d, p, err := m.find(diskPath)
	if err != nil {
		return err
	}
	if p != nil {
		return errors.New("choisis un disque entier, pas une partition")
	}
	if d.System {
		return errors.New("impossible de formater le disque système")
	}
	if fstype != "ext4" && fstype != "exfat" {
		return errors.New("format non supporté (ext4 ou exfat)")
	}
	label = nameRe.ReplaceAllString(label, "")
	if label == "" {
		label = "AzmoDisk"
	}
	if len(label) > 11 {
		label = label[:11]
	}
	part := d.Path + "1"
	if strings.HasPrefix(d.Name, "nvme") || strings.HasPrefix(d.Name, "mmcblk") {
		part = d.Path + "p1"
	}
	mkfs := []string{"mkfs.ext4", "-F", "-L", label, part}
	if fstype == "exfat" {
		mkfs = []string{"mkfs.exfat", "-n", label, part}
	}
	steps := []jobs.Step{}
	for _, pt := range d.Partitions {
		if pt.Mountpoint != "" {
			steps = append(steps, jobs.Step{Cmd: []string{"umount", pt.Path}})
		}
	}
	steps = append(steps,
		jobs.Step{Cmd: []string{"wipefs", "-a", d.Path}},
		jobs.Step{Func: func(logf func(string, ...any)) error {
			logf("$ sfdisk --label gpt %s\n", d.Path)
			cmd := exec.Command("sfdisk", "--label", "gpt", d.Path)
			cmd.Stdin = strings.NewReader(",,L\n")
			var b bytes.Buffer
			cmd.Stdout, cmd.Stderr = &b, &b
			err := cmd.Run()
			logf("%s", b.String())
			return err
		}},
		jobs.Step{Cmd: []string{"partprobe", d.Path}},
		jobs.Step{Cmd: []string{"udevadm", "settle"}},
		jobs.Step{Cmd: mkfs},
	)
	if !cmdutil.Has("partprobe") {
		steps = append(steps[:len(steps)-3], steps[len(steps)-2:]...)
	}
	return m.jobs.Start("storage", d.Name, "format", steps, nil)
}

const fstabMark = "# azmoos"

func readFstab() map[string]bool {
	out := map[string]bool{}
	b, _ := os.ReadFile("/etc/fstab")
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "UUID=") && strings.Contains(l, fstabMark) {
			out[strings.TrimPrefix(strings.Fields(l)[0], "UUID=")] = true
		}
	}
	return out
}

// SetAutoMount ajoute/retire la partition de /etc/fstab (montage au démarrage).
func (m *Manager) SetAutoMount(path string, on bool) error {
	_, p, err := m.find(path)
	if err != nil {
		return err
	}
	if p == nil || p.UUID == "" || p.System {
		return errors.New("partition non éligible")
	}
	b, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return err
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if strings.HasPrefix(l, "UUID="+p.UUID+" ") && strings.Contains(l, fstabMark) {
			continue
		}
		lines = append(lines, l)
	}
	if on {
		mp := p.Mountpoint
		if mp == "" {
			mp = filepath.Join(m.disksDir, mountName(p))
		}
		_ = os.MkdirAll(mp, 0o755)
		fstype := p.FSType
		opts := "defaults,nofail,x-systemd.device-timeout=10s"
		switch fstype {
		case "ntfs":
			fstype = "ntfs3"
			fallthrough
		case "vfat", "exfat":
			opts += ",uid=1000,gid=1000,umask=0002"
		}
		lines = append(lines, fmt.Sprintf("UUID=%s %s %s %s 0 0 %s", p.UUID, mp, fstype, opts, fstabMark))
	}
	tmp := "/etc/fstab.azmoos.tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, "/etc/fstab"); err != nil {
		return err
	}
	_, _ = cmdutil.Run("systemctl", "daemon-reload")
	return nil
}
