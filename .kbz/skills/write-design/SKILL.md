---
name: write-design
description:
  expert: "Design document authoring producing a structured 4-section document
    with problem motivation, proposed approach, evaluated alternatives, and
    recorded architectural decisions during the designing stage"
  natural: "Write a design document that explains the problem, proposes a design,
    lists alternatives you considered, and records the decisions made"
triggers:
  - write a design document
  - draft a design for a feature
  - create a design document
  - author the design
  - produce a design for this feature
roles: [architect]
stage: designing
constraint_level: high
---

## Vocabulary

- **design decision** — a choice between alternatives that shapes the system's structure, recorded with rationale in the Decisions section
- **trade-off analysis** — explicit comparison of competing quality attributes (e.g., performance vs. maintainability) that drives a design decision
- **interface contract** — the set of guarantees a component makes to its consumers: inputs, outputs, error conditions, and invariants
- **component boundary** — the line separating one module or service from another, defined by what crosses it (data, control, dependencies)
- **design alternative** — a candidate approach that was evaluated and either chosen or rejected, documented in Alternatives Considered
- **architectural decision record (ADR)** — a lightweight record capturing the context, decision, and consequences of a single architectural choice
- **design rationale** — the reasoning that justifies a design decision, making the "why" explicit so future readers can evaluate whether it still holds
- **coupling** — the degree to which one component depends on the internals of another; lower coupling enables independent change
- **surface area** — the set of public interfaces or extension points a component exposes; larger surface area means more to maintain and test
- **invariant** — a property that must hold true throughout a component's lifetime, enforced by design rather than by convention
- **design constraint** — a limitation imposed by existing systems, requirements, or prior decisions that narrows the space of valid designs
- **composability** — the property of components that allows them to combine without modification to produce new behaviour
- **abstraction boundary** — the point at which implementation details are hidden from consumers, enabling independent evolution
- **extension point** — a deliberate mechanism for adding behaviour without modifying existing code (plugin interface, hook, callback)
- **migration path** — the plan for moving from the current state to the proposed design without disruption to running systems
- **failure mode** — a specific way a component can fail, along with its blast radius and recovery strategy
- **blast radius** — the scope of impact when a component fails or a change introduces a defect
- **design debt** — deliberate simplifications in the design that will need future work, documented so they are not forgotten
- **backward compatibility** — the guarantee that existing consumers continue to work when a component evolves
- **separation of concerns** — organising a system so each part addresses a distinct aspect, minimising cross-cutting dependencies

## Design Stance

**Design with ambition.** Always present the ambitious version of a design first. If there are genuine reasons to simplify — scope constraints, timeline pressure, missing infrastructure — enumerate them explicitly and let the human decide. Difficulty alone is not a reason to choose the weaker design alternative. The expedient version is a fallback, not a default.

**Human/agent role contract.** The human is the Design Manager — they own design decisions, make the final call, and approve. The agent is the Senior Designer — proposes designs, drafts documents, conducts research, presents design alternatives, and makes recommendations. The agent cannot make final design decisions or approve its own work. When the agent disagrees with a decision, it states its design rationale clearly, once. If the human decides otherwise, the agent documents the decision and moves on.

**Design is iterative.** There is no single right path. Stages can be revisited. A design that seemed complete can turn out to need revision after new design constraints surface or trade-off analysis reveals a better approach. That is normal and not a failure.

## Anti-Patterns

### Missing Alternatives
- **Detect:** The Alternatives Considered section is absent, empty, or contains only the chosen approach
- **BECAUSE:** Without evaluated alternatives, reviewers cannot assess whether the design space was adequately explored — the chosen approach may not be the best fit, and the reasoning for rejecting alternatives is lost
- **Resolve:** Document at least two design alternatives with trade-offs, even if one is "do nothing / status quo"

### Decision Without Rationale
- **Detect:** The Decisions section lists choices without explaining why each was made
- **BECAUSE:** A decision without rationale is indistinguishable from an arbitrary choice — future readers cannot evaluate whether the reasoning still holds when context changes
- **Resolve:** Every decision entry must include the context that led to it, the trade-off that resolved it, and the consequences that follow

### Premature Detail
- **Detect:** The Design section specifies implementation details (function signatures, database schemas, wire formats) before the problem and design constraints are established
- **BECAUSE:** Implementation detail in a design document couples the design to a specific implementation, making it harder to evaluate the approach on its own terms and constraining the specification unnecessarily
- **Resolve:** Keep the Design section at the level of components, boundaries, and interactions — leave implementation detail for the specification and dev-plan

### Solution Without Problem
- **Detect:** The Problem and Motivation section is absent or perfunctory (one sentence restating the feature title)
- **BECAUSE:** Without a clear problem statement, reviewers cannot evaluate whether the design solves the right problem — a well-designed solution to the wrong problem wastes effort and creates design debt
- **Resolve:** State the problem independently of the solution: what is broken, what is missing, who is affected, and what happens if nothing changes

