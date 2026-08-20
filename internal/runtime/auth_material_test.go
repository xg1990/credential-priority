package runtime

import (
	"encoding/json"
	"testing"
)

func TestAccessTokenFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected string
	}{
		{
			name:     "standard access_token",
			jsonStr:  `{"access_token": "token-123"}`,
			expected: "token-123",
		},
		{
			name:     "session_key fallback",
			jsonStr:  `{"session_key": "sk-ant-sid-456"}`,
			expected: "sk-ant-sid-456",
		},
		{
			name:     "token fallback",
			jsonStr:  `{"token": "tok-789"}`,
			expected: "tok-789",
		},
		{
			name:     "empty",
			jsonStr:  `{}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := accessTokenFromJSON(json.RawMessage(tt.jsonStr))
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestOrganizationUUIDFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected string
	}{
		{
			name:     "organization_uuid",
			jsonStr:  `{"organization_uuid": "org-uuid-1"}`,
			expected: "org-uuid-1",
		},
		{
			name:     "org_uuid",
			jsonStr:  `{"org_uuid": "org-uuid-2"}`,
			expected: "org-uuid-2",
		},
		{
			name:     "organization_id",
			jsonStr:  `{"organization_id": "org-id-3"}`,
			expected: "org-id-3",
		},
		{
			name:     "empty",
			jsonStr:  `{}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := organizationUUIDFromJSON(json.RawMessage(tt.jsonStr))
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
