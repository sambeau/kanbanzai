---
# kanbanzai-managed: true
# kanbanzai-version: dev
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
# kanbanzai-managed: true
# kanbanzai-version: dev

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
- **Detect:** A requirement uses words like "should," "appropriate," "fast," "user-friendly," or "easy to use" without measurable criteria
- **BECAUSE:** Vague requirements cannot be verified — the implementer interprets "fast" as 500ms, the reviewer interprets it as 50ms, and acceptance becomes a negotiation instead of a measurement
- **Resolve:** Replace every qualitative term with a measurable threshold: "Response time for entity listing MUST be ≤ 200ms at p95 under 100 concurrent requests"

### Untestable Acceptance Criterion
- **Detect:** An acceptance criterion describes a desired property without specifying an observable outcome or verification method
- **BECAUSE:** If a criterion cannot be tested, it cannot be verified at the review stage — it becomes an opinion about whether the implementation "feels right" rather than a checkable condition
- **Resolve:** Rewrite each criterion as a testable assertion with a specific input, action, and expected observable outcome

### Solution Masquerading as Requirement
- **Detect:** A requirement specifies implementation details (database choice, API framework, specific algorithms) rather than behaviour
- **BECAUSE:** Implementation-level requirements couple the spec to a specific solution, preventing the implementer from choosing the best approach and making the spec brittle to technology changes
- **Resolve:** Rewrite as a behavioural requirement: describe what the system must do, not how it must do it internally. Implementation guidance belongs in the dev-plan.

### Missing Design Reference
- **Detect:** The Problem Statement section does not cite the parent design document by path or document ID
- **BECAUSE:** Without a design reference, the specification is unanchored — reviewers cannot verify that the requirements implement the agreed design, and design decisions may be silently contradicted
- **Resolve:** The Problem Statement must cite the design document and summarise which design decisions this specification implements

### Orphaned Requirement
- **Detect:** A requirement ID appears in the Requirements section but has no corresponding entry in the Verification Plan
- **BECAUSE:** A requirement without a verification method is unverifiable by definition — it will pass review by default and defects against it will not be caught until production
- **Resolve:** Every requirement ID must appear in the Verification Plan with a specific verification method

### Missing Constraints
- **Detect:** The Constraints section is absent or contains only technical constraints with no scope exclusions
- **BECAUSE:** Without explicit scope boundaries, implementers and reviewers have no shared understanding of what is out of scope — scope creep becomes invisible because there is no boundary to cross
- **Resolve:** List at least one scope exclusion ("This specification does NOT cover...") and any constraints inherited from the design or existing system

### Ambiguous Scope Boundary
- **Detect:** The Problem Statement or Constraints section uses phrases like "and related functionality" or "etc." when defining scope
- **BECAUSE:** Open-ended scope boundaries make it impossible to determine when implementation is complete — every reviewer will draw the boundary differently
- **Resolve:** Replace open-ended phrases with explicit lists. If the boundary genuinely cannot be enumerated, flag it as an open question for the human to resolve.

## Procedure

### Cross-Reference Check

**This step is required** and must be completed before any specification content is written.

**BECAUSE:** Adjacent features share boundaries, shared data structures, and overlapping invariants. A specification written without consulting related specifications can introduce requirements that contradict or silently diverge from established behaviour in neighbouring features. Cross-referencing before writing prevents inconsistency from being baked in from the start — it is far cheaper to align at this stage than to reconcile conflicting specifications during review or implementation.

1. **Verify the Related Work section.** Open the approved design document for this feature and confirm it contains a substantive Related Work section (not empty, not "TBD", not "N/A" without evidence). If the Related Work section is absent or is a placeholder, STOP and flag this to the orchestrator — the design must be updated before specification can proceed.
2. **Search related specs.** For each feature identified in the Related Work section of the design, retrieve its specification and extract its requirements: `doc_intel(action: "find", role: "requirement", scope: "<DOC-related-spec>")`.
3. **Identify consistency constraints.** Review the requirements found in Step 2. Identify any requirements in adjacent specifications that the current specification must be consistent with — shared data shapes, error contracts, lifecycle rules, or behavioural invariants that span features.
4. **Note deliberate divergences.** If this specification takes an approach that differs from an adjacent specification, document the reason explicitly in the current specification. A divergence without a documented reason is indistinguishable from an oversight.

### Step 1: Read the Design

1. Obtain the approved design document for this feature.
2. Read it fully. Understand the problem, the chosen design, the alternatives rejected, and the decisions made.
3. IF the design document is not approved → STOP. A specification must be based on an approved design.
4. IF the design is ambiguous or incomplete for any aspect you need to specify → STOP. Report the ambiguity. Do not infer design intent — the cost of specifying the wrong thing is high.

### Step 2: Define the Problem Scope

1. Write the Problem Statement section, citing the design document by path or document ID.
2. Summarise which design decisions this specification implements.
3. Establish what is in scope and what is explicitly out of scope.
4. IF the scope feels larger than a single specification → flag this for the human. It may need splitting.

### Step 2.5: Determine File Placement

Before writing any content, determine the correct path and filename.
Consult `.agents/skills/kanbanzai-documents/SKILL.md` § "Document Types
and Locations" for the canonical filename template and folder placement.

For a batch-scoped feature specification, the path is:
`work/{BatchID}-{batch-slug}/{BatchID}-{feature-seq}-spec-{slug}.md`

For a plan-scoped specification, the path is:
`work/{PlanID}-{plan-slug}/{PlanID}-spec-{slug}.md`

Use `doc(action: "path", type: "specification", parent: "<FEAT-xxx>")`
to obtain the exact path if available.

### Step 3: Write Requirements

1. Call `now` to get the current date. Record the returned value — you will use it in the document header's `Date` field. Do not guess or invent a date.
2. Derive functional requirements from the design. Each requirement gets a unique ID (REQ-001, REQ-002, ...).
3. Derive non-functional requirements from the design's quality attribute decisions. Each must have a measurable threshold.
4. For each requirement, verify it is testable: can you describe a specific input and expected observable outcome?
5. IF a requirement is not derivable from the design → it may be out of scope, or the design may need updating. Flag it.

### Step 4: Define Constraints and Acceptance Criteria

1. List constraints: scope exclusions, backward compatibility requirements, performance budgets, and any design constraints inherited from the design document.
2. Write acceptance criteria for each requirement. Use testable assertions with observable outcomes.
3. IF a criterion cannot be expressed as a testable assertion → the requirement may be too vague. Revisit the requirement.

### Step 5: Build the Verification Plan

1. For each acceptance criterion, specify the verification method: automated test, manual inspection, or demonstration.
2. Verify that every requirement ID has at least one entry in the Verification Plan.
3. IF a requirement has no feasible verification method → flag it. It may need to be rewritten or removed.

### Step 6: Self-Validate

1. Run the validation script: `.kbz/skills/write-spec/scripts/validate-spec-structure.sh <path>`
2. Verify that the Problem Statement references the design document.
3. Verify that every requirement ID appears in the Verification Plan.
4. Verify that every acceptance criterion is a testable assertion.
5. IF validation fails → fix the structural issue → re-validate.

### Step 7: Present for Review

1. Register the document with `doc(action: register, type: "specification")`.
2. Present the draft to the human reviewer.
3. An approved specification becomes the contract for implementation and review.

## Output Format

Begin every specification with a header table:

```
| Field  | Value                          |
|--------|--------------------------------|
| Date   | {value returned by `now`}      |
| Status | Draft                          |
| Author | {who is writing}               |
```

The specification has exactly 5 required sections. Use these headings:

```
## Problem Statement

State what this specification covers and why. Reference the parent design
document by path or document ID:

> This specification implements the design described in
> `work/design/<design-document>.md` (DOC-xxx).

Summarise the design decisions this specification addresses. Define the
scope boundary: what is included and what is explicitly excluded.

## Requirements

### Functional Requirements

Each requirement has a unique ID:

- **REQ-001:** [Description of required behaviour as an observable outcome]
- **REQ-002:** [Description of required behaviour]

### Non-Functional Requirements

Each must include a measurable threshold:

- **REQ-NF-001:** [Quality attribute] must meet [measurable threshold]
  under [specified conditions]

## Constraints

- What must NOT change (backward compatibility, existing interfaces)
- What limits apply (performance budgets, resource constraints)
- What is out of scope ("This specification does NOT cover...")
- Design constraints inherited from the parent design document

## Acceptance Criteria

Each criterion maps to one or more requirements:

- **AC-001 (REQ-001):** Given [precondition], when [action], then
  [observable outcome]
- **AC-002 (REQ-002):** Given [precondition], when [action], then
  [observable outcome]

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: [brief description] |
| AC-002 | Inspection | Code review verifying [specific property] |
| AC-003 | Demo | Demonstrate [behaviour] in [environment] |
```

## Examples

See [examples-write-spec.md](references/examples-write-spec.md) for worked specification examples: vague spec without verification, implementation spec, and behavioural spec with full traceability.

## Evaluation Criteria

1. Does the Problem Statement reference the parent design document by path or document ID? Weight: required.
2. Does every requirement describe observable behaviour rather than implementation detail? Weight: required.
3. Does every requirement have a unique ID that appears in the Verification Plan? Weight: required.
4. Is every acceptance criterion a testable assertion with a specific expected outcome? Weight: required.
5. Does the Constraints section include at least one explicit scope exclusion? Weight: high.
6. Do non-functional requirements include measurable thresholds with specified conditions? Weight: high.
7. Can an implementer write code from this specification without guessing intent? Weight: high.
8. Can a reviewer verify each acceptance criterion independently using the Verification Plan? Weight: high.

## Questions This Skill Answers

- How do I write a specification for a Kanbanzai feature?
- What sections does a specification document require?
- How do I write testable acceptance criteria?
- How should a specification reference its parent design document?
- What is the difference between a requirement and an acceptance criterion?
- How do I structure a verification plan?
- When should I stop and ask for clarification during specification writing?
- How do I handle non-functional requirements in a specification?
- What makes a specification ready for approval?
- How do I avoid writing implementation details in a specification?