# Specification: Startup Validation Hardening

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-11                     |
| Status | approved |
| Author | spec-author                    |
| Plan   | P64-binding-governance         |
| Tier   | feature                        |

---

## Problem Statement

This specification implements Phase 1 ("Stop the lying") of the design described in
`work/P64-binding-governance/P64-design-binding-governance-implementation-plan.md`
(DOC: P64-binding-governance/design-p64-design-binding-governance-implementation-plan, approved).

The Kanbanzai MCP server has validation code (`ValidateBindingFile`) that exists but is
never invoked in production. As diagnosed in the P64 research report:

- Four production `LoadBindingFile` call sites bypass `BindingRegistry.Load`, so
  `ValidateBindingFile` never runs outside of tests. Silent failures accumulate
  unchallenged (P64 Finding 5).
- The `validStages` allowlist in `internal/binding/model.go` is missing five stages
  that exist in the canonical YAML: `merging`, `verifying`, `batch-reviewing`,
  `doc-publishing`, `retro-fixing`. It still carries the stale `plan-reviewing`
  (P64 Findings 6, 8).
- `workableStatuses` in `internal/context/pipeline.go` has the same omissions and
  the same stale entry (P64 Finding 6).
- `validOrchestrations` only admits `single-agent` and `orchestrator-workers`;
  the `doc-publishing` binding uses `pipeline-coordinator`, which the validator
  would reject (P64 Finding 7).
- The embedded consumer copy (`internal/kbzinit/stage-bindings.yaml`) has drifted
  from the canonical file (`.kbz/stage-bindings.yaml`) — dropping three stages and
  the schema marker — and no CI test detects the drift (P64 Finding 8, P60 §M4).
- The `retro-fixing` binding lacks required fields (`orchestration`, `roles`,
  `skills`) and would fail validation if it were run.

**Scope.** This specification covers the five Phase 1 deliveries from the design:

1. **Loader wiring** — replace production `LoadBindingFile` call sites with
   `BindingRegistry.Load`, which runs `ValidateBindingFile`.
2. **Allowlists** — bring `validStages` and `workableStatuses` into agreement
   with the canonical YAML.
3. **Embedded copy** — CI test for structural equality between canonical and
   embedded `stage-bindings.yaml`; bring the embedded copy in line.
4. **Reachability tests** — test suite that every role/skill reference resolves
   on disk, every `FeatureStatus`/`BugStatus` constant is bound or explicitly
   declared out-of-pipeline, and `ValidateBindingFile` succeeds against the
   canonical file.
5. **Two failing bindings** — give `retro-fixing` minimum passthrough fields;
   add `pipeline-coordinator` to `validOrchestrations`.

**Out of scope.** Phase 1 does NOT implement tier-aware routing, bug-fix pipeline
bindings, the `Resolve` router, the generated registry, or any other Phase 2 or
Phase 3 deliverable from the design document. All current routing behaviour is
preserved; no feature is rerouted.

---

## Requirements

### Functional Requirements

- **REQ-001:** `BindingRegistry.Load` MUST be called at server startup for every
  binding file (project-owned and consumer-owned) instead of bare `LoadBindingFile`.
  On validation failure, the server MUST refuse to start and MUST surface each
  validation error with a fix hint.

- **REQ-002:** The `validStages` allowlist in `internal/binding/model.go` MUST
  contain exactly the stage keys defined in the canonical `stage-bindings.yaml`
  file: `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`,
  `merging`, `verifying`, `batch-reviewing`, `researching`, `documenting`,
  `doc-publishing`, `retro-fixing`. The stale entry `plan-reviewing` MUST be removed.

- **REQ-003:** The `workableStatuses` list in `internal/context/pipeline.go` MUST
  match the set of stage keys in REQ-002. The stale entry `plan-reviewing` MUST be
  removed.

- **REQ-004:** The `validOrchestrations` allowlist in `internal/binding/model.go`
  MUST include `pipeline-coordinator` in addition to `single-agent` and
  `orchestrator-workers`.

- **REQ-005:** The `retro-fixing` binding in `.kbz/stage-bindings.yaml` MUST
  include `orchestration: single-agent`, `roles: [orchestrator]`, and
  `skills: [orchestrate-development]` so that `ValidateBindingFile` succeeds
  against it.

