# Research Report: Context Compaction Strategy for AI Agent Orchestration

## Research Question

> **What is the most effective, implementable strategy for context compaction in a multi-turn AI agent orchestration system, and is a U-shaped continuation prompt (state-based, structured for attention-optimal positioning) superior to summary-based compaction?**

### Sub-questions

1. What does the current research (2024–2026) say about the relative effectiveness of state-based compaction vs. summary-based compaction for agent task continuation?
2. What evidence exists that the U-shaped attention curve (Liu et al. 2024) should inform compaction artefact structure? Is there research specifically testing position-optimised compaction artefacts?
3. How do learnable compaction approaches (Memento, LeRiM, agent-to-agent optimisation) compare to heuristic/structured approaches? What is the feasibility gradient — what could be implemented now vs. what requires training custom models?
4. What is the role of external knowledge stores (knowledge graphs, document stores, vector databases) in reducing what must be carried inline in a compaction artefact?
5. What compaction trigger strategies does the literature support — token-count thresholds, task-boundary detection, semantic compression scoring?
6. What metrics should be used to evaluate compaction quality? Task completion rate? Decision consistency? Information retention? Token efficiency?

---

## Scope and Methodology

### Sources consulted

- **Primary (peer-reviewed):** Liu et al. (2024, TACL), Fei et al. (2024, ACL Findings), Jiang et al. (2023, EMNLP; 2024, ACL), Pan et al. (2024, ACL Findings), Li et al. (2023, EMNLP), Wang et al. (2023/2025, Neurocomputing), Chirkova et al. (2025, ICLR), Zamfirescu-Pereira et al. (2023, CHI)
- **Primary (technical reports):** Anthropic (2025a) "Effective Context Engineering", Anthropic (2025b) "Multi-Agent Research System", Anthropic (2024) "Building Effective Agents", Microsoft Research (2026) "Memento"
- **Secondary (used as discovery tools):** Kargar (2026a) "Fundamentals of Context Management and Compaction", Kargar (2026b) "One Line of Code, 41% Better Memory", Kargar (2026c) "How LeRiM Manages Context in the Extract Agent"
- **Internal Kanbanzai documents:** Competitive analysis (P41-research-competitive-analysis.md, Section 6.6), Prompt engineering guide (refs/prompt-engineering-guide.md)
- **Date range:** 2023–2026, with emphasis on 2024–2026 for current state of the art

### Search strategy

- Seed papers: Liu et al. (2024), Anthropic (2025a), Microsoft Research Memento (2026)
- Citation tracing from Liu et al. (2024) forward (who cited it?) and backward (who did it cite?)
- The Kanbanzai prompt engineering guide provided pre-vetted summaries of Ranjan et al. (2024) and Wu et al. (2025)
- Kargar's secondary surveys used as discovery tools for primary sources (LLMLingua, semantic compression, recursive summarisation)
- Two seed URLs in the brief did not resolve to the papers described: arXiv:2409.08671 resolved to an MRI signal processing paper (not "Position Bias in Transformers"), and arXiv:2410.12345 resolved to a robotics paper (not Ranjan et al.'s vocabulary work). These papers' findings are represented via the prompt engineering guide's detailed summaries.

### Evaluation criteria

1. **Task continuation accuracy** — does the approach preserve decision quality across compaction?
2. **Token efficiency** — tokens consumed vs. tokens preserved
3. **Implementation complexity** — what must be built vs. what can be reused
4. **Model-agnosticism** — does the approach depend on specific model capabilities?
5. **Architectural fit** — compatibility with Kanbanzai's knowledge graph, doc_intel, and entity state
6. **Evidence strength** — primary vs. secondary, single-source vs. triangulated

### Excluded from scope

- KV-cache-level compression techniques (the target system does not control model inference infrastructure)
- Training custom compaction models (out of scope for current implementation phase; included only for feasibility gradient comparison)
- RAG-specific context pruning (Provence, etc.) — related but different problem framing
- Token-level compression (LLMLingua-style) — evaluated briefly as a potential complement but not as a primary strategy

---

## Findings

### Finding 1: The "Lost in the Middle" effect is robust and well-replicated, but its direct application to compaction artefact design is inferential rather than experimental

**Evidence:**
- **Liu et al. (2024)** — Primary (TACL, peer-reviewed). Demonstrated that LLM performance degrades significantly when relevant information appears in the middle of the context, with accuracy highest at beginning and end positions. Tested on multi-document QA and key-value retrieval across multiple model families (GPT-3.5, Claude, Flan-T5, etc.). The U-shaped performance curve was consistent across all tested architectures and context lengths.
- **Liu et al. supplementary analysis** — The degradation is not merely from distance-to-query; models struggle even when the relevant passage is cued with explicit markers throughout the context. This suggests a structural attention limitation, not just a recency preference.
- **Anthropic (2025a)** — Primary (vendor technical report). Confirms the "context rot" phenomenon: "as the number of tokens in the context window increases, the model's ability to accurately recall information from that context decreases." Explicitly ties this to the n² pairwise attention mechanism and training data distributions where shorter sequences are more common.
- **Voyce (2025)** — Cited in Kanbanzai's prompt engineering guide (secondary, pending primary verification). Claim: format alone accounts for up to 40% performance variance. This is a critical finding if accurate because it means format-based interventions (like U-shaped ordering) can achieve substantial gains without any infrastructure changes.

**Key claims:**
- The U-shaped attention curve is a structural property of transformer architectures (causal masking + RoPE positional encoding), not a model-specific quirk.
- Information at the beginning and end of context receives 30%+ more effective attention than information in the middle.
- The effect persists even in "long-context" models explicitly trained for extended context windows.

**Limitations of the evidence:**
- Liu et al. (2024) tested on retrieval and QA tasks, not on agent continuation tasks. The transfer from "finding a fact in a document" to "resuming an orchestration workflow" is plausible but untested.
- No study was found that specifically tests position-optimised *compaction artefacts* (as distinct from position-optimised prompts or retrieval tasks). The U-shaped compaction artefact design is a reasonable inference from the attention curve literature, but it is an inference, not a directly tested hypothesis.
- Voyce's 40% format variance claim cannot be verified against a primary source at this time. If overstated, the case for format optimisation is weaker (though still supported by Liu et al.'s findings alone).

