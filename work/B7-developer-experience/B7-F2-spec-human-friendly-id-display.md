# Human-Friendly ID Display Specification

| Document | Human-Friendly ID Display (Changes 1–3) |
|----------|-----------------------------------------|
| Status   | Draft                                   |
| Created  | 2026-03-28T11:42:33Z                    |
| Updated  | 2026-03-28T11:42:33Z                    |
| Feature  | FEAT-01KMT-40KKZZR5 (human-friendly-id-display) |
| Plan     | P7-developer-experience                 |
| Related  | `work/design/human-friendly-ids.md` §2 Changes 1–3 |

---

## 1. Purpose

This specification defines three coordinated changes that make entity IDs and
labels human-navigable in tool outputs:

- **Change 1** — Surface the existing `display_id` split format as primary in
  all tool responses (no data model change).
- **Change 2** — Always show the entity slug alongside the display ID in tool
  outputs (no data model change).
- **Change 3** — Add an optional `label` field to Feature and Task entities so
  humans and orchestrators can attach a short, memorable navigation prefix.

These changes are purely additive and display-oriented. No existing IDs,
storage keys, or YAML filenames change. Both the split and unsplit ID forms
continue to be accepted as input everywhere.

---

## 2. Goals

1. Tool outputs consistently surface `display_id` (the hyphenated split form)
   as the primary identifier for features and tasks.
2. The entity slug appears alongside the ID in all major tool responses, so a
   reader never needs a separate lookup to know what an entity is.
3. Features and tasks accept an optional `label` field (max 24 chars) for
   short human-assigned navigation labels like `G`, `G-1`, `D review-states`.
4. `entity list` supports filtering by label.
5. The `status` dashboard shows a label column when any entity in the scope has
   a label set, and omits it otherwise.
6. Both the split (`FEAT-01KMR-X1SEQV49`) and unsplit (`FEAT-01KMRX1SEQV49`)
   ID forms are accepted as input in all tool calls — no breaking change.
7. All existing tests pass; snapshot tests are updated where needed.

---

## 3. Scope

### 3.1 In scope

- `internal/model/` — Add `Label` field to Feature and Task struct types.
- `internal/storage/` — Persist and read `label` in Feature and Task YAML;
  position `label` after `slug`, before `status` in the canonical field order.
  Omit `label` from YAML output when empty (consistent with other optional
  fields).
- `internal/validate/` — Accept the hyphenated display_id form as equivalent
  input alongside the unsplit form in all entity lookup paths.
- `internal/mcp/` — Update all tool handlers to use `display_id` as the primary
  identifier in responses; append slug (and label when set) to entity references
  in `status`, `next`, `handoff`, `finish`, `entity list`, and `entity get`.
- `entity create` and `entity update` — Accept `label` parameter; validate max
  24 chars.
- `entity list` — Accept `label` filter parameter.
- `status` dashboard — Conditionally show label column.
- Tests — Update snapshot tests for entity response shape; add label CRUD tests.

### 3.2 Deferred / out of scope

- Review report naming and `work/reviews/` migration (Change 4) — covered by
  FEAT-01KMT-40P0AGS7.
- Enforcing label uniqueness within a plan — label is a display aid, not a
  constraint.
- Label-based routing or lifecycle behaviour — label is display only.
- Label prefix on Git branch names — slug alone is sufficient for branches.
- Retroactively assigning labels to existing entities — callers may do so via
  `entity update` but the system does not require it.
- Renaming any existing YAML state files or storage keys.
- Changes to plan IDs (`P7-developer-experience` is already human-friendly).

---

## 4. Acceptance Criteria

### 4.1 Model — Label field

**AC-01.** The Feature struct in `internal/model/` includes a `Label` field
(`string`) that is optional (zero value is empty string, treated as absent).

**AC-02.** The Task struct in `internal/model/` includes a `Label` field
(`string`) that is optional (zero value is empty string, treated as absent).

**AC-03.** A label value must not exceed 24 characters. Validation returns an
error if a label longer than 24 characters is provided.

**AC-04.** A label value of empty string is equivalent to absent — the field is
omitted from YAML output when empty, not serialised as `label: ""`.

### 4.2 Storage — Canonical field order

**AC-05.** When a Feature entity is serialised to YAML, the `label` field
appears after `slug` and before `status` in the field order. Features without a
label omit the field entirely.

**AC-06.** When a Task entity is serialised to YAML, the `label` field appears
after `slug` and before `status` in the field order. Tasks without a label omit
the field entirely.

**AC-07.** Round-trip serialisation is correct: a Feature YAML written with a
`label`, read back, and re-written produces identical output.

**AC-08.** Round-trip serialisation is correct for entities without a `label`:
no spurious `label:` key appears in the output.

### 4.3 Storage — Input acceptance

**AC-09.** `entity get` resolves an entity successfully when passed the split
form `FEAT-01KMR-X1SEQV49` (with embedded hyphen) or the unsplit form
`FEAT-01KMRX1SEQV49`. Both forms identify the same entity.

**AC-10.** `entity update`, `entity transition`, `handoff`, `finish`, and all
other tool actions that accept an entity ID resolve successfully when passed
either the split or unsplit form.

### 4.4 Display — `display_id` as primary

**AC-11.** All `entity` tool responses include both `display_id` (split form)
and `id` (unsplit form). `display_id` is the first ID field in the response.

**AC-12.** The `status` dashboard feature and task rows use `display_id` in
the ID column.

**AC-13.** The `next` queue listing uses `display_id` in the task row.

**AC-14.** `handoff` task header uses `display_id`.

**AC-15.** `finish` completion summary uses `display_id`.

**AC-16.** All `side_effects` entries that reference an entity use `display_id`.

**AC-17.** `entity list` table rows use `display_id` in the ID column.

### 4.5 Display — Slug alongside ID

**AC-18.** Wherever a feature ID appears in a tool response, the entity's slug
is shown in parentheses immediately after the display ID:
`FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)`.

**AC-19.** Wherever a task ID appears in a tool response, the task's slug is
shown in parentheses immediately after the display ID:
`TASK-01KMR-XK75ZA20 (update-agents-md)`.

**AC-20.** When a label is set on the entity, the display format in all
contexts is `FEAT-01KMR-X1SEQV49 (G policy-docs)` — the label replaces (or
abbreviates) the full slug, with label and slug-abbreviation separated by a
space. In space-constrained display (tables), the label is shown; in verbose
display (`entity get`), both label and full slug are shown.

**AC-21.** When no label is set, the full slug is shown in parentheses. No
empty parentheses appear.

### 4.6 Label — Create and update

**AC-22.** `entity create` with `type: feature` or `type: task` accepts an
optional `label` parameter and persists it.

**AC-23.** `entity update` with `label` set to a non-empty string updates the
label on the entity.

**AC-24.** `entity update` with `label` set to empty string (`""`) clears the
label (field is removed from YAML on next write).

**AC-25.** `entity get` returns the `label` field in the response when it is
set, and omits it when absent.

### 4.7 Label — List filter

**AC-26.** `entity list` with a `label` filter parameter returns only entities
whose `label` field exactly matches the provided value.

**AC-27.** `entity list` without a `label` filter returns all entities matching
the other filters, regardless of whether they have a label.

### 4.8 Status dashboard — Conditional label column

**AC-28.** The `status` dashboard includes a Label column in the feature table
when at least one feature in the scoped plan has a non-empty `label` set.

**AC-29.** The `status` dashboard omits the Label column entirely when no
features in the scoped plan have a label set, so the table is not cluttered for
plans that do not use labels.

**AC-30.** The same conditional logic applies to the task table when viewing a
feature's tasks: the Label column appears only when at least one task has a
label.

### 4.9 Backward compatibility

**AC-31.** All existing entity YAML files without a `label` field load without
error; the field defaults to empty/absent.

**AC-32.** All existing tool call sites that pass an unsplit entity ID continue
to work unchanged — no caller breakage.

**AC-33.** All existing snapshot tests that assert on entity response shape pass
after being updated to reflect the new `display_id` promotion and `label` field
presence. No functional test logic changes.

---

## 5. Display Convention Reference

| Context | Format |
|---------|--------|
| Label set | `FEAT-01KMR-X1SEQV49 (G policy-docs)` |
| Label not set | `FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)` |
| Verbose (`entity get`) with label | `FEAT-01KMR-X1SEQV49 · label: G · policy-and-documentation-updates` |
| Input — tool call argument | Accept both split and unsplit form |
| Storage — YAML keys | Always unsplit `FEAT-01KMRX1SEQV49` — no change |
| Storage — YAML filenames | Already correct — no change |

---

## 6. Test Plan

| Test | What it covers |
|------|----------------|
| `TestFeatureLabelRoundTrip` | Write feature with label, read back, compare YAML |
| `TestFeatureNoLabelOmitted` | Feature without label produces no `label:` key |
| `TestTaskLabelRoundTrip` | Write task with label, read back, compare YAML |
| `TestLabelMaxLength` | Label > 24 chars → validation error |
| `TestEntityGetSplitID` | `entity get` with split ID resolves correctly |
| `TestEntityGetUnsplitID` | `entity get` with unsplit ID resolves correctly |
| `TestEntityListLabelFilter` | `entity list` with `label: "G"` returns only matching entities |
| `TestDisplayIDPrimary` | All entity responses include `display_id` as first ID field |
| `TestSlugAlongsideID` | `entity get` response includes slug in parentheses after display_id |
| `TestLabelInDisplay` | When label set, label appears in display; full slug shown in verbose mode |
| `TestStatusDashboardLabelColumn` | Column present when any feature has label; absent otherwise |
| `TestEntityUpdateLabel` | Set, change, and clear label via `entity update` |