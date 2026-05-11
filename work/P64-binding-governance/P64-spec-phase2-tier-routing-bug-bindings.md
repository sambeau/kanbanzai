# Specification: Tier Routing and Bug Bindings

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-11                     |
| Status | approved |
| Author | spec-author                    |
| Plan   | P64-binding-governance         |
| Tier   | feature                        |

---

## Problem Statement

This specification implements Phase 2 ("Resolve the two failures and reduce drift") of the
Binding Governance Implementation Plan, as described in the approved design document:

> `work/P64-binding-governance/P64-design-binding-governance-implementation-plan.md`
> (DOC: P64-binding-governance/design-p64-design-binding-governance-implementation-plan).

Phase 2 addresses two concrete pipeline failures and eliminates the dead schema surface
that enabled them:

1. **Retro-fix routing is unwired.** Every feature carries a `tier` field, but the
   pipeline never reads it. The `implement-retro-fix` skill exists on disk but is
   unreachable from any code path because no binding resolves to it. The `retro-fixing`
   block in `stage-bindings.yaml` is a parallel, silently-broken routing mechanism that
   no feature status ever matches.

2. **Bug-fix has no pipeline binding.** Twelve `BugStatus` constants map to zero
   stage-binding keys. The `WorktreeTransitionHook` creates a worktree when a bug
   transitions to `in-progress`, then the pipeline's lifecycle validator rejects every
   `BugStatus` — stranding the agent inside a worktree it cannot use.

This specification implements two design decisions from the parent design:

- **Decision 1:** Resolve retro-fixing by deleting the YAML block and wiring
  `implement-retro-fix` through a FastTrack tier-conditional skill include in the
  `developing` path.

- **Decision 2:** Add `bug-developing` and `bug-reviewing` bindings rather than
  declare bugs out-of-pipeline.

It also removes orphaned tier-aware fields (`Profile`, `Tier`, `Modes`, `Verifying`)
from the `StageBinding` model — any surviving field must have a runtime consumer —
and documents `FastTrackConfig` as the system of record for tier-aware behaviour.

**Dependency.** This feature assumes Phase 1 (startup validation) is complete:
`ValidateBindingFile` is wired at server startup, allowlists are synced with the
canonical YAML, and the embedded copy is CI-enforced. Phase 2 changes are themselves
binding changes that must pass the validation Phase 1 enforces.

**Scope boundary.** This specification covers: removal of the `retro-fixing` YAML
block, FastTrack tier-conditional skill substitution for `retro_fix` features,
addition of `bug-developing` and `bug-reviewing` binding keys, the
`bugStatusToBindingKey` mapping, removal of orphaned `StageBinding` fields, and
the three enumerated failure-mode behaviours (no-binding, unknown-tier,
bug-out-of-pipeline).

**Explicitly out of scope:** Phase 3 router extraction (`internal/binding/router.go`
and `BindingResolution`), the generated `routing.yaml` companion file, P44 interface
design, `go generate` wiring, the `doc-publishing` `pipeline-coordinator` admission,
and any consumer-facing binding schema changes.

---

## Requirements

### Functional Requirements

- **REQ-001:** The `retro-fixing` top-level block MUST be removed from
  `.kbz/stage-bindings.yaml`. No key, sub-key, or value from that block may remain
  in the canonical or embedded copy.

- **REQ-002:** When the tier router resolves a feature whose `Tier` is `retro_fix`
  and whose status routes to the `developing` binding, the resolved `skills` list
  MUST substitute `implement-retro-fix` for `implement-task`. The pipeline's
  `stepLoadSkill` MUST remain unchanged; only the resolved skill list differs.

- **REQ-003:** Two new top-level binding keys, `bug-developing` and `bug-reviewing`,
  MUST be added to `.kbz/stage-bindings.yaml` and the embedded copy. Each MUST
  include the minimum required fields for its stage: `roles`, `skills`,
  `orchestration`, and any other fields required by `ValidateBindingFile`.

- **REQ-004:** The `workableStatuses` enumerator in the binding subsystem MUST be
  extended to include the `BugStatus` constants `in-progress` and `needs-review`.

- **REQ-005:** The pipeline's `stepResolveStage` MUST include a pure mapping function
  `bugStatusToBindingKey` that maps `BugStatus` values to `bug-*` binding keys:
  `in-progress` → `bug-developing`, `needs-review` → `bug-reviewing`. The function
  MUST be a pure data table — no I/O, no logging, no side effects.

