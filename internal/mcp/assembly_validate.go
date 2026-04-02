package mcp

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/stage"
)

// EntityGetter is the subset of service.EntityService needed for stage validation.
type EntityGetter interface {
	Get(entityType, id, slug string) (service.GetResult, error)
}

// StageValidationError carries the data needed for the actionable error template.
type StageValidationError struct {
	FeatureID    string
	CurrentState string
}

func (e *StageValidationError) Error() string {
	return fmt.Sprintf(
		"parent feature %s is in '%s', which is not an active working state.\n\n"+
			"To resolve:\n"+
			"1. Check feature status: entity(action: \"get\", id: \"%s\")\n"+
			"2. Advance the feature to a working state (designing, specifying, dev-planning,\n"+
			"   developing, or reviewing) before dispatching tasks.",
		e.FeatureID, e.CurrentState, e.FeatureID,
	)
}

// ValidateFeatureStage checks that the parent feature is in a working state.
//
// Returns:
//   - featureStatus: the resolved feature status, for use by stage-aware assembly
//   - err: nil if valid; StageValidationError if the feature is in a non-working state
//
// When parentFeatureID is empty or the feature cannot be loaded, returns
// ("", nil) to signal graceful degradation (FR-014). The caller should
// proceed with non-stage-aware assembly.
func ValidateFeatureStage(parentFeatureID string, entitySvc EntityGetter) (string, error) {
	if parentFeatureID == "" {
		return "", nil
	}

	result, err := entitySvc.Get("feature", parentFeatureID, "")
	if err != nil {
		// Feature not found — graceful degradation (FR-014).
		return "", nil
	}

	status, _ := result.State["status"].(string)
	if status == "" {
		return "", nil
	}

	if !stage.IsWorkingState(status) {
		return "", &StageValidationError{
			FeatureID:    parentFeatureID,
			CurrentState: status,
		}
	}

	return status, nil
}
