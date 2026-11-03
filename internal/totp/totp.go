// Package totp implémente les codes à usage unique basés sur le temps (RFC 6238),
// compatibles Google Authenticator, Authy, 1Password, Bitwarden…
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	Period = 30 // secondes
	Digits = 6
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret génère un secret aléatoire de 160 bits encodé en base32.
func NewSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return b32.EncodeToString(b)
}

// Code calcule le code pour un pas de temps donné.
func Code(secret string, step int64) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return "", err
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000), nil
}

// Step renvoie le pas de temps courant.
func Step(t time.Time) int64 { return t.Unix() / Period }

// Verify vérifie un code en tolérant ±1 pas (décalage d'horloge). Renvoie le pas
// utilisé, pour refuser la réutilisation d'un même code (lastStep).
func Verify(secret, code string, now time.Time, lastStep int64) (int64, bool) {
	code = strings.TrimSpace(strings.ReplaceAll(code, " ", ""))
	if len(code) != Digits {
		return 0, false
	}
	cur := Step(now)
	for _, s := range []int64{cur, cur - 1, cur + 1} {
		if s <= lastStep {
			continue
		}
		want, err := Code(secret, s)
		if err == nil && subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}

// URI construit le lien otpauth:// à encoder en QR code.
func URI(secret, account, issuer string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{"secret": {secret}, "issuer": {issuer}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// RecoveryCodes génère n codes de secours lisibles (ex : 7KQ2-M9XD).
func RecoveryCodes(n int) []string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]string, n)
	for i := range out {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		for j := range b {
			b[j] = alphabet[int(b[j])%len(alphabet)]
		}
		out[i] = string(b[:4]) + "-" + string(b[4:])
	}
	return out
}
