# Dev-Plan: Auto-Infer Document Owner from Path Context (B41-F1)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3AWTYSEQ           |
| Spec   | B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle |

---

## Overview

When `doc(action: register)` is called without an explicit `owner`, extract the
plan/batch slug from the file path and resolve it as the default owner. Warn if
the path is already registered under a different owner. Fall back to `PROJECT/`
when no entity matches the path.

---

## Task Breakdown

### T1 — Implement owner inference and path conflict warning

**Deliverable:** Updated `doc register` handler with owner inference.

**Scope:**
- Extract plan/batch slug from path components (e.g., `work/{slug}/...`).
- Resolve slug to entity via existing entity lookup (single cache/store read).
- If entity found: use as owner.
- If path already registered under different owner: emit warning.
- If no entity matches: fall back to `PROJECT/` (current behaviour).
- Explicit `owner` parameter always takes precedence.

**Dependencies:** None.

**Verification:** Code review of inference logic. Tests below.

**Estimated effort:** 1

### T2 — Write tests for owner inference

**Deliverable:** Tests covering all inference scenarios.

**Scope:**
- Register under batch folder → owner is batch ID (AC-001).
- Register with explicit owner → explicit used, not inferred (AC-002).
- Register path already under different owner → warning emitted (AC-003).
- Register under unresolvable path → fallback to PROJECT/ (AC-004).
- Confirm single cache read (AC-011).

**Dependencies:** T1.

**Verification:** `go test ./...` in doc package.

**Estimated effort:** 1

---

## Dependency Graph

```
T1 (inference logic) ──→ T2 (tests)
```

---

## Interface Contracts

No cross-task contracts — T2 tests the observable behaviour of T1's inference logic.

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
| AC-004 | T2 |
| AC-011 | T2 |
