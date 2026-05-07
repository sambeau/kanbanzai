# Specification: Default Tool Hint Fallbacks

## Overview

Add hardcoded default tool hints to `stepResolveToolHint` in the 3.0 context assembly pipeline. When no tool hints are configured (the current default), fall back to a built-in map covering the standard roles.

## Scope

`internal/context/pipeline.go` only. No new files, no config changes, no API changes.

## Functional Requirements

**FR-01.** A `defaultToolHints` map is defined in `pipeline.go` covering implementer-go, architect, reviewer-conformance, reviewer-quality, reviewer-security, and reviewer-testing. Each entry is a Markdown-formatted tool list including at minimum `search_graph`, `codebase_memory_mcp_search_code`, `get_code_snippet`, `kanbanzai_edit_file`, and `write_file`.

**FR-02.** `stepResolveToolHint` falls back to `defaultToolHints` when `MergedToolHints` is empty or the resolved role has no entry in merged hints. Config-supplied hints always take precedence.

**FR-03.** The "Available Tools" section (Position 6) appears in pipeline output for `implementer-go` role even when `.kbz/config.yaml` has no `tool_hints` entries.

## Acceptance Criteria

- [ ] AC-01. `handoff(task_id, role: "implementer-go")` returns a prompt containing `search_graph` and `kanbanzai_edit_file` in an "Available Tools" section when no tool hints are configured.
- [ ] AC-02. When `tool_hints.implementer-go` is set in config, the config value is used instead of the default.
- [ ] AC-03. `go test ./...` passes in `internal/context/` and `internal/mcp/`.
