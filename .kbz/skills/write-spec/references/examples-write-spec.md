# Write-Spec Examples

Worked examples of correct and incorrect specification patterns.
Linked from `.kbz/skills/write-spec/SKILL.md`.

---

## BAD: Vague Specification Without Verification

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

---

## BAD: Implementation Specification

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

---

## GOOD: Behavioural Specification with Full Traceability

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
