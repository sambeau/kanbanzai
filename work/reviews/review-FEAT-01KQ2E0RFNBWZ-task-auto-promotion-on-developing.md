# Feature Review: FEAT-01KQ2E0RFNBWZ — Task Auto-Promotion on Developing

| Field           | Value                                             |
|-----------------|---------------------------------------------------|
| Feature ID      | FEAT-01KQ2E0RFNBWZ                                |
| Slug            | task-auto-promotion-on-developing                 |
| Plan            | P34-agent-workflow-ergonomics                     |
| Review Date     | 2026-04-27                                        |
| Reviewer        | reviewer-conformance + reviewer-quality + reviewer-testing |
| Review Cycle    | 1                                                 |
| Verdict         | **approved_with_followups**                       |

---

## Review Unit

| Unit | Files | Spec |
|------|-------|------|
| task-auto-promotion | `internal/service/entities.go`, `internal/service/feature_promotion_hook.go`, `internal/service/feature_promotion_hook_test.go`, `internal/service/promote_queued_tasks_test.go`, `internal/mcp/server.go` | `work/spec/feat-01kq2e0rfnbwz-task-auto-promotion-on-developing.md` |

---

## Dimensions

### spec_conformance: pass

**Evidence:**

- AC-001 (FR-001, FR-006): Two queued tasks with no deps → both promoted to ready. Covered by `TestPromoteQueuedTasks_NoDeps`. `PromoteQueuedTasks` loops all tasks for the feature and calls `validate.ValidateTaskQueuedToReady` which passes for empty `depends_on`.
- AC-002 (FR-007): Queued task with all deps in `done` → promoted. Covered by `TestPromoteQueuedTasks_AllDoneDeps`. `taskStatuses` snapshot maps dep IDs to `"done"` which `ValidateTaskQueuedToReady` treats as terminal.
- AC-003 (FR-008): Queued task with queued dep → stays queued. Covered by `TestPromoteQueuedTasks_BlockedByQueued`. The dep itself has no deps so it is promoted to `ready` (non-terminal), leaving the blocked task in `queued`.
- AC-004 (FR-005): Task already in `ready` → unchanged. `PromoteQueuedTasks` skips tasks whose `status != "queued"` (`entities.go`).
- AC-005 (FR-002): Feature → `reviewing` transition → no promotion. Covered by `TestFeaturePromotionHook_NoFireOnOtherStatus`. `FeaturePromotionHook.OnStatusTransition` early-returns when `toStatus != "developing"`.
- AC-006 (FR-003): Error in one task's promotion → feature transition still succeeds. Covered by `TestPromoteQueuedTasks_FailureIsolation` (corrupt file triggers UpdateStatus failure; loop continues).
- AC-007 (FR-010): Second call with all tasks already ready → no-op, no error. Covered by `TestPromoteQueuedTasks_Idempotent`. Tasks in `ready` are skipped (FR-005 check).
- AC-008 (FR-001, FR-008): Dep-free task promoted; blocked task stays queued. Covered by `TestPromoteQueuedTasks_BlockedByQueued` which verifies both conditions in one test.
- FR-001 (hook registration): `NewFeaturePromotionHook(entitySvc)` is added to `NewCompositeTransitionHook` in `server.go:175`. Fires after the feature's new status is persisted, consistent with ASM-003.
- FR-003 (failure isolation): `FeaturePromotionHook.OnStatusTransition` catches the error from `PromoteQueuedTasks` and logs it, returning `nil`. The hook never propagates. Covered by `TestFeaturePromotionHook_ErrorLogged`.
- FR-009 (failure isolation per task): Per-task `UpdateStatus` failures are logged via `log.Printf` and the loop continues. ✅

**Findings:** None blocking.

---

### implementation_quality: pass_with_notes

**Evidence:**

- `PromoteQueuedTasks` builds a `taskStatuses` snapshot at the top of the function and uses it for dependency evaluation. This snapshot-then-act pattern is correct: tasks promoted during the loop do not retroactively unblock other tasks in the same call, which is the intended behaviour per FR-008.
- `ValidateTaskQueuedToReady` is reused from the existing validation layer (NFR-003), ensuring uniform lifecycle validation.
- The hook is registered via the existing `NewCompositeTransitionHook` pattern (consistent with worktree and dependency hooks), satisfying DEP-001.
- `FeaturePromotionHook.OnStatusTransition` uses `strings.EqualFold` for the entity type check and a direct `== "developing"` comparison for the status, which is correct given that status values are lowercase canonical strings.

**Findings:**

- [non-blocking] `PromoteQueuedTasks` calls `s.List("task")` to enumerate all tasks, then filters by `parent_feature` in Go code. DEP-002 notes a cache-backed filtered list operation should be available from P29. Using the unfiltered list with in-memory filtering achieves the same result and does not introduce a file-scan path (P29 cache makes `List` O(1) from SQLite). However, for projects with very large task counts, in-memory filtering over all tasks is less efficient than a targeted query. No functional impact currently.
  Recommendation: When a task-filtered-by-parent cache query is available, prefer it here.

---

### test_adequacy: pass

**Evidence:**

- `TestPromoteQueuedTasks_NoDeps`, `_AllDoneDeps`, `_BlockedByQueued`, `_AlreadyReady`, `_Idempotent`, `_OnlyOwnFeature`, `_FailureIsolation`: cover ACs 1–8 directly.
- `TestFeaturePromotionHook_FiresOnDeveloping`, `_NoFireOnOtherStatus`, `_NoFireOnNonFeature`, `_ErrorLogged`: cover hook integration at the boundary layer.
- `TestFeaturePromotionHook_NoFireOnOtherStatus` tests 5 non-developing statuses (specifying, reviewing, done, designing, dev-planning) in a single loop, providing coverage breadth without test bloat.
- `TestPromoteQueuedTasks_OnlyOwnFeature` verifies isolation between features sharing a plan — an important regression guard not explicitly required by an AC but necessary for correctness.
- Test file `promote_queued_tasks_test.go` uses shared helpers `setupPromoteTest`, `createTestTask`, `setTestDependsOn`, `advanceTaskTo` which are consistent with existing test conventions.

**Findings:** None.

---

## Finding Summary

| Severity | Count |
|----------|-------|
| Blocking | 0 |
| Non-blocking | 1 |
| Total | 1 |

---

## Test Results

```
go test ./internal/service/... — PASS (cached)
```

All tests pass. No new test failures introduced.

---

## Verdict

| Criterion | Status |
|-----------|--------|
| All ACs implemented and tested | ✅ |
| Spec conformance | ✅ pass |
| Implementation quality | ✅ pass_with_notes |
| Test adequacy | ✅ pass |
| No blocking findings | ✅ |
| Tests pass | ✅ |

**Feature Verdict: approved_with_followups** ✅

The implementation is correct and complete. The single non-blocking finding (in-memory task filtering instead of a targeted query) has no current functional impact and can be addressed when a filtered cache query becomes available.

---

*Review conducted using the `orchestrate-review` + `review-code` skills, `reviewer-conformance` / `reviewer-quality` / `reviewer-testing` roles.*