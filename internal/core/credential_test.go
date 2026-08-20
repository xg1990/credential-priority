package core

import "testing"

func TestCredential_Constants(t *testing.T) {
	if ProviderClaude != "claude" {
		t.Errorf("expected ProviderClaude to be 'claude', got %s", ProviderClaude)
	}
	if CredentialTypeClaude != "claude" {
		t.Errorf("expected CredentialTypeClaude to be 'claude', got %s", CredentialTypeClaude)
	}
	if StrategyClaude != "claude" {
		t.Errorf("expected StrategyClaude to be 'claude', got %s", StrategyClaude)
	}
}

func TestCredential_WithProbe(t *testing.T) {
	c := Credential{
		Name:      "test-claude",
		Provider:  ProviderClaude,
		Type:      CredentialTypeClaude,
		Priority:  10,
		Freshness: FreshnessUnknown,
	}

	probed := c.WithProbe(FreshnessFresh, ProbeStatusReady)
	if probed.Freshness != FreshnessFresh {
		t.Errorf("expected FreshnessFresh, got %s", probed.Freshness)
	}
	if probed.ProbeStatus != ProbeStatusReady {
		t.Errorf("expected ProbeStatusReady, got %s", probed.ProbeStatus)
	}
	if probed.Priority != 10 {
		t.Errorf("expected Priority 10 preserved, got %d", probed.Priority)
	}
}

func TestPromotionFromProbe(t *testing.T) {
	tests := []struct {
		freshness   Freshness
		probeStatus ProbeStatus
		expected    CanPromote
	}{
		{FreshnessFresh, ProbeStatusReady, CanPromoteAfterFreshProbe},
		{FreshnessStale, ProbeStatusReady, CannotPromote},
		{FreshnessFresh, ProbeStatusUnsupported, CannotPromote},
		{FreshnessUnknown, ProbeStatusUnknown, CannotPromote},
	}

	for _, tt := range tests {
		got := PromotionFromProbe(tt.freshness, tt.probeStatus)
		if got != tt.expected {
			t.Errorf("PromotionFromProbe(%v, %v) = %v, expected %v", tt.freshness, tt.probeStatus, got, tt.expected)
		}
	}
}
