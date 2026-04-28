# Implementation Plan: Stage-Aware Context Assembly

**Feature:** FEAT-01KN5-8J25K4QD (stage-aware-context-assembly)
**Specification:** `work/spec/3.0-stage-aware-context-assembly.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §7, §8, §9, §13

---

## 1. Overview

This plan decomposes the stage-aware context assembly specification (14 FRs, 4 NFRs) into seven tasks. The decomposition isolates write-sets so that tasks touching different files can run in parallel. A single foundational task (Task 1) establishes the shared stage configuration data that all other tasks consume.

### Scope boundaries (carried forward from specification)

**In scope:** Lifecycle validation on handoff/next, stage-specific content inclusion/exclusion, orchestration pattern signalling, effort budgets, tool subset guidance, filesystem-output convention, finish summary limit, hardcoded stage config, actionable error messages, graceful degradation.

**Out of scope:** Hard tool filtering, binding registry, skill content, gate enforcement on feature transitions, 10-step assembly pipeline mechanics, feature lifecycle state machine changes.

---

## 2. Task Breakdown

### Task 1: Stage configuration data module

**Objective:** Create a single Go source file containing all hardcoded stage configuration data — allowed feature states, orchestration patterns, effort budgets, tool subsets, and content inclusion/exclusion flags. This file is the single source of truth that all other tasks import. Its data structures must match the binding registry's `stage_bindings` schema shape (NFR-003) so a future feature can swap the data source without changing consumers.

**Specification references:** FR-003, FR-005, FR-006, FR-007, FR-008, FR-009, FR-011, NFR-003

**Input context:**
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-003 state mapping table, FR-005 inclusion/exclusion table, FR-006 orchestration patterns, FR-007 effort budget table, FR-008 tool subset table, FR-009 convention text, FR-011 co-location requirement
- `internal/model/entities.go` — `FeatureStatus` constants (`FeatureStatusDesigning`, `FeatureStatusSpecifying`, etc.)
- `refs/go-style.md` — naming, package design, interface conventions

**Output artifacts:**
- New file: `internal/stage/config.go`
- New file: `internal/stage/config_test.go`
- New file: `internal/stage/doc.go` (package doc comment)

**Dependencies:** None — this is the foundation task.

**Detailed design:**

The package `internal/stage` exports the following types and functions:

```go
package stage

// Stage represents a feature lifecycle stage name.
type Stage string

const (
    Designing   Stage = "designing"
    Specifying  Stage = "specifying"
    DevPlanning Stage = "dev-planning"
    Developing  Stage = "developing"
    Reviewing   Stage = "reviewing"
    NeedsRework Stage = "needs-rework"
)

// OrchestrationPattern is the orchestration mode for a stage.
type OrchestrationPattern string

const (
    SingleAgent         OrchestrationPattern = "single-agent"
    OrchestratorWorkers OrchestrationPattern = "orchestrator-workers"
)

// StageConfig holds all stage-specific configuration for context assembly.
type StageConfig struct {
    Orchestration       OrchestrationPattern
    EffortBudget        EffortBudget
    PrimaryTools        []string
    ExcludedTools       []string
    IncludeFilePaths    bool
    IncludeTestExpect   bool
    IncludeReviewRubric bool
    IncludeImplGuidance bool
    IncludePlanGuidance bool
    OutputConvention    bool // true for orchestrator-workers stages
    SpecMode            string // "full" or "relevant-sections"
}

// EffortBudget holds effort expectation text for a stage.
type EffortBudget struct {
    Text    string
    Warning string // "Do NOT skip..." line
}

// ForStage returns the StageConfig for the given feature lifecycle stage.
// Returns the config and true if found, zero value and false if the stage
// is not a working stage (e.g. "proposed", "done").
func ForStage(featureStatus string) (StageConfig, bool)

// IsWorkingState returns true if the feature status permits task work.
// Working states: designing, specifying, dev-planning, developing,
// reviewing, needs-rework.
func IsWorkingState(featureStatus string) bool

// AllStages returns all configured stage names (for test iteration).
func AllStages() []Stage
```

The internal lookup is a `map[Stage]StageConfig` initialised at package level. No `init()` function — use a package-level `var` with a composite literal.

**Test requirements:**
- Every stage from the spec has an entry in the config map
- `IsWorkingState` returns true for all 6 working states, false for `proposed`, `done`, `superseded`, `cancelled`
- `ForStage` returns correct orchestration pattern per FR-006 table
- `ForStage` returns correct effort budget text per FR-007 table
- `ForStage` returns correct tool lists per FR-008 table
- `ForStage` returns `OutputConvention: true` only for developing, reviewing, needs-rework

---

### Task 2: Lifecycle state validation

**Objective:** Implement the lifecycle validation function that both `handoff` and `next` (claim mode) call before assembling context. The function resolves the task's parent feature, reads its status, checks it against `stage.IsWorkingState`, and returns an actionable error on failure. This is a pure validation function — it does not modify any state.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-014

**Input context:**
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-001 through FR-004 acceptance criteria, FR-004 error template, FR-014 graceful degradation
- `internal/mcp/assembly.go` — current `asmInput` struct (has `parentFeature` and `entitySvc` fields)
- `internal/mcp/handoff_tool.go` — current handler reads `parent_feature` from task state
- `internal/mcp/next_tool.go` — `nextClaimMode` loads the task and reads `parent_feature`
- `internal/service/entity_service.go` — `EntityService.Get` API for loading features

**Output artifacts:**
- New file: `internal/mcp/assembly_validate.go`
- New file: `internal/mcp/assembly_validate_test.go`

**Dependencies:** Task 1 (imports `stage.IsWorkingState`)

**Interface contract (consumed by Tasks 3 and 4):**

```go
// ValidateFeatureStage checks that the parent feature is in a working state.
//
// Returns:
//   - featureStatus (string): the resolved feature status, for use by stage-aware assembly
//   - err: nil if valid; actionable error message if invalid or feature missing
//
// When parentFeatureID is empty or the feature cannot be loaded, returns
// ("", nil) to signal graceful degradation (FR-014). The caller should
// proceed with non-stage-aware assembly.
func ValidateFeatureStage(
    parentFeatureID string,
    entitySvc EntityGetter,
) (featureStatus string, err error)
```

The `EntityGetter` interface is defined at the consumer (this file), not the provider:

```go
// EntityGetter is the subset of service.EntityService needed for validation.
type EntityGetter interface {
    Get(entityType, id, slug string) (*service.ListResult, error)
}
```

**Error message format (FR-004):** The error returned by `ValidateFeatureStage` must follow the three-part template verbatim from the spec. The function constructs the message using `fmt.Errorf` with the task ID, feature ID, and current state interpolated.

Note: `ValidateFeatureStage` itself does not know the task ID (it only receives the feature ID). The task ID is interpolated by the caller (handoff/next handler) when wrapping the error for the MCP response. To support this, the function returns a structured sentinel error type:

```go
// StageValidationError carries the data needed for the actionable error template.
type StageValidationError struct {
    FeatureID    string
    CurrentState string
}

func (e *StageValidationError) Error() string // returns the full actionable message template
```

The callers in Tasks 3 and 4 check `errors.As` for `*StageValidationError` and interpolate the task ID into the final user-facing message.

**Graceful degradation (FR-014):** When `parentFeatureID` is empty or the feature entity cannot be loaded, the function returns `("", nil)` — no error, no feature status. The caller interprets this as "proceed without stage-aware assembly". This avoids breaking orphaned tasks or tasks from before the parent_feature field existed.

**Test requirements:**
- Feature in each working state → returns that status, nil error
- Feature in `proposed` → returns `StageValidationError`
- Feature in `done` → returns `StageValidationError`
- Feature in `superseded` → returns `StageValidationError`
- Feature in `cancelled` → returns `StageValidationError`
- Error message contains the feature ID and current state in quotes
- Error message contains a tool call example (`entity(action: "get", ...)`)
- Empty `parentFeatureID` → returns `("", nil)`
- Feature not found (entity service returns error) → returns `("", nil)`

---

### Task 3: Handoff tool integration (validation + stage-aware rendering)

**Objective:** Wire lifecycle validation into the `handoff` handler (before assembly) and extend the prompt renderer to include the stage-aware sections in the correct high-attention-zone order. When validation rejects the request, the handler returns an error JSON and does not proceed to assembly. When the parent feature is missing or unresolvable, the handler falls back to current non-stage-aware behaviour with `stage_aware: false` in metadata.

**Specification references:** FR-001, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-012, FR-014

**Input context:**
- `internal/mcp/handoff_tool.go` — current handler and `renderHandoffPrompt`
- `internal/mcp/assembly.go` — `assembleContext`, `asmInput`, `assembledContext`
- `internal/mcp/assembly_validate.go` (from Task 2) — `ValidateFeatureStage`
- `internal/stage/config.go` (from Task 1) — `ForStage`, `StageConfig`
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-012 section ordering, FR-005 inclusion/exclusion table, FR-009 exact text, FR-006 exact text, FR-007 exact text

**Output artifacts:**
- Modified: `internal/mcp/handoff_tool.go` — validation call before assembly, metadata fields
- Modified: `internal/mcp/assembly.go` — extend `asmInput` with `featureStage string` field; extend `assembledContext` with stage-aware fields; modify `assembleContext` to populate stage-aware fields when `featureStage` is set; modify content inclusion based on stage
- New or modified: `internal/mcp/handoff_tool_test.go` — new test cases
- Modified: `internal/mcp/assembly_test.go` — new test cases for stage-aware assembly

**Dependencies:** Task 1, Task 2

**Changes to `asmInput`:**

Add a `featureStage` field:

```go
type asmInput struct {
    // ... existing fields ...
    featureStage string // resolved feature lifecycle stage; empty = non-stage-aware
}
```

**Changes to `assembledContext`:**

Add stage-aware fields consumed by both handoff and next renderers:

```go
type assembledContext struct {
    // ... existing fields ...
    stageAware         bool   // true when stage-aware assembly was used
    featureStage       string // the resolved stage
    orchestrationText  string // rendered orchestration section (FR-006)
    effortBudgetText   string // rendered effort budget section (FR-007)
    toolSubsetText     string // rendered tool subset section (FR-008)
    outputConventionText string // rendered output convention section (FR-009), empty for single-agent
}
```

**Changes to `assembleContext`:**

When `input.featureStage` is non-empty:
1. Look up `stage.ForStage(input.featureStage)` to get `StageConfig`
2. Set `actx.stageAware = true`, `actx.featureStage = input.featureStage`
3. Render the orchestration, effort budget, tool subset, and output convention text blocks using the exact templates from the spec
4. Apply content inclusion/exclusion from `StageConfig`: skip file paths for designing/specifying stages (`IncludeFilePaths: false`), skip review rubrics for developing stage, etc.

When `input.featureStage` is empty, all stage-aware fields remain zero-valued (backward compatible).

**Changes to `renderHandoffPrompt`:**

Insert stage-aware sections after conventions and before task summary, per FR-012 ordering:

1. Role identity / conventions (existing)
2. **Orchestration pattern** (new — `actx.orchestrationText`)
3. **Effort expectations** (new — `actx.effortBudgetText`)
4. **Tools for this task** (new — `actx.toolSubsetText`)
5. **Output convention** (new — `actx.outputConventionText`, only when non-empty)
6. Task summary and description (existing)
7. Specification sections (existing)
8. Acceptance criteria (existing)
9. Knowledge constraints (existing)
10. File paths (existing — now conditionally included based on stage)
11. Additional instructions (existing)

**Changes to handoff handler:**

```
1. Load task (existing)
2. Validate task status (existing)
3. NEW: Extract parentFeature from task state
4. NEW: Call ValidateFeatureStage(parentFeature, entitySvc)
5. NEW: If StageValidationError → return handoffErrorJSON with actionable message (include task ID)
6. NEW: Set featureStage on asmInput
7. Pre-dispatch state commit (existing)
8. assembleContext (existing, now with featureStage)
9. renderHandoffPrompt (existing, now renders stage-aware sections)
10. NEW: Add stage_aware and feature_stage to context_metadata
```

**Test requirements:**
- Handoff on task with parent feature in `proposed` → error JSON, no assembly
- Handoff on task with parent feature in `developing` → success, prompt contains orchestration and effort sections
- Handoff on task with parent feature in `specifying` → prompt contains single-agent text, no file paths
- Handoff on task with no parent feature → succeeds, `stage_aware: false` in metadata
- Prompt section ordering: orchestration before task summary, after conventions
- Orchestration text matches spec verbatim for both patterns
- Effort budget text matches spec for each stage
- Tool subset text matches spec for each stage
- Output convention present for developing/reviewing/needs-rework, absent for designing/specifying/dev-planning
- File paths excluded for designing/specifying stages per FR-005

---

### Task 4: Next tool integration (validation + structured response fields)

**Objective:** Wire lifecycle validation into the `next` claim-mode handler (before claiming the task) and add stage-aware fields to the structured response. Queue mode is unaffected. When validation rejects the request, the handler returns an error and the task is NOT transitioned to active. When the parent feature is missing, fall back to non-stage-aware assembly with `stage_aware: false`.

**Specification references:** FR-002, FR-004, FR-013, FR-014

**Input context:**
- `internal/mcp/next_tool.go` — `nextClaimMode`, `nextContextToMap`
- `internal/mcp/assembly.go` — `assembleContext`, `assembledContext` (as modified by Task 3)
- `internal/mcp/assembly_validate.go` (from Task 2) — `ValidateFeatureStage`
- `internal/stage/config.go` (from Task 1) — `ForStage`, `StageConfig`
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-002 acceptance criteria, FR-013 field definitions

**Output artifacts:**
- Modified: `internal/mcp/next_tool.go` — validation call in `nextClaimMode` before task claim, updated `nextContextToMap`
- New or modified: `internal/mcp/next_tool_test.go` — new test cases

**Dependencies:** Task 1, Task 2, Task 3 (Task 3 modifies `assembledContext` and `assembleContext` that this task consumes)

**Changes to `nextClaimMode`:**

Insert validation between task load and task claim:

```
1. Resolve task ID (existing)
2. Load task, check status == "ready" (existing)
3. NEW: Extract parentFeature from task state
4. NEW: Call ValidateFeatureStage(parentFeature, entitySvc)
5. NEW: If StageValidationError → return error (task stays in "ready")
6. Claim task: ready → active (existing, now only reached if validation passes)
7. NEW: Set featureStage on asmInput
8. assembleContext (existing, now with featureStage)
9. Build response (existing, now with stage-aware fields)
```

The critical ordering guarantee: validation happens BEFORE the `dispatchSvc.DispatchTask` call (step 6). If validation fails, the task remains in `ready` status — it is never transitioned to `active`.

**Changes to `nextContextToMap`:**

Add the five new fields to the context map per FR-013:

```go
if actx.stageAware {
    out["stage_aware"] = true
    out["feature_stage"] = actx.featureStage
    out["orchestration_pattern"] = orchestrationPatternString // "single-agent" or "orchestrator-workers"
    out["effort_budget"] = map[string]any{
        "stage":   actx.featureStage,
        "text":    effortBudgetText,
        "warning": effortWarningText,
    }
    out["tool_subset"] = map[string]any{
        "primary":  primaryToolsList,
        "excluded": excludedToolsList,
    }
    if outputConventionText != "" {
        out["output_convention"] = outputConventionText
    }
} else {
    out["stage_aware"] = false
}
```

The `orchestration_pattern`, `effort_budget`, and `tool_subset` values come from the `StageConfig` looked up via `stage.ForStage(actx.featureStage)`. This is a second lookup (the first is in `assembleContext`) but it's an O(1) map access and avoids adding raw config fields to `assembledContext`.

**Changes to `nextClaimMode` for feature ID input (FR-002):**

When `next(id="FEAT-...")` is called, `nextResolveTaskID` finds the top ready task. The validation must still check the FEATURE status, not just the task status. The feature ID is already available since that's the input. Add validation after `nextResolveTaskID` returns and before the task is loaded:

```
1. nextResolveTaskID → taskID (existing)
2. NEW: If input was a feature ID, validate feature stage directly
3. Load task (existing)
4. NEW: If input was a task ID, extract parent_feature and validate
5. ... continue as above
```

**Test requirements:**
- `next(id="TASK-...")` with parent feature in `proposed` → error, task remains `ready`
- `next(id="TASK-...")` with parent feature in `developing` → task claimed, response includes `orchestration_pattern: "orchestrator-workers"`
- `next(id="FEAT-...")` with feature in `proposed` → error, no task claimed
- `next(id="FEAT-...")` with feature in `developing` → task claimed
- Queue mode `next()` → unaffected, no validation
- Claim response for `developing` includes `output_convention` field
- Claim response for `specifying` does NOT include `output_convention`
- Claim response includes `feature_stage` matching parent feature status
- Claim response includes `stage_aware: true` when stage-aware assembly succeeds
- Claim response for orphaned task includes `stage_aware: false`
- `tool_subset.primary` for `developing` matches FR-008 list exactly

---

### Task 5: Finish summary length limit

**Objective:** Add a 500-character maximum length validation on the `summary` field in the `finish` tool. The check applies in both single-item and batch modes. When the limit is exceeded, the tool returns an actionable error stating the limit and actual length. This task is fully independent — it touches only the finish tool, not the assembly pipeline.

**Specification references:** FR-010

**Input context:**
- `internal/mcp/finish_tool.go` — `finishOne` function (the existing `summary is required` check at line ~180)
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-010 acceptance criteria

**Output artifacts:**
- Modified: `internal/mcp/finish_tool.go` — add length check after the `summary is required` check in `finishOne`
- New or modified: `internal/mcp/finish_tool_test.go` — new test cases

**Dependencies:** None — fully independent of all other tasks.

**Implementation detail:**

In `finishOne`, immediately after the existing `summary is required` check:

```go
if len(input.Summary) > 500 {
    return nil, fmt.Errorf(
        "summary exceeds 500-character limit (%d characters provided). "+
            "Truncate the summary to 500 characters or fewer",
        len(input.Summary),
    )
}
```

This uses `len()` (byte length) which matches the spec's "500 characters" for ASCII-dominated summaries. The spec does not distinguish bytes from runes, and summaries are expected to be plain English text.

For batch mode: `finishOne` is already called per-item, so the limit is enforced independently per batch item — a single oversized summary fails that item without affecting others (FR-010 acceptance criterion).

**Test requirements:**
- 500-character summary → succeeds (passes through to existing logic)
- 501-character summary → returns error mentioning "500-character limit" and "501"
- 499-character summary → succeeds
- Empty summary → still rejected by existing "summary is required" (no regression)
- Batch mode: one item at 501 chars fails independently, other items succeed

---

### Task 6: Stage-aware content filtering in assembly

**Objective:** Implement the content inclusion/exclusion logic in `assembleContext` so that assembled context varies by stage. For designing/specifying stages, file paths and implementation tool guidance are excluded. For developing, review rubrics are excluded. For reviewing, implementation tool guidance is excluded. The spec document is included in full for specifying, but only relevant sections for developing/reviewing/needs-rework.

**Specification references:** FR-005, FR-012 (content aspects)

**Input context:**
- `internal/mcp/assembly.go` — `assembleContext`, `asmExtractFiles`, `asmExtractSpecSections`
- `internal/stage/config.go` (from Task 1) — `StageConfig.IncludeFilePaths`, `StageConfig.SpecMode`, etc.
- `work/spec/3.0-stage-aware-context-assembly.md` — FR-005 inclusion/exclusion table

**Output artifacts:**
- Modified: `internal/mcp/assembly.go` — conditional content inclusion based on `featureStage`
- Modified: `internal/mcp/assembly_test.go` — tests for content filtering per stage

**Dependencies:** Task 1, Task 3 (Task 3 adds `featureStage` to `asmInput` and stage-aware fields to `assembledContext`)

**Implementation detail:**

Inside `assembleContext`, after looking up the `StageConfig`:

1. **File paths:** Only call `asmExtractFiles` when `cfg.IncludeFilePaths` is true. For designing and specifying stages, `actx.filesContext` remains empty.

2. **Spec mode:** When `cfg.SpecMode == "full"`, load the entire spec document content instead of only traced sections. This requires calling `docRecordSvc` to find the spec document and reading its full content. When `cfg.SpecMode == "relevant-sections"`, use the existing `asmExtractSpecSections` behaviour.

3. **Review findings (needs-rework):** For the `needs-rework` stage, include previous review findings when available. This means looking up review-type documents for the parent feature via doc intelligence.

These are conditional branches within the existing `assembleContext` function — no new exported functions are needed.

**Test requirements:**
- Designing stage: `filesContext` is empty even when task has `files_planned`
- Specifying stage: `filesContext` is empty; spec sections contain full document content
- Developing stage: `filesContext` is populated; spec contains only relevant sections
- Reviewing stage: `filesContext` is populated; review rubric content is present
- Needs-rework stage: includes previous review findings when they exist

---

### Task 7: Integration tests and regression suite

**Objective:** Create end-to-end integration tests that exercise the full handoff and next flows with stage-aware context, verifying the complete path from tool call through validation, assembly, and response rendering. Also verify that existing (non-stage-aware) behaviour is preserved for backward compatibility (NFR-002).

**Specification references:** NFR-001, NFR-002, NFR-004 (and indirectly all FRs via end-to-end verification)

**Input context:**
- `internal/mcp/handoff_tool_test.go` — existing handoff test patterns
- `internal/mcp/assembly_test.go` — existing assembly test patterns
- `internal/mcp/next_tool_test.go` — existing next test patterns (if present)
- All files modified by Tasks 1–6

**Output artifacts:**
- New file: `internal/mcp/stage_integration_test.go`
- Modified (if needed): existing test files to prevent regressions

**Dependencies:** Tasks 1, 2, 3, 4, 5, 6 (all must complete before integration testing)

**Test scenarios:**

1. **Full handoff flow — developing stage:** Create mock task with parent feature in `developing`. Call handoff handler. Assert: prompt contains orchestration (multi-agent), effort budget (10–50 tool calls), tool subset (entity, handoff, next, finish...), output convention, file paths, spec sections. Assert section ordering matches FR-012.

2. **Full handoff flow — specifying stage:** Create mock task with parent feature in `specifying`. Call handoff handler. Assert: prompt contains orchestration (single-agent), effort budget (5–15 tool calls), tool subset, NO file paths, NO output convention, full spec document.

3. **Full next claim flow — developing stage:** Create mock task in `ready` with parent feature in `developing`. Call next handler with task ID. Assert: task transitions to `active`, response contains `stage_aware: true`, `orchestration_pattern: "orchestrator-workers"`, `effort_budget`, `tool_subset`, `output_convention`, `feature_stage: "developing"`.

4. **Validation rejection — handoff:** Create mock task with parent feature in `done`. Call handoff handler. Assert: error JSON returned, error contains task ID, feature ID, current state, recovery instructions.

5. **Validation rejection — next:** Create mock task in `ready` with parent feature in `proposed`. Call next handler. Assert: error returned, task remains in `ready`.

6. **Graceful degradation:** Create mock task with no parent feature. Call handoff handler. Assert: prompt is generated (non-stage-aware), metadata includes `stage_aware: false`.

7. **Backward compatibility:** Create mock task with parent feature in `developing`, call handoff with no role. Assert: existing sections (spec, knowledge, constraints) still present alongside new stage-aware sections.

8. **Finish summary limit:** Call finish with 501-character summary. Assert: error. Call with 500-character summary. Assert: passes through to existing logic.

---

## 3. Dependency Graph

```
Task 1: Stage config data module
  ├──► Task 2: Lifecycle state validation
  │      ├──► Task 3: Handoff tool integration
  │      │      ├──► Task 4: Next tool integration
  │      │      └──► Task 6: Stage-aware content filtering
  │      └──► Task 4: Next tool integration
  └──► Task 3: Handoff tool integration
       └──► Task 6: Stage-aware content filtering

Task 5: Finish summary limit (independent)

Task 7: Integration tests (depends on all of 1–6)
```

**Parallel execution opportunities:**

| Phase | Tasks | Notes |
|-------|-------|-------|
| Phase 1 | Task 1, Task 5 | Fully independent. Task 1 is the foundation; Task 5 touches only finish_tool.go. |
| Phase 2 | Task 2 | Depends on Task 1 only. |
| Phase 3 | Task 3 | Depends on Tasks 1 and 2. |
| Phase 4 | Task 4, Task 6 | Both depend on Task 3. Can run in parallel: Task 4 writes to next_tool.go; Task 6 writes to assembly.go (different sections than Task 3's changes). |
| Phase 5 | Task 7 | Integration tests, depends on all prior tasks. |

**Critical path:** Task 1 → Task 2 → Task 3 → Task 4 → Task 7

---

## 4. Write-Set Boundaries

Each task's primary write-set is documented here to prevent agent conflicts.

| Task | Creates | Modifies | Shared read |
|------|---------|----------|-------------|
| 1 | `internal/stage/config.go`, `internal/stage/config_test.go`, `internal/stage/doc.go` | — | `internal/model/entities.go` |
| 2 | `internal/mcp/assembly_validate.go`, `internal/mcp/assembly_validate_test.go` | — | `internal/mcp/assembly.go` |
| 3 | — | `internal/mcp/handoff_tool.go`, `internal/mcp/assembly.go` (types + assembleContext), `internal/mcp/handoff_tool_test.go`, `internal/mcp/assembly_test.go` | — |
| 4 | — | `internal/mcp/next_tool.go`, `internal/mcp/next_tool_test.go` | `internal/mcp/assembly.go` (reads assembledContext) |
| 5 | — | `internal/mcp/finish_tool.go`, `internal/mcp/finish_tool_test.go` | — |
| 6 | — | `internal/mcp/assembly.go` (assembleContext body — content filtering branches) | `internal/stage/config.go` |
| 7 | `internal/mcp/stage_integration_test.go` | — | All of the above |

**Conflict notes:**
- Tasks 3 and 6 both modify `internal/mcp/assembly.go`. Task 3 adds types and the stage-aware field population at the top of `assembleContext`. Task 6 adds conditional content filtering later in `assembleContext`. Task 6 depends on Task 3 to avoid conflicts.
- Task 4 only modifies `next_tool.go` and reads (does not write) `assembly.go`.
- Task 5 is fully isolated to `finish_tool.go`.

---

## 5. Traceability Matrix

| Requirement | Task(s) | Verification |
|---|---|---|
| FR-001 (Handoff validation) | Task 2, Task 3 | Unit test: mock feature in terminal state, assert error and no assembly |
| FR-002 (Next validation) | Task 2, Task 4 | Unit test: mock feature in `proposed`, assert error and task not transitioned |
| FR-003 (State mapping) | Task 1, Task 2 | Unit test: iterate all feature states, assert correct accept/reject |
| FR-004 (Error messages) | Task 2 | Unit test: assert error contains task ID, feature ID, state, tool call example |
| FR-005 (Stage-specific content) | Task 1, Task 3, Task 6 | Unit test per stage: assert presence/absence of each content category |
| FR-006 (Orchestration signal) | Task 1, Task 3 | Unit test per stage: assert correct signal text in assembled output |
| FR-007 (Effort budget) | Task 1, Task 3 | Unit test per stage: assert correct budget text and warning line |
| FR-008 (Tool subset) | Task 1, Task 3 | Unit test per stage: assert correct primary and excluded tool lists |
| FR-009 (Filesystem convention) | Task 1, Task 3 | Unit test: present for orchestrator-workers, absent for single-agent |
| FR-010 (Finish summary limit) | Task 5 | Unit test: 500 succeeds, 501 fails with limit stated |
| FR-011 (Hardcoded config) | Task 1 | Unit test: all stages have entries; code review: single file |
| FR-012 (Handoff rendering order) | Task 3 | Unit test: assert section ordering in rendered Markdown |
| FR-013 (Next structured response) | Task 4 | Unit test: assert new fields present and typed correctly |
| FR-014 (Graceful degradation) | Task 2, Task 3, Task 4 | Unit test: nil parent feature assembles without stage-aware sections |
| NFR-001 (Performance) | Task 1 | Design: O(1) map lookup, no I/O |
| NFR-002 (Backward compat) | Task 3, Task 4, Task 7 | Integration test: existing callers see unchanged behaviour |
| NFR-003 (Forward compat) | Task 1 | Design: data structure matches binding registry schema |
| NFR-004 (Testability) | All | Each behaviour independently testable without full MCP server |

Every FR appears in at least one task. Every task traces back to at least one FR.