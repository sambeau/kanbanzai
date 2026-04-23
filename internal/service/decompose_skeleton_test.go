package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// ─── buildSkeletonDevPlan unit tests ─────────────────────────────────────────

func TestBuildSkeletonDevPlan_Title(t *testing.T) {
	t.Parallel()
	tasks := []SkeletonTask{{ID: "TASK-01", Summary: "do something"}}
	out := buildSkeletonDevPlan("My Feature", "FEAT-01", time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), tasks)
	wantTitle := "# Dev Plan: My Feature (Decomposed)"
	if !strings.Contains(out, wantTitle) {
		t.Errorf("expected title %q in output, got:\n%s", wantTitle, out)
	}
}

func TestBuildSkeletonDevPlan_HeaderTableFields(t *testing.T) {
	t.Parallel()
	tasks := []SkeletonTask{{ID: "TASK-01", Summary: "do something"}}
	out := buildSkeletonDevPlan("Alpha", "FEAT-ABC", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), tasks)

	checks := []string{
		"| Feature | FEAT-ABC |",
		"| Created | 2025-06-01 |",
		"| Method  | decompose apply (auto-generated) |",
		"| Status  | Draft |",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestBuildSkeletonDevPlan_OverviewSection(t *testing.T) {
	t.Parallel()
	tasks := []SkeletonTask{{ID: "TASK-01", Summary: "do something"}}
	out := buildSkeletonDevPlan("Beta", "FEAT-B", time.Now(), tasks)

	if !strings.Contains(out, "## Overview") {
		t.Errorf("expected '## Overview' section in output")
	}
	if !strings.Contains(out, "decompose(action: \"apply\")") {
		t.Errorf("expected decompose apply mention in overview")
	}
}

func TestBuildSkeletonDevPlan_TasksTable(t *testing.T) {
	t.Parallel()
	tasks := []SkeletonTask{
		{ID: "TASK-001", Summary: "first task"},
		{ID: "TASK-002", Summary: "second task"},
		{ID: "TASK-003", Summary: "third task with longer summary"},
	}
	out := buildSkeletonDevPlan("Gamma", "FEAT-G", time.Now(), tasks)

	if !strings.Contains(out, "## Tasks") {
		t.Errorf("expected '## Tasks' section in output")
	}
	if !strings.Contains(out, "| Task ID | Summary |") {
		t.Errorf("expected Tasks table header in output")
	}
	for _, task := range tasks {
		wantRow := "| " + task.ID + " | " + task.Summary + " |"
		if !strings.Contains(out, wantRow) {
			t.Errorf("expected task row %q in output, got:\n%s", wantRow, out)
		}
	}
}

func TestBuildSkeletonDevPlan_ValidUTF8(t *testing.T) {
	t.Parallel()
	tasks := []SkeletonTask{{ID: "TASK-01", Summary: "UTF-8: héllo wörld ✓"}}
	out := buildSkeletonDevPlan("Unicode Feature", "FEAT-U", time.Now(), tasks)
	if !utf8.ValidString(out) {
		t.Error("buildSkeletonDevPlan output is not valid UTF-8")
	}
}

// ─── WriteSkeletonDevPlan integration tests ──────────────────────────────────

// setupSkeletonTest creates the minimal service scaffolding needed for
// WriteSkeletonDevPlan tests: entity service, doc service, a plan, and a
// feature with an approved spec. Returns the decompose service, feature ID,
// and feature slug.
func setupSkeletonTest(t *testing.T) (*DecomposeService, string, string) {
	t.Helper()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	docSvc := NewDocumentService(stateRoot, repoRoot)

	planID := "P1-skeleton-plan"
	writeDecomposeTestPlan(t, entitySvc, planID)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "my-feature",
		Parent:    planID,
		Summary:   "Feature for skeleton test",
		CreatedBy: "tester",
		Name:      "My Feature",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}
	featureID := featResult.ID
	featureSlug := "my-feature"

	// Write and approve a spec so the feature is in a valid state.
	specPath := "work/spec/my-spec.md"
	specFull := filepath.Join(repoRoot, specPath)
	if err := os.MkdirAll(filepath.Dir(specFull), 0o755); err != nil {
		t.Fatalf("mkdir spec: %v", err)
	}
	if err := os.WriteFile(specFull, []byte(specWithACs), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	specDoc, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path: specPath, Type: "specification",
		Title: "My Spec", Owner: featureID, CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit spec: %v", err)
	}
	if _, err := docSvc.ApproveDocument(ApproveDocumentInput{ID: specDoc.ID, ApprovedBy: "tester"}); err != nil {
		t.Fatalf("approve spec: %v", err)
	}
	if _, err := entitySvc.UpdateEntity(UpdateEntityInput{
		Type: "feature", ID: featureID, Slug: featureSlug,
		Fields: map[string]string{"spec": specDoc.ID},
	}); err != nil {
		t.Fatalf("link spec: %v", err)
	}

	svc := NewDecomposeService(entitySvc, docSvc)
	return svc, featureID, featureSlug
}

func TestWriteSkeletonDevPlan_NoPlanExists_Created(t *testing.T) {
	t.Parallel()
	svc, featureID, featureSlug := setupSkeletonTest(t)

	tasks := []SkeletonTask{
		{ID: "TASK-AAA", Summary: "task A"},
		{ID: "TASK-BBB", Summary: "task B"},
	}
	result, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}
	if result.Action != "created" {
		t.Errorf("expected action=created, got %q", result.Action)
	}
	if result.DocID == "" {
		t.Error("expected non-empty DocID")
	}
	wantPath := "work/dev-plan/" + featureSlug + "-decomposed.md"
	if result.FilePath != wantPath {
		t.Errorf("expected FilePath=%q, got %q", wantPath, result.FilePath)
	}

	// Verify the document is approved.
	docs, err := svc.docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 dev-plan doc, got %d", len(docs))
	}
	if docs[0].Status != "approved" {
		t.Errorf("expected doc status=approved, got %q", docs[0].Status)
	}

	// Verify the file exists on disk.
	repoRoot := svc.docSvc.RepoRoot()
	fullPath := filepath.Join(repoRoot, wantPath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("expected file to exist at %s: %v", fullPath, err)
	}
}

func TestWriteSkeletonDevPlan_SkeletonExists_Updated_CountStaysOne(t *testing.T) {
	// Cannot use t.Parallel() — writes to shared tmp dirs per test are fine,
	// but no package-level var mutation here; keep parallel-safe.
	t.Parallel()
	svc, featureID, _ := setupSkeletonTest(t)

	tasksFirst := []SkeletonTask{{ID: "TASK-001", Summary: "first"}}
	first, err := svc.WriteSkeletonDevPlan(featureID, tasksFirst)
	if err != nil {
		t.Fatalf("first WriteSkeletonDevPlan() error = %v", err)
	}
	if first.Action != "created" {
		t.Errorf("expected first action=created, got %q", first.Action)
	}

	// Second call with different tasks.
	tasksSecond := []SkeletonTask{
		{ID: "TASK-001", Summary: "first"},
		{ID: "TASK-002", Summary: "second"},
	}
	second, err := svc.WriteSkeletonDevPlan(featureID, tasksSecond)
	if err != nil {
		t.Fatalf("second WriteSkeletonDevPlan() error = %v", err)
	}
	if second.Action != "updated" {
		t.Errorf("expected second action=updated, got %q", second.Action)
	}
	if second.DocID != first.DocID {
		t.Errorf("expected same DocID on update; first=%q second=%q", first.DocID, second.DocID)
	}

	// Count must still be 1.
	docs, err := svc.docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 dev-plan doc after update, got %d", len(docs))
	}
	if docs[0].Status != "approved" {
		t.Errorf("expected doc status=approved after update, got %q", docs[0].Status)
	}
}

func TestWriteSkeletonDevPlan_NonSkeletonExists_Skipped(t *testing.T) {
	t.Parallel()
	svc, featureID, _ := setupSkeletonTest(t)

	// Pre-register a non-skeleton dev-plan at a different path.
	repoRoot := svc.docSvc.RepoRoot()
	nonSkeletonPath := "work/dev-plan/manual-dev-plan.md"
	fullPath := filepath.Join(repoRoot, nonSkeletonPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Manual Dev Plan\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := svc.docSvc.SubmitDocument(SubmitDocumentInput{
		Path: nonSkeletonPath, Type: "dev-plan",
		Title: "Manual Dev Plan", Owner: featureID, CreatedBy: "human",
	})
	if err != nil {
		t.Fatalf("submit non-skeleton doc: %v", err)
	}

	tasks := []SkeletonTask{{ID: "TASK-001", Summary: "auto task"}}
	result, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}
	if result.Action != "skipped" {
		t.Errorf("expected action=skipped when non-skeleton exists, got %q", result.Action)
	}

	// Count must still be 1 (the manual one, unchanged).
	docs, err := svc.docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 dev-plan doc (the manual one), got %d", len(docs))
	}
	if docs[0].Path != nonSkeletonPath {
		t.Errorf("expected non-skeleton path to be unchanged, got %q", docs[0].Path)
	}
}

func TestWriteSkeletonDevPlan_ZeroTasks_NoFileNoRegistration(t *testing.T) {
	t.Parallel()
	svc, featureID, featureSlug := setupSkeletonTest(t)

	result, err := svc.WriteSkeletonDevPlan(featureID, nil)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan(nil tasks) error = %v", err)
	}
	if result.Action != "skipped" {
		t.Errorf("expected action=skipped for zero tasks, got %q", result.Action)
	}

	// No file should have been written.
	repoRoot := svc.docSvc.RepoRoot()
	skeletonPath := filepath.Join(repoRoot, "work/dev-plan/"+featureSlug+"-decomposed.md")
	if _, err := os.Stat(skeletonPath); !os.IsNotExist(err) {
		t.Errorf("expected no skeleton file for zero tasks, but file exists at %s", skeletonPath)
	}

	// No document record should have been created.
	docs, err := svc.docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan"})
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 dev-plan docs for zero tasks, got %d", len(docs))
	}
}

func TestWriteSkeletonDevPlan_ResponseHasDocIDAndPath(t *testing.T) {
	t.Parallel()
	svc, featureID, featureSlug := setupSkeletonTest(t)

	tasks := []SkeletonTask{{ID: "TASK-X", Summary: "something"}}
	result, err := svc.WriteSkeletonDevPlan(featureID, tasks)
	if err != nil {
		t.Fatalf("WriteSkeletonDevPlan() error = %v", err)
	}
	if result.DocID == "" {
		t.Error("expected non-empty DocID in result")
	}
	wantPath := "work/dev-plan/" + featureSlug + "-decomposed.md"
	if result.FilePath != wantPath {
		t.Errorf("expected FilePath=%q, got %q", wantPath, result.FilePath)
	}
	// DocID should contain the feature ID as prefix.
	if !strings.HasPrefix(result.DocID, featureID+"/") {
		t.Errorf("expected DocID to start with %q, got %q", featureID+"/", result.DocID)
	}
}
