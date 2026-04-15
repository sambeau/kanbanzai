# Prompt Engineering for AI Agents

Research-backed guide for writing effective prompts. Referenced from
[AGENTS.md](../AGENTS.md).

Based on the "10 Claude Code Principles" (distilled from 17 peer-reviewed papers) and
orchestration research from Anthropic, Google, MetaGPT, SWE-agent, and Masters et al.
Full analysis in [`work/research/ai-agent-best-practices-research.md`](../work/research/ai-agent-best-practices-research.md)
and [`work/research/agent-orchestration-research.md`](../work/research/agent-orchestration-research.md).

---

## The one-sentence summary

> The words you choose, where you place them, and what rules you surface are more important
> than how many agents you deploy or how long your prompts are.

---

## 1. Identity: brief, real, no flattery

The PRISM persona framework (2024) is definitive: **brief identities (<50 tokens) using real
job titles outperform elaborate personas**. Flattery actively degrades output by activating
marketing and motivational text from training data.

The **15-year practitioner test** (Ranjan et al., 2024): would a senior expert with 15+ years
of domain experience use this exact term when talking with a peer? If yes, include it. If it
sounds like a job ad, cut it.

**Good — compact, real-world, specific:**

> You are a senior Go backend engineer specialising in concurrent systems and API design.

**Bad — elaborate, flattering, generic:**

> You are an extraordinarily talented, world-class software architect with decades of
> experience across every major programming language. You are known for your brilliant
> insights and exceptional attention to detail.

Structured machine-readable metadata (YAML headers, JSON) outperforms prose for identity
declaration because it is unambiguous and compact. But vocabulary matters more than format.

---

## 2. Vocabulary routing: the #1 quality lever

This is the headline finding. Ranjan et al. (2024) showed that **specific vocabulary acts as a
routing signal that determines which knowledge clusters the model activates**.

Include **15–30 precise domain terms** per prompt or skill. These are not decoration — they are
the primary mechanism by which the model accesses deeper, more specialised knowledge.

**Bad — generic vocabulary:**

> Review this code for security issues.

Routes to blog posts and generic security checklists.

**Good — domain-specific vocabulary:**

> Perform an OWASP Top 10 audit. Apply STRIDE threat modelling to all API endpoints. Check for
> CWE-89 (SQL injection), CWE-79 (XSS), and CWE-352 (CSRF). Verify input validation at trust
> boundaries. Confirm parameterised queries throughout.

Routes to security engineering knowledge clusters.

---

## 3. Structure for the U-shaped attention curve

Liu et al. (2024) and Wu et al. (2025) demonstrated that LLMs pay the most attention to content
at the **beginning** and **end** of context, with a **30%+ accuracy drop** for content in the
middle. Voyce (2025) showed prompt format alone accounts for up to **40% performance variance**.

### Attention-optimal section ordering

```
Position        Attention    What to put here
─────────────────────────────────────────────────────────
Top             HIGH         Identity + vocabulary payload (routing signal)
Near top        HIGH         Anti-patterns + constraints
Middle          LOW          Behavioural instructions (numbered steps survive)
Near bottom     RISING       Output format + examples (recency bias helps)
Bottom          HIGH         Retrieval anchors
```

This is why numbered imperative steps go in the middle — they are inherently more resilient to
attention degradation than prose. Critical constraints go at the top where attention is
strongest. Retrieval anchors go at the bottom where recency bias gives them a boost.

---

## 4. Constraints: always/never with reasons

Zamfirescu-Pereira et al. (CHI 2023) found that **combined positive instruction + negative
constraint is the strongest approach**. The optimal format is **"Always/Never X BECAUSE Y"** —
the BECAUSE clause is what makes rules generalisable to adjacent cases.

**Good — compact + generalisable:**

> NEVER use string concatenation for SQL queries BECAUSE it enables injection attacks across
> all SQL dialects.

**Bad — verbose + vague:**

> When writing database queries, you should always be careful about SQL injection. Make sure to
> use parameterised queries whenever possible. This is especially important when dealing with
> user input from web forms...

Rules without reasons become dead weight — they cannot be pruned because no one knows why they
were added.

---

## 5. Anti-patterns: name them

The research shows that **named anti-patterns activate expert knowledge clusters** while unnamed
problems get generic responses. Use the detect → name → explain → resolve → prevent pattern.

**Good:**

> **The Eager-Loading Trap**: loading all related records upfront when only a subset is needed.
> Detect: N+1 queries or bulk SELECT without LIMIT. Resolve: use lazy loading or explicit
> pagination.

**Bad:**

> Don't load too much data.

---

## 6. Less is more — but *structured* less

The research converges from multiple angles:

| Finding | Source |
|---------|--------|
| 15–40% context utilisation is the sweet spot | Anthropic (Sep 2025) |
| <50 token identities beat 200+ token personas | PRISM (2024) |
| 3 well-chosen examples match 9 in effectiveness | LangChain (2024) |
| At n=19 requirements, accuracy drops below n=5 | Vaarta Analytics (2026) |
| 4 agents beat 7+ (diminishing returns at 3) | DeepMind (2025) |

**Right-altitude prompting**: specific enough to route to expert knowledge, concise enough not
to overwhelm the attention budget. The sweet spot is **5–15 precise requirements**, not 19
fuzzy ones.

---

## 7. Examples: few-shot demonstrations, not decoration

LangChain (2024) showed that **3 well-chosen examples match 9** in effectiveness. Use 2–3
BAD vs GOOD pairs. Place them near the bottom of the prompt to benefit from recency bias.

The last example in a sequence has disproportionate influence. Structure so the most critical
convention appears last in each section.

---

## 8. Effort expectations

Anthropic's multi-agent research found that **agents struggle to judge appropriate effort**.
Embed explicit expectations:

> This specification task should involve 5–15 tool calls before producing output. Read the
> design document, query relevant knowledge entries, check for related decisions, and draft
> a structured specification. Do not proceed to implementation.

Without this, agents will allocate minimal effort to specification and maximum effort to
implementation — because writing code *feels* productive while writing documents does not.

---

## 9. Tool scoping

Every idle tool definition in a prompt consumes attention budget and degrades performance on
active tools. Google's research shows coordination overhead grows with tool count.

When delegating to sub-agents:

- Include only the tool definitions the agent will actually use
- A review sub-agent does not need `decompose`
- An implementation sub-agent does not need `retro`
- Explicitly state what tools to use and what to avoid

---

## 10. Output format

Agents produce consistent output when the format is defined. Freeform output without structured
templates is the root cause of inconsistent specifications and plans (MetaGPT, Microsoft,
Anthropic).

Define:

- Required sections with descriptions
- Minimum content expectations per section
- Cross-reference requirements
- Acceptance criteria format (must be testable/verifiable)

The template itself acts as a forcing function — agents must engage with each section, which
distributes effort across the output rather than allowing rush-to-completion.

---

## Quick-reference checklist

### Do

1. Lead with a brief, real-world identity (<50 tokens, real job title)
2. Front-load 15–30 domain vocabulary terms (the routing signal)
3. Name your anti-patterns — specific names activate expert knowledge
4. Use "Always/Never X BECAUSE Y" for constraints
5. Structure for the U-curve — constraints top, steps middle, anchors bottom
6. Give 2–3 BAD vs GOOD examples
7. Specify output format explicitly
8. Keep requirements to 5–15
9. Use structured formats (YAML headers, numbered lists, delimited sections)
10. Embed effort expectations ("expect 5–15 tool calls")

### Don't

1. Use flattery — "world-class expert" routes to marketing text
2. Write walls of prose — structure beats volume
3. Put critical information in the middle — it falls into the attention valley
4. Give 19 requirements when 7 will do — quality collapses at scale
5. Use generic vocabulary — "review for security" vs "OWASP Top 10 audit"
6. Skip negative constraints — positive + negative together is strongest
7. Build elaborate backstories — <50 tokens of identity beats 200+
8. Assume context persists — design every prompt for single-session completeness

---

## Template

Combining all findings into a single reference template.

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

### Section ordering rationale

| Position | Attention level | Section | Why here |
|----------|-----------------|---------|----------|
| Top | High | Identity + vocabulary | Routing signal — determines which knowledge clusters activate |
| Near top | High | Constraints + anti-patterns | Hard rules benefit from peak attention |
| Middle | Lower | Procedure | Numbered steps survive attention degradation |
| Near bottom | Rising | Output format + examples | Recency bias improves pattern matching |
| Bottom | High | Retrieval anchors | Benefit from recency bias and end-of-context attention |

---

## Applying this to Kanbanzai roles and skills

Kanbanzai's role + skill system is architecturally well-aligned with this research. Roles
define *who you are* (identity, vocabulary, anti-patterns). Skills define *what you are doing
right now* (procedure, output format, examples). Together they cover the full template.

The recommended skill architecture from the research:

```
skill-name/
├── SKILL.md (<500 lines)
│   ├── YAML frontmatter (name + dual-register description, ~100 words)
│   ├── Expert Vocabulary Payload (FIRST in body — routing signal)
│   ├── Anti-Pattern Watchlist (BEFORE behavioural instructions)
│   ├── Behavioural Instructions (ordered imperative steps with IF/THEN)
│   ├── Output Format
│   ├── Examples (2–3 BAD vs GOOD pairs)
│   └── Questions This Skill Answers (at END — retrieval anchors)
└── references/
    ├── anti-patterns-full.md
    ├── frameworks.md
    ├── evaluation-criteria.md
    └── checklists.md
```

**Dual-register descriptions** use expert terminology for routing depth alongside natural
language for trigger breadth. This ensures the skill is found by both precise queries and
fuzzy ones.

---

## Key research sources

| Source | Year | Key contribution |
|--------|------|------------------|
| Vaswani et al., "Attention Is All You Need" | 2017 | Transformer architecture, n² pairwise attention |
| Zamfirescu-Pereira et al., "Why Johnny Can't Prompt" | 2023 | Positive + negative constraints strongest together |
| Hong et al., MetaGPT | 2023 | Structured artefacts reduce errors ~40% |
| Liu et al., "Lost in the Middle" | 2024 | 30%+ accuracy drop for middle-of-context information |
| Ranjan et al., "One Word Is Not Enough" | 2024 | Vocabulary specificity routes to domain knowledge |
| PRISM Persona Framework | 2024 | <50 token identities optimal; flattery degrades output |
| LangChain Few-Shot Research | 2024 | 3 well-chosen examples match 9 |
| Wu et al., MIT Position Bias | 2025 | Causal masking and RoPE cause U-shaped attention |
| DeepMind Multi-Agent Scaling | 2025 | Saturation at 4 agents; superlinear coordination costs |
| Voyce, XML/Markdown Comparative Study | 2025 | Format alone accounts for up to 40% performance variance |
| Anthropic, "Effective Context Engineering" | 2025 | Attention budget, progressive disclosure |
| Vaarta Analytics, "Prompt Engineering Is System Design" | 2026 | At n=19 requirements, accuracy drops below n=5 |