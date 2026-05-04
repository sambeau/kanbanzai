# Review: Wisdom Forwarding

**Feature:** FEAT-01KQSP2DDATWT — wisdom-forwarding (B47-F1)
**Parent Plan:** P45-wisdom-forwarding
**Review date:** 2026-05-04
**Review cycle:** 1
**Reviewers dispatched:** reviewer-conformance, reviewer-quality, reviewer-testing

---

## Aggregate Verdict: needs_remediation

The implementation is structurally sound — all functional requirements are correctly implemented and the code is clean, well-factored, and self-documenting. However, two acceptance criteria (AC-010, AC-011) lack explicit test coverage. These ACs are architecturally guaranteed by the read-only nature of `asmLoadSiblingKnowledge`, but the specification requires tests for them and none exist.

## Per-Dimension Verdicts

| Dimension | Outcome | Blocking | Non-blocking |
|-----------|---------|----------|--------------|
| spec_conformance | pass_with_notes | 0 | 3 |
| implementation_quality | pass_with_notes | 0 | 4 |
| test_adequacy | pass_with_notes | 2 | 5 |

## Review Unit Breakdown

All 6 changed files were reviewed as a single review unit (feature ≤8 files, single concern).

| File | Conformance | Quality | Testing |
|------|-------------|---------|---------|
| `internal/service/knowledge.go` | ✓ | ✓ | ✓ |
| `internal/service/dispatch.go` | ✓ | ✓ | ✓ |
| `internal/mcp/finish_tool.go` | ✓ | ✓ | ✓ |
| `internal/mcp/assembly.go` | ✓ | ✓ | ✓ |
| `internal/mcp/handoff_tool.go` | ✓ | ✓ | ✓ |
| `internal/mcp/wisdom_forwarding_test.go` | ✓ | ✓ | ✓ |

---

## Blocking Findings

### B-01 — AC-010: No test verifies knowledge store is unchanged after forwarding
**Dimension:** test_adequacy
**Spec:** REQ-011 (Forwarding must not modify, delete, or alter any knowledge entry in the knowledge store. It is a read-only inclusion in the handoff context.)
**Location:** Spec verification plan requires a unit test; task TASK-01KQSP6D6XJ1Z (Task 7: Integration test) was closed without a dedicated integration test.
**Analysis:** The code in `asmLoadSiblingKnowledge` is transparently read-only — it only calls `svc.List()` and never calls `Contribute`, `Update`, `Retire`, or any write operation. The unit tests indirectly confirm the store is unchanged (they contribute entries and then query them successfully). However, the spec requires an explicit test: "after handoff completes, query the knowledge store; verify forwarded entries remain unchanged."
**Recommendation:** Add a unit test that captures the store entry count before and after calling `asmLoadSiblingKnowledge`, asserting they are identical.

### B-02 — AC-011: No test verifies lifecycle independence
**Dimension:** test_adequacy
**Spec:** REQ-012 (The forwarding mechanism must not change the knowledge lifecycle. Entries are still contributed at `finish` time and progress through contribute → confirm → stale → retire independently of whether they have been forwarded.)
**Location:** Spec verification plan requires a unit test; no such test exists in `wisdom_forwarding_test.go`.
**Analysis:** The forwarding code does not touch status fields, retire methods, or lifecycle transitions. Forwarding is purely a read operation. However, the AC exists to catch regressions where forwarding might accidentally take a lock, increment a counter, or modify state that blocks retirement. Without a regression test, a future change to `asmLoadSiblingKnowledge` could inadvertently interfere with lifecycle operations.
**Recommendation:** Add a test that contributes a knowledge entry, calls `asmLoadSiblingKnowledge` multiple times, then retires the entry via `knowledgeSvc.Retire()`, and asserts the retirement succeeds.

---

## Non-Blocking Findings

