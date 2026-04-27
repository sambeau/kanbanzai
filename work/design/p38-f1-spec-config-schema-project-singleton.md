# P38-F1: Config Schema and Project Singleton — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-27T17:17:18Z                                                     |
| Status  | draft                                                                    |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQK6DDDA                                                       |
| Design  | P38 Meta-Planning: Plans and Batches — D5, D7, D8                        |

---

## Overview

This specification defines the `.kbz/config.yaml` schema additions required for P38
(Meta-Planning: Plans and Batches). It introduces two separate prefix registries —
`plan_prefixes` for the new recursive plan entity and `batch_prefixes` for the renamed batch
entity (currently "plan") — that do not share a namespace and cannot contain the same prefix
character. Sequence counters for plans and batches remain scan-derived but are scoped
independently to each registry, so `P1` (a plan) and `B1` (a batch) can coexist without
conflict. An optional `project` singleton section is also added, holding a human-readable
project name, paths to project-level documents, and a list of constraint strings shown as
context when creating plans or batches. All new fields are optional; existing configurations
that do not use them continue to work unchanged.

---

## Functional Requirements

**FR-001 — `plan_prefixes` registry field**
The Config schema MUST support a `plan_prefixes` field in `.kbz/config.yaml`. When present,
it contains a list of `PrefixEntry` records (identical structure to the existing `prefixes`
field) that designate valid prefixes for plan entity IDs.

**FR-002 — `batch_prefixes` registry field**
The Config schema MUST support a `batch_prefixes` field in `.kbz/config.yaml`. When present,
it contains a list of `PrefixEntry` records that designate valid prefixes for batch entity IDs.

**FR-003 — Separate registry namespaces**
The `plan_prefixes` and `batch_prefixes` registries MUST NOT share a namespace. A prefix
character MAY appear in at most one of the two registries.

**FR-004 — Cross-registry uniqueness validation**
When loading or validating `.kbz/config.yaml`, the system MUST return an error if the same
prefix character appears in both `plan_prefixes` and `batch_prefixes`.

**FR-005 — Default plan prefix**
When `plan_prefixes` is absent or empty, the system MUST apply a default plan prefix of `P`
with name `"Plan"` in-memory. The default MUST NOT be written to disk unless explicitly
configured by the user.

**FR-006 — Default batch prefix**
When `batch_prefixes` is absent or empty, the system MUST apply a default batch prefix of `B`
with name `"Batch"` in-memory. The default MUST NOT be written to disk unless explicitly
configured by the user.

**FR-007 — Independent sequence counters**
Plan sequence numbering MUST be derived independently from batch sequence numbering. The
sequence number for a new plan is determined by scanning existing plan entity IDs against the
`plan_prefixes` registry. The sequence number for a new batch is determined by scanning
existing batch entity IDs against the `batch_prefixes` registry. Allocating a new plan number
MUST NOT affect the batch counter, and vice versa.

**FR-008 — Sequence counter independence across prefix letters**
`P1` (a plan entity using plan prefix `P`) and `B1` (a batch entity using batch prefix `B`)
MUST be assignable simultaneously without conflict. IDs are unique within their own entity
type and prefix; cross-type collision is impossible by design.

**FR-009 — Batch counter migration continuity**
At migration time (when existing plan entities are renamed to batch entities and their IDs
transition from the `P` prefix to the `B` prefix), the batch counter MUST pick up from the
highest existing plan number. The plan counter MUST start from 1 after migration, as the
plan state directory will be empty.

**FR-010 — `project` section in config**
The Config schema MUST support an optional top-level `project` section in
`.kbz/config.yaml`. The section MAY be absent entirely; its absence MUST NOT cause an error.

**FR-011 — `project.name` field**
The `project` section MUST support a `name` field: a human-readable string identifying the
project, used in dashboard headers and reports. The field is optional.

**FR-012 — `project.vision` field**
The `project` section MUST support a `vision` field: a repository-relative file path pointing
to the project vision document, expected to reside in `work/_project/`. The field is optional.
No validation of the path's existence is performed at config load time.

**FR-013 — `project.architecture` field**
The `project` section MUST support an `architecture` field: a repository-relative file path
pointing to the project architecture document, expected to reside in `work/_project/`. The
field is optional. No validation of the path's existence is performed at config load time.

**FR-014 — `project.constraints` field**
The `project` section MUST support a `constraints` field: a list of short human-readable
strings representing project-level constraints. These strings are surfaced as context when
creating plans or batches. The field is optional. An empty or absent list is valid.

**FR-015 — All project fields optional**
A project with no `project` section in `.kbz/config.yaml` MUST operate identically to how
Kanbanzai operates today. No field within the `project` section is required. Any combination
of fields (including zero) is valid.

---

## Non-Functional Requirements

**NFR-001 — Backward compatibility: existing `prefixes` field**
A `.kbz/config.yaml` containing only a `prefixes` field and no `plan_prefixes` or
`batch_prefixes` MUST continue to load, validate, and operate without error. The existing
`prefixes` field behaviour is unchanged.

**NFR-002 — Backward compatibility: absent `project` section**
A `.kbz/config.yaml` without a `project` section MUST continue to load and operate without
error. All project singleton fields MUST return their zero values (empty string, nil slice).

**NFR-003 — No disk bloat for unconfigured fields**
All new fields (`plan_prefixes`, `batch_prefixes`, `project` and its sub-fields) MUST be
tagged `omitempty` in the YAML struct. A round-trip save of an existing configuration that
does not use these fields MUST NOT add them to the written file.

**NFR-004 — PrefixEntry struct reuse**
Both `plan_prefixes` and `batch_prefixes` MUST use the same `PrefixEntry` struct as the
existing `prefixes` field, including support for `prefix`, `name`, `description`, and
`retired` subfields. No new struct type is introduced for prefix entries.

**NFR-005 — In-memory default application**
Default values (plan prefix `P`, batch prefix `B`) MUST be applied during the
`mergeDefaults` phase at load time, following the same pattern as existing phase defaults
(e.g. `mergePhase3Defaults`). They MUST NOT be injected at save time.

**NFR-006 — `project.name` is distinct from top-level `Config.Name`**
The new `project.name` field is a separate YAML key nested under `project`. The existing
top-level `name` field on `Config` remains unchanged. Both MAY be present simultaneously
during the transition period; they are not aliases.

---

## Scope

### In Scope

- Go struct additions: `PlanPrefixes []PrefixEntry`, `BatchPrefixes []PrefixEntry`, and a
  new `ProjectConfig` struct holding `Name`, `Vision`, `Architecture`, and `Constraints`
  fields added to `Config`
- YAML field definitions: `plan_prefixes`, `batch_prefixes`, `project` (all `omitempty`)
- Cross-registry validation in `Config.Validate()`: error if the same prefix character
  appears in both `plan_prefixes` and `batch_prefixes`
- Default value resolution in a new `mergeP38Defaults` (or equivalent) function:
  `plan_prefixes` defaults to `[{P, Plan}]`; `batch_prefixes` defaults to `[{B, Batch}]`
- Unit tests for the new fields, validation rules, defaults, and backward compatibility

### Explicitly Excluded

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

## Acceptance Criteria

**AC-FR-001** — Given a config with `plan_prefixes: [{prefix: P, name: Plan}]`:
- `LoadFrom` returns no error
- The plan prefix registry contains exactly one entry with prefix `P`

**AC-FR-002** — Given a config with `batch_prefixes: [{prefix: B, name: Batch}]`:
- `LoadFrom` returns no error
- The batch prefix registry contains exactly one entry with prefix `B`

**AC-FR-003** — Given a config with `plan_prefixes: [{prefix: P}]` and
`batch_prefixes: [{prefix: P}]` (same character in both):
- `Config.Validate()` returns a non-nil error
- The error message identifies the duplicate prefix character and names both registries

**AC-FR-004** — Given a config with `plan_prefixes: [{prefix: X}]` and
`batch_prefixes: [{prefix: Y}]` (distinct characters):
- `Config.Validate()` returns no error

**AC-FR-005** — Given a config with no `plan_prefixes` key:
- After load, the effective plan prefix is `P`
- `SaveTo` does not write `plan_prefixes` to disk

**AC-FR-006** — Given a config with no `batch_prefixes` key:
- After load, the effective batch prefix is `B`
- `SaveTo` does not write `batch_prefixes` to disk

**AC-FR-007** — Given a project with an existing plan `P3-foo` (plan entity) and batch `B3-bar`
(batch entity):
- The plan sequence scanner returns next = 4 (scans plan IDs only)
- The batch sequence scanner returns next = 4 (scans batch IDs only)
- Both numbers are allocated independently without conflict

**AC-FR-008** — After a plan→batch migration in which `P1`, `P2`, `P3` become `B1`, `B2`,
`B3`:
- The batch scanner returns next batch number = 4
- The plan state directory is empty; the plan scanner returns next plan number = 1

**AC-FR-009** — Given a config with a fully populated `project` section:
```yaml
project:
  name: "My Project"
  vision: "work/_project/vision.md"
  architecture: "work/_project/arch.md"
  constraints:
    - "Must support 100k users"
    - "All services independently deployable"
```
- `LoadFrom` returns no error
- `Config.Project.Name` == `"My Project"`
- `Config.Project.Vision` == `"work/_project/vision.md"`
- `Config.Project.Architecture` == `"work/_project/arch.md"`
- `Config.Project.Constraints` has length 2

**AC-FR-010** — Given a config with `project: {}` (empty section):
- `LoadFrom` returns no error
- All `Config.Project` sub-fields return zero values

**AC-FR-011** — Given a config with `project:` absent:
- `LoadFrom` returns no error
- `Config.Project` is zero-valued
- `SaveTo` does not write a `project` key to disk

**AC-FR-012** — Given a config with only `project.constraints` set (other project fields
absent):
- `LoadFrom` returns no error
- `Config.Project.Constraints` is populated; other fields are empty strings

**AC-NFR-001** — Given an existing v2 config with only `prefixes: [{prefix: P, name: Plan}]`
and no `plan_prefixes`, `batch_prefixes`, or `project` keys:
- `LoadFrom` returns no error
- Existing behaviour is fully preserved
- `SaveTo` does not add any of the new keys

**AC-NFR-002** — `plan_prefixes` and `batch_prefixes` each accept a `retired: true` entry:
- Retired entries are loaded correctly
- `IsActivePlanPrefix` returns false for a retired plan prefix
- `IsActiveBatchPrefix` returns false for a retired batch prefix

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
```

Now let me write that file and register it: