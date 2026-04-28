package service

import (
	"fmt"
	"strings"
	"testing"
)

// TestComputeFeatureRollup_EmptyFeature verifies that a feature with no tasks
// returns a zero-value rollup with nil TaskTotal.
func TestComputeFeatureRollup_EmptyFeature(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-test-plan"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "empty-feature",
		Parent:    planID,
		Summary:   "Empty feature with no tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	if rollup.TaskTotal != nil {
		t.Errorf("TaskTotal = %v, want nil (no tasks)", *rollup.TaskTotal)
	}
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0", rollup.Progress)
	}
	if rollup.TaskCount != 0 {
		t.Errorf("TaskCount = %v, want 0", rollup.TaskCount)
	}
	if rollup.EstimatedTaskCount != 0 {
		t.Errorf("EstimatedTaskCount = %v, want 0", rollup.EstimatedTaskCount)
	}
	if rollup.ExcludedTaskCount != 0 {
		t.Errorf("ExcludedTaskCount = %v, want 0", rollup.ExcludedTaskCount)
	}
}

// TestComputeFeatureRollup_NoEstimates verifies that tasks without estimates
// are counted in TaskCount but not EstimatedTaskCount, and TaskTotal is nil.
func TestComputeFeatureRollup_NoEstimates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-test-plan"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "unestimated-feature",
		Parent:    planID,
		Summary:   "Feature with unestimated tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// Create two tasks, neither gets an estimate.
	for i, slug := range []string{"task-alpha", "task-beta"} {
		_, err := svc.CreateTask(CreateTaskInput{
			Name:          "test",
			ParentFeature: feat.ID,
			Slug:          slug,
			Summary:       "Task without estimate",
		})
		if err != nil {
			t.Fatalf("CreateTask[%d] error: %v", i, err)
		}
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	if rollup.TaskTotal != nil {
		t.Errorf("TaskTotal = %v, want nil (no estimates)", *rollup.TaskTotal)
	}
	if rollup.TaskCount != 2 {
		t.Errorf("TaskCount = %v, want 2", rollup.TaskCount)
	}
	if rollup.EstimatedTaskCount != 0 {
		t.Errorf("EstimatedTaskCount = %v, want 0", rollup.EstimatedTaskCount)
	}
	if rollup.ExcludedTaskCount != 0 {
		t.Errorf("ExcludedTaskCount = %v, want 0", rollup.ExcludedTaskCount)
	}
}

// TestComputeFeatureRollup_WithNotPlanned verifies that not-planned tasks are
// excluded from TaskCount and EstimatedTaskCount, and counted in ExcludedTaskCount.
func TestComputeFeatureRollup_WithNotPlanned(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-test-plan"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "np-feature",
		Parent:    planID,
		Summary:   "Feature with a not-planned task",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// T1 will be transitioned to not-planned — should be excluded.
	t1, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "will-not-plan",
		Summary:       "This task is not planned",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 gets estimate=5, stays queued — should contribute to rollup.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "estimated-two",
		Summary:       "Task two with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// T3 gets estimate=3, stays queued — should contribute to rollup.
	t3, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "estimated-three",
		Summary:       "Task three with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T3 error: %v", err)
	}

	// T4: no estimate, stays queued — counted in TaskCount, not EstimatedTaskCount.
	_, err = svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "no-estimate-four",
		Summary:       "Task four without estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T4 error: %v", err)
	}

	// Transition T1 to not-planned (queued → not-planned).
	if _, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     t1.ID,
		Status: "not-planned",
	}); err != nil {
		t.Fatalf("UpdateStatus(T1, not-planned) error: %v", err)
	}

	// Set estimates on T2 and T3.
	if _, _, err := svc.SetEstimate("task", t2.ID, 5); err != nil {
		t.Fatalf("SetEstimate(T2, 5) error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", t3.ID, 3); err != nil {
		t.Fatalf("SetEstimate(T3, 3) error: %v", err)
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	// T1 excluded; T2, T3, T4 active in rollup.
	if rollup.TaskCount != 3 {
		t.Errorf("TaskCount = %v, want 3 (T2, T3, T4)", rollup.TaskCount)
	}
	if rollup.ExcludedTaskCount != 1 {
		t.Errorf("ExcludedTaskCount = %v, want 1 (T1)", rollup.ExcludedTaskCount)
	}
	if rollup.EstimatedTaskCount != 2 {
		t.Errorf("EstimatedTaskCount = %v, want 2 (T2, T3)", rollup.EstimatedTaskCount)
	}
	if rollup.TaskTotal == nil {
		t.Fatal("TaskTotal = nil, want 8")
	}
	if *rollup.TaskTotal != 8 {
		t.Errorf("*TaskTotal = %v, want 8 (5+3)", *rollup.TaskTotal)
	}
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0 (no done tasks)", rollup.Progress)
	}
}

