package config

import (
	"testing"
	"time"
)

func TestLoadBytes_ClaudePriorityRules_JSON(t *testing.T) {
	configJSON := `{
		"enabled": true,
		"auto_apply": true,
		"provider_scope": "claude",
		"interval": "10m",
		"priority_rules": {
			"enabled": true,
			"claude": {
				"free_depleted_priority": -1,
				"free_depleted_disabled": true,
				"paid_depleted_disabled": false
			}
		}
	}`

	cfg, err := LoadBytes([]byte(configJSON))
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if !cfg.Enabled {
		t.Errorf("expected Enabled=true")
	}
	if !cfg.AutoApply {
		t.Errorf("expected AutoApply=true")
	}
	if cfg.ProviderScope != ProviderScopeSelected {
		t.Errorf("expected ProviderScopeSelected, got %v", cfg.ProviderScope)
	}
	if len(cfg.SelectedProviders) != 1 || cfg.SelectedProviders[0] != "claude" {
		t.Errorf("expected SelectedProviders=['claude'], got %v", cfg.SelectedProviders)
	}
	if cfg.Interval != 10*time.Minute {
		t.Errorf("expected Interval=10m, got %v", cfg.Interval)
	}
	if !cfg.PriorityRules.Enabled {
		t.Errorf("expected PriorityRules.Enabled=true")
	}
	if cfg.PriorityRules.Claude.FreeDepletedPriority != -1 {
		t.Errorf("expected Claude.FreeDepletedPriority=-1, got %d", cfg.PriorityRules.Claude.FreeDepletedPriority)
	}
	if !cfg.PriorityRules.Claude.FreeDepletedDisabled {
		t.Errorf("expected Claude.FreeDepletedDisabled=true")
	}
	if cfg.PriorityRules.Claude.PaidDepletedDisabled {
		t.Errorf("expected Claude.PaidDepletedDisabled=false")
	}
}

func TestLoadBytes_ClaudePriorityRules_YAML(t *testing.T) {
	configYAML := `
enabled: true
auto_apply: true
provider_scope: "antigravity|codex|claude|xai"
priority_rules:
  enabled: true
  claude:
    free_depleted_priority: -1
    free_depleted_disabled: false
    paid_depleted_disabled: true
`

	cfg, err := LoadBytes([]byte(configYAML))
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if len(cfg.SelectedProviders) != 4 {
		t.Fatalf("expected 4 selected providers, got %v", cfg.SelectedProviders)
	}
	if cfg.PriorityRules.Claude.FreeDepletedDisabled {
		t.Errorf("expected Claude.FreeDepletedDisabled=false")
	}
	if !cfg.PriorityRules.Claude.PaidDepletedDisabled {
		t.Errorf("expected Claude.PaidDepletedDisabled=true")
	}
}

func TestLoadBytes_ClaudeLegacyKeepsEnabled(t *testing.T) {
	configJSON := `{
		"priority_rules": {
			"claude": {
				"paid_depleted_keeps_enabled": true
			}
		}
	}`

	cfg, err := LoadBytes([]byte(configJSON))
	if err != nil {
		t.Fatalf("LoadBytes failed: %v", err)
	}

	if cfg.PriorityRules.Claude.PaidDepletedDisabled {
		t.Errorf("expected PaidDepletedDisabled=false when paid_depleted_keeps_enabled=true")
	}
}

func TestLoadBytes_UnsupportedPriorityProvider(t *testing.T) {
	configJSON := `{
		"priority_rules": {
			"unknown_provider": {}
		}
	}`

	_, err := LoadBytes([]byte(configJSON))
	if err == nil {
		t.Errorf("expected error for unsupported priority provider, got nil")
	}
}
