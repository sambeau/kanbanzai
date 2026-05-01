# Dev-Plan: Expand Decompose AC Format Recognition (B40-F4)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG2Q5V4S6W           |
| Spec   | B40-fix-tool-correctness/spec-p40-spec-b40-tool-correctness |

---

## Overview

Extend `decompose(action: propose)` to recognise acceptance criteria in four
additional formats beyond the existing three: heading-based ACs, bold-with-
parenthetical references, Given/When/Then blocks, and numbered lists under an
"Acceptance Criteria" heading. On failure, emit a diagnostic with closest
candidates, line numbers, and an expected-format example.

---

## Task Breakdown

### T1 — Add new AC format patterns to the decompose parser

**Deliverable:** Updated AC extraction logic with four new format recognisers.

**Scope:**
- Add pattern for heading-based: `### AC-NNN` or `### AC-NNN: description`.
- Add pattern for bold-with-parenthetical: `**AC-NNN (REQ-NNN):** description`.
- Add pattern for Given/When/Then blocks under an AC section.
- Add pattern for numbered items under an "Acceptance Criteria" heading.
- Preserve existing patterns (bold-with-period, checklist).
- Ensure new patterns don't create false positives with existing patterns.

**Dependencies:** None.

**Verification:** Inspect parser regex/patterns for correctness.

**Estimated effort:** 2 (four new patterns, interaction testing with existing)

### T2 — Add diagnostic error on parse failure

**Deliverable:** Enhanced error message when no ACs are recognised.

**Scope:**
- When no ACs match any supported format, emit an error containing:
  - List of sections found in the document.
  - Closest unrecognised patterns with line numbers.
  - Example of an expected format.

**Dependencies:** T1 (diagnostic runs after new patterns are tried).

**Verification:** Inspect error output format against AC-009 requirements.

**Estimated effort:** 1 (error message enhancement)

### T3 — Write tests for all AC formats and diagnostic

**Deliverable:** Test file covering all formats and the failure diagnostic.

**Scope:**
- Test: heading-based ACs produce correct task proposal (AC-006).
- Test: bold-with-parenthetical ACs produce correct task proposal (AC-007).
- Test: Given/When/Then blocks produce correct task proposal (AC-008).
- Test: unrecognised format produces diagnostic with sections, candidates,
  and format example (AC-009).
- Test: existing formats (bold-period, checklist) still work (AC-010).

**Dependencies:** T1, T2 (needs both format recognition and diagnostic).

**Verification:** `go test ./...` passes all new and existing decompose tests.

**Estimated effort:** 2 (five test cases with spec fixtures)

---

## Dependency Graph

```
T1 (new formats) ──┬──→ T3 (tests)
T2 (diagnostic)  ──┘
```

T1 and T2 can run in parallel (different code areas: parser vs. error formatting).
T3 depends on both.

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T3 | T1 adds four new AC format recognisers to the parser. The existing two formats continue to work. T3 tests all six formats produce correct proposals. |
| T2 → T3 | T2 enhances the "no criteria found" error with section list, closest candidates, and format example. T3 verifies the diagnostic against AC-009. |

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-007 | T1 |
| REQ-008 | T2 |
| AC-006 | T3 |
| AC-007 | T3 |
| AC-008 | T3 |
| AC-009 | T3 |
| AC-010 | T3 |
