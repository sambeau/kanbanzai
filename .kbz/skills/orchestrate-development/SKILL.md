---
name: orchestrate-development
description:
  expert: "Development-stage orchestration procedure for coordinating
    parallel task dispatch with dependency-respecting sequencing,
    sub-agent lifecycle management, failure recovery, and context
    compaction across multi-task implementation sessions"
  natural: "Coordinates implementing a feature's tasks — figures out
    what can run in parallel, dispatches sub-agents, handles failures,
    and keeps context from growing out of control"
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

**Parallel dispatch:**
- **dispatch batch** — the set of tasks whose dependencies are all satisfied, dispatched simultaneously in one cycle
- **parallel-dispatchable** — a task eligible for dispatch because every task in its `depends_on` list has reached `done`
- **sub-agent spawn** — creating a new agent instance (via `handoff`) to execute a specific task in isolation
- **dispatch window** — the period between identifying a dispatch batch and receiving all sub-agent outcomes
- **file scope boundary** — per-agent file ownership declared at dispatch to prevent write conflicts between parallel agents

**Dependency ordering:**
- **dependency chain** — a sequence of tasks where each requires the prior one's output before it can begin
- **unmet dependency** — a predecessor task that has not reached `done`; any task depending on it must not be dispatched
- **topological order** — the dispatch sequence derived from declared task dependencies; parents before children
- **ready frontier** — the set of tasks with zero unmet dependencies at the current point in execution

**Context compaction:**
- **context utilisation** — approximate percentage of the context window consumed by accumulated conversation history
- **post-completion summary** — a 2–3 sentence reduction of a sub-agent's outcome, retaining only task ID and key results
- **document-based offloading** — writing accumulated orchestration progress to a registered document and starting a fresh session
- **single-feature scope** — constraining each orchestration session to one feature's tasks to prevent context growth across feature boundaries
- **compaction trigger** — the condition (task completion event or utilisation threshold) that initiates a compaction technique
- **context ceiling** — the practical upper bound on useful context; output quality degrades well before the hard token limit

**Failure recovery:**
- **task rework** — returning a failed task to `ready` status with updated guidance for a second attempt
- **failure escalation** — reporting a task failure to the human when automated recovery is not viable
- **retry budget** — the maximum number of re-dispatch attempts for a single task before escalating (typically one)
- **dispatch checkpoint** — a progress snapshot saved before dispatching, enabling recovery if the session is interrupted

## Anti-Patterns

### Dispatching With Unmet Dependencies
- **Detect:** A task is dispatched while one or more entries in its `depends_on` list have not reached `done`
- **BECAUSE:** The sub-agent will lack outputs from predecessor tasks — files, interfaces, type definitions — and will either guess at them (creating integration failures) or stall. Dependency violations are the most common cause of wasted dispatch cycles
- **Resolve:** Before each dispatch cycle, query the status of every task in the feature. Build the ready frontier from tasks with all dependencies satisfied. Only dispatch from the ready frontier

### Full-Output Retention
- **Detect:** The orchestrator keeps the complete sub-agent conversation or full diff output in its context after a task completes
- **BECAUSE:** Sub-agent outputs are verbose — diffs, test output, reasoning traces. Retaining them consumes context budget rapidly, leaving insufficient room for later dispatch cycles. After three or four retained outputs, the orchestrator's own reasoning quality degrades
- **Resolve:** Apply post-completion summary immediately: reduce each sub-agent outcome to 2–3 sentences plus the task ID. Discard the full output

### Multi-Feature Session
- **Detect:** The orchestrator attempts to dispatch tasks from two or more features within a single session
- **BECAUSE:** Each feature has its own dependency graph, file scopes, and spec context. Cross-feature orchestration doubles the context load without improving parallelism because inter-feature dependencies are handled at the feature lifecycle level, not the task level
- **Resolve:** Complete all tasks for one feature, write a progress summary, then begin a new session for the next feature. Use single-feature scope

### Fire-and-Forget Dispatch
- **Detect:** Sub-agents are spawned but their outcomes are never checked — the orchestrator assumes success and dispatches the next batch
- **BECAUSE:** A failed task whose failure is not detected will cause all dependent tasks to fail in turn. Compounding failures waste more time than waiting for confirmation
- **Resolve:** After each dispatch window closes, check every dispatched task's status. Verify `done` before moving dependent tasks to the ready frontier

### Serial Dispatch of Independent Tasks
- **Detect:** Tasks with no dependency relationship are dispatched one at a time, each waiting for the prior to complete
- **BECAUSE:** Serial dispatch of independent tasks wastes wall-clock time. If tasks A and B have no dependency edge, dispatching B only after A completes adds A's entire duration as unnecessary latency
- **Resolve:** Identify the full dispatch batch — all parallel-dispatchable tasks — and dispatch them simultaneously. Use the `conflict` tool to verify file scope boundaries do not overlap before parallel dispatch

### Ignoring Failure Signals
- **Detect:** A sub-agent reports a failure (test failures, missing context, spec ambiguity) and the orchestrator re-dispatches the same task without updating guidance
- **BECAUSE:** Re-dispatching without addressing the root cause produces the same failure. The retry budget exists to prevent infinite loops — one re-attempt with updated context, then escalate
- **Resolve:** Read the failure details. IF the failure is a missing file or context gap → update the handoff with the missing information and re-dispatch (counts as one retry). IF the failure is a spec ambiguity or design gap → escalate to the human

### Context Bloat Without Offloading
- **Detect:** Context utilisation exceeds 60% and the orchestrator continues dispatching without compaction
- **BECAUSE:** Beyond ~60% utilisation, the orchestrator's ability to track task states, remember dependency relationships, and make correct dispatch decisions degrades measurably. Pushing further produces subtle errors — wrong dispatch order, missed failures, lost progress
- **Resolve:** When context utilisation approaches 60%, trigger document-based offloading: write current progress to a registered document and start a fresh session

### Assigning Multiple Large-File Tasks to One Sub-Agent
- **Detect:** A sub-agent is dispatched with more than one task that involves reading or rewriting source files longer than ~300 lines.
- **BECAUSE:** Full-file rewrites embed entire file content in terminal tool calls. An agent accumulates the content of multiple large files in its context before completing the first task, saturating its context window and degrading output quality for later tasks. This was observed in the P24 pipeline.
- **Resolve:** For features with more than three tasks involving large source files, dispatch one sub-agent per task. Use the sizing rule above to identify when per-task isolation is required.

### Manual Prompt Composition
- **Detect:** The orchestrator writes a sub-agent prompt by hand rather than calling `handoff(task_id: "TASK-xxx")`.
- **BECAUSE:** Manual prompts omit the graph project name (so codebase-memory-mcp tools are unavailable to the sub-agent), omit knowledge entries relevant to the task, and omit structured spec sections assembled by the context pipeline. This was the root cause of zero graph tool usage in the P24 pipeline.
- **Resolve:** Always call `handoff(task_id)` to generate sub-agent prompts. The handoff tool assembles spec sections, knowledge constraints, file paths, role conventions, tool hints, and the graph project name. Supplement the generated prompt with file scope boundaries and parallel ownership constraints, but never replace it.

## Checklist

```
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
- [ ] Feature advanced beyond developing
```

## Procedure

### Phase 1: Read the Dev-Plan

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

**Rule:** Always use `handoff(task_id: "TASK-xxx")` to generate sub-agent prompts. Never compose implementation prompts manually — manual composition silently omits graph project context, knowledge entries, and spec sections.

0. **Before creating worktrees**, verify `.kbz/local.yaml` has `codebase_memory.graph_project` set
   (e.g. `Users-alice-Dev-myrepo`). If missing, add it now — worktrees created without it will not
   inject Code Graph context into sub-agent handoffs. Once set in local config, the `worktree` tool
   picks it up automatically and you never need to pass `graph_project` explicitly.
1. For each task in the dispatch batch, generate a sub-agent prompt using `handoff(task_id: "TASK-xxx")`.
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

2. **Transition the feature.** Call `entity(action: "transition", id: "FEAT-xxx", status: "reviewing")`.

3. **Merge and delete the branch.** This step is mandatory regardless of how work was committed.
   - **If a worktree exists:** call `pr(action: "create", entity_id: "FEAT-xxx")` (if `require_github_pr: true`), then `merge(action: "check", entity_id: "FEAT-xxx")`, then `merge(action: "execute", entity_id: "FEAT-xxx")`. The `merge execute` tool deletes the branch automatically (`delete_branch` defaults to `true`) — verify it did so.
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

## Output Format

At session end, produce a progress summary:

```
Feature: FEAT-xxx — <feature title>
Status: <all tasks done | in progress | blocked>

Completed this session:
- TASK-xxx: <2–3 sentence summary>
- TASK-yyy: <2–3 sentence summary>

Remaining:
- TASK-zzz: ready (depends on: none)
- TASK-aaa: blocked (depends on: TASK-bbb — in progress)

Failures/Escalations:
- TASK-ccc: <failure reason> — escalated to human

Context note: <offloaded at N% utilisation / completed in single session>
```

## Examples

### BAD: Serial dispatch with full output retention

```
Dispatched TASK-101 (add user model).
Waited for completion. Full output:
  [400 lines of diff, test output, reasoning trace]
Dispatched TASK-102 (add user API handler).
Waited for completion. Full output:
  [350 lines of diff, test output, reasoning trace]
Dispatched TASK-103 (add user validation).
  ... context window nearly full, responses becoming incoherent ...
```

WHY BAD: TASK-102 and TASK-103 had no dependency on each other — they should have been dispatched in parallel. Full sub-agent outputs were retained verbatim, consuming context budget that should have been available for later dispatch cycles. By the third task, context utilisation was too high for reliable orchestration. No post-completion summaries were applied.

### BAD: Dispatching with unmet dependencies

```
Ready frontier: TASK-201, TASK-202, TASK-203
Dispatched all three.
TASK-203 failed — it needed the interface defined by TASK-201, which
was still in progress when TASK-203 started.
Re-dispatched TASK-203. Failed again — TASK-201 still not done.
Re-dispatched TASK-203 a third time.
```

WHY BAD: TASK-203 had a dependency on TASK-201 but was treated as parallel-dispatchable. The dependency was either not declared (a decomposition defect) or not checked (an orchestration defect). Multiple re-dispatches without addressing the root cause wasted the retry budget and produced three identical failures.

### GOOD: Dependency-respecting parallel dispatch with compaction

```
Feature: FEAT-055 — Webhook delivery system
Phase 1: Read dev-plan. 6 tasks total.

Cycle 1 — Ready frontier: TASK-301 (data model), TASK-302 (config schema)
  No shared files — dispatched in parallel.
  TASK-301 done: Added webhook event model with 4 fields, migration included.
  TASK-302 done: Config schema with retry policy fields, validated by tests.
  [Full outputs discarded, summaries retained]

Cycle 2 — Ready frontier: TASK-303 (dispatcher, depends on 301+302),
  TASK-304 (delivery log, depends on 301)
  File scope check: TASK-303 owns dispatcher.go, TASK-304 owns
  delivery_log.go. No overlap — dispatched in parallel.
  TASK-303 done: Dispatcher with exponential backoff per config, 8 tests.
  TASK-304 done: Delivery log with per-attempt recording, 5 tests.

Cycle 3 — Ready frontier: TASK-305 (retry handler, depends on 303+304),
  TASK-306 (integration test, depends on 303+304)
  Context utilisation ~55% — proceeding but monitoring.
  Dispatched in parallel.
  TASK-305 done: Retry handler wired to dispatcher and log. 6 tests.
  TASK-306 done: End-to-end test covering dispatch → retry → log.

All 6 tasks done. Feature completion summary written.
```

WHY GOOD: Each dispatch cycle only includes tasks whose dependencies are satisfied. File scope boundaries are checked before parallel dispatch. Post-completion summaries replace full outputs after every completion. Context utilisation is monitored. The feature is completed in a single session because compaction kept the context manageable.

## Evaluation Criteria

1. Were tasks dispatched only when all their dependencies had reached `done`? Weight: required.
2. Were independent tasks dispatched in parallel rather than serially? Weight: required.
3. Was post-completion summarisation applied after every task completion? Weight: required.
4. Was context utilisation monitored, with document-based offloading triggered before quality degradation? Weight: high.
5. Was each orchestration session scoped to a single feature? Weight: high.
6. Were file scope boundaries checked and communicated to sub-agents before parallel dispatch? Weight: high.
7. Were task failures handled with root-cause analysis before re-dispatch or escalation? Weight: high.
8. Was a progress summary produced at session end? Weight: moderate.

## Questions This Skill Answers

- How do I coordinate implementing a feature's tasks?
- Which tasks can I dispatch in parallel right now?
- How do I prevent context from growing out of control during multi-task orchestration?
- When should I offload progress to a document and start a fresh session?
- What do I do when a sub-agent fails a task?
- How do I check for file conflicts between parallel tasks?
- What information should I include when dispatching a sub-agent?
- How do I track progress across dispatch cycles?
- When should I escalate a failure to a human instead of retrying?
- Why should I orchestrate one feature at a time?