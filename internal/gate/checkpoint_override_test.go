package gate

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/checkpoint"
)

func TestHandleCheckpointOverride_CreatesPendingCheckpoint(t *testing.T) {
	store := checkpoint.NewStore(t.TempDir())

	result, err := HandleCheckpointOverride(CheckpointOverrideParams{
		FeatureID:       "FEAT-001",
		FromStatus:      "designing",
		ToStatus:        "specifying",
		GateDescription: "design document required",
		OverrideReason:  "design exists externally",
		AgentIdentity:   "test-agent",
		CheckpointStore: store,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.CheckpointCreated {
		t.Error("expected CheckpointCreated to be true")
	}

	if !strings.HasPrefix(result.CheckpointID, "CHK-") {
		t.Errorf("expected CheckpointID to start with CHK-, got %q", result.CheckpointID)
	}
}

func TestHandleCheckpointOverride_QuestionContainsAllInfo(t *testing.T) {
	store := checkpoint.NewStore(t.TempDir())

	result, err := HandleCheckpointOverride(CheckpointOverrideParams{
		FeatureID:       "FEAT-042",
		FromStatus:      "specifying",
		ToStatus:        "dev-planning",
		GateDescription: "spec not approved",
		OverrideReason:  "spec reviewed offline",
		AgentIdentity:   "agent-x",
		CheckpointStore: store,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, err := store.Get(result.CheckpointID)
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}

	if record.Status != checkpoint.StatusPending {
		t.Errorf("expected status %q, got %q", checkpoint.StatusPending, record.Status)
	}

	for _, want := range []string{"FEAT-042", "specifying", "dev-planning", "spec not approved", "spec reviewed offline"} {
		if !strings.Contains(record.Question, want) {
			t.Errorf("question missing %q: %s", want, record.Question)
		}
	}

	for _, want := range []string{"FEAT-042", "specifying", "dev-planning", "checkpoint", "spec reviewed offline"} {
		if !strings.Contains(record.Context, want) {
			t.Errorf("context missing %q: %s", want, record.Context)
		}
	}

	if record.CreatedBy != "agent-x" {
		t.Errorf("expected created_by %q, got %q", "agent-x", record.CreatedBy)
	}
}

func TestHandleCheckpointOverride_DefaultCreatedBy(t *testing.T) {
	store := checkpoint.NewStore(t.TempDir())

	result, err := HandleCheckpointOverride(CheckpointOverrideParams{
		FeatureID:       "FEAT-001",
		FromStatus:      "designing",
		ToStatus:        "specifying",
		GateDescription: "design doc required",
		OverrideReason:  "reason",
		AgentIdentity:   "",
		CheckpointStore: store,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, err := store.Get(result.CheckpointID)
	if err != nil {
		t.Fatalf("failed to get checkpoint: %v", err)
	}

	if record.CreatedBy != "system" {
		t.Errorf("expected created_by %q when AgentIdentity is empty, got %q", "system", record.CreatedBy)
	}
}

func TestHandleCheckpointOverride_MessageContainsIDAndPollInstruction(t *testing.T) {
	store := checkpoint.NewStore(t.TempDir())

	result, err := HandleCheckpointOverride(CheckpointOverrideParams{
		FeatureID:       "FEAT-007",
		FromStatus:      "designing",
		ToStatus:        "specifying",
		GateDescription: "gate failed",
		OverrideReason:  "override reason",
		AgentIdentity:   "agent",
		CheckpointStore: store,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Message, result.CheckpointID) {
		t.Errorf("message missing checkpoint ID %q: %s", result.CheckpointID, result.Message)
	}

	if !strings.Contains(result.Message, "FEAT-007") {
		t.Errorf("message missing feature ID: %s", result.Message)
	}

	if !strings.Contains(result.Message, "checkpoint(action:") {
		t.Errorf("message missing poll instruction: %s", result.Message)
	}

	if !strings.Contains(result.Message, "checkpoint_id:") {
		t.Errorf("message missing checkpoint_id in poll instruction: %s", result.Message)
	}
}

func TestResolveCheckpointResponse_Approvals(t *testing.T) {
	approvals := []struct {
		name     string
		response string
	}{
		{"approved", "approved"},
		{"yes", "yes"},
		{"looks good to me", "looks good to me"},
		{"ok", "ok"},
		{"empty string", ""},
		{"notable contains no but not whole word", "notable"},
		{"cannot contains no but not whole word", "cannot"},
		{"I do not agree", "I do not agree"},
	}

	for _, tc := range approvals {
		t.Run(tc.name, func(t *testing.T) {
			if !ResolveCheckpointResponse(tc.response) {
				t.Errorf("expected approval for %q, got rejection", tc.response)
			}
		})
	}
}

func TestResolveCheckpointResponse_Rejections(t *testing.T) {
	rejections := []struct {
		name     string
		response string
	}{
		{"rejected", "rejected"},
		{"reject", "reject"},
		{"no", "no"},
		{"denied", "denied"},
		{"No with explanation", "No, this needs more work"},
		{"uppercase REJECTED", "REJECTED"},
		{"mixed case Denied", "Denied"},
	}

	for _, tc := range rejections {
		t.Run(tc.name, func(t *testing.T) {
			if ResolveCheckpointResponse(tc.response) {
				t.Errorf("expected rejection for %q, got approval", tc.response)
			}
		})
	}
}
