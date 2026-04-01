package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newAuditTestSetup creates a DocumentService backed by temp directories and
// returns the service (satisfies DocAuditStore), the repoRoot, and a helper to
// write .md files under repoRoot.
func newAuditTestSetup(t *testing.T) (*DocumentService, string) {
	t.Helper()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	return docSvc, repoRoot
}

// registerDoc is a test helper that submits and approves a document record so
// that AuditDocuments sees it as "registered".
func registerDoc(t *testing.T, svc *DocumentService, repoRoot, relPath string) string {
	t.Helper()
	full := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte("# Test\n\nContent."), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	res, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      relPath,
		Type:      "specification",
		Title:     "Test doc",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument(%s): %v", relPath, err)
	}
	return res.ID
}

// writeFile writes a file under repoRoot without registering it in the store.
func writeAuditFile(t *testing.T, repoRoot, relPath string) {
	t.Helper()
	full := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte("# Unregistered\n\nContent."), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// ─── Basic behaviour ──────────────────────────────────────────────────────────

func TestAuditDocuments_UnregisteredFile(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	writeAuditFile(t, repoRoot, "work/spec/unregistered.md")

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 1 {
		t.Errorf("TotalOnDisk = %d, want 1", result.Summary.TotalOnDisk)
	}
	if result.Summary.Unregistered != 1 {
		t.Errorf("Unregistered = %d, want 1", result.Summary.Unregistered)
	}
	if result.Summary.Registered != 0 {
		t.Errorf("Registered = %d, want 0", result.Summary.Registered)
	}
	if len(result.Unregistered) != 1 {
		t.Fatalf("len(Unregistered) = %d, want 1", len(result.Unregistered))
	}
	if result.Unregistered[0].Path != "work/spec/unregistered.md" {
		t.Errorf("Unregistered[0].Path = %q, want %q",
			result.Unregistered[0].Path, "work/spec/unregistered.md")
	}
}

func TestAuditDocuments_RegisteredFile(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	registerDoc(t, svc, repoRoot, "work/spec/registered.md")

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 1 {
		t.Errorf("TotalOnDisk = %d, want 1", result.Summary.TotalOnDisk)
	}
	if result.Summary.Registered != 1 {
		t.Errorf("Registered = %d, want 1", result.Summary.Registered)
	}
	if result.Summary.Unregistered != 0 {
		t.Errorf("Unregistered = %d, want 0", result.Summary.Unregistered)
	}
	// includeRegistered=false: Registered slice must be nil.
	if result.Registered != nil {
		t.Errorf("Registered slice should be nil when includeRegistered=false")
	}
}

func TestAuditDocuments_IncludeRegistered(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	docID := registerDoc(t, svc, repoRoot, "work/spec/registered.md")

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, true)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Registered == nil {
		t.Fatal("Registered slice should not be nil when includeRegistered=true")
	}
	if len(result.Registered) != 1 {
		t.Fatalf("len(Registered) = %d, want 1", len(result.Registered))
	}
	if result.Registered[0].Path != "work/spec/registered.md" {
		t.Errorf("Registered[0].Path = %q, want %q",
			result.Registered[0].Path, "work/spec/registered.md")
	}
	if result.Registered[0].DocID != docID {
		t.Errorf("Registered[0].DocID = %q, want %q", result.Registered[0].DocID, docID)
	}
}

func TestAuditDocuments_MissingRecord(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	// Register the document, then delete the file so it becomes "missing".
	docID := registerDoc(t, svc, repoRoot, "work/spec/will-be-deleted.md")

	// Remove the file from disk.
	if err := os.Remove(filepath.Join(repoRoot, "work/spec/will-be-deleted.md")); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 0 {
		t.Errorf("TotalOnDisk = %d, want 0", result.Summary.TotalOnDisk)
	}
	if result.Summary.Missing != 1 {
		t.Errorf("Missing = %d, want 1", result.Summary.Missing)
	}
	if len(result.Missing) != 1 {
		t.Fatalf("len(Missing) = %d, want 1", len(result.Missing))
	}
	if result.Missing[0].Path != "work/spec/will-be-deleted.md" {
		t.Errorf("Missing[0].Path = %q, want %q",
			result.Missing[0].Path, "work/spec/will-be-deleted.md")
	}
	if result.Missing[0].DocID != docID {
		t.Errorf("Missing[0].DocID = %q, want %q", result.Missing[0].DocID, docID)
	}
}

