// Package files fournit un gestionnaire de fichiers confiné à la racine des données.
package files

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Dir     bool      `json:"dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type FS struct{ root string }

func New(root string) *FS { return &FS{root: filepath.Clean(root)} }

var ErrOutside = errors.New("chemin hors de la zone autorisée")

// Resolve convertit un chemin relatif en chemin absolu sûr.
func (f *FS) Resolve(rel string) (string, error) {
	p := filepath.Join(f.root, filepath.Clean("/"+rel))
	if p != f.root && !strings.HasPrefix(p, f.root+string(os.PathSeparator)) {
		return "", ErrOutside
	}
	// Refuse les liens symboliques qui sortent de la racine.
	if real, err := filepath.EvalSymlinks(p); err == nil {
		rootReal, _ := filepath.EvalSymlinks(f.root)
		if real != rootReal && !strings.HasPrefix(real, rootReal+string(os.PathSeparator)) {
			return "", ErrOutside
		}
	}
	return p, nil
}

func (f *FS) rel(abs string) string {
	r, _ := filepath.Rel(f.root, abs)
	return "/" + filepath.ToSlash(strings.TrimPrefix(r, "."))
}

func (f *FS) List(rel string) ([]Entry, error) {
	p, err := f.Resolve(rel)
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, d := range des {
		info, err := d.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{Name: d.Name(), Path: f.rel(filepath.Join(p, d.Name())), Dir: info.IsDir(), Size: info.Size(), ModTime: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func validName(n string) bool {
	return n != "" && n != "." && n != ".." && !strings.ContainsAny(n, "/\\\x00")
}

func (f *FS) Mkdir(dir, name string) error {
	if !validName(name) {
		return errors.New("nom invalide")
	}
	p, err := f.Resolve(dir + "/" + name)
	if err != nil {
		return err
	}
	return os.Mkdir(p, 0o755)
}

func (f *FS) Rename(path, name string) error {
	if !validName(name) {
		return errors.New("nom invalide")
	}
	src, err := f.Resolve(path)
	if err != nil || src == f.root {
		return ErrOutside
	}
	return os.Rename(src, filepath.Join(filepath.Dir(src), name))
}

func (f *FS) Delete(path string) error {
	p, err := f.Resolve(path)
	if err != nil {
		return err
	}
	if p == f.root {
		return errors.New("impossible de supprimer la racine")
	}
	return os.RemoveAll(p)
}

// SaveStream enregistre un fichier envoyé (sans le charger en mémoire).
func (f *FS) SaveStream(dir, filename string, src io.Reader) error {
	name := filepath.Base(filename)
	if !validName(name) {
		return errors.New("nom invalide")
	}
	p, err := f.Resolve(dir + "/" + name)
	if err != nil {
		return err
	}
	tmp := p + ".azmo-upload"
	dst, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmp)
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Move déplace un fichier ou dossier dans un autre dossier.
func (f *FS) Move(path, toDir string) error {
	src, err := f.Resolve(path)
	if err != nil || src == f.root {
		return ErrOutside
	}
	dst, err := f.Resolve(toDir)
	if err != nil {
		return err
	}
	if st, err := os.Stat(dst); err != nil || !st.IsDir() {
		return errors.New("le dossier de destination n'existe pas")
	}
	if strings.HasPrefix(dst+string(os.PathSeparator), src+string(os.PathSeparator)) {
		return errors.New("impossible de déplacer un dossier dans lui-même")
	}
	target := filepath.Join(dst, filepath.Base(src))
	if _, err := os.Stat(target); err == nil {
		return errors.New("un élément du même nom existe déjà à destination")
	}
	return os.Rename(src, target)
}
