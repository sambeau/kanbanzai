---
name: write-dev-plan
description:
  expert: "Implementation plan authoring producing a structured 5-section
    dev-plan with scope tracing to specification, task breakdown with
    dependency ordering, risk assessment, and verification approach
    during the dev-planning stage"
  natural: "Write an implementation plan that breaks a specification into
    tasks with dependencies, estimates effort, assesses risks, and
    describes how to verify the spec is met"
triggers:
  - write an implementation plan
  - create a dev-plan for a feature
  - break a specification into tasks
  - author a development plan
  - produce a task breakdown
roles: [architect]
stage: dev-planning
constraint_level: high
---

## Vocabulary

- **task breakdown** — decomposition of a specification into discrete, assignable units of work, each with a clear deliverable and completion criterion
- **dependency graph** — directed acyclic graph of tasks where edges represent "must complete before" relationships, used to determine execution order
- **critical path** — the longest chain of dependent tasks that determines the minimum total duration of the plan
- **effort estimate** — assessment of work required for a task, expressed in story points or relative size, acknowledging uncertainty
- **vertical slice** — a task that delivers a thin but complete path through the system, from input to output, enabling early integration testing
- **parallelism** — tasks that have no dependency relationship and can execute concurrently by different agents or developers
- **blocking dependency** — a dependency where the downstream task cannot begin until the upstream task is complete, as opposed to a soft dependency where partial overlap is possible
- **risk mitigation** — a planned action that reduces the probability or impact of an identified risk before it materialises
- **verification approach** — the strategy for confirming that the implementation satisfies the acceptance criteria defined in the specification
- **scope reference** — an explicit citation of the parent specification that establishes traceability between the plan and what it implements
- **implementation order** — the sequence in which tasks should be executed, derived from the dependency graph and optimised for early feedback
- **risk probability** — the likelihood that an identified risk will occur, assessed relative to other risks in the plan
- **risk impact** — the consequence of a risk materialising, measured by how much rework or delay it would cause
- **deliverable** — the concrete output of a task: a file, a passing test, a configuration change, or a registered document
- **acceptance gate** — a verification checkpoint where task output is checked against specification requirements before downstream work begins
- **interface contract** — the agreed inputs, outputs, and invariants at a component boundary that tasks on either side must respect
- **task granularity** — the size of individual tasks, balanced between too large (unclear progress, hard to review) and too small (excessive coordination overhead)
- **risk register** — the collected set of identified risks with their probability, impact, and mitigation strategies
- **stub interface** — a minimal implementation of an interface contract that allows dependent tasks to proceed before the real implementation is complete
- **integration point** — a point in the plan where independently developed components must work together, requiring coordination and testing

## Anti-Patterns

### Monolithic Task
- **Detect:** A single task covers multiple components, multiple files, or more than one logical change
- **BECAUSE:** Large tasks hide progress, resist meaningful code review, and create merge conflicts — when a monolithic task fails review, the entire block of work requires rework rather than a targeted fix
- **Resolve:** Split until each task has a single deliverable that can be reviewed and verified independently. A task that touches more than three files or implements more than one acceptance criterion is a candidate for splitting.

### Missing Dependencies
- **Detect:** Tasks that logically depend on each other have no dependency edges in the graph, or the graph is absent entirely
- **BECAUSE:** Implicit ordering causes agents to start tasks before prerequisites are ready, producing integration failures that could have been avoided by sequencing work correctly
- **Resolve:** For every task, ask "what must exist before this task can start?" and record each answer as a dependency edge. Tasks with no dependencies are explicitly marked as parallelisable.

### Optimistic Estimation
- **Detect:** All tasks are estimated at the minimum possible effort with no risk buffer, or estimates are uniform across tasks of varying complexity
- **BECAUSE:** Underestimation cascades through the dependency graph — when an upstream task takes longer than estimated, every downstream task shifts, and the plan's credibility erodes
- **Resolve:** Estimate each task independently based on its specific complexity. Acknowledge uncertainty for tasks involving unfamiliar components or integration with external systems. Include at least one risk buffer task for the critical path.

