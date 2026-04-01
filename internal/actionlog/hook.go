package actionlog

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandlerFunc is the type signature for MCP tool handlers.
type HandlerFunc func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// Hook wraps MCP tool handlers to log each invocation after it completes.
type Hook struct {
	writer *Writer
	lookup StageLookup // may be nil
}

// NewHook creates a Hook. lookup may be nil if stage resolution is not needed.
func NewHook(writer *Writer, lookup StageLookup) *Hook {
	return &Hook{writer: writer, lookup: lookup}
}

// Wrap returns a handler that logs after the inner handler completes.
// Log errors are swallowed — they must not affect the tool response.
func (h *Hook) Wrap(toolName string, inner HandlerFunc) HandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := inner(ctx, req)

		args, _ := req.Params.Arguments.(map[string]any)

		// Extract action parameter.
		var action *string
		if a, ok := args["action"].(string); ok && a != "" {
			action = &a
		}

		// Extract entity ID.
		entityID := ExtractEntityID(args)

		// Resolve stage (never blocks or fails the handler).
		stage := ResolveStage(entityID, h.lookup)

		// Determine success and error type.
		success := err == nil && !isErrorResult(result)
		var errorType *string
		if !success {
			var classified string
			if err != nil {
				classified = ClassifyError(err)
			} else {
				classified = ErrorInternalError
			}
			errorType = &classified
		}

		e := Entry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tool:      toolName,
			Action:    action,
			EntityID:  entityID,
			Stage:     stage,
			Success:   success,
			ErrorType: errorType,
		}

		// Log is best-effort — errors are discarded per FR-018.
		_ = h.writer.Log(e)

		return result, err
	}
}

// isErrorResult inspects the tool result text for a top-level "error" key.
// This covers the case where the handler returns nil error but an error response.
func isErrorResult(result *mcp.CallToolResult) bool {
	if result == nil {
		return false
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			// Quick scan — does not need full JSON parsing.
			s := tc.Text
			for i := 0; i < len(s)-8; i++ {
				if s[i] == '"' && s[i+1] == 'e' && s[i+2] == 'r' &&
					s[i+3] == 'r' && s[i+4] == 'o' && s[i+5] == 'r' && s[i+6] == '"' {
					return true
				}
			}
		}
	}
	return false
}
