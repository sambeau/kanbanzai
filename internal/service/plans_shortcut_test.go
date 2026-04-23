package service

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

// helpers ------------------------------------------------------------------

// mustCreatePlan creates a plan in proposed status and returns its ID and slug.
func mustCreatePlan(t *testing.T, svc *EntityService, slug string) (id, planSlug string) {
	t.Helper()
	result, err := svc.CreatePlan(CreatePlanInput{
		Prefix:    "P",
		Slug:      slug,
		Name:      slug,
		Summary:   "test plan",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreatePlan(%q): %v", slug, err)
	}
	return result.ID, result.Slug
}

// mustCreateFeatureInState creates a feature attached to planID and transitions
// it to the given status. Returns the feature's ID and slug.
func mustCreateFeatureInState(t *testing.T, svc *EntityService, planID, featureSlug, targetStatus string) (id, slug string) {
	t.Helper()
	fr, err := svc.CreateFeature(CreateFeatureInput{
		Parent:    planID,
		Slug:      featureSlug,
		Name:      featureSlug,
		Summary:   "test feature",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%q, parent=%q): %v", featureSlug, planID, err)
	}
	// Walk the feature through each lifecycle step to reach targetStatus.
	transitions := []string{
		string(model.FeatureStatusProposed),
		string(model.FeatureStatusDesigning),
		string(model.FeatureStatusSpecifying),
		string(model.FeatureStatusDevPlanning),
		string(model.FeatureStatusDeveloping),
		string(model.FeatureStatusReviewing),
		string(model.FeatureStatusDone),
	}
	current := string(model.FeatureStatusProposed)
	for _, next := range transitions[1:] {
		if current == targetStatus {
			break
		}
		_, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   "feature",
			ID:     fr.ID,
			Slug:   fr.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(%q → %q): %v", current, next, err)
		}
		current = next
		if current == targetStatus {
			break
		}
	}
	if current != targetStatus {
		t.Fatalf("could not reach feature status %q for %q (stopped at %q)", targetStatus, featureSlug, current)
	}
	return fr.ID, fr.Slug
}

// AC-001: plan with one specifying feature → proposed→active succeeds without override flag --

func TestPlanShortcut_AC001_OneSpecifyingFeature(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac001-plan")
	mustCreateFeatureInState(t, svc, planID, "ac001-feat", string(model.FeatureStatusSpecifying))

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(proposed→active): unexpected error: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}
}

// AC-002: plan with features at specifying, developing, done → transition succeeds -----------

func TestPlanShortcut_AC002_MixedQualifyingFeatures(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac002-plan")
	mustCreateFeatureInState(t, svc, planID, "ac002-feat-a", string(model.FeatureStatusSpecifying))
	mustCreateFeatureInState(t, svc, planID, "ac002-feat-b", string(model.FeatureStatusDeveloping))
	mustCreateFeatureInState(t, svc, planID, "ac002-feat-c", string(model.FeatureStatusDone))

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(proposed→active): unexpected error: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}
}

// AC-003: plan with all features at designing → error returned -----------------------------

func TestPlanShortcut_AC003_AllFeaturesDesigning(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac003-plan")
	mustCreateFeatureInState(t, svc, planID, "ac003-feat-a", string(model.FeatureStatusDesigning))
	mustCreateFeatureInState(t, svc, planID, "ac003-feat-b", string(model.FeatureStatusDesigning))

	_, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AC-004: plan with no features → error contains "proposed → designing" ------------------

func TestPlanShortcut_AC004_NoFeatures(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac004-plan")

	_, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "proposed → designing") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "proposed → designing")
	}
}

// AC-005: plan with all features at proposed → error with directive ----------------------

func TestPlanShortcut_AC005_AllFeaturesProposed(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac005-plan")
	// Features start at proposed by default.
	mustCreateFeatureInState(t, svc, planID, "ac005-feat-a", string(model.FeatureStatusProposed))
	mustCreateFeatureInState(t, svc, planID, "ac005-feat-b", string(model.FeatureStatusProposed))

	_, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "proposed → designing") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "proposed → designing")
	}
}

// AC-006: override record text matches prescribed pattern with N=2 -----------------------

func TestPlanShortcut_AC006_OverrideRecordText_TwoFeatures(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac006-plan")
	mustCreateFeatureInState(t, svc, planID, "ac006-feat-a", string(model.FeatureStatusSpecifying))
	mustCreateFeatureInState(t, svc, planID, "ac006-feat-b", string(model.FeatureStatusDeveloping))

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus: %v", err)
	}

	overridesRaw, ok := result.State["overrides"]
	if !ok {
		t.Fatal("expected 'overrides' field in plan state")
	}
	overrides, ok := overridesRaw.([]any)
	if !ok || len(overrides) == 0 {
		t.Fatalf("expected non-empty overrides slice, got %T %v", overridesRaw, overridesRaw)
	}
	override, ok := overrides[0].(map[string]any)
	if !ok {
		t.Fatalf("expected override[0] to be map[string]any, got %T", overrides[0])
	}
	reason, _ := override["reason"].(string)
	want := "proposed → active shortcut: 2 feature(s) in post-designing state at transition time"
	if reason != want {
		t.Errorf("override reason = %q, want %q", reason, want)
	}
}

// AC-007: system-generated record identifiable by hardcoded prefix -----------------------

func TestPlanShortcut_AC007_OverrideRecordPrefix(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac007-plan")
	mustCreateFeatureInState(t, svc, planID, "ac007-feat", string(model.FeatureStatusSpecifying))

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus: %v", err)
	}

	overridesRaw, _ := result.State["overrides"]
	overrides, ok := overridesRaw.([]any)
	if !ok || len(overrides) == 0 {
		t.Fatal("expected non-empty overrides slice")
	}
	override, ok := overrides[0].(map[string]any)
	if !ok {
		t.Fatalf("expected override[0] to be map[string]any, got %T", overrides[0])
	}
	reason, _ := override["reason"].(string)
	const prefix = "proposed → active shortcut:"
	if !strings.HasPrefix(reason, prefix) {
		t.Errorf("override reason = %q, want prefix %q", reason, prefix)
	}
}

// AC-008: proposed → designing still succeeds for plan with no qualifying features --------

func TestPlanShortcut_AC008_ProposedToDesigning_Regression(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac008-plan")

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusDesigning))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(proposed→designing): unexpected error: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusDesigning) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusDesigning)
	}
}

// AC-009: designing → active still works (no doc gate for plans) -------------------------

func TestPlanShortcut_AC009_DesigningToActive_Regression(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac009-plan")

	// Advance to designing first.
	if _, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusDesigning)); err != nil {
		t.Fatalf("proposed→designing: %v", err)
	}

	// designing → active: no document gate exists for plans, should succeed.
	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(designing→active): unexpected error: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}
}

// AC-010: designing → active with registered design doc — no regression ------------------
// Plans have no document gate for designing→active; registering a design doc must not
// break the transition.

func TestPlanShortcut_AC010_DesigningToActive_WithDesignDoc_Regression(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac010-plan")

	// Advance to designing.
	if _, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusDesigning)); err != nil {
		t.Fatalf("proposed→designing: %v", err)
	}

	// Set a design doc reference on the plan (simulates doc being registered/approved).
	designRef := "P1-ac010-plan/design-doc"
	if _, err := svc.UpdatePlan(UpdatePlanInput{
		ID:     planID,
		Slug:   planSlug,
		Design: &designRef,
	}); err != nil {
		t.Fatalf("UpdatePlan (set design): %v", err)
	}

	// designing → active should still succeed — there is no doc gate for this transition.
	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(designing→active with design doc): unexpected error: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}
}
