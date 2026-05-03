| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03T20:05:20Z           |
| Status | Draft                          |
| Author | Sambeau                        |

# Coordination Database — Specification

## Problem Statement

This specification covers the PostgreSQL-backed coordination database for Kanbanzai team deployments. It implements the design described in `work/P48-coordination-server/P48-design-coordination-server.md` (P48-coordination-server/design-p48-design-coordination-server).

The coordination database provides centralized, collision-free ID allocation for plans, batches, features, and bugs when multiple Kanbanzai instances share a project. Single-user projects without a configured database continue to use local file-scanned allocation with no change in behaviour.

This specification implements the following design decisions from the parent design:

- **D1**: A coordination database as a lightweight intermediary, not a full state server.
- **D2**: Single-user mode is the default; team mode is opt-in via `coordination.database_url`.
- **D3**: The database stores counters and allocation registries, not entity state.
- **D4**: Bugs get project-scoped sequential IDs (`BUG-1`, `BUG-2`, ...).
- **D5**: Feature canonical IDs remain TSID-based; only display IDs are database-allocated.
- **D7**: PostgreSQL is the sole coordination backend.

**In scope:**

- The database schema (counters, batch_feature_seqs, allocations tables).
- The `allocate_id` SQL function for atomic, idempotent ID allocation.
- Kanbanzai's Go-side coordination layer: configuration parsing, connection management, and calling `allocate_id`.
- Fallback behaviour when the database is unreachable.
- Environment variable substitution for `database_url` in `.kbz/config.yaml`.

**Out of scope:**

- The additional coordination functions (knowledge entries, checkpoints, worktree tracking, merge locks, entity status cache). These are deferred per the design.
- The full centralized state server.
- Migration of existing entities from TSID bug IDs to sequential bug IDs.
- A standalone coordination binary. PostgreSQL is the sole backend.
- Task IDs (remain TSID-based) and document IDs (remain path-derived).

## Requirements

### Functional Requirements

#### Configuration

- **REQ-CFG-001:** Kanbanzai MUST accept a `coordination` section in `.kbz/config.yaml` with keys `database_url` and `project_id`.
- **REQ-CFG-002:** When `coordination.database_url` is absent or empty, Kanbanzai MUST operate in single-user mode: ID allocation scans local `.kbz/state/` directories and increments, with no database connection attempted.
- **REQ-CFG-003:** When `coordination.database_url` is present and non-empty, Kanbanzai MUST operate in team mode: ID allocation calls the coordination database.
- **REQ-CFG-004:** String values in `.kbz/config.yaml` containing `${ENV_VAR}` MUST be replaced with the value of the named environment variable at load time. If the environment variable is unset, Kanbanzai MUST report a clear error identifying the variable name.

#### Schema migration

- **REQ-SCH-001:** On first connection to the coordination database, Kanbanzai MUST ensure the coordination tables and the `allocate_id` function exist, using `CREATE TABLE IF NOT EXISTS` and `CREATE OR REPLACE FUNCTION` statements.
- **REQ-SCH-002:** The schema MUST match the design document exactly: `counters` table with `(project_id, entity_type)` primary key and `next_value` column; `batch_feature_seqs` table with `(project_id, batch_id)` primary key and `next_seq` column; `allocations` table with `(project_id, entity_type, slug)` primary key and `allocated_id`, `allocated_at` columns.

#### ID allocation — plan and batch

- **REQ-ALLOC-001:** When creating a plan in team mode, Kanbanzai MUST call `allocate_id(project_id, 'plan_{prefix}', '{prefix}', '{slug}')` and use the returned ID as the plan's canonical ID.
- **REQ-ALLOC-002:** When creating a batch in team mode, Kanbanzai MUST call `allocate_id(project_id, 'batch_{prefix}', '{prefix}', '{slug}')` and use the returned ID as the batch's canonical ID.
- **REQ-ALLOC-003:** The `allocate_id` function MUST allocate IDs atomically: two concurrent calls for the same `(project_id, entity_type)` MUST receive distinct, monotonically increasing numeric components.
- **REQ-ALLOC-004:** The `allocate_id` function MUST be idempotent: calling it twice with the same `(project_id, entity_type, slug)` MUST return the same ID both times. The counter MUST NOT be incremented on the second call.

