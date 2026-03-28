package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// setupStatusTest creates an entitySvc and docSvc backed by temp dirs,
// suitable for status tool unit tests.
func setupStatusTest(t *testing.T) (*service.EntityService, *service.DocumentService) {
	t.Helper()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	return entitySvc, docSvc
}

// createTestPlan creates a plan record directly via the storage layer,
// bypassing the config-dependent CreatePlan path so tests run without .kbz/config.yaml.
func createTestPlan(t *testing.T, entitySvc *service.EntityService, slug, title string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	// Use a deterministic but unique-enough ID for tests.
	id := "P1-" + slug
	record := storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":         id,
			"slug":       slug,
			"title":      title,
			"status":     "proposed",
			"summary":    "Test plan " + slug,
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createTestPlan(%s): %v", slug, err)
	}
	return id
}

// createStatusTestFeature creates a feature for status tests.
func createStatusTestFeature(t *testing.T, entitySvc *service.EntityService, parentPlanID, slug, summary string) string {
	t.Helper()
	result, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Slug:      slug,
		Parent:    parentPlanID,
		Summary:   summary,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%s): %v", slug, err)
	}
	return result.ID
}

// createStatusTestTask creates a task entity for status tests.
func createStatusTestTask(t *testing.T, entitySvc *service.EntityService, parentFeatID, slug, summary string) string {
	t.Helper()
	result, err := entitySvc.CreateTask(service.CreateTaskInput{
		ParentFeature: parentFeatID,
		Slug:          slug,
		Summary:       summary,
	})
	if err != nil {
		t.Fatalf("CreateTask(%s): %v", slug, err)
	}
	return result.ID
}

// callStatus invokes the status tool directly (not via MCP transport) and
// returns the parsed JSON response. Passes nil for the worktree store.
func callStatus(t *testing.T, entitySvc *service.EntityService, docSvc *service.DocumentService, id string) map[string]any {
	t.Helper()
	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": id})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("status handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse status result: %v\nraw: %s", err, text)
	}
	return parsed
}

// ─── inferIDType tests ────────────────────────────────────────────────────────

func TestInferIDType_Empty(t *testing.T) {
	t.Parallel()
	if got := inferIDType(""); got != idTypeNone {
		t.Errorf("inferIDType(\"\") = %v, want idTypeNone", got)
	}
}

func TestInferIDType_Feature(t *testing.T) {
	t.Parallel()
	cases := []string{"FEAT-01ABCDEF", "feat-01ABCDEF"}
	for _, id := range cases {
		if got := inferIDType(id); got != idTypeFeature {
			t.Errorf("inferIDType(%q) = %v, want idTypeFeature", id, got)
		}
	}
}

func TestInferIDType_Task(t *testing.T) {
	t.Parallel()
	cases := []string{"TASK-01ABCDEF", "task-01ABCDEF", "T-01ABCDEF", "t-01abcdef"}
	for _, id := range cases {
		if got := inferIDType(id); got != idTypeTask {
			t.Errorf("inferIDType(%q) = %v, want idTypeTask", id, got)
		}
	}
}

func TestInferIDType_Bug(t *testing.T) {
	t.Parallel()
	cases := []string{"BUG-01ABCDEF", "bug-01ABCDEF"}
	for _, id := range cases {
		if got := inferIDType(id); got != idTypeBug {
			t.Errorf("inferIDType(%q) = %v, want idTypeBug", id, got)
		}
	}
}

func TestInferIDType_Plan(t *testing.T) {
	t.Parallel()
	cases := []string{"P1-my-plan", "X42-something"}
	for _, id := range cases {
		if got := inferIDType(id); got != idTypePlan {
			t.Errorf("inferIDType(%q) = %v, want idTypePlan", id, got)
		}
	}
}

func TestInferIDType_Unknown(t *testing.T) {
	t.Parallel()
	cases := []string{"just-a-string", "12345", "UNKNOWN-TYPE"}
	for _, id := range cases {
		if got := inferIDType(id); got != idTypeUnknown {
			t.Errorf("inferIDType(%q) = %v, want idTypeUnknown", id, got)
		}
	}
}

// ─── isTerminalStatus tests ───────────────────────────────────────────────────

func TestIsTerminalStatus(t *testing.T) {
	t.Parallel()
	terminal := []string{"done", "not-planned", "duplicate"}
	for _, s := range terminal {
		if !isTerminalStatus(s) {
			t.Errorf("isTerminalStatus(%q) = false, want true", s)
		}
	}
	nonTerminal := []string{"queued", "ready", "active", "needs-review", "needs-rework", ""}
	for _, s := range nonTerminal {
		if isTerminalStatus(s) {
			t.Errorf("isTerminalStatus(%q) = true, want false", s)
		}
	}
}

// ─── hasDocType tests ─────────────────────────────────────────────────────────

func TestHasDocType(t *testing.T) {
	t.Parallel()

	docs := []service.DocumentResult{
		{Type: "specification", Status: "approved"},
		{Type: "design", Status: "draft"},
		{Type: "dev-plan", Status: "superseded"},
	}

	if !hasDocType(docs, "specification") {
		t.Error("hasDocType: specification should be found")
	}
	if !hasDocType(docs, "design") {
		t.Error("hasDocType: design should be found")
	}
	// superseded dev-plan should NOT count.
	if hasDocType(docs, "dev-plan") {
		t.Error("hasDocType: superseded dev-plan should not be found")
	}
	if hasDocType(docs, "report") {
		t.Error("hasDocType: report should not be found (not present)")
	}
}

// ─── synthesiseProject tests ──────────────────────────────────────────────────

func TestSynthesiseProject_Empty(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: status() returns project overview.
	entitySvc, docSvc := setupStatusTest(t)
	overview, err := synthesiseProject(entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesiseProject error: %v", err)
	}
	if overview.Scope != "project" {
		t.Errorf("Scope = %q, want project", overview.Scope)
	}
	if overview.GeneratedAt == "" {
		t.Error("GeneratedAt is empty")
	}
	if len(overview.Plans) != 0 {
		t.Errorf("Plans len = %d, want 0 (empty project)", len(overview.Plans))
	}
}

func TestSynthesiseProject_WithPlans(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: project overview covers multiple plans.
	entitySvc, docSvc := setupStatusTest(t)

	planAID := createTestPlan(t, entitySvc, "alpha", "Alpha Plan")
	planBID := createTestPlan(t, entitySvc, "beta", "Beta Plan")

	featAID := createStatusTestFeature(t, entitySvc, planAID, "feat-a", "Feature A")
	_ = createStatusTestTask(t, entitySvc, featAID, "task-1", "Task 1")

	_ = createStatusTestFeature(t, entitySvc, planBID, "feat-b", "Feature B")

	overview, err := synthesiseProject(entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesiseProject error: %v", err)
	}

	if len(overview.Plans) != 2 {
		t.Errorf("Plans len = %d, want 2", len(overview.Plans))
	}
	if overview.Total.Plans != 2 {
		t.Errorf("Total.Plans = %d, want 2", overview.Total.Plans)
	}
	if overview.Total.Features != 2 {
		t.Errorf("Total.Features = %d, want 2", overview.Total.Features)
	}
	if overview.Total.Tasks.Total != 1 {
		t.Errorf("Total.Tasks.Total = %d, want 1", overview.Total.Tasks.Total)
	}
}

func TestSynthesiseProject_HasHealthSummary(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 8: health summary is included in project view.
	entitySvc, docSvc := setupStatusTest(t)
	createTestPlan(t, entitySvc, "health-plan", "Health Plan")

	overview, err := synthesiseProject(entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesiseProject error: %v", err)
	}
	if overview.Health == nil {
		t.Fatal("Health is nil, want non-nil health summary")
	}
	// Errors and warnings should be non-negative (zero is fine for a clean project).
	if overview.Health.Errors < 0 {
		t.Errorf("Health.Errors = %d, want >= 0", overview.Health.Errors)
	}
	if overview.Health.Warnings < 0 {
		t.Errorf("Health.Warnings = %d, want >= 0", overview.Health.Warnings)
	}
}

// ─── synthesisePlan tests ─────────────────────────────────────────────────────

func TestSynthesisePlan_NotFound(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: unknown ID returns clear error.
	entitySvc, docSvc := setupStatusTest(t)
	_, err := synthesisePlan("P99-nonexistent", entitySvc, docSvc)
	if err == nil {
		t.Fatal("synthesisePlan: expected error for unknown plan, got nil")
	}
}

func TestSynthesisePlan_WithFeatures(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: plan dashboard with features in different states.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "test-plan", "Test Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "my-feature", "My Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "task-a", "Task A")
	_ = createStatusTestTask(t, entitySvc, featID, "task-b", "Task B")

	dashboard, err := synthesisePlan(planID, entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesisePlan error: %v", err)
	}
	if dashboard.Scope != "plan" {
		t.Errorf("Scope = %q, want plan", dashboard.Scope)
	}
	if dashboard.Plan.ID != planID {
		t.Errorf("Plan.ID = %q, want %q", dashboard.Plan.ID, planID)
	}
	if len(dashboard.Features) != 1 {
		t.Fatalf("Features len = %d, want 1", len(dashboard.Features))
	}
	if dashboard.Features[0].Tasks.Total != 2 {
		t.Errorf("Features[0].Tasks.Total = %d, want 2", dashboard.Features[0].Tasks.Total)
	}
}

func TestSynthesisePlan_DocGaps(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: plan dashboard reports document gaps.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "gap-plan", "Gap Plan")
	_ = createStatusTestFeature(t, entitySvc, planID, "no-spec-feature", "Feature Without Spec")

	dashboard, err := synthesisePlan(planID, entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesisePlan error: %v", err)
	}

	// Feature has no spec — should appear in doc_gaps.
	if len(dashboard.DocGaps) == 0 {
		t.Error("DocGaps is empty, want at least one gap (missing spec)")
	}
	found := false
	for _, gap := range dashboard.DocGaps {
		if strings.Contains(gap, "specification") || strings.Contains(gap, "spec") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DocGaps %v should mention missing specification", dashboard.DocGaps)
	}
}

func TestSynthesisePlan_HasHealthSummary(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 8: health summary is included in plan view.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "health-plan-dash", "Health Plan")

	dashboard, err := synthesisePlan(planID, entitySvc, docSvc)
	if err != nil {
		t.Fatalf("synthesisePlan error: %v", err)
	}
	if dashboard.Health == nil {
		t.Fatal("Health is nil, want non-nil health summary in plan dashboard")
	}
}

// ─── synthesiseFeature tests ──────────────────────────────────────────────────

func TestSynthesiseFeature_NotFound(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	_, err := synthesiseFeature("FEAT-01NOTEXIST1", entitySvc, docSvc, nil)
	if err == nil {
		t.Fatal("synthesiseFeature: expected error for unknown feature, got nil")
	}
}

func TestSynthesiseFeature_WithTasks(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: feature detail with task breakdown.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "feat-detail-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "detail-feat", "Detail Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "task-one", "Task One")
	_ = createStatusTestTask(t, entitySvc, featID, "task-two", "Task Two")
	_ = createStatusTestTask(t, entitySvc, featID, "task-three", "Task Three")

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}

	if detail.Scope != "feature" {
		t.Errorf("Scope = %q, want feature", detail.Scope)
	}
	if detail.Feature.ID != featID {
		t.Errorf("Feature.ID = %q, want %q", detail.Feature.ID, featID)
	}
	if len(detail.Tasks) != 3 {
		t.Errorf("Tasks len = %d, want 3", len(detail.Tasks))
	}
	if detail.TaskSummary.Total != 3 {
		t.Errorf("TaskSummary.Total = %d, want 3", detail.TaskSummary.Total)
	}
	if detail.GeneratedAt == "" {
		t.Error("GeneratedAt is empty")
	}
}

func TestSynthesiseFeature_PlanIDPopulated(t *testing.T) {
	t.Parallel()
	// Verifies that feature.plan_id is populated from the feature's parent field.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "plan-id-check", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "plan-id-feat", "Feature")

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}
	if detail.Feature.PlanID != planID {
		t.Errorf("Feature.PlanID = %q, want %q", detail.Feature.PlanID, planID)
	}
}

func TestSynthesiseFeature_AttentionIncludesMissingDocs(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "att-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "att-feat", "Attention Feature")

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}

	// With no tasks and no docs, attention should flag missing spec and no tasks.
	foundSpec := false
	foundTasks := false
	for _, item := range detail.Attention {
		if strings.Contains(strings.ToLower(item), "spec") {
			foundSpec = true
		}
		if strings.Contains(strings.ToLower(item), "task") || strings.Contains(strings.ToLower(item), "decompose") {
			foundTasks = true
		}
	}
	if !foundSpec {
		t.Errorf("Attention %v should mention missing specification", detail.Attention)
	}
	if !foundTasks {
		t.Errorf("Attention %v should mention missing tasks", detail.Attention)
	}
}

func TestSynthesiseFeature_WithWorktree(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 3: feature detail includes worktree when present.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "wt-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "wt-feat", "Worktree Feature")

	// Create a worktree store and insert a tracking record for the feature.
	stateRoot := t.TempDir()
	wtStore := worktree.NewStore(stateRoot)
	_, err := wtStore.Create(worktree.Record{
		EntityID:  featID,
		Branch:    "feat/wt-feat",
		Path:      ".kbz/worktrees/feat-wt-feat",
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("worktree.Create: %v", err)
	}

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}

	if detail.Worktree == nil {
		t.Fatal("Worktree is nil, want non-nil when a worktree record exists")
	}
	if detail.Worktree.Status != "active" {
		t.Errorf("Worktree.Status = %q, want active", detail.Worktree.Status)
	}
	if detail.Worktree.Branch != "feat/wt-feat" {
		t.Errorf("Worktree.Branch = %q, want feat/wt-feat", detail.Worktree.Branch)
	}
	if detail.Worktree.Path != ".kbz/worktrees/feat-wt-feat" {
		t.Errorf("Worktree.Path = %q, want .kbz/worktrees/feat-wt-feat", detail.Worktree.Path)
	}
}

func TestSynthesiseFeature_NoWorktree(t *testing.T) {
	t.Parallel()
	// Verifies that worktree field is omitted when no record exists.
	entitySvc, docSvc := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "no-wt-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "no-wt-feat", "Feature")

	// Store exists but has no record for this feature.
	stateRoot := t.TempDir()
	wtStore := worktree.NewStore(stateRoot)

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}
	if detail.Worktree != nil {
		t.Errorf("Worktree = %v, want nil when no worktree record exists", detail.Worktree)
	}
}

// ─── synthesiseTask tests ─────────────────────────────────────────────────────

func TestSynthesiseTask_NotFound(t *testing.T) {
	t.Parallel()
	entitySvc, _ := setupStatusTest(t)
	_, err := synthesiseTask("TASK-01NOTEXIST1", entitySvc)
	if err == nil {
		t.Fatal("synthesiseTask: expected error for unknown task, got nil")
	}
}

func TestSynthesiseTask_Basic(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: task detail with dependencies.
	entitySvc, _ := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "task-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "task-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "my-task", "My Task")

	detail, err := synthesiseTask(taskID, entitySvc)
	if err != nil {
		t.Fatalf("synthesiseTask error: %v", err)
	}

	if detail.Scope != "task" {
		t.Errorf("Scope = %q, want task", detail.Scope)
	}
	if detail.Task.ID != taskID {
		t.Errorf("Task.ID = %q, want %q", detail.Task.ID, taskID)
	}
	if detail.Task.ParentFeature != featID {
		t.Errorf("Task.ParentFeature = %q, want %q", detail.Task.ParentFeature, featID)
	}
	if detail.ParentFeature == nil {
		t.Error("ParentFeature = nil, want populated")
	} else if detail.ParentFeature.ID != featID {
		t.Errorf("ParentFeature.ID = %q, want %q", detail.ParentFeature.ID, featID)
	}
	if detail.GeneratedAt == "" {
		t.Error("GeneratedAt is empty")
	}
}

func TestSynthesiseTask_ParentFeaturePlanID(t *testing.T) {
	t.Parallel()
	// Verifies that parent_feature.plan_id is correctly populated from the feature's
	// "parent" field (not "owner", which does not exist on feature records).
	entitySvc, _ := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "plan-id-task-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "plan-id-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "plan-id-task", "Task")

	detail, err := synthesiseTask(taskID, entitySvc)
	if err != nil {
		t.Fatalf("synthesiseTask error: %v", err)
	}

	if detail.ParentFeature == nil {
		t.Fatal("ParentFeature is nil")
	}
	if detail.ParentFeature.PlanID != planID {
		t.Errorf("ParentFeature.PlanID = %q, want %q (check field reads 'parent' not 'owner')",
			detail.ParentFeature.PlanID, planID)
	}
}

func TestSynthesiseTask_AttentionReadyTask(t *testing.T) {
	t.Parallel()
	entitySvc, _ := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "ready-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "ready-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "ready-task", "Ready Task")

	// Advance task to ready status.
	_, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type:   "task",
		ID:     taskID,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus to ready: %v", err)
	}

	detail, err := synthesiseTask(taskID, entitySvc)
	if err != nil {
		t.Fatalf("synthesiseTask error: %v", err)
	}

	foundReady := false
	for _, item := range detail.Attention {
		if strings.Contains(strings.ToLower(item), "ready") || strings.Contains(strings.ToLower(item), "next") {
			foundReady = true
			break
		}
	}
	if !foundReady {
		t.Errorf("Attention %v should mention task is ready to claim", detail.Attention)
	}
}

func TestSynthesiseTask_DispatchInfo(t *testing.T) {
	t.Parallel()
	// Verifies §10.6: task detail includes dispatch info for active tasks.
	entitySvc, _ := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "dispatch-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "dispatch-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "dispatch-task", "Dispatch Task")

	// Manually write dispatch fields onto the task record, simulating what
	// dispatch_task does. Resolve the slug first since Store().Load requires it.
	taskGet, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("get task %s: %v", taskID, err)
	}
	taskRec, err := entitySvc.Store().Load("task", taskID, taskGet.Slug)
	if err != nil {
		t.Fatalf("read task record: %v", err)
	}
	taskRec.Fields["dispatched_to"] = "backend"
	taskRec.Fields["dispatched_at"] = "2026-01-01T10:00:00Z"
	taskRec.Fields["dispatched_by"] = "orchestrator-session-test"
	if _, err := entitySvc.Store().Write(taskRec); err != nil {
		t.Fatalf("write dispatch fields: %v", err)
	}

	detail, err := synthesiseTask(taskID, entitySvc)
	if err != nil {
		t.Fatalf("synthesiseTask error: %v", err)
	}

	if detail.Dispatch == nil {
		t.Fatal("Dispatch is nil, want non-nil for task with dispatched_to set")
	}
	if detail.Dispatch.DispatchedTo != "backend" {
		t.Errorf("Dispatch.DispatchedTo = %q, want backend", detail.Dispatch.DispatchedTo)
	}
	if detail.Dispatch.DispatchedAt != "2026-01-01T10:00:00Z" {
		t.Errorf("Dispatch.DispatchedAt = %q, want 2026-01-01T10:00:00Z", detail.Dispatch.DispatchedAt)
	}
	if detail.Dispatch.DispatchedBy != "orchestrator-session-test" {
		t.Errorf("Dispatch.DispatchedBy = %q, want orchestrator-session-test", detail.Dispatch.DispatchedBy)
	}
}

func TestSynthesiseTask_NoDispatchInfo(t *testing.T) {
	t.Parallel()
	// Verifies that dispatch field is omitted when task has not been dispatched.
	entitySvc, _ := setupStatusTest(t)

	planID := createTestPlan(t, entitySvc, "no-dispatch-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "no-dispatch-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "no-dispatch-task", "Task")

	detail, err := synthesiseTask(taskID, entitySvc)
	if err != nil {
		t.Fatalf("synthesiseTask error: %v", err)
	}
	if detail.Dispatch != nil {
		t.Errorf("Dispatch = %v, want nil for undispatched task", detail.Dispatch)
	}
}

// ─── status tool handler tests ────────────────────────────────────────────────

func TestStatusTool_UnknownIDFormat(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: unknown ID format returns clear error.
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil)

	req := makeRequest(map[string]any{"id": "TOTALLY-INVALID-ID-FORMAT"})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	errField, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error field, got: %v", parsed)
	}
	if errField["code"] != "unknown_id_format" {
		t.Errorf("error.code = %v, want unknown_id_format", errField["code"])
	}
}

func TestStatusTool_EntityNotFound(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: entity not found returns clear error.
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil)

	req := makeRequest(map[string]any{"id": "FEAT-01NOTEXIST1"})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	errField, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error field, got: %v", parsed)
	}
	code, _ := errField["code"].(string)
	if code != "not_found" && code != "status_error" {
		t.Errorf("error.code = %q, want not_found or status_error", code)
	}
}

func TestStatusTool_ProjectOverview(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: status() (no id) returns project overview.
	entitySvc, docSvc := setupStatusTest(t)
	createTestPlan(t, entitySvc, "p-one", "Plan One")

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "project" {
		t.Errorf("scope = %v, want project", parsed["scope"])
	}
}

func TestStatusTool_PlanDashboard(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: status(plan_id) returns plan dashboard.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "dashboard-plan", "Dashboard Plan")

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": planID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "plan" {
		t.Errorf("scope = %v, want plan", parsed["scope"])
	}
	planField, ok := parsed["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan field missing or wrong type: %v", parsed)
	}
	if planField["id"] != planID {
		t.Errorf("plan.id = %v, want %v", planField["id"], planID)
	}
}

func TestStatusTool_FeatureDetail(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "feat-detail-p", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "detail-f", "Detail Feature")

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": featID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "feature" {
		t.Errorf("scope = %v, want feature", parsed["scope"])
	}
}

func TestStatusTool_TaskDetail(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "task-detail-p", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "task-detail-f", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "task-detail-t", "Task")

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": taskID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "task" {
		t.Errorf("scope = %v, want task", parsed["scope"])
	}
}

func TestStatusTool_AttentionItemsGenerated(t *testing.T) {
	t.Parallel()
	// Verifies §30.4: attention items generated on plan dashboard with missing docs.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "att-items-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "att-feat", "Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "att-task", "Task")

	// Advance task to ready.
	taskRes, _ := entitySvc.List("task")
	if len(taskRes) > 0 {
		_, _ = entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     taskRes[0].ID,
			Status: "ready",
		})
	}

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": planID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	attention, _ := parsed["attention"].([]any)
	if len(attention) == 0 {
		t.Error("attention is empty, want at least one item (ready task or missing spec)")
	}
}

func TestStatusTool_ResponseHasGeneratedAt(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if _, ok := parsed["generated_at"]; !ok {
		t.Error("generated_at field missing from status response")
	}
}

func TestStatusTool_NoSideEffects(t *testing.T) {
	t.Parallel()
	// Verifies status is read-only (no side_effects field).
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	if strings.Contains(text, "side_effects") {
		t.Error("status response contains side_effects, but status is read-only")
	}
}

func TestStatusTool_ProjectOverview_HasHealth(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 8: health field present in project overview response.
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	health, ok := parsed["health"].(map[string]any)
	if !ok {
		t.Fatalf("health field missing or wrong type in project overview: %v", parsed)
	}
	if _, ok := health["errors"]; !ok {
		t.Error("health.errors field missing")
	}
	if _, ok := health["warnings"]; !ok {
		t.Error("health.warnings field missing")
	}
}

func TestStatusTool_PlanDashboard_HasHealth(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 8: health field present in plan dashboard response.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "health-dash-plan", "Health Plan")

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": planID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	health, ok := parsed["health"].(map[string]any)
	if !ok {
		t.Fatalf("health field missing or wrong type in plan dashboard: %v", parsed)
	}
	if _, ok := health["errors"]; !ok {
		t.Error("health.errors field missing")
	}
	if _, ok := health["warnings"]; !ok {
		t.Error("health.warnings field missing")
	}
}

func TestStatusTool_FeatureDetail_HasWorktreeWhenPresent(t *testing.T) {
	t.Parallel()
	// Verifies §30.4 criterion 3: feature detail includes worktree field when a
	// worktree record exists for the feature.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "wt-tool-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "wt-tool-feat", "Feature")

	stateRoot := t.TempDir()
	wtStore := worktree.NewStore(stateRoot)
	_, err := wtStore.Create(worktree.Record{
		EntityID:  featID,
		Branch:    "feat/wt-tool-feat",
		Path:      ".kbz/worktrees/feat-wt-tool-feat",
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("worktree.Create: %v", err)
	}

	tool := statusTool(entitySvc, docSvc, wtStore)
	req := makeRequest(map[string]any{"id": featID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "feature" {
		t.Fatalf("scope = %v, want feature", parsed["scope"])
	}
	wt, ok := parsed["worktree"].(map[string]any)
	if !ok {
		t.Fatalf("worktree field missing or wrong type: %v", parsed)
	}
	if wt["status"] != "active" {
		t.Errorf("worktree.status = %v, want active", wt["status"])
	}
	if wt["branch"] != "feat/wt-tool-feat" {
		t.Errorf("worktree.branch = %v, want feat/wt-tool-feat", wt["branch"])
	}
}

func TestStatusTool_TaskDetail_HasDispatch(t *testing.T) {
	t.Parallel()
	// Verifies §10.6: task detail includes dispatch info for a dispatched task.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "dispatch-tool-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "dispatch-tool-feat", "Feature")
	taskID := createStatusTestTask(t, entitySvc, featID, "dispatch-tool-task", "Task")

	// Manually write dispatch fields. Resolve the slug first since Store().Load requires it.
	taskGet, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("get task %s: %v", taskID, err)
	}
	taskRec, err := entitySvc.Store().Load("task", taskID, taskGet.Slug)
	if err != nil {
		t.Fatalf("read task record: %v", err)
	}
	taskRec.Fields["dispatched_to"] = "backend"
	taskRec.Fields["dispatched_at"] = "2026-06-01T09:00:00Z"
	taskRec.Fields["dispatched_by"] = "orch-test"
	if _, err := entitySvc.Store().Write(taskRec); err != nil {
		t.Fatalf("write task record: %v", err)
	}

	tool := statusTool(entitySvc, docSvc, nil)
	req := makeRequest(map[string]any{"id": taskID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}
	if parsed["scope"] != "task" {
		t.Fatalf("scope = %v, want task", parsed["scope"])
	}
	dispatch, ok := parsed["dispatch"].(map[string]any)
	if !ok {
		t.Fatalf("dispatch field missing or wrong type: %v", parsed)
	}
	if dispatch["dispatched_to"] != "backend" {
		t.Errorf("dispatch.dispatched_to = %v, want backend", dispatch["dispatched_to"])
	}
}
