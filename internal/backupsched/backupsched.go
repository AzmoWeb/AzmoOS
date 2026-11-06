// Package backupsched exécute les sauvegardes planifiées vers un dossier (disque USB,
// partage réseau…) ou vers le cloud avec rclone.
package backupsched

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/backup"
	"github.com/azmoweb/azmoos/internal/cmdutil"
	"github.com/azmoweb/azmoos/internal/config"
	"github.com/azmoweb/azmoos/internal/events"
	"github.com/azmoweb/azmoos/internal/jobs"
	"github.com/azmoweb/azmoos/internal/notify"
)

const prefix = "azmoos-sauvegarde-"

type Scheduler struct {
	Config  *config.Store
	Backup  *backup.Manager
	Jobs    *jobs.Runner
	Events  *events.Log
	Notify  *notify.Center
	DataDir string
	// RcloneConfig est le fichier de configuration rclone utilisé par Azmo OS.
	RcloneConfig string
}

// File est une sauvegarde présente dans une destination.
type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Time int64  `json:"time"`
}

var nameRe = regexp.MustCompile(`^` + prefix + `\d{4}-\d{2}-\d{2}-\d{4}\.tar\.gz$`)

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isRclone(target string) bool { return strings.HasPrefix(target, "rclone:") }

// localDir convertit une destination locale (relative à /DATA) en chemin absolu sûr.
func (s *Scheduler) localDir(target string) (string, error) {
	p := filepath.Join(s.DataDir, filepath.Clean("/"+target))
	if p != s.DataDir && !strings.HasPrefix(p, s.DataDir+string(os.PathSeparator)) {
		return "", errors.New("destination hors de /DATA")
	}
	return p, nil
}

func remote(target string) string { return strings.TrimPrefix(target, "rclone:") }

func (s *Scheduler) rclone(args ...string) (string, error) {
	return cmdutil.RunTimeout(2*time.Hour, "rclone", append([]string{"--config", s.RcloneConfig}, args...)...)
}

// RcloneInfo indique si rclone est installé et liste les comptes cloud configurés.
func (s *Scheduler) RcloneInfo() (installed bool, remotes []string) {
	remotes = []string{}
	if !cmdutil.Has("rclone") {
		return false, remotes
	}
	out, err := s.rclone("listremotes")
	if err == nil {
		for _, l := range strings.Fields(out) {
			remotes = append(remotes, strings.TrimSuffix(l, ":"))
		}
	}
	return true, remotes
}

// List renvoie les sauvegardes planifiées.
func (s *Scheduler) List() []config.Backup {
	out := []config.Backup{}
	s.Config.View(func(c *config.Config) { out = append(out, c.Backups...) })
	return out
}

func validate(b *config.Backup) error {
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" {
		b.Name = "Sauvegarde"
	}
	if b.Frequency != "daily" && b.Frequency != "weekly" {
		return errors.New("fréquence invalide")
	}
	if b.Hour < 0 || b.Hour > 23 || b.Weekday < 0 || b.Weekday > 6 {
		return errors.New("heure ou jour invalide")
	}
	if b.Keep < 1 || b.Keep > 365 {
		return errors.New("le nombre de sauvegardes à garder doit être entre 1 et 365")
	}
	if b.Target == "" {
		return errors.New("choisis une destination")
	}
	if isRclone(b.Target) {
		r := remote(b.Target)
		if !regexp.MustCompile(`^[A-Za-z0-9_. -]+:[^\x00]*$`).MatchString(r) || strings.Contains(r, "..") {
			return errors.New("destination cloud invalide (exemple : rclone:gdrive:AzmoOS)")
		}
	}
	return nil
}

// Save crée ou met à jour une planification.
func (s *Scheduler) Save(b config.Backup) (config.Backup, error) {
	if err := validate(&b); err != nil {
		return b, err
	}
	if !isRclone(b.Target) {
		dir, err := s.localDir(b.Target)
		if err != nil {
			return b, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return b, fmt.Errorf("impossible de créer le dossier de destination : %w", err)
		}
	}
	err := s.Config.Update(func(c *config.Config) error {
		if b.ID == "" {
			b.ID = newID()
			c.Backups = append(c.Backups, b)
			return nil
		}
		for i := range c.Backups {
			if c.Backups[i].ID == b.ID {
				// On garde l'historique d'exécution.
				b.LastRun, b.LastOK, b.LastError, b.LastFile, b.LastSize = c.Backups[i].LastRun, c.Backups[i].LastOK, c.Backups[i].LastError, c.Backups[i].LastFile, c.Backups[i].LastSize
				c.Backups[i] = b
				return nil
			}
		}
		return errors.New("planification introuvable")
	})
	return b, err
}

// Delete supprime une planification (les fichiers déjà créés sont conservés).
func (s *Scheduler) Delete(id string) error {
	return s.Config.Update(func(c *config.Config) error {
		c.Backups = slices.DeleteFunc(c.Backups, func(b config.Backup) bool { return b.ID == id })
		return nil
	})
}

func (s *Scheduler) get(id string) (config.Backup, bool) {
	for _, b := range s.List() {
		if b.ID == id {
			return b, true
		}
	}
	return config.Backup{}, false
}

// LastDue renvoie la dernière échéance passée (ou zéro).
func LastDue(b config.Backup, now time.Time) time.Time {
	t := time.Date(now.Year(), now.Month(), now.Day(), b.Hour, 0, 0, 0, now.Location())
	if b.Frequency == "weekly" {
		t = t.AddDate(0, 0, -((int(t.Weekday()) - b.Weekday + 7) % 7))
		if t.After(now) {
			t = t.AddDate(0, 0, -7)
		}
		return t
	}
	if t.After(now) {
		t = t.AddDate(0, 0, -1)
	}
	return t
}

// Next renvoie la prochaine échéance.
func Next(b config.Backup, now time.Time) time.Time {
	if b.Frequency == "weekly" {
		return LastDue(b, now).AddDate(0, 0, 7)
	}
	return LastDue(b, now).AddDate(0, 0, 1)
}

// Due indique si une sauvegarde doit être lancée (y compris une échéance manquée, Pi éteint).
func Due(b config.Backup, now time.Time) bool {
	if !b.Enabled {
		return false
	}
	d := LastDue(b, now)
	return b.LastRun < d.Unix() && now.Sub(d) < 7*24*time.Hour+time.Hour
}

// Loop vérifie chaque minute les sauvegardes à lancer.
func (s *Scheduler) Loop() {
	time.Sleep(90 * time.Second)
	for {
		now := time.Now()
		for _, b := range s.List() {
			if Due(b, now) && !s.Jobs.Running("backup:"+b.ID) {
				_ = s.Run(b.ID, "")
			}
		}
		time.Sleep(time.Minute)
	}
}

func (s *Scheduler) record(id string, ok bool, errMsg, file string, size int64) {
	_ = s.Config.Update(func(c *config.Config) error {
		for i := range c.Backups {
			if c.Backups[i].ID == id {
				c.Backups[i].LastRun, c.Backups[i].LastOK, c.Backups[i].LastError = time.Now().Unix(), ok, errMsg
				if ok {
					c.Backups[i].LastFile, c.Backups[i].LastSize = file, size
				}
			}
		}
		return nil
	})
}

// Run lance une sauvegarde maintenant (en tâche de fond).
func (s *Scheduler) Run(id, user string) error {
	b, ok := s.get(id)
	if !ok {
		return errors.New("planification introuvable")
	}
	name := prefix + time.Now().Format("2006-01-02-1504") + ".tar.gz"
	var size int64
	var runErr error
	steps := []jobs.Step{{Func: func(logf func(string, ...any)) error {
		runErr = s.execute(b, name, &size, logf)
		return runErr
	}}}
	return s.Jobs.Start("backup", b.ID, "backup", steps, func(ok bool) {
		lang := s.Notify.Settings().Lang
		L := func(fr, en string) string { return notify.T(lang, fr, en) }
		if ok {
			s.record(b.ID, true, "", name, size)
			s.Events.Add(events.Info, "system", user, "", fmt.Sprintf("Sauvegarde « %s » réussie : %s", b.Name, name))
			s.Notify.Resolve("backup:" + b.ID)
			if s.Notify.Settings().Backups {
				s.Notify.Raise("", "backups", notify.Info, L("Sauvegarde réussie", "Backup completed"),
					fmt.Sprintf(L("« %s » : %s enregistrée.", "“%s”: %s saved."), b.Name, name))
			}
			return
		}
		msg := "erreur inconnue"
		if runErr != nil {
			msg = runErr.Error()
		}
		s.record(b.ID, false, msg, "", 0)
		s.Events.Add(events.Error, "system", user, "", fmt.Sprintf("Échec de la sauvegarde « %s » : %s", b.Name, msg))
		if s.Notify.Settings().Backups {
			s.Notify.Raise("backup:"+b.ID, "backups", notify.Warning, L("La sauvegarde a échoué", "Backup failed"),
				fmt.Sprintf(L("« %s » n'a pas pu être enregistrée : %s", "“%s” could not be saved: %s"), b.Name, msg))
		}
	})
}

