# P29 Implementation Handover — State Store Read Path Performance

> ⚠️ **Before touching any file: create a worktree for each feature.**
> P31 has 2 active features in worktrees and 25 total active worktrees on this repo.
> Working in the main checkout will cause conflicts. See **Worktrees** section below.

## What this plan fixes

The entity service's read path (`List()`, `Get()`) performs a full directory scan on every
call. At ~450 task files, this causes MCP tool timeouts — `next`, `cleanup`, and
`worktree create` all fail. A SQLite cache already exists and is populated on every write
but has never been wired into reads. This plan wires it in.

---

## Three features, eight tasks

### FEAT-01KPXGZXX8BJZ — Cache warm-up at server start

The cache starts cold on every server restart. Fix: call `RebuildCache()` at startup.

| Task | ID | Status |
|---|---|---|
| Wire RebuildCache into server startup | `TASK-01KPYF49K4BGH` | **ready** |
| Write server startup warm-up tests | `TASK-01KPYF4Z4T23K` | queued (depends on above) |

**The change:** In `internal/mcp/server.go`, inside the `if c, err := cache.Open(...); err == nil { ... }` block, after `entitySvc.SetCache(c)` add:

```go
start := time.Now()
if n, err := entitySvc.RebuildCache(); err != nil {
    log.Printf("[server] cache warm-up failed (continuing without cache): %v", err)
} else {
    log.Printf("[server] cache warm-up: loaded %d entities in %s", n, time.Since(start))
}
```

That's the entire production change for this feature.

---

### FEAT-01KPXH0F5GFNV — Cache-first read path in EntityService

Wire `LookupByID` and `ListByType` into `Get()` and `List()`. Requires `IsWarm()` first.

| Task | ID | Status |
|---|---|---|
| Add `IsWarm` to `cache.Cache` | `TASK-01KPYF4C6FFDF` | **ready** |
| Implement `Get()` cache fast path | `TASK-01KPYF54MRS0C` | queued (depends on IsWarm) |
| Implement `List()` cache fast path | `TASK-01KPYF6049P5K` | queued (depends on IsWarm) |
| Write cache-first read path tests | `TASK-01KPYF6R1RC85` | queued (depends on Get + List) |

**Key design points:**

- `IsWarm(entityType string) bool` — in-process `map[string]bool` on the `Cache` struct.
  Set to `true` inside `Upsert` (per type) and `Rebuild` (per type in records).
  Returns `false` for any type not seen this session, even if the DB has rows.

- `Get()` fast path — prepend before the `ResolvePrefix` call when `slug == ""`:
  ```go
  if s.cache != nil && s.cache.IsWarm(entityType) {
      if slug, filePath, found := s.cache.LookupByID(entityType, entityID); found {
          // call s.store.Load(entityType, entityID, slug) directly
      }
      // miss: fall through to ResolvePrefix
  }
  ```

- `List()` fast path — prepend before `filepath.Glob`:
  ```go
  if s.cache != nil && s.cache.IsWarm(entityType) {
      rows, err := s.cache.ListByType(entityType)
      if err != nil {
          log.Printf("[entity] cache list error for %s (falling back): %v", entityType, err)
          // fall through
      } else {
          // deserialise fields_json for each row
          // corrupt fields_json → return error (no partial results)
          // empty rows on warm cache → valid, return empty slice (do NOT fall back)
          return results, nil
      }
  }
  ```

- Fallback always: nil cache, cold type, or `ListByType` error → existing filesystem path, unchanged.

---

### FEAT-01KPXH0WSTHAM — Cache eviction on entity delete

No production code changes needed. This feature is test-only: verify existing consistency
invariants are correct and establish the eviction API contract for future hard-delete paths.

| Task | ID | Status |
|---|---|---|
| Test UpdateStatus cache consistency | `TASK-01KPYF4EVVC8D` | **ready** |
| Write cache eviction invariant tests | `TASK-01KPYF54WNPME` | queued (depends on above) |

**What to verify:**
- After `UpdateStatus` → `done` or `cancelled`: `cache.LookupByID` returns `found=true` with updated status (the upsert is already there; this confirms it).
- After `UpdateEntity` slug rename: `cache.LookupByID` returns the new slug.
- `cache.Delete` → `cache.LookupByID` returns `found=false` (establishes the API for future hard-delete paths).
- Nil-cache paths: all entity operations complete without panic.

---

## Start here

Three tasks are `ready` and can be claimed immediately — two can run in parallel:

1. `TASK-01KPYF49K4BGH` — the warm-up wiring (`internal/mcp/server.go`, ~8 lines)
2. `TASK-01KPYF4C6FFDF` — `IsWarm` on `cache.Cache` (`internal/cache/cache.go`, ~15 lines)
3. `TASK-01KPYF4EVVC8D` — `UpdateStatus` cache test (`internal/service/entities_test.go`)

Tasks 1 and 2 unblock the remaining tasks in their respective features once done.

## Key files

| File | What changes |
|---|---|
| `internal/mcp/server.go` | Add `RebuildCache` call after `SetCache` |
| `internal/cache/cache.go` | Add `warm map[string]bool` field and `IsWarm` method; update `Upsert` and `Rebuild` to mark warm |
| `internal/service/entities.go` | Add cache fast paths to `Get()` and `List()` |
| `internal/cache/cache_test.go` | `IsWarm` unit tests |
| `internal/service/entities_test.go` | Fast path, fallback, and eviction invariant tests |

## Reference documents

| Document | Path |
|---|---|
| Design | `work/design/p29-state-store-read-performance.md` |
| Warm-up spec | `work/specs/feat-01kpxgzxx8bjz-cache-warm-up-server-start.md` |
| Cache reads spec | `work/specs/feat-01kpxh0f5gfnv-cache-first-read-path.md` |
| Eviction spec | `work/specs/feat-01kpxh0wstham-cache-eviction-on-delete.md` |
| Warm-up dev-plan | `work/dev-plans/feat-01kpxgzxx8bjz-cache-warm-up-server-start.md` |
| Cache reads dev-plan | `work/dev-plans/feat-01kpxh0f5gfnv-cache-first-read-path.md` |
| Eviction dev-plan | `work/dev-plans/feat-01kpxh0wstham-cache-eviction-on-delete.md` |

## Worktrees

Each feature must be developed in its own worktree. No worktrees exist for P29 yet —
create them before writing any code:

```
worktree(action: create, entity_id: FEAT-01KPXGZXX8BJZ)
worktree(action: create, entity_id: FEAT-01KPXH0F5GFNV)
worktree(action: create, entity_id: FEAT-01KPXH0WSTHAM)
```

All file edits for a feature must be made inside its worktree directory. Do not edit
files in the main checkout.

### ⚠️ Merge conflict warning: `internal/mcp/server.go`

P31 feature `FEAT-01KPXGVQY3KQC` (non-bypassable merge gate, actively in development)
also modifies `internal/mcp/server.go` — it threads `*service.DocumentService` into the
merge tool constructor. P29's `TASK-01KPYF49K4BGH` adds the `RebuildCache()` call in a
different part of the same file (the cache-open block).

These are non-overlapping changes. The conflict will surface only at PR merge time and
is trivial to resolve: whichever branch merges second must include both changes. No
special action is needed during development.

All other P29 files (`internal/cache/cache.go`, `internal/service/entities.go`, and
their test files) are not touched by any active P31 work.

---

## Workflow

- Create the feature worktree before starting (`worktree(action: create, entity_id: FEAT-...)`).
- Claim a task with `next(id: TASK-...)` before starting work.
- Make all edits inside the feature's worktree directory.
- Commit per task: `feat(cache): <description>` or `test(cache): <description>`.
- Complete with `finish(task_id: ..., summary: ...)` when done — this unblocks dependents.
- Run `go test ./internal/cache/... ./internal/service/... ./internal/mcp/...` before finishing each task.