// Package mcp sideeffect.go — side-effect reporting infrastructure for Kanbanzai 2.0 (Track B).
//
// Every 2.0 mutation tool returns a side_effects field listing cascades that
// occurred as a result of the requested operation (dependency unblocking,
// automatic worktree creation, feature lifecycle advances from document
// approval, etc.).
//
// Usage in a 2.0 tool handler:
//
//	func myToolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
//	    result, err := entitySvc.UpdateStatus(...)
//	    if err != nil {
//	        return ActionError("update_failed", err.Error(), nil), nil
//	    }
//	    // Convert service hook results to side effects.
//	    for _, t := range result.UnblockedTasks {
//	        PushSideEffect(ctx, SideEffect{
//	            Type:       SideEffectTaskUnblocked,
//	            EntityID:   t.TaskID,
//	            EntityType: "task",
//	            ToStatus:   t.Status,
//	        })
//	    }
//	    // ... build response ...
//	}
//
// The WithSideEffects middleware wrapper drains the collector after the
// handler returns and appends the side_effects array to the JSON response.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// ─── Side-effect types ──────────────────────────────────────────────────────

// SideEffectType is a machine-readable identifier for a cascade event.
type SideEffectType = string

const (
	// SideEffectStatusTransition reports that an entity's lifecycle status changed
	// as a cascade of the requested operation (e.g. feature advanced because a
	// spec was approved).
	SideEffectStatusTransition SideEffectType = "status_transition"

	// SideEffectTaskUnblocked reports that a task was promoted from queued/blocked
	// to ready because all of its dependencies reached a terminal state.
	SideEffectTaskUnblocked SideEffectType = "task_unblocked"

	// SideEffectWorktreeCreated reports that a Git worktree was automatically
	// created as a result of a task or bug status transition.
	SideEffectWorktreeCreated SideEffectType = "worktree_created"

	// SideEffectKnowledgeContributed reports that a knowledge entry was accepted
	// and stored (e.g. from an inline contribution in finish).
	SideEffectKnowledgeContributed SideEffectType = "knowledge_contributed"

	// SideEffectKnowledgeRejected reports that a knowledge entry was rejected
	// (duplicate or validation failure) during an inline contribution attempt.
	SideEffectKnowledgeRejected SideEffectType = "knowledge_rejected"

	// SideEffectRetroSignalContributed reports that a retrospective signal was
	// accepted and stored as a knowledge entry (e.g. from the retrospective
	// parameter in finish). See work/spec/workflow-retrospective.md §6.2.
	SideEffectRetroSignalContributed SideEffectType = "retrospective_signal_contributed"
)

// SideEffect describes a single cascade event that occurred as a result of
// a requested operation. Side effects are informational — the operation
// succeeded regardless of whether side effects were produced.
type SideEffect struct {
	// Type is the machine-readable event type (see SideEffect* constants).
	Type SideEffectType `json:"type"`

	// EntityID is the ID of the entity affected by the cascade.
	EntityID string `json:"entity_id,omitempty"`

	// EntityType is the type of the affected entity (task, feature, bug, …).
	EntityType string `json:"entity_type,omitempty"`

	// FromStatus is the status before the cascade transition (status_transition only).
	FromStatus string `json:"from_status,omitempty"`

	// ToStatus is the status after the cascade transition.
	ToStatus string `json:"to_status,omitempty"`

	// Trigger is a human-readable description of what caused the cascade.
	Trigger string `json:"trigger,omitempty"`

	// Extra holds type-specific additional fields (e.g. worktree path/branch,
	// knowledge entry ID, rejection reason). Omitted when empty.
	Extra map[string]string `json:"extra,omitempty"`
}

// ─── Side-effect collector ───────────────────────────────────────────────────

// SideEffectCollector accumulates side effects produced during a single MCP
// request. It is goroutine-safe.
type SideEffectCollector struct {
	mu         sync.Mutex
	effects    []SideEffect
	isMutation bool // set via SignalMutation; controls side_effects:[] in responses
}

// Push appends a side effect to the collector.
func (c *SideEffectCollector) Push(e SideEffect) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.effects = append(c.effects, e)
}

// Drain removes and returns all collected side effects. The collector is
// empty after this call.
func (c *SideEffectCollector) Drain() []SideEffect {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.effects
	c.effects = nil
	return out
}

// SetMutation marks this request as a mutation so that WithSideEffects
// always includes side_effects: [] in the response, even when no cascades
// occurred (spec §8.4: "The field is never omitted").
func (c *SideEffectCollector) SetMutation() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isMutation = true
}

// IsMutation reports whether this request was signalled as a mutation.
func (c *SideEffectCollector) IsMutation() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isMutation
}

// Len returns the number of side effects currently in the collector.
func (c *SideEffectCollector) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.effects)
}

// ─── Context key and helpers ─────────────────────────────────────────────────

// collectorKey is the unexported context key for the SideEffectCollector.
type collectorKey struct{}

// ContextWithCollector returns a new context carrying the given collector.
func ContextWithCollector(ctx context.Context, c *SideEffectCollector) context.Context {
	return context.WithValue(ctx, collectorKey{}, c)
}

