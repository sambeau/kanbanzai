# Workflow Overview

Kanbanzai structures work as a conversation between humans and AI agents that moves through staged phases — flexible and iterative during design, rigorous and specification-driven during implementation. This document explains how that workflow operates: what happens at each phase, who decides what, and how approvals move work forward.

It is written for design managers and product managers who have experience with agile workflows and some familiarity with agentic development. If you want a broader orientation to the system first, start with the [User Guide](user-guide.md). If you want to try it hands-on, see the [Getting Started guide](getting-started.md).

---

## The Kanbanzai workflow

The workflow has two distinct halves. The first half — planning, design, and specification — is a collaborative design conversation. You shape ideas through documents, revise them, make decisions, and narrow alternatives until a clear specification remains. This half is agile in the original sense: iterative, flexible, driven by conversation and revision.

The second half — development planning, implementation, and review — is specification-driven. Once a specification is approved, it becomes the binding contract for what gets built. Agents decompose it into tasks and implement against it, and reviewers check the result against it. This half is deliberately rigid because the cost of ambiguity rises sharply once multiple agents are writing code in parallel.

The two halves are not in tension. Flexibility during design is how you get the specification right. Rigour during implementation is how you get the code right. Kanbanzai formalises both into a single workflow with explicit gates between phases.

---

## How Kanbanzai compares to agile and specification-driven workflows

If you have worked with Scrum or Kanban, the design phases will feel familiar. Work moves through stages, priorities shift, documents get revised, and decisions emerge from conversation rather than a single upfront plan. Kanbanzai shares agile's emphasis on iteration and responsiveness during design.

If you have worked with specification-driven systems — where a formal specification defines what gets built and review checks conformance — the implementation phases will feel familiar. After specification approval, the spec is the blueprint. Agents implement against it. Review verifies against it. Changes to scope go back through the specification, not through ad-hoc decisions during coding.

The combination is the point. Kanbanzai is agile where flexibility matters (deciding what to build) and specification-driven where consistency matters (building it correctly). The label for the implementation phase is "specification-led" — not waterfall, because there is no big-upfront-design phase and because design decisions can still send work backward at any point.

---

## Design-led workflow

The first half of the workflow is a design conversation that narrows possibilities until a clear, approved specification remains. Three stages participate in this conversation: planning, design, and specification. They are not isolated sequential steps — they form a continuous refinement of intent.

### Planning

You identify a body of work and decide how to frame it. A new capability, a refactor, a set of related improvements — you set the scope and boundaries. Agents help by inspecting existing work, spotting overlap, and surfacing related decisions, but they do not choose what belongs in the plan. Planning is human-led.

There is no automatic gate. You decide when the scope is clear enough to justify a design document. That judgment is yours because only you know whether the work is concrete enough to design or still needs more discussion.

### Design

You create or revise a design document that captures the architectural direction: the problem, the proposed approach, key decisions, trade-offs, and affected interfaces. Agents can draft sections, research alternatives, compare approaches, and flag conflicts with existing architecture. They do not settle architectural decisions on their own — that authority stays with you.

The design document must be formally approved before specification can begin. Approval here means you are satisfied with the direction, not that every detail is final. The specification stage is where details become precise.

### Specification

You turn the approved design into a precise, testable specification. This is where intent becomes binding: numbered requirements, explicit acceptance criteria, stated scope boundaries. You are the final authority because the specification defines what "correct implementation" means.

Agents help surface missing requirements, edge cases, and ambiguities. They point out conflicts between the draft and existing behaviour. But agents must not fill specification gaps by guessing — they must resolve them through conversation with you before approval.

The specification must be approved before any implementation begins. This gate exists because the cost of a specification error multiplies across every agent that implements against it.

### The design conversation in practice

In practice, these three stages overlap and loop. A planning discussion reveals a design question. A design draft exposes a specification gap. A specification review sends you back to revise the design. The workflow allows and expects backward movement — it formalises the conversation, not a conveyor belt.

The human role through all of this is **design manager**: you set direction, make trade-off decisions, approve documents, and decide when the work is ready to move forward. The agent role is **senior designer**: drafting, researching, analysing alternatives, and flagging problems for your decision.

---

## Document-led process

Documents are the durable interface between human intent and agent execution. Four document types drive the workflow, and each one's approval opens the next phase of work.

### Design documents

A design document captures architectural direction: the problem being solved, the proposed approach, alternatives considered, and key decisions. Its approval signals that the direction is set and specification work can begin.

Design documents live in `work/design/` and are registered as document records of type `design`. The system tracks their approval status and content hash so that downstream work can verify it is building on an approved foundation.

### Specifications

A specification translates the design into numbered, testable requirements with explicit acceptance criteria. Its approval signals that the requirements are binding and development planning can begin.

