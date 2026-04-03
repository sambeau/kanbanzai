# A guide to modern technical writing

## 1. Voice and tone

### Write like you speak

Good technical writing sounds like a knowledgeable colleague explaining something clearly. Read your text aloud. If it sounds stiff, rewrite it.

- Use everyday words. Choose *use* over *utilise*, *remove* over *eliminate*, *tell* over *inform*.
- Use contractions: *it's, you'll, you're, we're, don't, can't, let's*. They signal a human voice rather than a corporate one.
- Don't mix a contraction and its spelled-out form in the same piece. Pick one and stick with it.
- Never contract a noun and a verb (*The system's running* → *The system is running*).
- Avoid awkward contractions: *there'd, it'll, they'd*.

### Address the reader directly

Use second person (*you, your*) as the default. It keeps the focus on the reader and naturally produces active voice.

- Drop *you can* when the sentence works without it. *You can store files online* → *Store files online*.
- Avoid *we* as a corporate voice. If *we recommend* is unavoidable, use it sparingly, and only when it avoids an awkward passive construction.
- Use first person (*I, my*) only in UI elements where the user is asserting control: *Remember my password*, *I agree to the terms of service*.

### Be friendly, not flippant

Project warmth without undermining credibility. Technical content can be human and still be precise. Don't overdo exclamation points. Save them for genuinely exciting moments, if any.

---

## 2. Sentences

### One idea per sentence

Each sentence should communicate a single idea, or at most two closely related ideas. If you find yourself joining clauses with *and*, *but*, and *so* more than once, split the sentence.

### Lead with what matters

Put the most important information at the start of the sentence. Readers scan the beginnings of sentences in an F-shaped pattern—the opening words carry the most weight.

Before:

> The Recommended Charts command on the Insert tab recommends charts that are likely to represent your data well.

After:

> Create the right chart for your data with Recommended Charts on the Insert tab.

### Keep subject and verb together

Separating the subject from its verb with long modifiers forces the reader to hold too much in working memory.

Weak:

> The file, which was created during yesterday's build and subsequently modified by the CI pipeline, failed validation.

Better:

> The file failed validation. It was created during yesterday's build and then modified by the CI pipeline.

### Watch your sentence length

Sentences over 25–30 words deserve a second look. Long sentences aren't always bad, but every long sentence should earn its length. If a sentence has more than two embedded clauses, split it.

---

## 3. Voice and verbs

### Prefer active voice

In active voice, the subject performs the action. Active voice is direct, clear, and usually shorter.

| Active | Passive |
|--------|---------|
| The installer copies the files. | The files are copied by the installer. |
| Divide your document into sections. | Your document can be divided into sections. |

### Use passive voice deliberately

Passive voice is the right choice when:

- You want to avoid blaming the reader, especially in error messages: *That site can't be found* rather than *You entered an invalid URL*.
- The receiver of the action matters more than the actor: *The transaction is committed when the user selects OK*.
- The actor is unknown or irrelevant.

If you can't name a good reason for using passive voice, switch to active.

### Unbury your verbs

Smothered verbs (nominalisations) make sentences sluggish. Turn the noun back into a verb.

| Smothered | Direct |
|-----------|--------|
| perform an installation | install |
| make a determination | determine |
| carry out an evaluation of | evaluate |
| establish connectivity | connect |

Never combine a smothered verb with passive voice. *An installation was performed by the team* → *The team installed the software*.

### Choose the right tense

- **Present tense** for how things work right now, established facts, and current system behaviour. This is your default tense.
- **Past tense** for describing what happened: steps already taken, results of a test, changes made in a previous release.
- **Imperative mood** for instructions and procedures: *Enter a file name, and then save the file.*

Don't switch tenses within a paragraph unless the context genuinely shifts (for example, moving from describing current behaviour to what changed in a previous version).

---

## 4. Word choice

### Prefer short, familiar words

| Use | Not |
|-----|-----|
| use | utilise, make use of |
| start | commence, initiate |
| end | terminate, cease |
| to | in order to, as a means to |
| also | in addition |
| because | due to the fact that |
| about | approximately, with regard to |
| if | in the event that |
| can | is able to |

### One word, one meaning

Use each term consistently to represent one concept. If you call it a *dashboard* in one section, don't switch to *control panel* in the next. Conversely, don't use the same word for two different things.

### Don't repurpose common words

Use words in their most familiar sense. Don't coin new terms from existing words (*bucketise*), and don't give ordinary words new meanings (*graveyard* to mean *archive*). Don't use verbs as nouns or nouns as verbs: *affect performance*, not *impact performance*; *download the paper*, not *get the download*.

### Cut the filler

- Remove unnecessary adverbs: *very, quite, really, easily, simply, basically, just*.
- Replace vague quantifiers with specifics: *several* → *three*, *some* → *15%*, *many* → *most*.
- Strip adjectives that don't add meaning. If every feature is *powerful* and every integration is *seamless*, neither word communicates anything.

### Use technical terms with care

- Don't use a technical term when a common word will do. Use *copy* instead of *rip*.
- When a technical term is the clearest option, define it in context on first use.
- Use one term consistently for one concept. Don't alternate between synonyms.
- Know your audience. Industry-specific terminology is fine for professional readers, but verify standard usage against authoritative sources.

### Avoid jargon

Jargon serves as shorthand among specialists, but it excludes everyone else. If a more familiar term exists, prefer it. Business and marketing jargon is never acceptable: *leverage* (meaning *use*), *synergise*, *paradigm shift*, *circle back*.

A quick test: if a reviewer questions a term, it's probably jargon. If you can't find it in a standard dictionary, spell it out or replace it.

---

## 5. Abbreviations and acronyms

- Keep abbreviations to a minimum. Each one adds cognitive load.
- Define at first use: *The command-line interface (CLI) supports the following flags*. After that, use the abbreviation alone.
- Don't use abbreviations in headings unless they're universally understood (API, URL, HTTP, HTML).
- Don't introduce an abbreviation you'll only use once. Just spell it out.
- Latin abbreviations (*e.g., i.e., etc.*) are fine in parenthetical text. In running prose, prefer *for example*, *that is*, and *and so on*.

---

## 6. Punctuation

### The serial comma

Always use the serial (Oxford) comma before the conjunction in a list of three or more items.

*Android, iOS, and Windows*—not *Android, iOS and Windows*.

### Commas

- After an introductory phrase: *With the new CLI, you can deploy in seconds.*
- To join independent clauses with a conjunction: *Select Options, and then select Enable.*
- Between two or more adjectives that modify the same noun (if you could reverse them or join them with *and*): *a fast, reliable connection.*
- Don't use a comma to join independent clauses without a conjunction. Use a semicolon or split into two sentences.
- Don't use a comma between verbs in a compound predicate: *The program evaluates your system and copies the files.*

### Periods

- One space after a period. Never two.
- Don't use periods in headings, subheadings, or UI labels.
- In lists: use a period after each item if any item is a complete sentence. Skip periods if all items are short phrases (three words or fewer).

### Semicolons

Semicolons signal complexity. Before using one, try to simplify the sentence by splitting it or converting it to a list.

Use semicolons:
- Between two independent clauses not joined by a conjunction: *Select Options; then select Automatic Backups.*
- To separate items in a series that already contains commas.

### Colons

- Don't end a heading or title with a colon.
- You can use a colon within a heading to separate a title and subtitle: *Get started with Azure IoT: An interactive guide*.
- Lowercase the word after a colon unless it's a proper noun.
- When introducing a list, use a colon if the introductory text directly references the items ("the following", "these", a stated number). Otherwise, use a period.

### Dashes and hyphens

These are not interchangeable.

- **Em dash (—)**: sets off a break in thought or a parenthetical remark. No spaces around it. Don't overuse—one per sentence at most.
- **En dash (–)**: used in ranges of numbers and dates: *2020–2024*, *pages 15–32*. No spaces (except in time stamps in UI).
- **Hyphen (-)**: joins compound modifiers before a noun: *Azure-supported features*, *real-time updates*.

### Apostrophes

- Use for possessives: *the server's configuration*, *users' passwords*.
- Use for contractions: *don't, it's, you're*.
- Don't use for possessive *its*: *The system restarted its services.*
- Don't use to form plurals: *APIs*, not *API's*.

For a complete reference, see [punctuation-guide.md](punctuation-guide.md).

---

## 7. Capitalisation

### Default to sentence case

Capitalise only the first word of a heading or phrase and any proper nouns. This is the rule for headings, titles, subheadings, UI labels, and list items.

Correct:

> Set up your development environment

Wrong:

> Set Up Your Development Environment

### When to capitalise

- The first word of a sentence or heading.
- Proper nouns: product names, service names, brand names, people's names, place names.
- The first word after a colon in a title.

### When not to capitalise

- Common technology terms: *cloud computing, open source, machine learning*.
- The spelled-out form of an acronym, unless it's a proper noun.
- Don't use ALL CAPS for emphasis. Use *italic* sparingly instead.

### Title case

Reserve title case for situations that require it: book titles, article titles in citations, and product or service names. When using title case, capitalise all words except articles (*a, an, the*), prepositions of four or fewer letters (*on, to, in, of*), and conjunctions (*and, but, or, nor, yet, so*), unless they're the first or last word.

---

## 8. Scannable content

### Structure for scanning

Readers don't read—they scan. Design every page so a reader who never reads a full paragraph can still navigate to what they need.

- Put the most important information above the fold (the first screen).
- Lead with keywords in headings, table entries, and paragraph openings.
- Break long content into short sections with clear headings.
- No walls of text. Three to seven lines per paragraph is the target.
- Single-sentence paragraphs are fine. Use them for emphasis.

### Headings

Headings are the skeleton of your document. If a reader only reads the headings, they should understand the structure and scope.

- Think of headings as an outline. Each heading should introduce its topic in an interesting, specific way.
- Keep headings short. Front-load the most important words.
- Use sentence-case capitalisation.
- Don't end headings with a period.
- Use parallel structure across headings at the same level. If one heading is a verb phrase, all headings at that level should be verb phrases.
- Use at most two or three levels of heading. If you need more, rethink your structure.
- Don't put two headings in a row without text between them.
- Consider task-oriented headings using infinitive phrases for how-to content: *To configure the database*.

### Lists

Lists transform dense text into something a reader can absorb at a glance.

- A list should have at least two items. Aim for no more than seven.
- **Bulleted lists**: for items that share a category but have no required order.
- **Numbered lists**: for sequential steps or ranked items.
- **Definition lists**: for terms with explanations. Bold the term, follow it with a period, then the definition.
- Make all list items parallel in grammatical structure.
- Capitalise the first word of every list item.
- Don't use semicolons, commas, or conjunctions at the end of list items.
- Use periods only when items are complete sentences (or when any single item in the list is a complete sentence).

### Tables

Tables make structured information easy to compare and find.

- Don't use a table when a list would do.
- Include a title or introductory sentence so the purpose is clear.
- Put the identifying information (names, commands, terms) in the leftmost column.
- Make entries parallel in structure.
- Use sentence-case capitalisation in headers and cells.
- Don't leave cells blank—use *Not applicable* or *None*.
- Keep tables responsive: limit columns, keep cell text short.

### White space

White space is a feature, not a bug. Use it deliberately.

- Extra space above headings signals a new topic.
- Short paragraphs with space between them are more inviting than dense blocks.
- Don't use extra blank lines to artificially increase spacing—especially in web content, where responsive layouts handle this automatically.

---

## 9. Procedures and instructions

### Writing step-by-step instructions

- Use a **numbered list** for multi-step procedures.
- Use a **bulleted list** (or a single paragraph) for single-step procedures.
- Each step should be one action. It's fine to combine short actions that happen in the same place.
- Write each step as a complete sentence. Start with a capital letter, end with a period.
- Start most steps with a verb: *Select Settings*, *Enter your password*, *Open the file*.
- If the reader needs to be in a specific location before acting, say where first: *On the Design tab, select Header Row.*
- Include the final action that completes the procedure (selecting OK, Apply, and so on).
- Don't overwhelm readers. If a procedure runs beyond what fits on one screen, consider splitting it.

### Describing interactions with UI

Use generic verbs that work regardless of input method (keyboard, mouse, touch, voice). Avoid input-specific terms like *click* or *swipe*.

| Verb | Use for |
|------|---------|
| **Select** | Buttons, checkboxes, options, list items, links, menu items, keys |
| **Open** | Apps, files, folders, panes |
| **Close** | Apps, dialogs, panes, tabs, notifications |
| **Go to** | Menus, tabs, pages, websites |
| **Enter** | Typing or inserting values in text fields |
| **Turn on / Turn off** | Toggle switches |
| **Clear** | Deselecting a checkbox |
| **Choose** | When the action is based on user preference, or when the word *Select* appears in the UI label |
| **Move / Drag** | Repositioning elements |

Use the angle-bracket shorthand for simple sequential paths: *Select Accounts > Other accounts > Add an account.* Reserve this for paths where every step uses the same interaction method.

### Keep instructions scannable

- Use headings so readers can jump to the procedure they need.
- Use parallel structure in procedural headings: *Create a profile*, *Add an account*, *Delete a record*.
- Don't repeat the heading in the introductory sentence. If the heading says *Create a profile*, the intro shouldn't begin *To create a profile…*

---

## 10. Paragraphs and flow

### Topic sentences

Open each paragraph with a sentence that states the paragraph's main point. A reader who only reads first sentences should still get the gist of the section.

### Logical order

Arrange sentences within a paragraph so that each one follows naturally from the last. Common ordering strategies:

- General → specific
- Cause → effect
- Problem → solution
- Chronological sequence

### Transitions and flow

Use linking words and phrases to connect sentences and show how ideas relate: *however, therefore, for example, in contrast, as a result, also, instead*. But don't overdo them—if the logical connection between two sentences is obvious, a transition word may be unnecessary.

### Parallel structure

When listing, comparing, or contrasting items, keep the grammatical structure consistent.

Inconsistent:

> The tool supports:
> - Scanning for vulnerabilities
> - To generate reports
> - Audit trail management

Consistent:

> The tool supports:
> - Scanning for vulnerabilities
> - Generating reports
> - Managing audit trails

---

## 11. Longer documents

Not every piece of technical writing is a short help article. For specifications, architecture documents, design rationales, and guides, additional structure matters.

### Problem statement

Summarise the problem in one short paragraph. If you can't, you may not understand it well enough yet.

### Introduction structure

For longer technical documents, follow a logical arc:

1. **Context**: what the reader needs to know to understand the problem.
2. **Gap**: what's missing, broken, or unsolved.
3. **Question or goal**: what this document addresses.
4. **Approach**: how the document (or the work it describes) tackles the problem.

### Sections and navigation

- Arrange sections in a logical sequence that matches how the reader will use the document.
- In long documents, include a table of contents with links.
- Consider adding *Back to top* links at the end of major sections.
- Break content into discrete, well-labelled sections rather than one continuous narrative.

### Tables and figures

- Give every table and figure a caption that can stand alone—a reader should understand the table without reading the surrounding text.
- Introduce each table or figure with a complete sentence (not ending in a colon).

### Summaries and abstracts

Write the summary last. It should accurately reflect the final document, not the document you planned to write. Keep it concise: the summary of a ten-page document should be one paragraph, not a full page.

---

## 12. Inclusive and accessible writing

- Use gender-neutral language. Rewrite to use *you* or *they* instead of *he/she*.
- Use *they* as a singular pronoun when referring to a person whose gender is unknown or non-binary.
- Don't use ableist language. Avoid *crippled*, *blind spot*, *sanity check* when a neutral alternative exists (*degraded*, *oversight*, *validation*).
- Write for screen readers: heading hierarchy matters, link text should be descriptive (*Read the configuration guide*, not *Click here*), and table headers should be specific.

---

## 13. Quick-reference checklist

Use this checklist when reviewing any document before publication.

### Sentences

- [ ] Each sentence communicates one idea (or two closely related ideas).
- [ ] The most important information comes first.
- [ ] Subject and verb are close together.
- [ ] No more than two embedded clauses per sentence.
- [ ] Sentences over 25–30 words have been considered for splitting.

### Voice and verbs

- [ ] Active voice is used by default.
- [ ] Passive voice is used deliberately and for a stated reason.
- [ ] Smothered verbs (nominalisations) have been replaced with direct verbs.
- [ ] No combination of smothered verb + passive voice.
- [ ] Instructions use imperative mood.

### Tense

- [ ] Present tense for current behaviour and established facts.
- [ ] Past tense for what happened (previous steps, results, changelogs).
- [ ] Tense changes between sentences reflect genuine shifts in context.

### Word choice

- [ ] Shorter, familiar words are preferred over longer alternatives.
- [ ] Wordy phrases have been replaced with single words.
- [ ] Redundant words and duplicate-meaning pairs have been removed.
- [ ] Vague quantifiers have been replaced with specific numbers where possible.
- [ ] Adjectives and adverbs are present only when they add meaning.
- [ ] Each term is used consistently for one concept.
- [ ] Technical terms are defined in context on first use.
- [ ] Jargon has been replaced with plain language where possible.

### Tone

- [ ] Contractions are used consistently (not mixed with spelled-out equivalents).
- [ ] The text addresses the reader as *you*.
- [ ] The tone is friendly but not flippant.
- [ ] *You can* has been removed where the sentence works without it.

### Paragraphs and flow

- [ ] Each paragraph begins with a topic sentence.
- [ ] Sentences within a paragraph are in logical order.
- [ ] Linking words create flow between sentences without being overused.
- [ ] Lists use parallel grammatical construction.
- [ ] Paragraphs are 3–7 lines. No walls of text.

### Headings

- [ ] Headings use sentence-case capitalisation.
- [ ] Headings are short and front-load important words.
- [ ] Headings at the same level use parallel structure.
- [ ] No period at the end of a heading.

### Lists

- [ ] Each list has at least two items.
- [ ] All items are parallel in structure.
- [ ] Punctuation is consistent (periods only if items are complete sentences).
- [ ] No semicolons or conjunctions at the end of items.

### Punctuation

- [ ] The serial (Oxford) comma is used in lists of three or more.
- [ ] One space after periods.
- [ ] Em dashes have no spaces; en dashes are used for ranges; hyphens join compound words.
- [ ] Semicolons and colons are used correctly, and complex sentences have been simplified where possible.

### Abbreviations and acronyms

- [ ] Kept to a minimum.
- [ ] Defined at first use.
- [ ] Not used in headings (unless universally known).

### Capitalisation

- [ ] Sentence case is the default for headings and labels.
- [ ] Proper nouns are capitalised; common nouns are not.
- [ ] No ALL CAPS for emphasis.

### Procedures (where applicable)

- [ ] Multi-step procedures use numbered lists.
- [ ] Each step is one action, starting with a verb.
- [ ] Location comes before the action in each step.
- [ ] Input-neutral verbs are used (select, enter, open—not click, tap, swipe).

### Document structure (for longer documents)

- [ ] The problem statement fits in one short paragraph.
- [ ] The introduction follows: context → gap → question → approach.
- [ ] Sections are in a logical sequence.
- [ ] White space is used to aid readability.
- [ ] Tables and figures have standalone captions.
- [ ] Any summary or abstract was written last and reflects the final content.