- **REQ-006:** A CI test MUST assert structural equality between
  `.kbz/stage-bindings.yaml` (the canonical file) and
  `internal/kbzinit/stage-bindings.yaml` (the embedded consumer copy). This test
  MUST fail CI if the files differ.

- **REQ-007:** The embedded consumer copy `internal/kbzinit/stage-bindings.yaml`
  MUST be updated to match the canonical file in structure. This includes
  restoring the three missing stages and the `schema_version: 2` marker.

- **REQ-008:** A reachability test suite MUST assert that every `roles` value
  and every `skills` value referenced in `.kbz/stage-bindings.yaml` resolves
  to a file on disk. A role referenced with no corresponding role file MUST
  produce a test failure. A skill on disk unreferenced by any binding MUST
  produce a CI warning (not a failure), with a known-allowlist for
  direct-trigger skills.

- **REQ-009:** A reachability test MUST assert that every `FeatureStatus`
  constant and every `BugStatus` constant defined in the codebase is either
  bound to a stage-binding key or explicitly listed in an
  "out-of-pipeline" declaration within the test file.

- **REQ-010:** A CI test MUST assert that `ValidateBindingFile` succeeds against
  the canonical `.kbz/stage-bindings.yaml` with a `RoleChecker` that resolves
  roles against the `.kbz/roles/` directory.

- **REQ-011:** A new CLI subcommand `kbz binding doctor` MUST run
  `ValidateBindingFile` against the current project's `stage-bindings.yaml` and
  report all errors and warnings without starting the server. It MUST exit
  non-zero if validation errors are found and zero otherwise.

- **REQ-012:** Server startup validation failure messages MUST include a
  pointer to `kbz binding doctor` as a recovery path and a reference to
  `kbz init --upgrade` for unmodified consumer files.

### Non-Functional Requirements

- **REQ-NF-001 (No regressions):** All existing callers of `LoadBindingFile`,
  `BindingRegistry`, and the pipeline's lifecycle validator MUST continue to
  function without behavioural change. No feature is rerouted; no skill is
  added or removed from any binding.

- **REQ-NF-002 (Backward compatibility):** No schema changes to
  `stage-bindings.yaml`, role files, skill files, or any state file format.
  The schema version MUST remain at `2`.

- **REQ-NF-003 (Test coverage):** Each functional requirement REQ-001 through
  REQ-012 MUST ship with at least one automated test that asserts the new
  behaviour.

---

## Constraints

- **Scope exclusion:** This specification does NOT cover tier-aware routing
  (retro-fix wiring through FastTrack, bug-fix pipeline bindings), the
  `Resolve` router, the generated `routing.yaml` registry, or any Phase 2 or
  Phase 3 delivery from the design document. These are deferred to subsequent
  specifications.

- **Scope exclusion:** This specification does NOT cover changes to the
  `Status` tool, the `Handoff` tool, the `Doc` tool, or any MCP tool
  behaviour beyond startup validation. All MCP tool behaviour remains
  identical to the current release.

- **Scope exclusion:** This specification does NOT cover removal of the
  `retro-fixing` YAML block, removal of orphaned `Profile`/`Tier`/`Modes`/
  `Verifying` fields from `StageBinding`, or addition of `bug-developing`/
  `bug-reviewing` bindings. These are Phase 2 work.

- **Backward compatibility:** The `LoadBindingFile` function signature and
  behaviour MUST remain unchanged — it continues to parse YAML without
  validation. Only the call sites change. The `BindingRegistry.Load` method
  MUST remain callable by existing consumers of `BindingRegistry`.

- **Inherited constraint from B69 spec (work/P64-binding-governance/P64-spec-b69-skills-discoverability-quick-patches.md):**
  No schema changes to binding YAML, skill files, role files, or state file
  format. No regressions: existing callers must continue to work.

- **Inherited constraint from Design Decision 3:** Validation failure is
  hard-fail at startup for ALL binding files — project-owned and
  consumer-owned. No two-class validation policy. Mitigation: `kbz binding
  doctor`.

- **Inherited constraint from Design Decision 5:** `doc-publishing` retains
  `pipeline-coordinator` orchestration; the validator admits it via a
  `validOrchestrations` entry. No further orchestration modes are added beyond
  the existing three.

- **Inherited constraint from Design:** The `retro-fixing` passthrough fields
  (`orchestration: single-agent`, `roles: [orchestrator]`, `skills:
  [orchestrate-development]`) are temporary and exist solely to pass
  validation. They do not make `retro-fixing` routable — the binding remains
  unreachable from the pipeline until Phase 2.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a server startup with a valid
  `stage-bindings.yaml`, when `BindingRegistry.Load` is called, then
  the server starts normally with no validation errors.

- **AC-002 (REQ-001):** Given a server startup with an invalid
  `stage-bindings.yaml` (e.g. a stage key not in `validStages`, or a
  missing `roles` field), when `BindingRegistry.Load` is called, then
  the server refuses to start and returns an error message containing the
  specific validation failure and a fix hint referencing `kbz binding doctor`.

- **AC-003 (REQ-002):** Given the current canonical `stage-bindings.yaml`
  (with stages `merging`, `verifying`, `batch-reviewing`, `doc-publishing`,
  `retro-fixing`), when `ValidateBindingFile` runs against it, then no
  "invalid stage name" errors are returned.

- **AC-004 (REQ-002):** Given a `stage-bindings.yaml` containing a
  `plan-reviewing` key, when `ValidateBindingFile` runs against it, then
  an "invalid stage name" error for `plan-reviewing` is returned.

- **AC-005 (REQ-003):** Given a feature with status `merging`, `verifying`,
  `batch-reviewing`, `doc-publishing`, or `retro-fixing`, when
  `stepValidateLifecycle` runs, then the status is accepted as workable
  (no "pipeline requires one of" error).

- **AC-006 (REQ-003):** Given a feature with status `plan-reviewing`, when
  `stepValidateLifecycle` runs, then a "pipeline requires one of" error is
  returned because `plan-reviewing` is not a workable status.

- **AC-007 (REQ-004):** Given a `stage-bindings.yaml` where `doc-publishing`
  has `orchestration: pipeline-coordinator`, when `ValidateBinding`
  runs against it, then no "invalid orchestration" error is returned.

- **AC-008 (REQ-005):** Given the canonical `stage-bindings.yaml`, when
  `ValidateBindingFile` runs against it with a valid `RoleChecker`, then
  no errors are returned for the `retro-fixing` binding (i.e. it has
  non-empty `roles`, non-empty `skills`, and a valid `orchestration`).

- **AC-009 (REQ-006):** Given any commit that changes
  `.kbz/stage-bindings.yaml` without a corresponding update to
  `internal/kbzinit/stage-bindings.yaml`, when the CI test suite runs,
  then the structural equality test fails.

- **AC-010 (REQ-007):** Given the post-Phase-1 state, when the structural
  equality test runs, then `.kbz/stage-bindings.yaml` and
  `internal/kbzinit/stage-bindings.yaml` are structurally identical
  (same top-level keys, same stage keys, same `schema_version`).

- **AC-011 (REQ-008):** Given the canonical `stage-bindings.yaml`, when
  the reachability test suite runs, then every `roles` value resolves
  to a file in `.kbz/roles/` and every `skills` value resolves to a
  file in `.kbz/skills/`.

- **AC-012 (REQ-009):** Given all Go constants in the `FeatureStatus` and
  `BugStatus` types, when the status coverage test runs, then every
  constant is either mapped to a binding key in `stage-bindings.yaml`
  or listed in the test's "out-of-pipeline" declaration.

- **AC-013 (REQ-010):** Given the canonical `stage-bindings.yaml` and
  the `.kbz/roles/` directory, when the validation success test runs,
  then `ValidateBindingFile` returns zero errors.

- **AC-014 (REQ-011):** Given a project with a valid
  `stage-bindings.yaml`, when `kbz binding doctor` is run, then it
  exits with code 0 and reports no errors.

- **AC-015 (REQ-011):** Given a project with an invalid
  `stage-bindings.yaml` (e.g. a stage entry with empty `roles`), when
  `kbz binding doctor` is run, then it exits with a non-zero code and
  reports each validation error with the stage name and the specific
  violation.

- **AC-016 (REQ-012):** Given a server startup validation failure, when
  the error message is surfaced, then it contains the string
  `kbz binding doctor` and a reference to `kbz init --upgrade` for
  unmodified consumer files.

- **AC-017 (REQ-NF-001):** Given the post-Phase-1 codebase, when all
  existing tests in `internal/binding/`, `internal/context/`,
  `internal/mcp/`, and `internal/kbzinit/` are run, then all tests
  that passed before Phase 1 continue to pass.

- **AC-018 (REQ-NF-002):** Given the post-Phase-1 codebase, when
  the canonical and embedded `stage-bindings.yaml` files are inspected,
  then the `schema_version` field is `2` in both files and no new
  top-level keys have been added to either file.

- **AC-019 (REQ-NF-003):** Given the post-Phase-1 codebase, when
  the test suite is inspected, then every functional requirement
  REQ-001 through REQ-012 has at least one corresponding automated
  test function.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: `BindingRegistry.Load` with a fixture YAML that passes all `ValidateBindingFile` checks; assert `Load()` returns `nil`. |
| AC-002 | Test | Automated test: `BindingRegistry.Load` with a fixture YAML containing an invalid stage name (e.g. `plan-reviewing`); assert `Load()` returns a non-nil error containing the invalid stage name and `kbz binding doctor`. |
| AC-003 | Test | Automated test: `ValidateBindingFile` against the canonical `.kbz/stage-bindings.yaml` with `RoleChecker` wired to `.kbz/roles/`; assert `result.Errors` is empty. |
| AC-004 | Test | Automated test: `ValidateBindingFile` against a fixture YAML with a `plan-reviewing` binding key; assert an "invalid stage name" error for `plan-reviewing` in `result.Errors`. |
| AC-005 | Test | Automated test: `stepValidateLifecycle` with feature statuses `merging`, `verifying`, `batch-reviewing`, `doc-publishing`, `retro-fixing`; assert each produces no error. |
| AC-006 | Test | Automated test: `stepValidateLifecycle` with feature status `plan-reviewing`; assert a lifecycle-validation error is returned. |
| AC-007 | Test | Automated test: `ValidateBinding` with `orchestration: pipeline-coordinator`; assert no "invalid orchestration" error. |
| AC-008 | Test | Automated test: `ValidateBindingFile` against the canonical YAML (post-update with passthrough fields); assert no errors for the `retro-fixing` binding. |
| AC-009 | Test | Automated CI test: read both YAML files, unmarshal both, assert `reflect.DeepEqual` on their `StageBindings` keys and `SchemaVersion`. |
| AC-010 | Inspection | Code review: diff between `.kbz/stage-bindings.yaml` and `internal/kbzinit/stage-bindings.yaml` after Phase 1 changes shows structural identity. |
| AC-011 | Test | Automated test: iterate all `roles` and `skills` values from parsed canonical YAML; for each, assert the corresponding file exists under `.kbz/roles/` or `.kbz/skills/`. |
| AC-012 | Test | Automated test: enumerate `FeatureStatus` and `BugStatus` constants via `go/ast` or a hardcoded reference list; assert each is either a key in the canonical YAML's `stage_bindings` or present in a declared out-of-pipeline list. |
| AC-013 | Test | Automated CI test: `ValidateBindingFile(canonicalYAML, realRoleChecker)` returns `len(result.Errors) == 0`. |
| AC-014 | Test | Automated test: invoke `kbz binding doctor` subprocess against a project directory with a valid binding file; assert exit code 0 and stdout contains no error markers. |
| AC-015 | Test | Automated test: invoke `kbz binding doctor` subprocess against a project directory with an intentionally broken binding file; assert exit code non-zero and stderr/stdout names the violating stage and field. |
| AC-016 | Test | Automated test: `BindingRegistry.Load` with a broken fixture; assert the returned error string contains `kbz binding doctor` and `kbz init --upgrade`. |
| AC-017 | Test | Run `go test ./internal/binding/... ./internal/context/... ./internal/mcp/... ./internal/kbzinit/...` against the final Phase 1 branch; assert all tests pass. |
| AC-018 | Inspection | Code review: verify `schema_version` is `2` in both `.kbz/stage-bindings.yaml` and `internal/kbzinit/stage-bindings.yaml`; verify no new top-level keys beyond those present pre-Phase-1. |
| AC-019 | Inspection | Test suite audit: confirm each of REQ-001 through REQ-012 has at least one test function asserting its behaviour. |
