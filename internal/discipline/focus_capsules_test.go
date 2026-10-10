package discipline

import "testing"

func TestFocusCapsuleBrowserDecisions(t *testing.T) {
	m := newJournalTestManager(t)
	capsule, err := m.SaveFocusCapsule(0, FocusCapsuleInput{
		Name:            "C++ Learning",
		AllowedDomains:  []string{"cppreference.com"},
		YouTubeChannels: []string{"@codelearn", "UC123"},
		BlockStyle:      "roast",
		Strict:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.StartFocusSession(0, "Learn C++", 25, false, capsule.ID); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		url      string
		channels []string
		allowed  bool
	}{
		{"same youtube channel", "https://www.youtube.com/watch?v=one", []string{"@CodeLearn"}, true},
		{"different youtube channel", "https://www.youtube.com/watch?v=two", []string{"@distraction"}, false},
		{"youtube homepage", "https://www.youtube.com/", nil, false},
		{"allowed documentation", "https://en.cppreference.com/w/cpp", nil, true},
		{"outside strict capsule", "https://example.com", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := m.EvaluateFocusPage(tc.url, tc.channels)
			if !decision.Active || decision.Allowed != tc.allowed || decision.Capsule != capsule.Name || decision.BlockStyle != "roast" {
				t.Fatalf("unexpected decision: %#v", decision)
			}
		})
	}
}

func TestFocusCapsulePersistsAndCanBeUpdated(t *testing.T) {
	m := newJournalTestManager(t)
	created, err := m.SaveFocusCapsule(0, FocusCapsuleInput{Name: "Development", AllowedApps: []string{"code.exe"}})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := m.SaveFocusCapsule(created.ID, FocusCapsuleInput{Name: "Development", AllowedApps: []string{"code.exe", "brave.exe"}, BlockStyle: "meme"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.AllowedApps) != 2 || updated.BlockStyle != "meme" {
		t.Fatalf("capsule update was not persisted: %#v", updated)
	}
	items, err := m.ListFocusCapsules()
	if err != nil || len(items) != 1 {
		t.Fatalf("capsule list mismatch: %#v %v", items, err)
	}
}

func TestFocusCapsuleLearnsInitialYouTubeChannel(t *testing.T) {
	m := newJournalTestManager(t)
	capsule, err := m.SaveFocusCapsule(0, FocusCapsuleInput{Name: "Study", Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	id, err := m.StartFocusSession(0, "Study", 25, false, capsule.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := m.EvaluateFocusPage("https://www.youtube.com/watch?v=first", []string{"Code Learn", "@codelearn"})
	if !first.Allowed {
		t.Fatalf("initial channel should be learned: %#v", first)
	}
	session, err := m.FocusSessionDetails(id)
	if err != nil || session.LockedYouTubeChannel != "@codelearn" {
		t.Fatalf("channel lock was not persisted: %#v %v", session, err)
	}
	if !m.EvaluateFocusPage("https://www.youtube.com/watch?v=next", []string{"@codelearn"}).Allowed {
		t.Fatal("a later video from the learned channel should be allowed")
	}
	if m.EvaluateFocusPage("https://www.youtube.com/watch?v=other", []string{"@other"}).Allowed {
		t.Fatal("a different channel should be blocked")
	}
}
