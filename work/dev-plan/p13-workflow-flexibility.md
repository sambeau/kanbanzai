# Implementation Plan: P13 Workflow Flexibility

**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` (P13-workflow-flexibility/design-workflow-completeness)
**Date:** 2026-04-02

---

## Scope

This plan implements the five features defined under P13-workflow-flexibility. It covers all
requirements in the following specifications:

| Feature | Spec Document |
|---------|--------------|
| FEAT-01KN07T66DZ68 (feature-completion-workflow) | `work/spec/completion-detection.md` |
| FEAT-01KN07T674DZM (main-branch-workflow) | `work/spec/direct-to-main-workflow.md` |
| FEAT-01KN07T66SAH3 (document-inheritance-and-freshness) | `work/spec/document-inheritance.md` |
| FEAT-01KN07T65HDZM (lifecycle-override-and-recovery) | `work/spec/task-crash-recovery.md` |
| FEAT-01KN07T660VVM (decompose-flexibility) | `work/spec/decomposition-grouping.md` |

All five features address the same root cause — incomplete "last mile" lifecycle tooling —
and share code paths in `status_tool.go`, the health subsystem, and the MCP tool surface.
They are planned together so that file-level sequencing constraints are explicit.

**Out of scope:**
- A composite `finish_feature` tool (deferred design decision)
- `checkGitActivitySince` integration with CI systems or remote refs
- Given/When/Then AC format parsing (explicitly deferred in decomposition spec C-03)
- Content hash staleness surfacing in health (deferred in document-inheritance spec)
- Making stall threshold or grouping thresholds user-configurable

---

## Task Breakdown

### Task 1: Feature completion attention items in `status_tool.go`

- **Description:** Add logic to `generateFeatureAttention` that detects when all child tasks
  are in terminal state and the feature is in `developing` or `needs-rework`. Emits the
  attention item `"{display_id} has {N}/{N} tasks done — ready to advance to reviewing"`.
  Applies the `"⚠️ STALE: "` prefix when the feature has been in `developing` for more than
  48 hours (using the `updated` timestamp). Guards against zero-task features.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — feature-level completion detection.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66DZ68 REQ-001 through REQ-005 (AC-001–009)

---

### Task 2: Plan completion detection in `status_tool.go`

- **Description:** Add logic to `synthesisePlan` that detects when all child features are in
  a finished state (`done`, `superseded`, `cancelled`) and the plan is not in `done`. Emits
  `"Plan {display_id} has all {N} features done — ready to close"`. Also propagate this item
  to project-level attention in the project overview path.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — plan-level completion detection.
- **Depends on:** Task 1 (same file; chain changes to avoid conflicts)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66DZ68 REQ-006 through REQ-009 (AC-010–015)

---

### Task 3: Health check extensions in `entity_consistency.go`

- **Description:** Two changes to `internal/health/entity_consistency.go`:
  1. Extend `CheckFeatureChildConsistency` to also flag a warning when a feature is in
     `developing` or `needs-rework` and all child tasks are terminal. Message format follows
     the existing pattern: `"feature {id} has all {N} child task(s) in terminal state but
     feature is {status}"`.
  2. Add a new `CheckPlanChildConsistency` function that warns when all child features of a
     plan are finished but the plan is not `done`, and also when a plan is `done` but has
     non-finished child features. Register the new check with the health runner.
- **Deliverable:** Updated `internal/health/entity_consistency.go` with both changes.
- **Depends on:** None (independent file)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66DZ68 REQ-010 through REQ-015 (AC-016–024)

---

### Task 4: Close-out skill updates

- **Description:** Update three skill files to document the close-out procedure:
  1. `.kbz/skills/orchestrate-development/SKILL.md` — add Phase 6: Close-Out (verify all
     tasks terminal, transition feature, PR if worktree exists, record summary) and checklist
     item `"- [ ] Feature advanced beyond developing"`.
  2. `.agents/skills/kanbanzai-agents/SKILL.md` — add "Feature Completion" section after
     "Finishing Tasks" covering feature transition, PR, merge, and worktree cleanup, with a
     reference to `orchestrate-development` for the full procedure.
  3. `.agents/skills/kanbanzai-workflow/SKILL.md` — add a trigger that ties the close-out
     procedure to the `status` attention item for all-tasks-done.
