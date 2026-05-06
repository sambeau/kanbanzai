# Dev-Plan: Fix 44 Failing Tests from Uncommitted Refactors

**Parent bug:** BUG-01KQZS69DAJFG
**Status:** draft

## Overview

Eight sequential fix steps, ordered by dependency. Production code fixes first (validate,
service layer), then test updates. Each step targets a specific root cause group.

## Task Breakdown

### Phase 1: Production Code Fixes

**T1: Restore merging/verifying states to validate/lifecycle.go**
- Add `"merging"` and `"verifying"` entries to `allowedTransitions[EntityFeature]`
- Transitions: `reviewing → merging`, `merging → verifying`, `verifying → done`, `verifying → needs-rework`
- Unblocks: 7 merge tests + 1 validate test

**T2: Fix isStrategicPlanFields classification in plans.go**
- Add `type` field to `planRecordFile` struct
- Read `Type` from `storage.EntityRecord` during file listing
- Replace status heuristics with explicit type check
- Unblocks: 5 rollup tests + 2 grandparent gate tests + 1 health check test

**T3: Fix intelligence access counters**
- Investigate why `AccessCount`, `LastAccessedAt`, `SectionAccess` aren't incrementing
- Fix the counter-increment code path
- Unblocks: 9 intelligence tests

### Phase 2: Test Updates

**T4: Update storage tests for batches/ path**
- `TestEntityStore_WriteAndLoad_Plan`: expect `batches/` not `plans/`
- `TestEntityStore_Load_FixtureFiles/plan`: fix fixture path
- `TestEntityStore_Load_EmptyFile`: fix expected path
- `TestEntityStore_Load_CorruptYAML`: fix expected path

**T5: Update handoff tests for pipeline v3.0**
- Change `"### Conventions"` prefix → `"## Task:"` in 7 tests
- Update `context_metadata` assertions to new structure
- Fix nil panic in `TestHandoff_ContextMetadataFields`

**T6: Update error message assertions**
- `TestGetPlan_InvalidIDFormat`: match "invalid Plan ID format" (or fix code to say Batch)
- `TestDisplayID_AC005_CreateFeatureRequiresParent`: match "parent plan or batch is required"

**T7: Update decompose test expectations**
- `TestRefuseToPropose_DiagnosticContent`: update expected diagnostic format
- `TestRefuseToPropose_DiagnosticWithBoldOutsideAC`: update expected diagnostic format
- `TestPairedTestTasks_DependsOnCorrectness`: fix DependsOn expectation
- `TestPairedTestTasks_TestOnlyAC_SingleTask`: fix slug suffix expectation

**T8: Fix TestNextTrimContext_T3BeforeT2**
- Update assertion to match current trimming order

## Dependency Graph

```
T1 ──┐
T2 ──┤
T3 ──┼── T4 ── T5 ── T6 ── T7 ── T8
     │
     └── (parallel — no shared files)
```

T1–T3 are independent production code fixes (disjoint file sets: `validate/lifecycle.go`,
`service/plans.go`, intelligence service). T4–T8 are test-only updates that can proceed
sequentially after production fixes.

## Interface Contracts

- `planRecordFile` struct gains `entityType string` field — callers in `listPlanRecordFiles` must populate it
- `isStrategicPlanFields()` signature unchanged but behavior switches from heuristic to explicit
- `allowedTransitions[EntityFeature]` gains two new state keys — no signature changes

## Traceability Matrix

| Task | FR | AC |
|------|-----|-----|
| T1 | FR-01 | AC-01, AC-02 |
| T2 | FR-02 | AC-06, AC-08, AC-10 |
| T3 | FR-03 | AC-07 |
| T4 | FR-04 | AC-05 |
| T5 | FR-05 | AC-03 |
| T6 | FR-06 | AC-06 |
| T7 | FR-07 | AC-09 |
| T8 | FR-08 | AC-04 |
