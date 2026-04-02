// Package git commit_test.go — unit tests for CommitStateIfDirty.
//
// These tests exercise the git utility function used by the handoff tool's
// pre-dispatch state commit logic (sub-agent-state-isolation spec).
//
// Each test creates a real git repository in a temporary directory so that
// git commands execute against a real .git object store. The tests do not
// depend on the caller's working directory or repository state.
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo initialises a minimal git repository in dir, sets a local
// user identity so commits succeed in CI environments, and creates an initial
// empty commit so HEAD exists.
func initTestRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@kanbanzai.test")
	run("config", "user.name", "Kanbanzai Test")
	// Create an initial empty commit so HEAD and the index exist.
	run("commit", "--allow-empty", "-m", "chore: initial empty commit")
}

// gitLogMessages returns the commit messages from git log in the repo.
func gitLogMessages(t *testing.T, dir string) []string {
	t.Helper()
	cmd := exec.Command("git", "log", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var messages []string
	for _, l := range lines {
		if l != "" {
			messages = append(messages, l)
		}
	}
	return messages
}

// gitShowFiles returns the list of files changed in the most recent commit.
func gitShowFiles(t *testing.T, dir string) []string {
	t.Helper()
	cmd := exec.Command("git", "show", "--name-only", "--format=", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, l := range lines {
		if l != "" {
			files = append(files, l)
		}
	}
	return files
}

// writeFile creates a file at relPath within dir (creating parent dirs as needed).
func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// countCommits returns the number of commits reachable from HEAD.
func countCommits(t *testing.T, dir string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list: %v", err)
	}
	n := 0
	for _, b := range strings.TrimSpace(string(out)) {
		if b >= '0' && b <= '9' {
			n = n*10 + int(b-'0')
		}
	}
	return n
}

// AC-10: When .kbz/state/ has no uncommitted changes, no commit is created.
func TestCommitStateIfDirty_Clean_NothingCommitted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	before := countCommits(t, dir)

	committed, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("CommitStateIfDirty: unexpected error: %v", err)
	}
	if committed {
		t.Error("committed = true, want false (nothing in .kbz/state/ to commit)")
	}

	after := countCommits(t, dir)
	if after != before {
		t.Errorf("commit count changed: %d → %d; no commit should be created when state is clean", before, after)
	}
}

// AC-07 + AC-09: When .kbz/state/ has uncommitted changes, a commit is created
// with the exact required message.
func TestCommitStateIfDirty_DirtyState_CommitCreated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	// Write a file under .kbz/state/ without committing it.
	writeFile(t, dir, ".kbz/state/tasks/TASK-001.yaml", "id: TASK-001\nstatus: active\n")

	before := countCommits(t, dir)

	committed, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("CommitStateIfDirty: unexpected error: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true (dirty .kbz/state/ should produce a commit)")
	}

	after := countCommits(t, dir)
	if after != before+1 {
		t.Errorf("commit count: %d → %d, want +1", before, after)
	}

	// AC-09: commit message must be exactly the required string.
	messages := gitLogMessages(t, dir)
	if len(messages) == 0 {
		t.Fatal("no commits found after CommitStateIfDirty")
	}
	wantMsg := "chore(kbz): persist workflow state before sub-agent dispatch"
	if messages[0] != wantMsg {
		t.Errorf("commit message = %q, want %q", messages[0], wantMsg)
	}
}

