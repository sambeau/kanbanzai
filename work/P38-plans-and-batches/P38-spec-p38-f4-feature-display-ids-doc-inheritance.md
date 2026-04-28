# P38-F4: Feature Display IDs and Document Inheritance — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKHK2GV                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — D3, D4, D6           |

---

## Overview

This specification defines two changes that follow from the batch entity rename
(P38-F3), as described in the P38 design document
`work/design/meta-planning-plans-and-batches.md` (D3, D4, D6).

First, feature display IDs change from `P{n}-F{n}` (plan-scoped) to `B{n}-F{n}`
(batch-scoped). Since features belong to batches (the renamed plan entity), their
short display identifier must use the batch prefix.

Second, the document gate lookup chain is extended from the current three-level lookup
(feature → feature-owned docs → parent docs) to a four-level lookup that includes the
batch's parent plan, enabling design inheritance across the plan/batch hierarchy.

---

## Scope

**In scope:**

- Update feature display ID format from `P{n}-F{n}` to `B{n}-F{n}` throughout the
  codebase
- Update the feature display ID generation to use the batch prefix registry
- Ensure all MCP tool responses, status dashboards, and log messages use `B{n}-F{n}`
- Extend document gate evaluation from three levels to four: feature-owned documents
  → parent batch documents → grandparent plan documents
- Update the document gate system to resolve a batch's parent plan and include plan-level
  design documents in the gate check
- Unit tests for the new display ID format and four-level gate lookup

**Explicitly excluded:**

- Updating existing feature records on disk from `P{n}-F{n}` to `B{n}-F{n}` (P38-F8)
- Renaming worktree folders or document filenames containing feature display IDs (P38-F8)
- Backward-compatible resolution of legacy `P{n}-F{n}` display IDs (redundant with
  P38-F3's batch-level legacy resolution)
- Changes to the canonical filename template (already handled by P37 and P38-F3)
- Denormalisation of display IDs into state files (display IDs remain computed, not
  stored)

---

## Functional Requirements

### Feature Display IDs

- **REQ-001:** When computing a feature's display ID, the system MUST use the batch prefix
  from the `batch_prefixes` config registry rather than the plan prefix. The format is
  `B{prefix}{n}-F{n}` (e.g. `B24-F3` for the third feature in batch `B24-auth-system`).

- **REQ-002:** The feature display ID MUST be derived from the batch's ID. Given a batch
  `B24-auth-system`, features created under it produce display IDs `B24-F1`, `B24-F2`,
  etc. The feature sequence number (`next_feature_seq`) is scoped to the batch.

- **REQ-003:** The `FormatFullDisplay` function (and any equivalent display formatting) MUST
  use the `B` prefix for feature display IDs when the feature's parent is a batch entity.

- **REQ-004:** All MCP tool responses that include feature display IDs MUST use the
  `B{n}-F{n}` format. This includes entity get/list, status dashboard, next, decompose,
  and finish responses.

- **REQ-005:** The format `B{n}-F{n}` applies to all new features created after P38-F3 is
  implemented. Existing features retain their current display IDs until migrated (P38-F8).

- **REQ-006:** Feature display ID format MUST NOT depend on whether the batch has a parent
  plan. A standalone batch and a batch under a plan both produce `B{n}-F{n}` feature
  display IDs.

### Document Gate Inheritance

- **REQ-007:** The document gate evaluation for a feature MUST check documents in the
  following order, stopping at the first match:
  1. Feature-owned documents (existing behaviour)
  2. Parent batch documents (existing behaviour, previously "parent plan documents")
  3. Grandparent plan documents (new — if the batch has a `parent` field referencing a
     plan ID)

- **REQ-008:** When a feature's parent batch has a `parent` field set to a plan ID, the
  gate evaluator MUST resolve that plan and include the plan's approved documents of the
  required type in the lookup chain.

- **REQ-009:** The four-level lookup MUST apply to all document gate checks: design gate,
  specification gate, and dev-plan gate. Each gate type looks for documents of its
  respective type at each level.

- **REQ-010:** If the batch has no parent plan, the gate lookup stops at level 2 (batch
  documents). The behaviour is identical to the current three-level lookup (with
  "batch" substituted for "plan" at level 2).

- **REQ-011:** A plan-level design document of the appropriate type that is approved MUST
  satisfy the corresponding gate for a feature in a child batch. For example, an approved
  design document owned by plan `P1-platform` satisfies the design gate for features in
  batch `B24-auth-system` (child of `P1-platform`).

- **REQ-012:** The gate lookup MUST return the first matching approved document found
  (nearest level first). Feature-level documents take precedence over batch-level, which
  take precedence over plan-level.

---

## Non-Functional Requirements

- **REQ-NF-001:** The four-level gate lookup MUST complete in constant time relative to the
  tree depth (three state store lookups maximum: feature, batch, plan).

- **REQ-NF-002:** When a plan is not found (e.g. batch's `parent` points to a deleted
  plan), the gate lookup MUST skip the plan level gracefully and return the result from
  the batch level. A missing plan MUST NOT cause a gate failure.

- **REQ-NF-003:** Feature display ID computation MUST NOT add measurable latency to entity
  operations. Display IDs are computed from in-memory data (batch ID prefix and feature
  sequence number); no additional I/O is needed.

---

## Constraints

- Feature display IDs are computed at runtime, not stored persistently. The
  `next_feature_seq` counter on the batch entity is the only persistent state.
- The `P{n}-F{n}` format is NOT removed for existing features during the transition. New
  features use `B{n}-F{n}`; existing features are migrated in P38-F8.
- Document gate changes are additive — the existing lookup logic is extended, not
  replaced. Level 1 and 2 behaviour is unchanged except for terminology ("batch" instead
  of "plan" at level 2).
- Plan documents used for inheritance must be registered with the document record system
  and have `owner` set to the plan ID.

---

## Acceptance Criteria

**AC-001.** Creating a feature under batch `B24-auth-system` produces a display
  ID of `B24-F1` (first feature). Creating a second feature under the same batch produces
  `B24-F2`.

**AC-002.** After creating feature `B24-F1`, the batch's `next_feature_seq`
  increments to 2. Creating the next feature produces `B24-F2`.

**AC-003.** `FormatFullDisplay` for a feature with parent batch `B24-auth`
  returns a string containing `B24-F{n}`.

**AC-004.** `entity(action: "get", id: "<feature-id>")` response includes
  `display_id` in `B{n}-F{n}` format. `status` dashboard renders features with the `B`
  prefix.

**AC-005.** An existing feature created before P38-F3 retains its current
  display ID. Only newly created features after P38-F3 use the `B{n}-F{n}` format.

**AC-006.** Creating a feature under standalone batch `B99-standalone` (no
  parent plan) produces display ID `B99-F1`.

**AC-007.** Given plan `P1-platform` → batch `B24-auth` →
  feature `FEAT-xxx`, and an approved specification owned by `P1-platform`:
  the specification gate check for the feature passes using the plan's specification.

**AC-008.** The four-level lookup works for design, specification, and dev-plan
  gate types. Each gate type looks for its corresponding document type at each level.

**AC-009.** A feature under a standalone batch (no parent plan) gates exactly
  as before: feature docs → batch docs. No error or extra lookup occurs for the missing
  plan level.

**AC-010.** An approved design document owned by plan `P1` satisfies the design
  gate for a feature in child batch `B24` when neither the feature nor the batch has its
  own design document.

**AC-011.** When a feature has its own approved design doc, and the batch also
  has an approved design doc, the gate returns the feature's document (nearest level
  wins).

