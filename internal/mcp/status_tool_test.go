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
func createTestPlan(t *testing.T, entitySvc *service.EntityService, slug, name string) string {
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
			"name":       name,
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
		Name:      "test",
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
		Name:          "test",
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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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
	overview, err := synthesiseProject(entitySvc, docSvc, nil, "")
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

	overview, err := synthesiseProject(entitySvc, docSvc, nil, "")
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

	overview, err := synthesiseProject(entitySvc, docSvc, nil, "")
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
	_, err := synthesiseFeature("FEAT-01NOTEXIST1", entitySvc, docSvc, nil, "", 7)
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

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil, "", 7)
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

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil, "", 7)
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

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, nil, "", 7)
	if err != nil {
		t.Fatalf("synthesiseFeature error: %v", err)
	}

	// With no tasks and no docs, attention should flag missing spec and no tasks.
	foundSpec := false
	foundTasks := false
	for _, item := range detail.Attention {
		if strings.Contains(strings.ToLower(item.Message), "spec") {
			foundSpec = true
		}
		if strings.Contains(strings.ToLower(item.Message), "task") || strings.Contains(strings.ToLower(item.Message), "decompose") {
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

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore, "", 7)
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

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore, "", 7)
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
		if strings.Contains(strings.ToLower(item.Message), "ready") || strings.Contains(strings.ToLower(item.Message), "next") {
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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)

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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)

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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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
	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	tool := statusTool(entitySvc, docSvc, wtStore, "", 0)
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

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

// ─── Orientation breadcrumb tests (AC-E1, AC-E2, AC-E3, AC-E7) ───────────────

func TestStatusTool_ProjectOverview_HasOrientation(t *testing.T) {
	t.Parallel()
	// AC-E1: status with no id returns orientation field.
	// AC-E2: orientation.message references getting-started skill path.
	// AC-E3: orientation.skills_path is ".agents/skills/".
	entitySvc, docSvc := setupStatusTest(t)
	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	// AC-E1: orientation field must be present.
	orientation, ok := parsed["orientation"].(map[string]any)
	if !ok {
		t.Fatalf("orientation field missing or wrong type in project overview: %v", parsed)
	}

	// AC-E2: message must reference the getting-started skill.
	message, _ := orientation["message"].(string)
	if !strings.Contains(message, "kanbanzai-getting-started") {
		t.Errorf("orientation.message does not reference getting-started skill: %q", message)
	}
	if !strings.Contains(message, ".agents/skills/") {
		t.Errorf("orientation.message does not reference .agents/skills/ path: %q", message)
	}

	// AC-E3: skills_path must be ".agents/skills/".
	skillsPath, _ := orientation["skills_path"].(string)
	if skillsPath != ".agents/skills/" {
		t.Errorf("orientation.skills_path = %q, want .agents/skills/", skillsPath)
	}
}

func TestStatusTool_ProjectOverview_OrientationDoesNotBreakExistingFields(t *testing.T) {
	t.Parallel()
	// AC-E7: existing fields in status response are unchanged by orientation addition.
	entitySvc, docSvc := setupStatusTest(t)
	createTestPlan(t, entitySvc, "orient-plan", "Orient Plan")

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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

	// All pre-existing top-level fields must still be present.
	for _, field := range []string{"scope", "plans", "total", "generated_at"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("expected field %q missing from project overview response", field)
		}
	}
	if parsed["scope"] != "project" {
		t.Errorf("scope = %v, want project", parsed["scope"])
	}
}

func TestStatusTool_PlanDashboard_NoOrientation(t *testing.T) {
	t.Parallel()
	// Orientation is project-scope only — plan dashboard must not include it.
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "no-orient-plan", "No Orient Plan")

	tool := statusTool(entitySvc, docSvc, nil, "", 0)
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
	if _, ok := parsed["orientation"]; ok {
		t.Error("orientation field must not appear in plan dashboard response")
	}
}

// ─── generateFeatureAttention tests ──────────────────────────────────────────

// TestFeatureAttention_AllTasksDone_Developing verifies that when all tasks are
// terminal and the feature is developing, a "ready to advance" item is emitted.
func TestFeatureAttention_AllTasksDone_Developing(t *testing.T) {
	tasks := []taskInfo{
		{Status: "done"},
		{Status: "done"},
		{Status: "not-planned"},
	}
	items := generateFeatureAttention(tasks, nil, 3, "FEAT-01", "FEAT-01", "developing", time.Time{}, true, true, 7, nil, false, "")
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "FEAT-01") && strings.Contains(item.Message, "3/3") && strings.Contains(item.Message, "ready to advance") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'ready to advance' attention item, got: %v", items)
	}
}

// TestFeatureAttention_AllTasksDone_NeedsRework verifies needs-rework also triggers the item.
func TestFeatureAttention_AllTasksDone_NeedsRework(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}, {Status: "done"}}
	items := generateFeatureAttention(tasks, nil, 2, "FEAT-02", "FEAT-02", "needs-rework", time.Time{}, true, true, 7, nil, false, "")
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "FEAT-02") && strings.Contains(item.Message, "2/2") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected completion item for needs-rework, got: %v", items)
	}
}

// TestFeatureAttention_AllTasksDone_Reviewing verifies that reviewing status
// does NOT trigger the completion item.
func TestFeatureAttention_AllTasksDone_Reviewing(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}, {Status: "done"}}
	items := generateFeatureAttention(tasks, nil, 2, "FEAT-03", "FEAT-03", "reviewing", time.Time{}, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.Contains(item.Message, "ready to advance") {
			t.Errorf("unexpected 'ready to advance' item for reviewing status: %s", item.Message)
		}
	}
}

// TestFeatureAttention_ZeroTasks_NoCompletionItem verifies that zero tasks
// does NOT trigger the completion item.
func TestFeatureAttention_ZeroTasks_NoCompletionItem(t *testing.T) {
	items := generateFeatureAttention(nil, nil, 0, "FEAT-04", "FEAT-04", "developing", time.Time{}, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.Contains(item.Message, "ready to advance") {
			t.Errorf("unexpected completion item for zero tasks: %s", item.Message)
		}
	}
}

// TestFeatureAttention_NonTerminalTask_NoCompletionItem verifies that a non-terminal
// task blocks the completion item.
func TestFeatureAttention_NonTerminalTask_NoCompletionItem(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}, {Status: "active"}}
	items := generateFeatureAttention(tasks, nil, 2, "FEAT-05", "FEAT-05", "developing", time.Time{}, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.Contains(item.Message, "ready to advance") {
			t.Errorf("unexpected completion item when active task present: %s", item.Message)
		}
	}
}

// TestFeatureAttention_StalePrefix_After48h verifies that the ⚠️ STALE prefix is
// added for features in developing that have been updated >48h ago.
func TestFeatureAttention_StalePrefix_After48h(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}}
	staleTime := time.Now().Add(-49 * time.Hour)
	items := generateFeatureAttention(tasks, nil, 1, "FEAT-06", "FEAT-06", "developing", staleTime, true, true, 7, nil, false, "")
	found := false
	for _, item := range items {
		if strings.HasPrefix(item.Message, "⚠️ STALE:") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected STALE prefix for 49h-old developing feature, got: %v", items)
	}
}

// TestFeatureAttention_NoStalePrefix_Recent verifies that a recently updated feature
// does NOT get the STALE prefix.
func TestFeatureAttention_NoStalePrefix_Recent(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}}
	recentTime := time.Now().Add(-1 * time.Hour)
	items := generateFeatureAttention(tasks, nil, 1, "FEAT-07", "FEAT-07", "developing", recentTime, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.HasPrefix(item.Message, "⚠️ STALE:") {
			t.Errorf("unexpected STALE prefix for recently updated feature: %s", item.Message)
		}
	}
}

// TestFeatureAttention_NeedsRework_NoStalePrefix verifies that needs-rework does NOT
// get the STALE prefix even if updated >48h ago.
func TestFeatureAttention_NeedsRework_NoStalePrefix(t *testing.T) {
	tasks := []taskInfo{{Status: "done"}}
	staleTime := time.Now().Add(-72 * time.Hour)
	items := generateFeatureAttention(tasks, nil, 1, "FEAT-08", "FEAT-08", "needs-rework", staleTime, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.HasPrefix(item.Message, "⚠️ STALE:") {
			t.Errorf("unexpected STALE prefix for needs-rework feature: %s", item.Message)
		}
	}
}

// TestFeatureAttention_InheritedSpec_NoWarning verifies that when inheritedHasSpec=true,
// the "Missing specification" attention item is NOT emitted.
func TestFeatureAttention_InheritedSpec_NoWarning(t *testing.T) {
	items := generateFeatureAttention(nil, nil, 0, "FEAT-09", "FEAT-09", "dev-planning", time.Time{}, true, true, 7, nil, false, "")
	for _, item := range items {
		if strings.Contains(item.Message, "Missing specification") {
			t.Errorf("unexpected Missing specification item when inherited: %s", item.Message)
		}
	}
}

// TestFeatureAttention_NoInheritedSpec_Warning verifies that when inheritedHasSpec=false
// and no feature-owned spec exists, the warning is emitted.
func TestFeatureAttention_NoInheritedSpec_Warning(t *testing.T) {
	items := generateFeatureAttention(nil, nil, 0, "FEAT-10", "FEAT-10", "specifying", time.Time{}, false, true, 7, nil, false, "")
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "Missing specification") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Missing specification warning, got: %v", items)
	}
}

// ─── missing_graph_index attention tests (AC-012, AC-013, AC-014) ────────────

// TestFeatureAttention_MissingGraphIndex verifies AC-012:
// Worktree exists and active with empty GraphProject → missing_graph_index emitted.
func TestFeatureAttention_MissingGraphIndex(t *testing.T) {
	t.Parallel()
	items := generateFeatureAttention(nil, nil, 1, "FEAT-GI1", "FEAT-GI1", "developing", time.Time{}, true, true, 7, nil, true, "")
	found := false
	for _, item := range items {
		if item.Type == "missing_graph_index" {
			found = true
			if item.Severity != "info" {
				t.Errorf("severity = %q, want info", item.Severity)
			}
			if !strings.Contains(item.Message, "index_repository") {
				t.Errorf("message should mention index_repository, got: %s", item.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected missing_graph_index attention item, got: %v", items)
	}
}

// TestFeatureAttention_NoMissingGraphIndex_ProjectSet verifies AC-013:
// Worktree exists with non-empty GraphProject → no missing_graph_index.
func TestFeatureAttention_NoMissingGraphIndex_ProjectSet(t *testing.T) {
	t.Parallel()
	items := generateFeatureAttention(nil, nil, 1, "FEAT-GI2", "FEAT-GI2", "developing", time.Time{}, true, true, 7, nil, true, "kanbanzai-FEAT-GI2")
	for _, item := range items {
		if item.Type == "missing_graph_index" {
			t.Errorf("unexpected missing_graph_index item when GraphProject is set: %v", item)
		}
	}
}

// TestFeatureAttention_NoMissingGraphIndex_NoWorktree verifies AC-014:
// No worktree → no missing_graph_index.
func TestFeatureAttention_NoMissingGraphIndex_NoWorktree(t *testing.T) {
	t.Parallel()
	items := generateFeatureAttention(nil, nil, 1, "FEAT-GI3", "FEAT-GI3", "developing", time.Time{}, true, true, 7, nil, false, "")
	for _, item := range items {
		if item.Type == "missing_graph_index" {
			t.Errorf("unexpected missing_graph_index item when no worktree exists: %v", item)
		}
	}
}

// TestSynthesiseFeature_MissingGraphIndex_Integration verifies that synthesiseFeature
// populates the missing_graph_index attention item when a worktree exists with empty
// GraphProject (AC-012 end-to-end via synthesise).
func TestSynthesiseFeature_MissingGraphIndex_Integration(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "gi-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "gi-feat", "Graph Index Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "gi-task", "task")

	wtStore := worktree.NewStore(t.TempDir())
	_, err := wtStore.Create(worktree.Record{
		EntityID:     featID,
		Branch:       "feat/gi-feat",
		Path:         ".worktrees/gi-feat",
		Status:       worktree.StatusActive,
		Created:      time.Now().UTC(),
		CreatedBy:    "tester",
		GraphProject: "", // empty — should trigger attention item
	})
	if err != nil {
		t.Fatalf("worktree.Create: %v", err)
	}

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore, "", 7)
	if err != nil {
		t.Fatalf("synthesiseFeature: %v", err)
	}

	found := false
	for _, item := range detail.Attention {
		if item.Type == "missing_graph_index" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_graph_index in attention, got: %v", detail.Attention)
	}
}

// TestSynthesiseFeature_NoMissingGraphIndex_ProjectSet_Integration verifies AC-013
// via synthesiseFeature: worktree with GraphProject set → no missing_graph_index.
func TestSynthesiseFeature_NoMissingGraphIndex_ProjectSet_Integration(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "gi-set-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "gi-set-feat", "Graph Index Set Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "gi-set-task", "task")

	wtStore := worktree.NewStore(t.TempDir())
	_, err := wtStore.Create(worktree.Record{
		EntityID:     featID,
		Branch:       "feat/gi-set-feat",
		Path:         ".worktrees/gi-set-feat",
		Status:       worktree.StatusActive,
		Created:      time.Now().UTC(),
		CreatedBy:    "tester",
		GraphProject: "kanbanzai-FEAT-XXX",
	})
	if err != nil {
		t.Fatalf("worktree.Create: %v", err)
	}

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore, "", 7)
	if err != nil {
		t.Fatalf("synthesiseFeature: %v", err)
	}

	for _, item := range detail.Attention {
		if item.Type == "missing_graph_index" {
			t.Errorf("unexpected missing_graph_index when GraphProject is set: %v", item)
		}
	}
}

// TestSynthesiseFeature_NoMissingGraphIndex_NoWorktree_Integration verifies AC-014
// via synthesiseFeature: no worktree → no missing_graph_index.
func TestSynthesiseFeature_NoMissingGraphIndex_NoWorktree_Integration(t *testing.T) {
	t.Parallel()
	entitySvc, docSvc := setupStatusTest(t)
	planID := createTestPlan(t, entitySvc, "gi-none-plan", "Plan")
	featID := createStatusTestFeature(t, entitySvc, planID, "gi-none-feat", "No WT Feature")
	_ = createStatusTestTask(t, entitySvc, featID, "gi-none-task", "task")

	wtStore := worktree.NewStore(t.TempDir())
	// No worktree created for this feature.

	detail, err := synthesiseFeature(featID, entitySvc, docSvc, wtStore, "", 7)
	if err != nil {
		t.Fatalf("synthesiseFeature: %v", err)
	}

	for _, item := range detail.Attention {
		if item.Type == "missing_graph_index" {
			t.Errorf("unexpected missing_graph_index when no worktree: %v", item)
		}
	}
}

// AC-018 note: The missing_graph_index attention item is inert metadata — it adds a
// string to the attention array without calling codebase_memory_mcp. When the MCP is
// unavailable, the GraphProject field is simply empty and no attention item is emitted,
// so all non-graph behaviour is identical. No separate test is needed beyond AC-014.

// ─── generatePlanAttention tests ─────────────────────────────────────────────

// TestPlanAttention_AllFeaturesDone_Active verifies plan completion detection.
func TestPlanAttention_AllFeaturesDone_Active(t *testing.T) {
	features := []featureSummary{
		{Status: "done"},
		{Status: "cancelled"},
		{Status: "done"},
	}
	items := generatePlanAttention(features, nil, "P13-workflow-flexibility", "active", true, 3)
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "P13-workflow-flexibility") && strings.Contains(item.Message, "all 3 features done") && strings.Contains(item.Message, "ready to close") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected plan completion item, got: %v", items)
	}
}

// TestPlanAttention_PlanAlreadyDone_NoItem verifies that a done plan doesn't get
// the completion item.
func TestPlanAttention_PlanAlreadyDone_NoItem(t *testing.T) {
	features := []featureSummary{{Status: "done"}}
	items := generatePlanAttention(features, nil, "P13", "done", true, 1)
	for _, item := range items {
		if strings.Contains(item.Message, "ready to close") {
			t.Errorf("unexpected completion item for already-done plan: %s", item.Message)
		}
	}
}

// TestPlanAttention_NonFinishedFeature_NoItem verifies that a plan with non-finished
// features doesn't get the completion item.
func TestPlanAttention_NonFinishedFeature_NoItem(t *testing.T) {
	features := []featureSummary{{Status: "done"}, {Status: "developing"}}
	items := generatePlanAttention(features, nil, "P13", "active", false, 2)
	for _, item := range items {
		if strings.Contains(item.Message, "ready to close") {
			t.Errorf("unexpected completion item when features not finished: %s", item.Message)
		}
	}
}

// TestPlanAttention_ZeroFeatures_NoItem verifies that zero features doesn't trigger
// the completion item (even if allFeaturesFinished is somehow true).
func TestPlanAttention_ZeroFeatures_NoItem(t *testing.T) {
	items := generatePlanAttention(nil, nil, "P13", "active", true, 0)
	for _, item := range items {
		if strings.Contains(item.Message, "ready to close") {
			t.Errorf("unexpected completion item for zero features: %s", item.Message)
		}
	}
}

// --- generateProjectAttention unit tests ---

func TestProjectAttention_PlanAllFeaturesDone_NotClosed_Fires(t *testing.T) {
	t.Parallel()
	plans := []planSummary{
		{DisplayID: "P99-test", Status: "reviewing", Features: 3, AllFeaturesFinished: true},
	}
	items := generateProjectAttention(plans, nil, nil, "")
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "P99-test") && strings.Contains(item.Message, "ready to close") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'ready to close' item for P99-test; got: %v", items)
	}
}

func TestProjectAttention_PlanAlreadyDone_NoCloseItem(t *testing.T) {
	t.Parallel()
	plans := []planSummary{
		{DisplayID: "P99-done", Status: "done", Features: 3, AllFeaturesFinished: true},
	}
	items := generateProjectAttention(plans, nil, nil, "")
	for _, item := range items {
		if strings.Contains(item.Message, "ready to close") {
			t.Errorf("plan already done should not produce 'ready to close'; got: %v", items)
		}
	}
}

func TestProjectAttention_PlanNotAllFeaturesDone_NoCloseItem(t *testing.T) {
	t.Parallel()
	plans := []planSummary{
		{DisplayID: "P99-partial", Status: "active", Features: 3, AllFeaturesFinished: false},
	}
	items := generateProjectAttention(plans, nil, nil, "")
	for _, item := range items {
		if strings.Contains(item.Message, "P99-partial") && strings.Contains(item.Message, "ready to close") {
			t.Errorf("plan with unfinished features should not produce 'ready to close'; got: %v", items)
		}
	}
}

func TestProjectAttention_StuckTask_NoDispatchedAt_NotFlagged(t *testing.T) {
	t.Parallel()
	tasks := []service.ListResult{
		{State: map[string]any{"status": "active", "id": "TASK-NODISPATCH"}},
	}
	items := generateProjectAttention(nil, tasks, nil, "")
	for _, item := range items {
		if strings.Contains(item.Message, "TASK-NODISPATCH") {
			t.Errorf("task without dispatched_at should not be flagged; got: %v", items)
		}
	}
}

func TestProjectAttention_StuckTask_RecentDispatch_NotFlagged(t *testing.T) {
	t.Parallel()
	recentDispatch := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	tasks := []service.ListResult{
		{State: map[string]any{
			"status":        "active",
			"id":            "TASK-RECENT",
			"dispatched_at": recentDispatch,
		}},
	}
	items := generateProjectAttention(nil, tasks, nil, "")
	for _, item := range items {
		if strings.Contains(item.Message, "TASK-RECENT") {
			t.Errorf("task dispatched 1h ago should not be flagged as stuck; got: %v", items)
		}
	}
}

func TestProjectAttention_StuckTask_OldDispatch_NoGitBranch_Flagged(t *testing.T) {
	t.Parallel()
	// Dispatch 25 hours ago with no worktree branch — IsTaskStuck returns true.
	oldDispatch := time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339)
	tasks := []service.ListResult{
		{State: map[string]any{
			"status":         "active",
			"id":             "TASK-STUCK01",
			"dispatched_at":  oldDispatch,
			"parent_feature": "FEAT-NOSTUB",
		}},
	}
	// Empty worktreeBranches — branch resolves to ""; checkGitActivitySince returns false.
	items := generateProjectAttention(nil, tasks, map[string]string{}, "")
	found := false
	for _, item := range items {
		if strings.Contains(item.Message, "TASK-STUCK01") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stuck-task attention item for TASK-STUCK01; got: %v", items)
	}
}
