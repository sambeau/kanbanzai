# Track C: Batch Operations — Review Findings

| Document        | Track C Review Findings                                         |
|-----------------|-----------------------------------------------------------------|
| Reviewed by     | Claude Sonnet 4.6                                               |
| Review date     | 2026-03-27T18:46:19Z                                            |
| Feature         | FEAT-01KMR8QW7A3A8 `review-batch-operations`                   |
| Spec sections   | §9 (Batch Operations), §30.3 (Batch operations acceptance criteria) |
| Impl plan       | `work/plan/kanbanzai-2.0-implementation-plan.md` §6 (Track C)  |
| Status          | **4 blocking findings, 4 non-blocking findings**                |

---

## Review Scope

Track C implements the batch operations infrastructure shared across all 2.0 tools:

- `internal/mcp/batch.go` — `ExecuteBatch`, `IsBatchInput`, `BatchResult`, `ItemResult`, `BatchSummary`
- `internal/mcp/batch_test.go` — unit tests for the batch infrastructure
- Integration in `finish_tool.go`, `entity_tool.go`, `doc_tool.go`, `estimate_tool.go`

The review covers: spec compliance, implementation correctness, test coverage quality, code
documentation, and agent/user-facing documentation.

---

## Spec Compliance Against §30.3 Acceptance Criteria

| Criterion | Status | Notes |
|-----------|--------|-------|
| Batch-capable tools accept an array of items | ✅ | `finish`, `entity(create)`, `doc(register/approve)` |
| Single-item calls return the single-item response shape | ✅ | Tested |
| Batch calls return the batch response shape | ✅ | `BatchResult` with `results`, `summary`, `side_effects` |
| A failure on one batch item does not prevent subsequent items | ✅ | `TestExecuteBatch_PartialFailure` |
| Batch responses include per-item `status` ("ok" or "error") | ✅ | |
| Batch responses include aggregate `side_effects` | ✅ | `TestExecuteBatch_SideEffectsAggregated` |
| Batches exceeding 100 items rejected with `batch_limit_exceeded` | ❌ | **F1** — error code is `internal_error` in JSON |
| `summary.total`, `summary.succeeded`, `summary.failed` correct | ✅ | |

---

## Findings

### F1 — BLOCKING: `batch_limit_exceeded` error code not surfaced in JSON response

**Spec reference:** §9.5, §30.3 criterion 7

**What the spec requires:**
> Requests exceeding this limit are rejected with error code `batch_limit_exceeded` before
> any processing occurs.

**What actually happens:**

`ExecuteBatch` returns `(nil, fmt.Errorf("batch_limit_exceeded: ..."))`. `WithSideEffects`
catches any `error` return from the inner handler and converts it to:

```json
{"error": {"code": "internal_error", "message": "batch_limit_exceeded: 101 items exceeds the maximum of 100 per batch call"}}
```

The error *message* starts with `batch_limit_exceeded:` but the error *code* field is
`"internal_error"`. An agent checking `error.code` cannot distinguish a batch limit
rejection from any other internal error.

**Test gap:** `TestExecuteBatch_LimitExceeded` only checks `err != nil` and that the handler
was not called. It does not verify the error code in the JSON response. The acceptance
criterion is therefore not fully tested.

**Fix:** Define a sentinel `*BatchLimitError` type in `batch.go`. In `WithSideEffects` (or
`buildErrorResult`), type-check for it and use `"batch_limit_exceeded"` as the code:

```go
// In batch.go
type BatchLimitError struct{ Count, Limit int }
func (e *BatchLimitError) Error() string { ... }

// In sideeffect.go / WithSideEffects
var limitErr *BatchLimitError
if errors.As(err, &limitErr) {
    return buildErrorResult("batch_limit_exceeded", err.Error(), nil, effects), nil
}
return buildErrorResult("internal_error", err.Error(), nil, effects), nil
```

Add an end-to-end test that calls a batch-capable tool with 101 items through the full MCP
handler and asserts `error.code == "batch_limit_exceeded"`.

---

### F2 — BLOCKING: Duplicate `side_effects` key when `SignalMutation` + `ExecuteBatch` with active cascades

**Affects:** `doc(action: "approve")` batch when approvals trigger entity lifecycle
transitions. Also `entity(action: "create")` and `doc(action: "register")` batch (benign
today but fragile).

**Root cause:**

In `doc_tool.go` `docApproveAction`, `SignalMutation(ctx)` is called before the
`IsBatchInput` / `ExecuteBatch` path:

