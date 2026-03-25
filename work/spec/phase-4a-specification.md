# Phase 4a Specification: Orchestration MVP

| Document | Phase 4a Specification: Orchestration MVP          |
|----------|----------------------------------------------------|
| Status   | Draft                                              |
| Created  | 2026-03-25                                         |
| Updated  | 2026-03-25                                         |
| Related  | `work/plan/phase-4-scope.md`                       |
|          | `work/plan/phase-4-decision-log.md`                |
|          | `work/design/workflow-design-basis.md` §14, §24    |
|          | `work/design/machine-context-design.md` §7, §8     |
|          | `work/design/estimation-and-progress-design.md`    |
|          | `work/spec/phase-3-specification.md`               |

---

## 1. Purpose

This specification defines the requirements for Phase 4a of the Kanbanzai workflow system. Phase 4a delivers the minimum viable orchestration layer — the tools that close the dispatch loop and satisfy the self-management threshold defined in P4-DES-003.

Phase 4a builds on:

- **Phase 1** — workflow kernel with entities, validation, and MCP interface
- **Phase 2a** — document intelligence and entity model evolution
- **Phase 2b** — context profiles, knowledge contribution, and confidence scoring
- **Phase 3** — git worktrees, merge gates, GitHub PR integration, and knowledge lifecycle automation

After Phase 4a is complete and validated on at least one real workload, Phase 4b development begins under self-management using Phase 4a tools.

---

## 2. Goals

1. **Enable orchestrated agent work** — An orchestrating agent can query what work is ready, claim and dispatch individual tasks to execution agents with complete context packets, collect results, and advance workflow state — all through the existing MCP interface.

2. **Enforce dependency-aware sequencing** — Tasks that depend on incomplete work cannot be dispatched. The system surfaces a ready queue of dependency-unblocked tasks.

3. **Complete the estimation layer** — The `estimation-and-progress-design.md` design is fully implemented, enabling backlog sizing, rollup progress tracking, and AI calibration against reference examples.

4. **Enable structured human escalation** — An orchestrating agent can pause, record a structured decision point, and wait for human input without losing orchestration context.

5. **Make context trimming visible** — Executing agents know what context was cut from their packet and can request missing entries explicitly.

6. **Reach the self-management threshold** — After Phase 4a is validated, the system manages Phase 4b's own development through the same process it defines.

---

## 3. Scope

### 3.1 In scope for Phase 4a

**Estimation:**

- `estimate` field on Task, Feature, Bug, and Epic entities
- Modified Fibonacci scale validation with per-type soft limit warnings
- Rollup queries: Feature effective estimate, Feature progress, Epic effective estimate, Epic progress
- `estimate_set` and `estimate_query` MCP tools and CLI commands
- AI calibration reference example management

**Dependency enforcement:**

- Enforcement rule in transition validator: `queued → ready` blocked if any `depends_on` task is not terminal
- `work_queue` MCP tool: promotes eligible tasks and returns ready queue
- `dependency_status` MCP tool: shows dependency state for a task
- Dependency cycle health check

**Task dispatch and completion:**

- Four new fields on Task schema: `claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`
- `dispatch_task` MCP tool: atomic claim, transition to `active`, context assembly, dispatch record
- `complete_task` MCP tool: close the dispatch loop with knowledge contribution
- `human_checkpoint` mechanism: structured pause for orchestrator escalation
- Stalled dispatch health check

**Context assembly enhancements:**

- `trimmed` field in `context_assemble` response: lists cut entries with ID, type, priority, size
- `orchestration_context` input parameter: orchestrator-injected handoff note

**Health check extensions:**

- Dependency cycle detection (error severity)
- Stalled dispatch detection (warning severity)
- Estimation coverage for active features (warning severity)

### 3.2 Deferred to Phase 4b

- Feature decomposition (`decompose_feature`, `decompose_review`)
- Worker review (`review_task_output`)
- Automatic dependency unblocking via `StatusTransitionHook`
- Conflict domain analysis (`conflict_domain_check`)
- Vertical slice decomposition guidance (`slice_analysis`)
- Incident entity and RootCauseAnalysis document type

### 3.3 Explicitly excluded

- General-purpose orchestration frameworks or external agent runtime integration
- Semantic search or embedding-based retrieval
- Cross-project knowledge sharing
- Webhook-based real-time synchronisation
- Automatic context assembly optimisation

---

## 4. Design Principles

### 4.1 Orchestration closes the loop, not the system

The system provides the tools. The orchestrating agent drives the loop: query the queue, pick a task, dispatch it, collect the result, advance state, repeat. Kanbanzai does not enforce a specific orchestration rhythm — it prevents incorrect moves (wrong state, unmet deps, conflicting claims) and provides complete information for each step.

### 4.2 Dependency enforcement is upstream of dispatch

Enforcement happens at the `queued → ready` transition, not at dispatch time. This means the system's state always reflects what is actually ready, regardless of how the orchestrator reaches the task. A task that appears in the work queue has already cleared its dependencies — the orchestrator does not need to re-check.

### 4.3 Dispatch is atomic

Only one agent can claim a given task. The `ready → active` transition is the atomic claiming gate. If two orchestrators attempt to dispatch the same task, exactly one succeeds; the other receives a clear refusal. This guarantee holds in the single-process MCP server through Go's per-request serialisation.

### 4.4 Self-management is earned, not assumed

The system manages Phase 4b development only after Phase 4a has been validated on real work. Feature-level transitions to `done` remain human-gated throughout Phase 4b. The `human_checkpoint` tool is the formal escalation path — it is not optional plumbing; it is a required safeguard before the self-management threshold is crossed.

### 4.5 Estimation is planning, not accounting

Estimates are a planning signal, not a commitment. Original estimates are preserved alongside computed rollups. Deltas are surfaced, not hidden. Soft limits warn rather than block. The goal is better-informed dispatch decisions, not false precision.

---

## 5. Approved Design Decisions

The following decisions are binding for this specification. See `work/plan/phase-4-decision-log.md` for full rationale.

| ID         | Decision                    | Summary                                                        |
|------------|-----------------------------|----------------------------------------------------------------|
| P4-DES-001 | Phase split                 | 4a = MVP; 4b = richer capabilities developed under self-management |
| P4-DES-002 | Estimation assignment       | Phase 4a; additive optional fields; no migrations needed       |
| P4-DES-003 | Self-management threshold   | After 4a validated; three safeguards; `human_checkpoint` required |
| P4-DES-004 | Dependency modelling        | Enforcement at `queued → ready`; manual polling in 4a          |
| P4-DES-005 | Agent delegation model      | Task state + four fields + `dispatch_task`; no new entity type |
| P4-DES-006 | Incidents and RCA           | Phase 4b only; not in scope for 4a                             |

---

## 6. Estimation

### 6.1 Purpose

Estimation provides a planning signal for the work queue: without sizing, the queue is a list; with it, the orchestrating agent can reason about capacity and prioritise accordingly. Rollup queries let humans and agents understand feature and epic progress without cascading file writes.

### 6.2 Estimation scale

All entities use the Modified Fibonacci sequence:

