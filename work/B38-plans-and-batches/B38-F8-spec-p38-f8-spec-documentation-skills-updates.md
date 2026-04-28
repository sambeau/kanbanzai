# P38-F8: Documentation and Skills Updates — Specification

| Field   | Value |
|---------|-------|
| Date    | 2026-04-27 |
| Status  | draft |
| Feature | FEAT-01KQ7YQKWTBRP |
| Design  | P38 Meta-Planning: Plans and Batches — §8, D1, D3 |

---

## Related Work

| Document | Relevance |
|----------|-----------|
| `work/design/meta-planning-plans-and-batches.md` §8 | Interaction with existing features — skills and tool descriptions must be updated |
| `work/design/meta-planning-plans-and-batches.md` D1 | Decision: tools, skills, and agent instructions must be updated to use the new terminology |
| `work/design/meta-planning-plans-and-batches.md` D3 | Decision: batches retain all current plan functionality; standalone batches work identically to today |
| `work/design/meta-planning-plans-and-batches.md` D4 | Decision: batch IDs use `B{n}-{slug}` prefix; plan IDs keep `P{n}-{slug}` |
| AGENTS.md (dual-write rule) | `.agents/skills/kanbanzai-<name>/SKILL.md` changes must be mirrored to `internal/kbzinit/skills/<name>/SKILL.md` in the same commit |

---

## Overview

P38 introduces two distinct entity types to replace the single overloaded "plan" entity:

- **plan** — a recursive strategic planning entity (lifecycle: `idea → shaping → ready → active → done`)
- **batch** — the renamed current plan entity; a unit of execution work (lifecycle: `proposed → designing → active → reviewing → done`)

Features belong to batches, not plans. Batches optionally belong to plans.

All agent-facing documentation that currently uses "plan" in the context of work execution (grouping features, tracking implementation, managing lifecycle state) must be updated to use "batch". Documentation about strategic direction, scope decomposition, and roadmaps uses "plan". The document type "dev-plan" is unchanged — it is a document type, not an entity reference.

This feature covers documentation, roles, and skills files only. No code changes.

---

## Vocabulary Rule

The following vocabulary rule applies across all updated files:

| Term | Meaning | Notes |
|------|---------|-------|
| **plan** | Strategic entity — scope decomposition, roadmap, long-term direction | New entity; lifecycle: `idea → shaping → ready → active → done`; ID prefix `P{n}` |
| **batch** | Execution entity — groups features for implementation, has agents work against it | Renamed current plan; lifecycle unchanged; ID prefix `B{n}` |
| **dev-plan** | Document type — the implementation plan document attached to a feature | Unchanged; still called "dev-plan"; not an entity reference |

When in doubt: if the sentence would have worked with the current "plan" entity (features belong to it, it has a proposed→done lifecycle, agents work against it), that concept is now "batch".

---

## Functional Requirements

### FR-001 — Vocabulary rule enforcement

All updated files MUST apply the vocabulary rule above consistently. No file MUST use "plan" to mean the execution work-grouping entity (current plan / future batch) after the update.

### FR-002 — `.kbz/stage-bindings.yaml` stage key rename

The `plan-reviewing` stage key MUST be renamed to `batch-reviewing`.

The description MUST be updated from "Reviewing a completed plan for aggregate delivery" to "Reviewing a completed batch for aggregate delivery".

The `notes` field MUST be updated to replace "at the plan level" with "at the batch level".

### FR-003 — `.kbz/skills/orchestrate-development/SKILL.md` Phase 0

The Phase 0 section header `_(plans with more than 3 features only)_` MUST be updated to `_(batches with more than 3 features only)_`.

All occurrences of "the plan" in Phase 0 body text (where referring to the entity containing features) MUST be replaced with "the batch".

### FR-004 — `.kbz/skills/orchestrate-development/SKILL.md` Close-Out

Phase 6 Close-Out step 7 reads "If this plan has a merge schedule…". This MUST be updated to "If this batch has a merge schedule…".

### FR-005 — `.agents/skills/kanbanzai-workflow/SKILL.md` stage gates table

The stage gates table row for the **Features** stage currently reads "Plan + Feature entities" under "What it produces". This MUST be updated to "Batch + Feature entities".

### FR-006 — `.agents/skills/kanbanzai-workflow/SKILL.md` resuming section

The section `## Resuming an in-flight plan` MUST be renamed to `## Resuming an in-flight batch`.

The section body MUST replace all references to "a plan that was already in progress" (and equivalent phrasings) with "a batch that was already in progress".

The entity ID example `PLAN-xxx` MUST be updated to `BATCH-xxx` throughout this section.

Step 2 ("Check the plan lifecycle state") MUST be updated to "Check the batch lifecycle state", and the example entity ID and status check commands MUST reference batch IDs and the batch lifecycle (`proposed`, `designing`, `active`).

### FR-007 — `.agents/skills/kanbanzai-workflow/SKILL.md` entity lifecycle reference

The sentence "For the legal state transitions for each entity type (feature, task, bug, plan)" MUST be updated to list both `plan` and `batch` as entity types: "(feature, task, bug, plan, batch)".

### FR-008 — `.agents/skills/kanbanzai-workflow/SKILL.md` Emergency Brake

The Emergency Brake item "You are about to create Plan, Feature, or Task entities" MUST be updated to include Batch: "Plan, Batch, Feature, or Task entities".

### FR-009 — `.agents/skills/kanbanzai-workflow/SKILL.md` example

The correct stage gate check example references `work/plan/feature-x-plan.md`. This path refers to a dev-plan document (not the plan entity). The inline comment MUST clarify this is a dev-plan document, not a plan entity, to prevent confusion under the new terminology.

### FR-010 — `.agents/skills/kanbanzai-planning/SKILL.md` vocabulary table

The vocabulary table MUST be extended to include entries for both the strategic plan entity and the operational batch entity. At minimum the following terms MUST be defined or updated:

- **plan** — the strategic entity; represents scope decomposition and long-term direction; recursive; lifecycle: `idea → shaping → ready → active → done`
- **batch** — the execution entity; groups features for delivery; replaces what was previously called "plan"; lifecycle: `proposed → designing → active → reviewing → done`
- **plan document** (existing term) — MUST be updated or replaced with **batch document** to reflect that the coordinating document for a group of features is now associated with a batch, not a plan

### FR-011 — `.agents/skills/kanbanzai-planning/SKILL.md` scope decisions section

The `### Feature vs. Plan` section MUST be replaced with two sections:

1. `### Feature vs. Batch` — covering when work is a single feature vs. a batch (previously "Feature vs. Plan"). Retains all current sizing signals and guidance, updated to use "batch".

2. `### Batch vs. Plan` — new section covering when to create a standalone batch vs. when to also create a strategic plan above it. MUST document:
   - A batch alone is sufficient for most work
   - A plan is warranted when multiple batches serve a shared strategic goal
   - Plans are optional — batches can exist with no parent plan
   - "Err towards fewer plans" guidance (strategic plans, not batches)

### FR-012 — `.agents/skills/kanbanzai-planning/SKILL.md` entity creation guidance

The guidance "Before creating any Plan or Feature entities" MUST be updated to include Batch: "Before creating any Plan, Batch, or Feature entities".

### FR-013 — `.agents/skills/kanbanzai-planning/SKILL.md` anti-patterns

The `### Monolithic Feature` anti-pattern currently advises splitting into features; its guidance to "split along vertical slices" and "each feature should have its own spec and worktree" is unchanged. However, any references to creating a "plan" as the grouping unit MUST be updated to "batch".

The `### Scope Creep in Planning` anti-pattern currently reads "Limit the plan to agreed scope." Any phrasing that refers to the execution work container MUST be updated to "batch". References to creating or growing the strategic planning layer MAY retain "plan".

### FR-014 — `.agents/skills/kanbanzai-planning/SKILL.md` examples

The planning output example that ends with "Proceed to design" MUST be reviewed. Any phrasing that implies a "plan" entity will be created as the work-grouping unit MUST be updated to use "batch".

The examples section MUST include at least one example that distinguishes between creating a batch (for immediate feature work) and creating a plan (for strategic decomposition).

### FR-015 — `.agents/skills/kanbanzai-getting-started/SKILL.md` resuming section

The section `## Resuming an in-flight plan` MUST be renamed to `## Resuming an in-flight batch`.

All steps within that section that reference plan lifecycle states (`proposed`, `designing`) or plan-entity transitions MUST be updated to use batch terminology and the `B{n}-{slug}` ID format.

Specifically: the `status(id: "P<N>-<slug>")` call example MUST be updated to `status(id: "B<N>-<slug>")`.

The `entity(action: "transition")` override examples MUST reference the batch lifecycle (`proposed → designing → active`), not a plan lifecycle.

### FR-016 — `.agents/skills/kanbanzai-getting-started/SKILL.md` vocabulary

