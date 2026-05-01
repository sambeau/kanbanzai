# Dev-Plan: Make Multi-Edit Calls Atomic (B40-F3)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG2Q5RS9G9           |
| Spec   | B40-fix-tool-correctness/spec-p40-spec-b40-tool-correctness |

---

## Overview

Currently, multi-edit `edit_file` calls apply matching edits and silently skip
non-matching ones — including deletions without replacements. Fix: pre-flight all
`old_text` patterns against the target file before applying any edits. On any
mismatch, apply zero edits and return an error naming the failed patterns. On all
matches, apply all edits sequentially.

---

## Task Breakdown

### T1 — Implement pre-flight match for multi-edit calls

**Deliverable:** Pre-flight logic in the `edit_file` handler.

**Scope:**
- Detect multi-edit calls (more than one entry in the `edits` array).
- Read the target file once (REQ-NF-003).
- Attempt to match all `old_text` patterns against the file content.
- If all match: proceed with sequential application (existing logic).
- If any fail: return error listing failed patterns, apply zero edits.

**Dependencies:** None.

**Verification:** Code review — confirm single read, all-or-nothing semantics.

**Estimated effort:** 1 (pre-flight logic)

### T2 — Write tests for multi-edit atomicity

**Deliverable:** Tests covering all multi-edit scenarios.

**Scope:**
- Test: one match + one mismatch → file unchanged, error names failed pattern (AC-004).
- Test: two matches → both edits applied (AC-005).
- Test: single-edit call unchanged (AC-011).
- Verify existing single-edit tests still pass.

**Dependencies:** T1 (needs the implementation to test against).

**Verification:** `go test ./...` passes all new and existing edit_file tests.

**Estimated effort:** 1 (three test cases)

---

## Dependency Graph

```
T1 (pre-flight logic) ──→ T2 (tests)
```

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T2 | After T1, multi-edit calls are atomic: all patterns match → all edits applied; any mismatch → zero edits + named error. Single-edit behaviour unchanged. T2 tests both branches. |

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-004 | T1 |
| REQ-005 | T1, T2 |
| REQ-006 | T1, T2 |
| REQ-009 | T1, T2 |
| REQ-NF-003 | T1 |
| AC-004 | T2 |
| AC-005 | T2 |
| AC-011 | T2 |
