---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: write-docs
description:
  expert: "First-draft technical documentation following inverted-pyramid
    structure with audience-aware progressive disclosure and
    implementation-verified accuracy"
  natural: "Write a first draft of a document — plan it, structure it,
    write it following the inverted pyramid"
triggers:
  - write documentation
  - draft a document
  - create a README
  - write a getting-started guide
  - write a manual
  - author technical documentation
roles: [documenter]
stage: documenting
constraint_level: medium
---

## Vocabulary

- **inverted pyramid** — the principle that the most important information comes first at every level (document, section, paragraph); applies simultaneously to content, tone, and technical depth
- **document purpose statement** — a single sentence capturing what the document must accomplish; if you cannot write it, you are not ready to write the document
- **key messages** — the 3–5 things the reader must understand after reading; everything in the document supports one of these; everything else gets cut
- **sentence outline** — an outline written as complete sentences rather than topic keywords; sentences force verbs and therefore meaning, revealing gaps that keywords hide
- **audience assumption** — an explicit statement of what knowledge the reader brings; all our documentation assumes Git fluency and command-line comfort; individual documents refine further
- **source of truth** — the implementation (code, configuration, runtime behaviour) is the authority for facts; design documents provide concepts and intentions but must be verified against code
- **progressive disclosure** — broad concepts before specific details; accessible language at the top, technical specifics deeper where the reader has chosen to look
- **front-loading** — placing the most important words first in headings, list items, and paragraph openings; the scanner's eye hits the first few words of each line
- **document type** — the structural template that matches the document's purpose: README, getting-started guide, manual, reference, or design document
- **tone gradient** — the shift from conversational and accessible at the top of a document or section to precise and formal as detail increases
- **cross-reference** — a link to a concept's home document rather than a duplicated explanation; each concept has exactly one home
- **opening** — the first thing the reader sees; must answer: what is this, who is it for, what will you learn
- **heading skeleton** — the ordered list of all headings in a document; if read alone, it should tell the document's story
- **topic sentence** — the first sentence of a paragraph, stating the paragraph's main point; a reader who only reads first sentences should get the gist
- **structural template** — the predefined section order for a given document type; described in `refs/documentation-structure-guide.md` §7
- **verification note** — a record of a factual claim checked against the implementation, or flagged because it could not be verified

## Anti-Patterns

### Writing Without a Plan

- **Detect:** Prose is started before defining purpose, audience, key messages, or
  outline.
- **BECAUSE:** Unplanned documents meander, bury key points, and require structural
  rework later — the most expensive kind of editing. Planning is cheap; restructuring
  is not.
- **Resolve:** Complete steps 1–5 of the planning procedure before writing any prose.

### Bottom-Up Structure

- **Detect:** Background, context, or history appears before the key point; the reader
  must scroll to find what the document is actually about.
- **BECAUSE:** Most readers scan from the top. If the key point is buried under context,
  it is invisible to the majority of readers who never scroll that far — violates the
  inverted pyramid at every level.
- **Resolve:** Open every document and every section with its key point. Move context
  and background below.

### Audience Mismatch

- **Detect:** A section aimed at designers uses command-line syntax without explanation,
  or a section aimed at developers over-explains basic concepts.
- **BECAUSE:** Writing at the wrong technical level wastes the reader's time — too
  technical and designers disengage, too basic and developers lose trust in the
  document's value.
- **Resolve:** State audience assumptions explicitly. Use the tone gradient and
  progressive disclosure — accessible at the top, technical at depth.

### Design-Doc-as-Truth

- **Detect:** Facts are copied from design documents or specifications without verifying
  against the implementation.
- **BECAUSE:** Design documents describe intentions; implementations describe reality —
  they diverge. Documentation that contradicts the implementation is wrong regardless of
  what the design says.
- **Resolve:** Use design documents for concepts and intentions. Verify every factual
  claim against code, configuration, or runtime behaviour.

### Summary First

- **Detect:** The introduction or summary is written before the body.
- **BECAUSE:** A summary written before the body reflects what you planned to write, not
  what you actually wrote. It will be inaccurate and need rewriting anyway.
- **Resolve:** Write the body first. Write the opening, summary, or abstract last.

## Checklist

### Planning

- [ ] The document's purpose is captured in a single sentence
- [ ] The document type has been identified and the appropriate structure chosen
- [ ] The target audience is identified, with assumptions stated
- [ ] 3–5 key messages are listed
- [ ] A sentence outline is written (sentences, not keywords)
- [ ] Key examples and figures are drafted before the prose
- [ ] Facts are verified against the implementation, not design documents alone

### Structure

- [ ] The opening answers: what is this, who is it for, what will you learn
- [ ] The inverted pyramid is followed at document, section, and paragraph level
- [ ] Tone is more accessible at the top, more precise deeper in
- [ ] Technical depth increases as the reader goes deeper
- [ ] Each section opens with its key point
- [ ] Each section covers one topic
- [ ] Headings form a readable outline on their own

## Procedure

### Step 1: Define the document's purpose

Write a single sentence that captures what the document must accomplish. This
sentence constrains every decision that follows — what to include, what to cut,
how deep to go. If you cannot write this sentence, you are not ready to write.

### Step 2: Identify the document type

Choose the structural template that matches the purpose: README, getting-started
guide, manual, reference, or design document. Read the structure for your type in
`refs/documentation-structure-guide.md` §7. The template gives you a section
order; do not invent one from scratch.

### Step 3: State audience assumptions

Who is the reader? What do they already know? What are they trying to accomplish?
Write these down explicitly. All our documentation assumes Git fluency and
command-line comfort. State any assumptions beyond these for this specific
document.

### Step 4: List 3–5 key messages

These are the things the reader must understand after reading. They become the
spine of your structure. Everything in the document supports one of these
messages. Everything that does not will be cut.

### Step 5: Create a sentence outline

Write each section's content as a complete sentence, not a keyword. Sentences
force verbs and therefore meaning. Compare: *"Configuration"* tells you nothing;
*"The user creates a configuration file that controls which features are
enabled"* tells you exactly what the section says. The outline also reveals gaps —
places where you do not yet know what you want to tell the reader.

### Step 6: Draft examples and figures first

Code examples, diagrams, tables, and transcripts are often the most valuable part
of a document. Draft them before writing prose. Prose exists to connect and
explain the examples, not the other way around.

### Step 7: Write the body following the inverted pyramid

At every level — document, section, paragraph — place the most important
information first. Follow the tone gradient: accessible and conversational at the
top, precise and formal at depth. Each section opens with its key point; each
paragraph opens with its topic sentence.

### Step 8: Verify facts against the implementation

Check every factual claim against code, configuration, or runtime behaviour.
Design documents provide concepts and intentions but must not be trusted as the
sole source of truth. Record what you verified and flag anything you could not
check.

### Step 9: Write the opening and any summary last

Summarise what you actually wrote, not what you planned to write. The opening
must answer three questions: what is this, who is it for, what will you learn.
Write it only after the body is complete.

## Output Format

The Write stage edits the document file directly — it does not just produce a report. Write the draft (or revise the existing document) in place, then produce the metadata and verification notes below as a record of what was produced. When the input is an existing document, edit the file to improve it rather than returning an assessment of what could be improved.

```
## Document Draft

**Purpose:** {one-sentence purpose statement}
**Type:** {README | getting-started | manual | reference | design}
**Audience:** {who this is for and what they already know}

**Key messages:**
1. {message}
2. {message}
3. {message}

---

{document body following the chosen structural template}

---

**Verification notes:**
- {list of facts verified against implementation}
- {list of facts that could not be verified — flagged for check stage}
```

## Examples

### BAD: Bottom-up opening

```
## Configuration

The configuration system was originally designed in Q3 2024 as part of
the platform modernisation initiative. It replaced the legacy INI-based
system that had been in use since version 1.2. The team evaluated several
alternatives including TOML, JSON, and YAML before settling on YAML due
to its readability advantages. Configuration files are stored in the
`.config/` directory.
```

**WHY BAD:** The reader scrolls through four sentences of history before learning
where configuration files actually live. The key fact — the location and format —
is buried at the end. A scanning reader stops before reaching it. This violates
the inverted pyramid at the section level.

### GOOD: Inverted-pyramid opening

```
## Configuration

Configuration files live in `.config/` and use YAML format. Each file
controls one aspect of the system — see the sections below for details.

The system migrated from INI to YAML in v2.0 for readability. If you are
upgrading from v1.x, see the migration guide.
```

**WHY GOOD:** The key fact is first. The reader who stops after one sentence still
knows where configuration lives and what format it uses. History appears only
where the reader who needs it will find it — below the key point, clearly marked
as upgrade context.

## Evaluation Criteria

These criteria are for evaluating the document draft, not for self-evaluation
during writing. They are phrased as gradable questions to support automated
LLM-as-judge evaluation.

1. Does the document open with a clear purpose statement that answers what, who,
   and why? **Weight: 0.20.**
2. Does every section follow the inverted pyramid — key point first, detail in
   descending order? **Weight: 0.20.**
3. Are audience assumptions stated explicitly, and does the tone gradient match
   them? **Weight: 0.15.**
4. Are facts verified against the implementation rather than design documents
   alone? **Weight: 0.15.**
5. Does the heading skeleton tell a coherent story when read in isolation?
   **Weight: 0.15.**
6. Are examples and figures present where they would communicate more efficiently
   than prose? **Weight: 0.15.**

## Questions This Skill Answers

- How do I structure a new document from scratch?
- What should I write first when starting a document?
- How do I make one document work for both designers and developers?
- Which structural template should I use for this document?
- How do I verify facts in documentation?
- When should I write the introduction?
- How detailed should my outline be before I start writing?