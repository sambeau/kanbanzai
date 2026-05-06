# Review Report: Decompose Proposal Quality (F2) — Cycle 2

| Field | Value |
|-------|-------|
| Feature | FEAT-01KQTNYN00M4P |
| Reviewer | reviewer-conformance |
| Date | 2026-05-06 |
| Verdict | approved_with_followups |

## Findings Resolution

### BF-1: paired=false opt-out
**Status:** resolved

**Evidence:**

1. **`DecomposeInput` struct** (`internal/service/decompose.go#L14-18`) defines `PairedTestTasks bool` with comment: "when true (default), produce impl+test task pairs per AC".

2. **MCP tool wire-up** (`internal/mcp/decompose_tool.go` decomposePropose handler) reads the flag via `req.GetBool("paired_test_tasks", true)` — defaulting to `true`, passing it through as `input.PairedTestTasks`.

3. **`generateProposal`** (`internal/service/decompose.go` function signature at ~L586) accepts `pairedTestTasks bool` and gates paired test task generation on it. In both the grouped-task path and the individual-task path, the paired test task creation is guarded by `if pairedTestTasks && !allCoversContainTest(implTask.Covers)`.

4. **Test `TestPairedTestTasks_OptOut_OneTaskPerAC`** (`internal/service/decompose_test.go#L2085`) calls `generateProposal(spec, "feat", "", 0, false)` and verifies:
   - 3 ACs produce exactly 3 tasks (one per AC)
   - No task slug ends with `-tests`
   - Test passes on current codebase

## Acceptance Criteria Verification

| Criterion | Spec | Test | Status |
|-----------|------|------|--------|
| AC-001 | Refuse-to-propose when no ACs | `TestDecomposeFeature_NoACs_ReturnsError` | ✅ PASS |
| AC-002 | 3 ACs → 6 tasks (default paired) | `TestPairedTestTasks_AC001_ThreeImplACs` | ✅ PASS |
| AC-003 | Test task `depends_on` → impl task | `TestPairedTestTasks_AC002_TestTaskFields` | ✅ PASS |
| AC-004 | `paired=false` → 1 task per AC | `TestPairedTestTasks_OptOut_OneTaskPerAC` | ✅ PASS |
| AC-005 | Dependency graph: no partial-completion edges | `TestPairedTestTasks_DependencyGraphCompleteNodes`, `_NoPartialCompletionEdges` | ✅ PASS |
| AC-006 | Testing-concern AC → single test task | `TestTestingConcern_*` (all 10 variants) | ✅ PASS |
| REQ-NF-001 | No measurable latency from refuse-to-propose | Conditional on already-parsed spec; existing tests show no perf regression | ✅ PASS |
| REQ-NF-002 | Default callers still receive valid proposals | All existing tests pass; task count change is by design | ✅ PASS |

All specification acceptance criteria pass their corresponding tests.

## Remaining Findings

### F1 (minor): Stale test `TestPairedTestTasks_OptOutFlagNotYetImplemented`

`internal/service/decompose_test.go#L3724` contains a test whose name, godoc, and TODO comment claim the opt-out flag is "not yet implemented":

```
// TestPairedTestTasks_OptOutFlagNotYetImplemented verifies the default
// behavior (paired output) and documents the expected behavior for the
// paired_test_tasks=false flag via skipped subtests. The flag does not
// exist yet in DecomposeInput or generateProposal, so the opt-out path
// cannot be tested.
```

The test body only validates the **default** (paired=true) path and logs:
```
t.Log("TODO: opt-out flag (paired_test_tasks=false) not yet implemented")
```

This is misleading — the flag is fully implemented and tested in `TestPairedTestTasks_OptOut_OneTaskPerAC`. The test name and comments are obsolete. This does not affect functionality but creates confusion for future maintainers.

**Recommendation:** Rename to `TestPairedTestTasks_DefaultPairedMode` and remove the stale TODO comment. The default-mode validation in the test body is still useful.

## Verdict

**`approved_with_followups`** — BF-1 is resolved. All six functional acceptance criteria (AC-001 through AC-006) and both non-functional requirements (REQ-NF-001, REQ-NF-002) are satisfied with passing tests. One minor follow-up: clean up the stale `TestPairedTestTasks_OptOutFlagNotYetImplemented` test name and comments (F1).
