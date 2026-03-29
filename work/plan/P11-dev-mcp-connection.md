# P11 Feature A: MCP Connection — Dev Plan

| Document | P11 Feature A: MCP Connection — Dev Plan            |
|----------|-----------------------------------------------------|
| Feature  | FEAT-01KMWJ3ZQ57ZY mcp-connection                   |
| Status   | Draft                                               |
| Related  | `work/spec/spec-mcp-connection.md`                  |
|          | `work/design/fresh-install-experience.md` §5.1      |

---

## 1. Approach

Extend `kbz init` to write two editor configuration files as part of every new-project and
existing-project init run. Both files carry a `_managed` marker so future runs can detect
whether they own the file and whether the schema has advanced. The managed marker is a
top-level `"_managed": { "tool": "kanbanzai", "version": 1 }` JSON field.

All writing logic lives in a new `internal/kbzinit/mcp_config.go` file so the two writers can
share the conflict-detection helper. A single `--skip-mcp` flag gates both writers and is
plumbed through `Options` in `init.go` and the CLI flag parser in `init_cmd.go`.

The `kanbanzai-getting-started` skill changes (AC-17 – AC-20) are verified by the test suite
but assumed to already be committed; Task 3 includes a confirmation step, not implementation.

---

## 2. Task Breakdown

| # | Task | Files touched | Size |
|---|------|---------------|------|
| 1 | Implement `.mcp.json` writer | `internal/kbzinit/mcp_config.go` (new) — `writeMCPConfig`, shared `managedMarker` helpers; `internal/kbzinit/init.go` — call `writeMCPConfig` from `runNewProject` and `runExistingProject` | M |
| 2 | Implement `.zed/settings.json` writer | `internal/kbzinit/mcp_config.go` — add `writeZedConfig`; only writes when `.zed/` directory exists at `baseDir`; reuses same version-aware conflict logic as Task 1 | S |
| 3 | Add `--skip-mcp` flag and tests | `internal/kbzinit/init.go` — add `SkipMCP bool` to `Options`, gate both writers; `cmd/kanbanzai/init_cmd.go` — add `--skip-mcp` flag; `internal/kbzinit/init_test.go` — tests for AC-01 – AC-16; confirm AC-17 – AC-20 pass against committed skill file | M |

### Conflict logic (shared by both writers)

All three cases apply equally to `.mcp.json` and `.zed/settings.json`:

| Existing file state | Action |
|---------------------|--------|
| No `_managed` key, or `_managed.tool` ≠ `"kanbanzai"` | Skip write; print warning naming the file |
| `_managed.version` < current schema version | Overwrite with current content |
| `_managed.version` == current schema version | No-op; no message emitted |

---

## 3. Dependencies

- **No cross-feature dependencies.** All three tasks are self-contained within
  `internal/kbzinit/` and `cmd/kanbanzai/`.
- Task 2 depends on Task 1 (shares helpers in `mcp_config.go`).
- Task 3 depends on Tasks 1 and 2 (tests exercise both writers and the skip flag).
- The getting-started skill content (AC-17 – AC-20) is a prerequisite assumed already
  committed; Task 3 only asserts it in tests.