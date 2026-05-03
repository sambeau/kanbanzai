# Design: Prompt Engineering Skill

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03                     |
| Status | Draft                          |
| Author | sambeau                        |

## Overview

This design proposes a new `prompt-engineering` skill for the `.agents/skills/`
system, distilling the research-backed prompt engineering guide
(`refs/prompt-engineering-guide.md`) into an agent-executable SKILL.md. The
skill covers the full prompt engineering methodology: U-shaped attention curve
section ordering, vocabulary routing, identity construction, constraint
formulation, anti-pattern naming, example crafting, and output format design.

This is a self-contained documentation skill — it produces no code changes to
the kanbanzai server. It creates a skill file and reference files that agents
use when writing or revising prompts, skills, and system instructions.

## Goals and Non-Goals

### Goals

- Create a `prompt-engineering` skill following the research-backed skill
  architecture from the skill-authoring-best-practices research
- Structure the skill for the U-shaped attention curve: vocabulary payload
  first, anti-patterns near top, procedure in the middle, examples near
  bottom, retrieval anchors last
- Include 15–30 domain vocabulary terms that route to prompt engineering
  knowledge clusters
- Provide 2–3 BAD vs GOOD example pairs showing prompt engineering
  principles applied to real prompt fragments
- Keep SKILL.md under 500 lines; move detailed reference material to
  Level 3 reference files
- Include a copy-paste checklist agents can use when authoring prompts
- Use dual-register descriptions (expert terminology + natural language)
  for both routing depth and trigger breadth

### Non-Goals

- Creating a role to accompany the skill — this is a standalone skill
  usable by any role
- Building evaluation infrastructure for the skill
- Modifying the MCP server or any Go code
- Creating a skill-authoring meta-skill (that's separate future work)
- Applying prompt engineering principles to existing skills — this skill
  teaches the methodology; skill retrofitting is a separate batch

## Problem and Motivation

### Context

`refs/prompt-engineering-guide.md` is a comprehensive reference document
distilling 17 peer-reviewed papers into 10 principles for writing effective
AI agent prompts. It covers the U-shaped attention curve, vocabulary routing,
identity construction, constraint formulation, anti-pattern naming, and
output format design. It includes a quick-reference checklist and a template
that synthesises all findings.

However, the guide is a **passive reference document** — the agent must
already know about it and choose to consult it. The research on skill
architecture (`work/research/skill-authoring-best-practices.md`) shows that
agents discover and apply skills through the trigger mechanism, not through
reference document reading. A `.md` file in `refs/` has no YAML frontmatter,
no description, and no trigger routing — it will never be loaded unless
explicitly referenced in a prompt or read by an agent that already knows the
path.

### The gap

The `refs/prompt-engineering-guide.md` has no corresponding skill in either
`.agents/skills/` or `.kbz/skills/`. This means:

1. **No automatic discovery.** Agents don't find it when they need to write
   or revise a prompt. They must be told about it.
2. **No procedural guidance.** The guide explains principles but does not
   provide a step-by-step procedure an agent can follow.
3. **No vocabulary routing.** The guide's domain terms aren't surfaced as
   routing signals that activate expert knowledge clusters.
4. **No examples.** The guide has a template but no concrete BAD vs GOOD
   pairs an agent can pattern-match against.
5. **No checklist.** The guide has a "quick-reference" checklist but it's
   not structured for agent copy-paste tracking.

### Why a skill is the right vehicle

Skills in the Kanbanzai system are the mechanism for surfacing procedural
knowledge at the right time. A `prompt-engineering` skill would:

- **Trigger automatically** when an agent is asked to write, revise, or
  evaluate a prompt or skill
- **Route to expert knowledge** via the vocabulary payload (attention
  mechanism, recency bias, causal masking, etc.)
- **Provide a procedure** the agent follows step by step
- **Prevent common mistakes** through named anti-patterns with BECAUSE
  clauses
- **Include a copy-paste checklist** for tracking progress

## Design

### Skill location and structure

The skill lives in `.agents/skills/prompt-engineering/` with progressive
disclosure across three levels:

```
.agents/skills/prompt-engineering/
├── SKILL.md                    # Level 2: routing + procedure (<500 lines)
│   ├── YAML frontmatter        # Level 1: always loaded (name + description)
│   ├── Vocabulary              # 15–30 terms, first body section
│   ├── Anti-Patterns           # Named patterns with BECAUSE clauses
│   ├── Checklist               # Copy-paste tracking checklist
│   ├── Procedure               # 5–10 imperative steps
│   ├── Output Format           # Template for prompt engineering output
│   ├── Examples                # 2–3 BAD vs GOOD pairs (best last)
│   └── Retrieval Anchors       # "Questions This Skill Answers"
└── references/                 # Level 3: loaded on demand
    ├── attention-curve.md      # Detailed explanation of U-shaped attention
    ├── vocabulary-routing.md   # Deep dive on domain terminology routing
    └── full-examples.md        # Extended annotated examples
```

### Section ordering (U-shaped attention curve)

Per Liu et al. (2024) and Wu et al. (2025), content at the beginning and
end of context receives disproportionate attention. The skill's body
follows this ordering:

| Position | Attention | Section | Rationale |
|----------|-----------|---------|-----------|
| Top | HIGH | Vocabulary payload | Primary routing signal — determines which knowledge clusters activate |
| Near top | HIGH | Anti-patterns | Hard-won lessons benefit from peak attention |
| Near top | HIGH | Checklist | Copy-paste tracking — must be seen early to be used |
| Middle | LOWER | Procedure | Numbered steps survive attention degradation |
| Near bottom | RISING | Output format + examples | Recency bias improves pattern matching |
| Bottom | HIGH | Retrieval anchors | End-of-context attention boost |

### YAML frontmatter design

The frontmatter follows the dual-register pattern from the research:
an expert register for routing depth and a natural register for trigger
breadth.

```yaml
name: prompt-engineering
description:
  expert: "Prompt and skill authoring following attention-optimised section
    ordering with vocabulary routing, named anti-patterns, BECAUSE-clause
    constraints, and retrieval anchors. Applies the U-shaped attention curve
    (Liu et al. 2024), few-shot demonstration patterns, and ALWAYS/NEVER
    constraint pairing."
  natural: "Write, revise, or evaluate a prompt, skill, or system instruction.
    Use when creating agent instructions, crafting few-shot examples, or
    structuring context for optimal attention allocation."
triggers:
  - write a prompt
  - revise a prompt
  - evaluate a prompt
  - create a skill
  - author system instructions
  - improve prompt quality
  - structure agent context
  - design agent instructions
  - write few-shot examples
  - craft a system prompt
constraint_level: medium
```

### Vocabulary payload (15–30 terms)

The vocabulary section is the first body content — it's the primary routing
signal. Each term passes the 15-year practitioner test: would a senior
prompt engineer with 15+ years of experience use this exact term when
speaking with a peer?

Core routing terms:

| Term | Why it routes |
|------|--------------|
| U-shaped attention curve | Liu et al. 2024 — anchors to "Lost in the Middle" research |
| vocabulary routing | Ranjan et al. 2024 — domain terminology as knowledge cluster selector |
| recency bias | Wu et al. 2025 — last-in-sequence preferential recall |
| causal masking | Architectural cause of attention degradation |
| RoPE positional encoding | Technical mechanism behind position-dependent attention |
| attention budget | Anthropic 2025 — token competition model |
| progressive disclosure | Three-level loading: metadata → instructions → resources |
| dual-register description | Expert terminology + natural language for discovery |
| 15-year practitioner test | PRISM-derived litmus for vocabulary inclusion |
| named anti-patterns | Specific names activate expert knowledge (vs. generic warnings) |
| BECAUSE clause | Generalisability constraint — why, not just what |
| few-shot demonstration | LangChain 2024 — 3 examples match 9 |
| retrieval anchors | End-of-context questions for semantic indexing |
| constraint pairing | Zamfirescu-Pereira et al. 2023 — positive + negative strongest |
| effort budget | Explicit tool-call expectations prevent underspecification |
| output template | Forcing function that distributes effort across sections |
| right-altitude prompting | 5–15 requirements is the sweet spot |
| n=19 cliff | Vaarta Analytics 2026 — quality collapse at 19+ requirements |
| attention valley | Middle-of-context accuracy degradation zone |
| identity construction | PRISM 2024 — <50 tokens, real job titles, no flattery |
| structured format | MetaGPT — structured artifacts reduce errors ~40% |
| copy-paste checklist | Visible step tracking prevents skipping |
| feedback loop | Validate → fix → repeat pattern |
| section ordering | Position-dependent attention allocation |
| trigger breadth | Natural-language coverage for skill discovery |

### Anti-patterns (named with BECAUSE clauses)

Each anti-pattern has a Detect signal, a BECAUSE clause explaining the
consequence chain, and a Resolve action:

1. **The Flattery Trap** — Detect: superlatives in identity ("world-class
   expert", "exceptionally talented"). BECAUSE: flattery activates
   marketing/motivational text from training data, degrading technical
   output. Resolve: use <50 token identities with real job titles.

2. **The Middle Drop** — Detect: critical constraints or rules placed after
   procedure steps. BECAUSE: Liu et al. (2024) showed 30%+ accuracy drop for
   middle-of-context information. Resolve: place constraints at top,
   procedure in middle, anchors at bottom.

3. **Generic Vocabulary** — Detect: prompts using general terms where
   domain-specific equivalents exist ("review for security" vs. "OWASP Top
   10 audit"). BECAUSE: generic vocabulary routes to blog-post-level
   knowledge; specific terminology routes to expert knowledge clusters.
   Resolve: include 15–30 domain-specific terms.

4. **Requirements Bloat** — Detect: more than 15 requirements in a prompt.
   BECAUSE: at n=19, accuracy drops below n=5 (Vaarta Analytics 2026).
   Resolve: keep to 5–15 precise requirements; move long tail to reference.

5. **Missing BECAUSE Clauses** — Detect: ALWAYS/NEVER rules without
   reasoning. BECAUSE: rules without reasons can't generalise to adjacent
   cases and become dead weight. Resolve: every imperative gets a BECAUSE.

6. **Example Starvation** — Detect: prompts with rules but no examples.
   BECAUSE: 3 well-chosen examples match 9 in effectiveness (LangChain
   2024); examples train understanding better than descriptions. Resolve:
   include at least 2 BAD vs GOOD pairs.

7. **Uniform Constraint Level** — Detect: same specificity of instruction
   across all prompt sections regardless of fragility. BECAUSE: fragile
   operations need exact steps; creative work needs principles. Uniform
   medium freedom degrades both. Resolve: differentiate low/medium/high
   constraint levels per section.

8. **Unnamed Problems** — Detect: describing a mistake without giving it a
   name. BECAUSE: named anti-patterns activate expert knowledge clusters
   while unnamed problems get generic responses. Resolve: use the detect →
   name → explain → resolve → prevent pattern.

### Procedure (5–7 steps)

1. **Identify the task structure.** Is it sequential (spec, design),
   evaluative (review), or parallelisable (implementation)? This determines
   the orchestration pattern and constraint level.

2. **Draft the vocabulary payload.** List 15–30 domain terms. Apply the
   15-year practitioner test to each. Strip terms the model already knows.
   Place this first in the prompt body.

3. **Write constraints as ALWAYS/NEVER with BECAUSE.** Pair positive and
   negative constraints. Every imperative gets a BECAUSE clause explaining
   why. Place constraints near the top.

4. **Name the anti-patterns.** For each likely mistake: give it a specific
   name, describe how to detect it, explain BECAUSE (consequence chain),
   and state the resolution.

5. **Structure for the U-curve.** Order sections: identity + vocabulary
   (top), constraints + anti-patterns (near top), procedure (middle),
   examples + output format (near bottom), retrieval anchors (bottom).

6. **Write 2–3 BAD vs GOOD example pairs.** Use concrete, realistic content.
   Place the best GOOD example last (recency bias). Explain why each is
   good or bad.

7. **Add retrieval anchors and checklist.** End with 5–10 natural-language
   questions. Include a copy-paste checklist at the top.

### Output format (what the skill produces)

When invoked, the skill guides the agent to produce a prompt or evaluate an
existing one. The output is either:

**For prompt authoring:** A complete prompt following the template from
`refs/prompt-engineering-guide.md` (identity, vocabulary, constraints,
anti-patterns, procedure, output format, examples, retrieval anchors).

**For prompt evaluation:** A structured review identifying U-curve
violations, vocabulary gaps, missing anti-pattern names, constraint
weaknesses, and example quality issues. Scored on each dimension 0.0–1.0.

### Constraint level

**Medium** — a preferred pattern exists (the template), but variation is
acceptable based on context. The procedure is guidance, not an exact script.

### Effort budget

When used for prompt authoring: **5–15 tool calls.** Read the existing
prompt or context, consult relevant research, draft, validate against
checklist, iterate.

When used for prompt evaluation: **5–10 tool calls.** Read the prompt,
check each dimension, produce scored verdict.

## Alternatives Considered

### Alternative A: Keep the guide as a reference document only

**Approach:** Add a line to `AGENTS.md` telling agents to read
`refs/prompt-engineering-guide.md` when writing prompts. No skill created.

**Trade-offs:**
- Easier: zero files to create, no skill to maintain
- Harder: relies on agents remembering to consult the guide; no automatic
  trigger; no procedural structure; no vocabulary routing

**Rejected because:** The research shows passive reference documents have
near-zero discovery rate. Agents don't consult them unless explicitly told.
A skill with YAML frontmatter triggers automatically when needed.

### Alternative B: Create a combined role + skill

**Approach:** Create both a `prompt-engineer` role (identity, vocabulary,
anti-patterns) and a `prompt-engineering` skill (procedure, examples, output
format). The role provides who-you-are; the skill provides what-you-do.

**Trade-offs:**
- Better: follows the Kanbanzai role/skill architecture; cleaner separation
- Harder: two files to maintain; a role is overkill for a skill that any
  agent can use

**Rejected because:** Prompt engineering is a cross-cutting skill, not a
specialist role. Any agent — architect, spec-author, implementer, reviewer —
may need to write or revise a prompt. Creating a dedicated role would
require role-switching to use the skill, adding friction without benefit.
The skill is designed to be usable by any role with the existing identity.

### Alternative C: Build prompt engineering into the MCP server

**Approach:** Add a `prompt(action: "validate")` tool to the MCP server
that applies structural checks (U-curve violations, vocabulary count,
example presence) programmatically.

**Trade-offs:**
- Better: deterministic enforcement of structural rules; no agent judgment
  needed
- Harder: significant implementation effort; structural checks can't assess
  semantic quality; requires Go code changes

**Rejected for now, but noted as future work.** A skill is the right first
step. If the skill proves valuable, hardening structural checks into MCP
tools follows the Hardening Principle (replace fuzzy LLM steps with
deterministic tools).

## Decisions

- **Decision:** Create a standalone `prompt-engineering` skill (no
  accompanying role)
- **Context:** The skill is useful to agents in any role. A dedicated role
  would require role-switching for a task that any agent can perform.
- **Rationale:** Cross-cutting skills shouldn't require role changes. The
  skill's vocabulary and anti-patterns are self-contained.
- **Consequences:** Any agent with any role can use the skill. The trade-off
  is that the skill must provide its own vocabulary context rather than
  inheriting from a role.

---

- **Decision:** Follow the progressive disclosure pattern: SKILL.md (<500
  lines) + reference files
- **Context:** The research shows that loading everything in one file forces
  all content into context. Reference material should be in Level 3 files
  loaded only when needed.
- **Rationale:** The full prompt-engineering-guide.md is ~250 lines of dense
  research. The skill needs vocabulary, anti-patterns, procedure, examples,
  and anchors. This won't fit under 500 lines with all detail. Reference
  files for attention-curve and vocabulary-routing deep dives keep SKILL.md
  lean.
- **Consequences:** Three reference files in `references/`. SKILL.md links
  to them one level deep.

---

- **Decision:** Use the output template from `refs/prompt-engineering-guide.md`
  as the skill's output format
- **Context:** The guide's template already synthesises all 10 principles
  into a single structure. Reusing it ensures consistency between the
  reference document and the skill.
- **Rationale:** Don't create a competing format when one already exists
  and is research-backed.
- **Consequences:** Agents using the skill produce prompts matching the
  template. The skill's examples should use this template.

---

- **Decision:** Place the skill in `.agents/skills/prompt-engineering/`
  (not `.kbz/skills/`)
- **Context:** Kanbanzai has two skill directories. `.agents/skills/` is
  for Kanbanzai workflow skills (how to use the system). `.kbz/skills/` is
  for task-execution skills (how to do specific work stages).
- **Rationale:** Prompt engineering is a general authoring skill, not a
  workflow stage. It's closest in nature to the existing `.agents/skills/`
  skills which cover cross-cutting system knowledge. The research on naming
  also notes that `.kbz/skills/` skills are tied to stage bindings, which
  this skill is not.
- **Consequences:** Skill is discovered by Claude Code's skill scanning.
  Not bound to a workflow stage. Usable by any role at any time.
