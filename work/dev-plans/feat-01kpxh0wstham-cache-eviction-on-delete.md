# Implementation Plan: Cache Eviction on Entity Delete (FEAT-01KPXH0WSTHAM)

| Field      | Value                                                              |
|------------|--------------------------------------------------------------------|
| Feature    | FEAT-01KPXH0WSTHAM                                                 |
| Plan       | P29-state-store-read-performance                                   |
| Spec ref   | `work/specs/feat-01kpxh0wstham-cache-eviction-on-delete.md`       |
| Design ref | `work/design/p29-state-store-read-performance.md`                  |

---

## Scope

Verify and test cache consistency invariants for entity deletion and terminal-state transitions in `internal/service/entities.go`. No new production code paths are required: `UpdateStatus` already calls `cacheUpsertFromResult`, and no hard-delete method currently exists on `EntityService`. This plan covers tests only, plus documentation of the eviction invariant for future code review.

---

## Task Breakdown

### TASK-01KPYF4EVVC8D — Test UpdateStatus cache consistency on terminal transitions

**Files:** `internal/service/entities_test.go`

**Work:**
- Write a test that calls `UpdateStatus` to transition an entity to `done`; assert `cache.LookupByID` returns `found=true` with status `"done"`.
- Write a test that calls `UpdateStatus` to transition an entity to `cancelled`; assert the same.
- Assert `cache.Delete` is never called (verified by the cache row remaining present).

**Covers:** FR-006 and its acceptance criteria.

---

### TASK-01KPYF54WNPME — Write cache eviction invariant tests

**Depends on:** TASK-01KPYF4EVVC8D

**Files:** `internal/service/entities_test.go`, `internal/cache/cache_test.go`

**Work:**
- Write a test for `UpdateEntity` slug rename: after rename, `cache.LookupByID(entityType, id)` returns the new slug (FR-003).
- Write a test with `s.cache == nil`: all entity operations (Create, UpdateStatus, UpdateEntity) complete without panic or cache-related error (FR-004).
- Write a direct `cache.Delete` test: call `cache.Upsert`, then `cache.Delete`, then assert `cache.LookupByID` returns `found=false` (FR-001 — establishes the eviction API contract for future hard-delete paths).

**Covers:** FR-001, FR-002, FR-003, FR-004.

---

## Dependency Graph

```
TASK-01KPYF4EVVC8D (UpdateStatus consistency test)
    └── TASK-01KPYF54WNPME (eviction invariant tests)
```

---

## Risk Assessment

### Risk: No hard-delete path currently exists

FR-001 cannot be tested end-to-end through `EntityService` because no hard-delete method exists. The test for FR-001 targets `cache.Delete` directly to establish the API contract. If a future hard-delete method is added, that PR must add a corresponding integration test. Severity: low — the spec explicitly acknowledges this and FR-005 designates it as a code-review gate.

---

## Verification Approach

- All tests must pass with `go test ./internal/service/... ./internal/cache/...`.
- No new production code changes; test-only.
- Confirm no regression in existing entity service tests.