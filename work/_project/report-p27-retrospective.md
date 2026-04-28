# Retrospective: P27 — Document Intelligence Adoption

**Plan:** P27-doc-intel-adoption
**Period:** 2026-04-22
**Scope:** Full plan delivery — Document Intelligence Adoption
**Author:** Orchestrating agent (Claude Sonnet 4.6) + captured session signals

---

## Executive Summary

P27 delivered the Document Intelligence Adoption plan, making `doc_intel` and knowledge systems
actively used by agents through mandated corpus consultation at design time, corpus completeness
improvements, and closing the knowledge feedback loop.

Four significant operational friction points were encountered during the session and captured for
this report. None are unique to P27 — all four are systemic workflow gaps that will recur on any
plan that: (a) resumes mid-lifecycle from a prior session, (b) uses `decompose` to create tasks,
(c) operates on a repository with many existing worktrees, or (d) begins with a handoff that
omits pre-dispatch setup steps. All four are actionable.

---

## Issue 1: Plan lifecycle gap — `proposed` → `active` (Significant)

### What happened

When the session resumed, the plan was at `proposed` but its features had already passed through
`designing`, `specifying`, and `dev-planning` in a prior session. Attempting to advance the plan
directly to `active` failed: the only valid transitions from `proposed` are `designing`,
`superseded`, and `cancelled`.

The workaround was to step the plan through `designing` first, then use an override to reach
`active`. This required two extra transition calls and an override justification for a situation
that was entirely legitimate — the plan's features were simply ahead of the plan's own status.

### Root cause

The plan lifecycle state machine does not have a concept of "features already in-flight". It
models the plan as always progressing forward with its features in lockstep. Resuming mid-lifecycle
from a prior session breaks this assumption.

### Consequences

- Two wasted transition calls and a gate override on every cross-session plan resume
- The override audit trail makes it appear that a gate was bypassed for a bad reason, when in
  fact the work was already complete
- Agents not familiar with this gap will attempt `proposed → active` directly, encounter an error,
  and may attempt multiple invalid recovery paths before finding the workaround

### Recommendation

Add a direct `proposed → active` transition that bypasses the `designing` intermediate step, or
document a "resume mid-lifecycle plan" procedure in `kanbanzai-workflow` SKILL.md that explicitly
covers the step-through-designing workaround and explains why the override is safe in this context.

---

## Issue 2: `decompose` creates tasks but no dev-plan document (Significant)

### What happened

Tasks were created via `decompose(action: apply)`. No dev-plan document was registered as a
side effect. When the session later attempted to advance features from `dev-planning` to
`developing`, the gate check failed: an approved dev-plan document is required.

Every feature whose tasks were created via `decompose` required a gate override to enter
`developing`. With four features in the plan, this meant four override calls.

### Root cause

`decompose(action: apply)` creates task entities but does not register a corresponding dev-plan
document. The gate that guards `dev-planning → developing` is unaware that `decompose` was used
as the planning mechanism — it only checks for a registered and approved dev-plan doc.

### Consequences

- Every plan that uses the `decompose` workflow hits this gate on every feature
- The required override implies the process was broken, when it was actually followed correctly
- If the agent does not recognise the root cause, it may attempt to manually write a dev-plan
  document mid-session, wasting time on boilerplate that serves no real purpose

### Recommendation

Two viable fixes:

1. **Register at apply time:** `decompose(action: apply)` registers a skeleton dev-plan document
   and marks it auto-approved, representing the task breakdown as the plan artifact.
2. **Dev-plan-exempt path:** Add a `decomposed: true` flag to features and relax the gate
   requirement when this flag is set, so the gate passes without a document override.

Option 1 is lower-risk and produces a useful audit artifact. Either fix would eliminate the
override requirement for the decompose workflow entirely.

---

## Issue 3: `worktree(action: create)` times out under large worktree counts (Significant)

### What happened

`worktree(action: create)` was called for four features at plan start — first in parallel, then
sequentially. All calls timed out. The repository had approximately 34 existing worktrees at
the time, which likely caused git lock contention or `git worktree list` slowdown that pushed
the operation past the tool's internal timeout.

The workaround was to fall back to `terminal` with `git worktree add` and wire up the worktree
record separately. This unblocked plan start but added friction at exactly the point where agents
are most likely to be dispatching in parallel.

### Root cause

The `worktree` tool does not appear to implement a retry or lock-aware strategy. With many
existing worktrees, git's internal locking (`.git/worktrees/*/lock`) or the overhead of
`git worktree list` serialisation causes the operation to stall.

### Consequences

- Hard blocker at plan start when multiple features need worktrees before dispatch
- Forces use of `terminal` as a manual fallback, which bypasses the tool's entity tracking
- Records created via the fallback path may be inconsistent with records created via the tool,
  depending on how wiring is done

### Recommendation

- Investigate git lock contention with large worktree counts; profile `worktree(action: create)`
  against repositories with 30+ existing worktrees
- Add a configurable timeout and automatic retry with backoff
- Document the `terminal` + manual-wiring fallback in the `worktree` tool description so agents
  have an explicit escape hatch rather than discovering it by accident

---

## Issue 4: "Ready to develop" handoff omits invisible setup steps (Moderate)

### What happened

The session began with a human handoff: "features are in dev-planning, tasks are created, ready
to develop." This implies the agent can immediately begin dispatching sub-agents. In practice,
four blocking setup steps were required first:

1. Commit orphaned working-tree changes from the prior session
2. Advance the plan from `proposed` to `active` (see Issue 1)
3. Override the dev-plan gate for all four features (see Issue 2)
4. Create a worktree for each of the four features (see Issue 3)

None of these steps were mentioned in the handoff summary or in the `kanbanzai-getting-started`
and `kanbanzai-workflow` skills. The agent had to discover each blocker in sequence, adding
latency before any real work could begin.

### Root cause

The getting-started and workflow skills describe the forward path (design → spec → plan → develop)
but do not document the "resume" case — a plan whose features are already partway through the
lifecycle at the start of a new session.

### Consequences

- Significant pre-dispatch latency on every cross-session plan resume
- Each invisible step is a potential failure point (timed-out worktree, missed commit, gate
  override not applied) that can surface as a confusing error later in the session
- Human handoff messages cannot reliably convey all required context, so agents must reconstruct
  the setup sequence from first principles on every resume

### Recommendation

Add a **"Resume plan" checklist** to `kanbanzai-getting-started` SKILL.md and
`kanbanzai-workflow` SKILL.md. The checklist should cover:

1. Run `git status` and commit any orphaned changes
2. Check the plan's current lifecycle state; if `proposed`, step through `designing` and override
   to `active`
3. For each feature, check whether a dev-plan doc is registered; if not (decompose workflow),
   apply the gate override
4. Confirm a worktree exists for each in-flight feature; create any missing ones

This checklist converts four invisible failure modes into a single, scannable procedure.

---

## What Worked Well

No explicit "worked well" signals were captured for P27 in this session. The friction items above
dominated session attention. Signals from other plans (P25, P26) continue to reflect that
sub-agent parallelism with disjoint file scopes and the doc-pipeline orchestration pattern
produce clean results when the pre-dispatch setup phase completes successfully.

---

## Recommendations for Next Iteration

| Priority | Recommendation | Addresses |
|----------|---------------|-----------|
| High | Add `proposed → active` direct transition or document the step-through-designing workaround | Issue 1 |
| High | Register auto-approved skeleton dev-plan at `decompose(action: apply)` time | Issue 2 |
| High | Investigate and fix `worktree(action: create)` timeout under large worktree counts | Issue 3 |
| Medium | Add "Resume plan" checklist to getting-started and workflow skills | Issue 4 |

Issues 1, 2, and 4 compound each other: every cross-session plan resume triggers all three.
Addressing Issues 1 and 2 would eliminate the override calls; addressing Issue 4 would ensure
agents know to expect them in the meantime. Issue 3 is independently blocking at plan scale.