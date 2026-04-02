package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

func TestSubmitDocument_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create a test document file
	docPath := "work/design/test-doc.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test Document\n\nThis is a test."), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	result, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test Design Document",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Verify result
	if result.Status != string(model.DocumentStatusDraft) {
		t.Errorf("Status = %q, want %q", result.Status, model.DocumentStatusDraft)
	}
	if result.Type != "design" {
		t.Errorf("Type = %q, want %q", result.Type, "design")
	}
	if result.Title != "Test Design Document" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Design Document")
	}
	if result.Owner != "FEAT-123" {
		t.Errorf("Owner = %q, want %q", result.Owner, "FEAT-123")
	}
	if result.ContentHash == "" {
		t.Error("ContentHash should not be empty")
	}
	if result.Path != docPath {
		t.Errorf("Path = %q, want %q", result.Path, docPath)
	}
}

func TestSubmitDocument_MissingFile(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "nonexistent/file.md",
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestSubmitDocument_InvalidType(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create a test document file
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "invalid-type",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err == nil {
		t.Fatal("expected error for invalid document type")
	}
	if !strings.Contains(err.Error(), "invalid document type") {
		t.Errorf("error = %q, want to contain 'invalid document type'", err.Error())
	}
}

func TestSubmitDocument_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	testCases := []struct {
		name  string
		input SubmitDocumentInput
		want  string
	}{
		{
			name: "missing path",
			input: SubmitDocumentInput{
				Type:      "design",
				Title:     "Test",
				CreatedBy: "tester",
			},
			want: "path is required",
		},
		{
			name: "missing type",
			input: SubmitDocumentInput{
				Path:      "test.md",
				Title:     "Test",
				CreatedBy: "tester",
			},
			want: "type is required",
		},
		{
			name: "missing title",
			input: SubmitDocumentInput{
				Path:      "test.md",
				Type:      "design",
				CreatedBy: "tester",
			},
			want: "title is required",
		},
		{
			name: "missing created_by",
			input: SubmitDocumentInput{
				Path:  "test.md",
				Type:  "design",
				Title: "Test",
			},
			want: "created_by is required",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.SubmitDocument(tc.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestApproveDocument_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve the document
	approveResult, err := svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	// Verify result
	if approveResult.Status != string(model.DocumentStatusApproved) {
		t.Errorf("Status = %q, want %q", approveResult.Status, model.DocumentStatusApproved)
	}
	if approveResult.ApprovedBy != "reviewer" {
		t.Errorf("ApprovedBy = %q, want %q", approveResult.ApprovedBy, "reviewer")
	}
	if approveResult.ApprovedAt == nil {
		t.Error("ApprovedAt should not be nil")
	}
}

func TestApproveDocument_NotDraft(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Approve once
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("first ApproveDocument() error = %v", err)
	}

	// Try to approve again
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer2",
	})
	if err == nil {
		t.Fatal("expected error for approving non-draft document")
	}
	if !strings.Contains(err.Error(), "cannot approve") {
		t.Errorf("error = %q, want to contain 'cannot approve'", err.Error())
	}
}

// TestApproveDocument_AutoRefreshHashOnApproval verifies that approving a
// document whose file has changed since registration succeeds and updates the
// stored content hash (FR-B06, FR-B07, AC-B08, AC-B09).
func TestApproveDocument_AutoRefreshHashOnApproval(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document.
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("original content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	originalHash := submitResult.ContentHash

	// Modify the file after submission (simulates editing before approval).
	if err := os.WriteFile(fullPath, []byte("modified content after review"), 0o644); err != nil {
		t.Fatalf("failed to modify document: %v", err)
	}

	// Approve — should succeed and auto-refresh the hash (FR-B06, AC-B08).
	approveResult, err := svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v; want success (hash should be auto-refreshed)", err)
	}

	// Status must be approved.
	if approveResult.Status != "approved" {
		t.Errorf("status = %q, want %q", approveResult.Status, "approved")
	}

	// Stored hash must now match the modified file content (AC-B09).
	if approveResult.ContentHash == originalHash {
		t.Error("content hash was not updated; want hash to reflect modified file")
	}
	// Verify against the actual file hash.
	currentHash, hashErr := storage.ComputeContentHash(fullPath)
	if hashErr != nil {
		t.Fatalf("compute current hash: %v", hashErr)
	}
	if approveResult.ContentHash != currentHash {
		t.Errorf("approved content hash = %q, want %q (current file hash)", approveResult.ContentHash, currentHash)
	}
}

