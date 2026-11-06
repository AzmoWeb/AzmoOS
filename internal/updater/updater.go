// Package updater met à jour Azmo OS depuis les Releases GitHub.
package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/azmoweb/azmoos/internal/jobs"
)

type Release struct {
	Version   string `json:"version"`
	Name      string `json:"name"`
	Notes     string `json:"notes"`
	Published string `json:"published"`
	URL       string `json:"url"`
	assets    map[string]string
}

type Info struct {
	Current   string   `json:"current"`
	Latest    *Release `json:"latest"`
	Available bool     `json:"available"`
	Checked   int64    `json:"checked"`
	Error     string   `json:"error,omitempty"`
}

type Updater struct {
	Repo    string // azmoweb/azmoos
	Version string
	WebDir  string // /usr/share/azmoos/web
	jobs    *jobs.Runner
	mu      sync.Mutex
	info    Info
	hc      *http.Client
}

func New(repo, version, webDir string, r *jobs.Runner) *Updater {
	return &Updater{Repo: repo, Version: version, WebDir: webDir, jobs: r, info: Info{Current: version},
		hc: &http.Client{Timeout: 60 * time.Second}}
}

// AssetName renvoie le nom du binaire pour cette machine.
func AssetName() string {
	arch := runtime.GOARCH
	if arch == "arm" {
		arch = "armv6"
	}
	return "azmod-linux-" + arch
}

// Newer compare deux versions vX.Y.Z.
func Newer(latest, current string) bool {
	parse := func(v string) []int {
		v = strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v")
		var out []int
		for _, p := range strings.Split(v, ".") {
			n, err := strconv.Atoi(p)
			if err != nil {
				return nil
			}
			out = append(out, n)
		}
		return out
	}
	l, c := parse(latest), parse(current)
	if l == nil {
		return false
	}
	if c == nil {
		return true // version de développement
	}
	for i := 0; i < len(l) || i < len(c); i++ {
		var a, b int
		if i < len(l) {
			a = l[i]
		}
		if i < len(c) {
			b = c[i]
		}
		if a != b {
			return a > b
		}
	}
	return false
}

// Info renvoie le dernier résultat de vérification.
func (u *Updater) Info() Info {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.info
}

// Check interroge GitHub pour la dernière version.
func (u *Updater) Check() Info {
	rel, err := u.fetchLatest()
	u.mu.Lock()
	defer u.mu.Unlock()
	u.info.Checked = time.Now().Unix()
	if err != nil {
		u.info.Error = err.Error()
		return u.info
	}
	u.info.Error = ""
	u.info.Latest = rel
	u.info.Available = Newer(rel.Version, u.Version)
	return u.info
}

func (u *Updater) fetchLatest() (*Release, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+u.Repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AzmoOS/"+u.Version)
	resp, err := u.hc.Do(req)
	if err != nil {
		return nil, errors.New("impossible de joindre GitHub (connexion internet ?)")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, errors.New("aucune version publiée pour l'instant")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub a répondu %s", resp.Status)
	}
	var raw struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
		HTMLURL     string `json:"html_url"`
		Assets      []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	r := &Release{Version: raw.TagName, Name: raw.Name, Notes: raw.Body, Published: raw.PublishedAt, URL: raw.HTMLURL, assets: map[string]string{}}
	for _, a := range raw.Assets {
		r.assets[a.Name] = a.URL
	}
	return r, nil
}

func (u *Updater) download(url, dst string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "AzmoOS/"+u.Version)
	c := &http.Client{Timeout: 10 * time.Minute}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("téléchargement : %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func sha(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Apply télécharge, vérifie et installe la dernière version, puis redémarre le service.
func (u *Updater) Apply(onDone func()) error {
	info := u.Info()
	if info.Latest == nil {
		info = u.Check()
	}
	if info.Latest == nil {
		return errors.New("aucune version disponible")
	}
	rel := info.Latest
	asset := AssetName()
	binURL, ok := rel.assets[asset]
	if !ok {
		return fmt.Errorf("la version %s ne contient pas %s", rel.Version, asset)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.EvalSymlinks(exe)
	tmpDir, err := os.MkdirTemp("", "azmoos-update-")
	if err != nil {
		return err
	}
	newBin := filepath.Join(tmpDir, asset)
	steps := []jobs.Step{
		{Func: func(logf func(string, ...any)) error {
			logf("Téléchargement de %s %s…\n", asset, rel.Version)
			return u.download(binURL, newBin)
		}},
		{Func: func(logf func(string, ...any)) error {
			sumURL, ok := rel.assets["SHA256SUMS"]
			if !ok {
				logf("⚠ Pas de SHA256SUMS dans la version : vérification d'intégrité ignorée\n")
				return nil
			}
			sumFile := filepath.Join(tmpDir, "SHA256SUMS")
			if err := u.download(sumURL, sumFile); err != nil {
				return err
			}
			b, _ := os.ReadFile(sumFile)
			got, err := sha(newBin)
			if err != nil {
				return err
			}
			for _, l := range strings.Split(string(b), "\n") {
				f := strings.Fields(l)
				if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset {
					if f[0] != got {
						return errors.New("somme SHA-256 incorrecte : fichier corrompu ou altéré")
					}
					logf("✔ Intégrité vérifiée (SHA-256)\n")
					return nil
				}
			}
			return errors.New("binaire absent de SHA256SUMS")
		}},
		{Func: func(logf func(string, ...any)) error {
			_ = os.Chmod(newBin, 0o755)
			out, err := exec.Command(newBin, "-version").CombinedOutput()
			if err != nil {
				return fmt.Errorf("le nouveau binaire ne démarre pas : %v", err)
			}
			logf("✔ %s", out)
			return nil
		}},
		{Func: func(logf func(string, ...any)) error {
			webURL, ok := rel.assets["azmoos-web.tar.gz"]
			if !ok || u.WebDir == "" {
				return nil
			}
			logf("Mise à jour de l'interface web…\n")
			arch := filepath.Join(tmpDir, "web.tar.gz")
			if err := u.download(webURL, arch); err != nil {
				return err
			}
			return extractWeb(arch, u.WebDir)
		}},
		{Func: func(logf func(string, ...any)) error {
			staged := exe + ".new"
			b, err := os.ReadFile(newBin)
			if err != nil {
				return err
			}
			if err := os.WriteFile(staged, b, 0o755); err != nil {
				return err
			}
			_ = os.Rename(exe, exe+".old")
			if err := os.Rename(staged, exe); err != nil {
				_ = os.Rename(exe+".old", exe)
				return err
			}
			logf("✔ Azmo OS %s installé. Redémarrage du service…\n", rel.Version)
			return nil
		}},
	}
	return u.jobs.Start("updater", "azmoos", "update", steps, func(ok bool) {
		os.RemoveAll(tmpDir)
		if ok && onDone != nil {
			onDone()
		}
	})
}

func extractWeb(archive, dst string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	staging := dst + ".new"
	_ = os.RemoveAll(staging)
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(h.Name)
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
			return errors.New("archive web invalide")
		}
		p := filepath.Join(staging, name)
		if h.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(p, 0o755)
			continue
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		out, err := os.Create(p)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	_ = os.RemoveAll(dst + ".old")
	_ = os.Rename(dst, dst+".old")
	if err := os.Rename(staging, dst); err != nil {
		_ = os.Rename(dst+".old", dst)
		return err
	}
	return os.RemoveAll(dst + ".old")
}