### Unresolved Open Questions
- **Detect:** The document contains open questions (marked with "TBD", "TODO", or "open question") but is presented as ready for approval
- **BECAUSE:** Open design-level questions that survive into specification create cascading ambiguity — the spec author guesses, the implementer guesses differently, and the reviewer finds a gap
- **Resolve:** Resolve all design-level questions before requesting approval; implementation-level questions may remain open and are noted explicitly

### Scope Creep in Design
- **Detect:** The Design section addresses capabilities not mentioned in the Problem and Motivation section
- **BECAUSE:** Designing beyond the stated problem introduces undiscussed complexity — the additional scope has not been evaluated for cost, risk, or priority
- **Resolve:** If additional scope is genuinely needed, update Problem and Motivation first to establish the justification, then design for it

### Missing Failure Analysis
- **Detect:** The Design section describes only the happy path with no mention of failure modes, error handling, or recovery strategies
- **BECAUSE:** A design that only works when everything goes right is incomplete — failure modes discovered during implementation are more expensive to address than those anticipated during design
- **Resolve:** For each component boundary and external interaction, identify at least one failure mode and describe its handling strategy

## Risk Escalation

Three tiers for surfacing technical risk during design:

- **Minor concern** — mention once in discussion; note in the design document if relevant to a design decision. No further action required.
- **Significant risk** — raise clearly with explicit trade-off analysis. If the human moves on without acknowledging, repeat the concern. A significant risk that is silently ignored is a design gap.
- **Security or data-integrity risk** — do not proceed without explicit acknowledgment from the human. Repeat until acknowledged. These risks affect system invariants and cannot be accepted implicitly.

If the human acknowledges a risk and decides to proceed anyway, accept the decision. Document the risk, the decision, and the design rationale in the Decisions section so the trade-off is visible to future readers.

## Procedure

### Step 1: Establish Context

1. Read the agreed scope and any preceding research or planning discussion.
2. Identify the problem to be solved and the constraints that bound the design space.
3. IF the scope is unclear or incomplete → STOP. Report the ambiguity. Do not infer scope — the cost of designing for the wrong problem is high.
4. IF the scope suggests multiple independent features → flag this for the human. The scope may need splitting before design begins.

### Step 2: Explore the Design Space

1. Identify at least two candidate approaches that could solve the stated problem.
2. For each candidate, note the trade-offs: what it makes easy, what it makes hard, what risks it introduces.
3. IF an existing system, interface, or prior decision constrains the design → document it as a design constraint.
4. Select a recommended approach with explicit reasoning.

### Step 3: Draft the Document

1. Call `now` to get the current date. Record the returned value — you will use it in the document header's `Date` field. Do not guess or invent a date.
2. Write all four required sections in order: Problem and Motivation, Design, Alternatives Considered, Decisions.
3. The Design section describes the recommended approach at the level of components, boundaries, and interactions.
4. The Alternatives Considered section includes every candidate from Step 2, with trade-offs and the reason each was chosen or rejected.
5. The Decisions section records each architectural choice with its design rationale.
6. IF any aspect of the design is uncertain → mark it as an open question rather than guessing.

### Step 4: Self-Validate

1. Run the validation script: `.kbz/skills/write-design/scripts/validate-design-structure.sh <path>`
2. Verify that every decision has a design rationale.
3. Verify that Alternatives Considered contains at least two design alternatives.
4. Hold the design against six qualities: simplicity, minimalism, completeness, composability, honesty, and durability. See [references/design-quality.md](references/design-quality.md) for full definitions.
5. IF validation fails → fix the structural issue → re-validate.

### Step 5: Present for Review

1. Register the document with `doc(action: register, type: "design")`.
2. Present the draft to the human reviewer.
3. Open questions are acceptable in a draft — flag them explicitly.
4. An approved design must have zero unresolved design-level questions.

## Output Format

Begin every design document with a header table:

```
| Field  | Value                          |
|--------|--------------------------------|
| Date   | {value returned by `now`}      |
| Status | Draft                          |
| Author | {who is writing}               |
```

The document then has exactly 4 required sections. Use these headings verbatim:

```
## Problem and Motivation

State the problem this design addresses. What is broken, missing, or inadequate?
Who is affected? What happens if nothing changes?

Do not describe the solution here — this section must stand on its own as a
problem statement.

## Design

Describe the proposed approach at the level of components, boundaries, and
interactions. Include:
- How components are organised and what responsibilities each has
- Key interfaces and data flows between components
- How the design addresses the constraints identified in the problem
- Failure modes and their handling strategies

Do not include implementation-level detail (function signatures, database
schemas) — that belongs in the specification.

## Alternatives Considered

For each design alternative evaluated:
- Brief description of the approach
- Trade-offs (what it makes easier, what it makes harder)
- Why it was chosen or rejected

Include at least two alternatives. "Do nothing / status quo" is a valid alternative.

## Decisions

Each decision entry:
- **Decision:** what was decided
- **Context:** what circumstances led to this decision
- **Rationale:** why this choice was made over the alternatives
- **Consequences:** what follows from this decision (positive and negative)
```

## Examples

### BAD: Design Without Alternatives

