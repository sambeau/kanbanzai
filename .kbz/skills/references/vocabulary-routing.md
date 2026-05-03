# Vocabulary Routing Deep Dive

> Reference file for `prompt-engineering` skill. Loaded on demand when
> the agent needs deeper understanding of vocabulary as a routing signal.

## The Finding

Ranjan et al. ("One Word Is Not Enough", 2024) demonstrated that the
specific words in a prompt determine which knowledge clusters the model
activates. This is not a metaphor — it is a measurable routing effect.

The vocabulary payload is the **primary quality lever** in prompt
engineering. More important than prompt length, more important than the
number of agents, more important than the orchestration pattern. The
words you choose and where you place them are the #1 determinant of
output quality.

## How It Works

Language models organise knowledge into clusters based on co-occurrence
patterns in training data. When a prompt contains domain-specific
terminology, it activates the specialised knowledge regions associated
with that terminology. When it contains only general terms, it activates
general knowledge regions.

**Example — security review:**

- "Review this code for security issues" → routes to blog-post-level
  security checklists and generic advice
- "Perform an OWASP Top 10 audit. Apply STRIDE threat modelling. Check
  for CWE-89, CWE-79, and CWE-352." → routes to security engineering
  knowledge clusters with specific vulnerability patterns, testing
  methodologies, and remediation strategies

The difference in output quality between these two prompts is large and
measurable — it's not a marginal improvement.

## The 15-Year Practitioner Test

Derived from the PRISM persona framework (2024):

> Would a senior expert with 15+ years of domain experience use this
> exact term when speaking with a peer?

If yes → include it. If no → cut it or replace it.

This test has two effects:
1. It eliminates general terms the model already knows (saves tokens)
2. It surfaces domain-specific terms that route to expert knowledge
   (improves quality)

**Terms that pass:** "parameterised query", "OWASP Top 10", "CWE-89",
"taint analysis", "trust boundary", "input sanitisation"

**Terms that fail:** "security issue", "code problem", "best practice",
"be careful", "watch out for"

## How Many Terms?

The research points to **15–30 domain-specific terms** as the sweet spot.
Fewer than 15 and the routing signal is too weak to reliably activate
expert clusters. More than 30 and diminishing returns set in, plus the
attention budget cost of the vocabulary section starts competing with
other sections.

The terms should be:
- Specific to the domain (not general workflow vocabulary)
- Passing the 15-year practitioner test
- Listed early in the prompt (first body section) where attention is
  highest

## What NOT to Include

The vocabulary section is NOT a glossary. Do not:
- Define terms the model already knows
- Include general-purpose terms ("prompt", "agent", "output", "task")
- Explain what the terms mean (unless a term is genuinely obscure)
- Use the vocabulary section as a concept introduction

The vocabulary section IS a routing signal. Think of it as metadata for
the model's internal retrieval system, not as education for the model.

## Vocabulary Placement

The vocabulary section must be the **first content in the prompt body**
(after identity). This is because:

1. Early-position tokens receive attention from all subsequent tokens
   (causal masking property)
2. The routing signal needs to be processed before the model begins
   interpreting the rest of the prompt
3. Knowledge cluster activation happens early and persists — priming
   the right clusters at the start shapes everything that follows

## Dual-Register Descriptions

For skill descriptions (YAML frontmatter), use two registers:

**Expert register** (for routing depth): "Prompt and skill authoring
following attention-optimised section ordering with vocabulary routing,
named anti-patterns, BECAUSE-clause constraints, and retrieval anchors.
Applies the U-shaped attention curve (Liu et al. 2024)."

**Natural register** (for trigger breadth): "Write, revise, or evaluate
a prompt, skill, or system instruction. Use when creating agent
instructions, crafting few-shot examples, or structuring context for
optimal attention allocation."

The expert register ensures the skill routes to deep knowledge when
triggered by precise queries. The natural register ensures the skill is
found by fuzzy queries that don't use domain terminology.

## Key Sources

- Ranjan et al., "One Word Is Not Enough" (2024)
- PRISM Persona Framework (2024) — <50 token identities, flattery
  degradation
- Anthropic, Skill Authoring Best Practices — "Claude is already very
  smart — only add context Claude doesn't already have"
- Zamfirescu-Pereira et al., "Why Johnny Can't Prompt" (CHI 2023) —
  vocabulary effects on output quality
