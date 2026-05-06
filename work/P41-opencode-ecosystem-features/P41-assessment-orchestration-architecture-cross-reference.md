# Integrated Architectural Assessment: Orchestration & Fast-Track Pipeline Strategy

**Audience:** Head of Product, Senior Architects  
**Date:** 2026-05-06  
**Source:** Cross-reference of P41 Strategy Report against P54, P52, P51, P44, P43 designs  
**Status:** Assessment

---

## Executive Summary

- **The five plans form a coherent, well-sequenced architecture.** Each plan addresses a specific context-rot mitigation from the strategy, and the dependency graph is logically sound. P51 clears the path, P52 fixes behaviour now, P44 delivers the strategic architecture, P43 validates the fresh-session pattern, and P54 closes the review-remediation gap that both fast-track and traditional workflows need.

- **P54 belongs in Horizon 1, not Horizon 0.** It depends on P51 (reliable handoff for remediation task creation), benefits from P52 (session-start audit detects remediation state), and its full value (automated finding-to-task mapping) requires P44's `dispatch_task` — but the document-driven first version can ship earlier. Sequencing P54 between P52 and P44 is correct.

- **Three capabilities the strategy calls for have no owning plan.** Context-rot monitoring (goal drift detection, context utilisation tracking, decision latency), context-budget recalibration (the standalone config fix), and token-budget communication to agents all need explicit ownership. The strategy report's "Immediate Action Plan" item 3 (context budget recalibration) is a five-minute config change that has no plan ID — it should be done as part of P51 or P52 rather than tracked separately.

- **P44's `dispatch_task` partially subsumes P51's handoff fix and P52's dispatch phase, but does not make either redundant.** P51's role-routing fix and legacy-path removal remain necessary prerequisites — `dispatch_task` internalises the pipeline but still needs a correct pipeline to internalise. P52's behavioural profile (session-start audit, no-implicit-gates, ghost-work detection) addresses orchestrator-level patterns that `dispatch_task` doesn't touch — those rules must still be followed when the orchestrator decides *what* to dispatch and *when* to stop.

- **The top architectural risk is the pipeline-becomes-invisible problem.** When `dispatch_task` internalises the pipeline, any pipeline bug becomes a silent failure — the orchestrator can't see a wrong prompt and can't compose a manual fallback. P51's thorough testing + P44's Phase 1 validation on non-critical features is the correct mitigation, but this risk needs explicit acceptance from the architecture team before P44 build begins.

---

## A. Strategy Alignment

### P54 — Review Remediation Workflow

**1. How does P54 address strategy recommendations?**

P54 addresses a structural gap that the strategy report identifies implicitly but doesn't call out by name: the orchestrator's manual work when a review fails. The strategy report's § "The critical distinction" table identifies that the orchestrator should receive "Lightweight profile: session-start audit, dispatch loop, close-out only" — but says nothing about what happens when a review *fails* and the feature needs rework. P54 fills that gap.

It also aligns with Horizon 1's architectural principle: "the orchestrator retains decision authority (flexible execution)" (strategy report § "Architecture Decision"). P54 gives the orchestrator a structured workflow for remediation decisions — finding extraction, ownership model, re-review closure — rather than ad-hoc manual interpretation.

**2. Tensions or contradictions?**

None. P54 explicitly defers to P53 for status/scope visibility and to P51+P52 for handoff reliability and session-start behaviour. It correctly identifies that remediation is orthogonal to fast-track: plan-level and batch-level reviews can fail for ordinary work too.

One nuance: P54's "Alternatives Considered" section rejects putting remediation into P52 fast-track — which is correct — but does not consider whether P44's `dispatch_task` could automate finding-to-task mapping. This is fine for the first version (document-driven) but should be revisited after P44 Phase 1. A `dispatch_task(category: "deep-reasoning", action: "remediate")` variant could parse review reports and propose remediation dev-plans automatically.

**3. Is P54 correctly sequenced?**

P54's stated dependencies are correct: it depends on P51 (handoff reliability) but not on P52 or P44. However, P54's full value — automated finding extraction, task creation, and re-review dispatch — benefits substantially from P44's `dispatch_task`. The current design sensibly starts document-driven and defers automation. It should follow P51 and run in parallel with P52, with the automated phase gated on P44 Phase 1.

