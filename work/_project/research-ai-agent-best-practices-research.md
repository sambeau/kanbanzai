# Research Report: AI Agent Best Practices for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-13 |
| Author | Research Agent |
| Status | Draft |
| Sources | "10 Claude Code Principles" (jdforsythe.github.io/10-principles) — a research-based distillation of 17 peer-reviewed papers |

---

## Executive Summary

This report analyses the "10 Claude Code Principles" — a research-backed framework for AI agent workflows distilled from 17 academic papers — and evaluates each principle against the Kanbanzai workflow system. The goal is to identify what Kanbanzai already does well, where meaningful improvements can be made, and what changes should inform the design of Kanbanzai 3.0.

**The headline finding is that Kanbanzai is architecturally well-aligned with the research.** The system's core design decisions — document-led workflow, structured YAML state, externalized plans, separation of orchestration from execution, human checkpoints, context profiles, and knowledge management — directly implement several of the principles without modification. This is not accidental; many of the same engineering instincts that produced this research also shaped Kanbanzai's design.

However, there are significant opportunities for improvement, particularly in:

1. **Skill architecture** — restructuring SKILLs to follow the attention-optimized section ordering and vocabulary routing patterns the research identifies as the primary quality lever.
2. **Context hygiene tooling** — building MCP tools that enforce progressive disclosure and minimal context loading, rather than relying on agent discipline.
3. **Specialized review panels** — evolving code review from a single-profile approach to domain-specialized reviewer panels with vocabulary routing.
4. **Institutional memory enforcement** — making the knowledge system proactively surface relevant always/never rules during context assembly, not just on query.
5. **Observability infrastructure** — adding structured logging for agent handoffs, tool calls, and review metrics to detect the failure modes the MAST taxonomy documents.
6. **Token economy awareness** — encoding cascade escalation logic and cost tracking into the orchestration tools.

Each principle is analysed individually below, followed by cross-cutting themes and a consolidated set of recommendations for Kanbanzai 3.0.

---

## Research Sources

The 10 Principles draw on 17 published sources spanning 2017–2026:

| Source | Year | Key Contribution | Relevant Principles |
|--------|------|-------------------|---------------------|
| Vaswani et al., "Attention Is All You Need" | 2017 | Transformer architecture, self-attention, n² pairwise relationships | P2 |
| Zamfirescu-Pereira et al., "Why Johnny Can't Prompt" (CHI) | 2023 | Positive + negative constraints together are strongest | P5 |
| Hong et al., MetaGPT | 2023 | Structured artifacts reduce errors ~40% vs. free dialogue | P4, P7 |
| Liu et al., "Lost in the Middle" | 2024 | 30%+ accuracy drop when critical info is in middle of context | P2, P10 |
| Ranjan et al., "One Word Is Not Enough" | 2024 | Vocabulary specificity activates domain-specific knowledge clusters | P6, P10 |
| PRISM Persona Framework | 2024 | <50 token identities optimal; flattery degrades output | P6, P10 |
| MAST Failure Taxonomy | 2024–2025 | 14 failure modes in multi-agent systems; rubber-stamp approval as #1 quality failure | P7, P8, P10 |
| Captain Agent Research | 2024 | Adaptive team composition outperforms static teams by 15–25% | P9 |
| LangChain Few-Shot Research | 2024 | 3 well-chosen examples match 9 in effectiveness | P3, P10 |
| Anthropic, "Building Effective Agents" | Dec 2024 | Agent vs. workflow distinction, structured handoffs | P7 |
| Wu et al., MIT Position Bias | 2025 | Causal masking and RoPE as architectural causes of U-shaped attention | P2 |
| DeepMind Multi-Agent Scaling | 2025 | 45% threshold, saturation at 4 agents, superlinear coordination costs | P9, P10 |
| Voyce, XML/Markdown Comparative Study | 2025 | Prompt format alone accounts for up to 40% performance variance | P3, P4 |
| Anthropic, "Effective Context Engineering" | Sep 2025 | Attention budget concept, progressive disclosure | P2, P9 |
| Anthropic, "Harness Design for Long-Running Development" | Mar 2026 | Separation of generation from evaluation improves quality | P6, P10 |
| Vaarta Analytics, "Prompt Engineering Is System Design" | 2026 | At n=19 requirements, accuracy drops below n=5 | P5, P10 |

---

## Principle-by-Principle Analysis

---

### Principle 1: The Hardening Principle

> *Every fuzzy LLM step that must behave identically every time must eventually be replaced by a deterministic tool.*

#### Key Research Findings

- LLMs are probabilistic — same input, different output. For mechanical steps (file I/O, format conversion, data lookup), this is a liability.
- Hardened pipelines go from ~70% reliability to 100%.
- The LLM's production role should be orchestration and fuzzy reasoning, not mechanical execution.
- Silent failures are the most expensive failures; hardened tools fail loudly or succeed completely.
- Use the LLM to prototype the tool that replaces the LLM.

#### What Kanbanzai Already Does Well

**This is Kanbanzai's foundational design philosophy.** The entire MCP server is an implementation of the Hardening Principle:

- **Entity lifecycle transitions** are deterministic state machines enforced by code (`internal/validate/`), not by LLM judgment. An agent cannot skip a lifecycle state — the tool rejects invalid transitions.
- **YAML serialisation** is canonical and deterministic (`internal/storage/`). Field order, formatting, encoding — all defined in code and tested with round-trip tests.
- **ID allocation** is deterministic (`internal/id/`). No LLM decides what the next feature ID should be.
- **Document registration, knowledge contribution, health checks** — all mechanical operations handled by deterministic MCP tools.
- **Context assembly** (`internal/context/assemble.go`) is a deterministic function: given a role profile and a task, it produces a structured context packet via code, not by asking an LLM to "figure out what context is needed."

The LLM's role in Kanbanzai is exactly what the research recommends: orchestration (deciding what tool to call), fuzzy reasoning (summarising, reviewing, decomposing work), and intent interpretation. Mechanical execution is handled by the MCP tools.

#### Actionable Improvements for Kanbanzai 3.0

1. **Harden decomposition validation.** The `decompose` tool currently relies on LLM judgment for task breakdown quality. The decomposition _output_ could be validated deterministically: does each task have a parent feature? Are file paths real? Do dependency references resolve? Add a deterministic validation pass after LLM-generated decomposition.

2. **Harden review finding classification.** The code review SKILL asks agents to classify findings as blocking/non-blocking. The classification criteria are well-defined in the SKILL. Some of these checks (e.g., "does the finding cite a specific spec requirement?") could be enforced by a deterministic post-processing step rather than relying on the LLM to self-police.

3. **Harden context budget estimation.** The code review SKILL documents expected context sizes per role. This could be a deterministic tool: given a review unit (file list + spec sections), estimate the token budget and warn if it exceeds the target range.

**Priority:** Medium. Kanbanzai already follows this principle architecturally. The improvements are incremental hardening of specific steps within already-functioning workflows.

---

### Principle 2: The Context Hygiene Principle

> *Context is your scarcest resource. Treat it like memory in an embedded system, not disk space on a server.*

#### Key Research Findings

- **The attention budget** (Anthropic, Sep 2025): every token competes with every other token for attention weight. Irrelevant tokens actively degrade performance.
- **U-shaped attention curve** (Liu et al., 2024): accuracy drops 30%+ when critical information is in the middle of context. Front-load constraints, back-load instructions.
- **Optimal utilisation zone**: 15–40% of the context window. Below ~10%, hallucination risk increases. Above ~60%, attention dilution dominates.
- **Progressive disclosure** works in four layers: always-loaded identity (~200–500 tokens), task-triggered SOPs (~500–2K), on-demand full docs (2K+), compressed summaries.
- **Cross-conversation isolation**: each session is completely isolated. Design for single-session completeness.
- **Context poisoning**: stale context is worse than no context — it actively misdirects.

#### What Kanbanzai Already Does Well

Kanbanzai's context system is directly aligned with this research:

- **Context profiles** (`.kbz/context/roles/`) implement progressive disclosure. The `base.yaml` profile is ~200 tokens of identity and conventions. The `developer.yaml` and `reviewer.yaml` profiles add task-specific context. The `handoff` tool assembles exactly the context a sub-agent needs — spec sections, knowledge entries, file paths — nothing more.
- **Context assembly** (`internal/context/assemble.go`) is the mechanism for "load only what the current task demands." It reads the role profile, resolves inheritance, gathers relevant knowledge entries, and produces a focused context packet.
- **Knowledge scoping**: knowledge entries have scope (project-level vs. session-level) and confidence scores. The system already filters by relevance rather than dumping everything.
- **Document intelligence** (`internal/docint/`) provides structural access to documents — agents can read specific sections via `doc_intel(action: "section")` rather than loading entire documents.

The code review SKILL's Context Budget Strategy section (lines 612–649) is an explicit implementation of the progressive disclosure pattern: orchestrators hold only metadata (~6–14 KB), sub-agents hold only their review unit (~12–30 KB).

#### Actionable Improvements for Kanbanzai 3.0

1. **Position-aware context assembly.** The current `assemble.go` constructs context packets but does not control the _ordering_ of content within the packet. The research says order matters enormously: identity and hard constraints first (high attention), supporting reference in the middle, step-by-step instructions and retrieval anchors last (high attention). Context assembly should enforce this ordering.

2. **Context budget estimation tool.** Add an MCP tool (or extend `handoff`) that estimates the token count of a context packet before dispatching a sub-agent. If the estimate exceeds the 40% utilisation threshold, warn the orchestrator and suggest splitting the work unit. This makes the "15–40% optimal zone" actionable rather than aspirational.

3. **Context freshness metadata.** Knowledge entries already have timestamps, but context profiles and SKILLs do not carry `last-verified` metadata. Adding this would allow the system to detect when a context profile is feeding stale conventions to agents — a direct implementation of the "stale context is poisoned context" finding.

4. **SKILL section ordering.** Restructure SKILLs to follow the attention curve: vocabulary/identity at the top, instructions in the middle (structured as numbered steps to survive attention degradation), retrieval anchors ("Questions This Skill Answers") at the bottom. The current code review SKILL does not follow this ordering.

5. **Warn on context bloat.** When `handoff` assembles a context packet, if the total knowledge entries + spec sections + conventions exceed a threshold, the tool should flag this and suggest the orchestrator split the unit of work. Currently, there is no feedback mechanism for context overload.

**Priority:** High. Context hygiene is the single most impactful improvement area. The infrastructure exists; the improvements are about making it position-aware, size-aware, and self-monitoring.

---

### Principle 3: The Living Documentation Principle

> *Documentation is context. Stale documentation is poisoned context.*

#### Key Research Findings

- Documentation IS few-shot context. Every example in docs is a demonstration the model pattern-matches against. Stale examples are poisoned examples.
- 3 well-chosen examples match 9 in effectiveness (LangChain, 2024).
- Prompt format alone accounts for up to 40% performance variance (Voyce, 2025). Structure matters as much as content.
- Machine-readable formats (YAML headers, delimited Markdown sections) outperform prose for both human and agent consumption.
- Automated freshness checks eliminate stale-doc incidents entirely.
- Recency bias: the last example in a sequence has disproportionate influence. Structure docs so the most critical convention appears last in each section.

#### What Kanbanzai Already Does Well

This is another area of strong alignment:

- **Document-led workflow** is Kanbanzai's core design. Work flows through design → spec → dev-plan → review, each with its own draft lifecycle. Documents are registered with the system (`doc` tool) and tracked with structural metadata.
- **Document intelligence** (`internal/docint/`) provides structural analysis — outlines, section-level access, concept extraction, classification. Agents do not need to read entire documents to find relevant information.
- **Content hash tracking**: document records store content hashes. When a document changes, `doc(action: "refresh")` detects the change and can demote approved documents back to draft. This is a built-in staleness detection mechanism.
- **YAML frontmatter and structured formats**: entity records, knowledge entries, and document records all use structured YAML. The serialisation is deterministic and machine-readable.
- **ADR-equivalent decisions**: Kanbanzai's decision entity type (`entity(action: "create", type: "decision")`) captures the _why_ behind decisions, not just the _what_.

