package api

import (
	"errors"
	"testing"
)

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

func TestFriendlyAIProviderErrorHidesUpstreamPayload(t *testing.T) {
	got := friendlyAIProviderError(errors.New(`Gemini returned HTTP 503: {"error":{"message":"high demand","internal":"large upstream payload"}}`))
	if got != "The AI provider is temporarily unavailable. Local analysis was used instead." {
		t.Fatalf("unexpected friendly error: %q", got)
	}
	if got := friendlyAIProviderError(errors.New("provider returned HTTP 429")); got != "The AI provider is rate-limited. Local analysis was used instead." {
		t.Fatalf("unexpected rate-limit error: %q", got)
	}
}
