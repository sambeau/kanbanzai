package binding

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// StageBinding represents a single stage's binding configuration.
type StageBinding struct {
	Description         string               `yaml:"description"`
	Orchestration       string               `yaml:"orchestration"`
	Roles               []string             `yaml:"roles"`
	Skills              []string             `yaml:"skills"`
	HumanGate           bool                 `yaml:"human_gate"`
	DocumentType        *string              `yaml:"document_type,omitempty"`
	Prerequisites       *Prerequisites       `yaml:"prerequisites,omitempty"`
	TransitionValidator *TransitionValidator `yaml:"transition_validator,omitempty"`
	Notes               string               `yaml:"notes,omitempty"`
	EffortBudget        string               `yaml:"effort_budget,omitempty"`
	MaxReviewCycles     *int                 `yaml:"max_review_cycles,omitempty"`
	SubAgents           *SubAgents           `yaml:"sub_agents,omitempty"`
	DocumentTemplate    *DocumentTemplate    `yaml:"document_template,omitempty"`

	// Profile, Tier, Modes, and Verifying support stages that opt into the
	// gated-mode profile schema (e.g. retro-fixing). They are decoded but not
	// yet consumed by the pipeline; full schema work is tracked separately.
	Profile   *bool                 `yaml:"profile,omitempty"`
	Tier      string                `yaml:"tier,omitempty"`
	Modes     map[string]*StageMode `yaml:"modes,omitempty"`
	Verifying *VerifyingBlock       `yaml:"verifying,omitempty"`
}

// StageMode declares the gate configuration for a single mode of a profiled
// stage (see StageBinding.Modes). All gate fields are optional strings such
// as "human", "auto", or "conditional".
type StageMode struct {
	DesignGate      string `yaml:"design_gate,omitempty"`
	SpecGate        string `yaml:"spec_gate,omitempty"`
	DevPlanGate     string `yaml:"dev_plan_gate,omitempty"`
	ReviewGate      string `yaml:"review_gate,omitempty"`
	MaxReviewCycles *int   `yaml:"max_review_cycles,omitempty"`
	Notes           string `yaml:"notes,omitempty"`
}

// VerifyingBlock declares the verifier role/skill bound to a profiled stage's
// post-implementation verification step.
type VerifyingBlock struct {
	Roles      []string `yaml:"roles,omitempty"`
	Skills     []string `yaml:"skills,omitempty"`
	DoDVariant string   `yaml:"dod_variant,omitempty"`
}

type Prerequisites struct {
	Documents      []DocumentPrereq `yaml:"documents,omitempty"`
	Tasks          *TaskPrereq      `yaml:"tasks,omitempty"`
	OverridePolicy string           `yaml:"override_policy,omitempty"` // "agent" (default) or "checkpoint"

	// Extensions holds any prerequisite type keys not natively understood by the
	// binding loader. Each key is passed to the registered evaluator dispatcher,
	// which returns "unknown prerequisite type" if no evaluator is registered.
	Extensions map[string]yaml.Node `yaml:"-"`
}

// TransitionValidator declares a validator hook that runs when a feature
// transitions out of this stage. It is separate from prerequisites —
// prerequisites gate entry into the stage, transition validators gate exit.
type TransitionValidator struct {
	Role     string `yaml:"role"`
	Skill    string `yaml:"skill"`
	GateMode string `yaml:"gate_mode"` // "auto" (default) or "human"
}

// UnmarshalYAML implements yaml.Unmarshaler so that unknown keys in a
// prerequisites block are captured in Extensions rather than causing a decode
// error. Known keys (documents, tasks, override_policy) are decoded into their
// typed fields as normal.
func (p *Prerequisites) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("prerequisites must be a mapping")
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		k := value.Content[i].Value
		v := value.Content[i+1]
		switch k {
		case "documents":
			if err := v.Decode(&p.Documents); err != nil {
				return fmt.Errorf("prerequisites.documents: %w", err)
			}
		case "tasks":
			p.Tasks = new(TaskPrereq)
			if err := v.Decode(p.Tasks); err != nil {
				return fmt.Errorf("prerequisites.tasks: %w", err)
			}
		case "override_policy":
			p.OverridePolicy = v.Value
		default:
			if p.Extensions == nil {
				p.Extensions = make(map[string]yaml.Node)
			}
			p.Extensions[k] = *v
		}
	}
	return nil
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
	SchemaVersion int                      `yaml:"schema_version"`
	StageBindings map[string]*StageBinding `yaml:"stage_bindings"`
}

var validOrchestrations = map[string]bool{
	"single-agent":         true,
	"orchestrator-workers": true,
	"pipeline-coordinator": true,
}

var validStages = map[string]bool{
	"designing":       true,
	"specifying":      true,
	"dev-planning":    true,
	"developing":      true,
	"reviewing":       true,
	"merging":         true,
	"verifying":       true,
	"batch-reviewing": true,
	"researching":     true,
	"documenting":     true,
	"doc-publishing":  true,
	"retro-fixing":    true,
}

var validTopologies = map[string]bool{
	"parallel":   true,
	"sequential": true,
	"single":     true,
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

	// REQ-004: sub_agents is valid with any orchestration that declares it.
	// orchestrator-workers still requires sub_agents.
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
		if !validTopologies[b.SubAgents.Topology] {
			errs = append(errs, fmt.Errorf("%s: sub_agents.topology must be one of: parallel, sequential, single", stageName))
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

	if b.Prerequisites != nil && b.Prerequisites.OverridePolicy != "" {
		switch b.Prerequisites.OverridePolicy {
		case "agent", "checkpoint":
			// valid
		default:
			errs = append(errs, fmt.Errorf("%s: prerequisites.override_policy must be \"agent\" or \"checkpoint\", got %q", stageName, b.Prerequisites.OverridePolicy))
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
