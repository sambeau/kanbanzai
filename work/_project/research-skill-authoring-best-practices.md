# Skill Authoring Best Practices: Research-Backed Guide

| Field | Value |
|-------|-------|
| Date | 2025-07-29 |
| Author | Synthesised from three research reports |
| Status | Reference |
| Sources | ai-agent-best-practices-research.md, agent-skills-research.md, agent-orchestration-research.md |

---

## Purpose

This document distils the research-backed principles for authoring effective agent
skills and designing features that require them. It is intended for any agent or human
working on the Kanbanzai skill system — whether creating a new skill, revising an
existing one, or designing a feature whose workflow will be encoded as a skill.

Every recommendation below is traceable to at least one academic paper or rigorously
evaluated industry source. Where claims rest on weaker evidence, this is noted.

---

## Part 1: The Science Behind Skill Design

### 1.1 The Attention Curve Determines Section Ordering

Transformers allocate attention unevenly across context. Two landmark findings shape
how skill content should be ordered:

- **U-shaped attention** (Liu et al., "Lost in the Middle", 2024): accuracy drops
  30%+ when critical information sits in the middle of the context window. Content at
  the beginning and end receives disproportionate attention due to causal masking and
  RoPE positional encoding (Wu et al., MIT, 2025).
- **Recency bias**: the last item in a sequence has outsized influence on generation.
  The best example in a skill should appear last.

**Implication for skills:** Place the vocabulary payload (routing signal) first in the
body, place retrieval anchors ("Questions This Skill Answers") last, and accept that
procedural steps in the middle will receive less attention — structure them as numbered
lists so they survive attention degradation.

**Evidence strength:** Strong. Two independent peer-reviewed studies plus Anthropic's
"Effective Context Engineering" (Sep 2025) confirm the pattern.

### 1.2 Vocabulary Routing Is the Primary Quality Lever

The words in a prompt determine which knowledge clusters the model activates. This is
not a metaphor — it is a measurable effect:

- **Vocabulary specificity** (Ranjan et al., "One Word Is Not Enough", 2024):
  domain-specific terminology activates specialised knowledge regions. "OWASP Top 10
  audit, STRIDE threat model" routes to security engineering knowledge. "Review the
  security" routes to blog-post-level advice.
- **The 15-year practitioner test** (derived from PRISM, 2024): would a senior expert
  with 15+ years of domain experience use this exact term when speaking with a peer?
  If yes, it belongs in the vocabulary payload.
- **Brief identities** (PRISM, 2024): persona descriptions under 50 tokens produce
  higher-quality output than elaborate personas over 100 tokens. Flattery and
  superlatives ("world-class expert") degrade output by activating
  motivational/marketing text.

**Implication for skills:** Every skill needs 15–30 precise domain terms as its first
body section. These terms are not definitions for the reader — they are routing signals
that prime the model. Strip general terms the model already knows. Include only terms
that activate the right knowledge cluster for the task.

**Evidence strength:** Strong. Peer-reviewed (Ranjan et al.) plus validated framework
(PRISM).

### 1.3 Context Is Scarce — Treat It Like Embedded Memory

- **The attention budget** (Anthropic, Sep 2025): every token in the context window
  competes with every other token for attention weight. Irrelevant tokens actively
  degrade performance on the relevant ones.
- **Optimal utilisation zone**: 15–40% of the context window. Below ~10%,
  hallucination risk increases (insufficient grounding). Above ~60%, attention
  dilution dominates.
- **Three-level progressive disclosure** (Anthropic platform docs, 2025):
  - Level 1 (Metadata): always loaded. Name + description. ~100 tokens per skill.
  - Level 2 (Instructions): loaded when the skill triggers. The SKILL.md body. Under
    500 lines.
  - Level 3 (Resources): loaded on demand. Reference files, scripts, assets.
    Effectively free until accessed.

**Implication for skills:** Every token must justify its presence. Strip explanations
of concepts the model already knows. Move detailed reference material to Level 3 files.
Keep SKILL.md as a routing document that tells the agent where to look.

**Evidence strength:** Strong. Anthropic's engineering guidance is empirically derived
from production systems. The optimal utilisation zone is consistent across multiple
sources.

### 1.4 Conciseness Is a Design Constraint

> "Claude is already very smart — only add context Claude doesn't already have."
> — Anthropic, Skill Authoring Best Practices

The litmus test for every paragraph in a skill:

1. Does the model really need this explanation?
2. Can I assume the model already knows this?
3. Does this paragraph justify its token cost?

**The n=19 cliff** (Vaarta Analytics, 2026): when a prompt contains 19 or more
requirements, accuracy drops below a prompt with just 5 requirements. Adding more
rules past a threshold actively hurts compliance.

**Implication for skills:** Keep the procedure lean. 5–10 steps, not 20. If a skill
needs extensive rules, move the long tail to reference files and keep the procedure
focused on the critical path.

**Evidence strength:** The n=19 finding is from industry research (Vaarta Analytics),
not peer-reviewed, but corroborated by the general "less is more" pattern across
multiple academic sources (PRISM, DeepMind scaling).

### 1.5 Examples Beat Rules

- **3 well-chosen examples match 9** in effectiveness (LangChain few-shot research,
  2024). The quality and relevance of examples matters far more than quantity.
- **Prompt format alone accounts for up to 40% performance variance** (Voyce, XML/
  Markdown comparative study, 2025). Structure matters as much as content.
- **Input/output examples** "train Claude's understanding better than abstract
  instructions" (Anthropic, Increase Consistency guide).

**Implication for skills:** Every skill should include at least one BAD/GOOD example
pair. Place the best GOOD example last (recency bias). Keep examples concrete — use
real-looking content, not "example placeholder here."

**Evidence strength:** Moderate-to-strong. The LangChain research is industry-sourced
but widely replicated. The Voyce study is a comparative evaluation, not peer-reviewed.

### 1.6 Enforceable Constraints Beat Advisory Instructions

Every source that compares "telling agents what to do" with "preventing them from doing
the wrong thing" finds the latter wins decisively:

- **MetaGPT** (Hong et al., ICLR 2024): structured intermediate artifacts with
  verification gates reduced cascading hallucination errors by ~40% versus free
  dialogue.
- **Masters et al.** (DAI 2025): formalises constraints as hard (ℋ — violation
  terminates workflow) versus soft (𝒮 — violation incurs penalty). The most reliable
  workflows use hard constraints for critical ordering.
- **Google Research** (Kim & Liu, 2026): centralised orchestration with validation
  bottleneck contained error amplification to 4.4× versus 17.2× for independent
  agents.

**Implication for skills:** Where a procedure must be followed exactly (lifecycle
transitions, document registration, stage gates), encode it as a deterministic tool or
script — not as instructions the agent follows on its honour. Skills should have low
constraint_level for fragile operations and high constraint_level for creative work.

**Evidence strength:** Strong. Multiple peer-reviewed sources converge independently.

### 1.7 Decomposition Quality Is the Critical Path

- **Masters et al.** (DAI 2025): "Performance gains correlate almost linearly with
  the quality of the induced task graph — underlining that structure learning, not raw
  language generation, is the critical path."
- **Google Research** (2026): the predictive model's strongest feature was
  "decomposability" — whether the task can be cleanly split into independent
  sub-tasks.
- **The sequential penalty** (Google, 2026): on tasks requiring strict sequential
  reasoning (specification, design, planning), every multi-agent variant tested
  degraded performance by 39–70%.

**Implication for skills and features:** When designing a feature that will require
skills, the decomposition of the feature into workflow stages matters more than the
quality of any individual stage's instructions. Sequential stages (specification,
design) should never be parallelised. Parallelisable stages (implementation across
files) benefit from orchestrator-worker patterns.

**Evidence strength:** Strong. Two independent academic sources confirm the finding.

### 1.8 Explain Why, Not Just What

> "Try to explain to the model why things are important in lieu of heavy-handed musty
> MUSTs. Use theory of mind and try to make the skill general and not super-narrow to
> specific examples."
> — Anthropic, skill-creator meta-skill

- **Combined positive + negative constraints** are the strongest approach
  (Zamfirescu-Pereira et al., CHI 2023, "Why Johnny Can't Prompt").
- **The BECAUSE clause** is what makes rules generalisable. "Always X" covers one
  case. "Do X because Y" generalises correctly to adjacent cases.
- **Named anti-patterns** activate expert knowledge clusters (derived from Ranjan et
  al., 2024). "The eager-loading trap" routes to specific knowledge; "don't load too
  much" routes to generic advice.

**Implication for skills:** Every anti-pattern needs a BECAUSE clause that explains the
consequence chain, not just a restatement of the detection signal. Prefer explanatory
tone over authoritarian imperatives. "Do X because Y" > "ALWAYS do X."

**Evidence strength:** Strong for the CHI 2023 finding. The named anti-pattern effect
is inferred from the vocabulary routing research — plausible but not directly tested.

### 1.9 Feedback Loops and Checklists Prevent Skipping

- **Copy-paste checklists** that agents track in their responses make step-skipping
  visible and harder to rationalise (Anthropic, Best Practices guide).
- **Validate → fix → repeat loops** improve output quality significantly (Anthropic,
  Best Practices guide).
- **Maker-checker patterns** with explicit acceptance criteria produce consistent
  pass/fail decisions (Microsoft, Agent Orchestration Patterns, 2026).

**Implication for skills:** Workflow-critical skills should include checklists agents
can copy. Stage gate skills should use validate-fix-repeat loops. The feedback loop
should be a tool call (deterministic), not a self-assessment (unreliable).

**Evidence strength:** Moderate. Based on Anthropic's and Microsoft's engineering
practices, not peer-reviewed studies. However, consistent with MetaGPT's verification
gate findings which are peer-reviewed.

### 1.10 Match Constraint Level to Task Risk

The "narrow bridge vs. open field" analogy from Anthropic's best practices:

| Constraint Level | When to Use | Examples |
|---|---|---|
| **Low** (exact steps) | Fragile operations; wrong execution is expensive to undo | Lifecycle transitions, document registration, stage gates |
| **Medium** (templates) | Preferred pattern exists, some variation acceptable | Specification writing, plan creation, structured review |
| **High** (guidance) | Multiple valid approaches; context-dependent | Design work, implementation, creative problem-solving |

**Implication for skills:** Every skill must declare its constraint_level in frontmatter
and match its procedure style to that level. Low-constraint skills use exact tool call
sequences. High-constraint skills use principles and guidance.

**Evidence strength:** Moderate. Anthropic's framework is internally consistent and
used in production. The specific thresholds are engineering judgment, not measured.

---

## Part 2: Designing Features That Will Require Skills

When designing a new feature whose workflow will be encoded as one or more skills,
apply these research-backed principles at design time — before any skill is written.

### 2.1 Map the Task Structure Before Choosing an Orchestration Pattern

Different task structures require different orchestration patterns. Using the wrong
pattern is a structural cause of quality degradation (Google Research, 2026):

| Task Structure | Optimal Pattern | Agent Count | Kanbanzai Example |
|---|---|---|---|
| Sequential, low tool density | Single agent, no parallelism | 1 | Specification writing |
| Evaluative, independent criteria | Maker-checker or specialist panel | 1–3 | Code review, plan review |
| Parallelisable, high tool density | Orchestrator-workers | 1 + N workers | Multi-file implementation |
| Sequential with decision points | Single agent + human gates | 1 + human | Design work |

**Key rule:** specification and design work should never be parallelised. Implementation
across independent files can be. Applying the parallel pattern to sequential work
degrades performance by 39–70% (Google Research, 2026).

### 2.2 Identify Hard vs. Soft Constraints

At design time, classify every constraint the feature introduces:

- **Hard constraints (ℋ):** must always hold; violation should be system-enforced.
  Examples: lifecycle state must be correct before transition; specification must exist
  before implementation tasks are created; document must be approved before stage gate
  passes.
- **Soft constraints (𝒮):** desirable but violable with penalties. Examples: code
  style conformance, test coverage targets, documentation quality.

Hard constraints should become tool-level enforcement (the tool refuses invalid
operations). Soft constraints should become review criteria. Skills should encode both
but distinguish them clearly.

### 2.3 Define Effort Budgets

Agents cannot judge appropriate effort without explicit guidance (Anthropic, multi-agent
system, 2025). When designing a feature's workflow, specify expected effort for each
stage:

- Specification: "5–15 tool calls. Read design, query knowledge, draft sections."
- Implementation: "10–50 tool calls per task. Read spec, implement, test, iterate."
- Review: "5–10 tool calls. Read artifact, check against criteria, produce verdict."

Embed these in the task context via handoff, not in the skill itself. Skills are
reusable; effort budgets are task-specific.

### 2.4 Design the Verification Gates

Following MetaGPT's assembly-line paradigm, every stage transition should have a
verification mechanism:

1. **Structural checks (programmatic):** required sections present, cross-references
   valid, acceptance criteria listed. These should be deterministic tools.
2. **Quality checks (LLM-as-judge):** completeness, consistency, testability. Score
   0.0–1.0 per dimension.
3. **Iteration cap:** maximum 2–3 revision cycles before escalating to human review.

Design these gates at feature-design time, not as afterthoughts during implementation.

### 2.5 Define the Output Templates

Every stage that produces a document (specification, design, plan, review report) needs
an output template. The template itself is a forcing function — agents must engage with
each section, which distributes effort rather than allowing rush-to-implementation.

Templates should specify:
- Required sections with descriptions
- Minimum content expectations per section
- Cross-reference requirements
- Acceptance criteria format

Match template strictness to constraint_level: strict for low-freedom tasks, flexible
for high-freedom tasks.

### 2.6 Plan for Institutional Memory

When designing a feature, consider what knowledge it will produce:

- What always/never rules will agents learn during implementation?
- What anti-patterns will be discovered?
- What vocabulary is specific to this feature's domain?

Design the skill so that knowledge capture happens naturally at task completion (via the
`finish` tool's knowledge parameter), not as a separate documentation step that will be
skipped under time pressure.

---

## Part 3: Skill Authoring Checklist

Use this checklist when creating or revising any skill. It synthesises the research
findings into a practical gate.

### Structure

- [ ] YAML frontmatter includes all six required fields (name, description, triggers,
      roles, stage, constraint_level)
- [ ] Description has both `expert` (routing) and `natural` (trigger) registers
- [ ] Description is assertive — includes "use even when..." clause for
      workflow-critical skills
- [ ] Body follows attention-curve ordering: Vocabulary → Anti-Patterns → Checklist →
      Procedure → Output Format → Examples → Evaluation Criteria → Questions
- [ ] SKILL.md is under 500 lines
- [ ] Extended content is in reference files, linked one level deep from SKILL.md

### Vocabulary

- [ ] 15–30 domain-specific terms, each passing the 15-year practitioner test
- [ ] No general-purpose terms the model already knows
- [ ] Terms are specific to the skill's domain, not generic workflow vocabulary

### Anti-Patterns

- [ ] 5–10 named anti-patterns
- [ ] Each has Detect, BECAUSE, and Resolve fields
- [ ] BECAUSE clause explains the consequence chain, not a restatement of Detect

### Procedure

- [ ] Constraint level matches task risk (low/medium/high)
- [ ] Procedure has 5–10 steps (not 20+)
- [ ] Steps include IF/THEN conditions for branching paths
- [ ] Uncertainty protocol: explicit STOP instruction for ambiguous inputs, positioned
      early

### Content Quality

- [ ] Every paragraph teaches something the model does not already know
- [ ] Terminology is consistent with vocabulary section throughout
- [ ] No time-sensitive content (no dates, use "current method" / "previous method")
- [ ] BECAUSE clauses on imperatives — explains why, not just what

### Examples

- [ ] At least one BAD/GOOD pair with explanations
- [ ] Best GOOD example appears last (recency bias)
- [ ] Examples use concrete, realistic content

### Evaluation

- [ ] 4–8 gradable evaluation criteria with weights
- [ ] At least one criterion is marked `required`
- [ ] Criteria are specific enough for LLM-as-judge scoring (0.0–1.0)

### Retrieval Anchors

- [ ] 5–10 natural-language questions in the final section
- [ ] Questions are specific to the skill's domain

---

## Part 4: Common Mistakes in Skill Authoring

These mistakes recur across the research and our own experience:

### Explaining What the Model Already Knows

Skills that explain what a "state machine" is, what "lifecycle transitions" mean, or
how YAML works waste tokens on knowledge the model already has. The model knows these
concepts. What it does not know is Kanbanzai's specific state machine, Kanbanzai's
specific transitions, and Kanbanzai's specific rules.

**Fix:** Delete every paragraph that would be true of any system. Keep only what is
true of this system specifically.

### Uniform Constraint Levels

Applying medium freedom uniformly means fragile operations (lifecycle transitions) are
under-constrained and creative work (design) is over-constrained. Both degrade output.

**Fix:** Declare constraint_level explicitly. Use exact tool call sequences for
low-freedom operations. Use principles and guidance for high-freedom work.

### Missing Examples

Abstract rules without examples are less effective than examples without rules. Multiple
sources confirm that input/output examples train understanding better than descriptions.

**Fix:** Invest more effort in curating 2–3 excellent examples than in writing 20 rules.

### Over-Specification

Adding more rules past a threshold (approximately 19 requirements per Vaarta Analytics)
actively hurts compliance. The instinct to add "one more rule" to fix a problem often
makes things worse.

**Fix:** When a skill isn't working, try removing content before adding it. Read agent
transcripts to identify what's actually going wrong — the problem may not be what you
assume.

### Unexplained Imperatives

"ALWAYS do X" without a reason covers one case. "Do X because Y" generalises to
adjacent cases. Rigid imperatives without reasoning become dead weight that cannot be
evaluated or pruned.

**Fix:** Every imperative gets a BECAUSE clause. If you can't explain why, the rule may
not be necessary.

### Loading Everything at Once

Skills that put all content in a single SKILL.md file force it all into context when
triggered. Reference material, detailed rubrics, and edge-case handling should be in
Level 3 reference files that are loaded only when needed.

**Fix:** Keep SKILL.md as a routing document. Move detail into reference files. Link
one level deep — never reference a file from another reference file.

---

## Part 5: Key Research Sources

For traceability, these are the academic and industry sources that underpin the
recommendations in this document:

### Peer-Reviewed

| Source | Year | Key Finding Used |
|--------|------|-----------------|
| Zamfirescu-Pereira et al., "Why Johnny Can't Prompt" (CHI) | 2023 | Combined positive + negative constraints are strongest |
| Hong et al., MetaGPT (ICLR) | 2024 | Structured artifacts reduce errors ~40% vs. free dialogue |
| Liu et al., "Lost in the Middle" | 2024 | 30%+ accuracy drop for mid-context critical information |
| Ranjan et al., "One Word Is Not Enough" | 2024 | Vocabulary specificity activates domain knowledge clusters |
| Yang et al., SWE-agent (Princeton) | 2024 | ACI design affects performance as much as model capability |
| Masters et al., "Manager Agent" (DAI) | 2025 | Decomposition quality is critical path; hard vs soft constraints |
| Wu et al., MIT Position Bias | 2025 | Causal masking and RoPE cause U-shaped attention |
| Kim & Liu, Google Research Scaling Study | 2026 | Sequential penalty (39–70% degradation); saturation at 4 agents |

### Industry (Empirically Validated)

| Source | Year | Key Finding Used |
|--------|------|-----------------|
| PRISM Persona Framework | 2024 | <50 token identities optimal; flattery degrades output |
| Anthropic, "Building Effective Agents" | 2024 | ACI design; tool poka-yoke; "more time on tools than prompts" |
| Anthropic, "Multi-Agent Research System" | 2025 | Effort scaling; delegation quality; 40% speedup from tool rewrites |
| Anthropic, "Effective Context Engineering" | 2025 | Attention budget; progressive disclosure |
| Anthropic, Skill Authoring Best Practices | 2025 | Conciseness; degrees of freedom; progressive disclosure; checklists |
| Anthropic, skill-creator meta-skill | 2025 | Create→test→review→improve loop; explanatory tone over imperatives |
| MAST Failure Taxonomy | 2024–25 | 14 failure modes; rubber-stamp approval as #1 quality failure |
| LangChain Few-Shot Research | 2024 | 3 well-chosen examples match 9 in effectiveness |
| Voyce, XML/Markdown Comparative | 2025 | Format alone accounts for up to 40% performance variance |
| Vaarta Analytics, "Prompt Engineering Is System Design" | 2026 | At n=19 requirements, accuracy drops below n=5 |
| Microsoft, Agent Orchestration Patterns | 2026 | Maker-checker loops; sequential gates; context management |