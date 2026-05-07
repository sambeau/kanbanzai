| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | architect                       |
| Status | approved |
| Plan   | P57-retro-pipeline-tightening |

# Design: Retrospective Pipeline Tightening

## Overview

Kanbanzai's retrospective system collects process observations from agents during task completion, synthesises them into ranked themes, and generates markdown reports. But the pipeline stops there: there is no path from a synthesised theme to an executable fix. The `retro_fix` fast-track tier exists in configuration with sensible gate modes (`design: human, spec: auto, dev_plan: auto, review: conditional`), but there is no entity creation path, no document template, no close-out verification, and no signal-to-fix traceability. This plan bridges that gap — creating a full retrospective fix pipeline where ranked themes spawn fix features that flow through design, spec, dev-plan, implementation, review, and verification with minimal human intervention.

## Goals and Non-Goals

### Goals

- Create an entity creation path from synthesised retro themes to executable fix features with `tier: retro_fix`
- Define document templates for retro fix specs (auto-generated from theme data) and dev-plans
- Add close-out verification for retro fixes with an adapted Definition of Done (signal-addressed check)
- Add a stage binding profile and skill for the retro-fix workflow
- Fix the retro `report` tool's document type registration (`"report"` → `"retro"`)
- Link fix features back to source signals for traceability (humans can see what was identified and what was done)
- Support two modes: **human-gated** (default — human picks themes and approves design) and **full-auto** (human initiates the cycle, system handles everything through to done)
- In full-auto mode: auto-pick themes by severity threshold, auto-generate design from theme data, auto-approve all stage gates

### Non-Goals

- Changing how agents contribute retrospective signals via `finish` (this works)
- Changing the synthesis engine's clustering or ranking algorithm
- Creating a new entity type (`RetroFix`) — reuse `Feature` with `tier: retro_fix`
- Making full-auto the default — human-gated remains the default mode; auto is opt-in per cycle
- Changing the `orchestrate-development` or `orchestrate-review` skills
- Building a dedicated `retro-analyst` role (defer to existing roles for now)

## Problem and Motivation

### Problem

The retrospective pipeline is half-built. Agents faithfully record friction signals. The synthesis engine clusters them into ranked themes. A markdown report is generated and registered. But nothing happens next — themes sit in a report with no path to resolution.

The `retro_fix` tier exists in `DefaultFastTrackConfig` with appropriate gate modes and a 3-cycle review cap. The conditional review gate correctly distinguishes doc-only changes (skip review) from implementation changes (full review panel). The tier inference rule tags features with `"retro"` → `retro_fix`. But no tool, skill, or procedure connects a synthesis theme to a feature entity.

The result: retrospective work requires manual orchestration. A human reads the synthesis report, manually creates features, manually writes specs, and manually routes them through the lifecycle. This is exactly the kind of toil the fast-track system was designed to eliminate.

### Evidence

1. **P40 and P50 retro batches:** Both required extensive manual setup — creating plan entities, manually writing design documents that restated synthesis themes, manually decomposing into features and tasks. The synthesis report was the starting point for a fully manual process.

2. **`retro_fix` tier is orphaned:** The tier configuration, conditional review logic, and tier inference rules all exist and are tested. But no creation path feeds entities into this tier. The `retro` tool only has `synthesise` and `report` actions — no `create_fix` or `spawn` action.

3. **Document type mismatch:** `retro(action: "report")` registers the generated report as `type: "report"` in the document store. The system defines `DocumentTypeRetro = "retro"` and the `work/retro/` directory has `default_type: retrospective`, but the tool hardcodes `"report"`. Retro reports are mixed into the general report pool.

4. **No close-out verification:** Features and bugs have Definition of Done checklists (10-item and 8-item respectively) enforced by a clean-context verifier sub-agent at the `verifying` stage. Retro fixes have no equivalent — no DoD, no verifier dispatch, no signal-addressed check.

### Why This Matters Now

P55 and P56 established the pattern: clean-context sub-agent verification, Definition of Done checklists, lifecycle gate enforcement, and role/tool hygiene. The retrospective pipeline is the last major workflow that hasn't received this treatment. The building blocks exist (tier config, conditional review, verifier role, synthesis engine) but they aren't connected.

### Motivation

The original vision for retrospective work was: AI agents record issues they encounter, and periodically, on human request, the system fixes its own biggest bugbears. The current system captures the issues but can't act on them. Closing this loop turns retrospective from a diagnostic tool into a self-improvement mechanism.

## Related Work

### Prior Research

- **P5-workflow-retrospective** — Original workflow retrospective design. Defined signal categories, severity levels, the `finish(retrospective: [...])` contribution mechanism, and the synthesis-report pipeline. Established the core vocabulary (signals, themes, categories, severity weights) that this plan builds on.

### Prior Designs

- **P55-design-orchestrator-context-hygiene.md** — Established the verifier sub-agent pattern: a clean-context sub-agent dispatched at close-out to run a binary checklist independently. Defined the 10-item Definition of Done. Component 7 (Close-Out Verification Sub-Agent) is the direct template for retro fix verification.

- **P56-design-bug-lifecycle-hardening.md** — Hardened the bug workflow with mandatory review gates, lifecycle enforcement, auto-generated specs from bug reports, worktree isolation, and an 8-item DoD adapted for bugs. The bug pipeline is the closest structural analog to what retro fixes need.

- **P52-design-fast-track-orchestration.md** — Defined the fast-track behavioural profile and the tier matrix. The `retro_fix` tier entry in `DefaultFastTrackConfig` originated here.

### Key Reference: Bug Pipeline Structure

The bug pipeline (P56) provides the template:

| Bug Element | Retro Fix Equivalent |
|---|---|
| `BUG-...` entity with `observed`/`expected` | `FEAT-...` entity with `tier: retro_fix` and retro signal IDs |
| Auto-generated spec from observed/expected | Auto-generated spec from theme observation/suggestion |
| Human gate at design + spec | Human gate at design only (spec is auto) |
| 8-item DoD with verifier dispatch | 10-item DoD (add signal-addressed) with verifier dispatch |
| `bug_fix` tier: review:auto, max_cycles:2 | `retro_fix` tier: review:conditional, max_cycles:3 |

### Constraining Decisions

- **Entity type decision:** This plan reuses `Feature` with `tier: retro_fix` rather than creating a new `RetroFix` entity type. New entity types require lifecycle definitions, transition validators, MCP tool wiring, CLI commands, and state store schema changes. Features already have all of this. The `tier` field and `tags: ["retro"]` provide sufficient typing.

- **Design-only human gate:** The `retro_fix` tier sets `design: human, spec: auto`. This is intentional — the human's judgment matters for approach selection (design), but the spec is a mechanical extraction of the theme's observation and suggestion. This is a lighter human touch than bugs (which gate at both design and spec).

## Design

### Design Principle

**Retro fixes are theme-driven features.** The synthesis engine identifies what to fix and suggests how. The human approves the approach at design time. Everything else — spec generation, dev-plan creation, task decomposition, implementation, review, and verification — follows the established feature lifecycle with retro-specific adaptations.

### Component 1: Theme-to-Entity Creation Path

Add a `create_fix` action to the `retro` tool. This is the single entry point for spawning retro fix features from synthesis themes, with two operating modes.

**Input:**
- `scope` — plan ID, feature ID, or `"project"` (must match a prior synthesis)
- `mode` — `"human-gated"` (default) or `"auto"`
- `theme_index` — the rank number of the theme from the synthesis result (human-gated mode only)
- `theme_count` — number of top themes to fix (auto mode only; default: top theme only)
- `severity_threshold` — minimum severity score for auto-picking (auto mode only; default: no threshold, use theme_count)
- `name` — human-readable feature name (optional; defaults to the theme title)
- `parent_plan` — plan ID to group fixes under (optional; auto-created if omitted, see below)

