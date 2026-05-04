package validate

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

// stubBindingLookup implements BindingLookup for testing.
type stubBindingLookup struct {
	bindings map[string]*binding.StageBinding
}

func (s *stubBindingLookup) LookupStage(stage string) (*binding.StageBinding, bool) {
	sb, ok := s.bindings[stage]
	return sb, ok
}

func TestValidatorDispatcher_NoValidatorForStage(t *testing.T) {
	lookup := &stubBindingLookup{
		bindings: map[string]*binding.StageBinding{
			"designing": {Description: "Design"},
		},
	}
	d := NewTransitionValidatorDispatcher(lookup)

	result, err := d.ValidateTransition(ValidatorDispatchInput{
		Feature:    &model.Feature{ID: "FEAT-001"},
		FromStatus: "designing",
		ToStatus:   "specifying",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result when no validator configured, got: %+v", result)
	}
}

func TestValidatorDispatcher_HumanGateMode_Skipped(t *testing.T) {
	lookup := &stubBindingLookup{
		bindings: map[string]*binding.StageBinding{
			"specifying": {
				Description: "Spec",
				TransitionValidator: &binding.TransitionValidator{
					Role:     "spec-validator",
					Skill:    "validate-spec",
					GateMode: "human",
				},
			},
		},
	}
	d := NewTransitionValidatorDispatcher(lookup)

	result, err := d.ValidateTransition(ValidatorDispatchInput{
		Feature:    &model.Feature{ID: "FEAT-001"},
		FromStatus: "specifying",
		ToStatus:   "dev-planning",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result when gate_mode is human, got: %+v", result)
	}
}

func TestValidatorDispatcher_FastTrackTier_Skipped(t *testing.T) {
	lookup := &stubBindingLookup{
		bindings: map[string]*binding.StageBinding{
			"specifying": {
				Description: "Spec",
				TransitionValidator: &binding.TransitionValidator{
					Role:     "spec-validator",
					Skill:    "validate-spec",
					GateMode: "auto",
				},
			},
		},
	}
	d := NewTransitionValidatorDispatcher(lookup)

	result, err := d.ValidateTransition(ValidatorDispatchInput{
		Feature:    &model.Feature{ID: "FEAT-001", Tier: "fast-track"},
		FromStatus: "specifying",
		ToStatus:   "dev-planning",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for fast-track tier, got: %+v", result)
	}
}

func TestValidatorDispatcher_Override_Skipped(t *testing.T) {
	lookup := &stubBindingLookup{
		bindings: map[string]*binding.StageBinding{
			"specifying": {
				Description: "Spec",
				TransitionValidator: &binding.TransitionValidator{
					Role:     "spec-validator",
					Skill:    "validate-spec",
					GateMode: "auto",
				},
			},
		},
	}
	d := NewTransitionValidatorDispatcher(lookup)

	result, err := d.ValidateTransition(ValidatorDispatchInput{
		Feature:        &model.Feature{ID: "FEAT-001"},
		FromStatus:     "specifying",
		ToStatus:       "dev-planning",
		Override:       true,
		OverrideReason: "testing",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result when override is true, got: %+v", result)
	}
}

func TestValidatorDispatcher_StandardTier_AutoMode_ReturnsNil(t *testing.T) {
	// For now, the dispatcher returns nil (no-op) because actual validation
	// dispatch is implemented by callers. The dispatcher only checks gate mode.
	lookup := &stubBindingLookup{
		bindings: map[string]*binding.StageBinding{
			"specifying": {
				Description: "Spec",
				TransitionValidator: &binding.TransitionValidator{
					Role:     "spec-validator",
					Skill:    "validate-spec",
					GateMode: "auto",
				},
			},
		},
	}
	d := NewTransitionValidatorDispatcher(lookup)

	result, err := d.ValidateTransition(ValidatorDispatchInput{
		Feature:    &model.Feature{ID: "FEAT-001"},
		FromStatus: "specifying",
		ToStatus:   "dev-planning",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for auto mode (with placeholder when no dispatch service)")
	}
	if !result.Passed {
		t.Error("auto mode placeholder should pass")
	}
	if result.BlockingFail {
		t.Error("auto mode placeholder should not be blocking")
	}
}

func TestTransitionGateMode(t *testing.T) {
	tests := []struct {
		name     string
		sb       *binding.StageBinding
		feature  *model.Feature
		wantMode string
	}{
		{
			name:     "nil binding returns empty",
			sb:       nil,
			feature:  &model.Feature{},
			wantMode: "",
		},
		{
			name: "no validator returns empty",
			sb: &binding.StageBinding{
				TransitionValidator: nil,
			},
			feature:  &model.Feature{},
			wantMode: "",
		},
		{
			name: "human gate mode returns human",
			sb: &binding.StageBinding{
				TransitionValidator: &binding.TransitionValidator{GateMode: "human"},
			},
			feature:  &model.Feature{},
			wantMode: "human",
		},
		{
			name: "fast-track tier returns human",
			sb: &binding.StageBinding{
				TransitionValidator: &binding.TransitionValidator{GateMode: "auto"},
			},
			feature:  &model.Feature{Tier: "fast-track"},
			wantMode: "human",
		},
		{
			name: "standard tier with auto returns auto",
			sb: &binding.StageBinding{
				TransitionValidator: &binding.TransitionValidator{GateMode: "auto"},
			},
			feature:  &model.Feature{},
			wantMode: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransitionGateMode(tt.sb, tt.feature)
			if got != tt.wantMode {
				t.Errorf("TransitionGateMode() = %q, want %q", got, tt.wantMode)
			}
		})
	}
}

func TestBuildTransitionValidatorError(t *testing.T) {
	result := ValidatorResult{
		Stage:        "specifying",
		Passed:       false,
		BlockingFail: true,
		ReportDocID:  "DOC-REPORT-001",
		Checks: []ValidatorCheck{
			{CheckID: "CHK-001", Passed: false, Blocking: true, Summary: "Missing acceptance criteria"},
			{CheckID: "CHK-002", Passed: false, Blocking: false, Summary: "Style inconsistency"},
			{CheckID: "CHK-003", Passed: true, Blocking: false, Summary: "All good"},
		},
	}

	err := BuildTransitionValidatorError(result)
	tvErr, ok := err.(*TransitionValidatorError)
	if !ok {
		t.Fatalf("expected *TransitionValidatorError, got %T", err)
	}

	if len(tvErr.BlockingIDs) != 1 || tvErr.BlockingIDs[0] != "CHK-001" {
		t.Errorf("expected blocking IDs [CHK-001], got %v", tvErr.BlockingIDs)
	}
	if len(tvErr.NonBlocking) != 1 || tvErr.NonBlocking[0] != "CHK-002" {
		t.Errorf("expected non-blocking IDs [CHK-002], got %v", tvErr.NonBlocking)
	}
	if tvErr.ReportDocID != "DOC-REPORT-001" {
		t.Errorf("expected report doc ID DOC-REPORT-001, got %s", tvErr.ReportDocID)
	}
	if !tvErr.HasBlocking() {
		t.Error("expected HasBlocking() to be true")
	}
}

func TestBuildTransitionValidatorError_NoBlocking(t *testing.T) {
	result := ValidatorResult{
		Stage:        "specifying",
		Passed:       true,
		BlockingFail: false,
		Checks: []ValidatorCheck{
			{CheckID: "CHK-001", Passed: true, Blocking: false, Summary: "All good"},
		},
	}

	err := BuildTransitionValidatorError(result)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	tvErr, ok := err.(*TransitionValidatorError)
	if !ok {
		t.Fatalf("expected *TransitionValidatorError, got %T", err)
	}
	if tvErr.HasBlocking() {
		t.Error("expected HasBlocking() to be false when no blocking failures")
	}
}
