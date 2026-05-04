# Prompt: Research-Informed Architecture Review — Model Routing Agent & Fast-Track Architecture

## Context

You are the software architect for Kanbanzai, a Git-native workflow system for human-AI collaborative software development. The product, design, and development teams have produced draft designs for two major new features under the P41 (OpenCode Ecosystem Features) plan:

1. **Model Routing Agent** (P44 feasibility design) — giving Kanbanzai control over AI model selection, thinking-level control, provider fallback, token tracking, and auto-compaction by owning the agent dispatch loop.
2. **Fast-Track Architecture** (P43 design) — replacing mechanical human approval gates (spec approval, dev-plan approval, review verdict) with automated, evidence-backed validation using validator roles, risk-tiered automation levels, and an auto-approval pipeline.

These designs must be grounded in Kanbanzai's existing research, not rediscover known findings or repeat mistakes the field has already documented.

## Your Task

Review the new feature designs against two foundational research documents:

- **`research-agent-orchestration-research.md`** — A synthesis of 12 authoritative sources (Anthropic's building-effective-agents and multi-agent research system posts, Google Research's scaling agent systems paper, Masters et al.'s manager-agent POSG model, MetaGPT's SOP encode approach, SWE-agent's ACI concept, Microsoft's orchestration patterns, and others) mapping findings to Kanbanzai's specific problems.
- **`research-orchestration-landscape-2025.md`** — A landscape review of the 2025 orchestration ecosystem: major frameworks (mcp-agent, OpenAI Agents SDK, LangGraph, AutoGen, CrewAI), dedicated MCP orchestration servers (Agent-MCP, Network-AI, kagan, etc.), and an assessment of what Kanbanzai uniquely brings to the table.

Your output should serve as a briefing for the product, design, and development teams — translating research findings into actionable architectural guidance.

## What to Evaluate

### For Model Routing Agent (P44)

1. **Architecture choice (embedded vs. separate server):** Does the research support the "middle ground" approach (Option C: build together, extract later)? Do the validated patterns around "expose orchestration as MCP tools, not a framework" (research §7.5) and "ephemeral agents + shared knowledge store" (§7.1) favour one architecture over another?

2. **Category system design:** P44 proposes 5 categories mapped to Kanbanzai roles. Google Research's predictive model found that "architecture must match task structure" — parallelisable tasks benefit from multi-agent (+81%) while sequential reasoning tasks degrade (39–70%). Does the category system align with the task structures Kanbanzai actually has? Are the mappings between categories, models, and workflow stages consistent with what the research says about when parallelism helps vs. hurts?

3. **Auto-compaction and the U-shaped continuation prompt:** P44 proposes auto-compaction triggering at a configurable threshold (e.g., 70%) using a U-shaped continuation prompt. The research validates context management as critical — Microsoft's patterns explicitly recommend "monitor accumulated context size and use compaction techniques between agents" and Anthropic's multi-agent system used subagent output to filesystem with lightweight references. Does the proposed compaction design align with these validated patterns, or are there alternative approaches from the research the team should consider?

4. **Provider fallback and token tracking:** The research emphasizes "budget awareness must be visible to the receiving agent" (§7.4) — agents should know what was trimmed and why. Does P44's token tracking design (per-request token counts, aggregated per-feature/per-batch) address this? Should there be feedback to the agent about its own token consumption to trigger self-regulation?

5. **Relationship to the orchestrator-worker pattern:** The research identifies orchestrator-workers as the right pattern for parallelisable implementation tasks but warns against it for sequential reasoning tasks. P44's model routing would be used to dispatch both types. Does the design account for this distinction?

### For Fast-Track Architecture (P43)

1. **Validator role design and the ACI principle:** The research's strongest finding is that "Agent-Computer Interface design is as important as model capability" (SWE-agent, Anthropic). P43 defines three validator roles with checklists. Are these designed as effective ACIs for the validator agents? Does the design account for the finding that "bad tool descriptions can send agents down completely wrong paths" — i.e., are the validation checks unambiguous enough for an agent to execute reliably?

2. **Enforceable constraints vs. advisory instructions:** The research is unequivocal: "Enforceable constraints beat advisory instructions" (§2.2). Every source comparing the two finds enforcement wins. P43's validators produce pass/fail verdicts — but are these *enforceable* within the workflow state machine, or are they advisory? MetaGPT's SOPs with intermediate verification gates, Microsoft's programmatic gates, and Masters et al.'s hard constraints (ℋ) all provide stronger models. Does P43 go far enough?

