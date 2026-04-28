# Design Proposal: Human-Friendly ID Display

- Status: proposal
- Date: 2026-03-28
- Author: orchestrator
- Related: `work/reports/kanbanzai-2.0-workflow-retrospective.md` §3.4 (R4)
- Retro item: R4 — Human-Friendly ID Display (P1, medium severity)
- Related: `work/reviews/` folder canonicalisation (see §2, Change 4)

---

## 1. Problem

Entity IDs are hard to read, remember, and communicate. Four distinct problems
contribute:

### A note on TSID structure

TSIDs encode a millisecond-precision timestamp in their high bits. This means
entities created close together in time share a common prefix:

```
FEAT-01KM8JT7542GZ   ← created ~16:14:22Z
FEAT-01KM8JTBFEJ4Q   ← created ~16:14:26Z  (share 01KM8JT)
FEAT-01KM8JTF0MK91   ← created ~16:14:30Z  (share 01KM8JTF0)
FEAT-01KM8JTF0VP0K   ← created ~16:14:30Z  (share 01KM8JTF0, same ms window)
```

This time-clustering is intentional and useful: it keeps related work together
in sorted listings, and the shared prefix visually signals "created at the same
time." The planned split format (`FEAT-01KM8-JTF0MK91`) was designed to make
this time-prefix boundary visible.

The YAML state filenames already demonstrate the right end state — every file
uses the full ID + slug (`FEAT-01KM8JTF0MK91-bootstrap-self-hosting.yaml`). The
pattern works well precisely because of the fixed-width ID: **agents read left**
(the ID prefix identifies and sorts), **humans read right** (the slug tells you
what it is). All four problems below are about extending this already-working
pattern more consistently across the system.


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

**1.4 — Review reports are misnamed and in the wrong folder.**
Feature review reports are currently placed in `work/reports/` alongside
unrelated general reports (retrospectives, friction analyses, audit outputs) and
named without a slug: `review-FEAT-01KMKRQSD1TKK.md`. This breaks the ID+slug
filename convention that works well everywhere else, makes the folder a mixed
grab-bag, and forces a lookup to know which feature a review file covers.
`work/reviews/` already exists as a folder and is the natural home for review
artifacts now that reviews are a first-class workflow gate.

The combined effect: a status table full of opaque TSIDs that requires constant
cross-referencing with plan documents, a text conversation that mixes
human-friendly label prose with machine-generated TSID references, and a review
archive that is hard to navigate by filename alone.

---

## 2. Proposed Changes

Four changes, independent but complementary.

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
FEAT-01KMR-X1SEQV49 (G policy-docs)
TASK-01KMR-XK75ZA20 (G-1 update-agents)
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

Labels combine a short ordering prefix with a short descriptive suffix — just
enough words to be self-describing without a lookup. Max 24 characters enforced.
Examples: `G policy-docs`, `D review-states`, `F orchestration`, `G-1 update-agents`.
The ordering prefix (`G`, `G-1`, `Phase2-F`) preserves sequence; the words remove
ambiguity. Ragged right across table rows is fine — the ID column provides the
consistent left anchor.

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

### Change 4 — Review report naming convention and `work/reviews/` as canonical folder

#### 4.1 Naming convention

Review reports must follow the same ID+slug pattern as YAML state files:

```
review-FEAT-01KMRX1SEQV49-policy-and-documentation-updates.md
review-FEAT-01KMR8QW7A3A8-review-batch-operations.md
review-BUG-01KMRX1F47Z94-some-bug-slug.md
```

**Format:** `review-{unsplit-id}-{slug}.md`

The unsplit ID is used in the filename (not the split display form) so the
filename is a valid path component and matches the YAML storage convention.
The slug follows immediately after, giving the human-readable suffix.

This convention is already partially present — the existing review reports use
`review-{id}.md` without the slug. Adding the slug costs nothing and makes
every filename self-describing.

#### 4.2 Canonical folder

| Folder | Contents |
|--------|----------|
| `work/reviews/` | Feature and bug review reports produced by the `reviewing` lifecycle gate — output of the formal review workflow |
| `work/reports/` | General-purpose reports: retrospectives, friction analyses, audit findings, research outputs, progress reports |

The distinction is: `work/reviews/` is workflow-coupled (every file corresponds
to a feature or bug passing through `reviewing`); `work/reports/` is
general-purpose.

`work/reviews/` already exists and contains `track-c-batch-operations-review.md`
from P4's manually-triggered review — confirming the folder was always intended
for this purpose.

#### 4.3 Updates required

- The code review SKILL (`skills/code-review.md`) Step 6 "Write review document"
  must specify both the naming convention and the `work/reviews/` destination.
- `work/bootstrap/bootstrap-workflow.md` document placement table must add a
  `work/reviews/` row and clarify the distinction from `work/reports/`.
- `AGENTS.md` repository structure listing must include `work/reviews/`.
- The five existing review reports in `work/reports/` must be renamed (adding
  the slug suffix) and moved to `work/reviews/`. Their document records must be
  updated to reflect the new paths (see §5, Migration).

---

## 3. Display Convention Summary

| Context | Format |
|---------|--------|
| Label set | `FEAT-01KMR-X1SEQV49 (G policy-docs)` |
| Label not set | `FEAT-01KMR-X1SEQV49 (policy-and-documentation-updates)` |
| Space-constrained (tables) | `FEAT-01KMR-X1SEQV49 (G policy-docs)` — label already compact, no truncation needed |
| Input (tool call arguments) | Accept both `FEAT-01KMR-X1SEQV49` and `FEAT-01KMRX1SEQV49` |
| Storage — YAML keys | Always unsplit `FEAT-01KMRX1SEQV49` — no change |
| Storage — YAML filenames | Already correct: `FEAT-01KMRX1SEQV49-policy-and-documentation-updates.yaml` |
| Generated filenames (review reports) | `review-FEAT-01KMRX1SEQV49-policy-and-documentation-updates.md` (Change 4) |

---

## 4. Scope

### In scope

| Component | Change |
|-----------|--------|
| `internal/model/` | Add `Label` field to Feature and Task types |
| `internal/storage/` | Persist and read `label`; add to canonical field order after `slug` |
| `internal/mcp/` | Surface `display_id` as primary in all responses; append slug/label to entity references in `status`, `next`, `handoff`, `finish`, `entity list/get` |
| `internal/validate/` | Accept hyphenated display_id form as equivalent input |
| `AGENTS.md` | Note `display_id` as preferred reference format; add `work/reviews/` to repository structure |
| `work/bootstrap/bootstrap-workflow.md` | Add `work/reviews/` to document placement table |
| `.skills/code-review.md` | Update Step 6 with review report naming convention and `work/reviews/` destination |
| `work/reviews/` | Migrate 5 existing review reports from `work/reports/`, renamed with slug suffix |
| Document records | Update 5 review report document records to reflect new paths |
| Test files | Update snapshot tests that assert on entity output shape |

### Out of scope

- Enforcing label uniqueness within a plan (convention, not constraint)
- Label-based routing or lifecycle behaviour — label is display only
- Renaming existing YAML state files or storage keys — unchanged
- Any change to plan IDs (already human-friendly: `P6-workflow-quality-and-review`)
- Retroactively assigning labels to existing entities (can be done manually via
  `entity update` if wanted, but not required)
- Renaming non-review files in `work/reports/`

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
- **Review report migration:** The five existing review reports require a
  rename-and-move. Each step: (1) rename file to add slug suffix, (2) move from
  `work/reports/` to `work/reviews/`, (3) update the document record's `path`
  field and `content_hash`. The reports are small and self-contained; there are
  no inbound links to update beyond the document records themselves.

  | Old path | New path |
  |----------|----------|
  | `work/reports/review-FEAT-01KMKRQSD1TKK.md` | `work/reviews/review-FEAT-01KMKRQSD1TKK-skills-content.md` |
  | `work/reports/review-FEAT-01KMRX1F47Z94.md` | `work/reviews/review-FEAT-01KMRX1F47Z94-review-lifecycle-states.md` |
  | `work/reports/review-FEAT-01KMRX1HG8BAX.md` | `work/reviews/review-FEAT-01KMRX1HG8BAX-reviewer-context-profile-and-skill.md` |
  | `work/reports/review-FEAT-01KMRX1QPN3CB.md` | `work/reviews/review-FEAT-01KMRX1QPN3CB-review-orchestration-pattern.md` |
  | `work/reports/review-FEAT-01KMRX1SEQV49.md` | `work/reviews/review-FEAT-01KMRX1SEQV49-policy-and-documentation-updates.md` |

---

## 6. Open Questions

| # | Question | Suggested answer |
|---|----------|-----------------|
| Q1 | Should label be searchable / filterable in `entity list`? | Yes — a label filter makes it easy to find "all tasks for Feature G" without knowing the FEAT ID. Low implementation cost; high navigation value. |
| Q2 | Should label appear in Git commit messages and worktree branch names? | Branch names already include the slug. Label could optionally prefix the branch: `feature/G-FEAT-01KMRX1SEQV49-policy-…`. Probably too verbose — slug alone is sufficient for branches. |
| Q3 | Should `status` dashboard include a label column? | Yes, when any feature in the plan has a label set. Omit the column entirely when no labels are present, to avoid clutter for plans that don't use them. |
| Q4 | Label format and length? | A label should be short ordering letters or numbers followed by a short descriptive word or two — just enough to be self-describing. The ordering prefix preserves sequence; the words remove the need for a lookup. Examples: `G policy-docs`, `D review-states`, `F orchestration`, `Phase2-E reviewer`. Max 24 characters enforced. Ragged right (variable label lengths across rows) is not a problem in table display — the ID and slug columns provide consistent structure. |
| Q5 | Should the `track-c-batch-operations-review.md` already in `work/reviews/` be renamed to follow the new convention? | Yes — rename to `review-FEAT-01KMR8QW7A3A8-review-batch-operations.md` in the same migration pass for consistency. |
| Q6 | Should `work/reviews/` be added to the document type mapping with its own type? | No — review reports remain type `report`. The folder distinction is organisational, not a schema change. The bootstrap-workflow.md placement table is the right place to document the folder's purpose. |