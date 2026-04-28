| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28                     |
| Status | Draft                          |
| Author | Claude Sonnet (via sambeau)    |

## Problem and Motivation

Kanbanzai's MCP server is the primary interface for AI agents. Every tool call that
reads entity state triggers filesystem operations, and some tool calls trigger
redundant reads of the same entity type. While the P29 design (`design-p29-state-store-read-performance`)
successfully wired the SQLite cache into `List()` and `Get()` read paths, several
gaps remain that cause unnecessary filesystem I/O on every agent interaction.

Specifically:

1. **`entityExists()` never consults the cache.** Every cross-reference validation,
   ID allocation uniqueness check, and parent-existence check hits the filesystem
   via `os.Stat` or `filepath.Glob`, even when the cache is warm and contains the
   answer. The cache's `Cache.EntityExists()` method exists and is unused.

2. **Redundant `List()` calls in the finish cascade.** When a task is completed,
   the dependency hook calls `List("task")`, the auto-advance hook calls
   `List("task")` (via `CheckAllTasksTerminal`), and the auto-advance plan hook
   calls `List("feature")` (via `CheckAllFeaturesTerminal`). With a warm cache
   these are cheap SQL queries, but each still deserialises every entity's
   `fields_json` from the cache — repeated work for the same data.

3. **The worktree store has no cache at all.** Its `List()` and `GetByEntityID()`
   methods do `os.ReadDir` and parse YAML on every call. At 45 worktrees this is
   not yet a bottleneck, but it grows linearly with project scale.

4. **Tool-level batching is incomplete.** Each `entity transition` or `entity get`
   is a separate MCP round-trip. While batch infrastructure exists
   (`internal/mcp/batch.go`), not all frequently-batched operations expose batch
   endpoints.

If nothing changes: agent tool calls will remain slower than necessary, each
cross-reference check will continue to hit the filesystem, and the user-visible
latency of operations like `finish` (which triggers a multi-step cascade) will
not improve beyond the P29 baseline.

## Design

### Overview

This design proposes three targeted improvements to close the gaps left by P29,
in order of impact:

1. **Cache-backed `entityExists()`** — add a cache fast-path that consults
   `Cache.EntityExists()` before falling back to filesystem glob.
2. **Reduce redundant cache deserialisation in cascades** — pass pre-loaded
   entity maps through the finish cascade so `CheckAllTasksTerminal` and
   `CheckAllFeaturesTerminal` receive data already in memory rather than
   re-fetching from the cache.
3. **Defer worktree store caching** — acknowledge the gap but scope it out of
   this design; the worktree count is small and the P29 design explicitly deferred it.

Tool-level batching (item 4 from the problem statement) is a tool-interface
concern, not a storage performance concern. It is acknowledged here as related
but deferred to a separate design.

### Component responsibilities

**`internal/service/entities.go` — `entityExists()`**

Current path unconditionally hits the filesystem:

```
entityExists(entityType, entityID):
    dir = entityDirectory(entityType)
    if plan type: os.Stat(dir + entityID + ".yaml")
    else:         filepath.Glob(dir + entityID + "-*.yaml")
```

New path (cache-fast-path first):

```
entityExists(entityType, entityID):
    if cache != nil && cache.IsWarm(entityType):
        return cache.EntityExists(entityType, entityID)
    // fallback: existing filesystem logic unchanged
```

The cache's `EntityExists()` is a single `SELECT COUNT(*) FROM entities WHERE
entity_type = ? AND id = ?` — O(1) indexed lookup. The fallback remains for
cold-cache scenarios.

**Callers affected:** `CreateTask` (parent feature validation), `allocateID`
(uniqueness check per allocation attempt), `HealthCheck.entityExists`
(cross-reference validation for every entity's parent/supersedes/depends_on
fields), and any future callers.

**`internal/service/entity_children.go` — cascade data passing**

Current pattern: each function in the finish cascade independently calls
`List("task")` or `List("feature")` and filters by parent ID:

```
CheckAllTasksTerminal(featureID):
    tasks = List("task")        // full scan + deserialise all tasks
    for each task:
        if task.parent != featureID: skip

