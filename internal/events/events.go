// Package events tient le journal des événements d'Azmo OS.
package events

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Niveaux.
const (
	Info    = "info"
	Warning = "warning"
	Error   = "error"
)

type Event struct {
	Time     int64  `json:"time"`
	Level    string `json:"level"`
	Category string `json:"category"` // auth, apps, system, storage, network, security, files
	User     string `json:"user,omitempty"`
	IP       string `json:"ip,omitempty"`
	Message  string `json:"message"`
}

const (
	maxMem  = 2000
	maxFile = 5 << 20
)

type Log struct {
	mu   sync.Mutex
	path string
	mem  []Event
}

func Open(dir string) *Log {
	l := &Log{path: filepath.Join(dir, "events.jsonl")}
	if f, err := os.Open(l.path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var e Event
			if json.Unmarshal(sc.Bytes(), &e) == nil {
				l.push(e)
			}
		}
		f.Close()
	}
	return l
}

func (l *Log) push(e Event) {
	l.mem = append(l.mem, e)
	if len(l.mem) > maxMem {
		l.mem = l.mem[len(l.mem)-maxMem:]
	}
}

// Add enregistre un événement.
func (l *Log) Add(level, category, user, ip, msg string) {
	e := Event{Time: time.Now().Unix(), Level: level, Category: category, User: user, IP: ip, Message: msg}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.push(e)
	if st, err := os.Stat(l.path); err == nil && st.Size() > maxFile {
		_ = os.Rename(l.path, l.path+".1")
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	b, _ := json.Marshal(e)
	_, _ = f.Write(append(b, '\n'))
	f.Close()
}

// List renvoie les événements les plus récents d'abord, filtrés par catégorie.
func (l *Log) List(category string, limit int) []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := []Event{}
	for i := len(l.mem) - 1; i >= 0 && len(out) < limit; i-- {
		if category == "" || l.mem[i].Category == category {
			out = append(out, l.mem[i])
		}
	}
	return out
}
