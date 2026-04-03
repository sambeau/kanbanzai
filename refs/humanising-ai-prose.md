# Humanising AI Prose: A Style Guide

This guide is a practical reference for editors revising AI-generated text into natural,
human-sounding prose. It is designed to complement standard writing style guides (such as
the Google Developer Documentation Style Guide or the Microsoft Writing Style Guide) rather
than replace them.

Use this document as a checklist during editing. If a draft triggers multiple items from
the lists below, the passage probably needs rewriting from scratch rather than
word-for-word fixes.

> **Source material.** This guide draws on Wikipedia's
> [Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing),
> published research on LLM lexical overrepresentation, and practical observations from
> editing AI-generated technical content.

---

## 1. Vocabulary

AI models regress to the statistical mean of their training data. The result is a narrow,
recognisable vocabulary that sounds authoritative but says very little. Strip it out.

### 1.1 Banned words

These words appear at vastly elevated rates in post-2022 text. Replace or remove every
occurrence. There is always a simpler, more precise alternative.

| Kill                | Simpler alternative              |
|---------------------|----------------------------------|
| delve / dive into   | examine, explain, look at        |
| leverage            | use                              |
| utilize             | use                              |
| facilitate          | help, support, enable            |
| optimize            | improve                          |
| spearhead           | lead                             |
| amplify             | increase, strengthen             |
| bolster             | support, strengthen              |
| foster              | encourage, support               |
| garner              | get, earn, attract               |
| harness             | use                              |
| empower             | let, allow, enable               |
| streamline          | simplify                         |
| elevate             | raise, improve                   |
| underscore          | show, stress, emphasise          |
| showcase            | show, demonstrate                |
| navigate (abstract) | deal with, handle, work through  |
| embark              | start, begin                     |
| unveil              | announce, release, show          |
| unlock              | enable, allow, improve           |
| unleash             | release, enable                  |

### 1.2 Inflated adjectives and adverbs

These words promise more than they deliver. Replace them with something specific or cut
them entirely.

- **Cut without replacement (almost always filler):** pivotal, vibrant, meticulous,
  seamless, effortless, cutting-edge, groundbreaking, transformative, revolutionary,
  game-changing, comprehensive, holistic, robust (when not describing fault-tolerance),
  innovative, dynamic.
- **Replace with specifics:** Instead of "a comprehensive solution," describe what the
  tool actually does. Instead of "robust performance," give a number or a comparison.

### 1.3 Abstract nouns used as metaphors

AI text leans on a handful of metaphors so heavily that they have become meaningless.

| Avoid             | Problem                                                      |
|-------------------|--------------------------------------------------------------|
| landscape         | Vague. Say what you mean: "the market," "the ecosystem."     |
| tapestry          | Almost never appropriate in technical writing.                |
| testament         | Inflated. "X shows Y" works better than "X is a testament."  |
| journey           | Overused to the point of parody. Say "process" or "effort."  |
| paradigm shift    | Rarely accurate. Describe the actual change.                 |
| ecosystem         | Fine in biology; check whether "system" or "tools" is meant. |
| synergy           | Say what the actual combined effect is.                      |

### 1.4 Empty hedging

AI models hedge to avoid being wrong. In technical writing, hedging makes you sound
uncertain and wastes the reader's time.

**Cut these phrases and make the claim directly:**

- "Generally speaking" → (just state it)
- "It could be argued that" → (argue it)
- "It is worth considering" → (consider it)
- "To some extent" → (quantify or cut)
- "It's important to note that" → (just state the note)
- "It should be noted that" → (state it)
- "It bears mentioning" → (mention it)
- "At its core" → (cut it)

If you genuinely are uncertain, say so explicitly: "We haven't measured X yet" is honest.
"It could potentially perhaps be the case that X" is noise.

---

## 2. Banned phrases and sentence patterns

If any of these patterns appear, the sentence needs rewriting from scratch. Word-swapping
will not fix the underlying problem.

### 2.1 Faux-insider openers

