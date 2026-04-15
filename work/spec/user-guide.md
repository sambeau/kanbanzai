# User Guide: Specification

| Document | User Guide Specification |
|----------|--------------------------|
| Status   | Draft |
| Feature  | FEAT-01KP8-T4HQMEA3 |
| Created  | 2026-04-15 |
| Design   | `work/design/public-release-documentation.md` §6 (DOC P22-documentation-for-public-release/design-public-release-documentation) |

---

## Overview

This specification defines the User Guide (`docs/user-guide.md`) — the hub document of the public documentation collection. The User Guide provides high-level orientation to the Kanbanzai system: enough to understand what it is, how the pieces fit together, and where to go for detail. It links to every other document in the collection but does not duplicate their content.

The User Guide is the first document produced in the P22 production order because all subsequent documents reference it.

---

## Scope

### In scope

- A single Markdown document at `docs/user-guide.md`.
- Conceptual coverage of every major Kanbanzai subsystem at orientation depth (one to three paragraphs each).
- Cross-references (links) to every other document in the public documentation set.
- A "where to go next" section that routes readers to the right document for their goal.
- Production through the full five-stage editorial pipeline (Write → Edit → Check → Style → Copyedit) with advisory checkpoints after Edit and Copyedit.

### Out of scope

- Installation, setup, or tutorial content (Getting Started guide owns this).
- Detailed workflow methodology (Workflow Overview owns this).
- Individual tool parameters, entity fields, or configuration keys (reference documents own these).
- Agent-facing instructions or skill definitions (these are internal, not public documentation).
- Diagrams or visual assets beyond what the Write stage produces inline.

---

## Functional Requirements

### Content Requirements

- **FR-001:** The document MUST contain a section explaining what Kanbanzai is — the system, the methodology, and the MCP server — in two to three paragraphs of plain prose.

- **FR-002:** The document MUST contain a section describing the collaboration model: humans own intent (goals, priorities, approvals, direction); agents own execution (decomposition, implementation, verification, status tracking); documents are the interface between them.

- **FR-003:** The document MUST contain a section summarising the stage-gate workflow — the lifecycle stages a feature passes through — with enough detail to orient the reader but without reproducing the Workflow Overview's depth.

- **FR-004:** The document MUST contain a section describing the document types that drive the workflow (design, specification, dev plan) and how they relate to lifecycle stages.

- **FR-005:** The document MUST contain a section explaining how the human controls the process through approvals — what approval means at each stage and how features can be returned to earlier stages.

- **FR-006:** The document MUST contain a section providing a brief overview of the bug lifecycle and incident tracking.

- **FR-007:** The document MUST contain a section describing what the orchestration system does: context assembly, role-based skills, task dispatch, parallel execution, and conflict awareness.

- **FR-008:** The document MUST contain a section explaining the knowledge system: what knowledge entries are, why they persist, and how they compound over time.

- **FR-009:** The document MUST contain a section on the retrospective workflow: recording signals, synthesising, and generating reports.

- **FR-010:** The document MUST contain a section covering concurrency and parallel development: worktrees, conflict domain analysis, and merge gates.

- **FR-011:** The document MUST contain a section describing the MCP server: what it is, how it runs, how tools are grouped, and the approximate tool count.

- **FR-012:** The document MUST contain a section explaining state and storage: where state lives (`.kbz/`), the Git-native model, and the distinction between committed state and derived cache.

- **FR-013:** The document MUST contain a "where to go next" section with signposted links routing the reader to the appropriate document based on their goal (trying it, understanding the workflow, looking up reference details).

### Structural Requirements

- **FR-014:** Each section MUST follow the inverted pyramid: the most important information appears first, with supporting detail below. A reader who stops after the first paragraph of any section should have the key concept.

- **FR-015:** The document MUST begin with a brief statement of its purpose and intended audience before any substantive content.

- **FR-016:** Section headings MUST be descriptive of content, not generic. Headings like "Overview" or "Details" are not acceptable for body sections.

### Cross-Reference Requirements

- **FR-017:** The document MUST contain at least one link to each of the following documents: Workflow Overview, Getting Started guide, Orchestration and Knowledge, Retrospectives, MCP Tool Reference, Schema Reference, and Configuration Reference.

- **FR-018:** Links to documents that do not yet exist at production time MUST use the planned file path from the design document's inventory (§4.1). These links will resolve as later documents are produced.

### Exclusion Requirements

- **FR-019:** The document MUST NOT contain installation instructions, setup procedures, or step-by-step tutorials. These belong in the Getting Started guide.

- **FR-020:** The document MUST NOT reproduce detailed workflow stage descriptions. It provides a summary; the Workflow Overview provides depth.

- **FR-021:** The document MUST NOT document individual MCP tool parameters, entity schema fields, or configuration keys. It describes systems at conceptual level; reference documents provide specifics.

- **FR-022:** The document MUST NOT contain agent-facing instructions, skill definitions, or role specifications. It is written for human readers.

---

## Non-Functional Requirements

- **NFR-001:** The document MUST be written for an audience of designer-developers and product/design managers who know roughly what Kanbanzai is (having read the README or an introduction) but have not used it.

- **NFR-002:** The document MUST use plain, direct prose in active voice and present tense. No marketing language, no hedging ("might", "could potentially"), no promotional framing.

- **NFR-003:** The document MUST be scannable: a reader skimming headings and first paragraphs should gain a useful mental model of the system without reading every word.

- **NFR-004:** The document MUST pass through the full five-stage editorial pipeline (Write → Edit → Check → Style → Copyedit) as defined in `work/design/documentation-pipeline.md`.

- **NFR-005:** Every factual claim in the document MUST be accurate against the current implementation at the time of the Check stage. The Check stage verifies this.

- **NFR-006:** The document MUST be free of AI writing artifacts as defined in the Style stage's banned vocabulary and pattern lists (`refs/humanising-ai-prose.md`).

- **NFR-007:** The complete document SHOULD be readable in under 15 minutes by a technical reader. This constrains each section to orientation depth, not comprehensive treatment.

---

## Acceptance Criteria

- **AC-001 (FR-001):** The document contains a section that names Kanbanzai as a system, a methodology, and an MCP server, and explains each in plain prose.

- **AC-002 (FR-002):** The document contains a section that explicitly states humans own intent, agents own execution, and documents are the interface — using those terms or direct equivalents.

- **AC-003 (FR-003):** The document contains a section that names the lifecycle stages a feature passes through and provides a summary (table or short descriptions) without exceeding one page of content.

- **AC-004 (FR-004):** The document contains a section listing the document types (design, specification, dev plan) and mapping each to its lifecycle stage.

- **AC-005 (FR-005):** The document contains a section that explains approval gates and states that features can be returned to earlier stages when issues are found.

- **AC-006 (FR-006):** The document contains a section covering bug lifecycle stages and incident tracking in no more than three paragraphs.

- **AC-007 (FR-007):** The document contains a section that names context assembly, roles, skills, task dispatch, parallel execution, and conflict awareness as orchestration capabilities.

- **AC-008 (FR-008):** The document contains a section explaining knowledge entries, their persistence across sessions, and their compounding value.

- **AC-009 (FR-009):** The document contains a section describing the retrospective workflow as: record signals → synthesise → report.

- **AC-010 (FR-010):** The document contains a section that names worktrees, conflict domain analysis, and merge gates as concurrency mechanisms.

- **AC-011 (FR-011):** The document contains a section describing the MCP server, its transport mechanism, and how tools are organised.

- **AC-012 (FR-012):** The document contains a section that identifies `.kbz/` as the state directory, describes the Git-native storage model, and distinguishes committed state from derived cache.

- **AC-013 (FR-013):** The document ends with a signposted section that provides at least three reader-goal-to-document mappings (e.g. "If you want to try it → Getting Started").

