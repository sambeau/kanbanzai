# Documentation Pipeline

A document passes through five editorial stages on its way from draft to publication. Each stage has a focused job, clear boundaries, and a dedicated SKILL that defines what to do and — just as importantly — what not to touch.

The pipeline works from large to small: structural decisions first, sentence-level polish last. There is no point perfecting a sentence that a structural edit will cut, and no point fact-checking a section that might be reorganised.

---

## The five stages

```
Write  →  Edit  →  Check  →  Style  →  Copyedit
```

| Stage | Skill | Job | Not your job |
|-------|-------|-----|--------------|
| **Write** | `write-docs` | Produce a well-structured first draft following the inverted pyramid | Polishing prose, removing AI artifacts, final punctuation |
| **Edit** | `edit-docs` | Verify and improve structure, scannability, tone gradient, audience fit | Rewriting sentences, checking facts, fixing punctuation |
| **Check** | `check-docs` | Verify every fact, test every code example, flag hallucinations and vague claims | Restructuring, prose style, removing AI clichés |
| **Style** | `style-docs` | Hunt and eliminate AI writing patterns: banned words, clichés, filler, robotic structure | Restructuring sections, checking facts, punctuation |
| **Copyedit** | `copyedit-docs` | Rework sentences for clarity, enforce active voice, simplify punctuation, ensure consistency | Restructuring, changing content, hunting AI artifacts |

Each stage operates at a smaller scale than the previous one. Each stage trusts that the previous stage did its job.

---

## How the styleguides are sliced

Each stage draws from a different subset of the project's style references. No stage needs to read all four guides.

| Stage | Primary reference | Also draws from |
|-------|-------------------|-----------------|
| **Write** | `documentation-structure-guide.md` (full) | `technical-writing-guide.md` §1 (voice), §4 (word choice), §8–11 (structure) |
| **Edit** | `documentation-structure-guide.md` §1, 4–6, 8 | `technical-writing-guide.md` §8, 10, 11 (scannability, flow, longer docs); `humanising-ai-prose.md` §6 (structural tells) |
| **Check** | `humanising-ai-prose.md` §5 (content and substance) | `documentation-structure-guide.md` §3 (source of truth) |
| **Style** | `humanising-ai-prose.md` §1–3, 7, 9 (vocabulary, phrases, structure, process, what not to fix) | `technical-writing-guide.md` §4 (word choice) |
| **Copyedit** | `punctuation-guide.md` (full) | `technical-writing-guide.md` §2–3, 5–7 (sentences, voice, abbreviations, punctuation, capitalisation); `humanising-ai-prose.md` §4 (punctuation tells) |

Each SKILL contains a distilled version of its relevant guidance inline. The full guides are available as Level 3 references when deeper detail is needed.

---

## Stage boundaries

The most important property of the pipeline is that each stage knows what it owns and what it must not touch.

### Write

**Owns:** document purpose, structure, outline, audience assumptions, content, inverted pyramid at every level, examples, figures.

**Does not touch:** prose polish. The writer focuses on getting the right content in the right structure. Imperfect sentences are fine — later stages will handle them.

### Edit

**Owns:** document-level and section-level structure, heading skeleton, inverted pyramid compliance, scannability, tone gradient, structural tells (inline-header lists, title case, emoji, unnecessary tables).

**Does not touch:** individual sentences, word choice, facts, punctuation. If a section has good structure but clunky prose, the editor leaves it for later stages.

**Flag-only:** if the editor finds a factual issue, they flag it as a comment for the Check stage. They do not fix it.

### Check

**Owns:** factual accuracy, code example verification, source-of-truth compliance, substance (vague claims, significance inflation, promotional language, superficial analysis).

**Does not touch:** structure, prose style, punctuation, AI artifacts. The checker's output is a list of findings, not a rewritten document.

**Flag-only:** if the checker finds a structural problem, they flag it for a possible second Edit pass.

### Style

**Owns:** AI-specific artifacts — banned words, inflated adjectives, faux-insider phrases, staccato rhetoric, hedging, tricolon overuse, rigid paragraph formula, robotic transitions, elegant variation, copula avoidance.

**Does not touch:** document structure (already verified), facts (already checked), punctuation (next stage handles it). The style editor rewrites sentences to eliminate AI patterns but does not restructure sections.

### Copyedit

**Owns:** sentence clarity, active/passive voice, smothered verbs, sentence length, punctuation, capitalisation, abbreviations, parallel structure, consistency, reading rhythm.

**Does not touch:** document structure, content, facts. The copy editor trusts that everything above the sentence level is already correct.

---

## Change tracking

Each stage should output:

1. **The revised document** (or, for the Check stage, the original document with annotations).
2. **A changelog** — a brief summary of what was changed and why.
3. **Flags** — issues outside this stage's scope that a previous or later stage should address.

The changelog lets a human reviewer understand the delta at each stage without re-reading the entire document.

---

## Human checkpoints

Two human reviews are recommended:

1. **After Edit** — structural decisions are expensive to undo. If the editor reorganised sections in a way you disagree with, catch it before three more stages build on that structure.
2. **After Copyedit** — a final read-through of the finished product.

The intermediate stages (Check, Style) are lower-risk. They can flow without human gates unless a flag is raised.

---

## When to re-enter the pipeline

Sometimes a later stage discovers a problem that belongs to an earlier stage. The rules:

- **Check finds a structural problem** → flag it. If severe, send the document back to Edit before continuing.
- **Style finds a factual issue** → flag it. Send to Check before continuing.
- **Any stage finds the document unsalvageable** → send it back to Write.

Re-entry should be rare. If a document routinely bounces between stages, the Write stage needs better guidance or the source material needs improvement.

---

## Idempotency

A well-designed stage should be safe to run twice. Running the copy editor on already-copyedited text should not degrade the output. This is a design goal for every SKILL — and worth testing when building or revising a stage.

---

## Diagrams and illustrations

Diagrams are a parallel concern, not a sequential stage. The Edit stage flags where a diagram would be more effective than prose. Diagram creation can happen alongside or after the text pipeline.

---

## SKILLs

Each stage has a corresponding SKILL in `.kbz/skills/`:

| Stage | SKILL path |
|-------|------------|
| Write | `.kbz/skills/write-docs/SKILL.md` |
| Edit | `.kbz/skills/edit-docs/SKILL.md` |
| Check | `.kbz/skills/check-docs/SKILL.md` |
| Style | `.kbz/skills/style-docs/SKILL.md` |
| Copyedit | `.kbz/skills/copyedit-docs/SKILL.md` |

Each SKILL contains a distilled subset of the styleguides targeted at that stage's specific responsibilities, plus the vocabulary, anti-patterns, procedure, and checklist for the stage.

---

## Roles

Each stage maps to a role that defines the agent's identity, vocabulary, and tool access. The writer stage uses the existing `documenter` role. The editorial stages require dedicated roles:

| Stage | Role |
|-------|------|
| Write | `documenter` (existing) |
| Edit | `doc-editor` |
| Check | `doc-checker` |
| Style | `doc-stylist` |
| Copyedit | `doc-copyeditor` |

---

*For the full styleguides, see: [documentation-structure-guide.md](documentation-structure-guide.md), [technical-writing-guide.md](technical-writing-guide.md), [humanising-ai-prose.md](humanising-ai-prose.md), [punctuation-guide.md](punctuation-guide.md).*