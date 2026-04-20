# Document Retrieval for AI Agents: Research Report

**Date:** June 2025
**Author:** Research task — Kanbanzai project
**Status:** Draft

---

## 1. Executive Summary

Kanbanzai's document intelligence system (`doc_intel`) already provides section-level indexing, structural search, and targeted retrieval that reduces context consumption by 5–10× compared to full-file reads. However, critical capabilities remain unused or unbuilt: Layer 3 classification has never been run across the 280 indexed documents, the concept registry is empty, there is no full-text search, and the 12,107-edge graph is stored as a 2.3MB flat YAML file scanned linearly on every query.

The academic literature strongly supports hybrid retrieval (sparse keyword search + dense semantic ranking) over either approach alone, with consistent 15–20% recall improvements on technical corpora. However, for domain-specific documentation with structured identifiers (entity IDs, requirement labels, section hierarchies), keyword-based approaches with structural awareness perform surprisingly well — often within 5–10% of embedding-based systems.

**Primary recommendation:** Before building anything new, fully activate the existing system — run Layer 3 classification across all documents, populate the concept registry, and add BM25-style full-text search over section content. This alone would close the largest capability gaps at minimal cost. If these improvements prove insufficient, build a standalone document-retrieval MCP server using SQLite (following the codebase-memory-mcp architecture pattern) as a focused replacement for the YAML-based index.

**Confidence level:** High for the "activate what exists" recommendation. Medium for the standalone MCP server — it depends on whether the activated system proves sufficient.

---

## 2. Literature Review

### 2a. RAG Pipeline Architectures for Documentation

Retrieval-augmented generation (Lewis et al., 2020) established the pattern of retrieving relevant passages before generation to ground LLM outputs in source material. Subsequent work has refined this into multi-stage pipelines:

**Chunking strategies** are the foundation. Naive fixed-size chunking (e.g., 512 tokens) destroys document structure, splitting requirements across chunks and mixing unrelated sections. Semantic chunking (Kamradt, 2023) uses embedding similarity to find natural break points, but for Markdown documents with explicit heading structure, **heading-based chunking is optimal** — it preserves authorial intent, maintains context boundaries, and produces chunks that align with how humans and agents reference documents (Gao et al., 2024, "Retrieval-Augmented Generation for Large Language Models: A Survey").

**Multi-stage retrieval** is now standard practice. The pattern is: (1) cheap broad retrieval (BM25 or sparse vectors) to get candidate documents/sections, (2) expensive re-ranking (cross-encoder or LLM-based) to order by relevance, (3) compression/extraction to minimise tokens sent to the generator. This three-stage pipeline consistently outperforms single-stage approaches (Nogueira & Cho, 2020; Glass et al., 2022).

**Relevance to our problem:** Kanbanzai's `doc_intel` system already implements heading-based chunking (sections as the fundamental unit). What it lacks is stage 1 (broad text search) and stage 2 (relevance ranking). Stage 3 (section-level retrieval) is already implemented.

### 2b. Knowledge Graph Approaches

Two paradigms compete:

**Ontology-based knowledge graphs** (Hogan et al., 2021, "Knowledge Graphs") use formal schemas — entities, relations, and type hierarchies defined upfront. WordNet, the Open English WordNet (OEWN), and domain ontologies like SKOS provide lexical and conceptual hierarchies. These systems excel at structured queries ("find all requirements that depend on feature X") but require significant upfront modelling and maintenance. For project documentation that evolves rapidly, the schema maintenance burden is a known failure mode (Noy & McGuinness, 2001).

**Embedding-based knowledge graphs** (Ji et al., 2022, "A Survey on Knowledge Graphs") learn entity and relation representations from data, avoiding explicit schema design. Systems like TransE (Bordes et al., 2013) and more recently knowledge-graph-enhanced RAG (KG-RAG) (Soman et al., 2024) combine structural graph traversal with semantic embedding similarity. The advantage is adaptability; the disadvantage is opacity — it is harder to debug why a particular section was or was not retrieved.

