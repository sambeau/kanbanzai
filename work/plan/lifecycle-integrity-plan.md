# Lifecycle Integrity and Proactive Status — Implementation Plan

> Plan for FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Plan: P19-workflow-lifecycle-integrity
> Specification: work/spec/lifecycle-integrity.md (FEAT-01KN83QN0VAFG/specification-lifecycle-integrity)

---

## Scope

This plan implements the requirements defined in `work/spec/lifecycle-integrity.md`
(FEAT-01KN83QN0VAFG/specification-lifecycle-integrity). It covers seven tasks
decomposed across three pillars:

- **Pillar A** (T1–T2): Child-state query helpers and lifecycle gates on terminal transitions
- **Pillar B** (T3–T4): Feature auto-advance to `reviewing` and plan auto-advance to `done`
- **Pillar C** (T5–T7): Structured `AttentionItem` schema, stale reviewing + bug warnings, health finding injection

This plan does not cover: changes to the `health` tool itself, any new entity fields,
background or event-driven processing, or auto-advancing past `reviewing` to `done`.

---

## Task Breakdown

### Task 1: Child-state query helpers

- **Description:** Extract the shared query logic needed by both the gate (Pillar A)
  and the auto-advance (Pillar B) into a small set of focused helper functions in the
  entity service. This task produces no user-visible behaviour — it is a foundation
  that the gate and auto-advance tasks build on.

- **Deliverable:**
  - New functions in `internal/service/entities.go` (or a new
    `internal/service/entity_children.go` if the file is too large to extend cleanly):
    - `countNonTerminalTasks(featureID string) (nonTerminal int, err error)` — returns
      the count of tasks with `parent_feature == featureID` that are in a non-terminal
      state. Terminal task states: `done`, `not-planned`, `duplicate`.
    - `countNonTerminalFeatures(planID string) (nonTerminal int, err error)` — returns
      the count of features with `parent == planID` that are in a non-terminal state.
      Terminal feature states: `done`, `superseded`, `cancelled`.
    - `checkAllTasksTerminal(featureID string) (allTerminal bool, hasOneDone bool, err error)` —
      returns whether all tasks are terminal AND whether at least one is `done`.
      Both values are needed to distinguish completion from abandonment.
    - `checkAllFeaturesTerminal(planID string) (allTerminal bool, hasOneDone bool, err error)` —
      mirror of the above at the plan/feature level.
  - Unit tests for all four helpers covering: empty children, mixed states, all
    terminal with at least one done, all terminal with none done (all not-planned),
    and storage error paths.

- **Depends on:** None

- **Effort:** Small

- **Spec requirements:** REQ-001, REQ-003, REQ-005, REQ-007, REQ-008, REQ-009,
  REQ-013, REQ-014

- **Input context:**
  - `internal/service/entities.go` — `List` method, existing cross-entity query patterns
  - `internal/model/` — entity kind constants and terminal state definitions
  - `internal/validate/validate.go` — `IsTerminalState` function (reuse, do not duplicate)
  - `internal/health/entity_consistency.go` — `CheckFeatureChildConsistency` for
    reference on how child queries are currently structured

- **Interface contract (consumed by T2 and T3):**
  All four helpers are methods on `*EntityService`. Exact signatures:
  ```
  func (s *EntityService) countNonTerminalTasks(featureID string) (int, error)
  func (s *EntityService) countNonTerminalFeatures(planID string) (int, error)
  func (s *EntityService) checkAllTasksTerminal(featureID string) (bool, bool, error)
  func (s *EntityService) checkAllFeaturesTerminal(planID string) (bool, bool, error)
  ```
  These are unexported (lowercase) — they are internal helpers, not part of the
  public service interface.

---

### Task 2: Lifecycle gates on terminal transitions

- **Description:** Add child-state precondition checks to the feature and plan
  transition handlers. When a feature is transitioning to `done`, `superseded`, or
  `cancelled`, query its tasks and block if any are non-terminal. Same check at plan
  level against features. Honour the existing `override` mechanism to bypass. Handle
  storage errors with a best-effort warning rather than a hard block.

