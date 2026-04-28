package service

import (
	"testing"
)

// TestFeaturePromotionHook_FiresOnDeveloping verifies AC-005 / FR-001:
// queued tasks are promoted when a feature transitions to developing.
func TestFeaturePromotionHook_FiresOnDeveloping(t *testing.T) {
	t.Parallel()

	svc, featID := setupPromoteTest(t)
	taskID, taskSlug := createTestTask(t, svc, featID, "hook-task", "Hook Task")

	hook := NewFeaturePromotionHook(svc)
	result := hook.OnStatusTransition("feature", featID, "hook-feat", "specifying", "developing", map[string]any{})

	if result != nil {
		t.Errorf("OnStatusTransition() returned %v, want nil", result)
	}

	res, err := svc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	if got := res.State["status"]; got != "ready" {
		t.Errorf("task status = %v, want ready", got)
	}
}

// TestFeaturePromotionHook_NoFireOnOtherStatus verifies FR-002:
// hook does NOT fire when feature transitions to a status other than developing.
func TestFeaturePromotionHook_NoFireOnOtherStatus(t *testing.T) {
	t.Parallel()

	svc, featID := setupPromoteTest(t)
	taskID, taskSlug := createTestTask(t, svc, featID, "hook-task", "Hook Task")

	hook := NewFeaturePromotionHook(svc)

	for _, toStatus := range []string{"specifying", "reviewing", "done", "designing", "dev-planning"} {
		result := hook.OnStatusTransition("feature", featID, "hook-feat", "drafting", toStatus, map[string]any{})
		if result != nil {
			t.Errorf("OnStatusTransition(toStatus=%q) returned %v, want nil", toStatus, result)
		}
	}

	// Task should still be queued — hook must not have fired.
	res, err := svc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	if got := res.State["status"]; got != "queued" {
		t.Errorf("task status = %v, want queued (hook should not have fired)", got)
	}
}

// TestFeaturePromotionHook_NoFireOnNonFeature verifies FR-002:
// hook does NOT fire when a non-feature entity transitions to developing.
func TestFeaturePromotionHook_NoFireOnNonFeature(t *testing.T) {
	t.Parallel()

	svc, featID := setupPromoteTest(t)
	taskID, taskSlug := createTestTask(t, svc, featID, "hook-task", "Hook Task")

	hook := NewFeaturePromotionHook(svc)
	result := hook.OnStatusTransition("task", featID, "some-task", "queued", "developing", map[string]any{})

	if result != nil {
		t.Errorf("OnStatusTransition() returned %v, want nil", result)
	}

	// Task status must be unchanged.
	res, err := svc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	if got := res.State["status"]; got != "queued" {
		t.Errorf("task status = %v, want queued (hook should not have fired)", got)
	}
}

// TestFeaturePromotionHook_ErrorLogged verifies FR-003:
// hook failure is logged and does not propagate (returns nil).
func TestFeaturePromotionHook_ErrorLogged(t *testing.T) {
	t.Parallel()

	svc, _ := setupPromoteTest(t)
	hook := NewFeaturePromotionHook(svc)

	// Use an unknown feature ID so PromoteQueuedTasks returns an error.
	// The hook should still return nil.
	result := hook.OnStatusTransition("feature", "FEAT-NONEXISTENT", "gone", "specifying", "developing", map[string]any{})

	if result != nil {
		t.Errorf("OnStatusTransition() returned %v, want nil even on error", result)
	}
}
