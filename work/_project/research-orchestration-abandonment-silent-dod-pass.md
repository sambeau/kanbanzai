# Investigation: Orchestration Abandonment → Silent DoD Pass

**Date:** 2026-05-10
**Trigger:** B64/P62 — batch marked done with incomplete features (2 queued tasks on F1, 9 queued on F2)

## Likely Root Cause

1. Orchestrator dispatches sub-agents for parallel task execution via `handoff`
2. Orchestrator loses context before completing the orchestration loop (context window exhaustion, session termination)
3. Remaining tasks stay `queued` — sub-agents may have completed work but `finish()` was never called
4. New orchestrator session starts, sees partial state, cannot distinguish "abandoned mid-flight" from "never started"
5. Override to `done` bypasses child-completion validation

## What to Investigate

### Investigation 1: Hard gate — block done with non-terminal children

**Problem:** Features transition to `done` (with or without override) while child tasks are queued/active. Batches transition to `done` while child features are non-terminal.

**Question to answer:** Where in `entity(action: transition)` or the batch lifecycle advance does this check belong? Is it a single validation function shared by feature and batch transitions, or two separate paths?

**Rough scope:** `internal/service/entity_transition.go` or equivalent — find the `reviewing → done` and batch `→ done` transition logic, add child-terminal precondition.

### Investigation 2: Orchestration session tracking

**Problem:** No record of whether an orchestration session completed normally or was abandoned. A relaunched orchestrator has no way to know it's resuming interrupted work.

**Questions to answer:**
- What would a minimal session marker look like? `{batch_id, started_at, closed_at?, tasks_dispatched[]}`
- Where would it live? `.kbz/sessions/`? In the entity store?
- How would `status` and `next` surface abandoned sessions?

### Investigation 3: Handoff dispatch log

**Problem:** `handoff` dispatches a sub-agent but the orchestrator has no structured record of which tasks were handed off, to which sub-agent session, and whether they completed.

**Questions to answer:**
- Does `handoff` already record anything we can query?
- Could `finish` cross-reference against dispatch records to detect completion gaps?
- Could the orchestration skill's startup procedure include "check for abandoned dispatches"?

### Investigation 4: Queued → Ready miscount in dashboard

**Problem:** The project-level `status` dashboard reports `queued` tasks as `ready`. This masks the true state — a relaunched orchestrator sees "12 tasks ready to work" instead of "11 tasks never started, feature possibly abandoned."

**Questions to answer:** Where does the dashboard aggregate task counts? Is this a deliberate mapping ("queued means ready to claim") or a bug?

## Priority

| # | Investigation | Why first |
|---|--------------|-----------|
| **1** | Hard gate on done with non-terminal children | Safety net — prevents recurrence regardless of root cause |
| **2** | Queued → Ready miscount | Quick fix with high diagnostic value for future orchestrators |
| **3** | Orchestration session tracking | Structural fix — makes abandonment visible |
| **4** | Handoff dispatch log | Deepest fix — enables smart resumption, not just detection |
