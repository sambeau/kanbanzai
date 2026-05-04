# Implementation Plan: Wisdom Forwarding

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | AI architect                   |

## Overview

This implementation plan breaks down the wisdom forwarding enhancement — automatically surfacing knowledge entries from completed sibling tasks during `handoff` context assembly — into seven tasks covering the knowledge model changes, `finish` tool updates, context assembly pipeline modifications, and comprehensive testing.

## Scope

This plan implements the wisdom forwarding enhancement defined in
`work/P45-wisdom-forwarding/P45-spec-wisdom-forwarding.md` (DOC-`P45-wisdom-forwarding/spec-p45-spec-wisdom-forwarding`).
It covers tasks T1–T7 below.

**In scope:** Automated inclusion of tier-2 knowledge entries from completed sibling tasks in `handoff` context assembly. Topic-based deduplication. Opt-out `forward` flag for non-forwardable entries. Distinct prompt placement for forwarded knowledge. No new tool parameters on `handoff`.

**Out of scope:** Plan-level or cross-feature forwarding. Content-based semantic deduplication. A full notepad system. Changes to knowledge lifecycle. New entity types, MCP tools, roles, or skills.

## Task Breakdown

### Task 1: Add `forward` opt-out flag to knowledge model and storage

- **Description:** Add a `forward` boolean field to knowledge entry records in the knowledge storage layer. When absent (legacy entries), default to forwardable for tier-2 entries. The `ContributeInput` struct in `internal/service/knowledge.go` and the `KnowledgeEntryInput` in `internal/service/dispatch.go` both gain an optional `Forward` field. The storage layer (`internal/storage/`) writes and reads the flag. No changes to the knowledge lifecycle.
- **Deliverable:** Updated `ContributeInput`, `KnowledgeEntryInput`, storage record schema, and `Contribute` method to accept and persist the `forward` flag.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-008, REQ-009.

### Task 2: Add `forward` flag to `finish` tool knowledge parsing

- **Description:** Update `parseFinishKnowledge` in `internal/mcp/finish_tool.go` to parse an optional `forward` boolean from each knowledge entry in the `finish` arguments. Pass it through to the knowledge contribution service. Tier-3 entries ignore the flag (always not-forwardable per REQ-005). Tier-2 entries default to forwardable when the flag is absent (REQ-009). Add `forward` to the `finish` tool schema so callers know the parameter exists.
- **Deliverable:** Updated `parseFinishKnowledge` and `finish` tool parameter documentation.
- **Depends on:** Task 1 (needs new `Forward` field on input structs).
- **Effort:** Small.
- **Spec requirement:** REQ-008, REQ-009.

### Task 3: Implement sibling knowledge query in context assembly

- **Description:** Add a new function `asmLoadSiblingKnowledge` in `internal/mcp/assembly.go` that, given a parent feature ID, queries the entity service for all completed sibling tasks, then queries the knowledge store for tier-2 entries contributed by those tasks. Filters out entries marked `forward: false`. Applies topic-based deduplication (most recent entry wins for same topic per REQ-006). Excludes entries whose topics match those already in the general knowledge set (REQ-007). Returns results ordered most-recent-first (REQ-NF-002). If there are zero completed siblings, returns an empty slice.
- **Deliverable:** New `asmLoadSiblingKnowledge` function with tests.
- **Depends on:** Task 1 (needs `forward` field in storage).
- **Effort:** Medium.
- **Spec requirement:** REQ-001, REQ-004, REQ-005, REQ-006, REQ-007, REQ-NF-001, REQ-NF-002.

### Task 4: Wire sibling knowledge into `assembleContext` and `renderHandoffPrompt`

- **Description:** Modify `assembleContext` in `internal/mcp/assembly.go` to call `asmLoadSiblingKnowledge` after `asmLoadKnowledge`, passing the parent feature ID. Store results in `assembledContext.siblingKnowledge` (new field). Add `entitySvc` to `asmInput` to enable sibling task queries. Update `renderHandoffPrompt` in `internal/mcp/handoff_tool.go` to render a new "### Surfaced Knowledge (from sibling tasks)" section after the "Known Constraints" section but before "Files". Each entry shows the knowledge content annotated with its source task ID. Omit the section entirely when there are no sibling entries (REQ-015 equivalent). The section label must be visually distinct from the general knowledge section (REQ-002, REQ-003, REQ-NF-003).
- **Deliverable:** Updated `assembledContext`, `asmInput`, `assembleContext`, and `renderHandoffPrompt`.
- **Depends on:** Task 3 (needs `asmLoadSiblingKnowledge`).
- **Effort:** Small.
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-NF-003.

### Task 5: Update `handoffTool` to pass entity service to assembler

