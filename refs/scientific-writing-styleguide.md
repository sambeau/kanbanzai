# Scientific Writing Styleguide

> *"There is only one essential goal in scientific writing: clarity."* — Robert Day
>
> *"If you can't explain it simply, you don't understand it well enough."* — Albert Einstein

This styleguide defines the writing standards for all our documentation. It is grounded in the principles of scientific writing as taught by Oxford University's MPLS Division (original material by Dr John Dixon, Libra Scientific Communications Ltd). The rules here apply to specifications, design documents, research notes, technical reports, and any prose we publish.

---

## Table of Contents

1. [Core Principle](#1-core-principle)
2. [Know Your Audience](#2-know-your-audience)
3. [Constructing Sentences](#3-constructing-sentences)
4. [Active and Passive Voice](#4-active-and-passive-voice)
5. [Using Verbs with Clarity](#5-using-verbs-with-clarity)
6. [Using Tenses Consistently](#6-using-tenses-consistently)
7. [Vocabulary, Adjectives, and Adverbs](#7-vocabulary-adjectives-and-adverbs)
8. [Paragraphs, Flow, and Connecting Methods](#8-paragraphs-flow-and-connecting-methods)
9. [Organisation and Appearance](#9-organisation-and-appearance)
10. [Writing a Document](#10-writing-a-document)
11. [Quick-Reference Checklist](#11-quick-reference-checklist)

---

## 1. Core Principle

Every decision in this styleguide serves one goal: **clarity**. If a rule ever conflicts with clarity, clarity wins.

---

## 2. Know Your Audience

Before you write a single sentence, answer these questions:

- **Who will read this?** Engineers on the team? External stakeholders? Future contributors who have no context?
- **What language and abbreviations are they familiar with?** Do not assume shared knowledge.
- **What is interesting and relevant to them?** Cut everything else.
- **How will they read it?** Most readers skim. They dip into sections rather than reading linearly. Therefore your writing needs structure, signposts, and logical flow so that a reader can find what they need without reading everything.

Give your readers clear, accurate language with straightforward nontechnical vocabulary. Do not distract them with dense text, unfamiliar jargon, or unexplained acronyms.

---

## 3. Constructing Sentences

### Rules

1. **One idea per sentence** — or at most two closely related ideas.
2. **Put the most important information first.** Lead with the subject.
3. **Keep the subject and verb together.** Do not separate them with long parenthetical clauses.
4. **Limit embedded clauses to two.** More than two pieces of extra information in a single sentence makes it hard to parse.
5. **Consider splitting any sentence longer than 30 words** into two sentences.
6. **Place lists at the end of a sentence**, not the beginning.

### Example — Before

> Factors such as retry logic, connection pooling, timeout configuration, and more recently circuit-breaker patterns affect service reliability.

### Example — After

> Service reliability is affected by factors such as retry logic, connection pooling, timeout configuration, and circuit-breaker patterns.

The revision puts the main point ("service reliability") first, keeps subject and verb together, and moves the list to the end where it is easier to scan.

### Splitting Long Sentences

When a sentence accumulates multiple subordinate clauses, split it and lead with the main clause.

**Before (one overloaded sentence):**

> If the deployment pipeline fails to notify the on-call engineer, even though alerts are configured, which they should be, teams will continue to experience delayed incident response because no one knows the build broke.

**After (two clearer sentences):**

> Teams will continue to experience delayed incident response when the deployment pipeline fails to notify the on-call engineer. Alerts should be configured, yet without reliable notification, no one knows the build broke.

---

## 4. Active and Passive Voice

### Active Voice

The subject performs the action. The object follows the verb.

> The team calculated the optimum pH.

Properties of the active voice:

- Usually more concise than the passive.
- Puts the subject (the doer) at the beginning of the sentence — useful when the doer matters.
- Does **not** require a personal pronoun. "Process X improves yield" is active voice without "I" or "we".

### Passive Voice

The object of the action comes first. The subject follows the verb (or is omitted entirely).

> The optimum pH was calculated by the team.

Properties of the passive voice:

- Sounds more formal.
- Can use more words.
- Enables an impersonal tone.
- Useful when the doer is unknown, obvious, or irrelevant: "Artificial intelligence has been the subject of considerable research for decades." (By whom? It doesn't matter.)
- Useful when the object is the main topic: "These dangerous emissions are produced by diesel engines." (The focus is the emissions, not the engines.)

### When to Use Which

| Use active voice when | Use passive voice when |
|-|-|
| You want conciseness | You want to balance an otherwise all-active paragraph |
| The doer matters and should be named | The doer is unknown, obvious, or irrelevant |
| You want to take responsibility for an action | The object or recipient is the main topic |
| Your audience or style guide prefers it | You need a more impersonal or formal tone |

**Best practice: blend both.** The most readable text uses a combination. A passage written entirely in one voice becomes monotonous or awkward.

### Using "We"

It is acceptable — and often preferable — to use "we" in technical documentation:

- **Do:** "We designed the retry mechanism to handle transient failures."
- **Acceptable alternative:** "The retry mechanism was designed to handle transient failures."

Use "we" when it adds clarity or takes responsibility. Default to active voice unless you have a specific reason to choose passive.

---

## 5. Using Verbs with Clarity

### Avoid Smothered Verbs

A "smothered verb" is a verb that has been converted into a noun, usually with a suffix like *-tion*, *-ance*, *-ment*, or *-ent*. The verb form is almost always shorter and clearer.

| Smothered (avoid) | Direct verb (prefer) |
|-|-|
| come to a decision | decide |
| provide assistance | assist |
| make an assessment | assess |
| perform an analysis | analyse |
| carry out an investigation | investigate |
| give consideration to | consider |
| make a determination | determine |

### The Danger: Smothered Verbs Combined with Passive Voice

On their own, neither smothered verbs nor the passive voice cause much harm. Combined, they produce bloated, hard-to-read prose.

| Combined (avoid) | Direct (prefer) |
|-|-|
| A calculation of the optimum pH was made by the team. (11 words) | The team calculated the optimum pH. (6 words) |
| An investigation into the failure was performed by the SRE team. (11 words) | The SRE team investigated the failure. (6 words) |

**Rule: If you spot a noun ending in *-tion* or *-ment* near "was made" or "was performed", rewrite the sentence with the verb form in active voice.**

---

## 6. Using Tenses Consistently

Different situations call for different tenses. The key is to be **consistent** for each situation type.

### When to Use Present Tense

- **Established facts:** "TLS encrypts data in transit."
- **Describing what the system does now:** "The service handles up to 10,000 requests per second."
- **Stating conclusions or beliefs resulting from the work:** "We conclude that approach B is more effective."

### When to Use Past Tense

- **Describing what you did (methods):** "We configured the load balancer with round-robin routing."
- **Describing what you found (results):** "Latency decreased by 40% after the migration."
- **Attributing previous work that is not yet established fact:** "Smith (2023) suggested that the cache layer introduced the bottleneck."

### Worked Example

> The system processes requests through a message queue *(present — established fact)*. In Q3, Kumar **proposed** that batching would reduce latency *(past — attribution to previous work)*. We **implemented** a batching strategy with a 50 ms window *(past — what we did)*. We **observed** a 35% reduction in p99 latency *(past — what we found)*. We conclude that batching is an effective optimisation for this workload *(present — conclusion)*.

**Rule: When tenses change between adjacent sentences, make sure the change reflects a genuine shift in situation (fact vs. method vs. result), not carelessness.**

---

## 7. Vocabulary, Adjectives, and Adverbs

### Prefer Shorter, Familiar Words

Do not use a long or obscure word when a short, common one will do.

| Avoid | Prefer |
|-|-|
| advantageous | better |
| indeterminate | unknown |
| constituent | part |
| utilise | use |
| facilitate | help, enable |
| commence | start, begin |
| terminate | end, stop |
| subsequently | then, later |
| in the event that | if |
| prior to | before |

### Replace Phrases with Single Words

| Phrase (avoid) | Word (prefer) |
|-|-|
| serves the function of being | is |
| in view of the fact that | because, since |
| it is possible that | perhaps, possibly |
| in order to | to |
| at the present time | now |
| due to the fact that | because |
| a sufficient number of | enough |
| has the ability to | can |

### Remove Redundant Words

These common pairings add nothing:

- ~~careful~~ consideration → consideration
- ~~definitely~~ proved → proved
- ~~entirely~~ eliminate → eliminate
- ~~completely~~ finished → finished
- ~~advance~~ planning → planning
- ~~basic~~ fundamentals → fundamentals

### Remove Duplicate-Meaning Pairs

- ~~each and every~~ → each (or every)
- ~~various different~~ → various (or different)
- ~~period of time~~ → period (or time)
- ~~future plans~~ → plans
- ~~end result~~ → result

### Use Imprecise Words with Care

Words like "several", "some", "many", "few" are inherently vague. Ask yourself: can I replace this with a number or a range? If so, do it.

- **Vague:** "The service experienced several outages."
- **Precise:** "The service experienced three outages in the past month."

Similarly, verbs like "affect", "change", and "impact" are imprecise. State *how* something was affected or *what* changed.

### Adjectives

Use strong adjectives when justified: "urgent", "dangerous", "essential". Leave out weak or hollow adjectives that add no information: "particular", "apparent", "notable", "interesting".

### Adverbs

Include an adverb when it is essential to meaning:

> Up to 85% of users **mistakenly** believe the feature is deprecated. ("Mistakenly" is essential — without it, the meaning reverses.)

Remove an adverb when it weakens the word it modifies or adds nothing:

> ~~really~~ important → important
>
> ~~very~~ dangerous → dangerous
>
> ~~quite~~ significant → significant

**Rule: If you can delete an adjective or adverb without changing the meaning, delete it.**

---

## 8. Paragraphs, Flow, and Connecting Methods

### Paragraph Construction

A paragraph is not just a visual break when text gets too long. A good paragraph has:

1. **A topic sentence** — the first sentence introduces what the paragraph is about.
2. **A group of related ideas** — each expressed in its own sentence.
3. **A logical order** — most important to least important, chronological, or cause to effect.
4. **Flow** — sentences connect to one another (see techniques below).
5. **A wrapping-up sentence** — the final sentence concludes the paragraph's point.

### Techniques for Creating Flow

**1. Add linking words and phrases** at the beginning of some (not all) sentences:

| Purpose | Examples |
|-|-|
| Sequence | Also, Moreover, First, Second, In addition, Next, Finally |
| Restatement | That is, To put it another way, In other words |
| Example | For example, For instance, such as, like |
| Reason | Because, Since |
| Consequence | Therefore, So, Thus, Hence, Consequently, Accordingly |
| Contrast | Nevertheless, However, In contrast, Conversely, Alternatively |
| Concession | Although, In spite of, Despite, While |
| Similarity | Similarly, In a similar way, Likewise |
| Addition | Further, Furthermore, In addition, Also |
| Conclusion | In conclusion, To conclude |
| Summary | In brief, To summarise, In summary |

Other useful openers:

| Type | Examples |
|-|-|
| Verb infinitive | To progress, To determine, To improve |
| Adverbial phrase | In recent years, With confidence, Without exception |
| Adverb | Recently, Immediately, Controversially |

**2. Join short, related sentences** — but don't let the combined sentence exceed ~30 words.

**3. Use parallel construction in lists.** When presenting a series, keep each element in the same grammatical form.

- **Good (parallel verbs):** "We need to appraise the current design, identify the key gaps, and determine what changes are required."
- **Bad (mixed forms):** "We need to appraise the current design, identification of key gaps, and then there should be a determination of changes."

### Example — Before (poor flow)

> The impact of caching on response time is debated. Cache hit rates in production have steadily improved. Controversy arises around cache invalidation strategies. The arguments often lack empirical backing. We remain influenced by anecdotal evidence. The confusion continues. We need to benchmark current behaviour. We need to identify the bottlenecks. We need to determine which invalidation strategy to adopt.

### Example — After (good flow)

> The impact of caching on response time is debated. In recent years, cache hit rates in production have steadily improved. Controversy always arises around cache invalidation strategies. However, the arguments often lack empirical backing, and we remain influenced by anecdotal evidence. Consequently, the confusion continues. To progress, we need to benchmark current behaviour, identify the bottlenecks, and determine which invalidation strategy to adopt.

Changes made: linking words added, short related sentences joined, and the final three sentences combined using parallel construction.

---

## 9. Organisation and Appearance

### Logical Structure

- Organise sections in a logical sequence: broad context first, then specifics, then conclusions.
- Use headings and subheadings liberally. They act as signposts for skimming readers.
- Use numbered or bulleted lists for sequences, options, or sets of related points.

### White Space

White space is not wasted space — it is a readability tool.

- Separate sections with blank lines.
- Keep paragraphs short (3–6 sentences is a good target).
- Do not produce "walls of text".
- Use tables to present structured comparisons rather than burying them in prose.

### Visual Aids

- Use diagrams, tables, and code examples where they communicate more efficiently than prose.
- Every figure or table should have a caption that lets the reader understand it without reading the surrounding text.

---

## 10. Writing a Document

This section provides guidance on structuring longer documents such as design documents, research reports, RFCs, and technical specifications. It adapts the manuscript conventions of scientific writing to our documentation needs.

### 10.1. Planning

Do not start by staring at a blank page. Instead:

1. **Start with the easiest section.** Often this is the methods/approach section — you are simply describing what you did or will do.
2. **Prepare your key figures and tables first.** These usually illustrate your main messages. Write a sentence or two summarising each one. Together, these form the basis of your conclusion and summary.
3. **Write a problem statement** before tackling the introduction or discussion. This is a short paragraph covering:
   - The **general problem area**.
   - The **specific problem** you are addressing.
   - The **extent of your contribution** — what you have done or propose to do about it.

   If you cannot write this summary, you are not yet clear enough about what you want to tell your audience.

4. **Use an outline or mind map.** Enter the main sections of your document, add ideas as sentences (not just keywords — sentences force you to include verbs and therefore meaning), and reorganise as the structure emerges.

### 10.2. Introduction

When readers finish your introduction, they should understand **why** the work was done and **why it matters**. They should be clear about the problem and want to read on.

**Guidelines:**

- **Include only what the reader needs to know.** Not a history lesson, not a literature review. Elevate the reader's knowledge from a reasonable starting point.
- **Do not start with something everyone knows** or information that is not relevant.
- **Say why the work is important and original** — modestly.
- **Deliver a clear and logical rationale.** Follow this sequence:
  1. **Context** — the broad problem area and what is established knowledge.
  2. **Gap** — what is not known, or what the specific problem is.
  3. **Question** — the research question, hypothesis, or objective.
  4. **Approach** — a brief summary of how you will address it.
- **Be mindful of busy readers.** Many introductions are too long. Ideally, the final paragraph summarises the question, the overall approach, and why the work matters. Many readers will skip straight to this final paragraph.

### 10.3. Methods / Approach

The methods section is a recipe. Its purpose is to allow someone else to reproduce or verify your work.

**Guidelines:**

- If you used a well-documented method, cite the original source. If you modified or created a new approach, describe it in full.
- Be as detailed as necessary, but as concise and readable as possible.
- **The acid test:** could a colleague repeat your approach after reading your description?
- Use subheadings — often these can mirror the corresponding results section.
- Use tables, flow charts, or diagrams where they help.

**Readers should be clear about:**

- Selection and source of materials, data, or participants.
- Design specifics — parameters, configurations, constraints.
- Outcome measures — what you measured and how.
- Statistical or analytical techniques, if applicable.
- Any ethical considerations or approvals.

### 10.4. Results / Findings

The results section presents what you found. It should allow the reader to answer the research question without having to refer back to the methods.

**Do:**

- Present the most important and relevant results in a logical sequence.
- Provide clear figures and tables, each with a helpful description (legend) summarising the relevant method.
- Avoid wordy repetition of information already presented visually.

**Do not:**

- Include every result — be selective, but include all results that are relevant.
- Repeat in body text data that are better shown in figures and tables.
- Draw conclusions (save those for the discussion).
- Relate findings to other work (save that for the discussion).

**Language guidance:**

- Reserve the word **"significant"** for statistical findings. If something is important, say "important" or "considerable" — and only in the discussion.
- Avoid subjective adverbs of magnitude like "markedly increased" or "greatly reduced". Instead, quantify: "three-fold increase" or "95% reduction".

### 10.5. Discussion / Analysis

The discussion places your findings in context: relating them to current knowledge, justifying the contribution, and using the appropriate language of reasoned argument.

**A good discussion includes:**

1. A summary of the main findings — best placed in the first paragraph.
2. Strengths and weaknesses of the study or approach.
3. Strengths and weaknesses in relation to other work or existing theory.
4. A balanced conclusion.
5. Unanswered questions and future direction.

**Common problems to avoid:**

- Failing to provide a balanced implication of the results.
- Beginning with a second introduction.
- Repeating all the results.
- Presenting an unstructured argument.
- Including irrelevant material.
- Overinterpreting results — no "marketing spin".

Keep the discussion focused and relevant. Readers will thank you for not going on for page after page.

### 10.6. Abstract / Summary

Write the abstract last. It should be an accurate reflection of the document and must not contain information absent from the body.

**An abstract answers four questions** (after Maeve O'Connor):

1. **Why did you start?** — the motivation.
2. **What did you do?** — the approach.
3. **What did you find?** — the key results.
4. **What do your findings mean?** — the conclusion.

**Proportions (as a rough guide):**

| Section | Share of abstract |
|-|-|
| Background | Keep it short |
| Methods | Minimum necessary |
| Results | The bulk — up to 50% |
| Conclusion | Essential — always include |

**Writing tips for abstracts:**

- Use short sentences.
- Use simple, specific words.
- Edit out waste words ruthlessly.
- Be consistent with mixed tenses (present and past will sit close together).
- Use the active voice where appropriate.
- Do not be afraid to use "we".
- Explain any abbreviations.

**Common errors to avoid:**

- Background too long — wastes valuable words.
- Question omitted or vague — the reader must guess.
- Answer not stated — the reader must guess.
- A result summarised without supporting data ("X was better than Y" with no numbers).
- Too many results — include only the main ones.
- "Conclusion creep" — the abstract conclusion says something different from or additional to the document's actual conclusion.

---

## 11. Quick-Reference Checklist

Use this checklist when reviewing any document before publication.

### Sentences
- [ ] Each sentence communicates one idea (or two closely related ideas).
- [ ] The most important information comes first.
- [ ] Subject and verb are kept together.
- [ ] No more than two embedded clauses per sentence.
- [ ] Sentences over 30 words have been considered for splitting.

### Voice and Verbs
- [ ] Active voice is used where appropriate.
- [ ] Passive voice is used deliberately (not by default).
- [ ] Smothered verbs have been replaced with direct verb forms.
- [ ] No combination of smothered verb + passive voice.

### Tenses
- [ ] Present tense for established facts, current system behaviour, and conclusions.
- [ ] Past tense for methods (what was done) and results (what was found).
- [ ] Tense changes between sentences reflect genuine shifts in context.

### Vocabulary
- [ ] Shorter, familiar words are preferred over longer alternatives.
- [ ] Wordy phrases have been replaced with single words.
- [ ] Redundant words and duplicate-meaning pairs have been removed.
- [ ] Vague quantifiers ("several", "some", "many") have been replaced with numbers where possible.
- [ ] Adjectives and adverbs are present only when they add meaning.

### Paragraphs and Flow
- [ ] Each paragraph begins with a topic sentence.
- [ ] Sentences within a paragraph are in logical order.
- [ ] Linking words and phrases create flow between sentences.
- [ ] Lists within sentences use parallel grammatical construction.
- [ ] Each paragraph ends with a wrapping-up sentence (where appropriate).

### Abbreviations and Acronyms
- [ ] Kept to a minimum.
- [ ] Defined at first use.
- [ ] Not used in headings (unless universally known, e.g. API, URL).

### Organisation and Appearance
- [ ] Sections are in a logical sequence.
- [ ] Headings and subheadings provide clear signposts.
- [ ] White space is used to aid readability.
- [ ] Tables and figures have captions that stand alone.
- [ ] No "walls of text".

### Document Structure (for longer documents)
- [ ] A problem statement can be written in one short paragraph.
- [ ] The introduction follows: context → gap → question → approach.
- [ ] Methods are detailed enough for someone else to reproduce.
- [ ] Results present only the relevant findings, without interpretation.
- [ ] Discussion is structured, balanced, and avoids overinterpretation.
- [ ] Abstract/summary was written last and accurately reflects the document.

---

*This styleguide is adapted from the Oxford University MPLS Division's guide to Scientific Writing (original material by Dr John Dixon, Libra Scientific Communications Ltd). It has been reformulated as an in-house reference for our documentation standards.*