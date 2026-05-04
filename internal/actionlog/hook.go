package actionlog

import (
	"context"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandlerFunc is the type signature for MCP tool handlers.
type HandlerFunc func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// Hook wraps MCP tool handlers to log each invocation after it completes.
type Hook struct {
	writer  *Writer
	lookup  StageLookup // may be nil
	version string
}

// NewHook creates a Hook. lookup may be nil if stage resolution is not needed.
// version is stamped as ServerVersion on every log entry.
func NewHook(writer *Writer, lookup StageLookup, version string) *Hook {
	return &Hook{writer: writer, lookup: lookup, version: version}
}

// SideEffectKey is a public context key for the side-effect collector.
// internal/mcp stores its *SideEffectCollector under this key so that
// Hook.Wrap can retrieve it without importing internal/mcp.
var SideEffectKey = &ctxKey{name: "side-effect-collector"}

// SideEffectInspector is satisfied by *internal/mcp.SideEffectCollector.
// It allows counting side effects by type without importing internal/mcp.
type SideEffectInspector interface {
	CountByType(typeName string) int
}

// countRejections inspects the side-effect collector on ctx (if present),
// counts SideEffectKnowledgeRejected events, and returns the count as a
// string if non-zero.
func countRejections(ctx context.Context) string {
	c, ok := ctx.Value(SideEffectKey).(SideEffectInspector)
	if !ok || c == nil {
		return ""
	}
	const rejectedType = "knowledge_rejected"
	n := c.CountByType(rejectedType)
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// Wrap returns a handler that logs after the inner handler completes.
// Log errors are swallowed — they must not affect the tool response.
// It places an annotation collector on the context so handlers can call
// AnnotateEntry; annotations are merged into Entry.Extra before writing.
func (h *Hook) Wrap(toolName string, inner HandlerFunc) HandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		// Extract action parameter.
		var action *string
		if a, ok := args["action"].(string); ok && a != "" {
			action = &a
		}

		// Extract entity ID and resolve stage BEFORE calling the inner handler,
		// so the logged stage reflects the state before any transition the call
		// may perform (FR-005 AC bullet 4).
		entityID := ExtractEntityID(args)
		stage := ResolveStage(entityID, h.lookup)

		// Place annotation collector on context for handlers (FR-004, FR-NF-002).
		annotatedCtx := contextWithCollector(ctx)

		result, err := inner(annotatedCtx, req)

		// Drain annotations collected during the call (FR-006).
		extra := drainCollector(annotatedCtx)

		// Count knowledge rejections from the side-effect collector (FR-010, FR-011).
		if rejectionCount := countRejections(ctx); rejectionCount != "" {
			if extra == nil {
				extra = make(map[string]string)
			}
			extra[AnnotationKBRejections] = rejectionCount
		}

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
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Tool:          toolName,
			Action:        action,
			EntityID:      entityID,
			Stage:         stage,
			ServerVersion: h.version,
			Success:       success,
			ErrorType:     errorType,
			Extra:         extra,
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
