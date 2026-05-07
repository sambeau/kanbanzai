| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | architect                       |
| Status | approved |
| Plan   | P55-orchestrator-context-hygiene |

# Design: Orchestrator Context Hygiene

## Overview

The Kanbanzai orchestrator accumulates implementation context over the course of a development cycle by reading source code before delegating to sub-agents. This causes context rot: forgotten constraints, skipped close-out steps, and degraded orchestration decisions. The fix is a set of procedural mitigations — an explicit anti-pattern, tool restrictions, hard constraints, and constraint pinning — that prevent the orchestrator from investigating code and keep it focused on coordination decisions. P44's `dispatch_task` is the long-term architectural fix; these mitigations provide immediate protection.

## Goals and Non-Goals

### Goals

- Prevent the orchestrator from reading implementation source code before delegating to sub-agents
- Add explicit anti-patterns and hard constraints that name the specific behaviour (pre-delegation code investigation)
- Restrict orchestrator tools to remove codebase investigation capabilities (`grep`, `search_graph`)
- Pin orchestrator identity constraints in every `next`/`handoff` response to maintain role awareness
- Reduce context rot and the associated close-out failures (forgotten review, merge, cleanup)

### Non-Goals

- Changing the `handoff` pipeline or `implement-task` skill
- Building P44 `dispatch_task` (this is the strategic architecture, not this plan)
- Restricting `read_file` to document-only paths (deferred to a future iteration)
- Changing task decomposition or the orchestrator-workers pattern

## Problem and Motivation

### Problem

The Kanbanzai orchestrator accumulates implementation context over the course of a development cycle, causing context rot that manifests as goal drift: forgotten constraints, skipped close-out steps (review, merge, cleanup), and degraded orchestration decisions by the end of long sessions.

The root cause is that the orchestrator — despite having no implementation tools (`write_file`, `terminal`, `kanbanzai_edit_file`) — retains investigation tools (`read_file`, `grep`, `search_graph`). It uses these to pre-understand code before delegating to sub-agents. Each investigation loads implementation details into the orchestrator's context that compete with orchestration rules and role constraints. Over 8–12 tasks, the accumulated code fragments push orchestration constraints into the attention valley, and the orchestrator reverts to general-purpose problem-solving behaviour.

### Evidence

This is not hypothetical. Two concrete data points:

1. **P50 incident (May 2026):** During fast-track implementation of 16 tasks across 4 features, the orchestrator forgot the fast-track "no human gates" constraint and stopped mid-pipeline for confirmation. The constraint appeared once at session start and was never reinforced. By mid-session, the `orchestrate-development` skill's "stop at 60% context" instruction (more recent, more procedurally embedded) overrode the tier constraint. This is goal drift caused by context accumulation.

2. **Observed orchestrator behaviour (current cycle):** The orchestrator has been observed reading implementation code, tracing logic, and attempting to understand bugs before handing them to sub-agents. This "helpful" pre-understanding is the pollution vector — every line of code read competes with the orchestrator's own role constraints.

3. **Fast-track review gap (current cycle):** The fast-track profile (`orchestrate-development` fast-track section) never dispatches review sub-agents. Phase 2 Close-Out transitions features to `done` or `reviewing` "as appropriate" but contains no step for dispatching specialist reviewers via `orchestrate-review`. This means the orchestrator either skips review entirely (transitioning directly to `done`) or performs its own self-review as part of close-out — reading code, checking correctness, and making review judgments. This directly violates the orchestrator's role: reviews belong to sub-agents with clean contexts, specialist reviewer roles, and the `review-code` skill. The `orchestrate-review` skill already exists and correctly dispatches reviewer sub-agents — it is simply never invoked from the fast-track path.

### Why This Matters Now

The system has invested heavily in specs, dev-plans, roles, skills, and the `handoff` pipeline precisely so that sub-agents receive everything they need. The orchestrator re-doing this understanding work:

- Duplicates effort (sub-agent re-investigates the same code)
- Accelerates context rot (implementation details crowd out orchestration rules)
- Causes forgotten close-out steps (review, merge, branch deletion, cleanup)
- Undermines the orchestrator-workers architecture pattern

### Motivation

The orchestrator should be a fast, low-thinking rule-follower — a coordinator, not a thinker. Every implementation detail it loads is a step toward losing that identity. The design goal is to make implementation investigation structurally impossible or procedurally prohibited, so the orchestrator stays focused on coordination decisions: which tasks, what order, how to handle failures.

## Related Work

### Prior Research

