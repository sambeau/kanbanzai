# Review: Lifecycle Integrity and Proactive Status

> Review of FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Plan: P19-workflow-lifecycle-integrity
> Specification: work/spec/lifecycle-integrity.md
> Branch: feature/FEAT-01KN83QN0VAFG-lifecycle-integrity-proactive-status
> Review cycle: 1
> Reviewer: orchestrator
> Date: 2026-04-04

---

## Scope

Files reviewed against the specification and quality standards:

- `internal/config/config.go`
- `internal/mcp/entity_tool.go`
- `internal/mcp/finish_tool.go`
- `internal/mcp/server.go`
- `internal/mcp/sideeffect.go`
- `internal/mcp/status_tool.go`
- `internal/mcp/status_tool_test.go`
- `internal/service/entity_children.go`
- `internal/service/entity_children_test.go`

All tests pass (`go test ./internal/service/... ./internal/mcp/...`).

---

## Summary

The implementation is substantially complete and well-structured. All three pillars are
present: lifecycle gates (Pillar A), auto-advance (Pillar B), and structured attention
items with stale-reviewing detection and bug warnings (Pillar C). The helper functions
in `entity_children.go` are clean, correctly isolated, and well-commented.

Three defects were found — two of them affecting the response contract promised by the
spec, one affecting config defaulting. Several test coverage gaps exist relative to the
Verification Plan's named test functions. One improvement around planDashboard health
injection is noted.

---

## Findings

### [defect] REQ-007: Gate query storage error is logged but not returned in the response

**Location:** `internal/mcp/entity_tool.go`
- Feature gate: lines ~676–679 (`if gateErr != nil { log.Printf(...) }`)
- Plan gate: lines ~636–639 (`if gateErr != nil { log.Printf(...) }`)

REQ-007 states: "The failure MUST be surfaced as a warning in the transition response."
Both the feature and plan gate error paths call `log.Printf` to record the issue but
return a response with no warning field — only `{"entity": {...}}`. Agents consuming
the response have no way to know the child-state check was skipped due to a storage
fault.

AC-009 (`TestFeatureGate_StorageErrorPermits`) is listed in the Verification Plan but
does not exist in the test suite (see test coverage finding below), so this behaviour
is currently untested.

The fix is to push a side effect or include a `warnings` field in the response when
`gateErr != nil`. Using the side-effect mechanism (consistent with how auto-advance
failures are surfaced) would be idiomatic:

```go
PushSideEffect(ctx, SideEffect{
    Type:    SideEffectStatusTransition,
    Trigger: fmt.Sprintf("child-state gate query failed for %s: %v; transition permitted", entityID, gateErr),
})
```

Severity: **defect** — response contract specified by REQ-007 is not met.

---

### [defect] REQ-023: `StaleReviewingDays` default (7) not applied for existing config files

**Location:** `internal/config/config.go`, `LoadFrom` function (lines ~315–349)

REQ-023 states: "The key MUST be optional in `config.yaml`; absence is treated as the
default [7]." When an existing `config.yaml` file omits the `lifecycle:` key, YAML
unmarshalling zero-values `cfg.Lifecycle.StaleReviewingDays` to `0`. Since there is no
`mergeLifecycleDefaults()` call in `LoadFrom` (analogous to the existing
`mergePhase3Defaults()`, `mergePhase4aDefaults()`, `mergePhase4bDefaults()` calls), the
stale reviewing check is silently disabled for any project whose config predates this
feature.

`DefaultConfig()` correctly sets `StaleReviewingDays: 7`, but this path is only taken
for brand-new projects. Pre-existing projects receive `0`.

Note: There is a known design constraint with `int` fields — a zero value cannot
distinguish "explicitly disabled" from "absent." This is the same pattern used elsewhere
in the config. The fix is to add the lifecycle merge call to `LoadFrom`:

```go
cfg.mergeLifecycleDefaults()
```

with a corresponding method:

```go
func (c *Config) mergeLifecycleDefaults() {
    if c.Lifecycle.StaleReviewingDays == 0 {
        c.Lifecycle.StaleReviewingDays = DefaultLifecycleConfig().StaleReviewingDays
    }
}
```

This accepts the same limitation as the existing merge helpers: a user who explicitly
writes `stale_reviewing_days: 0` in their config to disable the check will have it
overridden to 7. If the explicit-disable case must be preserved, the field type should
be changed to `*int`.

Severity: **defect** — REQ-023 is not satisfied for existing project configs.

---

### [defect] REQ-006: Override record not written for `superseded`/`cancelled` feature terminal transitions

**Location:** `internal/mcp/entity_tool.go`, feature child-state gate section
(lines ~673–693) and phase2 gate section (lines ~697–870)

REQ-006 states: "When bypassed, the override and reason MUST be recorded permanently on
the entity as an override record."

