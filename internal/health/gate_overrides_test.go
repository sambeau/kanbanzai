package health

import (
	"strings"
	"testing"
)

func TestCheckGateOverrides_NoFeatures(t *testing.T) {
	t.Parallel()

	result := CheckGateOverrides(nil)
	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckGateOverrides_FeatureWithNoOverrides(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id":     "FEAT-01AAAA",
			"status": "developing",
		},
	}

	result := CheckGateOverrides(features)
	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckGateOverrides_FeatureWithOneOverride(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id":     "FEAT-01BBBB",
			"status": "specifying",
			"overrides": []any{
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "design exists in external system",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckGateOverrides(features)
	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("issue.Severity = %q, want %q", issue.Severity, SeverityWarning)
	}
	if issue.EntityID != "FEAT-01BBBB" {
		t.Errorf("issue.EntityID = %q, want %q", issue.EntityID, "FEAT-01BBBB")
	}
	if !strings.Contains(issue.Message, "FEAT-01BBBB") {
		t.Errorf("message does not contain feature ID: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "designing") {
		t.Errorf("message does not contain from_status 'designing': %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "specifying") {
		t.Errorf("message does not contain to_status 'specifying': %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "design exists in external system") {
		t.Errorf("message does not contain reason: %s", issue.Message)
	}
}

func TestCheckGateOverrides_FeatureWithMultipleOverrides(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id":     "FEAT-01CCCC",
			"status": "developing",
			"overrides": []any{
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "first override",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
				map[string]any{
					"from_status": "specifying",
					"to_status":   "dev-planning",
					"reason":      "second override",
					"timestamp":   "2026-01-02T00:00:00Z",
				},
				map[string]any{
					"from_status": "dev-planning",
					"to_status":   "developing",
					"reason":      "third override",
					"timestamp":   "2026-01-03T00:00:00Z",
				},
			},
		},
	}

	result := CheckGateOverrides(features)
	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 3 {
		t.Fatalf("issues count = %d, want 3 (one per override)", len(result.Issues))
	}

	for i, issue := range result.Issues {
		if issue.Severity != SeverityWarning {
			t.Errorf("issues[%d].Severity = %q, want %q", i, issue.Severity, SeverityWarning)
		}
		if issue.EntityID != "FEAT-01CCCC" {
			t.Errorf("issues[%d].EntityID = %q, want %q", i, issue.EntityID, "FEAT-01CCCC")
		}
	}
}

func TestCheckGateOverrides_MultipleFeatures_SomeWithOverrides(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id":     "FEAT-01DDDD",
			"status": "developing",
			// No overrides field.
		},
		{
			"id":     "FEAT-01EEEE",
			"status": "specifying",
			"overrides": []any{
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "fast-track",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
			},
		},
		{
			"id":     "FEAT-01FFFF",
			"status": "proposed",
			// Empty overrides slice.
			"overrides": []any{},
		},
	}

	result := CheckGateOverrides(features)
	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	// Only FEAT-01EEEE has an override record.
	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].EntityID != "FEAT-01EEEE" {
		t.Errorf("issue.EntityID = %q, want %q", result.Issues[0].EntityID, "FEAT-01EEEE")
	}
}

func TestCheckGateOverrides_MessageFormat(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01GGGG",
			"overrides": []any{
				map[string]any{
					"from_status": "dev-planning",
					"to_status":   "developing",
					"reason":      "tasks will be created shortly",
					"timestamp":   "2026-06-01T12:00:00Z",
				},
			},
		},
	}

	result := CheckGateOverrides(features)
	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}

	msg := result.Issues[0].Message
	// Expected: "feature FEAT-01GGGG: gate override on dev-planning→developing: tasks will be created shortly"
	expected := "feature FEAT-01GGGG: gate override on dev-planning→developing: tasks will be created shortly"
	if msg != expected {
		t.Errorf("message = %q\nwant    = %q", msg, expected)
	}
}

func TestCheckGateOverrides_SkipsFeatureWithMissingID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			// No "id" field.
			"status": "developing",
			"overrides": []any{
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "should be skipped",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckGateOverrides(features)
	// Feature without ID is skipped entirely.
	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q (features without id are skipped)", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty (features without id are skipped)", result.Issues)
	}
}

func TestCheckGateOverrides_SkipsMalformedOverrideEntry(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01HHHH",
			"overrides": []any{
				// Valid entry.
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "valid override",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
				// Malformed entry (not a map).
				"not-a-map",
				// Another valid entry.
				map[string]any{
					"from_status": "specifying",
					"to_status":   "dev-planning",
					"reason":      "another valid",
					"timestamp":   "2026-01-02T00:00:00Z",
				},
			},
		},
	}

	result := CheckGateOverrides(features)
	// Only 2 valid entries → 2 warnings (malformed entry is skipped).
	if len(result.Issues) != 2 {
		t.Fatalf("issues count = %d, want 2 (malformed override skipped)", len(result.Issues))
	}
}
