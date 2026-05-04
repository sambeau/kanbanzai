| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | AI architect                   |

## Scope

This plan implements the requirements defined in
`work/P46-elicitation-checklist/P46-spec-elicitation-checklist.md`
(DOC-`FEAT-01KQSNSWZGSBP/spec-p46-spec-elicitation-checklist`). It covers
the single task of modifying `.kbz/skills/write-spec/SKILL.md` to insert the
7-item elicitation checklist between the Cross-Reference Check and Step 1.

This plan does not cover: interactive interview mode, codebase exploration
agents, a separate `elicit-requirements` skill, P43 fast-track integration,
or any changes outside the `write-spec` skill file.

## Task Breakdown

### Task 1: Insert elicitation checklist into write-spec skill

- **Description:** Add the 7-item elicitation checklist as a new section in
  `.kbz/skills/write-spec/SKILL.md`, positioned between the Cross-Reference
  Check and Step 1: Read the Design. Each checklist item must match the
  question wording, ordering, and STOP-on-ambiguity behaviour specified in
  REQ-001 through REQ-014. The checklist must also include the design-gate
  relationship statement (REQ-014), the revision-scope note (REQ-012), and
  the no-artifact instruction (REQ-013). All language must be imperative
  with IF/THEN conventions per REQ-NF-002.
- **Deliverable:** Modified `.kbz/skills/write-spec/SKILL.md` with the
  checklist inserted at the correct position.
- **Depends on:** None
- **Effort:** Small (0.5 story points)
- **Spec requirements:** REQ-001 through REQ-014, REQ-NF-001, REQ-NF-002

### Task 2: Verify checklist against acceptance criteria

- **Description:** Run through all 9 acceptance criteria (AC-001 through
  AC-009) against the modified skill file. Verify positioning, item count,
  question phrasing, STOP-on-ambiguity behaviour, revision-scope note,
  no-artifact constraint, design-gate statement, line count (≤60 added),
  and imperative language conventions. This is a manual inspection task —
  the verification plan maps every AC to inspection.
- **Deliverable:** Confirmation that all 9 acceptance criteria pass, or a
  list of failing criteria with remediation notes.
- **Depends on:** Task 1
- **Effort:** Small (0.5 story points)
- **Spec requirements:** AC-001 through AC-009 (verification coverage)

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1

Parallel groups: [Task 1]
Critical path: Task 1 → Task 2
```

## Risk Assessment

### Risk: Checklist insertion position is ambiguous

- **Probability:** Low
- **Impact:** Medium — inserting at the wrong position could disrupt the
  existing Cross-Reference Check or Step 1 flow.
- **Mitigation:** The spec's constraints section explicitly states the
  insertion point ("between the Cross-Reference Check and Step 1"). Task 1
  must match the exact section boundaries. AC-001 verifies positioning.
- **Affected tasks:** Task 1

### Risk: Line count exceeds 60-line budget

- **Probability:** Medium
- **Impact:** Low — exceeding the budget is a non-functional requirement
  failure and would require trimming.
- **Mitigation:** The 7 checklist items, header, and procedural notes must
  be concise. If the initial draft exceeds 60 lines, trim explanatory text
  before declaring the task complete.
- **Affected tasks:** Task 1, Task 2

### Risk: Checklist language drifts from imperative convention

- **Probability:** Low
- **Impact:** Low — inconsistent style reduces readability but doesn't
  affect function.
- **Mitigation:** Model the checklist language on existing STOP directives
  in the write-spec skill (e.g., "IF the design document is not approved →
  STOP"). AC-008 verifies imperative mood.
- **Affected tasks:** Task 1

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-001: Checklist positioned between Cross-Reference Check and Step 1 | Manual inspection | Task 2 |
| AC-002: Exactly 7 numbered items with specified questions | Manual inspection | Task 2 |
| AC-003: STOP-and-flag on unresolved ambiguity | Manual inspection | Task 2 |
| AC-004: Checklist does not run for non-scope-change revisions | Manual inspection | Task 2 |
| AC-005: No instruction to produce a written artifact | Manual inspection | Task 2 |
| AC-006: Design-gate relationship statement present | Manual inspection | Task 2 |
| AC-007: Diff adds ≤ 60 lines | Manual inspection (git diff) | Task 2 |
| AC-008: Imperative mood and IF/THEN convention | Manual inspection | Task 2 |
| AC-009: Each item phrased as a question requiring explicit answer | Manual inspection | Task 2 |
