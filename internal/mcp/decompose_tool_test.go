package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// setupDecomposeApplyTest creates an EntityService and a plan+feature for
// decompose apply tests. Returns (entitySvc, featureID).
func setupDecomposeApplyTest(t *testing.T) (*service.EntityService, string) {
	t.Helper()
	entitySvc := service.NewEntityService(t.TempDir())
	planID := createEntityTestPlan(t, entitySvc, "decomp-plan")
	featID := createEntityTestFeature(t, entitySvc, planID, "decomp-feat")
	return entitySvc, featID
}

// callDecomposeApply invokes the decomposeApply ActionHandler with a proposal
// containing the given task stubs. Disables the auto-commit side-effect.
// Returns the raw map[string]any result.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func callDecomposeApply(t *testing.T, entitySvc *service.EntityService, featID string, tasks []map[string]any) map[string]any {
	t.Helper()

	savedFn := decomposeCommitFunc
	decomposeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	defer func() { decomposeCommitFunc = savedFn }()

	proposal := map[string]any{
		"tasks":       tasks,
		"total_tasks": len(tasks),
		"slices":      []any{},
		"warnings":    []any{},
	}

	handler := decomposeApply(entitySvc, nil) // nil decomposeSvc: skip skeleton dev-plan
	req := makeRequest(map[string]any{
		"action":     "apply",
		"feature_id": featID,
		"proposal":   proposal,
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("decomposeApply error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any result, got %T", result)
	}
	return m
}

// proposedTask builds a minimal ProposedTask map suitable for use in proposals.
func proposedTask(slug string) map[string]any {
	return map[string]any{
		"slug":      slug,
		"name":      "Task " + slug,
		"summary":   "Summary for " + slug,
		"rationale": "Rationale for " + slug,
	}
}

// advanceTaskToDone transitions a task from queued → ready → active → done.
func advanceTaskToDone(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string) {
	t.Helper()
	for _, next := range []string{"ready", "active", "done"} {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     taskID,
			Slug:   taskSlug,
			Status: next,
		}); err != nil {
			t.Fatalf("advanceTaskToDone: advance %s to %s: %v", taskID, next, err)
		}
	}
}

// advanceTaskToNeedsRework transitions a task from queued → ready → active → needs-rework.
func advanceTaskToNeedsRework(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string) {
	t.Helper()
	for _, next := range []string{"ready", "active", "needs-rework"} {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     taskID,
			Slug:   taskSlug,
			Status: next,
		}); err != nil {
			t.Fatalf("advanceTaskToNeedsRework: advance %s to %s: %v", taskID, next, err)
		}
	}
}

// countDecomposeTasksByStatus counts tasks belonging to featID grouped by status.
func countDecomposeTasksByStatus(t *testing.T, entitySvc *service.EntityService, featID string) map[string]int {
	t.Helper()
	tasks, err := entitySvc.List("task")
	if err != nil {
		t.Fatalf("List tasks: %v", err)
	}
	counts := make(map[string]int)
	for _, tk := range tasks {
		if pf, _ := tk.State["parent_feature"].(string); pf != featID {
			continue
		}
		status, _ := tk.State["status"].(string)
		counts[status]++
	}
	return counts
}

// ─── Supersession pass unit tests ─────────────────────────────────────────────

// TestDecomposeApply_AC001_AllQueuedSuperseded covers AC-001:
// Feature with 5 queued tasks → all 5 superseded, superseded_count: 5,
// no warning field.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC001_AllQueuedSuperseded(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	for i := 0; i < 5; i++ {
		createEntityTestTask(t, entitySvc, featID, fmt.Sprintf("old-task-%d", i))
	}

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("new-task-1")})

	if sc := resp["superseded_count"]; sc != 5 {
		t.Errorf("superseded_count = %v, want 5", sc)
	}

	if _, ok := resp["warning"]; ok {
		t.Errorf("unexpected warning field: %v", resp["warning"])
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["not-planned"] != 5 {
		t.Errorf("not-planned tasks = %d, want 5 (all status counts: %v)", counts["not-planned"], counts)
	}
}

// TestDecomposeApply_AC002_NoExistingTasks covers AC-002:
// Feature with no existing tasks → superseded_count: 0, no warning field.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC002_NoExistingTasks(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("first-task")})

	sc, ok := resp["superseded_count"]
	if !ok {
		t.Fatal("superseded_count field missing from response")
	}
	if sc != 0 {
		t.Errorf("superseded_count = %v, want 0", sc)
	}

	if _, ok := resp["warning"]; ok {
		t.Errorf("unexpected warning field: %v", resp["warning"])
	}
}

// TestDecomposeApply_AC003_DonePlusQueued covers AC-003:
// Feature with 2 done + 3 queued → done preserved, 3 superseded,
// superseded_count: 3, no warning.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC003_DonePlusQueued(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	for i := 0; i < 2; i++ {
		id, slug := createEntityTestTask(t, entitySvc, featID, fmt.Sprintf("done-task-%d", i))
		advanceTaskToDone(t, entitySvc, id, slug)
	}
	for i := 0; i < 3; i++ {
		createEntityTestTask(t, entitySvc, featID, fmt.Sprintf("queued-task-%d", i))
	}

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("new-task")})

	if sc := resp["superseded_count"]; sc != 3 {
		t.Errorf("superseded_count = %v, want 3", sc)
	}

	if _, ok := resp["warning"]; ok {
		t.Errorf("unexpected warning field: %v", resp["warning"])
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["done"] != 2 {
		t.Errorf("done tasks = %d, want 2 (all status counts: %v)", counts["done"], counts)
	}
	if counts["not-planned"] != 3 {
		t.Errorf("not-planned tasks = %d, want 3 (all status counts: %v)", counts["not-planned"], counts)
	}
}

// TestDecomposeApply_AC004_ActivePlusQueued covers AC-004:
// Feature with 1 active + 3 queued → active preserved, 3 superseded,
// superseded_count: 3, warning present mentioning 1 task.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC004_ActivePlusQueued(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	activeID, activeSlug := createEntityTestTask(t, entitySvc, featID, "active-task")
	advanceToActive(t, entitySvc, activeID, activeSlug)

	for i := 0; i < 3; i++ {
		createEntityTestTask(t, entitySvc, featID, fmt.Sprintf("queued-task-%d", i))
	}

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("new-task")})

	if sc := resp["superseded_count"]; sc != 3 {
		t.Errorf("superseded_count = %v, want 3", sc)
	}

	warnRaw, ok := resp["warning"]
	if !ok {
		t.Fatal("expected warning field in response, got none")
	}
	warnStr, _ := warnRaw.(string)
	const wantWarn = "1 task(s) in active/needs-rework status were preserved; verify they are still needed."
	if warnStr != wantWarn {
		t.Errorf("warning = %q, want %q", warnStr, wantWarn)
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["active"] != 1 {
		t.Errorf("active tasks = %d, want 1 (all status counts: %v)", counts["active"], counts)
	}
}

// TestDecomposeApply_AC005_ReadyTaskPreserved covers AC-005:
// Feature with 1 ready task → ready preserved, superseded_count: 0,
// no warning.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC005_ReadyTaskPreserved(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	readyID, readySlug := createEntityTestTask(t, entitySvc, featID, "ready-task")
	advanceToReady(t, entitySvc, readyID, readySlug)

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("new-task")})

	if sc := resp["superseded_count"]; sc != 0 {
		t.Errorf("superseded_count = %v, want 0", sc)
	}

	if _, ok := resp["warning"]; ok {
		t.Errorf("unexpected warning field: %v", resp["warning"])
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["ready"] != 1 {
		t.Errorf("ready tasks = %d, want 1 (all status counts: %v)", counts["ready"], counts)
	}
}

