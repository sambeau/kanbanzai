---
name: implement-task
description:
  expert: "Structured task execution procedure for individual implementation
    work: spec-driven incremental development with acceptance criteria
    verification and test authoring"
  natural: "Guides you through implementing a single task — read what's
    required, build it, test it, verify it matches the spec"
triggers:
  - implement a task
  - execute implementation work
  - build a feature task
  - write code for a task
roles: [implementer, implementer-go]
stage: developing
constraint_level: medium
---

## Vocabulary

- **acceptance criterion** — a numbered, testable condition from the spec that defines "done" for a task
- **spec conformance** — degree to which implementation matches the spec's stated requirements, not inferred intent
- **spec citation** — explicit reference to a numbered spec requirement that drives an implementation choice
- **implementation choice** — a decision between two or more valid approaches; must cite a spec requirement as justification
- **assumption flag** — a marker on a decision made without spec backing; flagged for human review
- **scope boundary** — the set of files and behaviours this task is authorised to change; anything outside is scope creep
- **file scope** — the specific files this task is expected to create or modify; changes outside require justification
- **code path** — a distinct execution route through the implementation; each needs test coverage
- **test coverage** — the proportion of code paths exercised by tests, including happy path, error paths, and edge cases
- **happy path** — the expected success flow through the implementation
- **error path** — a flow triggered by invalid input, missing data, or failure conditions
- **edge case** — a boundary condition or unusual input within spec scope but outside the happy path
- **regression check** — confirmation that pre-existing tests still pass after changes
- **verification step** — a concrete action confirming an acceptance criterion is met
- **incremental commit** — a commit at a logical checkpoint within a task, before starting a different concern
- **context packet** — the assembled spec sections, knowledge entries, and file paths delivered by `next`
- **task completion summary** — the structured record of what was accomplished, how, and what was verified
- **side effect** — a behaviour change outside the task's stated scope caused by the implementation

## Anti-Patterns

### Scope Creep
- **Detect:** Implementation modifies files or behaviours outside the task's file scope or acceptance criteria
- **BECAUSE:** Each task has a defined scope boundary. Changes outside it risk conflicts with parallel tasks and introduce untested behaviours that no acceptance criterion verifies
- **Resolve:** Check every file modification against the task's file scope. If a change outside scope is genuinely needed, flag it as an assumption and note the justification

### Untested Code Path
- **Detect:** A code path (conditional branch, error handler, fallback) is added without a corresponding test
- **BECAUSE:** Untested code paths are invisible to verification — they may behave correctly now but regress silently. Every code path exists because of a requirement, and requirements need verification
- **Resolve:** For each code path added, write a test that exercises it. If a path is unreachable, remove it

### Spec Deviation
- **Detect:** Implementation behaviour differs from what the acceptance criterion specifies — different names, codes, ordering, or semantics
- **BECAUSE:** The spec is the contract. Deviations create integration failures when other tasks depend on the specified behaviour. An improvement that contradicts the spec is a defect
- **Resolve:** Re-read the acceptance criterion and match it exactly. If the spec appears wrong, STOP and report the issue — do not silently correct it

### Assumption Without Flag
- **Detect:** An implementation choice between alternatives is made without citing a spec requirement, and no assumption flag is recorded
- **BECAUSE:** Undocumented assumptions become invisible design decisions. Reviewers cannot distinguish intentional choices from accidental ones, and incorrect assumptions compound through dependent tasks
- **Resolve:** For every choice not covered by the spec, add an assumption flag in the task completion summary noting what was assumed and why

### Big-Bang Implementation
- **Detect:** All changes are made in a single pass with no incremental commits or intermediate verification
- **BECAUSE:** Large uncommitted changesets are fragile — a mistake late in the process can invalidate early work, and the implementation becomes difficult to review or recover from
- **Resolve:** Commit after each logical unit of work. Verify acceptance criteria incrementally

