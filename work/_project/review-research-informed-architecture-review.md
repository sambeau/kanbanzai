# Research-Informed Architecture Review: Model Routing Agent & Fast-Track Architecture

**Date:** 2025-07-10  
**Scope:** P44 (Model Routing Agent feasibility design) and P43 (Fast-Track Architecture design)  
**Sources:** `research-agent-orchestration-research.md`, `research-orchestration-landscape-2025.md`  
**Audience:** Product, design, and development teams

## 1. Executive Summary

**Five findings the teams must act on:**

1. **Both designs are on solid structural footing.** The research validates rather than contradicts the core architectural choices: Option C for model routing (build together, extract later), ephemeral validators in fresh sessions, enforceable stage gates, and the U-shaped continuation prompt for compaction. No red flags requiring fundamental redesign.

2. **Fast-track's validators need an ACI-quality pass before implementation.** The validation checklists (S1–S10, D1–D12, R1–R8) are the right *substance*, but the research on ACI design (§2.1) warns that "bad tool descriptions can send agents down completely wrong paths." Several checks (S5 "testable assertion", S7 "disguised implementation instruction", D7 "monolithic tasks") require LLM classification — these are inherently ambiguous. The design should add concrete classification rubrics (not just labels) and test them with 15–20 sample documents before building the pipeline.

3. **Fast-track should go further on enforceability.** P43's validators produce pass/fail *verdicts*, but the design doesn't specify what happens — programmatically — when a validator fails. The research (§2.2) is unequivocal: "enforceable constraints beat advisory instructions." If a plan-validator fails D4 (every spec requirement covered by ≥1 task), the workflow state machine must *refuse* to advance the feature to `developing`. The design implies this but doesn't state it. Make it explicit.

4. **The plan-validator doesn't evaluate decomposition quality — and the research says this is the single strongest predictor of workflow success.** D7–D8 touch on granularity and parallelism but are non-blocking. Masters et al. found that "performance gains correlate almost linearly with the quality of the induced task graph" (§2.3) and Anthropic warned that "without detailed task descriptions, agents duplicate work, leave gaps, or fail to find necessary information." The plan-validator should add at least one blocking check on task description quality — each task must have a non-trivial description (not just a title), clear boundaries, and explicit inputs/outputs.

5. **Model routing's category system aligns with task structure research — but the "quick" category is a trap.** Google's research (§2.4) found that applying multi-agent coordination to sequential reasoning tasks degrades performance 39–70%. Kanbanzai's workflow stages map naturally: *specification* = sequential (degraded by parallelism), *implementation* = parallelisable (improved by orchestration). The `deep-reasoning` → architecture/spec and `implementation` → coding categories align with this. But `quick` (Haiku → doc updates) risks routing tasks that are actually reasoning-light but sequentially dependent into a lower-capability model that may not handle dependency tracking. The research doesn't say "use cheaper models for simple tasks" — it says "match the model's cognitive profile to the task's structural demands." Consider whether "quick" should be a *model preference* (same model family, just smaller) rather than a distinct category.

## 2. Model Routing Agent (P44): Research Alignment

### 2.1 Architecture Choice: Option C (Build Together, Extract Later)

**Research support: STRONG ALIGNMENT**

P44's Option C — a clean `internal/routing/` package boundary shipped within `kbz serve`, extractable to `kbz-route` later — is exactly the right choice, and the research supports it from three independent angles:

- **§7.5 "Expose Orchestration as MCP Tools, Not a Framework":** "The orchestration state machine should be exposed as MCP tools that an agent calls in a conversation loop — not as a separate daemon, not as a Python framework." Option B (separate `kbz-route` server) would create exactly this problem: a separate server that needs its own protocol for the orchestration loop. Starting embedded keeps model routing as an MCP tool (`dispatch_task`) within the same server the orchestrator already calls.

- **§6 "Should kanbanzai Build Its Own Orchestration?":** "An external orchestrator could call `context_assemble` — but then it is just a caller. kanbanzai is still doing all the work." The same logic applies to model routing: if `kbz-route` is a separate server, it's just a caller of Kanbanzai's context assembly. The value is in *integrating* model selection with the context that Kanbanzai already produces. That integration is tighter — and the handoff friction lower — within a single binary.

- **Anthropic (building effective agents):** "The most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns." Option C is the simple, composable choice. Start with one binary; extract when the evidence says extraction is warranted.

