package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// makeDocFile creates a file at relPath (relative to repoRoot) and returns repoRoot.
func makeDocFile(t *testing.T, relPath string) string {
	t.Helper()
	repoRoot := t.TempDir()
	full := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return repoRoot
}

// newSvc returns a DocumentService backed by temp dirs, with the given repoRoot.
func newSvc(t *testing.T, repoRoot string) *DocumentService {
	t.Helper()
	stateRoot := t.TempDir()
	return NewDocumentService(stateRoot, repoRoot)
}

// submitAt creates a file at relPath and calls SubmitDocument with the given type.
func submitAt(t *testing.T, relPath, docType string) error {
	t.Helper()
	repoRoot := makeDocFile(t, relPath)
	svc := newSvc(t, repoRoot)
	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      relPath,
		Type:      docType,
		Title:     "Test",
		CreatedBy: "tester",
	})
	return err
}

// submitAtAndGet creates a file, submits it, and returns the DocumentResult.
func submitAtAndGet(t *testing.T, relPath, docType string) (string, error) {
	t.Helper()
	repoRoot := makeDocFile(t, relPath)
	svc := newSvc(t, repoRoot)
	res, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      relPath,
		Type:      docType,
		Title:     "Test",
		CreatedBy: "tester",
	})
	return res.Type, err
}

// ---------------------------------------------------------------------------
// AC-001: type "review" is accepted and stored as "review"
// ---------------------------------------------------------------------------

func TestAC001_TypeReview(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/review-something.md", "review")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "review" {
		t.Errorf("Type = %q, want %q", typ, "review")
	}
}

// ---------------------------------------------------------------------------
// AC-002: type "proposal" is accepted and stored as "proposal"
// ---------------------------------------------------------------------------

func TestAC002_TypeProposal(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/proposal-something.md", "proposal")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "proposal" {
		t.Errorf("Type = %q, want %q", typ, "proposal")
	}
}

// ---------------------------------------------------------------------------
// AC-003: invalid type error lists only 8 user-facing types
// ---------------------------------------------------------------------------

func TestAC003_InvalidTypeListsEightUserFacingTypes(t *testing.T) {
	t.Parallel()
	err := submitAt(t, "work/_project/spec-thing.md", "foo")
	if err == nil {
		t.Fatal("expected error for type 'foo'")
	}
	msg := err.Error()
	for _, want := range []string{"design", "spec", "dev-plan", "review", "report", "research", "retro", "proposal"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing type %q: %q", want, msg)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-008: invalid type error does NOT list policy or rca
// ---------------------------------------------------------------------------

func TestAC008_InvalidTypeErrorExcludesPolicyRca(t *testing.T) {
	t.Parallel()
	err := submitAt(t, "work/_project/spec-thing.md", "foo")
	if err == nil {
		t.Fatal("expected error for type 'foo'")
	}
	msg := err.Error()
	if strings.Contains(msg, "policy") {
		t.Errorf("error message should not contain 'policy': %q", msg)
	}
	if strings.Contains(msg, "rca") {
		t.Errorf("error message should not contain 'rca': %q", msg)
	}
}

// ---------------------------------------------------------------------------
// AC-004: type "specification" is normalised to "spec" on registration
// ---------------------------------------------------------------------------

func TestAC004_SpecificationNormalisedToSpec(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/spec-something.md", "specification")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "spec" {
		t.Errorf("Type = %q, want %q", typ, "spec")
	}
}

// ---------------------------------------------------------------------------
// AC-005: type "retrospective" is normalised to "retro" on registration
// ---------------------------------------------------------------------------

func TestAC005_RetrospectiveNormalisedToRetro(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/retro-something.md", "retrospective")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "retro" {
		t.Errorf("Type = %q, want %q", typ, "retro")
	}
}

// ---------------------------------------------------------------------------
// AC-006: type "policy" is accepted and stored as "policy"
// ---------------------------------------------------------------------------

func TestAC006_TypePolicy(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/policy-something.md", "policy")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "policy" {
		t.Errorf("Type = %q, want %q", typ, "policy")
	}
}

// ---------------------------------------------------------------------------
// AC-007: type "rca" is accepted and stored as "rca"
// ---------------------------------------------------------------------------

