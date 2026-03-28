---
name: kanbanzai-workflow
description: >
  Use when deciding what workflow stage work belongs to, whether to proceed
  to the next stage, whether a state transition is valid, or when to stop
  and ask the human. Activates for any question about lifecycle, stage gates,
  entity status, approval requirements, or uncertainty about whether to
  proceed — including "can I create a feature now?", "is this ready for
  spec?", "who needs to approve this?", or "should I stop and ask?". Use
  even when the agent is confident about the next step — workflow errors are
  expensive to undo.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Workflow

## Purpose

The workflow stage gates, entity lifecycle, and the rules for when to stop
and ask the human.

## When to Use

- When deciding what workflow stage work belongs to
- When deciding whether to proceed to the next stage
- When checking whether a state transition is valid
- When creating or transitioning entities (plans, features, tasks, bugs)
- When unsure whether human approval is needed

---

## Workflow Stage Gates

Work progresses through six stages. Each stage has a gate that must be passed
before proceeding.

| Stage | Who leads | What it produces | Gate to pass |
|---|---|---|---|
| **Planning** | Human | Agreed scope | Human signals readiness to design |
| **Design** | Human + Agent | Approved design document | Document approved before proceeding |
| **Features** | Agent | Plan + Feature entities | Design document must be approved |
| **Specification** | Human + Agent | Approved spec document | Features must exist |
| **Dev plan & tasks** | Agent | Task entities + dev plan | Spec must be approved |
| **Implementation** | Agent | Working code, tests, merged | Tasks must exist |

The full stage progression applies to features and plans. Bug fixes and small
improvements follow a lighter path — they do not need design documents or
specifications unless the fix involves a significant architectural change.

Work can move backwards — a design can be revisited after specification if a
flaw is discovered. Moving backwards is normal. Skipping forward is not.

---

## Human Ownership vs. Agent Ownership

**Humans own:** intent, priorities, approvals, product direction, and scope
decisions. Only a human can approve a design, approve a specification, or
decide what to build.

**Agents own:** execution, decomposition, implementation, tracking, and
verification. Agents create entities, dispatch tasks, write code, run tests,
and manage workflow state.

When in doubt about which side a decision falls on, it belongs to the human.

---

## The Emergency Brake

Stop and ask the human when any of these conditions are true:

- **Design content without an approved design.** You are about to write a
  document containing architecture decisions, technology choices, or system
  design without an approved design document to anchor it.
- **Entities without an approved design.** You are about to create Plan,
  Feature, or Task entities and no approved design document exists for the
  work.
- **Technology or architecture choices.** You are about to make a technology
  selection, define an API shape, choose a data model, or decide on system
  boundaries without explicit human approval.
- **Stage confusion.** You are unsure which workflow stage the current work
  belongs to.
- **Scope change.** The work you are doing has drifted beyond the scope of
  the task, feature, or plan you were given.

When the emergency brake fires, stop and ask. Do not proceed with a guess.
The cost of asking is low. The cost of building the wrong thing is high.

---

## Entity Lifecycle Transitions

For the legal state transitions for each entity type (feature, task, bug,
plan), see [references/lifecycle.md](references/lifecycle.md).

Agents must not perform transitions that are not listed there. If a
transition is needed that does not appear, ask the human.

---

## Stage-Specific Rules

### During planning
- Do not create entities yet — planning produces scope, not structure.
- See `kanbanzai-planning` for how to conduct a planning conversation.

### During design
- See `kanbanzai-design` for the design process, roles, and approval
  criteria.

### During specification
- Specifications are binding contracts. Be precise and testable.
- Every acceptance criterion must be verifiable.
- Do not add scope that was not in the approved design.

### During implementation
- See `kanbanzai-agents` for the dispatch-and-complete protocol and commit
  format.

---

## Documentation Accuracy

- **Code is truth** — if documentation conflicts with code, fix the
  documentation.
- **Spec is intent** — if code conflicts with the specification, surface
  the conflict to the human.
- Do not silently resolve spec-vs-code conflicts in either direction without
  human input.

---

## Gotchas

- If a Kanbanzai tool call fails (e.g., `entity` action: `transition`
  rejects a transition), read the error message — it usually names the valid
  transitions or states. Do not retry with the same arguments.
- The stage gates apply to the *entity type*, not the size of the work. A
  quick fix to a document doesn't need a full pipeline, but creating a new
  Feature entity always requires an approved design.
- Verbal approval ("LGTM", "Approved", "Let's move on") is sufficient to
  pass a gate. Record it with the appropriate tool call
  (`doc` action: `approve`) immediately so the system state matches reality.

---

## Related

- `kanbanzai-getting-started` — what to do at the start of a session
- `kanbanzai-planning` — how to run a planning conversation
- `kanbanzai-design` — how to collaborate on a design document
- `kanbanzai-documents` — document registration and approval workflow
- `kanbanzai-agents` — agent interaction protocol, commits, and context