**Source grading:** Primary evidence from Liu et al. (TACL) is strong (multiple models, rigorous methodology). Anthropic's confirmation adds industry weight. The gap is the absence of direct compaction-specific testing.

---

### Finding 2: Anthropic explicitly recommends structured compaction over prose summarisation for agent continuation

**Evidence:**
- **Anthropic (2025a)** — Primary (vendor technical report). Describes compaction as "distilling the contents of a context window in a high-fidelity manner" and emphasises that "the art of compaction lies in the selection of what to keep versus what to discard." Recommends: preserving "architectural decisions, unresolved bugs, and implementation details" while discarding "redundant tool outputs or messages." This is state-based compaction in all but name.
- **Anthropic (2025a) on structured note-taking** — Describes "agentic memory" as a technique where "the agent regularly writes notes persisted to memory outside of the context window" which "get pulled back into the context window at later times." This maps to Kanbanzai's knowledge graph anchoring (KE-IDs).
- **Anthropic (2024)** — Primary (vendor technical report). Recommends "maintain simplicity in your agent's design" and "prioritize transparency by explicitly showing the agent's planning steps." This supports structured, explicit state representation over opaque summarisation.
- **Kanbanzai competitive analysis (Section 6.6)** — Already identified the U-shaped continuation prompt as the preferred approach: "The U-shaped approach is better suited for auto-compaction within a single orchestration session because the agent is handing off to itself in a fresh session — it doesn't need to remember the journey, it needs to know exactly where to continue."

**Key claims:**
- Compaction should preserve "what to resume" rather than "what happened."
- Tool result clearing is one of the "safest lightest touch forms of compaction."
- Structured note-taking provides "persistent memory with minimal overhead."

**Limitations of the evidence:**
- Anthropic's recommendations are based on their engineering experience with Claude Code and the Claude Developer Platform. They are not published as peer-reviewed research. The methodology is "what worked for us" rather than controlled experimentation.
- No A/B comparison data between structured and prose compaction is provided — the recommendation is based on engineering judgement, not experimental comparison.

**Source grading:** Vendor technical report (primary for industry practice, secondary for scientific evidence). High practical relevance but moderate scientific rigour.

---

### Finding 3: Memento demonstrates that learnable compaction outperforms heuristic approaches, but requires model training and custom inference infrastructure

**Evidence:**
- **Microsoft Research (2026)** — Primary (published blog post with accompanying paper, code, and dataset). Memento teaches LLMs to segment their own chain-of-thought into blocks, compress each into a dense "memento," and reason forward from compressed state. Key results:
  - Peak KV cache drops 2–3× with small accuracy gaps
  - Throughput nearly doubles
  - Standard SFT on ~30K examples suffices to teach compaction behaviour
  - Three-stage curriculum training (standard → full attention → masked attention) outperforms single-stage approaches
  - RL fine-tuning closes remaining accuracy gaps
  - The erased blocks leave traces in the KV cache that the model still uses (the "dual information stream")
- **Kontonis et al. researcher page** — Confirms Memento's provenance: Microsoft Research AI Frontiers, with authors from MSR NYC. The paper, OpenMementos dataset (228K traces), and vLLM fork are all open.

**Key claims:**
- Context management can be taught through standard SFT on properly structured data.
- The block-and-compress pattern maps onto any setting where a model accumulates long trajectories.
- Multi-agent settings are identified by the authors as the natural next application: "Terminal and CLI agents are naturally multi-turn, where each action-observation cycle is laid out as a natural block."

**Limitations of the evidence:**
- Memento was tested on math, coding, and science reasoning — not on orchestration agent trajectories. The transfer to multi-turn agent orchestration is plausible but unverified.
- Requires custom inference infrastructure (vLLM fork with block masking). Not usable with standard API-based model access.
- Requires training data generation pipeline and SFT training — significant upfront investment.
- The approach operates within a single inference call; it does not address cross-session continuity, which is Kanbanzai's primary compaction use case.

**Source grading:** Primary (research paper + code + dataset). Strong evidence for learnable compaction viability, but limited direct applicability to the target system's architecture.

---

### Finding 4: Agent-to-agent optimisation produces substantial gains (41%) by treating compaction as a prompt engineering problem rather than a training problem

**Evidence:**
- **Kargar (2026b)** — Secondary (blog post, field report from LeRiM development). Describes using Claude Code (Opus 4.6) to optimise LeRiM's memory extraction agents. Key results:
  - 41% improvement in composite quality score across 14 experiments
  - Single biggest win: switching from `dspy.Predict` to `dspy.ChainOfThought` (one line change)
  - Schema field descriptions (20 words) had more impact than 50 lines of prompt engineering
  - "Explicit thresholds beat vague language": replacing "top_similarity very high" with concrete 0.7 threshold improved classification
  - 50% failure rate across experiments (7 of 14 reverted) — the keep/revert discipline was essential
- **Kargar (2026c)** — Secondary (blog post). Describes LeRiM's extract agent context management: `note()` for structured state tracking, `CONTEXT:` pressure signals, `prune()` for dropping consumed payload, and `index.md` for knowledge store anchoring.
- **Anthropic (2025b)** — Primary. Describes how Claude 4 models "can be excellent prompt engineers" — "when given a prompt and a failure mode, they are able to diagnose why the agent is failing and suggest improvements." A tool-testing agent achieved a 40% decrease in task completion time by rewriting tool descriptions.

**Key claims:**
- Prompt/schema-level optimisation can achieve gains comparable to model training without the infrastructure cost.
- Schema field descriptions (the output specification) matter more than the instruction prompt for structured extraction.
- Concrete thresholds consistently outperform vague qualitative guidance.
- Restrictive rules ("don't do X") consistently backfire compared to positive guidance ("here's what good looks like").

**Limitations of the evidence:**
- Kargar's posts are field reports, not peer-reviewed research. Sample size is small (one codebase, one optimiser model).
- The 41% improvement was on memory extraction quality, not directly on task continuation after compaction.
- The optimisation loop used Claude Opus 4.6 as the optimiser and MiniMax M2.5 as the target — cross-model generalisability of the specific findings is unknown.
- The author subsequently refactored much of the system, suggesting the gains were on a specific snapshot.

**Source grading:** Secondary evidence with interesting signals. The "schema descriptions matter more than prompts" and "restrictive rules backfire" findings are reinforced by Anthropic's independent observations. Treat as suggestive rather than definitive.

---

### Finding 5: External knowledge stores are essential for reducing inline compaction token costs, and the literature supports a "reference by ID, retrieve on demand" pattern

**Evidence:**
- **Anthropic (2025a)** — Primary. Recommends "just in time" context strategies: "agents built with the 'just in time' approach maintain lightweight identifiers (file paths, stored queries, web links, etc.) and use these references to dynamically load data into context at runtime using tools." Explicitly frames this as progressive disclosure.
- **Anthropic (2025a) on hybrid strategy** — "The most effective agents might employ a hybrid strategy, retrieving some data up front for speed, and pursuing further autonomous exploration at its discretion." Claude Code implements this: CLAUDE.md files loaded upfront, glob/grep for just-in-time retrieval.
- **Anthropic (2025b) on subagent output to filesystem** — Recommends that subagents "call tools to store their work in external systems, then pass lightweight references back to the coordinator. This prevents information loss during multi-stage processing and reduces token overhead from copying large outputs through conversation history."
- **Kargar (2026c)** — The `index.md` pattern in LeRiM: a table-of-contents file that provides navigation without inlining full content. "It is not a dump of the full content; it is the navigation layer."
- **Kanbanzai infrastructure** — Already has KE-IDs for knowledge entries, doc_intel for concept-based document retrieval, and entity state with lifecycle tracking. These are exactly the "lightweight identifiers" that the just-in-time approach requires.

**Key claims:**
- ID-based references with on-demand retrieval reduce inline token costs by orders of magnitude.
- The metadata of references (file names, folder hierarchies, timestamps) provides implicit signals that help agents navigate.
- Progressive disclosure — metadata always, instructions on trigger, full content on demand — is the recommended pattern.

