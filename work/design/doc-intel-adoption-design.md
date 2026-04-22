# Design: Document Intelligence Adoption Fixes

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

The Document Intelligence Usage Report (April 2026) identified two critical failures in
how agents interact with the `doc_intel` and `knowledge` systems:

1. **Layer 3 AI classification has never run.** 332 of 334 registered documents have no
   semantic classification. Concept search returns zero results. The richest query
   capabilities — concept-based retrieval, rationale extraction, role-based search
   across non-standard headings — are entirely unavailable.

2. **Knowledge entries are never retrieved via the API.** All 59 knowledge entries have
   `use_count: 0`. Agents are writing knowledge but not reading it. The closed loop —
   contribute on task completion, consume on the next task — is broken at the consumption
   end.

A third supporting issue compounds both: there is **no instrumentation** on how the
system is used, making it impossible to detect regressions in adoption or to identify
which documents and knowledge entries are most valuable.

These are **adoption failures**, not technical failures. The infrastructure exists. The
batch classification protocol is defined. Knowledge entries have been contributed. The
problem is that skills do not mandate the behaviours needed to use the system, so agents
skip those steps without realising they are skipping them.

This document complements `doc-intel-enhancement-design.md`, which addresses the
technical infrastructure (FTS5 search, SQLite graph, concept alias resolution). That
design makes the system more capable. This design makes agents actually use it.

### 1.1 Scope

**In scope:**

- Skill changes to mandate knowledge retrieval before task start
- Skill changes to mandate knowledge confirmation after task completion
- Classification trigger added to the review pipeline prerequisites
- Repositioning of the batch classification protocol in `kanbanzai-documents` skill
- Access instrumentation for `doc_intel` and `knowledge` tools
- Plan close-out knowledge confirmation step in `orchestrate-development`

**Out of scope:**

- Technical infrastructure changes (FTS5, SQLite, concept aliases — see enhancement design)
- Changes to the `doc_intel` tool's action surface beyond access logging
- Changes to how context packets are assembled by `next`
- Automated classification (classification remains agent-driven)
- Embedding-based or vector search (evaluate after enhancement design)

---

## 2. Root Cause Analysis

Understanding why adoption failed is required to design fixes that hold.

### 2.1 Why knowledge is not consumed

The `kanbanzai-agents` skill mandates knowledge **contribution** (`finish` retrospective
signals, `knowledge contribute`) but does not mandate knowledge **consumption**
(`knowledge list`). The checklist in `implement-task` and the Phase 1 procedure in
`implement-task` are both silent on knowledge retrieval.

Context assembly via `next(id)` surfaces knowledge entries automatically in the context
packet. But this is passive delivery — the context packet is bounded and may not surface
entries whose tags or scope do not match the task's file prefix. An agent doing active
work (e.g. a cross-cutting concern, a refactor, a review) will not receive relevant
knowledge unless it explicitly queries for it.

Furthermore, **`use_count` is only incremented by explicit API calls** (`knowledge list`,
`knowledge get`), not by context packet inclusion. So even if entries were surfaced in
context packets, the system cannot distinguish between "agent saw this and acted on it"
and "agent received this passively and ignored it".

