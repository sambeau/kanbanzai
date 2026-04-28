# P38-F3: Batch Entity Rename and B-Prefix IDs — Specification

| Field   | Value |
|---------|-------|
| Date    | 2026-04-27 |
| Status  | draft |
| Feature | FEAT-01KQ7YQKEEHFY |
| Design  | P38 Meta-Planning: Plans and Batches — §3, D3, D4, D7, D8 |

---

## Related Work

- **Design:** `work/design/meta-planning-plans-and-batches.md` — §3 (batch entity), D3, D4, D7, D8
- **P38-F1:** `work/design/p38-f1-spec-config-schema-project-singleton.md` — defines `batch_prefixes` registry, default `B` prefix, independent sequence counter
- **P38-F2:** `work/design/p38-f2-spec-plan-entity-data-model-lifecycle.md` — defines the new plan entity that inherits the `plan` kind and `.kbz/state/plans/` directory
- **P38-F7:** Physical file migration — moves existing `P{n}-*` files from `plans/` to `batches/`; on hold, not in scope here
- **Current model:** `internal/model/entities.go` — `Plan` struct, `EntityKindPlan`, `PlanStatus`, `IsPlanID`, `ParsePlanID`
- **Current service:** `internal/service/plans.go` — `CreatePlan`, `GetPlan`, `ListPlans`, `UpdatePlan`, `UpdatePlanStatus`, related helpers
- **Current storage:** `internal/storage/entity_store.go` — `entityDirectory`, `entityFileName`, `fieldOrderForEntityType`
- **Current MCP tools:** `internal/mcp/entity_tool.go`, `status_tool.go`, `next_tool.go`
- **Current lifecycle:** `internal/validate/lifecycle.go` — plan lifecycle transitions
- **Current health:** `internal/health/entity_consistency.go` — `CheckPlanChildConsistency`
- **ID utilities:** `internal/id/allocator.go`, `internal/id/display.go`
- **Docint extractor:** `internal/docint/extractor.go` — `planIDPattern` regex

---

## Overview

This feature renames the current `plan` entity to `batch` at the code level. The batch entity
is the operational work-grouping unit in Kanbanzai — the entity that contains features,
governs their lifecycle, and aggregates their progress. It retains all current plan
functionality unchanged.

New batches created after this feature is in production receive `B{n}-{slug}` IDs using the
`batch_prefixes` registry and an independent sequence counter (both introduced by F1).
Existing plan state files at `.kbz/state/plans/P{n}-*.yaml` remain in place; the system
falls back to that directory when a batch is not found in `.kbz/state/batches/`, emitting a
deprecation warning. F7 (on hold) will perform the physical migration.

This feature is a code-level rename only. It does not touch `.kbz/state/plans/` files, does
not migrate existing data, and does not change any feature or task entity behaviour.

**Relationship to F2:** F2 creates the new recursive plan entity, which takes the `plan` kind
and continues to use `.kbz/state/plans/`. F3 creates the batch entity, which takes the new
`batch` kind and uses `.kbz/state/batches/`. After both features are deployed:

- `entity(type: "plan", ...)` → new recursive plan entity (F2)
- `entity(type: "batch", ...)` → operational work-grouping entity (this feature)
- A P-prefix ID passed to a batch operation → fallback to `plans/` with deprecation warning

---

## Functional Requirements

### FR-001 — `EntityKindBatch` constant

The `internal/model` package MUST define a new exported constant:

```kanbanzai/internal/model/entities.go
EntityKindBatch EntityKind = "batch"
```

`EntityKindPlan` (`"plan"`) is retained unchanged; it is used by the new plan entity (F2).

---

### FR-002 — `Batch` model struct

The `internal/model` package MUST define a `Batch` struct with the following fields:

```kanbanzai/internal/model/entities.go
type Batch struct {
    ID        string     `yaml:"id"`
    Slug      string     `yaml:"slug"`
    Name      string     `yaml:"name"`
    Status    PlanStatus `yaml:"status"`
    Summary   string     `yaml:"summary"`
    Parent    string     `yaml:"parent,omitempty"`   // optional parent plan ID (F2 plan entity)
    Design    string     `yaml:"design,omitempty"`
    Tags      []string   `yaml:"tags,omitempty"`
    Created   time.Time  `yaml:"created"`
    CreatedBy string     `yaml:"created_by"`
    Updated   time.Time  `yaml:"updated"`
    NextFeatureSeq int   `yaml:"next_feature_seq,omitempty"` // per-batch feature display ID counter (P37)
    Supersedes   string  `yaml:"supersedes,omitempty"`
    SupersededBy string  `yaml:"superseded_by,omitempty"`
}
```

