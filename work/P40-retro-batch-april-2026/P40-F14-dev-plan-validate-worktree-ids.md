# Dev-Plan: Validate Entity IDs at Worktree Creation (B42-F4)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3V06KNZG           |
| Spec   | B42-worktree-cleanup-automation/spec-p40-spec-b42-worktree-cleanup |

---

## Overview

Add entity ID validation to `worktree(action: create)`. Reject IDs matching the
display-ID format (embedded hyphen after type prefix, e.g. `FEAT-01KQ7-JDT511BZ`).
Accept canonical ULID format. O(1) string check only.

---

## Task Breakdown

### T1 — Add entity ID validation and tests

**Deliverable:** Validation logic in worktree create handler, plus tests.

**Scope:**
- Check entity_id against display-ID pattern (embedded hyphen after prefix).
- Reject with error suggesting canonical form (REQ-008).
- Accept canonical ULID format (REQ-009).
- Validation is O(1) string check, no DB lookup (REQ-NF-003).
- Write tests: display-ID rejected with suggestion (AC-008); canonical ID
  accepted (AC-009).

**Dependencies:** None.

**Verification:** `go test ./...` in worktree package.

**Estimated effort:** 0.5

---

## Dependency Graph

```
T1 (validation + tests)
```

Single task.

---

## Interface Contracts

No cross-task contracts — single task.

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| REQ-008 | T1 |
| REQ-009 | T1 |
| REQ-NF-003 | T1 |
| AC-008 | T1 |
| AC-009 | T1 |
