// Package backup sauvegarde et restaure la configuration d'Azmo OS et des apps.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	Product  string   `json:"product"`
	Version  string   `json:"version"`
	Created  int64    `json:"created"`
	Hostname string   `json:"hostname"`
	AppData  bool     `json:"appData"`
	Apps     []string `json:"apps"`
}

type Manager struct {
	ConfDir string // /etc/azmoos
	AppsDir string // /DATA/AppData/.azmoos/apps
	DataDir string // /DATA
	Version string
}

// Write écrit une sauvegarde .tar.gz dans w.
func (m *Manager) Write(w io.Writer, withAppData bool) error {
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	host, _ := os.Hostname()
	man := Manifest{Product: "AzmoOS", Version: m.Version, Created: time.Now().Unix(), Hostname: host, AppData: withAppData, Apps: []string{}}
	entries, _ := os.ReadDir(m.AppsDir)
	for _, e := range entries {
		if e.IsDir() {
			man.Apps = append(man.Apps, e.Name())
		}
	}
	mb, _ := json.MarshalIndent(man, "", "  ")
	if err := writeBytes(tw, "manifest.json", mb); err != nil {
		return err
	}
	if err := addFile(tw, filepath.Join(m.ConfDir, "config.json"), "config/config.json"); err != nil {
		return err
	}
	if err := addTree(tw, m.AppsDir, "apps"); err != nil {
		return err
	}
	if withAppData {
		appData := filepath.Join(m.DataDir, "AppData")
		for _, id := range man.Apps {
			if err := addTree(tw, filepath.Join(appData, id), "appdata/"+id); err != nil {
				return err
			}
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func writeBytes(tw *tar.Writer, name string, b []byte) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(b)), ModTime: time.Now()}); err != nil {
		return err
	}
	_, err := tw.Write(b)
	return err
}

func addFile(tw *tar.Writer, src, name string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeBytes(tw, name, b)
}

func addTree(tw *tar.Writer, root, prefix string) error {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // on ignore les fichiers illisibles
		}
		rel, _ := filepath.Rel(root, p)
		name := filepath.ToSlash(filepath.Join(prefix, rel))
		info, err := d.Info()
		if err != nil {
			return nil
		}
		switch {
		case d.IsDir():
			return tw.WriteHeader(&tar.Header{Name: name + "/", Mode: int64(info.Mode().Perm()), Typeflag: tar.TypeDir, ModTime: info.ModTime()})
		case info.Mode().IsRegular():
			hdr, _ := tar.FileInfoHeader(info, "")
			hdr.Name = name
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			f, err := os.Open(p)
			if err != nil {
				return nil
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		case info.Mode()&fs.ModeSymlink != 0:
			target, _ := os.Readlink(p)
			hdr := &tar.Header{Name: name, Typeflag: tar.TypeSymlink, Linkname: target, ModTime: info.ModTime(), Mode: 0o777}
			return tw.WriteHeader(hdr)
		}
		return nil
	})
}

// Restore lit une sauvegarde et remet en place config, apps et données.
func (m *Manager) Restore(r io.Reader) (*Manifest, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.New("fichier de sauvegarde invalide")
	}
	tr := tar.NewReader(gz)
	staging, err := os.MkdirTemp(m.ConfDir, "restore-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("archive corrompue : %w", err)
		}
		name := filepath.Clean(h.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return nil, errors.New("archive suspecte (chemin interdit)")
		}
		dst := filepath.Join(staging, name)
		switch h.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(dst, 0o755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777|0o600)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return nil, err
			}
			f.Close()
		case tar.TypeSymlink:
			if filepath.IsAbs(h.Linkname) || strings.Contains(h.Linkname, "..") {
				continue // on n'accepte que les liens internes
			}
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			_ = os.Symlink(h.Linkname, dst)
		}
	}
	var man Manifest
	b, err := os.ReadFile(filepath.Join(staging, "manifest.json"))
	if err != nil || json.Unmarshal(b, &man) != nil || man.Product != "AzmoOS" {
		return nil, errors.New("ce fichier n'est pas une sauvegarde Azmo OS")
	}
	cfg := filepath.Join(staging, "config", "config.json")
	if _, err := os.Stat(cfg); err == nil {
		if err := copyFile(cfg, filepath.Join(m.ConfDir, "config.json"), 0o600); err != nil {
			return nil, err
		}
	}
	if err := replaceTree(filepath.Join(staging, "apps"), m.AppsDir); err != nil {
		return nil, err
	}
	if man.AppData {
		ad := filepath.Join(staging, "appdata")
		entries, _ := os.ReadDir(ad)
		for _, e := range entries {
			if err := replaceTree(filepath.Join(ad, e.Name()), filepath.Join(m.DataDir, "AppData", e.Name())); err != nil {
				return nil, err
			}
		}
	}
	return &man, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// replaceTree remplace dst par src (via renommage quand c'est possible).
func replaceTree(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(dst), 0o755)
	old := dst + ".avant-restauration"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(dst); err == nil {
		if err := os.Rename(dst, old); err != nil {
			return err
		}
	}
	if err := os.Rename(src, dst); err != nil {
		// Systèmes de fichiers différents : copie récursive.
		if err := copyTree(src, dst); err != nil {
			return err
		}
	}
	return os.RemoveAll(old)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		t := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(t, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			l, _ := os.Readlink(p)
			return os.Symlink(l, t)
		}
		info, _ := d.Info()
		return copyFile(p, t, info.Mode().Perm())
	})
}
