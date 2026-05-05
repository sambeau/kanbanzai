# Review: FEAT-01KQSP41PE6JP — Fast-Track Architecture Core Implementation (Cycle 2)

| Field        | Value                                          |
|--------------|------------------------------------------------|
| Date         | 2026-05-05                                     |
| Feature      | FEAT-01KQSP41PE6JP (B48-F1)                    |
| Plan         | P43-fast-track-architecture                    |
| Batch        | B48-fast-track-impl                            |
| Review Cycle | 2                                              |
| Reviewers    | reviewer-conformance, reviewer-quality, reviewer-testing, reviewer-security |
| Spec         | FEAT-01KQSP41PE6JP/spec-p43-spec-fast-track-architecture |

---

## Aggregate Verdict: **rejected** (needs_remediation)

Six blocking findings remain. Four of the five corrected cycle-1 blocking findings
(F2, F3, F4 and part of F1) were successfully resolved, but F1's sub-issue 3 (pipeline
auto-approval) carries over; two additional blocking gaps were introduced by the cycle-1
fixes; one security-grade finding was newly identified; and a structural workflow violation
was discovered during diff analysis.

---

## Cycle-1 Finding Closure

| Cycle-1 Finding | Status | Evidence |
|-----------------|--------|----------|
| F1 sub-1: `SpawnAgentDispatcher.Dispatch` returned error | ✅ Resolved | `validator_dispatch.go` — prompt built, no error returned |
| F1 sub-2: `ValidateTransition` always returned `(nil, nil)` | ✅ Resolved | `transition_validator.go` — real gate-mode evaluation with override/human/auto paths |
| F1 sub-3: pipeline auto-approval not connected | ❌ **Still present** → BLOCK-2 | `tryAutoValidate` dispatches but no code path calls `doc(approve)` on pass |
| F2: Bug.Tier absent; CreateBug didn't call `inferTier` | ✅ Resolved | `model/entities.go:L499`, `service/entities.go:L429,443` |
| F3: retro signal detection absent in `inferTier` | ✅ Resolved | `service/entities.go:L1091-1093` — `"retro"` tag added |
| F4: conditional gate logic for `retro_fix` deferred | ✅ Resolved | `transition_validator.go:L152-223` — `evaluateConditional` fully implemented |

---

## Structural Issue: Deliverables Committed to Wrong Branch

During diff analysis (`git diff main --diff-filter=D`), it was discovered that seven P43
deliverables are present on `main` but **absent from the feature branch**. These files
were committed to `main` directly rather than through the worktree:

| File | Note |
|------|------|
| `.kbz/roles/spec-validator.yaml` | Exists on main only (commit `5efea73f`) |
| `.kbz/roles/plan-validator.yaml` | Exists on main only |
| `.kbz/skills/validate-spec/SKILL.md` | Exists on main only (commit `05894807`) |
| `.kbz/skills/validate-review/SKILL.md` | Exists on main only (commit `3ae576fe`) |
| `work/P43-fast-track-architecture/validator-rubrics/spec-validator-rubrics.md` | Exists on main only |
| `work/P43-fast-track-architecture/validator-rubrics/plan-validator-rubrics.md` | Exists on main only |
| `work/P43-fast-track-architecture/validator-rubrics/review-gate-validator-rubrics.md` | Exists on main only |

**Impact:**
- The feature branch cannot be reviewed as a self-contained unit for these artifacts.
- A squash-merge will not lose these files (they already exist on `main`), but the
  feature branch's commit history does not reflect the actual implementation work.
- These files were reviewed for spec conformance by reading them from `main` directly.

The only validator artifacts properly committed to the feature branch are:
`review-gate-validator.yaml`, `validate-plan/SKILL.md`, and `validate-plan/references/examples.md`.

---

## Per-Dimension Verdicts

