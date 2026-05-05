---
name: write-spec
description:
  expert: "Specification authoring producing a gate-checkable 5-section specification
    with traceable requirements, testable acceptance criteria, and a verification plan
    cross-referenced to the parent design document and batch during the specifying stage"
  natural: "Write a specification that turns an approved design into verifiable
    requirements with acceptance criteria and a plan for how to verify each one"
triggers:
  - write a specification
  - draft a spec for a feature
  - create a specification document
  - author a spec from this design
  - produce acceptance criteria for a feature
roles: [spec-author]
stage: specifying
constraint_level: high
---

## Vocabulary

- **acceptance criterion** — a testable condition that must hold for a requirement to be considered met; each criterion has exactly one observable outcome
- **testable assertion** — a statement about system behaviour phrased so that a test, inspection, or demonstration can produce a pass/fail result
- **traceability** — the ability to follow a requirement from its origin in the design through the specification to its verification method
- **functional requirement** — a requirement describing what the system must do, identified by a unique requirement ID (e.g., REQ-001)
- **non-functional requirement** — a requirement describing a quality attribute (performance, security, reliability) with a measurable threshold
- **constraint** — a limitation imposed by existing systems, policies, or decisions that restricts the solution space without specifying behaviour
- **verification plan** — the mapping from each acceptance criterion to the method (test, inspection, demo) that will confirm it is met
- **requirement ID** — a unique identifier (e.g., REQ-001) assigned to each requirement for cross-referencing in reviews, plans, and tests
- **precondition** — a state that must hold before a specified behaviour is triggered
- **postcondition** — a state that must hold after a specified behaviour completes successfully
- **boundary condition** — an input or state at the edge of a valid range, where off-by-one and edge-case defects concentrate
- **scope exclusion** — an explicit statement of what the specification does not cover, preventing scope creep during implementation
- **specification ambiguity** — a statement in the spec that admits more than one reasonable interpretation, requiring clarification before implementation
- **requirement dependency** — a relationship where one requirement cannot be implemented or verified without another being met first
- **verification method** — the technique used to confirm an acceptance criterion: automated test, manual inspection, or live demonstration
- **observable behaviour** — an externally visible system action (output, state change, side effect) that can be checked without knowledge of internals
- **given/when/then** — a structured format for acceptance criteria: Given a precondition, When an action occurs, Then an observable outcome results
- **design reference** — a citation linking the specification back to the approved design document that motivated it
- **out of scope** — a capability or behaviour explicitly excluded from this specification, documented in the Constraints section
- **requirement conflict** — two or more requirements that cannot both be satisfied simultaneously, requiring resolution before implementation
- **parent batch** — the batch entity that owns the feature for which this specification is written; determines which group of features this spec's delivery is tracked against

## Anti-Patterns

### Vague Requirement
