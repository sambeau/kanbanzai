# Embedded Skills Specification

| Document | Embedded Skills Specification                                     |
|----------|-------------------------------------------------------------------|
| Status   | Draft                                                             |
| Feature  | FEAT-01KMWJ3ZQKK7C                                               |
| Related  | `work/design/fresh-install-experience.md` §5.2, decision FI-D-002 |
|          | `work/spec/skills-content.md`                                     |
|          | `work/spec/doc-currency-health-check.md`                          |
|          | `work/spec/review-lifecycle-states.md`                            |

---

## 1. Purpose

This specification defines the acceptance criteria for Feature B of the P11 Fresh Install Experience plan: the two new embedded skills added to `kbz init`, the updates to two existing skills, and the corresponding change to the `doc_currency_health` checker's scan targets.

After this feature ships:

- `kbz init` installs eight embedded skills instead of six.
- The two new skills (`kanbanzai-review`, `kanbanzai-plan-review`) provide agents with canonical, self-contained procedures for conducting code reviews and plan reviews.
- The `kanbanzai-workflow` skill fully describes the feature lifecycle, including the `reviewing` and `needs-rework` states that were previously absent.
- The `kanbanzai-documents` skill reflects the eight-directory document layout introduced in Feature C, including the two new document types.
- The `doc_currency_health` Tier 1 checker scans the new skill location (`.agents/skills/kanbanzai-*/SKILL.md`) instead of the retired `.skills/*.md` location.

---

## 2. Scope

### 2.1 In scope

- Two new skill files: `kanbanzai-review` and `kanbanzai-plan-review`.
- Updates to two existing skill files: `kanbanzai-workflow` and `kanbanzai-documents`.
- Update to `internal/mcp/doc_currency_health.go`: change the Tier 1 scan target from `.skills/*.md` to `.agents/skills/kanbanzai-*/SKILL.md`.
- Update to `internal/mcp/doc_currency_health_test.go` to reflect the new scan target.
- Registration of both new skill files in the `kbz init` embedded asset set so they are installed and updated by `--update-skills`.

### 2.2 Out of scope

- Content authoring for the other six skills (addressed in `work/spec/skills-content.md`).
- Changes to the `kanbanzai-agents`, `kanbanzai-getting-started`, `kanbanzai-planning`, or `kanbanzai-design` skills.
- The `kbz init` command implementation itself (addressed in Feature A).
- The `reviewer.yaml` context role (addressed in Feature C of P6).
- Changes to the feature lifecycle state machine (addressed in `work/spec/review-lifecycle-states.md`).

---

## 3. Acceptance Criteria

### 3.1 New skills installed by `kbz init`

**AC-01.** Running `kbz init` on a new project creates the file `.agents/skills/kanbanzai-review/SKILL.md`.

**AC-02.** Running `kbz init` on a new project creates the file `.agents/skills/kanbanzai-plan-review/SKILL.md`.

**AC-03.** Both new skill files contain the `kanbanzai-managed: true` marker in their YAML frontmatter, consistent with the other six embedded skills.

**AC-04.** Running `kbz init --update-skills` on an existing project overwrites `.agents/skills/kanbanzai-review/SKILL.md` and `.agents/skills/kanbanzai-plan-review/SKILL.md` with the current embedded versions.

**AC-05.** After `kbz init` completes, exactly eight skill directories exist under `.agents/skills/`: `kanbanzai-agents`, `kanbanzai-design`, `kanbanzai-documents`, `kanbanzai-getting-started`, `kanbanzai-planning`, `kanbanzai-workflow`, `kanbanzai-review`, `kanbanzai-plan-review`.

---

### 3.2 `kanbanzai-review` skill content

**AC-06.** The skill opens with a section that lists the inputs required before a review may begin: the feature ID, the feature's associated specification document(s), the worktree branch or PR diff, and the feature's dev-plan document.

**AC-07.** The skill contains per-dimension evaluation guidance for all five review dimensions: spec conformance, implementation quality, test adequacy, documentation currency, and workflow integrity. Each dimension section states what the reviewer must check and what constitutes a passing result.

**AC-08.** The skill defines a structured output format for the review report. The format includes: a per-dimension outcome block (pass/fail/partial with supporting findings), an overall verdict (approved, approved-with-notes, or changes-required), and a findings list that separates blocking issues from non-blocking observations.

**AC-09.** The skill defines the finding classification rule: a finding is blocking if it would prevent the feature from being merged to main; all other findings are non-blocking.

