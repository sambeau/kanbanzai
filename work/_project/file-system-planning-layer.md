# File-System Planning Layer: Shaping State Through Structured Documents

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-02                     |
| Status | Draft                          |
| Author | sambeau                        |

## Related Work

### Prior documents consulted

| Document | Type | Relationship |
|----------|------|-------------|
| `work/research/planning-admin-layer-exploration.md` | research | Identified five missing cognitive functions (idea capture, scope negotiation, dependency surfacing, prioritisation, progress synthesis) and recommended a tools-first, clerk-modelled approach. This design adopts those functions and the clerk concept wholesale. |
| `work/research/planning-admin-layer-architecture.md` | research | Proposed a `.plan/` sidecar for planning state separate from `.kbz/`. This design replaces the sidecar concept with the idea folder tree itself as the canonical state — no separate directory needed. |
| `work/research/hill-chart-shaping-metaphor.md` | research | Proposed the hill metaphor as a conversational framework and recommended deriving hill position from document artifacts. This design adopts hill-as-derived, not hill-as-stored. |
| `work/research/shaping-layer-entity-model.md` | research | Proposed the Idea → Plan → Milestone entity model, the one-to-one Idea-to-Plan handoff, milestones as picking lists, and the nine-action human surface. This design refines those entities into file-system primitives rather than database records. |
| `work/design/planning-admin-layer-brainstorm.md` | design | Explored human burdens (reassembly, idea capture gap, dependency blindness) and design directions. This design addresses the "gap between capture and structure" identified as the point where ideas die. |
| `work/design/meta-planning-plans-and-batches.md` | design | Separated strategic plans from execution batches. This design sits upstream of both — ideas are pre-plan, pre-commitment. A plan is what an idea becomes when the human says "build this." |

### Decisions that constrain this design

1. **D1 from Plans and Batches** — Plans are the execution commitment boundary. Ideas are pre-plan. The Idea → Plan handoff is the approval gate where shaping state becomes execution state.
2. **P1-DEC-006** — `.kbz/state/` owns execution state; `work/` owns human-authored content. This design places idea state in a human-owned directory tree outside `.kbz/`, consistent with that separation.

### How this design extends prior work

The prior research converged on an entity model (Ideas, Plans, Milestones) and a role (the Clerk) but deliberately deferred the storage question. The architecture report proposed a `.plan/` sidecar with YAML files. This design takes a different approach: **the file system is the database.** Idea folders contain documents; folder nesting expresses decomposition; front matter carries metadata; milestones are sections in markdown files. There is no sidecar, no hidden state directory, and no sync problem — because the files the human edits are the canonical state.

This is a philosophical choice, not just a storage choice. It means the human's natural file operations (create folder, write document, move directory) *are* the planning operations. The clerk reads, indexes, derives, and surfaces — but the human owns the files.

---

## Overview

This design defines a **file-system-based planning layer** that sits upstream of Kanbanzai's execution pipeline. It introduces three concepts — Ideas, Milestones, and Roadmaps — expressed entirely through folders, markdown documents, and front matter, with no new entity types in `.kbz/` and no new storage mechanism.

An **Idea** is a folder containing a markdown document and optional sub-ideas. Ideas form a tree through folder nesting. Ideas are pre-commitment — they represent intent being shaped. When an idea is sufficiently shaped (documents written, design approved), it **graduates** into a Plan in `.kbz/`. The Idea-to-Plan handoff is the commitment boundary.

A **Milestone** is a named section within a roadmap document that references a list of plans. Milestones are cross-cutting — they group plans from anywhere in the idea tree toward a shared delivery checkpoint.

A **Roadmap** is a markdown document containing an ordered sequence of milestone sections. Any idea folder can contain a `roadmap.md`, allowing recursive, scoped roadmaps for large sub-systems.

An AI **Clerk** reads the file tree, derives state (hill position, completion, stuckness, critical path), resolves cross-references, maintains stable IDs, and surfaces insights to the human. The clerk is a reader and indexer, not an owner — the human controls the files.

