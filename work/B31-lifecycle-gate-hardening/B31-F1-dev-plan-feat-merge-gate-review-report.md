| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-24                     |
| Status | Draft                          |
| Author | architect                      |

# Dev Plan: Non-bypassable Merge Gate for Review Report

## Scope

This plan implements the requirements defined in
`work/specs/spec-feat-merge-gate-review-report.md` (FEAT-01KPXGVQY3KQC). It covers
tasks T1–T8 below.

**In scope:**
- Adding `Bypassable bool` to `merge.GateResult` in `internal/merge/gate.go`.
- Backfilling `Bypassable: true` on all 7 existing gate condition implementations in
  `internal/merge/gates.go`.
- Adding a `DocService` interface and `DocSvc` field to `GateContext` in
  `internal/merge/gate.go` and `internal/merge/gates.go`.
- Implementing `ReviewReportExistsGate` in `internal/merge/gates.go`.
- Wiring `ReviewReportExistsGate` into `DefaultGates()` and adding a
  `NonBypassableBlockingFailures` helper in `internal/merge/checker.go`.
- Threading `*service.DocumentService` through the `MergeTool` call chain
  (`internal/mcp/merge_tool.go`, `internal/mcp/server.go`) and populating
  `GateContext.DocSvc`.
- Enforcing that `override: true` is rejected in `executeMerge` when any blocking gate
  has `Bypassable: false`.
- Unit tests for `ReviewReportExistsGate` (AC-001, AC-003, AC-004, AC-005).
- Integration and regression tests for the non-bypassable override path and backward
  compatibility (AC-002, AC-006, AC-007).

**Out of scope:**
- `OrphanedReviewingFeatureCheck` in the status dashboard (FEAT-01KPXGW5BCGY4).
- Changes to any other gate condition's override behaviour beyond the mechanical
  `Bypassable: true` backfill.
- Requiring the review report document to be in `approved` status.
- The `merge(action: check)` response format — the check action will naturally
  show `ReviewReportExistsGate` results because it calls `DefaultGates()` via
  `CheckGates`, but no additional formatting changes are required.

---

## Task Breakdown

### Task 1: Add `Bypassable bool` to `merge.GateResult`

- **Description:** Add a `Bypassable bool` field to the `GateResult` struct in
  `internal/merge/gate.go`. This is the structural foundation for REQ-001. Because Go's
  zero value for `bool` is `false`, this change alone would silently make all existing
  gates non-bypassable; Task 2 corrects that.
- **Deliverable:** `internal/merge/gate.go` with `Bypassable bool` in `GateResult`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-001.

### Task 2: Backfill `Bypassable: true` on all existing gate conditions

- **Description:** Update every existing `Check()` method in `internal/merge/gates.go`
  to explicitly set `Bypassable: true` in the initial `GateResult` struct literal. The
  seven gates are: `EntityDoneGate`, `TasksCompleteGate`, `VerificationExistsGate`,
  `VerificationPassedGate`, `BranchNotStaleGate`, `NoConflictsGate`,
  `HealthCheckCleanGate`. Each requires a single-line addition. Also scan
  `internal/merge/*_test.go` for any `GateResult{…}` struct literals constructed without
  `Bypassable` and add `Bypassable: true` to preserve test intent; missing the field
  would cause tests constructing expected `GateResult` values to fail after Task 1.
- **Deliverable:** `internal/merge/gates.go` with all 7 gates returning
  `Bypassable: true`; any affected test files updated.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-002.

### Task 3: Add `DocService` interface and `DocSvc` field to `GateContext`

- **Description:** Define a minimal `DocService` interface in `internal/merge/gate.go`:

  ```
  type DocService interface {
      ListDocuments(owner, docType string) ([]DocRecord, error)
  }
  ```

  Also define a minimal `DocRecord` struct (ID and Status fields) in the same file.
  Then add an optional `DocSvc DocService` field to `GateContext` in
  `internal/merge/gates.go`. The field is optional (nil-safe): gates that do not need
  the document service ignore it.

  The `DocService` interface method signature must be satisfied by
  `*service.DocumentService` via an adapter shim added in Task 6 (the existing
  `service.DocumentService.ListDocuments` method takes a `service.DocumentFilters`
  struct, not two string parameters, so a thin adapter will be needed in
  `internal/mcp/merge_tool.go`).
- **Deliverable:** `internal/merge/gate.go` with `DocService` interface and `DocRecord`
  type; `internal/merge/gates.go` with `DocSvc DocService` field on `GateContext`.
- **Depends on:** None (independent of Task 1).
- **Effort:** Small.
- **Spec requirement:** REQ-003, REQ-004, REQ-005, REQ-006 (prerequisite).

### Task 4: Implement `ReviewReportExistsGate`

- **Description:** Add the `ReviewReportExistsGate` struct and its `Gate` interface
  implementation to `internal/merge/gates.go`:
  - `Name()` returns `"review_report_exists"`.
  - `Severity()` returns `GateSeverityBlocking`.
  - `Check(ctx GateContext) GateResult`:
    1. Read `ctx.Entity["status"]`. If it is not `"reviewing"`, return
       `GateResult{…, Status: GateStatusPassed, Bypassable: true}` immediately
       (REQ-008, REQ-NF-002).
    2. If `ctx.DocSvc` is `nil`, return Pass (fail-open; prevents panics during
       tests or check-only calls that do not populate DocSvc).
    3. Call `ctx.DocSvc.ListDocuments(ctx.EntityID, "report")`.
    4. If the call returns an error: log a warning (using the standard `log` package
       or the slog pattern used elsewhere in the package) and return Pass (REQ-006).
    5. If the result set contains ≥ 1 document: return Pass (REQ-004).
    6. Return `GateResult{Name: …, Severity: …, Status: GateStatusFailed,
       Bypassable: false, Message: <REQ-007 wording>}` (REQ-005).

  The `Message` field MUST contain the exact wording from REQ-007, with the actual
  feature ID substituted for `FEAT-xxx`.
- **Deliverable:** `internal/merge/gates.go` with `ReviewReportExistsGate` fully
  implemented.
- **Depends on:** Task 1, Task 3.
- **Effort:** Medium.
- **Spec requirement:** REQ-003, REQ-004, REQ-005, REQ-006, REQ-007, REQ-008,
  REQ-NF-001, REQ-NF-002.

### Task 5: Wire gate into `DefaultGates()` and add `NonBypassableBlockingFailures` helper

- **Description:** Two changes to `internal/merge/checker.go`:
  1. Append `ReviewReportExistsGate{}` to the slice returned by `DefaultGates()`.
     Order: add it after `HealthCheckCleanGate{}` and before `BranchNotStaleGate{}`
     (i.e., after all unconditional blocking gates and before the warning gate).
  2. Add a new exported helper:
     ```
     func NonBypassableBlockingFailures(results []GateResult) []GateResult
     ```
     Returns the subset of results where `Severity == GateSeverityBlocking`,
     `Status == GateStatusFailed`, and `Bypassable == false`. Used by `executeMerge`
     in Task 6.
- **Deliverable:** `internal/merge/checker.go` with updated `DefaultGates()` and new
  helper.
- **Depends on:** Task 4.
- **Effort:** Small.
- **Spec requirement:** REQ-003 (wiring).

### Task 6: Thread `docSvc` through merge tool and enforce non-bypassable override rejection

- **Description:** Two interleaved concerns in the merge tool layer, both in
  `internal/mcp/merge_tool.go` and `internal/mcp/server.go`.

  **6a — Thread `*service.DocumentService`:**
  - Add `docSvc *service.DocumentService` parameter to `MergeTool`, `mergeTool`,
    `mergeCheckAction`, `mergeExecuteAction`, `checkMergeReadiness`, and
    `executeMerge`.
  - In both `checkMergeReadiness` and `executeMerge`, populate `GateContext.DocSvc`
    with a thin adapter that satisfies the `merge.DocService` interface by calling
    `docSvc.ListDocuments(service.DocumentFilters{Owner: owner, Type: docType})`.
  - In `server.go`, pass `docRecordSvc` as the new argument to `MergeTool(…)`.

  **6b — Enforce non-bypassable override rejection:**
  - In `executeMerge`, after `gateResult := merge.CheckGates(gateCtx)`, add a check:
    ```
    if nonBypassable := merge.NonBypassableBlockingFailures(gateResult.Gates);
       len(nonBypassable) > 0 {
        // Return error with first non-bypassable failure's Message verbatim.
        // This runs BEFORE the existing override check, and override is irrelevant.
    }
    ```
  - The existing `if gateResult.OverallStatus == merge.OverallStatusBlocked && !override`
    block remains unchanged for bypassable gates.
  - The override audit-log block (recording `OverrideRecords` for bypassed gates) must
    continue to work for bypassable gates; no change needed there.

