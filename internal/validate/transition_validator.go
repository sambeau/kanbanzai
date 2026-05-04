package validate

import (
	"context"
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
// This is distinct from ValidatorSummary (validator_dispatch.go) which is
// used for document validation dispatch results.
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
	// FilesModified lists files changed in this feature (for conditional gate evaluation).
	// When empty, conditional gates treat all changes as implementation changes.
	FilesModified []string
}

// TransitionValidatorDispatcher evaluates transition_validator hooks from stage bindings.
// It checks whether the from-stage has a transition_validator, evaluates the
// gate mode against the feature's tier, and returns validation results.
type TransitionValidatorDispatcher struct {
	cache       BindingLookup
	dispatchSvc ValidatorDispatcher // optional; when set, used for auto-mode validation dispatch
}

// BindingLookup is the interface for looking up stage bindings.
type BindingLookup interface {
	LookupStage(stage string) (*binding.StageBinding, bool)
}

// NewTransitionValidatorDispatcher creates a dispatcher backed by the given binding lookup.
func NewTransitionValidatorDispatcher(cache BindingLookup) *TransitionValidatorDispatcher {
	return &TransitionValidatorDispatcher{cache: cache}
}

// WithDispatch sets the validator dispatch service for auto-mode validation.
// When set, auto gate modes will dispatch actual validator sub-agents rather
// than returning placeholder pass results.
func (d *TransitionValidatorDispatcher) WithDispatch(svc ValidatorDispatcher) *TransitionValidatorDispatcher {
	d.dispatchSvc = svc
	return d
}

// ValidateTransition checks if the from-stage has a transition_validator hook,
// evaluates gate mode vs feature tier, and returns validation results.
//
// Rules:
//   - If no transition_validator for the from-stage → pass (nil result).
//   - If gate_mode is "human" → skip, pass.
//   - If gate_mode is "conditional" → skip, pass (conditional gates are handled
//     by model routing in P44).
//   - If feature.BlockedReason is set → skip, pass. The feature is in human
//     escalation and must not attempt further automated validation (REQ-PIPE-004).
//   - If override is true → skip, record override.
//   - Otherwise, run validation placeholder (returns a pass for now —
//     actual validation dispatch is implemented by callers via the
//     ValidatorDispatchFunc mechanism).
func (d *TransitionValidatorDispatcher) ValidateTransition(input ValidatorDispatchInput) (*ValidatorResult, error) {
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

	// Conditional gate mode — for retro_fix tier, check if changes are
	// documentation-only (REQ-TIER-004). Implementation files (outside
	// work/, docs/, refs/) trigger a full review panel; documentation-only
	// changes skip the review gate.
	//
	// File change detection uses feature.FilesModified when available.
	// When P44 model routing arrives, the conditional logic can be
	// enhanced without changing the validator framework.
	if gateMode == "conditional" {
		return d.evaluateConditional(input, tv)
	}

	// Fast-track tier skips automatic validation (human gate equivalent).
	if strings.EqualFold(input.Feature.Tier, "fast-track") {
		return nil, nil
	}

	// Feature is blocked — human escalation in progress (REQ-PIPE-004).
	// The system must not attempt further automated validation until the
	// human responds and clears the blocked_reason.
	if input.Feature.BlockedReason != "" {
		return nil, nil
	}

	// Override always available — skip validation.
	if input.Override {
		return nil, nil
	}

	// Auto gate mode: if a dispatch service is configured, run the validator.
	if gateMode == "auto" && d.dispatchSvc != nil {
		return d.runAutoValidation(input, tv)
	}

	// Auto gate mode without dispatch service: pass as placeholder.
	return &ValidatorResult{
		Stage:  input.FromStatus,
		Passed: true,
		Checks: []ValidatorCheck{
			{
				CheckID:   "AUTO_PLACEHOLDER",
				Passed:    true,
				Blocking:  false,
				Summary:   "auto gate mode: no dispatch service configured; treating as pass",
				CheckType: "notice",
			},
		},
		BlockingFail: false,
	}, nil
}

