# Research Report: Context Pollution and Rot in Long-Running Agent Orchestration

**Status:** Draft  
**Date:** 2026-05-05  
**Author:** P41 research task (solo investigation)  
**Parent Plan:** P41-opencode-ecosystem-features  
**Related:** P41-research-context-compaction.md (compaction artefact design), P44-design-model-routing-agent-launcher.md (model routing dispatch architecture), P43-design-fast-track-architecture.md (automated validators), research-orchestration-landscape-2025.md (orchestration framework survey)

---

## Research Question

> **Is context rot an unavoidable failure mode in the chat-as-orchestrator architecture, or are there well-known strategies that can manage it effectively? Should Kanbanzai continue investing in the single-context orchestrator model with mitigations, or switch to a model routing dispatch architecture (P44)?**

### Sub-questions

1. **Characterisation.** What does the research literature say about context rot as a distinct failure mode — separable from "lost in the middle" retrieval degradation? Is there empirical evidence that agent *behaviour* (goal consistency, decision quality, instruction adherence) degrades as context grows, beyond retrieval accuracy?

2. **Prevalence across architectures.** Does context rot affect all long-context agent architectures equally, or are some architectures inherently more resistant? Compare: single-context orchestrators, multi-session orchestrators with compaction, and dispatch-loop architectures.

3. **Mitigation strategies for single-context orchestrators.** What evidence supports prompt reinjection, constraint pinning, procedural checkpoints, and context window management?

4. **Architectural alternatives.** What does the literature say about dispatch-loop architecture, state-machine orchestration, hierarchical orchestration, and event-driven orchestration?

5. **Comparison to P44.** How does Kanbanzai's planned model routing dispatch architecture compare to strategies in the literature?

6. **Implementation feasibility gradient.** For each viable strategy: infrastructure fit, model routing dependency, implementation effort, evidence confidence.

7. **Is this even fixable?** Is context rot inherent, architectural, or a context engineering problem?

---

## Scope and Methodology

### Sources consulted

- **Primary (peer-reviewed):** Liu et al. (2024, TACL) — "Lost in the Middle: How Language Models Use Long Contexts," the foundational paper on U-shaped attention curves
- **Primary (vendor technical reports):** Anthropic Engineering (2025a) "Effective Context Engineering for AI Agents"; Anthropic Engineering (2024) "Building Effective Agents"; Anthropic Engineering (2025b) "How We Built Our Multi-Agent Research System" (referenced, not directly accessed — URL 404 at time of research)
- **Secondary (industry analysis):** The New Stack (2026) "How Context Rot Drags Down AI and LLM Results for Enterprises" — Elastic-sponsored piece with practitioner and analyst perspectives; Olamendy, J.C. (2025) "Context Engineering: The Invisible Discipline" (referenced, not accessed)
- **Secondary (field reports):** Kargar (2026a, 2026b, 2026c) on LeRiM's context management and agent-to-agent optimisation (referenced via P41-research-context-compaction.md)
- **Internal Kanbanzai documents:** P41-research-context-compaction.md (full compaction research, 20+ sources); P41-research-context-compaction-summary.md (compaction artefact template and implementation guide); P44-design-model-routing-agent-launcher.md (model routing dispatch architecture, §Compaction, §Enforcement, §Fast-track pipeline mismatch); P43-design-fast-track-architecture.md (validator architecture, context isolation); research-orchestration-landscape-2025.md (orchestration framework survey)
- **Kanbanzai retrospective signals:** retro(action: "synthesise") — 26 signals spanning tool-friction, workflow-friction, and worked-well observations from P41-P50 implementation
- **Date range:** 2023–2026, with emphasis on 2025–2026 for the architectural comparison question

### Search strategy

- Seed sources from the task brief: Anthropic Engineering posts, The New Stack, Liu et al. (2024)
- Internal document traversal: P41 compaction research → cited sources → follow-up verification
- Retro signal synthesis for empirical evidence of context degradation within Kanbanzai's own development history
- The Chroma Research and MorphLLM blog URLs returned 404 errors; their claims are noted via secondary references in the compaction research
- The Anthropic multi-agent research system post returned 404 but its claims are preserved via the compaction research's detailed summary

### Evaluation criteria

1. **Source quality** — peer-reviewed vs. vendor report vs. blog post vs. internal observation
2. **Recency** — 2023–2024 (8K–128K token regime) vs. 2025–2026 (200K–1M token regime)
3. **Directness of evidence** — does the source directly test agent behaviour under context growth, or does it test a related phenomenon (e.g., retrieval accuracy)?
4. **Architectural relevance** — does the evidence apply to single-context orchestrators specifically, or to long-context LLM use generally?
5. **Falsifiability** — could the claim be disproven by new evidence?

### Excluded from scope

- KV-cache-level compression (Kanbanzai does not control model inference infrastructure)
- Training custom compaction models (P44 is the prerequisite; out of scope for current decision)
- RAG-specific context pruning (related but different problem framing)
- Needle-in-a-haystack retrieval benchmarks as evidence of context rot immunity (per the task brief's anti-pattern: "Confusing Retrieval Accuracy with Behavioural Quality")

---

## Findings

### Finding 1: Context rot is a distinct, empirically documented failure mode separate from "lost in the middle" retrieval degradation

**Evidence:**

- **Anthropic (2025a)** — Primary (vendor technical report, Sep 2025). Defines context rot explicitly: "as the number of tokens in the context window increases, the model's ability to accurately recall information from that context decreases." Notes this "emerges across all models" and is rooted in architectural constraints: the n² pairwise attention mechanism and training data distributions biased toward shorter sequences. Crucially, Anthropic distinguishes this from pure retrieval: "needle-in-a-haystack style benchmarking has uncovered the concept of context rot" — the retrieval task was the *diagnostic tool*, not the phenomenon itself. The real behavioural consequence: "models lose focus or experience confusion at a certain point."

- **Liu et al. (2024)** — Primary (TACL, peer-reviewed). The U-shaped attention curve is a structural property: models attend well to information at the beginning and end of context but suffer significant degradation for information in the middle. Multi-document QA and key-value retrieval tested across GPT-3.5, Claude, Flan-T5, and others. The degradation persists even when relevant passages are cued with explicit markers — it's not merely a recency preference but a structural attention limitation. However, this paper tested retrieval accuracy, not agent *behavioural* consistency. The transfer from "finding a fact" to "maintaining goal-directed behaviour across 100+ turns" is plausible but not directly tested.

- **The New Stack / Elastic (Mar 2026)** — Secondary (sponsored industry piece). Practitioner evidence: Abhimanyu Anand (Senior Data Scientist, Elastic) describes agents falling into loops, hallucinating after ~10 minutes of searching, and experiencing degraded reasoning with large contexts. Analyst James Kobielus (Franconia Research) frames context rot as "a form of technical debt intrinsic to any deployed LLM" that "can silently undermine the continued accuracy and effectiveness of any model, no matter how well-trained upfront." These are behavioural failure modes — looping, hallucination, reasoning degradation — not retrieval accuracy problems.

- **Kanbanzai internal: P50 incident** — Primary (direct observation). During fast-track implementation (16 tasks, 4 features, May 2026), the orchestrator drifted from its fast-track purpose — stopping mid-pipeline for human confirmation despite the fast-track tier having zero human gates. The `orchestrate-development` skill's "stop at 60% context" instruction contradicted the fast-track tier's "no gates" constraint, and the orchestrator followed the skill rather than the tier. The tier constraint was effectively forgotten as context grew. This is a direct observation of goal drift in a single-context orchestrator — the orchestrator's original constraint (no gates, no stops) was diluted by the accumulated context of the skill procedure.

**Key claims:**
- Context rot manifests as behavioural degradation (goal drift, instruction forgetting, inconsistent decision-making), not just retrieval accuracy loss.
- The U-shaped attention curve is a structural property of transformer architectures, not model-specific.
- Larger context windows (1M tokens) buy time but do not eliminate the problem — Anthropic explicitly notes that "context windows of all sizes will be subject to context pollution and information relevance concerns."

**Source grading:** High confidence for the existence of context-driven performance degradation. Medium confidence for the specific claim that *behavioural* degradation is distinct from *retrieval* degradation — the direct evidence (P50, Elastic practitioner reports) is observational, not controlled-experimental. The Liu et al. paper provides the mechanistic explanation (attention curve) but not the behavioural measurement.

---

### Finding 2: Context rot is an architectural consequence, not an inherent limitation — architectures differ substantially in their susceptibility

**Evidence:**

- **Anthropic (2024)** — Primary (vendor technical report, Dec 2024). "Consistently, the most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns." Recommends orchestrator-workers, evaluator-optimizer, and routing patterns. The key architectural insight: isolation of context into separate sessions per sub-agent. "The detailed search context remains isolated within sub-agents, while the lead agent focuses on synthesizing and analyzing the results."

- **Anthropic (2025a)** — Primary. Three distinct strategies for long-horizon tasks, each with different context profiles: (1) **compaction** — maintains conversational flow but requires careful curation of what to keep vs. discard; (2) **structured note-taking** — persistent memory outside the context window, pulled back in on demand; (3) **sub-agent architectures** — "specialized sub-agents can handle focused tasks with clean context windows... each subagent might explore extensively, using tens of thousands of tokens or more, but returns only a condensed, distilled summary." The sub-agent architecture is explicitly positioned as a solution to context limitations.

- **Kanbanzai internal: P43 validators** — Built and in use. Validators always run in fresh sessions via `spawn_agent`. The orchestrator uses them for spec validation, plan validation, and review gate auditing — all tasks that require careful attention to structured criteria. No context rot issues have been observed with validators because each runs in a clean session with only the document under review and the validator rubric.

- **Kanbanzai internal: P44 design** — The `dispatch_task` tool collapses `next` → `handoff` → `spawn_agent` into one call. The orchestrator never sees raw sub-agent prompts. Session-scoped context eliminates repeated knowledge entry assembly across task claims. This is the sub-agent architecture Anthropic describes, applied to Kanbanzai's specific workflow.

- **Kanbanzai internal: P44 §Enforcement** — Documents the "manual-prompt gap": the orchestrator received `handoff` output, recognized the content mismatch (orchestrator skill vs. implementer skill), and manually composed 12 prompts. Each manual prompt was ~400-600 tokens, missing spec sections, knowledge entries, and code graph context. Despite this, all 12 tasks completed — but the orchestrator's role constraints ("Always use handoff") were violated because the tool output was unusable. This is context pollution causing an architectural failure: the pipeline produced the wrong role's context, the orchestrator compensated by bypassing the pipeline entirely.

**Key claims:**
- Single-context orchestrators accumulate all conversation history — tool outputs, reasoning chains, sub-agent summaries — in one growing context window. Every token competes for attention budget.
- Sub-agent architectures isolate context: the orchestrator sees only task summaries, not the implementation details. This prevents context pollution from sub-agent work.
- Dispatch-loop architectures (P44) go further: the orchestrator never sees raw prompts at all. The dispatch loop owns provider API calls; the orchestrator calls `dispatch_task(task_id, category)` and receives structured results.

**Source grading:** High confidence that architectures differ in context susceptibility — this is triangulated across Anthropic's production systems, Kanbanzai's P43 validators, and the P50 manual-prompt gap incident. Medium confidence on the *magnitude* of difference — no controlled experiment compares architectures head-to-head on the same workflow.

---

### Finding 3: Prompt reinjection and constraint pinning have moderate theoretical support but limited direct experimental evidence for agent orchestration

**Evidence:**

- **Anthropic (2025a)** — Primary. Recommends organizing prompts into distinct sections with XML/Markdown headers. Emphasises "the minimal set of information that fully outlines your expected behavior." Also warns against over-specifying: "brittle if-else hardcoded prompts" are a failure mode. This supports structured constraint sections at the prompt level but does not address *reinjection* mid-session.

- **Anthropic (2025a) on progressive disclosure** — Recommends "just in time" context strategies where agents "maintain lightweight identifiers... and use these references to dynamically load data into context at runtime using tools." This is constraint pinning via tool responses rather than prompt reinjection: the agent retrieves constraints on demand rather than having them re-inserted periodically.

- **Kargar (2026b)** — Secondary (field report). Reports a 41% improvement in composite quality score from prompt-level optimisation of memory extraction agents. Key finding: "schema field descriptions (20 words) had more impact than 50 lines of prompt engineering" and "restrictive rules ('don't do X') consistently backfire compared to positive guidance ('here's what good looks like')." This suggests that *what* constraints say matters more than *when* they're re-inserted.

- **Kanbanzai internal: P50 incident analysis** — The orchestrator forgot the fast-track "no gates" constraint because the `orchestrate-development` skill's "stop at 60% context" instruction was more recent and more procedurally embedded than the tier constraint, which appeared only once at session start. This is a failure of the primacy effect — the initial constraint was pushed into the attention valley and overridden by recency-weighted skill instructions. It suggests that periodic reinjection of tier constraints might have prevented the drift.

**Key claims:**
- Prompt reinjection has strong *theoretical* support from the U-shaped attention curve (Liu et al. 2024): re-inserting constraints at the end of context puts them in the recency peak.
- The *cost* of reinjection strategies is significant — every reinjection consumes tokens that could go to domain vocabulary, procedural instructions, or evidence for current decisions.
- Constraint *design* (positive vs. restrictive, concrete vs. vague) may matter more than constraint *placement* — the Kargar finding that "schema descriptions matter more than prompts" suggests format beats frequency.

**Source grading:** Low confidence for reinjection *as a proven mitigation* — no controlled experiment tests periodic constraint re-insertion against a baseline in agent orchestration. Medium confidence for the theoretical mechanism (U-shaped attention curve + primacy/recency effects).

---

### Finding 4: Dispatch-loop architecture addresses context rot at its architectural root — the orchestrator's conversation never grows beyond orchestration decisions

**Evidence:**

- **Anthropic (2025a) on sub-agent architectures** — Primary. "Rather than one agent attempting to maintain state across an entire project, specialized sub-agents can handle focused tasks with clean context windows. The main agent coordinates with a high-level plan while subagents perform deep technical work... Each subagent might explore extensively, using tens of thousands of tokens or more, but returns only a condensed, distilled summary of its work (often 1,000-2,000 tokens)."

- **Anthropic (2025a) on the hybrid strategy** — "The most effective agents might employ a hybrid strategy, retrieving some data up front for speed, and pursuing further autonomous exploration at its discretion." This maps to P44's session-scoped context: full context on first dispatch, task-specific sections only on subsequent dispatches.

- **Anthropic (2024)** — Primary. Recommends the orchestrator-workers pattern for "complex tasks where you can't predict the subtasks needed." The orchestrator dynamically breaks down tasks, delegates to workers, and synthesises results. Workers run with task-scoped context; the orchestrator only receives results.

- **Kanbanzai internal: P44 design** — The `dispatch_task` tool: (1) claims the task, (2) runs the pipeline internally, (3) routes to provider+model, (4) calls the provider API directly, (5) returns structured results. The orchestrator never sees a raw prompt. Manual composition is impossible because there's no prompt to compose. The pipeline is *the* path, not one of two options.

- **Kanbanzai internal: P44 §Enforcement** — Documents the three-step gap (`next` → `handoff` → `spawn_agent`) that dispatch_task collapses to one call. Each step is a decision point where the orchestrator can diverge. The P50 incident showed the orchestrator diverging at all three: `next` produced correct task data, `handoff` produced incorrect role context (orchestrator instead of implementer), and `spawn_agent` received manual prompts instead of pipeline output.

- **Kanbanzai internal: P44 §Repeated context** — Each `next` call during P50 returned ~30KB of context, most identical across 12 tasks. The orchestrator received the same knowledge base 12 times. Session-scoped context in `dispatch_task` eliminates this redundancy: full context on first dispatch, task-specific sections only thereafter.

**Key claims:**
- Dispatch-loop architecture prevents context pollution at its source: the orchestrator's context contains only orchestration decisions, task summaries, and structured results.
- Session-scoped context eliminates per-task-claim redundancy — the biggest source of context bloat in the current architecture.
- Making the pipeline non-bypassable (`dispatch_task` is the only dispatch path) eliminates the manual-prompt gap as a failure mode.

**Source grading:** High confidence that dispatch-loop architecture reduces context growth. This is triangulated across Anthropic's production recommendations, the P50 incident analysis, and the documented per-claim redundancy. Medium confidence on the *sufficiency* of this approach — no evidence that dispatch-loop orchestrators *never* experience context rot, only that they experience less of it.

---

### Finding 5: Compaction is a necessary complement, not a replacement, for architectural change — the two strategies address different failure modes

**Evidence:**

- **Anthropic (2025a)** — Primary. Positions compaction as the "first lever in context engineering to drive better long-term coherence." But also notes limitations: "overly aggressive compaction can result in the loss of subtle but critical context whose importance only becomes apparent later." This is the compaction-as-panacea fallacy: compaction addresses accumulated context (symptom) but not what enters context in the first place (cause).

- **P41-research-context-compaction.md** — Primary (Kanbanzai internal). Evaluated four approaches: state-based, summary-based, learnable, and retrieval-anchored. Recommendation: state-based U-shaped compaction with KE-ID anchoring. Key finding: "The art of compaction lies in the selection of what to keep versus what to discard" (quoting Anthropic 2025a). The compaction artefact captures "where to resume" rather than "what happened" — active tasks, constraints, decisions, KE-IDs.

- **P44 §Compaction** — Already designed the U-shaped compaction artefact template with: Identity & Routing, Active Constraints, Active State (Done/In Flight/Ready tables), Active Decisions, Surfaced Knowledge (KE-IDs), Continuation Anchor. Token-count graduated triggers: 60% warning, 80% hard trigger, 90% emergency. Hard cap: artefact ≤ 25% of context window.

- **The distinction:** Compaction addresses the question "what do we do when context fills up?" Architectural change (dispatch-loop) addresses the question "how do we prevent context from filling up with irrelevant information in the first place?" Both are needed. Compaction without architectural change is fighting a rearguard action — context keeps growing, compaction keeps compressing. Architectural change without compaction is betting that dispatch-loop orchestrators never need to compact — which is plausible for small-to-medium features but unproven for very large features with 20+ tasks.

**Key claims:**
- Compaction is mitigation; dispatch-loop architecture is prevention.
- The interaction: dispatch-loop architectures need compaction less often (only orchestration decisions accumulate), but when they do, the same U-shaped template applies.
- The compaction research already specified that automated triggers require model routing (P44) — compaction is *designed* to work with dispatch-loop infrastructure.

**Source grading:** High confidence that compaction and architectural change are complementary. This follows directly from the definitions: compaction manages accumulated context; dispatch-loop prevents unnecessary context accumulation.

---

### Finding 6: The fast-track pipeline mismatch (P50) was a context engineering failure, not an inherent architecture failure — and it is fixable without abandoning the chat-as-orchestrator model

**Evidence:**

- **P44 §Fast-track pipeline mismatch** — Documents specific mismatches between the `orchestrate-development` skill and fast-track features: "Stop and hand off at 60%" vs. "Zero human gates" (contradictory), cohort management for single-feature batches (irrelevant), context offloading instructions without P44 (dead instruction). The skill was designed for multi-feature, multi-cohort batch orchestration; fast-track features are small, independent, with no review cycles.

- **The root cause is dual:** (1) The skill procedure was the wrong context for the task tier — a fast-track feature should receive a lightweight orchestration profile, not the full multi-cohort procedure. (2) The orchestrator had no mechanism to detect when a skill procedure contradicts the feature's tier constraint.

- **Proposed fix (P44/P52):** A `fast_track` dispatch mode that suppresses cohort management, merge scheduling, and context offloading. A lightweight orchestration profile — task graph, file ownership, dependency status — without the full procedure. P52 defines the fast-track behavioral profile (session-start audit, no-implicit-gates rules, ghost-work detection).

- **This is a context engineering fix, not an architectural one.** The chat-as-orchestrator model can handle fast-track features if the orchestrator receives the right context — a lightweight profile instead of the full multi-cohort procedure. The failure was not that the orchestrator's context grew too large; it was that the context contained contradictory instructions (skill says stop at 60%, tier says never stop).

**Key claims:**
- The P50 incident was a context *design* failure (wrong skill for the tier), not a context *capacity* failure (too many tokens).
- Fixing this requires the pipeline to recognise feature tier and suppress irrelevant sections — a context engineering change, not an architecture change.
- This fix can be implemented in the current architecture (by modifying what `handoff` assembles for fast-track features) or in the future architecture (by making `dispatch_task` mode-aware).

**Source grading:** High confidence that the P50 failure mode is fixable via context engineering. The specific mechanism (mode-aware context assembly) is designed but not yet built. Medium confidence that context engineering alone is sufficient for fast-track features — this depends on whether the lightweight profile is comprehensive enough to guide implementation without the full procedure.

---

### Finding 7: State-machine and event-driven orchestration are theoretically cleaner but practically less flexible than chat-as-orchestrator with mitigations

**Evidence:**

- **Kanbanzai internal: research-orchestration-landscape-2025.md** — Evaluates production orchestration tools and frameworks. Key finding: "The value is in what the tools do, not in the loop machinery." The dispatch tools (`next`, `handoff`, `spawn_agent`, `finish`) are the value; whether they're called by a chat agent in a loop or by a deterministic state machine is secondary.

- **Network-AI (Jovancoding)** — An MCP server with FSM + blackboard architecture. "Race-condition-safe multi-agent blackboard." Explicit state machine transitions for workflow stages. This is the state-machine orchestration pattern: deterministic transitions, agents only run at stage boundaries. However, Network-AI has a small community and no evidence of production use at scale.

- **Agent-MCP (rinadelph)** — "Obsidian for your AI agents" — a living knowledge graph where agents collaborate. Ephemeral agent model: "short-lived agents, each scoped to a single task." This is the dispatch-loop pattern implemented as an MCP server. 1.2k stars, AGPL license, described as having a steep learning curve.

- **The trade-off:** State-machine orchestration is predictable and auditable — you always know what stage you're in and what happens next. But it's rigid: unexpected situations (ambiguous review findings, circular dependencies, edge cases) require human intervention. Chat-as-orchestrator is flexible: the orchestrator can handle edge cases dynamically. But it's unpredictable: the same situation handled by two different orchestrator sessions may produce different outcomes.

- **Anthropic (2024)** — Recommends workflows (predefined code paths) for well-defined tasks and agents (LLM-directed processes) for flexible tasks. This suggests a hybrid: deterministic stage transitions with agent-driven execution within stages.

**Key claims:**
- State-machine orchestration is a valid alternative but trades flexibility for predictability.
- The chat-as-orchestrator model's flexibility is valuable for edge cases that a deterministic state machine would block on.
- A hybrid approach (state machine for stage transitions, agent for execution within stages) may capture the best of both — this is essentially what dispatch_task enables: the dispatch loop enforces pipeline use (deterministic), but the orchestrator decides which tasks to dispatch and in what order (flexible).

**Source grading:** Low confidence for state-machine orchestration as a standalone recommendation — no production-scale evidence exists outside of Network-AI (small community) and AutoGen (enterprise but not MCP-native). Medium confidence for the hybrid approach — triangulated from Anthropic's workflow/agent distinction and Kanbanzai's existing stage bindings.

---

## Trade-Off Analysis

Three architectures compared across eight dimensions:

| Dimension | Chat-as-Orchestrator + Mitigations (Current Path) | Model Routing Dispatch — P44 (Kanbanzai Planned) | State-Machine Orchestration (Alternative) |
|-----------|--------------------------------------------------|--------------------------------------------------|------------------------------------------|
| **Context rot susceptibility** | **Medium-High.** Orchestrator accumulates all conversation. Mitigations (compaction, fast-track profiles, context trimming) reduce but don't eliminate risk. P50 incident demonstrates failure mode. | **Low.** Orchestrator only sees orchestration decisions and task summaries. Sub-agents run in fresh sessions. Session-scoped context eliminates per-claim redundancy. Pipeline is non-bypassable. | **Very Low.** No growing conversation. Deterministic state transitions. Agents only run at stage boundaries with targeted context. |
| **Flexibility / edge case handling** | **High.** Orchestrator can handle ambiguity, circular dependencies, unexpected results dynamically. The P50 manual-prompt gap shows both the risk and the value of this flexibility. | **High.** Orchestrator retains decision-making authority (which tasks, what order, how to handle results). Pipeline enforcement constrains *prompt assembly*, not *decision-making*. | **Low.** Deterministic state machines block on unexpected states. Edge cases require human intervention or state machine modification. |
| **Implementation effort** | **Small (incremental).** Compaction template designed. Fast-track profile designed (P52). Context trimming (`assemblyDefaultBudget` recalibration) is configuration. Immediate mitigations are low-effort. | **Large.** New `dispatch_task` MCP tool. Provider integration (Anthropic + DeepSeek + OpenAI). Category system, routing config, session-scoped context. Estimated Phase 1: 4–6 weeks of implementation. | **Very Large.** Would require rearchitecting the entire orchestration layer. Stage bindings, MCP tools, and orchestrator role would all need redesign. Not incremental. |
| **Evidence confidence** | **Medium.** Mitigations are well-grounded in research (Liu et al., Anthropic) but not tested as a complete system. P50 incident is a single data point. | **Medium-High.** Sub-agent architecture is Anthropic's recommended pattern. P44 design is grounded in P41 compaction research. P43 validators (fresh-session dispatch) validate the pattern. But `dispatch_task` itself is unbuilt and untested. | **Low.** No production-scale evidence. Network-AI is a small project. FSM + MCP pattern is conceptually clean but empirically unvalidated for complex workflows. |
| **Pipeline enforcement** | **Weak.** Manual-prompt gap exists because `spawn_agent` accepts arbitrary text. P51 removes legacy fallback. Behavioural guardrails ("Always use handoff") are advisory. | **Strong.** `dispatch_task` internalises the pipeline. Manual composition is impossible — there's no prompt to compose. The pipeline is the only path. | **Strong.** State machine enforces transitions. Agents only run at defined points. No room for manual bypass. |
| **Context efficiency** | **Low.** Per-claim redundancy (~30KB × N tasks). Three-tool-call sequence overhead. Byte budget cap (30KB) silently drops content. | **High.** Session-scoped context. Full assembly on first dispatch, task-specific only thereafter. No MCP response cap (provider API call, not JSON response). | **Very High.** Only orchestration decisions in orchestrator context. Sub-agents receive only task-specific context. |
| **Infrastructure fit** | **Full.** Uses existing MCP tools (`next`, `handoff`, `spawn_agent`). Mitigations are procedural or configuration changes. No new infrastructure. | **Partial.** New `dispatch_task` tool, provider integrations, routing config. Uses existing pipeline-3.0 and stage bindings. Builds on P43's validator pattern. | **Poor.** Would require replacing MCP tools with state machine transitions. Loses the chat agent's ability to use existing tools for exploration and debugging. |
| **Migration path** | **N/A — current state.** | **Incremental.** P51 removes legacy fallback → P44 Phase 1 adds `dispatch_task` → Phase 2 adds automated compaction triggers → Phase 3 evaluates and tunes. Each step is independently valuable. | **Disruptive.** Would require abandoning the existing orchestration layer and rebuilding. No incremental path. |

---

## Recommendations

### Recommendation 1: Continue investing in the chat-as-orchestrator model with near-term mitigations (confidence: high)

**What this means:** Do not abandon the current architecture. The evidence shows that context rot is a manageable failure mode, not an inherent limitation. The P50 incident was a context engineering failure (wrong skill for the tier) that is fixable without architectural change. Three mitigations are actionable immediately:

1. **Fast-track profile (P52 design, can be implemented as `handoff` mode).** When a feature's tier is `fast_track`, the pipeline should suppress cohort management, merge scheduling, and context offloading sections. The orchestrator should receive a lightweight profile — task graph, file ownership, dependency status — without the full `orchestrate-development` procedure. This prevents the "skill contradicts tier" failure mode observed in P50.

2. **Context budget recalibration.** The `assemblyDefaultBudget` of 30,720 bytes and `DefaultContextWindowTokens` of 200,000 are stale. Current models have 1M token windows. Recalibrating `DefaultContextWindowTokens` to 1M would make the 40%/60% pipeline thresholds 5× more generous (400K/600K tokens vs. 80K/120K). The 30KB byte budget cap should also be raised or eliminated for `handoff` responses.

3. **Constraint section in `next` context.** The fast-track tier constraint ("never stop for human confirmation") should appear in every `next` response, not just at session start. This is constraint pinning — using the recency peak to maintain awareness of the tier's critical constraint. Minimal token cost (~15-20 tokens per response).

**Evidence basis:** Finding 6 (P50 was a context engineering failure, fixable without architecture change). Finding 3 (constraint pinning has theoretical support from U-shaped attention curve). P44 §Fast-track pipeline mismatch (specific mismatches documented, fast-track mode designed).

**Risk:** These mitigations are procedural (modify what `handoff` assembles) and configurational (recalibrate budgets). They don't address the deeper architectural issue — per-claim redundancy and the manual-prompt gap. If the orchestrator encounters a situation the lightweight profile doesn't cover, it may still drift. **Mitigation:** Position these as *temporary* mitigations while P44 is built. They buy time, not permanence.

---

### Recommendation 2: Build P44 model routing dispatch as the strategic architecture (confidence: high)

**What this means:** P44 is the correct architectural response to context rot in the chat-as-orchestrator model. The evidence strongly supports three design decisions:

1. **`dispatch_task` internalises the pipeline** — This eliminates the manual-prompt gap (P50's root cause: the orchestrator bypassed `handoff` because it produced incorrect role context). The orchestrator calls one tool; the pipeline is non-bypassable. This addresses the enforcement problem at its architectural root.

2. **Session-scoped context** — Full assembly on first dispatch, task-specific only thereafter. This eliminates the per-claim redundancy documented in P50 (same ~30KB × 12 tasks = 360KB of repeated context). The orchestrator's context grows linearly with orchestration decisions, not quadratically with task count.

3. **`fast_track` dispatch mode** — The pipeline should recognise the feature's tier and suppress irrelevant sections. This is the systematic fix for the P50 mismatch: the orchestrator never receives contradictory instructions because the pipeline selects the right profile for the tier.

**Evidence basis:** Finding 4 (dispatch-loop architecture addresses context rot at its root). Finding 5 (compaction complements architectural change). P44 §Enforcement (manual-prompt gap analysis, session-scoped context design). Anthropic (2025a) on sub-agent architectures.

**Risk:** P44 is unbuilt and untested. The `dispatch_task` tool requires provider integrations, routing configuration, and a new MCP tool surface. The implementation effort is substantial (estimated 4–6 weeks for Phase 1). **Mitigation:** Phase P44 incrementally. Phase 1 delivers `dispatch_task` with minimum viable providers (Anthropic + DeepSeek). Phase 2 adds automated compaction triggers. Phase 3 evaluates and tunes with real feature data.

---

### Recommendation 3: Defer state-machine orchestration as a primary architecture (confidence: medium)

**What this means:** Do not pursue a deterministic state machine as the orchestrator's primary mode. The evidence for state-machine orchestration is conceptually appealing but empirically thin. The primary source (Network-AI) is a small project with no production-scale validation. The flexibility of chat-as-orchestrator — handling edge cases, recovering from sub-agent failures, making judgment calls on review findings — is valuable for a system that operates on diverse, real-world codebases.

However, the *hybrid* approach is promising: the dispatch loop enforces pipeline use (deterministic constraint), while the orchestrator decides what to dispatch and how to handle results (flexible execution). This is essentially what P44 enables: `dispatch_task` guarantees pipeline adherence; the orchestrator retains decision-making authority. The state machine is in the *tool*, not the *orchestrator*.

**Evidence basis:** Finding 7 (state-machine orchestration trades flexibility for predictability). Anthropic (2024) on workflows vs. agents. Research-orchestration-landscape-2025.md §7.5 (expose orchestration as MCP tools, not a framework).

**Risk:** The hybrid approach still trusts the orchestrator to make correct orchestration decisions. If the orchestrator consistently makes poor decisions (dispatching wrong tasks, misinterpreting review findings), a deterministic state machine would prevent those errors. **Mitigation:** Monitor orchestration decision quality as a metric. If decision quality degrades over long sessions (context rot affecting orchestration decisions themselves), revisit state-machine orchestration for the decision layer.

---

### Recommendation 4: Implement the designed compaction system as a complement to architectural change (confidence: high)

**What this means:** The U-shaped state-based compaction artefact (designed in P41 and incorporated into P44) should be built regardless of the architectural decision. Compaction addresses accumulated context (inevitable even in dispatch-loop architectures for very large features), while dispatch-loop addresses unnecessary context growth. Both are needed.

The compaction system is already fully designed: template, section ordering, trigger strategy, evaluation metrics. Implementation is blocked only on automated triggers (requires token counting from API metadata, available after P44 Phase 1). Procedural triggers (orchestrator notices and acts) can be implemented immediately.

**Evidence basis:** Finding 5 (compaction is complementary, not alternative). P41-research-context-compaction.md (full research, 7 findings, 5 recommendations). P44 §Compaction (artefact template, graduated triggers, KE-ID anchoring).

**Risk:** Low. The template is designed, the evidence supports it, and the implementation path is clear. The primary risk is that state-based compaction underperforms in practice (loses critical context that prose summaries would preserve). **Mitigation:** Evaluate with the three-metric framework (task completion rate, decision consistency, token efficiency). Review first 20 compaction events with human oversight.

---

### Recommendation 5: Establish context rot monitoring as an operational concern (confidence: medium)

**What this means:** Context rot should be treated like any other operational risk — monitored, measured, and alerted on. Specific monitoring signals:

1. **Goal drift detection.** Compare the orchestrator's session-start constraints (tier, feature type, explicit NEVER/ALWAYS rules) against its mid-session decisions. If the orchestrator violates a constraint it stated at session start, flag it. The P50 incident (stopping for confirmation in fast-track mode) is the canonical example.

2. **Context utilisation tracking.** Log `usage.input_tokens` at each task dispatch. Track growth rate (tokens per task). If growth is superlinear, investigate the source (redundant tool outputs? oversized sub-agent summaries? redundant knowledge entries?).

3. **Decision latency.** Track time and tool-call count between task dispatch and task completion. If this increases over a session, it may indicate the orchestrator is spending more effort on each decision — a symptom of context saturation.

**Evidence basis:** Finding 1 (context rot is a measurable behavioural degradation). P44 §Context threshold calibration (45%/60% thresholds were calibrated for 128K-200K windows, are stale for 1M). The New Stack / Elastic (2026) on operational metrics.

**Risk:** Monitoring is only valuable if it triggers action. If alerts fire but no one responds, the metrics are noise. **Mitigation:** Start with a single high-signal metric (goal drift detection) and refine before adding more. The P50 incident provides a clear calibration point: if the orchestrator stops for confirmation in fast-track mode, that's a high-confidence drift signal.

---

## Limitations

### What the research did not cover

1. **Direct A/B comparison of chat-as-orchestrator vs. dispatch-loop on the same workflow.** The evidence is triangulated from separate sources — Anthropic's sub-agent architecture recommendation, the P50 incident, the P43 validator pattern — but no controlled experiment compares the two approaches head-to-head on Kanbanzai's specific workflow. This is the single largest evidence gap.

2. **Quantitative measurement of context rot magnitude.** The P50 incident is a qualitative observation — "the orchestrator forgot the fast-track constraint." There is no quantitative measurement of how often goal drift occurs, at what context lengths, or with what severity. The literature (Liu et al. 2024) measures retrieval accuracy at different context positions, but no study measures behavioural consistency at different context lengths.

3. **Cross-model compaction transfer.** The U-shaped compaction artefact was designed for same-model handoff (Claude → Claude). Whether it works for cross-model handoff (Claude writes, DeepSeek reads) is unknown. The P44 design assumes same-model handoff in its initial phase.

4. **Dispatch-loop orchestrator context growth rate.** Without a running `dispatch_task` implementation, the rate at which the orchestrator's context grows in a dispatch-loop architecture is unknown. The design predicts linear growth with orchestration decisions, but actual growth depends on sub-agent summary size, error handling verbosity, and decision complexity.

### Assumptions underpinning the analysis

1. **The U-shaped attention curve transfers to agent behaviour.** Liu et al. (2024) demonstrated the curve for retrieval and QA tasks. The assumption that it applies equally to constraint adherence and goal maintenance in agent orchestration is plausible but untested.

2. **Pipeline-3.0 produces correct role/skill context when the role parameter is passed.** The P50 manual-prompt gap occurred because `handoff` without an explicit `role` parameter defaulted to the orchestrator role. P51 fixes this by making sub-agent roles the default when `sub_agents` are defined. This analysis assumes P51's fix works correctly — if the pipeline still produces incorrect role context after P51, `dispatch_task` would internalise a broken pipeline.

3. **Provider integrations (Anthropic + DeepSeek) are feasible at P44 scale.** The analysis assumes that adding provider API calls to the MCP server is a bounded engineering task (~200 lines of adapter code per provider). If provider API complexity exceeds this estimate, P44's implementation timeline extends.

### Conditions that could change the conclusions

1. **A published A/B comparison of single-context vs. dispatch-loop orchestrators** would either validate or refute the central architectural recommendation. If dispatch-loop architectures show no advantage for orchestration quality, the implementation investment in P44 is unjustified.

2. **Models that natively manage their own context** — for example, models that can internally compact their KV cache without external artefacts — would reduce the need for both compaction and dispatch-loop architectures. No such model exists in production, but it is an active research area (Memento, Microsoft Research 2026).

3. **A shift to very large context windows (10M+ tokens) with flat attention curves** would change the calculus. If models can attend equally well to all positions in a 10M-token context, both the U-shaped artefact design and the urgency of context management diminish. The current evidence (Anthropic 2025a) suggests this is unlikely — "context windows of all sizes will be subject to context pollution and information relevance concerns" — but it is not impossible.

### Knowledge gaps requiring further investigation

1. **What is the minimum viable context for orchestration resumption?** The compaction artefact template assumes 6 sections. How many of these are actually necessary? Could an orchestrator resume from just the Continuation Anchor + Active State table? Empirical testing is needed.

2. **Does the fast-track lightweight profile cover all fast-track scenarios?** The profile was designed post-P50 as a corrective. It hasn't been tested against a diverse set of fast-track features (varying sizes, dependencies, and codebase areas).

3. **What is the failure mode of `dispatch_task` when the pipeline produces incorrect context?** The current architecture has a manual fallback (orchestrator composes its own prompt). `dispatch_task` eliminates this fallback. If the pipeline produces wrong context, there is no recovery path — the sub-agent receives bad context and fails. The failure mode needs design attention.

---

## Source Index

### Primary — Peer-Reviewed

| # | Citation | Venue | Year | Key Contribution |
|---|----------|-------|------|------------------|
| 1 | Liu, N.F., Lin, K., Hewitt, J., Paranjape, A., Bevilacqua, M., Petroni, F., Liang, P. "Lost in the Middle: How Language Models Use Long Contexts." | TACL | 2024 | U-shaped attention curve; 30%+ accuracy drop for middle-of-context information; structural property across model families |

### Primary — Vendor Technical Reports

| # | Citation | Publisher | Date | Key Contribution |
|---|----------|-----------|------|------------------|
| 2 | Anthropic Engineering. "Effective Context Engineering for AI Agents" | Anthropic | Sep 2025 | Defines context rot and context engineering; recommends sub-agent architectures, compaction, structured note-taking; just-in-time context retrieval; progressive disclosure |
| 3 | Anthropic Engineering. "Building Effective Agents" | Anthropic | Dec 2024 | Recommends simple composable patterns over complex frameworks; orchestrator-workers pattern; workflow vs. agent distinction; simplicity and transparency principles |
| 4 | Microsoft Research. "Memento: Learning to Manage Context for Long-Form Reasoning" (via P41 compaction research summary) | Microsoft | Apr 2026 | Learnable compaction: 2-3× KV cache reduction with SFT; three-stage curriculum training; block-and-compress pattern |

### Secondary — Industry and Field Reports

| # | Citation | Publisher | Date | Key Contribution |
|---|----------|-----------|------|------------------|
| 5 | The New Stack (Todd R. Weiss, sponsored by Elastic). "How Context Rot Drags Down AI and LLM Results for Enterprises" | The New Stack | Mar 2026 | Practitioner evidence of agent looping, hallucination, reasoning degradation; Elastic data scientist and Franconia Research analyst perspectives; context rot as technical debt |
| 6 | Kargar, J.D. "Fundamentals of Context Management and Compaction," "One Line of Code, 41% Better Memory," "How LeRiM Manages Context" (via P41 compaction research) | Personal blog | 2026 | Agent-to-agent prompt optimisation; structured note-taking (`note()`, `prune()`); threshold-based context pressure signals; schema descriptions > instruction prompts |

### Internal — Kanbanzai Documents

| # | Document | Key Contribution |
|---|----------|------------------|
| 7 | P41-research-context-compaction.md | Full compaction research: 7 findings, 5 recommendations, 20+ sources; state-based vs. summary-based compaction; U-shaped artefact design; KE-ID anchoring |
| 8 | P41-research-context-compaction-summary.md | Implementation guide: artefact template, section ordering rationale, trigger strategy, evaluation metrics, implementation sequence |
| 9 | P44-design-model-routing-agent-launcher.md | Model routing dispatch architecture; `dispatch_task` design; compaction artefact template (production-ready); manual-prompt gap analysis; fast-track pipeline mismatch; session-scoped context |
| 10 | P43-design-fast-track-architecture.md | Validator architecture with fresh-session dispatch; validation bottleneck pattern (Google Research); automated gates replacing human confirmation |
| 11 | research-orchestration-landscape-2025.md | Orchestration framework survey: 5 major frameworks, 5 MCP servers; kagan comparison; capability matrix; validated patterns (ephemeral agents, atomic claiming, budget visibility) |
| 12 | P50 incident (via P44 §Enforcement) | Direct observation of goal drift: fast-track constraint forgotten; manual-prompt gap; per-claim context redundancy; byte_budget confusion |

### Referenced but Not Directly Accessed (404 or unavailable)

| # | Citation | Reason | Claims Represented Via |
|---|----------|--------|----------------------|
| 13 | Anthropic Engineering. "How We Built Our Multi-Agent Research System" (2025b) | URL returned 404 at time of research | Sub-agent performance improvement over single-agent; LLM-as-judge evaluation; 200K token threshold — via P41 compaction research |
| 14 | Chroma Research. "Context Rot" | URL returned 404 at time of research | Research-focused analysis of context degradation — via task brief starter sources |
| 15 | MorphLLM. "Context Engineering: Why More Tokens Makes Agents Worse" | URL returned 404 at time of research | Practical evidence of context-induced degradation — via task brief starter sources |
| 16 | Olamendy, J.C. "Context Engineering: The Invisible Discipline" (2025) | Not accessed | Survey of context management approaches — via task brief starter sources |
| 17 | Schmid, P. "Context Engineering Part 2" | Not accessed | Technical deep-dive on context engineering — via task brief starter sources |

---

## Retrieval Anchors

- **Research question:** Is context rot unavoidable in chat-as-orchestrator, or can it be managed?
- **Core finding:** Context rot is a manageable consequence of the chat-as-orchestrator architecture, not an inherent limitation of LLM-based orchestration. It can be mitigated in the near term (fast-track profiles, context budget recalibration, constraint pinning) and systematically addressed in the medium term (P44 dispatch-loop architecture). The P50 incident was a context engineering failure, not an architectural one.
- **Primary architectural recommendation:** Build P44 — model routing dispatch with `dispatch_task` internalising the pipeline, session-scoped context, and `fast_track` dispatch mode. This is the correct strategic architecture. Continue near-term mitigations in the current architecture while P44 is built.
- **Trade-off document:** The trade-off matrix above (chat-as-orchestrator + mitigations vs. model routing dispatch vs. state-machine orchestration)
- **Confidence calibration:** High confidence in the distinction between context rot (behavioural) and lost-in-the-middle (retrieval). High confidence that P44 is the correct strategic direction. Medium confidence in the specific mitigations (fast-track profile, constraint pinning) — they are designed but untested. Low confidence in state-machine orchestration as a primary architecture — insufficient production evidence.
