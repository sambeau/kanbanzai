# Phase 4 Scope and Planning

| Document | Phase 4 Scope and Planning |
|----------|---------------------------|
| Status   | Draft                     |
| Created  | 2026-03-25                |
| Updated  | 2026-03-25                |
| Related  | `work/design/workflow-design-basis.md` §14, §24 |
|          | `work/design/machine-context-design.md` §7, §8, §15 |
|          | `work/research/orchestration-landscape-2025.md` |
|          | `work/design/estimation-and-progress-design.md` |
|          | `work/plan/phase-3-scope.md` |

---

## 1. Purpose

This document defines the scope, approved design decisions, and planning structure for Phase 4: Orchestration.

Phase 4 closes the loop that Phases 1–3 have been building toward. Phase 1 created the workflow kernel. Phase 2 added document intelligence and context management. Phase 3 added git-native collaboration and knowledge lifecycle automation. Phase 4 adds the tools that allow an orchestrating agent to decompose work, manage a queue, dispatch execution agents with properly assembled context, and collect results — completing the four-tier delegation model described in `work/design/workflow-design-basis.md` §14.

The design goal is not to build a general-purpose orchestration framework. It is to add the small set of tools that, together with everything already built, allow an AI agent to act as a capable PM/orchestrator over a kanbanzai-tracked project — and ultimately for the system to manage substantial parts of its own continued development.

---

## 2. Background: The Orchestration Landscape Review

Before committing to Phase 4 scope, a structured review of the 2025 AI agent orchestration landscape was conducted (`work/research/orchestration-landscape-2025.md`). Key findings that directly shape this scope:

1. **No existing tool replicates the kanbanzai combination.** Task tracking, document intelligence, tiered knowledge lifecycle, byte-budgeted context assembly, and git worktree management do not exist together in any open-source tool. The closest competitors (kagan for dispatch + worktrees; Agent-MCP for task + knowledge RAG) each cover roughly half the surface, without the integrating context pipeline that makes dispatch genuinely useful.

2. **kanbanzai should build its own orchestration layer.** The context injection pipeline is load-bearing. External orchestration frameworks could call `context_assemble`, but they would be callers of kanbanzai's work, not replacements for it. Incorporating an external framework would add dependencies and incompatible assumptions about state management and agent identity.

3. **Five patterns validated by the field should be adopted rather than rediscovered** (see §4.2).

4. **Phase 4 implementation is smaller than it appears.** The heavy lifting is done. Phase 4 adds approximately five new tools, a few entity fields, and one validation rule addition. The context assembly, knowledge lifecycle, document intelligence, and git worktrees are already built.

5. **A web/desktop UI for designers and managers is a genuine future need**, informed by kagan's TUI/dashboard. This is not Phase 4 — orchestration must come first — but it is flagged now as a Phase 5 candidate and Phase 4's API surface should be designed with it in mind.

---

## 3. Phase 4 Goals

1. **Enable orchestrated agent work** — An orchestrating agent can query what work is ready, claim and dispatch individual tasks to execution agents with complete context packets, collect results, and advance workflow state — all through the existing MCP interface.

2. **Enforce dependency-aware sequencing** — Tasks that depend on incomplete work cannot be started. The system surfaces a ready queue of dependency-unblocked tasks.

3. **Complete the estimation layer** — The `estimation-and-progress-design.md` design is fully implemented, enabling backlog sizing, rollup progress tracking, and AI calibration against reference examples.

4. **Support feature decomposition** — An orchestrating agent can break a feature-with-specification into a task list using the document intelligence pipeline as input.

5. **Enable worker review** — An execution agent's output can be checked against the feature specification before the feature transitions to complete.

6. **Reach the self-management threshold** — After Phase 4a is complete and validated on at least one real workload, the system manages Phase 4b's own development through the same process it defines.

7. **Model incidents and root cause analyses** — Production-significant failures are tracked as first-class entities with their own lifecycle, linked to the bug and knowledge systems.

---

## 4. Design Decisions

The following decisions were made during Phase 4 scope planning. Human UX and strategic decisions are binding. Implementation and AI Agent UX decisions may be refined during specification.

### P4-DES-001: Phase Split

**Decision:** Phase 4 is split into Phase 4a (MVP orchestration, enables self-management) and Phase 4b (richer capabilities, self-managed).

