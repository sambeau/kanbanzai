# Dev-Plan: Error Classification for MCP Handlers

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | approved |
| Author | architect                      |

## Overview

This dev-plan implements the error classification spec:
`work/P50-retro-may-2026/P50-spec-error-classification.md`
(DOC-`FEAT-01KQTNYMZRT6V/spec-p50-spec-error-classification`).

Adds a thin classification layer to MCP tool handlers so every logged failure has a
diagnostic `error_type`. The `actionlog.Entry` struct and its five `ErrorType` constants
already exist — the gap is purely that handlers don't classify errors before returning them.

## Task Breakdown

### T1: Add error classification helper
- **Deliverable:** New function `classifyError(err error) string` in `internal/mcp/` that maps known error patterns to the five `ErrorType` constants (`validation_error`, `not_found`, `gate_failure`, `precondition_error`, `internal_error`)
- **Depends on:** nothing
- **Effort:** 1 (single function with pattern matching)
- **Parallelisable:** yes

### T2: Classify errors in high-volume tools (entity, finish, next, doc, decompose)
- **Deliverable:** Each of the five tool handlers updated to call `classifyError` and set `ErrorType` on the action log entry before returning errors. Success paths left unchanged.
- **Depends on:** T1
- **Effort:** 2 (five handlers, each with 2-5 error paths)
- **Parallelisable:** no

### T3: Classify errors in remaining MCP tools
- **Deliverable:** All remaining MCP tool handlers updated with error classification
- **Depends on:** T1
- **Effort:** 2 (remaining handlers)
- **Parallelisable:** yes (can run alongside T2 with T1 complete)

### T4: Add error classification tests
- **Deliverable:** Table-driven tests for each tool × error category combination, asserting correct `ErrorType`. Existing error-path tests verified to still pass byte-identical.
- **Depends on:** T2, T3
- **Effort:** 2 (test suite)
- **Parallelisable:** no

## Dependency Graph

```
T1 ──┬── T2 ──┬── T4
     │        │
     └── T3 ──┘
```

T2 and T3 can run in parallel after T1. T4 gates on both.

## Interface Contracts

- **classifyError** returns one of five string constants matching existing `actionlog.ErrorType` values
- No existing handler signatures change
- No new imports beyond what handlers already use

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 (validation_error) | T1, T2 |
| REQ-002 (not_found) | T1, T2 |
| REQ-003 (gate_failure) | T1, T2 |
| REQ-004 (precondition_error) | T1, T2 |
| REQ-005 (internal_error) | T1, T2 |
| REQ-006 (high-volume first) | T2 |
| REQ-007 (nil error → empty) | T2, T3 |
| REQ-NF-001 (no error string change) | T4 |
| REQ-NF-002 (no latency impact) | T4 |
