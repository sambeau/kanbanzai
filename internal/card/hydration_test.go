package card

import (
	"encoding/json"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
)

func TestHydrateBinding_NilBinding(t *testing.T) {
	p := HydrateBinding("developing", nil)

	if p.Stage != "developing" {
		t.Fatalf("got Stage=%q, want %q", p.Stage, "developing")
	}
	if len(p.Roles) != 0 {
		t.Errorf("expected no roles for nil binding, got %v", p.Roles)
	}
	if len(p.Skills) != 0 {
		t.Errorf("expected no skills for nil binding, got %v", p.Skills)
	}
	if p.HumanGate {
		t.Errorf("expected HumanGate=false for nil binding")
	}
	if p.EffortBudget != "" {
		t.Errorf("expected empty EffortBudget for nil binding, got %q", p.EffortBudget)
	}
	if p.Prerequisites != nil {
		t.Errorf("expected nil Prerequisites for nil binding")
	}
	if p.SubAgentProfile != nil {
		t.Errorf("expected nil SubAgentProfile for nil binding")
	}

	// Verify the JSON output is minimal — only stage is present.
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, absent := range []string{"roles", "skills", "human_gate", "effort_budget", "prerequisites", "sub_agent_profile"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON must not contain key %q for nil binding, but it does: %s", absent, data)
		}
	}
}

func TestHydrateBinding_MinimalBinding(t *testing.T) {
	b := &binding.StageBinding{
		Roles:  []string{"implementer-go"},
		Skills: []string{"implement-task"},
	}
	p := HydrateBinding("developing", b)

	if p.Stage != "developing" {
		t.Errorf("got Stage=%q, want %q", p.Stage, "developing")
	}
	if len(p.Roles) != 1 || p.Roles[0] != "implementer-go" {
		t.Errorf("got Roles=%v, want [implementer-go]", p.Roles)
	}
	if len(p.Skills) != 1 || p.Skills[0] != "implement-task" {
		t.Errorf("got Skills=%v, want [implement-task]", p.Skills)
	}
	if p.HumanGate {
		t.Errorf("expected HumanGate=false for minimal binding")
	}
	if p.EffortBudget != "" {
		t.Errorf("expected empty EffortBudget for minimal binding, got %q", p.EffortBudget)
	}
	if p.Prerequisites != nil {
		t.Errorf("expected nil Prerequisites for minimal binding, got %+v", p.Prerequisites)
	}
	if p.SubAgentProfile != nil {
		t.Errorf("expected nil SubAgentProfile for minimal binding, got %+v", p.SubAgentProfile)
	}

	// Verify JSON omits optional absent fields.
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, absent := range []string{"effort_budget", "prerequisites", "sub_agent_profile"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON must not contain key %q for minimal binding: %s", absent, data)
		}
	}
}

func TestHydrateBinding_FullBinding(t *testing.T) {
	allTerminal := true
	maxAgents := 3
	b := &binding.StageBinding{
		Roles:        []string{"orchestrator"},
		Skills:       []string{"orchestrate-development"},
		HumanGate:    true,
		EffortBudget: "1 sprint",
		Prerequisites: &binding.Prerequisites{
			Documents: []binding.DocumentPrereq{
				{Type: "spec", Status: "approved"},
				{Type: "dev-plan", Status: "approved"},
			},
			Tasks: &binding.TaskPrereq{
				AllTerminal: &allTerminal,
			},
		},
		SubAgents: &binding.SubAgents{
			Roles:     []string{"implementer-go"},
			Skills:    []string{"implement-task"},
			Topology:  "parallel",
			MaxAgents: &maxAgents,
		},
	}

	p := HydrateBinding("developing", b)

	if p.Stage != "developing" {
		t.Errorf("got Stage=%q, want %q", p.Stage, "developing")
	}
	if len(p.Roles) != 1 || p.Roles[0] != "orchestrator" {
		t.Errorf("got Roles=%v, want [orchestrator]", p.Roles)
	}
	if len(p.Skills) != 1 || p.Skills[0] != "orchestrate-development" {
		t.Errorf("got Skills=%v, want [orchestrate-development]", p.Skills)
	}
	if !p.HumanGate {
		t.Errorf("expected HumanGate=true")
	}
	if p.EffortBudget != "1 sprint" {
		t.Errorf("got EffortBudget=%q, want %q", p.EffortBudget, "1 sprint")
	}

	// Prerequisites.
	if p.Prerequisites == nil {
		t.Fatal("expected Prerequisites to be set")
	}
	if len(p.Prerequisites.Documents) != 2 {
		t.Errorf("got %d documents, want 2", len(p.Prerequisites.Documents))
	} else {
		if p.Prerequisites.Documents[0].Type != "spec" || p.Prerequisites.Documents[0].Status != "approved" {
			t.Errorf("unexpected first document prereq: %+v", p.Prerequisites.Documents[0])
		}
		if p.Prerequisites.Documents[1].Type != "dev-plan" || p.Prerequisites.Documents[1].Status != "approved" {
			t.Errorf("unexpected second document prereq: %+v", p.Prerequisites.Documents[1])
		}
	}
	if p.Prerequisites.Tasks == nil {
		t.Fatal("expected Tasks prereq to be set")
	}
	if p.Prerequisites.Tasks.AllTerminal == nil || !*p.Prerequisites.Tasks.AllTerminal {
		t.Errorf("expected AllTerminal=true")
	}
	if p.Prerequisites.Tasks.MinCount != nil {
		t.Errorf("expected nil MinCount, got %v", p.Prerequisites.Tasks.MinCount)
	}

	// Sub-agent profile.
	if p.SubAgentProfile == nil {
		t.Fatal("expected SubAgentProfile to be set")
	}
	if len(p.SubAgentProfile.Roles) != 1 || p.SubAgentProfile.Roles[0] != "implementer-go" {
		t.Errorf("got SubAgentProfile.Roles=%v, want [implementer-go]", p.SubAgentProfile.Roles)
	}
	if len(p.SubAgentProfile.Skills) != 1 || p.SubAgentProfile.Skills[0] != "implement-task" {
		t.Errorf("got SubAgentProfile.Skills=%v, want [implement-task]", p.SubAgentProfile.Skills)
	}
	if p.SubAgentProfile.Topology != "parallel" {
		t.Errorf("got SubAgentProfile.Topology=%q, want %q", p.SubAgentProfile.Topology, "parallel")
	}
	if p.SubAgentProfile.MaxAgents == nil || *p.SubAgentProfile.MaxAgents != 3 {
		t.Errorf("expected MaxAgents=3, got %v", p.SubAgentProfile.MaxAgents)
	}
}

func TestHydrateBinding_PrereqsDocumentsOnly(t *testing.T) {
	b := &binding.StageBinding{
		Roles:  []string{"spec-author"},
		Skills: []string{"write-spec"},
		Prerequisites: &binding.Prerequisites{
			Documents: []binding.DocumentPrereq{
				{Type: "design", Status: "approved"},
			},
		},
	}
	p := HydrateBinding("specifying", b)

	if p.Prerequisites == nil {
		t.Fatal("expected Prerequisites to be set")
	}
	if len(p.Prerequisites.Documents) != 1 {
		t.Fatalf("expected 1 document prereq, got %d", len(p.Prerequisites.Documents))
	}
	if p.Prerequisites.Documents[0].Type != "design" {
		t.Errorf("got Type=%q, want %q", p.Prerequisites.Documents[0].Type, "design")
	}
	if p.Prerequisites.Tasks != nil {
		t.Errorf("expected nil Tasks, got %+v", p.Prerequisites.Tasks)
	}
}

func TestHydrateBinding_EmptyPrereqsOmitted(t *testing.T) {
	// Prerequisites present but both documents and tasks are empty/nil —
	// the payload must omit prerequisites entirely (no spurious empty objects).
	b := &binding.StageBinding{
		Roles:         []string{"architect"},
		Skills:        []string{"write-design"},
		Prerequisites: &binding.Prerequisites{},
	}
	p := HydrateBinding("designing", b)

	if p.Prerequisites != nil {
		t.Errorf("expected nil Prerequisites when source has no documents or tasks, got %+v", p.Prerequisites)
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := m["prerequisites"]; ok {
		t.Errorf("JSON must not contain key \"prerequisites\" for empty prereqs: %s", data)
	}
}
