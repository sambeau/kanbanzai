# Worktree Graph Context — Specification

> Specification for FEAT-01KNA-11F5FAFW (worktree-graph-context)
> Plan: P21-codebase-memory-integration
> Design: work/design/codebase-memory-integration.md

---

## Overview

This specification covers Phase 2 of the codebase memory integration: flowing
graph project awareness through the worktree system into agent context. Today,
agents working on feature branches either do not use `codebase_memory_mcp` at
all, or use it inconsistently because they do not know which project name to
query. This feature adds a `GraphProject` field to `worktree.Record` and wires
it through `handoff`, `next`, `status`, and `cleanup` so that agents
automatically receive the correct project name, tool call examples, and
re-indexing instructions — without Kanbanzai's Go code ever calling
`codebase_memory_mcp` directly.

---

## Scope

**In scope:**

- New `GraphProject string` field on `worktree.Record`
- `worktree(action: create)` and `worktree(action: update)` accepting a
  `graph_project` parameter to set the field
- `handoff` emitting a `## Code Graph` section (project set, project empty,
  or omitted) based on worktree and `GraphProject` state
- `next(id)` including `graph_project` in structured context output
- `status` emitting a `missing_graph_index` info-level attention item when a
  worktree exists but `GraphProject` is empty
- `cleanup` and `worktree(action: remove)` noting the graph project for
  manual deletion when `GraphProject` is non-empty
- Section ordering: `## Code Graph` appears after `## Available Tools`

**Explicitly out of scope:**

- Kanbanzai Go code calling `codebase_memory_mcp` directly (MCP-to-MCP)
- Automatic indexing triggered by worktree creation or commits
- Making graph indexing a stage gate prerequisite
- Changes to `codebase_memory_mcp` itself
- Skill file and stage binding documentation updates (tracked separately)
- Phase 3 direct process integration

---

## Functional Requirements

**FR-001:** The `worktree.Record` struct MUST include a `GraphProject string`
field. The default value MUST be the empty string. Worktrees created before
this feature (with no `GraphProject` persisted) MUST deserialise with an empty
`GraphProject` and exhibit no change in behaviour.

**FR-002:** `worktree(action: create)` MUST accept an optional `graph_project`
string parameter. When provided, the value MUST be stored in the
`GraphProject` field of the new worktree record. When omitted, `GraphProject`
MUST be the empty string.

**FR-003:** `worktree(action: update)` MUST accept an optional `graph_project`
string parameter. When provided, the value MUST overwrite the existing
`GraphProject` field on the worktree record. When omitted, the existing value
MUST be preserved.

**FR-004:** When `handoff(task_id)` generates a sub-agent prompt and the
parent feature's worktree record has a non-empty `GraphProject`, the output
MUST include a `## Code Graph` section containing:

1. The project name
2. Per-tool call examples for `search_graph`, `trace_call_path`, `query_graph`,
   and `get_code_snippet`, each with the project name pre-filled
3. An instruction to prefer graph tools over `grep` or `read_file` for
   structural questions (callers, callees, symbol definitions, impact analysis)
4. A re-indexing instruction telling the agent to run `index_repository` on the
   worktree path if new files are created or code is significantly restructured

**FR-005:** When `handoff(task_id)` generates a sub-agent prompt and the parent
feature has a worktree but the worktree's `GraphProject` is empty, the output
MUST include a `## Code Graph` section that instructs the agent to run
`index_repository` on the worktree path before starting structural code
exploration.

**FR-006:** When `handoff(task_id)` generates a sub-agent prompt and no
worktree exists for the parent feature, the `## Code Graph` section MUST be
omitted entirely from the output.

**FR-007:** The `## Code Graph` section emitted by `handoff` MUST appear AFTER
the `## Available Tools` section (from FEAT-01KNA-11F3BBMP role-tool-hints)
when both are present.

**FR-008:** `next(id)` MUST include a `graph_project` string field in the
structured context output alongside the existing `worktree` field. When the
worktree record's `GraphProject` is non-empty, the value MUST be that project
name. When `GraphProject` is empty or no worktree exists, the value MUST be
the empty string.

**FR-009:** When `status` is called for a feature that has an active worktree
with an empty `GraphProject`, the response's attention array MUST contain an
item with `type: "missing_graph_index"`, `severity: "info"`, and a `message`
suggesting the agent index the worktree to enable graph-based navigation.

**FR-010:** The `missing_graph_index` attention item (FR-009) MUST NOT be
emitted when the worktree's `GraphProject` is non-empty, when the feature has
no worktree, or when the worktree has been removed.

**FR-011:** When `worktree(action: remove)` removes a worktree whose record
has a non-empty `GraphProject`, the response MUST include a note identifying
the graph project name and instructing the agent or user to run
`delete_project` to free the index.

**FR-012:** When `cleanup(action: execute)` removes a worktree whose record
has a non-empty `GraphProject`, the cleanup output MUST include a note
identifying the graph project name and instructing the agent or user to run
`delete_project` to free the index.

**FR-013:** If `codebase_memory_mcp` is unavailable (tools not present in
the session), all behaviour MUST be identical to today. The `GraphProject`
field, handoff sections, and attention items are inert metadata —
they MUST NOT cause errors or block any workflow operation.

---

## Non-Functional Requirements

**NFR-001:** The addition of the `GraphProject` field and its propagation
through `handoff`, `next`, `status`, and `cleanup` MUST NOT increase the p99
latency of any of these operations by more than 5ms. The field is read from
an already-loaded worktree record; no new I/O is introduced.

**NFR-002:** Worktree records persisted before this feature (lacking a
`GraphProject` key) MUST deserialise without error. The missing key MUST be
treated as the empty string. No data migration is required.

**NFR-003:** The `## Code Graph` section emitted by `handoff` MUST NOT exceed
500 bytes when `GraphProject` is set, to limit prompt token overhead.

---

## Acceptance Criteria

**AC-001 (FR-001):** Given a worktree record persisted before this feature
(no `GraphProject` key in YAML), when the record is loaded, then
`GraphProject` MUST be the empty string and all existing worktree behaviour
MUST be unchanged.

**AC-002 (FR-002):** Given `worktree(action: create, entity_id: "FEAT-XXX",
graph_project: "kanbanzai-FEAT-XXX")` is called, when the worktree record is
subsequently retrieved, then `GraphProject` MUST equal
`"kanbanzai-FEAT-XXX"`.

**AC-003 (FR-002):** Given `worktree(action: create, entity_id: "FEAT-XXX")`
is called without a `graph_project` parameter, when the worktree record is
subsequently retrieved, then `GraphProject` MUST be the empty string.

**AC-004 (FR-003):** Given a worktree with `GraphProject` equal to the empty
string, when `worktree(action: update, entity_id: "FEAT-XXX", graph_project:
"kanbanzai-FEAT-XXX")` is called, then `GraphProject` MUST be updated to
`"kanbanzai-FEAT-XXX"`.

**AC-005 (FR-003):** Given a worktree with `GraphProject` equal to
`"kanbanzai-FEAT-XXX"`, when `worktree(action: update, entity_id: "FEAT-XXX")`
is called without a `graph_project` parameter, then `GraphProject` MUST remain
`"kanbanzai-FEAT-XXX"`.

**AC-006 (FR-004):** Given a feature with a worktree whose `GraphProject` is
`"kanbanzai-FEAT-XXX"`, when `handoff(task_id)` is called for a task belonging
to that feature, then the output MUST contain a `## Code Graph` section that
includes the string `"kanbanzai-FEAT-XXX"`, at least four tool call examples
(`search_graph`, `trace_call_path`, `query_graph`, `get_code_snippet`), a
preference instruction for graph tools, and a re-indexing instruction.

**AC-007 (FR-005):** Given a feature with a worktree whose `GraphProject` is
the empty string, when `handoff(task_id)` is called for a task belonging to
that feature, then the output MUST contain a `## Code Graph` section that
includes an `index_repository` instruction with the worktree path.

**AC-008 (FR-006):** Given a feature with no worktree, when `handoff(task_id)`
is called for a task belonging to that feature, then the output MUST NOT
contain a `## Code Graph` section.

**AC-009 (FR-007):** Given a feature with both role-tool-hints and a non-empty
`GraphProject`, when `handoff(task_id)` is called, then the `## Code Graph`
heading MUST appear after the `## Available Tools` heading in the output.

**AC-010 (FR-008):** Given a feature with a worktree whose `GraphProject` is
`"kanbanzai-FEAT-XXX"`, when `next(id)` is called for that feature, then the
structured output MUST contain `graph_project: "kanbanzai-FEAT-XXX"`.

**AC-011 (FR-008):** Given a feature with no worktree, when `next(id)` is
called, then the structured output MUST contain `graph_project: ""`.

