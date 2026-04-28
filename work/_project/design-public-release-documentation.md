# Public Release Documentation Design

| Field | Value |
|-------|-------|
| Author | Sam Phillips |
| Created | 2026-07-19 |

Related:

- `work/design/documentation-for-public-release-proposal.txt` (original proposal)
- `work/design/documentation-pipeline.md` (editorial pipeline — production methodology for this workstream)
- `work/design/public-schema-interface.md` (schema as public contract)
- `work/design/fresh-install-experience.md` (init command and onboarding)
- `work/design/workflow-design-basis.md` §14 (human-AI delegation model)
- `work/design/consistent-front-matter.md` (document metadata conventions)

---

## Overview

This document designs the public-facing documentation set for the Kanbanzai 1.0 release. It defines what documents we produce, who they are for, what each one covers, how they relate to each other, and the editorial principles that govern them all.

This is a plan for documentation, not the documentation itself. It sits alongside other public-release readiness plans (schema stability, installation audit, etc.) as a standalone workstream.

---

## Goals and Non-Goals

### Goals

- Produce a complete, polished public documentation set: 6 new or rewritten documents plus 4 updated reference documents.
- Establish a clear production order so later documents can reference earlier ones.
- Define per-document briefs (purpose, audience, tone, structure) that feed directly into the editorial pipeline's Write stage.
- Position Kanbanzai honestly — articulating costs and benefits without overselling.

### Non-Goals

- Writing the documentation itself. This document designs the set; production follows via the editorial pipeline.
- Redesigning the editorial pipeline. The pipeline is defined in `work/design/documentation-pipeline.md` and used as-is.
- Changing the Kanbanzai workflow or MCP tools. This workstream produces documentation about the system, not changes to it.

---

## Problem Statement

The current documentation was written during development for developers of the system. It serves that audience adequately but fails the public release audience on several dimensions:

1. **No conceptual introduction.** A new user cannot understand what Kanbanzai is, how it differs from existing workflow systems, or why they might want it. The README jumps to installation. The workflow overview assumes familiarity with the stage-gate model.

2. **No guided path.** The getting-started guide covers installation and entity creation but stops at "create a plan, feature, and task." It does not walk a user through the actual workflow: designing, specifying, planning, implementing, and reviewing a feature.

3. **No manual.** There is no document that explains the Kanbanzai methodology — the design-led workflow, the document-driven process, the role of the human versus the AI agents — in a way that a design manager or product manager could read and act on.

4. **Reference documentation exists but is unstyled.** The schema reference, MCP tool reference, and configuration reference are accurate but were written for internal consumption. They need editorial polish, not rewriting.

5. **The README undersells the system.** It lists what agents can do but does not articulate the problems Kanbanzai solves, who it is for, or when it is the right choice.

---

## Design

The design covers editorial principles (§3.1–§3.5), the document inventory and production methodology (§4), per-document content briefs (§5–§10), reference documentation updates (§11), and cross-cutting concerns (§12).

### Design Principles

### 3.1 The Inverted Pyramid

Every document and every section within a document follows an inverted pyramid structure: the most important information first, supporting detail in descending order of importance.

This principle applies at three levels:

- **Content.** Broad concepts before specific details. A section opens with its key point, then elaborates.
- **Tone.** More conversational and accessible at the top of a document or section; more precise and formal as detail increases.
- **Technical proficiency.** Concepts are introduced without jargon at the top; technical specifics (commands, configuration, schema) appear deeper in the document where the reader has chosen to go looking for them.

### 3.2 Audience Assumptions

All public documentation assumes the reader:

- Has used a workflow process before — probably Scrum or Kanban. No need to explain what sprints, backlogs, or boards are.
- Has enough technical proficiency to use Git, create a repository, and work at the command line.
- Is a designer-developer: someone who both designs products and builds them, or who works closely with people who do.

Individual documents refine these assumptions further (see §5–§9).

### 3.3 Show, Don't Explain

Where possible, demonstrate a concept with a concrete example rather than describing it in prose. A three-line transcript of a chat interaction is worth more than a paragraph of explanation.

### 3.4 Link, Don't Repeat

Each concept has a home document. Other documents may reference it briefly and link to the home document for detail. Duplication across documents is kept to the minimum needed for each document to be readable on its own.

### 3.5 Honest Positioning

Documentation must be completely factual about what Kanbanzai does well, what it costs, and when it is not the right choice. Trust is more valuable than persuasion.

---

### Document Inventory

### 4.1 Documents to Produce (New or Rewritten)

| # | Document | Role | Home location |
|---|----------|------|---------------|
| 1 | README | Shop window and quickstart | `README.md` |
| 2 | User Guide | Base document — conceptual overview, links to everything | `docs/user-guide.md` |
| 3 | Workflow Overview | Philosophy, positioning, the full design-to-delivery process | `docs/workflow-overview.md` |
| 4 | Getting Started | Guided walkthrough: install to first feature delivered | `docs/getting-started.md` |
| 5 | Orchestration and Knowledge | Agent coordination, context assembly, knowledge system | `docs/orchestration-and-knowledge.md` |
| 6 | Retrospectives | Using the retrospective system | `docs/retrospectives.md` |

### 4.2 Documents to Keep and Update

These are technically accurate and structurally sound. They need editorial polish to match the new style but not structural rewriting.

| Document | Location | Update scope |
|----------|----------|--------------|
| Schema Reference | `docs/schema-reference.md` | Check → Style → Copyedit pipeline stages; verify completeness |
| MCP Tool Reference | `docs/mcp-tool-reference.md` | Check → Style → Copyedit pipeline stages; verify all actions documented |
| Configuration Reference | `docs/configuration-reference.md` | Check → Style → Copyedit pipeline stages; verify completeness |
| Viewer Agents Guide | `docs/kanbanzai-guide-for-viewer-agents.md` | Check → Style → Copyedit pipeline stages |

### 4.3 Production Order

Documents are produced in this order so that later documents can reference earlier ones:

1. **User Guide** — establishes the conceptual framework everything else references
2. **Workflow Overview** — the methodology in detail
3. **Orchestration and Knowledge** — the technical depth for agentic developers
4. **Retrospectives** — standalone, can be written in parallel with 3
5. **Getting Started** — the guided walkthrough, references all of the above
6. **README** — the shop window, written last when all links are available
7. **Reference doc updates** — Check → Style → Copyedit across schema, tools, config, viewer guide

The Getting Started guide is written after the manual collection because it needs to link into specific sections. The README is written last because it summarises everything.

Each document follows the production order above and then passes through the editorial pipeline (§4.4) before the next document is started. This ensures earlier documents are fully polished before later documents link into them.

### 4.4 Production Methodology: The Editorial Pipeline

Each document is produced through the five-stage editorial pipeline defined in `work/design/documentation-pipeline.md`. The pipeline runs Write → Edit → Check → Style → Copyedit, with each stage operating at a progressively smaller scale (document → section → claim → paragraph → sentence).

The per-document structure tables in §5–§10 serve as the brief for the Write stage: they define purpose, audience, tone, and content. The pipeline stages handle editorial quality — the design principles in §3 (inverted pyramid, audience assumptions, show-don't-tell) are already encoded in the pipeline's `write-docs` and `edit-docs` SKILLs.

**Pipeline configuration by document:**

| Document | Pipeline stages | Checkpoints | Rationale |
|----------|----------------|-------------|-----------|
| README | Full (Write → Edit → Check → Style → Copyedit) | **Hard** after Edit and Copyedit | Public-facing, high-stakes — structural decisions and final polish warrant human review |
| User Guide | Full | Advisory after Edit and Copyedit | Hub document; structural correctness matters but is lower risk than the README |
| Workflow Overview | Full | Advisory after Edit and Copyedit | Methodology document; standard pipeline treatment |
| Getting Started | Full | Advisory after Edit and Copyedit | Code examples get particular attention at the Check stage |
| Orchestration and Knowledge | Full | Advisory after Edit and Copyedit | Standard pipeline treatment |
| Retrospectives | Full | Advisory after Edit and Copyedit | Standard pipeline treatment |
| Reference doc updates | **Check → Style → Copyedit only** | Advisory after Copyedit | Structure is already sound; skip Write and Edit stages |

**Relationship to `update-docs`.** The editorial pipeline handles new document creation and editorial refinement. After the public release set is complete, future incremental updates to keep documentation in sync with code changes use the `update-docs` skill, not the full pipeline. The pipeline is re-invoked only when a document needs substantial revision.

---

### README

### 5.1 Purpose

The README is the shop window. It serves two audiences arriving from different directions:

- **Someone who just found the repo** — needs to understand what this is and whether it is worth their time, in under 60 seconds.
- **Someone ready to try it** — needs the shortest possible path from "I'm interested" to "it's running."

### 5.2 Audience

General technical audience. Assumes no prior knowledge of Kanbanzai. Assumes familiarity with Git, the command line, and the concept of AI-assisted development.

### 5.3 Tone

Conversational at the top, progressively more concrete toward the bottom. Confident but not boastful. Honest about trade-offs.

### 5.4 Structure

| Section | Content |
|---------|---------|
| **Opening paragraph** | One-sentence description, then 2–3 sentences on the core value proposition. No jargon. |
| **What problems does it solve?** | Problem-first framing aimed at two audiences: agentic developers (context loss, agent coordination, knowledge persistence) and design/product managers (structured design process, specification-led quality, approval control). |
| **What does using it look like?** | 5–10 lines showing the rhythm of interaction: write a design → discuss with the agent → approve the spec → say "go" → agents implement → review. Makes the workflow tangible without being a tutorial. |
| **Key capabilities** | Concise feature list, grouped by concern. Favour the orchestration, role-based skills, and knowledge system at least as prominently as the workflow features. |
| **When to use it / when not to** | Honest guidance — see §5.5. |
| **Quickstart** | Condensed install + init + "verify it works" — 10 lines maximum. Links to Getting Started for the full walkthrough. |
| **What gets stored** | Brief `.kbz/` directory description. The current README's tree diagram works well here. |
| **Further reading** | Links to User Guide, Getting Started, Workflow Overview, and reference docs. |

### 5.5 When to Use It / When Not To

This section must be honest about costs and benefits. The framing:

**Use Kanbanzai when:**
- Features regularly take more than one session to implement
- Multiple AI agents work on the same codebase
- Design decisions need to persist and be reviewable
- You care about specification-led quality — catching errors at design time rather than during implementation

**Think twice when:**
- You are building a weekend project or prototype where process overhead exceeds the work itself
- Every feature is simple and self-contained — done in a single session, no coordination needed
- You prefer to work without structured process

**The honest cost:**
- Kanbanzai adds process, time, and token overhead. On a small project, that overhead exceeds the savings.
- On a large project — multiple concurrent features, complex architecture, long-running work — the overhead is repaid through reduced rework, persistent knowledge, and coordinated parallel execution.
- The crossover point is roughly when features regularly span multiple sessions or when you have more than one active work stream.

**The investment:**
- Expect to spend 1–2 hours learning the workflow before it becomes natural.
- The first feature will feel slow. The fifth will not.

### 5.6 Visual Element

The README should include a workflow diagram showing the stage-gate progression (plan → design → specification → dev plan → implementation → integration) with approval gates marked. A Mermaid diagram in Markdown is sufficient — no external images required at launch. A logo or wordmark graphic can be added later if produced.

---

### User Guide (Base Document)

### 6.1 Purpose

The User Guide is the hub of the documentation collection. It provides a high-level orientation to the Kanbanzai system — enough to understand what it is, how the pieces fit together, and where to go for detail. It links to every other document in the collection.

It is not a tutorial (that is the Getting Started guide) and not a deep dive into any single topic (those are the standalone documents). It is the document you read second, after the README, when you have decided Kanbanzai is worth learning.

### 6.2 Audience

Designer-developers and product/design managers. Assumes the reader has read the README or otherwise knows roughly what Kanbanzai is.

### 6.3 Structure

| Section | Content | Links to |
|---------|---------|----------|
| **What is Kanbanzai?** | 2–3 paragraphs. The system, the methodology, the MCP server. What it manages on the user's behalf. | — |
| **The collaboration model** | Human owns intent; agents own execution; documents are the interface. Brief — the Workflow Overview has the detail. | Workflow Overview |
| **The stage-gate workflow** | One-paragraph summary of the six stages plus a diagram. Just enough to orient the reader. | Workflow Overview |
| **Documents drive everything** | What document types exist (proposal, research, design, specification, dev plan) and how they relate to workflow stages. | Workflow Overview |
| **Approval and control** | How the human controls the process through approvals. What approval means at each stage. How features can be returned to earlier stages. The "agile until specification, waterfall after" framing. | Workflow Overview |
| **Bugs and incidents** | Brief overview of the bug lifecycle (report → triage → reproduce → plan → fix → verify → close) and incident tracking. | Schema Reference |
| **Orchestration** | What the orchestration system does: context assembly, role-based skills, task dispatch, parallel execution, conflict awareness. One paragraph each. | Orchestration and Knowledge |
| **The knowledge system** | What knowledge entries are, why they persist, how they compound over time. | Orchestration and Knowledge |
| **Retrospectives** | One paragraph on the retro workflow: record signals → synthesise → report. | Retrospectives |
| **Concurrency and parallel development** | Worktrees, conflict domain analysis, merge gates — one paragraph each. | Orchestration and Knowledge |
| **The MCP server** | What it is, how it runs, 22 tools with multiple actions each. How tools are grouped. | MCP Tool Reference |
| **State and storage** | Where state lives (`.kbz/`), the Git-native model, what is committed vs derived. | Schema Reference, Configuration Reference |
| **Where to go next** | Signposted links: "If you want to try it → Getting Started. If you want to understand the workflow → Workflow Overview. If you want technical reference → Schema / MCP / Config." | All docs |

### 6.4 What This Document Is Not

The User Guide does not:

- Walk through installation or setup (Getting Started does that)
- Explain the workflow methodology in depth (Workflow Overview does that)
- Document individual tools, entity fields, or configuration keys (reference docs do that)
- Serve as a tutorial (Getting Started does that)

It provides just enough context to orient the reader and send them to the right place.

---

### Workflow Overview

### 7.1 Purpose

This is the methodology document. It explains the Kanbanzai workflow from the perspective of a human design manager or product manager — how work flows from proposal to shipped feature, what happens at each stage, and how the human controls the process.

### 7.2 Audience

Design managers and product managers. Assumes experience with agile workflows (Scrum or Kanban). Assumes experience with agentic development, its terms and methods.

### 7.3 Positioning

This document positions Kanbanzai relative to systems the reader already knows:

- **Agile (Scrum/Kanban):** Kanbanzai shares the iterative, flexible approach to design. Work can be returned to earlier stages. Epics go through multiple development cycles. Design is a conversation, not a handoff.
- **Specification-led systems:** After specification approval, implementation follows the spec. The spec is the blueprint; review checks implementation against it. This is deliberate rigidity — it prevents the expensive rework that happens when agents implement without a clear contract.
- **The combined model:** Kanbanzai is agile in design and rigorous in implementation. This is not a contradiction — it is how you get flexibility where flexibility matters (what to build) and consistency where consistency matters (how to build it).

The document should use "specification-led" as the primary label for the implementation phase. It may reference waterfall as a comparison ("if you are familiar with waterfall, the implementation phase will feel familiar") but should not label Kanbanzai itself as waterfall.

### 7.4 Structure

| Section | Content |
|---------|---------|
| **The Kanbanzai workflow** | What it is in one paragraph. The dual nature: agile design, rigorous implementation. |
| **How it compares** | Brief comparison with Scrum/Kanban and specification-led systems. Similarities and differences. Not a sales pitch — an orientation for someone who thinks in those frameworks. |
| **Design-led workflow** | The process of getting from proposal to specification. Drafts, revisions, decisions, the narrowing of alternatives until one design remains. The role of the design manager (human) vs the senior designer (AI agent). |
| **Document-led process** | The four document types — proposals, research, design documents, specifications — and how each drives the workflow. What the system manages vs what it merely stores. |
| **Specification-led implementation** | What happens after spec approval. The dev plan, task decomposition, orchestrated implementation. Why the human's role shifts from design manager to product manager (choosing *when* to implement, not *how*). |
| **Chat-based project management** | How the human interacts with the system through conversation rather than commands. Why chat is more agile than a rigid CLI or web app. The AI fills the project manager, senior designer, and development team roles. |
| **Approval stages and state** | How approval gates work. What states entities pass through. How the human controls progression. How features can be returned to earlier stages. A slightly more technical section — appropriate since readers reaching this depth want precision. |
| **The workflow diagram** | Visual representation of the full stage-gate flow with approval points. |

### 7.5 Content Currently in `docs/workflow-overview.md`

The existing workflow overview contains good material, particularly:

- The collaboration model (§1) — reusable almost verbatim
- The stage overview table (§2) — keep
- The stage detail (§3) — restructure to match the new section ordering
- The document-centric interface (§4) — fold into "Document-led process"
- Common failure modes (§5) — keep, possibly relocate to an appendix or tips section
- Feature and plan lifecycle summaries (§6–§7) — keep as reference

The rewrite restructures this material around the proposal's conceptual framework (design-led → document-led → specification-led → chat-based → approval) rather than the current stage-sequential structure.

---

### Getting Started Guide

### 8.1 Purpose

A hands-on walkthrough that takes a new user from zero to a completed feature. Show, don't tell. Every concept is introduced through a concrete action.

### 8.2 Audience

Designer-developers with enough technical skill to use Git and the command line. **Novice to agentic development** — unlike the manual, this guide should not assume familiarity with MCP, context windows, tool calling, or agent orchestration. Use widely-known terms freely; briefly explain lesser-known concepts (MCP, context assembly, agent roles) when they first appear.

### 8.3 Structure

The guide follows the sequential order of the process:

| Section | Content |
|---------|---------|
| **What you will build** | Brief description of the example feature. Something simple but non-trivial — enough to exercise the full workflow without overwhelming the reader. |
| **Install** | Install from source or binary. Verify. Handle the macOS PATH issue. |
| **Initialise** | `kanbanzai init` in a Git repo. What was created and why (brief). |
| **Connect your editor** | Editor-specific MCP configuration. Verify the server is running. |
| **Create a plan** | First MCP tool call. Explain what a plan is in one sentence. |
| **Write a design** | Create a design document. Show the chat interaction with the AI. This is the first moment the user sees the design-led workflow in action. |
| **Approve the design** | Show the approval step. Explain what approval means (one sentence, link to Workflow Overview). |
| **Create a specification** | Show the spec being generated from the design. |
| **Approve the specification** | Show approval. Note that the design phase is now complete. |
| **Create a dev plan** | Show decomposition into tasks. |
| **Implement** | Show the agent claiming and completing a task. |
| **Review and merge** | Show the review and merge process. |
| **What just happened** | Brief recap: you went from idea to merged code through a structured workflow. Link to the manual collection for deeper understanding. |

### 8.4 The Example Feature

The example should be:

- Simple enough to fit in a guide (no multi-file architectural changes)
- Complex enough to benefit from a design step (not "add a button")
- Relatable to the reader's likely projects
- Self-contained — does not require external services or complex setup

A good candidate: a CLI subcommand or a small utility function with clear inputs, outputs, and edge cases. The specific choice is deferred to implementation — but it must exercise design → spec → implement → review.

### 8.5 Relationship to Current `docs/getting-started.md`

The current getting-started guide is accurate and covers installation, init, and editor integration well. The rewrite preserves this material and extends it significantly: the current guide stops at "create a plan, feature, and task." The new guide continues through the full design-to-delivery workflow.

---

### Orchestration and Knowledge

### 9.1 Purpose

A standalone document for readers who want to understand how Kanbanzai coordinates AI agents and manages persistent knowledge. This is the technical-depth document for agentic developers — the audience most likely to evaluate Kanbanzai based on its orchestration capabilities.

### 9.2 Audience

Agentic developers and power users. Assumes familiarity with AI agent concepts: tool calling, context windows, system prompts, multi-agent coordination.

### 9.3 Structure

| Section | Content |
|---------|---------|
| **The coordination problem** | What goes wrong without structured orchestration: context loss, conflicting edits, repeated rediscovery, inconsistent conventions. Problem-first framing. |
| **Roles and skills** | The role system (`.kbz/roles/`): identity, vocabulary, anti-patterns, tool constraints. The skill system (`.kbz/skills/`): procedures, checklists, evaluation criteria. Stage bindings as the glue. |
| **Context assembly** | How `next` and `handoff` assemble context packets: role instructions, spec fragments, knowledge entries, file paths. How context is scoped to fit the model's context window. |
| **Task dispatch and orchestration** | How the orchestrator claims tasks, spawns sub-agents, and manages parallel execution. The work queue. Dependency ordering. |
| **Conflict awareness** | Conflict domain analysis: how the system flags file-level overlap between tasks before work starts. Worktrees for isolation. |
| **The knowledge system** | Knowledge entries: what they are, how they are contributed (via `finish`), confidence scoring, deduplication, TTL pruning, promotion from session-level to project-level. How knowledge compounds across sessions. |
| **Knowledge governance** | The distinction between decisions, knowledge entries, root cause analyses, specifications, and team conventions. Why governance matters — the knowledge base must not become a dumping ground. |
| **Merge gates** | What is checked before a merge: CI status, review approval, branch health, task completion. Override mechanisms. |
| **Concurrency model** | Worktrees for isolation, branch hygiene, merge strategy. How multiple agents work on the same codebase without stepping on each other. |

---

### Retrospectives

### 10.1 Purpose

A short standalone document explaining how to use Kanbanzai's retrospective system to capture process observations and synthesise them into actionable reports.

### 10.2 Audience

All users. No special technical knowledge required beyond basic Kanbanzai familiarity.

### 10.3 Structure

| Section | Content |
|---------|---------|
| **What retrospectives capture** | Retrospective signals: observations about workflow friction, tool gaps, spec ambiguity, things that worked well. Recorded at task completion via `finish`. |
| **Signal categories** | The categories (workflow-friction, tool-gap, tool-friction, spec-ambiguity, context-gap, decomposition-issue, design-gap, worked-well) with brief descriptions and examples. |
| **Recording signals** | How to include retrospective entries in `finish` calls. Severity levels (minor, moderate, significant). |
| **Synthesising** | Using `retro(action: "synthesise")` to cluster and rank signals across tasks, features, or the whole project. |
| **Generating reports** | Using `retro(action: "report")` to produce a markdown retrospective document. Scoping by plan, feature, or project. Time-range filtering. |
| **When to run a retro** | Suggested cadence: after completing a feature, after a plan milestone, or periodically on long-running projects. |

### 10.4 Size Estimate

This is a short document — likely 3–5 pages. It may feel thin as a standalone, but the alternative (burying it in the User Guide) risks it being missed. Retrospectives are a feature users will not discover unless it is surfaced clearly.

---

### Reference Documentation Updates

The four existing reference documents are kept and updated with a style pass rather than rewritten.

### 11.1 MCP Tool Reference

- Correct the tool count: 22 MCP tools with multiple actions each, not 97 individual tools. The document may enumerate all action-combinations for completeness but must be clear about the distinction.
- Apply the inverted pyramid within each tool section: purpose and most common usage first, edge cases and error conditions last.
- Verify all tools and actions are documented — the tool surface has evolved and the reference may have gaps.

### 11.2 Schema Reference

- Verify against current entity types and field definitions.
- Style pass for consistency with the new editorial tone.

### 11.3 Configuration Reference

- Verify against current `config.yaml` and `local.yaml` fields.
- Style pass.

### 11.4 Viewer Agents Guide

- Style pass.
- Verify accuracy — the viewer project may have evolved since this was written.

---

### Cross-Cutting Concerns

The editorial pipeline (§4.4) enforces most cross-cutting editorial concerns through its stage-specific SKILLs. The concerns below are inputs to those stages, recorded here for completeness and as guidance for the Write stage briefs.

- **Emoji policy.** Use emoji sparingly and only where they add to clarity or aid scanning structure. No emoji in headings, prose, or as bullet markers. Acceptable: status indicators (✅, ❌), section type markers in tables. Enforced by the **Edit** stage (structural tells).
- **Naming consistency.** Follow `AGENTS.md` conventions: **Kanbanzai** (the system/methodology), **`kanbanzai`** (the tool binary), **`.kbz/`** (the instance root). Enforced by the **Check** stage (verify against source of truth) and **Copyedit** stage (consistency).
- **Code examples.** Must be accurate and runnable. No hypothetical tool calls that would fail. Use placeholder format (`FEAT-xxxxx`) where entity IDs vary. Enforced by the **Check** stage (verify against implementation).
- **Versioning.** All public documentation should carry a last-updated date in front matter. Handled during the **Write** stage (document setup).

---

## Alternatives Considered

### Manual Is a Collection, Not a Single Document

The "manual" is a collection of documents with the User Guide as the base document, not a single monolithic file. This gives each topic a clear home, keeps diffs clean, and allows individual documents to be updated without touching the others. The alternative — a single monolithic manual — was rejected because it produces unwieldy diffs and forces unrelated sections to share a file.

### Workflow Overview Absorbs the Methodology Content

Rather than creating a separate "Kanbanzai Methodology" document, the Workflow Overview is expanded to cover philosophy, positioning, design-led workflow, document-led process, specification-led implementation, and chat-based project management. The alternative — a standalone methodology document — was rejected because these topics are tightly coupled and read naturally as a single narrative.

### Approval Stages Live in the Workflow Overview

Approval stages and state transitions are the mechanism that makes the workflow real. They belong in the Workflow Overview as a closing section that gives the reader the precise model after the conceptual narrative has established the "why."

### Bugs and Incidents Live in the User Guide

The bug lifecycle (report → triage → reproduce → plan → fix → verify → close) and incident tracking are covered in the User Guide rather than as a standalone document. They are important for every user but not deep enough to warrant their own document.

### Concurrency Lives in Orchestration and Knowledge

Worktrees, conflict domains, and merge gates are covered in the Orchestration and Knowledge document. They are part of the coordination story and their primary audience is agentic developers who need to understand parallel execution.

### The README Uses Problem-First Positioning

The README's "what problems does it solve" section frames capabilities as solutions to problems the reader is already experiencing, rather than as a feature list. The alternative — leading with a feature list — was rejected because it fails to connect with the reader's existing pain points.

### "Specification-Led" Not "Waterfall"

The implementation phase is described as "specification-led." Waterfall may be referenced as a comparison for readers familiar with the term, but Kanbanzai is not labelled as a waterfall system. The alternative — using "waterfall" as the primary label — was rejected because it carries negative connotations that misrepresent the deliberate design choice.

---

## Dependencies

- **Approved design:** This document must be approved before production begins.
- **Editorial pipeline:** `work/design/documentation-pipeline.md` — the five-stage pipeline (Write → Edit → Check → Style → Copyedit) and its roles, SKILLs, and stage binding. Must be implemented before document production. Status: ✅ implemented.
- **Styleguides:** `refs/documentation-structure-guide.md`, `refs/technical-writing-guide.md`, `refs/humanising-ai-prose.md`, `refs/punctuation-guide.md` — used by the editorial pipeline's stage-specific SKILLs.
- **Implementation as source of truth:** The Check stage verifies all factual claims against the current codebase. The implementation must be stable (no major refactors in flight) during document production.

---

## Open Questions

### Getting Started Example Feature

The specific feature used in the Getting Started walkthrough has not been chosen. It needs to be simple enough to fit in a guide, complex enough to exercise the full workflow, and relatable. Candidates include a CLI subcommand, a configuration option, or a small library utility. Decision deferred to implementation.

### Diagram Tooling → Resolved

Diagrams are produced during the Write stage. The editorial pipeline explicitly excludes diagram generation from its scope (pipeline design §2, Non-Goals). Mermaid is the default format — it renders in GitHub and most editors. If a diagram needs an alternative format, that is a Write-stage decision per document, not a cross-cutting concern.

### Retrospectives: Standalone or User Guide Section?

This design specifies retrospectives as a standalone document (§10). It is a borderline case — the content may be only 3–5 pages. If during writing it proves too thin to stand alone, it can be folded into the User Guide as a section. The standalone structure is the starting point.

### Tool Count Presentation → Resolved

The Check stage of the editorial pipeline verifies all factual claims against the implementation, including tool counts. The MCP tool reference will be corrected during its Check → Style → Copyedit pass (§4.2). The Check stage's structured output will flag the 97-vs-22 discrepancy and classify it as a hallucination finding, ensuring it is resolved before the Style and Copyedit stages run.

---

## Summary

The public release documentation set consists of:

- **6 new or rewritten documents:** README, User Guide, Workflow Overview, Getting Started, Orchestration and Knowledge, Retrospectives
- **4 updated reference documents:** Schema, MCP Tools, Configuration, Viewer Agents Guide
- **Production order:** User Guide → Workflow Overview → Orchestration and Knowledge → Retrospectives → Getting Started → README → Reference updates
- **Production methodology:** Each new document passes through the full five-stage editorial pipeline (Write → Edit → Check → Style → Copyedit). Reference doc updates skip Write and Edit and run Check → Style → Copyedit only. The README uses hard human checkpoints after Edit and Copyedit; all other documents use advisory checkpoints.

The editorial framework is the inverted pyramid applied to content, tone, and technical proficiency — encoded in the pipeline's `write-docs` and `edit-docs` SKILLs. The audience is designer-developers and design/product managers. The positioning is honest: Kanbanzai adds overhead that pays for itself on projects of sufficient scale and complexity, and the documentation says so clearly.