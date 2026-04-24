# Review: P29 State Store Read Path Performance

Feature batch: FEAT-01KPXGZXX8BJZ, FEAT-01KPXH0F5GFNV, FEAT-01KPXH0WSTHAM  
Plan: P29-state-store-read-performance  
Review cycle: 1

---

## Review units

1. `server-startup-cache-warmup`
   - `internal/mcp/server.go`
   - `internal/mcp/server_warmup_test.go`
2. `entity-cache-read-path`
   - `internal/cache/cache.go`
   - `internal/cache/cache_test.go`
   - `internal/service/entities.go`
   - `internal/service/entities_cache_read_test.go`
3. `entity-cache-eviction-tests`
   - `internal/service/entities_cache_test.go`

---

## Per-reviewer summary

### Reviewer: conformance
Review units: all three
Verdict: fail
Dimensions:
- warm-up feature: pass
- cache-first read path: fail
- eviction invariants: pass
Findings: 1 blocking, 1 non-blocking

### Reviewer: quality
Review unit: entity-cache-read-path
Verdict: fail
Dimensions:
- implementation quality: concern
- fallback semantics: fail
Findings: 1 blocking, 1 non-blocking

### Reviewer: testing
Review units: all three
Verdict: fail
Dimensions:
- warm-up feature: concern
- cache-first read path: concern
- eviction invariants: pass
Findings: 2 non-blocking

### Reviewer: security
Review unit: entity-cache-read-path
Verdict: pass
Dimensions:
- security-relevant correctness: pass
Findings: 0

---

## Collated findings (deduplicated)

### [B-1] (blocking)
Dimension: conformance, quality
Location: `internal/service/entities.go:147-159`
Spec ref: FEAT-01KPXH0F5GFNV `FR-008`, `AC-012`, `FR-011`, `AC-013`, `AC-014`
Description: `RebuildCache()` only rebuilds `feature`, `task`, `bug`, and `decision` entities. The cache-first read path is implemented generically in `Get()` and `List()`, but `plan` entities can never become warm via rebuild after server startup. That means the feature does not satisfy its own generic per-entity-type warm-cache contract after a restart.
Reported by: conformance, quality

### [NB-1] (non-blocking)
Dimension: testing
Location: `internal/mcp/server_warmup_test.go:24-126`
Spec ref: FEAT-01KPXGZXX8BJZ `AC-005`, `AC-006`
Description: The warm-up tests cover rebuild success/failure and no-cache regression, but they do not verify the required success/failure log output containing entity count, duration, and error text.
Reported by: testing

### [NB-2] (non-blocking)
Dimension: quality, testing
Location: `internal/service/entities.go:501-518`, `internal/service/entities.go:555-558`, `internal/service/entities_cache_read_test.go:29-611`
Spec ref: FEAT-01KPXH0F5GFNV `FR-003`, `FR-006`, `AC-004`, `AC-008`
Description: The cache-first implementation falls back correctly, but it does not log cache read-path failures before fallback, even though the spec requires cache errors to be logged. The tests also do not assert this observability contract.
Reported by: quality, testing

### [NB-3] (non-blocking)
Dimension: testing
Location: `internal/service/entities.go:503-518`, `internal/service/entities_cache_read_test.go:29-611`
Spec ref: FEAT-01KPXH0F5GFNV `FR-002`, `FR-003`
Description: There is no focused test for the stale-cache-entry branch in `Get()` where `LookupByID()` succeeds but `store.Load()` fails and the code falls back to `ResolvePrefix()`.
Reported by: testing

---

## Aggregate verdict

**Aggregate Verdict: rejected**

Rationale:
- Two of the three reviewed features are acceptable with follow-ups:
  - `FEAT-01KPXGZXX8BJZ` — approved with follow-ups
  - `FEAT-01KPXH0WSTHAM` — approved
- `FEAT-01KPXH0F5GFNV` has one blocking conformance/correctness issue: `RebuildCache()` does not warm all entity types served by the generic cache-first read path, so the feature does not fully satisfy its own spec after restart.

---

## Feature-by-feature verdicts

### FEAT-01KPXGZXX8BJZ — cache-warm-up-server-start
Verdict: approved_with_followups

Evidence:
- Startup warm-up is wired immediately after `SetCache()` and before the rest of server construction continues: `internal/mcp/server.go:90-99`
- Success and failure logs are present with count/duration and error text respectively: `internal/mcp/server.go:93-98`
- Tests cover rebuild success, rebuild failure on closed DB, nil-cache behavior, and no-cache `Get`/`List` regression: `internal/mcp/server_warmup_test.go:24-126`

Follow-up:
- Add explicit log-output assertions for AC-005 and AC-006.

### FEAT-01KPXH0F5GFNV — cache-first-read-path
Verdict: rejected

Evidence:
- `Get()` fast path is implemented and falls back correctly on miss: `internal/service/entities.go:499-523`
- `List()` fast path is implemented, returns error on corrupt `fields_json`, and preserves warm-empty semantics: `internal/service/entities.go:545-579`
- `IsWarm()` is implemented as in-process state and updated by `Upsert()` and `Rebuild()`: `internal/cache/cache.go:31-76`, `internal/cache/cache.go:107-130`, `internal/cache/cache.go:219-278`
- Tests cover fast paths, nil cache, cold type, corrupt JSON, `ListByType` error fallback, and result equivalence: `internal/service/entities_cache_read_test.go:29-611`
- Blocking issue: `RebuildCache()` omits `plan`, so not all entity types can become warm after startup: `internal/service/entities.go:147-159`

### FEAT-01KPXH0WSTHAM — cache-eviction-on-delete
Verdict: approved

Evidence:
- Tests verify terminal-state transitions upsert rather than evict cache rows: `internal/service/entities_cache_test.go:11-149`
- Tests verify slug rename updates cached slug in place: `internal/service/entities_cache_test.go:195-262`
- Tests verify nil-cache safety for `UpdateEntity`, `Get`, and `List`: `internal/service/entities_cache_test.go:264-373`
- Tests verify `cache.Delete()` contract directly, including non-existent row behavior: `internal/service/entities_cache_test.go:375-438`

---

## Remediation plan

1. [B-1] Extend `RebuildCache()` to include `plan` entities, or explicitly narrow the cache-first feature scope/spec so the implementation and contract match.
2. Add log assertions for warm-up success/failure tests.
3. Add logging for cache read-path fallback errors in `Get()` and `List()`, plus tests if the spec continues to require that observability.
4. Add a focused stale-cache-entry fallback test for `Get()`.
