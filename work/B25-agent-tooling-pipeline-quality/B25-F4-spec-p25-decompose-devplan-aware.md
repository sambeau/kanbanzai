# Specification: Dev-plan-aware task grouping in `decompose propose`

**Feature:** FEAT-01KPQ08YBJ5AK
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-decompose-devplan-aware.md
**Status:** Draft

---

## Overview

`decompose propose` currently applies a hardcoded acceptance-criteria heuristic to generate
task breakdowns. When a feature has an approved dev-plan document, that document already
encodes the authoritative task grouping, names, dependency graph, and effort estimates. This
specification defines the requirements for `decompose propose` to read the dev plan's Task
Breakdown section as the primary task source and fall back to the existing AC heuristic only
when no approved dev plan is available or parseable.

---

## Scope

### In scope

- Dev plan discovery: direct feature reference first, owner-query fallback second.
- Parsing the `## Task Breakdown` section of an approved dev plan document.
- Mapping parsed tasks to `ProposedTask` structs including `Name`, `Slug`, `Summary`,
  `Rationale`, `Covers`, `Estimate`, and `DependsOn`.
- Dependency resolution: converting `Task N` references in `Depends on` fields to task slugs.
- Fallback to the existing AC-based heuristic when no approved dev plan is found or the
  Task Breakdown section is absent or empty.
- Updating `GuidanceApplied` to reflect when the dev plan path was taken.
- Slice enrichment from spec analysis runs on all paths.

### Out of scope

- Modifying the dev plan document format or template.
- Parsing free-form prose task descriptions beyond the defined bolded-field list format.
- Changing `decompose review` or `decompose apply`.
- A user-facing parameter to `decompose propose` for controlling the grouping source.
- Using draft (unapproved) dev plan documents.
- Inferring a dev plan from git history or file naming conventions.

---

## Functional Requirements

**FR-001:** When `DecomposeFeature` is called for a feature that has an approved dev-plan
document, the service MUST attempt to source task proposals from that document's
`## Task Breakdown` section before applying the AC-based heuristic.

**Acceptance criteria:**
- A feature with an approved dev plan produces a proposal sourced from the dev plan when
  the Task Breakdown section is present and non-empty.
- A feature without any dev plan produces a proposal via the existing AC heuristic
  (behaviour unchanged).

---

**FR-002:** Dev plan discovery MUST proceed in two steps:
1. Read the feature's direct `dev_plan` reference field. If set, retrieve the document and
   verify its status is `approved`. If approved, use it.
2. If the direct reference is absent, not found, or not approved, query for documents with
   `owner = featureID`, `type = "dev-plan"`, `status = "approved"`. If multiple results,
   select the one with the latest `Updated` timestamp.

**Acceptance criteria:**
- A feature with a valid direct `dev_plan` reference uses that document without a fallback
  owner query.
- A feature whose direct reference points to a draft document falls through to the owner
  query.
- A feature with no direct reference but an owner-registered approved dev plan uses the
  owner-query result.
- A feature with no approved dev plan of any kind proceeds to the AC heuristic.

---

**FR-003:** Only approved dev-plan documents (status `"approved"`) MUST be used as task
sources. Draft dev plans MUST be skipped silently and trigger the AC heuristic fallback.

**Acceptance criteria:**
- A draft dev plan does not affect the proposal; the AC heuristic is used instead.
- No error is returned for a draft dev plan; the fallback is silent.

---

**FR-004:** Task Breakdown parsing MUST locate the `## Task Breakdown` heading using the
exact string match `## Task Breakdown` (case-insensitive). If the heading is absent, parsing
MUST return a failure result and the AC heuristic fallback MUST be triggered with a warning
added to `Proposal.Warnings`.

**Acceptance criteria:**
- A dev plan with `## Task Breakdown` produces tasks from that section.
- A dev plan without `## Task Breakdown` (e.g. heading spelled differently) falls back to
  the AC heuristic, and `Proposal.Warnings` contains a message indicating the fallback.

---

**FR-005:** Task Breakdown parsing MUST identify individual tasks by headings matching the
pattern `### Task N: <title>` where N is a positive integer and `<title>` is the task name.

**Acceptance criteria:**
- Each `### Task N: <title>` heading produces exactly one `ProposedTask`.
- A Task Breakdown section with no `### Task N:` headings produces zero tasks, triggering
  the fallback.

---

**FR-006:** For each parsed task, the following `ProposedTask` fields MUST be populated:

| Field | Source |
|-------|--------|
| `Name` | Title string after `"Task N: "` |
| `Slug` | `featureSlug + "-" + slugify(taskTitle)` using the existing `slugify` helper |
| `Summary` | Same as `Name` (task title) |
| `Rationale` | `"Sourced from dev-plan task N"` where N is the one-based task index |
| `Covers` | Comma-separated requirement IDs from the `**Spec requirements:**` field; nil if absent |
| `Estimate` | Mapped from `**Effort:**`: `Small → 1.0`, `Medium → 3.0`, `Large → 8.0`; nil if absent |
| `DependsOn` | Resolved slugs from `**Depends on:**` field (see FR-007) |

**Acceptance criteria:**
- All six fields are populated according to the table above for a fully-specified dev plan task.
- `Covers` is nil (not an empty slice) when the `Spec requirements` field is absent.
- `Estimate` is nil when the `Effort` field is absent.

---

**FR-007:** Dependency resolution MUST parse the `**Depends on:**` field value for all
occurrences of `Task N` (where N is a positive integer) and map each to the slug of the task
at that one-based index within the same Task Breakdown section.

**Acceptance criteria:**
- `"Depends on: Task 1"` resolves to the slug of the first parsed task.
- `"Depends on: Task 1, Task 3"` resolves to the slugs of tasks 1 and 3.
- `"Depends on: None"` and `"Depends on: None (independent)"` resolve to nil (no
  dependencies).
- Textual annotations (e.g. `"(same file; chain changes)"`) are discarded; only `Task N`
  patterns are extracted.
- A `Task N` reference where N is out of range for the parsed task list is silently dropped
  and a warning is added to `Proposal.Warnings`.

---

**FR-008:** When the dev plan Task Breakdown section is found but produces zero tasks after
parsing, the service MUST fall back to the AC heuristic and MUST add a warning to
`Proposal.Warnings` indicating the fallback reason.

**Acceptance criteria:**
- A Task Breakdown section with no `### Task N:` headings triggers the AC heuristic.
- `Proposal.Warnings` contains a message referencing the dev plan document ID and stating
  that the Task Breakdown was empty.

---

**FR-009:** When the dev plan path produces a valid proposal, `Proposal.GuidanceApplied`
MUST contain `"dev-plan-tasks"`. The guidance entries `"size-soft-limit-8"`,
`"explicit-dependencies"`, and `"role-assignment"` MUST still be appended. The
`"test-tasks-explicit"` entry MUST NOT be appended on the dev plan path.

**Acceptance criteria:**
- A proposal sourced from a dev plan includes `"dev-plan-tasks"` in `GuidanceApplied`.
- A proposal sourced from a dev plan does NOT include `"test-tasks-explicit"`.
- A proposal sourced from the AC heuristic includes `"test-tasks-explicit"` as before
  (behaviour unchanged).

---

**FR-010:** The zero-criteria spec gate (which blocks decomposition when the spec has no
parseable acceptance criteria) MUST only fire on the AC heuristic fallback path. It MUST NOT
fire when the dev plan path produces a valid proposal.

**Acceptance criteria:**
- A feature with an approved dev plan and a spec containing no parseable ACs produces a
  valid proposal (no error).
- A feature with no dev plan and a spec containing no parseable ACs returns the existing
  zero-criteria diagnostic error (behaviour unchanged).

---

**FR-011:** Slice enrichment (`analyzeSlices`) MUST run on the spec document on all code
paths, including when tasks are sourced from the dev plan.

**Acceptance criteria:**
- `Proposal.SliceDetails` is populated for proposals sourced from both the dev plan path
  and the AC heuristic path.

---

**FR-012:** The `Proposal` output schema (including `ProposedTask` fields and their types)
MUST be unchanged. No new top-level fields are added to `Proposal` or `ProposedTask`.
Callers and the `decompose apply` path require no modifications.

**Acceptance criteria:**
- `decompose apply` accepts a proposal produced by the dev plan path without modification.
- `decompose review` output is structurally identical for dev plan and AC heuristic proposals.

---

## Non-Functional Requirements

**NFR-001:** Dev plan discovery incurs at most two document-store reads (one direct-ref
lookup and one owner query). Both are local filesystem reads; no network calls are involved.
Latency impact MUST be negligible compared to the existing spec read.

**NFR-002:** All changes are confined to `internal/service/decompose.go` and
`internal/service/decompose_test.go`. No changes to `internal/mcp/decompose_tool.go`,
`internal/service/entities.go`, or any model layer are required.

**NFR-003:** The new `parseDevPlanTasks` function MUST be unexported. It is an
implementation detail of `DecomposeFeature` and MUST NOT become part of the package's
public API.

---

## Acceptance Criteria

**AC-001:** Given a feature with an approved dev plan whose Task Breakdown section defines
three tasks (Task 1, Task 2 depending on Task 1, Task 3 independent), calling
`decompose propose` produces a proposal with exactly three `ProposedTask` entries whose
names, slugs, and dependency wiring match the dev plan.

**AC-002:** Given a feature with a draft dev plan and a spec with parseable ACs, calling
`decompose propose` produces a proposal via the AC heuristic (dev plan is ignored) and no
error is returned.

**AC-003:** Given a feature with an approved dev plan whose `## Task Breakdown` heading is
absent (e.g. the heading reads `## Tasks`), calling `decompose propose` produces a proposal
via the AC heuristic and `Proposal.Warnings` contains a message about the missing section.

**AC-004:** Given a feature with an approved dev plan with tasks but no parseable ACs in
the spec, calling `decompose propose` returns a valid proposal sourced from the dev plan
(zero-criteria gate does not fire).

**AC-005:** Given a feature with no dev plan and a spec with no parseable ACs, calling
`decompose propose` returns the zero-criteria diagnostic error (behaviour unchanged).

**AC-006:** A `ProposedTask` sourced from a dev plan task with `**Effort:** Medium` has
`Estimate == 3.0`.

**AC-007:** A `ProposedTask` sourced from a dev plan task with no `**Spec requirements:**`
field has `Covers == nil`.

**AC-008:** `GuidanceApplied` for a dev plan sourced proposal contains `"dev-plan-tasks"`
and does not contain `"test-tasks-explicit"`.

**AC-009:** Slice analysis output (`SliceDetails`) is populated in proposals from both the
dev plan path and the AC heuristic path.

**AC-010:** A `decompose apply` call on an unmodified proposal produced by the dev plan
path creates all tasks without error.

---

## Dependencies and Assumptions

**DEP-001:** `internal/service/documents.go` — `DocumentService.ListDocuments` and
`GetDocumentContent` are the document access interface. No changes to this interface are
required; `DecomposeService` already holds a `*DocumentService` reference.

**DEP-002:** The dev plan document format uses `## Task Breakdown` as the exact section
heading and `### Task N: <title>` as the exact task heading format. This format is verified
consistent across P13, P24, and P25 dev plans. The spec relies on this format contract.

**DEP-003:** FEAT-01KPQ08Y71A8V (fix empty task names in decompose propose) also modifies
`internal/service/decompose.go`. These two features MUST be sequenced or their branches
coordinated to avoid merge conflicts. FEAT-01KPQ08Y71A8V is lower effort and SHOULD land
first.

**DEP-004:** The `slugify` helper in `internal/service/decompose.go` is reused for slug
generation in `parseDevPlanTasks`. No changes to `slugify` are required.

**ASSUME-001:** The feature model field `feat.State["dev_plan"]` stores the document record
ID of the feature's dev plan when one has been linked. This field is set by the document
registration workflow and is the canonical direct reference.

**ASSUME-002:** `DocumentService.ListDocuments` supports filtering by `owner`, `type`, and
`status` fields as used in the owner-query fallback.