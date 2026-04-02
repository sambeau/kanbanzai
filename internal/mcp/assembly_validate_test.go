package mcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// mockEntityGetter implements EntityGetter for testing.
type mockEntityGetter struct {
	status string
	err    error
}

func (m *mockEntityGetter) Get(entityType, id, slug string) (service.GetResult, error) {
	if m.err != nil {
		return service.GetResult{}, m.err
	}
	return service.GetResult{
		State: map[string]any{"status": m.status},
	}, nil
}

func TestValidateFeatureStage_WorkingStates(t *testing.T) {
	workingStates := []string{
		"designing", "specifying", "dev-planning",
		"developing", "reviewing", "needs-rework",
	}
	for _, status := range workingStates {
		mock := &mockEntityGetter{status: status}
		got, err := ValidateFeatureStage("FEAT-01TEST", mock)
		if err != nil {
			t.Errorf("status %q: unexpected error: %v", status, err)
		}
		if got != status {
			t.Errorf("status %q: got %q", status, got)
		}
	}
}

func TestValidateFeatureStage_NonWorkingStates(t *testing.T) {
	nonWorkingStates := []string{"proposed", "done", "superseded", "cancelled"}
	for _, status := range nonWorkingStates {
		mock := &mockEntityGetter{status: status}
		_, err := ValidateFeatureStage("FEAT-01TEST", mock)
		if err == nil {
			t.Errorf("status %q: expected error, got nil", status)
			continue
		}
		var valErr *StageValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("status %q: error is not StageValidationError: %T", status, err)
			continue
		}
		if valErr.FeatureID != "FEAT-01TEST" {
			t.Errorf("status %q: FeatureID = %q, want FEAT-01TEST", status, valErr.FeatureID)
		}
		if valErr.CurrentState != status {
			t.Errorf("status %q: CurrentState = %q", status, valErr.CurrentState)
		}
	}
}

func TestValidateFeatureStage_ErrorMessage(t *testing.T) {
	mock := &mockEntityGetter{status: "done"}
	_, err := ValidateFeatureStage("FEAT-01EXAMPLE", mock)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	// Must contain feature ID
	if !strings.Contains(msg, "FEAT-01EXAMPLE") {
		t.Errorf("error missing feature ID: %s", msg)
	}
	// Must contain state in quotes
	if !strings.Contains(msg, "'done'") {
		t.Errorf("error missing quoted state: %s", msg)
	}
	// Must contain tool call example
	if !strings.Contains(msg, `entity(action: "get"`) {
		t.Errorf("error missing tool call example: %s", msg)
	}
}

func TestValidateFeatureStage_EmptyParentID(t *testing.T) {
	mock := &mockEntityGetter{status: "developing"}
	status, err := ValidateFeatureStage("", mock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if status != "" {
		t.Errorf("status = %q, want empty", status)
	}
}

func TestValidateFeatureStage_FeatureNotFound(t *testing.T) {
	mock := &mockEntityGetter{err: errors.New("not found")}
	status, err := ValidateFeatureStage("FEAT-01MISSING", mock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if status != "" {
		t.Errorf("status = %q, want empty", status)
	}
}
