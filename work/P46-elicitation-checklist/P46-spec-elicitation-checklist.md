# Specification: Elicitation Checklist

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | AI spec-author                 |

## Problem Statement

This specification implements the elicitation checklist enhancement described in
`work/P46-elicitation-checklist/P46-design-elicitation-checklist.md` (DOC-`P46-elicitation-checklist/design-p46-design-elicitation-checklist`). The design introduces a structured checklist that runs before specification content is written, ensuring the spec-author has explicit answers to core intent questions before proceeding.

**In scope:** Add a 7-item pre-spec checklist to the `write-spec` skill. The checklist runs once before Step 1 (Read the Design) and after the Cross-Reference Check. It validates that core objective, scope boundaries, ambiguities, technical approach, test strategy, constraints, and dependencies are explicitly addressed before the spec-author writes any requirements. Checklist execution is procedural — it does not produce a written artifact.

**Out of scope:** Full Prometheus-style interactive interview mode. Codebase exploration agents. A separate `elicit-requirements` skill. Integration with P43 fast-track validators (P43 checks structural completeness; the checklist checks intent clarity — complementary but independent).

## Requirements

### Functional Requirements

- **REQ-001:** The `write-spec` skill must include a pre-spec checklist step placed after the Cross-Reference Check and before Step 1 (Read the Design).
- **REQ-002:** The checklist must contain exactly 7 items: core objective, scope boundaries, ambiguities, technical approach, test strategy, constraints, and dependencies.
- **REQ-003:** Each checklist item must be defined as a question that the spec-author must answer before writing any specification content.
- **REQ-004:** Item 1 (core objective) must ask: "What is the single most important thing this feature must accomplish? State it in one sentence."
- **REQ-005:** Item 2 (scope boundaries) must ask: "What is explicitly IN scope? What is explicitly OUT of scope? If either list is empty, the scope is not defined."
- **REQ-006:** Item 3 (ambiguities) must ask: "What aspects of the design are open to interpretation? List every ambiguity and the chosen resolution. If an ambiguity has no resolution, flag it — do not assume."
- **REQ-007:** Item 4 (technical approach) must ask: "What is the chosen approach? What alternatives were rejected and why?" and must instruct the spec-author to cite the design document (cross-reference check, not a design decision).
- **REQ-008:** Item 5 (test strategy) must ask: "How will correctness be verified? What kinds of tests (unit, integration, e2e) are expected? What edge cases must be covered?"
- **REQ-009:** Item 6 (constraints) must ask: "What constraints does the design impose? (Performance budgets, backward compatibility, API contracts, data migration requirements.)"
- **REQ-010:** Item 7 (dependencies) must ask: "What other features, packages, or external systems does this feature depend on? What depends on this feature?"
- **REQ-011:** When any checklist item cannot be answered from the design document alone, the spec-author must STOP and flag the unresolved item to the human before proceeding to Step 1. The spec-author must not infer or assume an answer.
- **REQ-012:** The checklist must run exactly once for the initial specification. It must not run for specification revisions unless the scope has changed.
- **REQ-013:** The checklist must not produce a separate written artifact. Answers to checklist items inform the specification but are not themselves a document.
- **REQ-014:** The checklist procedure must explicitly state its relationship to the design gate: ambiguities discovered by the checklist must be resolved in the design document, not in the specification. The checklist is a forcing function for design clarity, not a workaround for design gaps.

### Non-Functional Requirements

- **REQ-NF-001:** The checklist addition must not increase the `write-spec` skill file length by more than 60 lines beyond its current size.
- **REQ-NF-002:** The checklist must be expressed in imperative language consistent with the `write-spec` skill's existing procedure steps (numbered, imperative verbs, conditional logic where applicable).

## Constraints

