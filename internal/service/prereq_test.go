package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

// Helper to create a document file, submit it, and optionally approve it.
// Returns the document ID.
func submitAndApproveDoc(t *testing.T, docSvc *DocumentService, repoRoot, relPath, docType, owner string, approve bool) string {
	t.Helper()

	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test Document\nContent for "+relPath), 0o644); err != nil {
		t.Fatalf("write doc file: %v", err)
	}

	result, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      relPath,
		Type:      docType,
		Title:     "Test " + docType,
		Owner:     owner,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(%s): %v", relPath, err)
	}

	if approve {
		_, err := docSvc.ApproveDocument(ApproveDocumentInput{
			ID:         result.ID,
			ApprovedBy: "reviewer",
		})
		if err != nil {
			t.Fatalf("ApproveDocument(%s): %v", result.ID, err)
		}
	}

	return result.ID
}

func TestCheckFeatureGate_Designing_FeatureFieldRef(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Submit and approve a design document owned by the feature.
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/my-design.md", "design", "FEAT-01AAAAAAAAA01", true)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA01",
		Design: docID,
		Parent: "P1-test-plan",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected designing gate satisfied, got reason: %s", result.Reason)
	}
	if result.Stage != "designing" {
		t.Errorf("stage = %q, want %q", result.Stage, "designing")
	}
}

func TestCheckFeatureGate_Designing_FeatureOwnedDoc(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Submit and approve a design doc owned by the feature, but don't set the feature's Design field.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/feat-design.md", "design", "FEAT-01AAAAAAAAA02", true)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA02",
		Design: "", // no direct reference
		Parent: "P1-test-plan",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected designing gate satisfied via feature-owned doc, got reason: %s", result.Reason)
	}
}

func TestCheckFeatureGate_Designing_ParentPlanDoc(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Submit and approve a design doc owned by the parent plan.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/plan-design.md", "design", "P1-test-plan", true)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA03",
		Design: "", // no direct reference
		Parent: "P1-test-plan",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected designing gate satisfied via parent plan doc, got reason: %s", result.Reason)
	}
}

func TestCheckFeatureGate_Designing_Unsatisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA04",
		Parent: "P1-test-plan",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected designing gate unsatisfied, but got satisfied")
	}
	if result.Stage != "designing" {
		t.Errorf("stage = %q, want %q", result.Stage, "designing")
	}
}

func TestCheckFeatureGate_Designing_DraftDocNotSufficient(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Submit but do NOT approve a design document.
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/draft-design.md", "design", "FEAT-01AAAAAAAAA05", false)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA05",
		Design: docID,
		Parent: "P1-test-plan",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected designing gate unsatisfied for draft doc, but got satisfied")
	}
}

func TestCheckFeatureGate_Specifying(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/test-spec.md", "specification", "FEAT-01AAAAAAAAA06", true)

	feature := &model.Feature{
		ID:   "FEAT-01AAAAAAAAA06",
		Spec: docID,
	}

	result := CheckFeatureGate("specifying", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected specifying gate satisfied, got reason: %s", result.Reason)
	}
}

func TestCheckFeatureGate_DevPlanning(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/test-plan.md", "dev-plan", "FEAT-01AAAAAAAAA07", true)

	feature := &model.Feature{
		ID:      "FEAT-01AAAAAAAAA07",
		DevPlan: docID,
	}

	result := CheckFeatureGate("dev-planning", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected dev-planning gate satisfied, got reason: %s", result.Reason)
	}
}

func TestCheckFeatureGate_Developing_WithTasks(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01AAAAAAAAA08"

	// Write a task entity with this feature as parent.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA01", "test-task",
		makeTaskFields("T-01AAAAAAAAA01", "test-task", featureID, "queued", nil))

	feature := &model.Feature{
		ID: featureID,
	}

	result := CheckFeatureGate("developing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected developing gate satisfied, got reason: %s", result.Reason)
	}
}

func TestCheckFeatureGate_Developing_NoTasks(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{
		ID: "FEAT-01AAAAAAAAA09",
	}

	result := CheckFeatureGate("developing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected developing gate unsatisfied with no tasks, but got satisfied")
	}
}

