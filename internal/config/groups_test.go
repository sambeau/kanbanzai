package config

import (
	"strings"
	"testing"
)

func TestEffectiveGroups_DefaultFull(t *testing.T) {
	t.Parallel()

	// No mcp section → defaults to preset: full (all groups enabled).
	cfg := DefaultConfig()
	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("EffectiveGroups() warnings = %v, want none", warnings)
	}

	wantGroups := []string{
		GroupCore, GroupPlanning, GroupKnowledge,
		GroupGit, GroupDocuments, GroupIncidents, GroupCheckpoints,
	}
	for _, g := range wantGroups {
		if !groups[g] {
			t.Errorf("groups[%q] = false, want true (full preset)", g)
		}
	}
}

func TestEffectiveGroups_CoreAlwaysEnabled(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v", err)
	}
	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true (core is always enabled)")
	}
}

func TestEffectiveGroups_PresetMinimal(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.MCP.Preset = "minimal"

	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("EffectiveGroups() warnings = %v, want none", warnings)
	}

	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true")
	}
	for _, g := range []string{GroupPlanning, GroupKnowledge, GroupGit, GroupDocuments, GroupIncidents, GroupCheckpoints} {
		if groups[g] {
			t.Errorf("groups[%q] = true, want false (minimal preset)", g)
		}
	}
}

func TestEffectiveGroups_PresetOrchestration(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.MCP.Preset = "orchestration"

	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("EffectiveGroups() warnings = %v, want none", warnings)
	}

	for _, g := range []string{GroupCore, GroupPlanning, GroupGit} {
		if !groups[g] {
			t.Errorf("groups[%q] = false, want true (orchestration preset)", g)
		}
	}
	for _, g := range []string{GroupKnowledge, GroupDocuments, GroupIncidents, GroupCheckpoints} {
		if groups[g] {
			t.Errorf("groups[%q] = true, want false (orchestration preset)", g)
		}
	}
}

func TestEffectiveGroups_PresetFull(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.MCP.Preset = "full"

	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}

	for _, g := range []string{GroupCore, GroupPlanning, GroupKnowledge, GroupGit, GroupDocuments, GroupIncidents, GroupCheckpoints} {
		if !groups[g] {
			t.Errorf("groups[%q] = false, want true (full preset)", g)
		}
	}
}

func TestEffectiveGroups_UnknownPreset_Error(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.MCP.Preset = "bogus"

	_, _, err := cfg.EffectiveGroups()
	if err == nil {
		t.Fatal("EffectiveGroups() error = nil, want non-nil for unknown preset")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error %q does not mention the unknown preset name", err.Error())
	}
}

func TestEffectiveGroups_ExplicitOverrideEnablesGroup(t *testing.T) {
	t.Parallel()

	// preset: minimal + explicit checkpoints: true → core + checkpoints
	cfg := DefaultConfig()
	cfg.MCP.Preset = "minimal"
	cfg.MCP.Groups = map[string]bool{
		GroupCheckpoints: true,
	}

	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("EffectiveGroups() warnings = %v, want none", warnings)
	}

	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true")
	}
	if !groups[GroupCheckpoints] {
		t.Error("groups[checkpoints] = false, want true (explicit override)")
	}
	// planning still off
	if groups[GroupPlanning] {
		t.Error("groups[planning] = true, want false")
	}
}

func TestEffectiveGroups_ExplicitOverrideDisablesGroup(t *testing.T) {
	t.Parallel()

	// preset: full + explicit planning: false → all except planning
	cfg := DefaultConfig()
	cfg.MCP.Preset = "full"
	cfg.MCP.Groups = map[string]bool{
		GroupPlanning: false,
	}

	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}

	if groups[GroupPlanning] {
		t.Error("groups[planning] = true, want false (explicit override)")
	}
	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true")
	}
}

func TestEffectiveGroups_CoreFalseOverriddenWithWarning(t *testing.T) {
	t.Parallel()

	// Setting core: false produces a warning and is silently overridden to true.
	cfg := DefaultConfig()
	cfg.MCP.Preset = "minimal"
	cfg.MCP.Groups = map[string]bool{
		GroupCore: false,
	}

	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}

	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true (core cannot be disabled)")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "core") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a warning about core group, got: %v", warnings)
	}
}

func TestEffectiveGroups_UnknownGroupWarning(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		"nonexistent_group": true,
	}

	groups, warnings, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v, want nil", err)
	}

	// Unknown group should be ignored (not in groups map).
	if groups["nonexistent_group"] {
		t.Error("groups[nonexistent_group] = true, want false (unknown group should be ignored)")
	}

	// A warning should be emitted.
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "nonexistent_group") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a warning about nonexistent_group, got: %v", warnings)
	}
}

func TestEffectiveGroups_EmptyPresetDefaultsFull(t *testing.T) {
	t.Parallel()

	// Empty preset string (zero value) should behave as "full".
	cfg := DefaultConfig()
	cfg.MCP.Preset = ""

	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v", err)
	}

	for _, g := range []string{GroupCore, GroupPlanning, GroupKnowledge, GroupGit, GroupDocuments, GroupIncidents, GroupCheckpoints} {
		if !groups[g] {
			t.Errorf("groups[%q] = false, want true (empty preset → full)", g)
		}
	}
}

func TestEffectiveGroups_MultipleOverrides(t *testing.T) {
	t.Parallel()

	// preset: orchestration (core + planning + git)
	// + knowledge: true, git: false → core + planning + knowledge
	cfg := DefaultConfig()
	cfg.MCP.Preset = "orchestration"
	cfg.MCP.Groups = map[string]bool{
		GroupKnowledge: true,
		GroupGit:       false,
	}

	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		t.Fatalf("EffectiveGroups() error = %v", err)
	}

	if !groups[GroupCore] {
		t.Error("groups[core] = false, want true")
	}
	if !groups[GroupPlanning] {
		t.Error("groups[planning] = false, want true")
	}
	if !groups[GroupKnowledge] {
		t.Error("groups[knowledge] = false, want true (explicit override)")
	}
	if groups[GroupGit] {
		t.Error("groups[git] = true, want false (explicit override)")
	}
}
