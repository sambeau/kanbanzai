# Design: Document Intelligence Adoption and Integration

| Field         | Value                                                                     |
|---------------|---------------------------------------------------------------------------|
| Date          | 2026-04-23                                                                |
| Status        | Draft                                                                     |
| Author        | Architecture task                                                         |
| Based on      | `work/reports/doc-intel-usage-report.md`                                  |
| Research      | `work/research/document-retrieval-for-ai-agents.md` (Option A)            |
| Research      | `work/research/skill-authoring-best-practices.md`                         |
| Complements   | `work/design/doc-intel-enhancement-design.md` (technical infrastructure)  |

---

## 1. Purpose

### 1.1 The problem: design amnesia at scale

As a project grows to hundreds of features and tens of design documents, a failure
mode emerges that has nothing to do with code quality or test coverage. New designs are
written without awareness of related prior work. Features are built in isolation.
Decisions made six months ago are re-made, sometimes differently. The project becomes
less homogeneous over time — not because any individual decision was wrong, but because
each decision was made without adequate context about what came before.

This is **design amnesia at scale**. The knowledge was never lost — it is all stored in
workflow documents. But no single agent has read the whole corpus, so no single agent
can answer: *"What have we already decided about this? How does this feature relate to
that one? Why don't we already do it this way?"*

The human project owner becomes the only source of continuity. At more than ~100
features, that continuity begins to break down.

The `doc_intel` system exists to solve this problem. It is **institutional memory made
queryable**: a structured, classified index of every design decision, requirement, and
rationale in the project corpus, accessible in seconds rather than requiring an agent to
read dozens of documents.

### 1.2 Why classification — not grep — is the mechanism

A grep-based corpus search could partially address this problem, but it fails at the
critical case: the **unknown unknowns**. An architect designing a new feature knows their
problem domain but not what the existing corpus says about it. Grep requires the
searcher to already know the vocabulary. A classified concept graph surfaces relevant
prior work that the architect did not know to search for.

Concept-based retrieval answers questions grep cannot:
- *"What other features touch this same architectural concept, even if they use
  different terminology?"*
- *"What decisions have been made that constrain the approach I'm considering?"*
- *"What is the full landscape of prior work on this topic?"*

These are the questions that prevent design amnesia. None of them are answerable by
text matching alone. They require semantic classification — concepts, roles (decision,
rationale, requirement), and typed graph edges between sections and entities.

This reframes why Layer 3 classification matters. It is not primarily an instrumentation
improvement or a token-efficiency optimisation. **It is the primary retrieval mechanism
for design continuity.** Without a populated concept registry, the corpus cannot answer
design questions.

### 1.3 The current state is a paradox

The Document Intelligence Usage Report (April 2026) found that 332 of 334 registered
documents have no Layer 3 classification, the concept registry is empty, and all 59
knowledge entries have `use_count: 0`. The infrastructure is correct and well-designed.
The data is almost entirely absent.

This is the paradox: the system that is meant to prevent design amnesia cannot be used
until the corpus is classified, and the corpus is not being classified because no
workflow step mandates it.

This document defines how to break that loop.

### 1.4 Scope

**In scope:**

- Mandatory corpus consultation during the design stage (`write-design`, `write-spec`)
- Design document template: required Related Work section
- Corpus completeness: session-start audit, batch registration, onboarding procedure
- Classification as a mandatory step at registration, at review, and before design begins
- Mandatory knowledge retrieval at implementation time
- Access instrumentation for `doc_intel` and `knowledge` tools
- Plan close-out knowledge curation

**Out of scope:**

- Technical infrastructure changes (FTS5, SQLite, concept aliases — see enhancement design)
- Automated or pipeline-driven classification (classification remains agent-driven)
- Changes to the `doc_intel` tool action surface beyond access logging
- Embedding-based semantic search (evaluate after enhancement design)

### 1.5 Relationship to existing designs

