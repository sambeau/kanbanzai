# Dev Plan: Standalone bugs visible in status health

| Field      | Value                                                         |
|------------|---------------------------------------------------------------|
| Feature    | FEAT-01KPPG3MSRRCE — Standalone bugs visible in status health |
| Spec       | `work/spec/p24-standalone-bug-visibility.md`                  |
| Plan       | P24-retro-recommendations                                     |
| Status     | Draft                                                         |

---

## Overview

This plan implements the requirements defined in
`work/spec/p24-standalone-bug-visibility.md` for feature FEAT-01KPPG3MSRRCE.

The change is a targeted, additive block inside `synthesiseProject`
(`internal/mcp/status_tool.go`) that iterates all bugs after
`generateProjectAttention` and the health-check block complete, and appends an
`AttentionItem` for each bug that is open, standalone (`origin_feature == ""`),
and high/critical severity. No new abstractions, interfaces, or types are
introduced.

This plan covers two tasks:

- **T1** — Implement the standalone-bug attention block in `synthesiseProject`.
- **T2** — Write unit tests covering all acceptance criteria (AC-001–AC-012).

It does not cover plan-scope or feature-scope changes, changes to
`generateProjectAttention`, or any `AttentionItem` schema modifications.

---

## Task Breakdown

### Task 1: Implement standalone-bug attention block

- **Description:** Add the standalone-bug iteration block to `synthesiseProject`
  in `internal/mcp/status_tool.go`. The block calls `entitySvc.List("bug")`,
  filters to bugs where `origin_feature == ""`, status is not resolved, and
  severity is `"high"` or `"critical"`, then appends an `AttentionItem` for
  each match. The block must be placed after the health-check attention items
  are appended. Error from `List("bug")` is silently ignored (best-effort).

- **Deliverable:** Modified `internal/mcp/status_tool.go`.

- **Depends on:** None (independent — can start immediately).

- **Effort:** Small.

- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005,
  REQ-006, REQ-007, REQ-NF-001, REQ-NF-002, REQ-NF-003.

**Implementation notes:**

  - Insert after the closing `}` of the health-check block (the block that
    appends `health_error` and `health_warning` items), before the
    `return &projectOverview{...}` statement.
  - Filter conditions (all must hold):
    1. `origin_feature` absent or `== ""`
    2. `status` not in `{"done","closed","not-planned","duplicate","wont-fix"}`
    3. `severity == "high"` or `severity == "critical"`
  - `AttentionItem` fields:
    - `Type`: `"open_critical_bug"`
    - `Severity`: `"warning"`
    - `EntityID`: bug ID string
    - `DisplayID`: `id.FormatFullDisplay(bugID)`
    - `Message`: `"Standalone <severity> bug: <name>"`;
      fallback to `"Standalone <severity> bug: <bugID>"` when name is empty.
  - The `if allBugs, bugErr := entitySvc.List("bug"); bugErr == nil { ... }`
    guard pattern (used in `synthesiseFeature`) is the correct error-handling
    idiom — use it here too.

---

### Task 2: Write unit tests for standalone-bug visibility

- **Description:** Add unit tests to `internal/mcp/status_tool_test.go` that
  verify all acceptance criteria for the standalone-bug block. Tests use the
  existing `setupStatusTest` / `synthesiseProject` pattern already in the file.
  A `createStatusTestBug` helper function should be introduced to keep
  individual test functions concise.

- **Deliverable:** New test functions in `internal/mcp/status_tool_test.go`.

- **Depends on:** Task 1 (tests exercise the implemented block).

- **Effort:** Small.

- **Spec requirements:** AC-001 through AC-012 (all acceptance criteria).

**Test cases to cover:**

  | Test function | AC | Description |
  |---------------|----|-------------|
  | `TestSynthesiseProject_StandaloneBug_HighSeverity` | AC-001 | Standalone high bug → item present with correct fields |
  | `TestSynthesiseProject_StandaloneBug_CriticalSeverity` | AC-002 | Standalone critical bug → item present |
  | `TestSynthesiseProject_StandaloneBug_EmptyName` | AC-003 | Empty name → Message uses bug ID |
  | `TestSynthesiseProject_StandaloneBug_FeatureLinked_Excluded` | AC-004 | Feature-linked bug → NOT in project attention |
  | `TestSynthesiseProject_StandaloneBug_Closed_Excluded` | AC-005 | Closed standalone bug → NOT in project attention |
  | `TestSynthesiseProject_StandaloneBug_ResolvedStatuses_Excluded` | AC-006 | Parameterised over `done`, `not-planned`, `duplicate`, `wont-fix` → none appear |
  | `TestSynthesiseProject_StandaloneBug_MediumSeverity_Excluded` | AC-007 | Medium-severity standalone bug → NOT in project attention |
  | `TestSynthesiseProject_StandaloneBug_LowSeverity_Excluded` | AC-008 | Low-severity standalone bug → NOT in project attention |
  | `TestSynthesiseProject_StandaloneBug_OrderingPreserved` | AC-009 | Pre-existing items come before standalone-bug items |
  | `TestSynthesiseProject_StandaloneBug_ListError_Ignored` | AC-012 | `List("bug")` failure → status call still succeeds |

  AC-010 and AC-011 (plan scope / feature scope) are verified by inspection:
  the standalone-bug block exists only inside `synthesiseProject`, not in
  `synthesisePlan` or `synthesiseFeature`. A brief inline comment in the
  implementation (e.g. `// project scope only — see spec REQ-007`) makes this
  reviewable.

