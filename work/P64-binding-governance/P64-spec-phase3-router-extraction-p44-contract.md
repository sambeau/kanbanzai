| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-11                     |
| Status | Draft                          |
| Author | spec-author                    |

## Overview

This specification implements the design described in
`work/P64-binding-governance/P64-design-binding-governance-implementation-plan.md`
(DOC: P64-binding-governance/design-p64-design-binding-governance-implementation-plan, approved).

Phase 3 of the Binding Governance Implementation Plan establishes the architectural
contract for P44 by extracting routing into a single code-owned component and reducing
the synchronisation surface count from 7 to 4. The design chooses Option C (hybrid: code
routes, YAML describes) as the explicit architectural direction.

**Which design decisions this specification implements:**

- **Decision 4:** A single routing companion file (`internal/binding/routing.yaml`)
  generates the four hand-edited tables (`validStages`, `workableStatuses`,
  `FeatureStatus*` constants, `BugStatus*` constants) that currently drift out of sync.
- **P44 interface contract:** A pure `Resolve(featureID) → BindingResolution` function
  that P44's `StageController` and `PipelineTransitionHook.AfterTransition` can call
  synchronously without I/O.
- **Header documentation contract:** The canonical `stage-bindings.yaml` gains a header
  comment naming the boundary between routing decisions (code-owned) and agent behaviour
  descriptions (YAML-owned).

**Dependencies (assumed complete before this feature is implemented):**

- Phase 1: `ValidateBindingFile` invoked at server startup; allowlists synchronised;
  embedded YAML drift CI test in place.
- Phase 2: `retro-fixing` YAML block removed; `bug-developing`/`bug-reviewing` bindings
  added; tier-conditional skill substitution working in the developing path;
  `bugStatusToBindingKey` table exists.

## Scope

**In scope:**

- Subsystem A: `internal/binding/router.go` — the `Resolve` function, `BindingResolution`
  struct, and typed error surface.
- Subsystem C: `internal/binding/routing.yaml` — the routing companion file; the
  `go generate` step that produces the four tables from it.
- Subsystem B: The header comment in the canonical `stage-bindings.yaml`.
- P44 interface: `Resolve(featureID)` function signature, `BindingResolution` fields, and
  typed errors.

**Out of scope (this specification does NOT cover):**

- Implementation of the code generator itself. This spec defines the generator's input
  format (`routing.yaml`), output targets (the four tables), and `go generate` wiring;
  the generator's internal implementation is an implementation detail for the dev-plan.
- Phase 1 or Phase 2 work (startup validation, retro-fix binding removal, bug bindings).
  This spec assumes those phases are complete and their outputs are present.
- P44's `Provider` interface, `StageController` internals, token tracking, or fallback
  chains. Those belong to P44's own design.
- Consumer-facing configuration. `routing.yaml` is code-internal; consumers do not edit it.
- Doc-publishing routing. The `pipeline-coordinator` orchestration is not a feature
  lifecycle stage and remains outside `Resolve`.

## Functional Requirements

- **REQ-001:** The system MUST provide a pure function `Resolve(featureID) → BindingResolution`
  that accepts a feature identifier and returns a fully populated `BindingResolution` or
  a typed error. The function MUST perform no I/O, no logging, no network calls, and no
  file-system reads in the hot path.

- **REQ-002:** `BindingResolution` MUST expose the following fields:
  `BindingKey` (string), `SkillOverrides` ([]string), and `ModeProfile`
  (*FastTrackTier). `BindingKey` identifies the stage-binding key for the 3.0 pipeline
  to load. `SkillOverrides` contains tier-conditional skill substitutions (e.g.,
  `implement-retro-fix` substituted for `implement-task`). `ModeProfile` carries the
  tier-derived gate mode (`auto`/`human`/`conditional`) and max-cycles cap from
  `FastTrackConfig`.

- **REQ-003:** `Resolve` MUST accept `FastTrackConfig` as one of its inputs and use it to
  derive per-tier gate modes and max-review-cycle caps for the returned
  `BindingResolution`.

- **REQ-004:** The `Resolve` function MUST fail with a typed, distinguishable error for
  each of the following conditions: (a) `ErrNoBinding` — the feature's status has no
  routing entry; (b) `ErrUnknownTier` — the feature's tier is not a recognised tier value;
  (c) `ErrSkillMissing` — a skill referenced by the resolved binding is not present on
  disk.

- **REQ-005:** The pipeline's `stepLookupBinding` method MUST call `Resolve` to obtain the
  binding key instead of looking up the bare `state.Stage` directly from the binding
  registry.

- **REQ-006:** The system MUST provide a routing companion file at
  `internal/binding/routing.yaml` that declares, for every routable feature status and
  bug status, the binding key it maps to and whether skill substitution applies. Each
  entry MUST declare at minimum: the status name, the binding key, and whether it is a
  feature-status or bug-status entry.

