# Research Report: Agent Skills Architecture for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-14 |
| Author | Research Agent |
| Status | Draft |
| Scope | Skills system redesign research for Kanbanzai 3.0 |
| Sources | Anthropic Platform Documentation (Agent Skills Overview, Quickstart, Best Practices), Anthropic Guardrails Guides (Increase Consistency, Reduce Hallucinations), anthropics/skills repository (skill-creator), Prior Kanbanzai research (ai-agent-best-practices-research.md) |

---

## Executive Summary

This report analyses Anthropic's official Agent Skills documentation, guardrails guidance, and the open-source `skill-creator` meta-skill, then evaluates how each finding applies to the Kanbanzai skills system. The goal is to identify concrete, actionable improvements for Kanbanzai 3.0.

**Key findings:**

1. **Progressive disclosure is the architectural foundation** — Anthropic's three-level loading model (metadata → instructions → resources) is the single most important structural concept. Kanbanzai's current skills load everything at once via `AGENTS.md` references or context assembly, missing this entirely.

2. **Descriptions are the primary triggering mechanism** — Anthropic emphasises that skill descriptions should be "a little bit pushy" and include both *what* the skill does and *when* to use it. Our existing descriptions are good but discovery still fails because agents don't encounter them at the right moment.

3. **Conciseness is a design constraint, not a preference** — The official guidance says "Claude is already very smart — only add context Claude doesn't already have." Our skills over-explain common knowledge and under-explain Kanbanzai-specific conventions. This is backwards.

4. **Degrees of freedom should match task fragility** — Anthropic's "narrow bridge vs. open field" analogy maps directly to our problem: workflow stage gates need low freedom (exact steps), while design work needs high freedom (general guidance). We currently apply medium freedom uniformly.

5. **Feedback loops and checklists dramatically improve adherence** — The best practices guide shows that "validate → fix → repeat" loops and copy-paste checklists prevent step-skipping. This directly addresses our observed problem of agents rushing to implementation.

6. **The skill-creator meta-skill demonstrates evaluation-driven development** — Creating skills by writing tests first, running them, and iterating based on observed behavior is the gold standard. We have no skill evaluation process.

7. **Consistency techniques from the guardrails guides are directly applicable** — Specifying output formats, constraining with examples, using retrieval for contextual consistency, and chaining prompts for complex tasks all map to specific Kanbanzai improvements.

---

## Table of Contents

1. [Source Analysis: Agent Skills Overview](#1-source-analysis-agent-skills-overview)
2. [Source Analysis: Skills Quickstart](#2-source-analysis-skills-quickstart)
3. [Source Analysis: Skill Authoring Best Practices](#3-source-analysis-skill-authoring-best-practices)
4. [Source Analysis: Increase Output Consistency](#4-source-analysis-increase-output-consistency)
5. [Source Analysis: Reduce Hallucinations](#5-source-analysis-reduce-hallucinations)
6. [Source Analysis: The Skill-Creator Meta-Skill](#6-source-analysis-the-skill-creator-meta-skill)
7. [Cross-Cutting Themes](#7-cross-cutting-themes)
8. [Gap Analysis: Current Kanbanzai Skills vs. Best Practices](#8-gap-analysis-current-kanbanzai-skills-vs-best-practices)
9. [Recommendations](#9-recommendations)

---

## 1. Source Analysis: Agent Skills Overview

**Source:** https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview

### What the document says

The overview establishes the foundational architecture for Agent Skills:

- **Skills are modular capabilities** that package instructions, metadata, and optional resources. They are filesystem-based directories containing a `SKILL.md` file with YAML frontmatter, plus optional scripts, reference files, and assets.

- **Three-level progressive disclosure** is the core architectural pattern:
  - **Level 1 (Metadata):** Always loaded at startup. Only `name` and `description` from YAML frontmatter. ~100 tokens per skill. This is how Claude discovers skills.
  - **Level 2 (Instructions):** Loaded when the skill is triggered. The body of `SKILL.md`. Under 5k tokens.
  - **Level 3 (Resources):** Loaded as needed. Additional markdown files, scripts, reference materials. Effectively unlimited because they don't enter context until accessed.

- **Token cost is tiered** — metadata is cheap (always present), instructions are moderate (on-demand), resources are free until accessed.

- **Skills leverage filesystem access** — Claude reads files via bash. This means scripts can execute without their code entering the context window. Only script *output* consumes tokens.

- **Description is the discovery mechanism** — Claude uses the `name` and `description` to decide whether to trigger a skill. The description must include both what the skill does and when to use it.

### Application to Kanbanzai

**Progressive disclosure is the biggest architectural gap.** Our current skills system has two delivery mechanisms, neither of which implements progressive disclosure:

1. **`.agents/skills/kanbanzai-*/SKILL.md`** — These are discovered by Claude Code's built-in skill scanning and their metadata is loaded into the system prompt. This is correct for Level 1. But when triggered, the entire SKILL.md body loads at once with no further progressive disclosure. There are no Level 3 reference files.

2. **Context assembly via `handoff`/`next`** — When Kanbanzai assembles context for a task, it includes relevant skill content inline in the context packet. This bypasses the progressive disclosure model entirely — everything is loaded at once, competing for attention.

**Concrete implications:**

- We should restructure skills to use reference files for detailed procedures, keeping SKILL.md as a routing document.
- Workflow stage gate details, template examples, and anti-pattern lists should be in separate files referenced from SKILL.md, not inline.
- Scripts for deterministic operations (e.g., validating document structure, checking lifecycle prerequisites) should be bundled as executable resources, not described as procedures for the agent to follow.

**Token budget reality check:** With 6 kanbanzai skills at ~100 tokens metadata each, Level 1 costs ~600 tokens. This is acceptable. But if SKILL.md bodies average 2,000 tokens and multiple skills trigger simultaneously, Level 2 costs 4,000–10,000 tokens competing with conversation history and task context. This is where progressive disclosure into Level 3 reference files matters most.

---

## 2. Source Analysis: Skills Quickstart

**Source:** https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart

### What the document says

The quickstart is primarily a tutorial for using pre-built API skills (PowerPoint, Excel, Word, PDF). While the specific use case is different from Kanbanzai, the tutorial reveals important architectural details:

- **Skills are specified per-request** via the `container.skills` parameter. This means skill availability can be scoped to specific interactions.
- **Claude automatically matches tasks to relevant skills** based on the description. The system prompt contains skill metadata, and Claude decides which skill to trigger based on the user's request.
- **Progressive disclosure is automatic** — Claude determines the skill is relevant, loads its full instructions, then executes.

### Application to Kanbanzai

**Per-request skill scoping is relevant to our context assembly model.** When Kanbanzai assembles a context packet via `handoff` or `next`, it could include only the skill metadata relevant to the current workflow stage, rather than all skills. This mirrors the API's `container.skills` approach.

**The automatic matching model has implications for our discovery problem.** Our agents fail to discover skills not because the skills don't exist, but because:
- The agent doesn't encounter the description at the right moment
- The description doesn't match the agent's current framing of the task
- Multiple skills could apply and the agent picks none

The quickstart confirms that **description quality is the primary lever for discovery**. This is reinforced by the skill-creator's "description optimization" workflow (analysed in §6).

---

## 3. Source Analysis: Skill Authoring Best Practices

**Source:** https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices

This is the most actionable source. It provides specific, tested guidance for writing effective skills.

### 3.1 Core Principle: Conciseness

> "The context window is a public good."

The guide's central message is that every token in a skill must justify its presence. The default assumption is that **Claude is already very smart** — you should only add context Claude doesn't already have.

**The litmus test for every piece of content:**
- "Does Claude really need this explanation?"
- "Can I assume Claude knows this?"
- "Does this paragraph justify its token cost?"

**Application to Kanbanzai:** Our skills contain significant amounts of explanation that Claude already knows. For example, the `kanbanzai-workflow` skill explains what a "state machine" is and what "lifecycle transitions" mean. Claude knows these concepts. What Claude doesn't know is *Kanbanzai's specific* state machine, transitions, and rules. We should strip general explanations and focus entirely on Kanbanzai-specific knowledge.

### 3.2 Degrees of Freedom

The guide introduces a spectrum from high to low freedom:

| Freedom Level | When to use | Example |
|---|---|---|
| **High** (text instructions) | Multiple approaches valid, context-dependent | Code review process |
| **Medium** (pseudocode/templates) | Preferred pattern exists, some variation OK | Report generation |
| **Low** (exact scripts, no params) | Fragile operations, consistency critical | Database migrations |

The analogy: **narrow bridge (low freedom)** vs. **open field (high freedom)**.

**Application to Kanbanzai:** This maps directly to our workflow stages:

| Kanbanzai Stage | Appropriate Freedom | Why |
|---|---|---|
| Lifecycle transitions | **Low** | Wrong transitions are expensive to undo. Use exact tool calls. |
| Document registration | **Low** | Exact procedure, must be followed precisely. |
| Stage gate checks | **Low** | Deterministic — should be a script or tool, not instructions. |
| Specification writing | **Medium** | Template exists, but content varies by feature. |
| Design work | **High** | Creative, context-dependent, many valid approaches. |
| Code review | **Medium** | Structured checklist, but judgment calls on severity. |
| Implementation | **High** | Context-dependent, many valid approaches. |

Our current skills apply **uniform medium freedom** to everything. We should differentiate.

### 3.3 Naming Conventions

The guide recommends **gerund form** (verb + -ing) for skill names: `processing-pdfs`, `analyzing-spreadsheets`, `testing-code`.

**Application to Kanbanzai:** Our current naming uses `kanbanzai-{noun}` (`kanbanzai-workflow`, `kanbanzai-agents`, `kanbanzai-documents`). The gerund form would be:

| Current | Gerund Form | Assessment |
|---|---|---|
| `kanbanzai-workflow` | `managing-workflow` | Clearer activity |
| `kanbanzai-agents` | `dispatching-work` | More specific |
| `kanbanzai-documents` | `managing-documents` | Clearer activity |
| `kanbanzai-design` | `authoring-designs` | More specific |
| `kanbanzai-planning` | `creating-plans` | More specific |
| `kanbanzai-getting-started` | `getting-started` | Already gerund |

The gerund form communicates *activity*, which helps Claude match the skill to the current task. However, the `kanbanzai-` prefix serves as a namespace that groups our skills visually. **Recommendation:** evaluate whether the namespace prefix is worth the clarity cost.

### 3.4 Writing Effective Descriptions

Key guidance:
- **Always write in third person.** The description is injected into the system prompt. Inconsistent point-of-view causes discovery problems.
- **Be specific and include key terms.** Claude uses the description to choose from potentially 100+ available skills.
- **Include both what the skill does and when to use it.**

The guide provides this effective example:
> "Extract text and tables from PDF files, fill forms, merge documents. Use when working with PDF files or when the user mentions PDFs, forms, or document extraction."

**Application to Kanbanzai:** Our existing descriptions are actually quite good on this front — they include both purpose and trigger conditions. For example, the `kanbanzai-workflow` description explicitly lists trigger scenarios. However, the skill-creator analysis (§6) reveals that descriptions should be "a little bit pushy" to combat under-triggering. We should audit our descriptions against this standard.

### 3.5 Progressive Disclosure Patterns

The guide provides three concrete patterns:

**Pattern 1: High-level guide with references**
```
# PDF Processing
## Quick start
[basic usage]
## Advanced features
**Form filling**: See [FORMS.md](FORMS.md) for complete guide
**API reference**: See [REFERENCE.md](REFERENCE.md) for all methods
```

**Pattern 2: Domain-specific organization**
```
bigquery-skill/
├── SKILL.md (overview and navigation)
└── reference/
    ├── finance.md
    ├── sales.md
    └── product.md
```

**Pattern 3: Conditional details**
```
## Creating documents
Use docx-js for new documents. See [DOCX-JS.md](DOCX-JS.md).
## Editing documents
For simple edits, modify the XML directly.
**For tracked changes**: See [REDLINING.md](REDLINING.md)
```

**Critical rule:** **Keep references one level deep from SKILL.md.** Claude may partially read files that are referenced from other referenced files. All reference files should link directly from SKILL.md.

**Application to Kanbanzai:** This directly informs how we should restructure our skills:

```
kanbanzai-workflow/
├── SKILL.md (stage overview, routing, quick reference)
├── stage-gates.md (detailed gate requirements)
├── transitions.md (valid transition table, tool calls)
└── anti-patterns.md (common mistakes with BECAUSE clauses)
```

Each stage or topic gets its own reference file. SKILL.md serves as the router that tells Claude where to look. Claude loads only what's needed for the current task.

### 3.6 Workflows and Feedback Loops

The guide emphasises two patterns:

**Checklists for complex tasks:**
> "For particularly complex workflows, provide a checklist that Claude can copy into its response and check off as it progresses."

Example:
```
Copy this checklist and track your progress:
- [ ] Step 1: Read all source documents
- [ ] Step 2: Identify key themes
- [ ] Step 3: Cross-reference claims
```

**Feedback loops:**
> "Run validator → fix errors → repeat. This pattern greatly improves output quality."

**Application to Kanbanzai:** This is directly relevant to our observed problem of agents skipping workflow steps. If each workflow stage provides a **copy-paste checklist** that the agent must work through, it becomes much harder to skip steps. The agent has a visible, trackable record of what it has and hasn't done.

For stage gates specifically, a feedback loop pattern is appropriate:
1. Check prerequisites → if not met, address them → check again
2. Only proceed when all prerequisites pass

This should be implemented as a **tool call** (deterministic check via the MCP server), not as instructions the agent follows. This is the Hardening Principle from our previous research.

### 3.7 Content Guidelines

**Avoid time-sensitive information.** Use "current method" and "old patterns" sections instead of dates.

**Use consistent terminology.** Choose one term and use it throughout. The guide gives this example:
- Good: Always "API endpoint", always "field", always "extract"
- Bad: Mix "API endpoint"/"URL"/"API route"/"path"

**Application to Kanbanzai:** Terminology consistency is a known issue. Our documents sometimes use "feature" and "capability" interchangeably, or "task" and "work item." The vocabulary routing recommendation from our previous research is reinforced here — a fixed vocabulary list per skill prevents terminology drift.

### 3.8 Template and Examples Patterns

**Templates** define output format. Match strictness to requirements:
- Strict: "ALWAYS use this exact template structure"
- Flexible: "Here is a sensible default format, but use your best judgment"

**Examples** (input/output pairs) help Claude understand desired style better than descriptions alone.

**Application to Kanbanzai:** Our specification and plan-writing skills should include template patterns with appropriate strictness levels. Specifications need strict templates (required sections, format). Design documents need flexible templates (suggested structure, adapt as needed). We currently provide neither consistently.

### 3.9 Evaluation and Iteration

The guide advocates **evaluation-driven development:**
1. Identify gaps: Run Claude on tasks without a skill, document failures
2. Create evaluations: Build 3+ scenarios that test the gaps
3. Establish baseline: Measure performance without the skill
4. Write minimal instructions: Just enough to address gaps
5. Iterate: Execute evaluations, compare, refine

**Application to Kanbanzai:** We have never systematically evaluated our skills. We don't know which skills improve agent performance, which are ignored, or which actively hurt by consuming context tokens without adding value. An evaluation framework should be a prerequisite for any skill redesign.

### 3.10 Anti-Patterns

The guide identifies these anti-patterns:
- **Windows-style paths** (always use forward slashes)
- **Offering too many options** (provide a default with an escape hatch, not a menu)
- **Deeply nested references** (keep one level deep)
- **Assuming tools are installed** (be explicit about dependencies)
- **Voodoo constants** (all values must be justified and documented)

**Application to Kanbanzai:** The "too many options" anti-pattern is relevant. Our skills sometimes present alternative approaches without a clear default. For example, the document creation skill could present multiple registration patterns. Better: provide the default path, mention alternatives only as escape hatches.

### 3.11 The Best Practices Checklist

The guide concludes with a comprehensive checklist that every skill should pass before sharing. Key items:

**Core quality:**
- [ ] Description is specific and includes key terms
- [ ] Description includes both what the skill does and when to use it
- [ ] SKILL.md body is under 500 lines
- [ ] Additional details are in separate files
- [ ] No time-sensitive information
- [ ] Consistent terminology throughout
- [ ] Examples are concrete, not abstract
- [ ] File references are one level deep
- [ ] Progressive disclosure used appropriately
- [ ] Workflows have clear steps

**Testing:**
- [ ] At least three evaluations created
- [ ] Tested with real usage scenarios
- [ ] Team feedback incorporated

**Application to Kanbanzai:** We should adopt this checklist as a gate for all skill creation and updates. Currently, skills are created and committed without any structured quality check.

---

## 4. Source Analysis: Increase Output Consistency

**Source:** https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/increase-consistency

### What the document says

The guide provides five techniques for making Claude's responses more consistent:

1. **Specify the desired output format** — Use JSON, XML, or custom templates so Claude understands every formatting element required.

2. **Constrain with examples** — Provide input/output examples. "This trains Claude's understanding better than abstract instructions."

3. **Use retrieval for contextual consistency** — Ground responses in a fixed information set for tasks requiring consistent context.

4. **Chain prompts for complex tasks** — Break complex tasks into smaller, consistent subtasks. "Each subtask gets Claude's full attention, reducing inconsistency errors across scaled workflows."

5. **Keep Claude in character** — Use system prompts to define role and personality. Prepare Claude for common scenarios.

### Application to Kanbanzai

Each technique maps to a specific Kanbanzai improvement:

**1. Specify output format → Specification and plan templates**

When agents write specifications or dev plans, the output format should be precisely defined. Not "write a specification" but "produce a document with these exact sections in this order." This is the Template Pattern from the best practices guide.

Concrete action: Each document-producing skill should include an explicit output template.

**2. Constrain with examples → Good/bad output examples in skills**

Our skills lack examples almost entirely. The `kanbanzai-agents` skill describes how to write commit messages but doesn't include input/output examples. The `kanbanzai-documents` skill describes how to register documents but doesn't show a complete example registration sequence.

Concrete action: Add 2–3 concrete examples to each skill, showing both good and bad outputs with explanations of why.

**3. Retrieval for contextual consistency → Knowledge base integration**

Kanbanzai already has a knowledge base (`knowledge` tool). The consistency technique suggests grounding agent responses in this knowledge base — automatically surfacing relevant knowledge entries during context assembly so agents use consistent terminology and follow established conventions.

This reinforces the "institutional memory" recommendation from our previous research.

**4. Chain prompts → Task decomposition with sub-agent handoffs**

This is already a core part of Kanbanzai's architecture — `decompose` breaks work into tasks, `handoff` generates sub-agent prompts, and `spawn_agent` delegates. The consistency improvement is to ensure each sub-task's handoff prompt includes the relevant skill content, not just the task description.

**5. Keep Claude in character → Role profiles with vocabulary**

This reinforces the context profiles and vocabulary routing from our previous research. Each agent role (implementer, reviewer, orchestrator) should have a consistent identity established in the system prompt.

---

## 5. Source Analysis: Reduce Hallucinations

**Source:** https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/reduce-hallucinations

### What the document says

The guide provides basic and advanced techniques:

**Basic techniques:**
1. **Allow Claude to say "I don't know"** — Explicitly give permission to admit uncertainty.
2. **Use direct quotes for factual grounding** — For long documents, have Claude extract word-for-word quotes before performing tasks.
3. **Verify with citations** — Make responses auditable by requiring quotes and sources for claims.

**Advanced techniques:**
4. **Chain-of-thought verification** — Ask Claude to explain reasoning step-by-step before giving a final answer.
5. **Best-of-N verification** — Run the same prompt multiple times and compare outputs.
6. **Iterative refinement** — Use outputs as inputs for follow-up prompts to catch inconsistencies.
7. **External knowledge restriction** — "Explicitly instruct Claude to only use information from provided documents and not its general knowledge."

### Application to Kanbanzai

The most directly applicable techniques:

**"Allow Claude to say I don't know" → Explicit uncertainty protocol**

Our workflow skill should explicitly tell agents: "If you are unsure which stage a piece of work belongs to, or whether a transition is valid, stop and ask the human. Do not guess." This is already partially in the workflow skill but should be more prominent — positioned early in the document where attention is highest (per the attention curve research from our previous report).

**"External knowledge restriction" → Specification-only implementation**

When agents implement tasks, they should be explicitly told: "Implement according to the specification. Do not add features or make design decisions not covered by the spec. If the spec is ambiguous, stop and ask." This directly addresses the observed problem of agents being "too keen to get to implementation" and making undocumented design decisions.

**"Direct quotes for factual grounding" → Spec citation requirement**

When agents make implementation decisions, they should cite the specific section of the specification that justifies the decision. This creates an audit trail and forces agents to ground decisions in the spec rather than general knowledge.

**"Iterative refinement" → Review feedback loops**

The review workflow should use iterative refinement: reviewer identifies issues, implementer addresses them, reviewer re-checks. This is already part of Kanbanzai's lifecycle but could be made more explicit in the review skill.

---

## 6. Source Analysis: The Skill-Creator Meta-Skill

**Source:** https://github.com/anthropics/skills/tree/main/skills/skill-creator

### What the document says

The `skill-creator` is a comprehensive meta-skill for creating, testing, and iterating on skills. It is the most sophisticated skill in the anthropics/skills repository (108k stars). Key insights:

### 6.1 The Create → Test → Review → Improve Loop

The skill-creator defines a rigorous development process:

1. **Capture intent** — What should the skill do? When should it trigger? What's the expected output format? Should we set up test cases?
2. **Interview and research** — Proactively ask about edge cases, input/output formats, success criteria, dependencies.
3. **Write SKILL.md** — Based on the interview, fill in name, description, instructions.
4. **Create test cases** — 2–3 realistic test prompts (the kind of thing a real user would actually say).
5. **Run and evaluate** — Spawn sub-agents with and without the skill, compare outputs.
6. **Improve based on feedback** — Read transcripts, identify patterns, revise.
7. **Repeat until satisfied** — Expand test set, try at larger scale.

**Application to Kanbanzai:** We have no skill development process. Skills are written based on intuition and committed without testing. Adopting even a simplified version of this loop would significantly improve skill quality.

### 6.2 Description Optimization

The skill-creator includes a dedicated **description optimization workflow**:

1. Generate 20 eval queries — mix of should-trigger and should-not-trigger
2. Review with user
3. Run optimization loop (60% train, 40% test, 3 runs per query, iterate up to 5 times)
4. Apply the best description

Key insight about triggering:
> "Claude only consults skills for tasks it can't easily handle on its own — simple, one-step queries may not trigger a skill even if the description matches perfectly."

And the recommendation for description tone:
> "Currently Claude has a tendency to 'undertrigger' skills — to not use them when they'd be useful. To combat this, please make the skill descriptions a little bit 'pushy'."

**Application to Kanbanzai:** Our discovery problem may be partly a description tone problem. If Claude under-triggers skills by default, our descriptions need to be more assertive about when they apply. For example, instead of:

> "Use when deciding what workflow stage work belongs to"

Try:

> "Use when deciding what workflow stage work belongs to. Use even when the agent is confident about the next step — workflow errors are expensive to undo."

In fact, reviewing our current descriptions, the `kanbanzai-workflow` skill already does this: *"Use even when the agent is confident about the next step — workflow errors are expensive to undo."* This is good practice that should be applied to all skills.

### 6.3 Writing Style Philosophy

The skill-creator contains remarkable guidance on how to write skill instructions:

> "Try to explain to the model why things are important in lieu of heavy-handed musty MUSTs. Use theory of mind and try to make the skill general and not super-narrow to specific examples."

> "If you find yourself writing ALWAYS or NEVER in all caps, or using super rigid structures, that's a yellow flag — if possible, reframe and explain the reasoning so that the model understands why the thing you're asking for is important."

This directly contradicts a naive reading of our previous research, which recommended "Always/Never X BECAUSE Y" anti-patterns. The skill-creator says: explain the *why* instead of using rigid imperatives.

**Resolution:** These approaches are not contradictory. The key is the BECAUSE clause. The skill-creator is saying: don't just write "ALWAYS do X" — write "Do X because Y." The anti-pattern format "Always X BECAUSE Y" already includes the reasoning. The guidance is against *unexplained* imperatives, not against clear conventions. But the tone should be explanatory rather than authoritarian.

### 6.4 The Improvement Philosophy

The skill-creator's guidance on iterating skills is excellent:

> "**Generalize from the feedback.** The big picture thing that's happening here is that we're trying to create skills that can be used a million times across many different prompts. Rather than put in fiddly overfitty changes, or oppressively constrictive MUSTs, if there's some stubborn issue, you might try branching out and using different metaphors, or recommending different patterns of working."

> "**Keep the prompt lean.** Remove things that aren't pulling their weight. Make sure to read the transcripts, not just the final outputs — if it looks like the skill is making the model waste a bunch of time doing things that are unproductive, you can try getting rid of the parts of the skill that are making it do that."

> "**Look for repeated work across test cases.** If all 3 test cases resulted in the subagent writing a `create_docx.py` or a `build_chart.py`, that's a strong signal the skill should bundle that script."

**Application to Kanbanzai:** The "look for repeated work" principle is especially relevant. If agents consistently perform the same multi-step operation (e.g., checking lifecycle prerequisites, then making a transition, then registering a document), that sequence should be bundled as a tool or script, not described as instructions.

### 6.5 Evaluation Architecture

The skill-creator uses a structured evaluation system:

- **Eval JSON format** — `{id, prompt, expected_output, files, assertions}`
- **Assertions** — Objective, verifiable checks with descriptive names
- **Baseline comparison** — Every run includes a without-skill baseline
- **Grading** — Automated grading against assertions, with evidence
- **Benchmarking** — Aggregate pass rates, timing, and token usage
- **Viewer** — HTML interface for human review of qualitative outputs

**Application to Kanbanzai:** A full evaluation system like this is likely too heavy for our current stage, but the core pattern is valuable:

1. Define 2–3 test scenarios per skill
2. Run agents with and without the skill on the same scenarios
3. Compare outputs qualitatively
4. Track whether the skill improves outcomes

Even a lightweight version of this would be more rigorous than our current approach of "write it, commit it, hope it works."

### 6.6 Should We Adopt the Skill-Creator?

**Assessment:** The skill-creator is designed for general-purpose Claude Code skill development. It assumes:
- Interactive user sessions (interview, review, iterate)
- Access to sub-agents for parallel test runs
- A browser for the eval viewer
- The `claude` CLI for description optimization

For Kanbanzai's purposes:

| Feature | Useful? | Notes |
|---------|---------|-------|
| Create → test → review → improve loop | **Yes** | Core process we should adopt |
| Description optimization workflow | **Partially** | The concept is valuable; the tooling is overkill for 6-10 skills |
| Eval JSON format and assertions | **Yes** | Lightweight version for our skills |
| Blind comparison system | **No** | Too heavy for our scale |
| Packaging as .skill files | **No** | Not relevant to our filesystem-based deployment |

**Recommendation:** Do not adopt the skill-creator as-is (it's designed for a different context), but extract its evaluation-driven development process and description optimization concepts. The create → test → review → improve loop should become our standard skill development workflow.

---

## 7. Cross-Cutting Themes

Seven themes emerge across all sources:

### Theme 1: The Context Window Is a Shared Resource

Every source emphasises that skill content competes with conversation history, task context, and other skills for the agent's attention. This has two implications:

1. **Be concise** — Every token must justify its presence.
2. **Load on demand** — Don't put content into context until the agent needs it.

Kanbanzai's context assembly system should track estimated token usage and warn when context packets exceed optimal thresholds (per our previous research: 15–40% of window is optimal, >60% degrades output).

### Theme 2: Description Quality Determines Discovery

Skills that aren't triggered are worthless regardless of their content quality. The description is the only thing Claude sees before deciding to trigger a skill. It must be:
- Specific (include key terms the agent will be thinking about)
- Comprehensive (cover both what the skill does and when to use it)
- Assertive (push for triggering, especially in workflow-critical skills)

### Theme 3: Match Constraint Level to Task Risk

Low-freedom (exact procedures) for fragile operations. High-freedom (general guidance) for creative work. The cost of over-constraining creative work is mediocre output; the cost of under-constraining fragile operations is broken state.

### Theme 4: Examples Beat Rules

Multiple sources confirm that input/output examples are more effective than abstract instructions. 3 well-chosen examples can match 9 in effectiveness (per LangChain research cited in our previous report). Skills should lead with examples and use rules as scaffolding.

### Theme 5: Feedback Loops Prevent Skipping

Checklists and validate-fix-repeat loops are the most reliable way to prevent agents from skipping steps. This directly addresses our observed problem.

### Theme 6: Evaluation Must Precede Documentation

Write evaluations before writing extensive documentation. Identify what Claude actually gets wrong, then write the minimum instructions needed to address those specific failures. Don't document imagined problems.

### Theme 7: Explain Why, Not Just What

The skill-creator's writing philosophy — explain reasoning instead of using rigid imperatives — is consistent with the vocabulary routing research. When Claude understands *why* a convention exists, it can generalise correctly to novel situations. When it only knows *what* the rule is, it follows it rigidly in matching cases and ignores it in novel ones.

---

## 8. Gap Analysis: Current Kanbanzai Skills vs. Best Practices

### Current State

Kanbanzai has two skill systems:

**1. Legacy `.skills/` directory** — Three markdown files (`code-review.md`, `document-creation.md`, `plan-review.md`) plus a README. These are referenced from `AGENTS.md` but have no YAML frontmatter, no progressive disclosure, and no standardised structure.

**2. New `.agents/skills/kanbanzai-*/` directory** — Six skills with proper YAML frontmatter (`kanbanzai-workflow`, `kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-design`, `kanbanzai-planning`, `kanbanzai-getting-started`). These follow the `SKILL.md` convention and are discovered by Claude Code's skill scanning.

Additionally, there are **embedded skills in `internal/kbzinit/skills/`** that are installed into new projects during `kbz init`. These include the six kanbanzai skills plus three additional ones (`plan-review`, `review`, `specification`).

### Gap Table

| Best Practice | Current State | Gap Severity |
|---|---|---|
| **Progressive disclosure** (3-level loading) | Skills load entirely at Level 2. No Level 3 reference files. | **High** — most skills exceed the 500-line guideline |
| **Concise content** (only what Claude doesn't know) | Skills explain general concepts Claude already knows | **Medium** — wastes tokens, dilutes Kanbanzai-specific content |
| **Freedom levels** match task risk | Uniform medium freedom | **High** — fragile operations need low freedom |
| **Descriptions assertive** and trigger-optimized | Descriptions are good but not "pushy" enough for all skills | **Low** — `kanbanzai-workflow` already does this well |
| **Consistent terminology** | Vocabulary varies across skills and documents | **Medium** — causes inconsistent agent outputs |
| **Input/output examples** | Almost no examples in any skill | **High** — examples beat rules per multiple sources |
| **Copy-paste checklists** | No checklists in any skill | **High** — directly addresses step-skipping |
| **Feedback loops** (validate → fix → repeat) | Not present in skills; partially in MCP tools | **Medium** — tools help but skills should reinforce |
| **Evaluation-driven development** | No evaluation process exists | **High** — skills are untested |
| **One-level-deep references** | N/A — no reference files exist yet | **High** when we add them |
| **Template patterns** for output format | Minimal template guidance in specification skill | **Medium** — specifications and plans need templates |
| **Anti-pattern documentation** with reasoning | Not present | **Medium** — missed opportunity for institutional memory |
| **Naming conventions** (gerund form) | Noun-based naming (`kanbanzai-{noun}`) | **Low** — functional but less discoverable |
| **Testing with real scenarios** | Never done | **High** — we don't know which skills work |

### Dual-System Confusion

Having both `.skills/` and `.agents/skills/` creates confusion:
- Which system should agents use?
- The legacy `.skills/code-review.md` and the new `.agents/skills/kanbanzai-*/` overlap in scope
- The embedded `internal/kbzinit/skills/` adds a third copy of some skills

**Recommendation:** Consolidate to a single skill system. The `.agents/skills/` structure with YAML frontmatter is the correct approach. Legacy `.skills/` should be migrated and deprecated.

---

## 9. Recommendations

Ordered by expected impact, with effort estimates.

### 9.1 High Impact, Low Effort

**R1. Add copy-paste checklists to workflow-critical skills**

Add a checklist at the top of each workflow stage section that agents can copy and track:

```
## Before Creating a Feature

Copy this checklist:
- [ ] Design document exists and is approved
- [ ] Plan entity exists
- [ ] Human has signalled readiness to create features
- [ ] Feature scope is documented in the design
```

This directly addresses the "agents skip steps" problem. Effort: ~1 day.

**R2. Add 2–3 concrete examples to each skill**

For each skill, add input/output examples showing good and bad outcomes:
- Commit message skill: good and bad commit messages with explanations
- Document registration: complete registration sequence
- Stage gate checks: what a correct vs. incorrect transition looks like

This leverages the "examples beat rules" finding. Effort: ~2 days.

**R3. Audit and strengthen all skill descriptions**

Review every skill description against the "pushy description" standard. Ensure each description:
- Uses third person consistently
- Includes specific trigger terms the agent will be thinking about
- Includes an assertive "use even when..." clause for workflow-critical skills
- Covers both what and when

Effort: ~0.5 days.

### 9.2 High Impact, Medium Effort

**R4. Restructure skills with progressive disclosure**

For each skill that exceeds ~200 lines, split into:
- `SKILL.md` — Overview, routing, quick reference (~200 lines max)
- Reference files — Detailed procedures, examples, anti-patterns

Use the domain-specific organization pattern:
```
kanbanzai-workflow/
├── SKILL.md
├── stage-gates.md
├── transitions.md
└── anti-patterns.md
```

This is the single most impactful structural change. Effort: ~3 days.

**R5. Differentiate freedom levels by task type**

Rewrite skill procedures with appropriate constraint levels:
- **Low freedom** for lifecycle transitions, document registration, stage gate checks: exact tool call sequences, "do exactly this"
- **Medium freedom** for specification writing, plan creation: templates with flexibility
- **High freedom** for design work, implementation: general guidance, trust the agent

Effort: ~2 days.

**R6. Strip general knowledge, focus on Kanbanzai-specific content**

Audit each skill and remove:
- Explanations of concepts Claude already knows (state machines, lifecycles, etc.)
- Background context about why workflows exist in general
- Definitions of common terms

Replace with:
- Kanbanzai-specific rules and conventions
- Kanbanzai-specific vocabulary
- Kanbanzai-specific tool calls and their parameters

This is the "Claude is already smart" principle. Effort: ~2 days.

### 9.3 Medium Impact, Medium Effort

**R7. Consolidate skill systems**

Migrate `.skills/code-review.md` and `.skills/plan-review.md` into the `.agents/skills/` structure. Deprecate the `.skills/` directory. Ensure `internal/kbzinit/skills/` stays in sync.

Effort: ~1 day.

**R8. Add anti-pattern sections with reasoning**

For each skill, add an anti-patterns section:
```
## Common Mistakes

**Skipping the specification stage for "simple" features.**
Features that seem simple often have hidden complexity. The specification
stage exists to surface this complexity before implementation begins.
Without a spec, agents make undocumented design decisions that are
expensive to discover and undo during review.
```

Note the explanatory tone (per skill-creator guidance) rather than "NEVER skip the spec stage."

Effort: ~2 days.

**R9. Create a skill quality checklist gate**

Adopt the best practices checklist (§3.11) as a required gate for all skill creation and modification. Store it as a reference file that skill authors (human or agent) follow.

Effort: ~0.5 days.

### 9.4 Medium Impact, Higher Effort

**R10. Implement lightweight skill evaluation**

For each skill:
1. Define 3 test scenarios (realistic tasks)
2. Run agents with and without the skill on the same tasks
3. Record qualitative observations
4. Track whether the skill improves outcomes

This doesn't need the full skill-creator evaluation infrastructure. Even manual testing with documented results would be a massive improvement over zero testing.

Effort: ~3 days for initial round, ongoing.

**R11. Add vocabulary payloads to skills**

Per our previous research, add 15–30 precise domain terms per skill:
```
## Vocabulary

These terms have specific meanings in the Kanbanzai system:
- **feature**: A unit of work that travels the full lifecycle...
- **task**: An implementation unit within a feature...
- **stage gate**: A prerequisite check before advancing...
```

This activates domain-specific knowledge clusters per the vocabulary routing research.

Effort: ~2 days.

### 9.5 Lower Priority / Future

**R12. Implement description optimization for critical skills**

For the 2–3 most important skills (workflow, agents, documents), run a simplified version of the skill-creator's description optimization:
1. Write 10 test queries (should-trigger and should-not-trigger)
2. Test whether the skill triggers correctly
3. Iterate on the description

Effort: ~1 day per skill.

**R13. Evaluate adopting gerund naming**

Test whether renaming skills to gerund form (`managing-workflow`, `dispatching-work`) improves discovery. This is low-risk but may not be worth the migration effort for 6–10 skills.

Effort: ~0.5 days.

**R14. Build a skill-creator for Kanbanzai-specific skills**

A simplified version of the anthropic skill-creator, adapted for our context:
- Creates skills following our template and conventions
- Generates test scenarios
- Runs basic evaluations
- Iterates based on feedback

This is only worthwhile if we expect to create many more skills. Currently we have ~10. If the skill catalog grows significantly, this becomes valuable.

Effort: ~3–5 days.

---

## Appendix A: Key Quotes from Sources

### From the Skills Overview

> "Progressive disclosure ensures only relevant content occupies the context window at any given time."

> "Claude navigates your Skill like you'd reference specific sections of an onboarding guide, accessing exactly what each task requires."

### From the Best Practices Guide

> "The context window is a public good."

> "Claude is already very smart. Only add context Claude doesn't already have."

> "Think of Claude as a robot exploring a path: Narrow bridge with cliffs on both sides — there's only one safe way forward. Open field with no hazards — many paths lead to success."

> "Create evaluations BEFORE writing extensive documentation. This ensures your Skill solves real problems rather than documenting imagined ones."

### From the Skill-Creator

> "Try to explain to the model why things are important in lieu of heavy-handed musty MUSTs."

> "Currently Claude has a tendency to 'undertrigger' skills — to not use them when they'd be useful. To combat this, please make the skill descriptions a little bit 'pushy'."

> "If there's some stubborn issue, you might try branching out and using different metaphors, or recommending different patterns of working."

> "If all 3 test cases resulted in the subagent writing a create_docx.py or a build_chart.py, that's a strong signal the skill should bundle that script."

### From the Consistency Guide

> "Provide examples of your desired output. This trains Claude's understanding better than abstract instructions."

> "Break down complex tasks into smaller, consistent subtasks. Each subtask gets Claude's full attention, reducing inconsistency errors across scaled workflows."

### From the Hallucinations Guide

> "Explicitly give Claude permission to admit uncertainty. This simple technique can drastically reduce false information."

> "Explicitly instruct Claude to only use information from provided documents and not its general knowledge."

---

## Appendix B: Comparison — Kanbanzai Skills vs. Anthropic Recommended Structure

### Anthropic recommended structure

```
skill-name/
├── SKILL.md            # Required. YAML frontmatter + routing instructions (<500 lines)
│   ├── name            # lowercase, hyphens, max 64 chars
│   ├── description     # What it does + when to use it, max 1024 chars
│   └── body            # Overview, quick start, references to detail files
├── reference/          # Optional. Detail files loaded on demand
│   ├── procedures.md   # Step-by-step instructions for specific operations
│   ├── examples.md     # Input/output examples
│   └── anti-patterns.md # Common mistakes with explanations
└── scripts/            # Optional. Executable scripts (output enters context, not source)
    └── validate.py     # Deterministic checks
```

### Current Kanbanzai structure

```
.agents/skills/kanbanzai-workflow/
└── SKILL.md            # Everything in one file (>500 lines in some cases)
                        # No reference files, no scripts, no progressive disclosure
```

### Proposed Kanbanzai 3.0 structure

```
.agents/skills/kanbanzai-workflow/
├── SKILL.md            # Overview, stage table, routing to detail files (~200 lines)
├── stage-gates.md      # Detailed gate requirements per stage
├── transitions.md      # Valid transition table, exact tool calls
├── checklists.md       # Copy-paste checklists for each workflow stage
├── anti-patterns.md    # Common mistakes with BECAUSE reasoning
└── examples.md         # Good/bad examples of workflow decisions
```

---

## Appendix C: Sources Referenced

| # | Source | URL | Retrieved |
|---|--------|-----|-----------|
| 1 | Agent Skills Overview | https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview | 2025-07-14 |
| 2 | Agent Skills Quickstart | https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart | 2025-07-14 |
| 3 | Skill Authoring Best Practices | https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices | 2025-07-14 |
| 4 | Increase Output Consistency | https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/increase-consistency | 2025-07-14 |
| 5 | Reduce Hallucinations | https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/reduce-hallucinations | 2025-07-14 |
| 6 | Skill-Creator SKILL.md | https://github.com/anthropics/skills/tree/main/skills/skill-creator | 2025-07-14 |
| 7 | Prior Research: AI Agent Best Practices | work/research/ai-agent-best-practices-research.md | Internal |
| 8 | Prior Design: Skills System Redesign | work/design/skills-system-redesign.md | Internal |
| 9 | Equipping Agents for the Real World with Agent Skills (blog) | Referenced in Skills Overview; URL returned 404 at time of research | Not retrieved |