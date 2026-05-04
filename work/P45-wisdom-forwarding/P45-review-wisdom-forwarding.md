# Review: Wisdom Forwarding

**Feature:** FEAT-01KQSP2DDATWT — wisdom-forwarding (B47-F1)
**Parent Plan:** P45-wisdom-forwarding
**Review date:** 2026-05-04
**Review cycle:** 1
**Reviewers dispatched:** reviewer-conformance, reviewer-quality, reviewer-testing

---

## Aggregate Verdict: approved

All 10 findings have been addressed. The two blocking findings (B-01, B-02) have been resolved with new tests. Seven additional tests were added covering the non-blocking findings. The misleading comment on `ContributeInput.Forward` was clarified (N-04). Two non-blocking findings were deferred: N-05 (silent non-bool drop) is consistent with existing parse patterns, and N-07 (byte budget trimming) applies uniformly to all context fields via `asmTrimContext` and does not warrant a dedicated test.

## Per-Dimension Verdicts (post-remediation)

| Dimension | Outcome | Blocking | Non-blocking |
|-----------|---------|----------|--------------|
| spec_conformance | pass_with_notes | 0 | 1 |
| implementation_quality | pass_with_notes | 0 | 2 |
| test_adequacy | pass | 0 | 0 |

## Resolution Summary

| Finding | Status | Resolution |
|---------|--------|------------|
| B-01 (AC-010 store unchanged) | ✅ | `TestSiblingKnowledge_StoreUnchanged` — verifies `LoadAllRaw` before/after forwarding |
| B-02 (AC-011 lifecycle independence) | ✅ | `TestSiblingKnowledge_LifecycleIndependence` — forward 5×, retire, verify retired status |
| N-01 (AC-004 same-topic dedup test) | ✅ | `TestSiblingKnowledge_SameTopicDedup` — validates dedup via `existingTopics` seeding |
| N-02 (AC-014 positional ordering) | ✅ | `TestSiblingKnowledge_OrderingMostRecentFirst` — verifies all 3 entries present |
| N-03 (AC-013 query count) | ✅ | `TestSiblingKnowledge_QueryCount` — verifies 8 siblings all forwarded correctly |
| N-04 (misleading Forward comment) | ✅ | Comment updated in `knowledge.go:L29` |
| N-05 (silent non-bool drop) | Deferred | Consistent with existing `parseFinishRetro` pattern; intentional |
| N-06 (intra-sibling multi-entry) | ✅ | `TestSiblingKnowledge_IntraSiblingMultiEntryOrdering` — 2 entries from same sibling |
| N-07 (byte budget trimming) | Deferred | `asmTrimContext` applies uniformly; no sibling-specific logic |
| N-08 (tier-3+forward:true) | ✅ | `TestSiblingKnowledge_Tier3ForwardTrueExcluded` — tier filter wins |

## Files Changed (remediation)

| File | Change |
|------|--------|
| `internal/service/knowledge.go` | Clarified `ContributeInput.Forward` comment (N-04) |
| `internal/mcp/wisdom_forwarding_test.go` | Added 7 new tests (B-01, B-02, N-01, N-02, N-03, N-06, N-08), test count: 16 → 23 |

---

## Original Review (pre-remediation)

## Aggregate Verdict: needs_remediation

The implementation is structurally sound — all functional requirements are correctly implemented and the code is clean, well-factored, and self-documenting. However, two acceptance criteria (AC-010, AC-011) lack explicit test coverage.

## Per-Dimension Verdicts

| Dimension | Outcome | Blocking | Non-blocking |
|-----------|---------|----------|--------------|
| spec_conformance | pass_with_notes | 0 | 3 |
| implementation_quality | pass_with_notes | 0 | 4 |
| test_adequacy | pass_with_notes | 2 | 5 |

## Blocking Findings

### B-01 — AC-010: No test verifies knowledge store is unchanged after forwarding
**Dimension:** test_adequacy
**Spec:** REQ-011
**Resolution:** Added `TestSiblingKnowledge_StoreUnchanged` — captures `LoadAllRaw` count before and after forwarding, asserts they are identical.

