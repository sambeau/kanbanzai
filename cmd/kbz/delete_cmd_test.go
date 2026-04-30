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

// setupTestGitRepo creates a temp dir with a git repo and changes CWD via t.Chdir.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitEnv := append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gitEnv
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "test")
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}

// commitFile adds a file to git and commits it.
func commitFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, relPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, relPath), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	for _, args := range [][]string{
		{"git", "add", relPath},
		{"git", "commit", "--no-gpg-sign", "-m", "add " + relPath},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
}

// writeDocRecord writes a minimal YAML document record to .kbz/state/documents/.
func writeDocRecord(t *testing.T, dir, id, docPath, status string) {
	t.Helper()
	stateDir := filepath.Join(dir, ".kbz", "state", "documents")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filename := strings.ReplaceAll(id, "/", "--") + ".yaml"
	recordPath := filepath.Join(stateDir, filename)
	content := fmt.Sprintf(
		"id: %s\npath: %s\ntype: design\ntitle: Test Doc\nstatus: %s\ncontent_hash: abc\ncreated: \"2024-01-01T00:00:00Z\"\ncreated_by: test\nupdated: \"2024-01-01T00:00:00Z\"\n",
		id, docPath, status,
	)
	if err := os.WriteFile(recordPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeTestDeps(stdout *bytes.Buffer, stdinContent string) dependencies {
	return dependencies{
		stdout: stdout,
		stdin:  strings.NewReader(stdinContent),
	}
}

// TestDeleteRejectNonWorkPath covers AC-003 and AC-004.
func TestDeleteRejectNonWorkPath(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, path string }{
		{"internal_pkg", "internal/service/documents.go"},
		{"dot_kbz", ".kbz/state/documents/foo.yaml"},
		{"docs_dir", "docs/getting-started.md"},
		{"bare_file", "file.md"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := runDelete([]string{tc.path}, makeTestDeps(&buf, ""))
			if err == nil {
				t.Fatal("expected error for non-work/ path, got nil")
			}
			if !strings.Contains(err.Error(), "work/") {
				t.Errorf("error %q does not mention work/", err.Error())
			}
		})
	}
}

// TestDeleteMissingFile covers AC-002.
func TestDeleteMissingFile(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := runDelete([]string{"work/this-xyz987-does-not-exist.md"}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not mention not found", err.Error())
	}
}

// TestDeleteNoOsRemove covers AC-010: verifies git rm is used, not os.Remove.
func TestDeleteNoOsRemove(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("delete_cmd.go")
	if err != nil {
		t.Fatalf("reading delete_cmd.go: %v", err)
	}
	if strings.Contains(string(data), "os.Remove") {
		t.Error("delete_cmd.go contains os.Remove: deletion must go through git rm")
	}
}

// TestDeleteCLIDispatchWired covers AC-016.
func TestDeleteCLIDispatchWired(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := run([]string{"delete"}, makeTestDeps(&buf, ""))
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		t.Errorf("delete not dispatched; got unknown command error: %v", err)
	}
}

// TestDeleteProceedsOnValidPath covers AC-001.
func TestDeleteProceedsOnValidPath(t *testing.T) {
	dir := setupTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "work", "valid.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := runDelete([]string{"work/valid.md"}, makeTestDeps(&buf, "n\n"))
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "work/") {
			t.Errorf("got early-exit error %q; expected to reach prompt", err.Error())
		}
	}
	if !strings.Contains(buf.String(), "work/valid.md") {
		t.Errorf("stdout %q does not reference file path", buf.String())
	}
}