func TestCheckFeatureGate_Developing_TaskForDifferentFeature(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Write a task entity belonging to a different feature.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA02", "other-task",
		makeTaskFields("T-01AAAAAAAAA02", "other-task", "FEAT-01ZZZZZZZZZ99", "queued", nil))

	feature := &model.Feature{
		ID: "FEAT-01AAAAAAAAA10",
	}

	result := CheckFeatureGate("developing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected developing gate unsatisfied when tasks belong to another feature")
	}
}

func TestCheckFeatureGate_Reviewing_NeverSkippable(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{
		ID: "FEAT-01AAAAAAAAA11",
	}

	result := CheckFeatureGate("reviewing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected reviewing gate to never be satisfied")
	}
	if result.Stage != "reviewing" {
		t.Errorf("stage = %q, want %q", result.Stage, "reviewing")
	}
}

func TestCheckFeatureGate_UnknownStage(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{
		ID: "FEAT-01AAAAAAAAA12",
	}

	result := CheckFeatureGate("nonexistent", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatalf("expected unknown stage gate to be unsatisfied")
	}
}

func TestCheckFeatureGates_AllSatisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01AAAAAAAAA13"

	// Create approved documents for all three document stages.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/all-design.md", "design", featureID, true)
	specDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/all-spec.md", "specification", featureID, true)
	devPlanDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/all-plan.md", "dev-plan", featureID, true)

	// Create a child task.
	writeTestEntity(t, stateRoot, "task", "T-01AAAAAAAAA03", "all-task",
		makeTaskFields("T-01AAAAAAAAA03", "all-task", featureID, "queued", nil))

	feature := &model.Feature{
		ID:      featureID,
		Design:  designDocID,
		Spec:    specDocID,
		DevPlan: devPlanDocID,
		Parent:  "P1-test-plan",
	}

	results := CheckFeatureGates(feature, docSvc, entitySvc)
	if len(results) != 5 {
		t.Fatalf("expected 5 gate results, got %d", len(results))
	}

	// First four should be satisfied; reviewing should not.
	expectedStages := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	for i, r := range results {
		if r.Stage != expectedStages[i] {
			t.Errorf("results[%d].Stage = %q, want %q", i, r.Stage, expectedStages[i])
		}
		if i < 4 && !r.Satisfied {
			t.Errorf("results[%d] (%s) should be satisfied, reason: %s", i, r.Stage, r.Reason)
		}
		if i == 4 && r.Satisfied {
			t.Errorf("results[%d] (reviewing) should never be satisfied", i)
		}
	}
}

func TestCheckFeatureGates_NoneSatisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA14",
		Parent: "P1-empty-plan",
	}

	results := CheckFeatureGates(feature, docSvc, entitySvc)
	if len(results) != 5 {
		t.Fatalf("expected 5 gate results, got %d", len(results))
	}

	for _, r := range results {
		if r.Satisfied {
			t.Errorf("gate %q should be unsatisfied, but was satisfied", r.Stage)
		}
	}
}

func TestCheckFeatureGates_PartialSatisfaction(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01AAAAAAAAA15"

	// Only create a design document — spec and dev-plan are missing.
	designDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/partial-design.md", "design", featureID, true)

	feature := &model.Feature{
		ID:     featureID,
		Design: designDocID,
		Parent: "P1-test-plan",
	}

	results := CheckFeatureGates(feature, docSvc, entitySvc)

	// designing: satisfied, specifying: not, dev-planning: not, developing: not, reviewing: not
	if !results[0].Satisfied {
		t.Errorf("designing should be satisfied, reason: %s", results[0].Reason)
	}
	if results[1].Satisfied {
		t.Error("specifying should not be satisfied")
	}
	if results[2].Satisfied {
		t.Error("dev-planning should not be satisfied")
	}
	if results[3].Satisfied {
		t.Error("developing should not be satisfied")
	}
	if results[4].Satisfied {
		t.Error("reviewing should not be satisfied")
	}
}

