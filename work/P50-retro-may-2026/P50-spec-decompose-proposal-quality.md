# Specification: Decompose Proposal Quality

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | spec-author                    |

## Overview

This specification implements the decompose proposal quality improvements
described in `work/P50-retro-may-2026/P50-design-retro-may-2026.md`
(DOC-`P50-retro-may-2026/design-p50-design-retro-may-2026`), Feature 2.

`decompose propose` has three quality problems that cause agents to bypass
it and decompose manually: it produces plausible-but-wrong proposals when a
spec has no parseable acceptance criteria, it emits single tasks when agents
need implementation-plus-test task pairs, and its dependency graph assumes
partial task completion rather than treating each concern as a complete task.

## Scope

**In scope:**
- Refuse-to-propose mode when no acceptance criteria are found
- Implementation + test task pairs (paired-task output)
- Dependency graph fix: dependencies between complete tasks only

**Out of scope:**
- Redesigning the decompose prompt from scratch
- Changing the decompose review/apply workflow
- Adding a new document type for prompts

## Functional Requirements

- **REQ-001:** When `decompose propose` is called on a specification that
  has no parseable acceptance criteria (after P24 AC format recognition),
  the tool must return an error with the message: "Cannot decompose: no
  acceptance criteria found in spec {spec_id}. Ensure the spec uses
  **AC-NN.** format and the index is current."
- **REQ-002:** When `decompose propose` is called with paired-test-task
  mode enabled (the default), each acceptance criterion must produce two
  tasks: one implementation task and one paired test task.
- **REQ-003:** The paired test task must have its `depends_on` field set
  to the corresponding implementation task ID, so the implementation task
  must be completed before the test task becomes ready.
- **REQ-004:** Paired-test-task mode must be configurable via a flag on
  the `decompose propose` call. When disabled, the existing one-task-per-AC
  behaviour must be preserved.
- **REQ-005:** The dependency graph produced by `decompose propose` must
  only express dependencies between complete tasks. No task may depend on
  a partial-completion state of another task.
- **REQ-006:** When a given acceptance criterion is inherently a testing
  concern (e.g. "verify that X appears in the output"), the proposal must
  produce a single test task rather than a redundant impl+test pair.

## Non-Functional Requirements

- **REQ-NF-001:** The refuse-to-propose check must not add measurable
  latency — the AC extraction already runs; the check is a conditional on
  the result.
- **REQ-NF-002:** Existing `decompose propose` callers that use the
  default settings must still receive valid proposals, though the task
  count will change due to pairing.

## Constraints (Scope Exclusions)

- The `decompose propose` tool signature must not change — pairing must
  be controlled via an optional field in the existing input struct.
- The P24 AC format recognition logic must be the single source of truth
  for "has acceptance criteria" — refuse-to-propose must use the same
  parser.
- This specification does NOT change `decompose review` or `decompose
  apply` behaviour.
- This specification does NOT add a `prompt` document type.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a specification with no parseable acceptance
  criteria, when `decompose propose` is called, then the tool returns an
  error containing "Cannot decompose: no acceptance criteria found" and
  does not produce a proposal.
- **AC-002 (REQ-002):** Given a specification with three acceptance
  criteria (AC-001, AC-002, AC-003), when `decompose propose` is called
  with default settings, then the proposal contains six tasks: three
  implementation tasks and three test tasks.
- **AC-003 (REQ-003):** Given a paired-task proposal, when examining a
  test task's `depends_on` field, then it references the ID of the
  corresponding implementation task.
- **AC-004 (REQ-004):** Given a specification with acceptance criteria,
  when `decompose propose` is called with paired-test-task mode disabled,
  then the proposal contains one task per acceptance criterion (the
  existing behaviour).
- **AC-005 (REQ-005):** Given a proposal's dependency graph, no edge
  represents a dependency on a partial-completion state — every edge
  connects two complete task nodes.
- **AC-006 (REQ-006):** Given an acceptance criterion that is purely a
  testing concern (e.g. "verify that the error message format matches the
  spec"), when `decompose propose` runs, then the proposal produces a
  single test task for that criterion, not a redundant impl+test pair.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: call decompose propose with an AC-free spec, assert error returned |
| AC-002 | Test | Unit test: call decompose propose with 3-AC spec, assert 6 tasks (3 impl + 3 test) |
| AC-003 | Test | Unit test: for each test task in paired output, verify depends_on points to correct impl task |
| AC-004 | Test | Unit test: call decompose propose with paired=false, assert one task per AC |
| AC-005 | Test | Unit test: parse dependency graph, assert every dependency target is a complete task |
| AC-006 | Test | Unit test: spec with test-only AC produces single test task, not impl+test pair |
| REQ-NF-001 | Test | Benchmark decompose propose with and without refuse-to-propose logic, assert no measurable latency difference |
| REQ-NF-002 | Test | Run decompose propose with default settings on existing specs, assert proposals are structurally valid (correct task count change is expected, not a regression) |