The child-state gate is controlled by `!override` — when `override: true`, the entire
gate block is skipped with no record written. The implementation relies on the phase2
gate (`isPhase2Transition`) to record the override. This works partially for
`developing → done` because `done` is in `phase2Statuses`, so the phase2 gate fires and
records an override when its own gate is unsatisfied.

However, `superseded` and `cancelled` are **not** in `phase2Statuses`, so
`isPhase2Transition(from, "superseded")` returns `false`. The phase2 gate block is
skipped entirely. When a user bypasses the child-state gate with `override: true` on a
`superseded` or `cancelled` transition, no override record is written.

A secondary gap exists even for `done`: if the phase2 document gate happens to be
satisfied (all required docs present), the override branch is not entered and the
child-state bypass goes unrecorded.

The fix is to explicitly record an override for the child-state gate bypass before
skipping it, independent of the phase2 gate:

```go
if entityType == "feature" && isFeatureTerminalTransition(newStatus) {
    if !override {
        // ... existing blocking logic
    } else if strings.TrimSpace(overrideReason) != "" {
        // Record override for child-state gate bypass (REQ-006).
        // Deferred: full feature load happens in the phase2 gate below;
        // here we write a lightweight record so the bypass is never silent.
        _ = entitySvc.AppendOverrideRecord(entityID, currentStatus, newStatus, overrideReason)
    }
}
```

Severity: **defect** — REQ-006 is unmet for two of the three terminal statuses and for
`done` when docs are satisfied.

---

### [gap] Verification Plan test functions do not exist

**Location:** `internal/service/entity_children_test.go`, `internal/mcp/status_tool_test.go`

The Verification Plan in the specification names 30 test functions across AC-001 to
AC-030. The following required test functions are absent:

**Pillar A gate tests (should be in entity_tool or an entity_tool_test.go):**
- `TestFeatureGate_BlockedByNonTerminalTask` (AC-001)
- `TestFeatureGate_AllTasksDone` (AC-002)
- `TestFeatureGate_MixedTerminalStatuses` (AC-003)
- `TestFeatureGate_BlockedOnSuperseded` (AC-004)
- `TestFeatureGate_BlockedOnCancelled` (AC-005)
- `TestPlanGate_BlockedByNonTerminalFeature` (AC-006)
- `TestFeatureGate_NoChildren` (AC-007)
- `TestFeatureGate_OverrideBypass` (AC-008)
- `TestFeatureGate_StorageErrorPermits` (AC-009)

**Pillar B auto-advance integration tests:**
- `TestFeatureAutoAdvance_LastTaskDone` (AC-010) — the service test
  `TestMaybeAutoAdvanceFeature_DevelopingAllTasksDone` covers the helper, but
  not the full MCP handler side effect output
- `TestFeatureAutoAdvance_FromNeedsRework` (AC-011) — same caveat
- `TestFeatureAutoAdvance_NoGuardIfAllNotPlanned` (AC-012) — same caveat
- `TestFinishTriggerAutoAdvance` (AC-013) — absent entirely; no test verifies
  that the `finish` tool surfaces the feature auto-advance as a side effect
- `TestFeatureAutoAdvance_DoesNotFireFromReviewing` (AC-014) — same caveat
- `TestPlanAutoAdvance_LastFeatureDone` (AC-015) — same caveat
- `TestPlanAutoAdvance_NoGuardIfAllSuperseded` (AC-016) — same caveat
- `TestFeatureAutoAdvance_FailureSurfacedAsWarning` (AC-017) — absent; no test
  covers the warning side effect when auto-advance fails

What exists in `entity_children_test.go` tests the service-layer helpers
(`MaybeAutoAdvanceFeature`, `MaybeAutoAdvancePlan`, `CountNonTerminalTasks`, etc.)
in isolation. The full handler-level behavior — gate rejection response shape,
override recording, side-effect content, response field names — is untested.

Severity: **gap** — 17 of 30 specified test functions are absent; handler-level gate
and auto-advance behavior is unverified.

---

### [gap] AC-028 test does not inject health findings into the scenario

**Location:** `internal/mcp/status_tool_test.go`,
`TestProjectStatus_HealthFindingsInjected` (lines ~1571–1602)

AC-028 requires: "Given the entity health check returns two warnings for a specific
feature ID, when `status` is called with no `id` argument, then the attention array
MUST contain two items derived from those health findings, each with `entity_id` set to
the relevant feature ID."

The test creates a healthy project and asserts that **no** `health_error` items appear —
i.e. it only tests the clean-state path. It does not inject a health finding (e.g. an
orphaned task with an invalid `parent_feature`) and then verify that warnings appear in
the `attention` array as `health_warning` items with `entity_id` populated.

The positive path of REQ-021 (health findings ARE injected) is therefore not verified
by any test.

Severity: **gap** — AC-028 acceptance criterion is not tested.

---

### [improvement] REQ-020: Health findings not injected into `planDashboard` attention

**Location:** `internal/mcp/status_tool.go`, `synthesisePlan` (lines ~465–611)

REQ-020 states: "Health check findings that are actionable MUST additionally be surfaced
as `AttentionItem` entries in the `attention` array of the same response object." Both
`projectOverview` and `planDashboard` carry a `health` field (the compact
`{errors: N, warnings: N}` summary). The phrase "same response object" implies findings
should appear in both responses.

The implementation injects findings only in `synthesiseProject`. `synthesisePlan` calls
`buildHealthSummary` and populates `Health`, but never appends findings to `attention`.

Note: REQ-021 specifically scopes the health check run to "project-level `status` call
(no `id` argument)", which could be read as limiting injection to project scope. The
requirement is ambiguous. However, REQ-020's "same response object" language — combined
with both response types having a `health` field — suggests the intent was to inject in
both.

This finding is raised as an improvement rather than a defect given the ambiguity, but
it warrants a design clarification.

Severity: **improvement** — planDashboard attention may be incomplete per REQ-020.

---

### [nit] REQ-010 references `complete_task`, which is a removed legacy tool

**Location:** `work/spec/lifecycle-integrity.md`, REQ-010

REQ-010 lists three trigger operations for feature auto-advance: `entity(action:
transition)`, `finish`, and `complete_task`. The `complete_task` tool is explicitly
listed as a legacy tool in `TestServer_ListTools_NoLegacyTools` and is not present in
the MCP surface. `finish` is its replacement.

The implementation is correct — both `entity(action: transition)` and `finish` trigger
auto-advance. The spec simply retains a stale reference. The Verification Plan's
AC-013 (`TestFinishTriggerAutoAdvance`) tests the `finish` trigger, not `complete_task`.

Severity: **nit** — spec documentation inaccuracy; no implementation change needed.

---

### [nit] Bug open-status check includes undocumented `"closed"` state

**Location:** `internal/mcp/status_tool.go`, `synthesiseFeature` (lines ~799–808)

REQ-026 defines closed bug statuses as: `done`, `not-planned`, `duplicate`, `wont-fix`.
The implementation adds `"closed"` to this list:

```go
case "done", "closed", "not-planned", "duplicate", "wont-fix":
    continue
```

This is a reasonable safety net, but `"closed"` is not documented in REQ-026. If
`"closed"` is a valid bug lifecycle status, it should be added to the specification.
If it is not a reachable status, the extra case is harmless dead code.

Severity: **nit** — undocumented deviation from REQ-026; requires either a spec update
or removal of the `"closed"` case.

---

## Specification Completeness

| Requirement | Status | Notes |
|-------------|--------|-------|
| REQ-001 | ✅ Implemented | Feature terminal gate present |
| REQ-002 | ✅ Implemented | Error message includes count |
| REQ-003 | ✅ Implemented | Plan terminal gate present |
| REQ-004 | ✅ Implemented | Error message includes count |
| REQ-005 | ✅ Implemented | Zero-child case passes gate |
| REQ-006 | ⚠️ Partial | Override for `superseded`/`cancelled` not recorded — see defect #3 |
| REQ-007 | ❌ Defect | Storage error not surfaced in response — see defect #1 |
| REQ-008 | ✅ Implemented | Auto-advance from developing/needs-rework |
| REQ-009 | ✅ Implemented | All-not-planned guard present |
| REQ-010 | ✅ Implemented | Both `entity(transition)` and `finish` trigger auto-advance |
| REQ-011 | ✅ Implemented | Side effect includes from/to status |
| REQ-012 | ✅ Implemented | Auto-advance failure is a warning, not an error |
| REQ-013 | ✅ Implemented | Plan auto-advance chains active→reviewing→done |
| REQ-014 | ✅ Implemented | All-superseded guard present |
| REQ-015 | ✅ Implemented | Plan side effect recorded |
| REQ-016 | ✅ Implemented | Plan auto-advance failure is a warning |
| REQ-017 | ✅ Implemented | `AttentionItem` struct with all required fields |
| REQ-018 | ✅ Implemented | Existing message text preserved verbatim |
| REQ-019 | ✅ Implemented | All type values from registry used consistently |
| REQ-020 | ⚠️ Partial | planDashboard does not inject health findings — see improvement |
| REQ-021 | ✅ Implemented | Project-level status injects health findings |
| REQ-022 | ✅ Implemented | `stale_reviewing` item emitted when threshold exceeded |
| REQ-023 | ❌ Defect | Default (7) not applied for existing configs — see defect #2 |
| REQ-024 | ✅ Implemented | Zero-value `updated` skips stale check |
| REQ-025 | ✅ Implemented | `origin_feature` used correctly |
| REQ-026 | ✅ Implemented (with nit) | `"closed"` added beyond spec definition |
| REQ-027 | ✅ Implemented | Bug warnings are attention items only |
| REQ-NF-001 | ✅ Implemented | Single list scan, no N+1 queries |
| REQ-NF-002 | ✅ Implemented | `message` field preserves legacy string content |
| REQ-NF-003 | ✅ Implemented | All transitions are synchronous |

