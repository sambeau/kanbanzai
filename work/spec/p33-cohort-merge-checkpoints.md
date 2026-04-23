| Field  | Value                                         |
|--------|-----------------------------------------------|
| Date   | 2026-04-23                                    |
| Status | Draft                                         |
| Author | spec-author                                   |

# Specification: Cohort-Based Merge Checkpoints (P33)

## Related Work

### Prior documents consulted

| Document | Type | Relationship |
|----------|------|--------------|
| `work/research/cohort-merge-checkpoints.md` | Research | Primary source; establishes the evidence base (three P3 conflict-fix commits, P27 34-worktree timeout) and the three-change recommendation this spec formalises |
| `work/design/p33-cohort-merge-checkpoints.md` | Design | Parent design document; all requirements in this spec trace to decisions and enhancements defined there |
| `work/design/p25-orchestration-docs.md` | Design | P25 orchestration skill changes; constrains the phase numbering and editorial style of `orchestrate-development/SKILL.md` |
| `work/design/p29-state-store-read-performance.md` | Design | P29 read-path fix; contextualises the second-order effect of worktree accumulation on entity service performance |
| `work/spec/p25-orchestration-docs.md` | Specification | Prior spec for orchestration skill updates; establishes the pattern for Markdown-only skill change requirements |

### Decisions that constrain this specification

| Decision | Source | Constraint |
|----------|--------|------------|
| Feature-level conflict mode uses `feature_ids` as a mutually exclusive alternative to `task_ids` | Design P33-DEC-001 | The spec must not require or permit mixed `task_ids`+`feature_ids` inputs |
| `drift_days` is informational only — not mapped to a risk level | Design P33-DEC-002 | The spec must not require the server to classify drift as low/medium/high |
| Merge schedule is author-driven, not auto-generated | Design P33-DEC-003 | The spec must not require decompose to produce a merge schedule automatically |
| Decompose warning is non-blocking | Design P33-DEC-004 | The warning must not prevent proposal generation or the apply step |
| Cohort gates are advisory, not server-enforced | Design P33-DEC-005 | The server must not block `worktree(action: create)` or any other tool call based on cohort membership |

### Relationship to prior work

This specification extends the `conflict` tool (last modified in P4/P6 for task-level analysis) with a feature-level input mode. It adds a new section to an existing template file and two skill files that were last updated in P25 and P27 respectively. No prior spec covers feature-level conflict detection or the merge-schedule template section; this spec introduces both as new capabilities. The decompose quality-validation path was last specified in `work/spec/3.0-decomposition-quality-validation.md` (P17); the new warning added here is additive and does not modify existing validation checks.

---

## Overview

This specification defines three additive enhancements to prevent worktree drift in plans with four or more parallel features: (1) a `feature_ids` input mode for the `conflict` tool that performs feature-level file-overlap detection and reports branch age as `drift_days`; (2) a `## Merge Schedule` section added to the dev-plan template and a non-blocking decompose warning when the section is absent in large plans; and (3) Phase 0 cohort-setup guidance and a merge-checkpoint recognition step added to the `orchestrate-development` skill. No entity schema changes, server-enforced gates, or modifications to existing tool APIs are introduced.

---

## Scope

**In scope:**
- Adding `feature_ids` as an optional, mutually exclusive alternative to `task_ids` in the `conflict` tool's input
- Adding a `drift_days` informational field to the conflict result when feature IDs are provided and a worktree exists
- Adding a `FeatureConflictResult` response type (or equivalent named structure) returned when `feature_ids` mode is used
- Emitting a `"no_file_data"` warning per feature when `feature_ids` are provided but the feature's tasks have no `files_planned` values
- Adding a `## Merge Schedule` section to `work/templates/implementation-plan-prompt-template.md`
- Adding a cohort-authoring note to `.kbz/skills/write-dev-plan/SKILL.md`
- Adding a non-blocking warning to `decompose(action: "review")` when a plan has more than 3 features and the dev-plan has no `## Merge Schedule` heading
- Adding Phase 0 (Cohort Setup) and a merge-checkpoint step to `.kbz/skills/orchestrate-development/SKILL.md`

