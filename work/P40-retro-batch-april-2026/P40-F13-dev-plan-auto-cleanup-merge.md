# Dev-Plan: Auto-Schedule Cleanup on Merge (B42-F3)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3V03PAM4           |
| Spec   | B42-worktree-cleanup-automation/spec-p40-spec-b42-worktree-cleanup |

---

## Overview

After a successful merge, automatically schedule cleanup for the merged worktree
and delete the local tracking branch on squash merge. Cleanup failure is
best-effort — it must not cause the merge to be reported as failed.

---

## Task Breakdown

### T1 — Implement post-merge cleanup hook and tests

**Deliverable:** Post-merge side effect in merge execute, plus tests.

**Scope:**
- After merge success: call cleanup scheduling for the merged worktree.
- After squash merge: delete local tracking branch.
- If cleanup fails: log warning, report merge as success (REQ-NF-002).
- Write tests: merged worktree appears in cleanup list (AC-006); squash merge
  deletes local branch (AC-007); cleanup failure doesn't fail merge (AC-011).

**Dependencies:** None.

**Verification:** `go test ./...` in merge package.

**Estimated effort:** 1

---

## Dependency Graph

```
T1 (post-merge hook + tests)
```

Single task.

---

## Interface Contracts

No cross-task contracts — single task.

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| REQ-006 | T1 |
| REQ-007 | T1 |
| REQ-NF-002 | T1 |
| AC-006 | T1 |
| AC-007 | T1 |
| AC-011 | T1 |
