| Field  | Value                                                                 |
|--------|-----------------------------------------------------------------------|
| Date   | 2026-04-22                                                            |
| Status | approved |
| Author | spec-author                                                           |

# Specification: Doc-Intel Guide Enrichment

## Overview

This specification covers three additive enrichments to the `doc_intel` tool: adding a
`section_count` field to `pending` response entries, and adding `taxonomy` and
`suggested_classifications` blocks to `guide` responses, corresponding to §5.1, §5.2, and
§5.3 of the P28 design document.

## Problem Statement

This specification covers three additive enrichments to `doc_intel` responses for the
Kanbanzai project, corresponding to §5.1, §5.2, and §5.3 of the plan design document
`P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`.

The current `doc_intel` tool returns insufficient metadata in two key responses:

- `doc_intel(action: "pending")` returns a flat list of document IDs with no size metadata,
  forcing agents to call `guide` on each document before they can right-size classification
  batches.
- `doc_intel(action: "guide")` does not include the valid taxonomy (roles, confidence levels),
  causing agents to produce invalid role names observed in the pilot. It also provides no
  pre-populated classification hints, requiring agents to infer roles from scratch for every
  section.

## Scope

**In scope:**
- Adding `section_count` to each entry in the `pending` response.
- Adding a `taxonomy` block to the `guide` response.
- Adding a `suggested_classifications` array to the `guide` response.
- All changes confined to `internal/docint/` handler code and response structs.

**Out of scope:**
- Changes to `classify`, `outline`, `section`, `find`, `trace`, `impact`, or `search` actions.
- Changes to how classifications are stored or validated.
- Any UI or client-side changes.
- Modifications to the underlying Layer 1 index schema.

---

## Functional Requirements

**REQ-001 — `pending` response includes `section_count`**
`doc_intel(action: "pending")` must return a `section_count` integer field on each document
entry in its response. The value must equal the number of indexed sections for that document
as stored in the Layer 1 index. The existing structure of the `pending` response must not
change; `section_count` is purely additive.

**REQ-002 — `section_count` is non-negative and always present**
The `section_count` field must always be present on every document entry in the `pending`
response (never omitted, never null). For a document with no indexed sections the value must
be `0`.

**REQ-003 — `guide` response includes `taxonomy` block**
`doc_intel(action: "guide")` must include a `taxonomy` object in its response containing
exactly two keys:
- `roles`: an array of all valid `FragmentRole` string constants.
- `confidence`: the array `["high", "medium", "low"]`.

**REQ-004 — `taxonomy.roles` derived from `FragmentRole` constants**
The `roles` array in the `taxonomy` block must be derived at server build time from the
`FragmentRole` constants defined in `internal/docint/types.go`. It must not be a separate
hardcoded list. Any future constant added to `FragmentRole` must automatically appear in
`guide` responses without a separate code change.

**REQ-005 — `guide` response includes `suggested_classifications` array**
`doc_intel(action: "guide")` must include a `suggested_classifications` array in its
response. The array may be empty but must never be absent. Each entry must be a
classification object with the fields `section_path`, `role`, and `confidence`.

**REQ-006 — Suggestions are high-confidence heading-pattern matches only**
The `suggested_classifications` array must contain only entries where the section heading
matches a defined heading pattern with high confidence. Medium and low confidence matches
must not appear. The confidence value on every entry in the array must be `"high"`.

**REQ-007 — Minimum required heading-pattern → role mappings**
The heading-pattern matching must cover at minimum the following mappings
(case-insensitive, normalised whitespace):

| Heading pattern | Role |
|---|---|
| "Acceptance Criteria"; sections whose heading matches `AC-\d+` | `requirement` |
| "Purpose", "Motivation", "Problem Statement", "Problem and Motivation" | `rationale` |
| "Scope", "In Scope", "Out of Scope", "Deferred", "Excluded", "Non-Goals" | `constraint` |
| "Glossary", "Definitions", "Reference Table", "Definition" | `definition` |
| "Example", "Sample" | `example` |
| "Alternatives Considered", "Alternative" | `alternative` |
| Front matter metadata table (first section containing a key-value table) | `narrative` |
| "Overview", "Background", "Executive Summary" | `narrative` |
| "Decision"; sections whose heading matches `D\d+:` | `decision` |
| "Risk", "Risks" | `risk` |
| "Assumption", "Assumptions" | `assumption` |

**REQ-008 — Suggestions are informational only**
The server must not use `suggested_classifications` to auto-classify or auto-approve any
document. The array is informational; classification state is only updated by explicit
`doc_intel(action: "classify")` calls.

**REQ-009 — Existing `guide` response fields preserved**
The existing fields `id`, `outline`, `entity_refs`, `extraction_hints`, and `content_hash`
must remain present and unchanged in the `guide` response. No existing field may be
renamed, removed, or have its type changed.

## Non-Functional Requirements

**REQ-NF-001 — `pending` response latency overhead**
The addition of `section_count` must not increase the p95 latency of `doc_intel(action:
"pending")` by more than 10 ms over a baseline measured on a 50-document pending list,
given that the value is read from the existing Layer 1 index with no new computation.

**REQ-NF-002 — `guide` response latency overhead**
The addition of `taxonomy` and `suggested_classifications` must not increase the p95 latency
of `doc_intel(action: "guide")` by more than 25 ms for a document with up to 200 sections.

**REQ-NF-003 — No external calls for taxonomy or suggestions**
The `taxonomy` block and `suggested_classifications` array must be computed entirely in
process (no database round-trips beyond what `guide` already performs, no external service
calls).

**REQ-NF-004 — Backward compatibility**
Callers that ignore unknown fields must not break. The additions must not alter the JSON
key names or types of any pre-existing response field.

---

## Constraints

- The `FragmentRole` constants in `internal/docint/types.go` must not be changed as part of
  this feature; `REQ-004` requires reading them, not redefining them.
- The `suggested_classifications` array must not trigger any write to the document index or
  classification store; it is read-only / computed-on-the-fly.
- Changes must be confined to `internal/docint/` handler code and response structs. No
  changes to other packages unless strictly required for type imports.
- The `decompose`, `doc`, `entity`, and all other MCP tools are out of scope for this feature.
- Heading-pattern matching must use case-insensitive, normalised-whitespace comparison.
  Regex matching is permitted only for the `AC-\d+` and `D\d+:` patterns; all other patterns
  must be exact string matches after normalisation.

---

## Acceptance Criteria

**AC-001 (REQ-001, REQ-002):**
Given a pending document list with at least one document,
When `doc_intel(action: "pending")` is called,
Then every document entry in the response includes a `section_count` integer field equal to
the number of sections in the Layer 1 index for that document, and the field is present even
for documents with zero indexed sections (value `0`).

**AC-002 (REQ-003):**
Given any indexed document,
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then the response contains a `taxonomy` object with a `roles` array and a `confidence` array
`["high", "medium", "low"]`.

**AC-003 (REQ-004):**
Given that a new `FragmentRole` constant `"question"` is defined in
`internal/docint/types.go`,
When `doc_intel(action: "guide")` is called,
Then `"question"` appears in `taxonomy.roles` without any other code change.

**AC-004 (REQ-005, REQ-006):**
Given any indexed document,
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then the response contains a `suggested_classifications` array (possibly empty), and every
entry in the array has `confidence: "high"`.

**AC-005 (REQ-007):**
Given a document whose outline contains a section titled "Acceptance Criteria",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "requirement"` and
`confidence: "high"` for that section.

**AC-006 (REQ-007):**
Given a document whose first section is a front-matter key-value metadata table,
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "narrative"` and
`confidence: "high"` for that section.

**AC-007 (REQ-007):**
Given a document with a section titled "Alternatives Considered",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "alternative"` and
`confidence: "high"` for that section.

**AC-008 (REQ-008):**
Given a document with heading-matched sections in `suggested_classifications`,
When `doc_intel(action: "guide")` is called but `doc_intel(action: "classify")` is never
called,
Then the document's classification state in the index remains unchanged (no sections are
auto-classified).

**AC-009 (REQ-009):**
Given an existing consumer that reads the fields `id`, `outline`, `entity_refs`,
`extraction_hints`, and `content_hash` from a `guide` response,
When `doc_intel(action: "guide")` is called after this feature is deployed,
Then all five fields are present with their original names and types.

**AC-010 (REQ-NF-001):**
Given a pending list of 50 documents each with up to 500 sections,
When `doc_intel(action: "pending")` is called,
Then the response time increases by no more than 10 ms p95 compared to the pre-enrichment
baseline measured on the same dataset.

**AC-011 (REQ-NF-002):**
Given a document with 200 sections,
When `doc_intel(action: "guide")` is called,
Then the response time increases by no more than 25 ms p95 compared to the pre-enrichment
baseline for the same document.

---

## Verification Plan

| Criterion | Method     | Description |
|-----------|------------|-------------|
| AC-001    | Test       | Unit test: call `pending` handler with a seeded index; assert every entry has `section_count` equal to seeded value; assert value is `0` for a document with no sections. |
| AC-002    | Test       | Unit test: call `guide` handler; assert response JSON contains `taxonomy.roles` (array, non-empty) and `taxonomy.confidence == ["high","medium","low"]`. |
| AC-003    | Inspection | Code review confirms `taxonomy.roles` is populated by iterating `FragmentRole` constants, not a separate literal slice. |
| AC-004    | Test       | Unit test: call `guide` on a document with no recognisable headings; assert `suggested_classifications` is present and is an empty array. |
| AC-005    | Test       | Unit test: index a document with an "Acceptance Criteria" section; call `guide`; assert matching entry in `suggested_classifications` with `role:"requirement"`, `confidence:"high"`. |
| AC-006    | Test       | Unit test: index a document whose first section is a key-value table; call `guide`; assert `suggested_classifications` entry with `role:"narrative"`. |
| AC-007    | Test       | Unit test: index a document with "Alternatives Considered" section; call `guide`; assert `role:"alternative"` entry. |
| AC-008    | Test       | Integration test: call `guide` (which populates suggestions); then inspect classification store; assert no classification records written. |
| AC-009    | Test       | Regression test: assert pre-existing fields `id`, `outline`, `entity_refs`, `extraction_hints`, `content_hash` are present and type-correct in `guide` response after the change. |
| AC-010    | Test       | Benchmark test: seed 50 documents; measure `pending` p95 latency before and after; assert delta ≤ 10 ms. |
| AC-011    | Test       | Benchmark test: seed a 200-section document; measure `guide` p95 latency before and after; assert delta ≤ 25 ms. |