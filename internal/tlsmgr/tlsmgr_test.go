package tlsmgr

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"testing"
)

func TestLocalCAAndHTTPS(t *testing.T) {
	m := New(t.TempDir())
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "azmo") })
	if err := m.Start("127.0.0.1:0", h); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()
	info := m.Info()
	if info.CAFingerprint == "" || info.Expires == 0 {
		t.Fatalf("infos incomplètes : %+v", info)
	}
	// Le certificat serveur doit être valide pour 127.0.0.1 avec l'autorité locale.
	caPEM, _ := m.CAPEM()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("autorité illisible")
	}
	leaf, _ := x509.ParseCertificate(m.crt.Certificate[0])
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, DNSName: "127.0.0.1"}); err != nil {
		t.Fatalf("certificat non valide pour 127.0.0.1 : %v", err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, DNSName: "localhost"}); err != nil {
		t.Fatalf("certificat non valide pour localhost : %v", err)
	}

	// Contrainte de nom : l'autorité ne doit PAS pouvoir signer pour un vrai site.
	evil := *leaf
	evil.DNSNames = []string{"banque.fr"}
	evil.IPAddresses = nil
	der, err := x509.CreateCertificate(nopRand{}, &evil, m.ca, leaf.PublicKey, m.caK)
	if err == nil {
		c, _ := x509.ParseCertificate(der)
		if _, err := c.Verify(x509.VerifyOptions{Roots: pool, DNSName: "banque.fr"}); err == nil {
			t.Fatal("l'autorité locale a pu certifier banque.fr : contrainte de nom inefficace")
		}
	}

	// Le certificat existant est réutilisé (pas de réémission inutile).
	before := leaf.SerialNumber
	if err := m.Ensure(); err != nil {
		t.Fatal(err)
	}
	after, _ := x509.ParseCertificate(m.crt.Certificate[0])
	if after.SerialNumber.Cmp(before) != 0 {
		t.Fatal("le certificat a été réémis sans raison")
	}
	// Une régénération explicite produit un nouveau certificat.
	if err := m.Regenerate(); err != nil {
		t.Fatal(err)
	}
	again, _ := x509.ParseCertificate(m.crt.Certificate[0])
	if again.SerialNumber.Cmp(before) == 0 {
		t.Fatal("Regenerate n'a pas réémis le certificat")
	}
	_ = tls.VersionTLS12
}

type nopRand struct{}

func (nopRand) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = byte(i)
	}
	return len(b), nil
}
