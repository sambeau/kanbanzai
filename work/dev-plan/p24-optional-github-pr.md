# Dev Plan: Optional GitHub PR Creation

**Feature:** FEAT-01KPPG5XMJWT3
**Plan:** P24-retro-recommendations
**Specification:** `work/spec/p24-optional-github-pr.md`
**Status:** Draft

---

## Scope

This plan implements the requirements defined in
`work/spec/p24-optional-github-pr.md` (FEAT-01KPPG5XMJWT3/specification-p24-optional-github-pr).
It covers four tasks: extending the config struct, enforcing the PR gate in
the merge tool, writing tests, and updating workflow skill documentation.

It does not cover changes to the `pr` tool, GitHub API integration, new MCP
tool parameters, `.kbz/local.yaml` schema, `.kbz/state/` file formats, or
CI/branch-protection configuration.

---

## Task Breakdown

### Task 1: Extend `MergeConfig` with `require_github_pr`

- **Description:** Add `RequireGitHubPR *bool` field and `RequiresGitHubPR()
  bool` helper method to `MergeConfig` in `internal/config/config.go`.
  The field follows the same `*bool` pointer pattern as the existing
  `PostMergeInstall` field. The helper returns `true` only when the pointer
  is non-nil and points to `true`; nil and `false` both return `false`.
- **Deliverable:** Modified `internal/config/config.go` with the new field
  and method.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-001, FR-002, NFR-001, NFR-002.

### Task 2: Enforce PR gate in merge tool

- **Description:** Modify `checkMergeReadiness` in
  `internal/mcp/merge_tool.go` to read `cfg.Merge.RequiresGitHubPR()`.
  When `true` and no open PR is found (or PR retrieval fails), inject a
  `pr_gate` key with `status: "failed"` and a descriptive message into the
  response map — producing a blocking gate failure. When `true` and a PR is
  found but its state is not `"open"`, produce the same failure with the
  actual state in the message. When `false` or nil, preserve the current
  behaviour: attach `pr_status` informational data only if a token is
  present and PR status is available.

  Also update the `executeMerge` path (or its gate-check re-evaluation) so
  that `merge(action: "execute")` is blocked under the same conditions,
  consistent with the check result.
- **Deliverable:** Modified `internal/mcp/merge_tool.go`.
- **Depends on:** Task 1.
- **Effort:** Medium.
- **Spec requirements:** FR-003, FR-004, FR-005, FR-006.

### Task 3: Write tests

- **Description:** Add tests covering all config and gate behaviour:
  - Config YAML unmarshal tests (`require_github_pr: true` → non-nil pointer;
    absent field → nil pointer).
  - Unit tests for `RequiresGitHubPR()` covering nil, `false`, and `true`
    pointer inputs.
  - Integration tests for `checkMergeReadiness`:
    - `require_github_pr` nil, no GitHub token → no `pr_gate` failure.
    - `require_github_pr: true`, no PR found → `pr_gate.status == "failed"`.
    - `require_github_pr: true`, PR found but state not `"open"` →
      `pr_gate.status == "failed"` with state in message.
  - Regression test: config without field produces no new gate failures
    compared to pre-change behaviour.
  - Execute-path test: `require_github_pr: true`, no open PR → execute
    blocked.

  Config tests go in `internal/config/config_test.go` (or a new
  `merge_config_test.go` in the same package). Gate behaviour tests go in
  `internal/mcp/merge_tool_test.go`.
- **Deliverable:** New or modified test files in `internal/config/` and
  `internal/mcp/`.
- **Depends on:** Task 2.
- **Effort:** Medium.
- **Spec requirements:** AC-001 through AC-010 (all testable criteria).

### Task 4: Update workflow skill documentation

- **Description:** Apply the two-track PR policy documentation described in
  the design:
  1. `.agents/skills/kanbanzai-workflow/SKILL.md` — add a PR policy note
     after the "Before completing a feature" checklist explaining that
     `require_github_pr: false` (default) makes PR optional and
     `require_github_pr: true` enforces a blocking PR gate.
  2. `.agents/skills/kanbanzai-agents/SKILL.md` — replace the Feature
     Completion section's PR/merge procedure with a two-track description:
     one for `require_github_pr: true` (call `pr(action: "create")` first)
     and one for `require_github_pr: false` / default (call `merge` directly,
     `pr` step is optional).
- **Deliverable:** Modified `.agents/skills/kanbanzai-workflow/SKILL.md` and
  `.agents/skills/kanbanzai-agents/SKILL.md`.
- **Depends on:** None (independent of code changes; can run in parallel with
  Tasks 1 and 2).
- **Effort:** Small.
- **Spec requirements:** FR-007, FR-008, AC-011.

---

## Dependency Graph

```
Task 1: Extend MergeConfig          (no dependencies)
Task 4: Update skill docs           (no dependencies)
Task 2: Enforce PR gate      →  depends on Task 1
Task 3: Write tests          →  depends on Task 2
```

Parallel groups:
- **Group A (start immediately):** Task 1, Task 4
- **Group B (after Task 1):** Task 2
- **Group C (after Task 2):** Task 3

Critical path: **Task 1 → Task 2 → Task 3**

Task 4 is fully independent and can be completed at any point.

---

## Risk Assessment

### Risk: Inline vs. formal gate approach divergence

- **Probability:** Low.
- **Impact:** Low.
- **Mitigation:** Use the inline approach in `merge_tool.go` as the design
  explicitly designates it an acceptable first pass. This avoids needing to
  extend `GateContext` with a GitHub client. The formal `GitHubPRExistsGate`
  approach is deferred unless a second gate consumer emerges.
- **Affected tasks:** Task 2.

### Risk: `getPRStatus` is not easily mockable in tests

- **Probability:** Medium.
- **Impact:** Medium.
- **Mitigation:** For tests asserting gate failure when no PR exists, set no
  GitHub token in the test `localConfig`. This causes the existing code path
  to skip `getPRStatus` entirely — the absence of a token is equivalent to
  "no PR found" for gate purposes when `require_github_pr: true`. For the
  non-open PR case, inspect whether `getPRStatus` can be replaced with a
  function variable for injection; if not, write the test as a direct unit
  test of the response-building logic rather than an end-to-end call.
- **Affected tasks:** Task 3.

### Risk: Skill file section location changes between reading and editing

- **Probability:** Low.
- **Impact:** Low.
- **Mitigation:** Read the current skill files before editing to locate the
  exact section headers. Apply surgical edits rather than full rewrites to
  minimise merge risk.
- **Affected tasks:** Task 4.

---

## Verification Approach

| Acceptance Criterion | Verification Method  | Producing Task |
|----------------------|----------------------|----------------|
| AC-001 (FR-001)      | Unit test            | Task 3         |
| AC-002 (FR-001)      | Unit test            | Task 3         |
| AC-003 (FR-002)      | Unit test            | Task 3         |
| AC-004 (FR-002)      | Unit test            | Task 3         |
| AC-005 (FR-002)      | Unit test            | Task 3         |
| AC-006 (FR-003)      | Integration test     | Task 3         |
| AC-007 (FR-004)      | Integration test     | Task 3         |
| AC-008 (FR-005)      | Integration test     | Task 3         |
| AC-009 (FR-006)      | Integration test     | Task 3         |
| AC-010 (NFR-001)     | Regression test      | Task 3         |
| AC-011 (FR-007,008)  | Manual inspection    | Task 4         |