**Human-gated mode behaviour:**
1. Run `synthesise` with the given scope to get current themes
2. Select the theme at `theme_index`
3. Create a `Feature` entity with `tier: "retro_fix"`, tags linking to source signals, and an auto-generated summary
4. Register an auto-generated specification document from the theme data (Component 2)
5. Return the feature ID, spec document ID, and a summary. The human must then approve the design at the `designing` stage gate.

**Auto mode behaviour:**
1. Run `synthesise` with the given scope to get current themes
2. Select themes by `theme_count` (top N by severity score) and/or `severity_threshold` (all themes with severity score ≥ threshold)
3. For each selected theme:
   - Auto-create a parent plan if `parent_plan` is omitted (see below)
   - Create a `Feature` entity with `tier: "retro_fix"` and signal tags
   - Auto-generate and auto-approve a design document (Component 1a)
   - Auto-generate and auto-approve a specification document (Component 2)
   - Auto-generate and auto-approve a dev-plan document
   - Auto-decompose into tasks via `decompose(action: "propose", feature_id: "...")`
   - The feature then flows through the standard lifecycle with all gates set to `auto`
4. Return a batch summary: plan ID, list of feature IDs with their spec and dev-plan document IDs

The action is idempotent: if a feature already exists for the same theme (matched by signal ID set), skip creation rather than duplicating.

**Parent plan auto-creation:**
When `parent_plan` is omitted, the system auto-creates a plan named `Pnn-retro-fixes-{month-year}` (e.g. `P58-retro-fixes-june-2026`). This follows the convention established by P40 and P50: one plan per retro cycle, containing multiple fix features. The synthesis report generated at the start of the cycle serves as the plan's context. If a plan with the same month-year already exists, fixes are added to it rather than creating a duplicate.

When `parent_plan` is explicitly provided, fixes are grouped under that plan — useful when retro fixes are scoped to a specific parent plan's signals.

### Component 1a: Auto-Generated Design Document (Auto Mode)

In auto mode, the system auto-generates and auto-approves a design document for each retro fix feature. This satisfies the `designing` stage gate prerequisite (the `specifying` stage requires an approved design).

**Template mapping:**

| Design Section | Source |
|---|---|
| Overview | Theme title + representative observation |
| Goals and Non-Goals | Goal: implement the theme's top suggestion. Non-goal: any change not implied by the suggestion |
| Design | The suggestion verbatim, with implementation notes inferred from the signal category (e.g. tool-gap → new MCP tool, spec-ambiguity → spec template update) |
| Alternatives Considered | "Do nothing — accept the friction as a known limitation" (always included as the default alternative) |
| Dependencies | Inferred from affected files mentioned in signal content, if any |

The design document is registered as `type: "design"` with `owner: "<FEATURE-ID>"` and auto-approved immediately. It is deliberately minimal — it exists to satisfy the stage gate, not to provide creative design rationale. In auto mode, the theme's suggestion *is* the design.

### Component 2: Auto-Generated Retro Fix Specification

When a retro fix feature is created from a theme, auto-generate a specification document. The spec follows the standard template (Overview, Scope, Functional Requirements, Non-Functional Requirements, Acceptance Criteria) but derives content from the theme data.

**Template mapping:**

| Spec Section | Source |
|---|---|
| Overview | Theme title + representative observation (what agents experienced) |
| Scope | Signal category + affected components inferred from signal content |
| Functional Requirements | Top suggestion converted to requirement language ("The system shall...") |
| Non-Functional Requirements | Empty by default; populated if signals mention performance/reliability |
| Acceptance Criteria | Derived from the top suggestion: "Given [observed behaviour], when [fix is applied], then [expected behaviour]" |

The spec is registered as `type: "specification"` with `owner: "<FEATURE-ID>"` and auto-approved (the `spec: auto` gate means no human review is needed).

### Component 3: Retro Fix Definition of Done

Adapt the verifier's 10-item DoD for retro fixes. Items 1–9 are identical to the feature DoD (tasks terminal, changes committed, temp files removed, tests pass, code reviewed, lifecycle advanced, merge ancestry, branch deleted, worktrees removed). Item 10 is retro-specific:

**Item 10: Signals Addressed**

- **Condition:** The source retro signals that spawned this fix are linked to the fix entity and marked as addressed. No signal that generated this fix remains unresolved.
- **Verification action:** Call `entity(action: "get", id: "<FEATURE-ID>")` to retrieve the feature's tags (which contain signal IDs). For each signal ID, call `knowledge(action: "get", id: "<SIGNAL-ID>")` and confirm the entry is in `confirmed`, `retired`, or has been updated with a reference to the fix feature.
- **Pass criterion:** All source signals are confirmed, retired, or linked to this fix feature. No source signal remains in `contributed` status without a fix reference.
- **Fail criterion:** Any source signal is still in `contributed` status with no fix reference, or a signal ID in the feature's tags does not correspond to an existing knowledge entry.

### Component 4: Stage Binding Profile — `retro-fixing`

Add tier profiles to `stage-bindings.yaml` for retro fix work, covering both operating modes. Rather than creating new lifecycle stages, these profile the existing feature lifecycle with retro-specific gates and templates:

```yaml
retro-fixing:
  description: "Implementing a fix for a retrospective theme"
  profile: true
  tier: retro_fix
  modes:
    human-gated:
      design_gate: human
      spec_gate: auto
      dev_plan_gate: auto
      review_gate: conditional
      max_review_cycles: 3
      notes: >
        Human picks which themes to fix and approves the design approach.
        Spec is auto-generated from theme data. Review is conditional:
        doc-only changes skip review, implementation changes trigger full
        panel.
    auto:
      design_gate: auto
      spec_gate: auto
      dev_plan_gate: auto
      review_gate: conditional
      max_review_cycles: 3
      notes: >
        Human initiates the cycle and steps back. System auto-picks themes
        by severity, auto-generates and auto-approves design and spec, then
        decomposes, implements, reviews, and verifies without checkpoints.
        Human reviews results after completion.
  verifying:
    roles: [verifier]
    skills: [verify-closeout]
    dod_variant: retro-fix
```

### Component 5: Document Registration Fix

Fix `RetroService.Report()` in `internal/service/retro_synthesis.go` to register retro reports with the correct document type:

```go
// Current (wrong):
Type: "report",

// Fixed:
Type: "retro",
```

This ensures retro reports appear in `doc(action: "list", type: "retro")` queries and are not mixed into the general report pool.

### Component 6: Document Trail for Human Visibility

The document trail connects signal to fix to verification:

1. **Synthesis report** (`work/retro/retro-{scope}.md`) — lists all themes with signal IDs. After fixes are created, links to the fix features.
2. **Per-fix spec** (`work/{plan}/...-spec-{slug}.md`) — auto-generated specification derived from the theme.
3. **Per-fix dev-plan** (`work/{plan}/...-dev-plan-{slug}.md`) — standard dev-plan.
4. **Per-fix review report** (`work/review/review-{feature-id}.md`) — generated by the review panel.
5. **Close-out verification report** — produced by the verifier sub-agent, showing each DoD item's pass/fail status with evidence.
6. **Updated synthesis report** — the `retro(action: "report")` output should include a "Fixes Created" section linking to spawned features.

A human manager reads the synthesis report to see what was identified, follows links to individual fix features to see what was done, and checks verification reports for evidence of completion.

### Component 7: Skill — `implement-retro-fix`

Create a lightweight skill that codifies the retro fix workflow. It reuses existing skills for most stages but provides retro-specific context:

- **designing:** Standard `write-design` skill. The design document addresses the approach to fixing the theme. Human gate applies.
- **specifying:** Auto-generated spec (Component 2). No human gate — the spec is a mechanical extraction from the theme.
- **dev-planning:** Standard `decompose-feature` skill. Tasks are typically coarser than feature work (retro fixes are often systemic changes).
- **developing:** Standard `orchestrate-development` skill with `implement-task` sub-agents.
- **reviewing:** Standard `orchestrate-review` skill. The conditional gate applies: doc-only changes skip review.
- **verifying:** `verify-closeout` skill with the retro-adapted DoD (Component 3).

### What This Design Does NOT Do

- **Auto-picking themes.** The human decides which themes to fix and when. Frequency-threshold auto-picking is a future design.
- **Creating a `RetroFix` entity type.** Features with `tier: retro_fix` reuse all existing infrastructure.
- **Changing the signal contribution mechanism.** `finish(retrospective: [...])` is unchanged.
- **Creating a `retro-analyst` role.** Existing roles (architect for design, orchestrator for development, verifier for close-out) are sufficient.
- **Auto-closing the loop.** After a fix is verified, the source signals must be manually confirmed or retired. Full automation of signal lifecycle is deferred.
- **Cross-plan retro fix batching.** This design handles single-theme → single-feature creation. Batching multiple themes into a single plan is manual.

## Definition of Done

A single Definition of Done applies to all components. All seven components must be complete and verified:

| # | Component | Verification |
|---|---|---|
| 1 | Theme-to-entity creation path | `retro(action: "create_fix", scope: "...", theme_index: 1)` creates a `FEAT-...` with `tier: retro_fix`, correct tags, and an auto-generated spec |
| 2 | Auto-generated retro fix spec | Created spec has all required sections, derives content from theme data, is auto-approved |
| 3 | Retro fix DoD (10-item with signal-addressed) | Verifier sub-agent runs the adapted checklist independently; item 10 verifies signal resolution |
| 4 | Stage binding profile | `stage-bindings.yaml` documents the `retro-fixing` profile with correct gates |
| 5 | Document registration fix | `retro(action: "report")` registers as `type: "retro"`, queryable via `doc(action: "list", type: "retro")` |
| 6 | Document trail | Synthesis report links to spawned fix features; per-fix spec, dev-plan, review, and verification reports exist |
| 7 | `implement-retro-fix` skill | Skill file exists in `.kbz/skills/implement-retro-fix/SKILL.md`, referenced in stage bindings |

### Rationale for a Single Definition

Unlike P55 (which had 7 independent components deployable in parallel), these components are tightly coupled: the creation path (1) depends on the spec template (2), which depends on the document type fix (5). The DoD (3) and skill (7) are standalone but small. A single integrated feature is more appropriate than multiple parallel features.

## Alternatives Considered

### Alternative A: New `RetroFix` Entity Type

Create a dedicated `RetroFix` entity type with its own lifecycle, state machine, transition validators, and MCP tool wiring.

**Advantages:** Strong typing. Clean separation from features. Retro-specific lifecycle states (e.g., `theme_identified → fix_designed → fix_verified`).

**Disadvantages:** High implementation cost. Duplicates worktree, task, document, and transition infrastructure that features already provide. The `tier` field already exists and provides sufficient behavioural differentiation.

**Decision:** Rejected. Reuse `Feature` with `tier: retro_fix`. If retro fixes develop genuinely distinct lifecycle needs, a dedicated entity type can be extracted later.



### Alternative C: Dedicated `retro-analyst` Role

Create a new role (`retro-analyst`) with its own vocabulary, anti-patterns, and tool constraints, dispatched as a clean-context sub-agent for retro report generation.

**Advantages:** Follows the P55 pattern of clean-context dispatch for specialised work. Prevents the plan-review agent's saturated context from producing shallow retro analysis.

**Disadvantages:** The retro report is synthesis of pre-structured data, not open-ended analysis. The synthesis engine does the clustering and ranking; the agent just formats. A dedicated role adds complexity for marginal benefit at this stage.

**Decision:** Deferred. If retro report quality degrades (vague observations, missed patterns), revisit. For now, the synthesis engine does the heavy lifting.

### Alternative D: Do Nothing — Manual Retro Pipeline

Leave the pipeline as-is: humans read synthesis reports, manually create features, manually write specs, and manually route through the lifecycle.

**Advantages:** No implementation cost. Full human control.

**Disadvantages:** The `retro_fix` tier config, conditional review logic, and tier inference rules are dead code. The system captures signals but can't act on them. The vision of autonomous self-improvement is unrealised.

**Decision:** Rejected. The building blocks already exist; the gap is small.

## Decisions

### Decision 1: Reuse Feature entity with tier=retro_fix

Rather than creating a new `RetroFix` entity type, retro fixes are `Feature` entities with `tier: retro_fix` and `tags: ["retro", ...signal_ids]`. This reuses all existing lifecycle, worktree, task, document, and transition infrastructure.

### Decision 2: Add create_fix action to retro tool

The `retro` MCP tool gains a `create_fix` action that bridges synthesis to entity creation. It takes a scope and theme index, runs synthesis, and creates a feature entity with auto-generated spec.

### Decision 3: Design-only human gate

The `retro_fix` tier keeps `design: human, spec: auto`. The human approves the approach (design) but the spec is auto-generated from theme data. This is less human involvement than bugs (design + spec) because the suggestion IS the acceptance criteria.

### Decision 4: Retro-adapted DoD adds signal-addressed check

The verifier's 10-item DoD for retro fixes replaces the generic item 10 ("Knowledge curated and entities closed") with a retro-specific check: "Signals addressed — source retro signals are confirmed, retired, or linked to this fix."

### Decision 5: Fix report document type now

The one-line `"report"` → `"retro"` fix in `RetroService.Report()` is included in this plan because it's trivial to implement and blocks proper document-type querying.

### Decision 6: Single integrated feature, not parallel components

Unlike P55 (7 parallel features), this plan's components are tightly coupled. A single feature with sequential tasks is more appropriate. The feature spec will decompose into ordered tasks.

### Decision 7: Dual-mode — human-gated default, auto opt-in

Rather than choosing between human-gated and fully-automated, the design provides both. `create_fix` defaults to `mode: "human-gated"` with the existing `retro_fix` tier gates (design: human, spec: auto, dev_plan: auto, review: conditional). `mode: "auto"` sets all gates to auto and the system handles theme picking, design generation, and document approval without checkpoints. This enables the original retrospective experiment — "can the system fix its own bugbears without me?" — while keeping human-gated as the safe default.

## Dependencies

- **P55-orchestrator-context-hygiene** (done) — Provides the verifier role, `verify-closeout` skill, and 10-item DoD pattern that Component 3 adapts.
- **P56-bug-lifecycle-hardening** (in progress) — Provides the bug spec auto-generation pattern that Component 2 adapts for retro fix specs. Component 4 (stage binding profile) follows P56's lifecycle enforcement pattern.
- **P5-workflow-retrospective** (done) — Provides the signal collection, synthesis engine, and `retro` tool that this plan extends.
- **P52-fast-track-orchestration** (done) — Provides the tier matrix and `retro_fix` tier configuration that this plan activates.

## Open Questions

1. **Theme index stability:** If new signals arrive between synthesis and `create_fix`, the theme ranking may shift. Should `create_fix` cache the synthesis result, or re-synthesise and warn if the theme at the given index has changed?

2. **Signal retirement semantics:** When a fix is verified, should source signals be automatically retired, or should a human explicitly confirm? Automatic retirement closes the loop but may hide signals that need further attention.

3. **Multi-theme fixes:** A single fix might address multiple themes (e.g., a tool-gap fix also resolves a tool-friction theme). Should `create_fix` support multiple theme indices?

4. **Review panel composition for retro fixes:** The conditional gate dispatches the full review panel (conformance, quality, security, testing) for implementation changes. Should retro fixes use a lighter panel (conformance only) given they're typically smaller in scope?

5. **Skill location:** Should `implement-retro-fix` live in `.kbz/skills/` (agent-facing execution skill) or `.agents/skills/` (kanbanzai-system usage skill)? P55's `verify-closeout` lives in `.kbz/skills/` as an agent-execution skill. The retro fix skill is similar — it's an execution skill, not a system-usage skill.

6. **Theme count vs. severity threshold interaction in auto mode:** If both `theme_count` and `severity_threshold` are specified, do they combine (themes that satisfy both) or does one take precedence? The current design says "and/or" — this needs precise semantics.
