---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: decompose-feature
description:
  expert: "Feature-to-task decomposition procedure using the decompose tool
    with a 5-point validate → fix → re-validate loop ensuring dependency
    correctness, single-agent sizing, and coverage completeness"
  natural: "Breaks a feature into implementable tasks — proposes a
    decomposition, then validates it in a loop until every task has a
    description, dependencies are sound, sizing is right, and nothing
    is missing"
triggers:
  - decompose a feature into tasks
  - break down feature for implementation
  - create task breakdown
  - plan feature tasks
  - generate implementation plan from feature
roles: [architect]
stage: dev-planning
constraint_level: low
---

## Vocabulary

- **parent batch** — the batch entity that owns the feature being decomposed; tasks are created under the feature but the batch provides the execution context and dependency ordering across features
- **dependency edge** — an explicit declaration that task B requires task A's output before it can start
- **dependency declaration** — recording a dependency in the task's `depends_on` field so the orchestrator respects ordering
- **circular dependency** — two or more tasks that directly or transitively depend on each other, making dispatch impossible
- **topological sort** — ordering tasks so every task appears after all its dependencies; the orchestrator's dispatch sequence
- **dependency graph** — the directed graph of task-to-task dependency edges within a feature
- **vertical slice** — a task that delivers a thin, end-to-end piece of functionality across layers rather than a single-layer chunk
- **horizontal slice** — a task scoped to one layer (e.g. "all database migrations") that delivers no end-to-end behaviour on its own
- **integration surface** — the boundary where one task's output becomes another task's input; must be tested
- **feature boundary** — the outer scope of the feature; tasks should not extend beyond it
- **single-agent scope** — a task sized so one agent can complete it in one session without losing context
- **over-decomposition** — splitting work into tasks so small that coordination overhead exceeds implementation effort
- **task granularity** — the level of detail in the breakdown; too coarse leaves agents without direction, too fine creates dispatch overhead
- **atomic task** — a task that delivers a complete, testable unit of work with clear acceptance criteria
- **orphan task** — a task with no dependency connections to any other task in the feature; may indicate a gap or misplacement
- **coverage gap** — a feature requirement that has no corresponding task in the decomposition
- **missing integration task** — absence of a task that verifies independently-built components work together at their integration surfaces
- **missing test task** — absence of a task that creates or updates tests for the feature's acceptance criteria
- **gap scan** — a systematic check for feature requirements that lack corresponding tasks
- **task description** — the summary and acceptance criteria that tell an implementing agent what to build and how to verify it
- **decomposition proposal** — the initial breakdown output from the `decompose` tool, before validation
- **re-validation pass** — a repeat of the 5-point validation after fixing issues found in a prior pass
- **dispatch readiness** — the state where a decomposition's dependency graph is valid and every task is implementable

## Anti-Patterns

### Over-Decomposition
- **Detect:** More than 10 tasks for a feature, or tasks that take less than a few minutes of implementation each, or tasks that are just individual function writes
- **BECAUSE:** Coordination overhead grows with task count. Each task requires dispatch, monitoring, context assembly, and completion handling. When tasks are trivially small, the orchestration cost exceeds the implementation cost, and agents spend more time reading context packets than writing code
- **Resolve:** Merge related micro-tasks into vertical slices. A good task delivers a testable behaviour, not a single code change. If two tasks always need the same context to implement, they are one task

### Circular Dependencies
- **Detect:** Task A depends on Task B which depends on Task A (directly or through a chain). The `decompose` tool's validation catches direct cycles, but transitive cycles across longer chains may survive
- **BECAUSE:** Circular dependencies make the dependency graph unsortable — no valid dispatch order exists. The orchestrator will deadlock, with every task waiting for another to finish first
- **Resolve:** Identify the cycle and break it by extracting the shared concern into a new task that both depend on, or by removing the dependency that is weakest (often one direction is a "nice to have" rather than a hard requirement)

### Partial-Task-Completion Dependency
- **Detect:** A task depends on another task completing only *part* of its work — e.g. "Task B can start once Task A has written the schema, before Task A's tests are done"
- **BECAUSE:** The entity model only supports full-completion dependencies. A task is either done or it isn't — there is no "phase complete" signal. Declaring a dependency on partial completion means the orchestrator will either dispatch too early (if the dependency is omitted) or block unnecessarily (if it is declared on the full task)
- **Resolve:** Split the prerequisite task into two sequential tasks (Phase A and Phase B) and declare the dependency on Phase A. Make the split explicit in the task names and descriptions so the dependency is self-documenting

### Missing Integration Tasks
- **Detect:** Multiple tasks produce components that must work together (e.g. an API handler and its client, a storage layer and its consumer), but no task verifies they integrate correctly
- **BECAUSE:** Individual tasks verify their own acceptance criteria in isolation. Without an integration task, components may individually pass tests but fail when combined — different assumptions about interfaces, field names, error formats, or ordering
- **Resolve:** For every integration surface between tasks, add a task that wires the components together and verifies end-to-end behaviour. This task depends on all the component tasks it integrates

### Implicit Dependencies
- **Detect:** A task's description references output from another task ("use the schema from Task 3") but the dependency edge is not declared in `depends_on`
- **BECAUSE:** The orchestrator dispatches tasks based on declared dependencies, not description text. An undeclared dependency will be dispatched in parallel with or before its prerequisite, causing the implementing agent to work with missing or incomplete inputs
- **Resolve:** Scan every task description for references to other tasks' outputs. For each reference, add an explicit dependency declaration. If the reference is informational rather than a hard prerequisite, rewrite the description to remove the coupling

### Monolith Task
- **Detect:** A single task covers multiple acceptance criteria across different concerns, or its description spans more than one screen of context, or it touches files in many unrelated packages
- **BECAUSE:** Large tasks exceed single-agent scope. The agent loses context partway through, misses later acceptance criteria, and produces inconsistent implementations. Large tasks also block parallel dispatch — nothing that depends on them can start until the entire monolith completes
- **Resolve:** Split along vertical slices. Each resulting task should address one concern end-to-end with a small, focused set of acceptance criteria. A good heuristic: if a task has more than 4-5 acceptance criteria, it is likely a monolith

### Description-Free Tasks
- **Detect:** A task has a title but an empty or single-sentence description with no acceptance criteria
- **BECAUSE:** The implementing agent receives the description as its primary guidance. Without acceptance criteria, the agent must infer what "done" means — and its inference will differ from the architect's intent. The reviewer then has no baseline to evaluate against
- **Resolve:** Every task needs a description that states what the task produces, how to verify it, and what its acceptance criteria are. If you cannot write the description, the task is not yet understood well enough to decompose

### Test Gap
- **Detect:** The decomposition includes implementation tasks but no tasks dedicated to testing, or the existing test coverage for modified code is not addressed by any task
- **BECAUSE:** Implementation tasks often include unit tests alongside code, but integration tests, end-to-end tests, and test infrastructure updates need their own tasks. Without explicit test tasks, testing becomes an afterthought that gets skipped under time pressure
- **Resolve:** Add test tasks for integration verification, acceptance-criteria-level testing, and any test infrastructure that needs updating. Test tasks typically depend on the implementation tasks they verify

## Procedure

### Phase 1: Read Feature Context

1. Read the feature's specification and design documents. Understand the requirements, acceptance criteria, and any architectural constraints.
2. Note integration surfaces — where this feature connects to existing code or other features.
3. IF the spec is ambiguous, incomplete, or contradictory for any aspect of the decomposition → STOP. Report the ambiguity. Do not infer intent. Decomposing from a misunderstood spec produces tasks that solve the wrong problem.

### Phase 2: Generate Initial Decomposition

1. Use the `decompose` tool with action `propose` to generate an initial decomposition proposal.
2. Aim for vertical slices — each task delivers end-to-end behaviour across layers, not a horizontal layer slice.
3. Consider dependency ordering as you propose: which tasks produce interfaces that other tasks consume?
4. Include integration tasks where components built by different tasks must work together.
5. Include test tasks where acceptance-criteria-level verification requires dedicated work beyond unit tests.

**⚠ If the proposal is broken:** If `decompose propose` returns a proposal with empty task names, wrong task count, or a structure that cannot be corrected by adjusting input context, do not call `decompose apply`. Proceed to the Manual Fallback in Phase 4.

### Phase 3: Validate (The 5-Point Loop)

This is the most important phase. Decomposition quality is the strongest predictor of implementation success — a flawed task graph produces coordination failures, wasted work, and integration defects downstream. Run all five checks. If any check fails, fix the issue and re-validate from check 1.