#### Actionable Improvements for Kanbanzai 3.0

1. **Automated freshness checks for context profiles and SKILLs.** Document records have content hashes and staleness detection. Context profiles (`.kbz/context/roles/`) and SKILLs (`.skills/`) do not. These files are operational context that agents read on every task. They should have the same freshness tracking as registered documents. The `health` tool could flag SKILLs and context profiles that have not been verified in 30 days.

2. **Canonical examples in SKILLs.** The research says 2–3 BAD vs. GOOD examples per convention are the most effective teaching mechanism. Our current SKILLs have procedural steps but limited examples. The code review SKILL has structured output format examples, but the per-dimension guidance sections lack concrete before/after code examples.

3. **Recency-aware section ordering in generated documents.** When the system generates documents (review reports, context packets, sub-agent prompts), it should place the most critical constraint or convention _last_ in each section to exploit recency bias.

4. **Documentation-code contradiction detection.** The health check (`health` tool) could cross-reference documented conventions (in AGENTS.md, context profiles, SKILLs) against the actual codebase. For example: if a convention says "always use `T[]`" but the ESLint config does not enforce it, surface the contradiction.

**Priority:** Medium. The foundation is strong. The improvements are about extending staleness detection to all operational context files and adding concrete examples to SKILLs.

---

### Principle 4: The Disposable Blueprint Principle

> *Never implement without a saved, versioned plan artifact. And never fall in love with one.*

#### Key Research Findings

- Teams using structured artifacts produce ~40% fewer errors than those using free dialogue (Hong et al., 2023 — MetaGPT).
- Plans externalized to files survive context resets perfectly — no degradation, no information loss.
- Planning and implementation are different cognitive modes. Separate them.
- Code is disposable; the blueprint is where intellectual capital lives.
- Sunk-cost attachment to failing code is the most expensive habit in agentic development.
- Plans reviewed before code catches ~80% of structural issues when they are free to fix.

#### What Kanbanzai Already Does Well

**This principle describes Kanbanzai's entire workflow model:**

- **Features travel a state machine** from proposal → specification → … → done. The spec document IS the versioned blueprint. It exists before any code is written.
- **Plans are first-class entities** with their own lifecycle. Implementation plans (`work/plan/`) are committed to Git alongside the code.
- **The decompose tool** (`decompose(action: "propose")`) produces structured task breakdowns from specifications — the plan artifact that implementation works against.
- **The "kill the branch, refine the blueprint" pattern** is exactly what Kanbanzai's worktree system enables. Each feature gets its own Git worktree. If the approach fails, the worktree can be abandoned and a new one created from the refined plan.
- **Structured plan artifacts**: features, tasks, decisions — all stored as YAML with defined fields, versioned in Git, and accessible via MCP tools.
- **Human gates at plan finalisation**: the checkpoint tool allows humans to review plans before implementation begins.

#### Actionable Improvements for Kanbanzai 3.0

1. **Plan review before implementation as a default gate.** The system supports this but does not enforce it. Consider making plan review a required stage gate in the feature lifecycle — the spec must be approved before the feature can transition to `developing`. This is already partially true (features go through `specification` state), but the gate could be made more explicit: "no implementation tasks can be created until the spec document is approved."

2. **Blueprint versioning with diff tracking.** When a plan is revised mid-implementation (which the research says is expected and healthy), the system should track the revision as a version, not as a replacement. The current document refresh mechanism detects changes but does not preserve the history of what changed. A "plan v1 → plan v2" diff would make it clear what was learned from a failed approach.

3. **"Time to restart" metric.** The research suggests tracking "time from 'this approach is wrong' to 'clean restart with a better plan'" as a key metric. The worktree system already captures some of this data (worktree creation timestamps, branch lifecycle). Surface it in the status dashboard.

**Priority:** Low. Kanbanzai is strongly aligned here. Improvements are refinements, not structural changes.

---

### Principle 5: The Institutional Memory Principle

> *When an agent makes a mistake, don't just correct it — codify it forever.*

#### Key Research Findings

- Negative constraints steer the model away from the generic centre of its training distribution toward project-specific output.
- **"Always/Never X BECAUSE Y"** is the optimal format. The BECAUSE clause is what makes rules generalisable to adjacent cases.
- Combined positive instruction + negative constraint is the strongest approach (CHI 2023, "Why Johnny Can't Prompt").
- Named anti-patterns activate expert knowledge clusters. Unnamed problems get generic responses.
- Rules without reasons become dead weight — they cannot be pruned because no one knows why they were added.
- At n=19 requirements, accuracy drops below n=5 (Vaarta Analytics, 2026). Keep the list lean.

#### What Kanbanzai Already Does Well

- **The knowledge system** (`internal/knowledge/`) is a direct implementation of institutional memory. Knowledge entries are contributed via `finish` or `knowledge(action: "contribute")`, stored as YAML, scoped by tier (project-level vs. session-level), and surfaced during context assembly.
- **Confidence scoring** (Wilson score) and **deduplication** (Jaccard similarity) ensure the knowledge base does not accumulate redundant or low-confidence entries.
- **Knowledge lifecycle**: entries go through contributed → confirmed → retired states. Disputed and stale entries can be flagged and pruned.
- **Retrospective signals**: the `finish` tool captures workflow friction, tool gaps, spec ambiguity, and other observations. The `retro` tool synthesises these into themed clusters.
- **Context profiles** include conventions with structured entries (the `base.yaml` profile has entries like "Spec is law: if code contradicts the specification, surface the conflict — do not resolve silently").

#### Actionable Improvements for Kanbanzai 3.0

1. **"Always/Never BECAUSE" format enforcement.** The knowledge system accepts free-form content. For entries tagged as conventions or constraints, the system could enforce (or at least encourage) the "Always/Never X BECAUSE Y" format. The BECAUSE clause is what the research identifies as the key mechanism — without it, rules cover one case instead of generalising.

2. **Proactive knowledge surfacing during context assembly.** Currently, knowledge entries are available via `knowledge(action: "list")` or assembled into context packets. The improvement: when context assembly encounters a file path or topic that matches a knowledge entry's scope, automatically include relevant always/never rules in the context packet. For example, if a task involves `internal/storage/`, automatically surface any knowledge entries about YAML serialisation conventions. Make the institutional memory _automatic_, not query-dependent.

3. **Anti-pattern naming convention.** The research says named anti-patterns ("the eager-loading trap," "the N+1 migration") activate expert knowledge clusters. Kanbanzai's knowledge entries have topics but not a structured "anti-pattern name" field. Adding this would make knowledge entries more powerful as routing signals.

4. **Lean knowledge auditing.** The research warns that at n=19 requirements, accuracy drops below n=5. If the knowledge base grows past a certain size per scope, the system should warn that context budget is being consumed and suggest compaction. The `knowledge(action: "compact")` and `knowledge(action: "prune")` tools exist but are not triggered automatically.

5. **Session-to-permanent promotion workflow.** When an agent corrects a mistake in a session, it should be trivial to promote that correction to a permanent knowledge entry. The `knowledge(action: "promote")` tool exists, but the workflow should be smoother: detect corrections in sub-agent output and suggest codification.

**Priority:** High. The infrastructure exists, but the improvements would make institutional memory proactive rather than reactive — automatically surfacing relevant rules instead of requiring agents to query for them.

---

### Principle 6: The Specialized Review Principle

> *A generalist reviewer trends toward the median. Specialists find what generalists can't.*

#### Key Research Findings

- **Vocabulary routing** (Ranjan et al., 2024): specific vocabulary acts as a routing signal that determines which knowledge clusters the model activates. "OWASP Top 10 audit, STRIDE threat model" routes to security engineering; "review the security" routes to blog posts.
- **PRISM persona science**: brief identities (<50 tokens) produce higher-quality outputs than elaborate personas (100+ tokens). Flattery degrades output by activating motivational/marketing text.
- **The 15-year practitioner test**: would a senior expert with 15+ years of domain experience use this exact term when talking with a peer? If yes, it is the right vocabulary.
- **Self-evaluation fails**: the generator shares the evaluator's biases. Separate generation from evaluation.
- **Deterministic checks first, LLM review second**: build, lint, test before sending to reviewers.

#### What Kanbanzai Already Does Well

- **Separation of generation from evaluation**: the code review SKILL explicitly separates the agent that wrote the code from the agents that review it. The orchestration procedure dispatches review sub-agents that have no prior context about how the code was written.
- **Structured review dimensions**: the reviewer profile defines five specific dimensions (spec conformance, implementation quality, test adequacy, documentation currency, workflow integrity). This is specialisation by dimension.
- **Evidence-based review**: the code review SKILL requires findings to cite specific requirements or conventions. "LGTM" is not an acceptable output.
- **Deterministic checks first**: the plan review SKILL runs `go test -race ./...` and `health()` before any LLM-based review.

#### Actionable Improvements for Kanbanzai 3.0

1. **Vocabulary payloads in context profiles.** The reviewer profile currently lists review dimensions but does not include domain-specific vocabulary terms. For a Go project, the reviewer profile should include terms like "goroutine leak detection," "interface segregation," "error wrapping with %w," "table-driven test patterns." For security review, it should include "OWASP Top 10," "CWE classifications," "input validation boundaries." This vocabulary routing is what the research identifies as the _primary_ quality lever.

2. **Domain-specific reviewer profiles.** Instead of one `reviewer.yaml`, create specialised profiles: `reviewer-security.yaml`, `reviewer-performance.yaml`, `reviewer-go-idioms.yaml`. Each with <50 tokens of identity, 15–30 domain vocabulary terms, and 5–10 named anti-patterns with detection signals. The orchestrator dispatches the appropriate specialist(s) based on the file types and concerns in each review unit.

3. **Anti-flattery convention.** Add an explicit convention to the base profile: "Define competence through domain vocabulary and named anti-patterns, not through adjectives or superlatives. Never use 'expert,' 'world-class,' or similar flattery in sub-agent prompts." This prevents a common pattern where orchestrators try to improve sub-agent quality through praise, which the research shows degrades output.

4. **Review rejection requirements.** The code review SKILL already requires evidence for findings, but could be strengthened: require each reviewer to either identify at least one issue OR explicitly justify clearance with specific evidence. "No security issues found" is not enough. "No hardcoded secrets detected; all user input passes through validation middleware in auth.go L23–47; SQL queries use parameterised statements throughout" is evidence. This directly counters FM-3.1 (rubber-stamp approval).

5. **Parallel specialist dispatch.** The code review orchestration already supports parallel sub-agents. Extend this so that the same review unit can be reviewed by multiple specialists in parallel (e.g., a Go idioms reviewer and a security reviewer reading the same files). Each finds different issues through different vocabulary lenses.

**Priority:** High. This is the most impactful improvement for review quality. Vocabulary routing is the single highest-leverage intervention identified in the research, and the current system does not implement it.

---

### Principle 7: The Observability Imperative

> *If you can't see inside your pipeline, you're trusting it on faith.*

#### Key Research Findings

- **MAST failure taxonomy**: 14 distinct failure modes in multi-agent systems across three categories:
  - **Communication failures**: message loss (FM-1.1), misinterpretation (FM-1.2), information overload (FM-1.3), stale context (FM-1.4)
  - **Coordination failures**: deadlock (FM-2.1), race conditions (FM-2.2), role confusion (FM-2.3), authority vacuum (FM-2.4), resource contention (FM-2.5)
  - **Quality failures**: rubber-stamp approval (FM-3.1), error cascading (FM-3.2), LCD output (FM-3.3), groupthink (FM-3.4), regression (FM-3.5)
