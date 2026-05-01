# Dev-Plan: Add worktree gc for Orphaned Records (B42-F1)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3TZR2JDP           |
| Spec   | B42-worktree-cleanup-automation/spec-p40-spec-b42-worktree-cleanup |

---

## Overview

Add `worktree(action: gc)` — a new action that detects worktree state records
whose git worktree directories no longer exist (orphaned records), and removes
them. Supports `dry_run` for preview. Detection is filesystem-only, no git
invocation.

---

## Task Breakdown

### T1 — Implement gc action with dry_run

**Deliverable:** New `gc` action handler in the worktree tool.

**Scope:**
- Iterate `.kbz/state/worktrees/` records.
- For each: check if recorded directory path exists on disk.
- `dry_run: true` — list orphaned records with ID, entity ID, path.
- `dry_run: false` — remove orphaned state files.
- Do not invoke `git worktree remove` (directories don't exist).
- Detection is `os.Stat` only (REQ-NF-001).

**Dependencies:** None.

**Verification:** Code review. Tests below.

**Estimated effort:** 1

### T2 — Write tests for gc action

**Deliverable:** Tests covering all gc scenarios.

**Scope:**
- dry_run lists orphaned record (AC-001).
- gc removes orphaned state files, reports count (AC-002).
- gc does not remove records with existing directories (AC-003).
- Confirm no git invocation during detection (AC-010).

**Dependencies:** T1.

**Verification:** `go test ./...` in worktree package.

**Estimated effort:** 1

---

## Dependency Graph

```
T1 (gc action) ──→ T2 (tests)
```

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T2 | After T1, `worktree(action: gc)` accepts optional `dry_run` param. T2 tests all three detection scenarios and verifies no git calls. |

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| REQ-001 | T1 |
| REQ-002 | T1 |
| REQ-003 | T1 |
| REQ-NF-001 | T1, T2 |
| AC-001 | T2 |
| AC-002 | T2 |
| AC-003 | T2 |
| AC-010 | T2 |
