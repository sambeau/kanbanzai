package mcp

import (
	"strings"
	"testing"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
)

// TestConstraintPinning_Next_OrchestratorRole tests that the orchestrator role
// reminder appears in constraints when next is called with role "orchestrator".
func TestConstraintPinning_Next_OrchestratorRole(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "cp-orch")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-cp-orch")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-cp-orch")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id":   taskID,
		"role": "orchestrator",
	})

	ctxOut, ok := result["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'context' object in response")
	}

	constraints, ok := ctxOut["constraints"].([]any)
	if !ok {
		t.Fatalf("constraints field missing or wrong type: %T", ctxOut["constraints"])
	}

	foundReminder := false
	for _, c := range constraints {
		entry, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if entry["type"] == "role_reminder" {
			foundReminder = true
			content, _ := entry["content"].(string)
			if !strings.Contains(content, "You are the orchestrator") {
				t.Errorf("role_reminder content does not contain expected text: %q", content)
			}
			break
		}
	}

	if !foundReminder {
		t.Error("constraints should contain a role_reminder entry when role is orchestrator")
	}
}

// TestConstraintPinning_Next_NonOrchestratorRole tests that the orchestrator role
// reminder does NOT appear in constraints for non-orchestrator roles.
func TestConstraintPinning_Next_NonOrchestratorRole(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "cp-impl")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-cp-impl")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-cp-impl")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id":   taskID,
		"role": "implementer",
	})

	ctxOut, ok := result["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'context' object in response")
	}

	constraints, _ := ctxOut["constraints"].([]any)
	if constraints == nil {
		// constraints may be nil when no conventions are loaded (valid).
		// The key assertion is that no role_reminder exists.
		return
	}

	for _, c := range constraints {
		entry, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if entry["type"] == "role_reminder" {
			t.Error("constraints should NOT contain a role_reminder entry for non-orchestrator role")
			break
		}
	}
}

// TestConstraintPinning_Next_EveryResponse tests that the orchestrator role
// reminder appears in every response, not just the first — confirming it is
// stateless and unconditional.
func TestConstraintPinning_Next_EveryResponse(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "cp-every")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-cp-every")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-cp-every")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	for i := 0; i < 3; i++ {
		result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
			"id":   taskID,
			"role": "orchestrator",
		})

		ctxOut, ok := result["context"].(map[string]any)
		if !ok {
			t.Fatalf("call %d: expected 'context' object", i+1)
		}

		constraints, ok := ctxOut["constraints"].([]any)
		if !ok {
			t.Fatalf("call %d: constraints field missing", i+1)
		}

		foundReminder := false
		for _, c := range constraints {
			entry, ok := c.(map[string]any)
			if !ok {
				continue
			}
			if entry["type"] == "role_reminder" {
				foundReminder = true
				break
			}
		}

		if !foundReminder {
			t.Errorf("call %d: constraints should contain a role_reminder entry", i+1)
		}
	}
}

// TestConstraintPinning_OrchestratorRoleReminder_Constant tests that the
// OrchestratorRoleReminder constant is defined with the correct content.
func TestConstraintPinning_OrchestratorRoleReminder_Constant(t *testing.T) {
	t.Parallel()

	const reminder = kbzctx.OrchestratorRoleReminder
	if !strings.Contains(reminder, "orchestrator") {
		t.Error("OrchestratorRoleReminder should contain 'orchestrator'")
	}
	if !strings.Contains(reminder, "coordinate") {
		t.Error("OrchestratorRoleReminder should contain 'coordinate'")
	}
	if !strings.Contains(reminder, "Do not investigate") {
		t.Error("OrchestratorRoleReminder should contain 'Do not investigate'")
	}
	if !strings.Contains(reminder, "handoff") {
		t.Error("OrchestratorRoleReminder should contain 'handoff'")
	}
}
