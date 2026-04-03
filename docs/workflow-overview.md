# Workflow Overview

This document explains the workflow that Kanbanzai implements today. It is
written for you — a human collaborator working with AI agents to design,
specify, plan, implement, review, and merge software changes.

Kanbanzai is document-centric. You express intent through documents and chat.
Agents turn that intent into structured workflow state, implementation work, and
verification. If you want to understand what happens at each stage, who owns
which decisions, and which approvals move the work forward, start here.

For hands-on setup and day-to-day usage, see [Getting Started](getting-started.md).

---

## 1. The collaboration model

Kanbanzai coordinates work between humans and AI agents. Neither side works
alone, and neither side owns the whole process.

**You own intent.** You decide what to build, what matters now, what trade-offs
are acceptable, and when a document or change is good enough to approve. Design
direction, scope, priorities, and approval signals stay with you.

**Agents own execution.** Once intent is expressed clearly, agents create and
maintain the supporting workflow state, decompose work, assemble context,
implement tasks, run verification, prepare reviews, and keep progress moving.

**Documents and chat are the interface.** Humans should not need to think in
raw entity records. You write and review documents. You make decisions in
conversation. Agents mediate between that human interface and the internal
structured model.

This is not a one-way handoff. It is a staged conversation through durable
artifacts. You refine a design. An agent helps draft a specification. You
approve it. An agent decomposes it into tasks. Agents implement those tasks. You
review the results and approve the gates that belong to you.

The workflow below formalises that conversation into stages with explicit
prerequisites. Work can move backwards when a document is revised. It should not
skip forwards on guesswork.

---

## 2. Stage overview

The workflow moves from intent to implementation through seven stages. Each
stage produces an artifact that the next stage depends on, and each stage has a
clear gate that must pass before work can move on.

| Stage | What starts it | Main output | Gate to leave the stage |
|-------|----------------|-------------|--------------------------|
| Planning | You identify a body of work | Agreed scope and plan direction | You decide design work should begin |
| Design | A plan or feature needs architectural direction | Approved design document | Design document approved |
| Specification | Design direction is approved | Approved specification | Specification approved |
| Dev planning | Specification is approved | Approved dev plan and task breakdown | Dev plan approved |
| Developing | Approved tasks are ready to execute | Code, tests, docs, and completed tasks | All tasks complete |
| Reviewing | Implementation is complete | Review report and resolved findings | Human review approval |
| Integration | Review passes and merge gates pass | Merged change on `main` | Human merge approval |

These gates are deliberate. The cost of catching a bad decision rises as work
moves downstream. A design flaw caught during design costs far less than the
same flaw discovered after several agents have implemented against it.

---

## 3. Stage detail

### 3.1 Planning

**What happens.** You identify a body of work — a new capability, a substantial
refactor, or a set of related improvements — and decide how to frame it. In
Kanbanzai, planning is about agreeing on scope and ownership before design work
begins.

**Who drives it.** You do. Planning is human-led. Agents can help you inspect
existing work, spot overlap, and suggest boundaries, but they do not decide what
belongs in the plan.

**Output artifact.** A Plan entity that captures the scope of the work. The
plan record matters, but the more important output at this stage is shared human
understanding of what the work includes and where its boundaries are.

**Approval gate.** There is no automatic approval mechanism here. You decide
when the scope is clear enough to justify design work. In practice, this is the
moment when the work is concrete enough to deserve a design document rather than
more planning discussion.

**Agents should:**
- Help check whether the proposed work overlaps existing plans or features
- Surface related design documents, specs, or open bugs
- Create or update plan records when you ask

**Agents should not:**
- Create plans without your direction
- Choose the scope or priority of a plan
- Treat planning discussion as design approval

---

### 3.2 Design

**What happens.** You create or revise a design document that captures the
architectural direction for the feature: the problem, the proposed approach, key
decisions, trade-offs, and affected interfaces.

**Who drives it.** This stage is collaborative, but you own the decisions.
Agents can help draft, research, compare alternatives, and highlight conflicts.
They do not get to settle architecture on their own.

**Output artifact.** An approved design document, usually in `work/design/`,
registered as a document record of type `design`.

**Approval gate.** The design document must be approved before specification can
proceed. This is a document prerequisite, not a casual signal in chat.

**Agents should:**
- Draft sections when asked, making it clear they are proposals for review
- Research alternatives and summarise trade-offs
- Flag conflicts with existing architecture or prior decisions
- Register and refresh the document record as needed

**Agents should not:**
- Make architecture decisions on your behalf
- Treat a draft design as settled direction
- Advance past the design gate without formal approval

---

### 3.3 Specification

**What happens.** You and the agent turn the approved design into a precise,
testable specification. This is where intent becomes binding requirements and
acceptance criteria.

**Who drives it.** You are the final authority on the spec because it defines
what correct implementation means. Agents can help identify gaps, edge cases,
ambiguities, and traceability concerns.

**Output artifact.** An approved specification document in `work/spec/`,
registered as type `specification`.

**Approval gate.** The specification must be approved before dev planning can
begin. Until then, implementation should not start.

**Agents should:**
- Help surface missing requirements and edge cases
- Ask clarifying questions before approval
- Point out conflicts between the draft spec and existing behaviour

**Agents should not:**
- Fill gaps in an unapproved spec by guessing
- Change scope inside the spec without your approval
- Treat the design as a substitute for the spec

---

### 3.4 Dev planning

**What happens.** The approved specification is broken into implementable tasks
and a development plan. Each task should represent a coherent slice of work with
clear dependencies and traceability back to the specification.

**Who drives it.** Agents lead this stage because it depends on decomposition,
dependency analysis, and implementation sequencing. You review the resulting
plan and task breakdown.

**Output artifact.** An approved dev plan, usually in `work/plan/`, plus task
entities linked to the feature. The plan should show how the specification turns
into implementable slices, not just list tasks.

**Approval gate.** The dev plan must be approved before the feature should move
into development. Task decomposition is not self-approving.

**Agents should:**
- Propose tasks from the approved specification
- Record dependencies between tasks
- Keep tasks small enough to review and implement safely
- Ensure the task set covers the spec

**Agents should not:**
- Create ad-hoc tasks without structured decomposition
- Skip dependency analysis
- Start implementation before the dev plan is approved

---

### 3.5 Developing

**What happens.** Approved tasks enter the work queue. Agents claim ready tasks,
receive assembled context, implement changes, run verification, and complete the
work.

**Who drives it.** Agents do. This is the implementation stage. Your role is to
review checkpoints, answer questions that affect intent, and inspect progress
when needed.

**Output artifact.** Code changes, tests, documentation updates, knowledge
entries, and completed task records.

**Approval gate.** There is no human gate on every task, but there is still a
discipline gate: agents should work only from approved documents, on ready
tasks, with dependencies satisfied, and with verification performed before the
task is marked complete.

**Agents should:**
- Claim tasks through the work queue
- Read the assembled context before making changes
- Implement within the boundaries of the approved spec and dev plan
- Run verification and record what was checked
- Escalate ambiguity through a checkpoint instead of guessing

**Agents should not:**
- Work from an unapproved spec or dev plan
- Make design or product decisions during implementation
- Start tasks whose dependencies are incomplete
- Mark work done without verification

---

### 3.6 Reviewing

**What happens.** Once implementation is complete, the feature moves into
reviewing. Agents evaluate the result against the specification across multiple
dimensions, usually including conformance, quality, security, and testing.

**Who drives it.** Agents coordinate and perform the review work, but this stage
has a human gate. Review reports exist to help you make the approval decision.

**Output artifact.** A review report, usually recorded as a document of type
`report`, plus any resulting rework if findings block approval.

**Approval gate.** Human approval is required to leave the reviewing stage. If
blocking findings appear, the work goes back for rework before the feature can
move on.

**Agents should:**
- Review implementation against the approved specification
- Classify and report findings clearly
- Send work back for rework when findings are blocking

**Agents should not:**
- Treat a completed implementation as self-approving
- Ignore unresolved blocking findings
- Bypass the human approval gate

---

### 3.7 Integration

**What happens.** After review passes, the change is prepared for merge. Merge
gates check task completion, branch health, CI status, and review readiness
before the change lands on `main`.

**Who drives it.** This is collaborative. Agents prepare the branch, PR, and
merge checks. You make the final approval decision for merging.

**Output artifact.** A merged pull request, updated workflow state, and a
completed feature or bug.

**Approval gate.** Human approval is still required before the final merge. The
system can tell you whether the technical gates pass; it does not replace your
judgement about whether the work should land.

**Agents should:**
- Open or update the pull request when the work is ready
- Check merge gates before attempting the merge
- Report blockers clearly
- Clean up workflow state after the merge

**Agents should not:**
- Merge without your approval
- Override failing merge gates without explicit direction
- Skip review or gate checks because the change looks small

---

## 4. The document-centric interface

Kanbanzai is designed so that humans work through documents and conversation,
while agents maintain the structured bookkeeping behind the scenes.

### Documents drive the workflow

The core workflow documents are Markdown files in the repository:

```text
work/
├── design/   ← design documents
├── spec/     ← specifications
└── plan/     ← development plans and related planning documents
```

These are normal project files. You edit them in your editor, review them in
pull requests, and use them as the durable record of intent.

### Document lifecycle drives feature lifecycle

Feature progression is tied to approved documents:

| Document type | What approval means | Workflow consequence |
|---------------|---------------------|----------------------|
| Design | Architectural direction is set | Specification may begin |
| Specification | Requirements are binding | Dev planning may begin |
| Dev plan | Task breakdown is approved | Development may begin |

If a document is superseded, the owning feature can move backwards to the stage
that matches the new state of the document chain. That rollback is a feature,
not a failure. It keeps the workflow aligned with current intent.

### Document records track status, not replace the files

Document records exist so the system can track status, ownership, approval, and
content drift. The files remain the human-facing artifact. The records make the
workflow enforceable, but they are not meant to replace normal document editing
or pull-request review.

### Why documents, not raw entities

Humans reason in prose, diagrams, comparisons, and review comments. Requiring
you to operate the internal entity model directly would add ceremony without
making decisions clearer. Kanbanzai keeps the structured model strict, while
letting you work through the artifacts you already use.

---

## 5. Common failure modes

The stage-gate workflow exists because skipping stages creates expensive
rework. These are the failures it is designed to prevent, listed in the order
they usually appear.

### Creating tasks without a specification

Tasks created before the specification is approved do not have a stable basis
for correctness. Agents fill the gaps differently, and the result is rework.

**Fix:** Approve the specification before decomposition and implementation.

### Making architecture decisions without an approved design

If architectural decisions live only in chat or temporary agent context, later
agents cannot rely on them. Contradictory implementation decisions are likely to
follow.

**Fix:** Record the architecture in a design document and approve it before
moving forward.

### Skipping dev planning

Ad-hoc tasks hide dependencies, make parallel work unsafe, and weaken
traceability from implementation back to the specification.

**Fix:** Use structured decomposition and review the resulting task breakdown
before development begins.

### Treating agent context as a substitute for workflow documents

Context packets and knowledge entries help agents work efficiently. They do not
replace the design, specification, or dev plan.

**Fix:** Keep intent in approved documents. Let context packets support
execution, not define it.

### Implementing before the gates are open

A common downstream mistake is starting code changes before the required
documents are approved.

**Fix:** Treat stage gates as real prerequisites, not suggestions. If a gate is
missing, stop and resolve it before implementation continues.

---

## 6. Feature lifecycle summary

A feature moves through a staged lifecycle. In the normal forward path, that
looks like this:

```text
proposed → designing → specifying → dev-planning → developing → reviewing → done
```

The exact transition mechanics are enforced by the workflow tools, but the
high-level rule is straightforward:

- approved design unlocks specification
- approved specification unlocks dev planning
- approved dev plan unlocks development
- completed implementation unlocks review
- approved review unlocks integration, and a completed merge unlocks done

Features can also move backwards when a prerequisite document is superseded or
when review finds blocking problems that require rework.

---

## 7. Plan lifecycle summary

Plans bracket the feature work at a higher level:

```text
proposed → designing → active → done
```

At this level the idea is simpler:

- `proposed` means the scope exists but design work has not started in earnest
- `designing` means the body of work is being shaped through design documents
- `active` means features under the plan are being delivered
- `done` means the plan's scoped work is complete

Plans can also become `superseded` or `cancelled` when direction changes.

---

## What to read next

If you are new to the system, read the getting-started guide first. If you want
the reasoning behind this workflow, continue with the design documents.

- **[Getting Started](getting-started.md)** — installation, project setup, and
  the session loop
- **`work/design/workflow-design-basis.md`** — the consolidated workflow design
  basis
- **`work/design/document-centric-interface.md`** — the rationale for the
  document-centric human interface
- **`AGENTS.md`** — repository conventions and the required pre-task checklist