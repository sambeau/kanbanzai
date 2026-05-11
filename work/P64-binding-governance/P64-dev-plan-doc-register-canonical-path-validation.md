# Dev-Plan: doc register rejects non-canonical paths

## Overview

Single-task implementation: add path validation to `doc(action: register)` in `internal/mcp/doc_tool.go`.

## Task Breakdown

| # | Task | File | Description |
|---|------|------|-------------|
| 1 | Add canonical path validation | `internal/mcp/doc_tool.go` | Compare supplied path against `doc(action: path)` output; reject mismatches with error including canonical path |

## Dependency Graph

One task, no dependencies.

## Interface Contracts

No interface changes. Addition of a validation check within the existing register handler.

## Traceability Matrix

| Task | Spec Requirement |
|------|-----------------|
| 1 | FR-1, FR-2, NFR-1, AC-1, AC-2, AC-3 |