- The `write-spec` skill's existing procedure structure (Cross-Reference Check → Step 1–7) must not be renamed or renumbered. The new checklist step inserts between these existing sections.
- The spec-author's identity, vocabulary, and anti-patterns must not be modified.
- The checklist does not create a new role, new skill, new MCP tool, or new entity type.
- This specification does NOT cover: an interactive interview mode, codebase exploration agents, a separate `elicit-requirements` skill, or integration with the P43 fast-track spec-validator.
- The checklist does not replace or weaken the design gate. The design must still be approved before specification.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given the `write-spec` skill file, then the procedure section contains a checklist step positioned between the Cross-Reference Check and Step 1.
- **AC-002 (REQ-002–REQ-010):** Given the checklist section in the skill, then it contains exactly 7 numbered items matching the specified questions for core objective, scope boundaries, ambiguities, technical approach, test strategy, constraints, and dependencies.
- **AC-003 (REQ-011):** Given a design document with an unresolved ambiguity, when the spec-author reaches the ambiguities checklist item, then the procedure instructs the spec-author to STOP and flag to the human, using the existing "STOP and ask the human" convention from the `write-spec` skill.
- **AC-004 (REQ-012):** Given a specification revision where scope has not changed, when the spec-author runs the `write-spec` procedure, then the checklist step is not presented. (The procedure note: "The checklist runs once, before any specification content is written. It does not run for spec revisions unless the scope has changed.")
- **AC-005 (REQ-013):** Given the checklist section, then it contains no instruction to write output to a file, register a document, or produce an artifact. It is procedural only.
- **AC-006 (REQ-014):** Given the checklist section, then it includes a statement that ambiguities discovered by the checklist must be resolved in the design document, not in the specification.
- **AC-007 (REQ-NF-001):** Given the modified `write-spec` skill file, when compared with the current version, then the diff adds no more than 60 lines.
- **AC-008 (REQ-NF-002):** Given the checklist procedure text, then every sentence uses imperative mood (e.g., "Verify," "Flag," "State") and conditional logic uses the IF/THEN convention established in the existing skill file.
- **AC-009 (REQ-003):** Given the checklist, when a spec-author reads the core objective item, then the item is phrased as a question and the spec-author must produce an explicit answer before continuing.

## Verification Plan

| Requirement(s) | Method | Description |
|----------------|--------|-------------|
| REQ-001 | Inspection | Open the `write-spec` SKILL.md and verify the checklist step appears after Cross-Reference Check and before Step 1 in the Procedure section |
| REQ-002 | Inspection | Count the checklist items: exactly 7 numbered items must be present |
| REQ-003 | Inspection | Verify each checklist item is phrased as a question the spec-author must answer before writing spec content |
| REQ-004 | Inspection | Verify item 1 asks for the core objective in one sentence |
| REQ-005 | Inspection | Verify item 2 asks for in-scope and out-of-scope lists |
| REQ-006 | Inspection | Verify item 3 asks for ambiguities and resolutions |
| REQ-007 | Inspection | Verify item 4 asks for approach, rejected alternatives, and instructs citing the design document |
| REQ-008 | Inspection | Verify item 5 asks for test strategy (unit, integration, e2e) and edge cases |
| REQ-009 | Inspection | Verify item 6 asks for design-imposed constraints |
| REQ-010 | Inspection | Verify item 7 asks for dependencies and dependents |
| REQ-011 | Inspection | Verify the STOP-and-flag instruction uses the same language pattern as existing STOP directives in the skill file |
| REQ-012 | Inspection | Verify the procedure includes the note about the checklist not running for non-scope-change revisions |
| REQ-013 | Inspection | Search the checklist section for "register," "write," "output," "artifact," "document" — none should appear as instructions to produce output |
| REQ-014 | Inspection | Verify the design-gate statement appears in the checklist section |
| REQ-NF-001 | Inspection | Run `git diff` on the skill file and verify the added lines count is ≤ 60 |
| REQ-NF-002 | Inspection | Review every sentence in the checklist for imperative mood and IF/THEN convention |