| Points | Size | Approximate meaning                                      |
|--------|------|----------------------------------------------------------|
| 0      | —    | No effort required                                       |
| 0.5    | XXS  | Minimal; trivial change                                  |
| 1      | XS   | Simple, well-understood; likely done in one day          |
| 2      | S    | Requires some thought; routine work                      |
| 3      | M    | Well-understood with a few extra steps                   |
| 5      | L    | Complex or infrequent; may need collaboration            |
| 8      | XL   | Requires research; likely multiple contributors          |
| 13     | XXL  | Highly complex with many unknowns                        |
| 20     | —    | Roughly one month of work                                |
| 40     | —    | Roughly two months of work                               |
| 100    | —    | Roughly five months of work                              |

Rough conversion: approximately 5 points per week.

`estimate_set` must reject values not in this set with a clear validation error.

### 6.3 Entity fields

The `estimate` field is optional on all four entity types. It holds a single value from the scale.

| Entity  | Field     | Type   | Notes                                                     |
|---------|-----------|--------|-----------------------------------------------------------|
| Task    | `estimate`| number | Optional; soft limit 13                                   |
| Feature | `estimate`| number | Original top-down estimate; not overwritten by rollup     |
| Epic    | `estimate`| number | Original top-down estimate; not overwritten by rollup     |
| Bug     | `estimate`| number | Optional; soft limit 13; self-contained, does not roll up |

### 6.4 Rollup rules

Rollups are computed on read. They are never stored.

**Feature effective estimate** (used in Epic rollup):

1. If the Feature has child Tasks with estimates (in active or terminal states, excluding `not-planned` and `duplicate`): use the sum of those Task estimates.
2. If the Feature has no such Tasks: use the Feature's own `estimate` field.
3. If neither exists: no estimate available.

**Feature progress**: sum of `estimate` from child Tasks in `done` state.

**Epic effective estimate**: sum of effective estimates across all child Features (using the Feature rule above).

**Epic progress**: sum of progress values across all child Features.

**Excluded states**: `not-planned` and `duplicate` are excluded from all totals and progress. They represent discarded work, not completed work.

**Delta**: when both an original estimate and a computed total are available for the same entity, the delta (computed − original) is surfaced. A positive delta indicates underestimation or scope growth; a negative delta indicates overestimation or scope reduction. Both are informational.

### 6.5 Soft limits

Exceeding the soft limit produces a warning in the `estimate_set` response. It does not block the operation.

| Entity  | Soft limit | Rationale                                           |
|---------|------------|-----------------------------------------------------|
| Task    | 13         | Above 13 suggests the task needs further decomposition |
| Bug     | 13         | Most fixes should be task-sized or smaller           |
| Feature | 100        | Features should not span many months                 |
| Epic    | 100        | Epics are the largest planning unit                  |

### 6.6 Calibration reference examples

AI agents estimating tasks lack the shared calibration that human teams build over time. Reference examples anchor AI estimates to completed, human-approved work.

Reference examples are stored as Tier 2 KnowledgeEntry records with:

- `scope`: `project`
- `tags`: `["estimation-reference"]`
- `topic`: a stable identifier for the referenced entity (e.g. `estimation-ref-TASK-01JX...`)
- `content`: a human-readable description of the work and its estimate, e.g.:
  > `TASK-01JX was rated 3 and involved adding a single validation rule with tests (internal/validate/lifecycle.go, ~50 lines, one edge case).`

When `estimate_set` is called, the system retrieves all entries tagged `estimation-reference` and includes them in the response alongside the scale definitions, so the caller has calibration context at estimation time.

`estimate_reference_add` and `estimate_reference_remove` manage the reference set. References designated by a human are treated as ground truth and should not be TTL-pruned; `estimate_reference_add` sets `ttl_days: 0` on the contributed entry to exempt it from TTL expiry.

---

## 7. Dependency Enforcement

### 7.1 Purpose

Tasks that depend on incomplete work must not be started. The dependency enforcement layer ensures the work queue reflects genuinely available work, not optimistically available work.

### 7.2 Enforcement rule

The `queued → ready` transition is dependency-gated.

**Rule**: A task cannot transition from `queued` to `ready` if any task ID in its `depends_on` list is not in a terminal state. The terminal states that satisfy a dependency are: `done`, `not-planned`, `duplicate`.

This rule is enforced in `validate.ValidateTransition`. An attempted `queued → ready` transition on a task with unresolved dependencies must fail with a descriptive error listing which dependency tasks are not yet terminal and their current status.

**Dependency cycles**: A task may not depend on itself, directly or transitively. Cycle detection is a health check concern (§10.1) rather than a per-transition check — cycles are caught at health check time and surfaced as errors requiring human resolution.

**Cross-feature dependencies**: Dependencies may reference tasks in other features. Enforcement does not distinguish between intra- and inter-feature dependencies.

### 7.3 Work queue promotion

The `work_queue` tool is the Phase 4a mechanism for the `queued → ready` transition. When called, it:

1. Loads all tasks in `queued` status.
2. For each, checks whether all `depends_on` entries are in terminal states.
3. Transitions qualifying tasks from `queued` to `ready` (applying the enforcement rule; if a task has no `depends_on` entries, it qualifies immediately).
4. Returns all tasks currently in `ready` status (including those just promoted).

This write-through query pattern means the orchestrator sees an up-to-date ready queue on every call without requiring a separate promotion step.

**Ordering**: results are ordered by `estimate` ascending (tasks with smaller estimates first; tasks with no estimate sorted last), then by task age descending (oldest tasks first within the same estimate band). This surfaces small, long-waiting tasks first.

**Filtering**: an optional `role` parameter filters results to tasks whose parent feature's assigned role profile matches. If omitted, all ready tasks are returned regardless of role.

### 7.4 Dependency status

`dependency_status` returns the full dependency picture for a given task: each entry in `depends_on`, its current status, and whether it is blocking (not yet terminal) or resolved (terminal). This gives an orchestrating agent a clear view of what a blocked task is waiting on.

---

## 8. Task Dispatch and Completion

### 8.1 Purpose

Dispatch is the act of an orchestrating agent assigning a ready task to an executing agent. Completion is the act of the executing agent reporting results. Together they form the core orchestration loop:

```
work_queue → pick task → dispatch_task → [agent executes] → complete_task → work_queue → ...
```

The loop includes a human escalation path via `human_checkpoint` for decisions the orchestrator cannot resolve autonomously.

### 8.2 Dispatch fields

Four fields are added to the Task schema to record dispatch metadata:

| Field            | Type       | Description                                                              |
|------------------|------------|--------------------------------------------------------------------------|
| `claimed_at`     | timestamp  | Set when the task transitions to `active`; the atomic claiming timestamp |
| `dispatched_to`  | string     | Role profile ID the orchestrator assigned this task to (e.g. `backend`)  |
| `dispatched_at`  | timestamp  | When the orchestrator called `dispatch_task`                             |
| `dispatched_by`  | string     | Identity string of the orchestrating agent                               |

All four fields are optional and absent on tasks that have never been dispatched. They are set atomically as part of the `ready → active` transition in `dispatch_task`.

### 8.3 The dispatch loop

The orchestrating agent is responsible for driving the loop. The system provides tools that prevent incorrect moves; it does not enforce a specific loop structure. A typical Phase 4a orchestration session:

1. Call `work_queue` to get the current ready tasks (promotes eligible queued tasks as a side effect).
2. Select a task and call `dispatch_task` with the task ID, target role, and orchestrator identity.
3. Receive the context packet. Hand it to the executing agent.
4. The executing agent works. When complete, the orchestrating agent calls `complete_task` with results and knowledge contributions.
5. If a decision is needed that the orchestrator cannot resolve, call `human_checkpoint`. Stop dispatching. Poll `human_checkpoint_get` until the human responds. Resume.
6. Repeat from step 1.

Human approval is required for feature-level transitions to `done` (P4-DES-003). `complete_task` operates at the task level only. Features are advanced by a human after reviewing task completions.

### 8.4 `dispatch_task` semantics

`dispatch_task` is an atomic operation:

1. Load the task. Verify its status is `ready`. If the status is `active`, return a clear error: the task has already been claimed. If the status is anything else, return a clear error.
2. Verify all `depends_on` entries are in terminal states. (This is a belt-and-suspenders check; `ready` status implies dependencies were met at promotion time, but the intervening period could theoretically have seen an edge case.)
3. Transition task status from `ready` to `active`.
4. Set `claimed_at` to the current timestamp.
5. Set `dispatched_to`, `dispatched_at`, `dispatched_by` from the request.
6. Write the updated task record.
7. Call `context_assemble` with the task's role profile, task ID, and any provided `orchestration_context`.
8. Return the task's updated state and the assembled context packet.

If any step fails after the file write in step 6, the task is in `active` state. The operation should not attempt rollback — the task is claimed and that claim is real. The error response should include the task ID so the caller can recover manually.

### 8.5 `complete_task` semantics

`complete_task` closes the dispatch loop:

1. Load the task. Verify its status is `active`. If not, return a clear error.
2. Transition task status to `done` (default) or `needs-review` (if requested).
3. Set `completed` timestamp on the task.
4. Record the completion summary and optional metadata fields on the task.
5. Process each entry in `knowledge_entries` through the existing knowledge contribution pipeline (`knowledge_contribute`). Deduplication and rejection rules apply. Errors on individual entries are reported but do not block the overall completion.
6. Return the updated task state and a summary of knowledge contributions (accepted, rejected, deduplicated).

**Target status**: `done` is the default. An orchestrating agent that wants a human to review the task output before it is considered done should pass `to_status: needs-review`. Feature-level transitions to `done` remain human-gated regardless of task status.

**Completion metadata fields** (all optional):

| Field                    | Type     | Description                                           |
|--------------------------|----------|-------------------------------------------------------|
| `summary`                | string   | Required. Brief description of what was done.         |
| `files_modified`         | []string | Paths of files created or modified.                   |
| `verification_performed` | string   | Description of testing or verification done.          |
| `blockers_encountered`   | string   | Any obstacles noted for future tasks or agents.       |
| `knowledge_entries`      | []object | Knowledge to contribute (see §8.5 knowledge schema).  |

**Knowledge entry schema** (each entry in `knowledge_entries`):

| Field        | Type     | Required | Description                                              |
|--------------|----------|----------|----------------------------------------------------------|
| `topic`      | string   | yes      | Topic identifier (will be normalised)                    |
| `content`    | string   | yes      | Concise, actionable knowledge statement                  |
| `scope`      | string   | yes      | Profile name or `project`                                |
| `tier`       | number   | no       | 2 or 3 (default: 3)                                      |
| `tags`       | []string | no       | Classification tags                                      |

### 8.6 Human checkpoint mechanism

The `human_checkpoint` tool allows an orchestrating agent to pause and surface a decision to a human without losing orchestration context. It is a required Phase 4a deliverable (P4-DES-003).

**Behavioural contract for the orchestrating agent**: after calling `human_checkpoint`, the agent must stop dispatching new tasks until it calls `human_checkpoint_get` and receives `status: responded`. The checkpoint mechanism does not enforce this — it provides the record and the response; the agent is responsible for honouring the pause. This is a deliberate design: the system cannot reliably interrupt an agent's session, but it can provide a clear protocol that well-behaved agents follow.

**Lifecycle**:

1. Orchestrator calls `human_checkpoint` with a question, context, and orchestration summary.
2. System creates a checkpoint record with `status: pending`. Returns the checkpoint ID.
3. Human reviews the checkpoint (via `human_checkpoint_list` and `human_checkpoint_get` CLI commands, or directly in the state file).
4. Human calls `human_checkpoint_respond` with their response.
5. System sets `status: responded` and records `response` and `responded_at`.
6. Orchestrator polls `human_checkpoint_get`. On `status: responded`, reads the response and resumes.

A `pending` checkpoint does not block other system operations. Only the orchestrating agent's own dispatch loop is paused by convention.

### 8.7 Checkpoint record

Checkpoints are stored in `.kbz/state/checkpoints/` as YAML files. The ID prefix is `CHK`.

**Required fields:**

| Field                  | Type      | Description                                                   |
|------------------------|-----------|---------------------------------------------------------------|
| `id`                   | string    | Checkpoint ID (format: `CHK-{ulid}`)                          |
| `question`             | string    | The decision or question the orchestrator needs answered      |
| `context`              | string    | Background information to help the human answer               |
| `orchestration_summary`| string    | Brief state of the orchestration session at checkpoint time   |
| `status`               | enum      | `pending` or `responded`                                      |
| `created_at`           | timestamp | When the checkpoint was created                               |
| `created_by`           | string    | Identity of the orchestrating agent                           |
| `responded_at`         | timestamp | When the human responded (null until responded)               |
| `response`             | string    | The human's answer (null until responded)                     |

**Field order for deterministic YAML:**

```yaml
id: CHK-01JX...
question: "Should I prioritise the cache invalidation task or the API pagination fix?"
context: "Both are in ready status. Cache invalidation is estimated 5 points and blocks TASK-01JY. Pagination is estimated 3 points and has been waiting 4 days."
orchestration_summary: "Phase 4b track A — 7/14 tasks complete. Next two ready tasks have different urgency profiles."
status: pending
created_at: 2026-03-25T10:00:00Z
created_by: orchestrator-session-abc
responded_at: null
response: null
```

---

## 9. Context Assembly Enhancements

### 9.1 Trimming visibility

When `context_assemble` trims entries to stay within the byte budget, the response now includes a `trimmed` field listing what was cut. This allows the executing agent to make an informed decision about whether to request additional context before starting work.

**`trimmed` field**: an array of objects, one per cut entry, ordered by the priority in which they were trimmed (lowest priority first):

| Field       | Type   | Description                                                     |
|-------------|--------|-----------------------------------------------------------------|
| `entry_id`  | string | ID of the trimmed entry (KnowledgeEntry ID or document section reference) |
| `type`      | string | `knowledge` or `design`                                         |
| `topic`     | string | Topic or title of the entry (for knowledge entries) or section title (for design context) |
| `tier`      | number | Knowledge tier (for knowledge entries); omitted for design context |
| `size_bytes`| number | Size of the entry as it would have appeared in the packet       |

An empty `trimmed` array (or absent field) means nothing was cut.

### 9.2 Orchestration context injection

`context_assemble` accepts a new optional `orchestration_context` string parameter. When provided, it is injected into the context packet as a Tier 3 knowledge entry with directed scope (the role profile of the assembled context), attributed to the orchestrating agent.

This allows the dispatching orchestrator to include a handoff note — what happened before this task, what the next agent needs to know, any cross-task observations — without polluting the shared knowledge store with session-specific state.

The injected entry is assembled into the packet like any other Tier 3 entry, but it is not persisted to the knowledge store. It is ephemeral: created for the duration of the context packet assembly and not written to `.kbz/state/knowledge/`.

---

## 10. Health Check Extensions

Phase 4a adds three new health check categories to the existing `health_check` MCP tool.

### 10.1 Dependency cycle detection

**Category name**: `dependency_cycles`

**Severity**: error

**What is checked**: all tasks are loaded and their `depends_on` chains are traversed. Any cycle (task A depends on task B which depends, directly or transitively, on task A) is reported.

**Output per issue**:

- `severity`: error
- `task_ids`: the task IDs that form the cycle
- `message`: a description of the cycle, e.g. `"dependency cycle: TASK-01JX → TASK-01JY → TASK-01JX"`

A dependency cycle makes it impossible for any task in the cycle to reach `ready` status. It requires human intervention to break.

### 10.2 Stalled dispatch detection

**Category name**: `stalled_dispatches`

**Severity**: warning

**What is checked**: all tasks in `active` status where `dispatched_at` is older than the configured stall threshold (`dispatch.stall_threshold_days`, default: 3 days) and no git activity has been recorded on the associated worktree's branch since `dispatched_at`.

**Output per issue**:

- `severity`: warning
- `task_id`: the stalled task ID
- `dispatched_at`: when the task was dispatched
- `dispatched_to`: the role profile it was dispatched to
- `days_stalled`: number of days since dispatch with no activity
- `message`: e.g. `"TASK-01JX has been active for 5 days with no git activity since dispatch"`

A stalled dispatch typically means the executing agent's session ended without completion, or the task is genuinely stuck. Human intervention is needed to determine whether to re-dispatch or transition the task back to `ready`.

The git activity check is best-effort: if the task's parent feature has no associated worktree, the git check is skipped and the time threshold alone is used.

### 10.3 Estimation coverage

**Category name**: `estimation_coverage`

**Severity**: warning

**What is checked**: all Features in `active` or later statuses (i.e. statuses that appear after `draft` in the Feature lifecycle) that have child Tasks where none of the Tasks carry an `estimate` value.

**Output per issue**:

- `severity`: warning
- `feature_id`: the Feature with no estimated tasks
- `task_count`: number of child tasks
- `message`: e.g. `"FEAT-01JX has 4 active tasks with no estimates — work queue ordering will be incomplete"`

This check fires once the feature is actively being worked on. Features in `draft` or `proposed` status are excluded — early-stage features are expected to be unestimated.

---

## 11. Storage Model

### 11.1 Task schema additions

Seven new optional fields are added to the Task entity. All are absent from existing records; adding optional fields is backwards-compatible and requires no migration.

**Updated field list for Task** (fields marked `new` are additions):

| Field             | Type      | Required | Notes                                              |
|-------------------|-----------|----------|----------------------------------------------------|
| `id`              | string    | yes      | Task ID                                            |
| `parent_feature`  | string    | yes      | Parent Feature ID                                  |
| `slug`            | string    | yes      | Human-readable slug                                |
| `summary`         | string    | yes      | One-line description                               |
| `status`          | enum      | yes      | Task lifecycle status                              |
| `estimate`        | number    | no       | **new** Story points (Modified Fibonacci scale)    |
| `assignee`        | string    | no       | Assigned person or role                            |
| `depends_on`      | []string  | no       | Task IDs this task depends on                      |
| `files_planned`   | []string  | no       | Files expected to be modified                      |
| `started`         | timestamp | no       | When task was first moved to `active`              |
| `completed`       | timestamp | no       | When task reached a terminal status                |
| `claimed_at`      | timestamp | no       | **new** Set on `ready → active` by `dispatch_task` |
| `dispatched_to`   | string    | no       | **new** Role profile ID of the assigned executor   |
| `dispatched_at`   | timestamp | no       | **new** When `dispatch_task` was called            |
| `dispatched_by`   | string    | no       | **new** Identity of the dispatching orchestrator   |
| `completion_summary`| string  | no       | **new** Summary set by `complete_task`             |
| `verification`    | string    | no       | Verification requirement or result                 |
| `tags`            | []string  | no       | Classification tags                                |

**Deterministic field order for YAML serialisation:**

```
id, parent_feature, slug, summary, status, estimate, assignee, depends_on,
files_planned, started, completed, claimed_at, dispatched_to, dispatched_at,
dispatched_by, completion_summary, verification, tags
```

### 11.2 Feature schema additions

One new optional field is added to Feature:

| Field      | Type   | Required | Notes                                         |
|------------|--------|----------|-----------------------------------------------|
| `estimate` | number | no       | **new** Original top-down estimate; not overwritten by rollup |

`estimate` is inserted after `status` in the deterministic field order.

### 11.3 Epic schema additions

One new optional field is added to Epic:

| Field      | Type   | Required | Notes                                         |
|------------|--------|----------|-----------------------------------------------|
| `estimate` | number | no       | **new** Original top-down estimate; not overwritten by rollup |

`estimate` is inserted after `status` in the deterministic field order.

### 11.4 Bug schema additions

One new optional field is added to Bug:

| Field      | Type   | Required | Notes                                          |
|------------|--------|----------|------------------------------------------------|
| `estimate` | number | no       | **new** Story points; self-contained, not rolled up |

`estimate` is inserted after `status` in the deterministic field order.

### 11.5 Checkpoint record

Checkpoint records are stored in `.kbz/state/checkpoints/` as individual YAML files named `{id}.yaml`.

**Deterministic field order:**

```
id, question, context, orchestration_summary, status, created_at, created_by,
responded_at, response
```

Checkpoints are not full entity records — they have no parent entity, no lifecycle validation, and no ID prefix registration. They are lightweight tracking records, similar in structure to worktree records. The ID prefix `CHK` is used for readability but is not registered in the prefix registry.

Resolved checkpoints (`status: responded`) may be archived or deleted manually. No automated cleanup is defined in Phase 4a.

### 11.6 Deterministic formatting

All new entity fields and checkpoint records follow the YAML serialisation rules defined in P1-DEC-008:

- Block style for mappings and sequences
- Double-quoted strings only when required by YAML syntax
- Deterministic field order as defined above
- UTF-8, LF line endings, trailing newline
- No YAML tags, anchors, or aliases

Round-trip tests (write → read → write → compare) are required for all new schema additions.

---

## 12. MCP Interface

### 12.1 Estimation operations

#### `estimate_set`

Set or update the estimate for an entity.

**Input parameters:**

| Parameter   | Type   | Required | Description                                                |
|-------------|--------|----------|------------------------------------------------------------|
| `entity_id` | string | yes      | Task, Feature, Epic, or Bug ID                             |
| `estimate`  | number | yes      | Story points value (must be in Modified Fibonacci scale)   |

**Output:**

```yaml
entity_id: TASK-01JX...
entity_type: task
estimate: 5
soft_limit_warning: null   # or a warning string if over soft limit
references:                # current calibration reference examples
  - topic: estimation-ref-TASK-01JY...
    content: "TASK-01JY was rated 3 and involved adding a validation rule with tests (~50 lines, one edge case)."
  - topic: estimation-ref-TASK-01JZ...
    content: "TASK-01JZ was rated 8 and involved designing a new storage format with migration logic (~200 lines, three affected packages)."
scale:
  - points: 0.5
    meaning: Minimal; trivial change
  - points: 1
    meaning: Simple, well-understood; likely done in one day
  # ... remainder of scale
```

**Errors:**

- Invalid entity ID: entity not found
- Invalid estimate value: not in Modified Fibonacci scale

**Notes:** `soft_limit_warning` is populated (not null) when the estimate exceeds the entity type's soft limit. The operation succeeds regardless. `references` and `scale` are included in the response to give AI callers calibration context without a separate tool call.

---

#### `estimate_query`

Query an entity's estimate and computed rollup totals.

**Input parameters:**

| Parameter   | Type   | Required | Description                     |
|-------------|--------|-----------|---------------------------------|
| `entity_id` | string | yes       | Task, Feature, Epic, or Bug ID  |

**Output (Feature example):**

```yaml
entity_id: FEAT-01JX...
entity_type: feature
estimate: 8              # original top-down estimate (if set)
rollup:
  task_total: 16         # sum of child task estimates
  progress: 6            # sum of estimates from done tasks
  delta: +8              # task_total - estimate
  task_count: 5
  estimated_task_count: 4
  excluded_task_count: 1 # not-planned or duplicate
```

**Output (Task example):**

```yaml
entity_id: TASK-01JX...
entity_type: task
estimate: 5
rollup: null             # tasks have no children to roll up
```

**Output (Epic example):**

```yaml
entity_id: EPIC-01JX...
entity_type: epic
estimate: 40
rollup:
  feature_total: 37
  progress: 12
  delta: -3
  feature_count: 4
  estimated_feature_count: 4
```

**Notes:** `rollup` is `null` for Tasks and Bugs (they have no children in the hierarchy). `delta` is omitted when either the original estimate or the computed total is absent. `excluded_task_count` counts Tasks in `not-planned` or `duplicate` status.

---

#### `estimate_reference_add`

Add a calibration reference example.

**Input parameters:**

| Parameter   | Type   | Required | Description                                                     |
|-------------|--------|----------|-----------------------------------------------------------------|
| `entity_id` | string | yes      | The completed entity this example is based on                   |
| `content`   | string | yes      | Description of the work and its estimate (see §6.6 format)      |

**Output:**

```yaml
entry_id: KE-01JX...
entity_id: TASK-01JY...
topic: estimation-ref-TASK-01JY...
status: added
```

**Notes:** The entry is contributed as a Tier 2 KnowledgeEntry with `ttl_days: 0` (exempt from TTL pruning), scope `project`, and tag `estimation-reference`. The operation uses the knowledge contribution pipeline; deduplication rules apply.

---

#### `estimate_reference_remove`

Remove a calibration reference example.

**Input parameters:**

| Parameter   | Type   | Required | Description                                   |
|-------------|--------|----------|-----------------------------------------------|
| `entity_id` | string | yes      | The entity whose reference example is removed |

**Output:**

```yaml
entity_id: TASK-01JY...
entry_id: KE-01JX...
status: removed
```

**Errors:** No reference example found for the given entity ID.

---

### 12.2 Dependency operations

#### `work_queue`

Return the current ready task queue, promoting eligible queued tasks first.

**Input parameters:**

| Parameter | Type   | Required | Description                                                      |
|-----------|--------|----------|------------------------------------------------------------------|
| `role`    | string | no       | Filter results to tasks whose parent feature matches this role profile |

**Behaviour:**

1. Load all tasks in `queued` status. For each, attempt `queued → ready` promotion if all `depends_on` entries are terminal (same enforcement rule as §7.2). Tasks that fail promotion (unmet deps) are skipped silently.
2. Load all tasks currently in `ready` status.
3. Apply `role` filter if provided.
4. Sort: estimate ascending (null last), then age descending.
5. Return the list.

**Output:**

```yaml
queue:
  - task_id: TASK-01JX...
    slug: implement-jwt-middleware
    summary: Implement JWT authentication middleware with RS256 signature verification
    parent_feature: FEAT-01JA...
    feature_slug: user-authentication
    estimate: 5
    age_days: 3
    status: ready
  - task_id: TASK-01JY...
    slug: add-rate-limiting
    summary: Add rate limiting to the public API endpoints
    parent_feature: FEAT-01JB...
    feature_slug: api-stability
    estimate: 8
    age_days: 1
    status: ready
  - task_id: TASK-01JZ...
    slug: update-openapi-spec
    summary: Update OpenAPI spec to reflect new auth endpoints
    parent_feature: FEAT-01JA...
    feature_slug: user-authentication
    estimate: null
    age_days: 3
    status: ready
promoted_count: 2          # number of tasks promoted from queued → ready during this call
total_queued: 5            # total tasks remaining in queued (after promotion, still blocked)
```

**Notes:** `promoted_count` tells the orchestrator how many tasks just became available. `total_queued` gives visibility into the remaining backlog.

---

#### `dependency_status`

Show the dependency picture for a given task.

**Input parameters:**

| Parameter | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `task_id` | string | yes      | Task ID     |

**Output:**

```yaml
task_id: TASK-01JX...
slug: implement-jwt-middleware
status: queued
depends_on_count: 2
blocking_count: 1
dependencies:
  - task_id: TASK-01JW...
    slug: design-auth-schema
    status: done
    blocking: false
    terminal_state: done
  - task_id: TASK-01JV...
    slug: provision-secrets-manager
    status: active
    blocking: true
    terminal_state: null
```

**Notes:** `blocking: true` means the dependency is not yet terminal and is preventing this task from becoming `ready`. `blocking: false` means the dependency is satisfied.

---

### 12.3 Dispatch and completion operations

#### `dispatch_task`

Atomically claim a ready task and return its context packet.

**Input parameters:**

| Parameter              | Type   | Required | Description                                                       |
|------------------------|--------|----------|-------------------------------------------------------------------|
| `task_id`              | string | yes      | Task ID to dispatch (must be in `ready` status)                   |
| `role`                 | string | yes      | Role profile ID for the executing agent (e.g. `backend`)          |
| `dispatched_by`        | string | yes      | Identity string of the orchestrating agent                        |
| `orchestration_context`| string | no       | Handoff note injected into the context packet (see §9.2)          |
| `max_bytes`            | number | no       | Byte budget for context assembly (default: 30720)                 |

**Behaviour:** see §8.4.

**Output:**

```yaml
task:
  id: TASK-01JX...
  slug: implement-jwt-middleware
  summary: Implement JWT authentication middleware with RS256 signature verification
  status: active
  claimed_at: 2026-03-25T10:00:00Z
  dispatched_to: backend
  dispatched_at: 2026-03-25T10:00:00Z
  dispatched_by: orchestrator-session-abc
context:
  role: backend
  profile:
    # ... full role profile content
  knowledge:
    # ... knowledge entries (Tier 2 and scoped Tier 3)
  task_instructions:
    # ... task details, verification, acceptance criteria
  design_context:
    # ... relevant design document sections
  byte_usage: 18432
  byte_budget: 30720
  trimmed:
    - entry_id: KE-01JY...
      type: knowledge
      topic: legacy-auth-patterns
      tier: 3
      size_bytes: 1024
```

**Errors:**

- Task not found
- Task status is not `ready` (includes specific message if already `active`: "task TASK-01JX is already claimed — dispatched by {dispatched_by} at {claimed_at}")
- Dependency check failed (should not occur for a `ready` task, but reported if detected)
- Role profile not found

---

#### `complete_task`

Close the dispatch loop for a completed task.

**Input parameters:**

| Parameter                | Type     | Required | Description                                             |
|--------------------------|----------|----------|---------------------------------------------------------|
| `task_id`                | string   | yes      | Task ID to complete (must be in `active` status)        |
| `summary`                | string   | yes      | Brief description of what was accomplished              |
| `to_status`              | string   | no       | `done` (default) or `needs-review`                      |
| `files_modified`         | []string | no       | Files created or modified                               |
| `verification_performed` | string   | no       | Testing or verification carried out                     |
| `blockers_encountered`   | string   | no       | Obstacles noted for future work                         |
| `knowledge_entries`      | []object | no       | Knowledge to contribute (schema in §8.5)                |

**Behaviour:** see §8.5.

**Output:**

```yaml
task:
  id: TASK-01JX...
  status: done
  completed: 2026-03-25T14:30:00Z
  completion_summary: "Implemented JWT middleware using RS256. All tests pass. Key rotation via JWKS endpoint supported."
knowledge_contributions:
  accepted:
    - entry_id: KE-01JZ...
      topic: jwt-rs256-key-rotation
  rejected:
    - topic: golang-error-handling
      reason: "duplicate: existing entry KE-01JW covers this topic"
  total_attempted: 2
  total_accepted: 1
```

**Errors:**

- Task not found
- Task status is not `active`
- Invalid `to_status` value

---

#### `human_checkpoint`

Record a structured decision point and pause orchestration.

**Input parameters:**

| Parameter                | Type   | Required | Description                                                     |
|--------------------------|--------|----------|-----------------------------------------------------------------|
| `question`               | string | yes      | The decision or question requiring human input                  |
| `context`                | string | yes      | Background information to help the human answer                 |
| `orchestration_summary`  | string | yes      | Brief state of the orchestration session at checkpoint time     |
| `created_by`             | string | yes      | Identity of the orchestrating agent                             |

**Output:**

```yaml
checkpoint_id: CHK-01JX...
status: pending
created_at: 2026-03-25T10:00:00Z
message: "Checkpoint recorded. Stop dispatching new tasks. Poll human_checkpoint_get with checkpoint_id until status is 'responded'."
```

---

#### `human_checkpoint_respond`

Record a human response to a pending checkpoint.

**Input parameters:**

| Parameter       | Type   | Required | Description                                      |
|-----------------|--------|----------|--------------------------------------------------|
| `checkpoint_id` | string | yes      | CHK ID of the checkpoint to respond to           |
| `response`      | string | yes      | The human's answer or decision                   |

**Output:**

```yaml
checkpoint_id: CHK-01JX...
status: responded
responded_at: 2026-03-25T11:15:00Z
```

**Errors:**

- Checkpoint not found
- Checkpoint already has `status: responded`

---

#### `human_checkpoint_get`

Get the current state of a checkpoint.

**Input parameters:**

| Parameter       | Type   | Required | Description |
|-----------------|--------|----------|-------------|
| `checkpoint_id` | string | yes      | CHK ID      |

**Output:**

```yaml
checkpoint_id: CHK-01JX...
question: "Should I prioritise the cache invalidation task or the API pagination fix?"
context: "Both are in ready status. Cache invalidation is estimated 5 points and blocks TASK-01JY. Pagination is estimated 3 points and has been waiting 4 days."
orchestration_summary: "Phase 4b track A — 7/14 tasks complete."
status: responded
created_at: 2026-03-25T10:00:00Z
created_by: orchestrator-session-abc
responded_at: 2026-03-25T11:15:00Z
response: "Prioritise the cache invalidation — the downstream block on TASK-01JY is more important than the pagination age."
```

---

#### `human_checkpoint_list`

List all checkpoints, optionally filtered by status.

**Input parameters:**

| Parameter | Type   | Required | Description                                               |
|-----------|--------|----------|-----------------------------------------------------------|
| `status`  | string | no       | Filter by `pending` or `responded`; omit for all          |

**Output:**

```yaml
checkpoints:
  - checkpoint_id: CHK-01JX...
    question: "Should I prioritise cache invalidation or pagination fix?"
    status: pending
    created_at: 2026-03-25T10:00:00Z
    created_by: orchestrator-session-abc
  - checkpoint_id: CHK-01JW...
    question: "Is the rate limiting approach approved for the new API?"
    status: responded
    created_at: 2026-03-24T09:00:00Z
    created_by: orchestrator-session-xyz
    responded_at: 2026-03-24T10:30:00Z
total: 2
pending_count: 1
```

---

### 12.4 `context_assemble` extensions

The existing `context_assemble` tool is extended with two changes.

**New input parameter:**

| Parameter              | Type   | Required | Description                                         |
|------------------------|--------|----------|-----------------------------------------------------|
| `orchestration_context`| string | no       | Handoff note injected as ephemeral Tier 3 entry (§9.2) |

**New response field:**

The response object gains a `trimmed` array field. When nothing is trimmed, the field is an empty array. Schema is defined in §9.1.

No breaking changes are made to existing `context_assemble` behaviour or required parameters.

---

### 12.5 MCP validation

All new tools follow the validation pattern established in Phases 1–3:

- Unknown parameters are rejected with a descriptive error.
- Required parameters that are missing produce a clear error naming the missing field.
- Entity IDs that do not exist produce a "not found" error, not a panic.
- Invalid enum values (e.g. `to_status: cancelled`) produce a validation error listing accepted values.
- Estimation values not in the Modified Fibonacci scale produce a validation error listing valid values.

---

## 13. CLI Interface

### 13.1 Estimation commands

```
kbz estimate set <entity-id> <points>
  Set the estimate for a task, feature, epic, or bug.
  Shows calibration references and scale at confirmation.
  Example: kbz estimate set TASK-01JX 5

kbz estimate show <entity-id>
  Show an entity's estimate and rollup totals.
  Example: kbz estimate show FEAT-01JX

kbz estimate reference add <entity-id> "<description>"
  Add a completed entity as a calibration reference example.
  Example: kbz estimate reference add TASK-01JX "Rated 3; added validation rule with tests"

kbz estimate reference remove <entity-id>
  Remove a calibration reference example.
  Example: kbz estimate reference remove TASK-01JX

kbz estimate reference list
  List all current calibration reference examples.
```

### 13.2 Queue and dependency commands

```
kbz queue [--role <profile>]
  Show the current work queue (promotes eligible tasks as a side effect).
  Optional --role flag filters to a specific role profile.
  Example: kbz queue
  Example: kbz queue --role backend

kbz task deps <task-id>
  Show dependency status for a task: which deps are blocking, which are resolved.
  Example: kbz task deps TASK-01JX
```

### 13.3 Dispatch commands

```
kbz dispatch <task-id> --role <profile> --by <identity>
  Dispatch a task (claim it and print the context packet).
  Intended for human-initiated dispatch during Phase 4a validation.
  Example: kbz dispatch TASK-01JX --role backend --by sam

kbz task complete <task-id>
  Interactively complete a task: prompts for summary, files modified,
  verification performed, and knowledge entries.
  Example: kbz task complete TASK-01JX
```

### 13.4 Checkpoint commands

```
kbz checkpoint list [--status pending|responded]
  List checkpoints, optionally filtered by status.
  Example: kbz checkpoint list
  Example: kbz checkpoint list --status pending

kbz checkpoint show <checkpoint-id>
  Show a checkpoint in full, including response if responded.
  Example: kbz checkpoint show CHK-01JX

kbz checkpoint respond <checkpoint-id> "<response>"
  Respond to a pending checkpoint.
  Example: kbz checkpoint respond CHK-01JX "Proceed with GraphQL approach."
```