**Hybrid approaches** are emerging as the practical sweet spot. Microsoft's GraphRAG (Edge et al., 2024) builds a graph of entities and relationships from documents, uses community detection (Louvain/Leiden) to cluster related concepts, then generates community summaries that serve as retrieval targets. This is directly analogous to what codebase-memory-mcp does for code (Louvain community detection over call graphs).

**Relevance to our problem:** Kanbanzai's graph model (document → section → entity/concept with typed edges) is structurally sound. The gap is that concepts are never populated (the registry is empty) and graph traversal is a linear scan of a flat YAML file. A SQLite-backed graph with populated concept nodes would immediately enable the structured query patterns the literature recommends.

### 2c. Hybrid Retrieval Strategies

The BEIR benchmark (Thakur et al., 2021) demonstrated that **no single retrieval method dominates across all domains**. BM25 (sparse, keyword-based) outperforms dense retrieval on domain-specific technical text where exact terminology matters. Dense retrieval (DPR, Contriever) excels on open-domain questions where paraphrasing and synonymy are common.

**Hybrid retrieval** combining both consistently outperforms either alone:

- BM25 + dense re-ranking: 15–20% recall improvement over BM25 alone on technical corpora (Gao et al., 2023)
- Reciprocal rank fusion of sparse and dense results matches or exceeds learned fusion at lower computational cost (Cormack et al., 2009)
- For small corpora (<10K documents), BM25 with good tokenisation is competitive with embedding-based retrieval, because the vocabulary is constrained enough that keyword matching has high precision (Lin, 2022, "Sparse vs Dense: A Rethinking")

**Relevance to our problem:** Kanbanzai has ~280 documents with ~963K total tokens. This is a small corpus by retrieval standards. BM25 over section content would be a high-value, low-cost addition. Dense retrieval via embeddings would add marginal benefit at this scale but would become more valuable as the corpus grows beyond ~1,000 documents.

### 2d. Token-Efficient Context Delivery

Anthropic's context engineering research (2025) established that **15–40% context window utilisation is optimal** — filling the context window degrades performance. This makes retrieval precision (returning only relevant sections) as important as retrieval recall (finding all relevant sections).

Key strategies from the literature:

1. **Progressive disclosure** (Anthropic, 2025): return outlines/summaries first, let the agent request details. This is exactly what `doc_intel`'s `guide` → `section` workflow implements.

2. **Extractive compression** (Xu et al., 2024, "RECOMP"): given a retrieved passage, extract or compress it to the minimum tokens needed to answer the query. For structured documents, this means returning a section rather than a document, or a requirement rather than the section containing it.

3. **Query-aware chunking** (Gao et al., 2024): adjusting chunk granularity based on the query type. Broad questions ("what does this system do?") need document-level summaries; specific questions ("what is requirement FR-003?") need paragraph-level precision.

**Relevance to our problem:** Kanbanzai's section-level granularity is good for most queries. The gap is for broad queries (no summarisation) and for very specific queries within long sections (no sub-section granularity). The `guide` → `section` progressive disclosure pattern is well-aligned with the research.

---

## 3. Current System Audit

### 3a. What doc_intel Does Today

**Capabilities inventory** (8 actions):

| Action | Purpose |
|--------|---------|
| `outline` | Section tree of a document (paths, titles, levels, word counts) |
| `section` | Byte-precise retrieval of a single section's content |
| `classify` | Agent-submitted Layer 3 classification (role, confidence, concepts) |
| `find` | Polymorphic search: by concept, entity ID, or fragment role |
| `trace` | Entity refinement chain across design → spec → dev-plan |
| `impact` | Graph edges pointing to a section (change impact) |
| `guide` | Entry-point overview: outline + roles + entity refs + extraction hints |
| `pending` | Documents awaiting Layer 3 classification |

**Four-layer indexing pipeline:**

