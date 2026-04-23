# Design: Cohort-Based Merge Checkpoints (P33)

| Field  | Value                    |
|--------|--------------------------|
| Date   | 2026-06-17               |
| Status | approved |
| Author | Claude Sonnet 4.6        |
| Plan   | P33-cohort-merge-checkpoints |

---

## Overview

Worktree drift causes recurring merge conflicts and coordination overhead when four or more
features run in parallel — a pattern evidenced by three P3 "merge main + fix conflicts"
commits and the P27 34-second worktree timeout. This design adds a cohort model through
three additive changes: a feature-level input mode for the `conflict` tool, a `## Merge
Schedule` section in the dev-plan template, and Phase 0 cohort-setup guidance in the
`orchestrate-development` skill. No existing tool APIs, entity schemas, or workflow gates
are broken or removed.

## Goals and Non-Goals

**Goals:**
- Add `feature_ids` input mode to the `conflict` tool to enable feature-level overlap detection
- Add `drift_days` dimension to the conflict result so orchestrators can detect stale branches
- Add a `## Merge Schedule` section to `work/templates/implementation-plan-prompt-template.md`
- Add a decompose quality-validation warning when a plan has more than 3 features and no merge schedule block in its dev-plan
- Add authoring guidance to the `write-dev-plan` skill
- Add Phase 0 (plan-level conflict check and cohort grouping) and a merge-checkpoint step to the `orchestrate-development` skill

**Non-Goals:**
- No changes to the entity data model (no `cohort` field on feature entities)
- No automatic cohort inference from decompose — the merge schedule is a human/architect-authored artifact
- No changes to the `merge` tool, `pr` tool, or `branch` tool
- No enforcement of cohort gates by the server (advisory, not enforced)

## Dependencies

- `internal/service/conflict.go` — primary Go change target
- `internal/mcp/handlers_conflict.go` (or equivalent MCP handler) — add `feature_ids` parameter
- `work/templates/implementation-plan-prompt-template.md` — add Merge Schedule section
- `.kbz/skills/write-dev-plan/SKILL.md` — add cohort authoring guidance
- `.kbz/skills/orchestrate-development/SKILL.md` — add Phase 0 and checkpoint step

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `work/research/cohort-merge-checkpoints.md` | Research | Primary source; establishes evidence base (P3 drift commits, P27 worktree timeout) and the three-change recommendation |
| `work/design/p25-orchestration-docs.md` | Design | P25 orchestration skill changes; constrains the phase numbering and style of `orchestrate-development` |
| `work/design/p29-state-store-read-performance.md` | Design | P29 read-path fix; directly addresses the worktree timeout that P27 exposed (the second-order effect of drift accumulation) |

### Decisions that constrain this design

| Decision | Source | Constraint |
|----------|--------|------------|
| The conflict tool's existing `BranchLookup` interface provides `GetFilesOnBranch` | `conflict.go` architecture | The feature-level check must use this interface and must not introduce direct git calls |
| Cohort discipline is advisory, not server-enforced | Research §6 (harmful pre-merge cases) | Orchestrators may deviate with human approval; the server must not block on cohort membership |
| The dev-plan merge schedule is a human-authored artifact | Research §5.2 | Decompose may warn when the schedule is missing but must not block proposal generation |

### Open questions from prior work

- Should `conflict` warn (non-blocking) or error (blocking) when `feature_ids` are provided but tasks have no `files_planned` fields? (Most likely: warn with a "no file data" note per feature.)
- What is the right `drift_days` threshold for a "high" drift risk rating? Research uses a heuristic of "3+ features landed since branch creation" but proposes future empirical measurement.

---

## Problem and Motivation

When a plan runs more than three features in parallel, each feature's worktree branch
progressively diverges from `main` as other features merge. The longer a branch lives
without rebasing, the larger its merge surface — each subsequent merge must reconcile all
intermediate changes, and conflicts compound rather than isolate.

