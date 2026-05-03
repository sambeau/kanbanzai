| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03T19:34:00Z           |
| Status | approved |
| Author | Sambeau                        |

# Coordination Server — Design

## Related Work

### Prior documents consulted

- `work/_project/design-centralized-state-server.md` (2026-04-22, draft) — the full centralized state server design. This document defines the architectural framing that this design extends: a shared PostgreSQL-backed server per team/project, introduced as an alternative backend rather than a replacement. Its decisions — one canonical backend per project, service-layer decoupling as the first implementation step, Git-native remaining first-class — are directly inherited here.
- `work/_project/design-transition-history-storage.md` (2026-04-22) — the competing Git-native evolution path. Its Alternative 4 explicitly defers centralized state to the broader design above.
- `work/_project/research-storage-architecture-and-progress-visibility.md` (2026-04-24) — extends the state-backend comparison, confirms the same conclusion.
- `work/B38-plans-and-batches/B38-design-meta-planning-plans-and-batches.md` — source of the current Plan/Batch ID system, Decisions D4 (B prefix), D7 (separate prefix registries), D8 (independent counters).

### Decisions that constrain this design

1. From `design-centralized-state-server.md`:
   - "A centralized state server is a valid strategic direction for Kanbanzai, but it should be introduced as an alternative backend, not as an immediate replacement."
   - "One project must have exactly one canonical backend at a time."
   - "The first implementation step is service-layer decoupling from file-backed persistence."
   - "Git-native mode should remain a first-class supported option."
2. From `design-meta-planning-plans-and-batches.md`:
   - P prefix for plans, B prefix for batches.
   - Independent sequence counters for plans and batches.
   - Separate prefix registries.

### How this design extends prior work

The centralized-state-server design is a full architectural proposal covering database schemas, migration strategy, failure modes, and operational models. It is the right long-term direction. This design proposes a narrower, earlier step: a **coordination server** that handles only ID allocation and a small set of additional coordination functions. The coordination server is deliberately not a full centralized state server — it is a stepping stone that solves the immediate collision problem while paving the road for the broader centralized mode.

Crucially, the full centralized-state-server design requires service-layer decoupling as a prerequisite. The coordination server proposed here has a much narrower surface area and can be introduced without refactoring the entire persistence layer. It extends, rather than replaces, the current file-backed store.

## Overview

A lightweight coordination layer backed by a remote PostgreSQL database. It provides centralized ID allocation for plans, batches, features, and bugs in team deployments. Single-user projects continue to use local file-scanned allocation with zero configuration. Team projects point at a shared PostgreSQL database and get collision-free IDs. The database also handles a small set of additional coordination functions that benefit from a single authority: feature sequence counters, knowledge sharing, checkpoint visibility, worktree tracking, merge coordination, and entity status caching.

There is only one backend: **PostgreSQL**. There is no standalone binary, no file-backed alternative, no intermediate service. Kanbanzai connects directly to Postgres using a connection string in its project config. This keeps the design simple, eliminates a whole class of operational concerns (deploying and maintaining a separate coordination binary), and aligns with the centralized-state-server design's choice of PostgreSQL as the reference backend. Teams that don't want to run Postgres themselves can use a managed service — Supabase is a natural fit.

## Goals and Non-Goals

### Goals

- Eliminate ID collisions for Plans (`P{n}`), Batches (`B{n}`), Feature display IDs (`B{n}-F{m}`), and Bugs (`BUG-{n}`) in multi-user deployments.
- Require zero configuration for single-user projects — they continue unchanged.
- Use PostgreSQL as the sole coordination backend. No binary, no file-backed mode.
- Keep the schema minimal — a handful of tables for counters and coordination state.
- Serve as a stepping stone toward the full centralized state server design.
- Handle feature `next_feature_seq` counters to prevent two servers from allocating the same `B24-F3`.

### Non-Goals

- Replace the Git-native file store as canonical workflow state.
- Store or serve entity state beyond what is needed for ID allocation and coordination.
- Handle task IDs — tasks use TSID13 and have effectively zero collision risk.
- Handle document IDs — documents use owner/slug-derived IDs that are naturally namespaced.
- Provide a full PostgreSQL-backed centralized state database (that is the broader design, deferred).
- Support the centralized state server's broader query, analytics, or real-time state sharing goals.
- Provide a standalone coordination binary. PostgreSQL is the only backend.

