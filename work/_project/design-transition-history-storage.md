| Field  | Value |
|--------|-------|
| Date   | 2026-04-22T00:00:00Z |
| Status | Draft |
| Author | GPT-5.4 |

## Overview

This design proposes a Git-native evolution of Kanbanzai's workflow-state model: keep YAML entity records as the source of truth for current state, add per-entity JSONL transition logs as the canonical history of lifecycle changes, and reduce Git noise by moving from per-transition commits to milestone-based workflow flushes.

This document is one side of a deliberate design comparison. The competing centralized alternative is described in `work/design/centralized-state-server.md`, and the comparative assessment of the two directions is captured in `work/research/state-backend-comparison.md`.

## Goals and Non-Goals

### Goals

- Preserve Kanbanzai's Git-native storage model for canonical workflow state.
- Separate semantic transition history from Git commit history.
- Reduce noisy workflow-only commits that obscure code changes.
- Keep transition history append-only, inspectable, and queryable.
- Preserve a migration path to optional derived indexing without changing canonical storage.

### Non-Goals

- Introduce a shared centralized database as canonical state for this design direction.
- Redesign lifecycle state machines or workflow semantics.
- Require SQLite or any other database as part of the first implementation.
- Eliminate the possibility of a future centralized backend; that alternative is evaluated separately in `work/design/centralized-state-server.md`.

## Problem and Motivation

Kanbanzai currently derives much of its lifecycle transition history from Git commit history and, more recently, auto-commits workflow state after many MCP operations. This gives the system a durable audit trail, but it also creates a poor review experience: commit history becomes dominated by low-level workflow transitions rather than coherent code changes.

This is most visible in high-churn entity lifecycles such as bugs and tasks. A single bug can produce a sequence of commits like `reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed`, even when no code changed between several of those steps. The result is that Git history becomes harder to use for its primary human-facing jobs: understanding code evolution, reviewing implementation, and diagnosing regressions.

The current design also overloads Git with two responsibilities that are related but not identical:

1. **Durable snapshot transport** — persisting project state in a Git-native workflow system.
2. **Semantic event history** — recording who changed an entity from one lifecycle state to another, when, and why.

Using Git commits as the primary transition log has several limitations:

- intermediate transitions can be lost if multiple state changes happen before a commit boundary
- semantic queries require parsing commit messages rather than reading structured data
- concurrent agent activity produces interleaved histories that are hard to read per entity
- commit history becomes noisy in ways that directly undermine the project's own Git commit policy

Kanbanzai should preserve its Git-native model while separating these concerns. The system needs a structured, append-only transition history that is queryable and durable without requiring one Git commit per lifecycle event.

## Design

### Recommended approach

Introduce a **canonical append-only transition log stored as JSONL files in the repository**, while keeping the existing YAML entity records as the source of truth for current state. Git remains the transport and durability mechanism for project state, but it no longer serves as the primary semantic event log for lifecycle transitions.

This design has four parts:

1. **Canonical current state remains in YAML** under `.kbz/state/`.
2. **Canonical transition history is written to per-entity JSONL files** under `.kbz/state/transitions/`.
3. **Git commits become coarser-grained snapshots** of workflow state rather than one-commit-per-transition events.
4. **Optional SQLite indexing remains a future optimisation**, not part of the canonical storage model.

This is intentionally the Git-native side of the broader state-backend decision. The competing centralized approach is documented in `work/design/centralized-state-server.md`, while `work/research/state-backend-comparison.md` explains why this design is the lower-risk near-term response to the current commit-noise problem.

### Storage model

Each entity that supports lifecycle transitions gets a corresponding transition log file. The preferred layout is per-entity and per-type to minimise coupling and merge contention.

Proposed layout:

- `.kbz/state/transitions/features/FEAT-...jsonl`
- `.kbz/state/transitions/tasks/TASK-...jsonl`
- `.kbz/state/transitions/bugs/BUG-...jsonl`
- `.kbz/state/transitions/plans/P...jsonl`
- `.kbz/state/transitions/decisions/DEC-...jsonl`
- `.kbz/state/transitions/incidents/INC-...jsonl`

Each line is one immutable transition event encoded as JSON.

Example event shape:

- `entity_id` — canonical entity ID
- `entity_type` — feature, task, bug, plan, decision, incident
- `from` — previous status
- `to` — new status
- `at` — RFC 3339 UTC timestamp
- `by` — actor identity
- `reason` — optional free-text rationale
- `override` — optional boolean indicating gate bypass
- `override_reason` — optional explanation when override is used
- `source` — MCP tool or CLI action that triggered the transition
- `correlation_id` — optional operation identifier for grouping related writes
- `commit` — optional Git SHA if known at write or flush time

The log is append-only. Corrections are represented as new events, never by editing prior lines.

### Authority and consistency

The entity YAML file remains authoritative for current status. The transition log is authoritative for transition history. These two records are written together as one logical state update.

The write rule is:

- validate the transition against the lifecycle state machine
- update the entity's `status` field
- append the transition event to the entity's JSONL log
- persist both changes atomically from the perspective of the application

If the system cannot write both the YAML update and the JSONL append, the transition fails. The system must not leave the entity in a state where the current status changed but the transition event was not recorded, or vice versa.

### Commit granularity

This design deliberately decouples **state persistence** from **Git commit frequency**.

State files are still written immediately so that the working tree reflects the latest truth. Git commits, however, should occur at coherent workflow boundaries rather than every transition.

Recommended default flush boundaries:

- task completion via `finish`
- feature advancement into `reviewing` or `done`
- bug advancement into `needs-review`, `verified`, or `closed`
- document registration and approval operations that already bundle state with a work artifact
- pre-dispatch safety flush before sub-agent handoff
- explicit merge preparation and merge execution

This preserves durability in the repository working tree while making commit history more useful for humans.

### Query model

Per-entity history is retrieved by reading the entity's JSONL file in chronological order. Cross-entity queries are initially implemented by scanning transition files through service-layer helpers. This keeps the first implementation simple and Git-native.

Examples of supported queries:

- show the full lifecycle history for one entity
- show all entities that entered `needs-review` in a time range
- show all transitions performed by a given actor
- show all override transitions in a plan or feature subtree

If these queries become performance-sensitive, a derived SQLite index can be added later. That index must remain rebuildable from the canonical YAML and JSONL files and must not become the source of truth.

### Health and repair

Health checks should validate transition logs with the following rules:

1. the log file path matches a real entity
2. each event's `entity_id` and `entity_type` match the owning file
3. every `from → to` pair is legal for that entity type
4. timestamps are monotonically non-decreasing
5. the final event's `to` matches the entity YAML `status`
6. the chain is continuous: each event's `from` matches the previous event's `to`

Health failures should be surfaced as warnings or errors depending on severity:

- malformed JSONL or broken continuity is an error
- missing historical fields on older events is a warning during migration
- absence of a transition log for legacy entities is a warning or informational state, not an error

### Migration strategy

Migration should be incremental and low-risk.

#### Phase 1: dual-write for new transitions

- keep existing entity status updates
- add JSONL append on every successful transition
- keep existing auto-commit behaviour temporarily while the new log proves reliable

#### Phase 2: switch history consumers

- update status, health, and inspection paths to read transition history from JSONL instead of Git commit parsing where applicable
- add service helpers for per-entity and cross-entity history queries

#### Phase 3: reduce commit noise

- replace per-transition auto-commit with milestone-based flushes
- retain explicit safety flushes before sub-agent dispatch and merge-sensitive operations

#### Phase 4: optional indexing

- if query cost becomes material, add a derived SQLite index under `.kbz/cache/` or `.kbz/index/`
- rebuild the index from YAML and JSONL sources

Backfill from historical Git commits is optional and best-effort. Existing entities remain valid without complete historical logs. Once the feature is enabled, all new transitions must be recorded in JSONL.

### Failure modes and handling

This design introduces several failure modes that must be handled explicitly. Several of these are the inverse trade-offs of the centralized alternative in `work/design/centralized-state-server.md`: this design preserves inspectability and Git-native transport, but it accepts continued dependence on repository-local files and Git merge behavior.

#### Partial write risk

If YAML and JSONL writes are not coordinated, the system can drift. The implementation must use the same atomic-write discipline already used elsewhere in the repository and treat the pair as one logical transaction.

#### Merge conflicts on transition logs

Per-entity sharding reduces but does not eliminate conflicts. If two branches transition the same entity independently, the conflict is real and should surface. This is preferable to hiding the conflict in commit-message-derived history.

#### Log growth

Per-entity transition volume is expected to remain modest. JSONL growth is therefore acceptable. No retention or compaction policy is needed in the first version.

#### Query cost

Scanning JSONL files is acceptable at current scale. If it becomes slow, the derived SQLite index is the next step. The design intentionally avoids introducing SQLite as canonical storage prematurely.

## Alternatives Considered

### Alternative 1: Keep Git commits as the primary transition log

**Description:** Continue the current model where MCP tools auto-commit workflow state after transitions and derive history from commit messages.

**What it makes easier:**
- no new storage format
- no migration work
- every transition is immediately durable in Git history

