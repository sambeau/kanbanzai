package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/card"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── Handoff constraint card fixture helpers ───────────────────────────────────

// writeHCRoleFile writes a valid role YAML for the given ID to dir.
func writeHCRoleFile(t *testing.T, dir, id, identity string) {
	t.Helper()
	content := "id: " + id + "\n" +
		"identity: \"" + identity + "\"\n" +
		"vocabulary:\n" +
		"  - \"test term\"\n" +
		"anti_patterns:\n" +
		"  - name: \"Test Anti-Pattern\"\n" +
		"    detect: \"test detect\"\n" +
		"    because: \"test because\"\n" +
		"    resolve: \"test resolve\"\n"
	if err := os.WriteFile(filepath.Join(dir, id+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write role %s: %v", id, err)
	}
}

// writeHCBindingFile writes a minimal stage-bindings YAML to path.
func writeHCBindingFile(t *testing.T, path, stage, role, skill string) {
	t.Helper()
	content := "stage_bindings:\n  " + stage + ":\n    roles: [" + role + "]\n    skills: [" + skill + "]\n    human_gate: false\n    effort_budget: \"test effort\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write binding %s: %v", path, err)
	}
}

// writeHCRegistryFile writes a minimal constraints YAML to path.
func writeHCRegistryFile(t *testing.T, path, roleID, stage string) {
	t.Helper()
	content := "constraints:\n  - id: C-TEST-001\n    rule: \"Test constraint rule.\"\n    applies_to:\n      roles: [" + roleID + "]\n      stages: [" + stage + "]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write constraints %s: %v", path, err)
	}
}

// setupHandoffConstraintTest creates a fully wired handoff tool with role store,
// binding file, constraint registry, and pipeline for the given stage and role ID.
func setupHandoffConstraintTest(t *testing.T, stage, roleID, identity string) (
	entitySvc *service.EntityService,
	bf *binding.BindingFile,
	roleStore *kbzctx.RoleStore,
	reg *card.ConstraintRegistry,
) {
	t.Helper()

	entityRoot := t.TempDir()
	roleDir := t.TempDir()
	bindingPath := filepath.Join(t.TempDir(), "stage-bindings.yaml")
	constraintPath := filepath.Join(t.TempDir(), "constraints.yaml")

	writeHCRoleFile(t, roleDir, roleID, identity)
	writeHCBindingFile(t, bindingPath, stage, roleID, "test-skill")
	writeHCRegistryFile(t, constraintPath, roleID, stage)

	var errs []error
	bf, errs = binding.LoadBindingFile(bindingPath)
	if len(errs) > 0 {
		t.Fatalf("LoadBindingFile: %v", errs)
	}

	var err error
	reg, err = card.LoadConstraintRegistry(constraintPath)
	if err != nil {
		t.Fatalf("LoadConstraintRegistry: %v", err)
	}

	entitySvc = service.NewEntityService(entityRoot)
	roleStore = kbzctx.NewRoleStore(roleDir, roleDir)
	return
}

// callHandoffWithCard invokes the handoff tool with all constraint card dependencies.
func callHandoffWithCard(
	t *testing.T,
	entitySvc *service.EntityService,
	pipeline *kbzctx.Pipeline,
	bf *binding.BindingFile,
	roleStore *kbzctx.RoleStore,
	reg *card.ConstraintRegistry,
	args map[string]any,
) map[string]any {
	t.Helper()
	tool := handoffTool(entitySvc, pipeline, bf, roleStore, reg, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handoff handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse handoff result: %v\nraw: %s", err, text)
	}
	return parsed
}

// setupHandoffCardScenario creates a plan → feature → task chain with the feature
// advanced to "developing" and the task to "active" so handoff can run.
func setupHandoffCardScenario(t *testing.T, entitySvc *service.EntityService, suffix string) (taskID, taskSlug string) {
	t.Helper()
	planID := createHandoffPlan(t, entitySvc, "hocard-"+suffix+"-plan")
	featID := createHandoffFeature(t, entitySvc, planID, "hocard-"+suffix+"-feat")
	advanceHandoffFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug = createHandoffTask(t, entitySvc, featID, "hocard-"+suffix+"-task")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")
	return
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestHandoffConstraintCard_Present asserts the prompt begins with the constraint
// card when a role is resolved and stage binding is available (AC-004, REQ-005).
func TestHandoffConstraintCard_Present(t *testing.T) {
	entitySvc, bf, roleStore, reg := setupHandoffConstraintTest(t, "developing", "implementer-go", "Senior Go engineer")
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "present")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	prompt, ok := result["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		t.Fatalf("prompt missing or empty")
	}

	if !strings.HasPrefix(prompt, "---\n") {
		t.Errorf("prompt does not start with constraint card (expected '---\\n'); got prefix: %q",
			prompt[:min(80, len(prompt))])
	}
}

// TestHandoffConstraintCard_ContainsRoleAndStage asserts the card names the role
// identity and stage (AC-002, REQ-003).
func TestHandoffConstraintCard_ContainsRoleAndStage(t *testing.T) {
	entitySvc, bf, roleStore, reg := setupHandoffConstraintTest(t, "developing", "implementer-go", "Senior Go engineer")
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "content")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	prompt, _ := result["prompt"].(string)

	if !strings.Contains(prompt, "Senior Go engineer") {
		t.Errorf("prompt missing role identity; prompt:\n%s", prompt[:min(500, len(prompt))])
	}
	if !strings.Contains(prompt, "developing") {
		t.Errorf("prompt missing stage name; prompt:\n%s", prompt[:min(500, len(prompt))])
	}
}

// TestHandoffStageBinding_Fields asserts stage_binding contains expected fields
// from the fixture binding (AC-005, REQ-006).
func TestHandoffStageBinding_Fields(t *testing.T) {
	entitySvc, bf, roleStore, reg := setupHandoffConstraintTest(t, "developing", "implementer-go", "Senior Go engineer")
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "sb")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	sb, ok := result["stage_binding"].(map[string]any)
	if !ok {
		t.Fatalf("stage_binding missing or wrong type; keys: %v", mapKeys(result))
	}
	if sb["stage"] != "developing" {
		t.Errorf("stage_binding.stage = %v, want %q", sb["stage"], "developing")
	}
	if sb["roles"] == nil {
		t.Error("stage_binding.roles missing")
	}
	if sb["skills"] == nil {
		t.Error("stage_binding.skills missing")
	}
}

// TestHandoffExistingFields_Preserved asserts all prior handoff response fields
// (task_id, display_id, entity_ref, prompt, context_metadata) remain present
// after constraint card injection (AC-009, REQ-010).
func TestHandoffExistingFields_Preserved(t *testing.T) {
	entitySvc, bf, roleStore, reg := setupHandoffConstraintTest(t, "developing", "implementer-go", "Senior Go engineer")
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "preserve")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	if result["task_id"] == nil {
		t.Error("task_id field missing from response")
	}
	if result["display_id"] == nil {
		t.Error("display_id field missing from response")
	}
	if result["entity_ref"] == nil {
		t.Error("entity_ref field missing from response")
	}
	prompt, ok := result["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		t.Error("prompt field missing or empty")
	}
	cm, ok := result["context_metadata"].(map[string]any)
	if !ok {
		t.Fatal("context_metadata missing or wrong type")
	}
	if cm["assembly_path"] != "pipeline-3.0" {
		t.Errorf("context_metadata.assembly_path = %v, want pipeline-3.0", cm["assembly_path"])
	}
	if cm["sections"] == nil {
		t.Error("context_metadata.sections missing")
	}
	if result["stage_binding"] == nil {
		t.Error("stage_binding field missing from response")
	}
}

// TestHandoffConstraintCard_PrependedToPrompt asserts the constraint card is
// prepended to the prompt string (AC-004: card before task-specific instructions).
func TestHandoffConstraintCard_PrependedToPrompt(t *testing.T) {
	entitySvc, bf, roleStore, reg := setupHandoffConstraintTest(t, "developing", "implementer-go", "Senior Go engineer")
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "prepend")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	prompt, _ := result["prompt"].(string)

	// The card starts with "---\n" and ends with "---\n" before the task context.
	// The pipeline content starts with "## Task:".
	if !strings.Contains(prompt, "---\n**Role:**") {
		t.Error("prompt missing constraint card role line")
	}
	if !strings.Contains(prompt, "## Task:") {
		t.Error("prompt missing '## Task:' heading from pipeline")
	}
	cardEnd := strings.Index(prompt, "---\n## Task:")
	if cardEnd == -1 {
		// The card closing "---\n" might be followed by the pipeline content
		// with different spacing. Check that "## Task:" comes after the card.
		idxCardStart := strings.Index(prompt, "---\n")
		idxTask := strings.Index(prompt, "## Task:")
		if idxTask < idxCardStart {
			t.Errorf("'## Task:' appears before constraint card in prompt")
		}
	} else {
		// Card end found directly before "## Task:"
		_ = cardEnd // card ends right before pipeline content
	}
}

// TestHandoffConstraintCard_RoleNotFound_CardAbsent asserts that when the role
// specified does not exist in the role store, the prompt does not contain the
// card preamble but handoff still succeeds (AC-009).
func TestHandoffConstraintCard_RoleNotFound_CardAbsent(t *testing.T) {
	entityRoot := t.TempDir()
	roleDir := t.TempDir()
	bindingPath := filepath.Join(t.TempDir(), "stage-bindings.yaml")
	constraintPath := filepath.Join(t.TempDir(), "constraints.yaml")

	writeHCBindingFile(t, bindingPath, "developing", "missing-role", "test-skill")
	writeHCRegistryFile(t, constraintPath, "missing-role", "developing")

	bf, errs := binding.LoadBindingFile(bindingPath)
	if len(errs) > 0 {
		t.Fatalf("LoadBindingFile: %v", errs)
	}
	reg, err := card.LoadConstraintRegistry(constraintPath)
	if err != nil {
		t.Fatalf("LoadConstraintRegistry: %v", err)
	}

	entitySvc := service.NewEntityService(entityRoot)
	roleStore := kbzctx.NewRoleStore(roleDir, roleDir)

	taskID, _ := setupHandoffCardScenario(t, entitySvc, "missing-role")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
		"role":    "missing-role",
	})

	prompt, _ := result["prompt"].(string)
	if strings.HasPrefix(prompt, "---\n") {
		t.Error("constraint card preamble present in prompt despite role-not-found")
	}
	if prompt == "" {
		t.Error("prompt empty — handoff should have succeeded")
	}
	if result["task_id"] == nil {
		t.Error("task_id missing from response — handoff should have succeeded")
	}
}

// TestHandoffConstraintCard_NoRole_NoCard asserts that when no role is provided
// and the binding provides no fallback role, prompt does not contain card.
func TestHandoffConstraintCard_NoRole_NoCard(t *testing.T) {
	entityRoot := t.TempDir()
	roleDir := t.TempDir()
	bindingPath := filepath.Join(t.TempDir(), "stage-bindings.yaml")
	constraintPath := filepath.Join(t.TempDir(), "constraints.yaml")

	noRoleBinding := "stage_bindings:\n  developing:\n    roles: []\n    skills: [test-skill]\n    human_gate: false\n"
	if err := os.WriteFile(bindingPath, []byte(noRoleBinding), 0o644); err != nil {
		t.Fatalf("write binding: %v", err)
	}
	writeHCRegistryFile(t, constraintPath, "some-role", "developing")

	bf, errs := binding.LoadBindingFile(bindingPath)
	if len(errs) > 0 {
		t.Fatalf("LoadBindingFile: %v", errs)
	}
	reg, err := card.LoadConstraintRegistry(constraintPath)
	if err != nil {
		t.Fatalf("LoadConstraintRegistry: %v", err)
	}

	entitySvc := service.NewEntityService(entityRoot)
	roleStore := kbzctx.NewRoleStore(roleDir, roleDir)

	taskID, _ := setupHandoffCardScenario(t, entitySvc, "no-role")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), bf, roleStore, reg, map[string]any{
		"task_id": taskID,
	})

	prompt, _ := result["prompt"].(string)
	if strings.HasPrefix(prompt, "---\n") {
		t.Error("constraint card preamble present in prompt despite no role")
	}
	if result["stage_binding"] == nil {
		t.Error("stage_binding missing when role is absent")
	}
	if prompt == "" {
		t.Error("prompt empty — handoff should have succeeded")
	}
}

// TestHandoffConstraintCard_NilDependencies_NoCard asserts that when bf,
// roleStore, and constraintReg are all nil, handoff still works and no
// card appears in the prompt (backward compatibility).
func TestHandoffConstraintCard_NilDependencies_NoCard(t *testing.T) {
	entitySvc := setupHandoffTest(t)
	taskID, _ := setupHandoffCardScenario(t, entitySvc, "nil-deps")

	result := callHandoffWithCard(t, entitySvc, testHandoffPipeline(), nil, nil, nil, map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	prompt, _ := result["prompt"].(string)
	if strings.HasPrefix(prompt, "---\n") {
		t.Error("constraint card preamble present when all card dependencies are nil")
	}
	if prompt == "" {
		t.Error("prompt empty — handoff should have succeeded")
	}
	if result["stage_binding"] == nil {
		t.Error("stage_binding missing even with nil bf")
	}
}
