# Dev-Plan: Add bypassable Field to Merge Gate Results (B41-F3)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3AWZ3HEZ           |
| Spec   | B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle |

---

## Overview

Add a `bypassable: bool` field to each gate result in `merge(action: check)`
output. Hard gates like `review_report_exists` get `bypassable: false`. Soft
gates get `bypassable: true`. The field is informational only — no change to
`merge execute` behaviour.

---

## Task Breakdown

### T1 — Add bypassable field and tests

**Deliverable:** Updated merge check output format with bypassable field, plus tests.

**Scope:**
- Add `bypassable: bool` to the gate result struct.
- Mark hard gates (those that reject `override: true`) with `bypassable: false`.
- Mark soft gates with `bypassable: true`.
- Write test: call merge check, assert `review_report_exists` has bypassable: false,
  assert typical gates have bypassable: true (AC-007).
- Verify merge execute behaviour unchanged (REQ-NF-002).

**Dependencies:** None.

**Verification:** `go test ./...` in merge package.

**Estimated effort:** 1 (field addition + test)

---

## Dependency Graph

```
T1 (bypassable field + test)
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
| REQ-NF-002 | T1 |
| AC-007 | T1 |
