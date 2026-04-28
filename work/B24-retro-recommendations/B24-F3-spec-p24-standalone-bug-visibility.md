# Specification: Standalone bugs visible in status health

| Field       | Value                                                         |
|-------------|---------------------------------------------------------------|
| Feature     | FEAT-01KPPG3MSRRCE — Standalone bugs visible in status health |
| Design doc  | `work/design/p24-standalone-bug-visibility.md`                |
| Plan        | P24-retro-recommendations                                     |
| Status      | Draft                                                         |

---

## Overview

This specification covers a targeted extension to `synthesiseProject` in
`internal/mcp/status_tool.go` that surfaces open standalone bugs (those with no
`origin_feature` linkage) in the project-level `status` attention items.

P19 wired attention items for open high/critical bugs into the `status` tool, but
only for bugs linked to a specific in-flight feature via `origin_feature`. A bug
filed against general code with `origin_feature` absent or empty is invisible in
every `status` call. This spec closes that gap.

---

## Scope

**In scope:**
- Appending `AttentionItem` entries for qualifying standalone bugs to the
  `attention` slice returned by `synthesiseProject` (project-level `status()`
  with no ID argument).
- Defining the filter conditions, attention item shape, error-handling posture,
  and double-surfacing guard for the new standalone-bug block.

**Explicitly out of scope:**
- Surfacing standalone bugs at plan scope or feature scope.
- Surfacing standalone bugs with severity `medium` or `low`.
- Changing how feature-linked bugs are surfaced in `synthesiseFeature`.
- Adding or enforcing an `origin_feature` field validation mechanism.
- Introducing a new `AttentionItem.Type` constant for standalone bugs.

---

## Problem Statement

This specification implements the design described in
`work/design/p24-standalone-bug-visibility.md`.

The `status` tool (project scope) already surfaces attention items for open
high- and critical-severity bugs that are linked to an in-flight feature via
`origin_feature`. Bugs with no feature attachment (`origin_feature == ""`) are
permanently invisible in every `status` scope. This is a coverage gap identified
in P19 (source: REC-04).

The fix is a new, additive block inside `synthesiseProject` that iterates all
bugs after `generateProjectAttention` and the health-check block complete, and
appends an `AttentionItem` for each bug that is open, standalone, and
high/critical severity.

---

## Requirements

The requirements in this section are derived from design sections §2, §3, and §5
of `work/design/p24-standalone-bug-visibility.md`.

---

## Functional Requirements

- **REQ-001:** When `synthesiseProject` is called, it MUST query all bugs via
  `entitySvc.List("bug")` and append an `AttentionItem` to the project-level
  `attention` slice for each bug that satisfies **all three** of the following
  conditions simultaneously:
  1. `origin_feature` is absent or equal to `""` (empty string).
  2. `status` is not one of: `done`, `closed`, `not-planned`, `duplicate`,
     `wont-fix`.
  3. `severity` is `"high"` or `"critical"`.

- **REQ-002:** Each `AttentionItem` produced for a qualifying standalone bug
  MUST have the following field values:
  - `Type`: `"open_critical_bug"`
  - `Severity`: `"warning"`
  - `EntityID`: the bug's ID string
  - `DisplayID`: the result of `id.FormatFullDisplay(<bug ID>)`
  - `Message`: `"Standalone <severity> bug: <name>"` where `<severity>` is the
    bug's severity value and `<name>` is the bug's name. If the name field is
    absent or empty, `<name>` MUST be replaced by the bug's ID string, yielding
    `"Standalone <severity> bug: <bug ID>"`.

- **REQ-003:** The standalone-bug block MUST execute after
  `generateProjectAttention` returns **and** after the health-check attention
  items are appended, so that standalone-bug items appear last in the `attention`
  slice. The relative order and content of all pre-existing attention items MUST
  NOT change.

- **REQ-004:** A bug with a non-empty `origin_feature` value MUST NOT appear in
  the project-level `attention` slice as a result of this change. Feature-linked
  bugs are handled exclusively by `synthesiseFeature`; the two populations are
  disjoint by construction (`origin_feature == ""` vs `origin_feature != ""`).

- **REQ-005:** A standalone bug that has been transitioned to any resolved
  status (`done`, `closed`, `not-planned`, `duplicate`, or `wont-fix`) MUST NOT
  appear in the project-level `attention` slice.

- **REQ-006:** Standalone bugs with `severity` equal to `"medium"` or `"low"`
  MUST NOT appear in the project-level `attention` slice.

- **REQ-007:** Standalone bugs MUST NOT appear in plan-scoped or
  feature-scoped `status` responses. The standalone-bug block is implemented
  exclusively inside `synthesiseProject`.

---

## Non-Functional Requirements

- **REQ-NF-001:** If `entitySvc.List("bug")` returns an error,
  `synthesiseProject` MUST still return a valid `projectOverview` response.
  The standalone-bug block is best-effort: a bug-listing failure MUST NOT cause
  the `status` tool call to return an error.

- **REQ-NF-002:** The `AttentionItem` struct definition MUST NOT be modified.
  The type constant `"open_critical_bug"` is reused from the existing definition;
  no new struct fields or type constants are introduced by this change.

- **REQ-NF-003:** The change is purely additive. No existing `attention` items
  produced by `generateProjectAttention` or the health-check block may be
  removed, reordered, or altered by this change.

