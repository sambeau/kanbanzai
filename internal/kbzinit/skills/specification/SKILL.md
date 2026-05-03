---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: kanbanzai-specification
description: >
  Use this skill whenever you are writing, reviewing, or refining a specification
  document, drafting acceptance criteria, checking whether a specification is ready
  for approval, or determining what the specification stage requires. Also activates
  when asked about the difference between a design and a spec, when acceptance
  criteria seem ambiguous or untestable, or when the spec and design appear to
  conflict. Use even when the spec seems straightforward — weak acceptance criteria
  propagate silently into bad implementations.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill guides the specification process from an approved design document to an
approved specification, ready for dev plan and task decomposition. You are the
Specification Writer; the human is the Product Owner.

A specification is a **formal distillation of the approved design** — terse, testable
acceptance criteria that can be used to plan implementation, write tests, and verify the
result. It is not an extension of the design, not an implementation plan, and must not
contain code.

The three document stages each serve a distinct purpose:

| Document | Character | Contains |
|---|---|---|
| **Design** | Discursive prose | What to build and why; alternatives, decisions, rationale |
| **Specification** | Terse and formal | Verifiable acceptance criteria distilled from the design |
| **Dev plan** | Operational | Task breakdown, engineering decisions, code sketches |

If you find yourself writing code, implementation steps, or engineering notes in a
specification, stop — those belong in the dev plan.

## Roles

**You (Specification Writer):** Draft the specification, propose acceptance criteria,
identify edge cases, flag ambiguities, and keep the spec aligned with the approved
design. Drive the work forward. You do not own what "done" means — that belongs to
the human.

**Human (Product Owner):** Own the acceptance criteria, decide what "done" means,
and approve the specification. The spec is done when they say it is done.

## When to Use

- After a design document has been approved
- When drafting or reviewing acceptance criteria
- When checking whether a specification is ready for approval
- When the design leaves implementation questions the spec must resolve

## The Specification Process

Specification is iterative. A typical flow:

1. Design is approved — the stage gate to specification is open
2. Agent drafts the specification from the approved design
3. Human reviews and gives feedback
4. Agent revises; steps 2–3 repeat until no open questions remain
5. Human approves
6. Workflow moves to dev plan and task decomposition

Do not begin specification until the design is approved. If no approved design exists,
apply the emergency brake — see `kanbanzai-workflow`.

## What a Good Specification Contains

At minimum:

1. **Scope** — what is being specified and what is explicitly excluded
2. **Acceptance criteria** — each independently testable and verifiable (see below)
3. **Constraints** — performance, compatibility, security, or operational requirements
4. **Out-of-scope** — explicit exclusions that prevent scope creep downstream

A good specification is **terse**. Each section exists to support a set of acceptance
criteria, not to explain or re-argue the design. If a passage could be removed without
losing a verifiable criterion, remove it.

### What a Spec Must Not Contain

- **Code** — no implementation examples, no pseudocode, no schema definitions
- **Implementation notes** — no "how to build it" guidance; that belongs in the dev plan
- **Design rationale** — no "we chose X because Y"; the design document owns that
- **Alternatives** — a spec reflects one chosen direction (same rule as the design)
- **Narrative padding** — no prose that does not directly support a testable criterion

## Acceptance Criteria Quality Bar

Every acceptance criterion must be:

- **Testable** — an unambiguous pass/fail determination is possible without
  interpretation
- **Independent** — verifiable without reference to other criteria
- **Specific** — names the exact behaviour, input, output, or state being verified

"It works correctly" is not an acceptance criterion. Neither is "it handles errors
gracefully" or "performance is acceptable." Rewrite vague criteria as concrete,
falsifiable statements before marking a spec ready for approval.

When you find a criterion that fails the quality bar, rewrite it. Do not accept
vague criteria and hope the implementing agent will interpret them generously.

## The Approved Specification Invariant

A specification is ready for approval when:

1. All acceptance criteria meet the quality bar above
2. No unresolved questions remain
3. Scope matches the approved design — no additions, no omissions
4. One direction is chosen throughout — no alternatives or open forks

When the human approves verbally, record it immediately:

    doc(action="approve", id="DOC-...")

## Relationship to Design and Dev Plan

The pipeline is: **design → specification → dev plan**.

- The **design** says *what to build and why* — prose with decisions and rationale.
- The **specification** says *what to verify* — a terse formal distillation of the design.
- The **dev plan** says *how to build it* — tasks, engineering notes, code sketches.

A specification is a distillation, not an extension. Every requirement in the spec must
trace back to the approved design. If you find yourself writing acceptance criteria for
something not in the design, stop — that is scope addition and requires updating the
design first.

If the spec contradicts the design, the design governs. Surface the conflict to the
human — do not silently resolve it in either direction.

If the spec requires a decision the design left open, stop and ask. Adding scope
not in the approved design is a stage-gate violation — see `kanbanzai-workflow`
emergency brake.

---

## Gotchas

**Forgetting to register the document.** After creating the specification file, call
`doc` with action: `register` immediately. Unregistered specs are invisible to the
approval workflow, health checks, and task decomposition.

**Spec full of code or implementation detail.** If the specification contains code
snippets, schema definitions, pseudocode, or "how to build it" notes, it has absorbed
content that belongs in the dev plan. Move that content to the dev plan. A spec that
reads like an implementation guide will cause agents to skip the dev plan stage or
produce a dev plan that merely repeats the spec.

**Approving with ambiguous criteria.** The implementing agent will interpret ambiguous
criteria literally and differently than you intended. Fix criteria before approval,
not after rework.

**Adding scope not in the design.** If the spec starts to contain requirements that
were not in the approved design, stop. This is a scope addition — it may require
updating the design first. Apply the emergency brake.

**Editing an approved specification.** The same rules as an approved design — create
a new document and supersede the old one. Do not silently edit an approved spec.
The content hash is recorded at approval time; silent edits break referential
integrity.

    doc(action="supersede", id="old-DOC-...", superseded_by="new-DOC-...")

**Verbal approval not recorded.** Call `doc` with action: `approve` immediately
when the human approves in conversation. Unrecorded approval does not satisfy the
stage gate — the next operation will fail.

---

## Related

- `kanbanzai-design` — what happens before specification (the approved design is the
  input to this stage)
- `kanbanzai-workflow` — stage gates and the specification → dev plan progression
- `kanbanzai-documents` — document registration, drift, approval, supersession
- `kanbanzai-agents` — task dispatch and the dev plan stage that follows approval