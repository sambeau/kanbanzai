package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/invariants"
)

// ─── INV-002: Registered-entity requirement ───────────────────────────────────

// TestInvariant_INV002_Next_UnregisteredTask verifies that next refuses with
// INV-002 when the requested task ID is not registered in Kanbanzai workflow
// state (AC-003, REQ-005, REQ-011).
func TestInvariant_INV002_Next_UnregisteredTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	// Stub the dirty-state check so it does not interfere.
	checkKbzDirtyFuncMu.RLock()
	orig := checkKbzDirtyFunc
	checkKbzDirtyFuncMu.RUnlock()
	setCheckKbzDirtyFunc(func(string) ([]string, error) { return nil, nil })
	t.Cleanup(func() { restoreCheckKbzDirtyFunc(orig) })

	resp := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": "TASK-01ZZZZZZZZZZ9",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level 'error' object, got: %v", resp)
	}
	if errObj["code"] != invariants.INV002 {
		t.Errorf("code = %v, want %q", errObj["code"], invariants.INV002)
	}
	if _, hasOp := errObj["operation"]; !hasOp {
		t.Error("refusal missing 'operation' field")
	}
	if _, hasReason := errObj["reason"]; !hasReason {
		t.Error("refusal missing 'reason' field")
	}
	if _, hasNext := errObj["next_action"]; !hasNext {
		t.Error("refusal missing 'next_action' field")
	}
}

// TestInvariant_INV002_Handoff_UnregisteredTask verifies that handoff refuses
// with INV-002 when the requested task ID is not registered.
//
// NOTE (AC-002/P44): This test is skipped until P44's dispatch_task path is
// registered. When the full handoff-only dispatch path is live, the refusal
// behaviour documented here must be re-verified against the updated pipeline.
// At that point, remove the t.Skip and update the assertion as needed.
// Tracked dependency: P44-F1 — Model Routing Agent Launcher.
func TestInvariant_INV002_Handoff_UnregisteredTask(t *testing.T) {
	t.Skip("Pending AC-002: requires P44 dispatch_task to be registered before the full handoff-only dispatch path is verifiable; remove this skip when P44 lands.")

	entitySvc := setupHandoffTest(t)

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": "TASK-01ZZZZZZZZZZ9",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level 'error' object, got: %v", resp)
	}
	if errObj["code"] != invariants.INV002 {
		t.Errorf("code = %v, want %q", errObj["code"], invariants.INV002)
	}
	if _, hasOp := errObj["operation"]; !hasOp {
		t.Error("refusal missing 'operation' field")
	}
	if _, hasReason := errObj["reason"]; !hasReason {
		t.Error("refusal missing 'reason' field")
	}
	if _, hasNext := errObj["next_action"]; !hasNext {
		t.Error("refusal missing 'next_action' field")
	}
}

// ─── INV-003: Commit-before-task invariant ────────────────────────────────────

// TestInvariant_INV003_Next_OrphanedState verifies that next refuses to claim a
// task when checkKbzDirtyFunc reports orphaned workflow-state files (AC-004,
// REQ-006, REQ-011). The dirty-file stub replaces the real git call so the test
// does not depend on the actual working tree state.
func TestInvariant_INV003_Next_OrphanedState(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "inv003-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "inv003-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "inv003-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	dirtyFiles := []string{
		".kbz/state/tasks/TASK-01DIRTYDIRTY.yaml",
		".kbz/index/documents/DOC-999.yaml",
	}

	checkKbzDirtyFuncMu.RLock()
	orig := checkKbzDirtyFunc
	checkKbzDirtyFuncMu.RUnlock()
	setCheckKbzDirtyFunc(func(string) ([]string, error) { return dirtyFiles, nil })
	t.Cleanup(func() { restoreCheckKbzDirtyFunc(orig) })

	text := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// The response text must contain the INV-003 code. The refusal is embedded
	// as the message string inside the WithSideEffects error envelope.
	if !strings.Contains(text, invariants.INV003) {
		t.Errorf("response does not contain %q\nresponse: %s", invariants.INV003, text)
	}
	// Each dirty file must be listed so the caller knows what to commit/stash.
	for _, f := range dirtyFiles {
		if !strings.Contains(text, f) {
			t.Errorf("response does not mention dirty file %q\nresponse: %s", f, text)
		}
	}
}

// ─── INV-004: No shell reads of .kbz/state/ ──────────────────────────────────

