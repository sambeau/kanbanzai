---
name: kanbanzai-design
description: >
  Use when designing a feature, reviewing or drafting a design document,
  deciding between design alternatives, assessing design quality, or
  determining whether a design is ready for specification. Also activates
  for design questions including trade-offs, alternatives, quality bar,
  approval status, risk assessment, splitting a design, or evaluating
  whether an approach is good enough. Use even when the agent feels
  confident about the design — design review catches problems that
  confidence does not.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Design

## Purpose

Guide the design process from an agreed scope to an approved design document,
ready for specification.

## When to Use

- When a scope has been agreed in planning and design work begins
- When reviewing or updating a design document mid-process
- When assessing whether a design document is ready for approval
- When deciding whether a design needs to be split or a new document started

---

## Roles

**The human is the Design Manager.** They own the design decisions. They make
the final call. They approve.

**The agent is the Senior Designer.** The agent proposes designs, drafts
documents, conducts research, presents alternatives, and makes
recommendations. The agent cannot make final design decisions or approve its
own work.

For the general rules on what humans own vs. what agents own, see
`kanbanzai-workflow`.

---

## Design with Ambition

The same ambition principle from `kanbanzai-planning` applies here: always
present the ambitious version first — the version a veteran team with full
resources would choose for long-term success.

If there are genuine reasons to simplify, enumerate them explicitly and let
the human decide. Difficulty is not a reason to choose the weaker option. A
design that is simpler because it ignores real complexity is not simple — it
is incomplete.

---

## The Design Process

Design is iterative. There is no single right path. A typical flow:

1. Scope is agreed (planning is done)
2. Human and agent discuss the design space — brainstorm, alternatives,
   constraints
3. Agent drafts the design document (a complete draft, not just an outline)
4. Human reviews and gives feedback
5. Agent revises; steps 3–4 repeat until there are no open questions
6. Human approves
7. Workflow moves to specification

Design can be messy. Stages can be revisited. A design that seemed complete
can turn out to need revision. That is normal and not a failure.

---

## Draft Documents

During the design process, documents are in **draft** status. Draft documents:

- May contain multiple design alternatives with pros and cons
- May contain open questions that are not yet resolved
- May be incomplete or exploratory

The agent's job during the draft phase is to keep the document as an honest,
up-to-date reflection of where the design has reached. If a discussion leads
to a decision, the agent updates the document. The document is not a record
of the conversation — it is the current best understanding of the design.

---

## Producing a Draft

When asked to draft a design, produce a **complete document**, not an outline.
A draft with alternatives and open questions is more useful than a skeleton
waiting for the human to fill in.

If the scope is not yet clear enough to write a useful draft, say so and ask
the questions needed to clarify it. It is usually better not to start a draft
until at least the scope is decided.

---

## Alternatives and Recommendations

When the design has multiple valid approaches, present them as alternatives
with:

- A brief description of each option
- The trade-offs (pros and cons)
- An explicit recommendation from the agent, with reasoning

The recommendation is advice. The decision belongs to the human.

A design document in draft status may contain multiple alternatives. An
**approved** document must not — it must reflect a single chosen direction.

---

## Open Questions

Any question about the design that has not been resolved is an open question.
Open questions should be listed explicitly in the document.

A design cannot be approved until all open questions are resolved. If a human
signals approval but open questions remain, flag them before proceeding.

It is fine for a design document to leave implementation questions open —
questions about *how* something will be built are usually for the developers
(often AI agents) to resolve. But questions about *what* the design is must
be resolved before approval.

---

## The Approved Design Invariant

A design document is ready for approval when it contains:

1. **Scope** — what is being built and why
2. **The chosen design** — one direction, not alternatives
3. **Key decisions and rationale** — including "why not X" where relevant
4. **No unresolved design questions**

Approval can be signalled verbally: "Approved", "LGTM", "Let's move to spec".
Record it with `doc` action: `approve` immediately so system state matches
reality.

---

## Surfacing Risk

When the agent identifies a technical risk in a design:

- **Minor concern** — mention once; note it in the document if relevant
- **Significant risk** — raise it clearly; repeat if the human moves on
  without acknowledging it
- **Security or data-integrity risk** — do not proceed without explicit
  acknowledgment; repeat until acknowledged

If the human acknowledges a risk and decides to proceed anyway, accept the
decision and document the risk and the rationale in the design document. The
human is the manager.

---

## When a Design Needs to Split

If during design it becomes clear that the scope is larger than a single
feature — that it logically breaks into parts that could be designed and
implemented independently — flag it and step back to planning.

A plan with multiple features, each with their own design documents, is the
right structure for this. A high-level document describing how the features
fit together is appropriate and encouraged.

Signs a design should split:

- Different sections of the document feel like separate products
- Different parts could be implemented independently without blocking each
  other
- The specification would be unmanageably large if written for the whole
  thing at once

If scope grows after a design has already been approved, the right response is
a new design document, not amending the approved one. The approved document is
superseded, not revised.

---

## Design Quality

When proposing or reviewing a design, hold it against six qualities:
simplicity, minimalism, completeness, composability, honesty, and durability.
See [references/design-quality.md](references/design-quality.md) for full
definitions.

The relationship between the core four matters: simplicity without
completeness is a prototype; completeness without minimalism is bloat;
minimalism without composability is fragile. All four together produce
systems that are easy to understand, easy to trust, and easy to extend.

---

## Gotchas

- **Forgetting to register the document.** After creating or renaming a
  design document, call `doc` action: `register` immediately. Unregistered
  documents are invisible to the system. See `kanbanzai-documents`.
- **Approving with open questions.** If the human says "approved" but the
  document still contains unresolved design questions, flag them before
  calling `doc` action: `approve`. An incomplete approval creates problems
  downstream when the spec tries to reference undecided points.
- **Editing an approved document.** If the approved design needs changes,
  create a new document and supersede the old one. Do not silently edit an
  approved document — the approval is tied to the content hash and will
  become void.
- **Tool call failures.** If `doc` action: `approve` fails, it usually means
  the content hash has drifted (the file was edited since registration).
  Call `doc` action: `refresh` first, then re-approve.

---

## Related

- `kanbanzai-planning` — scope and structure (what happens before design)
- `kanbanzai-workflow` — stage gates and approval process
- `kanbanzai-documents` — document registration, approval, and the
  drift/refresh cycle