// TestApproveDocument_FileMissing verifies that approving a document whose
// file is missing on disk returns an error (FR-B07, AC-B10).
func TestApproveDocument_FileMissing(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document.
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}
	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Remove the file before approval.
	if err := os.Remove(fullPath); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	// Approve — should fail because the file is missing.
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err == nil {
		t.Fatal("expected error when file is missing, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain 'not found'", err.Error())
	}
}

func TestGetDocument_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test Document",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Get the document
	result, err := svc.GetDocument(submitResult.ID, false)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}

	if result.ID != submitResult.ID {
		t.Errorf("ID = %q, want %q", result.ID, submitResult.ID)
	}
	if result.Title != "Test Document" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Document")
	}
}

func TestGetDocument_DriftDetection(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("original content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Wait a moment to ensure mtime will be different
	time.Sleep(10 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(fullPath, []byte("modified content"), 0o644); err != nil {
		t.Fatalf("failed to modify document: %v", err)
	}

	// Get with drift check
	result, err := svc.GetDocument(submitResult.ID, true)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}

	if !result.Drift {
		t.Error("Drift = false, want true (content was modified)")
	}
	if result.CurrentHash == result.ContentHash {
		t.Error("CurrentHash should differ from ContentHash when drifted")
	}
}

func TestGetDocumentContent_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	expectedContent := "# Test\n\nThis is the content."
	if err := os.WriteFile(fullPath, []byte(expectedContent), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Get content
	content, result, err := svc.GetDocumentContent(submitResult.ID)
	if err != nil {
		t.Fatalf("GetDocumentContent() error = %v", err)
	}

	if content != expectedContent {
		t.Errorf("content = %q, want %q", content, expectedContent)
	}
	if result.ID != submitResult.ID {
		t.Errorf("result.ID = %q, want %q", result.ID, submitResult.ID)
	}
}

func TestListDocuments_Empty(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	results, err := svc.ListDocuments(DocumentFilters{})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len = %d, want 0", len(results))
	}
}

func TestListDocuments_WithFilters(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create some documents
	docs := []struct {
		path  string
		dtype string
		owner string
	}{
		{"design1.md", "design", "P1-test"},
		{"spec1.md", "specification", "FEAT-123"},
		{"design2.md", "design", "FEAT-456"},
	}

	for _, d := range docs {
		fullPath := filepath.Join(repoRoot, d.path)
		if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", d.path, err)
		}
		_, err := svc.SubmitDocument(SubmitDocumentInput{
			Path:      d.path,
			Type:      d.dtype,
			Title:     d.path,
			Owner:     d.owner,
			CreatedBy: "tester",
		})
		if err != nil {
			t.Fatalf("SubmitDocument(%s) error = %v", d.path, err)
		}
	}

	// Filter by type
	results, err := svc.ListDocuments(DocumentFilters{Type: "design"})
	if err != nil {
		t.Fatalf("ListDocuments(type=design) error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("type filter: len = %d, want 2", len(results))
	}

	// Filter by owner
	results, err = svc.ListDocuments(DocumentFilters{Owner: "FEAT-123"})
	if err != nil {
		t.Fatalf("ListDocuments(owner=FEAT-123) error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("owner filter: len = %d, want 1", len(results))
	}

	// Filter by status
	results, err = svc.ListDocuments(DocumentFilters{Status: "draft"})
	if err != nil {
		t.Fatalf("ListDocuments(status=draft) error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("status filter: len = %d, want 3", len(results))
	}
}

func TestSupersedeDocument_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create two documents
	doc1Path := "doc1.md"
	doc2Path := "doc2.md"
	for _, p := range []string{doc1Path, doc2Path} {
		fullPath := filepath.Join(repoRoot, p)
		if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", p, err)
		}
	}

	// Submit and approve first document
	submit1, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      doc1Path,
		Type:      "design",
		Title:     "Original Design",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(doc1) error = %v", err)
	}

	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submit1.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument(doc1) error = %v", err)
	}

	// Submit second document (the superseding one)
	submit2, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      doc2Path,
		Type:      "design",
		Title:     "New Design",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(doc2) error = %v", err)
	}

	// Supersede the first document
	result, err := svc.SupersedeDocument(SupersedeDocumentInput{
		ID:           submit1.ID,
		SupersededBy: submit2.ID,
	})
	if err != nil {
		t.Fatalf("SupersedeDocument() error = %v", err)
	}

	if result.Status != string(model.DocumentStatusSuperseded) {
		t.Errorf("Status = %q, want %q", result.Status, model.DocumentStatusSuperseded)
	}
	if result.SupersededBy != submit2.ID {
		t.Errorf("SupersededBy = %q, want %q", result.SupersededBy, submit2.ID)
	}

	// Check that the superseding document has the supersedes reference
	doc2, err := svc.GetDocument(submit2.ID, false)
	if err != nil {
		t.Fatalf("GetDocument(doc2) error = %v", err)
	}
	if doc2.Supersedes != submit1.ID {
		t.Errorf("doc2.Supersedes = %q, want %q", doc2.Supersedes, submit1.ID)
	}
}

