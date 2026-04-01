package skill

import (
	"fmt"
	"regexp"
)

// skillNameRegexp validates skill names: lowercase alphanumeric with hyphens, 2-40 chars.
var skillNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$|^[a-z0-9]{2}$`)

// roleIDRegexp validates role IDs: lowercase alphanumeric with hyphens, 2-30 chars.
var roleIDRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,28}[a-z0-9]$|^[a-z0-9]{2}$`)

var validStages = map[string]bool{
	"designing":      true,
	"specifying":     true,
	"dev-planning":   true,
	"developing":     true,
	"reviewing":      true,
	"researching":    true,
	"documenting":    true,
	"plan-reviewing": true,
}

var validConstraintLevels = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

// SkillDescription is the dual-register description (expert + natural).
type SkillDescription struct {
	Expert  string `yaml:"expert"  json:"expert"`
	Natural string `yaml:"natural" json:"natural"`
}

// SkillFrontmatter is the YAML frontmatter parsed from SKILL.md.
// Strict parsing: unknown fields are rejected.
type SkillFrontmatter struct {
	Name            string           `yaml:"name"`
	Description     SkillDescription `yaml:"description"`
	Triggers        []string         `yaml:"triggers"`
	Roles           []string         `yaml:"roles"`
	Stage           string           `yaml:"stage"`
	ConstraintLevel string           `yaml:"constraint_level"`
}

// BodySection represents a parsed ## section from the SKILL.md body.
type BodySection struct {
	Heading string // the heading text (e.g., "Vocabulary")
	Content string // everything between this heading and the next ## heading
}

// Skill is the fully parsed and validated representation of a skill directory.
type Skill struct {
	Frontmatter    SkillFrontmatter
	Sections       []BodySection
	ReferencePaths []string // paths relative to the skill directory
	ScriptPaths    []string // paths relative to the skill directory
	SourcePath     string   // absolute path to the SKILL.md file
}

// validateFrontmatter checks that a parsed frontmatter meets all invariants.
// It accumulates all errors rather than stopping at the first.
func validateFrontmatter(fm *SkillFrontmatter, expectedName string) []error {
	var errs []error

	if fm.Name == "" {
		errs = append(errs, fmt.Errorf("missing required field 'name'"))
	} else if !skillNameRegexp.MatchString(fm.Name) {
		errs = append(errs, fmt.Errorf("invalid skill name %q: must be lowercase alphanumeric and hyphens, 2-40 chars", fm.Name))
	}

	if fm.Name != "" && fm.Name != expectedName {
		errs = append(errs, fmt.Errorf("skill name %q does not match directory name %q", fm.Name, expectedName))
	}

	if fm.Description.Expert == "" {
		errs = append(errs, fmt.Errorf("missing required field 'description.expert'"))
	}
	if fm.Description.Natural == "" {
		errs = append(errs, fmt.Errorf("missing required field 'description.natural'"))
	}

	if len(fm.Triggers) == 0 {
		errs = append(errs, fmt.Errorf("missing required field 'triggers': at least one trigger is required"))
	}

	if len(fm.Roles) == 0 {
		errs = append(errs, fmt.Errorf("missing required field 'roles': at least one role is required"))
	} else {
		for _, role := range fm.Roles {
			if !roleIDRegexp.MatchString(role) {
				errs = append(errs, fmt.Errorf("invalid role ID %q: must be lowercase alphanumeric and hyphens, 2-30 chars", role))
			}
		}
	}

	if fm.Stage == "" {
		errs = append(errs, fmt.Errorf("missing required field 'stage'"))
	} else if !validStages[fm.Stage] {
		errs = append(errs, fmt.Errorf("invalid stage %q", fm.Stage))
	}

	if fm.ConstraintLevel == "" {
		errs = append(errs, fmt.Errorf("missing required field 'constraint_level'"))
	} else if !validConstraintLevels[fm.ConstraintLevel] {
		errs = append(errs, fmt.Errorf("invalid constraint_level %q", fm.ConstraintLevel))
	}

	return errs
}
