# P38-F6: MCP Tools and Status Dashboard Updates — Specification

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Date    | 2026-04-27                                                         |
| Status  | draft                                                              |
| Feature | FEAT-01KQ7YQKPT8HF                                                 |
| Design  | P38 Meta-Planning: Plans and Batches — §5, §8, D1, D3             |

---

## Related Work

- **P38 design** — `work/design/meta-planning-plans-and-batches.md` — the primary design source; §5 (progress rollup and status dashboard), §8 (interaction with existing features), D1 (separate plan/batch types), D3 (batches retain plan functionality)
- **P38-F1** — Config Schema and Project Singleton — `work/design/p38-f1-spec-config-schema-project-singleton.md` — defines `plan_prefixes` and `batch_prefixes` registries used by entity create
- **P38-F2** — Plan Entity Data Model and Lifecycle — `work/design/p38-f2-spec-plan-entity-data-model-lifecycle.md` — defines the new plan entity schema, lifecycle (`idea→shaping→ready→active→done`), and service operations that the entity tool will expose
- **P38-F3** (pending) — Batch Entity Data Model — defines the batch entity schema, lifecycle (`proposed→designing→active→done`), and service operations; a dependency for batch CRUD in the entity tool
- **P38-F4** (pending) — Storage and Migration — defines the `.kbz/state/batches/` directory, state file migration, and `IsBatchID`/`ParseBatchID` model functions that this spec assumes exist
- Current entity tool: `internal/mcp/entity_tool.go`
- Current status tool: `internal/mcp/status_tool.go`
- Current estimate tool: `internal/mcp/estimate_tool.go`
- Current next tool: `internal/mcp/next_tool.go`
- Current doc tool: `internal/mcp/doc_tool.go`
- Model definitions: `internal/model/entities.go`

---

## Overview

P38 introduces two distinct entity types to replace the current overloaded `plan` entity:

- **`plan`** — a new strategic planning entity with a planning-oriented lifecycle (`idea → shaping → ready → active → done`), capable of recursive nesting via a `parent` field. Plans do not directly contain features; they contain child plans and/or child batches. Plans use `P{n}-slug` IDs (e.g. `P1-social-platform`).
- **`batch`** — the renamed operational work-grouping entity, identical in functionality to the current `plan`. Batches contain features, have the existing work lifecycle (`proposed → designing → active → done`), and use `B{n}-slug` IDs (e.g. `B24-auth-system`). A batch may optionally reference a parent plan.

This feature updates all MCP tools that currently reference the `plan` entity to support this new distinction. The changes are primarily in the MCP tool layer (handlers, dispatch, descriptions, response schemas). Service layer operations (`CreateBatch`, `GetBatch`, `ListBatches`, `ComputeBatchRollup`, `IsBatchID`, etc.) are delivered by P38-F3 and P38-F4 and are treated as dependencies here.

The principal areas of change are:

1. **Entity tool** — Add `type: batch` CRUD; change `type: plan` to the new planning entity; update parameter descriptions; update type dispatch and ID inference.
2. **Status tool** — Project overview shows top-level plans with recursive progress and standalone batches; plan ID routes to a new plan-tree dashboard; batch ID routes to the current plan dashboard (renamed batch dashboard).
3. **Estimate tool** — Add `case "batch"` for feature-level rollup; plan estimate query uses new recursive plan rollup.
4. **Next tool** — Recognise batch IDs in queue claim mode; expose `batch_id` in parent feature context.
5. **Doc tool** — Update `owner` parameter description and inheritance lookup for batch → parent plan chain.
6. **Backward compatibility** — Accept `P{n}` IDs in positions expecting batch IDs, with a deprecation warning, during the transition period.

---

## Functional Requirements

### Entity Tool

**FR-001 — Add `batch` to the entity type enumeration**
The entity tool MUST accept `type: "batch"` as a valid entity type for `action: create` and `action: list`. The `type` parameter description MUST be updated to read:

> "Entity type: plan, batch, feature, task, bug, decision (required for create and list)"

The tool's top-level description MUST include "batches" in the list of manageable entity types.

