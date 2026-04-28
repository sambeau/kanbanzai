| Field  | Value                                                        |
|--------|--------------------------------------------------------------|
| Date   | 2026-04-23                                                   |
| Status | Draft                                                        |
| Author | spec-author                                                  |

# Specification: Orphaned Reviewing Feature Dashboard Warning

## Problem Statement

This specification covers the `OrphanedReviewingFeatureCheck` attention item added to the
status dashboard tool, as defined in Component 2 of the parent design document:
`work/design/p31-lifecycle-gate-hardening.md`.

A feature in `reviewing` status that has no registered review report document is an
**orphaned reviewing feature** — it is either mid-review (transient, benign) or has had
its review step silently bypassed (a workflow integrity problem). Without a visible signal,
orphaned reviewing features accumulate undetected until a merge attempt triggers the hard
gate check (FEAT-01KPXGVQY3KQC). This specification describes the soft, dashboard-level
signal that makes orphaned reviewing features visible during normal operation, before any
merge is attempted.

The design decision governing severity is Decision 3 from the parent design: the check
emits `severity: warning` (not `error`) because the condition may be transient (a review
in progress) and a warning is appropriate for a state that may or may not represent a
problem.

**In scope:**
- Adding `OrphanedReviewingFeatureCheck` to the attention item assembler.
- Running the check on project-level, plan-scoped, and feature-scoped `status()` calls.
- Emitting a `warning`-severity `AttentionItem` for each `reviewing` feature with no
  registered `report` document.
- Skipping the check silently when there are zero `reviewing` features or when
  `docRecordSvc` is unavailable.

**Explicitly out of scope:**
- The non-bypassable merge gate for missing review reports (FEAT-01KPXGVQY3KQC).
- Retroactive blocking or remediation of already-merged features left in `reviewing` status.
- Escalating severity based on how long a feature has been in `reviewing` status (noted
  as a future enhancement in the parent design, Decision 3).

---

## Requirements

### Functional Requirements

- **REQ-001:** The status tool's attention item assembler MUST include a check named
  `OrphanedReviewingFeatureCheck` that runs on every project-level `status()` call.

- **REQ-002:** `OrphanedReviewingFeatureCheck` MUST also run on plan-scoped
  `status(id: P-xxx)` calls, scoped to the features belonging to that plan.

- **REQ-003:** `OrphanedReviewingFeatureCheck` MUST also run on feature-scoped
  `status(id: FEAT-xxx)` calls, evaluated for that single feature only.

- **REQ-004:** For each feature with `status = reviewing`, the check MUST call
  `docRecordSvc.List()` filtered by `owner = feature ID` and `type = report`.

- **REQ-005:** If `docRecordSvc.List()` returns no documents for a `reviewing` feature,
  the check MUST emit an `AttentionItem` with:
  - `severity: warning`
  - `entity_id` set to the feature's ID
  - `message` matching the pattern:
    `"Feature FEAT-xxx (slug) is in 'reviewing' status with no registered review report"`
    where `FEAT-xxx` is the feature's ID and `slug` is the feature's slug.

- **REQ-006:** The check MUST be skipped entirely (no `docRecordSvc` calls made) when
  there are zero features in `reviewing` status in the applicable scope.

- **REQ-007:** If `docRecordSvc` is unavailable (e.g. returns an error or is nil), the
  check MUST be skipped silently — no `AttentionItem` is emitted and no error is surfaced
  to the caller.

- **REQ-008:** The check MUST NOT emit an `AttentionItem` for a `reviewing` feature that
  has at least one registered `report` document (regardless of that document's approval
  status).

### Non-Functional Requirements

- **REQ-NF-001:** The check MUST NOT make more than one `docRecordSvc.List()` call per
  `reviewing` feature per `status()` invocation.

- **REQ-NF-002:** The check MUST add no observable latency to `status()` calls when there
  are zero features in `reviewing` status.

---

## Constraints

- The `AttentionItem` struct and severity levels (`warning`, `info`, `error`) MUST NOT be
  changed. The check uses the existing `severity: warning` value.

- The attention item assembler's existing interface and invocation pattern MUST NOT change.
  `OrphanedReviewingFeatureCheck` is an additive check registered alongside existing checks.

- The check uses `docRecordSvc.List()` with `owner` and `type` filters only — it MUST NOT
  inspect document content or approval status.

- The check MUST be fail-open: an error from `docRecordSvc` suppresses the check but does
  not surface an error to the `status()` caller (consistent with the design's failure mode
  table and Decision 2's rationale applied to the dashboard context).

- This specification does NOT cover the merge gate (FEAT-01KPXGVQY3KQC), the
  `ReviewReportExistsGate`, or the `Bypassable bool` field on `gate.Result`. Those are
  governed by a separate specification.

- This specification does NOT cover any changes to how features transition into or out of
  `reviewing` status.

---

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-004, REQ-005):** Given a project with one feature in `reviewing`
  status and no registered `report` document for that feature, when `status()` is called
  at the project level, then the response includes exactly one `AttentionItem` with
  `severity: warning` and a message matching
  `"Feature FEAT-xxx (slug) is in 'reviewing' status with no registered review report"`.

- **AC-002 (REQ-002, REQ-004, REQ-005):** Given a plan with one feature in `reviewing`
  status and no registered `report` document for that feature, when `status(id: P-xxx)` is
  called for that plan, then the response includes a `warning` `AttentionItem` for that
  feature.

- **AC-003 (REQ-003, REQ-004, REQ-005):** Given a feature with `status = reviewing` and
  no registered `report` document, when `status(id: FEAT-xxx)` is called for that feature,
  then the response includes a `warning` `AttentionItem` for that feature.

- **AC-004 (REQ-008):** Given a feature with `status = reviewing` that has at least one
  registered `report` document (in any approval status), when `status()` is called, then
  no `AttentionItem` is emitted for that feature.

- **AC-005 (REQ-006):** Given a project with no features in `reviewing` status, when
  `status()` is called, then `docRecordSvc.List()` is not called and no warning
  `AttentionItem` related to orphaned reviewing is emitted.

- **AC-006 (REQ-007):** Given `docRecordSvc` is unavailable (returns an error or is nil),
  when `status()` is called (with one or more `reviewing` features), then the `status()`
  call completes successfully, no `AttentionItem` for an orphaned reviewing feature is
  emitted, and no error is returned to the caller.

- **AC-007 (REQ-001, REQ-005):** Given a project with multiple features in `reviewing`
  status, none of which have a registered `report` document, when `status()` is called,
  then exactly one `warning` `AttentionItem` is emitted per orphaned feature.

- **AC-008 (REQ-NF-001):** Given a project with N features in `reviewing` status when
  `status()` is called, then `docRecordSvc.List()` is called exactly N times.

---

## Verification Plan

| Criterion | Method      | Description                                                                                          |
|-----------|-------------|------------------------------------------------------------------------------------------------------|
| AC-001    | Test        | Unit test: single `reviewing` feature, no report → one `warning` AttentionItem with correct message  |
| AC-002    | Test        | Unit test: plan-scoped `status()` surfaces warning for orphaned `reviewing` feature in that plan     |
| AC-003    | Test        | Unit test: feature-scoped `status()` surfaces warning for the single orphaned `reviewing` feature    |
| AC-004    | Test        | Unit test: `reviewing` feature with an existing `report` doc → no `AttentionItem` emitted            |
| AC-005    | Test        | Unit test: no `reviewing` features → `docRecordSvc.List()` not called, no warning emitted            |
| AC-006    | Test        | Unit test: `docRecordSvc` returns error → `status()` returns no error and no orphan warning emitted  |
| AC-007    | Test        | Unit test: multiple orphaned `reviewing` features → one `warning` AttentionItem per feature          |
| AC-008    | Inspection  | Code review confirms `docRecordSvc.List()` is called in a loop over `reviewing` features with no batching shortcuts that could skip features or call multiple times per feature |