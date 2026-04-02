package service

import (
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
		Name: "test",
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
		Name: "test",
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
			Name: "test",
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
		Name: "test",
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
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "will-not-plan",
		Summary:       "This task is not planned",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 gets estimate=5, stays queued — should contribute to rollup.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "estimated-two",
		Summary:       "Task two with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// T3 gets estimate=3, stays queued — should contribute to rollup.
	t3, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "estimated-three",
		Summary:       "Task three with estimate",
	})
	if err != nil {
		t.Fatalf("CreateTask T3 error: %v", err)
	}

	// T4: no estimate, stays queued — counted in TaskCount, not EstimatedTaskCount.
	_, err = svc.CreateTask(CreateTaskInput{
		Name: "test",
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
		Name: "test",
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
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "will-duplicate",
		Summary:       "This task is a duplicate",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 gets estimate=8 — should contribute.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
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
		Name: "test",
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
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "task-to-complete",
		Summary:       "Task that will be completed",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 stays queued — estimate=3.
	t2, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
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
		Name: "test",
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
		Name: "test",
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
		Name: "test",
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
		Name: "test",
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
		Name: "test",
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
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "not-planned-task",
		Summary:       "Goes to not-planned",
	})
	if err != nil {
		t.Fatalf("CreateTask T1 error: %v", err)
	}

	// T2 → duplicate
	t2, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "duplicate-task",
		Summary:       "Goes to duplicate",
	})
	if err != nil {
		t.Fatalf("CreateTask T2 error: %v", err)
	}

	// T3 → estimate=2, stays active
	t3, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
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
		Name: "test",
		Slug:      "est-feature",
		Parent:    planID,
		Summary:   "Feature for estimate testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
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
		Name: "test",
		Slug:      "sl-feature",
		Parent:    planID,
		Summary:   "Feature for soft-limit testing",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature error: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
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
