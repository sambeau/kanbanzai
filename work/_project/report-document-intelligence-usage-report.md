# Document Intelligence Usage Report

## Overview

This report reviews the current state of Kanbanzai's Document Intelligence capability based on the repository state, indexed document records, implementation code, and the currently available action logs.

The goal is to answer:

1. What information has been stored?
2. How useful is the information it has stored?
3. How regularly is new information added?
4. How regularly is information accessed?
5. How often does information expire?

Where the available data does not support a confident answer, this report calls that out explicitly and proposes how to measure it in future.

## Executive Summary

Document Intelligence is already storing a substantial amount of structured information and appears effective at **document ingestion, structural indexing, cross-reference extraction, and search preparation**.

Its strongest current capabilities are:

- structural parsing of documents into sections
- extraction of entity references and cross-document links
- conventional role tagging from headings
- graph construction for document relationships
- full-text search infrastructure in the implementation
- targeted retrieval by entity, role, concept, section, and graph impact

Its weakest current area is **observability of actual usage and effectiveness**:

- there is very little recorded evidence of `doc_intel` tool usage in the available logs
- there is no durable access telemetry for indexed documents, sections, concepts, or graph edges
- there is no built-in quality scoring for whether stored intelligence was actually helpful
- there is no expiry model for document intelligence data comparable to knowledge TTL

The system is therefore much stronger at **storing intelligence** than at **proving that the intelligence is being used effectively**.

## Scope and Evidence Basis

This report is based on:

- registered document metadata in `.kbz/state/documents`
- document intelligence index files in `.kbz/index/documents`
- document intelligence implementation under `internal/docint/`, `internal/service/`, and `internal/mcp/`
- available action logs under `.kbz/logs`

This is a repository-state analysis, not a production telemetry analysis.

## 1. What information has been stored?

### 1.1 Registered document corpus

The repository currently contains a large registered document corpus.

Observed counts from document records:

- **334 total document records** inferred from status/type totals
- **202 approved**
- **132 draft**

By type:

- **106 specification**
- **95 dev-plan**
- **63 design**
- **57 report**
- **12 research**
- **1 retrospective**

This means the document system is already operating over a meaningful body of project knowledge rather than a toy dataset.

### 1.2 Document Intelligence index coverage

The document intelligence index currently contains approximately:

- **335 indexed document files** under `.kbz/index/documents`

That is broadly consistent with the registered corpus size, which suggests indexing coverage is high.

### 1.3 Stored per-document intelligence

From the implementation and sampled index files, each indexed document can store:

#### Layer 1: Structural skeleton

For each document:

- document ID
- document path
- content hash
- indexed timestamp
- hierarchical section tree

For each section:

- section path
- heading level
- title
- byte offset
- byte count
- word count
- content hash
- child sections

This is strong foundational data. It supports outline queries, section retrieval, drift detection, and precise addressing of document fragments.

#### Layer 2: Pattern-based extraction

Per document, the system stores:

- front matter fields
- entity references
- cross-document links
- conventional roles inferred from headings

Observed extracted data volume includes:

- **1,585 entity references**
- **2,319 cross-document links**

This is already a significant relationship graph.

#### Layer 3: Classification fields

The schema supports richer AI-assisted classification:

- fragment role
- confidence
- summary
- concepts introduced
- concepts used
- classifier model name
- classifier model version
- classification timestamp

However, in the sampled current index state:

- sampled recent doc-intel specification indexes were still `classified: false`
- repository-wide grep-based checks did not show clear evidence of persisted `classified: true` entries in the current checked files

So the capability exists, but current persisted classification coverage appears low or absent in the present repository snapshot.

#### Layer 4: Graph representation

The system stores graph edges such as:

- `CONTAINS`
- `REFERENCES`
- `LINKS_TO`
- `INTRODUCES`
- `USES`

It also supports a concept registry with:

- canonical concept name
- aliases
- introduced-in references
- used-in references

The implementation clearly supports this, but current persisted concept usage appears limited in the checked index files.

### 1.4 Search-oriented storage

The implementation also stores or can rebuild SQLite-backed search/index structures for:

- section full-text search
- graph edges
- entity references

This is important because it means the system is not limited to YAML scans; it has a path toward scalable retrieval.

### 1.5 Summary of what is stored

