# Specification: Cache Warm-Up at Server Start

**Feature:** FEAT-01KPXGZXX8BJZ  
**Plan:** P29-state-store-read-performance  
**Status:** Draft  

---

## 1. Related Work

| Artifact | Description |
|----------|-------------|
| `work/design/p29-state-store-read-performance.md` | Approved design document for P29; Decision 2 specifies best-effort cache warm-up at startup |
| `internal/cache` | SQLite-backed cache for entity state |
| `internal/mcp/server.go` | MCP server entry point where cache is opened and injected |
| `internal/service/entities.go` | `EntityService`, including `RebuildCache()` |

---

## 2. Overview

The Kanbanzai MCP server opens a SQLite cache at startup but does not populate it, leaving the cache cold. Cold-cache tool calls fall through to O(n) filesystem scans, which cause MCP timeouts at the current scale of ~447 task files.

This feature ensures the cache is populated immediately after it is opened at server startup, so that the first tool calls after startup benefit from cached reads. Warm-up is best-effort: a failure must not prevent the server from starting.

---

## 3. Scope

### In Scope

- Triggering a cache rebuild as part of the server startup sequence, after the cache is opened and injected into `EntityService`.
- Logging outcomes (success, entity count, duration, and any error) of the warm-up attempt.
- Ensuring a warm-up failure does not abort server startup.

### Explicitly Excluded

- Changes to `EntityService.List()`, `EntityService.Get()`, or any other read path.
- Cache eviction or invalidation logic.
- Cache-first read strategies or read-path fallback changes.
- Any warm-up triggered outside of server startup (e.g. on-demand, periodic, or lazy).
- Changes to `cache.Open()`, `cache.Rebuild()`, or any cache internals.

---

## 4. Functional Requirements

**FR-001 — Warm-up triggered at startup**  
When the MCP server starts and the cache opens successfully, a cache rebuild must be initiated before the server begins serving requests.

**FR-002 — Best-effort: failure does not abort startup**  
If the cache rebuild fails for any reason, the server must continue starting up and serve requests without a warm cache. The failure must not be surfaced as a fatal error.

**FR-003 — Success logged with entity count and duration**  
When the cache rebuild completes successfully, the server must emit a log entry that includes the number of entities loaded and the elapsed time.

**FR-004 — Failure logged with error detail**  
When the cache rebuild fails, the server must emit a log entry that includes the error and indicates that the server is continuing without a warm cache.

**FR-005 — No warm-up when cache is unavailable**  
When `cache.Open()` fails (e.g. the cache directory cannot be created or the database cannot be opened), no warm-up attempt must be made.

---

## 5. Non-Functional Requirements

**NFR-001 — Warm-up must complete before request serving begins**  
The cache rebuild must complete (or fail) synchronously during server startup, before the MCP transport begins accepting tool calls.

**NFR-002 — Warm-up must not block startup indefinitely**  
The warm-up must not cause the server startup sequence to hang. If `RebuildCache()` blocks, the blocking behaviour is bounded by the existing implementation; no additional timeout is imposed by this feature.

**NFR-003 — No regression to existing behaviour on cache-open failure**  
When the cache cannot be opened, server startup and request-handling behaviour must be identical to the behaviour before this feature was introduced.

---

## 6. Acceptance Criteria

### FR-001
- Given a server start with a reachable, writable cache directory, the cache is populated with entity data before the first tool call is handled.
- A cache that was cold before startup returns cached results for entity queries issued immediately after startup.

### FR-002
- Given a `RebuildCache()` call that returns an error, the server proceeds to the request-serving phase without exiting or panicking.
- Tool calls issued after a failed warm-up are handled (via filesystem fallback) without error attributable to the warm-up failure.

### FR-003
- Given a successful warm-up, the server log contains an entry at INFO level (or equivalent) that includes both a numeric entity count and an elapsed duration.

### FR-004
- Given a warm-up that returns an error, the server log contains an entry at WARN level (or equivalent) that includes the error text and a message indicating the server is running without a warm cache.

### FR-005
- Given a `cache.Open()` error at startup, no call to `RebuildCache()` is made and the server starts normally.
- The server log does not contain any warm-up success or failure entry in this scenario.

### NFR-001
- A tool call issued to a freshly started server (with a healthy cache) returns a response drawn from the cache, not from a filesystem scan.

### NFR-003
- End-to-end tests that simulate a missing or unwritable cache directory pass without change.

---

## 7. Dependencies and Assumptions

| Item | Detail |
|------|--------|
| `cache.Open()` | Must be called before warm-up; this feature assumes the existing call site in `server.go` is unchanged. |
| `EntityService.SetCache()` | Must be called before `RebuildCache()`; warm-up is only meaningful after the cache is injected. |
| `EntityService.RebuildCache()` | Returns `(int, error)`; the int is the count of entities loaded. This signature is assumed stable. |
| Filesystem state | Entity files must be present and readable for warm-up to succeed; this feature makes no guarantees about the content of the cache if files are corrupted or missing. |
| Single server instance | This feature assumes a single MCP server process; concurrent warm-ups from multiple processes are out of scope. |