**Phase 4a** delivers the minimum tooling for an orchestrating agent to manage a real workload inside kanbanzai: estimation, dependency enforcement, work queue, and the dispatch/complete loop. The test of Phase 4a is: can we use it to manage Phase 4b development?

**Phase 4b** builds on that foundation: decomposition tooling, worker review, conflict domain analysis, incident/RCA entities, and vertical slice guidance. Phase 4b is developed inside the system using the Phase 4a tooling.

**Rationale:** The split point is chosen so that the system bootstraps itself at the earliest possible moment. Building Phase 4a before using it for Phase 4b is consistent with the design principle of eating your own cooking as soon as it is edible.

---

### P4-DES-002: Estimation Assignment

**Decision:** Estimation is implemented as part of Phase 4a (or as pre-Phase 4a housekeeping, whichever comes first in the work queue).

The `estimation-and-progress-design.md` design is complete and the system is ready to receive it. Adding `estimate` as an optional field to existing entity schemas is purely additive and backwards-compatible — no migrations needed for existing records. New query tools for rollup complete the feature. It is a precondition for a useful work queue — without sizing, the queue is a list, not a plan.

**Rationale:** A backlog without estimates cannot be properly planned or sequenced. Estimation enables the orchestrating agent to reason about capacity and priority, not just order.

---

### P4-DES-003: Self-Management Threshold

**Decision:** The system begins self-managing Phase 4b development after Phase 4a is complete and has been validated on at least one real workload.

**Required safeguards before full self-management:**
1. Human approval remains required for all feature-level transitions to `done`. Tasks may be agent-autonomous; features stay human-gated.
2. The existing merge gate system must be enforced strictly: no feature merges without a clean health check and all tasks `done`.
3. An explicit human checkpoint mechanism must exist in Phase 4a — a way for an orchestrating agent to pause and surface a decision to a human without losing workflow state. This is a Phase 4a design requirement.

**Current maturity assessment (as of Phase 3 completion):** The system is at approximately 60–65% of self-management readiness. It can track work, manage documents, flag health issues, and produce complete context packets. It cannot yet decompose work, enforce sequencing, dispatch sessions, or prevent concurrent conflicts on the same task. These gaps are exactly what Phase 4a closes.

**Rationale:** Premature self-management would result in an orchestrating agent working around the system's gaps rather than through them, eroding trust in workflow state as the source of truth — the foundational design principle.

---

### P4-DES-004: Dependency Modelling

**Decision:** Dependency enforcement is added to the transition validator. Tasks cannot transition to `active` if any entry in `depends_on` is not yet `done` (or `not-planned` / `duplicate`).

**Phase 4a:** Enforcement + `work_queue` tool (returns tasks in `ready` status with all dependencies met). Manual unblocking: the orchestrator polls the work queue between assignments. When a dependency task completes, its dependants simply appear in the next `work_queue` result.

**Phase 4b:** Automatic unblocking via the `StatusTransitionHook` pattern (same mechanism used for automatic worktree creation in Phase 3). When a task reaches `done`, dependent tasks are automatically transitioned from `blocked` to `ready`. This is deferred to Phase 4b to validate the manual model first.

**Cross-feature dependencies:** The existing `depends_on: []string` field accepts any task ID, including tasks in other features. Enforcement does not distinguish between intra- and inter-feature dependencies — the dependency is real regardless of feature boundaries.

**Rationale:** Enforcement must precede the work queue. A work queue built on un-enforced dependencies would surface false-ready tasks and produce incorrect agent schedules.

---

### P4-DES-005: Agent Delegation Model

**Decision:** Delegation is modelled as task state, not as a separate entity type. The existing task state machine already provides most of what is needed; Phase 4a adds four fields and one new tool.

**Fields added to Task:**
- `claimed_at` (timestamp) — set when the task transitions to `active`. Provides the atomic claiming guarantee: only one agent can win the `ready → active` transition for a given task.
- `dispatched_to` (role profile ID, e.g. `backend`) — the role the orchestrator assigned this task to.
- `dispatched_at` (timestamp) — when the orchestrator dispatched the task.
- `dispatched_by` (identity string) — which orchestrating agent dispatched the task.

**The `dispatch_task` operation** atomically performs: validate task is in `ready` status → check all dependencies met → transition to `active` (setting `claimed_at`) → record `dispatched_to/at/by` → call `context_assemble` → return the context packet. If the task is already `active`, the operation fails with a clear refusal. This prevents two orchestrating agents from dispatching the same task.

