package health

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// CheckGateSources
// ---------------------------------------------------------------------------

func TestCheckGateSources_AllRegistry(t *testing.T) {
	t.Parallel()

	allStages := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	registryStages := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}

	result := CheckGateSources(registryStages, allStages)

	if len(result.Issues) != len(allStages) {
		t.Fatalf("issues count = %d, want %d", len(result.Issues), len(allStages))
	}
	for _, issue := range result.Issues {
		if issue.Severity != SeverityInfo {
			t.Errorf("severity = %q, want %q for %s", issue.Severity, SeverityInfo, issue.Message)
		}
		if !strings.Contains(issue.Message, "registry") {
			t.Errorf("expected 'registry' in message: %s", issue.Message)
		}
		if strings.Contains(issue.Message, "hardcoded") {
			t.Errorf("unexpected 'hardcoded' in message: %s", issue.Message)
		}
	}
}

func TestCheckGateSources_AllHardcoded(t *testing.T) {
	t.Parallel()

	allStages := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	var registryStages []string // empty — no registry

	result := CheckGateSources(registryStages, allStages)

	if len(result.Issues) != len(allStages) {
		t.Fatalf("issues count = %d, want %d", len(result.Issues), len(allStages))
	}
	for _, issue := range result.Issues {
		if issue.Severity != SeverityInfo {
			t.Errorf("severity = %q, want %q for %s", issue.Severity, SeverityInfo, issue.Message)
		}
		if !strings.Contains(issue.Message, "hardcoded") {
			t.Errorf("expected 'hardcoded' in message: %s", issue.Message)
		}
		if strings.Contains(issue.Message, "registry") {
			t.Errorf("unexpected 'registry' in message: %s", issue.Message)
		}
	}
}

func TestCheckGateSources_Mixed(t *testing.T) {
	t.Parallel()

	allStages := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	registryStages := []string{"designing", "developing"}

	result := CheckGateSources(registryStages, allStages)

	if len(result.Issues) != len(allStages) {
		t.Fatalf("issues count = %d, want %d", len(result.Issues), len(allStages))
	}

	// Build a map of stage → message for easy lookup.
	byStage := make(map[string]string)
	for _, issue := range result.Issues {
		for _, stage := range allStages {
			if strings.Contains(issue.Message, "stage "+stage+":") {
				byStage[stage] = issue.Message
			}
		}
	}

	for _, stage := range []string{"designing", "developing"} {
		msg, ok := byStage[stage]
		if !ok {
			t.Errorf("missing issue for registry stage %q", stage)
			continue
		}
		if !strings.Contains(msg, "registry") {
			t.Errorf("stage %q: expected 'registry' in message: %s", stage, msg)
		}
	}

	for _, stage := range []string{"specifying", "dev-planning", "reviewing"} {
		msg, ok := byStage[stage]
		if !ok {
			t.Errorf("missing issue for hardcoded stage %q", stage)
			continue
		}
		if !strings.Contains(msg, "hardcoded") {
			t.Errorf("stage %q: expected 'hardcoded' in message: %s", stage, msg)
		}
	}
}

func TestCheckGateSources_EmptyStages(t *testing.T) {
	t.Parallel()

	result := CheckGateSources(nil, nil)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckGateSources_MessageFormat(t *testing.T) {
	t.Parallel()

	result := CheckGateSources([]string{"designing"}, []string{"designing", "specifying"})

	if len(result.Issues) != 2 {
		t.Fatalf("issues count = %d, want 2", len(result.Issues))
	}

	// Check exact message format for registry stage.
	found := false
	for _, issue := range result.Issues {
		if issue.Message == "stage designing: gate source is registry" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("did not find exact registry message for designing")
	}

	// Check exact message format for hardcoded stage.
	found = false
	for _, issue := range result.Issues {
		if issue.Message == "stage specifying: gate source is hardcoded" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("did not find exact hardcoded message for specifying")
	}
}

// ---------------------------------------------------------------------------
// CheckCheckpointOverrides
// ---------------------------------------------------------------------------

func TestCheckCheckpointOverrides_WithCheckpointID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01AAAA",
			"overrides": []any{
				map[string]any{
					"from_status":   "designing",
					"to_status":     "specifying",
					"reason":        "pending human approval",
					"checkpoint_id": "CHK-01XXXX",
					"timestamp":     "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("severity = %q, want %q", issue.Severity, SeverityWarning)
	}
	if issue.EntityID != "FEAT-01AAAA" {
		t.Errorf("entityID = %q, want %q", issue.EntityID, "FEAT-01AAAA")
	}
	if !strings.Contains(issue.Message, "FEAT-01AAAA") {
		t.Errorf("message missing feature ID: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "designing") {
		t.Errorf("message missing from_status: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "specifying") {
		t.Errorf("message missing to_status: %s", issue.Message)
	}
	if !strings.Contains(issue.Message, "CHK-01XXXX") {
		t.Errorf("message missing checkpoint_id: %s", issue.Message)
	}
}

func TestCheckCheckpointOverrides_OverrideWithoutCheckpointID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01BBBB",
			"overrides": []any{
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "regular override, no checkpoint",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty (no checkpoint_id means not a checkpoint override)", result.Issues)
	}
}

func TestCheckCheckpointOverrides_NoOverrides(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id":     "FEAT-01CCCC",
			"status": "developing",
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckCheckpointOverrides_NoFeatures(t *testing.T) {
	t.Parallel()

	result := CheckCheckpointOverrides(nil)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckCheckpointOverrides_MixedOverrides(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01DDDD",
			"overrides": []any{
				// Regular override — no checkpoint_id.
				map[string]any{
					"from_status": "designing",
					"to_status":   "specifying",
					"reason":      "regular",
					"timestamp":   "2026-01-01T00:00:00Z",
				},
				// Checkpoint override — has checkpoint_id.
				map[string]any{
					"from_status":   "specifying",
					"to_status":     "dev-planning",
					"reason":        "pending approval",
					"checkpoint_id": "CHK-01YYYY",
					"timestamp":     "2026-01-02T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	// Only the checkpoint override should produce a warning.
	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}
	if !strings.Contains(result.Issues[0].Message, "CHK-01YYYY") {
		t.Errorf("message missing checkpoint_id: %s", result.Issues[0].Message)
	}
}

func TestCheckCheckpointOverrides_MultipleFeatures(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01EEEE",
			"overrides": []any{
				map[string]any{
					"from_status":   "designing",
					"to_status":     "specifying",
					"checkpoint_id": "CHK-01AAAA",
					"timestamp":     "2026-01-01T00:00:00Z",
				},
			},
		},
		{
			"id":     "FEAT-01FFFF",
			"status": "developing",
			// No overrides.
		},
		{
			"id": "FEAT-01GGGG",
			"overrides": []any{
				map[string]any{
					"from_status":   "dev-planning",
					"to_status":     "developing",
					"checkpoint_id": "CHK-01BBBB",
					"timestamp":     "2026-01-03T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityWarning {
		t.Errorf("status = %q, want %q", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("issues count = %d, want 2", len(result.Issues))
	}

	ids := make(map[string]bool)
	for _, issue := range result.Issues {
		ids[issue.EntityID] = true
	}
	if !ids["FEAT-01EEEE"] {
		t.Error("missing issue for FEAT-01EEEE")
	}
	if !ids["FEAT-01GGGG"] {
		t.Error("missing issue for FEAT-01GGGG")
	}
}

func TestCheckCheckpointOverrides_MessageFormat(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01HHHH",
			"overrides": []any{
				map[string]any{
					"from_status":   "dev-planning",
					"to_status":     "developing",
					"checkpoint_id": "CHK-01ZZZZ",
					"timestamp":     "2026-06-01T12:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if len(result.Issues) != 1 {
		t.Fatalf("issues count = %d, want 1", len(result.Issues))
	}

	expected := "feature FEAT-01HHHH: checkpoint override pending on dev-planning→developing (checkpoint CHK-01ZZZZ)"
	if result.Issues[0].Message != expected {
		t.Errorf("message = %q\nwant    = %q", result.Issues[0].Message, expected)
	}
}

func TestCheckCheckpointOverrides_SkipsFeatureWithMissingID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			// No "id" field.
			"overrides": []any{
				map[string]any{
					"from_status":   "designing",
					"to_status":     "specifying",
					"checkpoint_id": "CHK-01SKIP",
					"timestamp":     "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}

func TestCheckCheckpointOverrides_SkipsMalformedOverrideEntry(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01IIII",
			"overrides": []any{
				// Valid checkpoint override.
				map[string]any{
					"from_status":   "designing",
					"to_status":     "specifying",
					"checkpoint_id": "CHK-01GOOD",
					"timestamp":     "2026-01-01T00:00:00Z",
				},
				// Malformed entry (not a map).
				"not-a-map",
				// Another valid checkpoint override.
				map[string]any{
					"from_status":   "specifying",
					"to_status":     "dev-planning",
					"checkpoint_id": "CHK-01ALSO",
					"timestamp":     "2026-01-02T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if len(result.Issues) != 2 {
		t.Fatalf("issues count = %d, want 2 (malformed entry skipped)", len(result.Issues))
	}
}

func TestCheckCheckpointOverrides_EmptyCheckpointID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{
			"id": "FEAT-01JJJJ",
			"overrides": []any{
				map[string]any{
					"from_status":   "designing",
					"to_status":     "specifying",
					"checkpoint_id": "", // empty string — not a real checkpoint
					"timestamp":     "2026-01-01T00:00:00Z",
				},
			},
		},
	}

	result := CheckCheckpointOverrides(features)

	if result.Status != SeverityOK {
		t.Errorf("status = %q, want %q (empty checkpoint_id is not a checkpoint override)", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("issues = %v, want empty", result.Issues)
	}
}
