| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | architect                      |

## Overview

Track C of P61: add generation-token-based cache to KnowledgeService.Surface and de-duplicate redundant entitySvc.Get call in handoff handler.

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

## Interface Contracts

- `KnowledgeService.Generation() (string, error)` — new O(1) method
- `KnowledgeSurfacer.Surface()` — internal caching added; public API unchanged
- Handoff handler — internal de-duplication; MCP response contract unchanged

## Traceability Matrix

| Task | REQ | AC |
|------|-----|----|
| T1 | REQ-001 | AC-001 |
| T2 | REQ-002, REQ-003 | AC-002, AC-003, AC-004 |
| T3 | REQ-004 | AC-005 |
| T4 | REQ-NF-001, REQ-NF-002, REQ-NF-003 | AC-006, AC-007 |

## Verification Approach

All ACs verified by automated tests (T4). Benchmark tests for AC-006 (cached < 1ms) and AC-007 (cold < 100ms). Code review for AC-005 (single Get call).
