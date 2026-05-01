# Dev-Plan: Unify finish() State Propagation (B40-F2)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG2Q5Q1HX9           |
| Spec   | B40-fix-tool-correctness/spec-p40-spec-b40-tool-correctness |

---

## Overview

Fix the state propagation gap in `finish()`: after marking a task as `done`, the
entity YAML store is updated but the SQLite cache may not be invalidated
synchronously. Subsequent reads from the cache layer return stale `active` status.
Unify the write path so both the entity record and cache are updated before the
response returns.

---

## Task Breakdown

### T1 — Unify finish() write-through path

**Deliverable:** Updated `finish()` handler with synchronous cache invalidation.

**Scope:**
- Trace the `finish()` write path to identify where the entity YAML write occurs
  and where the cache update occurs (or is missing).
- Ensure the cache write/invalidation happens synchronously in the same call
  before the response is returned.
- Order: entity record written first, then cache invalidation/update.

**Dependencies:** None.

**Verification:** Code review — confirm write ordering and synchronicity.

**Estimated effort:** 1 (small write-path change)

### T2 — Write integration test for state consistency

**Deliverable:** Integration test verifying that after `finish()` returns success,
`entity get`, `entity list`, and gate checks all observe `done`.

**Scope:**
- Create a task in `active` status.
- Call `finish()`.
- Immediately call `entity get`, `entity list` (with parent_feature filter), and
  a feature transition gate check.
- Assert all three observe `done` (AC-003).
- Benchmark: compare p95 latency before/after fix (AC-013).

**Dependencies:** T1 (needs the fix to test against).

**Verification:** `go test ./...` passes integration test.

**Estimated effort:** 1 (integration test + benchmark)

---

## Dependency Graph

```
T1 (unify write path) ──→ T2 (integration test)
```

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T2 | After T1, `finish()` writes synchronously to both entity store and cache. T2 tests the observable behaviour: all read paths agree on `done` status within the same request context. |

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-002 | T1, T2 |
| REQ-003 | T1 |
| REQ-NF-002 | T1, T2 |
| AC-003 | T2 |
| AC-013 | T2 |
