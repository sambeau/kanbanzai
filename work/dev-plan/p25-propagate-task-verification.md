# Dev Plan: Propagate Task Verification to Feature Entity

**Feature:** FEAT-01KPQ08Y989P8
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Specification:** work/spec/p25-propagate-task-verification.md
**Status:** Draft

---

## Overview

This plan decomposes the propagation of task-level `verification` fields to the parent
feature entity into four sequential tasks. Tasks 1–3 are code changes; Task 4 is testing.
Tasks 1 and 3 can be written independently against the interface contracts below; Task 2
depends on Task 1 and Task 3 depends on Task 2.

---

## Dependency Graph

```
Task 1 (AggregateTaskVerification method)
    ↓
Task 2 (finishOne() integration)
    ↓
Task 3 (VerificationPassedGate partial warning)
    ↓
Task 4 (tests)
```

All tasks are sequential. Tasks 1 and 3 touch different files and could theoretically
be parallelised, but since Task 3's gate change is trivial and Task 4 needs all three
code changes in place, serialising avoids branch coordination overhead.

---

## Interface Contracts

### `AggregateTaskVerification` signature

```go
// VerificationAggregationResult is returned by AggregateTaskVerification.
type VerificationAggregationResult struct {
    Status  string // "passed", "partial", or "none"
    Written bool   // true if a write to the feature entity was performed
}

// AggregateTaskVerification aggregates per-task verification strings onto the
// parent feature entity. Must be called only after the all-terminal check passes.
// Best-effort: returns (nil-result, nil) when status is "none" (no write).
// Returns an error only for unexpected failures in entity reads/writes.
func (s *DispatchService) AggregateTaskVerification(featureID string) (*VerificationAggregationResult, error)
```

### `VerificationPassedGate.Check` return values

| `verification_status` value | Return value |
|-----------------------------|--------------|
| `"passed"` | `GateStatusPassed` |
| `"partial"` | `GateStatusWarning` |
| absent or any other value | `GateStatusFailed` |

---

## Task Breakdown

### Task 1: Implement `AggregateTaskVerification` on `DispatchService`

- **Description:** Add the `AggregateTaskVerification(featureID string) (*VerificationAggregationResult, error)` method to `DispatchService` in `internal/service/dispatch.go`. The method queries all sibling tasks for the feature, derives `verification_status` from their `verification` fields, builds the formatted summary string, and writes both fields to the feature entity via `entitySvc.UpdateEntity()` when status is non-`"none"`.
- **Deliverable:** `AggregateTaskVerification` method in `internal/service/dispatch.go`; `VerificationAggregationResult` struct (can be in same file or a new `internal/service/verification.go`).
- **Depends on:** None (independent first task).
- **Effort:** Medium
- **Spec requirements:** FR-001 (trigger condition logic), FR-003 (summary format), FR-004 (status derivation), FR-005 (no-write for none), FR-006 (overwrite), FR-007 (best-effort), FR-011 (method on DispatchService), NFR-001 (reuse sibling list), NFR-003 (no new dependencies)

**Input context:**
- `internal/service/dispatch.go` — `DispatchService` struct and existing methods; look at how `ListEntitiesFiltered` is called for the sibling task nudge in `finishOne()`.
- `internal/service/entities.go` — `UpdateEntity` signature; how `verification` and `verification_status` field names are used.
- Spec FR-003 for exact summary line format: `"<TASK-ID>: <verification text>"` with `(no verification recorded)` placeholder.
- Spec FR-004 for status derivation table.

**Output artifacts:**
- Modified `internal/service/dispatch.go` (new method)
- Possibly new `internal/service/verification.go` for the result struct if that keeps dispatch.go clean

---

### Task 2: Call aggregation from `finishOne()` and include result in MCP response

- **Description:** In `internal/mcp/finish_tool.go`, after the existing all-terminal sibling check in `finishOne()`, call `dispatchSvc.AggregateTaskVerification(parentFeatureID)`. The call is best-effort: log and continue on error. Include the result under `"verification_aggregation"` in the MCP response map when aggregation runs. In `finishBatch()`, defer aggregation until all items are processed.
- **Deliverable:** Modified `internal/mcp/finish_tool.go`.
- **Depends on:** Task 1 (needs `AggregateTaskVerification` to exist).
- **Effort:** Small
- **Spec requirements:** FR-001 (trigger condition in finishOne), FR-002 (batch deferral), FR-007 (best-effort), FR-008 (MCP response key)

**Input context:**
- `internal/mcp/finish_tool.go` — `finishOne()` function; identify the all-terminal sibling check block (the nudge-1 check). The aggregation call goes at the end of that block, after `CompleteTask` has succeeded.
- `internal/mcp/finish_tool.go` — `finishBatch()` function; identify where per-item processing ends and where the deferred aggregation trigger should go.
- Task 1's `AggregateTaskVerification` signature and `VerificationAggregationResult` type.
- Spec FR-008 for the `"verification_aggregation"` response key shape.

**Output artifacts:**
- Modified `internal/mcp/finish_tool.go`

---

### Task 3: Update `VerificationPassedGate` to warn on `"partial"` status

- **Description:** In `internal/merge/gates.go`, update `VerificationPassedGate.Check()` to return `GateStatusWarning` when `entity["verification_status"] == "partial"`. Leave all other cases unchanged: `"passed"` → `GateStatusPassed`, absent or other → `GateStatusFailed`.
- **Deliverable:** Modified `internal/merge/gates.go`.
- **Depends on:** Task 2 (logically after the aggregation path is wired; practically can be done in parallel with Task 2 since the files don't overlap, but serial is simpler).
- **Effort:** Small
- **Spec requirements:** FR-009, NFR-002 (no change to VerificationExistsGate)

**Input context:**
- `internal/merge/gates.go` — `VerificationPassedGate` struct and its `Check()` method. Identify where `verification_status` is read and compared to `"passed"`.
- `internal/merge/gates.go` — `GateStatusWarning` constant; verify it exists and is used elsewhere so we know the return type accepts it.

**Output artifacts:**
- Modified `internal/merge/gates.go`

---

### Task 4: Write tests

- **Description:** Add unit and integration tests covering the aggregation trigger, status derivation, summary format, gate behaviour, and the end-to-end finish→merge path.
- **Deliverable:** New or updated test files covering the acceptance criteria in the spec.
- **Depends on:** Tasks 1, 2, 3 (all code changes must be in place).
- **Effort:** Medium
- **Spec requirements:** All AC-001 through AC-015

**Input context:**
- `internal/service/dispatch_test.go` (or equivalent) — existing test patterns for `DispatchService`.
- `internal/mcp/finish_tool_test.go` — existing test patterns; how feature and task fixtures are set up for finish tests.
- `internal/merge/gates_test.go` — existing test patterns for gate checks.
- Spec §5 Acceptance Criteria table — each row corresponds to at least one test case.

**Test cases to cover (minimum):**

| Test | What it verifies |
|------|-----------------|
| `TestAggregateTaskVerification_AllPassed` | All done tasks with non-empty verification → status `"passed"`, feature entity written |
| `TestAggregateTaskVerification_Partial` | Mixed empty/non-empty → status `"partial"`, entity written |
| `TestAggregateTaskVerification_None` | All empty → status `"none"`, entity NOT written |
| `TestAggregateTaskVerification_WontDo` | All wont_do → status `"none"`, entity NOT written |
| `TestAggregateTaskVerification_WontDoExcludedFromSummary` | Mixed done+wont_do → wont_do absent from summary string |
| `TestAggregateTaskVerification_Overwrites` | Pre-existing verification field overwritten |
| `TestFinishOne_AggregatesOnLastTask` | Last task completion triggers aggregation; intermediate does not |
| `TestFinishBatch_DefersAggregation` | Batch finishing all remaining tasks triggers aggregation once |
| `TestFinishOne_AggregationWriteFailureDoesNotFailFinish` | Injected write error → task still marked done |
| `TestFinishOne_ResponseContainsAggregationKey` | MCP response contains `"verification_aggregation"` on last task |
| `TestVerificationPassedGate_Passed` | Gate passes for `"passed"` |
| `TestVerificationPassedGate_Partial` | Gate returns warning for `"partial"` |
| `TestVerificationPassedGate_Absent` | Gate fails for absent status |
| `TestVerificationPassedGate_None` | Gate fails for `"none"` |

**Output artifacts:**
- New/updated test functions in `internal/service/dispatch_test.go` (or `verification_test.go`)
- New/updated test functions in `internal/mcp/finish_tool_test.go`
- New/updated test functions in `internal/merge/gates_test.go`
