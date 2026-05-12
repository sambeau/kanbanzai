---
name: orchestrate-development
description:
  expert: "Multi-agent development orchestration with conflict-aware parallel
    dispatch, context compaction, failure handling, and lifecycle close-out
    for a single feature within a batch"
  natural: "Coordinate a team of agents to build a feature: dispatch tasks in
    parallel, monitor progress, handle failures, compact context, and close
    out the feature lifecycle"
triggers:
  - orchestrate development tasks
  - coordinate implementation work
  - dispatch sub-agents for development
  - manage parallel task execution
  - run a feature's dev-plan
roles: [orchestrator]
stage: developing
constraint_level: medium
---

## Vocabulary

- **ready frontier** — the set of tasks whose dependencies are all satisfied and which are eligible for dispatch; every dispatch cycle picks from this set
- **dispatch batch** — a group of tasks sent to sub-agents in a single parallel dispatch cycle; drawn from the ready frontier
- **conflict domain** — the set of files a task will modify; two tasks in the same dispatch batch must have non-overlapping conflict domains
- **command broadcast** — a terminal command feature file shared across all sub-agents in a dispatch batch to ensure consistent tool usage
- **context compaction** — summarising completed task outputs to 2–3 sentences each and discarding full outputs, preventing context saturation
- **document-based offloading** — writing a progress document and starting a fresh session when context utilisation exceeds 60%
- **cohort** — a subset of features within a batch that can be parallelised without file overlap; features with overlapping file scopes are serialised into different cohorts
- **post-completion summary** — the 2–3 sentence reduction of a sub-agent's full output, retaining the task ID, what was built, and whether it passed
- **failure classification** — separating recoverable failures (missing context, file path issues) from unrecoverable ones (spec ambiguity, design gaps) to determine retry vs. escalation
- **close-out checklist** — the sequence of lifecycle transitions, branch cleanup, and knowledge curation that completes a feature after all tasks are done
- **merge checkpoint** — the point after a cohort's features have merged where the orchestrator confirms no open branches remain before starting the next cohort's worktrees
- **feature completion summary** — a concise record of what the feature delivered, used by reviewers and the next orchestrator session

## Anti-Patterns

### Dispatching With Unmet Dependencies

- **Detect:** A task is dispatched before its `depends_on` tasks are in `done` status
- **BECAUSE:** The sub-agent will sit idle waiting for inputs that don't exist yet, wasting context budget and producing placeholder code that must be reworked
- **Resolve:** Verify all `depends_on` entries are `done` before adding a task to the dispatch batch. Use the ready frontier exclusively.

### Full-Output Retention

- **Detect:** The orchestrator's context contains full sub-agent outputs (diffs, tool call logs, reasoning traces) from completed tasks
- **BECAUSE:** Full outputs consume ~10–20% of context per task. After 4–5 tasks, the orchestrator has no room for new dispatch decisions, causing quality degradation or session failure
- **Resolve:** Apply post-completion summarisation immediately after each task completes. The summary replaces the full output.

### Multi-Feature Session

- **Detect:** The orchestrator is managing more than one feature in the same session
- **BECAUSE:** Each feature has its own task graph, file scopes, and spec context — combining them doubles context load without improving throughput, and the orchestrator loses per-feature focus
- **Resolve:** One feature per orchestration session. Feature completion summary marks the handoff point.

### Fire-and-Forget Dispatch

- **Detect:** Sub-agents are dispatched but their outcomes are not checked — the orchestrator assumes completion without verification
- **BECAUSE:** Sub-agents can silently fail (wrong return status, incomplete work, uncommitted changes) without the orchestrator noticing, creating gaps that surface during feature close-out
- **Resolve:** After each dispatch cycle, check every task's status. Apply post-completion summary or failure handling before the next cycle.

### Serial Dispatch of Independent Tasks

- **Detect:** Independent tasks (no dependency edges between them) are dispatched one at a time instead of in parallel
- **BECAUSE:** The sub-agent dispatch overhead is the same whether you dispatch 1 or 4 agents; serialising independent tasks wastes the parallel capacity the orchestrator-workers pattern exists to exploit
- **Resolve:** Always dispatch the full ready frontier in one parallel batch. Only serialise when conflict-domain analysis finds overlapping files.

### Ignoring Failure Signals

- **Detect:** A task fails but the orchestrator continues dispatching downstream dependents
- **BECAUSE:** Downstream tasks depend on upstream outputs; if the upstream failed, downstream work is invalid — the sub-agent builds against missing or incorrect inputs
- **Resolve:** When a task fails, block all dependents. Handle the failure (retry or escalate) before dispatching anything that depends on it.

### Context Bloat Without Offloading

- **Detect:** Context utilisation exceeds 60% but the orchestrator continues dispatching new tasks
- **BECAUSE:** Sustained high context utilisation degrades tool call accuracy, dependency tracking, and clinical judgement (Masters et al., 2025). The orchestrator misses failure signals it would normally catch.
- **Resolve:** Stop dispatching at 60%. Offload to a progress document. Start a fresh session. This is a mandatory offloading point, not a suggestion.

### Assigning Multiple Large-File Tasks to One Sub-Agent

- **Detect:** A single sub-agent receives multiple tasks where each task involves reading or rewriting files longer than ~300 lines
- **BECAUSE:** Full-file rewrites embed entire file content in terminal tool calls; an agent assigned multiple large-file tasks will saturate its context window before completing the second task
- **Resolve:** Dispatch one sub-agent per large-file task to give each a fresh context window

### Manual Prompt Composition

→ **INV-001 — Handoff-only dispatch.** Always use `handoff(task_id: "TASK-xxx", role: "implementer-go")` to generate sub-agent prompts. Manual composition violates INV-001: it silently omits graph project context, knowledge entries, and spec sections assembled by the pipeline. When P44's `dispatch_task` is available, use that instead.

> **Pre-delegation Code Investigation** is a canonical anti-pattern defined in the `orchestrator` role (`.kbz/roles/orchestrator.yaml`).

## Checklist

Copy this checklist and track your progress:
- [ ] Read the dev-plan and identified all tasks for this feature
- [ ] Queried current task statuses to build the ready frontier
- [ ] Verified file scope boundaries for parallel dispatch (no overlaps)
- [ ] Dispatched first batch of parallel-dispatchable tasks
- [ ] Monitored outcomes — confirmed done or handled failures
- [ ] Applied post-completion summary for each completed task
- [ ] Updated ready frontier after completions
- [ ] Checked context utilisation — offloaded if approaching 60%
- [ ] Repeated dispatch cycle until all tasks are done
- [ ] Wrote feature completion summary

## Procedure

### Phase 0: Cohort Setup _(batches with more than 3 features only)_

Skip this phase entirely if the batch has 3 or fewer features.

> **Test-health gate:** Before dispatching any cohort, call `test(action: "verify")`
> to confirm the test suite on `main` is green. If tests are failing, bail — do not
> dispatch sub-agents until the test suite is fixed. This prevents compounding
> pre-existing failures with new work.

1. Read the dev-plan's `## Merge Schedule` block. If a merge schedule is present, treat
   its cohort groupings as authoritative — record them and proceed to Phase 1 for
   cohort-1 features only.
2. If no merge schedule exists, call `conflict(action: "check", feature_ids: [...])` for
   all features in the batch to identify file-scope overlap.
3. Group features into cohorts based on the results: features with no file overlap may be
   parallelised (same cohort); overlapping features must be serialised (different cohorts).
   Target cohort size: 3–5 features.
4. Record the cohort plan in the session: "Cohort 1: FEAT-A, FEAT-B. Cohort 2: FEAT-C,
   FEAT-D, FEAT-E."
5. Create worktrees only for cohort-1 features. Do not create worktrees for later cohorts
   until the preceding cohort's merge checkpoint is confirmed clean.

### Phase 1: Read the Dev-Plan

> **INV-004 · No Code Investigation (ℋ):** Do not read source files, trace call
> paths, search the code graph, or shell-read `.kbz/state/` before dispatching. The
> sub-agent receives everything needed via `handoff`; if the dev-plan is unclear, flag
> it — do not read code or workflow state to compensate. See INV-004 and the
> orchestrator role's Pre-delegation Code Investigation anti-pattern.
>
> **Hard constraint (ℋ)** — violation blocks stage advance.

1. Call `status` with the feature ID to get the current state of all tasks.
1a. Call `knowledge(action: "list")` with feature-area tags and `status: "confirmed"` to surface project-level knowledge before dispatching any sub-agents. Review all returned entries. Note any that describe pitfalls, architectural constraints, or patterns relevant to this feature. Carry these entries forward: include them in each `handoff` tool call via the `instructions` parameter so sub-agents benefit from accumulated knowledge without having to rediscover it.
2. Read the dev-plan document linked from the feature entity. Understand the full task graph — which tasks exist, what each produces, and how they connect.
3. Note any tasks already completed from prior sessions. Build a running record of what is done and what remains.
4. IF the dev-plan is missing or the feature has no tasks → STOP. The feature is not ready for orchestration.

### Phase 2: Identify Parallel-Dispatchable Tasks

1. From the task list, identify every task whose `depends_on` entries are all in `done` status and that is itself in `ready` status. This is the ready frontier — your dispatch batch.
2. For each task in the batch, note its file scope (files it will create or modify).
3. IF two tasks in the batch modify the same file → use the `conflict` tool to assess risk. IF conflict risk is high → remove one task from this batch and dispatch it in the next cycle.
4. IF the ready frontier is empty and tasks remain undone → check for blocked tasks. IF all remaining tasks have unmet dependencies on non-done tasks → a dependency chain is stalled. Check whether a failed task is blocking the chain and handle per Phase 4.

### Phase 3: Dispatch Sub-Agents

**Sizing rule — one sub-agent per task for large-file features:**
For features with more than three tasks where tasks involve reading or rewriting source files longer than ~300 lines, dispatch one sub-agent per task rather than assigning multiple tasks to a single agent. Full-file rewrites embed entire file content in terminal tool calls; an agent assigned multiple large-file tasks will saturate its context window before completing the second task. Per-task isolation gives each agent a fresh context window sized for one file scope.

Features with small files or documentation-only tasks do not require per-task isolation — batch dispatch remains appropriate.

**Rule:** Always use `handoff(task_id: "TASK-xxx", role: "implementer-go")` to generate sub-agent prompts. Never compose implementation prompts manually — manual composition silently omits graph project context, knowledge entries, and spec sections.

0. **Before creating worktrees**, verify `.kbz/local.yaml` has `codebase_memory.graph_project` set
   (e.g. `Users-alice-Dev-myrepo`). If missing, add it now — worktrees created without it will not
   inject Code Graph context into sub-agent handoffs. Once set in local config, the `worktree` tool
   picks it up automatically and you never need to pass `graph_project` explicitly.
1. For each task in the dispatch batch, generate a sub-agent prompt using `handoff(task_id: "TASK-xxx", role: "implementer-go")`.
2. Include file scope boundaries in the dispatch — tell each agent which files it owns and which it must not modify (because a parallel agent owns them).
3. Include codebase knowledge graph context per `refs/sub-agents.md`: project name, tool preferences, and the propagation rule for nested delegation.
4. Dispatch all agents in the batch. Record which tasks were dispatched and when.

### Phase 4: Monitor Progress and Handle Failures

1. As sub-agents complete, check each task's status.
2. For each completed task:
   - Verify it reached `done` (not `needs-rework` or still `active`).
   - Apply post-completion summary immediately: reduce the outcome to 2–3 sentences plus the task ID. Do not retain the full sub-agent output.
3. For each failed task:
   - Read the failure details from the task or sub-agent output.
   - IF the failure is recoverable (missing context, wrong file path, minor misunderstanding) AND the retry budget is not exhausted → update the handoff with corrected information and re-dispatch once.
   - IF the failure is not recoverable (spec ambiguity, design gap, circular dependency discovered) OR the retry budget is exhausted → escalate to the human via a checkpoint. Do not re-dispatch.
4. After processing all outcomes, return to Phase 2 to identify the next dispatch batch from the updated ready frontier.

### Phase 5: Context Compaction

Apply these three techniques throughout orchestration, not only at the end:

**Technique 1 — Post-Completion Summarisation (after every task completion):**
When a sub-agent completes a task, reduce its outcome to a post-completion summary: the task ID, 2–3 sentences describing what was built and any notable decisions, and whether it passed or failed. Discard everything else — diffs, test output, reasoning traces, tool call logs. The summary is the only record you carry forward.

**Technique 2 — Document-Based Offloading (at ~60% context utilisation):**
When accumulated conversation history approaches 60% of your context window, stop dispatching new tasks. Write a progress document containing: which tasks are done (with their post-completion summaries), which tasks remain, what the current ready frontier is, and any failures or escalations pending. Register or update this as a document attached to the feature. Then start a fresh orchestration session that reads the progress document to resume. This prevents the quality degradation that occurs when orchestrating in a saturated context.

**Technique 3 — Single-Feature Scoping (session boundary rule):**
Each orchestration session handles exactly one feature. When all tasks for a feature are complete, write a feature completion summary and end the session. Begin the next feature in a new session. Do not attempt to orchestrate multiple features in one session because each feature's dependency graph, file scopes, and spec context are independent — combining them doubles context load without improving throughput.

### Phase 6: Close-Out

After all tasks reach a terminal state, the feature must be explicitly advanced through the remaining lifecycle stages. This phase ensures the feature is not left in `developing` with all work done.

1. **Verify all tasks are terminal.** Call `status(id: "FEAT-xxx")` and confirm all tasks show `done`, `not-planned`, or `duplicate`. If the attention item `"FEAT-xxx has N/N tasks done — ready to advance to reviewing"` is visible, all tasks are terminal.

2. **Transition the feature.** Call `entity(action: "transition", id: "FEAT-xxx", status: "reviewing")`. **INV-005** — Stage-transition gates are mandatory; the `entity` tool enforces all prerequisites (tasks terminal, required artefacts approved). Do not use `override: true` unless `override_reason` is provided and the prerequisite is genuinely inapplicable.

3. **Merge and delete the branch.** This step is mandatory regardless of how work was committed.
   - **If a worktree exists:** call `pr(action: "create", entity_id: "FEAT-xxx")` (if `require_github_pr: true`), then `merge(action: "check", entity_id: "FEAT-xxx")`, then `merge(action: "execute", entity_id: "FEAT-xxx")`. The `merge execute` tool deletes the branch automatically (`delete_branch` defaults to `true`) — verify it did so.
   - **Post-merge test verification:** After the merge, call `test(action: "run")` to
     confirm the test suite passes on `main`. Do not skip this step — it is the primary
     gate against introducing test failures into the main branch.
   - **If no worktree exists** (work was committed directly to the main branch): `merge` and `pr` will return `not_applicable`. In this case you **must** manually delete the feature branch: run `git branch -d feature/FEAT-xxx` (or `git branch -D` if git refuses due to squash history). **Do not skip this step.** Orphaned branches accumulate silently and are painful to audit later.

   > **Why this matters:** When branches are not deleted after merging, they appear unmerged at sprint boundaries, making it impossible for humans to tell what has and hasn't landed. Files committed to the branch but not included in the squash merge commit will be permanently lost. This has caused repeated incidents.

4. **Record a completion summary.** Call `finish` (or leave a completion note in the task) summarising what was implemented and any relevant observations.

4a. **Knowledge curation pass (primary curation mechanism for this feature).** Call `knowledge(action: "list")` with `status: "contributed"` and `tier: 2` to retrieve all tier 2 knowledge entries contributed during this feature's development. For each returned entry, apply one of three dispositions:
   - `knowledge(action: "confirm", id: "KE-xxx")` — entry proved accurate during the feature.
   - `knowledge(action: "flag", id: "KE-xxx", reason: "...")` — entry proved inaccurate or misleading.
   - `knowledge(action: "retire", id: "KE-xxx", reason: "...")` — entry has been superseded by architectural changes made during this feature.

   **Tier 3 entries:** Do NOT call `confirm` on tier 3 entries. Tier 3 entries are self-pruning and direct confirmation bypasses the promotion signal. Instead, for tier 3 entries that proved valuable, call `knowledge(action: "promote", id: "KE-xxx")` to elevate them to tier 2.

5. **Clean up worktrees.** If a worktree was created, run `worktree(action: "remove", entity_id: "FEAT-xxx")` after merging. Confirm with `git worktree list` that the worktree directory is gone.

6. **Verify branch is gone.** Run `git branch | grep FEAT-xxx` and confirm no output. If the branch still exists, delete it now. A feature is not truly closed out until its branch is absent from `git branch`.

7. **Cohort checkpoint.** If this batch has a merge schedule with multiple cohorts, check
   whether cohort-N+1 features exist. If so, return to Phase 0 for cohort N+1. Do not
   create cohort-N+1 worktrees until the cohort-N merge checkpoint is confirmed clean:
   no open feature branches from cohort N remain (`git branch | grep FEAT` returns only
   cohort-N+1 or later branches).

## Fast-Track Profile

When the feature's tier is `retro_fix` or `bug_fix`, or the batch is explicitly
marked fast-track, replace the full 6-phase procedure with this lightweight 3-phase flow.

### No-Stop Contract

You are in fast-track mode. You will NOT stop for confirmation at any point.
Use the `orchestrator` role (without `grep`/`search_graph`). Delegate all code
investigation to sub-agents — do not search, trace, or read source code.

The ONLY valid stop conditions are:
- All work is done (all features transitioned, all branches merged)
- A build failure requires code changes beyond the orchestrator's scope
- A missing dependency blocks all remaining tasks
- Spec ambiguity cannot be resolved without human input

If you find yourself about to produce a status table or summary, ask: is all work done?
If no, continue immediately without producing output. This contract is a structural
constraint — breaking it constitutes the Implicit Gate anti-pattern.

### Phase 0: Session Start Audit

Replaces Phase 0 (Cohort Setup) from the full procedure. Fast-track has no cohorts —
single-feature batches only.

> **Test-health gate:** Call `test(action: "verify")` before dispatching any
> sub-agents. If tests are failing on `main`, bail — the first job is to fix the
> test suite, not to start new work. The orchestrator may return to this feature
> once tests are green.

1. **Call `status()` to build authoritative entity state.** Call `status()` with the plan
   or batch ID. Identify:
   - Terminal tasks (done, not-planned, duplicate)
   - In-flight tasks (active, claimed)
   - Ready tasks (all dependencies satisfied)
   - Stuck tasks (queued with unmet dependencies, active but stale)

2. **Cross-reference task state against code existence.** For each task marked `done`,
   verify the described change exists in the codebase. If a task is `done` but its
   deliverable does not exist, flag it — the task state is unreliable.

3. **Check review-readiness drift before dispatch.** If a feature is in `reviewing` via
   override but still has non-terminal tasks, surface that as a blocker. Lifecycle
   status alone is not authoritative — verify task terminality independently.

4. **Check dirty working-tree state.** Run `git status`. Before implementing, classify
   any uncommitted changes as:
   - Current-scope changes (related to this feature)
   - Prior unfinished implementation work
   - Workflow/index metadata
   Do not proceed if the working tree contains unrelated implementation changes that
   would create merge conflicts.

5. **Identify ghost work.** For each ready task, check whether the described change
   already exists in the codebase. If yes, mark the task `not-planned` — do not claim
   or implement it. Ghost work wastes cycles and creates merge conflicts.

### Phase 1: Dispatch

Replaces Phase 1 (Read Dev-Plan), Phase 2 (Identify Parallel-Dispatchable), and
Phase 3 (Dispatch Sub-Agents) from the full procedure.

1. **Identify the ready frontier.** From the session-start audit, select all tasks in
   `ready` status with all `depends_on` tasks in a terminal state.

2. **Dispatch all ready frontier tasks in parallel.** Use
   `handoff(task_id: "TASK-xxx", role: "implementer-go")` for every task in the ready
   frontier. Dispatch them all in one batch — do not serialise independent tasks.

   **Integration note (P51):** The `role: "implementer-go"` parameter is explicit
   regardless of P51's handoff pipeline state. This ensures correctness even if P51
   default-role routing is not yet active.

   **Integration note (P44):** When `dispatch_task` arrives, this step changes from
   `handoff` + `spawn_agent` to `dispatch_task(task_id, category: "implementation")`.
   The behavioral rules (no stops, continuous polling, immediate dispatch of
   newly-unblocked tasks) remain identical — only the dispatch mechanism changes.

3. **Do NOT stop after dispatching.** If there is nothing to dispatch because tasks are
   still active, poll `status()` once per minute until something completes. Do not
   produce a status table or wait for confirmation.

4. **Dispatch newly-unblocked tasks immediately.** When a task completes, immediately
   dispatch any tasks that are now unblocked. Do not wait for the full batch to finish.

### Phase 2: Close-Out

Replaces Phase 4 (Monitor Progress), Phase 5 (Context Compaction), and Phase 6
(Close-Out) from the full procedure.

1. **Verify all tasks are terminal.** When all tasks reach a terminal state, confirm
   each feature can advance. Call `status()` and verify no non-terminal tasks remain.

2. **Dispatch review sub-agents.** Before transitioning any feature out of
   `developing`, dispatch at minimum one review sub-agent for each feature
   that modified source code (documentation-only features exempt):
   - `bug_fix` ≤5 files: one `reviewer-conformance` via `spawn_agent`
   - `retro_fix` with source changes: `reviewer-conformance` minimum
   - >5 files or other tiers: follow `orchestrate-review` Steps 3–6
   All reviewers use clean contexts. **Do not transition until findings are**
   **collated and no blocking findings remain.**

3. **Transition features.** Call `entity(action: "transition", id: "FEAT-xxx", status: ...)`
   to advance each feature through to `done` or `reviewing` as appropriate. Follow the
   full procedure's Phase 6 steps for merge, branch deletion, and worktree cleanup.

4. **Report completion.** Produce a completion summary listing features and their final
   status. This is the ONLY status output produced during a fast-track session — every
   other output is extraneous.

5. **Procedural compaction trigger.** Before close-out, estimate context utilisation. If
   at ~60%+ estimated utilisation and work remains for another feature, produce a
   compaction artefact using the U-shaped template and instruct the human to start a
   fresh session with it.

   **Integration note (P44):** This manual estimation step is a stopgap. P44 Phase 3b
   will replace it with automated token-count-based compaction triggers.

### Rules

These rules are structural constraints — violating any of them brings the fast-track
profile back into implicit full-orchestration behaviour:

- **NO stops at batch boundaries, milestone completions, or wave dispatches.** The only
  valid stop is the close-out completion report.
- **NO status tables until all work is done.** Status tables are pattern-matched as
  confirmation requests. Do not produce them.
- **NO ghost work.** Audit every ready task against the codebase before claiming. Mark
  already-completed work `not-planned`.
- **NO user summaries.** Trust `status()` output only. User-provided state summaries go
  stale and are not authoritative.
