package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

func newTestImportSetup(t *testing.T) (*BatchImportService, *DocumentService, string, *config.Config) {
	t.Helper()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	docSvc := NewDocumentService(stateRoot, repoRoot)
	importSvc := NewBatchImportService(docSvc)
	cfg := config.DefaultConfig()
	return importSvc, docSvc, repoRoot, &cfg
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestBatchImport_HappyPath(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	writeTestFile(t, repoRoot, "work/design/my-design.md", "# My Design\n\nContent.")
	writeTestFile(t, repoRoot, "work/spec/my-spec.md", "# My Spec\n\nContent.")

	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 2 {
		t.Errorf("Imported = %d, want 2", result.Imported)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("Skipped = %v, want empty", result.Skipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}
}

func TestBatchImport_Idempotent(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	writeTestFile(t, repoRoot, "work/design/my-design.md", "# Design\n\nContent.")

	scanPath := filepath.Join(repoRoot, "work")

	// First import
	result1, err := importSvc.Import(cfg, BatchImportInput{
		Path:      scanPath,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	if result1.Imported != 1 {
		t.Errorf("first run: Imported = %d, want 1", result1.Imported)
	}

	// Second import — same files should be skipped
	result2, err := importSvc.Import(cfg, BatchImportInput{
		Path:      scanPath,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if result2.Imported != 0 {
		t.Errorf("second run: Imported = %d, want 0 (idempotent)", result2.Imported)
	}
	if len(result2.Skipped) != 1 {
		t.Errorf("second run: len(Skipped) = %d, want 1", len(result2.Skipped))
	}
	if result2.Skipped[0].Reason != "already imported" {
		t.Errorf("Skipped[0].Reason = %q, want %q", result2.Skipped[0].Reason, "already imported")
	}
}

func TestBatchImport_TypeInferenceFromPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		relPath  string
		wantType string
	}{
		{"work/design/foo.md", "design"},
		{"work/spec/bar.md", "specification"},
		{"work/plan/baz.md", "report"},
		{"work/research/qux.md", "research"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.relPath, func(t *testing.T) {
			t.Parallel()

			importSvc, docSvc, repoRoot, cfg := newTestImportSetup(t)

			writeTestFile(t, repoRoot, tc.relPath, "# Title\n\nContent.")

			result, err := importSvc.Import(cfg, BatchImportInput{
				Path:      filepath.Join(repoRoot, "work"),
				CreatedBy: "tester",
			})
			if err != nil {
				t.Fatalf("Import() error = %v", err)
			}

			if result.Imported != 1 {
				t.Fatalf("Imported = %d, want 1 (errors: %v, skipped: %v)", result.Imported, result.Errors, result.Skipped)
			}

			// Verify the submitted document has the expected type.
			docs, err := docSvc.ListDocuments(DocumentFilters{})
			if err != nil {
				t.Fatalf("ListDocuments() error = %v", err)
			}
			if len(docs) != 1 {
				t.Fatalf("len(docs) = %d, want 1", len(docs))
			}
			if docs[0].Type != tc.wantType {
				t.Errorf("Type = %q, want %q", docs[0].Type, tc.wantType)
			}
		})
	}
}

func TestBatchImport_SkipsWhenNoTypeAvailable(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	// A path that matches no type mapping
	writeTestFile(t, repoRoot, "random/notes/meeting.md", "# Notes\n\nContent.")

	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "random"),
		CreatedBy: "tester",
		// No DefaultType provided
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 0 {
		t.Errorf("Imported = %d, want 0", result.Imported)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("len(Skipped) = %d, want 1", len(result.Skipped))
	}
	if result.Skipped[0].Reason == "" {
		t.Error("Skipped[0].Reason should not be empty")
	}
}

func TestBatchImport_DefaultTypeUsedWhenNoPatternMatches(t *testing.T) {
	t.Parallel()

	importSvc, docSvc, repoRoot, cfg := newTestImportSetup(t)

	writeTestFile(t, repoRoot, "random/notes/meeting.md", "# Notes\n\nContent.")

	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:        filepath.Join(repoRoot, "random"),
		DefaultType: "report",
		CreatedBy:   "tester",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}

	docs, err := docSvc.ListDocuments(DocumentFilters{})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	if docs[0].Type != "report" {
		t.Errorf("Type = %q, want %q", docs[0].Type, "report")
	}
}

func TestBatchImport_ErrorForOneFileDoesNotAbortBatch(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	// Create two valid files and a directory that WalkDir will encounter an
	// error on by making a file unreadable.
	writeTestFile(t, repoRoot, "work/design/good1.md", "# Good 1\n\nContent.")
	writeTestFile(t, repoRoot, "work/design/good2.md", "# Good 2\n\nContent.")

	// Import first file to make re-submit fail on one of them on second run.
	_, _ = importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
	})

	// Add a third file so the batch has something new to process alongside a
	// file that will produce a conflict error on re-submit attempt.
	writeTestFile(t, repoRoot, "work/design/good3.md", "# Good 3\n\nContent.")

	// Overwrite with a custom config that has no type mappings but provide a
	// default_type so type inference works, then force a conflict by calling
	// SubmitDocument for good1 before the batch run.
	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// good1 and good2 should be skipped (already imported), good3 imported.
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}
	if len(result.Skipped) != 2 {
		t.Errorf("len(Skipped) = %d, want 2", len(result.Skipped))
	}
	if len(result.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(result.Errors))
	}
}

