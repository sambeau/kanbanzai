package service

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

const testPlanIDConflict = "P1-conflict-plan"

// setupConflictTest creates an EntityService and ConflictService with a plan
// and feature ready for task creation. Returns the services and the feature ID.
func setupConflictTest(t *testing.T) (*ConflictService, *EntityService, string) {
	t.Helper()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	writeConflictTestPlan(t, entitySvc, testPlanIDConflict)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "conflict-feature",
		Parent:    testPlanIDConflict,
		Summary:   "Feature for conflict tests",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	return conflictSvc, entitySvc, featResult.ID
}

func writeConflictTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"title":      "Conflict Test Plan",
		"status":     "active",
		"summary":    "Test plan for conflict domain tests",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-03-19T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeConflictTestPlan(%s): %v", id, err)
	}
}

// createConflictTask creates a task under featureID and then overwrites its
// record to include files_planned and depends_on fields.
func createConflictTask(t *testing.T, svc *EntityService, featureID, slug, summary string, filesPlanned, dependsOn []string) string {
	t.Helper()

	result, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: featureID,
		Slug:          slug,
		Summary:       summary,
	})
	if err != nil {
		t.Fatalf("create task %s: %v", slug, err)
	}

	if len(filesPlanned) > 0 || len(dependsOn) > 0 {
		// Load the record, add the slice fields, write it back.
		record, err := svc.store.Load("task", result.ID, result.Slug)
		if err != nil {
			t.Fatalf("load task %s for update: %v", result.ID, err)
		}
		if len(filesPlanned) > 0 {
			record.Fields["files_planned"] = filesPlanned
		}
		if len(dependsOn) > 0 {
			record.Fields["depends_on"] = dependsOn
		}
		if _, err := svc.store.Write(record); err != nil {
			t.Fatalf("write task %s with extra fields: %v", result.ID, err)
		}
	}

	return result.ID
}

func TestConflictCheck_RequiresAtLeastTwoTasks(t *testing.T) {
	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	tests := []struct {
		name    string
		taskIDs []string
	}{
		{"zero tasks", nil},
		{"one task", []string{"TASK-FAKE123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: tt.taskIDs})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "at least two task IDs") {
				t.Fatalf("expected error about at least two task IDs, got: %v", err)
			}
		})
	}
}

func TestConflictCheck_FileOverlapDetected(t *testing.T) {
	conflictSvc, entitySvc, featureID := setupConflictTest(t)

	sharedFile := "internal/auth/handler.go"
	taskA := createConflictTask(t, entitySvc, featureID,
		"task-alpha", "Implement auth handler",
		[]string{sharedFile, "internal/auth/middleware.go"}, nil)
	taskB := createConflictTask(t, entitySvc, featureID,
		"task-beta", "Add rate limiting to handler",
		[]string{sharedFile, "internal/ratelimit/limiter.go"}, nil)

	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	pair := result.Pairs[0]

	if pair.Dimensions.FileOverlap.Risk != "medium" {
		t.Errorf("file overlap risk: got %q, want %q", pair.Dimensions.FileOverlap.Risk, "medium")
	}

	found := false
	for _, f := range pair.Dimensions.FileOverlap.SharedFiles {
		if f == sharedFile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("shared files %v does not contain %q", pair.Dimensions.FileOverlap.SharedFiles, sharedFile)
	}
}

func TestConflictCheck_DependencyOrderDetected(t *testing.T) {
	conflictSvc, entitySvc, featureID := setupConflictTest(t)

	taskB := createConflictTask(t, entitySvc, featureID,
		"task-dep-base", "Build database schema",
		nil, nil)
	taskA := createConflictTask(t, entitySvc, featureID,
		"task-dep-child", "Build API on top of schema",
		nil, []string{taskB})

	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	pair := result.Pairs[0]

	if pair.Dimensions.DependencyOrder.Risk != "high" {
		t.Errorf("dependency order risk: got %q, want %q", pair.Dimensions.DependencyOrder.Risk, "high")
	}
	if !strings.Contains(pair.Dimensions.DependencyOrder.Detail, taskA) ||
		!strings.Contains(pair.Dimensions.DependencyOrder.Detail, taskB) {
		t.Errorf("dependency detail should mention both task IDs, got: %s", pair.Dimensions.DependencyOrder.Detail)
	}
}

func TestConflictCheck_SerialiseRecommendation(t *testing.T) {
	conflictSvc, entitySvc, featureID := setupConflictTest(t)

	taskB := createConflictTask(t, entitySvc, featureID,
		"task-serial-base", "Create base module",
		nil, nil)
	taskA := createConflictTask(t, entitySvc, featureID,
		"task-serial-child", "Extend base module",
		nil, []string{taskB})

	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	if result.Pairs[0].Recommendation != "serialise" {
		t.Errorf("recommendation: got %q, want %q", result.Pairs[0].Recommendation, "serialise")
	}
}

func TestConflictCheck_NoConflict(t *testing.T) {
	conflictSvc, entitySvc, _ := setupConflictTest(t)

	// Use a feature whose slug tokens are all < 3 chars so
	// extractConflictKeywords ignores them, avoiding false boundary crossing.
	feat, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "nc",
		Parent:    testPlanIDConflict,
		Summary:   "no",
		CreatedBy: "t",
	})
	if err != nil {
		t.Fatalf("create no-conflict feature: %v", err)
	}
	ncFeat := feat.ID

	taskA := createConflictTask(t, entitySvc, ncFeat,
		"logging-cfg", "Configure stdout format",
		[]string{"internal/logging/logger.go"}, nil)
	taskB := createConflictTask(t, entitySvc, ncFeat,
		"db-migration", "Run schema upgrade scripts",
		[]string{"internal/database/migrate.go"}, nil)

	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if result.OverallRisk != "none" {
		t.Errorf("overall risk: got %q, want %q", result.OverallRisk, "none")
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	if result.Pairs[0].Recommendation != "safe_to_parallelise" {
		t.Errorf("recommendation: got %q, want %q", result.Pairs[0].Recommendation, "safe_to_parallelise")
	}
}

func TestConflictCheck_BoundaryCrossing(t *testing.T) {
	conflictSvc, entitySvc, featureID := setupConflictTest(t)

	// Both tasks share enough keywords (auth, token, validate) for medium boundary crossing.
	taskA := createConflictTask(t, entitySvc, featureID,
		"auth-token-validate", "Validate auth token signature",
		nil, nil)
	taskB := createConflictTask(t, entitySvc, featureID,
		"auth-token-refresh", "Refresh auth token validate expiry",
		nil, nil)

	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	pair := result.Pairs[0]

	bcRisk := pair.Dimensions.BoundaryCrossing.Risk
	if bcRisk == "none" {
		t.Error("boundary crossing risk should not be none for tasks with shared keywords")
	}
	if pair.Dimensions.BoundaryCrossing.Detail == "" {
		t.Error("boundary crossing detail should not be empty")
	}
	if !strings.Contains(pair.Dimensions.BoundaryCrossing.Detail, "shared terms") {
		t.Errorf("boundary crossing detail should mention shared terms, got: %s", pair.Dimensions.BoundaryCrossing.Detail)
	}
}

func TestConflictCheck_TransitiveDependency(t *testing.T) {
	conflictSvc, entitySvc, featureID := setupConflictTest(t)

	// C has no dependencies; B depends on C; A depends on B.
	// So A → B → C is a transitive chain: checking A vs C should detect it.
	taskC := createConflictTask(t, entitySvc, featureID,
		"task-trans-base", "Build foundation layer",
		nil, nil)
	taskB := createConflictTask(t, entitySvc, featureID,
		"task-trans-middle", "Build service layer",
		nil, []string{taskC})
	taskA := createConflictTask(t, entitySvc, featureID,
		"task-trans-top", "Build presentation layer",
		nil, []string{taskB})

	// All three tasks must be in the input so the transitive walk can follow
	// A → B → C through the allTasks map.
	result, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: []string{taskA, taskB, taskC}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	// Find the A-vs-C pair.
	var acPair *ConflictPairResult
	for i := range result.Pairs {
		p := &result.Pairs[i]
		if (p.TaskA == taskA && p.TaskB == taskC) || (p.TaskA == taskC && p.TaskB == taskA) {
			acPair = p
			break
		}
	}
	if acPair == nil {
		t.Fatal("pair for task A vs task C not found in results")
	}

	if acPair.Dimensions.DependencyOrder.Risk != "medium" {
		t.Errorf("transitive dependency risk: got %q, want %q", acPair.Dimensions.DependencyOrder.Risk, "medium")
	}
	if !strings.Contains(acPair.Dimensions.DependencyOrder.Detail, "transitive") {
		t.Errorf("transitive dependency detail should mention 'transitive', got: %s", acPair.Dimensions.DependencyOrder.Detail)
	}

	// A→B is direct, so that pair should be high risk.
	var abPair *ConflictPairResult
	for i := range result.Pairs {
		p := &result.Pairs[i]
		if (p.TaskA == taskA && p.TaskB == taskB) || (p.TaskA == taskB && p.TaskB == taskA) {
			abPair = p
			break
		}
	}
	if abPair == nil {
		t.Fatal("pair for task A vs task B not found in results")
	}
	if abPair.Dimensions.DependencyOrder.Risk != "high" {
		t.Errorf("direct dependency (A→B) risk: got %q, want %q", abPair.Dimensions.DependencyOrder.Risk, "high")
	}
}