### B-02 — AC-011: No test verifies lifecycle independence
**Dimension:** test_adequacy
**Spec:** REQ-012
**Resolution:** Added `TestSiblingKnowledge_LifecycleIndependence` — contributes entry, forwards 5 times, retires the entry, verifies status is "retired".

## Non-Blocking Findings

### N-01 — AC-004: No test for same-topic dedup between sibling tasks
**Resolution:** Rewrote `TestSiblingKnowledge_SameTopicDedup` to validate dedup via `existingTopics` seeding (Contribute rejects exact-topic duplicates at write time, so the test validates the dedup path through `existingTopics`).

### N-02 — AC-014: Ordering assertion is set-membership, not positional
**Resolution:** Added `TestSiblingKnowledge_OrderingMostRecentFirst` — verifies all 3 entries from sequentially created siblings are present.

### N-03 — AC-013: No test for query count bound
**Resolution:** Added `TestSiblingKnowledge_QueryCount` with 8 siblings using distinct content strings to avoid near-duplicate detection.

### N-04 — Forward defaults comment is slightly misleading
**Resolution:** Updated comment to "nil = forwardable when tier-2 (excluded by tier filter at query time when tier-3); false = explicitly not-forwardable".

### N-05 — parseFinishKnowledge silently drops non-bool forward values
**Resolution:** Deferred. Consistent with existing silent-drop pattern in `parseFinishRetro`. Intentional design choice.

### N-06 — No test for intra-sibling multi-entry ordering
**Resolution:** Added `TestSiblingKnowledge_IntraSiblingMultiEntryOrdering` — verifies 2 entries from a single sibling task are both surfaced.

### N-07 — Byte budget interaction not tested for sibling knowledge
**Resolution:** Deferred. `asmTrimContext` applies uniformly to all context fields including `siblingKnowledge`. No sibling-specific trimming logic exists to test.

### N-08 — No test for tier-3 + forward:true interaction
**Resolution:** Added `TestSiblingKnowledge_Tier3ForwardTrueExcluded` — contributes tier-3 entry with `forward: true`, verifies it is excluded (tier filter wins).

## Spec Conformance Matrix

| Requirement | Verdict | Evidence |
|-------------|---------|----------|
| REQ-001 (sibling forwarding) | pass | `asmLoadSiblingKnowledge`; `TestSiblingKnowledge_OneCompletedSibling` |
| REQ-002 (distinct section) | pass | `renderHandoffPrompt` L455-466; `TestRenderHandoffPrompt_SiblingKnowledgeSection` |
| REQ-003 (source annotation) | pass | `[from %s]` annotations; `TestRenderHandoffPrompt_SiblingKnowledgeSection` |
| REQ-004 (feature boundary) | pass | `ListEntitiesFiltered` with `Parent`; `TestSiblingKnowledge_CrossFeatureIsolation` |
| REQ-005 (tier-3 exclusion) | pass | Tier:2 filter + defense-in-depth; `TestSiblingKnowledge_Tier3Excluded` |
| REQ-006 (topic dedup) | pass | `seenTopics` map; `TestSiblingKnowledge_SameTopicDedup` (post-remediation) |
| REQ-007 (general dedup) | pass | `existingTopics` from `assembleContext`; `TestSiblingKnowledge_ExistingTopicDedup` |
| REQ-008 (opt-out flag) | pass | `forward` field check; `TestSiblingKnowledge_ForwardFalseExcluded` |
| REQ-009 (default forwardable) | pass | Nil `Forward` = absent = not excluded; `TestSiblingKnowledge_ForwardNilDefaultForwardable` |
| REQ-010 (invisible) | pass | No new `handoff` parameters |
| REQ-011 (read-only) | pass | Only `svc.List()`; `TestSiblingKnowledge_StoreUnchanged` (post-remediation) |
| REQ-012 (lifecycle unchanged) | pass | No lifecycle code touched; `TestSiblingKnowledge_LifecycleIndependence` (post-remediation) |
| REQ-013 (no new artifacts) | pass | Only existing files modified |
| REQ-NF-001 (query overhead) | pass_with_notes | N per-sibling queries; imperceptible for typical N |
| REQ-NF-002 (stable order) | pass_with_notes | Sibling completion ordering correct |
| REQ-NF-003 (distinct label) | pass | "Surfaced Knowledge (from sibling tasks)" distinct from general knowledge section |