**AC-012 (FR-009, FR-010):** Given a feature with an active worktree whose
`GraphProject` is the empty string, when `status(id: featureID)` is called,
then the attention array MUST contain an item with
`type: "missing_graph_index"` and `severity: "info"`.

**AC-013 (FR-010):** Given a feature with an active worktree whose
`GraphProject` is `"kanbanzai-FEAT-XXX"`, when `status(id: featureID)` is
called, then the attention array MUST NOT contain a `missing_graph_index` item.

**AC-014 (FR-010):** Given a feature with no worktree, when
`status(id: featureID)` is called, then the attention array MUST NOT contain
a `missing_graph_index` item.

**AC-015 (FR-011):** Given a worktree with `GraphProject` equal to
`"kanbanzai-FEAT-XXX"`, when `worktree(action: remove)` is called, then the
response MUST contain a note referencing `"kanbanzai-FEAT-XXX"` and
instructing the user to run `delete_project`.

**AC-016 (FR-011):** Given a worktree with `GraphProject` equal to the empty
string, when `worktree(action: remove)` is called, then the response MUST NOT
contain a graph project cleanup note.

**AC-017 (FR-012):** Given a worktree with `GraphProject` equal to
`"kanbanzai-FEAT-XXX"`, when `cleanup(action: execute)` removes it, then the
cleanup output MUST contain a note referencing `"kanbanzai-FEAT-XXX"` and
instructing the user to run `delete_project`.

**AC-018 (FR-013):** Given `codebase_memory_mcp` is not available in the
session, when any workflow operation is performed (handoff, next, status,
cleanup), then no errors MUST be raised and all non-graph behaviour MUST be
identical to before this feature.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Automated test | `TestWorktreeRecord_LegacyDeserialisation` — load a YAML record without `GraphProject`, assert empty string and no error |
| AC-002 | Automated test | `TestWorktreeCreate_WithGraphProject` — create with `graph_project` param, assert field persisted |
| AC-003 | Automated test | `TestWorktreeCreate_WithoutGraphProject` — create without param, assert empty string |
| AC-004 | Automated test | `TestWorktreeUpdate_SetGraphProject` — update empty record with `graph_project`, assert new value |
| AC-005 | Automated test | `TestWorktreeUpdate_PreservesGraphProject` — update without `graph_project` param, assert value unchanged |
| AC-006 | Automated test | `TestHandoff_GraphProjectSet` — handoff with non-empty `GraphProject`, assert `## Code Graph` section with project name, four tool examples, preference instruction, and re-index instruction |
| AC-007 | Automated test | `TestHandoff_GraphProjectEmpty` — handoff with empty `GraphProject` and active worktree, assert `## Code Graph` section with `index_repository` instruction |
| AC-008 | Automated test | `TestHandoff_NoWorktree` — handoff with no worktree, assert no `## Code Graph` section |
| AC-009 | Automated test | `TestHandoff_SectionOrdering` — handoff with both tool hints and graph project, assert `## Code Graph` follows `## Available Tools` |
| AC-010 | Automated test | `TestNext_GraphProjectInOutput` — next with non-empty `GraphProject`, assert `graph_project` field in structured output |
| AC-011 | Automated test | `TestNext_GraphProjectEmptyNoWorktree` — next with no worktree, assert `graph_project` is empty string |
| AC-012 | Automated test | `TestStatus_MissingGraphIndex` — status on feature with worktree and empty `GraphProject`, assert `missing_graph_index` info attention item |
| AC-013 | Automated test | `TestStatus_NoMissingGraphIndex_ProjectSet` — status with non-empty `GraphProject`, assert no `missing_graph_index` item |
| AC-014 | Automated test | `TestStatus_NoMissingGraphIndex_NoWorktree` — status with no worktree, assert no `missing_graph_index` item |
| AC-015 | Automated test | `TestWorktreeRemove_GraphProjectNote` — remove worktree with non-empty `GraphProject`, assert cleanup note with project name and `delete_project` instruction |
| AC-016 | Automated test | `TestWorktreeRemove_NoGraphProjectNote` — remove worktree with empty `GraphProject`, assert no cleanup note |
| AC-017 | Automated test | `TestCleanup_GraphProjectNote` — cleanup execute removes worktree with non-empty `GraphProject`, assert note in output |
| AC-018 | Automated test | `TestWorkflow_NoGraphToolsAvailable` — run handoff, next, status, cleanup without `codebase_memory_mcp`, assert no errors and identical non-graph behaviour |