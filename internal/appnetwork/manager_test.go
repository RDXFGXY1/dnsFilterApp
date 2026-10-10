package appnetwork

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

type fakePlatform struct {
	blocked string
	allowed string
	running []App
	blocks  []string
}

func (f *fakePlatform) Supported() bool             { return true }
func (f *fakePlatform) RunningApps() ([]App, error) { return f.running, nil }
func (f *fakePlatform) Block(path, rule string) error {
	f.blocked = path + "|" + rule
	f.blocks = append(f.blocks, path)
	return nil
}
func (f *fakePlatform) Allow(rule string) error { f.allowed = rule; return nil }

func TestBlockAndAllowApplication(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	fake := &fakePlatform{}
	m, err := newWithPlatform(db, fake)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "test-app-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_ = file.Close()
	if err := m.SetBlocked(path, true); err != nil {
		t.Fatal(err)
	}
	if fake.blocked == "" {
		t.Fatal("firewall block was not requested")
	}
	apps, err := m.Apps()
	if err != nil || len(apps) != 1 || !apps[0].Blocked {
		t.Fatalf("apps: %v %+v", err, apps)
	}
	if err := m.SetBlocked(path, false); err != nil {
		t.Fatal(err)
	}
	if fake.allowed == "" {
		t.Fatal("firewall allow was not requested")
	}
	events, err := m.Events(10)
	if err != nil || len(events) != 2 {
		t.Fatalf("events: %v %+v", err, events)
	}
}

func TestAllowStillWorksAfterExecutableWasRemoved(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	fake := &fakePlatform{}
	m, err := newWithPlatform(db, fake)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "removed-app-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_ = file.Close()
	if err = m.SetBlocked(path, true); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = m.SetBlocked(path, false); err != nil {
		t.Fatalf("allowing a removed executable should use its stored rule: %v", err)
	}
	if fake.allowed == "" {
		t.Fatal("stored firewall rule was not removed")
	}
}

func TestResetAllReleasesOwnedRules(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	fake := &fakePlatform{}
	m, err := newWithPlatform(db, fake)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "reset-app-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_ = file.Close()
	if err = m.SetBlocked(path, true); err != nil {
		t.Fatal(err)
	}
	if err = m.ResetAll(); err != nil {
		t.Fatal(err)
	}
	apps, err := m.Apps()
	if err != nil || len(apps) != 1 || apps[0].Blocked {
		t.Fatalf("reset state: err=%v apps=%+v", err, apps)
	}
	if fake.allowed == "" {
		t.Fatal("platform rule was not released")
	}
}

func TestRejectsNonExecutablePath(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	m, _ := newWithPlatform(db, &fakePlatform{})
	if err := m.SetBlocked("notes.txt", true); err == nil {
		t.Fatal("expected invalid executable error")
	}
}

func TestQuarantineAndRelease(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	fake := &fakePlatform{}
	m, err := newWithPlatform(db, fake)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "danger-*.exe")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	_ = file.Close()
	if err = m.Quarantine(path, "unexpected destination", 82); err != nil {
		t.Fatal(err)
	}
	items, err := m.Quarantined()
	if err != nil || len(items) != 1 || items[0].Risk != 82 {
		t.Fatalf("quarantine: %v %+v", err, items)
	}
	if err = m.ReleaseQuarantine(path); err != nil {
		t.Fatal(err)
	}
	items, _ = m.Quarantined()
	if len(items) != 0 || fake.allowed == "" {
		t.Fatalf("release failed: %+v", items)
	}
}

func TestFocusAllowlistBlocksOnlyNonAllowedAppsAndReleasesThem(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	t.Cleanup(func() { _ = db.Close() })
	allowedFile, _ := os.CreateTemp(t.TempDir(), "allowed-*.exe")
	blockedFile, _ := os.CreateTemp(t.TempDir(), "blocked-*.exe")
	_ = allowedFile.Close()
	_ = blockedFile.Close()
	fake := &fakePlatform{running: []App{{Name: "Allowed", Path: allowedFile.Name()}, {Name: "Blocked", Path: blockedFile.Name()}}}
	m, err := newWithPlatform(db, fake)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.BeginFocusAllowlist([]string{allowedFile.Name()}); err != nil {
		t.Fatal(err)
	}
	if len(fake.blocks) != 1 || fake.blocks[0] != blockedFile.Name() {
		t.Fatalf("unexpected focus blocks: %#v", fake.blocks)
	}
	if err = m.EndFocusAllowlist(); err != nil {
		t.Fatal(err)
	}
	if fake.allowed == "" {
		t.Fatal("focus rule was not released")
	}
}
