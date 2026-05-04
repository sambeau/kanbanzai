# Review: FEAT-01KQSP41PE6JP — Fast-Track Architecture Core Implementation (Cycle 3)

| Field        | Value                                          |
|--------------|------------------------------------------------|
| Date         | 2026-05-04                                     |
| Feature      | FEAT-01KQSP41PE6JP (B48-F1)                    |
| Plan         | P43-fast-track-architecture                    |
| Batch        | B48-fast-track-impl                            |
| Review Cycle | 3 ⚠️ (at cap — max_review_cycles=3)            |
| Reviewers    | reviewer-conformance, reviewer-quality, reviewer-testing, reviewer-security |
| Spec         | FEAT-01KQSP41PE6JP/spec-p43-spec-fast-track-architecture |

---

## Aggregate Verdict: **rejected** (needs_remediation)
**Escalation: Review cycle cap (3/3) reached — human checkpoint required.**

Six blocking findings remain. The feature branch is **76 commits behind main** and has **merge conflicts**. Fixes for all 6 cycle-2 blocking findings exist on `main` but have not been applied to the worktree branch.

---

## Structural Issue: Fixes on Main, Not on Feature Branch

Every blocking finding from cycle 2 has a fix present on `main` but **absent from the
feature worktree** (`feature/FEAT-01KQSP41PE6JP-fast-track-core`). The worktree branch
diverged from main and the cycle-2 remediation work was committed directly to `main` rather
than to the feature branch.

This means the feature branch **cannot be reviewed as a self-contained unit** — a reviewer
examining only the worktree would see all cycle-2 blocking findings as unresolved. A merge
from this branch would lose 76 commits of unrelated work on main, or produce merge conflicts
(as currently detected).

---

## Cycle-2 Finding Closure

| Cycle-2 Finding | Status | Evidence |
|-----------------|--------|----------|
| BLOCK-1: dispatchSvc not wired | ❌ **Still present on worktree** | `checkTransitionValidator` missing `docSvc`/`entitySvc` params and `.WithDispatch()` call. Fix exists on `main` at `entity_tool.go#L914-956`. |
| BLOCK-2: Auto-approval not implemented | ❌ **Still present on worktree** | `tryAutoValidate` returns `{status:"dispatched"}` and exits. Fix exists on `main`. |
| BLOCK-3: CreateBug test regression | ❌ **Still failing on worktree** | `wantState` missing `"tier": "bug_fix"`. Test confirmed failing: `go test -run TestEntityService_CreateBug_AppliesDefaults`. Fix exists on `main`. |
| BLOCK-4: Retro-tag inference test missing | ❌ **Still absent from worktree** | `TestFastTrack_TierInference_RetroTagInfersRetroFix` not in worktree integration tests. Exists on `main` at L517. |
| BLOCK-5: Key-name mismatch | ❌ **Still present on worktree** | Tests read `"blocking_fail"`; entity tool writes `"blocking"`. Fix exists on `main`. |
| BLOCK-6: PersistFeatureOverrides errors discarded | ❌ **Still present on worktree** | Errors at `entity_tool.go#L689, L696` not checked. Fix (log.Printf wrapping) exists on `main`. |

**Additionally, 7 deliverable files from the original task list were committed to `main`
directly and never appear in the worktree:**
- `.kbz/roles/spec-validator.yaml`
- `.kbz/roles/plan-validator.yaml`
- `.kbz/skills/validate-spec/SKILL.md`
- `.kbz/skills/validate-review/SKILL.md`
- `work/P43-fast-track-architecture/validator-rubrics/spec-validator-rubrics.md`
- `work/P43-fast-track-architecture/validator-rubrics/plan-validator-rubrics.md`
- `work/P43-fast-track-architecture/validator-rubrics/review-gate-validator-rubrics.md`

These files exist on `main` but cannot be reviewed in the context of the feature branch.

---

## Per-Dimension Verdicts

| Dimension | Verdict | Reviewer |
|-----------|---------|----------|
| spec_conformance | fail | reviewer-conformance (8 blocking findings) |
| implementation_quality | fail | reviewer-quality (1 blocking, 7 non-blocking) |
| test_adequacy | fail | reviewer-testing (2 blocking, 3 high, 3 medium) |
| error_handling | pass | reviewer-quality |
| naming_consistency | pass | reviewer-quality |
| go_idioms | pass_with_notes | reviewer-quality |
| package_cohesion | pass | reviewer-quality |
| dead_code | concern | reviewer-quality |
| input_validation | fail | reviewer-security |
| trust_boundary | fail | reviewer-security |
| integrity | fail | reviewer-security |
| audit_trail | concern | reviewer-security |
| privilege_escalation | fail | reviewer-security |

---

## Review Unit Dispatch

| Unit | Files | Reviewer |
|------|-------|----------|
| Spec conformance (all deliverables + bindings) | 15 files | reviewer-conformance |
| Core Go implementation | 9 production files | reviewer-quality |
| Test suite | 6 test files | reviewer-testing |
| Security surfaces | 7 files | reviewer-security |

---

## Blocking Findings

### B1: dispatchSvc never wired — auto-validation pipeline is a no-op
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-003, AC-PIPE-002, REQ-TRANS-001 through REQ-TRANS-003
- **Reported by:** reviewer-conformance, reviewer-quality, reviewer-security
- **Locations:**
  - `entity_tool.go#L908-951` — `checkTransitionValidator` never calls `.WithDispatch()`
  - `transition_validator.go#L131-142` — `AUTO_PLACEHOLDER` always taken (`dispatchSvc == nil`)
  - `validator_dispatch.go#L130-140` — `Dispatch()` returns unconditional `VerdictPass`
- **Description:** Every auto-mode gate transition returns a placeholder pass. No real validator is ever dispatched from the transition path. The `SpawnAgentDispatcher.Dispatch` method generates a prompt but returns `VerdictPass` unconditionally — the sub-agent override path is not implemented.
- **Remediation:** The fix exists on `main` (`entity_tool.go#L914-956`): add `docSvc`/`entitySvc` parameters to `checkTransitionValidator`, create a `SpawnAgentDispatcher`, call `.WithDispatch(dispatchSvc)`. Port this fix to the worktree.

### B2: Test regression — `TestEntityService_CreateBug_AppliesDefaults` fails
- **Severity:** blocking
- **Spec anchor:** REQ-INFER-002(b)
- **Location:** `entities_test.go#L168-182`
- **Description:** The `wantState` map does not include `"tier": "bug_fix"`, but `CreateBug` now calls `inferTier` which returns `"bug_fix"`. Confirmed failing at runtime.
- **Remediation:** Add `"tier": "bug_fix"` to `wantState`. Fix exists on `main`.

### B3: Key-name mismatch — `tv["blocking_fail"]` vs `tv["blocking"]`
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-003, AC-RVW-002
- **Locations:**
  - `entity_tool.go#L630` — response uses key `"blocking"`
  - `fast_track_integration_test.go#L213, L262, L304` — tests read `tv["blocking_fail"]`
- **Description:** Tests read a key that doesn't exist, making assertions silent no-ops.
- **Remediation:** Change `tv["blocking_fail"]` → `tv["blocking"]` in three test assertions. Fix exists on `main`.

### B4: PersistFeatureOverrides errors silently discarded
- **Severity:** blocking
- **Spec anchor:** REQ-TRANS-007 ("override usage tracked as metric")
- **Location:** `entity_tool.go#L687, L695`
- **Description:** Error return of `entitySvc.PersistFeatureOverrides` is not checked. Override metric silently lost.
- **Remediation:** Add error logging. Fix exists on `main`.

### B5: Deliverable files missing from worktree
- **Severity:** blocking
- **Spec anchors:** AC-SPEC-001, AC-PLAN-001, AC-SPEC-002, AC-RVW-002, AC-RUB-001
- **Description:** Seven deliverable files (2 roles, 2 skills, 3 rubrics) exist on `main` but are absent from the feature worktree. The feature branch cannot be reviewed as a self-contained unit.
- **Remediation:** Cherry-pick or rebase the missing files into the feature branch.

### B6: Tier downgrade via UpdateEntity — privilege escalation
- **Severity:** blocking (security)
- **Spec anchor:** REQ-TIER-004
- **CVSS:** 7.6 (AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:H)
- **Location:** `entities.go#L864-906` (UpdateEntity)
- **Description:** `UpdateEntity` accepts arbitrary `tier` with no validation. An attacker can downgrade a `critical` feature to bypass human gates.
- **Remediation:** Add tier immutability enforcement in `UpdateEntity`, or add tier validation in `ValidateRecord`.

---

## Non-Blocking Findings

| ID | Description | Location | Reporter |
|----|-------------|----------|----------|
| NB-1 | Auto-approval loop not closed: `tryAutoValidate` dispatches but never calls `doc(approve)` | `doc_tool.go#L445-460` | quality, conformance |
| NB-2 | Dead tier guard `"fast-track"` in `transition_validator.go` | `transition_validator.go#L133` | quality |
| NB-3 | `isDocOnlyChange` prefix `work/` matches `worktrees/` files | `transition_validator.go#L234` | quality, security |
| NB-4 | `ValidatorContext` fields unpopulated in `runAutoValidation` | `transition_validator.go#L251` | quality |
| NB-5 | `ValidatorSummary.Prompt` generated but never consumed | `validator_dispatch.go#L155` | quality |
| NB-6 | 3 integration tests use `t.Skip` — AC-TRANS-001/003, AC-RVW-002 untested | `fast_track_integration_test.go` | testing |
| NB-7 | `fail → not-approved` pipeline scenario has no test | `fast_track_integration_test.go` | testing |
| NB-8 | `Pipeline_AutoApprovesOnPass` uses `t.Log` not `t.Error` when key absent | `fast_track_integration_test.go#L613` | testing |
| NB-9 | Wall-clock timing tests inherently flaky | `fast_track_integration_test.go#L701-762` | testing |
| NB-10 | Cycle cap bypassable via `IncrementFeatureReviewCycle` error | `doc_tool.go#L429-433` | security |
| NB-11 | `blocked_reason` clearable without human mediation | `entities.go` | security |
| NB-12 | `log.Printf` for persist errors — callers can't observe failures | `doc_tool.go#L420, L429` | quality |
| NB-13 | Redundant `"specification"` check in `tryAutoValidate` | `doc_tool.go#L333` | quality |
| NB-14 | Tier matrix tests in MCP file duplicate config tests | `fast_track_integration_test.go#L409-480` | testing |

---

## Finding Summary

| Classification | Count |
|----------------|-------|
| Blocking | 6 |
| Non-blocking | 14 |
| **Total** | **20** |

---

## What Went Well

- Tier inference logic is correct and well-covered at the service layer (11 unit tests)
- `ValidatorDispatcher` interface design is clean — P44 model-routing swap remains feasible
- `FastTrackConfig` schema is comprehensive with thorough validation
- `evaluateConditional` for retro_fix tier is architecturally sound
- `tryAutoValidate` cycle tracking and escalation logic works correctly
- `TransitionValidatorDispatcher` gate mode evaluation is correct for all paths
- `BuildTransitionValidatorError` produces well-formatted error messages
- Override mechanism and human escape-hatch are implemented
- Doc tool auto-validation tests (8 tests) are comprehensive
- `PersistFeatureOverrides` function itself propagates errors properly

---

## Remediation Plan

### Immediate (fixes exist on main — port to worktree):

| Priority | Finding | Work | Main reference |
|----------|---------|------|----------------|
| 1 | B2 | Add `"tier": "bug_fix"` to `wantState` | `entities_test.go` on main |
| 2 | B4 | Add error logging for `PersistFeatureOverrides` | `entity_tool.go#L692, L699` on main |
| 3 | B3 | Change `tv["blocking_fail"]` → `tv["blocking"]` | `fast_track_integration_test.go` on main |
| 4 | B5 | Cherry-pick 7 deliverable files from main to worktree | Various `.kbz/roles/`, `.kbz/skills/`, `work/` |
| 5 | B1 | Port `checkTransitionValidator` wiring from main | `entity_tool.go#L914-956` on main |
| 6 | B6 | Add tier immutability/validation to `UpdateEntity` | New implementation |

### Structural remediation (before merge):
- Rebase the feature branch onto current `main` to resolve drift and merge conflicts
- Ensure all deliverables are committed to the feature branch, not directly to `main`

---

## Escalation Note

Review cycle cap (3/3) reached. Per the orchestrate-review skill: "If issues persist after
3 cycles, escalate to human via checkpoint." The recurring pattern is that validator
dispatch wiring has never been completed on the feature branch despite the fix existing
on `main`.