**FR-002 — `entity(action: create, type: batch)` — batch creation**
When `action: create` and `type: batch`, the entity tool MUST call the batch creation service with the following inputs accepted from the tool arguments:

| Field        | Required | Notes                                                              |
|--------------|----------|--------------------------------------------------------------------|
| `prefix`     | yes      | Single character; must be declared in the `batch_prefixes` registry |
| `slug`       | yes      | URL-safe lowercase                                                 |
| `name`       | yes      | Human-readable display name                                        |
| `summary`    | yes      | Brief description                                                  |
| `created_by` | yes      | Identity (auto-resolved if omitted)                                |
| `parent`     | no       | Parent plan ID (`P{n}-slug`). Validated: must reference an existing plan if set. |
| `design`     | no       | Design document reference                                          |
| `tags`       | no       | Freeform tags                                                      |

The created batch MUST have its initial status set to `proposed`. The batch ID format is `B{n}-{slug}`.

**FR-003 — `entity(action: create, type: plan)` — plan creation (new lifecycle)**
When `action: create` and `type: plan`, the entity tool MUST call the plan creation service with the following inputs:

| Field        | Required | Notes                                                               |
|--------------|----------|---------------------------------------------------------------------|
| `prefix`     | yes      | Single character; must be declared in the `plan_prefixes` registry  |
| `slug`       | yes      | URL-safe lowercase                                                  |
| `name`       | yes      | Human-readable display name                                         |
| `summary`    | yes      | Brief description                                                   |
| `created_by` | yes      | Identity (auto-resolved if omitted)                                 |
| `parent`     | no       | Parent plan ID (`P{n}-slug`) for recursive nesting                 |
| `design`     | no       | Design document reference                                           |
| `tags`       | no       | Freeform tags                                                       |

The created plan MUST have its initial status set to `idea` (not `proposed`). The plan ID format is `P{n}-{slug}` (unchanged from today).

> **Note:** This changes the current behaviour of `type: plan` create. Today, `type: plan` creates a work-grouping entity with `status: proposed` and plan lifecycle. After P38, `type: plan` creates a strategic planning entity with `status: idea` and the planning lifecycle. Work-grouping entities are created with `type: batch`.

**FR-004 — `entity(action: get, id: B{n}-...)` — batch get**
When `action: get` and the `id` value is a batch ID (matching `IsBatchID`), the entity tool MUST call the batch get service and return the batch's full state, using the same `entityFullRecord` response envelope as other entity types. The response `type` field MUST be `"batch"`.

**FR-005 — `entity(action: list, type: batch)` — batch list**
When `action: list` and `type: batch`, the entity tool MUST call the batch list service. The following filters MUST be passed through when provided:

- `status` — filter by batch lifecycle status
- `parent` — filter by parent plan ID; when set to `""` or omitted, returns all batches (both standalone and plan-parented); implementors MAY add a `standalone` filter value in a later feature
- `tags` — filter by tags

The response format MUST match the existing `entityListResponse` shape.

**FR-006 — `entity(action: list, type: plan)` — plan list with parent filter**
When `action: list` and `type: plan`, the entity tool MUST pass through the `parent` filter to the plan list service:
- `parent: ""` (explicit empty string) — return only top-level plans (no parent)
- `parent: "P1-social-platform"` — return only direct children of that plan
- `parent` absent — return all plans (current behaviour preserved)

**FR-007 — `entity(action: transition, id: B{n}-..., status: ...)` — batch lifecycle transition**
When the entity ID is a batch ID, the entity tool MUST use the batch lifecycle transition path (not the plan path). The allowed transitions follow the existing work lifecycle:

```
proposed → designing, active, superseded, cancelled
designing → active, proposed (backward), superseded, cancelled
active → reviewing, done, designing (backward), superseded, cancelled
reviewing → done, needs-rework, superseded, cancelled
done → superseded, cancelled
```

Terminal-state gate: before transitioning a batch to `done`, `superseded`, or `cancelled`, the tool MUST check for non-terminal features (using the existing `CountNonTerminalFeatures` mechanism). If non-terminal features exist and `override` is not set, the transition MUST be rejected with a structured error.

