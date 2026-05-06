# Procedure: Review Remediation Workflow

| Field    | Value                                                     |
|----------|-----------------------------------------------------------|
| Plan     | P54-review-remediation-workflow                           |
| Type     | procedure                                                 |
| Status   | Draft                                                     |
| Author   | sambeau (orchestrator)                                    |
| Date     | 2026-05-06                                                |
| Scope    | Orchestrator-facing — converting a failed review into remediation |

---

## Overview

This document defines the orchestrator-facing procedure for the review remediation workflow. When a formal review report returns an aggregate verdict of `fail`, this procedure converts the blocking findings into a remediation dev-plan, execution tasks, verification evidence, and a re-review report — without mutating the original review.

The workflow is document-driven (no new tool actions required). The orchestrator uses existing Kanbanzai tools (`doc`, `entity`, `decompose`, `finish`, `status`, `next`, `handoff`) to execute every step.

**When to use this procedure:** A review report (from any review stage — feature review, batch conformance review, plan-level review) has been approved with an aggregate verdict of `fail` and the orchestrator needs to convert it into structured, traceable remediation work.

**When not to use:** The review is still in draft (C-01: must be approved). The verdict is `pass` or `pass-with-notes` — no remediation is needed (AC-01.2).

**Prerequisites:**

- The review report document is registered and approved: `doc(action: "get", id: "REVIEW-DOC-ID")` shows `status: approved`.
- The review report's verdict is `fail`.
- The orchestrator has access to the affected feature/batch/plan entities.

---

## Phase 1: Entry Point — Read and Verify the Failed Review

### Step 1.1: Confirm the review report is approved

```text
doc(action: "get", id: "<review-doc-id>")
```

If the review is not approved, stop — draft reviews have no binding findings (C-01).

### Step 1.2: Extract the blocking findings

Read the review report content:

```text
doc(action: "content", id: "<review-doc-id>")
```

Identify every **blocking finding** in the report. Blocking findings are those classified as `fail` or `blocking` — they prevent the review from passing. Non-blocking findings (advisory, `pass-with-notes`) can be noted but do not require remediation tasks.

For each blocking finding, record:

| Field            | Source in review report                  |
|------------------|------------------------------------------|
| Finding ID       | The finding's unique identifier (e.g. BF-1, F-3) |
| Summary          | One-line description of the issue        |
| Affected entity  | Feature, batch, or plan that owns the problematic code/document |
| Spec reference   | The spec requirement or AC that is violated (if cited) |
| Evidence location| File path, line range, or document section |

**Tool calls used:** `doc(action: "get")`, `doc(action: "content")`

**Example (from P50 batch conformance review):**

| Finding ID | Summary | Affected Entity | Spec Reference |
|---|---|---|---|
| BF-1 | Code never committed — exists as dirty worktree state | B1-p51-exec (batch) | FR-02 |
| BF-2 | Worktree isolation violated | FEAT-01Kxxx (feature) | FR-03 |
| ... | ... | ... | ... |

### Step 1.3: Group findings by ownership scope

Sort the findings into three buckets based on what entity is affected:

1. **Single-feature:** findings scoped to one feature entity (FEAT-xxx).
2. **Batch-level:** findings that span multiple features within one batch, or that are about the batch itself.
3. **Cross-cutting:** findings that span multiple batches or represent a reusable workflow gap (e.g. "all features lack verification evidence").

This grouping determines the ownership model in Phase 3.

---

## Phase 2: Scope Inspection

### Step 2.1: Check entity scope and dirty state

For each affected entity, verify its current state. If P53 (Infrastructure Hygiene) is available, use its scope inspection tooling. If not (C-02 fallback), perform manual checks:

**With P53:**
```text
# P53 scope inspection (when available)
status(id: "<entity-id>")
```

**Without P53 (manual fallback):**
```text
# Check entity lifecycle state
entity(action: "get", id: "<entity-id>")

# Check worktree status (for feature entities)
worktree(action: "get", entity_id: "<entity-id>")

# Check for uncommitted changes in worktree
# (manual: cd to worktree path, git status)
```

Record any of the following in the remediation dev-plan's Risk Assessment section:
- Dirty working tree (uncommitted changes)
- Entity in unexpected lifecycle state
- Stale worktree that hasn't been cleaned up

### Step 2.2: Confirm the review report is the latest

Check that no newer review report exists for the same entity:

```text
doc(action: "list", owner: "<entity-id>", type: "report")
```

If a newer report exists with a different verdict, stop and re-evaluate — the remediation may already be handled or the situation may have changed.

---

## Phase 3: Determine Ownership Model and Create Entities

### Step 3.1: Apply the ownership decision tree

Based on the finding grouping from Step 1.3, apply the ownership model (FR-04):

```
┌─────────────────────────────────────────────────────────┐
│  Are ALL findings scoped to a SINGLE feature entity?     │
│                                                          │
│  YES ──► Single-feature model                            │
│          • Create remediation tasks under that FEAT-xxx  │
│          • Transition feature to needs-rework (if not    │
│            already)                                       │
│          • Register dev-plan under the feature            │
│                                                          │
│  NO ──► Do findings span MULTIPLE features               │
│         within ONE batch?                                │
│                                                          │
│         YES ──► Batch-level model                        │
│                 • Register dev-plan under the batch       │
│                 • Create per-feature tasks under each     │
│                   affected FEAT-xxx                       │
│                 • Transition affected features to         │
│                   needs-rework                            │
│                                                          │
│         NO ──► Cross-cutting model                       │
│                 • Create a new plan entity                │
│                 • Register dev-plan under the new plan    │
│                 • Create a batch under the plan for       │
│                   execution                               │
└─────────────────────────────────────────────────────────┘
```

### Step 3.2: Create entities if needed

**Single-feature model:**
```text
# If the feature is not already in a reworkable state:
entity(action: "transition", id: "<FEAT-xxx>", status: "needs-rework")
```

**Batch-level model:**
```text
# For each affected feature:
entity(action: "transition", id: "<FEAT-xxx>", status: "needs-rework")
```

**Cross-cutting model:**
```text
# Create a new plan
entity(action: "create", type: "strategic-plan", id: "P<n>-<slug>",
       name: "Remediation: <summary>", summary: "<description>",
       status: "active")

# Create a batch under the plan
entity(action: "create", type: "batch", id: "B<n>-<slug>",
       name: "Remediation Batch: <summary>", parent: "P<n>-<slug>")
```

---

## Phase 4: Produce the Remediation Dev-Plan

### Step 4.1: Create the dev-plan document

Write the remediation dev-plan following the six-section structure (FR-02). Use the template at [`P54-template-remediation-dev-plan.md`](P54-template-remediation-dev-plan.md) and populate it from the review report findings.

The six required sections are:

1. **Scope** — Cite the original review report ID, list the affected entities, and reference the spec documents that were violated.
2. **Task Breakdown** — One task or task group per blocking finding. Each task names the finding(s) it addresses.
3. **Dependency Graph** — Ordering for fixes that must land before tests or re-review.
4. **Risk Assessment** — Risks specific to this remediation: dirty trees, lifecycle drift, scope ambiguity, unavailability of dependent plans.
5. **Verification Approach** — How each finding will be verified as resolved (tests, manual inspection, re-review checklist).
6. **Traceability Matrix** — Original finding ID → remediation task(s) → verification method. Every blocking finding MUST appear. Deferred findings require an explicit closure decision with rationale and owner (FR-03, AC-03.2).

### Step 4.2: Register and approve the dev-plan

```text
# Determine the document path:
# - Single-feature: work/<feature-slug>/<feature-id>-dev-plan-remediation-<slug>.md
# - Batch-level:    work/<batch-slug>/<batch-id>-dev-plan-remediation-<slug>.md
# - Cross-cutting:  work/<plan-slug>/<plan-id>-dev-plan-remediation-<slug>.md

doc(action: "register", path: "<path>", type: "dev-plan",
    title: "Remediation Dev-Plan: <summary>", owner: "<entity-id>")

doc(action: "approve", id: "<doc-id>")
```

---

## Phase 5: Create Remediation Tasks

### Step 5.1: Decompose the dev-plan into tasks

Use the standard `decompose` workflow (C-04):

