# Dev Plan: Agentic Reviewing Stage Auto-Advance

**Feature:** FEAT-01KPQ08YE4399
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Spec:** work/spec/p25-agentic-review-auto-advance.md

---

## Overview

Four code changes across four files implement the `require_human_review` config flag and the conditional reviewing-stage auto-advance. Task 1 and Task 3 can proceed in parallel; Task 2 depends on both. Task 4 is independent and can run alongside any other task.

---

## Task Breakdown

### Task 1: Add `RequireHumanReview` to `MergeConfig`

- **Description:** Add the `RequireHumanReview *bool` field and `RequiresHumanReview() bool` accessor to `MergeConfig` in `internal/config/config.go`, following the exact structural pattern of `RequireGitHubPR`.
- **Deliverable:** Updated `internal/config/config.go` with new field and accessor. `DefaultConfig()` unchanged (zero value of `*bool` is nil, which is correct).
- **Depends on:** None (independent)
- **Effort:** Small
- **Spec requirements:** FR-001, FR-002, FR-003

### Task 2: Update `AdvanceFeatureStatus` halt-state branch

- **Description:** Add `RequiresHumanReview func() bool` to `AdvanceConfig` in `internal/service/advance.go`. Modify the `advanceStopStates` halt branch to be conditional: when `RequiresHumanReview` returns false, call `checkAllTasksHaveVerification`; if it passes, continue past `reviewing`; if it fails, halt with a descriptive `StoppedReason`. When `RequiresHumanReview` returns true, preserve the existing halt behaviour.
- **Deliverable:** Updated `internal/service/advance.go` with conditional halt logic and updated `AdvanceConfig` struct.
- **Depends on:** Task 1 (config accessor), Task 3 (checkAllTasksHaveVerification helper)
- **Effort:** Medium
- **Spec requirements:** FR-004, FR-005, FR-008

### Task 3: Add `checkAllTasksHaveVerification` helper

- **Description:** Add `checkAllTasksHaveVerification(feature *model.Feature, entitySvc *EntityService) error` to `internal/service/prereq.go`, following the structural pattern of `checkAllTasksTerminal`. Query all tasks for the feature; return nil if empty or all have non-empty `Verification`; return an error naming the first task without verification. A `needs-review` task with empty verification blocks auto-advance.
- **Deliverable:** Updated `internal/service/prereq.go` with new helper function.
- **Depends on:** None (independent)
- **Effort:** Small
- **Spec requirements:** FR-006

### Task 4: Inject `RequiresHumanReview` in MCP entity tool

- **Description:** Update the `AdvanceConfig` construction in `internal/mcp/entity_tool.go` to read `cfg.Merge.RequiresHumanReview` from the loaded project config and set it as the `RequiresHumanReview` field on `AdvanceConfig`. Follow the same injection pattern used for other config-sourced `AdvanceConfig` fields.
- **Deliverable:** Updated `internal/mcp/entity_tool.go` with new `RequiresHumanReview` field injection.
- **Depends on:** Task 1 (config field exists), Task 2 (AdvanceConfig field exists)
- **Effort:** Small
- **Spec requirements:** FR-007

### Task 5: Tests

- **Description:** Write and update tests covering: auto-advance past `reviewing` when all tasks verified and flag absent; halt at `reviewing` when tasks lack verification and flag absent; halt when flag is true; `checkAllTasksHaveVerification` with zero tasks, all-verified, partially-verified, and `needs-review`-without-verification cases. Update `TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing` to pass `RequiresHumanReview: func() bool { return true }` so it continues to test the mandatory-halt path.
- **Deliverable:** Updated and new tests in `internal/service/advance_test.go` and `internal/service/prereq_test.go`.
- **Depends on:** Task 2, Task 3
- **Effort:** Medium
- **Spec requirements:** NFR-003, NFR-004, AC-01 through AC-10

---

## Interface Contracts

**`AdvanceConfig.RequiresHumanReview`** — the field type is `func() bool`. A nil value is treated as returning false. The MCP layer (Task 4) sets this to `cfg.Merge.RequiresHumanReview` (the accessor method value), which is a zero-allocation method call.

**`checkAllTasksHaveVerification` signature:**
```
func checkAllTasksHaveVerification(feature *model.Feature, entitySvc *EntityService) error
```
Returns nil on success (auto-advance permitted). Returns a non-nil error naming the first unverified task on failure. Called only from within the `advanceStopStates` halt branch in `AdvanceFeatureStatus`.

---

## Execution Order

```
Task 1 ──┐
          ├──► Task 2 ──► Task 5
Task 3 ──┘

Task 4 (depends on Task 1 + Task 2, but is a small injection — can follow Task 2)
```

Tasks 1 and 3 can be implemented in parallel. Task 2 requires both. Task 4 follows Task 2. Task 5 follows Tasks 2 and 3.

---

## Key Files

| File | Change |
|------|--------|
| `internal/config/config.go` | New `RequireHumanReview *bool` field + accessor on `MergeConfig` |
| `internal/service/advance.go` | New `RequiresHumanReview func() bool` on `AdvanceConfig`; conditional halt branch |
| `internal/service/prereq.go` | New `checkAllTasksHaveVerification` helper |
| `internal/mcp/entity_tool.go` | Inject `RequiresHumanReview` into `AdvanceConfig` |
| `internal/service/advance_test.go` | Updated + new tests |
| `internal/service/prereq_test.go` | New tests for `checkAllTasksHaveVerification` |