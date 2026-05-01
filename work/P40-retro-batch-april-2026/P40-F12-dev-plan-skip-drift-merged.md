# Dev-Plan: Skip Drift Alerts for Merged Branches (B42-F2)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3TZZD0ZG           |
| Spec   | B42-worktree-cleanup-automation/spec-p40-spec-b42-worktree-cleanup |

---

## Overview

When the branch health check evaluates a branch whose worktree record has a
`merged_at` timestamp, suppress drift alerts. Merged branches are intentionally
stale. Unmerged branches continue to alert as before.

---

## Task Breakdown

### T1 — Add merged_at check to branch drift alert and tests

**Deliverable:** Updated branch health check with merged_at suppression, plus tests.

**Scope:**
- In the branch drift evaluation: check `worktree.merged_at`.
- If non-nil: skip drift alert for that branch.
- If nil: emit drift alert as currently (unchanged behaviour).
- Write tests: merged branch with drift → no alert (AC-004); unmerged branch
  with drift → alert emitted (AC-005).

**Dependencies:** None.

**Verification:** `go test ./...` in branch/health package.

**Estimated effort:** 1

---

## Dependency Graph

```
T1 (merged_at check + tests)
```

Single task.

---

## Interface Contracts

No cross-task contracts — single task.

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| REQ-004 | T1 |
| REQ-005 | T1 |
| AC-004 | T1 |
| AC-005 | T1 |
