# Conformance Review: FEAT-01KR3MDSZKAFG — High-Violation MCP Rule Invariants

**Reviewer Role:** reviewer-conformance
**Review Unit:** FEAT-01KR3MDSZKAFG
**Spec:** `work/B59-enforce-high-violation-rules-mcp-invariants/B59-F1-spec-high-violation-mcp-rule-invariants.md`
**Dev-Plan:** `work/B59-enforce-high-violation-rules-mcp-invariants/plan/B59-F1-dev-plan.md`
**Date:** 2026-05-08

## Files Reviewed

| File | Task | Purpose |
|------|------|---------|
| `internal/invariants/catalog.go` | T1 | Invariant codes, RefusalResponse, Format |
| `internal/invariants/catalog_test.go` | T1 | Round-trip JSON, byte-length, code constant tests |
| `internal/mcp/next_tool.go` | T2, T3, T4 | INV-002 refusal, INV-003 dirty check, INV-004 tool description |
| `internal/mcp/handoff_tool.go` | T2, T4 | INV-002 refusal, INV-004 tool description |
| `internal/git/dirty.go` | T3 | CheckKbzDirty: git status porcelain parser |
| `internal/git/dirty_test.go` | T3 | 8 tests covering clean, dirty, multi-dir, excluded paths, committed files, non-existent repo |
| `internal/mcp/assembly.go` | T4 | nextContextToMap workflow_state_warning field |
| `internal/context/pipeline.go` | T4 | RenderPrompt INV-004 invariants section injection |
| `internal/mcp/invariant_boundary_test.go` | T6 | 6 boundary tests (5 active, 1 skipped pending P44) |
| `.kbz/roles/orchestrator.yaml` | T5 | spawn_agent removal, INV-001 vocabulary, INV-005 alignment |
| `.kbz/skills/orchestrate-development/SKILL.md` | T5 | INV-001 and INV-004 prose de-duplication |

## Overall: needs_remediation

## Dimensions

### spec_conformance: pass_with_notes

**Evidence:**
- **AC-001:** Five invariant codes defined in `internal/invariants/catalog.go` L7-13. `TestInvariantCodes` verifies all five map to `"INV-001"` through `"INV-005"`.
- **AC-003:** `nextClaimMode` L211-219 returns INV-002 via `invariants.Format` when `entitySvc.Get("task", ...)` fails. `handoffTool` L96-102 returns INV-002 on task lookup failure. `TestInvariant_INV002_Next_UnregisteredTask` passes.
- **AC-004:** `nextClaimMode` L240-250 performs dirty-state check via `checkKbzDirtyFunc`, returns INV-003 with file list. Skipped on reclaim. `TestInvariant_INV003_Next_OrphanedState` passes.
- **AC-005:** `nextContextToMap` L455 always includes `workflow_state_warning` with INV-004 text. `nextTool` description L83-84 and `handoffTool` description L70-71 include INV-004. `RenderPrompt` L971-979 injects `## Invariants` section unconditionally. `TestInvariant_INV004_ContextWarning_Next` and `TestInvariant_INV004_ContextWarning_Handoff` pass.
- **AC-007:** INV-001 defined with no override path. No `override` field in `RefusalResponse` struct. `spawn_agent` removed from orchestrator tools list.
- **AC-008:** "Manual Prompt Composition" anti-pattern replaced with INV-001 cross-reference (SKILL.md L88-89). Phase 1 constraint block replaced with INV-004 cross-reference (SKILL.md L131-139). `spawn_agent` removed from orchestrator tools.
- **AC-009:** `RefusalResponse` struct has all four fields (Code, Operation, Reason, NextAction). `Format` serializes all four. `TestFormat_RoundTrip` verifies for all 5 invariant codes.
- **AC-010:** Six test functions in `invariant_boundary_test.go`. Five pass, one correctly skipped (INV-002 handoff pending P44).
- **AC-011:** `TestFormat_ByteLengthUpperBound` (catalog) and `TestInvariant_RefusalSize` (mcp) both pass. All refusals ≤ 1,200 bytes.

**Findings:**
- **[non-blocking] INV-002 coverage gap for feature/plan IDs in next** — `nextResolveTaskID` resolves feature/plan IDs to tasks before the INV-002 check, so a genuinely unregistered feature/plan ID produces a different error format rather than INV-002. The task-ID path is fully covered and this is the primary use case. (spec: AC-003, location: `next_tool.go` L482-517)
  - Recommendation: Either extend INV-002 to feature/plan lookup failures in `nextResolveTaskID`/`nextFindTopReadyTask`, or update the spec to clarify that INV-002 applies at the task-lookup boundary specifically.

- **[non-blocking] AC-002 pending P44 dependency** — Full verification that `dispatch_task` replaces `spawn_agent` at runtime cannot be completed until P44 lands. `TestInvariant_INV002_Handoff_UnregisteredTask` is correctly skipped with a clear dependency note. (spec: AC-002, location: `invariant_boundary_test.go` L56)
  - Recommendation: Re-run this review after P44's dispatch_task is registered.

- **[non-blocking] AC-006 deferred to P56** — INV-005 catalog entry defines artefact gate enforcement as mandatory, and gate text in role/skill files uses "Hard constraint (ℋ)" framing. However, the actual gate check logic for bugs is owned by P56 and is not implemented here. This is per spec: "P56 owns bug lifecycle gate checks." (spec: AC-006, location: N/A — design boundary)
  - Recommendation: No action needed at this time. P56 will close this loop.

### implementation_quality: concern

**Evidence:**
- **Clean package design:** `internal/invariants` is a focused, single-responsibility package. `internal/git/dirty.go` reuses existing `runGitCmd` from `commit.go`.
- **Consistent testability pattern:** `checkKbzDirtyFunc` (next_tool.go) and `commitStateFunc` (handoff_tool.go) both use package-level variable stubbing for test injection.
- **Proper reclaim skipping:** INV-003 dirty check is correctly skipped when `isReclaim == true`, preserving existing reclaim behavior.
- **Deterministic refusal format:** `Format` produces stable JSON with fixed field ordering.

**Findings:**
- **[blocking] Data race on shared `checkKbzDirtyFunc` in invariant boundary tests** — `TestInvariant_INV003_Next_OrphanedState` and `TestInvariant_INV004_ContextWarning_Next` both call `t.Parallel()` and both mutate the package-level `checkKbzDirtyFunc` variable. When tests run with default parallelism (`go test ./internal/mcp/...`), the two goroutines race on the shared variable, causing `TestInvariant_INV004_ContextWarning_Next` to receive the INV-003 stub's dirty file list and fail with a wrong-error-code response. Tests pass when run serially (`-p 1`) or in isolation. (location: `internal/mcp/invariant_boundary_test.go` L16-26, L82-122, L125-155)
  - Recommendation: Convert `checkKbzDirtyFunc` from a package-level variable to a field on a test-injectable struct (e.g., a `nextToolConfig`), or remove `t.Parallel()` from the two conflicting tests, or use a mutex-protected getter pattern (like a `sync/atomic.Value`). The same pattern used for `commitStateFunc` in `handoff_tool.go` has the same latent risk if handoff tests run in parallel.

### test_adequacy: pass_with_notes

**Evidence:**
- **T1 tests:** `TestFormat_RoundTrip` (5 subtests, all four JSON fields verified), `TestFormat_ByteLengthUpperBound` (5 codes at 400-byte reason limit), `TestInvariantCodes` (5 code constants).
- **T3 tests:** 8 test cases in `dirty_test.go` covering clean, dirty, multi-dir, excluded paths, committed files, non-existent repo, index/, context/.
- **T6 tests:** `TestInvariant_INV002_Next_UnregisteredTask`, `TestInvariant_INV003_Next_OrphanedState`, `TestInvariant_INV004_ContextWarning_Next`, `TestInvariant_INV004_ContextWarning_Handoff`, `TestInvariant_RefusalSize` — all 5 active tests pass. `TestInvariant_INV002_Handoff_UnregisteredTask` correctly skipped.
- **Test isolation quality:** Tests create temp repos, stub git calls, and use `t.Cleanup` for restoration.

**Findings:**
- **[non-blocking] No test for INV-003 dirty-check error path** — `CheckKbzDirty` can return an error (e.g., when `git status` fails). The `nextClaimMode` code handles this (`if dirtyErr == nil && len(dirtyFiles) > 0`), silently ignoring the error case. A test covering the error-return path (stub returns error) would improve coverage. (location: `next_tool.go` L241)
  - Recommendation: Add a test case where `checkKbzDirtyFunc` returns an error, verifying the claim proceeds normally (errors are non-blocking per the current implementation).

## Finding Summary

| Severity | Count |
|----------|-------|
| Blocking | 1 |
| Non-blocking | 4 |
| **Total** | **5** |

## Detailed Findings

### Finding 1 (blocking): Test race condition on shared `checkKbzDirtyFunc`

- **Dimension:** implementation_quality
- **Spec anchor:** AC-010 (tests must pass reliably)
- **Location:** `internal/mcp/invariant_boundary_test.go` L16-26, L125-155
- **Description:** `TestInvariant_INV003_Next_OrphanedState` and `TestInvariant_INV004_ContextWarning_Next` both use `t.Parallel()` and both mutate the package-level `checkKbzDirtyFunc`. This creates a data race — the INV-004 test can receive the INV-003 test's dirty-file stub, causing it to receive an INV-003 refusal instead of the expected context map. The failure is non-deterministic but reproducible with `go test ./internal/mcp/ -run 'Invariant' -count=5`.
- **Evidence:** Observed failure at 2026-05-08T18:43: `response missing 'context' map; got type <nil> response: map[error:map[code:internal_error message:{"error":{"code":"INV-003"...}}]]`
- **Remediation:** Either (a) remove `t.Parallel()` from these two tests, (b) convert `checkKbzDirtyFunc` to be passed as a parameter rather than a global, or (c) use `sync/atomic.Value` for the stub with proper cleanup ordering. Option (b) is preferred for architectural consistency — it would also address the latent risk with `commitStateFunc` in `handoff_tool.go`.

### Finding 2 (non-blocking): INV-002 coverage gap for feature/plan IDs in next

- **Dimension:** spec_conformance
- **Spec anchor:** AC-003, REQ-005
- **Location:** `internal/mcp/next_tool.go` L482-517
- **Description:** When `next` is called with a feature or plan ID that doesn't exist in the store, `nextResolveTaskID` → `nextFindTopReadyTask` produces an error that does not use the INV-002 format. Only task IDs get the structured INV-002 refusal. The spec says all entity identifier types should produce INV-002.
- **Remediation:** Extend INV-002 to `nextFindTopReadyTask` and `CrossEntityQuery` failure paths, or update AC-003 to scope INV-002 to the task-lookup boundary specifically.

### Finding 3 (non-blocking): AC-002 pending P44 dependency

- **Dimension:** spec_conformance
- **Spec anchor:** AC-002
- **Location:** `internal/mcp/invariant_boundary_test.go` L56
- **Description:** The handoff INV-002 test is correctly skipped with a clear dependency note. `spawn_agent` is removed from `orchestrator.yaml`, but full dispatch-path verification requires P44's `dispatch_task` tool to be live.
- **Remediation:** No code change needed. Re-verify when P44 lands.

### Finding 4 (non-blocking): AC-006 deferred to P56

- **Dimension:** spec_conformance
- **Spec anchor:** AC-006
- **Location:** N/A — design boundary
- **Description:** INV-005 defines artefact gate enforcement as mandatory. Gate text uses "Hard constraint (ℋ)" framing. However, bug gate check logic is owned by P56. This is an intentional design boundary per the spec.
- **Remediation:** No action needed. P56 will complete the implementation.

### Finding 5 (non-blocking): No test for INV-003 dirty-check error path

- **Dimension:** test_adequacy
- **Spec anchor:** AC-010
- **Location:** `internal/mcp/next_tool.go` L241
- **Description:** `nextClaimMode` silently ignores errors from `checkKbzDirtyFunc` (the condition is `if dirtyErr == nil && len(dirtyFiles) > 0`). No test verifies behavior when `checkKbzDirtyFunc` returns an error. While this is a reasonable design choice (don't block on git errors), it should have test coverage.
- **Remediation:** Add a test case where `checkKbzDirtyFunc` returns an error and verify the claim proceeds normally.

## Test Results

```
=== internal/invariants: PASS (3 test functions, 7 subtests)
=== internal/git:        PASS (8 CheckKbzDirty tests)
=== internal/context:    PASS (all pipeline tests including TestRenderPrompt)
=== internal/mcp:        PASS (5 active invariant tests, 1 skipped)
```

Note: Invariant tests pass reliably when run with `-p 1` (serial package execution). The race condition described in Finding 1 causes intermittent failures with default parallelism.
