# Dev Plan: Plan Lifecycle Proposed-to-Active Gap

**Feature:** FEAT-01KPVDDYVETV5
**Plan:** P28 — Doc-Intel Polish and Workflow Reliability
**Spec:** work/spec/p28-plan-lifecycle-proposed-active.md
**Status:** Draft

---

## Scope

This dev plan covers the implementation of a direct `proposed → active` transition for the
plan state machine (specification: `work/spec/p28-plan-lifecycle-proposed-active.md`).

The work adds a first-class `proposed → active` shortcut, guarded by a precondition that at
least one of the plan's features is in a post-designing lifecycle state (`specifying`,
`dev-planning`, `developing`, `reviewing`, or `done`). When the precondition is not met the
transition is rejected with a descriptive error directing the caller to use `proposed →
designing`. When it fires successfully a system-generated override record is written to the
entity's audit trail. The existing `proposed → designing → active` path is completely
unaffected.

Out of scope: changes to transitions from any state other than `proposed`; changes to the
`designing → active` gate; feature-level lifecycle changes; UI/CLI changes.

---

## Task Breakdown

### Task 1: Add `proposed → active` transition and precondition check

- **Description:** Extend the plan state machine to accept `proposed → active` as a legal
  first-class transition. Implement the in-flight features precondition: query the entity
  index for all features belonging to the plan and check whether at least one is in the
  post-designing set (`specifying`, `dev-planning`, `developing`, `reviewing`, `done`). If
  the precondition is satisfied, allow the transition to proceed; the caller does NOT need
  to supply `override: true`. Keep the `proposed → designing` and `proposed → superseded`/
  `proposed → cancelled` arcs entirely unchanged. Also keep the `designing → active` gate
  (approved design document check) entirely unchanged.
- **Deliverable:** Updated state machine / transition handler code in `internal/` accepting
  `proposed → active` when the precondition passes; precondition query helper.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirements:** REQ-001, REQ-002, REQ-005, REQ-006

### Task 2: Rejection error and system-generated override record

- **Description:** Using the infrastructure from Task 1, implement the two outcome paths.
  Rejection path: when the precondition check returns false (no features, all features at
  `proposed` or `designing`), return a descriptive error that includes a directive to use
  the `proposed → designing` path instead. Success path: after the transition fires, append
  a system-generated override record to the entity's audit trail. The record text MUST match
  exactly: `"proposed → active shortcut: N feature(s) in post-designing state at transition
  time"` where N is the count of qualifying features at the moment of the call. The record
  MUST be written to the same override-record store used by human-authored `override_reason`
  entries. The prefix `"proposed → active shortcut:"` MUST be hardcoded so that tooling can
  identify it programmatically.
- **Deliverable:** Rejection error with directive; system-generated override record written
  on success; both paths reachable via `entity(action: "transition")`.
- **Depends on:** Task 1
- **Effort:** Small
- **Spec requirements:** REQ-003, REQ-004, REQ-NF-001, REQ-NF-002, REQ-NF-003

### Task 3: Unit tests for transition, rejection, and override record

- **Description:** Add unit tests covering all AC-specified scenarios:
  - Happy path: plan with one `specifying` feature, no `override` flag → status becomes
    `active` (AC-001).
  - Happy path: plan with features at `specifying`, `developing`, and `done` → transition
    succeeds (AC-002).
  - Rejection: plan with all features at `designing` → error returned (AC-003).
  - Rejection: plan with no features → error contains `proposed → designing` directive
    (AC-004).
  - Rejection: plan with all features at `proposed` → error references `proposed →
    designing` (AC-005).
  - Override record: shortcut on plan with exactly two qualifying features → fetched entity
    carries record text matching `"proposed → active shortcut: 2 feature(s) in post-designing
    state at transition time"` (AC-006).
  - Prefix distinguishability: system-generated record has prefix `"proposed → active
    shortcut:"` (AC-007).
  - Regression: `proposed → designing` still succeeds for a plan with no qualifying features
    (AC-008).
  - Regression: `designing → active` without approved design doc still rejected (AC-009).
  - Regression: `designing → active` with approved design doc still succeeds (AC-010).
- **Deliverable:** New/updated test file(s) in `internal/` covering all ten unit-test
  scenarios above.
- **Depends on:** Task 1, Task 2
- **Effort:** Medium
- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006

### Task 4: Integration tests and latency verification

- **Description:** Add integration tests that exercise the full stack end-to-end:
  - Create a plan and features in various states; verify transition fires or rejects
    correctly as state composition changes (AC-012 — fresh state: transition a feature to
    `specifying`, immediately attempt the shortcut, confirm N reflects the newly-qualified
    feature).
  - Verify the existing `proposed → designing → active` path is unaffected end-to-end.
  - Time the shortcut transition wall-clock under local SQLite single-user load and assert
    it completes within 2 seconds (AC-011).
- **Deliverable:** Integration test(s) in `internal/` or the appropriate test package; latency
  assertion or benchmark confirming ≤ 2 s under local load.
- **Depends on:** Task 3
- **Effort:** Medium
- **Spec requirements:** REQ-NF-002, REQ-NF-003

---

## Dependency Graph

```
Task 1 (state machine transition + precondition check)
    └─► Task 2 (rejection error + override record)
            └─► Task 3 (unit tests)
                    └─► Task 4 (integration tests + latency)
```

Tasks are strictly sequential. Each task builds directly on the infrastructure of the previous.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Merge conflict with other P28 tasks that touch the plan state machine | Medium | Medium | Sequence this feature's branch after any concurrent state-machine work; inspect `internal/` state-machine files for concurrent modifications before starting. |
| Precondition query reads stale cached feature state | Low | High | REQ-NF-003 requires live index reads; ensure no cache layer sits between the precondition check and the entity index. Add AC-012 integration test to catch any regression. |
| System-generated override record indistinguishable from human records | Low | Medium | Hardcode the prefix `"proposed → active shortcut:"` and add AC-007 assertion in Task 3. |
| `designing → active` gate inadvertently weakened | Low | High | Task 3 regression tests (AC-009, AC-010) explicitly verify the gate is unchanged. |
| Latency regression from precondition query on large feature lists | Low | Low | Precondition only needs one qualifying feature; short-circuit on first match to keep the query O(1) in the happy path. |

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 — shortcut succeeds with one qualifying feature, no override flag | Unit test | Task 3 |
| AC-002 — shortcut succeeds when mix of qualifying states present | Unit test | Task 3 |
| AC-003 — rejected when all features at `designing` | Unit test | Task 3 |
| AC-004 — rejected with directive when no features exist | Unit test | Task 3 |
| AC-005 — rejected with directive when all features at `proposed` | Unit test | Task 3 |
| AC-006 — override record text matches prescribed pattern with N = 2 | Integration test | Task 4 |
| AC-007 — system-generated record identifiable by hardcoded prefix | Inspection + unit test | Task 3 |
| AC-008 — `proposed → designing` unaffected (regression) | Unit test | Task 3 |
| AC-009 — `designing → active` gate unchanged without approved doc | Unit test | Task 3 |
| AC-010 — `designing → active` gate unchanged with approved doc | Unit test | Task 3 |
| AC-011 — shortcut completes within 2 s under local SQLite load | Integration test / benchmark | Task 4 |
| AC-012 — freshly-qualified feature counted in N (no stale state) | Integration test | Task 4 |