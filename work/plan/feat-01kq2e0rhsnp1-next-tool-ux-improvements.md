# Implementation Plan: Next Tool UX Improvements

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Date    | 2026-04-25                                                         |
| Status  | Approved                                                           |
| Feature | FEAT-01KQ2E0RHSNP1 (idempotent-task-claim)                        |
| Spec    | `work/spec/feat-01kq2e0rhsnp1-next-tool-ux-improvements.md`       |
| Plan    | P34-agent-workflow-ergonomics                                      |

---

## 1. Scope

This plan implements the requirements defined in
`work/spec/feat-01kq2e0rhsnp1-next-tool-ux-improvements.md`. It covers two
independent, co-located changes to `internal/mcp/next_tool.go`:

1. **Idempotent task claim** — reroute the `active` branch in `nextClaimMode`
   to return the assembled context packet with `reclaimed: true`, instead of
   returning an error.
2. **Worktree path in context packet** — add a conditional `worktree_path`
   field to `nextContextToMap`, sourced from `actx.worktreePath` (already
   computed in `assembleContext`).

Both changes are confined to `internal/mcp/next_tool.go` and
`internal/mcp/assembly.go` (the latter read-only: it already sets
`actx.worktreePath`). No service-layer or storage changes are required.

**Out of scope:** Changes to any tool other than `next`; changes to
`assembleContext`; re-firing lifecycle hooks on reclaim; automatic
crash recovery for stale active tasks (P13 concern).

---

## 2. Interface Contracts

### `nextContextToMap` output map

One conditional key is added to the existing output of `nextContextToMap`:

```go
// Added when actx.worktreePath != "":
out["worktree_path"] = actx.worktreePath   // string, filesystem path
```

The key is omitted entirely (not `null`, not `""`) when `actx.worktreePath`
is the zero string — consistent with how `tool_hint`, `graph_project`, and
`active_experiments` are handled in the same function.

### `nextClaimMode` response (reclaim path)

The reclaim path returns the same `map[string]any` shape as a normal first
claim, with one additional boolean field:

```go
map[string]any{
    "task":      taskOut,        // same as first-claim shape
    "context":   nextContextToMap(actx),
    "reclaimed": true,           // present only on reclaim
}
```

A first-claim response does **not** include `"reclaimed"` (the key is
absent). Callers that do not inspect the field continue to work correctly.

### No changes to `assembleContext` or `assembledContext`

`actx.worktreePath` is already set in `assembleContext` when a worktree
record exists for the parent feature. Task 1 reads this field; it does not
modify `assembly.go`.

---

## 3. Task Breakdown

| # | Task | Files | Spec refs |
|---|------|-------|-----------|
| 1 | Idempotent claim: reroute `active` branch | `internal/mcp/next_tool.go` | FR-001–FR-005 |
| 2 | Worktree path: add field to `nextContextToMap` | `internal/mcp/next_tool.go` | FR-006–FR-008 |
| 3 | Tests for both changes | `internal/mcp/next_tool_test.go` | AC-001–AC-009, NFR-001–003 |

Tasks 1 and 2 can be implemented in either order or simultaneously; they touch
different parts of the same file. Task 3 depends on Tasks 1 and 2.

```
[Task 1]  ─┐
            ├─→  [Task 3: Tests]
[Task 2]  ─┘
```

---

## 4. Task Details

### Task 1: Idempotent claim — reroute `active` branch

**Objective:** Change the `"active"` case in `nextClaimMode` so that instead
of returning a hard error, it assembles and returns the full context packet
with `reclaimed: true`.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005.

**Input context:**

- Read `internal/mcp/next_tool.go` fully. Locate `nextClaimMode` and the
  `switch status { ... }` block (approximately line 207). The `"active"` case
  currently returns `nil, fmt.Errorf(...)`.
- The code below the switch (stage validation, dispatch, task reload, parent
  feature info, task summary, `assembleContext`, and final map construction)
  is the normal first-claim path. The reclaim path must execute a subset of
  this logic.
- The `"ready"` case falls through to the full claim logic.
  The `"active"` case must be refactored so it also reaches context assembly,
  but bypasses `ValidateFeatureStage`, `dispatchSvc.DispatchTask`, and the
  side-effect push.

**Recommended implementation approach:**

Introduce a boolean flag `isReclaim` set to `false` at the top of the
function. In the `"active"` case:

1. Set `isReclaim = true`.
2. Do **not** return; fall through (or `goto` / restructure the function to
   share the lower body).

Guard the dispatch-specific sections with `if !isReclaim { ... }`:

- `ValidateFeatureStage` call
- `dispatchSvc.DispatchTask` call
- The `PushSideEffect` call
- The task reload after dispatch (on the reclaim path, the already-loaded
  task record is current)

Build `taskOut` and call `assembleContext` identically for both paths.

Append `"reclaimed": true` to the returned map when `isReclaim`:

```go
result := map[string]any{
    "task":    taskOut,
    "context": nextContextToMap(actx),
}
if isReclaim {
    result["reclaimed"] = true
}
return result, nil
```

**Output artifacts:**

- `internal/mcp/next_tool.go` — modified `nextClaimMode`.

**Constraints:**

- Do **not** overwrite `claimed_at`, `dispatched_to`, or `dispatched_by` on
  the reclaim path (FR-003). The task record loaded before the switch contains
  the original dispatch metadata; surfacing it unchanged is sufficient.
- Do **not** re-fire the `OnStatusTransition` hook on the reclaim path (FR-004).
- The `"default"` case error (for tasks in `done`, `queued`, etc.) must remain
  completely unchanged (FR-005).

---

### Task 2: Worktree path — add field to `nextContextToMap`

**Objective:** Add a conditional `worktree_path` field to the map returned by
`nextContextToMap`, populated from `actx.worktreePath`.

**Specification references:** FR-006, FR-007, FR-008.

**Input context:**

- Read `internal/mcp/next_tool.go`. Locate `nextContextToMap` (approximately
  line 320). Find the `out` map construction and the conditional-field
  additions below it (`role_profile`, `spec_fallback_path`, `tool_hint`,
  `graph_project`, etc.).
- Read `internal/mcp/assembly.go`. Confirm that `assembledContext.worktreePath`
  is already set (the field is assigned `actx.worktreePath = wt.Path` in
  `assembleContext`). No change to assembly.go is required.

**Change:**

Add one conditional block to `nextContextToMap`, after the existing optional
field additions and before the `return out`:

```go
if actx.worktreePath != "" {
    out["worktree_path"] = actx.worktreePath
}
```

**Output artifacts:**

- `internal/mcp/next_tool.go` — one conditional field added to
  `nextContextToMap`.

**Constraints:**

- The key MUST be omitted entirely when `actx.worktreePath` is `""` — do not
  set `out["worktree_path"] = ""` or `out["worktree_path"] = nil` (FR-007,
  NFR-002).
- Do not modify `assembly.go`, `assembledContext`, or `assembleContext` (FR-008).

---

### Task 3: Tests

**Objective:** Cover all nine acceptance criteria with targeted unit tests.
Confirm both changes work independently and together (the combined AC-008 case).

**Specification references:** AC-001 through AC-009, NFR-001, NFR-002, NFR-003.

**Input context:**

- Read `internal/mcp/next_tool_test.go` in full before adding tests. Understand
  the existing helper patterns, mock setup (`setupNextTest` or similar), and
  how `nextClaimMode` is exercised.
- Read `internal/mcp/assembly_test.go` for `nextContextToMap` test patterns
  (the `worktree_path` test is similar to `TestNextContextToMap_WithToolHint`
  and `TestNextContextToMap_WithoutToolHint`).

**Test cases to add:**

For `nextContextToMap` (pure unit tests, no I/O):

| Test | Input `actx` | Expected output |
|------|-------------|-----------------|
| `TestNextContextToMap_WithWorktreePath` | `worktreePath: "/wt/feat-foo"` | `out["worktree_path"] == "/wt/feat-foo"` |
| `TestNextContextToMap_WithoutWorktreePath` | `worktreePath: ""` | `out` does not contain key `"worktree_path"` |

For `nextClaimMode` (integration-style tests using in-memory service):

| Test | Scenario | Expected behaviour |
|------|----------|--------------------|
| `TestNextClaimMode_AlreadyActive_ReturnsContextPacket` | Task in `active` status | Success response, `reclaimed: true` |
| `TestNextClaimMode_AlreadyActive_PreservesDispatchMeta` | Task in `active` with known `claimed_at` / `dispatched_to` | Response task metadata matches original dispatch values |
| `TestNextClaimMode_AlreadyActive_NoHookRefired` | Mock hook; task in `active` | Hook call count unchanged after reclaim |
| `TestNextClaimMode_DoneTask_StillErrors` | Task in `done` status | Returns error (unchanged behaviour) |
| `TestNextClaimMode_QueuedTask_StillErrors` | Task in `queued` status | Returns error (unchanged behaviour) |
| `TestNextClaimMode_FirstClaim_NoReclaimedField` | Task in `ready` status | Success response, no `reclaimed` key |
| `TestNextClaimMode_ActiveWithWorktree_BothFields` | Task `active`, feature has worktree | Response contains both `reclaimed: true` and `worktree_path` |

**Output artifacts:**

- `internal/mcp/next_tool_test.go` — new test functions.
- Optionally `internal/mcp/assembly_test.go` — new `nextContextToMap` unit
  tests for `worktree_path`, if preferred over adding them to `next_tool_test.go`.

**Constraints:**

- Tests must pass under `go test ./...` and `go test -race ./...`.
- Do not remove or modify any existing test case.
- Use the same mock/helper patterns already established in the test file.

---

## 5. Dependency Graph

```
Task 1 (idempotent claim)  ─┐
                             ├─→  Task 3 (tests)
Task 2 (worktree path)     ─┘
```

Parallel group: [Task 1, Task 2]
Critical path: Task 1 → Task 3 (or Task 2 → Task 3, same length)

Tasks 1 and 2 are fully independent — they modify different parts of
`next_tool.go` and neither depends on the other's output. They can be
written simultaneously or in either order. Task 3 requires both to be
complete to cover AC-008 (combined reclaim + worktree_path case).

---

## 6. Risk Assessment

### Risk: Reclaim path skips required validation

- **Probability:** medium
- **Impact:** medium
- **Mitigation:** The `ValidateFeatureStage` call enforces that the parent
  feature is in `developing`. On a reclaim, the task was already validated
  when it was first claimed — its parent feature was in `developing` then and
  should still be. Skipping re-validation on reclaim is correct and intentional
  (the task is already `active`). Test `TestNextClaimMode_AlreadyActive_ReturnsContextPacket`
  confirms the happy path; the reviewer should verify no gate is accidentally
  bypassed.
- **Affected tasks:** Task 1.

### Risk: `actx.worktreePath` is not populated in integration tests

- **Probability:** low
- **Impact:** low — affects test coverage completeness only, not production behaviour
- **Mitigation:** `assembleContext` sets `actx.worktreePath` from the worktree
  record. Integration tests for AC-006 and AC-008 must create a worktree record
  for the parent feature before exercising `nextClaimMode`. Check the existing
  worktree-related tests in `next_tool_test.go` for setup patterns.
- **Affected tasks:** Task 3.

---

## 7. Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 (active → context packet) | Unit test: `TestNextClaimMode_AlreadyActive_ReturnsContextPacket` | Task 3 |
| AC-002 (dispatch metadata preserved) | Unit test: `TestNextClaimMode_AlreadyActive_PreservesDispatchMeta` | Task 3 |
| AC-003 (no lifecycle side effect) | Unit test: `TestNextClaimMode_AlreadyActive_NoHookRefired` | Task 3 |
| AC-004 (done task still errors) | Unit test: `TestNextClaimMode_DoneTask_StillErrors` | Task 3 |
| AC-005 (queued task still errors) | Unit test: `TestNextClaimMode_QueuedTask_StillErrors` | Task 3 |
| AC-006 (worktree_path present when worktree exists) | Unit test: `TestNextContextToMap_WithWorktreePath` | Task 3 |
| AC-007 (worktree_path absent when no worktree) | Unit test: `TestNextContextToMap_WithoutWorktreePath` | Task 3 |
| AC-008 (reclaim + worktree_path combined) | Unit test: `TestNextClaimMode_ActiveWithWorktree_BothFields` | Task 3 |
| AC-009 (first claim has no reclaimed field) | Unit test: `TestNextClaimMode_FirstClaim_NoReclaimedField` | Task 3 |
| NFR-001 (backward compat) | Code inspection: callers ignoring `reclaimed` compile without change | Task 1 |
| NFR-002 (omission consistent) | Code inspection: `worktree_path` only written when non-empty | Task 2 |
| NFR-003 (no extra storage read) | Code inspection: task record loaded before switch is reused | Task 1 |

Run after all tasks complete:

```
go build ./...
go test ./internal/mcp/...
go test -race ./internal/mcp/...
go vet ./...
```

All must pass with zero failures and no new race conditions.