- **Deliverable:**
  - Modified `internal/service/entities.go` — gate logic injected into the feature
    and plan transition paths:
    - Before accepting a terminal transition on a feature: call
      `countNonTerminalTasks`; if count > 0 and override is false, return an error
      of the form `"feature <ID> has <N> non-terminal task(s); use override to bypass"`.
    - Before accepting a terminal transition on a plan: call
      `countNonTerminalFeatures`; same pattern.
    - If the query returns an error: log a warning, permit the transition (best-effort).
    - If `override: true` and `override_reason` is non-empty: skip the gate,
      record the override on the entity (existing mechanism — do not re-implement,
      just ensure the gate is skipped at the right point in the existing override flow).
  - Unit tests covering AC-001 through AC-009:
    - Feature blocked with ready task
    - Feature allowed with all tasks done
    - Feature allowed with mixed terminal states (done + not-planned + duplicate)
    - Feature blocked on `superseded` transition
    - Feature blocked on `cancelled` transition
    - Plan blocked with developing feature
    - Feature with no children passes unconditionally
    - Override bypasses gate and records override
    - Storage error permits transition with warning

- **Depends on:** Task 1

- **Effort:** Medium

- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007

- **Input context:**
  - `internal/service/entities.go` — existing `TransitionFeature` / `TransitionPlan`
    methods (or however terminal transitions are currently implemented); the existing
    override recording mechanism
  - `internal/model/` — terminal state sets for features and tasks
  - Task 1 output — `countNonTerminalTasks`, `countNonTerminalFeatures`

---

### Task 3: Feature auto-advance

- **Description:** After any operation that moves a task to a terminal state, check
  whether all sibling tasks are now terminal with at least one `done`. If so, and the
  parent feature is in `developing` or `needs-rework`, automatically transition the
  feature to `reviewing`. Record the advance as a side effect in the triggering
  operation's response. If the auto-advance itself fails, surface it as a warning side
  effect — do not fail the primary operation.

  The trigger must fire from three call sites:
  1. `entity(action: transition)` on a task reaching a terminal state
  2. `finish` (the most common path)
  3. `complete_task`

- **Deliverable:**
  - New unexported function in `internal/service/entities.go`:
    `func (s *EntityService) maybeAutoAdvanceFeature(featureID string) (advanced bool, err error)`
    — calls `checkAllTasksTerminal`; if the guard conditions are met, calls the feature
    transition to `reviewing`; returns whether the advance fired and any error.
  - Modified task transition path in `internal/service/entities.go`: call
    `maybeAutoAdvanceFeature` after successfully writing terminal task state; append
    the result as a side effect of type `feature_auto_advanced` with fields
    `entity_id`, `from_status`, `to_status`.
  - Modified `internal/mcp/finish_tool.go` (or `internal/service/finish.go` —
    follow the existing pattern): call `maybeAutoAdvanceFeature` on the parent feature
    after task state is written; include the side effect in the `finish` response.
  - Same change to `complete_task` if it is a distinct code path.
  - Unit tests covering AC-010 through AC-014 and AC-017:
    - Last task completed via entity transition triggers feature advance
    - Feature in `needs-rework` also advances
    - All tasks `not-planned` (none done) — no advance
    - `finish` call triggers advance
    - Feature already in `reviewing` — no second advance
    - Auto-advance failure surfaced as warning, task transition succeeds

- **Depends on:** Task 1

- **Effort:** Medium

- **Spec requirements:** REQ-008, REQ-009, REQ-010, REQ-011, REQ-012

- **Input context:**
  - `internal/service/entities.go` — task transition handler and side effect patterns
  - `internal/mcp/finish_tool.go` — finish tool response structure and side effect
    recording (look at how `task_unblocked` side effects are currently appended)
  - `internal/model/` — feature status constants (`developing`, `needs-rework`,
    `reviewing`)
  - Task 1 output — `checkAllTasksTerminal`

- **Interface contract (consumed by T4):**
  The pattern established by `maybeAutoAdvanceFeature` is mirrored at plan level by T4.
  T4 should follow the same function shape:
  ```
  func (s *EntityService) maybeAutoAdvancePlan(planID string) (advanced bool, err error)
  ```
  T4 can be implemented independently once T3 is done.

---

### Task 4: Plan auto-advance

- **Description:** Mirror of Task 3 at the plan level. After a feature transitions to
  a terminal state, check whether all sibling features are now terminal with at least
  one `done`. If so, and the plan is `active`, automatically transition the plan to
  `done`. Record as a side effect; surface failures as warnings without blocking the
  primary feature transition.

- **Deliverable:**
  - New unexported function in `internal/service/entities.go`:
    `func (s *EntityService) maybeAutoAdvancePlan(planID string) (advanced bool, err error)`
  - Modified feature transition path: call `maybeAutoAdvancePlan` after successfully
    writing terminal feature state; append `plan_auto_advanced` side effect.
  - Unit tests covering AC-015, AC-016:
    - Last feature done triggers plan advance
    - All features superseded (none done) — no advance

- **Depends on:** Task 3

- **Effort:** Small

- **Spec requirements:** REQ-013, REQ-014, REQ-015, REQ-016

- **Input context:**
  - `internal/service/entities.go` — feature transition handler, side effect pattern
    from Task 3
  - `internal/model/` — plan status constants (`active`, `done`)
  - Task 1 output — `checkAllFeaturesTerminal`
  - Task 3 output — `maybeAutoAdvanceFeature` as the pattern to follow

---

### Task 5: AttentionItem struct and schema migration

- **Description:** Define the `AttentionItem` struct and migrate all `attention`
  fields in `status` response objects from `[]string` to `[]AttentionItem`. Update
  every `generateXxxAttention` function to produce `[]AttentionItem`. Preserve all
  existing message text verbatim — this task changes structure only, not content.
  Assign the correct `type` value from the registry to each existing attention item.

- **Deliverable:**
  - New `AttentionItem` struct defined in `internal/mcp/status_tool.go`:
    ```go
    type AttentionItem struct {
        Type      string `json:"type"`
        Severity  string `json:"severity"`
        EntityID  string `json:"entity_id,omitempty"`
        DisplayID string `json:"display_id,omitempty"`
        Message   string `json:"message"`
    }
    ```
  - Updated response structs: `projectOverview`, `planDashboard`, `featureDetail`,
    `taskDetail`, `bugDetail` — `Attention` field type changed from `[]string` to
    `[]AttentionItem`.
  - Updated functions: `generateProjectAttention`, `generatePlanAttention`,
    `generateFeatureAttention`, `generateTaskAttention` — return type changed to
    `[]AttentionItem`; each existing string item wrapped with appropriate `type` and
    `severity` from the registry in REQ-019.
  - Type assignments for existing items:
    - "N task(s) ready to claim" → `ready_tasks` / `info`
    - "active task(s) stalled for >3 days" → `stalled_task` / `warning`
    - "has been active for >24h with no recent commits" → `stuck_task` / `warning`
    - "has all N/N tasks done — ready to advance" → `all_tasks_done` / `warning`
    - "Plan X has all N features done — ready to close" → `plan_ready_to_close` / `info`
    - "has no tasks" → `feature_no_tasks` / `info`
    - "Missing specification document" → `missing_spec` / `warning`
    - "Missing dev-plan document" → `missing_dev_plan` / `warning`
  - `entity_id` and `display_id` populated wherever the entity is already available
    in the generation function's context.
  - Unit tests covering AC-018 and AC-030:
    - Schema migration preserves existing message strings
    - Consumer reading only `message` field receives identical content

- **Depends on:** None

- **Effort:** Medium

- **Spec requirements:** REQ-017, REQ-018, REQ-019

- **Input context:**
  - `internal/mcp/status_tool.go` — full file; all status structs and generation
    functions (this is a large file; read the outline first, then targeted sections)