func TestCheckFeatureGate_ParentPlanFallback_AllDocTypes(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	planID := "P1-fallback-plan"

	// Create approved documents at the plan level for all three types.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/plan-fallback-design.md", "design", planID, true)
	submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/plan-fallback-spec.md", "specification", planID, true)
	submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/plan-fallback-devplan.md", "dev-plan", planID, true)

	// Feature has no document fields set, no feature-owned docs.
	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA16",
		Parent: planID,
	}

	tests := []struct {
		stage string
		want  bool
	}{
		{"designing", true},
		{"specifying", true},
		{"dev-planning", true},
	}

	for _, tc := range tests {
		result := CheckFeatureGate(tc.stage, feature, docSvc, entitySvc)
		if result.Satisfied != tc.want {
			t.Errorf("gate %q: satisfied=%v, want %v (reason: %s)", tc.stage, result.Satisfied, tc.want, result.Reason)
		}
	}
}

func TestCheckFeatureGate_NoParent_SkipsParentLookup(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Feature has no parent and no documents.
	feature := &model.Feature{
		ID:     "FEAT-01AAAAAAAAA17",
		Parent: "",
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected designing gate unsatisfied when feature has no parent and no docs")
	}
}

func TestCheckFeatureGate_LookupOrder_FeatureFieldFirst(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01AAAAAAAAA18"
	planID := "P1-lookup-order"

	// Create approved docs at both feature and plan level.
	featureDocID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/feat-prio.md", "design", featureID, true)
	submitAndApproveDoc(t, docSvc, repoRoot, "work/design/plan-prio.md", "design", planID, true)

	// Set the feature's Design field to the feature-owned doc.
	feature := &model.Feature{
		ID:     featureID,
		Design: featureDocID,
		Parent: planID,
	}

	result := CheckFeatureGate("designing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected designing gate satisfied, got reason: %s", result.Reason)
	}

	// Reason should mention the feature field reference, not the plan doc.
	if result.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckFeatureGate_Developing_AnyTaskStatus(t *testing.T) {
	t.Parallel()

	// The developing gate should be satisfied by tasks of any status.
	statuses := []string{
		string(model.TaskStatusQueued),
		string(model.TaskStatusReady),
		string(model.TaskStatusActive),
		string(model.TaskStatusDone),
	}

	for _, status := range statuses {
		status := status
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			stateRoot := t.TempDir()
			repoRoot := t.TempDir()
			docSvc := NewDocumentService(stateRoot, repoRoot)
			entitySvc := NewEntityService(stateRoot)

			featureID := "FEAT-01AAAAAAAAA19"
			writeTestEntity(t, stateRoot, "task", "T-01BBBBBBBBBB01", "status-task",
				makeTaskFields("T-01BBBBBBBBBB01", "status-task", featureID, status, nil))

			feature := &model.Feature{
				ID: featureID,
			}

			result := CheckFeatureGate("developing", feature, docSvc, entitySvc)
			if !result.Satisfied {
				t.Errorf("expected developing gate satisfied with task status %q, reason: %s", status, result.Reason)
			}
		})
	}
}

// ─── CheckTransitionGate tests ───────────────────────────────────────────────

func TestCheckTransitionGate_UngatedTransitions(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)
	feature := &model.Feature{ID: "FEAT-01CCCCCCCCCC01", Parent: "P1-test"}

	ungated := []struct{ from, to string }{
		{"proposed", "designing"},
		{"reviewing", "needs-rework"},
		{"proposed", "superseded"},
		{"designing", "superseded"},
		{"specifying", "cancelled"},
		{"developing", "cancelled"},
		// Phase 1 transitions are ungated
		{"draft", "in-review"},
		{"in-review", "approved"},
		{"approved", "in-progress"},
	}

	for _, tc := range ungated {
		tc := tc
		t.Run(tc.from+"→"+tc.to, func(t *testing.T) {
			t.Parallel()
			result := CheckTransitionGate(tc.from, tc.to, feature, docSvc, entitySvc)
			if !result.Satisfied {
				t.Errorf("expected ungated transition %s→%s to be satisfied, reason: %s", tc.from, tc.to, result.Reason)
			}
		})
	}
}