**Out of scope:**
- Changes to the entity data model (`cohort` field or any other new field on feature, task, or plan entities)
- Automatic cohort inference or merge-schedule generation by `decompose`
- Changes to the `merge`, `pr`, or `branch` tools
- Server-enforced gates blocking worktree creation or task dispatch based on cohort membership
- Changes to the `conflict` tool's existing `task_ids` behaviour
- Changes to `worktree`, `next`, `finish`, or any other MCP tool not named above
- Changes to any skill file other than `write-dev-plan/SKILL.md` and `orchestrate-development/SKILL.md`

---

## Functional Requirements

**FR-001 — `feature_ids` input parameter**
The `conflict` tool MUST accept an optional `feature_ids` parameter containing an array of one or more `FEAT-...` IDs. When `feature_ids` is provided, the tool MUST operate in feature-level mode and MUST NOT perform task-level analysis.

**FR-002 — Mutual exclusivity of `task_ids` and `feature_ids`**
If a caller supplies both `task_ids` and `feature_ids` in the same request, the `conflict` tool MUST return an error. The error message MUST state that the two parameters are mutually exclusive.

**FR-003 — Feature file-set aggregation**
When `feature_ids` is provided, the tool MUST resolve each feature ID to its constituent tasks by querying the entity store filtered by `parent_feature`. It MUST aggregate the `files_planned` values across all resolved tasks to produce a per-feature file set used for overlap detection.

**FR-004 — No-file-data warning**
When `feature_ids` is provided and a feature's resolved tasks have no `files_planned` values (all empty or absent), the tool MUST include a `"no_file_data"` warning in that feature's result entry. The tool MUST NOT return an error in this case; results for other features with file data MUST still be computed and returned.

**FR-005 — Feature-level overlap detection**
The tool MUST perform pairwise file-set intersection between all provided features (using the aggregated file sets from FR-003) and report an overlap risk for each pair, using the same risk vocabulary as the existing task-level mode (`safe_to_parallelise`, `serialise`, `checkpoint_required`).

**FR-006 — `drift_days` field**
For each feature in `feature_ids` mode, the tool MUST compute a `drift_days` value: the number of calendar days between the feature's worktree branch creation date and the current date, obtained via the existing `BranchLookup` interface. If no worktree record exists for the feature, the `drift_days` field MUST be omitted from that feature's result (not set to zero). The field MUST be informational only and MUST NOT be mapped to a risk level.

**FR-007 — `FeatureConflictResult` response structure**
When `feature_ids` mode is used, the tool MUST return a structured result that is distinct from the existing task-level result. The feature-level result MUST include, at minimum: the list of feature ID pairs checked, the overlap risk for each pair, and any per-feature warnings (FR-004). It MUST include `drift_days` where available (FR-006).

**FR-008 — Merge Schedule section in dev-plan template**
The file `work/templates/implementation-plan-prompt-template.md` MUST contain a `## Merge Schedule` section. The section MUST include a Markdown table template with columns `Cohort`, `Features`, and `Gate condition`. The section MUST include prose stating that it is required when the plan has more than 3 features, that cohort size should be 3–5 features, and that intra-cohort file overlap should be verified with `conflict(action: "check", feature_ids: [...])` before publishing the dev-plan.

**FR-009 — Cohort authoring note in `write-dev-plan` skill**
The file `.kbz/skills/write-dev-plan/SKILL.md` MUST include a note under its quality-checks step (or equivalent checklist section) stating: when the plan has more than 3 features, add a `## Merge Schedule` section grouping features into cohorts of 3–5, and use `conflict(action: "check", feature_ids: [...])` to verify no intra-cohort file overlap before publishing the dev-plan.

**FR-010 — Decompose warning for missing merge schedule**
`decompose(action: "review")` MUST emit a non-blocking warning when both of the following are true: (a) the parent plan has more than 3 features, and (b) the plan's associated dev-plan document does not contain a `## Merge Schedule` heading. The warning text MUST include the feature count and MUST recommend adding cohort groupings to prevent worktree drift. The warning MUST NOT prevent proposal generation or the `decompose(action: "apply")` step.

**FR-011 — Phase 0 in `orchestrate-development` skill**
The file `.kbz/skills/orchestrate-development/SKILL.md` MUST contain a **Phase 0: Cohort Setup** section positioned before the current Phase 1. Phase 0 MUST specify that it applies only to plans with more than 3 features and MUST include the following steps:
- Read the dev-plan's `## Merge Schedule` block if present; treat it as authoritative if found
- If no merge schedule exists, call `conflict(action: "check", feature_ids: [...])` for all plan features
- Group features into cohorts based on overlap results (no overlap → same cohort; overlap → different cohorts), targeting 3–5 features per cohort
- Record the cohort grouping in the session context
- Create worktrees only for cohort-1 features; defer worktree creation for later cohorts

**FR-012 — Merge-checkpoint step in `orchestrate-development` skill**
The `orchestrate-development` skill's Phase 6 (Close-Out) or equivalent section MUST include a merge-checkpoint recognition step. The step MUST instruct the orchestrator to: after merging all cohort-N features, check whether cohort-N+1 features exist; if so, return to Phase 0 for cohort N+1; and not create cohort-N+1 worktrees until the cohort-N merge checkpoint is confirmed clean (no open feature branches from cohort N).

---

## Non-Functional Requirements

**NFR-001 — No regression on existing `task_ids` behaviour**
The existing `conflict(action: "check", task_ids: [...])` behaviour MUST be unchanged. All existing tests for task-level conflict detection MUST continue to pass without modification.

**NFR-002 — `BranchLookup` interface only for git operations**
The feature-level conflict implementation MUST use only the existing `BranchLookup` interface for any git-level queries (branch file diff, branch creation date). It MUST NOT introduce direct shell calls or new git library invocations outside this interface.

**NFR-003 — Feature-level check uses existing `analyzePair` logic**
The file-overlap detection for feature pairs MUST reuse the existing `analyzePair` function (or its equivalent internal logic) rather than reimplementing overlap detection. The feature-level mode is a new input aggregation layer, not a new analysis engine.

**NFR-004 — Decompose warning is non-blocking**
The warning introduced by FR-010 MUST NOT cause `decompose(action: "review")` or `decompose(action: "apply")` to return an error or refuse to proceed. It MUST appear alongside the proposal, not instead of it.

**NFR-005 — Skill files are the sole documentation change target**
No AGENTS.md, README, or other documentation file other than the two skill files named in FR-009 and FR-011/FR-012 and the one template file named in FR-008 MUST be modified as part of this feature.

**NFR-006 — Test coverage for feature-level conflict**
The new feature-level conflict logic MUST be covered by unit tests that: (a) validate correct aggregation of `files_planned` across tasks; (b) validate that the `"no_file_data"` warning is emitted when no files are present; (c) validate that `drift_days` is populated when a worktree record exists and omitted when it does not; and (d) validate that supplying both `task_ids` and `feature_ids` returns an error.

---

## Acceptance Criteria

**AC-001 (FR-001, FR-002):** Given a `conflict` request containing both `task_ids` and `feature_ids`, when the tool is called, then it returns an error response stating the parameters are mutually exclusive, and no conflict analysis is performed.

**AC-002 (FR-001):** Given a `conflict` request containing only `feature_ids` with two or more valid feature IDs, when the tool is called, then it returns a feature-level result (not a task-level result) and does not error.

**AC-003 (FR-003, FR-005):** Given two features whose constituent tasks share one or more values in `files_planned`, when `conflict(action: "check", feature_ids: [FEAT-A, FEAT-B])` is called, then the result for the FEAT-A/FEAT-B pair reports a non-`safe_to_parallelise` risk.

**AC-004 (FR-003, FR-005):** Given two features whose constituent tasks have no overlapping `files_planned` values, when `conflict(action: "check", feature_ids: [FEAT-A, FEAT-B])` is called, then the result for the FEAT-A/FEAT-B pair reports `safe_to_parallelise`.

**AC-005 (FR-004):** Given a feature ID whose tasks all have empty `files_planned` fields, when `conflict(action: "check", feature_ids: [...])` is called, then the result for that feature includes a `"no_file_data"` warning and the tool does not return an error.

**AC-006 (FR-006):** Given a feature ID whose worktree was created N calendar days ago, when `conflict(action: "check", feature_ids: [...])` is called, then the result for that feature includes a `drift_days` value equal to N (± 1 for clock rounding). The `drift_days` field is not mapped to a risk level in the response.