**FR-008 — `entity(action: transition, id: P{n}-..., status: ...)` — plan lifecycle transition**
When the entity ID is a plan ID, the entity tool MUST use the plan lifecycle transition path with the new planning lifecycle:

```
idea → shaping, superseded, cancelled
shaping → ready, idea (backward), superseded, cancelled
ready → active, shaping (backward), superseded, cancelled
active → done, shaping (backward), superseded, cancelled
done → superseded, cancelled
```

No document-gate enforcement is applied to plan transitions (consistent with current plan behaviour and design §2 guidance). Plans are human-managed.

Terminal-state gate: before transitioning a plan to `done`, `superseded`, or `cancelled`, the tool MUST check for non-terminal child batches and non-terminal child plans. If any exist and `override` is not set, the transition MUST be rejected with a structured error identifying the count and type of blocking children.

**FR-009 — `entityKindFromType` — add batch mapping**
The `entityKindFromType` function MUST add `case "batch"` mapping to the new `validate.EntityBatch` kind constant (defined in P38-F4). The existing `case "plan"` mapping continues to map to `validate.EntityPlan`.

**FR-010 — `entityInferType` — add batch ID recognition**
The `entityInferType` function MUST be updated to recognise batch IDs via `model.IsBatchID(entityID)` (defined in P38-F4) and return `"batch"`. The updated logic MUST check batch before plan to avoid ambiguity:

```
IsBatchID(id) → "batch"
IsPlanID(id)  → "plan"
```

**FR-011 — `parent` parameter — accept batch IDs for feature create**
The `parent` parameter on the entity tool MUST be updated in its description to clarify that it accepts batch IDs for features:

> "Parent batch ID (B{n}-slug) for features — required on feature create to associate the feature with its batch. Also used as a filter on list. During migration, P{n}-slug is accepted as a batch ID with a deprecation warning. Note: tasks use parent_feature, not parent."

**FR-012 — `prefix` parameter — clarify plan vs batch registry**
The `prefix` parameter description MUST be updated to clarify that it selects a prefix from the appropriate registry (plan prefixes for `type: plan`, batch prefixes for `type: batch`):

> "Single-character ID prefix — must be declared in the plan_prefixes registry (plan create) or batch_prefixes registry (batch create)"

**FR-013 — Duplicate advisory — batch type**
The `entityDuplicateAdvisory` function MUST handle `entityType == "batch"` by listing existing batch entities (via the batch list service) and checking for name/summary similarity, following the same pattern as the existing plan advisory.

**FR-014 — `entityCurrentStatus` — add batch path**
The `entityCurrentStatus` function (used before transition for commit message generation) MUST add a `case "batch"` path that calls `entitySvc.GetBatch(entityID)` to retrieve the current status.

**FR-015 — `entityCreateOne` default error — include batch**
The `default` error branch in `entityCreateOne` MUST be updated to include `batch` in the list of valid types:

> "Cannot create entity: unknown type %q. Use one of: plan, batch, feature, task, bug, decision"

---

### Status Tool

**FR-016 — `inferIDType` — add batch ID type**
The `inferIDType` function MUST add a `idTypeBatch` constant and a corresponding case that recognises batch IDs via `model.IsBatchID(id)`. The switch MUST check batch before plan:

```go
case model.IsBatchID(id):
    return idTypeBatch
case model.IsPlanID(id):
    return idTypePlan
```

**FR-017 — Status tool description — mention batch IDs**
The status tool description MUST be updated to mention batch IDs explicitly:

> "Call with no id for project overview, plan ID (e.g. P1-slug) for plan tree dashboard, batch ID (e.g. B24-slug) for batch dashboard, FEAT-... for feature detail, TASK-... or BUG-... for task/bug detail."

The `id` parameter description MUST also be updated to list batch ID as a valid input.

**FR-018 — Status dispatch — batch routes to batch dashboard**
The status tool handler's `switch inferIDType(id)` MUST dispatch `idTypeBatch` to `synthesiseBatch(id, ...)`, which is the renamed (and otherwise unchanged) current `synthesisePlan` function.

**FR-019 — `synthesiseBatch` — rename and update scope field**
The current `synthesisePlan` function MUST be renamed `synthesiseBatch`. Its logic is unchanged except:
- It MUST call `entitySvc.GetBatch(batchID)` instead of `entitySvc.GetPlan(planID)`
- The response's `scope` field MUST be `"batch"`
- The `plan` header field in the response MUST be renamed to `batch` in the JSON output
- Error messages MUST say "batch" not "plan"
- The `planDashboard` struct type MAY be renamed `batchDashboard` in code

**FR-020 — `synthesisePlan` — new plan-tree dashboard**
A new `synthesisePlan` function MUST be implemented that produces a plan-tree dashboard for a given plan ID. It MUST:

1. Load the plan via `entitySvc.GetPlan(planID)`.
2. List direct child plans (via `entitySvc.ListPlans` filtered by `parent == planID`).
3. List direct child batches (via `entitySvc.ListBatches` filtered by `parent == planID`).
4. For each child batch, compute or approximate task-level progress (ready/active/done/total counts).
5. For each child plan, include its status and a summary of its own child counts (non-recursive for MVP; full recursive rollup is deferred to a later feature).
6. Return a response with `scope: "plan"` and the following fields:

```json
{
  "scope": "plan",
  "plan": {
    "display_id": "P1-social-platform",
    "id": "P1-social-platform",
    "slug": "social-platform",
    "name": "Social Media Platform",
    "status": "active",
    "parent": ""
  },
  "children": {
    "plans": [ /* child plan summaries */ ],
    "batches": [ /* child batch summaries */ ]
  },
  "attention": [ /* attention items */ ],
  "generated_at": "..."
}
```

Child batch summaries MUST include: `display_id`, `id`, `slug`, `name`, `status`, `features` (count), and `tasks` (ready/active/done/total). Child plan summaries MUST include: `display_id`, `id`, `slug`, `name`, `status`, `parent`.

**FR-021 — `synthesiseProject` — show top-level plans and standalone batches**
The `synthesiseProject` function MUST be updated to show both top-level plans and standalone batches:

1. **Top-level plans**: Load all plans via `entitySvc.ListPlans(PlanFilters{})`. Filter to those with no `parent` field set. Compute recursive progress for each by summing task counts across all child batches' features (and, for MVP, across any directly nested child batches at one level deep).
2. **Standalone batches**: Load all batches via `entitySvc.ListBatches(BatchFilters{})`. Filter to those with no `parent` field set (standalone batches, not parented to a plan).

The `projectOverview` response struct MUST be updated:
- The existing `Plans []planSummary` field is renamed in JSON output to `plans` and now contains **top-level plan** summaries.
- A new `Batches []batchSummary` field (JSON: `"batches"`) lists standalone batch summaries.
- The `planAggregate.Plans` count MUST count top-level plans. A new `Batches` count MUST count standalone batches. The `Features`, `Tasks` aggregate counts MUST include tasks under both plan-parented and standalone batches.

> **Backward compatibility note:** If the P38-F4 migration has not yet been applied to a repository, the batch store may be empty and the plans store may still contain work-grouping plans. `synthesiseProject` MUST handle this gracefully: if no batches are found, the `batches` field in the response MUST be an empty array (not omitted).

**FR-022 — Attention generation — update for plan/batch distinction**
The `generateProjectAttention` and `generatePlanAttention` functions MUST be updated:

- `generateProjectAttention`: existing logic referencing "plan" variables/comments MUST be renamed to "batch" where the intent is to refer to work-grouping entities. Plan-specific attention items (e.g. "plan needs design") MUST be added for top-level plans in `idea` or `shaping` status.
- A new `generatePlanChildAttention` function (or extension of `generatePlanAttention`) MUST produce attention items for the plan-tree dashboard, including:
  - Child batches blocked (all features blocked)
  - Child plans stuck in `idea` without a design document for more than a configurable period (best-effort, advisory only)

---

### Estimate Tool

**FR-023 — `estimateQueryAction` — add batch rollup**
The `estimateQueryAction` switch MUST add a `case "batch"` that calls `entitySvc.ComputeBatchRollup(result.ID)` (the batch equivalent of the current `ComputePlanRollup`). The response shape MUST mirror the current plan rollup response:

