| Field  | Value                                          |
|--------|------------------------------------------------|
| Date   | 2026-04-23                                     |
| Status | Draft                                          |
| Author | sambeau                                        |
| Plan   | P32-doc-intel-classification-pipeline-hardening |

## Related Work

### Prior documents consulted

- `work/research/doc-intel-recurring-issues-investigation.md` — root-cause cluster analysis that motivates this plan; Clusters C1 and C2 are the direct inputs
- `work/design/doc-intel-adoption-design.md` — P27 design that introduced the advisory nudge pattern this plan supersedes
- `work/design/p28-doc-intel-polish-workflow-reliability.md` — P28 enhancements to guide/pending responses; this plan extends rather than replaces that work

### Constraining decisions

- The `doc(action: "register")` response already includes a `classification_nudge` object with `message`, `content_hash`, and `outline` (added in P28). This plan builds on that structure.
- The `doc_intel(action: "guide")` response already includes `suggested_classifications` for heading-deterministic sections (Acceptance Criteria, Alternatives Considered). The heading-based concept suggestion extends this existing mechanism.
- The `doc_intel(action: "classify")` action accepts `classifications` as a JSON array of objects with `section_path`, `role`, `confidence`, and optional `concepts_intro` / `concepts_used`. This interface is fixed.
- Enforcement at `doc approve` time (not `classify` time) was established as the correct enforcement point by RG-2 in the investigation. The rationale: enforcing at classify time would break batch backlog runs where concept tagging is deferred by design.

### Open questions resolved by the investigation

- **RG-1 (heading-based vs entity-ref-based concept extraction):** Experiment confirmed ~56% of concepts are derivable from section headings; entity refs contribute nothing useful. Decision: use heading-based extraction only.
- **RG-2 (enforcement point):** Enforce at `doc approve` for specification, design, and dev-plan types. Do not enforce at classify time.

## Problem and Motivation

The doc-intel classification pipeline depends on agents completing three enrichment steps voluntarily: (1) calling `doc_intel guide` after registration, (2) calling `doc_intel classify` with role assignments, and (3) populating `concepts_intro` in at least one classified section. Retrospective evidence shows all three steps are regularly skipped or deferred:

- No concepts were tagged in the concept registry after the April 2026 pilot despite explicit skill guidance.
- Classification sessions consistently omit `concepts_intro`, producing structurally correct but semantically empty index entries.
- The `classification_nudge` returned by `doc register` is described in retrospective notes as "seen and deferred."

The investigation identifies two coupled root causes:

**C1 — Voluntary-step compliance failures.** The enrichment steps are advisory at the tool level. Agents are asked to self-motivate additional work while their attention is committed to a competing primary task (committing work, advancing to the next task). Advisory nudges consistently lose to task completion pressure. The fix requires escalating concept tagging from soft constraint (𝒮) to hard constraint (ℋ) by making `doc approve` refuse documents that have no concept-tagged sections.

**C2 — Tool information gap.** The current classify-on-register loop requires three tool calls (`register` → `guide` → `classify`) where two would suffice. The `guide` response does not include concept suggestions derived from section headings, so agents cannot complete a high-quality classify call without external knowledge of the role taxonomy. The fix requires the `guide` response to provide heading-derived concept suggestions alongside the role taxonomy so that a cold-context agent can classify correctly using only the `guide` output.

These two clusters are coupled at the classify-on-register loop: fixing enforcement (C1) without fixing tool affordance (C2) makes approval harder to reach; fixing tool affordance without enforcement leaves compliance voluntary. They must be addressed together.

If nothing changes, the concept registry remains empty, `doc_intel(action: "find", concept: ...)` returns no results, and the design-amnesia problem the entire doc-intel system was built to solve persists.

## Design

### Overview

Three server-side changes address C1 and C2 as a coordinated unit:

1. **Guide response concept enrichment (C2a)** — Add a `concepts_suggested` field to the `doc_intel guide` response. Each entry maps a section path to a list of concept name strings derived from the section's heading and its ancestors via a lightweight lexical extraction pass. This gives agents heading-derived concept candidates without requiring them to infer concepts from content they have not read.