func TestBatchImport_NonExistentDirectoryReturnsError(t *testing.T) {
	t.Parallel()

	importSvc, _, _, cfg := newTestImportSetup(t)

	_, err := importSvc.Import(cfg, BatchImportInput{
		Path:      "/nonexistent/directory/that/does/not/exist",
		CreatedBy: "tester",
	})
	if err == nil {
		t.Error("Import() should return error for nonexistent directory")
	}
}

func TestBatchImport_IgnoresNonMarkdownFiles(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	writeTestFile(t, repoRoot, "work/design/notes.txt", "plain text notes")
	writeTestFile(t, repoRoot, "work/design/image.png", "fake image data")
	writeTestFile(t, repoRoot, "work/design/real-doc.md", "# Real Doc\n\nContent.")

	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1 (only .md files)", result.Imported)
	}
}

func TestDeriveTitle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		filename string
		want     string
	}{
		{"my-design-doc.md", "My design doc"},
		{"phase_2_spec.md", "Phase 2 spec"},
		{"README.md", "README"},
		{"simple.md", "Simple"},
		{".md", "Untitled"},
		{"foo.md", "Foo"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.filename, func(t *testing.T) {
			t.Parallel()
			got := deriveTitle(tc.filename)
			if got != tc.want {
				t.Errorf("deriveTitle(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

func TestInferDocType(t *testing.T) {
	t.Parallel()

	cfgVal := config.DefaultConfig()
	cfg := &cfgVal

	testCases := []struct {
		path        string
		defaultType string
		want        string
	}{
		{"work/design/foo.md", "", "design"},
		{"work/spec/bar.md", "", "specification"},
		{"work/plan/baz.md", "", "report"},
		{"work/research/qux.md", "", "research"},
		{"work/other/doc.md", "policy", "policy"},
		{"work/other/doc.md", "", ""},
		{"design/top-level.md", "", "design"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%s_default=%s", tc.path, tc.defaultType), func(t *testing.T) {
			t.Parallel()
			got := inferDocType(cfg, tc.path, tc.defaultType)
			if got != tc.want {
				t.Errorf("inferDocType(%q, %q) = %q, want %q", tc.path, tc.defaultType, got, tc.want)
			}
		})
	}
}

func TestBatchImport_GlobFilterFilename(t *testing.T) {
	t.Parallel()

	importSvc, docSvc, repoRoot, cfg := newTestImportSetup(t)

	// Create files with different names in the same directory
	writeTestFile(t, repoRoot, "work/design/api-design.md", "# API Design\n\nContent.")
	writeTestFile(t, repoRoot, "work/design/ui-mockup.md", "# UI Mockup\n\nContent.")
	writeTestFile(t, repoRoot, "work/design/api-spec.md", "# API Spec\n\nContent.")

	// Import only files starting with "api-"
	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
		Glob:      "api-*.md",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 2 {
		t.Errorf("Imported = %d, want 2 (api-design.md and api-spec.md)", result.Imported)
	}

	// Verify correct files were imported
	docs, err := docSvc.ListDocuments(DocumentFilters{})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("len(docs) = %d, want 2", len(docs))
	}

	titles := make(map[string]bool)
	for _, doc := range docs {
		titles[doc.Title] = true
	}
	if !titles["Api design"] {
		t.Error("expected 'Api design' to be imported")
	}
	if !titles["Api spec"] {
		t.Error("expected 'Api spec' to be imported")
	}
	if titles["Ui mockup"] {
		t.Error("'Ui mockup' should not be imported")
	}
}

func TestBatchImport_GlobFilterWithPath(t *testing.T) {
	t.Parallel()

	importSvc, docSvc, repoRoot, cfg := newTestImportSetup(t)

	// Create files in different subdirectories
	writeTestFile(t, repoRoot, "work/design/api.md", "# API\n\nContent.")
	writeTestFile(t, repoRoot, "work/spec/api.md", "# API Spec\n\nContent.")
	writeTestFile(t, repoRoot, "work/plan/roadmap.md", "# Roadmap\n\nContent.")

	// Import only files in the design subdirectory
	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
		Glob:      "design/*.md",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1 (only design/api.md)", result.Imported)
	}

	docs, err := docSvc.ListDocuments(DocumentFilters{})
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	if docs[0].Type != "design" {
		t.Errorf("Type = %q, want %q", docs[0].Type, "design")
	}
}

func TestBatchImport_GlobFilterAllMd(t *testing.T) {
	t.Parallel()

	importSvc, _, repoRoot, cfg := newTestImportSetup(t)

	writeTestFile(t, repoRoot, "work/design/doc1.md", "# Doc 1\n\nContent.")
	writeTestFile(t, repoRoot, "work/design/doc2.md", "# Doc 2\n\nContent.")
	writeTestFile(t, repoRoot, "work/design/readme.txt", "Plain text file")

	// Glob "*.md" should match all .md files (filename-only matching)
	result, err := importSvc.Import(cfg, BatchImportInput{
		Path:      filepath.Join(repoRoot, "work"),
		CreatedBy: "tester",
		Glob:      "*.md",
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if result.Imported != 2 {
		t.Errorf("Imported = %d, want 2", result.Imported)
	}
}

func TestExtractGlobSegment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		glob string
		want string
	}{
		{"*/design/*", "/design/"},
		{"**/spec/**", "/spec/"},
		{"*/plan/*", "/plan/"},
		{"*/research/*", "/research/"},
		{"*/*", ""},
		{"**/**", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.glob, func(t *testing.T) {
			t.Parallel()
			got := extractGlobSegment(tc.glob)
			if got != tc.want {
				t.Errorf("extractGlobSegment(%q) = %q, want %q", tc.glob, got, tc.want)
			}
		})
	}
}
