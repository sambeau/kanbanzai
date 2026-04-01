# Specification: Decomposition Quality Validation (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J26CH63 (decomposition-quality-validation)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §11
**Status:** Draft

---

## Overview

This specification defines five structural validation checks that the `decompose(action: "review")` step MUST perform on a decomposition proposal before it can be applied. Decomposition quality is the strongest lever for multi-agent workflow performance — "performance gains correlate almost linearly with the quality of the induced task graph" (Masters et al.). These checks catch structural defects (missing descriptions, undeclared dependencies, oversized tasks, missing test coverage, and orphan tasks) early in the `propose → review → apply` pipeline, preventing low-quality task graphs from being created. Checks are classified as either errors (which block `apply`) or warnings (which are surfaced to the orchestrator for discretionary action). The validation checks are additive — they complement the existing review checks (gap analysis, oversized estimates, cycle detection, ambiguity detection) and do not replace them.

---

## Scope

### In scope

- Five new structural validation checks integrated into `decompose(action: "review")`
- Error vs. warning severity classification for each check
- Error findings blocking proposal application
- Warning findings included in review output for orchestrator decision-making
- Concrete, deterministic heuristics for each check (no LLM evaluation)
- Extension of the review output format to include explicit severity per finding

### Explicitly excluded

- Changes to `decompose(action: "propose")` — proposal generation is unchanged
- Changes to `decompose(action: "apply")` — application logic is unchanged; it continues to rely on review status to determine whether a proposal should be applied
- Changes to `decompose(action: "slice")` — slice analysis is unchanged
- LLM-based assessment of task description quality or completeness
- Evaluation of whether the decomposition covers the specification (this is the existing gap-analysis check, which is already implemented and out of scope for this feature)
- Semantic analysis of task descriptions beyond the defined pattern-matching heuristics
- Changes to the `Proposal` or `ProposedTask` input structures

---

## Functional Requirements

**FR-001:** The `decompose(action: "review")` step MUST perform a **description-present** check on every task in the proposal. A task fails this check if its `summary` field is empty or contains only whitespace after trimming. This check has **error** severity — any failure MUST block the proposal from being applied.

**Acceptance criteria:**
- A proposal where every task has a non-empty, non-whitespace-only summary produces no description-present findings
- A proposal containing a task with an empty string (`""`) summary produces an error-severity finding identifying the task by slug
- A proposal containing a task with a whitespace-only summary (`"   "`) produces an error-severity finding identifying the task by slug
- A proposal containing multiple tasks with empty summaries produces one finding per offending task
- The presence of any description-present finding causes the review status to be `fail`

---

**FR-002:** The `decompose(action: "review")` step MUST perform a **dependencies-declared** check on the proposal. A task A is considered to "reference" another task B if task A's `summary` or `rationale` contains the literal slug of task B as a substring (case-insensitive match). If task A references task B, then either B MUST appear in A's `depends_on` list OR A MUST appear in B's `depends_on` list. If neither condition holds, the check emits a finding. This check has **warning** severity.

**Acceptance criteria:**
- A proposal where task `setup-database` has summary "Create the database schema" and task `add-api` has summary "Add the API endpoints" (no cross-references) produces no dependencies-declared findings
- A proposal where task `add-api` has summary "Build API layer on top of setup-database work" and `setup-database` is NOT in `add-api.depends_on` produces a warning-severity finding identifying both slugs
- A proposal where task `add-api` references `setup-database` in its rationale and `setup-database` IS in `add-api.depends_on` produces no finding for that pair
- A proposal where task `add-api` references `setup-database` and `setup-database` lists `add-api` in its own `depends_on` produces no finding (reverse dependency satisfies the check)
- Slug matching is case-insensitive: a summary containing "Setup-Database" matches slug `setup-database`
- A slug appearing as a substring of a longer word does NOT count as a reference (e.g., slug `api` does not match the word "capital"). Slug matches MUST occur at word boundaries

