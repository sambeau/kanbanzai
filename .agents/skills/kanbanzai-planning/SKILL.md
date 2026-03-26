<!-- kanbanzai-managed: kanbanzai-planning v0.1.0 -->
# SKILL: Kanbanzai Planning

## Purpose

Guide a planning conversation to produce clear scope: what to build, how big it is, and how it
fits into the existing project structure — without making design or architecture decisions.

## When to Use

- When starting a new body of work and the scope is not yet defined
- When deciding whether something is a single feature, a plan with sub-features, or a small
  improvement to an existing feature
- When the human wants to think through what to do next
- Before creating any Plan or Feature entities

---

## The Agent's Role in Planning

Planning is human-led. The agent's job is to:

- **Ask clarifying questions** to help the human articulate scope
- **Reflect back** what the agent understands the scope to be, so the human can correct it
- **Flag scope issues** — things that seem too large, too small, or overlapping with existing work
- **Not make product decisions** — priority, scope, and direction belong to the human

The agent does not recommend what to build. The agent helps the human articulate what they
have already decided they want to build.

---

## Scope Decisions

### Feature vs. Plan

A **feature** is a single coherent piece of user-facing behaviour that can be designed,
specified, and implemented independently. It should be possible to:

- Write one design document for it
- Write one specification for it
- Implement it in a single worktree without significant conflict with other active features

A **plan** is a coordinating entity for a body of work that is too large or structurally
interconnected to treat as a single feature. Use a plan when:

- The work comprises multiple independent features that could be designed and implemented
  separately
- There is a high-level design document that describes how the features fit together
- There is a meaningful milestone or release that the features collectively deliver

**Err towards fewer plans.** A single feature does not need a plan. A plan with only one
feature is usually just a feature. Plans exist to coordinate sets of related features toward
a shared goal — not to add process overhead to simple work.

### Sizing Signals

A scope is probably **one feature** if:
- It can be described in one sentence
- It would produce one design document
- It could be implemented in a focused sprint

A scope is probably **multiple features** (and needs a plan) if:
- It has clearly independent parts that could be designed or implemented separately
- Different people or agents could work on different parts in parallel
- The work would naturally produce multiple design documents

A scope is probably **too large to plan yet** if:
- It is not yet clear what the individual features are
- Fundamental questions about the direction are still open

In that case, the right next step is a high-level design document first, not a plan.

---

## Running a Planning Conversation

Good planning questions:

- "What problem are we solving for the user?"
- "What would a user be able to do that they can't do now?"
- "Are there parts of this that are clearly independent and could be done separately?"
- "Is there anything in scope that would block the rest if it changed?"
- "What's out of scope for now?"

Signs that planning is drifting into design:

- Discussion of *how* something will work technically
- Discussion of data models, API shapes, or system boundaries
- Discussion of which libraries or technologies to use

When this happens, note it and bring the conversation back to scope: *"That sounds like a
design question — should we capture that as something to resolve in the design document?"*

---

## What Planning Produces

A completed planning conversation produces:

1. **A scope statement** — one or two sentences describing what the work is and is not
2. **A structural decision** — one feature, or a plan with N named features
3. **Agreement to proceed** — the human signals readiness to move to design

The agent does not need to write a planning document unless the scope is large or complex
enough that it would be easy to lose track of. For most work, the scope statement lives in
the feature or plan summary field.

---

## When Planning Is Done

Planning is done when:

- The scope is agreed
- The structural decision (feature vs. plan) is made
- The human signals readiness to proceed to design

The agent then creates the appropriate entities (Plan and/or Feature) and the workflow moves
to the design stage. Entity creation requires the scope to be clear but does not require a
design document to exist yet.

---

## Related

- `kanbanzai-workflow` — stage gates and when each stage requires human approval
- `kanbanzai-design` — what happens once scope is agreed