### N-01 — AC-004: No test for same-topic dedup between sibling tasks
**Dimension:** spec_conformance, test_adequacy
**Spec:** REQ-006
**Location:** `internal/mcp/wisdom_forwarding_test.go`
**Analysis:** The `seenTopics` map in `asmLoadSiblingKnowledge` correctly implements deduplication when two siblings contribute the same topic. However, no unit test creates two siblings with the same topic and verifies only one entry survives. Existing tests use distinct topics.
**Recommendation:** Add `TestSiblingKnowledge_SameTopicDedup` with two siblings contributing the same topic; assert only one entry.

### N-02 — AC-014: Ordering assertion is set-membership, not positional
**Dimension:** spec_conformance, test_adequacy
**Spec:** REQ-NF-002
**Location:** `internal/mcp/wisdom_forwarding_test.go` (`TestSiblingKnowledge_MultipleSiblingsOrdered`)
**Analysis:** The test verifies all three sibling entries are present via a map, not their positional order. Since all three tasks are created in the same test without time manipulation, their completion timestamps are effectively identical, making ordering non-deterministic.
**Recommendation:** Set explicit completion timestamps and assert positional order: `result[0].learnedFrom == task3ID`, `result[1].learnedFrom == task2ID`, etc.

### N-03 — AC-013: No test for query count bound (N+1)
**Dimension:** spec_conformance, test_adequacy
**Spec:** REQ-NF-001
**Location:** `internal/mcp/assembly.go:L635-L654`
**Analysis:** The implementation makes 1 `ListEntitiesFiltered` call + N `List` calls (one per sibling). The total overhead is N+1 queries plus 2 from `asmLoadKnowledge`, totaling N+3. While the spec's intent is clearly about imperceptible overhead (not strict algorithmic bounds), a test wrapping the `List` method with a counting interceptor would provide a durable guard against future inner-loop query additions.
**Recommendation:** Add a counting-interceptor test or adjust the spec bound to N+3.

### N-04 — Forward defaults comment is slightly misleading
**Dimension:** implementation_quality
**Spec:** REQ-009
**Location:** `internal/service/knowledge.go:L29`
**Analysis:** The comment on `ContributeInput.Forward` says "nil = default (true for tier-2, false for tier-3)" but the code never writes `forward: false` for tier-3 entries — it simply omits the field. Tier-3 exclusion happens at query time. If a future consumer reads the store without the tier filter, it would see nil forward on tier-3 entries and treat them as forwardable. The comment overstates the guarantee.
**Recommendation:** Clarify the comment: "nil = forwardable when tier-2 (determined at query time); tier-3 entries are excluded by tier filter regardless of this field."

### N-05 — parseFinishKnowledge silently drops non-bool forward values
**Dimension:** implementation_quality
**Location:** `internal/mcp/finish_tool.go:L539-543`
**Analysis:** If a caller passes `"forward": "yes"` (a string instead of bool), the value is silently discarded. This is consistent with the existing parse pattern (silent skip for malformed fields) but means callers get no feedback on type mismatches.
**Recommendation:** Consider whether the silent-drop pattern for `forward` is intentional. If so, document it. If validation errors are preferred, add a warning log or MCP warning response.

### N-06 — No test for intra-sibling multi-entry ordering
**Dimension:** implementation_quality, test_adequacy
**Spec:** REQ-NF-002
**Location:** `internal/mcp/assembly.go:L648-650`
**Analysis:** When a single sibling task contributes multiple knowledge entries, there is no intra-sibling ordering (entries are returned in whatever order `svc.List()` produces). The "most recently contributed first" requirement is satisfied at the sibling level but not within a sibling's entries.
**Recommendation:** Add explicit sorting within sibling entries by creation time, or clarify in the spec that intra-sibling ordering is undefined.

### N-07 — Byte budget interaction not tested for sibling knowledge
**Dimension:** implementation_quality
**Location:** `internal/mcp/assembly.go:L119, L302-304`
**Analysis:** `siblingKnowledge` contributes to `asmByteCount` and is subject to trimming when context exceeds the byte budget. There is no test verifying that sibling entries are trimmed under the same policy as general knowledge entries.
**Recommendation:** Add a test with large sibling knowledge entries that exceed the budget; verify sibling entries are trimmed.

