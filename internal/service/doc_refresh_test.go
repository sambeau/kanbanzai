package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kanbanzai/internal/model"
)

func TestRefreshContentHash_HashChanged_DraftDoc(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:        "FEAT-R1/design-v1",
		Path:      "work/design/r1.md",
		Type:      model.DocumentTypeDesign,
		Title:     "Design R1",
		Status:    model.DocumentStatusDraft,
		Created:   now,
		CreatedBy: "tester",
		Updated:   now,
	})

	// Modify the file so the hash differs from the recorded value.
	fullPath := filepath.Join(repoRoot, "work/design/r1.md")
	if err := os.WriteFile(fullPath, []byte("# Design R1\n\nUpdated content.\n"), 0o644); err != nil {
		t.Fatalf("modify doc file: %v", err)
	}

	result, err := svc.RefreshContentHash(RefreshInput{ID: "FEAT-R1/design-v1"})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if !result.Changed {
		t.Errorf("expected Changed=true")
	}
	if result.OldHash == result.NewHash {
		t.Errorf("expected OldHash != NewHash after content change")
	}
	if result.Status != string(model.DocumentStatusDraft) {
		t.Errorf("status = %q, want draft", result.Status)
	}
	if result.StatusTransition != "" {
		t.Errorf("expected no StatusTransition for draft doc, got %q", result.StatusTransition)
	}
	if result.ID != "FEAT-R1/design-v1" {
		t.Errorf("ID = %q, want FEAT-R1/design-v1", result.ID)
	}
	if result.Path != "work/design/r1.md" {
		t.Errorf("Path = %q, want work/design/r1.md", result.Path)
	}
}

func TestRefreshContentHash_HashChanged_ApprovedDoc(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:         "FEAT-R2/design-v1",
		Path:       "work/design/r2.md",
		Type:       model.DocumentTypeDesign,
		Title:      "Design R2",
		Status:     model.DocumentStatusApproved,
		Created:    now,
		CreatedBy:  "tester",
		Updated:    now,
		ApprovedBy: "reviewer",
	})

	// Modify the file so the hash differs.
	fullPath := filepath.Join(repoRoot, "work/design/r2.md")
	if err := os.WriteFile(fullPath, []byte("# Design R2\n\nRevised content.\n"), 0o644); err != nil {
		t.Fatalf("modify doc file: %v", err)
	}

	result, err := svc.RefreshContentHash(RefreshInput{ID: "FEAT-R2/design-v1"})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if !result.Changed {
		t.Errorf("expected Changed=true")
	}
	if result.Status != string(model.DocumentStatusDraft) {
		t.Errorf("status = %q, want draft (approved doc demoted on hash change)", result.Status)
	}
	if result.StatusTransition != "approved → draft" {
		t.Errorf("StatusTransition = %q, want %q", result.StatusTransition, "approved → draft")
	}
}

func TestRefreshContentHash_HashUnchanged(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:        "FEAT-R3/design-v1",
		Path:      "work/design/r3.md",
		Type:      model.DocumentTypeDesign,
		Title:     "Design R3",
		Status:    model.DocumentStatusDraft,
		Created:   now,
		CreatedBy: "tester",
		Updated:   now,
	})

	// Do NOT modify the file — hash should still match the recorded value.
	result, err := svc.RefreshContentHash(RefreshInput{ID: "FEAT-R3/design-v1"})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if result.Changed {
		t.Errorf("expected Changed=false when content is identical")
	}
	if result.OldHash != result.NewHash {
		t.Errorf("expected OldHash == NewHash when unchanged")
	}
	if result.Status != string(model.DocumentStatusDraft) {
		t.Errorf("status = %q, want draft", result.Status)
	}
	if result.StatusTransition != "" {
		t.Errorf("expected empty StatusTransition, got %q", result.StatusTransition)
	}
}

func TestRefreshContentHash_FileNotFound(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:        "FEAT-R4/design-v1",
		Path:      "work/design/r4.md",
		Type:      model.DocumentTypeDesign,
		Title:     "Design R4",
		Status:    model.DocumentStatusDraft,
		Created:   now,
		CreatedBy: "tester",
		Updated:   now,
	})

	// Delete the underlying file so the hash cannot be computed.
	fullPath := filepath.Join(repoRoot, "work/design/r4.md")
	if err := os.Remove(fullPath); err != nil {
		t.Fatalf("remove doc file: %v", err)
	}

	_, err := svc.RefreshContentHash(RefreshInput{ID: "FEAT-R4/design-v1"})
	if err == nil {
		t.Fatal("expected error when document file is missing")
	}
	if !strings.Contains(err.Error(), "work/design/r4.md") {
		t.Errorf("error %q does not mention file path", err.Error())
	}
}

func TestRefreshContentHash_RecordNotFoundByID(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	_, err := svc.RefreshContentHash(RefreshInput{ID: "FEAT-MISSING/design-v1"})
	if err == nil {
		t.Fatal("expected error for non-existent document record")
	}
}

func TestRefreshContentHash_RecordNotFoundByPath(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	_, err := svc.RefreshContentHash(RefreshInput{Path: "work/design/nonexistent.md"})
	if err == nil {
		t.Fatal("expected error for non-existent document path")
	}
}

func TestRefreshContentHash_EmptyInput(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	_, err := svc.RefreshContentHash(RefreshInput{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "id or path is required") {
		t.Errorf("error %q does not contain expected message", err.Error())
	}
}

func TestRefreshContentHash_PathLookup(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := newTestDocService(stateRoot, repoRoot)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	writeTestDocument(t, stateRoot, repoRoot, model.DocumentRecord{
		ID:        "FEAT-R8/design-v1",
		Path:      "work/design/r8.md",
		Type:      model.DocumentTypeDesign,
		Title:     "Design R8",
		Status:    model.DocumentStatusDraft,
		Created:   now,
		CreatedBy: "tester",
		Updated:   now,
	})

	// Modify the file so the hash changes.
	fullPath := filepath.Join(repoRoot, "work/design/r8.md")
	if err := os.WriteFile(fullPath, []byte("# Design R8\n\nNew content via path lookup.\n"), 0o644); err != nil {
		t.Fatalf("modify doc file: %v", err)
	}

	// Call with Path instead of ID — should resolve to the correct record.
	result, err := svc.RefreshContentHash(RefreshInput{Path: "work/design/r8.md"})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if !result.Changed {
		t.Errorf("expected Changed=true")
	}
	if result.ID != "FEAT-R8/design-v1" {
		t.Errorf("ID = %q, want FEAT-R8/design-v1", result.ID)
	}
	if result.Path != "work/design/r8.md" {
		t.Errorf("Path = %q, want work/design/r8.md", result.Path)
	}
}
