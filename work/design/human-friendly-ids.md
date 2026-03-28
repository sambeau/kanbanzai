# Design Proposal: Human-Friendly ID Display

- Status: proposal
- Date: 2026-03-28
- Author: orchestrator
- Related: `work/reports/kanbanzai-2.0-workflow-retrospective.md` §3.4 (R4)
- Retro item: R4 — Human-Friendly ID Display (P1, medium severity)

---

## 1. Problem

Entity IDs are hard to read, remember, and communicate. Three distinct problems
contribute:

**1.1 — The split format is implemented but not surfaced.**
Every entity already has a `display_id` field containing the planned hyphenated
split format (`FEAT-01KMR-X1SEQV49`). The retrospective noted this as unimplemented
(R4), but it was added as part of Kanbanzai 2.0. The problem is that tool outputs
and agent references consistently use the unsplit `id` field instead of
`display_id`, so the work was done but has had no user-visible effect.

**1.2 — Slugs are present on every entity but rarely shown.**
Every feature and task carries a slug — `policy-and-documentation-updates`,
`update-agents-md` — but it only appears in file paths and entity YAML. It is
almost never shown alongside an ID in tool outputs, status dashboards, or agent
references. A reader seeing `FEAT-01KMRX1SEQV49` in a tool response cannot tell
what feature that is without a separate lookup.

**1.3 — There is no short human-assigned label.**
In practice, features and tasks acquire informal names from the plan document:
"Feature G", "Track C", "Phase 2 Task 3". These labels are meaningful, memorable,
and how humans naturally navigate the work. They do not exist as a system concept.
The only way to use them is informally in conversation and planning documents,
which means the tool layer cannot surface or search by them.

The combined effect: a status table full of opaque TSIDs that requires constant
cross-referencing with plan documents, and a text conversation that mixes
human-friendly label prose with machine-generated TSID references.

---

## 2. Proposed Changes

Three changes, independent but complementary.

### Change 1 — Surface `display_id` everywhere (no data model change)

Make the split format the canonical display ID in all tool outputs. Concretely:

- All `entity` responses surface `display_id` as the primary identifier, with the
  unsplit `id` demoted to a secondary field (still present, still accepted as
  input).
- The `status` dashboard, `next`, `handoff`, `finish`, and all list views use
  `display_id` in tables and attention items.
- Agent instructions (AGENTS.md) note that `display_id` is the preferred form for
  all references in prose and in tool calls.
- Both forms continue to be accepted as input — no breaking change.

**Before:**
```
FEAT-01KMRX1SEQV49
TASK-01KMRXK75ZA20
```

**After:**
```
FEAT-01KMR-X1SEQV49
TASK-01KMR-XK75ZA20
```

The hyphen lets the eye anchor on the variable suffix. No data change; purely a
display promotion.

---

### Change 2 — Always show slug alongside the display ID (no data model change)

Establish a consistent display convention: wherever a feature or task ID appears
in a tool response, show the slug in parentheses immediately after.

**Format:** `FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)`

Apply this in:
- `status` dashboard feature and task rows
- `next` queue listings
- `entity get` primary header
- `entity list` table rows
- `handoff` task header
- `finish` completion summary
- All `side_effects` entries that reference an entity

When a label is also present (Change 3), prefer the label over the full slug in
space-constrained contexts:

`FEAT-01KMR-X1SEQV49 (G · policy-and-documentation-updates)`

Slugs are already on every entity. This is a rendering change only.

---

### Change 3 — Add an optional `label` field to features and tasks

Add a short, human-assigned label field. It is optional, free text, and has no
semantic effect on the system — it is purely a navigation and display aid.

**Schema addition:**

```yaml
# Feature entity
id: FEAT-01KMRX1SEQV49
slug: policy-and-documentation-updates
label: "G"                    # optional; set by the human or orchestrator
status: done
...
```

```yaml
# Task entity
id: TASK-01KMRXK75ZA20
slug: update-agents-md
label: "G-1"                  # optional
parent_feature: FEAT-01KMRX1SEQV49
...
```

**Display with label set:**

```
FEAT-01KMR-X1SEQV49 (G · policy-and-documentation-updates)
TASK-01KMR-XK75ZA20 (G-1 · update-agents-md)
```

**Display without label:**

```
FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)
TASK-01KMR-XK75ZA20 (update-agents-md)
```

#### Why a label rather than auto-assigned sequence numbers

