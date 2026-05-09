---
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
roles: [architect, spec-author, implementer, implementer-go, orchestrator,
  reviewer, reviewer-conformance, reviewer-quality, reviewer-security,
  reviewer-testing, researcher, documenter, doc-pipeline-orchestrator,
  doc-editor, doc-checker, doc-stylist, doc-copyeditor]
stage: designing
constraint_level: medium
---

# SKILL: Prompt Engineering

## Vocabulary

Write, revise, and evaluate prompts and skills using the research-backed
10 principles from `refs/prompt-engineering-guide.md`. This skill applies
the U-shaped attention curve, vocabulary routing, constraint pairing,
named anti-patterns, and few-shot demonstration patterns to produce
prompts that route to expert knowledge clusters.

These terms have specific meanings in prompt engineering research. They
are routing signals — they activate domain-specific knowledge clusters:

- **U-shaped attention curve** — accuracy is highest at context start and end, with 30%+ drop in the middle (Liu et al. 2024, Wu et al. 2025). Drives all section ordering decisions.
- **vocabulary routing** — domain-specific terminology determines which knowledge clusters the model activates (Ranjan et al. 2024). The primary quality lever.
- **recency bias** — the last item in a sequence has outsized influence on generation. Place best examples last.
- **causal masking** — architectural cause of U-shaped attention: each token only attends to preceding tokens.
- **RoPE positional encoding** — mechanism that creates position-dependent attention decay, producing the attention valley.
- **attention budget** — every token competes with every other token for attention weight. Irrelevant tokens actively degrade relevant ones (Anthropic 2025).
- **progressive disclosure** — three-level loading: metadata always loaded, instructions loaded on trigger, resources loaded on demand.
- **dual-register description** — expert terminology (for routing depth) paired with natural language (for trigger breadth).
- **15-year practitioner test** — litmus: would a senior practitioner with 15+ years experience use this term with a peer?
- **named anti-patterns** — specific names activate expert knowledge; unnamed problems get generic responses.
- **BECAUSE clause** — the reasoning that makes rules generalisable to adjacent cases. "Do X because Y" > "ALWAYS do X."
- **constraint pairing** — positive instruction + negative constraint together are strongest (Zamfirescu-Pereira et al. 2023).
- **few-shot demonstration** — 3 well-chosen examples match 9 in effectiveness (LangChain 2024). Quality over quantity.
- **retrieval anchors** — natural-language questions at the end of context that benefit from recency and end-of-context attention.
- **effort budget** — explicit tool-call expectations; agents cannot judge appropriate effort without guidance.
- **output template** — a forcing function that distributes agent effort across sections rather than allowing rush-to-completion.
- **right-altitude prompting** — 5–15 precise requirements is the sweet spot. More than 15 degrades compliance.
- **n=19 cliff** — at 19+ requirements, accuracy drops below a prompt with just 5 requirements (Vaarta Analytics 2026).
- **attention valley** — the middle-of-context zone where information suffers 30%+ accuracy degradation.
- **identity construction** — brief identities under 50 tokens using real job titles outperform elaborate personas (PRISM 2024).
- **flattery degradation** — superlatives ("world-class expert") activate marketing text, degrading technical output.
- **structured format** — YAML headers, numbered lists, and delimited sections outperform prose walls (MetaGPT, Voyce 2025).
- **copy-paste checklist** — visible step tracking that makes step-skipping harder to rationalise.
- **section ordering** — the deliberate placement of content based on position-dependent attention allocation.
- **trigger breadth** — natural-language coverage ensuring the skill is found by both precise and fuzzy queries.

## Anti-Patterns

### The Flattery Trap
**Detect:** Superlatives in identity ("world-class expert").
**BECAUSE:** PRISM (2024) — flattery activates marketing text, degrading technical output.
**Resolve:** <50 token identity, real job title, no adjectives.

### The Middle Drop
**Detect:** Critical constraints placed after procedural steps.
**BECAUSE:** Liu et al. (2024) — 30%+ accuracy drop for middle-of-context information.
**Resolve:** Constraints at top, procedure in middle, anchors at bottom.

### Generic Vocabulary
**Detect:** General terms where domain equivalents exist ("review for security" vs. "OWASP Top 10 audit").
**BECAUSE:** Ranjan et al. (2024) — generic terms route to blog-level knowledge; specific terms route to expert clusters.
**Resolve:** 15–30 domain terms, each passing the 15-year practitioner test.

### Requirements Bloat
**Detect:** More than 15 requirements in a single prompt.
**BECAUSE:** Vaarta Analytics (2026) — at n=19, accuracy drops below n=5.
**Resolve:** 5–15 requirements; move long tail to reference files.

### Missing BECAUSE Clauses
**Detect:** ALWAYS/NEVER rules without reasoning.
**BECAUSE:** Rules without reasons can't generalise to adjacent cases and become dead weight.
**Resolve:** Every imperative gets a BECAUSE clause. If you can't explain why, the rule may be unnecessary.

### Example Starvation
**Detect:** Prompts with rules but zero concrete examples.
**BECAUSE:** LangChain (2024) — 3 well-chosen examples match 9 in effectiveness.
**Resolve:** At least 2 BAD vs GOOD pairs, concrete content, best GOOD example last.

### Uniform Constraint Level
**Detect:** Same specificity across all sections regardless of fragility.
**BECAUSE:** Fragile ops need exact steps; creative work needs principles. Uniform medium degrades both.
**Resolve:** LOW for fragile ops, MEDIUM for templates, HIGH for creative work.

### Unnamed Problems
**Detect:** Describing a mistake without naming it ("don't load too much" vs. "The Eager-Loading Trap").
**BECAUSE:** Named concepts activate expert knowledge clusters; unnamed problems get generic responses.
**Resolve:** Use name → detect → BECAUSE → resolve → prevent pattern.

## Checklist

Copy this checklist and track your progress when authoring or revising a
prompt or skill:

```
Prompt Engineering Checklist

 ## Vocabulary
- [ ] 15–30 domain-specific terms included
- [ ] Each term passes the 15-year practitioner test
- [ ] No general-purpose terms the model already knows
- [ ] Vocabulary section placed FIRST in the prompt body

 ## Identity
- [ ] Identity is under 50 tokens
- [ ] Uses a real-world job title (no superlatives, no flattery)
- [ ] Structured format (YAML headers preferred over prose)

 ## Constraints
- [ ] At least one ALWAYS rule with BECAUSE clause
- [ ] At least one NEVER rule with BECAUSE clause
- [ ] Positive + negative constraints paired together
- [ ] 5–15 total requirements (not 19+)

 ## Anti-Patterns
- [ ] 5–10 named anti-patterns
- [ ] Each has Detect, BECAUSE, and Resolve fields
- [ ] Names are specific and memorable

 ## Structure (U-shaped attention curve)
- [ ] Vocabulary payload at the TOP
- [ ] Constraints and anti-patterns near the TOP
- [ ] Procedure in the MIDDLE (numbered steps)
- [ ] Examples near the BOTTOM (best example last)
- [ ] Retrieval anchors at the BOTTOM

 ## Examples

See [examples-prompt-engineering.md](references/examples-prompt-engineering.md) for worked prompt construction examples: no-vocabulary-routing anti-pattern, vocabulary-routed prompt, flattery-identity anti-pattern, and brief real-world identity.

## Evaluation Criteria

When evaluating a prompt or skill authored with this procedure, score
each dimension on a 0.0–1.0 scale:

- **Vocabulary (0.0–1.0):** 15–30 domain-specific terms present, each
  passing the 15-year practitioner test, no general terms the model
  already knows, placed first in the prompt body
- **Identity (0.0–1.0):** Under 50 tokens, real-world job title, no
  superlatives or flattery, structured format (YAML)
- **Constraints (0.0–1.0):** At least one ALWAYS/NEVER pair, every
  constraint has a BECAUSE clause, 5–15 total requirements
- **Anti-Patterns (0.0–1.0):** 5–10 named anti-patterns, each with
  Detect/BECAUSE/Resolve, names are specific and memorable
- **Structure (0.0–1.0):** Vocabulary at top, constraints near top,
  procedure in middle, examples near bottom, retrieval anchors last
- **Examples (0.0–1.0):** At least 2 BAD vs GOOD pairs, concrete
  realistic content, best GOOD example last, each explains WHY
- **Output Format (0.0–1.0):** Required sections defined with
  descriptions, template provided, acceptance criteria listed
- **Retrieval Anchors (0.0–1.0):** 5–10 natural-language questions at
  end, questions are specific to the prompt's domain

A score below 0.7 on any dimension requires a specific rewrite
suggestion naming the relevant anti-pattern.

## Questions This Skill Answers

- How do I write a prompt that activates expert knowledge instead of
  generic advice?
- What section ordering maximises agent attention and recall?
- How many domain terms should I include in a prompt?
- What makes an anti-pattern effective vs. just another rule?
- Why do examples matter more than rules?
- How do I structure the identity section without flattery?
- What's the right number of constraints for a prompt?
- How do I write a BECAUSE clause that generalises?
- Where should I place retrieval anchors in the prompt?
- How do I evaluate whether an existing prompt is well-engineered?

## References

For deeper background on specific topics:

- **Attention curve deep dive:** See `references/attention-curve.md` for
  the full Liu et al. (2024) and Wu et al. (2025) findings, with diagrams
  showing the accuracy drop across context positions.
- **Vocabulary routing deep dive:** See `references/vocabulary-routing.md`
  for the Ranjan et al. (2024) methodology, the 15-year practitioner test
  with examples, and guidance on selecting domain terms for any field.
- **Full annotated examples:** See `references/full-examples.md` for
  complete before/after prompt transformations with detailed annotations
  explaining each change in terms of the underlying research.
