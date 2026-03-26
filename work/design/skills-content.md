# Skills Content Design

- Status: draft design
- Purpose: define the content of each skill file installed by `kanbanzai init`
- Date: 2026-05-31
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

The six skills cover distinct concerns. Four are **procedural**: their content is well-defined
and is ready to be specified directly from this document. Two are **opinionated**
(`kanbanzai-planning` and `kanbanzai-design`): their content encodes collaborative norms that
require a design discussion before they can be written. Those sections are marked accordingly.

---

## 2. Skill File Format

Each skill is a standalone Markdown file at `.agents/skills/kanbanzai-{name}/SKILL.md`. Every
managed skill file begins with a machine-readable header comment:

```
<!-- kanbanzai-managed: {name} v{version} -->
```

This header allows `kanbanzai init --update-skills` to identify managed files, compare versions,
and safely overwrite outdated skills while refusing to overwrite user-created files.

All skill files follow the standard SKILL format:

```
# SKILL: {Name}

## Purpose
## When to Use
---
## Procedure / Rules / Content
---
## Common Issues     (where applicable)
---
## Verification      (where applicable)
---
## Related
```

Skills are human-readable documentation first. They should be concise, actionable, and free of
prose that belongs in design documents.

---

## 3. Procedural Skills

### 3.1 `kanbanzai-getting-started`

**Purpose:** Orient any agent at the start of a session in a Kanbanzai-managed project.

**When used:** This is the entry-point skill. An agent that has just opened a repository and
does not yet know what work to do — or what the current state of the project is — follows this
skill. It is the AI equivalent of "where do I start?".

**Content outline:**

1. **Before any work** — run `git status`; if there are uncommitted changes, commit or stash
   before starting new work. Never start new work on top of uncommitted changes from a different
   task.

2. **Understand the project** — check whether an `AGENTS.md` exists in the repository root.
   If it does, read it: it contains project-specific conventions, structure, and decisions.
   If it does not exist, the Kanbanzai skills are the primary orientation.

3. **Check the workflow state** — call the `work_queue` tool to see what tasks are ready.
   If the queue is empty, call `list_entities_filtered` to understand the current project
   state (active features, open bugs).

4. **Assemble context before starting a task** — before beginning work on any task or bug,
   call `context_assemble` with the appropriate role and task ID. Do not start implementation
   work without first reading the assembled context packet.

5. **Know your role in the workflow** — Kanbanzai separates human concerns (intent, priorities,
   approvals, product direction) from agent concerns (execution, decomposition, tracking).
   Agents must not make product decisions, skip workflow stage gates, or approve their own work.

6. **If you are unsure** — stop and ask. It is always better to ask than to make a wrong
   assumption about project structure, scope, or intent.

**Key constraints:**
- This skill must be short enough to read in full at the start of a session without feeling
  burdensome. Aim for under 60 lines of content.
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

| Stage | Who leads | What it produces | Gate |
|---|---|---|---|
| Planning | Human | Rough consensus to proceed | Human approval to move on |
| Design | Human + Agent | Approved design document in `work/design/` | Document must be approved before proceeding |
| Features | Agent | Plan + Feature entities | Design document must be approved |
| Specification | Human + Agent | Spec document in `work/spec/` | Features must exist |
| Dev plan & tasks | Agent | Task entities + dev plan | Spec must be approved |
| Implementation | Agent | Working code, tests, merged | Tasks must exist |

**Emergency brake** — a named set of conditions that require the agent to stop and ask the
human, including:
- About to write a document in `work/plan/` that contains "Decision:", "Architecture:", or
  "Technology Choice:" without an approved design document
- About to create Plan, Feature, or Task entities without an approved design
- About to make a technology or architecture choice without human approval
- Unsure which workflow stage the current work belongs to

**Lifecycle transitions** — a brief reference table for the legal state transitions for each
entity type (feature, task, bug). Agents must not perform illegal transitions.

**Human ownership** — a clear statement that humans own: intent, priorities, approvals, and
product direction. Agents own: execution, decomposition, implementation, and tracking.

**Key constraints:**
- The stage gate table is the heart of this skill. It must be present and accurate.
- The emergency brake section must be concrete — a list of specific triggers, not a vague
  instruction to "be careful".
- This skill deliberately overlaps with what `AGENTS.md` contains for this project; the skill
  is the portable version that is valid for any Kanbanzai project.

---

### 3.3 `kanbanzai-documents`

**Purpose:** Describe document types, where to place them, how to register them, and the
approval workflow.

**When used:** Whenever an agent creates, edits, or manages a document in the `work/` directory.

**Content outline:**

**Document types and locations:**

| Type | Directory | When to use |
|---|---|---|
| `design` | `work/design/` | Architecture, vision, approach decisions |
| `specification` | `work/spec/` | Acceptance criteria, binding contracts |
| `dev-plan` | `work/dev/` or `work/plan/` | Implementation plan, task breakdown |
| `research` | `work/research/` | Analysis, exploration, background |
| `report` | `work/reports/` | Review reports, audits, post-mortems |

**Registration procedure** — every document placed in a configured root must be registered
with `doc_record_submit` immediately after creation. Unregistered documents are invisible to
document intelligence, entity extraction, and health checks.

**Approval workflow:**
1. Agent or human creates the document.
2. Agent registers it with `doc_record_submit` — status is `draft`.
3. Human reviews the document and signals approval.
4. Agent calls `doc_record_approve` — status becomes `approved`.
5. Approved documents are the binding contracts for downstream work (spec → tasks,
   design → features).

**Drift and refresh** — if a document is edited after registration, its content hash becomes
stale. Use `doc_record_refresh` to update the hash before approving. Attempting to approve
a drifted document will fail.

**What agents must not do:**
- Do not create Plan or Feature entities that reference a document that is still in `draft`
  status.
- Do not edit an approved document without notifying the human; the approval becomes void when
  content drifts.
- Do not place design content in `work/plan/` — planning documents describe execution, not
  architecture.

**Key constraints:**
- The registration step must be clearly separated from document creation. The most common
  mistake is creating the file and forgetting to register it.
- The drift/refresh cycle must be explained because agents routinely edit documents and then
  attempt to approve them without refreshing.

---

### 3.4 `kanbanzai-agents`

**Purpose:** Define how agents interact with the Kanbanzai system: assembling context,
dispatching and completing tasks, writing commits, contributing knowledge, and spawning
sub-agents.

**When used:** Primarily by the orchestrating agent before starting implementation work and
after completing it. Also by sub-agents that need to understand the interaction protocol.

**Content outline:**

**Context assembly** — before beginning any task:
1. Call `context_assemble(role="{role}", task_id="{task_id}")` to get a context packet
   containing role instructions, relevant knowledge entries, and task details.
2. Read the full context packet before starting. Do not skip it.
3. After completing the task, call `context_report` to record which knowledge entries were
   used or found incorrect.

**Task dispatch and completion:**
1. Use `work_queue` to identify ready tasks.
2. Use `dispatch_task` to atomically claim a task (moves it from `ready` to `active`).
3. Do the work.
4. Use `complete_task` to close the task, providing a summary, files modified, verification
   performed, and any knowledge entries to contribute.

**Commit message format:**
```
<type>(<object-id>): <summary>
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `workflow`, `decision`, `chore`.
Add `!` after type for breaking changes.

Examples:
- `feat(FEAT-152): add profile editing API`
- `fix(BUG-027): prevent avatar uploads hanging on large files`
- `workflow(TASK-152.3): mark upload task complete after verification`

**Knowledge contribution** — when an agent learns something useful during a task that is not
already in the knowledge base, contribute it with `knowledge_contribute`. Set `tier: 3` for
session-level findings; the system promotes entries to tier 2 automatically based on usage.

**Sub-agent spawning** — when spawning a sub-agent:
1. Include the codebase knowledge graph project name in the delegation message.
2. Pass the relevant skill names so the sub-agent can read them.
3. `context_assemble` automatically includes relevant skill content in the context packet for
   sub-agents running through MCP.

**What agents must not do:**
- Do not commit directly to `main`.
- Do not use `_` to ignore errors.
- Do not skip `context_assemble` before implementation work.
- Do not complete tasks without providing a verification summary.

**Key constraints:**
- The commit message format table is a mandatory inclusion — it is the most frequently
  referenced piece of content in this skill.
- The sub-agent spawning section is brief here because the full protocol lives in
  `AGENTS.md`. The skill focuses on the Kanbanzai-specific obligations.

---

## 4. Opinionated Skills (Design Discussion Required)

The following two skills encode collaborative norms about *how agents and humans should behave
together* during planning and design. Unlike the procedural skills above, their content is not
derivable from existing specifications. They require a discussion about:

- What good planning looks like and what role the agent should play
- What good design collaboration looks like and when agents should hold back
- How to handle disagreement or ambiguity between human intent and agent suggestions
- Where the line is between "helping draft" and "making decisions"

These sections are placeholders. They will be filled in after the design discussion.

---

### 4.1 `kanbanzai-planning` *(Opinionated — Design Discussion Required)*

**Purpose:** Define how an agent should conduct itself during a planning conversation.

**Open questions for the design discussion:**

1. What is the agent's role during planning? Facilitator? Devil's advocate? Silent assistant?
2. When should an agent proactively suggest scope boundaries vs. wait to be asked?
3. How should an agent handle a planning conversation that is drifting toward design decisions?
   (i.e., the human is starting to make architecture choices during what should be a scoping
   conversation)
4. What does a "good output" from a planning session look like? Rough consensus? A written
   brief? Something the agent records?
5. How should the agent signal that it thinks a proposed scope is too large or too small?
6. What is the correct behaviour when the human asks the agent for an opinion on prioritisation?

**Provisional content areas** (subject to discussion):
- The distinction between planning (scope, priority, "what") and design (approach, "how")
- The questions an agent should ask to help a human clarify scope
- How to recognise when a planning conversation has concluded and is ready to move to design
- What the agent should record at the end of a planning session (if anything)

---

### 4.2 `kanbanzai-design` *(Opinionated — Design Discussion Required)*

**Purpose:** Define how an agent should collaborate on design documents: when to draft, when
to propose alternatives, when to surface tradeoffs, and when to stop and defer to the human.

**Open questions for the design discussion:**

1. What is the agent's role when asked to "draft a design"? Should it produce a complete
   document, or a structured outline for the human to fill in?
2. How should an agent present technical alternatives without inadvertently making the choice
   for the human?
3. When should an agent refuse a design request because the scope is under-defined?
4. What does "approved design" mean in practice? A verbal OK? A call to `doc_record_approve`?
   Something in between?
5. How should the agent handle a human who keeps adding scope to a design document that was
   already approved?
6. What is the correct behaviour when the agent believes a design decision is technically
   risky? Surface it once? Repeatedly? Block progress?
7. Should the agent ever suggest that a design document be split into smaller documents?
   Under what conditions?

**Provisional content areas** (subject to discussion):
- The difference between "drafting on behalf of the human" and "making design decisions"
- How to structure alternatives sections so the human is forced to make the choice
- When to request human review mid-document (not just at the end)
- The relationship between a design document and the decisions it encodes (and whether those
  should be explicitly extracted as decision records)

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
- The `<!-- kanbanzai-managed: {name} v{version} -->` header version is incremented.
- `kanbanzai init --update-skills` detects the version mismatch and overwrites the file.
- The old content is not preserved; skills are not user-editable.

If a project team wants to extend a skill with project-specific content, the correct mechanism
is a separate skill file (e.g., `.agents/skills/my-project-workflow/SKILL.md`) or additions
to `AGENTS.md`. The Kanbanzai-managed files must remain unmodified so they can be updated.