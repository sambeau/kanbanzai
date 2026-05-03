# Design: Wisdom Forwarding

**Plan ID:** P45-wisdom-forwarding  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §6.4

## Revision Notes (2026-07-18)

1. **Research divergence acknowledged (§Design):** The competitive analysis §6.4 describes forwarding as feature-scoped ("tasks in the same feature"), while the independent evaluation Finding 2.3 and Recommendation 6 describe OmO's notepad as plan-scoped (`{plan-name}`). The design retains feature-scoping (matching the competitive analysis which is the primary shaping artifact per P41), but explicitly notes the divergence and suggests plan-scoped forwarding as a potential future extension. Open Question 1 (which contradicted the Non-Goals by asking about batch extension) was removed — the non-goal states the decision.
2. **Over-specification removed:** The concrete prompt format syntax and `forward: false` JSON parameter shape were removed — these are implementation details that belong in a spec or dev-plan, not a design document. The mechanisms are described in prose instead.
3. No structural changes. No new sections. No scope expansion.

---

## Overview

When Task 1 discovers a convention, Task 5 should benefit automatically. Currently, Kanbanzai's `handoff` includes knowledge entries, but the process is pull-based — the orchestrator must explicitly query knowledge and forward it. OmO's model is push-based: learnings from completed tasks automatically flow to subsequent sub-agents. This is a small, standalone enhancement to the `handoff` context assembly pipeline.

Both the main competitive analysis and the independent companion evaluation identify this gap.

## Goals and Non-Goals

**Goals:**
- When `handoff` assembles context for a task, automatically include knowledge entries from sibling tasks completed earlier in the same feature
- Reduce repeated mistakes across tasks by surfacing relevant learnings
- No new roles, no new entities, no new MCP tools — just an enhancement to context assembly

**Non-Goals:**
- Not replacing the knowledge contribution system — entries are still contributed at `finish` time
- Not changing knowledge lifecycle (contribute → confirm → stale → retire)
- Not forwarding knowledge across features or batches — scoped to sibling tasks within the same feature
- Not adding a new UI or interaction pattern — this is invisible to the orchestrator

## Design

### Current Flow

```
Orchestrator dispatches Task 3
  → handoff(task_id: "TASK-003")
  → context assembly: spec sections + role + file paths + general knowledge
  → spawn_agent
```

Knowledge from Task 1 and Task 2 is available in the knowledge store, but the orchestrator must explicitly query and include it. In practice, this rarely happens — the orchestrator is focused on dispatch, not knowledge retrieval.

### Proposed Flow

```
Orchestrator dispatches Task 3
  → handoff(task_id: "TASK-003")
  → context assembly: spec sections + role + file paths + general knowledge
  → PLUS: knowledge entries from completed sibling tasks (TASK-001, TASK-002)
  → spawn_agent
```

### Implementation

Modify the context assembly pipeline (`handoff`):

1. When assembling context for a task, identify the parent feature
2. Query all completed sibling tasks under that feature
3. For each completed sibling, retrieve its contributed knowledge entries (from `finish` records)
4. Deduplicate against knowledge already included in the context packet
5. Include deduplicated entries in the handoff prompt, flagged as "surfaced from sibling task TASK-XXX"

**Scope:** Tier-2 (project-level) knowledge entries only. Tier-3 (session-level) entries are ephemeral and shouldn't be forwarded across tasks.

**Note on research divergence:** The competitive analysis §6.4 scopes forwarding to "tasks in the same feature." The independent evaluation Finding 2.3 describes OmO's notepad as plan-scoped (`.sisyphus/notepads/{plan-name}/`), and Recommendation 6 frames it as "intra-plan knowledge accumulation." These two research documents use different scope boundaries. This design follows the competitive analysis (feature-scoped) because it is the primary shaping artifact per P41. Extending forwarding to the plan level is a potential future enhancement if feature-scoped forwarding proves effective and cross-feature patterns within a plan become valuable.

**Deduplication:** If Task 1 and Task 2 both contributed a knowledge entry with the same topic, only include the most recent one. If the entry already appears in the general knowledge query, don't duplicate it. Start with topic-based deduplication (same topic = duplicate); content-based semantic deduplication is a potential refinement if topic-based proves insufficient.

### Prompt Placement

Knowledge from sibling tasks is included as a distinct section in the handoff prompt, separated from general knowledge entries so the sub-agent can distinguish between project-level knowledge and feature-specific tactical learnings.

### Opt-Out

Some knowledge entries are task-specific and shouldn't be forwarded. When contributing knowledge via `finish`, an optional opt-out flag allows the contributor to mark an entry as not-forwardable. Default is forwardable for tier-2 entries. Tier-3 entries are never forwarded regardless of flag.

## Alternatives Considered

### Full notepad system (OmO-style)

OmO's notepad system categorizes learnings into Conventions, Successes, Failures, Gotchas, Commands and writes them to `.sisyphus/notepads/{plan-name}/`. This is more structured than simple knowledge forwarding but adds file I/O and a new artifact type.

**Decision:** Start with simple knowledge forwarding. If categorization proves valuable, the notepad pattern can be added later as a separate enhancement. The knowledge entries already have tags and topics — categorization is a presentation layer on top.

### Orchestrator-managed forwarding

The orchestrator explicitly queries knowledge before each dispatch and includes relevant entries in the handoff instructions.

**Decision:** This is what we have today, and it doesn't happen in practice. Automation is the point of this enhancement.

### Status quo (no forwarding)

Knowledge is available in the store but requires explicit querying. Sub-agents don't benefit from sibling task learnings.

**Reject:** Both research documents identify this gap. The implementation cost is low — a query in the context assembly pipeline. The benefit (reduced repeated mistakes across tasks) justifies the effort.

## Dependencies

- None. This is a modification to the `handoff` / context assembly pipeline.
- No new roles, skills, entities, or MCP tools.
- No dependency on P42, P43, or P44.

## Open Questions

1. **Deduplication strategy:** Topic-based (same topic = duplicate) or content-based (semantic similarity)? Topic-based is simpler and sufficient for a first pass. Content-based deduplication adds complexity but catches near-duplicate entries with different topics. Start with topic-based; monitor for duplicates that slip through.

2. **Performance:** Querying knowledge entries for all completed siblings adds N queries per handoff call. Acceptable for small N (most features have <10 tasks). If this becomes a bottleneck, batch the query.

3. **Future scope expansion:** If feature-scoped forwarding proves effective, should forwarding extend to the plan level (all sibling features within the same plan)? This would align with the independent evaluation's plan-scoped framing. Defer until feature-scoped forwarding has deployment experience.
