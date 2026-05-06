package mcp

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

// TestToolDescriptions_TokenBudget verifies that every registered tool
// description fits within the 200-token budget (≤ 800 characters, using the
// standard ceil(len/4) approximation).
//
// The test does NOT fail-fast: it collects every violation and reports them all
// so that a single run surfaces all over-budget descriptions at once.
func TestToolDescriptions_TokenBudget(t *testing.T) {
	t.Parallel()

	const (
		maxTokens     = 200
		charsPerToken = 4
		maxChars      = maxTokens * charsPerToken // 800
	)

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"

	mcpSrv := newServerWithConfig(entityRoot, &cfg)
	tools := mcpSrv.ListTools()

	// Sort tool names for deterministic output.
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)

	type violation struct {
		name   string
		chars  int
		tokens int
	}
	var violations []violation

	for _, name := range names {
		st := tools[name]
		desc := st.Tool.Description
		chars := len(desc)
		tokens := (chars + charsPerToken - 1) / charsPerToken // ceil(chars/4)

		if chars > maxChars {
			violations = append(violations, violation{
				name:   name,
				chars:  chars,
				tokens: tokens,
			})
		}
	}

	if len(violations) == 0 {
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d tool(s) exceed the %d-token description budget:\n\n", len(violations), maxTokens)
	for _, v := range violations {
		fmt.Fprintf(&sb, "  tool %q: description is %d tokens (%d chars, limit %d chars / %d tokens)\n",
			v.name, v.tokens, v.chars, maxChars, maxTokens)
	}
	t.Error(sb.String())
}
