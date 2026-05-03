# Design: Wisdom Forwarding

**Plan ID:** P45-wisdom-forwarding  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §6.4

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

Modify `context_assemble` in `internal/context/assemble.go`:

1. When assembling context for a task, identify the parent feature
2. Query all completed sibling tasks under that feature
3. For each completed sibling, retrieve its contributed knowledge entries (from `finish` records)
4. Deduplicate against knowledge already included in the context packet
5. Include deduplicated entries in the handoff prompt, flagged as "surfaced from sibling task TASK-XXX"

**Scope:** Tier-2 (project-level) knowledge entries only. Tier-3 (session-level) entries are ephemeral and shouldn't be forwarded across tasks.

**Deduplication:** If Task 1 and Task 2 both contributed a knowledge entry with the same topic, only include the most recent one. If the entry already appears in the general knowledge query, don't duplicate it.

### Prompt Format

Knowledge from sibling tasks should be included as a distinct section in the handoff prompt:

```
## Knowledge Surfaced from Sibling Tasks

These entries were contributed during earlier tasks in this feature.
They may be relevant to your work.

- **KE-047 (from TASK-001):** The test fixture for ID formatting requires a seeded RNG.
- **KE-089 (from TASK-002):** The ID allocation path in internal/id/ is owned by FEAT-yyy — do not modify.
```

### Opt-Out

Some knowledge entries are task-specific and shouldn't be forwarded. When contributing knowledge via `finish`, add an optional `forward: false` flag:

```
finish(
  task_id: "TASK-001",
  knowledge: [
    { topic: "id-formatting", content: "...", forward: false }
  ]
)
```

Default is `forward: true` for tier-2 entries. Tier-3 entries are never forwarded regardless of flag.

## Alternatives Considered

### Full notepad system (OmO-style)

OmO's notepad system categorizes learnings into Conventions, Successes, Failures, Gotchas, Commands and writes them to `.sisyphus/notepads/{plan-name}/`. This is more structured than simple knowledge forwarding but adds file I/O and a new artifact type.

**Decision:** Start with simple knowledge forwarding. If categorization proves valuable, the notepad pattern can be added later as a separate enhancement. The knowledge entries already have tags and topics — categorization is a presentation layer on top.

### Orchestrator-managed forwarding

The orchestrator explicitly queries knowledge before each dispatch and includes relevant entries in the handoff instructions.

**Decision:** This is what we have today, and it doesn't happen in practice. Automation is the point of this enhancement.

## Dependencies

- None. This is a modification to the `handoff` / `context_assemble` pipeline.
- No new roles, skills, entities, or MCP tools.
- No dependency on P42, P43, or P44.

## Open Questions

1. **Scope boundary:** Should forwarding be limited to the same feature, or extended to the same batch? Cross-feature forwarding risks noise — patterns relevant to one feature may not apply to another in the same batch. Start with same-feature only.
2. **Deduplication strategy:** Topic-based (same topic = duplicate) or content-based (semantic similarity)? Topic-based is simpler and sufficient for a first pass.
3. **Performance:** Querying knowledge entries for all completed siblings adds N queries per handoff call. Acceptable for small N (most features have <10 tasks). If this becomes a bottleneck, batch the query.
