package invariants

import "encoding/json"

// Invariant codes — stable, never renamed.
const (
	INV001 = "INV-001" // Handoff-only dispatch — no direct spawn_agent composition
	INV002 = "INV-002" // Registered-entity required — next/handoff refuse unknown IDs
	INV003 = "INV-003" // Commit before task claim — next task-claim refuses orphaned workflow state
	INV004 = "INV-004" // No shell reads of .kbz/state/ — mandatory warning on all context surfaces
	INV005 = "INV-005" // Artefact gate enforcement — gates are mandatory, never advisory
)

// RefusalResponse carries the four required fields for a canonical MCP refusal.
type RefusalResponse struct {
	Code       string // one of INV001–INV005
	Operation  string // the refused operation (e.g. "next task-claim")
	Reason     string // human-readable reason (≤ 400 bytes recommended)
	NextAction string // what the caller should do instead
}

// errorPayload is the JSON wire shape.
type errorPayload struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Operation  string `json:"operation"`
	Reason     string `json:"reason"`
	NextAction string `json:"next_action"`
}

// Format serialises r to the canonical JSON refusal shape:
//
//	{"error":{"code":"INV-002","operation":"...","reason":"...","next_action":"..."}}
//
// Total output is guaranteed to be ≤ 1,200 bytes when Reason ≤ 400 bytes.
func Format(r RefusalResponse) string {
	p := errorPayload{
		Error: errorBody{
			Code:       r.Code,
			Message:    r.Reason,
			Operation:  r.Operation,
			Reason:     r.Reason,
			NextAction: r.NextAction,
		},
	}
	b, err := json.Marshal(p)
	if err != nil {
		// Unreachable: the struct only contains strings.
		return `{"error":{"code":"` + r.Code + `","operation":"","reason":"json marshal failed","next_action":""}}`
	}
	return string(b)
}