- **Deliverable:** Three updated skill files.
- **Depends on:** None (documentation only; can proceed before or after code changes)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T66DZ68 REQ-016 through REQ-019 (AC-025–027)

---

### Task 5: Tests for F1 — completion detection and health checks

- **Description:** Write unit tests covering all ACs from the completion-detection spec:
  - `internal/mcp/status_tool_test.go`: test `generateFeatureAttention` for all-terminal,
    mixed-terminal, zero-task, non-qualifying statuses, stale prefix, and no-stale-for-rework
    cases. Test plan-level attention items and project overview propagation.
  - `internal/health/entity_consistency_test.go`: test `CheckFeatureChildConsistency` for
    `developing`/`needs-rework` extension; test `CheckPlanChildConsistency` for all-finished,
    plan-done-with-non-finished, zero-feature, and partial-finished cases.
- **Deliverable:** Updated test files with new test functions covering AC-001 through AC-024.
- **Depends on:** Task 1, Task 2, Task 3
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66DZ68 Verification Plan

---

### Task 6: No-worktree graceful handling in `merge_tool.go` and `pr_tool.go`

- **Description:** Update both tools to return informational results instead of errors when
  the entity has no worktree record (`worktreeStore.GetByEntityID` returns `ErrNotFound`):
  - `merge(action: "check")` → `{"status": "not_applicable", "reason": "...", "recommendation": "..."}`
  - `merge(action: "execute")` → `{"status": "skipped", "reason": "..."}` (no git ops performed)
  - `pr(action: "create|status|update")` → `{"status": "not_applicable", "reason": "..."}`
  Non-`ErrNotFound` store errors still propagate. Invalid entity IDs still error. Worktree-
  present paths are unchanged. Bug entities receive the same treatment as feature entities.
- **Deliverable:** Updated `internal/mcp/merge_tool.go` and `internal/mcp/pr_tool.go`.
- **Depends on:** None (independent files)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T674DZM FR-1 through FR-7 (AC-1–12)

---

### Task 7: Tests for F2 — direct-to-main workflow

- **Description:** Write unit tests covering all ACs from the direct-to-main spec:
  - `internal/mcp/merge_tool_test.go`: `TestCheckMergeReadiness_NoWorktree`,
    `TestExecuteMerge_NoWorktree`, `TestExecuteMerge_NoWorktree_IgnoresOverride`,
    `TestCheckMergeReadiness_WorktreeStoreError`, `TestCheckMergeReadiness_WithWorktree_Unchanged`.
  - `internal/mcp/pr_tool_test.go`: `TestCreatePR_NoWorktree`, `TestGetPRStatusForEntity_NoWorktree`,
    `TestUpdatePR_NoWorktree`, `TestCreatePR_WithWorktree_Unchanged`.
  - Optional integration test: `TestDirectToMainWorkflow_EndToEnd` exercising a lifecycle
    advance without a worktree.
- **Deliverable:** Updated test files covering AC-1 through AC-12.
- **Depends on:** Task 6
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T674DZM Verification Plan

---

### Task 8: Document inheritance helper and `docGapsAction` update

- **Description:** Add a query-time inheritance helper (e.g. `resolveDocumentWithInheritance`)
  to `doc_tool.go` that:
  1. Calls `ListDocumentsByOwner(featureID)` first; if a document of the requested type is
     found (any status), returns it as authoritative.
  2. If not found, retrieves the feature's parent plan ID from `feat.State["parent"]` and
     calls `ListDocumentsByOwner(planID)`, returning only `approved` documents.
  3. Marks the result as `inherited: true` when resolved via fallback.
  Update `docGapsAction` to use this helper so that `doc(action: "gaps")` no longer emits
  false-positive gaps when the parent plan has an approved document of that type.
