# Workflow Completeness — Design

**Status:** Draft
**Plan:** P13-workflow-flexibility
**Date:** 2026-04-02
**Basis:**
- [Kanbanzai user feedback report](../reports/Kanbanzai%20user%20feedback%20report.txt)
- [V3.0 Feedback and Gap Analysis](../reports/v3-feedback-gap-analysis.md)
- [Post-P9 Feedback Analysis](../research/post-p9-feedback-analysis.md)
- Empirical evidence: P14 entity-names branch (15 commits, fully implemented, never merged); P18 auto-commit-and-doc-ops branch (2 commits, never merged); 8 plans with all work done but status never advanced to `done`; 44 stale worktrees accumulating on disk.

---

## Problem and Motivation

Kanbanzai has a "last mile" problem. Work gets done — tasks complete, code ships, tests
pass — but the workflow entities that track the work are not closed out. Features sit in
`developing` with 10/10 tasks done. Plans stay `active` or `proposed` after every feature
finishes. Worktree branches with completed implementations are never merged. The result is
a system that accurately tracks the *start* of work but silently loses track of *completion*.

This is not a theoretical concern. In a single audit session on 2026-04-02:

- **8 plans** had all tasks done but statuses ranging from `proposed` to `active` — none
  were `done`. They required manual transition through 3–4 lifecycle states each.
- **2 feature branches** contained fully implemented, tested work (17 commits total) that
  had never been merged to main. One required rebasing across 124 commits and resolving
  3 merge conflicts to recover.
- **44 worktrees** with associated git branches had accumulated on disk despite all their
  work already being in main.

The root causes are:

### 1. No detection of "all tasks done"

When the last task in a feature completes via `finish()`, nothing happens to the parent
feature. The `status` tool does not surface "all tasks done — ready to advance?" as an
attention item. The `health` check for feature-child consistency flags features in early
states (`proposed`, `designing`, `specifying`, `dev-planning`) with all tasks done, but
**misses `developing`** — the state where this problem actually occurs. There is no
plan-level equivalent at all.

### 2. No close-out procedure in any skill

The `orchestrate-development` skill says "write a feature completion summary and end the
session" — but never mentions transitioning the feature to `reviewing`, creating a PR,
merging the branch, or cleaning up the worktree. The `kanbanzai-agents` skill covers
task-level `finish()` but has zero coverage of feature-level completion. The
`kanbanzai-workflow` skill has a "before completing a feature" checklist but nothing
triggers it — an agent has to remember to check it after the last task finishes, and the
system provides no signal that this moment has arrived.

### 3. The merge/PR path is unusable for direct-to-main workflows

Both `merge` and `pr` tools hard-fail without a worktree:

```
Cannot execute merge for FEAT-xxx: no worktree exists for this entity.
```

When implementation is committed directly to the default branch — which is how most
solo development and small changes work in practice — the entire merge/PR toolchain is
a dead end. The lifecycle *can* advance to `done` without merging (the gates don't
enforce it), but this is an undocumented side-door, not a supported workflow.

### 4. Document gap checks produce false positives

Every feature belonging to a plan with an approved specification still shows "missing
specification" in status dashboards and health checks. The doc gap logic only queries
documents owned directly by the feature (`ListDocumentsByOwner(featureID)`) and never
walks up to the parent plan. In a plan with 8 features sharing one spec, this produces
8 false-positive warnings that drown out real attention items.

### 5. Stuck tasks after agent crashes

When an agent claims a task (moving it to `active`) and then crashes or is interrupted,
the task is stuck. `active → ready` is not a valid transition. The only escapes are
`needs-rework` (which implies the work is wrong, not that the agent died) or `not-planned`
(which abandons the task entirely). There is no "unclaim" mechanism.

### 6. Decomposition granularity

The `generateProposal` function hardcodes a one-AC-per-task strategy. When a specification
has 12 tightly-coupled acceptance criteria, this produces 13 tasks (12 + 1 test task) where
3–4 vertical slices would be more appropriate. Three of five requested AC formats are now
supported (checkboxes, numbered lists, bold-identifiers), but table and Given/When/Then
formats are not. The more impactful concern is the absence of any AC grouping logic.

---

## Design

The design is organised into six features, ordered by impact. Features 1–3 address the
core "last mile" problem. Features 4–6 address related workflow friction identified in the
same feedback round.

### Feature 1: Completion Detection and Attention Items

**Problem addressed:** §1 — no detection of "all tasks done"

