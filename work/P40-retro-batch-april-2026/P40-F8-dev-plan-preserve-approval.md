# Dev-Plan: Preserve Approval Status on Minor Doc Edits (B41-F2)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3AWX916X           |
| Spec   | B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle |

---

## Overview

`doc(action: refresh)` currently resets approval status to `draft` on every
content hash change. Detect the scope of the change: preserve approval for
formatting/whitespace-only changes, warn before resetting for substantive
changes.

---

## Task Breakdown

### T1 — Implement change-scope detection and tests

**Deliverable:** Updated `doc refresh` handler with scope detection, plus tests.

**Scope:**
- On content hash change, compare old and new content.
- If only whitespace/formatting changed: preserve approval status.
- If substantive change detected: warn caller, reset to draft on confirmation.
- Write tests: whitespace-only change preserves approval (AC-005); content
  change resets to draft with warning (AC-006).

**Dependencies:** None.

**Verification:** `go test ./...` in doc package.

**Estimated effort:** 1 (scope detection + 2 tests)

---

## Dependency Graph

```
T1 (scope detection + tests)
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
| AC-005 | T1 |
| AC-006 | T1 |
