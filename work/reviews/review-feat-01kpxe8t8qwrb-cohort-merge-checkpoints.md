# Review: Cohort-Based Merge Checkpoints (FEAT-01KPXE8T8QWRB)

**Reviewer:** orchestrator (automated)
**Date:** 2026-04-24
**Verdict:** Pass

---

## Summary

All four tasks of FEAT-01KPXE8T8QWRB were implemented and verified. The feature delivers:
1. `GetBranchCreatedAt` extension to `BranchLookup` (T1)
2. Feature-level conflict analysis with `drift_days` via `feature_ids` mode in the `conflict` tool (T2)
3. `## Merge Schedule` section in dev-plan template + cohort checklist in `write-dev-plan` skill (T3)
4. Non-blocking decompose warning for missing merge schedule + Phase 0 / step 7 in `orchestrate-development` skill (T4)

---

## Findings

### Conformance

| AC | Status | Notes |
|----|--------|-------|
| AC-001 | ✅ Pass | Mutual exclusivity error returned when both `task_ids` and `feature_ids` supplied |
| AC-002 | ✅ Pass | `feature_ids` mode returns `FeatureConflictResult`, not task-level result |
| AC-003 | ✅ Pass | Overlapping `files_planned` → non-`safe_to_parallelise` risk |
| AC-004 | ✅ Pass | No overlap → `safe_to_parallelise` |
| AC-005 | ✅ Pass | Empty `files_planned` → `NoFileData: true`, no error |
| AC-006 | ✅ Pass | `DriftDays` populated from `GetBranchCreatedAt` |
| AC-007 | ✅ Pass | `DriftDays` nil when no worktree record |
| AC-008 | ✅ Pass | `FeatureConflictResult` structure in response |
| AC-009 | ✅ Pass | `## Merge Schedule` section present in dev-plan template with required table and prose |
| AC-010 | ✅ Pass | Cohort checklist item present in `write-dev-plan` skill quality-checks section |
| AC-011 | ✅ Pass | Warning emitted when plan has >3 features and no `## Merge Schedule` heading |
| AC-012 | ✅ Pass | No warning when plan has ≤3 features |
| AC-013 | ✅ Pass | No warning when `## Merge Schedule` heading present |
| AC-014 | ✅ Pass | Phase 0 with all 5 steps present before Phase 1 in orchestrate-development skill |
| AC-015 | ✅ Pass | Step 7 cohort checkpoint present in Phase 6 Close-Out |
| AC-016 | ✅ Pass | Pre-existing task-level conflict tests pass unmodified |
| AC-017 | ✅ Pass | Unit tests cover aggregation, `no_file_data`, `drift_days` presence/absence, mutual exclusivity |

### Test Results

- `go test ./internal/service/...` — **PASS** (all new and existing tests)
- `go test ./internal/mcp/...` — Known pre-existing SQLite contention flakiness under parallel load (BUG-01KN87EEF2G49); all tests pass in isolation. No new failures introduced.

### Non-Functional

- **NFR-001** (no regression on `task_ids` behaviour): ✅ confirmed by AC-016
- **NFR-002** (`BranchLookup` interface only for git): ✅ `GetBranchCreatedAt` reads worktree record via store, no shell calls
- **NFR-003** (reuse `analyzePair`): ✅ `checkFeatures` builds synthetic `taskConflictInfo` and delegates to existing `analyzePair`
- **NFR-004** (decompose warning non-blocking): ✅ warning appended to `Warnings` slice, no error returned
- **NFR-005** (skill files only documentation target): ✅ only `write-dev-plan/SKILL.md` and `orchestrate-development/SKILL.md` modified

---

## Verdict

**Pass** — all 17 acceptance criteria met, no regressions introduced, non-functional requirements satisfied.
