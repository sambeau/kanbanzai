# Specification: Decompose Apply Supersession

| Field   | Value                                     |
|---------|-------------------------------------------|
| Date    | 2026-04-25                                |
| Status  | Draft                                     |
| Author  | Spec Author                               |
| Feature | FEAT-01KQ2E0RR40CD                        |
| Plan    | P34-agent-workflow-ergonomics             |

---

## 1. Related Work

**Design:** `work/design/design-p34-agent-workflow-ergonomics.md`
(`P34-agent-workflow-ergonomics/design-design-p34-agent-workflow-ergonomics`) —
§6 · Decompose apply — supersede prior queued task sets.

**Prior decisions and designs consulted:**

| Document | Relevance to this specification |
|----------|---------------------------------|
| P25 — Agent Tooling and Pipeline Quality | Addressed decompose reliability (empty names, dev-plan awareness). Did not change apply idempotency or task supersession. This specification closes that gap. |
| P19 — Workflow lifecycle integrity | Established `UpdateStatus` as the canonical method for transitioning task status. Supersession uses the same method to move `queued` tasks to `not-planned`. |

No prior specification was found for decompose apply supersession. The design's
Related Work section attests: "No directly related prior work was found" on this
specific concern.

---

## 2. Overview

This specification covers the supersession pass added to `decompose(action: apply)`
before new tasks are created. When a feature already has tasks from a prior apply
run, all tasks in `queued` status are transitioned to `not-planned` before the new
task set is written. Tasks in any other status are preserved. A `superseded_count`
field is added to the response, and a warning is surfaced when in-progress tasks
are detected. The result is an idempotent `decompose → apply` cycle: applying any
number of times to the same feature produces exactly one active `queued` task set.

---

## 3. Scope

**In scope:**

- A supersession pass at the start of `decomposeApply`, before Pass 1 (task
  creation), that transitions all existing `queued` tasks for the feature to
  `not-planned`.
- A partition rule that protects all tasks not in `queued` status.
- A `superseded_count` field in the `decompose(action: apply)` response.
- A `warning` field in the response when tasks in `active` or `needs-rework`
  status are detected and preserved.
- Idempotency guarantee: repeated apply runs produce exactly one `queued` task set.

**Out of scope:**

- Supersession of tasks in any status other than `queued` (ready, active,
  needs-rework, needs-review, done, not-planned, cancelled are all protected).
- Supersession during `decompose(action: propose)` — the proposal stage is
  read-only.
- Any change to Pass 1 (task creation logic) or Pass 2 (dependency wiring).
- Automatic clean-up of `not-planned` tasks from the dashboard.
- Changes to `decompose(action: review)`.

---

## 4. Functional Requirements

### Supersession pass

**FR-001** — When `decompose(action: apply)` is called, the system MUST execute a
supersession pass before creating any new tasks (before Pass 1).

**FR-002** — The supersession pass MUST list all tasks whose `parent_feature`
equals the target feature ID.

**FR-003** — The supersession pass MUST transition every task with `status ==
"queued"` to `status == "not-planned"` via `UpdateStatus`. These are the
**supersedable** tasks.

**FR-004** — The supersession pass MUST NOT modify any task whose status is any
value other than `"queued"`. The protected statuses are: `ready`, `active`,
`needs-rework`, `needs-review`, `done`, `not-planned`, `cancelled`.

### Response fields

**FR-005** — The `decompose(action: apply)` response MUST include a
`superseded_count` field containing the number of tasks transitioned to
`not-planned` by the supersession pass. When no tasks are superseded, the value
MUST be `0`.

**FR-006** — When one or more tasks with `status == "active"` or `status ==
"needs-rework"` exist for the feature at the time of the supersession pass, the
response MUST include a `warning` field with the message:
`"N task(s) in active/needs-rework status were preserved; verify they are still
needed."` where `N` is the count of such tasks.

**FR-007** — The presence of in-progress tasks (FR-006) MUST NOT block the
supersession pass or the subsequent task creation (Pass 1). The warning is
informational only.

### Idempotency

**FR-008** — After `decompose(action: apply)` completes successfully, the feature
MUST have exactly one set of tasks in `queued` status: those created by the current
apply run.

