# Implementation Plan: Decomposition Quality Validation

**Feature:** FEAT-01KN5-8J26CH63 (decomposition-quality-validation)
**Specification:** `work/spec/3.0-decomposition-quality-validation.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §11

---

## Overview

This plan decomposes the Decomposition Quality Validation specification into 5 implementation tasks. The feature adds 5 structural validation checks to `decompose(action: "review")`, extends the `Finding` type with a severity field, and backfills severity onto existing finding types — all without changing the MCP tool interface.

The key architectural decision is to implement each check as a standalone pure function (`func(Proposal) []Finding`) in a new file, keeping the existing `decompose.go` largely untouched. Task 1 establishes the shared types and wiring; Tasks 2–4 implement the checks in parallel; Task 5 adds integration-level tests.

---

## Scope boundaries (carried forward from specification)

**In scope:** Five new validation checks, severity field on findings, status determination logic update, additive integration with existing checks.

**Out of scope:** Changes to `propose`, `apply`, or `slice` actions. LLM-based quality assessment. Changes to `Proposal` or `ProposedTask` input structures. Semantic analysis beyond defined heuristics.

---

## Task breakdown

### Task 1: Finding severity field and review wiring

**Objective:** Add a `Severity` field to the `Finding` struct, backfill severity onto existing finding types, and wire the new validation check entry points into `ReviewProposal`. After this task, the existing checks produce severity-annotated findings and `ReviewProposal` calls placeholder functions for the new checks (which initially return nil). The status determination logic uses severity instead of type-based `isBlockingFinding`.

**Specification references:** FR-006, FR-007, FR-008, FR-009

**Input context:**
- `internal/service/decompose.go` — `Finding` struct (L54–58), `DecomposeReviewResult` (L61–67), `ReviewProposal` (L206–260), `isBlockingFinding` (L838–845), existing `check*` functions (L682–834)
- `internal/service/decompose_test.go` — existing review tests (L385–683)

**Output artifacts:**
- Modified `internal/service/decompose.go`:
  - Add `Severity string \`json:"severity"\`` field to `Finding` struct (values: `"error"`, `"warning"`)
  - Update `checkGaps` and `checkCycles` to set `Severity: "error"` on their findings
  - Update `checkOversized` and `checkAmbiguous` to set `Severity: "warning"` on their findings
  - Replace `isBlockingFinding` to check `f.Severity == "error"` instead of switching on type
  - Add call sites in `ReviewProposal` for the five new check functions (imported from new file, initially returning nil)
- New file `internal/service/decompose_validate.go`:
  - Package declaration and stub functions: `checkDescriptionPresent`, `checkDependenciesDeclared`, `checkSingleAgentSizing`, `checkTestingCoverage`, `checkOrphanTasks` — each with signature `func(Proposal) []Finding` returning `nil`
- Modified `internal/service/decompose_test.go`:
  - Update existing tests to assert `Severity` field on findings from existing check types
  - Add tests for FR-009 severity mapping: `gap` → error, `cycle` → error, `oversized` → warning, `ambiguous` → warning
  - Add test that new checks are called (proposal with empty summary produces finding after Task 2)

**Dependencies:** None — this is the foundation task.

**Interface contract:** All check functions (existing and new) MUST return `[]Finding` where every `Finding` has `Severity` set to `"error"` or `"warning"`. The `ReviewProposal` method appends findings from all checks into a single slice. Status determination uses severity only:
- `"fail"` if any finding has `Severity == "error"`
- `"warn"` if findings exist but none have `Severity == "error"`
- `"pass"` if no findings

---

### Task 2: Description-present and testing-coverage checks

**Objective:** Implement the `checkDescriptionPresent` and `checkTestingCoverage` validation checks. These are the two simplest checks — one per-task, one per-proposal — and are grouped together because they are both straightforward string inspection with no graph logic.

**Specification references:** FR-001, FR-004

**Input context:**
- `internal/service/decompose_validate.go` — stub functions from Task 1
- `internal/service/decompose.go` — `Proposal`, `ProposedTask`, `Finding` types
- Specification FR-001 acceptance criteria (empty, whitespace-only, multiple empties)
- Specification FR-004 acceptance criteria (keyword list, whole-word match, summary and rationale checked, proposal-level finding)

**Output artifacts:**
- Modified `internal/service/decompose_validate.go`:
  - `checkDescriptionPresent(Proposal) []Finding` — iterates tasks, trims summary, emits `Finding{Type: "empty-description", Severity: "error", TaskSlug: task.Slug, Detail: ...}` for each empty/whitespace-only summary
  - `checkTestingCoverage(Proposal) []Finding` — scans all tasks' summary and rationale for testing keywords using whole-word regex match (`\b` boundaries), emits at most one `Finding{Type: "missing-test-coverage", Severity: "warning", Detail: ...}` with no `TaskSlug` if no match found
  - Testing keyword list as a package-level variable: `test`, `tests`, `testing`, `verify`, `verifies`, `verification`, `validate`, `validates`, `validation`, `spec`, `coverage`, `assert`, `assertion`, `assertions`
- New file `internal/service/decompose_validate_test.go`:
  - `TestCheckDescriptionPresent_AllValid` — no findings
  - `TestCheckDescriptionPresent_EmptySummary` — error finding with task slug
  - `TestCheckDescriptionPresent_WhitespaceOnly` — error finding
  - `TestCheckDescriptionPresent_MultipleEmpty` — one finding per offending task
  - `TestCheckTestingCoverage_KeywordInSummary` — no findings
  - `TestCheckTestingCoverage_KeywordInRationale` — no findings
  - `TestCheckTestingCoverage_NoKeywords` — single warning finding, no task_slug
  - `TestCheckTestingCoverage_WholeWordOnly` — "contest" does not match "test"
  - `TestCheckTestingCoverage_SpecKeyword` — "spec" matches

**Dependencies:** Task 1 (Finding struct with Severity field, stub functions in place)

---

### Task 3: Dependencies-declared and orphan-tasks checks

**Objective:** Implement the `checkDependenciesDeclared` and `checkOrphanTasks` validation checks. These are grouped because they both operate on the proposal's dependency graph structure and share slug-matching logic.

**Specification references:** FR-002, FR-005

**Input context:**
- `internal/service/decompose_validate.go` — stub functions from Task 1
- `internal/service/decompose.go` — `Proposal`, `ProposedTask`, `Finding` types
- Specification FR-002 acceptance criteria (slug references in summary/rationale, case-insensitive, word-boundary matching, bidirectional dependency satisfaction)
- Specification FR-005 acceptance criteria (undirected graph, orphan = zero-degree node when graph has edges, cross-feature deps ignored)

**Output artifacts:**
- Modified `internal/service/decompose_validate.go`:
  - `checkDependenciesDeclared(Proposal) []Finding` — for each task, scan summary and rationale for other tasks' slugs (case-insensitive, word-boundary match). If task A references task B's slug, verify B is in A's `depends_on` OR A is in B's `depends_on`. Emit `Finding{Type: "undeclared-dependency", Severity: "warning", TaskSlug: taskA.Slug, Detail: ...}` identifying both slugs
  - `checkOrphanTasks(Proposal) []Finding` — build undirected adjacency from `depends_on` (only intra-proposal edges). If no edges exist, return nil. Otherwise emit `Finding{Type: "orphan-task", Severity: "warning", TaskSlug: orphan.Slug, Detail: ...}` for each zero-degree node
  - Helper: `slugMatchesAtWordBoundary(text, slug string) bool` — case-insensitive check that slug appears as a whole word/phrase in text (not as substring of a longer word)
- Modified `internal/service/decompose_validate_test.go`:
  - `TestCheckDependenciesDeclared_NoCrossReferences` — no findings
  - `TestCheckDependenciesDeclared_SlugInSummaryNoDep` — warning finding
  - `TestCheckDependenciesDeclared_SlugInRationaleWithDep` — no findings
  - `TestCheckDependenciesDeclared_ReverseDep` — no findings (B depends on A satisfies A referencing B)
  - `TestCheckDependenciesDeclared_CaseInsensitive` — "Setup-Database" matches slug `setup-database`
  - `TestCheckDependenciesDeclared_WordBoundary` — slug `api` does not match "capital"
  - `TestCheckOrphanTasks_NoDependencies` — check skipped, no findings
  - `TestCheckOrphanTasks_AllConnected` — no findings
  - `TestCheckOrphanTasks_DisconnectedTask` — warning finding identifying orphan
  - `TestCheckOrphanTasks_TwoSeparateChains` — no findings (all participate)
  - `TestCheckOrphanTasks_SingleTask` — no findings
  - `TestCheckOrphanTasks_CrossFeatureDepIgnored` — external dep does not count as edge

**Dependencies:** Task 1 (Finding struct with Severity field, stub functions in place)

---

### Task 4: Single-agent sizing check

**Objective:** Implement the `checkSingleAgentSizing` validation check. This is the most complex heuristic — it detects summaries containing multiple action clauses joined by coordinating separators — and benefits from focused implementation and testing.

**Specification references:** FR-003

**Input context:**
- `internal/service/decompose_validate.go` — stub function from Task 1
- `internal/service/decompose.go` — `Proposal`, `ProposedTask`, `Finding` types
- Specification FR-003 acceptance criteria (action verb list, separator list, verb-at-clause-start matching, substring non-match, noun conjunction non-match)

**Output artifacts:**
- Modified `internal/service/decompose_validate.go`:
  - `checkSingleAgentSizing(Proposal) []Finding` — for each task, split summary on coordinating separators, check whether two or more resulting clauses start with an action verb (word-boundary match at clause start). Emit `Finding{Type: "multi-agent-sizing", Severity: "warning", TaskSlug: task.Slug, Detail: ...}` identifying matched verbs and separator
  - Package-level variable `actionVerbs` — the 25 verbs from FR-003: implement, add, create, refactor, update, fix, remove, delete, migrate, configure, write, build, set up, modify, change, extract, move, rename, convert, integrate, replace, introduce, extend, redesign, rewrite
  - Package-level variable `coordinatingSeparators` — `" and "`, `" as well as "`, `" additionally "`, `" plus "`, `"; "`
- Modified `internal/service/decompose_validate_test.go`:
  - `TestCheckSingleAgentSizing_SingleVerb` — no findings
  - `TestCheckSingleAgentSizing_TwoVerbsWithAnd` — warning finding (implement ... and add)
  - `TestCheckSingleAgentSizing_TwoVerbsWithSemicolon` — warning finding (refactor ... ; migrate)
  - `TestCheckSingleAgentSizing_TwoVerbsWithAsWellAs` — warning finding (build ... as well as create)
  - `TestCheckSingleAgentSizing_NounConjunction` — no finding ("Implement request and response handling")
  - `TestCheckSingleAgentSizing_VerbNotInList` — no finding ("Implement ... and verify" — verify not in list)
  - `TestCheckSingleAgentSizing_SubstringNonMatch` — "irreplaceable" does not match "replace"
  - `TestCheckSingleAgentSizing_SetUpPhrase` — "set up" treated as single verb

**Dependencies:** Task 1 (Finding struct with Severity field, stub functions in place)

---

### Task 5: Integration tests and status determination

**Objective:** Add integration-level tests that exercise the full `ReviewProposal` path with combinations of old and new findings, verifying correct status determination, finding aggregation, and non-interference between check categories. This task validates FR-007 (status rules) and FR-008 (additive checks) end-to-end.

**Specification references:** FR-007, FR-008

**Input context:**
- `internal/service/decompose.go` — `ReviewProposal` method
- `internal/service/decompose_test.go` — existing review tests, `setupDecomposeTest` helper
- All check implementations from Tasks 1–4

