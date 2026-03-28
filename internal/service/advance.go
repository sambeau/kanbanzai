package service

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/model"
)

// AdvanceResult describes the outcome of an advance operation.
type AdvanceResult struct {
	FinalStatus     string   // where the feature ended up
	AdvancedThrough []string // intermediate states passed through
	StoppedReason   string   // empty if target reached, explanation if stopped early
}

// featureForwardPath is the sequential forward path for Phase 2 features.
var featureForwardPath = []string{
	string(model.FeatureStatusProposed),
	string(model.FeatureStatusDesigning),
	string(model.FeatureStatusSpecifying),
	string(model.FeatureStatusDevPlanning),
	string(model.FeatureStatusDeveloping),
	string(model.FeatureStatusReviewing),
	string(model.FeatureStatusDone),
}

// advanceStopStates are lifecycle states where advance always halts after
// entering. These represent mandatory human/orchestrator gates that cannot
// be auto-transitioned through.
var advanceStopStates = map[string]bool{
	string(model.FeatureStatusReviewing): true,
}

// featurePathIndex returns the index of a status in the forward path, or -1 if
// the status is not part of the sequential forward path.
func featurePathIndex(status string) int {
	for i, s := range featureForwardPath {
		if s == status {
			return i
		}
	}
	return -1
}

// AdvanceFeatureStatus walks a feature through multiple lifecycle states toward
// a target status, checking document prerequisites at each intermediate gate.
//
// For each intermediate state between the feature's current status and the
// target, CheckFeatureGate is called to determine whether the prerequisite is
// satisfied. If satisfied, the feature is transitioned through that state
// (persisted at each step). If not satisfied, the advance stops and returns a
// partial result explaining why.
//
// The target state itself is not gate-checked — only intermediate states are.
// States in advanceStopStates (e.g. "reviewing") are mandatory gates: advance
// transitions into them but never auto-transitions through them. This ensures
// human/orchestrator review cannot be skipped.
//
// Backward transitions are not supported and return an error.
func AdvanceFeatureStatus(
	feature *model.Feature,
	targetStatus string,
	entitySvc *EntityService,
	docSvc *DocumentService,
) (AdvanceResult, error) {
	currentStatus := string(feature.Status)

	// No-op: already at target.
	if currentStatus == targetStatus {
		return AdvanceResult{
			FinalStatus: currentStatus,
		}, nil
	}

	currentIdx := featurePathIndex(currentStatus)
	if currentIdx < 0 {
		return AdvanceResult{}, fmt.Errorf("current status %q is not on the feature forward path", currentStatus)
	}

	targetIdx := featurePathIndex(targetStatus)
	if targetIdx < 0 {
		return AdvanceResult{}, fmt.Errorf("target status %q is not on the feature forward path", targetStatus)
	}

	if targetIdx < currentIdx {
		return AdvanceResult{}, fmt.Errorf("cannot advance backward from %q to %q", currentStatus, targetStatus)
	}

	// Build the sequence of states to transition through (current+1 … target inclusive).
	path := featureForwardPath[currentIdx+1 : targetIdx+1]

	var advancedThrough []string

	for i, nextState := range path {
		isTarget := i == len(path)-1
		isStopState := advanceStopStates[nextState]

		// Stop states (e.g. reviewing) are mandatory gates: we transition
		// into them without a prerequisite check, then halt. Non-stop
		// intermediate states are gate-checked before entry.
		if !isStopState {
			var shouldCheck bool
			var gate string

			if !isTarget {
				shouldCheck = true
				gate = nextState
			}

			if shouldCheck {
				result := CheckFeatureGate(gate, feature, docSvc, entitySvc)
				if !result.Satisfied {
					return AdvanceResult{
						FinalStatus:     string(feature.Status),
						AdvancedThrough: advancedThrough,
						StoppedReason:   fmt.Sprintf("stopped before %s: %s", nextState, result.Reason),
					}, nil
				}
			}
		}

		// Persist the state change via UpdateStatus, which validates the
		// transition against the lifecycle state machine internally.
		_, err := entitySvc.UpdateStatus(UpdateStatusInput{
			Type:   "feature",
			ID:     feature.ID,
			Slug:   feature.Slug,
			Status: nextState,
		})
		if err != nil {
			return AdvanceResult{}, fmt.Errorf("persisting transition to %s: %w", nextState, err)
		}

		// Keep in-memory feature in sync with persisted state.
		feature.Status = model.FeatureStatus(nextState)
		advancedThrough = append(advancedThrough, nextState)

		// Halt after entering a stop state (unless it was the target).
		if isStopState && !isTarget {
			return AdvanceResult{
				FinalStatus:     nextState,
				AdvancedThrough: advancedThrough,
				StoppedReason:   fmt.Sprintf("stopped at %s: review is a mandatory gate that cannot be auto-advanced", nextState),
			}, nil
		}
	}

	return AdvanceResult{
		FinalStatus:     string(feature.Status),
		AdvancedThrough: advancedThrough,
	}, nil
}