**AC-10.** The skill contains an edge case section covering at minimum: how to handle a feature with no associated specification document, and how to handle a review where the implementation partially satisfies the spec.

**AC-11.** The skill contains a complete orchestration procedure describing the sequence of MCP tool calls an orchestrating agent should make when conducting a review: how to assemble context, dispatch review sub-agents, collect results, write the report, and transition the feature to `done` or `needs-rework`.

**AC-12.** The skill states that review reports are written to `work/review/` and names the file naming convention `review-{feature-id}-{slug}.md`.

**AC-13.** The skill does not reference any retired 1.0 tool name. Every MCP tool reference uses a name from the Kanbanzai 2.0 tool set.

---

### 3.3 `kanbanzai-plan-review` skill content

**AC-14.** The skill opens with a section that lists the inputs required before a plan review may begin: the plan ID, the plan's associated documents, and the list of features under the plan with their final statuses.

**AC-15.** The skill contains a plan scope verification section: it instructs the reviewer to confirm that the plan's stated goals are addressed by its features, and that no features were added mid-plan without a documented scope decision.

**AC-16.** The skill contains a feature completion checks section: it instructs the reviewer to verify that all features under the plan are in a terminal state (`done`, `cancelled`, or `superseded`) and that none remain in a non-terminal state.

**AC-17.** The skill contains a spec conformance section: it instructs the reviewer to confirm that each `done` feature has an associated specification document in `approved` status.

**AC-18.** The skill contains a documentation currency section: it instructs the reviewer to check that `AGENTS.md` Scope Guard mentions the plan, and that no feature spec documents under the plan remain in `draft` status.

**AC-19.** The skill contains a cross-cutting checks section covering at minimum: whether any known issues or deferred items were recorded as bugs or future features, and whether the plan's decision log captures the key architectural decisions made.

**AC-20.** The skill instructs the reviewer to contribute a retrospective signal (via the `retro` tool) summarising the plan's execution: what worked, what caused friction, and any tool gaps discovered.

**AC-21.** The skill defines the plan review report format, including: an overall verdict (approved or changes-required), a checklist of the checks in §3.3 above with pass/fail outcomes, and a findings list with blocking and non-blocking items separated.

**AC-22.** The skill states that plan review reports are written to `work/review/` and names the file naming convention `review-{plan-id}-{slug}.md`.

**AC-23.** The skill does not reference any retired 1.0 tool name. Every MCP tool reference uses a name from the Kanbanzai 2.0 tool set.

---

### 3.4 `kanbanzai-workflow` skill update

**AC-24.** The skill's feature lifecycle state table or narrative includes the `reviewing` state and states that it is entered when an agent transitions a feature from `developing` after implementation is complete.

**AC-25.** The skill describes what the `reviewing` state means: implementation is complete, a review is in progress, and the feature may not be merged until the review concludes.

**AC-26.** The skill's feature lifecycle state table or narrative includes the `needs-rework` state and describes how it is entered: a reviewer transitions the feature from `reviewing` to `needs-rework` when blocking findings are raised.

**AC-27.** The skill describes how `needs-rework` is resolved: the implementing agent addresses the blocking findings and transitions the feature back to `reviewing` (or to `developing` if substantial rework is required), after which a new review is conducted.

**AC-28.** The skill directs agents to read `kanbanzai-review` for the full review procedure. It does not inline the review procedure itself.

**AC-29.** The feature lifecycle table in the skill is consistent with the transition map defined in `work/spec/review-lifecycle-states.md` AC-03: no `developing → done` shortcut appears.

---

### 3.5 `kanbanzai-documents` skill update

**AC-30.** The skill contains a document type and directory table that lists all eight directories introduced in Feature C:

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

**AC-31.** The table in AC-30 includes entries for the two new document types: `plan` (mapped to `work/plan/`) and `retrospective` (mapped to `work/retro/`).

**AC-32.** The skill does not contain any reference to the string `.skills/` anywhere in its content.

**AC-33.** The skill does not reference any retired 1.0 tool name. Every MCP tool reference uses a name from the Kanbanzai 2.0 tool set.

---

### 3.6 `doc_currency_health` checker update

**AC-34.** The `collectTier1Files` function in `internal/mcp/doc_currency_health.go` scans `.agents/skills/kanbanzai-*/SKILL.md` files (using glob or directory walk) rather than `.skills/*.md`.

**AC-35.** The `collectTier1Files` function does not scan `.skills/` at all. No file path containing `.skills/` is returned by the function.

