# Specification: Non-bypassable Merge Gate for Review Report

**Feature:** FEAT-01KPXGVQY3KQC  
**Status:** Draft  
**Owner:** FEAT-01KPXGVQY3KQC

---

## 1. Related Work

- **Design:** `work/design/p31-lifecycle-gate-hardening.md` — Component 1: ReviewReportExistsGate
- **Feature:** FEAT-01KPXGVQY3KQC — Non-bypassable merge gate for review report

---

## 2. Overview

When a feature is in `reviewing` status, the merge tool must verify that at least one review report document has been registered before the merge is permitted to proceed. This gate cannot be bypassed by passing `override: true` to the merge tool. It enforces that a review report exists as a precondition for merging any feature that has entered the reviewing lifecycle stage.

---

## 3. Scope

**In scope:**
- A new merge gate condition (`ReviewReportExistsGate`) that activates when the target feature's lifecycle status is `reviewing`.
- A `Bypassable` field on the gate result type that controls whether `override: true` is honoured for a given gate result.
- Updated override logic in the merge tool that rejects overrides when the blocking gate is non-bypassable.
- A prescribed error message returned to the caller when this gate blocks.

**Out of scope:**
- Changes to gate evaluation for features in any lifecycle status other than `reviewing`.
- Changes to the bypassability of existing gate conditions.
- Validation or approval requirements on the report document itself (draft status is sufficient).
- Automatic registration of report documents.

---

## 4. Functional Requirements

### Gate activation

**FR-001** — The `ReviewReportExistsGate` MUST be evaluated when `merge(action: execute)` is called and the target feature's current lifecycle status is `reviewing`.

**FR-002** — The `ReviewReportExistsGate` MUST NOT be evaluated when the target feature's lifecycle status is any value other than `reviewing`.

### Gate pass condition

**FR-003** — The gate MUST return `Pass` when at least one document of type `report` is registered with an `owner` matching the target feature ID, regardless of that document's status (draft or approved both satisfy the condition).

### Gate block condition

**FR-004** — The gate MUST return `Blocked` when no document of type `report` is registered with an `owner` matching the target feature ID.

**FR-005** — When the gate returns `Blocked`, the merge tool MUST return an error message that includes all of the following:
  - The feature ID.
  - A statement that the feature is in `reviewing` status but no review report is registered.
  - Numbered resolution steps directing the caller to run the review, register the report, and retry the merge.
  - An explicit statement that this gate cannot be bypassed with `override: true`.

### Non-bypassable behaviour

**FR-006** — The gate result type MUST include a `Bypassable` boolean field.

**FR-007** — When `ReviewReportExistsGate` returns `Blocked`, the result MUST have `Bypassable` set to `false`.

**FR-008** — When the merge tool receives `override: true` and a blocking gate has `Bypassable` set to `false`, the merge MUST be rejected and MUST NOT proceed.

**FR-009** — When the merge tool receives `override: true` and all blocking gates have `Bypassable` set to `true`, the existing override behaviour MUST be preserved unchanged.

**FR-010** — All existing gate conditions MUST continue to set `Bypassable: true` (or its equivalent default), so their bypassability is unchanged by this change.

### Failure modes

**FR-011** — If the document service returns an error when the gate queries for report documents, the gate MUST return `Pass` (fail-open) and the merge MUST proceed.

**FR-012** — When the gate fails open due to a document service error, the system MUST emit a log warning.

---

## 5. Non-Functional Requirements

**NFR-001** — Gate evaluation MUST NOT increase merge tool latency by more than 200 ms under normal operating conditions.

**NFR-002** — The `Bypassable` field MUST be present on the gate result type used by all gate conditions, not only `ReviewReportExistsGate`, so the type contract is uniform.

---

## 6. Acceptance Criteria

**AC-001** — Given a feature in `reviewing` status with no registered report documents, when `merge(action: execute)` is called without `override: true`, then the merge is rejected and the error message contains the feature ID, a reference to `reviewing` status, resolution steps, and a statement that the gate cannot be bypassed.

**AC-002** — Given a feature in `reviewing` status with no registered report documents, when `merge(action: execute, override: true)` is called, then the merge is rejected (override is not honoured) and the error message states the gate cannot be bypassed.

**AC-003** — Given a feature in `reviewing` status with at least one registered report document whose status is `draft`, when `merge(action: execute)` is called, then the gate does not block the merge.

**AC-004** — Given a feature in `reviewing` status with at least one registered report document whose status is `approved`, when `merge(action: execute)` is called, then the gate does not block the merge.

**AC-005** — Given a feature NOT in `reviewing` status (e.g. `developing`), when `merge(action: execute)` is called with no registered report documents, then `ReviewReportExistsGate` does not block the merge.

**AC-006** — Given a feature blocked only by an existing (bypassable) gate condition, when `merge(action: execute, override: true)` is called, then the merge proceeds (existing override behaviour is unaffected).

**AC-007** — Given the document service returns an error while the gate is evaluating, when `merge(action: execute)` is called, then the gate passes, the merge proceeds, and a warning is logged.

---

## 7. Dependencies and Assumptions

**DEP-001** — The gate implementation depends on a document record service that supports listing documents filtered by `owner` and `type`.

**DEP-002** — The gate chain infrastructure must support plugging in new gate conditions without modification to the merge tool's core logic beyond wiring.

**ASM-001** — The document store reliably reflects registrations made via `doc(action: register)` before `merge(action: execute)` is called in normal (non-error) operation.

**ASM-002** — A feature's lifecycle status is the authoritative source for determining which gates are active; no secondary signal is required.

**ASM-003** — `Bypassable: false` is the complete mechanism for preventing override; no additional authentication or role check is required for this gate.