**What it makes harder:**
- commit history becomes dominated by workflow noise
- semantic queries remain convention-dependent and expensive
- intermediate transitions can still be obscured by commit boundaries
- Git is forced to serve as both transport and event log

**Why rejected:** This is the current pain point. It preserves the audit trail but degrades the human usefulness of commit history and does not scale cleanly to multi-agent workflows.

### Alternative 2: Store transition history on the entity YAML record

**Description:** Add a `transitions` sequence directly to each entity file, as proposed in the existing transition-log design.

**What it makes easier:**
- one file contains both current state and history
- no extra file lookup per entity
- simple mental model for inspection

**What it makes harder:**
- every transition rewrites the full entity file
- entity files become progressively noisier in diffs
- current-state edits and historical append traffic are tightly coupled
- merge conflicts on active entities become more likely because all history and current state share one file

**Why rejected:** This is workable, but it increases coupling between current state and historical metadata. A separate per-entity JSONL log better preserves append-only semantics and keeps entity YAML focused on current authoritative state.

### Alternative 3: Use SQLite as canonical transition storage

**Description:** Store transition history in an embedded SQLite database and query it directly.

**What it makes easier:**
- transactional writes
- indexed queries and analytics
- efficient cross-entity filtering

**What it makes harder:**
- canonical state becomes less transparent and less Git-native
- committed database files are poor review artifacts
- ignored database files are not portable canonical records
- authority boundaries between YAML and SQLite become more complex

**Why rejected:** SQLite is a strong indexing layer but the wrong first canonical store for this problem. It introduces architectural drift before the simpler Git-native option has been exhausted.

### Alternative 4: Use Postgres or another shared database

**Description:** Move transition history to a shared service-backed database.

**What it makes easier:**
- central coordination across users and agents
- strong concurrency guarantees
- rich analytics and real-time views

**What it makes harder:**
- requires infrastructure and operations
- weakens repository-local workflows
- shifts Kanbanzai away from its Git-native identity
- turns a storage refinement into a product-architecture change

**Why rejected:** This is disproportionate to the problem being solved. It may become appropriate only if Kanbanzai intentionally evolves into a centrally hosted workflow platform. That broader direction is explored in `work/design/centralized-state-server.md`, and the comparative recommendation is captured in `work/research/state-backend-comparison.md`.

## Dependencies

This design depends on and should remain aligned with:

- `work/design/git-commit-policy.md` for the requirement that commit history remain understandable and useful for review and diagnosis
- `work/design/transition-log-design.md` for the earlier on-entity transition-history concept that this document refines toward per-entity JSONL logs
- `work/design/centralized-state-server.md` for the competing centralized alternative
- `work/research/state-backend-comparison.md` for the comparative recommendation and trade-off framing

## Decisions

- **Decision:** Keep YAML entity records as the source of truth for current state.
  - **Context:** Current status is already modeled and validated in `.kbz/state/` entity files.
  - **Rationale:** This preserves the existing Git-native architecture and avoids a disruptive authority shift.
  - **Consequences:** Transition history must be stored separately and validated against the YAML status field.

- **Decision:** Store canonical transition history in per-entity JSONL files under `.kbz/state/transitions/`.
  - **Context:** The system needs append-only, structured history without turning Git commits into the event log.
  - **Rationale:** JSONL is simple, inspectable, append-friendly, and compatible with Git-native storage.
  - **Consequences:** The system must manage an additional file per entity and provide service helpers for reading and validating logs.

- **Decision:** Decouple immediate state writes from per-transition Git commits.
  - **Context:** The current one-commit-per-transition model creates noisy history that obscures code changes.
  - **Rationale:** State persistence and commit granularity solve different problems and should not be forced into the same boundary.
  - **Consequences:** The system needs explicit flush boundaries and safety rules for when dirty workflow state must be committed.

- **Decision:** Use milestone-based workflow commits as the default after migration.
  - **Context:** Some workflow state must still be committed promptly for collaboration and safety.
  - **Rationale:** Milestone-based commits preserve coherent history while still making state visible in Git at meaningful boundaries.
  - **Consequences:** Some transitions will exist in the working tree before they are grouped into a Git commit; tooling must flush before sensitive operations such as handoff and merge.

- **Decision:** Treat SQLite as an optional derived index, not canonical storage.
  - **Context:** Cross-entity history queries may eventually need indexed lookup.
  - **Rationale:** SQLite solves query performance well, but canonical storage should remain transparent, portable, and Git-tracked.
  - **Consequences:** If added later, SQLite must be rebuildable from YAML and JSONL and should not be committed.