---

### P52 — Fast-Track Orchestration Profile

**1. How does P52 address strategy recommendations?**

P52 is the **primary Horizon 0 deliverable**. It directly implements the strategy report's § "Horizon 0: Fix Fast-Track Now" recommendation #1. Every element of the 3-phase profile maps to a specific P50 failure:

| P50 Failure | P52 Fix |
|---|---|
| Orchestrator stopped 3 times at arbitrary breakpoints | Phase 0-2 removes all structural breakpoints; anti-pattern "Implicit Gate" |
| Ghost work discovered mid-implementation (F4/T4) | Phase 0 session-start audit: cross-reference task descriptions against existing code |
| Entity state ambiguity between user summaries and YAML | Phase 0: trust `status()` output only, not conversation |
| `handoff` producing orchestrator training material | Phase 1 explicitly uses `role: "implementer-go"` |
| Orchestrator following "stop at 60%" rule in fast-track | Removed entirely from fast-track profile |

**2. Tensions or contradictions?**

The strategy report recommends P52 as a "one-day skill documentation change" (strategy report § "Immediate Action Plan"). P52's own design says the same. No contradiction.

However, P52 correctly identifies that the "just tell the orchestrator not to stop" alternative doesn't work — the strategy report concurs, framing P50 as a "context engineering problem" where "the map itself created confusion" (strategy report § "The headline finding"). P52's profile-as-structural-fix approach is exactly what the strategy calls for.

**3. Is P52 correctly sequenced?**

Yes — Horizon 0, this week. P52's design explicitly notes that P44 will eventually replace the dispatch mechanism but the behavioural profile remains valid. The strategy report confirms: "P52 gives fast-track features a lightweight profile now and a dispatch mode target for P44's `fast_track` flag."

---

### P51 — Handoff Pipeline Unification

**1. How does P51 address strategy recommendations?**

P51 implements Horizon 0 recommendation #2: "Fix sub-agent role routing: when sub-agents are defined, default to sub-agent role/skill instead of orchestrator role/skill" (strategy report § "Handoff Pipeline Unification").

More foundationally, P51 removes the dual-path architecture that the strategy report identifies as a structural contributor to context rot. The strategy report's § "Why this happens" section lists "Dual-path architecture" as factor #1 in the manual-prompt gap. P51 eliminates this by making the 3.0 pipeline unconditional.

P51 also directly addresses the concrete P50 incident: the orchestrator received `handoff` output with orchestrator training material (~9K tokens of `orchestrate-development`), recognised the mismatch, and manually composed 12 prompts. P51's pipeline default change ensures this can't happen — `handoff(task_id)` without an explicit role defaults to `implementer-go`.

**2. Tensions or contradictions?**

None. The strategy report calls P51 a "one-day code cleanup" and a "prerequisite cleanup" for P44. P51's own design confirms this.

One subtlety: P51's design mentions adding a `topic` field to `asmTrimmedEntry` so the orchestrator can see *what* was trimmed. The strategy report doesn't discuss trimming visibility, but this is consistent with the strategy's emphasis on "budget awareness must be visible to the receiving agent."

**3. Is P51 correctly sequenced?**

Yes — Horizon 0, this week, and explicitly a prerequisite for P44. The strategy report states: "P51 cleans up the pipeline so P44 has a single path to internalize." P51's own design section "Interaction with P51 (immediate fix)" in P44 confirms this relationship.

---

### P44 — Model Routing & Agent Launcher

**1. How does P44 address strategy recommendations?**

P44 is the **strategic architecture decision**. It implements Horizon 1 recommendations #4 and #5 in full:

- `dispatch_task` collapses `next` → `handoff` → `spawn_agent` into a single tool call (strategy report § "Model Routing Dispatch")
- Session-scoped context eliminates per-claim redundancy (strategy report identifies this as "~300KB of redundant context" for 12 tasks)
- Provider routing, compaction triggers, and token budget communication all map to strategy report elements
- `fast_track` dispatch mode automates the P52 behavioural profile within the dispatch loop

The strategy report's architecture decision explicitly selects P44's approach over chat-as-orchestrator mitigations alone and state-machine orchestration.

