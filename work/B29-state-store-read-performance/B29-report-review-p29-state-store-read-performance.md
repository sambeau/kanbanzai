# Review: P29 State Store Read Path Performance

Feature batch: FEAT-01KPXGZXX8BJZ, FEAT-01KPXH0F5GFNV, FEAT-01KPXH0WSTHAM
Plan: P29-state-store-read-performance
Review cycle: 1 (initial) + remediation

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
Verdict: pass
Dimensions:
- warm-up feature: pass
- cache-first read path: pass (after remediation)
- eviction invariants: pass
Findings: 0 blocking, 0 non-blocking (post-remediation)

### Reviewer: quality
Review unit: entity-cache-read-path
Verdict: pass
Dimensions:
- implementation quality: pass (after NB-2 fix)
- fallback semantics: pass (after B-1 false-positive resolution)
Findings: 0 blocking, 0 non-blocking (post-remediation)

### Reviewer: testing
Review units: all three
Verdict: pass
Dimensions:
- warm-up feature: pass (after NB-1 fix)
- cache-first read path: pass (after NB-2 and NB-3 fixes)
- eviction invariants: pass
Findings: 0 (post-remediation)

### Reviewer: security
Review unit: entity-cache-read-path
Verdict: pass
Dimensions:
- security-relevant correctness: pass
Findings: 0

---

## Collated findings (deduplicated)

### [B-1] ~~(blocking)~~ → FALSE POSITIVE — resolved by code comment
Dimension: conformance, quality
Location: `internal/service/entities.go:147-159`
Spec ref: FEAT-01KPXH0F5GFNV `FR-008`, `AC-012`, `FR-011`, `AC-013`, `AC-014`

**Original description:** `RebuildCache()` only rebuilds `feature`, `task`, `bug`, and
`decision` entities. The cache-first read path is implemented generically in `Get()` and
`List()`, but `plan` entities can never become warm via rebuild after server startup.

**False-positive determination:** Investigated and confirmed that `EntityService.List("plan")`
and `EntityService.Get("plan", ...)` cannot succeed regardless of cache state. Plan files use a
slug-free `{id}.yaml` naming convention (`entityFileName` in `storage/entity_store.go` special-
cases plan type), but `storage.EntityStore.Load()` requires a non-empty slug and returns an
error when slug is empty. `parseRecordIdentity` for plan type always returns `slug=""`. As a
result, the plan entity type is structurally unsupported by `EntityService` — plans are managed
by a separate plan service. `grep` confirms `List("plan")` is called nowhere in the codebase.
The omission from `RebuildCache()` is therefore intentional and correct.

**Resolution:** Added a comment to `RebuildCache()` documenting the intentional exclusion of
`plan` so that future contributors understand the constraint.

Reported by: conformance, quality

---

### [NB-1] (non-blocking) → FIXED
Dimension: testing
Location: `internal/mcp/server_warmup_test.go`
Spec ref: FEAT-01KPXGZXX8BJZ `AC-005`, `AC-006`

**Description:** The warm-up tests covered rebuild success/failure and no-cache regression
but did not verify the required success/failure log output.

**Fix:** Added two sequential (non-parallel) tests that capture the global logger output and
assert the expected content:
- `TestRebuildCache_SuccessLogContainsCountAndDuration` — asserts log output contains
  `"[server] cache warm-up: loaded"`, a non-zero digit, and `" entities in "` (AC-005)
- `TestRebuildCache_FailureLogContainsErrorText` — asserts log output contains
  `"continuing without cache"` and the exact error string (AC-006)

Reported by: testing

---

### [NB-2] (non-blocking) → FIXED
Dimension: quality, testing
Location: `internal/service/entities.go` (Get and List fast-path fallback branches)
Spec ref: FEAT-01KPXH0F5GFNV `FR-003`, `FR-006`, `AC-004`, `AC-008`

**Description:** The cache-first implementation fell back to the filesystem correctly but
did not emit a log entry before doing so, violating the spec's observability requirement
for cache errors.

**Fix:**
- In `Get()`: added `log.Printf("[entity] cache hit but Load failed for %s/%s (falling back): %v", ...)` in the stale-entry fallback branch (after `store.Load` fails on a cache hit)
- In `List()`: added `log.Printf("[entity] cache ListByType error for %s (falling back): %v", ...)` in the `ListByType` error fallback branch