```json
{
  "entity_id": "B24-auth-system",
  "entity_type": "batch",
  "estimate": 40,
  "rollup": {
    "feature_total": 38,
    "progress": 0.75,
    "delta": -2,
    "feature_count": 3,
    "estimated_feature_count": 3
  }
}
```

**FR-024 — `estimateQueryAction` — plan rollup uses recursive computation**
The existing `case "plan"` in `estimateQueryAction` MUST call the new `entitySvc.ComputeRecursivePlanRollup(result.ID)` (defined in P38-F3 or equivalent service feature). The response shape MUST include recursive child counts:

```json
{
  "entity_id": "P1-social-platform",
  "entity_type": "plan",
  "estimate": null,
  "rollup": {
    "batch_total": 120,
    "feature_total": 115,
    "progress": 0.42,
    "batch_count": 4,
    "plan_count": 2
  }
}
```

If the recursive rollup service is not yet available (P38 service features pending), `estimateQueryAction` for `type: plan` MUST return `rollup: null` with a `note` field explaining that recursive rollup is not yet computed, rather than failing.

---

### Next Tool

**FR-025 — `nextInferEntityType` — recognise batch IDs**
The `nextInferEntityType` function MUST add recognition of batch IDs:

```go
case model.IsBatchID(id):
    return "batch"
```

The batch case MUST be checked before the plan case.

**FR-026 — `nextResolveTaskID` — handle batch scope**
The `nextResolveTaskID` function MUST add a `case "batch"` that queries tasks across all features in the batch, using `entitySvc.CrossEntityBatchQuery(id)` (or the equivalent batch version of `CrossEntityQuery`). The implementation MUST:
1. Load all ready tasks whose parent feature belongs to the specified batch.
2. Sort them by the existing `nextSortByQueueOrder` logic.
3. Return the highest-priority ready task ID.

Error messages MUST say "batch" not "plan".

