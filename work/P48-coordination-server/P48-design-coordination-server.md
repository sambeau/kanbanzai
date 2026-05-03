| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03T19:01:38Z           |
| Status | Draft                          |
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

A lightweight coordination server that provides centralized ID allocation for plans, batches, features, and bugs in team deployments. Single-user projects continue to use local file-scanned allocation with zero configuration. Team projects point at a shared coordination server and get collision-free IDs. The server also handles a small set of additional coordination functions that benefit from a single authority: feature sequence counters, batch/plan status visibility, and merge gating.

## Goals and Non-Goals

### Goals

- Eliminate ID collisions for Plans (`P{n}`), Batches (`B{n}`), Feature display IDs (`B{n}-F{m}`), and Bugs (`BUG-{n}`) in multi-user deployments.
- Require zero configuration for single-user projects — they continue unchanged.
- Be a simple, stateless (or near-stateless) service that a team can run with minimal operational burden.
- Serve as a stepping stone toward the full centralized state server design.
- Handle feature `next_feature_seq` counters to prevent two servers from allocating the same `B24-F3`.

### Non-Goals

- Replace the Git-native file store as canonical workflow state.
- Store or serve entity state beyond what is needed for ID allocation and coordination.
- Handle task IDs — tasks use TSID13 and have effectively zero collision risk.
- Handle document IDs — documents use owner/slug-derived IDs that are naturally namespaced.
- Provide a full PostgreSQL-backed centralized state database (that is the broader design, deferred).
- Support the centralized state server's broader query, analytics, or real-time state sharing goals.

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

The full centralized state server design solves this comprehensively but requires significant architectural change. This design proposes a minimal, focused solution: a coordination server that owns only ID allocation and a small set of tightly-scoped coordination functions.

## Design

### Single-user mode (no change)

When no coordination server is configured, Kanbanzai behaves exactly as it does today. ID allocation scans local `.kbz/state/` directories and increments. No new configuration, no new infrastructure, no change in behaviour.

This is the default. A single-user project never needs to know the coordination server exists.

### Team mode (coordination server)

A team deploys a single coordination server instance. Each team member's Kanbanzai server is configured to point at it.