- **Deliverable:** Updated `internal/mcp/merge_tool.go` and `internal/mcp/server.go`;
  non-bypassable override rejection verified to fire before the existing override path.
- **Depends on:** Task 3, Task 5.
- **Effort:** Medium.
- **Spec requirement:** REQ-001 (override enforcement), REQ-NF-001, REQ-NF-002.

### Task 7: Unit tests for `ReviewReportExistsGate`

- **Description:** Add test functions to `internal/merge/gates_test.go` covering:
  - Gate skips (returns Pass) when entity status is not `"reviewing"` — AC-005.
  - Gate returns Pass when `DocSvc.ListDocuments` returns ≥ 1 report (test with
    status `"draft"` and `"approved"` variants) — AC-003.
  - Gate returns `GateStatusFailed` with `Bypassable: false` and the REQ-007 message
    when the result set is empty — AC-001.
  - Gate returns Pass and does not return `GateStatusFailed` when `DocSvc.ListDocuments`
    returns an error — AC-004.
  - Gate returns Pass when `DocSvc` is `nil` (nil-safety / fail-open for check-only
    callers).
  Use a table-driven test with a stub `DocService` implementation local to the test
  file. Do not use `DefaultGates()`; test the gate struct directly.
- **Deliverable:** New test functions in `internal/merge/gates_test.go`.
- **Depends on:** Task 4, Task 5.
- **Effort:** Medium.
- **Spec requirement:** AC-001, AC-003, AC-004, AC-005.

### Task 8: Integration and regression tests

- **Description:** Add a new test file `internal/mcp/merge_tool_bypassable_test.go`
  with tests that exercise the full `executeMerge` path:
  - **AC-002 / REQ-001:** With a feature in `"reviewing"` status and no registered
    report, call `executeMerge` with `override: true`. Verify the call returns an error
    and the error message contains the REQ-007 wording and the phrase
    `"cannot be bypassed with override: true"`. A stub `DocService` returning empty
    results is sufficient; no real repo needed (use the existing pattern of injecting
    stub gate functions via `GateContext`).
  - **AC-006 / REQ-002:** With a feature blocked by an existing gate that has
    `Bypassable: true` (e.g., `entity_done` with status `"developing"`), call
    `executeMerge` with `override: true` and a valid reason. Verify the call succeeds
    past the gate check phase (i.e., the override is accepted for the bypassable gate
    and the error, if any, is from a later phase such as branch/git operations, not from
    the gate).
  - **AC-007 / REQ-007:** Inspect the error string returned by AC-002 and assert it
    contains the feature ID, the three numbered resolution steps, and the
    `"override: true"` bypass disclaimer. This can be a sub-assertion within the AC-002
    test case.
  Follow the pattern established in `internal/mcp/merge_tool_pr_gate_test.go` for
  setting up a test worktree store, entity service, and injecting stub functions via
  package-level vars where needed.
- **Deliverable:** `internal/mcp/merge_tool_bypassable_test.go` with AC-002, AC-006,
  AC-007 coverage.
- **Depends on:** Task 2, Task 6, Task 7.
- **Effort:** Medium.
- **Spec requirement:** AC-002, AC-006, AC-007.

---

## Dependency Graph

```
T1 (Add Bypassable to GateResult)          T3 (DocService interface + GateContext.DocSvc)
│                                          │
├──► T2 (Backfill Bypassable: true)        │
│                                          │
└──────────────────────────────────────────┤
                                           ▼
                                       T4 (Implement ReviewReportExistsGate)
                                           │
                                           ▼
                                       T5 (Wire into DefaultGates; NonBypassableBlockingFailures helper)
                                          ╱ ╲
                                         ▼   ▼
                           T6 (Thread docSvc;  T7 (Unit tests for
                           non-bypassable       ReviewReportExistsGate)
                           override rejection)
                                 │               │
                                 └───────┬───────┘
                                         │
                                 T2 ─────┤
                                         ▼
                                     T8 (Integration + regression tests)
```

**Parallel groups:**
- `{T1, T3}` — independent, no shared file
- `{T2}` — after T1 only; can start before T3 completes
- `{T4}` — after both T1 and T3 (T2 should complete before T4 to avoid merge conflicts in `gates.go`)
- `{T6, T7}` — after T5; T6 also requires T3 (already done); both can proceed in parallel

**Critical path:** T1 → T4 → T5 → T6 → T8

(T3 is on the critical path in practice because T4 requires T3; whichever of T1 or T3
finishes later gates T4. Both are small so the bottleneck is T4 onward.)

---

## Risk Assessment

### Risk: Go zero-value breaks existing gate tests

- **Probability:** High.
- **Impact:** Medium (CI failure, not a logic error).
- **Mitigation:** Task 2 explicitly requires scanning `internal/merge/*_test.go` for
  `GateResult{}` literals and adding `Bypassable: true`. Run `go test ./internal/merge/…`
  after Task 1 is committed to surface failures before proceeding to later tasks.
- **Affected tasks:** T1, T2, T8.

### Risk: `DocService` interface mismatch with `service.DocumentService`

- **Probability:** Medium.
- **Impact:** Medium (compilation failure; no logic risk).
- **Mitigation:** The adapter shim in Task 6 is the only coupling point. Define the
  `merge.DocService` interface in terms of (owner, docType string) parameters and write
  the shim in `merge_tool.go` against the concrete `service.DocumentService.ListDocuments`
  signature. Verify compilation after Task 6 before writing tests. If the
  `service.DocumentService` signature changes in a concurrent branch, update only the
  shim.
- **Affected tasks:** T3, T6.

### Risk: `executeMerge` existing test `TestMergeExecute_RequireGitHubPR_True_NoPR_Blocked` uses `override: true` to bypass gates

- **Probability:** Low (existing bypass is for bypassable gates).
- **Impact:** Low (test would need a nil or pass-through DocSvc to avoid triggering the
  new gate).
- **Mitigation:** After Task 6, check that the entity used in that test is not in
  `"reviewing"` status (it is not; it is a minimal stub entity). `ReviewReportExistsGate`
  returns Pass immediately for non-reviewing entities (REQ-008), so the test is unaffected.
  Add a comment in the test noting the dependency.
- **Affected tasks:** T6, T8.

### Risk: Exact REQ-007 message wording drifts between gate and merge tool

- **Probability:** Low.
- **Impact:** Medium (AC-007 inspection test will fail; user-facing message is wrong).
- **Mitigation:** The full REQ-007 message string is produced once, inside
  `ReviewReportExistsGate.Check()`, and stored in `GateResult.Message`. The merge tool
  passes this string through verbatim (it does not reformat it). AC-007 in Task 8
  asserts the exact substrings. Define the message as a named constant or package-level
  `var` in `gates.go` so that both the gate and the test reference the same source.
- **Affected tasks:** T4, T8.

### Risk: `checkMergeReadiness` (merge check action) needs `DocSvc` too

- **Probability:** Certain (identified in scope analysis).
- **Impact:** Low-medium (without DocSvc, `ReviewReportExistsGate` would fail-open via
  nil-check in T4; the check action would show the gate as `passed` even when it should
  show `blocked`).
- **Mitigation:** Task 6 explicitly threads `docSvc` into `checkMergeReadiness` as well
  as `executeMerge`, populating `GateContext.DocSvc` in both paths.
- **Affected tasks:** T6.

---

## Verification Approach

| Acceptance Criterion | Verification Method                | Producing Task |
|---------------------|------------------------------------|----------------|
| AC-001: reviewing + no report → blocked with correct message | Unit test (`gates_test.go`) | Task 7 |
| AC-002: reviewing + no report + `override: true` → rejected | Integration test (`merge_tool_bypassable_test.go`) | Task 8 |
| AC-003: reviewing + ≥1 report (draft and approved) → passes | Unit test (`gates_test.go`) | Task 7 |
| AC-004: reviewing + `ListDocuments` error → pass + log warning | Unit test (`gates_test.go`) | Task 7 |
| AC-005: non-reviewing feature → gate not evaluated | Unit test (`gates_test.go`) | Task 7 |
| AC-006: existing bypassable gate + `override: true` → still accepted | Integration test (`merge_tool_bypassable_test.go`) | Task 8 |
| AC-007: error message contains feature ID, 3-step instructions, bypass disclaimer | Code inspection + assertion in AC-002 test | Tasks 4, 8 |