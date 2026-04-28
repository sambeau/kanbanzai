# Implementation Plan: Cache-First Read Path in EntityService (FEAT-01KPXH0F5GFNV)

| Field      | Value                                             |
|------------|---------------------------------------------------|
| Feature    | FEAT-01KPXH0F5GFNV                                |
| Plan       | P29-state-store-read-performance                  |
| Spec ref   | work/specs/feat-01kpxh0f5gfnv-cache-first-read-path.md |

---

## Scope

Wire the existing SQLite cache into `EntityService.Get()` and `EntityService.List()` so that warm-cache calls resolve in O(1) time. Add `IsWarm(entityType string) bool` to `internal/cache/cache.go` to disambiguate cold-cache from legitimately-empty entity sets. Retain full filesystem fallback for nil-cache, cold-cache, and error scenarios.

Files changed:
- `internal/cache/cache.go` — add `IsWarm` method and warmth tracking field
- `internal/service/entities.go` — add fast paths to `Get()` and `List()`
- `internal/cache/cache_test.go` — IsWarm unit tests
- `internal/service/entities_test.go` — fast path + fallback unit tests

---

## Task Breakdown

### Task 1 — Add `IsWarm` per-type tracking to `cache.Cache` (TASK-01KPYF4C6FFDF)

**Outcome:** `cache.Cache` exposes `IsWarm(entityType string) bool` backed by an in-process `map[string]bool`. The map is updated (set to `true`) inside `Upsert` (for the upserted entity type) and inside `Rebuild` (for every entity type present in the rebuild records). `IsWarm` returns `false` for any type not yet seen this session, regardless of on-disk SQLite state.

**Implementation notes:**
- Add a `warm map[string]bool` field to the `Cache` struct (initialise lazily or in `Open`).
- In `Upsert`, after the SQL exec succeeds: `c.warm[row.EntityType] = true`.
- In `Rebuild`, after the transaction commits: mark warm for each distinct `EntityType` in the records slice.
- `IsWarm` is a simple map read — no SQL, no I/O.

**Verification:** Unit tests in `cache_test.go`: cold cache returns false; after `Upsert` returns true for that type only; after `Rebuild` returns true for all rebuilt types; opening a fresh `Cache` over an existing DB file returns false until `Upsert`/`Rebuild` is called.

---

### Task 2 — Implement `Get()` cache fast path (TASK-01KPYF54MRS0C)

**Depends on:** Task 1

**Outcome:** `EntityService.Get()` consults the cache before falling back to `ResolvePrefix()` when `slug == ""`.

**Implementation notes:**

Current code (slug == "" branch):
```
resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
```

New code (prepend before the above):
```go
if s.cache != nil && s.cache.IsWarm(entityType) {
    if slug, filePath, found := s.cache.LookupByID(entityType, entityID); found {
        record, err := s.store.LoadFromPath(entityType, entityID, slug, filePath)
        // ... build and return GetResult
    }
    // not found in cache — fall through to ResolvePrefix
}
```

If `LookupByID` returns an error path (it returns a bool, not an error), a miss simply falls through. No error to log for a miss.

Note: `LookupByID` returns `(slug, filePath string, found bool)` — it does not return an error. The only error scenario for the cache fast path in `Get()` is a store.Load failure after a cache hit, which is handled normally (return the error to the caller, since the file should exist if the cache is warm).

**Verification:** Unit tests covering AC-001 through AC-004.

---

### Task 3 — Implement `List()` cache fast path (TASK-01KPYF6049P5K)

**Depends on:** Task 1

**Outcome:** `EntityService.List()` consults the cache before falling back to `filepath.Glob`.

**Implementation notes:**

New code (prepend before the existing `filepath.Glob` block):
```go
if s.cache != nil && s.cache.IsWarm(entityType) {
    rows, err := s.cache.ListByType(entityType)
    if err != nil {
        log.Printf("[entity] cache ListByType error for %s (falling back): %v", entityType, err)
        // fall through to filesystem path
    } else {
        results := make([]ListResult, 0, len(rows))
        for _, row := range rows {
            var fields map[string]any
            if err := json.Unmarshal([]byte(row.FieldsJSON), &fields); err != nil {
                return nil, fmt.Errorf("corrupt cache entry for %s %s: %w", entityType, row.ID, err)
            }
            results = append(results, ListResult{
                Type: row.EntityType, ID: row.ID, Slug: row.Slug,
                Path: row.FilePath, State: fields,
            })
        }
        return results, nil
    }
}
```

Key points:
- On `ListByType` error: log and fall through (do not return the cache error).
- On corrupt `fields_json`: return an error immediately (FR-007 — no partial results).
- An empty `rows` slice with a warm cache is a valid result (entity type exists but has no entities) — return the empty slice, do not fall back.

**Verification:** Unit tests covering AC-005 through AC-009.

---

### Task 4 — Write cache-first read path tests (TASK-01KPYF6R1RC85)

**Depends on:** Tasks 2 and 3

**Outcome:** Full unit test coverage for all 14 acceptance criteria across `cache_test.go` and `entities_test.go`.

**Test structure:**

`internal/cache/cache_test.go` — IsWarm tests (AC-010, AC-011, AC-012):
- `TestIsWarm_AfterUpsert`
- `TestIsWarm_ColdAfterOpen` (open existing DB, confirm false)
- `TestIsWarm_AfterRebuild`

`internal/service/entities_test.go` — Get fast path (AC-001 to AC-004):
- `TestGet_CacheFastPath_Hit` — warm cache hit, no glob
- `TestGet_CacheFastPath_Miss` — warm cache miss, falls back
- `TestGet_NilCache_FallsBack`
- `TestGet_CacheError_FallsBack` (note: LookupByID has no error return; test the miss path instead)

`internal/service/entities_test.go` — List fast path (AC-005 to AC-009):
- `TestList_CacheFastPath_PopulatedType`
- `TestList_ColdType_FallsBack` (AC-006)
- `TestList_NilCache_FallsBack` (AC-007)
- `TestList_CacheError_FallsBack` — inject a wrapped cache that returns error (AC-008)
- `TestList_CorruptFieldsJSON_ReturnsError` (AC-009)

`internal/service/entities_test.go` — Result equivalence (AC-013, AC-014):
- `TestList_ResultEquivalence` — compare cache fast path vs filesystem path for same state
- `TestGet_ResultEquivalence` — compare cache hit vs filesystem for same entity

---

## Dependency Graph

```
Task 1 (IsWarm)
    ├── Task 2 (Get fast path)
    │       └── Task 4 (Tests)
    └── Task 3 (List fast path)
            └── Task 4 (Tests)
```

Tasks 2 and 3 can run in parallel after Task 1 completes. Task 4 runs after both 2 and 3.

---

## Risk Assessment

**Risk: `store.Load` signature may not accept a known file path directly.**
The current `Get()` calls `s.store.Load(entityType, entityID, slug)` which derives the path from type+id+slug. If the cache `filePath` diverges from the path `Load` would derive, a cache hit could point to a stale path. Mitigation: verify that `store.Load` derives the path deterministically from `(entityType, id, slug)` and that `cache.FilePath` always stores the same derived path. If so, the slug from the cache is sufficient and `filePath` is redundant — just use `slug` to call `store.Load(entityType, entityID, slug)` directly.

**Risk: `json.Unmarshal` into `map[string]any` may not round-trip all field types identically to YAML unmarshalling.**
Mitigation: the result equivalence tests (AC-013, AC-014) will catch any divergence. If types diverge (e.g. integers vs float64), normalise during `cacheUpsertFromResult` marshalling or during deserialisation.

---

## Verification Approach

- All acceptance criteria covered by unit tests in Tasks 2–4.
- `go test ./internal/cache/... ./internal/service/...` must pass with no new failures.
- No changes to MCP tool output format; no integration tests required beyond existing suite.