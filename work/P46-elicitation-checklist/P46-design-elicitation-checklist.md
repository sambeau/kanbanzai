# Design: Elicitation Checklist

**Plan ID:** P46-elicitation-checklist  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §6.3

## Overview

Adopt OmO's Prometheus-style systematic checklist into Kanbanzai's spec-author skill. The checklist ensures nothing is left implicit before a specification is written — core objective, scope boundaries, ambiguities, technical approach, and test strategy are explicitly addressed.

Kanbanzai already supports collaborative requirement discovery through natural discussion. That *is* requirement elicitation, and for an engaged human it works well. What Prometheus adds is: (a) a systematic checklist that prevents implicit assumptions from surviving into the spec, and (b) codebase-aware questioning — Prometheus explores the codebase *before* asking, so prompts are grounded in real patterns. The checklist pattern (without the full interview mode) is the valuable, low-effort part to adopt now.

This is the smallest enhancement in the P41 plan — a prompt addition to an existing skill. No new roles, no new entities, no architecture changes.

## Goals and Non-Goals

**Goals:**
- Add a structured pre-spec checklist to the `write-spec` skill
- Ensure every specification addresses: core objective, scope boundaries, ambiguities, approach, test strategy
- Prevent specs from being written against implicit, unstated assumptions
- Zero new infrastructure — just a skill update

**Non-Goals:**
- Not creating a new interview-mode role — that's a larger feature (see §6.3 discussion)
- Not replacing the design document — the design still owns architectural decisions
- Not adding codebase exploration agents (Explore/Librarian) — those require model routing (P44)
- Not changing the spec-author's identity or anti-patterns

## Design

### Checklist Addition to `write-spec` Skill

Add a new step before "Step 1: Read the Design" in the `write-spec` procedure:

```
### Step 0: Elicitation Checklist

Before writing any specification content, verify that the following
questions have explicit answers. If any answer is implicit or assumed,
STOP and ask the human before proceeding.

1. **Core objective:** What is the single most important thing this feature
   must accomplish? State it in one sentence.

2. **Scope boundaries:** What is explicitly IN scope? What is explicitly
   OUT of scope? If either list is empty, the scope is not defined.

3. **Ambiguities:** What aspects of the design are open to interpretation?
   List every ambiguity and the chosen resolution. If an ambiguity has no
   resolution, flag it — do not assume.

4. **Technical approach:** What is the chosen approach? What alternatives
   were rejected and why? (Cite the design document — this is a
   cross-reference check, not a design decision.)

5. **Test strategy:** How will correctness be verified? What kinds of tests
   (unit, integration, e2e) are expected? What edge cases must be covered?

6. **Constraints:** What constraints does the design impose? (Performance
   budgets, backward compatibility, API contracts, data migration requirements.)

7. **Dependencies:** What other features, packages, or external systems
   does this feature depend on? What depends on this feature?
```

### When the Checklist Fires

The checklist runs once, before any specification content is written. It does not run for spec revisions unless the scope has changed. The spec-author completes it, notes any flags, and presents unresolved items to the human before proceeding to Step 1.

### Integration with Existing Procedure

The existing `write-spec` procedure already has a Cross-Reference Check and Step 1 (Read the Design). The checklist sits before both:

```
Cross-Reference Check → Elicitation Checklist → Step 1: Read the Design → ...
```

The Cross-Reference Check stays first because it validates that the design has a substantive Related Work section. The checklist comes next because it validates that the spec-author understands what's being asked before reading the design in detail.

### Relationship to Design

The checklist does not replace or weaken the design gate. It validates that the spec-author has a clear understanding of the design's intent. Ambiguities discovered by the checklist should be resolved in the design document, not in the spec. The checklist is a forcing function for design clarity, not a workaround for design gaps.

## Alternatives Considered

### Full interview mode (Prometheus-style)

Create a new `spec-interviewer` role that conducts an interactive interview with the human, exploring the codebase and asking clarifying questions before specification begins.

**Defer:** This is a larger feature that benefits from codebase exploration (Explore/Librarian agents) and model routing (P44) for the interview agent. The checklist pattern delivers most of the value (preventing implicit assumptions) with none of the infrastructure cost. The interview mode can be added later if the checklist proves insufficient.

### No change (status quo)

The spec-author reads the design and writes the spec. If the design is ambiguous, the spec-author flags it.

**Reject:** The research shows that implicit assumptions are the primary source of spec-specification gaps. A structured checklist catches them before they become embedded in the spec. The cost of adding the checklist is near-zero — it's a prompt addition to an existing skill.

### Checklist as a separate skill

Create a new `elicit-requirements` skill that runs before `write-spec`.

**Reject:** The checklist is tightly coupled to spec writing — it validates readiness to write, not a separate activity. Embedding it in `write-spec` keeps the skill count down and ensures it's always run in the right context.

## Dependencies

- None. This is a modification to `.kbz/skills/write-spec/SKILL.md`.
- No new roles, skills, entities, or MCP tools.
- No dependency on P42, P43, P44, or P45.
- The checklist doesn't require codebase exploration — it works with the design document alone.

## Open Questions

1. **Should the checklist produce a written artifact?** Currently it's procedural — the spec-author completes it mentally and flags issues. A written checklist output (registered as a document) would create an audit trail but adds ceremony. Start without; add if gaps are still surfacing in review.
2. **Should the checklist apply to design documents too?** The design stage has its own quality criteria. The checklist is spec-specific — it validates readiness to specify, not readiness to design. But if design ambiguity is a recurring problem, a design-level checklist could be added separately.
3. **How does this interact with fast-track (P43)?** The spec-validator (P43) checks structural completeness and traceability. The elicitation checklist checks *intent clarity* — a different concern. They're complementary: the checklist prevents the spec-author from writing against vague intent; the validator catches structural gaps in the written spec. Both run, at different points.
