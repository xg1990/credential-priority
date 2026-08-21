package priority

import (
	"testing"
	"time"

	"credential-priority/internal/core"
)

func TestResetBoost_Claude(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	nearReset := now.Add(2 * time.Hour)
	farReset := now.Add(48 * time.Hour)

	options := Options{
		Now:              now,
		ResetBoostWithin: 24 * time.Hour,
		ResetBoost:       50,
	}

	tests := []struct {
		name     string
		item     PlanItem
		expected int
	}{
		{
			name: "claude pro near 5h reset",
			item: PlanItem{
				Credential: core.Credential{
					Provider: core.ProviderClaude,
					Type:     core.CredentialTypeClaude,
				},
				PlanType: core.PlanTypePro,
				ResetAt:  &nearReset,
			},
			expected: 50,
		},
		{
			name: "claude pro far reset",
			item: PlanItem{
				Credential: core.Credential{
					Provider: core.ProviderClaude,
					Type:     core.CredentialTypeClaude,
				},
				PlanType: core.PlanTypePro,
				ResetAt:  &farReset,
			},
			expected: 0,
		},
		{
			name: "claude free near reset",
			item: PlanItem{
				Credential: core.Credential{
					Provider: core.ProviderClaude,
					Type:     core.CredentialTypeClaude,
				},
				PlanType: core.PlanTypeFree,
				ResetAt:  &nearReset,
			},
			expected: 50,
		},
		{
			name: "claude with long window near reset",
			item: PlanItem{
				Credential: core.Credential{
					Provider: core.ProviderClaude,
					Type:     core.CredentialTypeClaude,
				},
				PlanType:          core.PlanTypePro,
				LongWindowResetAt: &nearReset,
			},
			expected: 50,
		},
		{
			name: "nil reset",
			item: PlanItem{
				Credential: core.Credential{
					Provider: core.ProviderClaude,
					Type:     core.CredentialTypeClaude,
				},
				PlanType: core.PlanTypePro,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resetBoost(tt.item, options)
			if got != tt.expected {
				t.Errorf("resetBoost() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestResetBoost_AntigravityAndCodexShortWindow(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	nearReset := now.Add(2 * time.Hour)
	farLongWindow := now.Add(48 * time.Hour)

	options := Options{
		Now:              now,
		ResetBoostWithin: 24 * time.Hour,
		ResetBoost:       50,
	}

	tests := []struct {
		name     string
		provider core.Provider
		item     PlanItem
		expected int
	}{
		{
			name:     "antigravity near 5h reset with far weekly window",
			provider: core.ProviderAntigravity,
			item: PlanItem{
				PlanType:          core.PlanTypePro,
				ResetAt:           &nearReset,
				LongWindowResetAt: &farLongWindow,
			},
			expected: 50,
		},
		{
			name:     "codex near 5h reset with far weekly window",
			provider: core.ProviderCodex,
			item: PlanItem{
				PlanType:          core.PlanTypePro,
				ResetAt:           &nearReset,
				LongWindowResetAt: &farLongWindow,
			},
			expected: 50,
		},
		{
			name:     "xai near 5h reset with far weekly window stays unboosted",
			provider: core.ProviderXAI,
			item: PlanItem{
				PlanType:          core.PlanTypePro,
				ResetAt:           &nearReset,
				LongWindowResetAt: &farLongWindow,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.item.Credential = core.Credential{Provider: tt.provider}
			got := resetBoost(tt.item, options)
			if got != tt.expected {
				t.Errorf("resetBoost() = %d, expected %d", got, tt.expected)
			}
		})
	}
}

func TestPlannedPriority_Claude(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	nearReset := now.Add(1 * time.Hour)
	farReset := now.Add(48 * time.Hour)

	options := Options{
		Now:              now,
		ResetBoostWithin: 24 * time.Hour,
		ResetBoost:       50,
	}

	boostedItem := PlanItem{
		Credential: core.Credential{Provider: core.ProviderClaude, Type: core.CredentialTypeClaude},
		PlanType:   core.PlanTypePro,
		ResetAt:    &nearReset,
	}
	if p := plannedPriority(boostedItem, 100, options); p != 999 {
		t.Errorf("expected boosted priority 999, got %d", p)
	}

	regularItem := PlanItem{
		Credential: core.Credential{Provider: core.ProviderClaude, Type: core.CredentialTypeClaude},
		PlanType:   core.PlanTypePro,
		ResetAt:    &farReset,
	}
	if p := plannedPriority(regularItem, 100, options); p != 100 {
		t.Errorf("expected regular priority 100, got %d", p)
	}
}