---

## 14. Configuration

Phase 4a adds the following project configuration options to `.kbz/config.yaml`.

```yaml
dispatch:
  stall_threshold_days: 3       # days after dispatch_at with no git activity before flagging as stalled
                                # default: 3; set to 0 to disable stalled dispatch health check

estimation:
  coverage_warn_at_status: active   # feature status at which missing task estimates trigger a warning
                                    # default: active
```

**Defaults are applied if the keys are absent.** Invalid values (e.g. negative `stall_threshold_days`) are rejected with a clear error on server startup.

No local configuration additions are required for Phase 4a.

---

## 15. Acceptance Criteria

### 15.1 Estimation

- [ ] `estimate` field is accepted and stored on Task, Feature, Epic, and Bug entities
- [ ] Modified Fibonacci values (0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100) are accepted; all other values are rejected with a validation error listing valid values
- [ ] Soft limit warnings are emitted for Task and Bug estimates above 13, and for Feature and Epic estimates above 100; the operation succeeds regardless
- [ ] Feature effective estimate rollup follows §6.4 rules: task total if tasks with estimates exist, own estimate otherwise
- [ ] Feature progress rollup correctly sums estimates from `done` child tasks only
- [ ] Epic effective estimate and progress roll up from Features following §6.4 rules
- [ ] `not-planned` and `duplicate` tasks are excluded from all rollup totals and progress
- [ ] `estimate_query` returns entity, rollup totals, progress, and delta where applicable
- [ ] Delta is shown when both original estimate and computed total are present; omitted when either is absent
- [ ] Reference examples are stored as Tier 2 KnowledgeEntry records with tag `estimation-reference` and `ttl_days: 0`
- [ ] `estimate_set` response includes current reference examples and scale definitions
- [ ] `estimate_reference_remove` correctly retires the reference entry
- [ ] Round-trip serialisation (write → read → write → compare) produces identical output for all entities with `estimate` field

### 15.2 Dependency enforcement

- [ ] `queued → ready` transition is blocked if any `depends_on` task is not in `done`, `not-planned`, or `duplicate` state
- [ ] The error message names the blocking dependency tasks and their current status
- [ ] `work_queue` promotes eligible `queued` tasks to `ready` before returning results
- [ ] `work_queue` returns only tasks in `ready` status
- [ ] `work_queue` result is ordered: estimate ascending (null last), then age descending
- [ ] `work_queue` `role` filter correctly limits results to tasks whose parent feature matches the profile
- [ ] `work_queue` includes `promoted_count` and `total_queued` in its response
- [ ] `dependency_status` shows all `depends_on` entries with their current status and blocking flag
- [ ] A task with no `depends_on` entries is immediately eligible for `queued → ready` promotion

### 15.3 Dispatch and completion

- [ ] `dispatch_task` requires task in `ready` status; returns a clear error if status is `active` (with claimed_at and dispatched_by in the error) or any other non-ready status
- [ ] `dispatch_task` atomically transitions task to `active` and sets all four dispatch fields (`claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`) in a single write
- [ ] A second `dispatch_task` call on the same task (already `active`) returns the "already claimed" error and does not modify the task
- [ ] `dispatch_task` response includes the assembled context packet with all sections
- [ ] `dispatch_task` response `context.trimmed` reflects entries that were cut
- [ ] `complete_task` requires task in `active` status; returns a clear error otherwise
- [ ] `complete_task` transitions task to `done` (default) or `needs-review` when `to_status: needs-review` is specified
- [ ] `complete_task` sets `completed` timestamp on the task
- [ ] `complete_task` stores `completion_summary` on the task
- [ ] Each valid knowledge entry in `knowledge_entries` is processed through the knowledge contribution pipeline
- [ ] Duplicate knowledge entries are rejected with a reason and do not block the overall completion
- [ ] `complete_task` response lists accepted, rejected, and total knowledge contributions
- [ ] `human_checkpoint` creates a CHK record with `status: pending`
- [ ] `human_checkpoint_respond` transitions checkpoint to `status: responded` and records `response` and `responded_at`
- [ ] `human_checkpoint_respond` on an already-responded checkpoint returns a clear error
- [ ] `human_checkpoint_get` returns full checkpoint state including response when responded
- [ ] `human_checkpoint_list` with `status: pending` returns only pending checkpoints
- [ ] `human_checkpoint_list` with `status: responded` returns only responded checkpoints

### 15.4 Context assembly enhancements

- [ ] `context_assemble` response includes `trimmed` field when entries are cut; `trimmed` is an empty array when nothing is cut
- [ ] Each entry in `trimmed` includes `entry_id`, `type`, `topic`, `tier` (for knowledge entries), and `size_bytes`
- [ ] `trimmed` entries are ordered by the priority in which they were trimmed (lowest priority first)
- [ ] `orchestration_context` parameter is accepted by `context_assemble`
- [ ] When `orchestration_context` is provided, its content is included in the context packet as an ephemeral Tier 3 entry
- [ ] The `orchestration_context` entry is not written to `.kbz/state/knowledge/`
- [ ] Existing `context_assemble` callers without `orchestration_context` or `trimmed` expectations are unaffected

### 15.5 Health checks

- [ ] Dependency cycle health check detects and reports cycles with error severity
- [ ] Dependency cycle report includes the task IDs that form the cycle
- [ ] Stalled dispatch health check flags active tasks past the configured `stall_threshold_days` with no git activity
- [ ] Stalled dispatch report includes `task_id`, `dispatched_at`, `dispatched_to`, and `days_stalled`
- [ ] Stalled dispatch check is disabled when `stall_threshold_days: 0`
- [ ] Estimation coverage check flags features in `active` or later status with no estimated child tasks
- [ ] All three new health check categories appear in `health_check` output under their category names
- [ ] Severity levels are correct: `dependency_cycles` = error, `stalled_dispatches` = warning, `estimation_coverage` = warning

### 15.6 Deterministic storage

- [ ] Task records with new fields serialise in the field order defined in §11.1
- [ ] Feature, Epic, and Bug records with `estimate` field serialise with `estimate` after `status`
- [ ] Checkpoint records serialise in the field order defined in §11.5
- [ ] Round-trip (write → read → write → compare) produces identical output for all new record types and field additions
- [ ] Checkpoint records with `null` values for `responded_at` and `response` serialise without omitting those fields (explicit nulls)

---

## 16. Open Questions Resolved During Specification

### 16.1 `work_queue` and the `queued → ready` transition

**Question:** In Phase 4a (no automatic `StatusTransitionHook`), what mechanism transitions tasks from `queued` to `ready`?

**Decision:** `work_queue` is the Phase 4a mechanism. It performs a write-through query: it promotes eligible `queued` tasks to `ready` before returning the ready list. The orchestrator drives this by calling `work_queue` between dispatch cycles. Phase 4b's `StatusTransitionHook` replaces this with automatic promotion on dependency completion.

**Rationale:** This keeps promotion logic in one place (the enforcement rule in the validator), avoids a separate explicit promotion tool, and matches the "poll the queue between assignments" model described in P4-DES-004. The write-through semantics are safe in a single-process server.

### 16.2 `complete_task` default target status

**Question:** Does `complete_task` transition to `done` or `needs-review` by default?

**Decision:** `done` is the default. P4-DES-003 states "Tasks may be agent-autonomous; features stay human-gated." `needs-review` is an explicit option for orchestrators that want a human to review task output before considering it done. Feature-level transitions to `done` remain human-gated regardless.

**Rationale:** Requiring human review at the task level for every task would defeat the purpose of autonomous orchestration. The feature gate provides the meaningful human checkpoint for delivered work.

### 16.3 Calibration reference example storage

**Question:** How are estimation reference examples stored, retrieved, and presented?

**Decision:** Tier 2 KnowledgeEntry records with tag `estimation-reference`, scope `project`, and `ttl_days: 0` (exempt from TTL pruning). Retrieved during `estimate_set` by querying for entries with this tag. The `estimate_set` response includes them alongside the scale definitions so AI callers receive calibration context at estimation time without a separate call.

**Rationale:** Reusing the existing knowledge store avoids a new storage format. TTL exemption is correct — reference examples are curated assets that should not expire. Tier 2 with project scope ensures they are available in all context assembly requests.

### 16.4 `orchestration_context` persistence

**Question:** Is the `orchestration_context` injection persisted to the knowledge store?

**Decision:** No. It is ephemeral — assembled into the context packet but not written to `.kbz/state/knowledge/`. The dispatching orchestrator's handoff note is session-specific and often not reusable across different task contexts.

**Rationale:** Persisting ephemeral session notes would pollute the knowledge store with low-value, context-specific content that the TTL system would then need to prune. If the orchestrator identifies genuinely reusable knowledge in the handoff, it should contribute it explicitly via `complete_task`'s `knowledge_entries` field.

### 16.5 Checkpoint cleanup policy

**Question:** Are responded checkpoints cleaned up automatically?

**Decision:** No automatic cleanup in Phase 4a. Resolved checkpoints may be deleted manually. This is consistent with Phase 3's pattern of not automatically deleting knowledge or decision records.

**Rationale:** Responded checkpoints may be valuable for retrospective review (what decisions were made during Phase 4b self-management?). Automatic deletion would lose this audit trail. If accumulation becomes a concern, a cleanup tool can be added in a later phase.

### 16.6 `work_queue` sorting tie-breaking

**Question:** For tasks with the same estimate and age, what determines order?

**Decision:** Task ID lexicographic order (ascending). This produces a stable, deterministic sort without requiring additional fields.

**Rationale:** Deterministic ordering matters for testability and for consistent orchestrator behaviour across runs. Any consistent tie-breaking rule is acceptable; lexicographic ID is the simplest.

---

## 17. Implementation Notes

### 17.1 Estimation implementation order

The estimation design (`estimation-and-progress-design.md`) should be implemented first — it can be treated as pre-Phase 4a housekeeping since it is purely additive and does not depend on the dispatch machinery. Completing estimation early means the work queue is immediately useful for capacity-aware scheduling when dependency enforcement lands.

### 17.2 `dispatch_task` atomicity

In the current single-process MCP server, Go's per-request goroutine serialisation combined with synchronous file writes provides adequate atomicity for `dispatch_task`. Two concurrent MCP requests to dispatch the same task will be serialised by Go's HTTP request handling; the first write wins, and the second will load the updated `active` status and return the "already claimed" error.

File-level locking (as used in Phase 1 for ID allocation) is not required for `dispatch_task` in the current architecture. If concurrent MCP server instances are ever deployed, this assumption should be revisited and explicit locking added.

### 17.3 Estimation rollup performance

Rollup queries traverse all child entities on every call. For typical project sizes (tens of features, hundreds of tasks), this is fast enough to compute on read without caching. If performance becomes a concern at larger scales, rollup values may be cached in the derived SQLite cache (`internal/cache/`) in a future phase. Do not add caching in Phase 4a — the design principle (§2.1 of `estimation-and-progress-design.md`) is compute on read, and premature optimisation here would complicate the implementation unnecessarily.

### 17.4 `work_queue` promotion side effects

`work_queue` modifies task state (promoting `queued` → `ready`) as a side effect of a query. Document this clearly in the tool description so callers understand it is not a pure read. The `StatusTransitionHook` fires for these promotions, which means worktree creation does not trigger (worktrees are tied to `active` transitions, not `ready`). Confirm this does not unintentionally trigger other hook logic when extending the hook in Phase 4b.

### 17.5 Checkpoint record format

Checkpoint records use explicit `null` values for `responded_at` and `response` rather than omitting the fields. This makes the unresponded state explicit and avoids confusion between "never responded" and "field not yet defined in schema". The deterministic serialiser must be extended to handle this case for checkpoint records specifically (all other optional fields in the system use omitempty).

### 17.6 `StatusTransitionHook` interaction

The existing `StatusTransitionHook` fires on all status transitions, including `queued → ready` (triggered by `work_queue`) and `ready → active` (triggered by `dispatch_task`). The current hook implementation (`WorktreeTransitionHook`) only acts on `active` transitions for tasks and `in-progress` transitions for bugs — so `queued → ready` transitions from `work_queue` will fire the hook but produce no side effects. Verify this in tests to ensure no unintended behaviour.

### 17.7 Phase 4a validation workload

Before Phase 4b begins, Phase 4a must be validated on at least one real workload. The natural candidate is using Phase 4a tools to manage a small piece of Phase 4b scoping or planning work. The validation should confirm:

1. The dispatch loop completes end-to-end without manual intervention beyond intended human checkpoints.
2. Estimation and the work queue produce useful scheduling decisions.
3. Knowledge contributions from `complete_task` are correctly integrated.
4. Health checks correctly identify any issues in the test workload.

This validation is a human responsibility — it cannot be automated.

---

## 18. Summary

Phase 4a delivers five feature tracks:

| Track | Feature                       | Core deliverable                                              |
|-------|-------------------------------|---------------------------------------------------------------|
| 1     | Estimation                    | `estimate` field on entities; rollup queries; AI calibration references |
| 2     | Dependency enforcement        | `queued → ready` gate; `work_queue` promotion; `dependency_status` |
| 3     | Task dispatch and completion  | `dispatch_task`; `complete_task`; `human_checkpoint` mechanism |
| 4     | Context assembly enhancements | `trimmed` visibility; `orchestration_context` injection        |
| 5     | Health check extensions       | Dependency cycles; stalled dispatches; estimation coverage     |

Together these close the orchestration loop: an agent can query what is ready, claim a task, receive complete context, execute, report results with knowledge, and repeat — with structured escalation to humans when needed and clear visibility into what is happening at every step.

The specification resolves six open questions (§16) and defines acceptance criteria across six categories (§15). All Phase 4b features — decomposition, worker review, conflict analysis, automatic unblocking, incidents, and RCA — are explicitly deferred and will be specified under self-management after Phase 4a is validated.

**Implementation sequence:**

1. Estimation (pre-4a housekeeping — purely additive, no dependencies)
2. Dependency enforcement (enforcement rule + `work_queue` + `dependency_status`)
3. Task dispatch and completion (`dispatch_task` + `complete_task` + `human_checkpoint`)
4. Context assembly enhancements (extend existing `context_assemble`)
5. Health check extensions (three new categories)
6. Validate Phase 4a on a real workload
7. Begin Phase 4b under self-management