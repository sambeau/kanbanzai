# Report: P27 Doc-Intel Adoption — Retrospective and Sprint Planning Input

| Field       | Value                                                                 |
|-------------|-----------------------------------------------------------------------|
| Date        | 2026-04-23                                                            |
| Status      | Draft                                                                 |
| Author      | Post-P28 analysis                                                     |
| Based on    | P28 Doc-Intel Usage Summary (submitted); P27 design success criteria; |
|             | live instrumentation data from P27 Fix 5                              |
| Plan        | P27-doc-intel-adoption                                                |
| Covers      | FEAT-01KPTHB5Z1H7T, FEAT-01KPTHB61WPT0, FEAT-01KPTHB649DPK,         |
|             | FEAT-01KPTHB66Y8TM                                                    |

---

## Executive Summary

P27 shipped six fixes to make document intelligence a routine part of the Kanbanzai
workflow rather than an unused system. One fix is an unambiguous success. Two are
structurally correct but behaviourally incomplete. Two show no detectable impact yet.
One — the instrumentation fix — is now providing real data that makes this assessment
possible in the first place.

The single most important finding from P28 evidence is not in the P28 usage report:
**the concept registry is empty**. The corpus has 9,784 classified sections with full
role distributions (requirement, decision, rationale, etc.), but zero concept nodes and
zero TAGGED_WITH edges. This means `doc_intel(action: "find", concept: "X")` returns
nothing for every X — the primary "unknown unknowns" discovery capability that
justifies the entire system is not operational, and has never been operational. The
April 2026 classification pilot produced role classifications but not concept tagging.
This is the highest-priority item for the next sprint.

---

## Background: P27 Goals and Success Criteria

P27 was designed around five compounding failures diagnosed in the April 2026 usage
report:

1. The design stage had no corpus consultation mandate.
2. The corpus could not be trusted to be complete.
3. Layer 3 classification was absent from all mandatory workflows.
4. Knowledge entries were contributed but never consumed.
5. Adoption regressions were invisible (no instrumentation).

The P27 design defined seven measurable success criteria (§12). This report evaluates
each against P28 evidence and live instrumentation data.

---

## Instrumentation Data (P27 Fix 5)

The access instrumentation shipped in P27 is now collecting data. The following
figures come from the live index as of 2026-04-23.

### Document access counts

365 document index files exist. 364 have at least one role classification. Of those,
the most-accessed documents by `access_count` are:

| Access count | Document |
|---|---|
| 29 | `FEAT-01KP8T4HXPEAY/report-retrospectives` |
| 29 | `FEAT-01KMRX1HG8BAX/report-review-feat-01kmrx1hg8bax` |
| 15 | `FEAT-01KPVDDYSEK8P/report-review-feat-01kpvddysek8p-doc-intel-register-workflow` |
| 13 | `FEAT-01KN588PGJNM0/specification-30-review-skill-content` (×2 entries) |
| 10 | `FEAT-01KPVDDYX73WB/dev-plan-p28-decompose-dev-plan-registration` |
| 10 | `FEAT-01KPVDDYSEK8P/dev-plan-p28-doc-intel-register-workflow` |
| 9  | `FEAT-01KPVDDYVETV5/dev-plan-p28-plan-lifecycle-proposed-active` |

The access pattern is coherent: P28 dev-plans and specifications are being consulted
during implementation. Review reports and specifications from the review-skill work
(P28 context) are the most heavily accessed non-P28 documents. Section-level access
(`SectionAccess`) is populated for 11 documents, indicating targeted `section` calls
have been made rather than only top-level `outline` reads.

### Knowledge entry access

64 knowledge entries exist. All are in `status: contributed` — none have been
confirmed or flagged. The legacy `use_count` is 0 for all entries.

The P27 `recent_use_count` field is non-zero for all 64 entries (distribution: 22 to
52 accesses). All accesses are timestamped to two 2026-04-23 windows
(`03:43:42Z` and `10:30:41Z`), indicating bulk `knowledge list` calls from context
assembly during orchestration sessions — not explicit knowledge queries by implementing
agents. The `recent_use_count` data confirms that knowledge entries are being surfaced
in context packets, but it does not confirm that agents are reading and acting on them.

### Concept registry

The `concepts.yaml` file contains `concepts: []`. The SQLite graph database has
zero `TAGGED_WITH` edges. CONTAINS (9,784), LINKS_TO (2,437), and REFERENCES (1,676)
edges are populated from structural and entity-reference data, but no concept nodes
have ever been created.

Only one document index file — `FEAT-01KPNNYZ1ZSS6/specification-doc-intel-concept-aliases`
— contains `concepts_intro` data. Every other classified document was classified with
role and confidence only.

### FTS search

9,784 sections are indexed in the SQLite FTS5 store across 364 documents. Text search
(`doc_intel action: search`) is operational. Role-based retrieval (`doc_intel action:
find, role: "decision"`) is operational via the graph. Concept-based retrieval
(`doc_intel action: find, concept: "X"`) is non-functional for all X.

---

## P27 Success Criteria: Verdict by Criterion

### SC-1: Design Continuity — ✅ Working

> New design documents contain a substantive Related Work section with at least one
> cross-reference to prior work, OR an explicit "no related work found" attestation.

The P28 design document (`work/design/p28-doc-intel-polish-workflow-reliability.md`)
contains a two-table Related Work section citing four specific prior documents,
including P27 specs (`FEAT-01KPTHB61WPT0/specification-doc-intel-corpus-hygiene`,
`FEAT-01KPNNYYXQSYW/specification-doc-intel-batch-classification`). It documents
four constraining decisions with source citations and section references, and
explicitly surfaces an open question from prior work. This is exactly the structure
Fix 1 mandated.

The P27 write-design Phase 0 procedure was followed. Phase 0 produced evidence of
prior-work discovery. The blocking conformance check was in a position to enforce
this; the P28 design satisfied it without triggering a rejection.

This is the primary success of P27.

### SC-2: Concept Search Functional — ❌ Not operational

> `doc_intel(action: "find", concept: "X")` returns results for at least three
> concepts introduced in the project.

Zero concept nodes exist. `find(concept: "X")` returns nothing for any X.

The April 2026 pilot classified 339 documents (and subsequent classification has
reached 364 documents) — but every classification operation specified `role` and
`confidence` only. The `concepts_intro` and `concepts_used` fields, which are
required to populate concept nodes and TAGGED_WITH edges, were never populated
in any classification pass.

The result is a system with rich structural classification (section roles are known
for 9,784 sections) but no semantic graph. Role-based queries work. Concept-based
queries — the primary mechanism for "unknown unknowns" discovery — do not.

This is the most significant unmet success criterion. It is also the fastest to
fix: a targeted reclassification pass over ~50 high-value specifications and
designs, this time explicitly populating `concepts_intro`, would make the concept
registry substantive within a single sprint.

### SC-3: Corpus Completeness — ⚠️ Partially met

> `doc(action: "audit")` shows <5% unregistered files in configured document roots.

365 document index files exist against 368 registered document state records —
a small discrepancy. However, this count is misleading. A large number of
registered state records (approximately 200, from P3–P22) have no corresponding
index file at all. These documents were registered before the bulk classification
pilot, never had `doc_intel guide` run against them, and are therefore invisible
to all doc_intel queries despite being registered.

The session-start corpus integrity check (Fix 2) addresses unregistered files but
not documents that are registered-but-unindexed. These pre-pilot documents represent
a second class of "invisible" corpus content that the integrity check does not cover.

The P28 report correctly identifies that P28 documents are pending classification,
but the larger and older problem — documents that have state records but no semantic
index — is more significant.

### SC-4: Classification Coverage — ⚠️ Partially met

> >80% of approved specifications and designs have Layer 3 classification.

364 of 365 index files have at least one role classification. On the face of it, this
exceeds the 80% threshold. However, two problems temper this reading.

**Depth variance**: P27 specs have 20–26 classified sections each. P28 specs have
exactly 5 each — uniform across all six, always the same structural paths (§1.2,
§1.2.1, §1.2.2, §1.3, §1.4) with the same role pattern (requirement/constraint).
This is a minimal structural classification, not a substantive semantic one. The P28
classify-on-register improvement shipped as part of P28 itself, and registration-time
classification is producing shallow rather than deep results.

**The pre-pilot backlog**: ~200 registered documents have no index files. These are
entirely unclassified in the operational sense — they are invisible to all doc_intel
queries.

**No concept population**: Role coverage (requirement, decision, etc.) is high, but
concept coverage is zero for all documents. Classification without concept tagging
is structurally useful (role-based queries work) but does not deliver the core
design-continuity capability.

### SC-5: Conformance Enforcement — ⚠️ Insufficient data

> The reviewer-conformance check catches at least one design in the first two plans.

P28 shipped one design document (`p28-doc-intel-polish-workflow-reliability.md`),
which passed the Related Work check. Whether the conformance reviewer explicitly
ran the blocking check or whether the design incidentally satisfied it cannot be
determined from index data alone. No rejection on Related Work grounds has been
recorded. This criterion requires at least two design documents to be reviewable;
P28 provided one. Assessment deferred to P29.

### SC-6: Knowledge Feedback Loop — ❌ Not met

> At least 30% of tier 2 knowledge entries reach `confirmed` status following the
> first plan close-out that includes the confirmation pass.

All 64 knowledge entries remain `contributed`. Zero confirmed. Zero flagged. Zero
retired. The close-out confirmation pass mandated in orchestrate-development Phase 6
Close-Out (Fix 6) has not been run.

The knowledge retrieval mandate (Fix 4) added `knowledge list` calls to
implement-task Phase 1. The `recent_use_count` data confirms entries are being
surfaced via context assembly. But P28 was a Go implementation sprint; sub-agents
implementing tasks did not run explicit `knowledge confirm` calls. The Phase 4
confirm/flag obligation in implement-task and the Phase 6 curation pass in
orchestrate-development have not yet been exercised in a complete plan cycle.

35 entries are tier 2; 29 are tier 3. The tier 2 entries — contributed knowledge
from substantive implementation work — are the primary target for the first
confirmation pass.

### SC-7: Observable Adoption — ✅ Now possible

> The next usage report can answer adoption questions from instrumentation data,
> without requiring a manual audit.

This is met. This report is itself evidence: it was produced from instrumentation
data without conducting a manual corpus audit. The Most Accessed Documents table
(available via `doc audit`), `knowledge list sort:recent`, and per-section access
counts provide continuous adoption signals.

The one gap: `recent_use_count` counts context-assembly list calls alongside
explicit query calls. A knowledge entry that appears in a context packet but is
never read by the agent is indistinguishable from one that is read and used.
This limits the signal's precision for distinguishing passive surfacing from active
consultation.

---

## Root Cause Analysis of Remaining Gaps

### Why is the concept registry empty?

The `classify` action schema has always included `concepts_intro` and
`concepts_used` fields. They are optional. During the April 2026 pilot, the
classification procedure focused on assigning `role` and `confidence` per section —
the minimum required for the `guide` taxonomy to be useful and for role-based
queries to work. No classification session instructed agents to populate
`concepts_intro`.

The skill files do not explicitly require concept tagging. The `guide` response
(pre-P28) did not include a concept taxonomy or suggestion list. With no prompt
to populate `concepts_intro` and no example of what a well-populated concept entry
looks like, agents defaulted to the minimum required fields.

The P28 sprint added taxonomy hints to the `guide` response — but the taxonomy
covers roles, not concepts. The concept-tagging gap is therefore not addressed by
any P27 or P28 change and will persist until explicitly targeted.

### Why is classification shallow at registration time?

The P28 classify-on-register workflow produces 5-section classifications for every
specification because the specification template has a uniform five-section structure
in its opening pages. An agent classifying at registration time reads the document
it just wrote — but classification from memory of a just-written document, without
re-reading sections, produces section-count-limited results. Thorough classification
requires reading each section with the classify call in mind. At registration, the
agent's attention is on committing and moving to the next task, not on reading back
what it just wrote.

The `classification_nudge` in the register response is a text string. It is seen
and deferred. This confirms the P27 design's own prediction (§11): "Skill changes
and template requirements reduce the probability of agents skipping steps; they do
not eliminate it."

### Why are knowledge entries unconfirmed?

The knowledge confirmation mandate (Fix 4) was added to implement-task Phase 4 and
orchestrate-development Phase 6. P28 was a six-feature Go implementation sprint.
The sub-agents who implemented those features are the agents who would run Phase 4
confirm/flag calls — but:

1. Sub-agents complete a task and stop. They do not have visibility of which
   knowledge entries from their context packet they actually used versus ignored.
2. The orchestrator runs the Phase 6 close-out, but P28 has not been formally
   closed out. No plan close-out confirmation pass has been run.

The confirmation mechanism is correctly placed in the orchestrate-development Phase
6 close-out, not in sub-agent Phase 4. The orchestrator has the full view of which
entries were surfaced across all sub-agents in the plan. The fix is to run the
close-out — not to change the skill.

---

## What Should Change for the Next Sprint

### Priority 1: Backfill the concept registry (1–2 days)

This is the highest-leverage action available. Classify the top 50 approved
specifications and designs from P15–P27 with explicit `concepts_intro` populated
for each significant section. Three to five concepts per section, drawn from the
section's vocabulary, is sufficient to make the concept graph substantive.

A classification with `concepts_intro` should look like:

```yaml
section_path: "2.1"
role: decision
confidence: high
concepts_intro:
  - name: stage-gate
    aliases:
      - lifecycle gate
      - workflow gate
  - name: override
```

After this pass, `doc_intel(action: "find", concept: "stage-gate")` will return
every specification section that introduces or discusses that concept — the
"unknown unknowns" query that is the core value proposition of the system.

The 50 highest-value documents for this pass are the approved specifications from
P20–P27 (the most recent and most referenced design history).

### Priority 2: Add concept guidance to the classify skill and guide response

The `guide` response now returns a `taxonomy` block (P28 change) covering roles.
It should also return a `concepts_suggested` list derived from entity references
found in the section graph: concept names extracted from headings and `REFERENCES`
edges that already exist.

The `kanbanzai-documents` Classification (Layer 3) section should include an
explicit example of a `concepts_intro`-populated classification call, and the
skill should state: "For each section that introduces a new concept, domain term,
or design pattern, populate `concepts_intro` with 2–5 normalised concept names."
Without this instruction, agents will continue to omit it.

### Priority 3: Run the P28 knowledge confirmation close-out

Call `knowledge(action: "list", status: "contributed", tier: 2)` and review the
35 tier 2 entries. For each:

- Entries that described accurate patterns (e.g. `mcp-thin-adapter-pattern`,
  `git-commit-message-format`, `error-handling-conventions`) → confirm.
- Entries that described problems now resolved → retire.
- Entries that proved inaccurate in P28 work → flag.

This is the first execution of the Fix 6 orchestrate-development Phase 6 close-out
procedure. Running it once establishes the pattern and begins converting the
knowledge base from an append-only log into a confidence-weighted reference.

### Priority 4: Programmatic enforcement for design-stage classification

Fix 3 framed the `classification_nudge` as "MUST follow before moving to the next
task." In practice, agents defer it. The P27 design anticipated this (§10 Phase 3)
and noted that skill-based enforcement should be validated and tightened if
classification coverage stalls.

It is stalling. P28 specs are classified with 5 sections each. The next sprint
should escalate from skill mandate to stage gate: a design document or specification
cannot be approved without at least one classified section with a concept introduced.
This is enforceable by checking the document's index file at `doc approve` time —
a small server-side change that requires no new tool actions.

### Priority 5: Index the pre-pilot backlog

~200 registered documents from P3–P22 have no index files. These are invisible to
all doc_intel queries. Running `doc_intel(action: "guide")` for each would build
their structural outlines, and a targeted batch classification pass on their
specifications and designs would extend the queryable corpus back to project
inception.

This is lower priority than concept backfill because the most relevant prior work
for current design decisions (P15–P27) is already indexed. But a designer working
on a problem first addressed in P8 will find nothing in the corpus without this.

---

## Metrics Available for the Next Sprint Retrospective

P27 Fix 5 makes the following metrics continuously available for P29:

| Question | How to answer |
|---|---|
| How many documents have concept nodes? | Count `TAGGED_WITH` edges in `graph.yaml` or SQLite |
| Which documents are most consulted? | `doc(action: "audit")` Most Accessed table |
| Are knowledge entries being used? | `knowledge(action: "list", sort: "recent")` — non-zero `recent_use_count` |
| Is the confirmation loop running? | Count entries with `status: confirmed` |
| Is classification shallow or deep? | Mean classified sections per document type |
| Are Related Work sections substantive? | Conformance reviewer rejection rate (manual) |

The precision gap identified above — `recent_use_count` counts context-assembly
list calls alongside intentional knowledge queries — can be partially addressed
by also tracking `use_count` (incremented only on explicit `knowledge get` calls).
If `recent_use_count` is rising but `use_count` remains zero, knowledge is being
surfaced in context but not explicitly consulted. If both rise together, agents
are actively querying the knowledge base.

---

## Summary Table

| P27 Fix | What it delivered | What's missing | Next action |
|---|---|---|---|
| Fix 1: Design-stage mandate | Phase 0 executed in P28; Related Work substantive | One data point only; conformance enforcement untested at scale | Continue; assess P29 design |
| Fix 2: Corpus completeness | Session-start check in skill | ~200 pre-pilot docs have no index | Batch `guide` + classify P3–P22 specs |
| Fix 3: Classification as obligation | Shallow classify-on-register working | No concept tagging; nudge deferred in practice | Concept backfill sprint; enforce in stage gate |
| Fix 4: Knowledge retrieval mandate | Entries surfaced in context packets | No explicit queries; no confirm/flag | Run P28 knowledge close-out |
| Fix 5: Instrumentation | Fully operational; data in this report | `recent_use_count` conflates passive/active | Track `use_count` alongside `recent_use_count` |
| Fix 6: Knowledge curation | Mandate in skills | Close-out not yet run | Run Phase 6 close-out for P28 |

The one-line summary: **the enforcement chain for design continuity works. The semantic
graph it relies on does not exist yet.** Fix that, and the system delivers its
core promise.