**Output artifacts:**
- Modified `internal/service/decompose_test.go`:
  - `TestReviewProposal_EmptyDescription_Fail` — proposal with empty summary → status `"fail"`, `blocking_count >= 1`
  - `TestReviewProposal_WarningsOnly_Warn` — proposal triggering only warning-severity findings (e.g., missing test coverage + orphan task) → status `"warn"`, `blocking_count == 0`
  - `TestReviewProposal_MixedOldAndNew` — proposal triggering both a `gap` finding and an `orphan-task` finding → both present in output, status `"fail"`
  - `TestReviewProposal_AllChecksClear_Pass` — well-formed proposal with descriptions, deps, testing keyword, single-verb summaries → status `"pass"`, no findings from new checks
  - `TestReviewProposal_ErrorAndWarningCombined` — proposal with `empty-description` (error) and `missing-test-coverage` (warning) → `blocking_count` counts only error findings
  - Verify `TotalFindings` counts all findings regardless of severity
  - Verify every finding in output has a non-empty `Severity` field

**Dependencies:** Tasks 1, 2, 3, 4 (all checks must be implemented)

---

## Dependency graph

```
            ┌──────────┐
            │  Task 1   │
            │  Finding  │
            │  severity │
            │  + wiring │
            └────┬──────┘
                 │
       ┌─────────┼─────────┐
       │         │         │
       ▼         ▼         ▼
  ┌─────────┐ ┌─────────┐ ┌─────────┐
  │ Task 2  │ │ Task 3  │ │ Task 4  │
  │ desc +  │ │ deps +  │ │ sizing  │
  │ testing │ │ orphans │ │         │
  └────┬────┘ └────┬────┘ └────┬────┘
       │           │           │
       └───────────┼───────────┘
                   │
              ┌────▼────┐
              │ Task 5  │
              │ integr. │
              │  tests  │
              └─────────┘
```

**Execution order:**
- **Phase 1:** Task 1 (serial — establishes shared types and wiring)
- **Phase 2:** Tasks 2, 3, 4 (parallel — independent check implementations)
- **Phase 3:** Task 5 (serial — requires all checks to be in place)

---

## Traceability matrix

| Requirement | Task(s) | Verification |
|---|---|---|
| FR-001 (description present) | Task 2 | Unit tests in `decompose_validate_test.go` |
| FR-002 (dependencies declared) | Task 3 | Unit tests in `decompose_validate_test.go` |
| FR-003 (single-agent sizing) | Task 4 | Unit tests in `decompose_validate_test.go` |
| FR-004 (testing coverage) | Task 2 | Unit tests in `decompose_validate_test.go` |
| FR-005 (no orphan tasks) | Task 3 | Unit tests in `decompose_validate_test.go` |
| FR-006 (finding format) | Task 1 | Unit tests for severity field on all finding types |
| FR-007 (status determination) | Task 1, Task 5 | Integration tests for pass/warn/fail logic |
| FR-008 (additive checks) | Task 1, Task 5 | Integration tests mixing old and new findings |
| FR-009 (existing severity mapping) | Task 1 | Unit tests for gap/cycle/oversized/ambiguous severity |
| NFR-001 (O(n²) performance) | Tasks 2, 3, 4 | Code review during implementation |
| NFR-002 (determinism) | Tasks 2, 3, 4 | Same-input tests, no randomness in implementation |
| NFR-003 (no external state) | Tasks 2, 3, 4 | Signature enforces `func(Proposal) []Finding` — no service deps |
| NFR-004 (interface stability) | Task 1 | No tool parameter changes; output-only addition |
| NFR-005 (test coverage) | Tasks 2, 3, 4, 5 | Dedicated test cases for each check |

---

## Files affected (summary)

| File | Action | Tasks |
|---|---|---|
| `internal/service/decompose.go` | Modify | Task 1 |
| `internal/service/decompose_validate.go` | Create | Tasks 1, 2, 3, 4 |
| `internal/service/decompose_validate_test.go` | Create | Tasks 2, 3, 4 |
| `internal/service/decompose_test.go` | Modify | Tasks 1, 5 |