Specifications live in `work/spec/` and are registered as type `specification`. The specification is the most consequential document in the workflow — everything downstream traces back to it.

### Development plans

A development plan breaks the specification into implementable tasks with dependencies, sequencing, and traceability back to the specification's requirements. Its approval signals that the task breakdown is sound and implementation can begin.

Development plans live in `work/plan/` and are registered as type `dev-plan`.

### Review reports

A review report evaluates the implementation against the specification across dimensions like conformance, quality, security, and test coverage. Its approval (by you) signals that the implementation is acceptable and the feature can be completed.

Review reports are registered as type `report`.

### What the system manages vs what it stores

The Markdown files are the human-facing artefacts — you edit them in your editor, review them in conversation, and treat them as the record of intent. Document records are the system-facing metadata: ownership, approval status, content hashes, and lifecycle tracking. The records make the workflow enforceable. The files remain the thing you actually read and write.

When a document is superseded — say, a specification is rewritten because requirements changed — the owning feature moves backward to the stage that matches the new document state. This rollback is designed behaviour. It keeps the workflow aligned with current intent rather than letting implementation drift from an outdated specification.

---

## Specification-led implementation

After the specification is approved, the workflow shifts character. The design conversation is over. What remains is execution: decomposing the specification into tasks, implementing them, and verifying the result.

### The development plan

Agents break the approved specification into a development plan: a set of tasks with explicit dependencies, interface contracts where tasks interact, and traceability from each task back to the specification requirements it addresses. They lead this stage because decomposition depends on implementation-level judgement about sequencing and interfaces. You review the resulting plan and approve it before implementation begins.

The development plan matters because ad-hoc task creation — agents inventing work without structured decomposition — hides dependencies, makes parallel work unsafe, and weakens the link between implementation and specification.

### Implementation

Approved tasks enter the work queue. Agents claim ready tasks (those whose dependencies are satisfied), receive assembled context, implement changes, run verification, and complete the work. Multiple agents can work in parallel on independent tasks.

Your role shifts here. During the design phases, you were the design manager — shaping direction, making trade-off decisions, approving documents. During implementation, you are closer to a product manager: you decide priorities, answer questions that affect intent, review progress, and approve the gates that belong to you. You do not manage task assignment, sequencing, or the mechanics of implementation — agents handle that within the boundaries the specification and development plan define.

This shift is deliberate. The specification captures your intent precisely enough that agents can execute without constant direction. If they cannot — if they keep raising questions about what the specification means — that is a signal the specification needs revision, not that agents need more supervision.

### Review

After all tasks are complete, the feature moves into review. Agents evaluate the implementation against the specification across multiple dimensions — typically conformance, code quality, security, and test coverage. Each dimension produces findings classified by severity.

Blocking findings send the work back for rework. The rework cycle can repeat, but not indefinitely — if review fails repeatedly, the issue escalates to you. Non-blocking findings are reported but do not prevent the feature from advancing.

Review produces a report. You review the report and decide whether to approve the feature. This is a human gate — the system can tell you whether findings are resolved, but the decision to accept the work is yours.

### Integration

After review passes, the system prepares the change for merge and checks merge gates: task completion, branch health, CI status, and review readiness. You make the final decision to merge. After the merge lands, the feature is done.

---

## Chat-based project management

You interact with Kanbanzai through conversation in your editor, not through a separate project management interface. There is no web dashboard to update, no CLI commands to memorise, no ticket system to maintain. You talk to an AI agent, and the agent maintains the structured workflow state on your behalf.

This is more agile than a traditional interface for the design phases, where work is exploratory and iterative. A conversation can shift direction, revisit a decision, explore an alternative, and return to the main thread — all without navigating forms or updating fields. The agent translates your conversational intent into the structured records the system needs.

The AI agent fills a composite role. During design, the agent acts as a senior designer: drafting documents, researching alternatives, analysing trade-offs for your decision. During implementation, the agent acts as a project manager and development team: decomposing work, dispatching tasks, tracking progress, and implementing changes. During review, the agent coordinates specialist reviewers and assembles findings for your decision.

Through all of this, you retain decision authority. The agent proposes; you approve. The agent implements; you review. The agent reports; you decide. The structured workflow — the stage gates, the document prerequisites, the approval points — exists to make that authority enforceable rather than aspirational.

---

## Lifecycle stages and gates

Every feature and plan advances through a defined set of stages, with approval gates that control each transition. The diagram below shows the complete stage-gate flow for a feature; the tables and descriptions that follow detail each gate and its prerequisite.

```text
                    ┌─────────┐
                    │proposed │
                    └────┬────┘
                         │  you decide to begin design
                         ▼
                    ┌─────────┐
                    │designing│
                    └────┬────┘
                         │  design document approved
                         ▼
                   ┌──────────┐
                   │specifying│◄──────────────────┐
                   └────┬─────┘                   │
                        │  specification approved │ spec superseded
                        ▼                         │
                  ┌────────────┐                  │
                  │dev-planning│                  │
                  └─────┬──────┘                  │
                        │  dev plan approved      │
                        ▼                         │
                  ┌───────────┐                   │
                  │developing │                   │
                  └─────┬─────┘                   │
                        │  all tasks complete     │
                        ▼                         │
                  ┌───────────┐    rework    ┌────┴───────┐
                  │ reviewing │─────────────►│needs-rework│
                  └─────┬─────┘              └────────────┘
                        │  review approved
                        ▼
                    ┌────┐
                    │done│
                    └────┘
```

Each box is a stage. Each arrow is a gate with its prerequisite labelled. The backward arrows show how work returns to earlier stages when the basis changes.

### Feature lifecycle

A feature moves through seven stages. Each stage has a gate — a prerequisite that must be satisfied before work advances.

```text
proposed → designing → specifying → dev-planning → developing → reviewing → done
```

| Transition | Gate |
|------------|------|
| proposed → designing | You decide design work should begin |
| designing → specifying | Design document approved |
| specifying → dev-planning | Specification approved |
| dev-planning → developing | Development plan approved |
| developing → reviewing | All tasks complete |
| reviewing → done | Review approved (human gate) |

Features can also move backward. A superseded specification sends the feature back to specifying. A review with blocking findings sends it to rework. Backward movement is a designed mechanism — it keeps work aligned with current intent rather than pushing forward on an outdated basis.

A feature can also be cancelled or superseded at any point if direction changes.

### Plan lifecycle

Plans group related features at a higher level. Their lifecycle is simpler:

```text
proposed → designing → active → reviewing → done
```

A plan is `proposed` when the scope exists but design work has not started. It moves to `designing` when you begin shaping the body of work through design documents. It becomes `active` when feature delivery begins. It moves to `reviewing` when you evaluate delivery at the aggregate level. It reaches `done` when the plan review is approved and all scoped features are complete.

Plans can also become `superseded` or `cancelled` when direction changes.

### Why gates exist

The cost of catching a bad decision rises as work moves downstream. A design flaw caught during design costs a document revision. The same flaw caught after three agents have implemented against it costs rework across multiple task branches, potential merge conflicts, and wasted verification effort.

Gates are the mechanism that makes early detection possible. They force you to record decisions in documents, review them, and approve them before downstream work begins. They are prerequisites, not suggestions.

---

## Common failure modes

The stage-gate workflow exists because skipping stages or forcing gates creates expensive rework. These are the failures it prevents.

### Creating tasks without a specification

Tasks created before the specification is approved have no stable basis for correctness. Different agents fill the gaps differently, and the result is contradictory implementations that need rework. The fix: approve the specification before decomposition and implementation begin.

### Making architecture decisions without an approved design

If architectural decisions live only in chat context or temporary notes, later agents cannot rely on them. Contradictory implementation decisions follow because each agent works from a different understanding of the architecture. The fix: record the architecture in a design document and approve it before moving forward.

### Skipping development planning

Ad-hoc tasks — created without structured decomposition from the specification — hide dependencies, make parallel work unsafe, and break traceability from implementation back to requirements. When something goes wrong, there is no clear path from the failing code to the requirement it was supposed to satisfy. The fix: use structured decomposition and review the task breakdown before development begins.

### Implementing before gates are satisfied

Starting code changes before the required documents are approved is the most common downstream failure. It feels productive — code is being written — but it builds on an unstable foundation. If the specification changes, the code changes too, and now the rework cost includes both the original implementation and the revision. The fix: treat stage gates as real prerequisites. If a gate is not satisfied, stop and resolve it.

### Treating agent context as a substitute for documents

Context packets and knowledge entries help agents work efficiently within a session. They do not replace the design, specification, or development plan. Intent that lives only in agent context is invisible to other agents, invisible to review, and lost between sessions. The fix: keep intent in approved documents. Let context support execution, not define it.

---

## What to read next

Where you go from here depends on what you need:

- **Try it yourself** — the [Getting Started guide](getting-started.md) walks through installation, project setup, and your first feature from start to finish.
- **Understand orchestration** — the [Orchestration and Knowledge](orchestration-and-knowledge.md) document covers how agents receive context, claim tasks, work in parallel, and contribute knowledge.
- **Understand retrospectives** — the [Retrospectives](retrospectives.md) document covers how the system records workflow signals, synthesises themes, and generates reports.
- **Look up specifics** — the [MCP Tool Reference](mcp-tool-reference.md), [Schema Reference](schema-reference.md), and [Configuration Reference](configuration-reference.md) cover tool parameters, entity fields, and configuration keys.
- **Return to the overview** — the [User Guide](user-guide.md) provides orientation to every part of the system.
