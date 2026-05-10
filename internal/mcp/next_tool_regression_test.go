// Package mcp — field-name regression guard for next tool responses.
//
// These tests assert that every field name present in prior versions of the
// next tool response is still present in the current version. Adding new
// fields is fine; removing or renaming existing fields is a breaking change.
package mcp

import (
	"encoding/json"
	"testing"
)

// nextContextAlwaysFields lists fields that are always present in the context
// map returned by nextContextToMap regardless of input values.
var nextContextAlwaysFields = []string{
	"spec_sections",
	"acceptance_criteria",
	"knowledge",
	"files_context",
	"constraints",
	"byte_usage",
	"byte_budget",
	"trimmed",
	"stage_aware",
	"graph_project",
	"workflow_state_warning",
}

// nextContextOptionalFields lists fields that appear only when the
// assembledContext has non-zero/non-empty values for them.
var nextContextOptionalFields = []string{
	"role_profile",
	"spec_fallback_path",
	"active_experiments",
	"feature_stage",
	"orchestration_pattern",
	"effort_budget",
	"tool_subset",
	"output_convention",
	"review_rubric",
	"test_expectations",
	"impl_guidance",
	"plan_guidance",
	"tool_hint",
	"worktree_path",
}

// nextContextAllFields is the combined set for meta-test validation.
var nextContextAllFields []string

func init() {
	nextContextAllFields = append(nextContextAllFields, nextContextAlwaysFields...)
	nextContextAllFields = append(nextContextAllFields, nextContextOptionalFields...)
}

// nextTopLevelFields is the canonical set of top-level field names expected
// in a successful next claim-mode response.
var nextTopLevelFields = []string{
	"task",
	"context",
	"stage_binding",
}

// nextTaskFields is the canonical set of field names expected inside the
// "task" object in a next claim-mode response.
var nextTaskFields = []string{
	"id",
	"display_id",
	"slug",
	"summary",
	"status",
	"parent_feature",
}

// TestNextContextToMap_AlwaysFieldsPresent asserts that always-present fields
// are in the output even with a minimal assembledContext.
func TestNextContextToMap_AlwaysFieldsPresent(t *testing.T) {
	actx := assembledContext{
		specSections: []asmSpecSection{},
		knowledge:    []asmKnowledgeEntry{},
		filesContext: []asmFileEntry{},
		trimmed:      []asmTrimmedEntry{},
		constraints:  nil,
		byteUsage:    0,
		byteBudget:   0,
		stageAware:   false,
	}

	out := nextContextToMap(actx)

	for _, name := range nextContextAlwaysFields {
		if _, ok := out[name]; !ok {
			t.Errorf("next context field %q is missing from nextContextToMap output", name)
		}
	}
}

// TestNextContextToMap_OptionalFieldsPresentWhenSet asserts that optional
// fields appear when the assembledContext has values for them.
func TestNextContextToMap_OptionalFieldsPresentWhenSet(t *testing.T) {
	actx := assembledContext{
		specSections:     []asmSpecSection{},
		knowledge:        []asmKnowledgeEntry{},
		filesContext:     []asmFileEntry{},
		trimmed:          []asmTrimmedEntry{},
		stageAware:       true,
		featureStage:     "developing",
		experimentNudge:  []asmExperimentNudge{{decisionID: "D-001", summary: "test"}},
		roleProfile:      "test-role",
		specFallbackPath: "test/path",
		reviewRubricText: "test rubric",
		testExpectText:   "test expectations",
		implGuidanceText: "test impl guidance",
		planGuidanceText: "test plan guidance",
		toolHint:         "test tool hint",
		graphProject:     "test-project",
		worktreePath:     "test/worktree",
	}

	out := nextContextToMap(actx)

	// All optional fields should be present when set.
	for _, name := range nextContextOptionalFields {
		if _, ok := out[name]; !ok {
			t.Errorf("next context optional field %q missing even when set", name)
		}
	}
}