### Specification Drift
- **Detect:** The task breakdown includes work that addresses requirements not present in the parent specification, or omits requirements that are present
- **BECAUSE:** A plan that diverges from its specification creates undocumented scope changes — extra work was never agreed upon, and missing work will surface as gaps during review
- **Resolve:** Trace every task back to a specific specification requirement. If a task cannot be traced, either the specification is incomplete (update it first) or the task is out of scope (remove it).

### Missing Verification
- **Detect:** The Verification Approach section is absent, generic ("run tests"), or does not map back to specific acceptance criteria
- **BECAUSE:** Without a concrete verification strategy, the plan has no way to confirm that implementation satisfies the specification — acceptance becomes subjective rather than checkable
- **Resolve:** For each acceptance criterion in the specification, state how it will be verified: which task produces the verification, what form it takes (test, inspection, demo), and what constitutes a pass.

### Serial-Only Planning
- **Detect:** Every task depends on the previous task, forming a single chain with no parallel paths
- **BECAUSE:** Fully serial plans waste available concurrency and extend the critical path unnecessarily — if two tasks have no data or interface dependency, serialising them adds latency without benefit
- **Resolve:** Review each dependency edge: does the downstream task genuinely require the upstream task's output? If not, remove the edge and mark both as parallelisable.

### Scope Without Spec Reference
- **Detect:** The Scope section does not cite the parent specification by path or document ID
- **BECAUSE:** Without traceability to the specification, the plan's scope is unanchored — changes to the specification cannot be propagated to the plan, and reviewers cannot verify that the plan covers the right requirements
- **Resolve:** The first paragraph of the Scope section must reference the parent specification explicitly, establishing the traceability chain.

## Procedure

### Step 1: Read and Understand the Specification

1. Read the parent specification fully. Identify all functional requirements, non-functional requirements, constraints, and acceptance criteria.
2. Note the acceptance criteria — these drive the verification approach.
3. IF the specification is ambiguous, incomplete, or contradictory → STOP. Report the ambiguity. Do not plan around assumptions — the cost of replanning is higher than the cost of clarifying the spec.
4. IF the specification scope is larger than expected → discuss with the human whether to split into multiple plans or proceed as one.

### Step 2: Decompose into Tasks

1. Break each requirement into one or more tasks with clear deliverables.
2. Each task should be independently reviewable and verifiable.
3. Prefer vertical slices over horizontal layers where possible — a vertical slice delivers early integration confidence.
4. For each task, identify: what it produces, what it depends on, and roughly how much effort it requires.
5. IF a requirement spans multiple components → create separate tasks for each component plus an integration task.

### Step 3: Build the Dependency Graph

1. For each task, determine which other tasks must complete first.
2. Identify the critical path — the longest dependency chain.
3. Identify parallelisable groups — tasks with no mutual dependencies.
4. IF the critical path is longer than necessary → look for false dependencies that can be broken with stub interfaces.
5. Verify the graph is acyclic. Circular dependencies indicate a decomposition problem.

### Step 4: Assess Risks

1. For each area of uncertainty, create a risk entry: what could go wrong, how likely, how impactful, what mitigates it.
2. Risks on the critical path deserve stronger mitigation because their impact cascades.
3. IF a risk is high-probability and high-impact → consider restructuring the plan to reduce exposure (e.g., tackle the risky task first to fail fast).

### Step 5: Define Verification Approach

1. Map each acceptance criterion from the specification to a verification method.
2. State which task produces each verification (test, inspection, demo).
3. Identify integration verification points where separately developed components must work together.

### Step 6: Draft and Validate

1. Call `now` to get the current date. Record the returned value — you will use it in the document header. Do not guess or invent a date.
2. Write all five required sections: Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach.
3. Run the validation script: `.kbz/skills/write-dev-plan/scripts/validate-dev-plan-structure.sh <path>`
4. Verify every task traces to a specification requirement.
5. Verify the Scope section references the parent specification.
6. IF validation fails → fix the structural issue → re-validate.

### Step 7: Register and Present

1. Register the document with `doc(action: register, type: "dev-plan")`.
   - For agent-authored dev-plans, registration and approval can be combined in one call:
     `doc(action: "register", type: "dev-plan", auto_approve: true, ...)` — the `auto_approve`
     flag is whitelisted for `dev-plan`, `research`, and `report` document types.
2. Present the plan to the human reviewer.
3. Be prepared to adjust task granularity, reorder priorities, or restructure dependencies based on feedback.

> **Note:** Approving a dev-plan does **not** automatically transition the feature to `developing`.
> After approval, explicitly call `entity(action: transition, id: "FEAT-xxx", status: "developing")`
> to advance the feature lifecycle before dispatching tasks.

## Output Format

Begin every implementation plan with a header table:

```
| Field  | Value                          |
|--------|--------------------------------|
| Date   | {value returned by `now`}      |
| Status | Draft                          |
| Author | {who is writing}               |
```

The implementation plan has exactly 5 required sections. Use these headings:

```
## Scope

State what this plan covers and what it does not. Reference the parent
specification by path or document ID in the first paragraph — this establishes
traceability.

Example: "This plan implements the requirements defined in
`work/spec/entity-caching.md` (DOC-...). It covers tasks T1–T8 below.
It does not cover monitoring or alerting, which are deferred to a
follow-up plan."

## Task Breakdown

For each task:

### Task N: Short Title

- **Description:** What this task produces.
- **Deliverable:** The concrete output (file, test, configuration).
- **Depends on:** Task IDs this task requires, or "None" if independent.
- **Effort:** Relative size (small / medium / large) or story points.
- **Spec requirement:** Which requirement(s) this task addresses.

## Dependency Graph

Show the dependency relationships. Use a text diagram or structured list:

  Task 1 (no dependencies)
  Task 2 (no dependencies)
  Task 3 → depends on Task 1
  Task 4 → depends on Task 1, Task 2
  Task 5 → depends on Task 3, Task 4

  Parallel groups: [Task 1, Task 2], [Task 3, Task 4]
  Critical path: Task 2 → Task 4 → Task 5

## Risk Assessment

For each identified risk:

### Risk: Short Description
- **Probability:** low / medium / high
- **Impact:** low / medium / high
- **Mitigation:** What action reduces the probability or impact.
- **Affected tasks:** Which tasks are exposed to this risk.

## Verification Approach

Map acceptance criteria to verification methods:

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-1: ...           | Unit test          | Task 3         |
| AC-2: ...           | Integration test   | Task 5         |
| AC-3: ...           | Manual inspection  | Task 4         |
```

## Examples

### BAD: Plan Without Dependencies or Verification

> ## Scope
> Implement the entity caching feature.
>
> ## Task Breakdown
> 1. Set up Redis
> 2. Add caching to the API
> 3. Write tests
> 4. Update docs
>
> ## Dependency Graph
> All tasks are sequential.
>
> ## Risk Assessment
> No significant risks identified.
>
> ## Verification Approach
> Run the test suite.

**WHY BAD:** Scope does not reference a specification — the plan is unanchored. Tasks lack deliverables, effort estimates, and spec traceability. The dependency graph is a single serial chain with no analysis of what is actually parallel. "No significant risks" suggests risks were not assessed. Verification is generic ("run the test suite") with no mapping to acceptance criteria.

### GOOD: Structured Plan with Traceability

> ## Scope
>
> This plan implements the entity caching specification defined in
> `work/spec/entity-caching.md` (DOC-042). It covers the cache layer
> implementation (REQ-001 through REQ-004) and cache invalidation
> (REQ-005, REQ-006). Monitoring (REQ-007) is deferred to a follow-up plan
> per discussion with the design owner.
>
> ## Task Breakdown
>
> ### Task 1: Define Cache Interface
> - **Description:** Create the `CacheReader` interface in the storage package
>   matching the contract from the design document.
> - **Deliverable:** `internal/storage/cache.go` with interface definition.
> - **Depends on:** None.
> - **Effort:** Small.
> - **Spec requirement:** REQ-001 (cache abstraction).
>
> ### Task 2: Implement Redis Cache Adapter
> - **Description:** Implement `CacheReader` backed by Redis, including
>   connection management and TTL configuration.
> - **Deliverable:** `internal/storage/redis_cache.go` and
>   `internal/storage/redis_cache_test.go`.
> - **Depends on:** Task 1.
> - **Effort:** Medium.
> - **Spec requirement:** REQ-002 (cache implementation).
>
> ### Task 3: Wire Cache into Entity Reader
> - **Description:** Modify `EntityReader` to use read-through caching via
>   the `CacheReader` interface.
> - **Deliverable:** Modified `internal/storage/entity_reader.go`.
> - **Depends on:** Task 1.
> - **Effort:** Medium.
> - **Spec requirement:** REQ-003 (read-through behaviour).
>
> ### Task 4: Implement Cache Invalidation
> - **Description:** Add write-through invalidation in `EntityWriter` to
>   clear cache entries on mutation.
> - **Deliverable:** Modified `internal/storage/entity_writer.go` and tests.
> - **Depends on:** Task 2, Task 3.
> - **Effort:** Medium.
> - **Spec requirement:** REQ-005, REQ-006 (invalidation on write).
>
> ### Task 5: Integration Tests
> - **Description:** End-to-end tests verifying read-through and
>   write-through behaviour across the storage layer.
> - **Deliverable:** `internal/storage/cache_integration_test.go`.
> - **Depends on:** Task 4.
> - **Effort:** Medium.
> - **Spec requirement:** AC-1 through AC-4.
>
> ## Dependency Graph
>
>     Task 1 (no dependencies)
>     Task 2 → depends on Task 1
>     Task 3 → depends on Task 1
>     Task 4 → depends on Task 2, Task 3
>     Task 5 → depends on Task 4
>
>     Parallel groups: [Task 2, Task 3]
>     Critical path: Task 1 → Task 2 → Task 4 → Task 5
>
> ## Risk Assessment
>
> ### Risk: Redis Connection Failures in CI
> - **Probability:** Medium.
> - **Impact:** Medium — blocks Task 2 and Task 5.
> - **Mitigation:** Use testcontainers or a mock Redis for unit tests.
>   Reserve real Redis for integration tests only.
> - **Affected tasks:** Task 2, Task 5.
>
> ### Risk: Cache Invalidation Race Condition
> - **Probability:** Low.
> - **Impact:** High — stale data served after writes.
> - **Mitigation:** Design invalidation to be idempotent. Add a test
>   that writes and reads concurrently to detect races.
> - **Affected tasks:** Task 4.
>
> ## Verification Approach
>
> | Acceptance Criterion | Method | Producing Task |
> |---|---|---|
> | AC-1: Cache hit returns entity | Unit test | Task 3 |
> | AC-2: Cache miss falls through to store | Unit test | Task 3 |
> | AC-3: Write invalidates cache entry | Unit test | Task 4 |
> | AC-4: End-to-end read/write/read cycle | Integration test | Task 5 |

**WHY GOOD:** Scope references the parent specification by path and document ID, establishing traceability. Each task has a concrete deliverable, a spec requirement anchor, and explicit dependencies. The dependency graph identifies a parallel group (Tasks 2 and 3) and the critical path. Risks are specific with mitigation strategies and affected tasks. Verification maps every acceptance criterion to a method and producing task — a reviewer can confirm complete coverage.

## Evaluation Criteria

1. Does the Scope section reference the parent specification by path or document ID? Weight: required.
2. Does every task in the Task Breakdown trace to a specific specification requirement? Weight: required.
3. Does the Dependency Graph identify at least one parallel group or explicitly state why all tasks are serial? Weight: required.
4. Does the Risk Assessment contain at least one risk with probability, impact, and mitigation? Weight: required.
5. Does the Verification Approach map every acceptance criterion to a verification method and producing task? Weight: high.
6. Are tasks granular enough that each has a single clear deliverable? Weight: high.
7. Does the plan identify the critical path through the dependency graph? Weight: high.
8. Are effort estimates provided for each task and differentiated by complexity? Weight: medium.

## Questions This Skill Answers

- How do I write an implementation plan for a Kanbanzai feature?
- What sections does a dev-plan require?
- How do I break a specification into tasks with dependencies?
- How do I build a dependency graph for a task breakdown?
- How do I assess risks in an implementation plan?
- How do I map acceptance criteria to verification methods?
- What level of task granularity is appropriate?
- When should I stop and ask for clarification during planning?
- How do I trace plan tasks back to specification requirements?
- How do I identify parallelisable tasks in a dependency graph?