// TestInvariant_INV004_ContextWarning_Next verifies that the context map
// returned by next task-claim always includes a workflow_state_warning field
// containing the INV-004 reference text (AC-005, REQ-007).
func TestInvariant_INV004_ContextWarning_Next(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "inv004-next-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "inv004-next-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "inv004-next-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Ensure no dirty-state refusal interferes.
	checkKbzDirtyFuncMu.RLock()
	orig := checkKbzDirtyFunc
	checkKbzDirtyFuncMu.RUnlock()
	setCheckKbzDirtyFunc(func(string) ([]string, error) { return nil, nil })
	t.Cleanup(func() { restoreCheckKbzDirtyFunc(orig) })

	resp := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	ctx, ok := resp["context"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'context' map; got type %T\nresponse: %v", resp["context"], resp)
	}
	warning, ok := ctx["workflow_state_warning"].(string)
	if !ok {
		t.Fatalf("context missing 'workflow_state_warning' string field; got: %v", ctx["workflow_state_warning"])
	}
	if !strings.Contains(warning, invariants.INV004) {
		t.Errorf("workflow_state_warning = %q; want it to contain %q", warning, invariants.INV004)
	}
	if !strings.Contains(warning, ".kbz/state/") {
		t.Errorf("workflow_state_warning = %q; want it to mention .kbz/state/", warning)
	}
}

// TestInvariant_INV004_ContextWarning_Handoff verifies that the rendered handoff
// prompt always contains an ## Invariants section with the INV-004 rule text
// (AC-005, REQ-007). This covers the rendered-surface boundary for INV-004.
func TestInvariant_INV004_ContextWarning_Handoff(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)

	taskID, taskSlug := createHandoffScenario(t, entitySvc, "inv004-ho")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": taskID,
	})

	prompt, ok := resp["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		t.Fatalf("expected non-empty prompt string, got: %v", resp["prompt"])
	}
	if !strings.Contains(prompt, "## Invariants") {
		t.Errorf("rendered prompt missing '## Invariants' section\nprompt: %s", prompt)
	}
	if !strings.Contains(prompt, invariants.INV004) {
		t.Errorf("rendered prompt does not contain %q\nprompt: %s", invariants.INV004, prompt)
	}
}

// ─── AC-011: Refusal byte-size limit ─────────────────────────────────────────

// TestInvariant_RefusalSize asserts that every structured refusal response
// body produced by invariants.Format is at or below 1,200 bytes (AC-011,
// REQ-NF-002). One representative refusal is generated for each invariant code.
func TestInvariant_RefusalSize(t *testing.T) {
	t.Parallel()

	cases := []invariants.RefusalResponse{
		{
			Code:       invariants.INV001,
			Operation:  "spawn_agent direct dispatch",
			Reason:     "Direct spawn_agent composition is not permitted. Use handoff to assemble the prompt via the pipeline, then dispatch via dispatch_task.",
			NextAction: `Use handoff(task_id: "TASK-...") to generate a pipeline-assembled prompt, then dispatch_task to launch the sub-agent.`,
		},
		{
			Code:       invariants.INV002,
			Operation:  "next task-claim",
			Reason:     "Task TASK-01ZZZZZZZZZZ9 is not registered in Kanbanzai workflow state.",
			NextAction: `Create the entity with entity(action: "create") or verify the ID with entity(action: "list", type: "task").`,
		},
		{
			Code:       invariants.INV003,
			Operation:  "next task-claim",
			Reason:     "Orphaned workflow state detected. Dirty files under .kbz/: .kbz/state/tasks/TASK-01DIRTY.yaml, .kbz/index/documents/DOC-001.yaml",
			NextAction: "Commit or stash the listed files, then retry next",
		},
		{
			Code:       invariants.INV004,
			Operation:  "shell read of .kbz/state/",
			Reason:     "Direct filesystem reads of .kbz/state/, .kbz/index/, or .kbz/context/ are not permitted via terminal or shell tools.",
			NextAction: "Use MCP workflow tools (entity, doc, status, knowledge) instead of shell/terminal reads.",
		},
		{
			Code:       invariants.INV005,
			Operation:  "feature lifecycle transition",
			Reason:     "Required artefact gate not satisfied: no approved spec document registered for this feature.",
			NextAction: `Register and approve a spec with doc(action: "register") then doc(action: "approve") before advancing the feature.`,
		},
	}

	const maxBytes = 1200
	for _, c := range cases {
		body := invariants.Format(c)
		if len(body) > maxBytes {
			t.Errorf("invariant %s refusal body = %d bytes, want <= %d\nbody: %s",
				c.Code, len(body), maxBytes, body)
		}
		// Also verify the output is valid JSON with the four required fields.
		var parsed struct {
			Error struct {
				Code       string `json:"code"`
				Operation  string `json:"operation"`
				Reason     string `json:"reason"`
				NextAction string `json:"next_action"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(body), &parsed); err != nil {
			t.Errorf("invariant %s: Format output is not valid JSON: %v\nbody: %s",
				c.Code, err, body)
			continue
		}
		if parsed.Error.Code != c.Code {
			t.Errorf("invariant %s: code round-trip: got %q, want %q", c.Code, parsed.Error.Code, c.Code)
		}
		if parsed.Error.Operation == "" {
			t.Errorf("invariant %s: 'operation' field is empty", c.Code)
		}
		if parsed.Error.Reason == "" {
			t.Errorf("invariant %s: 'reason' field is empty", c.Code)
		}
		if parsed.Error.NextAction == "" {
			t.Errorf("invariant %s: 'next_action' field is empty", c.Code)
		}
	}
}