// AC-08: The commit includes only files under .kbz/state/; unrelated working
// tree changes are not staged or committed.
func TestCommitStateIfDirty_OnlyStagedStateFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	// Write a state file (should be committed).
	writeFile(t, dir, ".kbz/state/features/FEAT-001.yaml", "id: FEAT-001\nstatus: developing\n")

	// Write an unrelated file in a different directory (must NOT be committed).
	writeFile(t, dir, "work/spec/some-spec.md", "# Some Spec\n")

	committed, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("CommitStateIfDirty: unexpected error: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true (state file should be committed)")
	}

	// Inspect the committed files.
	committedFiles := gitShowFiles(t, dir)
	for _, f := range committedFiles {
		if !strings.HasPrefix(f, ".kbz/state/") {
			t.Errorf("committed file %q is outside .kbz/state/; only state files should be committed", f)
		}
	}

	// The unrelated file must NOT appear in the commit.
	for _, f := range committedFiles {
		if strings.Contains(f, "work/spec") {
			t.Errorf("committed file %q should not be in the commit (outside .kbz/state/)", f)
		}
	}

	// Verify the unrelated file is still untracked (not staged).
	// Use --untracked-files=all to force individual file listing rather than
	// directory-level summary (which would show "?? work/" instead of the file).
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	cmd.Dir = dir
	out, _ := cmd.Output()
	statusStr := string(out)
	if !strings.Contains(statusStr, "work/spec/some-spec.md") {
		t.Errorf("unrelated file work/spec/some-spec.md should still appear as untracked after commit\ngit status output:\n%s", statusStr)
	}
}

// A second call with no new changes must not create another commit.
func TestCommitStateIfDirty_SecondCallWithCleanState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-001.yaml", "id: TASK-001\n")

	// First call — should commit.
	committed1, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("first CommitStateIfDirty: %v", err)
	}
	if !committed1 {
		t.Error("first call: committed = false, want true")
	}

	// Second call — state is already committed; should not create another commit.
	before := countCommits(t, dir)
	committed2, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("second CommitStateIfDirty: %v", err)
	}
	if committed2 {
		t.Error("second call: committed = true, want false (nothing new to commit)")
	}
	after := countCommits(t, dir)
	if after != before {
		t.Errorf("commit count changed on second call: %d → %d", before, after)
	}
}

// CommitStateIfDirty with a non-existent directory must return an error,
// not panic.
func TestCommitStateIfDirty_NonExistentRepo_ReturnsError(t *testing.T) {
	t.Parallel()
	// Pass a path that does not exist as a git repo.
	_, err := CommitStateIfDirty("/tmp/kanbanzai-nonexistent-test-repo-xyz")
	if err == nil {
		t.Error("expected error for non-existent repo path, got nil")
	}
}

// ─── CommitStateWithMessage tests ────────────────────────────────────────────

// AC-A01: CommitStateWithMessage creates a commit with the supplied message.
func TestCommitStateWithMessage_CustomMessage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-001.yaml", "id: TASK-001\nstatus: done\n")

	committed, err := CommitStateWithMessage(dir, "workflow(TASK-001): complete – do the thing")
	if err != nil {
		t.Fatalf("CommitStateWithMessage: unexpected error: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true")
	}

	messages := gitLogMessages(t, dir)
	if len(messages) == 0 {
		t.Fatal("no commits found")
	}
	wantMsg := "workflow(TASK-001): complete – do the thing"
	if messages[0] != wantMsg {
		t.Errorf("commit message = %q, want %q", messages[0], wantMsg)
	}
}

// AC-A02: CommitStateWithMessage returns (false, nil) when .kbz/state/ is clean.
func TestCommitStateWithMessage_Clean_NothingCommitted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	before := countCommits(t, dir)

	committed, err := CommitStateWithMessage(dir, "workflow: some message")
	if err != nil {
		t.Fatalf("CommitStateWithMessage: unexpected error: %v", err)
	}
	if committed {
		t.Error("committed = true, want false (nothing in .kbz/state/ to commit)")
	}

	after := countCommits(t, dir)
	if after != before {
		t.Errorf("commit count changed: %d → %d", before, after)
	}
}

// CommitStateIfDirty must continue to use the fixed message after the refactor (regression).
func TestCommitStateIfDirty_StillUsesFixedMessage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-002.yaml", "id: TASK-002\nstatus: active\n")

	committed, err := CommitStateIfDirty(dir)
	if err != nil {
		t.Fatalf("CommitStateIfDirty: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true")
	}

	messages := gitLogMessages(t, dir)
	wantMsg := "chore(kbz): persist workflow state before sub-agent dispatch"
	if len(messages) == 0 || messages[0] != wantMsg {
		t.Errorf("commit message = %q, want %q", messages[0], wantMsg)
	}
}

