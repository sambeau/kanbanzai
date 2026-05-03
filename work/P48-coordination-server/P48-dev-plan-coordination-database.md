| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03T20:19:17Z           |
| Status | approved |
| Author | Sambeau                        |

# Coordination Database — Dev Plan

## Overview

This dev-plan breaks the coordination database specification into 7 implementation tasks. The plan follows a bottom-up dependency order: config and connection infrastructure first (T1–T3), then wiring into entity creation (T4–T5), then fallback logic (T6), and finally integration tests (T7). T4 and T5 are parallelisable once T1 and T3 complete. The critical path is T1 → T2 → T3 → T4 → T6 → T7.

## Scope

This plan implements the requirements defined in `work/P48-coordination-server/P48-spec-coordination-database.md` (FEAT-01KQQQ3B9SVF9/spec-p48-spec-coordination-database). It covers tasks T1–T7 below plus integration testing.

It does **not** cover: the additional coordination functions deferred by the design (knowledge entries, checkpoints, worktree tracking, merge locks, entity status cache), migration of existing TSID-based bug IDs, or the full centralized state server.

## Task Breakdown

### T1: Add CoordinationConfig to project config struct

- **Description:** Add a `Coordination` struct with `DatabaseURL` and `ProjectID` fields to the project `Config` type. Implement environment variable substitution for any string config value containing `${ENV_VAR}` syntax. Add a `CoordinationEnabled()` method that returns true when `DatabaseURL` is non-empty.
- **Deliverable:** Updated `internal/config/config.go` with `CoordinationConfig` type and env var substitution logic. Updated `DefaultConfig()`.
- **Depends on:** None.
- **Effort:** Small (2-3 story points).
- **Spec requirements:** REQ-CFG-001, REQ-CFG-002, REQ-CFG-003, REQ-CFG-004.

### T2: Create the coordination Go package with pgx connection pool

- **Description:** Create `internal/coordination/` package containing a `DB` struct that wraps `pgxpool.Pool`. Provide `New()` that creates the pool from a database URL, `Close()`, and `Ping()`. The pool must use TLS. On startup failure (unreachable host), return an error that callers can use to decide fallback behaviour — do not block server startup.
- **Deliverable:** `internal/coordination/db.go` with `DB` struct and pool management. `go.mod` updated with `github.com/jackc/pgx/v5` dependency.
- **Depends on:** T1 (needs CoordinationConfig to read the URL).
- **Effort:** Medium (3-5 story points).
- **Spec requirements:** REQ-CONN-001, REQ-CONN-002, REQ-CONN-003, REQ-NF-005.

### T3: Implement schema migration and the allocate_id SQL function

- **Description:** Add a `Migrate()` method on the coordination `DB` that runs `CREATE TABLE IF NOT EXISTS` for `counters`, `batch_feature_seqs`, and `allocations` tables, and `CREATE OR REPLACE FUNCTION allocate_id` matching the design document's SQL exactly. Add an `AllocateID()` method that calls `SELECT allocate_id($1, $2, $3, $4)` and returns the ID string.
- **Deliverable:** `internal/coordination/db.go` extended with `Migrate()` and `AllocateID()` methods. The `allocate_id` function embedded as a Go constant containing the SQL DDL.
- **Depends on:** T2 (needs the DB pool).
- **Effort:** Medium (3-5 story points).
- **Spec requirements:** REQ-SCH-001, REQ-SCH-002, REQ-ALLOC-001 through REQ-ALLOC-004.

### T4: Wire coordination into entity creation — plan, batch, and bug

- **Description:** Modify the entity creation path in the `entity` MCP tool. When `coordination.database_url` is configured: before writing the entity YAML file, call the coordination database to allocate the ID. For plans: `entity_type = 'plan_{prefix}'`. For batches: `entity_type = 'batch_{prefix}'`. For bugs: `entity_type = 'bug'`, prefix `'BUG'`. Use the returned ID as the entity's canonical/bug ID. In single-user mode (no `database_url`), behaviour is unchanged.
- **Deliverable:** Modified `internal/mcp/entity_tool.go` (or wherever entity creation dispatch happens) with coordination calls gated on config.
- **Depends on:** T3 (needs AllocateID), T1 (needs config gate).
- **Effort:** Medium (3-5 story points).
- **Spec requirements:** REQ-ALLOC-001, REQ-ALLOC-002, REQ-ALLOC-005, REQ-ALLOC-006, REQ-ALLOC-010.

### T5: Wire coordination into feature display ID allocation

- **Description:** Modify the feature creation path. When `coordination.database_url` is configured: atomically increment `batch_feature_seqs.next_seq` for the parent batch to get the display sequence number. The feature's canonical ID remains TSID-based. The display ID (e.g., `B12-F3`) is constructed from the batch ID and the allocated sequence number. In single-user mode, use the existing local `next_feature_seq` counter as before.
- **Deliverable:** Modified feature creation code with coordination-backed display ID allocation.
- **Depends on:** T3 (needs DB), T1 (needs config gate).
- **Effort:** Small-Medium (2-3 story points).
- **Spec requirements:** REQ-ALLOC-007, REQ-ALLOC-008, REQ-ALLOC-009, REQ-ALLOC-011.

### T6: Implement fallback logic

- **Description:** Wrap coordination calls with fallback logic. When the database is unreachable (connection refused, timeout, auth failure), fall back to local allocation for that single attempt, emit a warning, and succeed. The next allocation attempt must try the database again (not remain in permanent fallback). The fallback must use the same local scan-and-increment logic as single-user mode. The database's idempotency check in `allocate_id` handles recovery when connectivity resumes.
- **Deliverable:** Fallback wrapper in `internal/coordination/` or at the entity creation call site. Warning emission via the existing logging mechanism.
- **Depends on:** T4, T5 (wraps the allocation calls those tasks introduce).
- **Effort:** Small (2-3 story points).
- **Spec requirements:** REQ-FAIL-001, REQ-FAIL-002, REQ-FAIL-003, REQ-FAIL-004.

### T7: Integration tests

- **Description:** Write tests covering the full coordination flow end-to-end. Use a real PostgreSQL instance (or testcontainer). Tests must cover: single-user mode unchanged, team mode allocates via database, idempotency (same slug returns same ID), concurrent allocation atomicity, feature display ID sequencing, fallback on unreachable database, recovery after fallback, env var substitution, schema migration idempotency, and pool lifecycle.
- **Deliverable:** `internal/coordination/db_test.go` with integration tests. Test helper that starts a Postgres testcontainer or connects to a `TEST_DATABASE_URL`.
- **Depends on:** T3, T4, T5, T6 (tests the integrated system).
- **Effort:** Large (5-8 story points).
- **Spec requirements:** All REQ-* and AC-* items in the specification.

## Dependency Graph

```
T1 (config + env var substitution)
T2 (pgx pool)          → depends on T1
T3 (schema + allocate) → depends on T2
T4 (entity wiring)     → depends on T1, T3
T5 (feature wiring)    → depends on T1, T3
T6 (fallback logic)    → depends on T4, T5
T7 (integration tests) → depends on T3, T4, T5, T6
```

**Parallel groups:** [T4, T5] can execute in parallel once T1 and T3 are done. **Critical path:** T1 → T2 → T3 → T4 → T6 → T7.

## Risk Assessment

### Risk: pgx pool integration with existing server lifecycle
- **Probability:** Medium. Kanbanzai currently has no database connections; integrating a connection pool into the server startup/shutdown lifecycle is new ground.
- **Impact:** Medium. If the pool lifecycle is wrong (e.g., connections leak), it affects server stability but not correctness of single-user mode.
- **Mitigation:** Keep the pool optional — only create it when `database_url` is configured. Ensure `Close()` is called on server shutdown. The pool creation failure must not prevent server startup (REQ-NF-005).
- **Affected tasks:** T2, T7.

### Risk: Concurrent allocation correctness under real Postgres load
- **Concept:** The `allocate_id` function uses `INSERT ... ON CONFLICT DO UPDATE` which is atomic in Postgres, but the idempotency check and counter increment are in the same transaction — test that the interaction is correct under concurrent callers.
- **Probability:** Low. The SQL pattern is well-understood and Postgres's MVCC guarantees are strong.
- **Impact:** High. If concurrent allocations produce duplicates, the whole point of the coordination database is undermined.
- **Mitigation:** Write a specific concurrent test (T7) that fires many goroutines at `allocate_id` and asserts no duplicates. If a race is found, add `SELECT ... FOR UPDATE` to the function.
- **Affected tasks:** T3, T7.

### Risk: Test dependency on a running Postgres instance
- **Probability:** High. CI environments and developer machines may not have Postgres available.
- **Impact:** Medium. Tests that require Postgres can't run in all environments, reducing test coverage.
- **Mitigation:** Use `testcontainers-go` to spin up a Postgres container for integration tests. Add a build tag or environment variable (`KANBANZAI_INTEGRATION_TESTS=1`) to allow skipping Postgres tests when unavailable.
- **Affected tasks:** T7.

