package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/actionlog"
)

// wrapWithTimeout wraps an MCP handler with a per-tool timeout budget.
// It creates a child context with the given timeout deadline. If the handler
// does not return before the deadline, context.DeadlineExceeded propagates
// through the handler chain (REQ-006).
func wrapWithTimeout(timeout time.Duration, inner actionlog.HandlerFunc) actionlog.HandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return inner(ctx, req)
	}
}

// wrapWithRecovery wraps an MCP handler with panic recovery. On panic, the
// panic value is caught, logged to stderr, and returned as a structured
// JSON error with code "internal_panic". This prevents silent timeouts when a
// handler crashes (REQ-004/REQ-005).
func wrapWithRecovery(toolName string, inner actionlog.HandlerFunc) actionlog.HandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (result *mcp.CallToolResult, retErr error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("PANIC recovered", "component", toolName, "panic", r)
				result = ActionError("internal_panic", fmt.Sprintf(
					"Tool %q panicked: %v. Report this as a bug with the tool name and server stderr log.", toolName, r), nil)
				retErr = nil
			}
		}()
		return inner(ctx, req)
	}
}
