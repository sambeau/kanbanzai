# Dev-Plan: Inline skills index in status() orientation block

## Overview

Single-task implementation: modify `synthesiseProject` in `internal/mcp/status_tool.go` to surface skills in the orientation block.

## Task Breakdown

| # | Task | File | Description |
|---|------|------|-------------|
| 1 | Add skills index to orientation block | `internal/mcp/status_tool.go` | Read skills from `.agents/skills/`, render name + summary inline, add context-aware suggestion |

## Dependency Graph

One task, no dependencies.

## Interface Contracts

No interface changes. The orientation block format in `status()` output changes.

## Traceability Matrix

| Task | Spec Requirement |
|------|-----------------|
| 1 | FR-1, FR-2, FR-3, FR-4, NFR-1, AC-1, AC-2, AC-3 |
