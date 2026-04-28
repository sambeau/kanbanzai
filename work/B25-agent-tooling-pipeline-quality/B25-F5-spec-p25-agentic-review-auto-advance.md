# Specification: Agentic Reviewing Stage Auto-Advance

**Feature:** FEAT-01KPQ08YE4399
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-agentic-review-auto-advance.md
**Status:** Draft

---

## Overview

This specification defines the behaviour of an opt-out reviewing stage gate for agentic
pipelines. A new project-level config flag, `require_human_review`, controls whether
`AdvanceFeatureStatus` halts unconditionally at the `reviewing` stage or auto-advances past
it when all tasks are terminal and all tasks have recorded verification. When the flag is
absent or `false`, the reviewing stage is skipped automatically in agentic pipelines.
When the flag is `true`, the existing mandatory-halt behaviour is preserved exactly.

---

## Scope

### In Scope

- New `RequireHumanReview *bool` field on `MergeConfig` in `internal/config/config.go`.
- Accessor method `RequiresHumanReview() bool` on `MergeConfig`.
- New `RequiresHumanReview func() bool` field on `AdvanceConfig` in
  `internal/service/advance.go`.
- Modified halt-state branch in `AdvanceFeatureStatus` to check the new config flag and,
  when review is not required, validate auto-advance conditions before continuing.
- New helper `checkAllTasksHaveVerification` in `internal/service/prereq.go`.
- Updated `AdvanceConfig` injection in `internal/mcp/entity_tool.go` to read
  `cfg.Merge.RequiresHumanReview`.

### Out of Scope

- Removing the `reviewing` stage from the lifecycle or state machine.
- Changing the `reviewing→done` transition gate (`checkReviewReportExists`).
- Propagating task verification fields to the feature entity (FEAT-01KPQ08Y989P8).
- Feature-level `require_human_review` config granularity.
- The `needs-rework` cycle logic beyond what is described in this spec.

---

## Functional Requirements

**FR-001:** `MergeConfig` in `internal/config/config.go` MUST include a new field
`RequireHumanReview *bool` with YAML key `require_human_review` and `omitempty`.

**Acceptance criteria:**
- A `.kbz/config.yaml` that omits `require_human_review` parses without error and leaves
  the field as `nil`.
- A `.kbz/config.yaml` with `require_human_review: true` sets the field to a non-nil
  pointer to `true`.
- A `.kbz/config.yaml` with `require_human_review: false` sets the field to a non-nil
  pointer to `false`.

---

**FR-002:** `MergeConfig` MUST expose an accessor method `RequiresHumanReview() bool` that
returns `true` if and only if `RequireHumanReview` is non-nil and `*RequireHumanReview` is
`true`. Nil and pointer-to-false MUST both return `false`.

**Acceptance criteria:**
- `RequiresHumanReview()` returns `false` when `RequireHumanReview` is nil.
- `RequiresHumanReview()` returns `false` when `RequireHumanReview` points to `false`.
- `RequiresHumanReview()` returns `true` when `RequireHumanReview` points to `true`.

---

**FR-003:** `DefaultConfig()` MUST NOT set `RequireHumanReview`. The zero value (`nil`)
MUST be the default, yielding `RequiresHumanReview() == false`.

**Acceptance criteria:**
- `DefaultConfig().Merge.RequireHumanReview` is nil.
- `DefaultConfig().Merge.RequiresHumanReview()` returns `false`.

---

**FR-004:** `AdvanceConfig` in `internal/service/advance.go` MUST include a new field
`RequiresHumanReview func() bool`. A nil function MUST be treated as returning `false`.

**Acceptance criteria:**
- A caller that leaves `RequiresHumanReview` nil on `AdvanceConfig` gets agentic
  auto-advance behaviour (human review not required).
- A caller that sets `RequiresHumanReview` to a function returning `true` gets
  mandatory-halt behaviour.

---

**FR-005:** When `AdvanceFeatureStatus` reaches the `reviewing` stop-state:
- If `RequiresHumanReview()` is `true`, the advance MUST halt and return an `AdvanceResult`
  with `StoppedReason` indicating `"require_human_review is true"`.
- If `RequiresHumanReview()` is `false` (or the field is nil), the advance MUST invoke
  `checkAllTasksHaveVerification`. If that check passes, the advance MUST continue past
  `reviewing`. If that check fails, the advance MUST halt at `reviewing` with a
  `StoppedReason` identifying the failing condition.

**Acceptance criteria:**
- With `require_human_review: true` in config, `advance` halts at `reviewing` and returns
  a result whose `StoppedReason` contains `"require_human_review"`.
- With `require_human_review` absent and all tasks verified, `advance` does not stop at
  `reviewing` and continues to `done` (or the next applicable stop-state).
- With `require_human_review` absent and at least one task missing verification, `advance`
  halts at `reviewing` with a `StoppedReason` identifying the unverified task(s).
- An explicit `entity(action: transition, status: reviewing)` transition continues to work
  independently of this flag.

---

**FR-006:** A new function `checkAllTasksHaveVerification(feature *model.Feature,
entitySvc *EntityService) error` MUST be added to `internal/service/prereq.go`. It MUST:
- Query all tasks whose `parent_feature` matches the feature ID.
- Return `nil` if the task list is empty (vacuously true).
- Return `nil` if every task has a non-empty `Verification` field.
- Return a descriptive error naming the first task that has an empty `Verification` field.

**Acceptance criteria:**
- A feature with zero tasks returns `nil`.
- A feature where all tasks are `done` with non-empty verification returns `nil`.
- A feature where one or more tasks have an empty verification field returns an error
  whose message identifies at least one task ID with missing verification.
- A task in `needs-review` status with an empty verification field causes the check to
  return an error (it does not auto-pass for `needs-review` tasks).

---

**FR-007:** The MCP layer in `internal/mcp/entity_tool.go` MUST inject
`RequiresHumanReview` into the `AdvanceConfig` it constructs, sourced from
`cfg.Merge.RequiresHumanReview`. The injected function MUST be equivalent to
`cfg.Merge.RequiresHumanReview()`.

**Acceptance criteria:**
- With `require_human_review: true` in `.kbz/config.yaml`, calling
  `entity(action: "transition", id: "FEAT-xxx", status: "done", advance: true)`
  halts at `reviewing`.
- With `require_human_review` absent from `.kbz/config.yaml` and all tasks verified,
  the same call advances past `reviewing` to `done`.

---

**FR-008:** The existing `advanceStopStates` halt for `reviewing` MUST remain in place
and MUST be the code path that enforces the conditional logic. The stop-state map itself
is not removed; the branch inside the halt-state block becomes conditional.

**Acceptance criteria:**
- The `advanceStopStates` map still contains `reviewing` as a key.
- An explicit `entity(action: transition, status: done)` issued after a feature is already
  in `reviewing` continues to work and is not affected by the `require_human_review` flag.

---

## Non-Functional Requirements

**NFR-001:** The config field MUST follow the exact structural pattern of `RequireGitHubPR`:
`*bool` type, `omitempty` YAML tag, nil-safe accessor method.

**NFR-002:** The change to `AdvanceFeatureStatus` MUST NOT affect any advance path that
does not pass through the `reviewing` stop-state.

**NFR-003:** Existing tests for mandatory-halt behaviour
(`TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing` or equivalent) MUST be updated
to reflect the new conditional logic and MUST continue to pass when
`RequiresHumanReview` returns `true`.

**NFR-004:** New tests MUST cover:
- Auto-advance past `reviewing` when all tasks have verification and `require_human_review`
  is false.
- Halt at `reviewing` when one or more tasks lack verification and `require_human_review`
  is false.
- Halt at `reviewing` when `require_human_review` is true, regardless of task verification.
- `checkAllTasksHaveVerification` with zero tasks, all-verified tasks, and partially-
  verified tasks.

---

## Acceptance Criteria

| ID | Criterion |
|----|-----------|
| AC-01 | `require_human_review` absent from config → `RequiresHumanReview()` returns `false` |
| AC-02 | `require_human_review: true` → advance halts at `reviewing` with descriptive reason |
| AC-03 | `require_human_review: false` (explicit) → behaves identically to absent |
| AC-04 | All tasks verified + flag absent → advance continues past `reviewing` to `done` |
| AC-05 | Any task missing verification + flag absent → advance halts at `reviewing` |
| AC-06 | Zero tasks for feature + flag absent → auto-advances past `reviewing` |
| AC-07 | `needs-review` task with empty verification → blocks auto-advance |
| AC-08 | Explicit `status: reviewing` and `status: done` transitions unaffected |
| AC-09 | `DefaultConfig()` leaves `RequireHumanReview` as nil |
| AC-10 | `AdvanceConfig.RequiresHumanReview` nil → treated as `false` |

---

## Dependencies and Assumptions

- `internal/config/config.go` `MergeConfig` is the canonical config struct; the `RequireGitHubPR` field is the structural template for `RequireHumanReview`.
- `internal/service/advance.go` `AdvanceConfig` is the existing parameter struct for `AdvanceFeatureStatus`; it already holds injected function fields.
- `internal/service/prereq.go` contains `checkAllTasksTerminal`; `checkAllTasksHaveVerification` follows the same file and signature pattern.
- `internal/mcp/entity_tool.go` already constructs and injects `AdvanceConfig` fields from project config; the new field follows the same injection pattern.
- The `require_github_pr` feature (P24) has already been merged; its implementation patterns are live and available as reference.
- Task `Verification` field is the string written by `CompleteTask` when `VerificationPerformed` is provided to `finish()`; its field name and storage location are stable.
- The `reviewing→done` transition gate (`checkReviewReportExists`) is on the explicit transition path and is not affected by this change.