- **AC-014 (FR-014, FR-015, FR-016):** The document opens with a purpose and audience statement. Every body section leads with its key concept in the first paragraph. No body section uses a generic heading.

- **AC-015 (FR-017):** The document contains at least one Markdown link to each of: `docs/workflow-overview.md`, `docs/getting-started.md`, `docs/orchestration-and-knowledge.md`, `docs/retrospectives.md`, `docs/mcp-tool-reference.md`, `docs/schema-reference.md`, and `docs/configuration-reference.md`.

- **AC-016 (FR-019, FR-020, FR-021, FR-022):** The document contains no installation steps, no tool parameter tables, no entity field definitions, no configuration key listings, and no agent-facing instructions.

- **AC-017 (NFR-001, NFR-002):** The document uses second person for any direct address, active voice throughout, present tense for descriptions, and contains no marketing language or hedging.

- **AC-018 (NFR-004):** The document has been processed through all five editorial pipeline stages, with a changelog recorded at each stage.

- **AC-019 (NFR-005):** The Check stage report for this document contains zero unresolved findings of severity "error" or "hallucination."

- **AC-020 (NFR-006):** The Style stage report for this document contains zero unresolved findings.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Read the "What is Kanbanzai?" section and verify it names the system, methodology, and MCP server |
| AC-002 | Inspection | Read the collaboration model section and verify the three-part framing (intent, execution, documents) |
| AC-003 | Inspection | Read the stage-gate section and verify it names all lifecycle stages without exceeding one page |
| AC-004 | Inspection | Read the documents section and verify each document type is mapped to a stage |
| AC-005 | Inspection | Read the approval section and verify it describes gates and stage return |
| AC-006 | Inspection | Read the bugs/incidents section and verify it is three paragraphs or fewer |
| AC-007 | Inspection | Read the orchestration section and verify all six named capabilities appear |
| AC-008 | Inspection | Read the knowledge section and verify persistence and compounding are described |
| AC-009 | Inspection | Read the retrospectives section and verify the three-step workflow is stated |
| AC-010 | Inspection | Read the concurrency section and verify worktrees, conflict analysis, and merge gates are named |
| AC-011 | Inspection | Read the MCP server section and verify transport and tool organisation are described |
| AC-012 | Inspection | Read the state/storage section and verify `.kbz/`, Git-native model, and committed-vs-derived distinction |
| AC-013 | Inspection | Read the final section and count reader-goal-to-document mappings (≥ 3) |
| AC-014 | Inspection | Verify opening statement, first-paragraph key concepts, and heading descriptiveness across all sections |
| AC-015 | Automated | Grep the document for Markdown links to each of the 7 required document paths |
| AC-016 | Inspection | Search the document for installation steps, parameter tables, field definitions, config listings, and agent instructions — verify none present |
| AC-017 | Inspection | Spot-check 10 paragraphs for voice, tense, and absence of marketing/hedging language |
| AC-018 | Inspection | Verify editorial pipeline changelogs exist for all 5 stages (Write, Edit, Check, Style, Copyedit) |
| AC-019 | Inspection | Review the Check stage report and verify zero unresolved error/hallucination findings |
| AC-020 | Inspection | Review the Style stage report and verify zero unresolved findings |

---

## Dependencies and Assumptions

- The design document (`work/design/public-release-documentation.md`) is approved. The User Guide's content brief is defined in §6.
- The editorial pipeline infrastructure is implemented: all five roles, all five skills, and the `doc-publishing` stage binding are available.
- The styleguides referenced by the pipeline stages exist: `refs/documentation-structure-guide.md`, `refs/technical-writing-guide.md`, `refs/humanising-ai-prose.md`, `refs/punctuation-guide.md`.
- The Kanbanzai implementation is stable — no major refactors are in flight that would invalidate factual claims during the Check stage.
- Documents linked from the User Guide (Workflow Overview, Getting Started, etc.) do not need to exist at User Guide production time. Links use planned file paths and will resolve as later documents are produced per the production order.