package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// setupTestRepo creates a temp directory with an initialized Git repo.
// Returns the repo path. Use t.Cleanup to ensure cleanup.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	return dir
}

// runGit runs a git command in the specified directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}

// createFile creates a file with content and returns its path.
func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

// commitFile stages and commits a file with the given message.
func commitFile(t *testing.T, dir, file, message string) {
	t.Helper()
	runGit(t, dir, "add", file)
	runGit(t, dir, "commit", "-m", message)
}

func TestGetFileLastModified_SingleCommit(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit a file
	createFile(t, repo, "README.md", "# Test")
	commitFile(t, repo, "README.md", "Initial commit")

	beforeTest := time.Now().Add(-time.Minute)

	commit, modifiedAt, err := GetFileLastModified(repo, "README.md")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	if commit == "" {
		t.Error("GetFileLastModified() commit is empty")
	}
	if len(commit) < 7 {
		t.Errorf("GetFileLastModified() commit = %q, expected full SHA", commit)
	}

	if modifiedAt.Before(beforeTest) {
		t.Errorf("GetFileLastModified() modifiedAt = %v, expected after %v", modifiedAt, beforeTest)
	}
	if modifiedAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("GetFileLastModified() modifiedAt = %v, expected before now", modifiedAt)
	}
}

func TestGetFileLastModified_MultipleCommits(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit initial file
	createFile(t, repo, "file.txt", "version 1")
	commitFile(t, repo, "file.txt", "First commit")

	// Sleep to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Modify and commit again
	createFile(t, repo, "file.txt", "version 2")
	commitFile(t, repo, "file.txt", "Second commit")

	commit1, _, err := GetFileLastModified(repo, "file.txt")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	// Sleep and modify again
	time.Sleep(100 * time.Millisecond)
	createFile(t, repo, "file.txt", "version 3")
	commitFile(t, repo, "file.txt", "Third commit")

	commit2, _, err := GetFileLastModified(repo, "file.txt")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	// The commits should be different
	if commit1 == commit2 {
		t.Error("GetFileLastModified() should return different commits after modification")
	}
}

func TestGetFileLastModified_NestedFile(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create nested directory structure
	createFile(t, repo, "internal/api/handler.go", "package api")
	commitFile(t, repo, "internal/api/handler.go", "Add handler")

	commit, _, err := GetFileLastModified(repo, "internal/api/handler.go")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	if commit == "" {
		t.Error("GetFileLastModified() commit is empty for nested file")
	}
}

func TestGetFileLastModified_FileNotFound(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create at least one commit so the repo is not empty
	createFile(t, repo, "other.txt", "content")
	commitFile(t, repo, "other.txt", "Initial")

	_, _, err := GetFileLastModified(repo, "nonexistent.txt")
	if err != ErrFileNotFound {
		t.Errorf("GetFileLastModified() error = %v, want ErrFileNotFound", err)
	}
}

func TestGetFileLastModified_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // Not a git repo

	_, _, err := GetFileLastModified(dir, "file.txt")
	if err != ErrNotARepository {
		t.Errorf("GetFileLastModified() error = %v, want ErrNotARepository", err)
	}
}

func TestGetCommitTimestamp_ValidCommit(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit a file
	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	beforeTest := time.Now().Add(-time.Minute)

	// Get the commit SHA
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse: %v", err)
	}
	commitSHA := string(out[:len(out)-1]) // trim newline

	timestamp, err := GetCommitTimestamp(repo, commitSHA)
	if err != nil {
		t.Fatalf("GetCommitTimestamp() error = %v", err)
	}

	if timestamp.Before(beforeTest) {
		t.Errorf("GetCommitTimestamp() = %v, expected after %v", timestamp, beforeTest)
	}
	if timestamp.After(time.Now().Add(time.Minute)) {
		t.Errorf("GetCommitTimestamp() = %v, expected before now", timestamp)
	}
}

func TestGetCommitTimestamp_ShortSHA(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	// Get the short commit SHA
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse: %v", err)
	}
	shortSHA := string(out[:len(out)-1]) // trim newline

	_, err = GetCommitTimestamp(repo, shortSHA)
	if err != nil {
		t.Errorf("GetCommitTimestamp() with short SHA error = %v", err)
	}
}

func TestGetCommitTimestamp_BranchName(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	// Get timestamp using branch name
	_, err := GetCommitTimestamp(repo, "HEAD")
	if err != nil {
		t.Errorf("GetCommitTimestamp() with HEAD error = %v", err)
	}
}

func TestGetCommitTimestamp_InvalidCommit(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	_, err := GetCommitTimestamp(repo, "0000000000000000000000000000000000000000")
	if err == nil {
		t.Error("GetCommitTimestamp() expected error for invalid commit")
	}
}

func TestGetCommitTimestamp_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // Not a git repo

	_, err := GetCommitTimestamp(dir, "HEAD")
	if err != ErrNotARepository {
		t.Errorf("GetCommitTimestamp() error = %v, want ErrNotARepository", err)
	}
}

func TestIsFileModifiedSince_ModifiedAfter(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create a reference time in the past
	past := time.Now().Add(-time.Hour)

	// Create and commit a file (will be newer than 'past')
	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	modified, err := IsFileModifiedSince(repo, "file.txt", past)
	if err != nil {
		t.Fatalf("IsFileModifiedSince() error = %v", err)
	}

	if !modified {
		t.Error("IsFileModifiedSince() = false, expected true (file was modified after 'past')")
	}
}

func TestIsFileModifiedSince_NotModifiedAfter(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit a file
	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	// Use a future time
	future := time.Now().Add(time.Hour)

	modified, err := IsFileModifiedSince(repo, "file.txt", future)
	if err != nil {
		t.Fatalf("IsFileModifiedSince() error = %v", err)
	}

	if modified {
		t.Error("IsFileModifiedSince() = true, expected false (file was not modified after 'future')")
	}
}

func TestIsFileModifiedSince_FileNotFound(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "other.txt", "content")
	commitFile(t, repo, "other.txt", "Initial")

	_, err := IsFileModifiedSince(repo, "nonexistent.txt", time.Now())
	if err != ErrFileNotFound {
		t.Errorf("IsFileModifiedSince() error = %v, want ErrFileNotFound", err)
	}
}

func TestIsFileModifiedSince_ExactTimestamp(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Test commit")

	// Get the exact modification time
	_, modifiedAt, err := GetFileLastModified(repo, "file.txt")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	// File modified at exactly 'since' should return false (not "after")
	modified, err := IsFileModifiedSince(repo, "file.txt", modifiedAt)
	if err != nil {
		t.Fatalf("IsFileModifiedSince() error = %v", err)
	}

	if modified {
		t.Error("IsFileModifiedSince() = true, expected false (file was not modified *after* exact time)")
	}
}
