# Implementation Plan: Task Auto-Promotion on Developing

| Field   | Value                                                                        |
|---------|------------------------------------------------------------------------------|
| Date    | 2026-04-25                                                                   |
| Status | approved |
| Feature | FEAT-01KQ2E0RFNBWZ (task-auto-promotion-on-developing)                       |
| Spec    | `work/spec/feat-01kq2e0rfnbwz-task-auto-promotion-on-developing.md`          |
| Design  | `work/design/design-p34-agent-workflow-ergonomics.md` §2                     |
| Plan    | P34 — Agent Workflow Ergonomics                                              |

---

## Overview

This plan implements the requirements in
`work/spec/feat-01kq2e0rfnbwz-task-auto-promotion-on-developing.md`. It covers:

- A new `PromoteQueuedTasks(featureID string)` function on `EntityService` that
  transitions dependency-free `queued` tasks to `ready`.
- A new `FeaturePromotionHook` implementing `StatusTransitionHook` that calls
  `PromoteQueuedTasks` when a feature transitions to `developing`.
- Wiring the new hook into the composite hook assembled at server startup.

It does **not** cover:
- Changes to the existing dependency auto-promotion logic (`DependencyUnblockingHook`).
- Any new lifecycle status names or transitions.
- Promotion triggered by any event other than a feature transitioning to `developing`.

---

## 2. Interface Contracts

### `PromoteQueuedTasks`

Added to `EntityService` in `internal/service/entities.go`:

```go
// PromoteQueuedTasks transitions all queued tasks with no unmet dependencies
// to ready for the given feature. Called by the OnStatusTransition hook when a
// feature enters developing. Individual failures are logged but do not abort
// the loop. Safe to call multiple times.
func (s *EntityService) PromoteQueuedTasks(featureID string) error
```

### `FeaturePromotionHook`

New file `internal/service/entity_lifecycle_hook.go`:

```go
// FeaturePromotionHook fires PromoteQueuedTasks when a feature enters developing.
type FeaturePromotionHook struct {
    svc *EntityService
}

func NewFeaturePromotionHook(svc *EntityService) *FeaturePromotionHook

func (h *FeaturePromotionHook) OnStatusTransition(
    entityType, entityID, slug, fromStatus, toStatus string,
    state map[string]any,
) *WorktreeResult
```

`OnStatusTransition` returns `nil` in all cases — it has no `WorktreeResult` to
surface. It fires only when `entityType == "feature"` and `toStatus == "developing"`.

### Hook wiring

The new hook is added to the `CompositeTransitionHook` at server startup. Locate
where `NewCompositeTransitionHook` is called (or where `SetStatusTransitionHook`
is invoked on the entity service) and include `NewFeaturePromotionHook(entitySvc)`.

---

## 3. Task Breakdown

| # | Task | Files | Spec refs |
|---|------|-------|-----------|
| 1 | Implement `PromoteQueuedTasks` | `internal/service/entities.go` | FR-004–FR-010 |
| 2 | Implement `FeaturePromotionHook` and wire into composite hook | `internal/service/entity_lifecycle_hook.go`, server startup | FR-001–FR-003 |
| 3 | Tests | `internal/service/entity_lifecycle_hook_test.go`, `internal/service/entities_test.go` | AC-001–AC-008 |

Tasks 1 and 2 have no mutual dependency — the hook calls `PromoteQueuedTasks`
which can be stubbed or written first. Task 3 depends on both.

```
[Task 1: PromoteQueuedTasks]  ──┐
                                 ├──→  [Task 3: Tests]
[Task 2: Hook + wiring]  ────────┘
```

---

## 4. Task Details

### Task 1: Implement `PromoteQueuedTasks`

**Objective:** Add a method to `EntityService` that promotes dependency-free
`queued` tasks to `ready` for a given feature.

**Specification references:** FR-004, FR-005, FR-006, FR-007, FR-008, FR-009,
FR-010, NFR-001, NFR-002, NFR-003.

**Input context:**

- Read `internal/service/entities.go` in full before editing. Understand how
  `ListTasksForFeature` (or the equivalent task-listing method) works and
  whether it is cache-backed.
- Read `internal/service/dependency_hook.go` to understand how the existing
  dependency auto-promotion logic checks terminal statuses — reuse the same
  terminal status set (`done`, `not-planned`, `cancelled`).
- Read `internal/cache/cache.go` to confirm the cache-backed list path used
  by task listing operations.
- Grep for `UpdateStatus` calls on tasks to confirm the correct call signature.

**Output artifacts:**

- `internal/service/entities.go` — add `PromoteQueuedTasks(featureID string) error`.

**Algorithm:**

1. List all tasks for `featureID` with `status == "queued"`. Use the cache-backed
   list operation.
2. For each task, read its `depends_on` slice from `task.State`.
3. If `depends_on` is empty, call `s.UpdateStatus("task", taskID, slug, "ready", ...)`.
4. If `depends_on` is non-empty, check each dependency's current status. If all
   are terminal (`done`, `not-planned`, `cancelled`), call `UpdateStatus`.
   Otherwise, skip the task — the `DependencyUnblockingHook` will promote it later.
5. Log and continue on per-task `UpdateStatus` errors (FR-009).
6. Return nil unless a structural error (e.g. list failure) occurs.

**Constraints:**

- Must use `UpdateStatus` for each task transition so lifecycle validation and
  hook firing apply uniformly (NFR-003).
- Must not call `UpdateStatus` on tasks already in `ready` or any non-`queued`
  status (FR-005).
- The terminal status set must be defined as a package-level variable or constant
  to keep it consistent with `DependencyUnblockingHook`.

---

### Task 2: Implement `FeaturePromotionHook` and wire

