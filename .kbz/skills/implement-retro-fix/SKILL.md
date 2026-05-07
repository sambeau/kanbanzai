---
name: implement-retro-fix
description:
  expert: "Stage-specific procedure for implementing retrospective fix features:
    theme-driven design, auto-generated spec from signal data, signal-addressed
    verification, and conditional review gate awareness"
  natural: "Guides you through implementing a retro fix feature — what to do at
    each lifecycle stage, how to handle the dual-mode (human-gated vs. auto)
    context, and how to trace implementation back to source signals"
triggers:
  - implement a retro fix
  - execute retro fix work
  - build a retrospective fix feature
  - work on a retro fix
  - retro fix lifecycle guidance
roles: [implementer, implementer-go, architect, spec-author]
stage: developing
constraint_level: medium
---

## Vocabulary

- **retro fix feature** — a `Feature` entity with `tier: retro_fix` created from a synthesis
  theme via `retro(action: "create_fix")`. Tags contain `"retro"` plus source signal IDs.
- **theme** — a ranked cluster of retrospective signals produced by `retro(action: "synthesise")`.
  Each theme has a title, representative observation, top suggestion, severity score, and
  constituent signal IDs.
- **signal** — a retrospective observation recorded via `finish(retrospective: [...])` during
  task completion. Signals carry a category, observation, severity, and optional suggestion.
- **human-gated** — the default `create_fix` mode (`mode: "human-gated"`) where the human
  selects a theme by index, writes and approves the design, and gates the workflow at the
  designing stage. Spec generation, dev-plan creation, and decomposition are automatic.
- **auto mode** — the fully automated `create_fix` mode (`mode: "auto"`) where themes are
  selected by severity threshold, design and spec are auto-generated from theme data, all
  stage gates are auto-approved, and the feature advances to `developing` without human
  checkpoints.
- **signal-addressed** — the retro-adapted Definition of Done check (item 10) that verifies
  all source signals linked to a retro fix feature are in `confirmed`, `retired`, or otherwise
  addressed status — none remain in `contributed`.
- **conditional review** — the `retro_fix` tier's review gate mode: documentation-only changes
  (files under `work/`, `docs/`, `refs/`) skip the review panel with an annotation;
  implementation changes (Go code, tool logic) trigger a full specialist review panel.
- **source signals** — the `KE-...` signal IDs extracted from the synthesis theme and stored
  in the feature's `tags` field. These are the traceability link from the fix back to the
  observations that motivated it.
- **parent plan** — a plan entity that groups retro fix features. In auto mode, a plan
  named `Pnn-retro-fixes-{month-year}` is auto-created if not specified.
- **theme_index** — the 1-based rank of a theme in the synthesis result, used in
  human-gated mode to select which theme to fix.
- **severity_threshold** — the minimum severity score a theme must have to be selected in
  auto mode. Combined with `theme_count` via intersection logic.
- **retro-fix DoD** — the 10-item Definition of Done checklist adapted for retro fixes,
  where item 10 checks signal resolution instead of knowledge curation. Invoked via
  `dod_variant: retro-fix` on the verifier.
- **conformance review** — the review dimension that verifies the implemented fix actually
  addresses the source signals' observations and suggestions. Unique to retro fixes.

## Anti-Patterns

### Writing Retro Fix Without Synthesise

- **Detect:** Agent begins designing or implementing a retro fix without first calling
  `retro(action: "synthesise")` to get current themes.
- **BECAUSE:** Synthesis is the single source of truth for what needs fixing. Without it,
  the agent works from memory or ad-hoc observations, missing signals recorded by other
  agents and the severity-weighted clustering that prioritises impact. The result is a fix
  that solves the wrong problem or ignores higher-severity issues.
- **Resolve:** Call `retro(action: "synthesise", scope: "project")` (or appropriate scope)
  before any retro fix work. Work from the ranked theme list, not from memory.

### Skipping Signal Traceability

- **Detect:** A retro fix feature is created or implemented without source signal IDs in
  its `tags` field, or the implementation does not reference which signals it addresses.
- **BECAUSE:** Signal traceability is the bridge from observation to resolution. Without
  it, the verifier cannot check signal-addressed status at close-out, humans cannot verify
  that the fix actually resolved the reported problems, and the retrospective pipeline
  breaks — themes get fixed but signals never transition out of `contributed`.
