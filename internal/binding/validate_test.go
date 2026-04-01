package binding

import (
	"strings"
	"testing"
)

func TestValidateBindingFile(t *testing.T) {
	// Helper role checkers.
	allRolesExist := func(string) bool { return true }
	noRolesExist := func(string) bool { return false }

	knownRoles := func(known map[string]bool) RoleChecker {
		return func(id string) bool { return known[id] }
	}

	validBindingFile := func() *BindingFile {
		return &BindingFile{
			StageBindings: map[string]*StageBinding{
				"designing": {
					Description:   "Design stage",
					Orchestration: "single-agent",
					Roles:         []string{"designer"},
					Skills:        []string{"design-skill"},
				},
				"reviewing": {
					Description:   "Review stage",
					Orchestration: "single-agent",
					Roles:         []string{"reviewer"},
					Skills:        []string{"code-review"},
					HumanGate:     true,
				},
			},
		}
	}

	tests := []struct {
		name         string
		bf           *BindingFile
		roleChecker  RoleChecker
		wantErrors   []string // substrings expected in error messages; nil means no errors
		wantWarnings []string // substrings expected in warnings; nil means no warnings
	}{
		{
			name:        "valid binding file passes with no errors or warnings",
			bf:          validBindingFile(),
			roleChecker: allRolesExist,
		},
		{
			name: "invalid stage name produces error with valid-stage list",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"cooking": {
						Description:   "Not a real stage",
						Orchestration: "single-agent",
						Roles:         []string{"chef"},
						Skills:        []string{"cooking-skill"},
					},
				},
			},
			roleChecker: allRolesExist,
			wantErrors:  []string{"invalid stage name \"cooking\"", "valid stages:"},
		},
		{
			name: "role exists produces no warning",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"developing": {
						Description:   "Dev stage",
						Orchestration: "single-agent",
						Roles:         []string{"backend"},
						Skills:        []string{"implementation"},
					},
				},
			},
			roleChecker: knownRoles(map[string]bool{"backend": true}),
		},
		{
			name: "role fallback to parent produces warning",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"developing": {
						Description:   "Dev stage",
						Orchestration: "single-agent",
						Roles:         []string{"reviewer-security"},
						Skills:        []string{"implementation"},
					},
				},
			},
			roleChecker:  knownRoles(map[string]bool{"reviewer": true}),
			wantWarnings: []string{"role \"reviewer-security\" not found, resolved via fallback to \"reviewer\" in stage \"developing\""},
		},
		{
			name: "role completely unresolvable produces warning not error",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"developing": {
						Description:   "Dev stage",
						Orchestration: "single-agent",
						Roles:         []string{"nonexistent"},
						Skills:        []string{"implementation"},
					},
				},
			},
			roleChecker:  noRolesExist,
			wantWarnings: []string{"role \"nonexistent\" not found in stage \"developing\""},
		},
		{
			name: "nil roleChecker skips role checks",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"designing": {
						Description:   "Design stage",
						Orchestration: "single-agent",
						Roles:         []string{"totally-missing-role"},
						Skills:        []string{"design-skill"},
					},
				},
			},
			roleChecker: nil,
		},
		{
			name: "multiple errors across bindings accumulated",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"designing": {
						Description:   "", // missing description
						Orchestration: "single-agent",
						Roles:         []string{"designer"},
						Skills:        []string{"design-skill"},
					},
					"reviewing": {
						Description:   "Review stage",
						Orchestration: "bogus", // invalid orchestration
						Roles:         []string{"reviewer"},
						Skills:        []string{"code-review"},
					},
				},
			},
			roleChecker: allRolesExist,
			wantErrors:  []string{"description must not be empty", "invalid orchestration"},
		},
		{
			name: "sub_agents with single-agent orchestration error propagated from ValidateBinding",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"developing": {
						Description:   "Dev stage",
						Orchestration: "single-agent",
						Roles:         []string{"lead"},
						Skills:        []string{"implementation"},
						SubAgents: &SubAgents{
							Roles:    []string{"worker"},
							Skills:   []string{"coding"},
							Topology: "parallel",
						},
					},
				},
			},
			roleChecker: allRolesExist,
			wantErrors:  []string{"sub_agents requires orchestration"},
		},
		{
			name: "role fallback checked for sub_agents roles too",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"developing": {
						Description:   "Dev stage",
						Orchestration: "orchestrator-workers",
						Roles:         []string{"lead"},
						Skills:        []string{"implementation"},
						SubAgents: &SubAgents{
							Roles:    []string{"backend-senior"},
							Skills:   []string{"coding"},
							Topology: "parallel",
						},
					},
				},
			},
			roleChecker:  knownRoles(map[string]bool{"lead": true, "backend": true}),
			wantWarnings: []string{"role \"backend-senior\" not found, resolved via fallback to \"backend\""},
		},
		{
			name: "role with no hyphen and not found produces unresolvable warning",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"designing": {
						Description:   "Design stage",
						Orchestration: "single-agent",
						Roles:         []string{"ghost"},
						Skills:        []string{"design-skill"},
					},
				},
			},
			roleChecker:  noRolesExist,
			wantWarnings: []string{"role \"ghost\" not found in stage \"designing\""},
		},
		{
			name: "role with hyphen but fallback also fails produces unresolvable warning",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"designing": {
						Description:   "Design stage",
						Orchestration: "single-agent",
						Roles:         []string{"ghost-rider"},
						Skills:        []string{"design-skill"},
					},
				},
			},
			roleChecker:  noRolesExist,
			wantWarnings: []string{"role \"ghost-rider\" not found in stage \"designing\""},
		},
		{
			name: "invalid stage and role fallback both reported",
			bf: &BindingFile{
				StageBindings: map[string]*StageBinding{
					"frying": {
						Description:   "Not a stage",
						Orchestration: "single-agent",
						Roles:         []string{"missing-role"},
						Skills:        []string{"frying-skill"},
					},
				},
			},
			roleChecker:  noRolesExist,
			wantErrors:   []string{"invalid stage name \"frying\""},
			wantWarnings: []string{"role \"missing-role\" not found in stage \"frying\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBindingFile(tt.bf, tt.roleChecker)

			// Check errors.
			if tt.wantErrors == nil {
				if len(result.Errors) != 0 {
					t.Errorf("expected no errors, got %d:", len(result.Errors))
					for _, e := range result.Errors {
						t.Errorf("  %s", e)
					}
				}
			} else {
				if len(result.Errors) == 0 {
					t.Fatal("expected errors, got none")
				}
				errStrings := make([]string, len(result.Errors))
				for i, e := range result.Errors {
					errStrings[i] = e.Error()
				}
				joined := strings.Join(errStrings, "\n")
				for _, want := range tt.wantErrors {
					if !strings.Contains(joined, want) {
						t.Errorf("expected error containing %q, got:\n%s", want, joined)
					}
				}
			}

			// Check warnings.
			if tt.wantWarnings == nil {
				if len(result.Warnings) != 0 {
					t.Errorf("expected no warnings, got %d:", len(result.Warnings))
					for _, w := range result.Warnings {
						t.Errorf("  %s", w)
					}
				}
			} else {
				if len(result.Warnings) == 0 {
					t.Fatal("expected warnings, got none")
				}
				joined := strings.Join(result.Warnings, "\n")
				for _, want := range tt.wantWarnings {
					if !strings.Contains(joined, want) {
						t.Errorf("expected warning containing %q, got:\n%s", want, joined)
					}
				}
			}
		})
	}
}
