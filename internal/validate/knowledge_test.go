package validate

import (
	"testing"
)

func TestIsKnownKnowledgeStatus(t *testing.T) {
	t.Parallel()

	known := []string{"contributed", "confirmed", "disputed", "stale", "retired"}
	for _, s := range known {
		if !IsKnownKnowledgeStatus(s) {
			t.Errorf("IsKnownKnowledgeStatus(%q) = false, want true", s)
		}
	}

	unknown := []string{"", "active", "done", "proposed", "archived"}
	for _, s := range unknown {
		if IsKnownKnowledgeStatus(s) {
			t.Errorf("IsKnownKnowledgeStatus(%q) = true, want false", s)
		}
	}
}

func TestCanTransitionKnowledge(t *testing.T) {
	t.Parallel()

	valid := []struct {
		from string
		to   string
	}{
		{"contributed", "confirmed"},
		{"contributed", "disputed"},
		{"contributed", "retired"},
		{"confirmed", "disputed"},
		{"confirmed", "stale"},
		{"confirmed", "retired"},
		{"disputed", "confirmed"},
		{"disputed", "retired"},
		{"stale", "confirmed"},
		{"stale", "retired"},
	}

	for _, tc := range valid {
		if !CanTransitionKnowledge(tc.from, tc.to) {
			t.Errorf("CanTransitionKnowledge(%q, %q) = false, want true", tc.from, tc.to)
		}
	}

	invalid := []struct {
		from string
		to   string
	}{
		// Terminal state
		{"retired", "contributed"},
		{"retired", "confirmed"},
		{"retired", "disputed"},
		{"retired", "stale"},
		// Self-transition
		{"contributed", "contributed"},
		{"confirmed", "confirmed"},
		// Not in transition table
		{"stale", "disputed"},
		{"disputed", "stale"},
		{"contributed", "stale"},
	}

	for _, tc := range invalid {
		if CanTransitionKnowledge(tc.from, tc.to) {
			t.Errorf("CanTransitionKnowledge(%q, %q) = true, want false", tc.from, tc.to)
		}
	}
}

func TestValidateKnowledgeTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{
			name:    "contributed to confirmed",
			from:    "contributed",
			to:      "confirmed",
			wantErr: false,
		},
		{
			name:    "contributed to disputed",
			from:    "contributed",
			to:      "disputed",
			wantErr: false,
		},
		{
			name:    "contributed to retired",
			from:    "contributed",
			to:      "retired",
			wantErr: false,
		},
		{
			name:    "confirmed to disputed",
			from:    "confirmed",
			to:      "disputed",
			wantErr: false,
		},
		{
			name:    "confirmed to stale",
			from:    "confirmed",
			to:      "stale",
			wantErr: false,
		},
		{
			name:    "confirmed to retired",
			from:    "confirmed",
			to:      "retired",
			wantErr: false,
		},
		{
			name:    "disputed to confirmed",
			from:    "disputed",
			to:      "confirmed",
			wantErr: false,
		},
		{
			name:    "disputed to retired",
			from:    "disputed",
			to:      "retired",
			wantErr: false,
		},
		{
			name:    "stale to confirmed",
			from:    "stale",
			to:      "confirmed",
			wantErr: false,
		},
		{
			name:    "stale to retired",
			from:    "stale",
			to:      "retired",
			wantErr: false,
		},
		// Invalid transitions
		{
			name:    "self-transition",
			from:    "contributed",
			to:      "contributed",
			wantErr: true,
		},
		{
			name:    "from terminal retired",
			from:    "retired",
			to:      "confirmed",
			wantErr: true,
		},
		{
			name:    "stale to disputed (not allowed)",
			from:    "stale",
			to:      "disputed",
			wantErr: true,
		},
		{
			name:    "contributed to stale (not allowed)",
			from:    "contributed",
			to:      "stale",
			wantErr: true,
		},
		{
			name:    "unknown from status",
			from:    "bogus",
			to:      "confirmed",
			wantErr: true,
		},
		{
			name:    "unknown to status",
			from:    "contributed",
			to:      "bogus",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateKnowledgeTransition(tc.from, tc.to)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateKnowledgeTransition(%q, %q) error = %v, wantErr %v", tc.from, tc.to, err, tc.wantErr)
			}
		})
	}
}

func TestValidateKnowledgeFields(t *testing.T) {
	t.Parallel()

	t.Run("valid fields", func(t *testing.T) {
		t.Parallel()
		fields := map[string]any{
			"id":         "KE-01ABC",
			"tier":       3,
			"topic":      "api-json-naming",
			"scope":      "project",
			"content":    "Use camelCase for JSON field names.",
			"status":     "contributed",
			"created":    "2024-01-01T00:00:00Z",
			"created_by": "agent",
			"updated":    "2024-01-01T00:00:00Z",
		}
		if err := ValidateKnowledgeFields(fields); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		t.Parallel()
		fields := map[string]any{
			"id":    "KE-01ABC",
			"tier":  3,
			"topic": "api-json-naming",
			// missing scope, content, status, created, created_by, updated
		}
		if err := ValidateKnowledgeFields(fields); err == nil {
			t.Error("expected error for missing required fields, got nil")
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		t.Parallel()
		fields := map[string]any{
			"id":         "KE-01ABC",
			"tier":       3,
			"topic":      "api-json-naming",
			"scope":      "project",
			"content":    "Use camelCase.",
			"status":     "invalid-status",
			"created":    "2024-01-01T00:00:00Z",
			"created_by": "agent",
			"updated":    "2024-01-01T00:00:00Z",
		}
		if err := ValidateKnowledgeFields(fields); err == nil {
			t.Error("expected error for invalid status, got nil")
		}
	})
}
