# Spec: Fix 44 Failing Tests from Uncommitted Refactors

**Parent bug:** BUG-01KQZS69DAJFG
**Status:** draft

## Overview

Uncommitted production code refactors on main caused 44 tests to fail across 4 packages.
This spec defines the remediation: fix production code where the refactors are incomplete,
and update tests where the refactors are intentional.

## Scope

| Package | Failing Tests | Root Cause Group |
|---------|--------------|------------------|
| `internal/validate` | 1 | Merging/verifying states missing from validator |
| `internal/mcp` | 16 | Merging/verifying states (7), handoff pipeline v3.0 (8), trim context order (1) |
| `internal/storage` | 4 | Plan→batch rename: storage paths |
| `internal/service` | 23 | Plan→batch rename (3), intelligence counters (9), plan rollup (5), decompose diagnostics (4), grandparent gates (2) |

## Functional Requirements

### FR-01: Restore merging/verifying feature lifecycle states
The `validate/lifecycle.go` `allowedTransitions[EntityFeature]` map must include
`merging` and `verifying` states with their valid transitions:
- `reviewing → merging`
- `merging → verifying`
- `verifying → done`, `verifying → needs-rework`

### FR-02: Fix entity classification heuristics
`isStrategicPlanFields()` in `internal/service/plans.go` must classify entities
by their explicit `type` field rather than status-based heuristics.
The `planRecordFile` struct must carry the `type` field from the YAML.

### FR-03: Fix intelligence access counters
All `doc_intel` access counter tests must pass. The in-memory store's access
tracking must function correctly for all operations.

### FR-04: Update storage tests for batches/ path
Storage tests must expect `batches/` directory paths instead of `plans/`.

### FR-05: Update handoff tests for pipeline v3.0
Handoff tests must assert the new prompt format starting with `## Task:` and
the new `context_metadata` structure.

### FR-06: Update error message assertions
Tests asserting specific error message text must match the current code output.

### FR-07: Update decompose test expectations
Decompose diagnostic format and task naming conventions must match current
code behavior.

### FR-08: Fix context trimming order test
`TestNextTrimContext_T3BeforeT2` assertion must match current trimming order.

## Acceptance Criteria

- [ ] **AC-01:** `TestValidNextStates` passes with correct developing state transitions
- [ ] **AC-02:** All 7 merge-related tests in `internal/mcp` pass (`TestMergeVerifyDone_*`, `TestExecuteMerge_*`)
- [ ] **AC-03:** All 8 handoff tests in `internal/mcp` pass (`TestHandoff_*`, `TestIntegration_NextHandoffFinish`)
- [ ] **AC-04:** `TestNextTrimContext_T3BeforeT2` passes
- [ ] **AC-05:** All 4 storage tests in `internal/storage` pass
- [ ] **AC-06:** All 3 plan→batch rename service tests pass (`TestGetPlan_InvalidIDFormat`, `TestDisplayID_AC005_CreateFeatureRequiresParent`, `TestEntityService_HealthCheck_CleanProject`)
- [ ] **AC-07:** All 9 intelligence access counter tests in `internal/service` pass
- [ ] **AC-08:** All 5 plan rollup tests in `internal/service` pass
- [ ] **AC-09:** All 4 decompose tests in `internal/service` pass
- [ ] **AC-10:** Both grandparent gate tests in `internal/service` pass
- [ ] **AC-11:** `go test ./...` passes with zero failures
