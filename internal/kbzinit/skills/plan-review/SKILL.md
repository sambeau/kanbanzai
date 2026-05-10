---
name: kanbanzai-plan-review
description: >
  Use this skill when conducting a plan review after all features in a plan are
  complete. Covers scope verification, feature completion checks, spec conformance,
  documentation currency, and retrospective contribution.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill defines the procedure for conducting a plan review after all features
in a plan have reached a terminal state. It covers required inputs, each check
category, the report format, and the full orchestration sequence.

---

## Required Inputs

Before beginning a plan review, assemble all of the following:

- **Plan ID** — the full plan ID (e.g. `P11-embedded-skills`).
- **Plan's associated documents** — all documents registered under the plan,
  including design documents, specifications, dev-plans, and any decision logs.
  Retrieve with `doc(action: "list", owner: "PLAN-ID")`.
- **List of features with final statuses** — all features belonging to the plan
  and their current lifecycle status. Retrieve with
  `entity(action: "list", type: "feature", parent: "PLAN-ID")`.

Confirm all inputs are present before proceeding. If features are not yet in a
terminal state, the plan review cannot begin — surface this to the human and wait.

---

## Plan Scope Verification

Confirm that the plan's stated goals are fully addressed by the set of features
that were delivered.

**Checks to perform:**

- Read the plan's design document(s) and identify the goals, objectives, and
  intended outcomes.
- For each goal or objective: identify which feature(s) address it. Note any goal
  that has no corresponding feature.
- Identify any features that were added to the plan after the design was approved.
  Each such addition must have a corresponding decision record; if none exists,
  record a **blocking finding**: "Scope addition has no documented decision."
- Confirm no features were silently removed. If a feature is `cancelled` or
  `superseded`, a decision record or documented rationale must exist.

**Finding classification:** Any undocumented scope addition or removal is blocking.
Goals from the original design that were never addressed are blocking unless a
decision record explicitly defers or removes them.

---

## Feature Completion Checks

Verify that every feature belonging to the plan is in a terminal lifecycle state.

**Terminal states:** `done`, `cancelled`, `superseded`.

**Non-terminal states** (none of these may remain): `proposed`, `designing`,
`specifying`, `dev-planning`, `developing`, `reviewing`, `needs-rework`, `blocked`.

**Checks to perform:**

```
entity(action: "list", type: "feature", parent: "PLAN-ID")
```

- For each feature: confirm its status is terminal.
- If any feature is in a non-terminal state, record a **blocking finding** naming
  the feature and its current status.
- For each `cancelled` or `superseded` feature: confirm a rationale is documented
  (in a decision record or in the feature's summary field).

The plan review cannot produce an `approved` verdict if any feature remains
non-terminal.

---

## Spec Conformance

Verify that every `done` feature has at least one associated specification document
in `approved` status.

**Checks to perform:**

```
doc(action: "list", owner: "FEAT-...")
```

Run this for each `done` feature. For each feature, confirm:

- At least one document of type `specification` is registered and has status `approved`.
- No specification document associated with this feature remains in `draft` status.

**Finding classification:**

- A `done` feature with no `approved` specification (and no design-only exception
  recorded) → **blocking finding**.
- A spec that is still `draft` for a `done` feature → **blocking finding**.
- A spec that was superseded by a newer approved version → pass (the latest
  approved version satisfies the gate).

---

## Documentation Currency

Verify that agent-facing documentation accurately reflects the completed plan.

**Checks to perform:**

```
health()
```

- Confirm the `health` tool does not surface doc-currency warnings related to this plan.
- Check AGENTS.md: the Scope Guard section must mention the plan (by plan ID prefix
  or slug). If it does not, record a **blocking finding**: "Plan not mentioned in
  AGENTS.md Scope Guard."
- Confirm there are no specification documents in `draft` status associated with
  `done` features under this plan (cross-reference with Spec Conformance above).
- Confirm the plan's dev-plan and design documents are in `approved` status or
  have been superseded by newer approved versions.

```
doc(action: "list", owner: "PLAN-ID")
```

---

## Cross-Cutting Checks

Verify that loose ends are captured and key decisions are recorded.

**Known issues and deferred items:**

- Any known issue or defect discovered during the plan's implementation but not
  fixed must be recorded as an open `bug` entity.
- Any accepted scope that was deferred to a future plan must be recorded as a
  `feature` entity under a future plan or as a decision record explaining the deferral.
- Check for items mentioned in feature summaries or review reports as "future work"
  or "deferred" that do not have corresponding entity records — these are **blocking
  findings** if the item is not captured anywhere.

**Decision log:**

- Key architectural decisions made during the plan must be captured in decision
  records or in the plan's design documents.
- Check for significant choices (technology, data model, API shape, process changes)
  that appear in feature summaries or review reports but are absent from any
  registered document — record a non-blocking finding for each such gap.

```
entity(action: "list", type: "decision")
doc_intel(action: "find", concept: "decision")
```

---

## Retrospective Contribution

After completing all checks, contribute a retrospective signal summarising the plan:

```
retro(action: "synthesise", scope: "PLAN-ID")
```

Review the synthesised signals and contribute a summary entry:

```
retro(action: "report",
    scope: "PLAN-ID",
    output_path: "work/retro/retro-PLAN-ID.md",
    title: "Retrospective: PLAN-ID")
```

The retrospective contribution must cover:

- What worked well during this plan's execution (workflow, tooling, decomposition,
  collaboration patterns).
- Sources of friction encountered (tool gaps, spec ambiguity, workflow issues,
  rework cycles).
- Suggestions for improving the next plan's execution.

This step is mandatory. A plan review that does not produce a retrospective
contribution is incomplete.

---

## Plan Review Report Format

Write the plan review report to:

    work/review/review-{plan-id}-{slug}.md

where `{plan-id}` is the full plan ID (e.g. `P11-embedded-skills`) and `{slug}`
is the plan's slug field.

Register and approve the report immediately after writing it:

```
doc(action: "register",
    path: "work/review/review-{plan-id}-{slug}.md",
    type: "report",
    title: "Plan Review: {plan name}",
    owner: "PLAN-ID")
doc(action: "approve", id: "DOC-...")
```

**Report structure:**

### Overall Verdict

Choose exactly one:

- `approved` — all blocking checks pass; the plan may be closed.
- `changes-required` — one or more blocking findings exist; the plan must not
  be closed until they are resolved.

### Checklist

Record a pass/fail result for each check category:

| Check | Result | Notes |
|---|---|---|
| Plan scope verification | pass / fail | |
| Feature completion | pass / fail | |
| Spec conformance | pass / fail | |
| Documentation currency | pass / fail | |
| Cross-cutting checks | pass / fail | |
| Retrospective contributed | pass / fail | |

### Blocking Findings

List each blocking finding with:
- Check category
- Description of the problem
- Required resolution

### Non-Blocking Findings

List each non-blocking finding with:
- Check category
- Description

---

## Orchestration Sequence

```
# 1. Assemble context
status(id: "PLAN-ID")
entity(action: "list", type: "feature", parent: "PLAN-ID")
doc(action: "list", owner: "PLAN-ID")

# 2. Run each check category (sections above)
health()

# 3. For each done feature, check spec documents
doc(action: "list", owner: "FEAT-...")

# 4. Contribute retrospective
retro(action: "synthesise", scope: "PLAN-ID")
retro(action: "report", scope: "PLAN-ID",
    output_path: "work/retro/retro-PLAN-ID.md",
    title: "Retrospective: PLAN-ID")

# 5. Write and register plan review report
doc(action: "register",
    path: "work/review/review-{plan-id}-{slug}.md",
    type: "report",
    title: "Plan Review: {plan name}",
    owner: "PLAN-ID")
doc(action: "approve", id: "DOC-...")

# 6. Transition plan if approved
entity(action: "transition", id: "PLAN-ID", status: "done")
```

---

## Related

- `kanbanzai-review` — feature-level code review procedure; run before plan review
- `kanbanzai-workflow` — feature and plan lifecycle states, stage gates
- `kanbanzai-documents` — document registration, approval, drift, and supersession