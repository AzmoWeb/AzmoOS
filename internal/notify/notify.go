// Package notify gère les alertes d'Azmo OS : centre de notifications dans
// l'interface, et envoi vers le téléphone (ntfy, Discord, Telegram).
package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/azmoweb/azmoos/internal/config"
)

// Niveaux.
const (
	Info     = "info"
	Warning  = "warning"
	Critical = "critical"
)

var levelRank = map[string]int{Info: 0, Warning: 1, Critical: 2}

type Notification struct {
	ID      int64  `json:"id"`
	Time    int64  `json:"time"`
	Level   string `json:"level"`
	Kind    string `json:"kind"` // temperature, disk, memory, apps, updates, security, backups, test
	Key     string `json:"key"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Read    bool   `json:"read"`
}

type state struct {
	Items  []Notification   `json:"items"`
	Active map[string]int64 `json:"active"` // alertes en cours (clé → date), pour ne pas répéter
	NextID int64            `json:"nextId"`
}

// Center conserve les notifications et les envoie.
type Center struct {
	mu   sync.Mutex
	path string
	st   state
	cfg  *config.Store
	hc   *http.Client
	host string
}

const maxItems = 300

func New(cfg *config.Store) *Center {
	c := &Center{path: filepath.Join(cfg.Dir(), "notifications.json"), cfg: cfg, hc: &http.Client{Timeout: 15 * time.Second}}
	c.host, _ = os.Hostname()
	if b, err := os.ReadFile(c.path); err == nil {
		_ = json.Unmarshal(b, &c.st)
	}
	if c.st.Active == nil {
		c.st.Active = map[string]int64{}
	}
	return c
}

func (c *Center) save() {
	b, _ := json.Marshal(&c.st)
	tmp := c.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, c.path)
	}
}

// Settings renvoie la configuration des alertes.
func (c *Center) Settings() config.Notify {
	var n config.Notify
	c.cfg.View(func(cfg *config.Config) { n = *cfg.Notify })
	return n
}

// Raise crée une alerte, sauf si la même (clé) est déjà en cours : on ne harcèle pas.
// Renvoie true si une notification a été créée.
func (c *Center) Raise(key, kind, level, title, message string) bool {
	c.mu.Lock()
	if _, active := c.st.Active[key]; active && key != "" {
		c.mu.Unlock()
		return false
	}
	c.st.NextID++
	n := Notification{ID: c.st.NextID, Time: time.Now().Unix(), Level: level, Kind: kind, Key: key, Title: title, Message: message}
	c.st.Items = append(c.st.Items, n)
	if len(c.st.Items) > maxItems {
		c.st.Items = c.st.Items[len(c.st.Items)-maxItems:]
	}
	if key != "" {
		c.st.Active[key] = n.Time
	}
	c.save()
	c.mu.Unlock()

	s := c.Settings()
	if levelRank[level] >= levelRank[s.PushLevel] {
		go func() {
			for name, err := range c.push(s, n) {
				if err != nil {
					log.Printf("notification %s : %v", name, err)
				}
			}
		}()
	}
	return true
}

// Resolve marque une alerte comme terminée : elle pourra se redéclencher plus tard.
func (c *Center) Resolve(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.st.Active[key]; ok {
		delete(c.st.Active, key)
		c.save()
	}
}

// Active indique si une alerte est en cours.
func (c *Center) Active(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.st.Active[key]
	return ok
}

// List renvoie les notifications, les plus récentes d'abord.
func (c *Center) List() (items []Notification, unread int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	items = make([]Notification, 0, len(c.st.Items))
	for i := len(c.st.Items) - 1; i >= 0; i-- {
		items = append(items, c.st.Items[i])
		if !c.st.Items[i].Read {
			unread++
		}
	}
	return
}

// MarkRead marque une notification (ou toutes si id = 0) comme lue.
func (c *Center) MarkRead(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.st.Items {
		if id == 0 || c.st.Items[i].ID == id {
			c.st.Items[i].Read = true
		}
	}
	c.save()
}

// Clear supprime toutes les notifications lues.
func (c *Center) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	kept := c.st.Items[:0]
	for _, n := range c.st.Items {
		if !n.Read {
			kept = append(kept, n)
		}
	}
	c.st.Items = kept
	c.save()
}

// ---------- Canaux ----------

var emoji = map[string]string{Info: "ℹ️", Warning: "⚠️", Critical: "🚨"}

func (c *Center) push(s config.Notify, n Notification) map[string]error {
	res := map[string]error{}
	title := fmt.Sprintf("%s %s — %s", emoji[n.Level], n.Title, c.host)
	if s.Ntfy.Enabled {
		res["ntfy"] = c.sendNtfy(s.Ntfy, n, title)
	}
	if s.Discord.Enabled {
		res["discord"] = c.sendDiscord(s.Discord, n, title)
	}
	if s.Telegram.Enabled {
		res["telegram"] = c.sendTelegram(s.Telegram, n, title)
	}
	return res
}

func (c *Center) postJSON(url string, body any, header map[string]string) error {
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AzmoOS")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return errors.New("service injoignable (connexion internet ?)")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"description"`
			Message     string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		msg := strings.TrimSpace(e.Error + " " + e.Description + " " + e.Message)
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("refusé : %s", msg)
	}
	return nil
}

func (c *Center) sendNtfy(ch config.NtfyChannel, n Notification, title string) error {
	if ch.Topic == "" {
		return errors.New("sujet (topic) ntfy manquant")
	}
	server := strings.TrimRight(ch.Server, "/")
	if server == "" {
		server = "https://ntfy.sh"
	}
	prio := map[string]int{Info: 3, Warning: 4, Critical: 5}[n.Level]
	tags := map[string][]string{"temperature": {"thermometer"}, "disk": {"floppy_disk"}, "memory": {"brain"}, "apps": {"package"},
		"updates": {"gift"}, "security": {"lock"}, "backups": {"card_file_box"}, "test": {"wave"}}[n.Kind]
	h := map[string]string{}
	if ch.Token != "" {
		h["Authorization"] = "Bearer " + ch.Token
	}
	// Publication JSON : les accents et emojis passent sans problème d'encodage.
	return c.postJSON(server, map[string]any{"topic": ch.Topic, "title": title, "message": n.Message, "priority": prio, "tags": tags}, h)
}

func (c *Center) sendDiscord(ch config.DiscordChannel, n Notification, title string) error {
	if !strings.HasPrefix(ch.Webhook, "https://") {
		return errors.New("adresse du webhook Discord invalide")
	}
	color := map[string]int{Info: 0x8b7bff, Warning: 0xf6b44b, Critical: 0xf2706b}[n.Level]
	return c.postJSON(ch.Webhook, map[string]any{
		"username": "Azmo OS",
		"embeds":   []map[string]any{{"title": title, "description": n.Message, "color": color, "footer": map[string]string{"text": "Made by Azmo"}}},
	}, nil)
}

func (c *Center) sendTelegram(ch config.TelegramChannel, n Notification, title string) error {
	if ch.Token == "" || ch.ChatID == "" {
		return errors.New("jeton du bot ou identifiant de discussion Telegram manquant")
	}
	text := "<b>" + html.EscapeString(title) + "</b>\n" + html.EscapeString(n.Message)
	return c.postJSON("https://api.telegram.org/bot"+ch.Token+"/sendMessage", map[string]any{"chat_id": ch.ChatID, "text": text, "parse_mode": "HTML"}, nil)
}

// Test envoie une notification d'essai sur chaque canal actif et renvoie le résultat par canal.
func (c *Center) Test() map[string]string {
	s := c.Settings()
	n := Notification{Time: time.Now().Unix(), Level: Info, Kind: "test",
		Title: T(s.Lang, "Test réussi", "Test successful"), Message: T(s.Lang, "Les notifications d'Azmo OS arrivent bien sur ce canal. 🎉", "Azmo OS notifications reach this channel. 🎉")}
	out := map[string]string{}
	res := c.push(s, n)
	names := make([]string, 0, len(res))
	for k := range res {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if res[k] != nil {
			out[k] = res[k].Error()
		} else {
			out[k] = "ok"
		}
	}
	return out
}

// T choisit le texte dans la langue des notifications.
func T(lang, fr, en string) string {
	if lang == "en" {
		return en
	}
	return fr
}
