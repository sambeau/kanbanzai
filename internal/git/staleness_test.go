package git

import (
	"testing"
	"time"
)

func TestCheckStaleness_NoAnchors(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create at least one commit
	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Initial")

	info, err := CheckStaleness(repo, nil, time.Now())
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if info.IsStale {
		t.Error("CheckStaleness() with no anchors should not be stale")
	}
	if len(info.StaleFiles) > 0 {
		t.Errorf("CheckStaleness() StaleFiles = %v, want empty", info.StaleFiles)
	}
}

func TestCheckStaleness_EmptyAnchors(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.txt", "content")
	commitFile(t, repo, "file.txt", "Initial")

	info, err := CheckStaleness(repo, []GitAnchor{}, time.Now())
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if info.IsStale {
		t.Error("CheckStaleness() with empty anchors should not be stale")
	}
}

func TestCheckStaleness_FileModifiedAfterConfirm(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create and commit initial file
	createFile(t, repo, "api/handler.go", "package api\n\n// v1")
	commitFile(t, repo, "api/handler.go", "Initial handler")

	// Record time after first commit, subtract 1 second to ensure "before" comparison works
	_, firstModified, err := GetFileLastModified(repo, "api/handler.go")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}
	confirmedAt := firstModified.Add(-time.Second)

	// Sleep to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	createFile(t, repo, "api/handler.go", "package api\n\n// v2")
	commitFile(t, repo, "api/handler.go", "Update handler")

	anchors := []GitAnchor{{Path: "api/handler.go"}}

	info, err := CheckStaleness(repo, anchors, confirmedAt)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckStaleness() IsStale = false, want true")
	}
	if len(info.StaleFiles) != 1 {
		t.Fatalf("CheckStaleness() StaleFiles len = %d, want 1", len(info.StaleFiles))
	}
	if info.StaleFiles[0].Path != "api/handler.go" {
		t.Errorf("CheckStaleness() StaleFiles[0].Path = %q, want %q", info.StaleFiles[0].Path, "api/handler.go")
	}
	if info.StaleFiles[0].Commit == "" {
		t.Error("CheckStaleness() StaleFiles[0].Commit should not be empty")
	}
	if info.StaleReason == "" {
		t.Error("CheckStaleness() StaleReason should not be empty")
	}
}

func TestCheckStaleness_FileNotModifiedAfterConfirm(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "api/handler.go", "package api")
	commitFile(t, repo, "api/handler.go", "Initial handler")

	// Use a future time as last_confirmed
	future := time.Now().Add(time.Hour)

	anchors := []GitAnchor{{Path: "api/handler.go"}}

	info, err := CheckStaleness(repo, anchors, future)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if info.IsStale {
		t.Error("CheckStaleness() IsStale = true, want false")
	}
	if len(info.StaleFiles) > 0 {
		t.Errorf("CheckStaleness() StaleFiles = %v, want empty", info.StaleFiles)
	}
}

func TestCheckStaleness_MultipleAnchors_OneStale(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create two files
	createFile(t, repo, "file1.go", "package main")
	createFile(t, repo, "file2.go", "package main")
	commitFile(t, repo, ".", "Initial commit")

	// Get the modification time - use this as confirmedAt
	// file2 will have the same modifiedAt, so it won't be "after" confirmedAt
	_, confirmedAt, err := GetFileLastModified(repo, "file1.go")
	if err != nil {
		t.Fatalf("GetFileLastModified() error = %v", err)
	}

	// Sleep to ensure the next commit has a different timestamp
	time.Sleep(1100 * time.Millisecond)

	// Only modify file1 - this will give file1 a new modifiedAt > confirmedAt
	createFile(t, repo, "file1.go", "package main // modified")
	commitFile(t, repo, "file1.go", "Update file1")

	anchors := []GitAnchor{
		{Path: "file1.go"},
		{Path: "file2.go"},
	}

	info, err := CheckStaleness(repo, anchors, confirmedAt)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckStaleness() IsStale = false, want true")
	}
	if len(info.StaleFiles) != 1 {
		t.Fatalf("CheckStaleness() StaleFiles len = %d, want 1", len(info.StaleFiles))
	}
	if info.StaleFiles[0].Path != "file1.go" {
		t.Errorf("CheckStaleness() stale file = %q, want file1.go", info.StaleFiles[0].Path)
	}
}

func TestCheckStaleness_MultipleAnchors_AllStale(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create two files
	createFile(t, repo, "file1.go", "v1")
	createFile(t, repo, "file2.go", "v1")
	commitFile(t, repo, ".", "Initial commit")

	// Use a time before the commit
	past := time.Now().Add(-time.Hour)

	anchors := []GitAnchor{
		{Path: "file1.go"},
		{Path: "file2.go"},
	}

	info, err := CheckStaleness(repo, anchors, past)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckStaleness() IsStale = false, want true")
	}
	if len(info.StaleFiles) != 2 {
		t.Errorf("CheckStaleness() StaleFiles len = %d, want 2", len(info.StaleFiles))
	}
}

func TestCheckStaleness_ZeroLastConfirmed(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.go", "content")
	commitFile(t, repo, "file.go", "Initial")

	anchors := []GitAnchor{{Path: "file.go"}}

	// Zero time means "never confirmed"
	info, err := CheckStaleness(repo, anchors, time.Time{})
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckStaleness() with zero lastConfirmed should be stale")
	}
	if info.StaleReason == "" {
		t.Error("CheckStaleness() StaleReason should indicate never confirmed")
	}
}