- **ALWAYS use `handoff(task_id, role: "implementer-go")` for sub-agent dispatch.**
  Never compose implementation prompts manually.

### Fast-Track Anti-Patterns

These are additions to the main Anti-Patterns section. They apply only when operating
under the fast-track profile.

#### Implicit Gate

- **Detect:** The orchestrator stops after a batch of work completes, presents a status
  table, and waits for confirmation.
- **BECAUSE:** Fast-track has no human gates. The no-stop contract prohibits stopping at
  any point other than the four valid stop conditions. An implicit gate wastes the
  human's time and adds latency the profile exists to remove.
- **Resolve:** After completing work, check whether more work exists. If yes, continue
  immediately. Do not produce summary output until all work is done.

#### Ghost Work Discovery

- **Detect:** The orchestrator discovers mid-implementation that a task's work already
  exists in the codebase.
- **BECAUSE:** The session-start audit should catch this. Implementing already-done work
  wastes cycles, creates unnecessary merge conflicts, and produces duplicate or
  conflicting changes.
- **Resolve:** Before claiming any task, verify its described change does not already
  exist in the codebase. If it does, mark `not-planned` and move on.

#### State Ambiguity Drift

- **Detect:** The orchestrator relies on a user's summary table or conversation context
  instead of calling `status()` to get authoritative entity state.
- **BECAUSE:** User summaries and conversation context go stale. Task YAML is the single
  source of truth for entity state. Trusting stale summaries leads to dispatching
  already-complete tasks, missing blockers, and operating on incorrect dependency graphs.
- **Resolve:** Call `status()` at session start. Trust the entity system, not the
  conversation. Reconcile any discrepancies before dispatching.

#### Milestone Pause

- **Detect:** The orchestrator treats "feature transitioned" or "wave dispatched" as a
  stop-and-report event, producing intermediate status output.
- **BECAUSE:** In fast-track, the only report that matters is the final completion
  report. Intermediate milestones are implementation details — pausing at them creates
  unnecessary friction and invites implicit gate behaviour.
- **Resolve:** After a milestone, immediately proceed to the next action. Do not produce
  a status table unless all work is done.

## Output Format

When a feature is completed (all tasks done and merged), produce a **Feature Completion
Summary** in this format:

```
Feature: FEAT-<nnn> — <name>
Batch: B<nnn>-<slug>
Tasks: N total, N done, N failed (retried and succeeded after escalation)

Summary: <2-3 sentences about what changed>

Notable decisions:
- <decision 1>
- <decision 2>

Tests: <test types run>, <result>
```

Include this in the close-out step via `finish(task_id: ..., summary: ...)`.

## Examples

See [examples-orchestrate-development.md](references/examples-orchestrate-development.md) for worked orchestration examples: serial dispatch anti-pattern, unmet dependency anti-pattern, and a full dependency-respecting parallel dispatch cycle.

## Evaluation Criteria

| # | Criterion | Weight |
|---|-----------|--------|
| 1 | All tasks in the ready frontier are dispatched in parallel (no unnecessary serialisation) | high |
| 2 | Post-completion summaries are applied immediately, full outputs are discarded | high |
| 3 | Context utilisation does not exceed 60% — offloading is triggered at threshold | required |
| 4 | Feature close-out is complete: transition, merge, branch delete, knowledge curation | required |
| 5 | Failed tasks are handled — retry or escalate — before dependents are dispatched | high |

## Questions This Skill Answers

- How do I orchestrate a feature's development?
- How do I identify which tasks can run in parallel?
- How do I handle a failed task?
- When should I offload context to a document?
- How do I close out a feature once all tasks are done?
- What do I do with the sub-agent output after a task completes?
- How do cohorts work in large batches?

## Related

- `implement-task` — what each sub-agent does when dispatched
- `review-code` — what reviewers check against
- `kanbanzai-agents` — context assembly, commit format, knowledge contribution
- `kanbanzai-workflow` — lifecycle transitions and stage gates
