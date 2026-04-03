# Worktree Graph Context — Development Plan

> Development plan for FEAT-01KNA-11F5FAFW (worktree-graph-context)
> Spec: work/spec/worktree-graph-context.md
> Design: work/design/codebase-memory-integration.md (Phase 2)

---

## Overview

This feature wires graph project awareness through the worktree system into agent context. A new `GraphProject` field on `worktree.Record` stores the `codebase_memory_mcp` project name. This value flows through `handoff` (as a `## Code Graph` prompt section with tool call examples), `next` (as a `graph_project` field in structured output), `status` (as a `missing_graph_index` attention item when empty), and `cleanup`/`remove` (as a note to delete the graph index). Kanbanzai never calls `codebase_memory_mcp` directly — it only emits metadata and instructions that agents act on.

Three tasks implement this in dependency order: field and params first, then context injection, then status/cleanup surfacing.

---

## Task Breakdown

| Task ID | Name | Description | Dependencies | Files |
|---------|------|-------------|--------------|-------|
| TASK-01KNA-M0VVWJT6 | GraphProject Field and Worktree Params | Add `GraphProject string` to `worktree.Record`; accept `graph_project` param in `worktree(action: create)` and `worktree(action: update)` | — | `internal/worktree/worktree.go`, `internal/mcp/worktree_tool.go`, `internal/worktree/worktree_test.go`, `internal/mcp/worktree_tool_test.go` |
| TASK-01KNA-M0VYTHP1 | Graph Context Handoff and Next | Three-state handoff logic (project set → examples, project empty → index instruction, no worktree → omit); section ordering after `## Available Tools`; `graph_project` field in `next` output | TASK-01KNA-M0VVWJT6 | `internal/context/pipeline.go`, `internal/mcp/handoff_tool.go`, `internal/mcp/next_tool.go`, `internal/context/pipeline_test.go`, `internal/mcp/handoff_tool_test.go`, `internal/mcp/next_tool_test.go` |
| TASK-01KNA-M0W16H7H | Graph Context Status and Cleanup | `missing_graph_index` info attention item in status when worktree exists but `GraphProject` is empty; cleanup/remove note when `GraphProject` is non-empty | TASK-01KNA-M0VVWJT6 | `internal/mcp/status_tool.go`, `internal/mcp/worktree_tool.go`, `internal/mcp/cleanup_tool.go`, `internal/mcp/status_tool_test.go`, `internal/mcp/worktree_tool_test.go`, `internal/mcp/cleanup_tool_test.go` |

---

## Dependency Graph

```
TASK-01KNA-M0VVWJT6 (GraphProject Field + Params)
├──► TASK-01KNA-M0VYTHP1 (Handoff + Next injection)
└──► TASK-01KNA-M0W16H7H (Status + Cleanup surfacing)
```

Tasks 2 and 3 are independent of each other and can run in parallel once task 1 is complete.

---

## Interface Contracts

### Worktree Record (task 1)

```go
// internal/worktree/worktree.go
type Record struct {
    // ... existing fields ...
    GraphProject string `yaml:"graph_project,omitempty"` // codebase_memory_mcp project name
}
```

### Worktree Tool Params (task 1)

`worktree(action: create)` and `worktree(action: update)` accept an optional `graph_project` string parameter. On create, omission means empty string. On update, omission preserves the existing value.

### Handoff Code Graph Section (task 2)

Three states:
1. **`GraphProject` non-empty** → `## Code Graph` section with project name, four tool call examples (`search_graph`, `trace_call_path`, `query_graph`, `get_code_snippet`), preference instruction, and re-indexing instruction. Must appear after `## Available Tools`.
2. **`GraphProject` empty, worktree exists** → `## Code Graph` section with `index_repository` instruction using worktree path.
3. **No worktree** → section omitted entirely.

### Next Structured Output (task 2)

```go
// internal/mcp/next_tool.go — nextContextToMap()
// Add alongside existing "worktree" field:
"graph_project": record.GraphProject  // empty string when unset or no worktree
```

### Status Attention Item (task 3)

```go
// When worktree exists and GraphProject == "":
AttentionItem{
    Type:     "missing_graph_index",
    Severity: "info",
    Message:  "Worktree has not been indexed for graph navigation. Run index_repository on the worktree path to enable graph-based code exploration.",
}
```

### Cleanup/Remove Note (task 3)

When `GraphProject` is non-empty, append a note to the response:
> Graph project `<name>` is no longer needed. Run `delete_project(project_name: "<name>")` to free the index.

---

## Traceability Matrix

| FR | Task | Acceptance Criteria |
|----|------|---------------------|
| FR-001 (GraphProject field, backward compat) | TASK-01KNA-M0VVWJT6 | AC-001 |
| FR-002 (create with graph_project) | TASK-01KNA-M0VVWJT6 | AC-002, AC-003 |
| FR-003 (update with graph_project) | TASK-01KNA-M0VVWJT6 | AC-004, AC-005 |
| FR-004 (handoff, project set) | TASK-01KNA-M0VYTHP1 | AC-006 |
| FR-005 (handoff, project empty) | TASK-01KNA-M0VYTHP1 | AC-007 |
| FR-006 (handoff, no worktree) | TASK-01KNA-M0VYTHP1 | AC-008 |
| FR-007 (section ordering) | TASK-01KNA-M0VYTHP1 | AC-009 |
| FR-008 (next graph_project field) | TASK-01KNA-M0VYTHP1 | AC-010, AC-011 |
| FR-009 (missing_graph_index attention) | TASK-01KNA-M0W16H7H | AC-012 |
| FR-010 (attention item suppression) | TASK-01KNA-M0W16H7H | AC-013, AC-014 |
| FR-011 (remove cleanup note) | TASK-01KNA-M0W16H7H | AC-015, AC-016 |
| FR-012 (cleanup cleanup note) | TASK-01KNA-M0W16H7H | AC-017 |
| FR-013 (graceful without codebase_memory_mcp) | TASK-01KNA-M0VYTHP1, TASK-01KNA-M0W16H7H | AC-018 |