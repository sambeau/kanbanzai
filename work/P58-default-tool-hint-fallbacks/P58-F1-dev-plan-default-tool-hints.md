# Dev Plan: Default Tool Hint Fallbacks

## Overview

One task: add `defaultToolHints` map and fallback clause to `stepResolveToolHint`.

## Task Breakdown

| Task | File | Change |
|------|------|--------|
| T1 | `internal/context/pipeline.go` | Add `defaultToolHints` map (~40 lines), modify `stepResolveToolHint` to fall back (~5 lines) |

## Dependencies

None. Single file, single task.

## Verification

`go test ./...` in `internal/context/` and `internal/mcp/`.
