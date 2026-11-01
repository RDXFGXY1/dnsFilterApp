package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLivePrivacyScrubsExistingAndNewEntries(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "privacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().Unix()
	if err := db.SaveLiveQueries([]LiveQuery{{Time: now, Domain: "allowed.example", ClientIP: "192.168.1.2"}, {Time: now, Domain: "blocked.example", ClientIP: "192.168.1.2", Blocked: true}}); err != nil {
		t.Fatal(err)
	}
	if err := db.SetLivePrivacy(LivePrivacy{RetentionHours: 24, HideClientIP: true, BlockedOnly: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveLiveQueries([]LiveQuery{{Time: now, Domain: "another.allowed", ClientIP: "192.168.1.3"}, {Time: now, Domain: "another.blocked", ClientIP: "192.168.1.3", Blocked: true}}); err != nil {
		t.Fatal(err)
	}
	items, err := db.GetLiveQueries(200)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("saved %d items, want 2 blocked items", len(items))
	}
	for _, item := range items {
		if !item.Blocked || item.ClientIP != "" {
			t.Fatalf("privacy not applied: %+v", item)
		}
	}
}

func TestNetworkTestAndDNSHealthHistoryPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	db, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.SaveNetworkTest(NetworkTestRecord{ID: "test-1", Status: "complete", Mode: "custom_data", LimitMB: 100, DownloadMbps: 80.2, StartedAt: now, FinishedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := db.LogDNSHealth("upstream", "error", "Upstream unavailable", "All resolvers failed"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tests, err := db.GetNetworkTestHistory(10)
	if err != nil || len(tests) != 1 || tests[0].LimitMB != 100 || tests[0].DownloadMbps != 80.2 {
		t.Fatalf("network history: %+v, %v", tests, err)
	}
	health, err := db.GetDNSHealthEvents(10)
	if err != nil || len(health) != 1 || health[0].Status != "error" {
		t.Fatalf("DNS health history: %+v, %v", health, err)
	}
}