At a high level, Document Intelligence currently stores:

- document metadata
- document structure
- section-level addressing
- content hashes for freshness validation
- entity references
- cross-document links
- heading-derived semantic roles
- optional AI classifications
- graph edges
- optional concept registry data
- search-oriented SQLite projections

That is a strong and well-layered storage model.

## 2. How useful is the information it has stored?

## 2.1 Strong usefulness areas

### Structural usefulness: high

The section tree, offsets, hashes, and titles are immediately useful for:

- outline generation
- section retrieval
- precise extraction guidance
- stable references into long documents
- change detection

This is clearly effective and already operational.

### Relationship usefulness: high

Entity references and cross-document links are highly useful because they enable:

- tracing a feature or task through documents
- finding where an entity is discussed
- impact analysis
- document navigation
- refinement-chain style workflows

With **1,585 entity references** and **2,319 cross-document links**, this is not hypothetical value; there is enough stored relationship data to support real navigation and traceability.

### Search usefulness: potentially high

The implementation supports:

- full-text section search
- role-filtered search
- document-type-filtered search
- graph impact queries
- entity-based lookup
- concept-based lookup

This is a strong retrieval surface. Even if usage is currently low, the stored information is structurally useful.

## 2.2 Moderate usefulness areas

### Conventional role tagging: moderate to high

Observed role counts include:

- `narrative`: 631
- `constraint`: 427
- `requirement`: 374
- `decision`: 159
- `alternative`: 96
- `assumption`: 89
- `risk`: 78
- `question`: 65
- `rationale`: 49
- `example`: 47
- `definition`: 25

This is useful because it gives the system a semantic layer without requiring full AI classification.

However, these roles are mostly heading-pattern-derived, so their precision depends on document authoring consistency. They are useful, but not as rich as validated fragment-level classification.

## 2.3 Weak usefulness areas

### AI classification usefulness: currently limited

The schema and service layer support rich classification, but the current repository snapshot does not show strong evidence that this layer is populated at scale.

That means the most advanced semantic features appear underutilized relative to the design.

### Measured usefulness: currently unanswerable

The biggest limitation is not whether the stored information *could* be useful, but whether we can prove it *has been* useful.

We currently do not have durable evidence for questions like:

- Did a `find(entity_id)` result help an agent complete work faster?
- Did `search` reduce manual document reading?
- Did `impact` identify the right downstream sections?
- Did role tagging improve extraction quality?
- Did concept links improve retrieval precision?

So the usefulness assessment is currently:

- **architecturally high**
- **operationally plausible**
- **empirically under-measured**

## 2.4 Overall usefulness judgment

My assessment is:

- **Stored information quality:** good
- **Coverage of useful structural/relational data:** good
- **Coverage of rich semantic classification:** limited
- **Evidence of realized effectiveness:** weak

So the system appears **effective as an indexing substrate**, but **not yet well-instrumented enough to demonstrate end-user effectiveness**.

## 3. How regularly is new information added?

## 3.1 Document registration activity

Document creation activity appears regular over the observed period.

Observed `created_at` counts by day include:

- 2026-03-26: 9
- 2026-03-27: 7
- 2026-03-28: 25
- 2026-03-29: 9
- 2026-03-30: 2
- 2026-04-01: 54
- 2026-04-02: 35
- 2026-04-03: 6
- 2026-04-04: 3
- 2026-04-15: 5
- 2026-04-20: 23
- 2026-04-21: 23
- 2026-04-22: 1

This suggests:

- bursts of heavy document creation around planning and feature waves
- continued additions through April
- a workflow where document production is active and ongoing

### Interpretation

New information is being added **regularly, but in bursts rather than evenly**.

That is normal for a workflow-driven system: documents are created when features, plans, or reviews are produced.

## 3.2 Indexing cadence

Because indexed document files exist for roughly the same scale as registered documents, ingestion appears to happen routinely alongside document creation.

The implementation also updates indexes on ingest and classification, so the intended model is near-immediate refresh rather than periodic batch-only indexing.

## 3.3 Classification cadence

This is much less clear.

The implementation supports classification, but the current repository evidence does not show strong persisted classification coverage. So the best current answer is:

- **new structural and relational information is added regularly**
- **new AI-classified semantic information is not clearly being added regularly**

## 3.4 Answer