#### ID allocation — bugs

- **REQ-ALLOC-005:** When creating a bug in team mode, Kanbanzai MUST call `allocate_id(project_id, 'bug', 'BUG', '{slug}')` and use the returned ID as the bug's display ID.
- **REQ-ALLOC-006:** Bug IDs allocated by the database MUST follow the format `BUG-{n}-{slug}` where `n` is a project-scoped sequential integer starting at 1.

#### ID allocation — feature display IDs

- **REQ-ALLOC-007:** When creating a feature in team mode, the feature's canonical ID MUST remain TSID-based (no change from current behaviour).
- **REQ-ALLOC-008:** When creating a feature in team mode, Kanbanzai MUST atomically increment `next_seq` in `batch_feature_seqs` for the parent batch's `(project_id, batch_id)` and return the incremented value as the feature's display sequence number.
- **REQ-ALLOC-009:** The feature display ID allocation MUST prevent two concurrent feature creations in the same batch from receiving the same display sequence number.

#### ID allocation — single-user mode

- **REQ-ALLOC-010:** In single-user mode, plan, batch, and bug ID allocation MUST behave identically to the current Kanbanzai behaviour: scan local `.kbz/state/` directories, find the highest existing number, and increment.
- **REQ-ALLOC-011:** In single-user mode, feature display ID allocation MUST behave identically to the current Kanbanzai behaviour: use the batch's local `next_feature_seq` counter.

#### Failure modes

- **REQ-FAIL-001:** When the coordination database is unreachable (connection refused, timeout, or authentication failure), Kanbanzai MUST fall back to local allocation for that single allocation attempt and MUST emit a warning to the user.
- **REQ-FAIL-002:** The fallback allocation in REQ-FAIL-001 MUST use the same local scan-and-increment logic as single-user mode.
- **REQ-FAIL-003:** After a successful fallback allocation, Kanbanzai MUST continue attempting to use the database for subsequent allocations (i.e., fallback is per-attempt, not session-wide).
- **REQ-FAIL-004:** When the database becomes reachable after a period of fallback allocations, the `allocate_id` function's idempotency check MUST prevent collisions: if a locally-allocated ID matches a slug already in the `allocations` table, the database returns the existing allocation. If the local counter has advanced past the database counter, the counter fast-forwards naturally on the next successful allocation.

#### Connection management

- **REQ-CONN-001:** Kanbanzai MUST use `jackc/pgx/v5` with `pgxpool` for database connections.
- **REQ-CONN-002:** The connection pool MUST be created once at startup (when `database_url` is configured) and reused for all coordination operations during the session.
- **REQ-CONN-003:** The connection pool MUST use TLS for all connections (pgx's default behaviour).

### Non-Functional Requirements

- **REQ-NF-001:** An ID allocation in team mode (excluding network latency) MUST complete within 100ms under normal database load.
- **REQ-NF-002:** The coordination database schema MUST use only standard SQL features available in PostgreSQL 14 and later. No extensions (beyond what pgx provides) are required.
- **REQ-NF-003:** The Go coordination layer MUST add no more than one new dependency (`jackc/pgx/v5`). No other third-party libraries are required for coordination.
- **REQ-NF-004:** Single-user mode MUST have zero performance overhead compared to current Kanbanzai behaviour — no database connection is attempted, no additional file I/O is performed.
- **REQ-NF-005:** A database connection failure during startup (pool creation) MUST NOT prevent Kanbanzai from starting. The failure MUST be reported as a warning, and Kanbanzai MUST operate in fallback mode for all allocations until the connection succeeds.

## Constraints

- This specification does NOT cover the additional coordination functions listed in the design (knowledge entries, checkpoints, worktree tracking, merge locks, entity status cache). These are deferred.
- Entity state files in `.kbz/state/` MUST remain the canonical source of entity data. The coordination database stores only counters and allocation registries.
- Existing Kanbanzai behaviour in single-user mode MUST NOT change. This is a non-negotiable backward-compatibility constraint.
- The `project_id` in the coordination config MUST be a stable identifier. Changing it after initial setup orphans existing allocation records (though the counters naturally fast-forward).
- Task IDs (`TASK-{TSID13}`) and document IDs remain unchanged by this specification.
- The `allocate_id` function MUST NOT be declared `SECURITY DEFINER` in the initial implementation. Teams that need scoped database users can add this later (see design §Future: scoped database users).

## Acceptance Criteria

### Configuration and mode selection

- **AC-CFG-01 (REQ-CFG-001, REQ-CFG-002):** Given a `.kbz/config.yaml` with no `coordination` section, when Kanbanzai allocates a batch ID, then it scans local `.kbz/state/batches/` and no database connection is attempted.
- **AC-CFG-02 (REQ-CFG-001, REQ-CFG-003):** Given a `.kbz/config.yaml` with `coordination.database_url` set to a valid Postgres URI and `coordination.project_id` set to `"test-project"`, when Kanbanzai allocates a batch ID, then it calls `allocate_id('test-project', 'batch_B', 'B', '<slug>')` on the database.
- **AC-CFG-03 (REQ-CFG-004):** Given a config value `"${TEST_DB_URL}"` and an environment variable `TEST_DB_URL=postgres://localhost/test`, when the config is loaded, then the value resolves to `postgres://localhost/test`.
- **AC-CFG-04 (REQ-CFG-004):** Given a config value `"${NONEXISTENT_VAR}"` and no such environment variable set, when the config is loaded, then Kanbanzai reports an error naming `NONEXISTENT_VAR`.

### Schema migration

- **AC-SCH-01 (REQ-SCH-001, REQ-SCH-002):** Given an empty coordination database with no coordination tables, when Kanbanzai connects for the first allocation, then the `counters`, `batch_feature_seqs`, and `allocations` tables exist, and the `allocate_id` function is defined.
- **AC-SCH-02 (REQ-SCH-001):** Given a coordination database that already has the coordination tables from a previous run, when Kanbanzai connects, then the `CREATE TABLE IF NOT EXISTS` statements succeed without error (idempotent).

### ID allocation — atomicity and idempotency

- **AC-ALLOC-01 (REQ-ALLOC-001, REQ-ALLOC-003):** Given a coordination database with `counters` showing `next_value = 5` for `('test-project', 'batch_B')`, when two concurrent batch creations are made with different slugs, then one receives `B5-slug-a` and the other receives `B6-slug-b` (or vice versa), and the counter advances to 7.
- **AC-ALLOC-02 (REQ-ALLOC-004):** Given a batch with slug `"auth-system"` has already been allocated as `B5-auth-system`, when `allocate_id('test-project', 'batch_B', 'B', 'auth-system')` is called again, then it returns `B5-auth-system` and the `counters.next_value` is unchanged.
- **AC-ALLOC-03 (REQ-ALLOC-005, REQ-ALLOC-006):** Given a coordination database with no prior bug allocations, when a bug with slug `"login-failure"` is created, then its ID is `BUG-1-login-failure`.
- **AC-ALLOC-04 (REQ-ALLOC-007, REQ-ALLOC-008):** Given a batch `B12-payments` with `next_seq = 3` in the database, when a feature is created in that batch, then its canonical ID is a TSID (e.g., `FEAT-01KMRX1SEQV49-add-paypal`) and its display ID is `B12-F3`.

### Single-user mode

- **AC-ALLOC-05 (REQ-ALLOC-010):** Given a project with no `coordination.database_url` and existing state files for batches B1 through B4, when a new batch with slug `"new-batch"` is created, then its ID is `B5-new-batch` and no database connection is attempted.

### Failure modes

- **AC-FAIL-01 (REQ-FAIL-001, REQ-FAIL-002):** Given a configured `database_url` pointing to an unreachable host, when a batch is created, then Kanbanzai falls back to local allocation, uses the local counter, emits a warning, and the batch is created successfully.
- **AC-FAIL-02 (REQ-FAIL-003):** Given the database was unreachable for one allocation (causing a fallback), when the database becomes reachable again, then the next allocation uses the database (does not remain in permanent fallback mode).
- **AC-FAIL-03 (REQ-FAIL-004):** Given a local fallback allocated `B10-some-slug` while the database counter was at 9, when the database becomes reachable and a new batch `"other-slug"` is created, then the database allocates `B11-other-slug` (counter fast-forwards past 10) or returns the existing allocation for `"some-slug"` if re-requested.

### Connection management

- **AC-CONN-01 (REQ-CONN-001, REQ-CONN-002):** Given a configured `database_url`, when Kanbanzai starts, then a `pgxpool` is created and reused for all subsequent allocations in the session.
- **AC-CONN-02 (REQ-CONN-003):** Given a Supabase `database_url` (which requires TLS), when Kanbanzai connects, then the TLS handshake succeeds without additional configuration.
- **AC-CONN-03 (REQ-NF-005):** Given a configured `database_url` pointing to an unreachable host, when Kanbanzai starts, then it starts successfully, emits a warning about the unavailable database, and operates in fallback mode.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-CFG-01 | Test | Automated test: start Kanbanzai with no `coordination` config, create a batch, assert no database connection was opened and local state was scanned |
| AC-CFG-02 | Test | Automated test: start Kanbanzai with a test database, create a batch, assert `allocate_id` was called with correct arguments |
| AC-CFG-03 | Test | Automated test: load config with `${TEST_DB_URL}`, set env var, assert resolved value |
| AC-CFG-04 | Test | Automated test: load config with `${NONEXISTENT_VAR}`, assert error message contains variable name |
| AC-SCH-01 | Test | Automated test: point at empty test database, trigger first allocation, assert tables and function exist |
| AC-SCH-02 | Test | Automated test: run schema migration twice against same database, assert no errors |
| AC-ALLOC-01 | Test | Automated test: two concurrent goroutines call `allocate_id`, assert distinct IDs and correct counter |
| AC-ALLOC-02 | Test | Automated test: call `allocate_id` twice with same arguments, assert same ID returned and counter unchanged |
| AC-ALLOC-03 | Test | Automated test: create bug, assert ID format `BUG-1-{slug}` |
| AC-ALLOC-04 | Test | Automated test: create feature in batch with known `next_seq`, assert canonical ID is TSID and display ID uses the sequence |
| AC-ALLOC-05 | Test | Automated test: create batch in single-user mode, assert local scan behaviour and no DB connection |
| AC-FAIL-01 | Test | Automated test: configure unreachable `database_url`, create batch, assert fallback allocation succeeds with warning |
| AC-FAIL-02 | Test | Automated test: simulate one failed connection then successful reconnection, assert subsequent allocation uses database |
| AC-FAIL-03 | Test | Automated test: simulate fallback allocation, then reconnect and create new entity, assert counter fast-forwards correctly |
| AC-CONN-01 | Test | Automated test: inspect connection pool after startup, assert pool is created and reused |
| AC-CONN-02 | Test | Test against a real Supabase project (or Postgres instance with `sslmode=require`), assert TLS connection succeeds |
| AC-CONN-03 | Test | Automated test: start Kanbanzai with unreachable `database_url`, assert process starts and warning is emitted |

## Dependencies

- `work/P48-coordination-server/P48-design-coordination-server.md` — parent design document.
- `github.com/jackc/pgx/v5` — Go Postgres driver (new dependency).
- `work/_project/design-centralized-state-server.md` — broader architecture this extends.
- `work/B38-plans-and-batches/B38-design-meta-planning-plans-and-batches.md` — Plan/Batch ID system.