Add three new attention items to the `status` tool's synthesis logic:

**Feature-level:** When all child tasks of a feature are in terminal state (`done`,
`not-planned`, `duplicate`) and the feature is in `developing` or `needs-rework`, surface:

```
FEAT-xxx has N/N tasks done — ready to advance to reviewing
```

**Plan-level:** When all features in a plan are in terminal state (`done`, `cancelled`,
`superseded`) and the plan is not `done`, surface:

```
Plan Pxx has all N features done — ready to close
```

**Stale developing:** When a feature has been in `developing` for more than 48 hours with
all tasks in terminal state, escalate the attention item severity by prepending "⚠️ STALE:"
to make it visually distinct from fresh completions.

Extend `health` check `CheckFeatureChildConsistency` to include `developing` and
`needs-rework` in the set of states flagged when all children are terminal. Currently
it only flags `proposed`, `designing`, `specifying`, and `dev-planning`.

Add `CheckPlanChildConsistency` health rule: warn when all features in a plan are
terminal but the plan is not `done`.

### Feature 2: Close-Out Procedure in Skills

**Problem addressed:** §2 — no close-out procedure in any skill

Update three skill files to cover the feature-to-done lifecycle:

**`orchestrate-development` (`.kbz/skills/orchestrate-development/SKILL.md`):**

Add a "Phase 6: Close-Out" after the current Phase 5, with the procedure:

1. Verify all tasks are in terminal state.
2. Transition feature to `reviewing` (or `done` if review is not required).
3. If a worktree exists: create PR via `pr(action: "create")`, then
   `merge(action: "check")` followed by `merge(action: "execute")`.
4. If no worktree exists (direct-to-main): transition directly.
5. Record a feature completion summary.

Add a checklist item: `- [ ] Feature advanced beyond developing`.

**`kanbanzai-agents` (`.agents/skills/kanbanzai-agents/SKILL.md`):**

Add a "Feature Completion" section after the existing "Finishing Tasks" section,
covering: feature transition, PR creation, merge, worktree cleanup. Reference
the `orchestrate-development` skill for the full procedure.

**`kanbanzai-workflow` (`.agents/skills/kanbanzai-workflow/SKILL.md`):**

Add a trigger to the "Before completing a feature" checklist: "This checklist
applies when `status` shows an attention item indicating all tasks are done."

### Feature 3: Direct-to-Main Workflow

**Problem addressed:** §3 — merge/PR path unusable without worktrees

Add a "direct commit" path for features that were implemented on the default branch
without a worktree:

**Option A — `merge` with skip semantics:** When `merge(action: "check")` is called
for an entity with no worktree, instead of erroring, return a result indicating no
merge is needed:

```json
{
  "status": "not_applicable",
  "reason": "no worktree exists — work was committed directly to the default branch",
  "recommendation": "advance the feature lifecycle directly"
}
```

`merge(action: "execute")` with no worktree would similarly return success with a
`"skipped"` status rather than failing.

**Option B — lifecycle gate awareness:** Add a `direct_commit` flag to the feature
entity (set during `entity update` or inferred from the absence of a worktree). When
set, the `reviewing → done` gate does not expect merge artifacts.

