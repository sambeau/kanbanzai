# P38-F4: Feature Display IDs and Document Inheritance — Specification

| Field   | Value |
|---------|-------|
| Date    | 2026-04-27 |
| Status  | draft |
| Feature | FEAT-01KQ7YQKHK2GV |
| Design  | P38 Meta-Planning: Plans and Batches — §6, §8, D4, D6 |

---

## Related Work

**Design:** `work/design/meta-planning-plans-and-batches.md` — §6 (impact on P37 file naming),
§8 (interaction with existing features), D4 (batch `B` prefix), D6 (design documents at any
level).

| Document | Relevance |
|----------|-----------|
| `work/design/p37-f1-spec-plan-scoped-feature-display-ids.md` | Defines `P{n}-F{m}` display ID format, `next_feature_seq` on plan state files, and `display_id` on feature state files. This feature modifies the format and moves the counter. |
| P38 Meta-Planning Design §3 | Batch entity: `next_feature_seq` moves from plan state file to batch state file. |
| P38 Meta-Planning Design §6 | Feature display IDs change from `P{n}-F{m}` to `B{n}-F{m}`; `next_feature_seq` moves to batch state files. |
| P38 Meta-Planning Design §8, D6 | Three-level document gate extends to four levels: feature → feature-owned → parent batch → grandparent plan. |
| P38-F1 (config schema, prefix registries) | Provides the `batch_prefixes` registry from which the batch prefix character is resolved. Needed to determine the prefix letter used in the display ID format string. |
| P38-F2 (plan entity data model) | Defines the new plan entity with `parent` field pointing to a parent plan. The grandparent plan is looked up via the batch's `parent` field. |
| P38-F3 (batch entity) | Renames the current plan entity to batch. Defines the batch state file schema including `next_feature_seq`. This feature's counter logic targets the batch state file produced by F3. |
| `internal/model/entities.go` | `Plan` struct (current), `Feature` struct — fields `DisplayID` and `NextFeatureSeq` are added by P37-F1; this feature updates their semantics for batch parents. |
| `internal/service/entities.go` | `CreateFeature` — counter allocation and display ID generation must be updated to target the batch state file. |
| `internal/gate/eval_documents.go` | `evalOneDocument` — three-level lookup chain must be extended to four levels. |
| `internal/gate/evaluator.go` | `PrereqEvalContext` — must carry a parent-entity resolver so Level 4 can look up the batch's parent plan. |

---

## Overview

This specification covers two related changes that adapt the P37-F1 display ID system and the
document gate system to the new two-layer entity hierarchy introduced in P38
(recursive plan + execution batch).

**Change 1 — Batch-scoped feature display IDs.** P37-F1 introduced display IDs in the form
`P{n}-F{m}`, where `{n}` is the plan number and `{m}` is a per-plan sequence counter. P38
renames the current plan entity to "batch" and assigns it a `B{prefix}{n}` ID format. This
feature updates the display ID template: features under a batch get `B{n}-F{m}` instead of
`P{n}-F{m}`. The `next_feature_seq` counter moves from the plan state file to the batch state
file. Features with a legacy `P{n}` parent (pre-migration) retain `P{n}-F{m}` display IDs for
backward compatibility.

**Change 2 — Four-level document gate lookup.** The current gate evaluator uses a three-level
chain when checking document prerequisites:

1. Feature's own document field reference
2. Documents owned by the feature
3. Documents owned by the parent (currently: plan)

This feature extends the chain by one level. When a feature's parent is a batch, and that
batch has a parent plan, the gate evaluator also checks:

4. Documents owned by the grandparent plan

This enables a feature under a batch under a plan to satisfy a design gate using the
grandparent plan's approved design document, without requiring the batch or feature to hold
its own copy.

---

## Functional Requirements

### FR-001 — Batch state file `next_feature_seq` counter

The Batch state file (`.kbz/state/batches/{id}.yaml`) MUST contain a `next_feature_seq`
field of type integer.

> This counter replaces the plan-side counter added by P37-F1. As of this feature, the
> authoritative counter is on the batch, not on a plan. Existing plan state files that carry
> `next_feature_seq` from P37-F1 are not affected by this feature; P38-F3 (batch entity) is
> responsible for defining the full batch schema. This requirement records that `next_feature_seq`
> belongs on the batch state file.

### FR-002 — Batch counter initialisation

When a new batch entity is created, `next_feature_seq` MUST be initialised to `1`.

### FR-003 — Batch counter increment

Each time a feature is successfully created under a batch, the batch's `next_feature_seq`
MUST be incremented by exactly 1.

### FR-004 — Counter write ordering

During feature creation under a batch, the batch's `next_feature_seq` MUST be written with
its incremented value BEFORE the feature state file is written to disk. This ordering ensures
that a process crash between the two writes leaves a gap in the sequence rather than producing
a duplicate `display_id`. Sequence gaps are acceptable; duplicate display IDs within a batch
are not.

### FR-005 — Feature display ID format for batch parents

When a feature is created under a batch parent, the `display_id` value MUST match the pattern
`{BatchPrefix}{n}-F{m}`, where:

- `{BatchPrefix}` is the prefix character of the batch ID (e.g. `B` for `B24-auth-system`).
- `{n}` is the integer component of the batch's ID (e.g. `24` for `B24-auth-system`).
- `{m}` is the value of `next_feature_seq` read from the batch state file before the
  increment (i.e. the sequence number allocated to this feature).

**Example:** Feature created under batch `B24-auth-system` when `next_feature_seq` is `3`
receives `display_id: B24-F3`.

### FR-006 — CreateFeature allocates display ID from batch counter

When `CreateFeature` is called with a batch entity as the parent, it MUST execute the
following steps in order:

1. Read the current value of `next_feature_seq` from the parent batch state file (call it N).
2. Compute `display_id` as `{BatchPrefix}{n}-F{N}` per FR-005.
3. Write the batch state file with `next_feature_seq` set to N+1.
4. Write the feature state file with `display_id` set to `{BatchPrefix}{n}-F{N}`.

If step 3 fails, `CreateFeature` MUST return an error and MUST NOT write the feature state
file. If step 4 fails, the batch counter has already been incremented; the sequence number N
is considered consumed and MUST NOT be reused.

### FR-007 — Backward compatibility: legacy plan parents retain `P{n}-F{m}`

Features whose `parent` field references a legacy plan entity (a plan ID in the `P{n}-{slug}`
format, not yet migrated to a batch) MUST continue to receive `display_id` values in the
`P{n}-F{m}` format using the plan's `next_feature_seq` counter, as defined by P37-F1.
The behaviour described in FR-005 and FR-006 applies only when the parent is a batch entity.

> This rule ensures no behavioural regression during the migration period when some plan
> entities have not yet been renamed to batches.

### FR-008 — Display ID input resolution: `B{n}-F{m}` pattern

The entity resolution layer MUST recognise input matching the pattern `B{n}-F{m}` (where `n`
and `m` are positive integers, e.g. `B24-F3`) and resolve it to the canonical
`FEAT-{TSID13}` ID of the matching feature before dispatching to any entity operation.

### FR-009 — Resolution is case-insensitive for both patterns

`B{n}-F{m}` pattern matching MUST be case-insensitive. The inputs `B24-F3`, `b24-f3`, and
`B24-f3` MUST all resolve to the same feature. The existing case-insensitivity for `P{n}-F{m}`
(P37-F1 REQ-010) is unchanged.

### FR-010 — All entity operations accept `B{n}-F{m}` identifiers

The following entity operations MUST accept a `B{n}-F{m}` identifier wherever a feature ID
is accepted as input:

- `entity get` — returns the feature matching the display ID.
- `entity update` — applies updates to the feature matching the display ID.
- `entity transition` — transitions the feature matching the display ID.
- `entity list` — when a `B{n}-F{m}` value is supplied as an ID filter, returns the matching
  feature.

### FR-011 — MCP output reflects batch-scoped display ID

MCP tool responses that include a feature's identity MUST include the `display_id` field in
`B{n}-F{m}` form for any feature that has a batch parent. This field MUST be the primary
human-facing identifier. The TSID break-hyphen form MAY also be present but MUST NOT be the
sole identifier shown.

---

### FR-012 — Four-level document gate lookup chain

The document prerequisite evaluation function (`evalOneDocument` in
`internal/gate/eval_documents.go`) MUST evaluate document prerequisites using the following
four-level lookup chain, terminating at the first satisfied level:

1. **Feature field reference.** If the feature's document field reference for the prerequisite
   type (`Design`, `Spec`, `DevPlan`) is non-empty, fetch that document record. If it exists
   and its status matches the required status, the gate is satisfied.

2. **Feature-owned documents.** List all approved documents of the required type owned by the
   feature. If any exist, the gate is satisfied.

3. **Parent batch-owned documents.** If the feature has a parent batch (its `parent` field
   references a batch entity), list all approved documents of the required type owned by the
   batch. If any exist, the gate is satisfied.

4. **Grandparent plan-owned documents.** If the feature's parent batch has a `parent` field
   that references a plan entity, list all approved documents of the required type owned by
   that plan. If any exist, the gate is satisfied.

If no level is satisfied, the gate result is unsatisfied.

### FR-013 — Grandparent plan lookup via batch parent field

To perform the Level 4 lookup (FR-012), the gate evaluator MUST:

1. Resolve the feature's parent entity using `Feature.Parent`.
2. Read the parent batch's state file and check its `parent` field.
3. If the batch's `parent` field is non-empty and references a plan entity
   (i.e. matches the plan ID format), use that plan ID as the owner for the Level 4
   document query.

### FR-014 — PrereqEvalContext carries a parent entity resolver

`PrereqEvalContext` (in `internal/gate/evaluator.go`) MUST be extended to carry the
information or service interface needed to resolve the parent batch's `parent` field. The
minimal interface exposes the ability to look up an entity by ID and return its `parent`
field value.

> Concrete shape is left to the implementer. A `ParentOf(entityID string) (string, error)`
> method on a minimal interface, or injecting the full `EntityService`, are both acceptable
> approaches. The chosen approach MUST NOT introduce import cycles.

### FR-015 — Lookup chain terminates at first satisfied level

The four-level lookup MUST short-circuit: once a level is satisfied, subsequent levels MUST
NOT be queried. A satisfied Level 2 result MUST prevent Level 3 and Level 4 queries from
running, and so on.

### FR-016 — Standalone features use existing three-level lookup

A feature with no parent batch (i.e. `Feature.Parent` is empty, or references a legacy plan
entity) MUST fall through only to Level 3, which checks the plan-owned documents. Level 4 is
skipped entirely for standalone features and for features under legacy plan parents.

> This rule preserves backward compatibility for the current three-level lookup chain when
> the parent is a plan (or absent).

### FR-017 — Level 3 and Level 4 reason strings

When a gate is satisfied by Level 3 (parent batch documents), the `GateResult.Reason` MUST
identify the source as the parent batch. When satisfied by Level 4 (grandparent plan
documents), the `Reason` MUST identify the source as the grandparent plan.

**Examples:**

- Level 3: `"design document DOC-xxx owned by parent batch is approved"`
- Level 4: `"design document DOC-xxx owned by grandparent plan is approved"`

---

## Non-Functional Requirements

**NFR-001 — No breaking change to three-level tests**
All existing tests for `evalOneDocument` (including
`TestEvalDocuments_SatisfiedByParentPlanDoc` and `TestEvalDocuments_NoParentSkipsLevel3`)
MUST continue to pass without modification after this feature's changes are applied. New test
cases MUST be added for Level 3 (batch) and Level 4 (grandparent plan) scenarios.

**NFR-002 — No duplicate display IDs within a batch**
Within a single batch, no two features MAY share the same `display_id` value. The write
ordering in FR-004 enforces this in the single-writer case. In the concurrent multi-writer
case, a Git merge conflict on `next_feature_seq` is the expected resolution mechanism.

**NFR-003 — Backward compatibility: canonical TSID input**
Canonical `FEAT-{TSID13}` identifiers MUST continue to work as input in all entity tools,
unchanged from current behaviour.

**NFR-004 — Backward compatibility: break-hyphen TSID input**
The TSID break-hyphen display form (e.g. `FEAT-01KMK-RQRRX3CC`) MUST continue to work as
input in all entity tools, unchanged from current behaviour.

**NFR-005 — No state filename changes**
This feature MUST NOT change the filenames of any existing `.kbz/state/` files. The canonical
filename for a feature state file remains `FEAT-{TSID13}-{slug}.yaml`.

**NFR-006 — Gate performance**
The four-level lookup MUST add at most one additional synchronous document-store read
compared with the existing three-level lookup when Level 4 is actually reached. When Level 3
is satisfied, Level 4 MUST NOT be queried (short-circuit, FR-015).

---

## Scope

### In scope

- `internal/model/entities.go`: The batch model (as produced by P38-F3) carries
  `NextFeatureSeq int` (`yaml: next_feature_seq`). The Feature model carries
  `DisplayID string` (`yaml: display_id,omitempty`) as defined by P37-F1; this feature
  ensures the value is set to `B{n}-F{m}` when the parent is a batch.
- `internal/service/entities.go` (and/or batch service): `CreateFeature` counter allocation
  branches on whether the parent is a batch or a legacy plan, applying the correct counter
  and format string.
- `internal/gate/eval_documents.go`: `evalOneDocument` extended to four levels.
- `internal/gate/evaluator.go`: `PrereqEvalContext` extended with a parent-entity resolver
  (FR-014).
- Input resolution (entity resolution layer): recognises `B{n}-F{m}` as a valid feature
  display ID pattern.
- MCP tool output: `display_id` in `B{n}-F{m}` form where applicable.
- Unit and integration tests for all new behaviour.

### Out of scope

- Migration of existing features from `P{n}-F{m}` to `B{n}-F{m}`. Migration is the
  responsibility of the P38 migration feature and is not specified here.
- Migration of existing plan state files to batch state files (P38-F3).
- `depends_on` enforcement in the plan or batch model (deferred per design D2).
- Status dashboard changes to display batch-scoped feature IDs (the `display_id` field on
  the feature is sufficient; the status tool reads it directly).
- CLI table-view changes beyond what uses the `display_id` field surfaced by MCP tools.

---

## Acceptance Criteria

**AC-001 (FR-001, FR-002):**
Given a `CreateBatch` call completes successfully,
when the resulting batch state file is read from disk,
then the YAML contains a `next_feature_seq` field with integer value `1`.

**AC-002 (FR-003):**
Given a batch state file contains `next_feature_seq: N`,
when a feature is successfully created under that batch,
then reading the batch state file from disk shows `next_feature_seq: N+1`.

**AC-003 (FR-005, FR-006):**
Given a feature is created under batch `B24-auth-system` when the batch's
`next_feature_seq` is `3`,
when the feature state file is read from disk,
then the YAML contains `display_id: B24-F3`.

**AC-004 (FR-004):**
Given a test harness that injects a fault immediately after writing the batch's incremented
`next_feature_seq` but before writing the feature state file,
when the test inspects all feature state files under the batch,
then no feature with `display_id: B24-F{N}` exists, and the batch's `next_feature_seq`
equals `N+1`, confirming a sequence gap rather than a duplicate allocation.

**AC-005 (FR-007):**
Given an existing plan `P37-file-names-and-actions` with `next_feature_seq: 5` that has not
been migrated to a batch,
when a feature is created under that plan,
then the feature state file contains `display_id: P37-F5` (not `B37-F5`), and the plan's
`next_feature_seq` is incremented to `6`.

**AC-006 (FR-008, FR-009):**
Given a feature with canonical ID `FEAT-01KMKRQRRX3CC` has `display_id: B24-F3`,
when `entity get` is called with id `B24-F3` or `b24-f3`,
then the response contains the same entity data as calling `entity get FEAT-01KMKRQRRX3CC`.

**AC-007 (FR-010 — entity update):**
Given a feature with `display_id: B24-F2`,
when `entity update` is called with id `B24-F2` and a new `summary` value,
then the feature's state file on disk reflects the updated summary.

**AC-008 (FR-010 — entity transition):**
Given a feature with `display_id: B24-F3` in `proposed` status,
when `entity transition` is called with id `B24-F3` and target status `designing`,
then the feature's status is updated to `designing` in its state file.

**AC-009 (FR-010 — entity list):**
Given a feature with `display_id: B24-F1`,
when `entity list` is called with `B24-F1` supplied as an ID filter,
then the response contains exactly that feature.

**AC-010 (FR-011):**
Given a feature with `display_id: B24-F1`,
when any MCP tool response includes a reference to that feature,
then the JSON payload contains a `display_id` field with value `B24-F1`.