2. **Guide suggested_classifications expansion (C2b)** — Extend `suggested_classifications` coverage beyond the two currently handled heading patterns (Acceptance Criteria, Alternatives Considered) to all sections whose heading unambiguously determines the fragment role (Problem and Motivation → `rationale`; Decisions → `decision`; Design → `decision`; Overview / Summary → `narrative`; Requirements / Goals → `requirement`; Risk / Risks → `risk`; Definition / Glossary → `definition`). This makes the `guide` response a complete starting point for role assignment on well-structured documents.

3. **Concept tagging approval gate (C1)** — Add a server-side check to `doc(action: "approve")` for documents of type `specification`, `design`, and `dev-plan`. If the document has been classified (at least one entry in the classification index) but no classified section has `concepts_intro` populated, approval is blocked with a structured error message that cites the offending document ID, the current `content_hash`, and a pointer to the `guide` action. Documents with no classification index entries at all are not blocked by this gate — the existing `classification_nudge` mechanism remains the first-call prompt for unclassified documents.

### Component responsibilities

**`internal/mcp/doc_intel_tool.go` — `docIntelGuideAction`**

Currently builds a `guideResponse` struct with `document_id`, `document_path`, `content_hash`, `classified`, `outline`, `entity_refs`, `extraction_hints`, `suggested_classifications`, and `taxonomy`. Two additions:

- `concepts_suggested`: `[]sectionConceptSuggestion` — one entry per section in the outline. Each entry contains `section_path` (string), `section_title` (string), and `suggested_concepts` ([]string). The concept list is derived by passing the section title (and parent title chain for nested sections) through a normalising lexical pass: strip stop words, split on `/`, `-`, and spaces, title-case each token, deduplicate. Entries with an empty `suggested_concepts` list are omitted.
- `suggested_classifications` expansion: the heading-match table in `buildSuggestedClassifications` (or equivalent helper) gains the additional heading patterns listed above.

**`internal/mcp/doc_tool.go` — `docApproveOne`**

After the existing `docSvc.ApproveDocument` call resolves successfully, add a pre-approval check (before the commit) that queries the intelligence service for the document's classification index. The check logic:

```
if doc.Type in {specification, design, dev-plan}:
    entries = intelSvc.GetClassifications(doc.ID)
    if len(entries) > 0 and not any(e.ConceptsIntro != nil and len(e.ConceptsIntro) > 0 for e in entries):
        return structured error: concept_tagging_required
```

The error response includes `document_id`, `content_hash` (from the index), and the instruction text:
> "At least one classified section must have concepts_intro populated. Call doc_intel(action: \"guide\", id: \"<id>\") to see concept suggestions, then doc_intel(action: \"classify\", ...) with concepts_intro on at least one section."

The `IntelligenceService` must expose a `GetClassifications(docID string) ([]ClassificationEntry, error)` method (or equivalent). If the intelligence service is unavailable (nil), the gate is skipped to preserve backward compatibility with projects that have not enabled doc-intel.

**Data flow for the happy path (post-design)**

```
doc register → classification_nudge (content_hash + outline)
                ↓
doc_intel guide → outline + suggested_classifications + concepts_suggested + taxonomy
                ↓
doc_intel classify (with concepts_intro on ≥1 section)
                ↓
doc approve → gate check passes → approved
```

**Data flow for the blocked path**

```
doc_intel classify (without concepts_intro)
                ↓
doc approve → gate check: classified but no concepts_intro → ERROR concept_tagging_required
                ↓  (agent sees error with content_hash and guide pointer)
doc_intel guide → concepts_suggested helps agent pick concepts
                ↓
doc_intel classify (with concepts_intro on ≥1 section)
                ↓
doc approve → gate check passes → approved
```

### Failure modes

- **Intelligence service unavailable at approve time:** Gate is skipped. The nudge path remains as a soft fallback. This preserves backward compatibility.
- **Document never classified:** Gate does not fire. The existing `classification_nudge` is the first-line prompt. This avoids blocking documents that are being approved before any classification has happened (e.g. policy documents, reports).
- **Concept extraction produces no candidates for a section:** The section is omitted from `concepts_suggested`. Agents must supply concepts manually. This is acceptable — the gate requires only one section with concepts, not all sections.
- **Large documents with 100+ sections:** The heading-based extraction pass is O(n) in section count and stateless. No caching is required. The `guide` call already reconstructs the outline on each call; concept extraction adds a constant-time pass per section title.

## Alternatives Considered

### Alternative A: Enforce concept tagging at classify time

Reject any `doc_intel classify` call where `concepts_intro` is absent on all sections.

**Trade-offs:** Simple single-point enforcement. But it breaks the two-phase batch workflow used during corpus backlog runs, where agents intentionally classify role/confidence in one pass and concepts in a second pass. It also penalises agents classifying documents that genuinely have no domain concepts (short policy documents, single-section reports).

**Rejected** because RG-2 investigation specifically evaluated this option and found the batch workflow breakage unacceptable. Approve-time enforcement is more targeted.

### Alternative B: Add concepts_suggested to the register nudge only

Include heading-derived concept suggestions in the `classification_nudge` payload returned by `doc register`, rather than in `guide`.

**Trade-offs:** Reduces the round-trip from three calls to two immediately at registration. But the register response is already large (outline + message + content_hash), and concepts derived at registration time are computed before the agent has reviewed the document. The `guide` action is the correct semantic home for extraction assistance — it is designed to help agents prepare for classification.

**Rejected** in favour of enriching `guide`, which is the intended entry point for classification preparation. The two-call path (`register` → `classify`) is already achievable because `register` returns `content_hash` and the outline; the missing piece is concept suggestions, which belong in `guide`.

### Alternative C: Populate concepts automatically from headings without agent involvement

Run the heading extraction pass at index time and pre-populate `concepts_intro` in the classification index, bypassing agent involvement entirely.

**Trade-offs:** Zero agent effort. But auto-populated concepts would be raw heading tokens, not validated domain concepts — the concept registry would become polluted with noise (section structural terms like "Overview", "Goals", "Requirements" are poor concepts). The value of `concepts_intro` is agent-validated, domain-meaningful concept names. Automated extraction is useful as a *suggestion* but not as a *record*.

**Rejected.** Suggestions (via `concepts_suggested`) are correct; auto-population is not.

### Alternative D: Do nothing — rely on stronger skill guidance

Rewrite `kanbanzai-documents` SKILL.md to more forcefully require `concepts_intro` on every classify call.

**Trade-offs:** Zero code change. But P27 tried this (stronger wording) and the investigation documents zero improvement in concept registry population. Advisory instructions consistently lose to task completion pressure when the agent's attention is on a competing primary task.

**Rejected.** The investigation explicitly rules this out as a structural solution. Hard constraints (ℋ) are required.

## Decisions

**Decision:** Add `concepts_suggested` to the `doc_intel guide` response rather than to any other response.
**Context:** Agents need concept candidates at the point they are preparing to classify, not at registration time when their attention is elsewhere.
**Rationale:** The `guide` action is explicitly designed as the classification preparation step. Placing suggestions here keeps the tool semantics coherent and avoids bloating the register response.
**Consequences:** Agents who skip the `guide` call will not see concept suggestions. This is acceptable because the approval gate (below) provides the backstop; the guide is the low-friction path, not the only path.

---

**Decision:** Extend `suggested_classifications` to all heading-deterministic sections.
**Context:** The current coverage is two patterns. The investigation found ~56% of sections are heading-deterministic. Partial coverage provides partial value and may mislead agents into thinking only the covered headings have suggestions.
**Rationale:** Complete the existing mechanism rather than leaving it partially implemented.
**Consequences:** More sections in the guide response will carry a `suggested_role`. Agents are free to override suggestions; the classify action does not enforce them.

---

**Decision:** Block `doc approve` when a classified document has no concept-tagged sections, for specification, design, and dev-plan types only.
**Context:** C1 root cause is that concept tagging is voluntary. The investigation establishes ℋ enforcement is required for the compliance rate the corpus depends on. Approve is the correct enforcement point (RG-2).
**Rationale:** Approve is a deliberate, non-batched, human/agent-triggered action. It is the last gate before a document enters the permanent record. Blocking here catches omissions without disrupting batch classification runs.
**Consequences:** Agents must populate `concepts_intro` on at least one section before approving affected document types. This adds one classify-with-concepts step to the workflow for these document types. Documents that have never been classified are not blocked — the gate fires only when classification records exist but concepts are absent.

---

**Decision:** Skip the concept-tagging gate when the intelligence service is unavailable.
**Context:** Projects that have not enabled doc-intel should not be blocked from approving documents.
**Rationale:** Backward compatibility. The gate is a doc-intel feature; it must not become a dependency for projects that do not use doc-intel.
**Consequences:** On projects without doc-intel, the gate is silently skipped. This is the same pattern used by the classification_nudge in `doc register`.