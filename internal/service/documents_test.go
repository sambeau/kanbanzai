package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kanbanzai/internal/model"
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

func TestApproveDocument_ContentHashMismatch(t *testing.T) {
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

	// Modify the file after submission
	if err := os.WriteFile(fullPath, []byte("modified content"), 0o644); err != nil {
		t.Fatalf("failed to modify document: %v", err)
	}

	// Try to approve - should fail due to hash mismatch
	_, err = svc.ApproveDocument(ApproveDocumentInput{
		ID:         submitResult.ID,
		ApprovedBy: "reviewer",
	})
	if err == nil {
		t.Fatal("expected error for content hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error = %q, want to contain 'hash mismatch'", err.Error())
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
