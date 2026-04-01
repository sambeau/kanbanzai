package actionlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanup_DeletesOldFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a file that is 31 days old.
	oldDate := time.Now().UTC().AddDate(0, 0, -31).Format("2006-01-02")
	oldFile := filepath.Join(dir, "actions-"+oldDate+".jsonl")
	if err := os.WriteFile(oldFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a file that is 1 day old (should be kept).
	recentDate := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	recentFile := filepath.Join(dir, "actions-"+recentDate+".jsonl")
	if err := os.WriteFile(recentFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(dir, time.Now().UTC()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old file should have been deleted")
	}
	if _, err := os.Stat(recentFile); err != nil {
		t.Errorf("recent file should still exist: %v", err)
	}
}

func TestCleanup_MissingDirIsNotError(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nonexistent")
	if err := Cleanup(dir, time.Now().UTC()); err != nil {
		t.Errorf("Cleanup with missing dir: %v, want nil", err)
	}
}

func TestCleanup_IgnoresNonLogFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create an unrelated file.
	other := filepath.Join(dir, "something.txt")
	if err := os.WriteFile(other, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(dir, time.Now().UTC()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := os.Stat(other); err != nil {
		t.Errorf("non-log file should not be deleted: %v", err)
	}
}

func TestCleanup_KeepsFilesExactlyAtCutoff(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)

	// Exactly at the 30-day cutoff — should be kept.
	cutoffDate := now.AddDate(0, 0, -30).Format("2006-01-02")
	cutoffFile := filepath.Join(dir, "actions-"+cutoffDate+".jsonl")
	if err := os.WriteFile(cutoffFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Cleanup(dir, now); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if _, err := os.Stat(cutoffFile); err != nil {
		t.Errorf("file at cutoff boundary should be kept: %v", err)
	}
}
