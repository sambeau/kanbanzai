package actionlog

import (
	"encoding/json"
	"testing"
)

func TestEntryJSONRoundTrip(t *testing.T) {
	t.Parallel()

	action := "create"
	entityID := "FEAT-001"
	stage := "developing"
	errType := ErrorGateFailure

	e := Entry{
		Timestamp: "2024-01-15T10:00:00Z",
		Tool:      "entity",
		Action:    &action,
		EntityID:  &entityID,
		Stage:     &stage,
		Success:   false,
		ErrorType: &errType,
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Entry
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Timestamp != e.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, e.Timestamp)
	}
	if got.Tool != e.Tool {
		t.Errorf("Tool: got %q, want %q", got.Tool, e.Tool)
	}
	if got.Action == nil || *got.Action != action {
		t.Errorf("Action: got %v, want %q", got.Action, action)
	}
	if got.EntityID == nil || *got.EntityID != entityID {
		t.Errorf("EntityID: got %v, want %q", got.EntityID, entityID)
	}
	if got.Stage == nil || *got.Stage != stage {
		t.Errorf("Stage: got %v, want %q", got.Stage, stage)
	}
	if got.Success != false {
		t.Errorf("Success: got %v, want false", got.Success)
	}
	if got.ErrorType == nil || *got.ErrorType != errType {
		t.Errorf("ErrorType: got %v, want %q", got.ErrorType, errType)
	}
}

func TestEntryNullFields(t *testing.T) {
	t.Parallel()

	e := Entry{
		Timestamp: "2024-01-15T10:00:00Z",
		Tool:      "status",
		Success:   true,
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	for _, field := range []string{"action", "entity_id", "stage", "error_type"} {
		v, ok := raw[field]
		if !ok {
			t.Errorf("field %q missing from JSON", field)
			continue
		}
		if v != nil {
			t.Errorf("field %q: got %v, want null", field, v)
		}
	}
}

func TestErrorTypeConstants(t *testing.T) {
	t.Parallel()

	constants := []string{
		ErrorGateFailure,
		ErrorValidationError,
		ErrorNotFound,
		ErrorPreconditionError,
		ErrorInternalError,
	}
	for _, c := range constants {
		if c == "" {
			t.Errorf("empty error type constant")
		}
	}
}

// AC-001: Old log without server_version parses with empty string.
func TestEntry_BackwardCompat_NoServerVersion(t *testing.T) {
	t.Parallel()

	// JSON from an older log (pre-instrumentation-expansion) that lacks
	// the server_version field entirely.
	oldJSON := `{"timestamp":"2025-01-01T00:00:00Z","tool":"entity","action":null,"entity_id":null,"stage":null,"success":true,"error_type":null}`

	var e Entry
	if err := json.Unmarshal([]byte(oldJSON), &e); err != nil {
		t.Fatalf("unmarshal old log without server_version: %v", err)
	}

	if e.ServerVersion != "" {
		t.Errorf("ServerVersion: got %q, want empty (AC-001)", e.ServerVersion)
	}
	if e.Extra != nil {
		t.Errorf("Extra: got %v, want nil (AC-003)", e.Extra)
	}
}

// AC-003: Old log without extra parses with nil map.
func TestEntry_BackwardCompat_NoExtra(t *testing.T) {
	t.Parallel()

	// JSON from an older log that has server_version but no extra field.
	oldJSON := `{"timestamp":"2025-01-01T00:00:00Z","tool":"entity","action":null,"entity_id":null,"stage":null,"server_version":"1.0","success":true,"error_type":null}`

	var e Entry
	if err := json.Unmarshal([]byte(oldJSON), &e); err != nil {
		t.Fatalf("unmarshal old log without extra: %v", err)
	}

	if e.ServerVersion != "1.0" {
		t.Errorf("ServerVersion: got %q, want %q", e.ServerVersion, "1.0")
	}
	if e.Extra != nil {
		t.Errorf("Extra: got %v, want nil (AC-003)", e.Extra)
	}
}