- **Deliverable:** Updated `internal/mcp/doc_tool.go` with helper and `docGapsAction` changes.
- **Depends on:** None (independent file)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66SAH3 FR-1 through FR-5 (AC-1–3, AC-7–9)

---

### Task 9: Document inheritance in `status_tool.go`

- **Description:** Update two functions in `internal/mcp/status_tool.go` to use the same
  plan-level fallback logic introduced in Task 8:
  1. `synthesisePlan` — when computing `has_spec` and `has_dev_plan` per feature row in the
     plan dashboard, fall back to approved plan docs when the feature has no direct document
     of that type. Suppress false-positive `doc_gaps` entries for features that inherit.
  2. `generateFeatureAttention` (or `synthesiseFeature`) — suppress "Missing specification
     document" and "Missing dev-plan document" attention items when the parent plan has an
     approved document of the relevant type.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — inheritance-aware doc gap checks.
- **Depends on:** Task 1, Task 2 (file coordination), Task 8 (inheritance helper)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66SAH3 FR-1 through FR-5 (AC-4–6, AC-9)

---

### Task 10: Tests for F3 — document inheritance

- **Description:** Write unit tests covering all ACs from the document-inheritance spec:
  - `docGapsAction` tests: fallback to approved plan doc, no fallback for draft plan doc,
    feature-own doc takes precedence, all three doc types inherit, no records written.
  - `synthesisePlan` tests: `has_spec`/`has_dev_plan` reflects inheritance, `doc_gaps`
    suppressed for inherited types.
  - `generateFeatureAttention` tests: attention suppressed when plan has approved doc,
    attention fires when plan has no approved doc.
  - Integration test: plan with approved spec + two feature children, verify no false positives.
- **Deliverable:** Updated test files covering AC-1 through AC-9.
- **Depends on:** Task 8, Task 9
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T66SAH3 Verification Plan

---

### Task 11: `active → ready` lifecycle transition

- **Description:** Add `"ready"` as a valid target state from `"active"` in the
  `allowedTransitions` map in `internal/validate/lifecycle.go` for task entities. No gate
  checks, no reason required, no side-effects beyond the transition itself. Confirm that
  `active → queued` remains rejected.
- **Deliverable:** Updated `internal/validate/lifecycle.go`.
- **Depends on:** None (single-line change, independent file)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T65HDZM FR-01 through FR-03 (AC-01–03, AC-09)

---

### Task 12: Implement `checkGitActivitySince`

- **Description:** Replace the stub `checkGitActivitySince` in `internal/health/phase4a.go`
  with a real implementation. It should run `git log <branch> --after=<timestamp> --oneline`
  (or equivalent) in the given repository path and return `true` if at least one commit is
  found. Return `false` on any error (invalid path, missing branch, git not found) — never
  propagate errors to callers.
- **Deliverable:** Updated `internal/health/phase4a.go`.
- **Depends on:** None (independent file)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T65HDZM FR-05, FR-06 (AC-06–08)

---

### Task 13: Stuck-task attention item in `status_tool.go`

- **Description:** Update the project/feature attention synthesis in `status_tool.go` to
  surface stuck-task attention items. When a task has been in `active` for more than 24 hours
  and `checkGitActivitySince` returns `false` for the parent feature's worktree branch (or
  no worktree exists), emit: `"TASK-xxx has been active for >24h with no recent commits —
  may need unclaim"`. Tasks with recent git activity are not flagged even if older than 24h.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — stuck-task attention items.
- **Depends on:** Task 9 (file coordination), Task 12 (git activity function)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T65HDZM FR-04, FR-06 (AC-04–05)

---

### Task 14: Tests for F4 — task crash recovery

- **Description:** Write unit and integration tests covering all ACs from the crash-recovery
  spec:
  - `internal/validate/` lifecycle tests: `active → ready` accepted, `active → queued`
    rejected, unclaimed task appears in work queue.
  - `internal/health/` tests for `checkGitActivitySince`: temporary git repo with known
    commits, assert true/false for before/after timestamps, assert false for bad path and
    missing branch.
  - Updated `CheckStalledDispatches` tests with functional `checkGitActivitySince`: verify
    tasks with recent activity are not flagged, tasks without activity are.
  - Integration test: claim task → advance to active → transition to ready → verify in queue.
- **Deliverable:** Updated test files covering AC-01 through AC-09.
- **Depends on:** Task 11, Task 12, Task 13
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T65HDZM Verification Plan

---

### Task 15: `Covers` field on `ProposedTask`

- **Description:** Add a `Covers []string` field (JSON key `"covers"`, `omitempty`) to the
  `ProposedTask` struct in `internal/service/decompose.go`. Update all existing call sites
  that construct `ProposedTask` values to populate `Covers` with the single AC text string
  they are responsible for (ungrouped tasks get a one-element slice). The test companion
  task gets a nil/empty `Covers`. Verify JSON serialisation omits the key when empty.
- **Deliverable:** Updated `internal/service/decompose.go` — `Covers` field addition.
- **Depends on:** None (independent file)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T660VVM FR-03 (AC-07–09, AC-22–23)

---

### Task 16: Section-based AC grouping in `generateProposal`

- **Description:** Update `generateProposal` in `internal/service/decompose.go` to group
  acceptance criteria by their `parentL2` value before creating tasks. Apply thresholds:
  - 1 AC → one task (current behaviour, now with `Covers`)
  - 2–4 ACs → one grouped task; slug derived from `slugify(parentL2)` prefixed with feature
    slug; summary `"Implement <section-name> (<N> criteria)"`; rationale lists all AC texts
  - 5+ ACs → one task per AC (current behaviour, now with `Covers`)
  ACs with empty `parentL2` form their own group with key `""`. Update `GuidanceApplied` to
  use `"group-by-section"` when at least one grouped task is produced, otherwise keep
  `"one-ac-per-task"`.
- **Deliverable:** Updated `internal/service/decompose.go` — grouping logic.
- **Depends on:** Task 15 (Covers field must exist)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T660VVM FR-01, FR-02, FR-04, FR-05 (AC-01–06, AC-10–14)

---

### Task 17: Update `checkGaps` for exact `Covers` matching

- **Description:** Update the `checkGaps` review function in `internal/service/decompose.go`
  to use exact string match against `Covers` entries when tasks have a non-empty `Covers`
  field, rather than keyword-overlap heuristics. When `Covers` is nil or empty, fall back to
  the existing keyword-overlap heuristic for backward compatibility with legacy proposals and
  the test companion task.
- **Deliverable:** Updated `internal/service/decompose.go` — gap check update.
- **Depends on:** Task 15 (Covers field must exist)
- **Effort:** Small
- **Spec requirement:** FEAT-01KN07T660VVM FR-06 (AC-15–17)

---

### Task 18: Table row extraction in `parseSpecStructure`

- **Description:** Extend `parseSpecStructure` in `internal/service/decompose.go` to detect
  markdown tables within acceptance-criteria sections and extract each data row as an
  `acceptanceCriterion`. Recognition pattern: header row with `|`, separator row matching
  `| --- |` (hyphens with optional alignment colons), followed by data rows. Each data row's
  cells are joined with ` — ` to form the AC text. Header and separator rows are excluded.
  `section` and `parentL2` are inherited from the enclosing section context, consistent with
  existing formats. Checkbox and table ACs coexist in the same section.
- **Deliverable:** Updated `internal/service/decompose.go` — table parsing.
- **Depends on:** Task 15 (Covers field must exist; table-extracted ACs feed into grouping)
- **Effort:** Medium
- **Spec requirement:** FEAT-01KN07T660VVM FR-07, FR-08 (AC-18–21)

---

### Task 19: Tests for F5 — decomposition grouping

