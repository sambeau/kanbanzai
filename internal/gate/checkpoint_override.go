package gate

import (
	"fmt"
	"regexp"
	"time"

	"github.com/sambeau/kanbanzai/internal/checkpoint"
)

// CheckpointOverrideParams contains the parameters for creating a checkpoint override.
type CheckpointOverrideParams struct {
	FeatureID       string
	FromStatus      string
	ToStatus        string
	GateDescription string
	OverrideReason  string
	AgentIdentity   string
	CheckpointStore *checkpoint.Store
}

// CheckpointOverrideResult contains the outcome of a checkpoint override request.
type CheckpointOverrideResult struct {
	CheckpointCreated bool
	CheckpointID      string
	Message           string
}

// rejectionPattern matches whole-word occurrences of rejection keywords.
var rejectionPattern = regexp.MustCompile(`(?i)\b(reject|rejected|denied|no)\b`)

// HandleCheckpointOverride creates a pending checkpoint that blocks a feature
// transition until a human approves or rejects the override.
func HandleCheckpointOverride(params CheckpointOverrideParams) (CheckpointOverrideResult, error) {
	createdBy := params.AgentIdentity
	if createdBy == "" {
		createdBy = "system"
	}

	question := fmt.Sprintf(
		"Gate override requires human approval: Feature %s transition %s→%s. Gate failed: %s. Agent override reason: %s. Approve or reject this override.",
		params.FeatureID, params.FromStatus, params.ToStatus,
		params.GateDescription, params.OverrideReason,
	)

	context := fmt.Sprintf(
		"Feature: %s | Transition: %s→%s | Policy: checkpoint | Override reason: %s",
		params.FeatureID, params.FromStatus, params.ToStatus,
		params.OverrideReason,
	)

	record, err := params.CheckpointStore.Create(checkpoint.Record{
		Question:  question,
		Context:   context,
		Status:    checkpoint.StatusPending,
		CreatedAt: time.Now().UTC(),
		CreatedBy: createdBy,
	})
	if err != nil {
		return CheckpointOverrideResult{}, fmt.Errorf("create checkpoint: %w", err)
	}

	message := fmt.Sprintf(
		"Checkpoint %s created. Feature %s is blocked until checkpoint is resolved. Poll with: checkpoint(action: \"get\", checkpoint_id: \"%s\")",
		record.ID, params.FeatureID, record.ID,
	)

	return CheckpointOverrideResult{
		CheckpointCreated: true,
		CheckpointID:      record.ID,
		Message:           message,
	}, nil
}

// ResolveCheckpointResponse determines whether a checkpoint response is an
// approval or rejection. Responses containing "reject", "rejected", "denied",
// or "no" as whole words (case-insensitive) are treated as rejections. All
// other responses are treated as approvals.
func ResolveCheckpointResponse(response string) bool {
	return !rejectionPattern.MatchString(response)
}