**AC-007 (FR-006):** Given a feature ID that has no worktree record, when `conflict(action: "check", feature_ids: [...])` is called, then the result for that feature omits the `drift_days` field entirely (it is not present as null or zero).

**AC-008 (FR-007):** Given a `conflict` call in `feature_ids` mode, when the response is inspected, then it contains a feature-level result structure (distinct from the task-level structure) that includes at minimum: the list of feature pairs checked, the risk for each pair, and any per-feature warnings.

**AC-009 (FR-008):** Given the file `work/templates/implementation-plan-prompt-template.md`, when it is read, then it contains a `## Merge Schedule` section with a Markdown table template containing the columns `Cohort`, `Features`, and `Gate condition`, and prose stating the section is required for plans with more than 3 features and recommending cohort sizes of 3–5.

**AC-010 (FR-009):** Given the file `.kbz/skills/write-dev-plan/SKILL.md`, when it is read, then it contains guidance in or adjacent to its quality-checks section stating that plans with more than 3 features must include a `## Merge Schedule` section and that `conflict(action: "check", feature_ids: [...])` should be used to verify no intra-cohort file overlap.

**AC-011 (FR-010):** Given a plan with 4 or more features whose dev-plan document contains no `## Merge Schedule` heading, when `decompose(action: "review")` is called, then the response includes a warning message containing the feature count and recommending cohort groupings. The response also contains the proposal and does not error.

**AC-012 (FR-010):** Given a plan with 3 or fewer features, when `decompose(action: "review")` is called, then no merge-schedule warning is emitted regardless of whether the dev-plan has a `## Merge Schedule` section.

**AC-013 (FR-010):** Given a plan with 4 or more features whose dev-plan document contains a `## Merge Schedule` heading, when `decompose(action: "review")` is called, then no merge-schedule warning is emitted.

**AC-014 (FR-011):** Given the file `.kbz/skills/orchestrate-development/SKILL.md`, when it is read, then it contains a Phase 0 section positioned before Phase 1 that includes all five steps specified in FR-011: read merge schedule, call feature-level conflict if no schedule, group into cohorts, record cohort plan, create worktrees for cohort 1 only.

**AC-015 (FR-012):** Given the file `.kbz/skills/orchestrate-development/SKILL.md`, when Phase 6 or the Close-Out section is read, then it contains a merge-checkpoint recognition step instructing the orchestrator to check for cohort-N+1 features after merging cohort N, return to Phase 0 for the next cohort, and not create subsequent-cohort worktrees until the current cohort is confirmed merged.

**AC-016 (NFR-001):** Given the existing test suite for `conflict` task-level behaviour, when all tests are run after this change, then all previously passing tests continue to pass without modification.

**AC-017 (NFR-006):** Given the unit test suite for the feature-level conflict logic, when run without a live git repository or entity store, then tests assert: correct `files_planned` aggregation across tasks, `"no_file_data"` warning emission, `drift_days` present/absent based on worktree existence, and error on mixed `task_ids`+`feature_ids` input.

---

## Dependencies and Assumptions

**Dependencies:**
- `internal/service/conflict.go` — the `ConflictCheckInput` struct and `analyzePair` logic must be accessible for extension; no structural rewrite is required
- `internal/mcp/` — the MCP handler for the `conflict` tool must be updated to accept and pass through the `feature_ids` parameter
- The `BranchLookup` interface must provide a method to retrieve branch creation date; if not currently present, it must be added as part of this feature
- `work/templates/implementation-plan-prompt-template.md` — must be writable and not locked by another in-flight feature
- `.kbz/skills/write-dev-plan/SKILL.md` and `.kbz/skills/orchestrate-development/SKILL.md` — must not be simultaneously modified by another active feature

**Assumptions:**
- `files_planned` fields on task entities are populated for tasks that have been through the dev-planning stage; the `"no_file_data"` path (FR-004) handles the common case where they are absent at planning time
- The entity service supports filtering tasks by `parent_feature`; if not, FR-003 requires a filter scan of all tasks by `parent_feature` field match
- The `decompose` service has access to the plan's feature count and can read dev-plan document content (or at minimum check for a section heading) at review time; this access pattern is consistent with existing decompose quality-validation checks
- Calendar-day arithmetic for `drift_days` uses UTC dates throughout to avoid timezone-dependent off-by-one errors