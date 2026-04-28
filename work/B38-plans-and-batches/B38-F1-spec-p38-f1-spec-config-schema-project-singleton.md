# P38-F1: Config Schema and Project Singleton — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T00:43:08Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQK6DDDA                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — D5, D7, D8            |

---

## Overview

This specification defines the `.kbz/config.yaml` schema additions required for P38
(Meta-Planning: Plans and Batches). It implements design decisions D5, D7, and D8 from the
parent design document `work/design/meta-planning-plans-and-batches.md`.

The specification introduces two separate prefix registries — `plan_prefixes` for the new
recursive plan entity and `batch_prefixes` for the renamed batch entity (currently "plan") —
that do not share a namespace and cannot contain the same prefix character. Sequence counters
for plans and batches remain scan-derived but are scoped independently to each registry, so
`P1` (a plan) and `B1` (a batch) can coexist without conflict.

An optional `project` singleton section is also added, holding a human-readable project name,
paths to project-level documents, and a list of constraint strings shown as context when
creating plans or batches. All new fields are optional; existing configurations that do not
use them continue to work unchanged.

---

## Scope

**In scope:**

- Go struct additions: `PlanPrefixes []PrefixEntry`, `BatchPrefixes []PrefixEntry`, and a
  new `ProjectConfig` struct holding `Name`, `Vision`, `Architecture`, and `Constraints`
  fields added to `Config`
- YAML field definitions: `plan_prefixes`, `batch_prefixes`, `project` (all `omitempty`)
- Cross-registry validation in `Config.Validate()`: error if the same prefix character
  appears in both `plan_prefixes` and `batch_prefixes`
- Default value resolution in a new `mergeP38Defaults` (or equivalent) function:
  `plan_prefixes` defaults to `[{P, Plan}]`; `batch_prefixes` defaults to `[{B, Batch}]`
- Unit tests for the new fields, validation rules, defaults, and backward compatibility

**Explicitly excluded:**

- Implementation of plan entity CRUD operations (separate feature in P38)
- Implementation of batch entity CRUD operations (separate feature in P38)
- Migration of existing plan state files to batch state files (separate migration feature)
- Status dashboard rendering of the `project` section (separate feature)
- Document registration of `project.vision` or `project.architecture` paths (separate feature)
- Schema version increment in `schema_version` (coordinated with the broader P38 schema
  migration; not part of this feature)
- Deprecation or removal of the existing `prefixes` field (migration feature)
- Persistence of sequence counters in config (counters remain scan-derived, not stored)
- Validation that `project.vision` or `project.architecture` paths exist on disk

---

## Functional Requirements

- **REQ-001:** The Config schema MUST support a `plan_prefixes` field in `.kbz/config.yaml`.
  When present, it contains a list of `PrefixEntry` records (identical structure to the
  existing `prefixes` field) that designate valid prefixes for plan entity IDs.

- **REQ-002:** The Config schema MUST support a `batch_prefixes` field in `.kbz/config.yaml`.
  When present, it contains a list of `PrefixEntry` records that designate valid prefixes
  for batch entity IDs.

- **REQ-003:** The `plan_prefixes` and `batch_prefixes` registries MUST NOT share a
  namespace. A prefix character MAY appear in at most one of the two registries.

- **REQ-004:** When loading or validating `.kbz/config.yaml`, the system MUST return an
  error if the same prefix character appears in both `plan_prefixes` and
  `batch_prefixes`.

- **REQ-005:** When `plan_prefixes` is absent or empty, the system MUST apply a default plan
  prefix of `P` with name `"Plan"` in-memory. The default MUST NOT be written to disk
  unless explicitly configured by the user.

- **REQ-006:** When `batch_prefixes` is absent or empty, the system MUST apply a default
  batch prefix of `B` with name `"Batch"` in-memory. The default MUST NOT be written to
  disk unless explicitly configured by the user.

- **REQ-007:** Plan sequence numbering MUST be derived independently from batch sequence
  numbering. The sequence number for a new plan is determined by scanning existing plan
  entity IDs against the `plan_prefixes` registry. The sequence number for a new batch is
  determined by scanning existing batch entity IDs against the `batch_prefixes` registry.
  Allocating a new plan number MUST NOT affect the batch counter, and vice versa.

- **REQ-008:** `P1` (a plan entity using plan prefix `P`) and `B1` (a batch entity using
  batch prefix `B`) MUST be assignable simultaneously without conflict. IDs are unique
  within their own entity type and prefix; cross-type collision is impossible by design.

- **REQ-009:** At migration time (when existing plan entities are renamed to batch entities
  and their IDs transition from the `P` prefix to the `B` prefix), the batch counter MUST
  pick up from the highest existing plan number. The plan counter MUST start from 1 after
  migration, as the plan state directory will be empty.

- **REQ-010:** The Config schema MUST support an optional top-level `project` section in
  `.kbz/config.yaml`. The section MAY be absent entirely; its absence MUST NOT cause an
  error.

- **REQ-011:** The `project` section MUST support a `name` field: a human-readable string
  identifying the project, used in dashboard headers and reports. The field is optional.

- **REQ-012:** The `project` section MUST support a `vision` field: a repository-relative
  file path pointing to the project vision document, expected to reside in
  `work/_project/`. The field is optional. No validation of the path's existence is
  performed at config load time.

- **REQ-013:** The `project` section MUST support an `architecture` field: a
  repository-relative file path pointing to the project architecture document, expected to
  reside in `work/_project/`. The field is optional. No validation of the path's existence
  is performed at config load time.

- **REQ-014:** The `project` section MUST support a `constraints` field: a list of short
  human-readable strings representing project-level constraints. These strings are surfaced
  as context when creating plans or batches. The field is optional. An empty or absent list
  is valid.

- **REQ-015:** A project with no `project` section in `.kbz/config.yaml` MUST operate
  identically to how Kanbanzai operates today. No field within the `project` section is
  required. Any combination of fields (including zero) is valid.

---

## Non-Functional Requirements

- **REQ-NF-001:** A `.kbz/config.yaml` containing only a `prefixes` field and no
  `plan_prefixes` or `batch_prefixes` MUST continue to load, validate, and operate without
  error. The existing `prefixes` field behaviour is unchanged.

- **REQ-NF-002:** A `.kbz/config.yaml` without a `project` section MUST continue to load
  and operate without error. All project singleton fields MUST return their zero values
  (empty string, nil slice).

- **REQ-NF-003:** All new fields (`plan_prefixes`, `batch_prefixes`, `project` and its
  sub-fields) MUST be tagged `omitempty` in the YAML struct. A round-trip save of an
  existing configuration that does not use these fields MUST NOT add them to the written
  file.

- **REQ-NF-004:** Both `plan_prefixes` and `batch_prefixes` MUST use the same `PrefixEntry`
  struct as the existing `prefixes` field, including support for `prefix`, `name`,
  `description`, and `retired` subfields. No new struct type is introduced for prefix
  entries.

- **REQ-NF-005:** Default values (plan prefix `P`, batch prefix `B`) MUST be applied during
  the `mergeDefaults` phase at load time, following the same pattern as existing phase
  defaults (e.g. `mergePhase3Defaults`). They MUST NOT be injected at save time.

- **REQ-NF-006:** The new `project.name` field is a separate YAML key nested under
  `project`. The existing top-level `name` field on `Config` remains unchanged. Both MAY be
  present simultaneously during the transition period; they are not aliases.

---

## Constraints

- The existing `prefixes` field is NOT deprecated by this specification. Deprecation and
  migration to `batch_prefixes` are the responsibility of the P38 entity migration feature.
- The `schema_version` field is NOT bumped by this specification. The version increment is
  coordinated with the broader P38 schema migration that also renames the plan entity to
  batch and updates all state file layouts.
- Sequence counters MUST remain scan-derived (computed from existing entity IDs on disk).
  This specification does NOT introduce stored counters in config.
- `PrefixEntry` struct is reused without modification. No new fields are added to it.
- The `work/_project/` folder is NOT created by this feature. It is defined in P37 and is
  expected to already exist or be created independently.
- `project.vision` and `project.architecture` paths are stored as opaque strings. No
  existence validation, document registration, or path canonicalisation is performed at
  config load time.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a config with `plan_prefixes: [{prefix: P, name: Plan}]`,
  `LoadFrom` returns no error and the plan prefix registry contains exactly one entry
  with prefix `P`.

- **AC-002 (REQ-002):** Given a config with `batch_prefixes: [{prefix: B, name: Batch}]`,
  `LoadFrom` returns no error and the batch prefix registry contains exactly one entry
  with prefix `B`.

- **AC-003 (REQ-003, REQ-004):** Given a config with `plan_prefixes: [{prefix: P}]` and
  `batch_prefixes: [{prefix: P}]` (same character in both), `Config.Validate()` returns a
  non-nil error and the error message identifies the duplicate prefix character and names
  both registries.

- **AC-004 (REQ-003):** Given a config with `plan_prefixes: [{prefix: X}]` and
  `batch_prefixes: [{prefix: Y}]` (distinct characters), `Config.Validate()` returns no
  error.

- **AC-005 (REQ-005):** Given a config with no `plan_prefixes` key, after load the
  effective plan prefix is `P` and `SaveTo` does not write `plan_prefixes` to disk.

- **AC-006 (REQ-006):** Given a config with no `batch_prefixes` key, after load the
  effective batch prefix is `B` and `SaveTo` does not write `batch_prefixes` to disk.

- **AC-007 (REQ-007):** Given a project with an existing plan `P3-foo` (plan entity) and
  batch `B3-bar` (batch entity), the plan sequence scanner returns next = 4 (scans plan
  IDs only) and the batch sequence scanner returns next = 4 (scans batch IDs only), both
  allocated independently without conflict.

- **AC-008 (REQ-009):** After a plan-to-batch migration in which `P1`, `P2`, `P3` become
  `B1`, `B2`, `B3`, the batch scanner returns next batch number = 4 and the plan scanner
  returns next plan number = 1 (plan state directory is empty).

- **AC-009 (REQ-010, REQ-011, REQ-012, REQ-013, REQ-014):** Given a config with a fully
  populated `project` section (`name: "My Project"`, `vision: "work/_project/vision.md"`,
  `architecture: "work/_project/arch.md"`, and two constraint strings), `LoadFrom` returns
  no error and all sub-fields are populated with the expected values.

- **AC-010 (REQ-010, REQ-015):** Given a config with `project: {}` (empty section),
  `LoadFrom` returns no error and all `Config.Project` sub-fields return zero values.

- **AC-011 (REQ-010, REQ-015, REQ-NF-003):** Given a config with `project:` absent,
  `LoadFrom` returns no error, `Config.Project` is zero-valued, and `SaveTo` does not
  write a `project` key to disk.

- **AC-012 (REQ-014, REQ-015):** Given a config with only `project.constraints` set (other
  project fields absent), `LoadFrom` returns no error, `Config.Project.Constraints` is
  populated, and other fields are empty strings.

- **AC-013 (REQ-NF-001):** Given an existing v2 config with only
  `prefixes: [{prefix: P, name: Plan}]` and no `plan_prefixes`, `batch_prefixes`, or
  `project` keys, `LoadFrom` returns no error, existing behaviour is fully preserved,
  and `SaveTo` does not add any of the new keys.

- **AC-014 (REQ-NF-004):** `plan_prefixes` and `batch_prefixes` each accept a `retired:
  true` entry. Retired entries are loaded correctly, `IsActivePlanPrefix` returns false
  for a retired plan prefix, and `IsActiveBatchPrefix` returns false for a retired batch
  prefix.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: load config with `plan_prefixes` set, assert `Config.PlanPrefixes` populated correctly |
| AC-002 | Test | Automated test: load config with `batch_prefixes` set, assert `Config.BatchPrefixes` populated correctly |
| AC-003 | Test | Automated test: load config with same prefix in both registries, assert `Validate()` returns error with descriptive message |
| AC-004 | Test | Automated test: load config with distinct prefixes in each registry, assert `Validate()` returns nil |
| AC-005 | Test | Automated test: load config without `plan_prefixes` key, assert `Config.PlanPrefixes` defaults to `[{P, Plan}]`, round-trip save omits key |
| AC-006 | Test | Automated test: load config without `batch_prefixes` key, assert `Config.BatchPrefixes` defaults to `[{B, Batch}]`, round-trip save omits key |
| AC-007 | Test | Automated test: create plan and batch state files with IDs P3/B3, assert plan scanner returns 4 and batch scanner returns 4 independently |
| AC-008 | Test | Automated test: simulate post-migration state (only batch files, empty plan dir), assert batch scanner returns 4 and plan scanner returns 1 |
| AC-009 | Test | Automated test: load config with fully populated `project` section, assert all sub-fields match input |
| AC-010 | Test | Automated test: load config with empty `project: {}`, assert all sub-fields are zero-valued |
| AC-011 | Test | Automated test: load config without `project` key, assert zero-valued, round-trip save omits key |
| AC-012 | Test | Automated test: load config with only `project.constraints`, assert constraints populated, other sub-fields empty |
| AC-013 | Test | Automated test: load legacy config with only `prefixes` field, assert no errors, round-trip has no new keys |
| AC-014 | Test | Automated test: load config with `retired: true` in both registries, assert `IsActivePlanPrefix`/`IsActiveBatchPrefix` return false for retired entries |

---

## Dependencies and Assumptions

- **Design source:** Decisions D5, D7, and D8 in
  `work/design/meta-planning-plans-and-batches.md` are the normative source for the
  requirements in this specification.

- **Existing `PrefixEntry` struct** (`internal/config/config.go`) is reused without
  modification for both `plan_prefixes` and `batch_prefixes`.

- **Sequence counters are scan-derived.** The current `NextPlanNumber` function derives the
  next number by scanning existing plan IDs on disk. Independence between plan and batch
  counters is achieved by scoping each scanner to its respective entity directory
  (`.kbz/state/plans/` for plans, `.kbz/state/batches/` for batches). No stored counter is
  introduced in config by this feature.

- **`project.name` vs top-level `Config.Name`:** The existing `Config.Name` (top-level YAML
  key `name`) was introduced for the `kanbanzai init --name` flag. The new `project.name`
  field is nested under the `project` key. Both fields coexist during the P38 transition
  period. A future cleanup feature may consolidate them.

- **`work/_project/` folder** (defined in P37) is the expected location for project-level
  documents referenced by `project.vision` and `project.architecture`. This feature does not
  create that folder or register the documents — it only stores the paths as strings in
  config.

- **Backward compatibility window:** The existing `prefixes` field is not deprecated by this
  feature. Deprecation and migration to `batch_prefixes` are the responsibility of the P38
  entity migration feature.

- **Schema version:** The `schema_version` field is not bumped by this feature alone. The
  version increment is coordinated with the broader P38 schema migration that also renames
  the plan entity to batch and updates all state file layouts.