- **REQ-007:** The following Go source constructs MUST be generated from
  `internal/binding/routing.yaml` via `go generate`: `validStages` (the allowlist used by
  `ValidateBindingFile`), `workableStatuses` (the allowlist used by the pipeline's
  lifecycle validator), the `FeatureStatus*` enum constants, and the `BugStatus*` enum
  constants. After generation, no hand-edited table of these four constructs may remain.
  The `bugStatusToBindingKey` mapping table from Phase 2 is also generated from this file.

- **REQ-008:** The canonical `.kbz/stage-bindings.yaml` MUST include a header comment that
  states: "This file describes how agents act per stage. It does not decide which agent
  runs — that is owned by `internal/binding/router.go` and `internal/config`
  (FastTrack)."

- **REQ-009:** After `Resolve` returns a `BindingResolution`, the existing 3.0 pipeline
  steps (load skill, load role, assemble context) MUST operate on the resolved binding
  key without requiring modification to their internal logic. The pipeline's
  `stepLookupBinding` is the only pipeline step that changes.

- **REQ-010:** The routing companion file `internal/binding/routing.yaml` MUST NOT be
  placed in `.kbz/` or any consumer-editable location. It is code-internal and consumers
  do not customise it.

## Non-Functional Requirements

- **REQ-NF-001:** `Resolve` MUST return within 1 millisecond at p99 under single-threaded
  invocation when called with a valid feature ID and a fully populated `FastTrackConfig`.

- **REQ-NF-002:** The `go generate` step producing the four tables from `routing.yaml`
  MUST complete within 500 milliseconds on standard development hardware.

- **REQ-NF-003:** No existing routing behaviour (feature routing, bug routing,
  FastTrack tier resolution) may regress. All features and bugs that were routable before
  Phase 3 MUST route to the same binding key and skill set after Phase 3.

## Constraints

- `Resolve` is a pure function. It MUST NOT perform I/O (no file reads, no network calls,
  no logging, no database queries) in the hot path. `FastTrackConfig` and any lookup
  tables derived from `routing.yaml` are passed in; `Resolve` reads only its arguments.
- The routing companion file lives at `internal/binding/routing.yaml`. It is NOT
  consumer-customisable. Placing it in `.kbz/` is explicitly forbidden by the design
  (Option C: routing is a code concern).
- The schema version of `stage-bindings.yaml` stays at `2`. A schema bump would require
  a separate design and is out of scope for this feature.
- `BindingResolution` MUST NOT expose `FastTrackConfig` directly. P44 consumes
  `BindingResolution`, not the config struct.
- The `pipeline-coordinator` orchestration path (`doc-publishing`) is NOT routed through
  `Resolve`. It remains a separately triggered code path outside the feature-lifecycle
  pipeline.
- No new synchronisation surfaces may be introduced. The post-Phase-3 surface count must
  be 4 (canonical YAML, routing companion file, role files, skill files).
- All existing tests for `stepLookupBinding`, `stepResolveStage`, and the binding
  registry lookups MUST continue to pass. Backward compatibility is required.
- `BindingResolution` MUST NOT expose YAML content directly. Its fields are derived from
  routing decisions and `FastTrackConfig`, not from `StageBinding` structs. The YAML's
  role/skill content is consumed by the existing 3.0 pipeline, not by this interface.
- The code generator itself is an implementation detail. This spec defines its
  input/output contract; the generator's internal implementation is out of scope.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a feature with `status: developing` and `tier: feature`,
  when `Resolve(featureID)` is called, then it returns a `BindingResolution` with
  `BindingKey == "developing"`, `ModeProfile.Review == "auto"`, and no error —
  without performing any file I/O or logging.

- **AC-002 (REQ-001, REQ-003):** Given a retro-fix feature with `status: developing` and
  `tier: retro_fix`, when `Resolve(featureID)` is called, then it returns a
  `BindingResolution` with `SkillOverrides` containing the substitution of
  `implement-retro-fix` for `implement-task` and `ModeProfile.Review == "conditional"`.

- **AC-003 (REQ-004):** Given a feature whose status has no entry in `routing.yaml`,
  when `Resolve(featureID)` is called, then it returns `ErrNoBinding` with a message
  naming the unresolvable status.

- **AC-004 (REQ-004):** Given a feature whose `tier` field is not one of `retro_fix`,
  `bug_fix`, `feature`, or `critical`, when `Resolve(featureID)` is called, then it
  returns `ErrUnknownTier` with a message naming the unrecognised tier.

- **AC-005 (REQ-004):** Given a feature whose resolved binding references a skill not on
  disk, when `Resolve(featureID)` is called, then it returns `ErrSkillMissing` with a
  message naming the missing skill and the binding key that required it.

- **AC-006 (REQ-005):** Given a feature in `status: developing`, when the pipeline runs
  `stepLookupBinding`, then `state.Stage` is used as input to `Resolve` and
  `state.Binding` is populated from the returned `BindingResolution.BindingKey`.

- **AC-007 (REQ-007):** Given `internal/binding/routing.yaml` with a declared status
  `developing`, when `go generate` runs, then `validStages` contains `"developing": true`,
  `workableStatuses` contains `"developing"`, and the `FeatureStatusDeveloping` constant
  is generated.