// ─── Invariant ────────────────────────────────────────────────────────────────

func TestAuditDocuments_SummaryInvariant(t *testing.T) {
	// registered + unregistered == total_on_disk must always hold.
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	registerDoc(t, svc, repoRoot, "work/spec/registered.md")
	writeAuditFile(t, repoRoot, "work/spec/unregistered-a.md")
	writeAuditFile(t, repoRoot, "work/spec/unregistered-b.md")

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 3 {
		t.Errorf("TotalOnDisk = %d, want 3", result.Summary.TotalOnDisk)
	}
	if result.Summary.Registered+result.Summary.Unregistered != result.Summary.TotalOnDisk {
		t.Errorf("invariant violated: Registered(%d) + Unregistered(%d) != TotalOnDisk(%d)",
			result.Summary.Registered, result.Summary.Unregistered, result.Summary.TotalOnDisk)
	}
}

// ─── Explicit path error (F-01) ───────────────────────────────────────────────

func TestAuditDocuments_ExplicitNonExistentPathReturnsError(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)

	_, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/does-not-exist"}, false)
	if err == nil {
		t.Fatal("AuditDocuments() should return error for explicit non-existent path")
	}
}

func TestAuditDocuments_DefaultDirsMissingDirsSilentlySkipped(t *testing.T) {
	// When dirs is empty (default mode), non-existent directories in the
	// default set must not cause an error.
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	// repoRoot has no subdirectories — all defaultAuditDirs are absent.
	// This must succeed and return an empty result.

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		nil, false)
	if err != nil {
		t.Fatalf("AuditDocuments() with missing default dirs error = %v", err)
	}
	if result.Summary.TotalOnDisk != 0 {
		t.Errorf("TotalOnDisk = %d, want 0", result.Summary.TotalOnDisk)
	}
}

// ─── Non-.md files ignored ────────────────────────────────────────────────────

func TestAuditDocuments_IgnoresNonMarkdownFiles(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	writeAuditFile(t, repoRoot, "work/spec/doc.md")

	// Write non-Markdown files that should be ignored.
	for _, name := range []string{"readme.txt", "image.png", "data.json"} {
		full := filepath.Join(repoRoot, "work/spec", name)
		if err := os.WriteFile(full, []byte("not markdown"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 1 {
		t.Errorf("TotalOnDisk = %d, want 1 (only .md files counted)", result.Summary.TotalOnDisk)
	}
}

// ─── Scoping ──────────────────────────────────────────────────────────────────

func TestAuditDocuments_MissingRecordOutsideScanDirNotReported(t *testing.T) {
	// A store record whose path is outside the scanned directories must not
	// appear in the Missing list.
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)

	// Register a doc in work/design (not in the scan path).
	registerDoc(t, svc, repoRoot, "work/design/outside.md")

	// Create work/spec so the explicit-path check passes (F-01), but leave it
	// empty so no on-disk files are found there.
	if err := os.MkdirAll(filepath.Join(repoRoot, "work/spec"), 0o755); err != nil {
		t.Fatalf("mkdir work/spec: %v", err)
	}

	// Scan only work/spec — work/design record must not show as missing.
	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.Missing != 0 {
		t.Errorf("Missing = %d, want 0 (record is outside scanned dir)", result.Summary.Missing)
	}
}

func TestAuditDocuments_MultipleDirectoriesScanned(t *testing.T) {
	t.Parallel()

	svc, repoRoot := newAuditTestSetup(t)
	writeAuditFile(t, repoRoot, "work/spec/spec-doc.md")
	writeAuditFile(t, repoRoot, "work/design/design-doc.md")

	result, err := AuditDocuments(context.Background(), svc, repoRoot,
		[]string{"work/spec", "work/design"}, false)
	if err != nil {
		t.Fatalf("AuditDocuments() error = %v", err)
	}

	if result.Summary.TotalOnDisk != 2 {
		t.Errorf("TotalOnDisk = %d, want 2", result.Summary.TotalOnDisk)
	}
	if result.Summary.Unregistered != 2 {
		t.Errorf("Unregistered = %d, want 2", result.Summary.Unregistered)
	}
}
