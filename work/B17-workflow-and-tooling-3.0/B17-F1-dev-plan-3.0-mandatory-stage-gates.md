# Implementation Plan: Mandatory Stage Gate Enforcement

**Feature:** FEAT-01KN5-8J24S2XW (mandatory-stage-gates)
**Specification:** `work/spec/3.0-mandatory-stage-gates.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §3, §6.1–6.3

---

## Scope Boundaries

Carried forward from the specification:

- **In scope:** Mandatory gate evaluation on all Phase 2 feature transitions, the complete 9-transition gate prerequisite table, override mechanism with logging and health flagging, actionable error messages for all gate failures.
- **Out of scope:** Binding registry gate integration, `checkpoint` override policy, review cycle counters, document structural checks, plan lifecycle gates, Phase 1 legacy feature transitions.

---

## Task Breakdown

### Task 1: Extend Feature Model with Override Records

**Objective:** Add an `Overrides` field to the `model.Feature` struct so that override transitions can be persisted on the entity. Define the `OverrideRecord` type with from-status, to-status, reason, and timestamp fields.

**Specification references:** FR-014, FR-016

**Input context:**
- `internal/model/entities.go` — current `Feature` struct (L290–319), `FeatureStatus` constants
- `internal/service/entities.go` — `featureFields` function (L1025–1069) that serialises features to field maps
- `refs/go-style.md` — YAML serialisation rules (deterministic field order, block style)

**Output artifacts:**
- Modified `internal/model/entities.go`:
  - New `OverrideRecord` struct with fields: `FromStatus string`, `ToStatus string`, `Reason string`, `Timestamp time.Time`
  - New `Overrides []OverrideRecord` field on `Feature` struct with `yaml:"overrides,omitempty"`
- Modified `internal/service/entities.go`:
  - Update `featureFields` to include `overrides` in the serialised field map when non-empty
  - Update `featureFromState` in `internal/mcp/entity_tool.go` to populate `Overrides` from state (or defer to Task 5 if cleaner)
- New test cases in `internal/model/entities_test.go` verifying round-trip serialisation of override records

**Dependencies:** None — this is a foundational data model change.

**Interface contract:** The `OverrideRecord` struct is the shared type used by Task 3 (gate enforcement writes overrides) and Task 6 (health check reads overrides):

```
type OverrideRecord struct {
    FromStatus string    `yaml:"from_status"`
    ToStatus   string    `yaml:"to_status"`
    Reason     string    `yaml:"reason"`
    Timestamp  time.Time `yaml:"timestamp"`
}
```

The `Feature.Overrides` field is `[]OverrideRecord` and is append-only (multiple overrides accumulate).

---

### Task 2: Expand Gate Prerequisites in `prereq.go`

**Objective:** Extend `CheckFeatureGate` to cover all nine gated transitions in the specification's prerequisite table. Currently it handles `designing`, `specifying`, `dev-planning`, and partially `developing`. Add gates for: `developing → reviewing` (all tasks terminal), `reviewing → done` (review report exists), `needs-rework → developing` (rework task exists), and `needs-rework → reviewing` (all tasks terminal). Refactor the function signature so callers pass the transition (from-status, to-status) rather than just the target stage.

**Specification references:** FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010

**Input context:**
- `internal/service/prereq.go` — current `CheckFeatureGate`, `checkDocumentGate`, `checkDevelopingGate`
- `internal/service/prereq_test.go` — existing test patterns and `submitAndApproveDoc` helper
- `internal/validate/lifecycle.go` — `DependencyTerminalStates`, `IsTaskDependencySatisfied`, terminal state definitions
- `internal/service/entities.go` — `EntityService.List` for querying child tasks
- `internal/service/documents.go` — `DocumentService.ListDocuments` for querying documents
- `internal/model/entities.go` — `DocumentTypeReport`, `TaskStatus*` constants

**Output artifacts:**
- Modified `internal/service/prereq.go`:
  - New function `CheckTransitionGate(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult` that dispatches to the correct prerequisite check based on the (from, to) pair
  - Retain existing `checkDocumentGate` (used by designing→specifying, specifying→dev-planning, dev-planning→developing doc sub-check)
  - Refactor `checkDevelopingGate` to check for both approved dev-plan AND at least one child task (FR-006)
  - New `checkAllTasksTerminal(feature, entitySvc) GateResult` — used by developing→reviewing (FR-007) and needs-rework→reviewing (FR-010)
  - New `checkReviewReportExists(feature, docSvc) GateResult` — checks for a registered report document owned by the feature (FR-008)
  - New `checkReworkTaskExists(feature, entitySvc) GateResult` — checks for at least one non-terminal child task (FR-009)
  - Logic to return `GateResult{Satisfied: true}` for ungated transitions: proposed→designing, reviewing→needs-rework (FR-003)
  - Logic to return `GateResult{Satisfied: true}` for terminal-state transitions: any→superseded, any→cancelled (FR-002)
- Modified `internal/service/prereq_test.go`:
  - Table-driven tests for all nine gated transitions covering satisfied and unsatisfied cases
  - Tests for ungated transitions (proposed→designing, reviewing→needs-rework)
  - Tests for terminal-state transitions (→superseded, →cancelled)

**Dependencies:** None — this is pure service-layer logic with no model changes required.

**Interface contract with Task 5:** The primary entry point is:

```
func CheckTransitionGate(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult
```

Task 5 (MCP wiring) calls this function for every feature transition. The `GateResult` struct is unchanged:

```
type GateResult struct {
    Stage     string
    Satisfied bool
    Reason    string
}
```

---

### Task 3: Actionable Error Message Formatter

**Objective:** Create a gate failure error formatter that produces structured, actionable error messages following the design template. Each gate failure must identify what failed, why, and provide recovery steps as tool call examples. Also produce a structured response map with `"error"` and `"gate_failed"` fields.

**Specification references:** FR-018, FR-019, FR-020, FR-021, FR-022, FR-023, FR-024, FR-025, FR-026

**Input context:**
- `work/spec/3.0-mandatory-stage-gates.md` §Actionable error messages — exact recovery step patterns for each gate
- `internal/mcp/entity_tool.go` — `entityTransitionError` function (L706–730) for the existing error response pattern
- `internal/service/prereq.go` — `GateResult` struct returned by gate checks
- `internal/model/entities.go` — `Feature` struct (for extracting IDs and parent references)

**Output artifacts:**
- New file `internal/service/gate_errors.go`:
  - `GateFailureResponse` function that takes a feature ID, from-status, to-status, `GateResult`, and supplemental data (e.g., list of non-terminal task IDs+statuses), and returns a `map[string]any` containing:
    - `"error"` — the actionable message string
    - `"gate_failed"` — `map[string]any{"from_status": ..., "to_status": ...}`
  - Per-gate recovery step builders:
    - `designGateRecovery(featureID, parentID string) string`
    - `specGateRecovery(featureID, parentID string) string`
    - `devPlanGateRecovery(featureID, parentID string) string`
    - `taskCompletionGateRecovery(featureID string, nonTerminalTasks []TaskStatusPair) string`
    - `reviewReportGateRecovery(featureID string) string`
    - `reworkTaskGateRecovery(featureID string) string`
  - A `TaskStatusPair` type: `struct { ID string; Status string }` for passing non-terminal task info into error messages
- New file `internal/service/gate_errors_test.go`:
  - Tests that each gate's error message contains the feature ID, from-status, to-status, and "To resolve:" section
  - Tests that recovery steps contain the correct tool call patterns (e.g., `doc(action: "list", ...)`)
  - Tests for determinism: same inputs produce identical messages (NFR-004)

**Dependencies:** Task 2 must complete first so that `GateResult` semantics are finalised (specifically, the supplemental data for task-completeness failures).

**Interface contract with Task 5:** Task 5 calls `GateFailureResponse` and returns its result directly as the MCP response body. The signature:

```
func GateFailureResponse(featureID, from, to string, gate GateResult, nonTerminalTasks []TaskStatusPair) map[string]any
```

---

### Task 4: Update `AdvanceFeatureStatus` for Mandatory Gates

**Objective:** Modify `AdvanceFeatureStatus` to enforce gates on every step (not just intermediate non-target steps), support override+reason passthrough, and log each overridden gate as an `OverrideRecord` on the feature. Currently, advance only checks gates on non-target intermediate states and skips stop states entirely. The new behaviour: every step checks its gate via `CheckTransitionGate`, override bypasses all failing gates using the single provided reason, and each bypass is logged.

**Specification references:** FR-001, FR-016, NFR-005

**Input context:**
- `internal/service/advance.go` — current `AdvanceFeatureStatus` implementation
- `internal/service/advance_test.go` — existing test patterns
- `internal/service/prereq.go` — `CheckTransitionGate` (from Task 2)
- `internal/model/entities.go` — `OverrideRecord` (from Task 1)

**Output artifacts:**
- Modified `internal/service/advance.go`:
  - Updated `AdvanceFeatureStatus` signature to accept override parameters: `override bool, overrideReason string`
  - At each step: call `CheckTransitionGate(currentStatus, nextStatus, ...)`. If gate fails and override is true, append an `OverrideRecord` to the feature and proceed. If gate fails and override is false, stop and return a partial result with the gate failure.
  - The `AdvanceResult` struct gains a new field: `OverriddenGates []string` listing transitions that were overridden
  - Preserve existing stop-state behaviour: `reviewing` remains a mandatory halt point (NFR-005)
  - Persist override records on the feature after each overridden step (write updated feature state)
- Modified `internal/service/advance_test.go`:
  - Tests for gate enforcement at every step (not just intermediate)
  - Tests for override passthrough: all gates bypassed with the single reason
  - Tests that each bypassed gate produces a separate override record
  - Tests that stop-state behaviour is preserved

**Dependencies:** Task 1 (OverrideRecord model), Task 2 (CheckTransitionGate)

**Interface contract with Task 5:** The updated function signature:

```
func AdvanceFeatureStatus(
    feature *model.Feature,
    targetStatus string,
    entitySvc *EntityService,
    docSvc *DocumentService,
    override bool,
    overrideReason string,
) (AdvanceResult, error)
```

`AdvanceResult` adds:

```
OverriddenGates []string // e.g., ["specifying→dev-planning", "dev-planning→developing"]
```

---

### Task 5: Wire Gate Enforcement into `entityTransitionAction`

**Objective:** Modify the MCP entity transition handler to enforce gates on single-step feature transitions, accept `override` and `override_reason` parameters, persist override records, and return structured gate failure responses. Connect all the pieces from Tasks 1–4 into the MCP layer.

**Specification references:** FR-001, FR-002, FR-003, FR-011, FR-012, FR-013, FR-016, FR-026, NFR-002

**Input context:**
- `internal/mcp/entity_tool.go` — `entityTransitionAction` (L555–636), `entityAdvanceFeature` (L640–681), `entityTool` tool definition (L39–100), `featureFromState` (L684–694)
- `internal/service/prereq.go` — `CheckTransitionGate` (from Task 2)
- `internal/service/gate_errors.go` — `GateFailureResponse` (from Task 3)
- `internal/service/advance.go` — updated `AdvanceFeatureStatus` (from Task 4)
- `internal/model/entities.go` — `Feature`, `OverrideRecord` (from Task 1)
- `internal/service/entities.go` — `EntityService.UpdateStatus`, `EntityService.Get`

**Output artifacts:**
- Modified `internal/mcp/entity_tool.go`:
  - Add `override` (boolean) and `override_reason` (string) parameter definitions to the tool schema
  - In `entityTransitionAction`:
    - Parse `override` and `override_reason` from args
    - For feature entities on Phase 2 transitions only (NFR-002): call `CheckTransitionGate`. If gate fails and no override, return `GateFailureResponse`. If gate fails and override without reason, return validation error (FR-012). If gate fails and override with reason, persist override record and proceed.
    - For terminal-state transitions (superseded, cancelled): skip gate evaluation entirely (FR-002)
    - For non-feature entities and Phase 1 feature transitions: no change (NFR-002)
  - In `entityAdvanceFeature`:
    - Pass `override` and `override_reason` through to `AdvanceFeatureStatus`
    - Include `overridden_gates` in the advance response when gates were overridden
  - Update `featureFromState` to populate `Overrides` from the entity state map
  - Helper: `isPhase2Transition(from, to string) bool` — returns true if both statuses are Phase 2 lifecycle states (proposed, designing, specifying, dev-planning, developing, reviewing, needs-rework, done). Returns false for Phase 1 states (draft, in-review, approved, in-progress, review).
- New file `internal/mcp/entity_transition_gate_test.go`:
  - Integration tests exercising the full MCP handler for:
    - Single-step gated transition without prerequisite → gate failure response
    - Single-step gated transition with prerequisite → success
    - Single-step override with reason → success + override logged
    - Override without reason → validation error (FR-012)
    - Override reason ignored when override is false (FR-013)
    - Terminal-state transitions bypass gates (FR-002)
    - Ungated transitions succeed without checks (FR-003)
    - Phase 1 transitions are not gated (NFR-002)
    - Advance with override → all gates bypassed, each logged (FR-016)
    - Same gate failure for single-step and advance paths (FR-001 acceptance criteria)

**Dependencies:** Task 1 (model), Task 2 (gate checks), Task 3 (error formatting), Task 4 (advance updates)

**Interface contract:** This is the integration point. No downstream tasks depend on this task's internal structure.

---

### Task 6: Override Health Check Warnings

**Objective:** Add a health check category that flags features with gate overrides as warnings. Each override record produces a separate warning identifying the feature, the overridden transition, and the reason.

**Specification references:** FR-015

**Input context:**
- `internal/health/health.go` — `Issue`, `CategoryResult`, `HealthResult` types
- `internal/health/check.go` — `RunHealthCheck`, `CheckOptions` (features are passed as `[]map[string]any`)
- `internal/health/categories.go` — existing category check functions (pattern to follow)
- `internal/mcp/health_tool.go` — `Phase3HealthChecker`, `mergeHealthResult` — how health categories are merged into the MCP response
- `internal/model/entities.go` — `OverrideRecord` (from Task 1)

**Output artifacts:**
- New file `internal/health/gate_overrides.go`:
  - `CheckGateOverrides(features []map[string]any) CategoryResult` — iterates features, extracts `overrides` field, produces a warning `Issue` per override record with message format: `"feature {ID}: gate override on {from}→{to}: {reason}"`
- New file `internal/health/gate_overrides_test.go`:
  - Test: feature with no overrides → no warnings
  - Test: feature with one override → one warning containing feature ID, transition, and reason
  - Test: feature with multiple overrides → one warning per override
  - Test: non-feature entities are not checked
- Modified `internal/health/check.go`:
  - Add `CheckGateOverrides` call in `RunHealthCheck`, gated on `opts.Features` being non-empty
  - Register under category name `"gate_overrides"`
- Modified `internal/mcp/health_tool.go`:
  - In `Phase3HealthChecker`: load features via entity service, extract feature field maps, call `CheckGateOverrides`, merge result into report

**Dependencies:** Task 1 (OverrideRecord model — must know the field name and structure in the serialised state map)

**Interface contract:** The health check reads `overrides` from the feature's field map (`[]any` of maps with `from_status`, `to_status`, `reason`, `timestamp` keys). This must match the serialisation produced by Task 1.

---

## Dependency Graph

```
Task 1 (Model)          Task 2 (Gate Logic)
    │                       │
    │                       ▼
    │                   Task 3 (Error Messages)
    │                       │
    ▼                       │
Task 4 (Advance)◄───────────┤
    │                       │
    ▼                       ▼
    └──────►Task 5 (MCP Wiring)◄────────┘
                │
Task 6 (Health)◄┘
    │
    ▼
  (done)
```

**Parallel execution opportunities:**
- **Wave 1:** Task 1 and Task 2 can run in parallel (no dependencies between them)
- **Wave 2:** Task 3 (depends on Task 2), Task 4 (depends on Task 1 + Task 2), Task 6 (depends on Task 1) — Task 3 and Task 6 can run in parallel once their dependencies complete
- **Wave 3:** Task 5 (depends on Tasks 1–4) — this is the serial bottleneck, the integration task

**Critical path:** Task 2 → Task 3 → Task 5, or Task 1 + Task 2 → Task 4 → Task 5

---

## Traceability Matrix

| FR | Description | Task(s) |
|----|-------------|---------|
| FR-001 | Gates mandatory on all transitions (single-step and advance) | Task 4, Task 5 |
| FR-002 | Terminal-state transitions ungated | Task 2, Task 5 |
| FR-003 | Transitions with no prerequisites ungated | Task 2, Task 5 |
| FR-004 | designing→specifying: approved design document | Task 2 |
| FR-005 | specifying→dev-planning: approved specification | Task 2 |
| FR-006 | dev-planning→developing: approved dev-plan + child tasks | Task 2 |
| FR-007 | developing→reviewing: all child tasks terminal | Task 2 |
| FR-008 | reviewing→done: review report exists | Task 2 |
| FR-009 | needs-rework→developing: rework task exists | Task 2 |
| FR-010 | needs-rework→reviewing: all child tasks terminal | Task 2 |
| FR-011 | Override + override_reason parameters | Task 5 |
| FR-012 | Override without reason rejected | Task 5 |
| FR-013 | override_reason ignored when override is false | Task 5 |
| FR-014 | Override records persisted on feature | Task 1, Task 5 |
| FR-015 | Health tool flags override features | Task 6 |
| FR-016 | Override works with advance, each gate logged | Task 4, Task 5 |
| FR-017 | All gates default to agent override policy | Task 5 |
| FR-018 | Actionable error message template | Task 3 |
| FR-019 | designing→specifying recovery steps | Task 3 |
| FR-020 | specifying→dev-planning recovery steps | Task 3 |
| FR-021 | dev-planning→developing recovery steps | Task 3 |
| FR-022 | developing→reviewing recovery steps (lists tasks) | Task 3 |
| FR-023 | reviewing→done recovery steps | Task 3 |
| FR-024 | needs-rework→developing recovery steps | Task 3 |
| FR-025 | needs-rework→reviewing recovery steps | Task 3 |
| FR-026 | Structured error response (not protocol error) | Task 3, Task 5 |

| NFR | Description | Task(s) |
|-----|-------------|---------|
| NFR-001 | Gate evaluation < 100ms latency | Task 2 |
| NFR-002 | Phase 1 transitions unaffected | Task 5 |
| NFR-003 | No security boundary for override | Task 5 |
| NFR-004 | Deterministic error messages | Task 3 |
| NFR-005 | Advance behaviour preserved | Task 4 |