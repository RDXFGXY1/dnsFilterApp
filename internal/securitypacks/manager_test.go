package securitypacks

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSignedPackLifecycle(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := New(t.TempDir(), map[string]string{"test-key": base64.StdEncoding.EncodeToString(publicKey)})
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		FormatVersion: FormatVersion, ID: "malware-core", Name: "Malware Core", Version: 1,
		Category: "malware", KeyID: "test-key", CreatedAt: time.Now().Add(-time.Hour).UTC(),
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(), Domains: []string{"bad.example", "evil.test"},
	}
	data := signTestPack(t, manifest, privateKey)

	info, err := manager.Install(data)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if info.DomainCount != 2 || len(manager.List()) != 1 {
		t.Fatalf("installed info = %#v, list = %#v", info, manager.List())
	}
	match, blocked := manager.Match("cdn.bad.example")
	if !blocked || match.PackID != "malware-core" || match.Category != "malware" {
		t.Fatalf("Match() = (%#v, %v)", match, blocked)
	}

	upgraded := manifest
	upgraded.Version = 2
	upgraded.Domains = []string{"new-threat.example"}
	if _, err := manager.Install(signTestPack(t, upgraded, privateKey)); err != nil {
		t.Fatalf("upgrade Install() error = %v", err)
	}
	if _, blocked := manager.Match("bad.example"); blocked {
		t.Fatal("old pack domains remained after upgrade")
	}
	if err := manager.Remove("malware-core"); err != nil || len(manager.List()) != 0 {
		t.Fatalf("Remove() error = %v, list = %#v", err, manager.List())
	}
}

func TestPackRejectsTamperingExpiryAndUnknownKey(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	manager, err := New(t.TempDir(), map[string]string{"test-key": base64.StdEncoding.EncodeToString(publicKey)})
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		FormatVersion: FormatVersion, ID: "threats", Name: "Threats", Version: 1, Category: "malware",
		KeyID: "test-key", CreatedAt: time.Now().Add(-time.Hour).UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC(),
		Domains: []string{"bad.example"},
	}
	valid := signTestPack(t, manifest, privateKey)
	tampered := []byte(strings.Replace(string(valid), "bad.example", "good.example", 1))
	if _, _, err := manager.Verify(tampered); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("tampered pack error = %v", err)
	}

	expired := manifest
	expired.ExpiresAt = time.Now().Add(-time.Minute).UTC()
	if _, _, err := manager.Verify(signTestPack(t, expired, privateKey)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired pack error = %v", err)
	}

	unknown := manifest
	unknown.KeyID = "stranger"
	if _, _, err := manager.Verify(signTestPack(t, unknown, privateKey)); err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("unknown-key pack error = %v", err)
	}
}

func TestPackStopsMatchingAfterRuntimeExpiry(t *testing.T) {
	manager := &Manager{
		packs: map[string]Info{},
		domains: map[string]Match{
			"expired.example": {PackID: "expired", ExpiresAt: time.Now().Add(-time.Second)},
		},
	}
	if _, blocked := manager.Match("expired.example"); blocked {
		t.Fatal("expired pack continued blocking without a restart")
	}
	if count := manager.DomainCount(); count != 0 {
		t.Fatalf("DomainCount() = %d, want 0 for expired pack", count)
	}
}

func signTestPack(t *testing.T, manifest Manifest, privateKey ed25519.PrivateKey) []byte {
	t.Helper()
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	pack := Pack{Manifest: manifest, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))}
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
