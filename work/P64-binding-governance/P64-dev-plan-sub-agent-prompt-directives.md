# Dev-Plan: Sub-agent prompt dispatch and worktree-write directives

## Overview

Single-task implementation: add directive blocks to `RenderPrompt` in `internal/context/pipeline.go`.

## Task Breakdown

| # | Task | File | Description |
|---|------|------|-------------|
| 1 | Add dispatch and worktree-write directives | `internal/context/pipeline.go` | Conditionally insert directive blocks: dispatch contract for orchestrator-workers, worktree-write for active worktrees |

## Dependency Graph

One task, no dependencies.

## Interface Contracts

No interface changes. The assembled handoff prompt content changes (additional directive blocks).

## Traceability Matrix

| Task | Spec Requirement |
|------|-----------------|
| 1 | FR-1, FR-2, FR-3, NFR-1, NFR-2, AC-1, AC-2, AC-3, AC-4 |
