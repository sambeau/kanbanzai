package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// TestDecomposeIntegration covers the full decompose apply → skeleton lifecycle:
// AC-001 (first apply creates + approves skeleton), AC-007 (second apply is
// idempotent), AC-003 (manual dev-plan is preserved), and the dev-planning →
// developing stage gate (passes after apply without a separate doc approve).
//
// The benchmark (AC-008) verifies that skeleton generation stays within 500ms
// of a bare file-write baseline.

// ─── helpers ─────────────────────────────────────────────────────────────────

// setupDevPlanningFeature builds a fully-wired test environment: entity + doc
// services, a plan, a feature with an approved spec, and the feature advanced
// to dev-planning status. Returns the DecomposeService, entity service, doc
// service, feature ID, and feature slug.
func setupDevPlanningFeature(t *testing.T) (svc *DecomposeService, entitySvc *EntityService, docSvc *DocumentService, featureID, featureSlug string) {
	t.Helper()

	// setupSkeletonTest creates services, plan, feature, and approved spec.
	svc, featureID, featureSlug = setupSkeletonTest(t)
	entitySvc = svc.entitySvc
	docSvc = svc.docSvc

	// Advance proposed → designing → specifying → dev-planning.
	// UpdateStatus only validates lifecycle graph transitions, not doc gates,
	// so this works in tests without needing design/spec docs linked.
	for _, status := range []string{"designing", "specifying", "dev-planning"} {
		if _, err := entitySvc.UpdateStatus(UpdateStatusInput{
			Type:   "feature",
			ID:     featureID,
			Slug:   featureSlug,
			Status: status,
		}); err != nil {
			t.Fatalf("advance feature to %s: %v", status, err)
		}
	}
	return
}

// ─── AC-001: first apply creates and approves the skeleton ──────────────────

func TestDecomposeIntegration_AC001_FirstApply_SkeletonCreated(t *testing.T) {
	t.Parallel()

	svc, _, docSvc, featureID, featureSlug := setupDevPlanningFeature(t)

	tasks := []SkeletonTask{
		{ID: "TASK-001", Summary: "implement the widget"},
		{ID: "TASK-002", Summary: "add widget tests"},
	}
	result, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}

	// Action must be "created" on first call.
	if result.Action != "created" {
		t.Errorf("Action = %q, want \"created\"", result.Action)
	}

	// Response must carry a non-empty DocID.
	if result.DocID == "" {
		t.Error("DocID is empty — response must include the registered document ID")
	}

	// Response path must match the convention.
	wantPath := "work/dev-plan/" + featureSlug + "-decomposed.md"
	if result.FilePath != wantPath {
		t.Errorf("FilePath = %q, want %q", result.FilePath, wantPath)
	}

	// Exactly one dev-plan document record must be registered with status=approved.
	docs, err := docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("doc count = %d, want 1", len(docs))
	}
	if docs[0].Status != "approved" {
		t.Errorf("doc status = %q, want \"approved\"", docs[0].Status)
	}
	if docs[0].ID != result.DocID {
		t.Errorf("listed doc ID %q != response DocID %q", docs[0].ID, result.DocID)
	}

	// The file must exist on disk.
	fullPath := filepath.Join(docSvc.RepoRoot(), wantPath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("skeleton file not found at %s: %v", fullPath, err)
	}

	// The file must contain task IDs.
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read skeleton file: %v", err)
	}
	for _, task := range tasks {
		if !strings.Contains(string(raw), task.ID) {
			t.Errorf("skeleton file does not contain task ID %q", task.ID)
		}
	}
}

// ─── AC-007: second apply is idempotent ──────────────────────────────────────

func TestDecomposeIntegration_AC007_SecondApply_Idempotent(t *testing.T) {
	t.Parallel()

	svc, _, docSvc, featureID, _ := setupDevPlanningFeature(t)

	firstTasks := []SkeletonTask{{ID: "TASK-001", Summary: "first task"}}
	first, err := svc.WriteSkeletonDevPlan(featureID, firstTasks)
	if err != nil {
		t.Fatalf("first WriteSkeletonDevPlan() error = %v", err)
	}
	if first.Action != "created" {
		t.Fatalf("first call: Action = %q, want \"created\"", first.Action)
	}

	// Second call with an updated (larger) task list.
	secondTasks := []SkeletonTask{
		{ID: "TASK-001", Summary: "first task"},
		{ID: "TASK-002", Summary: "second task"},
	}
	second, err := svc.WriteSkeletonDevPlan(featureID, secondTasks)
	if err != nil {
		t.Fatalf("second WriteSkeletonDevPlan() error = %v", err)
	}
	if second.Action != "updated" {
		t.Errorf("second call: Action = %q, want \"updated\"", second.Action)
	}

	// Same DocID must be returned on update (no new record created).
	if second.DocID != first.DocID {
		t.Errorf("DocID changed: first=%q second=%q — idempotency violation", first.DocID, second.DocID)
	}

	// Exactly one document record must exist after two applies.
	docs, err := docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("doc count = %d after second apply, want 1", len(docs))
	}
	if docs[0].Status != "approved" {
		t.Errorf("doc status = %q after second apply, want \"approved\"", docs[0].Status)
	}
}

// ─── AC-003: manually-authored dev-plan is preserved ─────────────────────────

func TestDecomposeIntegration_AC003_ManualDevPlan_Preserved(t *testing.T) {
	t.Parallel()

	svc, _, docSvc, featureID, _ := setupDevPlanningFeature(t)

	// Register a manual dev-plan at a non-convention path (as if hand-authored).
	manualPath := "work/dev-plan/my-feature-handcrafted.md"
	fullManualPath := filepath.Join(docSvc.RepoRoot(), manualPath)
	if err := os.MkdirAll(filepath.Dir(fullManualPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullManualPath, []byte("# Hand-crafted Dev Plan\n"), 0o644); err != nil {
		t.Fatalf("write manual plan: %v", err)
	}
	manualDoc, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      manualPath,
		Type:      "dev-plan",
		Title:     "Hand-crafted Dev Plan",
		Owner:     featureID,
		CreatedBy: "human-author",
	})
	if err != nil {
		t.Fatalf("register manual dev-plan: %v", err)
	}
	originalDocID := manualDoc.ID

	// Approve the manual doc so it is a realistic authored plan.
	if _, err := docSvc.ApproveDocument(ApproveDocumentInput{ID: originalDocID, ApprovedBy: "human-author"}); err != nil {
		t.Fatalf("approve manual dev-plan: %v", err)
	}

	// WriteSkeletonDevPlan must detect the non-skeleton doc and skip.
	tasks := []SkeletonTask{{ID: "TASK-001", Summary: "auto task"}}
	result, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}
	if result.Action != "skipped" {
		t.Errorf("Action = %q, want \"skipped\" when a manual dev-plan exists", result.Action)
	}

	// Exactly one document record must exist: the original manual plan, unchanged.
	docs, err := docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("doc count = %d, want 1 (the original manual plan)", len(docs))
	}
	if docs[0].ID != originalDocID {
		t.Errorf("doc ID changed: got %q, want original %q", docs[0].ID, originalDocID)
	}
	if docs[0].Path != manualPath {
		t.Errorf("doc path = %q, want %q — original record was modified", docs[0].Path, manualPath)
	}
}

// ─── Gate test: dev-planning → developing passes after apply ─────────────────

func TestDecomposeIntegration_GatePassesAfterApply(t *testing.T) {
	t.Parallel()

	svc, entitySvc, docSvc, featureID, featureSlug := setupDevPlanningFeature(t)

	// Before apply: gate must be unsatisfied (no dev-plan, no tasks yet).
	featureBefore := &model.Feature{
		ID:     featureID,
		Slug:   featureSlug,
		Parent: "P1-skeleton-plan",
		Status: model.FeatureStatus("dev-planning"),
	}
	gateBefore := CheckTransitionGate("dev-planning", "developing", featureBefore, docSvc, entitySvc)
	if gateBefore.Satisfied {
		t.Error("gate Satisfied = true before apply — expected unsatisfied (no dev-plan, no tasks)")
	}

	// Create a child task (simulates what decompose apply does before calling WriteSkeletonDevPlan).
	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featureID,
		Slug:          "integration-task",
		Name:          "Integration task",
		Summary:       "A task created by decompose apply",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Call WriteSkeletonDevPlan — no manual doc approve is invoked.
	tasks := []SkeletonTask{{ID: taskResult.ID, Summary: "A task created by decompose apply"}}
	skResult, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}
	if skResult.Action == "skipped" {
		t.Fatalf("unexpected Action=skipped — skeleton should have been created (action: %s)", skResult.Action)
	}

	// After apply: gate must be satisfied without any separate doc approve call.
	featureAfter := &model.Feature{
		ID:     featureID,
		Slug:   featureSlug,
		Parent: "P1-skeleton-plan",
		Status: model.FeatureStatus("dev-planning"),
	}
	gateAfter := CheckTransitionGate("dev-planning", "developing", featureAfter, docSvc, entitySvc)
	if !gateAfter.Satisfied {
		t.Errorf("gate Satisfied = false after apply — reason: %s", gateAfter.Reason)
	}
}

// ─── AC-008: skeleton latency ≤ 500ms over baseline ─────────────────────────

// BenchmarkDecomposeIntegration_SkeletonLatency measures WriteSkeletonDevPlan
// against a raw file-write baseline. Per AC-008 the latency delta must be ≤ 500ms.
//
// Run with:
//
//	go test ./internal/service/... -run ^$ -bench BenchmarkDecomposeIntegration -benchtime 5s
func BenchmarkDecomposeIntegration_SkeletonLatency(b *testing.B) {
	benchTasks := []SkeletonTask{
		{ID: "TASK-001", Summary: "task one"},
		{ID: "TASK-002", Summary: "task two"},
		{ID: "TASK-003", Summary: "task three"},
	}

	// Baseline: build Markdown + write file only (no registration, no doc service).
	b.Run("baseline_build_and_write", func(b *testing.B) {
		dir := b.TempDir()
		path := filepath.Join(dir, "baseline.md")
		fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		b.ResetTimer()
		for range b.N {
			content := buildSkeletonDevPlan("Bench Feature", "FEAT-BENCH", fixedTime, benchTasks)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				b.Fatalf("baseline write: %v", err)
			}
		}
	})

	// Full WriteSkeletonDevPlan: build, write, register + approve in doc service.
	b.Run("WriteSkeletonDevPlan", func(b *testing.B) {
		stateRoot := b.TempDir()
		repoRoot := b.TempDir()
		entitySvc := NewEntityService(stateRoot)
		docSvc := NewDocumentService(stateRoot, repoRoot)

		planID := "P1-bench-plan"
		benchWritePlanB(b, entitySvc, planID)

		decomposeSvc := NewDecomposeService(entitySvc, docSvc)

		b.ResetTimer()
		for i := range b.N {
			// Each iteration uses a fresh feature so idempotency logic takes the
			// "created" (not "updated") path — that's the hot case we want to measure.
			b.StopTimer()
			slug := "bench-feature-" + strconv.Itoa(i)
			feat, err := entitySvc.CreateFeature(CreateFeatureInput{
				Slug:      slug,
				Parent:    planID,
				Summary:   "Benchmark feature",
				CreatedBy: "bench",
				Name:      "Benchmark Feature",
			})
			if err != nil {
				b.Fatalf("CreateFeature iter %d: %v", i, err)
			}
			b.StartTimer()

			if _, err := decomposeSvc.WriteSkeletonDevPlan(feat.ID, benchTasks); err != nil {
				b.Fatalf("WriteSkeletonDevPlan iter %d: %v", i, err)
			}
		}
	})
}

// benchWritePlanB writes a minimal plan record for benchmark tests.
func benchWritePlanB(b *testing.B, svc *EntityService, id string) {
	b.Helper()
	_, _, slug := model.ParsePlanID(id)
	_, err := svc.store.Write(storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":         id,
			"slug":       slug,
			"name":       "Bench Plan",
			"status":     "active",
			"summary":    "Benchmark test plan",
			"created":    "2026-01-01T00:00:00Z",
			"created_by": "bench",
			"updated":    "2026-01-01T00:00:00Z",
		},
	})
	if err != nil {
		b.Fatalf("benchWritePlanB(%s): %v", id, err)
	}
}
