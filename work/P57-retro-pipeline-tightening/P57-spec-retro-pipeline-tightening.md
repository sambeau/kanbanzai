| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Status | Draft                          |
| Author | spec-author                     |

# Specification: Retrospective Pipeline Tightening

## Problem Statement

This specification implements the design described in
`work/P57-retro-pipeline-tightening/P57-design-retro-pipeline-tightening.md`
(DOC-P57-retro-pipeline-tightening/design-p57-design-retro-pipeline-tightening).

The retrospective pipeline collects process observations but has no path from
synthesised themes to executable fixes. The `retro_fix` fast-track tier exists
in configuration but is unreachable — no tool, skill, or procedure connects a
synthesis theme to a feature entity. This specification defines the bridge:
a `create_fix` action on the `retro` tool, auto-generated document templates
for retro fix specs and designs, an adapted Definition of Done with
signal-addressed verification, a stage binding profile, and a skill codifying
the retro fix workflow.

**In scope:** Theme-to-entity creation path (dual-mode: human-gated and auto),
auto-generated specification and design documents, retro-adapted Definition of
Done, stage binding profile, document type registration fix, document trail,
and `implement-retro-fix` skill.

**Out of scope:** Changing signal contribution via `finish`, changing the
synthesis engine, creating a new `RetroFix` entity type, auto-picking as the
default mode, changing `orchestrate-development` or `orchestrate-review`.

## Requirements

### Functional Requirements

- **REQ-001:** The `retro` MCP tool must accept a `create_fix` action that
  creates a `Feature` entity with `tier: retro_fix` from a synthesis theme.

- **REQ-002:** `create_fix` must accept a `mode` parameter with values
  `"human-gated"` (default) and `"auto"`.

- **REQ-003:** In `"human-gated"` mode, `create_fix` must accept a
  `theme_index` parameter selecting a single theme by its rank in the
  synthesis result.

- **REQ-004:** In `"auto"` mode, `create_fix` must accept `theme_count` and
  `severity_threshold` parameters. When both are specified, the selected
  themes are the intersection: themes that are both within the top
  `theme_count` AND have severity score ≥ `severity_threshold`. When only
  one is specified, it alone determines the selection.

- **REQ-005:** `create_fix` must accept an optional `parent_plan` parameter.
  When omitted, the system must auto-create a plan named
  `Pnn-retro-fixes-{month-year}`. If a plan for the same month-year already
  exists, fixes must be added to the existing plan.

- **REQ-006:** `create_fix` must be idempotent: if a feature already exists
  for the same theme (matched by signal ID set), creation must be skipped
  with the existing feature ID returned.

- **REQ-007:** `create_fix` must accept a `scope` parameter (plan ID, feature
  ID, or `"project"`). The scope is passed through to the underlying
  `synthesise` call.

- **REQ-008:** In auto mode, `create_fix` must auto-generate and auto-approve
  a design document for each created fix feature. The design document must
  include the sections: Overview, Goals and Non-Goals, Design, Alternatives
  Considered, and Dependencies.

- **REQ-009:** `create_fix` must auto-generate and auto-approve a
  specification document for each created fix feature. The spec must include
  the sections: Overview, Scope, Functional Requirements, Non-Functional
  Requirements, and Acceptance Criteria.

- **REQ-010:** The auto-generated specification must derive its content from
  the synthesis theme: the Overview from the theme title and representative
  observation, the Functional Requirements from the top suggestion converted
  to requirement language, and the Acceptance Criteria from the suggestion
  expressed as given/when/then.

- **REQ-011:** In auto mode, `create_fix` must auto-generate and auto-approve
  a dev-plan document and auto-decompose into tasks via
  `decompose(action: "propose")` for each created fix feature.

- **REQ-012:** The close-out verifier must support a retro-adapted Definition
  of Done variant where item 10 checks signal resolution: all source retro
  signals linked to the fix feature must be in `confirmed`, `retired`, or
  otherwise addressed status.

- **REQ-013:** `stage-bindings.yaml` must include a `retro-fixing` profile
  documenting the tier's gate modes for both `human-gated` and `auto` modes,
  the verifier dispatch configuration, and the `dod_variant: retro-fix`
  designation.

- **REQ-014:** `RetroService.Report()` must register generated reports with
  document type `"retro"` instead of `"report"`.

- **REQ-015:** `.kbz/skills/implement-retro-fix/SKILL.md` must exist and
  document the retro fix workflow stages, vocabulary, anti-patterns, and
  the procedure for each lifecycle stage under retro fix context.

