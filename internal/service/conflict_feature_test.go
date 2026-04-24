package service

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// testPlanIDConflictFeat is the plan used by conflict_feature_test.go tests.
const testPlanIDConflictFeat = "P1-cfeat-plan"

// newConflictFeatureEnv sets up a fresh EntityService backed by a temp dir,
// writes a test plan, and returns the EntityService and plan ID.
func newConflictFeatureEnv(t *testing.T) (*EntityService, string) {
	t.Helper()
	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)
	writeConflictTestPlan(t, entitySvc, testPlanIDConflictFeat)
	return entitySvc, testPlanIDConflictFeat
}

// newFeature creates a feature under planID and returns its ID.
func newFeature(t *testing.T, entitySvc *EntityService, planID, slug string) string {
	t.Helper()
	result, err := entitySvc.CreateFeature(CreateFeatureInput{
		Name:      "test " + slug,
		Slug:      slug,
		Parent:    planID,
		Summary:   "Feature " + slug,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature %s: %v", slug, err)
	}
	return result.ID
}

// AC-001: Both task_ids and feature_ids supplied → error with "mutually exclusive" message.
func TestConflictCheck_MutuallyExclusive(t *testing.T) {
	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	_, err := conflictSvc.Check(ConflictCheckInput{
		TaskIDs:    []string{"TASK-FAKE001"},
		FeatureIDs: []string{"FEAT-FAKE001"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected 'mutually exclusive' in error message, got: %v", err)
	}
}

// AC-002: Only feature_ids with 2+ valid features → returns FeatureConflictResult, no error.
func TestConflictFeatures_TwoValidFeatures(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	featA := newFeature(t, entitySvc, planID, "ac002-alpha")
	featB := newFeature(t, entitySvc, planID, "ac002-beta")

	createConflictTask(t, entitySvc, featA, "ac002-task-a", "Task for alpha", []string{"internal/foo/bar.go"}, nil)
	createConflictTask(t, entitySvc, featB, "ac002-task-b", "Task for beta", []string{"internal/baz/qux.go"}, nil)

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	if len(result.FeatureIDs) != 2 {
		t.Errorf("expected 2 FeatureIDs, got %d", len(result.FeatureIDs))
	}
	if len(result.Pairs) != 1 {
		t.Errorf("expected 1 pair, got %d", len(result.Pairs))
	}
	if len(result.Features) != 2 {
		t.Errorf("expected 2 feature infos, got %d", len(result.Features))
	}
}

// AC-003: Two features with overlapping files_planned → pair risk is not safe_to_parallelise.
func TestConflictFeatures_OverlappingFiles(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	featA := newFeature(t, entitySvc, planID, "ac003-a")
	featB := newFeature(t, entitySvc, planID, "ac003-b")

	shared := "internal/shared/handler.go"
	createConflictTask(t, entitySvc, featA, "ac003-task-a", "ac003 alpha task", []string{shared, "internal/a/other.go"}, nil)
	createConflictTask(t, entitySvc, featB, "ac003-task-b", "ac003 beta task", []string{shared, "internal/b/other.go"}, nil)

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	pair := result.Pairs[0]
	if pair.Recommendation == "safe_to_parallelise" {
		t.Errorf("expected non-safe recommendation for overlapping files, got: %s", pair.Recommendation)
	}
	if len(pair.Dimensions.FileOverlap.SharedFiles) == 0 {
		t.Error("expected shared files to be reported in dimensions")
	}
	found := false
	for _, f := range pair.Dimensions.FileOverlap.SharedFiles {
		if f == shared {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("shared file %q not found in SharedFiles: %v", shared, pair.Dimensions.FileOverlap.SharedFiles)
	}
}

// AC-004: Two features with no overlapping files_planned → pair risk is safe_to_parallelise.
func TestConflictFeatures_NoOverlappingFiles(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	// Use short slugs/summaries so extractConflictKeywords finds no shared terms.
	featA := newFeature(t, entitySvc, planID, "nc-x")
	featB := newFeature(t, entitySvc, planID, "nc-y")

	createConflictTask(t, entitySvc, featA, "nc-task-x", "nc", []string{"internal/logging/logger.go"}, nil)
	createConflictTask(t, entitySvc, featB, "nc-task-y", "nc", []string{"internal/database/migrate.go"}, nil)

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}
	if result.Pairs[0].Recommendation != "safe_to_parallelise" {
		t.Errorf("expected safe_to_parallelise, got: %s", result.Pairs[0].Recommendation)
	}
}

// AC-005: Feature whose tasks all have empty files_planned → NoFileData: true, no error.
func TestConflictFeatures_NoFileData(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	featA := newFeature(t, entitySvc, planID, "nofile-a")
	featB := newFeature(t, entitySvc, planID, "nofile-b")

	// Tasks exist but have no files_planned.
	createConflictTask(t, entitySvc, featA, "nofile-task-a", "task with no files", nil, nil)
	createConflictTask(t, entitySvc, featB, "nofile-task-b", "task with no files", nil, nil)

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	if len(result.Features) != 2 {
		t.Fatalf("expected 2 feature infos, got %d", len(result.Features))
	}
	for _, fi := range result.Features {
		if !fi.NoFileData {
			t.Errorf("feature %s: expected NoFileData=true, got false", fi.FeatureID)
		}
	}
}

// AC-006: Feature with worktree record created N days ago → DriftDays ≈ N (± 1).
func TestConflictFeatures_DriftDays_WithWorktree(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	mock := newMockBranchLookup()
	conflictSvc := NewConflictService(entitySvc, mock, "")

	featA := newFeature(t, entitySvc, planID, "drift-a")
	featB := newFeature(t, entitySvc, planID, "drift-b")

	const daysAgo = 7
	mock.branchCreatedAt[featA] = time.Now().AddDate(0, 0, -daysAgo)
	mock.branchCreatedAt[featB] = time.Now().AddDate(0, 0, -daysAgo)

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	for _, fi := range result.Features {
		if fi.DriftDays == nil {
			t.Errorf("feature %s: expected DriftDays to be set, got nil", fi.FeatureID)
			continue
		}
		got := *fi.DriftDays
		if got < daysAgo-1 || got > daysAgo+1 {
			t.Errorf("feature %s: DriftDays=%d, want ~%d (±1)", fi.FeatureID, got, daysAgo)
		}
	}
}

// AC-007: Feature with no worktree record → DriftDays is nil.
func TestConflictFeatures_DriftDays_NoWorktree(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	mock := newMockBranchLookup()
	conflictSvc := NewConflictService(entitySvc, mock, "")

	featA := newFeature(t, entitySvc, planID, "nowt-a")
	featB := newFeature(t, entitySvc, planID, "nowt-b")

	// Simulate no worktree record by returning an error for GetBranchCreatedAt.
	mock.createdAtErr[featA] = errors.New("no worktree record")
	mock.createdAtErr[featB] = errors.New("no worktree record")

	result, err := conflictSvc.CheckFeatures([]string{featA, featB})
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}
	for _, fi := range result.Features {
		if fi.DriftDays != nil {
			t.Errorf("feature %s: expected DriftDays=nil, got %d", fi.FeatureID, *fi.DriftDays)
		}
	}
}

// AC-008: feature_ids mode result contains FeatureConflictResult structure with pairs, risk, features.
func TestConflictFeatures_ResultStructure(t *testing.T) {
	entitySvc, planID := newConflictFeatureEnv(t)
	conflictSvc := NewConflictService(entitySvc, nil, "")

	featA := newFeature(t, entitySvc, planID, "struct-a")
	featB := newFeature(t, entitySvc, planID, "struct-b")
	featC := newFeature(t, entitySvc, planID, "struct-c")

	createConflictTask(t, entitySvc, featA, "struct-task-a", "task a", []string{"internal/a/a.go"}, nil)
	createConflictTask(t, entitySvc, featB, "struct-task-b", "task b", []string{"internal/b/b.go"}, nil)
	createConflictTask(t, entitySvc, featC, "struct-task-c", "task c", []string{"internal/c/c.go"}, nil)

	featureIDs := []string{featA, featB, featC}
	result, err := conflictSvc.CheckFeatures(featureIDs)
	if err != nil {
		t.Fatalf("CheckFeatures: %v", err)
	}

	if len(result.FeatureIDs) != 3 {
		t.Errorf("expected 3 FeatureIDs, got %d", len(result.FeatureIDs))
	}
	if result.OverallRisk == "" {
		t.Error("expected OverallRisk to be set")
	}
	// C(3,2) = 3 pairs
	if len(result.Pairs) != 3 {
		t.Errorf("expected 3 pairs for 3 features, got %d", len(result.Pairs))
	}
	if len(result.Features) != 3 {
		t.Errorf("expected 3 feature infos, got %d", len(result.Features))
	}
	for _, pair := range result.Pairs {
		if pair.FeatureA == "" || pair.FeatureB == "" {
			t.Errorf("pair missing feature IDs: %+v", pair)
		}
		if pair.Risk == "" {
			t.Errorf("pair missing risk: %+v", pair)
		}
		if pair.Recommendation == "" {
			t.Errorf("pair missing recommendation: %+v", pair)
		}
	}
}
