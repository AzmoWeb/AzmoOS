package totp

import (
	"encoding/base32"
	"testing"
	"time"
)

// Vecteurs de test officiels de la RFC 6238 (SHA-1, secret « 12345678901234567890 »).
func TestRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037", 20000000000: "353130"}
	for ts, want := range cases {
		got, err := Code(secret, ts/Period)
		if err != nil || got != want {
			t.Errorf("t=%d : %s, attendu %s (%v)", ts, got, want, err)
		}
	}
}

func TestVerifyWindowAndReplay(t *testing.T) {
	s := NewSecret()
	now := time.Unix(1_800_000_000, 0)
	code, _ := Code(s, Step(now)-1) // code de la période précédente : toléré
	step, ok := Verify(s, code, now, 0)
	if !ok {
		t.Fatal("le code de la période précédente devrait être accepté")
	}
	if _, ok := Verify(s, code, now, step); ok {
		t.Fatal("un code déjà utilisé ne doit pas être accepté deux fois")
	}
	old, _ := Code(s, Step(now)-3)
	if _, ok := Verify(s, old, now, 0); ok {
		t.Fatal("un code trop ancien ne doit pas être accepté")
	}
	if _, ok := Verify(s, "12345", now, 0); ok {
		t.Fatal("code de mauvaise longueur accepté")
	}
}

func TestRecoveryCodes(t *testing.T) {
	c := RecoveryCodes(8)
	seen := map[string]bool{}
	for _, x := range c {
		if len(x) != 9 || x[4] != '-' || seen[x] {
			t.Fatalf("code de secours invalide ou dupliqué : %q", x)
		}
		seen[x] = true
	}
}
