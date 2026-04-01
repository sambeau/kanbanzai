package service

import (
	"fmt"
	"strings"
)

// TaskStatusPair carries a task ID and its current status for use in gate
// failure error messages that identify non-terminal tasks (FR-022, FR-025).
type TaskStatusPair struct {
	ID     string
	Status string
}

// GateFailureResponse builds a structured gate failure response suitable for
// returning directly as an MCP tool result (FR-026). The response contains:
//   - "error": actionable message string following the FR-018 template
//   - "gate_failed": map identifying the from/to statuses of the failed transition
//
// nonTerminalTasks is used when the gate failure is due to outstanding child
// tasks (developing→reviewing, needs-rework→reviewing). Pass nil for other gates.
func GateFailureResponse(featureID, from, to string, gate GateResult, nonTerminalTasks []TaskStatusPair) map[string]any {
	msg := buildGateMessage(featureID, from, to, gate, nonTerminalTasks)
	return map[string]any{
		"error": msg,
		"gate_failed": map[string]any{
			"from_status": from,
			"to_status":   to,
		},
	}
}

// buildGateMessage formats the actionable error message (FR-018):
//
//	Cannot transition {id} from "{from}" to "{to}": {reason}.
//
//	To resolve:
//	  1. {step1}
//	  2. {step2}
func buildGateMessage(featureID, from, to string, gate GateResult, nonTerminalTasks []TaskStatusPair) string {
	reason := gate.Reason
	if reason == "" {
		reason = "prerequisite not met"
	}

	steps := recoverySteps(featureID, from, to, gate, nonTerminalTasks)

	var b strings.Builder
	fmt.Fprintf(&b, "Cannot transition %s from %q to %q: %s.\n\nTo resolve:\n", featureID, from, to, reason)
	for i, step := range steps {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, step)
	}
	return strings.TrimRight(b.String(), "\n")
}

// recoverySteps returns ordered recovery steps for the given gate failure.
func recoverySteps(featureID, from, to string, gate GateResult, nonTerminalTasks []TaskStatusPair) []string {
	switch from + "→" + to {
	case "designing→specifying":
		return designGateRecovery(featureID)
	case "specifying→dev-planning":
		return specGateRecovery(featureID)
	case "dev-planning→developing":
		return devPlanGateRecovery(featureID, gate.Reason)
	case "developing→reviewing", "needs-rework→reviewing":
		return taskCompletionGateRecovery(featureID, nonTerminalTasks)
	case "reviewing→done":
		return reviewReportGateRecovery(featureID)
	case "needs-rework→developing":
		return reworkTaskGateRecovery(featureID)
	default:
		return []string{"Verify gate prerequisites and retry the transition."}
	}
}

// designGateRecovery returns recovery steps for the designing→specifying gate
// failure (no approved design document). FR-019.
func designGateRecovery(featureID string) []string {
	return []string{
		fmt.Sprintf(`Check for pending documents: doc(action: "list", owner: %q, pending: true)`, featureID),
		fmt.Sprintf(`Approve an existing draft design document: doc(action: "approve", id: "DOC-...")`),
		fmt.Sprintf(`Register a new design document: doc(action: "register", path: "work/design/...", type: "design", owner: %q, title: "Design: ...")`, featureID),
	}
}

// specGateRecovery returns recovery steps for the specifying→dev-planning gate
// failure (no approved specification). FR-020.
func specGateRecovery(featureID string) []string {
	return []string{
		fmt.Sprintf(`Check for pending documents: doc(action: "list", owner: %q, pending: true)`, featureID),
		fmt.Sprintf(`Approve an existing draft specification: doc(action: "approve", id: "DOC-...")`),
		fmt.Sprintf(`Register a new specification: doc(action: "register", path: "work/spec/...", type: "specification", owner: %q, title: "Specification: ...")`, featureID),
	}
}

// devPlanGateRecovery returns recovery steps for the dev-planning→developing
// gate failure. Steps target whichever sub-prerequisite failed: the dev-plan
// document or the child task requirement. FR-021.
func devPlanGateRecovery(featureID, reason string) []string {
	// checkDevelopingGate returns "feature has no child tasks" when tasks are
	// missing; checkDocumentGate returns "no approved dev-plan document found".
	if strings.Contains(reason, "child task") {
		return []string{
			fmt.Sprintf(`Break the feature into tasks: decompose(action: "propose", feature_id: %q)`, featureID),
			fmt.Sprintf(`Or create a task directly: entity(action: "create", type: "task", parent_feature: %q, summary: "...", slug: "...")`, featureID),
		}
	}
	return []string{
		fmt.Sprintf(`Check for pending documents: doc(action: "list", owner: %q, pending: true)`, featureID),
		fmt.Sprintf(`Approve an existing draft dev-plan: doc(action: "approve", id: "DOC-...")`),
		fmt.Sprintf(`Register a new dev-plan: doc(action: "register", path: "work/plan/...", type: "dev-plan", owner: %q, title: "Implementation Plan: ...")`, featureID),
	}
}

// taskCompletionGateRecovery returns recovery steps when outstanding child tasks
// block a transition to review. Used by developing→reviewing (FR-022) and
// needs-rework→reviewing (FR-025).
func taskCompletionGateRecovery(featureID string, nonTerminalTasks []TaskStatusPair) []string {
	if len(nonTerminalTasks) == 0 {
		return []string{
			fmt.Sprintf(`List outstanding tasks: entity(action: "list", type: "task", parent: %q)`, featureID),
			`Complete or cancel all non-terminal tasks before proceeding to review.`,
		}
	}

	steps := make([]string, 0, len(nonTerminalTasks)*2)
	for _, task := range nonTerminalTasks {
		steps = append(steps,
			fmt.Sprintf(`Complete task %s (%s): finish(task_id: %q)`, task.ID, task.Status, task.ID),
			fmt.Sprintf(`Or mark not-planned: entity(action: "transition", id: %q, status: "not-planned")`, task.ID),
		)
	}
	return steps
}

// reviewReportGateRecovery returns recovery steps for the reviewing→done gate
// failure (no review report document). FR-023.
func reviewReportGateRecovery(featureID string) []string {
	return []string{
		fmt.Sprintf(`Register a review report: doc(action: "register", path: "work/reports/...", type: "report", owner: %q, title: "Review Report: ...")`, featureID),
		`The report documents review findings and must be registered before the feature can be marked done.`,
	}
}

// reworkTaskGateRecovery returns recovery steps for the needs-rework→developing
// gate failure (no rework tasks). FR-024.
func reworkTaskGateRecovery(featureID string) []string {
	return []string{
		fmt.Sprintf(`Create a rework task: entity(action: "create", type: "task", parent_feature: %q, summary: "Rework: ...", slug: "rework-...")`, featureID),
		`At least one non-terminal rework task must exist before resuming development.`,
	}
}
