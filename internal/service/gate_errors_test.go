package service

import (
	"strings"
	"testing"
)

func TestGateFailureResponse_Structure(t *testing.T) {
	t.Parallel()

	gate := GateResult{Satisfied: false, Reason: "no approved design document found"}
	resp := GateFailureResponse("FEAT-01AAAA", "designing", "specifying", gate, nil)

	if _, ok := resp["error"]; !ok {
		t.Fatal("response must contain 'error' field")
	}
	gateFailed, ok := resp["gate_failed"].(map[string]any)
	if !ok {
		t.Fatal("response must contain 'gate_failed' map")
	}
	if gateFailed["from_status"] != "designing" {
		t.Errorf("gate_failed.from_status = %q, want %q", gateFailed["from_status"], "designing")
	}
	if gateFailed["to_status"] != "specifying" {
		t.Errorf("gate_failed.to_status = %q, want %q", gateFailed["to_status"], "specifying")
	}
}

func TestGateFailureResponse_ErrorMessageContainsFeatureID(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01BBBB"
	gate := GateResult{Satisfied: false, Reason: "no approved design document found"}
	resp := GateFailureResponse(featureID, "designing", "specifying", gate, nil)

	msg, _ := resp["error"].(string)
	if !strings.Contains(msg, featureID) {
		t.Errorf("error message %q does not contain feature ID %q", msg, featureID)
	}
}

func TestGateFailureResponse_ErrorMessageContainsFromTo(t *testing.T) {
	t.Parallel()

	gate := GateResult{Satisfied: false, Reason: "no approved specification found"}
	resp := GateFailureResponse("FEAT-01CCCC", "specifying", "dev-planning", gate, nil)

	msg, _ := resp["error"].(string)
	if !strings.Contains(msg, "specifying") {
		t.Errorf("error message does not contain from-status %q: %s", "specifying", msg)
	}
	if !strings.Contains(msg, "dev-planning") {
		t.Errorf("error message does not contain to-status %q: %s", "dev-planning", msg)
	}
}

func TestGateFailureResponse_ErrorMessageContainsToResolve(t *testing.T) {
	t.Parallel()

	gate := GateResult{Satisfied: false, Reason: "no approved design document found"}
	resp := GateFailureResponse("FEAT-01DDDD", "designing", "specifying", gate, nil)

	msg, _ := resp["error"].(string)
	if !strings.Contains(msg, "To resolve:") {
		t.Errorf("error message does not contain 'To resolve:' section: %s", msg)
	}
}

func TestGateFailureResponse_Deterministic(t *testing.T) {
	t.Parallel()

	gate := GateResult{Satisfied: false, Reason: "no approved specification found"}
	resp1 := GateFailureResponse("FEAT-01EEEE", "specifying", "dev-planning", gate, nil)
	resp2 := GateFailureResponse("FEAT-01EEEE", "specifying", "dev-planning", gate, nil)

	msg1, _ := resp1["error"].(string)
	msg2, _ := resp2["error"].(string)
	if msg1 != msg2 {
		t.Errorf("gate failure messages not deterministic:\n  msg1=%q\n  msg2=%q", msg1, msg2)
	}
}

func TestDesignGateRecovery_ContainsDocCalls(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01FFFF"
	gate := GateResult{Satisfied: false, Reason: "no approved design document found"}
	resp := GateFailureResponse(featureID, "designing", "specifying", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`doc(action: "list"`,
		`doc(action: "approve"`,
		`doc(action: "register"`,
		`type: "design"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("design gate recovery missing %q in message: %s", want, msg)
		}
	}
}

func TestSpecGateRecovery_ContainsDocCalls(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01GGGG"
	gate := GateResult{Satisfied: false, Reason: "no approved specification document found"}
	resp := GateFailureResponse(featureID, "specifying", "dev-planning", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`doc(action: "list"`,
		`doc(action: "approve"`,
		`doc(action: "register"`,
		`type: "specification"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("spec gate recovery missing %q in message: %s", want, msg)
		}
	}
}

func TestDevPlanGateRecovery_MissingDoc(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01HHHH"
	gate := GateResult{Satisfied: false, Reason: "no approved dev-plan document found"}
	resp := GateFailureResponse(featureID, "dev-planning", "developing", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`doc(action: "register"`,
		`type: "dev-plan"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("dev-plan gate recovery (missing doc) missing %q in message: %s", want, msg)
		}
	}
}

func TestDevPlanGateRecovery_MissingTasks(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01IIII"
	gate := GateResult{Satisfied: false, Reason: "feature has no child tasks"}
	resp := GateFailureResponse(featureID, "dev-planning", "developing", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`decompose(action: "propose"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("dev-plan gate recovery (missing tasks) missing %q in message: %s", want, msg)
		}
	}
}

func TestTaskCompletionGateRecovery_ListsTasks(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01JJJJ"
	gate := GateResult{Satisfied: false, Reason: "non-terminal child tasks: TASK-01AA (active)"}
	nonTerminal := []TaskStatusPair{
		{ID: "TASK-01AA", Status: "active"},
		{ID: "TASK-01BB", Status: "ready"},
	}
	resp := GateFailureResponse(featureID, "developing", "reviewing", gate, nonTerminal)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		"TASK-01AA",
		"TASK-01BB",
		`finish(task_id:`,
		`entity(action: "transition"`,
		`"not-planned"`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("task completion gate recovery missing %q in message: %s", want, msg)
		}
	}
}

func TestTaskCompletionGateRecovery_NeedsReworkToReviewing(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01KKKK"
	gate := GateResult{Satisfied: false, Reason: "non-terminal child tasks: TASK-01CC (active)"}
	nonTerminal := []TaskStatusPair{{ID: "TASK-01CC", Status: "active"}}
	resp := GateFailureResponse(featureID, "needs-rework", "reviewing", gate, nonTerminal)
	msg, _ := resp["error"].(string)

	if !strings.Contains(msg, "TASK-01CC") {
		t.Errorf("needs-rework→reviewing gate recovery missing task ID: %s", msg)
	}
	if !strings.Contains(msg, `finish(task_id:`) {
		t.Errorf("needs-rework→reviewing gate recovery missing finish call: %s", msg)
	}
}

func TestReviewReportGateRecovery(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01LLLL"
	gate := GateResult{Satisfied: false, Reason: "no review report document registered for this feature"}
	resp := GateFailureResponse(featureID, "reviewing", "done", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`doc(action: "register"`,
		`type: "report"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("review report gate recovery missing %q in message: %s", want, msg)
		}
	}
}

func TestReworkTaskGateRecovery(t *testing.T) {
	t.Parallel()

	featureID := "FEAT-01MMMM"
	gate := GateResult{Satisfied: false, Reason: "no non-terminal rework tasks found"}
	resp := GateFailureResponse(featureID, "needs-rework", "developing", gate, nil)
	msg, _ := resp["error"].(string)

	for _, want := range []string{
		`entity(action: "create"`,
		`type: "task"`,
		featureID,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("rework task gate recovery missing %q in message: %s", want, msg)
		}
	}
}
