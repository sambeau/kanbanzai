# Review: FEAT-01KQ2E0RHSNP1 — Next Tool UX Improvements

| Field           | Value                                              |
|-----------------|----------------------------------------------------|
| Feature ID      | FEAT-01KQ2E0RHSNP1                                 |
| Slug            | idempotent-task-claim                              |
| Review Date     | 2026-04-27                                         |
| Reviewer        | reviewer-conformance + reviewer-quality + reviewer-testing |
| Review Cycle    | 1                                                  |
| Verdict         | **approved_with_followups**                        |

---

## Summary

Two small, independently motivated improvements to `next_tool.go`:

1. **Idempotent task claim** — when `next(id: TASK-...)` is called for a task
   already in `active` status, the tool now returns the assembled context packet
   with `reclaimed: true` instead of an error. This recovers gracefully from
   timeout-induced double-claims.

2. **Worktree path in context packet** — the context packet response is extended
   with a `worktree_path` field containing the filesystem path of the active
   worktree for the parent feature, eliminating a redundant `worktree(action: get)`
   round-trip.

---

## Review Unit

| Unit    | Files                                              | Spec Sections |
|---------|----------------------------------------------------|---------------|
| main    | `internal/mcp/next_tool.go`                        | §4, §6        |
|         | `internal/mcp/next_tool_ac_test.go` (new)          |               |
|         | `internal/mcp/next_tool_test.go` (updated)         |               |

---

## Spec Conformance

**Overall dimension outcome: pass**

Evidence:

- **AC-001 (FR-001, FR-002)** — `next_tool.go`: When `status == "active"`, the
  code sets `isReclaim = true` and falls through to context assembly (no error
  return). Response includes `result["reclaimed"] = true`. Verified by
  `TestNextClaimMode_AlreadyActive_ReturnsContextPacket`.

- **AC-002 (FR-003)** — `next_tool.go`: The entire claim block (including
  `DispatchTask` call that would overwrite dispatch metadata) is guarded by
  `if !isReclaim { ... }`. Original `dispatched_to` and `claimed_at` are
  preserved. Verified by `TestNextClaimMode_AlreadyActive_PreservesDispatchMeta`.

- **AC-003 (FR-004)** — `next_tool.go`: `DispatchTask`, `PushSideEffect`, and
  the reload are all inside `if !isReclaim { ... }`. No hook is re-fired on
  reclaim. No side effects emitted on reclaim. Verified by
  `TestNextClaimMode_AlreadyActive_NoHookRefired`.

- **AC-004 (FR-005)** — `next_tool.go`: `done` status hits the `default` branch
  which returns an error unchanged. Verified by
  `TestNextClaimMode_DoneTask_StillErrors`.

- **AC-005 (FR-005)** — `next_tool.go`: `queued` status also hits the `default`
  branch which returns an error unchanged. Verified by
  `TestNextClaimMode_QueuedTask_StillErrors`.

- **AC-006 (FR-006)** — `next_tool.go` (`nextContextToMap`): `worktree_path`
  is added to the output map when `actx.worktreePath != ""`. Verified by
  `TestNextTool_WorktreePath_Present` in `next_tool_test.go`.

- **AC-007 (FR-007)** — `nextContextToMap`: `worktree_path` is omitted (key
  absent) when `actx.worktreePath == ""`. Verified by
  `TestNextTool_WorktreePath_Absent` in `next_tool_test.go`.

- **AC-008 (FR-006, FR-001)** — Reclaim path follows the same `nextContextToMap`
  call as first-claim path; `worktree_path` is available if an active worktree
  exists. Verified by `TestNextClaimMode_AlreadyActive_ReturnsContextPacket`
  (context field present) combined with worktree-path tests.

- **AC-009 (FR-002)** — `next_tool.go`: `reclaimed` field is only added when
  `isReclaim == true`. On first-claim, `isReclaim = false` and the field is never
  set. Verified by `TestNextClaimMode_FirstClaim_NoReclaimedField`.

**All 9 acceptance criteria pass.** ✅

---

## Implementation Quality

**Overall dimension outcome: pass_with_notes**

Evidence:

- `isReclaim` flag cleanly separates the first-claim and reclaim paths. The
  `switch/case` on `status` remains readable, with the reclaim path clearly
  annotated with `// Already claimed — fall through and return context with reclaimed: true`.

- `ValidateFeatureStage` is correctly skipped on reclaim (FR-004). `featureStage`
  is declared as `var featureStage string` before the `if !isReclaim` block so
  it compiles correctly.

- `nextContextToMap` receives an `assembledContext` and adds `worktree_path`
  unconditionally via a single `if actx.worktreePath != ""` guard, consistent
  with how other optional fields (`tool_hint`, `graph_project`) are handled in
  the same function (NFR-002).

- The `resolveShortPlanRef` helper previously added to `entity_tool.go` is not
  present here (not in scope; short-ref resolution is a separate feature).

Findings:

- [non-blocking] On reclaim, `featureStage` is the empty string because
  `ValidateFeatureStage` is skipped. `assembleContext` receives an empty
  `featureStage` and therefore omits stage-binding context hints from the
  reclaim response. The spec requires the "assembled context packet" (AC-001)
  but does not define stage hints as required fields. The task and context
  fields are present in tests. This is a minor fidelity gap in reclaim
  responses — stage binding guidance is absent.
  (spec: FR-001, location: `next_tool.go` reclaim path)
  Recommendation: On a future improvement, consider looking up the feature
  stage independently during reclaim context assembly, or documenting that
  reclaim context omits stage hints intentionally.

---

## Test Adequacy

**Overall dimension outcome: pass**

Evidence:

- New file `next_tool_ac_test.go` (255 lines) provides dedicated tests for
  AC-001 through AC-005 and AC-009 with clear per-AC test names and struct
  comments.

- `next_tool_test.go` was updated to add worktree import and retains the
  pre-existing `TestNext_ClaimByTaskID_AlreadyActive` test (now verifying
  reclaim success rather than error return).

- All AC-001 through AC-009 are traceable to test functions.

- Edge cases covered: done task error (AC-004), queued task error (AC-005),
  no-reclaim field on first claim (AC-009), metadata preservation (AC-002),
  no side-effect emission on reclaim (AC-003).

- The intermittent failure of `TestDocIntelFind_EntityID_RelatedKnowledge_Dedup`
  observed during one test run is a pre-existing flaky test unrelated to this
  feature. It passes consistently when isolated and passes on main. This is a
  known issue (tracked separately) and is not a finding for this review.

---

## Finding Summary

| # | Classification   | Description                                                         |
|---|------------------|---------------------------------------------------------------------|
| 1 | non-blocking     | Reclaim context omits stage-binding hints (empty featureStage)      |

**Blocking: 0 | Non-blocking: 1 | Total: 1**

---

## Test Run

```
cd .worktrees/FEAT-01KQ2E0RHSNP1-idempotent-task-claim
go test ./internal/mcp/... — PASS (all new AC tests pass)
go test ./internal/mcp/... -run TestDocIntelFind_EntityID_RelatedKnowledge_Dedup -count=3 — PASS (3/3)
```

Pre-existing intermittent failure in `TestDocIntelFind_EntityID_RelatedKnowledge_Dedup`
is not a regression introduced by this feature.

---

## Verdict

| Criterion                           | Status |
|-------------------------------------|--------|
| All tasks in terminal state         | ✅ (3/3 done) |
| Specification approved              | ✅ |
| All acceptance criteria pass        | ✅ (9/9) |
| No blocking findings                | ✅ |
| No introduced test regressions      | ✅ |

**Feature Verdict: approved_with_followups** ✅

Feature FEAT-01KQ2E0RHSNP1 (Next Tool UX Improvements) passes review.
One non-blocking follow-up item: stage binding hints are absent from reclaim
responses. The feature may proceed to done.

---

*Review conducted using the `review-code` skill with combined reviewer-conformance,
reviewer-quality, and reviewer-testing roles.*