// TestDeleteApprovedWithoutForce covers AC-005.
func TestDeleteApprovedWithoutForce(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/approved.md"
	if err := os.WriteFile(filepath.Join(dir, fp), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeDocRecord(t, dir, "FEAT-T/design-approved", fp, "approved")
	var buf bytes.Buffer
	err := runDelete([]string{fp}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error for approved doc without --force")
	}
	if !strings.Contains(err.Error(), "approved") {
		t.Errorf("error %q does not mention approved", err.Error())
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error %q does not mention --force", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, fp)); statErr != nil {
		t.Error("file was deleted despite guard")
	}
}

// TestDeleteConfirmationPromptShown covers AC-006.
func TestDeleteConfirmationPromptShown(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/prompt.md"
	if err := os.WriteFile(filepath.Join(dir, fp), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_ = runDelete([]string{fp}, makeTestDeps(&buf, "n\n"))
	out := buf.String()
	if !strings.Contains(out, "Delete work/prompt.md") {
		t.Errorf("stdout %q missing prompt with file path", out)
	}
	if !strings.Contains(out, "[y/N]") {
		t.Errorf("stdout %q missing [y/N] prompt", out)
	}
}

// TestDeleteAbortOnN covers AC-007.
func TestDeleteAbortOnN(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/abort.md"
	if err := os.WriteFile(filepath.Join(dir, fp), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := runDelete([]string{fp}, makeTestDeps(&buf, "n\n"))
	if err != nil {
		t.Fatalf("expected nil error on n-abort, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, fp)); statErr != nil {
		t.Error("file was deleted after n-abort")
	}
	if !strings.Contains(buf.String(), "Aborted") {
		t.Errorf("stdout %q missing abort message", buf.String())
	}
}

// TestDeleteProceedOnY covers AC-008.
func TestDeleteProceedOnY(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/proceed.md"
	commitFile(t, dir, fp, "to delete")
	var buf bytes.Buffer
	err := runDelete([]string{fp}, makeTestDeps(&buf, "y\n"))
	if err != nil {
		t.Fatalf("expected nil error on y, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, fp)); statErr == nil {
		t.Error("file still exists after y-confirm")
	}
}

// TestDeleteForceSkipsPromptAndGuard covers AC-009.
func TestDeleteForceSkipsPromptAndGuard(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/force.md"
	commitFile(t, dir, fp, "approved content")
	writeDocRecord(t, dir, "FEAT-T/design-force", fp, "approved")
	var buf bytes.Buffer
	err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("expected nil error with --force, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, fp)); statErr == nil {
		t.Error("file still exists after --force")
	}
	if strings.Contains(buf.String(), "[y/N]") {
		t.Error("--force should not show confirmation prompt")
	}
}

// TestDeleteGitRmFailureAborts covers AC-011.
func TestDeleteGitRmFailureAborts(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/untracked.md"
	if err := os.WriteFile(filepath.Join(dir, fp), []byte("untracked"), 0o644); err != nil {
		t.Fatal(err)
	}
	const docID = "FEAT-T/design-untracked"
	writeDocRecord(t, dir, docID, fp, "draft")
	var buf bytes.Buffer
	err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, ""))
	if err == nil {
		t.Fatal("expected error when git rm fails on untracked file, got nil")
	}
	if !strings.Contains(err.Error(), "git rm failed") {
		t.Errorf("error %q does not mention git rm failure", err.Error())
	}
	recordFile := filepath.Join(dir, ".kbz", "state", "documents", strings.ReplaceAll(docID, "/", "--")+".yaml")
	if _, statErr := os.Stat(recordFile); statErr != nil {
		t.Error("document record was deleted despite git rm failure")
	}
}

// TestDeleteRecordRemovedAfterGitRm covers AC-012.
func TestDeleteRecordRemovedAfterGitRm(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/cleanup.md"
	commitFile(t, dir, fp, "remove me")
	const docID = "FEAT-T/design-cleanup"
	writeDocRecord(t, dir, docID, fp, "draft")
	var buf bytes.Buffer
	err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	recordFile := filepath.Join(dir, ".kbz", "state", "documents", strings.ReplaceAll(docID, "/", "--")+".yaml")
	if _, statErr := os.Stat(recordFile); statErr == nil {
		t.Error("document record still exists after successful deletion")
	}
}

// TestDeleteEntityRefClearing_CodeReviewVerified covers AC-013 via code inspection.
// Entity-ref clearing runs inside DeleteDocument via entityHook; this is tested in
// internal/service/documents_test.go. Here we verify the CLI calls DeleteDocument.
func TestDeleteEntityRefClearing_CodeReviewVerified(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("delete_cmd.go")
	if err != nil {
		t.Fatalf("reading delete_cmd.go: %v", err)
	}
	if !strings.Contains(string(data), "DeleteDocument") {
		t.Error("delete_cmd.go does not call DeleteDocument")
	}
	if !strings.Contains(string(data), "Force: true") {
		t.Error("delete_cmd.go does not pass Force: true to DeleteDocument")
	}
}

// TestDeleteSuccessOutput covers AC-014.
func TestDeleteSuccessOutput(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/success.md"
	commitFile(t, dir, fp, "content")
	const docID = "FEAT-T/design-success"
	writeDocRecord(t, dir, docID, fp, "draft")
	var buf bytes.Buffer
	err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "Deleted ") {
		t.Errorf("output %q does not start with Deleted ", out)
	}
	if !strings.Contains(out, fp) {
		t.Errorf("output %q does not contain file path", out)
	}
	if !strings.Contains(out, "document record") {
		t.Errorf("output %q does not mention document record", out)
	}
	if !strings.Contains(out, "removed") {
		t.Errorf("output %q does not contain removed", out)
	}
	if n := len(strings.Split(out, "\n")); n != 1 {
		t.Errorf("expected 1 output line, got %d: %q", n, out)
	}
}

// TestDeleteNoRecordWarning covers AC-015.
func TestDeleteNoRecordWarning(t *testing.T) {
	dir := setupTestGitRepo(t)
	const fp = "work/no-record.md"
	commitFile(t, dir, fp, "content")
	var buf bytes.Buffer
	err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, ""))
	if err != nil {
		t.Fatalf("expected nil error for no-record case, got: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No document record found") {
		t.Errorf("output %q missing no-record warning", out)
	}
	if !strings.Contains(out, "file deleted") {
		t.Errorf("output %q missing file deleted message", out)
	}
}

// TestDeleteExitCodes covers AC-017.
func TestDeleteExitCodes(t *testing.T) {
	t.Run("non_work_nonzero", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := runDelete([]string{"docs/readme.md"}, makeTestDeps(&buf, "")); err == nil {
			t.Error("expected non-zero exit for non-work/ path")
		}
	})
	t.Run("missing_file_nonzero", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		if err := runDelete([]string{"work/no-exist-abc.md"}, makeTestDeps(&buf, "")); err == nil {
			t.Error("expected non-zero exit for missing file")
		}
	})
	t.Run("user_abort_zero", func(t *testing.T) {
		dir := setupTestGitRepo(t)
		const fp = "work/abort-exit.md"
		_ = os.WriteFile(filepath.Join(dir, fp), []byte("x"), 0o644)
		var buf bytes.Buffer
		if err := runDelete([]string{fp}, makeTestDeps(&buf, "n\n")); err != nil {
			t.Errorf("expected zero exit on n-abort, got: %v", err)
		}
	})
	t.Run("approved_no_force_nonzero", func(t *testing.T) {
		dir := setupTestGitRepo(t)
		const fp = "work/appr.md"
		_ = os.WriteFile(filepath.Join(dir, fp), []byte("x"), 0o644)
		writeDocRecord(t, dir, "FEAT-X/design-appr", fp, "approved")
		var buf bytes.Buffer
		if err := runDelete([]string{fp}, makeTestDeps(&buf, "")); err == nil {
			t.Error("expected non-zero exit for approved doc without --force")
		}
	})
	t.Run("success_zero", func(t *testing.T) {
		dir := setupTestGitRepo(t)
		const fp = "work/success-exit.md"
		commitFile(t, dir, fp, "content")
		var buf bytes.Buffer
		if err := runDelete([]string{"--force", fp}, makeTestDeps(&buf, "")); err != nil {
			t.Errorf("expected zero exit on success, got: %v", err)
		}
	})
}