CheckAllFeaturesTerminal(planID):
    features = List("feature")  // full scan + deserialise all features
    for each feature:
        if feature.parent != planID: skip
```

New pattern: add internal variants that accept pre-loaded maps:

```
checkAllTasksTerminalCached(featureID, tasksByParent map[string][]ListResult):
    for each task in tasksByParent[featureID]:
        check terminal status

checkAllFeaturesTerminalCached(planID, featuresByParent map[string][]ListResult):
    for each feature in featuresByParent[planID]:
        check terminal status
```

The public `CheckAllTasksTerminal` and `CheckAllFeaturesTerminal` retain their
signatures for external callers and internally delegate to the cached variants
after loading data once.

The cascade orchestration in the lifecycle hook loads all tasks and features
once, builds parent-indexed maps, and passes them through the chain. This
reduces N independent cache deserialisations to 2 (one for tasks, one for
features).

### Failure modes

| Failure | Handling |
|---------|----------|
| Cache `EntityExists` returns error | Log warning; fall back to filesystem glob |
| Cache is nil or cold for entity type | Fall back to filesystem (existing behaviour) |
| Pre-loaded map missing expected parent key | Fall back to `List()` — the parent may have been created after the map was built |
| Stale cache after external YAML edit | Cache rebuilt at next server start / CLI invocation; worst case is a single stale read |

### What this design does NOT change

- The YAML flat-file store remains canonical. The cache remains derived and disposable.
- No new database tables, no schema migrations.
- MCP tool signatures and external behaviour are unchanged.
- The worktree store remains uncached (deferred per P29 Decision 4).

## Alternatives Considered

### Alternative 1: Status quo — do nothing

**Description:** Accept the current state where `entityExists` hits the
filesystem and cascades re-deserialise cache data.

**What it makes easier:** No code changes. No risk of cache consistency bugs.

**What it makes harder:** Every `CreateTask`, `allocateID`, and cross-reference
check hits the filesystem. The `finish` cascade re-deserialises task and feature
JSON from the cache multiple times per call. With 898 entities and growing,
these add unnecessary latency to every agent interaction.

**Rejected because:** The gaps are well-understood, the fixes are small, and the
cache infrastructure already supports the needed operations. Leaving known
performance on the table degrades the agent development loop.

### Alternative 2: Remove the filesystem fallback entirely

**Description:** Make the cache authoritative for reads. Remove `filepath.Glob`
and `os.Stat` fallbacks from `entityExists`. If the cache is cold, rebuild it
before serving.

**What it makes easier:** Simpler code. Single code path. Guaranteed fast reads.

**What it makes harder:** A corrupt or missing cache blocks all reads. Tests
that don't configure a cache break. Violates the P29 architectural principle
that the cache is "derived, local, disposable, rebuildable."

**Rejected because:** Same reasoning as P29 Alternative 2. The fallback path is
the safety net. Removing it inverts the authority model and breaks the
disposability guarantee.

### Alternative 3: Build a shared in-memory index for cross-reference lookups

**Description:** Instead of wiring `entityExists` to the SQLite cache, build an
in-memory `map[entityType]map[id]bool` at startup and use it for all existence
checks.

**What it makes easier:** Extremely fast lookups. No SQLite dependency for
existence checks.

**What it makes harder:** Must be kept consistent with writes. Two caching
systems (SQLite + in-memory map) to maintain. Memory footprint grows with
entity count. The SQLite cache already serves this purpose.

**Rejected because:** The SQLite cache's `EntityExists` method already exists
and is fit for purpose. Adding a second lookup structure duplicates the
existing investment without sufficient benefit.

### Alternative 4: Cache the worktree store now

**Description:** Extend the SQLite cache to cover worktree records, unifying
the caching strategy across all stores.

**What it makes easier:** Worktree operations become O(1). Architecture is
more uniform.

**What it makes harder:** Worktree semantics differ from entity semantics
(path field, branch tracking, cleanup_after timestamps). The cache schema
would need extension. At 45 worktrees, the performance gain is marginal.

**Rejected for this design:** P29 Decision 4 explicitly deferred worktree
caching. The cost/benefit hasn't changed. Revisit when worktree count exceeds
~200.

## Decisions

- **Decision:** Add cache fast-path to `entityExists()` using `Cache.EntityExists()`.
  - **Context:** `entityExists` is called from `CreateTask`, `allocateID`, and
    `HealthCheck` for every cross-reference validation. Each call currently hits
    the filesystem via `os.Stat` or `filepath.Glob`. The cache's `EntityExists()`
    method exists but is never consulted.
  - **Rationale:** This is the highest-ROI remaining gap from P29. The fix is a
    single-location change with an already-existing cache method. It eliminates
    filesystem I/O for every cross-reference check when the cache is warm.
  - **Consequences:** Filesystem fallback remains for cold-cache scenarios.
    Cache consistency is maintained by the existing write-through pattern
    (every write calls `cacheUpsertFromResult`).

- **Decision:** Add pre-loaded variants of `CheckAllTasksTerminal` and
  `CheckAllFeaturesTerminal` for use in the finish cascade.
  - **Context:** The finish cascade (`CompleteTask` → `evaluateDependents` →
    `MaybeAutoAdvanceFeature` → `MaybeAutoAdvancePlan`) currently causes each
    child-check function to independently call `List()` and deserialise all
    entities from the cache. This is redundant work for data already loaded.
  - **Rationale:** Passing pre-loaded parent-indexed maps through the cascade
    reduces N cache deserialisations to 2. The public API is preserved for
    external callers.
  - **Consequences:** The internal cascade functions gain an additional
    parameter. External callers are unaffected. The pre-loaded maps introduce
    a minor staleness window for entities created mid-cascade — this is
    acceptable since cascades complete in milliseconds and the caller retries
    on conflict.

- **Decision:** Defer worktree store caching.
  - **Context:** The worktree store has no cache. At 45 worktrees, the
    performance impact is negligible. P29 Decision 4 explicitly scoped this
    out.
  - **Rationale:** The cost of extending the cache schema and wiring it into
    the worktree store exceeds the benefit at current scale. Revisit when
    worktree count exceeds ~200.
  - **Consequences:** Worktree operations remain O(n) filesystem scans. This
    is acceptable at current scale.

- **Decision:** Defer tool-level batching improvements.
  - **Context:** Batch infrastructure exists but not all operations expose
    batch endpoints. This is a tool-interface concern, not a storage
    performance concern.
  - **Rationale:** Batching reduces MCP round-trips, not computation time.
    It is a separate design space from storage read-path performance.
  - **Consequences:** Agents continue to make individual MCP tool calls for
    entity operations. This is acceptable; the batch infrastructure can be
    extended independently.

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `design-p29-state-store-read-performance` | Design | Direct predecessor — wired cache into `List()` and `Get()`. The gaps this design addresses are the remaining work P29 scoped out or missed. |
| `feat-01kpxgzxx8bjz-cache-warm-up-server-start` | Spec | Specifies cache warm-up at server start — the mechanism this design relies on for cache availability. |
| `feat-01kpxh0f5gfnv-cache-first-read-path` | Spec | Specifies the cache-first pattern in `List()` and `Get()` — the pattern this design extends to `entityExists()`. |
| `design-workflow-design-basis` §7.1 | Design | Specifies the SQLite cache as "derived, local, disposable, rebuildable" — the architectural constraint this design respects. |

### Decisions that constrain this design

| Decision | Source | Constraint |
|----------|--------|------------|
| "Cache miss falls back to filesystem — cache is never the sole authority" | P29 Decision 3 | `entityExists` must preserve the filesystem fallback |
| "Worktree store is out of scope" | P29 Decision 4 | This design does not cache worktrees |
| "SQLite cache is derived and disposable" | workflow-design-basis §7.1 | The cache can be deleted and rebuilt; no feature may depend on its presence |
