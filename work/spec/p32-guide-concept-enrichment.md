| Field  | Value                                                                              |
|--------|------------------------------------------------------------------------------------|
| Date   | 2026-04-23                                                                         |
| Status | approved |
| Author | spec-author                                                                        |

# Specification: P32 Guide Response Concept and Classification Enrichment

## Overview

This specification covers two additive enrichments to the `doc_intel(action: "guide")`
response, corresponding to §C2a and §C2b of the P32 design document
`p32-doc-intel-classification-pipeline-hardening`.

- **C2a** — Add a `concepts_suggested` field to the `guide` response, providing per-section
  heading-derived concept name candidates to assist agents during classification preparation.
- **C2b** — Expand the existing `suggested_classifications` array (introduced in P28) to
  cover all heading-deterministic sections beyond the two patterns already handled.

---

## Scope

**In scope:**
- Adding a `concepts_suggested` array to the `guide` response (C2a).
- Expanding the `suggested_classifications` array with additional heading-pattern → role mappings (C2b).
- All changes confined to `internal/mcp/doc_intel_tool.go` and any helper functions it calls.

**Out of scope:**
- Changes to `classify`, `outline`, `section`, `find`, `trace`, `impact`, `search`, or `pending` actions.
- Changes to `internal/docint/types.go` or the classification store schema.
- Changes to how classifications are stored, validated, or auto-applied.
- Any UI or client-side changes.
- Modifications to the underlying Layer 1 index schema.

---

## Problem Statement

The `doc_intel(action: "guide")` response (defined in P28) includes a
`suggested_classifications` array that currently covers only two heading patterns:
"Acceptance Criteria" → `requirement` and "Alternatives Considered" → `alternative`.
Investigation shows approximately 56 % of sections in typical documents are
heading-deterministic, meaning their role can be confidently inferred from the section
title alone. The partial coverage provides incomplete starting-point information and may
mislead agents into believing unmatched sections have no deterministic role.

Additionally, the guide response provides no concept name candidates. Agents must infer
domain concepts from section content they may not yet have read, leading to low
`concepts_intro` population rates in the classification index. Heading text contains
sufficient signal to derive useful concept candidates with a lightweight lexical pass,
removing the need for agents to reason from scratch.

---

## Functional Requirements

**REQ-101 — `guide` response includes `concepts_suggested` field**
`doc_intel(action: "guide")` must include a `concepts_suggested` array in its response. The
field must always be present (never absent, never null). It may be an empty array when no
concepts can be derived from any section title.

**REQ-102 — `concepts_suggested` entry structure**
Each entry in `concepts_suggested` must be an object with exactly three fields:
- `section_path` (string) — the section path identifier, identical in format to the path
  values used in `outline` and `suggested_classifications`.
- `section_title` (string) — the raw heading text for that section as stored in the index.
- `suggested_concepts` ([]string) — one or more concept name strings derived from the
  section heading.

**REQ-103 — One entry per section with non-empty suggestions**
The `concepts_suggested` array must contain at most one entry per section. Sections for
which the derivation pass produces an empty `suggested_concepts` list must be omitted from
the array entirely.

**REQ-104 — Concept derivation from section title and ancestor titles**
The `suggested_concepts` list for a section must be derived by applying a normalising
lexical pass to the section title concatenated with its ancestor section titles (in
root-to-leaf order). The pass must:
1. Split each title on `/`, `-`, and whitespace characters.
2. Strip stop words (articles, prepositions, conjunctions, and common structural words such
   as "and", "the", "a", "an", "of", "for", "in", "on", "to", "or", "with", "by",
   "from", "at", "as", "is", "are", "be").
3. Title-case each remaining token.
4. Deduplicate, preserving first-occurrence order.
5. Discard any token shorter than two characters after normalisation.
The resulting list of tokens becomes `suggested_concepts`.

**REQ-105 — Existing `suggested_classifications` patterns unchanged**
The two heading-pattern → role mappings introduced in P28 must remain present and
unchanged:

| Heading pattern | Role |
|---|---|
| "Acceptance Criteria"; sections whose heading matches `AC-\d+` | `requirement` |
| "Alternatives Considered", "Alternative" | `alternative` |

These requirements are defined in P28 (REQ-007). This specification covers only the
expansion; it does not redefine or supersede the P28 behaviour.

**REQ-106 — `suggested_classifications` expansion: additional heading patterns**
The heading-pattern match table used by `buildSuggestedClassifications` (or equivalent
helper) must be extended to cover the following additional patterns. Matching is
case-insensitive prefix match on the section title after whitespace normalisation.

