# P11 Feature B: Embedded Skills — Dev Plan

| Document | P11 Feature B Dev Plan                                          |
|----------|-----------------------------------------------------------------|
| Feature  | FEAT-01KMWJ3ZQKK7C — embedded-skills                           |
| Status   | Draft                                                           |
| Related  | `work/spec/spec-embedded-skills.md`                             |
|          | `work/design/fresh-install-experience.md` §5.2, decision FI-D-002 |
|          | `work/spec/skills-content.md`                                   |
|          | `work/spec/doc-currency-health-check.md`                        |
|          | `work/spec/review-lifecycle-states.md`                          |
|          | `internal/kbzinit/skills.go`                                    |
|          | `internal/mcp/doc_currency_health.go`                           |

---

## 1. Implementation Approach

This feature has two distinct bodies of work that can proceed independently:

**Skill content (Tasks 1–3).** Four skill files are created or updated under
`internal/kbzinit/skills/`. Two are new (`review`, `plan-review`); two are
amended in-place (`workflow`, `documents`). Each new skill is registered in the
`skillNames` slice in `skills.go` so that `kbz init` embeds and installs it.
The embedded asset set grows from six to eight skills, satisfying AC-01 through
AC-05.

**Health checker update (Task 4).** The `collectTier1Files` function in
`internal/mcp/doc_currency_health.go` currently walks `.skills/*.md`. It is
updated to walk `.agents/skills/kanbanzai-*/SKILL.md` instead, matching the
actual install location. The accompanying test helper is updated to write
synthetic skill files to the new path so all existing tests continue to pass.

Tasks 1, 2, and 4 are fully independent and may be worked in parallel. Task 3
depends on the `plan` and `retrospective` document type names introduced by
Feature D (document-layout); it should be implemented after or alongside
Feature D.

---

## 2. Task Breakdown

| # | Task | Files Touched | Estimate |
|---|------|--------------|----------|
| 1 | Write `kanbanzai-review` skill | `internal/kbzinit/skills/review/SKILL.md` (new), `internal/kbzinit/skills.go` | M |
| 2 | Write `kanbanzai-plan-review` skill | `internal/kbzinit/skills/plan-review/SKILL.md` (new), `internal/kbzinit/skills.go` | M |
| 3 | Update `kanbanzai-workflow` and `kanbanzai-documents` skills | `internal/kbzinit/skills/workflow/SKILL.md`, `internal/kbzinit/skills/documents/SKILL.md` | S |
| 4 | Update `doc_currency_health` checker | `internal/mcp/doc_currency_health.go`, `internal/mcp/doc_currency_health_test.go` | S |

---

## 3. Task Details

### Task 1: Write `kanbanzai-review` skill

**New file:** `internal/kbzinit/skills/review/SKILL.md`

A complete, self-contained code review skill for agent use. Must cover:

- **Inputs section** — feature ID, associated specification document(s), worktree
  branch or PR diff, dev-plan document. Review may not begin until all four are
  available (AC-06).
- **Five-dimension evaluation guidance** — one section per dimension: spec
  conformance, implementation quality, test adequacy, documentation currency,
  workflow integrity. Each section states what to check and what constitutes a
  passing result (AC-07).
- **Structured output format** — per-dimension outcome block (pass / fail /
  partial with supporting findings), overall verdict (approved /
  approved-with-notes / changes-required), and a findings list separating
  blocking from non-blocking items (AC-08, AC-09).
- **Edge cases section** — how to handle a feature with no associated
  specification document, and how to handle partial spec satisfaction (AC-10).
- **Orchestration procedure** — sequential MCP tool calls: context assembly,
  sub-agent dispatch, findings collation, report writing, feature transition to
  `done` or `needs-rework` (AC-11).
- Review reports written to `work/review/` with naming convention
  `review-{feature-id}-{slug}.md` (AC-12).
- No retired 1.0 tool names (AC-13).
- YAML frontmatter with `kanbanzai-managed: true` marker (AC-03).

**Edit:** `internal/kbzinit/skills.go` — add `"review"` to the `skillNames`
slice (after `"planning"`, before `"workflow"`, or at the end — maintain
alphabetical or logical order consistent with existing entries).

---

### Task 2: Write `kanbanzai-plan-review` skill

**New file:** `internal/kbzinit/skills/plan-review/SKILL.md`

A complete, self-contained plan review skill for agent use. Must cover:

- **Inputs section** — plan ID, plan's associated documents, list of features
  under the plan with their final statuses (AC-14).
- **Plan scope verification** — confirm plan goals are addressed by features;
  confirm no mid-plan scope additions lack a documented scope decision (AC-15).
- **Feature completion checks** — verify all features are in a terminal state
  (`done`, `cancelled`, or `superseded`); flag any non-terminal features (AC-16).
- **Spec conformance** — confirm each `done` feature has an associated
  specification document in `approved` status (AC-17).
- **Documentation currency** — check `AGENTS.md` Scope Guard mentions the plan;
  confirm no feature spec documents under the plan remain in `draft` status (AC-18).
- **Cross-cutting checks** — known issues or deferred items recorded as bugs or
  future features; decision log captures key architectural decisions (AC-19).
- **Retrospective contribution** — instruct reviewer to contribute a
  retrospective signal via the `retro` tool summarising execution, friction, and
  tool gaps (AC-20).