**FR-009** — Calling `decompose(action: apply)` on the same feature multiple times
with the same proposal MUST result in the same final state each time (modulo
already-progressed tasks that are protected).

### Task creation

**FR-010** — Pass 1 (new task creation) MUST proceed unchanged after the
supersession pass completes, creating the new task set as defined by the proposal.

---

## 5. Non-Functional Requirements

**NFR-001** — The supersession pass MUST complete before any new task is written.
Partial state (some old tasks superseded, new tasks not yet created) MUST NOT
persist if the operation is interrupted; failures in Pass 1 after a completed
supersession pass are tolerated (the superseded tasks remain `not-planned`).

**NFR-002** — The `superseded_count` and `warning` fields MUST be present in the
response map under exactly the keys `"superseded_count"` and `"warning"`.
`"superseded_count"` MUST always be present; `"warning"` MUST be omitted when
there are no in-progress tasks.

**NFR-003** — The change MUST NOT affect `decompose(action: propose)` or
`decompose(action: review)` in any way.

---

## 6. Acceptance Criteria

**AC-001 (FR-003, FR-005)** — Given a feature with 5 tasks all in `queued` status,
when `decompose(action: apply)` is called with a new proposal, then all 5 existing
tasks are transitioned to `not-planned` and the response contains
`superseded_count: 5`.

**AC-002 (FR-005)** — Given a feature with no existing tasks, when
`decompose(action: apply)` is called, then the response contains
`superseded_count: 0` and no `warning` field.

**AC-003 (FR-004, FR-005)** — Given a feature with 2 tasks in `done` status and
3 tasks in `queued` status, when `decompose(action: apply)` is called, then the
2 `done` tasks are unchanged, the 3 `queued` tasks are transitioned to
`not-planned`, and the response contains `superseded_count: 3`.

**AC-004 (FR-004, FR-006)** — Given a feature with 1 task in `active` status and
3 tasks in `queued` status, when `decompose(action: apply)` is called, then the
3 `queued` tasks are transitioned to `not-planned`, the `active` task is
unchanged, the response contains `superseded_count: 3`, and the response contains
a `warning` field referencing 1 in-progress task.

**AC-005 (FR-004)** — Given a feature with 1 task in `ready` status, when
`decompose(action: apply)` is called, then the `ready` task is preserved (not
transitioned to `not-planned`) and `superseded_count: 0` is returned.

**AC-006 (FR-007, FR-010)** — Given a feature with 1 task in `active` status,
when `decompose(action: apply)` is called, then the new task set is created
successfully (Pass 1 is not blocked), the warning is present, and the `active`
task remains untouched.

**AC-007 (FR-008, FR-009)** — Given `decompose(action: apply)` is called three
times in succession on the same feature (with the same proposal each time), then
after the third call the feature has exactly one set of tasks in `queued` status
(matching the proposal), and all prior `queued` task sets from earlier runs are in
`not-planned` status.

**AC-008 (FR-006)** — Given a feature with 2 tasks in `needs-rework` status, when
`decompose(action: apply)` is called, then both tasks are preserved, the warning
field references 2 in-progress tasks, and `superseded_count: 0` is returned.

---

## 7. Dependencies and Assumptions

**DEP-001** — `UpdateStatus` must support transitioning a task from `queued` to
`not-planned`. This transition must be permitted by the task lifecycle state
machine.

**DEP-002** — A task listing operation scoped to `parent_feature` must be
available within `decomposeApply` to enumerate existing tasks before Pass 1 runs.

**ASM-001** — `not-planned` is the correct terminal status for a task that existed
but was superseded before work began. It is distinct from `cancelled` (which
implies the task was in flight) and from `done` (which implies completion).

**ASM-002** — The `ready` status represents explicit promotion intent (either by
an agent or by the auto-promotion hook). Superseding `ready` tasks would silently
undo deliberate state changes; this is why `ready` is protected rather than
supersedable (Design Decision 5).

**ASM-003** — The supersession pass and Pass 1 execute within a single
`decompose(action: apply)` call. There is no distributed transaction; if Pass 1
fails after the supersession pass completes, the old tasks remain in `not-planned`
and the caller must retry.