package health

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckTestSuite_EmptyPath(t *testing.T) {
	summary := CheckTestSuite("")
	if summary.TotalPackages != 0 {
		t.Errorf("TotalPackages: got %d, want 0", summary.TotalPackages)
	}
	if summary.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestCheckTestSuite_NoGoModDir(t *testing.T) {
	dir := t.TempDir()
	summary := CheckTestSuite(dir)
	if summary.TotalPackages != 0 {
		t.Errorf("TotalPackages: got %d, want 0", summary.TotalPackages)
	}
	if summary.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestCheckTestSuite_MockedPassing(t *testing.T) {
	origFn := checkTestSuiteFn
	t.Cleanup(func() { checkTestSuiteFn = origFn })

	checkTestSuiteFn = func(repoPath string, now time.Time) TestSuiteSummary {
		return TestSuiteSummary{
			TotalPackages:   10,
			PassingPackages: 10,
			FailingPackages: 0,
			Failures:        nil,
			HasFailure:      false,
			CheckedAt:       now.Format(time.RFC3339),
		}
	}

	summary := CheckTestSuite(".")
	if summary.TotalPackages != 10 {
		t.Errorf("TotalPackages: got %d, want 10", summary.TotalPackages)
	}
	if summary.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestCheckTestSuite_MockedFailing(t *testing.T) {
	origFn := checkTestSuiteFn
	t.Cleanup(func() { checkTestSuiteFn = origFn })

	checkTestSuiteFn = func(repoPath string, now time.Time) TestSuiteSummary {
		return TestSuiteSummary{
			TotalPackages:   10,
			PassingPackages: 8,
			FailingPackages: 2,
			Failures: []Failure{
				{Package: "./internal/foo", TestName: "TestFoo"},
				{Package: "./internal/bar", TestName: "TestBar"},
			},
			HasFailure: true,
			CheckedAt:  now.Format(time.RFC3339),
		}
	}

	summary := CheckTestSuite(".")
	if !summary.HasFailure {
		t.Error("HasFailure: got false, want true")
	}
	if len(summary.Failures) != 2 {
		t.Errorf("len(Failures): got %d, want 2", len(summary.Failures))
	}
	if summary.PassingPackages != 8 {
		t.Errorf("PassingPackages: got %d, want 8", summary.PassingPackages)
	}
}

func TestCheckTestSuite_CacheHit(t *testing.T) {
	dir := t.TempDir()

	// Create go.mod so test suite considers this a Go project.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a fresh cache.
	now := time.Now()
	cached := TestSuiteSummary{
		TotalPackages:   5,
		PassingPackages: 5,
		FailingPackages: 0,
		Failures:        nil,
		HasFailure:      false,
		CheckedAt:       now.Format(time.RFC3339),
	}
	cacheDir := filepath.Join(dir, ".kbz", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(filepath.Join(cacheDir, "test-suite-cache.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	summary := CheckTestSuite(dir)
	if summary.TotalPackages != 5 {
		t.Errorf("TotalPackages: got %d, want 5 (from cache)", summary.TotalPackages)
	}
}

func TestCheckTestSuite_StaleCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a stale cache (older than cacheStaleDuration).
	staleTime := time.Now().Add(-2 * cacheStaleDuration)
	cached := TestSuiteSummary{
		TotalPackages:   5,
		PassingPackages: 5,
		HasFailure:      false,
		CheckedAt:       staleTime.Format(time.RFC3339),
	}
	cacheDir := filepath.Join(dir, ".kbz", "cache")
	os.MkdirAll(cacheDir, 0755)
	data, _ := json.Marshal(cached)
	os.WriteFile(filepath.Join(cacheDir, "test-suite-cache.json"), data, 0644)

	// Since there's no real Go project here, it should return empty after cache miss.
	summary := CheckTestSuite(dir)
	// The defaultRunTestSuite will see go.mod exists but `go test` will fail
	// in a temp dir without actual packages. The result may have HasFailure.
	// We just verify it attempted to run (went past cache).
	if summary.CheckedAt == staleTime.Format(time.RFC3339) {
		t.Error("CheckedAt: cache timestamp not updated, suggests stale cache was used")
	}
}

func TestCheckTestSuiteHealth_Empty(t *testing.T) {
	origFn := checkTestSuiteFn
	t.Cleanup(func() { checkTestSuiteFn = origFn })

	// Mock empty result (no go.mod)
	checkTestSuiteFn = func(repoPath string, now time.Time) TestSuiteSummary {
		return TestSuiteSummary{}
	}

	result := CheckTestSuiteHealth(".")
	if len(result.Issues) != 0 {
		t.Errorf("Issues: got %d, want 0 for empty test suite", len(result.Issues))
	}
}

func TestCheckTestSuiteHealth_Passing(t *testing.T) {
	origFn := checkTestSuiteFn
	t.Cleanup(func() { checkTestSuiteFn = origFn })

	checkTestSuiteFn = func(repoPath string, now time.Time) TestSuiteSummary {
		return TestSuiteSummary{
			TotalPackages:   10,
			PassingPackages: 10,
			HasFailure:      false,
		}
	}

	result := CheckTestSuiteHealth(".")
	if len(result.Issues) != 1 {
		t.Fatalf("Issues: got %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Severity != SeverityOK {
		t.Errorf("Severity: got %v, want %v", result.Issues[0].Severity, SeverityOK)
	}
}

func TestCheckTestSuiteHealth_Failing(t *testing.T) {
	origFn := checkTestSuiteFn
	t.Cleanup(func() { checkTestSuiteFn = origFn })

	checkTestSuiteFn = func(repoPath string, now time.Time) TestSuiteSummary {
		return TestSuiteSummary{
			TotalPackages:   10,
			PassingPackages: 8,
			FailingPackages: 2,
			Failures: []Failure{
				{Package: "./internal/foo", TestName: "TestFoo"},
			},
			HasFailure: true,
		}
	}

	result := CheckTestSuiteHealth(".")
	if len(result.Issues) != 1 {
		t.Fatalf("Issues: got %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Severity != SeverityError {
		t.Errorf("Severity: got %v, want %v", result.Issues[0].Severity, SeverityError)
	}
}

func TestFormatTestSuiteSummary(t *testing.T) {
	summary := TestSuiteSummary{
		TotalPackages:   10,
		PassingPackages: 8,
		FailingPackages: 2,
		Failures: []Failure{
			{Package: "./internal/foo", TestName: "TestFoo"},
			{Package: "./internal/bar", TestName: "TestBar"},
		},
	}

	output := FormatTestSuiteSummary(summary)
	if output["total_packages"].(int) != 10 {
		t.Errorf("total_packages: got %d, want 10", output["total_packages"])
	}
	if output["passing_packages"].(int) != 8 {
		t.Errorf("passing_packages: got %d, want 8", output["passing_packages"])
	}
	if output["failing_packages"].(int) != 2 {
		t.Errorf("failing_packages: got %d, want 2", output["failing_packages"])
	}
	failures := output["failures"].([]map[string]string)
	if len(failures) != 2 {
		t.Errorf("len(failures): got %d, want 2", len(failures))
	}
}

func TestSaveAndLoadTestSuiteCache(t *testing.T) {
	dir := t.TempDir()

	now := time.Now()
	summary := &TestSuiteSummary{
		TotalPackages:   10,
		PassingPackages: 10,
		FailingPackages: 0,
		Failures:        nil,
		HasFailure:      false,
		CheckedAt:       now.Format(time.RFC3339),
	}

	// Save.
	saveTestSuiteCache(dir, summary)

	// Load.
	loaded := loadTestSuiteCache(dir, now)
	if loaded == nil {
		t.Fatal("loadTestSuiteCache returned nil after save")
	}
	if loaded.TotalPackages != 10 {
		t.Errorf("TotalPackages: got %d, want 10", loaded.TotalPackages)
	}
	if loaded.CheckedAt != summary.CheckedAt {
		t.Errorf("CheckedAt: got %q, want %q", loaded.CheckedAt, summary.CheckedAt)
	}

	// Load with stale time.
	staleTime := now.Add(cacheStaleDuration + time.Minute)
	loaded = loadTestSuiteCache(dir, staleTime)
	if loaded != nil {
		t.Error("loadTestSuiteCache: expected nil for stale cache")
	}
}
