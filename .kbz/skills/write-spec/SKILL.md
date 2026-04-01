---
name: write-spec
description:
  expert: "Specification authoring producing a gate-checkable 5-section specification
    with traceable requirements, testable acceptance criteria, and a verification plan
    cross-referenced to the parent design document during the specifying stage"
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

### Step 3: Write Requirements

1. Derive functional requirements from the design. Each requirement gets a unique ID (REQ-001, REQ-002, ...).
2. Derive non-functional requirements from the design's quality attribute decisions. Each must have a measurable threshold.
3. For each requirement, verify it is testable: can you describe a specific input and expected observable outcome?
4. IF a requirement is not derivable from the design → it may be out of scope, or the design may need updating. Flag it.

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

### BAD: Vague Specification Without Verification

> ## Problem Statement
> We need to add caching to improve performance.
>
> ## Requirements
> - The system should be fast
> - Caching should work correctly
> - The API should be user-friendly
>
> ## Constraints
> None
>
> ## Acceptance Criteria
> - Performance is acceptable
> - Caching works as expected
>
> ## Verification Plan
> Test everything.

**WHY BAD:** Problem Statement has no design reference and no scope boundary. Requirements are vague — "fast," "correctly," and "user-friendly" have no measurable meaning. No requirement IDs, so nothing is traceable. Constraints says "None," which means scope exclusions are missing. Acceptance criteria are untestable opinions. Verification Plan is a single word with no mapping to criteria.

### BAD: Implementation Specification

> ## Problem Statement
> Implement the caching design from `work/design/caching.md`.
>
> ## Requirements
> - **REQ-001:** Use Redis 7.x as the cache backend
> - **REQ-002:** Use the `go-redis/redis` v9 client library
> - **REQ-003:** Set TTL to 300 seconds for all cache entries
> - **REQ-004:** Use `entity:{id}` as the key format
>
> ## Constraints
> Must use the existing Redis cluster.
>
> ## Acceptance Criteria
> - **AC-001 (REQ-001):** Redis 7.x is installed and running
> - **AC-002 (REQ-002):** go-redis v9 is in go.mod
>
> ## Verification Plan
> | Criterion | Method | Description |
> |-----------|--------|-------------|
> | AC-001 | Inspection | Check Redis version |
> | AC-002 | Inspection | Check go.mod |

**WHY BAD:** Every requirement specifies implementation detail (specific library, specific key format, specific TTL) rather than behaviour. The acceptance criteria test the tools, not the system's behaviour. A different caching implementation that meets the same behavioural goals would "fail" this spec even if it works correctly.

### GOOD: Behavioural Specification with Full Traceability

> ## Problem Statement
>
> This specification implements the read-through caching design described in
> `work/design/entity-caching.md` (DOC-042). The design introduces a shared
> cache behind the `EntityReader` interface to keep p95 latency for
> `GET /entities` below the 200ms SLO as entity volume grows.
>
> **In scope:** Cache read path, cache invalidation on writes, cache health
> monitoring endpoint.
> **Out of scope:** Cache warm-up strategy, multi-region cache replication.
>
> ## Requirements
>
> ### Functional Requirements
>
> - **REQ-001:** Entity listing responses served from cache must return
>   the same data as a direct storage read for the same query parameters.
> - **REQ-002:** When an entity is created, updated, or deleted, the
>   corresponding cache entries must be invalidated before the write
>   operation returns to the caller.
> - **REQ-003:** The system must expose a health endpoint that reports
>   cache availability status (available, degraded, unavailable).
> - **REQ-004:** When the cache is unavailable, entity listing must fall
>   back to direct storage reads without returning an error to the caller.
>
> ### Non-Functional Requirements
>
> - **REQ-NF-001:** Cached entity listing responses must have p95 latency
>   ≤ 50ms under 100 concurrent requests.
> - **REQ-NF-002:** Cache miss penalty must not increase p95 latency beyond
>   250ms (50ms overhead above the current 200ms direct-read baseline).
>
> ## Constraints
>
> - The `EntityReader` interface must not change — the cache must be
>   transparent to consumers of this interface.
> - Existing entity listing tests must continue to pass without modification.
> - This specification does NOT cover cache warm-up on cold start or
>   multi-region replication.
>
> ## Acceptance Criteria
>
> - **AC-001 (REQ-001):** Given a cached entity listing, when the same
>   query is executed against storage directly, then both responses
>   contain identical entity data.
> - **AC-002 (REQ-002):** Given a cached entity, when that entity is
>   updated via the write path, then a subsequent cache read returns the
>   updated data (not the stale cached version).
> - **AC-003 (REQ-003):** Given the cache is running, when
>   `GET /health/cache` is called, then the response includes a status
>   field with value "available", "degraded", or "unavailable".
> - **AC-004 (REQ-004):** Given the cache is unavailable, when an entity
>   listing request arrives, then the response is served from storage
>   with no error visible to the caller.
> - **AC-005 (REQ-NF-001):** Given 100 concurrent entity listing requests
>   with a warm cache, then p95 response time is ≤ 50ms.
> - **AC-006 (REQ-NF-002):** Given 100 concurrent entity listing requests
>   with an empty cache, then p95 response time is ≤ 250ms.
>
> ## Verification Plan
>
> | Criterion | Method | Description |
> |-----------|--------|-------------|
> | AC-001 | Test | Comparison test: cache read vs. direct storage read |
> | AC-002 | Test | Write-then-read test confirming invalidation |
> | AC-003 | Test | Health endpoint integration test with cache up/down |
> | AC-004 | Test | Fault injection: disable cache, verify fallback |
> | AC-005 | Test | Load test with warm cache, assert p95 ≤ 50ms |
> | AC-006 | Test | Load test with cold cache, assert p95 ≤ 250ms |

**WHY GOOD:** Problem Statement cites the design document and defines scope boundaries explicitly. Every requirement describes observable behaviour, not implementation. Requirement IDs enable traceability. Non-functional requirements have measurable thresholds with specific conditions. Constraints include scope exclusions and interface stability guarantees. Every acceptance criterion is a testable assertion with given/when/then structure. The Verification Plan maps each criterion to a specific test strategy. A reviewer can verify every claim; an implementer cannot misinterpret the requirements.

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