**AC-36.** The Tier 1 check continues to scan `AGENTS.md` at the repo root, unchanged.

**AC-37.** The existing test `TestDocCurrencyHealth_DetectsStaleToolName` is updated to create a synthetic skill file at `.agents/skills/kanbanzai-example/SKILL.md` (instead of `.skills/example.md`) and the expected warning message references the new path.

**AC-38.** All other existing tests in `internal/mcp/doc_currency_health_test.go` continue to pass without modification to their assertions, adjusted only where a file path string literal refers to `.skills/`.

**AC-39.** `go test -race ./internal/mcp/...` passes after the changes.

---

## 4. Verification

| AC | Verification method |
|----|---------------------|
| AC-01 | Integration test: `kbz init` in a temp directory; assert `.agents/skills/kanbanzai-review/SKILL.md` exists |
| AC-02 | Integration test: `kbz init` in a temp directory; assert `.agents/skills/kanbanzai-plan-review/SKILL.md` exists |
| AC-03 | File inspection: grep for `kanbanzai-managed: true` in both new SKILL.md files |
| AC-04 | Integration test: `kbz init --update-skills` overwrites both files; assert content matches embedded version |
| AC-05 | Integration test: `kbz init`; assert exactly 8 subdirectories under `.agents/skills/` |
| AC-06 | Manual review: skill file contains an inputs section listing the four required inputs |
| AC-07 | Manual review: skill file contains five labelled dimension sections |
| AC-08 | Manual review: skill file defines structured output format with per-dimension outcomes, overall verdict, and separated findings |
| AC-09 | Manual review: skill file states the blocking/non-blocking classification rule |
| AC-10 | Manual review: skill file contains an edge case section covering the two specified scenarios |
| AC-11 | Manual review: skill file contains a sequential orchestration procedure referencing MCP tool calls |
| AC-12 | Manual review: skill file references `work/review/` and names the convention `review-{feature-id}-{slug}.md` |
| AC-13 | `grep` all 1.0 tool names against the skill file; zero matches |
| AC-14 | Manual review: plan review skill file contains an inputs section listing the three required inputs |
| AC-15 | Manual review: plan review skill file contains a plan scope verification section |
| AC-16 | Manual review: plan review skill file contains a feature completion checks section |
| AC-17 | Manual review: plan review skill file contains a spec conformance section |
| AC-18 | Manual review: plan review skill file contains a documentation currency section |
| AC-19 | Manual review: plan review skill file contains a cross-cutting checks section |
| AC-20 | Manual review: plan review skill file instructs the reviewer to contribute via the `retro` tool |
| AC-21 | Manual review: plan review skill file defines a report format with overall verdict, checklist, and separated findings |
| AC-22 | Manual review: plan review skill file references `work/review/` and names the convention `review-{plan-id}-{slug}.md` |
| AC-23 | `grep` all 1.0 tool names against the plan review skill file; zero matches |
| AC-24 | Manual review: `kanbanzai-workflow` skill includes `reviewing` in its lifecycle state coverage |
| AC-25 | Manual review: `kanbanzai-workflow` skill describes the meaning of the `reviewing` state |
| AC-26 | Manual review: `kanbanzai-workflow` skill includes `needs-rework` in its lifecycle state coverage |
| AC-27 | Manual review: `kanbanzai-workflow` skill describes how `needs-rework` is resolved |
| AC-28 | Manual review: `kanbanzai-workflow` skill cross-references `kanbanzai-review`; does not inline the procedure |
| AC-29 | Manual review: no `developing → done` transition appears in the workflow skill lifecycle table |
| AC-30 | Manual review: `kanbanzai-documents` skill contains a table with all eight directory rows |
| AC-31 | Manual review: the table includes `plan` and `retrospective` document type entries |
| AC-32 | `grep -r '\.skills/' .agents/skills/kanbanzai-documents/`; zero matches |
| AC-33 | `grep` all 1.0 tool names against the documents skill file; zero matches |
| AC-34 | Code review: `collectTier1Files` walks `.agents/skills/kanbanzai-*/SKILL.md` |
| AC-35 | Code review: no `.skills/` path string appears in `collectTier1Files` |
| AC-36 | Code review: `AGENTS.md` scan path is unchanged |
| AC-37 | Test review: `TestDocCurrencyHealth_DetectsStaleToolName` references `.agents/skills/kanbanzai-example/SKILL.md` |
| AC-38 | `go test ./internal/mcp/...`; all pre-existing tests pass |
| AC-39 | `go test -race ./internal/mcp/...`; no failures or data races |