The recommended approach is **Option A** — it requires no schema changes, works with
existing tool invocations, and the skill procedures naturally handle both paths ("if
worktree exists, merge; otherwise, advance directly").

### Feature 4: Document Inheritance for Gap Checks

**Problem addressed:** §4 — false-positive doc gap warnings

When checking whether a feature has a required document type (specification, design,
dev-plan), fall back to the parent plan's documents if the feature has none of its own.

The change affects three code paths:

1. `docGapsAction` in `doc_tool.go` — after `ListDocumentsByOwner(featureID)` returns
   no docs of the required type, also check `ListDocumentsByOwner(planID)`.
2. `synthesisePlan` / `synthesiseFeature` in `status_tool.go` — same fallback when
   computing `has_spec` and `has_dev_plan`.
3. `generateFeatureAttention` in `status_tool.go` — suppress "missing specification"
   when the parent plan has an approved spec.

A feature with its own specification always takes precedence over the plan's. The
inheritance is read-only — registering a document on the plan does not create records
on child features.

### Feature 5: Task Crash Recovery

**Problem addressed:** §5 — stuck tasks after agent crashes

Add `ready` to the valid transitions from `active` in the task lifecycle map:

```
active → ready    (unclaim / crash recovery)
```

This is a simple transition map change in `internal/validate/lifecycle.go`. No gate
checks are needed — if an agent wants to unclaim a task, it should be allowed to.

The `status` tool should surface stuck active tasks as an attention item: when a task
has been in `active` for more than 24 hours with no git activity on its parent feature's
worktree branch (if one exists), surface:

```
TASK-xxx has been active for >24h with no recent commits — may need unclaim
```

This requires implementing `checkGitActivitySince` in `internal/health/`, which is
currently a stub that always returns `false`.

### Feature 6: Decomposition Grouping

**Problem addressed:** §6 — one-AC-per-task granularity

This feature has two parts:

**6a. AC Grouping in `generateProposal`:**

After extracting acceptance criteria, group them by their parent section (`parentL2`
from the spec structure). ACs under the same section heading are likely to be
implementation-coupled. When a section contains 2–4 ACs, propose a single task covering
all of them. When a section contains 5+ ACs, apply the current one-per-task strategy.

The grouping is a heuristic applied during `propose` — the `review` step can still
split or merge tasks, and the human can reject the proposal. The `one-ac-per-task`
guidance rule becomes `group-by-section` when sections are detected.

Add a `covers` field to `ProposedTask` that lists the AC identifiers grouped into
that task, for traceability.

**6b. Additional format support (lower priority):**

Add table row parsing: detect markdown tables in acceptance-criteria sections and
extract each row as a criterion. Given/When/Then parsing is deferred — the bold-
identifier format (`**AC-01.**`) already serves the structured-AC use case well enough.

---

## Alternatives Considered

### Alternative 1: Auto-advance features on last task completion

Instead of surfacing attention items (Feature 1), automatically transition the feature
from `developing` → `reviewing` when the last task finishes.

**Trade-offs:**
- (+) Zero human/agent intervention required — the gap disappears entirely.
- (+) Simpler than maintaining attention items and relying on agents to act on them.
- (−) Violates the "nudge, don't auto-advance" principle that the system currently
  follows. The system enforces gates when asked, but never transitions on its own.
- (−) May surprise orchestrators who want to do post-implementation cleanup before
  triggering review.
- (−) `reviewing` is a mandatory human gate — auto-entering it means the human is
  suddenly expected to act without having initiated the transition.

**Verdict:** Rejected for the initial implementation. The attention-item approach is
lower risk and consistent with the existing design philosophy. If attention items
prove insufficient (agents still don't act on them), auto-advance can be added later
as a configurable behaviour in the binding registry.

### Alternative 2: Feature-level `finish()` tool

Create a dedicated `finish_feature` action (or extend `finish` to accept feature IDs)
that wraps the entire close-out sequence: verify tasks, transition, PR, merge, cleanup.

**Trade-offs:**
- (+) Single tool call replaces a 5-step procedure.
- (+) Matches the user feedback requesting "fewer, richer tools."
- (−) Conflates multiple decisions (is the feature ready? should we merge? should we
  clean up?) into one atomic operation with no ability to intervene between steps.
- (−) Error recovery is complex — if PR creation succeeds but merge fails, the tool
  must handle partial completion.
- (−) Adds a new tool to an already large surface.

**Verdict:** Deferred. The skill-based procedure (Feature 2) is more flexible and
debuggable. If the procedure proves too tedious, a composite tool can be built on top
of it later — the procedure documents exactly what the tool would need to do.

### Alternative 3: Do nothing — rely on periodic human audits

The problem was found during a manual audit. Continue relying on periodic human review
to catch stale entities.

**Trade-offs:**
- (+) Zero implementation cost.
- (−) The audit that found this problem took significant effort: rebasing branches,
  resolving merge conflicts, walking plans through 3–4 lifecycle states each.
- (−) The longer stale work sits, the harder recovery becomes (more drift, more
  conflicts, more context loss).
- (−) Contradicts the purpose of a workflow system — if humans must audit the system
  to ensure it tracks reality, the system is not doing its job.

**Verdict:** Rejected. The whole point of Kanbanzai is to make workflow state reliable
and visible. Silent state drift is a system defect, not an operational procedure.

### Alternative 4: Mandatory worktrees for all features

Instead of supporting direct-to-main workflows (Feature 3), require every feature to
use a worktree so the merge/PR path is always available.

**Trade-offs:**
- (+) Eliminates the "which path am I on?" ambiguity.
- (+) Every feature has a clean merge point and branch history.
- (−) Massive friction for small changes, documentation updates, and solo development.
- (−) The user feedback report explicitly identifies worktree ceremony as a pain point.
- (−) Creates 45+ worktrees for a typical plan — the exact accumulation problem we
  just cleaned up.

**Verdict:** Rejected. The system should support how people actually work, not force
a workflow to make the tooling simpler.

---

## Decisions

**Decision 1: Detection over automation for feature completion.**
- **Context:** Features silently stall in `developing` after all tasks complete.
- **Rationale:** Attention items in `status` and `health` are consistent with the
  existing "nudge, don't auto-advance" philosophy and are lower risk than automatic
  transitions. They surface the problem without removing human agency.
- **Consequences:** Agents and humans must still act on the signal. If this proves
  insufficient, auto-advance can be added as a binding-registry option in a future
  iteration.

**Decision 2: Skill procedures over composite tools for close-out.**
- **Context:** The close-out sequence involves 4–5 steps (verify, transition, PR,
  merge, cleanup) with decision points between each.
- **Rationale:** A documented procedure in the orchestration skill is more transparent
  and debuggable than a composite tool. Each step can be inspected, retried, or skipped
  independently. The procedure also serves as the specification for a future composite
  tool if one is warranted.
- **Consequences:** Close-out requires multiple tool calls. The skill must be clear
  enough that agents follow it reliably.

**Decision 3: Graceful skip over hard failure for worktree-less merge.**
- **Context:** `merge` and `pr` currently error when no worktree exists.
- **Rationale:** Returning an informational "not applicable" result is backward-
  compatible, requires no schema changes, and lets existing skill procedures handle
  both paths naturally. It also provides a clear signal in logs and status that the
  feature used a direct-commit workflow.
- **Consequences:** The `merge` tool gains a third outcome (`skipped`) alongside
  `success` and `failure`. Skills must handle this case.

**Decision 4: Plan-level document inheritance is read-only.**
- **Context:** Features generate false-positive "missing spec" warnings when the
  plan has a spec but individual features don't.
- **Rationale:** Inheriting documents for gap-check purposes reduces noise without
  creating implicit document records. The inheritance is a query-time fallback, not
  a data mutation.
- **Consequences:** A feature that genuinely needs its own spec (diverging from the
  plan) must register one explicitly to override the inherited signal.

**Decision 5: Simple transition for crash recovery, not a formal unclaim mechanism.**
- **Context:** Tasks stuck in `active` after agent crashes need a way back to `ready`.
- **Rationale:** Adding `active → ready` to the transition map is a one-line change
  that solves the problem. A formal "unclaim" action with assignee tracking would be
  over-engineering — Kanbanzai doesn't currently track task assignees.
- **Consequences:** Any agent can move any active task back to ready, not just the
  original claimer. This is acceptable because the system is collaborative, not
  competitive.

**Decision 6: Section-based AC grouping as a heuristic, not a rule.**
- **Context:** One-AC-per-task produces too many tasks for tightly-coupled criteria.
- **Rationale:** Grouping by parent section is a reasonable heuristic that the
  `review` step can override. Making it a hard rule would remove flexibility for
  specs where section boundaries don't align with implementation boundaries.
- **Consequences:** Decomposition output changes from "N tasks for N ACs" to
  "M tasks for N ACs where M ≤ N". Task descriptions must list which ACs they cover.

---

## Dependencies

- **Feature 1 (completion detection)** has no dependencies and can be implemented first.
- **Feature 2 (skill updates)** depends on Feature 1 — the skills reference the attention
  items as triggers for the close-out procedure.
- **Feature 3 (direct-to-main)** is independent and can be implemented in parallel with
  Features 1–2.
- **Feature 4 (document inheritance)** is independent.
- **Feature 5 (crash recovery)** is independent.
- **Feature 6 (decomposition grouping)** is independent.

Features 1 and 2 should be implemented first as they directly address the observed
failure mode. Features 3–6 can be prioritised based on current friction.

---

## Migration and Backward Compatibility

All changes are additive:

- New attention items appear in `status` output — existing consumers are not affected.
- New health checks add warnings — no existing checks are modified.
- Skill file updates add sections — no existing guidance is removed.
- `merge` and `pr` gain a new return shape (`skipped`) — existing `success`/`error`
  paths are unchanged.
- The `active → ready` transition is a new edge in the transition map — no existing
  transitions are removed or modified.
- Document inheritance is a query-time fallback — no document records are created or
  modified.
- AC grouping changes decompose `propose` output — the `review` and `apply` steps
  are unchanged.
```

Now let me register the document and transition the plan: