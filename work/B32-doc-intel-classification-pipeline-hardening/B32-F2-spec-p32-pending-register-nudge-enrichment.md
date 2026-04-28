| Field  | Value                                                                 |
|--------|-----------------------------------------------------------------------|
| Date   | 2026-04-23                                                            |
| Status | approved |
| Author | spec-author                                                           |

# Specification: P32 Pending and Register Nudge Response Enrichment

**Feature:** FEAT-01KPX5CVYWDFF  
**Design reference:** `work/design/p32-doc-intel-classification-pipeline-hardening.md`

## Overview

This specification covers the "pending and register nudge enrichment" work item within the
P32 plan. P28 already delivered the core response enrichments that this feature name
describes — `section_count` on `pending` responses and `content_hash` + `outline` in the
`classification_nudge` object returned by `doc register`. This specification documents what
P32 adds on top of that foundation: the conditions under which the two-call classify-on-register
path (`register` → `classify`) produces a *high-quality* result without an intermediate
`guide` call, and the `concepts_suggested` addition to the `guide` response that closes the
remaining information gap preventing agents from doing so reliably.

---

## Problem Statement

This specification implements the design described in
`work/design/p32-doc-intel-classification-pipeline-hardening.md`.

P28 established the structural preconditions for a two-call classify-on-register workflow:
`doc(action: "register")` now returns a `classification_nudge` object containing `message`,
`content_hash`, and `outline`. An agent that receives this response has every value required
by `doc_intel(action: "classify")` without calling `guide` first.

However, the investigation (Cluster C2) documents that agents consistently skipped the
classify step anyway. The root cause is not a missing field — it is that agents completing a
classify call with only the `register` response lack concept candidates for `concepts_intro`.
Without concept candidates, agents either omit `concepts_intro` (producing semantically empty
index entries) or skip classification entirely. The three-call path (`register` → `guide` →
`classify`) exists precisely because `guide` provides `taxonomy`, `suggested_classifications`,
and (after this feature) `concepts_suggested` — information that makes classification
tractable for a cold-context agent.

The P32 design resolves this by adding `concepts_suggested` to the `guide` response and
expanding `suggested_classifications` coverage. Together, these changes make the two-call
path (`register` → `classify`) a viable high-quality path *when the agent already has
document content in context* — the existing scenario for which P28's enriched `register`
response was designed. For cold-context agents, the recommended path remains three calls,
but the `guide` response is now richer.

---

## Scope

**In scope:**
- Confirming and specifying the constraints under which the two-call classify-on-register
  path (`register` → `classify`) produces a valid, concept-tagged classification.
- Specifying `concepts_suggested` as a new field in the `doc_intel guide` response.
- Specifying the expanded heading-pattern coverage for `suggested_classifications` in
  the `guide` response.

**Out of scope:**
- `section_count` on `pending` responses — fully specified in the P28 guide-enrichment spec
  (REQ-001, REQ-002) and already implemented.
- `content_hash` and `outline` in `classification_nudge` — fully specified in the P28
  register-workflow spec (REQ-001 through REQ-006) and already implemented.
- The `doc approve` concept-tagging gate — specified in a separate P32 spec.
- Changes to the `doc_intel(action: "classify")` interface.
- Changes to the `doc_intel(action: "pending")` or `doc(action: "register")` response shapes
  beyond what P28 already delivered.

---

## Functional Requirements

**REQ-001 — Two-call classify-on-register path is self-sufficient**  
An agent that has document content in context at registration time MUST be able to complete
a valid, concept-tagged `doc_intel(action: "classify")` call using only values from the
`doc(action: "register")` response — without issuing a `doc_intel(action: "guide")` call.
Specifically: `classification_nudge.content_hash` provides the `content_hash` argument,
and `classification_nudge.outline` provides the section structure needed to construct the
`classifications` array. This requirement is already structurally met by P28; it is stated
here to establish the baseline against which REQ-002 and REQ-003 are tested.

**REQ-002 — `guide` response includes `concepts_suggested`**  
`doc_intel(action: "guide")` MUST include a `concepts_suggested` field in its response. The
field MUST be an array of section-concept entries. Each entry MUST contain:
- `section_path` (string): the path of the section in the document outline.
- `section_title` (string): the display title of the section.
- `suggested_concepts` (array of strings): concept name candidates derived from the section
  title and its ancestor title chain.