func TestAC007_TypeRCA(t *testing.T) {
	t.Parallel()
	typ, err := submitAtAndGet(t, "work/_project/rca-something.md", "rca")
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if typ != "rca" {
		t.Errorf("Type = %q, want %q", typ, "rca")
	}
}

// ---------------------------------------------------------------------------
// AC-009: P37 plan folder with P37-spec-something.md passes filename validation
// ---------------------------------------------------------------------------

func TestAC009_PlanFilenameWithPlanIDPasses(t *testing.T) {
	t.Parallel()
	path := "work/P37-file-names-actions/P37-spec-something.md"
	if err := validateDocumentFilename(path); err != nil {
		t.Errorf("validateDocumentFilename(%q) error = %v, want nil", path, err)
	}
}

// ---------------------------------------------------------------------------
// AC-010: P37 plan folder with spec-something.md (no plan prefix) fails
// ---------------------------------------------------------------------------

func TestAC010_PlanFolderWithoutPlanPrefixFails(t *testing.T) {
	t.Parallel()
	path := "work/P37-file-names-actions/spec-something.md"
	if err := validateDocumentFilename(path); err == nil {
		t.Errorf("validateDocumentFilename(%q) want error for missing plan ID prefix", path)
	}
}

// ---------------------------------------------------------------------------
// AC-011: feature-scoped filename P37-F2-spec-enforcement.md passes
// ---------------------------------------------------------------------------

func TestAC011_FeatureScopedFilenamePassesValidation(t *testing.T) {
	t.Parallel()
	path := "work/P37-file-names-actions/P37-F2-spec-enforcement.md"
	if err := validateDocumentFilename(path); err != nil {
		t.Errorf("validateDocumentFilename(%q) error = %v, want nil", path, err)
	}
}

// ---------------------------------------------------------------------------
// AC-012: lowercase plan prefix p37-spec-something.md passes (case-insensitive)
// ---------------------------------------------------------------------------

func TestAC012_LowercasePlanPrefixPasses(t *testing.T) {
	t.Parallel()
	path := "work/p37-file-names-actions/p37-spec-something.md"
	if err := validateDocumentFilename(path); err != nil {
		t.Errorf("validateDocumentFilename(%q) error = %v, want nil", path, err)
	}
}

// ---------------------------------------------------------------------------
// AC-013: P37 file in P25 folder fails folder validation
// ---------------------------------------------------------------------------

func TestAC013_PlanIDMismatchFails(t *testing.T) {
	t.Parallel()
	path := "work/P25-other-plan/P37-spec-something.md"
	if err := validateDocumentFolder(path); err == nil {
		t.Errorf("validateDocumentFolder(%q) want error for plan ID mismatch", path)
	}
}

// ---------------------------------------------------------------------------
// AC-014: work/_project/research-ai-orchestration.md passes folder validation
// ---------------------------------------------------------------------------

func TestAC014_ProjectFolderResearchPasses(t *testing.T) {
	t.Parallel()
	path := "work/_project/research-ai-orchestration.md"
	if err := validateDocumentFolder(path); err != nil {
		t.Errorf("validateDocumentFolder(%q) error = %v, want nil", path, err)
	}
}

// ---------------------------------------------------------------------------
// AC-015: error message on failure is specific
// ---------------------------------------------------------------------------

func TestAC015_ErrorMessageIsSpecific(t *testing.T) {
	t.Parallel()

	// Filename error should name the expected pattern.
	filenameErr := validateDocumentFilename("work/P37-file-names-actions/spec-something.md")
	if filenameErr == nil {
		t.Fatal("expected filename validation error")
	}
	if !strings.Contains(filenameErr.Error(), "P37") {
		t.Errorf("filename error should contain plan ID 'P37': %q", filenameErr.Error())
	}

	// Folder error should name the expected directory.
	folderErr := validateDocumentFolder("work/P25-other-plan/P37-spec-something.md")
	if folderErr == nil {
		t.Fatal("expected folder validation error")
	}
	if !strings.Contains(folderErr.Error(), "work/P37") {
		t.Errorf("folder error should contain expected path 'work/P37': %q", folderErr.Error())
	}
}

// ---------------------------------------------------------------------------
// AC-016: work/templates/ files are exempt from all validation
// ---------------------------------------------------------------------------

