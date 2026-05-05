# Specification: Error Classification for MCP Handlers

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | spec-author                    |

## Overview

This specification implements the error classification feature described in
`work/P50-retro-may-2026/P50-design-retro-may-2026.md` (DOC-`P50-retro-may-2026/design-p50-design-retro-may-2026`), Feature 1.

The `actionlog.Entry` struct already defines five `ErrorType` constants
(`gate_failure`, `validation_error`, `not_found`, `precondition_error`,
`internal_error`) and the MCP handlers already return errors. The gap is
purely that handlers do not classify errors before returning them, so every
logged failure has an empty `error_type`. This makes instrumentation
non-diagnostic — operators cannot distinguish a validation failure from an
internal server error without reading log messages.

## Scope

**In scope:**
- Classify errors in MCP tool handlers so every logged failure has a meaningful `error_type`
- Prioritise high-volume tools: `entity`, `finish`, `next`, `doc`, `decompose`
- Extend to remaining tools after the high-volume set is complete

**Out of scope:**
- Changing error messages or error behaviour — this is classification only
- Adding new `ErrorType` constants
- Instrumentation dashboards or alerting rules

## Functional Requirements

- **REQ-001:** When an MCP tool handler returns an error caused by invalid
  input (malformed entity ID, missing required field, out-of-range value),
  the logged `actionlog.Entry` must have `ErrorType` set to
  `validation_error`.
- **REQ-002:** When an MCP tool handler returns an error caused by a
  referenced entity or document not existing, the logged `actionlog.Entry`
  must have `ErrorType` set to `not_found`.
- **REQ-003:** When an MCP tool handler returns an error caused by a stage
  gate prerequisite not being met (unapproved document, non-terminal
  tasks), the logged `actionlog.Entry` must have `ErrorType` set to
  `gate_failure`.
- **REQ-004:** When an MCP tool handler returns an error caused by an
  entity being in the wrong lifecycle state for the requested operation,
  the logged `actionlog.Entry` must have `ErrorType` set to
  `precondition_error`.
- **REQ-005:** When an MCP tool handler returns an error not matching any
  of the specific categories above (nil pointer dereference, unexpected
  failure, system error), the logged `actionlog.Entry` must have
  `ErrorType` set to `internal_error`.
- **REQ-006:** Error classification must be applied to the high-volume
  tools (`entity`, `finish`, `next`, `doc`, `decompose`) first. Remaining
  MCP tools must be classified in a follow-up pass.
- **REQ-007:** When a handler returns a nil error (success), the logged
  `actionlog.Entry` must have an empty `ErrorType` — no classification
  should be applied to successful calls.

## Non-Functional Requirements

- **REQ-NF-001:** Error classification must not change the error string
  returned to the MCP client — only the `ErrorType` field in the action
  log entry may change.
- **REQ-NF-002:** Error classification must not introduce additional
  latency beyond a single string-match or type-assertion check per error
  path.

## Constraints (Scope Exclusions)

- The `actionlog.Entry` struct and its `ErrorType` constants must not
  change — classification must use the existing five categories.
- Existing MCP tool handler signatures must not change.
- Existing tests for MCP tool error paths must continue to pass.
- This specification does NOT cover adding error classification to
  non-MCP codepaths (CLI handlers, internal service methods).

## Acceptance Criteria

- **AC-001 (REQ-001):** Given an `entity` tool call with a malformed
  entity ID (e.g. empty string), when the handler returns an error, then
  the action log entry has `ErrorType` = `validation_error`.
- **AC-002 (REQ-002):** Given a `doc(action: "get")` call with a
  non-existent document ID, when the handler returns an error, then the
  action log entry has `ErrorType` = `not_found`.
- **AC-003 (REQ-003):** Given an `entity(action: "transition")` call
  where the target stage's document prerequisites are not met, when the
  handler returns an error, then the action log entry has `ErrorType` =
  `gate_failure`.
- **AC-004 (REQ-004):** Given a `finish` call on a task that is not in
  `active` status, when the handler returns an error, then the action log
  entry has `ErrorType` = `precondition_error`.
- **AC-005 (REQ-005):** Given an MCP handler that encounters an
  unexpected nil pointer dereference, when the handler returns an error,
  then the action log entry has `ErrorType` = `internal_error`.
- **AC-006 (REQ-006):** Given the five high-volume tools (`entity`,
  `finish`, `next`, `doc`, `decompose`), when any error path is exercised,
  then the corresponding action log entry has a non-empty `ErrorType`
  matching the error's category.
- **AC-007 (REQ-007):** Given any MCP tool call that succeeds, when the
  handler returns nil, then the action log entry has an empty `ErrorType`.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: call `entity` with malformed ID, assert `ErrorType` is `validation_error` |
| AC-002 | Test | Unit test: call `doc(get)` with unknown ID, assert `ErrorType` is `not_found` |
| AC-003 | Test | Unit test: call `entity(transition)` with missing doc prereq, assert `ErrorType` is `gate_failure` |
| AC-004 | Test | Unit test: call `finish` on a `ready` task, assert `ErrorType` is `precondition_error` |
| AC-005 | Test | Unit test: induce nil dereference in handler, assert `ErrorType` is `internal_error` |
| AC-006 | Test | Table-driven test: for each of the five tools, exercise one error path per category and assert correct `ErrorType` |
| AC-007 | Test | Existing success-path tests must continue to pass with empty `ErrorType` |
| REQ-NF-001 | Test | Run existing error-path tests after classification: assert error strings returned to client are byte-identical to pre-classification output |
| REQ-NF-002 | Test | Benchmark error-path handlers before and after classification, assert no statistically significant latency increase |