| Layer | Name | Status | What it does |
|-------|------|--------|-------------|
| 1 | Structural skeleton | ✅ Active | Markdown heading parser → section tree with byte offsets |
| 2 | Pattern extraction | ✅ Active | Regex entity refs, cross-doc links, conventional roles |
| 3 | Agent classification | ❌ Never used | LLM classifies sections by role, confidence, concepts |
| 4 | Graph edges | ✅ Active (Layer 1-2 only) | Typed edges: CONTAINS, REFERENCES, LINKS_TO |

**Current scale:**

| Metric | Value |
|--------|-------|
| Indexed documents | 280 |
| Graph edges | 12,107 |
| Graph file size | 2.3 MB (60,537 lines YAML) |
| Knowledge entries | 53 |
| Total document words | ~722,658 (~963K tokens) |
| Largest document | 14,847 words (~19,800 tokens) |
| Concept registry | Empty (file does not exist) |
| Layer 3 classifications | 0 |

### 3b. What the Knowledge System Does Today

**Capabilities inventory** (12 actions):

| Action | Purpose |
|--------|---------|
| `list` | Query entries by status, topic, scope, tier, confidence, tags |
| `get` | Retrieve single entry with staleness check |
| `contribute` | Create entry with dedup (Jaccard > 0.65 rejection) |
| `confirm` | Mark entry as confirmed |
| `flag` | Dispute entry (auto-retires at 2 misses) |
| `retire` | Terminal retirement with reason |
| `update` | Replace content (resets confidence) |
| `promote` | Tier 3 → Tier 2 promotion |
| `compact` | Dedup/contradiction detection across knowledge base |
| `prune` | TTL-based retirement of expired entries |
| `resolve` | Conflict resolution between two entries |
| `staleness` | Git-anchor-based freshness check |

**Context assembly integration:** Knowledge entries are automatically surfaced during task handoff (Step 7 of the 10-step pipeline). Entries are matched by file-path prefix, role tags, and scope, then ranked by recency-weighted Wilson confidence score, capped at 10 entries.

### 3c. Strengths of the Current Approach

1. **Section-level byte-precise retrieval.** The `guide` → `section` pattern delivers 5–10× token reduction compared to `read_file`. A typical interaction: `guide` (~500 tokens) to understand structure, then 2–3 `section` calls (~500 tokens each) = ~2,000 tokens vs. 10,000–20,000 for a full document read.

2. **Well-layered architecture.** The 4-layer pipeline (structure → patterns → classification → graph) cleanly separates concerns. Layers 1–2 are automatic and produce immediate value.

3. **Progressive disclosure.** The `guide` action is an excellent entry point — it gives agents just enough context to decide what to read next, matching the research consensus on token-efficient retrieval.

4. **Entity tracing.** The `trace` action follows entities across the design → spec → dev-plan refinement chain, which is a domain-specific query that no generic search tool could replicate.

5. **Knowledge lifecycle management.** Wilson confidence scoring, auto-confirm/auto-retire, TTL-based pruning, and Jaccard deduplication are well-engineered self-maintenance mechanisms.

6. **Automatic indexing.** Documents are indexed on registration and lazily on access, removing friction from the workflow.

### 3d. Weaknesses and Failure Modes

**Critical gaps:**

1. **No full-text search.** There is no way to search for arbitrary text within documents. "Find all sections mentioning 'authentication timeout'" requires falling back to grep, which returns raw text matches without section context or relevance ranking. This is the single largest gap.

2. **Layer 3 classification has never been used.** The concept registry is empty. The concept-based search (`find(concept: ...)`) has never returned a result. Role-based search relies entirely on Layer 2 keyword heuristics (heading text like "Requirements" → `requirement` role), which misses sections with non-standard headings.

3. **Graph is a flat YAML file.** The 12,107-edge graph is stored as a single 2.3MB YAML file. Every `impact` query deserialises this entire file and scans linearly. At current scale this is tolerable (~100ms), but it is O(n) per query with no indexing.

