package config

import (
	"reflect"
	"testing"
)

func TestNormalizeSelectedProviders_Claude(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expected  []string
		expectErr bool
	}{
		{
			name:     "claude only",
			input:    []string{"claude"},
			expected: []string{"claude"},
		},
		{
			name:     "all four providers",
			input:    []string{"antigravity", "codex", "claude", "xai"},
			expected: []string{"antigravity", "codex", "claude", "xai"},
		},
		{
			name:     "with duplicates and whitespace",
			input:    []string{" claude ", "Claude", "codex"},
			expected: []string{"claude", "codex"},
		},
		{
			name:      "unsupported provider",
			input:     []string{"claude", "unsupported"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSelectedProviders(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestParseProviderScopeValue_Claude(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedScope    ProviderScope
		expectedSelected []string
		expectErr        bool
	}{
		{
			name:             "all",
			input:            "all",
			expectedScope:    ProviderScopeAll,
			expectedSelected: nil,
		},
		{
			name:             "claude single",
			input:            "claude",
			expectedScope:    ProviderScopeSelected,
			expectedSelected: []string{"claude"},
		},
		{
			name:             "pipe separated",
			input:            "antigravity|codex|claude|xai",
			expectedScope:    ProviderScopeSelected,
			expectedSelected: []string{"antigravity", "codex", "claude", "xai"},
		},
		{
			name:             "comma separated",
			input:            "claude, codex",
			expectedScope:    ProviderScopeSelected,
			expectedSelected: []string{"claude", "codex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope, selected, err := ParseProviderScopeValue(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if scope != tt.expectedScope {
				t.Errorf("expected scope %v, got %v", tt.expectedScope, scope)
			}
			if !reflect.DeepEqual(selected, tt.expectedSelected) {
				t.Errorf("expected selected %v, got %v", tt.expectedSelected, selected)
			}
		})
	}
}