- Most failure modes are **invisible without structured logging**.
- Structured artifact chains are debuggable; conversation logs are archaeology.
- Log at the boundaries: tool calls, LLM interactions, artifact handoffs, review actions.
- Always log both inputs and outputs — outputs without inputs make diagnosis impossible.

#### What Kanbanzai Already Does Well

- **Structured artifacts as audit trail**: every entity transition, knowledge contribution, document registration, and task completion is recorded as a YAML file in `.kbz/state/`. The sequence of state changes tells the story of what happened.
- **Health checks** (`health` tool) provide a project-wide consistency check — detecting orphaned tasks, stale knowledge, inconsistent entity state.
- **Review reports** (`work/reviews/`) are structured audit trails of what was found during review.
- **Retrospective signals** capture observations about workflow friction and tool gaps at task completion.
- **Decision records** capture why choices were made, not just what was decided.

#### Actionable Improvements for Kanbanzai 3.0

1. **Structured logging for sub-agent handoffs.** When an orchestrator dispatches a sub-agent via `spawn_agent`, log the handoff: what context was sent, what task was assigned, what the sub-agent returned. Currently, this happens in ephemeral agent sessions with no persistent record. Adding a `handoff_log` that records these exchanges would make debugging multi-agent workflows tractable.

2. **Review metrics tracking.** Track review metrics per review agent: approval rate, average time to review, number of findings per review. The research says an approval rate above 85% with review times under 5 seconds indicates rubber-stamping. These metrics should be queryable so orchestrators (and humans) can detect systematic review quality problems.

3. **MAST failure mode detection.** Implement automated detection for the highest-impact failure modes:
   - **FM-1.1 (Message Loss)**: when the orchestrator sends context to a sub-agent, hash the payload and verify the sub-agent acknowledges receipt.
   - **FM-3.1 (Rubber-Stamp Approval)**: if a review sub-agent returns a verdict of "approved" with zero findings in under a threshold time, flag it for the orchestrator.
   - **FM-3.2 (Error Cascading)**: track the provenance of each finding — which spec requirement, which source file, which review dimension. When a defect reaches production, trace it back through the artifact chain.

4. **Pipeline viewer for multi-agent workflows.** A diagnostic tool (MCP tool or CLI command) that reads the handoff logs and presents the artifact chain: "Orchestrator dispatched Task X to Sub-agent A with context hash abc123. Sub-agent A returned findings Y with hash def456. Orchestrator routed to remediation." This transforms debugging from archaeology to lookup.

5. **Tool call logging.** Log every MCP tool invocation with inputs, outputs, and duration. This is the "single highest-leverage observability investment" per the research. The MCP server handles all tool calls — adding structured logging at this layer would capture every agent action without requiring changes to individual tools.

**Priority:** Medium-High. The structured artifact trail is strong, but the sub-agent layer is a black box. Adding handoff logging and review metrics would close the most critical observability gap.

---

### Principle 8: The Strategic Human Gate Principle

> *Rubber-stamp approval is the single most common quality failure in multi-agent systems.*

#### Key Research Findings

- **FM-3.1 (Rubber-Stamp Approval)** is the most frequently observed quality failure in the MAST taxonomy. LLMs are sycophantic — they default to agreement.
- **FM-3.4 (Groupthink)**: multiple agents sharing the same base model share blind spots. More agents ≠ more perspectives.
- **Alignment-accuracy tradeoff** (PRISM): stronger personas improve instruction-following but can reduce factual accuracy.
- Strategic placement at 2–3 decision points beats comprehensive coverage at every step. Too many gates → human becomes the bottleneck. Too few → mistakes propagate.
- Gates should be **low-friction** (one-key approval) and **high-information** (agent presents summary + risks).
- A gate with 0% rejection rate is decorative, not functional. Healthy range: 5–20%.

#### What Kanbanzai Already Does Well

**This is one of Kanbanzai's strongest areas:**

- **The checkpoint tool** (`checkpoint(action: "create")`) is a direct implementation of strategic human gates. It creates a decision point with context, presents it to the human, and blocks further work until the human responds.
- **Feature lifecycle gates**: features must transition through proposal → specification → developing → reviewing → done. The reviewing stage is a built-in human gate.
- **Document approval gates**: spec documents must be approved before features can advance. This is the "review plans before code" pattern the research recommends.
- **The code review SKILL** explicitly defines when to create checkpoints: ambiguous findings, high-stakes features, disagreement between dimensions.
- **Low-friction design**: the checkpoint tool presents structured context (verdicts, risks, recommendations) so humans can make informed decisions quickly.

#### Actionable Improvements for Kanbanzai 3.0

1. **Gate rejection rate tracking.** Track the approval/rejection rate at each gate. If a gate has 0% rejections over 30+ decisions, surface this as a warning in the health check. This directly implements the research's recommendation to monitor gate effectiveness.

2. **Confidence-adaptive gating.** When the orchestrator has high confidence (well-understood task, clear spec, all tests pass), the gate could be lighter-weight — presenting a summary and allowing one-key approval. When confidence is low (novel pattern, ambiguous spec, partial test coverage), the gate should require more engagement — presenting specific risks and asking the human to acknowledge each one. The system already supports this via the checkpoint context field, but it could be formalised as a policy.

3. **Gate placement guidance in decomposition.** When the `decompose` tool creates a task breakdown, it could suggest where human gates should be placed based on the blast radius and reversibility of each task. The research's decision matrix (blast radius × cost to reverse) could be encoded as a heuristic.

4. **Anti-sycophancy in review prompts.** Add explicit anti-sycophancy instructions to the reviewer profile: "If you find zero issues, you must provide specific evidence for each dimension showing what you checked and why it passed. A clean review with no evidence is treated as a rubber stamp." This structural requirement is the direct counter to FM-3.1.

**Priority:** Medium. The infrastructure is strong. The improvements are about making gate effectiveness measurable and adding adaptive gating based on confidence.

---

### Principle 9: The Token Economy Principle

> *Tokens are money. Most people are burning it.*

#### Key Research Findings

- **DeepMind scaling data (2025)**: a 5-agent team costs 7x tokens but produces only 3.1x output (efficiency ratio: 0.44). At 7+ agents, output often degrades below a 4-agent team at 12x cost.
- **The 45% threshold**: if a single well-prompted agent achieves >45% of optimal performance, adding more agents yields diminishing returns.
- **Team size saturates at 4.** Diminishing returns start at 3.
- **Sequential reasoning tasks degrade 39–70% in multi-agent setups** — they should stay with single agents.
- **Cascade escalation**: Level 0 (single agent + tools) → Level 1 (worker + reviewer) → Level 2 (3–5 agent team) → Level 3 (multi-team with coordinator). Never skip levels.
- **Adaptive team composition** (Captain Agent, 2024) outperforms static teams by 15–25%.
- **Context loading costs are invisible but real**: every idle tool degrades attention on active tools.

#### What Kanbanzai Already Does Well

- **The `handoff` tool** assembles minimal context per task — it does not dump the entire project state.
- **Context profiles** are task-specific: a reviewer gets reviewer context, a developer gets developer context. Not everyone gets everything.
- **The code review orchestration** uses adaptive sub-agent dispatch: the orchestrator decomposes a review into 2–5 units and dispatches sub-agents only for the units that exist. A 3-file feature gets fewer sub-agents than a 30-file feature.
- **The `conflict` tool** checks for file overlap before dispatching parallel tasks — this avoids the coordination overhead of resolving conflicts after they occur.

#### Actionable Improvements for Kanbanzai 3.0

1. **Cascade escalation as a policy.** Encode the cascade pattern into the orchestration tools. When a human or orchestrator requests a multi-agent workflow, the system should first attempt Level 0 (single agent with tools). Only if that demonstrably fails should it escalate to Level 1, then Level 2. The `decompose` tool could include a "minimum viable team" recommendation alongside the task breakdown.

2. **Token budget tracking.** Add token cost estimation to the `status` dashboard for features and plans. "This feature used approximately X tokens across Y agent sessions." This makes the invisible cost visible and allows comparison: "Feature A cost 3x more tokens than Feature B of similar complexity — why?"

3. **Team size cap as a policy.** Encode a hard cap: no single feature uses more than 4 concurrent sub-agents without explicit human override (via checkpoint). The research says saturation occurs at 4 and degradation begins past that.

4. **Adaptive MCP tool loading.** This is the `jig`-equivalent for Kanbanzai: when assembling context for a sub-agent, include only the MCP tool definitions the sub-agent will actually need. A review sub-agent does not need the `decompose` tool. An implementation sub-agent does not need the `retro` tool. Currently, all tools are exposed to all agents regardless of their role.

5. **Single-agent-first recommendation.** When the `decompose` tool proposes a task breakdown, include an assessment of whether each task is simple enough for a single agent pass. Tag tasks as "single-agent" vs. "requires coordination." This makes the 45% threshold operationally concrete.

**Priority:** High. Token economy has direct cost implications. The cascade pattern and adaptive tool loading are the highest-value interventions.

---

### Principle 10: The Toolkit Principle

> *Knowledge without automation decays. Encode your principles into tools that enforce them automatically.*

#### Key Research Findings

This is the capstone — Principle 1 (Hardening) applied recursively to the process of building AI tools:

- **Skill architecture** should follow an attention-optimized structure:
  1. YAML frontmatter with dual-register description (~100 tokens)
  2. Expert Vocabulary Payload (FIRST in body — the routing signal)
  3. Anti-Pattern Watchlist (BEFORE behavioural instructions)
  4. Behavioural Instructions (ordered imperative steps)
  5. Output Format
  6. Examples (2–3 BAD vs. GOOD pairs)
  7. "Questions This Skill Answers" (at END — retrieval anchors)
- **Vocabulary routing** is the primary quality lever — 15–30 precise domain terms per skill.
- **Brief identities** (<50 tokens) using real job titles outperform elaborate personas.
- **Right-altitude prompting**: at n=19 requirements, accuracy drops below n=5. Balance between over-specification and under-specification.
- **Dual-register descriptions**: expert terminology for routing depth + natural language for trigger breadth.
- **Named anti-patterns** with detect → name → explain → resolve → prevent pattern.
- **Separated generation and evaluation** with weighted, gradable evaluation criteria.

#### What Kanbanzai Already Does Well

- **The SKILL system** (`.skills/`) already encodes procedures as reusable, versionable artifacts. The README defines clear criteria for when to create a SKILL and how to structure it.
- **Context profiles** encode role conventions and architectural knowledge.
- **The MCP tool surface** is itself a toolkit — 22+ tools that enforce workflow patterns deterministically.
- **AGENTS.md** serves as the project-wide configuration that every agent reads.

#### Actionable Improvements for Kanbanzai 3.0

1. **Restructure SKILLs to follow the research-backed architecture.** The current SKILL format (Purpose → When to Use → Procedure → Common Issues → Verification → Related) should be revised to:
   - YAML frontmatter with dual-register description
   - Expert Vocabulary Payload (first in body)
   - Anti-Pattern Watchlist
   - Procedure (imperative steps with IF/THEN conditions)
   - Output Format with examples
   - 2–3 BAD vs. GOOD canonical examples
   - "Questions This Skill Answers" (last in body, for retrieval anchoring)

2. **Vocabulary payloads for each SKILL.** Each SKILL should define 15–30 domain-specific terms that activate the right knowledge clusters. The code review SKILL should include terms like "cyclomatic complexity," "defensive copying," "invariant assertion," "contract violation." The plan review SKILL should include "acceptance criteria traceability," "scope creep detection," "dependency graph validation."

3. **SKILL generation tool.** A meta-SKILL (or MCP tool) that generates new SKILLs following the research-backed architecture. This is the Hardening Principle applied to SKILL creation: instead of relying on agents to manually follow the SKILL format, provide a deterministic template that enforces the structure.

4. **Profile-aware tool filtering.** Extend context profiles to declare which MCP tools are relevant for each role. When assembling context for a `reviewer` role, only include the tool definitions for tools the reviewer will use. This is the `jig` pattern — minimal context loading per session type.

5. **Gradable evaluation criteria in SKILLs.** The research recommends separating generation criteria from evaluation criteria, with evaluation phrased as gradable questions. "Can the reviewer identify the most critical finding in under 10 seconds?" is testable. "Is the review good?" is not. Add an evaluation criteria section to SKILLs that enables quality assessment of SKILL outputs.

**Priority:** High. SKILL architecture and vocabulary routing are the primary vehicles for improving agent output quality. This is where the research's recommendations translate most directly into Kanbanzai design changes.

---

## Cross-Cutting Themes

### Theme 1: Vocabulary Routing Is the Primary Quality Lever

The single most impactful finding across the research is that **the words you use in prompts determine which knowledge the model accesses**. This appears in Principle 6 (Specialized Review), Principle 10 (Toolkit), and is supported by Ranjan et al. (2024) and the PRISM framework.

Kanbanzai currently does not implement vocabulary routing. Context profiles contain conventions (what to do) but not vocabulary payloads (the domain-specific terms that route the model to expert knowledge clusters). Adding vocabulary payloads to context profiles and SKILLs is likely the single highest-ROI improvement for Kanbanzai 3.0.

### Theme 2: Position and Structure Matter as Much as Content

The U-shaped attention curve (Liu et al., 2024; Wu et al., 2025), the 40% performance variance from format alone (Voyce, 2025), and the recency bias finding all converge on the same conclusion: **how you structure context is as important as what you put in it.**

Kanbanzai assembles structured context packets, but does not control the ordering of content within those packets. Implementing position-aware context assembly — hard constraints first, instructions last, supporting material in the middle — would improve agent performance on every task without changing the content.

### Theme 3: Make the Invisible Visible

The MAST failure taxonomy, the token economy data, and the observability imperative all share a common message: **the most dangerous problems are the ones you cannot see.** Rubber-stamp approvals, token waste, stale context, message loss between sub-agents — all invisible without explicit instrumentation.

Kanbanzai has strong structural observability (entity state, document records, health checks) but weak operational observability (sub-agent handoffs, review metrics, token costs). Closing this gap would make systemic quality problems detectable before they cause damage.

### Theme 4: Less Is More — But Structured Less

The research consistently finds that less content, better structured, outperforms more content loosely organised:

- 15–40% context utilisation beats 60%+
- <50 token identities beat 200+ token personas
- 3 examples match 9
- 5 well-chosen requirements beat 19
- 4 agents beat 7+
- Cascade escalation beats start-at-maximum

Kanbanzai's design philosophy is already "the minimum context needed for each task." The improvement is making this philosophy measurable and enforceable — token budget estimation, team size caps, context loading audits.

### Theme 5: Harden What Repeats, Keep Fuzzy What Varies

The Hardening Principle (P1) and the Toolkit Principle (P10) bracket the series with the same message: **deterministic tools for mechanical work, LLMs for fuzzy reasoning.** This maps directly to Kanbanzai's architecture — MCP tools for workflow mechanics, agents for reasoning.

The opportunity is to continue pushing the boundary: harden decomposition validation, review finding classification, context budget estimation, and SKILL template generation into deterministic tools. Keep summarisation, creative decomposition, intent interpretation, and nuanced review judgment with the LLMs.

---

## Consolidated Recommendations for Kanbanzai 3.0

### Priority 1: High Impact, Strong Research Backing

| # | Recommendation | Source Principles | Effort |
|---|----------------|-------------------|--------|
| R1 | **Add vocabulary payloads to context profiles and SKILLs** — 15–30 domain-specific terms per role/skill following the 15-year practitioner test | P6, P10 | Medium |
| R2 | **Restructure SKILLs to follow attention-optimized architecture** — vocabulary first, anti-patterns before instructions, retrieval anchors last | P2, P10 | Medium |
| R3 | **Position-aware context assembly** — enforce ordering in assembled context packets: identity/constraints first, supporting material middle, instructions/anchors last | P2 | Medium |
| R4 | **Proactive knowledge surfacing** — automatically include relevant always/never rules in context packets based on task scope, not just on explicit query | P5 | Medium |
| R5 | **Domain-specific reviewer profiles** — replace single `reviewer.yaml` with specialised profiles (security, performance, idioms) each with vocabulary routing | P6 | Medium |
| R6 | **Cascade escalation policy** — encode the single-agent-first pattern into orchestration; recommend minimum viable team in decomposition | P9 | Low–Medium |
| R7 | **Adaptive MCP tool filtering** — expose only relevant tool definitions to each sub-agent role, not the full tool surface | P2, P9 | Medium |

### Priority 2: Valuable, Moderate Effort

| # | Recommendation | Source Principles | Effort |
|---|----------------|-------------------|--------|
| R8 | **Structured handoff logging** — log sub-agent dispatches with context hash, task, and return value for debugging multi-agent workflows | P7 | Medium |
| R9 | **Review metrics tracking** — track approval rate, finding count, and review duration per reviewer to detect rubber-stamping | P7, P8 | Low–Medium |
| R10 | **Context budget estimation** — estimate token count of context packets and warn when exceeding the 40% utilisation threshold | P2, P9 | Low |
| R11 | **"Always/Never BECAUSE" format for knowledge entries** — encourage or enforce the most effective constraint format for convention entries | P5 | Low |
| R12 | **Anti-sycophancy instructions in reviewer profiles** — require evidence-backed clearance justification, not just "no issues found" | P6, P8 | Low |
| R13 | **SKILL generation meta-SKILL** — a tool that creates new SKILLs following the research-backed architecture automatically | P10 | Medium |
| R14 | **Gate rejection rate monitoring** — track and report approval/rejection rates; flag gates with 0% rejection as potentially decorative | P8 | Low |

### Priority 3: Incremental Improvements

| # | Recommendation | Source Principles | Effort |
|---|----------------|-------------------|--------|
| R15 | **Freshness tracking for SKILLs and context profiles** — extend document staleness detection to all operational context files | P3 | Low |
| R16 | **Canonical BAD/GOOD examples in SKILLs** — add 2–3 concrete before/after examples to each SKILL dimension | P3, P10 | Low |
| R17 | **Token cost tracking in status dashboard** — surface approximate token usage per feature/plan | P9 | Medium |
| R18 | **Team size cap policy** — enforce max 4 concurrent sub-agents per feature without explicit human override | P9 | Low |
| R19 | **MAST failure mode detection** — automated detection for FM-1.1 (message loss) and FM-3.1 (rubber-stamp) in sub-agent workflows | P7, P8 | Medium–High |
| R20 | **Blueprint revision tracking** — when plans are revised mid-implementation, track the diff between versions | P4 | Low |

---

## What We Are Already Doing Right

It is worth explicitly recording the areas where Kanbanzai's existing design aligns with the research, so these design decisions are not inadvertently changed:

| Research Finding | Kanbanzai Implementation | Status |
|------------------|--------------------------|--------|
| Deterministic tools for mechanical work (P1) | MCP server handles all workflow mechanics; agents do reasoning | ✅ Strong |
| Externalized, versioned plan artifacts (P4) | Feature specs, plans, and decisions stored as YAML in Git | ✅ Strong |
| Structured artifacts reduce errors ~40% (P4, P7) | All entities are structured YAML with defined schemas | ✅ Strong |
| Separate generation from evaluation (P6) | Review sub-agents are separate from implementation agents | ✅ Strong |
| Evidence-based review, not "LGTM" (P6, P8) | Code review SKILL requires findings with citations | ✅ Strong |
| Human gates at high-stakes decisions (P8) | Checkpoint tool, document approval gates, feature lifecycle | ✅ Strong |
| Progressive disclosure / minimal context (P2) | Context profiles with inheritance; `handoff` assembles focused packets | ✅ Good |
| Institutional memory system (P5) | Knowledge entries with lifecycle, scoping, dedup, confidence | ✅ Good |
| Document freshness detection (P3) | Content hash tracking; refresh detects stale docs | ✅ Good |
| Deterministic checks before LLM review (P6) | Plan review runs `go test` and `health()` before LLM analysis | ✅ Good |
| Retrospective capture (P5, P7) | Retro signals captured at task completion; synthesised by `retro` | ✅ Good |
| Adaptive sub-agent dispatch (P9) | Code review orchestration adapts sub-agent count to feature size | ✅ Partial |

---

## Conclusion

The 10 Claude Code Principles, grounded in 17 peer-reviewed sources, validate many of Kanbanzai's core design decisions while identifying specific, high-leverage improvements.

The three highest-impact changes for Kanbanzai 3.0 are:

1. **Vocabulary routing** (R1, R5) — adding domain-specific vocabulary payloads to context profiles and SKILLs is the single most impactful change. It determines which knowledge the model accesses and is the primary lever for output quality.

2. **Position-aware context assembly** (R3) — enforcing the attention-optimal ordering (constraints first, instructions last) in every assembled context packet would improve agent performance on every task.

3. **Proactive institutional memory** (R4) — automatically surfacing relevant always/never rules during context assembly, rather than requiring agents to query the knowledge base, would make accumulated project knowledge work harder.

These three changes together address the research's central insight: **the words you choose, where you place them, and what rules you surface are more important than how many agents you deploy or how long your prompts are.**

---

## Appendix: MAST Failure Taxonomy Quick Reference

For use in designing detection mechanisms and review anti-patterns.

### Communication Failures
- **FM-1.1 Message Loss** — Agent output not received by next agent. Detect: hash handoff payloads.
- **FM-1.2 Misinterpretation** — Instruction understood differently than intended. Detect: require structured formats.
- **FM-1.3 Info Overload** — Too much context degrades performance. Detect: monitor token counts.
- **FM-1.4 Stale Context** — Outdated information in active context. Detect: timestamp all context.

### Coordination Failures
- **FM-2.1 Deadlock** — Agents waiting on each other. Detect: timeout all operations.
- **FM-2.2 Race Condition** — Output depends on execution order. Detect: sequence-number handoffs.
- **FM-2.3 Role Confusion** — Overlapping or unclear responsibilities. Detect: single-responsibility identities.
- **FM-2.4 Authority Vacuum** — No agent empowered to decide. Detect: explicit decision owners.
- **FM-2.5 Resource Contention** — Multiple agents modifying same resource. Detect: lock or queue shared resources.

### Quality Failures
- **FM-3.1 Rubber-Stamp Approval** — Reviewer approves without scrutiny. Detect: require specific citations.
- **FM-3.2 Error Cascading** — One agent's mistake amplified downstream. Detect: validate at each handoff.
- **FM-3.3 LCD Output** — Lowest-common-denominator quality. Detect: domain-specific quality bars.
- **FM-3.4 Groupthink** — Agents converge on same approach. Detect: diversity in specialist prompts.
- **FM-3.5 Regression** — Later agent undoes earlier agent's work. Detect: immutable completed sections.

---

## Appendix: Recommended Skill Architecture Template

Based on the research-backed framework from Principle 10:

```
skill-name/
├── SKILL.md (<500 lines)
│   ├── YAML frontmatter (name + dual-register description, ~100 words)
│   ├── Expert Vocabulary Payload (FIRST in body — routing signal)
│   ├── Anti-Pattern Watchlist (BEFORE behavioural instructions)
│   ├── Behavioural Instructions (ordered imperative steps with IF/THEN)
│   ├── Output Format
│   ├── Examples (2–3 BAD vs GOOD pairs)
│   └── Questions This Skill Answers (at END — 8–15 retrieval queries)
└── references/
    ├── anti-patterns-full.md
    ├── frameworks.md
    ├── evaluation-criteria.md
    └── checklists.md
```

**Section ordering rationale (attention curve):**
- **Top** (high attention): Vocabulary payload — the routing signal that determines which knowledge clusters activate.
- **Middle** (lower attention): Behavioural instructions — survive attention degradation because structured as numbered steps.
- **Bottom** (high attention): Retrieval anchors — benefit from recency bias and end-of-context attention.