Historical evidence from P3 (a plan that ran at 4–6 parallel features) shows three
distinct commits with the message pattern "merge main + fix conflicts". These commits
represent unplanned coordination work that was not tracked as tasks, and each one required
an orchestrator to stop forward progress, switch context, and manually resolve overlapping
changes. The P27 plan also produced a 34-second `worktree create` timeout that was traced
to the entity service's O(n) file scan — a second-order effect of the large number of
concurrent worktrees that drift-based serialization overhead creates.

The structural root cause is that the `conflict` tool operates only at the task level
(`task_ids`). It can identify which *tasks* overlap on files, but it cannot answer the
question an orchestrator actually needs at planning time: "do these *features* overlap
enough that merging them concurrently will produce conflicts?" Without a feature-level view,
orchestrators have no systematic way to group features before dispatching work, and no
vocabulary for expressing a merge order in the dev-plan document. The result is ad hoc
parallelism that is only discovered to be problematic once branches have been live for days.

This design adds the minimal tooling needed to make cohort discipline possible: a way to
check feature-level overlap before work starts, a place in the dev-plan to record merge
order, and skill guidance that makes Phase 0 cohort setup a standard step for large plans.

## Design

### Enhancement 1: Feature-level conflict analysis (conflict tool)

Add `feature_ids []string` to the `ConflictCheckInput` struct alongside the existing
`task_ids` field. The two fields are mutually exclusive — if both are provided, the handler
returns an error.

When `feature_ids` is provided:

1. For each feature ID, call `entitySvc.List("task")` filtered by `parent_feature` to
   resolve the feature's constituent tasks.
2. Aggregate `files_planned` across all tasks in that feature to build a per-feature file
   set. If a feature has no tasks with `files_planned` populated, include a `"no_file_data"`
   warning in the result for that feature rather than returning an error.
3. Run the existing `analyzePair` logic against the per-feature file sets (file-set
   intersection for overlap detection, dependency-order analysis).
4. Add a `drift_days` field to the result: compute the number of calendar days between the
   feature's worktree branch creation date and the current date using the `BranchLookup`
   interface. If no worktree exists for the feature, omit the field.
5. Return a `FeatureConflictResult` type (alongside the existing `TaskConflictResult`)
   when the feature-IDs mode is used.

The MCP tool handler adds `feature_ids` as an optional array parameter. Input validation
rejects requests that supply both `task_ids` and `feature_ids`.

### Enhancement 2: Merge schedule in dev-plan template and decompose warning

**Template change** — add a `## Merge Schedule` section to
`work/templates/implementation-plan-prompt-template.md` below the existing task-breakdown
section. The section provides a table template with columns: `| Cohort | Features | Gate condition |`.
Accompanying prose notes that the section is required when the plan has more than 3 features,
that cohort size should be 3–5, and that intra-cohort file overlap should be verified with
`conflict(action: "check", feature_ids: [...])` before publishing the dev-plan.

**Skill change** — add a note to `.kbz/skills/write-dev-plan/SKILL.md` under the
"Quality checks" step: "When the plan has more than 3 features, add a `## Merge Schedule`
section grouping features into cohorts of 3–5. Use `conflict(action: "check", feature_ids:
[...])` to verify no intra-cohort file overlap before publishing the dev-plan."

**Decompose warning** — in the quality-validation path (`internal/service/decompose.go` or
the proposal-review step), add a non-blocking warning emitted when:

- The plan's feature count exceeds 3, **and**
- The associated dev-plan document does not contain a `## Merge Schedule` heading.

Warning text: "Plan has N features but dev-plan has no Merge Schedule section. Consider
adding cohort groupings to prevent worktree drift."

This warning appears in the `decompose(action: "review")` output. It does not prevent
proposal generation or the `apply` step.

### Enhancement 3: Cohort-aware orchestrate-development skill

Insert **Phase 0: Cohort Setup** before the current Phase 1 in
`.kbz/skills/orchestrate-development/SKILL.md`. Phase 0 applies only when the plan has
more than 3 features.

