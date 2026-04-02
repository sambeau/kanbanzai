# Specification: Task Crash Recovery

**Status:** Draft
**Feature:** FEAT-01KN07T65HDZM (lifecycle-override-and-recovery)
**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` — Feature 5
**Date:** 2026-04-02

---

## Problem Statement

When an agent claims a task (transitioning it to `active`) and then crashes, is interrupted,
or is otherwise unable to continue, the task becomes stuck. The current task lifecycle in
`internal/validate/lifecycle.go` does not include `ready` as a valid target from `active`.
The only escapes are `needs-rework` (which implies the work product is defective, not that
the agent died) or `not-planned` (which abandons the task entirely). There is no "unclaim"
mechanism.

Additionally, there is no proactive detection of stuck tasks. The `checkGitActivitySince`
function in `internal/health/phase4a.go` is a stub that always returns `false`, so the
existing `CheckStalledDispatches` health check cannot distinguish genuinely stalled tasks
from tasks with recent git activity. Operators must manually inspect task state to find
stuck work.

## Requirements

### Functional Requirements

**FR-01. Active-to-ready transition.** The system shall allow transitioning a task from
`active` to `ready`. This transition represents an unclaim — the task returns to the work
queue and becomes available for any agent to pick up.

**FR-02. No gate checks on unclaim.** The `active → ready` transition shall not require
any gate checks, reason parameters, or special permissions. Any caller that can transition
a task (agent or human) shall be allowed to unclaim any active task via the standard
`entity(action: "transition", id: "TASK-xxx", status: "ready")` call.

**FR-03. No side-effects on unclaim.** The `active → ready` transition shall not clear or
modify task fields such as `dispatched_to`, `dispatched_at`, summary, or acceptance
criteria. The task returns to the ready pool with its existing metadata intact. The
`dispatched_to` and `dispatched_at` fields may retain their previous values — they are
overwritten on the next claim via `next(id)`.

**FR-04. Stuck-task attention items.** The `status` tool shall surface an attention item
when a task has been in `active` status for more than 24 hours with no recent git commits
on its parent feature's worktree branch. The attention item message shall be:
`"TASK-xxx has been active for >24h with no recent commits — may need unclaim"`.

**FR-05. Git activity detection.** The `checkGitActivitySince` function in
`internal/health/phase4a.go` shall be implemented to check whether a given branch has any
git commits after a given timestamp. It shall:
- Run `git log` (or equivalent) against the specified branch in the specified repository
  path, filtering for commits after the `since` timestamp.
- Return `true` if at least one commit exists after `since`, `false` otherwise.
- Return `false` on any error (best-effort; do not propagate errors).

**FR-06. Stuck-task detection with git activity.** When `checkGitActivitySince` returns
`true` for an active task's parent feature worktree branch, the `CheckStalledDispatches`
function shall not emit an attention item for that task, even if it has been active for
more than 24 hours.

## Constraints

**C-01.** The lifecycle change is confined to the `allowedTransitions` map in
`internal/validate/lifecycle.go`. No new transition types, hooks, or entity fields are
introduced.

**C-02.** The `checkGitActivitySince` implementation must be best-effort. It shall not
fail the health check or return an error if git is unavailable, the branch does not exist,
or the repository path is invalid. In all error cases it returns `false`.

**C-03.** The stalled-dispatch threshold (currently 24 hours) is controlled by the existing
`stallThresholdDays` parameter in `CheckStalledDispatches`. This specification does not
change the threshold or make it user-configurable beyond what already exists.

**C-04.** No changes to the MCP tool API surface. The `entity(action: "transition")`
tool already accepts any valid status string. Adding `ready` to the allowed targets from
`active` requires no schema changes.

## Acceptance Criteria

**AC-01. Lifecycle map includes active → ready.**
- **Given** a task in `active` status
- **When** a caller transitions it to `ready` via `entity(action: "transition", id: "TASK-xxx", status: "ready")`
- **Then** the transition succeeds and the task status becomes `ready`

**AC-02. No gate checks block the transition.**
- **Given** a task in `active` status with no special metadata or permissions
- **When** a caller transitions it to `ready`
- **Then** the transition succeeds without requiring a reason, override, or any additional parameters

**AC-03. Unclaimed task appears in work queue.**
- **Given** a task that was transitioned from `active` to `ready`
- **When** `next()` is called to inspect the work queue
- **Then** the task appears in the ready queue and can be claimed by any agent

**AC-04. Stuck task surfaces as attention item (no git activity).**
- **Given** a task in `active` status for more than 24 hours
- **And** the parent feature's worktree branch has no git commits since the task was dispatched
- **When** the `status` tool generates the project dashboard
- **Then** an attention item is emitted with the message: `"TASK-xxx has been active for >24h with no recent commits — may need unclaim"`

**AC-05. Active task with recent git activity is not flagged.**
- **Given** a task in `active` status for more than 24 hours
- **And** the parent feature's worktree branch has at least one git commit after the task's `dispatched_at` timestamp
- **When** the `status` tool generates the project dashboard
- **Then** no stalled-task attention item is emitted for that task

**AC-06. checkGitActivitySince detects commits after timestamp.**
- **Given** a repository path and branch with commits at times T1 and T2 (where T1 < T2)
- **When** `checkGitActivitySince(repoPath, branch, T1)` is called
- **Then** it returns `true`

**AC-07. checkGitActivitySince returns false when no commits exist after timestamp.**
- **Given** a repository path and branch whose latest commit is at time T1
- **When** `checkGitActivitySince(repoPath, branch, T2)` is called with T2 > T1
- **Then** it returns `false`

**AC-08. checkGitActivitySince is resilient to errors.**
- **Given** an invalid repository path, a non-existent branch, or git not being available
- **When** `checkGitActivitySince` is called
- **Then** it returns `false` without panicking or propagating an error

**AC-09. Invalid transitions from active are still rejected.**
- **Given** a task in `active` status
- **When** a caller transitions it to `queued`
- **Then** the transition is rejected with a validation error (confirming the lifecycle map was not over-broadened)

## Verification Plan

1. **Unit test — lifecycle map** (`internal/validate/`): Verify that `active → ready` is
   accepted by `ValidateTransition` for task entities, and that `active → queued` is still
   rejected. Covers AC-01, AC-02, AC-09.

2. **Unit test — `checkGitActivitySince`** (`internal/health/`): Create a temporary git
   repository with known commits at known times. Assert `true` when querying a timestamp
   before the latest commit, `false` when querying after. Assert `false` for a non-existent
   branch and for an invalid repo path. Covers AC-06, AC-07, AC-08.

3. **Unit test — `CheckStalledDispatches`** (`internal/health/`): Provide task maps with
   varying `dispatched_at` ages and mock worktree branches. With `checkGitActivitySince`
   now functional, verify that tasks with recent git activity are not flagged, while tasks
   without activity are. Covers AC-04, AC-05.

4. **Integration test — unclaim round-trip** (`internal/tool/`): Create a task, advance it
   to `active`, transition it to `ready`, then verify it appears in the `next()` work queue
   output. Covers AC-01, AC-03.