| Dimension | Verdict | Notes |
|-----------|---------|-------|
| spec_conformance | fail | Two blocking gaps; key artifacts on wrong branch |
| implementation_quality | fail | F1 dispatch wiring incomplete in transition path |
| error_handling | pass | N2 addressed; errors now logged |
| naming_consistency | pass | N4 and N14 addressed |
| go_idioms | pass_with_notes | N8, N9 addressed; ValidatorContext fields empty |
| package_cohesion | concern | N3 unaddressed |
| dead_code | concern | ValidatorSummary.Prompt never consumed |
| test_isolation | pass | All tests hermetic |
| assertion_quality | fail | BLOCK-3, BLOCK-5 test regressions |
| boundary_value_coverage | fail | retro-tag inference branch untested |
| error_path_coverage | pass_with_notes | Dispatch error path untested |
| spec_coverage_mapping | fail | BLOCK-4, BLOCK-5 + blocking validator tests always skip |
| integration_test_design | concern | Blocking validator tests never exit skip path |
| input_validation | pass_with_notes | isDocOnlyChange lacks path normalization |
| trust_boundary | concern | SpawnAgentDispatcher produces unconditional pass if wired to transition path |
| integrity | fail | Auto-approval gate not enforced; PersistFeatureOverrides error discarded |
| audit_trail | fail | Override record can be silently lost |
| privilege_escalation | concern | Tier downgrade unrestricted (no spec anchor; security concern) |

---

## Review Unit Dispatch

| Unit | Reviewer |
|------|----------|
| Spec conformance across all 16 deliverable files | reviewer-conformance |
| Core Go implementation (9 production files) | reviewer-quality |
| All test files + corresponding production files | reviewer-testing |
| Security surfaces (7 files) | reviewer-security |

---

## Blocking Findings

### BLOCK-1: `dispatchSvc` never wired — transition-path validator enforcement absent
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-003, AC-NF-003, REQ-TRANS-001, REQ-TRANS-002
- **Locations:**
  - `internal/mcp/entity_tool.go:L908-951` — `checkTransitionValidator` creates `TransitionValidatorDispatcher` without `.WithDispatch()`
  - `internal/validate/transition_validator.go:L131-142` — `AUTO_PLACEHOLDER` path always taken when `dispatchSvc == nil`
  - `internal/mcp/fast_track_integration_test.go:L199-202` — `t.Skip("transition_validator not wired")` confirms the gap
- **Description:** Every auto-mode gate transition returns a placeholder pass. `BuildTransitionValidatorError` is unreachable. No real validator is ever dispatched from the transition path, so AC-TRANS-001 (spec blocks on S1 failure) and AC-TRANS-003 (plan blocks on D13 failure) are not satisfied.
- **Secondary gap:** `runAutoValidation` builds `ValidatorContext` with `FeatureID` only; `DocumentPath`, `ParentDocPath`, and `RubricPath` are empty (`transition_validator.go:L247-250`). These fields must be populated for `buildPrompt` to produce a meaningful prompt.
- **Remediation:**
  1. Call `.WithDispatch(spawnAgentDispatcher)` inside `checkTransitionValidator`.
  2. Populate document-path fields in `runAutoValidation` by resolving the feature's current spec or dev-plan.
  3. Remove `t.Skip` from the three blocking-validator integration tests once wired.

### BLOCK-2: Auto-approval not implemented after validator pass
- **Severity:** blocking
- **Spec anchors:** AC-PIPE-002, REQ-PIPE-003
- **Location:** `internal/mcp/doc_tool.go:L445-460` (`tryAutoValidate`)
- **Description:** `tryAutoValidate` returns `{status:"dispatched"}` and exits. No subsequent code path calls `doc(approve)` when the validator returns pass. The validator prompt (`buildPrompt` instructions 1–7) contains no approval step. On pass, documents remain in draft status indefinitely. AC-PIPE-002 is not satisfied.
- **Remediation:** When `tryAutoValidate` resolves with a pass verdict, call `docApproveOne` programmatically, or include an explicit approval instruction in the validator prompt that the spawned sub-agent can execute.

### BLOCK-3: Test regression — `TestEntityService_CreateBug_AppliesDefaults` fails after F2 fix
- **Severity:** blocking
- **Spec anchor:** REQ-INFER-002(b)
- **Location:** `internal/service/entities_test.go:L149-184`
- **Description:** F2 made `inferTier` return `"bug_fix"` for every default bug (stored via `bugFields`). The pre-existing test compares against a `wantState` that does not include `"tier"`, causing `reflect.DeepEqual` to fail on every run.
- **Remediation:** Add `"tier": "bug_fix"` to `wantState` in `TestEntityService_CreateBug_AppliesDefaults`.

### BLOCK-4: `"retro"` tag → `TierRetroFix` inference branch has no test
- **Severity:** blocking
- **Spec anchors:** REQ-INFER-002(a), AC-INFER-001
- **Location:** `internal/service/entities.go:L1091-1093`
- **Description:** F3 added the retro-tag check to `inferTier`. No test exercises this branch. `TestFastTrack_TierInference_*` covers `"security"` → `critical` and explicit override; the retro-tag path is a dead zone.
- **Remediation:** Add a test case: create a feature with tag `"retro"` (no explicit tier), assert `tier == "retro_fix"`.

### BLOCK-5: Key-name mismatch in integration test assertions — dormant cascade
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-003, AC-RVW-002
- **Locations:**
  - `internal/mcp/entity_tool.go:L628` — response uses key `"blocking"`
  - `internal/mcp/fast_track_integration_test.go:L213, L262, L304` — tests check `tv["blocking_fail"]`
- **Description:** Three integration tests check `tv["blocking_fail"]` (nil/false) but the entity tool returns `tv["blocking"]`. Tests currently skip via `t.Skip()`, masking the mismatch. Resolving BLOCK-1 without fixing BLOCK-5 will immediately produce a false-failure cascade across all three AC-TRANS and AC-RVW-002 tests.
- **Remediation:** Change `tv["blocking_fail"]` → `tv["blocking"]` in the three test assertions (or align the entity tool response key to `"blocking_fail"`).

### BLOCK-6: `PersistFeatureOverrides` error silently discarded
- **Severity:** blocking
- **Spec anchor:** REQ-TRANS-007 ("override usage tracked as metric")
- **Location:** `internal/mcp/entity_tool.go:L693, L697`
- **Description:** At both call sites, the error return of `entitySvc.PersistFeatureOverrides` is not checked. The status transition has already committed (`UpdateStatus` ran earlier in the handler). If the override record write fails, the feature appears to have transitioned without a gate bypass and the override metric is silently lost.
- **Remediation:** Check and log (or return) the `PersistFeatureOverrides` error at both sites. Minimum: structured `log.Printf` with `WARNING` prefix; ideal: surface as a non-fatal warning in the tool response.

---

## Non-Blocking Findings

| ID | Description | Location | Cycle-1 Origin |
|----|-------------|----------|----------------|
| NB-1 | `inferTier` checks critical/security before retro, reversing REQ-INFER-002's priority order | `service/entities.go:L1082-1093` | New |
| NB-2 | `FilesModified` never populated in `checkTransitionValidator` → `COND_DOCS_ONLY`/`COND_IMPL_CHANGE` unreachable | `entity_tool.go:L918-930` | New |
| NB-3 | `ValidatorContext` document fields empty — wired dispatch would produce content-less prompt | `transition_validator.go:L247-250` | New |
| NB-4 | Transition validator override not recorded as metric (gate override IS recorded; validator-path override is not) | `entity_tool.go:L626-632` | New |
| NB-5 | `ValidatorSummary.Prompt` generated by every `Dispatch()` call but never consumed | `validator_dispatch.go:L155` | New |
| NB-6 | Dead tier guard `"fast-track"` in `transition_validator.go` — no config tier bears this name | `transition_validator.go:L118-120` | New |
| NB-7 | Cycle cap bypassable via `IncrementFeatureReviewCycle` store error — counter stalls | `doc_tool.go:L1444-1448` | New |
| NB-8 | `isDocOnlyChange` raw prefix matching — path traversal not guarded | `transition_validator.go:L163-173` | New |
| NB-9 | `SpawnAgentDispatcher` wired to transition path produces unconditional pass; no guardrail | `validator_dispatch.go:L125-140` | New |
| NB-10 | Tier downgrade unrestricted via `UpdateEntity` — no spec anchor for immutability, but security concern | `service/entities.go:L916-921` | New |
| NB-11 | N3 (cycle 1): `tryAutoValidate` wide import surface — refactor note added, no extraction | `doc_tool.go:L307-469` | N3 carry-over |
| NB-12 | N5 (cycle 1): wall-clock timing tests flaky — comment added, no fix | `fast_track_integration_test.go:L697-762` | N5 carry-over |
| NB-13 | N11 (cycle 1): `stageBindingsWithValidators` hardcoded — no comparison to production bindings | `fast_track_integration_test.go:L59-122` | N11 carry-over |
| NB-14 | N15 (cycle 1): AC-RVW-003 (R4 check) no dedicated integration test | — | N15 carry-over |
| NB-15 | N16 (cycle 1): blocking validator tests always skip (no injectable mock `dispatchSvc`) | `fast_track_integration_test.go:L200-264` | N16 carry-over |
| NB-16 | N17 (cycle 1): fail→not-approved pipeline scenario untested | — | N17 carry-over |
| NB-17 | Stale test comment "dispatcher returns nil (no-op)" contradicts actual behaviour | `transition_validator_test.go:L134-138` | New |

---

## Finding Summary

| Classification | Count |
|----------------|-------|
| Blocking | 6 |
| Non-blocking | 17 |
| **Total** | **23** |

---

## What Went Well

- F2, F3, and F4 are cleanly resolved with correct implementations and targeted code changes.
- F1 sub-1 and sub-2: `SpawnAgentDispatcher.Dispatch` and `ValidateTransition` are now
  meaningful implementations, not stubs. The `WithDispatch` + `runAutoValidation` architecture
  is sound and ready to be wired.
- N1 partially addressed: `CheckQualityReviewSignal` now accepts a tier config map for
  dynamic cycle-cap thresholding.
- N2, N4, N8, N9, N14 addressed: error logging, naming disambiguation, `*bool` removed,
  linear scan → map, `PlanStatus` deprecation annotation.
- `evaluateConditional` logic is complete and architecturally sound for the conditional
  gate use-case.
- `ValidatorDispatcher` interface design is clean; P44 model-routing swap remains feasible.
- Override mechanism and human escape-hatch work correctly and are tested.
- Config schema is comprehensive.

---

## Remediation Plan

| Priority | Finding | Work |
|----------|---------|------|
| 1 | BLOCK-3 | Add `"tier": "bug_fix"` to `wantState` in `TestEntityService_CreateBug_AppliesDefaults` |
| 2 | BLOCK-4 | Add retro-tag inference test |
| 3 | BLOCK-5 | Fix `tv["blocking_fail"]` → `tv["blocking"]` in three test assertions |
| 4 | BLOCK-6 | Check and log `PersistFeatureOverrides` error at `entity_tool.go:L693,697` |
| 5 | BLOCK-1 | Wire `dispatchSvc` into `checkTransitionValidator`; populate `ValidatorContext` document fields; remove `t.Skip` from blocking tests |
| 6 | BLOCK-2 | Implement auto-approval on pass verdict in the doc-registration pipeline |

BLOCK-3 through BLOCK-6 are small isolated fixes that should be completed first. BLOCK-1
and BLOCK-2 are interdependent and architecturally heavier; tackle them together after
the test infrastructure is clean.

**Structural note:** The seven deliverables committed to `main` instead of the feature
branch do not prevent a correct merge (they already exist on `main`). The workflow
violation should be noted in the feature retrospective.
