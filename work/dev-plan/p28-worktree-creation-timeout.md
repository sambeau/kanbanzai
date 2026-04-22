# Dev Plan: Worktree Creation Timeout Under Load

**Feature:** FEAT-01KPVDDYZ3182
**Plan:** P28 — Doc Intel Polish and Workflow Reliability
**Spec:** work/spec/p28-worktree-creation-timeout.md
**Design:** P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability
**Date:** 2026-04-22
**Status:** Draft

---

## Scope

This dev plan covers the reliability fix for `worktree(action: create)` timeouts when a
repository has approximately 34 or more existing worktrees. It is bounded by the specification
at `work/spec/p28-worktree-creation-timeout.md` and addresses three concerns:

1. Root-cause investigation and a targeted fix for the underlying timeout.
2. Exponential-backoff retry logic (3 attempts: 2 s / 4 s / 8 s, total budget ≤ 30 s).
3. Tool description update to document the `terminal` + `worktree(action: update)` fallback.

Out of scope: `worktree(action: get)`, `worktree(action: list)`, `worktree(action: remove)`,
new MCP actions or parameters, and any performance work beyond the identified root cause.

---

## Task Breakdown

### Task 1: Investigate and fix the root cause of worktree create timeout

- **Description:** Profile `git worktree list` and lock-file behaviour with 30+ worktrees to
  identify the primary cause of the timeout (O(n) list overhead, lock-file contention, internal
  timeout ceiling, or other). Apply a targeted fix that addresses the root cause directly.
  Add a code comment adjacent to the fix naming the identified cause and explaining why the
  fix resolves it (REQ-005). Do not add retry logic in this task — that is Task 2.
- **Deliverable:** Updated worktree handler with root-cause fix and explanatory code comment.
- **Depends on:** None
- **Effort:** large
- **Spec requirement:** REQ-001, REQ-005

### Task 2: Implement exponential-backoff retry and exhausted-retry error message

- **Description:** Wrap the `git worktree add` call in a retry loop: up to 3 attempts, initial
  backoff 2 s doubling each attempt (2 s, 4 s, 8 s), triggered on timeout or git lock errors.
  Enforce a total elapsed-time ceiling of 30 s; abort early if the next attempt would exceed
  it. On exhaustion, return an error message that contains: (a) the underlying error from the
  final attempt, (b) the attempt count ("3 attempts"), and (c) the fallback command
  `git worktree add <path> -b <branch>`. Retry logic must not mask or suppress the root-cause
  fix from Task 1; both must be present.
- **Deliverable:** Retry loop with backoff and exhausted-retry error message in the worktree
  create handler.
- **Depends on:** Task 1
- **Effort:** medium
- **Spec requirement:** REQ-002, REQ-003

### Task 3: Update worktree tool description with terminal fallback documentation

- **Description:** Update the MCP tool description string for the `worktree` tool so that it
  explicitly documents the `terminal` + `worktree(action: update)` fallback. The description
  must instruct the caller to run `git worktree add <path> -b <branch>` via `terminal` and
  then call `worktree(action: update, ...)` to register the worktree record manually when the
  tool times out. This task is independent of Task 1 and Task 2.
- **Deliverable:** Updated tool description string in the MCP server tool registry.
- **Depends on:** None
- **Effort:** small
- **Spec requirement:** REQ-004

### Task 4: Tests for retry logic, backoff intervals, and error message content

- **Description:** Write unit tests that cover the retry and backoff logic without requiring a
  real git repository with 34+ worktrees. Tests must assert: (a) exactly 3 attempts on total
  failure, (b) backoff sequence of 2 s / 4 s / 8 s (using a fake clock or mock sleep), (c)
  exhausted-retry error contains underlying error text, attempt count, and fallback command,
  (d) exactly 1 call and no sleep when the first attempt succeeds, (e) success on the first
  non-failing attempt when failures precede it. Also verify that `worktree(action: get)`,
  `worktree(action: list)`, and `worktree(action: remove)` response schemas are unchanged.
- **Deliverable:** New and updated tests in the worktree handler test file.
- **Depends on:** Task 2
- **Effort:** medium
- **Spec requirement:** REQ-002, REQ-003, REQ-NF-001, REQ-NF-002, REQ-NF-003

---

## Dependency Graph

```
Task 1 (root-cause investigation + fix)
    └─► Task 2 (retry with exponential backoff + error message)
            └─► Task 4 (tests)

Task 3 (tool description update)   [independent]
```

Task 1 must precede Task 2 because the fix must target the identified root cause; the retry
layer must not mask an unfixed problem. Task 3 can proceed in parallel with Tasks 1–2.
Task 4 requires Task 2 to be complete so the retry API is stable.

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Root cause is not one of the anticipated candidates (list overhead, lock contention, internal timeout) | Medium | High | Task 1 explicitly scopes investigation to four possible causes and permits "other named cause"; fix must be documented regardless |
| Retry delays (up to 14 s cumulative backoff) increase p99 latency noticeably in normal operation | Low | Medium | Retry only triggers on timeout or git lock error; happy-path (AC-003) incurs zero delay |
| Fake-clock approach in tests diverges from real sleep behaviour | Low | Low | Use a mockable sleep function injected at construction; verify the mock receives correct durations |
| Changes to worktree create handler inadvertently affect get/list/remove | Low | High | Task 4 includes regression assertions on the unaffected action response schemas (REQ-NF-002) |
| 30 s total budget still too long for some orchestration contexts | Low | Medium | Budget is spec-mandated (REQ-NF-001); document in tool description that long waits may occur under load |

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------:|----------------|
| AC-001: Code comment naming root cause is adjacent to fix | Code inspection | Task 1 |
| AC-002: 2 failures then success → 3 calls, correct backoff delays | Unit test | Task 4 |
| AC-003: First-attempt success → 1 call, no sleep | Unit test | Task 4 |
| AC-004: 3 consecutive failures → error contains underlying text, "3 attempts", fallback command | Unit test | Task 4 |
| AC-005: Tool description contains terminal + git worktree add + worktree(action: update) guidance | Code inspection | Task 3 |
| AC-006: Total elapsed time with 3 failures at max backoff ≤ 30 s (fake clock) | Unit test | Task 4 |
| AC-007: get / list / remove response schemas unchanged after this change | Unit / integration test | Task 4 |
| AC-008: Retry/backoff unit test runs without a live git repository and asserts all three fields | Unit test (CI) | Task 4 |