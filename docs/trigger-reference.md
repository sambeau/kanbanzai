# Trigger Reference

> This reference shows every phrase the system responds to, what must already be true for that phrase to work, exactly what the agent will do, and — crucially — what it will *not* do. The **"Does not produce"** row is as important as **"Produces"**: most confusion in practice comes from assuming side-effects that don't happen.
>
> For a narrative guide organised by workflow stage, see the [Conversational Guide](conversational-guide.md).
> For a conceptual overview of the workflow, see the [Workflow Overview](workflow-overview.md).

---

## System — Getting Your Bearings

### Check project status

**Say any of:**
- "What's the current status of the project?"
- "Show me the project dashboard"
- "What's ready to work on?"
- "What's blocked?"

**Requires:** Nothing — valid at any point in the workflow.

**Produces:**
- Summary of all active plans and features with their current lifecycle states
- Work queue: ready tasks sorted by priority
- Attention items: blocked work, features awaiting approval, overdue items
- Progress metrics per plan

**Does not produce:** Any state changes. Status checks are entirely read-only.

**Use when:** Starting any session. When returning after a gap. Any time you're unsure where things stand.

---

### Check a specific feature or task

**Say any of:**
- "What's the status of [feature name]?"
- "Show me FEAT-xxx"
- "What stage is [feature] at?"
- "Is the design for [feature] approved?"
- "How many tasks are done for [feature]?"

**Requires:** The entity exists.

**Produces:** Lifecycle state, document approval status for all associated documents, task completion percentage, and any blocking attention items.

**Does not produce:** Any state changes.

---

## Planning — Deciding What to Build

Planning is a free conversation. There are no formal trigger phrases — anything that describes new work, questions scope, or discusses what to build next is treated as a planning conversation.

### Start a planning conversation

**Say things like:**
- "I want to add [capability] to the system."
- "Here's my thinking on [topic] — can you give me some feedback?"
- "Should this be one feature or two?"
- "What would a complete implementation of [X] look like?"
- "Is this too big to tackle in one go?"
- "What should we work on next?"

**Requires:** Nothing formal.

**Produces:**
- A scoping conversation leading to agreed scope
- Once scope is agreed: Plan entity and/or Feature entities created at your explicit direction

**Does not produce:**
- Any document (design, spec, dev-plan)
- Automatic entity creation — entities are created explicitly once scope is agreed, not inferred

---

### Create a Plan entity

**Say:**
- "Create a Plan entity for [name]"
- "Set up a plan entity for [work description]"

> ⚠️ Do not say just "create a plan" — the word is overloaded. See [Dangerous Phrases](#️-dangerous-phrases) below.

**Requires:** Agreed scope from a planning conversation.

**Produces:** A Plan entity in `proposed` status.

**Does not produce:** Feature entities (must be created separately). Any document.

---

### Create a Feature entity

**Say:**
- "Create a Feature entity for [name]"
- "Add a feature for [X] under plan [Y]"
- "Set up [feature name] as a feature"

**Requires:** Clear scope. A parent plan entity if the feature belongs to a plan (plans are optional — features can stand alone).

**Produces:** A Feature entity in `designing` status, ready for design work to begin.

**Does not produce:** Any document. A Plan entity (not required).

---

## Designing — How It Should Work

### Write a design document

**Say any of:**
- "Write a design document for [feature]"
- "Draft a design for [feature]"
- "Create a design document for [feature]"
- "Author the design for [feature]"
- "Produce a design for this feature"

> ❌ Not a trigger: "Write a draft document" — too vague; the agent doesn't know which document type to write.

**Requires:**
- A Feature entity in `designing` status
- Agreed scope

**Full effect chain:**
1. Agent searches the document corpus for related prior decisions and constraints *(non-optional — design without corpus consultation is an anti-pattern)*
2. Agent identifies at least two candidate approaches and their trade-offs
3. Agent drafts all four required sections: Problem and Motivation, Design, Alternatives Considered, Decisions
4. Agent runs structural validation
5. Document registered in corpus as **draft** (not approved)
6. Agent presents the draft to you for review

**Produces:**
- A design document file (typically in `work/design/`)
- A registered draft document record

**Does not produce:**
- Plan or Feature entities
- A specification or dev-plan
- Any lifecycle state change on the feature (stays in `designing`)
- An approved document — approval is always a separate, explicit step

**After this:** Review the document. Iterate as needed. When satisfied, say "Approved" or "LGTM" — that is the only way to formally open the specifying stage gate.

---

### Approve the design

**Say any of:**
- "Approved"
- "LGTM"
- "Looks good, let's proceed"

**Requires:** A draft design document has been presented for review.

**Produces:**
- Document status changes from `draft` to `approved`
- Specifying stage gate opens
- Approval permanently recorded

**Does not produce:**
- Automatic advance of the feature lifecycle state
- Automatic writing of the specification
- Any code changes

---

## Specifying — What Exactly to Build

### Write a specification

**Say any of:**
- "Write a specification for [feature]"
- "Draft a spec for [feature]"
- "Create a specification document for [feature]"
- "Author a spec from this design"
- "Produce acceptance criteria for [feature]"

**Requires:**
- An **approved** design document for this feature
- Feature entity exists (in `specifying` status or advancing to it)

> If no approved design exists, the agent will stop and tell you. It will not write a spec from nothing, and you should not try to force it to.

**Full effect chain:**
1. Agent reads the approved design document fully
2. Agent checks related specifications for consistency (shared boundaries with adjacent features)
3. Agent derives numbered functional and non-functional requirements from the design
4. Agent writes testable acceptance criteria for each requirement
5. Agent builds a verification plan mapping each acceptance criterion to a verification method
6. Agent runs structural validation
7. Document registered as **draft**, presented to you

**Produces:**
- A specification document (typically in `work/spec/`)
- A registered draft document record

**Does not produce:**
- Task entities
- A dev-plan document
- Any lifecycle state change
- An approved document

**After this:** Review carefully — the spec is the binding contract for implementation. When satisfied, say "Approved."

> **Note for multi-feature plans:** Each Feature needs its own specification. There is no "write specs for all features in this plan" command — you must request and approve each one individually.

---

## Dev-Planning — How to Implement It

> ⚠️ **The dev-planning stage has two distinct parts:** writing the plan *document* and creating the *task entities*. These are separate operations with separate triggers. Triggering one does not automatically trigger the other. The recommended approach is to request both together using the compound phrase below.

### Write an implementation plan (dev-plan document)

**Say any of:**
- "Write a dev-plan for [feature]"
- "Write an implementation plan for [feature]"
- "Create a dev-plan for [feature]"
- "Author a development plan for [feature]"

**Requires:** An **approved** specification for this feature.

**Full effect chain:**
1. Agent reads the specification fully: requirements, acceptance criteria, constraints
2. Agent decomposes requirements into proposed tasks
3. Agent builds a dependency graph; identifies the critical path and parallelisable groups
4. Agent assesses risks on the critical path
5. Agent defines the verification approach — maps acceptance criteria to specific tasks
6. Agent runs structural validation
7. Document registered — dev-plans may be **auto-approved** by the agent without your explicit approval (unlike design and spec documents)
8. Document presented to you

**Produces:**
- A dev-plan document (typically in `work/plan/` or `work/dev/`)
- A registered, possibly auto-approved document record

**Does not produce:**
- Task entities — the document describes the plan; the actual tasks must be created in a separate step
- Any lifecycle state change on the feature

> ⚠️ **Critical:** Approving the dev-plan does **not** automatically advance the feature to `developing`. After approval, you must explicitly say "Advance [feature] to developing" — otherwise nothing proceeds.

---

### Decompose a feature into tasks

**Say any of:**
- "Decompose [feature] into tasks"
- "Break down [feature] for implementation"
- "Create the task breakdown for [feature]"
- "Plan the tasks for [feature]"
- "Generate implementation tasks from [feature]'s spec"

**Requires:** An approved specification (and ideally an approved dev-plan document).

**Full effect chain:**
1. Agent reads the feature's spec and design
2. Agent proposes a decomposition using vertical slices (end-to-end behaviour, not horizontal layers)
3. Agent validates the proposal on five criteria: non-empty descriptions, explicit dependency declarations, single-agent sizing, no circular dependencies, integration and test tasks present
4. Agent creates task entities in the workflow system
5. Dependency relationships wired between tasks

**Produces:**
- Task entities visible in the work queue
- Dependency graph relationships between tasks

**Does not produce:**
- A dev-plan document (that is the separate operation above)
- Any lifecycle state change

---

### Do both at once (recommended)

**Say:**
- "Write a dev-plan and decompose [feature] into tasks"
- "Write the dev-plan for [feature] and create the tasks"

This is the canonical dev-planning phrase. It combines both operations and ensures neither is accidentally omitted.

---

### Advance feature to developing

After the dev-plan is approved and tasks exist:

**Say any of:**
- "Advance [feature] to developing"
- "Move [feature] to developing status"
- "Start development on [feature]"

**Produces:** Feature lifecycle state transitions to `developing`. Tasks become claimable from the work queue.

**Does not produce:** Any code. This is purely a lifecycle state change.

---

## Developing — Building It

### Orchestrate full feature development

**Say any of:**
- "Orchestrate development for [feature]"
- "Run the dev-plan for [feature]"
- "Coordinate implementation of [feature]"
- "Dispatch sub-agents for [feature]"
- "Manage parallel task execution for [feature]"

**Requires:**
- Feature in `developing` status
- Task entities exist with dependency wiring
- Dev-plan is approved

**Full effect chain:**
1. Agent builds the ready frontier — tasks whose dependencies are all `done`
2. Agent checks for file-scope conflicts between parallel-dispatchable tasks
3. Agent dispatches sub-agents to implement ready tasks in parallel
4. Agent waits for outcomes; reduces each completed sub-agent's output to a short summary to manage context budget across long features
5. Agent verifies each task reached `done` before dispatching dependent tasks
6. On task failure: one retry with updated guidance; if the second attempt also fails, the agent escalates to you via a checkpoint
7. Cycle repeats until all tasks are `done`

**Produces:**
- Code changes committed per task to the feature branch
- Task entities advancing through `active` → `done`
- Knowledge entries contributed at task completion (lessons learned, gotchas)

**Does not produce:**
- Any code review — that is a separate stage after all tasks are complete
- Automatic merge or pull request

**You will be interrupted when:** A sub-agent encounters a genuine decision requiring your input — an architecture choice not covered by the spec, an ambiguous requirement, or a task that has failed twice without a path forward.

---

### Implement a specific task

**Say any of:**
- "Implement task [TASK-xxx]"
- "Execute implementation work on [task name]"
- "Build [task name]"
- "Write code for [task description]"

**Requires:** A specific task entity in `ready` status.

**Produces:** Code changes committed to the feature branch; task transitions to `done`.

---

## Reviewing — Checking It

### Orchestrate code review (full specialist panel)

**Say any of:**
- "Orchestrate a code review for [feature]"
- "Run the review for [feature]"
- "Coordinate the review team for [feature]"
- "Dispatch review sub-agents for [feature]"
- "Collate review findings for [feature]"

**Requires:**
- All tasks for the feature are complete (in `done` status)
- Feature is in (or advancing to) `reviewing` status

**Full effect chain:**
1. Agent groups changed files into coherent review units
2. Agent dispatches up to 4 specialist sub-agents in parallel:
   - **Conformance** — does the code implement the specification? Each requirement traced to code.
   - **Quality** — code structure, clarity, maintainability, complexity
   - **Security** — vulnerabilities, input handling, data exposure
   - **Testing** — test coverage, test quality, missing edge cases
3. Agent validates each sub-agent's output has evidence citations, not just verdicts
4. Agent deduplicates findings raised by multiple reviewers at the same location
5. Agent collates into a single review report with findings categorised as blocking or advisory
6. Report registered; presented to you

**Produces:**
- A review report document with findings as blocking or advisory
- Tasks may transition to `needs-rework` for blocking findings

**Does not produce:**
- Automatic resolution of findings
- Any merge or deployment
- Your approval — you decide which findings are acceptable

**After this:** Resolve any blocking findings (the agent creates rework tasks). When all blocking findings are addressed, say "Approved" to close the review and advance the feature to `done`.

---

### Review a single dimension

**Say any of:**
- "Review the code changes for [feature]"
- "Evaluate [feature] against its specification"
- "Check code quality for [feature]"
- "Produce review findings for [feature]"

**Produces:** Single-dimension review findings. This is one reviewer's perspective — not the full four-reviewer panel.

---

### Review plan completion

**Say any of:**
- "Review plan [X] for completion"
- "Check plan delivery status for [plan]"
- "Verify plan readiness for [plan]"
- "Audit plan conformance for [plan]"

**Requires:** All features in the plan are in a terminal state (done, cancelled, or superseded).

**Produces:**
- Aggregate delivery review across all features in the plan
- Conformance gaps: features not in terminal state, specs not approved, docs out of date
- A pass/fail verdict for the plan

---

## Documentation

### Update existing documentation

**Say any of:**
- "Update the documentation"
- "Fix stale docs"
- "Update the docs to reflect recent changes"
- "The documentation needs updating"
- "Refresh the project documentation"

**Produces:** Updated documentation files. No lifecycle state changes.

---

### Write new documentation

**Say any of:**
- "Write documentation for [topic]"
- "Draft a document on [topic]"
- "Create a README for [module]"
- "Write a getting-started guide for [topic]"
- "Write a manual for [feature]"
- "Author technical documentation for [topic]"

**Produces:** New documentation file, registered in corpus.

---

### Run the editorial pipeline

**Say any of:**
- "Run [document] through the editorial pipeline"
- "Publish [document] through the editorial pipeline"
- "Refine the documentation for [document]"
- "Run [document] through all editorial stages"
- "Coordinate editing of [document]"

**Produces:** Document passes through five sequential stages: write → edit → check → style → copyedit. A completion summary collating all stage changelogs is produced at the end.

Individual stages can also be triggered independently:

| Stage | Trigger phrases |
|---|---|
| Structural edit | "Edit document structure of [doc]", "developmental edit [doc]", "check document architecture of [doc]" |
| Fact-check | "Fact-check [doc]", "verify document accuracy", "QA [doc]", "check [doc] for hallucinations" |
| Style | "Humanise the prose in [doc]", "remove AI artifacts from [doc]", "strip AI clichés from [doc]" |
| Copy-edit | "Copy edit [doc]", "polish prose in [doc]", "proofread [doc]", "fix passive voice in [doc]" |

---

## Research

### Write a research report

**Say any of:**
- "Write a research report on [topic]"
- "Research [topic] for [feature]"
- "Investigate options for [topic] and produce a report"
- "Author a research document on [topic]"
- "Conduct research on [topic] and document the findings"

**Produces:** Research report document, registered in corpus. Informs design decisions but creates no entities and advances no lifecycle state.

---

## Codebase Audit

### Audit the codebase

**Say any of:**
- "Audit the codebase"
- "Run a quality sweep"
- "Check for dead code"
- "Code quality audit"
- "Sweep for quality issues"
- "Pre-release quality check"
- "360 review of the codebase"
- "Find linting errors across the whole project"

**Produces:** Quality report with findings across the full codebase. No lifecycle state changes.

---

## Approval Signals

Approvals are **explicit workflow actions**, not casual agreement. When you say one of these phrases while reviewing a document, the agent records your approval formally. Make sure you mean it.

| Say | Effect |
|---|---|
| "Approved" | Document status → `approved`; relevant stage gate opens |
| "LGTM" | Same as "Approved" |
| "Looks good, let's proceed" | Same as "Approved" |
| "Let's proceed" | Treated as approval when a document is under review |
| "Go ahead" | May be interpreted as approval — be explicit if you mean it |

**What approval does:**
1. Sets document status from `draft` to `approved` in the corpus
2. Opens any stage gate that required that document type as a prerequisite
3. Permanently recorded — survives all sessions; cannot be undone without explicit supersession

**What approval does NOT do:**
- Does not write the next document automatically
- Does not advance the feature lifecycle state — that is always a separate explicit step
- Does not merge or deploy any code
- Cannot be reversed with an informal "actually, let's change that" in conversation

**Informal agreement is not an approval.** Phrases like "that looks fine", "I like it", "not bad", or a casual "OK" said mid-conversation do not trigger formal recording. Only the explicit signals in the table above do.

---

## ⚠️ Dangerous Phrases

These phrases either are not in the trigger vocabulary, are ambiguous between multiple operations, or risk silently skipping an approval gate.

| Phrase | Problem | Use instead |
|---|---|---|
| "Create a plan" | Ambiguous: Plan entity, dev-plan document, or informal plan? | "Create a **Plan entity** for X" or "Write a **dev-plan** for X" |
| "Write a draft document" | Too vague — which document type? | "Write a **design document**", "Write a **spec**", "Write a **dev-plan**" |
| "Write a design and a spec" | Requests both in one go — will skip the approval gate between them | Say them separately; approve the design before asking for the spec |
| "Start building" (without tasks) | May attempt development when no tasks exist yet | Only use after tasks are created and the feature is in `developing` |
| "Let's just implement this" | May skip design, spec, and dev-planning entirely | Follow the stage gates; if you have a reason to skip, state it explicitly |
| "Write an implementation plan" | Writes the dev-plan document only — tasks are NOT created | Use "Write a dev-plan **and decompose** [feature] into tasks" |
| "Convert this design into a plan" | No trigger exists for this operation | "Create a Plan entity for this, then create Feature entities for X, Y, Z" |
| "Write specs for all features" | No trigger — spec is strictly per-feature | Request each feature's specification individually |
| "This looks good" | Not recorded as an approval | Say "**Approved**" when you intend to formally approve |
| "Can we skip the design?" | No conversational skip trigger | Say "I want to skip the design stage for [feature] because [reason]" |

---

*Trigger phrases are sourced from the `triggers:` frontmatter in `.kbz/skills/*/SKILL.md` and the approval signal vocabulary in `.kbz/skills/kanbanzai-workflow/SKILL.md`. Stage prerequisites are sourced from `.kbz/stage-bindings.yaml`.*
