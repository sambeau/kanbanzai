package kbzinit

import "testing"

// TestDecisionConstants verifies all four Decision constants are distinct and
// have the expected iota-derived values.
func TestDecisionConstants(t *testing.T) {
	cases := []struct {
		name string
		d    Decision
		want Decision
	}{
		{"Create", Create, 0},
		{"Overwrite", Overwrite, 1},
		{"NoOp", NoOp, 2},
		{"WarnSkip", WarnSkip, 3},
	}
	for _, tc := range cases {
		if tc.d != tc.want {
			t.Errorf("Decision %s = %d, want %d", tc.name, tc.d, tc.want)
		}
	}

	// Ensure all constants are distinct.
	all := []Decision{Create, Overwrite, NoOp, WarnSkip}
	seen := make(map[Decision]bool)
	for _, d := range all {
		if seen[d] {
			t.Errorf("duplicate Decision value %d", d)
		}
		seen[d] = true
	}
}

// TestArtifactFields is a compile-time check: if any required field is
// removed from Artifact, this test will fail to compile.
func TestArtifactFields(t *testing.T) {
	a := Artifact{
		Name:        "test",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/agents/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-agents/SKILL.md",
		Required:    true,
		Optional:    false,
		Marker: MarkerSpec{
			Comment:      "# kanbanzai-managed:",
			VersionKind:  IntCounter,
			CurrentValue: "1",
		},
	}
	if a.Name == "" {
		t.Error("Name must be set")
	}
	if a.Kind != WorkflowSkill {
		t.Error("Kind must be WorkflowSkill")
	}
	if !a.Required {
		t.Error("Required must be true")
	}
	if a.Optional {
		t.Error("Optional must be false")
	}
	if a.Marker.VersionKind != IntCounter {
		t.Error("Marker.VersionKind must be IntCounter")
	}
}

// TestArtifactKindConstants ensures all ArtifactKind constants are non-empty
// and distinct.
func TestArtifactKindConstants(t *testing.T) {
	kinds := []ArtifactKind{
		WorkflowSkill,
		TaskSkill,
		Role,
		AgentsMd,
		CopilotInstructions,
		StageBindings,
	}
	seen := make(map[ArtifactKind]bool)
	for _, k := range kinds {
		if k == "" {
			t.Error("ArtifactKind constant must not be empty")
		}
		if seen[k] {
			t.Errorf("duplicate ArtifactKind %q", k)
		}
		seen[k] = true
	}
}

// TestVersionKindConstants ensures both VersionKind constants are non-empty
// and distinct.
func TestVersionKindConstants(t *testing.T) {
	if IntCounter == "" {
		t.Error("IntCounter must not be empty")
	}
	if Semver == "" {
		t.Error("Semver must not be empty")
	}
	if IntCounter == Semver {
		t.Error("IntCounter and Semver must be distinct")
	}
}
