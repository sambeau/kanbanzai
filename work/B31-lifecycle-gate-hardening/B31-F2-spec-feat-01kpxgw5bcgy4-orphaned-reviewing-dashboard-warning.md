# Specification: Orphaned Reviewing Feature Dashboard Warning

**Feature:** FEAT-01KPXGW5BCGY4  
**Status:** Draft  
**Owner:** FEAT-01KPXGW5BCGY4

---

## 1. Related Work

- **Design:** `work/design/p31-lifecycle-gate-hardening.md` — Component 2: Orphaned Reviewing Feature Dashboard Warning
- **Feature:** FEAT-01KPXGW5BCGY4 — Orphaned reviewing feature dashboard warning
- **Decision 3 (design):** Dashboard check is a warning severity, not an error — surfaces the condition without blocking the user.

---

## 2. Overview

When a feature transitions to `reviewing` status, a review report document is expected to be registered as evidence of completed review work. Without a registered report, the feature is considered orphaned in the reviewing state — the workflow has stalled with no observable artifact.

This feature adds a new check (`OrphanedReviewingFeatureCheck`) to the `status` tool's attention item assembler. The check identifies features in `reviewing` status that have no registered document of type `report` owned by that feature, and surfaces them as `warning`-severity attention items on project-level, plan-scoped, and feature-scoped status dashboards.

---

## 3. Scope

**In scope:**
- A new attention item check that runs during `status()`, `status(id: P-xxx)`, and `status(id: FEAT-xxx)` calls.
- Detection of features in `reviewing` status with no registered `report`-type document owned by the feature.
- Emission of a `warning`-severity attention item for each such feature.

**Out of scope:**
- Blocking or preventing any lifecycle transition based on this condition.
- Retroactive remediation of pre-existing orphaned reviewing features.
- Any check against document types other than `report`.
- Any change to the reviewing lifecycle gate itself.

---

## 4. Functional Requirements

### FR-001 — Check triggers on project-level status
The `OrphanedReviewingFeatureCheck` MUST run when `status()` is called with no scoping argument (project-level scope).

### FR-002 — Check triggers on plan-scoped status
The `OrphanedReviewingFeatureCheck` MUST run when `status(id: P-xxx)` is called, scoped to features belonging to the specified plan.

### FR-003 — Check triggers on feature-scoped status
The `OrphanedReviewingFeatureCheck` MUST run when `status(id: FEAT-xxx)` is called, scoped to that single feature.

### FR-004 — Condition: feature in reviewing status with no report document
The check MUST emit an attention item for a feature if and only if:
- The feature has `status = reviewing`, AND
- No document of type `report` with `owner` equal to the feature ID is registered in the document store.

### FR-005 — Attention item severity
Each attention item emitted by this check MUST have `severity: warning`.

### FR-006 — Attention item entity reference
Each attention item MUST reference the `entity_id` of the affected feature (e.g. `FEAT-xxx`).

### FR-007 — Attention item message content
Each attention item MUST include a human-readable message in the form:
> `Feature FEAT-xxx (slug) is in 'reviewing' status with no registered review report`

where `FEAT-xxx` is the feature ID and `slug` is the feature's slug.

### FR-008 — Check skipped when no reviewing features exist
If no features in the relevant scope have `status = reviewing`, the check MUST produce no attention items and MUST NOT query the document store.

### FR-009 — Check skipped on document service unavailability
If the document record service is unavailable, the check MUST be skipped silently. No error or attention item is emitted due to service unavailability. Attention items from this check are best-effort.

### FR-010 — Pre-existing orphans surface on every status call
A feature that was already in `reviewing` status before this feature was deployed, and has no registered review report, MUST trigger the warning on every subsequent `status()` call until a qualifying report document is registered.

### FR-011 — One document query per reviewing feature
The check MUST query the document store once per reviewing feature in scope, filtered by `owner = feature ID` and `type = report`.

---

## 5. Non-Functional Requirements

### NFR-001 — No blocking behaviour
The check MUST NOT block or delay the `status()` response in any user-visible way beyond the cost of the document store queries described in FR-011.

### NFR-002 — Graceful degradation
Failure of the document store lookup for one feature MUST NOT prevent attention items from being emitted for other features in the same check run.

### NFR-003 — Scope isolation
A plan-scoped `status(id: P-xxx)` call MUST NOT emit attention items for features belonging to other plans.

---

## 6. Acceptance Criteria

### AC-001
Given a project-level `status()` call, when one or more features have `status = reviewing` and no registered `report` document, then the response includes a `warning`-severity attention item for each such feature.

### AC-002
Given a project-level `status()` call, when all features in `reviewing` status each have at least one registered `report` document owned by that feature, then no orphaned-reviewing attention items are present in the response.

### AC-003
Given a plan-scoped `status(id: P-xxx)` call, when a feature in that plan has `status = reviewing` and no registered `report` document, then a `warning`-severity attention item is present for that feature.

### AC-004
Given a plan-scoped `status(id: P-xxx)` call, when a feature in a *different* plan has `status = reviewing` and no registered `report` document, then no attention item for that other-plan feature is present in the response.

### AC-005
Given a feature-scoped `status(id: FEAT-xxx)` call, when that feature has `status = reviewing` and no registered `report` document, then a `warning`-severity attention item is present.

### AC-006
Given a feature-scoped `status(id: FEAT-xxx)` call, when that feature has `status = reviewing` and a registered `report` document owned by `FEAT-xxx` exists, then no orphaned-reviewing attention item is present.

### AC-007
Given a project-level `status()` call, when no features have `status = reviewing`, then no orphaned-reviewing attention items are present and no document store queries are made by this check.

### AC-008
Given the document record service is unavailable, when `status()` is called, then the response is returned without error and without any attention items attributable to document service failure.

### AC-009
Each attention item produced by this check has `severity = warning` (not `error`, `info`, or any other level).

### AC-010
Each attention item message matches the pattern: `Feature FEAT-xxx (slug) is in 'reviewing' status with no registered review report`, where `FEAT-xxx` and `slug` correspond to the affected feature.

---

## 7. Dependencies and Assumptions

### Dependencies

- **Document record service (`docRecordSvc`):** The check depends on the ability to call `docRecordSvc.List()` filtered by `owner` and `type`. This service must be accessible to the `status` tool's attention item assembler.
- **Status tool attention item assembler:** The assembler must support registering new check functions that return lists of `AttentionItem` structs with a severity field. This capability is assumed to exist per the current dashboard architecture.
- **Feature entity store:** The check requires the ability to enumerate features by lifecycle status within a given scope (project, plan, or single feature).

### Assumptions

- **A1:** The existing `status` tool assembler accepts pluggable check functions without architectural changes.
- **A2:** `AttentionItem` structs already carry `severity`, `entity_id`, and `message` fields, or these fields are trivially addable.
- **A3:** A `report`-type document registered with `owner = FEAT-xxx` is the canonical evidence of a completed review for that feature. No other document type serves this purpose.
- **A4:** Features that complete the reviewing stage and advance to a later status (e.g. `done`) are no longer in `reviewing` status and will not be surfaced by this check.
- **A5:** The slug of a feature is always available alongside its ID when the feature entity is loaded.