Reported by: quality, testing

---

### [NB-3] (non-blocking) → FIXED
Dimension: testing
Location: `internal/service/entities_cache_read_test.go`
Spec ref: FEAT-01KPXH0F5GFNV `FR-002`, `FR-003`

**Description:** No test exercised the branch where `LookupByID()` succeeds but
`store.Load()` fails, causing `Get()` to fall back to `ResolvePrefix()`.

**Fix:** Added `TestGet_StaleCache_FallsBack`. The test injects a fabricated cache row
(`FEAT-01FAKEDEADBEEF`) pointing to `/nonexistent/path.yaml` into a warm `"feature"` cache,
then calls `Get("feature", "FEAT-01FAKEDEADBEEF", "")`. Asserts that:
- The call does not panic
- Get falls back through ResolvePrefix (which also finds no file) and returns a non-nil error
- The stale-cache-entry path does not return a corrupt result

Reported by: testing

---

## Aggregate verdict

**Aggregate Verdict: approved**

All three features are approved for merge.

| Feature | Cycle 1 verdict | Post-remediation verdict |
|---------|----------------|--------------------------|
| FEAT-01KPXGZXX8BJZ — cache-warm-up-server-start | approved_with_followups | **approved** |
| FEAT-01KPXH0F5GFNV — cache-first-read-path | rejected | **approved** |
| FEAT-01KPXH0WSTHAM — cache-eviction-on-delete | approved | **approved** |

---

## Feature-by-feature verdicts (final)

### FEAT-01KPXGZXX8BJZ — cache-warm-up-server-start
**Verdict: approved**

Evidence:
- Startup warm-up wired immediately after `SetCache()`, before server begins serving
  requests: `internal/mcp/server.go:90-99`
- Success log emits count and elapsed duration; failure log emits error and
  "continuing without cache": `internal/mcp/server.go:93-98`
- Tests cover rebuild success (with cache population check), rebuild failure on closed DB,
  nil-cache behavior, no-cache `Get`/`List` regression, and log-output assertions for
  AC-005 and AC-006: `internal/mcp/server_warmup_test.go`

---

### FEAT-01KPXH0F5GFNV — cache-first-read-path
**Verdict: approved**

Evidence:
- `Get()` fast path: consults warm cache before `ResolvePrefix()`; falls back on miss,
  stale-load failure (now logged), or nil/cold cache:
  `internal/service/entities.go:499-527`
- `List()` fast path: serves warm-cache reads from `fields_json`; returns error on corrupt
  row; falls back on `ListByType` error (now logged); warm-empty type returns `[]` without
  fallback: `internal/service/entities.go:545-582`
- `IsWarm()` backed by in-process `map[string]bool`; updated on `Upsert()` and `Rebuild()`;
  returns `false` for any type not yet seen this session regardless of persisted SQLite rows:
  `internal/cache/cache.go:71-76`, `internal/cache/cache.go:107-130`, `internal/cache/cache.go:219-278`
- `RebuildCache()` intentionally excludes `plan` — comment added at `entities.go:147`
  explaining that `EntityService.List("plan")` is unsupported due to plan files using a
  slug-free naming convention; plans are managed by a separate plan service
- Full test suite: fast paths, nil cache, cold type, corrupt JSON, `ListByType` error
  fallback, stale-cache-entry fallback (new), result equivalence:
  `internal/service/entities_cache_read_test.go`
- `go test ./internal/cache/... ./internal/service/...` passes

---

### FEAT-01KPXH0WSTHAM — cache-eviction-on-delete
**Verdict: approved**

Evidence:
- `UpdateStatus` upserts (not evicts) cache rows for all transitions including terminal
  states: `internal/service/entities_cache_test.go:11-149`
- Slug rename via `UpdateEntity` updates cached slug in place (no stale row):
  `internal/service/entities_cache_test.go:195-262`
- Nil-cache safety for `UpdateEntity`, `Get`, and `List`: no panic, correct results:
  `internal/service/entities_cache_test.go:264-373`
- `cache.Delete()` eviction API contract: `LookupByID` returns `found=false` after delete;
  delete on non-existent row returns nil:
  `internal/service/entities_cache_test.go:375-438`
- `go test ./internal/service/...` passes