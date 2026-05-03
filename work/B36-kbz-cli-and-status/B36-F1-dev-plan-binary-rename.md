| Field  | Value                                   |
|--------|-----------------------------------------|
| Date   | 2026-04-30                              |
| Status | approved |
| Author | architect                               |

## Overview

This plan implements the binary rename from `kanbanzai` to `kbz` as specified in `work/spec/B36-F1-spec-binary-rename.md` (doc ID: `FEAT-01KQ2VF7PDZ2G/spec-b36-f1-spec-binary-rename`). It covers renaming the main package directory and build targets, updating MCP config templates with a version bump and migration detection, and sweeping documentation references. The 15 functional requirements and 15 acceptance criteria from the specification are consolidated into 5 vertical-slice tasks with clear dependency ordering.

## Scope

This plan implements the requirements defined in `work/spec/B36-F1-spec-binary-rename.md` (doc ID: `FEAT-01KQ2VF7PDZ2G/spec-b36-f1-spec-binary-rename`). It covers the binary rename path (Part A of the design). It does **not** cover the `kbz status` extension (B36-F2–F4), the Go module path, the MCP server protocol name `"kanbanzai"`, or any deprecation shim.

## Task Breakdown

### Task 1: Rename cmd/kanbanzai/ to cmd/kbz/ and update build system

- **Description:** Rename the main package directory from `cmd/kanbanzai/` to `cmd/kbz/`. Update the Makefile `BINARY` variable and `go build` / `go install` targets to point at `./cmd/kbz`. Verify `make build` produces a binary named `kbz`.
- **Deliverable:** `cmd/kbz/` source tree with all existing files; updated `Makefile`; `make build` produces `kbz` binary.
- **Depends on:** None.
- **Effort:** Medium (3 story points).
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004.

### Task 2: Update MCP config templates (command + mcpVersion bump)

- **Description:** In `internal/kbzinit/mcp_config.go`, change the `"command"` field from `"kanbanzai"` to `"kbz"` in both the `.mcp.json` template (`writeMCPConfig`) and the `.zed/settings.json` template (`writeZedConfig`). Bump the `mcpVersion` constant from `1` to `2` so managed configs carrying the old version are rewritten on next `kbz init`. Update all test fixtures in `internal/kbzinit/init_test.go` that assert `"command": "kanbanzai"` to expect `"command": "kbz"` instead. The server key (`"kanbanzai"` in `mcpServers` / `context_servers`) and per-tool permission keys must remain unchanged.
- **Deliverable:** Updated `mcpConfig` and `zedConfig` templates; `mcpVersion = 2`; updated test expectations.
- **Depends on:** Task 1 (binary rename completes first to avoid confusion).
- **Effort:** Medium (3 story points).
- **Spec requirements:** FR-005, FR-006, FR-007, FR-014, FR-015.

### Task 3: Add migration detection to kbz init

- **Description:** Add detection logic in `internal/kbzinit/mcp_config.go` that inspects existing managed configs for stale `"command": "kanbanzai"` values. When detected during `kbz init` or `kbz init --update-skills`, emit a human-readable warning to stdout instructing the user to re-run `kbz init`. The `.mcp.json` path uses the `_managed` marker for detection; the `.zed/settings.json` path inspects the `context_servers.kanbanzai.command` field. The warning must fire even when `mcpVersion` has already bumped the rewrite path — it serves as a user-visible confirmation that the migration occurred. Add test cases that seed a temp dir with stale configs, capture stdout, and assert the warning appears.
- **Deliverable:** Migration detection in `mcp_config.go`; test coverage in `init_test.go`.
- **Depends on:** Task 2 (needs the new `mcpVersion` and `"kbz"` command as the canonical value to compare against).
- **Effort:** Medium (5 story points).
- **Spec requirements:** FR-008, FR-009, NFR-003.

### Task 4: Documentation sweep

- **Description:** Update every documentation reference to the binary name from `kanbanzai` to `kbz` in: `AGENTS.md`, `README.md`, `docs/getting-started.md`, and all other files under `docs/` that contain shell command examples or install instructions. The `go install` line becomes `go install github.com/sambeau/kanbanzai/cmd/kbz@latest`. Prose references to "Kanbanzai" (the system name) and the server protocol name `"kanbanzai"` in config keys remain unchanged. Review each file for context to distinguish binary references from system-name references.
- **Deliverable:** Updated `AGENTS.md`, `README.md`, and `docs/` files with zero stale binary-name references in shell commands or install instructions.
- **Depends on:** Task 1 (so documentation matches the actual binary name).
- **Effort:** Medium (3 story points).
- **Spec requirements:** FR-010, FR-011, FR-012, FR-013.

### Task 5: Verification and regression suite

- **Description:** Run the full test suite (`go test ./...`) and verify zero failures. Verify preservation invariants: (a) Go module path and all `internal/` imports are unchanged (scan all `.go` files), (b) MCP server protocol name `"kanbanzai"` in `mcpServers` / `context_servers` keys is unchanged, (c) `ServerName` constant in `internal/mcp/server.go` is unchanged. Verify all 15 acceptance criteria pass: binary output name, config template content, migration warnings, documentation references, and test suite health. Fix any breakage discovered.
- **Deliverable:** Passing `go test ./...`; verification report confirming all 15 ACs pass.
- **Depends on:** Tasks 1–4 (all changes must be complete before final verification).
- **Effort:** Medium (3 story points).
- **Spec requirements:** FR-014, FR-015, NFR-001; validates all ACs.

## Dependency Graph

```
Task 1 (rename cmd + Makefile)     — no dependencies
Task 2 (update templates + version) — depends on Task 1
Task 3 (migration detection)        — depends on Task 2
Task 4 (documentation sweep)        — depends on Task 1
Task 5 (verification)              — depends on Tasks 1, 2, 3, 4

Parallel groups: [Task 2, Task 4] after Task 1 completes
Critical path: Task 1 → Task 2 → Task 3 → Task 5
```

## Risk Assessment

### Risk: Stale binary-name references in non-obvious locations

- **Probability:** Medium.
- **Impact:** Low (missed references cause confusion but not breakage; CI grep checks catch most).
- **Mitigation:** The documentation sweep (Task 4) uses `grep -r` across the entire repository plus `docs/` to locate all references. The verification task (Task 5) re-scans as a final gate.
- **Affected tasks:** Task 4, Task 5.

### Risk: Test fixtures with hardcoded `"kanbanzai"` command strings cause unexpected failures

- **Probability:** Medium.
- **Impact:** Medium (test failures block CI and task completion).
- **Mitigation:** Task 2 explicitly updates test fixtures in `init_test.go`. Task 5 runs the full suite and catches any remaining fixtures. The test fixture updates are surgical: only `"command"` values change; server key names and tool-permission keys stay the same.
- **Affected tasks:** Task 2, Task 5.

### Risk: Migration detection interacts poorly with version-gating in writeJSONConfig

- **Probability:** Low.
- **Impact:** Medium (migration warning might fire spuriously or not at all).
- **Mitigation:** The migration detection sits after the version-gating path in `writeMCPConfig` / `writeZedConfig`. It compares the existing file's command field against `"kbz"` (the canonical value) and fires only when a stale value is found. The version bump to `2` guarantees the rewrite path executes, and the warning adds a user-visible confirmation after rewrite. Task 3 includes test cases for both the `.mcp.json` and `.zed/settings.json` detection paths.
- **Affected tasks:** Task 3.

## Interface Contracts

The rename is self-contained within one Go module — there are no cross-service or cross-module interface changes. The internal interfaces that matter for correctness:

- **`internal/kbzinit.Initializer` → filesystem:** `writeMCPConfig` and `writeZedConfig` write `.mcp.json` and `.zed/settings.json` respectively. The schema of these files changes only in the `"command"` field value (`"kbz"` replacing `"kanbanzai"`). The `_managed` block schema (`tool`, `version`) is unchanged. The `context_servers` key schema is unchanged.
- **`cmd/kbz/main.go` → `internal/mcp.ServerName`:** The `ServerName` constant (`"kanbanzai"`) is preserved per FR-015. The binary rename does not touch this value.
- **Go module path:** `github.com/sambeau/kanbanzai` and all `internal/...` import paths are preserved per FR-014. No `go.mod` changes.
- **Makefile → `go build`:** The build target changes from `./cmd/kanbanzai` to `./cmd/kbz`. The `BINARY` variable changes from `kanbanzai` to `kbz`. No flags or ldflags change.

## Traceability Matrix

| Task | FRs Covered | ACs Verified |
|------|-------------|-------------|
| Task 1: Rename cmd + build system | FR-001, FR-002, FR-003, FR-004 | AC-001, AC-002, AC-003 |
| Task 2: Update MCP templates + version | FR-005, FR-006, FR-007, FR-014, FR-015 | AC-004, AC-005, AC-006, AC-014 |
| Task 3: Migration detection | FR-008, FR-009, NFR-003 | AC-007, AC-008 |
| Task 4: Documentation sweep | FR-010, FR-011, FR-012, FR-013 | AC-009, AC-010, AC-011, AC-012 |
| Task 5: Verification and invariants | FR-014, FR-015, NFR-001 | AC-013, AC-015, all above |

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-001 (binary named `kbz` after `make build`) | Build test: `make build && test -f kbz` | Task 1, Task 5 |
| AC-002 (`make install` produces `kbz`) | Build test: `make install && which kbz` | Task 1, Task 5 |
| AC-003 (`cmd/kbz/main.go` exists, `cmd/kanbanzai/` does not) | File scan | Task 1, Task 5 |
| AC-004 (`.mcp.json` template has `"command": "kbz"`) | Unit test: `kbz init` on temp dir, parse `.mcp.json` | Task 2, Task 5 |
| AC-005 (`.zed/settings.json` template has `"command": "kbz"`) | Unit test: `kbz init` on temp dir, parse `.zed/settings.json` | Task 2, Task 5 |
| AC-006 (old-version `.mcp.json` gets rewritten with new `mcpVersion`) | Unit test: seed old managed `.mcp.json`, run `kbz init`, assert rewrite | Task 2, Task 5 |
| AC-007 (stale `.mcp.json` `"command": "kanbanzai"` triggers warning on `--update-skills`) | Unit test: seed stale config, run init, capture stdout | Task 3, Task 5 |
| AC-008 (stale `.zed/settings.json` `"command": "kanbanzai"` triggers warning) | Unit test: seed stale config, run init, capture stdout | Task 3, Task 5 |
| AC-009 (no binary `kanbanzai` references in `AGENTS.md`) | File scan: `grep` for binary invocations | Task 4, Task 5 |
| AC-010 (no binary `kanbanzai` references in README) | File scan: `grep` for binary invocations | Task 4, Task 5 |
| AC-011 (no binary `kanbanzai` references in `docs/`) | File scan: `grep -r` in `docs/` | Task 4, Task 5 |
| AC-012 (`go install` line references `cmd/kbz`) | File scan: locate install instruction | Task 4, Task 5 |
| AC-013 (all import paths unchanged) | Static check: `grep -r 'github.com/sambeau/kanbanzai/internal' --include='*.go'` | Task 5 |
| AC-014 (server key is `"kanbanzai"`, not `"kbz"`) | Unit test: after `kbz init`, assert server key in configs | Task 2, Task 5 |
| AC-015 (full test suite passes) | `go test ./...` with zero failures | Task 5 |
