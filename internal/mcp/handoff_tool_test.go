package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// setupHandoffTest creates the services needed for handoff tool tests.
// Returns entitySvc, knowledgeSvc, profileStore, and the profileRoot directory
// (needed to write profile YAML files in individual tests).
func setupHandoffTest(t *testing.T) (*service.EntityService, *service.KnowledgeService, *kbzctx.ProfileStore, string) {
	t.Helper()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	profileRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	return entitySvc, knowledgeSvc, profileStore, profileRoot
}

// createHandoffScenario builds a plan → feature → task chain.
// Returns the task ID and slug.
func createHandoffScenario(t *testing.T, entitySvc *service.EntityService, suffix string) (taskID, taskSlug string) {
	t.Helper()
	planID := createHandoffPlan(t, entitySvc, "ho-plan-"+suffix)
	featID := createHandoffFeature(t, entitySvc, planID, "ho-feat-"+suffix)
	advanceHandoffFeatureTo(t, entitySvc, featID, "developing")
	return createHandoffTask(t, entitySvc, featID, "ho-task-"+suffix)
}

func createHandoffPlan(t *testing.T, entitySvc *service.EntityService, slug string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	id := "P1-" + slug
	record := storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id": id, "slug": slug, "title": "Test plan " + slug,
			"status": "proposed", "summary": "Test plan",
			"created": now, "created_by": "tester", "updated": now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createHandoffPlan(%s): %v", slug, err)
	}
	return id
}

func createHandoffFeature(t *testing.T, entitySvc *service.EntityService, planID, slug string) string {
	t.Helper()
	result, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Name: "test",
		Slug: slug, Parent: planID, Summary: "Feature " + slug, CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%s): %v", slug, err)
	}
	return result.ID
}

func createHandoffTask(t *testing.T, entitySvc *service.EntityService, featID, slug string) (string, string) {
	t.Helper()
	result, err := entitySvc.CreateTask(service.CreateTaskInput{
		Name: "test",
		ParentFeature: featID, Slug: slug, Summary: "Implement " + slug,
	})
	if err != nil {
		t.Fatalf("CreateTask(%s): %v", slug, err)
	}
	return result.ID, result.Slug
}

// advanceHandoffTaskTo transitions a task to the target status via the required chain.
func advanceHandoffTaskTo(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug, target string) {
	t.Helper()
	var chain []string
	switch target {
	case "ready":
		chain = []string{"ready"}
	case "active":
		chain = []string{"ready", "active"}
	case "needs-rework":
		chain = []string{"ready", "active", "needs-rework"}
	case "done":
		chain = []string{"ready", "active", "done"}
	case "not-planned":
		chain = []string{"not-planned"}
	case "duplicate":
		chain = []string{"ready", "duplicate"}
	default:
		t.Fatalf("advanceHandoffTaskTo: unsupported target %q", target)
	}
	for _, s := range chain {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "task", ID: taskID, Slug: taskSlug, Status: s,
		}); err != nil {
			t.Fatalf("advance %s to %s: %v", taskID, s, err)
		}
	}
}

// advanceHandoffFeatureTo transitions a feature to the target status via the required chain.
func advanceHandoffFeatureTo(t *testing.T, entitySvc *service.EntityService, featID, target string) {
	t.Helper()
	var chain []string
	switch target {
	case "designing":
		chain = []string{"designing"}
	case "specifying":
		chain = []string{"designing", "specifying"}
	case "dev-planning":
		chain = []string{"designing", "specifying", "dev-planning"}
	case "developing":
		chain = []string{"designing", "specifying", "dev-planning", "developing"}
	case "reviewing":
		chain = []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	default:
		t.Fatalf("advanceHandoffFeatureTo: unsupported target %q", target)
	}
	feat, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("advanceHandoffFeatureTo: get %s: %v", featID, err)
	}
	slug, _ := feat.State["slug"].(string)
	for _, s := range chain {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "feature", ID: featID, Slug: slug, Status: s,
		}); err != nil {
			t.Fatalf("advance feature %s to %s: %v", featID, s, err)
		}
	}
}

