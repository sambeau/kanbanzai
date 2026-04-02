# Specification: Direct-to-Main Workflow

**Status:** Draft
**Feature:** FEAT-01KN07T674DZM (main-branch-workflow)
**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` — Feature 3
**Date:** 2026-04-02

---

## Problem Statement

The `merge` and `pr` tools hard-fail when called for an entity that has no worktree:

```
Cannot execute merge for FEAT-xxx: no worktree exists for this entity.
```

When implementation is committed directly to the default branch — which is how most solo development and small changes work in practice — the entire merge/PR toolchain is a dead end. The feature lifecycle *can* advance from `developing` → `reviewing` → `done` without merge/PR steps (the stage gates do not enforce merge artifacts), but calling `merge` or `pr` during a close-out procedure produces a hard error that confuses agents and blocks automated workflows.

The system should treat direct-to-main as a first-class workflow. Tools that are not applicable should return informational results, not errors.

## Requirements

### Functional Requirements

**FR-1: `merge(action: "check")` returns not-applicable when no worktree exists.**

When `merge(action: "check")` is called for an entity that has no worktree record in the worktree store, the tool returns a successful result (not an error) with the following shape:

```json
{
  "status": "not_applicable",
  "reason": "no worktree exists — work was committed directly to the default branch",
  "recommendation": "advance the feature lifecycle directly"
}
```

The entity ID is still validated — an invalid entity ID (not prefixed with `FEAT-` or `BUG-`) still returns an error.

**FR-2: `merge(action: "execute")` returns skipped when no worktree exists.**

When `merge(action: "execute")` is called for an entity that has no worktree record, the tool returns a successful result with `"status": "skipped"` and an explanatory reason. No merge, branch deletion, or post-merge install is performed. The `override`, `override_reason`, `merge_strategy`, and `delete_branch` parameters are accepted but ignored.

**FR-3: `pr(action: "create")` returns not-applicable when no worktree exists.**

When `pr(action: "create")` is called for an entity that has no worktree record, the tool returns a successful result (not an error) indicating that no PR is needed because work was committed directly to the default branch. The `draft` parameter is accepted but ignored.

**FR-4: `pr(action: "status")` returns not-applicable when no worktree exists.**

When `pr(action: "status")` is called for an entity that has no worktree record, the tool returns a successful result indicating that no PR exists because work was committed directly to the default branch.

**FR-5: `pr(action: "update")` returns not-applicable when no worktree exists.**

When `pr(action: "update")` is called for an entity that has no worktree record, the tool returns a successful result indicating that no PR exists to update.

**FR-6: Worktree-based flows are unchanged.**

When a worktree exists for the entity, all five operations (`merge check`, `merge execute`, `pr create`, `pr status`, `pr update`) behave exactly as they do today. The no-worktree handling is only reached when `worktreeStore.GetByEntityID` returns `worktree.ErrNotFound`.

**FR-7: Non-worktree errors propagate unchanged.**

If `worktreeStore.GetByEntityID` returns an error other than `worktree.ErrNotFound` (e.g. store corruption, I/O failure), that error is propagated as-is. Only `ErrNotFound` triggers the graceful skip path.

## Constraints

**C-1: No schema changes.** No new fields are added to entity state, worktree records, or any YAML files. The behaviour is determined entirely by the presence or absence of a worktree record.

**C-2: No new tool parameters.** No `direct_commit` flag or similar parameter is added to `merge` or `pr`. The tools detect the workflow mode automatically from the worktree store.

**C-3: No lifecycle gate changes.** The stage gate prerequisites for `developing → reviewing` (all tasks terminal) and `reviewing → done` (report document) are unchanged. This feature only changes the `merge` and `pr` tool responses.

**C-4: Return shape consistency.** Each tool's not-applicable/skipped response is returned as a successful `map[string]any` result (not an error). The response always includes `"status"` and `"reason"` keys so that callers can distinguish the three outcomes: success, skipped/not-applicable, and error.

## Acceptance Criteria

**AC-1: merge check with no worktree**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `merge(action: "check", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns a successful result with `status` equal to `"not_applicable"`, a `reason` string explaining that no worktree exists, and a `recommendation` to advance the lifecycle directly

**AC-2: merge execute with no worktree**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `merge(action: "execute", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns a successful result with `status` equal to `"skipped"` and a `reason` string explaining that no worktree exists
- **And** no git merge, branch deletion, or post-merge install is performed

**AC-3: merge execute with no worktree ignores override parameters**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `merge(action: "execute", entity_id: "FEAT-xxx", override: true, override_reason: "test")` is called
- **Then** the tool returns the same skipped result as AC-2 without error

**AC-4: pr create with no worktree**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `pr(action: "create", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns a successful result with `status` equal to `"not_applicable"` and a `reason` string explaining that no worktree exists

**AC-5: pr status with no worktree**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `pr(action: "status", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns a successful result with `status` equal to `"not_applicable"` and a `reason` string explaining that no worktree exists

**AC-6: pr update with no worktree**

- **Given** a feature entity `FEAT-xxx` exists and no worktree record exists for `FEAT-xxx`
- **When** `pr(action: "update", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns a successful result with `status` equal to `"not_applicable"` and a `reason` string explaining that no worktree exists

**AC-7: merge check with existing worktree unchanged**

- **Given** a feature entity `FEAT-xxx` exists and a worktree record exists for `FEAT-xxx` with branch `feat/xxx`
- **When** `merge(action: "check", entity_id: "FEAT-xxx")` is called
- **Then** the tool returns the existing gate-check result format (with `overall_status`, per-gate results, and summary counts)

**AC-8: merge execute with existing worktree unchanged**

- **Given** a feature entity `FEAT-xxx` exists and a worktree record exists for `FEAT-xxx` with branch `feat/xxx`
- **When** `merge(action: "execute", entity_id: "FEAT-xxx")` is called
- **Then** the tool performs the merge as it does today (gate checks, git merge, optional branch deletion, post-merge install)

**AC-9: pr create with existing worktree unchanged**

- **Given** a feature entity `FEAT-xxx` exists and a worktree record exists for `FEAT-xxx` with branch `feat/xxx`
- **When** `pr(action: "create", entity_id: "FEAT-xxx")` is called
- **Then** the tool creates a GitHub pull request as it does today

**AC-10: invalid entity ID still errors**

- **Given** no entity with ID `NOTVALID-123` exists
- **When** `merge(action: "check", entity_id: "NOTVALID-123")` is called
- **Then** the tool returns an error (not a not-applicable result)

**AC-11: worktree store error propagates**

- **Given** `worktreeStore.GetByEntityID` returns an error that is not `worktree.ErrNotFound` (e.g. I/O error)
- **When** any `merge` or `pr` action is called for that entity
- **Then** the original error is propagated, not converted to a not-applicable result

**AC-12: bug entities supported**

- **Given** a bug entity `BUG-xxx` exists and no worktree record exists for `BUG-xxx`
- **When** `merge(action: "check", entity_id: "BUG-xxx")` is called
- **Then** the tool returns the same not-applicable result as for features (AC-1)

## Verification Plan

**Unit tests in `internal/mcp/merge_tool_test.go`:**

1. `TestCheckMergeReadiness_NoWorktree` — calls `checkMergeReadiness` with a worktree store that returns `ErrNotFound`. Asserts the result contains `"status": "not_applicable"` and no error is returned.
2. `TestExecuteMerge_NoWorktree` — calls `executeMerge` with a worktree store that returns `ErrNotFound`. Asserts the result contains `"status": "skipped"` and no error is returned.
3. `TestExecuteMerge_NoWorktree_IgnoresOverride` — same as above but with `override: true` and an `override_reason`. Asserts the same skipped result.
4. `TestCheckMergeReadiness_WorktreeStoreError` — calls `checkMergeReadiness` with a worktree store that returns a non-`ErrNotFound` error. Asserts the error is propagated.
5. `TestCheckMergeReadiness_WithWorktree_Unchanged` — calls `checkMergeReadiness` with a valid worktree. Asserts the existing gate-check result format is returned (regression guard).

**Unit tests in `internal/mcp/pr_tool_test.go`:**

6. `TestCreatePR_NoWorktree` — calls `createPR` with a worktree store that returns `ErrNotFound`. Asserts the result contains `"status": "not_applicable"` and no error.
7. `TestGetPRStatusForEntity_NoWorktree` — calls `getPRStatusForEntity` with a worktree store that returns `ErrNotFound`. Asserts the result contains `"status": "not_applicable"` and no error.
8. `TestUpdatePR_NoWorktree` — calls `updatePR` with a worktree store that returns `ErrNotFound`. Asserts the result contains `"status": "not_applicable"` and no error.
9. `TestCreatePR_WithWorktree_Unchanged` — calls `createPR` with a valid worktree (regression guard for AC-9).

**Integration test (optional, in `internal/mcp/` or a dedicated test file):**

10. `TestDirectToMainWorkflow_EndToEnd` — creates a feature, advances it through `developing → reviewing → done` without creating a worktree, calling `merge(action: "check")` and `pr(action: "create")` along the way. Asserts that both return not-applicable results and the lifecycle completes successfully.