# Retro Report 14 — P37-F3: kbz move Command Implementation

**Date:** 2026-04-27
**Author:** Claude Sonnet 4.6 (sub-agent, session resumed from handoff)

## Task

Resumed implementation of `kbz move` from a handoff summary. Four of five tasks
were already committed. The session covered: fixing a `--force` parsing bug in
`runMove`, writing 18 acceptance-criteria tests in `move_cmd_test.go`, and
attempting to open a pull request.

---

## What went well

**The handoff summary made context recovery fast.** The previous session left a
precise written summary: current branch, HEAD SHA, files modified, what was
done, what was not, and the exact bug to fix. Resuming from that took only a few
parallel file reads before I had a complete mental model. The handoff document
pattern is genuinely effective at spanning context-window boundaries — more so
than relying on `status` or `entity` alone, which give lifecycle state but not
implementation detail.

**The `status` tool gave an instant plan overview.** Calling `status("P37-file-names-and-actions")` returned a structured dashboard — feature list, task counts, health warnings, attention items — in a single call. When the user asked "what is the status of P37?" later in the session, the answer was immediate and complete. No file-walking required.

**The `finish` tool was clean and well-integrated.** Transitioning a task to
`done` with `files_modified` and `verification` in a single call worked without
friction. The tool correctly linked the completion to the parent feature and
produced useful side-effect metadata.

**The sub-agent produced correct output and committed cleanly.** Despite running
out of context, the sub-agent had already written both the bug fix and all 18
tests and committed before stopping. The commit message followed the correct
`type(scope): description` format. This was the best-case failure mode — work
done, handoff implicit in the commit log.

**`go test` as a verification loop was fast and reliable.** Running the tests
against the real git test helpers (`setupTestGitRepo`, `commitFile`, etc.) gave
high-confidence results — the `git log --follow` assertions in particular are
genuine integration-level checks that wouldn't pass against a mock.

---

## What didn't go well

**The sub-agent exhausted its context window mid-task.** The prompt I sent was
detailed and comprehensive — probably too comprehensive. By loading the
sub-agent with the full spec of 18 tests, all the helper signatures, all the
YAML format details, and the Go patterns to follow, I likely front-loaded too
much. The agent then had little budget left for the actual implementation loop.
A leaner prompt scoped to "fix the bug, write the tests, here are the helpers"
would have served better. This is partly a tool design question and partly a
prompt engineering question.

**The `pr` tool failed silently on missing credentials.** I called
`pr(action: "create", entity_id: ...)` and received a clean error saying GitHub
token is not configured. This is fine as an error message. What was less fine is
that nothing in the workflow signalled this would happen before I reached the
step. The `merge(action: "check")` gate, or even the worktree creation step,
could surface "no GitHub token — PR creation will not be available" as an early
warning. Instead, the credential absence only manifests at the moment you need
it, which is at the end of a multi-step workflow.

**Falling back from `pr` to `create_pull_request` also failed.** Having been
blocked by the MCP-level `pr` tool, I tried the raw `create_pull_request` GitHub
tool and received a 401. So both paths were blocked. The only working path was
`git push` (which succeeded via SSH/stored credentials) and then a manual URL.
The asymmetry — git push works, GitHub API doesn't — points to a credential
configuration gap that isn't surfaced anywhere in the workflow documentation.

**The `TestDisplayID_AC017_ResolutionPerformance` test flakes under parallel load.**
Running `go test ./...` triggered this test alongside the full suite and it
failed with 216ms vs a 100ms bound. Running it in isolation it passes
comfortably (~33ms). This is a pre-existing issue unrelated to the current
feature, but it produces noise in the full-suite signal. A test that can only
pass in isolation is testing the wrong thing.

**`entity(action: "transition")` to `reviewing` requires no prerequisite check.**
I transitioned F3 to `reviewing` without a PR existing, and the tool accepted
it. The attention item "Feature is in reviewing with no registered review report"
appeared in the `status` output afterward, but there was no friction at the
point of transition. A gate here would be more useful than a post-hoc warning.

---

## What to improve

**1. Surface credential gaps at workflow start, not at the end.**
When a worktree is created (or a feature moves to `developing`), the system
could check whether GitHub API credentials are present and emit a warning:
"No GitHub token configured — `pr(create)` will fail; set `github_token` in
`.kbz/local.yaml` or use `git push` + manual PR." One early warning beats a
late-stage failure.

**2. Add a `pr` fallback that prints the GitHub PR creation URL.**
When no token is available, `pr(action: "create")` could still be useful if it
printed the git push command and the `github.com/{owner}/{repo}/compare/{branch}`
URL. The user still has to click, but the workflow step isn't a dead end.

**3. Add a gate or prompt on `transition → reviewing` without a PR.**
Before accepting a `reviewing` transition, the tool could check whether a PR
URL is registered on the entity and, if not, prompt: "No PR registered — proceed
anyway? [y/N]". This would prevent the misleading state where a feature appears
to be under review but no review vehicle exists.

**4. Provide a sub-agent prompt budget guideline.**
The agents skill documentation already describes when to spawn sub-agents, but
there is no guidance on prompt size. A note like "keep sub-agent prompts under
~800 words of instructions; use `handoff(task_id)` to let the tool assemble
context rather than manually copying it" would prevent the over-specification
pattern that burned context in this session.

**5. Fix or reframe the `TestDisplayID_AC017_ResolutionPerformance` bound.**
Either raise the bound to something meaningful under parallel test-suite load
(e.g. 500ms), add a `t.Skip` when `-count > 1` or when detected as running
under `go test ./...`, or move it to a dedicated benchmark. A performance test
that flakes under normal CI conditions provides a false signal in both
directions.