## Problem and Motivation

Kanbanzai is Git-native: entity state lives in `.kbz/state/` YAML files, and Git is the transport for collaboration. For a single developer, this works well — there is one source of truth, one sequence counter, no collisions.

In a team, each developer runs their own Kanbanzai server (as an MCP server in their editor). When two developers create batches, features, or bugs, the local ID allocation scans the local `.kbz/state/` directory and increments. If neither has pushed yet, both allocate the same number. The result is a merge conflict or, worse, two different entities with the same ID committed to the repository.

The entities affected and their collision risk:

| Entity | ID format | Allocation strategy | Collision risk |
|--------|-----------|---------------------|----------------|
| Plan | `P{n}-slug` | Sequential scan of `plans/` | Low (teams coordinate plans) |
| Batch | `B{n}-slug` | Sequential scan of `batches/` | **High** |
| Feature display ID | `B{n}-F{m}` | `next_feature_seq` on batch | **High** |
| Bug | `BUG-{TSID13}` | Random TSID | Low per-ID, but TSIDs are not human-discussable |
| Task | `TASK-{TSID13}` | Random TSID | Negligible |
| Document | `{owner}/{type}-{slug}` | Path-derived | Negligible (same doc = same path) |

The Plan/Batch distinction (`B38`) made the problem worse: batches are created more frequently than plans were, and the `B` prefix carries no team-level namespace.

The full centralized state server design solves this comprehensively but requires significant architectural change. This design proposes a minimal, focused solution: a coordination database that owns only ID allocation and a small set of tightly-scoped coordination functions.

## Design

### Single-user mode (no change)

When no coordination database is configured, Kanbanzai behaves exactly as it does today. ID allocation scans local `.kbz/state/` directories and increments. No new configuration, no new infrastructure, no change in behaviour.

This is the default. A single-user project never needs to know the coordination database exists.

### Team mode (PostgreSQL)

A team needs a single source of truth for ID allocation. The coordination backend is a PostgreSQL database. Kanbanzai connects directly to Postgres and uses SQL queries for all coordination functions. PostgreSQL provides everything the coordination layer needs out of the box: atomic sequences for ID allocation, transactional counters for feature sequences, row-level locking for merge gating, and mature authentication.

Managed Postgres services (Supabase, Neon, Render, AWS RDS, etc.) make this accessible to teams that don't want to run their own database. The coordination tables are ordinary Postgres tables — no extensions, no stored procedures beyond simple SQL functions, no superuser privileges needed.

#### Configuration

In `.kbz/config.yaml`:

```yaml
coordination:
  database_url: "postgres://user:pass@host:5432/kanbanzai_coord"
  project_id: "my-project"
```

When `database_url` is absent or empty, Kanbanzai operates in single-user mode. When present, it operates in team mode. The `project_id` scopes state to a specific project — one database can serve multiple projects without ID collision.

The `project_id` should be a stable, human-readable slug for the project. It appears as a column in every table, so changing it after initial setup would orphan existing coordination state. Teams that need to rename a project should create a new `project_id` and accept that old IDs may not be recognized as allocated (the idempotency check will miss them, but the counter will have advanced past them regardless).

#### Database schema

The coordination state is expressed as three simple tables:

```sql
-- Per-project, per-entity-type sequential counters.
-- Plan counters are per-prefix (e.g. 'plan_P', 'plan_X').
-- Batch counters are per-prefix (e.g. 'batch_B').
-- Bug counter is 'bug'.
CREATE TABLE counters (
    project_id  TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    next_value  INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, entity_type)
);

-- Per-batch feature display ID sequences.
-- batch_id is the full canonical batch ID (e.g. 'B24-auth-system').
CREATE TABLE batch_feature_seqs (
    project_id TEXT NOT NULL,
    batch_id   TEXT NOT NULL,
    next_seq   INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (project_id, batch_id)
);

-- Allocation registry for idempotency.
-- Re-requesting the same project + entity_type + slug returns the same ID.
CREATE TABLE allocations (
    project_id   TEXT NOT NULL,
    entity_type  TEXT NOT NULL,
    slug         TEXT NOT NULL,
    allocated_id TEXT NOT NULL,
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entity_type, slug)
);
```

