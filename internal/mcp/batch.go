// Package mcp batch.go — batch operations infrastructure for Kanbanzai 2.0 (Track C).
//
// Every 2.0 tool that accepts a single entity ID also accepts an array of IDs.
// Single-item calls return the single-item response shape. Batch calls return
// the BatchResult shape with per-item results and a summary.
//
// Usage in a 2.0 tool handler:
//
//	func handleFinish(ctx context.Context, req mcp.CallToolRequest) (any, error) {
//	    args, _ := req.Params.Arguments.(map[string]any)
//
//	    // Single-item path
//	    taskID, _ := args["task_id"].(string)
//	    if tasks, ok := args["tasks"].([]any); ok {
//	        return ExecuteBatch(ctx, tasks, func(ctx context.Context, item any) (string, any, error) {
//	            id, _ := item.(string)
//	            result, err := finishOne(ctx, id, args)
//	            return id, result, err
//	        })
//	    }
//	    return finishOne(ctx, taskID, args)
//	}
package mcp

import (
	"context"
	"fmt"
)

// MaxBatchSize is the maximum number of items allowed in a single batch call.
// Batches exceeding this limit are rejected before any processing occurs.
const MaxBatchSize = 100

// BatchSummary holds aggregate counts for a batch operation.
type BatchSummary struct {
	Total     int `json:"total"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

// ItemResult holds the outcome of a single item within a batch operation.
type ItemResult struct {
	// ItemID is the identifier of the item (e.g. task ID, entity ID).
	// It is the value returned by the per-item handler's ID extractor.
	ItemID string `json:"item_id,omitempty"`

	// Status is "ok" when the item succeeded, "error" when it failed.
	Status string `json:"status"`

	// Data holds the per-item result payload on success. Omitted on error.
	Data any `json:"data,omitempty"`

	// Error describes the failure when Status is "error". Omitted on success.
	Error *ErrorDetail `json:"error,omitempty"`

	// SideEffects lists cascades produced by this item's operation.
	SideEffects []SideEffect `json:"side_effects,omitempty"`
}

// BatchResult is the response shape returned for batch operations.
// It is only used when the caller provides an array input; single-item
// calls return their own response shape directly.
type BatchResult struct {
	// Results contains one ItemResult per input item, in input order.
	Results []ItemResult `json:"results"`

	// Summary provides aggregate counts across all items.
	Summary BatchSummary `json:"summary"`

	// SideEffects is the union of all per-item side effects, aggregated for
	// callers that want a flat view of all cascades.
	SideEffects []SideEffect `json:"side_effects,omitempty"`
}

// BatchItemHandler is the per-item function called by ExecuteBatch.
// It receives the context (with a fresh side-effect sub-collector) and the
// raw item value from the input array.
// It returns: the item's ID string (for ItemResult.ItemID), the result
// data (for ItemResult.Data), and any error.
type BatchItemHandler func(ctx context.Context, item any) (itemID string, data any, err error)

// ExecuteBatch executes handler for each item in items, collecting per-item
// results with partial-failure semantics: a failure on item N does not prevent
// processing of item N+1.
//
// Side effects collected during each item's handler are attached to both the
// per-item ItemResult and the aggregate BatchResult.SideEffects list.
//
// If len(items) > MaxBatchSize, ExecuteBatch returns an error immediately
// without processing any items.
//
// The returned value is always a *BatchResult, which satisfies the any
// return type expected by 2.0 tool handlers.
func ExecuteBatch(ctx context.Context, items []any, handler BatchItemHandler) (any, error) {
	if len(items) > MaxBatchSize {
		return nil, fmt.Errorf("batch_limit_exceeded: %d items exceeds the maximum of %d per batch call", len(items), MaxBatchSize)
	}

	results := make([]ItemResult, 0, len(items))
	var allEffects []SideEffect
	succeeded := 0
	failed := 0

	for _, item := range items {
		// Create a sub-collector for this item so we can attribute side effects
		// to the specific item that produced them.
		subCollector := &SideEffectCollector{}
		itemCtx := ContextWithCollector(ctx, subCollector)

		itemID, data, err := handler(itemCtx, item)
		itemEffects := subCollector.Drain()

		var result ItemResult
		if err != nil {
			failed++
			result = ItemResult{
				ItemID:      itemID,
				Status:      "error",
				Error:       &ErrorDetail{Code: "item_error", Message: err.Error()},
				SideEffects: nonEmptyEffects(itemEffects),
			}
		} else {
			succeeded++
			result = ItemResult{
				ItemID:      itemID,
				Status:      "ok",
				Data:        data,
				SideEffects: nonEmptyEffects(itemEffects),
			}
		}

		results = append(results, result)
		allEffects = append(allEffects, itemEffects...)
	}

	return &BatchResult{
		Results: results,
		Summary: BatchSummary{
			Total:     len(items),
			Succeeded: succeeded,
			Failed:    failed,
		},
		SideEffects: nonEmptyEffects(allEffects),
	}, nil
}

// nonEmptyEffects returns effects if non-empty, nil otherwise.
// This keeps the JSON output clean by omitting empty arrays.
func nonEmptyEffects(effects []SideEffect) []SideEffect {
	if len(effects) == 0 {
		return nil
	}
	return effects
}

// IsBatchInput reports whether the arguments map contains a batch array
// parameter with the given key name. The value must be a non-nil []any.
//
// Usage:
//
//	if IsBatchInput(args, "tasks") {
//	    items, _ := args["tasks"].([]any)
//	    return ExecuteBatch(ctx, items, handler)
//	}
func IsBatchInput(args map[string]any, batchKey string) bool {
	if args == nil {
		return false
	}
	v, ok := args[batchKey]
	if !ok {
		return false
	}
	arr, ok := v.([]any)
	return ok && arr != nil
}
