package merge

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/sambeau/kanbanzai/internal/git"
)

// Gate defines the interface for a merge gate check.
type Gate interface {
	// Name returns the gate identifier.
	Name() string

	// Severity returns whether the gate is blocking or warning.
	Severity() GateSeverity

	// Check evaluates the gate and returns a result.
	Check(ctx GateContext) GateResult
}

// GateContext provides all the data needed to evaluate gates.
type GateContext struct {
	// RepoPath is the path to the Git repository.
	RepoPath string

	// EntityID is the ID of the entity being checked.
	EntityID string

	// Branch is the branch name to evaluate.
	Branch string

	// Entity contains the entity's fields from storage.
	Entity map[string]any

	// Tasks contains the fields of all child tasks.
	Tasks []map[string]any

	// Thresholds configures branch staleness thresholds.
	Thresholds git.BranchThresholds

	// ConflictChecker allows injection of conflict detection for testing.
	// If nil, uses git.HasMergeConflicts.
	ConflictChecker func(repoPath, branch, base string) (bool, error)

	// BranchStatusChecker allows injection of branch status evaluation for testing.
	// If nil, uses git.EvaluateBranchStatus.
	BranchStatusChecker func(repoPath, branch string, thresholds git.BranchThresholds) (git.BranchStatus, error)

	// DefaultBranchDetector allows injection of default branch detection for testing.
	// If nil, uses git.GetDefaultBranch.
	DefaultBranchDetector func(repoPath string) (string, error)

	// DocSvc provides access to the document store for gate evaluation.
	// If nil, gates that require it fail open.
	DocSvc DocService

	// TestRunner runs go test and returns parsed output. If nil, uses DefaultTestRunner.
	// RunTests is the function that executes the test suite and returns test results.
	// The returned string is the raw combined stdout+stderr from the test run.
	TestRunner func(repoPath string) (TestSuiteResult, string)
}

// TasksCompleteGate checks that all tasks are done or wont_do.
type TasksCompleteGate struct{}

func (g TasksCompleteGate) Name() string {
	return "tasks_complete"
}

func (g TasksCompleteGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g TasksCompleteGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	if len(ctx.Tasks) == 0 {
		// No tasks means nothing to verify - pass
		return result
	}

	var incomplete []string
	for _, task := range ctx.Tasks {
		status := toString(task["status"])
		taskID := toString(task["id"])

		// Terminal states that allow merge
		if status == "done" || status == "wont_do" {
			continue
		}

		// Track incomplete task
		if taskID != "" {
			incomplete = append(incomplete, taskID)
		} else {
			incomplete = append(incomplete, "(unknown)")
		}
	}

	if len(incomplete) > 0 {
		result.Status = GateStatusFailed
		if len(incomplete) == 1 {
			result.Message = fmt.Sprintf("task not complete: %s", incomplete[0])
		} else {
			result.Message = fmt.Sprintf("%d tasks not complete: %s",
				len(incomplete), strings.Join(incomplete, ", "))
		}
	}

	return result
}

// VerificationExistsGate checks that the entity has a non-empty verification field.
type VerificationExistsGate struct{}

func (g VerificationExistsGate) Name() string {
	return "verification_exists"
}

func (g VerificationExistsGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g VerificationExistsGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	verification := toString(ctx.Entity["verification"])
	if strings.TrimSpace(verification) == "" {
		result.Status = GateStatusFailed
		result.Message = "verification field is empty"
	}

	return result
}

// VerificationPassedGate checks that verification_status is "passed".
type VerificationPassedGate struct{}

func (g VerificationPassedGate) Name() string {
	return "verification_passed"
}

func (g VerificationPassedGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g VerificationPassedGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	status := toString(ctx.Entity["verification_status"])
	switch status {
	case "passed":
		// already GateStatusPassed
	case "partial":
		result.Status = GateStatusWarning
		result.Message = "verification_status is \"partial\", expected \"passed\""
	default:
		result.Status = GateStatusFailed
		if status == "" {
			result.Message = "verification_status not set"
		} else {
			result.Message = fmt.Sprintf("verification_status is %q, expected \"passed\"", status)
		}
	}

	return result
}

// BranchNotStaleGate checks that the branch is not stale.
type BranchNotStaleGate struct{}

func (g BranchNotStaleGate) Name() string {
	return "branch_not_stale"
}

func (g BranchNotStaleGate) Severity() GateSeverity {
	return GateSeverityWarning
}

func (g BranchNotStaleGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	if ctx.Branch == "" {
		result.Status = GateStatusWarning
		result.Message = "no branch specified"
		return result
	}

	if ctx.RepoPath == "" {
		result.Status = GateStatusWarning
		result.Message = "no repository path specified"
		return result
	}

	// Use injected checker or default
	evaluator := ctx.BranchStatusChecker
	if evaluator == nil {
		evaluator = git.EvaluateBranchStatus
	}

	thresholds := ctx.Thresholds
	if thresholds.StaleAfterDays == 0 && thresholds.DriftWarningCommits == 0 {
		thresholds = git.DefaultBranchThresholds()
	}

	status, err := evaluator(ctx.RepoPath, ctx.Branch, thresholds)
	if err != nil {
		result.Status = GateStatusWarning
		result.Message = fmt.Sprintf("cannot evaluate branch status: %v", err)
		return result
	}

	// Combine warnings and errors into messages
	var issues []string
	issues = append(issues, status.Warnings...)
	issues = append(issues, status.Errors...)

	if len(issues) > 0 {
		result.Status = GateStatusWarning
		result.Message = strings.Join(issues, "; ")
	}

	return result
}

// NoConflictsGate checks that the branch has no merge conflicts with main.
type NoConflictsGate struct{}

func (g NoConflictsGate) Name() string {
	return "no_conflicts"
}

func (g NoConflictsGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g NoConflictsGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	if ctx.Branch == "" {
		result.Status = GateStatusFailed
		result.Message = "no branch specified"
		return result
	}

	if ctx.RepoPath == "" {
		result.Status = GateStatusFailed
		result.Message = "no repository path specified"
		return result
	}

	// Use injected checker or default
	checker := ctx.ConflictChecker
	if checker == nil {
		checker = git.HasMergeConflicts
	}

	// Use injected detector or default
	detectDefault := ctx.DefaultBranchDetector
	if detectDefault == nil {
		detectDefault = git.GetDefaultBranch
	}

	baseBranch, err := detectDefault(ctx.RepoPath)
	if err != nil {
		result.Status = GateStatusFailed
		result.Message = fmt.Sprintf("cannot determine default branch: %v", err)
		return result
	}
	hasConflicts, err := checker(ctx.RepoPath, ctx.Branch, baseBranch)
	if err != nil {
		result.Status = GateStatusFailed
		result.Message = fmt.Sprintf("cannot check for conflicts: %v", err)
		return result
	}

	if hasConflicts {
		result.Status = GateStatusFailed
		result.Message = "branch has merge conflicts with main"
	}

	return result
}

// HealthCheckCleanGate checks that there are no blocking health-check errors.
// This is a placeholder implementation that always passes.
// TODO: implement using entity-specific health check errors. Until then, this
// gate provides no protection — merges proceed even when blocking health errors
// exist. Consider removing from DefaultGates() until a real implementation lands.
type HealthCheckCleanGate struct{}

func (g HealthCheckCleanGate) Name() string {
	return "health_check_clean"
}

func (g HealthCheckCleanGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g HealthCheckCleanGate) Check(ctx GateContext) GateResult {
	// TODO: wire up entity-scoped health check results here.
	// Placeholder: always passes — see comment on HealthCheckCleanGate.
	return GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}
}

// EntityDoneGate checks that the entity has reached its "done" lifecycle state.
// Features must be "done"; bugs must be "closed". Other entity types pass unconditionally.
// This gate exists in addition to tasks_complete because an entity can have all
// tasks done while the entity itself is still in review or another intermediate
// state. See spec §8.2.
type EntityDoneGate struct{}

func (g EntityDoneGate) Name() string {
	return "entity_done"
}

func (g EntityDoneGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g EntityDoneGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	status := toString(ctx.Entity["status"])

	switch {
	case strings.HasPrefix(ctx.EntityID, "FEAT-"):
		if status != "done" && status != "reviewing" {
			result.Status = GateStatusFailed
			if status == "" {
				result.Message = "feature status not set"
			} else {
				result.Message = fmt.Sprintf("feature status is %q, expected \"done\" or \"reviewing\"", status)
			}
		}
	case strings.HasPrefix(ctx.EntityID, "BUG-"):
		if status != "closed" {
			result.Status = GateStatusFailed
			if status == "" {
				result.Message = "bug status not set"
			} else {
				result.Message = fmt.Sprintf("bug status is %q, expected \"closed\"", status)
			}
		}
	}

	return result
}

// ReviewReportExistsGate checks that a feature in reviewing status has at least
// one registered report document. This gate is non-bypassable: override: true
// cannot skip it (FR-007, FR-008 in spec FEAT-01KPXGVQY3KQC).
type ReviewReportExistsGate struct{}

func (g ReviewReportExistsGate) Name() string {
	return "review_report_exists"
}

func (g ReviewReportExistsGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g ReviewReportExistsGate) Check(ctx GateContext) GateResult {
	// Only activate for reviewing features (FR-002).
	status := toString(ctx.Entity["status"])
	if status != "reviewing" {
		return GateResult{
			Name:       g.Name(),
			Severity:   g.Severity(),
			Status:     GateStatusPassed,
			Bypassable: true,
		}
	}

	// Fail-open: no doc service → pass with a log warning (FR-011, FR-012).
	if ctx.DocSvc == nil {
		slog.Warn("ReviewReportExistsGate: DocSvc is nil, failing open", "component", "merge gate", "entityID", ctx.EntityID)
		return GateResult{
			Name:       g.Name(),
			Severity:   g.Severity(),
			Status:     GateStatusPassed,
			Bypassable: true,
		}
	}

	docs, err := ctx.DocSvc.ListDocuments(DocFilters{Owner: ctx.EntityID, Type: "report"})
	if err != nil {
		// Fail-open on service error (FR-011, FR-012).
		slog.Warn("ReviewReportExistsGate: document service error, failing open", "component", "merge gate", "entityID", ctx.EntityID, "error", err)
		return GateResult{
			Name:       g.Name(),
			Severity:   g.Severity(),
			Status:     GateStatusPassed,
			Bypassable: true,
		}
	}

	if len(docs) == 0 {
		return GateResult{
			Name:       g.Name(),
			Severity:   g.Severity(),
			Status:     GateStatusFailed,
			Bypassable: false, // non-bypassable (FR-007)
			Message:    fmt.Sprintf("feature %s is in 'reviewing' status but no review report is registered", ctx.EntityID),
		}
	}

	return GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}
}

// TestSuiteResult holds the parsed output of a go test run.
type TestSuiteResult struct {
	// FailedPackages contains the names of packages with test failures.
	FailedPackages []string

	// FailingTests contains the names of individual failing tests.
	FailingTests []string

	// HasFailure is true when any test in the suite failed.
	HasFailure bool

	// TotalPackages is the total number of packages tested.
	TotalPackages int
}

// DefaultTestRunner runs go test ./... from the given repo directory
// and parses the output into a TestSuiteResult.
func DefaultTestRunner(repoPath string) (TestSuiteResult, string) {
	result := TestSuiteResult{}

	if repoPath == "" {
		return result, ""
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = repoPath
	raw, err := cmd.CombinedOutput()
	output := string(raw)

	if err != nil {
		result.HasFailure = true
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Count packages: lines starting with "ok", "FAIL", or "?"
		if strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "FAIL ") || strings.HasPrefix(line, "? ") {
			result.TotalPackages++
			if strings.HasPrefix(line, "FAIL ") {
				pkgName := strings.TrimSpace(strings.TrimPrefix(line, "FAIL "))
				result.FailedPackages = append(result.FailedPackages, pkgName)
			}
		}

		// Extract failing test names
		if strings.Contains(line, "--- FAIL:") {
			// Format: "    --- FAIL: TestName (0.00s)"
			parts := strings.SplitN(line, "--- FAIL:", 2)
			if len(parts) == 2 {
				testName := strings.TrimSpace(parts[1])
				// Remove duration suffix if present
				if idx := strings.LastIndex(testName, "("); idx > 0 {
					testName = strings.TrimSpace(testName[:idx])
				}
				result.FailingTests = append(result.FailingTests, testName)
			}
		}
	}

	return result, output
}

// TestSuiteGate checks that go test ./... passes before merge.
type TestSuiteGate struct{}

func (g TestSuiteGate) Name() string {
	return "test_suite_pass"
}

func (g TestSuiteGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g TestSuiteGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:       g.Name(),
		Severity:   g.Severity(),
		Status:     GateStatusPassed,
		Bypassable: true,
	}

	if ctx.RepoPath == "" {
		result.Status = GateStatusWarning
		result.Message = "no repository path specified — skipping test suite check"
		return result
	}

	runner := ctx.TestRunner
	if runner == nil {
		runner = DefaultTestRunner
	}

	testResult, _ := runner(ctx.RepoPath)

	if testResult.HasFailure {
		result.Status = GateStatusFailed
		if len(testResult.FailingTests) > 0 {
			result.Message = fmt.Sprintf("test suite failed: %d failing test(s): %s",
				len(testResult.FailingTests), strings.Join(testResult.FailingTests, ", "))
		} else {
			result.Message = fmt.Sprintf("test suite failed: %d package(s) with failures", len(testResult.FailedPackages))
		}
	} else {
		result.Message = fmt.Sprintf("all %d package(s) passed", testResult.TotalPackages)
	}

	return result
}

// toString extracts a string from an any value, returning "" if nil or not a string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}
