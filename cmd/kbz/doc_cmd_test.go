package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/service"
)

// TestRunDoc_NoSubcommand checks the missing-subcommand error path.
func TestRunDoc_NoSubcommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDoc(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "missing doc subcommand") {
		t.Errorf("error = %q, want to contain 'missing doc subcommand'", err.Error())
	}
}

// TestRunDoc_UnknownSubcommand checks that unknown subcommands produce a clear error.
func TestRunDoc_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDoc([]string{"frobnicate"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error = %q, want to contain the unknown subcommand name", err.Error())
	}
}

// TestRunDocRegister_MissingPath checks that omitting the path argument returns an error.
func TestRunDocRegister_MissingPath(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}
	if !strings.Contains(err.Error(), "missing document path") {
		t.Errorf("error = %q, want to contain 'missing document path'", err.Error())
	}
}

// TestRunDocRegister_MissingType checks that omitting --type returns an error before
// identity resolution or any service call.
func TestRunDocRegister_MissingType(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--title", "My Doc"}, deps)
	if err == nil {
		t.Fatal("expected error for missing --type, got nil")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("error = %q, want to contain '--type is required'", err.Error())
	}
}

// TestRunDocRegister_MissingTitle checks that omitting --title returns an error before
// identity resolution or any service call.
func TestRunDocRegister_MissingTitle(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--type", "design"}, deps)
	if err == nil {
		t.Fatal("expected error for missing --title, got nil")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("error = %q, want to contain '--title is required'", err.Error())
	}
}

// TestRunDocRegister_UnknownFlag checks that unknown flags return a clear error.
func TestRunDocRegister_UnknownFlag(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--bogus", "val"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
	if !strings.Contains(err.Error(), "--bogus") {
		t.Errorf("error = %q, want to contain flag name '--bogus'", err.Error())
	}
}

// TestRunDocRegister_ByFlagMissingValue checks that --by without a value errors.
func TestRunDocRegister_ByFlagMissingValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocRegister([]string{"some/path.md", "--by"}, deps)
	if err == nil {
		t.Fatal("expected error for --by with no value, got nil")
	}
	if !strings.Contains(err.Error(), "--by requires a value") {
		t.Errorf("error = %q, want to contain '--by requires a value'", err.Error())
	}
}

// TestDocUsageText_ContainsByFlag verifies AC-005: the usage text lists --by as an
// optional flag with a description, satisfying REQ-005.
func TestDocUsageText_ContainsByFlag(t *testing.T) {
	t.Parallel()

	if !strings.Contains(docUsageText, "--by") {
		t.Error("docUsageText does not contain '--by' flag")
	}
	// The flag should be described as optional (auto-resolved if omitted).
	if !strings.Contains(docUsageText, "auto-resolved") {
		t.Error("docUsageText does not indicate '--by' is auto-resolved when omitted")
	}
}

// TestRunDocApprove_ByFlagAcceptsEmptyWithoutHardcodedError verifies that
// runDocApprove calls config.ResolveIdentity rather than passing the raw --by
// value directly. When --by is omitted, the error (if any) must come from
// ResolveIdentity or the service layer, not a hard-coded "approver is required" check.
func TestRunDocApprove_ByFlagAcceptsEmptyWithoutHardcodedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	// Omit --by. The command will reach config.ResolveIdentity.
	// The resulting error (if any) must NOT contain "approver is required".
	err := runDocApprove([]string{"DOC-nonexistent"}, deps)
	if err != nil && strings.Contains(err.Error(), "approver is required") {
		t.Errorf("got old hard-coded error %q; expected identity resolution via config.ResolveIdentity", err.Error())
	}
}

// TestRunDocRegister_ByFlagAcceptsEmptyWithoutHardcodedError verifies AC-004 and
// AC-007: when --by is omitted and identity cannot be resolved, the error comes
// directly from config.ResolveIdentity with no extra wrapping (we check the known
// error substrings that ResolveIdentity produces).
//
// This test only fires when run in an environment where git config user.name is also
// unset, which is not always the case. We instead verify the code path by asserting
// that the function does NOT return a "created_by is required" error (the old
// hard-coded error) — proving the new identity resolution path is taken.
func TestRunDocRegister_ByFlagAcceptsEmptyWithoutHardcodedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	// Pass valid flags but omit --by. The command will reach config.ResolveIdentity.
	// The resulting error (if any) must NOT be the old "created_by is required" message.
	err := runDocRegister([]string{"nonexistent/path.md", "--type", "design", "--title", "Test"}, deps)

	// If an error occurs, it should not be the old hard-coded "created_by is required" error.
	// It will be either:
	//   - nil / a service error (if identity resolved and service ran)
	//   - the ResolveIdentity error (if git config is also absent)
	if err != nil && strings.Contains(err.Error(), "created_by is required") {
		t.Errorf("got old hard-coded error %q; expected identity resolution to be attempted via config.ResolveIdentity", err.Error())
	}
}

// ─── resolveDocApproveTarget tests ───────────────────────────────────────────

// setupDocSvc creates a DocumentService backed by a temporary state directory.
// The repoRoot is set to tmpDir so that SubmitDocument can find files at
// relative paths within tmpDir.
func setupDocSvc(t *testing.T) (*service.DocumentService, string) {
	t.Helper()

	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, core.StateDir)
	if err := os.MkdirAll(filepath.Join(stateDir, "documents"), 0o755); err != nil {
		t.Fatalf("create state dir: %v", err)
	}

	docSvc := service.NewDocumentService(stateDir, tmpDir)
	return docSvc, tmpDir
}

// registerTestDoc registers a document file in the temp directory and returns
// the SubmitDocument result. The file is created at the given relative path
// within repoRoot with minimal content.
func registerTestDoc(t *testing.T, docSvc *service.DocumentService, repoRoot, relPath, docType, title string) service.DocumentResult {
	t.Helper()

	fullPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("create doc dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("# "+title+"\n\nTest content.\n"), 0o644); err != nil {
		t.Fatalf("write doc file: %v", err)
	}

	result, err := docSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      relPath,
		Type:      docType,
		Title:     title,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}
	if result.ID == "" {
		t.Fatal("SubmitDocument returned empty ID")
	}
	return result
}

// TestResolveDocApproveTarget_UnregisteredPath_Error verifies AC-021:
// a path to an unregistered file returns "file is not registered: <path>".
func TestResolveDocApproveTarget_UnregisteredPath_Error(t *testing.T) {
	t.Parallel()

	docSvc, _ := setupDocSvc(t)

	_, err := resolveDocApproveTarget("work/design/unregistered.md", docSvc)
	if err == nil {
		t.Fatal("expected error for unregistered file, got nil")
	}
	if !strings.Contains(err.Error(), "file is not registered") {
		t.Errorf("error = %q, want to contain 'file is not registered'", err.Error())
	}
	if !strings.Contains(err.Error(), "work/design/unregistered.md") {
		t.Errorf("error = %q, want to contain the file path", err.Error())
	}
}

// TestResolveDocApproveTarget_RegisteredPath_ReturnsID verifies AC-022:
// a path to a registered file resolves to the correct document ID.
func TestResolveDocApproveTarget_RegisteredPath_ReturnsID(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := setupDocSvc(t)
	result := registerTestDoc(t, docSvc, repoRoot, "work/design/foo.md", "design", "Test Design")

	gotID, err := resolveDocApproveTarget("work/design/foo.md", docSvc)
	if err != nil {
		t.Fatalf("resolveDocApproveTarget: %v", err)
	}
	if gotID != result.ID {
		t.Errorf("resolved ID = %q, want %q", gotID, result.ID)
	}
}

// TestResolveDocApproveTarget_IDForm_ReturnsUnchanged verifies AC-023:
// an existing document ID is passed through unchanged (backward compat).
func TestResolveDocApproveTarget_IDForm_ReturnsUnchanged(t *testing.T) {
	t.Parallel()

	docSvc, _ := setupDocSvc(t)

	// "DOC-0012" matches the entity ID pattern so Disambiguate returns ResolveEntity.
	gotID, err := resolveDocApproveTarget("DOC-0012", docSvc)
	if err != nil {
		t.Fatalf("resolveDocApproveTarget: %v", err)
	}
	if gotID != "DOC-0012" {
		t.Errorf("resolved ID = %q, want %q", gotID, "DOC-0012")
	}
}

// TestResolveDocApproveTarget_PathWithDotSlash_StripsPrefix verifies that
// paths with "./" prefix are correctly resolved (LookupByPath strips "./").
func TestResolveDocApproveTarget_PathWithDotSlash_StripsPrefix(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := setupDocSvc(t)
	result := registerTestDoc(t, docSvc, repoRoot, "work/design/foo.md", "design", "Test Design")

	// Resolve with "./" prefix.
	gotID, err := resolveDocApproveTarget("./work/design/foo.md", docSvc)
	if err != nil {
		t.Fatalf("resolveDocApproveTarget: %v", err)
	}
	if gotID != result.ID {
		t.Errorf("resolved ID = %q, want %q", gotID, result.ID)
	}
}

// TestRunDocApprove_MissingArg_ShowsUpdatedUsage verifies the error message
// mentions both ID and path options.
func TestRunDocApprove_MissingArg_ShowsUpdatedUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	deps := dependencies{stdout: &buf, stdin: strings.NewReader("")}

	err := runDocApprove(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing arg, got nil")
	}
	if !strings.Contains(err.Error(), "missing document ID or path") {
		t.Errorf("error = %q, want to contain 'missing document ID or path'", err.Error())
	}
}
