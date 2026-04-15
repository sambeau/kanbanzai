# The document pipeline: a five-stage editorial system for AI-generated prose

AI writes fast. It also writes formulaic, vague, and structurally flat prose that reads like a keynote speech. The document pipeline exists to fix that — systematically, repeatably, and without relying on a single agent to be good at everything.

This guide explains how the pipeline works, what makes it different from asking an LLM to "write and then revise," and how you could build something similar in your own system.

---

## What the pipeline does

A document enters the pipeline as a purpose statement and source material (or an existing draft). It exits as a structurally sound, fact-checked, human-sounding document with a full audit trail of every change made at every stage.

Between entry and exit, five specialist agents process the document in sequence:

```
Write  →  Edit  →  Check  →  Style  →  Copyedit
```

Each agent has a single job. Each agent has explicit boundaries defining what it must not touch. Each agent produces a changelog so a human can review what happened without re-reading the entire document.

---

## Why five stages instead of one

The naive approach — "write this document and make it good" — fails for the same reason that asking one person to be the architect, builder, inspector, and interior designer of a house fails. The concerns are different, the evaluation criteria are different, and the failure modes interfere with each other.

Specific problems with single-pass document generation:

- **Structure gets sacrificed for prose.** The agent spends its effort on smooth sentences and neglects whether the document's sections are in the right order.
- **Facts go unchecked.** The same agent that invented a claim is unlikely to question it on a second pass.
- **AI artifacts survive.** The agent that wrote "leverages cutting-edge technology" will not flag it as a problem, because from its perspective it wrote a perfectly reasonable sentence.
- **Revisions are invisible.** Without changelogs, a human reviewer has to diff the before and after to understand what changed and why.

The pipeline solves these by separating concerns into stages that work from large to small. There is no point polishing a sentence that a structural edit will cut. There is no point fact-checking a section that might be reorganised.

---

## The five stages

### 1. Write

**Role:** Senior technical writer
**Job:** Produce a structured first draft following the inverted pyramid — most important information first at every level.
**Not its job:** Polishing prose, removing AI artifacts, or final punctuation.

The writer plans before writing: purpose statement, audience assumptions, key messages, sentence outline. It drafts examples and figures before prose, because prose exists to connect examples, not the other way around. It writes the introduction last, summarising what was actually written rather than what was planned.

**Edits the file directly.**

### 2. Edit

**Role:** Developmental editor
**Job:** Verify and improve structure — heading skeleton, inverted pyramid compliance, scannability, tone gradient, audience fit.
**Not its job:** Rewriting sentences, checking facts, or fixing punctuation.

The editor extracts the heading skeleton (all headings read in order) and evaluates whether it tells a coherent story. It checks whether the most important content appears first. It flags structural tells — the formulaic patterns that betray AI generation, like every list having exactly three items or every heading in Title Case.

**Produces a report.** The orchestrator applies structural changes to the file.

### 3. Check

**Role:** Technical fact-checker
**Job:** Verify every factual claim against the implementation, test every code example, flag hallucinations and vague claims.
**Not its job:** Restructuring, prose style, or removing AI clichés.

The checker treats the codebase as the source of truth, not design documents. Design documents describe intentions; implementations describe reality. Every API reference, flag name, file path, and code example is traced against the actual code. Claims are classified: hallucination (provably wrong), unverified (no source found), stale (outdated reference), vague (too general), inflated (importance without evidence), or promotional (advertising language).

**Produces a report.** The orchestrator applies factual corrections to the file.

### 4. Style

**Role:** AI prose editor
**Job:** Hunt and eliminate AI writing patterns — banned words, clichés, filler, robotic structure, significance inflation.
**Not its job:** Restructuring sections, checking facts, or fixing punctuation.

This stage targets the specific vocabulary and structural patterns that LLMs overuse. It maintains a banned-word list (delve, leverage, utilize, facilitate, and seventeen others) and a set of structural patterns to detect: faux-insider openers ("Here's what most people get wrong"), staccato rhetoric ("No config. No setup. No hassle."), hollow conclusions ("In summary, X represents a powerful approach to…"), and the rigid claim-support-restate paragraph formula.

When three or more AI tells cluster in a single passage, the stage flags it for full rewriting rather than word-by-word substitution. Swapping "leverage" for "use" in a sentence that still contains a cliché opener, a "not just X but Y" construction, and an inflated adjective produces text that still reads as machine-generated.

**Edits the file directly.**

### 5. Copyedit

**Role:** Senior copy editor
**Job:** Sentence-level polish — active voice, verb clarity, punctuation hygiene, consistency, reading rhythm.
**Not its job:** Restructuring, changing content, or hunting AI artifacts.

The copy editor enforces active voice by default (with deliberate exceptions for error messages and receiver emphasis), unburies smothered verbs ("perform an installation" → "install"), splits overlong sentences, and ensures internal consistency in contractions, capitalisation, terminology, and formatting.

**Edits the file directly.**

---

## Stage boundaries: the pipeline's most important property

Each stage knows what it owns and what it must not touch. This is the single most important design decision in the pipeline.

When an editor restructures a section, the fact-checker doesn't undo that work by rewriting sentences. When the fact-checker flags a hallucination, the style editor doesn't restructure the surrounding section to accommodate the fix. Each stage trusts that previous stages did their job and focuses entirely on its own scope.

Boundary violations are the primary failure mode the orchestrator watches for. After every stage, the orchestrator reviews the changelog against the stage's declared boundaries. If the fact-checker rewrote prose or the copy editor restructured sections, those changes are discarded and the stage re-runs with explicit boundary reminders.

---

## The orchestrator

A sixth agent — the pipeline coordinator — manages the sequence. It does not write or edit prose. Its responsibilities:

1. **Dispatch.** Pass the document and the previous stage's changelog to each successive stage.
2. **Apply reports.** The Edit and Check stages produce reports rather than editing the file. The orchestrator reads the report, applies the changes, and passes the modified file to the next stage. This separation exists because structural and factual editing require judgement about which findings to apply and which to defer.
3. **Boundary enforcement.** Review each stage's changelog for out-of-scope changes. Discard them if found.
4. **Re-entry.** If a later stage discovers a problem that belongs to an earlier stage (the fact-checker finds a structural problem, the style editor finds a hallucination), the orchestrator decides whether to send the document back. Re-entry is reserved for severe issues — minor cross-stage flags are recorded in the completion summary for human review.
5. **Checkpoints.** The orchestrator offers two advisory human review points: after Edit (because structural decisions are expensive to undo) and after Copyedit (a final read-through). These don't block the pipeline — if the human doesn't respond, the pipeline continues.
6. **Completion summary.** All five changelogs are collated into a single report showing the document's journey through the pipeline.

The orchestrator pattern is lighter than a full multi-agent system. It's sequential dispatch only — no parallelism, no complex coordination. Each stage runs to completion before the next starts.

---

## What makes this different

### Separation of concerns, not iterative refinement

Most AI writing workflows ask the model to write, then revise, then revise again. Each pass tries to improve everything at once. The pipeline instead assigns each concern to exactly one stage. Structure is decided once (at Edit) and then trusted. Facts are checked once (at Check) and then trusted. This prevents the oscillation problem where revision pass N undoes improvements from pass N-1.

### Explicit stage boundaries with enforcement

Telling an agent "focus on structure" is a suggestion. Giving it a role definition with vocabulary, anti-patterns, a "not your job" list, and an orchestrator that discards out-of-scope changes is a constraint. Boundaries are enforced, not requested.

### Change tracking as a first-class concern

Every stage produces a changelog. The orchestrator reviews changelogs for boundary violations. Humans review changelogs instead of diffing documents. The pipeline is auditable at every step. This solves the "what did the AI actually change?" problem that makes most AI writing workflows opaque.

### AI-artifact detection as a dedicated stage

Most writing tools treat AI artifacts as a post-processing problem — run a detector and swap some words. The Style stage treats it as an editorial discipline with its own vocabulary (banned words, fingerprint clusters, copula avoidance, elegant variation), its own procedures (scan for clusters before fixing individuals), and its own anti-patterns (word-swapping without rewriting, over-correction that breaks good grammar). It knows that three AI tells in one passage means rewrite from scratch, not patch word by word.

### Fact-checking against implementation, not design

The Check stage's source of truth is the codebase — code, configuration, runtime behaviour. Design documents describe intentions; implementations describe reality. This distinction matters because LLMs happily generate documentation that matches what a system was designed to do rather than what it actually does.

### The large-to-small principle

The five stages are ordered by the scale of their concerns: document structure, section structure, factual content, vocabulary patterns, sentence mechanics. Each stage operates at a smaller scale than the previous one. This ordering means that expensive rework (restructuring) happens early, and cheap rework (word choice) happens late. You never polish a sentence that will be cut.

---

## How to build something similar

You don't need Kanbanzai to use this pattern. The core ideas are portable to any system that can run multiple LLM calls in sequence and pass context between them.

### Step 1: Define your stages

Five stages work well for technical documentation. You might need fewer for simpler content (blog posts could skip the Check stage) or more for specialised content (academic writing might split Check into a citation-verification stage and a methodology-review stage). The key constraint: each stage must have a single, clearly bounded responsibility.

### Step 2: Write a role definition for each stage

Each role needs:

- **Identity.** A one-line description of who this agent is ("Senior technical writer", "Technical fact-checker"). This anchors the system prompt.
- **Vocabulary.** The terms this role uses, defined precisely. Shared vocabulary between the human, the orchestrator, and the stage agent prevents miscommunication.
- **Anti-patterns.** The specific failure modes this role must avoid, with detection criteria and resolution steps. Anti-patterns are more useful than instructions because they describe what the agent will do wrong by default.
- **Boundaries.** What this role owns and what it must not touch. Be explicit. "Does not touch: sentence-level prose, punctuation, facts" is better than "focuses on structure."

### Step 3: Write a skill definition for each stage

Each skill needs:

- **Procedure.** Step-by-step instructions for what the agent does, in order.
- **Output format.** What the stage produces — a revised file, a report, or both. Include a changelog template.
- **Checklist.** The items the agent checks off as it works through the procedure.
- **Examples.** A bad example (what the stage produces when it fails) and a good example (what it produces when it succeeds). Examples are worth more than pages of instruction.

### Step 4: Build the orchestrator

The orchestrator needs to:

1. Pass the document and previous changelog to each stage.
2. Apply report-based stages (Edit, Check) to the file before passing it to the next stage.
3. Review each changelog for boundary violations.
4. Decide whether to re-enter an earlier stage when a later stage flags a cross-stage issue.
5. Collate changelogs into a completion summary.

This can be a script, a workflow engine, or another LLM call with orchestration instructions. The simplest version is a sequential loop that calls each stage's prompt, checks the output, and passes it to the next stage.

### Step 5: Choose which stages edit the file directly

This is a design decision with tradeoffs:

- **Direct editing** (Write, Style, Copyedit) is simpler — the agent reads the file, modifies it, and outputs the result. The risk is that the agent makes out-of-scope changes.
- **Report-based editing** (Edit, Check) separates assessment from application. The agent produces findings; the orchestrator decides which to apply. This is more work but gives the orchestrator (and the human) a review point before changes are committed.

A reasonable default: stages that make judgement calls about *what* to change (structure, facts) produce reports. Stages that apply known patterns mechanically (vocabulary substitution, punctuation rules) edit directly.

### Step 6: Add change tracking

Every stage must output a changelog alongside the revised document. Without changelogs:

- The orchestrator can't check for boundary violations.
- The human reviewer has to diff before/after to understand what happened.
- You lose the ability to debug the pipeline when something goes wrong.

The changelog format doesn't need to be complex. Location, what changed, and why is enough.

### Step 7: Add human checkpoints

Offer the human a chance to review after the stages where mistakes are most expensive to undo. For the five-stage pipeline, that's after Edit (structural decisions) and after Copyedit (the finished product). Make checkpoints advisory, not blocking — a pipeline that waits indefinitely for human review will never finish.

---

## Limitations and tradeoffs

**It's slower.** Five sequential LLM calls take longer than one. For a short README, the pipeline is overkill. It earns its keep on documents longer than a few hundred words where structure, accuracy, and prose quality all matter.

**It requires good prompts.** Each stage is only as good as its role definition and skill procedure. Vague instructions produce vague results, same as with any LLM workflow. The upfront investment in writing precise role definitions, anti-patterns, and examples is significant.

**Boundary violations happen.** Despite explicit boundaries, agents sometimes edit outside their scope. The orchestrator catches most of these by reviewing changelogs, but some slip through. This is a fundamental limitation of using LLMs for constrained tasks — they follow instructions probabilistically, not deterministically.

**Re-entry is expensive.** Sending a document back to an earlier stage means re-running every stage after it. The pipeline is designed to minimise re-entry (reserve it for severe issues only), but when it happens, the cost is real.

---

## Further reading

If you're working within Kanbanzai, the internal references cover the implementation details:

| Topic | Path |
|-------|------|
| Pipeline stage summary and styleguide slicing | `/Users/samphillips/Dev/kanbanzai/refs/documentation-pipeline.md` |
| Document structure and inverted pyramid | `/Users/samphillips/Dev/kanbanzai/refs/documentation-structure-guide.md` |
| AI artifact detection and removal | `/Users/samphillips/Dev/kanbanzai/refs/humanising-ai-prose.md` |
| Punctuation rules and conventions | `/Users/samphillips/Dev/kanbanzai/refs/punctuation-guide.md` |
| Technical writing fundamentals | `/Users/samphillips/Dev/kanbanzai/refs/technical-writing-guide.md` |

The stage skills live in `/Users/samphillips/Dev/kanbanzai/.kbz/skills/` — each contains the full procedure, checklist, anti-patterns, output format, and examples for that stage. The role definitions live in `/Users/samphillips/Dev/kanbanzai/.kbz/roles/` and define identity, vocabulary, and tool access for each agent.
