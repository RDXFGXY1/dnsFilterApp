package discipline

import "testing"

func TestFocusSessionActivityAndAutomaticJournal(t *testing.T) {
	m := newJournalTestManager(t)
	id, err := m.StartFocusSession(7, "Study networking", 25, false)
	if err != nil {
		t.Fatal(err)
	}
	m.RecordFocusActivity("domain", "docs.example.com", 0)
	m.RecordFocusActivity("domain", "docs.example.com", 0)
	if err := m.AddDistraction(id, "Phone", "notification"); err != nil {
		t.Fatal(err)
	}
	session, err := m.EndFocusSession(id, "completed", "Learned DNS caching", 8, true)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != "completed" || len(session.Activity) != 1 || session.Activity[0].Observations != 2 || len(session.Distractions) != 1 {
		t.Fatalf("unexpected focus session: %#v", session)
	}
	entries, err := m.ListJournalEntries(JournalFilters{Query: "DNS caching"})
	if err != nil || len(entries) != 1 || entries[0].Focus != 8 {
		t.Fatalf("automatic journal entry missing: %#v %v", entries, err)
	}
}
