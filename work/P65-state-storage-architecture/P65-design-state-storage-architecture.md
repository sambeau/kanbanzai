# Design: State Storage Architecture for Kanbanzai

**Plan:** P65 — State storage architecture
**Status:** Draft
**Date:** 2026-05-12
**Author:** Architectural synthesis (Claude Opus 4.7)
**Source documents synthesised:**
- `work/_project/design-transition-history-storage.md` (Git-native JSONL approach)
- `work/P44-model-routing-agent-launcher/P44-design-deterministic-workflow-controller.md` §12 Open Question 1 (durable workflow state)
- `work/P44-model-routing-agent-launcher/P44-design-deterministic-workflow-controller.md` §5.5 (audit log requirements)

**Companion plan:** P44 — Model routing and agent launcher (the orchestration architecture journey). This design and P44 are architecturally coupled but separately deliverable. See §10.

---

## 1. Overview

This design proposes the long-term storage substrate for Kanbanzai's workflow data. It evaluates two directions:

1. **Extended Git-native** — keep YAML entity records authoritative, add per-entity JSONL transition logs, formalise milestone-based commits. This is the direction described in `design-transition-history-storage.md`.
2. **Transactional database** — promote a database (SQLite first, with Postgres as a future option) to the canonical store for state that requires transactional guarantees, queryability, durable workflow execution, or append-only audit semantics. Keep Git as transport for human-reviewable artefacts (documents, generated registry sections).

The recommendation is a **hybrid converging on a database for execution state**: ship the Git-native JSONL improvements as a near-term Phase 1, then promote SQLite to the canonical store for execution-tier data (durable workflow state, audit log, transition events, entity records) in Phase 2. Documents, knowledge entries, and project artefacts remain Git-native indefinitely.

This recommendation differs from `design-transition-history-storage.md`, which intentionally treats SQLite as a future-only optional index. It differs because P44's deterministic-orchestration architecture needs durable execution state, atomic multi-write transactions, and a queryable audit log on a much shorter timescale than the original Git-native design assumed. The storage decision was deferrable when Kanbanzai was a chat-orchestrated MCP server. It is no longer deferrable once code-driven controllers ship.

---

## 2. Goals and Non-Goals

### 2.1 Goals

1. **Provide a substrate for durable workflow execution.** P44 Phase 2 stage controllers must persist workflow state, replay from checkpoint, and survive process restarts. The substrate must support these without a custom append-only file format per workflow.
2. **Provide transactional multi-entity writes.** Today, transitioning a feature can require updating the feature, creating tasks, registering documents, and recording transition events. A YAML-on-disk model treats each as a separate filesystem write; a database can make them atomic.
3. **Provide a queryable audit log.** P44 §5.5 specifies an append-only structured event log used by the eval harness, drift-detection job, and human debugging. A database makes "every dispatch in the last week where the verifier failed" a SQL query, not a JSONL scan.
4. **Preserve Git-native semantics for human-reviewable artefacts.** Documents, knowledge entries, generated registry sections, and ADRs remain Markdown/YAML in the repo. They are reviewed in PRs, evolve with the codebase, and benefit from Git history.
5. **Preserve repository-local execution.** Kanbanzai must continue to work offline, on a single laptop, with no shared infrastructure. SQLite — file-based, embedded, zero-administration — preserves this property.
6. **Provide a clear migration path.** Existing installs must not be broken. The migration must be incremental, reversible, and tooled.
7. **Support eventual federation.** The architecture must not foreclose a future where multiple agents or developers share workflow state across a hosted backend (Postgres). This is a future option, not a Phase 1 requirement.

### 2.2 Non-Goals

- **Replacing the Git working tree for documents or code.** Documents stay where they are. Knowledge entries stay where they are. Source code stays where it is.
- **Building a hosted Kanbanzai SaaS in this plan.** Postgres readiness is a design constraint, not a deliverable.
- **Solving observability beyond the audit log.** Metrics, dashboards, and tracing are downstream consumers; this design provides the data, not the views.
- **Replacing the existing YAML entity model in user-visible terms.** From the agent's perspective, `entity(action: get)` returns the same shape regardless of whether the backing store is YAML files or SQLite rows.

---

## 3. Problem and Motivation

Kanbanzai's current storage model is YAML files in `.kbz/state/` plus Markdown documents in `work/`, with Git providing both transport and informal audit history. This worked while Kanbanzai was a chat-orchestrated MCP server: one agent at a time, sequential operations, infrequent state changes between commits.

Three pressures break this model:

### 3.1 Pressure 1: Commit noise

The current model auto-commits workflow state after most MCP operations to preserve audit history. The result is a Git history dominated by lifecycle transitions rather than coherent code changes. `design-transition-history-storage.md` documents this in detail. The Git-native JSONL approach addresses commit noise but does nothing for the next two pressures.

### 3.2 Pressure 2: Multi-write transactionality

A single MCP call can require multiple correlated writes:

- `entity(action: transition, advance: true)` walks a feature through several lifecycle states, each of which may trigger gates, side effects, and entity updates.
- `finish(task_id, knowledge: [...])` updates the task's status, contributes knowledge entries, and may trigger feature advancement.
- `decompose(action: apply)` creates many tasks atomically — partial creation leaves a feature with a half-decomposed plan.
- `merge(action: execute)` updates the feature, marks the worktree merged, registers a merge event, and triggers verification.

Today these are sequential filesystem writes with no rollback. A crash between writes leaves the system in an inconsistent state that `health` later detects and (sometimes) repairs. The transition-history design explicitly acknowledges this in its "two-file atomicity" section: it accepts eventual consistency and adds a health check to detect drift. That is a defensible position for a single-user, low-throughput tool. It is not a defensible position for the deterministic orchestration architecture P44 is building, where a stage controller may perform tens of correlated writes per dispatch and replay-from-crash is a first-class requirement.

### 3.3 Pressure 3: Durable workflow execution

P44 §5.2.5 commits to async stage controllers on a "minimal in-tree durable-execution layer (~500 LOC)". P44 §12 Open Question 1 asks: *"Where does `internal/durable` persist state?"* The options listed are SQLite, JSONL per workflow, or a custom append-only format.

Building durable execution on JSONL files is feasible but reinvents wheels: write-ahead logging, crash recovery, indexed queries, garbage collection, schema evolution. SQLite gives all of this for free, has been embedded in production systems for decades, and adds zero operational burden (it's a file).

The audit log (P44 §5.5) is the same shape: append-only, structured, queried by correlation ID, time range, entity, event type. A JSONL file works for thousands of events; SQLite scales to millions and supports the queries the eval harness and drift-detection jobs need.

### 3.4 Pressure 4: Concurrency

The current model is implicitly single-writer. Two agents (or two MCP clients) writing the same entity file race on the filesystem. P44 Phase 2 introduces parallel stage controllers that may concurrently update sibling tasks. Transitioning to a database with row-level locking eliminates an entire class of subtle bugs that the file-based model papers over.

### 3.5 What changed since the transition-history design was written

The transition-history design was correct for the system as it existed: a chat-orchestrated MCP server with low write-rate workflow state. Three things have changed:

- P44's deterministic-orchestration commitment makes durable workflow execution a near-term requirement, not a "if it becomes performance-sensitive" future option.
- P44's audit log requires queryability the JSONL approach does not provide cheaply.
- The four-plan record (P50–P58) demonstrated that the system needs much stronger consistency guarantees than the current YAML-on-disk model offers, because partial state is now actively dangerous (silent failures cascade into dispatched sub-agents).

The transition-history design's recommendation to defer SQLite was a reasonable bet against the wrong workload. Under the workload P44 creates, the bet no longer pays.

---

## 4. Recommendation

**Adopt SQLite as the canonical store for execution-tier state, in two phases. Keep the Git-native model for human-reviewable artefacts.**

### 4.1 Tiered storage model

| Tier | Data | Store | Rationale |
|---|---|---|---|
| **Execution** | Entity records, lifecycle status, transition events, durable workflow state, audit log, locks, leases | SQLite (`.kbz/state/state.db`) | Transactional, queryable, supports concurrency, supports replay, embedded |
| **Artefact** | Documents, dev-plans, specifications, designs, reports, ADRs | Git-tracked Markdown (current location) | Human-reviewable, evolves with codebase, benefits from PR review |
| **Knowledge** | Knowledge entries (KE-*), retrospective signals, profile YAML | Git-tracked YAML (current location) | Human-curatable, version-controlled, mergeable |
| **Index** | Document section graph, knowledge graph projections, code graph | SQLite (`.kbz/cache/index.db`) | Rebuildable from Git-tracked sources; not canonical |

The split is along a clear boundary: **state that changes during execution and needs transactional guarantees** goes to SQLite; **state that humans author, review, and evolve through PRs** stays in Git.

This is the same boundary that successful systems like Bazel (BUILD files in Git, action cache in SQLite), CocoaPods (Podfile in Git, Pods.lock in Git, derived data in SQLite), and modern site generators (content in Git, build cache in SQLite) draw.

### 4.2 What's NOT in SQLite

To be explicit about the boundary:

- **Not in SQLite:** documents, knowledge entries, profile YAML, stage bindings, skills, roles, ADRs, generated registry sections, source code, README, AGENTS.md.
- **In SQLite:** entity records (features, tasks, bugs, plans, batches, decisions, incidents), lifecycle transitions, durable workflow state (P44 Phase 2), audit events (P44 §5.5), worktree records, locks/leases, document records (the metadata about documents — the documents themselves stay in Git).

The document content stays Git-native; the *record* of "this document exists, was registered by X at time Y, approved by Z" moves to SQLite. This is symmetric with how the entity records work today: the entity YAML is the record, the documents linked from it are separate artefacts.

### 4.3 Why SQLite first, not Postgres

- **Zero operational burden.** A file. No server, no connection pool, no migrations infrastructure required (the migration tool is one binary).
- **Embedded.** Kanbanzai stays a self-contained CLI/MCP server. No infrastructure decisions imposed on consumers.
- **Production-grade.** SQLite is one of the most battle-tested codebases in the industry. Its consistency guarantees exceed anything a custom JSONL approach could offer.
- **Transactional.** Multi-entity writes become `BEGIN ... COMMIT`. Atomicity is the database's responsibility, not the application's.
- **Queryable.** The audit log queries P44 needs are SQL one-liners.
- **Concurrent.** SQLite's WAL mode handles multiple readers and a single writer well; for Kanbanzai's workload (single MCP server process, many in-process callers) this is sufficient.
- **Forward-compatible with Postgres.** SQL dialect differences are real but manageable. The SQL written for SQLite can target Postgres later with a thin compatibility layer.

### 4.4 What stays from the transition-history design

The transition-history design's core insight — *separate semantic transition history from Git commit history* — is preserved. Transition events become rows in SQLite instead of lines in JSONL files. The semantic content is identical. The win on commit noise is identical (commits no longer record every transition). The architectural posture is the same: one canonical record per concern, queryable on demand.

The transition-history design's per-entity JSONL approach can ship as a Phase 1 stepping stone (see §6). It addresses commit noise immediately and produces a structured event format that Phase 2 imports cleanly into SQLite tables.

---

## 5. Schema sketch

Indicative, not normative — final schema emerges during specification.

### 5.1 Core tables

```sql
-- Entity records (replaces .kbz/state/{features,tasks,bugs,...}/*.yaml)
CREATE TABLE entities (
    id              TEXT PRIMARY KEY,                -- "FEAT-01ABC..."
    type            TEXT NOT NULL,                   -- "feature" | "task" | "bug" | "plan" | "batch" | "decision" | "incident"
    slug            TEXT NOT NULL,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL,
    parent          TEXT,                            -- parent entity ID (FK soft)
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    metadata        JSON NOT NULL                    -- everything not promoted to a column
);
CREATE INDEX idx_entities_type_status ON entities(type, status);
CREATE INDEX idx_entities_parent ON entities(parent);

-- Lifecycle transitions (replaces JSONL transition logs)
CREATE TABLE transitions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id       TEXT NOT NULL REFERENCES entities(id),
    entity_type     TEXT NOT NULL,
    from_status     TEXT,                            -- NULL for creation
    to_status       TEXT NOT NULL,
    at              TIMESTAMP NOT NULL,
    by              TEXT NOT NULL,                   -- actor identity
    reason          TEXT,
    override        BOOLEAN NOT NULL DEFAULT FALSE,
    override_reason TEXT,
    source          TEXT NOT NULL,                   -- MCP tool / CLI action
    correlation_id  TEXT,                            -- groups multi-write operations
    commit_sha      TEXT                             -- Git SHA if known at write time
);
CREATE INDEX idx_transitions_entity ON transitions(entity_id, at);
CREATE INDEX idx_transitions_correlation ON transitions(correlation_id);

-- Document records (the metadata; the documents themselves stay in Git)
CREATE TABLE documents (
    id              TEXT PRIMARY KEY,                -- "P44-.../report-..."
    owner           TEXT NOT NULL,                   -- entity ID
    type            TEXT NOT NULL,
    title           TEXT NOT NULL,
    path            TEXT NOT NULL,                   -- repo-relative
    content_hash    TEXT NOT NULL,
    status          TEXT NOT NULL,                   -- "draft" | "approved" | "superseded"
    created_at      TIMESTAMP NOT NULL,
    approved_at     TIMESTAMP,
    approved_by     TEXT,
    superseded_by   TEXT REFERENCES documents(id)
);
```

### 5.2 P44 Phase 2 tables

```sql
-- Durable workflow state (P44 §5.2.5 internal/durable)
CREATE TABLE workflows (
    id              TEXT PRIMARY KEY,                -- correlation ID
    controller      TEXT NOT NULL,                   -- "developing" | "reviewing" | "verifying"
    entity_id       TEXT NOT NULL REFERENCES entities(id),
    state           TEXT NOT NULL,                   -- "running" | "waiting_signal" | "completed" | "failed"
    checkpoint      JSON NOT NULL,                   -- replay state
    started_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    next_run_at     TIMESTAMP,                       -- for scheduled retries
    failure_reason  TEXT
);
CREATE INDEX idx_workflows_state_next_run ON workflows(state, next_run_at);

-- Append-only audit log (P44 §5.5)
CREATE TABLE audit_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ts              TIMESTAMP NOT NULL,
    correlation_id  TEXT,
    event           TEXT NOT NULL,                   -- "dispatch" | "gate_eval" | "verifier_verdict" | "transition" | "provider_call" | "checkpoint" | "override"
    entity_id       TEXT,
    actor           TEXT NOT NULL,
    role            TEXT,
    skill           TEXT,
    model           TEXT,
    tokens_in       INTEGER,
    tokens_out      INTEGER,
    tool_calls      INTEGER,
    outcome         TEXT,                            -- "ok" | "fail" | "rework_required" | "checkpoint"
    details         JSON
);
CREATE INDEX idx_audit_ts ON audit_events(ts);
CREATE INDEX idx_audit_entity ON audit_events(entity_id, ts);
CREATE INDEX idx_audit_correlation ON audit_events(correlation_id);
CREATE INDEX idx_audit_event ON audit_events(event, ts);
```

### 5.3 Migration metadata

```sql
CREATE TABLE schema_migrations (
    version         INTEGER PRIMARY KEY,
    applied_at      TIMESTAMP NOT NULL
);
```

---

## 6. Phasing

### Phase 0 — Decision and pre-work (current)

- Approve this design.
- Confirm SQLite as the substrate (vs. extended JSONL-only).
- Spike the Go SQLite driver choice (`mattn/go-sqlite3` vs `modernc.org/sqlite`).
- Add ADR-NN: "State storage substrate".

### Phase 1 — Ship the JSONL transition log as a stepping stone (2–4 weeks)

This phase implements `design-transition-history-storage.md` exactly as drafted, with one small addition: the JSONL event format matches the SQLite `transitions` table schema 1:1 so Phase 2 import is mechanical.

**Deliverables:**
- Per-entity JSONL transition logs (`.kbz/state/transitions/<type>/<id>.jsonl`)
- Health check rule 7 (gap detection between YAML status and JSONL final event)
- Milestone-based commit boundaries (replace per-transition auto-commit)
- Service helpers for per-entity and cross-entity history queries

**Why this stepping stone:**
- Addresses commit noise immediately (the most visible current pain).
- Produces structured event data that Phase 2 imports without transformation.
- Unblocks P44 Phase 1 (which doesn't depend on durable workflow state — only Phase 2 does).
- Buys time to spike SQLite driver choice and integration patterns under low risk.

**Phase 1 exit criteria:**
- All transitions writing to JSONL.
- No per-transition commits.
- Health check detecting and repairing log gaps.
- 30+ days of dual-write data accumulated for Phase 2 import validation.

### Phase 2 — SQLite for execution-tier state (1–3 months)

**Deliverables:**
- `internal/storage/sqlite` package with tables from §5.
- `EntityStore` interface re-implemented over SQLite; YAML files become a *read-fallback* during migration, then deprecated.
- One-shot import tool: `kbz state import` reads existing YAML + JSONL and populates SQLite.
- Audit log writes go directly to SQLite (P44 §5.5).
- Durable workflow state goes to SQLite (P44 §5.2.5; this is what Open Question 1 was waiting for).
- `kbz state export` produces a portable bundle (YAML + JSONL) for backup, debugging, and sharing.
- Document *records* migrate to SQLite; document *content* stays in Git.

**Coexistence rules:**
- During Phase 2.0–2.N, YAML files remain on disk as a passive backup. Reads come from SQLite; writes go to both (dual-write) for the cohort window.
- Phase 2.exit: dual-write disabled, SQLite is sole authority, YAML files archived under `.kbz/state/legacy/` for one minor version.

**Phase 2 exit criteria:**
- All MCP tool reads served from SQLite.
- All MCP tool writes committed to SQLite.
- 60+ days of SQLite-only operation across at least 20 closed features and 50 closed tasks.
- Backup/restore tested end-to-end.
- Zero data-loss incidents.

### Phase 3 — Optional: Postgres readiness (6–12 months, gated on need)

Only triggered if Kanbanzai begins to support shared/hosted use cases.

- Abstract SQL behind a `Store` interface that can target SQLite or Postgres.
- Test suite runs against both.
- Postgres-specific tuning (connection pooling, advisory locks for cross-process coordination).

This phase has no fixed timeline and is included only to confirm Phase 2 doesn't foreclose it.

---

## 7. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Loss of human inspectability of state files | High concern, low actual impact | Medium | `kbz state export` produces YAML+JSONL bundle on demand; `kbz state inspect <id>` prints the same shape humans see today |
| Backup story unclear once SQLite is canonical | Medium | High | Phase 2 deliverable: `kbz state backup` produces a single-file bundle (DB copy + Git SHAs); `kbz state restore` rehydrates |
| Concurrent writes to SQLite from multiple MCP server processes | Low (not the deployment model today) | Medium | WAL mode supports multi-reader/single-writer; document the constraint; future Postgres path handles multi-writer |
| Schema migration goes wrong | Medium | High | Versioned migrations with explicit down-paths; full pre-migration backup; canary on a dev install before stable |
| Driver choice locks us in | Medium | Low | Both candidates (`mattn/go-sqlite3`, `modernc.org/sqlite`) implement `database/sql`; swap is mechanical |
| Phase 2 import bugs corrupt state | Low | High | Import is non-destructive (writes new DB, leaves YAML untouched); validate with read-back diff before flipping authority |
| Documents-in-Git, records-in-DB drift | Medium | Medium | Health check: every document record's `path` and `content_hash` must match a real file on disk |
| Commit-history loss for old transitions | Low | Low | Git history isn't deleted; the reduction is in *new* commits; old history remains queryable |
| Power users expect to grep `.kbz/state/*.yaml` | Medium | Low | `kbz state inspect` and `kbz state export` cover this; document the change clearly; keep Phase 2.0–2.N dual-write window long |
| Knowledge entries or profile YAML accidentally migrated | Low | Medium | Explicit allowlist of entity types in import; tests assert knowledge files untouched |

---

## 8. Alternatives Considered

### 8.1 Pure Git-native (the original transition-history design)

**Description:** Stop at Phase 1. JSONL transition logs, milestone commits, no SQLite.

**Why rejected as the long-term answer:** Doesn't solve P44 Phase 2's needs (durable workflow state, transactional multi-write, queryable audit log). Re-implementing those on JSONL is more work than adopting SQLite, and the result is worse on every axis (consistency, queryability, concurrency).

**Why preserved as Phase 1:** It's the right immediate response to commit noise and produces data that Phase 2 imports cleanly. Shipping it does not foreclose Phase 2.

### 8.2 SQLite-from-day-one (skip Phase 1)

**Description:** Go straight to SQLite for transitions, audit, workflows, entities. Do the migration in one phase.

**Why rejected:** P44 Phase 1 ships before P44 Phase 2 needs SQLite. Coupling P44 Phase 1 to a database migration delays P44 Phase 1 unnecessarily. Phase 1 here (JSONL) addresses commit noise without blocking anything.

### 8.3 Postgres from the start

**Description:** Skip SQLite, go straight to a hosted database.

**Why rejected:** Imposes infrastructure on every Kanbanzai consumer. Conflicts with the "works on a single laptop offline" requirement. Premature for a tool whose sharing model is currently Git pull/push.

### 8.4 Custom append-only file format

**Description:** Write a Kanbanzai-specific binary log for workflows and audit events.

**Why rejected:** Reinvents wheels. SQLite's WAL is already this format, with decades of production hardening, and is queryable.

### 8.5 Embed Bolt or another KV store

**Description:** Use a Go-native embedded KV store (BoltDB, BadgerDB) instead of SQLite.

**Why rejected:** No SQL means custom query layer for the audit log; no schema means custom migration tooling; no Postgres-compatible dialect means no future federation path. SQLite wins on every axis except raw write throughput, which is not Kanbanzai's bottleneck.

### 8.6 Two databases (one for execution, one for audit)

**Description:** Separate SQLite files for execution state and audit log, on the theory that audit grows fastest.

**Why rejected:** SQLite handles millions of rows in a single database without fuss; Kanbanzai's audit volume is tiny in absolute terms. Splitting introduces transactional boundaries that make multi-table queries painful.

---

## 9. Migration story for users

This is critical because Kanbanzai has consumers.

**No flag day.** Phase 1 (JSONL) and Phase 2 (SQLite) both run dual-write windows long enough for users to upgrade at their own pace.

**Backup before Phase 2.** The Phase 2 import tool refuses to run without a verified pre-migration backup.

**Rollback supported.** Phase 2.0–2.N keeps YAML files on disk; if SQLite causes a regression, the user flips a config flag and reads return to YAML.

**Inspectability preserved.** `kbz state inspect <id>` returns a YAML representation identical to what users grep today. `kbz state export` produces a bundle that a human can read.

**No skill changes.** Agents call the same MCP tools (`entity`, `doc`, `status`, etc.). The storage substrate is invisible to them.

**Documented breakage.** The single user-visible breakage is `grep .kbz/state/`-style scripts. We document this prominently in the Phase 2 release notes and provide `kbz state export` as the replacement.

---

## 10. Coupling with P44

P44 (the orchestration architecture journey) and P65 (this plan) are architecturally coupled but separately deliverable.

**P44 Phase 1 (chat-driven hardening)** has no dependency on this plan. It can ship over the existing YAML model.

**P44 Phase 2 (deterministic server)** has two hard dependencies on this plan:

1. **Durable workflow state.** P44 §5.2.5's "minimal in-tree durable-execution layer" is much smaller and more reliable if it persists state to SQLite rather than custom JSONL. Without SQLite, P44 Phase 2 either (a) accepts the JSONL custom format, paying the implementation and reliability cost, or (b) waits for this plan's Phase 2.

2. **Audit log queryability.** P44 §5.5 specifies queries the JSONL approach answers slowly and SQLite answers in milliseconds. The drift-detection job in particular is a SQL aggregation over weeks of events.

**Recommended sequencing:**

- P65 Phase 1 (JSONL transitions) ships in parallel with P44 Phase 1. They don't conflict.
- P65 Phase 2 (SQLite) starts after P65 Phase 1 stabilises and ideally lands before P44 Phase 2 begins. If P44 Phase 2 must start first, build P44's `internal/durable` against the same SQLite schema this plan defines, treating it as a one-component preview of P65 Phase 2.

**If only one plan ships:** P44 alone can succeed without P65 Phase 2 by accepting the JSONL custom format for durable workflow state. P65 Phase 2 alone delivers commit-noise improvement and queryable transition history but does not deliver the orchestration improvements. The two plans together compound.

---

## 11. Open Questions

These remain open and should be resolved during specification:

1. **SQLite driver choice.** `mattn/go-sqlite3` (cgo, mature) vs `modernc.org/sqlite` (pure Go, no cgo). Spike both during Phase 0.
2. **Document records: full migration or hybrid?** The strict reading of §4.1 puts document records in SQLite. An alternative keeps document records in `.kbz/state/documents/*.yaml` and only moves transition events. Resolve based on whether document approval becomes part of P44 Phase 2 controllers.
3. **Per-process or per-install database?** Single `state.db` per install is simpler. Per-worktree would allow worktree-isolated experimentation. Recommendation: per-install; worktrees use the same DB and rely on entity-scoped reads.
4. **Backup frequency policy.** Continuous (every commit), periodic (hourly), or on-demand (user-triggered)? Recommendation: on-demand + automatic before destructive operations (state migration, schema change).
5. **Schema versioning convention.** Recommendation: integer sequential, applied automatically on first write after upgrade, refusing to start if down-migration would lose data.
6. **What happens to `.kbz/audit/` (proposed by P44 §5.5)?** P44 specified JSONL for the audit log. This plan recommends SQLite from Phase 2 onwards. Resolve in P44 Phase 2 spec: ship JSONL audit in P44 Phase 1, migrate to SQLite during P65 Phase 2.
7. **Consumer install handling.** Existing installs need migration tooling on their next `kbz upgrade`. Define the upgrade UX explicitly.

---

## 12. Decisions

- **Decision 1:** Adopt SQLite as the canonical store for execution-tier state (entities, transitions, workflows, audit, document records). Keep Git-native storage for documents, knowledge, profiles, and code.
- **Decision 2:** Phase 1 ships the Git-native JSONL transition log unchanged from `design-transition-history-storage.md`. Phase 2 imports JSONL events into SQLite tables.
- **Decision 3:** Defer Postgres compatibility to Phase 3. Design SQL to be portable but do not pay portability cost in Phases 1–2.
- **Decision 4:** Dual-write windows on every authority shift. No flag days.
- **Decision 5:** Inspectability preserved via `kbz state inspect` and `kbz state export`, not via maintaining grep-able files.
- **Decision 6:** Coupling with P44 documented but not enforced — either plan can ship without the other; the combination compounds.

---

## 13. Cross-references

- **Companion plan:** P44 — Model routing and agent launcher (orchestration architecture journey)
- **Synthesises:** `work/_project/design-transition-history-storage.md`
- **Resolves P44 Open Question 1** (where does `internal/durable` persist state?): SQLite, in P65 Phase 2.
- **Provides substrate for P44 §5.5** (structured audit log).
- **Provides substrate for P44 §5.2.5** (durable execution layer).

---

*End of design.*
