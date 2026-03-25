package merge

// DefaultGates returns all gates in the standard evaluation order.
// Blocking gates are evaluated first, followed by warning gates.
func DefaultGates() []Gate {
	return []Gate{
		TasksCompleteGate{},
		VerificationExistsGate{},
		VerificationPassedGate{},
		NoConflictsGate{},
		HealthCheckCleanGate{},
		BranchNotStaleGate{},
	}
}

// CheckGates runs all gates and returns a combined result.
func CheckGates(ctx GateContext) GateCheckResult {
	return CheckGatesWithList(ctx, DefaultGates())
}

// CheckGatesWithList runs the specified gates and returns a combined result.
func CheckGatesWithList(ctx GateContext, gates []Gate) GateCheckResult {
	result := GateCheckResult{
		EntityID: ctx.EntityID,
		Branch:   ctx.Branch,
		Gates:    make([]GateResult, 0, len(gates)),
	}

	for _, gate := range gates {
		gateResult := gate.Check(ctx)
		result.Gates = append(result.Gates, gateResult)
	}

	result.OverallStatus = DetermineOverallStatus(result.Gates)
	return result
}

// DetermineOverallStatus determines the overall status from gate results.
//
// Logic:
//   - Any blocking gate with status=failed → "blocked"
//   - Any gate with status=warning or status=failed (non-blocking) → "warnings"
//   - All gates passed → "passed"
func DetermineOverallStatus(results []GateResult) string {
	hasWarnings := false

	for _, r := range results {
		// Blocking failure → blocked
		if r.Severity == GateSeverityBlocking && r.Status == GateStatusFailed {
			return OverallStatusBlocked
		}

		// Any warning or non-blocking failure → track as warning
		if r.Status == GateStatusWarning {
			hasWarnings = true
		}
		if r.Status == GateStatusFailed && r.Severity == GateSeverityWarning {
			hasWarnings = true
		}
	}

	if hasWarnings {
		return OverallStatusWarnings
	}

	return OverallStatusPassed
}

// CountByStatus counts gate results by status.
func CountByStatus(results []GateResult) (passed, failed, warning int) {
	for _, r := range results {
		switch r.Status {
		case GateStatusPassed:
			passed++
		case GateStatusFailed:
			failed++
		case GateStatusWarning:
			warning++
		}
	}
	return
}

// BlockingFailures returns all blocking gates that failed.
func BlockingFailures(results []GateResult) []GateResult {
	var failures []GateResult
	for _, r := range results {
		if r.Severity == GateSeverityBlocking && r.Status == GateStatusFailed {
			failures = append(failures, r)
		}
	}
	return failures
}
