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
constraint_level: medium
---

# SKILL: Prompt Engineering

## Purpose

Write, revise, and evaluate prompts and skills using the research-backed
10 principles from `refs/prompt-engineering-guide.md`. This skill applies
the U-shaped attention curve, vocabulary routing, constraint pairing,
named anti-patterns, and few-shot demonstration patterns to produce
prompts that route to expert knowledge clusters.

## Vocabulary

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
- [ ] At least 2 BAD vs GOOD example pairs
- [ ] Examples use concrete, realistic content
- [ ] Best GOOD example appears last (recency bias)
- [ ] Each example explains WHY it is good or bad

## Output Format
- [ ] Required sections defined with descriptions
- [ ] Template provided for the expected output
- [ ] Acceptance criteria or evaluation dimensions listed

## Retrieval Anchors
- [ ] 5–10 natural-language questions at the end
- [ ] Questions are specific to the prompt's domain
```

## Procedure

### When authoring a new prompt or skill

1. **Determine the task structure and constraint level.** Is this a
   fragile operation requiring exact steps (LOW freedom), structured work
   with a template (MEDIUM), or creative work with multiple valid
   approaches (HIGH)? Match the procedure's specificity to this level.

2. **Draft the vocabulary payload.** List 15–30 domain terms. Apply the
   15-year practitioner test: would a senior expert with 15+ years in
   this domain use this exact term with a peer? Strip general terms the
   model already knows. THIS GOES FIRST in the prompt body.

3. **Write identity.** Under 50 tokens. Real job title. No flattery, no
   superlatives, no elaborate backstory. Use structured format (YAML).

4. **Write constraints as ALWAYS/NEVER pairs.** Pair each positive
   instruction with a negative constraint. Every imperative gets a
   BECAUSE clause. Place constraints near the top of the prompt.

5. **Name the anti-patterns.** For each likely mistake, create an entry
   with: a specific name, how to detect it, BECAUSE (consequence chain),
   and how to resolve it. Place after constraints.

6. **Write the procedure.** 5–10 imperative steps. Use numbered format
   (survives attention degradation). Include IF/THEN branches for
   condition-dependent paths. Place in the middle.

7. **Define the output format.** Specify required sections, minimum
   content expectations, and cross-reference rules. Match strictness to
   constraint level.

8. **Write 2–3 BAD vs GOOD example pairs.** Use concrete, realistic
   content. Explain why each is good or bad. Place the best GOOD example
   last (recency bias).

9. **Add retrieval anchors.** 5–10 natural-language questions that this
   prompt answers. Place at the very end.

10. **Add a copy-paste checklist.** Place near the top (after vocabulary,
    before procedure) so agents see it early and can track progress.

### When evaluating an existing prompt or skill

1. Read the prompt in full. Identify its task structure and intended
   constraint level.
2. Check each dimension using the checklist above: vocabulary, identity,
   constraints, anti-patterns, structure, examples, output format,
   retrieval anchors.
3. For each violation, name the anti-pattern and explain BECAUSE.
4. Produce a scored verdict (0.0–1.0 per dimension) with specific
   rewrite suggestions for dimensions scoring below 0.7.

## Output Format

### For prompt authoring output

When you complete this procedure, produce a prompt following this template
(from `refs/prompt-engineering-guide.md`):

```
# [Role: brief, real-world title] (<50 tokens)

You are a senior [specific domain] engineer.

## Vocabulary

[15-30 domain-specific terms that route to expert knowledge]

Terms: [term1], [term2], [term3], ...

## Constraints

- ALWAYS [X] BECAUSE [Y]
- NEVER [X] BECAUSE [Y]
- ...

## Anti-Patterns

- **[Named Anti-Pattern]**: [detection signal] → [resolution]
- **[Named Anti-Pattern]**: [detection signal] → [resolution]

## Task

[Clear objective with explicit scope boundaries]

Expected effort: [N–M] tool calls.
Use tools: [specific subset relevant to this role]
Do NOT use: [tools irrelevant to this role]

## Procedure

1. [First step — imperative verb]
2. [Second step]
3. IF [condition] THEN [action] ELSE [alternative]
4. ...

## Output Format

[Exact structure of expected output with required sections]

## Examples

### Bad

[Concrete example of wrong output — and why it is wrong]

### Good

[Concrete example of correct output — and why it is correct]

### Bad

[Another wrong example]

### Good

[Another correct example]

## Retrieval Anchors

Questions this prompt answers:

- [Question 1]?
- [Question 2]?
- [Question 3]?
```

### For prompt evaluation output

```
## Prompt Evaluation: [prompt name]

| Dimension | Score | Notes |
|-----------|-------|-------|
| Vocabulary | 0.0–1.0 | [specific gap] |
| Identity | 0.0–1.0 | [specific issue] |
| Constraints | 0.0–1.0 | [specific weakness] |
| Anti-patterns | 0.0–1.0 | [missing patterns] |
| Structure (U-curve) | 0.0–1.0 | [ordering violations] |
| Examples | 0.0–1.0 | [example quality] |
| Output Format | 0.0–1.0 | [template issues] |
| Retrieval Anchors | 0.0–1.0 | [anchor quality] |

## Violations

[For each dimension scoring below 0.7: named anti-pattern, BECAUSE, and
specific rewrite suggestion]
```

## Examples

### BAD: A prompt without vocabulary routing or U-curve structure

```
You are an AI assistant that helps with code. You should be helpful and
careful. Please review the following code for any problems. Look for bugs,
security issues, and performance problems. Also check for style issues and
make sure it follows best practices. The code is in Python and uses Flask.
```

**Why this is bad:**
- **No vocabulary routing.** "Review for problems" routes to generic
  advice. No domain terms (OWASP, CWE, STRIDE, WSGI, SQLAlchemy session
  management, etc.) means no expert knowledge clusters are activated.
- **Flattery-adjacent identity.** "You are an AI assistant that helps" is
  generic and provides no professional stance.
- **No anti-patterns.** No named mistakes to watch for, so the agent
  applies its default (generic) review patterns.
- **No section ordering.** Everything is in one blob. No U-curve
  consideration.
- **No effort budget.** The agent could spend 1 tool call or 50.
- **No output format.** Freeform review produces inconsistent results.

### GOOD: The same task with vocabulary routing and U-curve structure

```yaml
# Security Code Reviewer

You are a senior application security engineer specialising in Python web
applications.

## Vocabulary

OWASP Top 10, STRIDE threat modelling, CWE-89 (SQL injection), CWE-79
(XSS), CWE-352 (CSRF), CWE-22 (path traversal), parameterised queries,
input validation boundary, ORM injection surface, session fixation,
content security policy, CORS misconfiguration, JWT validation,
dependency confusion, prototype pollution, mass assignment.

## Constraints

- ALWAYS trace user-controlled input to every sink BECAUSE a single
  missed taint path is a potential CWE-89 or CWE-79 vulnerability
- NEVER recommend string concatenation for SQL BECAUSE it enables
  injection across all database dialects, not just the one in use
- ALWAYS verify authorisation at every endpoint BECAUSE Flask's default
  routing has no built-in access control

## Anti-Patterns

- **The ORM Trust Fallacy**: assuming ORMs prevent injection → verify
  raw SQL and dynamic filter construction
- **The Decorator Mirage**: assuming route decorators enforce auth →
  check every endpoint for actual auth verification

## Task

Review the attached Flask application for security vulnerabilities.

Expected effort: 8–12 tool calls.
Use tools: read_file, grep, search_graph.
Do NOT use: decompose, entity, retro.

## Procedure

1. Read every route handler and trace user input to sinks
2. Check auth on each endpoint individually
3. IF input reaches SQL THEN verify parameterised query
4. IF input reaches HTML response THEN verify escaping context
5. Check dependency versions against known CVEs

## Output Format

| Endpoint | Method | Auth? | Input Validation | SQL Safe? | XSS Safe? | Notes |
```

**Why this is good:**
- **Vocabulary:** 16 domain terms route to security engineering knowledge.
- **Identity:** Under 50 tokens. Real job title. No flattery.
- **Constraints:** ALWAYS/NEVER pairs with BECAUSE clauses.
- **Anti-patterns:** Named, specific, with detection signals.
- **Procedure:** Numbered steps with IF/THEN branching.
- **Output format:** Structured table forces engagement with each
  dimension.
- **Effort budget:** Explicit 8–12 tool calls.

### BAD: Identity with flattery

```
You are an extraordinarily talented, world-class software architect with
decades of experience across every major programming language. You are
known for your brilliant insights, exceptional attention to detail, and
remarkable ability to solve the most complex problems with elegant
solutions.
```

**Why this is bad:**
- 60+ tokens of identity.
- "World-class", "extraordinarily talented", "brilliant insights" are
  all flattery terms that activate marketing/motivational text.
- "Every major programming language" is too broad — no routing to
  specific knowledge clusters.

### GOOD: Brief, real-world identity

```
You are a senior Go backend engineer specialising in concurrent systems
and API design.
```

**Why this is good:**
- Under 50 tokens (18 words).
- Real job title with specific specialisation.
- "Concurrent systems" and "API design" route to relevant knowledge.

## Retrieval Anchors

Questions this skill answers:

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
