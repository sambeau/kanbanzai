| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | Draft                          |
| Author | architect                      |

## Scope

This dev-plan implements the specification `FEAT-01KR46PKHSWQF/spec-p61-spec-perf-hygiene` (approved) covering Track C of P61: knowledge entry caching and handoff handler de-duplication.

## Task Breakdown

| Task | Description | Deliverable | Effort |
|------|-------------|-------------|--------|
| T1 | Add `Generation()` method to `KnowledgeService` | O(1) generation token | 1h |
| T2 | Add generation-keyed cache to `KnowledgeSurfacer.Surface()` | Cached surface with invalidation | 2h |
| T3 | De-duplicate `entitySvc.Get` in handoff handler | Single Get call | 0.5h |
| T4 | Write tests: cache hit/miss, generation invalidation, benchmark | Test suite covering all ACs | 2h |

## Dependency Graph

T1 → T2 (cache needs generation token). T3 independent. T4 depends on T1+T2+T3.

Critical path: T1 → T2 → T4 (~5h). T3 parallel with T1+T2.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Cache returns stale data after external modification | Low | Medium | Generation token from mtime; external changes trigger reload |
| Generation() mtime granularity insufficient | Low | Low | Include file count in token for additional precision |

## Verification Approach

All ACs verified by automated tests (T4). Benchmark tests for AC-006 (cached < 1ms) and AC-007 (cold < 100ms). Code review for AC-005 (single Get call).
