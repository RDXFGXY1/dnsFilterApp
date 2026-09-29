package hardsystem

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func testManager(t *testing.T) *Manager {
	db, err := sql.Open("sqlite", ":memory:")
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
func TestScoresExplicitPageHigh(t *testing.T) {
	m := testManager(t)
	r, err := m.Analyze(Request{URL: "https://example.test", Title: "Adult videos", Text: "porn xxx explicit video nude video hardcore adult content porn"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Score < 70 || r.Action != "block" {
		t.Fatalf("unexpected result: %+v", r)
	}
}
func TestAllowsOrdinaryPage(t *testing.T) {
	m := testManager(t)
	r, _ := m.Analyze(Request{URL: "https://docs.example", Title: "Programming guide", Text: "Learn how to build a secure web application with examples and documentation."})
	if r.Score != 0 || r.Action != "allow" {
		t.Fatalf("unexpected result: %+v", r)
	}
}
func TestEducationContextReducesWeakSignals(t *testing.T) {
	m := testManager(t)
	r, _ := m.Analyze(Request{URL: "https://health.example", Title: "Medical education", Text: "Educational health research about sexual safety and prevention."})
	if r.Action == "block" {
		t.Fatalf("false positive: %+v", r)
	}
}
func TestRecentDoesNotStorePageText(t *testing.T) {
	m := testManager(t)
	_, _ = m.Analyze(Request{URL: "https://example.test/private/path", Text: "secret-form-value porn xxx"})
	items, err := m.Recent(10)
	if err != nil || len(items) != 1 {
		t.Fatalf("recent: %v %+v", err, items)
	}
	if items[0].Domain != "example.test" {
		t.Fatalf("stored unexpected value: %+v", items[0])
	}
}
