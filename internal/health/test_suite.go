package health

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TestSuiteSummary holds the parsed result of a go test run.
type TestSuiteSummary struct {
	TotalPackages   int       `json:"total_packages"`
	PassingPackages int       `json:"passing_packages"`
	FailingPackages int       `json:"failing_packages"`
	Failures        []Failure `json:"failures,omitempty"`
	HasFailure      bool      `json:"has_failure"`
	CheckedAt       string    `json:"checked_at"`
}

// Failure represents a single test failure.
type Failure struct {
	Package  string `json:"package"`
	TestName string `json:"test_name"`
}

// cacheFileName is the name of the test suite cache file relative to repo root.
const cacheFileName = ".kbz/cache/test-suite-cache.json"

// cacheStaleDuration defines how long cached results are considered fresh.
const cacheStaleDuration = 30 * time.Minute

// CheckTestSuite runs go test ./... and returns a TestSuiteSummary.
// If a recent cache exists, uses cached results instead (NFR-004).
func CheckTestSuite(repoPath string) TestSuiteSummary {
	return checkTestSuiteFn(repoPath, time.Now())
}

// checkTestSuiteFn is a variable so tests can inject mocks.
var checkTestSuiteFn = defaultCheckTestSuite

func defaultCheckTestSuite(repoPath string, now time.Time) TestSuiteSummary {
	if repoPath == "" {
		return TestSuiteSummary{}
	}

	// Check for go.mod first.
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); os.IsNotExist(err) {
		return TestSuiteSummary{}
	}

	// Try cache first (NFR-004: must not run full test suite on every health call).
	if cached := loadTestSuiteCache(repoPath, now); cached != nil {
		return *cached
	}

	// Run go test ./...
	result := runAndParseTestSuite(repoPath, now)

	// Write cache.
	saveTestSuiteCache(repoPath, &result)

	return result
}

// runAndParseTestSuite executes go test and parses the output.
func runAndParseTestSuite(repoPath string, now time.Time) TestSuiteSummary {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = repoPath
	raw, err := cmd.CombinedOutput()
	output := string(raw)

	summary := TestSuiteSummary{
		CheckedAt: now.Format(time.RFC3339),
	}

	if err != nil {
		summary.HasFailure = true
	}

	// Track unique failing packages to avoid duplicates due to multi-line output.
	seenPackages := make(map[string]bool)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ok ") {
			summary.TotalPackages++
			summary.PassingPackages++
		} else if strings.HasPrefix(line, "FAIL ") {
			summary.TotalPackages++
			pkgName := strings.TrimSpace(strings.TrimPrefix(line, "FAIL "))
			if pkgName != "" && !seenPackages[pkgName] {
				seenPackages[pkgName] = true
				summary.FailingPackages++
			}
		} else if strings.HasPrefix(line, "? ") {
			summary.TotalPackages++
			summary.PassingPackages++
		}

		// Extract failing test names with their package context.
		if strings.Contains(line, "--- FAIL:") {
			parts := strings.SplitN(line, "--- FAIL:", 2)
			if len(parts) == 2 {
				testName := strings.TrimSpace(parts[1])
				if idx := strings.LastIndex(testName, "("); idx > 0 {
					testName = strings.TrimSpace(testName[:idx])
				}
				// Determine the package from context (the last "ok"/"FAIL" line before this)
				summary.Failures = append(summary.Failures, Failure{
					Package:  findPackageForTest(lines, line),
					TestName: testName,
				})
			}
		}
	}

	return summary
}

// findPackageForTest finds the most recent package line (ok/FAIL/?) before the given FAIL line.
func findPackageForTest(lines []string, failLine string) string {
	for i, line := range lines {
		if line == failLine && i > 0 {
			// Scan backwards from the FAIL line
			for j := i - 1; j >= 0; j-- {
				prev := lines[j]
				if strings.HasPrefix(prev, "ok ") {
					return strings.TrimSpace(strings.TrimPrefix(prev, "ok "))
				}
				if strings.HasPrefix(prev, "FAIL ") {
					return strings.TrimSpace(strings.TrimPrefix(prev, "FAIL "))
				}
			}
		}
	}
	return ""
}

// loadTestSuiteCache attempts to load a cached test suite result.
// Returns nil if cache is missing, stale, or unparseable.
func loadTestSuiteCache(repoPath string, now time.Time) *TestSuiteSummary {
	cachePath := filepath.Join(repoPath, cacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}

	var cached TestSuiteSummary
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}

	// Check staleness.
	checkedAt, err := time.Parse(time.RFC3339, cached.CheckedAt)
	if err != nil {
		return nil
	}

	if now.Sub(checkedAt) > cacheStaleDuration {
		return nil // cache is stale
	}

	return &cached
}

// saveTestSuiteCache writes the test suite result to the cache file.
func saveTestSuiteCache(repoPath string, summary *TestSuiteSummary) {
	cachePath := filepath.Join(repoPath, cacheFileName)

	// Ensure cache directory exists.
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return // best-effort
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return // best-effort
	}

	_ = os.WriteFile(cachePath, data, 0644) // best-effort
}

// CheckTestSuiteHealth checks the test suite status and returns a CategoryResult.
// Returns a single issue (ok/info/warning/error) based on test results.
func CheckTestSuiteHealth(repoPath string) CategoryResult {
	result := NewCategoryResult()

	summary := CheckTestSuite(repoPath)

	if summary.TotalPackages == 0 {
		// No go.mod or no packages tested — skip.
		return result
	}

	if summary.HasFailure {
		msg := fmt.Sprintf("test suite: %d package(s) tested, %d passed, %d failed",
			summary.TotalPackages, summary.PassingPackages, summary.FailingPackages)
		if len(summary.Failures) > 0 {
			failNames := make([]string, len(summary.Failures))
			for i, f := range summary.Failures {
				failNames[i] = f.TestName
			}
			msg += fmt.Sprintf("; failing tests: %s", strings.Join(failNames, ", "))
		}
		result.AddIssue(Issue{
			Severity: SeverityError,
			Message:  msg,
		})
	} else {
		result.AddIssue(Issue{
			Severity: SeverityOK,
			Message:  fmt.Sprintf("test suite: all %d package(s) passed", summary.TotalPackages),
		})
	}

	return result
}

// FormatTestSuiteSummary returns the test suite summary as a structured map for output.
func FormatTestSuiteSummary(summary TestSuiteSummary) map[string]any {
	output := map[string]any{
		"total_packages":   summary.TotalPackages,
		"passing_packages": summary.PassingPackages,
		"failing_packages": summary.FailingPackages,
	}

	if len(summary.Failures) > 0 {
		failures := make([]map[string]string, len(summary.Failures))
		for i, f := range summary.Failures {
			failures[i] = map[string]string{
				"package":   f.Package,
				"test_name": f.TestName,
			}
		}
		output["failures"] = failures
	}

	return output
}

// TestSuiteResult is the interface that wraps CheckTestSuite for use as a health category.
// It returns a CategoryResult that can be merged into the health report.
// This is the primary entry point from the MCP health tool.
func TestSuiteResult(repoPath string) CategoryResult {
	return CheckTestSuiteHealth(repoPath)
}
