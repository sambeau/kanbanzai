# Design Quality Principles

Six qualities to hold when proposing or reviewing a design. They are not a
checklist — they are a lens.

## Core Qualities

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

## The Relationship Between Them

Simplicity without completeness is a prototype. Completeness without
minimalism is bloat. Minimalism without composability is fragile. All four
together produce systems that are easy to understand, easy to trust, and easy
to extend.

## Supporting Qualities

- **Honest** — the design does not overclaim. It is truthful about what it
  does and does not do, and does not bury limitations or pretend trade-offs
  away. (Hiding complexity *for* the user — clean interfaces over exposed
  internals — is a design virtue, not a violation of this principle.)

- **Durable** — prefer designs that will not need revisiting in six months.

## Using This Lens

If a design feels wrong but the reason is hard to articulate, check it
against these six. Usually one is missing.

When evaluating an approach, consider both the ambitious and the expedient
version. Choose the ambitious version unless there is a genuine, articulated
reason it is wrong — not merely harder. Difficulty is not a valid objection
when the result is better.