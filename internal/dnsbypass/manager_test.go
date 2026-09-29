package dnsbypass

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"testing"
)

type fakePlatform struct{ active bool }

func (f *fakePlatform) Enable() error         { f.active = true; return nil }
func (f *fakePlatform) Disable() error        { f.active = false; return nil }
func (f *fakePlatform) Active() (bool, error) { return f.active, nil }
func (f *fakePlatform) Supported() bool       { return true }
func TestEnableDisable(t *testing.T) {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	p := &fakePlatform{}
	m, err := newWithPlatform(db, p)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	s, err := m.Status()
	if err != nil || !s.Enabled || !s.Enforced {
		t.Fatalf("status: %+v %v", s, err)
	}
	if err = m.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	s, _ = m.Status()
	if s.Enabled || s.Enforced {
		t.Fatalf("status: %+v", s)
	}
}