- **AC-008 (REQ-007):** Given a status is added to `routing.yaml` and `go generate` runs,
  when `ValidateBindingFile` executes, then the new status passes the stage-name allowlist
  check without requiring a separate hand-edit to `validStages`.

- **AC-009 (REQ-008):** Given the canonical `.kbz/stage-bindings.yaml`, when the file is
  read, then the first 20 lines contain a comment stating that routing is owned by
  `internal/binding/router.go` and `internal/config` (FastTrack).

- **AC-010 (REQ-009):** Given `Resolve` returns a `BindingResolution` with
  `BindingKey = "developing"`, when the pipeline continues past `stepLookupBinding`,
  then `stepLoadSkill` and `stepLoadRole` operate on the binding registry's
  `developing` entry without requiring modified logic.

- **AC-011 (REQ-010):** Given the repository file tree, then
  `internal/binding/routing.yaml` exists and `.kbz/routing.yaml` does not exist.

- **AC-012 (REQ-002):** Given `Resolve` returns a `BindingResolution`, when the caller
  inspects the struct, then `BindingKey` is a non-empty string, `SkillOverrides` is a
  string slice (possibly empty), and `ModeProfile` is either nil (tier has no FastTrack
  entry) or a populated `*FastTrackTier` with `Review`, `Design`, `Spec`, `DevPlan`
  gate modes and `MaxCycles` accessible.

- **AC-013 (REQ-NF-003):** Given all existing tests for the pipeline and binding registry,
  when the full test suite runs after Phase 3 implementation, then every previously
  passing test still passes.

- **AC-014 (REQ-NF-001):** Given a benchmark invoking `Resolve` with a valid feature ID
  and a fully populated `FastTrackConfig` in a single-threaded Go test, when the
  benchmark runs for 10,000 iterations, then the p99 latency is ≤ 1 millisecond.

- **AC-015 (REQ-NF-002):** Given a valid `internal/binding/routing.yaml` with all
  currently declared statuses, when `go generate` runs on standard development
  hardware, then the generation step completes within 500 milliseconds.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated unit test: call `Resolve` with a feature at `developing`/`feature` tier; assert `BindingKey == "developing"` and `ModeProfile.Review == "auto"`. Verify no I/O occurs by instrumenting the call with a filesystem mock or by confirming the function has no `fs.FS` or logger parameters. |
| AC-002 | Test | Automated unit test: call `Resolve` with a retro-fix feature at `developing` tier; assert `SkillOverrides` maps `implement-task` → `implement-retro-fix` and `ModeProfile.Review == "conditional"`. |
| AC-003 | Test | Automated unit test: call `Resolve` with a feature whose status is absent from the routing table; assert returned error is `ErrNoBinding` (via `errors.Is`) and message contains the status name. |
| AC-004 | Test | Automated unit test: call `Resolve` with a feature whose tier field is an invalid string (e.g., `"nonexistent"`); assert returned error is `ErrUnknownTier` (via `errors.Is`) and message contains the tier value. |
| AC-005 | Test | Automated unit test: call `Resolve` with a skill checker that returns `false` for the binding's required skill; assert returned error is `ErrSkillMissing` (via `errors.Is`) and message names the skill and binding key. |
| AC-006 | Test | Automated integration test: construct a pipeline with `Resolve` wired into `stepLookupBinding`; run the pipeline against a `developing` feature; assert `state.Binding` is populated from the registry using the key from `BindingResolution`. |
| AC-007 | Test | Automated generation test: run `go generate` against a known `routing.yaml`; assert the generated `validStages` map, `workableStatuses` slice, and enum constants match the YAML declarations. |
| AC-008 | Test | Automated generation + validation test: add a new status to `routing.yaml`, run `go generate`, then run `ValidateBindingFile` against `stage-bindings.yaml` with a binding for that status; assert no "invalid stage name" error. |
| AC-009 | Inspection | Code review: open `.kbz/stage-bindings.yaml` and verify the header comment within the first 20 lines names `internal/binding/router.go` and `internal/config` (FastTrack) as the routing owners. |
| AC-010 | Test | Automated integration test: run the full pipeline (steps 1–5) after `Resolve` integration; assert `stepLoadSkill` loads the correct skill file and `stepLoadRole` loads the correct role file without code changes to those steps. |
| AC-011 | Inspection | Code review: verify `internal/binding/routing.yaml` exists and no `routing.yaml` file exists under `.kbz/`. |
| AC-012 | Test | Automated unit test: call `Resolve` with a valid feature; type-assert each field of the returned `BindingResolution` and verify non-nil/non-empty constraints per AC-012. |
| AC-013 | Test | Run the full test suite (`go test ./...`) after Phase 3 changes; assert zero test regressions against the pre-Phase-3 baseline. |
| AC-014 | Test | Go benchmark test: run `Resolve` 10,000 times in a single-threaded benchmark; assert p99 latency ≤ 1ms. |
| AC-015 | Test | Timed generation test: run `go generate` with a stopwatch; assert wall-clock time ≤ 500ms on standard dev hardware. |
