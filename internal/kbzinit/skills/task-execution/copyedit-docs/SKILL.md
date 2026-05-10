---
name: copyedit-docs
description:
  expert: "Sentence-level copy editing enforcing active voice, verb clarity,
    front-loaded sentence structure, punctuation hygiene, and consistency —
    the final polish before publication"
  natural: "Polish the sentences — fix passive voice, simplify punctuation,
    tighten word choice, and make everything consistent"
triggers:
  - copy edit documentation
  - polish prose
  - fix passive voice
  - simplify sentences
  - proofread documentation
  - final edit
roles: [doc-copyeditor]
stage: documenting
constraint_level: medium
---

## Vocabulary

- **active voice** — the subject performs the action ("The installer copies the
  files"); the default for technical writing.
- **passive voice** — the subject receives the action ("The files are copied by the
  installer"); deliberate use only — in error messages, when the receiver matters more
  than the actor, or when the actor is unknown.
- **smothered verb** — a nominalisation that buries the action in a noun: "perform an
  installation" → "install", "make a determination" → "determine", "carry out an
  evaluation" → "evaluate".
- **sentence weight** — the principle that the most important information belongs at the
  start of the sentence; readers scan beginnings in an F-pattern.
- **serial comma** — the comma before the conjunction in a list of three or more items;
  always use it.
- **em-dash hygiene** — maximum one em-dash pair per paragraph; prefer full stops for
  sentence breaks and commas or parentheses for asides.
- **compound modifier** — two or more words acting as a single adjective before a noun,
  joined by a hyphen: "a well-established principle"; drop the hyphen after the noun:
  "the principle is well established".
- **parallel structure** — matching grammatical form in lists, comparisons, and
  contrasts; if one item is a verb phrase, all items should be verb phrases.
- **topic sentence** — the first sentence of each paragraph states its main point; a
  reader who only reads first sentences gets the gist.
- **reading rhythm** — varied sentence lengths creating natural cadence; short sentences
  for emphasis, longer sentences for explanation; monotone length is a readability
  problem.
- **contraction consistency** — using contractions (it's, you'll, don't) consistently
  throughout, or not using them at all; never mixing styles within a document.
- **sentence-case capitalisation** — capitalise only the first word and proper nouns in
  headings, labels, and table headers; the default style.
- **en dash** — used for ranges (pages 15–32, 2020–2024) with no spaces, and as
  parenthetical dashes with spaces on each side; distinct from hyphens and em dashes.
- **front-loading** — placing the most important words at the start of each sentence;
  "Create the right chart" not "The Recommended Charts command on the Insert tab
  recommends charts".

## Anti-Patterns

### Passive Default

- **Detect:** Passive voice is used without a deliberate reason — the text defaults to
  "X is done by Y" when "Y does X" would be clearer and shorter.
- **BECAUSE:** Passive voice is indirect; it adds words, obscures the actor, and makes
  instructions harder to follow — active voice is clearer in most technical writing
  contexts.
- **Resolve:** Switch to active voice unless the passive is deliberate: error messages
  ("That page can't be found"), emphasis on the receiver ("The transaction is committed
  when..."), or unknown actor.

### Punctuation Complexity

- **Detect:** Semicolons, nested clauses, and em-dash chains are used where two simple
  sentences would be clearer.
- **BECAUSE:** Complex punctuation increases cognitive load; most readers parse two
  short sentences faster than one long sentence with internal punctuation — simplicity
  is the default.
- **Resolve:** Try splitting into two sentences first. Use a semicolon only between
  closely related independent clauses where the connection is genuinely important. Limit
  em-dash pairs to one per paragraph.

### Consistency Drift

- **Detect:** Contractions mixed with spelled-out forms (it's / it is), capitalisation
  varies between headings, terminology shifts (dashboard / control panel), formatting is
  inconsistent.
- **BECAUSE:** Inconsistency distracts the reader and signals carelessness; in
  technical documentation, inconsistent terminology also creates genuine confusion about
  whether two terms refer to the same thing.
- **Resolve:** Pick one form and apply it throughout. For terminology, use the term
  defined in the vocabulary or glossary. For contractions, use them consistently or not
  at all.

### Over-Editing Voice

- **Detect:** The copy editor removes distinctive phrasing, flattens personality, or
  normalises every sentence to the same structure in pursuit of "correctness".
- **BECAUSE:** Good copy editing improves clarity without removing the author's voice;
  over-edited prose reads as sterile and generic — the goal is polished human writing,
  not standardised machine output.
- **Resolve:** Preserve distinctive phrasing that is clear and correct. Fix unclear or
  incorrect sentences. Leave clear and correct sentences alone, even if you would have
  written them differently.

### Abbreviation Sprawl

- **Detect:** Abbreviations or acronyms used without definition on first use, or
  defined once and then used excessively when the full term would be clearer.
- **BECAUSE:** Each abbreviation adds cognitive load; readers must remember what it
  stands for throughout the document — minimise abbreviations and define every one at
  first use.
- **Resolve:** Define at first use. Don't introduce an abbreviation used only once —
  just spell it out. Don't use abbreviations in headings unless universally known (API,
  URL, HTTP).

## Checklist

### Sentences

- [ ] Each sentence communicates one idea (or two closely related ideas)
- [ ] The most important information comes first in each sentence
- [ ] Subject and verb are close together
- [ ] Sentences over 25–30 words have been considered for splitting
- [ ] No more than two embedded clauses per sentence

### Voice and verbs

- [ ] Active voice is used by default
- [ ] Passive voice is used deliberately (error messages, receiver emphasis, unknown
      actor)
- [ ] Smothered verbs replaced with direct verbs ("perform installation" → "install")
- [ ] No combination of smothered verb + passive voice
- [ ] Instructions use imperative mood (start with a verb)

### Punctuation

- [ ] Serial (Oxford) comma used in lists of three or more
- [ ] One space after full stops
- [ ] Em dashes limited to one pair per paragraph; no em dashes in headings
- [ ] Semicolons used only between closely related independent clauses
- [ ] Apostrophes correct (its vs it's, no apostrophes in plurals)
- [ ] Hyphens in compound modifiers before nouns, dropped after
- [ ] En dashes for ranges (no spaces) and parenthetical asides (with spaces)

### Consistency

- [ ] Contractions used consistently (or not used at all)
- [ ] Sentence-case capitalisation in headings
- [ ] One term per concept throughout
- [ ] Abbreviations defined at first use
- [ ] No ALL CAPS for emphasis

## Procedure

### Step 1: Fix voice

Read each sentence asking: who is doing what? If the subject isn't performing the
action, switch to active voice — unless there's a deliberate reason to keep passive
(error message, receiver emphasis, unknown actor).

### Step 2: Unbury smothered verbs

"Perform an installation" → "install". "Make a determination" → "determine". "Establish
connectivity" → "connect". Never combine a smothered verb with passive voice.

### Step 3: Check sentence length and complexity

Sentences over 25–30 words get a second look. More than two embedded clauses — split.
Front-load the important information. Keep subject and verb close together.

### Step 4: Check punctuation systematically

Work through each punctuation rule:

- **Serial comma** in every list of three or more.
- **Em dashes**: max one pair per paragraph; prefer full stops.
- **Semicolons**: try two sentences first.
- **Colons**: only after a lead-in that references the list.
- **Apostrophes**: its / it's, no plural apostrophes.
- **Hyphens**: compound modifiers before the noun, not after.
- **En dashes**: ranges (no spaces), parenthetical asides (with spaces).
- **One space** after full stops.

### Step 5: Check parallel structure

Check every list. If one item starts with a verb, all items start with a verb. If one
item is a complete sentence, all items end with full stops. Matching grammatical form in
comparisons and contrasts.

### Step 6: Check capitalisation

Sentence case for headings and labels. No ALL CAPS for emphasis. Proper nouns
capitalised, common nouns not. The first word after a colon in a title is capitalised;
in running text, lowercase.

### Step 7: Check abbreviations

Defined at first use? Used only once (just spell it out)? Not in headings unless
universally known (API, URL, HTTP)?

### Step 8: Check consistency throughout

Contractions used uniformly? Same term for the same concept? Formatting consistent? No
drift between sections.

### Step 9: Check reading rhythm

Are sentence lengths varied? Short sentences for emphasis, longer for explanation? If
every sentence is the same length, vary them. Monotone rhythm is a readability problem.

### Step 10: Final read-through

Read the whole document looking for anything the previous steps missed. Check that no
meaning was changed during editing. Every edit must preserve the author's intent.

## Output Format

The Copyedit stage edits the document file directly — it does not just produce a report. Apply all changes to the file, then produce the changelog below as a record of what was changed. The orchestrator reviews the changelog for boundary violations before passing the revised document to the final checkpoint.

```
## Copy Edit Report

**Document:** {document path}
**Sentences revised:** {count}
**Passive → active conversions:** {count}
**Punctuation fixes:** {count}

### Changes

#### {Change description}
- **Location:** {section, line}
- **Type:** {passive-voice | smothered-verb | sentence-length | punctuation |
  parallel-structure | capitalisation | abbreviation | consistency | rhythm}
- **Before:** "{original}"
- **After:** "{revised}"

### Consistency Notes
{any document-wide consistency decisions made — e.g., "contractions used throughout",
"dashboard not control panel"}

### Summary
- Passive voice conversions: {count}
- Smothered verbs fixed: {count}
- Sentences split: {count}
- Punctuation corrections: {count}
- Consistency fixes: {count}
```

## Examples

### BAD: Copy edit that changes meaning

```
Before:
> "The configuration system validates input before any changes are
> persisted to disk."

After:
> "The configuration system validates input."
```

**WHY BAD:** The copy editor removed the second clause, changing the meaning. The
original sentence communicates that validation happens before persistence — a meaningful
technical detail. Copy editing preserves meaning. If the sentence is clear and correct,
leave it alone.

### GOOD: Copy edit that improves clarity without changing meaning

```
Before:
> "An installation of the software can be performed by the user by
> means of running the setup wizard, which will carry out a
> determination of the system requirements."

After:
> "Run the setup wizard to install the software. The wizard checks
> your system requirements first."
```

**WHY GOOD:** Two smothered verbs unburied ("perform an installation" → "install",
"carry out a determination" → "checks"). Passive voice switched to active and
imperative. One long sentence split into two. Meaning preserved — both versions say
the same thing, but the revised version says it in fewer words with clearer structure.

## Evaluation Criteria

These criteria are for evaluating the copy edit output, not for self-evaluation during
the edit. They are phrased as gradable questions to support automated LLM-as-judge
evaluation.

1. Is passive voice converted to active where appropriate, with deliberate passive use
   preserved? **Weight: 0.20.**
2. Are smothered verbs identified and replaced with direct verbs?
   **Weight: 0.20.**
3. Is punctuation correct throughout — serial comma, em-dash limits, apostrophes,
   hyphens? **Weight: 0.20.**
4. Is the document internally consistent — contractions, capitalisation, terminology,
   formatting? **Weight: 0.15.**
5. Is meaning preserved in every edit — no factual content removed or changed?
   **Weight: 0.15.**
6. Is reading rhythm varied — not every sentence the same length?
   **Weight: 0.10.**

## Questions This Skill Answers

- Is this document using active voice consistently?
- Are there smothered verbs hiding the action?
- Is punctuation correct and consistent?
- Is the document internally consistent in style?
- Are sentences too long or too complex?
- Is abbreviation use minimal and properly defined?
- Does the prose read naturally with varied rhythm?