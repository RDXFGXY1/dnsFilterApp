package discipline

import "testing"

func TestRescheduleReminderPreservesItemAndReopensIt(t *testing.T) {
	m := newJournalTestManager(t)
	id, err := m.CreateReminder("Submit project", "2026-10-27 09:00")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetReminderDone(id, true); err != nil {
		t.Fatal(err)
	}
	if err := m.RescheduleReminder(id, "bad date"); err == nil {
		t.Fatal("invalid date accepted")
	}
	if err := m.RescheduleReminder(id, "2026-10-27 10:00"); err != nil {
		t.Fatal(err)
	}
	items, err := m.ListReminders()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != id || items[0].RemindAt != "2026-10-27 10:00" || items[0].Done {
		t.Fatalf("unexpected reminder after snooze: %#v", items)
	}
}
