# Research: Cohort-Based Merge Checkpoints

**Status:** Draft
**Feature:** FEAT-01KPR-FCXNMNKM
**Task:** TASK-01KPR-FD8ZJFXP
**Date:** 2026-04-23

## 1. Executive Summary

Worktree drift is a real and documented problem in this project: P3 (8 features, 3 explicit
"merge main + fix conflicts" commits) and P27 (4 parallel features, a worktree-creation timeout
at 34+ worktrees) both show evidence of branch divergence stress. The `conflict` tool currently
operates at task level and is structurally incapable of detecting feature-level branch drift.
A lightweight cohort model — grouping 3–5 features by file-scope isolation and enforcing a
merge-to-main checkpoint between cohorts — would eliminate the bulk of observed drift with
minimal tooling change. **Recommendation: option (b) — build the cohort model**, prioritising
(1) feature-level conflict analysis in the `conflict` tool, (2) a `merge_schedule` block in
decompose output, and (3) a merge-checkpoint instruction in the `orchestrate-development` skill.

## 2. Problem Statement

**Worktree drift** is the condition where a feature branch diverges from `main` as other features
land. Drift has two direct costs:

1. **Merge conflicts.** The longer a branch lives unmerged, the greater the edit distance from
   main. Concurrent changes to shared files (skill files, AGENTS.md, Go source files touched by
   multiple features) compound this distance.

2. **Coordination overhead.** An orchestrator managing many parallel branches must track which
   branches have been rebased, whether a rebase broke a sub-agent's in-progress work, and how
   to sequence the final merge window.

In the current system, the orchestrator skill asks agents to use `conflict` before each parallel
dispatch batch, but `conflict` only reasons about two tasks in isolation. It has no mechanism to
ask: "given that FEAT-A has been running for three weeks and six other features have landed since
it branched, is it still safe to dispatch more tasks against it?" There is no merge schedule as
a plan-level artifact, and no tooling to enforce merge ordering between features.

**Scale threshold.** Evidence from this project suggests the threshold is approximately **4
parallel long-running features** touching shared files. Below that, ad-hoc discipline (the
orchestrator manually checking git history) is workable. Above it, conflicts become a recurring
tax rather than an occasional nuisance.

## 3. Historical Evidence

### P3 — kanbanzai-1.0 (8 features)

P3 ran 8 features, several in parallel: init-command, skills-content, hardening,
binary-distribution, public-schema-interface, user-documentation, agents-md-cleanup,
release-infrastructure. The git log shows three explicit drift-remediation commits:

- `workflow(FEAT-01KMKRQWF0FCH): merge main into hardening, fix conflicts and build errors`
- `workflow(FEAT-01KMKRQT9QCPR): merge main into binary-distribution, fix module paths and version output`
- `workflow(FEAT-01KMKRQRRX3CC): merge main into init-command, fix module path in init_cmd.go`

Each of these required manual rebasing and conflict resolution after the fact. The merge history
also shows two features (hardening, binary-distribution) that both had to merge main after
init-command landed, because init-command altered module paths that both downstream features
depended on.

A `merge main` commit is a lagging indicator: it only appears *after* the conflict is discovered,
and the fix effort is invisible in the commit count. This is precisely the pattern a cohort
checkpoint would eliminate — init-command should have been cohort 1, triggering a merge gate
before hardening and binary-distribution progressed further.

### P27 — doc-intel-adoption (4 features, 20 tasks)

P27 ran 4 parallel features (design-stage-consultation, corpus-hygiene-classification,
knowledge-lifecycle-mandate, doc-intel-instrumentation) across 20 tasks, all merged in the
same release window. The "mark all 20 P27 tasks done; register 4 P27 worktrees" commit shows
the orchestrator batched worktree registration after-the-fact, suggesting the worktrees were
created sequentially during the run. The worktree-creation-timeout bug (FEAT-01KPVDDYZ3182)
was opened during this plan and explicitly attributed to "the repository has many existing
worktrees (~34+)". At 34+ simultaneous worktrees, git's internal locking serialised worktree
operations and caused timeouts. This is a second-order effect of not cleaning up merged worktrees
between cohorts.

### P26 — workflow-scaling (created but not evidenced in git history)

P26 was created but leaves no substantial merge trace, suggesting it may have been a lighter
plan. Not enough data for a conflict assessment.

### Branch count

As of this research, `git branch -r | wc -l` returns 7 remote branches — the repository is
relatively clean now. However the 34+ worktree count during P27 indicates that active-session
worktree accumulation is the primary vector, not long-lived remote branches.

### Summary

| Plan | Features | Parallel branches | Drift incidents | Notes |
|------|----------|-------------------|-----------------|-------|
| P3   | 8        | 4–5 peak          | 3 explicit      | Module path conflicts; manual fix required |
| P27  | 4        | 4                 | 1 indirect      | Worktree lock contention at 34+ total |
| P28  | 5        | 3–4               | 0 recorded      | Smaller scope, better serialisation |

The evidence supports a **threshold of ~4 parallel features with shared-file overlap** as the
point where unassisted drift management breaks down.

## 4. What a Cohort Looks Like

A **cohort** is a set of features that can safely be developed in parallel and merged together
in a single window, because their file scopes do not overlap with each other or with pending
features in other cohorts.

### Grouping criteria

A feature belongs to cohort N if:
1. Its file scope does not overlap with any feature in the same cohort (checked via the conflict
   tool extended to feature level — see §5.1).
2. It has no declared dependency on an unmerged feature in a later cohort (dependency order).
3. Its estimated duration does not exceed a configurable max drift window (default: one plan
   "sprint" or the time to complete 4–6 other features).

In practice, a cohort of **3–5 features** is the right size. Smaller than 3 loses parallelism
benefit; larger than 5 recreates the P3 drift problem.

### Cohort boundaries

A **merge checkpoint** between cohort N and cohort N+1 means:
- All features in cohort N are merged to main before any feature in cohort N+1 begins development.
- The checkpoint is a hard gate, not advisory.
- An orchestrator that has dispatched all cohort-N work waits for all merges to complete before
  creating worktrees for cohort N+1 features.

### What triggers a checkpoint

- All features in the current cohort reach `reviewing` (or `done`).
- OR a new feature is proposed whose file scope conflicts with the current cohort (early
  checkpoint to drain the cohort).
- OR a configurable drift-age threshold is reached (e.g. a feature branch is more than N commits
  behind main).

### Example: P3 with cohorts

Under a cohort model, P3 would have been organised as:

**Cohort 1 (foundation):** init-command, skills-content
→ Merge gate: both merged before cohort 2 starts.

**Cohort 2 (main features):** hardening, binary-distribution, public-schema-interface
→ Merge gate: all three merged.

**Cohort 3 (wrap-up):** user-documentation, agents-md-cleanup, release-infrastructure

This sequencing eliminates the three drift-fix commits because hardening and binary-distribution
would not have started until init-command's module path changes were already in main.

## 5. Tooling Gap Analysis

### 5.1 Conflict Tool

**Current capability.** The `conflict` tool accepts an array of `TASK-...` IDs and performs
three pairwise checks: file overlap (planned files + git branch diff), dependency order
(explicit `depends_on` + transitive reachability), and boundary crossing (shared keyword
heuristic). It returns per-pair risk and a recommendation (`safe_to_parallelise`, `serialise`,
or `checkpoint_required`).

**Gap.** The tool operates at task level. It cannot answer "do FEAT-A and FEAT-B conflict?" as
a feature-to-feature question. There is no concept of a feature's aggregate file scope, no
branch-age drift detection, and no ability to reason about the ordering of feature merges.
Critically, it requires task IDs — you cannot call it during plan decomposition when features
exist but tasks have not yet been created.

**Extension needed.**

1. Add a `feature_ids` parameter (array of `FEAT-...` IDs) as an alternative to `task_ids`.
   When feature IDs are provided, aggregate all tasks' `files_planned` fields per feature and
   perform the same file-overlap check at the feature level.

2. Add a `branch_age_days` field to the feature-level result, computed from the worktree's
   branch creation date vs. `main` HEAD commit date. Surface this as a `drift` risk dimension
   alongside the existing three.

3. The `analyzePair` function in `internal/service/conflict.go` is already cleanly separated —
   a `checkFeaturePair` wrapper that aggregates task files would require ~50 lines of new Go
   against existing interfaces. The `BranchLookup` interface already provides `GetFilesOnBranch`
   which the feature-level check can reuse directly.

**No structural rewrite is needed.** The gap is additive: a new input mode and one new dimension
in the response.

### 5.2 Decompose / Dev-Planning

**Current output.** `decompose(action: propose)` returns a flat list of `ProposedTask` objects
with `depends_on` arrays. There is no feature-level grouping and no concept of merge ordering.
The dev-planning skill (`write-dev-plan/SKILL.md`) does not mention cohorts or merge scheduling.
The `decompose-feature` skill has quality validation checks (dependency declared, sizing,
testing) but no inter-feature ordering check.

**Gap.** When a plan has multiple features, there is no artifact that encodes which features
form a cohort and in what order cohorts merge. The orchestrator must infer this from task
dependencies, which only exist within a feature, not between features. An orchestrator starting
a new plan has no authoritative source for merge sequencing.

**Change needed.**

1. Add an optional `merge_schedule` block to the plan-level dev-plan document template:

   ```
   ## Merge Schedule
   | Cohort | Features | Gate condition |
   |--------|----------|----------------|
   | 1 | feat-a, feat-b | Both reach reviewing |
   | 2 | feat-c, feat-d, feat-e | All reach reviewing |
   ```

2. The `write-dev-plan` skill should instruct the architect to author this block when a plan
   has more than 3 features.

3. `decompose(action: propose)` does not need to produce the merge schedule automatically —
   it is a design-time artifact, not an algorithmic output. Requiring the author to think
   through file-scope isolation at planning time is the right forcing function.

4. The `decompose-feature` quality validation check should add a warning if the parent plan
   has more than 3 features and no merge schedule block exists in its dev-plan.

### 5.3 Orchestration Skill

**Current guidance.** The `orchestrate-development` skill Phase 2 instructs the orchestrator to
use `conflict` before each parallel dispatch batch to check task-level conflicts. Phase 6
(Close-Out) instructs branch deletion. There is no mention of cohort boundaries, merge
scheduling, or checking for cross-feature drift before starting a new feature's worktree.

**Gap.** The orchestrator has no instruction to:
- Read the plan's merge schedule before beginning development.
- Defer worktree creation for cohort-N+1 features until cohort-N is merged.
- Perform a feature-level conflict check at plan start (not just task-level at dispatch time).
- Recognise when it has reached a merge checkpoint.

**Addition needed.**

Add a new **Phase 0: Plan-Level Conflict Check** before the current Phase 1:

> 0. If the plan has more than 3 features, call `conflict` with `feature_ids` for all features
>    in the plan to identify file-scope overlap. Group features into cohorts based on the
>    overlap results and any explicit `merge_schedule` in the dev-plan. Record the cohort
>    grouping in a progress note. Only create worktrees for cohort-1 features.

And add to Phase 6:

> After all cohort-N features are merged, check whether cohort-N+1 features exist. If so,
> begin the orchestration loop again at Phase 1 for cohort N+1. Do not create cohort-N+1
> worktrees until the merge checkpoint is confirmed.

## 6. Harmful Pre-Merge Cases

There are cases where enforcing a merge checkpoint is actively harmful:

**1. Partially complete features.** A feature in `developing` with half its tasks done cannot
be merged — the merge gate requires all tasks terminal. A cohort checkpoint that gates on "all
features reach reviewing" would block the entire cohort if one feature is stuck. Mitigation:
the checkpoint condition should be "all cohort-N features reach `done` or are explicitly
withdrawn from the cohort", not a fixed task-completion threshold.

**2. Feature flags not implemented.** If a feature adds new functionality that should not be
user-visible until a later cohort is also merged (e.g. a new API endpoint + a UI feature that
uses it), merging the backend feature in cohort 1 and the UI feature in cohort 2 is safe only
if the endpoint is not inadvertently exposed. This is an application concern rather than a
workflow concern, but the dev-plan merge schedule should document it as a sequencing rationale.

**3. Experimental / proof-of-concept features.** A feature that is exploratory and may be
abandoned should not block a cohort checkpoint. Such features should be tagged or excluded from
cohort membership at plan time.

**4. Documentation-only features.** Features that only modify Markdown files (`.agents/skills/`,
`docs/`) never have merge conflicts with code features. Including them in a cohort adds no
conflict protection and blocks the checkpoint unnecessarily. They should be classified as
`cohort: any` and merged whenever convenient.

**5. Hotfixes.** If a critical bug is discovered during cohort-1 development, the hotfix should
merge immediately regardless of cohort state. Cohort discipline is a planning tool, not a
production-safety constraint.

## 7. Orchestrator Experience

Under the proposed cohort model, here is the step-by-step orchestrator experience for a
4-feature plan with a 2+2 cohort structure:

**Session start (plan-level):**

1. Call `status(id: "P-xxx")` to read the plan state and feature list.
2. Read the dev-plan's `## Merge Schedule` block (if present) to identify cohort groupings.
3. If no merge schedule exists and the plan has more than 3 features, call
   `conflict(action: "check", feature_ids: [...])` for all features to determine overlap.
   Group features into cohorts based on results: features with overlapping file scopes must be
   serialised (different cohorts); features with no overlap can be parallelised (same cohort).
4. Record the cohort plan in a progress note: "Cohort 1: FEAT-A, FEAT-B. Cohort 2: FEAT-C,
   FEAT-D."
5. Create worktrees only for cohort-1 features.

**Cohort-1 development (per current skill, Phases 1–5):**

No change to the existing parallel dispatch loop. The orchestrator dispatches tasks for
FEAT-A and FEAT-B in parallel, monitors progress, and handles failures as today.

**Reaching the merge checkpoint:**

