package discipline

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newJournalTestManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestJournalEntryLifecycleAndStats(t *testing.T) {
	m := newJournalTestManager(t)
	typeID, err := m.CreateJournalType("Studying", "#6be5bd")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("2006-01-02")
	id, err := m.CreateJournalEntry(JournalEntryInput{TypeID: typeID, Title: "Read networking notes", Content: "DNS cache and routing", Day: today, DurationMinutes: 45, Mood: 8, Focus: 9, Tags: "dns,study"})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := m.ListJournalEntries(JournalFilters{Query: "routing"})
	if err != nil || len(entries) != 1 || entries[0].TypeName != "Studying" {
		t.Fatalf("unexpected entries: %#v %v", entries, err)
	}
	stats, err := m.JournalStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalEntries != 1 || stats.ActiveDays != 1 || stats.DurationMinutes != 45 || stats.CurrentStreak != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if err := m.UpdateJournalEntry(id, JournalEntryInput{Title: "Updated", Day: today, Focus: 10, Pinned: true}); err != nil {
		t.Fatal(err)
	}
	entries, _ = m.ListJournalEntries(JournalFilters{})
	if len(entries) != 1 || !entries[0].Pinned || entries[0].Title != "Updated" {
		t.Fatalf("update missing: %#v", entries)
	}
	if err := m.DeleteJournalEntry(id); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteJournalTypeKeepsEntriesAsGeneral(t *testing.T) {
	m := newJournalTestManager(t)
	typeID, _ := m.CreateJournalType("Coding", "#ffffff")
	m.CreateJournalEntry(JournalEntryInput{TypeID: typeID, Title: "Build", Day: "2026-10-05"})
	if err := m.DeleteJournalType(typeID); err != nil {
		t.Fatal(err)
	}
	entries, _ := m.ListJournalEntries(JournalFilters{})
	if len(entries) != 1 || entries[0].TypeID != 0 || entries[0].TypeName != "General" {
		t.Fatalf("entry was not preserved: %#v", entries)
	}
}
