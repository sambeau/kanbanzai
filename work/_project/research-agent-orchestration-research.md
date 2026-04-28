# Research Report: AI Agent Orchestration for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-29 |
| Author | Research Agent |
| Status | Draft |
| Proposal | `work/research/AI-agent-orchestration-research-proposal.txt` |

---

## Executive Summary

This report synthesises findings from 12 authoritative sources — academic papers, industry engineering blogs, and reference architectures — on AI agent orchestration, with a focus on practical implications for the Kanbanzai 3.0 redesign. The research was conducted in response to observations that AI agents working with Kanbanzai exhibit inconsistent behaviour: skipping workflow steps, failing to discover tools, producing inconsistent specifications, and rushing to implementation.

**The headline finding is that Kanbanzai's problems are not primarily instruction-quality issues — they are well-documented, fundamental challenges of multi-agent workflow management.** The most rigorous study in our corpus (Masters et al., 2025) found that even GPT-5-based manager agents "struggle to jointly optimize for goal completion, constraint adherence, and workflow runtime" — a three-way trade-off that cannot be resolved through better prompting alone.

However, the research identifies concrete, practical mechanisms that significantly improve agent orchestration quality:

1. **Agent-Computer Interface (ACI) design** is as important as model capability. Tool descriptions, parameter naming, and format choices have measurable impact on agent performance — as much as upgrading the underlying model (SWE-agent; Anthropic).
2. **Standardised Operating Procedures (SOPs) encoded as enforceable constraints** outperform advisory instructions. Systems that *prevent* wrong-order execution outperform systems that *advise against* it (MetaGPT; Microsoft).
3. **Decomposition quality is the critical path.** Performance gains correlate almost linearly with the quality of the induced task graph, not with agent count or token budget (Masters et al.; Google Research).
4. **Architecture must match task structure.** Multi-agent coordination dramatically improves parallelisable tasks (+81%) but *degrades* sequential reasoning tasks by 39–70% (Google Research). Software specification is fundamentally sequential.
5. **Proactive orchestration beats reactive communication.** Stronger reasoning models don't just produce better answers — they use qualitatively different action patterns: more decomposition, more dependency tracking, more refinement. Weaker models default to messaging and status-checking (Masters et al.).

The report is organised into four parts: an annotated bibliography of sources reviewed, a synthesis of cross-cutting findings, a mapping of findings to Kanbanzai's specific problems, and a set of concrete recommendations for the 3.0 redesign.

---

## Table of Contents