`doc-intel-enhancement-design.md` defines five technical improvements: FTS5 full-text
search, SQLite graph storage, batch classification protocol, knowledge↔document
cross-queries, and concept alias resolution. Those changes make the system more
capable. This design makes agents use it — and specifically, use it at the moments that
matter most for design continuity.

The two designs are complementary and sequentially dependent in one direction: concept
search (`find(concept: ...)`) requires both the technical enhancement and the
classification data this design mandates. FTS search and SQLite migration can ship
independently of this design.

---

## 2. Root Cause Analysis

### 2.1 The primary gap: design stage has no corpus consultation mandate

The `write-design` and `write-spec` skills contain no requirement to consult the
existing corpus before writing. An architect can produce a complete design document
without calling a single `doc_intel` tool. This is the most critical gap: the design
stage is where decisions with long-term consequences are made, and it is entirely
unconnected to the accumulated design knowledge in the corpus.

Implementation-stage agents (reading specs to write code) are secondary. The problem
of design amnesia lives upstream, at the moment a new design is being written.

### 2.2 The corpus cannot be trusted to be complete

Nine files are on disk but unregistered. Four registered files are missing from disk.
The session-start checklist in `kanbanzai-getting-started` does not mandate a corpus
integrity check. An agent beginning a corpus search has no assurance that the search
is complete — the documents it finds may not represent everything that exists.

This problem is compounded when Kanbanzai is adopted on an existing project with
pre-existing documentation. Those documents are entirely invisible to `doc_intel` until
registered and classified. A corpus that is partially indexed is worse than one that is
fully indexed, because it creates false confidence: the agent searches, finds nothing
relevant, and proceeds — not realising the relevant document was never registered.

### 2.3 Layer 3 classification is absent from all mandatory workflows

The batch classification protocol exists in `kanbanzai-documents`, but it is positioned
at the end of a long skill (receiving less attention due to the U-shaped attention curve
established by Liu et al., "Lost in the Middle", 2024) and is framed as a reference
section rather than an obligation.

There is no mandatory trigger in the feature lifecycle. The `orchestrate-review` skill
does not classify documents before dispatching reviewers. The `write-design` skill does
not classify the design it produces. Classification is purely discretionary — and
discretionary tasks are skipped under time pressure.

### 2.4 Knowledge is not consumed at implementation time

`kanbanzai-agents` mandates knowledge contribution but not consumption. The `use_count`
for all 59 knowledge entries is zero. Context packets surface entries passively via
`next`, but `use_count` only increments on explicit API calls — so even entries that
appear in context packets register no usage. The feedback loop (contribute → consume →
confirm → increase confidence) is broken at the consumption step.

### 2.5 Instrumentation is absent

There are no access counters on `doc_intel` reads or `knowledge` retrievals. This means
adoption regressions are invisible until a full corpus audit is conducted — an expensive
and periodic operation rather than a continuous signal.

---

## 3. Fix 1: Design-Stage Corpus Consultation Mandate

### 3.1 Problem

Designs are written in isolation. The `write-design` and `write-spec` skills contain no
corpus consultation phase. The document template has no Required Work section. New
features can be designed without any awareness of related prior decisions.

### 3.2 Solution

Four complementary mechanisms, ordered from most to least immediate:

1. **`write-design` skill: mandatory discovery phase** before any content is written
2. **Design document template: required "Related Work" section** that cannot be empty
3. **`reviewer-conformance` skill: blocking check** for substantive Related Work
4. **`write-spec` skill: cross-reference check** for consistency with related specs

Together these form a complete enforcement chain: the architect is required to search,
required to document what they find, and the reviewer is required to verify they did.

### 3.3 Changes to `write-design/SKILL.md`

Add **Phase 0: Corpus Discovery** as the first phase, before any content is written:

```
Phase 0: Corpus Discovery

Before writing a single line of design content, search the corpus for
existing work that relates to this feature. This phase is not optional.

1. Search by concept. For each primary concept in this feature, call:
     doc_intel(action: "search", query: "<concept>")
   and for classified documents:
     doc_intel(action: "find", concept: "<concept>")
   Note: if concept search returns no results, the corpus may be
   unclassified — fall back to FTS search and grep.

2. Search by related entity. If the feature relates to known features,
   find all documents that reference them:
     doc_intel(action: "find", entity_id: "<FEAT-xxx>")

3. Search for prior decisions. For each related document found, extract
   its decisions:
     doc_intel(action: "find", role: "decision", scope: "<DOC-xxx>")

4. Synthesise. Produce a list of: related documents, relevant decisions
   that constrain this design, and open questions raised by prior work.

5. Write the Related Work section of the design document from this
   synthesis BEFORE writing any other section.

BECAUSE: A design written without corpus consultation may duplicate
prior work, contradict existing decisions, or miss the context that
explains why the current approach is the way it is. At project scale,
these omissions compound into the fragmentation and inconsistency that
make large projects unmaintainable. The Related Work section is the
primary mechanism for preventing this.
```

**Checklist addition:**

```
- [ ] Conducted corpus discovery (concept search, entity search, decision extraction)
- [ ] Wrote Related Work section before writing any design content
- [ ] Cross-referenced at least one prior decision that constrains this design, OR
      attested that corpus search found no related work
```

### 3.4 Design document template update

Add a **Required: Related Work** section to `work/templates/specification-prompt-template.md`
and the equivalent design template. The section must contain one of:

**Option A — Related work found:**
```
## Related Work

### Prior designs and specifications consulted
- [Design title](path/to/design.md) — [how it relates to this design]
- [Spec title](path/to/spec.md) — [how it relates]

### Decisions that constrain this design
- [Decision summary] (from [document], §[section]) — [how it applies here]

### How this design extends or diverges from prior work
[Narrative explaining relationship to the above]
```

**Option B — No related work found:**
```
## Related Work

Corpus search conducted for concepts: [X, Y, Z].
Entity search conducted for: [FEAT-xxx, FEAT-yyy] (if applicable).
No directly related prior work found in the classified corpus.
```

Option B is a valid answer. An empty or missing Related Work section is not. A design
that skips this section entirely is incomplete regardless of the quality of its other
content.

### 3.5 Changes to `reviewer-conformance/SKILL.md`

Add a **blocking check** for Related Work section quality:

```
Related Work Section Check (blocking):
- [ ] Related Work section is present in the design document
- [ ] Section contains either substantive cross-references OR an explicit
      "no related work found" attestation with search evidence
- [ ] If related documents exist in the corpus that clearly relate to this
      design, the design engages with them — it does not ignore them silently

A design that omits Related Work entirely, or that contains placeholder
text ("TBD", "N/A") without supporting evidence, is REJECTED. This is a
blocking finding.

BECAUSE: The Related Work section is the primary enforcement mechanism
for design continuity at scale. A system that does not enforce it provides
false assurance — the corpus grows but its influence on new work approaches
zero. An ignored Related Work requirement is worse than no requirement, because
it creates the appearance of a functioning system.
```

### 3.6 Changes to `write-spec/SKILL.md`

Add a **cross-reference check** at the start of specification writing:

```
Before writing specification content, verify:
1. The design document for this feature has a substantive Related Work section.
   If not, the design is incomplete — STOP and flag this to the orchestrator.
2. Search for specifications of features identified in the Related Work section:
     doc_intel(action: "find", role: "requirement", scope: "<DOC-related-spec>")
   Identify any requirements in adjacent specs that this spec must be consistent with.
3. Note deliberate divergences. If this spec takes a different approach than an
   adjacent spec, document why in the spec's design rationale section.

BECAUSE: Specifications that are unaware of related specs produce inconsistent
behaviour across features. A user encountering two features that handle the same
concept differently will correctly perceive the project as unfinished.
```

---

## 4. Fix 2: Corpus Completeness and Onboarding

### 4.1 Problem

The corpus cannot be trusted to be complete. Unregistered documents are invisible to
`doc_intel`, making corpus searches silently incomplete. When Kanbanzai is adopted on
an existing project, the entire pre-existing documentation corpus starts as invisible.

A partial index is worse than no index for design consultation: the architect searches,
finds nothing, and concludes there is no related work — not realising the most relevant
document was never registered.

### 4.2 Solution

Two mechanisms: a **session-start integrity check** for ongoing hygiene, and an
**onboarding procedure** for new or existing projects. Both use existing tools
(`doc audit`, `doc import`); what they add is mandatory integration into the workflow.

### 4.3 Session-start integrity check

Add a mandatory step to `kanbanzai-getting-started/SKILL.md`:

```
Corpus Integrity Check (at every session start):

1. Call doc(action: "audit") and review the output.
2. If audit shows files on disk but not registered:
     Call doc(action: "import", path: "work")
   to register all unregistered documents in configured roots.
3. If audit shows registered files missing from disk: flag these as stale
   records — call doc(action: "delete", id: "DOC-xxx") for each.
4. After any batch registration, run a classification pass on newly
   registered documents (see Fix 3).

BECAUSE: An incomplete corpus produces false negatives in design searches.
An architect who searches and finds nothing relevant may proceed without
knowing that the most relevant design document simply was not registered.
The integrity check takes seconds and removes this uncertainty.
```

### 4.4 New project onboarding

When Kanbanzai is initialised on a project for the first time (no prior `.kbz/`
directory), the getting-started skill should run the onboarding procedure:

```
1. Configure document roots in .kbz/config.yaml to cover all directories
   containing project documentation.
2. Run doc(action: "import", path: "<each-root>") for each configured root.
3. Verify with doc(action: "audit") — target: 0 unregistered files.
4. Run batch classification, prioritised by document type:
   a. Specifications first (most structured, highest retrieval value)
   b. Designs second (decisions, rationale, architectural context)
   c. Dev-plans third (task-oriented, lower discovery value)
   d. Research and reports last
   Estimate: ~5–10 minutes per document for classification, including
   guide + section reads + classify call. A 50-document corpus requires
   roughly 4–8 hours of agent time for full classification.
5. After classification, validate with:
     doc_intel(action: "find", role: "decision")
   If this returns results, the concept registry is populated and design
   consultation is functional.
```

### 4.5 Existing project adoption

When Kanbanzai is added to an existing project that already has documentation outside
standard `work/` directories (README files, architecture docs, decision logs, API
specs), the onboarding procedure must account for non-standard locations:

```
1. Audit the repository for documentation files not covered by configured roots:
     find . -name "*.md" | grep -v ".kbz"
2. Decide which documents should be in the corpus. Not all markdown files
   need to be registered — only those containing design decisions, specifications,
   architectural rationale, or requirements.
3. Add additional roots to .kbz/config.yaml as needed.
4. Register and classify as per §4.4.
```

The key principle: **the corpus should be complete enough that a designer can trust a
negative result**. If searching for a concept returns nothing, it should mean "this has
not been addressed" — not "this might have been addressed in an unregistered document."

---

## 5. Fix 3: Classification as the Primary Retrieval Mechanism

### 5.1 Reframing

The prior adoption design framed Layer 3 classification as an operational improvement —
better search results, more informative guides. This undersells it.

Classification is the mechanism that makes the corpus answer design questions. Without
it, `find(concept: "X")` always returns zero results, and design consultation degrades
to grep. With it, an architect can discover every document in the corpus that engages
with a concept — including documents that use different vocabulary — in a single tool
call.

The classification investment compounds: a document classified once remains classified
until its content changes, and every subsequent design query benefits.

### 5.2 Classification on registration (primary mechanism)

When an agent registers a document it has just written, it has the document content in
context. This is the lowest-cost moment to classify — no additional reads are required.

The `kanbanzai-documents` skill changes from Fix 2 of the original design apply here:
move the classification section earlier in the skill, reframe it as an active
obligation, and add it to the Document Creation Checklist as a required step.

The `doc register` response already includes a `classification_nudge`. This nudge
should be treated as a mandatory instruction, not an optional suggestion.

### 5.3 Classification before design begins (highest-value trigger)

The most valuable moment to classify a document is before a related design begins.
When a feature enters the `designing` stage, add a pre-design step:

```
Before writing any design content, verify that the corpus documents most
relevant to this feature are classified. Call doc_intel(action: "pending")
and classify any related unclassified documents using the priority order:
specification → design → dev-plan.

BECAUSE: An unclassified corpus produces false negatives in concept search.
Classifying related documents before starting design work ensures the
discovery phase (Fix 1, Phase 0) operates on a complete semantic index.
```

### 5.4 Classification at review (existing trigger, reframed)

The change to `orchestrate-review/SKILL.md` described in the previous design version
remains valid: classify the feature's documents as a Step 1 prerequisite before
dispatching reviewers. This is now understood as part of the classification pipeline
rather than a standalone reviewer concern — the review classification ensures the
classified corpus is current for the next design that may reference this feature's work.

### 5.5 Batch classification for the existing backlog

The 332 currently unclassified documents represent a one-time debt. Clearing this
backlog unlocks concept search across the full project history. The batch procedure
from `kanbanzai-documents` applies, with a recommended sequencing:

First pass: all approved specifications and designs (the highest-value documents for
design consultation). This alone, covering roughly 170 documents, would make the
concept registry substantive and enable the primary design-continuity use case.

Second pass: dev-plans and reports. Lower priority — these are useful for knowledge
extraction but less often the subject of cross-feature design consultation.

---

## 6. Fix 4: Mandatory Knowledge Retrieval at Implementation Time

### 6.1 Problem

Agents contribute knowledge but never read it. No skill mandates consumption. The
`use_count` is 0 for all 59 entries.

### 6.2 Solution

Add explicit, required knowledge retrieval steps to `implement-task/SKILL.md`,
`orchestrate-development/SKILL.md`, and `kanbanzai-agents/SKILL.md`.

### 6.3 Changes to `implement-task/SKILL.md`

**Phase 1 addition** (after claiming the task):

```
1a. Call knowledge(action: "list", tags: ["<domain>", "<feature-area>"]) using
    tags derived from the task's feature area. Review all returned entries before
    proceeding. Note any entries describing known pitfalls for this task's domain.
    BECAUSE: Knowledge entries record hard-won discoveries from previous tasks.
    An agent that skips this step re-discovers the same problems from scratch.
```

**Phase 4 addition** (in the finish call):

```
Call knowledge(action: "confirm", id: "<KE-id>") for entries that proved accurate.
Call knowledge(action: "flag", id: "<KE-id>") for entries that were inaccurate.
BECAUSE: Confirmation is how the knowledge base self-curates. Unflagged inaccurate
entries continue to mislead future agents indefinitely.
```

**Checklist additions:**
```
- [ ] Called knowledge list with domain-relevant tags before writing any code
- [ ] Confirmed accurate and flagged inaccurate knowledge entries after task completion
```

### 6.4 Changes to `orchestrate-development/SKILL.md`

**Phase 1 addition** (after reading the dev-plan):
```
1a. Call knowledge(action: "list", tags: ["<feature-area>"], status: "confirmed")
    and surface relevant entries to sub-agents via handoff instructions.
```

**Phase 6 Close-Out addition:**
```
4a. Confirm tier 2 knowledge entries contributed during this feature's development.
    Call knowledge(action: "list", status: "contributed", tier: 2) and confirm
    accurate entries. This is the primary knowledge curation mechanism.
```

### 6.5 Changes to `kanbanzai-agents/SKILL.md`

Add knowledge retrieval and confirmation to the Task Lifecycle Checklist and update
the Context Assembly section to mandate active querying after `next(id)` rather than
relying solely on the context packet's passive surfacing.

---

## 7. Fix 5: Access Instrumentation

### 7.1 Problem

Adoption regressions are invisible. The system cannot answer "how often is the
knowledge base accessed?" or "which document sections are most consulted?". Future
failures will require another full corpus audit to detect.

### 7.2 Knowledge base instrumentation

**New fields on `KnowledgeEntry`:**

| Field              | Type      | Description                                                    |
|--------------------|-----------|----------------------------------------------------------------|
| `last_accessed_at` | timestamp | Updated on every `list` or `get` call that returns this entry  |
| `recent_use_count` | int       | Rolling 30-day access count, separate from all-time `use_count`|

Add `sort: "recent"` to `knowledge list` to surface entries most accessed in the last
30 days — the recommended default for pre-task knowledge queries.

### 7.3 Document intelligence instrumentation

**New fields on `DocumentIndex`:** `access_count` (cumulative) and `last_accessed_at`.

**New fields on `SectionIndex`:** `access_count` (per-section `section` calls) and
`last_accessed_at`.

Increment document-level counters on `outline`, `guide`, `find`, `search`. Increment
section-level counters on `section`. Updates are written lazily; counts are approximate.

**`doc(action: "audit")` extension:** add a "Most Accessed Documents" table (top 10 by
30-day access count) to make the instrumentation actionable without requiring a
separate query.

---

## 8. Fix 6: Plan Close-Out Knowledge Curation

### 8.1 Problem

All knowledge entries remain in `contributed` status. The knowledge base is an
append-only log rather than a confidence-weighted reference.

### 8.2 Solution

Add a mandatory confirmation pass to `orchestrate-development` Phase 6 Close-Out and
to the plan review workflow:

```
At plan close-out:
1. Call knowledge(action: "list", status: "contributed", tier: 2)
2. For each relevant entry: confirm (accurate), flag (inaccurate), or
   retire (superseded by architectural changes in this plan)
3. Tier 3 entries are self-pruning — promote valuable ones to tier 2
   via knowledge(action: "promote") rather than confirming them.
```

---

## 9. What Changes

### 9.1 Skill files

| File | Change |
|------|--------|
| `.kbz/skills/write-design/SKILL.md` | Add Phase 0 corpus discovery; add Related Work to checklist |
| `.kbz/skills/write-spec/SKILL.md` | Add cross-reference check at phase start |
| `.kbz/roles/reviewer-conformance.yaml` or skill | Add Related Work blocking check |
| `.agents/skills/kanbanzai-getting-started/SKILL.md` | Add corpus integrity check; add onboarding procedure |
| `.kbz/skills/orchestrate-review/SKILL.md` | Add classification step to Step 1; add checklist item |
| `.agents/skills/kanbanzai-documents/SKILL.md` | Promote classification section; reframe as active obligation; add to checklist |
| `.kbz/skills/implement-task/SKILL.md` | Add knowledge list to Phase 1; add confirm/flag to Phase 4; add checklist items |
| `.kbz/skills/orchestrate-development/SKILL.md` | Add knowledge list to Phase 1; add confirmation pass to Phase 6 |
| `.agents/skills/kanbanzai-agents/SKILL.md` | Add retrieval/confirmation to Task Lifecycle Checklist; update Context Assembly |

### 9.2 Document templates

| File | Change |
|------|--------|
| `work/templates/specification-prompt-template.md` | Add Required: Related Work section with Option A / Option B structure |
| Design template (if separate) | Same Related Work requirement |

### 9.3 Go code changes (server)

| Component | Change |
|-----------|--------|
| `internal/knowledge/store.go` | Add `LastAccessedAt` and `RecentUseCount`; increment on `List` and `Get` |
| `internal/knowledge/surfacer.go` | Add `sort: "recent"` option |
| `internal/docint/index.go` | Add `AccessCount` and `LastAccessedAt` to `DocumentIndex` and `SectionIndex` |
| `internal/service/intelligence.go` | Increment counters on `Outline`, `Guide`, `Section`, `Find`, `Search` |
| `internal/service/document.go` | Extend audit with "Most Accessed" table |

### 9.4 No changes

- `doc_intel` tool action surface
- `doc` tool action surface (beyond audit output)
- Context assembly pipeline in `internal/service/context.go`
- Classification schema or `classify` action

---

## 10. Phasing

### Phase 1: Skill and template changes (immediate — no code required)

Update all skill files and document templates. This is the highest-leverage change
because it takes effect immediately for every agent on every task from this point
forward. The design-stage mandate (Fix 1) and corpus integrity check (Fix 2) are both
purely textual changes.

Concurrently: run the session-start corpus integrity check, register the 9 unregistered
files via `doc import`, and begin the batch classification backlog starting with approved
specifications and designs.

Expected outcome within two active plans: new designs contain Related Work sections.
Classification coverage reaches >50% of approved specs and designs.

### Phase 2: Access instrumentation (1 week)

Implement `LastAccessedAt`, `RecentUseCount` on `KnowledgeEntry` and `AccessCount` on
`DocumentIndex`/`SectionIndex`. Update the audit report.

Expected outcome: adoption is continuously measurable. Regressions are visible without
requiring a full corpus audit.

### Phase 3: Validate and tighten (after Phase 1 is running)

Review two completed plans to assess:
- Are Related Work sections substantive or pro-forma?
- Is classification running at registration?
- Is `use_count` for knowledge entries increasing?

If any of these signals are negative, identify the gap (skill instruction vs. actual
behaviour) and tighten the relevant constraint — moving from advisory language to
checklist items, or from checklist items to stage-gate enforcement.

---

## 11. What This Design Is Not

1. **Not a replacement for the enhancement design.** FTS5 search, SQLite graph
   migration, and concept alias resolution are covered by `doc-intel-enhancement-design.md`.
   That design makes the system more capable. This design makes agents use it correctly.

2. **Not automated classification.** Classification remains agent-driven. This design
   adds mandatory trigger points and ensures the corpus is complete enough to classify.
   It does not replace agent judgement with a pipeline.

3. **Not a specification.** Field names, error messages, schema migrations, and test
   cases belong in the feature specification. This document defines what to build and why.

4. **Not a guarantee.** Skill changes and template requirements reduce the probability
   of agents skipping steps; they do not eliminate it. Instrumentation (Phase 2) and
   reviewer enforcement (Fix 1 §3.5) are the feedback mechanisms that detect and correct
   drift.

---

## 12. Success Criteria

The primary test is behavioural, not metric: **can a designer, beginning a new feature,
quickly and reliably discover what the existing corpus already knows about their problem
domain?**

Specific criteria:

1. **Design continuity.** New design documents contain a substantive Related Work section
   with at least one cross-reference to prior work, OR an explicit "no related work
   found" attestation with evidence of the search conducted.

2. **Concept search functional.** `doc_intel(action: "find", concept: "X")` returns
   results for at least three concepts introduced in the project. An architect can answer
   "what else in the corpus touches this concept?" without reading full documents.

3. **Corpus completeness.** `doc(action: "audit")` shows <5% unregistered files in
   configured document roots. A designer can trust that a negative corpus search result
   means "not addressed" rather than "not registered."

4. **Classification coverage.** >80% of approved specifications and designs have Layer 3
   classification. The concept registry contains entries. `find(role: "decision")` returns
   substantive results across the project corpus.

5. **Conformance enforcement.** The reviewer-conformance check for Related Work catches
   at least one design that omits meaningful cross-references in the first two plans
   after these changes ship. If zero designs are ever rejected on this criterion, the
   enforcement is not functioning.

6. **Knowledge feedback loop closed.** At least 30% of tier 2 knowledge entries reach
   `status: confirmed` following the first plan close-out that includes the confirmation
   pass. `recent_use_count` is non-zero for entries that appear in task context packets.

7. **Observable adoption.** The next Document Intelligence Usage Report can answer:
   "How often is the knowledge base accessed?", "Which document sections are most
   consulted?", and "Are designs being written with corpus awareness?" — from
   instrumentation data, without requiring a manual audit.