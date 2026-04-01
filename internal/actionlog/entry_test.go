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
