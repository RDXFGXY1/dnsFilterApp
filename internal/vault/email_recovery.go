package vault

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type EmailRecoverySettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLSMode  string `json:"tls_mode"`
	From     string `json:"from"`
	To       string `json:"to"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type emailRecoveryFile struct {
	Version           int     `json:"version"`
	Host              string  `json:"host"`
	Port              int     `json:"port"`
	TLSMode           string  `json:"tls_mode,omitempty"`
	From              string  `json:"from"`
	To                string  `json:"to"`
	Username          string  `json:"username"`
	ProtectedPassword string  `json:"protected_password"`
	ProtectedSecret   string  `json:"protected_secret"`
	Master            wrapper `json:"master"`
}

var emailSender = sendRecoveryMail

func (m *Manager) emailRecoveryPath() string {
	return filepath.Join(m.directory, "email-recovery.json")
}

func (m *Manager) EmailRecoveryStatus() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, err := m.readEmailRecovery()
	if err != nil {
		return false, ""
	}
	return true, f.To
}

func (m *Manager) readEmailRecovery() (emailRecoveryFile, error) {
	var f emailRecoveryFile
	data, err := os.ReadFile(m.emailRecoveryPath())
	if err != nil {
		return f, err
	}
	if err = json.Unmarshal(data, &f); err != nil || f.Version != 1 {
		return f, errors.New("email recovery settings are damaged")
	}
	return f, nil
}

func (m *Manager) EnrollEmailRecovery(settings EmailRecoverySettings) error {
	settings.Host = strings.TrimSpace(settings.Host)
	settings.From = strings.TrimSpace(settings.From)
	settings.To = strings.TrimSpace(settings.To)
	settings.Username = strings.TrimSpace(settings.Username)
	if settings.TLSMode == "" {
		settings.TLSMode = smtpTLSMode(settings.Port, "")
	}
	if settings.Host == "" || strings.ContainsAny(settings.Host, "/\\:@ \t\r\n") || settings.Port < 1 || settings.Port > 65535 || (settings.TLSMode != "implicit" && settings.TLSMode != "starttls") || settings.Username == "" || settings.Password == "" {
		return errors.New("enter SMTP host, port, TLS mode, username, and app password")
	}
	from, err := mail.ParseAddress(settings.From)
	if err != nil || from.Address != settings.From {
		return errors.New("sender email is invalid")
	}
	to, err := mail.ParseAddress(settings.To)
	if err != nil || to.Address != settings.To {
		return errors.New("recovery email is invalid")
	}
	m.mu.Lock()
	m.expireLocked()
	if len(m.master) != 32 {
		m.mu.Unlock()
		return ErrLocked
	}
	master := append([]byte(nil), m.master...)
	m.mu.Unlock()
	defer zero(master)
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	defer zero(secret)
	wrapped, err := wrap(master, base64.RawStdEncoding.EncodeToString(secret))
	if err != nil {
		return err
	}
	protectedSecret, err := protectLocal(secret)
	if err != nil {
		return err
	}
	protectedPassword, err := protectLocal([]byte(settings.Password))
	if err != nil {
		return err
	}
	if err := emailSender(settings, "DNS Filter email recovery enabled", "Email recovery is now enabled for this DNS Filter installation. If you request a Vault password reset, a one-time code will arrive here. Keep your offline recovery code separately."); err != nil {
		return err
	}
	f := emailRecoveryFile{Version: 1, Host: settings.Host, Port: settings.Port, TLSMode: settings.TLSMode, From: settings.From, To: settings.To, Username: settings.Username, ProtectedPassword: base64.StdEncoding.EncodeToString(protectedPassword), ProtectedSecret: base64.StdEncoding.EncodeToString(protectedSecret), Master: wrapped}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := writeJSONAtomic(m.emailRecoveryPath(), f, 0600); err != nil {
		return err
	}
	return m.auditLocked("vault.email.enroll", "email recovery enrolled")
}

func (m *Manager) RequestEmailRecovery() error {
	m.mu.Lock()
	if time.Since(m.emailCodeSentAt) < time.Minute {
		m.mu.Unlock()
		return errors.New("wait one minute before requesting another code")
	}
	f, err := m.readEmailRecovery()
	if err != nil {
		m.mu.Unlock()
		return errors.New("email recovery is not configured")
	}
	m.mu.Unlock()
	protected, err := base64.StdEncoding.DecodeString(f.ProtectedPassword)
	if err != nil {
		return err
	}
	password, err := unprotectLocal(protected)
	if err != nil {
		return errors.New("email recovery belongs to another Windows service identity")
	}
	defer zero(password)
	random := make([]byte, 4)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	code := fmt.Sprintf("%08d", binary.BigEndian.Uint32(random)%100000000)
	settings := EmailRecoverySettings{Host: f.Host, Port: f.Port, TLSMode: smtpTLSMode(f.Port, f.TLSMode), From: f.From, To: f.To, Username: f.Username, Password: string(password)}
	if err := emailSender(settings, "DNS Filter Vault recovery code", "Your one-time Vault recovery code is: "+code+"\r\nIt expires in 10 minutes. If you did not request this, ignore this email."); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emailCodeHash = sha256.Sum256([]byte(code))
	m.emailCodeExpires = time.Now().Add(10 * time.Minute)
	m.emailCodeSentAt = time.Now()
	m.emailCodeFailures = 0
	return nil
}

func (m *Manager) RecoverWithEmailCode(code, newPassword string) error {
	if len(newPassword) < 12 {
		return errors.New("new vault password must contain at least 12 characters")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.emailCodeExpires.IsZero() || time.Now().After(m.emailCodeExpires) || m.emailCodeFailures >= 5 {
		return errors.New("recovery code expired; request a new one")
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(code)))
	if subtle.ConstantTimeCompare(hash[:], m.emailCodeHash[:]) != 1 {
		m.emailCodeFailures++
		return errors.New("invalid recovery code")
	}
	f, err := m.readEmailRecovery()
	if err != nil {
		return err
	}
	protected, err := base64.StdEncoding.DecodeString(f.ProtectedSecret)
	if err != nil {
		return err
	}
	secret, err := unprotectLocal(protected)
	if err != nil {
		return errors.New("email recovery belongs to another Windows service identity")
	}
	defer zero(secret)
	master, err := unwrap(f.Master, base64.RawStdEncoding.EncodeToString(secret))
	if err != nil {
		return err
	}
	defer zero(master)
	e, err := m.readEnvelope()
	if err != nil {
		return err
	}
	e.Password, err = wrap(master, newPassword)
	if err != nil {
		return err
	}
	if err := writeJSONAtomic(m.envelopePath, e, 0600); err != nil {
		return err
	}
	m.emailCodeExpires = time.Time{}
	m.emailCodeHash = [32]byte{}
	m.emailCodeFailures = 0
	m.setMaster(master)
	return m.auditLocked("vault.email.recover", "vault password reset with email code")
}

func sendRecoveryMail(settings EmailRecoverySettings, subject, body string) error {
	ip, err := resolveSMTPOutsideFilter(settings.Host)
	if err != nil {
		return fmt.Errorf("independent SMTP DNS lookup failed: %w", err)
	}
	address := net.JoinHostPort(ip, fmt.Sprint(settings.Port))
	connection, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return err
	}
	if smtpTLSMode(settings.Port, settings.TLSMode) == "implicit" {
		secure := tls.Client(connection, &tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12})
		if err := secure.Handshake(); err != nil {
			return fmt.Errorf("SMTP TLS failed: %w", err)
		}
		connection = secure
	}
	client, err := smtp.NewClient(connection, settings.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if smtpTLSMode(settings.Port, settings.TLSMode) == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not offer STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: settings.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if err := client.Auth(smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	if err := client.Mail(settings.From); err != nil {
		return err
	}
	if err := client.Rcpt(settings.To); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	message, err := buildRecoveryMessage(settings.From, settings.To, subject, body)
	if err != nil {
		w.Close()
		return err
	}
	if _, err := w.Write(message); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}

func smtpTLSMode(port int, mode string) string {
	if mode != "" {
		return mode
	}
	if port == 465 {
		return "implicit"
	}
	return "starttls"
}

// Resolve over a pinned HTTPS DNS endpoint so recovery still reaches the mail
// provider when this machine's DNS route or the local filter is unavailable.
func resolveSMTPOutsideFilter(host string) (string, error) {
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
		},
		TLSClientConfig: &tls.Config{ServerName: "cloudflare-dns.com", MinVersion: tls.VersionTLS12},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("DoH redirect refused") }}
	request, err := http.NewRequest(http.MethodGet, "https://cloudflare-dns.com/dns-query?name="+url.QueryEscape(host)+"&type=A", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/dns-json")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DoH returned %d", response.StatusCode)
	}
	var result struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&result); err != nil {
		return "", err
	}
	for _, answer := range result.Answer {
		if answer.Type == 1 {
			if ip := net.ParseIP(answer.Data); ip != nil && ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}
	return "", errors.New("SMTP host has no IPv4 address")
}
