# Review: FEAT-01KQSP41PE6JP — Fast-Track Architecture Core Implementation

| Field        | Value                          |
|--------------|--------------------------------|
| Date         | 2026-05-04                     |
| Feature      | FEAT-01KQSP41PE6JP (B48-F1)    |
| Plan         | P43-fast-track-architecture    |
| Batch        | B48-fast-track-impl            |
| Reviewers    | reviewer-conformance, reviewer-quality, reviewer-testing |
| Spec         | FEAT-01KQSP41PE6JP/spec-p43-spec-fast-track-architecture |

## Aggregate Verdict: **rejected** (needs_remediation)

The feature has 4 blocking findings that must be resolved before approval.

## Per-Dimension Verdicts

| Dimension | Verdict | Reviewer |
|-----------|---------|----------|
| spec_conformance | concern | reviewer-conformance |
| implementation_quality | fail | reviewer-quality |
| test_adequacy | concern | reviewer-testing |
| error_handling | pass_with_notes | reviewer-quality |
| package_cohesion | pass_with_notes | reviewer-quality |
| dead_code | concern | reviewer-quality |
| naming_consistency | pass | reviewer-quality |
| go_idioms | pass_with_notes | reviewer-quality |
| test_isolation | pass_with_notes | reviewer-testing |
| assertion_quality | concern | reviewer-testing |
| boundary_value_coverage | concern | reviewer-testing |
| error_path_coverage | pass | reviewer-testing |
| spec_coverage_mapping | concern | reviewer-testing |
| integration_test_design | pass_with_notes | reviewer-testing |

## Review Unit Dispatch

| Unit | Files | Reviewers |
|------|-------|-----------|
| All units (17 files) | Full file set | reviewer-conformance |
| Core Go code (11 files) | config, model, service, health, binding, gate, validate, mcp | reviewer-quality |
| All test + prod (12 files) | Test files + production code | reviewer-testing |

## Blocking Findings

### F1: Validator dispatch is a stub — validation pipeline not wired
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-002, AC-TRANS-003, AC-TRANS-004, AC-PIPE-002, AC-PIPE-003, REQ-TRANS-001 through REQ-TRANS-003
- **Locations:**
  - `internal/validate/validator_dispatch.go#L121-128` — `SpawnAgentDispatcher.Dispatch` returns `"not yet implemented"` error
  - `internal/validate/transition_validator.go#L72-97` — `ValidateTransition` always returns `(nil, nil)`
  - `internal/mcp/doc_tool.go#L322-463` — auto-approval on pass, findings on fail not connected
- **Root cause:** The dispatch infrastructure is complete but `spawn_agent` invocation and result handling is a stub.
- **Remediation:**
  1. Wire `SpawnAgentDispatcher.Dispatch` to call `spawn_agent` and parse results
  2. Handle pass/pass_with_notes/fail outcomes in `tryAutoValidate`
  3. Wire `TransitionValidatorDispatcher.ValidateTransition` to dispatch and return real results

### F2: Bug entities cannot receive a tier — AC-INFER-001 not satisfied
- **Severity:** blocking
- **Spec anchor:** AC-INFER-001, REQ-INFER-002(b)
- **Location:** `internal/service/entities.go#L1064-1080`, `internal/model/entities.go#L469-493`
- **Description:** Bug model has no `Tier` field; `CreateBug` doesn't call `inferTier`.
- **Remediation:** Add `Tier` to Bug model and call `inferTier` in `CreateBug`, or update spec.

### F3: Retro signal inference not implemented — REQ-INFER-002(a) not satisfied
- **Severity:** blocking
- **Spec anchor:** REQ-INFER-002(a)
- **Location:** `internal/service/entities.go#L1064-1080`
- **Description:** `inferTier` checks only `critical`/`security` tags, not retro signals.
- **Remediation:** Add retro signal detection to `inferTier`.

### F4: Conditional gate logic for retro_fix deferred — AC-TIER-002, AC-TIER-003 not implemented
- **Severity:** blocking
- **Spec anchors:** AC-TIER-002, AC-TIER-003, REQ-TIER-004
- **Location:** `internal/validate/transition_validator.go#L102`
- **Description:** Conditional gate mode defers to P44 with no implementation.
- **Remediation:** Implement file-change-type detection or formally defer with spec update.

## Non-Blocking Findings

### N1: Quality review signal heuristic fragile for critical tier
- **Location:** `internal/health/quality_review.go#L54-57`
- **Recommendation:** Pass tier config to compute threshold dynamically.

### N2: tryAutoValidate swallows PersistFeatureBlockedReason errors
- **Location:** `internal/mcp/doc_tool.go#L402`
- **Recommendation:** Log or return the error.

### N3: tryAutoValidate has wide import surface
- **Location:** `internal/mcp/doc_tool.go#L322-463`
- **Recommendation:** Extract pipeline orchestration to `internal/validate/pipeline.go`.

### N4: ValidatorResult vs ValidatorSummary naming collision
- **Location:** `internal/validate/transition_validator.go#L23`, `internal/validate/validator_dispatch.go#L49`
- **Recommendation:** Rename to `TransitionValidatorResult` / `DocumentValidatorSummary`.

### N5: Wall-clock timing tests for AC-NF-001 are flaky
- **Location:** `internal/mcp/fast_track_integration_test.go#L599-633`
- **Recommendation:** Test via tool-call counting, not wall-clock.

### N6: Integration tests log without asserting blocking
- **Location:** `internal/mcp/fast_track_integration_test.go#L155-286`
- **Recommendation:** Add blocking assertions or rename tests.

### N7–N17
- Missing edge case test for max_cycles <= 0
- AC-PIPE-003 pass→auto-approve only tested to dispatch point
- AC-PIPE-005 fail→not-approved has no test
- AC-TRANS-006 quality review signal has no direct test
- Config tests don't use t.Parallel()
- Test stage bindings may drift from production
- Bug Model PlanStatus naming ambiguity
- Enabled *bool complexity in FastTrackConfig
- validateGateMode linear scan vs map
- AC-RVW-003 no dedicated integration test for R4
- Tests for fully-wired validator dispatch needed post-wiring

## Finding Summary

| Classification | Count |
|----------------|-------|
| Blocking | 4 |
| Non-blocking | 17 |
| **Total** | **21** |

## What Went Well

- Architecture design is solid: `ValidatorDispatcher` interface, `FastTrackConfig`, `TransitionValidatorDispatcher`
- Stage bindings integration clean: transition validators configured in YAML, not hardcoded
- Session management abstraction correct: `ValidatorContext` carries only document/parent/rubric
- Tier inference for features correct and well-tested (11 tests)
- Config schema comprehensive with 62 passing tests
- Rubric files thorough with real Kanbanzai document examples
- Override mechanism complete with recording
- Cycle tracking and escalation work correctly

## Remediation Plan

1. **(F1)** Wire `SpawnAgentDispatcher.Dispatch` → `spawn_agent` → parse results → handle in pipeline
2. **(F2)** Add Bug.Tier + inferTier call in CreateBug (or update spec)
3. **(F3)** Add retro signal detection to inferTier
4. **(F4)** Implement file-change-type detection or formally defer to P44