```go
SignalMutation(ctx)           // marks outer collector as isMutation=true
if IsBatchInput(args, "ids") {
    return ExecuteBatch(ctx, items, handler) // sub-collectors capture all effects
}
```

`ExecuteBatch` creates a per-item sub-collector that shadows the outer one. All
`PushSideEffect` calls from `docApproveOne` go to the sub-collector. The outer collector
drains empty.

Back in `WithSideEffects`:

- `effects = outer_collector.Drain()` → empty
- `isMutation = true` (set above)
- `buildResult(*BatchResult, nil, true)` is called

`buildResult` sets `effects = []SideEffect{}` (non-nil because `isMutation=true`) and
injects `,"side_effects":[]` into the already-serialised `*BatchResult` JSON. When the
batch produced real side effects, `BatchResult.SideEffects` is already non-empty in the
JSON, giving:

```json
{
  "results": [...],
  "summary": {...},
  "side_effects": [{"type": "status_transition", ...}],
  "side_effects": []
}
```

Duplicate JSON keys: most parsers take the *last* value (`[]`), silently discarding the
actual side effects.

**Currently triggered when:** approving a batch of specification/design documents where the
approval cascades a feature's lifecycle status (e.g. `specifying → dev-planning`). This is
a normal production workflow.

**Not currently triggered for:** `entity(create)` batch (creation produces no side effects
today) and `doc(register)` batch (registration produces no side effects).

**Fix:** Add a type-check in `buildResult` for `*BatchResult`. When the result is a
`*BatchResult`, return it directly after ensuring `SideEffects` is non-nil for mutations,
rather than injecting into the serialised JSON:

```go
func buildResult(result any, effects []SideEffect, isMutation bool) *mcp.CallToolResult {
    // Batch results manage their own side_effects field.
    if br, ok := result.(*BatchResult); ok {
        if isMutation && br.SideEffects == nil {
            br.SideEffects = []SideEffect{}
        }
        b, _ := json.Marshal(br)
        return mcp.NewToolResultText(string(b))
    }
    // ... existing single-item logic unchanged
}
```

**Additional test required:** A test for `doc(approve)` batch where at least one document
approval triggers an entity transition side effect. Assert:
1. `BatchResult.side_effects` contains the transition.
2. Per-item `side_effects` contains the transition.
3. There is no duplicate `side_effects` key in the raw JSON.

---

### F3 — BLOCKING: `finish` tool missing `SignalMutation` — `side_effects: []` absent when no cascades occur

**Spec reference:** §8.4 — "When a mutation produces no side effects, the field is present
as an empty array. The field is never omitted."

**What actually happens:**

`finish_tool.go` has no `SignalMutation(ctx)` call. When `finishOne` completes a task with
no downstream dependents and no knowledge entries, `PushSideEffect` is never called. The
outer collector is empty and not marked as mutation. `buildResult` takes the fast path and
serialises the result without a `side_effects` field:

```json
{"task": {...}, "knowledge": {...}}
```

The spec §12.6 response example explicitly shows `side_effects` present. An agent that
unconditionally reads `response.side_effects` (as §8.4 permits) will get `undefined`.

**Fix:** Add `SignalMutation(ctx)` to the `finishTool` handler, just before the batch/single
branch. This requires **F2 to be fixed first** — once `buildResult` correctly handles
`*BatchResult`, calling `SignalMutation` before batch mode is safe.

**Note on the existing test:** `TestFinish_NoSideEffectsWhenNoDependents` does not assert
`side_effects` is absent — it checks that no `task_unblocked` effects are present. The test
is permissive and will pass after the fix. A positive assertion should be added: when
`finish` produces no cascades, `side_effects: []` is present in the response.

---

### F4 — BLOCKING: `estimate(set)` batch does not use the Track C shared infrastructure

**Affects:** `estimate_tool.go` in `estimateSetBatch`.

Track C's implementation plan (`§6 Track C`) states:
> The batch infrastructure is used by: finish (Track E), entity(create) (Track H),
> doc(register) and doc(approve) (Track I), **and estimate(set) (Track J)**.

In practice, `estimate_tool.go` has its own custom batch loop that does not call
`ExecuteBatch` or use `BatchResult`/`ItemResult`. The custom response shape diverges from
§9.4 in multiple ways:

| Dimension | Spec §9.4 | `estimate` batch actual |
|-----------|-----------|------------------------|
| Per-item success status | `"ok"` | `"set"` |
| Per-item error shape | `{code, message}` | plain string |
| Per-item id key | `item_id` | `entity_id` |
| Summary location | `summary.{total,succeeded,failed}` | top-level `total`, `succeeded`, `failed` |
| Batch limit check | 100-item reject | **absent** |
| `side_effects` per item | present | absent |
| Top-level `side_effects` | present | absent |

An agent relying on the standard batch response shape to process `estimate(set)` results
will fail silently.

**Primary owner:** Track J. However, the Track C plan incorrectly claims `estimate` uses the
shared infrastructure. Either the `estimate` tool must be updated to use `ExecuteBatch` and
the standard `BatchResult` shape, or the plan must be corrected to note the exception and
justify the deviation.

**Recommended fix:** Refactor `estimateSetBatch` to use `ExecuteBatch`. The response shape
then matches §9.4 automatically, and batch limit enforcement comes for free.

---

### F5 — Non-blocking: Per-item error code is always the generic `"item_error"`

**Spec reference:** §9.4 (example shows domain-specific codes like `"invalid_status"`)

`ExecuteBatch` hardcodes `Code: "item_error"` for all per-item failures:

```go
Error: &ErrorDetail{Code: "item_error", Message: err.Error()},
```

The spec example implies domain-specific codes propagate through:

```yaml
error:
  code: "invalid_status"
  message: "Task TASK-01JY... is in status 'done', expected 'active'"
```

The spec text does not explicitly require this, so `"item_error"` is defensible as a generic
wrapper. However, it reduces diagnostic value for agents trying to classify failures.

**Recommendation:** Low priority. If per-item handlers begin returning typed errors (e.g.
`*ValidationError` with an embedded code), `ExecuteBatch` could detect and propagate them.
Document the current behaviour explicitly in `batch.go` so future implementors understand
the limitation.

---

### F6 — Non-blocking: Test coverage gap — batch approval with entity transition side effects

`TestDocTool_Approve_Batch` approves two plain documents (no entity transitions). It does
not exercise the path where `docApproveOne` calls `PushSideEffect`. This means Finding F2
is not caught by any existing test.

**Recommended fix:** Add `TestDocTool_Approve_Batch_WithEntityTransition`:
1. Register two spec documents both owned by a feature in `specifying` status.
2. Approve them in a single batch call.
3. Assert:
   - `response.results[*].side_effects` contains the `status_transition` for each document.
   - `response.side_effects` (top-level aggregate) contains both transitions.
   - The raw JSON does not contain a duplicate `side_effects` key.

This test will fail until F2 is fixed, serving as the regression test.

---

### F7 — Non-blocking: `finish` batch not tested for `side_effects: []` when no cascades

`TestFinish_BatchAggregateSideEffects` correctly tests aggregate effects when cascades
occur. There is no test asserting that `side_effects: []` is present in a batch `finish`
response when no items produce cascades.

**Recommended fix:** Add `TestFinish_BatchNoSideEffectsPresent`: complete a batch of tasks
that have no downstream dependents and no knowledge entries. Assert `side_effects` is
present and empty (`[]`). This test depends on F3 being fixed.

---

### F8 — Non-blocking: Agent-facing documentation not updated for 2.0 batch operations

**Scope:** Partial Track K concern, but noteworthy at Track C review time.

The following documentation sources still describe 1.0 workflows and do not mention 2.0
batch operations:

- `.agents/skills/kanbanzai-agents/SKILL.md` — describes `work_queue` + `dispatch_task` +
  `complete_task`, not `next` + `finish` (with batch). Agents reading this SKILL will not
  know to use `finish(tasks: [...])` for batch completion.
- `docs/mcp-tool-reference.md` — documents 1.0 tools only. No entry for `finish`,
  `entity`, `doc`, `next`, `handoff`, `status`, or `health`.

The batch pattern (when to use it, parameter names per tool, response shape) is not
documented anywhere an agent would look before starting work.

**Recommended fix (Track K prerequisite):**

1. Update `kanbanzai-agents` SKILL to describe the 2.0 dispatch loop: `next` → implement
   → `finish`, with a note on batch completion.
2. Add a "Batch Operations" section to `docs/mcp-tool-reference.md` covering the §9.4
   response shape and the batch parameters for each batch-capable tool.
3. Update the tool descriptions in the skill before Track K removes the 1.0 tools —
   otherwise agents lose all guidance.

---

## Code Quality Assessment

### `internal/mcp/batch.go`

**Strengths:**
- Clear package-level doc comment with usage example. The example is accurate and useful.
- `MaxBatchSize = 100` is a named constant — easy to find and change.
- `IsBatchInput` is a clean predicate with a single responsibility.
- Sub-collector per item is the right design: effects are isolated, attributed, and
  aggregated correctly.
- `nonEmptyEffects` helper keeps JSON clean by omitting empty arrays.
- All exported types are documented.