- **Description:** The `handoffTool` function in `internal/mcp/handoff_tool.go` already has access to `entitySvc`. Ensure `entitySvc` is passed through `asmInput` so `assembleContext` can use it for sibling task queries. Wire the `handoff` MCP tool registration to include `entitySvc` in the assembly input. Verify the `entitySvc` is already available in the handler scope (it is — it's used for task loading and re-review guidance). No new tool parameters on `handoff` — the forwarding is invisible to the orchestrator (REQ-010).
- **Deliverable:** Updated `handoffTool` to pass `entitySvc` in `asmInput`.
- **Depends on:** Task 4 (needs new `entitySvc` field on `asmInput`).
- **Effort:** Small.
- **Spec requirement:** REQ-010.

### Task 6: Unit tests for all new functions

- **Description:** Write comprehensive unit tests covering: (a) `forward` flag parsing and default behavior in `finish`; (b) `forward` flag storage and retrieval in knowledge storage; (c) sibling knowledge query with various scenarios: zero siblings, one sibling, multiple siblings, deduplication by topic, exclusion of non-forwardable entries, tier-3 exclusion, cross-feature isolation; (d) rendering of sibling knowledge section in handoff prompt; (e) ordering by recency; (f) exclusion when general knowledge already contains the same topic; (g) empty section when no siblings. Test both `assembleContext` and `renderHandoffPrompt` integration.
- **Deliverable:** New and updated test files covering all acceptance criteria.
- **Depends on:** Tasks 1–5 (tests verify all above).
- **Effort:** Medium.
- **Spec requirement:** All acceptance criteria AC-001 through AC-015.

### Task 7: Integration test for end-to-end wisdom forwarding flow

- **Description:** Write an integration test (`internal/mcp/integration_test.go`) that creates a feature with multiple sibling tasks, completes them with knowledge entries, then calls `handoff` for a new task and verifies the prompt includes the forwarded knowledge section with correct annotations, deduplication, and ordering. Verify the knowledge store is unchanged after handoff (REQ-011). Verify knowledge lifecycle independence (REQ-012). Verify no new tools/roles/entities were created (REQ-013).
- **Deliverable:** Integration test in `internal/mcp/integration_test.go`.
- **Depends on:** Tasks 1–6 (tests the assembled end-to-end behavior).
- **Effort:** Medium.
- **Spec requirement:** AC-001, AC-010, AC-011, AC-012.

## Dependency Graph

```
Task 1 (opt-out flag on model) ──┬──→ Task 2 (finish tool parsing)
                                │
                                ├──→ Task 3 (sibling knowledge query)
                                │         │
                                │         └──→ Task 4 (wire into assembly + render)
                                │                   │
                                │                   └──→ Task 5 (handoff tool entitySvc)
                                │                             │
                                └─────────────────────────────┴──→ Task 6 (unit tests)
                                                                        │
                                                                        └──→ Task 7 (integration test)
```

Parallel groups: None (sequential dependency chain).

Critical path: Task 1 → Task 3 → Task 4 → Task 5 → Task 6 → Task 7.

Task 2 is in parallel with Task 3 (both depend on Task 1, neither depends on the other).

## Risk Assessment

### Risk: Knowledge store List method lacks `learned_from` filter

- **Probability:** Medium.
- **Impact:** Medium. The spec requires querying knowledge entries by originating task ID (`learned_from` field). The current `KnowledgeFilters` struct has no `LearnedFrom` filter. Without it, we must load all entries and filter in memory.
- **Mitigation:** Add a `LearnedFrom` filter to `KnowledgeFilters` as part of Task 3. This is a small schema addition and aligns with the existing filter pattern. If adding the filter proves unexpectedly complex, fall back to in-memory filtering (performance is acceptable for N ≤ 10 siblings per REQ-NF-001).
- **Affected tasks:** Task 3, Task 6.

### Risk: Entity service unavailable during context assembly

- **Probability:** Low. The `handoffTool` handler already has `entitySvc` and uses it for task loading. The assembly pipeline runs synchronously in the same handler scope.
- **Impact:** Medium. If sibling task queries fail, the handoff could error or silently skip forwarding.
- **Mitigation:** Follow the existing assembly pattern — all assembly operations are best-effort. If the entity query fails, log a warning and return an empty sibling knowledge set. The handoff still succeeds; it just won't include sibling knowledge. This matches the `asmLoadKnowledge` error handling pattern.
- **Affected tasks:** Task 3, Task 4.

### Risk: Topic-based deduplication misses near-duplicate entries

- **Probability:** Medium. Two entries with different but related topics (e.g. "edit_file worktree limitation" vs "worktree file writes") may describe the same constraint but have different normalized topics.
- **Impact:** Low. The sub-agent sees both entries and can reconcile them. The spec explicitly defers content-based deduplication. At worst, the agent sees redundant information.
- **Mitigation:** Monitor for duplicate reports in production. If topic-based deduplication proves insufficient, add content-based deduplication as a follow-up (already noted in the design document's Open Questions).
- **Affected tasks:** Task 3.

## Interface Contracts

### Task 1 → Task 2, Task 3

**Interface:** `ContributeInput.Forward` and `KnowledgeEntryInput.Forward` fields.
- **Producer (Task 1):** Adds `Forward *bool` to both structs and persists the field in the knowledge storage record.
- **Consumers (Task 2, Task 3):** Task 2 reads `Forward` from `finish` tool parameters. Task 3 reads `forward` from knowledge storage records to filter out non-forwardable entries.
- **Contract:** `Forward` is a pointer to bool. `nil` (absent) means default-forwardable for tier-2. `false` means explicitly not-forwardable. Tier-3 entries ignore this flag entirely (always not-forwardable).

### Task 3 → Task 4

**Interface:** `asmLoadSiblingKnowledge` function signature and `assembledContext.siblingKnowledge` field.
- **Producer (Task 3):** Provides `asmLoadSiblingKnowledge(svc, entitySvc, parentFeature, existingTopics) ([]asmKnowledgeEntry, error)`.
- **Consumer (Task 4):** Calls `asmLoadSiblingKnowledge` from `assembleContext`, stores results in `actx.siblingKnowledge`, and renders them in `renderHandoffPrompt`.
- **Contract:** Returned entries are deduplicated and sorted most-recent-first. Empty slice when no siblings or no forwardable entries. Error returns empty slice (best-effort pattern).

### Task 4 → Task 5

**Interface:** `asmInput.entitySvc` field.
- **Producer (Task 4):** Adds `entitySvc *service.EntityService` to `asmInput`.
- **Consumer (Task 5):** Passes existing `entitySvc` from handler scope into `asmInput`.
- **Contract:** `entitySvc` must be non-nil for sibling knowledge to be loaded. If nil, `asmLoadSiblingKnowledge` returns empty slice (graceful degradation).

### External interface stability

- `handoff` MCP tool: No parameter changes. Forwarding is invisible to callers (REQ-010).
- `finish` MCP tool: New optional `forward` boolean in each knowledge entry object. Absent = default forwardable (backward compatible).
- Knowledge store: New optional `forward` field in storage records. Legacy entries without the field are treated as forwardable for tier-2.

## Traceability Matrix

| Spec Requirement | Task(s) | Verification |
|-----------------|---------|-------------|
| REQ-001 (sibling knowledge in context) | Task 3, Task 4 | Unit test (Task 6) |
| REQ-002 (distinct section) | Task 4 | Unit test (Task 6) |
| REQ-003 (source task ID annotation) | Task 4 | Unit test (Task 6) |
| REQ-004 (feature boundary scoping) | Task 3 | Unit test (Task 6) |
| REQ-005 (tier-2 only) | Task 3 | Unit test (Task 6) |
| REQ-006 (topic dedup, most recent) | Task 3 | Unit test (Task 6) |
| REQ-007 (dedup vs general knowledge) | Task 3 | Unit test (Task 6) |
| REQ-008 (opt-out flag) | Task 1, Task 2 | Unit test (Task 6) |
| REQ-009 (default forwardable) | Task 1, Task 2 | Unit test (Task 6) |
| REQ-010 (invisible to orchestrator) | Task 5 | Unit test (Task 6) |
| REQ-011 (read-only, store unchanged) | — | Integration test (Task 7) |
| REQ-012 (lifecycle independence) | — | Integration test (Task 7) |
| REQ-013 (no new tools/roles/entities) | — | Inspection (Task 7) |
| REQ-NF-001 (query count ≤ N+1) | Task 3 | Unit test (Task 6) |
| REQ-NF-002 (most-recent-first ordering) | Task 3 | Unit test (Task 6) |
| REQ-NF-003 (distinct section label) | Task 4 | Unit test (Task 6) |

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|---------------|
| AC-001: Sibling knowledge appears in handoff context | Unit test | Task 6 |
| AC-002: Cross-feature isolation | Unit test | Task 6 |
| AC-003: Tier-3 exclusion | Unit test | Task 6 |
| AC-004: Topic deduplication (most recent wins) | Unit test | Task 6 |
| AC-005: Dedup against general knowledge | Unit test | Task 6 |
| AC-006: Forward=false exclusion | Unit test | Task 6 |
| AC-007: Default forwardable | Unit test | Task 6 |
| AC-008: Distinct section + task ID annotation | Unit test | Task 6 |
| AC-009: Invisible to orchestrator | Unit test / Inspection | Task 5, Task 6 |
| AC-010: Knowledge store unchanged | Integration test | Task 7 |
| AC-011: Lifecycle independence | Integration test | Task 7 |
| AC-012: No new tools/roles/entities | Inspection | Task 7 |
| AC-013: Query count ≤ N+1 | Unit test | Task 6 |
| AC-014: Most-recent-first ordering | Unit test | Task 6 |
| AC-015: Empty section when no siblings | Unit test | Task 6 |