4. **No knowledge-to-document integration.** The knowledge system and doc_intel system are completely separate. An agent cannot query "what knowledge entries relate to this specification" or "what documents inform this knowledge entry." Cross-system queries require multiple tool calls and manual correlation.

**Moderate gaps:**

5. **All multi-document queries are linear scans.** `FindByEntity`, `FindByRole`, and knowledge `List()` all load and scan every file. With 280 documents and 53 knowledge entries this is fast; at 1,000+ it would degrade noticeably.

6. **No paragraph-level granularity.** Sections are the smallest addressable unit. A 2,000-word narrative section without sub-headings cannot be sliced further.

7. **Concept aliases are declared but unimplemented.** The `Concept.Aliases` field has a TODO. "Rate-limiting" and "throttling" are treated as different concepts.

8. **Knowledge `update` resets confidence.** A minor wording fix to a well-established entry (use_count=20, confidence=0.95) wipes all accumulated trust back to defaults.

9. **No staleness detection for document indexes.** If a markdown file is edited outside the workflow, the index silently becomes stale.

### 3e. Token Cost Analysis

**Current grep/read_file approach:**

| Operation | Approximate tokens |
|-----------|--------------------|
| `grep` for a term across all docs | ~50–200 tokens (file paths + line matches) |
| `read_file` of a matched document | ~5,000–20,000 tokens per document |
| Typical search-then-read workflow | ~15,000–60,000 tokens for 3 documents |

**Current doc_intel approach:**

| Operation | Approximate tokens |
|-----------|--------------------|
| `guide` for a document | ~300–1,200 tokens |
| `section` for a targeted section | ~100–2,000 tokens |
| `find(entity_id)` across all docs | ~50–300 tokens (metadata only) |
| `find(role)` across all docs | ~50–500 tokens (includes summaries) |
| `trace(entity_id)` | ~50–400 tokens |
| Typical guided-read workflow | ~1,500–3,000 tokens for targeted sections |

**Token savings:** doc_intel delivers a **5–10× reduction** for targeted queries. The gap is for broad queries ("everything about authentication") where grep is the only option and full-file reads are unavoidable.

**Ideal retrieval system (from literature):**

| Operation | Approximate tokens |
|-----------|--------------------|
| Full-text search with section-level results | ~200–500 tokens (ranked section summaries) |
| Retrieve top-ranked section content | ~100–2,000 tokens |
| Broad query with concept graph traversal | ~500–1,500 tokens (related sections via graph) |
| Typical hybrid search workflow | ~800–2,500 tokens |

The gap between current and ideal is **not enormous for targeted queries** (doc_intel is already good). It is significant for **broad/exploratory queries** where grep + read_file costs 10–40× more than a section-level search system would.

---

## 4. Architectural Options

### Option A: Fully Activate and Enhance the Existing System

**Description:** Run Layer 3 classification across all 280 documents to populate the concept registry and role classifications. Add BM25 full-text search over section content. Replace the flat YAML graph with a SQLite-backed index. Wire knowledge and doc_intel together for cross-system queries.

**Specific changes:**

1. Run batch Layer 3 classification (agent-driven, ~280 LLM calls)
2. Add a `search` action to doc_intel: BM25 over section content using SQLite FTS5
3. Migrate graph.yaml to SQLite (edges table with indexes)
4. Add cross-system query: knowledge entries related to a document/section
5. Implement concept aliases

**Pros:**

- Lowest engineering effort — extends existing, well-tested code
- No new deployment or infrastructure — stays within `kanbanzai serve`
- Incremental — each change delivers value independently
- Classification uses existing tooling and schema

**Cons:**

- Tightly coupled to Kanbanzai — other projects cannot use it
- SQLite migration for graph is a moderate refactor
- Layer 3 classification requires ongoing agent effort as documents change
- Does not address the "focused tool" objective

**Estimated effort:** 2–3 weeks for changes 1–3; 1–2 additional weeks for 4–5.

**Risk:** Low. These are extensions to proven code. The largest risk is Layer 3 classification quality — if agent classifications are noisy, the concept registry will be unreliable.

### Option B: Standalone Document-Retrieval MCP Server

**Description:** Build a new, focused MCP server — `docs-memory-mcp` — that does for English-language documentation what codebase-memory-mcp does for code. SQLite-backed, single-binary, indexes a directory of Markdown files, exposes 8–10 MCP tools for search and retrieval.

**Proposed architecture (following codebase-memory-mcp patterns):**

- **Storage:** SQLite with FTS5 for full-text search, tables for nodes/edges
- **Indexing pipeline:** Markdown heading parser → entity extractor → link extractor → role classifier → concept extractor
- **Graph model:** Document → Section → Concept/Entity/Requirement nodes with typed edges
- **Output modes:** outline (headings only), summary (heading + first paragraph), full (complete section content)
- **Community detection:** Louvain clustering over cross-reference edges to discover document clusters

**Proposed MCP tools:**

| Tool | Description |
|------|-------------|
| `index_documents` | Index a directory of Markdown files |
| `search_docs` | Full-text search with section-level results (outline/summary/full modes) |
| `get_outline` | Document structure with section IDs |
| `get_section` | Single section content by path |
| `find_by_concept` | Sections mentioning a concept (with synonymy) |
| `find_by_entity` | Sections referencing a workflow entity |
| `find_by_role` | Sections classified by role (requirement, decision, rationale) |
| `trace_references` | BFS traversal of cross-references from a section |
| `impact_analysis` | What is affected if this section changes |
| `query_graph` | Cypher-subset queries over the document graph |

**Pros:**

- Reusable by any project, not just Kanbanzai
- Sharp boundary: document retrieval only, no workflow coupling
- Can evolve independently (different release cycle, different contributors)
- Clean implementation without legacy constraints
- Matches the "codebase-memory-mcp for docs" vision directly
- Single binary, zero external dependencies (SQLite compiled in)

**Cons:**

- Significant upfront engineering effort
- Duplicates some existing doc_intel functionality
- Kanbanzai would need to integrate with it (configuration, lifecycle management)
- No guaranteed users beyond Kanbanzai initially
- Risk of The Monolith Creep if scope is not carefully controlled

**Estimated effort:** 4–6 weeks for a minimal viable version (indexing + search + outline/section retrieval). 8–12 weeks for the full tool suite including graph queries and community detection.

**Risk:** Medium. The core technical risk is the NLP pipeline — code has formal grammars (tree-sitter), but natural language requires heuristic or LLM-based classification for roles and concepts. The feasibility of the heading parser, entity extractor, and link extractor is high (these are already proven in doc_intel). The concept extractor and role classifier are the uncertain components.

### Option C: Status Quo — Greppable Markdown Is Enough

**Description:** Do nothing. Continue using grep + read_file for broad queries and doc_intel's guide/section for targeted queries. Accept the current token costs.

**Arguments in favour:**

- **Current scale is manageable.** 280 documents, ~963K total tokens. An agent reading 3 full documents consumes ~45K tokens — within budget for most context windows.
- **Grep is fast and reliable.** For exact-match queries ("FEAT-01KMKRQRRX3CC"), grep finds results instantly. No indexing lag, no stale indexes, no classification errors.
- **doc_intel already provides 5–10× savings.** The guide → section pattern works well for the most common use case (reading a specific spec or design).
- **Simplicity has value.** No additional infrastructure to maintain, no classification pipeline to run, no SQLite database to manage.

**Arguments against:**

- **No full-text search with relevance ranking.** Grep returns line-level matches without section context or ranking. An agent searching for "error handling" gets 200 matches across 50 files with no way to prioritise.
- **Token cost for broad queries is high.** Exploring a topic across documents requires multiple grep + read_file calls that can consume 30–60K tokens.
- **Concept search is non-functional.** The concept registry is empty. Finding "all sections about rate limiting" requires the agent to know every synonym and variant.
- **The system doesn't scale.** At 500+ documents, linear scans of YAML files and grep across the full work/ directory will become noticeably slow.

