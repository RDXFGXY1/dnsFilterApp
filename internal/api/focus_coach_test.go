package api

import (
	"strings"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
)

func TestLocalFocusCoachUsesOnlySessionEvidence(t *testing.T) {
	session := discipline.FocusSession{
		TaskTitle:      "Learn C++",
		PlannedMinutes: 25,
		ActualSeconds:  1200,
		Activity: []discipline.FocusActivity{
			{Kind: "app", Name: "brave.exe", Observations: 5},
			{Kind: "domain", Name: "youtube.com", Observations: 8},
		},
		Distractions: []discipline.Distraction{{Label: "Phone"}},
	}
	message := localFocusCoach(session)
	if message.Category != "Focus coach" || message.ActionView != "command" {
		t.Fatalf("unexpected coach metadata: %#v", message)
	}
	for _, evidence := range []string{"20 focused minute", "1 application", "1 DNS-visible domain", "13 activity observations", "1"} {
		if !strings.Contains(message.Body, evidence) {
			t.Fatalf("coach review missing %q: %s", evidence, message.Body)
		}
	}
}
