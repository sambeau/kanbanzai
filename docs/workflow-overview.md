# Workflow Overview

This document describes the stage-gate workflow that Kanbanzai implements. It is
written for you — a human collaborator working alongside AI agents to build
software. If you want to understand what happens at each stage, who is
responsible for what, and where your decisions matter most, start here.

For hands-on setup and tool usage, see [Getting Started](getting-started.md).

---

## 1. The Collaboration Model

Kanbanzai coordinates work between humans and AI agents. Neither side operates
alone. The system is built around a clear division of ownership:

**You own intent.** You set goals. You decide priorities. You make architectural
choices. You approve designs and specifications. You decide when something is
good enough to ship. No agent makes these decisions for you, and no amount of
automation replaces your judgement on what to build or why.

**Agents own execution.** Once you have expressed your intent clearly — through
documents — agents break work into tasks, implement code, run verifications,
track status, and capture knowledge for future sessions. They operate within the
boundaries you set, not outside them.

**Documents are the interface between you.** The system mediates between human
intent and agent execution through structured documents: design documents,
specifications, and development plans. You write and review these documents.
Agents consume them for context and instructions. When an agent needs to
understand what to build, it reads your specification. When you need to
understand what was built, you read the agent's output and review its work.

This is a conversation through structured artifacts, not a handoff. You write a
design. An agent asks clarifying questions through checkpoints. You refine the
spec. The agent decomposes it into tasks. You review the breakdown. The agent
implements. You approve the merge. At every stage, both sides contribute — but
each side contributes what it is good at.

The workflow that follows formalises this conversation into six stages. Each
stage has a clear trigger, a defined output, and an approval gate that you
control.

---

## 2. Stage Overview

| Stage | Trigger | Output | Your Approval Gate |
|-------|---------|--------|--------------------|
| 1. Planning | You identify a body of work | Plan entity with summary and scope | You create and approve the plan |
| 2. Design | Plan activated, feature proposed | Design document in `work/design/` | You review and approve the design doc |
| 3. Specification | Design approved | Specification document in `work/spec/` | You review and approve the spec |
| 4. Dev Planning | Spec approved | Development plan with task decomposition | You review the task breakdown |
| 5. Task Execution | Tasks enter the work queue | Code changes, tests, documentation | Agent self-reviews; you spot-check |
| 6. Integration | All tasks complete | Merged feature branch, updated main | You approve the merge |

Each gate is a hard boundary. Work does not advance to the next stage until the
gate passes. This is deliberate: the cost of catching a bad decision increases
at every stage. A design problem caught during design costs minutes. The same
problem caught during task execution costs hours of rework across multiple
agents.

---

## 3. Stage Detail

### 3.1 Planning

**What happens.** You identify a body of work — a new capability, a large
refactor, a set of related improvements — and create a Plan to organise it. A
Plan groups related Features under a shared scope and purpose. You write a
summary that describes what the plan covers, why it matters, and roughly how
large it is.

**Who drives it.** You do. Planning is human-led. Agents can help by checking
for duplicate plans, suggesting scope boundaries, or surfacing related work, but
you make the call on what goes into a plan and what does not.

**Output artifact.** A Plan entity stored in `.kbz/state/plans/`. Plans are YAML
files committed to Git. Each plan has a prefix, a numeric ID, a slug, and a
summary. For example: `P3-auth-overhaul`.

**Approval gate.** You create the plan. It starts in `proposed` status. When you
are ready to begin design work, you transition it to `designing`. When design is
complete and you are ready for implementation, you move it to `active`. Each
transition is an explicit decision you make.

**Agents should:**
- Help you check whether a proposed plan duplicates existing work
- Suggest scope refinements based on the existing feature set
- Create features under a plan when you ask them to

**Agents should not:**
- Create plans without your direction
- Transition plan status without your approval
- Decide the scope or priority of a plan

---

### 3.2 Design

**What happens.** For each feature within a plan, you write a design document
that captures the architectural approach, key decisions, trade-offs considered,
and interfaces affected. The design does not need to be exhaustive — it needs to
be clear enough that a specification can be written from it, and that someone
reading it six months from now understands why you made the choices you did.

**Who drives it.** This is collaborative, but you own it. You can ask an agent
to draft sections, produce diagrams, or research alternatives. But architectural
decisions — what components to use, how data flows, what the API shape looks
like — are yours. An agent drafting a design section is writing a proposal for
your review, not making a decision.

**Output artifact.** A Markdown document in `work/design/`. You register it with
the system using `doc_record_submit` (type: `design`), which creates a document
record and links it to the owning feature. The document lives in your repository
as a normal file. The system tracks its status and content hash.

**Approval gate.** You approve the design document using `doc_record_approve`.
Approval advances the owning feature from `proposed` to `designing` status. An
approved design signals that the architectural direction is set and
specification work can begin.

**Agents should:**
- Draft sections when you ask, clearly marking them as proposals
- Research technical alternatives and present findings
- Flag inconsistencies between the design and existing system architecture
- Register the document with the system when you ask

**Agents should not:**
- Make architectural decisions
- Approve their own design proposals
- Treat a draft design as settled direction

---

### 3.3 Specification

**What happens.** You write a specification that translates the design into
precise, implementable requirements. The spec defines what the system must do
— inputs, outputs, behaviours, error handling, edge cases — in enough detail
that an agent can implement from it without guessing. This is where you define
acceptance criteria: the concrete, testable conditions that determine whether
the implementation is correct.

**Who drives it.** Collaborative, with you as the author and final authority.
Agents can help you identify gaps, suggest edge cases you may have missed, or
draft sections for your review. But the spec represents your intent. If it is
ambiguous, agents will guess, and guesses lead to rework.

**Output artifact.** A Markdown document in `work/spec/`. Registered with
`doc_record_submit` (type: `specification`) and linked to the feature. The
feature transitions to `specifying` status when specification work begins.

**Approval gate.** You approve the specification using `doc_record_approve`.
Approval advances the feature to `dev-planning` status. Once approved, the spec
becomes the authoritative source for what agents implement. Changes after
approval require superseding the document with a new version
(`doc_record_supersede`), which rolls the feature status back appropriately.

**Agents should:**
- Help you identify missing edge cases or ambiguous requirements
- Point out conflicts between the spec and existing system behaviour
- Ask clarifying questions before specification approval, not after

**Agents should not:**
- Make architecture decisions within the specification
- Change the scope of what is being built
- Treat an unapproved spec draft as implementable
- Resolve ambiguity by guessing — they should use `human_checkpoint` instead

---

### 3.4 Dev Planning

**What happens.** The approved specification is decomposed into implementable
tasks. Each task represents a vertical slice of work — a coherent change that
touches the necessary layers (data model, logic, API, tests) to deliver one
piece of observable behaviour. Tasks have explicit dependencies, size estimates,
and clear summaries tied back to spec acceptance criteria.

**Who drives it.** Agents lead this stage. They use `decompose_feature` to
propose a task breakdown based on the spec, and `slice_analysis` to identify
natural vertical slices through the system. You review the proposal. This is
where the agent's understanding of the codebase meets your understanding of the
requirements — if the decomposition does not make sense to you, reject it and
explain why.

**Output artifact.** A development plan document in `work/dev-plan/`, plus task
entities in `.kbz/state/tasks/`. Each task belongs to a feature and has a slug,
summary, dependencies, and (optionally) a story-point estimate. The feature
transitions to `dev-planning` status. The `decompose_review` tool validates the
proposal against the spec before tasks are created.

**Approval gate.** You review the task breakdown. Check that every acceptance
criterion from the spec is covered by at least one task. Check that task sizes
are reasonable (no single task should be larger than a few hours of work). Check
that dependencies make sense. Once you are satisfied, the feature advances to
`developing`.

**Agents should:**
- Propose task decompositions using `decompose_feature` and `slice_analysis`
- Identify dependencies between tasks
- Estimate task sizes
- Ensure full coverage of spec acceptance criteria

**Agents should not:**
- Create tasks without a decomposition review
- Skip dependency analysis
- Create tasks that span multiple unrelated acceptance criteria
- Proceed to implementation before you approve the breakdown

---

### 3.5 Task Execution

**What happens.** Tasks enter the work queue and are executed by agents. The
work queue (`work_queue`) automatically promotes tasks from `queued` to `ready`
status when all their dependencies have completed. An agent claims a task via
`dispatch_task`, which transitions it to `active`, sets up a worktree if needed,
and assembles a context packet with the relevant spec sections, knowledge
entries, and role conventions.

The agent implements the task — writing code, tests, and documentation — then
self-reviews using `review_task_output`, which checks the output against the
task's verification criteria and the parent feature spec. If the review passes,
the task moves to `needs-review`. If it fails, the task goes to `needs-rework`
with structured findings. Finally, the agent completes the task via
`complete_task`, which marks it `done` and unblocks any dependent tasks.

**Who drives it.** Agents lead. This is where the bulk of automated work
happens. Your role is oversight: spot-checking implementations, responding to
checkpoints, and reviewing the occasional task that needs human judgement.

**Output artifact.** Code changes, test files, and documentation in the feature
worktree. The feature is in `developing` status throughout this stage.
Knowledge entries contributed by agents during implementation are stored in
`.kbz/state/knowledge/`.

**Approval gate.** Each task has its own self-review gate via
`review_task_output`. You are not expected to review every task, but you should
spot-check regularly — especially early in a feature's implementation, when
misunderstandings are most likely and cheapest to correct.

**Agents should:**
- Assemble context before starting work (`context_assemble`)
- Implement within the boundaries set by the specification
- Self-review before marking tasks complete
- Use `human_checkpoint` when they encounter ambiguity or need a decision
- Contribute knowledge entries for non-obvious discoveries

**Agents should not:**
- Make design decisions — if something is ambiguous, escalate via checkpoint
- Change the public API or data model without human approval
- Skip self-review
- Work on tasks whose dependencies have not completed
- Guess at requirements when the spec is unclear

---

### 3.6 Integration

**What happens.** When all tasks for a feature are complete, the feature is
ready to merge. The `merge_readiness_check` tool evaluates all merge gates:
task completion, branch health, CI status, and review state. If all gates pass,
you approve the merge. The feature branch is merged to main, and the feature
transitions to `done`.

**Who drives it.** Collaborative. Agents prepare the merge (ensuring tests pass,
resolving conflicts, updating the PR description). You make the final merge
decision.

**Output artifact.** A merged feature branch on main. The feature entity moves
to `done` status. If the feature was the last one in a plan, the plan can
transition to `done` as well.

**Approval gate.** You approve the final merge to main. This is the last gate
before the code ships. `merge_readiness_check` tells you whether the technical
gates pass; the decision to merge is yours.

**Agents should:**
- Run `merge_readiness_check` and report the results
- Resolve merge conflicts when possible
- Update PR descriptions to reflect final state
- Create the PR via `pr_create` when the feature is ready

**Agents should not:**
- Merge to main without your approval
- Override failing merge gates without your explicit direction
- Skip the readiness check

---

## 4. The Document-Centric Interface

You interact with Kanbanzai primarily through documents, not by managing
entities directly. This is a deliberate design choice.

### Documents drive the workflow

The three core document types — designs, specifications, and dev plans — are
Markdown files that live in the `work/` directory of your repository:

```
work/
├── design/        ← design documents
├── spec/          ← specifications
└── dev-plan/      ← development plans
```

These are normal files. You edit them in your editor. You review them in pull
requests. You read them when you need to understand what is being built and why.

### Document lifecycle drives entity lifecycle

Approving a document advances the owning feature's status automatically:

| Document Type | On Approval | Feature Moves To |
|---------------|-------------|------------------|
| Design        | Design direction is set | `designing` |
| Specification | Requirements are locked | `dev-planning` |
| Dev Plan      | Task breakdown is approved | `developing` |

You do not need to manually update feature statuses. The document lifecycle
handles it. If a document is superseded (replaced by a newer version), the
feature status rolls back to the appropriate earlier stage.

### The document record tools

The `doc_record_*` tools manage the lifecycle of documents within the system:

- **`doc_record_submit`** — registers a document file with the system, creating
  a record in `draft` status. Computes a content hash for drift detection.
- **`doc_record_approve`** — transitions a document from `draft` to `approved`.
  Triggers the corresponding feature status transition.
- **`doc_record_supersede`** — marks an approved document as `superseded` and
  links to the replacement. May roll back the feature status.

The system also supports `research`, `report`, `policy`, and `rca` document
types for work that does not directly drive the feature lifecycle.

### Why documents, not entities

You think in documents. You write prose, draw diagrams, reason about trade-offs
in paragraphs. Requiring you to also maintain a parallel set of structured
entities — manually setting statuses, filling fields, linking records — adds
overhead without adding clarity. The system extracts structured information from
your documents. You write the design; agents create the entities.

---

## 5. Common Failure Modes

The stage-gate model exists because skipping stages is expensive. Here is what
goes wrong when the workflow is short-circuited.

### Creating tasks without a specification

When tasks are created without an approved spec, they lack clear acceptance
criteria. Agents guess at what "done" means. Different agents guess differently.
The resulting code is inconsistent, undertested, and expensive to rework. You
end up writing the spec after the fact, reverse-engineering requirements from
the implementation — which is harder and less reliable than writing them up
front.

**Fix:** Write the spec first. It does not need to be long. It needs to be
precise enough that you could tell someone whether their implementation is
correct without reading the code.

### Making architecture decisions without a design document

Without a design document, architectural decisions are implicit. They live in
chat messages, in agent context packets, in the heads of whoever happened to
be working that day. When a second agent starts working on a related feature,
it has no way to know about those decisions. It makes contradictory choices. The
code diverges. Reconciling the divergence costs more than writing the design
would have.

**Fix:** Write down your architectural decisions. The design document does not
need to be a formal architecture document. It needs to capture the choices you
made and why, so that anyone — human or agent — working on related code can
find them.

### Skipping dev planning

When tasks are created ad-hoc instead of through structured decomposition, three
things break. First, dependencies are missing or wrong, which means tasks
execute in the wrong order or block on each other unexpectedly. Second,
parallelism is unsafe — two agents working on overlapping files without
`conflict_domain_check` will produce merge conflicts or subtle bugs. Third,
estimates are meaningless because tasks are not consistently sized.

**Fix:** Use `decompose_feature` and `slice_analysis` to produce a structured
breakdown. Review it. The ten minutes you spend reviewing a task breakdown saves
hours of rework during execution.

### Conflating agent context with human workflow

Agent context packets and knowledge entries serve a specific purpose: they give
agents the right information at the right time during task execution. They are
not substitutes for your documents. A knowledge entry that says "timestamps must
be double-quoted in YAML" is useful context for an implementing agent. It is not
a specification. It does not tell you what the system is supposed to do, why it
works that way, or what the acceptance criteria are.

**Fix:** Keep the document pipeline intact. Your design documents, specs, and
dev plans are the authoritative record of intent. Agent context is operational
scaffolding — useful, but not a replacement for the artifacts you own.

---

## 6. Feature Lifecycle Summary

For reference, here is the complete feature lifecycle as driven by the
stage-gate workflow:

```
proposed → designing → specifying → dev-planning → developing → done
```

Each forward transition is triggered by a document approval or a workflow event:

| Transition | Triggered By |
|------------|-------------|
| `proposed` → `designing` | Design work begins on the feature |
| `designing` → `specifying` | Design document approved |
| `specifying` → `dev-planning` | Specification approved |
| `dev-planning` → `developing` | Dev plan approved, tasks created |
| `developing` → `done` | All tasks complete, feature merged |

Backward transitions are possible when a document is superseded. For example, if
you supersede an approved spec with a revised version, the feature rolls back to
`specifying` until the new spec is approved.

A feature can also move to `superseded` or `cancelled` from any non-terminal
state.

---

## 7. Plan Lifecycle Summary

Plans follow a simpler lifecycle that brackets the feature work:

```
proposed → designing → active → done
```

| Transition | Meaning |
|------------|---------|
| `proposed` → `designing` | You are working on the design for this body of work |
| `designing` → `active` | Design is complete, features are being implemented |
| `active` → `done` | All features in the plan are complete |

A plan can move to `superseded` or `cancelled` from any non-terminal state.

---

## What to Read Next

- **[Getting Started](getting-started.md)** — installation, configuration, and
  the session loop for day-to-day work
- **`work/design/workflow-design-basis.md`** — the full design rationale behind
  this workflow
- **`work/design/document-centric-interface.md`** — why documents are the human
  interface
- **`.kbz/context/roles/`** — context profiles that configure agent behaviour
  per role