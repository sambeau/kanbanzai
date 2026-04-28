# Plan Review: P17 — Workflow and Tooling 3.0

**Plan:** P17-workflow-and-tooling-3.0  
**Review date:** 2026-04-02  
**Reviewer:** Orchestrator (parallel sub-agent review)  
**Scope:** All 8 features in the plan

---

## Summary Verdict

**overall verdict: changes_required**

7 of 8 features have blocking findings. A total of 24 blocking findings were identified across the plan.
The single feature without blocking findings — `decomposition-quality-validation` — is approved with
follow-up notes. All other features require rework before they can be transitioned to `done`.

The most pervasive issue is **test coverage**: four features (mandatory-stage-gates,
stage-aware-context-assembly, review-rework-loop, document-structural-checks) have critical paths that
are entirely untested at the handler or integration level. Two additional features
(tool-description-audit, action-pattern-logging) have acceptance criteria that are unmet by the
delivered artifacts (missing test session records; permanently-nil metric lookup in the CLI). The
remaining blocking findings are implementation conformance gaps — logic implemented incorrectly or
required MCP surface absent.

---

## Per-Feature Verdicts

| Feature | Slug | Verdict | Blocking | Non-blocking |
|---------|------|---------|----------|--------------|
| FEAT-01KN58J24S2XW | mandatory-stage-gates | changes_required | 2 | 7 |
| FEAT-01KN58J257X02 | tool-description-audit | changes_required | 4 | 9 |
| FEAT-01KN58J25K4QD | stage-aware-context-assembly | changes_required | 4 | 10 |
| FEAT-01KN58J2606B0 | review-rework-loop | changes_required | 5 | 9 |
| FEAT-01KN58J26CH63 | decomposition-quality-validation | approved_with_followups | 0 | 8 |
| FEAT-01KN58J26RSB6 | document-structural-checks | changes_required | 5 | 8 |
| FEAT-01KN58J275BWJ | action-pattern-logging | changes_required | 3 | 8 |
| FEAT-01KN58J27H83N | binding-registry-gate-integration | changes_required | 1 | 10 |
| **Total** | | | **24** | **69** |

---

## Per-Dimension Verdicts (Plan-Level)

| Dimension | Verdict | Notes |
|-----------|---------|-------|
| Specification Conformance | fail | Multiple blocking implementation gaps across 5 features |
| Implementation Quality | pass_with_notes | No blocking quality issues; several non-trivial correctness risks noted |
| Test Adequacy | fail | Systematic absence of handler/integration tests for new behaviour in 4 features |
| Documentation Currency | pass_with_notes | Minor issues only; no blocking documentation problems |
| Workflow Integrity | pass_with_notes | No blocking workflow state issues |

---

## Blocking Findings

### FEAT-01KN58J24S2XW — mandatory-stage-gates

**B-01** `[blocking]` `internal/mcp/entity_tool.go:594-601` — **Specification Conformance**  
Advance path (`advance: true`) dispatches to `entityAdvanceFeature` without validating that
`override_reason` is non-empty when `override: true`. A call with `advance: true, override: true,
override_reason: ""` bypasses all gates and persists override records with empty reasons. The
validation at line 721 only executes in the non-advance branch. This violates FR-012 ("the tool MUST
reject the transition with an error requiring a reason") and §1.5 AC-2 ("Every override produces a
persisted record with a non-empty reason string. There is no way to override without providing a
reason.").  
spec_ref: §1.3.3 FR-012, §1.5 AC-2

**B-02** `[blocking]` `internal/mcp/entity_transition_gate_test.go` — **Test Adequacy**  
No MCP-layer integration test for `advance: true, override: true, override_reason: ""`. The gap
described in B-01 is undetected because there is no companion test. The advance path override
validation path is also not covered for the new `needs-rework` gated transitions at the MCP handler
level.  
spec_ref: §1.5 AC-2, §1.5 AC-6

---

### FEAT-01KN58J257X02 — tool-description-audit

**B-03** `[blocking]` `internal/mcp/decompose_tool.go:69-71` — **Specification Conformance**  
`decomposePropose`, `decomposeReview`, `decomposeApply`, and `decomposeSlice` use bare `inlineErr`
messages that do not follow the three-part FR-008 template. `decompose` is an FR-009 priority handler;
all input-validation paths must follow "Cannot {action}: {reason}.\n\nTo resolve:\n  {recovery_step}".
Compare: `checkpoint_tool.go` uses `inlineErr` correctly with the full template.  
spec_ref: §1.3.3 FR-008, FR-009, AC-3

**B-04** `[blocking]` `general` — **Specification Conformance**  
No token budget verification mechanism exists. `annotations_test.go` tests MCP hint annotations but
does not count description tokens. No script, CI step, or test exercises the ≤200-token constraint
against any tool description.  
spec_ref: §1.3.1 FR-005, AC-4

**B-05** `[blocking]` `work/evaluation/` — **Test Adequacy**  
No agent-driven test session records exist. The directory contains only `README.md` and `scenarios/`.
There is no `results/` subdirectory, no per-scenario pass/fail record, no documented observed tool
sequences, and no record of description rewrites made in response to failures. AC-5 requires
"documented results of at least one agent-driven test session per scenario covering that tier's tools."
This acceptance criterion is wholly unmet for all three priority tiers.  
spec_ref: §1.3.4 FR-013, FR-014, AC-5

**B-06** `[blocking]` `work/evaluation/scenarios/` — **Test Adequacy**  
`handoff` (P1), `merge` (P2), and `pr` (P2) appear in zero scenarios. `status` (P1) appears only as an
optional step in scenario 12. FR-011 requires scenarios to "collectively cover all Priority 1 and
Priority 2 tools from FR-007." At minimum three new scenarios are required.  
spec_ref: §1.3.4 FR-011, AC-6

---

### FEAT-01KN58J25K4QD — stage-aware-context-assembly

**B-07** `[blocking]` `internal/mcp/assembly.go:assembleContext` — **Specification Conformance**  
`IncludeReviewRubric`, `IncludeTestExpect`, `IncludeImplGuidance`, `IncludePlanGuidance`, and
`SpecMode` are declared in `StageConfig` and set for every stage, but **none are consulted by
`assembleContext`**. Only `IncludeFilePaths` drives a real content decision. The FR-005 acceptance
criteria "Context assembled for a task in `reviewing` includes the review rubric and verdict format"
and "Context assembled for a task in `needs-rework` includes previous review findings when available"
are unmet.  
spec_ref: §FR-005, §1.5 AC rows FR-005

**B-08** `[blocking]` `internal/mcp/assembly.go:asmExtractSpecSections` — **Specification Conformance**  
`SpecMode` (`"full"` vs `"relevant-sections"`) is never passed to or checked by
`asmExtractSpecSections`; extraction behaviour is identical for all stages. The AC "Context assembled
for `specifying` includes the full design document content (not just relevant sections)" is therefore
unmet.  
spec_ref: §FR-005, §1.5 AC row FR-005

**B-09** `[blocking]` No handler-level test for FR-001 — **Test Adequacy**  
No test calls `handoffTool` with a task whose parent feature is in `proposed` or `done` and asserts a
`stage_validation` error code is returned. `assembly_validate_test.go` tests the function in
isolation; `stage_integration_test.go:TestStageIntegration_ValidationRejection` also tests only the
function.  
spec_ref: §FR-001, §1.5 AC row FR-001

**B-10** `[blocking]` No handler-level test for FR-002 — **Test Adequacy**  
No test calls `nextTool` (claim mode) with a feature in `proposed` and asserts (a) an error is
returned and (b) the task status remains `ready`.  
spec_ref: §FR-002, §1.5 AC row FR-002

---

### FEAT-01KN58J2606B0 — review-rework-loop

**B-11** `[blocking]` `general` — **Test Adequacy**  
No tests exist for any behaviour specified in FR-001 through FR-010. A project-wide grep for
`IncrementFeatureReviewCycle`, `PersistFeatureBlockedReason`, `ReviewCycle`, `review_cycle`,
`blocked_reason`, `DefaultMaxReviewCycles`, and `ReviewCapReached` returns zero matches in any
`*_test.go` file.  
spec_ref: §1.5

**B-12** `[blocking]` `internal/service/prereq_test.go` — **Test Adequacy**  
No test covers `CheckTransitionGate("reviewing", "needs-rework", ...)`. The cap-check branch — the
most critical new logic — has no test. Missing cases: (a) cap reached at cycle 3 returns unsatisfied
+ `ReviewCapReached: true`; (b) cycle 2 below cap allows transition; (c) feature at cap with pass
verdict transitions to done normally.  
spec_ref: §1.5/FR-005

**B-13** `[blocking]` `internal/service/review_rework.go` — **Test Adequacy**  
No test file exists for `review_rework.go`. `IncrementFeatureReviewCycle` and
`PersistFeatureBlockedReason` have no unit tests covering: round-trip persistence, zero-value
initialisation for legacy records, counter increments from both 0→1 and N→N+1, clearing vs setting
`blocked_reason`.  
spec_ref: §1.5/FR-001, §1.5/FR-002, §1.5/FR-007

**B-14** `[blocking]` `internal/mcp/entity_tool_test.go` / `entity_transition_gate_test.go` — **Test Adequacy**  
No integration test covers the end-to-end cap-block path: cap reached → `blocked_reason` persisted →
checkpoint created → response contains `blocked_reason` and `checkpoint_id`. No test verifies that a
second `reviewing → needs-rework` attempt after the block is also rejected.  
spec_ref: §1.5/FR-005, §1.5/FR-006, §1.5/FR-007

**B-15** `[blocking]` `internal/mcp/handoff_tool_test.go` — **Test Adequacy**  
No test covers the re-review signal injection. Missing: (a) task whose parent feature is `reviewing`
with `review_cycle: 1` — no re-review guidance in prompt; (b) `review_cycle: 2` — guidance section
present and contains cycle number; (c) `review_cycle: 2` with existing `instructions` — guidance is
prepended, not lost.  
spec_ref: §1.5/FR-008

---

### FEAT-01KN58J26RSB6 — document-structural-checks

**B-16** `[blocking]` `internal/service/structural_gate.go:81-92` — **Specification Conformance**  
Acceptance criteria check fires at the wrong gate. The `if info.docType == "specification"` branch
runs at `specifying→dev-planning`, which causes `CheckAcceptanceCriteria` to execute there. Per
FR-001, acceptance criteria must be checked at `dev-planning→developing` on the approved
specification, not at `specifying→dev-planning`.  
spec_ref: §1.3.1 FR-001

**B-17** `[blocking]` `internal/service/structural_gate.go` — **Specification Conformance**  
At `dev-planning→developing`, `runStructuralChecksForGate` only receives the dev-plan document. It
never loads or checks the specification document at this gate. FR-001 mandates two document checks
here: required sections on the dev-plan AND acceptance criteria on the specification.  
spec_ref: §1.3.1 FR-001, §1.3.4 FR-006

**B-18** `[blocking]` `internal/service/structural_gate.go:84-91` — **Specification Conformance**  
Cross-reference check only populates `designPaths`/`designIDs` from `feature.Design`. When the
feature has no own design document but the parent plan owns one, both sets will be empty and
`CheckCrossReference` will always fail. FR-005 explicitly requires plan-level design documents to
satisfy the check.  
spec_ref: §1.3.3 FR-005

**B-19** `[blocking]` `internal/mcp/doc_tool.go` / `internal/structural/promotion.go` — **Specification Conformance**  
`PromotionState.RecordFalsePositive` is implemented but is never reachable by any MCP tool action.
FR-014 requires a callable mechanism (accessible to agents) for reporting false positives that resets
the consecutive-clean counter and demotes hard-gate checks. The internal method is dead code from an
MCP perspective.  
spec_ref: §1.3.7 FR-014

**B-20** `[blocking]` `internal/service/documents_test.go` — **Test Adequacy**  
No test exercises the quality evaluation approval gate with `RequireForApproval: true`. Three distinct
behaviours required by FR-018 are untested at the service level: (1) missing evaluation → blocked, (2)
`pass: false` → blocked with dimension scores, (3) passing evaluation → allowed.  
spec_ref: §1.5 FR-018

---

### FEAT-01KN58J275BWJ — action-pattern-logging

**B-21** `[blocking]` `cmd/kanbanzai/metrics_cmd.go:62` — **Specification Conformance**  
`ComputeMetrics(input, nil)` hard-codes `nil` for the `StageFeatureLookup` parameter on every
invocation. Because `ComputeMetrics` gates the `TimePerStage` and `RevisionCycleCounts` computations
behind `if lookup != nil`, these two metrics are permanently empty at the CLI level. Two of the five
required metrics are unreachable from the only user-facing interface.  
spec_ref: §FR-010, §FR-012

**B-22** `[blocking]` `internal/actionlog/hook.go:Wrap` — **Specification Conformance**  
Stage is resolved after the inner handler returns. For lifecycle-transition calls the entity service
already reflects the post-transition state when the lookup runs, so the logged `stage` field records
the new stage rather than the stage at the time of the call. The spec acceptance criterion is
explicit: "The stage reflects the state before any transition the current call may perform."  
spec_ref: §FR-005 (AC bullet 4)

**B-23** `[blocking]` `internal/actionlog/scenario.go:validateScenario` — **Specification Conformance**  
The `starting_state` and `expected_pattern` fields are declared required in FR-014 but are not checked
by `validateScenario`. A scenario file with either field absent is accepted silently rather than
rejected with an error. The acceptance criterion states: "A scenario file missing any required field
is rejected with an error identifying the missing field."  
spec_ref: §FR-014 (AC bullet 2)

---

### FEAT-01KN58J27H83N — binding-registry-gate-integration

**B-24** `[blocking]` `internal/gate/evaluator.go` + `internal/binding/loader.go` — **Specification Conformance**  
`LoadBindingFile` uses `decoder.KnownFields(true)`, which rejects the entire binding file on an
unknown field in the `prerequisites` block, propagating up to `RegistryCache.Get()` as a load
failure and triggering hardcoded fallback for all stages. The spec requires the specific gate to fail
with "an error message that names the unrecognised prerequisite type." The `dispatch` function's
"unknown prerequisite type" error path (evaluator.go:89–94) is dead code in practice because
`KnownFields(true)` prevents unknown fields from ever reaching the dispatcher. AC-7 ("extensibility
is demonstrated") cannot be met as described.  
spec_ref: §1.3.1 FR-004, §1.5 AC-7

---

## Non-Blocking Findings (Selected)

The following are notable non-blocking findings. The full list per feature is in the sub-agent outputs
archived below.

### mandatory-stage-gates
- `internal/service/entities.go` — `checkpoint_id` is not serialised in override records; health checks for checkpoint overrides are non-functional as written.
- `internal/mcp/health_gate_source.go` — `allGatedStages` is missing `done` (for the `reviewing→done` gate), producing misleading health coverage output.
- No benchmark tests for gate evaluation (NFR-001).

### tool-description-audit
- Several tool handlers (`entity_tool.go` transition path, `merge_tool.go`, `conflict_tool.go`, `cleanup_tool.go`, `retro_tool.go`) use bare error messages without the three-part template. These are P2/P3 handlers (FR-010 scope) but are still inconsistent with the stated standard.
- Evaluation scenario count (20) exceeds the FR-011 upper bound of 10. The excess scenarios are useful, but the deviation should be acknowledged.

### stage-aware-context-assembly
- `internal/mcp/handoff_tool.go` — lifecycle validation is only wired into the legacy path; the pipeline path bypasses it.
- `internal/mcp/assembly_validate.go` — blank feature status returns `("", nil)`, silently allowing non-stage-aware assembly for corrupted entities.
- Five `StageConfig` fields that are not yet consumed by `assembleContext` carry no stub annotation, making the dead configuration misleading to future readers.

### review-rework-loop
- Checkpoint is created with `checkpoint.NewStore(entitySvc.Root())` instead of the injected `checkpointStore`, bypassing DI.
- When gateRouter returns `Source == "registry"`, `ReviewCapReached` is not propagated from `GateResult`, silently bypassing cap-block behaviour for registry-sourced gates.
- `PersistFeatureBlockedReason` error is discarded with `_ =`, leaving entity unblocked while caller receives a blocked-state response.
- No guard against duplicate checkpoint creation on retried cap-block attempts.

### decomposition-quality-validation
- Separator regexes in `multiAgentFinding` are compiled on every invocation; these are fixed constants and should be precompiled.
- `clauseStartsWithVerb` uses `rune(rest[0])` (byte cast) instead of `utf8.DecodeRuneInString`, incorrect for non-ASCII input.
- Double-finding (empty-description + ambiguous) fires for summaries < 10 chars; the FR-001 AC language implies a single finding.
- `Finding.Type` field comment lists only legacy types; five new type identifiers are absent.

### document-structural-checks
- Hard-gate structural failures reuse generic recovery steps; FR-009 requires structurally-specific remediation steps in error messages.
- `CheckAcceptanceCriteria` heading-only fallback path does not check `Section.WordCount > 0`; a section titled "Acceptance Criteria" with no content incorrectly passes.
- Promotion state load failure silently skips all structural checks with no observable signal.
- `_ = ps.Save()` discards save errors; persistent state loss is silent.

### action-pattern-logging
- `isErrorResult` byte-scans for the literal token `"error"` anywhere in the response text, causing false positives when `"error"` appears as a JSON string value.
- `MetricsResult` fields `StructuralCheckRate` and `ToolSubsetCompliance` carry `omitempty` and are never populated; the spec requires "no data" reporting, not omission.
- `os.Exit(1)` called directly inside `runMetrics()`, bypassing deferred cleanup.

### binding-registry-gate-integration
- After a failed `LoadBindingFile`, `c.cached` and `c.cachedMtime` are not cleared; a subsequent corrupt file with the same mtime as the prior valid file returns stale cached data instead of nil.
- `NewRegistryCache` uses lazy initialisation; a malformed binding file is not surfaced at server startup as specified.
- `RegistryCache.Get()` does not call `ValidateBinding`; an invalid `override_policy` value passes structural parse but fails silently at policy resolution time.
- `TestIntegration_Extensibility_CustomEvaluator` permanently replaces the global `"documents"` evaluator without restoring it, risking test pollution.
- Advance mode + checkpoint policy integration path (AC-6) has zero test coverage.
- `status` dashboard reports 0 tasks for FEAT-01KN5-8J27H83N despite `entity(action: "list", parent: "FEAT-01KN58J27H83N", type: "task")` returning 14. The status synthesis tool's child-task discovery does not match the entity list filter, causing the dashboard to show "No tasks exist" for features that have tasks. This is a pre-existing bug in the status tool, not introduced by this feature, but was discovered during implementation.

---

## Reviewer Unit Breakdown

| Sub-agent | Feature | Spec doc | Tasks reviewed | Files reviewed |
|-----------|---------|----------|----------------|----------------|
| Sub-agent 1 | mandatory-stage-gates | `work/spec/3.0-mandatory-stage-gates.md` | 6 | 20 |
| Sub-agent 2 | tool-description-audit | `work/spec/3.0-tool-description-audit.md` | 8 | 20+ |
| Sub-agent 3 | stage-aware-context-assembly | `work/spec/3.0-stage-aware-context-assembly.md` | 7 | 15 |
| Sub-agent 4 | review-rework-loop | `work/spec/3.0-review-rework-loop.md` | 5 | 15 |
| Sub-agent 5 | decomposition-quality-validation | `work/spec/3.0-decomposition-quality-validation.md` | 5 | 5 |
| Sub-agent 6 | document-structural-checks | `work/spec/3.0-document-structural-checks.md` | 7 | 17 |
| Sub-agent 7 | action-pattern-logging | `work/spec/3.0-action-pattern-logging.md` | 7 | 16 |
| Sub-agent 8 | binding-registry-gate-integration | `work/spec/3.0-binding-registry-gate-integration.md` | 14 | 14 |

---

## Remediation Priorities

### Must fix before any feature transitions to `done`

These blocking findings should be addressed roughly in this order given their dependencies:

**Tier 1 — Safety and correctness (fix first)**
1. B-01 / B-02: Advance path override_reason bypass (mandatory-stage-gates)
2. B-22: Stage logged post-transition in action log hook (action-pattern-logging)
3. B-16 / B-17: Acceptance criteria check at wrong gate; missing spec check at dev-planning→developing (document-structural-checks)
4. B-18: Cross-reference check blind to parent-plan designs (document-structural-checks)

**Tier 2 — Missing MCP surface**
5. B-19: RecordFalsePositive unreachable from MCP (document-structural-checks)
6. B-24: KnownFields(true) kills extensibility path in binding registry (binding-registry-gate-integration)
7. B-07 / B-08: Stage content flags not actuated in assembleContext (stage-aware-context-assembly)
8. B-21: Nil StageFeatureLookup in kbz metrics CLI (action-pattern-logging)

**Tier 3 — Test coverage**
9. B-11 through B-15: All test coverage for review-rework-loop feature
10. B-09 / B-10: Handler-level tests for stage-aware-context-assembly FR-001/FR-002
11. B-20: Quality evaluation approval gate test (document-structural-checks)
12. B-23: Scenario required-field validation (action-pattern-logging)
13. B-03 / B-04: decompose error template + token verification (tool-description-audit)
14. B-05 / B-06: Agent test session records + missing tool scenarios (tool-description-audit)

### FEAT-01KN58J26CH63 (decomposition-quality-validation)

This feature may be transitioned to `done` immediately. The 8 non-blocking findings
are follow-up improvements, not blockers.