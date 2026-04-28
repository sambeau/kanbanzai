# Phase 4b Specification: Self-Managed Capabilities

| Document | Phase 4b Specification |
|----------|------------------------|
| Status   | Draft                  |
| Created  | 2026-03-25             |
| Updated  | 2026-03-25             |
| Related  | `work/plan/phase-4-scope.md` §7         |
|          | `work/plan/phase-4-decision-log.md`     |
|          | `work/spec/phase-4a-specification.md`   |
|          | `work/design/workflow-design-basis.md`  |

---

## 1. Purpose

This specification defines the requirements for Phase 4b of the Kanbanzai workflow system. Phase 4b delivers the richer orchestration capabilities that were deferred from Phase 4a, developed using the Phase 4a self-management tooling to validate the system managing its own construction.

Phase 4b builds on:

- **Phase 4a** — estimation, dependency enforcement, work queue, task dispatch and completion, human checkpoints
- **Phase 3** — Git integration, worktree management, GitHub PR, knowledge lifecycle
- **Phase 2b** — context profiles, knowledge contribution, confidence scoring
- **Phase 2a** — document intelligence, structural analysis, section tracing

---

## 2. Goals

1. **Close the decomposition gap** — An orchestrating agent can take a feature with a specification document and produce a proposed task list, reviewed against the spec before any tasks are written.

2. **Remove manual unblocking** — When a task completes, any newly unblocked tasks transition automatically. The orchestrator is notified rather than having to poll.

3. **Automate output review** — A completed task's output can be checked against its acceptance criteria and the parent feature's spec before it reaches human review.

4. **Make parallel work safer** — Before dispatching two tasks in parallel, the orchestrator can assess whether they are likely to conflict at the file or architectural boundary level.

5. **Surface vertical slice structure** — Given a feature, the system can identify how it could be cut into end-to-end thin slices, each independently testable, to support better decomposition decisions.

6. **Track incidents and root causes** — Production-significant failures have a first-class entity with MTTR-measurable timestamps, a linked structured RCA document, and health checks that surface open incidents without root cause analysis.

---

## 3. Scope

### 3.1 In scope for Phase 4b

- **Feature decomposition:** `decompose_feature` and `decompose_review` MCP tools; vertical slice guidance embedded in the decomposition tool; CLI command `kbz feature decompose`
- **Automatic dependency unblocking:** `StatusTransitionHook` extension that transitions newly unblocked tasks from `blocked` to `ready` when a dependency reaches a terminal state; `complete_task` response extended with `unblocked_tasks`
- **Worker review:** `review_task_output` MCP tool; integration with the document intelligence section-tracing pipeline; `needs-rework` task transition with recorded failure reason; CLI command `kbz task review`
- **Conflict domain analysis:** `conflict_domain_check` MCP tool; optional `--conflict-check` annotation on `work_queue`; CLI command `kbz queue --conflict-check`
- **Vertical slice decomposition guidance:** `slice_analysis` MCP tool; guidance embedded in `decompose_feature`; knowledge entries tagged with slice origin
- **Incidents and RCA:** `Incident` entity type with full lifecycle; `RootCauseAnalysis` document type; health check for unlinked incidents post-resolution; MCP tools `incident_create`, `incident_update`, `incident_list`, `incident_link_bug`; CLI commands `kbz incident create/list/show`

### 3.2 Deferred beyond Phase 4b

- Automated incident detection (pattern matching on bug clusters, health check failures)
- Cross-project knowledge sharing
- Multi-repository conflict domain analysis
- Automated MTTR reporting and dashboards
- RCA-triggered remediation tasks (auto-creating follow-up tasks from RCA findings)

### 3.3 Explicitly excluded

- Webhook-based real-time synchronisation
- GitLab, Bitbucket, or other platform support
- Semantic search or embedding-based retrieval
- Web or desktop UI

---

## 4. Design Principles

### 4.1 Decomposition is a preview, not a write

`decompose_feature` produces a proposed task list for review. Nothing is written to state until explicitly confirmed via a human checkpoint or an orchestrator acting with a confirmed checkpoint response. Proposals that are never confirmed disappear without side effects.

### 4.2 Automatic unblocking is conservative

The hook only transitions tasks from `blocked` to `ready`. It never transitions to `active`. It fires only when all `depends_on` entries are in terminal state. Partial unblocking (some deps done, some not) produces no transition.

### 4.3 Worker review is advisory at Tier 2

`review_task_output` returns structured findings. A failed review transitions the task to `needs-rework` — it does not delete work or auto-revert anything. A passing review transitions to `needs-review` for human sign-off; it does not bypass the human gate.

### 4.4 Conflict analysis is probabilistic

