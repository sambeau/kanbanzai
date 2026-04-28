# Implementation Plan: Cache Warm-Up at Server Start (FEAT-01KPXGZXX8BJZ)

| Field      | Value                                              |
|------------|----------------------------------------------------|
| Feature    | FEAT-01KPXGZXX8BJZ                                 |
| Plan       | P29-state-store-read-performance                   |
| Spec       | work/specs/feat-01kpxgzxx8bjz-cache-warm-up-server-start.md |
| Author     | Claude Sonnet 4.6                                  |

---

## Scope

Wire `entitySvc.RebuildCache()` into the server startup sequence in `internal/mcp/server.go`
so the SQLite cache is warm before the first tool call is handled. Add logging on both
success and failure paths. Write tests.

Two files change: `internal/mcp/server.go` (implementation) and one test file.

---

## Task Breakdown

### TASK-01KPY-F49K4BGH — Wire RebuildCache call into server startup

**File:** `internal/mcp/server.go`

Inside the existing `if c, err := cache.Open(cacheDir); err == nil { ... }` block,
after `entitySvc.SetCache(c)`, add a best-effort `RebuildCache` call:

- Record start time before the call.
- On success: `log.Printf("[server] cache warm-up: loaded %d entities in %s", n, elapsed)`
- On failure: `log.Printf("[server] cache warm-up failed (continuing without cache): %v", err)`

No other files change for this task.

### TASK-01KPY-F4Z4T23K — Write server startup warm-up tests

**Files:** `internal/mcp/server_test.go` or `internal/service/entities_test.go`

Test scenarios:
- Cache open succeeds and `RebuildCache` succeeds → entities queryable from cache before first call (AC-001, AC-002)
- Cache open succeeds but `RebuildCache` returns error → server proceeds; tool calls fall back to filesystem (AC-003, AC-004)
- Success path emits log entry containing entity count and duration string (AC-005)
- Failure path emits log entry containing error text (AC-006)
- Cache open fails → `RebuildCache` is never called; server proceeds normally (AC-007)
- No-cache path: server with nil cache behaves identically to pre-feature behaviour (AC-008, AC-009)

---

## Dependency Graph

```
TASK-01KPY-F49K4BGH (implement)
        │
        ▼
TASK-01KPY-F4Z4T23K (tests)
```

---

## Risk Assessment

**Risk: `RebuildCache` is slow on large repos**
- At ~450 entity files, rebuild is expected to complete in well under 1 second.
- Best-effort handling means a slow rebuild does not block the server indefinitely in
  practice; however no explicit timeout is added (per NFR-002).
- Mitigation: the startup log records the elapsed time, making any regression observable.

**Risk: Test for log output is fragile**
- Log output format is an implementation detail. Tests should match on substrings
  (count, "warm-up") rather than exact format strings.

---

## Verification Approach

- `go test ./internal/mcp/...` and `go test ./internal/service/...` must pass.
- Manual smoke test: start server, run `entity(action: list, type: task)` immediately —
  confirm response time is fast (cache hit) and log shows warm-up entry.
- Confirm no regression: start server with cache directory unwritable, run same tool call —
  confirm server starts and responds (filesystem fallback).