- **REQ-006:** All other `BugStatus` values (`reported`, `triaged`, `closed`,
  `resolved`, `verified`, `wontfix`, `duplicate`, `invalid`, `deferred`,
  `in-progress`, `needs-review` excepted) MUST remain out-of-pipeline by explicit
  declaration. Attempting to route through any of these statuses MUST produce a
  per-task failure with an explicit "out-of-pipeline" message.

- **REQ-007:** The orphaned fields `Profile`, `Tier`, `Modes`, and `Verifying` MUST
  be removed from the `StageBinding` struct in `internal/binding/`. No field on
  `StageBinding` may lack a runtime consumer after this change.

- **REQ-008:** `FastTrackConfig` MUST be documented in code (via Go doc comments on
  the struct and its public methods) and in the `stage-bindings.yaml` header as the
  system of record for tier-aware behaviour. The YAML header MUST state that the file
  describes how agents act per stage, not which agent runs — that is owned by
  `FastTrackConfig` (code).

- **REQ-009:** When the pipeline encounters a feature or bug status that maps to no
  binding key, it MUST produce a per-task failure with an actionable message that
  names the unresolvable status and states either "add a binding to
  `stage-bindings.yaml`" or "list it as out-of-pipeline."

- **REQ-010:** When the tier router resolves a feature whose `Tier` is not a member
  of the enumerated tier set (`feature`, `bug_fix`, `retro_fix`, `critical`), it
  MUST produce a hard failure at the `Resolve` call site. The failure message MUST
  name the unrecognised tier value.

- **REQ-011:** When the pipeline encounters a feature whose `Tier` is `bug_fix` and
  whose status has no `bug-*` binding, it MUST produce a per-task failure with an
  explicit out-of-pipeline message naming the status and tier.

- **REQ-012:** The synchronisation surface count MUST be reducible from 10 to 7 after
  Phase 2: the embedded-vs-canonical drift is CI-enforced (Phase 1), the
  validStages-vs-YAML drift is CI-enforced (Phase 1), and the orphaned schema fields
  are removed (this phase).

### Non-Functional Requirements

- **REQ-NF-001:** Backward compatibility. Existing routing behaviour for features
  whose `Tier` is `feature` or `critical` and whose status uses a standard binding
  (not `bug-*`) MUST be unchanged. No existing test for non-retro_fix, non-bug
  routing may regress.

- **REQ-NF-002:** Schema version. The `stage-bindings.yaml` `schema_version` MUST
  remain at `2`. No schema version bump is permitted.

- **REQ-NF-003:** Migration safety. Removing the `retro-fixing` block from the
  embedded YAML MUST be safe for consumers — no consumer can have been routing
  through it because it was unroutable. Adding `bug-developing` and
  `bug-reviewing` MUST be purely additive.

- **REQ-NF-004:** Test coverage. Every new code path introduced by this feature MUST
  have at least one automated test: tier-conditional skill substitution (REQ-002),
  `bugStatusToBindingKey` mapping (REQ-005), out-of-pipeline failure messages
  (REQ-006, REQ-011), unknown-tier failure (REQ-010), and no-binding failure
  (REQ-009).

- **REQ-NF-005:** Pure mapping function. `bugStatusToBindingKey` MUST be callable
  synchronously, with no I/O, no logging, and no allocation beyond its return value.
  This ensures it can be consumed by P44's `PipelineTransitionHook` in Phase 3
  without rework.

- **REQ-NF-006:** Field removal completeness. After REQ-007 is implemented, no
  reference to `Profile`, `Tier`, `Modes`, or `Verifying` on `StageBinding` may
  remain in any `.go` file. The compiler MUST reject any use of these fields.

---

## Constraints

- **Phase 1 dependency.** This feature MUST NOT be implemented before Phase 1
  (startup validation) is complete. `ValidateBindingFile` must be wired at server
  startup, allowlists must be synced, and the embedded copy must be CI-enforced.
  Phase 2 changes are binding changes that must pass Phase 1's validation.

- **No schema version bump.** The `schema_version` field in `stage-bindings.yaml`
  MUST remain `2`. A schema bump would be warranted only if Subsystem B's content
  shape changed; under this design it does not.

- **Backward compatibility.** Existing routing for `feature`-tier and `critical`-tier
  features MUST NOT change. The `stepLoadSkill` pipeline step MUST NOT be modified.
  The `WorktreeTransitionHook` MUST NOT be modified.

- **Scope exclusion — Phase 3 concerns.** This specification does NOT cover: router
  extraction into `internal/binding/router.go`, the `BindingResolution` struct, the
  generated `routing.yaml` companion file, `go generate` wiring, or the P44
  `Resolve(featureID) → BindingResolution` interface contract.

- **Scope exclusion — Phase 1 concerns.** This specification does NOT cover: wiring
  `ValidateBindingFile` at startup, syncing allowlists, CI-enforcing the embedded
  copy, adding `kbz binding doctor`, or validating `doc-publishing` orchestration.

- **Scope exclusion — consumer-facing changes.** This specification does NOT cover:
  any consumer-exposed API, CLI surface, or MCP tool signature change. All changes
  are internal to the binding subsystem and the pipeline.

- **Inherited constraint — P52/P43 (FastTrack).** Tier inference is performed at
  entity-creation time via `inferTier`; the tier is never re-inferred afterwards.
  This feature reads `Feature.Tier` for routing; it MUST NOT recompute or reassign it.

- **Inherited constraint — P51 (Handoff Pipeline).** The 3.0 pipeline is the only
  assembly path. This feature MUST NOT introduce a parallel routing path; its
  routing decision MUST feed into the existing pipeline's `stepLookupBinding`.

- **Inherited constraint — B69.** The `status()` orientation block already surfaces
  skill names inline. This feature MUST NOT modify the orientation block or any
  skill-discovery mechanism.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given the canonical `.kbz/stage-bindings.yaml`, when it is
  parsed by the binding loader, then no key matching `retro-fixing` exists at the
  top level, and `ValidateBindingFile` passes against the resulting struct.

- **AC-002 (REQ-002):** Given a feature with `Tier = retro_fix` and `Status =
  developing`, when the pipeline resolves the binding, then the resolved `skills`
  list contains `implement-retro-fix` and does NOT contain `implement-task`.

- **AC-003 (REQ-002, REQ-NF-001):** Given a feature with `Tier = feature` and
  `Status = developing`, when the pipeline resolves the binding, then the resolved
  `skills` list contains `implement-task` and does NOT contain `implement-retro-fix`.

- **AC-004 (REQ-003, REQ-004):** Given a bug with `Status = in-progress`, when
  `stepResolveStage` processes it, then it resolves to the `bug-developing` binding
  key, and the binding's `roles` and `skills` fields are populated with valid
  references that pass `ValidateBindingFile`.

- **AC-005 (REQ-003, REQ-004):** Given a bug with `Status = needs-review`, when
  `stepResolveStage` processes it, then it resolves to the `bug-reviewing` binding
  key, and the binding's `roles` and `skills` fields are populated with valid
  references that pass `ValidateBindingFile`.

- **AC-006 (REQ-005):** Given the `bugStatusToBindingKey` function, when called with
  `BugStatusInProgress`, then it returns the binding key `"bug-developing"`. When
  called with `BugStatusNeedsReview`, then it returns `"bug-reviewing"`.

- **AC-007 (REQ-006):** Given a bug with `Status = triaged` (or any BugStatus not
  mapped to a `bug-*` binding), when the pipeline attempts to resolve a binding,
  then it produces a per-task failure whose message contains the status name and the
  phrase "out-of-pipeline."

- **AC-008 (REQ-007):** Given the `StageBinding` struct in `internal/binding/`, when
  the codebase is compiled, then no field named `Profile`, `Tier`, `Modes`, or
  `Verifying` exists on the struct, and no `.go` file references any such field on a
  `StageBinding` value.

- **AC-009 (REQ-008):** Given the `FastTrackConfig` struct in `internal/config/`,
  when `go doc` is run against it, then the doc comment states that `FastTrackConfig`
  is the system of record for tier-aware behaviour. Given the canonical
  `stage-bindings.yaml`, when read, then its header comment states that routing
  decisions are owned by `FastTrackConfig` (code), not by this file.

- **AC-010 (REQ-009):** Given a feature whose status maps to no binding key (e.g., a
  hypothetical status added without a corresponding binding), when the pipeline
  resolves the binding, then it produces a per-task failure whose message names the
  unresolvable status and includes an actionable remediation hint.

- **AC-011 (REQ-010):** Given a feature whose `Tier` field contains a value not in
  the enumerated set (`feature`, `bug_fix`, `retro_fix`, `critical`), when
  `Resolve` is called, then it returns a typed error (`ErrUnknownTier`) whose
  message names the unrecognised tier value.

- **AC-012 (REQ-011):** Given a feature with `Tier = bug_fix` and a status that has
  no `bug-*` binding (e.g., `designing`), when the pipeline resolves the binding,
  then it produces a per-task failure whose message names both the status and the
  tier and states that the combination is out-of-pipeline.

