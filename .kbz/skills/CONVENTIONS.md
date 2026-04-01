# Skill Authoring Conventions

Shared formatting reference for all `.kbz/skills/*/SKILL.md` files. Consult this
when writing or reviewing skill content to ensure consistency across the skill library.

---

## Frontmatter Template

Every SKILL.md begins with a YAML frontmatter block containing all six required fields:

```yaml
---
name: <skill-name>
description:
  expert: "<technical description — what the skill does, when to use it, third person>"
  natural: "<casual description — plain language a human would use to request this task>"
triggers:
  - <trigger phrase 1>
  - <trigger phrase 2>
roles: [<role-id-1>]
stage: <workflow-stage>
constraint_level: low | medium | high
---
```

Field notes:
- **`description.expert`** activates deep knowledge on direct invocation. Uses third person.
- **`description.natural`** matches casual phrasing. Reads like a human request.
- **`triggers`** has at least two phrases that an orchestrator or human might use.
- **`constraint_level`** determines procedure style: `low` = exact step sequences,
  `medium` = templates with flexibility, `high` = guidance and principles.

---

## Section Ordering

Sections follow the attention-curve layout. This order is mandatory:

1. `## Vocabulary` — highest-attention position, activates knowledge clusters
2. `## Anti-Patterns` — what NOT to do, before the procedure
3. `## Checklist` — optional, include for `medium` or `low` constraint_level skills
4. `## Procedure` — numbered steps with IF/THEN conditions
5. `## Output Format` — structured template for the deliverable
6. `## Examples` — BAD/GOOD pairs with explanations
7. `## Evaluation Criteria` — gradable questions with weights
8. `## Questions This Skill Answers` — retrieval anchors, final section

---

## Vocabulary Format

15–30 domain-specific terms per skill. Each term must pass the **15-year practitioner
test**: a senior expert would use this exact term when speaking with a peer.

```markdown
- **term** — one-line definition specific to this skill's domain
```

Exclude general-purpose terms the model already knows (e.g., "variable", "function",
"YAML"). Include only terms that activate the right knowledge cluster for the task.

---

## Anti-Pattern Format

5–10 named anti-patterns per skill. Each has a `###` heading and three required fields:

```markdown
### Anti-Pattern Name
- **Detect:** observable signal that this anti-pattern is occurring
- **BECAUSE:** why this is harmful — the causal explanation, not a restatement of Detect
- **Resolve:** concrete action to fix or avoid the anti-pattern
```

The BECAUSE clause must explain *why*, not restate *what*. "BECAUSE this produces vague
output" restates the detection signal. "BECAUSE vague requirements cannot be verified
during review, leading to acceptance disputes" explains the consequence chain.

---

## Example Format

At least one BAD and one GOOD example per skill:

```markdown
### BAD: Short description

> (example content in blockquote)

**WHY BAD:** Explanation of what makes this a poor example.

### GOOD: Short description

> (example content in blockquote)

**WHY GOOD:** Explanation of what makes this effective.
```

Place the **best GOOD example last** in the section. The model weights the final example
most heavily during generation (recency bias).

---

## Evaluation Criteria Format

4–8 gradable questions about the skill's output. Each has a weight:

```markdown
1. Does the output contain [specific quality]? Weight: required.
2. Are [specific elements] present and well-formed? Weight: high.
3. Does [quality aspect] meet [threshold]? Weight: medium.
```

Weight meanings:
- `required` — output fails without this; at least one criterion per skill must be required
- `high` — strong quality signal
- `medium` — desirable but not critical

Criteria must be evaluable by an LLM-as-judge pass producing 0.0–1.0 scores. Avoid
subjective criteria ("is the writing good?"). Prefer specific, verifiable conditions.

---

## Questions This Skill Answers

5–10 natural-language queries as retrieval anchors in the final section:

```markdown
- How do I [specific task related to this skill]?
- What format should [specific output] use?
- When should I [specific decision point]?
```

Questions must be specific to the skill's domain, not generic.

---

## Line Budget

SKILL.md must stay **under 500 lines**. If content exceeds this:

- Move extended examples, detailed rubrics, or reference tables to `references/`
- Link from SKILL.md: `See [references/topic.md](references/topic.md)`
- Reference files must link directly from SKILL.md — never from other reference files
- Keep the SKILL.md self-contained for the common case; references handle edge cases

---

## Quality Constraints

Before shipping any skill content:

- **Novelty test:** Every paragraph teaches something the model does not already know.
  Delete explanations of general concepts. Keep only Kanbanzai-specific rules and
  domain-specific conventions.
- **Terminology consistency:** Use vocabulary terms exclusively within the skill. If the
  vocabulary says "acceptance criterion," never write "requirement" or "success condition"
  as a synonym in the procedure or examples.
- **BECAUSE clauses:** Every anti-pattern has one. Every imperative in the procedure
  explains why, not just what. "Do X because Y" generalises; "ALWAYS do X" is brittle.
- **Uncertainty protocol:** Every skill that produces output includes an explicit STOP
  instruction for ambiguous or incomplete inputs, positioned early in the procedure.
- **No time-sensitive content:** Use "current method" / "previous method" instead of dates.