// TestComputeFeatureRollup_WithDuplicate verifies that duplicate tasks are
// treated the same as not-planned: excluded from the rollup.
func TestComputeFeatureRollup_WithDuplicate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-test-plan"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "dup-feature",
		Parent:    planID,
		Summary:   "Feature with a duplicate task",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// T1 will be transitioned to duplicate — should be excluded.
	t1, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "will-duplicate",
		Summary:       "This task is a duplicate",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 gets estimate=8 — should contribute.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "real-task",
		Summary:       "Real task with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// Transition T1 to duplicate (queued → duplicate).
	if _, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     t1.ID,
		Status: "duplicate",
	}); err != nil {
		t.Fatalf("UpdateStatus(T1, duplicate) error: %v", err)
	}

	// Set estimate on T2.
	if _, _, err := svc.SetEstimate("task", t2.ID, 8); err != nil {
		t.Fatalf("SetEstimate(T2, 8) error: %v", err)
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	if rollup.TaskCount != 1 {
		t.Errorf("TaskCount = %v, want 1 (T2 only)", rollup.TaskCount)
	}
	if rollup.ExcludedTaskCount != 1 {
		t.Errorf("ExcludedTaskCount = %v, want 1 (T1 duplicate)", rollup.ExcludedTaskCount)
	}
	if rollup.EstimatedTaskCount != 1 {
		t.Errorf("EstimatedTaskCount = %v, want 1", rollup.EstimatedTaskCount)
	}
	if rollup.TaskTotal == nil {
		t.Fatal("TaskTotal = nil, want 8")
	}
	if *rollup.TaskTotal != 8 {
		t.Errorf("*TaskTotal = %v, want 8", *rollup.TaskTotal)
	}
}

// TestComputeFeatureRollup_WithDoneTask verifies that done tasks contribute to
// both TaskTotal and Progress.
func TestComputeFeatureRollup_WithDoneTask(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-done-test"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "done-feature",
		Parent:    planID,
		Summary:   "Feature with a completed task",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// T1 will reach done — estimate=5.
	t1, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "task-to-complete",
		Summary:       "Task that will be completed",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 stays queued — estimate=3.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "task-in-progress",
		Summary:       "Task still in queue",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// Set estimates before transitioning.
	if _, _, err := svc.SetEstimate("task", t1.ID, 5); err != nil {
		t.Fatalf("SetEstimate(T1, 5) error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", t2.ID, 3); err != nil {
		t.Fatalf("SetEstimate(T2, 3) error: %v", err)
	}

	// Walk T1 through the full lifecycle to done:
	// queued → ready → active → needs-review → done
	transitions := []string{"ready", "active", "needs-review", "done"}
	for _, status := range transitions {
		if _, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   "task",
			ID:     t1.ID,
			Status: status,
		}); err != nil {
			t.Fatalf("UpdateStatus(T1, %s) error: %v", status, err)
		}
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	if rollup.TaskCount != 2 {
		t.Errorf("TaskCount = %v, want 2", rollup.TaskCount)
	}
	if rollup.EstimatedTaskCount != 2 {
		t.Errorf("EstimatedTaskCount = %v, want 2", rollup.EstimatedTaskCount)
	}
	if rollup.ExcludedTaskCount != 0 {
		t.Errorf("ExcludedTaskCount = %v, want 0", rollup.ExcludedTaskCount)
	}
	if rollup.TaskTotal == nil {
		t.Fatal("TaskTotal = nil, want 8")
	}
	if *rollup.TaskTotal != 8 {
		t.Errorf("*TaskTotal = %v, want 8 (5+3)", *rollup.TaskTotal)
	}
	if rollup.Progress != 5 {
		t.Errorf("Progress = %v, want 5 (T1 done)", rollup.Progress)
	}
}

// TestComputeFeatureRollup_FeatureIsolation verifies that tasks belonging to a
// different feature are not included in another feature's rollup.
func TestComputeFeatureRollup_FeatureIsolation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-isolation-test"
	writeTestPlan(t, svc, planID)

	// Feature A — the one we'll query.
	featA, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-alpha",
		Parent:    planID,
		Summary:   "Feature A",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature A error: %v", err)
	}

	// Feature B — tasks here must NOT appear in Feature A's rollup.
	featB, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "feature-beta",
		Parent:    planID,
		Summary:   "Feature B",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature B error: %v", err)
	}

	// Create a task for Feature A with estimate=5.
	tA, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featA.ID,
		Slug:          "task-for-a",
		Summary:       "Task belonging to Feature A",
	})
	if err != nil {
		t.Fatalf("CreateTask(A) error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", tA.ID, 5); err != nil {
		t.Fatalf("SetEstimate(tA, 5) error: %v", err)
	}

	// Create a task for Feature B with estimate=13.
	tB, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featB.ID,
		Slug:          "task-for-b",
		Summary:       "Task belonging to Feature B",
	})
	if err != nil {
		t.Fatalf("CreateTask(B) error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", tB.ID, 13); err != nil {
		t.Fatalf("SetEstimate(tB, 13) error: %v", err)
	}

	// Rollup for Feature A should only see tA.
	rollupA, err := svc.ComputeFeatureRollup(featA.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup(A) error: %v", err)
	}

	if rollupA.TaskCount != 1 {
		t.Errorf("Feature A TaskCount = %v, want 1", rollupA.TaskCount)
	}
	if rollupA.TaskTotal == nil {
		t.Fatal("Feature A TaskTotal = nil, want 5")
	}
	if *rollupA.TaskTotal != 5 {
		t.Errorf("Feature A *TaskTotal = %v, want 5", *rollupA.TaskTotal)
	}

	// Rollup for Feature B should only see tB.
	rollupB, err := svc.ComputeFeatureRollup(featB.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup(B) error: %v", err)
	}

	if rollupB.TaskCount != 1 {
		t.Errorf("Feature B TaskCount = %v, want 1", rollupB.TaskCount)
	}
	if rollupB.TaskTotal == nil {
		t.Fatal("Feature B TaskTotal = nil, want 13")
	}
	if *rollupB.TaskTotal != 13 {
		t.Errorf("Feature B *TaskTotal = %v, want 13", *rollupB.TaskTotal)
	}
}

// TestComputeFeatureRollup_BothExcludedStates verifies that both not-planned and
// duplicate tasks are excluded together, and only active tasks count.
func TestComputeFeatureRollup_BothExcludedStates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-mixed-exclusion"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "mixed-feature",
		Parent:    planID,
		Summary:   "Feature with both excluded states",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// T1 → not-planned
	t1, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "not-planned-task",
		Summary:       "Goes to not-planned",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 → duplicate
	t2, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "duplicate-task",
		Summary:       "Goes to duplicate",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// T3 → estimate=2, stays active
	t3, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "active-task",
		Summary:       "Stays active with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T3 error: %v", err)
	}

	// Transition exclusions
	if _, err := svc.UpdateStatus(UpdateStatusInput{Type: "task", ID: t1.ID, Status: "not-planned"}); err != nil {
		t.Fatalf("UpdateStatus(T1, not-planned) error: %v", err)
	}
	if _, err := svc.UpdateStatus(UpdateStatusInput{Type: "task", ID: t2.ID, Status: "duplicate"}); err != nil {
		t.Fatalf("UpdateStatus(T2, duplicate) error: %v", err)
	}

	// Set estimate on T3
	if _, _, err := svc.SetEstimate("task", t3.ID, 2); err != nil {
		t.Fatalf("SetEstimate(T3, 2) error: %v", err)
	}

	rollup, err := svc.ComputeFeatureRollup(feat.ID)
	if err != nil {
		t.Fatalf("ComputeFeatureRollup error: %v", err)
	}

	if rollup.ExcludedTaskCount != 2 {
		t.Errorf("ExcludedTaskCount = %v, want 2 (T1+T2)", rollup.ExcludedTaskCount)
	}
	if rollup.TaskCount != 1 {
		t.Errorf("TaskCount = %v, want 1 (T3 only)", rollup.TaskCount)
	}
	if rollup.EstimatedTaskCount != 1 {
		t.Errorf("EstimatedTaskCount = %v, want 1", rollup.EstimatedTaskCount)
	}
	if rollup.TaskTotal == nil {
		t.Fatal("TaskTotal = nil, want 2")
	}
	if *rollup.TaskTotal != 2 {
		t.Errorf("*TaskTotal = %v, want 2", *rollup.TaskTotal)
	}
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0", rollup.Progress)
	}
}

