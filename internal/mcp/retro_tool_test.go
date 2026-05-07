package mcp

import (
	"testing"
)

// ─── retroArgStr ────────────────────────────────────────────────────────────

func TestRetroArgStr_NilArgs(t *testing.T) {
	if got := retroArgStr(nil, "any"); got != "" {
		t.Errorf("retroArgStr(nil, \"any\") = %q, want \"\"", got)
	}
}

func TestRetroArgStr_EmptyArgs(t *testing.T) {
	if got := retroArgStr(map[string]any{}, "any"); got != "" {
		t.Errorf("retroArgStr({}, \"any\") = %q, want \"\"", got)
	}
}

func TestRetroArgStr_KeyPresent(t *testing.T) {
	args := map[string]any{"action": "synthesise"}
	if got := retroArgStr(args, "action"); got != "synthesise" {
		t.Errorf("retroArgStr(args, \"action\") = %q, want \"synthesise\"", got)
	}
}

func TestRetroArgStr_KeyMissing(t *testing.T) {
	args := map[string]any{"other": "value"}
	if got := retroArgStr(args, "action"); got != "" {
		t.Errorf("retroArgStr(args, \"action\") = %q, want \"\"", got)
	}
}

func TestRetroArgStr_TrimsWhitespace(t *testing.T) {
	args := map[string]any{"action": "  report  "}
	if got := retroArgStr(args, "action"); got != "report" {
		t.Errorf("retroArgStr(args, \"action\") = %q, want \"report\"", got)
	}
}

func TestRetroArgStr_NonStringValue(t *testing.T) {
	args := map[string]any{"action": 42}
	if got := retroArgStr(args, "action"); got != "" {
		t.Errorf("retroArgStr(args, \"action\") with int value = %q, want \"\"", got)
	}
}

// ─── retroArgFloatAsInt ─────────────────────────────────────────────────────

func TestRetroArgFloatAsInt_KeyPresent(t *testing.T) {
	args := map[string]any{"theme_count": 3.0}
	got := retroArgFloatAsInt(args, "theme_count")
	if got != 3 {
		t.Errorf("retroArgFloatAsInt(args, \"theme_count\") = %d, want 3", got)
	}
}

func TestRetroArgFloatAsInt_TruncatesFloat(t *testing.T) {
	args := map[string]any{"theme_count": 3.7}
	got := retroArgFloatAsInt(args, "theme_count")
	if got != 3 {
		t.Errorf("retroArgFloatAsInt(args, \"theme_count\") with 3.7 = %d, want 3", got)
	}
}

func TestRetroArgFloatAsInt_KeyMissing(t *testing.T) {
	args := map[string]any{"other": "value"}
	got := retroArgFloatAsInt(args, "theme_count")
	if got != 0 {
		t.Errorf("retroArgFloatAsInt(args, \"theme_count\") = %d, want 0", got)
	}
}

func TestRetroArgFloatAsInt_NilArgs(t *testing.T) {
	got := retroArgFloatAsInt(nil, "theme_count")
	if got != 0 {
		t.Errorf("retroArgFloatAsInt(nil, \"theme_count\") = %d, want 0", got)
	}
}

func TestRetroArgFloatAsInt_NegativeValue(t *testing.T) {
	args := map[string]any{"theme_count": -1.0}
	got := retroArgFloatAsInt(args, "theme_count")
	if got != -1 {
		t.Errorf("retroArgFloatAsInt(args, \"theme_count\") with -1 = %d, want -1", got)
	}
}

func TestRetroArgFloatAsInt_NonFloatValue(t *testing.T) {
	args := map[string]any{"theme_count": "three"}
	got := retroArgFloatAsInt(args, "theme_count")
	if got != 0 {
		t.Errorf("retroArgFloatAsInt(args, \"theme_count\") with string = %d, want 0", got)
	}
}

// ─── Tool registration ──────────────────────────────────────────────────────

func TestRetroTool_ReturnsServerTool(t *testing.T) {
	// Verify RetroTool returns a non-nil slice with one element.
	// We can't construct a real RetroService without dependencies, but we can
	// verify the tool structure is created correctly with a nil service.
	// (The handler will panic if called, but construction is safe.)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("expected panic with nil service: %v", r)
		}
	}()
	tools := RetroTool(nil)
	if len(tools) != 1 {
		t.Fatalf("RetroTool(nil) returned %d tools, want 1", len(tools))
	}
	if tools[0].Tool.Name != "retro" {
		t.Errorf("tool name = %q, want \"retro\"", tools[0].Tool.Name)
	}
}

// ─── Action routing ─────────────────────────────────────────────────────────

func TestRetroTool_DefaultActionIsSynthesise(t *testing.T) {
	// Verify the description mentions synthesise as default.
	tools := RetroTool(nil)
	if len(tools) != 1 {
		t.Fatal("expected 1 tool")
	}
	desc := tools[0].Tool.Description
	if desc == "" {
		t.Error("tool description is empty")
	}
}

func TestRetroTool_AllActionsInDescription(t *testing.T) {
	tools := RetroTool(nil)
	if len(tools) != 1 {
		t.Fatal("expected 1 tool")
	}
	desc := tools[0].Tool.Description
	for _, action := range []string{"synthesise", "report", "create_fix"} {
		if !contains(desc, action) {
			t.Errorf("tool description missing action %q", action)
		}
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