func TestCheckStaleness_AnchorFileNotFound(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Create a different file so repo has commits
	createFile(t, repo, "other.go", "content")
	commitFile(t, repo, "other.go", "Initial")

	anchors := []GitAnchor{{Path: "nonexistent.go"}}

	info, err := CheckStaleness(repo, anchors, time.Now())
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	// Missing file should be treated as stale (the knowledge references something that doesn't exist)
	if !info.IsStale {
		t.Error("CheckStaleness() with missing anchor file should be stale")
	}
	if len(info.StaleFiles) != 1 {
		t.Fatalf("CheckStaleness() StaleFiles len = %d, want 1", len(info.StaleFiles))
	}
	if info.StaleFiles[0].Path != "nonexistent.go" {
		t.Errorf("CheckStaleness() stale file path = %q, want nonexistent.go", info.StaleFiles[0].Path)
	}
	if info.StaleFiles[0].Commit != "" {
		t.Errorf("CheckStaleness() missing file should have empty commit, got %q", info.StaleFiles[0].Commit)
	}
}

func TestCheckStaleness_MixedFoundAndMissing(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "exists.go", "content")
	commitFile(t, repo, "exists.go", "Initial")

	// Use past time so existing file is stale
	past := time.Now().Add(-time.Hour)

	anchors := []GitAnchor{
		{Path: "exists.go"},
		{Path: "missing.go"},
	}

	info, err := CheckStaleness(repo, anchors, past)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.IsStale {
		t.Error("CheckStaleness() should be stale")
	}
	if len(info.StaleFiles) != 2 {
		t.Fatalf("CheckStaleness() StaleFiles len = %d, want 2", len(info.StaleFiles))
	}
}

func TestCheckStaleness_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // Not a git repo

	anchors := []GitAnchor{{Path: "file.go"}}

	_, err := CheckStaleness(dir, anchors, time.Now())
	if err == nil {
		t.Error("CheckStaleness() expected error for non-repo")
	}
}

func TestCheckStaleness_LastConfirmedPreserved(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	createFile(t, repo, "file.go", "content")
	commitFile(t, repo, "file.go", "Initial")

	lastConfirmed := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	anchors := []GitAnchor{{Path: "file.go"}}

	info, err := CheckStaleness(repo, anchors, lastConfirmed)
	if err != nil {
		t.Fatalf("CheckStaleness() error = %v", err)
	}

	if !info.LastConfirmed.Equal(lastConfirmed) {
		t.Errorf("CheckStaleness() LastConfirmed = %v, want %v", info.LastConfirmed, lastConfirmed)
	}
}

func TestBuildStaleReason_SingleModified(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "file.go", ModifiedAt: time.Now(), Commit: "abc123"},
	}

	reason := buildStaleReason(files, time.Now().Add(-time.Hour))

	if reason != "Anchored file modified" {
		t.Errorf("buildStaleReason() = %q, want %q", reason, "Anchored file modified")
	}
}

func TestBuildStaleReason_MultipleModified(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "file1.go", ModifiedAt: time.Now(), Commit: "abc123"},
		{Path: "file2.go", ModifiedAt: time.Now(), Commit: "def456"},
	}

	reason := buildStaleReason(files, time.Now().Add(-time.Hour))

	expected := "Anchored files modified: 2 files"
	if reason != expected {
		t.Errorf("buildStaleReason() = %q, want %q", reason, expected)
	}
}

func TestBuildStaleReason_SingleMissing(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "missing.go", ModifiedAt: time.Time{}, Commit: ""},
	}

	reason := buildStaleReason(files, time.Now())

	if reason != "Anchored file not found: missing.go" {
		t.Errorf("buildStaleReason() = %q", reason)
	}
}

func TestBuildStaleReason_MultipleMissing(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "missing1.go", ModifiedAt: time.Time{}, Commit: ""},
		{Path: "missing2.go", ModifiedAt: time.Time{}, Commit: ""},
	}

	reason := buildStaleReason(files, time.Now())

	expected := "Anchored files not found: 2 files"
	if reason != expected {
		t.Errorf("buildStaleReason() = %q, want %q", reason, expected)
	}
}

func TestBuildStaleReason_Mixed(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "modified.go", ModifiedAt: time.Now(), Commit: "abc123"},
		{Path: "missing.go", ModifiedAt: time.Time{}, Commit: ""},
	}

	reason := buildStaleReason(files, time.Now().Add(-time.Hour))

	expected := "Anchored files changed: 1 modified, 1 not found"
	if reason != expected {
		t.Errorf("buildStaleReason() = %q, want %q", reason, expected)
	}
}

func TestBuildStaleReason_NeverConfirmed(t *testing.T) {
	t.Parallel()

	files := []StaleFile{
		{Path: "file.go", ModifiedAt: time.Now(), Commit: "abc123"},
	}

	// Zero time means never confirmed
	reason := buildStaleReason(files, time.Time{})

	if reason != "Anchored file modified (entry never confirmed)" {
		t.Errorf("buildStaleReason() = %q", reason)
	}
}

func TestBuildStaleReason_Empty(t *testing.T) {
	t.Parallel()

	reason := buildStaleReason(nil, time.Now())

	if reason != "" {
		t.Errorf("buildStaleReason(nil) = %q, want empty", reason)
	}
}