Auto-assigned plan-scoped sequence numbers (P6-1, P6-2, …) are tempting but
create problems:

- Deletions leave gaps, making sequences unreliable for navigation.
- The number reflects creation order, not semantic grouping — "Feature 4" carries
  no meaning, while "Feature G" or "Phase 2 / Policy" does.
- Concurrent feature creation across plan phases can produce misleading orderings.

A label is human-assigned and human-meaningful. "G" is the 7th letter because
the plan document called it Feature G in a specific phase context — not because
it was the 7th entity created. Letting the human (or orchestrator) assert the
label explicitly preserves that intent.

Labels are short (recommended: 1–4 characters or a brief phrase), unique within
a plan by convention (not enforced), and can encode plan-phase structure
naturally: `D`, `E`, `F`, `G` for Phase 2 features; `G-1`, `G-2`, `G-3` for
tasks within Feature G.

#### Setting labels

Via the `entity` tool:

```
entity(action: create, type: feature, label: "G", ...)
entity(action: update, id: "FEAT-01KMRX1SEQV49", label: "G")
```

Via `entity list` with label filter:

```
entity(action: list, type: feature, parent: "P6-...", label: "G")
```

The label is stored in the entity YAML under the existing
`fieldOrderForEntityType` convention — placed after `slug`, before `status`.

---

## 3. Display Convention Summary

| Context | Format |
|---------|--------|
| Label set | `FEAT-01KMR-X1SEQV49 (G · policy-and-documentation-updates)` |
| Label not set | `FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)` |
| Space-constrained (tables) | `FEAT-01KMR-X1SEQV49 (G)` or `FEAT-01KMR-X1SEQV49 (policy-…)` |
| Input (tool call arguments) | Accept both `FEAT-01KMR-X1SEQV49` and `FEAT-01KMRX1SEQV49` |
| Storage (YAML, filenames) | Always unsplit `FEAT-01KMRX1SEQV49` — no change |

---

## 4. Scope

### In scope

| Component | Change |
|-----------|--------|
| `internal/model/` | Add `Label` field to Feature and Task types |
| `internal/storage/` | Persist and read `label`; add to canonical field order after `slug` |
| `internal/mcp/` | Surface `display_id` as primary in all responses; append slug/label to entity references in `status`, `next`, `handoff`, `finish`, `entity list/get` |
| `internal/validate/` | Accept hyphenated display_id form as equivalent input |
| `AGENTS.md` | Note that `display_id` is the preferred reference format |
| Test files | Update snapshot tests that assert on entity output shape |

### Out of scope

- Enforcing label uniqueness within a plan (convention, not constraint)
- Label-based routing or lifecycle behaviour — label is display only
- Renaming existing YAML files or storage keys — storage IDs are unchanged
- Any change to plan IDs (already human-friendly: `P6-workflow-quality-and-review`)
- Retroactively assigning labels to existing entities (can be done manually via
  `entity update` if wanted, but not required)

---

## 5. Migration and Compatibility

- **Existing entities:** No migration needed. `label` defaults to empty/absent;
  all display logic falls back gracefully to slug-only format.
- **Existing tool calls:** The unsplit ID form continues to work everywhere as
  input. No caller breakage.
- **YAML files on disk:** Existing files without a `label` field are valid; the
  field is omitted rather than defaulted to an empty string (consistent with how
  other optional fields are handled).
- **Tests:** Snapshot tests that assert on the full entity response shape will
  need updating to accommodate the new `label` field and `display_id` promotion.
  Functional tests are unaffected.

---

## 6. Open Questions

| # | Question | Suggested answer |
|---|----------|-----------------|
| Q1 | Should label be searchable / filterable in `entity list`? | Yes — a label filter makes it easy to find "all tasks for Feature G" without knowing the FEAT ID. Low implementation cost; high navigation value. |
| Q2 | Should label appear in Git commit messages and worktree branch names? | Branch names already include the slug. Label could optionally prefix the branch: `feature/G-FEAT-01KMRX1SEQV49-policy-…`. Probably too verbose — slug alone is sufficient for branches. |
| Q3 | Should `status` dashboard include a label column? | Yes, when any feature in the plan has a label set. Omit the column entirely when no labels are present, to avoid clutter for plans that don't use them. |
| Q4 | Max label length? | Recommend 16 characters enforced; enough for "Phase 2 / G" but short enough to stay readable in table columns. |