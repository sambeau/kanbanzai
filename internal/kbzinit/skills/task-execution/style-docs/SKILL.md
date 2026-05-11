---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: style-docs
description:
  expert: "AI-artifact detection and removal targeting banned vocabulary,
    formulaic sentence patterns, robotic transitions, structural tells, and
    significance inflation — preserving author voice while eliminating
    machine fingerprints"
  natural: "Strip the AI out of the prose — kill the banned words, clichés,
    filler, hedging, and robotic patterns"
triggers:
  - humanise AI prose
  - remove AI artifacts
  - style-check documentation
  - strip AI clichés
  - clean up AI-generated text
  - dehumanise prose
roles: [doc-stylist]
stage: documenting
constraint_level: low
---

## Vocabulary

- **AI artifact** — a word, phrase, or structural pattern that appears at
  statistically elevated rates in AI-generated text; individually minor,
  collectively a fingerprint.
- **banned word** — a word so overused by AI models that it must be replaced on
  every occurrence: delve, leverage, utilize, facilitate, optimize, spearhead,
  amplify, bolster, foster, garner, harness, empower, streamline, elevate,
  underscore, showcase, navigate (abstract), embark, unveil, unlock, unleash.
- **inflated adjective** — an adjective that promises more than it delivers and
  should be cut or replaced with specifics: pivotal, vibrant, meticulous,
  seamless, effortless, cutting-edge, groundbreaking, transformative,
  revolutionary, comprehensive, holistic, robust (non-technical), innovative,
  dynamic.
- **filler phrase** — hedging or qualifying language that adds no meaning:
  "generally speaking", "it could be argued", "it's important to note", "it
  should be noted", "at its core".
- **tricolon** — a group of three parallel items; legitimate as an occasional
  device, robotic when every list has exactly three items.
- **1-2-3 formula** — the rigid paragraph pattern: topic sentence stating claim,
  supporting sentence with generic example, closing sentence restating the
  claim; numbing when every paragraph follows it.
- **robotic transition** — a formal transition word used mechanically at the
  start of sentences: Furthermore, Moreover, Additionally, Subsequently,
  Consequently, Nevertheless, Notably.
- **elegant variation** — swapping synonyms for the same concept to avoid
  repetition ("server" → "the machine" → "the instance" → "the compute
  resource"); confusing in technical writing where consistency is a virtue.
- **copula avoidance** — AI's tendency to replace "is" and "has" with inflated
  alternatives: "serves as" for "is", "boasts" for "has", "features" for "has".
- **significance inflation** — claiming importance without evidence: "plays a
  crucial role", "marks a significant shift", "is a testament to".
- **AI fingerprint cluster** — three or more distinct AI tells in a single
  passage; indicates the passage was generated wholesale and needs rewriting,
  not patching.
- **opening cliché** — a dead-giveaway opening phrase: "In today's digital
  landscape", "In the ever-evolving world of", "When it comes to", "Let's
  dive in".
- **hollow conclusion** — a vague, upbeat ending that adds nothing: "In summary,
  X represents a powerful approach to…"
- **staccato rhetoric** — short fragments arranged for dramatic effect, borrowed
  from copywriting: "No config. No setup. No hassle."
- **faux-insider opener** — a phrase that performs knowledge instead of stating
  it: "Here's what most people get wrong", "Here's the thing", "The truth
  about".

## Anti-Patterns

### Word-Swapping

- **Detect:** Individual banned words are replaced with synonyms but the
  underlying sentence structure remains formulaic.
- **BECAUSE:** AI artifacts rarely appear alone; swapping one word in a sentence
  with three other tells produces text that still reads as machine-generated —
  the sentence needs rewriting, not patching.
- **Resolve:** When a sentence contains a banned word, check for other tells in
  the same sentence and paragraph. If 3+ tells cluster together, rewrite the
  passage from scratch.

### Over-Correction

- **Detect:** Good grammar is broken for "authenticity"; formal language is
  replaced with informal language regardless of context; transition words are
  removed entirely.
- **BECAUSE:** The target is formulaic writing, not formal writing — correct
  grammar is not an AI tell, moderate use of transition words is fine, and
  technical writing is often appropriately formal.
- **Resolve:** Before removing or changing anything, check the "what not to fix"
  list (humanising guide §9). Preserve the author's voice and legitimate
  stylistic choices.

### Pattern Blindness

- **Detect:** Individual tells are fixed but the AI fingerprint cluster goes
  undetected — three or more tells in a single passage are edited one by one
  instead of triggering a full rewrite.
- **BECAUSE:** Word-by-word fixes in a passage that was generated wholesale
  produce Franken-prose — half human edits, half machine patterns — which is
  worse than either pure AI or pure human prose.
- **Resolve:** Before fixing individual tells, scan the passage for clusters. If
  3+ distinct tells appear, flag the passage for rewriting from scratch.

### Vocabulary Police

- **Detect:** Every occurrence of "however" or "also" is flagged; simple
  transition words are treated as AI tells.
- **BECAUSE:** Common transition words are legitimate English; the problem is
  mechanical overuse (starting every other sentence with "Furthermore"), not
  the existence of transition words — over-flagging creates busy work and
  erodes trust in the review.
- **Resolve:** Flag transition words only when they appear mechanically (e.g.,
  every paragraph opens with one) or when a simpler alternative exists; leave
  moderate use alone.

### Content Rewriting

- **Detect:** The style editor changes the meaning of sentences, adds new
  information, or removes factual content.
- **BECAUSE:** The style stage owns vocabulary and sentence patterns, not
  content; changing meaning creates text that hasn't been fact-checked and may
  introduce hallucinations.
- **Resolve:** Replace words and restructure sentences without changing meaning.
  If a sentence's meaning is unclear, flag it for the author rather than
  guessing.

## Checklist

### Vocabulary

- [ ] Every banned word (§1.1 list) has been found and replaced or removed
- [ ] Inflated adjectives and adverbs have been cut or replaced with specifics
- [ ] Abstract metaphor nouns (landscape, tapestry, journey, paradigm shift)
      have been replaced
- [ ] Empty hedging phrases have been cut

### Phrases and patterns

- [ ] Faux-insider openers have been rewritten as direct statements
- [ ] Staccato rhetoric has been combined into flowing sentences
- [ ] "Not just X, but Y" constructions have been simplified
- [ ] Opening clichés have been replaced with the actual first point
- [ ] Hollow conclusions have been replaced with specific takeaways or cut

### Sentence and paragraph structure

- [ ] Lists with exactly 3 items checked — vary where the count is artificial
- [ ] 1-2-3 paragraph formula broken — paragraph lengths vary
- [ ] Robotic transitions replaced with simpler alternatives or cut
- [ ] Elegant variation eliminated — same concept uses same term throughout
- [ ] Copula avoidance fixed — "serves as" → "is", "boasts" → "has" where
      appropriate

### AI fingerprint clusters

- [ ] Passages with 3+ distinct tells have been flagged for rewriting
- [ ] The revised text has been read aloud — it should not sound like a keynote
      speech

## Procedure

### Step 1: Read before editing

Read the entire document before touching anything. Ask:

- Does this say anything substantive, or is it waving its arms?
- Could I replace the subject with a different product and have the text still
  make sense? If so, the text is too generic.
- What is the one thing the reader should take away? Is that thing stated?

### Step 2: Scan for banned words

Search for every word in the banned list (humanising guide §1.1). Replace each
with its simpler alternative or remove it. The replacement table:

| Kill              | Use instead                      |
|-------------------|----------------------------------|
| delve / dive into | examine, explain, look at        |
| leverage          | use                              |
| utilize           | use                              |
| facilitate        | help, support, enable            |
| optimize          | improve                          |
| spearhead         | lead                             |
| amplify           | increase, strengthen             |
| bolster           | support, strengthen              |
| foster            | encourage, support               |
| garner            | get, earn, attract               |
| harness           | use                              |
| empower           | let, allow, enable               |
| streamline        | simplify                         |
| elevate           | raise, improve                   |
| underscore        | show, stress, emphasise          |
| showcase          | show, demonstrate                |
| navigate          | deal with, handle, work through  |
| embark            | start, begin                     |
| unveil            | announce, release, show          |
| unlock            | enable, allow, improve           |
| unleash           | release, enable                  |

### Step 3: Scan for inflated adjectives and abstract metaphors

Cut pivotal, seamless, groundbreaking, cutting-edge, transformative,
revolutionary, comprehensive, holistic, robust (non-technical), innovative,
dynamic, vibrant, meticulous, effortless. Replace abstract metaphors
(landscape, journey, tapestry, paradigm shift) with concrete terms.

### Step 4: Cut empty hedging

Remove "generally speaking", "it could be argued", "it's important to note",
"it should be noted", "it bears mentioning", "at its core", "to some extent",
"it is worth considering". State claims directly. If genuinely uncertain, say
so explicitly: "We haven't measured X yet."

### Step 5: Hunt sentence-level patterns

Check for:

1. **Faux-insider openers** — "Here's the thing", "What nobody tells you" →
   state the fact directly.
2. **Staccato rhetoric** — "No config. No setup. No hassle." → combine into
   a flowing sentence.
3. **"Not just X, but Y"** — drop the contrast. Say what the thing does.
4. **Opening clichés** — "In today's digital landscape" → replace with the
   actual first point.
5. **Hollow conclusions** — "In summary, X represents a powerful approach" →
   end with the last substantive point or cut.

Each of these needs rewriting from scratch — word-swapping will not fix them.

### Step 6: Check paragraph and list patterns

- Are all lists exactly 3 items? Vary where the count is artificial.
- Does every paragraph follow claim-support-restate? Vary paragraph length.
- Are transitions mechanical (every paragraph opens with Furthermore/Moreover)?
  Replace with simpler words or cut.
- Is the same concept called different names in different sentences (elegant
  variation)? Pick one term and use it consistently.

### Step 7: Fix copula avoidance

Replace inflated copula substitutes with the simple word:

| AI version                 | Human version           |
|----------------------------|-------------------------|
| serves as                  | is                      |
| stands as                  | is (or cut)             |
| boasts                     | has                     |
| features                   | has                     |
| offers a diverse array of  | has several             |

"Is" and "has" are fine words. Use them.

### Step 8: Check for AI fingerprint clusters

If any passage has 3+ distinct tells, flag it for full rewriting rather than
individual fixes. Three tells in one passage = generated wholesale. Patching
word by word produces Franken-prose.

### Step 9: Read aloud

Read the revised text aloud. Does it sound like a keynote speech — smooth,
even, relentlessly upbeat? Human prose has texture: short sentences next to
long ones, blunt statements next to nuanced ones, occasional roughness.

### Step 10: Verify preservation

Confirm that:

- Meaning has not changed.
- No factual content has been removed.
- The author's voice is preserved.
- The "what not to fix" list (humanising guide §9) is respected: correct
  grammar intact, formal register preserved where appropriate, moderate
  transition words left alone, lists not removed wholesale.

## Output Format

The Style stage edits the document file directly — it does not just produce a report. Apply all changes to the file, then produce the changelog below as a record of what was changed. The orchestrator reviews the changelog for boundary violations before passing the revised document to the next stage.

```
## Style Edit Report

**Document:** {document path}
**Banned words found:** {count}
**AI fingerprint clusters:** {count}
**Passages rewritten from scratch:** {count}

### Changes

#### {Change description}
- **Location:** {section, line}
- **Type:** {banned-word | inflated-adjective | hedging | cliché |
  structural-pattern | copula-avoidance | fingerprint-cluster}
- **Before:** "{original text}"
- **After:** "{revised text}"
- **Reason:** {why this change was made}

### Flagged for Author
{items where meaning was unclear or specifics are needed from the author}

### Summary
- Words replaced: {count}
- Phrases cut: {count}
- Sentences rewritten: {count}
- Passages rewritten from scratch: {count}
- Items flagged for author: {count}
```

## Examples

### BAD: Word-swapping without rewriting

Before:

> "In today's rapidly evolving landscape, our tool leverages cutting-edge
> technology to streamline workflows. It's not just a parser — it's a
> comprehensive solution that empowers teams to unlock their full potential."

After word-swap:

> "In today's rapidly changing environment, our tool uses modern
> technology to simplify workflows. It's not just a parser — it's a
> complete solution that enables teams to reach their full potential."

**WHY BAD:** Still formulaic. Still reads as AI. The banned words are gone but
the cliché opener, "not just X but Y" construction, and vague claims remain.
Three tells in one passage = rewrite from scratch.

### GOOD: Full rewrite of a fingerprint cluster

Before:

> "In today's rapidly evolving landscape, our tool leverages cutting-edge
> technology to streamline workflows. It's not just a parser — it's a
> comprehensive solution that empowers teams to unlock their full potential."

After rewrite:

> "The tool parses YAML configuration, validates it against a schema, and
> generates typed Go structs. Most configuration errors are caught before
> the code compiles."

**WHY GOOD:** The vague promises are replaced with what the tool actually does.
No banned words. No cliché opener. No "not just X but Y." No inflated
adjectives. The passage has specific, verifiable claims instead of arm-waving.

## Evaluation Criteria

These criteria are for evaluating the style edit output, not for self-evaluation
during the edit. Phrased as gradable questions to support automated
LLM-as-judge evaluation.

1. Are all banned words from the §1.1 list found and handled?
   **Weight: 0.25.**
2. Are AI fingerprint clusters detected and flagged for rewriting (not just
   patched word by word)? **Weight: 0.20.**
3. Are sentence-level patterns (faux-insider, staccato, "not just X but Y",
   clichés, hollow conclusions) identified and rewritten?
   **Weight: 0.20.**
4. Is the "what not to fix" list respected — no over-correction of legitimate
   formal language? **Weight: 0.15.**
5. Does the revised text preserve the original meaning without changing factual
   content? **Weight: 0.10.**
6. Does the revised text sound natural when read aloud — varied rhythm, not
   keynote-speech cadence? **Weight: 0.10.**

## Questions This Skill Answers

- Does this text contain AI writing artifacts?
- Which passages need rewriting from scratch vs individual fixes?
- Is this formal writing or formulaic writing?
- Are there AI fingerprint clusters?
- What's the banned-word count?
- Is the text over-corrected — has authentic voice been removed?
- Does it pass the read-aloud test?