The vocabulary term **Feature lifecycle state** lists examples `designing`, `specifying`, `implementing`. This is a feature lifecycle, not plan/batch, and does not need updating.

However, any vocabulary term or inline explanation that conflates the current "plan" (work-grouping) with the new "plan" (strategic) MUST be disambiguated.

### FR-017 — `.agents/skills/kanbanzai-agents/SKILL.md` entity names

The Entity Names section currently reads: "Do not include the parent plan or feature name in the entity name". This MUST be updated to: "Do not include the parent batch or feature name in the entity name (or parent plan name, if one exists)".

The Entity Names examples table MUST add a **Batch** example row alongside the existing **Plan** example row.

### FR-018 — `.github/copilot-instructions.md` stage bindings table

The stage bindings table row for `plan-reviewing` MUST be updated to `batch-reviewing` (both the stage name and the Role/Stage columns).

### FR-019 — `.github/copilot-instructions.md` plan/batch distinction summary

The copilot-instructions file MUST include a concise vocabulary summary explaining the plan/batch distinction, placed where agents can encounter it during orientation. The summary MUST cover:

- plan = strategic layer (lifecycle: `idea → shaping → ready → active → done`)
- batch = execution layer, replaces current plan (lifecycle: `proposed → designing → active → done`)
- dev-plan = document type, unchanged
- Features belong to batches; batches optionally belong to plans

### FR-020 — `AGENTS.md` repository structure

The repository structure section lists `.kbz/state/plans/ ← Plan entity files`. This MUST be updated to `.kbz/state/batches/ ← Batch entity files`. A new line MUST be added for `.kbz/state/plans/ ← Plan entity files` (strategic planning entities).

The `service/` directory description "entity, plan, and document record service logic" MUST be updated to "entity, batch, plan, and document record service logic".

### FR-021 — `refs/document-map.md` plan review skill entry

The Key Design Documents table entry "Plan review SKILL (procedure + checklist)" currently links to `.kbz/skills/review-plan/SKILL.md`. This MUST be updated to "Batch review SKILL (procedure + checklist)" with the same path, reflecting that the skill reviews a completed batch.

### FR-022 — `refs/document-map.md` P38 design entry

The document map MUST include an entry for `work/design/meta-planning-plans-and-batches.md` under Key Design Documents, with the topic "Plan and batch entity model; strategic vs. execution layer".

### FR-023 — Dual-write rule compliance

For each `.agents/skills/kanbanzai-<name>/SKILL.md` file updated by FR-005 through FR-017, the corresponding file under `internal/kbzinit/skills/<name>/SKILL.md` MUST receive the identical changes in the same commit.

The affected mirror pairs are:

| Source | Mirror |
|--------|--------|
| `.agents/skills/kanbanzai-workflow/SKILL.md` | `internal/kbzinit/skills/workflow/SKILL.md` |
| `.agents/skills/kanbanzai-planning/SKILL.md` | `internal/kbzinit/skills/planning/SKILL.md` |
| `.agents/skills/kanbanzai-getting-started/SKILL.md` | `internal/kbzinit/skills/getting-started/SKILL.md` |
| `.agents/skills/kanbanzai-agents/SKILL.md` | `internal/kbzinit/skills/agents/SKILL.md` |

### FR-024 — Preserve dev-plan terminology

No file updated under this feature MUST change any occurrence of "dev-plan" (the document type). The phrase "dev-plan" refers to a document, not an entity, and is correct both before and after P38.

Similarly, the skill names `write-dev-plan` and the stage binding `dev-planning` MUST NOT be altered. These refer to the activity of writing a dev-plan document, not to any plan entity.

### FR-025 — Preserve legitimate "plan" usage

Occurrences of "plan" that legitimately refer to the new strategic plan entity MUST NOT be changed. Examples of legitimate uses that survive the update:

- Creating a plan entity above a batch
- The plan's lifecycle: `idea → shaping → ready → active → done`
- "Before creating any Plan, Batch, or Feature entities" (the "Plan" here is the strategic entity)
- `kanbanzai-planning` skill name and triggers (the skill governs both planning and batch-scoping conversations)

---

## Non-Functional Requirements

### NFR-001 — Consistency within each file

Each updated file MUST be internally consistent. After update, no file MUST use both "plan" (in execution context) and "batch" for the same concept.

### NFR-002 — No new guidance invented

Updates MUST reflect the design decisions in `work/design/meta-planning-plans-and-batches.md`. No new workflow rules, lifecycle states, or policy decisions MUST be introduced beyond what the design specifies.

### NFR-003 — Commit discipline

All changes to a given agent skill and its mirror MUST be committed in the same commit (per the dual-write rule in AGENTS.md).

Cross-cutting terminology changes across multiple files (e.g. updating all "plan" → "batch" in execution context) MUST be committed as a single coherent commit, not scattered across multiple partial commits.

---

## Scope

**In scope:**
- `.kbz/stage-bindings.yaml`
- `.kbz/skills/orchestrate-development/SKILL.md`
- `.agents/skills/kanbanzai-workflow/SKILL.md` and its mirror
- `.agents/skills/kanbanzai-planning/SKILL.md` and its mirror
- `.agents/skills/kanbanzai-getting-started/SKILL.md` and its mirror
- `.agents/skills/kanbanzai-agents/SKILL.md` and its mirror
- `.github/copilot-instructions.md`
- `AGENTS.md`
- `refs/document-map.md`

**Out of scope:**
- Code changes (covered by P38-F1 through P38-F7)
- `.kbz/skills/write-dev-plan/SKILL.md` — "plan" in this file refers to the dev-plan document type; no changes needed
- `.kbz/skills/decompose-feature/SKILL.md` — the single "plan" occurrence is in a trigger string ("plan feature tasks"); may be reviewed but is low priority
- `.kbz/skills/implement-task/SKILL.md` — no plan/batch references found
- `.kbz/roles/` — no execution-context plan references found in the architect, orchestrator, reviewer-conformance, or other role files
- `internal/kbzinit/skills/plan-review/SKILL.md` — "plan-review" is a skill name for reviewing a completed batch; the file name need not change but its content SHOULD be reviewed for internal terminology
- Historical documents under `work/bootstrap/`, `work/plan/`, `work/spec/` — these predate P38 and are archival; updating them is not required

---

## Acceptance Criteria

- [ ] `git grep -n "plan" .kbz/stage-bindings.yaml` returns no lines where "plan" refers to the execution work-grouping entity
- [ ] The `batch-reviewing` key exists in `.kbz/stage-bindings.yaml` and `plan-reviewing` does not
- [ ] `kanbanzai-workflow/SKILL.md` and its mirror both contain "Resuming an in-flight batch" and do not contain "Resuming an in-flight plan"
- [ ] `kanbanzai-planning/SKILL.md` and its mirror both contain a `### Feature vs. Batch` section and a `### Batch vs. Plan` section; `### Feature vs. Plan` is absent
- [ ] `kanbanzai-planning/SKILL.md` vocabulary table defines both "plan" (strategic) and "batch" (execution) with their respective lifecycles
- [ ] `kanbanzai-getting-started/SKILL.md` and its mirror reference `B<N>-<slug>` batch ID format in the resuming section, not `P<N>-<slug>`
- [ ] `kanbanzai-agents/SKILL.md` and its mirror Entity Names section says "parent batch or feature name" (not "parent plan or feature name")
- [ ] `.github/copilot-instructions.md` contains a plan/batch vocabulary summary covering all four points from FR-019
- [ ] `AGENTS.md` repository structure lists both `.kbz/state/batches/` and `.kbz/state/plans/`
- [ ] `refs/document-map.md` contains an entry for the P38 meta-planning design document
- [ ] No updated file contains the string "dev-plan" changed to "dev-batch" or any similar corruption of the document type name
- [ ] For every `.agents/skills/kanbanzai-*/SKILL.md` updated, `git log --oneline` shows the mirror file was changed in the same commit

---

## Dependencies and Assumptions

### Dependencies

- **P38-F1 (plan entity model)** — the new plan entity and lifecycle must be defined before documentation can describe them accurately. This feature may be written in parallel with F1–F7 but its changes SHOULD be reviewed after the code features are merged to ensure accuracy.
- **P38-F3 (batch entity model)** — the batch lifecycle states and ID format must be confirmed before updating skill files that reference them.
- `work/design/meta-planning-plans-and-batches.md` must be approved (it is).

### Assumptions

- The dual-write rule (AGENTS.md) applies: `.agents/skills/kanbanzai-*/SKILL.md` changes must be mirrored to `internal/kbzinit/skills/`.
- The `internal/kbzinit/skills/plan-review/SKILL.md` mirror path does not change its filename even though the stage key changes to `batch-reviewing`; the mapping is by convention, not enforced by tooling.
- The `work/plan/` directory (containing dev-plan documents) keeps its current name; it holds document files, not entity state files, and the directory name is not governed by entity naming conventions.
- No new workflow policy decisions are required; all terminology choices are resolved by the design document.