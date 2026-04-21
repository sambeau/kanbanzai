package service

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
)

// makeFeatureForAdvance creates a Feature with the given ID, slug, status, and
// optional document references. It returns a pointer suitable for passing to
// AdvanceFeatureStatus.
func makeFeatureForAdvance(id, slug, parent, status string) *model.Feature {
	return &model.Feature{
		ID:     id,
		Slug:   slug,
		Parent: parent,
		Status: model.FeatureStatus(status),
	}
}

// setupAdvanceServices creates an EntityService and DocumentService backed by
// temporary directories. Returns stateRoot and repoRoot for further setup.
func setupAdvanceServices(t *testing.T) (stateRoot, repoRoot string, entitySvc *EntityService, docSvc *DocumentService) {
	t.Helper()
	stateRoot = t.TempDir()
	repoRoot = t.TempDir()
	entitySvc = NewEntityService(stateRoot)
	docSvc = NewDocumentService(stateRoot, repoRoot)
	return
}

// writeFeatureEntity writes a feature entity record to disk so that
// UpdateStatus can load and persist it.
func writeFeatureEntity(t *testing.T, stateRoot, id, slug, parent, status string, extras map[string]any) {
	t.Helper()
	fields := makeFeatureFields(id, slug, parent, status, nil)
	for k, v := range extras {
		fields[k] = v
	}
	writeTestEntity(t, stateRoot, "feature", id, slug, fields)
}

// TestAdvanceFeatureStatus_FullAdvance verifies that a feature can be advanced
// from proposed all the way to developing when all gate prerequisites are met.
// With the new gate semantics, every transition is gate-checked:
//   - proposed→designing: no gate
//   - designing→specifying: design doc required
//   - specifying→dev-planning: spec doc required
//   - dev-planning→developing: dev-plan doc + child task required
func TestAdvanceFeatureStatus_FullAdvance(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA01"
	slug := "full-advance"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/full.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/full.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/full.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA01", "full-task",
		makeTaskFields("T-01AAAAAAAAA01", "full-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}
	if len(result.OverriddenGates) != 0 {
		t.Errorf("OverriddenGates = %v, want empty", result.OverriddenGates)
	}

	wantThrough := []string{"designing", "specifying", "dev-planning", "developing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "developing" {
		t.Errorf("persisted status = %q, want %q", s, "developing")
	}
}

// TestAdvanceFeatureStatus_PartialAdvance_StopsAtSpecifying verifies that
// advance stops at specifying when only a design doc is present (no spec doc).
//
// With new gate semantics:
//   - proposed→designing: no gate → enters designing
//   - designing→specifying: design doc required → passes (doc present)
//   - specifying→dev-planning: spec doc required → FAILS (no spec doc)
//
// So the feature reaches specifying before stopping.
func TestAdvanceFeatureStatus_PartialAdvance_StopsAtSpecifying(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA02"
	slug := "partial-advance"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Only a design doc — no spec doc.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/partial.md", "design", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reaches specifying (design doc gate passes), stops there (no spec doc).
	if result.FinalStatus != "specifying" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "specifying")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}

	wantThrough := []string{"designing", "specifying"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "specifying" {
		t.Errorf("persisted status = %q, want %q", s, "specifying")
	}
}

func TestAdvanceFeatureStatus_TargetAlreadyReached(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA03"
	slug := "already-there"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "designing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "designing")

	result, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "designing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "designing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}
	if len(result.AdvancedThrough) != 0 {
		t.Errorf("AdvancedThrough = %v, want empty", result.AdvancedThrough)
	}
}

func TestAdvanceFeatureStatus_TargetBehindCurrent(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA04"
	slug := "behind"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "specifying", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "specifying")

	_, err := AdvanceFeatureStatus(feature, "proposed", entitySvc, docSvc, false, "", nil)
	if err == nil {
		t.Fatal("expected error for backward advance, got nil")
	}
}

// TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing verifies that advance
// stops at reviewing when require_human_review is true, even when targeting done.
// The task is in terminal state so developing→reviewing passes. This tests AC-02.
func TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA05"
	slug := "done-blocked"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/done.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/done.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/done.md", "dev-plan", featureID, true)
	// Task must be in a terminal state so the developing→reviewing gate passes.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA05", "done-task",
		makeTaskFields("T-01AAAAAAAAA05", "done-task", featureID, "done", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", &AdvanceConfig{
		RequiresHumanReview: func() bool { return true },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should stop at reviewing — require_human_review is true, never auto-advances past.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for review halt")
	}

	wantThrough := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

// TestAdvanceFeatureStatus_SingleStep verifies a one-step advance from proposed
// to designing. proposed→designing has no gate, so the design doc is not needed.
func TestAdvanceFeatureStatus_SingleStep(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA06"
	slug := "single-step"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "designing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "designing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "designing" {
		t.Errorf("AdvancedThrough = %v, want [designing]", result.AdvancedThrough)
	}
}

// TestAdvanceFeatureStatus_AllGatesUnsatisfied verifies that with no documents,
// advance from proposed to developing enters designing (no gate) but then stops
// because designing→specifying requires a design document.
func TestAdvanceFeatureStatus_AllGatesUnsatisfied(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA07"
	slug := "all-blocked"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// proposed→designing has no gate, so we enter designing.
	// designing→specifying requires a design doc which is absent, so we stop.
	if result.FinalStatus != "designing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "designing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "designing" {
		t.Errorf("AdvancedThrough = %v, want [designing]", result.AdvancedThrough)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "designing" {
		t.Errorf("persisted status = %q, want %q", s, "designing")
	}
}

// TestAdvanceFeatureStatus_AdvanceFromDeveloping_ToDone_StopsAtReviewing
// verifies that advance from developing to done stops at reviewing even without
// tasks (zero tasks → developing→reviewing gate passes vacuously).
func TestAdvanceFeatureStatus_AdvanceFromDeveloping_ToDone_StopsAtReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA08"
	slug := "dev-to-done"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// developing→reviewing: no tasks → vacuously terminal → gate passes.
	// reviewing is the mandatory halt state, so advance stops there.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for review halt")
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "reviewing" {
		t.Errorf("AdvancedThrough = %v, want [reviewing]", result.AdvancedThrough)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

// TestAdvanceFeatureStatus_AdvanceToReviewing_IsTarget verifies that when
// reviewing is the explicit target, advance transitions to it normally with
// no StoppedReason (the stop-state halt only fires when advancing past it).
func TestAdvanceFeatureStatus_AdvanceToReviewing_IsTarget(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA18"
	slug := "dev-to-rev"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "reviewing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// developing→reviewing: no tasks → vacuously passes. Reviewing is target → no StoppedReason.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty (target was reached)", result.StoppedReason)
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "reviewing" {
		t.Errorf("AdvancedThrough = %v, want [reviewing]", result.AdvancedThrough)
	}
}

// TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone verifies that a feature
// in reviewing can be advanced to done when a review report document exists.
func TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA20"
	slug := "rev-to-done"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "reviewing", nil)

	// reviewing→done requires a review report document.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/review.md", "report", featureID, false)

	feature := makeFeatureForAdvance(featureID, slug, parent, "reviewing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "done" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "done")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty (target was reached)", result.StoppedReason)
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "done" {
		t.Errorf("AdvancedThrough = %v, want [done]", result.AdvancedThrough)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "done" {
		t.Errorf("persisted status = %q, want %q", s, "done")
	}
}

// TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone_NoReport verifies that
// reviewing→done is gated: without a review report the advance stops.
func TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone_NoReport(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA21"
	slug := "rev-no-report"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "reviewing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "reviewing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// reviewing→done gate fails without a review report.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason (missing review report)")
	}
	if len(result.AdvancedThrough) != 0 {
		t.Errorf("AdvancedThrough = %v, want empty", result.AdvancedThrough)
	}
}

// TestAdvanceFeatureStatus_NeverAutoTransitionsThroughReviewing verifies that
// advance never auto-transitions through reviewing to reach done, even with all
// prerequisites satisfied. Tasks must be terminal for developing→reviewing.
func TestAdvanceFeatureStatus_NeverAutoTransitionsThroughReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA19"
	slug := "no-skip-review"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/noskip.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/noskip.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/noskip.md", "dev-plan", featureID, true)
	// Task must be terminal so developing→reviewing gate passes.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA19", "noskip-task",
		makeTaskFields("T-01AAAAAAAAA19", "noskip-task", featureID, "done", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must stop at reviewing, never reaching done.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.FinalStatus == "done" {
		t.Fatal("advance must never auto-transition through reviewing to done")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for mandatory review halt")
	}

	wantThrough := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}
}

func TestAdvanceFeatureStatus_InvalidCurrentStatus(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	_ = stateRoot

	feature := makeFeatureForAdvance("FEAT-01AAAAAAAAA09", "invalid-current", "", "draft")

	_, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc, false, "", nil)
	if err == nil {
		t.Fatal("expected error for non-forward-path current status, got nil")
	}
}

func TestAdvanceFeatureStatus_InvalidTargetStatus(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	_ = stateRoot

	feature := makeFeatureForAdvance("FEAT-01AAAAAAAAA10", "invalid-target", "", "proposed")

	_, err := AdvanceFeatureStatus(feature, "in-progress", entitySvc, docSvc, false, "", nil)
	if err == nil {
		t.Fatal("expected error for non-forward-path target status, got nil")
	}
}

// TestAdvanceFeatureStatus_PartialAdvance_StopsAtDevPlanning verifies that with
// design and spec docs but no dev-plan, advance stops at dev-planning.
//
// New gate semantics:
//   - proposed→designing: no gate → enters designing
//   - designing→specifying: design doc → passes → enters specifying
//   - specifying→dev-planning: spec doc → passes → enters dev-planning
//   - dev-planning→developing: dev-plan doc required → FAILS (no dev-plan)
func TestAdvanceFeatureStatus_PartialAdvance_StopsAtDevPlanning(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA11"
	slug := "stops-at-devplan"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/devplan.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/devplan.md", "specification", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Enters designing, specifying, dev-planning; fails at dev-planning→developing.
	if result.FinalStatus != "dev-planning" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "dev-planning")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}

	wantThrough := []string{"designing", "specifying", "dev-planning"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}
}

// TestAdvanceFeatureStatus_PartialAdvance_StopsAtDeveloping verifies that with
// all documents but no child tasks, the dev-planning→developing gate fails
// because it requires both an approved dev-plan AND at least one child task.
func TestAdvanceFeatureStatus_PartialAdvance_StopsAtDeveloping(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA12"
	slug := "stops-at-developing"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// All documents present, but no child tasks.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/notasks.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/notasks.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/notasks.md", "dev-plan", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dev-planning→developing requires dev-plan doc (✓) + child task (✗) → stops at dev-planning.
	if result.FinalStatus != "dev-planning" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "dev-planning")
	}
	if result.StoppedReason == "" {
		t.Errorf("expected non-empty StoppedReason (no tasks)")
	}

	wantThrough := []string{"designing", "specifying", "dev-planning"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
}

// TestAdvanceFeatureStatus_EachStepPersisted verifies that each state is
// persisted to disk as advance proceeds. Uses design+spec (no dev-plan) so
// advance stops at dev-planning after persisting designing and specifying.
func TestAdvanceFeatureStatus_EachStepPersisted(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA13"
	slug := "each-persisted"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/each.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/each.md", "specification", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Enters designing, specifying, dev-planning; fails at dev-planning→developing.
	if result.FinalStatus != "dev-planning" {
		t.Fatalf("FinalStatus = %q, want %q", result.FinalStatus, "dev-planning")
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "dev-planning" {
		t.Errorf("persisted status = %q, want %q", s, "dev-planning")
	}
}

// TestAdvanceFeatureStatus_MidPathStart verifies that advance works correctly
// when starting from a mid-path state (specifying).
// Requires spec doc (for specifying→dev-planning) and dev-plan+task (for dev-planning→developing).
func TestAdvanceFeatureStatus_MidPathStart(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA14"
	slug := "mid-start"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "specifying", nil)

	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/mid.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/mid.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA14", "mid-task",
		makeTaskFields("T-01AAAAAAAAA14", "mid-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "specifying")
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}

	wantThrough := []string{"dev-planning", "developing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}
}

// TestAdvanceFeatureStatus_ParentPlanDocs verifies that document gates are
// satisfied by documents owned by the parent plan.
// Gate: designing→specifying requires a design doc; it is owned by the parent plan.
func TestAdvanceFeatureStatus_ParentPlanDocs(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA15"
	slug := "parent-docs"
	parent := "P1-parent-docs"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Design doc owned by the parent plan — satisfies designing→specifying gate.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/parent.md", "design", parent, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "specifying", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// proposed→designing: no gate. designing→specifying: design doc (parent) → passes.
	if result.FinalStatus != "specifying" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "specifying")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}

	wantThrough := []string{"designing", "specifying"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}
}

// ─── Override tests ───────────────────────────────────────────────────────────

// TestAdvanceFeatureStatus_Override_BypassesAllGates verifies that advance with
// override=true and a non-empty reason bypasses all failing gates (FR-016).
func TestAdvanceFeatureStatus_Override_BypassesAllGates(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB01"
	slug := "override-all"
	parent := "P1-test-plan"

	// No documents, no tasks — all gates will fail without override.
	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, true, "fast-track for demo", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With override, all gates are bypassed — advance reaches the target.
	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty (override bypasses gates)", result.StoppedReason)
	}

	wantThrough := []string{"designing", "specifying", "dev-planning", "developing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
}

// TestAdvanceFeatureStatus_Override_LogsOverrideRecords verifies that each
// bypassed gate produces a separate OverrideRecord on the feature (FR-016).
func TestAdvanceFeatureStatus_Override_LogsOverrideRecords(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB02"
	slug := "override-log"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	reason := "skipping for integration test"

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, true, reason, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// proposed→designing: no gate (not overridden)
	// designing→specifying: OVERRIDDEN
	// specifying→dev-planning: OVERRIDDEN
	// dev-planning→developing: OVERRIDDEN
	wantOverridden := []string{
		"designing→specifying",
		"specifying→dev-planning",
		"dev-planning→developing",
	}
	if len(result.OverriddenGates) != len(wantOverridden) {
		t.Fatalf("OverriddenGates = %v, want %v", result.OverriddenGates, wantOverridden)
	}
	for i, want := range wantOverridden {
		if result.OverriddenGates[i] != want {
			t.Errorf("OverriddenGates[%d] = %q, want %q", i, result.OverriddenGates[i], want)
		}
	}

	// The feature struct should have override records appended.
	if len(feature.Overrides) != len(wantOverridden) {
		t.Fatalf("feature.Overrides len = %d, want %d", len(feature.Overrides), len(wantOverridden))
	}
	for i, o := range feature.Overrides {
		if o.Reason != reason {
			t.Errorf("Overrides[%d].Reason = %q, want %q", i, o.Reason, reason)
		}
		if o.Timestamp.IsZero() {
			t.Errorf("Overrides[%d].Timestamp is zero", i)
		}
	}
}

// TestAdvanceFeatureStatus_Override_PersistedOnDisk verifies that override
// records are written to the feature entity on disk (FR-014).
func TestAdvanceFeatureStatus_Override_PersistedOnDisk(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB03"
	slug := "override-persist"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "designing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "designing")

	_, err := AdvanceFeatureStatus(feature, "specifying", entitySvc, docSvc, true, "external spec exists", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the entity back from disk and verify the overrides field is present.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}

	overridesRaw, ok := got.State["overrides"]
	if !ok {
		t.Fatal("expected 'overrides' field in persisted feature state")
	}
	overrides, ok := overridesRaw.([]any)
	if !ok || len(overrides) == 0 {
		t.Fatalf("expected non-empty overrides slice in persisted state, got %T %v", overridesRaw, overridesRaw)
	}

	override, ok := overrides[0].(map[string]any)
	if !ok {
		t.Fatalf("expected override[0] to be map[string]any, got %T", overrides[0])
	}
	if override["from_status"] != "designing" {
		t.Errorf("override.from_status = %q, want %q", override["from_status"], "designing")
	}
	if override["to_status"] != "specifying" {
		t.Errorf("override.to_status = %q, want %q", override["to_status"], "specifying")
	}
	if override["reason"] != "external spec exists" {
		t.Errorf("override.reason = %q, want %q", override["reason"], "external spec exists")
	}
}

// TestAdvanceFeatureStatus_Override_StopStatePreserved verifies that the
// reviewing stop state is preserved even with override=true (NFR-005).
func TestAdvanceFeatureStatus_Override_StopStatePreserved(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB04"
	slug := "override-stop"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, true, "override all gates", &AdvanceConfig{
		RequiresHumanReview: func() bool { return true },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Override bypasses gate failures but reviewing is a mandatory halt — never skipped.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q (stop state preserved with override)", result.FinalStatus, "reviewing")
	}
	if result.FinalStatus == "done" {
		t.Fatal("override must not skip the reviewing stop state")
	}
	if result.StoppedReason == "" {
		t.Error("expected StoppedReason for reviewing halt")
	}
}

// TestAdvanceFeatureStatus_Override_NoOverriddenGatesWhenGatesPassed verifies
// that OverriddenGates is empty when all gates are satisfied without override.
func TestAdvanceFeatureStatus_Override_NoOverriddenGatesWhenGatesPassed(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB05"
	slug := "no-override-needed"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/clean.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/clean.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/clean.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01BBBBBBBBBB05", "clean-task",
		makeTaskFields("T-01BBBBBBBBBB05", "clean-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc, true, "just in case", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}
	// All gates were satisfied — no overrides logged.
	if len(result.OverriddenGates) != 0 {
		t.Errorf("OverriddenGates = %v, want empty (all gates satisfied)", result.OverriddenGates)
	}
	if len(feature.Overrides) != 0 {
		t.Errorf("feature.Overrides = %v, want empty (no gates bypassed)", feature.Overrides)
	}
}

// TestAdvanceFeatureStatus_Override_TimestampSetOnRecord verifies that override
// records have a non-zero timestamp close to now.
func TestAdvanceFeatureStatus_Override_TimestampSetOnRecord(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01BBBBBBBBBB06"
	slug := "override-ts"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "designing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "designing")

	before := time.Now()
	_, err := AdvanceFeatureStatus(feature, "specifying", entitySvc, docSvc, true, "ts test", nil)
	after := time.Now()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(feature.Overrides) == 0 {
		t.Fatal("expected at least one override record")
	}
	ts := feature.Overrides[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("override timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

// ─── Auto-advance reviewing tests (AC-02, AC-04 through AC-07) ───────────────

// TestAdvanceFeatureStatus_RequireHumanReview_True_Halts verifies that when
// RequiresHumanReview returns true the advance halts at reviewing regardless
// of task verification status (AC-02).
func TestAdvanceFeatureStatus_RequireHumanReview_True_Halts(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01EEEEEEEEEE01"
	slug := "rhr-true-halts"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	// Task is done AND has verification — would otherwise satisfy auto-advance.
	taskFields := makeTaskFields("T-01EEEEEEEEEE01", "verified-task", featureID, "done", nil)
	taskFields["verification"] = "all checks passed"
	writeTestEntity(t, stateRoot, "task", "T-01EEEEEEEEEE01", "verified-task", taskFields)

	// Report doc exists — reviewing→done gate would pass.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/rhr.md", "report", featureID, false)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", &AdvanceConfig{
		RequiresHumanReview: func() bool { return true },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RequiresHumanReview=true overrides all other conditions — halts at reviewing.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason when require_human_review is true")
	}
	if !strings.Contains(result.StoppedReason, "require_human_review") {
		t.Errorf("StoppedReason %q should mention require_human_review", result.StoppedReason)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

// TestAdvanceFeatureStatus_AutoAdvancePastReviewing_AllVerified verifies that
// when RequiresHumanReview is absent and all tasks have recorded verification,
// advance continues past reviewing to done (AC-04).
func TestAdvanceFeatureStatus_AutoAdvancePastReviewing_AllVerified(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01EEEEEEEEEE02"
	slug := "auto-advance-verified"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	// Task is done with verification recorded.
	taskFields := makeTaskFields("T-01EEEEEEEEEE02", "verified-task", featureID, "done", nil)
	taskFields["verification"] = "integration tests passed"
	writeTestEntity(t, stateRoot, "task", "T-01EEEEEEEEEE02", "verified-task", taskFields)

	// Report doc required for reviewing→done gate.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/auto.md", "report", featureID, false)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	// No RequiresHumanReview — advance should auto-proceed past reviewing.
	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "done" {
		t.Errorf("FinalStatus = %q, want %q (should auto-advance past reviewing)", result.FinalStatus, "done")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}

	wantThrough := []string{"reviewing", "done"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	for i, s := range wantThrough {
		if result.AdvancedThrough[i] != s {
			t.Errorf("AdvancedThrough[%d] = %q, want %q", i, result.AdvancedThrough[i], s)
		}
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "done" {
		t.Errorf("persisted status = %q, want %q", s, "done")
	}
}

// TestAdvanceFeatureStatus_HaltsAtReviewing_UnverifiedTask verifies that when
// RequiresHumanReview is absent but a task has no recorded verification, advance
// halts at reviewing with a StoppedReason identifying the unverified task (AC-05).
func TestAdvanceFeatureStatus_HaltsAtReviewing_UnverifiedTask(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01EEEEEEEEEE03"
	slug := "halts-unverified"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	// Task is done but has NO verification field.
	writeTestEntity(t, stateRoot, "task", "T-01EEEEEEEEEE03", "unverified-task",
		makeTaskFields("T-01EEEEEEEEEE03", "unverified-task", featureID, "done", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should halt at reviewing — task lacks verification.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for unverified task")
	}
	if !strings.Contains(result.StoppedReason, "T-01EEEEEEEEEE03") {
		t.Errorf("StoppedReason %q should identify unverified task T-01EEEEEEEEEE03", result.StoppedReason)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

// TestAdvanceFeatureStatus_AutoAdvancePastReviewing_ZeroTasks verifies that a
// feature with no child tasks auto-advances past reviewing when RequiresHumanReview
// is absent (AC-06: vacuous truth — zero tasks satisfy verification).
func TestAdvanceFeatureStatus_AutoAdvancePastReviewing_ZeroTasks(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01EEEEEEEEEE04"
	slug := "auto-advance-zero"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	// No tasks — checkAllTasksHaveVerification returns nil vacuously.

	// Report doc required for reviewing→done gate.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/zero.md", "report", featureID, false)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "done" {
		t.Errorf("FinalStatus = %q, want %q (zero tasks → auto-advance vacuously true)", result.FinalStatus, "done")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}

	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "done" {
		t.Errorf("persisted status = %q, want %q", s, "done")
	}
}

// TestAdvanceFeatureStatus_HaltsAtReviewing_NeedsReviewTask verifies that a task
// in needs-review status blocks the path to reviewing: needs-review is non-terminal
// so the developing→reviewing gate fails, preventing auto-advance consideration (AC-07).
// The unit-level AC-07 behavior (checkAllTasksHaveVerification) is covered by
// TestCheckAllTasksHaveVerification_NeedsReview in prereq_test.go.
func TestAdvanceFeatureStatus_HaltsAtReviewing_NeedsReviewTask(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01EEEEEEEEEE05"
	slug := "needs-review-blocks"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	// Task in needs-review state — non-terminal, blocks developing→reviewing gate.
	writeTestEntity(t, stateRoot, "task", "T-01EEEEEEEEEE05", "nr-task",
		makeTaskFields("T-01EEEEEEEEEE05", "nr-task", featureID, "needs-review", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc, false, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The needs-review task is non-terminal, blocking developing→reviewing.
	if result.FinalStatus == "done" {
		t.Error("advance must not reach done when a needs-review task exists")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}
	if !strings.Contains(result.StoppedReason, "T-01EEEEEEEEEE05") {
		t.Errorf("StoppedReason %q should identify the needs-review task", result.StoppedReason)
	}
}