- **Description:** Write unit tests in `internal/service/decompose_test.go` covering all ACs
  from the decomposition-grouping spec:
  - Grouping thresholds: 1, 2, 3, 4, 5, 6 ACs in a single section.
  - Mixed sections: one section with 3 ACs + another with 7 ACs → correct mixed output.
  - `Covers` field: populated on all tasks; nil on test companion; JSON omits key when empty.
  - Grouped task shape: slug derivation, summary format, rationale content.
  - `checkGaps` with Covers: exact match, nil fallback, missing AC produces gap finding.
  - Table parsing: data rows extracted, header/separator excluded, cell join format, mixed
    table+checkbox in same section.
  - Backward compatibility: existing tests pass; legacy proposals without `covers` deserialise
    without error.
  - Race detector: `go test -race ./internal/service/...` passes.
- **Deliverable:** Updated `internal/service/decompose_test.go` covering AC-01 through AC-26.
- **Depends on:** Task 16, Task 17, Task 18
- **Effort:** Large
- **Spec requirement:** FEAT-01KN07T660VVM Verification Plan

---

## Dependency Graph

```
T1  (no deps)
T3  (no deps)
T4  (no deps)
T6  (no deps)
T8  (no deps)
T11 (no deps)
T12 (no deps)
T15 (no deps)

T2  → T1
T7  → T6
T9  → T1, T2, T8        [file coordination on status_tool.go + logic dep on T8]
T10 → T8, T9
T13 → T9, T12           [file coordination on status_tool.go + logic dep on T12]
T5  → T1, T2, T3
T14 → T11, T12, T13
T16 → T15
T17 → T15
T18 → T15
T19 → T16, T17, T18
```

**Parallel groups:**
- **Group A** (start in parallel): T1, T3, T4, T6, T8, T11, T12, T15
- **Group B** (after Group A): T2 (after T1), T7 (after T6), T16 (after T15), T17 (after T15), T18 (after T15)
- **Group C** (after Group B): T5 (after T1, T2, T3), T9 (after T1, T2, T8), T19 (after T16, T17, T18)
- **Group D** (after Group C): T10 (after T8, T9), T13 (after T9, T12)
- **Group E** (after Group D): T14 (after T11, T12, T13)

**Critical path (longest dependency chain):**
T1 → T2 → T9 → T13 → T14 (5 steps, all involving `status_tool.go` coordination)

**Note on `status_tool.go` contention:** Tasks T1, T2, T9, and T13 all modify
`internal/mcp/status_tool.go`. They must be implemented sequentially or on separate branches
with explicit merge ordering. The dependency chain above enforces this. A single implementer
should carry all four tasks through to avoid mid-stream merge conflicts.

---

## Risk Assessment

### Risk: `status_tool.go` merge conflicts across features

- **Probability:** High
- **Impact:** Medium
- **Mitigation:** Assign T1, T2, T9, and T13 to a single implementer working in sequence on
  a single branch. The dependency chain is explicit; do not parallelise these four tasks.
- **Affected tasks:** T1, T2, T9, T13

---

### Risk: `checkGitActivitySince` produces false negatives in CI

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** The spec mandates best-effort, fail-safe behaviour — the function returns
  `false` on any error. False negatives produce spurious stuck-task attention items but never
  block workflow operations. The threshold (24h) is already generous. Ensure the git binary
  is available in the CI environment.
- **Affected tasks:** T12, T13, T14

---

### Risk: Document inheritance introduces false negatives for features with real gaps

- **Probability:** Low
- **Impact:** Medium
- **Mitigation:** FR-2 specifies that a feature's own document (any status, including draft)
  takes unconditional precedence over the plan's document. Only features with no document
  record at all inherit. Unit tests must cover the precedence case (AC-3). The `inherited`
  flag in output allows humans to identify the source of the resolution.
- **Affected tasks:** T8, T9, T10

---

### Risk: Grouping thresholds produce unexpected task counts for existing specs

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** Grouping is applied only during `decompose propose`. The human reviews the
  proposal before `decompose apply`. Thresholds are constants (2–4 group, 5+ ungroup) and
  well-defined in the spec. Backward-compatibility tests (AC-22, AC-23) confirm existing
  proposals without `Covers` continue to work. The `review` step can flag grouped tasks for
  splitting.