New information is added **regularly at the document and index level**, with noticeable bursts on active development days.  
New information is **not clearly being added regularly at the richer classification layer**.

## 4. How regularly is information accessed?

## 4.1 What we can measure

The available action logs provide some evidence of tool usage.

Observed totals from available logs:

- **1,718 total logged tool invocations**
- **11 `doc_intel` invocations**
- logs span **8 days**

Observed `doc_intel` actions:

- `find`: 1
- `outline`: 1
- `section`: 6
- `pending`: 2
- `search`: 1

## 4.2 Interpretation

This is a very low observed access rate for Document Intelligence relative to the rest of the system.

For comparison, the same logs show much heavier use of:

- entity operations
- document record operations
- task claiming
- status checks
- task completion

### What this likely means

One of these is true:

1. Document Intelligence is genuinely underused.
2. It is used mainly through indirect flows not visible as `doc_intel` calls.
3. The available logs cover only a limited recent window and are not representative.
4. Some document-intelligence-backed behavior is happening behind other tools rather than through explicit user-facing calls.

The implementation supports indirect use in context assembly and related workflows, so low direct `doc_intel` counts do **not** necessarily mean low total value.

## 4.3 What we cannot currently answer confidently

We cannot currently answer:

- how often indexed documents are loaded internally
- how often search results are returned
- how often graph queries succeed
- how often entity-reference lookups are used indirectly
- how often users abandon doc-intel queries after poor results
- which stored fields are most valuable in practice

The current logs are tool-invocation logs, not document-intelligence access telemetry.

## 4.4 Answer

Based on the available logs, **direct access appears infrequent**:

- only **11 direct `doc_intel` calls** in the available log window

But this is not enough to conclude low overall usefulness, because internal or indirect access is not well measured.

## 5. How often does information expire?

## 5.1 Current state

For Document Intelligence specifically, there does **not** appear to be a true expiry model.

What exists instead is:

- content-hash validation
- re-ingestion / overwrite behavior
- rebuild support for SQLite projections
- replacement of graph edges and refs when a document is re-indexed

This means document intelligence data is treated as **refreshable derived state**, not TTL-governed expiring state.

## 5.2 What does expire today?

The closest thing to expiry in the observed system is not document intelligence itself, but:

- action logs older than **30 days** are cleaned up
- knowledge entries have TTL concepts elsewhere in the system

So if the question is strictly about document intelligence data:

- **it does not appear to expire on a timed basis**

If the question includes the telemetry needed to assess usage:

- **the available access logs do expire after 30 days**

That is important, because it limits long-term effectiveness analysis.

## 5.3 Freshness vs expiry

Document Intelligence currently uses a **freshness model** rather than an **expiry model**:

- content hash mismatch prevents stale classifications from being blindly reused
- re-ingestion replaces outdated derived data
- graph and SQLite projections can be rebuilt

This is a sensible design for document-derived state.

## 5.4 Answer

- **Document intelligence records do not appear to expire on a TTL basis**
- **they are refreshed/replaced when documents change**
- **the main expiring artifact relevant to effectiveness analysis is the action log retention window of 30 days**

## Effectiveness Assessment

## What is working well

### 1. The storage model is strong

The layered model is well designed:

- structure
- extraction
- classification
- graph

This is a solid foundation.

### 2. Coverage is substantial

With roughly **335 indexed documents**, the system has enough corpus scale to be genuinely useful.

### 3. Relationship extraction is already valuable

The counts for entity references and cross-document links indicate real traceability value.

### 4. Search and graph capabilities are implemented in a scalable way

The SQLite-backed search path and graph query support are good signs for future scale.

## What is limiting effectiveness today

### 1. Rich semantic classification appears under-populated

The most advanced layer does not appear widely populated in the current snapshot.

### 2. Usage telemetry is weak

The system cannot currently prove:

- which queries are useful
- which stored fields are used
- which results lead to successful downstream work

### 3. Access evidence is too sparse

Only **11 direct `doc_intel` calls** are visible in the available logs.

### 4. Long-term analysis is constrained

Action logs are cleaned after 30 days, which is short for trend analysis.

## Direct Answers

## 1. What information has been stored?

Stored information includes:

- document metadata
- section trees with offsets, counts, and hashes
- front matter
- entity references
- cross-document links
- conventional semantic roles
- optional AI classifications
- graph edges
- optional concept registry data
- SQLite-backed search/index projections