### Test Afterthought
- **Detect:** Tests are written only after all implementation is complete, covering mainly the happy path
- **BECAUSE:** Tests written after the fact verify what was built rather than what was specified. They miss error paths and edge cases because the implementer's mental model is anchored to the working code
- **Resolve:** Write or update tests alongside implementation. When adding an error path, write its test before moving on

### Unclaimed Task
- **Detect:** Implementation begins without calling `next` to claim the task and receive the context packet
- **BECAUSE:** The context packet contains curated spec sections, knowledge entries, and file paths. Skipping it means re-discovering what the system already assembled and missing knowledge that prevents known pitfalls
- **Resolve:** Claim the task with `next(id: "TASK-xxx")` before writing any code

### Unreported Flaky Test

- **Detect:** Agent observes a test that fails then passes on retry (without any code change) and marks the task done without filing a BUG entity.
- **BECAUSE:** Intermittent test failures indicate non-determinism — a race condition, timing dependency, or environmental assumption. Not recording them means future agents encounter the same failure with no prior context, re-investigate from scratch, and potentially make the same "probably fine" call. The cumulative cost far exceeds the cost of filing one BUG entity.
- **Resolve:** File a BUG entity for every observed intermittent failure before calling `finish`. Include the test name, the failure message, and the conditions under which it was observed.

## Worktree File Editing

> **Note:** The `edit_file` tool supports worktrees via its `entity_id` parameter.
> When `entity_id` is provided, `edit_file` resolves the worktree path and applies
> edits there — mirroring `write_file`'s behaviour. `write_file(entity_id: ...)`
> remains the primary recommendation for worktree file creation.

When implementing tasks assigned to a worktree, use `write_file` with the
`entity_id` parameter. This scopes the write to the worktree's directory
automatically, regardless of file type.

### Worktree files — use `write_file` (all file types)

Use `write_file(entity_id: ...)` for all files inside a worktree — Go source,
Markdown, YAML, or any other type:

```
write_file(
  entity_id: "FEAT-...",
  path: "path/to/file.go",
  content: "<full file content>"
)
```

The `entity_id` resolves to the worktree's root directory automatically. The
`path` is relative to that root.

### Non-worktree files — use `edit_file`

For files in the main project root (not inside a worktree), use `edit_file` as
normal.

## Checklist

```
Copy this checklist and track your progress:
- [ ] Claimed the task with `next(id: "TASK-xxx")`
- [ ] Confirmed whether this task runs inside a worktree — if yes, use `write_file(entity_id: ...)` for all file types; `edit_file(entity_id: ...)` is also supported for granular edits
- [ ] Read the context packet — spec sections, knowledge entries, file paths
- [ ] Called knowledge list with domain-relevant tags before writing any code
- [ ] Listed all acceptance criteria for this task
- [ ] Confirmed file scope — which files to create or modify
- [ ] Implemented changes incrementally with commits at logical checkpoints
- [ ] Cited spec requirements for non-trivial implementation choices
- [ ] Flagged assumptions not covered by the spec
- [ ] Wrote or updated tests for every code path
- [ ] Ran the full test suite — all tests pass including regression
- [ ] Verified each acceptance criterion is met
- [ ] If any test failed intermittently (passed on retry without code change), filed a BUG entity before proceeding
- [ ] Completed the task with `finish` including summary and verification
- [ ] Confirmed accurate and flagged inaccurate knowledge entries after task completion
```

## Procedure

### Phase 1: Read Spec and Acceptance Criteria