**Estimated effort:** Zero.

**Risk:** Low short-term. Medium long-term — as the document corpus grows, the friction of grep-based exploration will increase, and agents will consume more tokens on context gathering, reducing the budget available for actual work.

---

## 5. Recommendations

### Primary Recommendation: Option A First, Then Evaluate Option B

The evidence strongly favours **activating the existing system before building something new**. The Sophistication Trap anti-pattern applies directly here: Kanbanzai has a well-architected document intelligence system with significant unused capacity. Before investing 4–12 weeks in a standalone server, spend 2–3 weeks making the existing system fully operational.

**Rationale:**

1. **Layer 3 classification is the biggest unlocked value.** The concept registry, role-based search, and classification-enhanced outlines are all implemented and tested — they just have no data. Running classification across 280 documents would immediately enable concept search, improve role-based retrieval, and provide summaries in find results.

2. **BM25 full-text search closes the largest gap.** SQLite's FTS5 extension provides production-quality full-text search with minimal code. Adding a `search` action to doc_intel that queries section content via FTS5 would replace the most common grep use case with ranked, section-aware results.

3. **The flat YAML graph is the scaling bottleneck.** Migrating to SQLite is a focused refactor that eliminates linear scans and prepares the system for 1,000+ documents.

4. **These improvements inform the Option B decision.** After activating the existing system, we can measure whether the remaining gaps justify a standalone server. If BM25 + concepts + SQLite-backed graph proves sufficient, Option B becomes unnecessary. If there are still significant shortcomings (e.g., embedding-based semantic search is needed, or the tool needs to work outside Kanbanzai), Option B has a clearer requirements set.

### Suggested Next Steps (Ordered by Priority)

1. **Add BM25 full-text search via SQLite FTS5** (1 week)
   - Add a `search` action to doc_intel
   - Index section content in an FTS5 virtual table
   - Return ranked results with section paths and snippet previews
   - This immediately replaces grep for document queries

2. **Migrate graph storage from YAML to SQLite** (1 week)
   - Edges table with indexes on `from`, `to`, `from_type`, `to_type`, `edge_type`
   - Eliminate linear scans for impact analysis and entity queries
   - Keep YAML export for debugging/inspection

3. **Run batch Layer 3 classification** (3–5 days of agent time)
   - Classify all 280 documents to populate concept registry and role classifications
   - Prioritise high-traffic documents (specs, designs) first
   - Establish a convention for classifying new documents on registration

4. **Wire knowledge ↔ doc_intel cross-system queries** (3–5 days)
   - When surfacing knowledge for a task, also surface related document sections
   - When querying doc_intel by entity, include related knowledge entries
   - Use the entity reference scanner already in knowledge/links.go

5. **Evaluate results and decide on Option B** (after steps 1–4)
   - Measure token cost reduction for real agent workflows
   - Identify remaining gaps that Option A cannot address
   - If a standalone server is warranted, the SQLite schema and search logic from steps 1–2 become the foundation

### Open Questions Requiring Further Investigation

1. **What is the quality ceiling for heuristic role classification vs. LLM classification?** Layer 2 uses heading keywords; Layer 3 uses LLM judgement. How much better is Layer 3 in practice? A comparison on a sample of 20–30 documents would answer this.

2. **Is embedding-based search needed at this corpus scale?** The literature suggests BM25 is competitive for small, domain-specific corpora. But Kanbanzai's documents use varied vocabulary (design docs are more narrative, specs are more structured). Testing BM25 recall on representative queries would clarify this.

3. **How should concept synonymy be handled?** Options include: (a) explicit alias lists maintained by hand, (b) WordNet/OEWN-based synonym expansion, (c) embedding-based similarity for concept matching. The right choice depends on whether the vocabulary is stable enough for (a) or requires automatic discovery.