Sections for which the extraction pass produces no candidates MUST be omitted from the
array. The array MAY be empty but MUST NOT be absent.

**REQ-003 — `concepts_suggested` derivation algorithm**  
The `suggested_concepts` list for each section MUST be derived by a normalising lexical
pass over the section title and its ancestor titles (for nested sections). The pass MUST:
1. Split on `/`, `-`, and whitespace.
2. Strip common English stop words (articles, prepositions, conjunctions).
3. Title-case each remaining token.
4. Deduplicate the resulting list.

The pass MUST NOT use external services, LLM inference, or database lookups beyond the
section outline already held in memory during the `guide` action.

**REQ-004 — Expanded `suggested_classifications` heading coverage**  
The `suggested_classifications` array in the `guide` response MUST cover all of the
following heading patterns in addition to those already specified in the P28 guide-enrichment
spec (REQ-007 of that spec):

| Heading pattern | Role |
|---|---|
| "Problem and Motivation", "Problem Statement", "Motivation" | `rationale` |
| "Decisions", "Decision" | `decision` |
| "Design" | `decision` |
| "Requirements", "Goals", "Non-Goals", "Acceptance Criteria" | `requirement` |
| "Overview", "Summary", "Executive Summary", "Background" | `narrative` |
| "Risk", "Risks" | `risk` |
| "Definition", "Definitions", "Glossary", "Reference Table" | `definition` |

Matching MUST be case-insensitive with normalised whitespace. The patterns above are
additive to the existing P28 coverage; no previously covered pattern may be removed.

**REQ-005 — Existing `guide` response fields preserved**  
The `concepts_suggested` addition MUST NOT alter the field names, types, or structure of
any pre-existing field in the `guide` response: `id`, `outline`, `entity_refs`,
`extraction_hints`, `content_hash`, `classified`, `suggested_classifications`, and
`taxonomy`.

## Non-Functional Requirements

**REQ-NF-001 — `guide` response latency overhead for `concepts_suggested`**  
The addition of `concepts_suggested` MUST NOT increase the p95 latency of
`doc_intel(action: "guide")` by more than 15 ms for a document with up to 200 sections,
given that extraction is a stateless O(n) pass over section titles already loaded for the
outline.

**REQ-NF-002 — No external calls for concept extraction**  
The `concepts_suggested` array MUST be computed entirely in-process. No database
round-trips beyond those already performed by the `guide` action, no external service
calls, and no file I/O are permitted during concept extraction.

**REQ-NF-003 — Backward compatibility**  
Callers that ignore unknown fields in the `guide` response MUST NOT break. The
`concepts_suggested` field is purely additive.

---

## Constraints

- `doc(action: "register")` and `doc_intel(action: "pending")` response shapes MUST NOT
  change as part of this feature; all required enrichments were delivered in P28.
- The `doc_intel(action: "classify")` interface (accepted parameters and response shape)
  MUST NOT change.
- `suggested_classifications` entries are informational. The server MUST NOT use them to
  auto-classify or write to the classification index.
- `concepts_suggested` entries are informational. The server MUST NOT use them to
  auto-populate `concepts_intro` in the classification index.
- The heading-pattern table in REQ-004 is additive; no P28-specified pattern may be
  removed or have its mapped role changed.
- Concept extraction MUST be case-insensitive and MUST NOT require the section text body —
  only the section title (and ancestor titles for nested sections) is used.
- This specification does NOT cover the `doc approve` concept-tagging gate; that is
  specified separately within P32.
- This specification does NOT cover auto-population of `concepts_intro` without agent
  involvement (Alternative C, rejected in the design).

---

## Acceptance Criteria

**AC-001 (REQ-001):**  
Given a document whose content is in the agent's context at registration time,  
when the agent calls `doc(action: "register")` and then immediately calls
`doc_intel(action: "classify")` using `content_hash` and `outline` from
`classification_nudge` in the register response (no intermediate `guide` call),  
then the classify call succeeds without a hash-mismatch error and the response is a valid
classification result.

**AC-002 (REQ-002):**  
Given any indexed document,  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then the response contains a `concepts_suggested` field that is a JSON array (never
absent), and each entry in the array contains `section_path` (string), `section_title`
(string), and `suggested_concepts` (array of strings, non-empty).

**AC-003 (REQ-002):**  
Given an indexed document in which no section title yields any non-stop-word tokens after
the normalising pass,  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then `concepts_suggested` is present and is an empty array.

**AC-004 (REQ-003):**  
Given a document with a section titled "Data Flow for the Happy Path" nested under a parent
section titled "Design",  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then the `concepts_suggested` entry for that section includes title-cased tokens derived
from both the section title and its ancestor titles (e.g. `"Data"`, `"Flow"`, `"Happy"`,
`"Path"`, `"Design"`), deduplicated, with stop words removed.

**AC-005 (REQ-003):**  
Given the concept extraction pass runs on a section whose title contains only stop words
(e.g. "And the"),  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then no entry for that section appears in `concepts_suggested`.

**AC-006 (REQ-004):**  
Given a document with a section titled "Problem and Motivation",  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then `suggested_classifications` includes an entry with `role: "rationale"` and
`confidence: "high"` for that section.

**AC-007 (REQ-004):**  
Given a document with a section titled "Design",  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then `suggested_classifications` includes an entry with `role: "decision"` and
`confidence: "high"` for that section.

**AC-008 (REQ-004):**  
Given a document with a section titled "Requirements",  
when `doc_intel(action: "guide", id: "<doc-id>")` is called,  
then `suggested_classifications` includes an entry with `role: "requirement"` and
`confidence: "high"` for that section.

**AC-009 (REQ-005):**  
Given an existing consumer that reads fields `id`, `outline`, `entity_refs`,
`extraction_hints`, `content_hash`, `classified`, `suggested_classifications`, and
`taxonomy` from a `guide` response,  
when `doc_intel(action: "guide")` is called after this feature is deployed,  
then all eight fields are present with their original names and types.

**AC-010 (REQ-NF-001):**  
Given a document with 200 sections,  
when `doc_intel(action: "guide")` is called,  
then the response time increases by no more than 15 ms p95 compared to the pre-enrichment
baseline for the same document.

**AC-011 (REQ-NF-002):**  
Given concept extraction is running during a `guide` call,  
when the extraction pass executes,  
then no new database queries, file reads, or external service calls are made beyond those
already issued by the `guide` action before this feature.

---

## Verification Plan

| Criterion | Method     | Description |
|-----------|------------|-------------|
| AC-001    | Test       | Integration test: register a fixture document; call classify with only `content_hash` and `outline` from the register nudge (no guide call); assert classify returns a success response with no hash-mismatch error. |
| AC-002    | Test       | Unit test: call `guide` handler with a seeded document; assert response JSON contains `concepts_suggested` as an array; assert each entry has `section_path` (string), `section_title` (string), and `suggested_concepts` (non-empty string array). |
| AC-003    | Test       | Unit test: seed a document whose all section titles consist only of stop words; call `guide`; assert `concepts_suggested` is present and is an empty array `[]`. |
| AC-004    | Test       | Unit test: seed a document with a two-level outline ("Design" → "Data Flow for the Happy Path"); call `guide`; assert the nested section's `suggested_concepts` includes title-cased tokens from both the section title and parent title, deduplicated, stop words removed. |
| AC-005    | Test       | Unit test: seed a section titled "And the"; assert it does not appear in `concepts_suggested` output. |
| AC-006    | Test       | Unit test: seed a document with a "Problem and Motivation" section; call `guide`; assert `suggested_classifications` contains `{role: "rationale", confidence: "high"}` for that section. |
| AC-007    | Test       | Unit test: seed a document with a "Design" section; call `guide`; assert `suggested_classifications` contains `{role: "decision", confidence: "high"}` for that section. |
| AC-008    | Test       | Unit test: seed a document with a "Requirements" section; call `guide`; assert `suggested_classifications` contains `{role: "requirement", confidence: "high"}` for that section. |
| AC-009    | Test       | Regression test: assert that fields `id`, `outline`, `entity_refs`, `extraction_hints`, `content_hash`, `classified`, `suggested_classifications`, and `taxonomy` are all present and type-correct in a `guide` response after the change. |
| AC-010    | Test       | Benchmark test: seed a 200-section document; measure `guide` p95 latency before and after concept extraction is added; assert delta ≤ 15 ms. |
| AC-011    | Inspection | Code review confirms the concept extraction function takes only the section title strings (already in memory from the outline pass) as input and makes no I/O calls. |
```

Now let me register the document: