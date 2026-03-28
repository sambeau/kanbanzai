package merge

import (
	"fmt"
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
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
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
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
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
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
	}

	status := toString(ctx.Entity["verification_status"])
	if status != "passed" {
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
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
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
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
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
type HealthCheckCleanGate struct{}

func (g HealthCheckCleanGate) Name() string {
	return "health_check_clean"
}

func (g HealthCheckCleanGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g HealthCheckCleanGate) Check(ctx GateContext) GateResult {
	// Placeholder: always pass for now
	// Full implementation would check entity-specific health errors
	return GateResult{
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
	}
}

// EntityDoneGate checks that the entity has reached its "done" lifecycle state.
// Features must be "done"; bugs must be "closed". Other entity types pass unconditionally.
type EntityDoneGate struct{}

func (g EntityDoneGate) Name() string {
	return "entity_done"
}

func (g EntityDoneGate) Severity() GateSeverity {
	return GateSeverityBlocking
}

func (g EntityDoneGate) Check(ctx GateContext) GateResult {
	result := GateResult{
		Name:     g.Name(),
		Severity: g.Severity(),
		Status:   GateStatusPassed,
	}

	status := toString(ctx.Entity["status"])

	switch {
	case strings.HasPrefix(ctx.EntityID, "FEAT-"):
		if status != "done" {
			result.Status = GateStatusFailed
			if status == "" {
				result.Message = "feature status not set"
			} else {
				result.Message = fmt.Sprintf("feature status is %q, expected \"done\"", status)
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