// CollectorFromContext retrieves the SideEffectCollector from the context.
// Returns nil if no collector is present (e.g. CLI or test contexts).
func CollectorFromContext(ctx context.Context) *SideEffectCollector {
	c, _ := ctx.Value(collectorKey{}).(*SideEffectCollector)
	return c
}

// PushSideEffect pushes a side effect onto the collector in the context.
// It is a no-op if the context carries no collector.
func PushSideEffect(ctx context.Context, e SideEffect) {
	if c := CollectorFromContext(ctx); c != nil {
		c.Push(e)
	}
}

// SignalMutation marks the current request as a mutation so that the
// response always includes side_effects: [] even when no cascades occur.
// Mutation tool actions (create, update, transition) call this at the start
// of their handler body (spec §8.4: "The field is never omitted").
// It is a no-op if the context carries no collector.
func SignalMutation(ctx context.Context) {
	if c := CollectorFromContext(ctx); c != nil {
		c.SetMutation()
	}
}

// ─── Middleware ──────────────────────────────────────────────────────────────

// sideEffectHandler is the type of the inner handler function that returns
// a JSON-serialisable result alongside any direct error.
// The outer WithSideEffects wrapper appends side_effects to the result.
type sideEffectHandler func(ctx context.Context, req mcp.CallToolRequest) (any, error)

// WithSideEffects wraps a tool handler with the side-effect collector lifecycle:
//  1. Creates a fresh SideEffectCollector for the request.
//  2. Attaches it to the context.
//  3. Calls the inner handler (which should use PushSideEffect to record cascades).
//  4. Drains the collector and, if non-empty, merges the side_effects field
//     into the JSON result before returning the CallToolResult.
//
// The inner handler returns (any, error).  If it returns an error the error
// is converted to an ActionError response; the side effects collected before
// the error are still included.
//
// Usage:
//
//	mcp.NewTool("my_tool", ...).WithHandler(WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
//	    // ... handler body ...
//	}))
func WithSideEffects(inner sideEffectHandler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		collector := &SideEffectCollector{}
		ctx = ContextWithCollector(ctx, collector)

		result, err := inner(ctx, req)

		effects := collector.Drain()
		isMutation := collector.IsMutation()

		if err != nil {
			// Use the specific error code for batch limit violations (spec §9.5).
			// All other errors fall back to the generic internal_error code.
			var limitErr *BatchLimitError
			if errors.As(err, &limitErr) {
				return buildErrorResult("batch_limit_exceeded", err.Error(), nil, effects), nil
			}
			return buildErrorResult("internal_error", err.Error(), nil, effects), nil
		}

		return buildResult(result, effects, isMutation), nil
	}
}

// buildResult serialises result to JSON, optionally merging in side_effects.
// If result is already a map, side_effects is added as a field. Otherwise the
// result is wrapped in an envelope with both "data" and "side_effects" fields.
//
// When isMutation is true (signalled via SignalMutation), side_effects: [] is
// always present even when no cascades occurred (spec §8.4: "The field is
// never omitted"). Read-only handlers that do not call SignalMutation omit
// the field entirely (spec §8.5).
//
// Special case: *BatchResult already carries its own top-level side_effects
// field. Injecting a second side_effects key would produce duplicate JSON keys,
// causing parsers to discard the real effects. Batch results are returned
// directly after ensuring side_effects is non-nil for mutations (spec §8.4).
func buildResult(result any, effects []SideEffect, isMutation bool) *mcp.CallToolResult {
	// Batch results manage their own side_effects field via their struct tag.
	// We must not inject a second side_effects key — that produces duplicate keys
	// in JSON, causing parsers to discard the real effects (F2).
	//
	// Strategy: marshal the BatchResult as-is (omitempty omits nil SideEffects),
	// then inject side_effects:[] only when the field was absent and this is a
	// mutation (spec §8.4: "The field is never omitted" for mutations).
	if br, ok := result.(*BatchResult); ok {
		b, err := json.Marshal(br)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf(`{"error":{"code":"serialisation_error","message":%q}}`, err.Error()))
		}
		text := string(b)
		// SideEffects == nil means omitempty removed it from the JSON.
		// Inject [] for mutations so callers can read side_effects unconditionally.
		if isMutation && br.SideEffects == nil {
			text = strings.TrimSuffix(text, "}") + `,"side_effects":[]}`
		}
		return mcp.NewToolResultText(text)
	}

	if result == nil && len(effects) == 0 && !isMutation {
		return mcp.NewToolResultText("{}")
	}

	// Fast path: no side effects and not a mutation — serialise directly.
	// Read-only operations do not include a side_effects field (spec §8.5).
	if len(effects) == 0 && !isMutation {
		b, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf(`{"error":{"code":"serialisation_error","message":%q}}`, err.Error()))
		}
		return mcp.NewToolResultText(string(b))
	}

	// Mutation with no cascades: use a non-nil empty slice so json.Marshal
	// produces [] not null (spec §8.4: "The field is never omitted").
	if len(effects) == 0 {
		effects = []SideEffect{}
	}

	// Merge side_effects into the result object when possible.
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error":{"code":"serialisation_error","message":%q}}`, err.Error()))
	}

	effectsBytes, err := json.Marshal(effects)
	if err != nil {
		return mcp.NewToolResultText(string(resultBytes))
	}

	// Attempt to inject side_effects into an existing JSON object.
	trimmed := strings.TrimSpace(string(resultBytes))
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		// Inject before the closing brace.
		inner := strings.TrimSuffix(trimmed, "}")
		if inner == "{" {
			// Empty object — just add side_effects.
			merged := fmt.Sprintf(`{"side_effects":%s}`, string(effectsBytes))
			return mcp.NewToolResultText(merged)
		}
		merged := fmt.Sprintf(`%s,"side_effects":%s}`, inner, string(effectsBytes))
		return mcp.NewToolResultText(merged)
	}

	// Non-object result: wrap in envelope.
	merged := fmt.Sprintf(`{"data":%s,"side_effects":%s}`, trimmed, string(effectsBytes))
	return mcp.NewToolResultText(merged)
}