**Helper function:**

  ```go
  // createStatusTestBug creates a bug for status tests.
  // Set originFeature to "" for standalone bugs.
  func createStatusTestBug(t *testing.T, entitySvc *service.EntityService,
      slug, name, severity, originFeature string) string
  ```

  Because `CreateBug` does not accept `origin_feature`, the helper should
  write `origin_feature` directly to the storage record after creation, using
  the same `entitySvc.Store().Write(record)` pattern used by other test helpers
  in `status_tool_test.go`. Alternatively, read the created record, mutate the
  field, and re-write it.

---

## Dependency Graph

```
T1: Implement standalone-bug block   (no dependencies — start immediately)
T2: Write unit tests                 → depends on T1
```

**Execution order:**

- T1 must complete before T2 begins (tests exercise T1's implementation).
- No parallelism is available given the two-task sequence.

**Critical path:** T1 → T2

---

## Interface Contracts

No new public interfaces, exported types, or cross-package APIs are introduced
by this change.

All types and functions used by T1 already exist in `status_tool.go`:

| Symbol | Package | Notes |
|--------|---------|-------|
| `entitySvc.List("bug")` | `internal/service` | Returns `[]ListResult`; already used in `synthesiseFeature` |
| `AttentionItem{...}` | `internal/mcp` | Struct unchanged; `Type: "open_critical_bug"` reused |
| `id.FormatFullDisplay` | `internal/id` | Already imported in `status_tool.go` |

The `createStatusTestBug` helper in T2 is test-internal (unexported). It does
not constitute a contract with other packages.

---

## Traceability Matrix

| Spec Requirement | Implemented by | Verified by |
|-----------------|----------------|-------------|
| REQ-001 | T1 | AC-001, AC-002 (T2) |
| REQ-002 | T1 | AC-001, AC-002, AC-003 (T2) |
| REQ-003 | T1 | AC-009 (T2) |
| REQ-004 | T1 | AC-004 (T2) |
| REQ-005 | T1 | AC-005, AC-006 (T2) |
| REQ-006 | T1 | AC-007, AC-008 (T2) |
| REQ-007 | T1 | AC-010, AC-011 (Inspection) |
| REQ-NF-001 | T1 | AC-012 (T2) |
| REQ-NF-002 | T1 | Inspection (no struct changes) |
| REQ-NF-003 | T1 | AC-009 (T2) |

All 10 requirements are covered. All 12 acceptance criteria are assigned to
a producing task.

---

## Risk Assessment

### Risk: `origin_feature` absent vs. empty-string

- **Probability:** Low
- **Impact:** Medium — a bug with `origin_feature` absent (key missing from
  state map) rather than set to `""` might not be caught if the filter uses
  strict equality on the string value.
- **Mitigation:** The filter must use `b.State["origin_feature"].(string)` with
  a zero-value fallback (as the design shows). A missing key yields `""` by Go's
  type assertion default, so absent and empty-string are handled identically.
  Add an explicit test case with an absent `origin_feature` key (covered by
  AC-001).
- **Affected tasks:** T1, T2.

### Risk: Test helper cannot set `origin_feature` via `CreateBug`

- **Probability:** Confirmed (CreateBug does not accept the field).
- **Impact:** Low — the workaround (read-mutate-rewrite via the storage layer)
  is already the established pattern in `status_tool_test.go`.
- **Mitigation:** Follow the existing `createTestPlan` pattern in T2: write
  the storage record directly. Document the pattern in a helper comment.
- **Affected tasks:** T2.

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 | Unit test (`TestSynthesiseProject_StandaloneBug_HighSeverity`) | T2 |
| AC-002 | Unit test (`TestSynthesiseProject_StandaloneBug_CriticalSeverity`) | T2 |
| AC-003 | Unit test (`TestSynthesiseProject_StandaloneBug_EmptyName`) | T2 |
| AC-004 | Unit test (`TestSynthesiseProject_StandaloneBug_FeatureLinked_Excluded`) | T2 |
| AC-005 | Unit test (`TestSynthesiseProject_StandaloneBug_Closed_Excluded`) | T2 |
| AC-006 | Unit test (`TestSynthesiseProject_StandaloneBug_ResolvedStatuses_Excluded`) | T2 |
| AC-007 | Unit test (`TestSynthesiseProject_StandaloneBug_MediumSeverity_Excluded`) | T2 |
| AC-008 | Unit test (`TestSynthesiseProject_StandaloneBug_LowSeverity_Excluded`) | T2 |
| AC-009 | Unit test (`TestSynthesiseProject_StandaloneBug_OrderingPreserved`) | T2 |
| AC-010 | Code inspection: block not present in `synthesisePlan` | T1 |
| AC-011 | Code inspection: block not present in `synthesiseFeature` | T1 |
| AC-012 | Unit test (`TestSynthesiseProject_StandaloneBug_ListError_Ignored`) | T2 |