package merge

import (
	"testing"
	"time"
)

func TestValidateOverride(t *testing.T) {
	tests := []struct {
		name    string
		req     OverrideRequest
		wantErr error
	}{
		{
			name: "valid request",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Gates:        []string{"tasks_complete"},
				Reason:       "Manual verification completed by QA team",
				OverriddenBy: "alice",
			},
			wantErr: nil,
		},
		{
			name: "valid request with empty gates",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Gates:        nil,
				Reason:       "Emergency hotfix approved by VP Eng",
				OverriddenBy: "bob",
			},
			wantErr: nil,
		},
		{
			name: "missing entity_id",
			req: OverrideRequest{
				EntityID:     "",
				Reason:       "Valid reason here",
				OverriddenBy: "alice",
			},
			wantErr: ErrOverrideNoEntityID,
		},
		{
			name: "missing reason",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Reason:       "",
				OverriddenBy: "alice",
			},
			wantErr: ErrOverrideNoReason,
		},
		{
			name: "reason too short",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Reason:       "too short",
				OverriddenBy: "alice",
			},
			wantErr: ErrOverrideReasonTooShort,
		},
		{
			name: "reason exactly 10 chars",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Reason:       "1234567890",
				OverriddenBy: "alice",
			},
			wantErr: nil,
		},
		{
			name: "missing overridden_by",
			req: OverrideRequest{
				EntityID:     "FEAT-001",
				Reason:       "Valid reason here",
				OverriddenBy: "",
			},
			wantErr: ErrOverrideNoOverriddenBy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOverride(tt.req)
			if err != tt.wantErr {
				t.Errorf("ValidateOverride: got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateOverrides_SpecificGates(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	req := OverrideRequest{
		EntityID:     "FEAT-001",
		Gates:        []string{"tasks_complete", "verification_passed"},
		Reason:       "Emergency release approved by CTO",
		OverriddenBy: "admin",
	}

	overrides := CreateOverrides(req, nil, now)

	if len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(overrides))
	}

	// Verify first override
	if overrides[0].Gate != "tasks_complete" {
		t.Errorf("override[0].Gate: got %q, want %q", overrides[0].Gate, "tasks_complete")
	}
	if overrides[0].Reason != req.Reason {
		t.Errorf("override[0].Reason: got %q, want %q", overrides[0].Reason, req.Reason)
	}
	if overrides[0].OverriddenBy != "admin" {
		t.Errorf("override[0].OverriddenBy: got %q, want %q", overrides[0].OverriddenBy, "admin")
	}
	if !overrides[0].OverriddenAt.Equal(now) {
		t.Errorf("override[0].OverriddenAt: got %v, want %v", overrides[0].OverriddenAt, now)
	}

	// Verify second override
	if overrides[1].Gate != "verification_passed" {
		t.Errorf("override[1].Gate: got %q, want %q", overrides[1].Gate, "verification_passed")
	}
}

func TestCreateOverrides_AllBlockingFailures(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	req := OverrideRequest{
		EntityID:     "FEAT-001",
		Gates:        nil, // Empty = override all blocking failures
		Reason:       "Blocking issues manually verified as non-critical",
		OverriddenBy: "alice",
	}

	blockingFailures := []GateResult{
		{Name: "tasks_complete", Status: GateStatusFailed, Severity: GateSeverityBlocking},
		{Name: "no_conflicts", Status: GateStatusFailed, Severity: GateSeverityBlocking},
	}

	overrides := CreateOverrides(req, blockingFailures, now)

	if len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(overrides))
	}

	if overrides[0].Gate != "tasks_complete" {
		t.Errorf("override[0].Gate: got %q, want %q", overrides[0].Gate, "tasks_complete")
	}
	if overrides[1].Gate != "no_conflicts" {
		t.Errorf("override[1].Gate: got %q, want %q", overrides[1].Gate, "no_conflicts")
	}
}

func TestCreateOverrides_EmptyBlockingFailures(t *testing.T) {
	now := time.Now()

	req := OverrideRequest{
		EntityID:     "FEAT-001",
		Gates:        nil,
		Reason:       "No blocking failures to override",
		OverriddenBy: "alice",
	}

	overrides := CreateOverrides(req, []GateResult{}, now)

	if len(overrides) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(overrides))
	}
}

func TestCreateOverrides_SpecificGatesIgnoresBlockingFailures(t *testing.T) {
	now := time.Now()

	req := OverrideRequest{
		EntityID:     "FEAT-001",
		Gates:        []string{"custom_gate"},
		Reason:       "Override specific gate only",
		OverriddenBy: "alice",
	}

	blockingFailures := []GateResult{
		{Name: "tasks_complete", Status: GateStatusFailed, Severity: GateSeverityBlocking},
	}

	overrides := CreateOverrides(req, blockingFailures, now)

	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}
	if overrides[0].Gate != "custom_gate" {
		t.Errorf("override[0].Gate: got %q, want %q", overrides[0].Gate, "custom_gate")
	}
}

func TestFormatOverride(t *testing.T) {
	o := Override{
		Gate:         "tasks_complete",
		Reason:       "Emergency deployment",
		OverriddenBy: "admin",
		OverriddenAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	got := FormatOverride(o)
	want := `gate "tasks_complete" overridden by admin at 2024-01-15T10:30:00Z: Emergency deployment`

	if got != want {
		t.Errorf("FormatOverride:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestOverride_ZeroValue(t *testing.T) {
	var o Override

	if o.Gate != "" {
		t.Errorf("zero Override.Gate: got %q, want empty", o.Gate)
	}
	if o.Reason != "" {
		t.Errorf("zero Override.Reason: got %q, want empty", o.Reason)
	}
	if o.OverriddenBy != "" {
		t.Errorf("zero Override.OverriddenBy: got %q, want empty", o.OverriddenBy)
	}
	if !o.OverriddenAt.IsZero() {
		t.Errorf("zero Override.OverriddenAt: got %v, want zero", o.OverriddenAt)
	}
}

func TestOverrideRequest_ZeroValue(t *testing.T) {
	var r OverrideRequest

	if r.EntityID != "" {
		t.Errorf("zero OverrideRequest.EntityID: got %q, want empty", r.EntityID)
	}
	if r.Gates != nil {
		t.Errorf("zero OverrideRequest.Gates: got %v, want nil", r.Gates)
	}
	if r.Reason != "" {
		t.Errorf("zero OverrideRequest.Reason: got %q, want empty", r.Reason)
	}
	if r.OverriddenBy != "" {
		t.Errorf("zero OverrideRequest.OverriddenBy: got %q, want empty", r.OverriddenBy)
	}
}

func TestOverrideErrors(t *testing.T) {
	// Ensure error messages are distinct
	errors := []error{
		ErrOverrideNoEntityID,
		ErrOverrideNoReason,
		ErrOverrideNoOverriddenBy,
		ErrOverrideReasonTooShort,
	}

	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
		if msg == "" {
			t.Error("error message should not be empty")
		}
	}
}
