---
name: kanbanzai-planning
description: >
  Use when starting a new body of work, scoping a feature or plan, deciding
  what to build next, or determining whether something is one feature or many.
  Also activates for planning, scoping, and ambition questions: "is this too
  big?", "should we split this?", "how ambitious should we be?", "what would
  a great team do here?", or any question about prioritisation, roadmap, or
  scope. Use even when the user doesn't explicitly say "planning" — any
  discussion about what to build next is a planning conversation. Use even
  for seemingly small changes — unplanned work accumulates into incoherent
  systems.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Planning

## Purpose

Guide a planning conversation to produce clear scope: what to build, how big
it is, and how it fits into the existing project structure — without making
design or architecture decisions.

## When to Use

- When starting a new body of work and the scope is not yet defined
- When deciding whether something is a single feature, a plan with
  sub-features, or a small improvement to an existing feature
- When the human wants to think through what to do next
- Before creating any Plan or Feature entities

---

## The Agent's Role in Planning

Planning is human-led. The agent's job is to:

- **Ask clarifying questions** to help the human articulate scope
- **Reflect back** what the agent understands the scope to be, so the human
  can correct it
- **Suggest and recommend** — the agent can propose options, flag
  opportunities, and recommend directions, but the human makes the final
  scoping decisions
- **Flag scope issues** — things that seem too large, too small, or
  overlapping with existing work

For the general rules on what humans own vs. what agents own, see
`kanbanzai-workflow`.

---

## Planning with Ambition

An AI agent team is not constrained by team size. Sub-agents can be spawned
for any domain, in any number, at any point in the process. The limit on what
gets built is the quality of the design — not the capacity of the team. Plan
accordingly.

When scoping work:

- **Start with the best version.** What would a world-class engineering team
  at a well-resourced company design? That is the baseline. Think big: what
  would Apple or Google build if this were their product?
- **Cut scope only for real reasons.** A real constraint is a technical
  limitation, fundamental complexity, or a genuine user need that demands a
  different approach. A convenience constraint — discomfort, impatience, or
  habit — is not a reason to reduce scope.
- **Present the ambitious version first.** Let the human decide whether to
  cut scope. Scope reduction is a decision with explicit reasons, not a
  default.

If a scope seems large, that is a reason to design it carefully — not a
reason to simplify it prematurely.

### Anti-Patterns to Recognise

These patterns masquerade as pragmatism but lead to inferior outcomes. Surface
them explicitly when they appear in a planning conversation:

- **Premature simplification.** "Let's just do the simple version for now" —
  when the simple version creates design debt that the better version would
  not.
- **Scope reduction as comfort.** "That's too ambitious" — when the ambition
  is achievable and the discomfort is unfamiliarity, not genuine complexity.
- **Deferred design.** "We can figure that out later" — when figuring it out
  now costs the same and prevents locking in a weak foundation.

---

## Scope Decisions

### Feature vs. Plan

A **feature** is a single coherent piece of user-facing behaviour that can be
designed, specified, and implemented independently. It should be possible to:

- Write one design document for it
- Write one specification for it
- Implement it in a single worktree without significant conflict with other
  active features

A **plan** is a coordinating entity for a body of work that is too large or
structurally interconnected to treat as a single feature. Use a plan when:

- The work comprises multiple independent features that could be designed and
  implemented separately
- There is a high-level design document that describes how the features fit
  together
- There is a meaningful milestone or release that the features collectively
  deliver

**Err towards fewer plans.** A single feature does not need a plan. A plan
with only one feature is usually just a feature. Plans exist to coordinate
sets of related features toward a shared goal — not to add process overhead
to simple work.

### Sizing Signals

A scope is probably **one feature** if:
- It can be described in one sentence
- It would produce one design document
- It could be implemented in a focused sprint

A scope is probably **multiple features** (and needs a plan) if:
- It has clearly independent parts that could be designed or implemented
  separately
- Different agents could work on different parts in parallel
- The work would naturally produce multiple design documents

A scope is probably **too large to plan yet** if:
- It is not yet clear what the individual features are
- Fundamental questions about the direction are still open

In that case, the right next step is a high-level design document first —
not a plan.

---

## Running a Planning Conversation

Good planning questions:

- "What problem are we solving for the user?"
- "What would a user be able to do that they can't do now?"
- "Are there parts of this that are clearly independent and could be done
  separately?"
- "Is there anything in scope that would block the rest if it changed?"
- "What's out of scope for now?"

Signs that planning is drifting into design:

- Discussion of *how* something will work technically
- Discussion of data models, API shapes, or system boundaries
- Discussion of which libraries or technologies to use

When this happens, note it and bring the conversation back to scope: *"That
sounds like a design question — should we capture that as something to resolve
in the design document?"*

---

## What Planning Produces

A completed planning conversation produces:

1. **A scope statement** — one or two sentences describing what the work is
   and is not
2. **A structural decision** — one feature, or a plan with N named features
3. **Agreement to proceed** — the human signals readiness to move to design

The agent does not need to write a planning document unless the scope is large
or complex enough that it would be easy to lose track of. For most work, the
scope statement lives in the feature or plan summary field.

---

## When Planning Is Done

Planning is done when:

- The scope is agreed
- The structural decision (feature vs. plan) is made
- The human signals readiness to proceed to design

The agent then creates the appropriate entities (Plan and/or Feature) and the
workflow moves to the design stage. Entity creation requires the scope to be
clear but does not require a design document to exist yet.

---

## Templates for Planning Outputs

When planning produces decisions to create specifications or implementation plans:
- **Specifications:** `work/templates/specification-prompt-template.md`
- **Implementation plans:** `work/templates/implementation-plan-prompt-template.md`

These templates define the expected structure and quality bar for each document type.

---

## Examples

**Good planning question sequence:**
1. "What problem are we solving?" → establishes scope
2. "Is this one feature or multiple?" → sizing decision
3. "What's the most ambitious version of this?" → prevents premature simplification
4. "What would we cut if we had to ship in half the time?" → reveals priorities
5. "Are there any dependencies on other work?" → surfaces blockers early

**Planning output that's ready to proceed:**
> **Scope:** Add lifecycle gate validation to the `finish` tool.
> **Structure:** Single feature — the scope is well-defined and can be specified, planned, and implemented as one unit.
> **Agreement:** Human confirmed scope. Proceed to design.

---

## Related

- `kanbanzai-workflow` — stage gates and when each stage requires human
  approval
- `kanbanzai-design` — what happens once scope is agreed