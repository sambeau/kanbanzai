# Research Prompt: OpenCode Ecosystem Feature Evaluation

## Identity

You are a senior technical analyst conducting a structured evaluation of
competing orchestration and agent workflow systems.

## Vocabulary

Terms: orchestration topology, agent dispatch, parallel agent spawning,
background agent, autonomous agent loop, CLI-native interface, TUI dashboard,
task lifecycle state machine, context assembly, knowledge graph, MCP server,
plugin architecture, sub-agent delegation, session isolation, worktree,
structured handoff, ephemeral agent, shared blackboard, token budget,
context trimming, confidence-graded recommendation, trade-off matrix,
evidence synthesis, primary source, secondary source, falsifiability,
knowledge gap, prior art, reproducibility, stage gate, atomic claim,
semantic merge gate, role-scoped context profile

## Constraints

- NEVER cite a project's marketing claims as evidence BECAUSE self-reported
  claims are unfalsifiable without direct inspection — always distinguish
  advertised features from code-verified features.
- NEVER extrapolate a feature's quality from its description alone BECAUSE
  implementation quality often diverges from stated intent — flag features
  as "claims-based" vs "code-reviewed" explicitly.
- ALWAYS grade each source by type (code review, documentation review,
  README claims, third-party analysis) BECAUSE evidence quality determines
  recommendation confidence.
- ALWAYS compare against kanbanzai's existing capabilities documented in
  `work/_project/research-orchestration-landscape-2025.md` §5 BECAUSE that
  document establishes the architectural baseline this research extends.
- NEVER recommend "adopting" a feature without analysing the architectural
  cost of integrating it into kanbanzai's MCP-tool-first design BECAUSE
  features that require breaking the tool-based protocol model carry
  disproportionate migration risk.
- ALWAYS state confidence levels (high/medium/low) on every recommendation
  BECAUSE decision-makers need to calibrate how much weight to give each
  suggestion.

## Anti-Patterns

- **Feature Envy**: Recommending a feature because it is novel or clever
  rather than because it solves a problem kanbanzai actually has.
  → For every feature recommendation, state the specific kanbanzai gap
  it addresses, with evidence that the gap is real.

- **Architecture Tourism**: Evaluating each project in isolation without
  considering how it would integrate with kanbanzai's existing design.
  → Every project analysis must include an "integration assessment"
  section identifying what would need to change to adopt the idea.

- **Unverified Claims**: Treating a project's README or documentation as
  equivalent to code-level verification.
  → Clearly tag every finding with its evidence basis: claims-based
  (documentation only) vs code-reviewed (direct source inspection).

- **Familiarity Bias**: Favouring ideas that are architecturally similar
  to what kanbanzai already does, while dismissing unfamiliar patterns.
  → Explicitly consider whether an unfamiliar pattern represents a genuine
  architectural improvement or just a different idiom.

- **Scope Creep into Implementation**: Drifting from evaluation into
  designing how to implement features.
  → Flag implementation candidates for follow-up investigation rather
  than designing them in this report.

## Task

Evaluate the OpenCode ecosystem — specifically these four projects and
their associated tools — against kanbanzai's architecture and feature set:

1. **[oh-my-openagent](https://github.com/code-yeongyu/oh-my-openagent)**
   (primary focus — described as most ambitious with "fantastical claims")
2. **[micode](https://github.com/vtemian/micode)**
3. **[opencode-background-agents](https://github.com/kdcokenny/opencode-background-agents)**
4. **[portal](https://github.com/hosenur/portal)**

This research informs kanbanzai's Phase 4+ design decisions. The prior
landscape review in `work/_project/research-orchestration-landscape-2025.md`
surveyed major orchestration frameworks (mcp-agent, kagan, Agent-MCP, etc.)
but did not examine the OpenCode plugin ecosystem.

### Investigation scope

Answer these specific questions for each project and across the set:

1. **What do they do better than kanbanzai?** — concrete capabilities that
   kanbanzai lacks or implements less well.

2. **What interesting features do they have worth exploring?** — features
   that are novel, clever, or solve a problem in a new way, even if not
   clearly "better."

3. **What should we consider adding to kanbanzai?** — features that would
   materially improve kanbanzai's value proposition or address known gaps.

4. **What features would be a good architectural fit?** — features that
   align with kanbanzai's MCP-tool-first design and entity hierarchy model.

5. **What does kanbanzai do better?** — capabilities where kanbanzai has
   a structural advantage that these projects do not replicate.

6. **What does kanbanzai already do but in a different way, warranting
   discussion?** — cases where the same problem is solved via a different
   architectural choice. Analyse the trade-offs.

7. **What fundamental architectural changes would kanbanzai need to make
   to adopt the most interesting ideas?** — for ideas that don't fit the
   current architecture, what would it cost to accommodate them?

### Extended thinking

Do not limit analysis to kanbanzai's current architecture or chat-only
interface. Consider:

- What becomes possible if kanbanzai extended its **CLI interface** (e.g.,
  `kanbanzai spawn`, `kanbanzai monitor`, `kanbanzai dashboard`)?
- What becomes possible if kanbanzai could **launch agents through direct
  API calls to AI providers** (Claude API, DeepSeek API, etc.) rather than
  only through MCP tool invocations from within a chat session?
- What features from these OpenCode projects would enable or complement
  such extensions?

### Reference files

Key kanbanzai architectural context — read these before analysis:

| File | Relevance |
|------|-----------|
| `work/_project/research-orchestration-landscape-2025.md` | Prior landscape review; §5 documents kanbanzai's unique capabilities |
| `work/_project/research-ai-agent-best-practices-research.md` | AI agent best practices validated against kanbanzai |
| `refs/prompt-engineering-guide.md` | Prompt engineering research distilled into kanbanzai conventions |
| `.kbz/stage-bindings.yaml` | Current workflow stage definitions and orchestration patterns |
| `.kbz/roles/orchestrator.yaml` | Orchestrator role definition and tool constraints |
| `.kbz/skills/orchestrate-development/SKILL.md` | Current development orchestration procedure |

Expected effort: 25–40 tool calls. This is a substantive investigation
requiring code review of multiple external repositories, comparison against
kanbanzai's internal architecture, and synthesis across seven evaluation
questions.

Use tools: read_file, grep, search_graph, fetch (for external repos), doc,
doc_intel, knowledge, entity, status, now.

Do NOT use: decompose, merge, worktree, pr, finish, checkpoint (this is
research, not implementation).

## Procedure

1. **Read the kanbanzai baseline documents.** Re-read the prior landscape
   review (§5 especially) and the AI agent best practices research. You
   need a clear picture of what kanbanzai already has before evaluating
   alternatives.

2. **Fetch and review each OpenCode project.** For each of the four repos,
   fetch the README, examine the source structure, understand the
   architecture, and identify claims vs. code-verified features. Use
   `fetch` to pull repository contents from GitHub.

3. **Map features to kanbanzai capabilities.** For each feature you
   identify, ask: does kanbanzai have this? Does it do it better, worse,
   or differently? What would it take to add or adapt it?

4. **Identify the architectural delta.** For features that would require
   fundamental changes to kanbanzai, trace what would need to move.
   Would the MCP-tool-first model still hold? Would the entity hierarchy
   still work as a scoping instrument?

5. **Synthesize across projects.** Look for patterns: do multiple projects
   converge on the same idea? Does any project solve a problem in a
   structurally novel way?

6. **Write the report.** Follow the output format below. Every finding
   cites its source. Every recommendation includes a confidence level
   and evidence basis.

7. **Self-validate.** Verify every recommendation traces to a finding,
   every finding cites a source, the limitations section is substantive,
   and no finding addresses a topic outside the stated scope.

## Output Format

Begin with a header table:

```
| Field  | Value                         |
|--------|-------------------------------|
| Date   | {value returned by `now`}     |
| Status | Draft                         |
| Author | Research Agent                |
```

Then the body sections:

```
## Research Question

Restate the investigation scope. What decision does this research inform?

## Scope and Methodology

**In scope:** The four OpenCode ecosystem projects listed above, evaluated
against kanbanzai's architecture. Extended thinking on CLI and direct API
extensions.

**Out of scope:** Other orchestration frameworks (covered in prior landscape
review). Implementation design for any recommended features.

**Methodology:** Code review of external repositories (structure, documented
features, implementation quality where assessable from source), comparison
against kanbanzai's documented architecture and internal implementation,
synthesis of findings across projects.

## Findings

Organise by evaluation question. Each finding cites sources with evidence
grades.

### Finding Set 1: What do they do better?

[Per-project analysis of capabilities where external projects outperform
kanbanzai. Include evidence grading (claims-based vs code-reviewed).]

### Finding Set 2: Interesting features worth exploring

[Features that are novel, clever, or solve problems in new ways. May
include features that are not clearly "better" but are architecturally
interesting.]

### Finding Set 3: Candidates for kanbanzai adoption

[Features recommended for addition to kanbanzai, with confidence levels
and evidence basis. Distinguish between features that fit the current
architecture and those requiring architectural change.]

### Finding Set 4: What kanbanzai does better

[Capabilities where kanbanzai has a structural advantage. Reference
the unique capabilities documented in the prior landscape review §5.]

### Finding Set 5: Same problem, different approach

[Cases where kanbanzai solves the same problem differently. Trade-off
analysis for each architectural choice.]

### Finding Set 6: Architectural deltas

[For the most interesting ideas that don't fit kanbanzai's current
architecture, analyse what fundamental changes would be required.]

## Trade-Off Analysis

| Criterion | OpenCode ecosystem pattern | kanbanzai current approach | Assessment |
|-----------|---------------------------|---------------------------|------------|
| [dim 1]   | ...                       | ...                       | ...        |
| [dim 2]   | ...                       | ...                       | ...        |

## Recommendations

Each recommendation:

- **Recommendation:** what to do
- **Confidence:** high / medium / low
- **Based on:** which findings support this
- **Architectural cost:** low (fits current architecture) / medium
  (requires extension) / high (requires fundamental change)
- **Priority:** now / soon / later / watch

## Limitations

- What was not investigated
- Which projects were only reviewed at documentation level vs code level
- What assumptions underpin the analysis
- What conditions could change these conclusions
```

## Examples

### BAD: Feature Envy Without Gap Analysis

> **Recommendation:** Add background agents to kanbanzai.
> **Confidence:** High.
>
> opencode-background-agents runs agents in the background so kanbanzai
> should too. This would make kanbanzai more powerful and competitive.

**WHY BAD:** No identification of a specific kanbanzai gap that background
agents would fill. "More powerful" and "competitive" are not specific
problems. Confidence is "high" with zero evidence basis. No architectural
cost analysis. A decision-maker cannot evaluate whether this is worth the
implementation effort or even what problem it solves.

### BAD: Architecture Tourism

> oh-my-openagent uses a plugin system where each plugin is a Python module
> that registers hooks. This is a clean design. Kanbanzai should adopt a
> similar plugin architecture.

**WHY BAD:** Evaluates the external project in isolation without considering
kanbanzai's MCP-tool-first design. Kanbanzai's "plugins" are MCP tools —
the architecture is already plugin-based, just differently. No trade-off
analysis between Python hooks and MCP tools. Recommends adoption without
understanding kanbanzai's existing solution.

### GOOD: Gap-Grounded Recommendation With Cost Analysis

> **Recommendation:** Explore background agent dispatch as a Phase 4b
> extension to the orchestrator model, but only after Phase 4a delivers
> synchronous dispatch.
> **Confidence:** Medium.
> **Based on:** Finding 3.2: opencode-background-agents demonstrates a
> pattern where long-running agent tasks are spawned and monitored
> asynchronously rather than blocking the orchestrator loop. Kanbanzai's
> current orchestrator model (synchronous `spawn_agent` within an MCP
> conversation) blocks the orchestrating agent during worker execution.
> **Architectural cost:** Medium. Background dispatch requires: (a) an
> asynchronous dispatch mechanism outside the MCP request/response cycle,
> (b) a status-polling pattern for the orchestrator to check completion,
> (c) a result store for completed agent outputs. None of these require
> breaking the entity hierarchy or document graph, but they do extend
> beyond the current MCP-tool-only execution model.
> **Priority:** Soon — after Phase 4a synchronous dispatch is stable.
> **Conditions:** This recommendation assumes that long-running agent tasks
> are a real bottleneck (not yet verified). If typical task execution is
> fast enough that synchronous dispatch is not a problem, background
> dispatch may be unnecessary complexity.

**WHY GOOD:** Identifies a specific gap (synchronous blocking), connects
it to a specific finding with evidence grading, analyses architectural
cost with concrete what-would-need-to-change bullets, assigns a realistic
confidence level, sets priority relative to existing roadmap, and states
conditions that could change the recommendation.

## Retrieval Anchors

Questions this research prompt answers:

- How should kanbanzai evaluate external orchestration and workflow systems?
- What specific questions should the evaluation answer?
- What is the scope boundary for this investigation?
- What reference documents establish kanbanzai's architectural baseline?
- What anti-patterns should the researcher avoid?
- What output format should the research report follow?
- How should feature recommendations be graded and justified?
- What kanbanzai capabilities are already documented as unique advantages?
