package service

import (
	"os"
	"path/filepath"
	"testing"

	"kanbanzai/internal/model"
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