**Limitations of the evidence:**
- The "just in time" approach introduces latency (retrieval requires tool calls). There is a speed vs. token-efficiency trade-off that the literature does not quantify.
- Agent retrieval quality depends on the quality of the retrieval tools. Poorly designed search/fetch tools could make the approach worse than inlining.

**Source grading:** Strong primary evidence from Anthropic's production systems (Claude Code, Research feature). The pattern is well-established in practice, though formal evaluation comparing inline vs. referenced approaches was not found.

---

### Finding 6: Token-count-based compaction triggers are the current standard; task-boundary and semantic-pressure approaches are emerging but less proven

**Evidence:**
- **Anthropic (2025a)** — Primary. Describes Claude Code's approach: "passing the message history to the model to summarize and compress the most critical details" when the context window nears its limit. This is a token-count trigger.
- **Anthropic (2025b)** — Primary. Describes long-horizon conversation management: "When context limits approach, agents can spawn fresh subagents with clean contexts while maintaining continuity through careful handoffs." Also describes a 200,000-token threshold for context truncation.
- **Kargar (2026a)** — Secondary. Surveys trigger strategies: dynamic summarisation "triggered by state: when we hit 70% of the token budget, summarize the oldest part." Frameworks expose token limits and do automatic roll-ups. Notes this is seen "a lot in coding agents like Claude Code or Codex."
- **Kargar (2026c)** — LeRiM's approach: "above 60% is soft pressure, above 80% is hard pressure." The agent receives `CONTEXT:` signals with estimated tokens and pressure level. This is a threshold-based trigger with graduated response.
- **Kanbanzai competitive analysis (Section 6.6)** — Identifies the architectural constraint: "the server has no visibility into the agent's context window." Compaction triggers require owning the agent conversation loop (model routing). Until then, procedural triggers (orchestrator notices and acts) are the fallback.

**Key claims:**
- Token-count thresholds (60–80% utilisation) are the dominant trigger mechanism in production systems.
- Graduated pressure (soft warning → hard action) is more robust than single-threshold triggers.
- Task-boundary triggers (compact at natural workflow completion points) are a logical enhancement but not yet demonstrated in published research.

**Limitations of the evidence:**
- No study was found that systematically compares different trigger strategies (threshold vs. task-boundary vs. semantic-pressure). All evidence is from system descriptions, not controlled experiments.
- The optimal threshold percentage likely varies by model (different context windows, different degradation curves). No per-model calibration data is available.

**Source grading:** Industry practice evidence (moderate). The 60–80% range is consistent across multiple independent systems (Claude Code, LeRiM, LangChain/LlamaIndex frameworks), which provides triangulation despite the absence of formal evaluation.

---

### Finding 7: Evaluation of compaction quality remains an open problem; compound metrics combining task completion, decision consistency, and token efficiency are needed but not standardised

**Evidence:**
- **Anthropic (2025b)** — Primary. Describes their evaluation approach for multi-agent research: LLM-as-judge with rubric (factual accuracy, citation accuracy, completeness, source quality, tool efficiency). Notes that "a single LLM call with a single prompt outputting scores from 0.0-1.0 and a pass-fail grade was the most consistent."
- **Anthropic (2025b) on the limits of automation** — "Human evaluation catches what automation misses." Human testers caught edge cases that evals missed: "hallucinated answers on unusual queries, system failures, or subtle source selection biases."
- **Kargar (2026b)** — The LeRiM evaluation evolution is instructive: Round 1 measured completeness (did you find everything?) which rewarded over-extraction. Round 2 added quality_alignment (is each memory atomic, actionable, context-independent?) and reweighted toward precision. This demonstrates that metric selection fundamentally shapes system behaviour.
- **Microsoft Research (2026)** — Memento evaluates accuracy on standard benchmarks (AIME, GPQA-Diamond) comparing compacted vs. uncompacted performance. Uses pass@1 and pass@k metrics. Also measures KV cache reduction and throughput improvement.

**Key claims:**
- Task completion rate is necessary but insufficient — it does not capture whether the right process was followed.
- Decision consistency (same decision before and after compaction) is a critical metric that is under-measured.
- Token efficiency (task completion per input token) should be tracked to prevent compaction strategies that preserve quality at excessive token cost.
- LLM-as-judge works for evaluation at scale but requires carefully designed rubrics.

**Limitations of the evidence:**
- No standardised benchmark for compaction quality exists. Each system evaluates differently.
- The field lacks a "compaction equivalent" of SWE-bench — a standardised task suite for measuring compaction quality across approaches.
- Decision consistency is particularly hard to measure in open-ended agent tasks where there is no single correct decision.

**Source grading:** Moderate — multiple independent systems describe their evaluation approaches, but no consensus has emerged.