1. [Sources Reviewed](#1-sources-reviewed)
2. [Cross-Cutting Findings](#2-cross-cutting-findings)
3. [Mapping to Kanbanzai's Problems](#3-mapping-to-kanbanzais-problems)
4. [Recommendations for Kanbanzai 3.0](#4-recommendations-for-kanbanzai-30)
5. [Further Research Avenues](#5-further-research-avenues)
6. [Bibliography](#6-bibliography)

---

## 1. Sources Reviewed

### 1.1 Primary Sources (High Relevance)

These sources directly address the problems Kanbanzai faces and offer actionable findings.

#### Anthropic — "Building Effective Agents" (Dec 2024)

- **URL:** https://www.anthropic.com/engineering/building-effective-agents
- **Type:** Industry engineering blog (Anthropic)
- **Relevance:** ⭐⭐⭐⭐⭐

Anthropic's foundational guide to agent system design, based on working with dozens of teams building LLM agents across industries. Defines a taxonomy of agentic system patterns: prompt chaining, routing, parallelisation, orchestrator-workers, evaluator-optimizer, and full autonomous agents. Each pattern has clear "when to use" and "when not to use" guidance.

**Key findings for Kanbanzai:**

- "The most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns." This validates Kanbanzai's core approach but warns against over-engineering the instruction layer.
- The Agent-Computer Interface (ACI) concept: "Think about how much effort goes into human-computer interfaces (HCI), and plan to invest just as much effort in creating good agent-computer interfaces." Tool descriptions, parameter naming, and format choices matter enormously.
- Tool poka-yoke: "Change the arguments so that it is harder to make mistakes." If the tool interface *allows* agents to skip steps, they will.
- "We actually spent more time optimizing our tools than the overall prompt" — when building their SWE-bench agent.

#### Anthropic — "How We Built Our Multi-Agent Research System" (Jun 2025)

- **URL:** https://www.anthropic.com/engineering/multi-agent-research-system
- **Type:** Industry engineering blog (Anthropic)
- **Relevance:** ⭐⭐⭐⭐⭐

Detailed engineering post-mortem on Anthropic's production multi-agent Research feature. Covers system architecture, prompt engineering for multi-agent systems, evaluation methodology, and production reliability challenges.

**Key findings for Kanbanzai:**

- **Delegation quality:** "Each subagent needs an objective, an output format, guidance on the tools and sources to use, and clear task boundaries. Without detailed task descriptions, agents duplicate work, leave gaps, or fail to find necessary information."
- **Effort scaling:** "Agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts." Simple tasks: 1 agent, 3–10 tool calls. Complex tasks: 10+ subagents with clearly divided responsibilities. Agents must be *told* how much thinking to do.
- **Tool design:** "Bad tool descriptions can send agents down completely wrong paths, so each tool needs a distinct purpose and a clear description." They built a tool-testing agent that used tools dozens of times and rewrote descriptions to avoid failures, achieving a 40% decrease in task completion time.
- **Let agents improve themselves:** Claude 4 models can diagnose prompt failures and suggest improvements when given a prompt and a failure mode.
- **Subagent output to filesystem:** "Direct subagent outputs can bypass the main coordinator for certain types of results, improving both fidelity and performance." Subagents write to external systems and pass lightweight references back — reducing token overhead from copying large outputs through conversation history.
- **Evaluation:** "Start evaluating immediately with small samples" — a set of about 20 test cases was enough to spot dramatic changes in early development. LLM-as-judge evaluation scales well when using a single prompt with 0.0–1.0 scores.

#### Google Research — "Towards a Science of Scaling Agent Systems" (Jan 2026)

- **URL:** https://research.google/blog/towards-a-science-of-scaling-agent-systems-when-and-why-agent-systems-work/
- **Paper:** Kim & Liu, Google Research
- **Type:** Academic research + industry blog
- **Relevance:** ⭐⭐⭐⭐⭐

The first quantitative scaling study of multi-agent systems. Evaluated 180 agent configurations across five canonical architectures (single-agent, independent, centralised, decentralised, hybrid) and four benchmarks. Derives predictive scaling principles rather than heuristics.

**Key findings for Kanbanzai:**

- **The "Alignment Principle":** On parallelisable tasks (e.g., financial analysis where distinct sub-analyses can run simultaneously), centralised multi-agent coordination improved performance by **80.9%** over a single agent.
- **The "Sequential Penalty":** On tasks requiring strict sequential reasoning (planning), **every** multi-agent variant degraded performance by **39–70%**. Communication overhead fragmented the reasoning process, consuming "cognitive budget" needed for the actual task.
- **The "Tool-Use Bottleneck":** As tasks require more tools (16+), the "tax" of coordinating multiple agents increases disproportionately.
- **Error amplification:** Independent multi-agent systems (agents working in parallel without communication) amplified errors by **17.2×**. Centralised systems (with an orchestrator) contained this to **4.4×**. The orchestrator acts as a "validation bottleneck" — catching errors before they propagate.
- **Predictive model:** They developed a model (R² = 0.513) using task properties (tool count, decomposability) to predict optimal architecture. It correctly identifies the best coordination strategy for **87% of unseen tasks**.

#### Masters et al. — "Orchestrating Human-AI Teams: The Manager Agent as a Unifying Research Challenge" (Oct 2025)

- **URL:** https://arxiv.org/abs/2510.02557
- **Published:** DAI 2025 (oral paper)
- **Type:** Academic (peer-reviewed)
- **Relevance:** ⭐⭐⭐⭐⭐

The most directly relevant academic paper. Formalises autonomous workflow management as a Partially Observable Stochastic Game (POSG) and proposes the "Autonomous Manager Agent" as a core research challenge. Releases MA-Gym, an evaluation framework with 20 diverse workflows. Evaluates GPT-5-based manager agents empirically.

**Key findings for Kanbanzai:**

- **The fundamental trade-off:** "Goal achievement, constraint adherence, and workflow runtime cannot all be maximized simultaneously." This is the core tension Kanbanzai faces.
- **GPT-5 vs GPT-4.1 — qualitatively different styles:**
  - GPT-5 performed 14.5× more task decompositions, 7.8× more refinements, 26× more dependency additions — *proactive orchestrator* style.
  - GPT-4.1 used 2.4× more messaging, 10× more status queries, 9× more no-ops — *reactive communicator* style.
  - "Stronger reasoning models support more proactive orchestration, but reasoning alone is insufficient."
- **Four foundational challenges:** (1) compositional reasoning for hierarchical decomposition, (2) multi-objective optimisation under shifting preferences, (3) coordination and planning in ad hoc teams, (4) governance and compliance by design. All four map directly to Kanbanzai's problem space.
- **Decomposition quality is the bottleneck:** "Performance gains correlate almost linearly with the quality of the induced task graph — underlining that structure learning, not raw language generation, is the critical path."
- **Hard vs soft constraints:** Hard constraints (ℋ) must always hold — violation terminates the workflow. Soft constraints (𝒮) can be violated with penalties. This distinction maps to Kanbanzai's lifecycle gates (hard) vs quality preferences (soft).
- **MA-Gym's action taxonomy** closely parallels Kanbanzai's MCP tools: `assign_task`, `decompose_task`, `refine_task`, `add_task_dependency`, `inspect_task`, `send_message`, `get_workflow_status`. This validates Kanbanzai's tool design.

#### MetaGPT — "Meta Programming for Multi-Agent Collaborative Framework" (Aug 2023)

- **URL:** https://arxiv.org/abs/2308.00352
- **Published:** ICLR 2024
- **Type:** Academic (peer-reviewed)
- **Relevance:** ⭐⭐⭐⭐⭐

The closest academic analogue to Kanbanzai. Introduces the concept of encoding Standardised Operating Procedures (SOPs) into prompt sequences for multi-agent software engineering. Uses an "assembly line paradigm" with role assignment and intermediate verification.

**Key findings for Kanbanzai:**

- **SOPs encoded as prompt sequences** with intermediate verification gates reduced cascading hallucination errors. Agents are *required* to produce structured intermediate artifacts that are verified before the next stage begins.
- **Assembly line paradigm:** Diverse roles assigned to agents, with structured handoffs. Each agent produces a defined output format (e.g., PRD → system design → API spec → code) that the next agent can parse and verify.
- **Structured intermediate artifacts beat freeform outputs.** MetaGPT generated more coherent software solutions than "chat-based" multi-agent systems where agents simply converse.
- **The key mechanism:** Verification of intermediate results. If an agent can skip straight to code without its specification being validated, quality degrades.

#### SWE-agent — "Agent-Computer Interfaces Enable Automated Software Engineering" (May 2024)

- **URL:** https://arxiv.org/abs/2405.15793
- **Published:** Princeton (ICLR 2025 submission context)
- **Type:** Academic (peer-reviewed)
- **Relevance:** ⭐⭐⭐⭐

The paper that coined the term "Agent-Computer Interface" (ACI). Core finding: **interface design affects agent performance as much as model capability.** Purpose-built interfaces for agents to navigate codebases, edit files, and run tests dramatically outperformed giving agents raw shell access.

**Key findings for Kanbanzai:**

- LM agents represent "a new category of end users with their own needs and abilities, and would benefit from specially-built interfaces."
- The ACI design significantly enhanced agents' ability to create/edit code, navigate repositories, and execute tests — not through better models, but through better tool design.
- Tool format choices matter: agents struggle with diffs (need to know chunk sizes before writing), JSON-escaped code, and tools that require maintaining accurate counts. Tools should match what the model has seen in training data.

### 1.2 Supporting Sources (Moderate Relevance)

These sources provide useful context, taxonomy, or background but are less directly actionable.

#### Microsoft — "AI Agent Orchestration Patterns" (Feb 2026)

- **URL:** https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns
- **Type:** Reference architecture (Microsoft Azure)
- **Relevance:** ⭐⭐⭐⭐

Comprehensive taxonomy of five orchestration patterns: sequential, concurrent, group chat, handoff, and magentic (dynamic planning). Excellent for naming and formalising what Kanbanzai already does.

**Key findings for Kanbanzai:**

- **Start with the right level of complexity.** Three levels: direct model call → single agent with tools → multi-agent orchestration. "Use the lowest level of complexity that reliably meets your requirements."
- **Maker-checker loops:** One agent creates, another evaluates against defined criteria. "This pattern requires clear acceptance criteria for the checker agent so that it can make consistent pass or fail decisions. An iteration cap is used to prevent infinite refinement loops."
- **Sequential orchestration with gates:** Programmatic checks on intermediate steps ensure the process stays on track. The choice of next agent is deterministic, not left to agent discretion.
- **Context management:** "Monitor accumulated context size and use compaction techniques between agents to prevent exceeding model limits or degrading response quality."
- **Common anti-patterns:** Creating unnecessary coordination complexity; sharing mutable state between concurrent agents; consuming excessive model resources because context windows grow as agents accumulate information.

#### AutoCodeRover — "Autonomous Program Improvement" (Apr 2024)

- **URL:** https://arxiv.org/abs/2404.05427
- **Published:** ISSTA 2024
- **Type:** Academic (peer-reviewed)
- **Relevance:** ⭐⭐⭐

Relevant for how agents should *search for context*. Uses AST-level program representation rather than treating code as files, and uses iterative, structured search to build context. Achieved better results at *lower cost* than SWE-agent.

**Key findings for Kanbanzai:**

- Working on structured program representations (AST) rather than raw file text produces better results.
- Iterative, structured context retrieval (search → evaluate → refine → search again) outperforms single-pass context gathering.
- Average cost of $0.43 USD per issue resolution, demonstrating that structured search is dramatically more token-efficient than brute-force context loading.

#### Guo et al. — "LLM-based Multi-Agents: A Survey of Progress and Challenges" (Jan 2024)

- **URL:** https://arxiv.org/abs/2402.01680
- **Type:** Academic survey
- **Relevance:** ⭐⭐⭐

Comprehensive survey covering agent profiling, communication protocols, and capability growth mechanisms. Good for ensuring no established approaches are missed.

**Key findings for Kanbanzai:**

- Communication taxonomy: cooperative, debate, and competitive paradigms. Kanbanzai's model is primarily cooperative with maker-checker (debate) elements in review.
- Agent profiling: agents with explicit role descriptions and constraints outperform generic agents given the same task.
- Memory mechanisms: short-term (conversation context), long-term (external stores), and episodic (task-specific learnings). Kanbanzai's knowledge system covers all three.

#### SWE-bench — "Can Language Models Resolve Real-World GitHub Issues?" (Oct 2023)

- **URL:** https://arxiv.org/abs/2310.06770
- **Published:** ICLR 2024
- **Type:** Academic (peer-reviewed, benchmark)
- **Relevance:** ⭐⭐⭐

The benchmark underpinning all SE agent research. Provides the evaluation methodology and dataset that SWE-agent, AutoCodeRover, and others are measured against. Useful if Kanbanzai develops its own evaluation framework for orchestration quality.

### 1.3 Contextual Sources (Lower Relevance)

#### IBM — "What is AI Agent Orchestration?"

- **URL:** https://www.ibm.com/think/topics/ai-agent-orchestration
- **Type:** Industry explainer
- **Relevance:** ⭐⭐

High-level overview of orchestration types (centralised, decentralised, hierarchical, federated). Useful for vocabulary but not directly actionable. Provides a reasonable taxonomy of challenges: multi-agent dependencies, coordination, scalability, fault tolerance, data privacy, and adaptability.

#### Orchestral AI — "A Framework for Agent Orchestration" (Jan 2026)

- **URL:** https://arxiv.org/abs/2601.02577
- **Type:** Academic framework paper
- **Relevance:** ⭐⭐

A Python agent framework paper. Interesting for specific tool design patterns (type-safe tools, pre/post execution hooks, read-before-edit safety, external modification detection), but primarily about framework plumbing rather than orchestration strategy. The hook system (pre-execution approval, post-execution summarisation) is a pattern worth noting.

### 1.4 Inaccessible Sources

The ACM paper (`dl.acm.org/doi/10.1145/3772429.3772439`) was initially blocked by Cloudflare but was later identified as Masters et al. (2025) and accessed via the arXiv preprint (arXiv:2510.02557). Full text was reviewed and is covered above.

---

## 2. Cross-Cutting Findings

Five themes emerge consistently across the sources.

### 2.1 Agent-Computer Interface Design Is a First-Class Concern

**Sources:** SWE-agent, Anthropic (both articles), Orchestral AI

The research consistently finds that tool interface design has as much impact on agent performance as model capability. This is not an intuitive finding — the natural assumption is that better models produce better results regardless of interface. But the evidence is clear:

- SWE-agent's custom ACI "significantly enhances an agent's ability to create and edit code files, navigate entire repositories, and execute tests" — not through a better model, but through better tools.
- Anthropic's multi-agent team achieved a "40% decrease in task completion time" by having an agent rewrite tool descriptions after testing them dozens of times.
- Anthropic's SWE-bench agent team "spent more time optimizing our tools than the overall prompt."

**Design principles from the research:**

1. **Put yourself in the model's shoes.** If the tool description and parameters aren't obvious, the model will misuse them.
2. **Keep formats close to training data.** Agents have seen humans using command lines, writing markdown, editing files. Tools that match these patterns perform better than novel formats.
3. **Eliminate formatting overhead.** Don't require agents to maintain accurate counts, produce valid JSON for complex structures, or compute diffs. These "overhead" requirements consume reasoning capacity.
4. **Make wrong usage hard (poka-yoke).** If parameters can be misused, they will be. Constrain parameter spaces, provide defaults, and reject invalid combinations at the tool level.
5. **Give each tool a distinct purpose.** Overlapping or ambiguous tool boundaries cause agents to choose the wrong tool. If two tools are easily confused, merge them or make the distinction unmistakable.

### 2.2 Enforceable Constraints Beat Advisory Instructions

**Sources:** MetaGPT, Microsoft, Masters et al., Google Research

Every source that compares "telling agents what to do" with "preventing agents from doing the wrong thing" finds the latter wins decisively.

- MetaGPT's SOPs encoded as prompt sequences with intermediate verification gates reduced cascading errors — agents *must* produce structured artifacts that pass verification before advancing.
- Microsoft's sequential orchestration pattern uses programmatic gates: "You can add programmatic checks on any intermediate steps to ensure that the process is still on track."
- Masters et al.'s POSG model explicitly distinguishes hard constraints (ℋ — must always hold, violation terminates) from soft constraints (𝒮 — can be violated with penalties). The most reliable workflows have hard constraints for critical ordering and soft constraints for quality preferences.
- Google's centralised orchestration contained error amplification to 4.4× (vs 17.2× for independent agents) specifically because the orchestrator validates intermediate outputs.

**The implication is stark:** If Kanbanzai's lifecycle gates are enforced through instructions ("you should complete the specification before implementation"), agents will skip them when under pressure or when the task seems simple. If they're enforced through tool constraints (the implementation tool *refuses to operate* on a feature that hasn't passed specification review), compliance is guaranteed.

### 2.3 Decomposition Quality Is the Critical Path

**Sources:** Masters et al., Anthropic (multi-agent), MetaGPT, Google Research

Multiple independent sources converge on the same finding: the quality of task decomposition — how well a high-level goal is broken into structured, concrete sub-tasks — is the single strongest predictor of overall workflow success.

- Masters et al.: "Performance gains correlate almost linearly with the quality of the induced task graph — underlining that structure learning, not raw language generation, is the critical path."
- Anthropic: Without detailed task descriptions for subagents, "agents duplicate work, leave gaps, or fail to find necessary information."
- MetaGPT: The assembly line paradigm explicitly structures decomposition as a phase with defined output formats.
- Google: The predictive model's strongest feature was "decomposability" — whether the task can be cleanly split into independent sub-tasks.

**Critical caveat from Google:** Decomposition only helps when sub-tasks are genuinely independent. For sequential reasoning tasks (specification, design, planning), decomposition into parallel sub-tasks actively *degrades* performance. The task structure must dictate the decomposition strategy, not the other way around.

### 2.4 Architecture Must Match Task Structure

**Sources:** Google Research, Microsoft, Anthropic (building effective agents)

This is the most counterintuitive finding: "more agents" is not universally better, and the optimal architecture depends on task properties.

- **Parallelisable tasks** (information gathering, independent analyses, multi-file code changes): Multi-agent coordination with centralised orchestration improves performance by up to 81%.
- **Sequential reasoning tasks** (specification writing, design decisions, planning): Every multi-agent variant tested degraded performance by 39–70%. The communication overhead fragments the reasoning process.
- **Tool-heavy tasks** (16+ tools): Coordination overhead grows disproportionately with tool count.

Google's predictive model identifies the optimal architecture for 87% of unseen tasks using just two properties: **sequential dependencies** and **tool density**. This suggests that Kanbanzai should not apply the same orchestration pattern to every workflow stage — specification (sequential, low tool density) should use a different pattern than implementation (parallelisable, high tool density).

### 2.5 Proactive Orchestration vs Reactive Communication

**Sources:** Masters et al., Anthropic (multi-agent)

The most operationally relevant finding for Kanbanzai's inconsistency problem. Masters et al. discovered that stronger reasoning models don't just produce "better" outputs — they exhibit qualitatively different *action patterns*:

| Behaviour | Proactive Orchestrator (GPT-5) | Reactive Communicator (GPT-4.1) |
|---|---|---|
| Task decomposition | 14.5× more frequent | Rare |
| Task refinement | 7.8× more frequent | Rare |
| Dependency tracking | 26× more frequent | Rare |
| Messaging | 0.4× (less) | 2.4× more frequent |
| Status checking | 0.1× (much less) | 10× more frequent |
| No-ops | 0.1× (much less) | 9× more frequent |

The proactive orchestrator builds structured chains: decompose → refine → assign. The reactive communicator loops: send_message → send_message → status_check.

**This maps directly to Kanbanzai's experience.** Some agents engage deeply with the workflow tools (decompose, plan, structure dependencies) while others skip straight to implementation, checking status occasionally but never properly decomposing or specifying. The difference is partly model capability and partly system design — if the system makes decomposition the path of least resistance, even weaker models will follow it.

---

## 3. Mapping to Kanbanzai's Problems

The research proposal identifies five symptoms of inconsistent agent behaviour. Here is what the research says about each.

### 3.1 "Agents don't discover the system / forget to use it"

**Root cause (per research):** Tool discoverability and description quality.

**Evidence:**

- Anthropic (multi-agent): "Tool design and selection are critical... Bad tool descriptions can send agents down completely wrong paths." They gave agents explicit heuristics: "examine all available tools first, match tool usage to user intent."
- Anthropic (building effective agents): "Put yourself in the model's shoes. Is it obvious how to use this tool, based on the description and parameters?"
- SWE-agent: Purpose-built interfaces for agents dramatically outperform raw access to the same underlying functionality.

**Diagnosis for Kanbanzai:** The MCP tool descriptions, parameter names, and `AGENTS.md` instructions may be optimised for human understanding, not agent understanding. An agent dropped into a fresh context with just the tool list should be able to infer the correct workflow. If the tool list reads as an undifferentiated wall of 22 tools with similar descriptions, agents will default to the tools they already know (read_file, grep, terminal) rather than learning new ones.

**Recommended interventions:**

1. **Audit tool descriptions from the agent's perspective.** For each MCP tool, ask: if I saw only the tool name, description, and parameter list, would I know *when* to use this tool and *instead of what*?
2. **Add explicit heuristics to tool descriptions.** Not just "what this tool does" but "use this tool INSTEAD OF reading .kbz/ files directly" or "call this BEFORE starting implementation."
3. **Consider a tool-testing agent.** Following Anthropic's approach, build an agent that attempts to use Kanbanzai tools on sample tasks, records failures, and rewrites descriptions. This is the most direct way to surface discoverability problems.
4. **Reduce tool count through consolidation.** The 2.0 MCP redesign consolidated tools, but 22 is still a lot. Google's research shows coordination overhead grows with tool count. Consider whether agents need all 22 tools in every context, or whether role-specific tool subsets would be more effective.

### 3.2 "Agents skip steps in the workflow"

**Root cause (per research):** Advisory constraints rather than enforceable constraints.

**Evidence:**

- MetaGPT: SOPs with intermediate verification gates — agents *must* produce verified artifacts before advancing.
- Microsoft: Sequential orchestration with programmatic gates.
- Masters et al.: Hard constraints (ℋ) that terminate the workflow on violation vs soft constraints (𝒮) that penalise violations. The most reliable workflows have hard constraints on critical ordering.
- Google: Centralised orchestration catches errors (4.4× amplification) vs independent execution amplifies them (17.2×).

**Diagnosis for Kanbanzai:** The lifecycle state machine is the right mechanism. But if agents can invoke implementation tools (spawn_agent for coding, terminal for running tests) on a feature that hasn't passed through specification review, the lifecycle is advisory rather than enforceable. The system *permits* skipping; it doesn't *prevent* it.

**Recommended interventions:**

1. **Make lifecycle gates enforceable at the tool level.** The `next` / `handoff` tools should refuse to assemble implementation context for a feature that hasn't passed specification review. Task completion tools should validate that the parent feature is in the correct state.
2. **Distinguish hard and soft constraints explicitly.** Hard: lifecycle state transitions must follow the state machine. Soft: code style, test coverage targets, documentation quality. Hard constraints should be system-enforced. Soft constraints should be flagged in review.
3. **Build verification into stage transitions.** Before a feature transitions from specification → development, run automated checks: Does the specification document exist? Does it have the required sections? Has it been approved? These checks should be programmatic, not agent-judged.

### 3.3 "Agents are too keen to get to implementation"

**Root cause (per research):** Agents cannot judge appropriate effort levels; sequential reasoning suffers from premature parallelisation.

**Evidence:**

- Anthropic (multi-agent): "Agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts." Explicit guidelines for effort allocation prevented overinvestment in simple queries and underinvestment in complex ones.
- Google: The sequential penalty — on tasks requiring step-by-step reasoning, multi-agent coordination degrades performance by 39–70%. Software specification is fundamentally sequential: understand the problem → design → specify → implement.
- Masters et al.: The "Assign-All" baseline (bulk upfront planning, skip reasoning) achieved higher goal completion but lower constraint adherence than the CoT baseline. Speed and compliance are in tension.

**Diagnosis for Kanbanzai:** Agents are optimised (by training) to be helpful and productive. Writing code *feels* productive; writing specifications *feels* like overhead. Without explicit effort budgets, agents will allocate minimal effort to specification and maximum effort to implementation — because implementation produces visible artifacts (code, tests, running programs) while specification produces documents that don't *do* anything.

**Recommended interventions:**

1. **Embed effort expectations in task context.** When the `handoff` or `next` tool assembles context for a specification task, include explicit guidance: "This specification task should involve 5–15 tool calls before producing output. Read the design document, query relevant knowledge entries, check for related decisions, and draft a structured specification. Do not proceed to implementation."
2. **Make specification a *required output*.** Following MetaGPT's assembly-line approach, require that specification tasks produce a structured document in a defined format. The document template itself guides effort — if the template has 8 sections that need filling, the agent can't skip to implementation after 2 tool calls.
3. **Separate specification and implementation agent sessions.** Google's sequential penalty finding suggests that specification should be a dedicated, sequential, single-agent activity — not parallelised with implementation. The specification agent should have *no access* to implementation tools (terminal, file editing). It should only have access to reading tools, knowledge queries, and document writing.

### 3.4 "Inconsistent specifications and plans"

**Root cause (per research):** Freeform output without structured templates or evaluation criteria.

**Evidence:**

- MetaGPT: Assembly line with structured intermediate artifacts. Each stage produces a defined output format that the next agent can parse and verify.
- Microsoft: Maker-checker loops with clear acceptance criteria: "This pattern requires clear acceptance criteria for the checker agent so that it can make consistent pass or fail decisions."
- Anthropic (multi-agent): "Each subagent needs an objective, an output format, guidance on the tools and sources to use, and clear task boundaries."
- Anthropic (multi-agent): For evaluation, "a single LLM call with a single prompt outputting scores from 0.0–1.0 and a pass-fail grade was the most consistent and aligned with human judgements."

**Diagnosis for Kanbanzai:** If a SKILL says "write a specification" without defining the exact sections, required fields, and acceptance criteria, agents will produce whatever format seems natural to them — which varies by model, by context window contents, and by how much context they've gathered. The inconsistency is an *output format* problem as much as a *quality* problem.

**Recommended interventions:**

1. **Define specification and plan templates with required sections.** Not just "write a specification" but "produce a specification with these sections: Problem Statement, Requirements (functional/non-functional), Constraints, Acceptance Criteria, Verification Plan. Each section must contain at least one concrete item."
2. **Implement maker-checker review.** Following Microsoft's pattern, specification tasks should be followed by an automated review step that checks: Are all required sections present? Are acceptance criteria testable? Does the specification reference the parent design document? The checker should have explicit pass/fail criteria.
3. **Use LLM-as-judge for qualitative evaluation.** Following Anthropic's approach, build a simple evaluation rubric: factual accuracy, completeness (are all required sections covered?), testability (are acceptance criteria concrete?), and consistency (does it align with the design document?). Score 0.0–1.0 on each dimension.

### 3.5 "Agents don't appear to understand the workflow system"

**Root cause (per research):** Insufficient orientation; reactive communication pattern; tool surface doesn't guide discovery.

**Evidence:**

- Anthropic (multi-agent): "Think like your agents. To iterate on prompts, you must understand their effects." They built simulations to watch agents work step-by-step to observe failure modes.
- Masters et al.: Weaker models default to "reactive communicator" patterns — messaging and status-checking rather than planning and structuring. Stronger models default to "proactive orchestrator" patterns.
- Google: Centralised orchestration with a validation bottleneck outperforms independent agent execution.
- Anthropic (building effective agents): "Frameworks can help you get started quickly, but don't hesitate to reduce abstraction layers."

**Diagnosis for Kanbanzai:** Agents may be receiving too much information about the workflow system (long AGENTS.md, multiple SKILL files, complex lifecycle documentation) without a clear *entry point* that shows them how to interact with it. The system description is optimised for human understanding (progressive disclosure through documents) rather than agent understanding (immediate tool-level affordances).

**Recommended interventions:**

1. **Mandatory orientation step.** Every agent session should begin with a structured orientation: call `status` to see current project state, call `next` to see the work queue. This should be enforced either through instructions that are always present, or through a tool that combines orientation into a single call.
2. **Progressive tool disclosure.** Rather than exposing all 22 MCP tools at once, consider role-based tool subsets. A specification agent doesn't need `worktree`, `merge`, `pr`, or `cleanup`. An implementation agent doesn't need `decompose` or `doc`. Fewer, more relevant tools reduce cognitive load and increase the probability of correct tool selection.
3. **Build understanding through use, not documentation.** Instead of requiring agents to read workflow documentation, design the tools so that *using them* teaches the workflow. The `next` tool already does this — it assembles context for a task, including the feature lifecycle state, relevant specifications, and knowledge entries. Double down on this approach: make the tools self-documenting through their inputs and outputs.

---

## 4. Recommendations for Kanbanzai 3.0

Based on the research findings, the following recommendations are ordered by expected impact.

### 4.1 High-Impact: Enforce Lifecycle Gates at the Tool Level

**Research basis:** MetaGPT (SOPs), Microsoft (gates), Masters et al. (hard constraints), Google (validation bottleneck)

Make the lifecycle state machine *architecturally enforceable*, not just advisory. Specifically:

- **`handoff` / `next` tools** should refuse to assemble implementation context for features not in the correct lifecycle state.
- **`finish` tool** should validate that the feature's prerequisites (specification approved, design document exists) are met before allowing task completion to advance the feature.
- **Sub-agent delegation** should include lifecycle state checks — the orchestrating agent should not be able to spawn an implementation agent for a feature that hasn't been specified.

This is the single highest-impact change because it converts the entire class of "agents skip steps" failures from a quality problem into an impossibility.

### 4.2 High-Impact: Redesign Tool Descriptions as ACIs

**Research basis:** SWE-agent (ACI), Anthropic (both articles), Google (tool-use bottleneck)

Treat MCP tool descriptions as a user interface designed for agents, not as API documentation for humans. Specifically:

- **Audit every tool description** for agent comprehension. Each should answer: What does this do? When should I use it? What should I use *instead* if this isn't right?
- **Add negative guidance** ("do NOT read .kbz/ files directly — use this tool instead").
- **Consider a tool-testing protocol** where an agent attempts common workflows using only the tool list, and failures are used to improve descriptions.
- **Evaluate role-based tool subsets** to reduce cognitive load per agent session.

### 4.3 High-Impact: Structured Output Templates for Specifications and Plans

**Research basis:** MetaGPT (structured artifacts), Anthropic (output format requirements), Microsoft (acceptance criteria)

Define mandatory output templates for specification, plan, and review documents. Each template should specify:

- Required sections with descriptions
- Minimum content expectations per section
- Cross-reference requirements (must reference design document, must reference relevant decisions)
- Acceptance criteria format (must be testable/verifiable)

The template itself acts as a forcing function — agents must engage with each section, which distributes effort across the specification rather than allowing rush-to-implementation.

### 4.4 Medium-Impact: Effort Budgets per Workflow Stage

**Research basis:** Anthropic (effort scaling), Google (sequential penalty), Masters et al. (proactive vs reactive)

Embed explicit effort expectations in task context assembly. When the `handoff` tool assembles context for a task, include metadata about expected effort:

- Specification tasks: "Expect 5–15 tool calls. Read design docs, query knowledge, draft sections."
- Implementation tasks: "Expect 10–50 tool calls. Read spec, implement, test, iterate."
- Review tasks: "Expect 5–10 tool calls. Read artifact, check against criteria, produce pass/fail."

This directly addresses the "agents struggle to judge appropriate effort" finding and nudges reactive communicators toward proactive orchestration patterns.

### 4.5 Medium-Impact: Maker-Checker Review Automation

**Research basis:** Microsoft (maker-checker loops), Anthropic (LLM-as-judge), Masters et al. (constraint adherence)

Implement automated review as a standard workflow step:

1. **Structural checks** (programmatic): Required sections present? Cross-references valid? Acceptance criteria listed?
2. **Quality checks** (LLM-as-judge): Completeness score (0.0–1.0), consistency with design document, testability of acceptance criteria.
3. **Iteration cap:** Maximum 2–3 revision cycles before escalating to human review.

This addresses specification inconsistency without requiring human review of every artifact — humans review the review criteria, not every specification.

### 4.6 Medium-Impact: Match Orchestration Pattern to Task Type

**Research basis:** Google (sequential penalty, alignment principle), Microsoft (pattern selection)

Different workflow stages should use different orchestration patterns:

| Stage | Task Structure | Recommended Pattern | Agent Count |
|---|---|---|---|
| Specification | Sequential, low tool density | Single agent, no parallelism | 1 |
| Design review | Evaluative, low tool density | Maker-checker (2 agents) | 2 |
| Implementation | Parallelisable, high tool density | Orchestrator-workers | 1 + N workers |
| Code review | Evaluative, medium tool density | Single agent or panel | 1–3 |
| Integration testing | Sequential, medium tool density | Single agent with tools | 1 |

The key insight: specification and design review should *never* be parallelised. Implementation can be. Applying the wrong pattern to the wrong stage is a structural cause of quality degradation.

### 4.7 Lower-Impact: Observability and Evaluation Infrastructure

**Research basis:** Anthropic (both articles), Masters et al. (MA-Gym metrics)

Build infrastructure to measure orchestration quality:

- **Action pattern logging:** Track which MCP tools each agent session calls, in what order, and how many times. This enables detecting reactive-communicator patterns (lots of messaging, few decompositions) vs proactive-orchestrator patterns.
- **Stage-level metrics:** Time spent in each lifecycle stage, number of revision cycles, pass rate of automated checks.
- **Small-sample evaluation:** Following Anthropic's approach, maintain a set of 15–20 representative workflow scenarios and evaluate orchestration changes against them. "A set of about 20 queries representing real usage patterns" was sufficient to spot dramatic impacts.

### 4.8 Lower-Impact: Context Compaction Between Agent Sessions

**Research basis:** Microsoft (context management), Anthropic (multi-agent, subagent output to filesystem)

When orchestrating multi-agent workflows, prevent context growth from degrading quality:

- **Subagent outputs should be summarised.** Following Anthropic's pattern, subagents should write detailed results to the filesystem and pass lightweight references back to the orchestrator.
- **The `handoff` tool should assemble focused context.** It already does this to some degree — the recommendation is to be more aggressive about excluding irrelevant context and including only what the receiving agent needs.
- **Monitor context window utilisation.** If agents routinely hit context limits, they're receiving too much context. Track this and use it to improve context assembly.

---

## 5. Further Research Avenues

### 5.1 Academic Sources

- **Semantic Scholar** (`semanticscholar.org`): Search for "multi-agent software engineering" and "LLM agent workflow" filtered to 2024–2025. The citation graph from MetaGPT and SWE-agent leads to a rich cluster of follow-up work.
- **ICLR 2024/2025 and ICSE 2024/2025 proceedings:** The intersection of AI agents and software engineering is a hot topic at both venues. Search for "LLM agent software development."
- **Google DeepMind's ADAS (Automated Design of Agentic Systems):** Research on automatically designing agent architectures, potentially informing how Kanbanzai self-improves its own orchestration.
- **Cemri et al. (2025), "Why Do Multi-Agent LLM Systems Fail?"** (arXiv:2503.13657): Cited by Masters et al., directly studies failure modes. Could provide a taxonomy of Kanbanzai failure modes.
- **Wang et al. (2025), "MegaAgent: A large-scale autonomous LLM-based multi-agent system without predefined SOPs"** (ACL 2025 Findings): Contrasts with MetaGPT's SOP approach — potentially useful as a counterpoint.

### 5.2 Industry Sources

- **Anthropic's research page** (anthropic.com/engineering): Regular engineering blogs. The Claude Code team's practices around `CLAUDE.md`, custom commands, and hooks are a lived example of operationalising agent instructions.
- **OpenAI's "Practices for Governing Agentic AI Systems":** Policy-oriented, but useful framing for human-AI delegation boundaries.
- **The MA-Gym codebase** (github.com/DeepFlow-research/manager_agent_gym): Open-source evaluation framework. Could be adapted to evaluate Kanbanzai orchestration changes directly.

### 5.3 Adjacent Research

- **Instruction-following evaluation:** How well do LLMs follow complex, multi-step instructions? This is a related but distinct research area from agent orchestration. Papers on instruction-following benchmarks (IFEval, FollowBench) could inform how Kanbanzai's instructions should be structured for maximum compliance.
- **Prompt engineering for structured output:** Research on getting LLMs to produce consistent structured output (JSON mode, constrained decoding, output grammars) is relevant to the specification template recommendation.
- **Human-AI teaming:** Vats et al. (2024), "A Survey on Human-AI Teaming with Large Pre-Trained Models" (arXiv:2403.04931). Relevant to Kanbanzai's document-centric human interface model.

---

## 6. Bibliography

### Academic Papers

1. Masters, C., Vellanki, A., Shangguan, J., Kultys, B., Gilmore, J., Moore, A., & Albrecht, S.V. (2025). "Orchestrating Human-AI Teams: The Manager Agent as a Unifying Research Challenge." *DAI 2025* (oral). arXiv:2510.02557.

2. Hong, S., Zhuge, M., Chen, J., Zheng, X., et al. (2023). "MetaGPT: Meta Programming for A Multi-Agent Collaborative Framework." *ICLR 2024*. arXiv:2308.00352.

3. Yang, J., Jimenez, C.E., Wettig, A., Lieret, K., Yao, S., Narasimhan, K., & Press, O. (2024). "SWE-agent: Agent-Computer Interfaces Enable Automated Software Engineering." arXiv:2405.15793.

4. Zhang, Y., Ruan, H., Fan, Z., & Roychoudhury, A. (2024). "AutoCodeRover: Autonomous Program Improvement." *ISSTA 2024*. arXiv:2404.05427.

5. Jimenez, C.E., Yang, J., Wettig, A., Yao, S., Pei, K., Press, O., & Narasimhan, K. (2023). "SWE-bench: Can Language Models Resolve Real-World GitHub Issues?" *ICLR 2024*. arXiv:2310.06770.

6. Guo, T., Chen, X., Wang, Y., Chang, R., Pei, S., Chawla, N.V., Wiest, O., & Zhang, X. (2024). "Large Language Model based Multi-Agents: A Survey of Progress and Challenges." arXiv:2402.01680.

7. Roman, A. & Roman, J. (2026). "Orchestral AI: A Framework for Agent Orchestration." arXiv:2601.02577.

### Industry Sources

8. Anthropic. (2024). "Building Effective Agents." https://www.anthropic.com/engineering/building-effective-agents

9. Anthropic. (2025). "How We Built Our Multi-Agent Research System." https://www.anthropic.com/engineering/multi-agent-research-system

10. Kim, Y. & Liu, X. (2026). "Towards a Science of Scaling Agent Systems: When and Why Agent Systems Work." Google Research Blog. https://research.google/blog/towards-a-science-of-scaling-agent-systems-when-and-why-agent-systems-work/

11. Microsoft. (2026). "AI Agent Orchestration Patterns." Azure Architecture Center. https://learn.microsoft.com/en-us/azure/architecture/ai-ml/guide/ai-agent-design-patterns

12. IBM. (2025). "What is AI Agent Orchestration?" IBM Think. https://www.ibm.com/think/topics/ai-agent-orchestration