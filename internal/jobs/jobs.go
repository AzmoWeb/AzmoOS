// Package jobs exécute des tâches longues en arrière-plan avec journal.
package jobs

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

type Job struct {
	ID     string    `json:"id"`     // clé unique (ex: "app:jellyfin", "system:upgrade")
	Kind   string    `json:"kind"`   // app, system, storage, updater
	Target string    `json:"target"` // ex: jellyfin
	Action string    `json:"action"`
	Status string    `json:"status"` // running, done, error
	Log    string    `json:"log"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end,omitempty"`
}

// Step est une étape : une commande, ou une fonction Go.
type Step struct {
	Cmd  []string
	Env  []string
	Func func(logf func(string, ...any)) error
}

type Runner struct {
	mu   sync.Mutex
	jobs map[string]*Job
	// OnFinish est appelé à la fin de chaque tâche (journalisation).
	OnFinish func(j Job)
}

func NewRunner() *Runner { return &Runner{jobs: map[string]*Job{}} }

var ErrBusy = errors.New("une opération est déjà en cours")

// Running indique si une tâche est en cours pour cette clé.
func (r *Runner) Running(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.jobs[id]
	return ok && j.Status == "running"
}

// Start lance les étapes dans l'ordre ; s'arrête à la première erreur.
func (r *Runner) Start(kind, target, action string, steps []Step, after func(ok bool)) error {
	id := kind + ":" + target
	r.mu.Lock()
	if j, ok := r.jobs[id]; ok && j.Status == "running" {
		r.mu.Unlock()
		return ErrBusy
	}
	j := &Job{ID: id, Kind: kind, Target: target, Action: action, Status: "running", Start: time.Now()}
	r.jobs[id] = j
	r.mu.Unlock()

	logf := func(f string, a ...any) {
		r.mu.Lock()
		j.Log += fmt.Sprintf(f, a...)
		if len(j.Log) > 256<<10 {
			j.Log = "…\n" + j.Log[len(j.Log)-200<<10:]
		}
		r.mu.Unlock()
	}
	go func() {
		ok := true
		for _, s := range steps {
			var err error
			if s.Func != nil {
				err = s.Func(logf)
			} else {
				logf("$ %s\n", strings.Join(s.Cmd, " "))
				cmd := exec.Command(s.Cmd[0], s.Cmd[1:]...)
				if s.Env != nil {
					cmd.Env = append(cmd.Environ(), s.Env...)
				}
				w := writerFunc(func(p []byte) { logf("%s", p) })
				cmd.Stdout, cmd.Stderr = w, w
				err = cmd.Run()
			}
			if err != nil {
				logf("\n✘ Erreur : %v\n", err)
				ok = false
				break
			}
		}
		if after != nil {
			after(ok)
		}
		r.mu.Lock()
		j.End = time.Now()
		if ok {
			j.Status = "done"
		} else {
			j.Status = "error"
		}
		cp := *j
		r.mu.Unlock()
		if r.OnFinish != nil {
			r.OnFinish(cp)
		}
	}()
	return nil
}

// List renvoie les tâches, les plus récentes d'abord. Les tâches terminées depuis
// plus d'une heure sont oubliées.
func (r *Runner) List() []Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Job{}
	for k, j := range r.jobs {
		if j.Status != "running" && time.Since(j.End) > time.Hour {
			delete(r.jobs, k)
			continue
		}
		out = append(out, *j)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Start.After(out[k].Start) })
	return out
}

// Get renvoie une tâche.
func (r *Runner) Get(id string) (Job, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.jobs[id]
	if !ok {
		return Job{}, false
	}
	return *j, true
}

type writerFunc func(p []byte)

func (w writerFunc) Write(p []byte) (int, error) { w(p); return len(p), nil }
