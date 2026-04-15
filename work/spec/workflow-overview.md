# Workflow Overview: Specification

| Document | Workflow Overview Specification |
|----------|--------------------------------|
| Status   | Draft |
| Feature  | FEAT-01KP8-T4HSX6QZ |
| Created  | 2026-04-15 |
| Design   | `work/design/public-release-documentation.md` §7 (DOC P22-documentation-for-public-release/design-public-release-documentation) |

---

## Overview

This specification defines the Workflow Overview (`docs/workflow-overview.md`) — the methodology document of the public documentation collection. The Workflow Overview explains how work flows from proposal to shipped feature in Kanbanzai, what happens at each stage, who decides what, and how the human controls the process through approvals and conversation.

The document is the second produced in the P22 production order. It is the primary target for readers who want to understand the Kanbanzai workflow before trying it. The User Guide links to it as the authoritative source on workflow methodology.

---

## Scope

### In scope

- A single Markdown document at `docs/workflow-overview.md`, replacing the existing file.
- The conceptual framework defined in design §7.4: the dual nature of the workflow (agile design, rigorous implementation), comparison with known methodologies, and the five thematic sections (design-led → document-led → specification-led → chat-based → approval stages).
- Restructuring and incorporating reusable material from the existing `docs/workflow-overview.md` as identified in design §7.5.
- A workflow diagram showing the full stage-gate flow with approval points.
- Cross-references to the User Guide, Getting Started guide, Orchestration and Knowledge, and reference documents.
- Production through the full five-stage editorial pipeline (Write → Edit → Check → Style → Copyedit) with advisory checkpoints after Edit and Copyedit.

### Out of scope

- Installation, setup, or tutorial content (Getting Started guide owns this).
- Orchestration internals: roles, skills, context assembly, task dispatch, conflict domains (Orchestration and Knowledge document owns these).
- Individual tool parameters, entity fields, or configuration keys (reference documents own these).
- Agent-facing instructions, skill definitions, or role specifications (internal, not public documentation).
- The retrospective system (Retrospectives document owns this).
- Diagrams or visual assets beyond the workflow diagram produced inline during the Write stage.

---

## Functional Requirements

### Framing and Positioning

- **FR-001:** The document MUST open with a framing section that describes the Kanbanzai workflow in one paragraph, stating its dual nature: agile in design and rigorous in implementation.

- **FR-002:** The document MUST contain a section comparing the Kanbanzai workflow with systems the reader already knows: agile (Scrum/Kanban) for the design phases, and specification-led systems for the implementation phases. The section MUST state similarities and differences without promotional framing.

- **FR-003:** The comparison section MUST use "specification-led" as the primary label for the implementation phase. It MAY reference waterfall as a familiar comparison point but MUST NOT label Kanbanzai itself as waterfall.

### Design-Led Workflow

- **FR-004:** The document MUST contain a section titled "Design-led workflow" (or direct equivalent) describing the process from proposal to approved specification: drafts, revisions, decisions, the narrowing of alternatives until one design remains.

- **FR-005:** The design-led workflow section MUST describe the roles of the human (design manager, decision-maker) and the AI agent (senior designer, researcher, drafter) during the design phases.

- **FR-006:** The design-led workflow section MUST cover the planning, design, and specification stages as a continuous design conversation, not as isolated sequential steps.

### Document-Led Process

- **FR-007:** The document MUST contain a section titled "Document-led process" (or direct equivalent) describing the four document types that drive the workflow and how each relates to workflow progression.

- **FR-008:** The document-led process section MUST name and describe each document type: design documents, specifications, development plans, and reports (review reports). For each type, it MUST state what the document captures and what its approval unlocks.

- **FR-009:** The document-led process section MUST explain the distinction between what the system manages (document records, approval status, content hashes) and what it merely stores (the Markdown files themselves).

### Specification-Led Implementation

- **FR-010:** The document MUST contain a section titled "Specification-led implementation" (or direct equivalent) describing what happens after specification approval: dev plan creation, task decomposition, orchestrated implementation.

- **FR-011:** The specification-led implementation section MUST explain the shift in the human's role from design manager (shaping what to build) to product manager (choosing when to implement and reviewing results), and why this shift occurs.

- **FR-012:** The specification-led implementation section MUST describe the dev plan as the bridge between the specification and implementation: how the specification is broken into tasks with dependencies and traceability.

### Chat-Based Project Management

- **FR-013:** The document MUST contain a section titled "Chat-based project management" (or direct equivalent) describing how the human interacts with the system through conversation rather than commands or a dedicated interface.

- **FR-014:** The chat-based project management section MUST explain why conversation is more agile than a rigid CLI or web interface for the design-phase interactions.

- **FR-015:** The chat-based project management section MUST describe the AI's composite role: the agent fills the project manager, senior designer, and development team roles, while the human retains decision authority.

### Approval Stages and State

- **FR-016:** The document MUST contain a section titled "Approval stages and state" (or direct equivalent) describing how approval gates work, what states entities pass through, and how the human controls progression.

- **FR-017:** The approval stages section MUST list the feature lifecycle stages: proposed → designing → specifying → dev-planning → developing → reviewing → done.