> ## Problem and Motivation
> We need a caching layer for the API.
>
> ## Design
> Use Redis as a caching layer. Keys will be formatted as `entity:{id}`.
> TTL will be 5 minutes. We'll use the go-redis client library.
>
> ## Alternatives Considered
> Redis is the industry standard for caching.
>
> ## Decisions
> Use Redis.

**WHY BAD:** Problem statement is one sentence with no context on what is slow, who is affected, or what the impact is. The Design section jumps to implementation detail (specific library, key format) instead of describing component boundaries and interactions. Alternatives Considered does not list any design alternatives — it just justifies the chosen approach. Decisions has no design rationale, context, or consequences.

### GOOD: Structured Design with Trade-Off Analysis

> ## Problem and Motivation
>
> The entity listing endpoint (`GET /entities`) performs a full table scan on
> every request. At current growth rates, p95 latency will exceed the 200ms
> SLO within 8 weeks. The endpoint is called ~10k times/day by the dashboard
> and by CI pipelines checking entity status.
>
> Doing nothing means SLO violations that degrade dashboard responsiveness
> and slow CI feedback loops.
>
> ## Design
>
> Introduce a read-through cache between the API handler and the storage layer.
> The cache sits behind the existing `EntityReader` interface, so consumers
> are unaffected. Cache invalidation uses write-through: the `EntityWriter`
> clears relevant cache entries on mutation.
>
> The component boundary is the `EntityReader` interface. No component outside
> the storage package interacts with the cache directly. This keeps the blast
> radius of cache-related bugs contained to the storage layer.
>
> **Failure mode:** If the cache is unavailable, the storage layer falls back
> to direct database reads. Latency degrades but functionality is preserved.
>
> ## Alternatives Considered
>
> **A. In-process LRU cache.** Simple to implement, no infrastructure dependency.
> Trade-off: cache is per-process, so multiple instances serve stale data after
> writes to other instances. Rejected because the system runs multiple replicas.
>
> **B. External shared cache.** Shared across replicas, supports TTL and eviction
> policies. Trade-off: adds an infrastructure dependency and a network hop.
> Chosen because cross-replica consistency is required.
>
> **C. Database-level query caching.** Minimal code change. Trade-off: no control
> over eviction policy; cache is invalidated on any table write, not just relevant
> entities. Rejected because invalidation granularity is too coarse.
>
> ## Decisions
>
> **Decision:** Use an external shared cache behind the `EntityReader` interface.
> **Context:** Multiple replicas serve the same endpoint; in-process caching
> produces inconsistent reads across replicas.
> **Rationale:** Cross-replica consistency requires a shared cache. Placing it
> behind the existing interface contract minimises blast radius and avoids
> coupling the cache implementation to consumers.
> **Consequences:** Adds an infrastructure dependency. Requires cache health
> monitoring. The `EntityReader` interface remains unchanged, so no consumer
> code changes are needed.

**WHY GOOD:** Problem statement quantifies the issue (p95 latency, growth rate, call volume) and states the consequence of inaction. Design describes components and boundaries without implementation detail. Three design alternatives with explicit trade-offs and clear accept/reject reasoning. Failure mode is identified with a recovery strategy. Decision entry has full context, design rationale, and consequences — a future reader can evaluate whether the reasoning still holds.

## Operational Guidance

**Draft lifecycle.** During design, documents are in draft status. Drafts may contain design alternatives and open questions. Keep the document as an honest, up-to-date reflection of where the design has reached — do not present a draft as more settled than it is.

**Design splitting.** If scope is larger than a single feature, flag it and step back to planning. Signs that splitting is needed: different sections feel like separate products, parts could be implemented independently with no interface contract between them, or the resulting specification would be unmanageably large.

**Gotchas.**

- Register the document after creation — call `doc(action: "register")` immediately. An unregistered design document is invisible to the workflow.
- Do not approve a design document with unresolved design-level questions. Implementation-level questions may remain open.
- Do not edit an approved document — supersede it with `doc(action: "supersede")` instead. Approved documents are immutable records.
- If `doc(action: "approve")` fails due to content hash drift, call `doc(action: "refresh")` first to re-sync.

**Next steps after design.** Use `work/templates/specification-prompt-template.md` for the specification. See the `kanbanzai-documents` skill for the full registration and approval procedure.

## Evaluation Criteria

1. Does the Problem and Motivation section state the problem independently of the solution? Weight: required.
2. Does the Design section describe components, boundaries, and interactions without implementation-level detail? Weight: required.
3. Does Alternatives Considered contain at least two design alternatives with trade-offs? Weight: required.
4. Does every entry in Decisions include design rationale explaining why the choice was made? Weight: required.
5. Are failure modes or error handling strategies addressed in the Design section? Weight: high.
6. Is the design scoped to the stated problem without scope creep? Weight: high.
7. Can a spec author write a complete specification from this design without guessing intent? Weight: high.

## Questions This Skill Answers

- How do I write a design document for a Kanbanzai feature?
- What sections does a design document require?
- How do I structure the Alternatives Considered section?
- What level of detail belongs in a design document vs. a specification?
- How do I record architectural decisions with rationale?
- When should I stop and ask for clarification during design?
- What makes a design document ready for approval?
- How do I handle open questions in a design draft?