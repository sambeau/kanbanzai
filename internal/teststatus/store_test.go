package teststatus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testDir creates a temporary directory and returns its path, the directory
// itself (for cleanup), and a cleanup function.
func testDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "teststatus-*")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

func TestReadRecord_FileNotFound(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	rec, err := ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: unexpected error: %v", err)
	}
	if rec.Result != ResultUnknown {
		t.Errorf("expected Result=%q, got %q", ResultUnknown, rec.Result)
	}
	if rec.LastRun != nil {
		t.Errorf("expected nil LastRun, got %v", rec.LastRun)
	}
}

func TestRoundTrip(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	now := time.Now().Truncate(time.Second) // YAML has second precision
	orig := Record{
		LastRun: &now,
		Result:  ResultPass,
		Summary: "All tests passed",
		Runner:  "agent",
		Trigger: "post-merge",
	}

	if err := WriteRecord(dir, orig); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	got, err := ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}

	if got.Result != orig.Result {
		t.Errorf("Result: got %q, want %q", got.Result, orig.Result)
	}
	if got.Summary != orig.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, orig.Summary)
	}
	if got.Runner != orig.Runner {
		t.Errorf("Runner: got %q, want %q", got.Runner, orig.Runner)
	}
	if got.Trigger != orig.Trigger {
		t.Errorf("Trigger: got %q, want %q", got.Trigger, orig.Trigger)
	}
	if got.LastRun == nil {
		t.Fatal("LastRun is nil, want non-nil")
	}
	if !got.LastRun.Equal(now) {
		t.Errorf("LastRun: got %v, want %v", got.LastRun, now)
	}
}

func TestRoundTrip_WithFailures(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	now := time.Now().Truncate(time.Second)
	orig := Record{
		LastRun: &now,
		Result:  ResultFail,
		Summary: "2 tests failed",
		Failures: []Failure{
			{Package: "./internal/foo", Test: "TestBar", Message: "expected 42, got 0"},
			{Package: "./internal/baz", Test: "TestQux", Message: "timeout"},
		},
	}

	if err := WriteRecord(dir, orig); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	got, err := ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}

	if len(got.Failures) != len(orig.Failures) {
		t.Fatalf("len(Failures): got %d, want %d", len(got.Failures), len(orig.Failures))
	}
	for i, f := range got.Failures {
		if f != orig.Failures[i] {
			t.Errorf("Failures[%d]: got %+v, want %+v", i, f, orig.Failures[i])
		}
	}
}

func TestIsStale_NilLastRun(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	rec := Record{Result: ResultUnknown}
	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if !stale {
		t.Errorf("expected stale=true for nil LastRun")
	}
}

func TestIsStale_AfterTouchingGoFile(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	// Create a .go file with an old timestamp.
	now := time.Now()
	old := now.Add(-1 * time.Hour)
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(goFile, old, old); err != nil {
		t.Fatal(err)
	}

	rec := Record{LastRun: &now}

	// Should not be stale: old .go file.
	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false before touch")
	}

	// Now touch the file.
	newTime := now.Add(1 * time.Minute)
	if err := os.Chtimes(goFile, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	stale, err = IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale after touch: %v", err)
	}
	if !stale {
		t.Errorf("expected stale=true after touch")
	}
}

func TestIsStale_NoGoFilesChanged(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	// Create a .go file and set its mtime to before last run.
	now := time.Now()
	old := now.Add(-1 * time.Hour)
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(goFile, old, old); err != nil {
		t.Fatal(err)
	}

	rec := Record{LastRun: &now}
	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false when no .go files changed")
	}
}

func TestIsStale_ExcludesHiddenDirs(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	// Create a hidden directory with a .go file.
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	hiddenGo := filepath.Join(hiddenDir, "secret.go")
	if err := os.WriteFile(hiddenGo, []byte("package secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	rec := Record{LastRun: &now}

	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false when .go file is in hidden directory")
	}
}

func TestIsStale_ExcludesVendor(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	vendorDir := filepath.Join(dir, "vendor", "example.com", "foo")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	vendorGo := filepath.Join(vendorDir, "foo.go")
	if err := os.WriteFile(vendorGo, []byte("package foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	rec := Record{LastRun: &now}

	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false when .go file is in vendor/")
	}
}

func TestIsStale_ExcludesWorktrees(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	wtDir := filepath.Join(dir, ".worktrees", "FEAT-xxxx")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wtGo := filepath.Join(wtDir, "work.go")
	if err := os.WriteFile(wtGo, []byte("package work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	rec := Record{LastRun: &now}

	stale, err := IsStale(dir, rec)
	if err != nil {
		t.Fatalf("IsStale: %v", err)
	}
	if stale {
		t.Errorf("expected stale=false when .go file is in .worktrees/")
	}
}

func TestWriteRecord_CreatesDir(t *testing.T) {
	dir, cleanup := testDir(t)
	t.Cleanup(cleanup)

	// Remove the dir to ensure WriteRecord creates it.
	os.RemoveAll(dir)

	now := time.Now().Truncate(time.Second)
	rec := Record{
		LastRun: &now,
		Result:  ResultPass,
	}

	if err := WriteRecord(dir, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	// Verify the file exists and is readable.
	got, err := ReadRecord(dir)
	if err != nil {
		t.Fatalf("ReadRecord after WriteRecord: %v", err)
	}
	if got.Result != ResultPass {
		t.Errorf("Result: got %q, want %q", got.Result, ResultPass)
	}
}
