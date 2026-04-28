# P38-F7: Documentation and Skills Updates — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKWTBRP                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §1, §2, §3, D1, D3   |

---

## Overview

This specification defines the documentation and skills updates required to reflect the
plan/batch distinction introduced by P38. It covers updates to all agent-facing materials
referenced in `AGENTS.md` and `copilot-instructions.md`, as described in the P38 design
document `work/design/meta-planning-plans-and-batches.md`.

Every reference to the current "plan" entity (as an operational work-grouping container)
must be updated to use "batch" terminology. The new recursive "plan" entity must be
documented alongside the renamed "batch" entity, with clear guidance on when to use each.

---

## Scope

**In scope:**

- `.kbz/roles/` — update all role YAML files that reference "plan" as work container
- `.kbz/skills/` — update all skill files with plan/batch terminology
- `.kbz/stage-bindings.yaml` — update entity type references
- `AGENTS.md` — update entity hierarchy, conventions, and vocabulary
- `.github/copilot-instructions.md` — update entity references in the instruction table
- `refs/` directory — update reference files that reference plan entities
- Any templates that reference plan entity types

**Explicitly excluded:**

- Creating new documentation (this is a terminology update, not new content)
- Writing plan-level documentation (plans are human-managed; their documentation is
  project-specific)
- Updating code comments or Go documentation (handled in respective code features)
- Migrating existing document files (P38-F8)

---

## Functional Requirements

### Roles (`.kbz/roles/`)

- **REQ-001:** Every role YAML file that references the current "plan" entity in the
  context of work grouping (features, tasks, execution) MUST use "batch" terminology.

- **REQ-002:** Roles that reference the entity hierarchy MUST list both plan (strategic)
  and batch (operational) entities where the hierarchy is described.

- **REQ-003:** The `orchestrator` role MUST reference "batch" for work coordination and
  "plan" for strategic scope. The orchestrator's vocabulary section MUST include both
  "batch" and "plan" terms.

- **REQ-004:** The `implementer-go` role vocabulary MUST be updated to reference "batch"
  where it previously referenced "plan" for feature ownership.

### Skills (`.kbz/skills/`)

- **REQ-005:** Every `SKILL.md` file that references "plan" as the parent of features
  MUST use "batch" terminology for that relationship.

- **REQ-006:** The `write-design` skill MUST document that design documents can be owned
  by either a plan (strategic design) or a batch (operational design).

- **REQ-007:** The `write-spec` skill MUST reference "batch" as the parent entity for
  features in the scope and dependency sections.

- **REQ-008:** The `orchestrate-development` skill MUST use "batch" for the work
  container that holds features being developed.

- **REQ-009:** The `orchestrate-review` skill MUST reference batches for feature-scoped
  reviews.

- **REQ-010:** The `decompose-feature` skill MUST reference batches as the owning entity
  for features being decomposed.

### Stage Bindings (`.kbz/stage-bindings.yaml`)

- **REQ-011:** Stage bindings that reference plan entities for ownership or prerequisites
  MUST be updated to use correct plan/batch terminology.

- **REQ-012:** Any stage prerequisites that reference "plan" documents for gate
  evaluation MUST specify whether the prerequisite applies at the plan level, batch
  level, or both.

### Agent Instructions (`AGENTS.md`, `copilot-instructions.md`)

- **REQ-013:** `AGENTS.md` MUST document the new entity hierarchy: plan (recursive,
  strategic) → batch (operational) → feature → task. The entity relationship diagram
  and vocabulary section must be updated.

- **REQ-014:** `AGENTS.md` MUST include guidance on when to create a plan vs a batch:
  plans for strategic decomposition, batches for executable work grouping.

- **REQ-015:** `.github/copilot-instructions.md` MUST update the skills table, roles
  table, and any entity references to use "batch" where appropriate.

- **REQ-016:** All references to "plan" in copilot instructions that mean the
  work-grouping entity MUST change to "batch." References to the new planning entity
  use "plan."

### Reference Files (`refs/`)

- **REQ-017:** `refs/go-style.md` — if it references plan entities, update to
  plan/batch distinction.

- **REQ-018:** `refs/testing.md` — if test conventions reference plan entities, update
  terminology.

- **REQ-019:** `refs/sub-agents.md` — if sub-agent delegation references plan entities,
  update to plan/batch distinction.

- **REQ-020:** `refs/document-map.md` — update the document-to-topic map to reflect
  plan-owned and batch-owned document types.

- **REQ-021:** `refs/knowledge-graph.md` — update knowledge graph usage references for
  plan and batch entities.

---

## Non-Functional Requirements

- **REQ-NF-001:** All documentation changes MUST be terminology-only. No procedural
  changes, no new rules, and no removed rules are introduced by this feature.

- **REQ-NF-002:** Consistency check: after this feature, a grep for "plan" in the
  updated files MUST return zero false positives where "batch" is the correct term.
  References to "plan" should only refer to the strategic planning entity.

- **REQ-NF-003:** File-level consistency: every file listed in the scope that references
  either entity type MUST use the terms consistently throughout. A single file must
  not mix "plan" (as work container) and "batch" for the same concept.

---

## Constraints

- This is a terminology update only. No new content, no new procedures, no workflow
  changes are introduced.
- The `plan` term is retained for the strategic planning entity. It is not removed from
  the vocabulary.
- `AGENTS.md` and `copilot-instructions.md` are the canonical entry points for agents.
  Updates to these files must maintain structural compatibility with the Kanbanzai
  agent onboarding flow.
- Role and skill files use YAML frontmatter or markdown structure that must be preserved.
  Only the text content changes.

---

## Acceptance Criteria

**AC-001.** Every `.kbz/roles/*.yaml` file has been reviewed. "Plan" as
  work container is replaced with "batch." The term "plan" appears only in the context
  of the strategic planning entity.

**AC-002.** Every `.kbz/skills/**/SKILL.md` file has been reviewed and
  updated consistently.

**AC-003.** `write-design/SKILL.md` documents that design docs can live at
  either the plan level or batch level.

**AC-004.** `.kbz/stage-bindings.yaml` references correct entity types for
  document ownership and gate prerequisites.

**AC-005.** `AGENTS.md` entity hierarchy section shows:
  Plan → Batch → Feature → Task (with descriptions of each).

**AC-006.** `AGENTS.md` includes guidance distinguishing when to use a plan
  vs a batch.

**AC-007.** `.github/copilot-instructions.md` skills and roles tables use
  correct plan/batch terminology.

**AC-008.** All reference files in `refs/` use consistent
  plan/batch terminology.

**AC-009.** A grep for `\bplan\b` (case-insensitive) in the updated files
  returns only references to the strategic planning entity, not the work-grouping
  entity.

**AC-010.** No single file uses both "plan" and "batch" to refer to the
  same entity type.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Manual review of all `.kbz/roles/*.yaml` files for correct terminology |
| AC-002 | Inspection | Manual review of all `.kbz/skills/**/SKILL.md` files |
| AC-003 | Inspection | Verify `write-design/SKILL.md` mentions both plan and batch design ownership |
| AC-004 | Inspection | Verify `.kbz/stage-bindings.yaml` entity references |
| AC-005 | Inspection | Verify `AGENTS.md` entity hierarchy section |
| AC-006 | Inspection | Verify `AGENTS.md` plan-vs-batch guidance |
| AC-007 | Inspection | Verify `.github/copilot-instructions.md` tables |
| AC-008 | Inspection | Verify all `refs/*.md` files |
| AC-009 | Test | Automated grep for incorrect "plan" usage across all updated files |
| AC-010 | Inspection | Per-file consistency check — no mixed usage within a file |

---

## Dependencies and Assumptions

- **P38-F2 (Plan Entity):** The new plan entity's lifecycle and semantics must be stable
  before documentation can accurately describe it.
- **P38-F3 (Batch Entity Rename):** The batch entity must be established before
  documentation references to "batch" make sense.
- **All other P38-F* features:** Documentation is typically the last feature in a plan,
  written after the system behaviour is stable. This specification assumes F2–F6 are
  complete or sufficiently stable that their semantics are known.
- **Existing documentation standards:** Role and skill files follow established YAML and
  Markdown conventions. This feature does not change those conventions.
