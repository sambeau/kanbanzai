| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | spec-author                    |

## Problem Statement

This specification implements Track C of the design described in
`work/P61-handoff-resilience-binding-hardening/P61-design-handoff-resilience.md`
(P61-handoff-resilience-binding-hardening/design-p61-design-handoff-resilience, approved).

Two concrete weaknesses addressed:
- **T3:** `KnowledgeService.Surface()` loads all entries on every call with no caching — at 140+ entries and growing, this is unnecessary I/O
- **T7:** `entitySvc.Get(parent_feature)` is called twice in the handoff handler for the same feature

**Scope:** Add a generation-token-based cache to KnowledgeService; de-duplicate the redundant Get call in handoff.
**Out of scope:** Full knowledge store caching layer, async pre-computation.

## Requirements

### Functional Requirements

- **REQ-001:** `KnowledgeService` must expose a `Generation() (string, error)` method derived from directory mtime and file count (no full scan).
- **REQ-002:** `KnowledgeSurfacer.Surface()` must cache loaded entries keyed by generation token; reload only on generation mismatch.
- **REQ-003:** The cache must invalidate correctly after `kbz knowledge contribute` adds a new entry.
- **REQ-004:** The handoff handler must call `entitySvc.Get(parent_feature)` exactly once and reuse the result for both re-review guidance and `input.FeatureState`.

### Non-Functional Requirements

- **REQ-NF-001:** Cached `Surface()` path must complete in under 1 ms for the current 140-entry store.
- **REQ-NF-002:** Cold `Surface()` path (cache miss) must complete in under 100 ms for the current 140-entry store.
- **REQ-NF-003:** Generation token computation must be O(1) — no directory listing, no file reads.

## Constraints

- Must NOT change the public API of `KnowledgeSurfacer` beyond adding caching internally.
- Must NOT introduce a global mutex or lock that could cause contention.
- Must NOT change the MCP response contract for handoff.
- Out of scope: async pre-computation, cache warming, TTL-based expiry.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a knowledge directory, when `Generation()` is called, then it returns a token string derived from directory mtime and file count.
- **AC-002 (REQ-002a cache hit):** Given a warm cache with matching generation token, when `Surface()` is called, then it returns cached results without re-reading files.
- **AC-003 (REQ-002b cache miss):** Given a stale cache with mismatched generation token, when `Surface()` is called, then it reloads entries and updates the cache.
- **AC-004 (REQ-003):** Given a cache populated at generation G1, when `kbz knowledge contribute` adds an entry producing generation G2, then the next `Surface()` call detects the mismatch and reloads.
- **AC-005 (REQ-004):** Given the handoff handler, when it processes a request, then `entitySvc.Get(parent_feature)` is called exactly once.
- **AC-006 (REQ-NF-001):** Given a warm cache, when `Surface()` is called, then it completes in under 1 ms.
- **AC-007 (REQ-NF-002):** Given a cold cache, when `Surface()` is called, then it completes in under 100 ms.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated: Generation() returns expected token format |
| AC-002 | Test | Automated: benchmark Surface() with warm cache |
| AC-003 | Test | Automated: mutate directory, verify cache miss triggers reload |
| AC-004 | Test | Automated: contribute entry, verify generation changes |
| AC-005 | Inspection | Code review: verify single entitySvc.Get call in handoff handler |
| AC-006 | Test | Automated: benchmark cached Surface() < 1ms |
| AC-007 | Test | Automated: benchmark cold Surface() < 100ms |