- **Interface contract (consumed by T6 and T7):**
  `AttentionItem` is the only shared type. Both T6 and T7 construct `[]AttentionItem`
  slices and append them into the appropriate `attention` fields — no additional
  interface beyond the struct definition is required.

---

### Task 6: Stale reviewing detection and bug warnings

- **Description:** Add two new attention item generators to `generateFeatureAttention`:
  (1) stale reviewing — emit `stale_reviewing` if the feature has been in `reviewing`
  for longer than the configured threshold; (2) open critical/high bug warnings — emit
  `open_critical_bug` for each qualifying open bug against the feature. Add the
  `lifecycle.stale_reviewing_days` config key with a default of 7.

- **Deliverable:**
  - Modified `internal/config/config.go` (or equivalent config struct file):
    new `Lifecycle.StaleReviewingDays int` field, default 7, read from
    `lifecycle.stale_reviewing_days` in `config.yaml`. Zero value disables the check.
  - Modified `internal/mcp/status_tool.go` — `generateFeatureAttention`:
    - Stale reviewing: if `featureStatus == "reviewing"` and `featureUpdated` is
      non-zero and `time.Since(featureUpdated) > threshold`, append:
      ```
      AttentionItem{Type: "stale_reviewing", Severity: "warning",
          EntityID: featureID, DisplayID: featureDisplayID,
          Message: fmt.Sprintf("Feature has been in reviewing for %d days", days)}
      ```
    - Bug warnings: load open bugs with `origin_feature == featureID`; for each bug
      where `(severity == "critical") OR (priority == "critical" OR priority == "high")`
      AND status not in `{done, not-planned, duplicate, wont-fix}`, append:
      ```
      AttentionItem{Type: "open_critical_bug", Severity: "warning",
          EntityID: bugID, DisplayID: bugDisplayID,
          Message: fmt.Sprintf("Open %s bug: %s", severity/priority, bugName)}
      ```
  - The `generateFeatureAttention` function signature must be extended to accept the
    config threshold (pass as an `int` parameter, not a global) and the entity service
    (or a pre-loaded slice of bugs) for the bug query.
  - Unit tests covering AC-019 through AC-027:
    - `all_tasks_done` item present when feature developing with all tasks terminal
    - Stale reviewing fires when over threshold, absent when under, absent when
      threshold is 0, absent when `updated` field is zero
    - Bug warning fires for critical severity, high priority; absent for low/low;
      absent when feature is done

- **Depends on:** Task 5

- **Effort:** Medium

- **Spec requirements:** REQ-022, REQ-023, REQ-024, REQ-025, REQ-026, REQ-027,
  REQ-NF-001 (the bug query should be scoped to the specific feature, not a full scan)

- **Input context:**
  - `internal/mcp/status_tool.go` — `generateFeatureAttention`, `synthesiseFeature`,
    `featureDetail` struct
  - `internal/config/` — config struct and YAML parsing conventions
  - `internal/service/entities.go` — how bugs are listed/queried (look for existing
    bug list patterns); `origin_feature` field on bug entities
  - `internal/model/` — bug severity and priority constants
  - Task 5 output — `AttentionItem` struct

---

### Task 7: Health findings injection into project-level status

- **Description:** At project-level `status` (no `id` argument), run the entity health
  check and inject its findings as `AttentionItem` entries into the `attention` array.
  Retain the `health` summary field (`{errors: N, warnings: N}`) alongside the injected
  items. Scoped calls (plan, feature, task) do not run the full health check.

- **Deliverable:**
  - Modified `internal/mcp/status_tool.go` — `synthesiseProject`:
    - After building the existing attention items, call `entitySvc.HealthCheck()`.
    - For each error in the report: append `AttentionItem{Type: "health_error",
      Severity: "error", EntityID: finding.EntityID, Message: finding.Message}`.
    - For each warning: append `AttentionItem{Type: "health_warning",
      Severity: "warning", EntityID: finding.EntityID, Message: finding.Message}`.
    - If `HealthCheck()` fails: do not fail the status call; append no health items
      (the existing `buildHealthSummary` already handles this gracefully — follow
      the same pattern).
    - The `health` summary field (`statusHealthSummary`) is retained unchanged.
  - Unit tests covering AC-028 and AC-029:
    - Mock health check returning two warnings: assert two `health_warning` items
      in `attention`, each with correct `entity_id`
    - Assert `health` summary counts field still present alongside attention items

- **Depends on:** Task 5

- **Effort:** Small

- **Spec requirements:** REQ-020, REQ-021

- **Input context:**
  - `internal/mcp/status_tool.go` — `synthesiseProject`, `buildHealthSummary`,
    `projectOverview` struct
  - `internal/service/entities.go` — `HealthCheck()` method
  - `internal/validate/health.go` — `HealthReport` struct (errors and warnings fields)
  - Task 5 output — `AttentionItem` struct

---

## Dependency Graph

```
T1: Child-state query helpers          (no dependencies)
T2: Lifecycle gates                    → depends on T1
T3: Feature auto-advance               → depends on T1
T4: Plan auto-advance                  → depends on T3
T5: AttentionItem struct + migration   (no dependencies)
T6: Stale reviewing + bug warnings     → depends on T5
T7: Health findings injection          → depends on T5
```

**Parallel execution groups:**

- **Round 1 (parallel):** T1, T5 — independent foundations
- **Round 2 (parallel):** T2, T3, T6, T7 — T2 and T3 unblock after T1; T6 and T7
  unblock after T5; all four can run concurrently
- **Round 3 (serial):** T4 — depends on T3

**Critical path:** T1 → T3 → T4 (3 tasks, approximately medium + medium + small)

**Maximum parallelism:** 4 agents in round 2

---

## Risk Assessment

### Risk: Entity service file size

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** `internal/service/entities.go` is already large. If adding T1–T4
  makes it unwieldy, extract helpers into a new `internal/service/entity_children.go`.
  The split is clean — the four helpers and two auto-advance functions have no
  dependencies on the rest of the file beyond `s.List()`.
- **Affected tasks:** T1, T2, T3, T4

### Risk: `finish` side effect ordering

- **Probability:** Low
- **Impact:** Medium
- **Mitigation:** The `finish` tool already appends `task_unblocked` side effects
  when dependencies are resolved. The feature auto-advance side effect must be appended
  after the task state write succeeds and after unblocking side effects, not before.
  Read the existing side effect ordering in `finish_tool.go` carefully before adding
  the new side effect. Tests in T3 should assert the side effect is present in the
  `finish` response specifically.
- **Affected tasks:** T3

### Risk: Plan auto-advance firing on plan state the gate prevents writing

- **Probability:** Low
- **Impact:** Low
- **Mitigation:** Plan auto-advance (T4) transitions the plan to `done`. The gate
  (T2) also checks plans before `done`. These do not conflict: the gate fires on
  explicit `entity(transition)` calls; auto-advance fires internally and bypasses
  the gate (it has full knowledge that all features are terminal — the guard condition
  IS the gate). Make sure the internal auto-advance call does not route through the
  gated transition path, or that it uses an equivalent of `override: true` internally.
- **Affected tasks:** T2, T4

### Risk: `generateFeatureAttention` signature change breaks callers

- **Probability:** Medium
- **Impact:** Low
- **Mitigation:** T6 extends the `generateFeatureAttention` signature (adding config
  threshold and bug query). Find all call sites before modifying the signature. There
  should be exactly one call site in `synthesiseFeature`. Update it and the tests in
  the same commit.
- **Affected tasks:** T6

### Risk: Health check latency at project-level status

- **Probability:** Low
- **Impact:** Low
- **Mitigation:** `entitySvc.HealthCheck()` is already called by `buildHealthSummary`
  within `synthesiseProject`. T7 reuses the same call — do not call it twice. Cache
  the result of the single `HealthCheck()` call and use it for both the summary counts
  and the attention item injection.
