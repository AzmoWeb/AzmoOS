package store

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/azmoweb/azmoos/internal/cmdutil"
	"github.com/azmoweb/azmoos/internal/jobs"
)

// Résultats possibles d'AutoUpdate.
const (
	UpdUpToDate = "uptodate" // aucune nouvelle image
	UpdUpdated  = "updated"  // nouvelles images, conteneurs recréés
	UpdSkipped  = "skipped"  // app arrêtée : on n'y touche pas
)

// AutoUpdate cherche de nouvelles images pour une app en marche et, s'il y en a,
// recrée ses conteneurs. Contrairement à Update, l'appel attend la fin, et une app
// déjà à jour n'est pas redémarrée. Une app arrêtée est laissée telle quelle.
func (s *Store) AutoUpdate(id string) (string, error) {
	if !idRe.MatchString(id) || !s.isInstalled(id) {
		return "", errors.New("application non installée")
	}
	result := UpdUpToDate
	done := make(chan bool, 1)
	step := jobs.Step{Func: func(logf func(string, ...any)) error {
		dk := func(args ...string) ([]string, error) {
			logf("$ docker %s\n", strings.Join(args, " "))
			out, err := cmdutil.RunTimeout(30*time.Minute, "docker", args...)
			if err != nil {
				return nil, err
			}
			return strings.Fields(out), nil
		}
		running, err := dk(s.composeArgs(id, "ps", "-q")...)
		if err != nil {
			return err
		}
		if len(running) == 0 {
			result = UpdSkipped
			logf("L'application est arrêtée : on la laisse comme elle est.\n")
			return nil
		}
		containers, err := dk(s.composeArgs(id, "ps", "-aq")...)
		if err != nil {
			return err
		}
		before, err := dk(append([]string{"inspect", "--format", "{{.Image}}"}, containers...)...)
		if err != nil {
			return err
		}
		if _, err := dk(s.composeArgs(id, "pull", "--quiet")...); err != nil {
			return err
		}
		refs, err := dk(s.composeArgs(id, "config", "--images")...)
		if err != nil {
			return err
		}
		if len(refs) == 0 {
			return nil
		}
		after, err := dk(append([]string{"image", "inspect", "--format", "{{.Id}}"}, refs...)...)
		if err != nil {
			return err
		}
		changed := false
		for _, img := range after {
			if !slices.Contains(before, img) {
				changed = true
			}
		}
		if !changed {
			logf("Déjà à jour ✓\n")
			return nil
		}
		if _, err := dk(s.composeArgs(id, "up", "-d", "--remove-orphans")...); err != nil {
			return err
		}
		_, _ = dk("image", "prune", "-f")
		result = UpdUpdated
		logf("Mise à jour installée ✓\n")
		return nil
	}}
	var jobErr error
	if err := s.jobs.Start("app", id, "auto-update", []jobs.Step{step}, func(ok bool) { done <- ok }); err != nil {
		return "", err
	}
	if !<-done {
		jobErr = errors.New("la mise à jour a échoué (voir le journal de l'app)")
		if j, ok := s.jobs.Get("app:" + id); ok {
			if l := lastErrLine(j.Log); l != "" {
				jobErr = errors.New(l)
			}
		}
		return "", friendly(jobErr)
	}
	return result, nil
}

// friendly remplace les erreurs Docker courantes par une explication.
func friendly(err error) error {
	m := err.Error()
	switch {
	case strings.Contains(m, "429 Too Many Requests") || strings.Contains(m, "toomanyrequests"):
		return errors.New("Docker Hub limite les téléchargements pour le moment ; nouvel essai au prochain passage")
	case strings.Contains(m, "Cannot connect to the Docker daemon"):
		return errors.New("Docker ne répond pas")
	case strings.Contains(m, "no such host") || strings.Contains(m, "i/o timeout") || strings.Contains(m, "network is unreachable"):
		return errors.New("pas d'accès à Internet pour télécharger la nouvelle version")
	case strings.Contains(m, "no space left on device"):
		return errors.New("plus assez de place sur le disque pour télécharger la nouvelle version")
	}
	return err
}

// lastErrLine renvoie la dernière ligne utile d'un journal (le message d'erreur).
func lastErrLine(log string) string {
	lines := strings.Split(strings.TrimSpace(log), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "✘ Erreur :"))
		if l != "" && !strings.HasPrefix(l, "$ ") {
			if len(l) > 300 {
				l = l[:300] + "…"
			}
			return l
		}
	}
	return ""
}
