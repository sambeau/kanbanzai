# Spec Validation Report: Error Classification for MCP Handlers

**Validator:** spec-validator  
**Spec:** `FEAT-01KQTNYMZRT6V/spec-p50-spec-error-classification`  
**Design:** `FEAT-01KQTNYMZRT6V/design-p50-design-retro-may-2026`  
**Date:** 2026-05-04

## Verdict: pass_with_notes

All blocking checks pass. One non-blocking check produced findings.

## Summary

| Field | Value |
|-------|-------|
| Spec | `work/P50-retro-may-2026/P50-spec-error-classification.md` |
| Validator | spec-validator |
| Verdict | pass_with_notes |
| Evidence Score | 10/10 checks evaluated |
| Blocking | 6/6 passed |
| Non-Blocking | 1/4 with findings |
| Escalated | 0 borderline findings |

## Blocking Checks — All Passed (6/6)

### S1 — Required Sections Present: PASS
All five required sections present and non-empty: Overview (maps to Problem Statement), Scope, Functional Requirements + Non-Functional Requirements (collectively Requirements), Constraints (Scope Exclusions), Acceptance Criteria, Verification Plan. Word count distribution: 76/65/215/49/61/211/138.

### S2 — Design Document Reference: PASS
Overview references the design by both path (`work/P50-retro-may-2026/P50-design-retro-may-2026.md`) and document ID (`P50-retro-may-2026/design-p50-design-retro-may-2026`).

### S3 — Unique REQ-IDs: PASS
9 requirement IDs identified: REQ-001 through REQ-007, REQ-NF-001, REQ-NF-002. All follow `REQ-NNN` or `REQ-NF-NNN` format. No duplicates, no malformed IDs, no requirement bullet without an ID.

### S4 — Verification Plan Coverage: PASS
All 9 REQ-IDs appear in the Verification Plan. REQ-001→AC-001 through REQ-007→AC-007, REQ-NF-001 and REQ-NF-002 have dedicated verification entries. 0 orphaned IDs.

### S5 — Testable Acceptance Criteria: PASS
All 7 acceptance criteria reviewed against the S5 rubric. Each criterion:
- States observable behaviour (logged `actionlog.Entry` field values, error strings)
- Has explicit preconditions (given), actions (when), and outcomes (then)
- Contains no subjective terms without measurable anchors
- `ErrorType` values are specifically named constants — a tester can assert exact string equality

AC-006 ("when any error path is exercised, then the corresponding action log entry has a non-empty ErrorType matching the error's category") is a meta-criterion covering the five tools collectively. It passes because the verification method (table-driven test) makes it enumerable: each tool × error category combination produces a specific, testable assertion.

### S10 — Non-Functional Measurable Thresholds: PASS
- REQ-NF-001: "must not change the error string" — binary equality check, testable.
- REQ-NF-002: "must not introduce additional latency beyond a single string-match or type-assertion check" — the benchmark test in the verification plan anchors this to a statistical significance check.

## Non-Blocking Findings (1/4)

### S6 — Checkbox Format: FINDING (non-blocking)
Acceptance criteria use `- **AC-NNN (REQ-NNN):**` format with Given/When/Then prose. The stage bindings template specifies `acceptance_criteria_format: "checkbox"` (markdown `- [ ]`). All 7 criteria are structurally well-formed (each has explicit conditions and observable outcomes) but none use checkbox format.

**Recommendation:** Convert to checkbox format or update the stage binding template to accept prose Given/When/Then as an alternative. The criteria are testable and structure is clear — this is a formatting convention mismatch, not a quality issue.

### S7 — No Implementation Instructions: PASS
All 7 requirements describe observable behaviour (ErrorType field values in action log entries), not implementation details. References to `actionlog.Entry` and `ErrorType` are existing system interfaces — acceptable under the S7 rubric (naming is design-authorised from the design document's Feature 1 section). No Go struct names, algorithms, or API signatures prescribed.

### S8 — Scope States Both Directions: PASS
Scope section explicitly enumerates "In scope" (3 items) and "Out of scope" (3 items), plus additional scope exclusions in Constraints section. Both directions present.

### S9 — No Orphaned Requirements: PASS
All 7 functional requirements trace to the parent design document:
- REQ-001 through REQ-005: Design §Feature 1, "Approach" paragraph (error classification taxonomy)
- REQ-006: Design §Feature 1, "Prioritise the high-volume tools first"
- REQ-007: Design §Feature 1, implicit in "it doesn't change error behaviour, only labels what's already happening"
- REQ-NF-001, REQ-NF-002: Design §Feature 1, "Risks" section (additive, no behaviour change)

0 orphaned requirements.