---

## Constraints

- The resolved-status skip list (`done`, `closed`, `not-planned`, `duplicate`,
  `wont-fix`) MUST match the skip list used in `synthesiseFeature` for
  feature-linked bugs (as established in the existing codebase at REQ-026).
- The severity threshold (`"high"` / `"critical"`) MUST match the existing
  threshold used for feature-linked bug warnings (REQ-025 in the existing
  codebase).
- The `AttentionItem.Type` value `"open_critical_bug"` is reused unchanged so
  that consumers already handling this type for feature-linked bugs will handle
  standalone bugs without a schema change.
- This specification does NOT cover changes to `generateProjectAttention`,
  `generateFeatureAttention`, or `synthesisePlan`.

---

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a standalone bug with `severity: "high"`,
  `origin_feature` absent, and `status: "reported"`, when `synthesiseProject` is
  called, then the returned `attention` slice contains an item with
  `Type == "open_critical_bug"`, `Severity == "warning"`,
  `EntityID == <bug ID>`, `DisplayID == id.FormatFullDisplay(<bug ID>)`, and
  `Message == "Standalone high bug: <bug name>"`.

- **AC-002 (REQ-001, REQ-002):** Given a standalone bug with
  `severity: "critical"` and `origin_feature == ""`, when `synthesiseProject`
  is called, then the `attention` slice contains an item with
  `Type == "open_critical_bug"` and `Message == "Standalone critical bug: <bug name>"`.

- **AC-003 (REQ-002):** Given a standalone bug whose `name` field is empty or
  absent, when `synthesiseProject` is called, then the `Message` of the
  corresponding `AttentionItem` is `"Standalone <severity> bug: <bug ID>"`.

- **AC-004 (REQ-004):** Given a bug with `severity: "high"` and a non-empty
  `origin_feature` referencing a real feature ID, when `synthesiseProject` is
  called, then the `attention` slice does NOT contain an item with
  `EntityID == <that bug's ID>`.

- **AC-005 (REQ-005):** Given a standalone bug with `severity: "high"` that has
  been transitioned to `closed`, when `synthesiseProject` is called, then the
  `attention` slice does NOT contain an item for that bug.

- **AC-006 (REQ-005):** Given standalone bugs each in one of the states `done`,
  `not-planned`, `duplicate`, and `wont-fix`, when `synthesiseProject` is
  called, then none of those bugs appear in the `attention` slice.

- **AC-007 (REQ-006):** Given a standalone bug with `severity: "medium"`, when
  `synthesiseProject` is called, then the `attention` slice does NOT contain an
  item for that bug.

- **AC-008 (REQ-006):** Given a standalone bug with `severity: "low"`, when
  `synthesiseProject` is called, then the `attention` slice does NOT contain an
  item for that bug.

- **AC-009 (REQ-003):** Given pre-existing attention items from
  `generateProjectAttention` and the health-check block, when `synthesiseProject`
  is called with one or more qualifying standalone bugs, then those pre-existing
  items appear before all standalone-bug items in the `attention` slice, and
  their content is unchanged.

- **AC-010 (REQ-007):** Given a standalone bug with `severity: "critical"`, a
  plan-scoped `status(id: <plan ID>)` call does NOT include an item for that bug
  in the plan-level `attention` slice.

- **AC-011 (REQ-007):** Given a standalone bug with `severity: "critical"`, a
  feature-scoped `status(id: <feature ID>)` call does NOT include an item for
  that bug in the feature-level `attention` slice.

- **AC-012 (REQ-NF-001):** Given that `entitySvc.List("bug")` returns an error,
  when `synthesiseProject` is called, then the call succeeds and returns a valid
  `projectOverview` response with no standalone-bug attention items and no error.

---

## Verification Plan

| Criterion | Method     | Description                                                                                                |
|-----------|------------|------------------------------------------------------------------------------------------------------------|
| AC-001    | Test       | Unit test: standalone high-severity bug in `reported` state; assert item present with all correct fields   |
| AC-002    | Test       | Unit test: standalone critical-severity bug with empty `origin_feature`; assert item present               |
| AC-003    | Test       | Unit test: standalone bug with empty name; assert `Message` uses bug ID instead of name                    |
| AC-004    | Test       | Unit test: feature-linked high-severity bug; assert it does NOT appear in project-level attention          |
| AC-005    | Test       | Unit test: standalone high bug transitioned to `closed`; assert item absent                                |
| AC-006    | Test       | Unit test: parameterised over `done`, `not-planned`, `duplicate`, `wont-fix`; assert each absent           |
| AC-007    | Test       | Unit test: standalone medium-severity bug; assert absent from attention                                    |
| AC-008    | Test       | Unit test: standalone low-severity bug; assert absent from attention                                       |
| AC-009    | Test       | Unit test: assert standalone-bug items follow pre-existing items; verify pre-existing items unchanged      |
| AC-010    | Inspection | Code review: confirm standalone-bug block is only inside `synthesiseProject`, not `synthesisePlan`         |
| AC-011    | Inspection | Code review: confirm standalone-bug block is only inside `synthesiseProject`, not `synthesiseFeature`      |
| AC-012    | Test       | Unit test: inject `List("bug")` error via fake; assert `synthesiseProject` returns success with no error   |