# Decompose-Feature Examples

Worked examples of correct and incorrect feature decomposition.
Linked from `.kbz/skills/decompose-feature/SKILL.md`.

---

## BAD: Horizontal slicing with no integration

```
Feature: FEAT-088 — Add user notification preferences

Task 1: Create database migration for preferences table
Task 2: Add preferences API endpoints
Task 3: Build preferences UI components
Task 4: Write notification filtering logic
```

**WHY BAD:** Four horizontal slices, each covering one layer. No dependency declarations — can Task 2 be implemented without Task 1's schema? No integration task verifies that the API actually reads from the database or that the UI calls the API correctly. No test tasks. Task descriptions are titles only — an implementing agent receives no acceptance criteria and must guess the table schema, endpoint paths, and UI behaviour.

---

## BAD: Over-decomposed with implicit dependencies

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

**WHY BAD:** 11 tasks for a focused feature. Tasks 2-4 are micro-tasks that would each take minutes — the coordination overhead of dispatching, monitoring, and integrating them exceeds the implementation cost. Tasks 6-8 are test afterthoughts separated from the code they test. Task 5 implicitly depends on Tasks 1-4 but no dependency edges are declared. Tasks 2-4 have no acceptance criteria beyond their title.

---

## GOOD: Vertical slices with validated dependencies

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

**WHY GOOD:** Three tasks instead of eleven. Task 1 is a vertical slice — it delivers the complete retry mechanism end-to-end including the database change, logic, and unit tests. Task 2 adds observability as a separate concern with a clear dependency. Task 3 is an explicit integration task that depends on both. Every task has a description with acceptance criteria and spec citations. Dependencies are declared. Each task is single-agent scope. The validation loop caught a missing dependency and fixed it.