- **P41-research-context-pollution-and-rot.md** — Definitive research on context rot in agent orchestration. Finding 1 establishes context rot as a distinct behavioural failure mode. Finding 4 identifies dispatch-loop architecture as the architectural fix. Finding 6 analyses the P50 incident as a context engineering failure. Recommendation 1 calls for near-term mitigations including constraint pinning.

- **P41-research-context-compaction.md** — Compaction research establishing the U-shaped artefact template. Distinguishes compaction (mitigation of accumulated context) from architectural prevention (stopping context from entering in the first place).

### Prior Designs

- **P44-design-model-routing-agent-launcher.md** — The strategic architecture: `dispatch_task` tool that internalises the pipeline, eliminates the manual-prompt gap, and provides session-scoped context. This is the long-term fix that makes pre-delegation investigation structurally impossible (the orchestrator never sees prompts to investigate).

- **P52-design-fast-track-orchestration.md** — The fast-track behavioural profile designed post-P50. Defines the session-start audit, no-implicit-gates rules, and ghost-work detection. Relevant because it demonstrates the pattern of adding explicit behavioural constraints to prevent context-driven drift.

### Constraining Decisions

- **P44 §Enforcement:** The manual-prompt gap exists because `spawn_agent` accepts arbitrary text. `dispatch_task` makes the pipeline non-bypassable. Until P44 is built, procedural guardrails must fill the gap.

- **P52 no-stop contract:** "You will NOT stop for confirmation at any point." This is an example of a hard constraint that was forgotten during P50 — demonstrating why constraints need reinforcement (pinning) and why the orchestrator needs protection from its own helpfulness.

## Design

### Design Principle

**The orchestrator delegates implementation understanding, not just implementation work.** The sub-agent's fresh context window and the `implement-task` skill exist precisely to absorb the implementation details that would pollute the orchestrator. The orchestrator's only implementation-adjacent responsibility is reading the dev-plan (a coordination document, not source code).

### Component 1: Explicit Anti-Pattern — Pre-delegation Code Investigation

Add to the orchestrator role (`orchestrator.yaml`) and `orchestrate-development` skill a new anti-pattern:

```yaml
- name: "Pre-delegation Code Investigation"
  detect: "Orchestrator reads source files, traces call paths, or searches the
           code graph to understand implementation details before delegating"
  because: "Implementation understanding belongs to the sub-agent. The dev-plan,
           spec, and handoff context give the sub-agent everything needed. Every
           code fragment the orchestrator loads competes with orchestration
           constraints and accelerates context rot — causing forgotten close-out
           steps and goal drift by end of cycle."
  resolve: "Delegate immediately via handoff. Trust the pipeline. If the
           dev-plan is unclear, fix the dev-plan — don't read the code."
```

### Component 2: Tool List Restriction

Remove `grep` and `search_graph` from the orchestrator's tool list. The orchestrator needs `read_file` for reading orchestration documents (dev-plans, specs, skills, roles) but does not need codebase search tools. If structural questions arise during orchestration (e.g., "does a file exist?"), the orchestrator should delegate that to a sub-agent or use `status`/`entity` tools that operate on workflow state, not source code.

Current orchestrator tools:

```
entity, doc, doc_intel, knowledge, status, next, handoff, finish,
decompose, estimate, conflict, checkpoint, health, branch, worktree,
pr, merge, read_file, grep, search_graph
```

Proposed orchestrator tools:

```
entity, doc, doc_intel, knowledge, status, next, handoff, finish,
decompose, estimate, conflict, checkpoint, health, branch, worktree,
pr, merge, read_file
```

`read_file` is retained because the orchestrator must read orchestration inputs: dev-plans, specs, skill files, role files, stage bindings, and progress documents. These are coordination documents, not implementation code.

### Component 3: Procedure Constraint — Phase 1 of orchestrate-development

Add an explicit constraint at the start of Phase 1 (Read the Dev-Plan) in `orchestrate-development/SKILL.md`:

> **Constraint ℋ — No Code Investigation:** Do not read source files, trace call
> paths, or search the code graph to "understand" implementation areas before
> dispatching. The sub-agent receives the dev-plan, spec sections, knowledge
> entries, and file paths via `handoff` — this is sufficient context for
> implementation. Every line of source code you read competes with orchestration
> constraints and accelerates context rot. If the dev-plan is unclear about
> what to build, flag it — don't read the code to compensate.

Marked as a **hard constraint (ℋ)** — non-negotiable, violation blocks stage advance.

