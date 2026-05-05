# Design: Fast-Track Orchestration Profile

**Plan ID:** P52-fast-track-orchestration
**Status:** Shaping
**Parent:** P50 (Retrospective Fixes — May 2026)

## Overview

The fast-track pipeline was built to eliminate human gates for retro fixes and small independent features. The infrastructure works — zero human gates, straightforward entity transitions, parallel sub-agent dispatch. But the orchestrator's *behavior* doesn't match the infrastructure's intent. Even with explicit "use fast track — no stops, no breaks" instructions, the orchestrator halts at arbitrary breakpoints, treats summary output as implicit gates, and wastes cycles on ghost work that a session-start audit would catch.

These are not code defects. They're model-level patterns: the orchestrator is pattern-matching "summary output → wait for response" from training data, and the fast-track instruction isn't strong enough to override it. The fix is a dedicated lightweight orchestration profile — not a new skill, but a behavioral mode that the existing `orchestrate-development` skill switches into when the feature tier indicates fast-track.

## Goals

1. **No implicit gates.** The orchestrator must not stop unless it encounters a genuine blocker (build failure, missing dependency, spec ambiguity). "Batch complete" is not a stop condition.
2. **Session-start state audit.** A single `status()` call at session start must resolve entity state ambiguity — which tasks are actually done, which are stuck, which are ghost work.
3. **Ghost-work detection before implementation.** Before claiming a task, verify its code changes aren't already present. If the work is done, mark `not-planned` and move on — don't discover this mid-implementation.
4. **Lightweight context.** Skip cohort management, merge scheduling, and context offloading for fast-track features. These are designed for multi-feature batches.
5. **No stop-at-milestone behavior.** Completing a feature, transitioning a batch, or dispatching a wave is not a stop condition. Only stop when all work is done or a genuine blocker exists.

## Non-Goals

- Not changing the `orchestrate-development` skill's core procedure — this is an additive profile, not a rewrite
- Not changing the pipeline's context assembly (that's P51)
- Not building session-scoped context (that's P44)
- Not fixing the plan numbering reuse — tracked separately

## Design

### The fast-track behavioral profile

This is a **behavioral profile**, not a code change. It lives in the `orchestrate-development` skill as an alternate procedure branch that activates when the feature's tier (or batch's tier) indicates fast-track. It replaces the full 6-phase procedure with a 3-phase lightweight flow:

**Phase 0: Session Start Audit (replaces Cohort Setup)**

1. Call `status()` with the batch or plan ID. Do not proceed until you have confirmed:
   - Which tasks are terminal (done, not-planned, duplicate)
   - Which tasks are in-flight (active, claimed)
   - Which tasks are ready to dispatch
   - Which tasks are stuck (queued with unmet dependencies, active but stale)
2. Cross-reference against the dev-plan. If any task marked done doesn't exist in the code, flag it.
3. Identify ghost work: for each ready task, check whether the described change already exists. If yes, mark `not-planned` — do not claim or implement.

**Phase 1: Dispatch (replaces Read Dev-Plan + Identify Parallel-Dispatchable + Dispatch Sub-Agents)**

1. From the session-start audit, identify the ready frontier — all tasks in `ready` status with satisfied dependencies.
2. Dispatch all ready frontier tasks in parallel using `handoff(task_id: "TASK-xxx", role: "implementer-go")`.
3. Do NOT stop after dispatching. If there's nothing to dispatch because tasks are still active, poll `status()` once per minute until something completes.
4. When a task completes, immediately dispatch any newly-unblocked tasks.

**Phase 2: Close-Out (replaces Monitor Progress + Context Compaction + Close-Out)**

1. When all tasks are terminal, verify each feature can advance.
2. Transition features through to `done` or `reviewing` as appropriate.
3. Report completion: list features and their final status.

### What's removed vs. full orchestration

| Full orchestration | Fast-track profile | Why removed |
|---|---|---|
| Phase 0: Cohort Setup | Phase 0: Session Start Audit | No cohorts in fast-track |
| Merge schedule parsing | Skipped | Single-feature batches |
| Conflict analysis | Skipped | Single feature = no cross-feature conflicts |
| Phase 1: Read Dev-Plan | Merged into Dispatch phase | Same info from status audit |
| Phase 2: Identify Parallel-Dispatchable | Merged into Dispatch phase | Same logic, less ceremony |
| Phase 4: Monitor and Handle Failures | Merged into Close-Out | Continuous polling, no manual checkpoints |
| Phase 5: Context Compaction | Skipped | No multi-session for small features |
| Stop-at-60% rule | Removed | Contradicts "zero human gates" |

