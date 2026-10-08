package filter

import (
	"path/filepath"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
)

func persistentRulesTestEngine(t *testing.T, db *database.DB) *Engine {
	t.Helper()
	return &Engine{
		cfg:           &config.Config{},
		db:            db,
		log:           logger.Get(),
		whitelist:     make(map[string]bool),
		customBlocked: make(map[string]bool),
	}
}

func TestDashboardRulesSurviveDatabaseReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "rules.db")
	db, err := database.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	first := persistentRulesTestEngine(t, db)
	if err := first.AddToWhitelist("*.Spotify.COM."); err != nil {
		t.Fatal(err)
	}
	if err := first.AddToCustomBlocklist("blocked.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := database.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	restarted := persistentRulesTestEngine(t, reopened)
	if err := restarted.loadPersistentRules(); err != nil {
		t.Fatal(err)
	}
	if !restarted.isWhitelisted("dealer.spotify.com") {
		t.Fatal("wildcard whitelist rule was not restored after database reopen")
	}
	if !restarted.customBlocked["blocked.example.com"] {
		t.Fatal("custom block rule was not restored after database reopen")
	}
}

func TestRemovedDashboardRulesStayRemoved(t *testing.T) {
	db, err := database.New(filepath.Join(t.TempDir(), "rules.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first := persistentRulesTestEngine(t, db)
	if err := first.AddToWhitelist("*.spotify.com"); err != nil {
		t.Fatal(err)
	}
	if err := first.RemoveFromWhitelist("*.spotify.com"); err != nil {
		t.Fatal(err)
	}
	if err := first.AddToCustomBlocklist("blocked.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := first.RemoveFromCustomBlocklist("blocked.example.com"); err != nil {
		t.Fatal(err)
	}

	restarted := persistentRulesTestEngine(t, db)
	if err := restarted.loadPersistentRules(); err != nil {
		t.Fatal(err)
	}
	if restarted.isWhitelisted("dealer.spotify.com") {
		t.Fatal("removed wildcard whitelist rule returned after reload")
	}
	if restarted.customBlocked["blocked.example.com"] {
		t.Fatal("removed custom block rule returned after reload")
	}
}
