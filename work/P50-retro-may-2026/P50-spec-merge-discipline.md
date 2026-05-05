# Specification: Merge Discipline and Definition of Done

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | approved |
| Author | spec-author                    |

## Overview

This specification implements the merge discipline and definition-of-done changes described in
`work/P50-retro-may-2026/P50-design-merge-discipline.md`
(DOC-`FEAT-01KQTWFY52EG1/design-p50-design-merge-discipline`), Feature 5.

Thirty unmerged branches were discovered during the P50 merge operation — features in `done`
or `reviewing` status whose code never reached main. The root cause is that the feature
lifecycle has no merge step and no post-merge verification. A feature can reach `done` with
its branch still open and its code untested on main.

## Scope

**In scope:**
- Two new lifecycle stages: `merging` and `verifying`
- Stage bindings for both new stages
- Merge prompt added to `orchestrate-review` and `kanbanzai-agents` skills
- Worktree `merged` status fix: verify branch is on main before marking merged
- Skill file changes follow dual-write rule

**Out of scope:**
- Fixing pre-existing test failures (28 as of 2026-05-05 — tracked separately)
- CI/CD integration — verification runs locally
- Auto-merge — agents still call merge explicitly
- Changes to the `merge` tool's core behaviour

## Functional Requirements

- **REQ-001:** The feature lifecycle must include a `merging` stage after `reviewing`.
  Transition from `reviewing` to `merging` requires: all tasks terminal, at least one
  approved report document.
- **REQ-002:** The feature lifecycle must include a `verifying` stage after `merging`.
  Transition from `merging` to `verifying` requires: merge to main executed successfully
  (branch is ancestor of main).
- **REQ-003:** The `verifying` stage must run `go build ./...` from the repository root.
  If the build fails, the feature must transition to `needs-rework` with the build
  output attached as the reason.
- **REQ-004:** The `verifying` stage must run `go test ./...` from the repository root.
  If any test fails, the feature must transition to `needs-rework` with the failing
  test output attached as the reason.
- **REQ-005:** If both build and tests pass, the feature must advance from `verifying`
  to `done`.
- **REQ-006:** After `merge(action: "execute")` succeeds, the worktree record must
  verify that the feature branch is an ancestor of main (`git merge-base --is-ancestor`).
  Only if verified must the worktree record be marked as `merged`. Otherwise the
  worktree remains `active` and the feature remains in `merging`.
- **REQ-007:** The `orchestrate-review` skill must include a merge prompt instructing
  the orchestrator to: verify PR exists, check merge gates, execute merge, run
  build/tests, and transition to `done` on success.
- **REQ-008:** The `kanbanzai-agents` skill must include the merge prompt so agents
  working without an orchestrator also know to merge after review.
- **REQ-009:** All skill file changes must be mirrored to `internal/kbzinit/skills/`
  per the dual-write rule.
- **REQ-010:** The `merging` stage gate must be `auto` — no human gate, but the
  merge prompt ensures the agent takes the necessary actions.

## Non-Functional Requirements

- **REQ-NF-001:** The `merging` and `verifying` stages must be optional for features
  created before this change — existing `done` features must not be forced through
  the new stages retroactively.
- **REQ-NF-002:** Build and test verification must complete within 5 minutes on a
  typical development machine.

## Constraints (Scope Exclusions)

- The `merge` tool's existing behaviour must not change — only the lifecycle and
  prompts change.
- The `merging` stage does NOT add a new MCP tool action — it uses existing `pr`,
  `merge(action: "check")`, and `merge(action: "execute")`.
- Pre-existing test failures must not block the `verifying` stage — the stage
  detects regressions, not fixes existing problems. Known failures must be
  documented in a waiver file.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a feature in `reviewing` with all tasks terminal
  and an approved report, when `entity(action: "transition", status: "merging")`
  is called, then the feature advances to `merging`.
- **AC-002 (REQ-002):** Given a feature in `merging` whose branch has been merged
  to main, when `entity(action: "transition", status: "verifying")` is called,
  then the feature advances to `verifying`.
- **AC-003 (REQ-003):** Given a feature in `verifying`, when `go build ./...`
  fails, then the feature transitions to `needs-rework` with the build error
  in the transition reason.
- **AC-004 (REQ-004):** Given a feature in `verifying`, when `go test ./...`
  fails, then the feature transitions to `needs-rework` with the test failure
  output in the transition reason.
- **AC-005 (REQ-005):** Given a feature in `verifying`, when `go build ./...`
  and `go test ./...` both pass, then the feature advances to `done`.
- **AC-006 (REQ-006):** Given a successful `merge(action: "execute")` call,
  when the feature branch is an ancestor of main, then the worktree record
  is marked `merged`. When the branch is not an ancestor of main, then the
  worktree record remains `active`.
- **AC-007 (REQ-007):** Given the `orchestrate-review` skill file, when the
  post-review section is inspected, then it contains a merge prompt with the
  five steps from the design (verify PR, check gates, execute merge, run
  build/tests, transition to done).
- **AC-008 (REQ-008):** Given the `kanbanzai-agents` skill file, when the
  merge-related guidance is inspected, then it contains the same merge prompt.
- **AC-009 (REQ-009):** Given a change to `.agents/skills/kanbanzai-agents/SKILL.md`
  or `.kbz/skills/orchestrate-review/SKILL.md`, then the corresponding change
  exists in `internal/kbzinit/skills/`.
- **AC-010 (REQ-010):** Given a feature in `merging` with the merge prompt
  satisfied, when the agent completes all merge steps, then the feature
  advances to `verifying` without a human gate check.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: transition feature from reviewing to merging with satisfied prerequisites |
| AC-002 | Test | Unit test: transition feature from merging to verifying after branch is on main |
| AC-003 | Test | Integration test: simulate build failure, assert transition to needs-rework |
| AC-004 | Test | Integration test: simulate test failure, assert transition to needs-rework |
| AC-005 | Test | Integration test: build+tests pass, assert transition to done |
| AC-006 | Test | Unit test: merge.execute with branch on main → worktree merged; branch not on main → worktree active |
| AC-007 | Inspection | Review `orchestrate-review` SKILL.md for merge prompt content |
| AC-008 | Inspection | Review `kanbanzai-agents` SKILL.md for merge prompt content |
| AC-009 | Inspection | Diff review: verify dual-write in same commit |
| AC-010 | Test | Unit test: merging stage gate mode is auto (no human checkpoint required) |