- **Report format** — overall verdict (approved / changes-required), checklist
  of all checks with pass/fail outcomes, findings list with blocking and
  non-blocking items separated (AC-21).
- Reports written to `work/review/` with naming convention
  `review-{plan-id}-{slug}.md` (AC-22).
- No retired 1.0 tool names (AC-23).
- YAML frontmatter with `kanbanzai-managed: true` marker (AC-03).

**Edit:** `internal/kbzinit/skills.go` — add `"plan-review"` to the
`skillNames` slice alongside the `"review"` entry added in Task 1.

---

### Task 3: Update `kanbanzai-workflow` and `kanbanzai-documents` skills

> **Note:** This task references the `plan` and `retrospective` document type
> names introduced by Feature D (document-layout). Implement after or alongside
> Feature D to avoid inconsistency.

**Edit:** `internal/kbzinit/skills/workflow/SKILL.md`

Add a Stage Gates entry covering the `reviewing` and `needs-rework` lifecycle
states:

- `reviewing` — entered when implementation is complete; feature may not be
  merged until the review concludes (AC-24, AC-25).
- `needs-rework` — entered when a reviewer raises blocking findings; resolved by
  addressing findings and transitioning back to `reviewing` (or `developing` for
  substantial rework), after which a new review is conducted (AC-26, AC-27).
- Cross-reference `kanbanzai-review` for the full review procedure; do not
  inline the procedure (AC-28).
- Ensure no `developing → done` direct transition appears in the lifecycle
  table (AC-29).

**Edit:** `internal/kbzinit/skills/documents/SKILL.md`

Update the document type/directory table to eight rows:

| Directory | Document type |
|-----------|---------------|
| `work/design/` | `design` |
| `work/spec/` | `specification` |
| `work/plan/` | `plan` |
| `work/dev/` | `dev-plan` |
| `work/research/` | `research` |
| `work/report/` | `report` |
| `work/review/` | `report` |
| `work/retro/` | `retrospective` |

Update the description field (or equivalent prose) to list all eight document
roots. Remove any reference to the `.skills/` path (AC-30, AC-31, AC-32,
AC-33).

---

### Task 4: Update `doc_currency_health` checker

**Edit:** `internal/mcp/doc_currency_health.go`

Replace the `.skills/*.md` scan in `collectTier1Files` with a
`.agents/skills/kanbanzai-*/SKILL.md` walk:

- Read the subdirectories of `.agents/skills/`, filter to those whose name
  starts with `kanbanzai-`, and collect the `SKILL.md` file within each (using
  `os.ReadDir` or `filepath.Glob`).
- Remove all references to `.skills/` from the function body.
- The `AGENTS.md` scan at the repo root is unchanged (AC-34, AC-35, AC-36).

**Edit:** `internal/mcp/doc_currency_health_test.go`

- Update the `writeSkillFile` helper to write to
  `.agents/skills/kanbanzai-<name>/SKILL.md` instead of `.skills/<name>.md`.
  The function signature may gain a `skillDirName` parameter or the name
  argument may be interpreted as the subdirectory name.
- Update `TestDocCurrencyHealth_DetectsStaleToolName` to create the synthetic
  skill at `.agents/skills/kanbanzai-example/SKILL.md` and update the expected
  warning path string accordingly (AC-37).
- Adjust any other test that contains a `.skills/` string literal to use the
  new path (AC-38).
- Verify `go test -race ./internal/mcp/...` passes (AC-39).

---

## 4. Dependencies

| Task | Depends on | Notes |
|------|-----------|-------|
| Task 1 | — | Independent; can start immediately |
| Task 2 | — | Independent; can start immediately |
| Task 3 | Feature D (document-layout) | Requires `plan` and `retrospective` type names to exist before updating the documents skill table |
| Task 4 | — | Independent; can start immediately |
| Tasks 1 & 2 | Each other (skills.go) | Both edit `skillNames` in `skills.go`; coordinate to avoid a merge conflict or sequence them |

Tasks 1, 2, and 4 can be executed in parallel provided the `skills.go` edit is
handled as a single coordinated change. Task 3 is gated on Feature D.

### External dependencies

| Item | Owner | Status |
|------|-------|--------|
| `plan` and `retrospective` document type names | Feature D (P11) | Required before Task 3 |
| `reviewing` / `needs-rework` lifecycle states in the state machine | `work/spec/review-lifecycle-states.md` | Must be implemented before Task 3's workflow skill update is meaningful |

### Acceptance criteria cross-reference

All acceptance criteria from `work/spec/spec-embedded-skills.md` are covered by
the tasks above:

| Spec section | AC range | Covered by |
|---|---|---|
| §3.1 New skills installed by `kbz init` | AC-01 – AC-05 | Tasks 1, 2 (skill files + `skillNames`) |
| §3.2 `kanbanzai-review` skill content | AC-06 – AC-13 | Task 1 |
| §3.3 `kanbanzai-plan-review` skill content | AC-14 – AC-23 | Task 2 |
| §3.4 `kanbanzai-workflow` skill update | AC-24 – AC-29 | Task 3 |
| §3.5 `kanbanzai-documents` skill update | AC-30 – AC-33 | Task 3 |
| §3.6 `doc_currency_health` checker update | AC-34 – AC-39 | Task 4 |