### N-08 — Intra-sibling dedup: no dedicated test for tier-3 + forward:true interaction
**Dimension:** test_adequacy
**Location:** `internal/mcp/wisdom_forwarding_test.go`
**Analysis:** `TestSiblingKnowledge_Tier3Excluded` only tests tier-3 without a forward flag. No test covers the boundary case of a tier-3 entry with `forward: true` — the tier filter should correctly exclude it before the forward check.
**Recommendation:** Add a boundary test: tier-3 entry with `forward: true` is excluded (tier filter wins).

---

## Spec Conformance Matrix

| Requirement | Verdict | Evidence |
|-------------|---------|----------|
| REQ-001 (sibling forwarding) | pass | `asmLoadSiblingKnowledge` L614-684; tested by `TestSiblingKnowledge_OneCompletedSibling` |
| REQ-002 (distinct section) | pass | `renderHandoffPrompt` L455-466; tested by `TestRenderHandoffPrompt_SiblingKnowledgeSection` |
| REQ-003 (source annotation) | pass | `renderHandoffPrompt` L458 `[from %s]`; tested by `TestRenderHandoffPrompt_SiblingKnowledgeSection` |
| REQ-004 (feature boundary) | pass | `ListEntitiesFiltered` with `Parent`; tested by `TestSiblingKnowledge_CrossFeatureIsolation` |
| REQ-005 (tier-3 exclusion) | pass | Tier:2 filter L655 + defense-in-depth check L663-665; tested by `TestSiblingKnowledge_Tier3Excluded` |
| REQ-006 (topic dedup) | pass_with_notes | `seenTopics` map L644-671; logic present but no dedicated intra-sibling test (N-01) |
| REQ-007 (general dedup) | pass | `existingTopics` passed from `assembleContext` L268-270; tested by `TestSiblingKnowledge_ExistingTopicDedup` |
| REQ-008 (opt-out flag) | pass | `forward` field check L660-663; tested by `TestSiblingKnowledge_ForwardFalseExcluded` |
| REQ-009 (default forwardable) | pass | Nil `Forward` = field absent = not excluded; tested by `TestSiblingKnowledge_ForwardNilDefaultForwardable` |
| REQ-010 (invisible) | pass | No new `handoff` parameters; sibling loading internal to `assembleContext` L267-273 |
| REQ-011 (read-only) | pass_with_notes | Code is read-only (only `svc.List`); no explicit regression test (B-01) |
| REQ-012 (lifecycle unchanged) | pass_with_notes | No lifecycle code touched; no retirement test (B-02) |
| REQ-013 (no new artifacts) | pass | Only existing files modified; confirmed by inspection |
| REQ-NF-001 (query overhead) | pass_with_notes | N+1 entity + knowledge queries; exceeds spec bound by 1 but imperceptible (N-03) |
| REQ-NF-002 (stable order) | pass_with_notes | Sibling completion ordering correct; intra-sibling ordering untested (N-02, N-06) |
| REQ-NF-003 (distinct label) | pass | "Surfaced Knowledge (from sibling tasks)" vs "Known Constraints (from knowledge base)" |

---

## Finding Summary

| Classification | Count |
|----------------|-------|
| Blocking | 2 |
| Non-blocking | 8 |
| **Total** | **10** |

---

## Resolution Guidance

The two blocking findings are missing tests for architecturally guaranteed behavior:
- **B-01 (AC-010):** The code is read-only — it cannot modify the knowledge store. Add a simple before/after count assertion.
- **B-02 (AC-011):** The code does not touch lifecycle state. Add a contribute → forward → retire round-trip test.

Both are low-effort additions (~10 lines each). Task 7 (integration test) was marked done but did not produce these tests. Either:
1. **Fast path:** Reopen Task 7 and add the two missing tests, or
2. **Override path:** Accept the structural guarantee as sufficient and advance to done.
