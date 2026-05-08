# Whole-Project Formal Review: Kanbanzai

Date: 2026-05-08
Scope: Whole repository — Go code, MCP/CLI workflow tooling, embedded init assets, workflow documentation, tests, and project conventions.
Owner: P60-whole-project-remediation-cycle
Purpose: Basis report for a remediation cycle.

## Overall Verdict

**has_critical_findings** — the project is not currently release-ready because the suite does not compile/pass and at least one advertised blocking merge gate is a placeholder that always passes.

## Repository State Note

At the start of the review, `git status --short` showed one modified file: `.kbz/skills/verify-closeout/SKILL.md`.

After review tooling/sub-agent reads and test commands, `git status --short` showed additional modified/untracked `.kbz/index`, `.kbz/state/documents`, and `work/` files. The reviewer did not intentionally edit or create project files during the review and did not revert anything because those changes may be concurrent workflow state or human/agent work.

## Executive Summary

### Release Blockers

1. **The Go test suite does not compile/pass.**
   - `go vet ./...` fails because `internal/service` tests do not compile.
   - `go test -race ./...` fails due `internal/service` compile errors and `internal/kbzinit` test failures.
   - `staticcheck ./...` could not run because `staticcheck` is not installed.
2. **A blocking merge gate is registered but always passes.**
   - `HealthCheckCleanGate` is included in the default merge gates but is explicitly a placeholder returning `GateStatusPassed`.
3. **`kanbanzai init` managed-skill idempotency is broken.**
   - The embedded `orchestrate-development` task skill lacks managed marker/version lines, so a first init can write a file that a second init rejects as unmanaged.
4. **Embedded distribution assets have drifted from project-local workflow assets.**
   - `.agents/skills/kanbanzai-agents/SKILL.md` and `.agents/skills/kanbanzai-getting-started/SKILL.md` differ from their embedded `internal/kbzinit/skills/` counterparts.
   - Project stage bindings include stages/assets not present in embedded init assets.
5. **Several filesystem path boundaries are vulnerable to traversal or symlink escape.**
   - Document paths, MCP file tools, and `kbz move`/`kbz delete` should all use a shared canonical containment resolver.

## Checks Run

### Repository and Workflow Checks

- `git status --short`
- `AGENTS.md` reviewed
- `.kbz/stage-bindings.yaml` reviewed
- Reviewer roles and `audit-codebase` skill reviewed
- Codebase memory graph skill docs reviewed before graph use

### Static and Test Checks

- `go vet ./...` — failed
- `staticcheck ./...` — not run; tool missing
- `go test -race ./...` — failed
- `go test ./internal/service` — failed
- `go test ./internal/kbzinit` — failed

### Graph and Structural Checks

- Index status: ready
- Graph schema inspected
- Architecture overview inspected
- High fan-out query run
- Change-coupling query run
- Dead-code query attempted; no reliable candidates returned by the available degree query path
- Spot checks performed with `grep` and targeted file reads

## Finding Summary

| Severity | Count | Summary |
|---|---:|---|
| Critical | 4 | Test suite failure, placeholder blocking merge gate, init idempotency failure, stale lifecycle/test API drift |
| Major | 10 | Path traversal/symlink boundary issues, embedded asset drift, stage-binding drift, high-complexity lifecycle/merge/server hotspots, background goroutine test leakage |
| Minor | 5 | Staticcheck missing, HTTP timeout absence, action log ignored close errors, marker detection looseness, documentation contradictions |
| Total | 19 | Whole-project audit findings |

## Critical Findings

### C1 — The project test suite does not currently compile or pass

Dimension: test_health / static_analysis
Severity: critical

Evidence:

- `go vet ./...` fails with `internal/service/bug_gate_test.go:92:17: undefined: model.BugStatusVerified`.
- `go test ./internal/service` fails with `undefined: model.BugStatusVerified`, `result1.VerifierTimedOut undefined`, and `result2.VerifierTimedOut undefined`.
- `internal/model/entities.go` defines `BugStatusVerifying`, not `BugStatusVerified`.
- `internal/service/prereq.go` defines `GateResult` with `VerifierPrompt` and `NeedsVerifier`, but no `VerifierTimedOut`.
- `internal/service/prereq.go` routes bug close-out through `needs-review → verifying → closed`.
- `internal/service/bug_gate_test.go` still tests `VerifiedToClosed` and asserts `VerifierTimedOut`.

Impact:

This blocks all meaningful release confidence. Since packages fail at compile time, the suite cannot act as a regression gate, and `go vet` cannot complete.

Recommendation:

- Update `internal/service/bug_gate_test.go` from `BugStatusVerified` to `BugStatusVerifying`.
- Rename `VerifiedToClosed` tests/comments to `VerifyingToClosed`.
- Decide the intended timeout contract:
  - Either add `VerifierTimedOut bool` back to `GateResult`, or
  - Update tests to assert the current contract via `Satisfied`, `NeedsVerifier`, and timeout text in `Reason`.

### C2 — `HealthCheckCleanGate` is registered as a blocking merge gate but always passes

Dimension: structural_quality / workflow_safety
Severity: critical

Evidence:

- `internal/merge/checker.go` includes `HealthCheckCleanGate{}` in `DefaultGates()`.
- `internal/merge/gates.go` states the gate is a placeholder and returns `GateStatusPassed` unconditionally.
- The code comment explicitly says it “provides no protection” and “merges proceed even when blocking health errors exist.”

Impact:

This creates a false safety signal. Users and agents see a blocking `health_check_clean` gate in the merge gate list, but it cannot block unsafe merges. That is especially serious in Kanbanzai because merge gates are the workflow’s last automated quality boundary.

Recommendation:

Either implement entity-scoped health checking and fail on blocking health errors, or remove `HealthCheckCleanGate{}` from `DefaultGates()` until it has real behavior. Do not keep a blocking gate that always passes.

### C3 — `kanbanzai init` task-skill idempotency is broken by missing managed markers

Dimension: workflow_conformance / distribution
Severity: critical

Evidence:

- `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md` starts with YAML frontmatter but lacks `# kanbanzai-managed:` and `# kanbanzai-version:`.
- `internal/kbzinit/task_skills.go` installs embedded task skills into `.kbz/skills/`.
- `internal/kbzinit/skills.go` transforms marker/version lines only if they already exist.
- `go test ./internal/kbzinit` fails multiple idempotency/update tests because generated `.kbz/skills/orchestrate-development/SKILL.md` is later rejected as unmanaged.

Impact:

A fresh `kanbanzai init` can install a task skill that a second `kanbanzai init` or `--update-skills` refuses to manage. That breaks idempotency and makes newly initialized projects brittle.

Recommendation:

- Add managed marker/version lines to the embedded `orchestrate-development` seed.
- Add a direct test that every `taskSkillNames` embedded seed contains exactly one managed marker and one version marker.
- Consider making `transformSkillContent` fail fast if an installable embedded seed lacks required marker lines.

### C4 — Embedded workflow skill seeds have drifted from `.agents` source files

Dimension: workflow_conformance / distribution
Severity: critical-to-major; classified critical because tests fail and new installs receive stale guidance.

Evidence:

`go test ./internal/kbzinit` reports:

- `skill "agents": embedded seed differs from .agents/skills/ counterpart`
- `skill "getting-started": embedded seed differs from .agents/skills/ counterpart`

Relevant files:

- `.agents/skills/kanbanzai-agents/SKILL.md`
- `internal/kbzinit/skills/agents/SKILL.md`
- `.agents/skills/kanbanzai-getting-started/SKILL.md`
- `internal/kbzinit/skills/getting-started/SKILL.md`

Impact:

Newly initialized projects receive stale workflow guidance. The missing guidance is operationally significant: state-commit discipline and stale-server diagnostics affect day-to-day reliability.

Recommendation:

- Sync `.agents/skills/kanbanzai-agents/SKILL.md` to `internal/kbzinit/skills/agents/SKILL.md`.
- Sync `.agents/skills/kanbanzai-getting-started/SKILL.md` to `internal/kbzinit/skills/getting-started/SKILL.md`.
- Extend `AGENTS.md` dual-write guidance to cover `.kbz/skills/<name>/SKILL.md` ↔ `internal/kbzinit/skills/task-execution/<name>/SKILL.md`, because tests enforce that relationship.

## Major Findings

### M1 — Document/report path handling can escape intended repository boundaries

Dimension: security / filesystem safety
Severity: major

Evidence:

- `internal/service/documents.go` trims `input.Path` and joins it with `s.repoRoot`.
- `internal/service/doc_validate.go` validates naming/folder conventions but does not reject absolute paths, `..` traversal, or resolved paths outside the repository.
- `internal/service/documents.go` later reads document content with `filepath.Join(s.repoRoot, result.Path)`.

Impact:

If an MCP caller supplies traversal-style paths, document operations may hash, read, move, delete, or patch files outside the intended document scope.

Recommendation:

Create one shared repository-relative path resolver for document/report paths. Reject absolute paths, clean and normalize, reject paths whose relative form starts with `..`, enforce allowed roots after cleaning, resolve final paths, and verify containment under `repoRoot`.

### M2 — MCP file tools use lexical containment checks that are symlink-unsafe

Dimension: security / filesystem safety
Severity: major

Evidence:

- `internal/mcp/read_file_tool.go` uses `filepath.Abs`, `Clean`, and prefix checks before `os.ReadFile`.
- `internal/mcp/write_file_tool.go` uses the same lexical containment approach before `MkdirAll` and atomic write.
- The checks do not resolve symlinks in existing parent directories before file access.

Impact:

A symlink inside the repository or worktree can point outside the root. A path such as `repo/linkdir/file` can pass lexical prefix checks while file operations follow the symlink outside the intended containment boundary.

Recommendation:

Use symlink-aware containment: `EvalSymlinks` the root, resolve and validate existing parent directories, reject symlink components for write targets or use no-follow/openat-style safeguards where available, and revalidate immediately before final read/write/rename.

### M3 — `kbz move` and `kbz delete` can bypass the `work/` guard with traversal paths

Dimension: security / CLI safety
Severity: major

Evidence:

- `cmd/kbz/delete_cmd.go` checks `strings.HasPrefix(path, "work/")` before `os.Stat` and `git rm`.
- `cmd/kbz/move_cmd.go` checks `strings.HasPrefix(srcPath, "work/")` before `os.Stat` and `git mv`.
- A path like `work/../README.md` satisfies the prefix check while resolving outside `work/`.

Impact:

This bypasses the command’s safety boundary and could delete or move tracked repository files outside `work/`.

Recommendation:

Clean and canonicalize paths before checking policy, reject absolute paths, reject paths that escape the repository or are not under normalized `work/`, and use `git rm -- <path>` and `git mv -- <src> <dst>`.

### M4 — Project stage bindings and embedded init stage bindings appear to have drifted

Dimension: workflow_conformance / distribution
Severity: major

Evidence:

- Project `.kbz/stage-bindings.yaml` includes `merging`, `verifying`, and `retro-fixing`.
- Embedded init stage bindings reportedly do not include those stages.
- `verify-closeout` and `verifier` appear project-local, while the project instructions present verifying as an active workflow stage.

Impact:

Newly initialized Kanbanzai projects may not receive the same lifecycle pipeline this repository uses. This is dangerous if `verifying` and `verify-closeout` are product workflow assets rather than project-local experiments.

Recommendation:

Decide whether `merging`, `verifying`, `retro-fixing`, `verifier`, and `verify-closeout` are product assets. If yes, copy stage-binding changes into embedded init assets, add embedded `verify-closeout` skill, add embedded `verifier` role, and add tests that every embedded stage binding references existing embedded roles/skills. If no, update project-facing instructions to clearly mark these as Kanbanzai-project-local.

### M5 — `AGENTS.md` dual-write rule is incomplete relative to enforced tests

Dimension: workflow_conformance
Severity: major

Evidence:

- `AGENTS.md` says task-execution skills under `.kbz/skills/` are project-local and no dual-write applies.
- `internal/kbzinit/task_skills.go` embeds task-execution skills from `internal/kbzinit/skills/task-execution`.
- `internal/kbzinit/skills_consistency_test.go` enforces embedded task skills match `.kbz/skills`.

Impact:

The documentation tells agents not to dual-write `.kbz/skills`, while tests require embedded task-skill seeds to stay synchronized. This contradiction likely contributed to current drift.

Recommendation:

Update `AGENTS.md` to document `.kbz/skills/<name>/SKILL.md` ↔ `internal/kbzinit/skills/task-execution/<name>/SKILL.md`. Also clarify which marker/version metadata belongs in project-local files versus embedded install seeds.

### M6 — `newServerWithConfig` is a high fan-out bootstrap hotspot

Dimension: structural_quality
Severity: major

Evidence:

Graph query identified `newServerWithConfig` in `internal/mcp/server.go` with 71 outgoing calls. It constructs services, loads bindings, initializes caches, wires action logs/checkpoints/gates, registers tools, and registers health checks.

Impact:

This function is a central merge-conflict and regression hotspot. It couples service construction, binding loading, gate routing, tool registration, and health registration.

Recommendation:

Keep `newServerWithConfig` as a thin orchestrator and extract focused builders such as `buildCoreServices`, `buildContextPipeline`, `buildGateRouter`, `registerCoreTools`, `registerGitTools`, and `registerHealthTools`.

### M7 — `entityTransitionAction` is a lifecycle monolith

Dimension: structural_quality
Severity: major

Evidence:

Graph query identified `entityTransitionAction` in `internal/mcp/entity_tool.go` with 25 outgoing calls. It mixes argument parsing, entity resolution, feature transition validation, batch terminal-state gating, bug lifecycle gating, override/checkpoint handling, mutation, and side effects.

Impact:

Lifecycle invariants are hard to reason about independently. Fixing one entity-kind path risks regressions in another.

Recommendation:

Split responsibilities between an MCP argument/response handler, transition service, gate response adapter, and side-effect dispatcher.

### M8 — `executeMerge` couples too many high-risk phases

Dimension: structural_quality / workflow_safety
Severity: major

Evidence:

Graph query identified `executeMerge` in `internal/mcp/merge_tool.go` with 14 outgoing calls. It combines gate evaluation, entity/worktree/task resolution, PR policy, Git merge operations, branch deletion, worktree updates, lifecycle advancement, and verification-stage behavior.

Impact:

Merge execution is one of the most safety-critical workflows. Coupling all phases in one function makes partial failure semantics hard to audit.

Recommendation:

Extract explicit merge phases: preflight gates, PR policy, Git merge, worktree cleanup scheduling, lifecycle advancement, and verification. Each phase should return structured results.

### M9 — Background access-tracking goroutines leak into test cleanup

Dimension: testing / concurrency
Severity: major

Evidence:

- `internal/service/knowledge.go` spawns goroutines in `Get` and `List` and exposes `Close()` to wait.
- `internal/mcp/next_tool_test.go` creates `KnowledgeService` and `IntelligenceService` but does not register cleanup to close them.
- A sub-review reproduced a temp directory cleanup failure under `-race`, consistent with background goroutines writing during cleanup.

Impact:

This creates flaky tests and weakens confidence in race detector results.

Recommendation:

Add `t.Cleanup(knowledgeSvc.Close)` where tests create `KnowledgeService`, add equivalent cleanup for `IntelligenceService` where needed, and consider synchronous or disabled access tracking for tests.

### M10 — Managed marker/version metadata is inconsistent

Dimension: workflow_conformance / maintainability
Severity: major

Evidence:

Observed patterns include YAML metadata with `kanbanzai-managed` / `version`, comment markers `# kanbanzai-managed:` / `# kanbanzai-version:`, missing markers in at least one installable embedded task skill, and duplicate marker lines in some embedded seeds.

Impact:

Install/update logic updates only comment-style markers, which can leave stale YAML metadata. Missing or duplicate markers make managed/unmanaged behavior difficult to reason about.

Recommendation:

Standardize one marker scheme for installer control. Use comment markers only for install/version control, ensure every embedded installable skill has exactly one managed marker and one version marker, and remove stale YAML version fields from embedded seeds or update the transformer to rewrite them consistently.

## Minor Findings

### m1 — `staticcheck` is not installed

Dimension: static_analysis
Severity: minor

Evidence: `staticcheck ./...` failed with `sh: staticcheck: command not found`.

Impact: The audit could not complete the configured static analysis checklist.

Recommendation: Install `staticcheck` in the development/test environment and add it to CI if it is intended to be mandatory.

### m2 — GitHub HTTP client has no default timeout

Dimension: security / reliability
Severity: minor

Evidence: `internal/github/client.go` constructs `&http.Client{}` without a timeout.

Impact: GitHub API operations can hang indefinitely if the caller context lacks a deadline.

Recommendation: Set a conservative default timeout while preserving `NewClientWithHTTPClient` for tests/custom clients.

### m3 — Action log rotation ignores flush/close errors

Dimension: observability / resource lifecycle
Severity: minor

Evidence: `internal/actionlog/writer.go` ignores errors during rotation but handles them on explicit close.

Impact: Action log data could be silently lost during rotation.

Recommendation: Return or log flush/close errors during rotation.

### m4 — Marker detection scans whole files despite comments claiming stricter placement

Dimension: workflow_conformance
Severity: minor

Evidence: Installer helper functions scan all lines for managed markers, while comments/errors imply markers are in frontmatter or line 1.

Impact: A marker-like line in body content could incorrectly classify a file as managed.

Recommendation: Restrict marker detection to the first YAML frontmatter block for skill files, or update documentation/errors to match the implemented loose behavior. Prefer stricter detection.

### m5 — `AGENTS.md` contains stale or contradictory `.kbz/` documentation

Dimension: documentation conformance
Severity: minor-to-major; classified minor because the store discipline section is correct but the repository map is confusing.

Evidence:

- `AGENTS.md` repository structure labels `.kbz/` as project-local workflow state “not committed”.
- Later store discipline says `.kbz/state/` is versioned and must be committed.

Impact:

Agents may misclassify workflow state as local-only and fail to commit required `.kbz/state` or `.kbz/index` changes.

Recommendation:

Clarify `.kbz/` categories: versioned `.kbz/state`, relevant `.kbz/index`, roles, skills, and stage bindings; local/uncommitted `.kbz/local.yaml` and cache directories.

## Structural Quality Notes

The graph and architecture checks show a mature but increasingly centralized codebase. The most important structural hotspots are:

| Symbol | File | Outgoing calls | Triage |
|---|---|---:|---|
| `newServerWithConfig` | `internal/mcp/server.go` | 71 | Major hotspot |
| `run` | `cmd/kbz/main.go` | 32 | Mostly CLI dispatch; monitor |
| `entityTransitionAction` | `internal/mcp/entity_tool.go` | 25 | Major lifecycle monolith |
| `docTool` | `internal/mcp/doc_tool.go` | 19 | Mostly dispatch/schema; monitor |
| `knowledgeTool` | `internal/mcp/knowledge_tool.go` | 14 | Mostly dispatch/schema; monitor |
| `executeMerge` | `internal/mcp/merge_tool.go` | 14 | Major workflow hotspot |
| `assembleContext` | `internal/mcp/assembly.go` | 13 | Major context-assembly hub |
| `RunHealthCheck` | `internal/health/check.go` | 13 | Aggregator; acceptable if kept declarative |

Not all high fan-out is a defect. Dispatch tables and aggregators can legitimately have high fan-out. The major concerns are where high fan-out combines policy, mutation, side effects, and external operations.

## Security Review Summary

No production hardcoded secrets were found in the reviewed evidence. Test tokens and placeholders appear to be test/documentation values.

Security-relevant positives:

- GitHub tokens appear to be read from local config rather than hardcoded.
- Production command execution generally uses `exec.Command` argument arrays rather than shell interpolation.
- No obvious code-executing deserialization pattern was found.

Security-relevant concerns:

1. Document path containment is insufficient.
2. MCP file tools are symlink-unsafe.
3. `kbz move`/`kbz delete` `work/` guard is traversal-bypassable.
4. GitHub HTTP client lacks a timeout.

## Testing Review Summary

Current test health is not acceptable for release:

- `internal/service` does not compile.
- `internal/kbzinit` has multiple failing tests around init idempotency and embedded skill drift.
- `go test -race ./...` fails.
- Background goroutine cleanup creates race/test isolation risk.
- Several tests still use manual `os.Chdir`; not a current blocker, but future flake risk.

Recommended test stabilization order:

1. Fix `internal/service` compile drift.
2. Fix `orchestrate-development` managed marker seed.
3. Sync embedded `agents` and `getting-started` seeds.
4. Add service `Close()` cleanup in MCP test helpers.
5. Add direct first-run assertions for task-skill marker/version quality.
6. Replace manual `os.Chdir` with `t.Chdir` where practical.

## Recommended Remediation Plan

### Phase 1 — Restore release gate confidence

1. Fix `internal/service` test compile failures.
2. Fix `internal/kbzinit` idempotency failures.
3. Sync embedded skill seeds.
4. Install/run `staticcheck`.
5. Re-run `go vet ./...`, `staticcheck ./...`, `go test ./...`, and `go test -race ./...`.

### Phase 2 — Fix workflow safety gaps

1. Implement or remove `HealthCheckCleanGate`.
2. Clarify and sync stage-binding/init assets for `verifying`, `merging`, `retro-fixing`, `verifier`, and `verify-closeout`.
3. Update `AGENTS.md` dual-write and `.kbz/` versioned-state guidance.

### Phase 3 — Harden path boundaries

1. Add a shared canonical path resolver.
2. Apply it to document service paths.
3. Apply symlink-safe containment to MCP read/write/edit tools.
4. Normalize and guard CLI `kbz move`/`kbz delete` paths before policy checks.

### Phase 4 — Reduce structural hotspots

1. Extract server bootstrap builders from `newServerWithConfig`.
2. Split `entityTransitionAction`.
3. Split `executeMerge` into explicit phases.
4. Split `assembleContext` into pipeline stages.

### Phase 5 — Improve test isolation and operational reliability

1. Ensure `KnowledgeService.Close()` and `IntelligenceService.Close()` are registered in test helpers.
2. Add a default GitHub HTTP timeout.
3. Handle or log action log rotation flush/close errors.
4. Tighten managed marker detection.

## Final Verdict

Kanbanzai has a strong architecture and workflow model, but the current repository state has critical release-blocking drift between tests, lifecycle states, embedded distribution assets, and workflow documentation.

The immediate priority is restoring trust in the release gates:

1. Make the suite compile and pass.
2. Fix init idempotency.
3. Remove or implement the placeholder blocking merge gate.
4. Sync embedded workflow assets with project-local sources.
5. Harden path containment before relying on MCP/CLI file operations as safe boundaries.