// TestSetEstimate_ValidValues verifies that SetEstimate succeeds for all valid
// scale values and rejects invalid ones.
func TestSetEstimate_ValidAndInvalid(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-estimate-test"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "est-feature",
		Parent:    planID,
		Summary:   "Feature for estimate testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "estimable-task",
		Summary:       "Task to estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// All valid scale values should succeed
	for _, v := range EstimationScale {
		v := v
		if _, _, err := svc.SetEstimate("task", task.ID, v); err != nil {
			t.Errorf("SetEstimate(task, %v) returned unexpected error: %v", v, err)
		}
	}

	// Invalid value should fail
	_, _, err = svc.SetEstimate("task", task.ID, 7)
	if err == nil {
		t.Error("SetEstimate(task, 7) = nil, want error for invalid scale value")
	}
}

// TestSetEstimate_SoftLimitWarning verifies that SetEstimate returns a warning
// when the estimate exceeds the soft limit for the entity type.
func TestSetEstimate_SoftLimitWarning(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-softlimit-test"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "sl-feature",
		Parent:    planID,
		Summary:   "Feature for soft-limit testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "large-task",
		Summary:       "Task with a large estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// 13 is the task soft limit — should produce no warning
	_, warning, err := svc.SetEstimate("task", task.ID, 13)
	if err != nil {
		t.Fatalf("SetEstimate(13) error: %v", err)
	}
	if warning != "" {
		t.Errorf("SetEstimate(task, 13) warning = %q, want empty (at limit)", warning)
	}

	// 20 exceeds the task soft limit — should produce a warning
	_, warning, err = svc.SetEstimate("task", task.ID, 20)
	if err != nil {
		t.Fatalf("SetEstimate(20) error: %v", err)
	}
	if warning == "" {
		t.Error("SetEstimate(task, 20) warning = empty, want non-empty (exceeds task soft limit)")
	}
}

// TestSetEstimate_InvalidEntityTypeOrNonexistent verifies error paths for
// SetEstimate with an invalid entity type and a non-existent entity ID.
func TestSetEstimate_InvalidEntityTypeOrNonexistent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Invalid entity type should fail (no "nonexistents" directory).
	_, _, err := svc.SetEstimate("nonexistent", "anything", 5)
	if err == nil {
		t.Error("SetEstimate with invalid entity type should fail")
	}

	// Non-existent task should fail (no file matching TASK-nonexistent in tasks/).
	_, _, err = svc.SetEstimate("task", "TASK-nonexistent", 5)
	if err == nil {
		t.Error("SetEstimate with non-existent task ID should fail")
	}

	// Non-existent feature should also fail.
	_, _, err = svc.SetEstimate("feature", "FEAT-nonexistent", 5)
	if err == nil {
		t.Error("SetEstimate with non-existent feature ID should fail")
	}
}

// ─── ComputeBatchRollup tests ─────────────────────────────────────────────────

// TestComputeBatchRollup_EmptyBatch verifies that a batch with no features
// returns a zero-value rollup with nil FeatureTotal.
func TestComputeBatchRollup_EmptyBatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	batchID := "B1-empty-batch"
	writeTestPlan(t, svc, batchID)

	rollup, err := svc.ComputeBatchRollup(batchID)
	if err != nil {
		t.Fatalf("ComputeBatchRollup error: %v", err)
	}

	if rollup.FeatureTotal != nil {
		t.Errorf("FeatureTotal = %v, want nil (no features)", *rollup.FeatureTotal)
	}
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0", rollup.Progress)
	}
	if rollup.FeatureCount != 0 {
		t.Errorf("FeatureCount = %v, want 0", rollup.FeatureCount)
	}
	if rollup.EstimatedFeatureCount != 0 {
		t.Errorf("EstimatedFeatureCount = %v, want 0", rollup.EstimatedFeatureCount)
	}
}

// TestComputeBatchRollup_TaskTotal verifies that task estimates are summed
// across features in the batch.
func TestComputeBatchRollup_TaskTotal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	batchID := "B1-task-total"
	writeTestPlan(t, svc, batchID)

	featA, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "Feature A",
		Slug:      "feat-a",
		Parent:    batchID,
		Summary:   "Feature with tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature A error: %v", err)
	}

	featB, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "Feature B",
		Slug:      "feat-b",
		Parent:    batchID,
		Summary:   "Feature with own estimate",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature B error: %v", err)
	}

	// Feature A: three tasks with estimates 2, 3, 5 (all active).
	taskA1, err := svc.CreateTask(CreateTaskInput{
		Name:          "Task A1",
		ParentFeature: featA.ID,
		Slug:          "task-a1",
		Summary:       "Task A1 summary",
	})
	if err != nil {
		t.Fatalf("CreateTask A1 error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", taskA1.ID, 2); err != nil {
		t.Fatalf("SetEstimate A1: %v", err)
	}
	taskA2, err := svc.CreateTask(CreateTaskInput{
		Name:          "Task A2",
		ParentFeature: featA.ID,
		Slug:          "task-a2",
		Summary:       "Task A2 summary",
	})
	if err != nil {
		t.Fatalf("CreateTask A2 error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", taskA2.ID, 3); err != nil {
		t.Fatalf("SetEstimate A2: %v", err)
	}
	taskA3, err := svc.CreateTask(CreateTaskInput{
		Name:          "Task A3",
		ParentFeature: featA.ID,
		Slug:          "task-a3",
		Summary:       "Task A3 summary",
	})
	if err != nil {
		t.Fatalf("CreateTask A3 error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", taskA3.ID, 5); err != nil {
		t.Fatalf("SetEstimate A3: %v", err)
	}

	// Feature B: no tasks, own estimate of 8.
	if _, _, err := svc.SetEstimate("feature", featB.ID, 8); err != nil {
		t.Fatalf("SetEstimate featB: %v", err)
	}

	rollup, err := svc.ComputeBatchRollup(batchID)
	if err != nil {
		t.Fatalf("ComputeBatchRollup error: %v", err)
	}

	// FeatureTotal = 10 (A task sum) + 8 (B own estimate) = 18
	if rollup.FeatureTotal == nil {
		t.Fatal("FeatureTotal is nil, want 18")
	}
	if *rollup.FeatureTotal != 18 {
		t.Errorf("FeatureTotal = %v, want 18", *rollup.FeatureTotal)
	}
	// No done tasks → Progress = 0
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0", rollup.Progress)
	}
	if rollup.FeatureCount != 2 {
		t.Errorf("FeatureCount = %v, want 2", rollup.FeatureCount)
	}
	if rollup.EstimatedFeatureCount != 2 {
		t.Errorf("EstimatedFeatureCount = %v, want 2", rollup.EstimatedFeatureCount)
	}
}

// TestComputeBatchRollup_WithProgress verifies that done tasks contribute
// to Progress but not-planned tasks are excluded.
func TestComputeBatchRollup_WithProgress(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	batchID := "B1-progress-batch"
	writeTestPlan(t, svc, batchID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "Feature",
		Slug:      "main-feat",
		Parent:    batchID,
		Summary:   "Feature with mixed tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	// Done task — estimate 5
	doneTask, err := svc.CreateTask(CreateTaskInput{
		Name:          "Done Task",
		ParentFeature: feat.ID,
		Slug:          "done-task",
		Summary:       "Done task summary",
	})
	if err != nil {
		t.Fatalf("CreateTask done error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", doneTask.ID, 5); err != nil {
		t.Fatalf("SetEstimate done: %v", err)
	}
	makeTaskDone(t, svc, doneTask.ID, doneTask.Slug)

	// Active task — estimate 3
	activeTask, err := svc.CreateTask(CreateTaskInput{
		Name:          "Active Task",
		ParentFeature: feat.ID,
		Slug:          "active-task",
		Summary:       "Active task summary",
	})
	if err != nil {
		t.Fatalf("CreateTask active error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", activeTask.ID, 3); err != nil {
		t.Fatalf("SetEstimate active: %v", err)
	}

	// Not-planned task — estimate 8 (should be excluded)
	npTask, err := svc.CreateTask(CreateTaskInput{
		Name:          "Not Planned",
		ParentFeature: feat.ID,
		Slug:          "not-planned-task",
		Summary:       "Not planned task summary",
	})
	if err != nil {
		t.Fatalf("CreateTask not-planned error: %v", err)
	}
	if _, _, err := svc.SetEstimate("task", npTask.ID, 8); err != nil {
		t.Fatalf("SetEstimate np: %v", err)
	}
	makeTaskNotPlanned(t, svc, npTask.ID, npTask.Slug)

	rollup, err := svc.ComputeBatchRollup(batchID)
	if err != nil {
		t.Fatalf("ComputeBatchRollup error: %v", err)
	}

	// FeatureTotal = 5 (done) + 3 (active) = 8; not-planned excluded
	if rollup.FeatureTotal == nil {
		t.Fatal("FeatureTotal is nil, want 8")
	}
	if *rollup.FeatureTotal != 8 {
		t.Errorf("FeatureTotal = %v, want 8", *rollup.FeatureTotal)
	}
	// Progress = 5 (only the done task)
	if rollup.Progress != 5 {
		t.Errorf("Progress = %v, want 5", rollup.Progress)
	}
	if rollup.FeatureCount != 1 {
		t.Errorf("FeatureCount = %v, want 1", rollup.FeatureCount)
	}
}

// ─── ComputePlanRollup tests ──────────────────────────────────────────────────

// createFeatureWithTask creates a feature with one task, sets the task estimate,
// and optionally marks it done.
func createFeatureAndTask(t *testing.T, svc *EntityService, batchID, featSlug, taskSlug string, estimate float64, done bool) {
	t.Helper()
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "Feature " + featSlug,
		Slug:      featSlug,
		Parent:    batchID,
		Summary:   "Feature for rollup testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature %s error: %v", featSlug, err)
	}
	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "Task " + taskSlug,
		ParentFeature: feat.ID,
		Slug:          taskSlug,
		Summary:       "Task for rollup testing",
	})
	if err != nil {
		t.Fatalf("CreateTask %s error: %v", taskSlug, err)
	}
	if _, _, err := svc.SetEstimate("task", task.ID, estimate); err != nil {
		t.Fatalf("SetEstimate %s: %v", taskSlug, err)
	}
	if done {
		makeTaskDone(t, svc, task.ID, task.Slug)
	}
}

