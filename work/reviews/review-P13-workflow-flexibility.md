# Review: P13 Workflow Flexibility

**Plan:** P13-workflow-flexibility
**Date:** 2026-04-10
**Reviewer:** orchestrator
**Status:** Resolved

**Subject:** P13 Workflow Flexibility implementation — five features addressing the "last mile" gap in
workflow lifecycle completion, direct-to-main workflows, document inheritance, task crash recovery,
and decomposition granularity.

**Specifications reviewed:**
- `work/spec/completion-detection.md` — FEAT-01KN07T66DZ68
- `work/spec/direct-to-main-workflow.md` — FEAT-01KN07T674DZM
- `work/spec/document-inheritance.md` — FEAT-01KN07T66SAH3
- `work/spec/task-crash-recovery.md` — FEAT-01KN07T65HDZM
- `work/spec/decomposition-grouping.md` — FEAT-01KN07T660VVM

**Design document:** `work/design/workflow-completeness.md`
**Implementation plan:** `work/dev-plan/p13-workflow-flexibility.md`

**Scope reviewed:**
- `internal/mcp/status_tool.go` and `status_tool_test.go`
- `internal/health/entity_consistency.go` and `entity_consistency_test.go`
- `internal/health/phase4a.go`
- `internal/validate/lifecycle.go` and `lifecycle_test.go`
- `internal/mcp/merge_tool.go` and `merge_tool_test.go`
- `internal/mcp/pr_tool.go` and `pr_tool_test.go`
- `internal/mcp/doc_tool.go` and `doc_tool_test.go`
- `internal/service/decompose.go` and `decompose_test.go`
- `.kbz/skills/orchestrate-development/SKILL.md`
- `.agents/skills/kanbanzai-agents/SKILL.md`
- `.agents/skills/kanbanzai-workflow/SKILL.md`

---

## Summary

All 19 tasks across 5 features are done. All tests pass (`go test ./...` — 100% green). All five
specification documents are registered and approved against their features. The core implementation
is correct and complete. The review identified 1 gap, 2 improvements, 5 nits, and 1 false positive.

All findings have been resolved. The plan is ready to advance to done.

---

## Findings

---

### [gap] ~~FEAT-01KN07T66SAH3 AC-3 not covered by any test~~ ✅ Resolved

**Location:** `internal/mcp/doc_tool_test.go`
**Spec requirement:** document-inheritance spec AC-3 — "Plan P has approved spec; Feature F has its
own draft spec → spec appears in gaps with feature's draft doc ID and status `draft`; plan's spec
does NOT appear."

