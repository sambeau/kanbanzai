# Specification: Completion Detection and Close-Out Procedures

**Status:** Draft
**Feature:** FEAT-01KN07T66DZ68 (feature-completion-workflow)
**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` — Features 1 and 2
**Date:** 2026-04-02

---

## Problem Statement

When the last task in a feature completes via `finish()`, nothing signals that the parent
feature is ready to advance. Features sit in `developing` with all tasks done, plans stay
`active` after every feature finishes, and no skill describes how to close out a feature.
The `health` check for feature-child consistency flags features in early states (`proposed`,
`designing`, `specifying`, `dev-planning`) but misses `developing` and `needs-rework` —
the states where the problem actually occurs. There is no plan-level equivalent at all.

This specification covers three concerns from the design:

1. **Completion detection** — attention items in `status` that surface when all children
   of a feature or plan are finished.
2. **Health check extensions** — consistency warnings that catch the same conditions in
   the `health` tool.
3. **Skill updates** — close-out procedures added to three skill files so agents and
   humans know what to do when the detection fires.

### In Scope

- Feature-level attention items in `status` (feature detail and plan dashboard).
- Plan-level attention items in `status` (plan dashboard and project overview).
- Stale-detection escalation for features stuck in `developing`.
- Extension of `CheckFeatureChildConsistency` to include `developing` and `needs-rework`.
- New `CheckPlanChildConsistency` health rule.
- Close-out procedure additions to `orchestrate-development`, `kanbanzai-agents`, and
  `kanbanzai-workflow` skills.

### Out of Scope

- Automatic lifecycle transitions (design Decision 1 rejects auto-advance).
- Composite close-out tools (design Decision 2 favours skill procedures).
- Direct-to-main workflow changes (design Feature 3, separate specification).
- Document inheritance for gap checks (design Feature 4, separate specification).
- Task crash recovery (design Feature 5, separate specification).

---

## Requirements

### Functional Requirements

**Feature-Level Completion Detection**

**REQ-001:** When the `status` tool synthesises a feature detail view, and the feature is
in `developing` or `needs-rework`, and every child task is in a terminal state (`done`,
`not-planned`, or `duplicate`), the attention items list MUST include an item of the form:
`"{display_id} has {N}/{N} tasks done — ready to advance to reviewing"` where `{display_id}`
is the feature's display ID and `{N}` is the total number of child tasks.

**REQ-002:** The attention item described in REQ-001 MUST NOT appear when the feature has
zero child tasks.

**REQ-003:** The attention item described in REQ-001 MUST NOT appear when the feature is
in any status other than `developing` or `needs-rework` (e.g. `proposed`, `designing`,
`specifying`, `dev-planning`, `reviewing`, `done`, `superseded`, `cancelled`).

**REQ-004:** When a feature qualifies for the attention item in REQ-001, and the feature
has been in the `developing` state for more than 48 hours (measured from the feature's
`updated` timestamp to the current time), the attention item text MUST be prefixed with
`"⚠️ STALE: "`, producing: `"⚠️ STALE: {display_id} has {N}/{N} tasks done — ready to advance to reviewing"`.

**REQ-005:** The stale prefix in REQ-004 MUST only apply to features in `developing`. It
MUST NOT apply to features in `needs-rework`, because entering `needs-rework` is itself a
recent action that resets staleness expectations.

**Plan-Level Completion Detection**

**REQ-006:** When the `status` tool synthesises a plan dashboard, and the plan is not in
`done` status, and every child feature is in a finished state (`done`, `superseded`, or
`cancelled`), and the plan has at least one child feature, the attention items list MUST
include an item of the form:
`"Plan {display_id} has all {N} features done — ready to close"` where `{display_id}` is
the plan's display ID and `{N}` is the total number of child features.

**REQ-007:** The attention item described in REQ-006 MUST NOT appear when the plan has zero
child features.

**REQ-008:** The attention item described in REQ-006 MUST NOT appear when the plan is
already in `done` status.

**REQ-009:** When the `status` tool synthesises the project overview, and any plan qualifies
for the attention item in REQ-006, that attention item MUST also appear in the project-level
attention items list.

**Health Check: Feature-Child Consistency Extension**

**REQ-010:** The `CheckFeatureChildConsistency` health rule MUST flag a warning when all
child tasks of a feature are in terminal state and the feature is in `developing` or
`needs-rework`, in addition to the existing early-state checks (`proposed`, `designing`,
`specifying`, `dev-planning`).

**REQ-011:** The warning message for the extended check in REQ-010 MUST follow the existing
message format: `"feature {id} has all {N} child task(s) in terminal state but feature is {status}"`.

**Health Check: Plan-Child Consistency**

**REQ-012:** A new `CheckPlanChildConsistency` health rule MUST warn when all child features
of a plan are in a finished state (`done`, `superseded`, or `cancelled`) and the plan is not
in `done` status.

**REQ-013:** The warning in REQ-012 MUST NOT fire when a plan has zero child features.

**REQ-014:** The warning message for REQ-012 MUST follow the format:
`"plan {id} has all {N} child feature(s) in finished state but plan is {status}"`.

**REQ-015:** `CheckPlanChildConsistency` MUST also warn when a plan is in `done` status
but has child features that are not in a finished state (`done`, `superseded`, or
`cancelled`), following the format:
`"plan {id} is done but has {M} non-finished child feature(s)"`.

**Skill Updates: Close-Out Procedures**

**REQ-016:** The `orchestrate-development` skill (`.kbz/skills/orchestrate-development/SKILL.md`)
MUST include a new phase (Phase 6: Close-Out) after the existing final phase, with a
procedure that covers: (a) verify all tasks are in terminal state, (b) transition the
feature to `reviewing` or `done`, (c) if a worktree exists, create a PR and merge,
(d) record a feature completion summary.

**REQ-017:** The `orchestrate-development` skill MUST include a checklist item:
`"- [ ] Feature advanced beyond developing"`.

**REQ-018:** The `kanbanzai-agents` skill (`.agents/skills/kanbanzai-agents/SKILL.md`) MUST
include a "Feature Completion" section after the existing "Finishing Tasks" section, covering
feature transition, PR creation, merge, and worktree cleanup. It MUST reference the
`orchestrate-development` skill for the full procedure.

**REQ-019:** The `kanbanzai-workflow` skill (`.agents/skills/kanbanzai-workflow/SKILL.md`)
MUST include a trigger that ties the close-out procedure to the `status` attention item,
stating that the close-out checklist applies when `status` shows an attention item indicating
all tasks are done.

### Non-Functional Requirements

**REQ-020:** The attention item generation for REQ-001 and REQ-006 MUST NOT increase the
time complexity of `status` synthesis beyond the existing O(features × tasks) traversal.
No additional storage reads beyond what `status` already performs are permitted.

**REQ-021:** The 48-hour threshold in REQ-004 MUST be derived from the feature entity's
existing `updated` timestamp field. No new timestamp fields are required.

---

## Constraints

1. **No automatic transitions.** Detection MUST surface attention items and health warnings
   only. It MUST NOT automatically advance features or plans through their lifecycle
   (design Decision 1).

2. **No new tools.** Close-out is a documented skill procedure using existing tools
   (`entity`, `status`, `pr`, `merge`). No composite tool is introduced (design Decision 2).

3. **Existing message format.** Health check messages MUST follow the established format
   used by `CheckFeatureChildConsistency` (severity: warning, entity ID, descriptive message).

4. **Attention item cap.** Attention items are subject to the existing `maxAttentionItems`
   cap. Completion items MUST NOT bypass this cap.

5. **Terminal state definitions are authoritative.** Task terminal states are defined by
   `validate.IsTerminalState` (`done`, `not-planned`, `duplicate`). Feature finished states
   for plan-level checks are `done`, `superseded`, `cancelled` — matching the existing
   `featureTerminalOrDone` set in `entity_consistency.go`. These definitions MUST NOT be
   modified.

6. **Backward compatibility.** Existing attention items and health check results MUST NOT
   change for features/plans that do not match the new conditions.

---

## Acceptance Criteria

### Feature-Level Completion Detection

**AC-001** (REQ-001, REQ-002)
Given a feature in `developing` with 5 child tasks all in `done` status,
when the `status` tool generates the feature detail view,
then the attention items list contains `"FEAT-xxx has 5/5 tasks done — ready to advance to reviewing"`.

**AC-002** (REQ-001)
Given a feature in `developing` with 4 tasks in `done` and 1 in `not-planned`,
when the `status` tool generates the feature detail view,
then the attention items list contains `"FEAT-xxx has 5/5 tasks done — ready to advance to reviewing"`
(because `not-planned` is a terminal state and all 5 tasks are accounted for).

**AC-003** (REQ-001)
Given a feature in `needs-rework` with 3 child tasks all in terminal state,
when the `status` tool generates the feature detail view,
then the attention items list contains `"FEAT-xxx has 3/3 tasks done — ready to advance to reviewing"`.

**AC-004** (REQ-003)
Given a feature in `reviewing` with all child tasks in terminal state,
when the `status` tool generates the feature detail view,
then the attention items list does NOT contain a "ready to advance" item.

**AC-005** (REQ-002)
Given a feature in `developing` with zero child tasks,
when the `status` tool generates the feature detail view,
then the attention items list does NOT contain a "ready to advance" item.

**AC-006** (REQ-001)
Given a feature in `developing` with 3 tasks in `done` and 1 in `active`,
when the `status` tool generates the feature detail view,
then the attention items list does NOT contain a "ready to advance" item.

**AC-007** (REQ-004, REQ-005)
Given a feature in `developing` with all 4 tasks in terminal state,
and the feature's `updated` timestamp is more than 48 hours ago,
when the `status` tool generates the feature detail view,
then the attention item text starts with `"⚠️ STALE: "`.

**AC-008** (REQ-004)
Given a feature in `developing` with all tasks terminal,
and the feature's `updated` timestamp is less than 48 hours ago,
when the `status` tool generates the feature detail view,
then the attention item text does NOT start with `"⚠️ STALE: "`.

**AC-009** (REQ-005)
Given a feature in `needs-rework` with all tasks terminal,
and the feature's `updated` timestamp is more than 48 hours ago,
when the `status` tool generates the feature detail view,
then the attention item text does NOT start with `"⚠️ STALE: "`.

### Plan-Level Completion Detection

**AC-010** (REQ-006, REQ-007)
Given a plan in `active` with 3 child features all in `done` status,
when the `status` tool generates the plan dashboard,
then the attention items list contains `"Plan Pxx has all 3 features done — ready to close"`.

**AC-011** (REQ-006)
Given a plan in `active` with 2 features in `done` and 1 in `cancelled`,
when the `status` tool generates the plan dashboard,
then the attention items list contains `"Plan Pxx has all 3 features done — ready to close"`
(because `cancelled` is a finished state).

**AC-012** (REQ-008)
Given a plan in `done` with all features in `done`,
when the `status` tool generates the plan dashboard,
then the attention items list does NOT contain a "ready to close" item.

**AC-013** (REQ-006)
Given a plan in `active` with 2 features in `done` and 1 in `developing`,
when the `status` tool generates the plan dashboard,
then the attention items list does NOT contain a "ready to close" item.

**AC-014** (REQ-007)
Given a plan in `active` with zero child features,
when the `status` tool generates the plan dashboard,
then the attention items list does NOT contain a "ready to close" item.

**AC-015** (REQ-009)
Given a plan that qualifies for the "ready to close" attention item,
when the `status` tool generates the project overview,
then the project-level attention items list also contains the "ready to close" item for that plan.

### Health Check: Feature-Child Consistency Extension

**AC-016** (REQ-010, REQ-011)
Given a feature in `developing` with all 3 child tasks in terminal state,
when `CheckFeatureChildConsistency` runs,
then it returns a warning: `"feature FEAT-xxx has all 3 child task(s) in terminal state but feature is developing"`.

**AC-017** (REQ-010, REQ-011)
Given a feature in `needs-rework` with all child tasks in terminal state,
when `CheckFeatureChildConsistency` runs,
then it returns a warning with the same format, ending `"but feature is needs-rework"`.

**AC-018** (REQ-010)
Given a feature in `developing` with 2 tasks in `done` and 1 in `active`,
when `CheckFeatureChildConsistency` runs,
then it does NOT return a warning about all children being terminal.

**AC-019** (REQ-010)
Given a feature in `reviewing` with all child tasks in terminal state,
when `CheckFeatureChildConsistency` runs,
then it does NOT return a warning about all children being terminal
(because `reviewing` is neither an early state nor a flagged developing/rework state — it
is the expected next step).

### Health Check: Plan-Child Consistency

**AC-020** (REQ-012, REQ-014)
Given a plan in `active` with all 4 child features in finished state (`done`, `superseded`, or `cancelled`),
when `CheckPlanChildConsistency` runs,
then it returns a warning: `"plan Pxx has all 4 child feature(s) in finished state but plan is active"`.

**AC-021** (REQ-013)
Given a plan in `active` with zero child features,
when `CheckPlanChildConsistency` runs,
then it does NOT return a warning.

**AC-022** (REQ-012)
Given a plan in `active` with 3 features in `done` and 1 in `developing`,
when `CheckPlanChildConsistency` runs,
then it does NOT return a "all finished" warning.

**AC-023** (REQ-015)
Given a plan in `done` with 2 features in `done` and 1 in `developing`,
when `CheckPlanChildConsistency` runs,
then it returns a warning: `"plan Pxx is done but has 1 non-finished child feature(s)"`.

**AC-024** (REQ-012)
Given a plan in `done` with all features in finished state,
when `CheckPlanChildConsistency` runs,
then it does NOT return any warning.

### Skill Updates

**AC-025** (REQ-016, REQ-017)
Given the `orchestrate-development` skill file,
then it contains a "Phase 6: Close-Out" section that describes: verifying all tasks are
terminal, transitioning the feature, creating a PR if a worktree exists, and recording a
completion summary; and it contains a checklist item `"- [ ] Feature advanced beyond developing"`.

**AC-026** (REQ-018)
Given the `kanbanzai-agents` skill file,
then it contains a "Feature Completion" section after "Finishing Tasks" that covers feature
transition, PR creation, merge, and worktree cleanup, and references the
`orchestrate-development` skill.

**AC-027** (REQ-019)
Given the `kanbanzai-workflow` skill file,
then it contains a trigger statement that ties the close-out procedure to the `status`
attention item for all-tasks-done.

---

## Verification Plan

| AC | Test Method | Location |
|----|-------------|----------|
| AC-001 | Unit test: `generateFeatureAttention` with all-terminal tasks and `developing` status | `internal/mcp/status_tool_test.go` |
| AC-002 | Unit test: mixed terminal states (`done` + `not-planned`) still trigger the item | `internal/mcp/status_tool_test.go` |
| AC-003 | Unit test: `needs-rework` status triggers the item | `internal/mcp/status_tool_test.go` |
| AC-004 | Unit test: `reviewing` status does NOT trigger the item | `internal/mcp/status_tool_test.go` |
| AC-005 | Unit test: zero tasks does NOT trigger the item | `internal/mcp/status_tool_test.go` |
| AC-006 | Unit test: non-terminal task present does NOT trigger the item | `internal/mcp/status_tool_test.go` |
| AC-007 | Unit test: `updated` >48h ago produces `⚠️ STALE:` prefix | `internal/mcp/status_tool_test.go` |
| AC-008 | Unit test: `updated` <48h ago does NOT produce stale prefix | `internal/mcp/status_tool_test.go` |
| AC-009 | Unit test: `needs-rework` >48h does NOT produce stale prefix | `internal/mcp/status_tool_test.go` |
| AC-010 | Unit test: plan attention with all features finished | `internal/mcp/status_tool_test.go` |
| AC-011 | Unit test: plan attention with mixed finished states (`done` + `cancelled`) | `internal/mcp/status_tool_test.go` |
| AC-012 | Unit test: plan in `done` does NOT trigger the item | `internal/mcp/status_tool_test.go` |
| AC-013 | Unit test: plan with non-finished feature does NOT trigger | `internal/mcp/status_tool_test.go` |
| AC-014 | Unit test: plan with zero features does NOT trigger | `internal/mcp/status_tool_test.go` |
| AC-015 | Integration test: project overview includes plan-level completion item | `internal/mcp/status_tool_test.go` |
| AC-016 | Unit test: `CheckFeatureChildConsistency` flags `developing` | `internal/health/entity_consistency_test.go` |
| AC-017 | Unit test: `CheckFeatureChildConsistency` flags `needs-rework` | `internal/health/entity_consistency_test.go` |
| AC-018 | Unit test: non-terminal task present, no warning | `internal/health/entity_consistency_test.go` |
| AC-019 | Unit test: `reviewing` not flagged | `internal/health/entity_consistency_test.go` |
| AC-020 | Unit test: `CheckPlanChildConsistency` flags all-finished plan | `internal/health/entity_consistency_test.go` |
| AC-021 | Unit test: zero features, no warning | `internal/health/entity_consistency_test.go` |
| AC-022 | Unit test: non-finished feature present, no warning | `internal/health/entity_consistency_test.go` |
| AC-023 | Unit test: plan `done` with non-finished children warns | `internal/health/entity_consistency_test.go` |
| AC-024 | Unit test: plan `done` with all finished, no warning | `internal/health/entity_consistency_test.go` |
| AC-025 | Manual review: skill file contains Phase 6 and checklist item | `.kbz/skills/orchestrate-development/SKILL.md` |
| AC-026 | Manual review: skill file contains Feature Completion section | `.agents/skills/kanbanzai-agents/SKILL.md` |
| AC-027 | Manual review: skill file contains trigger statement | `.agents/skills/kanbanzai-workflow/SKILL.md` |