// Package tlsmgr fournit le HTTPS local : une autorité de certification propre au Pi
// (limitée aux noms et adresses locaux) et un certificat serveur renouvelé automatiquement.
package tlsmgr

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Réseaux privés autorisés par la contrainte de nom de l'autorité.
var privateNets = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "100.64.0.0/10", "169.254.0.0/16", "fc00::/7", "fe80::/10", "::1/128"}

type Manager struct {
	dir string
	mu  sync.RWMutex
	ca  *x509.Certificate
	caK *ecdsa.PrivateKey
	crt *tls.Certificate
	srv *http.Server
}

func New(confDir string) *Manager { return &Manager{dir: filepath.Join(confDir, "tls")} }

func (m *Manager) path(n string) string { return filepath.Join(m.dir, n) }

func writePEM(path, typ string, der []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), mode)
}

func readPEM(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, errors.New("PEM invalide : " + path)
	}
	return blk.Bytes, nil
}

func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	return n
}

// ensureCA charge ou crée l'autorité locale.
func (m *Manager) ensureCA() error {
	if m.ca != nil {
		return nil
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return err
	}
	if der, err := readPEM(m.path("ca.crt")); err == nil {
		if kd, err := readPEM(m.path("ca.key")); err == nil {
			ca, err1 := x509.ParseCertificate(der)
			k, err2 := x509.ParseECPrivateKey(kd)
			if err1 == nil && err2 == nil && time.Until(ca.NotAfter) > 30*24*time.Hour {
				m.ca, m.caK = ca, k
				return nil
			}
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	var nets []*net.IPNet
	for _, c := range privateNets {
		_, n, _ := net.ParseCIDR(c)
		nets = append(nets, n)
	}
	dns := []string{"local", "localhost", "home.arpa", "lan", "internal", "ts.net"}
	if host != "" && !strings.Contains(host, ".") {
		dns = append(dns, host)
	}
	tpl := &x509.Certificate{
		SerialNumber:                serial(),
		Subject:                     pkix.Name{CommonName: "Azmo OS — autorité locale (" + host + ")", Organization: []string{"Azmo OS"}},
		NotBefore:                   time.Now().Add(-time.Hour),
		NotAfter:                    time.Now().AddDate(10, 0, 0),
		IsCA:                        true,
		BasicConstraintsValid:       true,
		MaxPathLenZero:              true,
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		PermittedDNSDomainsCritical: true,
		PermittedDNSDomains:         dns,
		PermittedIPRanges:           nets,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	kd, _ := x509.MarshalECPrivateKey(key)
	if err := writePEM(m.path("ca.key"), "EC PRIVATE KEY", kd, 0o600); err != nil {
		return err
	}
	if err := writePEM(m.path("ca.crt"), "CERTIFICATE", der, 0o644); err != nil {
		return err
	}
	m.ca, _ = x509.ParseCertificate(der)
	m.caK = key
	_ = os.Remove(m.path("server.crt")) // le certificat serveur doit être réémis
	return nil
}

// localNames renvoie les noms et adresses par lesquels on joint le Pi.
func localNames() ([]string, []net.IP) {
	host, _ := os.Hostname()
	names := []string{"localhost"}
	if host != "" {
		names = append(names, host, host+".local")
	}
	ips := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || strings.HasPrefix(i.Name, "veth") || strings.HasPrefix(i.Name, "docker") || strings.HasPrefix(i.Name, "br-") {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && !n.IP.IsLoopback() {
				ips = append(ips, n.IP)
			}
		}
	}
	return names, ips
}

func sameSANs(c *x509.Certificate, names []string, ips []net.IP) bool {
	a := append([]string(nil), c.DNSNames...)
	b := append([]string(nil), names...)
	sort.Strings(a)
	sort.Strings(b)
	if !slices.Equal(a, b) {
		return false
	}
	have := map[string]bool{}
	for _, ip := range c.IPAddresses {
		have[ip.String()] = true
	}
	for _, ip := range ips {
		if !have[ip.String()] {
			return false
		}
	}
	return len(c.IPAddresses) == len(ips)
}

// Ensure crée/renouvelle le certificat serveur si nécessaire (nouvelle IP, nouveau nom, expiration).
func (m *Manager) Ensure() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureCA(); err != nil {
		return err
	}
	names, ips := localNames()
	// Seules les adresses privées sont permises par l'autorité.
	ips = slices.DeleteFunc(ips, func(ip net.IP) bool {
		return !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !inCGNAT(ip)
	})
	if c, err := tls.LoadX509KeyPair(m.path("server.crt"), m.path("server.key")); err == nil {
		leaf, _ := x509.ParseCertificate(c.Certificate[0])
		if leaf != nil && sameSANs(leaf, names, ips) && time.Until(leaf.NotAfter) > 30*24*time.Hour && leaf.CheckSignatureFrom(m.ca) == nil {
			m.crt = &c
			return nil
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	tpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: names[len(names)-1], Organization: []string{"Azmo OS"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(0, 0, 397), // limite acceptée par Apple et Chrome
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     names,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, m.ca, &key.PublicKey, m.caK)
	if err != nil {
		return err
	}
	kd, _ := x509.MarshalECPrivateKey(key)
	if err := writePEM(m.path("server.key"), "EC PRIVATE KEY", kd, 0o600); err != nil {
		return err
	}
	chain := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: m.ca.Raw})...)
	if err := os.WriteFile(m.path("server.crt"), chain, 0o644); err != nil {
		return err
	}
	c, err := tls.LoadX509KeyPair(m.path("server.crt"), m.path("server.key"))
	if err != nil {
		return err
	}
	m.crt = &c
	return nil
}

func inCGNAT(ip net.IP) bool {
	_, n, _ := net.ParseCIDR("100.64.0.0/10")
	return n.Contains(ip)
}

// Regenerate force l'émission d'un nouveau certificat serveur.
func (m *Manager) Regenerate() error {
	_ = os.Remove(m.path("server.crt"))
	return m.Ensure()
}

// CAPEM renvoie le certificat de l'autorité (à installer sur les appareils).
func (m *Manager) CAPEM() ([]byte, error) {
	m.mu.Lock()
	err := m.ensureCA()
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(m.path("ca.crt"))
}

// Info décrit l'état des certificats.
type Info struct {
	Running       bool     `json:"running"`
	CAFingerprint string   `json:"caFingerprint"`
	CAExpires     int64    `json:"caExpires"`
	Names         []string `json:"names"`
	Expires       int64    `json:"expires"`
}

func (m *Manager) Info() Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inf := Info{Running: m.srv != nil, Names: []string{}}
	if m.ca != nil {
		sum := sha256.Sum256(m.ca.Raw)
		h := strings.ToUpper(hex.EncodeToString(sum[:]))
		var parts []string
		for i := 0; i < len(h); i += 2 {
			parts = append(parts, h[i:i+2])
		}
		inf.CAFingerprint = strings.Join(parts, ":")
		inf.CAExpires = m.ca.NotAfter.Unix()
	}
	if m.crt != nil {
		if leaf, err := x509.ParseCertificate(m.crt.Certificate[0]); err == nil {
			inf.Names = append(inf.Names, leaf.DNSNames...)
			for _, ip := range leaf.IPAddresses {
				inf.Names = append(inf.Names, ip.String())
			}
			inf.Expires = leaf.NotAfter.Unix()
		}
	}
	return inf
}

func (m *Manager) getCert(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.crt == nil {
		return nil, errors.New("certificat indisponible")
	}
	return m.crt, nil
}

// Start lance le serveur HTTPS (idempotent). Le certificat est revérifié toutes les 6 h
// pour suivre les changements d'adresse IP.
func (m *Manager) Start(addr string, h http.Handler) error {
	if err := m.Ensure(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.srv != nil {
		m.mu.Unlock()
		return nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("port HTTPS indisponible (%s) : %w", addr, err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, GetCertificate: m.getCert}}
	m.srv = srv
	m.mu.Unlock()
	go func() {
		if err := srv.ServeTLS(ln, "", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTPS : %v", err)
		}
	}()
	go func() {
		for {
			time.Sleep(6 * time.Hour)
			m.mu.RLock()
			running := m.srv == srv
			m.mu.RUnlock()
			if !running {
				return
			}
			if err := m.Ensure(); err != nil {
				log.Printf("HTTPS : renouvellement : %v", err)
			}
		}
	}()
	return nil
}

// Stop arrête le serveur HTTPS.
func (m *Manager) Stop() {
	m.mu.Lock()
	srv := m.srv
	m.srv = nil
	m.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

// Running indique si le HTTPS est actif.
func (m *Manager) Running() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.srv != nil
}
