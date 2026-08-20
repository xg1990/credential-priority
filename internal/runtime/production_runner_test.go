package runtime

import (
	"context"
	"testing"

	"credential-priority/internal/apply"
	"credential-priority/internal/core"
	"credential-priority/internal/priority"
)

type recordingApplyHost struct {
	priorityWrites int
	disabledWrites int
}

func (h *recordingApplyHost) PatchPriority(_ context.Context, _ string, _ int) error {
	h.priorityWrites++
	return nil
}

func (h *recordingApplyHost) PatchDisabled(_ context.Context, _ string, _ bool) error {
	h.disabledWrites++
	return nil
}

type recordingApplyAuditor struct{}

func (recordingApplyAuditor) SaveSnapshot(context.Context, apply.PlanSnapshot) error { return nil }
func (recordingApplyAuditor) RecordEvent(context.Context, apply.AuditEvent) error     { return nil }

func TestPreserveProbeFailureState(t *testing.T) {
	credential := core.Credential{
		AuthIndex: "claude-auth",
		Provider:  core.ProviderClaude,
		Priority:  90,
		Disabled:  false,
	}
	plan := priority.Plan{
		Items: []priority.PlanItem{{
			Credential:    credential,
			Priority:      999,
			Disabled:      true,
			EvidenceFresh: true,
			ForceWrite:    true,
			Reason:        "provider priority uniqueness",
		}},
		Changes: []priority.Change{{
			Credential:    credential,
			Priority:      999,
			Disabled:      true,
			EvidenceFresh: true,
			Reason:        "provider priority uniqueness",
		}},
	}
	evidence := []priority.ProbeEvidence{{
		Provider:  core.ProviderClaude,
		AuthIndex: credential.AuthIndex,
		Status:    priority.EvidenceStatusProbeFailed,
	}}

	plan = preserveProbeFailureState(plan, evidence)
	item := plan.Items[0]
	if item.Priority != credential.Priority || item.Disabled != credential.Disabled {
		t.Fatalf("state = priority %d disabled %t, want priority %d disabled %t", item.Priority, item.Disabled, credential.Priority, credential.Disabled)
	}
	if item.EvidenceFresh || item.ForceWrite {
		t.Fatal("probe failure must not retain write eligibility")
	}
	if item.Reason != "failedQuotaFetch" {
		t.Fatalf("reason = %q, want failedQuotaFetch", item.Reason)
	}
	if len(plan.Changes) != 0 {
		t.Fatalf("changes = %d, want 0", len(plan.Changes))
	}

	host := &recordingApplyHost{}
	result, err := apply.Apply(context.Background(), apply.Request{
		Host:    host,
		Auditor: recordingApplyAuditor{},
		Plan:    plan,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Attempted != 0 || host.priorityWrites != 0 || host.disabledWrites != 0 {
		t.Fatalf("unexpected write: result=%+v priorityWrites=%d disabledWrites=%d", result, host.priorityWrites, host.disabledWrites)
	}
}

func TestPreserveProbeFailureStateLeavesTrustedQuotaChange(t *testing.T) {
	failedCredential := core.Credential{AuthIndex: "claude-failed", Provider: core.ProviderClaude, Priority: 90}
	trustedCredential := core.Credential{AuthIndex: "claude-depleted", Provider: core.ProviderClaude, Priority: 90}
	plan := priority.Plan{
		Items: []priority.PlanItem{
			{Credential: failedCredential, Priority: 999, Disabled: true, EvidenceFresh: true, ForceWrite: true},
			{Credential: trustedCredential, Priority: -1, Disabled: true, EvidenceFresh: true, Reason: "fresh remaining depleted"},
		},
		Changes: []priority.Change{
			{Credential: failedCredential, Priority: 999, Disabled: true, EvidenceFresh: true},
			{Credential: trustedCredential, Priority: -1, Disabled: true, EvidenceFresh: true, Reason: "fresh remaining depleted"},
		},
	}
	evidence := []priority.ProbeEvidence{
		{Provider: core.ProviderClaude, AuthIndex: failedCredential.AuthIndex, Status: priority.EvidenceStatusProbeFailed},
		{Provider: core.ProviderClaude, AuthIndex: trustedCredential.AuthIndex, Status: priority.EvidenceStatusReady, EvidenceFresh: true},
	}

	plan = preserveProbeFailureState(plan, evidence)
	if got := plan.Items[0]; got.Priority != 90 || got.Disabled || got.EvidenceFresh || got.ForceWrite {
		t.Fatalf("failed item = %+v, want preserved state", got)
	}
	if len(plan.Changes) != 1 || plan.Changes[0].Credential.AuthIndex != trustedCredential.AuthIndex {
		t.Fatalf("changes = %+v, want only trusted quota change", plan.Changes)
	}
}
