package git

import (
	"os/exec"
	"testing"
)

// TestCheckKbzDirty_Clean_ReturnsNil verifies that a clean repo returns (nil, nil).
func TestCheckKbzDirty_Clean_ReturnsNil(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no dirty files, got %v", files)
	}
}

// TestCheckKbzDirty_DirtyState_ReturnsFiles verifies that an untracked file
// under .kbz/state/ is reported.
func TestCheckKbzDirty_DirtyState_ReturnsFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-001.yaml", "id: TASK-001\nstatus: active\n")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected dirty files, got none")
	}
	found := false
	for _, f := range files {
		if f == ".kbz/state/tasks/TASK-001.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected .kbz/state/tasks/TASK-001.yaml in dirty list, got %v", files)
	}
}

// TestCheckKbzDirty_DirtyIndex_ReturnsFiles verifies that an untracked file
// under .kbz/index/ is reported.
func TestCheckKbzDirty_DirtyIndex_ReturnsFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/index/documents/DOC-001.yaml", "id: DOC-001\n")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected dirty files, got none")
	}
	found := false
	for _, f := range files {
		if f == ".kbz/index/documents/DOC-001.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected .kbz/index/documents/DOC-001.yaml in dirty list, got %v", files)
	}
}

// TestCheckKbzDirty_DirtyContext_ReturnsFiles verifies that an untracked file
// under .kbz/context/ is reported.
func TestCheckKbzDirty_DirtyContext_ReturnsFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/context/profiles/default.yaml", "role: default\n")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected dirty files, got none")
	}
	found := false
	for _, f := range files {
		if f == ".kbz/context/profiles/default.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected .kbz/context/profiles/default.yaml in dirty list, got %v", files)
	}
}

// TestCheckKbzDirty_OutsidePathsIgnored verifies that dirty files outside the
// three .kbz/ subdirectories do not appear in the result.
func TestCheckKbzDirty_OutsidePathsIgnored(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, "work/spec/some-spec.md", "# Spec\n")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no dirty files from outside paths, got %v", files)
	}
}

// TestCheckKbzDirty_AllThreeDirs verifies that dirty files in all three
// directories are returned together.
func TestCheckKbzDirty_AllThreeDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-002.yaml", "id: TASK-002\n")
	writeFile(t, dir, ".kbz/index/documents/DOC-002.yaml", "id: DOC-002\n")
	writeFile(t, dir, ".kbz/context/profiles/agent.yaml", "role: agent\n")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	if len(files) < 3 {
		t.Errorf("expected at least 3 dirty files, got %v", files)
	}
}

// TestCheckKbzDirty_CommittedFile_NotReported verifies that a previously
// committed file that is now clean does not appear in the result.
func TestCheckKbzDirty_CommittedFile_NotReported(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initTestRepo(t, dir)

	writeFile(t, dir, ".kbz/state/tasks/TASK-003.yaml", "id: TASK-003\n")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", ".kbz/state/tasks/TASK-003.yaml")
	run("commit", "-m", "chore: add task file")

	files, err := CheckKbzDirty(dir)
	if err != nil {
		t.Fatalf("CheckKbzDirty: unexpected error: %v", err)
	}
	for _, f := range files {
		if f == ".kbz/state/tasks/TASK-003.yaml" {
			t.Errorf("committed file %q should not appear as dirty", f)
		}
	}
}

// TestCheckKbzDirty_NonExistentRepo_ReturnsError verifies that a non-existent
// repo path returns an error.
func TestCheckKbzDirty_NonExistentRepo_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := CheckKbzDirty("/tmp/kanbanzai-nonexistent-dirty-test-xyz")
	if err == nil {
		t.Error("expected error for non-existent repo path, got nil")
	}
}