**2. Tensions or contradictions?**

The strategy report says P44 is a "feasibility design only" and should "not commit to building until P42 and P43 are stable." However, the strategy report's own "Immediate Action Plan" schedules P44 Phase 1 for "This Quarter" — which implies the feasibility design has been reviewed and the decision to build has been made (or will be made imminently). P44's design document still says "Do not commit to building until A and B are stable."

**This is a sequencing tension:** if P44 Phase 1 starts this quarter, the "wait for A and B to be stable" constraint needs to be evaluated against whether P43 (B) is stable enough. P43's validators are "already built and working" (strategy report § "Automated Validators — Already Built"). If that's the case, the constraint is met and P44 can proceed. If not, the Horizon 1 timeline is at risk.

**3. Is P44 correctly sequenced?**

Yes — Horizon 1, this quarter, after P51 cleanup and P52 behavioural profile are in place. The strategy report confirms this ordering and P44's design explicitly describes the P51 → P44 dependency.

---

### P43 — Fast-Track Architecture (Validators)

**1. How does P43 address strategy recommendations?**

P43 is the **proof of pattern** for Horizon 2. The strategy report § "Automated Validators (P43) — Already Built" positions P43 as "the working proof that fresh-session, rubric-based sub-agent dispatch works." Its validators run in clean sessions with only the document under review and the validator rubric — the exact pattern that `dispatch_task` generalises.

P43 also directly addresses the strategy's emphasis on "enforceable constraints beat advisory instructions" (strategy report references MetaGPT SOPs and Microsoft programmatic gates). P43's stage transition hooks are enforceable — `entity(action: "transition")` returns an error with validator findings if a blocking check failed.

**2. Tensions or contradictions?**

None. P43's `dispatch_validator` abstraction is designed for forward compatibility with P44 — it uses `spawn_agent` today and routes through model routing dispatch when P44 is built. The strategy report confirms this as the correct pattern.

One open question from the strategy report: "Should validators use a different model than authors/reviewers?" P43's design says yes — validators are compliance-audit profile, value consistency over creativity. The strategy report doesn't take a position but P43's answer is architecturally sound given the research on audit tasks.

**3. Is P43 correctly sequenced?**

P43 is already built (Horizon 2 track). The strategy report says "Continue expanding validator coverage." P43's own design lists expansion items (design validator, threshold tuning). These are ongoing work, correctly positioned as parallel to P44 build.

---

## B. Dependency and Execution Order

### Where does P54 belong?

**P54 belongs in Horizon 1, between P52 and P44.** Reasoning:

- **Not Horizon 0:** P54 depends on P51 (handoff reliability for remediation task dispatch). P51 is Horizon 0. P54 also benefits from P52's session-start audit (detecting remediation state) but is not blocked by it. The strategy report's Horizon 0 items are all immediate P50 fixes — P54 addresses a workflow gap exposed by P50 but not a P50 failure itself.

- **Horizon 1, early:** P54's document-driven first version (finding extraction → remediation dev-plan → re-review checklist) can ship as soon as P51 is done. This version is procedural — the orchestrator follows a documented workflow. It doesn't require `dispatch_task`.

- **Horizon 1, later (automated phase):** When P44 Phase 1 delivers `dispatch_task`, P54 can add automated finding-to-task mapping. A `dispatch_task(category: "deep-reasoning", action: "remediate")` call could parse a review report, propose remediation task groupings, and generate a draft dev-plan. This is the natural evolution of P54 from documented workflow → tool-assisted workflow.

**Recommended placement:**

```
Horizon 0 (this week):  P51 → P52 + context budget recalibration
Horizon 1 (this quarter): P54 (document-driven) → P44 Phase 1 → P54 (automated phase)
Horizon 2 (ongoing):     P43 expansion, context-rot monitoring
```

P54 can start in parallel with P44 feasibility review, ship its document-driven version before P44 Phase 1 begins, and add automation after `dispatch_task` is available.

---

## C. Architecture Integrity

### 1. Do these plans form a coherent architecture?

**Yes.** Together, they form a pipeline-enforced, context-rot-resilient orchestration system:

| Layer | Plan | Role in Architecture |
|---|---|---|
| **Pipeline Integrity** | P51 | Single, correct assembly path. No silent degradation. |
| **Orchestrator Behaviour** | P52 | Lightweight profile for fast-track; session-start audit prevents ghost work and state ambiguity |
| **Review Closure** | P54 | Structured remediation when reviews fail; finding-to-task traceability |
| **Dispatch Architecture** | P44 | Non-bypassable pipeline; session-scoped context; provider routing; automated compaction |
| **Quality Gates** | P43 | Fresh-session validators with enforceable stage transitions; proof of pattern for `dispatch_task` |

The layers stack: P51 ensures correctness → P52 ensures appropriate behaviour → P54 closes the review loop → P44 makes the pipeline the only path → P43 validates the outputs.

**What is missing:**

1. **No plan owns context-rot monitoring** (strategy report § "Context Rot Monitoring"). Goal drift detection, context utilisation tracking, and decision latency instrumentation are recommended but unassigned. They could live in P44 (since `dispatch_task` has token tracking) or as a new P5x monitoring plan.

2. **No plan owns context-budget recalibration** as a standalone deliverable. The strategy report lists it as Horizon 0 item #3 ("Recalibrate context budgets"): update `DefaultContextWindowTokens` to 1,000,000 and raise `assemblyDefaultBudget`. This is a five-minute config change. It should be done as part of P51's implementation (since P51 touches `assembly.go`) or P52's, but it needs an explicit owner so it doesn't fall through the cracks.

3. **No plan owns token-budget communication to agents** (strategy report/P44 § "Agent-Facing Token Budget Communication"). P44 designs the mechanism but it's part of the Phase 1 `dispatch_task` scope, not a separate plan. This is fine — it's a feature of `dispatch_task`, not a standalone capability — but it should be explicitly listed in P44's Phase 1 acceptance criteria to ensure it doesn't get deferred.

### 2. Overlapping responsibilities

| Plans | Overlap | Risk |
|---|---|---|
| P51 + P44 | Both address the manual-prompt gap. P51 fixes the pipeline default and removes the legacy fallback. P44 eliminates the gap entirely by making `dispatch_task` the only dispatch path. | **Low.** P51 is a prerequisite, not a competitor. P51 ensures the pipeline is correct; P44 ensures it's the only option. |
| P52 + P44 | P52's Phase 1 (Dispatch) uses `handoff` + `spawn_agent`. P44 replaces this with `dispatch_task`. Both define a `fast_track` mode. | **Low.** P52 explicitly says "When P44's `dispatch_task` arrives, the fast-track profile's Dispatch phase changes... the behavioral rules remain the same." P52's Phase 0 and Phase 2 (audit and close-out) are orchestrator-level behaviours that `dispatch_task` doesn't touch. |
| P43 + P44 | P43's `dispatch_validator` abstraction today uses `spawn_agent`; when P44 is built, it routes through the `audit` category. | **Low by design.** P43 explicitly includes forward-compatible `dispatch_validator` abstraction. The transition is a configuration change, not a code change. |
| P53 + P54 | P53 owns status/scope visibility and dirty-work attribution. P54 consumes these. | **Low.** P54 explicitly defers to P53 rather than duplicating. |

No high-risk overlaps. The plans have been designed with awareness of each other's boundaries.

### 3. Unassigned capabilities from the strategy report

| Strategy Capability | Owning Plan? | Recommendation |
|---|---|---|
| Context-rot monitoring (goal drift, utilisation, latency) | **None** | Create a lightweight P5x monitoring plan or assign to P44 Phase 2. The instrumentation hooks should be added during P44 Phase 1 even if the dashboard comes later. |
| Context-budget recalibration | **None** | Assign to P51 implementation. It's a config change in `assembly.go` and `pipeline.go`; natural to do alongside the P51 cleanup. Add explicit acceptance criterion to P51. |
| Token-budget communication to agents | P44 (implicit) | Already designed in P44 § "Agent-Facing Token Budget Communication." Ensure it's in P44 Phase 1 acceptance criteria so it ships with `dispatch_task`. |
| Compaction procedural triggers (pre-P44) | P44 (Phase 3a) | P44's Phase 3a is "immediate, no dependencies" — the procedural triggers can be implemented today. Consider pulling Phase 3a into Horizon 0 as part of P52's close-out phase. |
| `finish` summary limit documentation | P51 (partial) | P51's open questions mention documenting the 500-character limit. Assign to P51. |

