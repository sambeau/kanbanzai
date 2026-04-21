# Design: `write_file` MCP Tool for Worktree-Safe File Writing

**Feature:** FEAT-01KPQ08Y47522
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Status:** Draft

---

## Overview

Agents working inside a Git worktree cannot use `edit_file` for file writes because that tool operates on the main working tree. The workarounds agents currently use — `python3 -c` with triple-quoted strings, or heredoc `cat << 'EOF'` — are fragile for Go source files: triple-quote collisions break the Python path; a bare `EOF` line breaks the heredoc path; both embed entire file contents in a shell command, inflating context-window cost.

This design proposes a new `write_file` MCP tool that accepts `path` and `content` as first-class JSON string parameters, resolves the path against the correct filesystem root (repo root or active worktree), and writes the file atomically. No shell quoting is involved; the server receives the content as a parsed JSON string and writes it verbatim.

---

## Goals and Non-Goals

### Goals

- Provide a reliable mechanism for writing any file (Go source, Markdown, YAML, generated code) from within an agent context, including from inside a Git worktree.
- Eliminate shell-quoting failures by treating file content as a JSON string parameter, not as shell-embedded data.
- Write files atomically (write-to-temp-then-rename) to prevent partial writes on crash or interruption.
- Resolve paths safely: relative to the active worktree's filesystem root when a `worktree_id` is supplied, or relative to the repo root otherwise.
- Prevent path traversal: a resolved path that escapes its root must be rejected.
- Create intermediate directories automatically so agents can write into new subdirectories without a separate `mkdir` step.
- Register the tool within the existing `GroupGit` configuration group, alongside the other worktree-related tools.

### Non-Goals

- **Replacing `edit_file` for main-tree editing.** `edit_file` continues to work as-is for agents that are not in a worktree. `write_file` complements rather than replaces it.
- **Making `edit_file` worktree-aware.** That is a separate, higher-effort change (see Alternatives Considered).
- **Diff-based or patch-based edits.** `write_file` writes the entire file content. Partial edits remain the responsibility of `edit_file` or agent-side content manipulation.
- **Binary file support.** Content is a UTF-8 JSON string. Binary files are out of scope.
- **Access control or permission escalation.** The tool runs with the same OS permissions as the MCP server process; no privilege-elevation mechanism is added.
- **Line-ending normalisation.** Content is written verbatim; the agent is responsible for correct line endings.

---

## Design

### Component Boundaries

The new tool is a single self-contained file, `internal/mcp/write_file_tool.go`, consistent with the one-file-per-tool convention already established in `internal/mcp/`. It has two external dependencies:

1. **`internal/fsutil.WriteFileAtomic`** — the existing atomic write primitive. It creates a temp file in the same directory as the target, writes content, sets permissions, and renames into place. No new write logic is introduced.
2. **`internal/worktree.Store`** — the existing worktree record store. When a `worktree_id` is supplied, the tool looks up the worktree record to obtain its `Path` field (the filesystem directory of the checked-out branch), which becomes the resolution root.

The tool does not depend on `service.EntityService`, `service.DocumentService`, or any planning layer. Its dependency graph is flat: `mcp layer → fsutil + worktree.Store`.

### Parameter Schema

| Parameter | Type | Required | Description |
|---|---|---|---|
| `path` | string | yes | File path to write. Relative to repo root (no `worktree_id`) or to the worktree's checked-out directory (`worktree_id` supplied). Absolute paths are also accepted and must resolve within the appropriate root. |
| `content` | string | yes | Full file content to write, as a UTF-8 string. Written verbatim. |
| `worktree_id` | string | no | ID of an active worktree record (e.g. `WT-01JX…`). When supplied, `path` is resolved relative to that worktree's filesystem directory. When omitted, `path` is resolved relative to the repo root. |

Response on success:

```json
{
  "path": "<resolved absolute path>",
  "bytes": 1234
}
```

Response on failure uses the `inlineErr(code, message)` convention already used by all tools in `internal/mcp/`.

### Path Resolution

Path resolution follows a two-step contract:

1. **Determine the root directory.**
   - If `worktree_id` is absent: root = `repoRoot` (the same `repoRoot` string already threaded through every `GroupGit` tool constructor via `newServerWithConfig`).
   - If `worktree_id` is present: look up the worktree record in `worktree.Store`. If the record is not found or its status is not `active`, return an `inlineErr`. Root = the record's `Path` field (the on-disk worktree checkout directory).

2. **Resolve and validate the target path.**
   - If `path` is relative, join it onto root.
   - Call `filepath.Clean` on the result.
   - Verify that the cleaned path has root as a prefix (using `strings.HasPrefix` after ensuring root has a trailing separator). If not, return a `path_traversal` error and do not write.

This two-step contract means an agent can pass `internal/mcp/server.go` with a worktree ID and the tool writes to `.worktrees/feat-xxx/internal/mcp/server.go`, leaving the main working tree untouched.

### Directory Creation

Before invoking `WriteFileAtomic`, the tool calls `os.MkdirAll` on the target file's parent directory with mode `0o755`. `WriteFileAtomic` requires the directory to exist (it calls `os.CreateTemp(dir, …)` internally). Creating the directory as a prerequisite means agents can write files into new packages or subdirectories in a single tool call.

### Atomic Write

File content is passed as `[]byte` to `fsutil.WriteFileAtomic` with permission `0o644`. `WriteFileAtomic` handles the temp-file creation, write, chmod, and rename sequence. The deferred cleanup in `WriteFileAtomic` ensures the temp file is removed on any failure path.

### Error Handling

| Condition | Error Code | Agent Guidance |
|---|---|---|
| `path` is empty | `missing_parameter` | Provide a non-empty path. |
| `content` is missing from request | `missing_parameter` | Provide the content parameter. |
| `worktree_id` supplied but record not found | `worktree_not_found` | Verify the worktree ID with `worktree(action: "get")`. |
| `worktree_id` supplied but worktree is not active | `worktree_not_active` | Only active worktrees can be written to. |
| Resolved path escapes the root | `path_traversal` | Use a path relative to the worktree or repo root; do not use `..` segments that escape it. |
| `os.MkdirAll` fails | propagated as `tool error` | Likely a permissions issue on the host filesystem. |
| `WriteFileAtomic` fails | propagated as `tool error` | Includes the wrapped OS error for diagnosis. |

### Security Constraints

- **Path traversal prevention** is the primary security constraint. Any `path` argument that resolves to a location outside the designated root is rejected before any filesystem operation.
- The tool does **not** restrict by file extension or content. The MCP server runs under the developer's local user account; the constraint model mirrors what `terminal` already allows.
- The tool does **not** dereference symlinks for the traversal check. If a symlink within the worktree points outside the root, that is a host-level concern outside the scope of this tool.

### Tool Registration

`WriteFileTool` is registered in `GroupGit` in `newServerWithConfig`, immediately after `WorktreeTool`. The constructor signature follows the existing pattern:

```
WriteFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool
```

It carries `ReadOnlyHint: false`, `DestructiveHint: false` (it writes but does not delete), `IdempotentHint: false`, `OpenWorldHint: false`.

---

## Alternatives Considered

### 1. Make `edit_file` Worktree-Aware (Proposal P1)

**Approach:** Modify `edit_file` so that when the path falls inside a known active worktree directory (`.worktrees/<branch>/`), it applies the edit to that path rather than to the main working tree.

**What it makes easier:** No change required to agent skills or prompts — `edit_file` continues to be the one tool for file edits everywhere. The diff/patch edit model (rather than full-content replace) is preserved, which is more efficient for large files with small edits.

**What it makes harder:** `edit_file` is not part of the Kanbanzai codebase — it is provided by the outer IDE/agent runtime (Zed, Cursor, etc.). Making it worktree-aware requires changes in that external system, which is outside the project's control. Even if the outer runtime were modified, it must correctly identify active worktrees, which requires reading Kanbanzai's internal state.

**Why rejected for this feature:** This fix is the right long-term solution, but it requires changes to a third-party tool boundary and cannot be delivered as part of a Kanbanzai-internal change. `write_file` is a self-contained fix entirely within the project's control.

### 2. Base64-Encoded Write via `terminal`

**Approach:** Agents base64-encode file content client-side, pass it as a single hex/base64 string to a `terminal` command such as `echo '<base64>' | base64 -d > path`, and the shell decodes it.

**What it makes easier:** No new MCP tool is required. Base64-encoded content has no shell-quoting collisions.

**What it makes harder:** Base64 encoding is a mechanical transformation agents must apply to every file write. It increases token count (base64 output is ~33% larger than source). The agent must also select the correct `base64` flags (`-d` vs `-D` on macOS vs Linux), and the command fails silently on systems without `base64` in `PATH`. This is a new class of fragility, not a removal of it.

**Why rejected:** Exchanges one brittle shell pattern for another. Does not address the root cause (shell-mediated file writing). Adds cognitive and token overhead to every file write operation.

### 3. Status Quo — Update `implement-task/SKILL.md` Only (Proposal P2)

**Approach:** Leave the tool surface unchanged. Update the skill documentation to recommend heredoc (`cat > file << 'GOEOF'`) as the primary pattern for Go files, with fallback instructions for delimiter collisions.

**What it makes easier:** Zero implementation cost. Agents get clearer guidance immediately.

**What it makes harder:** Heredocs remain fragile for Go files containing a bare `GOEOF` line (or whatever delimiter is chosen). The problem is probabilistic, not eliminated. Every file write in a worktree still consumes extra context window for the shell scaffolding. Agents must remember the pattern, and the pattern differs by file type.

**Why rejected as a standalone fix:** P2 is a useful quick win and should be shipped regardless, but it does not resolve the root cause. Skill documentation can only mitigate; it cannot make the underlying mechanism reliable.

---

## Dependencies

| Dependency | Kind | Notes |
|---|---|---|
| `internal/fsutil.WriteFileAtomic` | Internal | Existing function. No changes required. |
| `internal/worktree.Store` | Internal | Existing store. `GetByID` or equivalent lookup method required; if absent, a lookup-by-ID method may need to be added to the store interface (implementation concern, not a design risk). |
| `github.com/mark3labs/mcp-go` | External (existing) | Already vendored. `mcp.NewTool`, `mcp.WithString`, `server.ServerTool` are used by every other tool. |
| `internal/mcp.inlineErr` | Internal | Package-private helper already used across all tools in the package. No new dependency. |
| `internal/config.GroupGit` | Internal | Registration group constant. No changes to group definitions required. |

### Open Questions

1. **`worktree.Store.GetByID` availability.** The worktree store is currently queried by entity ID in all existing tool code. Path resolution for `write_file` requires lookup by worktree record ID (the `WT-...` identifier). Does `worktree.Store` already expose a `GetByID` method, or does one need to be added? If the latter, that addition should be called out in the specification as a prerequisite task.

2. **`worktree_id` vs entity-scoped path.** The proposal sketch uses `worktree_id` (the record's own ID). An alternative is to accept `entity_id` (e.g. `FEAT-...`) and derive the worktree from that. `entity_id` may be more natural for agents since they already hold the entity ID in context. The specification should settle this before implementation.

3. **Permission bits.** `0o644` is assumed for all written files. Go source files and Markdown files are correctly handled by this. If the tool is later used to write executable scripts, the permission assumption will be wrong. For now, `0o644` is hardcoded; the spec may add an optional `mode` parameter if a concrete need is identified.
```

Now let me register this document: