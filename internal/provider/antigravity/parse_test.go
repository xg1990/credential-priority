package antigravity

import (
	"testing"
	"time"

	"credential-priority/internal/config"
	"credential-priority/internal/core"
)

func TestParseAvailableModels_GeminiGroup(t *testing.T) {
	observedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)

	payload := `{
		"models": {
			"gemini-2.5-pro": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.85,
					"resetTime": "2026-08-20T15:00:00Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Provider != core.ProviderAntigravity {
		t.Errorf("expected ProviderAntigravity, got %v", result.Provider)
	}
	if result.Remaining == nil || *result.Remaining != 85 {
		t.Errorf("expected remaining 85, got %v", result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(resetAt) {
		t.Errorf("expected resetAt %v, got %v", resetAt, result.ResetAt)
	}
}

func TestParseAvailableModels_ClaudeGPTGroup(t *testing.T) {
	observedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)

	payload := `{
		"models": {
			"claude-3-7-sonnet": {
				"modelProvider": "anthropic",
				"quotaInfo": {
					"remainingFraction": 0.60,
					"resetTime": "2026-08-20T16:00:00Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupClaudeGPT)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Remaining == nil || *result.Remaining != 60 {
		t.Errorf("expected remaining 60, got %v", result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(resetAt) {
		t.Errorf("expected resetAt %v, got %v", resetAt, result.ResetAt)
	}
}
