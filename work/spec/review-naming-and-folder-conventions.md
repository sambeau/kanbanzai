# Review Naming and Folder Conventions Specification

| Document | Review Naming and Folder Conventions |
|----------|--------------------------------------|
| Status   | Draft                                |
| Created  | 2026-03-28T11:42:33Z                 |
| Updated  | 2026-03-28T11:42:33Z                 |
| Feature  | FEAT-01KMT-40P0AGS7 (review-naming-and-folder-conventions) |
| Plan     | P7-developer-experience              |
| Related  | `work/design/human-friendly-ids.md` §2 Change 4, §5 Migration |

---

## 1. Purpose

This specification defines the canonical naming convention for review report
files and establishes `work/reviews/` as the dedicated folder for all review
artifacts produced by the feature review lifecycle gate.

It also covers the one-time migration of five existing review reports from
`work/reports/` to `work/reviews/` with corrected filenames, and the
corresponding updates to the code review SKILL, bootstrap workflow document,
`AGENTS.md`, and document state records.

This is Change 4 from the Human-Friendly ID Display design proposal. It is a
documentation and tooling-convention change only — no model schema changes, no
Go code changes, and no MCP tool changes are required.

---

## 2. Goals

1. Review reports follow a consistent `review-{id}-{slug}.md` naming convention
   that mirrors the ID+slug pattern used in YAML state filenames.
2. `work/reviews/` is the sole canonical destination for review artifacts
   produced by the formal review workflow.
3. `work/reports/` is reserved for general-purpose reports: retrospectives,
   friction analyses, audit findings, research outputs, and progress reports.
4. The code review SKILL instructs agents to write review reports to
   `work/reviews/` with the correct filename format.
5. `work/bootstrap/bootstrap-workflow.md` and `AGENTS.md` document the folder
   distinction explicitly.
6. All five existing misplaced/misnamed review reports are migrated to the
   correct location and name.
7. Document state records for migrated reports reflect the new paths and current
   content hashes.

---

## 3. Scope

### 3.1 In scope

- Renaming and moving five existing review reports (see §6 Migration Table).
- Updating the five corresponding document state YAML records to reflect the
  new paths and recomputed content hashes.
- Updating `.skills/code-review.md` Step 6 to specify the `work/reviews/`
  destination and `review-{id}-{slug}.md` filename convention.
- Updating `work/bootstrap/bootstrap-workflow.md` document placement table to
  add a `work/reviews/` row and clarify its distinction from `work/reports/`.
- Updating `AGENTS.md` repository structure listing to include `work/reviews/`.

### 3.2 Explicitly excluded

- Go code changes of any kind.
- MCP tool changes.
- Model or storage schema changes.
- Renaming non-review files in `work/reports/`.
- Renaming existing YAML state files in `.kbz/state/`.
- Retroactively applying labels or display_id changes (those are Features
  FEAT-01KMT-40GZSMHB and FEAT-01KMT-40KKZZR5).
- Adding a new document type for review reports — they remain type `report`.

---

## 4. Naming Convention

### 4.1 Format

Review report filenames must follow this format:

```
review-{unsplit-id}-{slug}.md
```

- `{unsplit-id}` is the full entity ID without hyphens in the TSID portion
  (e.g. `FEAT-01KMRX1SEQV49`, not `FEAT-01KMR-X1SEQV49`). This matches
  the convention used in YAML state filenames and is a valid path component.
- `{slug}` is the entity's slug field verbatim
  (e.g. `policy-and-documentation-updates`).
- The file extension is `.md`.

**Examples:**

```
review-FEAT-01KMRX1SEQV49-policy-and-documentation-updates.md
review-FEAT-01KMR8QW7A3A8-review-batch-operations.md
review-BUG-01KMRX1F47Z94-some-bug-slug.md
```

### 4.2 Destination folder

All review reports are written to `work/reviews/`.

---

## 5. Folder Distinction

| Folder | Contents |
|--------|----------|
| `work/reviews/` | Feature and bug review reports produced by the `reviewing` lifecycle gate — output of the formal review workflow. Every file corresponds to a feature or bug that has passed through the `reviewing` state. |
| `work/reports/` | General-purpose reports: retrospectives, friction analyses, audit findings, research outputs, and progress reports. Not lifecycle-coupled. |

The key distinction: `work/reviews/` is workflow-coupled (every file has a
corresponding entity); `work/reports/` is general-purpose.

---

## 6. Acceptance Criteria

### 6.1 Review report naming convention and folder

**AC-01.** The file `.skills/code-review.md` is updated so that the step
describing where to write the review document specifies:
- Destination folder: `work/reviews/`
- Filename format: `review-{unsplit-id}-{slug}.md`
- A concrete example filename is included.

**AC-02.** `work/bootstrap/bootstrap-workflow.md` document placement table
includes a row for `work/reviews/` with description "Review reports produced
by the formal `reviewing` lifecycle gate; one file per reviewed feature or bug."

**AC-03.** `work/bootstrap/bootstrap-workflow.md` clarifies that `work/reports/`
is for general-purpose reports and does not include review lifecycle artifacts.

**AC-04.** `AGENTS.md` repository structure section lists `work/reviews/` with
a description consistent with §5 of this specification.

### 6.2 Migration of existing review reports

**AC-05.** The following five files are renamed and moved. The old paths no
longer exist; the new paths exist and contain the same content as the
originals:

| Old path | New path |
|----------|----------|
| `work/reports/review-FEAT-01KMKRQSD1TKK.md` | `work/reviews/review-FEAT-01KMKRQSD1TKK-skills-content.md` |
| `work/reports/review-FEAT-01KMRX1F47Z94.md` | `work/reviews/review-FEAT-01KMRX1F47Z94-review-lifecycle-states.md` |
| `work/reports/review-FEAT-01KMRX1HG8BAX.md` | `work/reviews/review-FEAT-01KMRX1HG8BAX-reviewer-context-profile-and-skill.md` |
| `work/reports/review-FEAT-01KMRX1QPN3CB.md` | `work/reviews/review-FEAT-01KMRX1QPN3CB-review-orchestration-pattern.md` |
| `work/reports/review-FEAT-01KMRX1SEQV49.md` | `work/reviews/review-FEAT-01KMRX1SEQV49-policy-and-documentation-updates.md` |

**AC-06.** The existing review report already in `work/reviews/` is renamed for
consistency:

| Old path | New path |
|----------|----------|
| `work/reviews/track-c-batch-operations-review.md` | `work/reviews/review-FEAT-01KMR8QW7A3A8-review-batch-operations.md` |

**AC-07.** For each file migrated in AC-05 and AC-06, the corresponding
document state record in `.kbz/state/documents/` has its `path` field updated
to the new path and its `content_hash` updated to match the SHA-256 hash of
the file's current content.

### 6.3 No regressions

**AC-08.** No files in `work/reports/` are renamed, moved, or modified except
those listed in AC-05 (the five old review reports that are being moved).

**AC-09.** No YAML entity state files in `.kbz/state/` are modified.

**AC-10.** No Go source files are modified.

---

## 7. Verification

After implementation:

1. `ls work/reviews/` lists exactly six files — the five migrated reports plus
   the pre-existing `track-c-batch-operations-review.md` renamed per AC-06 —
   all following the `review-{id}-{slug}.md` convention.
2. `ls work/reports/` contains no files matching `review-FEAT-*.md` or
   `review-BUG-*.md`.
3. `.skills/code-review.md` grep for `work/reviews` returns at least one match
   in the "write review document" step.
4. `work/bootstrap/bootstrap-workflow.md` grep for `work/reviews` returns at
   least one match in the document placement section.
5. `AGENTS.md` grep for `work/reviews` returns at least one match in the
   repository structure listing.
6. `kbz doc validate` (or equivalent) reports no content-hash mismatches for
   any of the six migrated document records.