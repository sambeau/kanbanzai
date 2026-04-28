# Dev Plan: Doc-Intel Guide Enrichment

**Feature:** FEAT-01KPVDDYQQS1Y
**Plan:** P28 — Doc-Intel Polish and Workflow Reliability
**Spec:** work/spec/p28-doc-intel-guide-enrichment.md
**Status:** Draft

---

## Overview

Add three additive enrichments to `doc_intel` responses: `section_count` on `pending` entries,
and `taxonomy` + `suggested_classifications` on `guide` responses. All changes are confined to
`internal/docint/` handler code and response structs.

---

## Scope

This dev plan implements §5.1, §5.2, and §5.3 of the design document
`P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`,
as fully specified in `work/spec/p28-doc-intel-guide-enrichment.md`.

In scope:
- Adding `section_count` to each entry in the `pending` response struct and handler.
- Adding a `taxonomy` block to the `guide` response struct and handler, derived from
  `FragmentRole` constants in `internal/docint/types.go`.
- Adding a `suggested_classifications` array to the `guide` response struct and handler,
  driven by a static heading-pattern table with case-insensitive normalised-whitespace matching.
- Unit, integration, and benchmark tests for all three additions.

Out of scope: Changes to `classify`, `outline`, `section`, `find`, `trace`, `impact`, or
`search` actions; changes to classification storage or validation; UI or client changes;
modifications to the Layer 1 index schema.

---

## Task Breakdown

### Task 1: Add `section_count` to `pending` response

- **Description:** Extend the `pending` response struct with a `SectionCount int` field and
  update the `pending` handler to populate it from the Layer 1 index for each document entry.
  The field must always be present (zero for documents with no indexed sections). No other
  fields in the response may change.
- **Deliverable:** Updated response struct and handler in `internal/docint/` with `section_count`
  populated on every pending entry.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-001, REQ-002

### Task 2: Add `taxonomy` block to `guide` response

- **Description:** Extend the `guide` response struct with a `Taxonomy` object containing
  `roles` and `confidence` fields. Populate `roles` by iterating the `FragmentRole` constants
  defined in `internal/docint/types.go` (not a separate hardcoded slice). Populate `confidence`
  with the fixed array `["high", "medium", "low"]`. No existing `guide` response fields may
  be renamed, removed, or have their types changed.
- **Deliverable:** Updated response struct and handler in `internal/docint/` with `taxonomy`
  block present in every `guide` response.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-003, REQ-004

### Task 3: Add `suggested_classifications` to `guide` response

- **Description:** Extend the `guide` response struct with a `SuggestedClassifications` array.
  Implement a static heading-pattern table covering all mappings in REQ-007 (case-insensitive,
  normalised-whitespace exact matching; regex only for `AC-\d+` and `D\d+:` patterns). For
  each section in the document outline, match its heading against the table and include an
  entry with `section_path`, `role`, and `confidence: "high"` when a match is found. The array
  must always be present (empty when no headings match). The handler must not write to the
  classification store.
- **Deliverable:** Updated response struct and handler in `internal/docint/` with
  `suggested_classifications` populated; heading-pattern table implemented.
- **Depends on:** None
- **Effort:** medium
- **Spec requirement:** REQ-005, REQ-006, REQ-007, REQ-008

### Task 4: Tests for all three enrichments

- **Description:** Write tests covering:
  - Unit: `pending` handler returns `section_count` equal to seeded value; value is `0` for
    documents with no sections (AC-001).
  - Unit: `guide` handler returns `taxonomy.roles` (non-empty array) and
    `taxonomy.confidence == ["high","medium","low"]` (AC-002).
  - Inspection-backed test: confirm `taxonomy.roles` is populated by iterating `FragmentRole`
    constants, not a literal slice (AC-003).
  - Unit: `guide` on a document with no recognisable headings returns `suggested_classifications`
    as a present, empty array (AC-004).
  - Unit: document with "Acceptance Criteria" section → `role:"requirement"` entry (AC-005).
  - Unit: document whose first section is a key-value table → `role:"narrative"` entry (AC-006).
  - Unit: document with "Alternatives Considered" → `role:"alternative"` entry (AC-007).
  - Integration: call `guide` (populating suggestions), then assert no classification records
    written to the index (AC-008).
  - Regression: assert pre-existing fields `id`, `outline`, `entity_refs`, `extraction_hints`,
    `content_hash` are present and type-correct after the change (AC-009).
  - Benchmark: 50-document pending list; assert p95 latency increase ≤ 10 ms (AC-010).
  - Benchmark: 200-section document; assert `guide` p95 latency increase ≤ 25 ms (AC-011).
- **Deliverable:** New and updated test files in `internal/docint/` covering AC-001 through AC-011.
- **Depends on:** Task 1, Task 2, Task 3
- **Effort:** medium
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007, REQ-008, REQ-009

---

## Dependency Graph

```
Task 1 (section_count on pending) ──────────────────────┐
Task 2 (taxonomy block on guide) ────────────────────────┼──► Task 4 (tests)
Task 3 (suggested_classifications on guide) ─────────────┘
```

Tasks 1, 2, and 3 are fully independent (different response structs / handler paths) and can
run in parallel. Task 4 depends on all three because its tests compile against the combined
response types.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `FragmentRole` constant enumeration requires reflection or a manual registry slice | Medium | Low | Inspect `internal/docint/types.go` before implementing; if constants are iota-typed ints, a `String()`-based or a package-level `AllRoles` var is the idiomatic solution |
| Heading-pattern table grows unwieldy or has ambiguous overlaps | Low | Low | REQ-007 defines the minimum set; implement as a sorted slice of `{pattern, role}` structs with a clear match precedence (regex patterns checked last) |
| Layer 1 index access for `section_count` introduces unexpected latency | Low | Medium | Value is already indexed (REQ-NF-001 allows ≤10 ms delta); benchmark in Task 4 confirms compliance |
| `suggested_classifications` accidentally writes to the classification store | Low | High | Handler must be read-only; AC-008 integration test catches any accidental write |
| Merge conflict with other in-flight `internal/docint/` changes | Low | Medium | Coordinate branch ordering; Tasks 1–3 touch disjoint struct fields, so conflict surface is small |

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001: `section_count` present on every pending entry, `0` for empty docs | Unit test | Task 4 |
| AC-002: `taxonomy` block with `roles` array and `confidence: ["high","medium","low"]` | Unit test | Task 4 |
| AC-003: `taxonomy.roles` derived from `FragmentRole` constants, not a literal slice | Code inspection + test | Task 4 |
| AC-004: `suggested_classifications` always present, empty array when no matches | Unit test | Task 4 |
| AC-005: "Acceptance Criteria" heading → `role:"requirement"` entry | Unit test | Task 4 |
| AC-006: Front-matter key-value table → `role:"narrative"` entry | Unit test | Task 4 |
| AC-007: "Alternatives Considered" → `role:"alternative"` entry | Unit test | Task 4 |
| AC-008: `guide` call does not write to classification store | Integration test | Task 4 |
| AC-009: Pre-existing `guide` fields unchanged after enrichment | Regression test | Task 4 |
| AC-010: `pending` p95 latency increase ≤ 10 ms (50-doc list) | Benchmark test | Task 4 |
| AC-011: `guide` p95 latency increase ≤ 25 ms (200-section doc) | Benchmark test | Task 4 |