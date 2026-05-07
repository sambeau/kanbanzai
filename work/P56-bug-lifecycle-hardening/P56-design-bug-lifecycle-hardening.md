| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | architect                       |
| Status | draft                          |
| Plan   | P56-bug-lifecycle-hardening    |

# Design: Bug Lifecycle Hardening

## Overview

The bug workflow (`bug_fix` tier) lacks the structural integrity of the feature workflow. Bugs can skip the review stage, have no lifecycle gate enforcement, no review cycle tracking, no canonical document paths for specs, and no worktree enforcement. This design makes the bug workflow structurally equivalent to features: every bug passes through a mandatory review gate with a spec to review against, worktrees isolate parallel fixes, and a shared definition of done replaces judgment with verification.

## Goals and Non-Goals

### Goals

- Make `needs-review` a mandatory stop-state in the bug lifecycle — no bug can bypass review
- Add lifecycle gate enforcement for bug transitions (equivalent to `CheckTransitionGate` for features)
- Add `review_cycle` tracking for bugs so the `MaxCycles` tier setting is enforced
- Auto-generate a specification document from the bug's `observed`/`expected` fields so reviewers have a spec to evaluate against
- Add a `fix_plan` field to the bug entity model for inline dev-plan content
- Give bugs canonical document paths (`work/bugs/<slug>/`)
- Enforce worktree usage for `in-progress` bugs — reject file mutations scoped to the repo root when a bug has an active worktree
- Adopt P55's definition of done, adapted for the bug lifecycle

### Non-Goals

- Changing the feature lifecycle or feature gate system
- Adding task decomposition for bugs (bugs remain single-entity fixes)
- Adding `advance` support for bugs (bug lifecycle is short enough to step manually)
- Adding merge/PR workflow for bugs (direct-to-main or worktree merge are both accepted)
- Changing the `retro_fix`, `feature`, or `critical` tier configurations
- Building P44 `dispatch_task` (this is procedural enforcement, not architectural)

## Problem and Motivation

### Problem

Three structural gaps make the bug workflow less rigorous than the feature workflow:

1. **No review enforcement.** The bug lifecycle includes `needs-review` but nothing prevents an agent from transitioning `in-progress → verified` or `needs-review → closed` without review. The `bug_fix` tier has `Review: auto` but the auto-validation pipeline has no server-side mechanism to dispatch reviewers for bugs.

2. **No spec to review against.** The `bug_fix` tier has `Spec: human` — requiring a human-written and human-approved specification — but bugs have no canonical document path and no mechanism to generate a spec. Reviewers have nothing to check conformance against except the bug's `expected` field, which is unstructured and not registered as a document.

3. **No gate enforcement.** `CheckTransitionGate` and `checkTransitionValidator` only apply to feature entities. Bugs pass through `entity(action: "transition")` with only the basic lifecycle graph check — no document prerequisites, no task-completeness checks, no review report requirements.

### Evidence

The audit of the current `bug_fix` tier revealed (see P55 analysis thread):

- **3 of 3 recently closed bugs** showed no evidence of formal review — no review reports registered, no reviewer sub-agents dispatched
- **2 of 3 recently closed bugs** had no worktree records — fixes were applied directly to the repo root with no isolation
- **0 bugs** had specification documents registered — the `Spec: human` gate was never satisfied
- **0 bugs** had `review_cycle` tracking — the `MaxCycles: 2` cap has no enforcement mechanism
- The `auto_validation` key in doc registration responses exists only in tests, not in production code — the fast-track auto-validation pipeline was designed but never wired

### Why This Matters Now

Bugs are code changes. Code changes need review. The system already has the infrastructure — reviewer roles, `orchestrate-review` skill, `TransitionValidatorDispatcher`, worktree hooks — but it is not applied to bugs. P55 Component 5 adds fast-track review dispatch from the orchestrator side; this design adds the server-side enforcement that makes review non-bypassable.

### Motivation

A `bug_fix` should differ from a `feature` only in scope, not in quality requirements. The bug lifecycle should have the same structural integrity: mandatory review, a spec to review against, worktree isolation, and a verifiable definition of done.

## Related Work

### Prior Research

- **P55-design-orchestrator-context-hygiene.md** — Defines the shared definition of done that this design adopts and adapts. Component 5 (Fast-Track Review Dispatch) provides the orchestrator-side review mechanism that this design enforces server-side.
- **P41-research-context-pollution-and-rot.md** — Finding 6 analyses goal drift caused by skipped close-out steps, including review.

### Prior Designs

- **P52-design-fast-track-orchestration.md** — The fast-track behavioural profile. Establishes the pattern that fast-track means no human gates, not fewer steps.
- **B17-workflow-and-tooling-3.0** — Feature gate enforcement (`CheckTransitionGate`) that this design mirrors for bugs.

### Constraining Decisions

- **P55 Decision 6:** Dispatch review sub-agents in fast-track close-out. This design makes that decision enforceable at the server level.
- **P55 Definition of Done:** All 10 conditions apply to features. This design adapts them for bugs (8 conditions).
- **P52 no-stop contract:** Fast-track means no human gates. This design preserves that — review is automated, not human-gated. The `bug_fix` tier's `Review: auto` gate mode means the system dispatches reviewers, not that review is skipped.

## Design

### Design Principle

**A bug is a small feature.** Every structural guarantee the feature lifecycle provides — mandatory review, spec traceability, worktree isolation, gate enforcement — should apply to bugs. The only difference is scale: bugs have inline specs (no separate document), a shorter lifecycle (no design/spec/dev-plan stages), and lighter review (one reviewer, not a panel).

### Component 1: Mandatory `needs-review` Stop-State

Add `needs-review` to a new `bugStopStates` map (mirroring `advanceStopStates` for features). When a bug enters `needs-review`, it cannot transition further until review is complete.

```go
var bugStopStates = map[string]bool{
    string(model.BugStatusNeedsReview): true,
}
```

The gate enforcement (Component 3) ensures the bug cannot leave `needs-review` without a review report. This is the server-side complement to P55 Component 5: the orchestrator dispatches the reviewer; the server enforces that it happened.

### Component 2: Auto-Generated Specification from Bug Report

When a bug is created, auto-generate a specification document from the `observed` and `expected` fields. Register it at `work/bugs/<slug>/spec.md` and auto-approve it (the bug report _is_ the spec).

The generated spec has a fixed structure:

```markdown
# Bug Specification: <bug name>

## Observed Behaviour
<bug.observed>

## Expected Behaviour
<bug.expected>

## Severity
<bug.severity> | Priority: <bug.priority> | Type: <bug.type>
```

This gives reviewers a concrete document to evaluate the fix against. The `Spec` gate mode for `bug_fix` changes from `human` to `auto` — the bug report is already human-authored via the `observed`/`expected` fields; the system generates and approves the spec automatically.

### Component 3: Bug Lifecycle Gate Enforcement

Add `CheckBugTransitionGate` — a bug equivalent of `CheckTransitionGate` — that enforces:

| Transition | Gate |
|---|---|
| `in-progress → needs-review` | A fix must be applied (worktree has commits beyond the base) |
| `needs-review → verified` | A review report document must be registered and owned by the bug |
| `needs-review → in-progress` | Review cycle cap not reached; increments `review_cycle` |
| `verified → closed` | All changes landed on main; worktree and branch cleaned up |

Wire this into `entityTransitionAction` for `entityType == "bug"` (currently the code falls through with no gate checks for bugs).

### Component 4: Review Cycle Tracking for Bugs

Add a `review_cycle` counter to the bug entity model (mirroring `Feature.ReviewCycle`). Increment it on each `needs-review → in-progress` (rework) transition. When `review_cycle >= tier.MaxCycles`, block the transition and escalate to a human checkpoint. This makes the `bug_fix` tier's `MaxCycles: 2` setting enforceable.

### Component 5: `fix_plan` Field

Add a `fix_plan` field to `CreateBugInput` and the bug entity model. This is a Markdown string describing the intended fix approach. It serves as an inline dev-plan, avoiding the need for a separate document. The reviewer can check that the implementation matches the fix plan.

```go
type CreateBugInput struct {
    // ... existing fields ...
    FixPlan string // inline dev-plan for the fix
}
```

### Component 6: Canonical Document Paths for Bugs

Add canonical path resolution for bug entities in `CanonicalDocPath`:

```
work/bugs/<slug>/spec.md       → specification (auto-generated)
work/bugs/<slug>/fix-plan.md   → dev-plan (from fix_plan field, if present)
work/reviews/review-<bug-id>-<slug>.md → review report
```

This gives bugs the same document structure as features, just under `work/bugs/` instead of `work/<feature-slug>/`.

### Component 7: Worktree Enforcement

Two changes:

1. **Reject repo-root mutations for `in-progress` bugs with active worktrees.** When `kanbanzai_edit_file` or `write_file` is called without an `entity_id` and there exists an `in-progress` bug with an active worktree, warn or reject. This prevents the "three parallel fixes on main" scenario.

2. **Auto-create worktree on `in-progress` transition.** Already implemented (`WorktreeTransitionHook.handleBugInProgress`). This design adds a health check that flags `in-progress` bugs without worktrees.

### Component 8: Adopt P55 Definition of Done

Adopt P55's definition of done, adapted for the bug lifecycle (8 conditions instead of 10 — bugs don't have tasks, and merge/ancestry are combined):

1. **Fix verified against expected behaviour** — every claim in `expected` addressed; `verification` field populated
2. **All changes committed** — `git status` clean
3. **Temporary files removed** — scratch scripts, repro files, debug output deleted
4. **Tests pass** — `go test ./...` passes on worktree and main; new tests reproduce the bug and verify the fix
5. **Code reviewed** — at minimum one review sub-agent dispatched, no blocking findings, review doc registered at `work/reviews/review-{bug-id}-{slug}.md`
6. **Bug advanced through full lifecycle** — `in-progress → needs-review → verified → closed`; no stage skipped; `needs-review` is a mandatory stop-state
7. **Changes landed on main** — ancestry verified; worktree merged or direct-to-main confirmed
8. **Worktree and branch cleaned up** — `worktree(action: "remove")` called; branch deleted

### What This Design Does NOT Do

- **Does not add task decomposition for bugs.** Bugs remain single-entity fixes. The `fix_plan` field provides enough structure for the reviewer without the overhead of task management.
- **Does not add `advance` support for bugs.** The bug lifecycle is short (4 transitions from `in-progress` to `closed`). Manual stepping is acceptable.
- **Does not change feature lifecycle behaviour.** The bug gate system is additive, not a refactor of existing feature gates.
- **Does not require P44.** These are procedural and server-side enforcements that work in the current architecture.
- **Does not change the `retro_fix` or `critical` tier configs.** Only `bug_fix` is in scope.

## Alternatives Considered

### Alternative A: Do Nothing — Bugs Stay as They Are

**Rejected because:** The audit showed concrete gaps: no review enforcement, no specs, no worktree isolation. The system has the infrastructure to close these gaps; not doing so is technical debt that accumulates with every bug fixed without review.

### Alternative B: Full Feature Lifecycle for Bugs

Route bugs through the full feature lifecycle: `proposed → designing → specifying → dev-planning → developing → reviewing → done`. Bugs would get full document packages, task decomposition, and the complete gate system.

**Rejected because:** Too heavy. A bug fix is a single change, not a multi-task feature. The overhead of creating design docs, specs, dev-plans, and tasks for a one-line fix is disproportionate. The inline approach (bug report as spec, `fix_plan` as dev-plan) provides equivalent quality with appropriate overhead.

### Alternative C: Skip the Bug Lifecycle Entirely — Fix on Main

Treat bugs as exceptions to the workflow: fix directly on main, no lifecycle, no review. This is the current _de facto_ behaviour for some bugs.

**Rejected because:** It removes all quality guarantees. Even small fixes introduce regressions. The P55 definition of done exists precisely because "it's a small fix, it's probably fine" is a judgment that degrades with context accumulation.

### Alternative D: Merge Bug Lifecycle into Feature Lifecycle

Replace the bug entity type with a feature flagged as `tier: bug_fix`. Bugs become features with abbreviated document requirements.

**Rejected because:** The bug entity type serves a distinct purpose — issue tracking, triage, reproduction. Collapsing bugs into features loses the `reported → triaged → reproduced → planned` pipeline that gives bugs their investigative front-end. The lifecycle stages serve different functions: bug stages are about understanding the problem; feature stages are about building the solution.

## Decisions

### Decision 1: Bugs get inline specs, not separate documents

**Rationale:** The bug report (`observed`/`expected`) already contains the information a spec would. Auto-generating a spec document from these fields gives reviewers a standard artefact without requiring the reporter to write a separate document. This satisfies the `Spec` gate without human overhead.

### Decision 2: `needs-review` is a mandatory stop-state, not a gate-mode toggle

**Rationale:** P55 already establishes that review is not optional for fast-track — the orchestrator must dispatch reviewers. Making `needs-review` a server-enforced stop-state means the system guarantees review happened, not just that the orchestrator was told to do it. This is defense in depth: the orchestrator dispatches (P55 Component 5); the server enforces (this design).

### Decision 3: `bug_fix` Spec gate mode changes from `human` to `auto`

**Rationale:** With auto-generated specs from bug reports (Component 2), the spec is always present and always accurate (it _is_ the bug report). There is no additional human step to gate. The human already wrote the spec when they created the bug. Changing to `auto` removes a gate that provided no additional safety while adding friction.

### Decision 4: One reviewer for bugs, not a panel

**Rationale:** P55 Component 5 already defines this: "For `bug_fix` features with ≤5 files changed, a single `reviewer-conformance` sub-agent is sufficient." This design endorses that and adds that the conformance reviewer evaluates the fix against the auto-generated spec (the bug's `expected` field). A full panel (conformance + quality + security + testing) is disproportionate for a targeted fix.

### Decision 5: Worktree enforcement is a health warning, not a hard block (initially)

**Rationale:** The worktree auto-creation hook already exists and works for most cases. Adding a hard block on repo-root mutations would break edge cases (e.g., config changes, document-only fixes). Start with a health check warning that flags `in-progress` bugs without worktrees, and a warning on repo-root writes when active bug worktrees exist. Escalate to a hard block after gathering data on false positives.

### Decision 6: Adopt P55 Definition of Done verbatim where possible, adapt where necessary

**Rationale:** P55 establishes that fast-track and full-procedure share one definition of done. This design applies the same principle: `bug_fix` and `feature` share one standard. The 8-condition bug DoD is a direct adaptation of P55's 10-condition feature DoD, with items merged only where the bug lifecycle genuinely differs (no tasks, no separate merge stage).

## Dependencies

- **P55-orchestrator-context-hygiene** — Component 5 (Fast-Track Review Dispatch) provides the orchestrator-side review mechanism. This design adds server-side enforcement. P55 should land first so the orchestrator knows how to dispatch reviewers before the server requires it.
- **P52-fast-track-orchestration** — The no-stop contract and fast-track behavioural profile that this design operates within.
- **orchestrate-review/SKILL.md** — Referenced (not modified); provides the review sub-agent dispatch procedure that P55 Component 5 invokes.
- **internal/validate/lifecycle.go** — Modified to add `bugStopStates` and bug gate enforcement.
- **internal/service/prereq.go** — Modified to add `CheckBugTransitionGate`.
- **internal/model/entities.go** — Modified to add `fix_plan` and `review_cycle` to bug model.

## Definition of Done

Every bug — regardless of tier — must satisfy all eight conditions before reaching `closed`. Fast-track means no human gates, not fewer steps. This list is the contract: if a condition cannot be verified, the bug is not closed.

1. **Fix verified against expected behaviour** — every claim in the bug's `expected` field has been addressed. The `verification` field is populated with concrete confirmation (test output, reproduction steps, or manual verification notes).

2. **All changes committed** — `git status` is clean. No uncommitted source files, test files, workflow state, or temporary artifacts.

3. **Temporary files removed** — scratch scripts, repro files, debug output, or manual test fixtures used during the fix are deleted.

4. **Tests pass** — `go test ./...` passes on the worktree branch before landing and on `main` after. New tests exist that reproduce the bug and verify the fix.

5. **Code reviewed** — at minimum one review sub-agent with clean context has been dispatched (via `orchestrate-review`), findings collated, and no blocking findings remain. The reviewer evaluates the fix against the bug's auto-generated spec (the `expected` field). A review document is registered at `work/reviews/review-{bug-id}-{slug}.md`.

6. **Bug advanced through full lifecycle** — `in-progress → needs-review → verified → closed`. Each transition is an explicit `entity(action: "transition")` call. No stage is skipped. `needs-review` is a mandatory stop-state — the bug cannot advance past it without a review report.

7. **Changes landed on main** — the fix is on the `main` branch. For worktree-based fixes, `merge(action: "execute")` succeeded and ancestry is verified. For direct-to-main fixes (trivial one-liners), `git merge-base --is-ancestor HEAD main` confirms the change is reachable from main.

8. **Worktree and branch cleaned up** — if a worktree was used, `worktree(action: "remove")` has been called and `git worktree list` confirms the directory is gone. The branch is deleted and `git branch | grep <bug-id>` returns nothing.

### Rationale for a Single Definition

The fast-track profile and the full procedure share one definition of done. A `bug_fix` is still a code change — it differs from a `feature` only in scope, not in quality requirements. Fast-track means no human gates, not fewer verification steps. The list above replaces judgment ("this is a small fix, it's probably fine") with verification.

## Open Questions

1. **Should `in-progress` bugs without worktrees block transitions?** Currently, the worktree hook auto-creates worktrees on `in-progress` transition, but there is no enforcement that the worktree is used. Decision 5 starts with health warnings. Should this escalate to a hard block? When?

2. **Should the auto-generated spec be editable?** If the bug reporter's `expected` field is incomplete or incorrect, the spec is wrong. Should the spec be editable after auto-generation, or should the bug's `expected` field be the single source of truth (edit the bug, regenerate the spec)?

3. **How does `fix_plan` interact with the `DevPlan: auto` gate mode?** If `DevPlan` is `auto`, the system should auto-approve the `fix_plan` when present. But `fix_plan` is optional — what if it's empty? Does the gate pass vacuously, or does it require content?

4. **Should bugs support the full `merging → verifying` sub-lifecycle?** This design allows direct-to-main for trivial fixes (DoD item 7). But for worktree-based fixes, should bugs go through `merging → verifying` like features do, or is `verified` sufficient as a single post-review state?

5. **Should `bug_fix` tier change be backported to existing bugs?** Existing bugs have no tier, no spec, no `fix_plan`. Should the system infer `bug_fix` tier and auto-generate specs for existing `in-progress` bugs, or only for newly created ones?
