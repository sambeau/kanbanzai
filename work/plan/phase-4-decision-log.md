# Phase 4 Decision Log

| Document | Phase 4 Decision Log |
|----------|----------------------|
| Status   | Active               |
| Created  | 2026-03-25           |
| Updated  | 2026-03-25           |
| Related  | `work/plan/phase-4-scope.md` |
|          | `work/design/workflow-design-basis.md` §14, §24 |
|          | `work/design/machine-context-design.md` §7, §8, §15 |
|          | `work/research/orchestration-landscape-2025.md` |
|          | `work/design/estimation-and-progress-design.md` |

---

## Decision Register

| ID         | Topic                           | Status   | Date       |
|------------|---------------------------------|----------|------------|
| P4-DES-001 | Phase split                     | accepted | 2026-03-25 |
| P4-DES-002 | Estimation assignment           | accepted | 2026-03-25 |
| P4-DES-003 | Self-management threshold       | accepted | 2026-03-25 |
| P4-DES-004 | Dependency modelling            | accepted | 2026-03-25 |
| P4-DES-005 | Agent delegation model          | accepted | 2026-03-25 |
| P4-DES-006 | Incidents and root cause analysis | accepted | 2026-03-25 |

---

## `P4-DES-001: Phase split`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4, product-scope, orchestration
- Related:
  - `work/plan/phase-4-scope.md` §3 (goal 6), §6, §7
  - P4-DES-003

### Decision

Phase 4 is split into **Phase 4a** (MVP orchestration, enables self-management) and **Phase 4b** (richer capabilities, self-managed).

**Phase 4a** delivers the minimum tooling for an orchestrating agent to manage a real workload: estimation, dependency enforcement, work queue, and the dispatch/complete loop.

**Phase 4b** builds decomposition tooling, worker review, conflict domain analysis, incident/RCA entities, and automatic dependency unblocking — developed inside the system using Phase 4a tools.

**Gate:** Phase 4b does not begin until Phase 4a is complete and has been validated on at least one real workload (see P4-DES-003).

### Rationale

The split point is chosen so the system bootstraps itself at the earliest possible moment. This is consistent with the design principle of using the tool on real work as soon as it is minimally viable. Building Phase 4a before using it for Phase 4b is the first concrete application of the self-management goal.

### Alternatives Considered

- **Single phase.** Rejected — delays self-management until all Phase 4 features are complete; misses the bootstrapping opportunity.
- **Different split point (e.g., include decomposition in 4a).** Rejected — decomposition is valuable but not required to manage a pre-decomposed backlog. The dispatch/complete loop is the critical gap; decomposition and review build on top of it.

### Consequences

- Phase 4a must be validated on real work before Phase 4b begins.
- Phase 4b is the first phase developed under self-management.
- Two separate specifications are required: one for Phase 4a now, one for Phase 4b after Phase 4a validation (written under self-management).

### Follow-up Needed

- Write the Phase 4a specification.
- After Phase 4a validation, write the Phase 4b specification using Phase 4a tooling.

---

## `P4-DES-002: Estimation assignment`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4, estimation, product-scope
- Related:
  - `work/design/estimation-and-progress-design.md`
  - `work/plan/phase-4-scope.md` §6.1

### Decision

Estimation is implemented as **part of Phase 4a** (or as pre-Phase 4a housekeeping, whichever comes first in the work queue).

The `estimation-and-progress-design.md` design is complete. Adding `estimate` as an optional field to existing entity schemas is purely additive and backwards-compatible — no migrations needed for existing records. New rollup query tools complete the feature.

### Rationale

A work queue without estimates is a list, not a plan. Estimation is a precondition for an orchestrating agent to reason about capacity and priority rather than just order. Deferring it to Phase 4b would mean managing Phase 4b under a queue that cannot be properly planned or sequenced.

### Alternatives Considered

- **Defer to Phase 4b.** Rejected — the work queue is less useful without sizing; the orchestrating agent cannot reason about capacity.
- **Skip estimation entirely.** Rejected — the design is complete and the implementation is purely additive. The cost is low and the value is immediate.