3. **Decomposition quality as the critical path:** Multiple sources converge on "decomposition quality is the single strongest predictor of overall workflow success" (§2.3). P43's plan-validator checks structural completeness (all tasks reference spec requirements, dependency graph is acyclic). But does it evaluate *decomposition quality* — task granularity, independence, clear boundaries? The research warns that "without detailed task descriptions, agents duplicate work, leave gaps, or fail to find necessary information" (Anthropic multi-agent). Are the plan-validator's checks sufficient to catch poor decomposition?

4. **Risk tiers and the sequential penalty:** P43 defines risk tiers that control which gates are automated vs. human. Google Research found that applying the wrong orchestration pattern to a task type causes structural quality degradation. Does the risk tier system risk applying validator agents to tasks where they're structurally inappropriate? For example, if `retro_fix` skips all gates, does that risk introducing unverified changes?

5. **Session management and context degradation:** P43 mandates that validators must run in fresh sessions via `spawn_agent`. This directly implements the "ephemeral agents + shared knowledge store" pattern (§7.1) and avoids the context degradation that the research warns about. Are there additional session-management patterns from the research (subagent output to filesystem, context budget visibility, output summarization) that should be incorporated?

### Cross-Cutting Concerns

1. **Model selection for validators vs. authors:** The research on "proactive orchestrator vs. reactive communicator" patterns (Masters et al.) shows that model capability qualitatively changes action patterns — stronger models decompose 14.5× more, track dependencies 26× more. P43's open question #5 asks whether validators should use different models. Given the research, is the audit/compliance cognitive profile of validators different enough from creative authoring to warrant model differentiation even before model routing is built? P43 currently says "same model with near-zero temperature and different role prompt" — does the research support this, or does it suggest validators need the stronger reasoning model?

2. **Interaction between fast-track and model routing:** P43's validators running as spawned agents would benefit from model routing — the `audit` category maps directly to validator roles. The P41 plan defers model routing implementation until after fast-track is stable. Should the fast-track design anticipate model routing (e.g., through an abstraction layer for agent dispatch), or is "same model, different temperature" sufficient for the initial implementation?

3. **The validation bottleneck pattern:** Google Research found that centralized orchestration contained error amplification to 4.4× (vs. 17.2× for independent agents) because "the orchestrator acts as a validation bottleneck." P43's validators are exactly this — validation bottlenecks at each stage gate. Does the design fully leverage this pattern, or are there additional points where validation bottlenecks could prevent error propagation?

4. **What the research says NOT to build:** The research explicitly warns against several things. Check whether either design falls into these traps:
   - Over-engineering: "The most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns" (Anthropic).
   - Wrong architecture for the task: applying multi-agent coordination to sequential reasoning tasks degrades performance 39–70% (Google).
   - Unnecessary coordination complexity (Microsoft anti-pattern).
   - Sharing mutable state between concurrent agents (Microsoft anti-pattern).

## Output Format

Please produce a structured review organized as:

1. **Executive Summary** — 3–5 key findings the teams must act on
2. **Model Routing Agent: Research Alignment** — per-point evaluation above
3. **Fast-Track Architecture: Research Alignment** — per-point evaluation above
4. **Cross-Cutting Findings** — interactions, tensions, and synergies between the two features
5. **Risk Register** — specific risks from the research that the current designs don't mitigate
6. **Recommendations** — prioritized, actionable changes to the designs, with explicit citations to the research that supports each recommendation

## Source Documents

- `work/_project/research-agent-orchestration-research.md` — primary research synthesis (12 sources, 5 cross-cutting findings, 8 recommendations for Kanbanzai 3.0)
- `work/_project/research-orchestration-landscape-2025.md` — landscape review (frameworks, MCP servers, capability matrix, Kanbanzai's unique capabilities)
- `work/P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md` — model routing feasibility design
- `work/P43-fast-track-architecture/P43-design-fast-track-architecture.md` — fast-track architecture design
- `work/P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md` — parent plan with dependency structure
- `work/P41-opencode-ecosystem-features/P41-research-competitive-analysis.md` — competitive analysis (OmO, micode, etc.)

## Constraints

- Focus on research-to-design alignment, not on whether the features are good ideas in the abstract
- Cite specific sections from the research documents (e.g., "§2.2 Enforceable Constraints Beat Advisory Instructions") rather than paraphrasing
- If the research is silent on a design choice, say so — don't invent findings
- Distinguish between findings that should change the design vs. findings that confirm the design is on the right track
- Keep recommendations concrete: what to change, why, and which research supports it
