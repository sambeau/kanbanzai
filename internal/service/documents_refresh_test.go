package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// TestRefreshDocument_WhitespaceOnlyChangePreservesApproval covers AC-005 (REQ-004).
// An approved document with a whitespace-only change should remain approved after refresh.
func TestRefreshDocument_WhitespaceOnlyChangePreservesApproval(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and register an approved document.
	docPath := "work/test/whitespace-doc.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test Document\n\nThis is the content.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "Whitespace Test Doc",
		Owner:     "FEAT-001",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve it.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Make a whitespace-only change: add trailing spaces and extra blank lines.
	if err := os.WriteFile(fullPath, []byte("# Test Document  \n\n\nThis is the content.  \n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	// Refresh — should preserve approval.
	refreshResult, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() error = %v", err)
	}

	if !refreshResult.Changed {
		t.Error("expected Changed=true for whitespace-modified file")
	}
	if refreshResult.Status != string(model.DocumentStatusApproved) {
		t.Errorf("Status = %q, want %q (approval should be preserved for whitespace-only changes)", refreshResult.Status, model.DocumentStatusApproved)
	}
	if refreshResult.StatusTransition != "" {
		t.Errorf("StatusTransition = %q, want empty (no transition for formatting-only changes)", refreshResult.StatusTransition)
	}
	if refreshResult.Message == "" {
		t.Error("expected a Message explaining that approval was preserved")
	}
}

// TestRefreshDocument_ContentChangeResetsToDraft covers AC-006 (REQ-005).
// An approved document with a substantive content change should reset to draft with a warning.
func TestRefreshDocument_ContentChangeResetsToDraft(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and register an approved document.
	docPath := "work/test/content-change-doc.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test Document\n\nOriginal content.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "Content Change Test Doc",
		Owner:     "FEAT-002",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve it.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Make a substantive change: modify actual content.
	if err := os.WriteFile(fullPath, []byte("# Test Document\n\nCompletely different content here.\n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	// Refresh — should reset to draft with warning.
	refreshResult, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() error = %v", err)
	}

	if !refreshResult.Changed {
		t.Error("expected Changed=true for content-modified file")
	}
	if refreshResult.Status != string(model.DocumentStatusDraft) {
		t.Errorf("Status = %q, want %q (approval should be reset for substantive changes)", refreshResult.Status, model.DocumentStatusDraft)
	}
	if refreshResult.StatusTransition != "approved → draft" {
		t.Errorf("StatusTransition = %q, want %q", refreshResult.StatusTransition, "approved → draft")
	}
	if refreshResult.Message == "" {
		t.Error("expected a warning Message about approval reset")
	}
}

// TestRefreshContentHash_WhitespaceOnlyPreservesApproval covers AC-005 for
// the RefreshContentHash path (used by the MCP doc refresh action).
func TestRefreshContentHash_WhitespaceOnlyPreservesApproval(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/test/refresh-content-hash-ws.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test\n\nContent here.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "RefreshContentHash WS Test",
		Owner:     "FEAT-003",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve it.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Whitespace-only change.
	if err := os.WriteFile(fullPath, []byte("# Test  \n\n\nContent here.  \n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	refreshResult, err := svc.RefreshContentHash(RefreshInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if !refreshResult.Changed {
		t.Error("expected Changed=true")
	}
	if refreshResult.Status != string(model.DocumentStatusApproved) {
		t.Errorf("Status = %q, want %q", refreshResult.Status, model.DocumentStatusApproved)
	}
	if refreshResult.StatusTransition != "" {
		t.Errorf("StatusTransition = %q, want empty", refreshResult.StatusTransition)
	}
}

// TestRefreshContentHash_ContentChangeResetsToDraft covers AC-006 for
// the RefreshContentHash path.
func TestRefreshContentHash_ContentChangeResetsToDraft(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/test/refresh-content-hash-sub.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test\n\nOriginal.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "RefreshContentHash Content Test",
		Owner:     "FEAT-004",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve it.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Substantive content change.
	if err := os.WriteFile(fullPath, []byte("# Test\n\nTotally different.\n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	refreshResult, err := svc.RefreshContentHash(RefreshInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshContentHash() error = %v", err)
	}

	if !refreshResult.Changed {
		t.Error("expected Changed=true")
	}
	if refreshResult.Status != string(model.DocumentStatusDraft) {
		t.Errorf("Status = %q, want %q", refreshResult.Status, model.DocumentStatusDraft)
	}
	if refreshResult.StatusTransition != "approved → draft" {
		t.Errorf("StatusTransition = %q, want %q", refreshResult.StatusTransition, "approved → draft")
	}
	if refreshResult.Message == "" {
		t.Error("expected a warning Message")
	}
}

// TestRefreshDocument_NoCanonicalHashFallback covers the edge case where a
// document was registered before canonical hashes existed. In that case,
// a content change on an approved document should still reset to draft
// (the safe default).
func TestRefreshDocument_NoCanonicalHashFallback(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/test/legacy-doc.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Legacy\n\nOld content.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "Legacy Doc Without Canonical Hash",
		Owner:     "FEAT-005",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve it.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Simulate a legacy record by clearing the canonical hash directly.
	record, loadErr := svc.store.Load(submitResult.ID)
	if loadErr != nil {
		t.Fatalf("Load() error = %v", loadErr)
	}
	doc := storage.RecordToDocument(record)
	doc.CanonicalContentHash = ""
	updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
	if _, err := svc.store.Write(updatedRecord); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Make any change (even whitespace) — should reset to draft since no canonical hash exists.
	if err := os.WriteFile(fullPath, []byte("# Legacy  \n\nOld content.  \n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	refreshResult, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() error = %v", err)
	}

	// Without a canonical hash, even whitespace changes should reset to draft (safe default).
	if refreshResult.Status != string(model.DocumentStatusDraft) {
		t.Errorf("Status = %q, want %q (no canonical hash should trigger safe reset)", refreshResult.Status, model.DocumentStatusDraft)
	}
	if refreshResult.StatusTransition != "approved → draft" {
		t.Errorf("StatusTransition = %q, want %q", refreshResult.StatusTransition, "approved → draft")
	}
}

// TestRefreshDocument_ChangeTwice_FormattingThenContent tracks a document
// through two refresh cycles: first a formatting change (preserves approval),
// then a content change (resets to draft).
func TestRefreshDocument_ChangeTwice_FormattingThenContent(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/test/two-changes.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Doc\n\nVersion 1.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "Two Changes Test",
		Owner:     "FEAT-006",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "tester",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// First change: whitespace only — should preserve approval.
	if err := os.WriteFile(fullPath, []byte("# Doc  \n\nVersion 1.  \n"), 0o644); err != nil {
		t.Fatalf("failed to update document (ws): %v", err)
	}

	r1, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() #1 error = %v", err)
	}
	if r1.Status != string(model.DocumentStatusApproved) {
		t.Errorf("after formatting change: Status = %q, want approved", r1.Status)
	}

	// Second change: content change — should reset to draft.
	if err := os.WriteFile(fullPath, []byte("# Doc\n\nVersion 2 - new content.\n"), 0o644); err != nil {
		t.Fatalf("failed to update document (content): %v", err)
	}

	r2, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() #2 error = %v", err)
	}
	if r2.Status != string(model.DocumentStatusDraft) {
		t.Errorf("after content change: Status = %q, want draft", r2.Status)
	}
	if r2.StatusTransition != "approved → draft" {
		t.Errorf("after content change: StatusTransition = %q, want %q", r2.StatusTransition, "approved → draft")
	}
}

// TestRefreshDocument_NotApproved_NoSideEffects ensures that documents in
// draft status don't have spurious messages or transitions on refresh.
func TestRefreshDocument_NotApproved_NoSideEffects(t *testing.T) {
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/test/draft-doc.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Draft\n\nContent.\n"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "spec",
		Title:     "Draft Doc",
		Owner:     "FEAT-007",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Modify content.
	if err := os.WriteFile(fullPath, []byte("# Draft\n\nNew content.\n"), 0o644); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	refreshResult, err := svc.RefreshDocument(RefreshDocumentInput{ID: submitResult.ID})
	if err != nil {
		t.Fatalf("RefreshDocument() error = %v", err)
	}

	if refreshResult.Status != string(model.DocumentStatusDraft) {
		t.Errorf("Status = %q, want draft", refreshResult.Status)
	}
	if refreshResult.StatusTransition != "" {
		t.Errorf("StatusTransition = %q, want empty for non-approved doc", refreshResult.StatusTransition)
	}
}
