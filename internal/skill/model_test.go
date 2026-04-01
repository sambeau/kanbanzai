package skill

import (
	"strings"
	"testing"
)

func validFrontmatter() SkillFrontmatter {
	return SkillFrontmatter{
		Name: "my-skill",
		Description: SkillDescription{
			Expert:  "Expert description of the skill",
			Natural: "Natural language description of the skill",
		},
		Triggers:        []string{"when the user asks about X"},
		Roles:           []string{"backend", "frontend"},
		Stage:           "developing",
		ConstraintLevel: "medium",
	}
}

func TestValidateFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		modify       func(*SkillFrontmatter)
		expectedName string
		wantErrs     int
		wantSubstr   string // substring expected in at least one error
	}{
		{
			name:         "valid frontmatter passes",
			modify:       func(_ *SkillFrontmatter) {},
			expectedName: "my-skill",
			wantErrs:     0,
		},
		{
			name:         "missing name",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "missing required field 'name'",
		},
		{
			name:         "invalid name format - uppercase",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "My-Skill" },
			expectedName: "My-Skill",
			wantErrs:     1,
			wantSubstr:   "invalid skill name",
		},
		{
			name:         "invalid name format - too short",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "a" },
			expectedName: "a",
			wantErrs:     1,
			wantSubstr:   "invalid skill name",
		},
		{
			name:         "invalid name format - leading hyphen",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "-bad" },
			expectedName: "-bad",
			wantErrs:     1,
			wantSubstr:   "invalid skill name",
		},
		{
			name:         "invalid name format - trailing hyphen",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "bad-" },
			expectedName: "bad-",
			wantErrs:     1,
			wantSubstr:   "invalid skill name",
		},
		{
			name:         "name does not match expected",
			modify:       func(fm *SkillFrontmatter) { fm.Name = "other-skill" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "does not match directory name",
		},
		{
			name:         "empty description expert",
			modify:       func(fm *SkillFrontmatter) { fm.Description.Expert = "" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "description.expert",
		},
		{
			name:         "empty description natural",
			modify:       func(fm *SkillFrontmatter) { fm.Description.Natural = "" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "description.natural",
		},
		{
			name:         "empty triggers",
			modify:       func(fm *SkillFrontmatter) { fm.Triggers = nil },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "triggers",
		},
		{
			name:         "empty triggers - zero length slice",
			modify:       func(fm *SkillFrontmatter) { fm.Triggers = []string{} },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "triggers",
		},
		{
			name:         "empty roles",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = nil },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "roles",
		},
		{
			name:         "empty roles - zero length slice",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = []string{} },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "roles",
		},
		{
			name:         "invalid role ID format - uppercase",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = []string{"Backend"} },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "invalid role ID",
		},
		{
			name:         "invalid role ID format - single char",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = []string{"a"} },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "invalid role ID",
		},
		{
			name:         "invalid role ID format - underscore",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = []string{"has_under"} },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "invalid role ID",
		},
		{
			name:         "multiple invalid role IDs",
			modify:       func(fm *SkillFrontmatter) { fm.Roles = []string{"BAD", "Also Bad"} },
			expectedName: "my-skill",
			wantErrs:     2,
			wantSubstr:   "invalid role ID",
		},
		{
			name:         "invalid stage",
			modify:       func(fm *SkillFrontmatter) { fm.Stage = "flying" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "invalid stage",
		},
		{
			name:         "missing stage",
			modify:       func(fm *SkillFrontmatter) { fm.Stage = "" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "missing required field 'stage'",
		},
		{
			name:         "invalid constraint level",
			modify:       func(fm *SkillFrontmatter) { fm.ConstraintLevel = "extreme" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "invalid constraint_level",
		},
		{
			name:         "missing constraint level",
			modify:       func(fm *SkillFrontmatter) { fm.ConstraintLevel = "" },
			expectedName: "my-skill",
			wantErrs:     1,
			wantSubstr:   "missing required field 'constraint_level'",
		},
		{
			name: "multiple errors accumulated",
			modify: func(fm *SkillFrontmatter) {
				fm.Name = ""
				fm.Description.Expert = ""
				fm.Description.Natural = ""
				fm.Triggers = nil
				fm.Roles = nil
				fm.Stage = ""
				fm.ConstraintLevel = ""
			},
			expectedName: "my-skill",
			wantErrs:     7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fm := validFrontmatter()
			tc.modify(&fm)

			errs := validateFrontmatter(&fm, tc.expectedName)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d", len(errs), tc.wantErrs)
				for i, err := range errs {
					t.Errorf("  error[%d]: %v", i, err)
				}
				return
			}

			if tc.wantSubstr != "" {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), tc.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected an error containing %q, got:", tc.wantSubstr)
					for i, err := range errs {
						t.Errorf("  error[%d]: %v", i, err)
					}
				}
			}
		})
	}
}

func TestValidStages(t *testing.T) {
	stages := []string{
		"designing", "specifying", "dev-planning", "developing",
		"reviewing", "researching", "documenting", "plan-reviewing",
	}
	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			fm := validFrontmatter()
			fm.Stage = stage
			errs := validateFrontmatter(&fm, "my-skill")
			if len(errs) != 0 {
				t.Errorf("stage %q should be valid, got errors: %v", stage, errs)
			}
		})
	}
}

func TestValidConstraintLevels(t *testing.T) {
	levels := []string{"low", "medium", "high"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			fm := validFrontmatter()
			fm.ConstraintLevel = level
			errs := validateFrontmatter(&fm, "my-skill")
			if len(errs) != 0 {
				t.Errorf("constraint_level %q should be valid, got errors: %v", level, errs)
			}
		})
	}
}

func TestSkillNameRegexp(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"ab", true},
		{"a1", true},
		{"my-skill", true},
		{"skill-name-42", true},
		{"ab-cd-ef-gh-ij-kl-mn-op-qr-st-uv-wx-yz", true}, // 40 chars
		{"a", false},
		{"", false},
		{"-bad", false},
		{"bad-", false},
		{"Bad", false},
		{"has space", false},
		{"has_under", false},
	}

	for _, tc := range tests {
		matched := skillNameRegexp.MatchString(tc.name)
		if matched != tc.valid {
			t.Errorf("skill name %q: got valid=%v, want valid=%v", tc.name, matched, tc.valid)
		}
	}
}

func TestRoleIDRegexp(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"ab", true},
		{"backend", true},
		{"dev-ops", true},
		{"a", false},
		{"", false},
		{"-bad", false},
		{"bad-", false},
		{"Bad", false},
		{"has space", false},
		{"has_under", false},
	}

	for _, tc := range tests {
		matched := roleIDRegexp.MatchString(tc.id)
		if matched != tc.valid {
			t.Errorf("role ID %q: got valid=%v, want valid=%v", tc.id, matched, tc.valid)
		}
	}
}
