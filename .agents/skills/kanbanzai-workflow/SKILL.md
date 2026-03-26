---
name: kanbanzai-workflow
description: >
  Use when deciding what workflow stage the current work belongs to, whether
  to proceed to the next stage, whether a state transition is valid, or when
  to stop and ask the human. Also activates for questions about lifecycle,
  stage gates, entity status, approval requirements, "can I create a feature
  now?", "is this ready for spec?", or "who needs to approve this?".
metadata:
  kanbanzai-managed: "true"
  version: "0.1.0"
---

# SKILL: Kanbanzai Workflow

## Purpose

Describe the workflow stage gates, entity lifecycle, and the rules for when
to stop and ask the human.

## When to Use

- When deciding what workflow stage the current work belongs to
- When deciding whether to proceed to the next stage
- When checking whether a state transition is valid
- When creating or transitioning entities (plans, features, tasks, bugs)
- When unsure whether human approval is needed

---

## Workflow Stage Gates

Work progresses through six stages. Each stage has a gate that must be passed
before proceeding. Do not skip stages.

| Stage | Who leads | What it produces | Gate to pass |
|---|---|---|---|
| **Planning** | Human | Agreed scope | Human signals readiness to design |
| **Design** | Human + Agent | Approved design document | Document approved before proceeding |
| **Features** | Agent | Plan + Feature entities | Design document must be approved |
| **Specification** | Human + Agent | Approved spec document | Features must exist |
| **Dev plan & tasks** | Agent | Task entities + dev plan | Spec must be approved |
| **Implementation** | Agent | Working code, tests, merged | Tasks must exist |

The stages are sequential but not rigid. Work can move backwards: a design
can be revisited after specification if a flaw is discovered. Moving backwards
is normal. Skipping forward is not.

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

### Feature

```
proposed → designing → spec-ready → dev-ready → active → done
                                                       → needs-rework → active
Any non-terminal → not-planned
```

### Task

```
queued → ready → active → done
                       → needs-review → done
                       → needs-rework → active
Any non-terminal → not-planned
```

### Bug

```
reported → triaged → active → done
                            → needs-review → done
                            → needs-rework → active
Any non-terminal → not-planned
Any non-terminal → duplicate
```

### Plan

```
proposed → designing → active → done
Any non-terminal → superseded
Any non-terminal → cancelled
```

Agents must not perform transitions that are not listed above. If a
transition is needed that does not appear here, ask the human.

---

## Stage-Specific Rules

### During planning
- Do not create entities yet — planning produces scope, not structure.
- If the conversation drifts into design, redirect it. See the
  `kanbanzai-planning` skill.

### During design
- The agent is the Senior Designer; the human is the Design Manager.
- Draft documents may contain alternatives and open questions.
- Approved documents must contain one chosen direction with no unresolved
  design questions. See the `kanbanzai-design` skill.

### During specification
- Specifications are binding contracts. Be precise and testable.
- Every acceptance criterion must be verifiable.
- Do not add scope that was not in the approved design.

### During implementation
- Assemble context before starting any task (`context_assemble`).
- Commit at logical checkpoints. A change is not done until it is committed.
- Do not commit directly to `main`. Work on feature or bug branches.
- Complete tasks with a summary, files modified, and verification performed.

---

## Related

- `kanbanzai-getting-started` — what to do at the start of a session
- `kanbanzai-planning` — how to run a planning conversation
- `kanbanzai-design` — how to collaborate on a design document
- `kanbanzai-documents` — document registration and approval workflow
- `kanbanzai-agents` — agent interaction protocol, commits, and context