```text
decompose(action: "propose", feature_id: "<FEAT-xxx>")
# Review the proposal
decompose(action: "review", feature_id: "<FEAT-xxx>", proposal: <proposal-object>)
# Apply to create tasks
decompose(action: "apply", feature_id: "<FEAT-xxx>", proposal: <proposal-object>)
```

**If `decompose` cannot parse the dev-plan format** (known limitation with certain AC formats — see BUG-01KPVGMMP56GC), create tasks directly:

```text
entity(action: "create", type: "task", parent_feature: "<FEAT-xxx>",
       name: "<task name>", summary: "<task description>",
       depends_on: ["<dependency-task-id>", ...])
```

### Step 5.2: Verify the Traceability Matrix is complete

After task creation, cross-check:
- Every blocking finding from the original review has at least one associated task or a deferral with rationale.
- No finding is silently omitted (AC-SPEC-02).
- Task dependencies match the dev-plan's Dependency Graph.

---

## Phase 6: Execute Remediation

### Step 6.1: Dispatch implementation tasks

Remediation tasks follow the standard implementation lifecycle (C-03). Dispatch them using the normal orchestrator-workers pattern:

```text
# Claim and dispatch each ready task
next(id: "<TASK-xxx>")

# For sub-agent dispatch, use handoff with the correct role
handoff(task_id: "<TASK-xxx>", role: "implementer-go")
# → spawn_agent with the handoff output
```

When dispatching via `handoff`, ensure the role is `implementer-go` (not `orchestrator`). P51 ensures sub-agents receive the correct skill context (FR-08, AC-SPEC-05).

### Step 6.2: Monitor progress

Use `status(id: "<FEAT-xxx>")` to track task completion. Each task follows the standard path: `active → done` (or `needs-review` if review is required).

Key fields to check in `status` output:

- **`task_summary`** — confirms the count of tasks in each state (`queued`, `ready`, `active`, `done`). All remediation tasks should eventually reach `done: <N>` with zero in `queued`, `ready`, and `active`.
- **`attention`** — watch for `ready_tasks` entries disappearing as tasks are claimed and completed. New `attention` items (e.g., `blocked_tasks`) signal issues needing intervention.
- **`progress`** — track `progress_pct` approaching 100% as tasks complete.

If a task remains in `active` or `ready` for an extended period, check the entity record for dispatch details: `entity(action: "get", id: "<TASK-xxx>")`.

### Step 6.3: Verify each finding

As tasks complete, confirm that verification evidence exists for each finding (FR-05):

- **AC-05.1:** Task completion alone is not sufficient — there must be explicit verification evidence (test results, inspection notes, documentation updates).
- **AC-05.2:** Each resolved finding must be cited in the re-review report with its finding ID.

---

## Phase 7: Produce the Re-Review Report

### Step 7.1: Create the re-review report

When all remediation tasks are terminal and verification evidence is collected, produce a re-review report using the template at [`P54-template-re-review-report.md`](P54-template-re-review-report.md).

The re-review report reuses the `report` document type with `review_remediation` subtype (FR-07). Required contents:

- Original review report ID and citation
- Per-finding resolution status table (resolved / deferred) with finding IDs
- Verification evidence for each resolved finding
- Aggregate resolution verdict

### Step 7.2: Register the re-review report

```text
doc(action: "register", path: "<path>", type: "report",
    title: "Re-Review Report: <summary>", owner: "<entity-id>")
```

Note: The document type is `report`. The `review_remediation` subtype is metadata that distinguishes re-review reports from original reviews.

### Step 7.3: Verify original review immutability

Confirm the original review report has not been modified (FR-06):

```text
doc(action: "get", id: "<original-review-doc-id>")
# Verify: status is still "approved", content_hash is unchanged
```

The original review report remains immutable evidence of the failed state. Resolution is recorded in the re-review report, not by editing or superseding the original (AC-06.1, AC-06.2).

---

## Phase 8: Close-Out

### Step 8.1: Transition affected entities

**Single-feature model:**
```text
# If all remediation tasks are done and re-review is approved:
entity(action: "transition", id: "<FEAT-xxx>", status: "reviewing")
# Proceed through standard review → merge → done lifecycle
```

**Batch-level model:**
```text
# Transition each affected feature back through review
# When all features are done, the batch can advance to reviewing
```

**Cross-cutting model:**
```text
# The new plan follows its own lifecycle independently
```

### Step 8.2: Verify the audit trail

Walk the full chain in both directions (NFR-02, AC-SPEC-07):

**Forward trace:**
```
Original Review → finding ID → Remediation Dev-Plan (Traceability Matrix) →
Remediation Task(s) → Verification Evidence → Re-Review Report → finding resolved
```

**Backward trace:**
```
Re-Review Report → finding ID → Original Review → finding description →
Remediation Dev-Plan → Task(s) → Implementation commit(s)
```

Every link must be traversable. If any link is broken, the remediation is incomplete.

### Step 8.3: Close out the remediation

- All remediation tasks are `done`
- Re-review report is approved
- Original review is unchanged and still `approved`
- Affected entities are back in a normal lifecycle state
- Clean up worktrees: `cleanup(action: "execute")` for merged features

---

## Tool Reference

This procedure uses only existing Kanbanzai tools (NFR-03, AC-SPEC-08):

| Phase | Tools Used |
|-------|------------|
| Entry Point | `doc(action: "get")`, `doc(action: "content")` |
| Scope Inspection | `status`, `entity(action: "get")`, `worktree(action: "get")` |
| Ownership Model | `entity(action: "transition")`, `entity(action: "create")` |
| Dev-Plan | `doc(action: "register")`, `doc(action: "approve")` |
| Task Creation | `decompose(action: "propose/review/apply")`, `entity(action: "create")` |
| Execution | `next`, `handoff`, `finish`, `status` |
| Re-Review | `doc(action: "register")`, `doc(action: "get")` |
| Close-Out | `entity(action: "transition")`, `cleanup(action: "execute")` |

No new MCP tool actions are required for the document-driven v1.

---

## Integration Points

### P51 — Handoff Reliability

When dispatching remediation implementation tasks via `handoff`, use `role: "implementer-go"` to ensure sub-agents receive the correct skill context (not the orchestrator role). P51's role routing fix makes this reliable (FR-08).

### P52 — Session-Start Audit

P52's session-start audit detects features in remediation state (`needs-rework`) and surfaces them. This procedure does not duplicate P52 — it consumes P52's output when available (FR-09).

### P53 — Scope Inspection

When P53 is available, use its scope inspection and dirty-work attribution tools in Phase 2. When P53 is not available, use the manual fallback steps described in Step 2.1 (FR-10, C-02).

---

## Constraints

1. **C-01:** The original review report must be approved before remediation begins. A draft review has no binding findings.
2. **C-02:** If P53 is not implemented, perform scope inspection manually as described in Step 2.1.
3. **C-03:** Remediation tasks follow the standard implementation task lifecycle — no special remediation-only states.
4. **C-04:** Use `decompose` for task creation. Fall back to direct `entity(action: "create")` only if decompose cannot parse the dev-plan format.
5. **C-05:** Automated finding extraction is deferred to P44 Phase 1. The document-driven procedure is the canonical path until then.

---

## Example: P50 Remediation

The P50 batch conformance review produced 10 blocking findings (BF-1 through BF-10) against the B1-p51-exec batch. The remediation followed this procedure manually:

- **Ownership:** Batch-level (findings spanned 4 features within B1-p51-exec).
- **Dev-Plan:** `P50-dev-plan-review-remediation.md` with all six sections populated.
- **Tasks:** Per-feature remediation tasks under each affected feature.
- **Re-Review:** Re-review report citing BF-1 through BF-10 with resolution status.

The P50 walkthrough validated that the procedure produces complete coverage (all 10 findings mapped to tasks), correct ownership (batch-level dev-plan, per-feature tasks), and a traversable audit trail.

---

## See Also

- [P54 Design: Review Remediation Workflow](P54-design-review-remediation-workflow.md)
- [P54 Specification: Review Remediation Workflow](P54-spec-review-remediation-workflow.md)
- [P54 Dev-Plan: Review Remediation Workflow](P54-dev-plan-review-remediation-workflow.md)
- [P54 Remediation Dev-Plan Template](P54-template-remediation-dev-plan.md)
- [P54 Re-Review Report Template](P54-template-re-review-report.md)
- [P54 Ownership Model Guide](P54-guide-ownership-model.md)
