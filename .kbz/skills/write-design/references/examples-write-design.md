# Write-Design Examples

Worked examples of correct and incorrect design document patterns.
Linked from `.kbz/skills/write-design/SKILL.md`.

---

## BAD: Design Without Alternatives

> ## Problem and Motivation
> We need a caching layer for the API.
>
> ## Design
> Use Redis as a caching layer. Keys will be formatted as `entity:{id}`.
> TTL will be 5 minutes. We'll use the go-redis client library.
>
> ## Alternatives Considered
> Redis is the industry standard for caching.
>
> ## Decisions
> Use Redis.

**WHY BAD:** Problem statement is one sentence with no context on what is slow, who is affected, or what the impact is. The Design section jumps to implementation detail (specific library, key format) instead of describing component boundaries and interactions. Alternatives Considered does not list any design alternatives — it just justifies the chosen approach. Decisions has no design rationale, context, or consequences.

---

## GOOD: Structured Design with Trade-Off Analysis

> ## Problem and Motivation
>
> The entity listing endpoint (`GET /entities`) performs a full table scan on
> every request. At current growth rates, p95 latency will exceed the 200ms
> SLO within 8 weeks. The endpoint is called ~10k times/day by the dashboard
> and by CI pipelines checking entity status.
>
> Doing nothing means SLO violations that degrade dashboard responsiveness
> and slow CI feedback loops.
>
> ## Design
>
> Introduce a read-through cache between the API handler and the storage layer.
> The cache sits behind the existing `EntityReader` interface, so consumers
> are unaffected. Cache invalidation uses write-through: the `EntityWriter`
> clears relevant cache entries on mutation.
>
> The component boundary is the `EntityReader` interface. No component outside
> the storage package interacts with the cache directly. This keeps the blast
> radius of cache-related bugs contained to the storage layer.
>
> **Failure mode:** If the cache is unavailable, the storage layer falls back
> to direct database reads. Latency degrades but functionality is preserved.
>
> ## Alternatives Considered
>
> **A. In-process LRU cache.** Simple to implement, no infrastructure dependency.
> Trade-off: cache is per-process, so multiple instances serve stale data after
> writes to other instances. Rejected because the system runs multiple replicas.
>
> **B. External shared cache.** Shared across replicas, supports TTL and eviction
> policies. Trade-off: adds an infrastructure dependency and a network hop.
> Chosen because cross-replica consistency is required.
>
> **C. Database-level query caching.** Minimal code change. Trade-off: no control
> over eviction policy; cache is invalidated on any table write, not just relevant
> entities. Rejected because invalidation granularity is too coarse.
>
> ## Decisions
>
> **Decision:** Use an external shared cache behind the `EntityReader` interface.
> **Context:** Multiple replicas serve the same endpoint; in-process caching
> produces inconsistent reads across replicas.
> **Rationale:** Cross-replica consistency requires a shared cache. Placing it
> behind the existing interface contract minimises blast radius and avoids
> coupling the cache implementation to consumers.
> **Consequences:** Adds an infrastructure dependency. Requires cache health
> monitoring. The `EntityReader` interface remains unchanged, so no consumer
> code changes are needed.

**WHY GOOD:** Problem statement quantifies the issue (p95 latency, growth rate, call volume) and states the consequence of inaction. Design describes components and boundaries without implementation detail. Three design alternatives with explicit trade-offs and clear accept/reject reasoning. Failure mode is identified with a recovery strategy. Decision entry has full context, design rationale, and consequences — a future reader can evaluate whether the reasoning still holds.
