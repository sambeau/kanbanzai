package context

import (
	"strings"
	"testing"
)

func TestValidateRole(t *testing.T) {
	validRole := func() *Role {
		return &Role{
			ID:       "reviewer-security",
			Identity: "Senior application security engineer",
			Vocabulary: []string{
				"OWASP Top 10", "CVE", "threat model",
			},
			AntiPatterns: []AntiPattern{
				{
					Name:    "Security Theatre",
					Detect:  "Flagging theoretical vulnerabilities without exploit path",
					Because: "Wastes developer time on non-issues",
					Resolve: "Require a concrete attack scenario for each finding",
				},
			},
			Tools: []string{"entity", "grep"},
		}
	}

	tests := []struct {
		name        string
		role        *Role
		expectedID  string
		wantErr     bool
		errContains []string
	}{
		{
			name:       "valid role passes",
			role:       validRole(),
			expectedID: "reviewer-security",
			wantErr:    false,
		},
		{
			name: "valid minimal role (no anti_patterns, no tools)",
			role: &Role{
				ID:         "base",
				Identity:   "Software engineer",
				Vocabulary: []string{"code review"},
			},
			expectedID: "base",
			wantErr:    false,
		},
		{
			name: "valid 2-char id",
			role: &Role{
				ID:         "ab",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID: "ab",
			wantErr:    false,
		},
		{
			name: "missing id",
			role: &Role{
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID:  "some-file",
			wantErr:     true,
			errContains: []string{"missing required field 'id'"},
		},
		{
			name: "invalid id format - uppercase",
			role: &Role{
				ID:         "BadId",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID:  "BadId",
			wantErr:     true,
			errContains: []string{"invalid id", "lowercase alphanumeric"},
		},
		{
			name: "invalid id format - leading hyphen",
			role: &Role{
				ID:         "-bad",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID:  "-bad",
			wantErr:     true,
			errContains: []string{"invalid id"},
		},
		{
			name: "invalid id format - too long",
			role: &Role{
				ID:         "this-id-is-way-too-long-for-the-limit",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID:  "this-id-is-way-too-long-for-the-limit",
			wantErr:     true,
			errContains: []string{"invalid id"},
		},
		{
			name: "invalid id format - single char",
			role: &Role{
				ID:         "a",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
			},
			expectedID:  "a",
			wantErr:     true,
			errContains: []string{"invalid id"},
		},
		{
			name: "id does not match filename",
			role: &Role{
				ID:         "reviewer",
				Identity:   "Code reviewer",
				Vocabulary: []string{"code review"},
			},
			expectedID:  "reviewer-security",
			wantErr:     true,
			errContains: []string{"does not match filename"},
		},
		{
			name: "missing identity",
			role: &Role{
				ID:         "test-role",
				Vocabulary: []string{"term"},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"missing required field 'identity'"},
		},
		{
			name: "identity exceeds 50 tokens",
			role: &Role{
				ID:         "test-role",
				Identity:   strings.Repeat("word ", 51),
				Vocabulary: []string{"term"},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"exceeds 50-token limit"},
		},
		{
			name: "identity exactly 50 tokens passes",
			role: &Role{
				ID:         "test-role",
				Identity:   strings.TrimSpace(strings.Repeat("word ", 50)),
				Vocabulary: []string{"term"},
			},
			expectedID: "test-role",
			wantErr:    false,
		},
		{
			name: "missing vocabulary",
			role: &Role{
				ID:       "test-role",
				Identity: "Engineer",
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"missing required field 'vocabulary'", "non-empty"},
		},
		{
			name: "empty vocabulary list",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"vocabulary", "non-empty"},
		},
		{
			name: "anti_pattern missing name",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{Detect: "d", Because: "b", Resolve: "r"},
				},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"anti_patterns[0]", "missing required field 'name'"},
		},
		{
			name: "anti_pattern missing detect",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{Name: "n", Because: "b", Resolve: "r"},
				},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"anti_patterns[0]", "missing required field 'detect'"},
		},
		{
			name: "anti_pattern missing because",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{Name: "n", Detect: "d", Resolve: "r"},
				},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"anti_patterns[0]", "missing required field 'because'"},
		},
		{
			name: "anti_pattern missing resolve",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{Name: "n", Detect: "d", Because: "b"},
				},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"anti_patterns[0]", "missing required field 'resolve'"},
		},
		{
			name: "anti_pattern all fields empty",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{},
				},
			},
			expectedID: "test-role",
			wantErr:    true,
			errContains: []string{
				"anti_patterns[0]: missing required field 'name'",
				"anti_patterns[0]: missing required field 'detect'",
				"anti_patterns[0]: missing required field 'because'",
				"anti_patterns[0]: missing required field 'resolve'",
			},
		},
		{
			name: "multiple anti_patterns validated independently",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				AntiPatterns: []AntiPattern{
					{Name: "good", Detect: "d", Because: "b", Resolve: "r"},
					{Name: "", Detect: "d", Because: "b", Resolve: "r"},
				},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"anti_patterns[1]", "missing required field 'name'"},
		},
		{
			name: "tools with duplicates",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				Tools:      []string{"entity", "grep", "entity"},
			},
			expectedID:  "test-role",
			wantErr:     true,
			errContains: []string{"tools: duplicate entry \"entity\""},
		},
		{
			name: "empty tools list is valid",
			role: &Role{
				ID:         "test-role",
				Identity:   "Engineer",
				Vocabulary: []string{"term"},
				Tools:      []string{},
			},
			expectedID: "test-role",
			wantErr:    false,
		},
		{
			name: "multiple errors accumulated",
			role: &Role{
				ID: "", // missing
				// Identity missing
				// Vocabulary missing
				AntiPatterns: []AntiPattern{
					{Name: "n"}, // missing detect, because, resolve
				},
				Tools: []string{"a", "a"}, // duplicate
			},
			expectedID: "bad-role",
			wantErr:    true,
			errContains: []string{
				"missing required field 'id'",
				"missing required field 'identity'",
				"vocabulary",
				"anti_patterns[0]: missing required field 'detect'",
				"anti_patterns[0]: missing required field 'because'",
				"anti_patterns[0]: missing required field 'resolve'",
				"tools: duplicate entry",
			},
		},
		{
			name: "inherits field does not affect validation",
			role: &Role{
				ID:         "child-role",
				Inherits:   "parent-role",
				Identity:   "Child engineer",
				Vocabulary: []string{"child-term"},
			},
			expectedID: "child-role",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRole(tt.role, tt.expectedID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				for _, substr := range tt.errContains {
					if !strings.Contains(err.Error(), substr) {
						t.Errorf("error %q does not contain %q", err.Error(), substr)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAntiPatternStruct(t *testing.T) {
	ap := AntiPattern{
		Name:    "Test",
		Detect:  "Detection signal",
		Because: "Reason",
		Resolve: "Resolution",
	}
	if ap.Name != "Test" || ap.Detect != "Detection signal" || ap.Because != "Reason" || ap.Resolve != "Resolution" {
		t.Error("AntiPattern struct fields not set correctly")
	}
}

func TestResolvedRoleStruct(t *testing.T) {
	rr := ResolvedRole{
		ID:         "test",
		Identity:   "Test engineer",
		Vocabulary: []string{"a", "b"},
		AntiPatterns: []AntiPattern{
			{Name: "ap1", Detect: "d", Because: "b", Resolve: "r"},
		},
		Tools: []string{"entity"},
	}
	if rr.ID != "test" {
		t.Error("ResolvedRole.ID not set")
	}
	if rr.Identity != "Test engineer" {
		t.Error("ResolvedRole.Identity not set")
	}
	if len(rr.Vocabulary) != 2 {
		t.Errorf("expected 2 vocabulary entries, got %d", len(rr.Vocabulary))
	}
	if len(rr.AntiPatterns) != 1 {
		t.Errorf("expected 1 anti-pattern, got %d", len(rr.AntiPatterns))
	}
	if len(rr.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(rr.Tools))
	}
}
