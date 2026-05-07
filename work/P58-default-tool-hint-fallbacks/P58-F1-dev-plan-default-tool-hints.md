# Dev Plan: Default Tool Hint Fallbacks

## Overview

One task: add `defaultToolHints` map and fallback clause to `stepResolveToolHint`.

## Task Breakdown

| Task | File | Change |
|------|------|--------|
| T1 | `internal/context/pipeline.go` | Add `defaultToolHints` map (~40 lines), modify `stepResolveToolHint` to fall back (~5 lines) |

## Dependencies

None. Single file, single task.

## Dependency Graph

No dependencies. Single standalone task.

## Interface Contracts

No interface changes. The `stepResolveToolHint` method signature is unchanged. The `defaultToolHints` map is an unexported package-level variable.

## Traceability Matrix

| FR | Task | Verification |
|----|------|-------------|
| FR-01 | T1 | AC-01, AC-02 |
| FR-02 | T1 | AC-02 |
| FR-03 | T1 | AC-01 |

## Verification

`go test ./...` in `internal/context/` and `internal/mcp/`.