// writeHandoffProfile writes a YAML profile file directly into profileRoot.
func writeHandoffProfile(t *testing.T, profileRoot, id string, conventions []string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("id: " + id + "\n")
	sb.WriteString("description: Test profile for " + id + "\n")
	if len(conventions) > 0 {
		sb.WriteString("conventions:\n")
		for _, c := range conventions {
			sb.WriteString("  - " + c + "\n")
		}
	}
	if err := os.WriteFile(filepath.Join(profileRoot, id+".yaml"), []byte(sb.String()), 0600); err != nil {
		t.Fatalf("write profile %s: %v", id, err)
	}
}

// addHandoffKnowledge contributes a knowledge entry with the given scope and tier.
func addHandoffKnowledge(t *testing.T, svc *service.KnowledgeService, topic, content, scope string, tier int) {
	t.Helper()
	if _, _, err := svc.Contribute(service.ContributeInput{
		Topic:     topic,
		Content:   content,
		Scope:     scope,
		Tier:      tier,
		CreatedBy: "tester",
	}); err != nil {
		t.Fatalf("Contribute knowledge %q: %v", topic, err)
	}
}

// callHandoff invokes the handoff tool and returns the raw response text.
func callHandoff(
	t *testing.T,
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	args map[string]any,
) string {
	t.Helper()
	tool := handoffTool(entitySvc, profileStore, knowledgeSvc, nil, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handoff handler error: %v", err)
	}
	return extractText(t, result)
}

// callHandoffJSON invokes the handoff tool and parses the result as JSON.
func callHandoffJSON(
	t *testing.T,
	entitySvc *service.EntityService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	args map[string]any,
) map[string]any {
	t.Helper()
	text := callHandoff(t, entitySvc, profileStore, knowledgeSvc, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse handoff result: %v\nraw: %s", err, text)
	}
	return parsed
}

// setHandoffFilesPlanned writes files_planned onto an existing task record.
func setHandoffFilesPlanned(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string, files []string) {
	t.Helper()
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task %s: %v", taskID, err)
	}
	anyFiles := make([]any, len(files))
	for i, f := range files {
		anyFiles[i] = f
	}
	rec.Fields["files_planned"] = anyFiles
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("write files_planned for %s: %v", taskID, err)
	}
}

// ─── AC1: returns a complete prompt string ────────────────────────────────────

// TestHandoff_ReturnsPromptString verifies that handoff(task_id) returns a
// complete Markdown prompt string with the task_id field present.
func TestHandoff_ReturnsPromptString(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac1")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	// top-level task_id must echo the input.
	if resp["task_id"] != taskID {
		t.Errorf("task_id = %v, want %s", resp["task_id"], taskID)
	}

	prompt, ok := resp["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		t.Fatalf("expected non-empty prompt string, got: %v", resp["prompt"])
	}

	// Must start with conventions (high-attention zone per FR-012).
	if !strings.HasPrefix(prompt, "### Conventions") {
		t.Errorf("prompt does not open with '### Conventions' heading; got prefix: %q",
			prompt[:min(80, len(prompt))])
	}
}

// ─── AC2: prompt includes required sections ───────────────────────────────────

// TestHandoff_PromptContainsSummary verifies that the task summary appears in
// the prompt under a ### Summary section.
func TestHandoff_PromptContainsSummary(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac2sum")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "### Summary") {
		t.Errorf("prompt missing '### Summary' section:\n%s", prompt)
	}
	// Task summary was set to "Implement ho-task-ac2sum" by createHandoffTask.
	if !strings.Contains(prompt, "Implement ho-task-ac2sum") {
		t.Errorf("prompt does not contain task summary text:\n%s", prompt)
	}
}

// TestHandoff_PromptContainsKnowledge verifies that knowledge entries appear in
// the prompt under ### Known Constraints.
func TestHandoff_PromptContainsKnowledge(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac2ke")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	addHandoffKnowledge(t, knowledgeSvc,
		"auth-pattern",
		"Use http.Handler middleware wrapping, not per-route checks",
		"project", 2)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "### Known Constraints (from knowledge base)") {
		t.Errorf("prompt missing '### Known Constraints' section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Use http.Handler middleware wrapping") {
		t.Errorf("prompt missing knowledge entry content:\n%s", prompt)
	}
}

// TestHandoff_PromptContainsFiles verifies that files_planned entries appear in
// the prompt under ### Files.
func TestHandoff_PromptContainsFiles(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac2files")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")
	setHandoffFilesPlanned(t, entitySvc, taskID, taskSlug,
		[]string{"internal/auth/middleware.go", "internal/auth/middleware_test.go"})

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "### Files") {
		t.Errorf("prompt missing '### Files' section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "internal/auth/middleware.go") {
		t.Errorf("prompt missing first file path:\n%s", prompt)
	}
	if !strings.Contains(prompt, "internal/auth/middleware_test.go") {
		t.Errorf("prompt missing second file path:\n%s", prompt)
	}
}

// TestHandoff_PromptContainsConventions verifies that the ### Conventions section
// is always present and includes the commit format line.
func TestHandoff_PromptContainsConventions(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac2conv")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "### Conventions") {
		t.Errorf("prompt missing '### Conventions' section:\n%s", prompt)
	}
	// The commit format line always includes the task ID.
	if !strings.Contains(prompt, "Commit format: feat("+taskID+")") {
		t.Errorf("prompt missing commit format convention with task ID:\n%s", prompt)
	}
}

// ─── AC3: prompt is suitable for spawn_agent ─────────────────────────────────

// TestHandoff_PromptSuitableForSpawnAgent verifies that the prompt is a coherent
// Markdown message with all major sections when all inputs are provided.
func TestHandoff_PromptSuitableForSpawnAgent(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, profileRoot := setupHandoffTest(t)

	writeHandoffProfile(t, profileRoot, "be", []string{"Run go test -race ./..."})

	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac3")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")
	setHandoffFilesPlanned(t, entitySvc, taskID, taskSlug, []string{"internal/foo/bar.go"})
	addHandoffKnowledge(t, knowledgeSvc, "no-globals", "No global mutable state", "project", 2)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id":      taskID,
		"role":         "be",
		"instructions": "Check with orchestrator before adding dependencies.",
	})

	prompt := resp["prompt"].(string)

	// All expected top-level sections must be present.
	// Note: ### Acceptance Criteria is conditional on extracted criteria from
	// spec sections; it is not tested here because the test environment has no
	// document intelligence. See TestRenderHandoffPrompt_AcceptanceCriteria for
	// the direct rendering test.
	for _, section := range []string{
		"## Task:",
		"### Summary",
		"### Known Constraints (from knowledge base)",
		"### Files",
		"### Conventions",
		"### Additional Instructions",
	} {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing expected section %q for spawn_agent use:\n%s", section, prompt)
		}
	}
}

// ─── AC4: context_metadata ────────────────────────────────────────────────────

// TestHandoff_ContextMetadataFields verifies that context_metadata contains all
// required fields: spec_sections_included, knowledge_entries_included,
// byte_usage, byte_budget, and trimmed.
func TestHandoff_ContextMetadataFields(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac4")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")
	addHandoffKnowledge(t, knowledgeSvc, "ac4-ke", "Some constraint", "project", 2)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	meta, ok := resp["context_metadata"].(map[string]any)
	if !ok {
		t.Fatalf("context_metadata missing or wrong type: %T %v",
			resp["context_metadata"], resp["context_metadata"])
	}

	checkMetaFloat := func(key string, wantPositive bool) {
		t.Helper()
		v, ok := meta[key].(float64)
		if !ok {
			t.Errorf("context_metadata.%s missing or not a number", key)
			return
		}
		if wantPositive && v <= 0 {
			t.Errorf("context_metadata.%s = %v, want > 0", key, v)
		}
	}

	checkMetaFloat("byte_usage", true)
	checkMetaFloat("byte_budget", true)
	checkMetaFloat("spec_sections_included", false) // may be 0 — no doc intel in tests
	checkMetaFloat("knowledge_entries_included", true)

	if int(meta["byte_budget"].(float64)) != assemblyDefaultBudget {
		t.Errorf("byte_budget = %v, want %d", meta["byte_budget"], assemblyDefaultBudget)
	}

	if _, ok := meta["trimmed"]; !ok {
		t.Error("context_metadata.trimmed field missing")
	}
	if _, ok := meta["trimmed"].([]any); !ok {
		t.Errorf("context_metadata.trimmed should be a list, got %T", meta["trimmed"])
	}
}

// TestHandoff_TrimmedListPresent verifies that the trimmed field is always a
// list (empty when no trimming occurred).
func TestHandoff_TrimmedListPresent(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac4trim")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	meta := resp["context_metadata"].(map[string]any)
	trimmed, ok := meta["trimmed"].([]any)
	if !ok {
		t.Fatalf("context_metadata.trimmed should be []any, got %T", meta["trimmed"])
	}
	// For a small task with no knowledge, trimmed list should be empty.
	if len(trimmed) != 0 {
		t.Errorf("expected empty trimmed list for small task, got %d entries", len(trimmed))
	}
}

// ─── AC5: accepted statuses ───────────────────────────────────────────────────

// TestHandoff_AcceptsActiveStatus verifies that handoff succeeds for an active task.
func TestHandoff_AcceptsActiveStatus(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac5active")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	if _, hasError := resp["error"]; hasError {
		t.Errorf("unexpected error for active task: %v", resp["error"])
	}
	if resp["prompt"] == nil {
		t.Error("expected prompt for active task, got nil")
	}
}

// TestHandoff_AcceptsReadyStatus verifies that handoff succeeds for a ready task.
func TestHandoff_AcceptsReadyStatus(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac5ready")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "ready")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	if _, hasError := resp["error"]; hasError {
		t.Errorf("unexpected error for ready task: %v", resp["error"])
	}
	if resp["prompt"] == nil {
		t.Error("expected prompt for ready task, got nil")
	}
}

// TestHandoff_AcceptsNeedsReworkStatus verifies that handoff succeeds for a
// needs-rework task.
func TestHandoff_AcceptsNeedsReworkStatus(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac5rework")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "needs-rework")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	if _, hasError := resp["error"]; hasError {
		t.Errorf("unexpected error for needs-rework task: %v", resp["error"])
	}
	if resp["prompt"] == nil {
		t.Error("expected prompt for needs-rework task, got nil")
	}
}

// ─── AC6: read-only ───────────────────────────────────────────────────────────

// TestHandoff_ReadOnly verifies that calling handoff does not change the task's
// lifecycle status.
func TestHandoff_ReadOnly(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac6")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	callHandoff(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	after, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after handoff: %v", err)
	}
	statusAfter, _ := after.State["status"].(string)
	if statusAfter != "active" {
		t.Errorf("task status changed to %q after handoff; must remain active", statusAfter)
	}
}

// TestHandoff_ReadOnlyForReady verifies that handoff does not transition a ready
// task to active (unlike next/finish which have lenient lifecycle).
func TestHandoff_ReadOnlyForReady(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac6ready")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "ready")

	callHandoff(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	after, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after handoff: %v", err)
	}
	statusAfter, _ := after.State["status"].(string)
	if statusAfter != "ready" {
		t.Errorf("task status changed from ready to %q; handoff must be read-only", statusAfter)
	}
}

// ─── AC7: role shapes context ─────────────────────────────────────────────────

// TestHandoff_RoleShapesKnowledge verifies that role-scoped knowledge entries
// appear in the prompt when the matching role is provided, and are excluded for
// a different role.
func TestHandoff_RoleShapesKnowledge(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac7ke")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	addHandoffKnowledge(t, knowledgeSvc, "backend-rule", "Backend-specific constraint about caching", "backend", 2)
	addHandoffKnowledge(t, knowledgeSvc, "frontend-rule", "Frontend-specific constraint about rendering", "frontend", 2)

	// With role=backend: backend entry must appear; frontend entry must not.
	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
		"role":    "backend",
	})
	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "Backend-specific constraint about caching") {
		t.Errorf("expected backend knowledge in prompt (role=backend):\n%s", prompt)
	}
	if strings.Contains(prompt, "Frontend-specific constraint about rendering") {
		t.Errorf("unexpected frontend knowledge in prompt (role=backend):\n%s", prompt)
	}
}

// TestHandoff_RoleConventionsIncluded verifies that when a role profile exists
// with conventions, those conventions appear in the prompt.
func TestHandoff_RoleConventionsIncluded(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, profileRoot := setupHandoffTest(t)

	writeHandoffProfile(t, profileRoot, "backend", []string{
		"Run go test -race ./...",
		"Use context.Context for cancellation",
	})

	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac7conv")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
		"role":    "backend",
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "Run go test -race ./...") {
		t.Errorf("expected first role convention in prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Use context.Context for cancellation") {
		t.Errorf("expected second role convention in prompt:\n%s", prompt)
	}
}

// ─── AC8: instructions ────────────────────────────────────────────────────────

// TestHandoff_InstructionsIncluded verifies that when the instructions parameter
// is provided, it appears in the prompt under ### Additional Instructions.
func TestHandoff_InstructionsIncluded(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac8instr")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id":      taskID,
		"instructions": "Do not modify the database schema without approval.",
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "### Additional Instructions") {
		t.Errorf("prompt missing '### Additional Instructions' section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Do not modify the database schema without approval.") {
		t.Errorf("prompt missing instructions text:\n%s", prompt)
	}
}

// TestHandoff_NoInstructionsSectionWhenAbsent verifies that the ### Additional
// Instructions section is omitted when instructions is not provided.
func TestHandoff_NoInstructionsSectionWhenAbsent(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac8noinstr")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if strings.Contains(prompt, "### Additional Instructions") {
		t.Errorf("prompt must not contain '### Additional Instructions' when not provided:\n%s", prompt)
	}
}

// ─── AC9: terminal status error ───────────────────────────────────────────────

// TestHandoff_TerminalStatus_Done verifies that handoff on a done task returns
// a terminal_status error.
func TestHandoff_TerminalStatus_Done(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac9done")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "done")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for done task, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != "terminal_status" {
		t.Errorf("error code = %q, want \"terminal_status\"", code)
	}
	if msg, _ := errObj["message"].(string); !strings.Contains(msg, "terminal") {
		t.Errorf("error message does not mention terminal; got: %q", msg)
	}
}

// TestHandoff_TerminalStatus_NotPlanned verifies that handoff on a not-planned
// task returns a terminal_status error.
func TestHandoff_TerminalStatus_NotPlanned(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac9np")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "not-planned")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for not-planned task, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != "terminal_status" {
		t.Errorf("error code = %q, want \"terminal_status\"", code)
	}
}

// TestHandoff_TerminalStatus_Duplicate verifies that handoff on a duplicate task
// returns a terminal_status error.
func TestHandoff_TerminalStatus_Duplicate(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "ac9dup")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "duplicate")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for duplicate task, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != "terminal_status" {
		t.Errorf("error code = %q, want \"terminal_status\"", code)
	}
}

// ─── Edge cases ───────────────────────────────────────────────────────────────

// TestHandoff_QueuedStatusReturnsError verifies that a queued task (valid
// lifecycle state, but not accepted by handoff) returns an error.
func TestHandoff_QueuedStatusReturnsError(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	// Task starts in queued status — no advancement.
	taskID, _ := createHandoffScenario(t, entitySvc, "queued")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for queued task, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code == "" {
		t.Error("expected non-empty error code for queued task")
	}
}

// TestHandoff_TaskNotFound verifies that a non-existent task ID returns a
// not_found error.
func TestHandoff_TaskNotFound(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": "TASK-01NOTAREALID00000000000",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for unknown task ID, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("error code = %q, want \"not_found\"", code)
	}
}

// TestHandoff_ProjectScopedKnowledgeWithNoRole verifies that when no role is
// provided, only project-scoped knowledge entries appear.
func TestHandoff_ProjectScopedKnowledgeWithNoRole(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "noscope")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	addHandoffKnowledge(t, knowledgeSvc, "global-rule", "Always use structured logging", "project", 2)
	addHandoffKnowledge(t, knowledgeSvc, "be-rule", "Backend-only note", "backend", 2)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
		// no role
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "Always use structured logging") {
		t.Errorf("expected project-scoped knowledge when no role given:\n%s", prompt)
	}
	if strings.Contains(prompt, "Backend-only note") {
		t.Errorf("unexpected backend-scoped knowledge when no role given:\n%s", prompt)
	}
}

// TestHandoff_ByteUsageIsPositive verifies that byte_usage reflects the
// assembled content and does not exceed byte_budget.
func TestHandoff_ByteUsageIsPositive(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "bytes")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	// Add knowledge so there is actual content to measure.
	addHandoffKnowledge(t, knowledgeSvc, "byte-test", "Some constraint for byte measurement", "project", 2)

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	meta := resp["context_metadata"].(map[string]any)
	byteUsage := int(meta["byte_usage"].(float64))
	byteBudget := int(meta["byte_budget"].(float64))

	if byteUsage <= 0 {
		t.Errorf("byte_usage = %d, want > 0", byteUsage)
	}
	if byteUsage > byteBudget {
		t.Errorf("byte_usage (%d) > byte_budget (%d)", byteUsage, byteBudget)
	}
}

// ─── renderHandoffPrompt unit tests ──────────────────────────────────────────
//
// These tests call renderHandoffPrompt directly so they can inject pre-built
// assembledContext values (including spec-derived acceptance criteria) without
// requiring a live document intelligence index.

// TestRenderHandoffPrompt_AcceptanceCriteria verifies that when the assembled
// context contains acceptance criteria they are rendered in the prompt under
// ### Acceptance Criteria (spec §30.7 criterion 2; implementation plan G.2).
func TestRenderHandoffPrompt_AcceptanceCriteria(t *testing.T) {
	t.Parallel()

	taskState := map[string]any{
		"id":      "TASK-01RENDER0000000000001",
		"summary": "Implement the authentication flow",
	}
	actx := assembledContext{
		acceptanceCriteria: []string{
			"The system MUST authenticate users via JWT",
			"Expired tokens SHALL return HTTP 401",
			"Missing Authorization header SHALL return HTTP 401",
		},
		byteBudget: assemblyDefaultBudget,
	}

	prompt := renderHandoffPrompt(taskState, actx, "")

	if !strings.Contains(prompt, "### Acceptance Criteria") {
		t.Fatalf("prompt missing '### Acceptance Criteria' section:\n%s", prompt)
	}
	for _, criterion := range actx.acceptanceCriteria {
		if !strings.Contains(prompt, criterion) {
			t.Errorf("prompt missing acceptance criterion %q:\n%s", criterion, prompt)
		}
	}
	// Conventions now come first (high-attention zone), then acceptance criteria.
	conv := strings.Index(prompt, "### Conventions")
	crit := strings.Index(prompt, "### Acceptance Criteria")
	if conv >= crit {
		t.Errorf("'### Conventions' (pos %d) must appear before '### Acceptance Criteria' (pos %d)", conv, crit)
	}
}

// TestRenderHandoffPrompt_AcceptanceCriteriaOmittedWhenEmpty verifies that the
// ### Acceptance Criteria section is absent when no criteria were extracted.
func TestRenderHandoffPrompt_AcceptanceCriteriaOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	taskState := map[string]any{
		"id":      "TASK-01RENDER0000000000002",
		"summary": "Implement the widget",
	}
	actx := assembledContext{
		acceptanceCriteria: nil,
		byteBudget:         assemblyDefaultBudget,
	}

	prompt := renderHandoffPrompt(taskState, actx, "")

	if strings.Contains(prompt, "### Acceptance Criteria") {
		t.Errorf("prompt must not contain '### Acceptance Criteria' when criteria slice is nil:\n%s", prompt)
	}
}

// TestRenderHandoffPrompt_SectionOrder verifies the canonical ordering:
// Summary → Specification → Acceptance Criteria → Known Constraints → Files → Conventions.
func TestRenderHandoffPrompt_SectionOrder(t *testing.T) {
	t.Parallel()

	taskState := map[string]any{
		"id":      "TASK-01RENDER0000000000003",
		"summary": "Implement full stack feature",
	}
	actx := assembledContext{
		specSections: []asmSpecSection{
			{document: "spec.md", section: "Overview", content: "The system overview."},
		},
		acceptanceCriteria: []string{"Feature MUST work end-to-end"},
		knowledge: []asmKnowledgeEntry{
			{topic: "pattern", content: "Use the established pattern", scope: "project", confidence: 0.9, tier: 2},
		},
		filesContext: []asmFileEntry{{path: "internal/feature/feature.go"}},
		byteBudget:   assemblyDefaultBudget,
	}
	actx.byteUsage = asmByteCount(actx)

	prompt := renderHandoffPrompt(taskState, actx, "Additional note.")

	positions := map[string]int{}
	for _, section := range []string{
		"### Conventions",
		"### Summary",
		"### Specification",
		"### Acceptance Criteria",
		"### Known Constraints",
		"### Files",
		"### Additional Instructions",
	} {
		idx := strings.Index(prompt, section)
		if idx < 0 {
			t.Errorf("prompt missing section %q", section)
		}
		positions[section] = idx
	}

	ordered := []string{
		"### Conventions",
		"### Summary",
		"### Specification",
		"### Acceptance Criteria",
		"### Known Constraints",
		"### Files",
		"### Additional Instructions",
	}
	for i := 1; i < len(ordered); i++ {
		prev, curr := ordered[i-1], ordered[i]
		if positions[prev] >= positions[curr] {
			t.Errorf("section order violation: %q (pos %d) must come before %q (pos %d)",
				prev, positions[prev], curr, positions[curr])
		}
	}
}

// ─── Pre-dispatch state commit tests (AC-07, AC-11) ──────────────────────────

// AC-07: When handoff is called, the commitStateFunc is invoked before context
// assembly begins. This test verifies the call happens by injecting a stub.
func TestHandoff_PreDispatchCommit_CalledBeforeAssembly(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "pre-commit")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	commitCalled := false
	savedFn := commitStateFunc
	commitStateFunc = func(repoRoot string) (bool, error) {
		commitCalled = true
		return false, nil // simulate nothing to commit
	}
	defer func() { commitStateFunc = savedFn }()

	callHandoff(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	if !commitCalled {
		t.Error("commitStateFunc was not called; pre-dispatch state commit must be attempted before context assembly")
	}
}

// AC-11: If the pre-dispatch commit fails, handoff logs a warning and
// proceeds normally — the failure must not prevent context assembly or
// the prompt being returned.
func TestHandoff_PreDispatchCommit_FailureDoesNotBlockHandoff(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "commit-fail")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	savedFn := commitStateFunc
	commitStateFunc = func(repoRoot string) (bool, error) {
		return false, fmt.Errorf("simulated git commit failure: lock file exists")
	}
	defer func() { commitStateFunc = savedFn }()

	// Handoff must still succeed and return a prompt.
	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	// The response must not contain a top-level "error" key.
	if _, hasErr := resp["error"]; hasErr {
		t.Errorf("handoff returned error after commit failure; should proceed normally: %v", resp["error"])
	}
	prompt, _ := resp["prompt"].(string)
	if prompt == "" {
		t.Error("prompt is empty after commit failure; handoff must return a prompt regardless of commit errors")
	}
}

// The repoRoot passed to commitStateFunc must be "." (the server default).
func TestHandoff_PreDispatchCommit_UsesRepoRootDot(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "repo-root")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	var capturedRoot string
	savedFn := commitStateFunc
	commitStateFunc = func(repoRoot string) (bool, error) {
		capturedRoot = repoRoot
		return false, nil
	}
	defer func() { commitStateFunc = savedFn }()

	callHandoff(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	if capturedRoot != "." {
		t.Errorf("commitStateFunc called with repoRoot=%q, want \".\"", capturedRoot)
	}
}

// ─── B-15: re-review guidance injection ──────────────────────────────────────

// TestHandoff_ReReviewGuidance_NotInjectedAtCycleOne verifies that no
// re-review guidance appears in the prompt when the parent feature's
// review_cycle is 1 (below the injection threshold of >= 2).
func TestHandoff_ReReviewGuidance_NotInjectedAtCycleOne(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)

	planID := createHandoffPlan(t, entitySvc, "rerev-c1")
	featID := createHandoffFeature(t, entitySvc, planID, "rerev-feat-c1")
	advanceHandoffFeatureTo(t, entitySvc, featID, "reviewing")
	// Increment review_cycle once → cycle 1 (below the >= 2 threshold).
	if err := entitySvc.IncrementFeatureReviewCycle(featID, ""); err != nil {
		t.Fatalf("IncrementFeatureReviewCycle: %v", err)
	}

	taskID, taskSlug := createHandoffTask(t, entitySvc, featID, "rerev-task-c1")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if strings.Contains(prompt, "Re-Review Guidance") {
		t.Errorf("expected no re-review guidance for review_cycle=1; found in prompt:\n%s", prompt)
	}
}

// TestHandoff_ReReviewGuidance_InjectedAtCycleTwo verifies that the re-review
// guidance section appears when the parent feature's review_cycle is 2 (at the
// injection threshold of >= 2), and that it contains the cycle number.
func TestHandoff_ReReviewGuidance_InjectedAtCycleTwo(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)

	planID := createHandoffPlan(t, entitySvc, "rerev-c2")
	featID := createHandoffFeature(t, entitySvc, planID, "rerev-feat-c2")
	advanceHandoffFeatureTo(t, entitySvc, featID, "reviewing")
	// Increment review_cycle twice → cycle 2 (at the threshold).
	for i := 0; i < 2; i++ {
		if err := entitySvc.IncrementFeatureReviewCycle(featID, ""); err != nil {
			t.Fatalf("IncrementFeatureReviewCycle #%d: %v", i+1, err)
		}
	}

	taskID, taskSlug := createHandoffTask(t, entitySvc, featID, "rerev-task-c2")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	prompt := resp["prompt"].(string)
	if !strings.Contains(prompt, "Re-Review Guidance") {
		t.Errorf("expected re-review guidance for review_cycle=2; not found in prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Cycle 2") {
		t.Errorf("expected cycle number 2 in guidance; not found in prompt:\n%s", prompt)
	}
}

// TestHandoff_ReReviewGuidance_PrependsExistingInstructions verifies that when
// both re-review guidance (review_cycle >= 2) and caller-supplied instructions
// are present, the guidance is prepended before the instructions so neither
// is lost.
func TestHandoff_ReReviewGuidance_PrependsExistingInstructions(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)

	planID := createHandoffPlan(t, entitySvc, "rerev-pre")
	featID := createHandoffFeature(t, entitySvc, planID, "rerev-feat-pre")
	advanceHandoffFeatureTo(t, entitySvc, featID, "reviewing")
	for i := 0; i < 2; i++ {
		if err := entitySvc.IncrementFeatureReviewCycle(featID, ""); err != nil {
			t.Fatalf("IncrementFeatureReviewCycle #%d: %v", i+1, err)
		}
	}

	taskID, taskSlug := createHandoffTask(t, entitySvc, featID, "rerev-task-pre")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	const extraInstructions = "Focus on security boundaries only."
	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id":      taskID,
		"instructions": extraInstructions,
	})

	prompt := resp["prompt"].(string)

	// Both guidance and caller instructions must appear.
	if !strings.Contains(prompt, "Re-Review Guidance") {
		t.Errorf("expected re-review guidance in prompt; not found:\n%s", prompt)
	}
	if !strings.Contains(prompt, extraInstructions) {
		t.Errorf("expected caller instructions %q in prompt; not found:\n%s", extraInstructions, prompt)
	}

	// Guidance must come before the caller instructions.
	guidanceIdx := strings.Index(prompt, "Re-Review Guidance")
	instrIdx := strings.Index(prompt, extraInstructions)
	if guidanceIdx >= instrIdx {
		t.Errorf("re-review guidance (at %d) must appear before caller instructions (at %d)", guidanceIdx, instrIdx)
	}
}

// ─── Stage validation (FR-001) ────────────────────────────────────────────────

// TestHandoff_ProposedFeature_StageValidationError verifies that calling
// handoff for a task whose parent feature is in "proposed" status (a
// non-working state) returns an error response with code "stage_validation"
// (FR-001 / B-09).
//
// The task is advanced to "ready" so the handler passes the task-status check
// and reaches the feature-stage validation path.
func TestHandoff_ProposedFeature_StageValidationError(t *testing.T) {
	t.Parallel()
	entitySvc, knowledgeSvc, profileStore, _ := setupHandoffTest(t)

	// Build plan → feature (stays in "proposed") → task.
	planID := createHandoffPlan(t, entitySvc, "proposed-feat-plan")
	featID := createHandoffFeature(t, entitySvc, planID, "proposed-feat")
	// Feature deliberately left in "proposed" — do NOT call advanceHandoffFeatureTo.

	taskID, taskSlug := createHandoffTask(t, entitySvc, featID, "proposed-feat-task")
	// Advance the task to "ready" so the status gate passes.
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "ready")

	resp := callHandoffJSON(t, entitySvc, profileStore, knowledgeSvc, map[string]any{
		"task_id": taskID,
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object for proposed-feature task, got: %v", resp)
	}
	if code, _ := errObj["code"].(string); code != "stage_validation" {
		t.Errorf("error code = %q, want \"stage_validation\"", code)
	}
	if msg, _ := errObj["message"].(string); msg == "" {
		t.Error("expected non-empty error message")
	}
}
