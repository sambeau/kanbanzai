# Dev-Plan: Fast-Track Architecture

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | AI architect                   |

## Overview

This plan implements the fast-track architecture specification defined in
`work/P43-fast-track-architecture/P43-spec-fast-track-architecture.md` (DOC-`FEAT-01KQSP41PE6JP/spec-p43-spec-fast-track-architecture`). It covers all functional requirements (REQ-SPEC-001 through REQ-SESS-004) and non-functional requirements (REQ-NF-001 through REQ-NF-003).

It does **not** cover: the P44 model routing integration (separate specification), changes to the design stage gate, or replacement of the specialist review panel. The `dispatch_validator` abstraction is implemented for forward compatibility with P44, but the P44 model routing integration itself is out of scope.

## Task Breakdown

### Task 1: Define fast-track config schema and loading (TASK-01KQSP86204H4)

- **Description:** Define the `fast_track` configuration block schema and implement loading in `internal/config/`. The config block includes: `enabled` (bool), `default_tier` (one of `retro_fix`/`bug_fix`/`feature`/`critical`), and a `tiers` map with per-tier automation matrix (`design`/`spec`/`dev-plan`/`review` gate modes: `auto`/`human`/`conditional`) and `max_auto_cycles` (int). Implement Go structs, YAML deserialisation with defaults, validation (valid tier names, valid gate modes), and unit tests. Wire into the config loading path so `fast_track` config is available to downstream packages.
- **Deliverable:** Modified `internal/config/` with fast-track config types, loader, and tests.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-TIER-001, REQ-TIER-002, REQ-TIER-003, REQ-TIER-005, REQ-TIER-006.

### Task 2: Create spec-validator role (TASK-01KQSP8X96XG5)

- **Description:** Create `.kbz/roles/spec-validator.yaml` inheriting from `base`. Identity text: "Senior requirements quality auditor. Verify that specifications are complete, testable, and traceable to their parent design. Do not evaluate whether requirements are *correct* — only whether they are *well-formed* and *complete*." Define anti-patterns: Assumed Traceability (assuming a requirement is traceable without verifying), Content Judgment (evaluating requirement correctness instead of well-formedness), Hallucinated Completeness (declaring a spec complete without evidence). Define tool constraints: `read_file`, `doc`, `doc_intel` for document reading and structural analysis.
- **Deliverable:** `.kbz/roles/spec-validator.yaml`.
- **Depends on:** Task 1 (config must exist before roles reference tier concepts).
- **Effort:** Small.
- **Spec requirement:** REQ-SPEC-001, REQ-SPEC-004.

### Task 3: Create plan-validator role (TASK-01KQSP8X96FBT)

- **Description:** Create `.kbz/roles/plan-validator.yaml` inheriting from `base`. Identity text: "Senior implementation plan auditor. Verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. Do not evaluate whether task ordering is *optimal* — only whether it is *valid* and *complete*." Define anti-patterns: Phantom Traceability (claiming traceability without spec references), Architectural Second-Guessing (re-litigating architecture decisions), Unverified File References (accepting file paths without validating they exist or are noted as new). Define tool constraints.
- **Deliverable:** `.kbz/roles/plan-validator.yaml`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-PLAN-001, REQ-PLAN-004.

### Task 4: Create review-gate-validator role (TASK-01KQSP8X9B2WE)

- **Description:** Create `.kbz/roles/review-gate-validator.yaml` inheriting from `reviewer` (not `base` — this validator audits reviews, which IS a kind of review). Identity text: "Senior review quality auditor. Verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code — audit the review process itself." Define anti-patterns: Re-Reviewing (evaluating code instead of reviewing the review process), Rubber-Stamp Acceptance (accepting superficial review output as sufficient). Define tool constraints.
- **Deliverable:** `.kbz/roles/review-gate-validator.yaml`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirement:** REQ-RVW-001, REQ-RVW-004.

### Task 5: Create validator rubrics (TASK-01KQSP98PT9H9)

