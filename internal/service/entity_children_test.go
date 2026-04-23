package service

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// writeTestFeatureWithStatus writes a feature record directly to storage with the
// given status, bypassing lifecycle validation. This lets tests set up features in
// terminal states (done, superseded, cancelled) without walking the full lifecycle.
func writeTestFeatureWithStatus(t *testing.T, svc *EntityService, id, slug, planID, status string) {
	t.Helper()
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"parent":     planID,
		"status":     status,
		"summary":    "Test feature for " + slug,
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "test",
		"name":       slug,
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindFeature),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestFeatureWithStatus(%s) error = %v", id, err)
	}
}

// writeTestTaskWithStatus writes a task record directly to storage with the given
// status, bypassing lifecycle validation.
func writeTestTaskWithStatus(t *testing.T, svc *EntityService, id, slug, featureID, status string) {
	t.Helper()
	fields := map[string]any{
		"id":             id,
		"slug":           slug,
		"parent_feature": featureID,
		"status":         status,
		"summary":        "Test task for " + slug,
		"name":           slug,
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindTask),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeTestTaskWithStatus(%s) error = %v", id, err)
	}
}

// makeTaskDone transitions a task from its initial "queued" state to "done"
// via the normal lifecycle: queued → ready → active → done.
func makeTaskDone(t *testing.T, svc *EntityService, id, slug string) {
	t.Helper()
	for _, next := range []string{"ready", "active", "done"} {
		_, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   "task",
			ID:     id,
			Slug:   slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("makeTaskDone: UpdateStatus(%s → %s) error = %v", id, next, err)
		}
	}
}

// makeTaskNotPlanned transitions a task from "queued" to "not-planned".
func makeTaskNotPlanned(t *testing.T, svc *EntityService, id, slug string) {
	t.Helper()
	_, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     id,
		Slug:   slug,
		Status: "not-planned",
	})
	if err != nil {
		t.Fatalf("makeTaskNotPlanned: UpdateStatus(%s) error = %v", id, err)
	}
}

// makeTaskReady transitions a task from "queued" to "ready".
func makeTaskReady(t *testing.T, svc *EntityService, id, slug string) {
	t.Helper()
	_, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     id,
		Slug:   slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("makeTaskReady: UpdateStatus(%s) error = %v", id, err)
	}
}

// --- checkAllTasksTerminal ---

func TestCheckAllTasksTerminal_NoTasks(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-no-tasks"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "empty feature",
		Slug:      "empty-feature",
		Parent:    planID,
		Summary:   "Feature with no tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	allTerminal, hasOneDone, err := svc.CheckAllTasksTerminal(feat.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (no tasks)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (no tasks)")
	}
}

func TestCheckAllTasksTerminal_AllDone(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-all-done"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "feature",
		Slug:      "feature-all-done",
		Parent:    planID,
		Summary:   "Feature with all done tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "task one", Slug: "task-one", ParentFeature: feat.ID, Summary: "first",
	})
	if err != nil {
		t.Fatalf("CreateTask(task1) error = %v", err)
	}
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "task two", Slug: "task-two", ParentFeature: feat.ID, Summary: "second",
	})
	if err != nil {
		t.Fatalf("CreateTask(task2) error = %v", err)
	}

	makeTaskDone(t, svc, task1.ID, task1.Slug)
	makeTaskDone(t, svc, task2.ID, task2.Slug)

	allTerminal, hasOneDone, err := svc.CheckAllTasksTerminal(feat.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (all done)")
	}
	if !hasOneDone {
		t.Errorf("hasOneDone = false, want true (tasks are done)")
	}
}

func TestCheckAllTasksTerminal_AllNotPlanned(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-not-planned"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "feature",
		Slug:      "feature-not-planned",
		Parent:    planID,
		Summary:   "Feature with not-planned tasks",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "task one", Slug: "task-np-one", ParentFeature: feat.ID, Summary: "first",
	})
	if err != nil {
		t.Fatalf("CreateTask(task1) error = %v", err)
	}
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "task two", Slug: "task-np-two", ParentFeature: feat.ID, Summary: "second",
	})
	if err != nil {
		t.Fatalf("CreateTask(task2) error = %v", err)
	}

	makeTaskNotPlanned(t, svc, task1.ID, task1.Slug)
	makeTaskNotPlanned(t, svc, task2.ID, task2.Slug)

	allTerminal, hasOneDone, err := svc.CheckAllTasksTerminal(feat.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (all not-planned)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (no done tasks)")
	}
}

func TestCheckAllTasksTerminal_MixedStates(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-mixed"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "feature",
		Slug:      "feature-mixed",
		Parent:    planID,
		Summary:   "Feature with mixed task states",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "task done", Slug: "task-mixed-done", ParentFeature: feat.ID, Summary: "done one",
	})
	if err != nil {
		t.Fatalf("CreateTask(done) error = %v", err)
	}
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "task ready", Slug: "task-mixed-ready", ParentFeature: feat.ID, Summary: "ready one",
	})
	if err != nil {
		t.Fatalf("CreateTask(ready) error = %v", err)
	}

	makeTaskDone(t, svc, task1.ID, task1.Slug)
	makeTaskReady(t, svc, task2.ID, task2.Slug)

	allTerminal, hasOneDone, err := svc.CheckAllTasksTerminal(feat.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal() error = %v", err)
	}
	if allTerminal {
		t.Errorf("allTerminal = true, want false (one task is ready)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (not all terminal)")
	}
}

func TestCheckAllTasksTerminal_IsolatesFeature(t *testing.T) {
	t.Parallel()

	// Tasks belonging to other features must not affect the result.
	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-isolation"
	writeTestPlan(t, svc, planID)

	feat1, err := svc.CreateFeature(CreateFeatureInput{
		Name: "f1", Slug: "feat-iso-one", Parent: planID, Summary: "one", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat1) error = %v", err)
	}
	feat2, err := svc.CreateFeature(CreateFeatureInput{
		Name: "f2", Slug: "feat-iso-two", Parent: planID, Summary: "two", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature(feat2) error = %v", err)
	}

	// feat1 has one done task.
	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "t1", Slug: "iso-task-one", ParentFeature: feat1.ID, Summary: "t1",
	})
	if err != nil {
		t.Fatalf("CreateTask(t1) error = %v", err)
	}
	makeTaskDone(t, svc, task1.ID, task1.Slug)

	// feat2 has one ready (non-terminal) task.
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "t2", Slug: "iso-task-two", ParentFeature: feat2.ID, Summary: "t2",
	})
	if err != nil {
		t.Fatalf("CreateTask(t2) error = %v", err)
	}
	makeTaskReady(t, svc, task2.ID, task2.Slug)

	// feat1 should still report allTerminal=true.
	allTerminal, hasOneDone, err := svc.CheckAllTasksTerminal(feat1.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal(feat1) error = %v", err)
	}
	if !allTerminal {
		t.Errorf("feat1: allTerminal = false, want true")
	}
	if !hasOneDone {
		t.Errorf("feat1: hasOneDone = false, want true")
	}

	// feat2 should report allTerminal=false.
	allTerminal2, _, err := svc.CheckAllTasksTerminal(feat2.ID)
	if err != nil {
		t.Fatalf("checkAllTasksTerminal(feat2) error = %v", err)
	}
	if allTerminal2 {
		t.Errorf("feat2: allTerminal = true, want false")
	}
}

// --- countNonTerminalTasks ---

func TestCountNonTerminalTasks_OneReady(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-tasks"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "f",
		Slug:      "feat-cnt",
		Parent:    planID,
		Summary:   "count test feature",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// One done task (terminal), one ready task (non-terminal).
	taskDone, err := svc.CreateTask(CreateTaskInput{
		Name: "td", Slug: "cnt-done", ParentFeature: feat.ID, Summary: "done task",
	})
	if err != nil {
		t.Fatalf("CreateTask(done) error = %v", err)
	}
	makeTaskDone(t, svc, taskDone.ID, taskDone.Slug)

	taskReady, err := svc.CreateTask(CreateTaskInput{
		Name: "tr", Slug: "cnt-ready", ParentFeature: feat.ID, Summary: "ready task",
	})
	if err != nil {
		t.Fatalf("CreateTask(ready) error = %v", err)
	}
	makeTaskReady(t, svc, taskReady.ID, taskReady.Slug)

	count, err := svc.CountNonTerminalTasks(feat.ID)
	if err != nil {
		t.Fatalf("countNonTerminalTasks() error = %v", err)
	}
	if count != 1 {
		t.Errorf("countNonTerminalTasks() = %d, want 1", count)
	}
}

func TestCountNonTerminalTasks_AllTerminal(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-all-term"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "f", Slug: "feat-cnt-term", Parent: planID, Summary: "all terminal", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "t", Slug: "cnt-term-task", ParentFeature: feat.ID, Summary: "done",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	makeTaskDone(t, svc, task.ID, task.Slug)

	count, err := svc.CountNonTerminalTasks(feat.ID)
	if err != nil {
		t.Fatalf("countNonTerminalTasks() error = %v", err)
	}
	if count != 0 {
		t.Errorf("countNonTerminalTasks() = %d, want 0", count)
	}
}

func TestCountNonTerminalTasks_NoTasks(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-none"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "f", Slug: "feat-cnt-none", Parent: planID, Summary: "no tasks", CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	count, err := svc.CountNonTerminalTasks(feat.ID)
	if err != nil {
		t.Fatalf("countNonTerminalTasks() error = %v", err)
	}
	if count != 0 {
		t.Errorf("countNonTerminalTasks() = %d, want 0", count)
	}
}

// --- checkAllFeaturesTerminal ---

func TestCheckAllFeaturesTerminal_NoFeatures(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-feat-none"
	writeTestPlan(t, svc, planID)

	allTerminal, hasOneDone, err := svc.CheckAllFeaturesTerminal(planID)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (no features)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (no features)")
	}
}

func TestCheckAllFeaturesTerminal_AllDone(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-feat-done"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTFEATURE1", "feat-done-one", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTFEATURE2", "feat-done-two", planID, string(model.FeatureStatusDone))

	allTerminal, hasOneDone, err := svc.CheckAllFeaturesTerminal(planID)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (all done)")
	}
	if !hasOneDone {
		t.Errorf("hasOneDone = false, want true (features are done)")
	}
}

func TestCheckAllFeaturesTerminal_AllSuperseded(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-feat-sup"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTSUPERSEDED", "feat-sup-one", planID, string(model.FeatureStatusSuperseded))

	allTerminal, hasOneDone, err := svc.CheckAllFeaturesTerminal(planID)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal() error = %v", err)
	}
	if !allTerminal {
		t.Errorf("allTerminal = false, want true (superseded is terminal)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (no done features)")
	}
}

func TestCheckAllFeaturesTerminal_MixedStates(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-feat-mixed"
	writeTestPlan(t, svc, planID)

	// One done (terminal) + one active (non-terminal).
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTMIXDONE1", "feat-mix-done", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTMIXDEV1", "feat-mix-dev", planID, string(model.FeatureStatusDeveloping))

	allTerminal, hasOneDone, err := svc.CheckAllFeaturesTerminal(planID)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal() error = %v", err)
	}
	if allTerminal {
		t.Errorf("allTerminal = true, want false (one feature is developing)")
	}
	if hasOneDone {
		t.Errorf("hasOneDone = true, want false (not all terminal)")
	}
}

func TestCheckAllFeaturesTerminal_IsolatesPlan(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planA := "P1-plan-a"
	planB := "P2-plan-b"
	writeTestPlan(t, svc, planA)
	writeTestPlan(t, svc, planB)

	// planA has one done feature.
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTA000001", "feat-a-done", planA, string(model.FeatureStatusDone))

	// planB has one active (non-terminal) feature.
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01TESTB000001", "feat-b-active", planB, string(model.FeatureStatusDeveloping))

	allTerminalA, hasOneDoneA, err := svc.CheckAllFeaturesTerminal(planA)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal(planA) error = %v", err)
	}
	if !allTerminalA {
		t.Errorf("planA: allTerminal = false, want true")
	}
	if !hasOneDoneA {
		t.Errorf("planA: hasOneDone = false, want true")
	}

	allTerminalB, _, err := svc.CheckAllFeaturesTerminal(planB)
	if err != nil {
		t.Fatalf("checkAllFeaturesTerminal(planB) error = %v", err)
	}
	if allTerminalB {
		t.Errorf("planB: allTerminal = true, want false")
	}
}

// --- countNonTerminalFeatures ---

func TestCountNonTerminalFeatures_Mixed(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-feat"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTDONE00001", "cnt-feat-done", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTDEV00001", "cnt-feat-dev", planID, string(model.FeatureStatusDeveloping))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTSPEC0001", "cnt-feat-spec", planID, string(model.FeatureStatusSpecifying))

	count, err := svc.CountNonTerminalFeatures(planID)
	if err != nil {
		t.Fatalf("countNonTerminalFeatures() error = %v", err)
	}
	if count != 2 {
		t.Errorf("countNonTerminalFeatures() = %d, want 2", count)
	}
}

func TestCountNonTerminalFeatures_AllTerminal(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-feat-term"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTTERM0001", "cnt-term-done", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTTERM0002", "cnt-term-sup", planID, string(model.FeatureStatusSuperseded))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01CNTTERM0003", "cnt-term-can", planID, string(model.FeatureStatusCancelled))

	count, err := svc.CountNonTerminalFeatures(planID)
	if err != nil {
		t.Fatalf("countNonTerminalFeatures() error = %v", err)
	}
	if count != 0 {
		t.Errorf("countNonTerminalFeatures() = %d, want 0", count)
	}
}

func TestCountNonTerminalFeatures_NoFeatures(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-cnt-feat-none"
	writeTestPlan(t, svc, planID)

	count, err := svc.CountNonTerminalFeatures(planID)
	if err != nil {
		t.Fatalf("countNonTerminalFeatures() error = %v", err)
	}
	if count != 0 {
		t.Errorf("countNonTerminalFeatures() = %d, want 0", count)
	}
}

// --- MaybeAutoAdvanceFeature ---

func TestMaybeAutoAdvanceFeature_DevelopingAllTasksDone(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-feat-dev"
	writeTestPlan(t, svc, planID)

	featID := "FEAT-01ADVDEV00001"
	writeTestFeatureWithStatus(t, svc, featID, "adv-feat-dev", planID,
		string(model.FeatureStatusDeveloping))

	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "task one", Slug: "adv-dev-t1", ParentFeature: featID, Summary: "first",
	})
	if err != nil {
		t.Fatalf("CreateTask(1) error = %v", err)
	}
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "task two", Slug: "adv-dev-t2", ParentFeature: featID, Summary: "second",
	})
	if err != nil {
		t.Fatalf("CreateTask(2) error = %v", err)
	}
	makeTaskDone(t, svc, task1.ID, task1.Slug)
	makeTaskDone(t, svc, task2.ID, task2.Slug)

	advanced, err := svc.MaybeAutoAdvanceFeature(featID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvanceFeature() error = %v", err)
	}
	if !advanced {
		t.Errorf("advanced = false, want true (developing + all tasks done)")
	}

	feat, err := svc.Get("feature", featID, "adv-feat-dev")
	if err != nil {
		t.Fatalf("Get feature after advance error = %v", err)
	}
	if got := feat.State["status"]; got != string(model.FeatureStatusReviewing) {
		t.Errorf("feature status = %q, want %q", got, model.FeatureStatusReviewing)
	}
}

func TestMaybeAutoAdvanceFeature_NeedsReworkAllTasksDone(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-feat-rw"
	writeTestPlan(t, svc, planID)

	featID := "FEAT-01ADVREWORK01"
	writeTestFeatureWithStatus(t, svc, featID, "adv-feat-rw", planID,
		string(model.FeatureStatusNeedsRework))

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "task one", Slug: "adv-rw-t1", ParentFeature: featID, Summary: "first",
	})
	if err != nil {
		t.Fatalf("CreateTask error = %v", err)
	}
	makeTaskDone(t, svc, task.ID, task.Slug)

	advanced, err := svc.MaybeAutoAdvanceFeature(featID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvanceFeature() error = %v", err)
	}
	if !advanced {
		t.Errorf("advanced = false, want true (needs-rework + all tasks done)")
	}

	feat, err := svc.Get("feature", featID, "adv-feat-rw")
	if err != nil {
		t.Fatalf("Get feature after advance error = %v", err)
	}
	if got := feat.State["status"]; got != string(model.FeatureStatusReviewing) {
		t.Errorf("feature status = %q, want %q", got, model.FeatureStatusReviewing)
	}
}

func TestMaybeAutoAdvanceFeature_OneTaskActive(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-feat-act"
	writeTestPlan(t, svc, planID)

	featID := "FEAT-01ADVDEVACT01"
	writeTestFeatureWithStatus(t, svc, featID, "adv-feat-act", planID,
		string(model.FeatureStatusDeveloping))

	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "done task", Slug: "adv-act-t1", ParentFeature: featID, Summary: "done",
	})
	if err != nil {
		t.Fatalf("CreateTask(done) error = %v", err)
	}
	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "active task", Slug: "adv-act-t2", ParentFeature: featID, Summary: "active",
	})
	if err != nil {
		t.Fatalf("CreateTask(active) error = %v", err)
	}
	makeTaskDone(t, svc, task1.ID, task1.Slug)
	// Walk task2 to active: queued → ready → active
	for _, next := range []string{"ready", "active"} {
		_, err = svc.UpdateStatus(UpdateStatusInput{
			Type: "task", ID: task2.ID, Slug: task2.Slug, Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(task2 → %s) error = %v", next, err)
		}
	}

	advanced, err := svc.MaybeAutoAdvanceFeature(featID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvanceFeature() error = %v", err)
	}
	if advanced {
		t.Errorf("advanced = true, want false (one task still active)")
	}
}

func TestMaybeAutoAdvanceFeature_AllTasksNotPlanned(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-feat-np"
	writeTestPlan(t, svc, planID)

	featID := "FEAT-01ADVDEVNP001"
	writeTestFeatureWithStatus(t, svc, featID, "adv-feat-np", planID,
		string(model.FeatureStatusDeveloping))

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "np task", Slug: "adv-np-t1", ParentFeature: featID, Summary: "np",
	})
	if err != nil {
		t.Fatalf("CreateTask error = %v", err)
	}
	makeTaskNotPlanned(t, svc, task.ID, task.Slug)

	advanced, err := svc.MaybeAutoAdvanceFeature(featID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvanceFeature() error = %v", err)
	}
	if advanced {
		t.Errorf("advanced = true, want false (all tasks not-planned, no done tasks)")
	}
}

func TestMaybeAutoAdvanceFeature_AlreadyReviewing(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-feat-rev"
	writeTestPlan(t, svc, planID)

	featID := "FEAT-01ADVREVIEW01"
	writeTestFeatureWithStatus(t, svc, featID, "adv-feat-rev", planID,
		string(model.FeatureStatusReviewing))

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "done task", Slug: "adv-rev-t1", ParentFeature: featID, Summary: "done",
	})
	if err != nil {
		t.Fatalf("CreateTask error = %v", err)
	}
	makeTaskDone(t, svc, task.ID, task.Slug)

	advanced, err := svc.MaybeAutoAdvanceFeature(featID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvanceFeature() error = %v", err)
	}
	if advanced {
		t.Errorf("advanced = true, want false (feature already reviewing)")
	}
}

// --- MaybeAutoAdvancePlan ---

func TestMaybeAutoAdvancePlan_AllFeaturesDone(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-plandone"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01ADVPLDONE01", "adv-pl-done-one", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01ADVPLDONE02", "adv-pl-done-two", planID, string(model.FeatureStatusDone))

	advanced, err := svc.MaybeAutoAdvancePlan(planID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvancePlan() error = %v", err)
	}
	if !advanced {
		t.Errorf("advanced = false, want true (active plan + all features done)")
	}

	plan, err := svc.GetPlan(planID)
	if err != nil {
		t.Fatalf("GetPlan after advance error = %v", err)
	}
	if got := plan.State["status"]; got != string(model.PlanStatusDone) {
		t.Errorf("plan status = %q, want %q", got, model.PlanStatusDone)
	}
}

func TestMaybeAutoAdvancePlan_AllFeaturesSuperseded(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-plansup"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01ADVPLSUP001", "adv-pl-sup", planID, string(model.FeatureStatusSuperseded))

	advanced, err := svc.MaybeAutoAdvancePlan(planID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvancePlan() error = %v", err)
	}
	if advanced {
		t.Errorf("advanced = true, want false (all features superseded, none done)")
	}
}

func TestMaybeAutoAdvancePlan_OneFeatureDeveloping(t *testing.T) {
	t.Parallel()

	svc := newTestEntityService(t.TempDir(), "2026-03-19T12:00:00Z")
	planID := "P1-adv-plandev"
	writeTestPlan(t, svc, planID)

	writeTestFeatureWithStatus(t, svc,
		"FEAT-01ADVPLDEV001", "adv-pl-dev-done", planID, string(model.FeatureStatusDone))
	writeTestFeatureWithStatus(t, svc,
		"FEAT-01ADVPLDEV002", "adv-pl-dev-dev", planID, string(model.FeatureStatusDeveloping))

	advanced, err := svc.MaybeAutoAdvancePlan(planID)
	if err != nil {
		t.Fatalf("MaybeAutoAdvancePlan() error = %v", err)
	}
	if advanced {
		t.Errorf("advanced = true, want false (one feature still developing)")
	}
}
