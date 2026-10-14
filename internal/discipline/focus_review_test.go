package discipline

import "testing"

func TestFocusReviewIsLinkedToSession(t *testing.T) {
	m := newJournalTestManager(t)
	sessionID, err := m.StartFocusSession(0, "Write release notes", 25, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.EndFocusSession(sessionID, "completed", "", 8, false); err != nil {
		t.Fatal(err)
	}
	messageID, err := m.CreateInboxMessage(InboxInput{Subject: "Focus review", Body: "A useful review"})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.SetFocusReviewState(sessionID, "ready", "ai", messageID, ""); err != nil {
		t.Fatal(err)
	}
	session, err := m.FocusSessionDetails(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if session.ReviewStatus != "ready" || session.ReviewMode != "ai" || session.Review == nil || session.Review.ID != messageID {
		t.Fatalf("review was not restored with session: %#v", session)
	}
}