The coordination server is a lightweight HTTP service. It does not store entity state — it stores only counters and a registry of allocated IDs. It does not need a database; an in-memory store with a JSON file for persistence across restarts is sufficient for the initial version. (PostgreSQL can be introduced later if needed, consistent with the centralized-state-server's reference backend choice.)

#### Configuration

In `.kbz/config.yaml`:

```yaml
coordination:
  server_url: "https://kbz-coord.example.com"
  project_id: "my-project"
```

When `coordination.server_url` is absent or empty, Kanbanzai operates in single-user mode. When present, it operates in team mode.

The `project_id` scopes the coordinator's state to a specific project — one coordination server can serve multiple projects without ID collision.

#### API surface

```
POST /v1/allocate
  Request:  { "project_id": "...", "entity_type": "batch", "prefix": "B", "slug": "my-batch" }
  Response: { "id": "B47-my-batch", "number": 47 }

POST /v1/allocate-feature-display-id
  Request:  { "project_id": "...", "batch_id": "B24-auth-system" }
  Response: { "display_id": "B24-F3", "sequence": 3 }

POST /v1/allocate-bug-id
  Request:  { "project_id": "..." }
  Response: { "id": "BUG-47", "number": 47 }

GET /v1/next-counter?project_id=...&entity_type=batch&prefix=B
  Response: { "next": 47 }
```

The server maintains:

- A per-project, per-entity-type counter for plans and batches.
- A per-project counter for bugs.
- A per-batch `next_feature_seq` counter.
- A registry of allocated IDs (idempotency: re-requesting the same slug returns the same ID).

#### ID allocation flow (team mode)

1. User creates a batch via `entity(action: "create", type: "batch", ...)`.
2. The MCP tool checks for `coordination.server_url` in config.
3. If present, it calls `POST /v1/allocate` with the entity type, prefix, and slug.
4. The server allocates the next number, records the allocation, returns the full ID.
5. The MCP tool writes the entity YAML file with the server-allocated ID.

Bugs are similar but project-scoped (no batch prefix needed).

Features are different: their canonical ID remains TSID-based (no change), but their display ID (`B{n}-F{m}`) is allocated via `POST /v1/allocate-feature-display-id`, which increments the batch's `next_feature_seq` counter atomically on the server.

#### What the coordination server stores

```
projects/
  my-project/
    counters/
      plan_P: 47
      batch_B: 48
      bug: 23
    batches/
      B24-auth-system/
        feature_seq: 5
    allocations/
      P47-coordination-server
      B48-something-else
      BUG-23
```

This is intentionally minimal. The server does not store entity names, statuses, summaries, or any other state. It is an ID allocator with a small amount of coordination state.

### Additional coordination functions

The coordination server's narrow charter is ID allocation, but a few additional functions naturally fit the same pattern (single authority, low data volume, immediate value in team settings):

#### 1. Feature sequence counters

Already covered above — the server owns `next_feature_seq` per batch.

#### 2. Batch/Plan status visibility

When a user calls `status()`, the server can provide the current known state of all batches and plans. This is not canonical state (the files are), but it provides immediate visibility without waiting for a Git push/pull cycle. The server learns of status changes when IDs are allocated or when the client explicitly notifies it:

```
POST /v1/notify-status
  Request:  { "project_id": "...", "entity_id": "B47-my-batch", "status": "active" }
  Response: { "ok": true }
```

This is optional — the server's status view is a cache, not the source of truth.

#### 3. Merge gating

The coordination server can act as a simple merge queue: before merging, a client checks whether any other merge is in progress for the same batch or feature. This prevents two users from simultaneously merging conflicting state changes.

```
POST /v1/acquire-merge-lock
  Request:  { "project_id": "...", "entity_id": "B47-my-batch" }
  Response: { "acquired": true, "lock_id": "..." }

POST /v1/release-merge-lock
  Request:  { "project_id": "...", "lock_id": "..." }
  Response: { "ok": true }
```

This is also optional — if the server is unavailable, the merge proceeds without the lock (with a warning).

### Failure modes

#### Server unavailable

If the coordination server is unreachable, Kanbanzai falls back to local allocation with a warning. The next time the server is reachable, the local counter may have advanced past the server's counter. The server detects this (the allocated ID already exists in its registry) and fast-forwards its counter.

This means brief server outages do not block work, but they reintroduce the (small) window for collisions. Teams that want zero collision risk should ensure the server is available.

#### Network partition

If two users are partitioned from each other but both can reach the server, there is no collision risk — the server serializes allocations. If a user is partitioned from the server, they fall back to local allocation (see above).

#### Server data loss

If the coordination server loses its data (e.g., the JSON persistence file is deleted), it re-derives its counters by scanning the repository's `.kbz/state/` directory at startup. This is the same scan-based approach used in single-user mode, but run once at server startup rather than per-allocation. The scanned state may be slightly stale (between pushes), so the server adds a configurable buffer (default: 100) to avoid re-allocating IDs that were allocated but not yet pushed.

### Relationship to the full centralized state server

This coordination server is designed to be forward-compatible with the centralized-state-server design. Specifically:

1. **Same server URL configuration** — the `coordination.server_url` field can evolve into the centralized state server's endpoint.
2. **Same project scoping** — `project_id` applies to both.
3. **Incremental adoption** — a team can start with the coordination server for ID allocation, then later adopt the full centralized state server for entity state. The coordination server's counter state migrates naturally into the centralized server's database.

The coordination server does **not** require the service-layer decoupling that the full centralized state design calls for. It is a narrow HTTP call inserted at the point of ID allocation, not a replacement of the persistence layer.

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

## Decisions

### D1: Introduce a coordination server as a lightweight intermediary, not a full state server

**Context:** The ID collision problem needs a solution now. The full centralized state server is architecturally sound but requires significant groundwork.

**Rationale:** A coordination server that handles only ID allocation and a few coordination functions solves the immediate problem with minimal architectural change. It sits alongside the existing file-backed store rather than replacing it. It can evolve into the full centralized state server when the team is ready.

**Consequences:**
- Single-user projects continue unchanged (zero-config default).
- Team projects get collision-free IDs with a simple server deployment.
- The coordination server's API surface is narrow enough that it can be implemented quickly.
- The coordination server is forward-compatible with the centralized-state-server design.

### D2: Single-user mode is the default; team mode is opt-in via configuration

**Context:** Most Kanbanzai users are single developers. Adding a server requirement by default would destroy the zero-config experience.

**Rationale:** The `coordination.server_url` config key gates all behaviour. Absent → single-user mode (today's behaviour). Present → team mode (coordination server).

**Consequences:**
- No breaking change for existing users.
- Teams add one config line to enable coordination.
- The coordination server is invisible to single users.

### D3: The coordination server stores counters and allocation registries, not entity state

**Context:** The full centralized state server stores canonical entity state in a database. This design is narrower.

**Rationale:** Keeping the server's stored data minimal (counters only) makes it simple to implement, deploy, and recover. Entity state remains in `.kbz/state/` files, committed to Git.

**Consequences:**
- The server can be backed by a simple JSON file, not a database.
- Recovery from data loss is a scan of the repository.
- The server does not need to understand entity schemas or lifecycles.

### D4: Bugs get project-scoped sequential IDs (`BUG-1`, `BUG-2`, ...)

**Context:** Bugs are often discussed by humans, but they don't always have a batch parent. When they do, the parent batch may be closed. A batch-scoped ID like `B24-BUG3` would be confusing when B24 is done.

**Rationale:** A project-global sequential `BUG-{n}` counter is simple, human-friendly, and doesn't require a living parent batch. The coordination server owns the counter, eliminating collisions.

**Consequences:**
- Bug IDs change from TSID (`BUG-01J4AR7WHN4F2`) to sequential (`BUG-47`).
- Bugs created before the coordination server is adopted retain their TSID — no migration needed (TSID and sequential IDs are distinguishable by format).
- The `BUG-{n}` format is consistent with `P{n}` and `B{n}`.

### D5: Feature canonical IDs remain TSID-based; only display IDs are server-allocated

**Context:** Features have two identities — a TSID canonical ID (machine identity) and a batch-scoped display ID (`B24-F3`, human-facing). The collision risk is on the display ID.

**Rationale:** Changing the canonical ID format for features is a larger change than needed. The TSID canonical ID works well for file naming and internal references. Only the display ID needs coordination.

**Consequences:**
- Feature state files keep their current naming: `FEAT-01KMRX1SEQV49-{slug}.yaml`.
- The display ID field in the feature YAML is server-allocated.
- The `next_feature_seq` counter moves from batch state files to the coordination server.

### D6: The coordination server is forward-compatible with the centralized state server

**Context:** The centralized-state-server design is the long-term direction. This design should not create obstacles to that path.

**Rationale:** The coordination server's config format, API patterns, and project scoping are designed to evolve naturally into the centralized state server. The coordination server's counter state can be imported into a future PostgreSQL database. No migration dead-ends.

**Consequences:**
- `coordination.server_url` is intentionally vague — it can point to a coordination server now or a full state server later.
- The API is versioned (`/v1/`) to allow evolution.
- The coordination server's data model is a subset of what the centralized server would store.

## Open Questions

1. **Server deployment model.** How do teams deploy the coordination server? Options: a single binary (`kbz-coord serve`), a Docker container, a cloud service. The design should not prescribe one, but the simplest path should be documented.

2. **Authentication.** Should the coordination server require authentication? For teams behind a VPN, probably not. For internet-facing deployments, yes. This can be deferred — the initial version can assume a trusted network.

3. **Server data persistence format.** JSON file is proposed for simplicity. SQLite is an alternative that provides transactional safety without a separate database process. This is an implementation detail but worth deciding before building.

4. **Bug ID migration strategy.** When a team adopts the coordination server, existing bugs have TSID IDs. New bugs get sequential IDs. Should existing bugs be left as-is (no migration) or re-numbered? Leaning toward no migration — TSID and sequential are distinguishable by format and coexist fine.

5. **Should the coordination server also handle Decision IDs?** Decisions (`DEC-{TSID13}`) are currently TSID-based and rarely discussed by humans. Leaning toward leaving them as-is, but the question is open.

## Dependencies

- `work/_project/design-centralized-state-server.md` — the broader architecture this extends.
- `work/B38-plans-and-batches/B38-design-meta-planning-plans-and-batches.md` — the Plan/Batch ID system this modifies.
- `work/_project/research-storage-architecture-and-progress-visibility.md` — confirms the direction.
