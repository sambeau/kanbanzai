// Package health provides health check functionality for the Kanbanzai system.
// It checks various aspects of project health including worktrees, branches,
// knowledge entries, and cleanup status.
package health

// Severity indicates the severity of a health issue.
type Severity string

const (
	// SeverityOK indicates no issues.
	SeverityOK Severity = "ok"
	// SeverityInfo indicates an informational observation that requires no action.
	SeverityInfo Severity = "info"
	// SeverityWarning indicates a non-critical issue that should be addressed.
	SeverityWarning Severity = "warning"
	// SeverityError indicates a critical issue requiring immediate attention.
	SeverityError Severity = "error"
)

// Issue represents a single health check issue.
type Issue struct {
	// Severity indicates how critical this issue is.
	Severity Severity

	// EntityID is set for entity-related issues (worktree, feature, bug, etc).
	EntityID string

	// EntryID is set for knowledge-related issues.
	EntryID string

	// Entries is set for multi-entry issues like conflicts.
	Entries []string

	// Message describes the issue.
	Message string
}

// CategoryResult is the result of checking a single category.
type CategoryResult struct {
	// Status is the overall status of this category (worst severity of all issues).
	Status Severity

	// Issues contains all issues found in this category.
	Issues []Issue
}

// HealthResult is the combined result of all health checks.
type HealthResult struct {
	// Status is the overall status (worst of all categories).
	Status Severity

	// Categories contains results for each checked category.
	Categories map[string]CategoryResult
}

// NewCategoryResult creates a new CategoryResult with status OK and no issues.
func NewCategoryResult() CategoryResult {
	return CategoryResult{
		Status: SeverityOK,
		Issues: nil,
	}
}

// AddIssue adds an issue to the category result and updates the status.
func (r *CategoryResult) AddIssue(issue Issue) {
	r.Issues = append(r.Issues, issue)
	r.Status = WorstSeverity(r.Status, issue.Severity)
}

// WorstSeverity returns the more severe of two severities.
// Error > Warning > Info > OK.
func WorstSeverity(a, b Severity) Severity {
	if a == SeverityError || b == SeverityError {
		return SeverityError
	}
	if a == SeverityWarning || b == SeverityWarning {
		return SeverityWarning
	}
	if a == SeverityInfo || b == SeverityInfo {
		return SeverityInfo
	}
	return SeverityOK
}