`TestDocTool_Gaps_FeatureDocTakesPrecedenceOverPlan` covered the case where *both* the feature and
the plan had an *approved* doc. No test covered the scenario where the feature had a *draft* doc and
the plan had an *approved* doc. This is the most important edge case for FR-2 ("a feature with a
draft spec does NOT inherit the plan's approved spec") because it verifies that the inheritance
suppression is correctly blocked by a feature-owned draft — not only by a feature-owned approved doc.

**Resolution:** Added `TestDocTool_Gaps_FeatureDraftDocBlocksPlanInheritance` to `doc_tool_test.go`.
The test registers a plan-approved spec and a feature-draft spec, calls `doc(action: "gaps")`, and
asserts that the feature's draft appears in `gaps` and the plan's approved doc does not appear as
inherited.

---

### [improvement] ~~`generateProjectAttention` has no direct unit tests~~ ✅ Resolved

**Location:** `internal/mcp/status_tool.go`, `internal/mcp/status_tool_test.go`
**Spec requirement:** completion-detection spec REQ-009/AC-015 (plan completion propagated to
project-level attention), FEAT-01KN07T65HDZM FR-04/AC-04–05 (stuck-task items at project level)

`generateProjectAttention` contained the live stuck-task >24h detection loop and the plan completion
propagation. There were no `TestProjectAttention_*` unit tests targeting this function directly.
Coverage came only via integration-style `TestSynthesiseProject_*` tests.

**Resolution:** Added six direct unit tests for `generateProjectAttention` to `status_tool_test.go`:
- `TestProjectAttention_PlanAllFeaturesDone_NotClosed_Fires`
- `TestProjectAttention_PlanAlreadyDone_NoCloseItem`
- `TestProjectAttention_PlanNotAllFeaturesDone_NoCloseItem`
- `TestProjectAttention_StuckTask_NoDispatchedAt_NotFlagged`
- `TestProjectAttention_StuckTask_RecentDispatch_NotFlagged`
- `TestProjectAttention_StuckTask_OldDispatch_NoGitBranch_Flagged`

---

### [improvement] ~~Two independent stuck-task detection implementations~~ ✅ Resolved

**Location:** `internal/health/phase4a.go` (`CheckStalledDispatches`) and
`internal/mcp/status_tool.go` (`generateProjectAttention`)
**Spec requirement:** FEAT-01KN07T65HDZM FR-04, FR-06

Two separate code paths detected stuck active tasks with different threshold expressions and no
shared predicate, creating a maintenance risk where a future change to the stuck-task logic would
need to be applied in two places.

**Resolution:** Extracted `IsTaskStuck(dispatchedAt time.Time, threshold time.Duration, repoPath,
branch string) bool` as an exported function in `internal/health/phase4a.go`. Updated
`CheckStalledDispatches` and `generateProjectAttention` to both delegate to this single predicate.
The threshold is now explicitly passed at each call site (24h for the MCP path, configurable days
for the health-check path), keeping the two callers' distinct threshold policies visible.

---

### [nit] ~~Private `isTerminalStatus` duplicates `validate.IsTerminalState`~~ ✅ Resolved

**Location:** `internal/mcp/status_tool.go`

`status_tool.go` contained a private `isTerminalStatus` helper with a hardcoded switch statement
that replicated `validate.IsTerminalState(model.EntityKindTask, status)`. A further call site in
`handoff_tool.go` (same package) was also using the private helper.

**Resolution:** Added `"github.com/sambeau/kanbanzai/internal/validate"` to the import block in
`status_tool.go`. Replaced all call sites (`generateFeatureAttention`, `resolveDependencies`,
`handoff_tool.go`) with `validate.IsTerminalState(model.EntityKindTask, ...)`. Deleted the private
`isTerminalStatus` function and its corresponding unit test (which is now fully covered by
`internal/validate/lifecycle_test.go`).

---

### [nit] ~~Grouped task `Summary` and `Rationale` format not asserted in tests~~ ✅ Resolved

**Location:** `internal/service/decompose_test.go`
**Spec requirement:** decomposition-grouping spec AC-13, AC-14

`TestGrouping_MixedSections` verified grouped task count, slug, and `Covers` length but never
asserted the `Summary` string format or `Rationale` content.

**Resolution:** Added assertions to `TestGrouping_MixedSections`:
- AC-13: `sectionATask.Summary == "Implement Section A (3 criteria)"`
- AC-14: `sectionATask.Rationale` contains each of the three AC text strings

---

### [nit] ~~`merge execute` and `pr` no-worktree responses omit `recommendation` field~~ ✅ Resolved

**Location:** `internal/mcp/merge_tool.go`, `internal/mcp/pr_tool.go`
**Spec requirement:** direct-to-main spec FR-1 through FR-5

The asymmetry between `merge check` (returns `{status, reason, recommendation}`) and `merge
execute` / all PR actions (return only `{status, reason}`) was spec-conformant but undocumented,
risking future "fix" attempts.

**Resolution:** Added clarifying comments to the `ErrNotFound` blocks in `executeMerge` and
`createPR` explaining that the omission of `recommendation` is intentional — execute and PR actions
have no follow-up action for the caller when the operation is skipped or not applicable.

---

### [nit] ~~`inTableData` flag not reset on section header transition~~ ✅ False positive

**Location:** `internal/service/decompose.go` — `parseSpecStructure`
**Spec requirement:** decomposition-grouping spec FR-07, FR-08

The review finding claimed that `inTableData` was not reset when the parser encountered a new `##`
section header. On closer inspection, `inTableData = false` is explicitly set at line L339 on the
same code path where `inACSection` is reassigned — both flags are reset together whenever a new
section header is encountered. **No fix required.** The state machine correctly handles this case.

---

### [nit] ~~Zero-timestamp guard silently suppresses stale prefix on old entities~~ ✅ Resolved

**Location:** `internal/mcp/status_tool.go` — `generateFeatureAttention`
**Spec requirement:** completion-detection spec REQ-004, AC-007

The `!featureUpdated.IsZero()` guard silently skipped the stale prefix for entities with no
`updated` field, with no code comment explaining the intent.

**Resolution:** Added a five-line comment directly above the condition explaining that the
`IsZero()` guard is intentional — entities without a populated `updated` field are treated as
"unknown age" rather than "infinitely old", and will not show the ⚠️ STALE prefix. This is accepted
as a defensive correctness trade-off.

---

## Specification Completeness

All requirements in all five specifications are implemented. Summary:

| Spec | Feature ID | FRs | ACs | All Implemented? |
|------|-----------|-----|-----|-----------------|
| completion-detection.md | FEAT-01KN07T66DZ68 | 19 REQs + 2 NFRs | 27 | ✅ Yes |
| direct-to-main-workflow.md | FEAT-01KN07T674DZM | 7 | 12 | ✅ Yes |
| document-inheritance.md | FEAT-01KN07T66SAH3 | 5 | 9 | ✅ Yes |
| task-crash-recovery.md | FEAT-01KN07T65HDZM | 6 | 9 | ✅ Yes |
| decomposition-grouping.md | FEAT-01KN07T660VVM | 8 | 26 | ✅ Yes |

Notable completions confirmed:
- `active → ready` lifecycle transition added to `internal/validate/lifecycle.go`
- `checkGitActivitySince` fully implemented (not a stub) in `internal/health/phase4a.go`
- All three skill files updated with the close-out procedure (Phase 6 in orchestrate-development,
  Feature Completion section in kanbanzai-agents, trigger in kanbanzai-workflow)
- Plan completion attention propagated to project-level (`generateProjectAttention`)
- `CheckPlanChildConsistency` health rule added and registered
- `CheckFeatureChildConsistency` extended to cover `developing` and `needs-rework`

---

## Test Coverage

Tests pass (`go test ./...` — all green). Coverage after fixes:

| Area | Tests | Notes |
|------|-------|-------|
| `generateFeatureAttention` | 10 dedicated unit tests | All ACs covered including zero-task, stale prefix, needs-rework exclusion |
| `generatePlanAttention` | 4 dedicated unit tests | All branches covered |
| `generateProjectAttention` | 6 dedicated unit tests | Added in review — plan completion and stuck-task paths covered |
| `CheckFeatureChildConsistency` | 16 tests | developing + needs-rework extensions covered |
| `CheckPlanChildConsistency` | 5 tests | All 4 branches covered |
| `active → ready` lifecycle | 2 dedicated tests + table-driven suite | Both positive and negative cases explicit |
| `checkGitActivitySince` / `IsTaskStuck` | Unit tests with temp git repo | true/false/error cases covered |
| Merge no-worktree | 8 tests (inner + handler level) | Thorough; broken-store path explicit |
| PR no-worktree | 12 tests (inner function only) | No through-handler tests (low risk; accepted) |
| Doc inheritance | 5 tests | AC-3 (feature draft + plan approved) added in review |
| Decomposition grouping | Full threshold coverage | Summary and rationale assertions added in review |
| Table row extraction | 2 dedicated tests | Header/separator exclusion confirmed |

---

## Workflow Document Currency

All workflow entities accurately reflect the current state:
- All 19 tasks are in `done` status
- All 5 features are in `reviewing` status
- P13 plan is in `reviewing` status
- 5 specification documents are registered and approved against their respective features
- Design document (`work/design/workflow-completeness.md`) and implementation plan
  (`work/dev-plan/p13-workflow-flexibility.md`) are registered and approved against the plan

No stale or inconsistent document records found.

---

## Findings Summary

| ID | Severity | Location | Summary | Status |
|----|----------|----------|---------|--------|
| F-1 | **gap** | `doc_tool_test.go` | AC-3 (feature draft + plan approved) not tested | ✅ Resolved |
| F-2 | **improvement** | `status_tool_test.go` | No direct unit tests for `generateProjectAttention` | ✅ Resolved |
| F-3 | **improvement** | `phase4a.go` + `status_tool.go` | Two independent stuck-task detection implementations | ✅ Resolved |
| F-4 | nit | `status_tool.go` | Private `isTerminalStatus` duplicates `validate.IsTerminalState` | ✅ Resolved |
| F-5 | nit | `decompose_test.go` | Grouped task Summary and Rationale not asserted (AC-13, AC-14) | ✅ Resolved |
| F-6 | nit | `merge_tool.go` + `pr_tool.go` | `execute` and PR responses omit `recommendation` (intentional but undocumented) | ✅ Resolved |
| F-7 | nit | `decompose.go` | `inTableData` not reset on section header transition | ✅ False positive — already handled at L339 |
| F-8 | nit | `status_tool.go` | Zero-timestamp guard silently skips stale prefix on old entities | ✅ Resolved |