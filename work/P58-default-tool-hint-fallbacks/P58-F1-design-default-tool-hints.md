| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07T14:34:00Z           |
| Status | Approved                       |
| Author | sambeau                        |

## Overview

Add hardcoded default tool hints to the 3.0 context assembly pipeline so sub-agent prompts always include `codebase_memory_mcp_search_code`, `kanbanzai_edit_file`, `search_graph`, and other role-scoped tools — even when no `tool_hints` config exists.

## Goals and Non-Goals

**Goals:** One hardcoded `defaultToolHints` map in `internal/context/pipeline.go`, modified `stepResolveToolHint` to fall back when `MergedToolHints` is empty.

**Non-Goals:** No config changes, no pipeline restructure, no new MCP tools.

## Design

See `work/P44-model-routing-agent-launcher/P44-F1-design-prompt-assembly-gate.md` §2.2 ("Change 1: Hardcoded default tool hints") for the full design. P58 implements exactly that change in isolation.

Scope: `internal/context/pipeline.go` — one map, one fallback clause in `stepResolveToolHint`. ~50 lines.

## Alternatives Considered

Per the full P44 design §3, the hardcoded-defaults approach was chosen over config-only and prompter-sub-agent alternatives.

## Dependencies

None. This is a standalone change to the pipeline.
