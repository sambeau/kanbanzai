# Shaping Layer: Entity Model and Data Flow

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-01T13:22:20Z          |
| Status | Draft                         |
| Author | researcher                    |

## Research Question

What entity model should sit upstream of Kanbanzai's execution layer to support the uncertainty-reduction phase — moving from "I have an idea" to "I know what I'm building"? This report synthesizes findings from the three prior shaping-layer research reports (exploration, architecture, hill-chart metaphor), competitive analysis of OmniFocus, OmniPlan, monday.com, and Basecamp's Hill Charts, and subsequent design conversation to propose a concrete entity model and data flow for the shaping layer.

## Scope and Methodology

**In scope:**
- Entity model for the shaping layer: Ideas, Plans, Milestones, and their relationships
- The boundary between shaping state and execution state (`.kbz/`)
- How the AI clerk operates across the entity model
- The human interaction surface: what actions the human takes, what the clerk handles
- Monorepo assumption for MVP; multi-repo deferred

**Out of scope:**
- Detailed MCP tool signatures or API design
- Role design (Admin Assistant vs. tools-only — deferred per prior research)
- Visual UI design
- Multi-repo architecture
- Time-boxing or appetite mechanisms

**Methodology:** Synthesis of prior research (planning-admin-layer exploration, architecture analysis, hill-chart metaphor report), competitive analysis (OmniFocus containers and sequential/parallel model, OmniPlan milestones and dependencies, monday.com portfolios and AI features, Basecamp Hill Charts), and structured design conversation.

## Findings

### Finding 1: Existing Tools Solve the Wrong Problem

Analysis of OmniFocus, OmniPlan, monday.com, and Basecamp reveals a consistent pattern: all are built for humans managing humans. They optimize for visibility, accountability, resource allocation, and status reporting — downhill concerns. None address the uphill phase where the real cognitive work happens: figuring out what to build in the first place.

What Kanbanzai needs is not another project management dashboard. It needs a structured path from intent to commitment — a shaping layer where an AI clerk handles the organizational load while the human makes decisions. The competitive analysis validates this gap: even monday.com's AI features (sidekick, agents, risk insights) are downstream tools that operate on already-structured work. They don't help you get from "I have an idea" to structured work.

Source: Competitive analysis — OmniFocus documentation, OmniPlan documentation, monday.com pricing/feature pages, Shape Up chapters 3, 11-13. Primary, 2019-2026.

### Finding 2: The Entity Model — Ideas, Plans, Milestones

The shaping layer introduces two new concepts above Kanbanzai's existing Plan/Batch/Feature/Task hierarchy:

**Idea** — A node in a generative tree. Ideas are the shaping layer's primary entity. They represent intent at any level of specificity: from vague ("better developer experience") to concrete ("stack trace formatting"). Ideas decompose into sub-ideas. Ideas are never "done" — they either graduate into plans, get abandoned, or persist as long-term direction that continues to generate new sub-ideas.

Properties:
- Title and description (free text)
- Parent idea (for tree structure)
- Hill position (derived from associated documents — see Finding 3)
- Associated documents (research, design drafts)
- Status: active, dormant, graduated, abandoned

**Plan** — The handoff artifact. A plan is created when an idea has been shaped enough to commit to building. It is the boundary between shaping and execution: once a plan is created in `.kbz/`, it enters Kanbanzai's existing lifecycle. The relationship between idea and plan is one-to-one: one idea produces exactly one plan. If an idea is too large for a single plan, it should be decomposed into sub-ideas first.

Properties:
- Derived from exactly one idea
- Contains one or more batches (Kanbanzai execution containers)
- Registered in `.kbz/` with full lifecycle state machine
- Carries the approved design and specification documents

**Milestone** — A zero-duration marker representing a significant checkpoint. A milestone is a picking list across plans: it selects plans from anywhere in the idea tree and groups them as a shared commitment. Milestones are cross-cutting — they don't own ideas, they reference plans. This matches how releases work: you pull from the backlog, you don't organize the backlog around releases.

Properties:
- Title and description
- Target date (optional — for the clerk's critical path tracking, not for time-boxing)
- Selected plans (references, not children)
- Critical path (computed by clerk from plan dependencies and hill positions)
- Status: proposed, active, completed, abandoned

**Why milestones are not batches:** A batch is an execution container — it groups features for coordinated implementation within a single worktree context. A milestone is a commitment marker — it groups plans (which contain batches) toward a meaningful checkpoint. A milestone might be satisfied by multiple batches across multiple plans. A single batch might contribute to multiple milestones. They operate at different levels.

**Why ideas are not plans:** An idea is pre-commitment. It lives in the shaping space, can be vague, can decompose, can be abandoned without consequence. A plan is post-commitment. It lives in `.kbz/`, has a lifecycle state machine, triggers worktree creation and task dispatch. The boundary is approval — the human says "yes, build this" and the idea becomes a plan.

Source: Design synthesis — prior research reports, competitive analysis, structured conversation. Primary, 2026.

### Finding 3: Hill Position Is Derived from Documents, Not Stored

Following the "build your way uphill" principle from Shape Up, hill position is not a subjective confidence rating. It is derived from concrete artifacts:

| Hill Position | Artifacts | Meaning |
|---|---|---|
| Bottom | Conversation only | Intent exists, nothing written |
| Lower uphill | Research report registered | Options explored, constraints documented |
| Mid-uphill | Design document (draft) | Approach selected, tradeoffs articulated |
| Upper uphill | Design document (approved) | Approach validated, open questions resolved |
| Top of hill | Specification (approved) | All requirements defined, ready for dev-plan |
| Downhill | Dev-plan → Tasks → Implementation | Kanbanzai's existing execution pipeline |

This makes hill position objective and auditable. The clerk derives position from document state rather than asking the human for a confidence score. An idea with an approved specification is at the top of the hill by definition. An idea with no documents is at the bottom. The clerk surfaces position; the human doesn't set it.

Source: Hill chart metaphor report (Finding 7, Recommendation 3). Primary, 2026.

### Finding 4: The AI Clerk's Responsibilities

The AI clerk operates across the full entity model. Its job is to structure, surface, and track — never to decide. Responsibilities by entity:

**Ideas:**
- Capture intent from conversation and reflect it back for confirmation
- Propose decomposition: "This idea is too large — should we break it into sub-ideas?"
- Surface related existing plans and decisions
- Track hill position based on document state
- Flag stuck ideas: "This has been at mid-uphill for two weeks"

**Plans (at handoff):**
- Produce the design and specification documents
- Guide the human through shaping conversations
- Register the plan in `.kbz/` when approved
- Decompose into batches and features

**Milestones:**
- Track critical paths across selected plans
- Surface risks: "Plan P4 is stuck uphill — this puts the milestone at risk"
- Propose plan selection: "These three plans are at the top of the hill and ready to commit"

**Execution (read-only):**
- Monitor plan/batch/feature status in `.kbz/`
- Surface execution status back to milestone views
- Flag when execution discoveries require revisiting shaping decisions

Source: Exploration report (Finding 4 — clerical roles), hill chart report (Finding 7 — AI partner contribution), design conversation. Primary, 2026.

### Finding 5: The Human Interaction Surface

The human only does what the AI clerk cannot: decide, approve, direct. The interaction surface is minimal:

| Action | What it does | Context |
|--------|-------------|---------|
| **Start something** | Begin a new shaping conversation. Creates an Idea. | "I have an idea about..." |
| **Say what you want** | The ongoing conversation. Free text. Clerk structures it. | Shaping discussions |
| **Pick a direction** | Choose between options the clerk surfaces. | Tradeoff decisions |
| **Approve** | The top-of-the-hill gate. Moves an Idea to a Plan in `.kbz/`. | Design/spec approval |
| **Reject / Rework** | Send work back downhill. | "This isn't right" |
| **Cut scope** | Remove parts. "Ship this subset." | Scope decisions |
| **Group** | Confirm or reshape clerk-proposed groupings. | Idea decomposition |
| **Split** | Confirm or adjust clerk-proposed splits. | Breaking down large ideas |
| **Sequence** | Establish priority and dependency. | "Do this first, then that" |
| **Check status** | A minimal viewer. What's uphill, what's at the top, what's stuck. | Orientation |

The clerk handles everything else: searching, structuring, drafting, decomposing, flagging, reminding, tracking.

Source: Design conversation — competitive analysis synthesis. Primary, 2026.

### Finding 6: Views, Not Entities

The same underlying data supports multiple views without requiring new entity types:

| View | Purpose | What it shows |
|------|---------|---------------|
| **Focus** | Deep shaping of one thing | One idea and its sub-ideas. Documents. Hill position. Full context for decisions. |
| **Horizon** | Portfolio awareness | All ideas at a glance. Hill positions only. Stuck items highlighted. Ready items surfaced. |
| **Chain** | Dependency consequences | What depends on what. "If I approve this, what's waiting on it?" Clerk-surfaced, not persistent. |

The milestone and comparison views emerge from these three — milestones are horizon view filtered to a specific commitment, comparison is focus view with sibling context. Three views, not five. The clerk uses all of them internally; the human primarily needs Focus and Horizon.

Source: Design conversation. Primary, 2026.

## The Data Flow

```
Shaping Space (clerk-managed, outside .kbz/)     Execution Layer (.kbz/)
                                                          
Idea: "Better DX"                                        
  ├── Idea: "Faster tests"                               
  │     └── Idea: "Parallel exec" ──→ Plan: P3 ──→ Batch B4 ──→ Features ──→ Tasks
  │                                                              
  ├── Idea: "Clearer errors"                             
  │     ├── Idea: "Stack traces" ──→ Plan: P4 ──→ Batch B5
  │     └── Idea: "MCP context"  ──→ Plan: P5 ──→ Batch B6
  │                                                      
  └── Idea: "Plugin system" (still shaping)              
                                                          
Milestone: "DX Sprint"                                   
  ├── Plan: P3                                           
  ├── Plan: P4                                           
  └── Plan: P5                                           
```

**Key boundaries:**
- Idea → Plan: The top of the hill. Human approval. Irreversible commitment. Plan enters `.kbz/`.
- Plan → Batch: Kanbanzai's existing decomposition. Clerk or orchestrator handles.
- Plan → Milestone: A selection, not a structural relationship. Plans can exist without milestones (direct execution path). Milestones are optional grouping for coordinated delivery.

**What lives where:**
- Ideas, milestones, and shaping conversations live outside `.kbz/`. They are the clerk's domain.
- Plans, batches, features, tasks live inside `.kbz/`. They are the execution system's domain.
- The clerk reads `.kbz/` for status but writes only to the shaping space (except at handoff, when it creates plans).

## MVP Scope

For the initial implementation, assuming a monorepo:

1. **Ideas** as the shaping entity — a tree of intent from vague to concrete
2. **One-to-one Idea → Plan handoff** — the approval gate
3. **Milestones** as optional picking lists across plans
4. **Hill position** derived from document state
5. **Clerk** operating across the full model with the responsibilities in Finding 4
6. **Human surface** limited to the nine actions in Finding 5
7. **Views** limited to Focus and Horizon (Chain surfaced by clerk in conversation)

**Deferred:** Multi-repo architecture, visual UI beyond a minimal viewer, time-boxing/appetite mechanisms, Admin Assistant role, `.plan/` sidecar vs. clerk-owned state (implementation detail for the shaping store).

## Relationship to Prior Research

This report is the synthesis that the three May 1st reports converged toward:

- **Exploration report** identified five missing cognitive functions. This report maps them to clerk responsibilities across the entity model: idea capture → Idea entity, scope negotiation → shaping conversation + document pipeline, dependency surfacing → Chain view + clerk tracking, prioritisation → milestone selection + sequencing, progress synthesis → Horizon view + hill position tracking.

- **Architecture report** proposed `.plan/` sidecar and data separation. This report adopts the separation principle (shaping state outside `.kbz/`) but defers the specific storage mechanism to implementation. The Idea/Plan boundary provides the clean separation the architecture report sought.

- **Hill chart report** proposed the hill metaphor as a conversational framework with two levels (portfolio and execution). This report collapses that to a single hill — the idea-to-plan journey — with milestones providing the portfolio-level grouping without a separate hill. Hill position is derived from documents, as the hill chart report recommended.

- **Competitive analysis** (OmniFocus, OmniPlan, monday.com) validated that existing tools solve the wrong problem but contributed useful concepts: OmniFocus's containers and sequential/parallel model inform the idea tree structure, OmniPlan's milestones as zero-duration markers inform the milestone concept, monday.com's AI features validate the clerk concept at market scale.

## Limitations

- **No implementation validation.** The entity model and data flow are design hypotheses. They have not been tested against real shaping workflows. The boundary between idea and plan may prove fuzzier in practice than in theory.

- **Monorepo assumption.** The multi-repo problem is real and deferred. The shaping layer's data model shouldn't need to change when multi-repo support is added, but the storage and clerk implementation will.

- **No user research.** The nine-action human surface is derived from first principles and competitive analysis, not from observation of humans attempting to shape work with an AI clerk. The actual interaction pattern may differ.

- **Clerk storage undefined.** This report deliberately defers the question of how and where the clerk stores shaping state (ideas, milestones, conversations). Options include a `.plan/` sidecar, a SQLite database, or purely in-memory with document-based persistence. The choice affects implementation but not the entity model.

- **Scope of "shaping" is still evolving.** The plan lifecycle includes a `shaping` stage, but what happens during shaping is not yet fully specified. This report proposes a concrete model, but it may need revision as the shaping stage's responsibilities are further defined through use.