- **Affected tasks:** T7

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001: Feature gate blocks on ready task | Automated test (`TestFeatureGate_BlockedByNonTerminalTask`) | T2 |
| AC-002: Feature gate passes when all done | Automated test (`TestFeatureGate_AllTasksDone`) | T2 |
| AC-003: Mixed terminal states pass gate | Automated test (`TestFeatureGate_MixedTerminalStatuses`) | T2 |
| AC-004: Gate fires on superseded | Automated test (`TestFeatureGate_BlockedOnSuperseded`) | T2 |
| AC-005: Gate fires on cancelled | Automated test (`TestFeatureGate_BlockedOnCancelled`) | T2 |
| AC-006: Plan gate blocks on developing feature | Automated test (`TestPlanGate_BlockedByNonTerminalFeature`) | T2 |
| AC-007: No children passes unconditionally | Automated test (`TestFeatureGate_NoChildren`) | T2 |
| AC-008: Override bypasses gate and records | Automated test (`TestFeatureGate_OverrideBypass`) | T2 |
| AC-009: Storage error permits with warning | Automated test (`TestFeatureGate_StorageErrorPermits`) | T2 |
| AC-010: Last task done advances feature | Automated test (`TestFeatureAutoAdvance_LastTaskDone`) | T3 |
| AC-011: Advance from needs-rework | Automated test (`TestFeatureAutoAdvance_FromNeedsRework`) | T3 |
| AC-012: All not-planned — no advance | Automated test (`TestFeatureAutoAdvance_NoGuardIfAllNotPlanned`) | T3 |
| AC-013: finish triggers advance | Automated test (`TestFinishTriggerAutoAdvance`) | T3 |
| AC-014: No advance when already reviewing | Automated test (`TestFeatureAutoAdvance_DoesNotFireFromReviewing`) | T3 |
| AC-015: Last feature done advances plan | Automated test (`TestPlanAutoAdvance_LastFeatureDone`) | T4 |
| AC-016: All superseded — no plan advance | Automated test (`TestPlanAutoAdvance_NoGuardIfAllSuperseded`) | T4 |
| AC-017: Auto-advance failure is a warning | Automated test (`TestFeatureAutoAdvance_FailureSurfacedAsWarning`) | T3 |
| AC-018: Schema preserves message content | Automated test (`TestAttentionItem_SchemaPreservesMessage`) | T5 |
| AC-019: all_tasks_done item present | Automated test (`TestStatusAttention_AllTasksDone`) | T6 |
| AC-020: Stale reviewing fires over threshold | Automated test (`TestStatusAttention_StaleReviewing_Over`) | T6 |
| AC-021: Stale reviewing absent under threshold | Automated test (`TestStatusAttention_StaleReviewing_Under`) | T6 |
| AC-022: Stale reviewing disabled at 0 | Automated test (`TestStatusAttention_StaleReviewing_Disabled`) | T6 |
| AC-023: Stale reviewing skipped with no updated | Automated test (`TestStatusAttention_StaleReviewing_NoUpdated`) | T6 |
| AC-024: Critical severity bug fires warning | Automated test (`TestStatusAttention_CriticalBug_SeverityCritical`) | T6 |
| AC-025: High priority bug fires warning | Automated test (`TestStatusAttention_CriticalBug_PriorityHigh`) | T6 |
| AC-026: Low/low bug absent | Automated test (`TestStatusAttention_CriticalBug_LowSeverity`) | T6 |
| AC-027: Bug on closed feature absent | Automated test (`TestStatusAttention_CriticalBug_ClosedFeature`) | T6 |
| AC-028: Health findings injected at project level | Automated test (`TestStatusAttention_HealthFindingsInjected`) | T7 |
| AC-029: Health summary counts retained | Automated test (`TestStatusAttention_HealthSummaryRetained`) | T7 |
| AC-030: Backward compatibility of message field | Automated test (`TestAttentionItem_BackwardCompatibility`) | T5 |