func TestValidateDocument_Valid(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	issues, err := svc.ValidateDocument(submitResult.ID)
	if err != nil {
		t.Fatalf("ValidateDocument() error = %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("issues = %v, want empty", issues)
	}
}

func TestValidateDocument_MissingFile(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Delete the file
	if err := os.Remove(fullPath); err != nil {
		t.Fatalf("failed to remove document: %v", err)
	}

	issues, err := svc.ValidateDocument(submitResult.ID)
	if err != nil {
		t.Fatalf("ValidateDocument() error = %v", err)
	}
	if len(issues) == 0 {
		t.Error("expected validation issues for missing file")
	}

	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "not found") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'not found' issue, got %v", issues)
	}
}

func TestDocumentExists(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Check non-existent
	if svc.DocumentExists("nonexistent") {
		t.Error("DocumentExists() = true for non-existent document")
	}

	// Create and submit a document
	docPath := "test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create document: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Check existent
	if !svc.DocumentExists(submitResult.ID) {
		t.Error("DocumentExists() = false for existing document")
	}
}

func TestListPendingDocuments(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	// Create some documents
	for i, name := range []string{"doc1.md", "doc2.md", "doc3.md"} {
		fullPath := filepath.Join(repoRoot, name)
		if err := os.WriteFile(fullPath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
		submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
			Path:      name,
			Type:      "design",
			Title:     name,
			CreatedBy: "tester",
		})
		if err != nil {
			t.Fatalf("SubmitDocument(%s) error = %v", name, err)
		}

		// Approve the first document
		if i == 0 {
			_, err = svc.ApproveDocument(ApproveDocumentInput{
				ID:         submitResult.ID,
				ApprovedBy: "reviewer",
			})
			if err != nil {
				t.Fatalf("ApproveDocument(%s) error = %v", name, err)
			}
		}
	}

	// List pending (should be 2)
	pending, err := svc.ListPendingDocuments()
	if err != nil {
		t.Fatalf("ListPendingDocuments() error = %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("len = %d, want 2", len(pending))
	}
}