**Check 1 — Every task has a clear, non-empty description:**
Verify each task has a description that states what it produces, lists acceptance criteria, and provides enough context for an implementing agent to start work without guessing.
IF a task lacks a description or has only a title → write the description before proceeding.

**Check 2 — Dependencies between tasks are explicitly declared:**
For each task, check whether it references another task's output. Every such reference must have a corresponding dependency declaration. Scan descriptions for phrases like "using the X from Task N" or "after the schema is created" — these signal undeclared dependencies.
IF an implicit dependency is found → add the dependency edge.

**Check 3 — Each task is sized for single-agent completion:**
Review each task's scope. Can one agent complete it in a single session with the context packet it will receive? Tasks with more than 4-5 acceptance criteria, tasks touching many unrelated files, or tasks requiring deep context across multiple subsystems are too large.
IF a task exceeds single-agent scope → split it into vertical slices.

**Check 4 — No circular dependencies exist:**
Trace the dependency graph. Verify that a topological sort is possible — every task can be ordered after all its dependencies. Check transitively, not just direct edges.
IF a cycle exists → break it by extracting the shared concern or removing the weakest dependency direction.

**Check 5 — Integration and test tasks are present:**
For every integration surface between tasks, verify an integration task exists. For every cluster of acceptance criteria that requires more than unit-level testing, verify a test task exists.
IF integration or test tasks are missing → add them with appropriate dependencies.

**Re-validation:** After fixing any issue, return to Check 1 and run all five checks again. A fix in one area (e.g. splitting a monolith) may introduce new issues in another (e.g. missing dependency declarations on the new tasks). The loop terminates when all five checks pass in a single pass.

### Phase 4: Apply the Decomposition

> **Note:** Decomposition occurs during the `dev-planning` stage. The feature does **not** need
> to be in `developing` status first — decompose while the feature is still in `dev-planning`.

1. Once all five checks pass, use the `decompose` tool with action `apply` to create the tasks.
2. Verify the created tasks match your validated proposal — correct descriptions, correct dependencies.

#### Manual Fallback — when `decompose propose` is wrong or crashes

Use `entity(action: "create", type: "task")` directly. Required fields: `name`, `summary`, `parent_feature`. Optional but recommended: `depends_on` (array of TASK-... IDs for dependency wiring).

Create tasks in dependency order so IDs are available for `depends_on` references before they are needed.

Minimal wiring example:
```
entity(action: "create", type: "task",
  name: "Task 2: Do the second thing",
  summary: "...",
  parent_feature: "FEAT-xxx",
  depends_on: ["TASK-01KPQ..."])   # ID of Task 1 created first
```

Verify the created tasks with `status(id: "FEAT-xxx")` before proceeding.

The manual fallback is the escape hatch — `decompose propose` + `decompose apply` remains the primary path.

### Phase 5: Record Observations

1. Note any spec ambiguities discovered during decomposition — these are valuable signals for the spec author.
2. Note any architectural concerns — integration surfaces that seem fragile, components that may need interface agreements.
3. IF the feature is large enough that decomposition required significant analytical work, consider contributing a knowledge entry about the decomposition rationale.

## Output Format

The primary output is the set of created tasks (via the `decompose` tool). Additionally, record a summary of the decomposition rationale:

```
Feature: FEAT-xxx
Tasks created: N

Decomposition rationale:
- [Why this slicing approach was chosen]
- [Key dependency chains and their reasoning]
- [Integration surfaces identified]

Validation passes: [number of validate → fix cycles before all 5 checks passed]

Observations:
- [Spec ambiguities found, if any]
- [Architectural concerns, if any]
- [Assumptions made, if any]
```

## Examples

### BAD: Horizontal slicing with no integration

```
Feature: FEAT-088 — Add user notification preferences

Task 1: Create database migration for preferences table
Task 2: Add preferences API endpoints
Task 3: Build preferences UI components
Task 4: Write notification filtering logic
```

WHY BAD: Four horizontal slices, each covering one layer. No dependency declarations — can Task 2 be implemented without Task 1's schema? No integration task verifies that the API actually reads from the database or that the UI calls the API correctly. No test tasks. Task descriptions are titles only — an implementing agent receives no acceptance criteria and must guess the table schema, endpoint paths, and UI behaviour.

### BAD: Over-decomposed with implicit dependencies

