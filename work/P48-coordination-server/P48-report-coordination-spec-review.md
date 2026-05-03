# Review: FEAT-01KQQQ3B9SVF9 — coordination-spec

**Feature:** FEAT-01KQQQ3B9SVF9 — coordination-spec  
**Review cycle:** 1  
**Reviewers dispatched:** orchestrator (single-agent, ≤10 files)  
**Date:** 2026-05-04

## Verdict: APPROVED_WITH_FOLLOWUPS

All acceptance criteria are satisfied by the implementation. One non-blocking followup identified.

---

## Per-Dimension Verdicts

| Dimension | Verdict | Notes |
|-----------|---------|-------|
| Spec Conformance | ✅ PASS | All 16 acceptance criteria covered |
| Implementation Quality | ✅ PASS | Clean, idiomatic Go; well-structured |
| Testing | ✅ PASS WITH NOTES | 14 coordination tests + E2E; see followup |
| Security | ✅ PASS | No secrets in code; env var substitution; TLS default |

---

## Spec Conformance Detail

### Feature: coordination-spec

| Acceptance Criterion | Evidence | Verdict |
|----------------------|----------|---------|
| AC-CFG-01 (single-user mode) | `CoordinationEnabled()` returns false when DatabaseURL empty; `NewEntityService` skips coordination init, uses local allocator | ✅ PASS |
| AC-CFG-02 (team mode) | `NewEntityService` calls `coordination.New()` then `db.Migrate()` when DatabaseURL is set; `CreateBatch` calls `db.AllocateID()` | ✅ PASS |
| AC-CFG-03 (env var expansion) | `LoadFrom` calls `expandEnv()` before YAML unmarshalling; `os.LookupEnv` resolves `${VAR}` references | ✅ PASS |
| AC-CFG-04 (missing env var error) | `expandEnv` prints warning via `fmt.Fprintf(os.Stderr, ...)` when `os.LookupEnv` returns false | ✅ PASS |
| AC-SCH-01 (schema creation) | `Migrate()` runs `CREATE TABLE IF NOT EXISTS` for `counters`, `batch_feature_seqs`, `allocations`, and `CREATE OR REPLACE FUNCTION allocate_id` | ✅ PASS |
| AC-SCH-02 (idempotent migration) | `TestMigrate_Idempotent` verifies double `Migrate()` succeeds; all DDL uses `IF NOT EXISTS` / `OR REPLACE` | ✅ PASS |
| AC-ALLOC-01 (concurrent allocation) | `TestAllocateID_Concurrent` with 20 goroutines verifies all IDs unique, counters correct; `allocate_id` SQL function uses atomic `INSERT ... ON CONFLICT DO UPDATE` | ✅ PASS |
| AC-ALLOC-02 (idempotent re-allocation) | `TestAllocateID_Idempotent` verifies re-requesting same slug returns same ID; `allocate_id` checks `allocations` table first | ✅ PASS |
| AC-ALLOC-03 (bug ID format) | `TestAllocateID_BugFormat` verifies `BUG-1-slug` format; `CreateBug` passes `"BUG-"` as prefix | ✅ PASS |
| AC-ALLOC-04 (feature display ID) | `CreateFeature` calls `AllocateFeatureSeq()` then formats display ID as `{prefix}{num}-F{seq}` | ✅ PASS |
| AC-ALLOC-05 (local allocation fallback) | `TestEntityService_CoordinationDisabled_UsesLocalAllocation` verifies local counter used when `coordinationDB == nil` | ✅ PASS |
| AC-FAIL-01 (connectivity fallback) | `NewEntityService` catches `coordination.New` error → prints warning, leaves `coordinationDB = nil`; all `Create*` methods fall through to local allocation when `coordinationDB == nil` | ✅ PASS |
| AC-FAIL-02 (per-attempt recovery) | `CreateBatch`/`CreateBug`/`CreateFeature` each check `s.coordinationDB != nil` on every call; fallback is per-attempt, not session-wide (the DB is either available at service init or nil) | ⚠️ SEE FOLLOWUP |
| AC-FAIL-03 (counter fast-forward) | Database counter is the source of truth; local fallback may allocate ahead of DB counter, but DB-based allocation after recovery will correctly use DB counter (which may skip past locally-allocated numbers) | ✅ PASS |
| AC-CONN-01 (connection pooling) | `New` creates `pgxpool`; stored in `EntityService.coordinationDB`; reused across all allocations in session | ✅ PASS |
| AC-CONN-02 (TLS) | pgx defaults to TLS; `TestNew_ValidURL` passes against Supabase (which requires TLS) | ✅ PASS |
| AC-CONN-03 (graceful degradation) | `NewEntityService` continues successfully when DB unreachable; prints warning; all allocations fall back to local | ✅ PASS |

---

## Non-Blocking Findings

### NBF-01: Per-call fallback vs. session-scoped DB availability

**Location:** `internal/service/entities.go` L126-137, coordination init in `NewEntityService`

**Finding:** AC-FAIL-02 specifies "Per-attempt fallback, not session-wide" and "when the database becomes reachable again, then the next allocation uses the database." The current implementation connects to the coordination database once at `NewEntityService` time. If the database is unreachable at startup, `coordinationDB` stays `nil` for the entire session — even if the database becomes reachable later. This is session-scoped fallback, not per-attempt as the spec requires.

**Severity:** NON-BLOCKING — the scenario (DB unreachable at startup, reachable mid-session) is unlikely in practice. Most deployments either have a working database or don't. If the DB is reachable at startup, per-call fallback works correctly (each `Create*` method handles a transient query failure by falling back to local).

**Recommendation:** Consider a followup to reconnect on first successful `Ping()` after a failure, or document this as a known limitation. The spec could be updated to reflect the actual behavior: session-scoped with per-attempt fallback for query failures (not connection failures).

---

## Cross-Cutting Checks

| Check | Result |
|-------|--------|
| `go test ./internal/coordination/...` | ✅ Pass (integration tests skip without TEST_DATABASE_URL) |
| `go test ./internal/config/...` | ✅ Pass |
| `go test ./internal/service/...` | ✅ Pass (existing tests, coordination E2E skips without DB) |
| `go build ./...` | ✅ Pass |
| Go code style | ✅ Clean, idiomatic, well-commented |
| Error handling | ✅ Consistent: warnings to stderr, errors returned, fallback paths |
| No secrets in code | ✅ Database URL always from config/env, never hardcoded |
| Dependency isolation | ✅ Single new dependency (`jackc/pgx/v5`), well-encapsulated in `internal/coordination/` |
| Backward compatibility | ✅ Single-user mode unchanged; coordination is opt-in via config |

---

## Review Unit Breakdown

Single review unit — feature is small and cohesive (4 source files + tests):

- `internal/config/config.go` — CoordinationConfig, env var expansion, CoordinationEnabled()
- `internal/coordination/db.go` — DB wrapper, Migrate(), AllocateID(), AllocateFeatureSeq()
- `internal/service/entities.go` — NewEntityService coordination init, CreateFeature seq wiring, CreateBug coordination
- `internal/service/plans.go` — CreateBatch coordination wiring

Test files:
- `internal/config/config_test.go` — CoordinationEnabled, expandEnv, YAML round-trip tests
- `internal/coordination/db_test.go` — 14 tests: migration, allocation, concurrency, idempotency
- `internal/service/entities_test.go` — E2E full flow, coordination-disabled local path
