package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitStaged commits all staged changes in the repo. Used to make git log
// --follow work after a git mv (which only stages the rename).
func commitStaged(t *testing.T, dir string) {
	t.Helper()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	cmd := exec.Command("git", "commit", "--no-gpg-sign", "-m", "commit staged changes")
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commitStaged: git commit: %v\n%s", err, out)
	}
}

// ── Test helpers ─────────────────────────────────────────────────────────────

// writePlanState writes a minimal plan YAML at .kbz/state/plans/{planID}.yaml.
// planID must be in the form "{Prefix}{n}-{slug}", e.g. "P1-source-plan".
func writePlanState(t *testing.T, dir, planID string) {
	t.Helper()
	// Derive slug: everything after the first hyphen.
	idx := strings.Index(planID, "-")
	if idx < 0 {
		t.Fatalf("writePlanState: invalid planID %q (no hyphen)", planID)
	}
	slug := planID[idx+1:]

	stateDir := filepath.Join(dir, ".kbz", "state", "plans")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(
		"id: %s\nslug: %s\nname: Test Plan\nstatus: active\nsummary: Test plan\ncreated: \"2024-01-01T00:00:00Z\"\ncreated_by: test\nupdated: \"2024-01-01T00:00:00Z\"\nnext_feature_seq: 1\n",
		planID, slug,
	)
	if err := os.WriteFile(filepath.Join(stateDir, planID+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeFeatureState writes a minimal feature YAML at
// .kbz/state/features/{featureID}-{slug}.yaml.
func writeFeatureState(t *testing.T, dir, featureID, slug, displayID, parentPlanID string) {
	t.Helper()
	stateDir := filepath.Join(dir, ".kbz", "state", "features")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf(
		"id: %s\nslug: %s\nname: Test Feature\nparent: %s\nstatus: developing\nsummary: A test feature\ncreated: \"2024-01-01T00:00:00Z\"\ncreated_by: test\nupdated: \"2024-01-01T00:00:00Z\"\ndisplay_id: %s\n",
		featureID, slug, parentPlanID, displayID,
	)
	filename := featureID + "-" + slug + ".yaml"
	if err := os.WriteFile(filepath.Join(stateDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeDocRecordOwned writes a document YAML record that includes an owner field.
func writeDocRecordOwned(t *testing.T, dir, id, docPath, status, owner string) {
	t.Helper()
	stateDir := filepath.Join(dir, ".kbz", "state", "documents")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filename := strings.ReplaceAll(id, "/", "--") + ".yaml"
	content := fmt.Sprintf(
		"id: %s\npath: %s\ntype: design\ntitle: Test Doc\nstatus: %s\nowner: %s\ncontent_hash: abc\ncreated: \"2024-01-01T00:00:00Z\"\ncreated_by: test\nupdated: \"2024-01-01T00:00:00Z\"\n",
		id, docPath, status, owner,
	)
	if err := os.WriteFile(filepath.Join(stateDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── Mode 1 unit tests (parallel, no git repo required) ────────────────────────

// TestModeDetection_SlashInPath covers AC-001:
// a path containing "/" is recognised as a file path (Mode 1) and does NOT
// produce the "does not look like a file path" error.
func TestModeDetection_SlashInPath(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runMove([]string{"work/foo/bar.md", "P37"}, makeTestDeps(&buf, ""))
	if err != nil && strings.Contains(err.Error(), "does not look like a file path") {
		t.Errorf("got mode-detection error %q; slash in path should trigger Mode 1", err.Error())
	}
}

// TestModeDetection_DotMdExtension covers AC-002:
// a bare ".md" filename (no slash) still triggers Mode 1 detection, which then
// fails with "not within work/" rather than "does not look like a file path".
func TestModeDetection_DotMdExtension(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runMove([]string{"spec.md", "P37"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for non-work/ path, got nil")
	}
	if strings.Contains(err.Error(), "does not look like a file path") {
		t.Errorf("got mode-detection error %q; .md extension should trigger Mode 1", err.Error())
	}
	if !strings.Contains(err.Error(), "not within work/") {
		t.Errorf("error %q does not mention 'not within work/'", err.Error())
	}
}

// TestRejectOutsideWorkDir covers AC-003:
// a Mode 1 path that does not begin with "work/" is rejected.
func TestRejectOutsideWorkDir(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runMove([]string{"docs/foo.md", "P37"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for non-work/ path, got nil")
	}
	if !strings.Contains(err.Error(), "not within work/") {
		t.Errorf("error %q does not mention 'not within work/'", err.Error())
	}
}

// TestModeDetection_FeatureDisplayID covers AC-011:
// "P37-F1" is recognised as a Mode 2 arg and does NOT produce the
// "does not look like a file path" error.
func TestModeDetection_FeatureDisplayID(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runMove([]string{"P37-F1", "P25"}, makeTestDeps(&buf, ""))
	// Must go to Mode 2, so the error (if any) must NOT be the mode-detection sentinel.
	if err != nil && strings.Contains(err.Error(), "does not look like a file path") {
		t.Errorf("got mode-detection error %q; P37-F1 should trigger Mode 2", err.Error())
	}
}

// ── Mode 1 integration tests ──────────────────────────────────────────────────

// TestFileMoveSourceNotFound covers AC-004:
// a missing source file returns an error containing "not found".
func TestFileMoveSourceNotFound(t *testing.T) {
	setupTestGitRepo(t)
	var buf bytes.Buffer
	err := runMove([]string{"work/nonexistent-xyz987.md", "P37"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for missing source file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

// TestFileMoveTargetPlanNotFound covers AC-005:
// when the target plan does not exist, the error contains "not found".
func TestFileMoveTargetPlanNotFound(t *testing.T) {
	dir := setupTestGitRepo(t)
	// Create the plans directory (empty) so resolvePlanArg can scan it and
	// return "plan not found" rather than a directory-read error.
	if err := os.MkdirAll(filepath.Join(dir, ".kbz", "state", "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work", "foo.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := runMove([]string{"work/foo.md", "P99"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for unknown plan, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

// TestFileMoveRegisteredDocument covers AC-006 (happy path):
// a committed, registered document is moved via git mv; the source disappears,
// the target appears, git history is preserved, and the record is updated.
func TestFileMoveRegisteredDocument(t *testing.T) {
	dir := setupTestGitRepo(t)
	writePlanState(t, dir, "P1-test-plan")
	commitFile(t, dir, "work/P1-design-foo.md", "original content")
	writeDocRecord(t, dir, "FEAT-001/design-foo", "work/P1-design-foo.md", "draft")

	var buf bytes.Buffer
	if err := runMove([]string{"work/P1-design-foo.md", "P1"}, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMove() error = %v", err)
	}

	// Source must be gone.
	if _, err := os.Stat(filepath.Join(dir, "work", "P1-design-foo.md")); err == nil {
		t.Error("source file still exists after move")
	}

	// Target must exist at canonical path.
	dstPath := "work/P1-test-plan/P1-design-foo.md"
	if _, err := os.Stat(filepath.Join(dir, dstPath)); err != nil {
		t.Errorf("target file %q not found after move: %v", dstPath, err)
	}

	// Commit the staged rename so git log --follow can trace the history.
	commitStaged(t, dir)

	// git log --follow must show at least one commit (proves git mv was used).
	logCmd := exec.Command("git", "log", "--follow", "--oneline", dstPath)
	logCmd.Dir = dir
	out, cmdErr := logCmd.CombinedOutput()
	if cmdErr != nil {
		t.Fatalf("git log --follow: %v\n%s", cmdErr, out)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Error("git log --follow returned no commits — git mv was not used")
	}

	// Output must mention the move.
	if !strings.Contains(buf.String(), dstPath) {
		t.Errorf("stdout %q does not mention destination path", buf.String())
	}
}

// TestFileMoveCreatesPlanFolder covers AC-007:
// when the target plan folder does not yet exist it is created automatically.
func TestFileMoveCreatesPlanFolder(t *testing.T) {
	dir := setupTestGitRepo(t)
	writePlanState(t, dir, "P2-fresh-plan")
	commitFile(t, dir, "work/P2-design-doc.md", "content")

	// Confirm the folder does not pre-exist.
	targetFolder := filepath.Join(dir, "work", "P2-fresh-plan")
	if _, err := os.Stat(targetFolder); err == nil {
		t.Fatal("target folder already exists — test setup error")
	}

	var buf bytes.Buffer
	if err := runMove([]string{"work/P2-design-doc.md", "P2"}, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMove() error = %v", err)
	}

	if _, err := os.Stat(targetFolder); err != nil {
		t.Errorf("target folder %q was not created: %v", targetFolder, err)
	}
}

// TestFileMoveTargetAlreadyExists covers AC-008:
// if the computed target path already exists, the move is refused.
func TestFileMoveTargetAlreadyExists(t *testing.T) {
	dir := setupTestGitRepo(t)
	writePlanState(t, dir, "P1-test-plan")
	commitFile(t, dir, "work/P1-design-foo.md", "source content")

	// Pre-create the exact path that runMove would compute as the destination.
	if err := os.MkdirAll(filepath.Join(dir, "work", "P1-test-plan"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "work", "P1-test-plan", "P1-design-foo.md"),
		[]byte("already here"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := runMove([]string{"work/P1-design-foo.md", "P1"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error when target already exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q does not mention 'already exists'", err.Error())
	}
}

// TestFileMoveUnregisteredFile covers AC-009:
// a file with no document record is still moved; stdout warns about the missing record.
func TestFileMoveUnregisteredFile(t *testing.T) {
	dir := setupTestGitRepo(t)
	writePlanState(t, dir, "P1-test-plan")
	commitFile(t, dir, "work/P1-design-bar.md", "content")

	var buf bytes.Buffer
	if err := runMove([]string{"work/P1-design-bar.md", "P1"}, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMove() error = %v", err)
	}
	if !strings.Contains(buf.String(), "No document record found") {
		t.Errorf("stdout %q missing 'No document record found' warning", buf.String())
	}
}

// TestFileMoveNoOsRename covers AC-010 via code inspection:
// move_cmd.go must not use os.Rename — the file move must go through git mv.
func TestFileMoveNoOsRename(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("move_cmd.go")
	if err != nil {
		t.Fatalf("reading move_cmd.go: %v", err)
	}
	if strings.Contains(string(data), "os.Rename") {
		t.Error("move_cmd.go contains os.Rename: file move must go through git mv")
	}
}

// ── Mode 2 integration tests ──────────────────────────────────────────────────

// TestReParentFeatureNotFound covers AC-012:
// an unknown feature display ID returns an error containing "not found".
func TestReParentFeatureNotFound(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")

	var buf bytes.Buffer
	err := runMoveFeature("P99-F1", "P1", false, stateRoot, dir, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for unknown display ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain 'not found'", err.Error())
	}
}

// TestReParentSamePlan covers AC-013:
// re-parenting to the plan the feature already belongs to is rejected.
func TestReParentSamePlan(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	var buf bytes.Buffer
	err := runMoveFeature("P1-F1", "P1", false, stateRoot, dir, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error when feature already belongs to target plan, got nil")
	}
	if !strings.Contains(err.Error(), "already") && !strings.Contains(err.Error(), "nothing to do") {
		t.Errorf("error %q does not mention 'already' or 'nothing to do'", err.Error())
	}
}

// TestReParentConfirmationPrompt covers AC-014:
// without --force, the user is shown a "[y/N]" prompt; answering "n" aborts
// without error.
func TestReParentConfirmationPrompt(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writePlanState(t, dir, "P2-target-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	var buf bytes.Buffer
	err := runMoveFeature("P1-F1", "P2", false, stateRoot, dir, makeTestDeps(&buf, "n\n"))
	if err != nil {
		t.Fatalf("expected nil error on 'n' abort, got: %v", err)
	}
	if !strings.Contains(buf.String(), "[y/N]") {
		t.Errorf("stdout %q missing '[y/N]' prompt", buf.String())
	}
}

// TestReParentForceFlag covers AC-015:
// passing force=true skips the confirmation prompt entirely.
func TestReParentForceFlag(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writePlanState(t, dir, "P2-target-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	var buf bytes.Buffer
	err := runMoveFeature("P1-F1", "P2", true, stateRoot, dir, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("runMoveFeature() with force=true error = %v", err)
	}
	if strings.Contains(buf.String(), "[y/N]") {
		t.Error("force=true should not show confirmation prompt, but '[y/N]' was found in output")
	}
}

// TestReParentEntityUpdate covers AC-016:
// after a successful re-parent the feature's parent field is updated on disk.
func TestReParentEntityUpdate(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writePlanState(t, dir, "P2-target-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	var buf bytes.Buffer
	if err := runMoveFeature("P1-F1", "P2", true, stateRoot, dir, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMoveFeature() error = %v", err)
	}

	featFile := filepath.Join(dir, ".kbz", "state", "features", "FEAT-001-my-feature.yaml")
	data, err := os.ReadFile(featFile)
	if err != nil {
		t.Fatalf("read feature YAML after re-parent: %v", err)
	}
	if !strings.Contains(string(data), "P2-target-plan") {
		t.Errorf("feature YAML does not contain new parent 'P2-target-plan':\n%s", string(data))
	}
	if !strings.Contains(string(data), "display_id: P2-F1") {
		t.Errorf("feature YAML does not contain new display_id 'P2-F1':\n%s", string(data))
	}

	// Plan next_feature_seq must have been incremented (was 1, now 2).
	planFile := filepath.Join(dir, ".kbz", "state", "plans", "P2-target-plan.yaml")
	planData, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("read plan YAML after re-parent: %v", err)
	}
	if !strings.Contains(string(planData), "next_feature_seq: 2") {
		t.Errorf("plan YAML does not contain incremented 'next_feature_seq: 2':\n%s", string(planData))
	}
}

// TestReParentDocumentMoves covers AC-017:
// documents owned by the feature are moved via git mv so that
// git log --follow shows the original commit history at the new path.
func TestReParentDocumentMoves(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writePlanState(t, dir, "P2-target-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	// Commit the source document so git mv can track it.
	commitFile(t, dir, "work/P1-design-foo.md", "document content")
	writeDocRecordOwned(t, dir, "FEAT-001/design-foo", "work/P1-design-foo.md", "draft", "FEAT-001")

	var buf bytes.Buffer
	if err := runMoveFeature("P1-F1", "P2", true, stateRoot, dir, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMoveFeature() error = %v", err)
	}

	// Source file must be gone.
	if _, err := os.Stat(filepath.Join(dir, "work", "P1-design-foo.md")); err == nil {
		t.Error("source document still exists after re-parent")
	}

	// Target file must exist at the canonical new path.
	newPath := "work/P2-target-plan/P2-design-foo.md"
	if _, err := os.Stat(filepath.Join(dir, newPath)); err != nil {
		t.Errorf("target document %q not found after re-parent: %v", newPath, err)
	}

	// Commit the staged rename so git log --follow can trace the history.
	commitStaged(t, dir)

	// git log --follow must show at least one commit, proving git mv was used.
	logCmd := exec.Command("git", "log", "--follow", "--oneline", newPath)
	logCmd.Dir = dir
	out, cmdErr := logCmd.CombinedOutput()
	if cmdErr != nil {
		t.Fatalf("git log --follow %s: %v\n%s", newPath, cmdErr, out)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Error("git log --follow returned no commits — git mv was not used")
	}

	// AC-017: Document record owner must be updated to the target plan ID,
	// not the feature's canonical ID.
	docStateDir := filepath.Join(dir, ".kbz", "state", "documents")
	docFile := filepath.Join(docStateDir, "FEAT-001--design-foo.yaml")
	docData, err := os.ReadFile(docFile)
	if err != nil {
		t.Fatalf("read document record after re-parent: %v", err)
	}
	if !strings.Contains(string(docData), "owner: P2-target-plan") {
		t.Errorf("document record does not contain owner 'P2-target-plan':\n%s", string(docData))
	}
}

// TestReParentOutputSummary covers AC-018:
// successful re-parent output contains both "Moved feature" and the old doc path.
func TestReParentOutputSummary(t *testing.T) {
	dir := setupTestGitRepo(t)
	stateRoot := filepath.Join(dir, ".kbz", "state")
	writePlanState(t, dir, "P1-source-plan")
	writePlanState(t, dir, "P2-target-plan")
	writeFeatureState(t, dir, "FEAT-001", "my-feature", "P1-F1", "P1-source-plan")

	commitFile(t, dir, "work/P1-design-foo.md", "document content")
	writeDocRecordOwned(t, dir, "FEAT-001/design-foo", "work/P1-design-foo.md", "draft", "FEAT-001")

	var buf bytes.Buffer
	if err := runMoveFeature("P1-F1", "P2", true, stateRoot, dir, makeTestDeps(&buf, "")); err != nil {
		t.Fatalf("runMoveFeature() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved feature") {
		t.Errorf("output %q missing 'Moved feature' line", out)
	}
	if !strings.Contains(out, "work/P1-design-foo.md") {
		t.Errorf("output %q missing original document path", out)
	}
}
