# Specification: `write_file` MCP Tool for Worktree-Safe File Writing

**Feature:** FEAT-01KPQ08Y47522
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Status:** Draft

---

## 1. Overview

This specification defines the `write_file` MCP tool: a new server-side tool that accepts a file path and full UTF-8 content as first-class JSON string parameters and writes the file atomically to the correct filesystem root — either the active worktree directory for a given entity or the repository root. The tool eliminates shell-quoting failures that arise when agents write Go source files inside Git worktrees using `python3 -c` or heredoc workarounds.

---

## 2. Scope

### In Scope

- A new `write_file` MCP tool registered in `GroupGit` in `internal/mcp/`.
- Path resolution against the repo root (default) or an active worktree's directory (`entity_id` supplied).
- Path traversal prevention: any resolved path that escapes its designated root is rejected.
- Atomic write: content is written to a temp file and renamed into place.
- Automatic intermediate directory creation before writing.
- Tool registration alongside `WorktreeTool` in `newServerWithConfig`.
- A `GetByEntityID` lookup method on `worktree.Store` (or equivalent) if one does not already exist.

### Out of Scope

- Making `edit_file` worktree-aware.
- Diff-based or patch-based partial edits.
- Binary file support.
- Line-ending normalisation.
- File permission escalation or access control beyond existing OS permissions.
- An optional `mode` parameter for executable scripts.

---

## 3. Functional Requirements

**FR-001:** The `write_file` tool MUST accept three parameters: `path` (string, required), `content` (string, required), and `entity_id` (string, optional).

**FR-002:** When `entity_id` is absent, the tool MUST resolve `path` relative to the repository root (`repoRoot`).

**FR-003:** When `entity_id` is present, the tool MUST look up the active worktree record for that entity. It MUST use the worktree record's `Path` field (the on-disk checkout directory) as the resolution root. If no active worktree is found for the given `entity_id`, the tool MUST return a `worktree_not_found` error and perform no filesystem operation.

**FR-004:** If `path` is relative, the tool MUST join it onto the determined root before validation. If `path` is absolute, it MUST still be validated against the root.

**FR-005:** After resolving the path, the tool MUST call `filepath.Clean` on the result, then verify that the cleaned path has the root directory as a prefix (with a trailing path separator). If the resolved path escapes the root, the tool MUST return a `path_traversal` error and perform no filesystem operation.

**FR-006:** Before writing, the tool MUST call `os.MkdirAll` on the parent directory of the resolved path with mode `0o755`. If this fails, the tool MUST return an error.

**FR-007:** The tool MUST write file content atomically using `fsutil.WriteFileAtomic` with permission bits `0o644`. `WriteFileAtomic` writes to a temp file and renames into place; partial writes on crash or interruption are prevented.

**FR-008:** On success, the tool MUST return a JSON response containing the resolved absolute path (`"path"`) and the number of bytes written (`"bytes"`).

**FR-009:** The tool MUST return a `missing_parameter` error if `path` is empty or if `content` is absent from the request.

**FR-010:** The tool MUST be implemented in a single file `internal/mcp/write_file_tool.go`, following the one-file-per-tool convention of the `internal/mcp/` package.

**FR-011:** The tool MUST be registered in `GroupGit` within `newServerWithConfig`, immediately after `WorktreeTool`.

**FR-012:** The tool constructor MUST have the signature `WriteFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool`.

**FR-013:** The tool MUST carry hint flags: `ReadOnlyHint: false`, `DestructiveHint: false`, `IdempotentHint: false`, `OpenWorldHint: false`.

**FR-014:** `worktree.Store` MUST expose a method to look up an active worktree record by `entity_id`. If this method does not exist, it MUST be added as a prerequisite.

---

## 4. Non-Functional Requirements

**NFR-001:** The tool MUST NOT introduce any dependency on `service.EntityService`, `service.DocumentService`, or any planning-layer service. Its dependency graph is flat: MCP layer → `fsutil` + `worktree.Store`.

**NFR-002:** Content is treated as a UTF-8 string and written verbatim. The tool MUST NOT normalise line endings or perform any content transformation.

**NFR-003:** Error responses MUST use the `inlineErr(code, message)` convention used by all other tools in `internal/mcp/`.

**NFR-004:** Path traversal prevention MUST use `strings.HasPrefix` on the cleaned path against the root (with trailing separator appended to the root). Symlinks within the worktree that point outside the root are not the tool's responsibility to detect.

---

## 5. Acceptance Criteria

**AC-01 — Basic write to repo root:**
Calling `write_file` with a valid relative `path` and `content`, without `entity_id`, creates the file at `<repoRoot>/<path>` with the exact content provided. The response contains the resolved absolute path and byte count.

**AC-02 — Write to worktree:**
Calling `write_file` with a valid `path`, `content`, and a valid `entity_id` whose active worktree exists creates the file at `<worktreePath>/<path>`. The file in the main repo working tree is unaffected.

**AC-03 — Directory auto-creation:**
Calling `write_file` with a `path` whose parent directories do not exist causes those directories to be created before the file is written. The call succeeds.

**AC-04 — Atomic write:**
If the process is interrupted after the temp file is written but before the rename, no partial file appears at the target path.

**AC-05 — Path traversal rejected:**
Calling `write_file` with `path = "../../etc/passwd"` (or any path resolving outside the root) returns a `path_traversal` error. No filesystem write is performed.

**AC-06 — Missing `entity_id` worktree:**
Calling `write_file` with an `entity_id` that has no active worktree returns a `worktree_not_found` error. No filesystem write is performed.

**AC-07 — Empty path rejected:**
Calling `write_file` with an empty `path` returns a `missing_parameter` error.

**AC-08 — Missing content rejected:**
Calling `write_file` without a `content` parameter returns a `missing_parameter` error.

**AC-09 — Go source files:**
A Go source file containing backticks, single quotes, and double quotes in string literals and doc comments is written correctly. The content on disk matches the input byte-for-byte.

**AC-10 — Permission bits:**
Files written by the tool have permission `0o644` (owner read/write, group and other read-only).

---

## 6. Dependencies and Assumptions

- `internal/fsutil.WriteFileAtomic` exists and is correct. No changes to it are required.
- `worktree.Store` must support lookup by `entity_id`. If it only supports lookup by worktree record ID, adding `GetByEntityID` (or equivalent) is a prerequisite task for the implementer.
- `github.com/mark3labs/mcp-go` is already vendored; `mcp.NewTool`, `mcp.WithString`, and `server.ServerTool` are available.
- `internal/mcp.inlineErr` is a package-private helper available within `internal/mcp/`.
- The `repoRoot` string is already threaded through `newServerWithConfig` and available to all `GroupGit` tool constructors.
- The tool runs with the same OS-level permissions as the MCP server process; no privilege elevation is assumed.