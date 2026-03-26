---
name: kanbanzai-planning
description: >
  Use this skill during any planning conversation: when scoping new work, deciding whether
  something is one feature or a plan with multiple features, responding to a human request
  to build something new, or when a conversation might drift into design decisions before
  scope is established.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill guides a planning conversation to produce clear scope — what to build, how big
it is, and how it fits into the project — without making design or architecture decisions.
Planning produces a scope statement and a structural decision, not a planning document.

## Your Role

You are an active participant in planning, not a silent facilitator. Suggest options, flag
opportunities, and recommend directions. The human makes the final scoping decisions.

**Ambition principle:** An AI agent team is not constrained by team size — sub-agents can
be spawned for any domain in any number. The limit on what gets built is the quality of the
design, not the capacity of the team. Present the ambitious version first. Scope reduction
requires explicit reasons from the human.

## Sizing Signals

Use these signals to determine the right structure:

**One feature** (no plan needed): The scope is describable in one sentence, will produce
one design document, and can be implemented in one sprint.

**Multiple features, needs a plan:** The scope contains independently designable parts that
could be worked on in parallel. The parts would produce separate design documents. Err toward
fewer plans — a plan with one feature is usually just a feature.

**Too large to plan yet:** It is not yet clear what the individual features are. Write a
high-level design document first, then return to planning.

## Feature vs. Plan

A **feature** is a single coherent piece of behaviour that can be designed, specified, and
implemented independently.

A **plan** coordinates multiple features toward a shared milestone.

When deciding, ask: could any part of this be designed and built without the rest? If yes,
those parts are separate features and need a plan to coordinate them.

## Anti-Patterns to Name

When any of these appear in conversation, name them explicitly and redirect:

- **Premature simplification:** "Let's just do the simple version" — name it and ask what
  is being left out and why.
- **Comfort-driven scope reduction:** "That's too ambitious" — name it and ask for the
  specific reason the ambitious version is not achievable.
- **Deferred design:** "We can figure that out later" — name it and identify whether it is
  a scoping question or a design question.

## Drift Into Design

When the conversation moves into *how* something will work — data models, API shapes,
technology choices — redirect:

> "That sounds like a design question. Should we capture it in the design document?"

Planning produces scope. Design produces decisions. Do not let planning accumulate design
decisions informally — they will not be tracked or approved.

## Planning Output

A planning conversation is complete when it produces:

1. A scope statement: what is being built and approximately why
2. A structural decision: one feature, or a plan with N named features
3. A human signal to proceed to design

No formal planning document is required unless the scope is complex enough that the scope
statement would be hard to track without one.

---

## Gotchas

**Scope agreed in conversation is not recorded:** Decisions made conversationally are not
tracked unless recorded in an entity or document. For anything complex, ask whether a scope
document is needed before moving on.

**"Simple version" framing:** This often signals a desire to avoid design work, not a
genuine simplicity constraint. Ask what the simple version omits and whether those omissions
are acceptable.

---

## Related

- `kanbanzai-workflow` — stage gates and the design → features progression
- `kanbanzai-design` — what happens after planning produces a scope