**Weaknesses:**
- `ExecuteBatch` returns `(any, error)` — the error path is reserved for the batch limit
  check. The error type is an untyped `fmt.Errorf` string, so callers cannot distinguish it
  from other errors without string matching (see F1).
- The per-item error code `"item_error"` is baked in (see F5). A comment noting this and
  pointing to a future improvement would help.

### `internal/mcp/batch_test.go`

**Strengths:**
- Comprehensive coverage of the infrastructure itself: single item, multiple items, partial
  failure, all fail, limit exceeded, exactly at limit, empty batch, side effects aggregated,
  side effects absent when empty, input order preserved, JSON shape, all `IsBatchInput`
  variants. This is thorough and well-structured.
- Tests are parallel, table-driven where appropriate, and have clear names.
- Spec references (`// Verifies §30.3`) are present and accurate.

**Weaknesses:**
- `TestExecuteBatch_LimitExceeded` does not verify the JSON error code (see F1, F5).
- No integration test exercises the full MCP layer with 101 items to verify the end-to-end
  error response shape.

### Integration in `finish_tool.go`, `entity_tool.go`, `doc_tool.go`

**Strengths:**
- The batch/single branch (`IsBatchInput` → `ExecuteBatch` else single path) is consistent
  across all three files.
- `parseFinishItem` and `parseFinishArgs` cleanly separate single vs. batch input parsing.
- Batch-mode tool descriptions explain the pattern for agents.

**Weaknesses:**
- Missing `SignalMutation` in `finish_tool.go` (F3).
- `SignalMutation` called before `ExecuteBatch` in `entity_tool.go` and `doc_tool.go` (F2).

### `estimate_tool.go` (Track J, noted here for completeness)

Does not use the shared batch infrastructure (F4). The `estimateSetBatch` function is
correct in its own terms (partial failure, counting) but incompatible with the standard
§9.4 response shape.

---

## Workflow Document Accuracy

`work/plan/kanbanzai-2.0-implementation-plan.md` §6 marks Track C as `✓ COMPLETE` and
states "All 8 spec §30.3 acceptance criteria verified by passing tests."

This claim is **not fully accurate**:

1. The `batch_limit_exceeded` error code criterion (F1) is not verified — the test only
   checks that `err != nil`.
2. The implementation plan states "estimate(set) uses Track C batch infrastructure" but the
   estimate tool has its own custom implementation (F4).
3. The F2 bug (duplicate `side_effects`) is a runtime correctness issue not caught by
   existing tests.

The plan should be updated to reflect these open items.

---

## Summary Table

| ID | Severity | Description | File(s) |
|----|----------|-------------|---------|
| F1 | **Blocking** | `batch_limit_exceeded` code surfaces as `internal_error` | `batch.go`, `sideeffect.go`, `batch_test.go` |
| F2 | **Blocking** | Duplicate `side_effects` key from `SignalMutation` + `ExecuteBatch` | `sideeffect.go`, `doc_tool.go`, `entity_tool.go` |
| F3 | **Blocking** | `finish` missing `SignalMutation` — `side_effects: []` absent | `finish_tool.go`, `finish_tool_test.go` |
| F4 | **Blocking** | `estimate(set)` batch uses non-standard shape, no limit check | `estimate_tool.go` |
| F5 | Non-blocking | Per-item error code always `"item_error"` | `batch.go` |
| F6 | Non-blocking | No test for batch doc approve + entity transition side effects | `doc_tool_test.go` |
| F7 | Non-blocking | No test for `finish` batch returning `side_effects: []` | `finish_tool_test.go` |
| F8 | Non-blocking | Agent/user docs not updated for 2.0 batch operations | `.agents/skills/`, `docs/` |

---

## Remediation Tasks to Create

1. **Fix `batch_limit_exceeded` error code** (F1) — Define `*BatchLimitError` in `batch.go`;
   detect it in `WithSideEffects`; add end-to-end test through MCP handler.

2. **Fix duplicate `side_effects` in `buildResult` for `*BatchResult`** (F2) — Type-check
   in `buildResult`; add regression test for batch doc approve with entity transition.

3. **Add `SignalMutation` to `finish` tool** (F3) — Depends on task 2 being done first;
   add test for `side_effects: []` when no cascades.

4. **Migrate `estimate(set)` batch to use `ExecuteBatch`** (F4) — Align response shape with
   §9.4; add batch limit enforcement; update tests.

5. **Document batch operations for agents** (F8) — Update `kanbanzai-agents` SKILL and
   `docs/mcp-tool-reference.md`; mark as Track K prerequisite.