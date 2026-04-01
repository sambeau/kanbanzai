package service

import (
	"fmt"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
)

// AdvanceResult describes the outcome of an advance operation.
type AdvanceResult struct {
	FinalStatus     string   // where the feature ended up
	AdvancedThrough []string // intermediate states passed through
	StoppedReason   string   // empty if target reached, explanation if stopped early
	OverriddenGates []string // transitions where a gate was bypassed via override (FR-016)
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
// a target status, enforcing gate prerequisites at every step via
// CheckTransitionGate — including the target state itself (FR-001).
//
// When override is true and overrideReason is a non-empty string, any failing
// gate is bypassed: an OverrideRecord is appended to the feature, persisted to
// disk immediately, and the advance continues (FR-016). Each bypassed gate is
// recorded as a separate entry.
//
// States in advanceStopStates (e.g. "reviewing") are mandatory halt points:
// advance transitions into them and then stops, even if the target lies beyond.
// This behaviour is preserved regardless of gate override (NFR-005).
//
// Backward transitions are not supported and return an error.
func AdvanceFeatureStatus(
	feature *model.Feature,
	targetStatus string,
	entitySvc *EntityService,
	docSvc *DocumentService,
	override bool,
	overrideReason string,
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
	var overriddenGates []string

	for i, nextState := range path {
		isTarget := i == len(path)-1
		fromState := string(feature.Status)

		// Evaluate the gate for this transition on every step (FR-001).
		gateResult := CheckTransitionGate(fromState, nextState, feature, docSvc, entitySvc)
		if !gateResult.Satisfied {
			if !override {
				// Gate failed with no override: halt here, before transitioning.
				return AdvanceResult{
					FinalStatus:     fromState,
					AdvancedThrough: advancedThrough,
					OverriddenGates: overriddenGates,
					StoppedReason:   fmt.Sprintf("stopped before %s: %s", nextState, gateResult.Reason),
				}, nil
			}

			// Gate failed with override: record the bypass and continue (FR-016).
			or := model.OverrideRecord{
				FromStatus: fromState,
				ToStatus:   nextState,
				Reason:     overrideReason,
				Timestamp:  time.Now(),
			}
			feature.Overrides = append(feature.Overrides, or)
			if err := entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides); err != nil {
				return AdvanceResult{}, fmt.Errorf("persisting override record for %s→%s: %w", fromState, nextState, err)
			}
			overriddenGates = append(overriddenGates, fromState+"→"+nextState)
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

		// Halt after entering a stop state (unless it was the explicit target).
		if advanceStopStates[nextState] && !isTarget {
			return AdvanceResult{
				FinalStatus:     nextState,
				AdvancedThrough: advancedThrough,
				OverriddenGates: overriddenGates,
				StoppedReason:   fmt.Sprintf("stopped at %s: review is a mandatory gate that cannot be auto-advanced", nextState),
			}, nil
		}
	}

	return AdvanceResult{
		FinalStatus:     string(feature.Status),
		AdvancedThrough: advancedThrough,
		OverriddenGates: overriddenGates,
	}, nil
}
