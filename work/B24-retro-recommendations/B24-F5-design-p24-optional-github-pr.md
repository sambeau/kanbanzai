# Design: Optional GitHub PR Creation

**Feature:** FEAT-01KPPG5XMJWT3  
**Plan:** P24-retro-recommendations  
**Status:** Draft  
**Author:** architect

---

## Overview

In AI-agent-only workflows, creating a GitHub pull request before merging is
unnecessary overhead. The merge workflow currently encourages `pr(action:
"create")` as a standard close-out step, even when no human review is needed
and the PR will be merged immediately without comment.

This design adds a `require_github_pr` configuration field to `MergeConfig`
in `.kbz/config.yaml`. When the field is `false` (the default), the `pr`
step is optional — the merge workflow proceeds without a PR and the merge
gate check does not fail because no PR exists. When `true`, the merge gate
check enforces that a GitHub PR is open before `merge(action: "execute")`
is called.

No existing behaviour changes for projects that have `require_github_pr:
true` set, or that rely on the current workflow documentation.

---

## Goals and Non-Goals

### Goals

- Add `require_github_pr` field to `MergeConfig` in `config.go`.
- Default to `false`: AI-agent workflows skip PR creation without warnings.
- When `true`: merge gate check actively fails if no open PR exists for the
  feature branch.
- Update `kanbanzai-workflow` skill to document the two-track path (PR
  required vs. optional).
- Add a test that sets `require_github_pr: false` and confirms the merge
  gate passes without a PR.

### Non-Goals

- No changes to GitHub API integration or the `pr` tool itself.
- No changes to how PRs are created when they are requested.
- No new MCP tool parameters or tool surface changes.
- No auto-creation of PRs — this is purely a gate enforcement toggle.
- No changes to the `local.yaml` schema (this is a project policy, not a
  per-machine preference).

---

## Design

### 1. Where `require_github_pr` lives

`require_github_pr` belongs in `.kbz/config.yaml` (project-wide), not in
`.kbz/local.yaml` (per-machine).

**Rationale:** Whether a project requires a GitHub PR before merging is a
team-level policy decision that should be consistent across all machines
working on the same project. `local.yaml` holds per-machine secrets and
preferences (GitHub token, user name, graph project name). Putting a policy
field there would mean each developer independently decides whether PRs are
required, which defeats the purpose of a team convention. `config.yaml` is
committed to the repository and applies uniformly.

The field is added to the existing `MergeConfig` struct, which already holds
`PostMergeInstall`. Pattern precedent: `PostMergeInstall` is a `*bool`
(pointer, tri-state nil/true/false) to distinguish "not set" from "explicitly
false." `require_github_pr` follows the same pattern.

#### Change to `internal/config/config.go` — `MergeConfig` struct (L142–146)

Current:

```go
// MergeConfig holds settings for merge operations.
type MergeConfig struct {
	// PostMergeInstall controls whether to automatically rebuild and install
	// the binary after a successful merge. Defaults to true (nil = true).
	PostMergeInstall *bool `yaml:"post_merge_install,omitempty"`
}
```

Replacement:

```go
// MergeConfig holds settings for merge operations.
type MergeConfig struct {
	// PostMergeInstall controls whether to automatically rebuild and install
	// the binary after a successful merge. Defaults to true (nil = true).
	PostMergeInstall *bool `yaml:"post_merge_install,omitempty"`
	// RequireGitHubPR controls whether a GitHub PR must exist before a merge
	// can be executed. When nil or false (the default), the PR step is
	// optional — AI-agent-only workflows can merge without a PR. When true,
	// merge(action: "check") will fail if no open PR exists for the entity
	// branch.
	RequireGitHubPR *bool `yaml:"require_github_pr,omitempty"`
}
```

A helper method on `MergeConfig` (or inline at the call site) resolves the
pointer to a concrete bool:

```go
// RequiresGitHubPR returns true only when RequireGitHubPR is explicitly set
// to true. nil and false both map to false (opt-in, not opt-out).
func (m MergeConfig) RequiresGitHubPR() bool {
	return m.RequireGitHubPR != nil && *m.RequireGitHubPR
}
```

### 2. Default value: `false`

The default is `false` (PR optional). Justification:

