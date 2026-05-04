# Specification: Wisdom Forwarding

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | AI spec-author                 |

## Problem Statement

This specification implements the wisdom forwarding enhancement described in
`work/P45-wisdom-forwarding/P45-design-wisdom-forwarding.md` (DOC-`P45-wisdom-forwarding/design-p45-design-wisdom-forwarding`). The design introduces automated forwarding of knowledge entries from completed sibling tasks when `handoff` assembles context for a new task dispatch.

Today, knowledge entries from completed tasks are available in the knowledge store but the orchestrator must explicitly query and include them — which rarely happens in practice. This specification defines the behavior that ensures sub-agents automatically receive relevant learnings from sibling tasks within the same feature, without orchestrator intervention.

**In scope:** Automated inclusion of tier-2 knowledge entries from completed sibling tasks in `handoff` context assembly. Topic-based deduplication. Opt-out mechanism for non-forwardable entries. Distinct prompt placement separating forwarded knowledge from general knowledge.

**Out of scope:** Plan-level or cross-feature forwarding. Content-based semantic deduplication. A full notepad system with categorization. Any change to the knowledge lifecycle (contribute → confirm → stale → retire). Any new entity type, MCP tool, role, or skill.

## Requirements

### Functional Requirements

- **REQ-001:** When `handoff` assembles context for a task, it must query all completed sibling tasks within the same parent feature and include their contributed tier-2 knowledge entries in the assembled context.
- **REQ-002:** Forwarded knowledge entries must be presented in a distinct section of the handoff prompt, visually separated from general knowledge entries, so the sub-agent can distinguish between project-level knowledge and feature-specific tactical learnings from sibling tasks.
- **REQ-003:** Each forwarded knowledge entry must be annotated with the source sibling task ID from which it originated (e.g., "surfaced from sibling task TASK-001").
- **REQ-004:** Forwarding must be scoped to the parent feature boundary. Knowledge entries from tasks in different features must not be included, even if those features belong to the same batch or plan.
- **REQ-005:** Only tier-2 (project-level) knowledge entries must be forwarded. Tier-3 (session-level) entries must never be forwarded regardless of any opt-out flag.
- **REQ-006:** When two or more completed sibling tasks have contributed knowledge entries with the same topic, only the most recently contributed entry must be included. Topic-based deduplication is required — content-based semantic deduplication is deferred.
- **REQ-007:** When a knowledge entry from a sibling task has the same topic as an entry already included in the general knowledge query, the forwarded entry must be excluded to avoid duplication.
- **REQ-008:** Knowledge contributors must be able to mark a tier-2 knowledge entry as not-forwardable at contribution time via an opt-out flag. Entries marked as not-forwardable must be excluded from automatic forwarding.
- **REQ-009:** Tier-2 entries contributed without an explicit opt-out flag must be treated as forwardable by default.
- **REQ-010:** The forwarding behavior must be invisible to the orchestrator. The orchestrator calls `handoff` with the same parameters as today; the inclusion of sibling knowledge happens automatically within the context assembly pipeline.
- **REQ-011:** Forwarding must not modify, delete, or alter any knowledge entry in the knowledge store. It is a read-only inclusion in the handoff context.
- **REQ-012:** The forwarding mechanism must not change the knowledge lifecycle. Entries are still contributed at `finish` time and progress through contribute → confirm → stale → retire independently of whether they have been forwarded.
- **REQ-013:** The forwarding mechanism must not introduce any new entity type, MCP tool, role, or skill. It is a modification to the existing `handoff` context assembly pipeline only.

### Non-Functional Requirements

- **REQ-NF-001:** For a feature with N completed sibling tasks, the forwarding query overhead must not add more than N+1 knowledge queries to the `handoff` call. For typical features (N ≤ 10), this overhead must be imperceptible to the orchestrator.
- **REQ-NF-002:** Forwarded knowledge entries must be included in the handoff prompt in a stable order: most recently contributed first, then by task completion order.
- **REQ-NF-003:** The forwarding section header in the handoff prompt must be distinguishable from general knowledge by a distinct label that signals "these entries were contributed by sibling tasks."

## Constraints

