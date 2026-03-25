package merge

import "testing"

func TestGateStatusConstants(t *testing.T) {
	tests := []struct {
		status GateStatus
		want   string
	}{
		{GateStatusPassed, "passed"},
		{GateStatusFailed, "failed"},
		{GateStatusWarning, "warning"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("GateStatus %q: got %q, want %q", tt.status, string(tt.status), tt.want)
		}
	}
}

func TestGateSeverityConstants(t *testing.T) {
	tests := []struct {
		severity GateSeverity
		want     string
	}{
		{GateSeverityBlocking, "blocking"},
		{GateSeverityWarning, "warning"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.want {
			t.Errorf("GateSeverity %q: got %q, want %q", tt.severity, string(tt.severity), tt.want)
		}
	}
}

func TestOverallStatusConstants(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{OverallStatusPassed, "passed"},
		{OverallStatusWarnings, "warnings"},
		{OverallStatusBlocked, "blocked"},
	}

	for _, tt := range tests {
		if tt.status != tt.want {
			t.Errorf("OverallStatus: got %q, want %q", tt.status, tt.want)
		}
	}
}

func TestGateResult_ZeroValue(t *testing.T) {
	var r GateResult

	if r.Name != "" {
		t.Errorf("zero GateResult.Name: got %q, want empty", r.Name)
	}
	if r.Status != "" {
		t.Errorf("zero GateResult.Status: got %q, want empty", r.Status)
	}
	if r.Severity != "" {
		t.Errorf("zero GateResult.Severity: got %q, want empty", r.Severity)
	}
	if r.Message != "" {
		t.Errorf("zero GateResult.Message: got %q, want empty", r.Message)
	}
}

func TestGateCheckResult_ZeroValue(t *testing.T) {
	var r GateCheckResult

	if r.EntityID != "" {
		t.Errorf("zero GateCheckResult.EntityID: got %q, want empty", r.EntityID)
	}
	if r.Branch != "" {
		t.Errorf("zero GateCheckResult.Branch: got %q, want empty", r.Branch)
	}
	if r.OverallStatus != "" {
		t.Errorf("zero GateCheckResult.OverallStatus: got %q, want empty", r.OverallStatus)
	}
	if r.Gates != nil {
		t.Errorf("zero GateCheckResult.Gates: got %v, want nil", r.Gates)
	}
}