// evaluateConditional implements the conditional gate logic for retro_fix
// (REQ-TIER-004). Documentation-only changes (files only under work/,
// docs/, or refs/) skip the review gate with an explicit annotation.
// Implementation changes trigger the full review panel.
//
// When FilesModified is empty (no file list available), treat as
// implementation change (conservative). P44 model routing will enhance
// this with more accurate file classification.
func (d *TransitionValidatorDispatcher) evaluateConditional(input ValidatorDispatchInput, tv *binding.TransitionValidator) (*ValidatorResult, error) {
	isDocOnly := len(input.FilesModified) > 0
	for _, f := range input.FilesModified {
		if !isDocOnlyChange(f) {
			isDocOnly = false
			break
		}
	}

	if isDocOnly {
		// Documentation-only change: skip review gate (REQ-TIER-004).
		return &ValidatorResult{
			Stage:  input.FromStatus,
			Passed: true,
			Checks: []ValidatorCheck{
				{
					CheckID:   "COND_DOCS_ONLY",
					Passed:    true,
					Blocking:  false,
					Summary:   "documentation-only change: review gate skipped per REQ-TIER-004",
					CheckType: "notice",
				},
			},
			BlockingFail: false,
		}, nil
	}

	// Implementation change on retro_fix: requires specialist review panel.
	// If dispatch service is configured, run the validator.
	if d.dispatchSvc != nil {
		return d.runAutoValidation(input, tv)
	}

	// No dispatch service: return a result indicating review is required.
	return &ValidatorResult{
		Stage:  input.FromStatus,
		Passed: false,
		Checks: []ValidatorCheck{
			{
				CheckID:   "COND_IMPL_CHANGE",
				Passed:    false,
				Blocking:  true,
				Summary:   "implementation change on retro_fix tier: specialist review panel required (REQ-TIER-004)",
				CheckType: "conditional",
			},
		},
		BlockingFail: true,
	}, nil
}

// isDocOnlyChange returns true if the file path is under a documentation-only
// directory (work/, docs/, refs/). Implementation files are everything else.
func isDocOnlyChange(path string) bool {
	docPrefixes := []string{"work/", "docs/", "refs/", ".kbz/roles/", ".kbz/skills/", ".agents/"}
	for _, prefix := range docPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// runAutoValidation dispatches the actual validator via the dispatch service
// and converts the summary to a ValidatorResult.
func (d *TransitionValidatorDispatcher) runAutoValidation(input ValidatorDispatchInput, tv *binding.TransitionValidator) (*ValidatorResult, error) {
	vctx := ValidatorContext{
		FeatureID: input.Feature.ID,
	}
	summary, err := d.dispatchSvc.Dispatch(context.Background(), tv.Role, tv.Skill, vctx)
	if err != nil {
		return nil, fmt.Errorf("validator dispatch failed: %w", err)
	}

	checks := make([]ValidatorCheck, 0, summary.BlockingCount+summary.NonBlockingCount)
	// Without the full report we can't enumerate individual checks, but we
	// can capture the aggregate outcome from the summary.
	if summary.Verdict == VerdictFail || summary.Verdict == VerdictPassWithNotes {
		checks = append(checks, ValidatorCheck{
			CheckID:   "AGGREGATE",
			Passed:    summary.Verdict != VerdictFail,
			Blocking:  summary.BlockingCount > 0,
			Summary:   fmt.Sprintf("validator returned %s: %d blocking, %d non-blocking", summary.Verdict, summary.BlockingCount, summary.NonBlockingCount),
			CheckType: "aggregate",
		})
	}

	return &ValidatorResult{
		Stage:        input.FromStatus,
		Passed:       summary.Verdict != VerdictFail,
		Checks:       checks,
		ReportDocID:  summary.ReportDocID,
		BlockingFail: summary.BlockingCount > 0,
	}, nil
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
