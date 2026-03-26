# Skills Content Design

- Status: draft design
- Purpose: define the content of each skill file installed by `kanbanzai init`
- Date: 2026-05-31
- Updated: 2026-06-01
- Related:
  - `work/design/kanbanzai-1.0.md` §4
  - `work/design/init-command.md` §2
  - `work/design/agent-interaction-protocol.md`
  - `work/design/document-centric-interface.md`
  - Feature: FEAT-01KMKRQSD1TKK

---

## 1. Purpose

This document defines the content of the six skill files that `kanbanzai init` installs into
`.agents/skills/kanbanzai-*/SKILL.md`. Skills are the primary way Kanbanzai teaches AI agents
how to work in a Kanbanzai-managed project. They are discovered automatically by AI-enabled
editors and delivered to sub-agents through `context_assemble`.

The six skills cover distinct concerns. Four are **procedural** — their content is derivable
directly from existing specifications and workflow rules. Two are **opinionated**
(`kanbanzai-planning` and `kanbanzai-design`) — their content encodes collaborative norms
developed through design discussion. All six are now defined and implemented.

---

## 2. Skill File Format

Each skill is a directory at `.agents/skills/kanbanzai-{name}/` containing a `SKILL.md` file,
following the [agentskills.io specification](https://agentskills.io/specification). This
structure allows AI-enabled editors (VS Code, Claude Code, and others) to discover and activate
skills automatically via progressive disclosure.

### Frontmatter

Every skill file begins with YAML frontmatter:

```yaml
---
name: kanbanzai-{name}
description: >
  Imperative description of when to activate the skill. Should be keyword-rich
  and "pushy" — explicitly list contexts where the skill applies, including
  cases where the agent might not think to trigger it unprompted.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---
```

The `name` field must match the parent directory name. The `description` is the primary
mechanism agents use to decide whether to activate the skill — it is loaded at startup
for all skills and must clearly convey when the skill is useful.

The `metadata` block carries the managed marker and version. These allow `kanbanzai init
--update-skills` to identify managed files, compare versions, and safely overwrite outdated
skills while refusing to overwrite files without the marker (which are user-created).

### Body Structure

The Markdown body follows this general structure, adapted to each skill's needs:

```
## Purpose
## When to Use
---
## [Content sections — procedural steps, rules, principles]
---
## Gotchas
## Related
```

**Gotchas sections** are high-value content that describe specific failure modes and their
consequences. "If you forget X, Y will happen" is more actionable than "do not forget X."
The consequence is what makes the instruction stick.

### Progressive Disclosure

Skills follow the agentskills.io progressive disclosure model:

- **Frontmatter** (name + description) — loaded at startup for all skills; ~100 tokens
- **SKILL.md body** — loaded when the skill activates; target under 200 lines
- **`references/` files** — loaded on demand for detailed reference material

Reference files are placed in a `references/` subdirectory within the skill directory. The
skill body instructs the agent when to load them, so detail only enters the context window
when it is needed.

Currently:

- `kanbanzai-workflow/references/lifecycle.md` — entity lifecycle transition diagrams for all
  entity types (feature, task, bug, plan)
- `kanbanzai-design/references/design-quality.md` — full definitions of the six design quality
  principles

---

## 3. Procedural Skills

### 3.1 `kanbanzai-getting-started`

**Purpose:** Orient any agent at the start of a session in a Kanbanzai-managed project.

**When used:** This is the entry-point skill. An agent that has just opened a repository and
does not yet know what work to do — or what the current state of the project is — follows this
skill. It is the AI equivalent of "where do I start?".

**Content outline:**

1. **Before any work** — run `git status`; if there are uncommitted changes, commit or stash
   before starting new work. Never start new work on top of uncommitted changes from a
   different task.

2. **Understand the project** — check whether an `AGENTS.md` exists in the repository root.
   If it does, read it: it contains project-specific conventions, structure, and decisions.
   If it does not exist, the Kanbanzai skills are the primary orientation.

3. **Check the workflow state** — call the `work_queue` tool to see what tasks are ready.
   If the queue is empty, call `list_entities_filtered` to understand the current project
   state (active features, open bugs).

4. **Assemble context before starting a task** — before beginning work on any task, call
   `context_assemble` with the appropriate role and task ID. See `kanbanzai-agents` for the
   full dispatch-and-complete protocol.

5. **Understand the workflow** — Kanbanzai has stage gates that require human approval at
   specific points. See `kanbanzai-workflow` for the rules, the human/agent ownership
   boundary, and when to stop and ask.

**Key constraints:**
- This skill must be short enough to read in full at the start of a session without feeling
  burdensome. Target under 70 lines total.
- It links to the other skills but does not duplicate their content.
- It must not contain project-specific information (that belongs in `AGENTS.md`).

---

### 3.2 `kanbanzai-workflow`

**Purpose:** Describe the workflow stage gates, entity lifecycle, and the rules for when to
stop and ask the human.

**When used:** Any time an agent is making progress decisions: deciding what to build, whether
to proceed to the next stage, whether a state transition is valid, or whether to create new
entities.

**Content outline:**

**Stage gates** — the six stages of the workflow and what each requires:

| Stage | Who leads | What it produces | Gate to pass |
|---|---|---|---|
| Planning | Human | Agreed scope | Human signals readiness to design |
| Design | Human + Agent | Approved design document | Document approved before proceeding |
| Features | Agent | Plan + Feature entities | Design document must be approved |
| Specification | Human + Agent | Approved spec document | Features must exist |
| Dev plan & tasks | Agent | Task entities + dev plan | Spec must be approved |
| Implementation | Agent | Working code, tests, merged | Tasks must exist |

The full progression applies to features and plans. Bug fixes and small improvements follow
a lighter path — no design document or specification needed unless the fix involves a
significant architectural change.

**Human ownership** — the canonical statement of what humans own vs. what agents own. This
skill is the authoritative home for this principle; other skills reference it rather than
restating it.

**Emergency brake** — a named set of conditions that require the agent to stop and ask the
human. Concrete triggers, not vague advice:
- About to write design content without an approved design document
- About to create Plan, Feature, or Task entities without an approved design
- About to make a technology or architecture choice without explicit human approval
- Unsure which workflow stage the current work belongs to
- Work has drifted beyond the scope of the task, feature, or plan

**Entity lifecycle transitions** — moved to `references/lifecycle.md` (progressive disclosure).
The skill body points to the reference file rather than embedding the transition diagrams.

**Gotchas** — three specific failure modes:
- Tool call failures: read the error message, it names the valid transitions; do not retry
  with the same arguments
- Stage gates apply to the entity type, not the size of the work
- Verbal approval must be recorded with `doc_record_approve` immediately

**Key constraints:**
- The stage gate table is the heart of this skill. It must be present and accurate.
- The emergency brake section must be concrete triggers, not a vague instruction to be careful.

---

### 3.3 `kanbanzai-documents`

**Purpose:** Describe document types, where to place them, how to register them, and the
approval workflow.

**When used:** Whenever an agent creates, edits, or manages a document in the `work/`
directory. The description is written to trigger even when the agent doesn't explicitly think
about registration — any file creation in a configured root should activate this skill.

**Content outline:**

**Document types and locations:**

| Type | Typical directory | When to use |
|---|---|---|
| `design` | `work/design/` | Architecture, vision, approach decisions |
| `specification` | `work/spec/` | Acceptance criteria, binding contracts |
| `dev-plan` | `work/dev/` or `work/plan/` | Implementation plans, task breakdowns |
| `research` | `work/research/` | Analysis, exploration, background |
| `report` | `work/reports/` | Review reports, audits, post-mortems |
| `policy` | `work/design/` | Standing rules, process definitions |

**Registration procedure** — every document placed in a configured root must be registered
with `doc_record_submit` immediately after creation. Unregistered documents are invisible to
document intelligence, entity extraction, approval workflow, and health checks.

**Approval workflow** — three-status lifecycle: draft → approved → superseded. Approved
documents are the binding basis for downstream work (approved design → features can be
created; approved spec → tasks can be decomposed).

**Drift and refresh** — if a document is edited after registration, its content hash becomes
stale. Use `doc_record_refresh` before approving. Attempting to approve a drifted document
will fail.

**Supersession** — when a document is replaced, create the new document, register it, then
call `doc_record_supersede` on the old one.

**Gotchas** — four specific failure modes with consequences:
- Forgot to register: the document is invisible to the system; `batch_import_documents` as
  safety net
- Editing after approval: approval is silently void; must notify human and re-approve
- Design in wrong place: design decisions belong in `work/design/`, not `work/plan/`
- Tool call failures: read the error message, fix the underlying issue, do not retry blindly

**Key constraints:**
- The registration step must be clearly separated from document creation.
- The drift/refresh cycle must be explained with consequences — it is the most common cause
  of approval failures.

---

### 3.4 `kanbanzai-agents`

**Purpose:** Define how agents interact with the Kanbanzai system: assembling context,
dispatching and completing tasks, writing commits, contributing knowledge, and spawning
sub-agents.

**When used:** Primarily by the orchestrating agent before starting implementation work and
after completing it. Also by sub-agents that need to understand the interaction protocol.
The description is written to trigger even for tasks that seem simple.

**Content outline:**

**Context assembly** — the canonical home for `context_assemble` usage:
1. Call `context_assemble(role="<role>", task_id="<task_id>")` to get a context packet.
2. The packet is byte-budgeted and prioritised — it contains what matters most for this
   task and role.
3. After completing the task, call `context_report` to record which knowledge entries were
   used or found incorrect (drives auto-confirmation and auto-retirement of entries).

**Task dispatch and completion:**
1. Use `work_queue` to identify ready tasks.
2. Use `dispatch_task` to atomically claim a task (moves it from `ready` to `active`).
3. Do the work.
4. Use `complete_task` with summary, files modified, verification performed, and any
   knowledge entries to contribute.

**Commit message format** — the canonical home for the commit format. Includes the type
table (`feat`, `fix`, `docs`, `test`, `refactor`, `workflow`, `decision`, `chore`) and
examples. Other skills reference this skill rather than restating the format.

**Knowledge contribution** — `knowledge_contribute` with tier 3 for session-level findings;
the system auto-promotes entries to tier 2 based on usage.

**Sub-agent spawning** — include context (project name, file scope boundaries, conventions)
in every delegation. `context_assemble` is the primary skill delivery mechanism for
MCP-connected sub-agents.

**Gotchas** — three specific failure modes:
- `dispatch_task` fails: another agent claimed the task; call `work_queue` again
- Tool call failures: read the error message before retrying
- Missing verification summary: `complete_task` will succeed but degrades review quality

**Key constraints:**
- The commit message format table is a mandatory inclusion.
- Generic prohibitions (product decisions, approving own work) are not restated here — they
  belong in `kanbanzai-workflow`. This skill focuses on the interaction protocol.

---

## 4. Opinionated Skills

The following two skills encode collaborative norms about how agents and humans work together
during planning and design. Their content was developed through design discussion; the
decisions are recorded below and reflected in the implemented skill files.

---

### 4.1 `kanbanzai-planning`

**Purpose:** Guide a planning conversation to produce clear scope — what to build, how big it
is, and how it fits into the project structure — without making design or architecture
decisions.

**Key decisions:**

**Agent's role:** The agent is an active participant, not a silent facilitator. It can suggest
options, flag opportunities, and recommend directions. The human makes the final scoping
decisions. (An earlier formulation said "the agent does not recommend what to build" — this
was too restrictive and has been revised.)

**Ambition principle:** An AI agent team is not constrained by team size. Sub-agents can be
spawned for any domain in any number. The limit on what gets built is the quality of the
design, not the capacity of the team. The agent presents the ambitious version first; scope
reduction requires explicit reasons from the human.

**Anti-patterns to surface:** premature simplification ("let's just do the simple version"),
scope reduction as comfort ("that's too ambitious"), and deferred design ("we can figure that
out later"). When any of these appear in a planning conversation, the agent names them
explicitly.

**Feature vs. plan:** A feature is a single coherent piece of behaviour that can be designed,
specified, and implemented independently. A plan coordinates multiple features toward a shared
milestone. Err towards fewer plans — a plan with one feature is usually just a feature.

**Sizing signals:**
- One feature: describable in one sentence, one design document, one implementation sprint
- Multiple features (needs a plan): independently designable parts, parallel-workable, would
  produce multiple design documents
- Too large to plan yet: not yet clear what the individual features are; write a high-level
  design document first

**Drift into design:** When the conversation moves into *how* something will work (data models,
API shapes, technology choices), the agent redirects: *"That sounds like a design question —
should we capture it in the design document?"*

**Planning output:** A scope statement, a structural decision (one feature or a plan with N
named features), and the human's signal to proceed. No formal planning document required
unless the scope is complex enough to be hard to track.

---

### 4.2 `kanbanzai-design`

**Purpose:** Guide the design process from an agreed scope to an approved design document,
ready for specification.

**Key decisions:**

**Roles:** The human is the Design Manager — they own decisions, make the final call, and
approve. The agent is the Senior Designer — proposes, drafts, researches, and recommends, but
cannot make final design decisions or approve its own work. This mirrors a standard design
team: the senior designer drives the work, the manager owns the outcomes.

**Drafting:** When asked to draft a design, produce a complete document, not an outline. A
draft with alternatives and open questions is more useful than a skeleton. Do not start a
draft until at least the scope is decided.

**Alternatives:** Present multiple approaches with descriptions, trade-offs, and an explicit
recommendation from the agent. The recommendation is advice; the decision belongs to the
human. Draft documents may contain alternatives; approved documents must not — they reflect
one chosen direction.

**Open questions:** Any unresolved design question must be listed explicitly in the document.
A design cannot be approved until all design questions are resolved. Implementation questions
(how it will be built) may remain open; design questions (what it is) may not.

**Approved design invariant:** A design document is ready for approval when it contains:
(1) scope — what is being built and why; (2) one chosen direction; (3) key decisions and
rationale, including "why not X"; (4) no unresolved design questions. Approval can be verbal;
record it with `doc_record_approve` immediately.

**Risk surfacing:** Minor concerns → mention once. Significant risks → repeat until
acknowledged. Security or data-integrity risks → do not proceed without explicit
acknowledgment.

**When to split:** If a design logically breaks into independently designable and
implementable parts, step back to planning. A plan with multiple features and a high-level
umbrella document is the right structure. Signs a design should split: different sections feel
like separate products, different parts could be implemented without blocking each other, the
specification would be unmanageably large.

**Scope growth after approval:** Create a new design document and supersede the old one. Do
not silently amend an approved document.

**Design quality:** Six qualities serve as a lens: simplicity, minimalism, completeness,
composability, honesty, and durability. The relationship between the core four matters:
simplicity without completeness is a prototype; completeness without minimalism is bloat;
minimalism without composability is fragile. Full definitions and context are in
`references/design-quality.md`.

---

## 5. Relationship Between Skills and `AGENTS.md`

The Kanbanzai skills are **portable**. They describe how to use Kanbanzai in any project. They
must not contain project-specific information.

A project's `AGENTS.md` is **project-specific**. It describes the project's own conventions,
structure, decisions, and reading order. It may reference the Kanbanzai skills, but must not
duplicate their content.

The relationship is:

```
AGENTS.md
  ├─ Project conventions, structure, reading order
  ├─ Project-specific decisions and constraints
  └─ "Follow the kanbanzai-workflow skill for stage gate rules"

.agents/skills/kanbanzai-*/SKILL.md
  ├─ How to use Kanbanzai in any project
  ├─ Tool usage patterns, commit format, lifecycle rules
  └─ Delivered to sub-agents via context_assemble
```

The existence of skills does not make `AGENTS.md` optional for existing projects. For new
projects, `kanbanzai init` creates a starter `AGENTS.md` that references the installed skills
and invites the human to add project-specific content.

---

## 6. Content Maintenance

Skill files are versioned with the Kanbanzai binary. When a skill's content changes:

- The `metadata.version` field in the frontmatter is incremented (semver: `0.1.0`, `0.2.0`,
  etc.).
- `kanbanzai init --update-skills` reads the `metadata.kanbanzai-managed` marker to identify
  managed files, compares `metadata.version` to the current binary's version for that skill,
  and overwrites if the installed version is older.
- Files without the `kanbanzai-managed` metadata marker are not touched — they are assumed to
  be user-created files.

If a project team wants to extend a skill with project-specific content, the correct mechanism
is a separate skill file (e.g., `.agents/skills/my-project-workflow/SKILL.md`) or additions
to `AGENTS.md`. The Kanbanzai-managed files must remain unmodified so they can be updated
cleanly.