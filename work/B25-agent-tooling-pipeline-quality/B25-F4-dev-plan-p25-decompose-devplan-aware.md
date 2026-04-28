# Dev Plan: Dev-plan-aware task grouping in `decompose propose`

**Feature:** FEAT-01KPQ08YBJ5AK
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Spec:** work/spec/p25-decompose-devplan-aware.md
**Status:** Draft

---

## Overview

Implement dev-plan-aware task grouping in `decompose propose`. When an approved dev-plan
document is linked to the feature, `DecomposeFeature` reads its `## Task Breakdown` section
and uses the tasks defined there as the authoritative proposal. Falls back to the existing
AC-based heuristic when no approved dev plan is available or the Task Breakdown section is
absent/empty.

All changes are confined to `internal/service/decompose.go` and
`internal/service/decompose_test.go`.

---

## Task Breakdown

### Task 1: Add `parseDevPlanTasks` and dev plan discovery to `DecomposeFeature`

- **Description:** Implement the `parseDevPlanTasks(featureSlug string, content []byte) ([]ProposedTask, bool)` unexported helper and integrate dev plan discovery into `DecomposeFeature`. Discovery checks `feat.State["dev_plan"]` first (direct reference), then falls back to an owner-query for `type="dev-plan"`, `status="approved"`. When a valid parse result is returned, build the `Proposal` from the parsed tasks, set `GuidanceApplied = ["dev-plan-tasks", ...]` (excluding `"test-tasks-explicit"`), and skip the AC heuristic. Move the zero-criteria spec gate inside the AC fallback branch.
- **Deliverable:** Updated `internal/service/decompose.go` with `parseDevPlanTasks` and the modified `DecomposeFeature` flow.
- **Depends on:** None (independent)
- **Effort:** Large
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012

### Task 2: Write tests for dev-plan-aware decomposition

- **Description:** Add tests to `internal/service/decompose_test.go` covering:
  - Feature with approved dev plan + valid Task Breakdown → tasks sourced from dev plan (AC-001)
  - Feature with draft dev plan → AC heuristic used, no error (AC-002)
  - Feature with approved dev plan missing `## Task Breakdown` → AC heuristic + warning (AC-003)
  - Feature with approved dev plan + no parseable spec ACs → valid proposal, no zero-criteria error (AC-004)
  - Feature with no dev plan + no parseable ACs → zero-criteria error (AC-005)
  - `Estimate` mapping: `Medium` → `3.0` (AC-006)
  - `Covers` nil when `Spec requirements` absent (AC-007)
  - `GuidanceApplied` contains `"dev-plan-tasks"` and not `"test-tasks-explicit"` (AC-008)
  - `SliceDetails` populated on dev plan path (AC-009)
  - `decompose apply` succeeds with dev-plan-sourced proposal (AC-010)
- **Deliverable:** Updated `internal/service/decompose_test.go` with new test cases.
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirements:** AC-001 through AC-010

---

## Dependency Graph

```
Task 1 (parseDevPlanTasks + DecomposeFeature integration)
    └─► Task 2 (tests)
```

Tasks 1 and 2 are strictly sequential. Task 2 requires the implementation from Task 1 to compile.

---

## Interface Contracts

### `parseDevPlanTasks` signature

```go
func parseDevPlanTasks(featureSlug string, content []byte) ([]ProposedTask, bool)
```

- Returns `(tasks, true)` when parsing succeeds and produces ≥1 task.
- Returns `(nil, false)` when the `## Task Breakdown` heading is absent, the section is empty, or parsing yields zero tasks.
- Reuses the existing `slugify` helper for slug generation.
- `ProposedTask.Rationale` is set to `"Sourced from dev-plan task N"` (one-based index).

### `DecomposeFeature` flow change (insertion point)

The new dev plan path is inserted between the spec approval gate (step 3) and the existing AC parse step. When the dev plan path succeeds, the zero-criteria gate and `generateProposal` call are bypassed entirely. Slice enrichment (`analyzeSlices`) runs in all paths.

---

## Files to Read

- `internal/service/decompose.go` — full file; understand `DecomposeFeature`, `generateProposal`, `slugify`, `ProposedTask`, `Proposal`, `GuidanceApplied` constants.
- `internal/service/decompose_test.go` — existing test patterns to follow.
- `internal/service/documents.go` — `DocumentService.ListDocuments`, `GetDocumentContent` signatures.
- `work/spec/p25-decompose-devplan-aware.md` — authoritative requirements.
- `work/design/p25-decompose-devplan-aware.md` — rationale and dependency resolution details.

---

## Notes

- FEAT-01KPQ08Y71A8V (fix empty task names) also modifies `decompose.go`. Sequence this feature after that one lands to avoid merge conflicts, or coordinate branches.
- Do not apply the `"test-tasks-explicit"` guidance rule on the dev plan path (per FR-009 and the design decision).
- The zero-criteria gate moves inside the fallback path — this is intentional (FR-010).

---

## Traceability Matrix

| Requirement | Task |
|-------------|------|
| FR-001 | Task 1 |
| FR-002 | Task 1 |
| FR-003 | Task 1 |
| FR-004 | Task 1 |
| FR-005 | Task 1 |
| FR-006 | Task 1 |
| FR-007 | Task 1 |
| FR-008 | Task 1 |
| FR-009 | Task 1 |
| FR-010 | Task 1 |
| FR-011 | Task 1 |
| FR-012 | Task 1 |
| AC-001 | Task 2 |
| AC-002 | Task 2 |
| AC-003 | Task 2 |
| AC-004 | Task 2 |
| AC-005 | Task 2 |
| AC-006 | Task 2 |
| AC-007 | Task 2 |
| AC-008 | Task 2 |
| AC-009 | Task 2 |
| AC-010 | Task 2 |