## Goals and Non-Goals

**Goals:**

- Express the full planning lifecycle (capture → shape → commit) through files and folders the human controls
- Support recursive idea decomposition through folder nesting — an idea can contain sub-ideas to any depth
- Support recursive roadmaps — any idea folder can have a `roadmap.md` scoped to its sub-tree
- Support cross-tree linking — ideas can reference ideas in other branches via `relies_on`
- Derive hill position from document artifacts, not from stored metadata
- Keep the human's files as the single source of truth — no sync, no sidecar, no hidden database
- Enable the clerk to assign stable IDs on demand for cross-reference stability
- Define milestones as a convention within roadmap documents, not as a new entity type
- Keep the Idea → Plan graduation boundary clean and one-way

**Non-Goals:**

- Adding new entity types to `.kbz/` (Ideas and Milestones are file-system concepts, not `.kbz/` entities)
- Building a visual UI or drag-and-drop interface (deferred; the file system is the interface)
- Time-boxing or appetite mechanisms (there is no fixed-duration cycle)
- Multi-repo architecture (the design assumes a single repository)
- Multi-human collaboration mechanics (the design assumes a single human planning, with git as the async sharing mechanism)
- Replacing or modifying the existing Plan/Batch/Feature/Task entity model
- Defining the clerk's full MCP tool surface (this is a data-model and conceptual design, not a tool specification)

## Design

### 1. The idea tree

An idea is a folder. The folder contains a markdown document that describes the idea, plus optional supporting documents (research notes, design drafts, specifications) and optional sub-idea folders. Folder nesting expresses decomposition: a parent idea breaks down into child ideas.

```
ideas/
├── better-dx/                          ← Idea folder
│   ├── README.md                       ← Idea document (required)
│   ├── research/
│   │   └── test-bottlenecks.md         ← Supporting document
│   ├── roadmap.md                      ← Optional scoped roadmap (see §4)
│   │
│   ├── faster-tests/                   ← Sub-idea folder
│   │   ├── README.md
│   │   ├── design.md                   ← Design document (draft or approved)
│   │   └── parallel-exec/              ← Sub-sub-idea
│   │       ├── README.md
│   │       ├── design.md
│   │       └── spec.md                 ← Specification (approved)
│   │
│   ├── clearer-errors/                 ← Sub-idea
│   │   ├── README.md
│   │   └── design.md
│   │
│   └── plugin-system/                  ← Sub-idea with its own roadmap
│       ├── README.md
│       └── roadmap.md                  ← Scoped roadmap for this sub-tree
│
└── performance/                        ← Top-level idea
    ├── README.md
    └── faster-builds/
        └── README.md
```

#### The idea document

Every idea folder must contain exactly one markdown document that serves as the idea's canonical description. Front matter carries the idea's metadata:

```yaml
---
id: IDEA-better-dx                    # Optional stable ID (assigned by clerk)
title: Better Developer Experience    # Required
status: active                        # active | dormant | graduated | abandoned
relies_on:                            # Cross-tree dependencies (optional)
  - IDEA-faster-builds                #   Stable ID (preferred)
  - ../performance/faster-builds      #   Relative path (clerk resolves to ID)
graduated: P32-dx-improvements        # Set when idea becomes a plan (clerk-managed)
tags: [dx, performance, usability]    # Optional
---
```

**Status values:**

| Status | Meaning |
|--------|---------|
| `active` | Being shaped — documents are being written, conversations are ongoing |
| `dormant` | Paused — not abandoned, but not currently being shaped |
| `graduated` | Has become a plan in `.kbz/`. The idea folder persists as historical record. |
| `abandoned` | Will not proceed. The folder remains for reference. |

**What the clerk derives from the idea document and folder:**

- **Hill position** — from the documents present (see §5)
- **Completeness** — does the idea have sub-ideas? Are they graduated?
- **Staleness** — time since last file modification in the folder
- **Cross-reference health** — are `relies_on` targets resolvable?

#### Why folders, not a flat list

Folders express decomposition naturally. The human organizes ideas spatially — "these three things are part of the DX improvement." The alternative (a flat list with a `parent` field in front matter) requires the human to maintain parent-child relationships in metadata, which is fragile and unergonomic. Folders make the relationship visible, navigable, and immune to metadata drift — moving a folder moves the relationship.

### 2. Cross-tree linking with `relies_on`

Folder nesting expresses decomposition (parent → child). But ideas often depend on ideas in other branches — a backend idea might depend on a design-system idea; a plugin-system idea might depend on a performance idea. `relies_on` in front matter expresses these cross-tree dependencies.

```yaml
---
relies_on:
  - IDEA-faster-builds                   # Stable ID
  - ../performance/faster-builds         # Relative path (clerk promotes to ID)
---
```

**How the clerk manages `relies_on`:**

1. On indexing, the clerk encounters a relative path reference. It resolves the path to a folder, checks whether that folder's idea document has an `id`, and if not, assigns one. It then replaces the path with the stable ID in a proposed edit (or, if configured, does so automatically).
2. When the human moves or renames a folder, the clerk detects the broken reference on next index and surfaces: "`clearer-errors/README.md` references `../performance/faster-builds` which no longer resolves. Was it moved?"
3. The clerk can surface the dependency graph: "If you approve `faster-builds`, these three ideas are waiting on it."

IDs are assigned on demand — an idea only gets an `id` when something references it, or when the human explicitly requests one. This keeps the common case (isolated ideas with no cross-references) free of ID overhead.

### 3. Milestones

A milestone is a named section within a roadmap document that groups plans toward a shared delivery checkpoint. A milestone is not a new entity type — it is a convention for how roadmap documents are structured.

A milestone section contains:

- A heading (the milestone name)
- Optional descriptive text (narrative about the milestone's purpose)
- A list of plan references (checkbox items)

```markdown
## Milestone: Core Registry

The core registry is the foundation — plugins can't be loaded or discovered
without it. We're shipping this first so the public API milestone has
something to build on.

- [ ] P10-plugin-architecture
- [ ] P12-plugin-sandbox
```

**Checkbox semantics:**

| State | Meaning |
|-------|---------|
| `- [ ]` | Plan not yet complete |
| `- [x]` | Plan complete (human marks; clerk can propose) |

The clerk reads the referenced plans from `.kbz/` and derives milestone state:

- **Completion percentage** — plans completed / total plans
- **Status** — proposed (no plans started), active (some plans in progress), completed (all plans done), blocked (a plan is stuck uphill)
- **Critical path** — which incomplete plan is the bottleneck based on dependencies and hill position
- **Risk flags** — "P10 is still at mid-uphill with no approved design — this milestone has a Q2 target"

**Why milestones reference plans, not ideas:**
A milestone is a commitment to deliver. Ideas are pre-commitment — they may never graduate. Referencing plans means the milestone's scope is concrete. If the human wants to include a still-shaping idea in a milestone, they write the idea's path as a placeholder, and the clerk surfaces it as unresolved: "This milestone references an idea that hasn't graduated yet."

### 4. Roadmaps

A roadmap is a markdown document containing milestone sections in an order that represents the intended delivery sequence. The order of milestones in the document *is* the sequence.

```markdown
# Plugin System Roadmap

## Q2 2026

### Milestone: Core Registry

- [ ] P10-plugin-architecture
- [ ] P12-plugin-sandbox

### Milestone: Public API

- [ ] P11-plugin-registry
- [ ] P13-plugin-sdk

## Q3 2026

### Milestone: Marketplace

- [ ] P14-plugin-discovery
- [ ] P15-plugin-reviews
```

**Where roadmaps live:**

- A **project-level roadmap** at `roadmap.md` in the repository root (or `ideas/roadmap.md`) provides the high-level delivery sequence across all workstreams.
- A **scoped roadmap** at any idea folder (`ideas/plugin-system/roadmap.md`) provides the delivery sequence for that sub-tree. It can reference plans that graduated from its own sub-ideas and from elsewhere.

**Roadmap nesting:**

The project-level roadmap can reference a scoped roadmap as a single line item. The clerk reads the nested roadmap and surfaces its contents inline when reporting. The human doesn't need to duplicate plan lists across roadmaps.

**The difference between a nested roadmap and plan decomposition:**

A plan can contain sub-plans (the recursive plan model from the Plans and Batches design). A scoped roadmap inside an idea is different: it expresses *intended* delivery sequence before commitment. The plans referenced may not all exist yet — some may still be ideas. A plan with sub-plans is post-commitment: the work is approved and in `.kbz/`. A roadmap is pre-commitment: it's a shaping artifact that reduces uncertainty about delivery order.

When all the ideas in a scoped roadmap have graduated and their plans are active, the roadmap has served its purpose. It remains as a historical record of the shaping intent.

### 5. Hill position

Following the hill-chart metaphor, hill position measures how far an idea has progressed through uncertainty reduction. It is **derived from documented artifacts**, never stored.

| Hill Position | Artifacts Present | Meaning |
|---|---|---|
| Bottom | `README.md` only | Intent captured, nothing explored |
| Lower uphill | Research document(s) present | Options explored, constraints understood |
| Mid-uphill | Design document (draft) | Approach selected, tradeoffs articulated |
| Upper uphill | Design document (approved) | Approach validated, open questions resolved |
| Top of hill | Specification (approved) | All requirements defined, ready for handoff |
| Downhill | Graduated to plan | In `.kbz/` — the execution pipeline owns it now |

The clerk derives hill position by examining the documents in the idea folder:

- Does the folder contain a research document? → at least lower uphill
- Does it contain a design document? Is it approved? → mid to upper uphill
- Does it contain an approved specification? → top of hill
- Has the idea graduated (`graduated: P{n}` in front matter)? → downhill

A stuck idea is one that has been at the same hill position (no new documents, no document status changes) for an extended period. The clerk surfaces stuck ideas; the threshold is configurable but defaults to the time since the most recent file modification in the folder.

### 6. Idea → Plan graduation

Graduation is the commitment boundary. When the human approves an idea for execution:

1. The human (or clerk, at human direction) creates a plan in `.kbz/` via the existing `entity(action: create, type: plan, ...)` tool.
2. The clerk sets `status: graduated` and `graduated: P{n}` in the idea's front matter.
3. The idea folder remains in place with all its documents — it is now historical record of the shaping process.
4. The clerk updates any `relies_on` references pointing to this idea to point to the plan instead (or keeps both — the plan for execution tracking, the idea for provenance).

**Why the idea folder persists:**
The research, design drafts, and shaping conversations in the idea folder are valuable context. When someone later asks "why did we build this?", the answer is in the idea folder. When a new team member joins, the idea folder shows the thinking that led to the plan. Disk is cheap; lost context is expensive.

### 7. The clerk

The clerk is an AI role that reads the idea tree and `.kbz/` execution state, derives insights, and surfaces them to the human. The clerk does not own the files — the human does. The clerk's responsibilities:

**Indexing and resolution:**
- Walk the idea tree, parse every idea document's front matter
- Resolve `relies_on` path references to stable IDs, assign IDs on demand
- Index all roadmap documents, parse milestone sections, resolve plan references
- Detect broken references and surface them
- Cross-reference idea state with `.kbz/` plan/batch/feature status

**Derivation (never stored):**
- Hill position for every idea
- Stuckness detection (time since last activity at current hill position)
- Milestone completion percentage and critical path
- Portfolio-level synthesis: what's at the top of the hill and ready to graduate, what's stuck, what's dormant and might be abandoned

**Surfacing (human-directed):**
- Answer "where are we on X?" by synthesizing idea state, plan status, and roadmap position
- Answer "what should I work on next?" by surfacing top-of-hill ideas and at-risk milestones
- Flag: "This idea has been at mid-uphill for three weeks. Its design document hasn't been updated."
- Flag: "P10 in the Core Registry milestone is stuck at mid-uphill. This blocks the Public API milestone."
- Propose reordering: "Based on plan dependencies, Marketplace should come after Public API, not before."

**Writing (when asked or when safe):**
- Propose front matter changes via PR or direct edit (human-configurable)
- Assign stable IDs when needed for cross-references
- Update `graduated` and `status` fields at graduation
- Fix broken `relies_on` references when the target has clearly been renamed

**What the clerk does not do:**
- Decide what to build or in what order (the human decides)
- Delete or archive idea folders (the human controls the file system)
- Modify document content beyond front matter and reference fixups
- Create plans in `.kbz/` without human approval

### 8. The human surface

The human interacts with the planning layer through two surfaces:

**The file system (primary):**
- Create an idea folder, write a `README.md` → an idea exists
- Move a folder → the idea tree is restructured
- Add a `design.md` → the idea moves uphill
- Edit `roadmap.md` → milestones are reordered
- Check a box in a milestone → a plan is marked complete

**The clerk (supportive):**
- "What's ready to graduate?"
- "What's stuck?"
- "Show me the critical path for the Q2 milestones."
- "What depends on P10?"
- "Propose a milestone for these three plans."
- "Fix my broken references."

### 9. Views, not entities

The clerk surfaces the same underlying data through different views. No view requires a new entity type or stored structure:

| View | What it shows | Derived from |
|------|--------------|--------------|
| **Focus** | One idea: its documents, hill position, sub-ideas, dependencies | The idea folder and its contents |
| **Horizon** | All ideas at a glance: hill positions, stuck items, ready-to-graduate items | The full idea tree walk |
| **Roadmap** | Milestones in sequence with derived completion and risk | A specific `roadmap.md` file |
| **Chain** | What depends on what. "If I approve this, what's waiting?" | `relies_on` graph across the tree |

### 10. Directory layout

The idea tree lives in `ideas/` at the repository root, adjacent to but separate from `work/` (execution artifacts) and `.kbz/` (execution state):

```
my-project/
├── ideas/                         ← Shaping space
│   ├── roadmap.md                 ← Project-level roadmap (optional)
│   ├── better-dx/                 ← Idea folder
│   │   ├── README.md
│   │   ├── roadmap.md             ← Scoped roadmap (optional)
│   │   ├── faster-tests/          ← Sub-idea
│   │   │   └── README.md
│   │   └── clearer-errors/        ← Sub-idea
│   │       └── README.md
│   └── performance/
│       └── README.md
│
├── work/                          ← Execution artifacts (existing)
│   ├── design/
│   ├── spec/
│   └── plan/
│
├── .kbz/                          ← Execution state (existing)
│   └── state/
│       ├── plans/
│       ├── batches/
│       └── features/
│
└── roadmap.md                     ← Alternative: project roadmap at repo root
```

**Why `ideas/` rather than a `.plan/` sidecar:**
The architecture research proposed `.plan/` as a hidden directory. This design uses a visible, top-level `ideas/` directory. Reasoning:

- The human owns these files — they shouldn't be hidden
- `ideas/` is browseable in GitHub, IDE file trees, and file explorers
- The dot-prefix (`.plan/`) signals "tool-managed, don't touch" — but the whole point is that the human *should* touch these files
- `ideas/` matches the conceptual model: this is where ideas live before they become plans

## Alternatives Considered

### Alternative A: `.plan/` sidecar with YAML state files

**Proposed in:** `work/research/planning-admin-layer-architecture.md`

A `.plan/` directory containing `roadmap.yaml`, `proposals/`, `priorities.yaml`, etc. The clerk reads and writes these files. The human can edit them but the format is structured YAML.

**Rejected because:** It creates the same human-friction problem as `.kbz/` — the human must learn a schema and format to participate. The folder-and-markdown approach uses skills the human already has: creating folders, writing documents, checking boxes. It also eliminates the sync problem — if the clerk writes `.plan/roadmap.yaml` and the human edits `roadmap.md`, which is authoritative? In this design, the markdown file is the only copy.

### Alternative B: Database-backed planning layer

Planning state lives in a SQLite database (or `.kbz/` extended with idea and milestone entity types). The clerk is the primary writer. The human uses MCP tools or a visual UI.

**Rejected because:** It takes control away from the human. The file system is the most natural interface for organizing ideas — humans already create folders and write documents to think through problems. A database creates a walled garden where the human can only interact through tools. It also introduces sync complexity: if a document references an idea, and the idea is in a database, the document can't carry a simple file-path reference.

### Alternative C: Front matter as the only metadata store

No `id` field, no `relies_on` — all relationships expressed through folder nesting and document text. The clerk does natural-language parsing to surface dependencies.

**Rejected because:** Natural-language parsing is too unreliable for dependency tracking. The `relies_on` field is a small, structured concession that makes the clerk's job deterministic. The human writes it once; the clerk maintains it. The trade-off (a few lines of YAML vs. probabilistic parsing) strongly favours the structured approach.

### Alternative D: Milestones as YAML sidecar files

Each milestone is a separate `.yaml` file in a `milestones/` directory, rather than a section in a markdown document.

**Rejected because:** It fragments the roadmap narrative. A single `roadmap.md` document tells the story of the delivery sequence — the human can read it top to bottom and understand the plan. Splitting milestones into separate files destroys that narrative and creates the same "database disguised as files" problem that `.kbz/` has. The markdown-in-one-file approach also makes reordering trivial — cut and paste a section.

### Alternative E: `README.md` vs. `idea.md` for the idea document filename

**`README.md` pros:**
- Auto-renders in GitHub when browsing folders — the idea is immediately visible
- Familiar convention — READMEs are expected in project folders
- No new filename to invent or explain

**`README.md` cons:**
- In an editor or IDE, multiple tabs all titled "README.md" are disorienting — you can't tell which idea you're in
- Semantically, a README describes *the folder it's in*, not the *concept the folder represents*. An idea isn't "a folder with a README" — it's a concept the folder represents. `README.md` blurs that distinction
- GitHub auto-renders READMEs below the file listing, treating every idea folder as if it's a project root. For deeply nested idea trees, this can be visual noise

**`idea.md` pros:**
- Explicit — the filename *is* the semantic signal: "this file defines the idea"
- Distinctive in tab titles, search, and file trees — you always know you're looking at an idea document
- Doesn't trigger GitHub's README rendering (which may be preferable for deeply nested trees)

**`idea.md` cons:**
- Doesn't auto-render in GitHub folder views — you must click into the file to read the idea
- Unfamiliar convention requiring explanation

This design uses `README.md` in examples. The question should be resolved before implementation — and could be resolved simply by allowing both: the clerk recognizes whichever filename is present.

## Decisions

### D1: Ideas are folders, not database records

**Decision:** An idea is a folder containing a markdown document. The folder's location in the tree expresses its relationship to other ideas. The folder's contents (documents) determine its hill position. There is no idea entity in `.kbz/`.

**Rationale:** The human already knows how to create folders and write markdown. Making ideas into folders eliminates the capture gap — the human can jot an idea in a `README.md` without touching any tool. The clerk indexes it later. This also makes the tree structure visible and navigable in any file browser, IDE, or GitHub.

### D2: Front matter carries the minimum metadata needed for tool support

**Decision:** Front matter fields are limited to `id`, `title`, `status`, `relies_on`, `graduated`, and `tags`. All other state (hill position, staleness, completion, cross-reference health) is derived by the clerk.

**Rationale:** Every front matter field is a maintenance burden — the human must keep it accurate, or the clerk must manage it. Minimising front matter reduces both burdens. The fields that remain are those the clerk cannot reliably derive: what the idea is called (`title`), what state the human considers it to be in (`status`), and what it depends on (`relies_on`).

### D3: Hill position is derived from documents, not stored

**Decision:** The clerk determines hill position by examining which documents exist in the idea folder and their approval status. Hill position is never written to front matter.

**Rationale:** Stored hill position would be a subjective confidence rating that the human must remember to update. Derived hill position is objective and self-maintaining — when the human approves a design document, the idea automatically moves uphill. This is consistent with the hill-chart research's Recommendation 3.

### D4: Milestones are sections in roadmap documents

**Decision:** A milestone is a markdown section (heading) containing a description and a checkbox list of plan references. Milestones have no existence independent of the roadmap document that contains them.

**Rationale:** A milestone's meaning comes from its position in a sequence and its relationship to other milestones. Embedding milestones in a roadmap document preserves that context. It also makes milestone creation and reordering trivial — the human edits the document, reorders sections, adds or removes plan references. The clerk re-parses on next index.

### D5: Roadmaps are recursive

**Decision:** Any idea folder may contain a `roadmap.md` scoped to its sub-tree. A roadmap at a parent level may reference a child roadmap as a single item. The clerk reads nested roadmaps and resolves references.

**Rationale:** Large sub-systems within a project naturally have their own delivery sequence. Forcing all milestones into a single project-level roadmap creates an unwieldy document and obscures sub-system structure. Recursive roadmaps allow the human to organize delivery sequencing the same way they organize ideas — hierarchically.

### D6: Stable IDs are assigned on demand by the clerk

**Decision:** An idea only receives a stable `id` when something references it (via `relies_on` or a milestone) or when the human explicitly requests one. The clerk manages ID assignment and resolution.

**Rationale:** Most ideas will never be referenced from outside their immediate parent — they live and die within their sub-tree. Requiring IDs for every idea would add administrative overhead to the common case. On-demand assignment keeps the system lightweight while still supporting stable cross-references when needed.

## Open Questions

### OQ1: What is the idea document filename?

See Alternative E for the trade-off analysis between `README.md` and `idea.md`. The simplest resolution may be to support both: the clerk recognizes whichever of `README.md` or `idea.md` is present in the folder (with `idea.md` taking precedence if both exist). The human chooses based on their preference.

### OQ2: Where does the project-level roadmap live?

Candidates: `roadmap.md` at the repository root (most discoverable, renders on GitHub project page), `ideas/roadmap.md` (keeps all planning artifacts within `ideas/`), or both (root for project-level, nested for scoped roadmaps).

### OQ3: Should the clerk auto-fix broken references or only surface them?

When the human moves a folder and a `relies_on` path breaks, should the clerk:
- A) Surface the break and ask the human to fix it (safer, more human control)
- B) Auto-fix if the resolution is unambiguous — the target folder still exists at a new path and has the same `id` (more convenient, small risk of incorrect resolution)
- C) Auto-fix but create a commit the human can review and revert

### OQ4: What is the directory name for the idea tree?

`ideas/` is used throughout this design. Alternatives include `shaping/` (matches the shaping-layer vocabulary of the prior research), `directions/`, or `upstream/` (signals the relationship to `.kbz/`). `ideas/` is preferred because it's self-explanatory to a new contributor and does not require understanding the shaping metaphor.

### OQ5: Do graduated ideas stay in the tree or move to an archive?

This design proposes graduated ideas stay in place. The `status: graduated` field distinguishes them. Moving them to an `archive/` directory would require updating all references and destroy the tree structure that provides context. However, a large number of graduated ideas could clutter the active idea tree over time. A middle ground: the clerk can propose archiving an idea when all its sub-ideas have also graduated and the plan's work has been merged.

## Dependencies

- **The existing Plan entity in `.kbz/`** — the Idea → Plan handoff creates a plan via `entity(action: create, type: plan, ...)`. No changes to the plan entity are required.
- **The existing document intelligence system (`doc_intel`)** — the clerk will likely use doc-intel to parse document content and determine document types for hill position derivation.
- **File-system watching** — for the clerk to operate efficiently, it benefits from knowing when files change. This could be a file watcher (e.g., `fsnotify`) or a periodic re-index triggered by the human or by MCP tool calls.
- **The existing `id` allocation system** — stable idea IDs (`IDEA-{slug}`) may use the same allocator infrastructure as plans and batches, or a simpler scheme (since ideas are fewer and less performance-critical than `.kbz/` entities).