### Consequences

- `estimate` field added to Task, Feature, Bug, and Epic entity schemas.
- Computed rollup queries added: Feature effective estimate, Feature progress, Epic effective estimate, Epic progress.
- Soft limit warnings per entity type (not errors).
- `estimate_query` MCP tool and CLI commands (`kbz estimate set`, `kbz estimate show`).
- Reference example management for AI calibration consistency.

### Follow-up Needed

- Estimation must be complete before `work_queue` is finalised for capacity-aware scheduling.
- Open questions in `estimation-and-progress-design.md` §10 (missing estimates warning, re-estimate history, reference example storage) to be resolved in the Phase 4a specification.

---

## `P4-DES-003: Self-management threshold`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4, orchestration, human-ux, self-management
- Related:
  - `work/plan/phase-4-scope.md` §3 (goal 6), §10.1
  - P4-DES-001

### Decision

The system begins self-managing Phase 4b development **after Phase 4a is complete and has been validated on at least one real workload**.

**Required safeguards before full self-management:**

1. Human approval remains required for all feature-level transitions to `done`. Tasks may be agent-autonomous; features stay human-gated.
2. The existing merge gate system must be enforced strictly: no feature merges without a clean health check and all tasks `done`.
3. An explicit `human_checkpoint` mechanism must exist in Phase 4a — a way for an orchestrating agent to pause and surface a decision to a human without losing workflow state. This is a Phase 4a design requirement, not an optional enhancement.

**Maturity assessment at Phase 3 completion:** approximately 60–65% of self-management readiness. The system can track work, manage documents, flag health issues, and produce complete context packets. It cannot yet decompose work, enforce sequencing, dispatch sessions, or prevent concurrent task conflicts. Phase 4a closes these gaps.

### Rationale

Premature self-management would result in an orchestrating agent working around the system's gaps rather than through them, eroding trust in workflow state as the source of truth — the foundational design principle. The validation requirement ensures the orchestration loop is proven before it is used to manage its own further development.

Human gating on feature transitions to `done` ensures a human remains accountable for each meaningful unit of delivered work throughout Phase 4b, regardless of how autonomous task execution becomes.

### Alternatives Considered

- **Begin self-management immediately on Phase 4a completion, no validation required.** Rejected — an unvalidated orchestration loop managing its own development compounds any defects.
- **Never self-managing; humans always manage Phase 4b work directly.** Rejected — self-management is an explicit Phase 4 goal and the natural test of whether the system works.
- **No safeguards; full autonomy from Phase 4b onward.** Rejected — feature-level human gating is a minimal and low-friction accountability check that protects against silent quality degradation.

### Consequences

- `human_checkpoint` MCP tool is a required Phase 4a deliverable.
- Feature transitions to `done` remain human-gated throughout Phase 4b.
- An explicit validation workload must be run on Phase 4a before Phase 4b begins.
- Phase 4b specification is written under self-management using Phase 4a tooling.

### Follow-up Needed

- Define the `human_checkpoint` mechanism (stored state, resumption model) in the Phase 4a specification.
- Define what constitutes a satisfactory "validation on a real workload" before Phase 4b begins.

---

## `P4-DES-004: Dependency modelling`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4, orchestration, dependency-management
- Related:
  - `work/plan/phase-4-scope.md` §6.2, §7.2
  - `work/design/workflow-design-basis.md` §9.4

### Decision

Dependency enforcement is implemented in two stages:

**Phase 4a — enforcement with manual polling:**
- The transition validator blocks tasks from entering `ready` status if any `depends_on` entry is not yet in a terminal state (`done`, `not-planned`, or `duplicate`).
- `work_queue` MCP tool returns tasks in `ready` status with all dependencies met.
- `dependency_status` MCP tool shows all dependencies and their current status for a given task.
- The orchestrator polls `work_queue` between assignments; newly unblocked tasks appear naturally in the next result.

**Phase 4b — automatic unblocking:**
- When a task reaches `done`, the `StatusTransitionHook` checks all tasks whose `depends_on` includes this task. Any now fully unblocked are transitioned automatically from `blocked` to `ready`.
- The `complete_task` response lists newly unblocked tasks.

Cross-feature dependencies are supported: the existing `depends_on: []string` field accepts any task ID, and enforcement does not distinguish between intra- and inter-feature dependencies.

### Rationale

Enforcement must precede the work queue. A work queue built on un-enforced dependencies would surface false-ready tasks and produce incorrect agent schedules. The two-stage model validates the manual polling approach before adding automation, keeping Phase 4a minimal and verifiable.

Automatic unblocking is deferred to Phase 4b because it requires the `StatusTransitionHook` pattern and is a pure quality-of-life improvement once the enforced model is proven.

### Alternatives Considered

- **Enforcement at dispatch time only (in `dispatch_task`).** Rejected — enforcement would not prevent manual status transitions that bypass dispatch; state machine integrity requires enforcement at the transition layer.
- **Automatic unblocking from the start (Phase 4a).** Rejected — adds complexity before the manual model is validated; the hook mechanism exists but automatic side effects on task transitions warrant careful testing.
- **No enforcement; advisory only.** Rejected — an unenforced work queue is a suggestion, not a plan.

### Consequences

- The `queued → ready` transition is dependency-gated: it requires all `depends_on` entries to be in terminal state.
- A new health check category covers dependency cycles (tasks whose `depends_on` chains form a cycle).
- Phase 4b extends the existing `StatusTransitionHook` (used for automatic worktree creation) for automatic dependency unblocking.

### Follow-up Needed

- Define the exact enforcement point in the transition validator in the Phase 4a specification.
- Define cycle detection behaviour: cycles are a health check concern; the spec should clarify whether a cycle is also refused at transition time.

---

## `P4-DES-005: Agent delegation model`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4, orchestration, agent-delegation
- Related:
  - `work/plan/phase-4-scope.md` §6.3
  - `work/design/workflow-design-basis.md` §14
  - `work/design/machine-context-design.md` §7

### Decision

Delegation is modelled as **task state**, not as a separate entity type. The existing task state machine already provides most of what is needed; Phase 4a adds four fields and one new tool.

**Fields added to Task:**
- `claimed_at` (timestamp) — set when the task transitions to `active`. Provides the atomic claiming guarantee: only one agent can win the `ready → active` transition for a given task.
- `dispatched_to` (role profile ID) — the role profile the orchestrator assigned this task to.
- `dispatched_at` (timestamp) — when the orchestrator dispatched the task.
- `dispatched_by` (identity string) — which orchestrating agent dispatched the task.

**`dispatch_task` operation:** atomically validates the task is in `ready` status → checks all dependencies met → transitions to `active` (setting `claimed_at`) → records `dispatched_to/at/by` → calls `context_assemble` → returns the context packet. If the task is already `active`, the operation fails with a clear refusal. This prevents two orchestrating agents from dispatching the same task.

No Delegation entity is created. The task itself is the delegation record.

### Rationale

A separate Delegation entity would model something already captured in task state. Keeping delegation in the task avoids a new entity type, new storage format, and new query surface without adding expressiveness. The four-tier hierarchy (human → orchestrator → worker → system) is a behavioural convention reinforced by the tools; it does not require structural representation beyond what already exists.

The `claimed_at` timestamp provides the atomic claiming guarantee without requiring distributed locks: in a single-process MCP server, the `ready → active` transition is serialised, and any concurrent dispatch attempt on the same task is refused by the transition validator.

### Alternatives Considered

- **Separate Delegation entity.** Rejected — adds a new entity type, storage schema, ID prefix, and query surface to model something already expressed in task state. No expressiveness gain.
- **Agent-managed external state.** Rejected — coordination state must live in the system to be queryable, auditable, and health-checked. External state is invisible to the workflow.

### Consequences

- Four new fields on the Task schema: `claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`.
- `dispatch_task` MCP tool atomically claims and dispatches a task, returning a context packet.
- `complete_task` MCP tool closes the dispatch loop, accepting knowledge contribution entries.
- The task record serves as the full dispatch audit trail.

### Follow-up Needed

