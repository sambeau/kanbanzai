| Field  | Value                                                                 |
|--------|-----------------------------------------------------------------------|
| Date   | 2026-04-23                                                            |
| Status | approved |
| Author | spec-author                                                           |

# Specification: Concept Tagging Approval Gate

**Feature:** FEAT-01KPX5CW0QSM7  
**Design reference:** `work/design/p32-doc-intel-classification-pipeline-hardening.md` §C1

## Overview

This specification defines the server-side approval gate that blocks `doc(action: "approve")`
for specification, design, and dev-plan documents that have been classified but have no
`concepts_intro`-populated sections. It escalates concept tagging from a soft advisory
constraint (the `classification_nudge`) to a hard enforcement point at approval time.

---

## Problem Statement

The `doc(action: "approve")` flow (specified in `work/spec/p28-doc-intel-register-workflow.md`)
approves a document and advances it into the permanent record. Documents of type
`specification`, `design`, and `dev-plan` drive the knowledge graph: their classified
sections are the primary source of concept relationships across the corpus. When agents
classify these documents without populating `concepts_intro` on any section, the concept
registry receives no new entries and the graph degrades over time.

Prior attempts to address this through advisory skill guidance produced no measurable
improvement in concept tagging compliance (P27 investigation). A hard gate at approval
time is required. The approve action is the correct enforcement point: it is deliberate,
non-batched, and the last gate before a document becomes permanent.

---

## Scope

**In scope:**
- Adding a concept-tagging gate to `doc(action: "approve")` for document types `specification`, `design`, and `dev-plan`.
- Adding a `GetClassifications` method to the `IntelligenceService` interface.
- Returning a structured `concept_tagging_required` error when the gate fires.
- Skipping the gate when the intelligence service is nil.

**Out of scope:**
- Document types `policy`, `report`, and `research` — these pass through the existing approval flow unchanged.
- Documents with zero classification entries — the existing `classification_nudge` mechanism covers these.
- Changes to any MCP tool actions, tool names, or tool parameters.
- Changes to the `doc_intel` classification or guide flows.
- Any UI or client-side changes.
- Modifications to the underlying Layer 1 index schema.

---

## Functional Requirements

**REQ-001 — Gate scope: document types.**  
The approval gate MUST fire only when the document's type is one of: `specification`,
`design`, or `dev-plan`. For all other document types (`policy`, `report`, `research`),
the gate MUST NOT fire and approval MUST proceed as today.

**REQ-002 — Gate condition: classifications exist.**  
The approval gate MUST fire only when the document has at least one classification entry
in the index (i.e. `GetClassifications(doc.ID)` returns a non-empty slice). If the
document has zero classification entries, the gate MUST NOT fire.

**REQ-003 — Gate condition: no concepts_intro populated.**  
The approval gate MUST fire only when none of the document's classification entries has
a non-empty `concepts_intro` field. If at least one entry has `concepts_intro` with one
or more values, the gate MUST NOT fire and approval MUST proceed as today.

**REQ-004 — Combined gate predicate.**  
The gate fires if and only if all three conditions hold simultaneously:

1. `doc.Type ∈ {specification, design, dev-plan}` (REQ-001)
2. `len(GetClassifications(doc.ID)) > 0` (REQ-002)
3. No entry in `GetClassifications(doc.ID)` has a non-empty `concepts_intro` field (REQ-003)

The gate MUST NOT fire if any one of these conditions is false.

**REQ-005 — Blocked approval error response.**  
When the gate fires, `doc(action: "approve")` MUST return a structured error (not a
successful approval). The error payload MUST include exactly these fields:

- `error_code` (string): `"concept_tagging_required"`
- `document_id` (string): the ID of the document being approved
- `content_hash` (string): the `content_hash` from the most recent classification index
  entry for the document
- `message` (string): the instruction text (REQ-006)

The document MUST NOT be approved when the gate fires. The document's status MUST remain
unchanged.

**REQ-006 — Error message text.**  
The `message` field in the error payload MUST contain:

> "At least one classified section must have concepts_intro populated. Call
> doc_intel(action: \"guide\", id: \"<id>\") to see concept suggestions, then
> doc_intel(action: \"classify\", ...) with concepts_intro on at least one section."

Where `<id>` is the document ID being approved.

**REQ-007 — Skip when intelligence service unavailable.**  
When the intelligence service is unavailable (nil or not configured), the gate check MUST
be skipped entirely and approval MUST proceed as today. The gate MUST NOT return an error
solely because the intelligence service is absent.

**REQ-008 — `GetClassifications` method.**  
The `IntelligenceService` interface MUST expose a method with the signature:

```
GetClassifications(docID string) ([]ClassificationEntry, error)
```

This method returns all classification entries in the index for the given document ID.
An empty slice (not an error) MUST be returned when the document has no classification
entries. The gate uses this method to evaluate REQ-002 and REQ-003.

**REQ-009 — No new tool actions or parameters.**  
This feature MUST NOT introduce new MCP tool actions, tool names, or tool parameters.
The `doc(action: "approve")` interface is unchanged except that it may now return the
`concept_tagging_required` error for in-scope document types.

**REQ-010 — Placement of gate check.**  
The gate check MUST execute before the document status is committed as approved. It MUST
be evaluated as a pre-approval guard within `docApproveOne`, after the document record
is resolved but before `docSvc.ApproveDocument` (or equivalent commit) is called.

---

## Non-Functional Requirements

**REQ-NF-001 — Gate latency.**  
The additional latency introduced by the gate check (the `GetClassifications` call and
predicate evaluation) MUST NOT increase the p99 latency of `doc(action: "approve")` by
more than 20 ms on a document with up to 200 classification entries, measured on the same
hardware as the baseline (no gate).

**REQ-NF-002 — Backward compatibility.**  
Projects that have not enabled doc-intel (intelligence service is nil) MUST observe
identical `doc(action: "approve")` behaviour before and after this feature is deployed.
No new configuration, environment variable, or feature flag is required to preserve the
existing approval flow.

**REQ-NF-003 — Idempotency.**  
If the gate fires and the agent subsequently populates `concepts_intro` on one or more
sections (via `doc_intel(action: "classify")`) and retries `doc(action: "approve")`, the
retry MUST succeed (assuming no other approval precondition has changed). The gate MUST
re-evaluate on each approve call and MUST NOT cache a previous blocked result.

---

## Constraints

- The gate MUST NOT fire for document types `policy`, `report`, or `research`. These
  types are explicitly out of scope and MUST pass through the existing approval flow
  without any classification check.
- The gate MUST NOT fire for documents with zero classification entries. The existing
  `classification_nudge` mechanism (from `doc(action: "register")`) remains the
  first-line prompt for unclassified documents.
- The existing `doc(action: "approve")` contract defined in
  `work/spec/p28-doc-intel-register-workflow.md` MUST NOT be broken. All previously
  valid approval requests that would have succeeded MUST continue to succeed, except for
  those newly blocked by the gate (classified in-scope documents with no `concepts_intro`).
- The `GetClassifications` method MUST be additive to the `IntelligenceService`
  interface. No existing methods on that interface may be removed or have their
  signatures changed.
- The intelligence service nil-skip (REQ-007) MUST be implemented as an explicit nil
  guard, not as a caught error or panic recovery.

---

## Acceptance Criteria

**AC-001.** (REQ-001, REQ-009) Given a document of type `policy`, `report`, or `research`
that has been classified with no `concepts_intro`, when `doc(action: "approve")` is
called, then the document is approved successfully and no `concept_tagging_required`
error is returned.

**AC-002.** (REQ-002) Given a document of type `specification` with zero classification
entries in the index, when `doc(action: "approve")` is called, then the document is
approved successfully and no `concept_tagging_required` error is returned.

**AC-003.** (REQ-003, REQ-004) Given a document of type `design` that has classification
entries and at least one entry with a non-empty `concepts_intro`, when
`doc(action: "approve")` is called, then the document is approved successfully.

**AC-004.** (REQ-004, REQ-005, REQ-006) Given a document of type `specification` that has
at least one classification entry and no entry with a non-empty `concepts_intro`, when
`doc(action: "approve")` is called, then:
- The response contains an error with `error_code: "concept_tagging_required"`
- The response contains the correct `document_id` and a non-empty `content_hash`
- The response `message` contains the instruction text from REQ-006
- The document status is NOT changed to approved

**AC-005.** (REQ-004, REQ-005) Given a document of type `dev-plan` that has at least one
classification entry and no entry with a non-empty `concepts_intro`, when
`doc(action: "approve")` is called, then the response contains
`error_code: "concept_tagging_required"` and the document is not approved.

**AC-006.** (REQ-005) Given that the gate fires for a document, when the error response
is inspected, then `content_hash` matches the `content_hash` present in the document's
most recent classification index entry.

**AC-007.** (REQ-007) Given the intelligence service is nil (not configured), when
`doc(action: "approve")` is called for a `specification` document that would otherwise
trigger the gate, then the document is approved successfully (gate is skipped).

**AC-008.** (REQ-008) Given the `IntelligenceService` interface, when `GetClassifications`
is called with a document ID that has no classification entries, then it returns an empty
slice and a nil error (not an error).

**AC-009.** (REQ-010) Given the gate fires, when the document status is inspected
immediately after the failed approval, then the status is identical to its status before
the approve call was made (not partially mutated).

**AC-010.** (REQ-NF-003) Given a `doc(action: "approve")` call that was blocked by the
gate, when the agent calls `doc_intel(action: "classify")` with `concepts_intro` populated
on at least one section, and then retries `doc(action: "approve")`, then the retry
succeeds and the document is approved.

**AC-011.** (REQ-NF-002) Given a project where the intelligence service is not
initialised, when `doc(action: "approve")` is called for any document type, then the
behaviour is identical to the pre-feature baseline (no errors, no gate, no latency
increase).

---

## Verification Plan

| Criterion | Method     | Description                                                                                                                                                     |
|-----------|------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Test       | Unit test: call `docApproveOne` with fixture documents of types `policy`, `report`, `research` that have mock classifications with no `concepts_intro`; assert approval succeeds and response contains no `concept_tagging_required`. |
| AC-002    | Test       | Unit test: call `docApproveOne` with a `specification` fixture and a mock `GetClassifications` returning an empty slice; assert approval succeeds.              |
| AC-003    | Test       | Unit test: call `docApproveOne` with a `design` fixture whose mock classifications include one entry with a non-empty `concepts_intro`; assert approval succeeds. |
| AC-004    | Test       | Unit test: call `docApproveOne` with a `specification` fixture whose mock classifications have no `concepts_intro` on any entry; assert response has `error_code: "concept_tagging_required"`, correct `document_id`, non-empty `content_hash`, and correct `message` text; assert document status unchanged. |
| AC-005    | Test       | Unit test: same as AC-004 but with a `dev-plan` fixture; assert `concept_tagging_required` is returned.                                                         |
| AC-006    | Test       | Unit test: assert that `content_hash` in the error response equals the `content_hash` from the mock `GetClassifications` return value.                           |
| AC-007    | Test       | Unit test: initialise `docApproveOne` with a nil intelligence service; call approve on a `specification` with mock store classifications present; assert approval succeeds. |
| AC-008    | Test       | Unit test: call the `GetClassifications` implementation with a document ID not present in the index; assert return value is an empty (non-nil) slice and error is nil. |
| AC-009    | Test       | Unit test: assert that after a gate-blocked approve call, the document record in the mock store has the same status field as before the call.                    |
| AC-010    | Test       | Integration test: (1) call approve → blocked; (2) call `doc_intel classify` with `concepts_intro` on one section; (3) call approve again → assert success.      |
| AC-011    | Test       | Unit test: construct handler with nil intelligence service; call approve for all in-scope types; assert responses are identical to baseline (no gate error, document approved). |
```

Now let me write the file and register it: