# Implementation Plan: Batch Handler False-Positive Fix

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPKCJWPF (batch-handler-false-positive)                  |
| Spec     | `work/spec/batch-handler-false-positive.md`                        |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §5         |
| Plan     | P15 — Kanbanzai 2.5 Infrastructure Hardening                       |

---

## 1. Implementation Approach

This is the smallest fix in P15. `ExecuteBatch` in `internal/mcp/batch.go` needs
a single additional check after each handler returns: if the Go error is `nil`
but the data payload matches the tool-result error convention, the item must be
classified as failed rather than succeeded.

The work is self-contained in two files — the fix and its tests — and has no
dependencies on other P15 features. It is recommended as the second item to
implement (after the `docint` fix) because it makes all subsequent batch
operations trustworthy.

**Execution order:**

```
[Task 1: Fix ExecuteBatch]  ──→  [Task 2: Tests]
```

Task 2 can be written in TDD style before Task 1, but must pass after Task 1.
Both tasks touch the same two files and should be delivered together.

---

## 2. Interface Contract

The `BatchItemHandler` signature MUST NOT change:

```
type BatchItemHandler func(ctx context.Context, item any) (itemID string, data any, err error)
```

The tool-result error pattern to detect (established by existing MCP convention):

```
data is map[string]any  AND  data["error"] is a non-empty string
```

No new types, no new exported symbols. The fix is internal to `ExecuteBatch`.

---

## 3. Task Breakdown

| # | Task | Files | Spec refs |
|---|------|-------|-----------|
| 1 | Fix `ExecuteBatch` tool-error detection | `internal/mcp/batch.go` | REQ-01–REQ-10 |
| 2 | Unit tests | `internal/mcp/batch_test.go` | AC-12–AC-15 |

---

## 4. Task Details

### Task 1: Fix `ExecuteBatch`

**Objective:** After a handler returns `(itemID, data, nil)`, inspect `data`
for the tool-result error pattern before marking the item as succeeded. If the
pattern is detected, mark the item as failed and populate `ItemResult.Error`
with the error string from the payload.

**Specification references:** REQ-01, REQ-02, REQ-03, REQ-04, REQ-05, REQ-06,
REQ-07, REQ-08, REQ-09, REQ-10.

**Input context:**

- Read `internal/mcp/batch.go` in full before editing. Locate `ExecuteBatch`
  and the `ItemResult` struct (or wherever per-item results are assembled).
- Read `internal/mcp/batch_test.go` to understand existing test coverage and
  result shape.
- Grep for `"error"` key usage across `internal/mcp/` to confirm the
  `map[string]any{"error": "..."}` convention is the current inline-error
  pattern. Do not invent a new convention.

**Output artifacts:**

- `internal/mcp/batch.go` — add a helper function (unexported) that tests
  whether an `any` value matches the tool-result error pattern, and call it
  inside `ExecuteBatch` after each `(data, nil)` return. The helper should:
  1. Type-assert `data` to `map[string]any`; return `("", false)` if it fails.
  2. Read the `"error"` key; return `("", false)` if absent or not a string.
  3. Return `(errMsg, true)` if the key is a non-empty string.

**Constraints:**

- Do not change the `BatchItemHandler` type signature (REQ-09).
- Do not modify any handler that calls `ExecuteBatch` (REQ-10), unless a
  handler is found to return a genuine success payload that happens to have an
  `"error"` key (DEP-02 scenario — flag this to the human before changing it).
- The fix must be a pure addition inside `ExecuteBatch`; no structural
  refactoring of the function.

---

### Task 2: Unit Tests

**Objective:** Confirm all four acceptance criteria with targeted unit tests.
No existing tests should regress.

**Specification references:** AC-12, AC-13, AC-14, AC-15. Also covers REQ-06,
REQ-07, REQ-08 as regression invariants.

**Input context:**

- Read `internal/mcp/batch_test.go` for existing test patterns and helper
  conventions before adding new cases.
- The spec acceptance criteria map directly to four test cases (one per AC).

**Output artifacts:**

- `internal/mcp/batch_test.go` — add the following test cases (table-driven
  if the existing suite uses that style, otherwise separate test functions):

  | Test case | Handler returns | Expected outcome |
  |-----------|-----------------|-----------------|
  | AC-12 | `(id, map{"error":"msg"}, nil)` | item counted as failed |
  | AC-13 | same as AC-12 | `ItemResult.Status == "error"`, `Error == "msg"` |
  | AC-14 | `(id, map{"result":"ok"}, nil)` | item counted as succeeded |
  | AC-15 | `(id, nil, errors.New("boom"))` | item counted as failed, `Error == "boom"` |

  Add one additional case to confirm REQ-08: a batch with two items — one
  success and one tool-error — produces `succeeded: 1, failed: 1`.

**Constraints:**

- Tests must pass under `go test ./...` and `go test -race ./...`.
- Do not remove or modify any existing test case.

---

## 5. Acceptance Criteria Traceability

| AC  | Task |
|-----|------|
| AC-12 | Task 1 (fix), Task 2 (test) |
| AC-13 | Task 1 (fix), Task 2 (test) |
| AC-14 | Task 1 (fix), Task 2 (test) |
| AC-15 | Task 2 (regression test) |

All requirements REQ-01 through REQ-10 are satisfied by Task 1. No requirement
is untraced.

---

## 6. Scope Boundaries (carried from spec)

**In scope:** Detection of `map[string]any{"error": <string>}` payloads in
`ExecuteBatch`; correct `ItemResult` population; regression test coverage.

**Out of scope:** Changes to any MCP tool handler; changes to the
`BatchItemHandler` signature; introduction of new error types or sentinel
values; any change to batch input parsing or dispatch logic.

---

## 7. Verification

Run after both tasks are complete:

```
go build ./...
go test ./internal/mcp/...
go test -race ./internal/mcp/...
go vet ./...
```

All must pass with zero failures and no new race conditions.