---

## Test Coverage Assessment

| AC | Test exists | Notes |
|----|-------------|-------|
| AC-001 | ❌ | No handler-level gate test |
| AC-002 | ❌ | No handler-level gate test |
| AC-003 | ❌ | No handler-level gate test |
| AC-004 | ❌ | No handler-level gate test |
| AC-005 | ❌ | No handler-level gate test |
| AC-006 | ❌ | No handler-level plan gate test |
| AC-007 | ❌ | No handler-level test |
| AC-008 | ❌ | No override test; override defect untested |
| AC-009 | ❌ | Storage error path untested |
| AC-010 | ⚠️ Partial | Service helper tested; MCP side effect not verified |
| AC-011 | ⚠️ Partial | Service helper tested; handler not |
| AC-012 | ⚠️ Partial | Service helper tested; handler not |
| AC-013 | ❌ | finish-triggered auto-advance not tested |
| AC-014 | ⚠️ Partial | Service helper tested; handler not |
| AC-015 | ⚠️ Partial | Service helper tested; handler not |
| AC-016 | ⚠️ Partial | Service helper tested; handler not |
| AC-017 | ❌ | Auto-advance failure warning side effect not tested |
| AC-018 | ✅ | `TestAttentionItem_SchemaPreservesMessage` equivalent present |
| AC-019 | ✅ | `TestFeatureAttention_AllTasksDone_Developing` |
| AC-020 | ✅ | `TestFeatureAttention_StaleReviewing_OverThreshold` |
| AC-021 | ✅ | `TestFeatureAttention_StaleReviewing_UnderThreshold` |
| AC-022 | ✅ | `TestFeatureAttention_StaleReviewing_DisabledThreshold` |
| AC-023 | ✅ | `TestFeatureAttention_StaleReviewing_ZeroUpdated` |
| AC-024 | ✅ | `TestFeatureAttention_OpenCriticalBug_CriticalSeverity` |
| AC-025 | ✅ | `TestFeatureAttention_OpenCriticalBug_HighPriority` |
| AC-026 | ✅ | `TestFeatureAttention_OpenCriticalBug_LowSeverityLowPriority` |
| AC-027 | ✅ | `TestFeatureAttention_OpenCriticalBug_FeatureDone` |
| AC-028 | ⚠️ Partial | Only clean-state path tested; positive injection not verified |
| AC-029 | ✅ | `TestProjectStatus_HealthSummaryAlwaysPresent` |
| AC-030 | ✅ | `TestStatusTool_AttentionItemsGenerated` covers message field |

---

## Code Quality Notes

The implementation is generally clean and idiomatic Go.

- `entity_children.go` is well-structured with clear function boundaries and
  appropriate godoc. The `isFeatureEffectivelyTerminal` unexported helper is correctly
  explained (why `done` must be explicit alongside `validate.IsTerminalState`).
- The `MaybeAutoAdvancePlan` chain through `reviewing → done` is clearly commented and
  matches the expected plan lifecycle.
- The `generateFeatureAttention` function in `status_tool.go` is long but logically
  sequential; each attention type is isolated and straightforward to follow.
- The `AttentionItem` struct is well-placed in `status_tool.go` alongside the response
  types that use it.
- The decision to test Pillar C at the `generateXxxAttention` function level (rather
  than full integration) is a reasonable trade-off and yields fast, focused tests.

---

## Required Actions Before Merge

1. **Fix defect #1** — Surface gate query storage error in the transition response
   (REQ-007). Using a side effect is the idiomatic approach.

2. **Fix defect #2** — Add `mergeLifecycleDefaults()` to `LoadFrom` in `config.go`
   so existing project configs default `StaleReviewingDays` to 7 (REQ-023).

3. **Fix defect #3** — Write the override record when the child-state gate is bypassed
   on `superseded` and `cancelled` feature transitions (REQ-006). The `done` path also
   needs explicit handling independent of the phase2 gate.

4. **Address test gap** — Add at minimum:
   - Handler-level tests for the gate rejection response shape (AC-001, AC-002, AC-006)
   - `TestFinishTriggerAutoAdvance` verifying finish side effect for auto-advance
     (AC-013)
   - `TestFeatureGate_StorageErrorPermits` verifying the response contract for
     storage errors (AC-009)
   - A positive health findings injection test for AC-028

The improvement (planDashboard health injection) and the two nits do not block merge
but should be tracked.