**AC-012.** Gate evaluation completes in three state store lookups or fewer
  for the four-level case (feature, batch, plan).

**AC-013.** A batch with `parent: "P99-nonexistent"` (deleted plan) gates
  correctly at the batch level without errors from the missing plan.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: create features under batch, verify display IDs `B24-F1`, `B24-F2` |
| AC-002 | Test | Automated test: assert `next_feature_seq` increments correctly per batch |
| AC-003 | Test | Automated test: call `FormatFullDisplay` on a feature, assert `B{n}-F{n}` in result |
| AC-004 | Test | Automated test: query entity/stats for feature, verify display ID format |
| AC-005 | Test | Automated test: verify existing feature display IDs unchanged after P38-F3 |
| AC-006 | Test | Automated test: standalone batch feature display ID uses `B` prefix |
| AC-007 | Test | Automated test: feature under plan→batch chain resolves spec gate from plan-level doc |
| AC-008 | Test | Automated test: verify design, spec, and dev-plan gates all use four-level lookup |
| AC-009 | Test | Automated test: standalone batch feature gates at batch level, no plan lookup attempted |
| AC-010 | Test | Automated test: approved plan design doc satisfies design gate for grandchild feature |
| AC-011 | Test | Automated test: feature-level doc takes precedence over batch-level over plan-level |
| AC-012 | Inspection | Code review: verify maximum three lookups per gate evaluation |
| AC-013 | Test | Automated test: feature under batch with dangling `parent` reference gates correctly |

---

## Dependencies and Assumptions

- **P38-F3 (Batch Entity Rename):** Feature display IDs depend on the batch entity existing
  with the `B` prefix and the batch prefix registry being operational.
- **P38-F2 (Plan Entity):** The four-level gate lookup depends on the plan entity
  existing and being queryable by ID.
- **P38-F1 (Config Schema):** The `batch_prefixes` registry provides the prefix character
  used in display ID generation.
- **Document gate system (`internal/gate/`):** The gate lookup is extended from three to
  four levels. The existing feature-level and batch-level lookup logic is preserved.
- **Existing feature records:** Features created before P38-F3 may still show `P{n}-F{n}`
  display IDs. Full migration of display IDs is handled by P38-F8. The display ID
  computation for pre-existing features may produce `P{n}` until migration.
- **Plan document ownership:** Plan-level documents referenced in the inheritance chain
  must be registered with `owner: "P{n}-{slug}"` (the plan ID). This is consistent with
  existing document registration conventions.
