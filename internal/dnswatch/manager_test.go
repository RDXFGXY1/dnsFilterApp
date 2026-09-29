package dnswatch

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestEventSignatureChangesOnlyForRouteState(t *testing.T) {
	a := Event{Interface: "Wi-Fi", DNS: "127.0.0.1, ::1", Protected: true}
	b := a
	if signature(a) != signature(b) {
		t.Fatal("identical route states must have identical signatures")
	}
	b.DNS = "1.1.1.1"
	b.Protected = false
	if signature(a) == signature(b) {
		t.Fatal("external DNS change must produce a different signature")
	}
}

func TestAutoRepairRestoresChangedAdapterRoute(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "route.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	protected := false
	repairs := 0
	status := func() (string, string, bool, error) {
		if protected {
			return "Wi-Fi 2", "127.0.0.1, ::1", true, nil
		}
		return "Wi-Fi 2", "192.168.1.1", false, nil
	}
	setDNS := func() error { repairs++; protected = true; return nil }
	m, err := newManager(db, true, status, setDNS, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if repairs != 1 || !m.Current().Protected || m.Current().Interface != "Wi-Fi 2" {
		t.Fatalf("route was not repaired: repairs=%d current=%+v", repairs, m.Current())
	}
}

func TestDisabledAutoRepairOnlyObserves(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "route.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repairs := 0
	m, err := newManager(db, false, func() (string, string, bool, error) {
		return "Ethernet", "1.1.1.1", false, nil
	}, func() error { repairs++; return nil }, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if repairs != 0 || m.Current().Protected {
		t.Fatalf("disabled route watch changed DNS: repairs=%d current=%+v", repairs, m.Current())
	}
}

func TestAutoRepairFailureIsRecordedWithoutClaimingProtection(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "route.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	m, err := newManager(db, true, func() (string, string, bool, error) {
		return "Wi-Fi", "192.168.1.1", false, nil
	}, func() error { return errors.New("administrator permission required") }, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	current := m.Current()
	if current.Protected || current.Error != "administrator permission required" {
		t.Fatalf("failed repair must remain visible and unprotected: %+v", current)
	}
}