ID allocation is a single SQL function — all the logic is in the database, no application code needed:

```sql
CREATE OR REPLACE FUNCTION allocate_id(
    p_project_id  TEXT,
    p_entity_type TEXT,
    p_prefix      TEXT,
    p_slug        TEXT
) RETURNS TEXT AS $$
DECLARE
    existing TEXT;
    next_val INTEGER;
    result_id TEXT;
BEGIN
    -- Idempotency: return existing allocation for same slug.
    SELECT allocated_id INTO existing
    FROM allocations
    WHERE project_id = p_project_id
      AND entity_type = p_entity_type
      AND slug = p_slug;
    IF existing IS NOT NULL THEN
        RETURN existing;
    END IF;

    -- Atomically increment the counter.
    INSERT INTO counters (project_id, entity_type, next_value)
    VALUES (p_project_id, p_entity_type, 2)
    ON CONFLICT (project_id, entity_type)
    DO UPDATE SET next_value = counters.next_value + 1
    RETURNING next_value - 1 INTO next_val;

    -- Record the allocation.
    result_id := p_prefix || next_val || '-' || p_slug;
    INSERT INTO allocations (project_id, entity_type, slug, allocated_id)
    VALUES (p_project_id, p_entity_type, p_slug, result_id);

    RETURN result_id;
END;
$$ LANGUAGE plpgsql;
```

Feature display ID allocation and bug ID allocation follow the same pattern, targeting `batch_feature_seqs` or `counters` with `entity_type = 'bug'` respectively.

The schema is intentionally minimal. It stores IDs, sequence counters, and coordination metadata — not entity names, statuses, summaries, or any other workflow state. This is coordination state, not canonical entity state.

#### ID allocation flow (team mode)

1. User creates a batch via `entity(action: "create", type: "batch", ...)`.
2. The MCP tool checks for `coordination.database_url` in config.
3. If present: opens a connection, calls `SELECT allocate_id(...)`, returns the full ID.
4. The MCP tool writes the entity YAML file with the server-allocated ID.

Bugs are similar but project-scoped (no batch prefix needed).

Features are different: their canonical ID remains TSID-based (no change), but their display ID (`B{n}-F{m}`) is allocated by atomically incrementing the batch's `next_feature_seq` counter in the database.

### Additional coordination functions

The coordination database's narrow charter is ID allocation, but several additional functions naturally fit the same pattern (single authority, low data volume, immediate value in team settings). These are listed here for completeness — the initial implementation focuses on ID allocation and feature sequence counters. The remaining functions are deferred until the database proves its value.

Each function is expressed as a SQL table. No extra processes, no HTTP endpoints, no binary to deploy — just tables in the same Postgres database.

#### 1. Feature sequence counters

Already covered above — the `batch_feature_seqs` table owns `next_feature_seq` per batch. Two servers creating features in the same batch will not produce the same display ID. This is part of the initial implementation.

#### 2. Knowledge entries — cross-instance distribution

Knowledge entries are currently contributed locally, written to `.kbz/state/knowledge/`, and merged via Git. A shared table would make knowledge immediately visible to all instances — no push/pull delay. This is especially valuable for operationally critical knowledge like "we just learned that function X has a concurrency bug" — you want that visible to other agents immediately.

```sql
CREATE TABLE knowledge_entries (
    project_id TEXT NOT NULL,
    entry_id   TEXT NOT NULL DEFAULT ('KE-' || gen_random_uuid()::text),
    topic      TEXT NOT NULL,
    content    TEXT NOT NULL,
    tags       TEXT[] NOT NULL DEFAULT '{}',
    tier       INTEGER NOT NULL DEFAULT 3,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entry_id)
);
CREATE INDEX idx_knowledge_since ON knowledge_entries (project_id, created_at);
```