// ─── Action dispatcher ───────────────────────────────────────────────────────

// ActionHandler is a function that handles a specific action within a consolidated tool.
// It receives the context (with side-effect collector attached) and the full request.
// It returns a JSON-serialisable result or an error.
type ActionHandler func(ctx context.Context, req mcp.CallToolRequest) (any, error)

// DispatchAction routes the "action" parameter to the appropriate handler.
// If the action is unrecognised, it returns an unknown_action error listing
// the valid actions for the tool. This is the standard dispatch pattern for
// all 2.0 consolidated tools.
//
// Usage:
//
//	handlers := map[string]ActionHandler{
//	    "create": handleCreate,
//	    "get":    handleGet,
//	    "list":   handleList,
//	}
//	return DispatchAction(ctx, req, handlers)
func DispatchAction(ctx context.Context, req mcp.CallToolRequest, handlers map[string]ActionHandler) (any, error) {
	args, _ := req.Params.Arguments.(map[string]any)
	action, _ := args["action"].(string)
	if action == "" {
		valid := sortedKeys(handlers)
		return nil, fmt.Errorf("missing required parameter \"action\"; valid actions: %s", strings.Join(valid, ", "))
	}

	handler, ok := handlers[action]
	if !ok {
		valid := sortedKeys(handlers)
		return nil, &UnknownActionError{Action: action, ValidActions: valid}
	}

	return handler(ctx, req)
}

// UnknownActionError is returned when an unrecognised action is dispatched.
// It carries the structured error code and message for the standard error shape.
type UnknownActionError struct {
	Action       string
	ValidActions []string
}

// Error implements the error interface.
func (e *UnknownActionError) Error() string {
	return fmt.Sprintf("unknown action %q; valid actions: %s", e.Action, strings.Join(e.ValidActions, ", "))
}

// ─── Error response shape ────────────────────────────────────────────────────

// ErrorResponse is the standard error structure returned by all 2.0 tools
// when an operation fails. It is serialised into the CallToolResult text.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
	// SideEffects lists any cascades that occurred before the failure, if any.
	SideEffects []SideEffect `json:"side_effects,omitempty"`
}

// ErrorDetail holds the machine-readable code and human-readable message.
type ErrorDetail struct {
	// Code is a machine-readable error identifier (e.g. "not_found", "invalid_action").
	Code string `json:"code"`
	// Message is a human-readable description of the error.
	Message string `json:"message"`
	// Details provides additional context (e.g. valid actions, current status).
	Details map[string]any `json:"details,omitempty"`
}

// ActionError creates a *mcp.CallToolResult representing a structured tool
// error in the 2.0 error shape. The result is NOT marked IsError — it is
// returned as a successful MCP call with an error payload, which gives the
// agent structured information to act on.
func ActionError(code, message string, details map[string]any) *mcp.CallToolResult {
	return buildErrorResult(code, message, details, nil)
}

// buildErrorResult constructs the full error result including any side effects.
func buildErrorResult(code, message string, details map[string]any, effects []SideEffect) *mcp.CallToolResult {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	if len(effects) > 0 {
		resp.SideEffects = effects
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`{"error":{"code":"serialisation_error","message":%q}}`, err.Error()))
	}
	return mcp.NewToolResultText(string(b))
}

// ─── Inline error helper ─────────────────────────────────────────────────────

// inlineErr returns a structured error response as an (any, error) pair suitable
// for use inside WithSideEffects action handlers. Unlike ActionError (which
// returns *mcp.CallToolResult and must only be used as the outer handler result),
// inlineErr returns a serialisable map that WithSideEffects will embed correctly
// into the JSON response — including any collected side effects.
//
// Usage inside an ActionHandler:
//
//	return inlineErr("missing_parameter", "feature_id is required")
func inlineErr(code, message string) (any, error) {
	return map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}, nil
}

// ─── Utilities ───────────────────────────────────────────────────────────────

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]ActionHandler) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — maps are small (< 20 actions per tool).
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
