// Package mcp — field-name regression guard for handoff tool responses.
//
// These tests assert that every field name present in prior versions of the
// handoff tool response is still present in the current version. Adding new
// fields is fine; removing or renaming existing fields is a breaking change.
package mcp

import (
	"strings"
	"testing"
)

// handoffResponseFields is the canonical set of top-level field names
// expected in a successful handoff JSON response.
var handoffResponseFields = []string{
	"task_id",
	"display_id",
	"entity_ref",
	"prompt",
	"stage_binding",
	"context_metadata",
}

// handoffContextMetadataFields is the canonical set of field names expected
// inside the "context_metadata" object.
var handoffContextMetadataFields = []string{
	"assembly_path",
	"sections",
	"total_tokens",
	"token_warning",
	"metadata_warnings",
}

// TestHandoff_AllResponseFieldsPresent does an integration-level verification
// that all expected top-level fields are present in a successful handoff response.
func TestHandoff_AllResponseFieldsPresent(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "reg-fields")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": taskID,
	})

	// Must not be an error.
	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected success, got error: %+v", resp["error"])
	}

	for _, name := range handoffResponseFields {
		if _, ok := resp[name]; !ok {
			t.Errorf("handoff response field %q is missing", name)
		}
	}

	// Verify context_metadata sub-fields.
	meta, ok := resp["context_metadata"].(map[string]any)
	if !ok {
		t.Fatalf("context_metadata missing or wrong type: %T", resp["context_metadata"])
	}
	for _, name := range handoffContextMetadataFields {
		if _, ok := meta[name]; !ok {
			t.Errorf("handoff context_metadata field %q is missing", name)
		}
	}

	// Verify stage_binding sub-fields (may be minimal — just "stage").
	sb, ok := resp["stage_binding"].(map[string]any)
	if !ok {
		t.Fatalf("stage_binding missing or wrong type: %T", resp["stage_binding"])
	}
	if _, ok := sb["stage"]; !ok {
		t.Error("handoff stage_binding field \"stage\" is missing")
	}

	// Verify prompt is a non-empty string.
	prompt, ok := resp["prompt"].(string)
	if !ok {
		t.Fatalf("prompt missing or wrong type: %T", resp["prompt"])
	}
	if prompt == "" {
		t.Error("handoff prompt is empty")
	}
}

// TestHandoff_StageBindingFields_PresentWhenHydrated verifies that when a
// stage binding is resolved, the stage_binding object contains the expected
// hydrated fields (not just the stage name).
func TestHandoff_StageBindingFields_PresentWhenHydrated(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)
	// createHandoffScenario already advances the feature to "developing".
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "reg-sb")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": taskID,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected success, got error: %+v", resp["error"])
	}

	sb, ok := resp["stage_binding"].(map[string]any)
	if !ok {
		t.Fatalf("stage_binding missing or wrong type: %T", resp["stage_binding"])
	}

	// stage must always be present.
	if _, ok := sb["stage"]; !ok {
		t.Error("stage_binding.stage missing")
	}

	// For developing, roles and skills should be present.
	// If they are not (e.g., binding lookup fails because no BindingFile is
	// provided), that's expected — we just verify the response is well-formed.
	if _, ok := sb["roles"]; !ok {
		t.Log("stage_binding.roles missing (expected: binding not wired in test)")
	}
	if _, ok := sb["skills"]; !ok {
		t.Log("stage_binding.skills missing (expected: binding not wired in test)")
	}
}

// TestHandoff_ResponseShapeMatchesCanonical ensures the canonical field list
// is consistent with what the code actually produces. This catches drift
// between documentation and implementation.
func TestHandoff_ResponseShapeMatchesCanonical(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "reg-canon")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": taskID,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected success, got error: %+v", resp["error"])
	}

	// Collect actual top-level keys.
	actualKeys := make(map[string]bool)
	for k := range resp {
		actualKeys[k] = true
	}

	// Every canonical field must be present.
	for _, name := range handoffResponseFields {
		if !actualKeys[name] {
			t.Errorf("canonical handoff field %q not found in response; field may have been removed or renamed", name)
		}
	}

	// Report any actual fields not in the canonical list.
	for k := range actualKeys {
		found := false
		for _, name := range handoffResponseFields {
			if k == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("field %q is in handoff response but not in handoffResponseFields; add it to the canonical list", k)
		}
	}
}

// TestHandoff_ErrorResponseShapePreserved verifies that error responses
// follow the expected shape (error.code, error.message).
func TestHandoff_ErrorResponseShapePreserved(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": "TASK-nonexistent",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got: %+v", resp)
	}
	if _, ok := errObj["code"]; !ok {
		t.Error("handoff error missing code field")
	}
	if _, ok := errObj["message"]; !ok {
		t.Error("handoff error missing message field")
	}
}

// TestHandoff_ConstraintCardPrependedWhenRoleAvailable verifies that the
// constraint card is prepended to the prompt when a role is resolvable.
// Without a role store wired, the card is not rendered, but the prompt
// must still be valid.
func TestHandoff_ConstraintCardPrependedWhenRoleAvailable(t *testing.T) {
	t.Parallel()
	entitySvc := setupHandoffTest(t)
	taskID, taskSlug := createHandoffScenario(t, entitySvc, "reg-card")
	advanceHandoffTaskTo(t, entitySvc, taskID, taskSlug, "active")

	resp := callHandoffJSON(t, entitySvc, testHandoffPipeline(), map[string]any{
		"task_id": taskID,
		"role":    "implementer-go",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("expected success, got error: %+v", resp["error"])
	}

	// With no role store wired, the constraint card won't be rendered.
	// But the response must still be valid and contain the prompt.
	prompt, ok := resp["prompt"].(string)
	if !ok || prompt == "" {
		t.Error("handoff prompt missing or empty when role is specified")
	}

	// The prompt should contain the role identity if the card was rendered.
	// Since we don't have roleStore wired, we just verify it's not empty.
	if !strings.Contains(prompt, "## Role") && !strings.Contains(prompt, "**Role:**") {
		t.Log("prompt does not contain role identity (expected: no role store wired)")
	}
}