// ─── CommitStateAndPaths tests ────────────────────────────────────────────────

// AC-A03: CommitStateAndPaths creates a single commit containing both
// .kbz/state/ files and the extra paths.
func TestCommitStateAndPaths_StagesStateAndExtraPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/documents/DOC-001.yaml", "id: DOC-001\n")
	writeFile(t, dir, "work/spec/my-spec.md", "# My Spec\n")

	committed, err := CommitStateAndPaths(dir, "workflow(DOC-001): register specification", "work/spec/my-spec.md")
	if err != nil {
		t.Fatalf("CommitStateAndPaths: unexpected error: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true")
	}

	committedFiles := gitShowFiles(t, dir)
	hasState := false
	hasExtra := false
	for _, f := range committedFiles {
		if strings.HasPrefix(f, ".kbz/state/") {
			hasState = true
		}
		if f == "work/spec/my-spec.md" {
			hasExtra = true
		}
	}
	if !hasState {
		t.Error("commit does not contain any .kbz/state/ file")
	}
	if !hasExtra {
		t.Error("commit does not contain work/spec/my-spec.md")
	}

	// Verify it was a single commit (count increased by exactly 1).
	if countCommits(t, dir) != 2 {
		t.Errorf("expected 2 total commits, got %d", countCommits(t, dir))
	}
}

// AC-A04: CommitStateAndPaths does not stage files outside .kbz/state/ and
// the explicit extraPaths.
func TestCommitStateAndPaths_DoesNotStageOtherFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/documents/DOC-002.yaml", "id: DOC-002\n")
	writeFile(t, dir, "work/spec/my-spec.md", "# My Spec\n")
	writeFile(t, dir, "work/design/other.md", "# Other\n") // NOT in extraPaths

	committed, err := CommitStateAndPaths(dir, "workflow(DOC-002): register specification", "work/spec/my-spec.md")
	if err != nil {
		t.Fatalf("CommitStateAndPaths: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true")
	}

	committedFiles := gitShowFiles(t, dir)
	for _, f := range committedFiles {
		if f == "work/design/other.md" {
			t.Errorf("file %q should not have been committed (not in extraPaths)", f)
		}
	}
}

// CommitStateAndPaths with no dirty files returns (false, nil).
func TestCommitStateAndPaths_Clean_NothingCommitted(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	before := countCommits(t, dir)

	committed, err := CommitStateAndPaths(dir, "workflow: test", "work/spec/nonexistent.md")
	if err != nil {
		t.Fatalf("CommitStateAndPaths: unexpected error: %v", err)
	}
	if committed {
		t.Error("committed = true, want false (nothing dirty)")
	}

	after := countCommits(t, dir)
	if after != before {
		t.Errorf("commit count changed: %d → %d", before, after)
	}
}

// CommitStateAndPaths with only an extra path dirty (no state changes) commits
// just the extra path.
func TestCommitStateAndPaths_OnlyExtraPathDirty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	// Only the extra path is dirty; .kbz/state/ is clean.
	writeFile(t, dir, "work/spec/my-spec.md", "# New Spec\n")

	committed, err := CommitStateAndPaths(dir, "workflow(DOC-003): register specification", "work/spec/my-spec.md")
	if err != nil {
		t.Fatalf("CommitStateAndPaths: %v", err)
	}
	if !committed {
		t.Error("committed = false, want true (extra path is dirty)")
	}

	committedFiles := gitShowFiles(t, dir)
	hasExtra := false
	for _, f := range committedFiles {
		if f == "work/spec/my-spec.md" {
			hasExtra = true
		}
	}
	if !hasExtra {
		t.Error("commit does not contain work/spec/my-spec.md")
	}
}
