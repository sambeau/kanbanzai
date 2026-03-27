package mcp

import (
	"sort"
	"testing"

	"kanbanzai/internal/config"
)

// TestGroupToolNames_MembershipMap verifies that every group has a non-empty tool list
// and that no tool name appears in more than one group.
// Verifies §30.1: group membership is well-defined and non-overlapping.
func TestGroupToolNames_MembershipMap(t *testing.T) {
	t.Parallel()

	// Every known group must appear in the map.
	for g := range config.KnownGroups {
		tools, ok := GroupToolNames[g]
		if !ok {
			t.Errorf("group %q missing from GroupToolNames", g)
			continue
		}
		if len(tools) == 0 {
			t.Errorf("group %q has no tools in GroupToolNames", g)
		}
	}

	// No tool name should appear in more than one group.
	seen := make(map[string]string) // tool name → group name
	for group, tools := range GroupToolNames {
		for _, tool := range tools {
			if prev, dup := seen[tool]; dup {
				t.Errorf("tool %q appears in both group %q and group %q", tool, prev, group)
			}
			seen[tool] = group
		}
	}
}

// TestGroupToolNames_CoreGroup verifies the core group contains exactly the expected tools.
func TestGroupToolNames_CoreGroup(t *testing.T) {
	t.Parallel()

	want := []string{"status", "next", "finish", "handoff", "entity", "doc", "health"}
	got := append([]string(nil), GroupToolNames[config.GroupCore]...)

	sort.Strings(want)
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("core group: got %d tools %v, want %d tools %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("core group tool[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestGroupToolNames_TotalToolCount verifies the total 2.0 tool count is 20 (spec §6.4).
func TestGroupToolNames_TotalToolCount(t *testing.T) {
	t.Parallel()

	total := 0
	for _, tools := range GroupToolNames {
		total += len(tools)
	}
	const want = 21
	if total != want {
		t.Errorf("total 2.0 tool count = %d, want %d", total, want)
	}
}

// TestResolveServerGroups_CoreAlwaysEnabled verifies the core group is always present.
func TestResolveServerGroups_CoreAlwaysEnabled(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	groups := resolveServerGroups(&cfg)

	if !groups[config.GroupCore] {
		t.Error("resolveServerGroups: core group not enabled for minimal preset")
	}
}

// TestResolveServerGroups_FullPreset verifies all groups are enabled under full preset.
func TestResolveServerGroups_FullPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"
	groups := resolveServerGroups(&cfg)

	for g := range config.KnownGroups {
		if !groups[g] {
			t.Errorf("resolveServerGroups: group %q not enabled for full preset", g)
		}
	}
}

// TestResolveServerGroups_MinimalPreset verifies only core (+ legacy) is enabled.
func TestResolveServerGroups_MinimalPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	groups := resolveServerGroups(&cfg)

	if !groups[config.GroupCore] {
		t.Error("groups[core] = false, want true")
	}
	for _, g := range []string{
		config.GroupPlanning, config.GroupKnowledge, config.GroupGit,
		config.GroupDocuments, config.GroupIncidents, config.GroupCheckpoints,
	} {
		if groups[g] {
			t.Errorf("groups[%q] = true, want false (minimal preset)", g)
		}
	}
}

// TestResolveServerGroups_OrchestrationPreset verifies core, planning, git are enabled.
func TestResolveServerGroups_OrchestrationPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "orchestration"
	groups := resolveServerGroups(&cfg)

	for _, g := range []string{config.GroupCore, config.GroupPlanning, config.GroupGit} {
		if !groups[g] {
			t.Errorf("groups[%q] = false, want true (orchestration preset)", g)
		}
	}
	for _, g := range []string{config.GroupKnowledge, config.GroupDocuments, config.GroupIncidents, config.GroupCheckpoints} {
		if groups[g] {
			t.Errorf("groups[%q] = true, want false (orchestration preset)", g)
		}
	}
}

// TestResolveServerGroups_UnknownPresetFallbackToFull verifies that an unknown preset
// degrades gracefully to full instead of crashing.
func TestResolveServerGroups_UnknownPresetFallbackToFull(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "nonexistent_preset"
	groups := resolveServerGroups(&cfg)

	// Must not panic; must return a usable group map with at least core.
	if !groups[config.GroupCore] {
		t.Error("resolveServerGroups: core not present after unknown-preset fallback")
	}
}

// TestResolveServerGroups_ExplicitOverrideApplied verifies explicit group overrides
// take effect after preset resolution.
func TestResolveServerGroups_ExplicitOverrideApplied(t *testing.T) {
	t.Parallel()

	// preset: minimal + checkpoints: true → core + checkpoints (+ legacy)
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	cfg.MCP.Groups = map[string]bool{
		config.GroupCheckpoints: true,
	}
	groups := resolveServerGroups(&cfg)

	if !groups[config.GroupCore] {
		t.Error("groups[core] = false, want true")
	}
	if !groups[config.GroupCheckpoints] {
		t.Error("groups[checkpoints] = false, want true (explicit override)")
	}
	if groups[config.GroupPlanning] {
		t.Error("groups[planning] = true, want false")
	}
}

// TestResolveServerGroups_CoreCannotBeDisabled verifies core stays enabled even when
// the config explicitly sets core: false.
func TestResolveServerGroups_CoreCannotBeDisabled(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore: false,
	}
	groups := resolveServerGroups(&cfg)

	if !groups[config.GroupCore] {
		t.Error("groups[core] = false, want true (core cannot be disabled)")
	}
}