**AC-011 (FR-012 — Level 3: parent batch):**
Given:
- A feature with no `design` field reference and no feature-owned design documents,
- The feature's parent is batch `B24-auth-system`,
- Batch `B24-auth-system` has an approved design document `DOC-batch-design`,
when the design gate is evaluated for the feature,
then the gate result is satisfied, with a reason identifying `DOC-batch-design` as the
parent batch document.

**AC-012 (FR-012 — Level 4: grandparent plan):**
Given:
- A feature with no design field reference, no feature-owned design documents, and no
  batch-owned design documents,
- The feature's parent is batch `B24-auth-system`,
- Batch `B24-auth-system` has a `parent` field referencing plan `P1-social-platform`,
- Plan `P1-social-platform` has an approved design document `DOC-plan-design`,
when the design gate is evaluated for the feature,
then the gate result is satisfied, with a reason identifying `DOC-plan-design` as the
grandparent plan document.

**AC-013 (FR-015 — short-circuit: Level 3 satisfied, Level 4 skipped):**
Given:
- A feature with no feature-owned design documents,
- The feature's parent batch has an approved design document,
- The batch's parent plan also has an approved design document,
when the design gate is evaluated,
then the gate result is satisfied by Level 3, and the Level 4 plan document query is NOT
executed (verified by a mock that asserts the plan owner is never queried).

**AC-014 (FR-016 — standalone feature uses three-level lookup):**
Given a feature whose `parent` field is empty (no parent),
when the design gate is evaluated and no feature-level documents exist,
then Level 4 is not reached and the result is unsatisfied (same as current behaviour for
features with no parent).

**AC-015 (FR-017 — reason strings):**
Given a gate satisfied by Level 3,
then `GateResult.Reason` contains the substring `"parent batch"`.

Given a gate satisfied by Level 4,
then `GateResult.Reason` contains the substring `"grandparent plan"`.

**AC-016 (NFR-001 — existing tests still pass):**
The tests `TestEvalDocuments_SatisfiedByParentPlanDoc`,
`TestEvalDocuments_NoParentSkipsLevel3`, and all other existing `eval_documents` tests pass
without modification after this feature's changes are applied.

---

## Dependencies and Assumptions

### Dependencies

| Feature | Why |
|---------|-----|
| **P38-F1** (config schema, batch prefix registry) | Provides the `batch_prefixes` registry and `IsBatchID` / `ParseBatchID` functions needed to identify whether a parent entity is a batch and to extract its prefix character and number for the display ID format string. |
| **P38-F2** (plan entity) | Provides the new plan entity with its `parent` field. Level 4 gate lookup reads the batch's `parent` field and checks whether it references a plan entity. |
| **P38-F3** (batch entity) | Renames the current plan entity to batch; defines the batch state file schema including `next_feature_seq`. All counter allocation in this feature targets the batch state file. |
| **P37-F1** (plan-scoped feature display IDs) | Adds `next_feature_seq` to plan state files, `display_id` to feature state files, and wires display ID allocation into `CreateFeature`. This feature builds directly on those foundations, updating the format string and counter source for batch parents. |

### Assumptions

1. By the time this feature is implemented, P37-F1 has been merged and `model.Feature.DisplayID`
   and `model.Plan.NextFeatureSeq` fields exist in the codebase. This feature does not
   re-introduce those fields; it modifies where the counter lives (batch instead of plan) and
   what format string is used for the value.

2. P38-F3 has defined a `Batch` model (or adapted `Plan` model under the `batch` entity kind)
   with a `NextFeatureSeq int` field and a `Parent string` field. `CreateFeature` will be
   updated to accept batch IDs as the `parent` parameter.

3. `IsBatchID` and `ParseBatchID` utility functions (or equivalent) will be available in the
   `internal/model` or `internal/id` packages by the time this feature is implemented. These
   are needed to distinguish batch parents from plan parents and to extract `{prefix}{n}` from
   a batch ID like `B24-auth-system`.

4. The document gate evaluator's `PrereqEvalContext` is the right extension point for passing
   the parent entity resolver. No architectural changes outside `internal/gate/` are required.

5. During the transition period, a feature's `parent` field may reference either a legacy plan
   entity or a batch entity. The code distinguishes these by entity kind, not by prefix
   character alone.