`Batch.GetKind()` MUST return `EntityKindBatch`.

`PlanStatus` is reused for the batch lifecycle (see FR-005). The existing `Plan` struct is
retained unchanged for use by F2.

---

### FR-003 — `IsBatchID` and `ParseBatchID` functions

The `internal/model` package MUST expose:

```kanbanzai/internal/model/entities.go
// IsBatchID returns true if the given ID matches the batch ID pattern.
// Batch IDs have the same structural format as plan IDs: {X}{n}-{slug}.
// Both B-prefix IDs (new batches) and P-prefix IDs (legacy plan-era batches)
// match this function.
func IsBatchID(id string) bool

// ParseBatchID extracts the prefix, number, and slug from a batch ID.
// Returns empty strings if the ID is not a valid batch ID.
func ParseBatchID(id string) (prefix, number, slug string)
```

Both functions MUST use the same structural logic as `IsPlanID` and `ParsePlanID`. Since the
ID format is identical (`{letter}{digits}-{slug}`), `IsBatchID(id)` is equivalent to
`IsPlanID(id)`. Both are retained and named precisely to reflect intended use.

`IsPlanID` and `ParsePlanID` are retained unchanged for use by the F2 plan entity and
backward-compatibility code.

---

### FR-004 — Batch YAML field order

The storage layer (`internal/storage/entity_store.go`) MUST define a canonical YAML field
order for `"batch"` entities:

```
id, slug, name, status, summary, parent, design, next_feature_seq, tags,
created, created_by, updated, supersedes, superseded_by
```

Fields not present in the state map are omitted (as with all entity types). The `parent`
field, when absent, is omitted rather than serialised as an empty string.

---

### FR-005 — Batch lifecycle

The batch lifecycle is identical to the current plan lifecycle:

```
proposed → designing → active → reviewing → done
```

Terminal states: `superseded`, `cancelled` (reachable from any non-terminal state).

Valid transitions:

| From        | To (allowed)                                         |
|-------------|------------------------------------------------------|
| `proposed`  | `designing`, `active`*, `superseded`, `cancelled`    |
| `designing` | `active`, `superseded`, `cancelled`                  |
| `active`    | `reviewing`, `superseded`, `cancelled`               |
| `reviewing` | `done`, `active`, `superseded`, `cancelled`          |
| `done`      | `superseded`, `cancelled`                            |

*`proposed → active` shortcut requires at least one feature in a post-designing state; the
service layer enforces this and appends a system-generated override record.

The `internal/validate/lifecycle.go` module MUST register `EntityKindBatch` with this
transition table and entry state `"proposed"`.

`BatchStatus` type aliases are NOT introduced. `PlanStatus` constants (`PlanStatusProposed`,
`PlanStatusActive`, etc.) are reused for both batch and plan lifecycle values, since they
share the same state strings. Code that dispatches on `EntityKindBatch` uses `PlanStatus`
values for state comparisons.

---

### FR-006 — Batch storage directory

New batch entities MUST be written to `.kbz/state/batches/{id}.yaml`.

The `entityDirectory` function in `internal/storage/entity_store.go` MUST map entity type
`"batch"` to the directory name `"batches"`.

Batch files MUST use the `{id}.yaml` filename convention (not `{id}-{slug}.yaml`), consistent
with the current plan file convention. The `entityFileName` function MUST be updated to apply
this rule for entity type `"batch"`.

---

### FR-007 — Batch ID format for new batches

New batches allocated by `CreateBatch` MUST use the `batch_prefixes` registry (FR-002 of
P38-F1). The default batch prefix when the registry is absent or empty is `B` (FR-006 of
P38-F1).

The resulting ID has the form `B{n}-{slug}` (e.g. `B24-auth-system`), where `{n}` is the
next available number determined by scanning existing batch IDs in `.kbz/state/batches/`
using the `NextBatchNumber` function (the batch-scoped analogue of `NextPlanNumber`, added as
part of this feature).

`NextBatchNumber` in `internal/config/config.go` MUST operate against the `batch_prefixes`
registry and accept a scanner function that lists existing batch IDs (from the `batches/`
directory only — not `plans/`). It MUST NOT share state with `NextPlanNumber`.