`conflict_domain_check` surfaces risk, not certainty. Its output is a risk level with supporting evidence (shared files, overlapping planned files, architectural boundary crossing). The orchestrator decides whether to proceed in parallel, serialise, or escalate to a human checkpoint.

### 4.5 Incidents are not bugs

An incident represents a production-significant failure event, which may be caused by one or more bugs, or by operational factors with no bug at all. The `Incident` entity carries MTTR-relevant timestamps and links bugs as contributing factors. It does not replace or extend Bug.

---

## 5. Approved Design Decisions

Phase 4b is governed by the following accepted decisions in `work/plan/phase-4-decision-log.md`:

| Decision | Topic | Key Choice |
|----------|-------|------------|
| P4-DES-001 | Phase split | Phase 4b developed under self-management using Phase 4a tooling |
| P4-DES-003 | Self-management threshold | Phase 4b is the first phase built entirely inside the system |
| P4-DES-004 | Dependency modelling | Automatic unblocking via `StatusTransitionHook` (Phase 4b) |
| P4-DES-006 | Incidents and RCA | `Incident` entity + `RootCauseAnalysis` document type |
| P4-DES-007 | Document store deprecation | Phase 1 doc store removed at Phase 4b start; path-reference model only |

---

## 6. Feature Decomposition

### 6.1 Purpose

Decomposition is the step between a feature with a specification and a backlog of actionable tasks. Currently it is purely human work. `decompose_feature` makes this a tool-assisted process: the system proposes a task list from the spec, applies vertical slice principles and size guidance, and requires confirmation before writing anything.

`decompose_review` provides a second-pass quality check on any proposed decomposition, checking it against the spec for gaps and overlaps before the orchestrator confirms.

### 6.2 `decompose_feature`

**Input:**

```
feature_id     string   required   FEAT ID of the feature to decompose
context        string   optional   Additional guidance for the decomposition (passed as orchestration_context)
```

**Behaviour:**

1. Load the feature and resolve its `spec` document reference. If no spec document is registered on the feature, return an error: `feature FEAT-xxx has no linked specification document`.
2. Retrieve the spec document content via the Phase 2a document record path (`path` field in the document record).
3. Apply the embedded decomposition guidance (§6.5) to produce a proposed task list.
4. Return the proposal as a preview. **Do not write any tasks.**

**Output:**

```
feature_id         string
feature_slug       string
spec_document_id   string
proposal:
  tasks:
    - slug          string   proposed slug
      summary       string   one-line task summary
      role          string   suggested role profile (if determinable)
      estimate      number   suggested story point estimate (if determinable)
      depends_on    []string slugs of other proposed tasks this one depends on
      rationale     string   why this task exists and what slice it belongs to
  total_tasks       int
  estimated_total   number   sum of suggested estimates (null if any are absent)
  slices:           []string names of identified vertical slices
  warnings:         []string gaps, oversized tasks, unclear dependencies
guidance_applied:   []string list of decomposition rules that influenced the output
```

### 6.3 `decompose_review`

**Input:**

```
feature_id   string   required   FEAT ID
proposal     object   required   The proposal object from decompose_feature output
```

**Behaviour:**

1. Load the feature's spec document (same as `decompose_feature`).
2. Check the proposal against the spec for: uncovered acceptance criteria, tasks with no traceable spec section, tasks whose summary is ambiguous, task estimates above the soft limit (13 points), and dependency cycles within the proposal.
3. Return structured findings.

**Output:**

```
feature_id    string
status        string   "pass" | "fail" | "warn"
findings:
  - type       string   "gap" | "overlap" | "oversized" | "ambiguous" | "cycle"
    task_slug  string   the affected proposed task (or null for feature-level findings)
    detail     string   human-readable description
total_findings   int
blocking_count   int   count of findings that should block confirmation
```

### 6.4 Confirmation and task creation

After reviewing a proposal and resolving findings, the orchestrator confirms by calling `create_task` for each proposed task in the correct dependency order. This is intentional: task creation remains a first-class operation, not a side effect of `decompose_feature`. The proposal provides the inputs; the orchestrator drives the writes.

For automated confirmation, the orchestrator SHOULD raise a `human_checkpoint` before writing tasks from a proposal, allowing a human to review before anything is persisted.

### 6.5 Embedded decomposition guidance

The following rules are applied by `decompose_feature` and reported in `guidance_applied`:

1. **Vertical slice first.** Each task should produce end-to-end value through the stack (data → logic → interface), not a horizontal layer (e.g., "all database changes"). Prefer tasks that can be independently tested.
2. **One acceptance criterion per task.** If the spec has five acceptance criteria for a feature, the decomposition should have at most five tasks (often fewer through grouping).
3. **Size soft limit.** No proposed task should exceed 8 story points. Tasks estimated above 8 are flagged as oversized.
4. **Explicit dependencies.** If task B requires output from task A, the dependency must be declared in `depends_on`. Do not leave implicit ordering.
5. **Role assignment.** If the project has context profiles, map each task to the most appropriate role. Tasks that span roles should be split or flagged.
6. **Test tasks are explicit.** If the spec requires tests (the default), include a test task or explicitly include testing in the relevant implementation task summary.

### 6.6 CLI

```
kbz feature decompose <feature-id>
  Propose a task decomposition for a feature. Prints the proposal.
  Does not write any tasks.
  Example: kbz feature decompose FEAT-01JX

kbz feature decompose <feature-id> --confirm
  After reviewing the proposal, write the proposed tasks.
  Raises a human checkpoint before writing. Exits if checkpoint is denied.
  Example: kbz feature decompose FEAT-01JX --confirm
```

---

## 7. Automatic Dependency Unblocking

### 7.1 Purpose

In Phase 4a, the orchestrator discovers newly-unblocked tasks by calling `work_queue`, which promotes `queued` tasks to `ready` as a side effect. Tasks that are explicitly `blocked` (manually set) are never automatically promoted. Phase 4b removes the distinction: when any task reaches a terminal state, the system checks all tasks with a `depends_on` reference to it and promotes any that are now fully unblocked.

### 7.2 StatusTransitionHook extension

The existing `StatusTransitionHook` in `EntityService.UpdateStatus` fires after every successful task status transition. Phase 4b extends this hook with a `DependencyUnblockingHook` that fires when a task transitions to `done`, `not-planned`, or `duplicate` (terminal states).

**Hook behaviour:**

1. Load all tasks with a `depends_on` entry containing the just-completed task ID.
2. For each candidate task in `blocked` or `queued` status: check whether all entries in its `depends_on` list are now in terminal state.
3. If all dependencies are terminal: transition the task from `blocked` or `queued` to `ready`. This is a system-initiated transition; it bypasses the dependency enforcement gate (which it already satisfies by construction).
4. If the transition succeeds: add the task ID and slug to the hook result's `unblocked_tasks` list.
5. If the transition fails for any reason: log a warning; do not fail the original transition.

**Failure isolation:** A failure in the hook must never cause the original transition (task completing) to fail. Hook errors are warnings, not errors.

### 7.3 `complete_task` response extension

The `CompleteResult` and the `complete_task` MCP tool response gain an `unblocked_tasks` field:

```
unblocked_tasks:
  - task_id    string
    slug       string
    status     string   always "ready"
```

When no tasks are unblocked, `unblocked_tasks` is an empty array. It is never omitted.

### 7.4 Interaction with `work_queue`

`work_queue` continues to promote `queued` tasks as before. The automatic hook handles `blocked` tasks. Both paths converge on `ready` status. An orchestrator can rely on either path to surface work; the hook simply removes the need to poll for previously-blocked tasks.

---

## 8. Worker Review

### 8.1 Purpose

When an agent completes a task, the output (code, documents, configuration) should be checked against what was actually asked for before it enters the human review queue. `review_task_output` automates this first-pass check using the task's verification criteria, the parent feature's spec, and optionally the actual output artefacts.

### 8.2 `review_task_output`

**Input:**

```
task_id              string     required   TASK ID of the completed or active task
output_files         []string   optional   Paths of files produced or modified by this task
output_summary       string     optional   Agent's description of what was done
```

**Behaviour:**

1. Load the task. If status is not `active`, `done`, or `needs-review`, return an error.
2. Load the parent feature and resolve its `spec` document reference. If no spec is registered, proceed with task-level checks only (no spec-level checks are performed; a warning is added to findings).
3. **Task-level check:** Verify that the task's `verification` field criteria are addressed. If `output_files` is provided, check that each file exists on disk. If `output_summary` is provided, check that it addresses the task summary.
4. **Spec-level check** (if spec available): Use the document intelligence section-tracing pipeline to find spec sections relevant to this task. Check that the `output_summary` and `output_files` plausibly satisfy those sections. This is heuristic — findings are "possible gap" rather than "definite gap".
5. **Aggregate result:** If any blocking findings exist, the review fails. If only warnings exist, the review passes with warnings.

**Output:**

```
task_id        string
task_slug      string
status         string   "pass" | "pass_with_warnings" | "fail"
findings:
  - severity   string   "error" | "warning"
    type       string   "missing_file" | "verification_unmet" | "spec_gap" | "no_spec"
    detail     string
total_findings   int
blocking_count   int
```

**State transitions triggered by `review_task_output`:**

- `status: fail` → task transitions to `needs-rework`; `rework_reason` field set to a summary of blocking findings
- `status: pass` or `status: pass_with_warnings` → task transitions to `needs-review` (human sign-off queue)
- If the task is already in `needs-review` or `done`: the review runs and returns findings but does not trigger a further transition

### 8.3 `rework_reason` task field

A new optional field `rework_reason` is added to the Task schema to record why a task was sent back for rework. It is set by `review_task_output` on a failing review and cleared when the task is reactivated (`needs-rework → active`).

### 8.4 Document intelligence integration

Section tracing is provided by the existing `internal/docint/` pipeline. `review_task_output` uses `doc_trace` to find spec sections whose headings or content match the task summary and slug. This is a best-effort lookup; if `doc_trace` returns no matches, the spec-level check is skipped and a `no_spec_sections_found` warning is added.

### 8.5 CLI

```
kbz task review <task-id> [--files <path,...>] [--summary "<text>"]
  Run worker review on a task.
  Example: kbz task review TASK-01JX
  Example: kbz task review TASK-01JX --files internal/auth/jwt.go --summary "Implemented RS256 middleware"
```

---

## 9. Conflict Domain Analysis

### 9.1 Purpose

Parallel agent work is a core Phase 3/4 capability, but parallel tasks may collide — editing the same files, touching the same architectural boundaries, or requiring a specific execution order that is not captured in `depends_on`. `conflict_domain_check` surfaces this risk before dispatch so the orchestrator can make an informed decision.

### 9.2 `conflict_domain_check`

**Input:**

```
task_ids   []string   required   Two or more TASK IDs to check for conflict risk
```

**Behaviour:**

1. For each task, collect: `files_planned` field, parent feature slug, parent feature's spec document reference, and the worktree branch (if a worktree exists for the parent feature).
2. For each pair of tasks: run the three analysis dimensions (§9.3).
3. Aggregate pair-level results into an overall risk assessment.

**Output:**

```
task_ids       []string
overall_risk   string   "none" | "low" | "medium" | "high"
pairs:
  - task_a     string
    task_b     string
    risk       string   "none" | "low" | "medium" | "high"
    dimensions:
      file_overlap:
        risk          string
        shared_files  []string   files in both tasks' files_planned
        git_conflicts []string   files with recent conflicting edits on both branches
      dependency_order:
        risk    string
        detail  string
      boundary_crossing:
        risk    string
        detail  string
    recommendation   string   "safe_to_parallelise" | "serialise" | "checkpoint_required"
```

### 9.3 Analysis dimensions

**File overlap:** Compare `files_planned` lists between tasks. If the same file appears in both, this is a direct overlap (medium risk minimum). If worktree branches exist for both tasks, run `git log --name-only` on each branch since its creation and report files with edits on both branches (high risk).

**Dependency order:** Check whether either task's `depends_on` list references the other, or whether a path exists through `depends_on` chains connecting the two. A direct or transitive ordering dependency means the tasks should not run in parallel.

**Architectural boundary crossing:** Using the parent feature slugs and spec document subjects, assess whether the tasks touch the same architectural area (e.g., both modifying authentication, both touching the storage layer). This is a heuristic based on keyword matching in summaries and spec sections. Flag as low risk when boundaries overlap, medium if both tasks have substantial overlap in spec sections.

### 9.4 `work_queue` integration

`work_queue` gains an optional boolean parameter `conflict_check`. When `true`, for each ready task, the tool checks it against all currently `active` tasks and annotates the result with a `conflict_risk` field.

```
# Extended work_queue item when conflict_check: true
task_id          string
slug             string
...existing fields...
conflict_risk    string   "none" | "low" | "medium" | "high"   (omitted when conflict_check not requested)
conflict_with    []string   task IDs of active tasks that conflict
```

### 9.5 CLI

```
kbz queue [--role <profile>] [--conflict-check]
  Show the current work queue. With --conflict-check, annotates each task
  with its conflict risk against currently active tasks.
  Example: kbz queue --conflict-check
```

---

## 10. Vertical Slice Decomposition Guidance

### 10.1 Purpose

Vertical slice thinking is embedded in `decompose_feature` (§6.5), but a dedicated `slice_analysis` tool provides a standalone analysis of a feature's slice structure without committing to a full decomposition. This supports planning conversations and helps the orchestrator understand the feature's shape before choosing a decomposition strategy.

### 10.2 `slice_analysis`

**Input:**

```
feature_id   string   required   FEAT ID
```

**Behaviour:**