// TestNextContextToMap_OptionalFieldsAbsentWhenEmpty asserts that optional
// fields are absent when zero/empty.
func TestNextContextToMap_OptionalFieldsAbsentWhenEmpty(t *testing.T) {
	actx := assembledContext{
		specSections: []asmSpecSection{},
		knowledge:    []asmKnowledgeEntry{},
		filesContext: []asmFileEntry{},
		trimmed:      []asmTrimmedEntry{},
		constraints:  nil,
		byteUsage:    0,
		byteBudget:   0,
		stageAware:   false,
	}

	out := nextContextToMap(actx)

	absentWhenEmpty := []string{
		"role_profile", "spec_fallback_path", "active_experiments",
		"review_rubric", "test_expectations", "impl_guidance",
		"plan_guidance", "tool_hint", "worktree_path",
	}
	for _, name := range absentWhenEmpty {
		if _, ok := out[name]; ok {
			t.Errorf("next context field %q should be absent when empty, but is present", name)
		}
	}
}

// TestNextClaimMode_AllTopLevelFieldsPresent does an integration-level
// verification that all expected top-level fields are present in a successful
// next claim response.
func TestNextClaimMode_AllTopLevelFieldsPresent(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "reg-top-level")
	featID := createNextTestFeature(t, entitySvc, planID, "reg-top-level")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "reg-top-level")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse next result: %v\nraw: %s", err, raw)
	}

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("expected success, got error: %s", raw)
	}

	for _, name := range nextTopLevelFields {
		if _, ok := result[name]; !ok {
			t.Errorf("next top-level field %q is missing from response", name)
		}
	}

	// Verify task sub-fields.
	task, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatal("task field missing or wrong type")
	}
	for _, name := range nextTaskFields {
		if _, ok := task[name]; !ok {
			t.Errorf("next task field %q is missing", name)
		}
	}

	// parent_feature sub-fields.
	pf, ok := task["parent_feature"].(map[string]any)
	if !ok {
		t.Fatal("parent_feature missing or wrong type")
	}
	for _, name := range []string{"id", "display_id", "slug", "plan_id"} {
		if _, ok := pf[name]; !ok {
			t.Errorf("next parent_feature field %q is missing", name)
		}
	}
}

// TestNextContextToMap_FieldNamesMatchCanonicalList is a meta-test that ensures
// the canonical field name list is consistent with what nextContextToMap
// actually produces — catching drift between the list and the code.
func TestNextContextToMap_FieldNamesMatchCanonicalList(t *testing.T) {
	actx := assembledContext{
		specSections:     []asmSpecSection{},
		knowledge:        []asmKnowledgeEntry{},
		filesContext:     []asmFileEntry{},
		trimmed:          []asmTrimmedEntry{},
		stageAware:       true,
		featureStage:     "developing",
		experimentNudge:  []asmExperimentNudge{{decisionID: "D-001", summary: "test"}},
		roleProfile:      "test-role",
		specFallbackPath: "test/path",
		reviewRubricText: "test rubric",
		testExpectText:   "test expectations",
		implGuidanceText: "test impl guidance",
		planGuidanceText: "test plan guidance",
		toolHint:         "test tool hint",
		graphProject:     "test-project",
		worktreePath:     "test/worktree",
	}

	out := nextContextToMap(actx)

	// Collect all keys from the actual output.
	actualKeys := make(map[string]bool)
	for k := range out {
		actualKeys[k] = true
	}

	// Every canonical field must appear in the actual output (all are set above).
	for _, name := range nextContextAllFields {
		if !actualKeys[name] {
			t.Errorf("canonical field %q not found in nextContextToMap output; field may have been removed or renamed", name)
		}
	}

	// Report any actual fields not in the canonical list (new fields that need
	// to be added to nextContextAllFields).
	for k := range actualKeys {
		found := false
		for _, name := range nextContextAllFields {
			if k == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("field %q is in nextContextToMap output but not in nextContextAllFields; add it to the canonical list", k)
		}
	}
}

// TestNextTool_QueueModeFieldsPreserved verifies that the queue mode response
// fields are preserved.
func TestNextTool_QueueModeFieldsPreserved(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	queueFields := []string{"queue", "promoted_count", "total_queued"}
	for _, name := range queueFields {
		if _, ok := result[name]; !ok {
			t.Errorf("next queue mode field %q is missing", name)
		}
	}
}
