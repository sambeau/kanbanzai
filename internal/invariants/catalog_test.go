package invariants_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/invariants"
)

// TestFormat_RoundTrip verifies that Format produces valid JSON containing
// all four required fields with the correct values.
func TestFormat_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		r    invariants.RefusalResponse
	}{
		{
			name: "INV-001 handoff-only dispatch",
			r: invariants.RefusalResponse{
				Code:       invariants.INV001,
				Operation:  "spawn_agent",
				Reason:     "Direct spawn_agent dispatch is not permitted. Use the canonical dispatch path.",
				NextAction: "Use handoff to assemble the sub-agent prompt, then dispatch via dispatch_task.",
			},
		},
		{
			name: "INV-002 registered-entity required",
			r: invariants.RefusalResponse{
				Code:       invariants.INV002,
				Operation:  "next task-claim",
				Reason:     "Task TASK-01ABCDEF12345 is not registered in Kanbanzai workflow state.",
				NextAction: `Create the entity with entity(action: "create") or verify the ID.`,
			},
		},
		{
			name: "INV-003 commit before task claim",
			r: invariants.RefusalResponse{
				Code:       invariants.INV003,
				Operation:  "next task-claim",
				Reason:     "Orphaned workflow state: .kbz/state/tasks/TASK-xxx.yaml, .kbz/index/graph.yaml",
				NextAction: "Commit or stash the listed files, then retry next.",
			},
		},
		{
			name: "INV-004 no shell reads of .kbz",
			r: invariants.RefusalResponse{
				Code:       invariants.INV004,
				Operation:  "context assembly",
				Reason:     "Shell reads of .kbz/state/ are not permitted.",
				NextAction: "Use MCP workflow tools (entity, doc, status, knowledge) instead.",
			},
		},
		{
			name: "INV-005 artefact gate enforcement",
			r: invariants.RefusalResponse{
				Code:       invariants.INV005,
				Operation:  "feature transition",
				Reason:     "Required artefact gate not satisfied: dev-plan not approved.",
				NextAction: "Approve the dev-plan with doc(action: \"approve\") before advancing.",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := invariants.Format(tc.r)

			// Must be valid JSON.
			var parsed map[string]any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("Format output is not valid JSON: %v\noutput: %s", err, got)
			}

			// Must have top-level "error" key.
			errVal, ok := parsed["error"]
			if !ok {
				t.Fatalf("missing top-level 'error' key in: %s", got)
			}
			errMap, ok := errVal.(map[string]any)
			if !ok {
				t.Fatalf("'error' is not an object in: %s", got)
			}

			// Round-trip: all four fields must match.
			for field, want := range map[string]string{
				"code":        tc.r.Code,
				"operation":   tc.r.Operation,
				"reason":      tc.r.Reason,
				"next_action": tc.r.NextAction,
			} {
				got, ok := errMap[field].(string)
				if !ok {
					t.Errorf("field %q missing or not a string", field)
					continue
				}
				if got != want {
					t.Errorf("field %q: got %q, want %q", field, got, want)
				}
			}
		})
	}
}

// TestFormat_ByteLengthUpperBound verifies that no refusal response body
// exceeds 1,200 bytes (REQ-NF-002 / AC-011).
func TestFormat_ByteLengthUpperBound(t *testing.T) {
	// Construct a response with a Reason at the recommended 400-byte limit
	// to prove the total stays within 1,200 bytes even at that boundary.
	reason400 := strings.Repeat("x", 400)
	nextAction200 := strings.Repeat("y", 200)

	cases := []invariants.RefusalResponse{
		{Code: invariants.INV001, Operation: "spawn_agent", Reason: reason400, NextAction: nextAction200},
		{Code: invariants.INV002, Operation: "next task-claim", Reason: reason400, NextAction: nextAction200},
		{Code: invariants.INV003, Operation: "next task-claim", Reason: reason400, NextAction: nextAction200},
		{Code: invariants.INV004, Operation: "context assembly", Reason: reason400, NextAction: nextAction200},
		{Code: invariants.INV005, Operation: "feature transition", Reason: reason400, NextAction: nextAction200},
	}

	const maxBytes = 1200
	for _, r := range cases {
		result := invariants.Format(r)
		if n := len(result); n > maxBytes {
			t.Errorf("Format(%q) = %d bytes, exceeds %d-byte limit", r.Code, n, maxBytes)
		}
	}
}

// TestInvariantCodes verifies the five stable code constants are exactly as specified.
func TestInvariantCodes(t *testing.T) {
	codes := map[string]string{
		"INV001": invariants.INV001,
		"INV002": invariants.INV002,
		"INV003": invariants.INV003,
		"INV004": invariants.INV004,
		"INV005": invariants.INV005,
	}
	want := map[string]string{
		"INV001": "INV-001",
		"INV002": "INV-002",
		"INV003": "INV-003",
		"INV004": "INV-004",
		"INV005": "INV-005",
	}
	for name, got := range codes {
		if got != want[name] {
			t.Errorf("constant %s = %q, want %q", name, got, want[name])
		}
	}
}