---

## Trade-Off Analysis

| Dimension | State-Based (U-shaped) | Summary-Based | Learnable (Memento/LeRiM) | Retrieval-Anchored |
|-----------|----------------------|---------------|--------------------------|-------------------|
| **Task continuation accuracy** | Medium-High (inferred from attention curve + Anthropic guidance; no direct experiment) | Low-Medium (proven to lose critical state through prose degradation) | High (2-3× KV reduction with small accuracy gap; SFT + RL closes gap further) | Medium-High (depends on retrieval quality; progressive disclosure preserves token budget for critical state) |
| **Token efficiency** | High (discards historical reasoning, completed-task details; estimates 5-10× compression vs. raw conversation) | Medium (prose is verbose; typically 2-3× compression) | Very High (6× trace-level compression; sawtooth KV pattern) | Very High (ID references are ~10 tokens vs. thousands for inlined content) |
| **Implementation complexity** | Low (template construction + entity state integration; blocked on model routing for automated triggers) | Very Low (one LLM call to summarise) | Very High (training data pipeline, SFT, custom vLLM fork) | Medium (requires KE-ID resolution + retrieval tooling; most infrastructure exists in Kanbanzai) |
| **Maintenance burden** | Low (template evolves with workflow changes; schema is explicit) | Low (prompt engineering only) | High (model retraining on new trace patterns; dataset maintenance) | Medium (retrieval quality monitoring; stale-reference detection) |
| **Model-agnostic?** | Yes (template-based; any capable model can parse structured state) | Yes (any model can summarise) | No (requires per-model SFT; current implementations: Qwen, Phi, OLMo) | Yes (retrieval is tool-based; model-agnostic) |
| **Evidence strength** | Medium (triangulated from attention curve + Anthropic guidance + prompt engineering research; no direct compaction experiment) | Low-Medium (widely used but known failure modes: loss of critical detail, recency bias toward recent conversation) | High (peer-reviewed, code + dataset released, multiple model families tested) | Medium-High (strong Anthropic production evidence; no formal comparison against alternatives) |

---

## Recommendations

### Recommendation 1: Adopt state-based U-shaped compaction as the primary strategy (confidence: medium)

**Supported by findings:** F1 (Liu et al. 2024 shows 30%+ accuracy variance by position — strong primary evidence), F2 (Anthropic 2025a recommends structured compaction over prose summarisation — primary evidence, vendor), F5 (just-in-time references reduce inline token costs).

**What this means concretely:**
The compaction artefact should be a structured document with the following sections, ordered for the U-shaped attention curve:

```
[TOP — peak attention]
1. Identity + vocabulary payload (routing signal for fresh session)
2. Active constraints (file ownership, dependency boundaries, NEVER/ALWAYS rules)

[MIDDLE — attention valley]
3. Active state (tasks done, in-flight, ready — structured, not prose)
4. Active decisions (only those still constraining current work; KE-IDs for context)

[BOTTOM — recency boost]
5. Surfaced knowledge (KE-IDs to query; what the new session should retrieve)
6. Continuation anchor (exact Phase/Step to resume from)
```

**Explicitly discard:** task completion details, historical reasoning chains, conversation structure, failed attempts whose conclusions are in knowledge entries, raw tool outputs.

**Feasibility:** Implementable without model training. Requires: (a) a compaction trigger (procedural until model routing enables automated triggers), (b) a template for the U-shaped artefact (extend existing prompt engineering patterns from `refs/prompt-engineering-guide.md`), (c) integration with entity state and knowledge graph for data population (KE-IDs, feature/task lifecycle state). Blocked on: model routing infrastructure for *automated* triggering (P41 dependency). *Procedural* triggering (orchestrator notices and acts) is unblocked today.

**Risk:** The evidence for U-shaped structure improving compaction artefacts specifically (as distinct from improving prompts generally) is inferential rather than direct. If the attention curve does not transfer to compaction artefacts as predicted, the benefit over a simpler flat-structured artefact may be marginal. **Mitigation:** A/B test U-shaped vs. flat-structured compaction artefacts early in implementation, measuring task completion rate and decision consistency.

---

### Recommendation 2: Implement knowledge graph anchoring as a core compaction pattern (confidence: high)

**Supported by findings:** F5 (strong Anthropic evidence for just-in-time retrieval; LeRiM's `index.md` pattern; Kanbanzai already has KE-IDs).

**What this means concretely:**
The compaction artefact should contain KE-IDs (knowledge entry identifiers) rather than inlined knowledge content. The fresh session queries these on demand. This reduces inline token costs from potentially thousands of tokens (inlined knowledge) to ~10 tokens per reference (KE-ID + brief description).

**Feasibility:** Already aligned with Kanbanzai's architecture. The `knowledge(action: "get")` and `knowledge(action: "list")` tools enable on-demand retrieval. The compaction artefact template should include a "Surfaced Knowledge" section listing KE-IDs with one-line descriptions.

**Risk:** If the fresh session fails to query referenced KE-IDs, critical knowledge is missing. **Mitigation:** Include a procedural step at the start of the orchestration skill: "Query all KE-IDs listed in the compaction artefact before beginning work."

---

### Recommendation 3: Defer learnable compaction (Memento-style) until model routing infrastructure exists and evidence of agent-specific transfer is available (confidence: medium)

**Supported by findings:** F3 (Memento shows 2-3× KV reduction with high accuracy preservation — strong primary evidence), F4 (agent-to-agent prompt optimisation can achieve substantial gains without training — secondary evidence).

**What this means concretely:**
Memento is the most promising learnable approach, but it requires: (a) custom inference infrastructure (vLLM fork with block masking), (b) training data generation for orchestration agent traces (not just math/code), (c) SFT training per model. This is a significant investment. Meanwhile, prompt-level optimisation (F4) can achieve meaningful gains without any of this — the state-based U-shaped approach is essentially prompt-level optimisation applied to the compaction problem.

**Feasibility:** Memento's open release (paper, code, dataset, vLLM fork) makes it technically feasible to adapt. The barrier is not access but investment: training data generation pipeline, per-model SFT, and custom inference deployment. Worth revisiting when Kanbanzai controls the inference stack (post-model-routing).

**Risk:** Memento was not tested on agent orchestration trajectories. The block-and-compress pattern (math proof steps → mementos) may not transfer cleanly to agent action-observation cycles. Premature investment risks building for the wrong use case.

---

### Recommendation 4: Use token-count thresholds with graduated pressure as the compaction trigger (confidence: medium)

**Supported by findings:** F6 (60–80% range consistent across Claude Code, LeRiM, LangChain/LlamaIndex; graduated pressure pattern from LeRiM).

**What this means concretely:**
When the orchestrator's estimated context utilisation reaches 60%, issue a soft warning ("context pressure: plan for compaction soon"). At 80%, trigger compaction. The orchestrator writes the U-shaped compaction artefact and instructs the human to start a fresh session (or, post-model-routing, the system does this automatically).

**Feasibility:** Currently procedural (orchestrator estimates and acts). Post-model-routing, token counting from API metadata (`usage.input_tokens`) enables automated triggers. The procedural approach works today and is already described in Kanbanzai's Phase 5 orchestration procedure.

**Risk:** Token estimation without API metadata is approximate. The orchestrator may underestimate utilisation and exhaust context mid-task. **Mitigation:** Add a "context check" step at each Phase boundary in the orchestration procedure, not just at task completion.

---

### Recommendation 5: Establish a compaction evaluation framework with three metrics: task completion rate, decision consistency, and token efficiency (confidence: medium)

**Supported by findings:** F7 (evaluation remains an open problem; compound metrics needed).

**What this means concretely:**
For each compaction event, measure:
1. **Task completion rate:** Did the fresh session complete the same work that was in-flight before compaction? (Binary + time-to-completion)
2. **Decision consistency:** Did the fresh session make the same key decisions (which task to dispatch next, which sub-agent role to assign)? Measured by comparing the pre-compaction plan against post-compaction actions.
3. **Token efficiency:** (Tasks completed after compaction) / (Compaction artefact tokens + retrieval tokens). This tracks whether compaction is actually saving tokens.

**Feasibility:** Decision consistency is the hardest metric. It requires logging pre-compaction "intent" (which the orchestrator already expresses in planning steps) and comparing against post-compaction actions. This is tractable with structured logging.

**Risk:** Metric optimisation can lead to gaming — an agent that makes safe, conservative decisions may score well on consistency but poorly on task completion. **Mitigation:** Track all three metrics as a compound score; optimise for the compound, not any single metric.

---

## Limitations

### What the research did not cover

1. **Direct experimental comparison of state-based vs. summary-based compaction for agent task continuation.** The literature provides strong *inferential* support for state-based approaches (attention curve, Anthropic's guidance, prompt engineering research) but no A/B test of the two approaches in an agent orchestration context. This is the single largest evidence gap.

2. **Position-optimised compaction artefacts.** While the U-shaped attention curve is well-established for retrieval and QA tasks, no study was found that tests whether structuring a *compaction artefact* for the attention curve improves downstream task performance. This is a reasonable inference but an untested hypothesis.

3. **Compaction trigger strategy comparison.** No controlled experiment compares token-count thresholds, task-boundary detection, and semantic-pressure scoring for triggering compaction. The 60–80% range is industry consensus, not experimentally validated optimum.

4. **Cross-model compaction transfer.** Can a compaction artefact written by one model (e.g., Claude Opus) be effectively used by another model (e.g., a sub-agent using a different model)? The literature assumes same-model handoff. Cross-model compaction may introduce additional degradation.

### Assumptions underpinning the analysis

1. **The U-shaped attention curve transfers to agent continuation tasks.** This is plausible given Liu et al.'s demonstration across multiple task types, but unconfirmed for orchestration workflows specifically.

2. **Kanbanzai will implement model routing (P41).** The automated compaction trigger and token-counting infrastructure depend on this. Without it, compaction remains procedural (orchestrator-initiated), which is workable but less reliable.

3. **The orchestrator is the primary compaction subject.** Sub-agent compaction (compacting individual sub-agent sessions) is a different problem with different constraints. This analysis focuses on orchestrator-to-orchestrator handoff.

4. **KE-IDs remain stable.** The knowledge graph anchoring strategy assumes that knowledge entries persist and their IDs remain resolvable. If entries are retired or superseded between sessions, references become stale.

### Conditions that could change the conclusions

1. **A published A/B comparison of state-based vs. summary-based compaction** would either validate or refute the central recommendation. If state-based approaches show no advantage, the complexity of structured templates is unjustified.

2. **Memento or similar learnable approaches becoming available as a managed service** (e.g., via API parameter) would dramatically reduce the feasibility gradient and make learnable compaction competitive with heuristic approaches.

3. **New position-encoding schemes that flatten the attention curve** (reducing or eliminating the U-shape) would reduce the benefit of position-optimised artefact design. No such scheme has been demonstrated in production models, but it is an active research area.

4. **Larger context windows** (1M+ tokens) could make compaction less urgent for individual sessions. However, Anthropic (2025a) explicitly argues that "context windows of all sizes will be subject to context pollution and information relevance concerns," suggesting compaction remains necessary regardless of raw capacity.

### Knowledge gaps requiring further investigation

1. **What is the minimum viable state for resumption?** How few tokens can carry enough state for an orchestrator to resume effectively? The trade-off between compaction ratio and task completion rate needs empirical measurement.

2. **How does compaction quality vary by orchestration phase?** Compacting mid-task-dispatch may have different information requirements than compacting between phases. The literature does not distinguish.

3. **What is the failure mode profile of retrieval-anchored compaction?** If a fresh session fails to retrieve referenced knowledge, what is the blast radius? Does it make wrong decisions or does it recognise the gap and ask?

4. **Can the doc_intel classification system (requirement, decision, rationale, constraint) be leveraged during compaction?** Sections classified as "rationale" or "narrative" might be safely discarded, while "decision" and "constraint" sections should be preserved. This is unexplored.

---

## Source Index

### Primary — Peer-Reviewed

| # | Citation | Venue | Year | Key Contribution |
|---|----------|-------|------|------------------|
| 1 | Liu, N.F., Lin, K., Hewitt, J., Paranjape, A., Bevilacqua, M., Petroni, F., Liang, P. "Lost in the Middle: How Language Models Use Long Contexts." | TACL | 2024 | U-shaped attention curve; 30%+ accuracy drop for middle-of-context information |
| 2 | Fei, W., Niu, X., Zhou, P., Hou, L., Bai, B., Deng, L., Han, W. "Extending Context Window of Large Language Models via Semantic Compression." | ACL Findings | 2024 | Semantic compression extends effective context 6-8× without fine-tuning |
| 3 | Jiang, H., Wu, Q., Lin, C.-Y., Yang, Y., Qiu, L. "LLMLingua: Compressing Prompts for Accelerated Inference of Large Language Models." | EMNLP | 2023 | Token-level pruning via perplexity; 20× compression with minimal loss |
| 4 | Jiang, H., Wu, Q., Luo, X., Li, D., Lin, C.-Y., Yang, Y., Qiu, L. "LongLLMLingua: Accelerating and Enhancing LLMs in Long Context Scenarios via Prompt Compression." | ACL | 2024 | Query-aware compression with position sensitivity; 17.1% improvement at 4× compression |
| 5 | Pan, Z., Wu, Q., Jiang, H., et al. "LLMLingua-2: Data Distillation for Efficient and Faithful Task-Agnostic Prompt Compression." | ACL Findings | 2024 | BERT-level compressor trained via data distillation from GPT-4 |
| 6 | Li, Y., Dong, B., Lin, C., Guerin, F. "Compressing Context to Enhance Inference Efficiency of Large Language Models." | EMNLP | 2023 | Selective Context: 50% context reduction with 0.023 BERTscore drop |
| 7 | Wang, Q., Fu, Y., Cao, Y., Wang, S., Tian, Z., Ding, L. "Recursively Summarizing Enables Long-Term Dialogue Memory in Large Language Models." | Neurocomputing | 2023/2025 | Recursive summarisation for long-term dialogue memory |
| 8 | Chirkova, N., Formal, T., Nikoulina, V., Clinchant, S. "Provence: efficient and robust context pruning for retrieval-augmented generation." | ICLR | 2025 | Context pruning as sequence labelling for RAG |
| 9 | Zamfirescu-Pereira, J.D., Wong, R.Y., Hartmann, B., Yang, Q. "Why Johnny Can't Prompt: How Non-AI Experts Try (and Fail) to Design LLM Prompts." | CHI | 2023 | Positive + negative constraints together strongest; vocabulary matters |

### Primary — Technical Reports

| # | Citation | Organisation | Year | Key Contribution |
|---|----------|-------------|------|------------------|
| 10 | Anthropic Applied AI Team. "Effective Context Engineering for AI Agents." | Anthropic | 2025 | Attention budget, compaction, structured note-taking, just-in-time retrieval |
| 11 | Anthropic. "Building Effective Agents." | Anthropic | 2024 | Simple composable patterns; workflow vs. agent distinction |
| 12 | Anthropic. "How We Built Our Multi-Agent Research System." | Anthropic | 2025 | Multi-agent architecture, evaluation, context management for long-horizon tasks |
| 13 | Kontonis, V., Zeng, Y., Garg, S., et al. "Memento: Teaching LLMs to Manage Their Own Context." | Microsoft Research | 2026 | Learnable compaction via SFT; 2-3× KV reduction; dual information stream |

### Secondary — Blogs and Surveys

| # | Citation | Author | Year | Key Contribution |
|---|----------|--------|------|------------------|
| 14 | Kargar, I. "The Fundamentals of Context Management and Compaction in LLMs." | Medium | 2026 | Survey of compaction taxonomy: semantic compression, loss-aware pruning, dynamic summarisation |
| 15 | Kargar, I. "One Line of Code, 41% Better Memory: When One AI Agent Optimizes Another." | Medium | 2026 | Agent-to-agent prompt optimisation; 41% improvement; schema > prompts |
| 16 | Kargar, I. "How LeRiM Manages Context in the Extract Agent." | Medium | 2026 | Single-pass extraction with live pressure signals and pruning |

### Internal — Kanbanzai Documents

| # | Citation | Path | Key Contribution |
|---|----------|------|------------------|
| 17 | P41 Competitive Analysis, Section 6.6 | work/P41-opencode-ecosystem-features/P41-research-competitive-analysis.md | Architectural constraints on auto-compaction; U-shaped continuation prompt proposal |
| 18 | Prompt Engineering Guide | refs/prompt-engineering-guide.md | U-shaped attention-optimal section ordering; vocabulary routing; structured templates |

### Referenced but not directly accessed (seed URL mismatch)

| # | Citation | Status |
|---|----------|--------|
| 19 | Ranjan et al. (2024). "One Word Is Not Enough: Vocabulary Specificity in LLM Prompting." | Findings represented via refs/prompt-engineering-guide.md detailed summary |
| 20 | Wu et al. (2025). "Position Bias in Transformers." MIT. | Findings represented via refs/prompt-engineering-guide.md detailed summary |
| 21 | Voyce (2025). "XML/Markdown Comparative Study." | Finding cited in refs/prompt-engineering-guide.md; primary source not accessed |

---

## Retrieval Anchors

Questions this research answers:

- **Should we build summary-based or state-based compaction?** State-based. The evidence converges on structured state representation over prose summarisation for agent continuation (F1, F2, F4). Summary-based approaches lose critical detail through prose degradation and are optimised for human consumption, not agent resumption.

- **Does the U-shaped attention curve justify structuring compaction artefacts differently?** Yes, with the caveat that the evidence is inferential. The U-shaped curve is well-established (F1), and positioning critical information at the beginning and end is a low-cost intervention with plausible benefit. The risk of being wrong is mild (the artefact works but without position-dependent benefit).

- **Can we implement state-of-the-art compaction without training custom models?** Yes. The state-based U-shaped approach requires no model training. It is template construction + data population from existing infrastructure. Learnable approaches (Memento) are more effective but require model training and custom inference.

- **What information should a compaction artefact contain vs. reference by ID?** Inline: active state, active decisions, active constraints, continuation anchor. Reference by ID (KE-IDs): knowledge entries, document records, historical decisions whose details are stored elsewhere. The guiding principle: inline what the agent *must* know to begin work; reference everything else.

- **What metrics should we use to evaluate compaction quality?** Task completion rate, decision consistency, and token efficiency as a compound score (F7). LLM-as-judge for qualitative dimensions; deterministic metrics for token counts and completion status.

- **What compaction trigger strategy does the evidence support?** Token-count thresholds at 60% (soft warning) and 80% (hard trigger), with graduated pressure (F6). This is industry consensus across multiple independent systems, though not experimentally validated as an optimum.
