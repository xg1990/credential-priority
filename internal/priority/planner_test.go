package priority

import (
	"testing"
	"time"

	"credential-priority/internal/core"
)

func TestPlanFreshOnly_Claude_PositiveRemaining(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(30 * time.Hour) // not near reset (> 24h)
	rem1 := int64(45)
	rem2 := int64(30)

	credentials := []core.Credential{
		{
			Name:      "claude-1",
			AuthIndex: "auth-c1",
			Provider:  core.ProviderClaude,
			Type:      core.CredentialTypeClaude,
			Priority:  0,
		},
		{
			Name:      "claude-2",
			AuthIndex: "auth-c2",
			Provider:  core.ProviderClaude,
			Type:      core.CredentialTypeClaude,
			Priority:  0,
		},
	}

	evidence := []ProbeEvidence{
		{
			Provider:      core.ProviderClaude,
			AuthIndex:     "auth-c1",
			ObservedAt:    now,
			ResetAt:       &resetAt,
			Remaining:     &rem1,
			Freshness:     core.FreshnessFresh,
			ProbeStatus:   core.ProbeStatusReady,
			Status:        EvidenceStatusReady,
			PlanType:      core.PlanTypePro,
			EvidenceFresh: true,
		},
		{
			Provider:      core.ProviderClaude,
			AuthIndex:     "auth-c2",
			ObservedAt:    now,
			ResetAt:       &resetAt,
			Remaining:     &rem2,
			Freshness:     core.FreshnessFresh,
			ProbeStatus:   core.ProbeStatusReady,
			Status:        EvidenceStatusReady,
			PlanType:      core.PlanTypePro,
			EvidenceFresh: true,
		},
	}

	options := Options{
		Now:         now,
		MaxPriority: 100,
		StartPriorityByProvider: map[core.Provider]int{
			core.ProviderClaude: 100,
		},
		ResetBoostWithin: 24 * time.Hour,
		ResetBoost:       50,
	}

	plan := PlanFreshOnly(credentials, evidence, options)
	if len(plan.Items) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan.Items))
	}

	// Higher remaining (45) should have higher priority (100), then (30) gets (99)
	var item1, item2 *PlanItem
	for i := range plan.Items {
		if plan.Items[i].Credential.AuthIndex == "auth-c1" {
			item1 = &plan.Items[i]
		} else if plan.Items[i].Credential.AuthIndex == "auth-c2" {
			item2 = &plan.Items[i]
		}
	}

	if item1 == nil || item2 == nil {
		t.Fatalf("missing items in plan")
	}
	if item1.Priority != 100 {
		t.Errorf("expected item1 priority 100, got %d", item1.Priority)
	}
	if item2.Priority != 99 {
		t.Errorf("expected item2 priority 99, got %d", item2.Priority)
	}
	if item1.Disabled || item2.Disabled {
		t.Errorf("expected active credentials not disabled")
	}
}

func TestPlanFreshOnly_Claude_NearResetBoost(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAtNear := now.Add(30 * time.Minute) // near reset (< 24h)
	rem := int64(10)

	credentials := []core.Credential{
		{
			Name:      "claude-boost",
			AuthIndex: "auth-cb",
			Provider:  core.ProviderClaude,
			Type:      core.CredentialTypeClaude,
			Priority:  10,
		},
	}

	evidence := []ProbeEvidence{
		{
			Provider:      core.ProviderClaude,
			AuthIndex:     "auth-cb",
			ObservedAt:    now,
			ResetAt:       &resetAtNear,
			Remaining:     &rem,
			Freshness:     core.FreshnessFresh,
			ProbeStatus:   core.ProbeStatusReady,
			Status:        EvidenceStatusReady,
			PlanType:      core.PlanTypePro,
			EvidenceFresh: true,
		},
	}

	options := Options{
		Now:         now,
		MaxPriority: 100,
		StartPriorityByProvider: map[core.Provider]int{
			core.ProviderClaude: 100,
		},
		ResetBoostWithin: 24 * time.Hour,
		ResetBoost:       50,
	}

	plan := PlanFreshOnly(credentials, evidence, options)
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan.Items))
	}
	if plan.Items[0].Priority != 999 {
		t.Errorf("expected boosted priority 999, got %d", plan.Items[0].Priority)
	}
}

func TestPlanFreshOnly_Claude_FreeDepleted(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(2 * time.Hour)
	zeroRem := int64(0)

	credentials := []core.Credential{
		{
			Name:      "claude-free",
			AuthIndex: "auth-cf",
			Provider:  core.ProviderClaude,
			Type:      core.CredentialTypeClaude,
			Disabled:  false,
		},
	}

	evidence := []ProbeEvidence{
		{
			Provider:      core.ProviderClaude,
			AuthIndex:     "auth-cf",
			ObservedAt:    now,
			ResetAt:       &resetAt,
			Remaining:     &zeroRem,
			Freshness:     core.FreshnessFresh,
			ProbeStatus:   core.ProbeStatusReady,
			Status:        EvidenceStatusReady,
			PlanType:      core.PlanTypeFree,
			EvidenceFresh: true,
		},
	}

	depletedPriority := -1
	depletedDisabled := true
	options := Options{
		Now:                        now,
		MaxPriority:                100,
		ClaudeFreeDepletedPriority: &depletedPriority,
		ClaudeFreeDepletedDisabled: &depletedDisabled,
	}

	plan := PlanFreshOnly(credentials, evidence, options)
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan.Items))
	}
	if plan.Items[0].Priority != -1 {
		t.Errorf("expected priority -1, got %d", plan.Items[0].Priority)
	}
	if !plan.Items[0].Disabled {
		t.Errorf("expected Disabled=true for free depleted")
	}
}

func TestPlanFreshOnly_Claude_PaidDepleted(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(2 * time.Hour)
	zeroRem := int64(0)

	credentials := []core.Credential{
		{
			Name:      "claude-pro",
			AuthIndex: "auth-cp",
			Provider:  core.ProviderClaude,
			Type:      core.CredentialTypeClaude,
			Disabled:  false,
		},
	}

	evidence := []ProbeEvidence{
		{
			Provider:      core.ProviderClaude,
			AuthIndex:     "auth-cp",
			ObservedAt:    now,
			ResetAt:       &resetAt,
			Remaining:     &zeroRem,
			Freshness:     core.FreshnessFresh,
			ProbeStatus:   core.ProbeStatusReady,
			Status:        EvidenceStatusReady,
			PlanType:      core.PlanTypePro,
			EvidenceFresh: true,
		},
	}

	depletedPriority := -1
	depletedDisabled := false // paid depleted keeps enabled
	options := Options{
		Now:                        now,
		MaxPriority:                100,
		ClaudeFreeDepletedPriority: &depletedPriority,
		ClaudePaidDepletedDisabled: &depletedDisabled,
	}

	plan := PlanFreshOnly(credentials, evidence, options)
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan.Items))
	}
	if plan.Items[0].Priority != -1 {
		t.Errorf("expected priority -1, got %d", plan.Items[0].Priority)
	}
	if plan.Items[0].Disabled {
		t.Errorf("expected Disabled=false for paid depleted")
	}
}

func TestPlanFreshOnly_MultiProvider(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(30 * time.Hour)
	rem := int64(50)

	credentials := []core.Credential{
		{Name: "c1", AuthIndex: "auth-claude", Provider: core.ProviderClaude, Type: core.CredentialTypeClaude},
		{Name: "c2", AuthIndex: "auth-codex", Provider: core.ProviderCodex, Type: core.CredentialTypeCodex},
		{Name: "c3", AuthIndex: "auth-ag", Provider: core.ProviderAntigravity, Type: core.CredentialTypeAntigravity},
	}

	evidence := []ProbeEvidence{
		{Provider: core.ProviderClaude, AuthIndex: "auth-claude", ObservedAt: now, ResetAt: &resetAt, Remaining: &rem, Freshness: core.FreshnessFresh, ProbeStatus: core.ProbeStatusReady, Status: EvidenceStatusReady, PlanType: core.PlanTypePro, EvidenceFresh: true},
		{Provider: core.ProviderCodex, AuthIndex: "auth-codex", ObservedAt: now, ResetAt: &resetAt, Remaining: &rem, Freshness: core.FreshnessFresh, ProbeStatus: core.ProbeStatusReady, Status: EvidenceStatusReady, PlanType: core.PlanTypePro, EvidenceFresh: true},
		{Provider: core.ProviderAntigravity, AuthIndex: "auth-ag", ObservedAt: now, ResetAt: &resetAt, Remaining: &rem, Freshness: core.FreshnessFresh, ProbeStatus: core.ProbeStatusReady, Status: EvidenceStatusReady, PlanType: core.PlanTypePro, EvidenceFresh: true},
	}

	options := Options{
		Now:         now,
		MaxPriority: 100,
		StartPriorityByProvider: map[core.Provider]int{
			core.ProviderClaude:      100,
			core.ProviderCodex:       100,
			core.ProviderAntigravity: 100,
		},
	}

	plan := PlanFreshOnly(credentials, evidence, options)
	if len(plan.Items) != 3 {
		t.Fatalf("expected 3 plan items, got %d", len(plan.Items))
	}

	for _, item := range plan.Items {
		if item.Priority != 100 {
			t.Errorf("expected priority 100 for %s, got %d", item.Credential.Name, item.Priority)
		}
	}
}
