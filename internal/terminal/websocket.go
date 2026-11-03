package terminal

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Implémentation minimale de WebSocket (RFC 6455), suffisante pour un terminal :
// trames texte/binaires, ping/pong, fermeture. Pas d'extension ni de compression.

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const (
	opCont   = 0x0
	opText   = 0x1
	opBinary = 0x2
	opClose  = 0x8
	opPing   = 0x9
	opPong   = 0xA
)

// Conn est une connexion WebSocket côté serveur.
type Conn struct {
	c   net.Conn
	br  *bufio.Reader
	wmu sync.Mutex
}

// SameOrigin vérifie que la page qui ouvre le WebSocket vient du même site
// (protection contre le « cross-site WebSocket hijacking »).
func SameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return false
	}
	u, err := url.Parse(o)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// Upgrade réalise la poignée de main WebSocket.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
		http.Error(w, "WebSocket attendu", http.StatusBadRequest)
		return nil, errors.New("pas une requête WebSocket")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" || r.Header.Get("Sec-WebSocket-Version") != "13" {
		http.Error(w, "version WebSocket non prise en charge", http.StatusBadRequest)
		return nil, errors.New("poignée de main invalide")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("connexion non détournable")
	}
	c, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	h := sha1.Sum([]byte(key + wsGUID))
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
		base64.StdEncoding.EncodeToString(h[:]) + "\r\n\r\n"
	if _, err := c.Write([]byte(resp)); err != nil {
		c.Close()
		return nil, err
	}
	return &Conn{c: c, br: brw.Reader}, nil
}

// AcceptKey calcule la valeur Sec-WebSocket-Accept (exposée pour les tests).
func AcceptKey(key string) string {
	h := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

const maxFrame = 1 << 20

// ReadMessage lit le prochain message de données (texte ou binaire), en gérant ping/close.
func (ws *Conn) ReadMessage() (op byte, data []byte, err error) {
	var msg []byte
	var msgOp byte
	for {
		var hdr [2]byte
		if _, err = io.ReadFull(ws.br, hdr[:]); err != nil {
			return
		}
		fin := hdr[0]&0x80 != 0
		opcode := hdr[0] & 0x0f
		masked := hdr[1]&0x80 != 0
		n := uint64(hdr[1] & 0x7f)
		switch n {
		case 126:
			var b [2]byte
			if _, err = io.ReadFull(ws.br, b[:]); err != nil {
				return
			}
			n = uint64(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err = io.ReadFull(ws.br, b[:]); err != nil {
				return
			}
			n = binary.BigEndian.Uint64(b[:])
		}
		if !masked {
			return 0, nil, errors.New("trame client non masquée")
		}
		if n > maxFrame || uint64(len(msg))+n > maxFrame {
			return 0, nil, errors.New("message trop grand")
		}
		var mask [4]byte
		if _, err = io.ReadFull(ws.br, mask[:]); err != nil {
			return
		}
		payload := make([]byte, n)
		if _, err = io.ReadFull(ws.br, payload); err != nil {
			return
		}
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
		switch opcode {
		case opPing:
			_ = ws.write(opPong, payload)
			continue
		case opPong:
			continue
		case opClose:
			_ = ws.write(opClose, nil)
			return opClose, nil, io.EOF
		case opText, opBinary:
			msgOp, msg = opcode, payload
		case opCont:
			msg = append(msg, payload...)
		default:
			return 0, nil, errors.New("opcode inconnu")
		}
		if fin {
			return msgOp, msg, nil
		}
	}
}

func (ws *Conn) write(op byte, data []byte) error {
	ws.wmu.Lock()
	defer ws.wmu.Unlock()
	hdr := []byte{0x80 | op}
	switch n := len(data); {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n <= 0xffff:
		hdr = append(hdr, 126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}
	if _, err := ws.c.Write(hdr); err != nil {
		return err
	}
	_, err := ws.c.Write(data)
	return err
}

// WriteBinary envoie des données binaires.
func (ws *Conn) WriteBinary(b []byte) error { return ws.write(opBinary, b) }

// WriteText envoie un message texte.
func (ws *Conn) WriteText(s string) error { return ws.write(opText, []byte(s)) }

// Close ferme la connexion.
func (ws *Conn) Close() error {
	_ = ws.write(opClose, []byte{0x03, 0xe8})
	return ws.c.Close()
}