Scale observed:

- about **335 indexed documents**
- **1,585 entity references**
- **2,319 cross-document links**

## 2. How useful is the information it has stored?

My assessment:

- **highly useful** for structure, navigation, traceability, and search
- **moderately useful** for heading-derived semantic roles
- **currently limited in realized value** for richer AI classification because that layer does not appear broadly populated
- **not well measured** in terms of actual downstream effectiveness

## 3. How regularly is new information added?

Regularly, in bursts.

Observed document creation spans many active days, with especially heavy additions on:

- 2026-04-01: 54
- 2026-04-02: 35
- 2026-04-20: 23
- 2026-04-21: 23

So new information is being added often enough to keep the corpus growing meaningfully.

## 4. How regularly is information accessed?

Directly observed access is low in the available logs:

- **11 direct `doc_intel` calls**
- across **8 logged days**

This is enough to say direct access is currently infrequent in the visible window, but not enough to measure total effective use.

## 5. How often does information expire?

Document intelligence data itself does not appear to expire by TTL.

Instead:

- it is refreshed or replaced when source documents change
- stale classifications are guarded by content-hash validation
- action logs expire after **30 days**

## What is currently unanswerable?

The following are not answerable with confidence from current data:

### A. Which stored information is actually most used?

We know what is stored, but not which fields are most frequently read or most helpful.

### B. Which queries are successful?

We do not have durable metrics for:

- zero-result rate
- click-through or follow-up rate
- whether returned results were used downstream

### C. Whether Document Intelligence improves workflow outcomes

We cannot currently quantify whether it reduces:

- time to find relevant docs
- time to assemble context
- review misses
- cross-document inconsistency

### D. How often semantic classification is refreshed after document edits

The schema supports this, but current telemetry is insufficient to measure it well.

## Recommendations for Future Measurement

## 1. Add document-intelligence access telemetry

Record durable events for:

- outline viewed
- section viewed
- search executed
- search result count
- entity trace executed
- impact query executed
- concept lookup executed
- pending classification viewed

For each event, capture:

- timestamp
- action
- document ID
- section path if applicable
- result count
- latency
- caller context or workflow stage if available

## 2. Add effectiveness metrics, not just usage metrics

Track:

- zero-result search rate
- average results returned
- repeated-query rate
- follow-up action rate after a doc-intel query
- whether a query is followed by task completion, doc approval, or review resolution

These would give a much better picture of usefulness.

## 3. Add classification coverage metrics

Track and report:

- indexed documents
- classified documents
- classification coverage by type
- average classified sections per document
- stale classification count after document edits

This would show whether the richer semantic layer is actually active.

## 4. Extend retention for analysis data

Thirty days is short for effectiveness analysis.

Consider:

- keeping aggregated usage summaries indefinitely
- keeping raw access logs longer
- rolling up daily/weekly doc-intel metrics before raw log cleanup

## 5. Add quality evaluation loops

For search and retrieval actions, capture lightweight quality signals such as:

- “useful / not useful”
- “found what I needed”
- “missing expected result”
- “wrong section / wrong document”

Even sparse feedback would be more informative than raw invocation counts alone.

## 6. Add a periodic health/effectiveness report

A recurring report could include:

- corpus growth
- classification coverage
- top accessed documents
- top accessed sections
- most common search terms
- zero-result searches
- stale index count
- rebuild frequency
- access trends by week

## Final Conclusion

Kanbanzai's Document Intelligence system appears **effective as a document indexing and retrieval foundation**, with strong storage of structural and relational information across a substantial corpus.

It is already good at answering questions like:

- what documents exist
- how they are structured
- where entities are referenced
- how documents link to each other
- which sections likely contain requirements, constraints, or decisions

However, its **measured effectiveness is currently hard to prove**, because:

- direct usage appears low in the available logs
- richer semantic classification does not appear broadly populated
- access telemetry is too limited to show which stored information is actually helping users

So the current verdict is:

- **storage effectiveness: strong**
- **retrieval capability: promising**
- **observed usage evidence: weak**
- **measurable end-user effectiveness: not yet well instrumented**

In short: Document Intelligence looks like a strong subsystem whose **technical foundation is ahead of its observability**.