| Heading prefix (case-insensitive) | Role |
|---|---|
| "Problem and Motivation" | `rationale` |
| "Decisions" | `decision` |
| "Design" | `decision` |
| "Overview", "Summary" | `narrative` |
| "Requirements", "Goals" | `requirement` |
| "Risk", "Risks" | `risk` |
| "Definition", "Glossary" | `definition` |

Where a heading prefix in this table overlaps with a pattern already covered by P28 (e.g.
"Definition" / "Glossary" appearing in REQ-007 of P28), the P28 mapping takes precedence.
There must be no duplication of entries in the output array.

**REQ-107 — `suggested_classifications` entries from expanded patterns follow existing rules**
Every entry added to `suggested_classifications` by the expanded patterns must comply with
the rules established in P28 (REQ-005 and REQ-006): the field must be an array of objects
each containing `section_path`, `role`, and `confidence`; `confidence` must be `"high"`;
the array may be empty but must never be absent.

**REQ-108 — Existing `guide` response fields preserved**
All fields present in the `guide` response before this change — `id`, `outline`,
`entity_refs`, `extraction_hints`, `content_hash`, `taxonomy`, and
`suggested_classifications` — must remain present and unchanged in name and type after this
feature is deployed.

---

## Non-Functional Requirements

**REQ-NF-101 — `guide` response latency overhead**
The addition of `concepts_suggested` and the expanded `suggested_classifications` patterns
must not increase the p95 latency of `doc_intel(action: "guide")` by more than 10 ms for a
document with up to 200 sections, beyond the 25 ms budget already established by P28
(REQ-NF-002).

**REQ-NF-102 — No external calls for concept derivation**
The `concepts_suggested` array must be computed entirely in process using only the section
titles already fetched as part of the existing `guide` outline construction. No additional
database round-trips or external service calls are permitted.

**REQ-NF-103 — Concept derivation is O(n) in section count**
The normalising lexical pass must be stateless and O(n) in the number of sections. No
caching or pre-computation outside the request is required.

---

## Constraints

- This specification is strictly additive. No existing `guide` response field may be
  renamed, removed, or have its JSON type changed.
- Changes must be confined to `internal/mcp/doc_intel_tool.go` (and any helper functions it
  calls). No changes to `internal/docint/types.go` or to the classification store are
  permitted.
- The `suggested_classifications` array must not trigger any write to the document index or
  classification store; it remains computed-on-the-fly and informational only (P28
  constraint, unchanged).
- The stop-word list used in REQ-104 is fixed at implementation time. It must not be
  configurable via the MCP tool call parameters.
- Prefix matching in REQ-106 must use case-insensitive, normalised-whitespace comparison.
  Regex matching is permitted only for the `AC-\d+` and `D\d+:` patterns inherited from
  P28; the new patterns must use prefix comparison.

---

## Acceptance Criteria

**AC-101 (REQ-101):**
Given any indexed document,
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then the response contains a `concepts_suggested` field that is a JSON array (possibly
empty) and is never absent.

**AC-102 (REQ-102, REQ-103):**
Given a document whose outline contains a section titled "Risk Assessment",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `concepts_suggested` contains exactly one entry for that section with
`section_title: "Risk Assessment"` and a non-empty `suggested_concepts` array, and no
second entry for the same `section_path` exists.

**AC-103 (REQ-104):**
Given a section titled "Design / Architecture" with a parent section titled "Overview",
When `doc_intel(action: "guide")` is called,
Then the `suggested_concepts` for that section includes tokens derived from both
"Overview" and "Design / Architecture" (e.g. `["Overview", "Design", "Architecture"]`),
with stop words removed and tokens deduplicated.

**AC-104 (REQ-104):**
Given a section title that consists entirely of stop words (e.g. "In and Of"),
When `doc_intel(action: "guide")` is called,
Then that section is omitted from `concepts_suggested` (no entry with an empty
`suggested_concepts` list is emitted).

**AC-105 (REQ-106):**
Given a document with a section whose title is "Problem and Motivation",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "rationale"` and
`confidence: "high"` for that section.

**AC-106 (REQ-106):**
Given a document with a section whose title is "Decisions",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "decision"` and
`confidence: "high"` for that section.

**AC-107 (REQ-106):**
Given a document with a section whose title is "Design",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes an entry with `role: "decision"` and
`confidence: "high"` for that section.

**AC-108 (REQ-106):**
Given a document with sections titled "Overview" and "Summary",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes entries with `role: "narrative"` and
`confidence: "high"` for both sections.

**AC-109 (REQ-106):**
Given a document with sections titled "Requirements" and "Goals",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes entries with `role: "requirement"` and
`confidence: "high"` for both sections.

**AC-110 (REQ-106):**
Given a document with sections titled "Risk" and "Risks",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes entries with `role: "risk"` and
`confidence: "high"` for both sections.

**AC-111 (REQ-106):**
Given a document with sections titled "Definition" and "Glossary",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` includes entries with `role: "definition"` and
`confidence: "high"` for both sections.

**AC-112 (REQ-105):**
Given a document with a section titled "Acceptance Criteria" and a section titled
"Alternatives Considered",
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then `suggested_classifications` still includes entries with `role: "requirement"` for
"Acceptance Criteria" and `role: "alternative"` for "Alternatives Considered",
unchanged from P28 behaviour.

**AC-113 (REQ-107):**
Given a document where expanded patterns match several sections,
When `doc_intel(action: "guide", id: "<doc-id>")` is called,
Then every entry contributed by the expanded patterns has `confidence: "high"` and
contains the fields `section_path`, `role`, and `confidence`.

**AC-114 (REQ-108):**
Given an existing consumer that reads `id`, `outline`, `entity_refs`, `extraction_hints`,
`content_hash`, `taxonomy`, and `suggested_classifications` from a `guide` response,
When `doc_intel(action: "guide")` is called after this feature is deployed,
Then all seven fields are present with their original names and types.

**AC-115 (REQ-NF-101):**
Given a document with 200 sections,
When `doc_intel(action: "guide")` is called,
Then the p95 response time does not increase by more than 10 ms compared to the same call
without the `concepts_suggested` additions.

---

## Verification Plan

| Criterion | Method     | Description |
|-----------|------------|-------------|
| AC-101    | Test       | Unit test: call `guide` handler on a seeded document; assert response JSON has a `concepts_suggested` key whose value is an array (empty array is acceptable). |
| AC-102    | Test       | Unit test: seed a document with a "Risk Assessment" section; call `guide`; assert exactly one `concepts_suggested` entry for that `section_path`; assert `suggested_concepts` is non-empty. |
| AC-103    | Test       | Unit test: seed a nested section "Design / Architecture" under parent "Overview"; call `guide`; assert `suggested_concepts` contains tokens from both titles with stop words removed and no duplicates. |
| AC-104    | Test       | Unit test: seed a document with a section whose title normalises to all stop words; call `guide`; assert no entry with empty `suggested_concepts` appears in `concepts_suggested`. |
| AC-105    | Test       | Unit test: seed a document with "Problem and Motivation" section; call `guide`; assert `suggested_classifications` entry with `role:"rationale"`, `confidence:"high"`. |
| AC-106    | Test       | Unit test: seed a document with "Decisions" section; call `guide`; assert `suggested_classifications` entry with `role:"decision"`, `confidence:"high"`. |
| AC-107    | Test       | Unit test: seed a document with "Design" section; call `guide`; assert `suggested_classifications` entry with `role:"decision"`, `confidence:"high"`. |
| AC-108    | Test       | Unit test: seed a document with "Overview" and "Summary" sections; call `guide`; assert both sections have `role:"narrative"`, `confidence:"high"` entries. |
| AC-109    | Test       | Unit test: seed a document with "Requirements" and "Goals" sections; call `guide`; assert both sections have `role:"requirement"`, `confidence:"high"` entries. |
| AC-110    | Test       | Unit test: seed a document with "Risk" and "Risks" sections; call `guide`; assert both sections have `role:"risk"`, `confidence:"high"` entries. |
| AC-111    | Test       | Unit test: seed a document with "Definition" and "Glossary" sections; call `guide`; assert both sections have `role:"definition"`, `confidence:"high"` entries. |
| AC-112    | Regression | Regression test: seed a document with "Acceptance Criteria" and "Alternatives Considered" sections; call `guide`; assert P28-defined entries still present with correct roles and `confidence:"high"`. |
| AC-113    | Test       | Unit test: seed a document matching multiple expanded patterns; call `guide`; assert all expanded entries carry `section_path`, `role`, and `confidence:"high"`. |
| AC-114    | Regression | Regression test: assert pre-existing fields `id`, `outline`, `entity_refs`, `extraction_hints`, `content_hash`, `taxonomy`, and `suggested_classifications` are present and type-correct in `guide` response after the change. |
| AC-115    | Benchmark  | Benchmark test: seed a 200-section document; measure `guide` p95 latency with and without `concepts_suggested` additions; assert delta ≤ 10 ms. |
```

Now let me register the document: