---
name: kanbanzai-getting-started
description: >
  Use at the start of every agent session, even if the task seems obvious and
  you think you already know what to do. Activates when the agent has just
  opened a repository, does not know what to do, needs to orient itself, or is
  beginning any new session. Also activates for "where do I start?", "what
  should I work on?", "what is the current state?". Skipping orientation leads
  to wasted effort and missed context.
metadata:
  kanbanzai-managed: "true"
  version: "0.4.0"
---

# SKILL: Kanbanzai Getting Started

## Purpose

Orient an agent at the start of a session in a Kanbanzai-managed project.

## When to Use

- At the beginning of any new agent session
- When you don't know what work to do or where to start
- When resuming work after a break

## Vocabulary

- **Session orientation** — The mandatory startup sequence that establishes what state the project is in, what work is available, and what conventions apply before any implementation begins.
- **Work queue** — The prioritised list of ready tasks returned by `next()`. The single source of truth for what an agent should work on.
- **Context packet** — The assembled bundle of spec sections, knowledge entries, file paths, and role conventions returned when claiming a task with `next(id)`.
- **Clean slate** — The state where `git status` shows no uncommitted changes from previous work. Required before starting any new task.
- **Task claim** — The act of calling `next(id)` to transition a task from `ready` to `active` and receive its context packet.
- **Feature lifecycle state** — The current workflow stage of a feature entity (e.g. `designing`, `specifying`, `implementing`). Determines which skills, roles, and gates apply.
- **AGENTS.md** — The project-specific conventions file in the repository root. Contains structure, build commands, Git discipline, and the pre-task checklist.
- **Stage binding** — The mapping in `.kbz/stage-bindings.yaml` that connects each workflow stage to its required role, skill, and prerequisites.
- **Project status** — The synthesised dashboard returned by `status()` showing progress, blocked items, and attention items across the project.

---

## Session Start Checklist

Copy this checklist at the beginning of every session:

- [ ] **Clean slate** — Run `git status`. If changes are coherent and complete, commit them now. If changes are incomplete or belong to a different task, inform the human and stop — do not stash. Never use `git stash` in a Kanbanzai project: stashed changes hide workflow state from other agents and are silently lost across worktree switches.
- [ ] **Store check** — If `git status` shows uncommitted `.kbz/` files, commit them now. These are versioned project state, not ephemeral cache. Do not stash, discard, or `.gitignore` them.
- [ ] **Corpus integrity check** — Call `doc(action: "audit")` and review its output before proceeding. If audit shows files on disk not registered → call `doc(action: "import", path: "work")` to register all unregistered documents. If audit shows registered records whose files are missing from disk → call `doc(action: "delete", id: "DOC-xxx")` for each stale record. After any batch registration, run a classification pass on newly registered documents (see the Classification (Layer 3) section in `kanbanzai-documents`) before proceeding. *Rationale: an incomplete corpus produces false negatives in design searches, and a designer who finds no results cannot distinguish "not addressed" from "not registered".*
- [ ] **Read project context** — Read `AGENTS.md` if you have not this session.
- [ ] **Check the work queue** — Call `next()` to see what is ready.
- [ ] **Claim your task** — Call `next(id: "TASK-xxx")` to get full context for your chosen task.
- [ ] **Understand the workflow** — If unsure about the current stage, check the `kanbanzai-workflow` skill.

### Clean slate

Run `git status`. If there are uncommitted changes from previous work:

- Changes are coherent and complete → **commit them now**, then proceed.
- Changes belong to a different task or are incomplete → **inform the human and stop**. Do not stash, do not discard.

Never use `git stash` in a Kanbanzai project. Stashed changes hide workflow
state from parallel agents, are silently lost when switching worktrees, and
bypass the commit history that makes code review meaningful.

### Commit workflow state

Even when the working tree looks clean for code, run:

```
git status
```

and look specifically for untracked or modified files under `.kbz/state/`,
`.kbz/index/`, or `.kbz/context/`. These are versioned project state — entity
records, document metadata, knowledge entries — that other agents depend on.

If any appear:

1. Stage and commit them immediately before starting any new work:
   ```
   git add .kbz/
   git commit -m "workflow(<context>): commit orphaned state files"
   ```
2. Do not stash, discard, or `.gitignore` them.

MCP tools (`entity`, `doc`, `finish`, `decompose`, `merge`) auto-commit state
changes during normal operation. Orphaned files appear when a previous session
was interrupted before the auto-commit could run. They are rare but consequential:
stale state causes parallel agents to read incorrect entity status and produce
conflicting transitions.

### Corpus integrity check

Call `doc(action: "audit")` at the start of each session and review its output:

- **Unregistered files** — Files on disk in configured roots that have no document record.
  If found: call `doc(action: "import", path: "work")` to register all unregistered
  documents in a single batch operation.
- **Stale records** — Registered document records whose files are missing from disk.
  If found: call `doc(action: "delete", id: "DOC-xxx")` for each stale record to remove
  the orphaned reference.

After any batch registration triggered by the integrity check, run a classification pass
on the newly registered documents before proceeding. Follow the Classification (Layer 3)
section in `kanbanzai-documents` for the step-by-step procedure.

**Rationale:** An incomplete corpus produces false negatives in design searches. A designer
who searches for an architectural decision and finds no results cannot distinguish "this has
not been addressed" from "this was addressed in an unregistered document." Classification
ensures concept search returns accurate results.

### Read the project context

Check whether an `AGENTS.md` exists in the repository root. If it does, read
it — it contains project-specific conventions, structure, and decisions. If it
does not, these Kanbanzai skills are your primary orientation.

### Check the work queue

Call `next` (without an ID) to see what tasks are ready. If the queue is
empty, call `status` or `entity` action: `list` to understand the current
project state — active features, open bugs, what stage things are in.

### Claim your task

Call `next` with a task ID to claim it and get your instructions and
context. See `kanbanzai-agents` for the full dispatch-and-complete
protocol.

### Understand the workflow

Kanbanzai has stage gates that require human approval at specific points.
See `kanbanzai-workflow` for the rules, the human/agent ownership boundary,
and when to stop and ask.

Each workflow stage (designing, specifying, developing, reviewing, etc.) maps
to a specific **role** and **skill** via `.kbz/stage-bindings.yaml`. Read the
binding for your current stage to know which role to adopt and which skill
procedure to follow. The task-execution skills themselves live in
`.kbz/skills/` (e.g. `write-design`, `write-spec`, `review-code`,
`orchestrate-review`).

---

## New-Project Onboarding

Applies when Kanbanzai is initialised for the first time in a repository with no
prior `.kbz/` directory.

### Steps

1. **Configure document roots.** Add your document directories to `.kbz/config.yaml`
   under `documents.roots`. Typically these include `work/design/`, `work/spec/`,
   `work/dev/`, `work/research/`, and `work/reports/`.

2. **Import all documents.** For each configured root, run:
   ```
   doc(action: "import", path: "<each-root>")
   ```
   This registers all existing documents in that root. The import is idempotent —
   running it multiple times is safe.

3. **Verify with audit.** Call `doc(action: "audit")` and confirm zero unregistered
   files remain. Resolve any residual gaps before proceeding.

4. **Run batch classification.** Classify all registered documents in priority order:
   - Specifications first (most structured, highest value for concept search)
   - Designs second (narrative + architectural decisions)
   - Dev-plans third (task-oriented)
   - Research and reports last (lowest priority)

   Follow the Classification (Layer 3) section in `kanbanzai-documents` for the
   step-by-step procedure.

5. **Validate the concept registry.** Call:
   ```
   doc_intel(action: "find", role: "decision")
   ```
   If results are returned, the concept registry is populated and classification
   was successful. If no results are returned, re-check that documents were classified
   (not just registered).

**Time estimate:** Approximately 5–10 minutes per document for classification. For a
50-document corpus, expect approximately 4–8 hours of elapsed time.

---

## Existing-Project Adoption

Applies when Kanbanzai is added to a project that already has documents outside
standard `work/` directories.

### Steps

1. **Audit the repository for markdown files.** Identify files not covered by your
   configured roots:
   ```
   find . -name "*.md" | grep -v ".kbz"
   ```
   Review the output for design decisions, specifications, architectural rationale,
   and requirements documents.

2. **Decide which documents belong in the corpus.** Not all markdown files are
   workflow documents. Include: design decisions, specifications, architectural
   rationale, requirements. Exclude: README files, changelogs, generated documentation,
   and transient notes.

3. **Add additional roots as needed.** If documents are found outside standard
   directories, add those directories to `.kbz/config.yaml` under `documents.roots`.

4. **Register and classify.** Follow the New-Project Onboarding procedure (above)
   to import, verify, and classify all documents.

**Key principle:** The corpus must be complete enough that a negative search result
means "this has not been addressed" rather than "this might have been addressed in
an unregistered document."

---

## Anti-Patterns

### Skipping Orientation

- **Detect:** Agent starts implementing without calling `next()` or reading AGENTS.md.
- **BECAUSE:** Without orientation, the agent misses project conventions, active decisions, and existing work — leading to duplicated effort or conflicting changes.
- **Resolve:** Always run the session start checklist before writing any code.

### Stale Context Carry-Over

- **Detect:** Agent assumes context from a previous session is still current (e.g. task status, branch state, file contents).
- **BECAUSE:** Entity states, knowledge entries, and file contents change between sessions. Stale assumptions produce incorrect implementation decisions that compound as work progresses.
- **Resolve:** Always check `git status` and call `next()` at session start. Treat every session as a fresh start.

### Store Neglect

- **Detect:** Uncommitted `.kbz/state/` files visible in `git status` at session start.
- **BECAUSE:** Store drift causes race conditions when parallel agents read stale entity state, leading to conflicting transitions and lost updates.
- **Resolve:** Commit `.kbz/` files immediately. Do not stash, discard, or gitignore them.

### Shell-Querying Workflow State Files

- **Detect:** Agent runs `cat`, `grep`, `find`, or similar shell commands against `.kbz/state/`, `.kbz/index/`, or `.kbz/context/` directories to retrieve entity data.
- **BECAUSE:** Raw YAML files contain unresolved state. MCP tools apply lifecycle resolution, inheritance, computed fields, and cross-reference validation. Shell queries bypass all of this and produce subtly wrong results — wrong status, missing computed fields, stale index data — that lead to incorrect implementation decisions.
- **Resolve:** Use MCP tools exclusively for all workflow state queries:
  - Entity status → `entity(action: "get", id: "...")`
  - Project overview → `status()`
  - Knowledge entries → `knowledge(action: "list")`
  - Documents → `doc(action: "get", path: "...")`
  Never read `.kbz/state/` files with shell tools or `read_file`.

---

## Evaluation Criteria

1. **Did the agent run `git status` and address uncommitted changes before starting work?** — Required
2. **Did the agent call `next()` to check the work queue?** — Required
3. **Did the agent claim a specific task before beginning implementation?** — High
4. **Did the agent read AGENTS.md if it was their first action in the session?** — Medium

---

## Questions This Skill Answers

- What do I do at the start of a session?
- How do I find out what work is available?
- How do I claim a task?
- What if there are uncommitted changes from a previous session?
- Where are the project conventions?
- What is the difference between system skills and task-execution skills?
- How do I know what stage the project is in?

---

## Related

- `kanbanzai-workflow` — stage gates, lifecycle, when to stop and ask
- `kanbanzai-documents` — document types, registration, approval
- `kanbanzai-agents` — context assembly, commits, task dispatch, knowledge
- `kanbanzai-planning` — how to run a planning conversation
- `write-design` — how to collaborate on design documents