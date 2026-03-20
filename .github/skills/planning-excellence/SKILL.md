---
name: planning-excellence
description: >
  This skill should be activated when planning, designing, scoping, or making
  architectural decisions. Use when the user asks about "planning", "design",
  "architecture", "how should we build", "what's the best approach",
  "phase planning", "roadmap", "scope", "decision making", "trade-offs",
  "should we simplify", "is this good enough", "what would experts do",
  "ambition", "quality bar", or any request to evaluate, compare, or choose
  between approaches.
---

# Planning Excellence

## The Ambition Principle

Plan and design as if the team has world-class expertise, unlimited skilled
labor, and no artificial time pressure. This is not aspirational — it is the
operating assumption. The team has the resources to build the best version of
whatever is designed.

This principle does not mean ignoring constraints. It means that when a
constraint is accepted, it must be a *real* constraint (technical limitation,
fundamental complexity, user need) — not a *convenience* constraint (laziness,
impatience, fear of difficulty, habit).

## The Quality Test

At every decision point — architecture, scope, decomposition, tooling,
process — apply these questions in order:

1. **Is this the best we can do?** Not the fastest. Not the simplest. The best.
2. **Is there a better version we could work towards?** A design that is
   cleaner, more composable, more complete, more future-proof.
3. **Is this the plan a veteran team would choose?** Experienced engineers
   who have seen what works at scale and what creates regret.
4. **Are we cutting scope for the right reason?** Reducing scope to sharpen
   focus is disciplined. Reducing scope to avoid difficulty is a shortcut.
5. **Will this decision age well?** In six months, will this feel like the
   right call — or like a compromise that now needs rework?

If any answer raises doubt, explore the alternative before committing.

## Design Quality Heuristic

Good design is recognised by four qualities. Use these as a lens when
evaluating any architecture, interface, data model, or plan:

- **Simplicity.** The design should be as simple as the problem allows —
  but no simpler. Simplicity is achieved by finding the right abstractions,
  not by removing necessary ones. A design that is simple because it ignores
  real complexity is not simple — it is incomplete.
- **Minimalism.** Every element earns its place. No redundant layers, no
  speculative features, no ceremony that does not serve a concrete purpose.
  Minimalism is not austerity — it is the discipline of including only what
  matters and ensuring everything included matters fully.
- **Completeness.** The design covers its stated scope without gaps. Every
  edge case has a defined behavior. Every state has a defined transition.
  Every interface has a defined contract. Completeness is what separates a
  design from a sketch — the 20% that makes the other 80% trustworthy.
- **Composability.** Components connect through clear interfaces, not
  through shared assumptions or hidden coupling. Each piece can be understood,
  tested, replaced, and extended independently. Composable systems survive
  change; monolithic systems resist it.

The relationship between these four matters. Simplicity without completeness
is a prototype. Completeness without minimalism is bloat. Minimalism without
composability is fragile. All four together produce systems that are easy to
understand, easy to trust, and easy to extend.

When a design feels wrong but the reason is hard to articulate, check it
against these four qualities. Usually one is missing.

## Anti-Patterns to Reject

Recognise and refuse these patterns. They masquerade as pragmatism but
produce inferior outcomes:

- **Premature simplification.** "Let's just do the simple version for now" —
  when the simple version creates design debt that the better version would
  not. Simplicity is a goal; *simplistic* is a failure mode.
- **Deferred design.** "We can figure that out later" — when figuring it out
  now costs the same and avoids locking in a weak foundation.
- **Scope reduction as comfort.** "That's too ambitious" — when the ambition
  is achievable and the discomfort is just unfamiliarity. Scope should be
  bounded by *focus*, not by *timidity*.
- **Architecture by convenience.** Choosing a weaker structure because it is
  faster to implement, when the stronger structure would serve every future
  need better.
- **Cargo-cult pragmatism.** Invoking "pragmatism" or "YAGNI" to justify
  skipping work that is clearly needed. True pragmatism builds the right
  thing; false pragmatism avoids the hard thing.
- **Coverage retreat.** Reducing test coverage, skipping documentation, or
  deferring verification "to move faster." Velocity without quality is not
  velocity — it is future cost.
- **The 80% trap.** Delivering 80% of a feature and calling it done. The
  remaining 20% is usually the part that makes it *work well*.

## Planning Discipline

### Structure Plans for Quality, Not Speed

Organise work into phases with clear scope, explicit boundaries, and
acceptance criteria. Each phase should deliver a *complete* capability — not
a partially-working skeleton.

- **Phased delivery.** Break large efforts into phases. Each phase must
  stand on its own as a useful, tested, documented increment.
- **Scope guards.** Define what is *out* of scope as clearly as what is *in*
  scope. This prevents drift without requiring artificial narrowness.
- **Decision records.** Capture non-trivial decisions with rationale,
  alternatives considered, and consequences. Decisions are first-class
  artifacts, not afterthoughts.
- **Audit and remediate.** After implementing a phase, audit it against
  the specification. Fix gaps before moving forward. Do not accumulate
  unverified work.

### Document Authority Hierarchy

Maintain a clear hierarchy of document authority to prevent contradiction
and ambiguity:

1. **Specification** — the binding contract. What must be built.
2. **Design** — the consolidated vision. Why it should be built this way.
3. **Implementation plan** — the execution guide. How to build it.
4. **Decision log** — the record of choices. What was decided and why.

If documents conflict, higher-authority documents take precedence. If code
conflicts with specification, surface the conflict — do not resolve it
silently.

### The Bootstrap Principle

When building a system that will eventually manage its own processes, design
for the *mature* system, not the *current* system. Build what the tool *will
be*, not what is easy to build today. The early versions should be a true
subset of the eventual design — not a divergent prototype that must be
replaced.

## Applying This to Concrete Decisions

When evaluating an approach, frame the analysis as:

- **Option A** — the version that is easiest to implement.
- **Option B** — the version that a veteran team with full resources would
  choose if building for long-term success.
- **Choose Option B** unless there is a genuine, articulated reason why
  Option B is wrong (not just harder). If Option B is harder but better,
  the difficulty is not a valid objection.

When proposing a plan, present the ambitious version first. If there are
genuine reasons to reduce scope, enumerate them explicitly — and let the
human decide whether those reasons are sufficient.

## What This Skill Does NOT Cover

- **Tool-specific usage** (MCP servers, CLI commands, code search tools) —
  use the appropriate tool-specific skills for those.
- **Language-specific coding style** — refer to project conventions and
  language-specific guidance.
- **Project-specific scope and constraints** — refer to the project's own
  specification and planning documents (e.g., `AGENTS.md`, spec files).

This skill provides the *philosophy and decision framework* that sits
above all of those. It answers the question: "Given that we *could* do
either, which should we choose?"
