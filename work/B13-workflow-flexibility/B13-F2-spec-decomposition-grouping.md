# Specification: Decomposition Grouping

**Status:** Draft
**Feature:** FEAT-01KN07T660VVM (decompose-flexibility)
**Plan:** P13-workflow-flexibility
**Design:** `work/design/workflow-completeness.md` — Feature 6
**Date:** 2026-04-02

---

## Problem Statement

The `generateProposal` function in `internal/service/decompose.go` hardcodes a
one-AC-per-task decomposition strategy. When a specification has many tightly-coupled
acceptance criteria under the same section, this produces an excessive number of tasks.
For example, a spec with 12 acceptance criteria under 3 logical sections produces 13
tasks (12 + 1 test task) when 4–5 grouped tasks would be more appropriate.

The `acceptanceCriterion` type already tracks `parentL2` (the enclosing level-2 section
title), but `generateProposal` ignores it — every AC becomes its own task regardless of
section affinity.

Additionally, acceptance criteria written as markdown table rows are not recognised by
`parseSpecStructure`. Specs that use tables to enumerate criteria (e.g. error code
tables, field validation matrices) produce zero ACs and fall through to an error.

---

## Requirements

### Functional Requirements

**FR-01. Section-based AC grouping.** When `generateProposal` builds the task list
from parsed acceptance criteria, it must group ACs that share the same `parentL2`
value before creating tasks. The grouping strategy is determined by the number of
ACs in each group:

- **1 AC:** Produce one task for that AC (current behaviour, unchanged).
- **2–4 ACs:** Produce a single task that covers all ACs in the group.
- **5+ ACs:** Produce one task per AC (current behaviour, unchanged).

**FR-02. Grouped task naming.** A grouped task (2–4 ACs) must derive its slug
from the `parentL2` section title via `slugify`, prefixed with the feature slug.
Its `Summary` must reference the section name and the count of covered criteria.
Its `Rationale` must list each covered AC text.

**FR-03. `covers` field on `ProposedTask`.** `ProposedTask` gains a `Covers`
field (`[]string`, JSON key `"covers"`, omitempty). For every proposed task — grouped
or ungrouped — `Covers` contains the AC text strings that the task is responsible
for. For ungrouped tasks (1 AC or 5+ ACs), `Covers` has exactly one element. For
grouped tasks (2–4 ACs), `Covers` has 2–4 elements. The test companion task has
an empty `Covers`.

**FR-04. Guidance rule change.** When section-based grouping produces at least one
grouped task, the `"one-ac-per-task"` entry in `GuidanceApplied` is replaced by
`"group-by-section"`. When no grouping occurs (all sections have 1 or 5+ ACs), the
existing `"one-ac-per-task"` guidance is reported as before.

**FR-05. Empty `parentL2` handling.** ACs with an empty `parentL2` (appearing
before any level-2 header) are treated as their own group with key `""`. The
same 1 / 2–4 / 5+ thresholds apply.

**FR-06. Gap check compatibility.** The `checkGaps` review function must recognise
grouped tasks. When a task has a non-empty `Covers` field, gap checking uses exact
string match against `Covers` entries instead of keyword-overlap heuristics against
`Summary` and `Rationale`. An AC is covered if its text appears in any task's
`Covers` slice. For tasks with an empty `Covers` (e.g. legacy proposals or test
companion tasks), the existing keyword-overlap heuristic remains the fallback.

**FR-07. Table row extraction.** `parseSpecStructure` must detect markdown tables
within acceptance-criteria sections and extract each data row as an
`acceptanceCriterion`. A markdown table is identified by:
- A header row containing `|` characters.
- A separator row matching the pattern `| --- | --- |` (one or more columns,
  hyphens with optional colons for alignment).
- One or more data rows containing `|` characters.

Each data row becomes one AC whose `text` is the concatenation of the cell values
joined by ` — ` (space-em-dash-space), trimmed of leading/trailing whitespace and
pipe characters. The `section` and `parentL2` fields are inherited from the
enclosing section context, consistent with existing formats.