**Phase 0 — Cohort Setup** _(plans with more than 3 features only)_

- **0.1** Read the dev-plan's `## Merge Schedule` block if present. If a merge schedule
  exists, treat its cohort groupings as authoritative and proceed to Phase 1 for cohort-1
  features only.
- **0.2** If no merge schedule exists, call `conflict(action: "check", feature_ids: [...])`
  for all features in the plan to identify file-scope overlap.
- **0.3** Group features into cohorts: features with no file overlap may be parallelised
  (same cohort); overlapping features must be serialised (different cohorts). Target cohort
  size: 3–5 features.
- **0.4** Record the cohort plan in the session context: "Cohort 1: FEAT-A, FEAT-B, FEAT-C.
  Cohort 2: FEAT-D, FEAT-E."
- **0.5** Create worktrees only for cohort-1 features. Do not create worktrees for
  cohort-2+ features until the preceding cohort's merge checkpoint is confirmed clean.

Add a **merge-checkpoint recognition** step to Phase 6 (Close-Out):

After merging all cohort-N features, check whether cohort-N+1 features exist. If so,
return to Phase 0 for cohort N+1. Do not create cohort-N+1 worktrees until the cohort-N
merge checkpoint is confirmed clean: `git status` shows no open feature branches from
cohort N and all cohort-N PRs are merged.

## Alternatives Considered

### Alternative 1: Automatic cohort inference in decompose

Decompose could analyse `files_planned` fields at proposal time and automatically assign
features to cohorts. Rejected because `files_planned` fields are typically populated during
implementation, not at planning time — most features have no file data when decompose runs.
The conflict check would return "no file data" results for the majority of features, making
automatic inference unreliable. Deferred to a follow-up once `files_planned` discipline
improves across plans.

### Alternative 2: Cohort field on feature entity

Adding a `cohort` field to the feature entity schema would make cohort membership
queryable via the `entity` tool. Rejected because it adds schema complexity for a field
that is advisory and already readable from the dev-plan document. Document-level cohort
tracking is sufficient for human and LLM orchestrators and requires no entity model
migration or backward-compatibility handling for existing feature records.

### Alternative 3: Server-enforced cohort gates (blocking worktree creation for cohort N+1)

The server could refuse `worktree(action: "create")` calls for cohort-N+1 features until
all cohort-N worktrees are merged. Rejected because it would require the server to parse
the dev-plan document at tool-call time and encode workflow policy in the binary. This
violates the design principle that workflow policy lives in skills and documents, not in
the server. Cohort discipline is advisory; orchestrators may deviate with human approval.

### Alternative 4: Status quo — do nothing

Rejected based on historical evidence (three P3 conflict-fix commits; P27 worktree
timeout) and the trajectory that at ~600 task files the problem will compound with the
read-path performance regression addressed in P29. The cost of the three additive changes
in this design is low relative to the recurring coordination overhead that unmanaged drift
produces.

## Decisions

| ID | Decision | Rationale |
|----|----------|-----------|
| P33-DEC-001 | Feature-level conflict mode uses `feature_ids` as a mutually exclusive alternative to `task_ids` | Keeps the API surface clean; prevents ambiguous mixed inputs where task and feature scopes overlap |
| P33-DEC-002 | `drift_days` is informational only — not mapped to a risk level | Drift thresholds are not yet empirically established; a raw number is more useful to orchestrators than a premature risk categorisation |
| P33-DEC-003 | Merge schedule is author-driven, not auto-generated | Forces the architect to reason about file-scope isolation at design time; algorithmic inference is unreliable until `files_planned` discipline improves across the project |
| P33-DEC-004 | Decompose warning is non-blocking | A missing merge schedule is a quality signal, not a correctness failure; blocking proposal generation would break the workflow for small plans that hit the N>3 threshold incidentally |
| P33-DEC-005 | Cohort gates are advisory, not server-enforced | Workflow policy belongs in skills; server enforcement adds coupling between the binary and a changeable workflow convention and removes the orchestrator's ability to deviate with human approval |
