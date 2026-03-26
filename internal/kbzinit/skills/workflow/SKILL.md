---
name: kanbanzai-workflow
description: >
  Use this skill whenever you are making progress decisions: deciding what to build,
  whether to proceed to the next stage, whether a state transition is valid, or whether
  to create new entities. Also activates when resolving entity lifecycle errors,
  determining human vs. agent responsibilities, or deciding whether to stop and ask.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill defines the workflow stage gates, human/agent ownership boundary, and the
conditions that require you to stop and ask the human.

## Stage Gates

Work progresses through six stages. Each stage has a gate that must pass before
proceeding to the next.

| Stage | Who leads | Output | Gate to pass |
|---|---|---|---|
| Planning | Human | Agreed scope | Human signals readiness to design |
| Design | Human + Agent | Approved design document | Document approved |
| Features | Agent | Plan + Feature entities | Design document approved |
| Specification | Human + Agent | Approved spec document | Features exist |
| Dev plan & tasks | Agent | Task entities + dev plan | Spec approved |
| Implementation | Agent | Working code, tests, merged | Tasks exist |

Bug fixes and small improvements follow a lighter path — no design document or
specification needed unless the fix involves a significant architectural change.

## Human vs. Agent Ownership

**Humans own:** intent, priorities, approvals, and product direction. Humans make
technology choices, approve design documents, and accept completed work.

**Agents own:** execution — decomposing work, implementing, verifying, tracking status,
and maintaining consistency — within the guardrails set by humans.

Agents never:
- Approve their own work
- Make final architecture or technology decisions
- Proceed past a stage gate without human approval
- Create design content without an approved design document

## Emergency Brake

Stop and ask the human before proceeding if any of the following are true:

- You are about to write design content (data models, API shapes, technology choices)
  without an approved design document
- You are about to create Plan, Feature, or Task entities without an approved design
- You are about to make a technology or architecture choice without explicit human approval
- You are unsure which workflow stage the current work belongs to
- Work has drifted beyond the scope of the current task, feature, or plan

When in doubt, surface the question rather than guessing.

## Entity Lifecycle

Entity lifecycle transitions are enforced. Invalid transitions are rejected with an error
that names the valid transitions from the current state.

Load `references/lifecycle.md` for transition diagrams covering feature, task, bug, and
plan entities.

---

## Gotchas

**Tool call failures:** Read the error message. It names the valid transitions from the
current state. Do not retry with the same arguments — identify the correct transition first.

**Stage gates apply by entity type, not work size:** A one-line bug fix and a major feature
both follow the same lifecycle for their entity type. The size of the work does not change
the rules.

**Verbal approval must be recorded immediately:** When a human approves a document in
conversation, call `doc_record_approve` immediately. Verbal approval that is not recorded
does not satisfy the stage gate — the next operation will fail.

---

## Related

- `kanbanzai-getting-started` — session orientation
- `kanbanzai-documents` — registration, approval, drift, supersession
- `kanbanzai-agents` — context assembly, task dispatch, commit format
- `references/lifecycle.md` — entity lifecycle transition diagrams