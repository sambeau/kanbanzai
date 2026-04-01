package binding

import (
	"strings"
	"testing"
)

func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func validMinimalBinding() *StageBinding {
	return &StageBinding{
		Description:   "A test stage",
		Orchestration: "single-agent",
		Roles:         []string{"designer"},
		Skills:        []string{"design-skill"},
	}
}

func validFullBinding() *StageBinding {
	return &StageBinding{
		Description:   "Full stage with all fields",
		Orchestration: "orchestrator-workers",
		Roles:         []string{"orchestrator", "lead"},
		Skills:        []string{"planning", "code-review"},
		HumanGate:     true,
		DocumentType:  strPtr("specification"),
		Prerequisites: &Prerequisites{
			Documents: []DocumentPrereq{
				{Type: "design", Status: "approved"},
			},
			Tasks: &TaskPrereq{
				AllTerminal: boolPtr(true),
			},
		},
		Notes:           "Some notes",
		EffortBudget:    "medium",
		MaxReviewCycles: intPtr(3),
		SubAgents: &SubAgents{
			Roles:     []string{"backend", "frontend"},
			Skills:    []string{"implementation"},
			Topology:  "parallel",
			MaxAgents: intPtr(4),
		},
		DocumentTemplate: &DocumentTemplate{
			RequiredSections:         []string{"overview", "details"},
			CrossReferences:          []string{"design-doc"},
			AcceptanceCriteriaFormat: "checklist",
		},
	}
}

func TestValidateBinding(t *testing.T) {
	tests := []struct {
		name       string
		binding    *StageBinding
		stage      string
		wantErrors []string // substrings expected in error messages; nil means no errors
	}{
		{
			name:       "valid minimal binding",
			binding:    validMinimalBinding(),
			stage:      "designing",
			wantErrors: nil,
		},
		{
			name:       "valid full binding",
			binding:    validFullBinding(),
			stage:      "developing",
			wantErrors: nil,
		},
		{
			name: "missing description",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Description = ""
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"description must not be empty"},
		},
		{
			name: "invalid orchestration",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "multi-agent"
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid orchestration"},
		},
		{
			name: "empty roles",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Roles = nil
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"roles must not be empty"},
		},
		{
			name: "invalid role ID format",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Roles = []string{"Invalid-Role!"}
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid role ID"},
		},
		{
			name: "empty skills",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Skills = nil
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"skills must not be empty"},
		},
		{
			name: "invalid skill name format",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Skills = []string{"Bad Skill Name"}
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid skill name"},
		},
		{
			name: "sub_agents with single-agent orchestration",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "single-agent"
				b.SubAgents = &SubAgents{
					Roles:    []string{"worker"},
					Skills:   []string{"impl"},
					Topology: "parallel",
				}
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"sub_agents requires orchestration"},
		},
		{
			name: "orchestrator-workers without sub_agents",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = nil
				return b
			}(),
			stage:      "developing",
			wantErrors: []string{"orchestration \"orchestrator-workers\" requires sub_agents"},
		},
		{
			name: "sub_agents missing required fields",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = &SubAgents{
					Roles:    nil,
					Skills:   nil,
					Topology: "sequential",
				}
				return b
			}(),
			stage: "developing",
			wantErrors: []string{
				"sub_agents.roles must not be empty",
				"sub_agents.skills must not be empty",
				"sub_agents.topology must be \"parallel\"",
			},
		},
		{
			name: "sub_agents max_agents less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = &SubAgents{
					Roles:     []string{"worker"},
					Skills:    []string{"impl"},
					Topology:  "parallel",
					MaxAgents: intPtr(0),
				}
				return b
			}(),
			stage:      "developing",
			wantErrors: []string{"sub_agents.max_agents must be >= 1"},
		},
		{
			name: "prerequisites tasks with both min_count and all_terminal",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{
						MinCount:    intPtr(2),
						AllTerminal: boolPtr(true),
					},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"exactly one of min_count or all_terminal, not both"},
		},
		{
			name: "prerequisites tasks with neither min_count nor all_terminal",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"exactly one of min_count or all_terminal"},
		},
		{
			name: "prerequisites tasks min_count less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{
						MinCount: intPtr(0),
					},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"min_count must be >= 1"},
		},
		{
			name: "document_template with empty required_sections",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.DocumentTemplate = &DocumentTemplate{
					RequiredSections: nil,
				}
				return b
			}(),
			stage:      "specifying",
			wantErrors: []string{"document_template.required_sections must not be empty"},
		},
		{
			name: "max_review_cycles less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.MaxReviewCycles = intPtr(0)
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"max_review_cycles must be >= 1"},
		},
		{
			name: "multiple errors accumulated",
			binding: &StageBinding{
				Description:     "",
				Orchestration:   "bogus",
				Roles:           nil,
				Skills:          nil,
				MaxReviewCycles: intPtr(0),
			},
			stage: "designing",
			wantErrors: []string{
				"description must not be empty",
				"invalid orchestration",
				"roles must not be empty",
				"skills must not be empty",
				"max_review_cycles must be >= 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateBinding(tt.binding, tt.stage)

			if tt.wantErrors == nil {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %d:", len(errs))
					for _, e := range errs {
						t.Errorf("  %s", e)
					}
				}
				return
			}

			if len(errs) < len(tt.wantErrors) {
				t.Errorf("expected at least %d errors, got %d:", len(tt.wantErrors), len(errs))
				for _, e := range errs {
					t.Errorf("  %s", e)
				}
				return
			}

			errStrings := make([]string, len(errs))
			for i, e := range errs {
				errStrings[i] = e.Error()
			}
			joined := strings.Join(errStrings, "\n")

			for _, want := range tt.wantErrors {
				if !strings.Contains(joined, want) {
					t.Errorf("expected error containing %q, got errors:\n%s", want, joined)
				}
			}

			// Verify stage name is included in every error message.
			for _, e := range errs {
				if !strings.Contains(e.Error(), tt.stage) {
					t.Errorf("error %q does not contain stage name %q", e, tt.stage)
				}
			}
		})
	}
}

func TestRoleIDRegexp(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ab", true},
		{"designer", true},
		{"backend-dev", true},
		{"a1-b2-c3", true},
		{"a", false},
		{"-bad", false},
		{"bad-", false},
		{"BAD", false},
		{"has space", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := roleIDRegexp.MatchString(tt.input); got != tt.valid {
				t.Errorf("roleIDRegexp.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

func TestSkillNameRegexp(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ab", true},
		{"design-skill", true},
		{"code-review-and-analysis", true},
		{"a", false},
		{"-bad", false},
		{"bad-", false},
		{"BAD", false},
		{"has space", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := skillNameRegexp.MatchString(tt.input); got != tt.valid {
				t.Errorf("skillNameRegexp.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
