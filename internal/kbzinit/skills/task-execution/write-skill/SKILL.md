---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: write-skill
description:
  expert: "Skill authoring producing a research-backed SKILL.md with vocabulary
    payload, named anti-patterns, attention-curve section ordering, progressive
    disclosure via reference files, and gradable evaluation criteria for any
    workflow stage"
  natural: "Write a new skill or revise an existing one, following the
    research-backed conventions for structure, vocabulary, examples, and
    evaluation"
triggers:
  - write a new skill
  - create a skill for a workflow stage
  - author a SKILL.md
  - design a skill
  - revise an existing skill
  - improve a skill
roles: [architect, spec-author]
stage: designing
constraint_level: medium
---

## Vocabulary

- **vocabulary payload** — the 15–30 domain-specific terms placed first in the skill body that activate the model's specialised knowledge clusters for the task domain
- **attention curve** — the U-shaped distribution of transformer attention across context position; content at the start and end receives disproportionate weight, the middle receives least
- **progressive disclosure** — three-level loading architecture: Level 1 metadata (~100 tokens, always loaded), Level 2 instructions (SKILL.md body, loaded on trigger), Level 3 resources (reference files, loaded on demand)
- **constraint level** — the degree of procedural freedom a skill grants: low (exact steps), medium (templates with flexibility), high (principles and guidance)
- **routing signal** — vocabulary or phrasing that directs the model toward specific knowledge regions; domain-specific terms route to expert knowledge, generic terms route to surface-level advice
- **retrieval anchor** — a natural-language question placed at the end of a skill that exploits recency bias to improve the model's ability to match future queries to this skill's content
- **dual-register description** — a description with both an expert register (technical terms for routing depth) and a natural register (casual phrasing for trigger breadth)
- **anti-pattern name** — a concise, memorable label for a common mistake; named patterns activate deeper knowledge than unnamed descriptions of the same problem
- **BECAUSE clause** — the causal explanation attached to an anti-pattern or imperative that makes the rule generalisable to adjacent cases rather than covering only the literal case
- **recency bias** — the model's tendency to weight the final item in a sequence most heavily during generation; the best example should appear last
- **15-year practitioner test** — a filter for vocabulary terms: would a senior expert with 15+ years in this domain use this exact term when speaking with a peer?
- **n=19 cliff** — the finding that prompt accuracy at 19 requirements drops below accuracy at 5 requirements; adding rules past a threshold actively hurts compliance
- **maker-checker pattern** — a workflow where one agent produces output and a separate agent evaluates it against defined criteria
- **evaluation criterion** — a specific, gradable question about a skill's output that an LLM-as-judge can score on a 0.0–1.0 scale
- **token budget** — the share of the context window a skill consumes; the optimal zone is 15–40% of total window capacity across all loaded content
- **forcing function** — a structural element (template, checklist, required section) that compels the agent to distribute effort rather than shortcutting

## Anti-Patterns

### Explaining Common Knowledge
- **Detect:** The skill defines or explains concepts the model already knows — state machines, YAML syntax, what a "function" is, how HTTP works
- **BECAUSE:** Every token spent on common knowledge competes for attention with the Kanbanzai-specific rules the skill exists to teach, diluting the skill's actual value; at high token counts, this dilution measurably degrades output quality (Anthropic, "Effective Context Engineering", 2025)
- **Resolve:** Delete every paragraph that would be true of any system. Keep only what is specific to this project, this workflow, or this domain

### Uniform Constraint Level
- **Detect:** The skill applies medium-freedom procedures to both fragile operations (lifecycle transitions) and creative work (design choices)
- **BECAUSE:** Fragile operations under-constrained by medium freedom get skipped or improvised; creative work over-constrained by medium freedom produces mediocre output — both failure modes are documented in the orchestration research (Google Research, 2026; Anthropic Best Practices)
- **Resolve:** Declare `constraint_level` explicitly in frontmatter. Use exact tool-call sequences for low, templates for medium, principles for high

