package terminal

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Exemple de la RFC 6455, section 1.3.
func TestAcceptKey(t *testing.T) {
	if got := AcceptKey("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("Sec-WebSocket-Accept incorrect : %s", got)
	}
}

func TestSameOrigin(t *testing.T) {
	r := httptest.NewRequest("GET", "http://azmoos.local/api/terminal", nil)
	r.Host = "azmoos.local"
	r.Header.Set("Origin", "http://azmoos.local")
	if !SameOrigin(r) {
		t.Fatal("même origine refusée")
	}
	r.Header.Set("Origin", "http://pirate.example")
	if SameOrigin(r) {
		t.Fatal("origine étrangère acceptée")
	}
	r.Header.Del("Origin")
	if SameOrigin(r) {
		t.Fatal("absence d'Origin acceptée")
	}
}

// clientFrame construit une trame client masquée.
func clientFrame(op byte, payload []byte) []byte {
	var mask [4]byte
	_, _ = rand.Read(mask[:])
	f := []byte{0x80 | op}
	n := len(payload)
	switch {
	case n < 126:
		f = append(f, 0x80|byte(n))
	default:
		f = append(f, 0x80|126, byte(n>>8), byte(n))
	}
	f = append(f, mask[:]...)
	for i, b := range payload {
		f = append(f, b^mask[i%4])
	}
	return f
}

// Session complète : shell réel dans un PTY, commande, lecture du résultat.
func TestTerminalSession(t *testing.T) {
	m := &Manager{Max: 2}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := Upgrade(w, r)
		if err != nil {
			return
		}
		acc := Account{Name: "test", Home: t.TempDir(), Shell: "/bin/sh", Root: true}
		_ = m.Serve(ws, acc)
	}))
	defer srv.Close()

	c, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	key := base64.StdEncoding.EncodeToString([]byte("azmo-test-key-16"))
	io.WriteString(c, "GET / HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: "+key+"\r\nSec-WebSocket-Version: 13\r\n\r\n")
	br := bufio.NewReader(c)
	status, _ := br.ReadString('\n')
	if !strings.Contains(status, "101") {
		t.Fatalf("poignée de main refusée : %s", status)
	}
	for {
		l, _ := br.ReadString('\n')
		if l == "\r\n" {
			break
		}
	}
	c.Write(clientFrame(opText, []byte(`1{"cols":120,"rows":40}`)))
	c.Write(clientFrame(opText, []byte("0echo AZMO_$((40+2)); stty size\n")))

	var out strings.Builder
	deadline := time.Now().Add(5 * time.Second)
	c.SetReadDeadline(deadline)
	for time.Now().Before(deadline) && !(strings.Contains(out.String(), "AZMO_42") && strings.Contains(out.String(), "40 120")) {
		var hdr [2]byte
		if _, err := io.ReadFull(br, hdr[:]); err != nil {
			break
		}
		n := int(hdr[1] & 0x7f)
		if n == 126 {
			var b [2]byte
			io.ReadFull(br, b[:])
			n = int(binary.BigEndian.Uint16(b[:]))
		}
		p := make([]byte, n)
		io.ReadFull(br, p)
		out.Write(p)
	}
	if !strings.Contains(out.String(), "AZMO_42") {
		t.Fatalf("la commande n'a pas été exécutée, sortie : %q", out.String())
	}
	if !strings.Contains(out.String(), "40 120") {
		t.Fatalf("le redimensionnement n'a pas été appliqué, sortie : %q", out.String())
	}
	c.Write(clientFrame(opText, []byte("0exit\n")))
}
