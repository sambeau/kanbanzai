# Dev Plan: Remove Legacy Assembly Path

**Feature:** FEAT-01KQYZZFGJA93
**Specification:** FEAT-01KQYZZFGJA93/spec-p51-spec-remove-legacy-assembly-path (approved)
**Date:** 2026-05-06

## Overview

Removes the legacy 2.0 context assembly fallback from handoff, making the 3.0 pipeline the single unconditional code path. Deletes tryPipeline, assembleContext, buildLegacyResponse, renderHandoffPrompt and related types. Simplifies HandoffTools from 9 to 2 parameters. Five tasks.

## Scope

Implements FR-001 through FR-009 and FR-NF-001 through FR-NF-003. Five tasks: remove legacy code from handoff_tool.go and assembly.go, simplify HandoffTools signature, make pipeline the only path, and update tests.

## Task Breakdown

### Task 1: Remove legacy functions from handoff_tool.go (TASK-01KQZ302QWNCW)
- Remove `tryPipeline`, `buildLegacyResponse`, `renderHandoffPrompt` functions
- Simplify `HandoffTools` signature to `entitySvc` + `pipeline` only
- ACs: AC-004, AC-006

### Task 2: Remove assembleContext and legacy types from assembly.go (TASK-01KQZ302R0ADQ)
- Remove `assembleContext`, `asmInput`, `assembledContext` and asm* helper functions/types
- Preserve anything still referenced by `next_tool.go`
- ACs: AC-005, AC-008

### Task 3: Update server.go HandoffTools wiring (TASK-01KQZ302QX3A0)
- Update `HandoffTools(...)` call to pass only `entitySvc` and `pipeline`
- Remove 7 dropped parameters from handoff wiring
- AC: AC-007

### Task 4: Make handoff pipeline-only with hard errors (TASK-01KQZ302QZYFZ)
- Replace `tryPipeline` with direct `pipeline.Run` call
- Hard error: nil pipeline → "stage-bindings.yaml"
- Hard error: missing parent feature → pipeline lifecycle validation
- Hard error: missing stage binding → pipeline binding lookup
- ACs: AC-001, AC-002, AC-003

### Task 5: Update tests and verify build for legacy removal (TASK-01KQZ302QYMHP)
- Update fallback/legacy tests to verify hard errors
- Verify `go build ./...` succeeds
- Verify `go test ./internal/mcp/...` passes
- grep for removed function names → zero matches
- Dependencies: Tasks 1-4
- ACs: AC-007, AC-008, AC-009

## Dependency Graph

```
T1 (remove handoff_tool.go legacy)
  ├── T3 (update server.go wiring) ── depends on T1
  └── T4 (pipeline-only + hard errors) ── depends on T1

T2 (remove assembly.go legacy) ── independent of T1/T3/T4

T5 (tests + build) ── depends on all T1-T4
```

T1, T2 can run in parallel. T3 and T4 depend on T1. T5 is the integration gate.

## Risk Assessment

- **Risk:** Removing types from assembly.go breaks `next_tool.go` → **Medium**. Mitigation: grep for all removed symbols in `next_tool.go` before deletion; preserve any shared types.
- **Risk:** HandoffTools signature change breaks callers beyond server.go → **Low**. `grep` for `HandoffTools(` shows only server.go as caller.
- **Risk:** Fallback tests converted to hard-error tests miss edge cases → **Low**. The hard-error conditions are simple: nil pipeline, missing feature, missing binding.

## Verification Approach

| AC | Method | Task |
|----|--------|------|
| AC-001 | Integration test: pipeline metadata in response | T4, T5 |
| AC-002 | Unit test: nil pipeline → "stage-bindings.yaml" | T4, T5 |
| AC-003 | Unit test: no parent feature → pipeline error | T4, T5 |
| AC-004 | grep: zero matches for removed functions in handoff_tool.go | T1, T5 |
| AC-005 | grep: zero matches for removed types in assembly.go | T2, T5 |
| AC-006 | Inspection: HandoffTools signature has 2 params | T1, T5 |
| AC-007 | `go build ./...` succeeds | T3, T5 |
| AC-008 | `go test ./internal/mcp/...` passes | T2, T5 |
| AC-009 | grep: fallback tests → hard error assertions | T5 |

## Interface Contracts

- **HandoffTools** — Signature: `func HandoffTools(entitySvc EntityService, pipeline *Pipeline) []Tool`. Removed 7 parameters: profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, mergedToolHints, roleStore, worktreeStore.
- **handoff tool** — Calls `pipeline.Run` directly; no nil check, no feature check, no binding check outside the pipeline.
- **assembly.go** — Only symbols referenced by `next_tool.go` preserved. All handoff-only symbols deleted.

## Traceability Matrix

| FR | Task |
|----|------|
| FR-001 | T4 |
| FR-002 | T4 |
| FR-003 | T4 |
| FR-004 | T1 |
| FR-005 | T2 |
| FR-006 | T1 |
| FR-007 | T3 |
| FR-008 | T2 |
| FR-009 | T5 |
| FR-NF-001 | T4, T5 |
| FR-NF-002 | T5 |
| FR-NF-003 | T5 |
