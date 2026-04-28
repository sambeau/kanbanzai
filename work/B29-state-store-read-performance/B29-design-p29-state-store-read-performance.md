# Design: State Store Read Path Performance (P29)

| Field  | Value                    |
|--------|--------------------------|
| Date   | 2026-04-23               |
| Status | Draft                    |
| Author | Claude Sonnet 4.6        |
| Plan   | P29-state-store-read-performance |

---

## Overview

The entity service's read path (`List()`, `Get()`, `ResolvePrefix()`) performs a full
directory scan on every call, reading every YAML file of the requested entity type from
disk. A SQLite cache already exists and is populated on every write, but is never
consulted on reads. This design wires that cache into the read path and ensures it is
warm at server startup, converting O(n) disk scans into O(1) SQL lookups.

## Goals and Non-Goals

**Goals:**
- Wire the existing SQLite cache into `List()` and `Get()` / `ResolvePrefix()` read paths
- Ensure the cache is rebuilt at server startup so post-restart tool calls hit a warm cache
- Eliminate O(n) full-directory scans as the common-case code path for task, feature, and bug reads
- Maintain filesystem fallback correctness when the cache is cold, nil, or returns an error

**Non-Goals:**
- Restructuring or replacing the YAML flat-file canonical store
- Caching the worktree store (`internal/worktree/store.go`) — out of scope for this plan
- Changing any MCP tool's public API or response format
- Addressing prompt inflation or lifecycle gate issues (P30, P31)

## Dependencies

- `internal/cache` — existing SQLite cache package; may need a `ListByType` method added
- `internal/service/entities.go` — primary change target for read path wiring
- `internal/mcp/server.go` — startup sequence change to call `RebuildCache()`
- No external dependencies; no schema migrations required

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `work/design/workflow-design-basis.md` §7.1 | Design | Specifies SQLite cache for fast querying, indexing, and dependency analysis alongside flat-file canonical store — the architectural intent this design implements |
| `work/design/workflow-design-basis.md` §7.5 | Design | Specifies one file per entity — constrains the canonical store structure, not the cache |
| `work/research/p28-issues-investigation.md` §2.1, §3.1 | Research | Identifies the O(n) scan as the shared root cause of `next`, `worktree create`, and `cleanup` timeouts; confirms cache is write-only on the read path |
| `work/reports/retro-p28-doc-intel-polish-workflow-reliability.md` | Report | Records the concrete failure: `next` timeout at 447 task files, `cleanup` timeout, `worktree create` timeout even after removing 33 stale worktrees |
| `work/dev-plan/p28-worktree-creation-timeout.md` | Dev plan | Prior band-aid fix: early-termination scan added to `worktree.Store.GetByEntityID` — addressed worktree-level symptom only, did not touch entity service |

### Decisions that constrain this design

| Decision | Source | Constraint |
|----------|--------|------------|
| "Canonical state should live in git-tracked structured text files" | workflow-design-basis §7.1 | The YAML flat-file store remains the canonical source of truth; the cache is derived and disposable |
| "A local SQLite cache should be used for fast querying, indexing, dependency analysis" | workflow-design-basis §7.1 | The cache is the intended read acceleration layer — this design implements what §7.1 already decided |
| "The cache should be derived, local, disposable, rebuildable" | workflow-design-basis §7.1 | Any cache strategy must be safe to delete and regenerate from the YAML files |
| "One file per entity" | workflow-design-basis §7.5 | The directory-scan pattern is load-bearing for correctness of `List()` — it must remain the fallback, not be removed |

### Open questions from prior work

- §7.1 specifies the cache should serve reads but does not specify the consistency model — this design must decide: write-through (cache updated synchronously on every write) vs. write-behind (cache updated asynchronously) vs. read-fallback (miss falls back to filesystem).
- Prior band-aid (P28 worktree `GetByEntityID`) used a different strategy (early termination, not cache) — this design must decide whether to unify approaches or leave worktree store unchanged.

---

## Problem and Motivation

The entity service's read path performs a full directory scan on every call. `List(entityType)` calls `filepath.Glob` over the entity directory and deserialises every YAML file it finds. `Get(entityType, id, "")` — the form used by every MCP tool that receives an ID but not a slug — calls `ResolvePrefix()`, which also calls `filepath.Glob` over all files to match a filename prefix. At P28 scale:

- 447 task YAML files
- 122 feature YAML files
- 45 worktree YAML files
- 1,091 total state files

A single `next(id: TASK-xxx)` call in claim mode triggers at minimum two full directory scans: `ResolvePrefix` in `Get` (447 files), and `List("task")` in `evaluateDependents` when the task transitions to active (447 files again). A `next()` queue-mode call triggers `List("task")` in `WorkQueue`. The `cleanup` tool triggers `store.List()` twice in the execute path. The `worktree(action: create)` tool calls `entitySvc.Get(entityType, entityID, "")` — a `ResolvePrefix` scan.

A SQLite cache (`internal/cache`) exists and is populated on every write via `cacheUpsertFromResult`. The server opens this cache at startup (`internal/mcp/server.go:86–94`) but does **not** call `RebuildCache()` — the cache starts cold on every server start. Even if the cache were warm, `List()` and `Get()` contain no `s.cache` reference and never consult it.

The result is that every read operation is O(n) in the number of entity files, the file count grows monotonically with each plan (~15–25 tasks per plan), and the MCP client timeout threshold has already been crossed at current scale. Without a structural fix, the `next` tool — called on every single task claim — will become progressively less reliable throughout P29 and beyond.

**If nothing changes:** at ~600 task files (approximately P31–P32), the dependency hook scan on task completion will by itself approach the timeout threshold. `next` in queue mode will be unreliable. `cleanup` will be unable to run, causing stale worktrees to accumulate further, which will cause additional scan degradation.

---

## Design

### Overview

The fix has three parts that together close the gap between the architectural intent (§7.1) and the current implementation:

1. **Cache warm-up at server start:** Call `RebuildCache()` immediately after `SetCache()` in `internal/mcp/server.go`. This ensures the cache is always warm when the server begins accepting tool calls.

