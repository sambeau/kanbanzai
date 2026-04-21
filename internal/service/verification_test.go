package service

import (
	"strings"
	"testing"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// vtSetup creates a fresh service trio for verification-aggregation tests.
func vtSetup(t *testing.T) (*EntityService, *DispatchService) {
	t.Helper()
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)
	return entitySvc, dispatchSvc
}

// vtFeature creates a plan and a feature under it; returns featureID.
func vtFeature(t *testing.T, entitySvc *EntityService, suffix string) string {
	t.Helper()
	planID := "P1-vt-" + suffix
	writeDispatchTestPlan(t, entitySvc, planID)
	result, err := entitySvc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "vt-feat-" + suffix,
		Parent:    planID,
		Summary:   "Feature for verification test",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("vtFeature(%s): %v", suffix, err)
	}
	return result.ID
}

// vtTask creates a task under featID and returns its ID.
func vtTask(t *testing.T, entitySvc *EntityService, featID, suffix string) string {
	t.Helper()
	result, err := entitySvc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featID,
		Slug:          "vt-task-" + suffix,
		Summary:       "Task " + suffix,
	})
	if err != nil {
		t.Fatalf("vtTask(%s): %v", suffix, err)
	}
	return result.ID
}

// vtDone advances a task (queued->ready->active->done) with optional verification text.
func vtDone(t *testing.T, entitySvc *EntityService, dispatchSvc *DispatchService, taskID, verification string) {
	t.Helper()
	if _, err := entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Status: "ready",
	}); err != nil {
		t.Fatalf("vtDone ready %s: %v", taskID, err)
	}
	if _, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "test",
	}); err != nil {
		t.Fatalf("vtDone dispatch %s: %v", taskID, err)
	}
	if _, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:                taskID,
		Summary:               "done",
		VerificationPerformed: verification,
	}); err != nil {
		t.Fatalf("vtDone complete %s: %v", taskID, err)
	}
}

// vtNotPlanned sets a queued task to not-planned (the "wont_do" terminal state).
func vtNotPlanned(t *testing.T, entitySvc *EntityService, taskID string) {
	t.Helper()
	if _, err := entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Status: "not-planned",
	}); err != nil {
		t.Fatalf("vtNotPlanned %s: %v", taskID, err)
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestAggregateTaskVerification_AllPassed: all done tasks have non-empty verification ->
// status "passed", feature entity written (FR-004).
func TestAggregateTaskVerification_AllPassed(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "allpassed")
	t1 := vtTask(t, entitySvc, featID, "ap-1")
	t2 := vtTask(t, entitySvc, featID, "ap-2")
	vtDone(t, entitySvc, dispatchSvc, t1, "ran unit tests, all green")
	vtDone(t, entitySvc, dispatchSvc, t2, "manual smoke test passed")

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "passed" {
		t.Errorf("Status = %q, want %q", result.Status, "passed")
	}
	if !result.Written {
		t.Error("Written = false, want true")
	}
	// Feature entity must have verification_status and verification set.
	feat, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("get feature: %v", err)
	}
	if vs, _ := feat.State["verification_status"].(string); vs != "passed" {
		t.Errorf("feature.verification_status = %q, want %q", vs, "passed")
	}
	if v, _ := feat.State["verification"].(string); v == "" {
		t.Error("feature.verification is empty, want a non-empty summary")
	}
}

// TestAggregateTaskVerification_Partial: mix of done tasks with and without verification ->
// status "partial", feature entity written (FR-004).
func TestAggregateTaskVerification_Partial(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "partial")
	t1 := vtTask(t, entitySvc, featID, "pt-1")
	t2 := vtTask(t, entitySvc, featID, "pt-2")
	vtDone(t, entitySvc, dispatchSvc, t1, "unit tests pass")
	vtDone(t, entitySvc, dispatchSvc, t2, "") // no verification

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "partial" {
		t.Errorf("Status = %q, want %q", result.Status, "partial")
	}
	if !result.Written {
		t.Error("Written = false, want true")
	}
	feat, _ := entitySvc.Get("feature", featID, "")
	if vs, _ := feat.State["verification_status"].(string); vs != "partial" {
		t.Errorf("feature.verification_status = %q, want %q", vs, "partial")
	}
	// Summary must include the placeholder for the empty-verification task (FR-003).
	if v, _ := feat.State["verification"].(string); !strings.Contains(v, "(no verification recorded)") {
		t.Errorf("feature.verification %q should contain placeholder for empty task", v)
	}
}

// TestAggregateTaskVerification_None: all done tasks have empty verification ->
// status "none", feature entity NOT written (FR-004, FR-005).
func TestAggregateTaskVerification_None(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "none")
	t1 := vtTask(t, entitySvc, featID, "nn-1")
	t2 := vtTask(t, entitySvc, featID, "nn-2")
	vtDone(t, entitySvc, dispatchSvc, t1, "")
	vtDone(t, entitySvc, dispatchSvc, t2, "")

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "none" {
		t.Errorf("Status = %q, want %q", result.Status, "none")
	}
	if result.Written {
		t.Error("Written = true, want false (no write when status is none)")
	}
	// Feature entity must NOT have verification_status set.
	feat, _ := entitySvc.Get("feature", featID, "")
	if vs := feat.State["verification_status"]; vs != nil && vs != "" {
		t.Errorf("feature.verification_status = %v, want not set", vs)
	}
}

// TestAggregateTaskVerification_WontDo: all tasks are not-planned (wont_do) ->
// excluded from aggregation, status "none", no write (FR-004, FR-005).
func TestAggregateTaskVerification_WontDo(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "wontdo")
	t1 := vtTask(t, entitySvc, featID, "wd-1")
	t2 := vtTask(t, entitySvc, featID, "wd-2")
	vtNotPlanned(t, entitySvc, t1)
	vtNotPlanned(t, entitySvc, t2)

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "none" {
		t.Errorf("Status = %q, want %q", result.Status, "none")
	}
	if result.Written {
		t.Error("Written = true, want false")
	}
}

// TestAggregateTaskVerification_WontDoExcludedFromSummary: not-planned tasks must
// not appear in the verification summary string (FR-003).
func TestAggregateTaskVerification_WontDoExcludedFromSummary(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "wontdoexcl")
	t1 := vtTask(t, entitySvc, featID, "we-done")
	t2 := vtTask(t, entitySvc, featID, "we-notplanned")
	vtDone(t, entitySvc, dispatchSvc, t1, "unit tests passed")
	vtNotPlanned(t, entitySvc, t2)

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "passed" {
		t.Errorf("Status = %q, want %q (only done tasks counted)", result.Status, "passed")
	}
	feat, _ := entitySvc.Get("feature", featID, "")
	verif, _ := feat.State["verification"].(string)
	if !strings.Contains(verif, t1) {
		t.Errorf("summary %q should contain done task %s", verif, t1)
	}
	if strings.Contains(verif, t2) {
		t.Errorf("summary %q must NOT contain not-planned task %s", verif, t2)
	}
}

// TestAggregateTaskVerification_Overwrites: a pre-existing verification field on the
// feature entity must be replaced by the new aggregated value (not appended).
func TestAggregateTaskVerification_Overwrites(t *testing.T) {
	entitySvc, dispatchSvc := vtSetup(t)
	featID := vtFeature(t, entitySvc, "overwrite")

	// Pre-populate the feature with stale verification data.
	if _, err := entitySvc.UpdateEntity(UpdateEntityInput{
		Type: "feature",
		ID:   featID,
		Fields: map[string]string{
			"verification":        "old stale value",
			"verification_status": "partial",
		},
	}); err != nil {
		t.Fatalf("pre-populate feature: %v", err)
	}

	t1 := vtTask(t, entitySvc, featID, "ow-1")
	vtDone(t, entitySvc, dispatchSvc, t1, "all assertions pass")

	result, err := dispatchSvc.AggregateTaskVerification(featID)
	if err != nil {
		t.Fatalf("AggregateTaskVerification: %v", err)
	}
	if result.Status != "passed" {
		t.Errorf("Status = %q, want %q", result.Status, "passed")
	}

	feat, _ := entitySvc.Get("feature", featID, "")
	verif, _ := feat.State["verification"].(string)
	if verif == "old stale value" {
		t.Error("verification was not overwritten (old value persists)")
	}
	if !strings.Contains(verif, "all assertions pass") {
		t.Errorf("new verification %q should contain the new task verification", verif)
	}
	if vs, _ := feat.State["verification_status"].(string); vs != "passed" {
		t.Errorf("verification_status = %q, want %q", vs, "passed")
	}
}