---

### FR-008 — Backward-compatible loading: fallback to `plans/`

When the system attempts to load a batch entity and the file is not found in
`.kbz/state/batches/`, it MUST attempt to load the file from `.kbz/state/plans/` as a
fallback. This fallback MUST apply when:

- A P-prefix ID is provided directly (e.g. `GetBatch("P1-auth-system")`)
- A B-prefix ID is provided but the file has not yet been migrated (e.g. due to a rename)

When the fallback path is used, the service layer MUST include a `deprecation_warning` field
in the response:

```
"deprecation_warning": "Batch P1-auth-system was loaded from the legacy plans/ directory. Run F7 migration to move it to batches/."
```

This warning MUST be surfaced in all tool responses that trigger the fallback (create not
applicable; get, list, update, transition all apply).

---

### FR-009 — Backward-compatible listing: scan both directories

`ListBatches` MUST scan both `.kbz/state/batches/` and `.kbz/state/plans/` for YAML files
matching the batch ID format. Files found in `batches/` take precedence; if a file with the
same ID appears in both directories (which should not happen in normal operation), the
`batches/` version is used.

Each result sourced from the `plans/` directory MUST include the `deprecation_warning` field
described in FR-008.

---

### FR-010 — `CreateBatch` service method

The `internal/service` package MUST expose `CreateBatch(CreateBatchInput) (CreateResult, error)`.

`CreateBatchInput` fields:

| Field       | Type     | Required | Notes |
|-------------|----------|----------|-------|
| `Prefix`    | string   | yes      | Must be declared in `batch_prefixes` registry |
| `Slug`      | string   | yes      | URL-safe lowercase; normalised on write |
| `Name`      | string   | yes      | Human-readable display name |
| `Summary`   | string   | yes      | Brief description |
| `CreatedBy` | string   | yes      | Identity of creator |
| `Parent`    | string   | no       | Optional parent plan ID (F2 plan entity) |
| `Tags`      | []string | no       | Freeform lowercase tags |

`CreateBatch` MUST:
1. Validate all required fields are non-empty.
2. Validate the prefix against `batch_prefixes`; reject retired prefixes.
3. Allocate the next available batch number via `NextBatchNumber`.
4. If `Parent` is non-empty, validate it refers to an existing plan entity (type `"plan"`,
   in `.kbz/state/plans/`). If the referenced entity is a batch, return an error: "parent
   must reference a plan entity, not a batch".
5. Set initial status to `"proposed"`.
6. Write the new entity to `.kbz/state/batches/{id}.yaml`.
7. Update the entity cache.

---

### FR-011 — `GetBatch` service method

The `internal/service` package MUST expose `GetBatch(id string) (ListResult, error)`.

`GetBatch` MUST:
1. Validate that `id` is non-empty and matches `IsBatchID`.
2. Attempt to load from `.kbz/state/batches/{id}.yaml`.
3. If not found, fall back to `.kbz/state/plans/{id}.yaml` and attach a deprecation warning
   (FR-008).
4. Return `ErrNotFound` if neither path yields a file.

---

### FR-012 — `ListBatches` service method

The `internal/service` package MUST expose
`ListBatches(filters BatchFilters) ([]ListResult, error)`.

`BatchFilters` supports the same fields as the current `PlanFilters`: `Status`, `Prefix`,
`Tags`.

Results from both directories are merged (FR-009). Results are not sorted by the service
layer; callers may sort as needed.

---

### FR-013 — `UpdateBatch` and `UpdateBatchStatus` service methods

The `internal/service` package MUST expose:

- `UpdateBatch(UpdateBatchInput) (ListResult, error)` — updates mutable fields (`name`,
  `summary`, `design`, `tags`).
- `UpdateBatchStatus(id, slug, newStatus string) (ListResult, error)` — transitions the
  batch lifecycle.

Both methods MUST:
1. Load the existing batch via `GetBatch` (applying the fallback).
2. Apply the update.
3. Write back to the same path from which the entity was loaded (preserving location until
   F7 migration).
4. Attach a deprecation warning if the entity was sourced from `plans/`.

`UpdateBatchStatus` MUST enforce:
- Valid lifecycle transitions via `validate.ValidateTransition(EntityKindBatch, ...)`.
- The `proposed → active` shortcut precondition (FR-005).
- Terminal transition guard: reject transitions to `done`, `cancelled`, or `superseded` when
  non-terminal features exist, unless `override: true` is set.

---

### FR-014 — `MaybeAutoAdvanceBatch` service method

The `internal/service` package MUST expose
`MaybeAutoAdvanceBatch(batchID string) (advanced bool, err error)`.

This method replaces `MaybeAutoAdvancePlan` for the batch entity. When the last non-terminal
feature of a batch reaches a terminal state, this method MUST:

1. Load the batch via `GetBatch`.
2. Check whether the batch is in `active` status.
3. List all features whose `parent` field equals `batchID`.
4. If all features are in terminal states and at least one is `done`: transition the batch
   `active → reviewing → done` (chaining both transitions atomically within the method).
5. Return `advanced = true` if the transition was performed.

`MaybeAutoAdvancePlan` is retained for the new plan entity (F2) and MUST NOT be removed.

---

### FR-015 — Entity tool: `type: "batch"` support

The `entity` MCP tool MUST handle `type: "batch"` for all actions:

- `create`: call `CreateBatch`; the `parent` argument is accepted (optional plan ID).
- `get`: call `GetBatch`.
- `list`: call `ListBatches`.
- `update`: call `UpdateBatch`.
- `transition`: call `UpdateBatchStatus`, enforcing the terminal transition guard and
  supporting the `override` parameter.

The `entity` tool schema MUST document `"batch"` as a valid entity type in the `type` field
enumeration.

---

### FR-016 — Entity tool: `type: "plan"` backward compatibility for batches

During the transition period (before F7 migration), the `entity` tool MUST continue to route
`type: "plan"` operations to the new plan service (F2). It MUST NOT silently re-route
`type: "plan"` to batch operations. Users who want to operate on legacy plan-era batches
MUST use `type: "batch"` with a P-prefix ID.

---

### FR-017 — Entity type inference from ID

`entityInferType` in `internal/mcp/entity_tool.go` and `nextInferEntityType` in
`internal/mcp/next_tool.go` MUST be updated:

- If `IsBatchID(id)` is true AND the entity is found in `.kbz/state/batches/`, infer type
  `"batch"`.
- If `IsBatchID(id)` is true AND the entity is found in `.kbz/state/plans/` (not in
  `batches/`), infer type `"batch"` (with deprecation warning on subsequent load).
- `IsPlanID` continues to be used for plan entity inference; after F2 and F3 are deployed,
  only entities in `.kbz/state/plans/` that match the F2 plan schema are inferred as `"plan"`.

For `inferIDType` in `internal/mcp/status_tool.go`, a new `idTypeBatch` constant MUST be
added alongside the existing `idTypePlan`. B-prefix IDs MUST map to `idTypeBatch`. P-prefix
IDs MUST be probed: if the file is found in `batches/`, route to `idTypeBatch`; if in
`plans/`, route to `idTypePlan` (for new plan entities) or `idTypeBatch` (for legacy plans
that haven't been migrated). During the transition period, P-prefix ID routing SHOULD
attempt `batches/` first, then `plans/`.

---

### FR-018 — Status tool: batch dashboard

`synthesisePlan` in `internal/mcp/status_tool.go` MUST be complemented by
`synthesiseBatch(batchID string, ...) (*batchDashboard, error)`.

The batch dashboard has the same structure as the current plan dashboard
(`planDashboard` / `planHeader`), updated to use "batch" terminology in JSON field names:

```json
{
  "scope": "batch",
  "batch": { "display_id": "B24-auth-system", "id": "...", "slug": "...", "name": "...", "status": "..." },
  "features": [...],
  "doc_gaps": [...],
  "health": {...},
  "attention": [...],
  "generated_at": "..."
}
```

`synthesisePlan` continues to exist and renders the F2 plan entity dashboard.

---

### FR-019 — Side effect: `batch_auto_advanced`

`internal/mcp/sideeffect.go` MUST define:

```kanbanzai/internal/mcp/sideeffect.go
SideEffectBatchAutoAdvanced SideEffectType = "batch_auto_advanced"
```

This side effect is emitted when `MaybeAutoAdvanceBatch` transitions a batch to `done`.

`SideEffectPlanAutoAdvanced` is retained for the new plan entity (F2).

---

### FR-020 — Feature parent validation for batches

`CreateFeature` in `internal/service/entities.go` MUST validate the `parent` field against
both the batch entity type and, during the transition period, legacy plan files:

1. Check whether `parent` matches `IsBatchID`.
2. If so, verify the entity exists by checking `batches/` and then falling back to `plans/`.
3. Accept the parent as valid if found in either location.
4. If the `parent` field matches `IsPlanID` but the ID does not resolve to any known entity,
   return `ErrReferenceNotFound`.

Existing features with `parent: P{n}-slug` continue to work because P-prefix IDs satisfy
`IsBatchID` and resolve via the fallback.

---

### FR-021 — `entityExists` for batch type

`entityExists` in `internal/service/entities.go` MUST handle entity type `"batch"`:

1. Check for `{id}.yaml` in `.kbz/state/batches/`.
2. If not found, check for `{id}.yaml` in `.kbz/state/plans/` (fallback for P-prefix legacy
   files during the transition period).
3. Return `true` if found in either location.

---

### FR-022 — Validate layer: `EntityBatch` constant

`internal/validate/lifecycle.go` MUST define:

```kanbanzai/internal/validate/lifecycle.go
EntityBatch = model.EntityKindBatch
```

The `entryStates`, `terminalStates`, and `allowedTransitions` maps MUST include
`EntityBatch` with the transition table from FR-005 (identical to the current `EntityPlan`
entries).

`internal/validate/entity.go` MUST include `EntityBatch` in `requiredFields`:

```
EntityBatch: {"id", "slug", "name", "status", "summary", "created", "created_by"}
```

The `parent` field is optional and is NOT included in required fields.

---

### FR-023 — Health check: batch-child consistency

`internal/health/entity_consistency.go` MUST expose
`CheckBatchChildConsistency(batches, features []map[string]any) CategoryResult`.

This function has the same logic as `CheckPlanChildConsistency` but operates on batch
entities. It checks:

1. Warning: All child features are finished but the batch is not `done`.
2. Warning: Batch is `done` but has non-finished child features.

`CheckPlanChildConsistency` is retained for the new plan entity (F2).

`internal/health/check.go` MUST invoke `CheckBatchChildConsistency` and include its result
under the category key `"batch_child_consistency"`.

---

### FR-024 — ID allocator: batch kind support

`internal/id/allocator.go` `Validate` MUST handle `model.EntityKindBatch`:

```kanbanzai/internal/id/allocator.go
if entityKind == model.EntityKindBatch {
    if !model.IsBatchID(id) {
        return fmt.Errorf("invalid batch ID %q: must match {prefix}{number}-{slug} format", id)
    }
    return nil
}
```

---

### FR-025 — Display ID utilities: batch pass-through

`internal/id/display.go` functions `FormatFullDisplay`, `FormatShortDisplay`, and
`StripBreakHyphens` MUST treat batch IDs the same way they treat plan IDs: pass through
unchanged without adding break hyphens or uppercasing the slug. The `IsBatchID` check MUST
be added alongside the existing `IsPlanID` check in each function.

---

### FR-026 — Docint extractor: batch ID recognition

The `planIDPattern` regex in `internal/docint/extractor.go` already matches all IDs of the
form `[A-Z]\d+-[a-z][-a-z0-9]+`, which includes B-prefix batch IDs. No regex change is
required. However, matched entities MUST be classified as:

- `"batch"` when the first character is the batch prefix (`B` by default, or any character
  in `batch_prefixes`).
- `"plan"` when the first character is the plan prefix (`P` by default, or any character in
  `plan_prefixes`).

If the distinction cannot be made statically (e.g. both registries are not loaded in the
docint context), the type `"plan"` is used as the default classification for all
`{letter}{digits}-{slug}` patterns, preserving existing behaviour. This is acceptable during
the transition period.

---

### FR-027 — `kbzschema` public package: Batch type and constants

`kbzschema/types.go` MUST define:

```kanbanzai/kbzschema/types.go
// BatchStatus constants (reuse same string values as PlanStatus).
const (
    BatchStatusProposed   = "proposed"
    BatchStatusDesigning  = "designing"
    BatchStatusActive     = "active"
    BatchStatusReviewing  = "reviewing"
    BatchStatusDone       = "done"
    BatchStatusSuperseded = "superseded"
    BatchStatusCancelled  = "cancelled"
)

type Batch struct {
    ID             string   `yaml:"id"`
    Slug           string   `yaml:"slug"`
    Name           string   `yaml:"name"`
    Status         string   `yaml:"status"`
    Summary        string   `yaml:"summary"`
    Parent         string   `yaml:"parent,omitempty"`
    Design         string   `yaml:"design,omitempty"`
    NextFeatureSeq int      `yaml:"next_feature_seq,omitempty"`
    Tags           []string `yaml:"tags,omitempty"`
    Created        string   `yaml:"created"`
    CreatedBy      string   `yaml:"created_by"`
    Updated        string   `yaml:"updated"`
    Supersedes     string   `yaml:"supersedes,omitempty"`
    SupersededBy   string   `yaml:"superseded_by,omitempty"`
}
```

---

### FR-028 — `kbzschema` Reader: `GetBatch` and `ListBatches`

`kbzschema/reader.go` MUST expose:

- `GetBatch(id string) (Batch, error)` — reads from `batches/{id}.yaml`; falls back to
  `plans/{id}.yaml` if not found.
- `ListBatches() ([]Batch, error)` — scans both `batches/` and `plans/` (deduplicating on
  ID, preferring `batches/`).

The existing `GetPlan` and `ListPlans` methods are retained for the F2 plan entity.

---

### FR-029 — CLI: `entity create batch` command

`cmd/kanbanzai/entity_cmd.go` MUST handle `entity create batch` by mapping to
`CreateBatch`. Accepted flags: `--prefix`, `--slug`, `--name`, `--summary`,
`--created_by`, `--parent`, `--tags`.

`entity get batch`, `entity list batches` (or `entity list batch`), and
`entity transition batch` MUST be handled analogously.

The `entityService` interface in `cmd/kanbanzai/main.go` MUST add:

```kanbanzai/cmd/kanbanzai/main.go
CreateBatch(service.CreateBatchInput) (service.CreateResult, error)
GetBatch(id string) (service.ListResult, error)
ListBatches(filters service.BatchFilters) ([]service.ListResult, error)
UpdateBatch(service.UpdateBatchInput) (service.ListResult, error)
UpdateBatchStatus(id, slug, newStatus string) (service.ListResult, error)
```

---

### FR-030 — `batch` accepted in type-polymorphic paths

Every location in the codebase that accepts an entity type string via a `switch` or `if`
block and has a `"plan"` case MUST add a `"batch"` case routing to the corresponding batch
method. Specific call sites:

| File | Function | Current case | Required addition |
|------|----------|--------------|-------------------|
| `internal/service/entities.go` | `validateKindForType` | `"plan"` → `validate.EntityPlan` | `"batch"` → `validate.EntityBatch` |
| `internal/service/entities.go` | `Get` | routes to `GetPlan` | add route to `GetBatch` |
| `internal/service/entities.go` | `List` | routes to `ListPlans` | add route to `ListBatches` |
| `internal/service/entities.go` | `HealthCheck` | checks plan via file | add analogous batch file check |
| `internal/service/entity_lifecycle_hook.go` | `TransitionStatus` | `"plan"` case | add `"batch"` case |
| `internal/service/entities.go` | `parseRecordIdentity` | `"plan"` case | add `"batch"` case |
| `cmd/kanbanzai/estimate_cmd.go` | `runEstimate` | tries `"plan"` | add `"batch"` to tried types |
| `internal/mcp/entity_tool.go` | `entityKindFromType` | `"plan"` | add `"batch"` |
| `internal/mcp/entity_tool.go` | `entityCurrentStatus` | `"plan"` | add `"batch"` |
| `internal/mcp/entity_tool.go` | `entityDuplicateAdvisory` | `"plan"` | add `"batch"` |

---

## Non-Functional Requirements

**NFR-001 — No data loss:** This feature MUST NOT delete, overwrite, or modify any existing
`.kbz/state/plans/` files. All writes go to `.kbz/state/batches/`.

**NFR-002 — Backward compatibility:** All existing workflows using plan entities via the MCP
`entity` tool continue to function without modification. P-prefix IDs passed to batch
operations resolve via the `plans/` fallback. Features with `parent: P{n}-slug` continue to
work.

**NFR-003 — Deprecation warning format:** The deprecation warning MUST be a stable string
field (`"deprecation_warning"`) in the JSON response object, not a log line or error. Tools
that embed the response in a larger object MUST propagate the warning to the outer response.

**NFR-004 — No renamed exported symbols in `kbzschema`:** The public `kbzschema` package
MUST add new symbols (`Batch`, `BatchStatus*`, `GetBatch`, `ListBatches`) without removing
`Plan`, `PlanStatus*`, `GetPlan`, or `ListPlans`. Downstream consumers of `kbzschema` MUST
NOT require a breaking change.

**NFR-005 — `PlanStatus` constants reused without aliasing:** `BatchStatus*` constants in
`kbzschema` are new string constants with identical values to their `PlanStatus*`
counterparts. Inside the internal packages, `PlanStatus` values are used directly for batch
lifecycle state comparisons.

---

## Scope

### In scope

- `internal/model/entities.go`: `EntityKindBatch`, `Batch` struct, `IsBatchID`, `ParseBatchID`
- `internal/storage/entity_store.go`: `"batch"` directory and filename handling, field order
- `internal/service/plans.go`: new batch service methods (CreateBatch, GetBatch, ListBatches,
  UpdateBatch, UpdateBatchStatus, writeBatch, loadBatch, listBatchIDs, NextBatchNumber, and
  all helpers)
- `internal/service/entities.go`: CreateFeature parent validation, entityExists, Get, List,
  validateKindForType, parseRecordIdentity, HealthCheck
- `internal/service/entity_children.go`: MaybeAutoAdvanceBatch
- `internal/service/entity_lifecycle_hook.go`: "batch" case in TransitionStatus
- `internal/mcp/entity_tool.go`: "batch" case in create/get/update/transition/infer/advisory
- `internal/mcp/status_tool.go`: idTypeBatch, synthesiseBatch, batchDashboard, batchHeader,
  generateBatchAttention
- `internal/mcp/next_tool.go`: "batch" inference from IsBatchID
- `internal/mcp/sideeffect.go`: SideEffectBatchAutoAdvanced
- `internal/validate/lifecycle.go`: EntityBatch, transitions, entry state
- `internal/validate/entity.go`: requiredFields for EntityBatch
- `internal/validate/health.go`: batch parent reference checks
- `internal/health/entity_consistency.go`: CheckBatchChildConsistency
- `internal/health/check.go`: invoke CheckBatchChildConsistency
- `internal/id/allocator.go`: EntityKindBatch validation
- `internal/id/display.go`: IsBatchID pass-through in FormatFullDisplay, FormatShortDisplay, StripBreakHyphens
- `internal/config/config.go`: NextBatchNumber using batch_prefixes registry
- `kbzschema/types.go`: Batch struct, BatchStatus constants
- `kbzschema/reader.go`: GetBatch, ListBatches with plans/ fallback
- `cmd/kanbanzai/entity_cmd.go`: create/get/list/transition for "batch"
- `cmd/kanbanzai/main.go`: entityService interface additions
- `cmd/kanbanzai/estimate_cmd.go`: try "batch" type in fallback chain
- All associated test files for the above

### Out of scope

- Physical migration of `.kbz/state/plans/P{n}-*` files to `batches/` (F7, on hold)
- Per-batch feature display IDs (`B24-F3` format) — counter mechanics covered by P37-F1
- Recursive plan entity (F2)
- Project singleton (F1)
- Recursive rollup through the plan tree (F5)
- Removal of `MaybeAutoAdvancePlan`, `GetPlan`, `ListPlans`, `PlanStatus`, `IsPlanID`,
  `ParsePlanID`, or any other existing plan symbols (retained for F2 plan entity use)
- Removal of `kbzschema.Plan`, `kbzschema.PlanStatus*`, or `kbzschema.ListPlans` (retained
  for backward-compatible public API)
- Changing the feature, task, bug, decision, or incident entity models

---

## Acceptance Criteria

**AC-001:** `entity(action: "create", type: "batch", prefix: "B", slug: "auth-system", name:
"Auth System", summary: "...", created_by: "user")` returns a result with `id` matching
`/^B\d+-auth-system$/` and writes a file at `.kbz/state/batches/B{n}-auth-system.yaml`.

**AC-002:** The YAML file written by AC-001 contains fields in the order: `id`, `slug`,
`name`, `status`, `summary`, `created`, `created_by`, `updated` (optional fields omitted
when empty).

**AC-003:** `entity(action: "create", type: "batch", ...)` with `parent: "P1-social-platform"`
succeeds when `P1-social-platform` is a valid plan entity in `.kbz/state/plans/`. It fails
with a clear error if `P1-social-platform` is itself a batch entity.

**AC-004:** `entity(action: "get", type: "batch", id: "P1-legacy")` with `P1-legacy.yaml`
absent from `batches/` but present in `plans/` returns the entity with a
`deprecation_warning` field in the response.

**AC-005:** `entity(action: "get", type: "batch", id: "P1-legacy")` with `P1-legacy.yaml`
absent from both `batches/` and `plans/` returns a not-found error.

**AC-006:** `entity(action: "list", type: "batch")` returns entities from both `batches/`
and `plans/` directories. Entities sourced from `plans/` include a `deprecation_warning`.
No entity appears twice.

**AC-007:** `entity(action: "transition", type: "batch", id: "B1-auth-system", status:
"designing")` transitions the batch from `proposed` to `designing` and writes the updated
file back to `batches/`.

**AC-008:** `entity(action: "transition", type: "batch", id: "B1-auth-system", status:
"done")` when non-terminal features exist (and `override: false`) returns an error describing
the blocking features.

**AC-009:** After the last non-terminal feature of a batch transitions to `done`, a
`batch_auto_advanced` side effect is present in the response and the batch has status `done`.

**AC-010:** `entity(action: "create", type: "feature", parent: "P1-legacy", ...)` succeeds
when `P1-legacy.yaml` exists in `.kbz/state/plans/`, confirming backward-compatible feature
parent resolution.

**AC-011:** `entity(action: "create", type: "feature", parent: "B1-auth-system", ...)` succeeds
when `B1-auth-system.yaml` exists in `.kbz/state/batches/`.

**AC-012:** `status(id: "B1-auth-system")` returns a batch dashboard JSON object with
`"scope": "batch"` and a `"batch"` header key.

**AC-013:** `status(id: "P1-legacy")` where `P1-legacy` is a legacy plan file in `plans/`
returns a batch dashboard (not a plan dashboard), with a deprecation warning present.

**AC-014:** `entity(action: "create", type: "plan", ...)` continues to create a new plan
entity (F2), not a batch entity.

**AC-015:** `IsBatchID("B24-auth-system")` returns `true`. `IsBatchID("P1-old-plan")`
returns `true`. `IsBatchID("FEAT-01ABC")` returns `false`.

**AC-016:** `kbzschema.NewReader(repoRoot).ListBatches()` returns all batches from both
`batches/` and `plans/` directories; entities from `plans/` do not shadow entities from
`batches/`.

**AC-017:** `go test ./...` passes with no regressions in existing plan-related tests (which
continue to exercise the F2 plan entity path).

---

## Dependencies and Assumptions

**DEP-001 — P38-F1 (Config schema and project singleton):** This feature depends on F1 for:
- The `batch_prefixes` registry field in `Config`
- The `NextBatchNumber` function's integration with `batch_prefixes`
- The default batch prefix `"B"` in-memory fallback

If F1 is not deployed, `CreateBatch` MUST fall back to using `B` as the default prefix,
derived in-memory (consistent with F1's FR-006). Batch creation MUST NOT fail in the absence
of a `batch_prefixes` config entry.

**DEP-002 — P38-F2 (Plan entity data model):** This feature depends on F2 for the existence
of the plan entity kind in `.kbz/state/plans/`. F3 assumes that after F2 is deployed,
`.kbz/state/plans/` may contain both new plan entities (F2 format) and legacy batch files
(pre-F3 format). The batch service treats all P-prefix files in `plans/` as potential legacy
batches during the transition period.

**DEP-003 — P38-F7 (Physical migration):** F3 deliberately defers file migration to F7.
Until F7 runs, P-prefix batch files remain in `plans/`. The fallback behaviour in FR-008 and
FR-009 bridges this gap. No action is required from operators between F3 and F7 deployment.

**ASS-001:** The `batch_prefixes` registry does not contain the prefix character `P` (ensured
by F1's cross-registry uniqueness validation, FR-003 of F1). The B-prefix default does not
conflict with the P-prefix plan default.

**ASS-002:** Existing callers of `kbzschema.GetPlan` and `kbzschema.ListPlans` are not
disrupted because those methods continue to read from `plans/` and return `Plan` structs
unchanged.

**ASS-003:** Tests that create plan entities directly via `storage.EntityRecord{Type: "plan",
...}` continue to work; they exercise the F2 plan entity path. New batch tests use
`storage.EntityRecord{Type: "batch", ...}` and write to `batches/`.