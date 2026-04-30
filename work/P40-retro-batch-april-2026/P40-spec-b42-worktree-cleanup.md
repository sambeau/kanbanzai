# Specification: Worktree and Cleanup Automation

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | Draft                         |
| Author | Spec Author                   |
| Batch  | B42-worktree-cleanup-automation |
| Design | P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements |

---

## Problem Statement

This specification implements Workstream D of the design described in
`work/P40-retro-batch-april-2026/P40-design-retro-batch-improvements.md`
(P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements, approved).

The April 2026 branch audit (retro-report-12) surfaced 60+ health errors from
orphaned worktree records, squash-merged branches showing critical drift alerts,
and stale remote tracking branches surviving weeks after merge. The worktree
lifecycle has no automated cleanup: records accumulate, directories are manually
deleted leaving orphaned state files, and the `worktree(action: remove)` tool
fails for records whose git directories no longer exist. Additionally,
`worktree(action: create)` accepts display-format entity IDs, creating ghost
records pointing to non-existent entities.

This specification addresses four fixes from the design's Workstream D:

1. **D1 — Add `worktree(action: gc)` for orphaned records:** Detect and remove
   worktree state records whose git worktree directories no longer exist, with
   a `dry_run` preview mode.
2. **D2 — Skip drift alerts for merged branches:** When a worktree record has a
   `merged_at` timestamp, suppress branch drift alerts — the branch is
   intentionally stale.
3. **D3 — Auto-schedule cleanup on merge:** After a successful merge, automatically
   clean up the worktree and delete the remote tracking branch.
4. **D4 — Validate entity IDs at worktree creation:** Reject display-format IDs
   (with embedded segment hyphen) at `worktree(action: create)` time.

**Scope inclusion:** The `worktree` tool (gc action, create validation), the
`branch` tool (drift alert suppression), and the `merge` tool (post-merge
cleanup scheduling).

**Scope exclusion:** The `cleanup` tool itself (it already handles merged/abandoned
worktrees correctly — the gap is that it's not called automatically). New
worktree lifecycle states. Changes to git worktree internals (removal still
delegates to `git worktree remove`). The `AutoDeleteRemoteBranch` config
default change (configuration, not code).

---

## Requirements

### Functional Requirements

- **REQ-001:** `worktree(action: gc)` MUST detect worktree records in
  `.kbz/state/worktrees/` whose git worktree directory (the filesystem path
  recorded in the worktree record) does not exist. These are orphaned records.

- **REQ-002:** `worktree(action: gc, dry_run: true)` MUST list all orphaned
  records without removing them. The response MUST include the worktree ID,
  entity ID, and recorded path for each orphaned record.

- **REQ-003:** `worktree(action: gc)` (with `dry_run` omitted or `false`) MUST
  remove the state files for all detected orphaned records. It MUST NOT attempt
  to run `git worktree remove` on directories that do not exist.

- **REQ-004:** When the `branch` tool evaluates a branch associated with a
  worktree record that has a non-nil `merged_at` timestamp, the tool MUST
  suppress drift alerts for that branch. The branch is intentionally stale.

- **REQ-005:** When the `branch` tool evaluates a branch associated with a
  worktree record that has no `merged_at` timestamp (or a nil value), drift
  alerts MUST be emitted as currently — no change in behaviour for unmerged
  branches.

- **REQ-006:** After `merge(action: execute)` successfully merges a feature
  branch, the tool MUST automatically schedule cleanup for the merged worktree
  by calling the equivalent of `cleanup(action: execute, worktree_id: ...)`.

- **REQ-007:** After a successful squash merge, `merge(action: execute)` MUST
  delete the local tracking branch to prevent false drift alerts from the
  now-stale branch reference.

- **REQ-008:** `worktree(action: create)` MUST validate the `entity_id` parameter
  against the canonical ID format. If the ID contains an embedded hyphen after
  the type prefix (matching the display-ID pattern `FEAT-XXXX-XXXXXXXX`), the
  tool MUST reject it with an error message suggesting the canonical form.

- **REQ-009:** `worktree(action: create)` MUST accept entity IDs in canonical
  ULID format (e.g., `FEAT-01KQG3J1RG3J3`) without change — only display-format
  IDs with embedded segment hyphens should be rejected.

### Non-Functional Requirements

- **REQ-NF-001:** `worktree(action: gc)` MUST NOT require git to be invoked for
  orphan detection. Detection is purely filesystem-based: does the recorded
  directory path exist?

- **REQ-NF-002:** Post-merge cleanup scheduling MUST be best-effort — if cleanup
  fails, the merge MUST still be reported as successful and the failure logged.

- **REQ-NF-003:** Entity ID validation at worktree creation MUST complete in
  O(1) time — a simple string check against the display-ID pattern, not a
  database lookup.

---

## Constraints

- The `worktree` tool's existing actions (create, get, list, remove, update) are
  preserved. `gc` is a new action.
- The `branch` tool's existing output format is preserved. Drift suppression is
  a filter on the alert emission, not a format change.
- The `merge` tool's existing check/execute lifecycle is preserved. Cleanup
  scheduling is a post-merge side effect.
- This specification does NOT change the default value of `AutoDeleteRemoteBranch`
  (that is a configuration change). It ensures the post-merge hook calls the
  deletion when the config flag is true.
- This specification does NOT change how `worktree(action: remove)` handles
  records with intact directories — that path already works correctly.

---

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a worktree state file at
  `.kbz/state/worktrees/WT-EXAMPLE.yaml` whose recorded directory path does
  not exist on disk, when `worktree(action: gc, dry_run: true)` is called,
  then `WT-EXAMPLE` appears in the orphaned records list with its entity ID
  and recorded path.

- **AC-002 (REQ-003):** Given two orphaned worktree records, when
  `worktree(action: gc)` is called, then both state files are removed from
  `.kbz/state/worktrees/` and the response reports two records removed.

- **AC-003 (REQ-001):** Given a worktree state file whose recorded directory
  path does exist on disk, when `worktree(action: gc)` is called, then that
  record is NOT removed — it is not orphaned.

- **AC-004 (REQ-004):** Given a worktree record with `merged_at:
  "2026-04-15T10:00:00Z"` and a branch 321 commits behind main, when the
  branch health check runs, then no drift alert is emitted for that branch.

- **AC-005 (REQ-005):** Given a worktree record with no `merged_at` timestamp
  and a branch 50 commits behind main, when the branch health check runs,
  then a drift alert IS emitted — unmerged branches still alert.

- **AC-006 (REQ-006):** Given a feature with a worktree ready to merge, when
  `merge(action: execute)` succeeds, then the merged worktree appears in
  `cleanup(action: list)` output as scheduled for cleanup (or is already
  cleaned up).

- **AC-007 (REQ-007):** Given a feature branch merged via squash, when
  `merge(action: execute)` completes, then the local tracking branch for
  that feature no longer exists (`git branch` does not list it).

- **AC-008 (REQ-008):** Given a call to `worktree(action: create, entity_id:
  "FEAT-01KQ7-JDT511BZ")` (display-ID format with embedded hyphen), when the
  tool executes, then it returns an error containing "display ID" and
  suggesting the canonical form `FEAT-01KQ7JDT511BZ`.

- **AC-009 (REQ-009):** Given a call to `worktree(action: create, entity_id:
  "FEAT-01KQG3J1RG3J3")` (canonical ULID format), when the tool executes,
  then the worktree is created normally — no false positive rejection.

- **AC-010 (REQ-NF-001):** Given `worktree(action: gc)`, when the operation
  executes, then no `git worktree` commands are invoked during orphan
  detection — detection is a filesystem existence check only.

- **AC-011 (REQ-NF-002):** Given a successful merge where the post-merge
  cleanup scheduling fails (e.g., cleanup tool unavailable), when
  `merge(action: execute)` returns, then the merge is reported as
  successful with a warning about cleanup failure — not as a merge failure.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: create orphaned state file (no directory), call gc dry_run, assert record listed |
| AC-002 | Test | Automated test: create two orphaned state files, call gc, assert both removed and count correct |
| AC-003 | Test | Automated test: create worktree record with existing directory, call gc, assert not removed |
| AC-004 | Test | Automated test: create worktree record with merged_at and stale branch, run branch check, assert no drift alert |
| AC-005 | Test | Automated test: create worktree record without merged_at and stale branch, run branch check, assert drift alert emitted |
| AC-006 | Test | Integration test: merge a feature, verify worktree appears in cleanup list or is already cleaned |
| AC-007 | Test | Automated test: squash-merge a branch, verify local tracking branch deleted |
| AC-008 | Test | Automated test: call worktree create with display-ID format, assert error with suggestion |
| AC-009 | Test | Automated test: call worktree create with canonical ULID, assert success |
| AC-010 | Inspection | Code review: confirm gc detection does not invoke git; pure os.Stat/exists check |
| AC-011 | Test | Automated test: simulate cleanup failure during post-merge, assert merge reports success with warning |