- **REQ-016:** In auto mode, `create_fix` must return a batch summary
  containing: the plan ID, a list of created feature IDs, and for each
  feature its spec and dev-plan document IDs.

- **REQ-017:** A `Feature` entity created by `create_fix` must have
  `tags` containing `"retro"` followed by the source signal IDs that
  generated the theme.

- **REQ-018:** In auto mode, `create_fix` must transition each created
  feature's lifecycle through all stages with auto gates: `designing`
  (auto-approve design), `specifying` (auto-approve spec), `dev-planning`
  (auto-approve dev-plan), then into `developing` for implementation.

### Non-Functional Requirements

- **REQ-NF-001:** `create_fix` in auto mode with `theme_count: 3` must
  complete the full creation pipeline (synthesis, plan creation, feature
  creation, document generation, document approval) for all three themes
  within 30 tool calls total.

- **REQ-NF-002:** The auto-generated design document must be recognizable
  as a valid design document — it must pass `doc(action: "validate")`
  without structural errors.

- **REQ-NF-003:** The auto-generated specification must be recognizable
  as a valid specification — it must contain all five required sections
  (Problem Statement, Requirements, Constraints, Acceptance Criteria,
  Verification Plan) with non-empty content in each.

- **REQ-NF-004:** The existing `retro_fix` tier configuration in
  `DefaultFastTrackConfig` must not change. The gate mode overrides for
  auto mode must be applied at runtime, not by modifying the tier config.

- **REQ-NF-005:** `create_fix` must not modify or delete existing entity
  state for features or plans it did not create itself. Idempotency must
  be read-only for pre-existing entities.

## Constraints

- The `Feature` entity type must not be extended with new fields — the
  `tier` field and `tags` array are sufficient for retro fix typing.
- No new entity type (`RetroFix`) may be created.
- The `finish` tool's `retrospective` parameter must not change.
- The synthesis engine's clustering and ranking algorithms must not change.
- The existing `retro_fix` tier configuration in `DefaultFastTrackConfig`
  must remain at its current values (design: human, spec: auto, dev_plan:
  auto, review: conditional, max_cycles: 3).
- The conditional review gate logic in `evaluateConditional` must not change.
- This specification does NOT cover automatic signal retirement after fix
  verification — signals must be manually confirmed or retired.
- This specification does NOT cover multi-theme fix creation (a single
  `create_fix` call targeting multiple theme indices).
- The `designing` stage binding's human gate must remain in effect for
  `"human-gated"` mode. Auto mode overrides must be scoped to the specific
  features created in that mode.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a prior `retro(action: "synthesise")`
  with scope `"project"`, when `retro(action: "create_fix", scope: "project",
  theme_index: 1)` is called, then a `FEAT-...` entity is returned with
  `tier: retro_fix`, `tags` containing `"retro"` plus signal IDs, and a
  registered specification document.

- **AC-002 (REQ-003):** Given a synthesis with 5 ranked themes, when
  `create_fix(mode: "human-gated", theme_index: 3)` is called, then the
  feature is created from the theme ranked #3, not any other theme.

- **AC-003 (REQ-004):** Given a synthesis with themes at severity scores
  [50, 30, 20, 15, 10], when `create_fix(mode: "auto", theme_count: 3,
  severity_threshold: 20)` is called, then exactly 2 features are created
  (themes with scores 50 and 30; the theme at score 20 meets the threshold
  but is outside the top 3, and the theme at score 15 is outside both).

- **AC-004 (REQ-005):** Given no `parent_plan` is provided, when
  `create_fix(mode: "auto")` is called, then a plan named
  `Pnn-retro-fixes-may-2026` (or current month-year) is created and the
  fix feature is placed under it.

- **AC-005 (REQ-005):** Given a plan `P58-retro-fixes-june-2026` already
  exists, when `create_fix(mode: "auto")` is called in June 2026, then
  the new fix feature is added to the existing plan, not a duplicate.

- **AC-006 (REQ-006):** Given a fix feature already exists for theme #1
  (matched by signal IDs), when `create_fix(theme_index: 1)` is called
  again, then the existing feature ID is returned and no new feature
  is created.

- **AC-007 (REQ-008):** Given `create_fix(mode: "auto", theme_count: 1)`
  is called, when the created feature is inspected with
  `doc(action: "list", owner: "<FEATURE-ID>", type: "design")`, then an
  approved design document exists with all required sections populated
  from the theme data.

- **AC-008 (REQ-009, REQ-010):** Given `create_fix` creates a feature from
  a theme with suggestion "Add X to the handoff assembly pipeline", when
  the auto-generated spec is read, then it contains a functional requirement
  describing the addition of X and an acceptance criterion expressing the
  suggestion as a given/when/then assertion.

- **AC-009 (REQ-011):** Given `create_fix(mode: "auto")` creates a feature,
  when tasks are listed via `entity(action: "list", type: "task",
  parent_feature: "<FEATURE-ID>")`, then at least one task exists.

- **AC-010 (REQ-012):** Given a retro fix feature with source signal IDs
  in its tags, when the verifier runs the retro-adapted DoD, then item 10
  checks that all source signals are confirmed, retired, or linked — and
  fails if any signal is still in `contributed` status.

- **AC-011 (REQ-013):** Given the system is initialised, when
  `stage-bindings.yaml` is read, then a `retro-fixing` profile exists
  with `modes.human-gated` and `modes.auto` sections, each containing
  the correct gate mode values.

- **AC-012 (REQ-014):** Given `retro(action: "report", scope: "project",
  output_path: "work/retro/test-report.md")` is called, when the document
  is queried via `doc(action: "get", id: "...")`, then its `type` field
  is `"retro"`, not `"report"`.

- **AC-013 (REQ-015):** Given the repository, when
  `.kbz/skills/implement-retro-fix/SKILL.md` is read, then it contains
  sections for vocabulary, anti-patterns, procedure, and stage-specific
  guidance covering designing, specifying, dev-planning, developing,
  reviewing, and verifying stages.

- **AC-014 (REQ-016):** Given `create_fix(mode: "auto", theme_count: 2)`
  completes, when the response is examined, then it contains a plan ID
  and two feature IDs, each with a spec and dev-plan document ID.

- **AC-015 (REQ-017):** Given a feature created by `create_fix` from a
  theme containing signal IDs `KE-001` and `KE-002`, when the feature is
  retrieved, then its `tags` field is `["retro", "KE-001", "KE-002"]`.

- **AC-016 (REQ-018):** Given `create_fix(mode: "auto")` creates a feature,
  when the feature's status is checked, then it has advanced through
  `designing`, `specifying`, and `dev-planning` to `developing` without
  any human checkpoint having been raised.

- **AC-017 (REQ-NF-004):** Given the system configuration, when
  `DefaultFastTrackConfig()` is called, then the `retro_fix` tier config
  is unchanged from its pre-P57 values.

- **AC-018 (REQ-NF-005):** Given a `FEAT-...` entity exists that was not
  created by `create_fix`, when `create_fix` runs in a scope that includes
  that feature's signals, then the pre-existing feature is not modified.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Integration test: run synthesise, call create_fix, assert feature entity shape |
| AC-002 | Test | Unit test: create_fix with explicit theme_index, verify correct theme selected |
| AC-003 | Test | Unit test: intersection logic for theme_count + severity_threshold |
| AC-004 | Test | Integration test: create_fix without parent_plan, verify plan auto-created with correct name |
| AC-005 | Test | Integration test: create_fix when plan exists, verify feature added to existing plan |
| AC-006 | Test | Unit test: double create_fix call, verify second call returns existing feature |
| AC-007 | Test | Integration test: auto mode, verify design doc exists, approved, with correct sections |
| AC-008 | Test | Integration test: auto mode, verify spec content derived from theme suggestion |
| AC-009 | Test | Integration test: auto mode, verify tasks decomposed after creation |
| AC-010 | Test | Unit test: verifier DoD item 10 with contributed signals, verify fail; with confirmed signals, verify pass |
| AC-011 | Inspection | Read stage-bindings.yaml, verify retro-fixing profile structure |
| AC-012 | Test | Integration test: retro report, verify doc type is "retro" |
| AC-013 | Inspection | Read implement-retro-fix SKILL.md, verify required sections present |
| AC-014 | Test | Integration test: auto mode with theme_count:2, verify response shape |
| AC-015 | Test | Unit test: feature tags after create_fix, verify signal IDs present |
| AC-016 | Test | Integration test: auto mode, verify feature status is "developing" with no checkpoints |
| AC-017 | Test | Unit test: DefaultFastTrackConfig(), verify retro_fix tier unchanged |
| AC-018 | Test | Integration test: create_fix in scope with pre-existing features, verify no mutation |