func (s *Scheduler) execute(b config.Backup, name string, size *int64, logf func(string, ...any)) error {
	var tmpDir string
	if isRclone(b.Target) {
		tmpDir = filepath.Join(s.DataDir, "Backups", ".envoi")
	} else {
		d, err := s.localDir(b.Target)
		if err != nil {
			return err
		}
		tmpDir = d
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("destination inaccessible (disque débranché ?) : %w", err)
	}
	tmp := filepath.Join(tmpDir, "."+name+".partiel")
	logf("Création de %s…\n", name)
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("écriture impossible (disque débranché ou en lecture seule ?) : %w", err)
	}
	if err := s.Backup.Write(f, b.AppData); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	st, _ := os.Stat(tmp)
	*size = st.Size()
	logf("Archive prête (%d octets).\n", *size)

	if isRclone(b.Target) {
		defer os.Remove(tmp)
		dest := strings.TrimRight(remote(b.Target), "/") + "/" + name
		logf("Envoi vers %s…\n", dest)
		if out, err := s.rclone("copyto", tmp, dest); err != nil {
			return fmt.Errorf("envoi vers le cloud impossible : %v %s", err, out)
		}
		return s.pruneRemote(b, logf)
	}
	final := filepath.Join(tmpDir, name)
	if err := os.Rename(tmp, final); err != nil {
		return err
	}
	return s.pruneLocal(b, tmpDir, logf)
}

func keepNewest(names []string, keep int) []string {
	sort.Strings(names) // le nom contient la date : ordre chronologique
	if len(names) <= keep {
		return nil
	}
	return names[:len(names)-keep]
}

func (s *Scheduler) pruneLocal(b config.Backup, dir string, logf func(string, ...any)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if nameRe.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	for _, n := range keepNewest(names, b.Keep) {
		logf("Suppression de l'ancienne sauvegarde %s\n", n)
		_ = os.Remove(filepath.Join(dir, n))
	}
	return nil
}

func (s *Scheduler) pruneRemote(b config.Backup, logf func(string, ...any)) error {
	base := strings.TrimRight(remote(b.Target), "/")
	out, err := s.rclone("lsf", base, "--files-only", "--include", prefix+"*.tar.gz")
	if err != nil {
		return nil // l'envoi a réussi : un échec du ménage n'est pas bloquant
	}
	var names []string
	for _, l := range strings.Split(out, "\n") {
		if nameRe.MatchString(strings.TrimSpace(l)) {
			names = append(names, strings.TrimSpace(l))
		}
	}
	for _, n := range keepNewest(names, b.Keep) {
		logf("Suppression de l'ancienne sauvegarde %s\n", n)
		_, _ = s.rclone("deletefile", base+"/"+n)
	}
	return nil
}

// Files liste les sauvegardes présentes dans la destination.
func (s *Scheduler) Files(id string) ([]File, error) {
	b, ok := s.get(id)
	if !ok {
		return nil, errors.New("planification introuvable")
	}
	out := []File{}
	if isRclone(b.Target) {
		raw, err := s.rclone("lsjson", strings.TrimRight(remote(b.Target), "/"), "--files-only")
		if err != nil {
			return nil, fmt.Errorf("lecture du cloud impossible : %v", err)
		}
		var items []struct {
			Name    string
			Size    int64
			ModTime time.Time
		}
		_ = json.Unmarshal([]byte(raw), &items)
		for _, it := range items {
			if nameRe.MatchString(it.Name) {
				out = append(out, File{it.Name, it.Size, it.ModTime.Unix()})
			}
		}
	} else {
		dir, err := s.localDir(b.Target)
		if err != nil {
			return nil, err
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if !nameRe.MatchString(e.Name()) {
				continue
			}
			if info, err := e.Info(); err == nil {
				out = append(out, File{e.Name(), info.Size(), info.ModTime().Unix()})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out, nil
}

// Open ouvre une sauvegarde de la destination (téléchargée si elle est dans le cloud).
func (s *Scheduler) Open(id, file string) (*os.File, func(), error) {
	b, ok := s.get(id)
	if !ok || !nameRe.MatchString(file) {
		return nil, nil, errors.New("sauvegarde introuvable")
	}
	if !isRclone(b.Target) {
		dir, err := s.localDir(b.Target)
		if err != nil {
			return nil, nil, err
		}
		f, err := os.Open(filepath.Join(dir, file))
		return f, func() {}, err
	}
	tmp := filepath.Join(os.TempDir(), "azmoos-restauration-"+file)
	cmd := exec.Command("rclone", "--config", s.RcloneConfig, "copyto", strings.TrimRight(remote(b.Target), "/")+"/"+file, tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("téléchargement depuis le cloud impossible : %s", strings.TrimSpace(string(out)))
	}
	f, err := os.Open(tmp)
	return f, func() { os.Remove(tmp) }, err
}
