// Package config gère la configuration persistante d'AzmoOS.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// User est le compte administrateur.
type User struct {
	Username string `json:"username"`
	Hash     string `json:"hash"`
	Salt     string `json:"salt"`
}

// Config est le contenu de config.json.
type Config struct {
	User     *User             `json:"user,omitempty"`
	Sessions map[string]int64  `json:"sessions,omitempty"` // token -> expiration (unix)
	Settings map[string]string `json:"settings,omitempty"`
}

// Store protège la config et la sauvegarde sur disque.
type Store struct {
	mu   sync.RWMutex
	path string
	c    Config
}

// Load lit (ou crée) la configuration.
func Load(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "config.json")}
	b, err := os.ReadFile(s.path)
	if err == nil {
		if err := json.Unmarshal(b, &s.c); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.c.Sessions == nil {
		s.c.Sessions = map[string]int64{}
	}
	if s.c.Settings == nil {
		s.c.Settings = map[string]string{}
	}
	return s, nil
}

// View exécute fn en lecture.
func (s *Store) View(fn func(c *Config)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(&s.c)
}

// Update exécute fn puis sauvegarde.
func (s *Store) Update(fn func(c *Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.c)
	b, err := json.MarshalIndent(&s.c, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