```
Feature: FEAT-091 — Add webhook retry logic

Task 1: Add retry count column to webhooks table
Task 2: Create RetryPolicy struct
Task 3: Write calculateBackoff function
Task 4: Write shouldRetry function
Task 5: Add retry loop to webhook dispatcher
Task 6: Write test for calculateBackoff
Task 7: Write test for shouldRetry
Task 8: Write test for retry loop
Task 9: Add retry metrics counter
Task 10: Update webhook status on final failure
Task 11: Write integration test for full retry flow
```

WHY BAD: 11 tasks for a focused feature. Tasks 2-4 are micro-tasks that would each take minutes — the coordination overhead of dispatching, monitoring, and integrating them exceeds the implementation cost. Tasks 6-8 are test afterthoughts separated from the code they test. Task 5 implicitly depends on Tasks 1-4 but no dependency edges are declared. Tasks 2-4 have no acceptance criteria beyond their title.

### GOOD: Vertical slices with validated dependencies

```
Feature: FEAT-091 — Add webhook retry logic

Task 1: Implement retry mechanism for webhook delivery
  Description: Add retry logic to the webhook dispatcher. When a delivery
  fails, retry up to 3 times with exponential backoff (1s, 2s, 4s) per
  spec §4.2. Add retry_count column to webhooks table. Record each
  attempt in the delivery log per spec §4.3.
  Acceptance criteria:
  - AC-1: Failed deliveries retry up to 3 times
  - AC-2: Backoff follows 1s/2s/4s exponential pattern
  - AC-3: Each attempt is recorded in the delivery log
  - AC-4: Webhook status set to 'failed' after exhausting retries
  Tests: Unit tests for retry logic, backoff timing, and status transitions.
  Depends on: (none — first task)

Task 2: Add retry observability and metrics
  Description: Add metrics counters for retry attempts, successes, and
  exhaustions. Expose via the existing metrics endpoint.
  Acceptance criteria:
  - AC-1: retry_attempt counter incremented on each retry
  - AC-2: retry_success counter incremented on successful retry
  - AC-3: retry_exhausted counter incremented when retries exhausted
  Depends on: Task 1

Task 3: Integration test for retry flow
  Description: End-to-end test that triggers a webhook delivery failure,
  verifies retries occur with correct timing, and confirms delivery log
  entries and final status.
  Acceptance criteria:
  - AC-1: Test covers successful retry on second attempt
  - AC-2: Test covers retry exhaustion after 3 failures
  - AC-3: Test verifies delivery log has one entry per attempt
  Depends on: Task 1, Task 2

Validation: 2 passes. First pass found Task 2 missing dependency on Task 1.
Fixed and re-validated — all 5 checks passed on second pass.
```

WHY GOOD: Three tasks instead of eleven. Task 1 is a vertical slice — it delivers the complete retry mechanism end-to-end including the database change, logic, and unit tests. Task 2 adds observability as a separate concern with a clear dependency. Task 3 is an explicit integration task that depends on both. Every task has a description with acceptance criteria and spec citations. Dependencies are declared. Each task is single-agent scope. The validation loop caught a missing dependency and fixed it.

## Evaluation Criteria

1. Does every task have a non-empty description with acceptance criteria? Weight: required.
2. Are all dependencies explicitly declared (no implicit references to other tasks' outputs)? Weight: required.
3. Is each task sized for single-agent completion? Weight: required.
4. Is the dependency graph acyclic (topologically sortable)? Weight: required.
5. Are integration tasks present for every integration surface? Weight: high.
6. Are test tasks present where acceptance-criteria-level testing requires dedicated work? Weight: high.
7. Does the decomposition use vertical slices rather than horizontal layers? Weight: high.
8. Was the 5-point validation loop run to completion (all checks pass in one pass)? Weight: required.
9. Are coverage gaps identified and addressed (every feature requirement maps to a task)? Weight: high.
10. Is the task count proportionate to the feature's complexity (no over-decomposition)? Weight: moderate.

## Questions This Skill Answers

- How do I break a feature into implementable tasks?
- What makes a good task boundary?
- How do I validate a decomposition before committing to it?
- What should I check for in task dependencies?
- How do I detect circular dependencies in a task graph?
- When is a feature over-decomposed?
- What integration tasks does my decomposition need?
- How do I write task descriptions that give implementing agents enough context?
- What does the validate → fix → re-validate loop look like in practice?
- How do I know when a decomposition is ready for dispatch?