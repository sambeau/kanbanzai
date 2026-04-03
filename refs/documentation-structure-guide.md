# Documentation Structure and Planning Guide

> *Structure is what turns information into documentation.*

This guide defines how we **plan**, **organise**, and **structure** documentation. It does not cover prose style, grammar, or punctuation — for those, see [technical-writing-guide.md](technical-writing-guide.md) and [scientific-writing-styleguide.md](scientific-writing-styleguide.md).

**Style and usage.** Follow Merriam-Webster's Collegiate Dictionary for spelling and The Chicago Manual of Style for style and usage. Where these conflict, prefer The Chicago Manual for style questions and Merriam-Webster for spelling. Exceptions are noted throughout our guides.

---

## Table of Contents

1. [The Inverted Pyramid](#1-the-inverted-pyramid)
2. [Know Your Audience](#2-know-your-audience)
3. [Planning a Document](#3-planning-a-document)
4. [Document-Level Structure](#4-document-level-structure)
5. [Section-Level Structure](#5-section-level-structure)
6. [Structuring for Scannability](#6-structuring-for-scannability)
7. [Document Types](#7-document-types)
8. [Cross-Document Architecture](#8-cross-document-architecture)
9. [Tables, Figures, and Examples](#9-tables-figures-and-examples)
10. [Checklist](#10-checklist)

---

## 1. The Inverted Pyramid

The inverted pyramid is the foundation of how we structure every document and every section within it. The most important information comes first. Supporting detail follows in descending order of importance. The reader who stops reading at any point has already absorbed the most valuable content.

This principle operates at three levels simultaneously:

### Content

Broad concepts before specific details. A section opens with its key point, then elaborates. A document opens with what it is and why it matters, then goes deeper.

### Tone

More conversational and accessible at the top of a document or section. More precise and formal as detail increases. A README introduction can be chatty; a configuration reference table should be exact and unambiguous.

### Technical depth

Concepts are introduced without jargon at the top. Technical specifics — commands, configuration, schema, code — appear deeper in the document, where the reader has actively chosen to go looking for them.

**Why this works.** Most readers scan. They dip into sections rather than reading linearly. The inverted pyramid means that scanning always yields the most important information first. It also means that each audience finds what they need at the depth where they naturally stop reading:

- A **designer or manager** reads the opening of each section and gets the concepts, the reasoning, the "what and why".
- A **developer** reads deeper into the same sections and gets the specifics, the commands, the configuration, the "how exactly".

One document serves both audiences without separate versions.

---

## 2. Know Your Audience

Before you write, answer these questions:

- **Who will read this?** What is their role? What are they trying to accomplish?
- **What do they already know?** Do not explain what they already understand. Do not assume knowledge they lack.
- **How will they read it?** Linearly (a getting-started guide) or by dipping in (a reference)?
- **What do they need to walk away with?** Focus on that. Cut everything else.

### Our two primary audiences

**Technical designers, managers, and hobbyists.** They understand technology conceptually. They make architectural and product decisions. They use tools but don't necessarily build them. They want to understand what something does, why it exists, and when to use it. They read openings and summaries. They rarely read deep technical detail.

**Skilled developers.** They build software. They want to know how to use something, how it works internally, and how to extend it. They scan for code examples, commands, and configuration. They read deep into sections when they need specifics.

### Shared assumptions

All our documentation assumes the reader:

- Can use Git, create a repository, and work at the command line.
- Is comfortable reading structured technical text.
- Is a designer-developer — someone who both designs and builds, or who works closely with people who do.

Individual documents should state any assumptions beyond these.

### Tailoring within a single document

You do not need separate documents for each audience. The inverted pyramid handles this naturally:

- **Top of a section:** accessible language, concepts, the "what and why" — serves designers, managers, and developers alike.
- **Deeper in a section:** precise technical detail, commands, configuration, code — serves developers who need specifics.

When a section is exclusively technical (e.g. a CLI reference), say so upfront: *"This section covers the command-line options for `kanbanzai serve`."* The designer skips it. The developer dives in. Neither wastes time.

---

## 3. Planning a Document

Do not start with a blank page and write from the top. Plan first.

### Step 1: Define the document's job

Write a single sentence that captures what the document must accomplish. If you cannot write this sentence, you are not ready to write the document.

> *"This document teaches a new user how to install, configure, and run the tool for the first time."*

> *"This document is a reference for every CLI command, its options, and its output."*

### Step 2: Know what kind of document you are writing

Different documents have different structures. Decide which type this is before you outline it. Section 7 covers the common types and their structures.

### Step 3: Identify your key messages

List the three to five things the reader must understand after reading this document. These become the spine of your structure. Everything in the document should support one of these messages. Everything that doesn't, cut.

### Step 4: Outline with sentences, not keywords

Write your outline as sentences rather than topic labels. Sentences force you to include verbs and therefore meaning. Compare:

- ❌ *"Configuration"*
- ✅ *"The user creates a configuration file that controls which features are enabled."*

The sentence outline tells you what each section actually says. It also reveals gaps — places where you don't yet know what you want to tell the reader.

### Step 5: Prepare examples and figures first

If the document will include code examples, diagrams, tables, or transcripts, draft these before writing the prose. They are often the most valuable part of the document. Prose exists to connect and explain them.

### Step 6: Write the summary last

Whether it's an abstract, a TL;DR, or an introductory paragraph, write it after the body is finished. Summarise what you actually wrote, not what you planned to write.

### Source of truth

Use the implementation as the single source of truth for facts — what a system does, how it behaves, what its commands and options are. Refer to design documents for concepts and intentions, but verify facts against the code. Documentation that contradicts the implementation is wrong.

---

## 4. Document-Level Structure

Every document follows the inverted pyramid at the top level. The opening answers the most important questions; subsequent sections provide increasing detail.

### Opening

The first thing a reader sees must answer: *What is this document? Why should I care?* For most of our documents, this means:

1. **What this is** — one sentence identifying the document and its subject.
2. **Who it's for** — who should read it (and who can skip it).
3. **What you'll learn or find here** — the payoff for reading.

This is not a lengthy introduction. Three to five sentences is often enough.

### Navigation

For documents longer than a few screens:

- Include a table of contents with links.
- Arrange sections in the order the reader needs them, not the order you wrote them.
- Consider the reader's journey: context before action, concepts before specifics, common cases before edge cases.

### Introduction structure for longer documents

When a longer document needs a proper introduction (design documents, guides, specifications), follow this arc:

1. **Context** — what the reader needs to know to understand the problem.
2. **Gap** — what's missing, broken, or unsolved.
3. **Goal** — what this document addresses.
4. **Approach** — how the document (or the work it describes) tackles the problem.

Keep it tight. Many readers skip straight to the final paragraph of an introduction. Make sure that paragraph summarises the goal and approach clearly enough to stand alone.

### Closing

End with one of:

- **A summary** for longer documents — one short paragraph restating the key points.
- **Next steps** for guides and tutorials — what the reader should do now.
- **Related documents** when there is a clear follow-on.

Do not introduce new information in a closing section.

---

## 5. Section-Level Structure

Every section is a miniature inverted pyramid.

### Open with the point

The first sentence of a section states what the section is about and why it matters. A reader who only reads opening sentences should understand the document's structure and key messages.

### Follow the pyramid

After the opening, provide detail in descending order of importance:

1. The concept or key point.
2. The most common or important details.
3. Edge cases, exceptions, advanced detail.

### One idea per section

Each section should cover one topic. If a section covers two topics, split it. If you can't write a heading that accurately describes the section's content, the section probably lacks focus.

### Paragraphs

A paragraph is not a visual break when text gets too long. A good paragraph has:

1. **A topic sentence** — the first sentence introduces what the paragraph is about.
2. **Related ideas** — each in its own sentence, arranged in logical order (general → specific, cause → effect, problem → solution).
3. **Flow** — sentences connect to one another through linking words and logical progression.

Keep paragraphs short: three to seven lines. Single-sentence paragraphs are fine for emphasis. No walls of text.

### Transitions between sections

Each section should follow naturally from the last. If the connection isn't obvious, add a brief transitional sentence at the start of the new section. But don't force transitions where a clean heading break is clearer.

---

## 6. Structuring for Scannability

Readers don't read — they scan. Design every page so a reader who never reads a full paragraph can still navigate to what they need.

### Headings

Headings are the skeleton of your document. If a reader only reads the headings, they should understand the structure and scope.

- Think of headings as an outline. Each heading introduces its topic in a specific way.
- Keep headings short. Front-load the most important words.
- Use sentence-case capitalisation. Don't end headings with a period.
- Use parallel structure across headings at the same level. If one heading is a verb phrase, all headings at that level should be.
- Use at most two or three levels of heading. If you need more, rethink your structure.
- Don't put two headings in a row without text between them.
- For how-to content, consider task-oriented headings: *"Create a configuration file"*, *"Run the test suite"*.

### Lists

Lists transform dense text into something a reader absorbs at a glance.

- **Bulleted lists** for items that share a category but have no required order.
- **Numbered lists** for sequential steps or ranked items.
- A list should have at least two items. Aim for no more than seven.
- Make all list items parallel in grammatical structure.
- Capitalise the first word of every list item.
- Use periods only when items are complete sentences.

### White space

White space is a readability tool, not wasted space.

- Extra space above headings signals a new topic.
- Short paragraphs with space between them are more inviting than dense blocks.
- Separate sections with blank lines.

### Front-loading

Lead with keywords in headings, table entries, list items, and paragraph openings. The scanner's eye hits the first few words of each line. Make those words count.

---

## 7. Document Types

Different documents serve different purposes and need different structures. Here are the types we create most often, with the structure each one should follow.

### README

**Purpose:** First contact. Tells the reader what this is, why they'd use it, and how to get started.

**Structure:**

1. **Title and one-line description** — what this project or component is.
2. **Why this exists** — the problem it solves, in two or three sentences.
3. **Quick start** — the fastest path to a working result. Install, configure, run.
4. **Key concepts** — brief definitions of terms or ideas the reader needs. Keep it short; link to the manual for detail.
5. **Usage examples** — two or three concrete examples showing common tasks.
6. **Configuration** — how to customise behaviour. Only the most important options; link to a full reference if one exists.
7. **Links** — to the manual, reference, contributing guide, and other related documents.

**Audience notes:** A README serves both audiences equally. The quick start and examples are what developers want. The "why this exists" and "key concepts" sections serve designers and evaluators. The inverted pyramid means a manager can read the first three sections and understand what the tool does; a developer can read deeper for configuration and usage.

### Getting-Started Guide

**Purpose:** Walk a new user through first-time setup to a working result. This is a tutorial — a guided journey with a specific outcome.

**Structure:**

1. **What you'll build or achieve** — state the goal upfront so the reader knows what success looks like.
2. **Prerequisites** — what the reader needs before starting (tools, accounts, knowledge). Be specific.
3. **Steps** — numbered, sequential, one action per step. Each step starts with a verb.
4. **Verification** — how to confirm each major step worked. Don't make the reader proceed on faith.
5. **Next steps** — where to go from here. Link to the manual, a more advanced guide, or a reference.

**Audience notes:** Getting-started guides are for both audiences but lean toward the designer/hobbyist end. Keep the language accessible. Explain *why* a step is needed, not just *what* to do. Developers appreciate this too — it builds a mental model.

### Manual

**Purpose:** Comprehensive documentation of a product or system. The reader consults it when they need to understand something or accomplish a task.

**Structure:**

1. **Introduction** — what the product does, who the manual is for, how to use it.
2. **Concepts** — the mental model. Key ideas, architecture, terminology. Lighter on detail, heavier on understanding.
3. **Task-oriented sections** — grouped by what the user is trying to do, not by how the system is organised internally. Each section follows: goal → steps → result.
4. **Advanced topics** — configuration, customisation, extension, troubleshooting.
5. **Reference** — exhaustive detail (commands, options, configuration schema). This can be a separate document if it's large.

**Audience notes:** The inverted pyramid is critical here. Concepts sections are where designers and managers get value. Task-oriented sections serve everyone. Advanced topics and reference sections are primarily for developers. A well-structured manual lets each reader stop at the depth they need.

### Reference

**Purpose:** Exhaustive, look-up-optimised documentation. The reader knows what they're looking for and wants to find it fast.

**Structure:**

- Organise by the thing being referenced (commands, functions, configuration keys, API endpoints), not by workflow.
- Alphabetical or logical grouping — whichever the reader would expect.
- Each entry follows a consistent format: name, description, syntax/signature, parameters/options, examples, notes.
- Include a searchable index or table of contents.

**Audience notes:** References are primarily for developers. Keep prose minimal. Accuracy and completeness matter more than readability. Every entry should have at least one example.

### Design or Architecture Document

**Purpose:** Explain what a system does, why it was designed this way, and what alternatives were considered.

**Structure:**

1. **Problem statement** — in one short paragraph. If you can't write this, you don't understand the problem well enough.
2. **Context** — what the reader needs to know to understand the problem.
3. **Approach** — what solution was chosen and why.
4. **Alternatives considered** — what was rejected and why. This section is often the most valuable to future readers.
5. **Key decisions** — specific choices and their rationale.
6. **Implications** — what this means for the rest of the system.

**Audience notes:** Design documents serve both audiences heavily. Designers need the problem statement, context, and approach. Developers need the key decisions and implications. The inverted pyramid means a busy stakeholder can read sections 1–3 and understand the design; a developer implementing it reads through to the end.

---

## 8. Cross-Document Architecture

### Link, don't repeat

Each concept has a home document. Other documents may reference it briefly and link to the home document for full detail. Duplication across documents is kept to the minimum needed for each document to be readable on its own.

If you find yourself copying a paragraph from one document into another, stop. Write a one-sentence summary and link to the source.

### Show, don't explain

Where possible, demonstrate a concept with a concrete example rather than describing it in prose. A three-line code snippet or a short terminal transcript is worth more than a paragraph of explanation.

### Honest positioning

Documentation must be completely factual about what a product does well, what it costs (in time, complexity, or money), and when it is not the right choice. Trust is more valuable than persuasion.

### Consistent terminology

Use one term for one concept across all documents. Define it in its home document. Use it consistently everywhere else. Do not invent synonyms for variety.

---

## 9. Tables, Figures, and Examples

### Tables

- Don't use a table when a list would do. Use tables for structured comparisons or multi-attribute data.
- Use tables sparingly. Avoid large tables if you can.
- Put identifying information (names, commands, terms) in the leftmost column.
- Make entries parallel in structure.
- Give every table a caption or introductory sentence so its purpose is clear.
- Don't leave cells blank — use *N/A* or *None*.
- Keep tables responsive: limit columns, keep cell text short.

### Figures and diagrams

- Use diagrams where they communicate more efficiently than prose — architecture overviews, data flow, state machines.
- Every figure should have a caption that lets the reader understand it without reading the surrounding text.
- Introduce each figure in the text with a complete sentence.

### Code examples

- Every reference entry and every concept that involves code should include at least one example.
- Keep examples minimal — show the thing being documented and nothing else.
- Use realistic values, not `foo` and `bar`, when realistic values help understanding.
- If an example requires context (a specific file, a running server), state the prerequisites.

### Emoji

Use emoji sparingly, either to aid navigation or understanding. Emoji can carry hidden meanings and ambiguity. Avoid ✅ and ❌ — use text instead. Avoid using emoji as bullet points unless they greatly aid clarity and are not ambiguous.

---

## 10. Checklist

Use this checklist when planning, writing, or reviewing any document.

### Planning

- [ ] The document's purpose is captured in a single sentence.
- [ ] The document type has been identified and the appropriate structure chosen.
- [ ] The target audience has been identified, with any assumptions stated.
- [ ] Three to five key messages have been listed.
- [ ] An outline has been written using sentences, not keywords.
- [ ] Key examples, figures, and tables have been drafted before the prose.
- [ ] Facts have been verified against the implementation, not design documents alone.

### Document-level structure

- [ ] The opening answers: what is this, who is it for, what will you learn.
- [ ] The introduction (if present) follows: context → gap → goal → approach.
- [ ] Sections are arranged in the order the reader needs them.
- [ ] A table of contents is included for longer documents.
- [ ] Any summary or abstract was written last and reflects the final content.
- [ ] The document ends with next steps, a summary, or links to related documents.

### Inverted pyramid

- [ ] The most important information appears first — in the document and in every section.
- [ ] Tone is more accessible at the top, more precise deeper in.
- [ ] Technical depth increases as the reader goes deeper.
- [ ] A reader who stops at any point has absorbed the most valuable content so far.
- [ ] Both audiences (designers and developers) find what they need at the depth they naturally reach.

### Section-level structure

- [ ] Each section opens with its key point.
- [ ] Each section covers one topic.
- [ ] Detail follows in descending order of importance.
- [ ] Each paragraph opens with a topic sentence.
- [ ] Paragraphs are 3–7 lines. No walls of text.

### Scannability

- [ ] Headings form a readable outline on their own.
- [ ] Headings use sentence case, are short, and front-load important words.
- [ ] Headings at the same level use parallel structure.
- [ ] Lists are used for sets of related items; numbered lists for sequential steps.
- [ ] List items are parallel in structure.
- [ ] White space separates sections and aids readability.
- [ ] Keywords are front-loaded in headings, list items, and paragraph openings.

### Tables, figures, and examples

- [ ] Tables are used only when a list won't do.
- [ ] Every table and figure has a caption or introductory sentence.
- [ ] Every concept that involves code includes at least one example.
- [ ] Examples are minimal and use realistic values.

### Cross-document

- [ ] Each concept has one home document; other documents link to it.
- [ ] No paragraph-length duplication between documents.
- [ ] Terminology is consistent across all documents.
- [ ] Claims about product capabilities are factual and honest.

### Audience

- [ ] The document states who it is for.
- [ ] Exclusively technical sections are marked so non-technical readers can skip them.
- [ ] Concepts are explained without jargon at the top of each section.
- [ ] Technical specifics appear deeper, where readers have chosen to look for them.

---

*This guide covers structure and planning. For prose style, grammar, sentence construction, and punctuation, see [technical-writing-guide.md](technical-writing-guide.md). For detailed guidance on voice, verbs, vocabulary, and flow, see [scientific-writing-styleguide.md](scientific-writing-styleguide.md).*