### Anti-patterns specific to fast-track

These are additions to the existing `orchestrate-development` anti-patterns:

- **Implicit Gate:** Stopping after a batch of work completes, presenting a status table, and waiting for confirmation. **Because:** Fast-track has no human gates. The only valid stop conditions are: all work done, build failure, missing dependency, or spec ambiguity. **Resolve:** After completing work, check whether more work exists. If yes, continue immediately. Do not produce summary output until all work is done.

- **Ghost Work Discovery:** Discovering mid-implementation that a task's work is already done. **Because:** The session-start audit should catch this. Implementing already-done work wastes cycles and creates merge conflicts. **Resolve:** Before claiming any task, verify its described change doesn't already exist in the codebase. If it does, mark `not-planned` and move on.

- **State Ambiguity Drift:** Relying on a user's summary table instead of calling `status()` to get authoritative entity state. **Because:** User summaries go stale. Task YAML is authoritative. **Resolve:** Call `status()` at session start. Trust the entity system, not the conversation.

- **Milestone Pause:** Treating "feature transitioned" or "wave dispatched" as a stop-and-report event. **Because:** In fast-track, the only report that matters is the final completion report. Intermediate milestones are implementation details. **Resolve:** After a milestone, immediately proceed to the next action. Do not produce a status table unless all work is done.

### Skill documentation changes

The `orchestrate-development` skill gains a new section after the Procedure:

```markdown
## Fast-Track Profile

When the feature's tier is `retro_fix` or the batch is explicitly marked fast-track,
replace the full procedure with this lightweight flow:

### Phase 0: Session Start Audit
[as designed above]

### Phase 1: Dispatch
[as designed above]

### Phase 2: Close-Out
[as designed above]

### Rules
- NO stops at batch boundaries, milestone completions, or wave dispatches
- NO status tables until all work is done
- NO ghost work — audit before claiming
- NO user summaries — trust `status()` output only
- ALWAYS use `handoff(task_id, role: "implementer-go")` for sub-agent dispatch
```

### Integration with P51

P51 fixes the pipeline so `handoff` defaults to sub-agent roles. The fast-track profile's dispatch phase explicitly uses `role: "implementer-go"` to ensure correctness regardless of P51's state.

### Integration with P44

When P44's `dispatch_task` arrives, the fast-track profile's Dispatch phase changes from `handoff` + `spawn_agent` to `dispatch_task(task_id, category: "implementation")`. The behavioral rules (no stops, no ghost work, no implicit gates) remain the same — only the dispatch mechanism changes.

## Findings from P50 Implementation

### Implicit gate behavior (second retrospective)

During the `state_modified` implementation, the orchestrator stopped three times at arbitrary breakpoints despite explicit fast-track instructions:

1. After presenting an analysis of remaining work (before any implementation)
2. After implementing two tasks (F4/T2 and F2/T5) — asked whether to continue
3. After transitioning all features to `reviewing` — produced a status table and stopped

Each stop was the model pattern-matching "summary output → wait for response" from training data. The fast-track instruction ("use fast track to complete without human gates") was present but not strong enough to override the pattern. This confirms that a behavioral profile needs more than a one-line instruction — it needs its own anti-patterns, explicit non-stop rules, and a different procedure structure that doesn't create natural breakpoints.

### Ghost work discovery

F4/T4 was `queued` but the work it described was already done in existing documentation. The orchestrator discovered this mid-implementation and correctly marked it `not-planned`, but the friction of discovering it mid-flow was unnecessary. A session-start audit of task descriptions against existing code would have caught it before any work began.

### Entity state ambiguity

The user's summary table ("F2: 4/5, F3: 3/3 ✅") didn't match the actual task state files. Some tasks showed `done` in YAML but weren't recognized as terminal by the gate system (F2/T5). Some showed `queued` but the user considered them complete (F5/T2-T4-T7). The batch entity (B49) wasn't registered in the entity system, so `status()` couldn't provide a batch-scoped dashboard. The orchestrator spent significant time reconciling user claims against YAML state.

### finish summary limit

The `finish` tool has a 500-character summary limit that isn't documented in the handler flow or surfaced clearly in the error message. The orchestrator hit this twice and had to learn the limit by failing. For fast-track close-out, where multiple tasks complete rapidly, this creates unnecessary retry friction.

## Alternatives Considered

### Just tell the orchestrator not to stop

**Idea:** Add stronger language to the fast-track instruction — "ABSOLUTELY NO STOPS" or similar emphasis.

**Reject:** P50 proved this doesn't work. The orchestrator received explicit fast-track instructions and still stopped three times. The problem is structural — the `orchestrate-development` procedure creates natural breakpoints between phases. A one-line instruction can't override the model's pattern-matching on these breakpoints.

### Create a separate fast-track skill

**Idea:** Write a new `orchestrate-fast-track` skill independent of `orchestrate-development`.

**Reject:** Most of the dispatch logic is the same (claim tasks, generate handoff prompts, dispatch in parallel, monitor completion). A separate skill would duplicate this content and drift out of sync. A profile within the existing skill is simpler and stays current.

### Wait for P44

**Idea:** Don't build a fast-track profile now — P44's `dispatch_task` will eliminate most of the breakpoints by collapsing dispatch into a single call.

**Reject:** P44 is a feasibility design, not an active build. Fast-track is being used now in P50. The behavioral fixes (session-start audit, no-implicit-gates rules, ghost-work detection) are valuable regardless of the dispatch mechanism.

## Dependencies

- No code dependencies — this is a skill documentation change
- P51 (handoff pipeline unification) is a companion, not a prerequisite
- P44 will eventually replace the dispatch mechanism, but the behavioral profile remains valid

## Open Questions

1. **How should the orchestrator detect fast-track mode?** By the feature's `tier` field (`retro_fix`), by a batch-level flag, or by an explicit instruction in the prompt? (Recommend: feature tier — it's already in the entity state and doesn't require additional configuration.)
2. **Should the session-start audit be automated?** A tool that cross-references task state against code existence would eliminate the manual ghost-work check. (Recommend: defer — valuable but not blocking; manual audit is sufficient for now.)
3. **Should the `finish` summary limit be documented in the error message?** The current message says "summary exceeds 500-character limit" but doesn't state the limit in the handler. (Recommend: yes — add the limit to the `finish` tool description so orchestrators know before they hit it.)

## Findings from System Instrumentation (May 2026 Retrospective)

### Stale binary caused false verification failures

During the `state_modified` implementation, 3 of 9 acceptance criteria appeared to fail when the code was correct. The running `kbz serve` binary was 3 days stale — new MCP tool parameters existed in source but not in the running server. The `server_info` tool detected the mismatch (`git_sha: unknown` vs. install record SHA) but the orchestrator didn't call it at session start. For fast-track, where features move from spec to done without review, a stale binary could mean an entire pipeline runs against old tool definitions.

**For P52:** The session-start audit should include a `server_info` call to verify the binary is current. If `git_sha` is unknown or doesn't match the install record, the orchestrator should warn the human before proceeding.

### Health checker false positives

The health dashboard reports 7 features as "orphaned reviewing" when they're actually done via override transitions. The checker flags missing review documents without checking whether the feature's current status is `done`. Additionally, done plans (P40, P48) show as "ready to close" attention items perpetually — there's no action to take but the dashboard can't suppress them. For fast-track, where override transitions are the norm (no review cycles), these false positives create noise that undermines trust in the health dashboard.

### Store consistency issues

At session start, `.kbz/` state files are frequently found modified or untracked — 10 modified index files and 12 untracked files on 2026-05-03 alone. The MCP server auto-commits workflow state changes, but if a tool call times out or is interrupted, the commit may not happen. Additionally, the SQLite cache can disagree with YAML state: tasks showing `done` via `entity(get)` but `ready` via `entity(list)`. For fast-track, where the orchestrator must trust entity state to determine the ready frontier, cache staleness can cause tasks to be dispatched twice or skipped entirely.

### Plan numbering reuse

Creating plans via `entity(action: "create")` reuses stale P1 numbers instead of assigning the next available (P51, P52). The root cause is that `listAllPlanIDs` scans batch plans, not strategic plans. For fast-track, where plans may be created frequently for small retro batches, incorrect plan numbering creates confusion and potential ID collisions.