6. When all cohort-1 features are in `reviewing` or `done`, the orchestrator recognises the
   merge checkpoint. The signal is: `status(id: "P-xxx")` shows no `developing` features in
   cohort 1, and no `needs-rework` features in cohort 1.
7. Complete Phase 6 (Close-Out) for each cohort-1 feature: merge, delete branch, remove
   worktree.
8. Verify: `git branch | grep FEAT-A` and `FEAT-B` return no output.
9. Log the checkpoint: "Cohort 1 merged. Main is clean. Starting cohort 2."

**Cohort-2 development:**

10. Create worktrees for FEAT-C and FEAT-D (cohort 2 starts fresh from the updated main).
11. Resume from Phase 1 for cohort-2 features.

**What the orchestrator does when a checkpoint is missed (drift detected):**

If, during cohort-2 development, `conflict(action: "check", feature_ids: ["FEAT-C", "FEAT-D"])`
returns a high-risk result tracing to a recent main-branch change, the orchestrator should:
- Stop dispatching new tasks for affected features.
- Rebase the affected feature branch onto main (using `terminal`).
- Re-run the conflict check.
- Only resume dispatch if the rebase is clean.

This mirrors the existing Phase 4 failure-handling pattern and requires no new escape hatches.

## 8. Recommendation

**Option (b): design and build a minimal cohort model.**

The historical evidence (three explicit "merge main + fix conflicts" commits in P3; the 34+
worktree timeout in P27) demonstrates that ad-hoc file-scope discipline breaks down at 4+
parallel features with shared-file overlap. The current tooling has a clear structural gap: the
`conflict` tool is task-scoped, decompose produces no merge schedule, and the orchestration
skill has no merge-checkpoint concept.

The required changes are additive and low-risk. No existing tool or workflow is broken; the
new capabilities layer on top of what already works.

**Prioritised build list:**

1. **Feature-level conflict analysis** (conflict tool, `internal/service/conflict.go`).
   Add `feature_ids` input mode. Aggregate task `files_planned` per feature. Reuse existing
   `BranchLookup.GetFilesOnBranch` for git-level diff comparison. Add `drift_days` dimension.
   Estimated: ~80–120 lines of Go, no schema changes. This is the highest-value change because
   it unblocks plan-time conflict detection before tasks exist.

2. **Merge schedule in dev-plan template and decompose warning**.
   Add `## Merge Schedule` table to `work/templates/implementation-plan-prompt-template.md`.
   Add a warning to `decompose` quality validation when a plan has more than 3 features and no
   merge schedule block is present in the dev-plan. Add authoring guidance to the
   `write-dev-plan` skill. Estimated: documentation + ~20 lines of Go for the validation check.

3. **Cohort-aware orchestration in the orchestrate-development skill**.
   Add Phase 0 (plan-level conflict check and cohort grouping) and a merge-checkpoint
   recognition step to Phase 6. No code change required — this is a pure skill update.
   Estimated: ~40 lines of Markdown.

These three changes, taken together, address the full gap surface identified in §5 and provide
the orchestrator experience described in §7.

## 9. Open Questions

1. **Automatic cohort inference vs. human-authored merge schedule.** The recommendation above
   leaves cohort authoring to the architect (with a `conflict` check as a prompt). An
   alternative is to have `decompose(action: propose)` auto-generate a suggested merge schedule
   based on feature-level file overlap — but this requires the feature-level conflict extension
   (item 1 above) to land first, and it requires task `files_planned` fields to be populated
   at decompose time (currently they are often populated during implementation, not planning).
   Deferred pending feature-level conflict extension.

2. **Drift threshold.** What is the right maximum drift window before a cohort checkpoint
   becomes mandatory? This research uses "3+ features landed since branch creation" as a
   heuristic, but no empirical measurement has been done on this project. A future data
   collection pass over P3 and P27 branch histories could provide a concrete number.

3. **Interaction with the review lifecycle.** The merge checkpoint currently assumes features
   reach `reviewing` or `done` before the next cohort starts. If a feature requires multiple
   review cycles (max 3 per the binding registry), the cohort-1 checkpoint could be delayed
   significantly. Should a feature be allowed to "exit" a cohort early (i.e. cohort-2 starts
   while cohort-1 review cycles are still in progress)? This is safe only if cohort-2 file
   scopes do not overlap with the features still under review — which the extended conflict tool
   could verify.

4. **Tooling for cohort state tracking.** The current entity model has no `cohort` field.
   Should cohort membership be stored as a field on the feature entity, as a tag, or only in
   the dev-plan document? Storing it as an entity field would enable `entity(action: list,
   cohort: 1)` queries but adds schema complexity. The document-only approach is simpler and
   sufficient for human + LLM orchestrators.