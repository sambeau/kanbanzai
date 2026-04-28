# Specification: Task Auto-Promotion on Developing

| Field   | Value                          |
|---------|--------------------------------|
| Date    | 2026-04-25                     |
| Status  | Draft                          |
| Author  | Spec Author                    |
| Feature | FEAT-01KQ2E0RFNBWZ             |
| Plan    | P34-agent-workflow-ergonomics  |

---

## 1. Related Work

**Design:** `work/design/design-p34-agent-workflow-ergonomics.md`
(`P34-agent-workflow-ergonomics/design-design-p34-agent-workflow-ergonomics`) —
§2 · Task auto-promotion on `developing` transition.

**Prior decisions and designs consulted:**

| Document | Relevance to this specification |
|----------|---------------------------------|
| P19 — Workflow lifecycle integrity | Introduced `OnStatusTransition` hook and the auto-advance mechanism that promotes a parent feature when all child tasks reach terminal state. This specification implements the inverse: child tasks are promoted when their parent feature advances to `developing`. The same hook contract applies: hook failures are best-effort and MUST NOT block the triggering transition. |
| P29 — State Store Read Path Performance | `PromoteQueuedTasks` calls the cache-backed `ListTasksForFeature` operation. The O(1) list performance introduced in P29 makes synchronous promotion within the transition request acceptable. |

No prior specification was found for child-task auto-promotion on parent feature
transition. The design's Related Work section attests: "No directly related prior
work was found on plan ID prefix shorthand resolution or idempotent task claiming"
(the auto-promotion mechanism is confirmed novel in this codebase).

---

## 2. Overview

This specification covers the automatic promotion of dependency-free tasks from
`queued` to `ready` when their parent feature transitions to `developing`. A new
function `PromoteQueuedTasks` is registered as a behaviour on the existing
`OnStatusTransition` hook. Tasks with no unmet dependencies are promoted
immediately; tasks whose dependencies are not yet complete are left in `queued`
for the existing dependency auto-promotion logic to handle when those dependencies
finish.

---

## 3. Scope

**In scope:**
- A new function `PromoteQueuedTasks(featureID string)` in the entity service.
- Registration of `PromoteQueuedTasks` as a hook behaviour on the
  `OnStatusTransition` event when a feature transitions to `developing`.
- Correct identification of tasks whose `depends_on` list is fully satisfied
  (empty, or all entries in a terminal status).

**Out of scope:**
- Changes to the existing dependency auto-promotion logic that promotes tasks
  when their individual dependencies complete.
- Promotion of tasks on any feature transition other than `→ developing`.
- Introduction of any new lifecycle status names or changes to existing ones.
- Promotion of tasks in any status other than `queued`.

---

## 4. Functional Requirements

### Hook registration

**FR-001** — When a feature transitions to `developing`, the system MUST call
`PromoteQueuedTasks(featureID)` via the `OnStatusTransition` hook, after the
feature's new status has been written.

**FR-002** — `PromoteQueuedTasks` MUST NOT be called when a feature transitions to
any status other than `developing`.

**FR-003** — A failure inside `PromoteQueuedTasks` MUST be logged and MUST NOT
cause the feature's `→ developing` transition to return an error. The hook
contract inherited from P19 requires best-effort execution.

### Task discovery

**FR-004** — `PromoteQueuedTasks` MUST list all tasks whose `parent_feature`
equals the given `featureID` and whose `status` is `queued`.

**FR-005** — Tasks in any status other than `queued` (e.g. `ready`, `active`,
`done`, `not-planned`, `cancelled`) MUST NOT be modified by `PromoteQueuedTasks`.

### Dependency evaluation

**FR-006** — For each `queued` task, `PromoteQueuedTasks` MUST inspect its
`depends_on` list. If the list is empty, the task MUST be transitioned
`queued → ready`.

**FR-007** — If a task's `depends_on` list is non-empty and every entry in the
list is in a terminal status (`done`, `not-planned`, or `cancelled`), the task
MUST be transitioned `queued → ready`.

**FR-008** — If a task's `depends_on` list contains any entry that is not in a
terminal status, the task MUST be left in `queued`. The existing dependency
auto-promotion logic will promote it when its blocking dependency completes.

### Failure isolation

**FR-009** — If transitioning an individual task from `queued` to `ready` fails,
the error MUST be logged and the loop MUST continue to the next task. A partial
promotion result is preferable to no promotion.

### Idempotency

**FR-010** — Calling `PromoteQueuedTasks` more than once for the same feature
MUST be safe. On the second call, tasks already in `ready` or any non-`queued`
status are not touched (FR-005 ensures this).

---

## 5. Non-Functional Requirements

**NFR-001** — `PromoteQueuedTasks` MUST use the cache-backed task list operation
(introduced in P29) to enumerate tasks for the feature. It MUST NOT introduce a
new O(n) file-scan code path for listing tasks.

**NFR-002** — `PromoteQueuedTasks` MUST run synchronously within the transition
request, consistent with how worktree auto-creation is implemented on the same
hook.

**NFR-003** — The implementation MUST reuse the existing `UpdateStatus` method for
each individual task transition, so that the standard status-change lifecycle
logic (validation, hook firing, persistence) is applied uniformly.

---

## 6. Acceptance Criteria

**AC-001 (FR-001, FR-006)** — Given a feature with two `queued` tasks that have
no `depends_on` entries, when the feature transitions to `developing`, then both
tasks are transitioned to `ready`.

**AC-002 (FR-007)** — Given a feature with a `queued` task whose `depends_on`
list contains only tasks in `done` status, when the feature transitions to
`developing`, then that task is transitioned to `ready`.

**AC-003 (FR-008)** — Given a feature with a `queued` task whose `depends_on`
list contains a task still in `queued` status, when the feature transitions to
`developing`, then that task remains in `queued` after the transition.

**AC-004 (FR-005)** — Given a feature with a task already in `ready` status,
when the feature transitions to `developing`, then that task's status is
unchanged.

**AC-005 (FR-002)** — Given a feature transitioning to a status other than
`developing` (e.g. `reviewing`), when tasks are in `queued` status, then no
task promotion occurs.

**AC-006 (FR-003)** — Given that `PromoteQueuedTasks` encounters an error when
transitioning one task, when the feature transitions to `developing`, then the
feature transition still succeeds, the error is logged, and promotion continues
for remaining tasks.

**AC-007 (FR-010)** — Given a feature already in `developing` with tasks already
in `ready`, when `PromoteQueuedTasks` is called a second time for the same
feature, then no task statuses change and no error is returned.

**AC-008 (FR-001, FR-008)** — Given a feature with one dependency-free `queued`
task and one `queued` task blocked by a non-terminal dependency, when the feature
transitions to `developing`, then only the dependency-free task is promoted to
`ready`; the blocked task remains `queued`.

---

## 7. Dependencies and Assumptions

**DEP-001** — The `OnStatusTransition` hook mechanism (introduced in P19) must
support multiple registered behaviours. `PromoteQueuedTasks` is registered
alongside the existing worktree auto-creation behaviour.

**DEP-002** — A cache-backed list operation for tasks filtered by `parent_feature`
and `status` must be available, consistent with the P29 read-path performance
work.

**DEP-003** — `UpdateStatus` must be callable for individual task transitions
within `PromoteQueuedTasks`. Its existing lifecycle validation must treat
`queued → ready` as a valid transition.

**ASM-001** — Terminal statuses are exactly `done`, `not-planned`, and
`cancelled`. No new terminal status is introduced by this feature.

**ASM-002** — `depends_on` entries reference task slugs or IDs that are
resolvable at the time `PromoteQueuedTasks` runs. Unresolvable entries are treated
as non-terminal (conservative: the task is not promoted).

**ASM-003** — The `OnStatusTransition` hook fires after the feature's new status
has been durably written, so tasks promoted in the hook observe the feature as
already in `developing`.