These phrases perform knowledge instead of stating it. They are rhetorical tricks borrowed
from listicles and marketing copy.

- "Here's what most people get wrong…"
- "Here's the thing…" / "Here's why…"
- "Here's the secret…" / "The trick is…"
- "What nobody tells you…"
- "The truth about…"
- "Let's be honest…"

**Fix:** State the fact directly. If there is a genuine misconception worth correcting,
describe the misconception and the correction as plain exposition.

### 2.2 Staccato rhetoric

Short fragments arranged for dramatic effect. This is copywriting technique, not
technical prose.

| Pattern                     | Example                                   |
|-----------------------------|-------------------------------------------|
| Parallel fragments          | "No config. No setup. No hassle."         |
| Setup-reversal              | "We thought X. We were wrong."            |
| Fragment-as-punchline       | Ending a paragraph with a 3–5 word punch. |
| "And" for false drama       | "It parses YAML. And it validates it."    |
| "Not X. But Y."             | "It's not a framework. It's a mindset."   |

**Fix:** Combine into a flowing sentence that explains the *why*. "We expected X, but
testing showed Y because Z" is more useful than the dramatic pause.

### 2.3 "Not just X, but Y" constructions

AI models overuse negative parallelisms that sound like they are correcting a misconception
the reader never had.

- "It's not just a tool — it's a philosophy."
- "Not only does it parse YAML, but it also validates schemas."
- "This isn't merely a refactor. It's a rethinking of the entire approach."

**Fix:** Drop the contrast. Say what the thing actually does: "It parses YAML and validates
schemas." If the contrast is genuinely important, make sure the reader actually holds the
misconception you are correcting.

### 2.4 Opening clichés

These openers are dead giveaways. Replace them with your actual first point.

- "In today's digital landscape…"
- "In the ever-evolving world of…"
- "In an era where…"
- "When it comes to…"
- "At its core…"
- "Let's dive in."

### 2.5 Hollow conclusions

AI text often ends with a vague, upbeat summary that adds nothing.

- "In summary, X represents a powerful approach to…"
- "By leveraging X, teams can unlock…"
- "Overall, X stands as a testament to…"
- "Despite challenges, the future looks promising."

**Fix:** End with the last substantive point. If the piece needs a conclusion, summarise
the specific takeaways, not the vibes.

---

## 3. Sentence and paragraph structure

### 3.1 The tricolon problem

AI loves triplets. Three adjectives, three bullet points, three parallel clauses. One or
two triplets in a document is fine — it is a legitimate rhetorical device. But when every
list has exactly three items and every noun has exactly two adjectives, the rhythm becomes
robotic.

**Symptoms:**

- "enthusiasm, experience, and expertise"
- "fast, flexible, and reliable"
- "designed, developed, and deployed"
- Every bulleted list has 3 or 5 items.

**Fix:** Vary list lengths. Use two items. Use four. Use one sentence instead of a list.
If you genuinely have three things to say, say them — but break the pattern elsewhere.

### 3.2 The 1-2-3 paragraph formula

AI often writes in rigid, predictable paragraphs:

1. Topic sentence stating the claim.
2. Supporting sentence with a generic example.
3. Closing sentence restating the claim in different words.

Every paragraph follows this pattern, creating a numbing rhythm.

**Fix:** Vary paragraph length. Some paragraphs should be one sentence. Some should be
five. Lead with an example sometimes. Occasionally let the evidence speak without a
summary sentence at the end.

### 3.3 Robotic transitions

AI overuses formal transition words at the start of sentences.

| Overused               | Simpler alternative           |
|------------------------|-------------------------------|
| Furthermore            | Also, and                     |
| Moreover               | Also, and, plus               |
| Additionally           | Also                          |
| Subsequently           | Then, later, after that       |
| Consequently           | So                            |
| Nevertheless           | But, still, even so           |
| It is worth noting     | (cut entirely)                |
| Notably                | (cut, or fold into sentence)  |

Not every sentence needs a signpost. If the logic flows naturally, the reader does not
need a transition word to follow it.

### 3.4 Elegant variation (thesaurus syndrome)

AI avoids repeating words, often to absurd effect. A "server" becomes "the machine," then
"the instance," then "the compute resource." This is confusing, not elegant.

**Fix:** In technical writing, consistency is a virtue. Call a server a server every time.
Repeat the term. The reader is not reading for literary variety — they are reading for
clarity.

### 3.5 Copula avoidance

AI text systematically avoids "is" and "are," replacing them with inflated alternatives.

| AI version                          | Human version                  |
|-------------------------------------|--------------------------------|
| "serves as the primary entry point" | "is the primary entry point"   |
| "stands as a reminder"              | "is a reminder" (or just cut)  |
| "boasts a wide range of features"   | "has many features"            |
| "features four separate spaces"     | "has four spaces"              |
| "offers a diverse array of options" | "has several options"          |

"Is" and "has" are fine words. Use them.

---

## 4. Punctuation

### 4.1 Em-dash overload

AI uses em dashes as a universal connector — joining clauses, inserting asides, replacing
commas, colons, and full stops — in a way that becomes a visual fingerprint of generated
text.

**Rules:**

- **One em-dash pair per paragraph, maximum.** If you have more, convert the extras to
  commas, parentheses, colons, or full stops.
- **Never use em dashes in headings or titles.**
- Prefer a full stop and a new sentence over an em-dash-connected thought. Shorter
  sentences are almost always clearer.

### 4.2 Colon overload in headings

AI-generated headings often use colons to create a two-part structure:

- "Deployment: A Practical Guide"
- "Error Handling: Best Practices and Patterns"

One or two of these in a document is fine. When every heading follows the pattern, it
reads like a slide deck. Vary your heading style.

### 4.3 Excessive bolding

AI bolds key terms as if writing study notes. In technical prose, bold should be rare:
use it for introducing a term for the first time, or for UI element names if your style
guide requires it. Do not bold every occurrence of a concept, and do not bold phrases for
emphasis in running text.

### 4.4 Curly quotes and apostrophes

Some AI models output curly (typographic) quotation marks — "like this" — instead of
straight quotes — "like this". If your project uses straight quotes in code and prose
(as most technical projects do), search-and-replace curly variants.

---

## 5. Content and substance

### 5.1 Strip significance inflation

AI text constantly tells you how important things are instead of showing you. Watch for:

- "X plays a crucial/vital/pivotal role in…"
- "X marks a significant shift in…"
- "X underscores the importance of…"
- "X is a testament to…"
- "This highlights the enduring legacy of…"
- "Contributing to the broader…"

**Fix:** Delete the significance claim. If the importance is not obvious from the facts
themselves, add concrete evidence (a number, a comparison, a consequence) instead of an
adjective.

### 5.2 Replace vague claims with specifics

AI writing is often "low signal" — many words conveying little information.

| Vague (AI)                                       | Specific (human)                                    |
|--------------------------------------------------|-----------------------------------------------------|
| "a comprehensive solution"                       | "it automates monthly payroll billing"               |
| "significantly improves performance"             | "reduces p99 latency from 200ms to 45ms"            |
| "a wide range of use cases"                      | "batch processing, streaming, and ad-hoc queries"   |
| "designed with scalability in mind"              | "tested to 10,000 concurrent connections"            |
| "leverages cutting-edge technology"              | "uses gRPC for transport and Raft for consensus"    |

If you cannot replace a vague claim with a specific one, the claim probably should not be
in the document.

### 5.3 Cut superficial analysis

AI appends shallow commentary to facts, usually with a present participle ("-ing") phrase.

- "The library was released in 2019, **marking a significant milestone** in the
  project's evolution."
- "The API supports pagination, **ensuring that clients can efficiently retrieve** large
  datasets."
- "It was written in Go, **reflecting the team's commitment to** performance."

**Fix:** Delete the participle phrase. The fact stands on its own. If the analysis is
genuinely important, give it its own sentence with evidence.

### 5.4 Remove promotional language

AI drifts toward advertising copy, even when describing mundane technical components.

**Words that signal promotion:** boasts, showcases, enhances, exemplifies, commitment to,
nestled, in the heart of, renowned, featuring, diverse array, natural beauty.

**Fix:** Use neutral, descriptive language. "The library provides three serialization
formats" not "The library boasts a diverse array of powerful serialization options."

### 5.5 Remove "challenges and future outlook" boilerplate

AI loves to end with a section about challenges faced and future prospects. The formula
is: "Despite [positive words], X faces challenges including [generic list]. Despite these
challenges, [optimistic speculation]."

If there are genuine challenges worth documenting, describe them concretely. Otherwise,
cut the section.

---

## 6. Structural tells

### 6.1 Inline-header lists

AI formats bulleted lists with a bold header, a colon, and a description on the same line:

- **Parsing:** The system parses incoming YAML files and validates their structure.
- **Routing:** Requests are routed to the appropriate handler based on the URL path.
- **Logging:** All events are logged to stdout in JSON format.

This format is occasionally useful, but AI uses it for everything. When the descriptions
are a single sentence, prose is usually better: "The system parses incoming YAML, routes
requests by URL path, and logs events as JSON to stdout."

### 6.2 Unnecessary tables

AI creates small tables that would work better as a sentence or two. If a table has only
two columns and fewer than four rows, consider whether prose would be clearer.

### 6.3 Title case in headings

AI defaults to Title Case for All Headings. Most technical style guides prefer sentence
case (only capitalise the first word and proper nouns). Check your project's convention
and apply it consistently.

### 6.4 Emoji in technical prose

Do not use emoji in headings, bullet points, or running text. They are appropriate in
casual communication (chat, social media) but not in technical documentation.

---

## 7. Editing process

### Step 1: Read the whole piece first

Do not start fixing word by word. Read the entire draft and ask:

- Does this say anything substantive, or is it just waving its arms?
- Could I replace the subject with a completely different product/project and have the
  text still make sense? If so, the text is too generic.
- What is the one thing the reader should take away? Is that thing actually stated?

### Step 2: Delete first, rewrite second

Cut every sentence that fails this test: "Does this sentence contain information the
reader did not already have?" Significance claims, restated conclusions, and vague
analyses almost always fail.

### Step 3: Check for the AI fingerprint cluster

AI tells rarely appear alone. If you find one (an em dash, a "delve," a tricolon), search
for others. The presence of three or more distinct tells in a single passage means the
passage was likely generated wholesale and needs rewriting, not patching.

### Step 4: Read it aloud

AI prose has a distinctive cadence — smooth, even, and relentlessly upbeat. Human prose
has texture: short sentences next to long ones, blunt statements next to nuanced ones,
occasional roughness. If the text sounds like a keynote speech when read aloud, it needs
more variation.

### Step 5: Add your actual opinion

AI is trained to be neutral and inoffensive. Technical writing benefits from a point of
view: "We chose X over Y because Z" is more useful than "Both X and Y offer compelling
advantages for modern development workflows." If you have a recommendation, state it.
If you have a caveat, state it. The reader is here for your judgement, not for a
diplomatic summary of all possible positions.

---

## 8. Quick-reference checklist

Use this when reviewing a draft. If you check more than three boxes, consider rewriting
the passage rather than editing it.

- [ ] Contains words from the banned list (§1.1)
- [ ] Uses inflated adjectives with no specifics (§1.2)
- [ ] Opens with a cliché (§2.4)
- [ ] Contains faux-insider phrasing (§2.1)
- [ ] Uses staccato rhetoric or dramatic fragments (§2.2)
- [ ] Uses "not just X, but Y" constructions (§2.3)
- [ ] Every list has exactly three items (§3.1)
- [ ] Every paragraph follows the same claim-support-restate structure (§3.2)
- [ ] More than two em dashes per paragraph (§4.1)
- [ ] Bold text used for emphasis in running prose (§4.3)
- [ ] Claims of significance without evidence (§5.1)
- [ ] Vague claims that could apply to any product (§5.2)
- [ ] Participle phrases providing shallow analysis (§5.3)
- [ ] Promotional adjectives (§5.4)
- [ ] "Despite challenges" formula (§5.5)
- [ ] Ends with a hollow conclusion (§2.5)

---

## 9. What not to "fix"

Not everything that looks like AI writing is AI writing, and not every AI pattern is
automatically bad. Do not over-correct in these areas:

- **Correct grammar and spelling.** Good grammar is not an AI tell. Do not introduce
  errors for "authenticity."
- **Formal register.** Technical writing is often formal. Formality is fine. The problem
  is *formulaic* writing, not formal writing.
- **Transition words in moderation.** "However" and "also" are useful words. The problem
  is using them as mechanical sentence starters on every other line.
- **The word "is."** AI avoids it. You should not. But you also should not insert it where
  a more precise verb genuinely works better.
- **Lists.** Bulleted lists are a legitimate formatting tool. The problem is AI's
  tendency to reach for them when prose would work, and to format them with the
  bold-header-colon pattern every time.

---

## 10. Summary of principles

1. **Specifics beat adjectives.** A number, a name, or a concrete example is worth more
   than any superlative.
2. **Short words beat long words.** "Use" beats "utilize." "Help" beats "facilitate."
   "Show" beats "showcase."
3. **Varied rhythm beats even rhythm.** Mix sentence lengths. Mix paragraph lengths.
   Break patterns.
4. **Stating a fact beats announcing its importance.** Let the reader judge significance.
5. **One good sentence beats three that say the same thing.** Delete the restatements.
6. **An honest opinion beats diplomatic neutrality.** Technical readers want your
   recommendation, not a survey of all possible positions.
7. **Silence beats noise.** If a sentence adds no information, remove it. A shorter
   document that says something is better than a longer one that doesn't.
```

Here's the style guide at `refs/humanising-ai-prose.md`. Let me walk you through the key design decisions:

**Structure.** The guide is organised into 10 sections that mirror the editing process — start with vocabulary (the easiest, most mechanical fixes), work through sentence structure and punctuation, then move to deeper content and substance issues, and end with the actual editing workflow.

**What it covers:**

1. **Vocabulary** (§1) — Consolidated banned-word tables from all your sources plus the Wikipedia page, with concrete replacements. Includes inflated adjectives, abstract metaphor nouns, and hedging phrases.
2. **Banned phrases** (§2) — Patterns that need structural rewriting, not word swaps: faux-insider openers, staccato rhetoric, "not just X, but Y," opening clichés, hollow conclusions.
3. **Sentence/paragraph structure** (§3) — Tricolon overload, the rigid 1-2-3 paragraph formula, robotic transitions, elegant variation (thesaurus syndrome), and copula avoidance (AI's allergy to "is").
4. **Punctuation** (§4) — Em-dash overload, colon-heavy headings, excessive bolding, curly quotes.
5. **Content and substance** (§5) — The deeper problems: significance inflation, vague claims, superficial analysis with participle phrases, promotional language, "despite challenges" boilerplate.
6. **Structural tells** (§6) — Inline-header lists, unnecessary tables, title case, emoji.
7. **Editing process** (§7) — A five-step workflow: read first, delete before rewriting, check for tell clusters, read aloud, add your opinion.
8. **Quick-reference checklist** (§8) — A checkbox list for reviewing drafts. More than three checks = rewrite the passage.
9. **What not to "fix"** (§9) — Guards against over-correction (good grammar isn't a tell, formality isn't the enemy).
10. **Summary of principles** (§10) — Seven guiding rules, each one sentence.

The Wikipedia page was invaluable for the copula-avoidance pattern (§3.5), significance inflation (§5.1), elegant variation (§3.4), and superficial participle-phrase analysis (§5.3) — patterns that the other sources didn't cover as well.