- **Description:** Create rubric files in `work/P43-fast-track-architecture/validator-rubrics/`: `spec-validator-rubrics.md`, `plan-validator-rubrics.md`, `review-gate-validator-rubrics.md`. Each file defines pass/fail/borderline-escalate patterns for LLM-classification checks: S5 (testable AC), S7 (implementation instruction detection), D7 (monolithic task detection), D10 (risk assessment non-empty), D13 (actionable task description), R2 (rubber-stamp detection), R3 (severity inflation). Each check needs: (1) pass definition with 2-3 positive examples, (2) fail definition with 2-3 negative examples, (3) borderline-escalate pattern for ambiguous cases. Test rubrics against 15-20 real Kanbanzai documents from the `work/` directory.
- **Deliverable:** Three rubric files in `work/P43-fast-track-architecture/validator-rubrics/`.
- **Depends on:** Task 2, Task 3, Task 4 (rubrics reference check IDs defined in the roles).
- **Effort:** Medium.
- **Spec requirement:** REQ-RUB-001, REQ-RUB-002, REQ-RUB-003.

### Task 6: Create spec-validator skill (TASK-01KQSPA1EJQP4)

- **Description:** Create `.kbz/skills/validate-spec/SKILL.md` with the spec-validator procedure. Define all 10 validation checks (S1-S10): S1,S2,S3,S4,S5,S10 blocking; S6,S7,S8,S9 non-blocking. Procedure: read spec document via `doc(action: content)`, read parent design document, execute each check using rubrics for LLM-classification checks, produce two outputs: (a) summary for orchestrator — verdict (pass/pass_with_notes/fail), blocking/non-blocking finding counts, evidence score, reference to full report document ID; (b) full report written to document store via `doc(action: register, type: report)`. Must run in fresh session — receives only the document, parent document, and checklist with rubrics. Must complete within 5 tool calls for typical specs (under 200 lines).
- **Deliverable:** `.kbz/skills/validate-spec/SKILL.md`.
- **Depends on:** Task 2 (role defines the checks), Task 5 (rubrics needed for LLM-classification checks).
- **Effort:** Medium.
- **Spec requirement:** REQ-SPEC-002, REQ-SPEC-003, REQ-SESS-001, REQ-SESS-002, REQ-SESS-003, REQ-NF-001.

### Task 7: Create plan-validator skill (TASK-01KQSPA1EGS2G)

- **Description:** Create `.kbz/skills/validate-plan/SKILL.md` with the plan-validator procedure. Define all 13 validation checks (D1-D13): D1,D2,D3,D4,D5,D6,D9,D13 blocking; D7,D8,D10,D11,D12 non-blocking. Procedure: read dev-plan doc, read parent spec doc, execute each check. D6 requires topological sort verification of the dependency graph. D13 requires word-count check on task descriptions (>=50 words) and semantic check for inputs/outputs/done criteria using rubrics. D3,D4 require cross-referencing task spec-requirement fields against the spec's REQ-IDs. Produce summary and full report as with spec-validator. Fresh session discipline applies.
- **Deliverable:** `.kbz/skills/validate-plan/SKILL.md`.
- **Depends on:** Task 3 (role defines the checks), Task 5 (rubrics needed for D7,D10,D13).
- **Effort:** Medium.
- **Spec requirement:** REQ-PLAN-002, REQ-PLAN-003, REQ-SESS-001, REQ-SESS-002, REQ-SESS-003.

### Task 8: Create review-gate-validator skill (TASK-01KQSPA1EH8E9)

- **Description:** Create `.kbz/skills/validate-review/SKILL.md` with the review-gate-validator procedure. Define all 8 validation checks (R1-R8): R1,R2,R4,R5,R7 blocking; R3,R6,R8 non-blocking. Procedure: read review documents, execute each check. R2 (rubber-stamp detection) uses rubrics to distinguish genuine reviews from "LGTM" responses. R4 verifies every blocking finding cites a spec requirement (cross-reference with spec document). R5 checks aggregate verdict consistency against per-dimension outcomes. Produce summary and full report. Fresh session discipline applies.
- **Deliverable:** `.kbz/skills/validate-review/SKILL.md`.
- **Depends on:** Task 4 (role defines the checks), Task 5 (rubrics needed for R2,R3).
- **Effort:** Medium.
- **Spec requirement:** REQ-RVW-002, REQ-RVW-003, REQ-SESS-001, REQ-SESS-002, REQ-SESS-003.

### Task 9: Implement tier inference on entity creation (TASK-01KQSPABM1FWR)

- **Description:** Implement tier inference logic in the entity creation path. Rules: (1) if feature has a retro signal -> `retro_fix`, (2) if feature is a bug entity type -> `bug_fix`, (3) if feature has `critical` or `security` tag -> `critical`, (4) otherwise -> config `default_tier`. An explicitly set `tier` field on the entity overrides all inference. Add a `Tier` field to the feature entity model in `internal/model/`. Wire inference into entity creation in `internal/service/`. Add unit tests for each inference rule, the override precedence, and the default fallback. Verify tier is stored and retrievable.
- **Deliverable:** Modified `internal/model/`, `internal/service/`, and tests.
- **Depends on:** Task 1 (config must be loadable for `default_tier`).
- **Effort:** Small.
- **Spec requirement:** REQ-INFER-001, REQ-INFER-002, REQ-INFER-003.

### Task 10: Implement dispatch_validator abstraction (TASK-01KQSPANMG2RD)

- **Description:** Create the `dispatch_validator(role, skill, context)` abstraction in `internal/validate/` (new package, or extend existing). Define a Go interface: `ValidatorDispatcher` with method `Dispatch(role string, skill string, context ValidatorContext) (ValidatorSummary, error)`. Implement the concrete dispatcher using `spawn_agent` with fresh context. The context struct carries: document under validation (path + content hash), parent document (path + content hash), checklist with rubrics (loaded from the rubric files). Returns `ValidatorSummary` (verdict, blocking/non-blocking counts, evidence score, report document ID). The dispatcher writes the full report to the document store via `doc(action: register, type: report)`. Design the interface so future P44 model routing can replace `spawn_agent` by injecting a different `ValidatorDispatcher` implementation. The abstraction must NOT hardcode `spawn_agent` in the interface definition.
- **Deliverable:** New or modified `internal/validate/` package with interface, implementation, and tests.
- **Depends on:** Task 1 (config needed for context assembly).
- **Effort:** Medium.
- **Spec requirement:** REQ-SESS-004, REQ-PIPE-002.

### Task 11: Implement stage transition validator hooks (TASK-01KQSPB0YKQZB)

- **Description:** Add `transition_validator` hooks to `.kbz/stage-bindings.yaml` and implement enforcement in entity transition logic. Define hooks for: `specifying -> dev-planning` (spec-validator), `dev-planning -> developing` (plan-validator), `reviewing -> done` (review-gate-validator). When transition is attempted: (1) check the feature's risk tier to determine if the gate should be auto-validated or human-gated, (2) if auto-validated and no valid cached verdict exists, dispatch the validator, (3) if blocking checks fail, reject the transition with error containing: failing check IDs, blocking/non-blocking classification per check, and validator report document ID, (4) if non-blocking only, allow transition and attach findings to the document record. Human override via `entity(action: transition, override: true, override_reason: "...")` always available; override usage tracked as metric. Non-blocking findings visible in `status` dashboards. Implement in `internal/validate/` and wire into `internal/service/` transition logic.
- **Deliverable:** Modified `.kbz/stage-bindings.yaml`, `internal/validate/`, `internal/service/`, and tests.
- **Depends on:** Task 1 (config for tier lookup), Task 9 (tier field on feature entity).
- **Effort:** Large.
- **Spec requirement:** REQ-TRANS-001 through REQ-TRANS-007.

### Task 12: Implement validation pipeline with auto-approval (TASK-01KQSPBDH66PJ)

- **Description:** Implement the validation pipeline triggered on document registration. When a spec or dev-plan is registered via `doc(action: register)`: (1) determine the owning feature and its risk tier, (2) check whether automated validation is allowed for this stage per the tier's automation matrix, (3) if allowed, dispatch the appropriate validator via `dispatch_validator`, (4) on `pass`: auto-approve the document via `doc(action: approve)`, (5) on `pass_with_notes`: approve + attach non-blocking findings to the document record, surface in `status` output, (6) on `fail`: do NOT approve, surface findings to the human (via checkpoint or error message), feature stays in current stage. Track auto-validation cycle count per feature. Design stage must never be auto-validated regardless of tier. The `retro_fix` review gate must be conditional: implementation changes (files outside `work/`, `docs/`, `refs/`) trigger specialist review + validator audit; documentation-only changes skip the review gate entirely. Implement in the `internal/mcp/` doc registration handler.
- **Deliverable:** Modified `internal/mcp/` doc registration handler, pipeline logic, and tests.
- **Depends on:** Task 10 (dispatch_validator), Task 11 (transition hooks define which validator per stage), Task 6 (spec-validator skill), Task 7 (plan-validator skill), Task 8 (review-gate-validator skill).
- **Effort:** Large.
- **Spec requirement:** REQ-PIPE-001 through REQ-PIPE-007, REQ-TIER-004.

### Task 13: Implement cycle tracking and human escalation (TASK-01KQSPBPCV01S)

- **Description:** Implement `max_auto_cycles` enforcement. Track per-feature cycle count (number of fix-validate iterations). When count reaches the tier's cap (`retro_fix`=3, `bug_fix`=2, `feature`=2, `critical`=0), stop automated validation and escalate to human via `checkpoint(action: create)`. The system must not attempt further automated validation until the human responds. Implement accumulation tracking for non-blocking findings across multiple features — when patterns emerge (e.g., same non-blocking check fires on 3+ features), trigger a quality review signal visible in the project health dashboard via `health` check integration. Wire the cycle count into the validation pipeline (Task 12). Cycle count resets on successful pass.
- **Deliverable:** Modified `internal/validate/`, `internal/health/`, and tests.
- **Depends on:** Task 12 (pipeline must be built first; cycle tracking integrates into it).
- **Effort:** Small.
- **Spec requirement:** REQ-TIER-003, REQ-PIPE-006, REQ-TRANS-006.

### Task 14: Integration tests for fast-track system (TASK-01KQSPC240XPZ)

- **Description:** Write comprehensive integration tests covering the full fast-track system: (1) spec-validator blocks transition on missing Verification Plan (AC-TRANS-001), (2) plan-validator blocks transition on cyclic dependency graph (AC-TRANS-003), (3) review-gate-validator blocks on rubber-stamp review (AC-RVW-002), (4) non-blocking findings don't prevent transition but attach to doc record (AC-TRANS-004), (5) human override bypasses validator block and is recorded (AC-TRANS-005), (6) each risk tier automation matrix is respected (AC-TIER-001 through AC-TIER-004), (7) tier inference rules produce correct tiers (AC-INFER-001 through AC-INFER-003), (8) validation pipeline auto-approves on pass, escalates on fail (AC-PIPE-001 through AC-PIPE-005), (9) cycle cap triggers human escalation (AC-PIPE-004), (10) design gate never auto-validates (AC-PIPE-005), (11) validator completes within 5 tool calls (AC-NF-001), (12) transition check adds less than 2s latency when no validator is required (AC-NF-002), (13) error messages contain check IDs, severity, and report reference (AC-NF-003). Use the test fixture infrastructure to create test entities, documents, and simulate validator runs.
- **Deliverable:** New integration test files in `internal/validate/` or `internal/mcp/`.
- **Depends on:** Task 12 (pipeline and transition hooks must be fully functional).
- **Effort:** Large.
- **Spec requirement:** All acceptance criteria (AC-SPEC-001 through AC-NF-003).

## Dependency Graph

```
Task 1  (fast-track-config)              -- no dependencies
Task 2  (spec-validator-role)            -> depends on Task 1
Task 3  (plan-validator-role)            -> depends on Task 1
Task 4  (review-gate-validator-role)     -> depends on Task 1
Task 5  (validator-rubrics)              -> depends on Task 2, Task 3, Task 4
Task 6  (validate-spec-skill)            -> depends on Task 2, Task 5
Task 7  (validate-plan-skill)            -> depends on Task 3, Task 5
Task 8  (validate-review-skill)          -> depends on Task 4, Task 5
Task 9  (tier-inference)                 -> depends on Task 1
Task 10 (dispatch-validator)             -> depends on Task 1
Task 11 (transition-validator-hooks)     -> depends on Task 1, Task 9
Task 12 (validation-pipeline)            -> depends on Task 10, Task 11, Task 6, Task 7, Task 8
Task 13 (cycle-tracking-escalation)      -> depends on Task 12
Task 14 (integration-tests)              -> depends on Task 12
```

**Parallel groups:**
- Wave 1: [Task 2, Task 3, Task 4] (all role definitions — independent of each other, all depend on Task 1)
- Wave 1: [Task 9, Task 10] (tier inference and dispatch validator — independent of each other, both depend on Task 1)
- Wave 2: [Task 6, Task 7, Task 8] (all skills — independent of each other, depend on roles and rubrics)
- Wave 3: Task 11 (can start after Task 1 + Task 9)
- Wave 4: Task 12 (integrates everything)
- Wave 5: [Task 13, Task 14] (independent of each other, both integrate with Task 12)

**Critical path:** Task 1 -> Task 2 -> Task 5 -> Task 6 -> Task 12 -> Task 14
(Or: Task 1 -> Task 2 -> Task 5 -> Task 7 -> Task 12 -> Task 14 — similar duration)

## Risk Assessment

### Risk: Validator inconsistency across fresh sessions
- **Probability:** Medium.
- **Impact:** High — inconsistent validation verdicts erode trust in automation.
- **Mitigation:** Rubrics with concrete pass/fail/borderline examples reduce variance. Test rubrics against 15-20 real documents before pipeline activation (Task 5). Track verdict patterns across sessions; if inconsistency detected, flag in health dashboard.
- **Affected tasks:** Task 5, Task 6, Task 7, Task 8, Task 12.

### Risk: Override abuse diluting validator effectiveness
- **Probability:** Low.
- **Impact:** Medium — if humans override validators routinely, the system reverts to manual gating.
- **Mitigation:** Track override usage as a metric per check. If a specific check is consistently overridden, surface in health dashboard and flag for rubric refinement. Override requires explicit reason.
- **Affected tasks:** Task 11.

### Risk: Transition hook latency blocking the critical path
- **Probability:** Low.
- **Impact:** Medium — if validator dispatch is slow, transition calls become sluggish.
- **Mitigation:** REQ-NF-002 requires less than 2s latency when no validator runs (tier check is a config lookup). When a validator is required, the check is async-eligible — transition can proceed with a cached verdict. Performance tests in Task 14 verify.
- **Affected tasks:** Task 11, Task 12, Task 14.

### Risk: dispatch_validator abstraction too thin for P44 integration
- **Probability:** Low.
- **Impact:** High — if the abstraction doesn't cleanly separate dispatch mechanism from validator logic, P44 integration requires validator rewrites.
- **Mitigation:** Define a Go interface (`ValidatorDispatcher`) with a single `Dispatch` method. The concrete implementation lives behind the interface. Validator code calls the interface, not `spawn_agent` directly. Review the interface boundary in Task 10 before implementation.
- **Affected tasks:** Task 10, Task 12.

### Risk: retro_fix documentation-only change detection false negatives
- **Probability:** Medium.
- **Impact:** Medium — a retro_fix with unreviewed code changes would skip the review gate.
- **Mitigation:** Change scope detection must be explicit: check all modified files against the `work/`, `docs/`, `refs/` prefixes. Any file outside these prefixes triggers the specialist review path. Integration test (Task 14) covers this edge case.
- **Affected tasks:** Task 12, Task 14.

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|---------------|
| AC-SPEC-001: spec-validator returns structured verdict | Unit test | Task 6 |
| AC-SPEC-002: missing Verification Plan -> S1 blocking fail | Unit test | Task 6 |
| AC-SPEC-003: only non-blocking failures -> pass_with_notes | Unit test | Task 6 |
| AC-SPEC-004: untestable AC -> S5 blocking fail | Unit test | Task 6 |
| AC-PLAN-001: plan-validator returns structured verdict | Unit test | Task 7 |
| AC-PLAN-002: cyclic deps -> D6 blocking fail | Unit test | Task 7 |
| AC-PLAN-003: vague task description -> D13 blocking fail | Unit test | Task 7 |
| AC-PLAN-004: only non-blocking failures -> pass_with_notes | Unit test | Task 7 |
| AC-RVW-001: review-gate-validator returns structured verdict | Unit test | Task 8 |
| AC-RVW-002: rubber-stamp review -> R2 blocking fail | Unit test | Task 8 |
| AC-RVW-003: unanchored blocking finding -> R4 blocking fail | Unit test | Task 8 |
| AC-RUB-001: rubric files exist with definitions | Inspection | Task 5 |
| AC-RUB-002: rubrics tested against Kanbanzai docs | Inspection | Task 5 |
| AC-TRANS-001: spec failure blocks transition to dev-planning | Integration test | Task 14 |
| AC-TRANS-002: clean spec allows transition | Integration test | Task 14 |
| AC-TRANS-003: dev-plan failure blocks transition to developing | Integration test | Task 14 |
| AC-TRANS-004: non-blocking findings attach to doc record | Integration test | Task 14 |
| AC-TRANS-005: override bypasses validator block | Integration test | Task 14 |
| AC-TIER-001: critical tier requires human for spec | Integration test | Task 14 |
| AC-TIER-002: retro_fix docs-only skips review gate | Integration test | Task 14 |
| AC-TIER-003: retro_fix with code changes runs review panel | Integration test | Task 14 |
| AC-TIER-004: default_tier applied on feature creation | Integration test | Task 14 |
| AC-INFER-001: bug entity -> bug_fix tier | Unit test | Task 9 |
| AC-INFER-002: security tag -> critical tier | Unit test | Task 9 |
| AC-INFER-003: explicit tier overrides inference | Unit test | Task 9 |
| AC-PIPE-001: spec registration triggers auto-validation | Integration test | Task 14 |
| AC-PIPE-002: validator pass -> auto-approve | Integration test | Task 14 |
| AC-PIPE-003: validator fail -> not approved, findings surfaced | Integration test | Task 14 |
| AC-PIPE-004: cycle cap -> escalate to human | Integration test | Task 14 |
| AC-PIPE-005: design registration -> no validator dispatched | Integration test | Task 14 |
| AC-SESS-001: validator receives only doc + parent + checklist | Inspection | Task 6, Task 7, Task 8 |
| AC-SESS-002: orchestrator receives summary only | Integration test | Task 14 |
| AC-SESS-003: full report retrievable via doc content | Integration test | Task 14 |
| AC-SESS-004: dispatch_validator is abstracted behind interface | Inspection | Task 10 |
| AC-NF-001: spec-validator completes within 5 tool calls | Performance test | Task 14 |
| AC-NF-002: transition check adds <2s when no validator runs | Performance test | Task 14 |
| AC-NF-003: error message contains check IDs, severity, report ref | Integration test | Task 14 |

## Interface Contracts

### ValidatorDispatcher interface

```go
// ValidatorContext carries the document and rubric context for a validation run.
type ValidatorContext struct {
    DocumentPath    string // path to document under validation
    DocumentType    string // "specification", "dev-plan", or "report"
    ParentDocPath   string // path to parent document (design for spec, spec for dev-plan)
    RubricPath      string // path to the rubric file for this validator
    FeatureID       string // owning feature ID
}

// ValidatorSummary is the lightweight result returned to the orchestrator.
type ValidatorSummary struct {
    Verdict             string // "pass", "pass_with_notes", or "fail"
    BlockingCount       int
    NonBlockingCount    int
    EvidenceScore       float64 // 0.0-1.0
    ReportDocID         string  // document ID of the full report
}

// ValidatorDispatcher dispatches a validator in a fresh session.
type ValidatorDispatcher interface {
    Dispatch(ctx context.Context, role string, skill string, vctx ValidatorContext) (ValidatorSummary, error)
}
```

**Contract between orchestrator and validator:** The dispatcher passes `ValidatorContext` to the spawned agent. The agent reads the document, parent document, and rubric; runs validation checks; produces a `ValidatorSummary` (returned to the orchestrator) and a full report (written to the document store). The orchestrator never needs to read the full report — it uses the summary for flow control. The report is retrievable via `doc(action: content, id: summary.ReportDocID)` for human audit.

**Contract between dispatcher and P44:** The `ValidatorDispatcher` interface is the abstraction boundary. P44 model routing provides an alternative `ValidatorDispatcher` implementation that routes through the model routing dispatch loop. Validator code (`validate-spec`, `validate-plan`, `validate-review` skills) calls `Dispatch()` on the interface — they never reference `spawn_agent` directly. Switching to P44 is a configuration change (inject different `ValidatorDispatcher` implementation), not a code change in validators.

### FastTrackConfig shape

