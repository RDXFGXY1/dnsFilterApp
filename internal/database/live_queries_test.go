package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLiveQueryHistoryKeepsLatest200(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	db, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	batch := make([]LiveQuery, 205)
	base := time.Now().Unix() - 205
	for i := range batch {
		batch[i] = LiveQuery{Time: base + int64(i+1), Domain: "example.org", ClientIP: "127.0.0.1", Type: "A", Blocked: i%2 == 0}
	}
	if err := db.SaveLiveQueries(batch); err != nil {
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
	items, err := db.GetLiveQueries(200)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 200 || items[0].Time != base+205 || items[199].Time != base+6 {
		t.Fatalf("unexpected persisted history: count=%d newest=%d oldest=%d", len(items), items[0].Time, items[len(items)-1].Time)
	}
	if err := db.ClearLiveQueries(); err != nil {
		t.Fatal(err)
	}
	items, err = db.GetLiveQueries(200)
	if err != nil || len(items) != 0 {
		t.Fatalf("clear failed: count=%d err=%v", len(items), err)
	}
}