// TestComputePlanRollup_EmptyPlan verifies that a plan with no children
// returns a zero-value rollup with nil Total.
func TestComputePlanRollup_EmptyPlan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	planID := "P1-empty"
	writeTestStrategicPlan(t, svc, planID, "active")

	rollup, err := svc.ComputePlanRollup(planID)
	if err != nil {
		t.Fatalf("ComputePlanRollup error: %v", err)
	}

	if rollup.Total != nil {
		t.Errorf("Total = %v, want nil (no children)", *rollup.Total)
	}
	if rollup.Progress != 0 {
		t.Errorf("Progress = %v, want 0", rollup.Progress)
	}
	if rollup.BatchCount != 0 {
		t.Errorf("BatchCount = %v, want 0", rollup.BatchCount)
	}
	if rollup.PlanCount != 0 {
		t.Errorf("PlanCount = %v, want 0", rollup.PlanCount)
	}
}

// TestComputePlanRollup_WithChildBatches verifies that child batch totals
// are aggregated correctly.
func TestComputePlanRollup_WithChildBatches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	planID := "P1-batch-parent"
	writeTestStrategicPlan(t, svc, planID, "active")

	// Batch A: FeatureTotal = 10, Progress = 5
	batchA := "B1-batch-a"
	writeTestBatchWithParent(t, svc, batchA, planID)
	createFeatureAndTask(t, svc, batchA, "feat-a1", "task-a1", 5, true)  // done: progress 5
	createFeatureAndTask(t, svc, batchA, "feat-a2", "task-a2", 5, false) // active: no progress

	// Batch B: FeatureTotal = 5, Progress = 5
	batchB := "B2-batch-b"
	writeTestBatchWithParent(t, svc, batchB, planID)
	createFeatureAndTask(t, svc, batchB, "feat-b1", "task-b1", 5, true) // done: progress 5

	rollup, err := svc.ComputePlanRollup(planID)
	if err != nil {
		t.Fatalf("ComputePlanRollup error: %v", err)
	}

	if rollup.Total == nil {
		t.Fatal("Total is nil, want 15")
	}
	if *rollup.Total != 15 {
		t.Errorf("Total = %v, want 15", *rollup.Total)
	}
	if rollup.Progress != 10 {
		t.Errorf("Progress = %v, want 10", rollup.Progress)
	}
	if rollup.BatchCount != 2 {
		t.Errorf("BatchCount = %v, want 2", rollup.BatchCount)
	}
	if rollup.PlanCount != 0 {
		t.Errorf("PlanCount = %v, want 0", rollup.PlanCount)
	}
}

