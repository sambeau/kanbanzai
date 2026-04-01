package service

// checkDescriptionPresent returns an error finding for each task with an
// empty or whitespace-only summary.
func checkDescriptionPresent(p Proposal) []Finding {
	return nil
}

// checkTestingCoverage returns a warning finding if no task mentions testing
// or verification in its summary or rationale.
func checkTestingCoverage(p Proposal) []Finding {
	return nil
}

// checkDependenciesDeclared returns a warning finding when a task references
// another task's slug in its summary or rationale without declaring a
// dependency.
func checkDependenciesDeclared(p Proposal) []Finding {
	return nil
}

// checkOrphanTasks returns a warning finding for each task that has no
// dependency edges while other tasks do.
func checkOrphanTasks(p Proposal) []Finding {
	return nil
}

// checkSingleAgentSizing returns a warning finding for each task whose
// summary contains multiple action clauses separated by coordinating
// conjunctions, suggesting the task is too large for a single agent.
func checkSingleAgentSizing(p Proposal) []Finding {
	return nil
}
