| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28                     |
| Status | Draft                          |
| Author | Claude Sonnet (via sambeau)    |
| Plan   | design-performance-follow-up   |

## Scope

This plan implements the performance follow-up design defined in
`work/_project/design-performance-follow-up.md`
(PROJECT/design-design-performance-follow-up). It covers cache-backed
`entityExists()` and cascade optimisation for the task-finish path.
Worktree store caching and tool-level batching are out of scope per the
design's non-goals.

## Task Breakdown

### Task 1: Wire cache into entityExists()

- **Description:** Add a cache fast-path to `EntityService.entityExists()` that
  consults `Cache.EntityExists()` before falling back to the filesystem glob/stat.
  The cache method already exists; this task only wires it into the service layer.
- **Deliverable:** Modified `internal/service/entities.go` — `entityExists()` method.
- **Depends on:** None.
- **Effort:** Small — single-location change, existing cache method.
- **Design reference:** Decision 1 (cache-backed entityExists).

### Task 2: Add pre-loaded cascade variants

- **Description:** Add internal `checkAllTasksTerminalCached` and
  `checkAllFeaturesTerminalCached` variants to `entity_children.go` that accept
  pre-loaded parent-indexed entity maps. Wire them into the lifecycle hook's
  finish cascade so `MaybeAutoAdvanceFeature` and `MaybeAutoAdvancePlan` share
  a single load of tasks and features.
- **Deliverable:** Modified `internal/service/entity_children.go` and the
  lifecycle hook caller.
- **Depends on:** None (independent of Task 1).
- **Effort:** Medium — new function signatures, map-building logic, hook wiring.
- **Design reference:** Decision 2 (pre-loaded cascade variants).

### Task 3: Tests

- **Description:** Add tests for cache-backed `entityExists()` (cache hit, cache
  miss fallback, cache nil, cache cold) and for the cascade variants (correct
  terminal detection with pre-loaded maps, behaviour when parent key is missing).
- **Deliverable:** Extended `internal/service/entities_test.go` and
  `internal/service/entity_children_test.go`.
- **Depends on:** Task 1, Task 2.
- **Effort:** Medium.
- **Design reference:** All decisions.

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 (no dependencies)
Task 3 → depends on Task 1, Task 2

Parallel groups: [Task 1, Task 2]
Critical path: Task 1 → Task 3 (or Task 2 → Task 3)
```

## Risk Assessment

### Risk: Cache EntityExists returns false negative after external YAML edit
- **Probability:** Low.
- **Impact:** Low — single stale read, corrected on next cache rebuild.
- **Mitigation:** Filesystem fallback is preserved. Cache is rebuilt at every
  server start and CLI invocation. External YAML edits between cache rebuilds
  are rare.
- **Affected tasks:** Task 1.

### Risk: Pre-loaded cascade maps miss an entity created mid-cascade
- **Probability:** Very low — cascades complete in milliseconds.
- **Impact:** Low — `MaybeAutoAdvanceFeature` returns false and the advance
  is retried on the next task completion.
- **Mitigation:** Fall back to `List()` when the pre-loaded map is missing
  the expected parent key. The feature status gate is re-checked on every
  `finish` call.
- **Affected tasks:** Task 2.

## Verification Approach

| Acceptance Criterion | Method | Producing Task |
|---------------------|--------|---------------|
| `entityExists` returns true from cache when entity cached | Unit test | Task 3 |
| `entityExists` falls back to filesystem when cache nil/cold | Unit test | Task 3 |
| `entityExists` falls back to filesystem when cache returns error | Unit test | Task 3 |
| Cascade with pre-loaded maps matches full-scan behaviour | Unit test | Task 3 |
| Cascade falls back when parent key absent from pre-loaded map | Unit test | Task 3 |
| `CreateTask` parent-existence check uses cache when warm | Existing tests pass | Task 1 |
| `finish` cascade completes without re-scanning entities | Manual timing | Task 2 |
| `kanbanzai status` timing unchanged or improved | Manual timing | Task 1, Task 2 |