func TestCheckTransitionGate_DesigningToSpecifying_Satisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC02"
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/design/d2s.md", "design", featureID, true)

	feature := &model.Feature{ID: featureID, Design: docID, Parent: "P1-test"}
	result := CheckTransitionGate("designing", "specifying", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected designing→specifying satisfied, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_DesigningToSpecifying_Unsatisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{ID: "FEAT-01CCCCCCCCCC03", Parent: "P1-test"}
	result := CheckTransitionGate("designing", "specifying", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected designing→specifying unsatisfied with no design doc")
	}
}

func TestCheckTransitionGate_SpecifyingToDevPlanning_Satisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC04"
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/spec/s2d.md", "specification", featureID, true)

	feature := &model.Feature{ID: featureID, Spec: docID}
	result := CheckTransitionGate("specifying", "dev-planning", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected specifying→dev-planning satisfied, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_SpecifyingToDevPlanning_Unsatisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{ID: "FEAT-01CCCCCCCCCC05", Parent: "P1-test"}
	result := CheckTransitionGate("specifying", "dev-planning", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected specifying→dev-planning unsatisfied with no spec doc")
	}
}

func TestCheckTransitionGate_DevPlanningToDeveloping_Satisfied(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC06"
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/dp2dev.md", "dev-plan", featureID, true)
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC01", "dp-task",
		makeTaskFields("T-01CCCCCCCCCC01", "dp-task", featureID, "queued", nil))

	feature := &model.Feature{ID: featureID, DevPlan: docID}
	result := CheckTransitionGate("dev-planning", "developing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected dev-planning→developing satisfied, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_DevPlanningToDeveloping_NoDoc(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC07"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC02", "nodoc-task",
		makeTaskFields("T-01CCCCCCCCCC02", "nodoc-task", featureID, "queued", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("dev-planning", "developing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected dev-planning→developing unsatisfied when dev-plan doc is missing")
	}
}

func TestCheckTransitionGate_DevPlanningToDeveloping_NoTasks(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC08"
	docID := submitAndApproveDoc(t, docSvc, repoRoot, "work/plan/notask.md", "dev-plan", featureID, true)

	feature := &model.Feature{ID: featureID, DevPlan: docID}
	result := CheckTransitionGate("dev-planning", "developing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected dev-planning→developing unsatisfied when no child tasks exist")
	}
}

func TestCheckTransitionGate_DevelopingToReviewing_AllTerminal(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC09"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC03", "done-task",
		makeTaskFields("T-01CCCCCCCCCC03", "done-task", featureID, "done", nil))
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC04", "np-task",
		makeTaskFields("T-01CCCCCCCCCC04", "np-task", featureID, "not-planned", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("developing", "reviewing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected developing→reviewing satisfied with all tasks terminal, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_DevelopingToReviewing_NonTerminalTask(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC10"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC05", "active-task",
		makeTaskFields("T-01CCCCCCCCCC05", "active-task", featureID, "active", nil))
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC06", "done-task2",
		makeTaskFields("T-01CCCCCCCCCC06", "done-task2", featureID, "done", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("developing", "reviewing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected developing→reviewing unsatisfied with non-terminal task")
	}
	if result.Reason == "" {
		t.Fatal("expected non-empty reason identifying the non-terminal task")
	}
}

func TestCheckTransitionGate_DevelopingToReviewing_NoTasks(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{ID: "FEAT-01CCCCCCCCCC11"}
	result := CheckTransitionGate("developing", "reviewing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected developing→reviewing satisfied with no tasks (vacuously), reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_ReviewingToDone_ReportExists(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC12"
	// Register report but do NOT approve — the gate only requires existence.
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/review.md", "report", featureID, false)

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("reviewing", "done", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected reviewing→done satisfied with registered report, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_ReviewingToDone_NoReport(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{ID: "FEAT-01CCCCCCCCCC13"}
	result := CheckTransitionGate("reviewing", "done", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected reviewing→done unsatisfied with no report document")
	}
}

func TestCheckTransitionGate_NeedsReworkToDeveloping_HasNonTerminalTask(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC14"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC07", "rework-task",
		makeTaskFields("T-01CCCCCCCCCC07", "rework-task", featureID, "ready", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("needs-rework", "developing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected needs-rework→developing satisfied with non-terminal task, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_NeedsReworkToDeveloping_AllTerminal(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC15"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC08", "done-rework",
		makeTaskFields("T-01CCCCCCCCCC08", "done-rework", featureID, "done", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("needs-rework", "developing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected needs-rework→developing unsatisfied when all tasks are terminal")
	}
}

func TestCheckTransitionGate_NeedsReworkToReviewing_AllTerminal(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC16"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC09", "done-nr",
		makeTaskFields("T-01CCCCCCCCCC09", "done-nr", featureID, "done", nil))
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC10", "dup-nr",
		makeTaskFields("T-01CCCCCCCCCC10", "dup-nr", featureID, "duplicate", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("needs-rework", "reviewing", feature, docSvc, entitySvc)
	if !result.Satisfied {
		t.Fatalf("expected needs-rework→reviewing satisfied with all tasks terminal, reason: %s", result.Reason)
	}
}

func TestCheckTransitionGate_NeedsReworkToReviewing_NonTerminalTask(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01CCCCCCCCCC17"
	writeTestEntity(t, stateRoot, "task", "T-01CCCCCCCCCC11", "active-nr",
		makeTaskFields("T-01CCCCCCCCCC11", "active-nr", featureID, "active", nil))

	feature := &model.Feature{ID: featureID}
	result := CheckTransitionGate("needs-rework", "reviewing", feature, docSvc, entitySvc)
	if result.Satisfied {
		t.Fatal("expected needs-rework→reviewing unsatisfied with non-terminal task")
	}
}

// ─── B-12: reviewing→needs-rework cap-check branch ───────────────────────────

// TestCheckTransitionGate_ReviewingToNeedsRework_CapReached verifies that when
// a feature's review_cycle equals DefaultMaxReviewCycles the gate returns
// Satisfied=false and ReviewCapReached=true.
func TestCheckTransitionGate_ReviewingToNeedsRework_CapReached(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Feature at the iteration cap (review_cycle == DefaultMaxReviewCycles == 3).
	feature := &model.Feature{
		ID:          "FEAT-01DDDDDDDDDD01",
		ReviewCycle: DefaultMaxReviewCycles,
	}
	result := CheckTransitionGate("reviewing", "needs-rework", feature, docSvc, entitySvc)

	if result.Satisfied {
		t.Fatal("expected reviewing→needs-rework unsatisfied when cap is reached")
	}
	if !result.ReviewCapReached {
		t.Errorf("expected ReviewCapReached=true at cap, got false")
	}
	if result.Reason == "" {
		t.Error("expected non-empty Reason when cap is reached")
	}
}

// TestCheckTransitionGate_ReviewingToNeedsRework_BelowCap verifies that when
// a feature's review_cycle is one below DefaultMaxReviewCycles the gate is
// satisfied and ReviewCapReached remains false.
func TestCheckTransitionGate_ReviewingToNeedsRework_BelowCap(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	// Feature one below the cap (review_cycle == DefaultMaxReviewCycles-1 == 2).
	feature := &model.Feature{
		ID:          "FEAT-01DDDDDDDDDD02",
		ReviewCycle: DefaultMaxReviewCycles - 1,
	}
	result := CheckTransitionGate("reviewing", "needs-rework", feature, docSvc, entitySvc)

	if !result.Satisfied {
		t.Fatalf("expected reviewing→needs-rework satisfied below cap, reason: %s", result.Reason)
	}
	if result.ReviewCapReached {
		t.Error("expected ReviewCapReached=false below cap")
	}
}

// TestCheckTransitionGate_ReviewingToDone_AtCap_Allowed verifies that a
// reviewing→done transition (pass verdict) is unaffected by the review cap —
// the done gate only checks for a report document, not the cycle count.
func TestCheckTransitionGate_ReviewingToDone_AtCap_Allowed(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01DDDDDDDDDD03"
	// A review report is required for reviewing→done; register one (unapproved is fine).
	submitAndApproveDoc(t, docSvc, repoRoot, "work/reports/cap-done.md", "report", featureID, false)

	// Feature at the cap — the done transition must still be allowed.
	feature := &model.Feature{
		ID:          featureID,
		ReviewCycle: DefaultMaxReviewCycles,
	}
	result := CheckTransitionGate("reviewing", "done", feature, docSvc, entitySvc)

	if !result.Satisfied {
		t.Fatalf("expected reviewing→done satisfied at cap (pass verdict always allowed), reason: %s", result.Reason)
	}
	if result.ReviewCapReached {
		t.Errorf("expected ReviewCapReached=false for reviewing→done transition")
	}
}

// ─── checkAllTasksHaveVerification unit tests ─────────────────────────────────

// TestCheckAllTasksHaveVerification_ZeroTasks verifies that a feature with no
// child tasks returns nil (vacuously true) (AC-06).
func TestCheckAllTasksHaveVerification_ZeroTasks(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	feature := &model.Feature{ID: "FEAT-01FFFFFFFFFFF01"}
	err := checkAllTasksHaveVerification(feature, entitySvc)
	if err != nil {
		t.Errorf("expected nil for feature with no tasks, got: %v", err)
	}
}

// TestCheckAllTasksHaveVerification_AllVerified verifies that when all child
// tasks have a non-empty verification field the function returns nil (AC-04).
func TestCheckAllTasksHaveVerification_AllVerified(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01FFFFFFFFFFF02"

	taskA := makeTaskFields("T-01FFFFFFFFFFF01", "task-a", featureID, "done", nil)
	taskA["verification"] = "unit tests passed"
	writeTestEntity(t, stateRoot, "task", "T-01FFFFFFFFFFF01", "task-a", taskA)

	taskB := makeTaskFields("T-01FFFFFFFFFFF02", "task-b", featureID, "done", nil)
	taskB["verification"] = "integration tests passed"
	writeTestEntity(t, stateRoot, "task", "T-01FFFFFFFFFFF02", "task-b", taskB)

	feature := &model.Feature{ID: featureID}
	err := checkAllTasksHaveVerification(feature, entitySvc)
	if err != nil {
		t.Errorf("expected nil when all tasks have verification, got: %v", err)
	}
}

// TestCheckAllTasksHaveVerification_OneEmpty verifies that when one task has an
// empty verification field the function returns an error naming that task (AC-05).
func TestCheckAllTasksHaveVerification_OneEmpty(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01FFFFFFFFFFF03"

	// Task A has verification.
	taskA := makeTaskFields("T-01FFFFFFFFFFF03", "task-with-verif", featureID, "done", nil)
	taskA["verification"] = "all checks passed"
	writeTestEntity(t, stateRoot, "task", "T-01FFFFFFFFFFF03", "task-with-verif", taskA)

	// Task B has NO verification.
	writeTestEntity(t, stateRoot, "task", "T-01FFFFFFFFFFF04", "task-no-verif",
		makeTaskFields("T-01FFFFFFFFFFF04", "task-no-verif", featureID, "done", nil))

	feature := &model.Feature{ID: featureID}
	err := checkAllTasksHaveVerification(feature, entitySvc)
	if err == nil {
		t.Fatal("expected error when one task has empty verification, got nil")
	}
	if !strings.Contains(err.Error(), "T-01FFFFFFFFFFF04") {
		t.Errorf("error %q should identify unverified task T-01FFFFFFFFFFF04", err.Error())
	}
}

// TestCheckAllTasksHaveVerification_NeedsReview verifies that a task in
// needs-review status with an empty verification field causes the function to
// return an error — it does not auto-pass for needs-review tasks (AC-07).
func TestCheckAllTasksHaveVerification_NeedsReview(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	featureID := "FEAT-01FFFFFFFFFFF04"

	// Task is in needs-review state with NO verification recorded.
	writeTestEntity(t, stateRoot, "task", "T-01FFFFFFFFFFF05", "nr-task",
		makeTaskFields("T-01FFFFFFFFFFF05", "nr-task", featureID, "needs-review", nil))

	feature := &model.Feature{ID: featureID}
	err := checkAllTasksHaveVerification(feature, entitySvc)
	if err == nil {
		t.Fatal("expected error for needs-review task with empty verification, got nil")
	}
	if !strings.Contains(err.Error(), "T-01FFFFFFFFFFF05") {
		t.Errorf("error %q should identify needs-review task T-01FFFFFFFFFFF05", err.Error())
	}
}


