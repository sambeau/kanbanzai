| Field  | Value                                              |
|--------|----------------------------------------------------|
| Date   | 2025-07-14                                         |
| Status | approved |
| Author | spec-author                                        |

# Specification: Worktree Creation Timeout Under Load

## Overview

This specification covers the reliability failure in `worktree(action: create)` when a
repository has approximately 34 or more existing worktrees, defining requirements for
root-cause investigation, exponential-backoff retry logic, improved error messaging on
exhaustion, and tool-description updates to document the `terminal` fallback procedure.

## Problem Statement

This specification covers the reliability failure in `worktree(action: create)` when a repository
has approximately 34 or more existing worktrees. Under these conditions the tool times out, likely
due to `git worktree list` serialisation overhead or `.git/worktrees/*/lock` file contention; it
has no retry logic. Sequential calls also time out, making this a hard blocker at plan start when
multiple features require worktrees before agent dispatch.

**Parent design document:** `P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`
(§Sprint 2, Issue 3).

## Scope

**In scope:**
- Root-cause investigation and targeted fix for the `worktree(action: create)` timeout.
- Exponential-backoff retry logic on timeout and git lock errors.
- Error message content when all retries are exhausted.
- MCP tool-description update documenting the `terminal` fallback.
- Code comment documenting the investigation result.

**Out of scope:**
- `worktree(action: get)`, `worktree(action: list)`, and `worktree(action: remove)` — these
  actions must not be changed.
- New MCP tool actions or parameters.
- Performance optimisation of git operations beyond the identified root cause fix.

---

## Functional Requirements

**REQ-001** — Root-cause investigation  
The implementation must profile and identify which of the following is the primary cause of the
timeout before applying a fix: (a) `git worktree list` taking O(n) time proportional to the
number of existing worktrees; (b) lock-file contention from concurrent `git worktree add` calls;
(c) the MCP tool's internal timeout being set too low; or (d) another identifiable cause. The fix
must address the identified root cause directly.

**REQ-002** — Retry with exponential backoff  
`worktree(action: create)` must retry the git worktree creation operation up to 3 times when the
operation fails due to a timeout or a git lock error. The initial backoff interval must be 2
seconds, doubling on each subsequent attempt (2 s, 4 s, 8 s).

**REQ-003** — Exhausted-retry error content  
After 3 consecutive failures, `worktree(action: create)` must return an error response that
contains all three of the following:
1. The underlying error message from the final attempt.
2. The number of attempts made (3).
3. The recommended fallback command in the form:
   `git worktree add <path> -b <branch>`

**REQ-004** — Tool description fallback documentation  
The MCP tool description for the `worktree` tool must include explicit documentation of the
`terminal` + `worktree(action: update)` fallback. The description must state, verbatim or
semantically equivalent:
> "If this tool times out, use `terminal` with `git worktree add <path> -b <branch>` then call
> `worktree(action: update, ...)` to register the worktree record manually."

**REQ-005** — Root-cause code comment  
The source file(s) where the root-cause fix is applied must include a code comment explaining
which root cause was identified and why the chosen fix addresses it. The comment must be present
adjacent to the fix, not in a separate documentation file.

## Non-Functional Requirements

**REQ-NF-001** — Total retry budget  
The total elapsed time for `worktree(action: create)` across all retry attempts must not exceed
30 seconds (3 attempts × up to ~8 s backoff each, plus git operation time). The implementation
must enforce this ceiling and abort if it would be exceeded.

**REQ-NF-002** — No regression on unaffected actions  
`worktree(action: get)`, `worktree(action: list)`, and `worktree(action: remove)` must complete
within their existing latency budgets. No timing regression is permitted on these actions.

**REQ-NF-003** — Retry logic test coverage  
The retry and backoff logic must be covered by at least one unit test that validates: correct
attempt count, correct backoff intervals, and correct error content on exhaustion — without
requiring a real git repository with 34+ worktrees.

