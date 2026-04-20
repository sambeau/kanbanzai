# Specification: Standalone bugs visible in status health

| Field       | Value                                                        |
|-------------|--------------------------------------------------------------|
| Feature     | FEAT-01KPPG3MSRRCE — Standalone bugs visible in status health |
| Design doc  | `work/design/p24-standalone-bug-visibility.md`               |
| Plan        | P24-retro-recommendations                                    |
| Status      | Draft                                                        |

---

## Problem Statement

This specification implements the design described in
`work/design/p24-standalone-bug-visibility.md`.

The `status` tool (project scope) surfaces attention items for open high- and
critical-severity bugs, but only for bugs linked to a specific in-flight feature
via the `origin_feature` field. Bugs filed against general code with no feature
attachment (`origin_feature == ""`) are invisible in every `status` call — at
project scope, plan scope, and feature scope.

This specification covers the targeted extension to `synthesiseProject` in
`internal/mcp/status_tool.go` that closes this gap by appending attention items
for open standalone bugs of severity `high` or `critical`.

**In scope:**
- Surfacing qualifying standalone bugs in the `attention` slice of the
  project-level `status` response (i.e. `status()` with no ID argument).
- Defining the exact filter conditions, attention item shape, error-handling
  posture, and double-surfacing guard for this new block.

**Explicitly out of scope:**
- Surfacing standalone bugs at plan scope or feature scope.
- Surfacing standalone bugs with severity `medium` or `low`.
- Changing how feature-linked bugs are surfaced in `synthesiseFeature`.
- Adding or enforcing an `origin_feature` field validation mechanism.
- Introducing a new `AttentionItem.Type` value for standalone bugs.

---

## Requirements

### Functional Requirements

- **REQ-001:** When `synthesiseProject` is called, it MUST query all bugs using
  `entitySvc.List("bug")` and append an `AttentionItem` to the project-level
  `attention` slice for each bug that satisfies all three conditions:
  1. `origin_feature` is absent or empty (`""`).
  2. `status` is not one of `done`, `closed`, `not-planned`, `duplicate`,
     or `wont-fix`.
  3. `severity` is `high` or `critical`.

- **REQ-002:** Each `AttentionItem` produced for a qualifying standalone bug
  MUST have the following field values:
  - `Type`: `"open_critical_bug"`
  - `Severity`: `"warning"`
  - `EntityID`: the bug's ID string
  - `DisplayID`: the result of `id.FormatFullDisplay(<bug ID>)`
  - `Message`: `"Standalone <severity> bug: <name>"` where `<severity>` is the
    bug's severity value and `<name>` is the bug's name string. If the name
    field is absent or empty, `<name>` MUST be replaced by the bug's ID string,
    yielding `"Standalone <severity> bug: <bug ID>"`.

- **REQ-003:** The standalone-bug block MUST execute after
  `generateProjectAttention` returns and after the health-check attention items
  are appended, so that standalone-bug items are added last in the `attention`
  slice. The relative order of pre-existing attention items MUST NOT change.

- **REQ-004:** A bug with a non-empty `origin_feature` value MUST NOT appear in
  the project-level `attention` slice as a result of this change. Feature-linked
  bugs are handled exclusively by `synthesiseFeature`; the two populations are
  disjoint by construction.

- **REQ-005:** A qualifying standalone bug that is subsequently transitioned to
  any resolved status (`done`, `closed`, `not-planned`, `duplicate`,
  `wont-fix`) MUST no longer appear in the project-level `attention` slice.

- **REQ-006:** Standalone bugs with severity `medium` or `low` MUST NOT appear
  in the project-level `attention` slice.

- **REQ-007:** Standalone bugs MUST NOT appear in plan-scoped or
  feature-scoped `status` responses.

### Non-Functional Requirements

- **REQ-NF-001:** If `entitySvc.List("bug")` returns an error, `synthesiseProject`
  MUST still return a valid `projectOverview` response with no additional error
  propagation. The standalone-bug block is best-effort; a bug-listing failure
  MUST NOT cause the `status` tool call to fail.

- **REQ-NF-002:** The `AttentionItem` struct definition MUST NOT be modified.
  The `"open_critical_bug"` type value is reused from the existing definition;
  no new type constants or struct fields are introduced.

---

## Constraints

- The change is purely additive. No existing `attention` items produced by
  `generateProjectAttention` or the health-check block may be removed,
  reordered, or altered.
- The `AttentionItem.Type` value `"open_critical_bug"` is reused unchanged.
  Consumers that already handle this type for feature-linked bugs will handle
  standalone bugs without a schema change.
- The resolved-status skip list (`done`, `closed`, `not-planned`, `duplicate`,
  `wont-fix`) MUST match the list used in `synthesiseFeature` for feature-linked
  bugs (established by REQ-026 in the existing codebase).
- The severity threshold (`high` / `critical`) MUST match the existing threshold
  used for feature-linked bug warnings (REQ-025 in the existing codebase).
- This specification does NOT cover changes to `generateProjectAttention`,
  `generateFeatureAttention`, or `synthesisePlan`.

---

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a standalone bug with `severity: high`
  and `origin_feature` absent, and status `reported`, when `synthesiseProject`
  is called, then the returned `attention` slice contains exactly one item with
  `Type == "open_critical_bug"`, `Severity == "warning"`,
  `EntityID == <bug ID>`, and `Message == "Standalone high bug: <bug name>"`.

- **AC-002 (REQ-001, REQ-002):** Given a standalone bug with `severity: critical`
  and an empty `origin_feature` field, when `synthesiseProject` is called, then
  the returned `attention` slice contains an item with `Type == "open_critical_bug"`
  and `Message == "Standalone critical bug: <bug name>"`.

- **AC-003 (REQ-002):** Given a standalone bug whose `name` field is empty or
  absent, when `synthesiseProject` is called, then the `Message` of the
  corresponding `AttentionItem` is `"Standalone <severity> bug: <bug ID>"`.

- **AC-004 (REQ-004):** Given a bug with `severity: high` and a non-empty
  `origin_feature` referencing a real feature ID, when `synthesiseProject` is
  called, then the `attention` slice does NOT contain an item with
  `EntityID == <that bug's ID>`.

- **AC-005 (REQ-005):** Given a standalone bug with `severity: high` that is
  transitioned to `closed`, when `synthesiseProject` is called, then the
  `attention` slice does NOT contain an item for that bug.

- **AC-006 (REQ-005):** Given standalone bugs transitioned to each of `done`,
  `not-planned`, `duplicate`, and `wont-fix`, when `synthesiseProject` is
  called, then none of those bugs appear in the `attention` slice.

- **AC-007 (REQ-006):** Given a standalone bug with `severity: medium`, when
  `synthesiseProject` is called, then the `attention` slice does NOT contain
  an item for that bug.

- **AC-008 (REQ-006):** Given a standalone bug with `severity: low`, when
  `synthesiseProject` is called, then the `attention` slice does NOT contain
  an item for that bug.

- **AC-009 (REQ-003):** Given pre-existing attention items produced by
  `generateProjectAttention` and the health-check block, when `synthesiseProject`
  is called with standalone bugs present, then the pre-existing items appear
  before standalone-bug items in the `attention` slice, and their content is
  unchanged.

- **AC-010 (REQ-007):** Given a standalone bug with `severity: critical`, when
  `synthesiseProject` is called with a plan ID argument (plan scope), then the
  plan-level `attention` slice does NOT contain an item for that standalone bug.

- **AC-011 (REQ-007):** Given a standalone bug with `severity: critical`, when
  `synthesiseProject` is called with a feature ID argument (feature scope), then
  the feature-level `attention` slice does NOT contain an item for that
  standalone bug.

- **AC-012 (REQ-NF-001):** Given that `entitySvc.List("bug")` returns an error,
  when `synthesiseProject` is called, then the call succeeds and returns a valid
  `projectOverview` with no standalone-bug attention items and no error.

---

## Verification Plan

| Criterion | Method      | Description                                                                                              |
|-----------|-------------|----------------------------------------------------------------------------------------------------------|
| AC-001    | Test        | Automated unit test: create standalone high-severity bug; assert item present with correct fields        |
| AC-002    | Test        | Automated unit test: create standalone critical-severity bug; assert item present with correct fields    |
| AC-003    | Test        | Automated unit test: create standalone bug with empty name; assert Message uses bug ID                   |
| AC-004    | Test        | Automated unit test: create feature-linked high-severity bug; assert it does NOT appear at project scope |
| AC-005    | Test        | Automated unit test: close standalone high bug; assert item absent from attention slice                  |
| AC-006    | Test        | Automated unit test: parameterised over `done`, `not-planned`, `duplicate`, `wont-fix`; assert absent   |
| AC-007    | Test        | Automated unit test: create standalone medium-severity bug; assert absent                                |
| AC-008    | Test        | Automated unit test: create standalone low-severity bug; assert absent                                   |
| AC-009    | Test        | Automated unit test: assert standalone-bug items appear after pre-existing items; existing items unchanged|
| AC-010    | Inspection  | Code review: confirm standalone-bug block is inside `synthesiseProject` only, not `synthesisePlan`       |
| AC-011    | Inspection  | Code review: confirm standalone-bug block is inside `synthesiseProject` only, not `synthesiseFeature`    |
| AC-012    | Test        | Automated unit test: inject List("bug") error; assert `synthesiseProject` returns no error               |