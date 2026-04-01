package binding

import (
	"fmt"
	"regexp"
)

// StageBinding represents a single stage's binding configuration.
type StageBinding struct {
	Description      string            `yaml:"description"`
	Orchestration    string            `yaml:"orchestration"`
	Roles            []string          `yaml:"roles"`
	Skills           []string          `yaml:"skills"`
	HumanGate        bool              `yaml:"human_gate"`
	DocumentType     *string           `yaml:"document_type,omitempty"`
	Prerequisites    *Prerequisites    `yaml:"prerequisites,omitempty"`
	Notes            string            `yaml:"notes,omitempty"`
	EffortBudget     string            `yaml:"effort_budget,omitempty"`
	MaxReviewCycles  *int              `yaml:"max_review_cycles,omitempty"`
	SubAgents        *SubAgents        `yaml:"sub_agents,omitempty"`
	DocumentTemplate *DocumentTemplate `yaml:"document_template,omitempty"`
}

// Prerequisites declares what must be true before entering the stage.
type Prerequisites struct {
	Documents []DocumentPrereq `yaml:"documents,omitempty"`
	Tasks     *TaskPrereq      `yaml:"tasks,omitempty"`
}

// DocumentPrereq is a single document prerequisite declaration.
type DocumentPrereq struct {
	Type   string `yaml:"type"`
	Status string `yaml:"status"`
}

// TaskPrereq declares task completion prerequisites.
// Exactly one of MinCount or AllTerminal may be set, not both.
type TaskPrereq struct {
	MinCount    *int  `yaml:"min_count,omitempty"`
	AllTerminal *bool `yaml:"all_terminal,omitempty"`
}

// SubAgents declares the worker configuration for orchestrator-workers stages.
type SubAgents struct {
	Roles     []string `yaml:"roles"`
	Skills    []string `yaml:"skills"`
	Topology  string   `yaml:"topology"`
	MaxAgents *int     `yaml:"max_agents,omitempty"`
}

// DocumentTemplate declares required structure for documents produced in a stage.
type DocumentTemplate struct {
	RequiredSections         []string `yaml:"required_sections"`
	CrossReferences          []string `yaml:"cross_references,omitempty"`
	AcceptanceCriteriaFormat string   `yaml:"acceptance_criteria_format,omitempty"`
}

// BindingFile is the top-level structure of stage-bindings.yaml.
type BindingFile struct {
	StageBindings map[string]*StageBinding `yaml:"stage_bindings"`
}

var validOrchestrations = map[string]bool{
	"single-agent":         true,
	"orchestrator-workers": true,
}

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

var roleIDRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,28}[a-z0-9]$|^[a-z0-9]{2}$`)
var skillNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$|^[a-z0-9]{2}$`)

// ValidateBinding checks a StageBinding for correctness and returns all errors found.
func ValidateBinding(b *StageBinding, stageName string) []error {
	var errs []error

	if b.Description == "" {
		errs = append(errs, fmt.Errorf("%s: description must not be empty", stageName))
	}

	if !validOrchestrations[b.Orchestration] {
		errs = append(errs, fmt.Errorf("%s: invalid orchestration %q", stageName, b.Orchestration))
	}

	if len(b.Roles) == 0 {
		errs = append(errs, fmt.Errorf("%s: roles must not be empty", stageName))
	}
	for _, r := range b.Roles {
		if !roleIDRegexp.MatchString(r) {
			errs = append(errs, fmt.Errorf("%s: invalid role ID %q", stageName, r))
		}
	}

	if len(b.Skills) == 0 {
		errs = append(errs, fmt.Errorf("%s: skills must not be empty", stageName))
	}
	for _, s := range b.Skills {
		if !skillNameRegexp.MatchString(s) {
			errs = append(errs, fmt.Errorf("%s: invalid skill name %q", stageName, s))
		}
	}

	if b.SubAgents != nil && b.Orchestration != "orchestrator-workers" {
		errs = append(errs, fmt.Errorf("%s: sub_agents requires orchestration \"orchestrator-workers\"", stageName))
	}
	if b.Orchestration == "orchestrator-workers" && b.SubAgents == nil {
		errs = append(errs, fmt.Errorf("%s: orchestration \"orchestrator-workers\" requires sub_agents", stageName))
	}

	if b.SubAgents != nil {
		if len(b.SubAgents.Roles) == 0 {
			errs = append(errs, fmt.Errorf("%s: sub_agents.roles must not be empty", stageName))
		}
		if len(b.SubAgents.Skills) == 0 {
			errs = append(errs, fmt.Errorf("%s: sub_agents.skills must not be empty", stageName))
		}
		if b.SubAgents.Topology != "parallel" {
			errs = append(errs, fmt.Errorf("%s: sub_agents.topology must be \"parallel\"", stageName))
		}
		if b.SubAgents.MaxAgents != nil && *b.SubAgents.MaxAgents < 1 {
			errs = append(errs, fmt.Errorf("%s: sub_agents.max_agents must be >= 1", stageName))
		}
	}

	if b.Prerequisites != nil && b.Prerequisites.Tasks != nil {
		tp := b.Prerequisites.Tasks
		hasMin := tp.MinCount != nil
		hasAll := tp.AllTerminal != nil
		if hasMin && hasAll {
			errs = append(errs, fmt.Errorf("%s: prerequisites.tasks must set exactly one of min_count or all_terminal, not both", stageName))
		}
		if !hasMin && !hasAll {
			errs = append(errs, fmt.Errorf("%s: prerequisites.tasks must set exactly one of min_count or all_terminal", stageName))
		}
		if hasMin && *tp.MinCount < 1 {
			errs = append(errs, fmt.Errorf("%s: prerequisites.tasks.min_count must be >= 1", stageName))
		}
	}

	if b.DocumentTemplate != nil {
		if len(b.DocumentTemplate.RequiredSections) == 0 {
			errs = append(errs, fmt.Errorf("%s: document_template.required_sections must not be empty", stageName))
		}
	}

	if b.MaxReviewCycles != nil && *b.MaxReviewCycles < 1 {
		errs = append(errs, fmt.Errorf("%s: max_review_cycles must be >= 1", stageName))
	}

	return errs
}