```go
type FastTrackConfig struct {
    Enabled     bool              `yaml:"enabled"`
    DefaultTier string            `yaml:"default_tier"` // "feature"
    Tiers       map[string]TierConfig `yaml:"tiers"`
}

type TierConfig struct {
    Design   string `yaml:"design"`    // "auto", "human", or "conditional"
    Spec     string `yaml:"spec"`      // "auto" or "human"
    DevPlan  string `yaml:"dev-plan"`  // "auto" or "human"
    Review   string `yaml:"review"`    // "auto", "human", or "conditional"
    MaxCycles int   `yaml:"max_cycles"`
}
```

**Contract between config and transition hooks:** `entity(action: transition)` reads the feature's tier from the entity model, looks up `TierConfig` for that tier, and checks the gate mode for the current stage transition. If mode is `auto`, the validator is dispatched. If `human`, the transition proceeds without validation (current behavior). If `conditional` (review gate for `retro_fix`), change scope detection runs to decide.

### Entity model extension

```go
// Feature entity gains a Tier field
type Feature struct {
    // ... existing fields ...
    Tier string `yaml:"tier,omitempty"` // "retro_fix", "bug_fix", "feature", "critical"
}
```

**Contract between entity creation and tier inference:** `entity(action: create, type: feature)` calls tier inference before persisting. If `tier` is explicitly set, it is used as-is. Otherwise, inference rules run: retro signal -> `retro_fix`, bug type -> `bug_fix`, `critical`/`security` tag -> `critical`, else -> `FastTrackConfig.DefaultTier`. The inferred or explicit tier is stored on the entity and never re-inferred.

### Stage binding extension

```yaml
stage_bindings:
  # ... existing bindings ...
  specifying:
    transition_validator:
      role: spec-validator
      skill: validate-spec
      blocking: true
  dev-planning:
    transition_validator:
      role: plan-validator
      skill: validate-plan
      blocking: true
  reviewing:
    transition_validator:
      role: review-gate-validator
      skill: validate-review
      blocking: true
```

**Contract between stage bindings and transition logic:** The `transition_validator` block defines which validator to run before a feature can transition OUT of this stage. `blocking: true` means validator blocking failures prevent transition. The transition logic reads this from `.kbz/stage-bindings.yaml`, not from hardcoded constants.

## Traceability Matrix

| Spec Requirement | Task(s) | Acceptance Criteria |
|-----------------|---------|-------------------|
| REQ-SPEC-001, REQ-SPEC-004 | Task 2 (spec-validator role) | AC-SPEC-001 |
| REQ-SPEC-002, REQ-SPEC-003 | Task 6 (validate-spec skill) | AC-SPEC-002, AC-SPEC-003, AC-SPEC-004 |
| REQ-PLAN-001, REQ-PLAN-004 | Task 3 (plan-validator role) | AC-PLAN-001 |
| REQ-PLAN-002, REQ-PLAN-003 | Task 7 (validate-plan skill) | AC-PLAN-002, AC-PLAN-003, AC-PLAN-004 |
| REQ-RVW-001, REQ-RVW-004 | Task 4 (review-gate-validator role) | AC-RVW-001 |
| REQ-RVW-002, REQ-RVW-003 | Task 8 (validate-review skill) | AC-RVW-002, AC-RVW-003 |
| REQ-RUB-001, REQ-RUB-002, REQ-RUB-003 | Task 5 (validator rubrics) | AC-RUB-001, AC-RUB-002 |
| REQ-TRANS-001 through REQ-TRANS-007 | Task 11 (transition hooks), Task 13 (cycle tracking) | AC-TRANS-001 through AC-TRANS-005 |
| REQ-TIER-001 through REQ-TIER-006 | Task 1 (fast-track config), Task 12 (pipeline) | AC-TIER-001 through AC-TIER-004 |
| REQ-INFER-001, REQ-INFER-002, REQ-INFER-003 | Task 9 (tier inference) | AC-INFER-001, AC-INFER-002, AC-INFER-003 |
| REQ-PIPE-001 through REQ-PIPE-007 | Task 12 (validation pipeline) | AC-PIPE-001 through AC-PIPE-005 |
| REQ-SESS-001 through REQ-SESS-004 | Task 6, Task 7, Task 8, Task 10 (dispatch) | AC-SESS-001 through AC-SESS-004 |
| REQ-NF-001, REQ-NF-002, REQ-NF-003 | Task 14 (integration tests) | AC-NF-001, AC-NF-002, AC-NF-003 |