### Risk: Environment variable substitution interacts poorly with existing config parsing
- **Probability:** Low. The substitution is a simple `os.ExpandEnv` pass over string values after YAML unmarshalling.
- **Impact:** Low. If a config value legitimately contains `${...}` that isn't an env var reference, it will be silently replaced with an empty string.
- **Mitigation:** Document that literal `${...}` in config values must be escaped as `$${...}`. This is consistent with `os.ExpandEnv` behaviour.
- **Affected tasks:** T1.

## Verification Approach

Every acceptance criterion in the specification maps to a verification method in T7 (integration tests). The mapping:

| Acceptance criteria group | Verified by | Method |
|---|---|---|
| AC-CFG-01 through AC-CFG-04 (config) | T7 | Automated tests: single-user mode, team mode, env var substitution, missing env var |
| AC-SCH-01, AC-SCH-02 (schema) | T7 | Automated tests: migration on empty DB, idempotent re-run |
| AC-ALLOC-01 through AC-ALLOC-04 (allocation) | T7 | Automated tests: concurrent allocation, idempotency, bug ID format, feature display ID |
| AC-ALLOC-05 (single-user) | T7 | Automated test: zero-config batch creation unchanged |
| AC-FAIL-01 through AC-FAIL-03 (failure) | T7 | Automated tests: fallback, reconnection, counter fast-forward |
| AC-CONN-01 through AC-CONN-03 (connection) | T7 | Automated tests: pool lifecycle, TLS connection, graceful startup with unreachable DB |

Additionally, T2 should include a unit test for pool creation with an invalid URL (confirming it doesn't panic), and T1 should include unit tests for env var substitution edge cases.

## Interface Contracts

### T1 → T2: CoordinationConfig
- **Provided by T1:** `config.Coordination` struct with `DatabaseURL string` and `ProjectID string` fields. `config.CoordinationEnabled() bool` method. All string config values have `${ENV_VAR}` expanded at load time.
- **Consumed by T2:** `DatabaseURL` is passed to `coordination.New()`.

### T2 → T3: coordination.DB pool
- **Provided by T2:** `coordination.DB` struct wrapping `*pgxpool.Pool`. Constructor `New(ctx, databaseURL) (*DB, error)`. `Close()` and `Ping(ctx) error` methods. Pool is nil-safe (callers check `db != nil`).
- **Consumed by T3:** `Migrate(ctx)` and `AllocateID(ctx, projectID, entityType, prefix, slug)` are methods added to `*DB`.

### T3 → T4, T5: AllocateID and feature sequence methods
- **Provided by T3:** `(*DB).AllocateID(ctx, projectID, entityType, prefix, slug) (string, error)`. `(*DB).AllocateFeatureSeq(ctx, projectID, batchID) (int, error)`.
- **Consumed by T4:** `AllocateID` for plan, batch, and bug entity creation.
- **Consumed by T5:** `AllocateFeatureSeq` for feature display ID allocation.

### T4, T5 → T6: Allocation call sites
- **Provided by T4, T5:** Entity creation code paths call `coordinationDB.AllocateID(...)` or `coordinationDB.AllocateFeatureSeq(...)` gated on `config.CoordinationEnabled()`.
- **Consumed by T6:** Fallback wrapper intercepts these calls, catches connectivity errors, and delegates to local allocation.

## Traceability Matrix

| Spec requirement | Task(s) |
|---|---|
| REQ-CFG-001, REQ-CFG-002, REQ-CFG-003, REQ-CFG-004 | T1 |
| REQ-CONN-001, REQ-CONN-002, REQ-CONN-003, REQ-NF-005 | T2 |
| REQ-SCH-001, REQ-SCH-002, REQ-ALLOC-001 through REQ-ALLOC-004 | T3 |
| REQ-ALLOC-001, REQ-ALLOC-002, REQ-ALLOC-005, REQ-ALLOC-006, REQ-ALLOC-010 | T4 |
| REQ-ALLOC-007, REQ-ALLOC-008, REQ-ALLOC-009, REQ-ALLOC-011 | T5 |
| REQ-FAIL-001, REQ-FAIL-002, REQ-FAIL-003, REQ-FAIL-004 | T6 |
| REQ-NF-001, REQ-NF-002, REQ-NF-003, REQ-NF-004 (non-functional verification) | T7 |
| All acceptance criteria (AC-CFG-* through AC-CONN-*) | T7 |

## Dependencies

- `work/P48-coordination-server/P48-spec-coordination-database.md` — parent specification.
- `github.com/jackc/pgx/v5` — Go Postgres driver (new dependency, added in T2).
- `github.com/testcontainers/testcontainers-go` — optional test dependency for T7 (Postgres testcontainer).
