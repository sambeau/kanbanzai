# Specification: Document Inheritance for Gap Checks

**Status:** Draft
**Feature:** FEAT-01KN07T66SAH3 (document-inheritance-and-freshness)
**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` — Feature 4
**Date:** 2026-04-02

---

## Problem Statement

Every feature belonging to a plan with an approved specification still shows "Missing specification document" in status dashboards, doc gap checks, and health reports. The current implementation queries documents owned directly by the feature (`ListDocumentsByOwner(featureID)`) and never walks up to the parent plan. In a plan with 8 features sharing one plan-level spec, this produces 8 false-positive warnings that drown out real attention items.

The same problem affects `has_spec` and `has_dev_plan` flags on feature summaries within plan dashboards, and the `doc(action: "gaps")` tool output. Agents and humans cannot distinguish real documentation gaps from inherited-but-unrecognised ones.

---

## Requirements

### Functional Requirements

**FR-1: Query-time fallback to parent plan documents.**
When determining whether a feature satisfies a required document type (specification, design, dev-plan), the system must first check documents owned by the feature. If no document of that type is found on the feature, the system must check documents owned by the feature's parent plan.

**FR-2: Feature documents take precedence.**
If a feature has its own document of a given type — regardless of that document's status — the system must use the feature's document and must not fall back to the plan. A feature with a draft spec of its own does not inherit the plan's approved spec; the feature's draft is the authoritative record.

**FR-3: Only approved plan documents satisfy inheritance.**
A plan-level document must have status `approved` to satisfy an inherited gap check. Draft, superseded, or any other non-approved plan documents do not count. A feature with no spec of its own and a plan with only a draft spec must still report a gap.

**FR-4: Inheritance is read-only.**
Registering a document on a plan must not create, modify, or duplicate document records on child features. Inheritance is resolved at query time only. No document records are written to feature state as a side effect of inheritance.

**FR-5: Inherited documents are distinguishable in output.**
When a gap check or status query resolves a document type via plan inheritance, the output must indicate that the document is inherited rather than directly owned. This allows agents and humans to understand the provenance of the document.

---

## Constraints

**C-1: Three code paths must be updated.** The following are the only code paths that require changes:

1. `docGapsAction` in `doc_tool.go` — the `doc(action: "gaps")` handler.
2. `synthesisePlan` in `status_tool.go` — computes `has_spec` and `has_dev_plan` per feature in plan dashboards.
3. `synthesiseFeature` / `generateFeatureAttention` in `status_tool.go` — computes attention items including "Missing specification document" and "Missing dev-plan document".

**C-2: Parent plan ID is available on feature records.** The feature's parent plan is stored in the `parent` field of the feature entity state (`feat.State["parent"]`). No additional entity lookups are needed to discover the plan ID.

**C-3: No schema changes.** No new fields are added to entity state files. No migration is required. The change is purely in query-time resolution logic.

**C-4: Existing `ListDocumentsByOwner` API is sufficient.** The fallback queries the same `ListDocumentsByOwner(planID)` function. No new service-layer methods are needed.

---

## Acceptance Criteria

### AC-1: doc(action: "gaps") inherits plan spec

**Given** a plan P with an approved specification document registered to it,
**and** a feature F belonging to plan P with no specification document of its own,
**when** `doc(action: "gaps", feature_id: "F")` is called,
**then** the specification type must not appear in the `gaps` array,
**and** it must appear in the `present` array with a field indicating it is inherited from the plan.

### AC-2: doc(action: "gaps") does not inherit draft plan docs

**Given** a plan P with only a draft (unapproved) specification document,
**and** a feature F belonging to plan P with no specification document of its own,
**when** `doc(action: "gaps", feature_id: "F")` is called,
**then** the specification type must appear in the `gaps` array with status `missing`.

### AC-3: Feature's own document takes precedence over plan

**Given** a plan P with an approved specification document,
**and** a feature F belonging to plan P that has its own draft specification document,
**when** `doc(action: "gaps", feature_id: "F")` is called,
**then** the specification type must appear in the `gaps` array with the feature's draft document ID and status `draft`,
**and** the plan's approved specification must not appear in the result.

### AC-4: Plan dashboard has_spec reflects inheritance

**Given** a plan P with an approved specification document,
**and** a feature F belonging to plan P with no specification document of its own,
**when** `status(id: "P")` is called to render the plan dashboard,
**then** the feature summary for F must report `has_spec: true`.

### AC-5: Feature attention suppresses false-positive warnings

**Given** a plan P with an approved specification document and an approved dev-plan document,
**and** a feature F belonging to plan P with neither document type of its own,
**when** `status(id: "F")` is called to render the feature detail,
**then** the attention items must not include "Missing specification document" or "Missing dev-plan document".

### AC-6: Feature attention still warns when plan has no approved doc

**Given** a plan P with no approved specification document,
**and** a feature F belonging to plan P with no specification document of its own,
**when** `status(id: "F")` is called,
**then** the attention items must include "Missing specification document".

### AC-7: Inheritance applies to all three document types

**Given** a plan P with approved documents of types specification, design, and dev-plan,
**and** a feature F belonging to plan P with none of those document types,
**when** `doc(action: "gaps", feature_id: "F")` is called,
**then** the `gaps` array must be empty,
**and** all three types must appear in the `present` array as inherited.

### AC-8: No document records created on features

**Given** a plan P with an approved specification document,
**and** a feature F belonging to plan P,
**when** `doc(action: "gaps", feature_id: "F")` is called,
**then** calling `doc(action: "list", owner: "F")` must return no specification document,
**and** no new document record files must exist in `.kbz/state/documents/` for feature F's specification.

### AC-9: Plan dashboard doc gaps suppressed

**Given** a plan P with an approved specification document,
**and** features F1 and F2 belonging to plan P, neither with their own specification,
**when** `status(id: "P")` is called,
**then** the `doc_gaps` array must not contain entries for F1 or F2 regarding missing specification.

---

## Verification Plan

1. **Unit tests for `docGapsAction`:** Test the fallback logic with combinations of feature-owned docs, plan-owned docs (approved vs draft), and no docs. Verify precedence (AC-1, AC-2, AC-3, AC-7, AC-8).

2. **Unit tests for `synthesisePlan`:** Test `has_spec` and `has_dev_plan` computation with plan-level inheritance. Verify plan dashboard doc gaps are suppressed (AC-4, AC-9).

3. **Unit tests for `generateFeatureAttention`:** Pass doc lists that include inherited plan documents and verify attention items are correctly suppressed or retained (AC-5, AC-6).

4. **Integration test:** Create a plan with an approved spec, add two features with no docs, and call `status` and `doc(action: "gaps")` end-to-end. Verify no false-positive warnings appear.

5. **Negative test — read-only invariant:** After running inheritance queries, verify no new document records were written to the store (AC-8).