1. Load the feature and its spec document.
2. Identify the feature's primary user-visible outcomes from the spec (acceptance criteria, goal statements).
3. For each outcome, identify the stack layers it touches (storage, service, MCP/API, CLI/UI).
4. Group outcomes into candidate slices: coherent end-to-end cuts that each deliver independent, testable value.
5. Identify dependencies between slices (slice B requires slice A's storage layer to exist).

**Output:**

```
feature_id     string
feature_slug   string
slices:
  - name          string   short name for this slice
    outcomes      []string spec outcomes covered by this slice
    layers        []string stack layers touched
    estimate      string   rough relative size: "small" | "medium" | "large"
    depends_on    []string names of slices this one depends on
    rationale     string
total_slices    int
analysis_notes  string   overall observations about sliceability
```

### 10.3 Knowledge tagging

When tasks are created from a decomposition that used `slice_analysis` output, the creating agent SHOULD include the slice name in the task's `tags` field (e.g., `slice:auth-flow`). Knowledge entries contributed via `complete_task` on those tasks inherit the slice tag, enabling future conflict domain analysis to use slice provenance.

---

## 11. Incidents and Root Cause Analysis

### 11.1 Purpose

Per P4-DES-006, production-significant failures warrant a structured entity with MTTR-measurable timestamps, links to contributing bugs, and a linked RCA document whose approved content feeds the knowledge store. The `Incident` entity and `RootCauseAnalysis` document type implement this.

### 11.2 Incident entity

**ID prefix:** `INC` (TSID13-based, e.g., `INC-01JX...`)

**Lifecycle:**
```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
```

**Terminal states:** `closed`

**Severity levels:** `critical` | `high` | `medium` | `low`

**Schema:**

```yaml
id:                  INC-01JX...
slug:                payment-service-outage-2026-03
title:               Payment service outage
status:              resolved
severity:            high
reported_by:         sam
detected_at:         "2026-03-20T14:30:00Z"
triaged_at:          "2026-03-20T14:45:00Z"   # optional
mitigated_at:        "2026-03-20T16:00:00Z"   # optional
resolved_at:         "2026-03-20T18:00:00Z"   # optional
affected_features:                             # optional
  - FEAT-01JX...
linked_bugs:                                   # optional
  - BUG-01JX...
linked_rca:          DOC-01JX...              # optional, set when RCA is approved
summary:             Payment service returned 503 for 90 minutes due to database connection pool exhaustion.
created:             "2026-03-20T14:30:00Z"
created_by:          sam
updated:             "2026-03-20T18:00:00Z"
```

**Canonical field order:** `id`, `slug`, `title`, `status`, `severity`, `reported_by`, `detected_at`, `triaged_at`, `mitigated_at`, `resolved_at`, `affected_features`, `linked_bugs`, `linked_rca`, `summary`, `created`, `created_by`, `updated`

**Lifecycle transition rules:**

| From | To | Notes |
|------|----|-------|
| `reported` | `triaged` | — |
| `reported` | `closed` | If trivial / false alarm |
| `triaged` | `investigating` | — |
| `triaged` | `closed` | If not reproducible |
| `investigating` | `root-cause-identified` | — |
| `root-cause-identified` | `mitigated` | — |
| `root-cause-identified` | `investigating` | If root cause is revised |
| `mitigated` | `resolved` | — |
| `mitigated` | `investigating` | If mitigation was incomplete |
| `resolved` | `closed` | After RCA is approved |
| Any non-closed | `closed` | Override path |

### 11.3 RootCauseAnalysis document type

**Document type string:** `rca`

**Lifecycle:** Follows the existing document pipeline — `submitted → normalised → approved`. On approval, key findings are contributed to the knowledge store.

**Required front-matter fields:**
```
incident_ids: []string   one or more INC IDs this RCA covers
severity:     string     highest severity incident covered
```

**On approval:** The approving agent SHOULD contribute the following as Tier 2 knowledge entries with scope `project`:
- The root cause (topic: `rca-root-cause-{incident-slug}`)
- The mitigation steps (topic: `rca-mitigation-{incident-slug}`)
- Any systemic findings (topic: `rca-systemic-{incident-slug}`)

Contribution follows the standard `knowledge_contribute` path. The RCA document ID is stored on each contributed entry as `learned_from`.

After RCA approval, the linked incident's `linked_rca` field is set to the RCA document ID and its status may be advanced to `resolved` or `closed` by the human.

### 11.4 Health check extension

A new health check category `unlinked_resolved_incidents` checks for incidents in `resolved` or `root-cause-identified` status that have no `linked_rca` and whose `resolved_at` (or `updated`) is older than a configurable threshold.

- Severity: `warning`
- Default threshold: 7 days after resolution
- Config: `incidents.rca_link_warn_after_days` (default: 7; 0 disables)

### 11.5 MCP tools

**`incident_create`**

```
slug          string   required
title         string   required
severity      string   required   "critical" | "high" | "medium" | "low"
summary       string   required
reported_by   string   required
detected_at   string   optional   ISO 8601; defaults to now
```

Returns the created incident record.

**`incident_update`**

```
incident_id        string   required
status             string   optional   new lifecycle status
severity           string   optional
summary            string   optional
triaged_at         string   optional
mitigated_at       string   optional
resolved_at        string   optional
affected_features  []string optional   replaces existing list
```

Returns the updated incident record.

**`incident_list`**

```
status    string   optional   filter by status
severity  string   optional   filter by severity
```

Returns a list of incidents with summary fields.

**`incident_link_bug`**

```
incident_id   string   required
bug_id        string   required
```

Adds the bug to the incident's `linked_bugs` list. Idempotent.

### 11.6 CLI

```
kbz incident create --slug <slug> --title "<title>" --severity <level> --summary "<text>" --reported_by <identity>
  Create a new incident.
  Example: kbz incident create --slug db-pool-exhaustion --title "DB pool exhaustion" --severity high --summary "..." --reported_by sam

kbz incident list [--status <status>] [--severity <level>]
  List incidents with optional filters.
  Example: kbz incident list --status investigating

kbz incident show <incident-id>
  Show a full incident record.
  Example: kbz incident show INC-01JX
```

---

## 12. Storage Model

### 12.1 New entity type: Incident

Stored in `.kbz/state/incidents/INC-{TSID13}-{slug}.yaml`. Field order per §11.2. The `incidents` directory is created on first write.

ID allocation uses the existing TSID13 allocator with prefix `INC`.

### 12.2 Task schema additions

One new optional field is added to the Task schema:

```
rework_reason   string   optional   set by review_task_output on fail; cleared on active transition
```

**Updated canonical field order** (additions in bold):

`id`, `parent_feature`, `slug`, `summary`, `status`, `estimate`, `assignee`, `depends_on`, `files_planned`, `started`, `completed`, `claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`, `completion_summary`, **`rework_reason`**, `verification`, `tags`

### 12.3 CompleteResult schema extension

`complete_task` MCP response gains `unblocked_tasks` (§7.3). This is a response-only field; it is not stored on the task entity.

### 12.4 WorkQueueItem schema extension

`work_queue` response items gain optional `conflict_risk` and `conflict_with` fields when `conflict_check: true` is passed (§9.4). These are response-only fields.

### 12.5 New document type: rca

The `rca` document type is registered alongside existing types (`design`, `specification`, `dev-plan`, etc.). It follows the same document pipeline. Front-matter must include `incident_ids` (list of INC IDs) and `severity`.

### 12.6 Deterministic formatting

All new records must pass round-trip serialisation tests (write → read → write → compare). Incident records use block-style YAML per P1-DEC-008. All optional fields use `omitempty`; `rework_reason` on Task uses `omitempty`.

---

## 13. MCP Interface Summary

Phase 4b adds the following MCP tools:

| Tool | Feature | Description |
|------|---------|-------------|
| `decompose_feature` | Feature Decomposition | Produce a proposed task list from a feature's spec |
| `decompose_review` | Feature Decomposition | Review a proposal against the spec for gaps and overlaps |
| `slice_analysis` | Vertical Slices | Identify vertical slice structure in a feature |
| `review_task_output` | Worker Review | Check task output against acceptance criteria and spec |
| `conflict_domain_check` | Conflict Analysis | Assess conflict risk between tasks proposed for parallel execution |
| `incident_create` | Incidents/RCA | Create a new incident |
| `incident_update` | Incidents/RCA | Update incident fields or status |
| `incident_list` | Incidents/RCA | List incidents with optional filters |
| `incident_link_bug` | Incidents/RCA | Link a bug to an incident |

Existing tools extended:

| Tool | Change |
|------|--------|
| `complete_task` | Response gains `unblocked_tasks` |
| `work_queue` | Gains optional `conflict_check` parameter; items gain `conflict_risk` and `conflict_with` when enabled |
| `health_check` | New category `unlinked_resolved_incidents` |

---

## 14. CLI Interface Summary

Phase 4b adds the following CLI commands:

```
kbz feature decompose <id> [--confirm]
kbz task review <id> [--files <paths>] [--summary "<text>"]
kbz queue [--role <profile>] [--conflict-check]
kbz incident create --slug --title --severity --summary --reported_by
kbz incident list [--status] [--severity]
kbz incident show <id>
```

---

## 15. Configuration

Phase 4b adds two configuration keys under `incidents`:

```yaml
incidents:
  rca_link_warn_after_days: 7   # days after resolution before warning; 0 disables
```

And one key under `decomposition`:

```yaml
decomposition:
  max_tasks_per_feature: 20   # soft limit on proposed tasks; proposals above this produce a warning
```

---

## 16. Acceptance Criteria

### 16.1 Feature decomposition

- [ ] `decompose_feature` returns a proposal without writing any tasks
- [ ] `decompose_feature` returns an error when the feature has no linked spec document
- [ ] Proposal includes `slug`, `summary`, `estimate`, `depends_on`, and `rationale` for each proposed task
- [ ] Proposal includes identified vertical slices and any warnings
- [ ] `guidance_applied` lists the decomposition rules that influenced the output
- [ ] `decompose_review` detects uncovered spec acceptance criteria and reports them as `gap` findings
- [ ] `decompose_review` detects tasks above 8 points and reports them as `oversized` findings
- [ ] `decompose_review` detects dependency cycles within the proposal and reports them as `cycle` findings
- [ ] `decompose_review` returns `status: pass` for a well-formed proposal
- [ ] `decompose_review` returns `status: fail` when blocking findings exist
- [ ] CLI `kbz feature decompose <id>` prints the proposal without writing tasks
- [ ] CLI `kbz feature decompose <id> --confirm` creates a human checkpoint before writing tasks

### 16.2 Automatic dependency unblocking

- [ ] When a task transitions to `done`, all tasks whose `depends_on` includes it are evaluated
- [ ] A task with all dependencies in terminal state transitions from `blocked` to `ready` automatically
- [ ] A task with some but not all dependencies terminal is not transitioned
- [ ] `complete_task` response includes `unblocked_tasks` listing any tasks promoted by this completion
- [ ] `complete_task` response includes `unblocked_tasks` as an empty array when no tasks are unblocked
- [ ] A failure in the unblocking hook does not cause the original task transition to fail
- [ ] `not-planned` and `duplicate` terminal states satisfy dependency checks identically to `done`

### 16.3 Worker review

- [ ] `review_task_output` returns `status: fail` when `output_files` contains a file that does not exist on disk
- [ ] `review_task_output` transitions the task to `needs-rework` and sets `rework_reason` on a failing review
- [ ] `review_task_output` transitions the task to `needs-review` on a passing review
- [ ] `review_task_output` includes spec-level findings when the feature has a linked spec document
- [ ] `review_task_output` adds a `no_spec` warning finding when no spec document is registered on the feature
- [ ] `rework_reason` is cleared when a task transitions from `needs-rework` to `active`
- [ ] `review_task_output` on a task already in `needs-review` or `done` returns findings without triggering a state transition
- [ ] Round-trip serialisation of Task with `rework_reason` field produces identical output

### 16.4 Conflict domain analysis

- [ ] `conflict_domain_check` requires at least two task IDs; returns an error with fewer
- [ ] File overlap is detected when the same path appears in both tasks' `files_planned`
- [ ] A direct `depends_on` ordering between two tasks is reported as a `dependency_order` risk
- [ ] `conflict_domain_check` returns `recommendation: serialise` when dependency ordering is detected
- [ ] `work_queue` with `conflict_check: true` annotates each ready task with `conflict_risk` and `conflict_with`
- [ ] `work_queue` without `conflict_check` behaves identically to Phase 4a (no new fields)

### 16.5 Vertical slice analysis

- [ ] `slice_analysis` returns an error when the feature has no linked spec document
- [ ] `slice_analysis` identifies at least one slice for any feature with a multi-criterion spec
- [ ] Each slice includes `name`, `outcomes`, `layers`, `estimate`, and `rationale`
- [ ] Inter-slice dependencies are identified when one slice's outputs are required by another

### 16.6 Incidents and RCA

- [ ] `incident_create` creates an incident in `reported` status with all required fields
- [ ] `incident_create` rejects invalid severity values with a clear error
- [ ] `incident_update` enforces lifecycle transition rules; invalid transitions return a clear error
- [ ] `incident_link_bug` adds the bug to `linked_bugs`; calling it twice with the same bug is idempotent
- [ ] `incident_list` with `status` filter returns only matching incidents
- [ ] `incident_list` with no filter returns all incidents
- [ ] RCA document type is accepted by `submit_document`
- [ ] Health check `unlinked_resolved_incidents` flags incidents in `resolved` status older than `rca_link_warn_after_days`
- [ ] Health check is suppressed when `rca_link_warn_after_days: 0`
- [ ] Round-trip serialisation of Incident entity in canonical field order produces identical output
- [ ] Incident ID uses `INC-` prefix with TSID13

### 16.7 Phase 1 document store removal

- [x] `internal/document/` package is removed
- [x] `.kbz/docs/` directory is removed from the repository
- [x] `doc` CLI command group is removed
- [x] `submit_document`, `approve_document`, `list_documents`, `scaffold_document`, `validate_document` MCP tools are removed or replaced by Phase 2a document record equivalents
- [x] `TestServer_ListTools` in `internal/mcp/server_test.go` is updated to reflect removed tools
- [x] The Phase 4b spec and implementation plan are registered as Phase 2a document records (path reference + content hash) before any removal occurs — this is the validation gate per P4-DES-007

---

## 17. Implementation Notes

### 17.1 Phase 4b begins with document store removal

Before implementing any Phase 4b feature tracks, execute the P4-DES-007 removal:

1. Register this spec and the Phase 4b implementation plan as Phase 2a document records.
2. Verify they can be retrieved via `get_entity` on the document record.
3. Remove `internal/document/`, `.kbz/docs/`, and the associated CLI/MCP surface.
4. Update `TestServer_ListTools`.
5. Confirm `go test -race ./...` passes before proceeding.

This establishes a clean foundation and validates the document record path before any feature work depends on it.

### 17.2 Automatic unblocking hook integration

The `DependencyUnblockingHook` must be added to the `StatusTransitionHook` chain in `EntityService.UpdateStatus`. The hook fires after the transition is committed. It must not acquire locks or perform long-running operations — it reads and writes entity state using the same `EntityService`, which is already protected by the per-request serialisation model.

Test the hook with at least: a task completing with no dependents (no-op), a task completing that fully unblocks one other, a task completing that partially unblocks (not all deps terminal), and a chain of three tasks where completing A unblocks B but not C until B also completes.

### 17.3 Decomposition as a pure read with structured output

`decompose_feature` must not write any state. Its output is a structured proposal returned to the caller. There is no "pending decomposition" entity. If the caller closes the session, the proposal is gone. This is intentional: the only durable state from decomposition is the tasks the orchestrator creates after confirmation.

### 17.4 Review tool heuristics

The spec-level check in `review_task_output` is heuristic. It uses `doc_trace` from `internal/docint/` which may return imprecise matches. The tool should never report `status: fail` solely on the basis of a spec-level finding — spec findings are always `warning` severity. Only task-level findings (missing files, verification field unmet) produce `error` severity that can cause a `fail` result.

### 17.5 Incident ID prefix registration

`INC` must be added to the `TypePrefix` function in `internal/id/allocator.go` and to `EntityKindFromPrefix`. A new `EntityKindIncident` constant must be defined in `internal/model/entities.go`. The `incidents` directory is created automatically by the store on first write, matching the behaviour of other entity directories.

### 17.6 Phase 4a punch list items

The Phase 4a punch list items (PL-1 through PL-5 in `work/plan/phase-4a-implementation-plan.md`) should be resolved before Phase 4b feature implementation begins. The CLI commands (PL-2) and MCP integration tests (PL-3) in particular are necessary for the Phase 4b CLI to be built on a clean foundation.

### 17.7 Phase 4b validation

Phase 4b is itself developed under self-management (P4-DES-001). Each feature track should be implemented as a dispatched task and completed via `complete_task`. The `review_task_output` tool, once implemented, should be used to review its own subsequent tasks — i.e., worker review reviews the implementation of conflict domain analysis, etc.

---

## 18. Summary

Phase 4b delivers six feature tracks that complete the orchestration model and add operational depth:

| Track | Feature | Core deliverable |
|-------|---------|-----------------|
| 1 | Feature Decomposition | `decompose_feature` + `decompose_review`; vertical slice guidance; confirmation before write |
| 2 | Automatic Dependency Unblocking | `StatusTransitionHook` extension; `complete_task` gains `unblocked_tasks` |
| 3 | Worker Review | `review_task_output`; `needs-rework` transition; spec-section tracing |
| 4 | Conflict Domain Analysis | `conflict_domain_check`; `work_queue --conflict-check` annotation |
| 5 | Vertical Slice Guidance | `slice_analysis`; guidance embedded in `decompose_feature`; slice tagging |
| 6 | Incidents and RCA | `Incident` entity; `RootCauseAnalysis` document type; health check |

Phase 4b also executes the P4-DES-007 document store removal as a first step, registering its own planning documents as the validation gate before removing the legacy Phase 1 store.

The specification resolves the main open questions from Phase 4a (§16 of that spec) and defines 44 acceptance criteria across seven categories. Implementation begins with the document store removal, followed by the Phase 4a punch list, followed by the six feature tracks.

**Gate for Phase 5:** All Phase 4b acceptance criteria verified, all Phase 4a punch list items resolved, `go test -race ./...` clean, no blocking health check errors.