- **AC-013 (REQ-012):** Given the post-Phase-1 baseline of 10 synchronisation
  surfaces, when Phase 2 changes are complete, then the number of independently
  maintained synchronisation surfaces is 7: the embedded-vs-canonical drift is
  CI-enforced, the validStages-vs-YAML drift is CI-enforced, and the orphaned
  `StageBinding` fields are removed.

- **AC-014 (REQ-NF-002):** Given the canonical `stage-bindings.yaml` after all
  Phase 2 changes, when its `schema_version` field is read, then its value is `2`.

- **AC-015 (REQ-NF-004):** Given the test suite, when `go test
  ./internal/binding/... ./internal/context/...` is run, then at least one test
  case exists for each of: tier-conditional skill substitution, `bugStatusToBindingKey`
  mapping, out-of-pipeline failure for unmapped BugStatus, unknown-tier hard failure,
  and no-binding failure message.

- **AC-016 (REQ-NF-005):** Given the `bugStatusToBindingKey` function signature, when
  inspected, then it accepts a `BugStatus` value and returns a `string` (the binding
  key). It performs no I/O, imports no I/O packages, and contains no logging calls.

- **AC-017 (regression -- REQ-NF-001):** Given the existing test suite, when all
  tests in `internal/binding/`, `internal/context/`, and `internal/config/` are run,
  then no test that passed before Phase 2 changes regresses.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: parse `.kbz/stage-bindings.yaml`, assert no `retro-fixing` key exists; run `ValidateBindingFile`, assert no error. |
| AC-002 | Test | Automated test: create a mock feature with `Tier=retro_fix` and `Status=developing`, call the binding resolver, assert `skills` list equals `["implement-retro-fix"]`. |
| AC-003 | Test | Automated test: create a mock feature with `Tier=feature` and `Status=developing`, call the binding resolver, assert `skills` list does NOT contain `implement-retro-fix`. |
| AC-004 | Test | Automated test: create a bug with `Status=in-progress`, call `stepResolveStage`, assert resolved binding key is `bug-developing` and its fields pass validation. |
| AC-005 | Test | Automated test: create a bug with `Status=needs-review`, call `stepResolveStage`, assert resolved binding key is `bug-reviewing` and its fields pass validation. |
| AC-006 | Test | Automated unit test: call `bugStatusToBindingKey` with `BugStatusInProgress` and `BugStatusNeedsReview`, assert correct return values. Table-driven test covering all mapped statuses. |
| AC-007 | Test | Automated test: create a bug with `Status=triaged`, call the binding resolver, assert error message contains "triaged" and "out-of-pipeline". |
| AC-008 | Inspection | Compile-time enforcement: removing the fields from the struct and all references guarantees compilation failure if any remain. A `grep` for `\.Profile\b`, `\.Tier\b`, `\.Modes\b`, `\.Verifying\b` on `StageBinding` values confirms zero matches. |
| AC-009 | Inspection | Code review: verify `FastTrackConfig` godoc contains "system of record" language; verify `stage-bindings.yaml` header comment names `FastTrackConfig` as routing owner. |
| AC-010 | Test | Automated test: create a feature with a status that maps to no binding, assert error message contains the status name and an actionable hint. |
| AC-011 | Test | Automated test: create a feature with an invalid tier string, call `Resolve`, assert it returns a typed `ErrUnknownTier` error naming the invalid value. |
| AC-012 | Test | Automated test: create a feature with `Tier=bug_fix` and a status with no `bug-*` binding, assert error message names both the status and tier and says "out-of-pipeline". |
| AC-013 | Inspection | Manual count: post-implementation, enumerate the independently-maintained synchronisation surfaces and verify the count is 7. Compare against the pre-Phase-2 baseline of 10. |
| AC-014 | Inspection | Read `schema_version` from `.kbz/stage-bindings.yaml` and `internal/kbzinit/stage-bindings.yaml`, assert both are `2`. |
| AC-015 | Inspection | Code review: verify the test suite includes test functions for each new code path listed in REQ-NF-004. Run `go test -v` and confirm all named tests exist and pass. |
| AC-016 | Inspection | Code review: inspect the `bugStatusToBindingKey` function body. Verify it contains only a `switch` or `map` lookup, no `log.*` calls, no `os.*` calls, no file or network I/O. |
| AC-017 | Test | Run the full test suite: `go test ./internal/binding/... ./internal/context/... ./internal/config/...`. Assert zero regressions from the pre-Phase-2 baseline. |