### 4. Impact of `dispatch_task` on P51, P52, and P54

**P51 — Partially subsumed, not redundant.**

When `dispatch_task` arrives:
- P51's **legacy path removal** is a prerequisite — `dispatch_task` internalises the 3.0 pipeline, so there must be only one pipeline to internalise. This work remains essential.
- P51's **role routing fix** becomes less critical at the `handoff` level (because orchestrators won't call `handoff` for sub-agent dispatch) but remains important for any direct `handoff` usage and for `next` output. The pipeline default change should still ship.
- P51's **trimming metadata** improvement (topic-level detail in trimmed entries) becomes less critical because `dispatch_task` doesn't go through the 30KB MCP response cap — but `next` still does, so it's still valuable.

**Verdict:** P51 is not redundant. Ship as designed.

**P52 — Partially subsumed, not redundant.**

When `dispatch_task` arrives:
- P52's **Phase 1 (Dispatch)** changes: `handoff` + `spawn_agent` → `dispatch_task(category: "implementation")`. The mechanism changes; the behavioural rules don't.
- P52's **Phase 0 (Session Start Audit)** is orchestrator-level behaviour that `dispatch_task` doesn't touch. Ghost-work detection, state-ambiguity resolution, dirty-work classification, and `server_info` verification all remain orchestrator responsibilities.
- P52's **Phase 2 (Close-Out)** remains orchestrator-level. Feature transitions, completion reporting, and verification are decisions the orchestrator makes.
- P52's **anti-patterns** (Implicit Gate, Ghost Work Discovery, State Ambiguity Drift, Milestone Pause) all remain relevant — they govern orchestrator behaviour *between* `dispatch_task` calls.

**Verdict:** P52's behavioural profile remains valid. The dispatch mechanism changes but the behavioural rules are invariant. P52 should ship as designed, with a note that Phase 1's dispatch mechanism is replaced by `dispatch_task` when P44 Phase 1 arrives.

**P54 — Enhanced, not redundant.**

When `dispatch_task` arrives:
- P54's document-driven first version is unaffected — it's a procedural workflow the orchestrator follows.
- P54's automated phase (finding extraction → remediation dev-plan) can use `dispatch_task(category: "deep-reasoning")` to parse review reports and propose remediation plans. This is an enhancement, not a replacement.
- P54's remediation task dispatch can use `dispatch_task(category: "implementation")` for implementing fixes, same as any other implementation task.

**Verdict:** P54 is not redundant. `dispatch_task` enables P54's automated phase but doesn't replace the workflow itself.

---

## D. Risk Assessment

### Top 3 Architectural Risks

#### Risk 1: Pipeline becomes invisible (the silent-failure problem)

**Severity:** **High**  
**Affected plans:** P44, P51, P52  
**Description:** When `dispatch_task` internalises the pipeline, the orchestrator never sees the assembled prompt. Today, if `handoff` produces wrong output (as in P50), the orchestrator can detect the mismatch and manually compose prompts. After P44, a pipeline bug — wrong role resolution, missing knowledge entries, truncated spec sections — produces a silently degraded sub-agent with no orchestrator visibility and no manual fallback.

The strategy report's risk register identifies a related risk: "P44 `dispatch_task` has no fallback if the pipeline produces wrong context" (severity: Medium). This assessment raises it to **High** because the P50 incident demonstrated that pipeline misconfiguration happens in practice, and the orchestrator's ability to detect and compensate was the safety net.

**Mitigation:**
1. P51 must be thoroughly tested before P44 Phase 1 begins — the pipeline must be proven correct before it becomes invisible.
2. P44 Phase 1 should include pipeline health assertions: before each `dispatch_task` call, verify that role resolution, skill loading, knowledge assembly, and code-graph context all succeeded.
3. P44 should maintain a "pipeline debug" mode (gated behind a config flag) that returns the assembled prompt alongside the result for human inspection during validation.
4. First 20 `dispatch_task` runs should be on non-critical features with human comparison of pipeline output vs. expected context.
5. **Acceptance gate:** P44 must not enter production until 20 consecutive `dispatch_task` calls produce correct role/skill/knowledge context verified by human audit.

#### Risk 2: P52 behavioural profile is necessary but insufficient

**Severity:** **Medium**  
**Affected plans:** P52, P44  
**Description:** P52 addresses the structural breakpoints that caused P50's stops by removing phases and adding anti-patterns. But the strategy report notes: "the 'summary output → wait' pattern is deep in training data. Monitor the first 5 fast-track runs for any remaining stop behavior." If the model's pattern-matching on "summary output → wait for response" is strong enough, even a profile without structural breakpoints may not prevent all implicit gates.

Compounding this: P52's Phase 0 session-start audit is valuable but the orchestrator must remember to run it. If context rot degrades the orchestrator's adherence to the profile over long sessions, the audit could be skipped — and ghost work or state ambiguity would go undetected.

**Mitigation:**
1. P44's `fast_track` dispatch mode should *automate* the session-start audit checks that are mechanical (cross-reference task state vs. code, verify `server_info` binary currency). The orchestrator shouldn't need to remember — `dispatch_task` should refuse to dispatch if the audit hasn't run.
2. Track implicit-gate frequency as a metric. If P52 reduces but doesn't eliminate implicit gates, the remaining gates should be analysed for common patterns and fed back into the anti-pattern list.
3. Consider adding a "no-stop contract" preamble to the fast-track profile that explicitly states: "You are in fast-track mode. You will NOT stop for confirmation at any point. The ONLY valid stop conditions are: all work done, build failure, missing dependency, or spec ambiguity."

#### Risk 3: Compaction artefact quality is unproven in production

**Severity:** **Medium**  
**Affected plans:** P44, P52  
**Description:** The U-shaped compaction artefact template is well-researched (Liu et al. 2024, Anthropic engineering) but has never been tested in Kanbanzai's production workflow. The evaluation metrics (task completion rate, decision consistency, token efficiency) are defined but won't have data until 20+ compaction events have occurred. If the artefact template has blind spots — decisions not captured, knowledge references that fail to resolve, continuation anchors that are ambiguous — large features could experience context-rot failures even after compaction.

Additionally, the compaction trigger strategy has a procedural phase (orchestrator estimates context pressure) and an automated phase (token-count-based triggers). The procedural phase relies on the orchestrator accurately estimating its own context utilisation — the same orchestrator that P50 showed can be unreliable under context pressure.

**Mitigation:**
1. Run the first 10 compactions with human oversight: a human reviews the artefact before the fresh session starts, comparing against the pre-compaction state.
2. Implement Phase 3a procedural compaction triggers immediately (no dependencies on model routing) and validate on real features before P44 Phase 3b automated triggers.
3. Add a "compaction dry-run" mode: the orchestrator produces a compaction artefact but doesn't hand off. A human reviews the artefact for completeness. Only after 5 successful dry-runs does actual compaction begin.
4. Track compaction failure modes explicitly: decision inconsistency, missed knowledge references, ambiguous continuation anchors. Feed findings back into the template.

---

## Summary of Recommendations

| # | Recommendation | Priority |
|---|---|---|
| 1 | Assign context-budget recalibration to P51 implementation (add acceptance criterion) | Immediate |
| 2 | Sequence P54 between P52 and P44: document-driven version after P51, automated phase after P44 Phase 1 | This quarter |
| 3 | Create explicit ownership for context-rot monitoring (new P5x or P44 Phase 2) | This quarter |
| 4 | Add pipeline-health assertions and debug mode to P44 Phase 1 (mitigation for Risk 1) | Before P44 build |
| 5 | Track P52 implicit-gate frequency as a metric; automate session-start audit in P44 `fast_track` mode | Ongoing |
| 6 | Run first 10 compactions with human oversight; validate procedural triggers before automated triggers | Before P44 Phase 3b |
| 7 | Ensure token-budget communication is in P44 Phase 1 acceptance criteria | Before P44 build |
| 8 | Resolve the "wait for A and B to be stable" vs. "P44 this quarter" tension — confirm P43 stability gates P44 build start | Immediate |