The skill-authoring research identifies the root mechanic: advisory instructions ("you
may call `knowledge list`") do not produce reliable behaviour. Only checklist items with
concrete tool call sequences produce consistent compliance (Anthropic, Effective Context
Engineering, 2025; MetaGPT, ICLR 2024). The fix is to make knowledge retrieval a
**required checklist step with an explicit tool call pattern**, not an optional
recommendation.

### 2.2 Why Layer 3 classification is not running

The batch classification protocol exists in the `kanbanzai-documents` skill and the
`doc-intel-enhancement-design.md`. The problem is structural:

- The batch classification section appears at the **end** of the `kanbanzai-documents`
  skill. Research on U-shaped attention in transformers (Liu et al., "Lost in the Middle",
  2024) predicts that content at the end of a long skill receives less attention than
  content at the beginning. A protocol placed in the final section of a long skill will
  be read less frequently than one placed in the body.

- There is **no trigger** in the feature lifecycle. Classification has no mandatory
  integration point — it is a discretionary task that requires an agent to
  spontaneously decide to run it. Discretionary tasks with no enforcement are skipped
  under time pressure.

- The `orchestrate-review` skill's Step 1 (verify prerequisites) checks for a spec and
  confirms the feature is in `reviewing` status, but does **not** trigger classification
  of the feature's documents. This is the highest-leverage integration point: a reviewer
  agent needs semantically rich outlines, and the feature's documents are known at review
  time.

### 2.3 Why instrumentation is absent

Access instrumentation was never designed. The usage report cannot answer "which document
sections are most consulted" or "how often is the knowledge base accessed" because no
counters exist. This is not a retroactive failure — it was an omission in the original
system design. Without instrumentation, future regressions in adoption will go undetected
until a usage audit catches them.

---

## 3. Fix 1: Mandatory Knowledge Retrieval in Skills

### 3.1 Problem

Agents contribute knowledge but never read it. No skill mandates consumption.

### 3.2 Solution

Add an explicit, required knowledge retrieval step to three skill files:

1. `implement-task/SKILL.md` — pre-task Phase 1
2. `orchestrate-development/SKILL.md` — Phase 1 (Read the Dev-Plan)
3. `.agents/skills/kanbanzai-agents/SKILL.md` — Task Lifecycle Checklist

The step follows a fixed pattern: query `knowledge list` with relevant tags, then review
the results before beginning any implementation work. This is a checklist item, not an
advisory note, so it cannot be silently skipped.

### 3.3 Changes to `implement-task/SKILL.md`

**Checklist addition** (after "Read the context packet" step):

```
- [ ] Called `knowledge list` with tags relevant to this task's domain — reviewed entries before writing any code
```

**Phase 1 addition** (after step 1, claim the task):

```
1a. Call `knowledge(action: "list", tags: ["<domain>", "<feature-area>"])` using tags
    derived from the task's feature area (e.g. "storage", "cli", "review", "doc-intel").
    Review all returned entries before proceeding. If an entry describes a known pitfall
    or anti-pattern relevant to this task, note it.
    BECAUSE: Knowledge entries record hard-won discoveries from previous tasks —
    pitfalls, workarounds, policy decisions. An agent that skips this step re-discovers
    the same problems and makes the same mistakes, wasting cycles that the knowledge
    system was built to prevent.
```

**Post-task Phase 4 addition** (in the `finish` call guidance):

```
When calling finish, also call:
  knowledge(action: "confirm", id: "<KE-id>")
for any knowledge entry that proved accurate and useful during this task.
  knowledge(action: "flag", id: "<KE-id>")
for any entry that was inaccurate or misleading.
BECAUSE: Confirmation and flagging are how the knowledge base self-curates.
Entries that are consistently confirmed gain higher confidence scores and are
prioritised in future context packets. Entries that are consistently flagged
are automatically retired.
```

**BAD example addition:**

```
BAD: Implement without consulting knowledge
  Agent claims TASK-212 (add FTS5 search to doc_intel).
  Proceeds directly to implementation.
  Spends 40 minutes debugging a SQLite FTS5 gotcha with the tokeniser.

WHY BAD: Knowledge entry KE-01KPNN describes this exact issue with a
workaround, contributed by a previous agent during P23. The knowledge was
available; the agent never checked.

GOOD: Consult knowledge before implementing
  Agent claims TASK-212.
  Calls knowledge(action: "list", tags: ["doc-intel", "sqlite", "storage"]).
  Finds KE-01KPNN: "SQLite FTS5 with the unicode61 tokeniser requires
  explicit column filters when queries include punctuation..."
  Notes the workaround before opening any source files.
  Implementation avoids the known gotcha entirely.
```

### 3.4 Changes to `orchestrate-development/SKILL.md`

**Phase 1 addition** (after reading the dev-plan):

```
1a. Before dispatching any tasks, call:
    knowledge(action: "list", tags: ["<feature-area>"], status: "confirmed")
    and review the confirmed entries relevant to this feature's domain.
    BECAUSE: Confirmed knowledge entries record architectural patterns,
    anti-patterns, and policy decisions that apply across all tasks in the
    feature. Distributing this knowledge to sub-agents via their handoff
    prompts prevents repeated mistakes across the task graph.
1b. Surface relevant entries to sub-agents by including them in the
    `instructions` parameter of `handoff(task_id, instructions: "...")`.
```

**Phase 6 Close-Out addition** (after step 4, record completion summary):

```
4a. Confirm tier 2 knowledge entries contributed during this feature's
    development. Call `knowledge(action: "list", status: "contributed", tier: 2)`
    and for each entry that proved accurate, call `knowledge(action: "confirm")`.
    BECAUSE: Tier 2 entries start as "contributed" and only become trusted
    references after confirmation. An uncurated knowledge base accumulates noise
    alongside signal; the confirmation pass at feature close-out is the primary
    curation mechanism.
```

### 3.5 Changes to `kanbanzai-agents/SKILL.md`

**Task Lifecycle Checklist addition:**

```
- [ ] Called `knowledge list` with domain-relevant tags before starting implementation
- [ ] Confirmed accurate knowledge entries and flagged inaccurate ones after completing the task
```

**Context Assembly section update:**

Replace the current passive description of knowledge surfacing with an active retrieval
requirement:

```
After calling `next(id)` to claim a task, actively query the knowledge base:
  knowledge(action: "list", tags: ["<domain>", "<feature-area>"])
The context packet surfaces the highest-ranked matching entries automatically,
but active querying finds entries the automatic matching may miss — especially
for cross-cutting concerns or tasks whose file scope spans multiple domains.

After completing the task:
- Call knowledge(action: "confirm") on entries that proved accurate and useful.
- Call knowledge(action: "flag") on entries that were inaccurate or misleading.
These signals are how the knowledge base self-curates. Skipping them degrades
the base for every future agent.
```

---

## 4. Fix 2: Classification Trigger in the Review Pipeline

### 4.1 Problem

Layer 3 classification has no mandatory trigger in the feature lifecycle. The batch
protocol exists but is discretionary, placed at the end of a long skill, and rarely
executed.

### 4.2 Solution

Two changes:

1. Add a mandatory classification step to `orchestrate-review/SKILL.md` Step 1
   (verify prerequisites). Before dispatching review sub-agents, classify the feature's
   documents. This ensures reviewer agents operate on semantically enriched outlines.

2. Promote the batch classification protocol within `kanbanzai-documents/SKILL.md` from
   its current end-of-file position to a more prominent location, and reframe it as an
   **active obligation** rather than a reference section.

### 4.3 Changes to `orchestrate-review/SKILL.md`

**Step 1 addition** (after confirming spec exists, before step 2 decompose):

```
1b. Classify any unclassified documents owned by the feature.
    i.   Call doc_intel(action: "pending") and filter for documents whose
         owner matches this feature's ID.
    ii.  For each unclassified feature document, run the classification
         protocol:
         a. doc_intel(action: "guide", id: "DOC-xxx") — get outline + content hash
         b. Read any sections needed to assign roles
         c. doc_intel(action: "classify", id: "DOC-xxx", content_hash: "...", ...)
    iii. Classification is not a blocking prerequisite — if context budget is
         exhausted before all documents are classified, proceed with reviewing.
         Prioritise: specification first, design second, dev-plan third.
    BECAUSE: Reviewer sub-agents use doc_intel to navigate documents. Layer 3
    classification enables role-based search (find all requirements, find all
    decisions) and produces richer guides. A reviewer working on an unclassified
    corpus falls back to structural navigation only, missing the semantic layer
    that identifies rationale, constraints, and design decisions.
```

**Checklist addition:**

```
- [ ] Classified unclassified feature documents (or confirmed context budget insufficient)
```

### 4.4 Changes to `kanbanzai-documents/SKILL.md`

**Move** the "Batch Classification Protocol" section from its current end-of-file
position to immediately after the "Registration" section — placing it where agents
encounter it before they have mentally committed to finishing the registration workflow.

**Rename** the section from "Batch Classification Protocol" to
**"Classification (Layer 3)"** and reframe its opening to be action-oriented:

```
## Classification (Layer 3)

**After registering a document, classify it immediately if you have the
document content in context.** Do not defer classification to a batch run.

The `doc register` response includes a `classification_nudge` indicating which
`doc_intel` calls to make. Follow it before moving to the next task.

BECAUSE: Layer 3 classification is what enables concept search, role-based
retrieval, and semantic guides. Documents classified at registration time
remain classified as long as content does not change. Documents deferred to
batch runs accumulate as a growing backlog that is never fully cleared — as
the corpus grows, the backlog grows faster than batch runs can clear it.
```

**Add to the Document Creation Checklist:**

```
- [ ] Classified the document with doc_intel(action: "classify") if content was in context
```

---

## 5. Fix 3: Access Instrumentation

### 5.1 Problem

The system cannot answer:
- How often is the knowledge base accessed?
- Which document sections are most consulted?
- Which knowledge entries are actually useful (vs. merely contributed)?
- Is adoption improving or degrading after these fixes are deployed?

Without instrumentation, these questions are permanently unanswerable, and future
adoption failures will require another full audit to detect.

### 5.2 Solution

Add lightweight access counters and timestamps to both systems.

### 5.3 Knowledge base instrumentation

**New fields on `KnowledgeEntry`:**

| Field               | Type      | Description                                        |
|---------------------|-----------|----------------------------------------------------|
| `last_accessed_at`  | timestamp | Set on every `knowledge list` or `knowledge get` call that returns this entry |
| `recent_use_count`  | int       | Rolling 30-day window count of accesses. Separate from `use_count` (all-time). |

**Behaviour:**

- `knowledge(action: "list", ...)` increments `recent_use_count` and sets
  `last_accessed_at` for every entry included in the response.
- `knowledge(action: "get", id: "KE-xxx")` increments `recent_use_count` and sets
  `last_accessed_at` for the retrieved entry.
- `recent_use_count` is decremented by a daily TTL sweep that removes accesses older
  than 30 days from the rolling window. The simplest implementation is to store a list
  of access timestamps and compute the count on read.
- `knowledge(action: "list")` default output surfaces `recent_use_count` alongside
  `use_count` so agents can distinguish active entries from stale ones.

**New `knowledge(action: "list")` sort option:**

Add `sort: "recent"` to surface entries accessed most frequently in the last 30 days.
This is the recommended sort order for pre-task knowledge queries, because it surfaces
entries that recent agents found useful.

### 5.4 Document intelligence instrumentation

**New fields on `DocumentIndex` (per document):**

| Field               | Type      | Description                                        |
|---------------------|-----------|----------------------------------------------------|
| `access_count`      | int       | Cumulative count of `outline`, `section`, `guide`, `find`, `search` calls |
| `last_accessed_at`  | timestamp | Timestamp of the most recent access to this document |

**New fields on `SectionIndex` (per section, stored in the document index file):**

| Field               | Type      | Description                                        |
|---------------------|-----------|----------------------------------------------------|
| `access_count`      | int       | Count of `section` calls targeting this section path |
| `last_accessed_at`  | timestamp | Timestamp of the most recent `section` read        |

**Behaviour:**

- `doc_intel(action: "outline", id: "DOC-xxx")` increments the document-level
  `access_count` and updates `last_accessed_at`.
- `doc_intel(action: "guide", id: "DOC-xxx")` increments document-level counter.
- `doc_intel(action: "section", id: "DOC-xxx", section_path: "3.2")` increments
  both the document-level counter and the section-level counter for path `3.2`.
- `doc_intel(action: "find", ...)` and `doc_intel(action: "search", ...)` increment
  counters for every document that appears in the result set.
- Counter updates are written lazily — they are not committed to disk on every call.
  A flush every N calls or on process shutdown is acceptable. Counts are approximate.

**Storage:** Document-level counters live in the per-document index file
(`.kbz/index/docs/<doc-id>.yaml`). Section-level counters live in the same file under
each section entry. If SQLite storage is adopted per the enhancement design, counters
move to the SQLite `sections` table.

**New `doc(action: "audit")` output field:**

Extend the audit report with a "Most Accessed Documents" table showing the top 10
documents by `access_count` in the last 30 days. This makes the instrumentation
actionable — admins can see what the corpus is actually used for.

### 5.5 What instrumentation does not track

- Individual agent identity (no per-agent attribution — privacy and simplicity)
- Query content (what search terms were used — too verbose)
- Latency per call (out of scope for this design)

---

## 6. Fix 4: Plan Close-Out Knowledge Confirmation

### 6.1 Problem

All 59 knowledge entries have `use_count: 0` and `status: contributed`. No agent has
ever called `knowledge(action: "confirm")`. The knowledge base is an append-only log
rather than a curated, confidence-weighted reference.

### 6.2 Solution

Add a mandatory knowledge confirmation pass to `orchestrate-development/SKILL.md`
Phase 6 (Close-Out) and to the plan review checklist in `review-plan/SKILL.md`.

This was described in section 3.4 above for the feature close-out. The plan close-out
equivalent:

### 6.3 Changes to plan review workflow

When a plan transitions to complete, the orchestrator should:

1. Call `knowledge(action: "list", status: "contributed", tier: 2)` to retrieve all
   unconfirmed tier 2 entries.
2. For each entry whose topic is relevant to this plan's work, review the content.
3. Call `knowledge(action: "confirm")` for accurate entries.
4. Call `knowledge(action: "flag")` for entries that proved inaccurate.
5. Call `knowledge(action: "retire", reason: "superseded by ...")` for entries that are
   no longer relevant due to architectural changes delivered in this plan.

This pass transforms the knowledge base from a write-only retrospective log into a
curated reference that accumulates confidence over time.

### 6.4 Tier 3 entries

Tier 3 entries (session-level, 30-day TTL) do not require a confirmation pass at plan
close-out. They are self-pruning. An agent that finds a tier 3 entry useful should
contribute a tier 2 entry with the same content — `knowledge(action: "promote")` does
this in one call.

---

## 7. What Changes

### 7.1 Skill files

| File | Change |
|------|--------|
| `.kbz/skills/implement-task/SKILL.md` | Add `knowledge list` step to Phase 1 and checklist; add `knowledge confirm/flag` to Phase 4; add BAD/GOOD example for knowledge usage |
| `.kbz/skills/orchestrate-development/SKILL.md` | Add `knowledge list` to Phase 1; add knowledge confirmation pass to Phase 6 Close-Out |
| `.agents/skills/kanbanzai-agents/SKILL.md` | Add knowledge retrieval and confirmation to Task Lifecycle Checklist; update Context Assembly section |
| `.kbz/skills/orchestrate-review/SKILL.md` | Add classification step to Step 1 prerequisites and checklist |
| `.agents/skills/kanbanzai-documents/SKILL.md` | Move "Batch Classification Protocol" section earlier (after Registration); rename and reframe as an active obligation; add classification step to Document Creation Checklist |

### 7.2 Go code changes (server)

| Component | Change |
|-----------|--------|
| `internal/knowledge/store.go` | Add `LastAccessedAt` and `RecentUseCount` fields to `KnowledgeEntry`; increment on `List` and `Get` |
| `internal/knowledge/surfacer.go` | Add `sort: "recent"` option to `List` |
| `internal/docint/index.go` | Add `AccessCount` and `LastAccessedAt` to `DocumentIndex` and `SectionIndex` |
| `internal/service/intelligence.go` | Increment document and section counters on `Outline`, `Guide`, `Section`, `Find`, `Search` |
| `internal/service/document.go` | Extend audit report with "Most Accessed" table |

### 7.3 No changes

- The `doc_intel` tool's action surface (no new actions required by this design)
- The `doc` tool's action surface
- The context assembly pipeline in `internal/service/context.go`
- The classification schema or `classify` action

---

## 8. Phasing

### Phase 1: Skill changes (immediate — no code required)

Update the five skill files listed in §7.1. This is the highest-leverage change
because it directly addresses the adoption failures for agents acting on any task
from this point forward. No new code needs to ship before these changes take effect.

Expected outcome: agents begin calling `knowledge list` before tasks and `knowledge
confirm/flag` after tasks. Classification begins to accumulate at review time.

### Phase 2: Access instrumentation (1 week)

Implement the `LastAccessedAt` and `RecentUseCount` fields on `KnowledgeEntry` and
the `AccessCount` fields on `DocumentIndex` and `SectionIndex`. Update the audit
report.

Expected outcome: the system can answer "how often is the knowledge base accessed"
and "which document sections are most consulted". Adoption regressions become visible
without requiring a full corpus audit.

### Phase 3: Close-out confirmation integration (after Phase 1 is running)

After skills are updated and agents begin confirming knowledge entries, validate that
`use_count` and `status: confirmed` are increasing. If not, this indicates a gap
between the skill mandate and actual agent behaviour — investigate and tighten the
constraint.

The `knowledge(action: "prune")` command can be run at plan boundaries to retire
tier 3 entries past their TTL and compact the knowledge base.

---

## 9. What This Design Is Not

1. **Not a replacement for the enhancement design.** FTS5 search, SQLite graph migration,
   and concept alias resolution are covered by `doc-intel-enhancement-design.md`. This
   document is about adoption, not capability.

2. **Not automated classification.** Classification remains agent-driven. This design
   adds the mandatory trigger points; it does not replace agent judgement with a pipeline.

3. **Not a specification.** Exact field names, error messages, schema migrations, and
   test cases belong in the feature specification. This document defines what to build
   and why.

4. **Not a guarantee.** Skill changes reduce the probability of agents skipping steps;
   they do not eliminate it. The instrumentation in Phase 2 is the feedback mechanism
   that detects when adoption regresses despite the skill mandates.

---

## 10. Success Criteria

After all three phases are complete, the following should be true:

1. `knowledge(action: "list")` returns entries with non-zero `recent_use_count` for
   the most frequently used tier 2 entries.
2. At least 50% of tier 2 knowledge entries have `status: confirmed` following a plan
   close-out that includes the confirmation pass.
3. Layer 3 classification coverage reaches ≥50% of approved specifications and designs
   within two active development plans.
4. `doc(action: "audit")` surfaces a non-empty "Most Accessed Documents" table showing
   the corpus sections agents are relying on most.
5. A reviewer agent can call `doc_intel(action: "find", role: "decision")` on a feature's
   design document and receive classified decision sections, not zero results.
6. The next Document Intelligence Usage Report can answer: "How often is the knowledge
   base accessed?" and "Which document sections are most consulted?"