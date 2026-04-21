package service

import (
	"fmt"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/structural"
)

// AdvanceResult describes the outcome of an advance operation.
type AdvanceResult struct {
	FinalStatus      string                   // where the feature ended up
	AdvancedThrough  []string                 // intermediate states passed through
	StoppedReason    string                   // empty if target reached, explanation if stopped early
	OverriddenGates  []string                 // transitions where a gate was bypassed via override (FR-016)
	StructuralChecks []structural.CheckResult // accumulated from all gate checks
	CheckpointGate   string                   // stage name where a checkpoint was created (empty if none)
	CheckpointID     string                   // checkpoint ID if a checkpoint-policy gate halted advance
}

// GateCheckFunc checks the gate for a specific transition and returns the result.
// The Source field on the returned GateResult indicates whether the gate was
// evaluated from the registry or from hardcoded logic.
type GateCheckFunc func(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult

// OverridePolicyFunc returns the override policy ("agent" or "checkpoint") for a target stage.
type OverridePolicyFunc func(to string) string

// CheckpointCreateFunc creates a checkpoint for a gate override with checkpoint policy.
// Returns the checkpoint ID or an error.
type CheckpointCreateFunc func(featureID, fromStatus, toStatus, gateReason, overrideReason string) (checkpointID string, err error)

// AdvanceConfig holds optional gate-routing dependencies for AdvanceFeatureStatus.
// When nil or when individual fields are nil, the advance function uses defaults:
// CheckGate defaults to CheckTransitionGate, OverridePolicy defaults to "agent",
// and OnCheckpoint defaults to nil (no checkpoint support).
type AdvanceConfig struct {
	CheckGate           GateCheckFunc        // nil → uses CheckTransitionGate
	OverridePolicy      OverridePolicyFunc   // nil → always returns "agent"
	OnCheckpoint        CheckpointCreateFunc // nil → no checkpoint support (agent override only)
	RequiresHumanReview func() bool          // nil → false (no human review required)
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
// When cfg is non-nil and provides a gate checker, override policy, and
// checkpoint handler, the advance respects per-gate override policies:
//   - "agent" policy: override proceeds immediately (existing behaviour).
//   - "checkpoint" policy: a checkpoint is created and the advance halts at
//     that gate. The feature does NOT transition past the checkpoint gate.
//     Gates with "agent" policy earlier in the path are overridden normally.
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
	cfg *AdvanceConfig,
) (AdvanceResult, error) {
	currentStatus := string(feature.Status)

	// Resolve effective gate functions from config.
	checkGate := resolveCheckGate(cfg)
	overridePolicy := resolveOverridePolicy(cfg)
	onCheckpoint := resolveOnCheckpoint(cfg)

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
	var structuralChecks []structural.CheckResult

	for i, nextState := range path {
		isTarget := i == len(path)-1
		fromState := string(feature.Status)

		// Evaluate the gate for this transition on every step (FR-001).
		gateResult := checkGate(fromState, nextState, feature, docSvc, entitySvc)
		// Collect structural checks regardless of gate outcome.
		structuralChecks = append(structuralChecks, gateResult.StructuralChecks...)
		if !gateResult.Satisfied {
			if !override {
				// Gate failed with no override: halt here, before transitioning.
				return AdvanceResult{
					FinalStatus:      fromState,
					AdvancedThrough:  advancedThrough,
					OverriddenGates:  overriddenGates,
					StoppedReason:    fmt.Sprintf("stopped before %s: %s", nextState, gateResult.Reason),
					StructuralChecks: structuralChecks,
				}, nil
			}

			// Check the override policy for this gate.
			policy := overridePolicy(nextState)

			if policy == "checkpoint" && onCheckpoint != nil {
				// Checkpoint policy: create a checkpoint and halt the advance.
				chkID, err := onCheckpoint(feature.ID, fromState, nextState, gateResult.Reason, overrideReason)
				if err != nil {
					return AdvanceResult{}, fmt.Errorf("creating checkpoint for %s→%s: %w", fromState, nextState, err)
				}

				// Record the override with checkpoint ID.
				or := model.OverrideRecord{
					FromStatus:   fromState,
					ToStatus:     nextState,
					Reason:       overrideReason,
					Timestamp:    time.Now(),
					CheckpointID: chkID,
				}
				feature.Overrides = append(feature.Overrides, or)
				if err := entitySvc.PersistFeatureOverrides(feature.ID, feature.Slug, feature.Overrides); err != nil {
					return AdvanceResult{}, fmt.Errorf("persisting override record for %s→%s: %w", fromState, nextState, err)
				}

				return AdvanceResult{
					FinalStatus:      fromState,
					AdvancedThrough:  advancedThrough,
					OverriddenGates:  overriddenGates,
					StoppedReason:    fmt.Sprintf("stopped before %s: checkpoint override required (checkpoint %s created)", nextState, chkID),
					StructuralChecks: structuralChecks,
					CheckpointGate:   nextState,
					CheckpointID:     chkID,
				}, nil
			}

			// Agent policy (or no checkpoint handler): override and continue (FR-016).
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

		// Increment review_cycle each time the feature enters reviewing (FR-002).
		if nextState == string(model.FeatureStatusReviewing) {
			if err := entitySvc.IncrementFeatureReviewCycle(feature.ID, feature.Slug); err != nil {
				return AdvanceResult{}, fmt.Errorf("incrementing review_cycle for %s: %w", feature.ID, err)
			}
			feature.ReviewCycle++
		}

		// Halt after entering a stop state (unless it was the explicit target).
		if advanceStopStates[nextState] && !isTarget {
			requiresHumanReview := cfg != nil && cfg.RequiresHumanReview != nil && cfg.RequiresHumanReview()
			if requiresHumanReview {
				return AdvanceResult{
					FinalStatus:      nextState,
					AdvancedThrough:  advancedThrough,
					OverriddenGates:  overriddenGates,
					StoppedReason:    "stopped at reviewing: require_human_review is true",
					StructuralChecks: structuralChecks,
				}, nil
			}
			// Check auto-advance eligibility.
			if err := checkAllTasksHaveVerification(feature, entitySvc); err != nil {
				return AdvanceResult{
					FinalStatus:      nextState,
					AdvancedThrough:  advancedThrough,
					OverriddenGates:  overriddenGates,
					StoppedReason:    fmt.Sprintf("stopped at reviewing: %s", err),
					StructuralChecks: structuralChecks,
				}, nil
			}
			// All conditions satisfied — continue past reviewing.
			continue
		}
	}

	return AdvanceResult{
		FinalStatus:      string(feature.Status),
		AdvancedThrough:  advancedThrough,
		OverriddenGates:  overriddenGates,
		StructuralChecks: structuralChecks,
	}, nil
}

// resolveCheckGate returns the gate check function from config, or the default.
func resolveCheckGate(cfg *AdvanceConfig) GateCheckFunc {
	if cfg != nil && cfg.CheckGate != nil {
		return cfg.CheckGate
	}
	return func(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult {
		return CheckTransitionGate(from, to, feature, docSvc, entitySvc)
	}
}

// resolveOverridePolicy returns the policy function from config, or a function
// that always returns "agent".
func resolveOverridePolicy(cfg *AdvanceConfig) OverridePolicyFunc {
	if cfg != nil && cfg.OverridePolicy != nil {
		return cfg.OverridePolicy
	}
	return func(string) string { return "agent" }
}

// resolveOnCheckpoint returns the checkpoint handler from config, or nil.
func resolveOnCheckpoint(cfg *AdvanceConfig) CheckpointCreateFunc {
	if cfg != nil {
		return cfg.OnCheckpoint
	}
	return nil
}