- The `handoff` tool's external interface must not change. The assembler automatically enriches the context; callers are unaffected.
- The knowledge store API must not change. Forwarding uses existing query operations — no new knowledge storage methods.
- The `finish` tool must support the opt-out flag but must not require changes to existing `finish` call patterns. Callers who do not specify an opt-out flag get the default (forwardable for tier-2).
- This specification does NOT cover: forwarding across feature boundaries, content-based deduplication, a notepad/categorization system, or changes to knowledge lifecycle state transitions.
- The forwarding section must be included after general knowledge entries in the handoff prompt, so the sub-agent first sees project-level knowledge then feature-specific tactical learnings.
- The forwarding section in the prompt must be omitted entirely when there are no completed sibling tasks or when all completed siblings have zero forwardable knowledge entries.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-004):** Given a feature with two completed sibling tasks (TASK-001 contributed KE-A, TASK-002 contributed KE-B), when `handoff` assembles context for TASK-003 in the same feature, then the assembled context includes both KE-A and KE-B.
- **AC-002 (REQ-004):** Given a feature FEAT-A with completed task TASK-001 (contributed KE-X) and a different feature FEAT-B with a new task TASK-005, when `handoff` assembles context for TASK-005, then KE-X is NOT included (different feature boundary).
- **AC-003 (REQ-005):** Given a completed sibling task that contributed one tier-2 entry and one tier-3 entry, when `handoff` assembles context for a new task in the same feature, then only the tier-2 entry is included. The tier-3 entry is excluded.
- **AC-004 (REQ-006):** Given two completed sibling tasks that both contributed knowledge entries with the same topic ("edit_file worktree limitation"), when `handoff` assembles context, then only the most recently contributed entry is included. The older entry is excluded.
- **AC-005 (REQ-007):** Given a completed sibling task contributed KE-047 and the general knowledge query also returns KE-047, when `handoff` assembles context, then the forwarded copy of KE-047 is excluded (already present in general knowledge).
- **AC-006 (REQ-008, REQ-009):** Given a completed sibling task that contributed a tier-2 entry with `forward: false`, when `handoff` assembles context for a new task in the same feature, then that entry is excluded from forwarding.
- **AC-007 (REQ-008):** Given a completed sibling task that contributed a tier-2 entry without specifying a forwarding flag, when `handoff` assembles context, then that entry is included (default forwardable).
- **AC-008 (REQ-002, REQ-003):** Given forwarded knowledge entries are included in the context, then they appear in a distinct section labeled to indicate they are from sibling tasks, and each entry is annotated with its source task ID.
- **AC-009 (REQ-010):** Given a feature with completed sibling tasks, when the orchestrator calls `handoff` with the same parameters it uses today, then the forwarding behavior executes without any additional orchestrator action.
- **AC-010 (REQ-011):** Given forwarded knowledge entries are included in a handoff context, when the handoff completes, then the knowledge store is unchanged — no entries were modified, deleted, or re-registered.
- **AC-011 (REQ-012):** Given a knowledge entry has been forwarded 10 times, when the entry is independently retired via the normal knowledge lifecycle, then the retirement succeeds and is not blocked or affected by forwarding history.
- **AC-012 (REQ-013):** Given the implemented forwarding mechanism, then no new MCP tool, role YAML file, skill directory, or entity type definition exists as a result of this feature.
- **AC-013 (REQ-NF-001):** Given a feature with 8 completed sibling tasks, when `handoff` is called for a new task, then the total knowledge queries executed are ≤ 9 (one per sibling plus one general query).
- **AC-014 (REQ-NF-002):** Given three completed sibling tasks completed in order TASK-001 (oldest), TASK-002, TASK-003 (newest), each contributing one knowledge entry, when `handoff` assembles context, then the forwarded entries appear in order: TASK-003 entry first, then TASK-002, then TASK-001.
- **AC-015 (REQ-001):** Given a feature with zero completed sibling tasks, when `handoff` is called for the first task in the feature, then the handoff prompt contains no forwarding section.

## Verification Plan

| Requirement(s) | Method | Description |
|----------------|--------|-------------|
| REQ-001 | Test | Unit test: mock feature with 2 completed siblings each with knowledge entries; verify both entries appear in assembled context |
| REQ-002 | Inspection | Review the handoff prompt output format; verify forwarded entries appear under a distinct section header separate from general knowledge |
| REQ-003 | Inspection | Review the handoff prompt output; verify each forwarded entry includes a source task ID annotation |
| REQ-004 | Test | Unit test: two features, each with completed tasks; verify forwarding is isolated per feature and cross-feature entries are excluded |
| REQ-005 | Test | Unit test: sibling with one tier-2 and one tier-3 entry; verify only tier-2 is forwarded |
| REQ-006 | Test | Unit test: two siblings with same-topic entries; verify only the most recent is included |
| REQ-007 | Test | Unit test: sibling entry with topic matching general knowledge entry; verify the forwarded copy is excluded |
| REQ-008 | Test | Unit test: sibling entry marked not-forwardable; verify exclusion from forwarded context |
| REQ-009 | Test | Unit test: sibling entry with no opt-out flag; verify it is included (default forwardable) |
| REQ-010 | Test | Integration test: call `handoff` with standard parameters; verify forwarding occurs without additional orchestrator action |
| REQ-011 | Test | Unit test: after handoff completes, query the knowledge store; verify forwarded entries remain unchanged |
| REQ-012 | Test | Unit test: forward an entry multiple times, then retire it; verify retirement succeeds independently |
| REQ-013 | Inspection | Audit the file tree; verify no new role, skill, tool, or entity type was created |
| REQ-NF-001 | Test | Performance test: 8 siblings; verify knowledge query count ≤ 9 |
| REQ-NF-002 | Test | Unit test: 3 siblings with different completion times; verify forwarded entries ordered most-recent-first |
| REQ-NF-003 | Inspection | Review the forwarding section header label; verify it is distinct from the general knowledge section header |
