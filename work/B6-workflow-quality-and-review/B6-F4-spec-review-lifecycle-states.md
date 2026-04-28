# Review Lifecycle States Specification

| Document | Review Lifecycle States |
|----------|------------------------|
| Status   | Draft                  |
| Created  | 2026-03-28T00:26:29Z   |
| Updated  | 2026-03-28T00:26:29Z   |
| Related  | `work/design/code-review-workflow.md` §4, §12, §14.1 |

---

## 1. Purpose

This specification defines two new feature lifecycle states — `reviewing` and `needs-rework` — and the corresponding transition map changes that enforce a mandatory review gate before a feature can be marked done. It also removes the `developing → done` shortcut, ensuring every feature passes through a review step.

The work is scoped to **Feature D** of plan `P6-workflow-quality-and-review`.

---

## 2. Goals

1. Features can no longer transition directly from `developing` to `done`.
2. A `reviewing` state exists to signal that implementation is complete and review is in progress.
3. A `needs-rework` state exists to signal that review found blocking issues requiring further implementation work.
4. The transition map is updated consistently across the model, lifecycle validator, and smart-transition logic.
5. `ValidNextStates` exposes the new states so callers receive accurate guidance.
6. All lifecycle error messages for invalid transitions list the valid next states.
7. Health checks correctly interpret the new states as non-terminal where appropriate.
8. Smart-skip (`advance: true`) does not bypass the `reviewing` gate.
9. All existing tests continue to pass.

---

## 3. Scope

### 3.1 In scope

- Adding `FeatureStatusReviewing` and `FeatureStatusNeedsRework` constants to `internal/model/entities.go`.
- Updating the Phase 2 feature transition map in `internal/validate/lifecycle.go` to match the new map defined in §4 of the design.
- Removing the `developing → done` transition.
- Updating `ValidNextStates` to include the new states.
- Updating lifecycle error messages to include valid next states.
- Updating health check logic in `internal/health/` where it inspects feature lifecycle completeness (e.g. checks that distinguish terminal from non-terminal states, or checks for features with all-terminal children).
- Updating merge gate checks in `internal/merge/` that inspect feature status: `done` remains the required status before merge readiness; `reviewing` and `needs-rework` are non-terminal blocking states.
- Verifying that the `advance: true` smart-transition path stops at `reviewing` when advancing from `developing`.
- Adding or updating tests to cover all of the above.

### 3.2 Deferred

- UI or CLI rendering of the new states (display labels, colour coding).
- Automated creation of review tasks when a feature enters `reviewing`.
- Notifications or webhooks triggered by transitions to `reviewing` or `needs-rework`.

### 3.3 Explicitly excluded

- Changes to the task or bug lifecycle state machines.
- Changes to the merge gate infrastructure itself (merge gates are an upstream check; this feature only ensures `reviewing` and `needs-rework` are correctly treated as non-merge-ready states).
- Changes to the `work/spec/phase-*.md` family of documents.

---

## 4. Acceptance Criteria

### 4.1 New status constants

**AC-01.** `FeatureStatusReviewing` is defined in `internal/model/entities.go` with string value `"reviewing"`.

**AC-02.** `FeatureStatusNeedsRework` is defined in `internal/model/entities.go` with string value `"needs-rework"`.

### 4.2 Updated transition map

**AC-03.** The feature transition map in `internal/validate/lifecycle.go` matches the following exactly:

```
proposed     → designing, specifying, superseded, cancelled
designing    → specifying, superseded, cancelled
specifying   → dev-planning, designing, superseded, cancelled
dev-planning → developing, specifying, superseded, cancelled
developing   → reviewing, dev-planning, superseded, cancelled
reviewing    → done, needs-rework, superseded, cancelled
needs-rework → developing, reviewing, superseded, cancelled
```

**AC-04.** Requesting a transition from `developing` to `done` returns an error. The error message identifies `reviewing` as the correct next state (or lists all valid next states for `developing`).

**AC-05.** Requesting a transition from `reviewing` to `done` succeeds.

**AC-06.** Requesting a transition from `reviewing` to `needs-rework` succeeds.

**AC-07.** Requesting a transition from `needs-rework` to `developing` succeeds.

**AC-08.** Requesting a transition from `needs-rework` to `reviewing` succeeds (quick-fix path).

**AC-09.** All pre-existing valid transitions (those not involving the new states and not the removed `developing → done`) continue to succeed.

**AC-10.** All pre-existing invalid transitions continue to be rejected.

### 4.3 ValidNextStates

**AC-11.** `ValidNextStates` called with `developing` returns a set that includes `reviewing` and does **not** include `done`.

**AC-12.** `ValidNextStates` called with `reviewing` returns a set that includes `done` and `needs-rework`.

**AC-13.** `ValidNextStates` called with `needs-rework` returns a set that includes `developing` and `reviewing`.

### 4.4 Error messages

**AC-14.** When a feature transition is rejected because the target state is not reachable from the current state, the error message lists the valid next states for the current state.

**AC-15.** Specifically, attempting `developing → done` produces an error message that includes `"reviewing"` (or the full list of valid next states for `developing`).

### 4.5 Smart-skip (`advance: true`) composition

**AC-16.** When `advance: true` is applied to a feature in `developing` status, the feature advances to `reviewing` and stops there (i.e. it does **not** skip to `done`).

**AC-17.** When `advance: true` is applied to a feature in `reviewing` status with no blocking issues and all prerequisites satisfied, the feature may advance to `done`.

**AC-18.** The smart-skip chain never produces a direct `developing → done` transition, regardless of document prerequisites or other conditions.

### 4.6 Health checks

**AC-19.** Health checks that identify features with all-terminal children but a non-terminal feature status treat both `reviewing` and `needs-rework` as non-terminal. A feature in `reviewing` with all tasks done does **not** trigger a "stalled feature" or "forgotten completion" health warning.

**AC-20.** Health checks that identify terminal features with non-terminal children correctly exclude `reviewing` and `needs-rework` from the "terminal" category.

### 4.7 Merge gate integration

**AC-21.** The merge readiness check continues to require feature status `done` before a feature is considered merge-ready. A feature in `reviewing` or `needs-rework` fails the merge readiness check.

**AC-22.** The merge gate check does not need to be modified to implement AC-21 — the existing check against `done` is sufficient. No regression is introduced.

### 4.8 Regression

**AC-23.** `go test ./...` passes with no failures after all changes are applied.

**AC-24.** `go test -race ./...` passes with no data-race failures.

**AC-25.** `go vet ./...` reports no issues in modified packages.

---

## 5. Verification

| AC | Verification method |
|----|---------------------|
| AC-01 | Unit test: constant value equals `"reviewing"` |
| AC-02 | Unit test: constant value equals `"needs-rework"` |
| AC-03 | Unit test: iterate the transition map and assert each entry matches the expected set |
| AC-04 | Unit test: `Transition(developing, done)` returns non-nil error containing `"reviewing"` |
| AC-05 | Unit test: `Transition(reviewing, done)` returns nil error |
| AC-06 | Unit test: `Transition(reviewing, needs-rework)` returns nil error |
| AC-07 | Unit test: `Transition(needs-rework, developing)` returns nil error |
| AC-08 | Unit test: `Transition(needs-rework, reviewing)` returns nil error |
| AC-09 | Existing transition tests continue to pass without modification |
| AC-10 | Existing invalid-transition tests continue to pass without modification |
| AC-11 | Unit test: `ValidNextStates(developing)` contains `reviewing`, excludes `done` |
| AC-12 | Unit test: `ValidNextStates(reviewing)` contains `done` and `needs-rework` |
| AC-13 | Unit test: `ValidNextStates(needs-rework)` contains `developing` and `reviewing` |
| AC-14 | Unit test: error message from any rejected transition contains the valid-next-states list |
| AC-15 | Unit test: error from `developing → done` contains `"reviewing"` |
| AC-16 | Unit test / integration test: `advance` from `developing` lands at `reviewing` |
| AC-17 | Unit test / integration test: `advance` from `reviewing` with prerequisites met lands at `done` |
| AC-18 | Unit test: no `advance` path from `developing` reaches `done` in a single step |
| AC-19 | Unit test: health check does not flag `reviewing` feature with all-terminal tasks |
| AC-20 | Unit test: health check does not treat `reviewing` or `needs-rework` as terminal |
| AC-21 | Unit test: merge readiness check returns not-ready for features in `reviewing` or `needs-rework` |
| AC-22 | Code review: merge gate source unchanged, no regression introduced |
| AC-23 | CI: `go test ./...` green |
| AC-24 | CI: `go test -race ./...` green |
| AC-25 | CI: `go vet ./...` green |