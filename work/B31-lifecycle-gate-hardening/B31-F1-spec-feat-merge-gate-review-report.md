| Field  | Value                                                              |
|--------|--------------------------------------------------------------------|
| Date   | 2026-04-23                                                         |
| Status | Draft                                                              |
| Author | spec-author                                                        |

# Specification: Non-bypassable merge gate for review report

## Problem Statement

This specification covers the implementation of a non-bypassable merge gate that blocks
merging a feature in `reviewing` status when no review report document is registered in
the document store.

> This specification implements the design described in
> `work/design/p31-lifecycle-gate-hardening.md`.

The relevant design decisions addressed here are:

- **Decision 1:** The `ReviewReportExistsGate` is not bypassable with `override: true`
  when the feature is in `reviewing` status. Filing a review report is a minimal,
  always-achievable action; the absence of a report is evidence of a skipped step, not a
  legitimate blocker.
- **Decision 2:** If `docRecordSvc.List()` returns an error during gate evaluation, the
  gate returns `Pass` (fail-open) and logs a warning, so that a broken document index
  does not render the merge tool inoperable.
- **Decision 4:** The `gate.Result` type gains a `Bypassable bool` field. Existing gate
  conditions default to `Bypassable: true` (no behaviour change). Only
  `ReviewReportExistsGate` sets `Bypassable: false`.

**In scope:**
- Adding `ReviewReportExistsGate` to the merge gate evaluation chain for `reviewing`-stage
  features.
- Adding `Bypassable bool` to the `gate.Result` type.
- Enforcing that `override: true` is rejected when the blocking gate has `Bypassable: false`.
- Fail-open behaviour when `docRecordSvc.List()` returns an error.
- Actionable error message returned to the caller when the gate fires.

**Explicitly out of scope:**
- The status dashboard orphaned-reviewing warning (`OrphanedReviewingFeatureCheck`) —
  that is covered by FEAT-01KPXGW5BCGY4.
- Changes to any other gate condition or its override behaviour.
- Requiring the report document to be in `approved` status (a `draft` report is
  sufficient to satisfy the gate).

---

## Requirements

### Functional Requirements

- **REQ-001:** The `gate.Result` type MUST include a `Bypassable bool` field. When this
  field is `false`, the merge tool MUST reject an `override: true` request and return an
  error without proceeding with the merge.

- **REQ-002:** All existing gate conditions MUST default to `Bypassable: true`, preserving
  their current override behaviour without any code change to those gates.

- **REQ-003:** A new gate condition, `ReviewReportExistsGate`, MUST be added to the gate
  evaluation chain that runs when the feature's lifecycle status is `reviewing`.

- **REQ-004:** `ReviewReportExistsGate` MUST call `docRecordSvc.List()` filtered by
  `owner = featureID` and `type = report`. If the result set contains at least one
  document (any status), the gate MUST return `Pass`.

- **REQ-005:** `ReviewReportExistsGate` MUST return `Blocked` with `Bypassable: false`
  when `docRecordSvc.List()` returns an empty result set.

- **REQ-006:** If `docRecordSvc.List()` returns an error, `ReviewReportExistsGate` MUST
  return `Pass` (fail-open) and MUST emit a log warning describing the error.

- **REQ-007:** When `ReviewReportExistsGate` fires (returns `Blocked`), the merge tool
  MUST return the following actionable error message to the caller (substituting the
  actual feature ID for `FEAT-xxx`):

  ```
  Cannot merge FEAT-xxx: feature is in 'reviewing' status but no review report is
  registered.

  To resolve:
    1. Run the review: handoff(task_id: ...) with role: reviewer-conformance (and
       other reviewer roles per the developing stage binding).
    2. Register the report: doc(action: register, type: report, owner: FEAT-xxx, ...).
    3. Retry merge(action: execute, entity_id: FEAT-xxx).

  If the feature was reviewed but the report was not registered, register it now.
  This gate cannot be bypassed with override: true — a report must exist.
  ```

- **REQ-008:** The gate MUST NOT fire for features whose lifecycle status is anything
  other than `reviewing`.

### Non-Functional Requirements

- **REQ-NF-001:** The additional `docRecordSvc.List()` call introduced by
  `ReviewReportExistsGate` MUST add no more than one synchronous I/O call to the merge
  gate evaluation path for `reviewing`-stage features under normal operating conditions.

- **REQ-NF-002:** Existing merge gate behaviour for features NOT in `reviewing` status
  MUST be unchanged — no additional latency, no additional I/O calls.

---

## Constraints

- The `Bypassable bool` field MUST be backward-compatible: all existing gate conditions
  that do not explicitly set `Bypassable: false` MUST continue to function identically to
  their current behaviour (i.e., they remain bypassable with `override: true`).
- The `gate.Result` type interface change MUST NOT require updates to existing gate
  condition implementations beyond their default zero-value for `Bypassable` (which, as
  `false`, would break existing gates). The default value for `Bypassable` MUST be `true`
  for all existing gates, which means the field is opt-in for non-bypassability — existing
  gates MAY need to be updated to explicitly set `Bypassable: true`, or an alternative
  constructor / default mechanism MUST ensure backward compatibility.
- The merge tool's existing override audit-log behaviour MUST remain intact for gates
  where `Bypassable: true`.
- This specification does NOT cover the `OrphanedReviewingFeatureCheck` in the status
  dashboard.
- This specification does NOT require the review report to be in `approved` status; a
  `draft` report satisfies the gate.
- The gate must not break the merge tool when the document service is unavailable
  (fail-open is mandatory, per Decision 2).

---

## Acceptance Criteria

- **AC-001 (REQ-003, REQ-004, REQ-005):** Given a feature in `reviewing` status with no
  registered `report` document, when `merge(action: execute, entity_id: FEAT-xxx)` is
  called (with or without `override: true`), then the merge is blocked and the actionable
  error message (REQ-007) is returned.

- **AC-002 (REQ-001, REQ-005):** Given a feature in `reviewing` status with no registered
  `report` document, when `merge(action: execute, entity_id: FEAT-xxx, override: true)`
  is called, then the override is rejected (not accepted), and the error message states
  that this gate cannot be bypassed with `override: true`.

- **AC-003 (REQ-003, REQ-004):** Given a feature in `reviewing` status with at least one
  registered `report` document (status: `draft` or `approved`), when
  `merge(action: execute, entity_id: FEAT-xxx)` is called (all other gates passing), then
  `ReviewReportExistsGate` passes and does not block the merge.

- **AC-004 (REQ-006):** Given a feature in `reviewing` status and `docRecordSvc.List()`
  returning an error, when `merge(action: execute, entity_id: FEAT-xxx)` is called, then
  `ReviewReportExistsGate` returns `Pass`, a log warning is emitted, and the merge is not
  blocked by this gate.

- **AC-005 (REQ-008):** Given a feature NOT in `reviewing` status (e.g., `developing`),
  when `merge(action: execute, entity_id: FEAT-xxx)` is called, then
  `ReviewReportExistsGate` does not execute and no report existence check is performed.

- **AC-006 (REQ-002):** Given an existing gate condition that currently allows
  `override: true`, after the `Bypassable bool` field is added to `gate.Result`, when
  `merge(action: execute, entity_id: FEAT-xxx, override: true)` is called for a feature
  blocked by that existing gate, then the override is still accepted (existing behaviour
  is preserved).

- **AC-007 (REQ-007):** Given a feature in `reviewing` status with no registered `report`
  document, when `merge(action: execute, entity_id: FEAT-xxx)` is called, then the error
  message returned includes the feature ID, the three-step resolution instructions, and
  the explicit statement that `override: true` cannot bypass this gate.

---

## Verification Plan

| Criterion | Method      | Description                                                                                                  |
|-----------|-------------|--------------------------------------------------------------------------------------------------------------|
| AC-001    | Test        | Automated unit test: `ReviewReportExistsGate` returns `Blocked{Bypassable: false}` when doc list is empty.  |
| AC-002    | Test        | Automated integration test: merge tool rejects `override: true` when the blocking gate has `Bypassable: false`. |
| AC-003    | Test        | Automated unit test: gate returns `Pass` when `docRecordSvc.List()` returns ≥1 report (draft and approved variants). |
| AC-004    | Test        | Automated unit test: gate returns `Pass` and emits log warning when `docRecordSvc.List()` returns an error.  |
| AC-005    | Test        | Automated unit test: `ReviewReportExistsGate` is not invoked (or is skipped) for features not in `reviewing` status. |
| AC-006    | Test        | Automated regression test: existing gate conditions with `override: true` continue to be accepted after `Bypassable` field introduction. |
| AC-007    | Inspection  | Code review verifying that the error message string returned by the merge tool matches the exact wording specified in REQ-007. |