- Define atomicity implementation for `dispatch_task` in the Phase 4a specification (single-process serialisation, file-level locking, or optimistic locking).
- Define what `complete_task` accepts as knowledge contribution entries.

---

## `P4-DES-006: Incidents and root cause analysis`

- Status: accepted
- Date: 2026-03-25
- Scope: phase-4b, incidents, rca
- Related:
  - `work/plan/phase-4-scope.md` §7.6
  - `work/research/orchestration-landscape-2025.md`

### Decision

**Incident** is a first-class entity with a heavyweight model. **RootCauseAnalysis** is a document type. Both are implemented in **Phase 4b**.

**Incident lifecycle:**
```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
```

**Incident fields include:** `severity`, `detected_at`, `mitigated_at`, `resolved_at` (for MTTR measurement), `affected_features`, `linked_bugs`, `linked_rca`.

**RCA as document type:** reviewed and approved via the existing document pipeline. On approval, key findings are contributed to the knowledge store as Tier 2 entries with scope `project`. RCA is linked to one or more Incidents via an `incident_ids` field.

### Rationale

Incidents have different stakeholders, different timestamp semantics (detected-at, escalated-at, mitigated-at, resolved-at for MTTR measurement), and a one-to-many relationship with bugs (multiple symptoms from one incident). A Bug subtype cannot cleanly represent these without polluting the Bug schema with many optional fields that normal bugs never use.

RCA maps cleanly onto the existing document lifecycle: it is reviewed, can be approved or rejected, may supersede a previous RCA, and its approved content feeds the knowledge store. Modelling RCA as a document rather than an entity keeps the entity model focused on workflow state, not prose.

Phase 4b placement is appropriate: incidents and RCA are not needed for the Phase 4a orchestration MVP and do not block the self-management threshold. They add the most value once the system is managing its own development and production-significant failures are worth tracking through the same workflow.

### Alternatives Considered

- **Extend the Bug entity.** Rejected — the timestamp semantics and one-to-many bug relationship require too many Bug-specific optional fields, degrading schema clarity for the common case.
- **Document-only model (no Incident entity).** Rejected — MTTR measurement and health checks (open incidents without a linked RCA) require queryable structured state, not just prose.
- **Phase 4a placement.** Rejected — incidents and RCA are not needed for the orchestration MVP. Moving them to 4a adds scope without contributing to the self-management threshold.

### Consequences

- New `Incident` entity type with its own ID prefix, schema, and lifecycle in Phase 4b.
- New `RootCauseAnalysis` document type integrated with the existing document pipeline.
- RCA approval triggers knowledge contribution at Tier 2.
- New health check: open incidents without a linked RCA after a configurable post-resolution threshold.
- MCP tools: `incident_create`, `incident_update`, `incident_list`, `incident_link_bug`.
- CLI commands: `kbz incident create`, `kbz incident list`, `kbz incident show <id>`.

### Follow-up Needed

- Define the Incident ID prefix in the Phase 4b specification.
- Define MTTR reporting requirements (which timestamps are mandatory vs. optional).
- Define the RCA knowledge contribution schema (which fields are extracted as Tier 2 entries).

---

## Summary

Phase 4 has six accepted design decisions covering product scope, orchestration model, and human UX concerns:

| ID         | Topic                      | Key Choice                                                   |
|------------|----------------------------|--------------------------------------------------------------|
| P4-DES-001 | Phase split                | 4a = MVP orchestration; 4b = richer capabilities, self-managed |
| P4-DES-002 | Estimation assignment      | Phase 4a (precondition for a useful work queue)              |
| P4-DES-003 | Self-management threshold  | After 4a validated; three safeguards; human_checkpoint required |
| P4-DES-004 | Dependency modelling       | Transition-validator enforcement (4a); automatic hook (4b)   |
| P4-DES-005 | Agent delegation model     | Task state + four fields + dispatch_task tool; no new entity |
| P4-DES-006 | Incidents and RCA          | Incident entity + RCA document type; both Phase 4b           |

Implementation details and AI Agent UX decisions are deferred to the Phase 4a specification.