// TestComputePlanRollup_WithChildPlans verifies that child plan totals
// are aggregated recursively.
func TestComputePlanRollup_WithChildPlans(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	planID := "P1-plan-parent"
	writeTestStrategicPlan(t, svc, planID, "active")

	// Child plan P2 with one batch: FeatureTotal = 13, Progress = 13
	childPlanID := "P2-child"
	writeTestStrategicPlanWithParent(t, svc, childPlanID, "active", planID, 0)
	batchC := "B1-child-batch"
	writeTestBatchWithParent(t, svc, batchC, childPlanID)
	createFeatureAndTask(t, svc, batchC, "feat-c1", "task-c1", 13, true) // done: progress 13

	rollup, err := svc.ComputePlanRollup(planID)
	if err != nil {
		t.Fatalf("ComputePlanRollup error: %v", err)
	}

	if rollup.Total == nil {
		t.Fatal("Total is nil, want 13")
	}
	if *rollup.Total != 13 {
		t.Errorf("Total = %v, want 13", *rollup.Total)
	}
	if rollup.Progress != 13 {
		t.Errorf("Progress = %v, want 13", rollup.Progress)
	}
	if rollup.BatchCount != 0 {
		t.Errorf("BatchCount = %v, want 0", rollup.BatchCount)
	}
	if rollup.PlanCount != 1 {
		t.Errorf("PlanCount = %v, want 1", rollup.PlanCount)
	}
}

// TestComputePlanRollup_MixedChildren verifies aggregation across both
// child batches and child plans.
func TestComputePlanRollup_MixedChildren(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	planID := "P1-mixed"
	writeTestStrategicPlan(t, svc, planID, "active")

	// Direct child batch: FeatureTotal = 8, Progress = 5
	batchD := "B1-mixed-batch"
	writeTestBatchWithParent(t, svc, batchD, planID)
	createFeatureAndTask(t, svc, batchD, "feat-d1", "task-d1", 5, true)  // done: progress 5
	createFeatureAndTask(t, svc, batchD, "feat-d2", "task-d2", 3, false) // active: no progress

	// Direct child plan: Total = 8, Progress = 8
	childPlanID := "P2-mixed-child"
	writeTestStrategicPlanWithParent(t, svc, childPlanID, "active", planID, 0)
	batchE := "B2-mixed-child-batch"
	writeTestBatchWithParent(t, svc, batchE, childPlanID)
	createFeatureAndTask(t, svc, batchE, "feat-e1", "task-e1", 8, true) // done: progress 8

	rollup, err := svc.ComputePlanRollup(planID)
	if err != nil {
		t.Fatalf("ComputePlanRollup error: %v", err)
	}

	// Total = 8 (batch) + 8 (plan) = 16
	if rollup.Total == nil {
		t.Fatal("Total is nil, want 16")
	}
	if *rollup.Total != 16 {
		t.Errorf("Total = %v, want 16", *rollup.Total)
	}
	// Progress = 5 (batch) + 8 (plan) = 13
	if rollup.Progress != 13 {
		t.Errorf("Progress = %v, want 13", rollup.Progress)
	}
	if rollup.BatchCount != 1 {
		t.Errorf("BatchCount = %v, want 1", rollup.BatchCount)
	}
	if rollup.PlanCount != 1 {
		t.Errorf("PlanCount = %v, want 1", rollup.PlanCount)
	}
}

// TestComputePlanRollup_ThreeLevels verifies recursive aggregation through
// three levels: grandparent → parent → child batch.
func TestComputePlanRollup_ThreeLevels(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	grandparentID := "P1-grandparent"
	writeTestStrategicPlan(t, svc, grandparentID, "active")

	parentID := "P2-parent"
	writeTestStrategicPlanWithParent(t, svc, parentID, "active", grandparentID, 0)

	childID := "P3-child"
	writeTestStrategicPlanWithParent(t, svc, childID, "active", parentID, 0)

	batchF := "B1-deep-batch"
	writeTestBatchWithParent(t, svc, batchF, childID)
	createFeatureAndTask(t, svc, batchF, "feat-f1", "task-f1", 13, true) // done: progress 13
	createFeatureAndTask(t, svc, batchF, "feat-f2", "task-f2", 5, false) // active: no progress

	rollup, err := svc.ComputePlanRollup(grandparentID)
	if err != nil {
		t.Fatalf("ComputePlanRollup error: %v", err)
	}

	// Total = 18 (from batch under child), Progress = 13
	if rollup.Total == nil {
		t.Fatal("Total is nil, want 18")
	}
	if *rollup.Total != 18 {
		t.Errorf("Total = %v, want 18", *rollup.Total)
	}
	if rollup.Progress != 13 {
		t.Errorf("Progress = %v, want 13", rollup.Progress)
	}
	if rollup.BatchCount != 0 {
		t.Errorf("BatchCount = %v, want 0 (no direct batches)", rollup.BatchCount)
	}
	if rollup.PlanCount != 1 {
		t.Errorf("PlanCount = %v, want 1", rollup.PlanCount)
	}
}

// TestComputePlanRollup_SiblingIsolation verifies that two sibling plans
// compute independent rollups.
func TestComputePlanRollup_SiblingIsolation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	writeTestStrategicPlan(t, svc, "P1-sib-a", "active")
	writeTestStrategicPlan(t, svc, "P2-sib-b", "active")

	// Batch under P1-sib-a: FeatureTotal = 8
	writeTestBatchWithParent(t, svc, "B1-sib-a-batch", "P1-sib-a")
	createFeatureAndTask(t, svc, "B1-sib-a-batch", "feat-sa1", "task-sa1", 8, false)

	// Batch under P2-sib-b: FeatureTotal = 13
	writeTestBatchWithParent(t, svc, "B2-sib-b-batch", "P2-sib-b")
	createFeatureAndTask(t, svc, "B2-sib-b-batch", "feat-sb1", "task-sb1", 13, false)

	rollupA, err := svc.ComputePlanRollup("P1-sib-a")
	if err != nil {
		t.Fatalf("ComputePlanRollup(P1-sib-a) error: %v", err)
	}
	if rollupA.Total == nil || *rollupA.Total != 8 {
		t.Errorf("P1-sib-a Total = %v, want 8", rollupA.Total)
	}

	rollupB, err := svc.ComputePlanRollup("P2-sib-b")
	if err != nil {
		t.Fatalf("ComputePlanRollup(P2-sib-b) error: %v", err)
	}
	if rollupB.Total == nil || *rollupB.Total != 13 {
		t.Errorf("P2-sib-b Total = %v, want 13", rollupB.Total)
	}

	// P1-sib-a should NOT include P2-sib-b's total.
	if rollupA.Total != nil && *rollupA.Total == 21 {
		t.Error("P1-sib-a erroneously includes P2-sib-b's total")
	}
}

// TestComputePlanRollup_DepthLimitExceeded verifies that ComputePlanRollup
// returns a depth-limit error when the plan hierarchy exceeds maxPlanRollupDepth.
func TestComputePlanRollup_DepthLimitExceeded(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-04-28T12:00:00Z")

	// Create a chain deeper than maxPlanRollupDepth (50).
	writeTestStrategicPlan(t, svc, "P1-level-1", "active")
	parent := "P1-level-1"
	for i := 2; i <= 52; i++ {
		id := fmt.Sprintf("P1-level-%d", i)
		writeTestStrategicPlanWithParent(t, svc, id, "active", parent, 0)
		parent = id
	}

	// ComputePlanRollup on the root should hit the depth guard.
	_, err := svc.ComputePlanRollup("P1-level-1")
	if err == nil {
		t.Fatal("ComputePlanRollup with >50 levels should return error")
	}
	if !strings.Contains(err.Error(), "depth limit exceeded") {
		t.Fatalf("Error should mention 'depth limit exceeded', got: %v", err)
	}
}
