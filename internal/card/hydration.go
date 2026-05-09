package card

import (
	"github.com/sambeau/kanbanzai/internal/binding"
)

// BindingDocPrereq is the serialised form of a single document prerequisite.
type BindingDocPrereq struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// BindingTaskPrereq is the serialised form of a task completion prerequisite.
type BindingTaskPrereq struct {
	MinCount    *int  `json:"min_count,omitempty"`
	AllTerminal *bool `json:"all_terminal,omitempty"`
}

// BindingPrereqs is the serialised form of stage prerequisites.
// Fields absent in the source binding are omitted from JSON output.
type BindingPrereqs struct {
	Documents []BindingDocPrereq `json:"documents,omitempty"`
	Tasks     *BindingTaskPrereq `json:"tasks,omitempty"`
}

// BindingSubAgentProfile is the serialised form of the sub-agents configuration.
type BindingSubAgentProfile struct {
	Roles     []string `json:"roles"`
	Skills    []string `json:"skills"`
	Topology  string   `json:"topology"`
	MaxAgents *int     `json:"max_agents,omitempty"`
}

// BindingPayload carries machine-readable stage-binding fields for inclusion in
// task context responses (next, handoff). Fields absent in the source binding
// are omitted rather than zeroed so consumers see no spurious empty objects
// (REQ-006, AC-005).
type BindingPayload struct {
	Stage           string                  `json:"stage"`
	Roles           []string                `json:"roles,omitempty"`
	Skills          []string                `json:"skills,omitempty"`
	HumanGate       bool                    `json:"human_gate,omitempty"`
	EffortBudget    string                  `json:"effort_budget,omitempty"`
	Prerequisites   *BindingPrereqs         `json:"prerequisites,omitempty"`
	SubAgentProfile *BindingSubAgentProfile `json:"sub_agent_profile,omitempty"`
}

// HydrateBinding extracts key stage-binding fields from b and returns a
// BindingPayload for inclusion in task context responses. When b is nil a
// payload carrying only the stage name is returned; callers will not panic on
// a nil binding. Optional fields absent in b (nil pointers, empty strings,
// false booleans) are omitted from the returned payload rather than zeroed.
func HydrateBinding(stage string, b *binding.StageBinding) BindingPayload {
	p := BindingPayload{Stage: stage}
	if b == nil {
		return p
	}

	p.Roles = b.Roles
	p.Skills = b.Skills
	p.HumanGate = b.HumanGate
	p.EffortBudget = b.EffortBudget

	if b.Prerequisites != nil {
		bp := &BindingPrereqs{}
		for _, d := range b.Prerequisites.Documents {
			bp.Documents = append(bp.Documents, BindingDocPrereq{
				Type:   d.Type,
				Status: d.Status,
			})
		}
		if b.Prerequisites.Tasks != nil {
			bp.Tasks = &BindingTaskPrereq{
				MinCount:    b.Prerequisites.Tasks.MinCount,
				AllTerminal: b.Prerequisites.Tasks.AllTerminal,
			}
		}
		// Only attach prerequisites when at least one field was populated.
		if len(bp.Documents) > 0 || bp.Tasks != nil {
			p.Prerequisites = bp
		}
	}

	if b.SubAgents != nil {
		p.SubAgentProfile = &BindingSubAgentProfile{
			Roles:     b.SubAgents.Roles,
			Skills:    b.SubAgents.Skills,
			Topology:  b.SubAgents.Topology,
			MaxAgents: b.SubAgents.MaxAgents,
		}
	}

	return p
}
