# Planning and Administration Layer: Research Report

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-01T10:40:52Z          |
| Status | Draft                         |
| Author | researcher                    |

## Research Question

What planning and administration functions should sit in front of Kanbanzai's existing project-management and orchestrator layer to support intake, scoping, prioritisation, and cross-plan awareness for human-AI hybrid teams? What should that layer look like conceptually — should it be modelled on named human jobs, decomposed functional roles, or something else? And where should it live: inside the existing CLI/MCP, or as a separate visual tool?

This research was requested as a brainstorming exercise to explore the blank space upstream of Kanbanzai's current tool surface before any implementation decisions are made. No design or architecture decisions are being taken at this stage.

## Scope and Methodology

**In scope:**
- The cognitive functions needed to bridge raw ideas ("we should build X") and structured plan/batch entities
- How existing tools (Linear, Trello Butler, GitHub Projects, Notion) handle planning intake and where they'd fail for human-AI hybrid teams
- What pre-computer clerical and administrative roles can tell us about the cognitive functions a planning-admin AI should perform
- Whether this layer should be modelled on named human jobs or on decomposed functional roles
- Whether this should live inside the existing CLI/MCP or as a separate visual UI

**Out of scope:**
- Detailed data model or API design (implementation detail)
- Specific tool names or MCP tool signatures (implementation detail)
- Integration with specific external systems (Jira, Linear, etc.)
- Cost analysis or build-vs-buy decisions
- Timeline, staffing, or project planning for building this layer

**Methodology:** Literature review (academic papers on AI planning assistants, human-AI collaboration patterns), competitive analysis (project management tools and their intake UX), historical analogy (pre-computer clerical and administrative roles), and internal system analysis (current Kanbanzai MCP tool surface and stage bindings). Evidence graded by source type, recency, and authority.

## Findings

### Finding 1: Kanbanzai's Current Surface Ends at Structured Entity Creation

The existing MCP tool surface provides comprehensive management of entities once they exist: lifecycle transitions, document registration, task decomposition, parallel dispatch, conflict detection, review orchestration, merge gates, and knowledge curation. But there is no tool-mediated path from "we have an idea" to "we have a batch with well-scoped features."

The `kanbanzai-planning` skill (`.agents/skills/kanbanzai-planning/SKILL.md`) describes a human-led conversational process — the agent asks clarifying questions, reflects scope back, flags sizing issues — but this conversation produces only an informal scope agreement. Entity creation requires a separate, explicit call to `entity(action: create)`. The conversation and the creation step are disconnected.

The only entity-creation path is direct: `entity(action: create, type: plan/batch/feature, name: ..., summary: ...)`. This is form-like — the agent fills in fields and submits. There is no structured intake, no scoping wizard, no guided decomposition into batches/features, and no cross-plan awareness tool that surfaces dependencies or conflicts between strategic plans.

Source: Codebase analysis — `internal/mcp/entity_tool.go` (entity creation), `.agents/skills/kanbanzai-planning/SKILL.md` (planning conversation guide). Primary, current.

### Finding 2: The Gap Is Not in Tool Count but in Workflow Integration

Kanbanzai already has 21 MCP tools across entity management, document intelligence, orchestration, and knowledge curation. The gap is not a missing tool — it's the absence of an intake workflow that bridges unstructured human intent and structured entity creation.

Specifically, five cognitive functions have no tool support:

| Cognitive Function | What Exists | What's Missing |
|---|---|---|
| **Idea capture** | Nothing — ideas live in human memory or external notes | A lightweight capture point that records intent before it's shaped into entities |
| **Scope negotiation** | `kanbanzai-planning` skill (conversational, agent-led) | Tool support that structures the conversation output into a draft plan/batch/feature proposal the human can revise |
| **Dependency surfacing** | `conflict` tool (file-level within a feature) | Cross-plan dependency detection — "Plan A's batch B2 depends on Plan C's feature F3" |
| **Prioritisation** | Task queue ordering (estimate × age, within a feature) | Cross-plan/batch prioritisation — "which of these three plans should we work on first?" |
| **Progress synthesis** | `status()` (per-entity dashboard) | Portfolio-level synthesis — "across all active plans, what's blocked, what's close to done, and what needs attention?" |

These are the functions that bridge "we should build something" and "here are your structured entities, go implement." They are fundamentally *administrative* rather than *project-management* or *orchestration* — they deal with information intake, organisation, and surfacing rather than execution coordination.

Source: Codebase analysis — MCP tool inventory (21 tools via `internal/mcp/`), `kanbanzai-planning` skill content. Primary, current.

### Finding 3: Existing PM Tools Solve for Human-Only Teams, Not Human-AI Hybrid Teams

**Linear** provides a keyboard-first, speed-optimised issue tracker with excellent keyboard shortcuts, quick-add, and command-palette interaction. Its intake is form-like but fast — `Cmd+K` opens a command palette, issue creation prefills from context. However, Linear has no concept of AI agents as team members. Its workflow assumes all work is performed by humans who need notifications, assignments, and status updates. The rich notification and assignment model that makes Linear work for human teams would be noise for an AI orchestrator that manages task dispatch internally.

**Trello Butler** provides rule-based automation (triggers + actions on cards and boards). It automates repetitive clerical tasks — moving cards when due dates pass, assigning members on label changes, posting checklist reminders. Butler's approach is a good conceptual match for the clerical/admin functions Kanbanzai needs: automated triage, status propagation, reminder generation. However, Butler's rule engine is designed for human card management, not for generating structured plans from unstructured intake. The rules operate on cards that already exist — they don't help create them from intent.

**GitHub Projects** provides spreadsheet-like views with flexible grouping and filtering. Its strength is in cross-repository visibility and custom field-based workflows. But like Linear, it assumes humans are the primary actors — AI agents aren't first-class participants.

**Notion** provides the richest intake experience: free-form pages that can contain databases, checklists, and structured blocks. This is the closest analogue to what Kanbanzai needs — an unstructured-to-structured intake surface. However, Notion's AI features (Q&A, auto-fill) are designed for content generation, not for workflow orchestration or task dispatch.

The common thread: all existing tools treat planning as a human activity supported by forms and views. None treat planning as a conversation between a human and an AI agent where the agent actively structures, challenges, and refines scope. For human-AI hybrid teams, the intake surface needs to be conversational (matching how humans describe intent) but produce structured output (matching how AI agents consume work).

Source: Competitive analysis — Linear documentation (`linear.app/docs`), Trello Butler documentation (`trello.com/butler`), GitHub Projects documentation (`docs.github.com/en/issues/planning-and-tracking`), Notion AI documentation (`notion.so/product/ai`). Secondary, 2025–2026.

### Finding 4: Pre-Computer Clerical Roles Map Well to the Missing Cognitive Functions

Before computers, large organisations employed dedicated clerical staff whose job was to manage information flow between decision-makers and execution teams. Examining these roles reveals a natural decomposition of the intake/admin space:

**The Correspondence Clerk (intake + triage):** Received incoming letters, requests, and reports; sorted them by priority and category; routed them to the appropriate decision-maker; maintained a correspondence log. In Kanbanzai terms: captures unstructured intent, classifies it (is this a plan, a batch, a feature, a bug?), and routes it to the right workflow stage.

**The Committee Secretary (scope negotiation):** Attended meetings, took minutes, circulated draft decisions for review, incorporated amendments, and published final resolutions. In Kanbanzai terms: structures the planning conversation into a draft plan/batch, surfaces it for human review, incorporates feedback, and finalises the scope agreement.

**The Schedule Clerk (dependency + priority tracking):** Maintained the master schedule showing what work was in progress, what was blocked waiting for what, and what was coming next. Updated the schedule as new work arrived and priorities shifted. In Kanbanzai terms: cross-plan dependency detection, priority ordering, and portfolio-level progress synthesis.

**The Filing Clerk (information organisation):** Maintained the registry of decisions, correspondence, and records; ensured everything was findable later. In Kanbanzai terms: this is what Kanbanzai's document intelligence system already does — but the intake layer needs to ensure information is filed *as it arrives*, not retroactively.

These roles share a common characteristic: they are **information-flow roles**, not decision-making roles. They don't decide what to build — they structure, organise, and surface information so decision-makers (the human) can decide effectively. This maps precisely to what Kanbanzai needs upstream of its orchestrator. The orchestrator is the foreman (dispatches work, tracks completion). The intake layer is the clerk (captures intent, structures it, surfaces dependencies and priorities).

Source: Historical analogy — JoAnne Yates, *Control Through Communication: The Rise of System in American Management* (Johns Hopkins University Press, 1989); Delphine Gardey, *Écrire, calculer, classer: Comment une révolution de papier a transformé les sociétés contemporaines* (La Découverte, 2008). Secondary, academic.

### Finding 5: Named Human Jobs vs. Decomposed Functional Roles — A Design Tension

Modelling this layer on named human jobs (a "Clerk" role, a "Secretary" role) has the advantage of being immediately comprehensible to humans — people know how to delegate to a clerk or a secretary. The user prompt highlights: "Humans struggle with parallel dependency tracking (stated limitation) → this maps to the historical role of a project clerk who maintained the master schedule." This is a strong argument for job-named roles: they match human delegation instincts.

However, Kanbanzai's existing role system models roles on *functions* within a workflow stage, not on human jobs. The architect doesn't "do architecture" — they write design documents at the designing stage. The reviewer-conformance doesn't "do conformance review" — they review code against specs at the reviewing stage. Roles are stage-bound and skill-bound, with narrow responsibilities.

There are three candidate approaches:

**Option A: Single "Admin Assistant" role** — inheriting from a clerk archetype, owning the full intake pipeline (capture, scope negotiation, dependency surfacing, prioritisation). This matches how humans expect to delegate ("my assistant handles it") and keeps the role count low. The assistant would use multiple skills (intake, scoping, scheduling) but present a single identity.

**Option B: Decomposed functional roles** — separate Intake Clerk (capture + triage), Scoping Facilitator (scope negotiation), and Schedule Analyst (dependency + priority). This matches Kanbanzai's existing single-skill-per-role pattern but triples the role count. Each role would be bound to a specific intake sub-stage.

**Option C: Conversational surface with no new roles** — the intake happens through structured conversation tools (a "planning intake" MCP tool that guides the human through scoping questions and produces a draft plan/batch) without dedicated roles. The existing `kanbanzai-planning` skill already describes this pattern — the agent facilitates planning conversation as its current role (orchestrator, architect, etc.). The tool does the structuring; the agent does the facilitating.

**Trade-off analysis:**

| Criterion | A: Admin Assistant | B: Decomposed Roles | C: Tools Only |
|-----------|-------------------|--------------------|--------------------|
| Human delegation clarity | High — "my assistant" | Medium — "which clerk?" | Low — no named identity |
| Consistency with existing role system | Low — multi-skill role | High — single-skill roles | High — no new roles needed |
| Role count impact | +1 role, +3-4 skills | +3-4 roles, +3-4 skills | 0 roles, +tools |
| Conversational fluency | High — single persona | Medium — handoffs between roles | High — agent stays in current role |
| Complexity of implementation | Medium | High | Low |
| Matches planning UI shape | Favours conversational | Favours form-based | Favours tool-mediated |

Option C (conversational surface with no new roles) aligns best with Kanbanzai's existing architecture — the `kanbanzai-planning` skill already describes agent-facilitated planning conversation. Adding MCP tools for intake would augment the existing conversational pattern rather than replacing it with a new role system.

However, Option A (Admin Assistant) has a distinct advantage: when humans encounter friction — "I can't track all my plans" — they naturally think "I need an assistant." Naming the role after this instinct makes the system legible. The tension is between architectural consistency (Option C or B) and human mental-model match (Option A).

Source: Internal system analysis — `.kbz/roles/` (18 role files with single-skill binding pattern), `.kbz/stage-bindings.yaml` (stage-to-role mapping). Primary, current.

### Finding 6: The CLI/MCP vs. Visual UI Question Depends on the Primary Interaction Mode

If the intake layer is primarily **conversational** (human describes intent, agent structures it, human reviews and approves), then CLI/MCP is the natural home — the conversation already happens there, and adding intake tools to the existing MCP server keeps the interaction in one place.

If the intake layer is primarily **visual/navigational** (human browses plans, drags priorities, sees dependency graphs), then a separate visual UI is warranted — MCP tools return structured data but cannot render interactive visualisations.

The pre-computer clerical analogy suggests a hybrid: the clerk (MCP tools) processes information and surfaces it, but the human decision-maker might want a visual dashboard for the portfolio-level view. This is analogous to how Kanbanzai currently uses `status()` for dashboards — the tool returns structured data, but the human viewing it sees a markdown table. A visual UI would be a rendering layer on top of the same structured data.

The strongest argument for staying within CLI/MCP initially: Kanbanzai's entire workflow is CLI/MCP-mediated. Splitting intake into a separate visual tool creates a context-switching cost that may reduce adoption. The strongest argument for a separate visual UI: dependency graphs and portfolio-level views are inherently spatial — they benefit from visual rendering in ways that markdown tables cannot match.

Source: Internal system analysis — current MCP tool surface (21 tools, all text-structured), `status()` tool output format (synthesised dashboards as markdown). Primary, current.

## Trade-Off Analysis

The core design decision is not "what tools to build" but "what cognitive model should organise the intake/admin layer." The tools follow the model.

| Criterion | Job-Named Roles (Option A) | Decomposed Functional Roles (Option B) | Tools-Only Surface (Option C) |
|-----------|---------------------------|---------------------------------------|-------------------------------|
| **Architectural fit** with existing role system | Low — breaks single-skill pattern | High — matches existing pattern | High — no role changes |
| **Human mental model** | High — "my assistant handles it" | Medium — functional but abstract | Low — no persona to delegate to |
| **Implementation cost** | Medium | High (3-4 roles × skill files × stage bindings) | Low (MCP tools only) |
| **Extensibility** | Medium — assistant scope grows unboundedly | High — new functions are new roles | High — new functions are new tools |
| **Discovery** | High — single entry point | Medium — need to know which clerk | Low — need to know which tool |
| **Human gate placement** | Natural: "assistant proposes, I approve" | Complex: which stages need gates? | Flexible: gates attach to tool calls |
| **Matches Kanbanzai's conversation-first pattern** | Yes — conversational delegation | No — form-based handoffs | Yes — tool-mediated conversation |

## Recommendations

### Recommendation 1: Adopt a Tools-First Approach for the Intake Layer

**Recommendation:** Start with conversational MCP tools for intake, scoping, and cross-plan awareness — without adding new roles. Model the tools on the five missing cognitive functions (Finding 2) and validate their usefulness before deciding whether role personas would improve the experience.

**Confidence:** Medium
**Based on:** Findings 2, 5, and 6
**Conditions:** This recommendation assumes the existing `kanbanzai-planning` conversational pattern is the right foundation and that tools will slot into it naturally. If the human-AI planning conversation turns out to be significantly different from the current conversational pattern (e.g., more form-like, or more visual), reconsider.

### Recommendation 2: Model the Tool Surface on Clerical Functions, Not on PM Feature Lists

**Recommendation:** Rather than implementing "Gantt charts for AI agents" or "resource levelling," build tools that perform the five clerical functions: capture, scope negotiation, dependency surfacing, prioritisation, and progress synthesis (Finding 4). These are information-flow functions — they structure and surface rather than decide. This keeps the human firmly in the decision-making role while giving them the information they need to decide well.

**Confidence:** High
**Based on:** Findings 3 and 4
**Conditions:** Applies regardless of whether tools live in CLI/MCP or a visual UI.

### Recommendation 3: Keep the Intake Layer in CLI/MCP Initially

**Recommendation:** Build the intake tools as MCP tools within the existing CLI server. The conversational planning pattern already exists there. A visual UI can be added later as a rendering layer on the same structured data if portfolio-level views prove necessary.

**Confidence:** Medium
**Based on:** Finding 6
**Conditions:** If early use reveals that portfolio-level views (dependency graphs, cross-plan status) are essential for decision-making and markdown rendering is insufficient, escalate to a visual UI investigation. A small prototype visual dashboard consuming the same MCP data could validate or refute the need cheaply.

### Recommendation 4: Defer the Role Question Until the Tool Shape Is Validated

**Recommendation:** Build the intake MCP tools first. Use them through the existing orchestrator role (or architect, depending on the lifecycle stage of the conversation). After a batch or two of real planning intake, evaluate: does the absence of a named persona (Admin Assistant) cause friction? Do humans struggle to know which tool to use? If yes, introduce an Admin Assistant role that wraps the intake tools behind a single conversational identity (Option A). If no, the tools-only approach is sufficient.

**Confidence:** Medium
**Based on:** Finding 5
**Conditions:** This is a deliberate deferral, not a decision against roles. The role question is a UX question — is a named persona necessary for humans to delegate effectively? — and UX questions are best answered with evidence from use.

## Limitations

- **No user research:** The analysis of what humans need from a planning intake layer is based on system analysis and analogy, not on direct observation of users attempting to plan work in Kanbanzai. The five missing cognitive functions (Finding 2) are derived from tool-gap analysis, not from user interviews or task analysis. They may be incomplete or misprioritised.

- **No prototyping:** The recommendation to build tools before roles assumes the tools will be used conversationally through existing roles. If the tools turn out to need a more structured interaction pattern (e.g., multi-turn wizards with state), the conversational assumption may fail and roles may be needed sooner.

- **Competitive analysis limited to documentation:** The analysis of Linear, Trello Butler, GitHub Projects, and Notion is based on documentation and public descriptions, not hands-on use. Features that are discoverable only through use (keyboard shortcuts, AI features, integration behaviour) may have been missed.

- **Historical analogy is suggestive, not prescriptive:** Pre-computer clerical roles faced different constraints (paper, manual routing, physical proximity) than AI-mediated workflows. The mapping of clerical functions to Kanbanzai tools is an organising metaphor, not a proven design pattern. Over-adherence to the analogy could produce tools that solve 19th-century problems instead of 21st-century ones.

- **Scope of "planning admin" is fuzzy:** This research treats planning and administration as a single upstream layer. In practice, there may be a meaningful distinction between *planning* (strategic scoping and prioritisation) and *administration* (information triage, routing, filing) that warrants separate treatment. Further decomposition of this space may reveal distinct roles or tools.