// TestDecomposeApply_AC006_ActiveDoesNotBlockTaskCreation covers AC-006:
// Feature with 1 active task → new tasks still created (Pass 1 not blocked),
// warning present.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC006_ActiveDoesNotBlockTaskCreation(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	activeID, activeSlug := createEntityTestTask(t, entitySvc, featID, "active-task")
	advanceToActive(t, entitySvc, activeID, activeSlug)

	newProposal := []map[string]any{
		proposedTask("new-task-a"),
		proposedTask("new-task-b"),
	}
	resp := callDecomposeApply(t, entitySvc, featID, newProposal)

	if tc := resp["total_created"]; tc != 2 {
		t.Errorf("total_created = %v, want 2", tc)
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["queued"] != 2 {
		t.Errorf("queued (newly created) tasks = %d, want 2 (all status counts: %v)", counts["queued"], counts)
	}

	if _, ok := resp["warning"]; !ok {
		t.Error("expected warning field in response, got none")
	}
}

// TestDecomposeApply_AC007_IdempotentMultipleCalls covers AC-007:
// Apply called 3× on same feature → after 3rd call, exactly 1 set of
// queued tasks remains (previous sets are not-planned).
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC007_IdempotentMultipleCalls(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	proposal := []map[string]any{
		proposedTask("task-alpha"),
		proposedTask("task-beta"),
	}

	for i := 1; i <= 3; i++ {
		resp := callDecomposeApply(t, entitySvc, featID, proposal)
		if tc := resp["total_created"]; tc != 2 {
			t.Errorf("call %d: total_created = %v, want 2", i, tc)
		}
	}

	// After 3 applies, only the most recent 2 tasks should be queued;
	// the two prior rounds (4 tasks total) should be not-planned.
	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["queued"] != 2 {
		t.Errorf("queued tasks = %d, want 2 (all status counts: %v)", counts["queued"], counts)
	}
	if counts["not-planned"] != 4 {
		t.Errorf("not-planned tasks = %d, want 4 (all status counts: %v)", counts["not-planned"], counts)
	}
}

// TestDecomposeApply_AC008_NeedsReworkPreserved covers AC-008:
// Feature with 2 needs-rework tasks → both preserved, warning mentions 2,
// superseded_count: 0.
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_AC008_NeedsReworkPreserved(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	for i := 0; i < 2; i++ {
		id, slug := createEntityTestTask(t, entitySvc, featID, fmt.Sprintf("rework-task-%d", i))
		advanceTaskToNeedsRework(t, entitySvc, id, slug)
	}

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("new-task")})

	if sc := resp["superseded_count"]; sc != 0 {
		t.Errorf("superseded_count = %v, want 0", sc)
	}

	warnRaw, ok := resp["warning"]
	if !ok {
		t.Fatal("expected warning field in response, got none")
	}
	warnStr, _ := warnRaw.(string)
	const wantWarn = "2 task(s) in active/needs-rework status were preserved; verify they are still needed."
	if warnStr != wantWarn {
		t.Errorf("warning = %q, want %q", warnStr, wantWarn)
	}

	counts := countDecomposeTasksByStatus(t, entitySvc, featID)
	if counts["needs-rework"] != 2 {
		t.Errorf("needs-rework tasks = %d, want 2 (all status counts: %v)", counts["needs-rework"], counts)
	}
}

// TestDecomposeApply_SupplementarySupersededCountAlwaysPresent verifies that
// the superseded_count key is always present in the response, even when 0
// (NFR: consistent response shape).
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_SupplementarySupersededCountAlwaysPresent(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("only-task")})

	if _, ok := resp["superseded_count"]; !ok {
		t.Error("superseded_count field is missing from response when count is 0")
	}
}

// TestDecomposeApply_SupplementaryWarningOmittedNotEmpty verifies that the
// warning key is absent (not set to "") when there are no in-progress tasks
// (NFR-002: clean response when no warning needed).
//
// Not parallel: overrides package-level decomposeCommitFunc.
func TestDecomposeApply_SupplementaryWarningOmittedNotEmpty(t *testing.T) {
	entitySvc, featID := setupDecomposeApplyTest(t)

	resp := callDecomposeApply(t, entitySvc, featID, []map[string]any{proposedTask("only-task")})

	warn, ok := resp["warning"]
	if ok && warn != nil {
		warnStr, isStr := warn.(string)
		if !isStr || strings.TrimSpace(warnStr) != "" {
			t.Errorf("warning should be absent when no in-progress tasks, got %q", warn)
		}
	}
}
