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
  version: "0.3.0"
---

# SKILL: Kanbanzai Planning

## Purpose

Guide a planning conversation to produce clear scope: what to build, how big
it is, and how it fits into the existing project structure — without making
design or architecture decisions.

## Vocabulary

| Term | Definition |
|------|-----------|
| **scope agreement** | The outcome of a planning conversation — a shared understanding of what is and is not included in the work. |
| **feature decomposition** | Breaking a large scope into independent features that can each be designed, specified, and implemented separately. |
| **plan document** | A coordinating document that describes how multiple features fit together toward a shared goal. |
| **acceptance criteria** | Testable conditions that define when a requirement is satisfied. Every requirement must have at least one. |
| **effort estimate** | A story-point sizing on the Modified Fibonacci scale, set before work begins via `estimate(action: set)`. |
| **dependency graph** | The ordering relationships between tasks — which tasks must complete before others can start. |
| **implementation plan** | A dev-plan document that maps spec requirements to tasks with ordering, dependencies, and verification steps. |
| **vertical slice** | A feature subset that delivers end-to-end functionality through all layers, useful for validating architecture early. |

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
for any domain, in any number, at any point. The limit on what gets built is
the quality of the design — not the capacity of the team. Plan accordingly.

When scoping work:

- **Start with the best version.** What would a world-class engineering team
  at a well-resourced company design? That is the baseline.
- **Cut scope only for real reasons.** A real constraint is a technical
  limitation, fundamental complexity, or a genuine user need. Discomfort,
  impatience, or habit is not a reason to reduce scope.
- **Present the ambitious version first.** Let the human decide whether to
  cut scope. Scope reduction is a decision with explicit reasons, not a
  default.

If a scope seems large, that is a reason to design it carefully — not a
reason to simplify it prematurely.

---

## Scope Decisions

### Feature vs. Plan

A **feature** is a single coherent piece of user-facing behaviour that can be
designed, specified, and implemented independently. It should be possible to:

- Write one design document for it
- Write one specification for it
- Implement it in a single worktree without significant conflict

A **plan** is a coordinating entity for work that is too large or
interconnected to treat as a single feature. Use a plan when:

- The work comprises multiple independent features
- A high-level design document describes how the features fit together
- There is a meaningful milestone the features collectively deliver

**Err towards fewer plans.** A single feature does not need a plan. A plan
with only one feature is usually just a feature.

### Sizing Signals

A scope is probably **one feature** if:
- It can be described in one sentence
- It would produce one design document
- It could be implemented in a focused sprint

A scope is probably **multiple features** (and needs a plan) if:
- It has clearly independent parts
- Different agents could work on different parts in parallel
- The work would naturally produce multiple design documents

A scope is probably **too large to plan yet** if:
- It is not yet clear what the individual features are
- Fundamental questions about the direction are still open

In that case, the right next step is a high-level design document — not a plan.

---

## Running a Planning Conversation

Good planning questions:

- "What problem are we solving for the user?"
- "What would a user be able to do that they can't do now?"
- "Are there parts of this that are clearly independent?"
- "Is there anything in scope that would block the rest if it changed?"
- "What's out of scope for now?"

Signs that planning is drifting into design:

- Discussion of *how* something will work technically
- Discussion of data models, API shapes, or system boundaries
- Discussion of which libraries or technologies to use

When this happens, note it and redirect: *"That sounds like a design
question — should we capture that for the design document?"*

---

## What Planning Produces

A completed planning conversation produces:

1. **A scope statement** — one or two sentences describing what the work is
   and is not
2. **A structural decision** — one feature, or a plan with N named features
3. **Agreement to proceed** — the human signals readiness to move to design

The agent does not need to write a planning document unless the scope is
complex enough that it would be easy to lose track of. For most work, the
scope statement lives in the feature or plan summary field.

---

## When Planning Is Done

Planning is done when:

- The scope is agreed
- The structural decision (feature vs. plan) is made
- The human signals readiness to proceed to design

The agent then creates the appropriate entities (Plan and/or Feature) and the
workflow moves to the design stage. Entity creation requires clear scope but
does not require a design document to exist yet.

---

## Templates for Planning Outputs

When planning produces decisions to create specifications or implementation plans:
- **Specifications:** `work/templates/specification-prompt-template.md`
- **Implementation plans:** `work/templates/implementation-plan-prompt-template.md`

---

## Anti-Patterns

### Scope Creep in Planning

- **Detect:** Plan includes features not discussed in the scope agreement.
- **BECAUSE:** Undiscussed scope has not been evaluated for cost, risk, or
  priority. Including it silently bypasses the human's ability to make
  informed trade-offs, and the unvetted work often conflicts with or
  duplicates existing features.
- **Resolve:** Limit the plan to agreed scope. Flag additional scope for
  separate discussion with the human.

### Monolithic Feature

- **Detect:** A single feature covers multiple independent deliverables.
- **BECAUSE:** Large features resist parallel implementation and make progress
  tracking meaningless — the feature is either 0% or 100% done. Review
  becomes unwieldy, and a defect in one part blocks delivery of all parts.
- **Resolve:** Split into independent features that can be designed and
  implemented separately. Each feature should have its own spec and worktree.

### Missing Acceptance Criteria

- **Detect:** Spec has features or requirements without testable acceptance
  criteria.
- **BECAUSE:** Without criteria, "done" is undefined. Implementers make
  implicit assumptions about behaviour, reviewers cannot verify delivery,
  and disagreements surface late — during implementation or after merge.
- **Resolve:** Every requirement must have at least one testable criterion.
  If a criterion cannot be written, the requirement is not yet understood.

### Premature Simplification

- **Detect:** Scope is reduced before the ambitious version has been
  evaluated — e.g. "let's just do the simple version for now."
- **BECAUSE:** The simple version often creates design debt that the better
  version would not. Simplifying before evaluation forecloses options
  without understanding the cost of the shortcut.
- **Resolve:** Present the ambitious version first. Cut scope only for real
  constraints, with explicit reasons documented.

### Deferred Design

- **Detect:** Key structural questions are postponed — e.g. "we can figure
  that out later."
- **BECAUSE:** Figuring it out later costs more when the foundation is
  already locked in. Early structural decisions constrain all downstream
  work; deferring them risks building on a weak foundation.
- **Resolve:** Resolve structural questions during planning or flag them as
  mandatory design-phase inputs.

---

## Evaluation Criteria

| # | Question | Weight |
|---|----------|--------|
| 1 | Does the plan document contain a dependency graph showing task ordering? | required |
| 2 | Does every feature in the plan have a linked specification or clear acceptance criteria? | required |
| 3 | Are features scoped to be independently implementable? | high |
| 4 | Is the task breakdown between 8 and 15 tasks per feature? | high |

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
> **Structure:** Single feature — well-defined, can be specified and
> implemented as one unit.
> **Agreement:** Human confirmed scope. Proceed to design.

---

## Questions This Skill Answers

- How do I run a planning conversation?
- How do I break a feature into tasks?
- What makes a good acceptance criterion?
- When should a feature be split into multiple features?
- How do I estimate task effort?
- What goes in a plan document vs. a spec document?
- How do I handle dependencies between tasks?

---

## Related

- `kanbanzai-workflow` — stage gates and when each stage requires human
  approval
- `write-design` — what happens once scope is agreed