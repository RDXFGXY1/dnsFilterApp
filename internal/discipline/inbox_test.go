package discipline

import "testing"

func TestInboxLifecycleAndBriefing(t *testing.T) {
	m := newJournalTestManager(t)
	id, err := m.CreateInboxMessage(InboxInput{Subject: "Review your week", Body: "One task needs attention", ActionView: "tasks"})
	if err != nil || id == 0 {
		t.Fatalf("create inbox message: id=%d err=%v", id, err)
	}
	items, err := m.ListInboxMessages("unread", "review")
	if err != nil || len(items) != 1 {
		t.Fatalf("list unread: %#v err=%v", items, err)
	}
	read, star := true, true
	if err = m.UpdateInboxMessage(id, InboxPatch{IsRead: &read, Starred: &star}); err != nil {
		t.Fatal(err)
	}
	stats, err := m.InboxStats()
	if err != nil || stats.Unread != 0 || stats.Starred != 1 {
		t.Fatalf("bad stats: %#v err=%v", stats, err)
	}
	if _, err = m.GenerateDailyBriefing(); err != nil {
		t.Fatal(err)
	}
	items, _ = m.ListInboxMessages("inbox", "briefing")
	if len(items) != 1 {
		t.Fatalf("daily briefing missing: %#v", items)
	}
	archived := true
	if err = m.UpdateInboxMessage(id, InboxPatch{Archived: &archived}); err != nil {
		t.Fatal(err)
	}
	items, _ = m.ListInboxMessages("archived", "")
	if len(items) != 1 {
		t.Fatalf("archive missing: %#v", items)
	}
	if err = m.DeleteInboxMessage(id); err != nil {
		t.Fatal(err)
	}
}
