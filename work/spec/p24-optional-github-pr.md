# Specification: Optional GitHub PR Creation

**Feature:** FEAT-01KPPG5XMJWT3
**Plan:** P24-retro-recommendations
**Design:** `work/design/p24-optional-github-pr.md`
**Status:** Draft

---

## Problem Statement

This specification implements the design described in
`work/design/p24-optional-github-pr.md`.

The current Kanbanzai merge workflow treats `pr(action: "create")` as a
standard close-out step before every merge, even in AI-agent-only workflows
where no human review occurs and the PR is merged immediately without
comment. The design introduces a `require_github_pr` configuration field in
`.kbz/config.yaml` so that teams can explicitly enforce PR creation before
merge (opt-in, `true`) or skip it entirely (the default, `false`/nil).

**In scope:**

- Adding `RequireGitHubPR *bool` to `MergeConfig` in
  `internal/config/config.go`.
- Adding a `RequiresGitHubPR() bool` helper method on `MergeConfig`.
- Modifying the merge gate check and execute path to enforce the PR gate
  when `require_github_pr: true` and to leave PR status informational-only
  when `false` or unset.
- Updating `.agents/skills/kanbanzai-workflow/SKILL.md` with a two-track PR
  policy note.
- Updating `.agents/skills/kanbanzai-agents/SKILL.md` Feature Completion
  section with a two-track merge procedure.
- Tests covering both `require_github_pr: true` and nil/false configurations.

**Out of scope:**

- Changes to the `pr` tool or GitHub API integration.
- New MCP tool parameters or tool surface changes.
- Auto-creation of PRs.
- Changes to `.kbz/local.yaml` schema.
- Changes to `.kbz/state/` file formats or the database schema.

---

## Requirements

### Functional Requirements

- **REQ-001:** `MergeConfig` in `internal/config/config.go` MUST include a
  new field `RequireGitHubPR *bool` with YAML tag
  `require_github_pr,omitempty`.

- **REQ-002:** `MergeConfig` MUST expose a `RequiresGitHubPR() bool` helper
  method that returns `true` if and only if `RequireGitHubPR` is a non-nil
  pointer to `true`. A nil pointer and a pointer to `false` MUST both cause
  the method to return `false`.

- **REQ-003:** When `RequiresGitHubPR()` returns `false`, `merge(action:
  "check")` MUST NOT fail due to the absence of a GitHub PR. Any available
  PR status MUST be attached to the response as informational data only and
  MUST NOT cause a blocking gate failure.

- **REQ-004:** When `RequiresGitHubPR()` returns `true` and no open GitHub
  PR can be retrieved for the entity branch, `merge(action: "check")` MUST
  return a blocking gate failure. The response MUST contain a `pr_gate` key
  with `status: "failed"` and a message identifying the missing PR as the
  cause.

- **REQ-005:** When `RequiresGitHubPR()` returns `true` and a PR is
  retrieved but its state is not `"open"`, `merge(action: "check")` MUST
  return a blocking gate failure. The response MUST contain a `pr_gate` key
  with `status: "failed"` and a message that includes the actual PR state
  value.

- **REQ-006:** `merge(action: "execute")` MUST be blocked when
  `RequiresGitHubPR()` returns `true` and no open PR exists for the entity
  branch, consistent with the gate failure produced by `merge(action:
  "check")`.

- **REQ-007:** The `.agents/skills/kanbanzai-workflow/SKILL.md` "Before
  completing a feature" section MUST be updated to include a PR policy note
  that describes both tracks: what happens when `require_github_pr: false`
  (default — PR optional, merge proceeds directly) and when
  `require_github_pr: true` (PR required, merge check fails without an open
  PR).

- **REQ-008:** The `.agents/skills/kanbanzai-agents/SKILL.md` Feature
  Completion section MUST be updated to describe both workflow tracks: the
  track that calls `pr(action: "create")` before merge (when
  `require_github_pr: true`) and the track that calls `merge` directly
  (when `require_github_pr: false`, the default).

### Non-Functional Requirements

- **REQ-NF-001:** Existing projects that do not set `require_github_pr` in
  `.kbz/config.yaml` MUST experience no change in merge behaviour. The
  absence of the field MUST be treated identically to
  `require_github_pr: false`.

- **REQ-NF-002:** The `require_github_pr` field MUST reside in
  `.kbz/config.yaml` (project-wide policy). The `.kbz/local.yaml` schema
  MUST NOT be modified.

- **REQ-NF-003:** The external MCP tool interface for the `merge` and `pr`
  tools MUST NOT gain new parameters as a result of this change.

---

## Constraints

- The `pr` tool, GitHub API integration, and `pr(action: "create")` workflow
  must remain unchanged in their existing behaviour.
- No new external Go dependencies may be introduced.
- The `RequireGitHubPR` field uses `*bool` (pointer), following the same
  pattern as the existing `PostMergeInstall *bool` field in `MergeConfig`.
  This distinguishes "not set" from "explicitly false" without requiring a
  migration.
- `require_github_pr` is a project-level policy field committed to the
  repository; it is not a per-machine preference.
- This specification does NOT cover changes to CI triggers, PR templates,
  GitHub branch protection rules, or any other GitHub repository settings.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a `MergeConfig` YAML containing
  `require_github_pr: true`, when the config is unmarshalled, then
  `RequireGitHubPR` is a non-nil pointer whose dereferenced value is `true`.

- **AC-002 (REQ-001):** Given a `MergeConfig` YAML that omits the
  `require_github_pr` key, when the config is unmarshalled, then
  `RequireGitHubPR` is `nil`.

- **AC-003 (REQ-002):** Given `RequireGitHubPR` is `nil`, when
  `RequiresGitHubPR()` is called, then it returns `false`.

- **AC-004 (REQ-002):** Given `RequireGitHubPR` is a pointer to `false`,
  when `RequiresGitHubPR()` is called, then it returns `false`.

- **AC-005 (REQ-002):** Given `RequireGitHubPR` is a pointer to `true`,
  when `RequiresGitHubPR()` is called, then it returns `true`.

- **AC-006 (REQ-003):** Given `require_github_pr` is unset (nil) and no
  GitHub token is configured, when `merge(action: "check")` is called on a
  feature with all other gates passing, then the response does not contain a
  `pr_gate` key and the overall result is not a PR-related gate failure.

- **AC-007 (REQ-004):** Given `require_github_pr: true` and a GitHub token
  is configured, when `merge(action: "check")` is called and no PR is found
  for the entity branch, then the response contains a `pr_gate` key with
  `status == "failed"`.

- **AC-008 (REQ-005):** Given `require_github_pr: true` and a GitHub token
  is configured, when `merge(action: "check")` is called and a PR is found
  whose state is not `"open"`, then the response contains a `pr_gate` key
  with `status == "failed"` and the message includes the actual state string.

- **AC-009 (REQ-006):** Given `require_github_pr: true` and no open PR
  exists for the entity branch, when `merge(action: "execute")` is called,
  then the merge is blocked and the error identifies the missing open PR as
  the cause.

- **AC-010 (REQ-NF-001):** Given an existing `.kbz/config.yaml` that does
  not contain `require_github_pr`, when the updated code is used, then merge
  check and execute behaviour are identical to pre-change behaviour (no new
  PR gate failures introduced).

- **AC-011 (REQ-007, REQ-008):** Given the updated skill files are read,
  when an agent follows the "Before completing a feature" checklist or the
  Feature Completion procedure, then the agent can determine whether to call
  `pr(action: "create")` by inspecting the `merge.require_github_pr` setting
  in `.kbz/config.yaml`.

---

## Verification Plan

| Criterion | Method      | Description                                                                                     |
|-----------|-------------|-------------------------------------------------------------------------------------------------|
| AC-001    | Test        | YAML unmarshal test: `require_github_pr: true` → non-nil pointer to `true`                     |
| AC-002    | Test        | YAML unmarshal test: absent field → `nil` pointer                                               |
| AC-003    | Test        | Unit test: `RequiresGitHubPR()` returns `false` when pointer is nil                            |
| AC-004    | Test        | Unit test: `RequiresGitHubPR()` returns `false` when pointer is `false`                        |
| AC-005    | Test        | Unit test: `RequiresGitHubPR()` returns `true` when pointer is `true`                          |
| AC-006    | Test        | Integration test: merge check with nil config and no GitHub token passes without PR gate        |
| AC-007    | Test        | Integration test: `require_github_pr: true`, no PR → `pr_gate.status == "failed"`              |
| AC-008    | Test        | Integration test: `require_github_pr: true`, non-open PR → `pr_gate.status == "failed"` with state in message |
| AC-009    | Test        | Integration test: `require_github_pr: true`, no open PR → execute blocked with PR error        |
| AC-010    | Test        | Regression test: config without field behaves as before (no new gate failures)                  |
| AC-011    | Inspection  | Review updated skill files for presence of two-track PR policy note and procedure               |