package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── Test setup helpers ───────────────────────────────────────────────────────

// setupEntityToolTest creates the EntityService needed for entity tool tests.
func setupEntityToolTest(t *testing.T) *service.EntityService {
	t.Helper()
	return service.NewEntityService(t.TempDir())
}

// createEntityTestPlan writes a plan record directly (no config.yaml needed).
func createEntityTestPlan(t *testing.T, entitySvc *service.EntityService, slug string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	planID := "P1-" + slug
	record := storage.EntityRecord{
		Type: "plan",
		ID:   planID,
		Slug: slug,
		Fields: map[string]any{
			"id":         planID,
			"slug":       slug,
			"title":      "Test plan " + slug,
			"status":     "proposed",
			"summary":    "Test plan summary",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createEntityTestPlan(%s): %v", slug, err)
	}
	return planID
}

// createEntityTestFeature creates a feature entity for tests.
func createEntityTestFeature(t *testing.T, entitySvc *service.EntityService, planID, slug string) string {
	t.Helper()
	result, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Slug:      slug,
		Parent:    planID,
		Summary:   "Test feature " + slug,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%s): %v", slug, err)
	}
	return result.ID
}

// createEntityTestTask creates a task entity for tests. Returns (ID, slug).
func createEntityTestTask(t *testing.T, entitySvc *service.EntityService, featID, slug string) (string, string) {
	t.Helper()
	result, err := entitySvc.CreateTask(service.CreateTaskInput{
		ParentFeature: featID,
		Slug:          slug,
		Summary:       "Test task " + slug,
	})
	if err != nil {
		t.Fatalf("CreateTask(%s): %v", slug, err)
	}
	return result.ID, result.Slug
}

// callEntityTool invokes the entity tool and returns the raw text response.
func callEntityTool(t *testing.T, entitySvc *service.EntityService, args map[string]any) string {
	t.Helper()
	tool := entityTool(entitySvc, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("entity handler error: %v", err)
	}
	return extractText(t, result)
}

// callEntityToolWithDocSvc invokes the entity tool with a DocumentService and returns raw text.
func callEntityToolWithDocSvc(t *testing.T, entitySvc *service.EntityService, docSvc *service.DocumentService, args map[string]any) string {
	t.Helper()
	tool := entityTool(entitySvc, docSvc)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("entity handler error: %v", err)
	}
	return extractText(t, result)
}

// callEntityToolWithDocSvcJSON invokes the entity tool with a DocumentService and parses the result as JSON.
func callEntityToolWithDocSvcJSON(t *testing.T, entitySvc *service.EntityService, docSvc *service.DocumentService, args map[string]any) map[string]any {
	t.Helper()
	text := callEntityToolWithDocSvc(t, entitySvc, docSvc, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse entity result: %v\nraw: %s", err, text)
	}
	return parsed
}

// callEntityToolJSON invokes the entity tool and parses the result as JSON.
func callEntityToolJSON(t *testing.T, entitySvc *service.EntityService, args map[string]any) map[string]any {
	t.Helper()
	text := callEntityTool(t, entitySvc, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse entity result: %v\nraw: %s", err, text)
	}
	return parsed
}

// ─── create action ────────────────────────────────────────────────────────────

func TestEntity_Create_Task(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-ct1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-ct1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":         "create",
		"type":           "task",
		"parent_feature": featID,
		"slug":           "new-task",
		"summary":        "A new task",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object in response, got: %v", result)
	}
	if entity["type"] != "task" {
		t.Errorf("entity.type = %v, want task", entity["type"])
	}
	if entity["slug"] != "new-task" {
		t.Errorf("entity.slug = %v, want new-task", entity["slug"])
	}
	if entity["status"] != "queued" {
		t.Errorf("entity.status = %v, want queued", entity["status"])
	}
	if entity["id"] == nil || entity["id"] == "" {
		t.Error("entity.id should be set")
	}
	if entity["display_id"] == nil {
		t.Error("entity.display_id should be set")
	}
}

func TestEntity_Create_Feature(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-cf1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "create",
		"type":    "feature",
		"slug":    "new-feature",
		"parent":  planID,
		"summary": "A new feature",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object in response, got: %v", result)
	}
	if entity["type"] != "feature" {
		t.Errorf("entity.type = %v, want feature", entity["type"])
	}
	if entity["status"] != "proposed" {
		t.Errorf("entity.status = %v, want proposed", entity["status"])
	}
}

func TestEntity_Create_Bug(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":      "create",
		"type":        "bug",
		"slug":        "crash-on-login",
		"title":       "App crashes on login",
		"reported_by": "user@example.com",
		"observed":    "App crashes",
		"expected":    "Should log in",
		"severity":    "high",
		"priority":    "high",
		"bug_type":    "implementation-defect",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["type"] != "bug" {
		t.Errorf("entity.type = %v, want bug", entity["type"])
	}
}

func TestEntity_Create_Epic(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "create",
		"type":    "epic",
		"slug":    "big-initiative",
		"title":   "Big Initiative",
		"summary": "A large-scale effort",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["type"] != "epic" {
		t.Errorf("entity.type = %v, want epic", entity["type"])
	}
}

func TestEntity_Create_Decision(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":    "create",
		"type":      "decision",
		"slug":      "use-postgres",
		"summary":   "Use PostgreSQL for primary storage",
		"rationale": "Better support for complex queries and ACID compliance",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["type"] != "decision" {
		t.Errorf("entity.type = %v, want decision", entity["type"])
	}
}

func TestEntity_Create_MissingType(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "create",
		"slug":    "something",
		"summary": "Something",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing type, got: %v", result)
	}
}

func TestEntity_Create_UnknownType(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "create",
		"type":    "wombat",
		"slug":    "something",
		"summary": "Something",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for unknown type, got: %v", result)
	}
}

func TestEntity_Create_BatchTasks(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-cb1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-cb1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "create",
		"type":   "task",
		"entities": []any{
			map[string]any{
				"parent_feature": featID,
				"slug":           "batch-task-1",
				"summary":        "First batch task",
			},
			map[string]any{
				"parent_feature": featID,
				"slug":           "batch-task-2",
				"summary":        "Second batch task",
			},
		},
	})

	// Batch result has "results" and "summary" fields.
	summary, ok := result["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected batch summary in response, got: %v", result)
	}
	if summary["total"].(float64) != 2 {
		t.Errorf("summary.total = %v, want 2", summary["total"])
	}
	if summary["succeeded"].(float64) != 2 {
		t.Errorf("summary.succeeded = %v, want 2", summary["succeeded"])
	}
}

// ─── mutation always includes side_effects: [] ────────────────────────────────

func TestEntity_Create_MutationHasSideEffectsField(t *testing.T) {
	// Verifies §8.4 + §30.2: create (mutation) always includes side_effects: []
	// in the response, even when no cascades occurred.
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-mse1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mse1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":         "create",
		"type":           "task",
		"parent_feature": featID,
		"slug":           "mse-task",
		"summary":        "Mutation side effects test task",
	})

	// side_effects: [] must be present for all mutations (spec §8.4).
	sideEffects, ok := result["side_effects"]
	if !ok {
		t.Fatal("side_effects missing from create (mutation) response — spec §8.4 requires it")
	}
	arr, _ := sideEffects.([]any)
	if len(arr) != 0 {
		t.Errorf("expected side_effects: [], got %v", sideEffects)
	}
}

// ─── get action ───────────────────────────────────────────────────────────────

func TestEntity_Get_Task(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-gt1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-gt1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-gt1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
		"id":     taskID,
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["id"] != taskID {
		t.Errorf("entity.id = %v, want %v", entity["id"], taskID)
	}
	if entity["type"] != "task" {
		t.Errorf("entity.type = %v, want task", entity["type"])
	}
}

func TestEntity_Get_Feature(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-gf1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-gf1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
		"id":     featID,
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["id"] != featID {
		t.Errorf("entity.id = %v, want %v", entity["id"], featID)
	}
	if entity["type"] != "feature" {
		t.Errorf("entity.type = %v, want feature", entity["type"])
	}
}

func TestEntity_Get_Plan(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-gp1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
		"id":     planID,
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["id"] != planID {
		t.Errorf("entity.id = %v, want %v", entity["id"], planID)
	}
}

func TestEntity_Get_NotFound(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
		"id":     "FEAT-01ZZZZZZZZZZ",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for not-found entity, got: %v", result)
	}
}

func TestEntity_Get_MissingID(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing ID, got: %v", result)
	}
}

func TestEntity_Get_UnknownIDFormat(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "get",
		"id":     "UNKNOWN-123",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for unrecognised ID format, got: %v", result)
	}
}

// ─── list action ─────────────────────────────────────────────────────────────

func TestEntity_List_Tasks(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-lt1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-lt1")
	createEntityTestTask(t, entitySvc, featID, "task-lt1")
	createEntityTestTask(t, entitySvc, featID, "task-lt2")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "task",
	})

	entities, _ := result["entities"].([]any)
	if len(entities) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(entities))
	}
	total, _ := result["total"].(float64)
	if int(total) != 2 {
		t.Errorf("total = %v, want 2", total)
	}
}

func TestEntity_List_TasksFilteredByParent(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-lp1")
	featID1 := createEntityTestFeature(t, entitySvc, planID, "feat-lp1a")
	featID2 := createEntityTestFeature(t, entitySvc, planID, "feat-lp1b")

	createEntityTestTask(t, entitySvc, featID1, "task-lp1a1")
	createEntityTestTask(t, entitySvc, featID1, "task-lp1a2")
	createEntityTestTask(t, entitySvc, featID2, "task-lp1b1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "task",
		"parent": featID1,
	})

	entities, _ := result["entities"].([]any)
	if len(entities) != 2 {
		t.Errorf("expected 2 tasks in feat1, got %d", len(entities))
	}
}

func TestEntity_List_FeaturesFilteredByParent(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID1 := createEntityTestPlan(t, entitySvc, "ent-lfp1")
	planID2 := createEntityTestPlan(t, entitySvc, "ent-lfp2")

	createEntityTestFeature(t, entitySvc, planID1, "feat-lfp1a")
	createEntityTestFeature(t, entitySvc, planID1, "feat-lfp1b")
	createEntityTestFeature(t, entitySvc, planID2, "feat-lfp2a")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "feature",
		"parent": planID1,
	})

	entities, _ := result["entities"].([]any)
	if len(entities) != 2 {
		t.Errorf("expected 2 features in plan1, got %d", len(entities))
	}
}

func TestEntity_List_FilterByStatus(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-ls1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-ls1")

	taskID1, taskSlug1 := createEntityTestTask(t, entitySvc, featID, "task-ls1a")
	createEntityTestTask(t, entitySvc, featID, "task-ls1b")

	// Advance one task to ready.
	if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type:   "task",
		ID:     taskID1,
		Slug:   taskSlug1,
		Status: "ready",
	}); err != nil {
		t.Fatalf("advance to ready: %v", err)
	}

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "task",
		"status": "ready",
	})

	entities, _ := result["entities"].([]any)
	if len(entities) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(entities))
	}
}

func TestEntity_List_Plans(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	createEntityTestPlan(t, entitySvc, "ent-lpl1")
	createEntityTestPlan(t, entitySvc, "ent-lpl2")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "plan",
	})

	entities, _ := result["entities"].([]any)
	if len(entities) != 2 {
		t.Errorf("expected 2 plans, got %d", len(entities))
	}
}

func TestEntity_List_MissingType(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing type, got: %v", result)
	}
}

func TestEntity_List_SummaryFields(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-lsf1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-lsf1")
	createEntityTestTask(t, entitySvc, featID, "task-lsf1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "list",
		"type":   "task",
	})

	entities, _ := result["entities"].([]any)
	if len(entities) == 0 {
		t.Fatal("expected at least one entity")
	}
	item, _ := entities[0].(map[string]any)
	for _, field := range []string{"id", "type", "slug", "status", "display_id"} {
		if item[field] == nil || item[field] == "" {
			t.Errorf("list item missing field %q", field)
		}
	}
}

// ─── update action ────────────────────────────────────────────────────────────

func TestEntity_Update_TaskSummary(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-ut1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-ut1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-ut1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "update",
		"id":      taskID,
		"summary": "Updated task summary",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["summary"] != "Updated task summary" {
		t.Errorf("entity.summary = %v, want 'Updated task summary'", entity["summary"])
	}
}

func TestEntity_Update_MissingID(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "update",
		"summary": "Updated summary",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing ID, got: %v", result)
	}
}

func TestEntity_Update_IgnoresStatusAndIDChanges(t *testing.T) {
	// Verifies §14.6 + §30.8: update cannot change id or status.
	// The implementation silently ignores these fields — status stays unchanged.
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-uig1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-uig1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-uig1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "update",
		"id":      taskID,
		"status":  "done", // must be silently ignored
		"summary": "Updated summary",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' in response, got: %v", result)
	}
	// Status must not have changed — update cannot change lifecycle status.
	if entity["status"] == "done" {
		t.Error("update changed entity status to 'done' — must not happen; use transition instead")
	}
	if entity["status"] != "queued" {
		t.Errorf("entity.status = %v, want queued (unchanged)", entity["status"])
	}
	// Summary should have updated.
	if entity["summary"] != "Updated summary" {
		t.Errorf("entity.summary = %v, want 'Updated summary'", entity["summary"])
	}
}

func TestEntity_Update_TaskDependsOn(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-ud1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-ud1")
	taskID1, _ := createEntityTestTask(t, entitySvc, featID, "task-ud1a")
	taskID2, _ := createEntityTestTask(t, entitySvc, featID, "task-ud1b")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":     "update",
		"id":         taskID2,
		"depends_on": []any{taskID1},
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}

	deps, ok := entity["depends_on"].([]any)
	if !ok {
		t.Fatalf("expected depends_on to be a list, got: %T (%v)", entity["depends_on"], entity["depends_on"])
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d: %v", len(deps), deps)
	}
	if deps[0] != taskID1 {
		t.Errorf("depends_on[0] = %v, want %s", deps[0], taskID1)
	}
}

func TestEntity_Update_DependsOnRejectsNonTask(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-udn1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-udn1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":     "update",
		"id":         featID,
		"depends_on": []any{"TASK-01ZZZZZZZZZZ1"},
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for depends_on on feature, got: %v", result)
	}
}

func TestEntity_Update_DependsOnRejectsInvalidID(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-udi1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-udi1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-udi1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":     "update",
		"id":         taskID,
		"depends_on": []any{"FEAT-01ZZZZZZZZZZ1"},
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for non-TASK ID in depends_on, got: %v", result)
	}
}

// ─── transition action ────────────────────────────────────────────────────────

func TestEntity_Transition_TaskQueuedToReady(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-tr1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-tr1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-tr1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "ready",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["status"] != "ready" {
		t.Errorf("entity.status = %v, want ready", entity["status"])
	}
}

func TestEntity_Transition_FeatureToDesigning(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-trf1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-trf1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "designing",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["status"] != "designing" {
		t.Errorf("entity.status = %v, want designing", entity["status"])
	}
}

func TestEntity_Transition_InvalidTransition(t *testing.T) {
	// Verifies §14.7 + §30.8 + H.17: invalid transitions return an error naming
	// the current status, the requested status, and the valid transitions.
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-ti1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-ti1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-ti1")

	// queued → done is not a valid transition.
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "done",
	})

	errField, hasErr := result["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("expected error for invalid transition, got: %v", result)
	}

	// Error must use the structured invalid_transition code.
	if errField["code"] != "invalid_transition" {
		t.Errorf("error.code = %v, want invalid_transition", errField["code"])
	}

	// Details must include current_status, requested_status, and valid_transitions
	// so agents can correct the call without guessing (spec §14.7).
	details, _ := errField["details"].(map[string]any)
	if details == nil {
		t.Fatal("error.details missing from invalid_transition error")
	}
	if details["current_status"] != "queued" {
		t.Errorf("details.current_status = %v, want queued", details["current_status"])
	}
	if details["requested_status"] != "done" {
		t.Errorf("details.requested_status = %v, want done", details["requested_status"])
	}
	validTransitions, _ := details["valid_transitions"].([]any)
	if len(validTransitions) == 0 {
		t.Error("details.valid_transitions missing or empty — agents need this to correct the request")
	}
}

func TestEntity_Transition_MissingStatus(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-tm1")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-tm1")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-tm1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     taskID,
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing status, got: %v", result)
	}
}

func TestEntity_Transition_PlanStatus(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "ent-tpl1")

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     planID,
		"status": "designing",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["status"] != "designing" {
		t.Errorf("entity.status = %v, want designing", entity["status"])
	}
}

// ─── missing action ───────────────────────────────────────────────────────────

func TestEntity_MissingAction(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"type": "task",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for missing action, got: %v", result)
	}
}

func TestEntity_UnknownAction(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "delete",
		"id":     "TASK-01JX123",
	})

	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for unknown action, got: %v", result)
	}
}

// ─── type inference ───────────────────────────────────────────────────────────

func TestEntityInferType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id       string
		wantType string
		wantOK   bool
	}{
		{"FEAT-01JX123", "feature", true},
		{"feat-01JX123", "feature", true},
		{"TASK-01JX123", "task", true},
		{"task-01JX123", "task", true},
		{"T-01JX123", "task", true},
		{"t-01JX123", "task", true},
		{"BUG-01JX123", "bug", true},
		{"bug-01JX123", "bug", true},
		{"EPIC-MYSLUG-xyz", "epic", true},
		{"DEC-01JX123", "decision", true},
		{"INC-01JX123", "incident", true},
		{"P1-my-plan", "plan", true},
		{"P2-another-plan", "plan", true},
		{"", "", false},
		{"random-string", "", false},
		{"UNKNOWN-123", "", false},
	}

	for _, tt := range tests {
		gotType, gotOK := entityInferType(tt.id)
		if gotOK != tt.wantOK {
			t.Errorf("entityInferType(%q).ok = %v, want %v", tt.id, gotOK, tt.wantOK)
		}
		if gotType != tt.wantType {
			t.Errorf("entityInferType(%q).type = %q, want %q", tt.id, gotType, tt.wantType)
		}
	}
}

// ─── entityFullRecord ─────────────────────────────────────────────────────────

func TestEntityFullRecord(t *testing.T) {
	t.Parallel()
	state := map[string]any{
		"status":  "proposed",
		"summary": "Test summary",
	}
	record := entityFullRecord("FEAT-01JX123", "feature", "my-feature", state)

	if record["id"] != "FEAT-01JX123" {
		t.Errorf("record.id = %v", record["id"])
	}
	if record["type"] != "feature" {
		t.Errorf("record.type = %v", record["type"])
	}
	if record["slug"] != "my-feature" {
		t.Errorf("record.slug = %v", record["slug"])
	}
	if record["display_id"] == nil || record["display_id"] == "" {
		t.Error("record.display_id should be set")
	}
	// Original state fields preserved.
	if record["status"] != "proposed" {
		t.Errorf("record.status = %v, want proposed", record["status"])
	}
	if record["summary"] != "Test summary" {
		t.Errorf("record.summary = %v, want 'Test summary'", record["summary"])
	}
}

// ─── entityHasAnyTag ─────────────────────────────────────────────────────────

func TestEntityHasAnyTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		state      map[string]any
		filterTags []string
		want       bool
	}{
		{
			name:       "match string slice",
			state:      map[string]any{"tags": []string{"alpha", "beta"}},
			filterTags: []string{"beta"},
			want:       true,
		},
		{
			name:       "match any slice",
			state:      map[string]any{"tags": []any{"alpha", "beta"}},
			filterTags: []string{"alpha"},
			want:       true,
		},
		{
			name:       "no match",
			state:      map[string]any{"tags": []any{"alpha"}},
			filterTags: []string{"gamma"},
			want:       false,
		},
		{
			name:       "empty state tags",
			state:      map[string]any{},
			filterTags: []string{"alpha"},
			want:       false,
		},
		{
			name:       "empty filter tags",
			state:      map[string]any{"tags": []any{"alpha"}},
			filterTags: []string{},
			want:       false,
		},
		{
			name:       "case insensitive",
			state:      map[string]any{"tags": []any{"Alpha"}},
			filterTags: []string{"alpha"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entityHasAnyTag(tt.state, tt.filterTags)
			if got != tt.want {
				t.Errorf("entityHasAnyTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── entityArgStringSlice ────────────────────────────────────────────────────

func TestEntityArgStringSlice(t *testing.T) {
	t.Parallel()
	args := map[string]any{
		"tags": []any{"alpha", "beta", "gamma"},
	}
	got := entityArgStringSlice(args, "tags")
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(got))
	}
	if strings.Join(got, ",") != "alpha,beta,gamma" {
		t.Errorf("got %v", got)
	}
}

func TestEntityArgStringSlice_Missing(t *testing.T) {
	t.Parallel()
	got := entityArgStringSlice(map[string]any{}, "tags")
	if got != nil {
		t.Errorf("expected nil for missing key, got %v", got)
	}
}

// ─── plan creation ────────────────────────────────────────────────────────────

// ─── Advance transition tests ─────────────────────────────────────────────────

func TestEntity_Transition_AdvanceFeature_HappyPath(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "adv-hp")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-hp")

	// Submit and approve a design document so the designing gate is satisfied.
	setupApprovedDoc(t, docSvc, repoRoot, "work/design/adv-design.md", "design", featID)

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      featID,
		"status":  "specifying",
		"advance": true,
	})

	status, _ := result["status"].(string)
	if status != "specifying" {
		t.Errorf("status = %q, want specifying", status)
	}

	advThrough, _ := result["advanced_through"].([]any)
	if len(advThrough) < 1 {
		t.Fatalf("expected advanced_through to have at least 1 entry, got %v", advThrough)
	}
	if advThrough[0] != "designing" {
		t.Errorf("advanced_through[0] = %v, want designing", advThrough[0])
	}

	msg, _ := result["message"].(string)
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestEntity_Transition_AdvanceFeature_StopsAtGate(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "adv-stop")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-stop")

	// No documents approved — advance should stop at the first gate (designing).
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      featID,
		"status":  "developing",
		"advance": true,
	})

	status, _ := result["status"].(string)
	if status != "proposed" {
		t.Errorf("status = %q, want proposed (should not advance without docs)", status)
	}

	stoppedReason, _ := result["stopped_reason"].(string)
	if stoppedReason == "" {
		t.Error("expected non-empty stopped_reason when gate blocks advance")
	}

	msg, _ := result["message"].(string)
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestEntity_Transition_AdvanceNonFeature_Error(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "adv-nf")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-nf")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-adv-nf")

	// WithSideEffects wraps errors into a JSON error response (no Go error returned).
	text := callEntityToolWithDocSvc(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      taskID,
		"status":  "ready",
		"advance": true,
	})
	if !strings.Contains(text, "advance is only supported for feature entities") {
		t.Errorf("expected error about advance not supported, got: %s", text)
	}
}

func TestEntity_Transition_AdvanceFalse_NormalBehaviour(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "adv-false")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-false")

	// advance=false should behave exactly like a normal transition.
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "transition",
		"id":      featID,
		"status":  "designing",
		"advance": false,
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["status"] != "designing" {
		t.Errorf("entity.status = %v, want designing", entity["status"])
	}
}

func TestEntity_Transition_AdvanceNotProvided_NormalBehaviour(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)

	planID := createEntityTestPlan(t, entitySvc, "adv-omit")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-omit")

	// No advance parameter at all — normal single-step transition.
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "designing",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' object, got: %v", result)
	}
	if entity["status"] != "designing" {
		t.Errorf("entity.status = %v, want designing", entity["status"])
	}
}

// setupApprovedDoc creates a document file, submits it, and approves it.
func setupApprovedDoc(t *testing.T, docSvc *service.DocumentService, repoRoot, path, docType, owner string) string {
	t.Helper()
	return submitAndApproveTestDoc(t, docSvc, repoRoot, path, docType, owner, true)
}

// submitAndApproveTestDoc creates a doc file, submits, and optionally approves it.
func submitAndApproveTestDoc(t *testing.T, docSvc *service.DocumentService, repoRoot, relPath, docType, owner string, approve bool) string {
	t.Helper()

	// Create the document file on disk.
	fullPath := repoRoot + "/" + relPath
	dir := fullPath[:strings.LastIndex(fullPath, "/")]

	if err := mkdirAll(dir); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := writeFile(fullPath, "# Test Document\n\nContent for testing.\n"); err != nil {
		t.Fatalf("write %s: %v", fullPath, err)
	}

	rec, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      relPath,
		Type:      docType,
		Title:     "Test " + docType,
		Owner:     owner,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit doc: %v", err)
	}

	if approve {
		if _, err := docSvc.ApproveDocument(service.ApproveDocumentInput{ID: rec.ID, ApprovedBy: "tester"}); err != nil {
			t.Fatalf("approve doc: %v", err)
		}
	}

	return rec.ID
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestEntity_Create_Plan(t *testing.T) {
	// Plan creation requires a valid prefix registered in .kbz/config.yaml.
	// This test is skipped in CI / fresh checkouts that lack a config file.
	// Verifies §30.8: entity(action: "create", type: "plan", ...) creates a plan.
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("skipping plan creation test: config not available: %v", err)
	}
	testPrefix := "P"
	if !cfg.IsActivePrefix(testPrefix) {
		t.Skipf("skipping plan creation test: prefix %q not active in config", testPrefix)
	}

	entitySvc := setupEntityToolTest(t)

	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action":  "create",
		"type":    "plan",
		"prefix":  testPrefix,
		"slug":    "entity-tool-test-plan",
		"title":   "Entity Tool Test Plan",
		"summary": "A plan created via the entity tool to verify routing",
	})

	entity, ok := result["entity"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entity' in response, got: %v", result)
	}
	if entity["type"] != "plan" {
		t.Errorf("entity.type = %v, want plan", entity["type"])
	}
	if entity["status"] != "proposed" {
		t.Errorf("entity.status = %v, want proposed", entity["status"])
	}
	// side_effects: [] must be present in mutation responses (spec §8.4).
	sideEffects, ok := result["side_effects"]
	if !ok {
		t.Fatal("side_effects missing from plan create (mutation) response")
	}
	arr, _ := sideEffects.([]any)
	if len(arr) != 0 {
		t.Errorf("expected empty side_effects, got %v", arr)
	}
}
