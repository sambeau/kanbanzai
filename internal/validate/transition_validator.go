package validate

import (
	"fmt"
	"strings"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/gate"
	"github.com/sambeau/kanbanzai/internal/model"
)

// ValidatorCheck represents a single check result from a transition validator.
type ValidatorCheck struct {
	CheckID   string `json:"check_id"`
	Passed    bool   `json:"passed"`
	Blocking  bool   `json:"blocking"`
	Summary   string `json:"summary"`
	Detail    string `json:"detail,omitempty"`
	CheckType string `json:"check_type"`
}

// ValidatorResult holds the outcome of all transition validator checks.
type ValidatorResult struct {
	Stage        string           `json:"stage"`
	Passed       bool             `json:"passed"`
	Checks       []ValidatorCheck `json:"checks"`
	ReportDocID  string           `json:"report_doc_id,omitempty"`
	BlockingFail bool             `json:"blocking_fail"`
}

// ValidatorDispatchInput carries the context needed to run transition validators.
type ValidatorDispatchInput struct {
	Feature        *model.Feature
	FromStatus     string
	ToStatus       string
	Override       bool
	OverrideReason string
}

// ValidatorDispatcher evaluates transition_validator hooks from stage bindings.
// It checks whether the from-stage has a transition_validator, evaluates the
// gate mode against the feature's tier, and returns validation results.
type ValidatorDispatcher struct {
	cache BindingLookup
}

// BindingLookup is the interface for looking up stage bindings.
type BindingLookup interface {
	LookupStage(stage string) (*binding.StageBinding, bool)
}

// NewValidatorDispatcher creates a dispatcher backed by the given binding lookup.
func NewValidatorDispatcher(cache BindingLookup) *ValidatorDispatcher {
	return &ValidatorDispatcher{cache: cache}
}

// ValidateTransition checks if the from-stage has a transition_validator hook,
// evaluates gate mode vs feature tier, and returns validation results.
//
// Rules:
//   - If no transition_validator for the from-stage → pass (nil result).
//   - If gate_mode is "human" → skip, pass.
//   - If feature.Tier is "fast-track" and gate_mode is "auto" → skip, pass.
//   - If override is true → skip, record override.
//   - Otherwise, run validation placeholder (returns a pass for now —
//     actual validation dispatch is implemented by callers via the
//     ValidatorDispatchFunc mechanism).
func (d *ValidatorDispatcher) ValidateTransition(input ValidatorDispatchInput) (*ValidatorResult, error) {
	stageBinding, ok := d.cache.LookupStage(input.FromStatus)
	if !ok || stageBinding == nil || stageBinding.TransitionValidator == nil {
		return nil, nil // no validator hook for this stage
	}

	tv := stageBinding.TransitionValidator

	// Determine effective gate mode.
	gateMode := tv.GateMode
	if gateMode == "" {
		gateMode = "auto"
	}

	// Human gate mode always skips validation.
	if gateMode == "human" {
		return nil, nil
	}

	// Fast-track tier skips automatic validation.
	if strings.EqualFold(input.Feature.Tier, "fast-track") {
		return nil, nil
	}

	// Override always available — skip validation.
	if input.Override {
		return nil, nil
	}

	return nil, nil
}

// BuildTransitionValidatorError constructs a structured error for blocking
// validator failures. The error message includes failing check IDs, blocking/non-blocking
// classification, and the validator report document ID.
func BuildTransitionValidatorError(result ValidatorResult) error {
	var blockingIDs []string
	var nonBlockingIDs []string
	for _, c := range result.Checks {
		if !c.Passed && c.Blocking {
			blockingIDs = append(blockingIDs, c.CheckID)
		}
		if !c.Passed && !c.Blocking {
			nonBlockingIDs = append(nonBlockingIDs, c.CheckID)
		}
	}

	parts := []string{
		fmt.Sprintf("transition validator %q (%s/%s) failed for stage %q",
			result.Stage,
			"transition_validator",
			"validate",
			result.Stage),
	}

	if len(blockingIDs) > 0 {
		parts = append(parts, fmt.Sprintf("blocking checks: %s", strings.Join(blockingIDs, ", ")))
	}
	if len(nonBlockingIDs) > 0 {
		parts = append(parts, fmt.Sprintf("non-blocking checks: %s", strings.Join(nonBlockingIDs, ", ")))
	}
	if result.ReportDocID != "" {
		parts = append(parts, fmt.Sprintf("validator report: %s", result.ReportDocID))
	}

	return &TransitionValidatorError{
		Stage:       result.Stage,
		Message:     strings.Join(parts, "; "),
		BlockingIDs: blockingIDs,
		NonBlocking: nonBlockingIDs,
		ReportDocID: result.ReportDocID,
	}
}

// TransitionValidatorError is a structured error for validator failures.
type TransitionValidatorError struct {
	Stage       string
	Message     string
	BlockingIDs []string
	NonBlocking []string
	ReportDocID string
}

func (e *TransitionValidatorError) Error() string {
	return e.Message
}

// HasBlocking returns true if any blocking checks failed.
func (e *TransitionValidatorError) HasBlocking() bool {
	return len(e.BlockingIDs) > 0
}

// TransitionGateMode determines the effective gate mode for a transition
// given the stage binding and feature tier.
func TransitionGateMode(sb *binding.StageBinding, feature *model.Feature) string {
	if sb == nil || sb.TransitionValidator == nil {
		return ""
	}
	tv := sb.TransitionValidator
	if tv.GateMode == "human" {
		return "human"
	}
	if strings.EqualFold(feature.Tier, "fast-track") {
		return "human" // fast-track features treat auto as human
	}
	return "auto"
}

// RegistryCacheBindingLookup adapts gate.RegistryCache to the BindingLookup interface.
type RegistryCacheBindingLookup struct {
	Cache *gate.RegistryCache
}

// LookupStage looks up a stage binding from the registry cache.
func (a *RegistryCacheBindingLookup) LookupStage(stage string) (*binding.StageBinding, bool) {
	if a.Cache == nil {
		return nil, false
	}
	bf, err := a.Cache.Get()
	if err != nil || bf == nil {
		return nil, false
	}
	sb, ok := bf.StageBindings[stage]
	return sb, ok
}