1. Claim the task with `next(id: "TASK-xxx")`. Read the full context packet.
1a. Call `knowledge(action: "list")` with tags derived from this task's feature area (e.g. the domain, the subsystem, the Go package being modified). Review all returned entries. Note any entry that describes a known pitfall, anti-pattern, or constraint relevant to this task. **BECAUSE** knowledge entries record hard-won discoveries from previous tasks and an agent that skips this step risks re-discovering the same problems from scratch.
2. List every acceptance criterion for this task explicitly.
3. Note the file scope — which files you are expected to create or modify.
4. IF any acceptance criterion is ambiguous, incomplete, or contradictory → STOP. Report the ambiguity. Do not infer intent. The cost of asking is low; the cost of guessing wrong compounds through review and rework.
5. IF the context packet is missing spec sections or file paths → STOP. Report missing context.

### Phase 2: Implement

1. Work through acceptance criteria in order. For each criterion, make the minimal changes needed to satisfy it.
2. When choosing between implementation alternatives, cite the spec requirement that drives the choice. If no requirement covers the decision, record it as an assumption flag.
3. Commit at logical checkpoints — after completing a coherent change, before starting a different concern.
4. Stay within the scope boundary. If you discover work needed outside scope, note it in the completion summary — do not do it.

### Phase 3: Write or Update Tests

1. For each code path added or modified, write a test that exercises it.
2. Cover the happy path, error paths, and edge cases identified from acceptance criteria.
3. IF an existing test breaks due to your changes, determine whether the test or the implementation is wrong. Fix the correct one.

### Phase 4: Verify

1. Run the full test suite. All tests must pass, including pre-existing tests (regression check).
   - If any test fails intermittently — passes on retry without any code change — do not mark the task done without first filing a BUG entity:
     ```
     entity(action: "create", type: "bug", name: "<test name> fails intermittently",
            observed: "<what was seen>", expected: "test passes consistently",
            severity: "medium", priority: "medium")
     ```
     Record the BUG ID in the task completion summary. Intermittent failures are not "probably fine" — they indicate non-determinism that will compound in future tasks.
2. Walk through each acceptance criterion. Confirm the implementation satisfies it and identify the test that verifies it.
3. IF any criterion is not met → return to Phase 2 and address the gap.
4. Complete the task with `finish`, including:
   - Summary of what was accomplished and the approach taken
   - Files modified
   - Verification performed — which criteria were checked and how
   - Assumption flags and any retrospective observations
5. For each knowledge entry that proved accurate and useful during this task, call `knowledge(action: "confirm", id: "KE-xxx")`.
6. For each knowledge entry that proved inaccurate or misleading, call `knowledge(action: "flag", id: "KE-xxx", reason: "...")`. **BECAUSE** confirmation is the mechanism by which the knowledge base self-curates, and unflagged inaccurate entries continue to mislead future agents indefinitely.

## Examples

### BAD: Scope creep with missing tests

```
Task: TASK-042 — Add validation for email field on user profile

Changes made:
- Added email regex validation in profile handler
- Refactored the entire validation module to use a builder pattern
- Updated 3 unrelated validators to use the new pattern
- Added test for email validation happy path
```

WHY BAD: The task scope boundary was email validation. Refactoring the validation module and touching unrelated validators is scope creep — it changes files other tasks depend on and introduces risk with no acceptance criterion backing. Only the happy path is tested; error paths (invalid format, empty string, overlong input) are untested code paths.

### BAD: Implementation without spec citation

```
Task: TASK-087 — Implement retry logic for webhook delivery

Implementation: Exponential backoff with base 2s, max 5 retries,
jitter of ±500ms. Tests cover successful retry and max-retry exhaustion.
```

WHY BAD: The retry parameters (base delay, max retries, jitter range) are implementation choices with no spec citation. Were these in the spec, or invented? If the spec said "3 retries with 1s base," this deviates. If the spec was silent, these are assumption flags that need documenting. A reviewer cannot tell the difference.

### GOOD: Spec-grounded implementation with full coverage

