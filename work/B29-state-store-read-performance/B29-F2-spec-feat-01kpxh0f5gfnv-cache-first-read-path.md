# Specification: Cache-First Read Path in EntityService

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Feature     | FEAT-01KPXH0F5GFNV                                 |
| Plan        | P29-state-store-read-performance                   |
| Status      | draft                                              |
| Author      | Claude Sonnet 4.6                                  |
| Date        | 2026-05-28                                         |
| Design ref  | `work/design/p29-state-store-read-performance.md`  |

---

## 1. Related Work

| Document | Type | Relevance |
|----------|------|-----------|
| `work/design/p29-state-store-read-performance.md` | Design | Approved design for this feature; all decisions in this spec trace to it |
| `work/design/workflow-design-basis.md` section 7.1 | Design | Establishes the SQLite cache as the intended read-acceleration layer alongside the flat-file canonical store |
| `work/research/p28-issues-investigation.md` sections 2.1, 3.1 | Research | Identifies O(n) directory scans as root cause of MCP tool timeouts at ~447 task files |
| `work/reports/retro-p28-doc-intel-polish-workflow-reliability.md` | Report | Records concrete timeout failures: `next`, `cleanup`, `worktree create` |

---

## 2. Overview

`EntityService.Get()` and `EntityService.List()` in `internal/service/entities.go` perform O(n) filesystem scans on every call, reading all entity YAML files of the requested type regardless of how many exist. A SQLite cache (`internal/cache`) is already populated synchronously on every write path but is never consulted on reads.

This feature wires the existing cache into the `Get()` and `List()` read paths so that warm-cache calls resolve in O(1) time. Filesystem fallback is retained for cold-cache, nil-cache, and error scenarios. An `IsWarm` mechanism disambiguates empty cache results from cold-cache results so that legitimately empty entity sets are served from the cache rather than triggering unnecessary fallback scans.

---

## 3. Scope

### In scope

- Fast path in `EntityService.Get()`: consult cache before filesystem when cache is available and warm for the requested entity type
- Fast path in `EntityService.List()`: consult cache before filesystem when cache is available and warm for the requested entity type
- A cache-warmth tracking mechanism (`IsWarm` or equivalent) that distinguishes "no entities of this type" from "cache not yet populated for this type"
- Filesystem fallback when the cache is nil, not warm for the requested type, or returns an error
- All changes confined to `internal/service/entities.go` and `internal/cache/cache.go`

### Explicitly excluded

- Server startup cache warm-up (separate feature in P29)
- Cache eviction / invalidation on delete
- Changes to any write path
- Changes to `internal/worktree/store.go` or any store outside `entities.go`
- Changes to MCP tool signatures, descriptions, or response formats
- New SQLite tables or schema migrations
- Any change to the YAML flat-file canonical store

---

## 4. Functional Requirements

### FR-001 — Get fast path: cache lookup before filesystem

When `EntityService.Get()` is called with an entity type and ID, and the cache is non-nil and warm for that entity type, the service resolves the entity's file path from the cache and loads the file directly without performing a prefix scan or directory glob.

### FR-002 — Get fallback: filesystem scan on cache miss

When `EntityService.Get()` is called and the cache is nil, not warm for the requested entity type, or does not contain a row for the requested ID, the service falls back to the existing `ResolvePrefix()` / `filepath.Glob` path and returns the same result as it would have before this feature.

### FR-003 — Get fallback: filesystem scan on cache error

When `EntityService.Get()` is called and the cache lookup returns an error, the service falls back to the existing `ResolvePrefix()` / `filepath.Glob` path. The error is logged; it is not returned to the caller.

### FR-004 — List fast path: cache query before filesystem

When `EntityService.List()` is called with an entity type, and the cache is non-nil and warm for that entity type, the service returns all entities by deserialising the cached `fields_json` for each row, without performing a directory glob or reading any YAML file.

### FR-005 — List fallback: filesystem scan on cache miss

When `EntityService.List()` is called and the cache is nil or not warm for the requested entity type, the service falls back to the existing `filepath.Glob` path and returns the same result as it would have before this feature.

### FR-006 — List fallback: filesystem scan on cache error

When `EntityService.List()` is called and the cache query returns an error, the service falls back to the existing `filepath.Glob` path. The error is logged; it is not returned to the caller.

### FR-007 — List error on corrupt fields_json

When `EntityService.List()` takes the cache fast path and a cached row's `fields_json` cannot be deserialised, the service returns an error. It does not silently omit the row or return partial results.

### FR-008 — Cache warmth tracking: IsWarm per entity type

The cache exposes a mechanism (`IsWarm(entityType string) bool` or equivalent) that returns `true` for a given entity type only after at least one `Upsert` or `Rebuild` call has been made for that type in the current server session.

### FR-009 — Cache warmth tracking: cold type returns false

`IsWarm` returns `false` for an entity type that has never been the target of an `Upsert` or `Rebuild` call since the cache was opened, even if the underlying SQLite table contains rows for that type from a prior session.

### FR-010 — Downstream callers benefit automatically

`EntityService.List()` callers — including `WorkQueue()` and `evaluateDependents()` — observe cache-backed reads without any changes to their own code once `List()` is cache-backed.

### FR-011 — No change to results when cache is warm

For any entity type and any set of entities, the result returned by `Get()` or `List()` via the cache fast path is identical in content to the result that would be returned by the filesystem path for the same inputs.

### FR-012 — No change to write paths

No write path in `EntityService` or `internal/cache` is modified by this feature. `cacheUpsertFromResult` behaviour is unchanged.

---

## 5. Non-Functional Requirements

### NFR-001 — Get complexity on warm cache

`EntityService.Get()` with a warm cache and a cache hit performs O(1) operations: one cache key lookup and one file read by known path. It does not perform any directory scan.

### NFR-002 — List complexity on warm cache

`EntityService.List()` with a warm cache performs O(1) operations: one SQL query returning all rows for the type, plus O(k) deserialisation where k is the number of entities of that type. It does not perform any directory glob or open any YAML file.

### NFR-003 — No regression on cold cache

`EntityService.Get()` and `EntityService.List()` with a nil or cold cache perform identically in correctness and code path to the pre-feature implementation.

### NFR-004 — Cache errors do not surface to callers

A cache error in the read path does not change the return signature or error semantics visible to callers of `Get()` or `List()`. Callers receive either a correct result (from cache or fallback) or the same filesystem-path error they would have received before this feature.

### NFR-005 — IsWarm is cheap

The `IsWarm` mechanism does not perform any SQL query, file I/O, or allocation on the hot path. It reads from in-process state only (e.g. a boolean flag or set keyed by entity type).

---

## 6. Acceptance Criteria

### AC for FR-001 / FR-002 / FR-003 (Get path)

- **AC-001a:** Given a warm cache containing a row for entity type `task` and ID `TASK-001`, calling `Get("task", "TASK-001", "")` returns the correct entity without invoking `ResolvePrefix()` or `filepath.Glob`.
- **AC-001b:** Given a warm cache that does not contain a row for `TASK-999`, calling `Get("task", "TASK-999", "")` falls back to the filesystem path and returns the same entity (or error) as the pre-feature implementation.
- **AC-001c:** Given a nil cache, calling `Get("task", "TASK-001", "")` falls back to the filesystem path and returns the correct entity.
- **AC-001d:** Given a cache that returns an error on lookup, calling `Get("task", "TASK-001", "")` falls back to the filesystem path; the error is not propagated to the caller.

### AC for FR-004 / FR-005 / FR-006 / FR-007 (List path)

- **AC-004a:** Given a warm cache containing rows for entity type `task`, calling `List("task")` returns all entities deserialised from `fields_json` without performing a `filepath.Glob`.
- **AC-004b:** Given a cache with no warmth record for entity type `decision`, calling `List("decision")` falls back to the filesystem path regardless of whether the SQLite table contains rows for that type from a prior session.
- **AC-004c:** Given a nil cache, calling `List("task")` falls back to the filesystem path and returns the correct entity list.
- **AC-004d:** Given a cache that returns an error on `ListByType`, calling `List("task")` falls back to the filesystem path; the error is not propagated to the caller.
- **AC-004e:** Given a warm cache where one row has a malformed `fields_json` value, calling `List("task")` returns an error rather than a partial list.

### AC for FR-008 / FR-009 (IsWarm)

- **AC-008a:** After opening a fresh cache and calling `Upsert` for entity type `task`, `IsWarm("task")` returns `true`.
- **AC-008b:** After opening a fresh cache without any `Upsert` or `Rebuild` call, `IsWarm("task")` returns `false` even if the SQLite database on disk already contains `task` rows from a prior session.
- **AC-008c:** After opening a fresh cache and calling `Rebuild` with task entities, `IsWarm("task")` returns `true`.

### AC for FR-011 (result equivalence)

- **AC-011a:** For any entity type with at least one entity on disk, the list of entity IDs returned by `List()` via the cache fast path equals the list returned by the filesystem path for the same state.
- **AC-011b:** For any entity ID present on disk, the entity returned by `Get()` via the cache fast path is deeply equal to the entity returned by the filesystem path.

---

## 7. Dependencies and Assumptions

### Dependencies

| Dependency | Nature |
|------------|--------|
| `internal/cache.Cache` — `LookupByID(entityType, id string) (slug, filePath string, found bool)` | Used by `Get()` fast path. Already present. |
| `internal/cache.Cache` — `ListByType(entityType string) ([]EntityRow, error)` | Used by `List()` fast path. Already present. |
| `internal/cache.EntityRow.FieldsJSON` | Must contain a JSON-serialisable representation of full entity fields. Already present. |
| `internal/cache.Cache` — `IsWarm(entityType string) bool` (or equivalent) | Required to disambiguate empty cache results (FR-008, FR-009). May need to be added to `cache.go`. |
| All write paths call `cacheUpsertFromResult` | Assumed already correct; not verified or changed by this feature. |

### Assumptions

- **A-001:** The SQLite cache is opened before `EntityService` begins serving requests. This spec does not address server startup ordering.
- **A-002:** `cacheUpsertFromResult` is called on every successful entity write in the current codebase. If any write path omits this call, cache reads may return stale data; fixing such omissions is out of scope.
- **A-003:** The `fields_json` stored in each `EntityRow` is sufficient to reconstruct the full entity map returned by the current `List()` filesystem path. No additional YAML file reads are required to serve a `List()` call from the cache.
- **A-004:** The server is single-process. No cross-process cache coherency is required.
- **A-005:** `IsWarm` only needs to track warmth at entity-type granularity (not per-ID). Per-type tracking is sufficient to distinguish a legitimately empty type from a cold-cache type.
```

Now let me register the document: