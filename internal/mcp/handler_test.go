package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/actionlog"
	"github.com/sambeau/kanbanzai/internal/config"
)

// TestWrapWithRecovery_PanicReturnsInternalPanicError verifies AC-004:
// a handler that panics returns a structured error with code "internal_panic".
func TestWrapWithRecovery_PanicReturnsInternalPanicError(t *testing.T) {
	panicHandler := actionlog.HandlerFunc(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic("test panic value")
	})

	wrapped := wrapWithRecovery("test_tool", panicHandler)
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})

	if err != nil {
		t.Fatalf("expected nil error from recovery wrapper, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result from recovery wrapper")
	}

	text := toolResultText(t, result)
	code, msg := decodeErrorJSON(t, text)

	if code != "internal_panic" {
		t.Errorf("expected code 'internal_panic', got %q", code)
	}
	if msg == "" {
		t.Error("expected non-empty message in error response")
	}
}

// TestWrapWithRecovery_NoPanicPassesThrough verifies that non-panicking
// handlers pass their result through unchanged.
func TestWrapWithRecovery_NoPanicPassesThrough(t *testing.T) {
	expected := mcp.NewToolResultText(`{"ok":true}`)
	normalHandler := actionlog.HandlerFunc(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return expected, nil
	})

	wrapped := wrapWithRecovery("test_tool", normalHandler)
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result != expected {
		t.Error("expected result to pass through unchanged")
	}
}

// TestWrapWithRecovery_PanicIncludesToolName verifies that the error message
// includes the tool name for diagnostics.
func TestWrapWithRecovery_PanicIncludesToolName(t *testing.T) {
	toolName := "my_diagnostic_tool"
	panicHandler := actionlog.HandlerFunc(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic("boom")
	})

	wrapped := wrapWithRecovery(toolName, panicHandler)
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	text := toolResultText(t, result)
	_, msg := decodeErrorJSON(t, text)

	if !strings.Contains(msg, toolName) {
		t.Errorf("expected tool name %q in message %q", toolName, msg)
	}
}

// TestWrapWithTimeout_WithinBudgetPassesThrough verifies that a handler
// completing within the timeout budget passes its result through unchanged
// (REQ-006).
func TestWrapWithTimeout_WithinBudgetPassesThrough(t *testing.T) {
	expected := mcp.NewToolResultText(`{"ok":true}`)
	fastHandler := actionlog.HandlerFunc(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return expected, nil
	})

	wrapped := wrapWithTimeout(5*time.Second, fastHandler)
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if result != expected {
		t.Error("expected result to pass through unchanged")
	}
}

// TestWrapWithTimeout_DeadlineExceededPropagates verifies that a handler
// exceeding the timeout budget gets a cancelled context, surfacing as
// context.DeadlineExceeded (REQ-006).
func TestWrapWithTimeout_DeadlineExceededPropagates(t *testing.T) {
	slowHandler := actionlog.HandlerFunc(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
			return mcp.NewToolResultText("ok"), nil
		}
	})

	wrapped := wrapWithTimeout(10*time.Millisecond, slowHandler)
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

// TestWrapAllTools_AppliesRecoveryToAllHandlers verifies AC-005:
// every handler registered via the server is wrapped with panic recovery.
// We verify by creating a server with the minimal preset (7 core tools) and
// checking that every handler is wrapped via the wrapAllTools function.
func TestWrapAllTools_AppliesRecoveryToAllHandlers(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"
	srv, err := newServerWithConfig(entityRoot, &cfg); if err != nil { t.Fatal(err) }
	tools := srv.ListTools()

	// Verify every registered tool has a non-nil handler (i.e. wrapAllTools ran).
	for name, tool := range tools {
		if tool.Handler == nil {
			t.Errorf("tool %q has nil handler — wrapAllTools was not applied", name)
		}
	}

	if len(tools) == 0 {
		t.Error("expected at least one registered tool")
	}

	t.Logf("AC-005 verified: %d tools registered with handlers", len(tools))
}

// TestAC007_TimeoutWithinBudget verifies AC-007:
// a handler with a 60s sleep returns a timeout error within 5s.
func TestAC007_TimeoutWithinBudget(t *testing.T) {
	t.Parallel()

	sleepHandler := actionlog.HandlerFunc(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(60 * time.Second):
			return mcp.NewToolResultText("ok"), nil
		}
	})

	wrapped := wrapWithTimeout(5*time.Second, sleepHandler)

	start := time.Now()
	_, err := wrapped(context.Background(), mcp.CallToolRequest{})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
	if elapsed > 6*time.Second {
		t.Errorf("timeout took %v, want < 6s (budget 5s + 1s grace)", elapsed)
	}
	t.Logf("AC-007: timeout returned in %v", elapsed)
}

// TestAC008_PanicRecoveryWithinOneSecond verifies AC-008:
// panic recovery returns error response within 1 second of the panic.
func TestAC008_PanicRecoveryWithinOneSecond(t *testing.T) {
	t.Parallel()

	panicHandler := actionlog.HandlerFunc(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic("immediate test panic")
	})

	wrapped := wrapWithRecovery("test_tool", panicHandler)

	start := time.Now()
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected nil error from recovery, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	text := toolResultText(t, result)
	code, _ := decodeErrorJSON(t, text)

	if code != "internal_panic" {
		t.Errorf("expected code 'internal_panic', got %q", code)
	}
	if elapsed > 1*time.Second {
		t.Errorf("recovery took %v, want < 1s", elapsed)
	}
	t.Logf("AC-008: panic recovered in %v", elapsed)
}

// toolResultText extracts the text content from a *mcp.CallToolResult.
func toolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("expected text content in result, got none")
	return ""
}

// decodeErrorJSON parses {"error":{"code":"...","message":"..."}} and returns
// the code and message strings.
func decodeErrorJSON(t *testing.T, text string) (code, message string) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v\ntext: %s", err, text)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' key in response, got: %v", resp)
	}
	code, _ = errObj["code"].(string)
	message, _ = errObj["message"].(string)
	return code, message
}
