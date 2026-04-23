package service

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
)

// AC-011: shortcut transition completes within 2s under local SQLite load -----

func TestPlanShortcutIntegration_AC011_LatencyUnder2s(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac011-plan")

	// Create several features to simulate a realistic plan load.
	for i, status := range []string{
		string(model.FeatureStatusSpecifying),
		string(model.FeatureStatusDeveloping),
		string(model.FeatureStatusReviewing),
		string(model.FeatureStatusDesigning),
		string(model.FeatureStatusProposed),
	} {
		slug := strings.ReplaceAll("ac011-feat-"+status+"-"+string(rune('a'+i)), "/", "-")
		mustCreateFeatureInState(t, svc, planID, slug, status)
	}

	start := time.Now()
	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("UpdatePlanStatus(proposed→active): %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}
	if elapsed > 2*time.Second {
		t.Errorf("UpdatePlanStatus took %v, want ≤ 2s", elapsed)
	}
}

// AC-012: freshly-qualified feature is counted in N (no stale cache) ----------

func TestPlanShortcutIntegration_AC012_FreshStateNoStaleCache(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "ac012-plan")

	// Start with a feature that is only at proposed — shortcut should fail.
	featID, featSlug := mustCreateFeatureInState(t, svc, planID, "ac012-feat", string(model.FeatureStatusProposed))

	_, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err == nil {
		t.Fatal("expected error before feature is post-designing, got nil")
	}
	if !strings.Contains(err.Error(), "proposed → designing") {
		t.Errorf("pre-transition error = %q, want to contain 'proposed → designing'", err.Error())
	}

	// Now transition the feature to specifying — immediately attempt the shortcut.
	if _, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "feature",
		ID:     featID,
		Slug:   featSlug,
		Status: string(model.FeatureStatusDesigning),
	}); err != nil {
		t.Fatalf("proposed→designing feature: %v", err)
	}
	if _, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "feature",
		ID:     featID,
		Slug:   featSlug,
		Status: string(model.FeatureStatusSpecifying),
	}); err != nil {
		t.Fatalf("designing→specifying feature: %v", err)
	}

	result, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("UpdatePlanStatus(proposed→active) after feature transition: %v", err)
	}
	if got := result.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("status = %q, want %q", got, model.PlanStatusActive)
	}

	// Override record must reflect N = 1 (the freshly-qualified feature).
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
	want := "proposed → active shortcut: 1 feature(s) in post-designing state at transition time"
	if reason != want {
		t.Errorf("override reason = %q, want %q", reason, want)
	}
}

// End-to-end: proposed → designing → active path is unaffected ----------------

func TestPlanShortcutIntegration_ProposedDesigningActive_EndToEnd(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	planID, planSlug := mustCreatePlan(t, svc, "e2e-plan")

	// Transition proposed → designing (should work regardless of feature state).
	r1, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusDesigning))
	if err != nil {
		t.Fatalf("proposed→designing: %v", err)
	}
	if got := r1.State["status"]; got != string(model.PlanStatusDesigning) {
		t.Errorf("after proposed→designing: status = %q, want %q", got, model.PlanStatusDesigning)
	}

	// Transition designing → active (no doc gate for plans — should succeed).
	r2, err := svc.UpdatePlanStatus(planID, planSlug, string(model.PlanStatusActive))
	if err != nil {
		t.Fatalf("designing→active: %v", err)
	}
	if got := r2.State["status"]; got != string(model.PlanStatusActive) {
		t.Errorf("after designing→active: status = %q, want %q", got, model.PlanStatusActive)
	}

	// The designing→active path must NOT write any override records.
	if overridesRaw, ok := r2.State["overrides"]; ok {
		if overrides, ok := overridesRaw.([]any); ok && len(overrides) > 0 {
			t.Errorf("expected no override records for designing→active path, got %v", overrides)
		}
	}
}