- Kanbanzai is primarily designed for AI-agent workflows where a human does
  not sit in a review loop between every feature branch and main. Requiring
  a PR by default would add friction to the dominant use case.
- Opting **in** to PR enforcement is the right model: teams that want GitHub
  PR history set `require_github_pr: true` in their `config.yaml`.
- Backward compatibility: existing projects that have never set the field
  behave exactly as they do today (PR status is informational, not blocking).
- The `*bool` pointer type means `nil` and `false` are both treated as
  "not required," which avoids any migration burden.

### 3. Merge workflow change — where the config is read and how `pr` is skipped

The gate enforcement lives in `internal/mcp/merge_tool.go` inside the
`checkMergeReadiness` function (L168–242). Currently, `getPRStatus` is called
when a GitHub token is present and its result is attached to the response as
`pr_status` — informational only.

The change adds a new branch: when `cfg.Merge.RequiresGitHubPR()` is `true`,
`checkMergeReadiness` promotes the absence (or non-open state) of a PR to a
blocking gate failure.

#### Change to `internal/mcp/merge_tool.go` — `checkMergeReadiness` (L229–242)

The config is loaded via `config.LoadOrDefault()`. The PR status block
currently at the end of `checkMergeReadiness` becomes:

```go
// Check PR status if GitHub is configured.
// When require_github_pr is true, a missing or non-open PR is a blocking
// gate failure. When false (default), PR status is informational only.
cfg := config.LoadOrDefault()
if localConfig != nil && localConfig.GetGitHubToken() != "" {
    prStatus, err := getPRStatus(ctx, repoPath, wt.Branch, localConfig)
    if cfg.Merge.RequiresGitHubPR() {
        if err != nil || prStatus == nil {
            // Surface as a blocking gate failure in the formatted response.
            resp["pr_gate"] = map[string]any{
                "status":  "failed",
                "message": "require_github_pr is true but no PR status could be retrieved",
            }
        } else if state, _ := prStatus["state"].(string); state != "open" {
            resp["pr_gate"] = map[string]any{
                "status":  "failed",
                "message": fmt.Sprintf("require_github_pr is true but PR state is %q, expected \"open\"", state),
            }
        } else {
            resp["pr_status"] = prStatus
        }
    } else {
        if err == nil && prStatus != nil {
            resp["pr_status"] = prStatus
        }
    }
}
```

The `mergeExecuteAction` path already re-runs `checkMergeReadiness` logic via
`merge.CheckGates`. Because the new PR gate failure is surfaced in the check
response (not as a hard `GateContext` gate), `executeMerge` additionally reads
the same config and calls `getPRStatus` with the same logic — returning an
error string if `RequiresGitHubPR()` is true and no open PR exists.

> **Implementation note:** The cleanest implementation threads
> `require_github_pr` through as a formal `merge.Gate` implementation
> (`GitHubPRExistsGate`) in `internal/merge/gates.go`, added to
> `DefaultGates()` only when the config flag is true. This keeps all gate
> logic in one place and makes the execute path consistent with the check
> path. The gate would need a GitHub client injected via `GateContext`. The
> alternative — checking inline in `merge_tool.go` — is simpler but creates
> a parallel path outside the gate system. The formal gate approach is
> preferred; the inline approach is acceptable as a first pass.

### 4. Documentation change — `kanbanzai-workflow` skill

The "Feature Completion" section in `kanbanzai-agents` skill and the "Before
completing a feature" gate checklist in `kanbanzai-workflow` skill both
mention `pr(action: "create")`. These need a two-track description.

#### Change to `.agents/skills/kanbanzai-workflow/SKILL.md` — "Before completing a feature" checklist

Add after the existing checklist:

```
> **PR policy:** Whether a GitHub PR is required before merge depends on the
> project's `merge.require_github_pr` setting in `.kbz/config.yaml`.
>
> - `require_github_pr: false` (default) — PR is optional. AI-agent
>   workflows can call `merge(action: "execute")` directly after all tasks
>   are done. The merge check will not fail for absence of a PR.
> - `require_github_pr: true` — A GitHub PR must be open before
>   `merge(action: "execute")`. Call `pr(action: "create")` first. The
>   merge check will fail with a blocking gate error if no open PR exists.
```

#### Change to `.agents/skills/kanbanzai-agents/SKILL.md` — Feature Completion section

The current text reads:

```
2. **PR and merge** (if a worktree exists): `pr(action: "create",
   entity_id: "FEAT-xxx")`, then `merge(action: "check", entity_id:
   "FEAT-xxx")`, then `merge(action: "execute", entity_id: "FEAT-xxx")`.
   If the tools return `not_applicable` (no worktree), skip these steps.
```

Replace with:

```
2. **PR and merge** (if a worktree exists): Check whether the project
   requires a GitHub PR (`merge.require_github_pr` in `.kbz/config.yaml`).
   - If `require_github_pr: true`: call `pr(action: "create", entity_id:
     "FEAT-xxx")` first, then `merge(action: "check")`, then
     `merge(action: "execute")`.
   - If `require_github_pr: false` (default, AI-agent workflow): call
     `merge(action: "check")` then `merge(action: "execute")` directly.
     The `pr` step is optional.
   If the tools return `not_applicable` (no worktree), skip these steps.
```

### 5. Test: `require_github_pr: false` passes merge check without a PR

Add a test in `internal/mcp/merge_tool_test.go` (or a new
`internal/merge/gates_test.go` case if the gate approach is used):

```
TestMergeCheck_RequireGitHubPR_False_PassesWithoutPR
  Setup:
    - Create a feature entity in "done" status.
    - Create a worktree record with a branch.
    - Set config.Merge.RequireGitHubPR = nil (unset, equivalent to false).
    - No GitHub token configured in localConfig (so getPRStatus is never called).
  Action:
    - Call checkMergeReadiness for the feature.
  Assert:
    - Response overall_status does not contain a PR-related blocking failure.
    - Response does not contain pr_gate key.
    - Response may contain pr_status only if GitHub token is set and PR exists.

TestMergeCheck_RequireGitHubPR_True_FailsWithoutPR
  Setup:
    - Same entity and worktree as above.
    - Set config.Merge.RequireGitHubPR = pointer to true.
    - Inject a getPRStatus mock that returns nil (no PR found).
  Action:
    - Call checkMergeReadiness.
  Assert:
    - Response contains pr_gate with status "failed".
    - merge(action: "execute") would be blocked.
```

---

## Alternatives Considered

### A. Put `require_github_pr` in `local.yaml`

Rejected. PR enforcement is a team/project policy, not a per-machine
preference. Putting it in `local.yaml` means different developers on the same
project can have different merge behaviour, defeating the purpose of a
consistent workflow policy. See §3 above.

### B. Default to `true` (require PR by default)

Rejected. The dominant Kanbanzai use case is AI-agent-only development where
PRs add no value. Defaulting to `true` would break existing agent workflows
and add friction for the majority of users. Teams that want PR history should
explicitly opt in.

### C. Remove the `pr` step from the workflow documentation entirely

Rejected. Some teams (especially those with mixed human/AI workflows) rely on
GitHub PRs for audit trail, code review, and CI triggers. The two-track
approach preserves full PR functionality for teams that want it while making
it genuinely optional for teams that don't.

### D. Add a `--no-pr` flag to `merge(action: "execute")`

Rejected. A per-call flag makes the behaviour inconsistent across invocations.
A project-level config setting ensures that all agents working on the same
project always follow the same policy without needing to pass a flag on every
call.

### E. Implement the gate inline in `merge_tool.go` rather than as a formal `Gate`

Acceptable as a first pass. The inline approach is simpler and avoids needing
to inject a GitHub client into `GateContext`. However, it creates a parallel
enforcement path outside the gate system, making the check/execute split
harder to maintain. Prefer the formal gate approach for long-term consistency.

---

## Dependencies

- `internal/config/config.go` — `MergeConfig` struct and `RequiresGitHubPR()`
  helper.
- `internal/mcp/merge_tool.go` — `checkMergeReadiness` and `executeMerge`
  functions.
- `internal/merge/gates.go` — optional: `GitHubPRExistsGate` (if formal gate
  approach is chosen).
- `internal/merge/gate.go` — `GateContext` extension (if formal gate approach
  injects GitHub client).
- `.agents/skills/kanbanzai-workflow/SKILL.md` — PR policy documentation.
- `.agents/skills/kanbanzai-agents/SKILL.md` — Feature Completion procedure.
- No database schema changes.
- No new external dependencies.
- No changes to `.kbz/state/` file formats.