### Missing Examples
- **Detect:** The skill has procedural rules but no BAD/GOOD example pairs
- **BECAUSE:** Three well-chosen examples match nine in effectiveness (LangChain, 2024), and input/output examples train understanding better than abstract instructions (Anthropic, Increase Consistency guide) — a skill without examples is leaving the highest-leverage teaching mechanism unused
- **Resolve:** Add at least one BAD/GOOD pair per skill. Invest more effort in curating 2–3 excellent examples than in writing 20 additional rules

### Over-Specification
- **Detect:** The procedure section has more than 15 steps, or the skill body exceeds 500 lines without reference files
- **BECAUSE:** At approximately 19 requirements, accuracy drops below a prompt with only 5 requirements (Vaarta Analytics, 2026) — more rules past a threshold actively hurts compliance rather than improving it
- **Resolve:** Move detail into reference files. Keep the procedure focused on the 5–10 critical-path steps. When the skill isn't working, try removing content before adding it

### Unexplained Imperatives
- **Detect:** Rules use "ALWAYS" or "NEVER" without a BECAUSE clause explaining why
- **BECAUSE:** "Always X" covers one literal case; "Do X because Y" generalises correctly to adjacent cases the rule author didn't anticipate (Zamfirescu-Pereira et al., CHI 2023) — unexplained rules also become dead weight that cannot be evaluated or pruned because no one knows why they were added
- **Resolve:** Attach a BECAUSE clause to every imperative. If you cannot explain why the rule exists, the rule may not be necessary

### Monolithic SKILL.md
- **Detect:** All skill content — procedures, examples, rubrics, reference tables — lives in a single SKILL.md file with no reference directory
- **BECAUSE:** When the skill triggers, its entire body loads into context at Level 2, competing for attention budget; detailed rubrics and edge-case handling that are only needed occasionally should be Level 3 resources loaded on demand (Anthropic, Skills Overview)
- **Resolve:** Keep SKILL.md as a routing document under 500 lines. Move extended content to `references/` and link one level deep

### Flat Description
- **Detect:** The skill description uses only generic language without domain-specific terms or explicit trigger conditions
- **BECAUSE:** The description is the only content the model sees before deciding whether to trigger the skill; without domain terms for routing depth and explicit "use when..." clauses for trigger breadth, the skill will under-trigger — and a skill that doesn't trigger is worthless regardless of its content quality
- **Resolve:** Write a dual-register description: expert register with precise terminology, natural register with casual phrasing. Add an assertive "use even when..." clause for workflow-critical skills

## Checklist

Copy this checklist and track progress when authoring a new skill:

- [ ] Identify the workflow stage this skill serves and the role(s) that will use it
- [ ] Determine the constraint level (low/medium/high) based on task fragility
- [ ] Draft the dual-register description (expert + natural)
- [ ] Write 15–30 vocabulary terms, each passing the 15-year practitioner test
- [ ] Write 5–10 named anti-patterns with Detect/BECAUSE/Resolve
- [ ] Write the procedure (5–10 steps with IF/THEN conditions)
- [ ] Add uncertainty protocol (STOP instruction) early in the procedure
- [ ] Define the output format with a structured template
- [ ] Create at least one BAD/GOOD example pair (best GOOD example last)
- [ ] Write 4–8 gradable evaluation criteria with weights
- [ ] Write 5–10 retrieval anchor questions for the final section
- [ ] Verify SKILL.md is under 500 lines; move overflow to `references/`
- [ ] Run the novelty test: delete every paragraph the model already knows
- [ ] Verify terminology consistency against vocabulary section

## Procedure

1. **Read the conventions file** at `.kbz/skills/CONVENTIONS.md` before writing anything. It defines the mandatory frontmatter fields, section ordering, and formatting rules. Do not rely on memory of the conventions — read the file because it is the source of truth and may have been updated.