- **FR-018:** The approval stages section MUST describe how features can move backward — when a document is revised, when review finds blocking problems — and that backward movement is a designed mechanism, not a failure.

- **FR-019:** The approval stages section MUST describe the plan lifecycle stages: proposed → designing → active → done, with superseded and cancelled as terminal alternatives.

- **FR-020:** The approval stages section MUST describe the document prerequisites at each gate: approved design unlocks specification, approved specification unlocks dev planning, approved dev plan unlocks development, completed tasks unlock review, approved review unlocks done.

### Workflow Diagram

- **FR-021:** The document MUST contain a visual representation of the full stage-gate flow with approval points clearly marked. This MAY be a text-based diagram (ASCII or Mermaid) or a described figure.

### Common Failure Modes

- **FR-022:** The document MUST contain a section describing common workflow failure modes and why the stage-gate structure prevents them. The section MUST cover at minimum: creating tasks without a specification, making architecture decisions without an approved design, skipping dev planning, and implementing before gates are satisfied.

### Structural Requirements

- **FR-023:** Each section MUST follow the inverted pyramid: the most important information appears first, with supporting detail below. A reader who stops after the first paragraph of any section should have the key concept.

- **FR-024:** The document MUST begin with a brief statement of its purpose and intended audience before any substantive content.

- **FR-025:** Section headings MUST be descriptive of content, not generic. Headings like "Overview" or "Details" are not acceptable for body sections.

### Cross-Reference Requirements

- **FR-026:** The document MUST contain at least one link to each of the following documents: User Guide (`docs/user-guide.md`), Getting Started guide (`docs/getting-started.md`), Orchestration and Knowledge (`docs/orchestration-and-knowledge.md`).

- **FR-027:** The document MUST contain a "where to go next" section at the end that routes readers to the appropriate document based on their goal.

- **FR-028:** Links to documents that do not yet exist at production time MUST use the planned file path from the design document's inventory (§4.1). These links will resolve as later documents are produced.

### Exclusion Requirements

- **FR-029:** The document MUST NOT contain installation instructions, setup procedures, or step-by-step tutorials. These belong in the Getting Started guide.

- **FR-030:** The document MUST NOT describe orchestration internals (context assembly algorithms, role resolution, skill selection, handoff prompt generation). These belong in the Orchestration and Knowledge document.

- **FR-031:** The document MUST NOT document individual MCP tool parameters, entity schema fields, or configuration keys. It describes workflow methodology; reference documents provide specifics.

- **FR-032:** The document MUST NOT contain agent-facing instructions, skill definitions, or role specifications. It is written for human readers.

- **FR-033:** The document MUST NOT duplicate the User Guide's orientation-level coverage of non-workflow topics (knowledge system, incident tracking, MCP tool groups). It may reference those topics briefly when relevant to workflow but MUST link to the User Guide or appropriate document rather than restating them.

---

## Non-Functional Requirements

- **NFR-001:** The document MUST be written for an audience of design managers and product managers who have experience with agile workflows (Scrum or Kanban) and familiarity with agentic development.

- **NFR-002:** The document MUST use plain, direct prose in active voice and present tense. No marketing language, no hedging ("might", "could potentially"), no promotional framing.

- **NFR-003:** The document MUST be scannable: a reader skimming headings and first paragraphs should gain a useful mental model of the workflow methodology without reading every word.

- **NFR-004:** The document MUST pass through the full five-stage editorial pipeline (Write → Edit → Check → Style → Copyedit) as defined in `work/design/documentation-pipeline.md`.

- **NFR-005:** Every factual claim in the document MUST be accurate against the current implementation at the time of the Check stage. The Check stage verifies this.

- **NFR-006:** The document MUST be free of AI writing artifacts as defined in the Style stage's banned vocabulary and pattern lists (`refs/humanising-ai-prose.md`).

- **NFR-007:** The document SHOULD be readable in under 20 minutes by a technical reader. This is longer than the User Guide's 15-minute target because the Workflow Overview provides depth on a single topic rather than breadth across all topics.

- **NFR-008:** The document MUST position the Kanbanzai workflow relative to systems the reader already knows, as specified in design §7.3. It MUST NOT assume the reader has used Kanbanzai before.

---

## Acceptance Criteria

- **AC-001 (FR-001):** The document opens with a framing section that describes the workflow's dual nature — agile in design, rigorous in implementation — in one paragraph.

- **AC-002 (FR-002, FR-003):** The document contains a comparison section that names agile (Scrum/Kanban) and specification-led systems, states similarities and differences, uses "specification-led" as the primary label, and does not label Kanbanzai as waterfall.

- **AC-003 (FR-004, FR-005, FR-006):** The document contains a design-led workflow section that describes proposal-to-specification as a continuous design conversation, names the human role (design manager) and agent role (senior designer), and covers planning, design, and specification stages.

- **AC-004 (FR-007, FR-008, FR-009):** The document contains a document-led process section that names design documents, specifications, development plans, and reports; states what each document's approval unlocks; and distinguishes system-managed records from the Markdown files themselves.

- **AC-005 (FR-010, FR-011, FR-012):** The document contains a specification-led implementation section that describes post-spec workflow, explains the human role shift from design manager to product manager, and describes the dev plan as the bridge between specification and implementation.

- **AC-006 (FR-013, FR-014, FR-015):** The document contains a chat-based project management section that describes conversational interaction, explains why conversation suits the design phases, and names the AI's composite role (project manager, senior designer, development team).

- **AC-007 (FR-016, FR-017, FR-018, FR-019, FR-020):** The document contains an approval stages section that lists all feature lifecycle stages (proposed through done), all plan lifecycle stages (proposed through done plus terminal alternatives), describes backward movement as designed, and maps each gate to its document prerequisite.

- **AC-008 (FR-021):** The document contains a visual representation of the stage-gate flow with approval points marked.

- **AC-009 (FR-022):** The document contains a failure modes section covering at minimum: tasks without a spec, architecture decisions without a design, skipping dev planning, and implementing before gates.

- **AC-010 (FR-023, FR-024, FR-025):** The document opens with a purpose and audience statement. Every body section leads with its key concept in the first paragraph. No body section uses a generic heading.

- **AC-011 (FR-026):** The document contains at least one Markdown link to each of: `docs/user-guide.md`, `docs/getting-started.md`, `docs/orchestration-and-knowledge.md`.

- **AC-012 (FR-027):** The document ends with a signposted section providing reader-goal-to-document mappings.

- **AC-013 (FR-029, FR-030, FR-031, FR-032, FR-033):** The document contains no installation steps, no orchestration internals (context assembly, role resolution, skill selection), no tool parameter tables, no entity field definitions, no agent-facing instructions, and no duplicated User Guide orientation material.

- **AC-014 (NFR-001, NFR-002):** The document uses active voice throughout, present tense for descriptions, and contains no marketing language or hedging.

- **AC-015 (NFR-004):** The document has been processed through all five editorial pipeline stages, with a changelog recorded at each stage.

- **AC-016 (NFR-005):** The Check stage report for this document contains zero unresolved findings of severity "error" or "hallucination."

- **AC-017 (NFR-006):** The Style stage report for this document contains zero unresolved findings.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Read the opening section and verify dual-nature framing in one paragraph |
| AC-002 | Inspection | Read the comparison section; verify agile and specification-led are named, "specification-led" is primary label, "waterfall" is not applied to Kanbanzai |
| AC-003 | Inspection | Read the design-led workflow section; verify continuous conversation framing, named roles, and coverage of planning/design/specification stages |
| AC-004 | Inspection | Read the document-led process section; verify four document types named, approval consequences stated, managed-vs-stored distinction present |
| AC-005 | Inspection | Read the specification-led implementation section; verify post-spec workflow, role shift explanation, dev plan as bridge |
| AC-006 | Inspection | Read the chat-based section; verify conversational interaction described, rationale stated, composite AI role named |
| AC-007 | Inspection | Read the approval stages section; verify all feature stages listed, all plan stages listed, backward movement described, gate-to-document mapping present |
| AC-008 | Inspection | Verify a visual workflow diagram is present with approval points marked |
| AC-009 | Inspection | Read the failure modes section; verify the four minimum failure modes are covered |
| AC-010 | Inspection | Verify opening statement, first-paragraph key concepts, and heading descriptiveness across all sections |
| AC-011 | Automated | Grep the document for Markdown links to `user-guide.md`, `getting-started.md`, `orchestration-and-knowledge.md` |
| AC-012 | Inspection | Read the final section and verify reader-goal-to-document mappings are present |
| AC-013 | Inspection | Search the document for installation steps, orchestration internals, parameter tables, field definitions, agent instructions — verify none present |
| AC-014 | Inspection | Spot-check 10 paragraphs for voice, tense, and absence of marketing/hedging language |
| AC-015 | Inspection | Verify editorial pipeline changelogs exist for all 5 stages (Write, Edit, Check, Style, Copyedit) |
| AC-016 | Inspection | Review the Check stage report and verify zero unresolved error/hallucination findings |
| AC-017 | Inspection | Review the Style stage report and verify zero unresolved findings |

---

## Dependencies and Assumptions

- The design document (`work/design/public-release-documentation.md`) is approved. The Workflow Overview's content brief is defined in §7.
- The User Guide (`docs/user-guide.md`) has been produced and is available for cross-referencing. The Workflow Overview is the second document in the production order.
- The existing `docs/workflow-overview.md` contains reusable material as identified in design §7.5. The Write stage should restructure this material around the new conceptual framework rather than writing from scratch.
- The editorial pipeline infrastructure is implemented: all five roles, all five skills, and the `doc-publishing` stage binding are available.
- The styleguides referenced by the pipeline stages exist: `refs/documentation-structure-guide.md`, `refs/technical-writing-guide.md`, `refs/humanising-ai-prose.md`, `refs/punctuation-guide.md`.
- The Kanbanzai implementation is stable — no major refactors are in flight that would invalidate factual claims during the Check stage.
- Documents linked from the Workflow Overview (Getting Started, Orchestration and Knowledge) do not need to exist at production time. Links use planned file paths and will resolve as later documents are produced per the production order.