- **Affected tasks:** T16, T19

---

### Risk: `active → ready` transition is misused as a general-purpose reset

- **Probability:** Low
- **Impact:** Low
- **Mitigation:** The transition is intentional and documented in skills. The `dispatched_to`
  and `dispatched_at` fields are preserved (not cleared) on unclaim. The next `next(id)` call
  overwrites them on re-claim. No special guard is added — per FR-02 and C-01, simplicity is
  the design intent.
- **Affected tasks:** T11, T14

---

### Risk: Table row extraction in `parseSpecStructure` is fragile for irregular tables

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** The feature is lower priority than grouping (C-05). If table extraction is
  too fragile, T18 can be deferred without affecting T16, T17, or T19 (except table-parsing
  tests). The implementation should be tolerant of alignment colons and variable column counts
  but is not required to handle nested tables or HTML-formatted cells.
- **Affected tasks:** T18, T19

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| FEAT-01KN07T66DZ68 AC-001–009 | Unit tests: `generateFeatureAttention` | T5 |
| FEAT-01KN07T66DZ68 AC-010–015 | Unit/integration tests: plan attention, project overview | T5 |
| FEAT-01KN07T66DZ68 AC-016–019 | Unit tests: `CheckFeatureChildConsistency` extension | T5 |
| FEAT-01KN07T66DZ68 AC-020–024 | Unit tests: `CheckPlanChildConsistency` | T5 |
| FEAT-01KN07T66DZ68 AC-025–027 | Manual review of skill files | T4 |
| FEAT-01KN07T674DZM AC-1–6 | Unit tests: no-worktree returns in merge + pr | T7 |
| FEAT-01KN07T674DZM AC-7–9 | Unit tests: worktree-present regressions unchanged | T7 |
| FEAT-01KN07T674DZM AC-10–12 | Unit tests: invalid entity errors, store error propagation | T7 |
| FEAT-01KN07T66SAH3 AC-1–3, AC-7–8 | Unit tests: `docGapsAction` inheritance | T10 |
| FEAT-01KN07T66SAH3 AC-4, AC-9 | Unit tests: `synthesisePlan` has_spec/doc_gaps | T10 |
| FEAT-01KN07T66SAH3 AC-5–6 | Unit tests: `generateFeatureAttention` suppression | T10 |
| FEAT-01KN07T66SAH3 AC-8 | Negative test: no doc records written to store | T10 |
| FEAT-01KN07T65HDZM AC-01–03, AC-09 | Unit tests: lifecycle map transitions | T14 |
| FEAT-01KN07T65HDZM AC-06–08 | Unit tests: `checkGitActivitySince` with temp git repo | T14 |
| FEAT-01KN07T65HDZM AC-04–05 | Unit tests: `CheckStalledDispatches` with real git activity | T14 |
| FEAT-01KN07T65HDZM AC-01–03 | Integration test: unclaim round-trip via `next()` | T14 |
| FEAT-01KN07T660VVM AC-01–06, AC-10–11 | Unit tests: grouping thresholds and guidance | T19 |
| FEAT-01KN07T660VVM AC-07–09 | Unit tests: `Covers` field population and JSON omitempty | T19 |
| FEAT-01KN07T660VVM AC-12–14 | Unit tests: grouped task slug, summary, rationale | T19 |
| FEAT-01KN07T660VVM AC-15–17 | Unit tests: `checkGaps` exact match and keyword fallback | T19 |
| FEAT-01KN07T660VVM AC-18–21 | Unit tests: table row extraction from `parseSpecStructure` | T19 |
| FEAT-01KN07T660VVM AC-22–23 | Backward-compat tests: no-Covers proposals deserialise cleanly | T19 |
| FEAT-01KN07T660VVM AC-24–26 | All tests pass; `go test -race ./...` passes | T19 |
| All features | `go test ./...` passes on main after each task merge | All |