Knowledge entries would continue to be written to `.kbz/state/knowledge/` files for Git history, but the database table provides the real-time distribution channel. The database is not canonical for knowledge — files are.

#### 3. Checkpoints

Checkpoints (`checkpoint` tool) block work until a human responds. Currently, a checkpoint created by one server instance isn't visible to another until pushed. A shared table would make checkpoints globally visible and prevent two servers from both waiting on the same human decision independently.

```sql
CREATE TABLE checkpoints (
    project_id    TEXT NOT NULL,
    checkpoint_id TEXT NOT NULL DEFAULT ('CHK-' || gen_random_uuid()::text),
    question      TEXT NOT NULL,
    context       TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending | responded
    response      TEXT,
    created_by    TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at  TIMESTAMPTZ,
    PRIMARY KEY (project_id, checkpoint_id)
);
```

#### 4. Worktree and branch tracking

Worktree records are per-instance by nature (a worktree is a local directory), but knowing which worktrees exist across the team avoids double-booking. Currently, two people can create worktrees for the same feature on different machines without knowing.

```sql
CREATE TABLE worktrees (
    project_id TEXT NOT NULL,
    entity_id  TEXT NOT NULL,
    branch     TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entity_id, created_by)
);
```

#### 5. Merge coordination

The `merge` tool checks gates locally. In a team, two people might both think they're clear to merge. A merge lock table prevents races (similar to a CI merge queue):

```sql
CREATE TABLE merge_locks (
    project_id  TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    acquired_by TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entity_id)
);
```

Acquiring a lock is `INSERT ... ON CONFLICT DO NOTHING` — if a row already exists, someone else holds the lock. Locks auto-expire: a periodic cleanup query removes rows older than N minutes (`DELETE FROM merge_locks WHERE acquired_at < now() - interval '5 minutes'`). This is optional — if the database is unavailable, the merge proceeds without the lock (with a warning).

#### 6. Entity status visibility

Features moving through `developing → reviewing → done` — if two agents are working on different tasks of the same feature, they both need current feature status. Currently this lags behind Git push/pull cycles. An eventually-consistent cache table, updated when clients notify it of transitions:

```sql
CREATE TABLE entity_status (
    project_id TEXT NOT NULL,
    entity_id  TEXT NOT NULL,
    status     TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, entity_id)
);
```

Clients call `INSERT ... ON CONFLICT UPDATE` to notify of a status change, and `SELECT` to query current known status. This is a cache — files in `.kbz/state/` remain canonical.

#### What stays in the repository

The dividing line:

| In the coordination database | In the repository |
|---|---|
| IDs, sequence counters | Source code |
| Knowledge entries (distribution) | Documents (specs, designs) |
| Checkpoints | Configuration |
| Worktree inventory | Skills, roles, stage bindings |
| Merge queue state | Entity state files (canonical) |
| Entity status cache | Knowledge entry files (canonical) |

The coordination database is a coordination layer — it makes the existing Git-native state work better for teams. It is not a replacement for that state.

### Deployment models

#### Supabase (managed Postgres)

[Supabase](https://supabase.com/docs) provides a managed Postgres database with connection pooling, SSL, and a generous free tier. This is the recommended path for teams that don't want to manage their own database.

**Setup**: Create a Supabase project, get the connection string from the dashboard (Settings → Database → Connection string), and add it to `.kbz/config.yaml`. That's it. No Supabase-specific features are required — the coordination tables are standard Postgres tables. The connection uses the standard `postgresql://` URI format.

**Connection pooling**: Supabase provides Supavisor (session mode and transaction mode) and PgBouncer (dedicated pooler, paid tiers). Kanbanzai's coordination connections are short-lived (allocate an ID and disconnect), so transaction mode pooling is appropriate. The connection string port determines the pooler: port `6543` for transaction mode, port `5432` for direct/session mode.

**SSL**: Supabase requires SSL connections. Most Go Postgres drivers (pgx, lib/pq) enable SSL by default when connecting to Supabase. The connection string should include `?sslmode=require` if the driver doesn't default to it.

**Auth**: See Authentication section below.

#### Self-managed Postgres

Teams can point at any PostgreSQL 14+ instance. The coordination tables use only standard SQL features available in all supported Postgres versions. No extensions are required beyond what a standard Postgres install provides.

#### Other managed services

Neon, Render, AWS RDS, Google Cloud SQL, and Azure Database for PostgreSQL all work identically — the coordination layer only needs a connection string. No provider-specific features are used.

### Authentication

Authentication is a first-class concern for any shared database. The approach depends on the deployment model, but the fundamental mechanism is the same: the Postgres connection string carries credentials, and Postgres role-based access control (RBAC) enforces permissions.

#### Connection-string authentication (all deployments)

The `database_url` in `.kbz/config.yaml` contains the credentials:

```
postgres://user:password@host:5432/kanbanzai_coord
```

This is the standard Postgres authentication model. The database authenticates the user via password (md5 or SCRAM-SHA-256). The coordination tables are owned by this database user, and all coordination operations run as this user.

**Important: `.kbz/config.yaml` is committed to Git.** It is project configuration, not machine-local state (unlike `local.yaml`, which is gitignored). This means the database password *will* be committed to the repository unless environment variable substitution is used. The design therefore treats env var substitution as a hard requirement for the coordination config — see below.

#### Supabase-specific auth

Supabase provides two authentication paths:

1. **Database password (recommended for coordination).** Use the database password from the Supabase dashboard (Settings → Database → Database password). The connection string is:

   ```
   postgresql://postgres:[YOUR-PASSWORD]@db.xxxxx.supabase.co:5432/postgres
   ```

   Or via Supavisor transaction mode:

   ```
   postgres://postgres:[YOUR-PASSWORD]@db.xxxxx.supabase.co:6543/postgres
   ```

   This uses the `postgres` role with full access to the `public` schema. The coordination tables live in the `public` schema by default.

2. **Supabase Auth with Row Level Security (not needed for coordination).** Supabase Auth provides JWT-based user authentication, social login, and RLS policies. These are designed for end-user applications, not for backend coordination between MCP servers. The coordination layer does not use Supabase Auth — it uses the database password directly. If a team later adopts the full centralized state server and wants per-user access control, Supabase Auth and RLS can be layered on top of the coordination tables.

#### Environment variable substitution (required)

Because `.kbz/config.yaml` is committed to Git, the database password must not appear in it directly. The config must support environment variable substitution:

```yaml
coordination:
  database_url: "${KANBANZAI_DATABASE_URL}"
  project_id: "my-project"
```

Any string value containing `${ENV_VAR}` is replaced with the value of `ENV_VAR` at load time. Each team member sets `KANBANZAI_DATABASE_URL` in their shell environment (`.zshrc`, `.bashrc`, or editor MCP config). This keeps the password out of version control while keeping the non-secret parts of the coordination config (like `project_id`) in the committed file where they belong.

This env var substitution should be implemented as a general config feature — it isn't coordination-specific. Other config fields that contain sensitive values (e.g., future API keys) will benefit from it too.

#### Security considerations

| Concern | Mitigation |
|---|---|
| Database password committed to repo | Prevented by env var substitution. The password lives in each developer's environment, never in the committed config. |
| All team members share the same database user | For the initial implementation, this is fine. The coordination database stores counters and allocations, not user-specific data. Teams can create per-user database roles later if needed. |
| Network exposure | Supabase and managed services provide SSL by default. Self-managed Postgres should be configured with SSL. |
| Accidental or malicious counter manipulation | The `allocate_id` function is the only write path for ID allocation. Teams can lock down the `counters` table to be writable only by the `allocate_id` function's owner (using `SECURITY DEFINER`) and grant `EXECUTE` on the function to application users. This is a future enhancement, not needed for the initial implementation. |

#### Future: scoped database users

For teams that want finer-grained access control, the coordination schema can be set up with a dedicated `kanbanzai_coord` role that only has `EXECUTE` on the `allocate_id` function and `SELECT`/`INSERT` on the coordination tables. The `allocate_id` function would be declared `SECURITY DEFINER` so it runs with the table owner's privileges. This is deferred — the initial implementation uses the database owner role directly.

### Failure modes

#### Database unavailable

If the coordination database is unreachable, Kanbanzai falls back to local allocation with a warning. The next time the database is reachable, the local counter may have advanced past the stored counter. The database detects this (the allocated ID already exists in its registry) and fast-forwards its counter.

This means brief outages do not block work, but they reintroduce the (small) window for collisions. Teams that want zero collision risk should ensure the database is available.

#### Network partition

If two users are partitioned from each other but both can reach the coordination database, there is no collision risk — the database serializes allocations. If a user is partitioned from the database, they fall back to local allocation (see above).

#### Database data loss

If the coordination tables are lost, recovery is a scan of the repository:

1. Scan `.kbz/state/` for all entity files.
2. Parse the highest plan number, batch number, bug number, and per-batch feature sequences.
3. Rebuild the `counters`, `batch_feature_seqs`, and `allocations` tables from the scanned state.
4. Add a configurable buffer (default: 100) to the counters to avoid re-allocating IDs that were allocated but not yet pushed.

Standard PostgreSQL backups (Supabase provides automated backups on all tiers) make this scenario unlikely.

### Connection management

Kanbanzai's coordination connections are short-lived: open a connection, execute a query, close the connection. This is by design — coordination happens at entity creation time, which is infrequent relative to other operations.

For the Go implementation, `jackc/pgx` is the recommended driver. It is the most actively maintained Postgres driver for Go, supports connection pooling via `pgxpool`, and handles Supabase's SSL requirements correctly.

The implementation pattern:

```go
// Pseudocode for the coordination layer in Go.
// Not final — the actual implementation will depend on the service-layer design.

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

type CoordinationDB struct {
    pool *pgxpool.Pool
}

func NewCoordinationDB(ctx context.Context, databaseURL string) (*CoordinationDB, error) {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        return nil, fmt.Errorf("coordination db: %w", err)
    }
    return &CoordinationDB{pool: pool}, nil
}

func (db *CoordinationDB) AllocateID(ctx context.Context, projectID, entityType, prefix, slug string) (string, error) {
    var id string
    err := db.pool.QueryRow(ctx,
        "SELECT allocate_id($1, $2, $3, $4)",
        projectID, entityType, prefix, slug,
    ).Scan(&id)
    return id, err
}
```

The pool is created once at startup and reused. For teams that prefer not to hold a persistent connection, a single-connection mode can be used instead — open, query, close per allocation. The pool is the default because it avoids the overhead of TLS handshakes on every allocation.

### Relationship to the full centralized state server

This coordination design is forward-compatible with the centralized-state-server design. Specifically:

1. **Same database** — the coordination tables live in PostgreSQL, the same reference backend chosen by the centralized state design. Adopting the full centralized state server means adding tables, not migrating to a different system.
2. **Same project scoping** — `project_id` applies to both.
3. **Incremental adoption** — a team can start with coordination tables for ID allocation, then later adopt the full centralized state schema for entity state. The coordination tables' data migrates naturally.

The coordination layer does **not** require the service-layer decoupling that the full centralized state design calls for. It is a set of SQL queries inserted at the point of ID allocation, not a replacement of the persistence layer.

## Alternatives Considered

### Alternative A: Do nothing — accept collisions as merge conflicts

Let Git merge conflicts handle ID collisions. When two users allocate `B47`, the second to push gets a conflict and renumbers.

**Trade-offs:**
- Zero implementation cost.
- Collisions are rare in practice for small teams.
- But: collisions produce confusing error states (two entities with the same ID, broken references). Feature display ID collisions are worse — features are created frequently and the `next_feature_seq` counter is per-batch.
- But: the problem gets worse as the team grows. It's a scaling friction that makes Kanbanzai feel unreliable.

**Rejected because:** The collision problem is real and will worsen as Kanbanzai is adopted by larger teams. Accepting it as "just a merge conflict" understates the confusion it causes.

### Alternative B: Jump directly to the full centralized state server

Implement the `design-centralized-state-server.md` plan: PostgreSQL-backed server, service-layer decoupling, canonical state in the database.

**Trade-offs:**
- Solves ID allocation and much more in one move.
- But: requires significant architectural change (persistence abstraction, migration tooling, dual-mode support).
- But: the design itself recommends starting with service-layer decoupling and introducing centralized state as an additive backend.
- But: many teams only need ID allocation — the full server is overkill for the immediate problem.

**Rejected for now, not rejected permanently.** The full centralized state server is the right long-term direction. This coordination server is a stepping stone that solves the immediate problem while building toward it.

### Alternative C: Namespace batch IDs by user

Add a user identifier to batch IDs: `B-sambeau-47` or similar.

**Trade-offs:**
- No server needed — users can't collide with each other.
- But: makes IDs longer and less readable. `B-sambeau-47-F3` vs `B47-F3`.
- But: doesn't solve feature display ID collisions (two users in the same batch).
- But: IDs change if work is handed off between users.
- But: clashes with the existing sequential numbering model.

**Rejected because:** User-namespaced IDs are verbose and don't reflect how teams actually work — work moves between people, and IDs should be stable regardless of who created them.

### Alternative D: Use TSID for all entity types

Give plans and batches TSID-based IDs like features have: `B-01KMRX1SEQV49` instead of `B47`.

**Trade-offs:**
- No collisions, no server needed.
- But: Plan and batch IDs are discussed by humans constantly. `B47` is conversational; `B-01KMRX1SEQV49` is not.
- But: Feature display IDs (`B47-F3`) depend on batch sequential numbers — even if batch IDs were TSIDs, feature display IDs still need a counter.
- But: Would require changing the entire ID system that was just redesigned in B38.

**Rejected because:** The sequential, human-friendly nature of plan and batch IDs is a feature, not a bug. The problem is the allocation mechanism, not the ID format.

### Alternative E: Standalone coordination binary (file-backed or embedded database)

A standalone `kbz-coord serve` binary that stores counters and allocation registries in a JSON file or embedded SQLite database. No external database required.

**Trade-offs:**
- Zero dependencies — no Postgres needed.
- But: teams must deploy and maintain another process.
- But: file-backed persistence has the same durability concerns as the current `.kbz/state/` files (what if the server's disk fails?).
- But: the coordination binary becomes a single point of failure that teams must monitor and maintain.
- But: the centralized-state-server design already targets PostgreSQL — a file-backed coordination server is a dead end that doesn't evolve into the broader design.
- But: managed Postgres is widely available and inexpensive (Supabase free tier, Neon free tier). The operational burden of a separate binary exceeds the operational burden of a managed database.

**Rejected because:** PostgreSQL is the simplest coordination backend — no extra process, no binary to deploy, built-in auth, and forward-compatible with the centralized state server. Managed Postgres makes the operational burden near zero. A standalone binary adds complexity without adding value.

## Decisions

### D1: Introduce a coordination database as a lightweight intermediary, not a full state server

**Context:** The ID collision problem needs a solution now. The full centralized state server is architecturally sound but requires significant groundwork.

**Rationale:** A coordination database that handles only ID allocation and a few coordination functions solves the immediate problem with minimal architectural change. It sits alongside the existing file-backed store rather than replacing it. It can evolve into the full centralized state server when the team is ready.

**Consequences:**
- Single-user projects continue unchanged (zero-config default).
- Team projects get collision-free IDs with a simple PostgreSQL connection string in their config.
- The coordination database's surface area is narrow enough that it can be implemented quickly.
- The coordination database is forward-compatible with the centralized-state-server design.

### D2: Single-user mode is the default; team mode is opt-in via configuration

**Context:** Most Kanbanzai users are single developers. Adding a database requirement by default would destroy the zero-config experience.

**Rationale:** The `coordination.database_url` config key gates all behaviour. Absent → single-user mode (today's behaviour). Present → team mode (coordination database).

**Consequences:**
- No breaking change for existing users.
- Teams add one config line to enable coordination.
- The coordination database is invisible to single users.

### D3: The coordination database stores counters and allocation registries, not entity state

**Context:** The full centralized state server stores canonical entity state in a database. This design is narrower.

**Rationale:** Keeping the stored data minimal (counters and coordination tables only) makes the schema simple and the operational burden low. Entity state remains in `.kbz/state/` files, committed to Git.

**Consequences:**
- The schema is two tables plus a SQL function for ID allocation; additional coordination tables are added incrementally.
- Recovery from data loss is a scan of the repository.
- The database does not need to understand entity schemas or lifecycles.

### D4: Bugs get project-scoped sequential IDs (`BUG-1`, `BUG-2`, ...)

**Context:** Bugs are often discussed by humans, but they don't always have a batch parent. When they do, the parent batch may be closed. A batch-scoped ID like `B24-BUG3` would be confusing when B24 is done.

**Rationale:** A project-global sequential `BUG-{n}` counter is simple, human-friendly, and doesn't require a living parent batch. The coordination database owns the counter, eliminating collisions.

**Consequences:**
- Bug IDs change from TSID (`BUG-01J4AR7WHN4F2`) to sequential (`BUG-47`).
- Bugs created before the coordination database is adopted retain their TSID — no migration needed (TSID and sequential IDs are distinguishable by format).
- The `BUG-{n}` format is consistent with `P{n}` and `B{n}`.

### D5: Feature canonical IDs remain TSID-based; only display IDs are server-allocated

**Context:** Features have two identities — a TSID canonical ID (machine identity) and a batch-scoped display ID (`B24-F3`, human-facing). The collision risk is on the display ID.

**Rationale:** Changing the canonical ID format for features is a larger change than needed. The TSID canonical ID works well for file naming and internal references. Only the display ID needs coordination.

**Consequences:**
- Feature state files keep their current naming: `FEAT-01KMRX1SEQV49-{slug}.yaml`.
- The display ID field in the feature YAML is server-allocated.
- The `next_feature_seq` counter moves from batch state files to the coordination database.

### D6: The coordination database is forward-compatible with the centralized state server

**Context:** The centralized-state-server design is the long-term direction. This design should not create obstacles to that path.

**Rationale:** The coordination database's config format, schema patterns, and project scoping are designed to evolve naturally into the centralized state server. The coordination tables' data can be imported into a future full-state schema. No migration dead-ends.

**Consequences:**
- `coordination.database_url` is intentionally generic — the same config key points to the same Postgres database that the centralized state server would use.
- The coordination tables are a subset of what the centralized server would store.
- Adding the centralized state server later means adding tables and migrating entity state, not changing databases.

### D7: PostgreSQL is the sole coordination backend

**Context:** The original draft considered two backends: PostgreSQL and a standalone `kbz-coord serve` binary. After evaluation, the binary option was rejected.

**Rationale:** A standalone coordination binary introduces operational complexity (deploy, monitor, upgrade, back up) that exceeds the complexity of using managed Postgres. Supabase and Neon offer free tiers that make the operational burden near zero. PostgreSQL also provides built-in authentication, atomic sequences, and transactional guarantees that a file-backed binary would need to implement from scratch. Most importantly, the centralized-state-server design already targets PostgreSQL — a binary backend is a dead end.

**Consequences:**
- Only one backend to implement, test, document, and support.
- Teams that can't or won't use PostgreSQL remain in single-user mode.
- No migration path needed from binary → Postgres.

## Open Questions

None at this stage. The following were resolved during design:

- **Schema migration:** `CREATE TABLE IF NOT EXISTS` on first connection. Supabase's `postgres` role has the necessary privileges; self-managed teams can run the DDL separately if preferred.
- **Bug ID migration:** Existing bugs keep TSID IDs; new bugs get sequential IDs. The formats are distinguishable — no migration needed.
- **Decision IDs:** Stay TSID-based. No coordination needed.
- **Connection pool lifecycle:** A connection pool is the default (avoids TLS handshake overhead). A single-connection-per-allocation mode is supported for infrequent use. Database traffic is very low even for large teams — allocations happen at entity creation time, not on every tool call.

## Dependencies

- `work/_project/design-centralized-state-server.md` — the broader architecture this extends.
- `work/B38-plans-and-batches/B38-design-meta-planning-plans-and-batches.md` — the Plan/Batch ID system this modifies.
- `work/_project/research-storage-architecture-and-progress-visibility.md` — confirms the direction.
- `github.com/jackc/pgx/v5` — recommended Go Postgres driver (new dependency).
