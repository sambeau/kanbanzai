# Implement-Task Examples

Worked examples of correct and incorrect task implementation patterns.
Linked from `.kbz/skills/implement-task/SKILL.md`.

---

## BAD: Scope creep with missing tests

```
Task: TASK-042 — Add validation for email field on user profile

Changes made:
- Added email regex validation in profile handler
- Refactored the entire validation module to use a builder pattern
- Updated 3 unrelated validators to use the new pattern
- Added test for email validation happy path
```

**WHY BAD:** The task scope boundary was email validation. Refactoring the validation module and touching unrelated validators is scope creep — it changes files other tasks depend on and introduces risk with no acceptance criterion backing. Only the happy path is tested; error paths (invalid format, empty string, overlong input) are untested code paths.

---

## BAD: Implementation without spec citation

```
Task: TASK-087 — Implement retry logic for webhook delivery

Implementation: Exponential backoff with base 2s, max 5 retries,
jitter of ±500ms. Tests cover successful retry and max-retry exhaustion.
```

**WHY BAD:** The retry parameters (base delay, max retries, jitter range) are implementation choices with no spec citation. Were these in the spec, or invented? If the spec said "3 retries with 1s base," this deviates. If the spec was silent, these are assumption flags that need documenting. A reviewer cannot tell the difference.

---

## GOOD: Spec-grounded implementation with full coverage

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

**WHY GOOD:** Every implementation choice cites a spec requirement. The jitter assumption is explicitly flagged rather than silently decided. All code paths have tests — happy path, exhaustion, timing, logging, and the no-retry edge case. Scope is exactly what the task requires.

---

## BAD: Skipping knowledge retrieval and re-discovering a known issue

```
Task: TASK-103 — Add rate limiting to the webhook endpoint

Implementation:
  Added token-bucket limiter in webhook handler. Chose 100 req/s limit.
  Ran into an issue: the rate limiter state was being reset on every
  request because it was initialized inside the handler function instead
  of at package level. Spent 2 hours debugging before finding the fix.
```

**WHY BAD:** The root cause of the wasted time was the ABSENCE of a `knowledge list` call at the start of the task. The knowledge base contained an entry tagged `["rate-limiting", "go"]` stating: "Rate limiter instances must be initialised at package scope, not inside handlers — handler-scoped initialisers reset on every request." The agent re-discovered this known pitfall from scratch, spending 2 hours on an issue that a 30-second knowledge check would have prevented.

---

## GOOD: Using knowledge retrieval to avoid a known pitfall

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

**WHY GOOD:** The agent called `knowledge list` with domain-relevant tags before writing any implementation code. Finding KE-0047 prevented the per-request reinitialisation mistake before it could be made. The knowledge retrieval step took seconds; the avoided debugging session would have taken hours.
