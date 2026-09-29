package connectionfreeze

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"testing"
)

type fakeBackend struct {
	closed bool
	events []Event
}

func (b *fakeBackend) Close() error             { b.closed = true; return nil }
func (b *fakeBackend) Events() ([]Event, error) { e := b.events; b.events = nil; return e, nil }
func testManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	m, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close(); db.Close() })
	return m
}
func TestLifecycleAndStaleTimer(t *testing.T) {
	m := testManager(t)
	var instances []*fakeBackend
	m.start = func() (backend, error) { b := &fakeBackend{}; instances = append(instances, b); return b, nil }
	if err := m.Start(-1); err == nil {
		t.Fatal("invalid duration accepted")
	}
	if len(instances) != 0 {
		t.Fatal("invalid duration reached firewall")
	}
	if err := m.Start(1); err != nil {
		t.Fatal(err)
	}
	old := m.generation
	if err := m.Start(1); err == nil {
		t.Fatal("duplicate start accepted")
	}
	if err := m.Stop("test stop"); err != nil {
		t.Fatal(err)
	}
	if !instances[0].closed {
		t.Fatal("firewall rules not closed")
	}
	if err := m.Start(1); err != nil {
		t.Fatal(err)
	}
	m.expire(old)
	if !m.Status().Active {
		t.Fatal("stale timer stopped new session")
	}
	m.expire(m.generation)
	if m.Status().Active {
		t.Fatal("expiry did not stop session")
	}
	if !instances[1].closed {
		t.Fatal("expiry left firewall rules")
	}
}
func TestStartFailureAndPersistedEvents(t *testing.T) {
	m := testManager(t)
	m.start = func() (backend, error) { return nil, fmt.Errorf("permission denied") }
	if err := m.Start(15); err == nil || m.Status().Active {
		t.Fatal("failed start marked active")
	}
	b := &fakeBackend{events: []Event{{Action: "blocked", App: "game.exe", Remote: "192.0.2.1:443", Protocol: "TCP"}}}
	m.start = func() (backend, error) { return b, nil }
	if err := m.Start(15); err != nil {
		t.Fatal(err)
	}
	m.Stop("user stopped")
	events, err := m.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 || events[1].App != "game.exe" || events[0].Action != "disabled" {
		t.Fatalf("missing session or connection events: %+v", events)
	}
	m.Close()
	if err := m.Start(15); err == nil {
		t.Fatal("closed manager restarted")
	}
}

func TestManualFreezeHasNoTimer(t *testing.T) {
	m := testManager(t)
	b := &fakeBackend{}
	m.start = func() (backend, error) { return b, nil }
	if err := m.Start(0); err != nil {
		t.Fatal(err)
	}
	status := m.Status()
	if !status.Active || !status.Manual || !status.Until.IsZero() || m.timer != nil {
		t.Fatalf("manual mode unexpectedly timed: %+v", status)
	}
	m.expire(m.generation)
	if !m.Status().Active {
		t.Fatal("manual mode expired")
	}
	if err := m.Stop("Manual stop"); err != nil {
		t.Fatal(err)
	}
	if m.Status().Active || m.Status().Manual || !b.closed {
		t.Fatal("manual stop did not release rules")
	}
	if err := m.Start(1); err != nil {
		t.Fatal(err)
	}
	if m.Status().Manual || m.timer == nil {
		t.Fatal("timed mode inherited manual configuration")
	}
}