---

## Constraints

- `worktree(action: get)`, `worktree(action: list)`, and `worktree(action: remove)` must not be
  modified in behaviour, signature, or response schema.
- No new MCP tool actions or parameters may be introduced by this feature.
- The fallback procedure described in both the error message (REQ-003) and the tool description
  (REQ-004) must use only `terminal` and existing `worktree` tool actions.
- The JSON tag audit and `classification_nudge` enhancement (§Sprint 1) are out of scope for this
  feature.
- The retry logic must not mask or suppress the root-cause fix; both must be present.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given the `worktree(action: create)` implementation, when the code is
inspected, then a code comment adjacent to the root-cause fix identifies the specific cause (one
of: `git worktree list` O(n) overhead, lock-file contention, internal timeout ceiling, or other
named cause) and explains why the fix addresses it.

**AC-002 (REQ-002):** Given a repository where `git worktree add` fails on the first two attempts
with a lock error or timeout, when `worktree(action: create)` is called, then the operation is
retried up to 3 times with backoff intervals of approximately 2 s, 4 s, and 8 s respectively,
and succeeds on the first non-failing attempt.

**AC-003 (REQ-002):** Given a repository where `git worktree add` succeeds on the first attempt,
when `worktree(action: create)` is called, then no retry delay is incurred and the operation
completes normally.

**AC-004 (REQ-003):** Given a repository where `git worktree add` fails on all 3 attempts, when
`worktree(action: create)` is called, then the returned error message contains: (a) the underlying
error text from the final attempt, (b) the string "3 attempts" or equivalent, and (c) a
`git worktree add <path> -b <branch>` fallback command string.

**AC-005 (REQ-004):** Given the MCP server tool registry, when the `worktree` tool description
is inspected, then it contains explicit guidance directing the caller to use `terminal` +
`git worktree add <path> -b <branch>` and then `worktree(action: update, ...)` when the tool
times out.

**AC-006 (REQ-NF-001):** Given all 3 retry attempts fail at maximum backoff, when the total wall
time is measured, then it does not exceed 30 seconds from the first attempt to error return.

**AC-007 (REQ-NF-002):** Given a repository with 34+ existing worktrees, when
`worktree(action: get)`, `worktree(action: list)`, or `worktree(action: remove)` is called, then
each action completes within its pre-existing latency budget and returns the same response schema
as before this change.

**AC-008 (REQ-NF-003):** Given the test suite, when the retry/backoff unit test is run without a
real git repository, then it asserts: the operation is attempted exactly 3 times on total failure,
the backoff sequence is 2 s / 4 s / 8 s, and the exhausted-retry error contains all three
required fields (error message, attempt count, fallback command).

---

## Verification Plan

| Criterion | Method     | Description                                                                                                                         |
|-----------|------------|-------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Inspection | Read the source file containing the root-cause fix; confirm a code comment naming the cause and justifying the fix is present.      |
| AC-002    | Test       | Unit test that injects 2 lock-error failures then 1 success; assert 3 total calls and correct backoff delays.                       |
| AC-003    | Test       | Unit test with an immediately-succeeding mock; assert exactly 1 call with no sleep invoked.                                         |
| AC-004    | Test       | Unit test that injects 3 consecutive failures; assert the error string contains the underlying error, "3 attempts", and the fallback command. |
| AC-005    | Inspection | Read the MCP server tool description string for `worktree`; confirm the `terminal` + `git worktree add` + `worktree(action: update)` fallback is documented. |
| AC-006    | Test       | Unit test with 3 failures at max backoff; assert total elapsed time (using a fake clock) does not exceed 30 s.                      |
| AC-007    | Test       | Integration or unit test calling `get`, `list`, and `remove` before and after the change; assert response schemas are unchanged.    |
| AC-008    | Test       | Run the retry/backoff unit test in CI without a live git repository; assert all three sub-assertions pass.                          |