- **Resolve:** Ensure the feature's `tags` field includes all source signal IDs from the
  theme. In the implementation, cite signal IDs when addressing specific observations.
  At close-out, verify every source signal is in a terminal state.

### Human-Gated Assumption in Auto Mode

- **Detect:** Agent working on an auto-mode retro fix feature pauses at the designing
  stage waiting for human design approval, or stops at the specifying stage for a human
  gate.
- **BECAUSE:** In auto mode, `create_fix` auto-generates and auto-approves the design,
  spec, and dev-plan documents. The feature advances to `developing` without human
  checkpoints. An agent that treats auto-mode features the same as human-gated features
  introduces phantom gates that break the fast-track pipeline.
- **Resolve:** Check the feature's creation context. If created in auto mode, the design
  and spec are already approved — proceed directly to implementation. Only human-gated
  features require human design approval at the designing stage.

### Ignoring Conditional Review Gate

- **Detect:** Agent implements Go code changes for a retro fix feature and proceeds to
  merge without running a specialist review panel, treating it like a documentation-only
  change.
- **BECAUSE:** The `retro_fix` tier's conditional review gate distinguishes
  documentation-only changes (skip review) from implementation changes (full review
  panel). Skipping review on implementation changes means code lands on main without
  conformance, quality, security, or testing scrutiny.
- **Resolve:** Before advancing past the reviewing stage, check whether the feature's
  file changes include implementation files (Go source outside `work/`, `docs/`,
  `refs/`). If yes, a full specialist review panel is required. If documentation-only,
  the review gate can be skipped with an explicit annotation.

### Implementing Without Reading the Theme

- **Detect:** Agent reads only the auto-generated spec or task list and begins coding
  without reading the synthesis theme's full observation and suggestion.
- **BECAUSE:** The auto-generated spec is a mechanical extraction — it captures the
  suggestion as a requirement but may lose nuance from the original observations.
  The theme's representative observation contains the actual pain point; the suggestion
  is one possible fix. Understanding both prevents implementing a technically correct
  solution that doesn't address the root cause.
- **Resolve:** Call `retro(action: "synthesise")` and read the full theme entry
  (observation + suggestion + constituent signals) before writing code. Cross-check
  implementation decisions against the original observation.

### Signal Retirement Neglect

- **Detect:** The retro fix is implemented and merged, but source signals remain in
  `contributed` status — they were never confirmed or retired.
- **BECAUSE:** The retro-adapted DoD's item 10 explicitly checks signal resolution.
  A fix that doesn't close the loop leaves stale signals that will re-appear in future
  syntheses, causing the same theme to be re-surfaced. The pipeline's self-improvement
  loop requires signals to reach terminal states.
- **Resolve:** After implementation and verification, for each source signal ID in the
  feature's tags, call `knowledge(action: "confirm", id: "KE-xxx")` if the fix addressed
  the observation, or `knowledge(action: "retire", id: "KE-xxx", reason: "...")` if the
  observation is no longer relevant. This must be done before the verifier runs the
  retro-fix DoD.

## Procedure

This procedure is organised by lifecycle stage. Retro fix features follow the standard
feature lifecycle (`designing → specifying → dev-planning → developing → reviewing →
merging → verifying → done`) with modifications at each stage for the retro fix context.

### Designing

**Human-gated mode:** The human writes and approves the design document. The design must:
1. Restate the theme's observation and suggestion in the Overview section.
2. Identify the approach — how the fix will address the observation.
3. List the source signal IDs that motivated the fix.
4. Follow the standard design document template (Overview, Goals and Non-Goals, Design,
   Alternatives Considered, Dependencies).

After the human approves the design, the feature advances to specifying. The spec is
auto-generated from the theme data by `create_fix`.

**Auto mode:** The design is auto-generated from the theme by `create_fix` and
auto-approved. The design document is derived from the theme title (Overview), observation
(Goals), and suggestion (Design approach). Do not re-create or modify the auto-generated
design — it is already approved. Proceed directly to implementation.

### Specifying

In both modes, the specification is auto-generated from the theme data. The spec contains:
- **Overview:** Derived from the theme title and representative observation.
- **Scope:** The fix's boundary, derived from the suggestion's affected area.
- **Functional Requirements:** The top suggestion converted to requirement language.
- **Non-Functional Requirements:** Standard constraints (no regression, tests required).
- **Acceptance Criteria:** The suggestion expressed as given/when/then assertions.

The spec is auto-approved. Do not manually rewrite the spec — it is derived from the
theme and approved as-is. If the auto-generated spec is missing or insufficient, report
the gap rather than silently rewriting it.

### Dev-Planning

In both modes, a dev-plan document is auto-generated and auto-approved, and tasks are
decomposed via `decompose(action: "propose")`. The dev-plan derives tasks from the spec's
acceptance criteria.

If the feature was created in auto mode, all of this is already done — no action needed.
If the feature was created in human-gated mode, the orchestrator handles dev-plan
creation and decomposition after the human approves the design.

### Developing

Follow the `implement-task` skill for each task. Retro fix specific additions:

1. **Before starting implementation**, call `knowledge(action: "list")` with the source
   signal IDs to re-read the original observations. Confirm understanding of the problem.
2. **Cite source signal IDs** in task completion summaries when addressing specific
   observations. Example: "Implemented X per signal KE-001's suggestion."
3. **Commit incrementally** at logical checkpoints within each task.
4. **Stay within scope.** The auto-generated spec defines the fix's boundary. If you
   discover additional issues beyond scope, note them for a future retro cycle — do not
   expand the fix's scope.
5. **Flag assumptions** not covered by the auto-generated spec. Auto-generated specs may
   be less detailed than human-written ones.

### Reviewing

The `retro_fix` tier uses **conditional review**. Determine which path applies:

**Documentation-only changes** (files only under `work/`, `docs/`, `refs/`):
- The review gate can be skipped with an explicit annotation per REQ-TIER-004.
- Advance through reviewing without dispatching specialist review panel.

**Implementation changes** (Go source, tool logic, test files outside doc directories):
- A full specialist review panel is required: conformance, quality, security, testing.
- Conformance review is especially important: verify the implementation addresses the
  source signals' observations — not just that it matches the spec.
- The `orchestrate-review` skill handles panel dispatch. Do not skip.

### Verifying

The verifier runs the retro-adapted Definition of Done (10 items) via
`dod_variant: retro-fix`. The key difference from the standard DoD is item 10:

- **Standard DoD item 10:** All knowledge entries contributed during the feature are
  confirmed, flagged, or retired.
- **Retro-fix DoD item 10:** All source signals linked to the fix feature (identified
  by `KE-...` IDs in the feature's `tags`) must be in `confirmed`, `retired`, or
  otherwise addressed status. Any signal still in `contributed` status fails the check.

Before the verifier runs, ensure:
1. For each source signal ID in the feature's tags, the signal has been confirmed or
   retired via `knowledge(action: "confirm"|"retire", ...)`.
2. The feature has passed through all prior lifecycle stages.
3. All tasks are terminal.

The verifier dispatches with clean context and returns a structured pass/fail report.
On all-pass: transition to `done`. On failure: route to remediation.

## Checklist

```
Copy this checklist and track your progress:
- [ ] Called retro(action: "synthesise") to get current themes
- [ ] Read the full theme entry (observation + suggestion + constituent signals)
- [ ] Confirmed whether the feature was created in human-gated or auto mode
- [ ] Human-gated: human design approved before proceeding
- [ ] Auto mode: design and spec are auto-generated and approved — do not rewrite
- [ ] Source signal IDs are in the feature's tags field
- [ ] Implementation cites source signal IDs for traceability
- [ ] Called knowledge(action: "list") with source signal IDs before coding
- [ ] Conditional review gate assessed: documentation-only or implementation changes?
- [ ] Implementation changes: full specialist review panel completed
- [ ] All source signals confirmed or retired before verifier runs
- [ ] Verifier dispatched with dod_variant: retro-fix
- [ ] DoD item 10 (signal-addressed) passes
```

## Output Format

When completing a retro fix task via `finish`, include in the summary:

```
- Feature mode: human-gated | auto
- Theme: <theme title>
- Source signals: KE-xxx, KE-xxx
- Signals addressed: confirmed/retired status for each
- Review path: documentation-only (skipped) | implementation (panel completed)
```

## Questions This Skill Answers

- How is a retro fix feature different from a standard feature?
- What do I do at each lifecycle stage for a retro fix?
- How does the dual-mode (human-gated vs. auto) affect my workflow?
- What is signal traceability and why does it matter?
- When does conditional review require a full review panel vs. skipping?
- How do I close the loop on source signals after implementation?
- What does the retro-adapted Definition of Done check?