2. **Cache-first read path in `EntityService`:** Add a fast path in `Get()` that consults the cache before falling back to filesystem. Add a `ListFromCache()` method (or wire the existing cache's query interface) for `List()`, falling back to the current `filepath.Glob` scan when the cache returns a miss (cache is nil, or entity type not present). Extend `cacheUpsertFromResult` to cover all write paths already identified (it already does — the gap is only on the read side).

3. **Cache invalidation on delete:** Ensure `UpdateStatus` (which performs the effective "tombstone" of terminal-state transitions) and any explicit delete paths call a corresponding cache eviction so the cache never serves stale data for deleted entities.

### Component responsibilities

**`internal/cache` (existing, minor extension)**

The cache already stores entity rows indexed by `(entity_type, id)`. It needs one additional query method:
- `ListByType(entityType string) ([]EntityRow, error)` — returns all rows for a given entity type. This is a single `SELECT * FROM entities WHERE entity_type = ?` — O(1) with a type index.

If the cache already has a method covering this query (it may, since `RebuildCache` reads by type) — use the existing method; do not add a duplicate.

**`internal/service/entities.go` — `Get()`**

Current path: when `slug == ""`, call `ResolvePrefix()` → `filepath.Glob` → scan all filenames.

New path (fast path first):
1. If `s.cache != nil`: call `s.cache.GetByID(entityType, entityID)` — O(1) lookup by primary key.
2. If hit: slug is available in the cached row; call `s.store.Load(entityType, id, slug)` directly — O(1) file read by known path.
3. If miss (cache nil, or row not found): fall back to existing `ResolvePrefix()` path unchanged.

The fallback guarantees correctness on cold start, after cache eviction, or in tests that do not configure a cache.

**`internal/service/entities.go` — `List()`**

Current path: `filepath.Glob` + read every file.

New path:
1. If `s.cache != nil`: call `s.cache.ListByType(entityType)` — O(1) SQL query.
2. If hit (non-empty result): deserialise the cached `fields_json` column for each row — no filesystem I/O.
3. If miss (cache nil, or empty result for type): fall back to existing `filepath.Glob` path unchanged.

**Important:** An empty result from the cache is ambiguous — it could mean "no entities of this type exist" or "cache is cold for this type." The safe interpretation is: if the cache returns zero rows for a type that is expected to have entries (e.g. `"task"` when tasks are known to exist), treat it as a miss and fall back. The heuristic: always fall back when the cache returns zero rows for `task`, `feature`, and `bug` types, since these will always be non-empty in a live project. For `decision` and other sparse types, zero rows is a valid result — use the cache result as-is.

A cleaner alternative: add a `IsWarm(entityType string) bool` method to the cache that returns true only after at least one `Upsert` or `Rebuild` call for that type. This eliminates the ambiguity without a heuristic. Prefer this if the cache implementation supports it cheaply.

**`internal/mcp/server.go` — startup sequence**

After `entitySvc.SetCache(c)` at line 86–94, add:
```
if _, err := entitySvc.RebuildCache(); err != nil {
    log.Printf("[server] cache rebuild failed (continuing without cache): %v", err)
}
```

The rebuild is best-effort: if it fails (permissions, corrupt cache file), the server continues with cache disabled and falls back to filesystem reads. This matches the existing pattern where cache open failure is already handled gracefully (`if c, err := cache.Open(...); err == nil { ... }`).

**`internal/service/dependency_hook.go` — `evaluateDependents()`**

This calls `h.entitySvc.List("task")` directly. No change to this code is needed: once `List()` is cache-backed, `evaluateDependents` automatically benefits. The scan is O(1) SQL instead of O(n) filesystem.

**`internal/service/queue.go` — `WorkQueue()`**

Same as above — calls `s.List("task")` which becomes cache-backed automatically.

### Consistency model: write-through with fallback reads

Every write path already calls `cacheUpsertFromResult`. This is a synchronous write-through: the cache is updated before the MCP tool call returns. Reads consult the cache first and fall back to filesystem on miss.

This means the cache is **always at least as fresh as the last completed write in the current server session**. Across server restarts, the warm-up `RebuildCache()` call re-synchronises the cache from the canonical YAML files before the first tool call is handled.

**Staleness window:** There is no staleness window. A write that succeeds updates the cache before returning. A server restart triggers a full rebuild before serving requests. The only gap is YAML files written by external tools (git operations, manual edits) between server restarts — these are covered by the startup rebuild.

### Failure modes

| Failure | Handling |
|---------|----------|
| Cache open fails at startup | Log warning; serve without cache; all reads fall back to filesystem |
| Cache rebuild fails at startup | Log warning; serve without cache; first write will start populating it |
| Cache `GetByID` returns error | Log warning; fall back to `ResolvePrefix()` filesystem scan for this call |
| Cache `ListByType` returns error | Log warning; fall back to `filepath.Glob` filesystem scan for this call |
| Cache row's `fields_json` is corrupt | Return error from `List()`; do not silently return partial data |

### What this design does NOT change

- The YAML flat-file store remains the canonical source of truth. No YAML files are removed or restructured.
- The `ResolvePrefix()` and `filepath.Glob` paths remain in place as fallbacks. They are not removed.
- The worktree store (`internal/worktree/store.go`) is out of scope. The P28 band-aid (`GetByEntityID` early termination) remains; this design does not touch it. A future plan can unify the approaches.
- The `cache.Cache` data model is not changed — no new tables, no schema migrations.
- MCP tool signatures, descriptions, and external behaviour are unchanged.

---

## Alternatives Considered

### Alternative 1: In-memory map index (skip SQLite)

Build an in-memory `map[string]map[string]EntityRow` (keyed by `entityType → id`) inside `EntityService` at startup by scanning all YAML files once. Subsequent `Get` and `List` calls read from the map.

**What it makes easier:** No SQLite dependency; purely in-process; very fast reads.

**What it makes harder:**
- Memory footprint grows with entity count (1,091 files × average YAML size).
- The map must be kept consistent across concurrent writes. The current server is single-process stdio, so goroutine safety is manageable — but the cache already provides this.
- Two caching systems (SQLite cache + in-memory map) would exist simultaneously, creating maintenance confusion.
- The existing SQLite cache already provides indexed lookups; duplicating it in memory wastes the prior investment.

**Rejected because:** The SQLite cache already exists, is already populated on writes, and is already opened at startup. Wiring it into reads is a smaller change than building a second in-memory index, and it reuses the existing consistency model.

### Alternative 2: Rebuild cache on every server start, remove filesystem fallback

Treat cache availability as a hard requirement: if cache rebuild fails, refuse to start. Remove the `filepath.Glob` fallback paths entirely.

**What it makes easier:** Simpler read path; no branching logic; guaranteed fast reads.

**What it makes harder:**
- A corrupt or missing cache file prevents the server from starting, blocking all tool calls.
- Tests that do not configure a cache break.
- The architectural intent (§7.1: cache is "disposable, rebuildable") implies it should be possible to delete the cache and recover — a hard dependency inverts this.

**Rejected because:** The resilience constraint from §7.1 ("disposable") requires that the cache can be absent without breaking correctness. The fallback path must remain. The complexity cost of maintaining two paths is acceptable: the fast path is the common case; the fallback is the safety net.

### Alternative 3: Add a lookup index file (e.g., `.kbz/state/tasks/.index.json`)

Write a JSON index file alongside the YAML files mapping entity IDs to slugs. Reads consult the index file first.

**What it makes easier:** Simple; no SQLite; index survives server restarts without a rebuild step.

**What it makes harder:**
- A second file alongside YAML files introduces a new consistency obligation: the index must be updated atomically with every YAML write, or divergence occurs.
- Git tracks the index file, causing noisy commits (every entity write produces two diffs: the YAML and the index).
- Concurrent writes to the index file require locking; YAML files are already one-file-per-entity to avoid this.
- The existing SQLite cache already serves this purpose and is already excluded from git.

**Rejected because:** Adds a git-tracked mutable index with concurrent-write hazards, while the SQLite cache already provides a non-git-tracked, concurrently-safe alternative.

### Alternative 4: Status quo — do nothing

Accept the current O(n) scan. Periodically run `cleanup` manually to prune task files. Adjust expectations.

**What it makes easier:** No code change required.

**What it makes harder:**
- The `next` tool is already timing out at 447 task files. Each additional plan adds ~20 tasks. By P32 (~540 files), timeouts will be consistent and project-wide, not occasional.
- The `cleanup` tool itself scans files and is already timing out — it cannot be the mitigation for the same problem it shares.
- Manual periodic cleanup is an operational tax that compounds with each plan and has been demonstrated to be unreliable (33 stale worktrees accumulated in P28 without being caught).

**Rejected because:** The breaking point has already been reached. "Do nothing" is not a viable option for the tool most critical to daily workflow.

---

## Decisions

**Decision 1: Wire existing SQLite cache into `List()` and `Get()` read paths**
- **Context:** A SQLite cache exists, is populated on every write, and is opened at server startup, but `List()` and `Get()` do not consult it.
- **Rationale:** The design basis (§7.1) explicitly specified this cache for read acceleration. Implementing what was already decided is the lowest-risk, lowest-complexity path. All write paths already maintain the cache; the read paths are the only gap.
- **Consequences:** Reads become O(1) SQL queries after cache warm-up. The cache must be kept consistent with writes (already is, via `cacheUpsertFromResult`). The filesystem fallback remains for cold-cache scenarios.

**Decision 2: Rebuild cache at server startup (best-effort)**
- **Context:** The server opens the cache but does not rebuild it. The cache starts cold on every server restart. Post-restart tool calls — the exact scenario that caused the P28 timeout — hit a cold cache and fall through to O(n) filesystem scans.
- **Rationale:** Rebuilding at startup eliminates the cold-start failure mode. Best-effort (continue without cache on failure) preserves the "disposable" property from §7.1 and matches the existing error-handling pattern for cache open.
- **Consequences:** Server startup time increases by the time to scan all entity files once (currently O(n) but a one-time cost, not per-tool-call). This is the correct trade-off: pay once at startup rather than on every tool call.

**Decision 3: Cache miss falls back to filesystem — cache is never the sole authority**
- **Context:** An empty cache result is ambiguous: it could be a genuinely empty entity set or an unwarmed cache. Incorrect cache reads that return empty results would silently break tool correctness.
- **Rationale:** The flat-file store is the canonical source of truth (§7.1). The cache is derived and disposable. Correctness requires that a cache miss always produces the correct result via fallback, not a silent empty return.
- **Consequences:** The filesystem scan code is retained. A small number of tool calls (on cold start before rebuild completes, or after cache errors) will still pay the O(n) cost. This is acceptable: the common case is fast, the fallback is correct.

**Decision 4: Worktree store is out of scope for this plan**
- **Context:** The worktree store (`internal/worktree/store.go`) has its own `List()` and `GetByEntityID()` methods with a P28 band-aid early-termination fix. It is not backed by the entity service's SQLite cache.
- **Rationale:** The worktree store is a separate package with its own storage abstraction. Unifying it with the entity service cache in the same plan would widen scope significantly and create risk. At 45 worktree files, the performance impact is smaller than for tasks (447 files). The P28 band-aid is sufficient for the near term.
- **Consequences:** The worktree store retains its O(n) scan. A future plan can unify the caching strategy across all stores.