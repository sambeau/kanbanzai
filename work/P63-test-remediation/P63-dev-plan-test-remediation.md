# B68 Dev-Plan — Test Remediation Implementation

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-10                     |
| Status | Draft                          |

## Overview

This dev-plan covers the implementation of P63 test remediation across 4 features in batch B68. All 4 features are independent and can be implemented in parallel. The only internal dependency is that Phase 1's verify task depends on the other Phase 1 tasks completing first.

## Task Breakdown

### FEAT-01KR8YRQN1HPG — Phase 1: Fix All 111 Failing Tests

| Task | Slug | Summary |
|------|------|---------|
| TASK-01KR8YZDB9AM1 | fix-plan-type | Fix planEntityTypeFromID to use model.EntityKindPlan. ~95 tests. |
| TASK-01KR8Z05Y5FZB | nil-guard-bug-gate | Nil guard in checkBugWorktreeHasCommits. |
| TASK-01KR8Z05YNZAH | sync-embedded-seeds | Sync embedded seed files. |
| TASK-01KR8Z05YH85N | fix-mcp-regressions | Fix MCP regression contracts. |
| TASK-01KR8Z05YKQ03 | verify-tests-pass | Verify go test ./... exits 0 (depends on all above). |

### FEAT-01KR8YRQN0P0W — Phase 2: Test Infrastructure Hardening

| Task | Slug | Summary |
|------|------|---------|
| TASK-01KR8Z11BJ6YA | consolidate-test-helpers | Consolidate writeTestPlan helpers. |
| TASK-01KR8Z11BAXBZ | replace-fragile-checks | Replace id[0]=='B' checks. |
| TASK-01KR8Z11BA20C | identify-duplicate-tests | Identify duplicate coverage. |
| TASK-01KR8Z11B8M2Y | remove-obsolete-tests | Remove tests for removed features. |

### FEAT-01KR8YRQN0D6G — Phase 3: Enforcement Mechanisms

| Task | Slug | Summary |
|------|------|---------|
| TASK-01KR8Z1RFPS36 | precommit-hook | Implement pre-commit hook. |
| TASK-01KR8Z1RF645K | merge-gate-test | Integrate go test into merge gate. |
| TASK-01KR8Z1RFGD8S | kbz-doctor-tests | Add test check to kbz doctor. |
| TASK-01KR8Z1RFN7N8 | health-mcp-tests | Add test status to health MCP. |

### FEAT-01KR8YRQN1M56 — Phase 4: DoD and Policy Updates

| Task | Slug | Summary |
|------|------|---------|
| TASK-01KR8Z2FNJMXX | update-dod | Update Definition of Done. |
| TASK-01KR8Z2FNHB2W | update-agents-md | Update AGENTS.md. |
| TASK-01KR8Z2FN2C29 | status-dashboard-tests | Add test health to status dashboard. |
| TASK-01KR8Z2FN9F8X | test-removal-convention | Document test removal commit convention. |

## Dependency Graph

```
Phase 1: fix-plan-type, nil-guard-bug-gate, sync-embedded-seeds, fix-mcp-regressions → verify-tests-pass
Phase 2: No internal dependencies (all tasks independent)
Phase 3: No internal dependencies (all tasks independent)
Phase 4: No internal dependencies (all tasks independent)

Cross-feature: None. All 4 features are fully independent.
```

## Interface Contracts

No new interfaces. All changes are:
- Test-only fixes (Phase 1)
- Test infrastructure refactors (Phase 2)
- New enforcement code with existing interfaces (Phase 3)
- Documentation changes (Phase 4)

## Traceability Matrix

All tasks trace to P63 specification requirements. See `work/P63-test-remediation/P63-spec-test-remediation.md` for full traceability.
