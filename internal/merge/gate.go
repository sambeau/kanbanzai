// Package merge provides merge gate checking for features and bugs.
// It validates that entities meet all required conditions before
// their branches can be merged.
package merge

// GateStatus represents the result of a gate check.
type GateStatus string

const (
	GateStatusPassed  GateStatus = "passed"
	GateStatusFailed  GateStatus = "failed"
	GateStatusWarning GateStatus = "warning"
)

// GateSeverity indicates whether a gate blocks merge.
type GateSeverity string

const (
	GateSeverityBlocking GateSeverity = "blocking"
	GateSeverityWarning  GateSeverity = "warning"
)

// GateResult is the result of checking a single gate.
type GateResult struct {
	// Name identifies the gate (e.g., "tasks_complete").
	Name string

	// Status is the result of the gate check.
	Status GateStatus

	// Severity indicates whether this gate blocks merge.
	Severity GateSeverity

	// Message provides a human-readable explanation.
	// Empty if the gate passed.
	Message string
}

// GateCheckResult is the combined result of all gate checks.
type GateCheckResult struct {
	// EntityID is the ID of the entity being checked.
	EntityID string

	// Branch is the branch name being evaluated.
	Branch string

	// OverallStatus summarizes the gate check outcome.
	// Values: "passed", "warnings", "blocked"
	OverallStatus string

	// Gates contains individual gate results in check order.
	Gates []GateResult
}

// OverallStatusPassed indicates all gates passed.
const OverallStatusPassed = "passed"

// OverallStatusWarnings indicates some gates have warnings but none are blocked.
const OverallStatusWarnings = "warnings"

// OverallStatusBlocked indicates at least one blocking gate failed.
const OverallStatusBlocked = "blocked"