**No Delegation entity is created.** The task itself is the delegation record. The `dispatched_to/at/by` fields provide the audit trail.

**Rationale:** A separate Delegation entity would model something that is already captured in task state. Keeping delegation in the task avoids a new entity type, new storage format, and new query surface without adding expressiveness. The four-tier hierarchy is a behavioural convention reinforced by the tools; it does not require a structural representation beyond what already exists.

---

### P4-DES-006: Incidents and Root Cause Analysis

**Decision:** Incident as a first-class entity (heavyweight model); RootCauseAnalysis as a document type. Both implemented in Phase 4b.

**Incident lifecycle:**
```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
```

**Why the heavyweight model:** Incidents have different stakeholders, different timestamp semantics (detected-at, escalated-at, mitigated-at, resolved-at for MTTR measurement), and a one-to-many relationship with bugs (multiple symptoms from one incident). A bug type cannot cleanly represent these without polluting the bug schema with many optional fields that normal bugs don't use.

**RCA as a document type:** RCA is reviewed, approved, and may supersede previous RCAs. This maps cleanly onto the existing `DocumentService` + document lifecycle. On approval, RCA key findings are contributed to the knowledge store as Tier 2 entries. RCA is linked to one or more Incidents via an `incident_ids` field.

**Phase placement:** Phase 4b. Incidents and RCA are not needed for the Phase 4a orchestration MVP and do not block the self-management threshold. They add value once the system is managing itself and production incidents are worth tracking through the same workflow.

---

## 5. Decisions Deferred to Specification and Development

### 5.1 Implementation decisions (developer-resolvable during specification)

| Decision | Notes |
|---|---|
| `work_queue` filtering and sorting | Which fields drive ordering — estimate, priority, dependency depth, age? |
| `dispatch_task` atomicity implementation | File-level locking, optimistic locking, or rely on single-process serialisation |
| `complete_task` knowledge contribution schema | What fields does the agent provide for knowledge contribution on task completion? |
| Estimation rollup query caching | Rollups are computed on read; whether to cache them in the derived cache is an implementation choice |
| `human_checkpoint` mechanism design | What does pausing an orchestrating agent look like in terms of stored state and resumption? |
| Decomposition output format | Does `decompose_feature` produce a preview for human confirmation or write tasks directly? |
| Worker review pass/fail criteria | Which gate failures trigger a review cycle versus a fail-and-escalate? |

### 5.2 AI Agent UX decisions (best resolved by an agent during specification)

| Decision | Notes |
|---|---|
| `dispatch_task` context packet format | Should the packet include an orchestration summary (who assigned this, why, what came before)? |
| `work_queue` response shape | What information does an orchestrating agent need per task to make a good dispatch decision? Effort to include estimate, dependency count, age, profile match. |
| `complete_task` feedback fields | What should an executing agent be expected to report on completion beyond knowledge entries? (Blockers encountered? Files modified? Verification performed?) |
| `decompose_feature` heuristics | What signals from the spec document guide decomposition? How does the agent decide on task granularity and vertical slice boundaries? |
| Estimation calibration reference set | How are reference examples stored, retrieved, and presented at estimation time? The `estimation-and-progress-design.md` §8 outlines the requirement but not the implementation. |
| Trimming visibility format | What should the `trimmed` field in `context_assemble` look like so an executing agent can make informed decisions about what to request additionally? |

---

## 6. Phase 4a Feature Breakdown

Phase 4a is the minimum viable orchestration layer. It delivers the tools that close the dispatch loop and satisfy the self-management threshold.

### 6.1 Estimation

Implement the complete `estimation-and-progress-design.md` design.

- Add optional `estimate` field to Task, Feature, Bug, and Epic entities
- Implement computed rollup queries: Feature effective estimate, Feature progress, Epic effective estimate, Epic progress
- Soft limit warnings per entity type (not errors)
- `estimate_query` MCP tool: returns entity with rollup totals and delta from original estimate
- Estimation operation: presents scale definitions and reference examples; records estimate
- Reference example management: add/remove calibration anchors for AI consistency
- CLI commands: `kbz estimate set`, `kbz estimate show`

### 6.2 Dependency Enforcement

- Add enforcement to `validate.ValidateTransition`: tasks cannot transition to `active` if any `depends_on` task is not in a terminal state (`done`, `not-planned`, `duplicate`)
- `work_queue` MCP tool: returns all tasks in `ready` status with all dependencies met, ordered by priority/estimate/age. Optionally filtered by role profile.
- `dependency_status` MCP tool: for a given task, shows all dependencies and their current status — which are blocking and which are resolved
- CLI commands: `kbz queue`, `kbz task deps <id>`

### 6.3 Task Dispatch and Completion

The core orchestration loop.

- Add `claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by` fields to Task schema
- `dispatch_task` MCP tool: atomically claims task (ready → active), records dispatch fields, assembles and returns context packet. Fails clearly if task is not ready or already claimed.
- `complete_task` MCP tool: transitions task to `needs-review` or `done` (depending on workflow), accepts knowledge contribution entries, records completion summary. Triggers knowledge contribution via the existing `knowledge_contribute` pipeline.
- `human_checkpoint` MCP tool: orchestrating agent pauses, records a structured question or decision point in workflow state, and halts dispatch until a human responds via `human_checkpoint_respond`. Preserves orchestration context so work can resume without re-reading the entire backlog.
- CLI commands: `kbz dispatch <task-id>`, `kbz task complete <task-id>`

### 6.4 Context Assembly Enhancements

- Add `trimmed` field to `context_assemble` response: lists entries that were cut (entry ID, type, priority, size in bytes) so executing agents know what context is missing and can request it
- Add `orchestration_context` optional field to `context_assemble` input: allows the dispatching orchestrator to inject a brief handoff note (what was done before, what the next agent needs to know) as a Tier 3 session knowledge item with directed scope

### 6.5 Health Check Extensions

- New health check category: dependency cycles (tasks whose `depends_on` chains form a cycle)
- New health check category: stalled dispatches (`active` tasks with `dispatched_at` older than a configurable threshold with no recent git activity)
- New health check category: estimation coverage (features in `active` or later status with no estimates on child tasks)

---

## 7. Phase 4b Feature Breakdown

Phase 4b builds on the Phase 4a foundation. It is developed inside the system using the Phase 4a orchestration tools — validating self-management in practice.

### 7.1 Feature Decomposition

- `decompose_feature` MCP tool: given a Feature with a specification document, produces a proposed task list following vertical slice principles. Output is a preview requiring human (or orchestrator-with-checkpoint) confirmation before tasks are written.
- Decomposition guidance embedded in the tool: vertical slice principles, task size soft limits, dependency identification heuristics, role assignment suggestions based on available profiles
- `decompose_review` MCP tool: reviews a proposed decomposition against the spec and identifies gaps, overlaps, or tasks that are too large
- CLI commands: `kbz feature decompose <id>`

### 7.2 Automatic Dependency Unblocking

- When a task transitions to `done`, the `StatusTransitionHook` checks all tasks whose `depends_on` includes this task. Any that are now fully unblocked (all dependencies in terminal state) are transitioned from `blocked` to `ready` automatically.
- This replaces the Phase 4a manual polling model. The orchestrating agent receives a notification in the `complete_task` response listing newly unblocked tasks.

### 7.3 Worker Review

- `review_task_output` MCP tool: given a task ID and optional reference to the agent's output artefacts, checks the output against: (1) task acceptance criteria, (2) the parent feature's specification document sections relevant to this task, (3) the verification requirements recorded in the task. Returns a structured pass/fail result with actionable failure descriptions.
- Integration with the document intelligence pipeline: section tracing via `doc_trace` ensures the review checks against the actual spec content, not just the task summary.
- A failed review transitions the task to `needs-rework` and records the failure reason. A passed review transitions to `needs-review` for human sign-off.

### 7.4 Conflict Domain Analysis

- `conflict_domain_check` MCP tool: given two or more tasks proposed for parallel execution, assesses whether they are likely to produce merge conflicts. Analysis dimensions: file overlap (via git history and planned files), dependency ordering, architectural boundary crossing.
- Exposed in `work_queue` as an optional annotation: tasks marked as conflict-risky with respect to currently active tasks are flagged.
- CLI commands: `kbz queue --conflict-check`

### 7.5 Vertical Slice Decomposition Guidance

- Guidance integrated into `decompose_feature` (see §7.1)
- `slice_analysis` MCP tool: given a feature, identifies how it could be decomposed into vertical slices — end-to-end thin cuts through the stack, each independently deployable/testable. Surfaces the analysis as candidate task groupings.
- Knowledge entries contributed post-merge are tagged with their slice origin to support future conflict domain analysis.

### 7.6 Incidents and Root Cause Analysis

Per P4-DES-006:

- `Incident` entity type with lifecycle: `reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed`
- Fields: `severity`, `detected_at`, `mitigated_at`, `resolved_at`, `affected_features`, `linked_bugs`, `linked_rca`
- `RootCauseAnalysis` document type: owned by an Incident. Reviewed and approved via the existing document pipeline. On approval, key findings contributed to the knowledge store as Tier 2 entries with scope `project`.
- Health check: open incidents without a linked RCA after configurable threshold post-resolution.
- MCP tools: `incident_create`, `incident_update`, `incident_list`, `incident_link_bug`
- CLI commands: `kbz incident create`, `kbz incident list`, `kbz incident show <id>`

---

## 8. New Items from the Orchestration Landscape Review

The following items were identified in `work/research/orchestration-landscape-2025.md` and are not present in the original Phase 4 design. They have been incorporated into the feature breakdown above.

| Item | Phase | Source |
|---|---|---|
| `work_queue` tool — tasks ready for dispatch | 4a | Landscape review: core gap |
| `dispatch_task` tool — atomic claim + context + dispatch record | 4a | Landscape review: atomic claiming pattern |
| `complete_task` tool — closes the dispatch loop | 4a | Landscape review: loop closure |
| `human_checkpoint` tool — structured pause for human decision | 4a | Self-management safeguard (P4-DES-003) |
| `claimed_at` / `dispatched_to` / `dispatched_at` / `dispatched_by` on Task | 4a | Delegation model (P4-DES-005) |
| Trimming visibility in `context_assemble` | 4a | Landscape review: budget visibility pattern |
| Dependency enforcement in transition validator | 4a | Dependency modelling (P4-DES-004) |
| Estimation tools | 4a | Ready-to-implement design, previously unassigned |
| Automatic dependency unblocking via StatusTransitionHook | 4b | Dependency modelling (P4-DES-004) |
| Incident entity + RCA document type | 4b | Landscape review + human decision log |
| Web/desktop UI for designers and managers | Phase 5 | kagan comparison (landscape review §8) |

---

## 9. Dependencies

### Internal dependencies (within Phase 4)

- Estimation (§6.1) must be complete before `work_queue` is meaningful for capacity-aware scheduling
- Dependency enforcement (§6.2) must be complete before `dispatch_task` (§6.3) — dispatching to a blocked task would be incorrect
- Phase 4a must be complete and validated before Phase 4b work begins under self-management

### External dependencies (prior phases)

- Phase 3 `StatusTransitionHook` — used for automatic worktree creation; will be extended in Phase 4b for automatic dependency unblocking. Already implemented.
- Phase 2b `context_assemble` — the core of `dispatch_task`'s context packet. Already implemented; §6.4 adds trimming visibility.
- Phase 2b knowledge contribution pipeline — used by `complete_task`. Already implemented.
- Phase 3 merge gates — used by worker review (§7.3) to confirm branch health as part of output review. Already implemented.
- Phase 2a document intelligence — used by `decompose_feature` and `review_task_output`. Already implemented.

### External tool dependencies

None introduced. Phase 4 is built entirely on existing kanbanzai infrastructure and the existing MCP protocol. No new runtime dependencies are required.

---

## 10. Risks

### 10.1 Self-management transition risk

**Risk:** Switching to self-managed development for Phase 4b before the tools are mature enough creates a brittle bootstrap loop — the system is managing its own development with tools that don't yet work reliably.

**Mitigation:** Human approval on all feature transitions to `done` throughout Phase 4b. Explicit `human_checkpoint` tool allows the orchestrating agent to escalate decisions. The merge gate system prevents incomplete work from being marked done.

### 10.2 Estimation calibration cold start

**Risk:** AI agents estimating tasks without a calibrated reference set produce unreliable estimates that compound through rollups, making the work queue misleading.

**Mitigation:** The first Phase 4a tasks should be estimated by a human and designated as calibration anchors. The estimation design (§8) addresses this explicitly. Estimates are treated as provisional until a human confirms them during the early period.

### 10.3 Decomposition quality

**Risk:** `decompose_feature` produces task lists that are too coarse, too fine, or miss dependencies — undermining the value of the work queue.

**Mitigation:** Decomposition output is a preview requiring confirmation before tasks are written (§7.1). `decompose_review` provides a second pass. The Phase 4b `worker_review` tool provides feedback when decomposition produces tasks that do not satisfy the specification.

### 10.4 Parallel agent conflicts

**Risk:** Multiple agents claiming tasks simultaneously leads to conflicts, duplicated work, or corrupted worktree state.

**Mitigation:** `dispatch_task` atomicity (P4-DES-005) ensures only one agent can claim a given task. The existing one-worktree-per-feature model (P3-DES-003) ensures parallel agents on different features don't share worktree state. Conflict domain analysis (§7.4) flags tasks that are risky to parallelise.

### 10.5 Context packet quality degradation

**Risk:** As the knowledge store grows, Tier 3 trimming becomes more aggressive, and dispatched agents receive progressively less context. Quality degrades silently.

**Mitigation:** The `trimmed` field in `context_assemble` responses (§6.4) makes trimming visible. Health check coverage for stale Tier 3 entries and the existing knowledge pruning tools prevent accumulation. Context rot detection is a design-time concern and should be monitored from Phase 4a onward.

---

## 11. Future Considerations

### Phase 5: Web/Desktop UI for Designers and Managers

The kagan comparison (landscape review §8) identifies a real gap: visibility for non-technical stakeholders. Designers tracking feature progress, managers reviewing sprint health, and stakeholders reading design documents need a read-oriented interface that does not require CLI or MCP access.

Candidate capabilities for a future UI track:
- Entity status and lifecycle progress (Epic → Feature → Task dashboard)
- Document access and reading (design docs, specs, dev plans)
- Worktree and branch health visualisation
- Estimation and progress metrics
- Knowledge and decision browser scoped to a feature
- Incident status board

Phase 4 API surface should be designed with this in mind. Avoid coupling query responses to agent-specific conventions that a human dashboard would not need. The underlying data model (YAML files, the derived cache, MCP tools) provides a stable substrate for a future UI layer.

### Possible future phases

- GitLab, Bitbucket, or other platform support (deferred from Phase 3)
- Cross-project knowledge sharing (explicitly deferred)
- Embedding-based semantic similarity for deduplication (deferred from Phase 3 compaction)
- Webhook-based real-time sync
- Automatic context assembly optimisation (learning which context entries are most useful per task type)
- Confidence score time decay for knowledge entries unused over extended periods

---

## 12. Next Steps

1. **Write the Phase 4 decision log** — Record P4-DES-001 through P4-DES-006 in `work/plan/phase-4-decision-log.md` following the Phase 3 format.
2. **Implement estimation** — `work/design/estimation-and-progress-design.md` is complete. Implement as pre-Phase 4a housekeeping or the first Phase 4a task.
3. **Write the Phase 4a specification** — Detailed acceptance criteria for §6.1–§6.5. Use the Phase 3 spec as structural template.
4. **Write the Phase 4a implementation plan** — Break §6.1–§6.5 into implementation tracks with dependencies and sizing.
5. **Validate Phase 4a** — Complete at least one real workload (e.g. Phase 4b task decomposition) using the Phase 4a tools before handing development to self-management.
6. **Write Phase 4b specification** — After Phase 4a validation, write the Phase 4b spec under self-management using Phase 4a tooling.

---

## 13. Summary

Phase 4 completes the kanbanzai system by adding the tools that close the orchestration loop: estimation, dependency enforcement, work queue, and atomic dispatch. The heavy lifting — context assembly, knowledge lifecycle, document intelligence, git worktrees, merge gates — is already done in Phases 1–3.

Phase 4a delivers the minimum tooling to enable a capable orchestrating agent, and sets the self-management threshold: after validation, Phase 4b is developed inside the system using Phase 4a tools. Phase 4b adds richer decomposition, worker review, conflict analysis, and incident tracking on top of the Phase 4a foundation.

The architecture is clear: kanbanzai provides the knowledge and context; the orchestrator is Claude (or any model) calling kanbanzai's MCP tools in a loop. Phase 4 adds the tools that make that loop complete.

| Phase | Core deliverables | Gate |
|---|---|---|
| Pre-4a | Estimation implementation | Estimation complete and validated |
| 4a | Dependency enforcement, work queue, dispatch/complete loop, context assembly enhancements | Phase 4a validated on one real workload |
| 4b | Decomposition, worker review, conflict analysis, automatic unblocking, incidents/RCA | Self-managed under Phase 4a tooling |
| Phase 5 | Web/desktop UI | API surface stable from Phase 4 |