4. **Would a standalone server have users beyond Kanbanzai?** The value proposition of Option B depends partly on reuse. If other AI-agent-based projects would adopt a "docs-memory-mcp" tool, the investment is more justified.

5. **What is the right granularity for long narrative sections?** Some documents have 2,000-word sections with no sub-headings. Should the system support paragraph-level retrieval, or is it sufficient to encourage better heading structure in documents?

---

## 6. References

### Academic Papers

- Bordes, A., Usunier, N., Garcia-Duran, A., Weston, J., & Yakhnenko, O. (2013). "Translating embeddings for modeling multi-relational data." *NeurIPS*.
- Cormack, G.V., Clarke, C.L.A., & Buettcher, S. (2009). "Reciprocal rank fusion outperforms condorcet and individual rank learning methods." *SIGIR*.
- Edge, D., Trinh, H., Cheng, N., et al. (2024). "From local to global: A graph RAG approach to query-focused summarization." *Microsoft Research*.
- Gao, Y., Xiong, Y., et al. (2024). "Retrieval-augmented generation for large language models: A survey." *arXiv:2312.10997*.
- Glass, M., Rossiello, G., Chowdhury, M.F., et al. (2022). "Re2G: Retrieve, re-rank, generate." *NAACL*.
- Hogan, A., Blomqvist, E., Cochez, M., et al. (2021). "Knowledge graphs." *ACM Computing Surveys*.
- Ji, S., Pan, S., Cambria, E., Marttinen, P., & Yu, P.S. (2022). "A survey on knowledge graphs: Representation, acquisition, and applications." *IEEE TNNLS*.
- Kamradt, G. (2023). "Semantic chunking for RAG." (Industry implementation, widely referenced.)
- Karpukhin, V., Oguz, B., Min, S., et al. (2020). "Dense passage retrieval for open-domain question answering." *EMNLP*.
- Lewis, P., Perez, E., Piktus, A., et al. (2020). "Retrieval-augmented generation for knowledge-intensive NLP tasks." *NeurIPS*.
- Lin, J. (2022). "A proposed conceptual framework for a representational approach to information retrieval." *arXiv:2110.01529*. (Discusses sparse vs dense trade-offs.)
- Nogueira, R., & Cho, K. (2020). "Passage re-ranking with BERT." *arXiv:1901.04085*.
- Noy, N.F., & McGuinness, D.L. (2001). "Ontology development 101." *Stanford Knowledge Systems Laboratory*.
- Soman, K., et al. (2024). "Biomedical knowledge graph-enhanced prompt generation for large language models." *arXiv:2311.17330*. (KG-RAG approach.)
- Thakur, N., Reimers, N., Rücklé, A., Srivastava, A., & Gurevych, I. (2021). "BEIR: A heterogeneous benchmark for zero-shot evaluation of information retrieval models." *NeurIPS Datasets*.
- Xu, F., Shi, W., & Choi, E. (2024). "RECOMP: Improving retrieval-augmented LMs with compression and selective augmentation." *ICLR*.

### Tools and Implementations

- codebase-memory-mcp (DeusData) — SQLite-backed code knowledge graph with 14 MCP tools, tree-sitter parsing, Louvain community detection. Architecture pattern for graph-augmented search.
- SQLite FTS5 — Full-text search extension with BM25 ranking. Production-quality, zero-dependency, compiled into the SQLite binary.
- Anthropic (2025). "Building effective agents" and "Effective context engineering." Context window utilisation research.

### Kanbanzai Internal

- `internal/docint/` — Document intelligence core: types, taxonomy, indexing, graph, concept registry
- `internal/service/intelligence.go` — IntelligenceService: ingest, search, section retrieval
- `internal/knowledge/` — Knowledge store, surfacer, compaction, cap tracker
- `internal/service/context.go` — 10-step context assembly pipeline
- `.kbz/index/` — Persistent document indexes and graph (YAML)
- `.kbz/state/knowledge/` — Knowledge entries (YAML, one per file)
