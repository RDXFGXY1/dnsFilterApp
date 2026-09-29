package api

import (
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/aiblock"
)

func TestDecisionGuidance(t *testing.T) {
	tests := []struct {
		name    string
		blocked bool
		ai      aiblock.PredictionResult
		want    string
	}{
		{name: "blocked is high risk", blocked: true, want: "high"},
		{name: "strong model signal needs review", ai: aiblock.PredictionResult{Category: "malware", Confidence: .91}, want: "review"},
		{name: "safe allowed domain is low risk", ai: aiblock.PredictionResult{Category: "safe", Confidence: .95}, want: "low"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, recommendation := decisionGuidance(tt.blocked, tt.ai)
			if got != tt.want {
				t.Fatalf("risk = %q, want %q", got, tt.want)
			}
			if recommendation == "" {
				t.Fatal("recommendation is empty")
			}
		})
	}
}
