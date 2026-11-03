// Package terminal fournit un terminal web : un shell dans un pseudo-terminal,
// relié au navigateur par WebSocket.
package terminal

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Account décrit l'utilisateur système sous lequel tourne le shell.
type Account struct {
	Name  string `json:"name"`
	Home  string `json:"home"`
	Shell string `json:"shell"`
	Root  bool   `json:"root"`
	uid   uint32
	gid   uint32
	gids  []uint32
}

// DefaultAccount choisit l'utilisateur principal du Pi (UID 1000, qui a sudo sur
// Raspberry Pi OS) ; à défaut, root.
func DefaultAccount() Account {
	if u, err := user.LookupId("1000"); err == nil {
		return accountFor(u)
	}
	u, _ := user.Current()
	if u == nil {
		return Account{Name: "root", Home: "/root", Shell: "/bin/bash", Root: true}
	}
	return accountFor(u)
}

func accountFor(u *user.User) Account {
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)
	a := Account{Name: u.Username, Home: u.HomeDir, Root: uid == 0, uid: uint32(uid), gid: uint32(gid), Shell: loginShell(u.Username)}
	if ids, err := u.GroupIds(); err == nil {
		for _, g := range ids {
			if n, err := strconv.Atoi(g); err == nil {
				a.gids = append(a.gids, uint32(n))
			}
		}
	}
	return a
}

func loginShell(name string) string {
	b, _ := os.ReadFile("/etc/passwd")
	for _, l := range splitLines(string(b)) {
		f := splitColon(l)
		if len(f) == 7 && f[0] == name && f[6] != "" && f[6] != "/usr/sbin/nologin" && f[6] != "/bin/false" {
			return f[6]
		}
	}
	for _, s := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(s); err == nil {
			return s
		}
	}
	return "/bin/sh"
}

func splitLines(s string) (out []string) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func splitColon(s string) (out []string) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// Manager limite le nombre de sessions simultanées.
type Manager struct {
	mu     sync.Mutex
	active int
	Max    int
}

var ErrTooMany = errors.New("trop de terminaux ouverts en même temps")

// Serve relie une connexion WebSocket à un nouveau shell. Bloque jusqu'à la fin de la session.
//
// Protocole : le client envoie des trames texte dont le premier caractère indique le type :
// « 0 » + données saisies, « 1 » + JSON {"cols":…,"rows":…} pour redimensionner.
// Le serveur envoie la sortie du terminal en trames binaires.
func (m *Manager) Serve(ws *Conn, acc Account) error {
	m.mu.Lock()
	if m.Max > 0 && m.active >= m.Max {
		m.mu.Unlock()
		_ = ws.WriteText("\r\n" + ErrTooMany.Error() + "\r\n")
		return ErrTooMany
	}
	m.active++
	m.mu.Unlock()
	defer func() { m.mu.Lock(); m.active--; m.mu.Unlock() }()

	master, slave, err := openPTY()
	if err != nil {
		return err
	}
	defer master.Close()
	_ = setSize(master, 30, 100)

	cmd := exec.Command(acc.Shell, "-l")
	cmd.Dir = acc.Home
	cmd.Env = []string{
		"TERM=xterm-256color", "COLORTERM=truecolor", "LANG=C.UTF-8",
		"HOME=" + acc.Home, "USER=" + acc.Name, "LOGNAME=" + acc.Name, "SHELL=" + acc.Shell,
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = sysProcAttr(acc.uid, acc.gid, acc.gids, !acc.Root && os.Geteuid() == 0)
	if err := cmd.Start(); err != nil {
		slave.Close()
		return err
	}
	slave.Close()

	done := make(chan struct{})
	// Sortie du shell → navigateur
	go func() {
		buf := make([]byte, 32<<10)
		for {
			n, err := master.Read(buf)
			if n > 0 {
				if ws.WriteBinary(buf[:n]) != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()
	// Navigateur → shell
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil || len(msg) == 0 {
				break
			}
			switch msg[0] {
			case '0':
				if _, err := master.Write(msg[1:]); err != nil {
					return
				}
			case '1':
				var sz struct{ Cols, Rows uint16 }
				if json.Unmarshal(msg[1:], &sz) == nil && sz.Cols > 0 && sz.Rows > 0 && sz.Cols < 1000 && sz.Rows < 500 {
					_ = setSize(master, sz.Rows, sz.Cols)
				}
			}
		}
		// Navigateur fermé : on raccroche le shell et tout son groupe de processus.
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGHUP)
			time.AfterFunc(3*time.Second, func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
		}
	}()
	_ = cmd.Wait()
	<-done
	_ = ws.WriteText("\r\n\x1b[2m[session terminée]\x1b[0m\r\n")
	return ws.Close()
}