func TestIsValidEntityID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id   string
		want bool
	}{
		{"FEAT-123", true},
		{"TASK-abc", true},
		{"BUG-01ABC", true},
		{"DEC-xyz", true},
		{"EPIC-TEST", true},
		{"P1-basic", true},
		{"X99-test", true},
		{"invalid", false},
		{"", false},
		{"123-numeric", false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			got := isValidEntityID(tc.id)
			if got != tc.want {
				t.Errorf("isValidEntityID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

// --- Lifecycle hook test helpers ---

type mockEntityHook struct {
	transitions []transitionCall
	docRefs     []docRefCall
	entityType  string
	status      string
	err         error
}

type transitionCall struct {
	entityID  string
	newStatus string
}

type docRefCall struct {
	entityID string
	docField string
	docID    string
}

func (m *mockEntityHook) TransitionStatus(entityID, newStatus string) error {
	m.transitions = append(m.transitions, transitionCall{entityID, newStatus})
	return m.err
}

func (m *mockEntityHook) SetDocumentRef(entityID, docField, docID string) error {
	m.docRefs = append(m.docRefs, docRefCall{entityID, docField, docID})
	return m.err
}

func (m *mockEntityHook) GetEntityStatus(entityID string) (string, string, error) {
	return m.entityType, m.status, m.err
}

// --- Lifecycle hook integration tests ---

func TestApproveDocument_TransitionsFeatureOnSpecApproval(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	mock := &mockEntityHook{entityType: "feature", status: "specifying"}
	svc.SetEntityHook(mock)

	docPath := "spec.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("spec content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "specification",
		Title:     "Feature Spec",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	// Reset recorded calls before the operation under test
	mock.transitions = nil

	approveResult, err := svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	if approveResult.Status != string(model.DocumentStatusApproved) {
		t.Errorf("Status = %q, want %q", approveResult.Status, model.DocumentStatusApproved)
	}

	if len(mock.transitions) != 1 {
		t.Fatalf("expected 1 transition call, got %d", len(mock.transitions))
	}
	if mock.transitions[0].entityID != "FEAT-123" {
		t.Errorf("transition entityID = %q, want %q", mock.transitions[0].entityID, "FEAT-123")
	}
	if mock.transitions[0].newStatus != "dev-planning" {
		t.Errorf("transition newStatus = %q, want %q", mock.transitions[0].newStatus, "dev-planning")
	}
}

func TestApproveDocument_TransitionsPlanOnDesignApproval(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	mock := &mockEntityHook{entityType: "plan", status: "designing"}
	svc.SetEntityHook(mock)

	docPath := "design.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("design content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Plan Design",
		Owner:     "P1-basic-ui",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	mock.transitions = nil

	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	if len(mock.transitions) != 1 {
		t.Fatalf("expected 1 transition call, got %d", len(mock.transitions))
	}
	if mock.transitions[0].entityID != "P1-basic-ui" {
		t.Errorf("transition entityID = %q, want %q", mock.transitions[0].entityID, "P1-basic-ui")
	}
	if mock.transitions[0].newStatus != "active" {
		t.Errorf("transition newStatus = %q, want %q", mock.transitions[0].newStatus, "active")
	}
}

func TestSupersedeDocument_RevertsFeatureOnSpecSupersession(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	mock := &mockEntityHook{entityType: "feature", status: "dev-planning"}
	svc.SetEntityHook(mock)

	for _, name := range []string{"spec-v1.md", "spec-v2.md"} {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	submit1, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "spec-v1.md",
		Type:      "specification",
		Title:     "Spec V1",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(v1) error = %v", err)
	}

	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submit1.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument(v1) error = %v", err)
	}

	submit2, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "spec-v2.md",
		Type:      "specification",
		Title:     "Spec V2",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(v2) error = %v", err)
	}

	// Reset before the operation under test
	mock.transitions = nil

	_, err = svc.SupersedeDocument(SupersedeDocumentInput{
		ID:           submit1.ID,
		SupersededBy: submit2.ID,
	})
	if err != nil {
		t.Fatalf("SupersedeDocument() error = %v", err)
	}

	if len(mock.transitions) != 1 {
		t.Fatalf("expected 1 transition call, got %d", len(mock.transitions))
	}
	if mock.transitions[0].entityID != "FEAT-123" {
		t.Errorf("transition entityID = %q, want %q", mock.transitions[0].entityID, "FEAT-123")
	}
	if mock.transitions[0].newStatus != "specifying" {
		t.Errorf("transition newStatus = %q, want %q", mock.transitions[0].newStatus, "specifying")
	}
}

func TestSubmitDocument_SetsDocRefOnOwner(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	mock := &mockEntityHook{entityType: "feature", status: "proposed"}
	svc.SetEntityHook(mock)

	docPath := "my-spec.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("spec content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "specification",
		Title:     "Test Spec",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	if len(mock.docRefs) != 1 {
		t.Fatalf("expected 1 docRef call, got %d", len(mock.docRefs))
	}
	if mock.docRefs[0].entityID != "FEAT-123" {
		t.Errorf("docRef entityID = %q, want %q", mock.docRefs[0].entityID, "FEAT-123")
	}
	if mock.docRefs[0].docField != "spec" {
		t.Errorf("docRef docField = %q, want %q", mock.docRefs[0].docField, "spec")
	}
	if mock.docRefs[0].docID != result.ID {
		t.Errorf("docRef docID = %q, want %q", mock.docRefs[0].docID, result.ID)
	}
}

func TestSubmitDocument_TransitionsFeatureOnDesignCreation(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	mock := &mockEntityHook{entityType: "feature", status: "proposed"}
	svc.SetEntityHook(mock)

	docPath := "design.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("design content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test Design",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	if len(mock.transitions) != 1 {
		t.Fatalf("expected 1 transition call, got %d", len(mock.transitions))
	}
	if mock.transitions[0].entityID != "FEAT-123" {
		t.Errorf("transition entityID = %q, want %q", mock.transitions[0].entityID, "FEAT-123")
	}
	if mock.transitions[0].newStatus != "designing" {
		t.Errorf("transition newStatus = %q, want %q", mock.transitions[0].newStatus, "designing")
	}
}

// ─── DocEntityTransition tests (Track B B.9 / Track I prerequisite) ──────────

func TestApproveDocument_ReportsEntityTransition(t *testing.T) {
	// Verifies that ApproveDocument populates result.EntityTransition when the
	// approval triggers a lifecycle transition on the owning entity. Track I
	// uses this to push status_transition side effects (spec §30.2 criterion 7).
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	// Feature is in "specifying" status; approving the spec advances it to "dev-planning".
	mock := &mockEntityHook{entityType: "feature", status: "specifying"}
	svc.SetEntityHook(mock)

	docPath := "spec.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("spec content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "specification",
		Title:     "Feature Spec",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	approveResult, err := svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	if approveResult.EntityTransition == nil {
		t.Fatal("EntityTransition is nil; want non-nil (approval should report the triggered transition)")
	}
	et := approveResult.EntityTransition
	if et.EntityID != "FEAT-123" {
		t.Errorf("EntityTransition.EntityID = %q, want FEAT-123", et.EntityID)
	}
	if et.EntityType != "feature" {
		t.Errorf("EntityTransition.EntityType = %q, want feature", et.EntityType)
	}
	if et.FromStatus != "specifying" {
		t.Errorf("EntityTransition.FromStatus = %q, want specifying", et.FromStatus)
	}
	if et.ToStatus != "dev-planning" {
		t.Errorf("EntityTransition.ToStatus = %q, want dev-planning", et.ToStatus)
	}
}

func TestApproveDocument_NoEntityTransition_WhenAlreadyAtTargetStatus(t *testing.T) {
	// Verifies that EntityTransition is nil when the entity is already at the
	// target status (idempotent approval — no real transition occurred).
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	// Feature is already in "dev-planning" — approving the spec is a no-op.
	mock := &mockEntityHook{entityType: "feature", status: "dev-planning"}
	svc.SetEntityHook(mock)

	docPath := "spec.md"
	if err := os.WriteFile(filepath.Join(repoRoot, docPath), []byte("spec content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	submitResult, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "specification",
		Title:     "Feature Spec",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	approveResult, err := svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	if approveResult.EntityTransition != nil {
		t.Errorf("EntityTransition = %+v, want nil (entity already at target status)", approveResult.EntityTransition)
	}
}

func TestSupersedeDocument_ReportsEntityTransition(t *testing.T) {
	// Verifies that SupersedeDocument populates result.EntityTransition when
	// supersession triggers a backward lifecycle transition on the owning entity.
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	// Feature is in "dev-planning"; superseding the spec reverts it to "specifying".
	mock := &mockEntityHook{entityType: "feature", status: "dev-planning"}
	svc.SetEntityHook(mock)

	for _, name := range []string{"spec-v1.md", "spec-v2.md"} {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	v1, err := svc.SubmitDocument(SubmitDocumentInput{
		Path: "spec-v1.md", Type: "specification", Title: "Spec V1",
		Owner: "FEAT-123", CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(v1) error = %v", err)
	}
	_, err = svc.ApproveDocument(ApproveDocumentInput{ID: v1.ID, ApprovedBy: "reviewer"})
	if err != nil {
		t.Fatalf("ApproveDocument(v1) error = %v", err)
	}

	v2, err := svc.SubmitDocument(SubmitDocumentInput{
		Path: "spec-v2.md", Type: "specification", Title: "Spec V2",
		Owner: "FEAT-123", CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(v2) error = %v", err)
	}

	supersedeResult, err := svc.SupersedeDocument(SupersedeDocumentInput{
		ID:           v1.ID,
		SupersededBy: v2.ID,
	})
	if err != nil {
		t.Fatalf("SupersedeDocument() error = %v", err)
	}

	if supersedeResult.EntityTransition == nil {
		t.Fatal("EntityTransition is nil; want non-nil (supersession should report the backward transition)")
	}
	et := supersedeResult.EntityTransition
	if et.EntityID != "FEAT-123" {
		t.Errorf("EntityTransition.EntityID = %q, want FEAT-123", et.EntityID)
	}
	if et.EntityType != "feature" {
		t.Errorf("EntityTransition.EntityType = %q, want feature", et.EntityType)
	}
	if et.FromStatus != "dev-planning" {
		t.Errorf("EntityTransition.FromStatus = %q, want dev-planning", et.FromStatus)
	}
	if et.ToStatus != "specifying" {
		t.Errorf("EntityTransition.ToStatus = %q, want specifying", et.ToStatus)
	}
}

func TestOperations_NoHook_StillWork(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)
	// No hook set — all operations must still work

	for _, name := range []string{"doc1.md", "doc2.md"} {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	submit1, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "doc1.md",
		Type:      "design",
		Title:     "Design",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}

	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submit1.ID,
		ApprovedBy: "reviewer",
	})
	if err != nil {
		t.Fatalf("ApproveDocument() error = %v", err)
	}

	submit2, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "doc2.md",
		Type:      "design",
		Title:     "Design V2",
		Owner:     "FEAT-123",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(doc2) error = %v", err)
	}

	_, err = svc.SupersedeDocument(SupersedeDocumentInput{
		ID:           submit1.ID,
		SupersededBy: submit2.ID,
	})
	if err != nil {
		t.Fatalf("SupersedeDocument() error = %v", err)
	}
}

// TestSubmitDocument_NewDocumentTypes verifies that SubmitDocument accepts the
// plan and retrospective document types added in P11.
func TestSubmitDocument_NewDocumentTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		docType string
		dir     string
	}{
		{"plan", "work/plan"},
		{"retrospective", "work/retro"},
	}

	for _, tc := range cases {
		t.Run(tc.docType, func(t *testing.T) {
			stateRoot := t.TempDir()
			repoRoot := t.TempDir()
			svc := NewDocumentService(stateRoot, repoRoot)

			docPath := tc.dir + "/test-doc.md"
			fullPath := filepath.Join(repoRoot, docPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(fullPath, []byte("# Test\n\nContent."), 0o644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			result, err := svc.SubmitDocument(SubmitDocumentInput{
				Path:      docPath,
				Type:      tc.docType,
				Title:     "Test " + tc.docType + " document",
				CreatedBy: "tester",
			})
			if err != nil {
				t.Fatalf("SubmitDocument(type=%q) error = %v", tc.docType, err)
			}
			if result.Type != tc.docType {
				t.Errorf("Type = %q, want %q", result.Type, tc.docType)
			}
		})
	}
}

func TestAttachQualityEvaluation_Success(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/design/test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test\n\nContent."), 0o644); err != nil {
		t.Fatal(err)
	}

	submitted, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Test Design",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}

	eval := model.QualityEvaluation{
		OverallScore: 0.85,
		Pass:         true,
		EvaluatedAt:  time.Now().UTC(),
		Evaluator:    "claude-sonnet-4.5",
		Dimensions: map[string]float64{
			"clarity":      0.9,
			"completeness": 0.8,
		},
	}

	result, err := svc.AttachQualityEvaluation(AttachEvaluationInput{
		ID:         submitted.ID,
		Evaluation: eval,
	})
	if err != nil {
		t.Fatalf("AttachQualityEvaluation: %v", err)
	}

	if result.QualityEvaluation == nil {
		t.Fatal("QualityEvaluation should not be nil")
	}
	if result.QualityEvaluation.OverallScore != 0.85 {
		t.Errorf("OverallScore = %g, want 0.85", result.QualityEvaluation.OverallScore)
	}
	if !result.QualityEvaluation.Pass {
		t.Error("Pass = false, want true")
	}
	if result.QualityEvaluation.Evaluator != "claude-sonnet-4.5" {
		t.Errorf("Evaluator = %q, want claude-sonnet-4.5", result.QualityEvaluation.Evaluator)
	}
}

func TestAttachQualityEvaluation_ValidationErrors(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/design/test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}
	submitted, err := svc.SubmitDocument(SubmitDocumentInput{
		Path: docPath, Type: "design", Title: "T", CreatedBy: "u",
	})
	if err != nil {
		t.Fatal(err)
	}

	base := model.QualityEvaluation{
		OverallScore: 0.8,
		Pass:         true,
		EvaluatedAt:  time.Now().UTC(),
		Evaluator:    "model",
		Dimensions:   map[string]float64{"clarity": 0.8},
	}

	tests := []struct {
		name   string
		mutate func(*model.QualityEvaluation)
	}{
		{"missing id", nil}, // handled separately
		{"empty evaluator", func(e *model.QualityEvaluation) { e.Evaluator = "" }},
		{"empty dimensions", func(e *model.QualityEvaluation) { e.Dimensions = nil }},
		{"zero evaluated_at", func(e *model.QualityEvaluation) { e.EvaluatedAt = time.Time{} }},
		{"score too high", func(e *model.QualityEvaluation) { e.OverallScore = 1.1 }},
		{"score negative", func(e *model.QualityEvaluation) { e.OverallScore = -0.1 }},
		{"dim score invalid", func(e *model.QualityEvaluation) { e.Dimensions["clarity"] = 1.5 }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.name == "missing id" {
				_, err := svc.AttachQualityEvaluation(AttachEvaluationInput{
					ID:         "",
					Evaluation: base,
				})
				if err == nil {
					t.Error("expected error for missing id")
				}
				return
			}
			eval := base
			tc.mutate(&eval)
			_, err := svc.AttachQualityEvaluation(AttachEvaluationInput{
				ID:         submitted.ID,
				Evaluation: eval,
			})
			if err == nil {
				t.Errorf("test %q: expected error", tc.name)
			}
		})
	}
}