2. **Identify the skill's purpose and constraint level.** Determine which workflow stage the skill serves, which role(s) will use it, and whether the task is fragile (low freedom), templated (medium freedom), or creative (high freedom). The constraint level determines the procedure style. IF the task involves lifecycle transitions, document registration, or stage gates, the constraint level should be low. IF the task involves specification writing, structured review, or plan creation, medium. IF the task involves design, research, or implementation, high.

3. **Draft the frontmatter.** Write all six required fields: `name`, `description` (with `expert` and `natural` registers), `triggers` (at least two), `roles`, `stage`, and `constraint_level`. The expert description should include the precise domain terms that will route the model to the right knowledge. The natural description should read like something a human would say when requesting this task. Make descriptions assertive — for workflow-critical skills, include a "use even when..." clause to combat under-triggering.

4. **Write the vocabulary payload.** This is the first section in the body and the most important because it occupies the highest-attention position. Select 15–30 terms specific to the skill's domain. Apply the 15-year practitioner test to each term. Exclude terms the model already knows generically. Each term gets a one-line definition specific to how the term is used in this skill's context.

5. **Write the anti-patterns.** Place these before the procedure because agents need to know what NOT to do before they learn what TO do. Name each anti-pattern memorably — named patterns activate deeper knowledge. Each anti-pattern needs three fields: Detect (the observable signal), BECAUSE (the causal consequence chain), and Resolve (the concrete corrective action). The BECAUSE clause must explain *why*, not restate *what*.

6. **Write the procedure.** Use numbered steps with IF/THEN conditions for branching. Keep to 5–10 steps. Include an explicit uncertainty protocol (a STOP instruction telling the agent to pause and ask for clarification) positioned early in the procedure. Match the level of detail to the constraint level: exact tool calls for low, templates with flexibility for medium, principles and judgment for high. IF the procedure exceeds 10 steps, split it — move the less-critical path into a reference file.

7. **Define the output format.** Specify what the skill produces as a structured template. Match template strictness to constraint level. Low-constraint skills should have exact field-by-field templates. High-constraint skills should have suggested structures with flexibility.

8. **Create examples.** Write at least one BAD/GOOD pair. Use concrete, realistic content — not placeholders. Each example needs an explanation of WHY it is bad or good. Place the best GOOD example last in the section to exploit recency bias. IF the skill has multiple distinct output types, provide one pair per type.

9. **Write evaluation criteria.** Define 4–8 gradable questions about the skill's output. Each criterion should be evaluable by an LLM-as-judge producing 0.0–1.0 scores. At least one must be marked `required`. Avoid subjective criteria ("is it well-written?"); prefer specific, verifiable conditions ("does the output contain all required sections?").

10. **Write retrieval anchors.** In the final section ("Questions This Skill Answers"), write 5–10 natural-language questions that someone might ask when they need this skill. These exploit the high-attention position at the end of context to improve future skill discovery.

11. **Review against quality constraints.** Run the novelty test: re-read every paragraph and delete anything the model already knows generically. Check terminology consistency — every term used in the procedure must match the vocabulary section. Verify the SKILL.md is under 500 lines. IF it exceeds 500 lines, move reference material, extended examples, or detailed rubrics into a `references/` directory and link them one level deep from SKILL.md.

## Output Format

The skill produces a directory with this structure:

```
skill-name/
├── SKILL.md              # Under 500 lines
│   ├── YAML frontmatter  # All six required fields
│   ├── Vocabulary         # 15–30 domain terms (first section)
│   ├── Anti-Patterns      # 5–10 named patterns with Detect/BECAUSE/Resolve
│   ├── Checklist          # Copy-paste tracking list (medium/low constraint)
│   ├── Procedure          # 5–10 numbered steps with IF/THEN
│   ├── Output Format      # Structured template for the deliverable
│   ├── Examples           # BAD/GOOD pairs with explanations
│   ├── Evaluation Criteria # 4–8 gradable questions with weights
│   └── Questions          # 5–10 retrieval anchors (final section)
└── references/            # Optional, for overflow content
    ├── extended-examples.md
    └── detailed-rubric.md
```

## Examples

### BAD: Vocabulary section with generic terms

> ## Vocabulary
>
> - **function** — a reusable block of code that performs a specific task
> - **variable** — a named storage location for data
> - **YAML** — a human-readable data serialisation language
> - **state machine** — a system that transitions between defined states
> - **review** — the process of examining work for quality

**WHY BAD:** Every term here is general knowledge the model already has. None activates specialised knowledge. The token cost is pure waste — it competes for attention with the skill's actual content without adding any routing value.

### BAD: Anti-pattern without BECAUSE clause

> ### Skipping Specification
> - **Detect:** Agent proceeds directly to implementation without writing a spec
> - **Resolve:** Write a specification before implementing

**WHY BAD:** Without a BECAUSE clause, the rule covers only this literal case. An agent encountering a similar but not identical situation (e.g., skipping design review, or writing a minimal spec to tick the box) has no reasoning to generalise from.

### GOOD: Anti-pattern with full consequence chain

> ### Skipping Specification
> - **Detect:** Agent proceeds directly to implementation without writing a spec document
> - **BECAUSE:** Undocumented design decisions made during implementation are invisible to reviewers and future maintainers — the cost of discovering and correcting them during review is 5–10× higher than addressing them during specification, and decisions not captured in the spec cannot be traced when requirements change
> - **Resolve:** Write a specification covering at least: problem statement, requirements, constraints, and acceptance criteria before any implementation task is created

**WHY GOOD:** The BECAUSE clause explains the full consequence chain — from invisible decisions through to costly review corrections and lost traceability. An agent reading this can generalise: any step that captures decisions early and makes them visible to reviewers serves the same purpose, even if the specific format varies.

### GOOD: Vocabulary with domain-specific routing terms

> ## Vocabulary
>
> - **vocabulary payload** — the 15–30 domain terms placed first in a skill body that activate specialised knowledge clusters
> - **attention curve** — the U-shaped attention distribution across context position; start and end receive highest weight
> - **constraint level** — the degree of procedural freedom: low (exact steps), medium (templates), high (principles)
> - **dual-register description** — expert terminology for routing depth plus natural language for trigger breadth
> - **BECAUSE clause** — the causal explanation that makes an anti-pattern rule generalisable beyond its literal case

**WHY GOOD:** Every term is specific to the skill authoring domain and passes the 15-year practitioner test. None is general knowledge. Each definition is scoped to how the term functions within skills, not its general meaning. These terms route the model toward prompt engineering and instructional design knowledge rather than generic software engineering.

## Evaluation Criteria

1. Does the skill include all six required frontmatter fields with a dual-register description? Weight: required.
2. Does the vocabulary section contain 15–30 terms that pass the 15-year practitioner test, with no general-knowledge terms? Weight: required.
3. Does every anti-pattern have Detect, BECAUSE, and Resolve fields, where BECAUSE explains the consequence chain rather than restating Detect? Weight: required.
4. Does the procedure match the declared constraint level (exact steps for low, templates for medium, principles for high)? Weight: high.
5. Is there at least one BAD/GOOD example pair with explanations, with the best GOOD example last? Weight: high.
6. Is the SKILL.md under 500 lines, with overflow content in reference files linked one level deep? Weight: high.
7. Does the novelty test pass — does every paragraph teach something the model does not already know? Weight: medium.
8. Are evaluation criteria specific enough for LLM-as-judge scoring on a 0.0–1.0 scale? Weight: medium.

## Questions This Skill Answers

- How do I create a new skill for a workflow stage?
- What sections should a SKILL.md contain and in what order?
- How many vocabulary terms should a skill have and how do I choose them?
- What makes a good anti-pattern entry versus a weak one?
- How do I decide the constraint level for a new skill?
- What is the right length for a SKILL.md and when should I use reference files?
- How do I write a dual-register description that triggers reliably?
- What does an effective BAD/GOOD example pair look like?
- How do I write evaluation criteria that an LLM-as-judge can score?
- When should I stop adding rules and start removing them?