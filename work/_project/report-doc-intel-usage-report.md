# Document Intelligence Usage Report

**Date:** 2026-04-22  
**Scope:** Kanbanzai MCP Server — `doc_intel` and related knowledge systems  
**Purpose:** Assess the effectiveness of the Document Intelligence subsystem by auditing what has been stored, how it is used, and where gaps exist.

---

## Executive Summary

The Document Intelligence system is **architecturally healthy and structurally active** — every registered document has been parsed into a hierarchical section index and pattern-based role extraction is functioning correctly across hundreds of documents. However, two significant gaps limit its practical effectiveness:

1. **AI classification (Layer 3) is pending for 332 of 334 registered documents.** The LLM-assisted semantic classification that would enable the richest queries (concept search, decision extraction, rationale tracing) has essentially never run.
2. **Knowledge entries are never retrieved via the API.** All 59 knowledge entries have a `use_count` of 0, meaning agents are accumulating knowledge but not consuming it through the tool surface.

---

## 1. What Information Has Been Stored

### 1.1 Document Registry

The document registry contains **334 registered documents** spanning the full project history from March 25 to April 22, 2026.

| Document Type    | Count (approx) | Notes |
|------------------|---------------|-------|
| `specification`  | ~110          | Per-feature specs, the most common type |
| `dev-plan`       | ~95           | Implementation plans, often paired 1:1 with specs |
| `design`         | ~60           | Architectural and feature design documents |
| `report`         | ~55           | Reviews, retrospectives, audits, handoff notes |
| `research`       | ~11           | Background analyses |
| `retrospective`  | ~3            | Formal retro documents |

**Approval status:**
- Approved: ~202 documents (~60%)
- Draft: ~132 documents (~40%)
- Sole approver in all cases: `sambeau`

**Registry health (from `doc audit`):**
- Files on disk: 284
- Registered and on-disk: 275
- Registered but missing from disk: 4 (stale records)
- On-disk but unregistered: 9 (orphaned files including `docs/user-guide.md`, `work/design/skills-system-redesign-v2.md`)

Documents are linked to workflow entities (features and plans) via the `owner` field, establishing traceability from design intent through to review outcomes.

### 1.2 Document Intelligence Index

The `doc_intel` index stores **structural section trees (Layer 1)** for all registered documents. Each document is decomposed into a hierarchy of sections with:

- Section path (e.g., `1.3.2`)
- Heading level and title
- Byte offsets and word counts
- Content hashes for change detection

A sample outline for `work/design/document-intelligence-design.md` spans 19 top-level sections with 3 levels of nesting — demonstrating that even the most complex documents are fully parsed.

**Pattern-based role extraction (Layer 2)** is also active. A query for all sections tagged with the `requirement` role returned **372 section matches** across dozens of documents, all with `confidence: high`. The pattern matcher correctly identifies sections titled "Acceptance Criteria", "Functional Requirements", "Non-Functional Requirements", etc., without any LLM involvement.

**Full-text search (FTS)** is operational via SQLite FTS5, implemented as part of Plan P23. A search for `"document intelligence"` returned 2 precise section matches with BM25 relevance scores, confirming the engine is functioning.

### 1.3 Knowledge Base

The knowledge base holds **59 entries** contributed over the project lifetime.

| Tier | TTL | Count | Topics |
|------|-----|-------|--------|
| Tier 2 (project-level) | 90 days | ~35 | Architecture, conventions, anti-patterns, policy |
| Tier 3 (session-level) | 30 days  | ~24 | Retrospective signals, worked-well notes, tool friction |

Representative tier 2 entries (permanent project knowledge):
- `document-intelligence-three-layer-architecture` — the canonical description of Layers 1–3
- `decompose-propose-stale-index-fallback` — root cause of a major workflow failure with fix direction
- `docint-does-not-classify-ac-pattern-as-acceptance-criteria` — significant tool gap (now resolved in P24)
- `p7-retro-git-stash-destroys-mcp-state` — critical workflow hazard with detailed diagnosis
- `mcp-thin-adapter-pattern` — architectural convention for all tool implementations
- `flaky-tests-policy` — agent behaviour policy for CI failures

Representative tier 3 entries (session signals):
- Various `retro-task-*` entries recording worked-well observations and friction notes
- `worktree-file-editing-pattern` — discovered workaround for a tool gap

**Staleness check:** 0 stale entries detected. All entries are within their TTL windows.

---

## 2. How Useful Is the Stored Information

### 2.1 Document Structure Index — High Value

The section hierarchy index is the most consistently useful component. It enables:

- **`doc_intel outline`**: Precise navigation to any section of any document without reading the full file.
- **`doc_intel section`**: Targeted retrieval of a specific section by path.
- **`doc_intel find(role: requirement)`**: Cross-corpus discovery of all acceptance criteria sections (372 results), which directly powers the `decompose` tool's AC extraction.

This layer is the backbone of the `decompose propose` workflow. The fix delivered in P24 (`p24-ac-pattern-and-decompose`) was only possible because the structural index existed; the bug was in the pattern classifier, not the index itself.

### 2.2 Full-Text Search — Functionally Present, Underused

FTS is operational but returns sparse results for natural language queries. A search for `"document intelligence"` returned only 2 results, even though the concept appears throughout the corpus. This suggests either:
- FTS5 requires BM25-style keyword decomposition rather than phrase queries
- The index was only recently built (P23 shipped April 20) and the corpus hadn't been re-indexed

FTS5 is best used for targeted keyword searches (e.g., `doc_intel search query:"sqlite FTS"`) rather than concept searches.

### 2.3 AI Classification (Layer 3) — Almost Entirely Absent

**332 of 334 documents are pending AI classification.** This is the most significant gap. Layer 3 classification would:

- Enable concept-based retrieval (currently returns 0 results — confirmed by `find(concept: "acceptance criteria")`)
- Assign semantic roles beyond what pattern matching can detect (rationale, constraint, assumption, design decision)
- Power the `doc_intel guide` action for document-level AI summaries
- Enable the `impact` action for change propagation analysis

Until Layer 3 runs, `doc_intel` is a **structural navigation tool** rather than a semantic intelligence layer.

### 2.4 Knowledge Base — Stored but Not Consumed

Every single knowledge entry has `use_count: 0`. This is the starkest finding in this audit. The knowledge base is being **written to** (by `finish` retrospective signals) but never **read from** via the tool API.

The intended usage pattern — agents calling `knowledge list` before writing retrospective or review documents, as mandated by the kanbanzai-agents skill — does not appear to be happening consistently. Knowledge entry `KE-01KMT9J3YKJCB` explicitly documents this anti-pattern: *"During P7, the agent wrote the entire retrospective from in-session memory, missing cross-plan learning signals."*

The knowledge that is stored is, however, of high quality where tier 2. Entries like `decompose-propose-stale-index-fallback` and `p7-retro-git-stash-destroys-mcp-state` represent accumulated diagnostic knowledge that took significant effort to discover and would have been painful to rediscover. The value is latent — it is there, but agents are not drawing on it.

---

## 3. How Regularly Is New Information Added

### 3.1 Documents

Document creation is continuous and tightly coupled to feature development. Over approximately 4 weeks (March 25 – April 22, 2026):

- **334 documents registered** ≈ ~12 per day on average
- Creation is bursty: large batches coincide with plan delivery days (P16 delivered 26+ documents in a single session on April 1–2; P25 delivered 14 documents on April 21)
- The workflow enforces document creation at every lifecycle stage (design → spec → dev-plan → review), so document volume scales linearly with feature count

**Trend:** Document creation is healthy and self-sustaining as a workflow byproduct.

### 3.2 Knowledge Entries

Knowledge creation is less systematic:
- 59 entries over ~4 weeks ≈ ~2 per day
- Tier 3 entries are created as a side effect of task completion (via `finish` retrospective signals)
- Tier 2 entries require deliberate authorship and appear in clusters following significant workflow incidents (P7 git stash disaster produced 3 entries in one sitting)
- No entries have been promoted to tier 1 (global)

**Trend:** Reactive rather than proactive. Entries are created after pain, not in anticipation of it.

### 3.3 AI Classification

**No classification events have occurred.** The `doc_intel classify` action requires an external LLM agent to call it explicitly with a classification payload. The batch classification pipeline (P23, `FEAT-01KPNNYYXQSYW`) provides the mechanism but does not run automatically. Classification requires a deliberate orchestration trigger.

---

## 4. How Regularly Is Information Accessed

### 4.1 Knowledge Base Accesses

**Use count: 0 for all 59 entries.**

This metric is unambiguous. Agents are not calling `knowledge list` or `knowledge get` before beginning work. The intended closed loop — contribute knowledge on task completion, consume it on the next related task — is broken at the consumption end.

### 4.2 Document Section Accesses

There is **no access counter on doc_intel section reads**. The system tracks document registration and classification state but does not instrument `outline`, `section`, `find`, or `search` call frequency. This makes it impossible to quantify how often the structural index is used.

From agent session traces visible in retrospective entries, it is clear that `doc_intel outline` and `doc_intel section` are used by sub-agents during task implementation (e.g., the review pipeline uses `doc_intel find` to locate acceptance criteria sections). However, frequency cannot be quantified.

### 4.3 FTS Search Accesses

Similarly untracked. The FTS engine is available but no telemetry exists for query frequency or result quality.

---

## 5. How Often Does Information Expire

### 5.1 Knowledge Entries

The TTL system is configured but **no entries have expired** in the project's lifetime:

- Tier 3 (30-day TTL): All tier 3 entries were created within the last 30 days; none are yet expired
- Tier 2 (90-day TTL): All tier 2 entries are within their 90-day window
- The `staleness` check returned 0 stale entries

**Expected first expiry:** Tier 3 entries created in late March 2026 will expire around late April 2026. Some session-level retrospective notes (e.g., `KE-01KMS0EE96969`, `KE-01KMS0EE97M2P` from March 28) are approaching their 30-day limit.

