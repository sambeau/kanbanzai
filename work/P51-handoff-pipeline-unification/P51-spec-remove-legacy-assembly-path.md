| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T16:19:45Z           |
| Status | Draft                          |
| Author | spec-author                     |

# Specification: Remove Legacy Assembly Path

**Feature:** FEAT-01KQYZZFGJA93 (Remove Legacy Assembly Path)
**Parent Batch:** B1-p51-exec
**Design:** `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md`

## Overview

This specification implements the legacy assembly path removal described in `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md` (design document for P51, §1–§3). It removes the dual-path architecture from `handoff` by deleting the legacy 2.0 fallback code and making the 3.0 pipeline the single, unconditional code path with hard errors for misconfiguration.

## Scope

**In scope:**
- Remove `tryPipeline`, `buildLegacyResponse`, `renderHandoffPrompt` functions
- Remove `assembleContext` and all its helper functions and types
- Simplify `HandoffTools` signature (remove 7 parameters)
- Update handoff tests to verify hard errors instead of fallback behavior

**Out of scope:**
- Removing the `next` tool's use of `assembleContext` (deferred to follow-up)
- Changing the pipeline itself
- Building `dispatch_task` (P44)

## Related Work

Concepts searched: `assembleContext`, `renderHandoffPrompt`, `buildLegacyResponse`, `asmInput`, `assembledContext`, `tryPipeline`, `legacy assembly fallback`, `handoffTool`.

Entity IDs searched: P50, P51, FEAT-01KQYZZFGJA93.

Prior specifications searched: none found.

**Attestation:** No directly related prior work was found in the corpus. The design document (P51-design-handoff-pipeline-unification, §1–§3) exhaustively enumerates the legacy functions, types, and constants to remove and the callers to update.

## Problem Statement

The `handoff` MCP tool currently has a dual-path architecture: a 3.0 pipeline (roles, skills, vocabulary, anti-patterns, knowledge, code graph) and a legacy 2.0 fallback (`assembleContext` + `renderHandoffPrompt` → `buildLegacyResponse`). The fallback activates silently when a stage binding is missing, the pipeline is unconfigured, or the task has no parent feature. All three conditions indicate misconfiguration, not a valid reason to silently degrade context quality.

The legacy path produces prompts without role-attuned vocabulary, without skill-specific anti-patterns, and without code-graph context — exactly the deficiencies that the pipeline was built to address. After the P50 incident demonstrated that the pipeline is the active path for all real-world calls, keeping the legacy code creates dead code risk, makes the `handoffTool` signature unnecessarily wide (10+ dependencies), and complicates P44's planned `dispatch_task` internalization.

## Functional Requirements

- **FR-001:** The `handoffTool` function MUST call `pipeline.Run(input)` directly without checking for nil pipeline, empty parent feature, or missing stage binding. The pipeline's own validation steps (`stepValidateLifecycle`, `stepLookupBinding`) handle these errors.
- **FR-002:** When the pipeline is nil (stage-bindings.yaml not loaded), `handoff` MUST return a hard error with the message containing "stage-bindings.yaml".
- **FR-003:** When the task has no parent feature, `handoff` MUST return a hard error from the pipeline's lifecycle validation step (existing behavior in the pipeline path, now the only path).
- **FR-004:** The `tryPipeline`, `buildLegacyResponse`, and `renderHandoffPrompt` functions MUST be removed from `internal/mcp/handoff_tool.go`.
- **FR-005:** The `assembleContext` function and the `asmInput` / `assembledContext` types MUST be removed from `internal/mcp/assembly.go`. All `asm*` helper functions and types that are only used by `assembleContext` MUST be removed.
- **FR-006:** The `HandoffTools` factory function MUST accept only `entitySvc` and `pipeline` parameters. The following parameters MUST be removed from the signature: `profileStore`, `knowledgeSvc`, `intelligenceSvc`, `docRecordSvc`, `mergedToolHints`, `roleStore`, `worktreeStore`.
- **FR-007:** Callers of `HandoffTools` (in `internal/mcp/server.go`) MUST be updated to pass only `entitySvc` and `pipeline`.
- **FR-008:** Functions and types still used by the `next` tool in `assembly.go` MUST be preserved. If `assembleContext` shares types with `next`'s assembly path, those shared types MUST be extracted to a separate file or left in place with only the handoff-specific code removed.
- **FR-009:** All tests that verify fallback behavior (legacy assembly path activation) MUST be updated to verify hard errors instead.

## Non-Functional Requirements

- **FR-NF-001:** Removing the legacy path MUST NOT change the pipeline-based handoff output. For every valid input, the handoff response MUST be identical before and after the removal.
- **FR-NF-002:** The `go build ./...` command MUST succeed with no errors after the removal.
- **FR-NF-003:** All existing tests in `internal/mcp/` that do not test legacy fallback behavior MUST continue to pass without modification.

## Constraints

- The `next` tool's use of `assembleContext` MUST continue to work — do not remove types or functions that `next_tool.go` depends on.
- The stage-aware guidance text constants (`asmReviewRubricText`, `asmTestExpectText`, etc.) must be preserved if `next` uses them.
- The `handoff` tool's MCP tool name MUST NOT change — only the implementation is changing.

## Acceptance Criteria

- **AC-001 (FR-001):** Given a valid pipeline, task, and feature, when `handoff(task_id: "TASK-xxx")` is called, then the response is produced by `pipeline.Run` and has `assembly_path: "pipeline-3.0"` in metadata.
- **AC-002 (FR-002):** Given a nil pipeline (no stage-bindings.yaml), when `handoff` is called, then the error message contains "stage-bindings.yaml".
- **AC-003 (FR-003):** Given a task with no parent feature, when `handoff` is called, then the error is returned from the pipeline (not from a separate nil check in handoff).
- **AC-004 (FR-004):** When `grep` for `tryPipeline`, `buildLegacyResponse`, `renderHandoffPrompt` in `internal/mcp/handoff_tool.go`, then zero matches are found.
- **AC-005 (FR-005):** When `grep` for `assembleContext`, `asmInput`, `assembledContext` in `internal/mcp/assembly.go`, then zero matches are found — except any still referenced by `next_tool.go`.
- **AC-006 (FR-006):** When inspecting the `HandoffTools` function signature, then it accepts `entitySvc` (type `EntityService`) and `pipeline` (type `*Pipeline`) only.
- **AC-007 (FR-007):** When building the project with `go build ./...`, then compilation succeeds with no errors.
- **AC-008 (FR-008):** When running `go test ./internal/mcp/...`, then all non-legacy tests pass.
- **AC-009 (FR-009):** When `grep` for `TestHandoff.*[Ff]allback` or `TestHandoff.*[Ll]egacy` in `internal/mcp/handoff_tool_test.go`, then any matching tests verify hard errors, not fallback content.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Integration test: call handoff with valid inputs, assert pipeline metadata |
| AC-002 | Test | Unit test: nil pipeline → assert error contains "stage-bindings.yaml" |
| AC-003 | Test | Unit test: task with no parent feature → assert pipeline error propagated |
| AC-004 | Inspection | grep for removed function names in handoff_tool.go |
| AC-005 | Inspection | grep for removed types in assembly.go, cross-reference with next_tool.go imports |
| AC-006 | Inspection | Read HandoffTools signature, verify parameter count |
| AC-007 | Test | `go build ./...` succeeds |
| AC-008 | Test | `go test ./internal/mcp/...` passes |
| AC-009 | Inspection | grep handoff test file for fallback/legacy tests → verify hard error assertions |
