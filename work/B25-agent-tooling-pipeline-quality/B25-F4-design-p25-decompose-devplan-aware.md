# Design: Dev-plan-aware task grouping in `decompose propose`

**Feature:** FEAT-01KPQ08YBJ5AK
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Status:** Draft
**Date:** 2026-05-20

---

## Overview

`decompose propose` produces a task breakdown by parsing the feature's approved
specification document and applying a set of hardcoded heuristics. The primary
heuristic groups acceptance criteria by their parent level-2 section: when a
section contains two to four ACs, they are merged into one task; when a section
contains one AC or five or more ACs, each AC becomes its own task. A test task
is appended unconditionally if none is detected.

This heuristic works well when the intended task granularity aligns with
acceptance-criteria granularity — typically one task per logical code unit.
It degrades whenever the intended grouping cuts across AC boundaries.

**Impact in P24:** FEAT-01KPPG4SXY6T0 (workflow hygiene docs) had 13 acceptance
criteria, each scoped to a single file-level change. Its dev plan defined four
tasks — one per target file — as the intended unit of work. `decompose propose`
produced 13 tasks (one per AC), ignoring the dev plan entirely. The proposal was
structurally wrong: 13 sequential documentation edits rather than 4 parallelisable
ones. The agent discarded the proposal and created tasks manually using
`entity(action: create)`.

The root cause is architectural: `DecomposeFeature` is aware of the feature's
specification document (read from `feat.State["spec"]`) but is unaware of the
feature's dev plan document. The dev plan already encodes the correct task
grouping, task names, dependency graph, and effort estimates — exactly the
information the heuristic is trying to derive. Reading it directly eliminates
the derivation step and the class of errors it produces.

---

## Goals and Non-Goals

### Goals

- When an approved dev-plan document is linked to the feature, `decompose propose`
  reads its `## Task Breakdown` section and uses the tasks defined there as the
  authoritative proposal, bypassing the AC-based heuristic.
- When no approved dev plan exists, behaviour is unchanged: fall back to the
  current AC-based heuristic.
- When a dev plan exists but its Task Breakdown section is absent or produces
  zero tasks, emit a warning and fall back to the AC-based heuristic.
- The `Proposal` output schema is unchanged; callers and the `decompose apply`
  path require no modifications.
- The spec approval gate and spec existence gate remain in place.

### Non-Goals

- Modifying the dev plan document format or template — the design relies on
  the existing standardised format.
- Parsing free-form prose task descriptions into sub-tasks or estimates —
  only the Task Breakdown section is consumed.
- Changing `decompose review` or `decompose apply`.
- Introducing a user-facing parameter to `decompose propose` for controlling
  the grouping source (see Alternatives Considered).
- Inferring a dev plan from git history or file naming conventions when no
  registered document record exists.
- Parsing dev plans that are in `draft` status — only approved documents are
  used, matching the precedent set by the spec approval gate.

---

## Design

### 1. Dev plan discovery

`DecomposeFeature` already holds references to both `entitySvc` and `docSvc`.
The feature model stores an optional direct link to its dev plan document in
`feat.State["dev_plan"]` (mapped to `model.Feature.DevPlan`). This is the
canonical reference — when it is set, it is always the correct document.

Discovery proceeds in two steps:

1. **Direct reference:** Read `feat.State["dev_plan"].(string)`. If non-empty,
   call `docSvc.GetDocumentContent(devPlanDocID)` and verify the returned
   document record has `Status == "approved"`. If not approved, skip and
   proceed to step 2.

2. **Owner lookup fallback:** Call
   `docSvc.ListDocuments(DocumentFilters{Owner: featureID, Type: "dev-plan", Status: "approved"})`.
   If multiple results are returned, select the one with the latest `Updated`
   timestamp. This handles features where the dev plan was registered before
   the direct-link convention was established.

If neither step yields an approved dev plan, proceed to the existing
`generateProposal` path without change.

### 2. Task Breakdown parsing

The dev plan format used across P13, P24, and P25 is consistent and
machine-parseable:

```
## Task Breakdown

### Task 1: <task title>

- **Description:** <prose>
- **Deliverable:** <prose>
- **Depends on:** None  |  Task 2  |  Task 1, Task 3  |  None (independent)
- **Effort:** Small  |  Medium  |  Large
- **Spec requirements:** FR-001, FR-002

### Task 2: <task title>

- **Description:** <prose>
- **Depends on:** Task 1
...
```

A new unexported function `parseDevPlanTasks(featureSlug string, content []byte)
([]ProposedTask, bool)` performs the parse. `bool` is `true` when parsing
succeeds and produces at least one task; `false` when it should be treated as a
parse miss (triggers fallback).

**Parse procedure:**

1. Locate the `## Task Breakdown` heading by scanning for a line matching
   `(?m)^## Task Breakdown\s*$` (case-insensitive). If absent, return `(nil, false)`.

2. Extract the body of the Task Breakdown section: everything from the heading
   until the next `##`-level heading (or end of file).

3. Within that body, find all task headings matching
   `(?m)^### Task \d+: (.+)$`. Record each match as a candidate task with:
   - its one-based index
   - its title (capture group 1, trimmed)
   - its byte offset within the body (to extract the task's own content block)

4. For each candidate task, extract the block of text from its heading until
   the next `###`-level heading (or end of the section). Within that block,
   parse the following fields using bolded-key list-item regexes of the form
   `(?m)^\s*[-*]\s+\*\*<Key>:\*\*\s+(.+)$`:
   - `Depends on` → raw dependency string
   - `Effort` → effort label (Small / Medium / Large)
   - `Spec requirements` → comma-separated requirement IDs

5. Map each candidate task to a `ProposedTask`:
   - `Slug`: `featureSlug + "-" + slugify(taskTitle)` using the existing `slugify` helper.
   - `Name`: taskTitle (the string after `"Task N: "`).
   - `Summary`: taskTitle.
   - `Rationale`: `"Sourced from dev-plan task " + strconv.Itoa(index)`.
   - `Covers`: requirement IDs split from the Spec requirements field. If absent,
     `Covers` is nil (not an error — some dev plans omit the field).
   - `Estimate`: Small → `1.0`, Medium → `3.0`, Large → `8.0`, absent → `nil`.
   - `DependsOn`: resolved in step 6 below.

6. **Dependency resolution.** Build an index from task position to slug. Parse
   the raw `Depends on` string for all occurrences of `Task (\d+)` using
   `regexp.FindAllStringSubmatch`. Map each captured integer to the slug of
   that task. The textual addenda (e.g. `"(same file; chain changes)"`) are
   discarded — they are human notes, not machine-readable dependencies. If the
   raw string matches `(?i)^none` or produces zero `Task N` matches, `DependsOn`
   is nil.

7. If the final task list is empty, return `(nil, false)`.

### 3. Integration point in `DecomposeFeature`

The new path is inserted between step 4 (spec approval gate) and the existing
step 5 (parse spec structure), and the zero-criteria gate moves inside the
fallback branch. The revised flow:

```
1. Load feature
2. Verify spec is linked
3. Load spec content; verify approved
4. Attempt dev plan discovery (steps 1–2 of §1 above)
5. If approved dev plan found:
   a. Call parseDevPlanTasks(featureSlug, devPlanContent)
   b. If parse succeeds (returns true):
      - Build Proposal from returned tasks
      - Set GuidanceApplied = ["dev-plan-tasks"]
      - Append info warning: "Tasks sourced from dev-plan <docID>"
      - Skip to step 7 (slice enrichment)
   c. If parse fails (returns false):
      - Append warning: "dev-plan <docID> found but Task Breakdown absent or empty — falling back to AC heuristic"
      - Continue to step 6
6. (Fallback) Parse spec for AC structure
   - Gate: spec must contain parseable acceptance criteria
   - Call generateProposal(spec, featureSlug, input.Context, maxTasks)
7. Enrich proposal with SliceDetails from spec analysis (always)
8. Return DecomposeResult
```

The zero-criteria gate moves to step 6 (inside the fallback path). A feature
with an approved dev plan that parses successfully does not require AC-parseable
spec content — the task grouping authority has been delegated to the dev plan.

### 4. Guidance and provenance

When the dev plan path is taken, `GuidanceApplied` contains `"dev-plan-tasks"`
in place of `"one-ac-per-task"` or `"group-by-section"`. The other guidance
entries (`"size-soft-limit-8"`, `"explicit-dependencies"`, `"role-assignment"`)
are still appended — they represent review-time checks that apply regardless of
task source. The `"test-tasks-explicit"` rule is **not** applied on the dev plan
path: if the dev plan author chose not to define an explicit test task, that is
an intentional decision for this feature type (e.g. documentation-only features).

### 5. Slice enrichment

`analyzeSlices(spec, content)` is called on the spec in all paths. Slice
analysis is independent of task grouping — it reflects the architectural
decomposition of the spec, which is valuable context for reviewers even when
tasks are sourced from the dev plan.

### 6. Failure modes

| Condition | Handling |
|-----------|----------|
| `feat.State["dev_plan"]` is non-empty but document is not found | Log as warning; attempt owner lookup fallback |
| Approved dev plan found; `## Task Breakdown` section absent | Warning in `Proposal.Warnings`; fall back to AC heuristic |
| Task Breakdown found; all tasks parse to zero-task list | Warning; fall back |
| `Depends on` references a `Task N` index out of range | That dependency is silently dropped; a warning is added |
| Dev plan content is empty or unreadable | Treat as parse failure; fall back |
| Spec has zero ACs and no dev plan exists | Existing `buildZeroCriteriaDiagnostic` error (unchanged) |
| Spec has zero ACs but dev plan produces tasks | Success — zero-AC gate does not fire on the dev plan path |

---

## Alternatives Considered

### Alternative 1: Parse the dev plan (recommended — this design)

**Approach:** `DecomposeFeature` discovers the feature's approved dev plan,
parses the standardised Task Breakdown section, and uses those tasks as the
proposal. Falls back to the AC heuristic if no dev plan exists or parsing fails.

**What it makes easy:**
- Eliminates the P24 failure mode entirely for features with dev plans.
- No new tool parameters or UX changes required.
- The dev plan format is already standardised and consistently used.
- The fallback path preserves all existing behaviour; no regression risk for
  features without dev plans.

**What it makes harder:**
- `DecomposeFeature` now has a document-lookup step that was not previously
  present, adding latency proportional to the document store read.
- The parser must be robust to format variations (e.g. `**Spec requirement:**`
  vs `**Spec requirements:**`, trailing punctuation). These are low-risk but
  require test coverage.

**Why chosen:** The dev plan is the authoritative source of task grouping
intent. Parsing it directly eliminates an entire class of mismatch errors
without introducing a new API surface or workflow step.

---

### Alternative 2: Add a `grouping_hint` parameter to `decompose propose`

**Approach:** Expose a `grouping_hint` string parameter (e.g. `"file-per-task"`,
`"ac-per-task"`, `"one-task"`) that callers can pass to override the heuristic.

**What it makes easy:**
- No document parsing required; the heuristic can be steered by the caller.
- Explicit, inspectable — the caller declares the grouping intent at call time.

**What it makes harder:**
- Agents must know to set the hint and must know which hint applies to the
  current feature. This requires the orchestrating agent to read the dev plan,
  understand its intent, and translate it to a hint — exactly the reasoning
  step we want to automate.
- Adds a new API parameter that must be documented, maintained, and handled
  in the fallback case.
- The hint vocabulary is a lossy encoding of the dev plan. A `"file-per-task"`
  hint cannot capture the actual task names, dependency graph, or effort
  estimates that the dev plan defines.
- Does not solve the case where the agent fails to set the hint (which is
  exactly the failure mode observed in P24).

**Why rejected:** Shifts the problem from the tool to the caller. Agents
already have access to the dev plan; the tool should use it directly rather
than requiring agents to translate it into a hint.

---

### Alternative 3: Leave the heuristic unchanged; document the manual fallback

**Approach:** Accept the current heuristic as-is. Add a note to the
`decompose-feature/SKILL.md` skill explaining that agents should manually
create tasks via `entity(action: create)` when the proposal is wrong.

**What it makes easy:**
- No code changes required.
- The manual fallback already works; documentation reduces agent confusion.

**What it makes harder:**
- Manual task creation requires agents to understand the internal task schema,
  dependency wiring format, and slug conventions — knowledge that is not reliably
  present in sub-agent context.
- The failure mode recurs for every feature with a dev plan, across all future
  plans. The cost compounds.
- `decompose propose` becomes a tool that is only reliable for a subset of
  features, with no signal to the caller about which subset.

**Why rejected:** Documents a workaround without addressing the root cause.
The P24 failure produced two manual workarounds in one sprint; the cost is
not negligible. Alternative 3 (P5 in the proposal) remains a useful complement
to this design — the fallback note should still be added — but it does not
replace this fix.

---

## Dependencies

| Dependency | Type | Notes |
|------------|------|-------|
| `internal/service/decompose.go` — `DecomposeFeature` | Code change | New dev plan discovery and parse step inserted before the existing AC heuristic path |
| `internal/service/decompose.go` — new `parseDevPlanTasks` | Code addition | Unexported; tested in `decompose_test.go` |
| `internal/service/documents.go` — `DocumentService.ListDocuments`, `GetDocumentContent` | Existing interface | No changes required; `DecomposeService` already holds a `*DocumentService` reference |
| `internal/service/decompose.go` — existing `slugify` helper | Code reuse | `parseDevPlanTasks` uses `slugify` for consistent slug generation |
| Dev plan document format (`work/dev-plan/` template) | Format contract | Design relies on `## Task Breakdown` + `### Task N: <title>` + bolded field list structure. Verified consistent across P13, P24, and P25 dev plans. |
| `FEAT-01KPQ08YBJ8J0` (P3 — fix empty task names) | Parallel feature | P3 and P4 both touch `decompose.go`; they should be sequenced or their branches coordinated to avoid merge conflicts. P3 is lower effort and can land first. |

---

## Decisions

**Decision:** Discover dev plan via direct feature reference first, owner-lookup second.

**Context:** The feature model stores `dev_plan` as a direct document ID reference
(set when the dev plan is registered with `owner: FEAT-...`). This reference is
the canonical link. However, some features in earlier plans may have dev plans
registered before the direct-link convention was enforced.

**Rationale:** Checking the direct reference first is O(1) and always correct
when present. The owner lookup fallback ensures coverage for historical features
without imposing extra lookups on the common case.

**Consequences:** Two document-store reads in the worst case (direct ref not
found or not approved → owner lookup). Both are local filesystem reads; latency
impact is negligible.

---

**Decision:** Only use approved dev plans, not drafts.

**Context:** The existing spec approval gate (`Status == "approved"`) guards the
decomposition path. Applying the same gate to dev plans is consistent and prevents
incomplete or in-progress plans from contaminating the proposal.

**Rationale:** A draft dev plan may still be under revision. Using it would
produce a proposal that diverges from the final approved intent. Approved status
is the feature system's signal that a document is authoritative.

**Consequences:** A feature with a draft dev plan falls back to the AC heuristic.
This is the correct behaviour — if the dev plan is not yet approved, the heuristic
is no worse than it is today.

---

**Decision:** Move the zero-criteria spec gate inside the fallback path.

**Context:** Currently the zero-criteria gate fires before any heuristic is
applied, unconditionally. If a dev plan is present and produces tasks, the spec's
AC count is irrelevant to task generation.

**Rationale:** Documentation-only features (like the P24 hygiene docs feature)
legitimately have ACs that don't parse well under the current regex-based
extractor. Requiring a parseable spec even when the dev plan is the authoritative
source would create a false gate that blocks correct proposals.

**Consequences:** Features with approved dev plans and non-parseable specs
will succeed where they currently fail. Features without dev plans are
unaffected — the gate still fires for them. This slightly increases the surface
where `DecomposeFeature` succeeds; the spec approval gate (step 3) still ensures
the spec exists and is approved before any path proceeds.

---

**Decision:** Do not apply the `test-tasks-explicit` guidance rule on the dev plan path.

**Context:** `generateProposal` unconditionally appends a test task when no
task with "test" in the summary is found. This heuristic is appropriate for
code features but not for documentation-only features, configuration changes,
or other feature types where the dev plan author has deliberately omitted a
test task.

**Rationale:** The dev plan author has already made a conscious task-grouping
decision. Appending a test task that was intentionally omitted contradicts the
dev plan's authority. If the dev plan does include a test task, it will be
parsed and included naturally.

**Consequences:** Proposals generated from dev plans may lack an explicit test
task for features where one is appropriate. This is an acceptable trade-off:
the dev plan is the right place to declare a test task, and agents writing dev
plans should be guided to include one when relevant (a note in the dev plan
template is the appropriate mechanism).

---

**Decision:** Always run slice analysis against the spec, regardless of task source.

**Context:** `analyzeSlices` operates on the parsed spec structure to identify
architectural layers and estimate complexity. It produces `SliceDetails` in
the proposal, which are informational for reviewers.

**Rationale:** Slice analysis is independent of task grouping. It reflects the
architectural composition of the feature as expressed in the spec and remains
useful for reviewers even when the task list comes from the dev plan.

**Consequences:** `parseSpecStructure` is called even on the dev plan path,
adding a spec parse step that is otherwise only needed for the fallback.
This is a minor inefficiency offset by the value of consistent slice
analysis output.

---

## Decisions

- **Decision:** The `## Task Breakdown` heading is treated as a strict contract.
  **Rationale:** Every dev plan produced in this project uses the template heading verbatim.
  Accepting aliases (e.g. `## Tasks`) would require maintaining a list of synonyms with no
  concrete benefit; the template is the enforced interface. If the heading is absent or
  spelled differently, the parser falls through to the AC-based heuristic — which is the
  correct and safe fallback. Aliases can be added in a follow-up if real-world divergence is
  observed.

- **Decision:** `Summary` is derived from the task title only, not from `**Description:**`.
  **Rationale:** Extracting the first sentence of a multi-line `**Description:**` block
  requires nested block parsing with no deterministic terminator. Title text is a single
  line with clear boundaries and is sufficient for `Summary` purposes. Richer summaries can
  be added later if title-only summaries prove inadequate in practice.
```

Now let me register the document: