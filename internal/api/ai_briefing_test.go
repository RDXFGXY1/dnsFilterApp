package api

import "testing"

func TestParseAIBriefingValidatesAndNormalizes(t *testing.T) {
	x, err := parseAIBriefing("```json\n{\"subject\":\"Weekly risk\",\"body\":\"Two tasks are overdue.\",\"category\":\"planning\",\"severity\":\"attention\",\"action_label\":\"Review\",\"action_view\":\"tasks\"}\n```")
	if err != nil || x.Subject != "Weekly risk" || x.ActionView != "tasks" {
		t.Fatalf("unexpected briefing: %#v err=%v", x, err)
	}
	x, err = parseAIBriefing(`{"subject":"Safe","body":"Body","severity":"invented","action_view":"settings","action_label":"Unsafe"}`)
	if err != nil || x.Severity != "normal" || x.ActionView != "" || x.ActionLabel != "" {
		t.Fatalf("normalization failed: %#v err=%v", x, err)
	}
}
