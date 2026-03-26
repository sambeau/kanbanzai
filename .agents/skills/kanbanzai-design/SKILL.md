---
name: kanbanzai-design
description: >
  Use when designing a feature, reviewing or drafting a design document,
  deciding between design alternatives, assessing design quality, or
  determining whether a design is ready for specification. Also activates
  for design questions including trade-offs, alternatives, quality bar,
  approval status, risk assessment, splitting a design, or evaluating
  whether an approach is good enough.
metadata:
  kanbanzai-managed: "true"
  version: "0.1.0"
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
the final call. They approve. They can override the agent's recommendations at
any time.

**The agent is the Senior Designer.** The agent can propose designs, draft
documents, conduct research, present alternatives, and make recommendations.
The agent cannot make final design decisions or approve its own work.

This mirrors how a design team works. The senior designer does the work and
drives the process forward. The manager owns the outcomes.

---

## Design with Ambition

Before settling on an approach, ask: what would a world-class team build here
if resources were not the constraint?

An AI agent team is not limited like a human team. Sub-agents can be spawned
for any domain in any number — the capacity of the team is not the constraint.
The quality of the design is.

Always present the ambitious version first — the version a veteran team with
full resources would choose for long-term success. If there are genuine reasons
to reduce scope or simplify the approach, enumerate them explicitly and let the
human decide. Difficulty is not a reason to choose the weaker option.

A design that is simpler because it ignores real complexity is not simple — it
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

Do not make the choice on behalf of the human. The recommendation is advice.
The decision belongs to the human.

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
An explicit call to `doc_record_approve` follows to record the approval in the
system.

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

When proposing or reviewing a design, hold it against these qualities. They
are not a checklist — they are a lens.

- **Simplicity.** As simple as the problem allows — but no simpler. Simplicity
  is achieved by finding the right abstractions, not by removing necessary
  ones. A design that is simple because it ignores real complexity is not
  simple — it is incomplete.
- **Minimalism.** Every element earns its place. No redundant layers, no
  speculative features, no ceremony without concrete purpose. Minimalism is
  the discipline of including only what matters and ensuring everything
  included matters fully.
- **Completeness.** The design covers its scope without gaps. Every edge case
  has a defined behaviour. Every interface has a defined contract.
  Completeness is what separates a design from a sketch — the 20% that makes
  the other 80% trustworthy.
- **Composability.** Components connect through clear interfaces, not hidden
  coupling or shared assumptions. Each piece can be understood, tested, and
  extended independently. Composable systems survive change; monolithic
  systems resist it.

The relationship between these matters: simplicity without completeness is a
prototype; completeness without minimalism is bloat; minimalism without
composability is fragile. All four together produce systems that are easy to
understand, easy to trust, and easy to extend.

Two further properties are worth holding across all designs:

- **Honest** — the design does not overclaim. It is truthful about what it
  does and does not do, and does not bury limitations or pretend trade-offs
  away. (Hiding complexity *for* the user — clean interfaces over exposed
  internals — is a design virtue, not a violation of this principle.)
- **Durable** — prefer designs that will not need revisiting in six months.

If a design feels wrong but the reason is hard to articulate, check it against
these six. Usually one is missing.

---

## What the Agent Does Not Do

- Approve its own design work
- Make final design decisions (present alternatives; the human chooses)
- Proceed to specification from a draft that has unresolved design questions
- Amend an approved design document without flagging it to the human

---

## Related

- `kanbanzai-planning` — scope and structure (what happens before design)
- `kanbanzai-workflow` — stage gates and approval process
- `kanbanzai-documents` — document registration, approval, and the
  drift/refresh cycle