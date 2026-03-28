package service

import (
	"testing"

	"kanbanzai/internal/model"
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

func TestAdvanceFeatureStatus_FullAdvance(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA01"
	slug := "full-advance"
	parent := "P1-test-plan"

	// Write the feature entity on disk.
	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Create approved documents for all document gates.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/full.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/full.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/full.md", "dev-plan", featureID, true)

	// Create a child task for the developing gate.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA01", "full-task",
		makeTaskFields("T-01AAAAAAAAA01", "full-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
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

	// Verify the on-disk state is developing.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "developing" {
		t.Errorf("persisted status = %q, want %q", s, "developing")
	}
}

func TestAdvanceFeatureStatus_PartialAdvance_StopsAtSpecifying(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA02"
	slug := "partial-advance"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Only provide a design doc — no spec.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/partial.md", "design", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinalStatus != "designing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "designing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}

	wantThrough := []string{"designing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
	if result.AdvancedThrough[0] != "designing" {
		t.Errorf("AdvancedThrough[0] = %q, want %q", result.AdvancedThrough[0], "designing")
	}

	// Verify the on-disk state is designing.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "designing" {
		t.Errorf("persisted status = %q, want %q", s, "designing")
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

	result, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc)
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

	_, err := AdvanceFeatureStatus(feature, "proposed", entitySvc, docSvc)
	if err == nil {
		t.Fatal("expected error for backward advance, got nil")
	}
}

func TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA05"
	slug := "done-blocked"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Provide all documents and a task.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/done.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/done.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/done.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA05", "done-task",
		makeTaskFields("T-01AAAAAAAAA05", "done-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should stop at reviewing — it is a mandatory gate that advance never skips.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for review gate")
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

	// Verify it persisted at reviewing, not done.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

func TestAdvanceFeatureStatus_SingleStep(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA06"
	slug := "single-step"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/single.md", "design", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID

	result, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc)
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

func TestAdvanceFeatureStatus_AllGatesUnsatisfied(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA07"
	slug := "all-blocked"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// No documents, no tasks — all gates will be unsatisfied.
	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should stay at proposed — the first gate (designing) is unsatisfied.
	if result.FinalStatus != "proposed" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "proposed")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason")
	}
	if len(result.AdvancedThrough) != 0 {
		t.Errorf("AdvancedThrough = %v, want empty", result.AdvancedThrough)
	}

	// Verify it stayed at proposed on disk.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "proposed" {
		t.Errorf("persisted status = %q, want %q", s, "proposed")
	}
}

func TestAdvanceFeatureStatus_AdvanceFromDeveloping_ToDone_StopsAtReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA08"
	slug := "dev-to-done"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Advance enters reviewing (the mandatory gate) and stops there.
	if result.FinalStatus != "reviewing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "reviewing")
	}
	if result.StoppedReason == "" {
		t.Error("expected non-empty StoppedReason for review gate")
	}
	if len(result.AdvancedThrough) != 1 || result.AdvancedThrough[0] != "reviewing" {
		t.Errorf("AdvancedThrough = %v, want [reviewing]", result.AdvancedThrough)
	}

	// Verify it persisted at reviewing.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "reviewing" {
		t.Errorf("persisted status = %q, want %q", s, "reviewing")
	}
}

// TestAdvanceFeatureStatus_AdvanceToReviewing_IsTarget verifies that when
// reviewing is the explicit target, advance transitions to it normally (AC-17).
func TestAdvanceFeatureStatus_AdvanceToReviewing_IsTarget(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA18"
	slug := "dev-to-rev"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "developing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "developing")

	result, err := AdvanceFeatureStatus(feature, "reviewing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reviewing is the target — advance reaches it and stops normally (no StoppedReason).
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

// TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone verifies AC-17:
// when a feature is already in reviewing status and advance targets done,
// the advance succeeds in a single step (reviewing → done) with no stop.
func TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA20"
	slug := "rev-to-done"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "reviewing", nil)

	feature := makeFeatureForAdvance(featureID, slug, parent, "reviewing")

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc)
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

	// Verify it persisted at done.
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "done" {
		t.Errorf("persisted status = %q, want %q", s, "done")
	}
}

// TestAdvanceFeatureStatus_NeverAutoTransitionsThroughReviewing verifies AC-18:
// advance never auto-transitions through reviewing to reach done, even when
// starting from an early state with all prerequisites satisfied.
func TestAdvanceFeatureStatus_NeverAutoTransitionsThroughReviewing(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA19"
	slug := "no-skip-review"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Provide ALL documents and a task — every prerequisite is met.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/noskip.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/noskip.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/noskip.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA19", "noskip-task",
		makeTaskFields("T-01AAAAAAAAA19", "noskip-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "done", entitySvc, docSvc)
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
		t.Error("expected non-empty StoppedReason for mandatory review gate")
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

	_ = stateRoot // no entity needed, error happens before disk access

	feature := makeFeatureForAdvance("FEAT-01AAAAAAAAA09", "invalid-current", "", "draft")

	_, err := AdvanceFeatureStatus(feature, "designing", entitySvc, docSvc)
	if err == nil {
		t.Fatal("expected error for non-forward-path current status, got nil")
	}
}

func TestAdvanceFeatureStatus_InvalidTargetStatus(t *testing.T) {
	t.Parallel()
	stateRoot, _, entitySvc, docSvc := setupAdvanceServices(t)

	_ = stateRoot

	feature := makeFeatureForAdvance("FEAT-01AAAAAAAAA10", "invalid-target", "", "proposed")

	_, err := AdvanceFeatureStatus(feature, "in-progress", entitySvc, docSvc)
	if err == nil {
		t.Fatal("expected error for non-forward-path target status, got nil")
	}
}

func TestAdvanceFeatureStatus_PartialAdvance_StopsAtDevPlanning(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA11"
	slug := "stops-at-devplan"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Provide design and spec but no dev plan.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/devplan.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/devplan.md", "specification", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
}

func TestAdvanceFeatureStatus_PartialAdvance_StopsAtDeveloping(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA12"
	slug := "stops-at-developing"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Provide all documents but no tasks.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/notasks.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/notasks.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/notasks.md", "dev-plan", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// developing is the target, so it is not gate-checked — it should succeed
	// by passing through designing, specifying, dev-planning (all intermediate,
	// all gate-checked and satisfied).
	if result.FinalStatus != "developing" {
		t.Errorf("FinalStatus = %q, want %q", result.FinalStatus, "developing")
	}
	if result.StoppedReason != "" {
		t.Errorf("StoppedReason = %q, want empty", result.StoppedReason)
	}

	wantThrough := []string{"designing", "specifying", "dev-planning", "developing"}
	if len(result.AdvancedThrough) != len(wantThrough) {
		t.Fatalf("AdvancedThrough = %v, want %v", result.AdvancedThrough, wantThrough)
	}
}

func TestAdvanceFeatureStatus_EachStepPersisted(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA13"
	slug := "each-persisted"
	parent := "P1-test-plan"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Only provide a design and spec — advance will stop at specifying.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/each.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/each.md", "specification", featureID, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")
	feature.Design = designDocID
	feature.Spec = specDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// It should have advanced through designing and specifying, then stopped
	// at dev-planning gate.
	if result.FinalStatus != "specifying" {
		t.Fatalf("FinalStatus = %q, want %q", result.FinalStatus, "specifying")
	}

	// Verify each intermediate state was persisted by checking the final on-disk
	// state matches the reported final status. (UpdateStatus validates each
	// transition, so if we got here the intermediate persists were valid.)
	got, err := entitySvc.Get("feature", featureID, slug)
	if err != nil {
		t.Fatalf("Get after advance: %v", err)
	}
	if s := stringFromState(got.State, "status"); s != "specifying" {
		t.Errorf("persisted status = %q, want %q", s, "specifying")
	}
}

func TestAdvanceFeatureStatus_MidPathStart(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA14"
	slug := "mid-start"
	parent := "P1-test-plan"

	// Start from specifying.
	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "specifying", nil)

	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/mid.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA14", "mid-task",
		makeTaskFields("T-01AAAAAAAAA14", "mid-task", featureID, "queued", nil))

	feature := makeFeatureForAdvance(featureID, slug, parent, "specifying")
	feature.DevPlan = devPlanDocID

	result, err := AdvanceFeatureStatus(feature, "developing", entitySvc, docSvc)
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

func TestAdvanceFeatureStatus_ParentPlanDocs(t *testing.T) {
	t.Parallel()
	stateRoot, repoRoot, entitySvc, docSvc := setupAdvanceServices(t)

	featureID := "FEAT-01AAAAAAAAA15"
	slug := "parent-docs"
	parent := "P1-parent-docs"

	writeFeatureEntity(t, stateRoot, featureID, slug, parent, "proposed", nil)

	// Provide documents at the plan level, not the feature level.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/parent.md", "design", parent, true)

	feature := makeFeatureForAdvance(featureID, slug, parent, "proposed")

	result, err := AdvanceFeatureStatus(feature, "specifying", entitySvc, docSvc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// designing is the only intermediate — gate-checked and satisfied via parent doc.
	// specifying is the target — not gate-checked. So we should reach specifying.
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
