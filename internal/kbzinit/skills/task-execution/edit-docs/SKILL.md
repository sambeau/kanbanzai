---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: edit-docs
description:
  expert: "Structural and developmental editing verifying inverted-pyramid
    compliance, section architecture, scannability, tone gradient, and
    audience fit — without touching sentence-level prose"
  natural: "Check a document's structure and organisation — is the pyramid
    right, do the headings tell a story, is it scannable?"
triggers:
  - edit document structure
  - review document organisation
  - check document architecture
  - developmental edit
  - structural review of documentation
roles: [doc-editor]
stage: documenting
constraint_level: medium
---

## Vocabulary

- **inverted pyramid compliance** — whether a document places the most important
  content first at every level: document, section, paragraph. The reader who stops
  at any point has already absorbed the most valuable content.
- **heading skeleton** — the ordered list of all headings, read alone as a document
  outline. If the skeleton doesn't tell a clear story, the structure needs work.
- **tone gradient** — the progression from conversational at the top to precise at
  depth. Applies within documents and within sections. Accessible language serves
  all readers; technical precision at depth serves those who go looking for it.
- **scannability** — whether a reader who only scans headings and first sentences
  can navigate to what they need. Most readers scan; structure must reward scanning.
- **section focus** — each section covering exactly one topic. If you can't write a
  heading that describes the section's content, it lacks focus.
- **structural tell** — a formatting pattern that reveals AI generation: inline-header
  lists, title-case headings, emoji bullets, mechanical three-item lists. These erode
  reader trust and indicate formulaic rather than intentional structure.
- **front-loading** — placing important words first in headings, list items, paragraph
  openings. The scanner's eye hits the first few words of each line — make them count.
- **parallel structure** — matching grammatical form across headings and list items at
  the same level. If one heading is a verb phrase, all headings at that level should be.
- **document arc** — the logical progression from opening through body to closing.
  Sections arranged in the order the reader needs them, not the order the author
  thought of them.
- **wall of text** — a paragraph exceeding 7 lines with no visual breaks. A
  scannability failure that causes readers to skip content entirely.
- **topic sentence** — the first sentence of each paragraph, stating the paragraph's
  point. A reader who only reads topic sentences should get the gist of the section.
- **cross-reference integrity** — all links between documents resolve to valid, current
  targets. Each concept links to its home document rather than being duplicated.
- **white space** — deliberate spacing between sections and paragraphs as a readability
  tool. White space is a feature, not wasted space.

## Anti-Patterns

### Flat Structure

- **Detect:** All sections carry equal weight; no pyramid — the document reads like a
  flat list of topics rather than a progression from important to detailed.
- **BECAUSE:** Readers scan from the top. A flat structure forces them to read
  everything to find what matters. The inverted pyramid exists so they don't have to.
- **Resolve:** Reorder sections by importance. Within each section, lead with the key
  point and follow with supporting detail in descending order.

### Heading Skeleton Failure

- **Detect:** Reading only the headings does not reveal the document's structure or
  scope. Headings are vague ("Overview," "Details," "Notes") or inconsistent in level.
- **BECAUSE:** Most readers navigate by headings. Vague headings force them to read
  body text to understand what a section contains, defeating the purpose of document
  structure.
- **Resolve:** Rewrite headings to be specific and front-loaded. Ensure headings at the
  same level use parallel structure. Limit to 2–3 heading levels.

### Wall of Text

- **Detect:** Paragraphs exceed 7 lines. Sections contain no lists, no white space, no
  visual breaks.
- **BECAUSE:** Dense text blocks cause readers to skip content entirely. Short
  paragraphs, lists, and white space make content inviting and scannable.
- **Resolve:** Break paragraphs at 3–7 lines. Convert sequences to numbered lists.
  Convert groups to bulleted lists. Add white space between sections.

### Missing Opening Point

- **Detect:** A section launches into detail without first stating what the section is
  about and why it matters.
- **BECAUSE:** A reader who scans only the first sentence of each section should
  understand the document's structure and key messages. Missing opening points break
  this contract.
- **Resolve:** Open every section with its key point. Move context and elaboration
  below the opening statement.

### Structural Tell Blindness

- **Detect:** AI formatting patterns pass without being flagged: every list uses
  bold-header-colon format (inline-header lists), all headings use Title Case, emoji
  appear as bullet points, every list has exactly 3 items.
- **BECAUSE:** Structural tells make documentation look machine-generated, eroding
  reader trust. They also indicate the structure was generated formulaically rather
  than designed for the content.
- **Resolve:** Convert inline-header lists to prose where descriptions are single
  sentences. Use sentence case for headings. Remove emoji from technical prose. Vary
  list lengths to match the content.

### Tone Inversion

- **Detect:** The document opens with dense technical jargon and becomes more accessible
  deeper in — the tone gradient is backwards.
- **BECAUSE:** The inverted pyramid applies to tone as well as content. Accessible
  language at the top serves all readers, while technical precision at depth serves
  developers who have actively chosen to go looking for it.
- **Resolve:** Rewrite the opening in accessible language. Move jargon and technical
  specifics deeper into the section.

## Checklist

Copy this checklist and track your progress:

### Inverted pyramid

- [ ] The most important information appears first in the document and in every section
- [ ] Tone is more accessible at the top, more precise deeper in
- [ ] Technical depth increases as the reader goes deeper
- [ ] A reader who stops at any point has absorbed the most valuable content so far

### Section structure

- [ ] Each section opens with its key point
- [ ] Each section covers one topic
- [ ] Detail follows in descending order of importance
- [ ] Each paragraph opens with a topic sentence
- [ ] Paragraphs are 3–7 lines; no walls of text

### Scannability

- [ ] Headings form a readable outline on their own
- [ ] Headings use sentence case, are short, and front-load important words
- [ ] Headings at the same level use parallel structure
- [ ] Lists are used for sets of related items; numbered lists for sequential steps
- [ ] List items are parallel in structure
- [ ] White space separates sections and aids readability
- [ ] No structural tells (inline-header lists used formulaically, title case, emoji
  bullets, mechanical tricolons)

### Cross-document

- [ ] Each concept links to its home document rather than being duplicated
- [ ] Terminology is consistent across all referenced documents

## Procedure

### Step 1: Read purpose and audience

Read the document's purpose statement and audience assumptions. If they are
missing, flag this as a finding before proceeding. A document without a stated
purpose cannot be structurally evaluated against intent.

### Step 2: Extract the heading skeleton

Copy all headings in order and read them as a standalone outline.

1. Does the skeleton tell a clear story?
2. Are headings specific — could a reader navigate by headings alone?
3. Are headings front-loaded with important words?
4. Do headings at the same level use parallel structure?
5. Are heading levels consistent (no jumps from H2 to H4)?

Record the skeleton in the output. It is evidence for every structural finding.

### Step 3: Check inverted pyramid at document level

1. Are the most important sections first?
2. Does the opening answer what this is, who it's for, and why it matters?
3. Could the document be truncated at any point and still be useful up to that
   point?

### Step 4: Check inverted pyramid within each section

1. Does each section open with its key point?
2. Does detail follow in descending order of importance?
3. Are there sections that bury the point under context or background?

### Step 5: Check tone gradient

1. Is language accessible at the top of the document and each section?
2. Does technical precision increase with depth?
3. Is the gradient inverted anywhere — jargon at the top, plain language below?

### Step 6: Check scannability

1. Headings: specific, front-loaded, sentence case, parallel at each level.
2. Lists: used for groups and sequences, items parallel in structure.
3. White space: present between sections and paragraphs.
4. Front-loading: important words first in headings, list items, and paragraph
   openings.
5. Paragraph length: 3–7 lines. Flag any wall of text.

### Step 7: Check for structural tells

1. Inline-header lists used formulaically (bold term, colon, description on
   every list item).
2. Title-case headings where sentence case is the project convention.
3. Emoji in headings or bullet points.
4. Every list having exactly 3 items.

Flag for the style editing stage if the issue is AI-artifact level.

### Step 8: Check cross-reference integrity

1. Do links between documents resolve to valid targets?
2. Is any concept duplicated across documents instead of cross-referenced?
3. Is terminology consistent with the project's vocabulary?

### Step 9: Check document type structure

Does the document follow the structural template for its type (README,
getting-started, manual, reference, design) as defined in the documentation
structure guide? Flag missing required sections.

### Step 10: Compile findings

Each finding includes:

1. **Location** — section or line range.
2. **Issue** — what's wrong, stated specifically.
3. **Why it matters** — impact on readers.
4. **Recommendation** — a concrete fix.

Classify each finding as:

- **structural-blocking** — must be fixed before later pipeline stages. The
  document's organisation prevents readers from finding or understanding content.
- **structural-suggestion** — an improvement that would help readers but does
  not block later stages.

## Output Format

The Edit stage produces a report — it does not edit the document file directly. The orchestrator applies structural-blocking findings to the document before passing it to the next stage. The report must contain enough detail (locations, recommendations) for the orchestrator to make the changes.

```
## Structural Edit Report

**Document:** {document path}
**Document type:** {README | getting-started | manual | reference | design}
**Purpose statement:** {found | missing}

### Heading Skeleton
{copy of all headings in order}
**Assessment:** {clear story | needs rework | vague headings | inconsistent levels}

### Findings

#### {Finding title}
- **Location:** {section or line}
- **Classification:** {structural-blocking | structural-suggestion}
- **Issue:** {what's wrong}
- **Why it matters:** {impact on readers}
- **Recommendation:** {specific fix}

### Summary
- Structural-blocking findings: {count}
- Structural suggestions: {count}
- Overall structure assessment: {sound | needs rework | major restructure needed}
```

## Examples

### BAD: Rubber-stamp structural edit

```
The document is well-structured and follows a logical order. The headings
are clear and the content flows well. Some sections could be shorter but
overall the organisation is good.
```

**WHY BAD:** No heading skeleton extracted. No specific findings. No evidence.
No locations. "Well-structured" and "flows well" are qualitative impressions, not
structural analysis. A human cannot determine what was actually checked. This is
indistinguishable from not editing at all.

### GOOD: Evidence-backed structural edit

```
## Structural Edit Report

**Document:** docs/getting-started.md
**Document type:** getting-started
**Purpose statement:** found

### Heading Skeleton
1. Introduction
2. Background
3. Configuration
4. Quick Start
5. Reference

**Assessment:** Inverted-pyramid violation — "Background" (§2) appears before
"Quick Start" (§4). Most readers want the quick start first.

### Findings

#### Inverted pyramid: Background before Quick Start
- **Location:** §2 and §4
- **Classification:** structural-blocking
- **Issue:** Background context (§2) precedes the quick start (§4). Readers
  scanning from the top encounter history before they learn how to use the tool.
- **Why it matters:** Getting-started documents exist to get readers running
  quickly. Burying the quick start below background content means most readers
  scroll past content they came for.
- **Recommendation:** Move Quick Start to §2. Move Background to §4 or fold
  relevant context into the introduction.

#### Wall of text in Configuration
- **Location:** §3, paragraphs 2–3
- **Classification:** structural-suggestion
- **Issue:** Two paragraphs of 12 and 9 lines each describe configuration
  options as prose. No lists, no visual breaks.
- **Why it matters:** Readers scanning for a specific option must read every
  line. Dense blocks cause readers to skip content entirely.
- **Recommendation:** Convert configuration options to a bulleted list or
  table. Break the remaining prose into 3–5 line paragraphs.

#### Structural tell: inline-header lists in §5
- **Location:** §5 Reference, all three lists
- **Classification:** structural-suggestion
- **Issue:** Every list in the reference section uses bold-header-colon format
  with exactly three items. The pattern is formulaic rather than designed for
  the content.
- **Why it matters:** Mechanical formatting signals AI generation and erodes
  trust. List lengths should match the content, not a template.
- **Recommendation:** Convert single-sentence descriptions to prose. Vary list
  lengths. Flag for the style stage.

### Summary
- Structural-blocking findings: 1
- Structural suggestions: 2
- Overall structure assessment: needs rework
```

**WHY GOOD:** Heading skeleton extracted and assessed. Each finding has a specific
location, a concrete issue, an explanation of reader impact, and a recommendation.
Findings are classified by severity. The reviewer checked pyramid, scannability, and
structural tells — and produced evidence for each. A human can verify every claim.

## Evaluation Criteria

These criteria are for evaluating the structural edit output, not for
self-evaluation during the edit. They are phrased as gradable questions to
support automated LLM-as-judge evaluation.

1. Are all structural findings backed by specific location and evidence, not
   qualitative impressions? **Weight: 0.25.**
2. Is the heading skeleton extracted and assessed? **Weight: 0.20.**
3. Are inverted-pyramid violations identified at both document and section
   level? **Weight: 0.20.**
4. Are scannability issues (walls of text, missing white space, vague headings)
   caught? **Weight: 0.15.**
5. Are structural tells flagged? **Weight: 0.10.**
6. Does the review stay within scope — structure only, no sentence-level
   editing? **Weight: 0.10.**

## Questions This Skill Answers

- Does this document follow the inverted pyramid?
- Do the headings tell a coherent story?
- Is this document scannable?
- Is the tone gradient right — accessible at top, precise at depth?
- Does this document match the structural template for its type?
- Are there AI structural tells that need fixing?
- Where should sections be reordered?
- What structural issues must be fixed before the style editing stage?
- Does the document's arc match the order the reader needs?