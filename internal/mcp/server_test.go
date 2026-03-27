package mcp

import (
	"sort"
	"testing"

	"kanbanzai/internal/config"
)

// testServerToolNames creates a 2.0 MCP server with the given config and
// returns the set of registered tool names. Uses the unexported
// newServerWithConfig so the test exercises the real registration path.
func testServerToolNames(t *testing.T, cfg *config.Config) map[string]bool {
	t.Helper()
	entityRoot := t.TempDir()
	srv := newServerWithConfig(entityRoot, cfg)
	tools := srv.ListTools()
	names := make(map[string]bool, len(tools))
	for name := range tools {
		names[name] = true
	}
	return names
}

// TestServer_ListTools_MinimalPreset verifies that preset: minimal registers
// exactly the 7 core-group tools (spec §30.11).
func TestServer_ListTools_MinimalPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	tools := testServerToolNames(t, &cfg)

	wantTools := []string{
		"status", "next", "finish", "handoff", "entity", "doc", "health",
	}

	if len(tools) != len(wantTools) {
		got := testSortedKeys(tools)
		t.Fatalf("minimal preset: got %d tools %v, want %d tools %v",
			len(tools), got, len(wantTools), wantTools)
	}

	for _, name := range wantTools {
		if !tools[name] {
			t.Errorf("minimal preset: missing tool %q", name)
		}
	}
}

// TestServer_ListTools_OrchestrationPreset verifies that preset: orchestration
// registers exactly 16 tools: core + planning + git (spec §30.11).
func TestServer_ListTools_OrchestrationPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "orchestration"
	tools := testServerToolNames(t, &cfg)

	wantTools := []string{
		// core (7)
		"status", "next", "finish", "handoff", "entity", "doc", "health",
		// planning (4)
		"decompose", "estimate", "conflict", "retro",
		// git (5)
		"worktree", "merge", "pr", "branch", "cleanup",
	}

	if len(tools) != len(wantTools) {
		got := testSortedKeys(tools)
		t.Fatalf("orchestration preset: got %d tools %v, want %d tools %v",
			len(tools), got, len(wantTools), wantTools)
	}

	for _, name := range wantTools {
		if !tools[name] {
			t.Errorf("orchestration preset: missing tool %q", name)
		}
	}
}

// TestServer_ListTools_FullPreset verifies that preset: full registers
// exactly 21 tools — all groups (spec §30.11).
func TestServer_ListTools_FullPreset(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"
	tools := testServerToolNames(t, &cfg)

	wantTools := []string{
		// core (7)
		"status", "next", "finish", "handoff", "entity", "doc", "health",
		// planning (4)
		"decompose", "estimate", "conflict", "retro",
		// knowledge (2)
		"knowledge", "profile",
		// git (5)
		"worktree", "merge", "pr", "branch", "cleanup",
		// documents (1)
		"doc_intel",
		// incidents (1)
		"incident",
		// checkpoints (1)
		"checkpoint",
	}

	if len(tools) != 21 {
		got := testSortedKeys(tools)
		t.Fatalf("full preset: got %d tools %v, want 21", len(tools), got)
	}

	for _, name := range wantTools {
		if !tools[name] {
			t.Errorf("full preset: missing tool %q", name)
		}
	}
}

// TestServer_ListTools_DefaultConfigIsFull verifies that when no mcp section
// is configured, the default preset is "full" and all 20 tools are registered.
func TestServer_ListTools_DefaultConfigIsFull(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	tools := testServerToolNames(t, &cfg)

	if len(tools) != 21 {
		got := testSortedKeys(tools)
		t.Fatalf("default config: got %d tools %v, want 21", len(tools), got)
	}
}

// TestServer_ListTools_NoLegacyTools verifies that no 1.0 legacy tools are
// registered — the _legacy group has been removed (Track K).
func TestServer_ListTools_NoLegacyTools(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"
	tools := testServerToolNames(t, &cfg)

	legacyNames := []string{
		"create_epic", "create_feature", "create_task", "create_bug",
		"record_decision", "get_entity", "list_entities",
		"update_status", "update_entity", "validate_candidate",
		"health_check", "rebuild_cache",
		"estimate_set", "estimate_query", "estimate_reference_add", "estimate_reference_remove",
		"work_queue", "dependency_status", "dispatch_task", "complete_task",
		"human_checkpoint", "human_checkpoint_respond",
		"human_checkpoint_get", "human_checkpoint_list",
		"incident_create", "incident_update", "incident_list", "incident_link_bug",
		"decompose_feature", "decompose_review", "slice_analysis",
		"review_task_output", "conflict_domain_check",
	}

	for _, name := range legacyNames {
		if tools[name] {
			t.Errorf("legacy tool %q should not be registered after Track K removal", name)
		}
	}
}

// TestServer_ListTools_ExplicitGroupOverride verifies that explicit group
// overrides extend the preset (e.g., minimal + checkpoints: true).
func TestServer_ListTools_ExplicitGroupOverride(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	cfg.MCP.Groups = map[string]bool{
		config.GroupCheckpoints: true,
	}
	tools := testServerToolNames(t, &cfg)

	// Should have core (7) + checkpoints (1) = 8.
	if len(tools) != 8 {
		got := testSortedKeys(tools)
		t.Fatalf("minimal+checkpoints: got %d tools %v, want 8", len(tools), got)
	}

	if !tools["checkpoint"] {
		t.Error("checkpoint tool missing after explicit group override")
	}
}

func testSortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
