package service

import (
	"fmt"

	"kanbanzai/internal/model"
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
	string(model.FeatureStatusDone),
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
// The exception is "done": the "reviewing" gate is always checked before
// entering done, and it is never satisfied, so advance always stops at
// developing.
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

		// Gate-check intermediate states using their own name as the gate.
		// The target is not gate-checked, except "done" which requires the
		// "reviewing" gate (always unsatisfied).
		var shouldCheck bool
		var gate string

		if !isTarget {
			shouldCheck = true
			gate = nextState
		} else if nextState == string(model.FeatureStatusDone) {
			shouldCheck = true
			gate = "reviewing"
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
	}

	return AdvanceResult{
		FinalStatus:     string(feature.Status),
		AdvancedThrough: advancedThrough,
	}, nil
}