func TestAttachQualityEvaluation_Persistence(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	svc := NewDocumentService(stateRoot, repoRoot)

	docPath := "work/design/test.md"
	fullPath := filepath.Join(repoRoot, docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}
	submitted, err := svc.SubmitDocument(SubmitDocumentInput{
		Path: docPath, Type: "design", Title: "T", CreatedBy: "u",
	})
	if err != nil {
		t.Fatal(err)
	}

	eval := model.QualityEvaluation{
		OverallScore: 0.75,
		Pass:         true,
		EvaluatedAt:  time.Now().UTC(),
		Evaluator:    "test-model",
		Dimensions:   map[string]float64{"a": 0.7, "b": 0.8},
	}
	if _, err := svc.AttachQualityEvaluation(AttachEvaluationInput{ID: submitted.ID, Evaluation: eval}); err != nil {
		t.Fatalf("AttachQualityEvaluation: %v", err)
	}

	// Re-fetch via GetDocument
	got, err := svc.GetDocument(submitted.ID, false)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	if got.QualityEvaluation == nil {
		t.Fatal("GetDocument: QualityEvaluation is nil after persist")
	}
	if got.QualityEvaluation.OverallScore != 0.75 {
		t.Errorf("OverallScore = %g, want 0.75", got.QualityEvaluation.OverallScore)
	}
}