---

**FR-003:** The `decompose(action: "review")` step MUST perform a **single-agent-sizing** check on every task in the proposal. A task fails this check if its summary suggests multiple independent work items. The heuristic MUST detect summaries that contain two or more action clauses joined by a coordinating separator. Specifically:

- **Action verbs:** implement, add, create, refactor, update, fix, remove, delete, migrate, configure, write, build, set up, modify, change, extract, move, rename, convert, integrate, replace, introduce, extend, redesign, rewrite.
- **Coordinating separators:** " and " (the word "and" surrounded by spaces), " as well as ", " additionally ", " plus ", or a semicolon followed by a space ("; ").
- A task triggers this check if its summary matches the pattern: `<action verb> ... <separator> ... <action verb>` — that is, two or more distinct action verbs appear with at least one coordinating separator between them.

This check has **warning** severity.

**Acceptance criteria:**
- A task with summary "Implement the login endpoint" produces no single-agent-sizing finding
- A task with summary "Implement the login endpoint and add the registration endpoint" produces a warning-severity finding (two action verbs: "implement" and "add", separated by " and ")
- A task with summary "Implement the login endpoint and verify it handles errors" does NOT produce a finding — "verify" is not in the action verb list, so there is only one action clause
- A task with summary "Refactor the database layer; migrate to the new ORM" produces a warning-severity finding (two action verbs separated by "; ")
- A task with summary "Build the CLI tool as well as create the configuration parser" produces a warning-severity finding
- A task with summary "Implement request and response handling" does NOT produce a finding — there is only one action verb ("implement"); "and" connects nouns, not action clauses
- The action verb match MUST occur at the start of a clause (after a separator or at the start of the summary), not as a substring within a word (e.g., "replace" does not match within "irreplaceable")
- The finding detail MUST identify the task slug and quote the matched action verbs and separator

---

**FR-004:** The `decompose(action: "review")` step MUST perform a **testing-coverage** check on the proposal as a whole. The proposal passes this check if at least one task's `summary` or `rationale` contains one or more of the following keywords (case-insensitive, whole-word match): "test", "tests", "testing", "verify", "verifies", "verification", "validate", "validates", "validation", "spec" (as in test spec), "coverage", "assert", "assertion", "assertions". The proposal fails this check if no task matches any of these keywords. This check has **warning** severity and produces at most one finding per proposal (it is a proposal-level check, not a per-task check).

**Acceptance criteria:**
- A proposal where one task has summary "Write unit tests for the API layer" produces no testing-coverage finding
- A proposal where one task has rationale "This task includes verification of edge cases" produces no testing-coverage finding
- A proposal where no task mentions any testing keyword produces a single warning-severity finding with no `task_slug` (it is a proposal-level finding)
- A proposal where the only testing keyword appears in a task's `rationale` (not `summary`) still passes — both fields are checked
- The keyword match MUST be whole-word: "contest" does not satisfy the check via substring match on "test"
- The finding detail MUST state that no task in the proposal addresses testing or verification

---

**FR-005:** The `decompose(action: "review")` step MUST perform a **no-orphan-tasks** check on the proposal. This check applies only when the proposal contains at least one dependency relationship (at least one task has a non-empty `depends_on` list). When it applies, the check builds an undirected graph where each task is a node and each dependency relationship adds an edge between the two tasks. A task is an **orphan** if it is in a connected component by itself — that is, it has no dependency edges (neither depends on another task nor is depended upon by another task) while other tasks in the proposal do have dependency edges. This check has **warning** severity and produces one finding per orphan task.

**Acceptance criteria:**
- A proposal where no task has any `depends_on` entries produces no orphan-task findings (the check does not apply when there are no dependencies)
- A proposal with tasks A→B→C (A depends on B, B depends on C) and task D with no dependencies and no task depending on D produces a warning-severity finding identifying D as an orphan
- A proposal with tasks A→B and C→D (two separate chains) produces no orphan-task findings — all tasks participate in at least one dependency relationship
- A proposal with tasks A→B→C where all tasks are connected produces no orphan-task findings
- A proposal with a single task (no dependencies possible) produces no findings
- Dependencies referencing slugs not present in the proposal (e.g., cross-feature dependencies) are ignored for the purpose of this check — only intra-proposal relationships count
- The finding detail MUST identify the orphan task by slug and state that it is disconnected from the dependency graph

---

**FR-006:** Each finding produced by the validation checks MUST include an explicit **severity** field in the review output with value `"error"` or `"warning"`. The existing finding fields (`type`, `task_slug`, `detail`) MUST continue to be present. The new finding types MUST use the following type identifiers:

| Check | Finding type | Severity |
|---|---|---|
| Description present | `empty-description` | `error` |
| Dependencies declared | `undeclared-dependency` | `warning` |
| Single-agent sizing | `multi-agent-sizing` | `warning` |
| Testing coverage | `missing-test-coverage` | `warning` |
| No orphan tasks | `orphan-task` | `warning` |

**Acceptance criteria:**
- Every finding in the review output includes a `severity` field with value `"error"` or `"warning"`
- The `type` field for each new check uses the identifier from the table above
- Existing finding types (`gap`, `oversized`, `cycle`, `ambiguous`) continue to function and are unaffected by this change

---

**FR-007:** The review output's `status` field MUST reflect the combined result of all checks — both the existing checks and the new validation checks. The status determination rules are:

- `"fail"` — if `blocking_count > 0` (any error-severity finding exists)
- `"warn"` — if `blocking_count == 0` and `total_findings > 0`
- `"pass"` — if `total_findings == 0`

The `blocking_count` field MUST count only error-severity findings. Findings with `"error"` severity from the new checks (currently only `empty-description`) MUST be counted alongside existing blocking finding types (`gap`, `cycle`).

**Acceptance criteria:**
- A proposal that triggers only warning-severity findings produces status `"warn"` with `blocking_count: 0`
- A proposal that triggers an `empty-description` finding produces status `"fail"` with `blocking_count >= 1`
- A proposal that triggers both an `empty-description` finding and a `missing-test-coverage` finding produces status `"fail"` with `blocking_count` equal to the number of error-severity findings only
- A proposal that passes all checks (existing and new) with no findings produces status `"pass"`
- The `total_findings` field counts all findings regardless of severity

---

**FR-008:** The new validation checks MUST be additive — they MUST NOT replace, modify, or interfere with the existing review checks (gap analysis, oversized estimate detection, cycle detection, ambiguity detection). The review step MUST run all existing checks AND all new validation checks, collecting findings from both into a single findings list.

**Acceptance criteria:**
- A proposal that triggers both an existing `gap` finding and a new `orphan-task` finding includes both findings in the output
- The order of checks does not affect the results — all checks run independently and their findings are combined
- Removing a new validation check (hypothetically) would leave all existing checks functional and unchanged

---

**FR-009:** Existing finding types MUST be assigned explicit severity values for consistency with the new output format. The mapping MUST be:

| Existing type | Severity |
|---|---|
| `gap` | `error` |
| `cycle` | `error` |
| `oversized` | `warning` |
| `ambiguous` | `warning` |

This mapping is consistent with the existing `isBlockingFinding` logic — types that currently block (`gap`, `cycle`) become `error`; types that currently do not block (`oversized`, `ambiguous`) become `warning`.

**Acceptance criteria:**
- An existing `gap` finding includes `severity: "error"` in the output
- An existing `cycle` finding includes `severity: "error"` in the output
- An existing `oversized` finding includes `severity: "warning"` in the output
- An existing `ambiguous` finding includes `severity: "warning"` in the output
- The blocking behavior of existing finding types is unchanged

---

## Non-Functional Requirements

**NFR-001:** The validation checks MUST execute in O(n²) time or better, where n is the number of tasks in the proposal. No check may introduce cubic or worse complexity. This ensures the review step remains fast for proposals with up to 100 tasks.

**NFR-002:** The validation checks MUST be deterministic — given the same proposal input, they MUST always produce the same findings in the same order. No randomness or external state (beyond the proposal itself) may influence the check results.

**NFR-003:** The validation checks MUST NOT make network calls, read from the filesystem, or depend on any state external to the proposal object passed to the review step. The checks operate purely on the structural content of the `Proposal` and its `ProposedTask` entries. (This distinguishes them from existing checks like gap analysis, which reads the spec document.)

**NFR-004:** The validation checks MUST NOT change the MCP tool interface — no new parameters, no removed parameters, no changes to required/optional status of existing parameters. The only output change is the addition of the `severity` field to each finding and the new finding types.

**NFR-005:** All new validation check logic MUST have unit test coverage. Each check MUST be testable in isolation (given a `Proposal`, return findings) without requiring entity store, document store, or other service dependencies.

---

## Acceptance Criteria

| Requirement | Verification method |
|---|---|
| FR-001 (description present) | Unit tests: empty summary, whitespace-only summary, valid summary, multiple empty summaries |
| FR-002 (dependencies declared) | Unit tests: no cross-references, slug in summary without dep, slug in rationale with dep, reverse dep, case-insensitive match, word boundary match |
| FR-003 (single-agent sizing) | Unit tests: single action verb, two action verbs with "and", two action verbs with ";", noun conjunction (no match), verb not at clause start, substring non-match |
| FR-004 (testing coverage) | Unit tests: keyword in summary, keyword in rationale, no keywords anywhere, whole-word match only |
| FR-005 (no orphan tasks) | Unit tests: no dependencies (skip), connected graph, disconnected task, multiple disconnected tasks, cross-feature deps ignored |
| FR-006 (finding format) | Unit tests: all new finding types include severity field with correct value; existing types also include severity |
| FR-007 (status determination) | Unit tests: pass/warn/fail scenarios combining old and new findings |
| FR-008 (additive checks) | Unit tests: mixed findings from old and new checks appear together in output |
| FR-009 (existing severity mapping) | Unit tests: existing finding types carry correct severity values |
| NFR-001 (performance) | Code review: verify no nested loops beyond O(n²) |
| NFR-002 (determinism) | Unit tests: same input produces identical output across multiple runs |
| NFR-003 (no external state) | Code review: new check functions accept only `Proposal` (no service dependencies) |
| NFR-004 (interface stability) | Integration test: existing callers of `decompose(action: "review")` continue to work without changes |
| NFR-005 (test coverage) | Each check function has dedicated test cases covering pass and fail paths |

---

## Dependencies and Assumptions

1. **Existing review infrastructure:** The `DecomposeService.ReviewProposal` method and its existing checks (`checkGaps`, `checkOversized`, `checkCycles`, `checkAmbiguous`) are stable and will not be modified by other concurrent work. The new checks integrate alongside them.

2. **Proposal structure:** The `ProposedTask` struct's existing fields (`Slug`, `Summary`, `Rationale`, `DependsOn`) are sufficient for all five checks. No changes to the proposal input format are required.

3. **Finding type extensibility:** The `Finding` struct can be extended with a `Severity` field (or equivalent) without breaking existing consumers, because the MCP tool output is JSON and new fields are additive.

4. **Slug uniqueness:** Task slugs within a single proposal are assumed to be unique. If duplicate slugs exist, the dependency-declared and orphan-task checks may produce undefined results (this is an existing assumption in the `apply` action's slug-to-ID mapping).

5. **Action verb list stability:** The action verb list in FR-003 is a fixed set defined by this specification. Adding or removing verbs is a specification change, not a configuration change. The list is intentionally conservative to minimize false positives.

6. **Testing keyword list stability:** The testing keyword list in FR-004 is a fixed set defined by this specification. The same stability guarantee applies.