### Component 4: Constraint Pinning in handoff/next Responses

The orchestrator's critical identity constraint — *"I coordinate; sub-agents implement"* — should appear in every `next` and `handoff` response, not just at session start. This uses the recency peak of the U-shaped attention curve to maintain role awareness.

Minimal token cost (~15–20 tokens per response). Example:

> **Role reminder:** You are the orchestrator — coordinate, dispatch, verify.
> Do not investigate implementation code. Sub-agents handle all code reading
> and writing via `handoff`.

### Component 5: Fast-Track Review Dispatch

**Problem:** The fast-track profile's Phase 2 Close-Out transitions features to `done` or `reviewing` "as appropriate" but never dispatches review sub-agents. Review sub-agents exist — the `orchestrate-review` skill dispatches specialist reviewers (`reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`) with the `review-code` skill in clean contexts. But the fast-track path never invokes this skill.

**Fix:** Add an explicit step to the fast-track Phase 2 Close-Out that requires dispatching review sub-agents before transitioning a feature out of `developing`:

> **Before transitioning any feature to `done` or `reviewing`:** For each feature that modified source code (not documentation-only changes), dispatch at minimum one review sub-agent:
> - Read `orchestrate-review/SKILL.md` and follow Steps 3–6 (select reviewer, dispatch, collate, aggregate verdict).
> - For `bug_fix` features with ≤5 files changed, a single `reviewer-conformance` sub-agent is sufficient.
> - For `retro_fix` features with source changes, dispatch `reviewer-conformance` at minimum.
> - Dispatch review sub-agents in parallel using `spawn_agent` with clean contexts.
> - Only proceed to transition after collating findings.

**Rationale:** Bugs are code changes. Code changes need review. The fact that a bug fix is small doesn't mean it's correct — small changes can introduce regressions, violate invariants, or miss edge cases. The review sub-agent has a clean context, specialist role, and structured procedure — it catches issues the orchestrator (operating with accumulated implementation context) will miss.

### Component 6: Fast-Track Integration

The fast-track profile (`orchestrate-development` fast-track section and P52 design) already has a no-stop contract and rules against implicit gates. This design adds: fast-track sessions should load a lightweight version of the orchestrator role that excludes `grep` and `search_graph` entirely, and the constraint pinning message should be included in every fast-track dispatch cycle.

### Component 7: Close-Out Verification Sub-Agent

**Problem:** The orchestrator is the worst agent to verify close-out. By the end of a cycle, its context is saturated with implementation fragments, its role constraints have drifted into the attention valley, and close-out steps (merge verification, branch deletion, knowledge curation) are exactly the steps it forgets. The Definition of Done lists ten conditions — the orchestrator cannot be trusted to verify them all from memory.

**Solution:** Delegate close-out verification to a sub-agent with a clean context and a strict checklist. This follows the same pattern as review (specialist sub-agent, clean context, structured output) applied to the `verifying` stage.

**New role: `verifier`** — identity: "Methodical close-out auditor." Tools: `entity`, `status`, `doc`, `knowledge`, `read_file`, `grep`. Vocabulary: checklist-driven, evidence-backed, binary verdict (pass/fail per item). Anti-patterns: assumption-without-evidence, skipping checklist items, accepting orchestrator assurances.

**New skill: `verify-closeout`** — a strict 10-item checklist mapped to the Definition of Done. Each item has a concrete verification action (e.g., "run `git status --porcelain` and confirm no output," "run `git merge-base --is-ancestor` and confirm exit zero," "run `go test ./...` and confirm all pass"). Output is a structured pass/fail report per item with evidence.

**Pipeline integration:**

- The `verifying` stage dispatches the `verifier` sub-agent via `spawn_agent` with the `verify-closeout` skill.
- The verifier receives: the feature ID, the Definition of Done checklist, and the current entity state.
- The verifier runs each verification action independently — it does not trust the orchestrator's claims.
- The verifier returns a structured report: each DoD item marked `pass` or `fail` with evidence.
- The orchestrator's only job: read the report. If all pass, transition to `done`. If any fail, route to remediation.
- This applies to both full-procedure and fast-track features. Fast-track means no human gates during *development* — verification is not a human gate, it's an automated checklist.

**Why this works:** The verifier has no accumulated context. It sees only the checklist, the feature ID, and the verification tools. It cannot be talked into skipping steps because it doesn't converse — it checks and reports. This is the same architectural pattern that makes review sub-agents effective: clean context + structured procedure + binary output.

### What This Design Does NOT Do

- **Does not change the `handoff` pipeline.** Sub-agents continue receiving the same assembled context.
- **Does not change the `implement-task` skill.** Sub-agents continue with their existing procedure.
- **Does not require P44.** These mitigations work in the current architecture.
- **Does not prevent the orchestrator from reading documents.** `read_file` remains for orchestration inputs (dev-plans, specs, skills, progress documents).
- **Does not change task decomposition.** The orchestrator still reviews task breakdowns and identifies parallel-dispatchable tasks.
- **Does not require the orchestrator to verify close-out itself.** Verification is delegated to a clean-context sub-agent — the orchestrator only reads the result.

## Definition of Done

Every feature — regardless of tier — must satisfy all ten conditions before reaching `done`. Fast-track means no human gates, not fewer steps. This list is the contract: if a condition cannot be verified, the feature is not done.

1. **All tasks terminal** — every task under the feature is `done`, `not-planned`, or `duplicate`. No task remains in `ready`, `active`, `needs-review`, or `needs-rework`. No lifecycle stages were skipped.

2. **All changes committed** — `git status` is clean. No uncommitted source files, test files, workflow state, or temporary artifacts. If a file is not committed and not tracked, it does not exist at close-out.

3. **Temporary files removed** — any scratch scripts, repro files, debug output, or manual test fixtures used during development are deleted. Files required for the ongoing test suite to run are committed in appropriate locations (not left in the worktree root). The recommended pattern for Go is `t.TempDir()` for automated test isolation; for manual artifacts, the worktree itself is the sandbox — removal on close-out is sufficient.

4. **Tests pass** — `go test ./...` passes on the feature branch before merge and on `main` after merge. Suitable new tests exist for the change: at minimum, every acceptance criterion has a corresponding test.

5. **Code reviewed** — at minimum one review sub-agent with clean context has been dispatched (via `orchestrate-review`), findings collated, and no blocking findings remain. Non-blocking findings addressed in at least one round. A review document is registered at `work/reviews/review-{feature-id}-{slug}.md`.

6. **Feature advanced through full lifecycle** — `developing → reviewing → merging → verifying → done`. Each transition is an explicit `entity(action: "transition")` call. No stage is skipped.

7. **Merged and ancestry verified** — `merge(action: "execute")` succeeded and `git merge-base --is-ancestor <feature-branch> main` exits zero. The merge is confirmed in git, not just in entity state.

8. **Branch deleted and verified absent** — `git branch | grep <feature-id>` returns nothing. The branch is gone. This prevents the "what has and hasn't landed" ambiguity that has caused repeated incidents.

9. **Worktrees removed** — `worktree(action: "remove")` called. `git worktree list` confirms the worktree directory is gone. No orphaned worktree directories accumulate.

10. **Knowledge curated and entities closed** — tier-2 knowledge entries contributed during the feature are confirmed, flagged, or retired. Related entities (bugs, decisions) are transitioned to terminal states. No loose ends remain.

### Rationale for a Single Definition

The fast-track profile and the full procedure share one definition of done. Fast-track differs only in *how* work is dispatched (no human gates, no stop-and-confirm, continuous polling), not in *what constitutes done*. Without an explicit, shared contract, the orchestrator treats "done" as a judgment call — and with accumulated implementation context, that judgment degrades. The list above replaces judgment with verification.

**Enforcement:** The ten items are verified by a close-out verifier sub-agent (see Component 7) with a clean context and a strict checklist. The orchestrator does not self-verify — it delegates verification, reads the structured pass/fail report, and acts on the result. This ensures the DoD is not a memory exercise for a context-polluted orchestrator.

## Alternatives Considered

### Alternative A: Do Nothing — Rely on Existing Anti-Patterns

The orchestrator role already has anti-patterns for context bloat and manual prompt composition. In theory, these should prevent investigation-driven rot.

**Trade-offs:**
- **Makes easy:** No changes required.
- **Makes hard:** The existing anti-patterns don't name the specific behaviour (pre-delegation code reading) and are routinely violated. The P50 incident happened despite these anti-patterns being present.
- **Risks:** Continued context rot, continued forgotten close-out steps, continued goal drift. The status quo.

**Rejected because:** The evidence (P50, current cycle observations) shows existing guardrails are insufficient. The anti-patterns describe symptoms (context bloat) but not the cause (pre-delegation investigation).

### Alternative B: Build P44 dispatch_task First

Skip procedural mitigations and go directly to the architectural fix: `dispatch_task` with session-scoped context and non-bypassable pipeline.

**Trade-offs:**
- **Makes easy:** Architecturally clean — eliminates the problem at root. No procedural guardrails to maintain.
- **Makes hard:** P44 is estimated at 4–6 weeks of implementation. Context rot continues during that time.
- **Risks:** P44 is unbuilt and untested. If P44's Phase 1 has unexpected complexity or delays, the problem persists untreated.

**Rejected as the sole approach because:** Procedural mitigations (anti-patterns, tool restrictions, constraint language) can be implemented in hours, not weeks. They buy time and provide immediate protection while P44 is built. They also serve as a fallback if P44 encounters delays.

### Alternative C: State-Machine Orchestration

Replace the chat-as-orchestrator model with a deterministic state machine. The orchestrator becomes a state transition engine — no conversation, no context accumulation, no rot.

**Trade-offs:**
- **Makes easy:** Zero context rot. Deterministic, auditable behaviour.
- **Makes hard:** Loses all flexibility — edge cases, ambiguous review findings, circular dependencies all require human intervention. Would require rearchitecting the entire orchestration layer.
- **Risks:** No production-scale evidence for state-machine orchestration in MCP-native workflows. Implementation effort is very large. No incremental path.

**Rejected because:** P41 research Finding 7 found low confidence for state-machine orchestration as a standalone recommendation. The hybrid approach (dispatch loop enforces pipeline, orchestrator retains decision authority) captures the benefits without the rigidity.

### Alternative D: Compaction-Only Approach

Rely solely on context compaction (U-shaped artefact, 60%/80% triggers) to manage accumulated context without restricting what enters context.

**Trade-offs:**
- **Makes easy:** Compaction is already designed (P41, P44 §Compaction). No new constraints on orchestrator behaviour.
- **Makes hard:** Compaction addresses accumulated context (symptom) but not what enters context (cause). The orchestrator still loads implementation details, still experiences mid-cycle drift, still forgets close-out steps — just gets a fresh session periodically.
- **Risks:** Compaction without prevention is a rearguard action. Context keeps growing, compaction keeps compressing. The orchestrator still degrades between compaction events.

**Rejected as the sole approach because:** P41 research Finding 5 explicitly states compaction is a complement to architectural change, not a replacement. Prevention (stopping implementation context from entering) and mitigation (compacting what does accumulate) address different failure modes and are both needed.

## Decisions

### Decision 1: Add "Pre-delegation Code Investigation" anti-pattern

**Rationale:** The existing anti-patterns describe symptoms (context bloat, manual prompt composition) but not the specific causal behaviour: reading implementation code before delegating. Naming the behaviour makes it detectable and resolvable. This is the lowest-effort, highest-signal intervention.

**References:** P50 incident (goal drift from accumulated context), P41 Finding 1 (context rot as behavioural degradation).

### Decision 2: Remove grep and search_graph from orchestrator tools

**Rationale:** These tools serve no orchestration purpose. The orchestrator uses them to investigate code — which is the pollution vector. `read_file` is retained for reading orchestration documents. If the orchestrator needs structural information about the codebase (e.g., "does this function exist?"), it should delegate that question.

**Risk:** The orchestrator may occasionally need `grep` for legitimate orchestration tasks (e.g., searching for todos in documents). **Mitigation:** If legitimate use cases emerge during implementation, `grep` can be restored with a scoped constraint rather than removed entirely.

### Decision 3: Add hard constraint (ℋ) to orchestrate-development Phase 1

**Rationale:** A hard constraint in the procedure carries more weight than an anti-pattern. Anti-patterns say "this is wrong"; hard constraints say "this blocks stage advance." Marking code investigation as ℋ makes it a gate violation, not a style issue.

**References:** P41 Finding 3 (constraint design matters — restrictive rules backfire, but hard constraints on defined violations are effective).

### Decision 4: Implement constraint pinning in next/handoff responses

**Rationale:** The U-shaped attention curve means constraints stated once at session start fall into the attention valley. Re-stating the orchestrator's identity constraint in every `next`/`handoff` response places it in the recency peak. Minimal token cost.

**References:** Liu et al. (2024) on U-shaped attention curve. P50 incident (constraint stated once, forgotten mid-session).

### Decision 5: Procedural mitigations now, P44 as strategic architecture

**Rationale:** The procedural mitigations (anti-patterns, tool restrictions, constraint language) are implementable immediately and provide protection while P44 is built. P44's `dispatch_task` is the architectural fix that makes these procedural guardrails unnecessary by making the pipeline non-bypassable. This is a two-phase approach: stop the bleeding now, fix the architecture later.

**References:** P41 Recommendation 1 (near-term mitigations) and Recommendation 2 (P44 as strategic architecture).

### Decision 6: Dispatch review sub-agents in fast-track close-out

**Rationale:** The fast-track profile currently has no review step — the orchestrator either skips review (transitioning directly to `done`) or performs self-review. Both are wrong. Review is a specialist task requiring clean context, structured dimensions, and evidence-backed findings. The `orchestrate-review` skill already provides this — it just needs to be invoked from the fast-track path. Adding a mandatory review-dispatch step before close-out transition ensures every code change (including bug fixes) receives at minimum a conformance review from a sub-agent with a clean context.

**Risk:** Adding review to fast-track increases total cycle time. **Mitigation:** For bug_fix features with 5 files or fewer, dispatch only one reviewer (conformance) rather than the full panel. Fast-track remains fast; it just doesn't skip quality.

**References:** `orchestrate-review` skill (Steps 3-6: select reviewer, dispatch, collate, aggregate verdict). Reviewer roles with `review-code` skill provide clean-context specialist review.

### Decision 7: Delegate close-out verification to a clean-context sub-agent

**Rationale:** The orchestrator is the worst agent to verify close-out — by end of cycle its context is saturated and it forgets steps. A `verifier` sub-agent with clean context, a strict `verify-closeout` checklist mapped to the Definition of Done, and no conversational interface eliminates judgment calls. The orchestrator reads the pass/fail report and acts — it does not self-verify.

**Risk:** Adding a verification stage increases total cycle time by one sub-agent dispatch. **Mitigation:** The verifier runs in parallel with no other work — it's a single dispatch with a fixed checklist. Expected duration: 30-60 seconds for automated checks (git status, go test, merge ancestry).

**References:** Same architectural pattern as review sub-agents (clean context + structured procedure + binary output). Definition of Done (10 items, each with concrete verification action).

## Dependencies

- **P44-design-model-routing-agent-launcher** — The strategic architecture that makes these procedural mitigations unnecessary. This design is a bridge to P44, not a replacement.
- **P52-design-fast-track-orchestration** — The fast-track behavioural profile that already defines no-stop contracts and implicit-gate rules. This design extends those guardrails.
- **P41-research-context-pollution-and-rot** — The evidence base for all recommendations.
- **orchestrator.yaml** role file — Modified to add the anti-pattern and remove tools.
- **orchestrate-development/SKILL.md** — Modified to add the hard constraint in Phase 1 and the fast-track review dispatch step.
- **orchestrate-review/SKILL.md** — Referenced (not modified); provides the review sub-agent dispatch procedure.
- **verifier.yaml** role file — New. Close-out auditor with checklist-driven identity and binary-verdict output.
- **verify-closeout/SKILL.md** — New. Strict 10-item checklist mapped to the Definition of Done with concrete verification actions per item.
- **orchestrate-development/SKILL.md** — Modified (verifying stage): dispatches verifier sub-agent before transitioning to `done`.

## Open Questions

1. **Does `read_file` alone provide enough of a backdoor?** The orchestrator could still use `read_file` to read source files (not just documents). A future iteration might consider restricting `read_file` to `.md`, `.yaml`, and `.kbz/` paths only. Deferred until we have data on whether the anti-pattern alone is sufficient.

2. **What is the legitimate orchestration use case for `grep` and `search_graph`?** If one exists (e.g., searching for `TODO` markers across the project for planning purposes), the tools should be constrained rather than removed. This should be investigated during implementation.

3. **How should `search_graph` restriction interact with codebase-memory-mcp skills?** The `.github/skills/codebase-memory-*/SKILL.md` files instruct agents to use graph tools. If the orchestrator role forbids graph tools, this instruction needs to be contextualised per-role. The skill files should note that graph tools are for implementer and reviewer roles, not orchestrator.

4. **How should fast-track invoke `orchestrate-review` without the orchestrator performing review work?** The Definition of Done resolves the lifecycle question: fast-track follows the full `developing → reviewing → merging → verifying → done` sequence, same as the full procedure. The open question is mechanics: the `orchestrate-review` skill is designed as a separate orchestration session. Should fast-track close-out spawn a fresh orchestrator session for review? Should it invoke `orchestrate-review` procedure steps directly? Or should a new lightweight review-dispatch tool handle it? This is a procedural integration question to resolve during implementation, not a design-level ambiguity.
