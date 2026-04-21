# Implementation Plan: Fix Empty Task Names in `decompose propose`

**Feature:** FEAT-01KPQ08Y71A8V
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Specification:** work/spec/p25-fix-decompose-empty-names.md
**Design:** work/design/p25-fix-decompose-empty-names.md

---

## Overview

Two tasks. All changes are confined to `internal/service/decompose.go` and
`internal/service/decompose_test.go`. Task 1 implements the fix; Task 2 adds
test coverage. Tasks can be developed with Task 2 depending on Task 1's
interface contract.

---

## Interface Contract

The helper introduced in Task 1 has the following signature:

```go
func deriveTaskName(text, fallback string) string
```

- `text`: raw AC text (may be empty, may contain bold-ident prefix)
- `fallback`: caller-supplied fallback string, guaranteed non-empty
- Returns a non-empty string that passes `validate.ValidateName`

Task 2 tests this function directly via package-internal tests.

---

## Task Breakdown

### Task 1: Implement `deriveTaskName` and populate `Name` in `generateProposal`

**Objective:** Add the `deriveTaskName` helper to `internal/service/decompose.go`
and call it in all three task-construction code paths in `generateProposal` so
that every `ProposedTask` has a non-empty, valid `Name` field.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006,
FR-007, FR-008, FR-009, NFR-001, NFR-002, NFR-003

**Input context:**
- `internal/service/decompose.go` — `generateProposal`, `ProposedTask` struct,
  existing `slugify` helper; the three task-construction paths are:
  (a) grouped task (2–4 ACs per L2 section), (b) individual task (1 AC or 5+
  ACs), (c) the unconditionally-appended test task
- `internal/validate/entity.go` — `ValidateName` rules: non-empty, ≤60 chars,
  no colon, no phase prefix
- Spec FR-004: strip pattern is the strict regex `^[A-Z]+-\d+: ` (one or more
  uppercase letters, hyphen, one or more digits, colon, space)
- Spec FR-006: grouped path → `"Implement " + sectionTitle`; empty title →
  `"Implement grouped tasks"`
- Spec FR-007: test task → constant `"Write tests"`

**Output artifacts:**
- Modified `internal/service/decompose.go`:
  - New unexported function `deriveTaskName(text, fallback string) string`
  - `generateProposal` grouped-task path: set `Name = deriveTaskName("Implement "+sectionTitle, "Implement grouped tasks")`
  - `generateProposal` individual-task path: set `Name = deriveTaskName(ac.text, fmt.Sprintf("Implement AC-%03d", taskIndex+i+1))`
  - Test task: set `Name = "Write tests"`

**Dependencies:** None (first task)

---

### Task 2: Write tests for `deriveTaskName` and end-to-end `decompose apply`

**Objective:** Add and update tests in `internal/service/decompose_test.go` to
assert that the `Name` field is always non-empty and valid, that bold-ident ACs
produce colon-free names, and that a full propose→apply round-trip succeeds
without error.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006,
FR-007, AC-01, AC-02, AC-03, AC-04, AC-05

**Input context:**
- `internal/service/decompose_test.go` — existing test suite; extend
  `TestDecomposeFeature_ProposalProduced` to assert `task.Name != ""`
- `internal/validate/entity.go` — `ValidateName` for assertions
- Interface contract from Task 1: `deriveTaskName(text, fallback string) string`

**Output artifacts:**
- Modified `internal/service/decompose_test.go`:
  - Updated `TestDecomposeFeature_ProposalProduced`: add assertion
    `assert.NotEmpty(t, task.Name)` for every task in the proposal
  - New `TestDeriveTaskName_BoldIdentPrefix`: table-driven test covering
    bold-ident format, plain prose with colon, empty input, >60 char input
  - New `TestDecomposeFeature_BoldACSpec_NameHasNoColon`: spec fixture using
    bold-ident format; asserts all task names contain no colon and pass
    `validate.ValidateName`
  - New `TestDecomposeApply_SucceedsWithProposedNames`: calls propose on a
    feature with a valid spec, passes unmodified proposal to `decomposeApply`,
    asserts all tasks created without error

**Dependencies:** Task 1 must complete first

---

## Dependency Graph

```
Task 1 (implement fix)
    └── Task 2 (tests)
```

Tasks are strictly serial: Task 2 requires the interface from Task 1.

---

## Execution Notes

- Both tasks touch only `internal/service/decompose.go` and
  `internal/service/decompose_test.go`. No other files are modified.
- The `FEAT-01KPQ08YBJ5AK` (dev-plan-aware grouping) feature also modifies
  `decompose.go`. If both features are in flight simultaneously, coordinate
  branch merges to avoid conflicts. This feature should land first.
- Run `go test ./internal/service/...` to verify all tests pass.