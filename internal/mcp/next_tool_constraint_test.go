package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/card"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── constraint card test helpers ─────────────────────────────────────────────

// constraintCardSetup creates a role store, binding file, and constraint registry
// for testing the constraint card and stage binding injection.
func constraintCardSetup(t *testing.T) (
	*service.EntityService,
	*service.DispatchService,
	*kbzctx.RoleStore,
	*binding.BindingFile,
	*card.ConstraintRegistry,
) {
	t.Helper()

	entitySvc, dispatchSvc := setupNextTest(t)

	// Role store with an implementer-go role (the role used in nextClaimMode tests).
	roleDir := t.TempDir()
	roleYAML := `id: implementer-go
identity: "Go Implementer"
vocabulary:
  - implement
  - code
  - test
anti_patterns:
  - name: skip-tests
    detect: "No test files in changes"
    because: "Untested code is fragile"
    resolve: "Add test coverage"
`
	if err := os.WriteFile(filepath.Join(roleDir, "implementer-go.yaml"), []byte(roleYAML), 0o644); err != nil {
		t.Fatalf("write role: %v", err)
	}
	roleStore := kbzctx.NewRoleStore(roleDir, "")

	// Binding file with a developing stage binding.
	bf := &binding.BindingFile{
		StageBindings: map[string]*binding.StageBinding{
			"developing": {
				Description:   "Implementing tasks",
				Orchestration: "orchestrator-workers",
				Roles:         []string{"implementer-go"},
				Skills:        []string{"implement-task"},
				HumanGate:     false,
				EffortBudget:  "10-50 tool calls",
				SubAgents: &binding.SubAgents{
					Roles:     []string{"implementer-go"},
					Skills:    []string{"implement-task"},
					Topology:  "parallel",
					MaxAgents: intPtr(4),
				},
			},
		},
	}

	// Constraint registry with entries for the developing stage.
	yamlContent := `constraints:
  - id: C-REQ-001
    rule: "Always run tests before committing"
    applies_to:
      roles: [implementer-go]
      stages: [developing]
  - id: C-REQ-002
    rule: "Follow commit message format"
    applies_to:
      roles: [implementer-go]
      stages: [developing, reviewing]
  - id: C-REQ-003
    rule: "Do not skip workflow stages"
    applies_to:
      roles: [implementer-go, architect]
      stages: [designing, specifying, developing]
  - id: C-SPEC-001
    rule: "Spec-only constraint"
    applies_to:
      roles: [spec-author]
      stages: [specifying]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "constraints.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write constraints: %v", err)
	}
	constraintReg, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("load constraint registry: %v", err)
	}

	return entitySvc, dispatchSvc, roleStore, bf, constraintReg
}

// callNextClaimModeJSON invokes nextClaimMode directly and parses the result.
func callNextClaimModeJSON(
	t *testing.T,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	roleStore *kbzctx.RoleStore,
	bf *binding.BindingFile,
	constraintReg *card.ConstraintRegistry,
	id, role, featureStage string,
) map[string]any {
	t.Helper()
	result, err := nextClaimMode(
		context.Background(),
		id, role,
		entitySvc, dispatchSvc,
		nil, nil, nil, nil, // profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc
		nil,                // mergedToolHints
		roleStore, nil,     // roleStore, worktreeStore
		bf, constraintReg,
	)
	if err != nil {
		t.Fatalf("nextClaimMode error: %v", err)
	}
	// Marshal → unmarshal to get a plain map (result is map[string]any but
	// may contain typed values that json.Marshal handles correctly).
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return parsed
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestNextClaimMode_ConstraintCardAndStageBinding_Present(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	// Create a plan, feature at developing stage, and a ready task.
	planID := createNextTestPlan(t, entitySvc, "cc-test")
	featID := createNextTestFeature(t, entitySvc, planID, "cc-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "cc-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Claim with a role that matches the binding.
	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "implementer-go", "developing")

	// constraint_card must be present and non-empty.
	cardStr, ok := result["constraint_card"].(string)
	if !ok {
		t.Fatal("expected constraint_card string field in result")
	}
	if cardStr == "" {
		t.Fatal("constraint_card must not be empty")
	}
	if !strings.Contains(cardStr, "**Role:**") {
		t.Errorf("constraint_card missing role header: %s", cardStr)
	}
	if !strings.Contains(cardStr, "implementer-go") {
		t.Errorf("constraint_card missing role ID: %s", cardStr)
	}
	if !strings.Contains(cardStr, "**Stage:** developing") {
		t.Errorf("constraint_card missing stage: %s", cardStr)
	}
	// Must contain at least the C-REQ-001 constraint (matched on implementer-go + developing).
	if !strings.Contains(cardStr, "Always run tests before committing") {
		t.Errorf("constraint_card missing expected constraint C-REQ-001: %s", cardStr)
	}
	// C-SPEC-001 should NOT appear (spec-author + specifying, not our role/stage).
	if strings.Contains(cardStr, "Spec-only constraint") {
		t.Error("constraint_card should not contain spec-only constraint C-SPEC-001")
	}

	// stage_binding must be present with correct structure.
	sb, ok := result["stage_binding"].(map[string]any)
	if !ok {
		t.Fatal("expected stage_binding object in result")
	}
	if sb["stage"] != "developing" {
		t.Errorf("stage_binding.stage = %q, want %q", sb["stage"], "developing")
	}
	roles, _ := sb["roles"].([]any)
	if len(roles) != 1 || roles[0] != "implementer-go" {
		t.Errorf("stage_binding.roles = %v, want [implementer-go]", roles)
	}
	skills, _ := sb["skills"].([]any)
	if len(skills) != 1 || skills[0] != "implement-task" {
		t.Errorf("stage_binding.skills = %v, want [implement-task]", skills)
	}
	if sb["effort_budget"] != "10-50 tool calls" {
		t.Errorf("stage_binding.effort_budget = %v", sb["effort_budget"])
	}
	subAgents, ok := sb["sub_agent_profile"].(map[string]any)
	if !ok {
		t.Fatal("stage_binding.sub_agent_profile missing")
	}
	if subAgents["topology"] != "parallel" {
		t.Errorf("sub_agent_profile.topology = %v", subAgents["topology"])
	}
}

func TestNextClaimMode_ConstraintCard_BeforeContext(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	planID := createNextTestPlan(t, entitySvc, "order-test")
	featID := createNextTestFeature(t, entitySvc, planID, "order-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "order-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result, err := nextClaimMode(
		context.Background(),
		taskID, "implementer-go",
		entitySvc, dispatchSvc,
		nil, nil, nil, nil,
		nil, roleStore, nil,
		bf, constraintReg,
	)
	if err != nil {
		t.Fatalf("nextClaimMode: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ccIdx := strings.Index(string(raw), `"constraint_card"`)
	ctxIdx := strings.Index(string(raw), `"context"`)
	taskIdx := strings.Index(string(raw), `"task"`)
	sbIdx := strings.Index(string(raw), `"stage_binding"`)

	if ccIdx == -1 {
		t.Fatal("constraint_card not found in JSON output")
	}
	if ctxIdx == -1 {
		t.Fatal("context not found in JSON output")
	}
	// constraint_card must appear before context.
	if ccIdx > ctxIdx {
		t.Errorf("constraint_card (pos %d) must appear before context (pos %d) in JSON", ccIdx, ctxIdx)
	}
	// All fields must be top-level.
	if taskIdx == -1 {
		t.Error("task not found in JSON output")
	}
	if sbIdx == -1 {
		t.Error("stage_binding not found in JSON output")
	}
}

func TestNextClaimMode_ConstraintCard_NoRole_BindingFallback(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	planID := createNextTestPlan(t, entitySvc, "norole-test")
	featID := createNextTestFeature(t, entitySvc, planID, "norole-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "norole-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Empty role — implementation falls back to binding's first role.
	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "", "developing")

	// constraint_card should be present (fallback to binding's first role: implementer-go).
	cardStr, ok := result["constraint_card"].(string)
	if !ok {
		t.Fatal("constraint_card should be present via binding role fallback")
	}
	if !strings.Contains(cardStr, "implementer-go") {
		t.Errorf("constraint_card should use fallback role 'implementer-go': %s", cardStr)
	}

	// stage_binding should always be present (independent of role).
	if _, ok := result["stage_binding"]; !ok {
		t.Error("stage_binding should be present even without explicit role")
	}
}

func TestNextClaimMode_ConstraintCard_ExistingFieldsUnchanged(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	planID := createNextTestPlan(t, entitySvc, "fields-test")
	featID := createNextTestFeature(t, entitySvc, planID, "fields-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "fields-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "implementer-go", "developing")

	// Verify existing fields remain intact.
	task, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatal("expected task field in result")
	}
	if task["id"] != taskID {
		t.Errorf("task.id = %v, want %v", task["id"], taskID)
	}

	contextMap, ok := result["context"].(map[string]any)
	if !ok {
		t.Fatal("expected context field in result")
	}
	if _, ok := contextMap["spec_sections"]; !ok {
		t.Error("context missing spec_sections")
	}

	// reclaimed should NOT be present on first claim.
	if _, ok := result["reclaimed"]; ok {
		t.Error("reclaimed should not be present on first claim")
	}
}

func TestNextClaimMode_ConstraintCard_ReclaimedPreserved(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	planID := createNextTestPlan(t, entitySvc, "reclaim-test")
	featID := createNextTestFeature(t, entitySvc, planID, "reclaim-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "reclaim-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// First claim.
	_ = callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "implementer-go", "developing")

	// Second claim (reclaim) — now task is active.
	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "implementer-go", "developing")

	// reclaimed must be true.
	if result["reclaimed"] != true {
		t.Errorf("expected reclaimed=true on second claim, got %v", result["reclaimed"])
	}

	// constraint_card and stage_binding must still be present on reclaim.
	if _, ok := result["constraint_card"]; !ok {
		t.Error("constraint_card missing on reclaim")
	}
	if _, ok := result["stage_binding"]; !ok {
		t.Error("stage_binding missing on reclaim")
	}
}

func TestNextClaimMode_ConstraintCard_RendererError_ManyConstraints(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc := setupNextTest(t)

	// Role store with a valid role.
	roleDir := t.TempDir()
	roleYAML := `id: implementer-go
identity: "Go Implementer"
vocabulary:
  - test
anti_patterns:
  - name: test
    detect: "d"
    because: "b"
    resolve: "r"
`
	if err := os.WriteFile(filepath.Join(roleDir, "implementer-go.yaml"), []byte(roleYAML), 0o644); err != nil {
		t.Fatalf("write role: %v", err)
	}
	roleStore := kbzctx.NewRoleStore(roleDir, "")

	bf := &binding.BindingFile{
		StageBindings: map[string]*binding.StageBinding{
			"developing": {
				Description:   "Test",
				Orchestration: "single-agent",
				Roles:         []string{"implementer-go"},
				Skills:        []string{"test"},
			},
		},
	}

	// Create a constraint registry with 30 entries — enough to exceed the
	// 25 non-empty-line limit of the renderer (REQ-NF-001, REQ-NF-002).
	var yamlLines []string
	yamlLines = append(yamlLines, "constraints:")
	for i := 1; i <= 30; i++ {
		yamlLines = append(yamlLines, fmt.Sprintf(`  - id: C-%03d
    rule: "Constraint number %d: this is a very long and detailed rule that must be followed."
    applies_to:
      roles: [implementer-go]
      stages: [developing]
`, i, i))
	}
	yamlContent := strings.Join(yamlLines, "\n")

	dir := t.TempDir()
	path := filepath.Join(dir, "constraints.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write constraints: %v", err)
	}
	constraintReg, err := card.LoadConstraintRegistry(path)
	if err != nil {
		t.Fatalf("load constraint registry: %v", err)
	}

	planID := createNextTestPlan(t, entitySvc, "overflow-test")
	featID := createNextTestFeature(t, entitySvc, planID, "overflow-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "overflow-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	_, err = nextClaimMode(
		context.Background(),
		taskID, "implementer-go",
		entitySvc, dispatchSvc,
		nil, nil, nil, nil,
		nil, roleStore, nil,
		bf, constraintReg,
	)
	if err == nil {
		t.Fatal("expected error when constraint card exceeds line/budget limit")
	}
	if !strings.Contains(err.Error(), "constraint card") {
		t.Errorf("error should mention constraint card, got: %v", err)
	}
}

func TestNextClaimMode_StageBinding_NoBindingFile(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc := setupNextTest(t)

	// Role store with a valid role.
	roleDir := t.TempDir()
	roleYAML := `id: implementer-go
identity: "Go Implementer"
vocabulary:
  - test
anti_patterns:
  - name: test
    detect: "d"
    because: "b"
    resolve: "r"
`
	if err := os.WriteFile(filepath.Join(roleDir, "implementer-go.yaml"), []byte(roleYAML), 0o644); err != nil {
		t.Fatalf("write role: %v", err)
	}
	roleStore := kbzctx.NewRoleStore(roleDir, "")

	planID := createNextTestPlan(t, entitySvc, "nobf-test")
	featID := createNextTestFeature(t, entitySvc, planID, "nobf-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "nobf-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, nil, nil, taskID, "implementer-go", "developing")

	// stage_binding should be present with only the stage name.
	sb, ok := result["stage_binding"].(map[string]any)
	if !ok {
		t.Fatal("stage_binding should be present even with nil binding file")
	}
	if sb["stage"] != "developing" {
		t.Errorf("stage_binding.stage = %q", sb["stage"])
	}
	// No roles/skills when no binding.
	if _, exists := sb["roles"]; exists {
		t.Error("stage_binding.roles should not be present with nil binding file")
	}

	// constraint_card should still render (with UNKNOWN STAGE fallback).
	cardStr, ok := result["constraint_card"].(string)
	if !ok {
		t.Fatal("constraint_card missing with nil binding file")
	}
	if !strings.Contains(cardStr, "UNKNOWN STAGE") {
		t.Errorf("constraint_card should contain UNKNOWN STAGE fallback: %s", cardStr)
	}
}

func TestNextClaimMode_StageBinding_ConstraintsOnlyForRoleAndStage(t *testing.T) {
	t.Parallel()

	entitySvc, dispatchSvc, roleStore, bf, constraintReg := constraintCardSetup(t)

	planID := createNextTestPlan(t, entitySvc, "filter-test")
	featID := createNextTestFeature(t, entitySvc, planID, "filter-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "filter-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextClaimModeJSON(t, entitySvc, dispatchSvc, roleStore, bf, constraintReg, taskID, "implementer-go", "developing")

	cardStr, _ := result["constraint_card"].(string)

	// C-REQ-001: implementer-go + developing → must appear.
	if !strings.Contains(cardStr, "Always run tests before committing") {
		t.Error("C-REQ-001 should appear (implementer-go + developing)")
	}
	// C-REQ-003: implementer-go + [designing, specifying, developing] → must appear (developing matches).
	if !strings.Contains(cardStr, "Do not skip workflow stages") {
		t.Error("C-REQ-003 should appear (implementer-go + developing)")
	}
	// C-SPEC-001: spec-author + specifying → must NOT appear.
	if strings.Contains(cardStr, "Spec-only constraint") {
		t.Error("C-SPEC-001 should NOT appear (spec-author + specifying)")
	}
}

func intPtr(n int) *int {
	return &n
}
