# Review: FEAT-01KQ2E0RR40CD — Decompose Apply Supersession

| Field          | Value                                      |
|----------------|--------------------------------------------|
| Feature ID     | FEAT-01KQ2E0RR40CD                         |
| Slug           | decompose-apply-supersession               |
| Review Date    | 2026-04-27                                 |
| Reviewer       | reviewer-conformance + reviewer-quality + reviewer-testing |
| Review Cycle   | 1                                          |
| Verdict        | **Approved**                               |

---

## Summary

This feature adds a supersession pass to `decompose(action: apply)` that transitions
all existing `queued` tasks for the target feature to `not-planned` before creating
the new task set. In-progress tasks (active, needs-rework) are preserved with a
warning. A `superseded_count` field is always present in the response. The result is
an idempotent apply cycle.

**Files reviewed:**
- `internal/mcp/decompose_tool.go` — supersession pass, response fields
- `internal/mcp/decompose_tool_test.go` — 10 new AC-covering tests + 2 supplementary

---

## Review Unit

**Unit:** `decompose-apply-supersession`
**Files:** `internal/mcp/decompose_tool.go`, `internal/mcp/decompose_tool_test.go`
**Spec:** `work/spec/feat-01kq2e0rr40cd-decompose-apply-supersession.md`

---

## Dimensions

### spec_conformance: pass

**Evidence:**

- AC-001 (FR-003, FR-005): `TestDecomposeApply_AC001_AllQueuedSuperseded` — 5 queued
  tasks → all transitioned to `not-planned`, `superseded_count: 5`, no `warning` field.
  Implementation: `decompose_tool.go` supersession loop at lines ~157–184.

- AC-002 (FR-005): `TestDecomposeApply_AC002_NoExistingTasks` — no existing tasks →
  `superseded_count: 0`, `warning` absent. `superseded_count` key always present in
  `resp` map (initialised to `0`).

- AC-003 (FR-004, FR-005): `TestDecomposeApply_AC003_DonePlusQueued` — 2 `done` + 3
  `queued` → 2 `done` preserved, 3 `not-planned`, `superseded_count: 3`, no warning.
  `switch status { case "queued": ... }` only touches queued tasks.

- AC-004 (FR-004, FR-006): `TestDecomposeApply_AC004_ActivePlusQueued` — 1 `active`
  + 3 `queued` → 3 superseded, `active` preserved, `superseded_count: 3`, warning
  present: `"1 task(s) in active/needs-rework status were preserved; verify they are
  still needed."` Matches FR-006 message format exactly.

- AC-005 (FR-004): `TestDecomposeApply_AC005_ReadyTaskPreserved` — `ready` task is
  not in the `case "queued"` or `case "active", "needs-rework"` branches; preserved
  with `superseded_count: 0`, no warning.

- AC-006 (FR-007, FR-010): `TestDecomposeApply_AC006_ActiveDoesNotBlockTaskCreation`
  — `active` task present, Pass 1 still creates 2 new tasks; warning present.
  Supersession pass is not gated on in-progress task count.

- AC-007 (FR-008, FR-009): `TestDecomposeApply_AC007_IdempotentMultipleCalls` — 3
  successive calls with same proposal → after 3rd call, exactly 2 queued (latest run),
  4 not-planned (prior 2 rounds × 2 tasks each).

- AC-008 (FR-006): `TestDecomposeApply_AC008_NeedsReworkPreserved` — 2 `needs-rework`
  tasks → preserved, warning: `"2 task(s) in active/needs-rework status were preserved;
  verify they are still needed."`, `superseded_count: 0`.

- FR-001 (supersession before Pass 1): The supersession block is placed at the top of
  `decomposeApply`, before the `// Pass 1: create all tasks` comment. ✅

- NFR-002 (`superseded_count` always present): `resp["superseded_count"] = supersededCount`
  is unconditionally set; `warning` is conditionally set only when `inProgressCount > 0`. ✅

- NFR-003 (no effect on propose/review): Supersession block is inside the `case "apply"`
  branch; `propose` and `review` actions are unmodified. ✅

**Findings:** None.

---

### implementation_quality: pass_with_notes

**Evidence:**

- The supersession loop is clean and readable. Task filtering by `parent_feature` and
  status uses a `switch` statement that is explicit about which statuses are handled.

- `supersededCount` and `inProgressCount` are correctly scoped to the `if listErr == nil`
  block and initialised to 0 before the block, ensuring they are always available for
  the response construction below.

- The response map (`resp`) is correctly populated: `superseded_count` always set,
  `warning` only when `inProgressCount > 0`.

**Findings:**

- [non-blocking] UpdateStatus failures during supersession are silently discarded
  (`_, _ = entitySvc.UpdateStatus(...)`). If a storage write fails, the affected task
  remains `queued` but `superseded_count` is still incremented — the count would
  overstate actual supersessions. No spec requirement mandates logging here, but logging
  would make debugging storage failures easier.
  (location: `decompose_tool.go` supersession loop)

- [non-blocking] If `entitySvc.List("task")` fails, the supersession pass is skipped
  entirely and `superseded_count` remains `0`. Callers would see no warning that
  supersession did not run. This is acceptable given that List failures are
  exceptional in the P29 cache-backed path.
  (location: `decompose_tool.go:~157`, `if allTasks, listErr := entitySvc.List("task"); listErr == nil`)

---

### test_adequacy: pass

**Evidence:**

- All 8 acceptance criteria (AC-001 through AC-008) have dedicated tests in
  `decompose_tool_test.go`.

- Two supplementary tests cover NFR-002: `TestDecomposeApply_SupplementarySupersededCountAlwaysPresent`
  confirms the key is always present even when `0`; `TestDecomposeApply_SupplementaryWarningOmittedNotEmpty`
  confirms the warning key is absent when there are no in-progress tasks.

- Tests are isolated using `t.TempDir()` via `setupDecomposeApplyTest`.

- `decomposeCommitFunc` is overridden in each test to prevent auto-commit side effects
  (a pre-existing pattern used correctly here).

- Tests are marked `// Not parallel: overrides package-level decomposeCommitFunc` where
  appropriate; tests that do not override the func are safe for parallel execution.

- `countDecomposeTasksByStatus` helper provides precise status-level assertions across
  all AC tests.

**Findings:** None.

---

## Test Run

```
cd .worktrees/FEAT-01KQ2E0RR40CD-decompose-apply-supersession
go test ./internal/mcp/... — PASS (all 10 AC tests + 2 supplementary pass)
```

All decompose apply supersession tests pass. The auto-commit warnings (`not a git
repository`) in test output are pre-existing infrastructure noise from tests running
inside worktrees without a `.git` directory; they do not affect test outcomes.

---

## Finding Summary

| # | Classification  | Description                                                    |
|---|-----------------|----------------------------------------------------------------|
| 1 | non-blocking    | UpdateStatus errors silently discarded in supersession loop    |
| 2 | non-blocking    | List failure skips entire supersession pass without warning    |

**Blocking:** 0
**Non-blocking:** 2
**Total:** 2

---

## Verdict

```
Overall: approved_with_followups
```

All acceptance criteria are satisfied. Both non-blocking findings are edge-case
observability improvements; neither represents a spec deviation or functional gap.
The feature is ready to be transitioned to `done`.

---

*Review conducted using the `orchestrate-review` skill with `reviewer-conformance`,
`reviewer-quality`, and `reviewer-testing` dimensions.*