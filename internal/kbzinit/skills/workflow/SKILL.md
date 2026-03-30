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

Work progresses through seven stages. Each stage has a gate that must pass before
proceeding to the next.

| Stage | Who leads | Output | Gate to pass |
|---|---|---|---|
| Planning | Human | Agreed scope | Human signals readiness to design |
| Design | Human + Agent | Approved design document | Document approved |
| Features | Agent | Plan + Feature entities | Design document approved |
| Specification | Human + Agent | Approved spec document | Features exist |
| Dev plan & tasks | Agent | Task entities + dev plan | Spec approved |
| Implementation | Agent | Working code, passing tests | Tasks exist |
| Review | Agent | Review report, verdict | Implementation complete; feature in `reviewing` |

Implementation does **not** proceed directly to merge. After implementation is complete,
the feature must pass through the Review stage. See **Feature Lifecycle States** below
and read `kanbanzai-review` for the full review procedure.

Bug fixes and small improvements follow a lighter path — no design document or
specification needed unless the fix involves a significant architectural change.

---

## Feature Lifecycle States

Feature entities move through an enforced set of lifecycle states. The complete
path from creation to merge is:

    proposed → designing → specifying → dev-planning → developing → reviewing → done

There is no shortcut from `developing` directly to `done`. Every feature must pass
through `reviewing` before it may be marked done or merged.

### `reviewing`

- **Entered from:** `developing`, when the implementing agent has completed all tasks
  and the implementation is ready for review.
- **Meaning:** Implementation is complete; a code review is in progress. The feature
  may not be merged until the review concludes with an `approved` or
  `approved-with-notes` verdict.
- **Who transitions to this state:** The implementing agent, after all tasks are `done`
  or `cancelled`.
- **Exit transitions:** → `done` (review approved) or → `needs-rework` (review found
  blocking issues).

To transition into reviewing:

```
entity(action: "transition", id: "FEAT-...", status: "reviewing")
```

For the full review procedure, read `kanbanzai-review`.

### `needs-rework`

- **Entered from:** `reviewing`, when the reviewer finds one or more blocking findings
  that prevent merge.
- **Meaning:** The feature has blocking findings that must be addressed before it can
  be re-reviewed and merged.
- **Who transitions to this state:** The reviewing agent, after writing the review
  report with a `changes-required` verdict.
- **Exit transitions:**
  - → `reviewing`, after the implementing agent has addressed all blocking findings
    and the feature is ready for re-review.
  - → `developing`, if the required changes are substantial enough to warrant
    reopening tasks and resuming full implementation work.

To address rework and return to review:

```
entity(action: "transition", id: "FEAT-...", status: "reviewing")
```

The knowledge base will contain a rework-findings entry contributed by the reviewer.
Read it before beginning rework.

---

## Human vs. Agent Ownership

**Humans own:** intent, priorities, approvals, and product direction. Humans make
technology choices, approve design documents, and accept completed work.

**Agents own:** execution — decomposing work, implementing, reviewing, verifying,
tracking status, and maintaining consistency — within the guardrails set by humans.

Agents never:
- Approve their own work
- Make final architecture or technology decisions
- Proceed past a stage gate without human approval
- Create design content without an approved design document
- Merge a feature that has not passed the Review stage

---

## Emergency Brake

Stop and ask the human before proceeding if any of the following are true:

- You are about to create a document in `work/` or modify entity state in `.kbz/state/`
  by writing files directly instead of using the corresponding kanbanzai MCP tool
  (`doc`, `entity`). Direct file writes bypass lifecycle enforcement and health checks.
- You are about to write design content (data models, API shapes, technology choices)
  without an approved design document
- You are about to create Plan, Feature, or Task entities without an approved design
- You are about to make a technology or architecture choice without explicit human approval
- You are unsure which workflow stage the current work belongs to
- Work has drifted beyond the scope of the current task, feature, or plan
- A review has returned `changes-required` and the scope of rework is unclear

When in doubt, surface the question rather than guessing.

---

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

**Implementation does not merge directly:** After all tasks are done, transition the feature
to `reviewing` and follow the review procedure in `kanbanzai-review`. Do not call
`merge` until a review report with an approved verdict exists.

**Verbal approval must be recorded immediately:** When a human approves a document in
conversation, call `doc` with action: `approve` immediately. Verbal approval that is not
recorded does not satisfy the stage gate — the next operation will fail.

---

## Related

- `kanbanzai-getting-started` — session orientation
- `kanbanzai-documents` — registration, approval, drift, supersession
- `kanbanzai-agents` — context assembly, task dispatch, commit format
- `kanbanzai-review` — code review procedure
- `references/lifecycle.md` — entity lifecycle transition diagrams