### 5.2 Documents

Documents have **no expiry mechanism.** The registry uses `content_hash` for change detection but has no TTL or staleness policy. Documents accumulate indefinitely. Draft documents from March 2026 (e.g., phase-1 through phase-4 implementation plans, now superseded by the kanbanzai 2.0+ architecture) remain registered and indexed but have no mechanism to be flagged as obsolete.

### 5.3 AI Classifications

Since no classifications exist, there is nothing to expire. The classification state is binary: pending or classified. Future classifications, once made, would be invalidated by content hash changes (the index tracks `content_hash` per section), requiring reclassification when documents change.

---

## 6. Unanswerable Questions and Proposed Improvements

### Q: How often is the knowledge base accessed?

**Currently unanswerable.** `use_count` is tracked but consistently 0. Even if it were non-zero, there is no timestamp on accesses, so frequency cannot be measured.

**Proposal:** Add a `last_accessed_at` timestamp and `recent_use_count` (rolling 30-day window) to knowledge entries. Surface "high-use entries" in the `knowledge list` default output so agents are drawn to the most proven knowledge.

### Q: Which document sections are most consulted?

**Currently unanswerable.** Section reads are not instrumented.

**Proposal:** Add an access log or hit counter to `doc_intel section` reads. Even a simple per-document `access_count` and `last_accessed_at` would reveal which specs are being actively referenced versus which are archival.

### Q: How accurate is the pattern-based role extraction?

**Partially answerable.** The 372 "requirement" role assignments all have `confidence: high`, and cross-checking a sample confirms they are correct (the pattern correctly identifies heading titles like "Acceptance Criteria"). However, false negatives — requirements expressed in prose rather than headings — are invisible.

**Proposal:** Once Layer 3 AI classification runs, compare its role assignments against Layer 2 pattern results. Track false-positive and false-negative rates per document type. The `doc record_false_positive` action exists for this purpose.

### Q: What fraction of knowledge entries are actually correct?

**Currently unanswerable.** All 59 entries are in `contributed` status; none have been `confirmed`. The distinction between contributed and confirmed exists but no agent has promoted any entry.

**Proposal:** During plan review gates, schedule a `knowledge confirm` pass for all tier 2 entries contributed during that plan's execution. This would transform the knowledge base from a write-only log into a curated, trusted reference.

### Q: When will Layer 3 classification run?

**Currently indeterminate.** The batch classification mechanism (P23) exists but has no automatic trigger. Without explicit orchestration, the 332 pending documents will remain unclassified indefinitely.

**Proposal:** Add a `doc_intel classify_pending` job to the project startup checklist in `AGENTS.md`, or trigger it automatically when a feature enters the `reviewing` stage (so that the spec and design documents are classified before a reviewer agent reads them).

---

## 7. Summary Scorecard

| Dimension | Status | Score |
|---|---|---|
| Document registration completeness | 275/284 on-disk files registered; 4 missing, 9 unregistered | ✅ Good |
| Section index coverage | 334/334 documents indexed with full section trees | ✅ Excellent |
| Pattern-based role extraction | 372 classified sections, all high confidence | ✅ Working |
| AI classification (Layer 3) | 332/334 pending; essentially none classified | ❌ Not started |
| Concept model | 0 concepts extracted | ❌ Empty |
| Knowledge base volume | 59 entries, healthy mix of tiers | ✅ Good |
| Knowledge base consumption | use_count = 0 for all entries | ❌ Not consumed |
| Knowledge base staleness | 0 stale entries | ✅ Good |
| FTS search | Operational; query patterns may need tuning | ⚠️ Partial |
| Access instrumentation | Not present for sections or searches | ⚠️ Gap |

---

## 8. Recommendations (Priority Order)

1. **Trigger Layer 3 batch classification.** Run `doc_intel classify` across the corpus. Until this happens, the semantic layer does not exist. Even a partial run on approved specifications would unlock concept search and rationale extraction for the most critical documents.

2. **Enforce `knowledge list` at task start.** Add an explicit pre-task step to the `implement-task` and `orchestrate-development` SKILL.md files requiring agents to call `knowledge list` with relevant tags. The knowledge is stored but not consulted.

3. **Confirm tier 2 knowledge entries.** Schedule a knowledge confirmation pass at plan close-out. Promotes the knowledge base from an append log to a curated reference.

4. **Register unregistered files.** 9 files on disk are unregistered (`docs/user-guide.md`, etc.). These should be registered or deliberately excluded.

5. **Add document access instrumentation.** A simple `access_count` and `last_accessed_at` on `doc_intel` section reads would reveal which documents agents actually rely on, enabling prioritised classification and curation.

6. **Create a classification trigger in the review pipeline.** When a feature transitions to `reviewing`, automatically enqueue its design and specification documents for Layer 3 classification. This ensures reviewer agents have semantic indexes for the documents they are about to assess.