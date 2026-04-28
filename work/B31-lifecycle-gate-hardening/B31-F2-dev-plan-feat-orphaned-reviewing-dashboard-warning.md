| Field  | Value                                                             |
|--------|-------------------------------------------------------------------|
| Date   | 2026-04-24                                                        |
| Status | Draft                                                             |
| Author | architect                                                         |

# Dev Plan: Orphaned Reviewing Feature Dashboard Warning

## Scope

This plan implements the requirements defined in
`work/specs/spec-feat-orphaned-reviewing-dashboard-warning.md` for feature
FEAT-01KPXGW5BCGY4. It covers all tasks required to deliver the
`OrphanedReviewingFeatureCheck` attention item in the status dashboard tool,
scoped to project-level (REQ-001), plan-level (REQ-002), and feature-level
(REQ-003) `status()` calls.

It does not cover:
- The non-bypassable merge gate for missing review reports (FEAT-01KPXGVQY3KQC),
  which is governed by a separate specification.
- Changes to how features transition into or out of `reviewing` status.
- Any modifications to `AttentionItem` struct fields or severity levels.

All implementation targets the file `internal/mcp/status_tool.go` and its
companion test file `internal/mcp/status_tool_test.go`. No new packages or
public interfaces are introduced; the change is additive within the existing
attention-generation pattern.

---

## Task Breakdown

### Task 1: Add `reviewingCandidate` type and `generateOrphanedReviewingAttention` helper

- **Description:** Introduce a `reviewingCandidate` struct (`{ ID, Slug,
  DisplayID string }`) and a new function
  `generateOrphanedReviewingAttention(candidates []reviewingCandidate,
  docSvc *service.DocumentService) []AttentionItem` in `status_tool.go`,
  adjacent to the existing `generateProjectAttention` /
  `generatePlanAttention` / `generateFeatureAttention` generators.

  The function must:
  - Return nil immediately if `docSvc == nil` (REQ-007).
  - Return nil immediately if `len(candidates) == 0` (REQ-006, REQ-NF-002).
  - For each candidate call
    `docSvc.ListDocuments(service.DocumentFilters{Owner: c.ID, Type: "report"})`.
  - On error from `ListDocuments`: skip the candidate silently (REQ-007).
  - If `ListDocuments` returns zero documents: append an `AttentionItem` with
    `Type: "orphaned_reviewing_feature"`, `Severity: "warning"`,
    `EntityID: c.ID`, `DisplayID: c.DisplayID`, and
    `Message: fmt.Sprintf("Feature %s (%s) is in 'reviewing' status with no registered review report", c.DisplayID, c.Slug)` (REQ-005).
  - If `ListDocuments` returns one or more documents: skip (REQ-008).

  This function is purpose-built for the project scope (where per-feature docs
  are not pre-fetched). Plan and feature scopes will reuse pre-fetched docs
  directly to satisfy REQ-NF-001 (no more than one List() call per reviewing
  feature per `status()` invocation). The `reviewingCandidate` type and the
  exact `AttentionItem` field values defined here serve as the canonical format
  for all three scopes.

- **Deliverable:** `reviewingCandidate` struct and
  `generateOrphanedReviewingAttention` function added to
  `internal/mcp/status_tool.go`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-004, REQ-005, REQ-006, REQ-007, REQ-008,
  REQ-NF-001, REQ-NF-002.

---

### Task 2: Wire orphaned reviewing check into `synthesiseProject` (project scope)

- **Description:** In `synthesiseProject`, after `allFeatures` is collected,
  build a `[]reviewingCandidate` slice by iterating `allFeatures` and filtering
  for features where `State["status"] == "reviewing"`. Each candidate is
  constructed as `{ ID: f.ID, Slug: f.Slug, DisplayID: id.FormatFullDisplay(f.ID) }`.

  Call `generateOrphanedReviewingAttention(candidates, docSvc)` and append the
  returned items to the `attention` slice (after `generateProjectAttention` and
  health/bug items, consistent with the existing pattern of appending
  supplemental checks inline).

  The `docSvc == nil` and `len(candidates) == 0` guards are already handled
  inside the helper (Task 1), so no additional nil-check is needed at the call
  site.

- **Deliverable:** Modified `synthesiseProject` in `internal/mcp/status_tool.go`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-001, REQ-006.

---

### Task 3: Wire orphaned reviewing check into `synthesisePlan` (plan scope)

- **Description:** In `synthesisePlan`, the `docsPerFeature` map is already
  built (one `docSvc.ListDocumentsByOwner` call per plan feature). Reuse this
  pre-fetched data to avoid additional `docSvc` calls (REQ-NF-001).

  After the `featureSummaries` loop (or within it), and guarded by
  `docSvc != nil`, iterate the features belonging to this plan. For each with
  `fstatus == "reviewing"`:
  - Check whether `docsPerFeature[f.ID]` contains any entry with `Type == "report"`.
  - If none found: append an `AttentionItem` using the same field values and
    message format established in Task 1.

  Append the resulting items to the `attention` slice after the
  `generatePlanAttention` call, consistent with the supplemental-check pattern
  already present in `synthesiseProject`.

- **Deliverable:** Modified `synthesisePlan` in `internal/mcp/status_tool.go`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-002, REQ-007, REQ-008, REQ-NF-001.

---

### Task 4: Wire orphaned reviewing check into `synthesiseFeature` (feature scope)

- **Description:** In `synthesiseFeature`, after the `docs []docInfo` slice is
  built via `docSvc.ListDocumentsByOwner`, add the orphaned reviewing check
  inline — guarded by `docSvc != nil && fstatus == "reviewing"`.

  Check whether any entry in `docs` has `Type == "report"`. If none found,
  construct an `AttentionItem` with the same field values and message format
  established in Task 1 and append it to the `attention` slice (after the
  `generateFeatureAttention` call, before the `fblockedReason` prepend).

  Using `docs` (already fetched via `ListDocumentsByOwner`) means zero
  additional `docSvc` calls at feature scope, satisfying REQ-NF-001.

  The `docSvc == nil` guard distinguishes "service unavailable" (REQ-007:
  skip silently) from "service available but returned no report docs"
  (REQ-005: emit warning).

- **Deliverable:** Modified `synthesiseFeature` in `internal/mcp/status_tool.go`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-003, REQ-007, REQ-008, REQ-NF-001.

---

### Task 5: Unit tests for all acceptance criteria

- **Description:** Add test coverage for AC-001 through AC-008 to
  `internal/mcp/status_tool_test.go`.

  Tests should use the existing `setupStatusTest` helper (real `entitySvc` and
  `docSvc` backed by temp dirs) and the `createStatusTestFeature` /
  `createTestPlan` helpers already present in the test file. Feature status
  transitions should be applied directly via `entitySvc` storage writes (the
  pattern already used in existing status tests).

  Required test cases:
  - **AC-001**: Project scope — one `reviewing` feature, no report doc →
    exactly one `warning` `AttentionItem` with the correct message pattern.
  - **AC-002**: Plan scope — one `reviewing` feature in the plan, no report doc
    → `warning` `AttentionItem` in plan dashboard.
  - **AC-003**: Feature scope — `reviewing` feature, no report doc →
    `warning` `AttentionItem` in feature detail.
  - **AC-004**: `reviewing` feature with at least one `report` doc (any status)
    → no `orphaned_reviewing_feature` `AttentionItem` emitted.
  - **AC-005**: No `reviewing` features in scope → no `orphaned_reviewing_feature`
    items, and `docSvc.ListDocuments` is not called for any feature (verify by
    running with a nil `docSvc` and confirming no panic/error).
  - **AC-006**: `docSvc` unavailable (pass `nil` docSvc to the synthesise
    function) → `status()` completes successfully, no orphan warning emitted.
  - **AC-007**: Multiple `reviewing` features with no report docs → one
    `warning` `AttentionItem` per orphaned feature.
  - **AC-008**: N `reviewing` features → inspected via code review that the
    loop makes exactly one `ListDocuments` call per candidate (covered by
    inspection note in the verification approach below; the unit tests for
    AC-001 and AC-007 provide indirect confirmation via correct item counts).

- **Deliverable:** New test functions in `internal/mcp/status_tool_test.go`
  under a `// ─── OrphanedReviewingFeatureCheck tests ───` heading.
- **Depends on:** Task 1, Task 2, Task 3, Task 4.
- **Effort:** Medium.
- **Spec requirement:** AC-001 through AC-008.

---

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 1
Task 4 → depends on Task 1
Task 5 → depends on Task 1, Task 2, Task 3, Task 4

Parallel groups: [Task 2, Task 3, Task 4]
Critical path:   Task 1 → Task 5
                 (Tasks 2, 3, 4 in parallel; all must complete before Task 5)
```

Task 1 is the only blocking prerequisite for the parallel trio (Tasks 2–4).
Tasks 2, 3, and 4 touch different synthesise functions and have no shared
write surface — they can be dispatched simultaneously once Task 1 is complete.
Task 5 tests all three wire-up points and must run last.

---

## Risk Assessment

### Risk: Redundant `docSvc` calls at plan or feature scope violating REQ-NF-001

- **Probability:** Medium — easy to accidentally call `generateOrphanedReviewingAttention`
  (which makes new docSvc calls) from `synthesisePlan` or `synthesiseFeature`
  instead of reusing pre-fetched docs.
- **Impact:** High — violates a non-functional requirement and silently doubles
  doc-store reads for every reviewing feature in plan/feature-scope views.
- **Mitigation:** Tasks 3 and 4 explicitly specify reuse of pre-fetched data
  (`docsPerFeature` and `docs` respectively). Code review for Task 5 should
  confirm no new `ListDocuments` calls are made at plan/feature scope by
  tracing the call path from `synthesisePlan` and `synthesiseFeature`.
- **Affected tasks:** Task 3, Task 4.

### Risk: `docSvc == nil` not guarded at feature scope, causing silent false positive

- **Probability:** Low — the guard is explicit in Task 4's description, but
  easy to omit if implementing quickly.
- **Impact:** Medium — without the nil guard, a feature with `reviewing` status
  and an empty `docs` slice (because `docSvc` was nil) would emit a spurious
  warning, violating REQ-007.
- **Mitigation:** Task 4 description makes the `docSvc != nil &&
  fstatus == "reviewing"` compound guard explicit. AC-006 test (nil docSvc)
  provides automated regression coverage.
- **Affected tasks:** Task 4, Task 5.

### Risk: Message format divergence across scopes

- **Probability:** Low — Tasks 3 and 4 reference Task 1's canonical format,
  but copying the format string by hand in two places risks drift.
- **Impact:** Low — the spec's message pattern (REQ-005) is tested per-scope
  in Task 5, so drift would be caught by tests.
- **Mitigation:** Consider extracting the message string into a named constant
  or using the `generateOrphanedReviewingAttention` helper for all three scopes
  if a future refactor reduces the call-count concern (e.g. via a
  pre-fetched-docs overload). For now, tests in Task 5 assert the exact message
  pattern for each scope independently.
- **Affected tasks:** Task 3, Task 4, Task 5.

### Risk: `maxAttentionItems` cap suppressing orphaned reviewing items

- **Probability:** Low — the existing `generateProjectAttention` and
  `generatePlanAttention` generators respect `maxAttentionItems = 5`, but the
  new check appends items after those generators return.
- **Impact:** Medium — if the existing generators already emit 5 items, the
  orphaned reviewing warnings would be silently dropped without a visible
  signal.
- **Mitigation:** Because `generateOrphanedReviewingAttention` appends after
  `generateProjectAttention` (following the same pattern as health errors and
  standalone bugs, which also bypass the cap), this is the intended behaviour.
  No cap is applied to the supplemental appends. Document this clearly in the
  implementation to prevent future refactors from incorrectly applying the cap.
- **Affected tasks:** Task 2, Task 3.

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---|---|---|
| AC-001: Project scope, one reviewing feature, no report → one `warning` item with correct message | Unit test (`synthesiseProject` integration via temp store) | Task 5 |
| AC-002: Plan scope, one reviewing feature, no report → `warning` item in plan dashboard | Unit test (`synthesisePlan` integration via temp store) | Task 5 |
| AC-003: Feature scope, reviewing feature, no report → `warning` item in feature detail | Unit test (`synthesiseFeature` integration via temp store) | Task 5 |
| AC-004: Reviewing feature with at least one report doc → no `orphaned_reviewing_feature` item | Unit test (register report doc, assert no item emitted) | Task 5 |
| AC-005: No reviewing features → `docSvc.ListDocuments` not called, no warning | Unit test (nil docSvc, non-reviewing features only, no panic) | Task 5 |
| AC-006: `docSvc` nil/unavailable → `status()` succeeds, no orphan warning | Unit test (nil docSvc, reviewing feature present, assert success + no item) | Task 5 |
| AC-007: Multiple reviewing features with no reports → one item per feature | Unit test (N=3 reviewing features, assert N warning items) | Task 5 |
| AC-008: N reviewing features → exactly N `ListDocuments` calls | Code inspection (review loop in `generateOrphanedReviewingAttention`; AC-001 and AC-007 tests provide indirect confirmation) | Task 1, Task 5 |
```

Now let me register and auto-approve the document: