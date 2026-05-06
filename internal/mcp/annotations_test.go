package mcp

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

func TestAllToolsHaveAnnotations(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"

	mcpSrv := newServerWithConfig(entityRoot, &cfg)
	tools := mcpSrv.ListTools()

	for name, st := range tools {
		ann := st.Tool.Annotations
		if ann.ReadOnlyHint == nil {
			t.Errorf("tool %q: ReadOnlyHint is nil", name)
		}
		if ann.DestructiveHint == nil {
			t.Errorf("tool %q: DestructiveHint is nil", name)
		}
		if ann.IdempotentHint == nil {
			t.Errorf("tool %q: IdempotentHint is nil", name)
		}
		if ann.OpenWorldHint == nil {
			t.Errorf("tool %q: OpenWorldHint is nil", name)
		}
	}
}

func TestToolAnnotationTiers(t *testing.T) {
	t.Parallel()

	tier1 := map[string]bool{
		"status": true, "health": true, "handoff": true,
		"conflict": true, "server_info": true,
	}
	tier3 := map[string]bool{
		"cleanup": true, "merge": true, "pr": true, "worktree": true,
	}

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"

	mcpSrv := newServerWithConfig(entityRoot, &cfg)
	tools := mcpSrv.ListTools()

	for name, st := range tools {
		ann := st.Tool.Annotations
		if ann.ReadOnlyHint == nil || ann.DestructiveHint == nil {
			continue // already caught by previous test
		}
		if tier1[name] && !*ann.ReadOnlyHint {
			t.Errorf("tier1 tool %q: ReadOnlyHint should be true", name)
		}
		if tier3[name] && !(*ann.DestructiveHint || *ann.OpenWorldHint) {
			t.Errorf("tier3 tool %q: DestructiveHint or OpenWorldHint should be true", name)
		}
	}
}