**The "middle ground" characterization is accurate but shouldn't imply indecision.** The design frames this as "middle ground" between embedded and separate. The research suggests embedded is the *correct* initial choice, not a compromise. The extraction path is a hedge against future evidence, not a sign that the initial choice is suboptimal.

**Recommendation:** Ship Option C with confidence. The research validates it. The `internal/routing/` package boundary should be the only architectural commitment at this stage — the code inside can evolve independently.

### 2.2 Category System Design

**Research support: PARTIAL ALIGNMENT — one concern**

The 5-category system (deep-reasoning, implementation, quick, review, audit) maps reasonably to Kanbanzai's role system and workflow stages. The research supports the general approach:

- **§2.4 "Architecture Must Match Task Structure":** Google's predictive model identifies optimal architecture using task properties. The categories implicitly encode task structure: `deep-reasoning` ≈ sequential, low tool density; `implementation` ≈ parallelisable, high tool density. This is the right axis to optimize on.

- **§2.5 "Proactive Orchestration vs Reactive Communication":** Masters et al. found that stronger models (GPT-5) exhibit qualitatively different action patterns — 14.5× more decomposition, 26× more dependency tracking. If Kanbanzai routes `deep-reasoning` tasks (spec writing, architecture) to stronger models and `implementation` tasks to mid-tier models, this aligns with the evidence: specification benefits from the "proactive orchestrator" cognitive profile; implementation can succeed with a more limited one.

- **Masters et al. on "Assign-All" baseline:** The finding that bulk upfront planning without reasoning achieves higher goal completion but lower constraint adherence maps to the tension between `quick` (Haiku, fast, less careful) and `deep-reasoning` (Opus, slower, more careful). For compliance-heavy tasks (specification, review), constraint adherence matters more than goal completion speed.

**Concern: The "quick" category is structurally underspecified.**

P44's `quick` category maps to "simple fixes, typos, documentation" with Haiku as the preferred model. But the research doesn't support "use a weaker model for simple tasks" as a general principle. Google found that applying the *wrong architecture* degrades performance — it says nothing about model capability vs. task simplicity. The risk is that "simple" tasks turn out to have hidden sequential dependencies or require dependency tracking that Haiku's weaker reasoning can't handle.

The competitive analysis (§6.5) acknowledges this implicitly: "Agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts." The `quick` category effectively makes Kanbanzai the judge of appropriate effort — but Kanbanzai doesn't yet have the signals to make that judgment reliably. A typo fix that touches 3 files with cascading implications isn't "quick" anymore.

**Recommendation:** Keep 5 categories but treat `quick` as a *model size preference within the same family* (Sonnet → Haiku; GPT-5.4 → GPT-5.4-mini) rather than a category that implies the task is simple. Add a constraint: `quick` should only be used when the orchestrator explicitly flags the task as "low complexity, ≤1 file, no structural changes." If the orchestrator can't make that determination, default to `implementation`.

### 2.3 Auto-Compaction and the U-Shaped Continuation Prompt

**Research support: STRONG ALIGNMENT — with one addition**

The U-shaped continuation prompt design directly implements validated patterns:

- **§7.4 "Budget Awareness Must Be Visible to the Receiving Agent":** "The receiving agent should know what was trimmed and why — so it can request missing context explicitly rather than proceeding with an incomplete picture and hallucinating the gaps." The U-shaped prompt captures exactly this: active state, active decisions, active constraints, surfaced knowledge. It tells the agent what's still relevant, not what happened historically.

- **Microsoft (context management):** "Monitor accumulated context size and use compaction techniques between agents to prevent exceeding model limits or degrading response quality." The 70% threshold trigger is a concrete implementation of "monitor accumulated context size." The U-shaped prompt is the "compaction technique."

- **Anthropic (multi-agent, subagent output to filesystem):** "Direct subagent outputs can bypass the main coordinator for certain types of results, improving both fidelity and performance." The U-shaped prompt is the analogue: lightweight references (KE-IDs, decisions, task states) instead of full conversation history.

**Addition: The research suggests an explicit "what was trimmed" field.**

§7.4 recommends: "Phase 4 should add a `trimmed` field to `context_assemble` responses listing what was cut (entry IDs, priorities, sizes) so executing agents can make informed decisions about what to pull additionally." The U-shaped prompt should include a `trimmed_context` section: "The following knowledge entries and design sections were active in the previous session but excluded from this compaction: KE-047 (confidence 0.45), KE-089 (confidence 0.32), §2.3 of design DOC-012. Query them with `knowledge(action: 'get', id: 'KE-047')` if they become relevant."

This directly addresses §7.4's finding: "It is not enough for the server to respect a byte budget during assembly. The receiving agent should know what was trimmed and why." Without this, the agent in the fresh session has no way to know what context it's missing — it can't request what it doesn't know existed.

**Recommendation:** The U-shaped prompt design is correct. Add a `trimmed_context` section as research §7.4 advises. The "compact-orchestration-session" skill should produce both the active state and the trimmed list.

### 2.4 Provider Fallback and Token Tracking

**Research support: ALIGNMENT — with a gap on agent self-regulation**

P44's token tracking design (per-request token counts, aggregated per-feature/per-batch) is necessary but not sufficient. The research identifies a gap:

- **§7.4 "Budget Awareness Must Be Visible to the Receiving Agent":** This principle applies not just to context assembly but to the agent's own consumption. "The receiving agent should know what was trimmed and why — so it can request missing context explicitly."

- **Network-AI (landscape review §3.2):** "Budget tracking and token management are explicit features." Budget tracking must be a first-class feature, not bolted on later — and the agent needs visibility into it.

- **Anthropic (effort scaling):** "Agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts." Token consumption is a form of effort. If agents don't know their own token budget, they can't self-regulate.

**The gap: P44 tracks tokens but doesn't feed them back to the agent for self-regulation.** The research suggests token budgets should be communicated to the agent at dispatch time: "You have 50K tokens for this task. You've used 12K so far. If you approach the limit, prioritize the remaining work or request continuation in a fresh session." This transforms token tracking from a monitoring feature into a self-regulation mechanism — the agent makes its own tradeoffs about what to invest tokens in.

**Recommendation:** Add agent-facing token budget communication to the `dispatch_task` tool. The agent should receive its budget and current consumption as part of the task context. This implements the "effort scaling" finding from Anthropic and the "budget awareness" principle from §7.4.

### 2.5 Relationship to the Orchestrator-Worker Pattern

**Research support: ALIGNMENT — the design accounts for this distinction implicitly but should make it explicit**

P44's category system implicitly distinguishes between sequential reasoning tasks and parallelisable implementation tasks through the `deep-reasoning` vs. `implementation` categories. But the design doesn't explicitly connect categories to orchestration patterns.

- **§2.4 "Architecture Must Match Task Structure":** Google found that "on tasks requiring strict sequential reasoning (planning), every multi-agent variant degraded performance by 39–70%." `deep-reasoning` tasks (architecture, spec-writing) are sequential reasoning — they should NEVER be parallelised. `implementation` tasks are parallelisable — they benefit from orchestrator-workers (+81%).

- **§4.6 "Match Orchestration Pattern to Task Type":** The research recommends different orchestration patterns for different workflow stages. Specification = single agent, no parallelism. Implementation = orchestrator-workers. Review = maker-checker or panel.

P44's model routing would dispatch both types. The design should explicitly state: the `deep-reasoning` category should spawn a single agent (no workers), while `implementation` can spawn workers. The category system is the mechanism for matching the orchestration pattern to the task type — but this connection isn't drawn in the design.

**Recommendation:** Add a mapping from category to orchestration pattern: `deep-reasoning` → single-agent, sequential; `implementation` → orchestrator-workers; `review` → maker-checker; `audit` → single-agent, low-temperature. This makes the research finding actionable through the category system.

## 3. Fast-Track Architecture (P43): Research Alignment

### 3.1 Validator Role Design and the ACI Principle

**Research support: PARTIAL ALIGNMENT — the checklists need ACI design work before implementation**

P43's validator roles are well-designed at the identity level (anti-patterns, cognitive profile, what they do and don't evaluate). The checklists (S1–S10, D1–D12, R1–R8) are complete and well-scoped. The research validates the general approach:

- **§2.2 "Enforceable Constraints Beat Advisory Instructions":** Validators as automated enforcement of structural constraints is exactly what the research prescribes.
- **MetaGPT (verification gates):** "Agents must produce structured artifacts that pass verification before advancing." P43's validators are these verification gates.
- **§7.1 "Ephemeral Agents + Shared Knowledge Store":** P43's session management (validators always run in fresh sessions via `spawn_agent`) directly implements this validated pattern.

**But the checklists haven't been designed as ACIs yet.** The research's strongest finding (§2.1) is that "interface design affects agent performance as much as model capability." Several checks are inherently ambiguous for an agent:

- **S5: "Every acceptance criterion is testable" — LLM classification.** What makes a criterion "testable"? The validator needs a classification rubric, not just a label. Example rubric: "A testable criterion names a specific observable condition that can be verified without implementation knowledge. Counter-examples: 'the system should be fast' (no threshold), 'the system should work correctly' (no observable condition)."

- **S7: "No requirement is a disguised implementation instruction" — LLM classification.** This is subtle even for humans. The rubric needs examples: "'Use Redis for caching' is an implementation instruction. 'Cached reads must return in <10ms' is a requirement. However, 'Use a cache for frequent reads' is ambiguous — it names a solution pattern but doesn't mandate a specific technology. Classify as borderline → escalate."

- **D7: "No monolithic tasks (>3 files or >1 AC)" — structural but judgment call.** Three files that are all in the same package and highly cohesive is not monolithic. Three files across different services is. The check needs a cohesion heuristic, not just a count.

**The research predicts this problem.** Anthropic (multi-agent): "Bad tool descriptions can send agents down completely wrong paths." For validators, the "tool descriptions" are the check rubrics. Without concrete, example-driven rubrics, validators will make inconsistent judgments — the same document might pass or fail depending on context window contents, model temperature, or phrasing.

**Recommendation:** Before implementing the validation pipeline, build a "validator rubric pack" — for each LLM-classification check, a rubric with:
1. A clear definition of what "pass" means with 2–3 positive examples
2. A clear definition of what "fail" means with 2–3 negative examples
3. A "borderline → escalate" pattern for ambiguous cases
4. Test the rubrics on 15–20 real Kanbanzai specifications and plans (following Anthropic's evaluation approach: "a set of about 20 queries representing real usage patterns" was sufficient to spot dramatic impacts)

### 3.2 Enforceable Constraints vs. Advisory Instructions

**Research support: PARTIAL ALIGNMENT — design implies enforcement but doesn't specify the mechanism**

P43's validators produce pass/fail/pass_with_notes verdicts. The design says "fail → escalate to human" and "pass → auto-approve." But it doesn't specify what the workflow state machine does when a validator fails.

- **§2.2 "Enforceable Constraints Beat Advisory Instructions":** This is the research's most emphatic finding. Every source comparing the two finds enforcement wins. MetaGPT: verification gates that *must* pass before advancing. Microsoft: programmatic gates that *block* progress. Masters et al.: hard constraints (ℋ) that *terminate* the workflow on violation.

- **§4.1 "Enforce Lifecycle Gates at the Tool Level":** The research explicitly recommends: "`handoff` / `next` tools should refuse to assemble implementation context for features not in the correct lifecycle state." This is the enforcement mechanism.

**The gap:** P43 defines what validators *produce* (verdicts) but not what the system *does* with those verdicts. If a plan-validator fails D4 (spec requirement not covered by task), does the feature stay in `dev-planning`? Can the orchestrator override? Can a human force the transition?

The answer is implied: yes, the feature stays in `dev-planning`; yes, override exists as an escape hatch. But making this explicit — and implementing it as a programmatic gate, not an advisory recommendation — is the difference between a system that *prevents* bad plans from advancing and one that *suggests* they shouldn't.

**Masters et al.'s hard vs. soft constraint distinction maps cleanly to P43:**

| Constraint Type | P43 Analogue | Mechanism |
|---|---|---|
| Hard (ℋ) — violation terminates | Blocking checks (S1–S5, S10; D1–D6, D9; R1–R2, R4–R5, R7) | Transition refused by state machine |
| Soft (𝒮) — violation with penalties | Non-blocking checks | Flagged in document, doesn't block transition |

**Recommendation:** Make enforcement explicit: add a `transition_validator` hook to the stage binding that, before advancing a feature from `specifying → dev-planning` or `dev-planning → developing`, checks: (a) was the validator run? (b) did it pass all blocking checks? If no, refuse the transition. This converts P43 from advisory to enforceable — the single highest-impact change the research supports.

### 3.3 Decomposition Quality as the Critical Path

**Research support: MISALIGNMENT — this is the most significant gap in the current design**

Multiple independent sources converge on decomposition quality as the critical path (§2.3). Yet P43's plan-validator treats decomposition quality as non-blocking:

- **D7 (monolithic tasks):** Non-blocking. "No task touches >3 files or >1 acceptance criterion."
- **D8 (parallelisable marking):** Non-blocking. "Tasks with no shared files or dependencies are marked parallelisable."
- **No check on task description quality.** No check that tasks have clear boundaries, explicit inputs/outputs, or sufficient detail for an agent to execute without ambiguity.

The research says this matters *enormously*:

- **Masters et al.:** "Performance gains correlate almost linearly with the quality of the induced task graph — underlining that structure learning, not raw language generation, is the critical path."
- **Anthropic (multi-agent):** "Without detailed task descriptions, agents duplicate work, leave gaps, or fail to find necessary information."
- **§7.2 "Task Decomposition Must Gate Dispatch":** "Never dispatch an agent to a task that is not already granular, unambiguous, and properly specified. The orchestration loop must refuse to dispatch a task that is not in `ready` status — meaning it has a summary, acceptance criteria, and resolved dependencies."

P43's plan-validator checks structural completeness (all sections present, cross-references valid, dependency graph acyclic) but doesn't evaluate whether the decomposition is *good*. A plan can pass all D1–D12 checks and still have tasks that are too vague for agents to execute independently — e.g., "Implement the authentication layer" with no further detail.

**The design's rationale is that "architectural judgment" belongs to the human.** But the research distinguishes between *architectural* judgment (what to build, how to structure the system) and *decomposition* judgment (how to break work into executable tasks). Decomposition is a mechanical derivative of the specification — if the spec defines 5 requirements, the plan should decompose them into tasks that each address 1–2 requirements with clear boundaries. This is automatable.

**Recommendation:** Add at least one blocking check on decomposition quality:

- **D13 (BLOCKING): Task descriptions are actionable.** Every task has a description of ≥50 words that includes: what the task produces, what inputs it requires (files, knowledge, dependencies), and what "done" means beyond the acceptance criterion. This directly implements Anthropic's finding: "Each subagent needs an objective, an output format, guidance on the tools and sources to use, and clear task boundaries."

This doesn't require architectural judgment — it's a structural completeness check on the task descriptions themselves.

### 3.4 Risk Tiers and the Sequential Penalty

**Research support: ALIGNMENT — with one concern about `retro_fix`**

The risk tier design is well-structured and maps to the research's findings on matching architecture to task structure:

- **`feature` tier (design human, rest auto):** Aligns with the design gate as the anchor for architectural judgment (§11.8 rationale). Design is sequential reasoning (§2.4) — it benefits from human judgment, not automation.

- **`critical` tier (all human):** Implements the "validation bottleneck" pattern (§2.2) — human validation at every gate prevents error amplification. Google found centralized orchestration contained error amplification to 4.4× vs. 17.2× for independent agents. All-human gates is the strongest form of this.

- **`bug_fix` tier (spec human, rest auto):** Reasonable middle ground. The spec gate remains human because the fix approach may have implications beyond the immediate bug.

**Concern: `retro_fix` skips all gates, including review.**

The `retro_fix` tier's rationale (§11.6) is sound: "Design is derived from evidence, not intent." But skipping the review gate means implementation goes to merge with only validator approval — the review-gate-validator audits the review *process*, not the code. If there's no review panel, there's nothing to audit.

The competitive analysis (§11.6) shows the flow: "Specialist review panel → review-gate-validator runs → pass → merge." But the tier definition shows `review: false` with `validators: [spec-validator, plan-validator, review-gate-validator]`. The review-gate-validator auditing a non-existent review panel would either rubber-stamp (finding R1 would fail — no reviewer outputs) or fail spuriously.

This appears to be a design inconsistency, not a research misalignment. Either `retro_fix` should run a review panel (which the validator then audits) or should skip the review-gate-validator.

**Recommendation:** Clarify the `retro_fix` flow. If implementation is involved (not just config/docs changes), a review panel should run. The validator audits the review. If the change is documentation-only, skip review entirely but add a check: "No implementation files changed."

### 3.5 Session Management and Context Degradation

**Research support: STRONG ALIGNMENT — with one addition from the research**

P43's session management design is one of the strongest parts of the architecture:

- **Fresh sessions via `spawn_agent`:** Directly implements §7.1 "Ephemeral Agents + Shared Knowledge Store."
- **Clean context (document only, not the conversation that produced it):** Implements the insight from the research that "a validator running in the author's session sees the reasoning that produced the document and is biased by it."
- **Output reduction (verdict + N findings + evidence score):** Prevents orchestrator context bloat.
- **Cycle tracking (`max_auto_cycles`):** Prevents infinite fix-validate loops.

**Addition: Subagent output to filesystem pattern.**

Anthropic's multi-agent team used a pattern P43 could adopt: "Direct subagent outputs can bypass the main coordinator for certain types of results, improving both fidelity and performance." For validators, the full validation report (detailed per-check analysis, evidence citations, uncertain findings) should be written to the document store. The orchestrator receives only the summary verdict. This is already partially in P43 ("Full output offloaded to document record") but could be strengthened:

- The validator writes a structured validation report to `work/{feature}/reports/{validator}-{timestamp}.md`
- The report is registered as a `report` document type via `doc(action: 'register')`
- The orchestrator receives: verdict, number of blocking/non-blocking findings, evidence score, and a reference to the full report

This prevents the orchestrator from ever needing to hold full validator output in its context while making it available for human audit.

**Recommendation:** Formalize the validator output pattern: full report → document store; summary → orchestrator. This is already partially in the design but should be structured as a standard pattern for all validators.

## 4. Cross-Cutting Findings

### 4.1 Model Selection for Validators vs. Authors

**Research support: The research SUPPORTS model differentiation between validators and authors — but not necessarily now**

P43's open question #5 asks whether validators should use a different model. The research provides a nuanced answer:

- **Masters et al. (proactive orchestrator vs. reactive communicator):** The cognitive profiles of authors (creative synthesis, decomposition, pattern matching) and validators (compliance audit, checklist execution, binary classification) are qualitatively different. Validators don't need the "proactive orchestrator" capabilities — they need thoroughness, consistency, and resistance to sycophancy.

- **Anthropic (LLM-as-judge):** "A single LLM call with a single prompt outputting scores from 0.0–1.0 and a pass-fail grade was the most consistent and aligned with human judgements." This suggests validators perform well as simple, rubric-driven evaluators — which doesn't require the strongest model.

- **Google (tool-use bottleneck):** Validators use few tools (read document, read parent document, check against rubric). Low tool density tasks don't benefit from the coordination overhead of stronger models.

**The research supports P43's current approach (same model, near-zero temperature, different role prompt) as a reasonable starting point.** But it also suggests that when model routing is available, validators should be a distinct category (`audit`) with a model optimized for consistency and low hallucination rather than creative reasoning. The key question is whether validators need *stronger* reasoning or *more consistent* reasoning. The research suggests the latter — GPT-5.4 at near-zero temperature is a better fit for audit tasks than Claude Opus with extended thinking, because the audit cognitive profile values consistency over creative depth.

**Recommendation:** P43's "same model, different temperature" is correct for initial implementation. When P44's model routing is built, the `audit` category should use a model optimized for consistency (GPT-5.4 near-zero or Claude with temperature=0), not the strongest reasoning model. The research validates that audit and authoring are different enough cognitive profiles to warrant model differentiation.

### 4.2 Interaction Between Fast-Track and Model Routing

**Research support: The designs should anticipate each other without coupling**

P41's dependency structure intentionally separates P43 (fast-track) and P44 (model routing) — P43 builds first, P44 feasibility design proceeds in parallel but implementation waits until P43 is stable. The research supports this sequencing:

- **§6 "Should kanbanzai Build Its Own Orchestration?":** "The heavy lifting — context assembly, knowledge lifecycle, document intelligence, git worktrees, health checks, semantic merge gates — is done. Phase 4a adds approximately five tools and a few entity fields to complete the orchestration loop." Fast-track is Phase 4a work — it uses existing capabilities. Model routing is Phase 4b+ — it requires new capabilities (provider integration, agent runtime).

- **Anthropic (building effective agents):** "Use the lowest level of complexity that reliably meets your requirements." P43 works at the current complexity level. P44 adds complexity. Build P43 first to validate the automated gate pattern before adding model routing complexity.

**However, the validator `spawn_agent` calls in P43 should go through an abstraction, not directly to the model client.** Currently, `spawn_agent` delegates to the MCP client's agent dispatch. When model routing is built, validators should route through the dispatch loop instead — but if P43 hardcodes the current `spawn_agent` mechanism, retrofitting will be harder.

**Recommendation:** Add a thin abstraction layer for agent dispatch in P43. The validator pipeline calls `dispatch_validator(role, skill, context)` which internally uses `spawn_agent`. When P44 is built, `dispatch_validator` can route through the model routing dispatch loop without changing validator code. This is an implementation detail, not a design change — but it should be noted in the P43 design as forward-compatibility.

### 4.3 The Validation Bottleneck Pattern

**Research support: P43 fully leverages this — it IS the validation bottleneck**

Google found that centralized orchestration contained error amplification to 4.4× (vs. 17.2× for independent agents) because "the orchestrator acts as a validation bottleneck" (§2.2). P43's validators at each stage gate are exactly this pattern — they catch errors at spec, plan, and review stages before they propagate to implementation (where errors are 4.4× more expensive to fix).

The design already leverages this well:
- Spec validator catches requirement gaps before anyone writes code
- Plan validator catches decomposition issues before anyone dispatches tasks
- Review gate validator catches review quality issues before merge

**No additional validation bottleneck points are needed — but the design should highlight that it IS implementing this pattern.** The research finding is strong enough that P43 should cite it explicitly as architectural validation.

**Recommendation:** Add a note to the P43 design citing Google's "validation bottleneck" finding (§2.2) as explicit architectural validation for why validators exist at every stage gate, not just at review.

### 4.4 What the Research Says NOT to Build

**Research support: Both designs avoid the major anti-patterns — with one caution**

The research explicitly warns against four things. Here's how the designs fare:

| Anti-Pattern | Research Source | P43 Assessment | P44 Assessment |
|---|---|---|---|
| Over-engineering: complex frameworks when simple patterns work | Anthropic (building effective agents) | ✅ Clean. Uses existing `spawn_agent`, `doc_intel`. No new framework. | ⚠️ Caution. Provider integration, fallback chains, token tracking, thinking control, compaction — the scope is large. Build the MVP (Anthropic + OpenAI, 3 categories, token tracking → report only) before adding DeepSeek, auto-compaction, and full category system. |
| Wrong architecture for task: multi-agent on sequential reasoning | Google (sequential penalty) | ✅ Correct. Validators are single-agent, not panels. | ✅ Correct. Categories distinguish sequential vs. parallelisable tasks. |
| Unnecessary coordination complexity | Microsoft (anti-pattern) | ✅ Clean. Validators don't coordinate with each other — sequential pipeline. | ⚠️ Caution. Model routing adds coordination between Kanbanzai and provider APIs. The fallback chain logic should be simple (try A, if fail try B, if fail try C) — not a complex health-check/heuristic system. |
| Sharing mutable state between concurrent agents | Microsoft (anti-pattern) | ✅ Protected. Validators in fresh sessions, no shared state. | ✅ Protected. Each dispatch is an independent API call. |

**The caution for P44:** The scope is large, and the research repeatedly warns against building too much at once. The feasibility design lists 5 features unlocked by model routing (thinking-level control, auto-compaction, true Ralph Loop, provider fallback, cost tracking) plus a 5-category system and 3-provider integration. The research says: "Start with the right level of complexity. Use the lowest level of complexity that reliably meets your requirements" (Microsoft). The MVP should be: 1 provider (Anthropic), 2 categories (deep-reasoning, implementation), token tracking → report only. Validate that before building the full system.

## 5. Risk Register

Risks from the research that the current designs don't fully mitigate:

| # | Risk | Severity | Reference | Mitigation |
|---|---|---|---|---|
| R1 | **Validator inconsistency on LLM-classification checks.** S5, S7, D7, D10 rely on LLM judgment without concrete rubrics. Different runs may produce different verdicts. | High | §2.1 (ACI), Anthropic (bad tool descriptions) | Build validator rubrics with positive/negative examples. Test on 15–20 real documents before pipeline implementation. |
| R2 | **Plan-validator doesn't evaluate decomposition quality.** A structurally complete plan with vague task descriptions passes validation but produces poor implementation outcomes. | High | §2.3 (decomposition quality), Masters et al., Anthropic (detailed task descriptions) | Add D13 (blocking): task descriptions must be actionable (≥50 words, clear inputs/outputs, explicit boundaries). |
| R3 | **`retro_fix` tier skips review with no code audit.** If implementation changes occur, skipping review entirely means no specialist scrutiny before merge. | Medium | MetaGPT (verification gates), Masters et al. (hard constraints) | Clarify `retro_fix` flow: if implementation files change, run review panel + review-gate-validator. If docs only, skip review entirely with explicit check. |
| R4 | **Model routing scope creep.** 5 features, 5 categories, 3 providers, fallback chains, token tracking, auto-compaction, Ralph Loop — the MVP could become the full system before validation. | Medium | Anthropic (simple, composable patterns), Microsoft (lowest complexity) | Define MVP explicitly: 1 provider, 2 categories, token tracking → report. Build auto-compaction and Ralph Loop as separate phases. |
| R5 | **Fast-track enforcement gap.** Validators produce verdicts but the design doesn't specify programmatic enforcement of those verdicts at the state machine level. An orchestrator under pressure could ignore a validator failure. | Medium | §2.2 (enforceable constraints), §4.1 (tool-level enforcement) | Add `transition_validator` hooks to stage bindings. Feature cannot advance if blocking checks failed. |
| R6 | **`quick` category dispatches reasoning-dependent tasks to weaker models.** Simple-looking tasks may have hidden sequential dependencies or dependency-tracking requirements that Haiku can't handle. | Low | §2.4 (architecture must match task structure), Masters et al. (proactive vs. reactive) | Constrain `quick` to tasks the orchestrator explicitly flags as "low complexity, ≤1 file, no structural changes." Default to `implementation` if uncertain. |
| R7 | **U-shaped prompt lacks trimmed-context visibility.** The agent in the fresh session can't request context it doesn't know was removed. | Low | §7.4 (budget awareness) | Add `trimmed_context` section to compaction artifact listing trimmed entries with IDs and reasons. |

## 6. Recommendations

Prioritized, with research citations:

### High Priority (do before implementation)

1. **Add programmatic enforcement of validator verdicts to stage transitions.** When a blocking check fails, the feature CANNOT advance. This converts fast-track from advisory to enforceable — the single highest-impact change the research supports. *Reference: §2.2 "Enforceable Constraints Beat Advisory Instructions," §4.1 "Enforce Lifecycle Gates at the Tool Level," MetaGPT SOPs with verification gates, Masters et al. hard constraints (ℋ).*

2. **Add a blocking check on task description quality (D13).** Each task must have an actionable description: what it produces, inputs required, what "done" means. *Reference: §2.3 "Decomposition Quality Is the Critical Path," Masters et al. (performance correlates linearly with task graph quality), Anthropic (without detailed descriptions, agents duplicate work).*

3. **Build validator rubrics for all LLM-classification checks.** Each check (S5, S7, D7, D10, R2, R3) needs a rubric with pass/fail examples and a "borderline → escalate" pattern. Test on 15–20 real documents. *Reference: §2.1 "Agent-Computer Interface Design Is a First-Class Concern," Anthropic (bad tool descriptions send agents down wrong paths; 40% time reduction from iterating tool descriptions).*

### Medium Priority (do during implementation)

4. **Add a `trimmed_context` section to the U-shaped compaction prompt.** List what was removed with entry IDs and reasons so the fresh-session agent can request it. *Reference: §7.4 "Budget Awareness Must Be Visible to the Receiving Agent."*

5. **Add agent-facing token budget communication to `dispatch_task`.** The agent should know its budget and current consumption for self-regulation. *Reference: §7.4, Anthropic (effort scaling rules), Network-AI (budget tracking as first-class feature).*

6. **Map categories to orchestration patterns explicitly.** `deep-reasoning` → single-agent; `implementation` → orchestrator-workers. *Reference: §2.4 "Architecture Must Match Task Structure," §4.6 "Match Orchestration Pattern to Task Type," Google (sequential penalty, alignment principle).*

7. **Formalize validator output pattern.** Full report → document store; summary → orchestrator. *Reference: Anthropic (subagent output to filesystem), §7.1 "Ephemeral Agents + Shared Knowledge Store."*

8. **Add forward-compatible dispatch abstraction in P43.** Validator pipeline calls `dispatch_validator()` which uses `spawn_agent` today and model routing tomorrow. *Reference: P41 dependency structure, §6 "Should kanbanzai Build Its Own Orchestration?"*

### Lower Priority (post-MVP refinement)

9. **Clarify `retro_fix` review flow.** If implementation changes, run review panel. If docs only, skip with explicit check. *Reference: MetaGPT (verification gates at every stage).*

10. **Define model routing MVP explicitly.** 1 provider, 2 categories, token tracking → report. Build auto-compaction and Ralph Loop as separate phases. *Reference: Anthropic (simple, composable patterns), Microsoft (lowest complexity level).*

11. **Constrain `quick` category.** Only dispatch to `quick` when the orchestrator explicitly determines the task is low-complexity. *Reference: §2.4, Masters et al. (Assign-All achieves higher goal completion but lower constraint adherence).*

12. **Cite Google's "validation bottleneck" finding in P43 design.** The validator-at-every-gate pattern is explicitly validated by research. *Reference: §2.2, Google (centralized orchestration error amplification 4.4× vs. 17.2×).*