```
Task: TASK-087 — Implement retry logic for webhook delivery

Phase 1 — Acceptance criteria:
  AC-1: Retry failed deliveries up to 3 times (spec §4.2)
  AC-2: Use exponential backoff starting at 1 second (spec §4.2)
  AC-3: Record each attempt in the delivery log (spec §4.3)

Phase 2 — Implementation:
  Retry loop in webhook dispatcher (webhook_dispatch.go L45-78).
  Backoff: 1s, 2s, 4s per AC-2. No jitter — spec does not mention it.
  Assumption flag: jitter may be desirable; not adding without spec backing.
  Delivery log recording per AC-3 (delivery_log.go L102-115).

Phase 3 — Tests:
  TestRetry_SuccessOnSecondAttempt — exercises happy path (AC-1)
  TestRetry_ExhaustedAfterThreeAttempts — max retries reached (AC-1)
  TestRetry_BackoffTiming — verifies 1s/2s/4s delays (AC-2)
  TestRetry_DeliveryLogRecorded — log entry per attempt (AC-3)
  TestRetry_FirstAttemptSuccess — no retry needed (edge case)

Phase 4: All tests pass. Each acceptance criterion verified.
Assumption flagged: no jitter.
```

WHY GOOD: Every implementation choice cites a spec requirement. The jitter assumption is explicitly flagged rather than silently decided. All code paths have tests — happy path, exhaustion, timing, logging, and the no-retry edge case. Scope is exactly what the task requires.

### BAD: Skipping knowledge retrieval and re-discovering a known issue

```
Task: TASK-103 — Add rate limiting to the webhook endpoint

Implementation:
  Added token-bucket limiter in webhook handler. Chose 100 req/s limit.
  Ran into an issue: the rate limiter state was being reset on every
  request because it was initialized inside the handler function instead
  of at package level. Spent 2 hours debugging before finding the fix.
```

WHY BAD: The root cause of the wasted time was the ABSENCE of a `knowledge list` call at the start of the task. The knowledge base contained an entry tagged `["rate-limiting", "go"]` stating: "Rate limiter instances must be initialised at package scope, not inside handlers — handler-scoped initialisers reset on every request." The agent re-discovered this known pitfall from scratch, spending 2 hours on an issue that a 30-second knowledge check would have prevented.

### GOOD: Using knowledge retrieval to avoid a known pitfall

```
Task: TASK-103 — Add rate limiting to the webhook endpoint

Phase 1 — Knowledge retrieval:
  Called knowledge(action: "list", tags: ["rate-limiting", "go", "webhook"]).
  Found entry KE-0047: "Rate limiter instances must be initialised at package
  scope, not inside handlers — handler-scoped initialisers reset on every
  request." Noted: initialise limiter at package level, not inside handler.

Phase 2 — Implementation:
  Declared token-bucket limiter as a package-level var in webhook_handler.go.
  Chose 100 req/s limit per spec §5.1.
  Handler references the package-level instance — no per-request reinitialisation.
```

WHY GOOD: The agent called `knowledge list` with domain-relevant tags before writing any implementation code. Finding KE-0047 prevented the per-request reinitialisation mistake before it could be made. The knowledge retrieval step took seconds; the avoided debugging session would have taken hours.

## Evaluation Criteria

1. Does the implementation satisfy every acceptance criterion stated in the task? Weight: required.
2. Are non-trivial implementation choices backed by spec citations? Weight: required.
3. Does every added code path have a corresponding test? Weight: required.
4. Are assumptions explicitly flagged in the completion summary? Weight: high.
5. Did the implementation stay within the task's scope boundary? Weight: high.
6. Were incremental commits made at logical checkpoints? Weight: moderate.
7. Does the task completion summary describe what was done, how, and what was verified? Weight: high.

## Questions This Skill Answers

- How do I start implementing an assigned task?
- What should I read before writing any code?
- When should I stop and ask instead of making an assumption?
- How do I justify an implementation choice between alternatives?
- What does a well-structured task completion look like?
- How do I decide what is in scope vs. out of scope for a task?
- When should I commit during implementation?
- What tests do I need to write for my changes?