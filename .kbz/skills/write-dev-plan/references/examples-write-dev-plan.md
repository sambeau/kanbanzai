# Write-Dev-Plan Examples

Worked examples of correct and incorrect dev-plan patterns.
Linked from `.kbz/skills/write-dev-plan/SKILL.md`.

---

## BAD: Plan Without Dependencies or Verification

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

---

## GOOD: Structured Plan with Traceability

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