func TestAC016_TemplateFilesExemptFromValidation(t *testing.T) {
	t.Parallel()
	repoRoot := makeDocFile(t, "work/templates/my-template.md")
	svc := newSvc(t, repoRoot)
	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "work/templates/my-template.md",
		Type:      "design",
		Title:     "Template",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Errorf("SubmitDocument() for templates/ path error = %v, want nil", err)
	}
}

// ---------------------------------------------------------------------------
// AC-017: docs/ files are exempt from folder validation
// ---------------------------------------------------------------------------

func TestAC017_DocsFilesExemptFromFolderValidation(t *testing.T) {
	t.Parallel()
	repoRoot := makeDocFile(t, "docs/architecture/overview.md")
	svc := newSvc(t, repoRoot)
	_, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "docs/architecture/overview.md",
		Type:      "design",
		Title:     "Overview",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Errorf("SubmitDocument() for docs/ path error = %v, want nil", err)
	}
}

// ---------------------------------------------------------------------------
// AC-023: folder mismatch error includes specific expected directory
// ---------------------------------------------------------------------------

func TestAC023_FolderErrorIncludesSpecificDirectory(t *testing.T) {
	t.Parallel()
	path := "work/P25-other-plan/P37-spec-something.md"
	err := validateDocumentFolder(path)
	if err == nil {
		t.Fatal("expected folder validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "work/P37") {
		t.Errorf("error message should include specific expected directory 'work/P37': %q", msg)
	}
	// Must not be generic only.
	if !strings.Contains(msg, "P37") {
		t.Errorf("error is too generic, missing plan ID: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// AC-018: loading an existing record with a non-conforming path succeeds
// ---------------------------------------------------------------------------

func TestAC018_LoadNonConformingPathSucceeds(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := storage.NewDocumentStore(root)

	// Write a record whose path doesn't match any current template.
	id := "PROJ/old-style-doc"
	fields := map[string]any{
		"id":           id,
		"path":         "work/some-random-dir/my_OLD_doc.md",
		"type":         "design",
		"title":        "Old Style Doc",
		"status":       "draft",
		"content_hash": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"created":      "2026-01-01T00:00:00Z",
		"created_by":   "sam",
		"updated":      "2026-01-01T00:00:00Z",
	}
	rec := storage.DocumentRecord{ID: id, Fields: fields}
	if _, err := store.Write(rec); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil (no validation on load)", err)
	}
	doc := storage.RecordToDocument(loaded)
	if doc.Path != "work/some-random-dir/my_OLD_doc.md" {
		t.Errorf("Path = %q, want %q", doc.Path, "work/some-random-dir/my_OLD_doc.md")
	}
	if doc.Title != "Old Style Doc" {
		t.Errorf("Title = %q, want %q", doc.Title, "Old Style Doc")
	}
}

// ---------------------------------------------------------------------------
// AC-019: "specification" stored on disk is normalised to "spec" on deserialise
// ---------------------------------------------------------------------------

func TestAC019_SpecificationNormalisedToSpecOnDeserialise(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := storage.NewDocumentStore(root)

	id := "PROJ/legacy-spec"
	fields := map[string]any{
		"id":           id,
		"path":         "work/design/old-spec.md",
		"type":         "specification",
		"title":        "Legacy Spec",
		"status":       "draft",
		"content_hash": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"created":      "2026-01-01T00:00:00Z",
		"created_by":   "sam",
		"updated":      "2026-01-01T00:00:00Z",
	}
	rec := storage.DocumentRecord{ID: id, Fields: fields}
	if _, err := store.Write(rec); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	doc := storage.RecordToDocument(loaded)
	if doc.Type != model.DocumentTypeSpec {
		t.Errorf("Type = %q, want %q", doc.Type, model.DocumentTypeSpec)
	}
}

// ---------------------------------------------------------------------------
// AC-020: "retrospective" stored on disk is normalised to "retro" on deserialise
// ---------------------------------------------------------------------------

func TestAC020_RetrospectiveNormalisedToRetroOnDeserialise(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := storage.NewDocumentStore(root)

	id := "PROJ/legacy-retro"
	fields := map[string]any{
		"id":           id,
		"path":         "work/design/old-retro.md",
		"type":         "retrospective",
		"title":        "Legacy Retro",
		"status":       "draft",
		"content_hash": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"created":      "2026-01-01T00:00:00Z",
		"created_by":   "sam",
		"updated":      "2026-01-01T00:00:00Z",
	}
	rec := storage.DocumentRecord{ID: id, Fields: fields}
	if _, err := store.Write(rec); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	doc := storage.RecordToDocument(loaded)
	if doc.Type != model.DocumentTypeRetro {
		t.Errorf("Type = %q, want %q", doc.Type, model.DocumentTypeRetro)
	}
}

// ---------------------------------------------------------------------------
// AC-021: "plan" stored on disk loads without error
// ---------------------------------------------------------------------------

func TestAC021_LegacyPlanTypeLoadsWithoutError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := storage.NewDocumentStore(root)

	id := "PROJ/legacy-plan"
	fields := map[string]any{
		"id":           id,
		"path":         "work/design/old-plan.md",
		"type":         "plan",
		"title":        "Legacy Plan",
		"status":       "draft",
		"content_hash": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"created":      "2026-01-01T00:00:00Z",
		"created_by":   "sam",
		"updated":      "2026-01-01T00:00:00Z",
	}
	rec := storage.DocumentRecord{ID: id, Fields: fields}
	if _, err := store.Write(rec); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	doc := storage.RecordToDocument(loaded)
	if doc.Title != "Legacy Plan" {
		t.Errorf("Title = %q, want %q", doc.Title, "Legacy Plan")
	}
}

// ---------------------------------------------------------------------------
// Additional: validateDocumentFilename unit tests for edge cases
// ---------------------------------------------------------------------------

func TestValidateDocumentFilename_TemplatesExempt(t *testing.T) {
	t.Parallel()
	paths := []string{
		"work/templates/my-template.md",
		"work/templates/P37-spec-thing.md",
	}
	for _, p := range paths {
		if err := validateDocumentFilename(p); err != nil {
			t.Errorf("validateDocumentFilename(%q) = %v, want nil", p, err)
		}
	}
}

func TestValidateDocumentFilename_DocsExempt(t *testing.T) {
	t.Parallel()
	paths := []string{
		"docs/architecture/overview.md",
		"docs/random-name.md",
	}
	for _, p := range paths {
		if err := validateDocumentFilename(p); err != nil {
			t.Errorf("validateDocumentFilename(%q) = %v, want nil", p, err)
		}
	}
}

func TestValidateDocumentFilename_ProjectFolder(t *testing.T) {
	t.Parallel()
	valid := []string{
		"work/_project/spec-something.md",
		"work/_project/design.md",
		"work/_project/dev-plan-overview.md",
		"work/_project/retro-q1.md",
	}
	for _, p := range valid {
		if err := validateDocumentFilename(p); err != nil {
			t.Errorf("validateDocumentFilename(%q) = %v, want nil", p, err)
		}
	}

	invalid := []string{
		"work/_project/P37-spec-something.md", // has plan prefix, wrong folder
		"work/_project/something-random.md",   // not a type prefix
	}
	for _, p := range invalid {
		if err := validateDocumentFilename(p); err == nil {
			t.Errorf("validateDocumentFilename(%q) want error", p)
		}
	}
}

func TestValidateDocumentFilename_PlanFolder(t *testing.T) {
	t.Parallel()
	valid := []string{
		"work/P37-file-names-actions/P37-spec-something.md",
		"work/P37-file-names-actions/P37-design-overview.md",
		"work/P37-file-names-actions/P37-F2-spec-enforcement.md",
		"work/P37-file-names-actions/P37-F12-dev-plan-impl.md",
		"work/p37-file-names-actions/p37-spec-something.md",
	}
	for _, p := range valid {
		if err := validateDocumentFilename(p); err != nil {
			t.Errorf("validateDocumentFilename(%q) = %v, want nil", p, err)
		}
	}

	invalid := []string{
		"work/P37-file-names-actions/spec-something.md",     // no plan prefix
		"work/P37-file-names-actions/P25-spec-something.md", // wrong plan ID
	}
	for _, p := range invalid {
		if err := validateDocumentFilename(p); err == nil {
			t.Errorf("validateDocumentFilename(%q) want error", p)
		}
	}
}

func TestValidateDocumentFolder_TemplatesExempt(t *testing.T) {
	t.Parallel()
	if err := validateDocumentFolder("work/templates/spec-thing.md"); err != nil {
		t.Errorf("validateDocumentFolder() = %v, want nil for templates path", err)
	}
}

func TestValidateDocumentFolder_DocsExempt(t *testing.T) {
	t.Parallel()
	if err := validateDocumentFolder("docs/architecture/overview.md"); err != nil {
		t.Errorf("validateDocumentFolder() = %v, want nil for docs/ path", err)
	}
}

func TestValidateDocumentFolder_PlanIDMustMatchFolder(t *testing.T) {
	t.Parallel()
	// Correct: P37 file in P37 folder.
	if err := validateDocumentFolder("work/P37-file-names/P37-spec-thing.md"); err != nil {
		t.Errorf("validateDocumentFolder() = %v, want nil", err)
	}
	// Wrong: P37 file in P25 folder.
	if err := validateDocumentFolder("work/P25-other/P37-spec-thing.md"); err == nil {
		t.Errorf("validateDocumentFolder() want error for P37 file in P25 folder")
	}
}

func TestValidateDocumentFolder_TypePrefixMustBeInProject(t *testing.T) {
	t.Parallel()
	// Correct: type-prefix file in _project.
	if err := validateDocumentFolder("work/_project/spec-thing.md"); err != nil {
		t.Errorf("validateDocumentFolder() = %v, want nil", err)
	}
	// Wrong: type-prefix file in a plan folder.
	if err := validateDocumentFolder("work/P37-plans/spec-thing.md"); err == nil {
		t.Errorf("validateDocumentFolder() want error for type-prefix file outside _project")
	}
}

// ---------------------------------------------------------------------------
// Ensure existing service tests still pass (regression guard):
// SubmitDocument with existing "design" path and type still works.
// ---------------------------------------------------------------------------

func TestAC_Regression_ExistingDesignTypeStillWorks(t *testing.T) {
	t.Parallel()
	repoRoot := makeDocFile(t, "work/_project/design-overview.md")
	svc := newSvc(t, repoRoot)
	res, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      "work/_project/design-overview.md",
		Type:      "design",
		Title:     "Design Overview",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument() error = %v", err)
	}
	if res.Type != "design" {
		t.Errorf("Type = %q, want design", res.Type)
	}
}

// ---------------------------------------------------------------------------
// Verify that the storage package does not call validateDocumentFilename
// on load (REQ-011 guard): RecordToDocument always returns the stored path unchanged.
// ---------------------------------------------------------------------------

func TestAC018_StorageDoesNotValidateOnLoad(t *testing.T) {
	t.Parallel()

	// Create a record with a completely invalid path.
	rec := storage.DocumentRecord{
		ID: "TEST/invalid-path",
		Fields: map[string]any{
			"id":           "TEST/invalid-path",
			"path":         "INVALID_FOLDER/WRONG_NAME.md",
			"type":         "design",
			"title":        "Invalid Path Test",
			"status":       "draft",
			"content_hash": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			"created":      "2026-01-01T00:00:00Z",
			"created_by":   "sam",
			"updated":      "2026-01-01T00:00:00Z",
		},
	}

	root := t.TempDir()
	store := storage.NewDocumentStore(root)
	if _, err := store.Write(rec); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := store.Load("TEST/invalid-path")
	if err != nil {
		t.Fatalf("Load() should not validate path — error = %v", err)
	}
	doc := storage.RecordToDocument(loaded)
	if doc.Path != "INVALID_FOLDER/WRONG_NAME.md" {
		t.Errorf("Path = %q, want unchanged value", doc.Path)
	}
}

// ---------------------------------------------------------------------------
// Verify that model.ValidDocumentType("plan") returns true (C-005).
// ---------------------------------------------------------------------------

func TestValidDocumentType_PlanReturnsTrueForLegacyUse(t *testing.T) {
	t.Parallel()
	if !model.ValidDocumentType("plan") {
		t.Error("ValidDocumentType(\"plan\") = false, want true (legacy type)")
	}
}