**FR-08. Table header exclusion.** The table header row and separator row are
never extracted as acceptance criteria. Only data rows below the separator are
extracted.

---

## Constraints

**C-01.** The grouping logic is a heuristic applied during `decompose propose`.
The `review` step can still flag grouped tasks for splitting, and the human can
reject the proposal. No enforcement is added to `review` or `apply`.

**C-02.** The thresholds (2–4 for grouping, 5+ for one-per-task) are constants,
not configuration. They may be promoted to configuration in a future feature but
are not configurable in this specification.

**C-03.** Given/When/Then format parsing is explicitly deferred. This spec does
not add support for parsing Given/When/Then blocks as acceptance criteria.

**C-04.** The `covers` field is additive. Existing proposals without `covers`
(including those serialised in prior `decompose propose` outputs) remain valid.
All consumers treat a nil or empty `Covers` as "not specified" and fall back to
existing behaviour.

**C-05.** The table extraction (FR-07, FR-08) is lower priority than the grouping
logic (FR-01 through FR-06). If implementation is staged, grouping ships first.

---

## Acceptance Criteria

### Section-Based Grouping

**AC-01.** Given a spec with a level-2 section containing exactly 3 acceptance
criteria, when `generateProposal` is called, then it produces one task for that
section with `Covers` containing all 3 AC texts.

**AC-02.** Given a spec with a level-2 section containing exactly 2 acceptance
criteria, when `generateProposal` is called, then it produces one task for that
section with `Covers` containing both AC texts.

**AC-03.** Given a spec with a level-2 section containing exactly 4 acceptance
criteria, when `generateProposal` is called, then it produces one task for that
section with `Covers` containing all 4 AC texts.

**AC-04.** Given a spec with a level-2 section containing exactly 5 acceptance
criteria, when `generateProposal` is called, then it produces 5 individual tasks,
each with a single-element `Covers`.

**AC-05.** Given a spec with a level-2 section containing exactly 1 acceptance
criterion, when `generateProposal` is called, then it produces 1 task with a
single-element `Covers` (current behaviour preserved).

**AC-06.** Given a spec with two level-2 sections — one containing 3 ACs and
another containing 6 ACs — when `generateProposal` is called, then it produces
1 grouped task for the first section and 6 individual tasks for the second section
(7 tasks total, plus the test companion task).

### Covers Field

**AC-07.** Given any task produced by `generateProposal`, when the task covers
one or more acceptance criteria, then `task.Covers` is non-nil and contains the
exact AC text strings from the parsed spec.

**AC-08.** Given the test companion task produced by `generateProposal`, then
`task.Covers` is nil or empty.

**AC-09.** Given a `ProposedTask` serialised to JSON, when `Covers` is empty,
then the `"covers"` key is omitted from the output (omitempty behaviour).

### Guidance Rule

**AC-10.** Given a spec where section-based grouping produces at least one
grouped task (2–4 ACs in a section), when `generateProposal` returns, then
`GuidanceApplied` contains `"group-by-section"` and does not contain
`"one-ac-per-task"`.

**AC-11.** Given a spec where no section-based grouping occurs (all sections
have 1 or 5+ ACs), when `generateProposal` returns, then `GuidanceApplied`
contains `"one-ac-per-task"` and does not contain `"group-by-section"`.

### Grouped Task Shape

**AC-12.** Given a grouped task produced from a section titled "Error Handling"
under feature slug `my-feature`, then the task slug is `my-feature-error-handling`
(derived from `slugify` of the section title).

**AC-13.** Given a grouped task covering 3 ACs, then the task `Summary` contains
the section name and the string "3 criteria" (exact phrasing:
`"Implement <section-name> (3 criteria)"`).

**AC-14.** Given a grouped task covering 3 ACs, then the task `Rationale`
includes all 3 AC texts so they are visible in proposal output.

### Gap Check Compatibility

**AC-15.** Given a proposal where all tasks have populated `Covers` fields,
when `checkGaps` runs during `review`, then each AC is matched by exact string
lookup in `Covers` slices, not by keyword overlap.

**AC-16.** Given a proposal where a task's `Covers` is nil (legacy format),
when `checkGaps` runs, then the existing keyword-overlap heuristic is used for
that task.

**AC-17.** Given a proposal where one AC text is missing from all `Covers`
slices, when `checkGaps` runs, then a gap finding with severity `"error"` is
produced for that AC.

### Table Row Extraction

**AC-18.** Given a spec with an acceptance-criteria section containing a
markdown table with a header row, a separator row, and 3 data rows, when
`parseSpecStructure` is called, then 3 `acceptanceCriterion` entries are
produced — one per data row.

**AC-19.** Given a markdown table with columns `| ID | Condition | Expected |`,
when a data row contains `| T-01 | Input is empty | Return error |`, then the
extracted AC text is `"T-01 — Input is empty — Return error"`.

**AC-20.** Given a markdown table, the header row and separator row are not
extracted as acceptance criteria.

**AC-21.** Given a spec with both checkbox ACs and table ACs in the same
acceptance-criteria section, when `parseSpecStructure` is called, then both
formats are extracted and all ACs have the correct `section` and `parentL2`.

### Backward Compatibility

**AC-22.** Given a spec with no sections containing 2–4 ACs, when
`generateProposal` is called, then the output is identical to the current
behaviour (one task per AC, `"one-ac-per-task"` in guidance, plus test
companion task). The only addition is the `Covers` field on each task.

**AC-23.** Given an existing serialised proposal without `covers` fields,
when it is deserialised into `Proposal`, then `Covers` on each `ProposedTask`
is nil and all downstream processing (review, apply) works without error.

### Testing

**AC-24.** All new behaviour has unit tests in `internal/service/decompose_test.go`.

**AC-25.** All existing tests in `internal/service/decompose_test.go` continue
to pass without modification (except adding expected `Covers` values to
assertions where the field is now populated).

**AC-26.** All tests pass under `go test -race ./...`.

---

## Verification Plan

1. **Unit tests for grouping thresholds.** Test `generateProposal` with specs
   constructed to hit each threshold boundary: 1 AC, 2 ACs, 3 ACs, 4 ACs,
   5 ACs, and 6 ACs in a single section. Verify task count, `Covers` contents,
   and guidance applied. (AC-01 through AC-06, AC-10, AC-11)

2. **Unit tests for mixed sections.** Test with a spec containing multiple
   level-2 sections with different AC counts (e.g. one section with 3, another
   with 7). Verify the correct mix of grouped and ungrouped tasks. (AC-06)

3. **Unit tests for `Covers` field.** Verify `Covers` is populated on all
   tasks including ungrouped ones, and is empty/nil on the test companion
   task. Verify JSON serialisation omits the key when empty. (AC-07 through
   AC-09)

4. **Unit tests for grouped task shape.** Verify slug derivation from section
   title, summary format, and rationale content for grouped tasks. (AC-12
   through AC-14)

5. **Unit tests for gap check with `Covers`.** Test `checkGaps` with tasks
   that have populated `Covers` — verify exact match. Test with nil `Covers`
   — verify keyword fallback. Test with a missing AC — verify gap finding
   is produced. (AC-15 through AC-17)

6. **Unit tests for table parsing.** Test `parseSpecStructure` with a markdown
   table in an AC section. Verify data rows are extracted, header/separator
   rows are excluded, cell values are joined correctly, and `section`/`parentL2`
   are inherited. (AC-18 through AC-21)

7. **Backward compatibility.** Run the full existing test suite and verify no
   regressions. Deserialise a JSON proposal without `covers` and verify nil
   handling. (AC-22, AC-23)

8. **Race detector.** Run `go test -race ./internal/service/...` to confirm
   no data races in the new grouping logic. (AC-26)