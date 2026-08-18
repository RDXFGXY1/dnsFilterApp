package blockpage

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// tlsManager is a tiny on-the-fly certificate authority. It owns a local root
// CA and mints a leaf certificate for whatever hostname the browser asks for
// (via SNI), so the block page can be served over HTTPS for any domain once the
// CA is trusted on the device.
type tlsManager struct {
	caCert  *x509.Certificate
	caKey   *rsa.PrivateKey
	caPEM   []byte
	leafKey *rsa.PrivateKey
	dir     string

	mu    sync.RWMutex
	cache map[string]*tls.Certificate
}

// EnableTLS ensures a local CA exists and starts an HTTPS block-page server on
// the given port using dynamically-generated per-SNI certificates.
func (bps *BlockPageServer) EnableTLS(dir string, port int) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	mgr, err := newTLSManager(dir)
	if err != nil {
		return err
	}
	bps.tlsMgr = mgr

	tlsCfg := &tls.Config{
		GetCertificate: mgr.getCertificate,
		MinVersion:     tls.VersionTLS12,
	}
	srv := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   bps.handler,
		TLSConfig: tlsCfg,
	}
	go func() {
		// Certs are supplied by GetCertificate, so cert/key files are empty.
		_ = srv.ListenAndServeTLS("", "")
	}()
	return nil
}

// CACertPath returns the path to the root CA certificate (for the user to
// install), or "" if TLS is not enabled.
func (bps *BlockPageServer) CACertPath() string {
	if bps.tlsMgr == nil {
		return ""
	}
	return filepath.Join(bps.tlsMgr.dir, "ca.crt")
}

func newTLSManager(dir string) (*tlsManager, error) {
	m := &tlsManager{dir: dir, cache: make(map[string]*tls.Certificate)}

	caCrtPath := filepath.Join(dir, "ca.crt")
	caKeyPath := filepath.Join(dir, "ca.key")

	if crtPEM, err := os.ReadFile(caCrtPath); err == nil {
		if keyPEM, err := os.ReadFile(caKeyPath); err == nil {
			if err := m.load(crtPEM, keyPEM); err == nil {
				return m, nil
			}
		}
	}

	// Generate a fresh CA.
	if err := m.generateCA(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(caCrtPath, m.caPEM, 0644); err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(m.caKey)})
	if err := os.WriteFile(caKeyPath, keyPEM, 0600); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *tlsManager) load(crtPEM, keyPEM []byte) error {
	crtBlock, _ := pem.Decode(crtPEM)
	keyBlock, _ := pem.Decode(keyPEM)
	if crtBlock == nil || keyBlock == nil {
		return fmt.Errorf("invalid CA PEM")
	}
	cert, err := x509.ParseCertificate(crtBlock.Bytes)
	if err != nil {
		return err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	m.caCert, m.caKey, m.caPEM, m.leafKey = cert, key, crtPEM, leafKey
	return nil
}

func (m *tlsManager) generateCA() error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "DNS Filter Local CA", Organization: []string{"DNS Filter"}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return err
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	m.caCert = cert
	m.caKey = key
	m.caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	m.leafKey = leafKey
	return nil
}

// getCertificate mints (and caches) a leaf certificate for the requested SNI.
func (m *tlsManager) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName
	if host == "" {
		host = "blocked.local"
	}

	m.mu.RLock()
	if c, ok := m.cache[host]; ok {
		m.mu.RUnlock()
		return c, nil
	}
	m.mu.RUnlock()

	leaf, err := m.mintLeaf(host)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.cache[host] = leaf
	m.mu.Unlock()
	return leaf, nil
}

func (m *tlsManager) mintLeaf(host string) (*tls.Certificate, error) {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{host},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, m.caCert, &m.leafKey.PublicKey, m.caKey)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{
		Certificate: [][]byte{der, m.caCert.Raw},
		PrivateKey:  m.leafKey,
	}, nil
}