**Objective:** Create a new `StatusTransitionHook` implementation that calls
`PromoteQueuedTasks` on `feature → developing`, and register it in the composite
hook.

**Specification references:** FR-001, FR-002, FR-003, NFR-002.

**Input context:**

- Read `internal/service/status_transition_hook.go` in full to understand the
  `StatusTransitionHook` interface, `CompositeTransitionHook`, and
  `WorktreeTransitionHook` as a pattern to follow.
- Grep for `SetStatusTransitionHook` across `internal/` and `cmd/` to find
  where the composite hook is assembled and registered. That is where the new
  hook must be added.
- Read `internal/service/dependency_hook.go` for the existing non-worktree hook
  pattern (returns nil from `OnStatusTransition`, uses recover for panic safety).

**Output artifacts:**

- `internal/service/entity_lifecycle_hook.go` — new file containing
  `FeaturePromotionHook` and `NewFeaturePromotionHook`.
- Server startup file (wherever `CompositeTransitionHook` is assembled) — add
  `NewFeaturePromotionHook(entitySvc)` to the hook list.

**Implementation notes:**

- Guard clause: return `nil` immediately unless `entityType == "feature"` and
  `toStatus == "developing"`. This matches the design (FR-002) and keeps the
  hot path fast.
- Wrap the `PromoteQueuedTasks` call in a `recover()` defer, consistent with
  `DependencyUnblockingHook`, so a panic never propagates to the caller (FR-003).
- Log failures from `PromoteQueuedTasks` but do not return them as an error
  through the hook result — the hook result is `*WorktreeResult`, not an error.
- Always return `nil` (this hook has no worktree to surface).

**Constraints:**

- Must not modify `WorktreeTransitionHook` or `DependencyUnblockingHook`.
- The hook must be stateless beyond the `svc *EntityService` reference.

---

### Task 3: Tests

**Objective:** Cover all eight acceptance criteria with targeted unit and
integration tests.

**Specification references:** AC-001 through AC-008, NFR-001.

**Input context:**

- Read `internal/service/dependency_hook_test.go` for test patterns (table-driven
  cases, `newTestEntityService`, `writeTestTask` helpers).
- Read `internal/service/status_transition_hook_test.go` to understand
  `mockStatusTransitionHook` and composite hook test patterns.
- Read `internal/service/entities_test.go` for `newTestEntityService` setup and
  task creation helpers.

**Output artifacts:**

- `internal/service/entity_lifecycle_hook_test.go` — tests for `FeaturePromotionHook`:
  - Hook ignores non-feature entity types (AC-005).
  - Hook ignores feature transitions to statuses other than `developing` (AC-005).
  - Hook calls `PromoteQueuedTasks` when feature → developing (AC-001).
  - Hook returns nil even when `PromoteQueuedTasks` encounters an error (AC-006).

- `internal/service/entities_test.go` — tests for `PromoteQueuedTasks`:
  - Queued tasks with no `depends_on` → promoted to `ready` (AC-001).
  - Queued tasks with all-terminal deps → promoted to `ready` (AC-002).
  - Queued tasks with non-terminal deps → remain `queued` (AC-003).
  - Tasks already in `ready` → unchanged (AC-004).
  - One promotion failure → loop continues, other tasks promoted (AC-006).
  - Calling twice on same feature → idempotent, no error (AC-007).
  - Mixed dependency-free + blocked tasks → only dependency-free promoted (AC-008).

**Constraints:**

- Tests must pass under `go test ./...` and `go test -race ./...`.
- Use `t.TempDir()` for test roots; do not rely on filesystem state between tests.
- Do not modify existing tests.

---

## 5. Dependency Graph

```
Task 1 (PromoteQueuedTasks)  ──┐
                                ├──→  Task 3 (Tests)
Task 2 (Hook + wiring)  ────────┘

Parallel group: [Task 1, Task 2]
Critical path: Task 1 → Task 3
```

---

## 6. Risk Assessment

### Risk: Task listing returns non-cache-backed results

- **Probability:** low
- **Impact:** medium — promotion runs synchronously in the transition request;
  an O(n) file scan on a large feature set could add latency.
- **Mitigation:** Verify in Task 1 that the list operation used is the same
  cache-backed path introduced in P29. If not, use the cache explicitly.
- **Affected tasks:** Task 1.

### Risk: `depends_on` entries reference unknown task IDs

- **Probability:** low
- **Impact:** low — tasks with unresolvable dependencies are conservatively left
  in `queued` (ASM-002 in the spec). No failure, just no promotion.
- **Mitigation:** Treat a lookup error as non-terminal: log and skip.
- **Affected tasks:** Task 1.

### Risk: Hook wiring location not obvious

- **Probability:** low
- **Impact:** low — the hook simply does not fire if not wired.
- **Mitigation:** Search for all `SetStatusTransitionHook` calls before writing.
  There should be exactly one call site in server startup.
- **Affected tasks:** Task 2.

---

## Traceability Matrix

Run after all tasks complete:

```
go build ./...
go test ./internal/service/...
go test -race ./internal/service/...
go vet ./...
```

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------| ---------------|
| AC-001: dependency-free queued tasks → ready on feature → developing | Unit test | Task 3 |
| AC-002: all-terminal deps → promoted | Unit test | Task 3 |
| AC-003: non-terminal dep → remains queued | Unit test | Task 3 |
| AC-004: ready task unchanged | Unit test | Task 3 |
| AC-005: no promotion on non-developing transitions | Unit test | Task 3 |
| AC-006: promotion failure → transition still succeeds | Unit test | Task 3 |
| AC-007: calling twice is idempotent | Unit test | Task 3 |
| AC-008: mixed tasks → only dependency-free promoted | Unit test | Task 3 |