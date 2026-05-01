# Dev-Plan: Fix entity list parent_feature Filter (B40-F1)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG2Q5MDC6E           |
| Spec   | B40-fix-tool-correctness/spec-p40-spec-b40-tool-correctness |

---

## Overview

Single bug fix: the `parent_feature` filter on `entity(action: list, type: task)`
currently ignores the filter and returns all tasks. Fix the filter to scope
results to the specified feature. One task.

---

## Task Breakdown

### T1 — Fix parent_feature filter and add tests

**Deliverable:** Corrected filter logic in the entity list handler, plus tests.

**Scope:**
- Locate the `parent_feature` filter application in the entity service list path.
- Fix the filter to return only tasks whose `parent_feature` matches the provided ID.
- Verify unfiltered listing (no `parent_feature` param) still returns all tasks.
- Write tests: filter returns correct subset; filter with no matching tasks returns
  empty; unfiltered listing unchanged.

**Dependencies:** None.

**Verification:** `go test ./...` in affected packages; AC-001, AC-002, AC-012.

**Estimated effort:** 1 (single code change + 3 test cases)

---

## Dependency Graph

```
T1 (fix filter + tests)
```

Single task, no dependencies.

---

## Interface Contracts

No cross-task contracts — single task.

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| REQ-001 | T1 |
| REQ-NF-001 | T1 |
| AC-001 | T1 |
| AC-002 | T1 |
| AC-012 | T1 |
