# Implementation Plan: `write_file` MCP Tool for Worktree-Safe File Writing

**Feature:** FEAT-01KPQ08Y47522
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Specification:** work/spec/p25-write-file-tool.md

---

## Overview

Three tasks in dependency order. Task 1 extends `worktree.Store` with an entity-scoped lookup
method. Task 2 implements the tool itself and registers it. Task 3 writes tests.
Tasks 2 and 3 depend on Task 1; Task 3 depends on Task 2.

---

## Interface Contract

The following must be agreed before Tasks 2 and 3 begin work:

**`worktree.Store.GetByEntityID`**
```go
// GetByEntityID returns the active worktree record for the given entity ID,
// or (nil, nil) when no active worktree exists for that entity.
func (s *Store) GetByEntityID(entityID string) (*WorktreeRecord, error)
```

**`WriteFileTool` constructor**
```go
func WriteFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool
```

**MCP response on success**
```json
{ "path": "<resolved absolute path>", "bytes": <int> }
```

**Error codes**
| Code | Condition |
|------|-----------|
| `missing_parameter` | `path` is empty or `content` is absent |
| `worktree_not_found` | `entity_id` supplied but no active worktree exists |
| `path_traversal` | Resolved path escapes designated root |

---

## Task Breakdown

### Task 1: Add `GetByEntityID` to `worktree.Store`

- **Description:** Extend `worktree.Store` with a method that looks up the active worktree
  record for a given entity ID. This is the prerequisite for the tool to resolve paths against
  the correct worktree directory.
- **Spec references:** FR-003, FR-014
- **Input context:**
  - `internal/worktree/` — the worktree store implementation; read the existing store methods
    (especially how active-status filtering and record retrieval work) before writing the new one.
  - `internal/model/` — the `WorktreeRecord` type and its `EntityID` and `Status` fields.
- **Output artifacts:**
  - Modified `internal/worktree/store.go` (or the relevant store file): new exported method
    `GetByEntityID(entityID string) (*WorktreeRecord, error)` that queries all records,
    filters for `status == "active"` and `EntityID == entityID`, and returns the first match
    or `(nil, nil)` if none found.
  - New test in the appropriate `_test.go` file asserting the method returns the correct record
    when an active worktree exists for the entity, and returns `(nil, nil)` when none exists.
- **Dependencies:** None
- **Effort:** Small

---

### Task 2: Implement `WriteFileTool` and register in `GroupGit`

- **Description:** Create `internal/mcp/write_file_tool.go` implementing the `write_file`
  MCP tool with the full path-resolution, traversal-prevention, directory-creation, and
  atomic-write behaviour specified. Register the tool in `GroupGit` immediately after
  `WorktreeTool`.
- **Spec references:** FR-001 through FR-013
- **Input context:**
  - `internal/mcp/` — study `worktree_tool.go` (and one other tool file) for the one-file-per-tool
    pattern, `inlineErr` usage, `mcp.NewTool` / `mcp.WithString` API, and hint flags.
  - `internal/mcp/server.go` — `newServerWithConfig` function where `GroupGit` tools are
    registered; find the `WorktreeTool` registration site for the insertion point.
  - `internal/fsutil/` — `WriteFileAtomic` function signature and behaviour.
  - `internal/worktree/store.go` — `GetByEntityID` interface from Task 1.
  - `internal/config/` — `GroupGit` constant.
  - Spec FR-005: traversal check uses `filepath.Clean` + `strings.HasPrefix` with trailing
    separator on root.
  - Spec FR-006: `os.MkdirAll(parentDir, 0o755)` before writing.
  - Spec FR-007: `fsutil.WriteFileAtomic(resolvedPath, []byte(content), 0o644)`.
- **Output artifacts:**
  - New file `internal/mcp/write_file_tool.go` implementing `WriteFileTool`.
  - Modified `internal/mcp/server.go`: `WriteFileTool(repoRoot, worktreeStore)` call added
    to `GroupGit` section immediately after `WorktreeTool`.
- **Dependencies:** Task 1 (requires `GetByEntityID` on `worktree.Store`)
- **Effort:** Medium

---

### Task 3: Write tests for `WriteFileTool`

- **Description:** Write the full test suite for `write_file_tool.go` covering all acceptance
  criteria from the specification.
- **Spec references:** AC-01 through AC-10
- **Input context:**
  - `internal/mcp/write_file_tool.go` from Task 2.
  - Existing MCP tool tests (e.g. `worktree_tool_test.go`) for test harness patterns,
    how to invoke tool handlers, and how to assert MCP response JSON.
  - The agreed error codes from the Interface Contract above.
- **Output artifacts:**
  - New file `internal/mcp/write_file_tool_test.go` with test cases:
    - AC-01: basic write to repo root with valid path + content (no entity_id)
    - AC-02: write to worktree path when entity_id is supplied and active worktree exists
    - AC-03: intermediate directories created when they don't exist
    - AC-05: path traversal (`../../etc/passwd`) returns `path_traversal` error
    - AC-06: entity_id with no active worktree returns `worktree_not_found` error
    - AC-07: empty path returns `missing_parameter` error
    - AC-08: absent content returns `missing_parameter` error
    - AC-09: Go source file content with backticks, single quotes, and double quotes
      is written byte-for-byte correctly
    - AC-10: written file has permission `0o644`
- **Dependencies:** Task 2
- **Effort:** Medium