**FR-027 — `nextClaimMode` — `batch_id` in parent feature info**
In the `parentFeatureInfo` map returned by `nextClaimMode`, the field currently named `plan_id` (which holds the feature's `parent` field value) MUST be renamed to `batch_id`. A deprecated alias `plan_id` MAY be included alongside `batch_id` with a value of `null` and a `deprecated: true` flag during the transition period, if backward compatibility with existing agent prompts is required.

**FR-028 — Next tool `id` parameter description — mention batch IDs**
The `id` parameter description in the next tool MUST include batch IDs:

> "Task ID (TASK-... or T-...), Feature ID (FEAT-...), Batch ID (e.g. B24-slug), or Plan ID (e.g. P1-slug) to claim. Omit to inspect the ready queue."

---

### Doc Tool

**FR-029 — `owner` parameter description — include batch IDs**
The `owner` parameter description in the doc tool MUST be updated to:

> "Parent Plan, Batch, or Feature ID (register, list, import)"

**FR-030 — `docGapsAction` — three-level inheritance: feature → batch → plan**
The `docGapsAction` inheritance lookup currently walks feature → parent plan (two levels). After P38, features parent to batches, and batches may parent to plans. The inheritance lookup MUST be extended to three levels:

1. Feature-owned documents (direct).
2. Parent batch documents (feature's `parent` field references a batch).
3. Grandparent plan documents (batch's `parent` field references a plan, if set).

An approved document at any level satisfies the gap check. The `inherited: true` flag MUST be present on any inherited document in the gaps response, and the `inherited_from` field MUST identify whether the source is `"batch"` or `"plan"`.

---

### Decompose, Merge, PR, and Worktree Tools

**FR-031 — Decompose tool — no interface changes required**
The `decompose` tool operates solely on `feature_id`. Features continue to reference their parent entity via the `parent` field (which will hold a batch ID after migration). No changes to the decompose tool interface or dispatch logic are required by this feature. Tool description and parameter documentation do not reference plan or batch and need no update.

**FR-032 — Merge, PR, and worktree tools — no changes required**
The `merge`, `pr`, and `worktree` tools operate exclusively on `FEAT-...` and `BUG-...` entity IDs. No plan or batch references appear in their interfaces or core logic. These tools require no changes for P38-F6.

---

### Backward Compatibility

**FR-033 — Accept `P{n}` IDs as batch IDs with deprecation warning**
During the migration transition period, tools that accept batch IDs MUST also accept `P{n}-slug` IDs and treat them as batch IDs. When a `P{n}` ID is used in a batch position, the tool response MUST include a top-level `deprecation_warning` field:

```json
{
  "deprecation_warning": "ID 'P24-auth-system' uses the legacy plan prefix. After migration, this entity will use the batch ID 'B24-auth-system'. Update all references to use the new batch ID."
}
```

This applies to:
- `entity(action: get, id: P{n}-...)` when the entity is now a batch
- `entity(action: transition, id: P{n}-..., status: ...)` when the entity is now a batch
- `status(id: P{n}-...)` when the entity is now a batch
- `next(id: P{n}-...)` when used as a batch scope
- `estimate(action: query, entity_id: P{n}-...)` when the entity is now a batch

**FR-034 — `entityInferType` backward-compat ambiguity resolution**
When `model.IsPlanID(id)` matches but `model.IsBatchID(id)` also matches (during the transition period when the same prefix format could be either type), the entity tool MUST look up the entity in storage to disambiguate. If found in the batch store, it MUST return `"batch"`. If found in the plan store, it MUST return `"plan"`. If not found in either, it MUST return an error.

---

## Non-Functional Requirements

**NFR-001 — Tool description budget**
All updated tool descriptions MUST remain within the token budget enforced by `tool_description_budget_test.go`. Descriptions must be updated without exceeding existing character limits. Where necessary, descriptions should be made more concise to accommodate new type names.

**NFR-002 — Error message format**
All new and updated error messages MUST follow the existing Kanbanzai error message convention:
> "Cannot {verb} {noun}: {reason}.\n\nTo resolve:\n  {actionable suggestion}"

Error messages for batch operations MUST say "batch" not "plan".

**NFR-003 — No new tool registrations**
This feature MUST NOT add new top-level MCP tools. All batch and plan functionality is exposed through existing tools (`entity`, `status`, `estimate`, `next`, `doc`) by extending their type dispatch. The total tool count in `server.go` MUST remain unchanged.

**NFR-004 — Response shape stability for batch dashboard**
The batch dashboard response (FR-019) MUST be a drop-in replacement for the current plan dashboard response, with only the following changes: `scope: "batch"` instead of `"plan"`, and the `plan` header object renamed to `batch`. Existing agent code that reads the plan dashboard response shape MUST continue to work if it reads by field name and ignores the `scope` value.

**NFR-005 — Graceful degradation when service layer is not ready**
If the batch service layer (P38-F3/F4) is not yet deployed, calling `entity(action: create, type: batch, ...)` MUST return a clear `not_implemented` error rather than a panic or an opaque internal error.

**NFR-006 — Auto-commit messages updated**
Auto-commit messages generated by the entity tool after batch operations MUST use the word "batch":
- Create: `workflow(B24-auth-system): create batch`
- Transition: `workflow(B24-auth-system): transition proposed → designing`

---

## Scope

**In scope:**
- Entity tool: type dispatch for `plan` (new lifecycle) and `batch` (work lifecycle), including CRUD and transition handlers
- Entity tool: parameter description updates for `type`, `parent`, `prefix`
- Status tool: `inferIDType` extension, `synthesiseBatch` (renamed), new `synthesisePlan` (tree view), `synthesiseProject` split into plans + standalone batches
- Estimate tool: `case "batch"` rollup; plan estimate uses recursive rollup service
- Next tool: batch ID recognition, `nextResolveTaskID` batch case, `batch_id` in parent feature info
- Doc tool: `owner` description update, three-level inheritance in `docGapsAction`
- Backward compatibility: `P{n}` IDs accepted as batch IDs with deprecation warning
- All tool description and parameter documentation updates

**Out of scope:**
- Service layer changes (`EntityService`, `CreateBatch`, `GetBatch`, `ListBatches`, `ComputeBatchRollup`, `ComputeRecursivePlanRollup`) — covered by P38-F3 and P38-F4
- Model layer changes (`model.Batch`, `model.BatchStatus`, `IsBatchID`, `ParseBatchID`, `EntityKindBatch`) — covered by P38-F4
- Storage migration (`.kbz/state/plans/` → `.kbz/state/batches/`, ID renaming) — covered by P38-F4
- Work folder renaming (`P{n}-slug/` → `B{n}-slug/`) — covered by P38-F4 or a separate migration feature
- `kbz` CLI sub-commands for plan/batch management
- Recursive plan rollup service implementation — covered by P38-F3 or equivalent
- Handoff tool (`handoff`) — references plan context indirectly via feature assembly; updates deferred to a follow-on feature once batch migration is complete
- Retro tool, knowledge tool, checkpoint tool, incident tool — no plan/batch references in their interfaces

---

## Acceptance Criteria

**AC-001 — Batch create via entity tool**
`entity(action: "create", type: "batch", prefix: "B", slug: "auth-system", name: "Auth System", summary: "OAuth2 and passcode auth")` creates a batch entity with ID matching `B{n}-auth-system` and `status: "proposed"`. The response `type` field is `"batch"`.

**AC-002 — Plan create via entity tool (new lifecycle)**
`entity(action: "create", type: "plan", prefix: "P", slug: "social-platform", name: "Social Media Platform", summary: "End-to-end social platform")` creates a plan entity with ID matching `P{n}-social-platform` and `status: "idea"` (not `"proposed"`). The response `type` field is `"plan"`.

**AC-003 — Plan transition: idea → shaping**
`entity(action: "transition", id: "P1-social-platform", status: "shaping")` succeeds and returns the plan with `status: "shaping"`.

**AC-004 — Plan transition: invalid (idea → active)**
`entity(action: "transition", id: "P1-social-platform", status: "active")` returns an error indicating the transition from `idea` to `active` is not allowed.

**AC-005 — Batch transition: proposed → designing**
`entity(action: "transition", id: "B24-auth-system", status: "designing")` succeeds and returns the batch with `status: "designing"`.

**AC-006 — Batch terminal gate: non-terminal features block done**
`entity(action: "transition", id: "B24-auth-system", status: "done")` when one or more non-terminal features exist returns a gate failure error identifying the number of blocking features. With `override: true` and `override_reason: "..."`, the transition succeeds.

**AC-007 — `entity(action: list, type: batch)` returns batches**
Calling `entity(action: "list", type: "batch")` returns a list of batch entities. Calling `entity(action: "list", type: "plan")` returns a list of plan entities. The two lists are disjoint.

**AC-008 — `entity(action: list, type: plan, parent: "")` returns top-level plans**
Calling `entity(action: "list", type: "plan", parent: "")` returns only plans whose `parent` field is empty or unset.

**AC-009 — `status()` project overview shows plans and standalone batches**
`status()` (no ID) returns a response containing both a `"plans"` array (top-level plans) and a `"batches"` array (standalone batches with no parent plan). Task counts in `total` aggregate across both.

**AC-010 — `status("P1-social-platform")` returns plan-tree dashboard**
`status(id: "P1-social-platform")` returns a response with `scope: "plan"`, a `plan` header object, and a `children` object containing `plans` (direct child plans) and `batches` (direct child batches). The response does NOT include a top-level `features` array.

**AC-011 — `status("B24-auth-system")` returns batch dashboard**
`status(id: "B24-auth-system")` returns a response with `scope: "batch"` and a `batch` header object. The `features` array lists features belonging to this batch. The response is functionally identical to the current plan dashboard.

**AC-012 — Estimate query on batch returns feature rollup**
`estimate(action: "query", entity_id: "B24-auth-system")` returns `entity_type: "batch"` and a `rollup` with `feature_total`, `feature_count`, and `progress`.

**AC-013 — Estimate query on plan returns plan rollup**
`estimate(action: "query", entity_id: "P1-social-platform")` returns `entity_type: "plan"` and a `rollup` with `batch_count`, `plan_count`, and `progress`.

**AC-014 — `next(id: "B24-auth-system")` claims a ready task in the batch**
Calling `next(id: "B24-auth-system")` finds the highest-priority ready task among all features belonging to `B24-auth-system` and transitions it to active. The returned `task.parent_feature` includes `batch_id: "B24-auth-system"`.

**AC-015 — `next(id: "P1-social-platform")` claims a ready task in any child batch**
Calling `next(id: "P1-social-platform")` finds the highest-priority ready task across all batches (and their features) that are children of `P1-social-platform`.

**AC-016 — Doc `owner` accepts batch ID**
`doc(action: "register", path: "work/B24-auth-system/B24-design-auth-system.md", type: "design", title: "Auth System Design", owner: "B24-auth-system")` registers the document with `owner: "B24-auth-system"` without error.

**AC-017 — Doc gaps inheritance: feature → batch → plan**
For a feature whose parent batch has a parent plan, and the plan has an approved design document, `doc(action: "gaps", feature_id: "FEAT-...")` reports the design gap as satisfied with `inherited: true` and `inherited_from: "plan"`.

**AC-018 — Backward compat: `P{n}` as batch ID returns deprecation warning**
After migration, calling `entity(action: "get", id: "P24-auth-system")` on an entity now stored as a batch returns the batch record AND a `deprecation_warning` field in the response.

**AC-019 — Unknown type returns error including "batch"**
`entity(action: "create", type: "sprint", ...)` returns an error message that lists valid types including "batch": "use one of: plan, batch, feature, task, bug, decision".

---

## Dependencies and Assumptions

### Dependencies

| Feature / Component | What is needed |
|---------------------|----------------|
| P38-F2 (Plan Entity Data Model) | `entitySvc.CreatePlan` with new `idea` entry state; `entitySvc.UpdatePlanStatus` accepting new planning lifecycle transitions; `model.PlanPlanningStatus` (or equivalent) constants for `idea`, `shaping`, `ready` |
| P38-F3 (Batch Entity Data Model) | `entitySvc.CreateBatch`, `entitySvc.GetBatch`, `entitySvc.ListBatches`, `entitySvc.UpdateBatchStatus`, `entitySvc.ComputeBatchRollup`, `entitySvc.ComputeRecursivePlanRollup`, `entitySvc.CrossEntityBatchQuery`, `model.EntityKindBatch`, `validate.EntityBatch` |
| P38-F4 (Storage and Migration) | `model.IsBatchID`, `model.ParseBatchID`, `.kbz/state/batches/` store; batch state files accessible via entity service |
| P38-F1 (Config Schema) | `plan_prefixes` and `batch_prefixes` registries in config; `id.Allocator` able to allocate from either registry |

### Assumptions

1. The service layer is extended with distinct `CreatePlan` (planning lifecycle, `idea` entry) and `CreateBatch` (work lifecycle, `proposed` entry) operations before this feature is implemented. If the service layer is not ready, the entity tool MUST return a `not_implemented` error rather than silently using the wrong operation.

2. `IsBatchID` returns false for `P{n}-slug` IDs and true for `B{n}-slug` IDs. During the migration transition period, the disambiguation described in FR-034 applies. After migration is complete, `IsBatchID` and `IsPlanID` are mutually exclusive and FR-034 is no longer needed.

3. The `parent` field on `model.Plan` (new planning entity) is added by P38-F2. The `parent` field on `model.Batch` (renamed current plan) is added by P38-F3. Both reference a plan ID (`P{n}-slug`).

4. The current `entity_tool.go` `case "plan"` in `entityCreateOne` is **replaced** by the new planning-entity behaviour (not duplicated). Agents that were using `type: plan` to create work-grouping entities MUST be updated to use `type: batch`.

5. The `synthesiseBatch` function (renamed `synthesisePlan`) MUST produce output that existing agent code reading the plan dashboard can consume without modification, given that `scope` changes from `"plan"` to `"batch"` and the `plan` header key changes to `batch`. This is an acceptable breaking change; agent instructions will be updated separately (P38 doc-publishing feature).

6. The tool description budget test (`tool_description_budget_test.go`) will be updated as part of this feature to accommodate new type names without requiring description truncation that would lose meaningful content.
```

Now let me register the spec document: