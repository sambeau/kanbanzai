# Retrospective Recommendations Report

| Field | Value |
|-------|-------|
| Scope | project |
| Signals analysed | 16 retrospective + 54 knowledge entries |
| Period | 2026-03-28 to 2026-04-20 |
| Author | Retrospective synthesis |

---

## Executive Summary

This report consolidates all retrospective signals and knowledge base entries accumulated since P7. Three root causes account for the majority of friction: **workflow state is not committed promptly**, **agents have multiple escape hatches that bypass the tool system**, and **the `decompose` tool cannot read the project's own acceptance-criteria format**. Fixing these three things would eliminate the most frequently recurring problems. A secondary cluster of issues relates to agent-driven quality discipline — flaky tests being dismissed, standalone bugs being invisible in status, and sub-agents having the ability to destroy uncommitted state.

The report is organised into four tiers: **Fix Now** (bugs or gaps causing active harm), **Improve Soon** (friction that compounds over time), **Reinforce** (policies that need enforcement, not code), and **Keep Doing** (patterns that should be preserved and promoted).

---

## Tier 1 — Fix Now

These are defects or gaps that are actively causing incorrect or destructive behaviour.

---

### REC-01 · `docint` does not classify the `**AC-NN.**` acceptance criteria pattern

**Category:** tool-gap  
**Source:** KE-01KMT5TC2FB1G, KE-01KMT4T26W1VZ, KE-01KMT4T59DB2N  
**Impact:** Every feature decomposition in this project

`internal/docint` does not recognise the `**AC-NN.**` pattern used in every specification in this repo as acceptance criteria. As a result, `decompose propose` finds no ACs and silently falls back to generating one task per section heading — output that looks plausible (`Implement 1. Purpose`, `Implement 3. Scope`) but is functionally useless. Tasks have been created manually for every feature decomposition. This is not a workaround; it is the only path that works.

The fallback output also lacks any prominent warning at the task level. The single warning is buried in the proposal object. An agent reviewing the output quickly will miss it.

**Recommendations:**

1. Teach `internal/docint/classifier.go` (or the Layer 2 extractor) to recognise the `**AC-NN.**` and `- [ ] **AC-NN.**` patterns and tag the containing section as `acceptance-criteria` role.
2. Update `decompose propose` to refuse to generate a proposal (hard stop with a clear error message) when the spec is indexed but no acceptance criteria fragments are found, rather than falling back to section headers.
3. If a fallback must exist, mark every generated task as `derived-from-heading` explicitly at the task level, not just in a top-level warning.

---

### REC-02 · Git stash destroys uncommitted MCP state; AGENTS.md instructs agents to stash

**Category:** workflow-friction / tool-gap  
**Source:** KE-01KMT93GM5MGF, KE-01KMT93WRC43V  
**Impact:** Permanent, unrecoverable loss of entity transitions and doc approvals

In P7, a sub-agent followed the "stash uncommitted changes before starting new work" instruction in AGENTS.md and stashed `.kbz/state/` changes written by MCP tool calls in the main agent's session. Feature statuses reverted from `developing` to `specifying`, spec approvals were lost, and task statuses reset. The stash was never popped. The main agent misdiagnosed the cause as a stale binary for several minutes.

The instruction is correct in intent but catastrophically wrong in effect when `.kbz/state/` is tracked in git and MCP tools write to it without committing.

**Recommendations:**

1. **Remove** the "stash" instruction from AGENTS.md and the `kanbanzai-getting-started` skill. Replace it with: "Commit any `.kbz/state/` or `.kbz/index/` changes before starting new work."
2. Add an explicit rule: **never `git stash` in a Kanbanzai project without first verifying there are no `.kbz/` changes in the working tree.**
3. Consider whether the MCP server should auto-commit `.kbz/state/` changes after every mutating tool call (auto-commit is already planned — see P18 context). Until it does, the manual commit requirement must be prominent.
4. Update the `kanbanzai-agents` skill to note: "Before spawning any sub-agent that may run git operations, commit all `.kbz/` changes."

---

### REC-03 · `doc approve` does not update the `Status` field in the document file

**Category:** tool-gap  
**Source:** KE-01KMT5T79D9Q1  
**Impact:** Every spec and design approved via `doc approve`

Approving a document via `doc(action: "approve")` updates the record in `.kbz/state/documents/` but leaves the `Status: Draft` line in the Markdown file unchanged. The system record says `approved`; the file says `Draft`. Human readers — and agents reading the file directly — will see the wrong status.

**Recommendations:**

1. `doc approve` should patch the `Status:` field in the document's front matter or bullet-list header as a side effect of approval. If the file does not contain a Status field, it should add one.
2. As an interim measure, add a step to AGENTS.md: "After `doc(action: \"approve\")`, update the `Status:` field in the document file to `approved`."

---

### REC-04 · Standalone bugs are invisible in `status` health outputs

**Category:** spec-gap  
**Source:** KE-01KN87PQD7S3K  
**Impact:** Bugs filed against general code areas are permanently invisible

P19's attention-item system (C4) only surfaces bugs linked to an in-flight feature. A high-severity bug filed against a general code area (a race condition, a build issue, a tool defect) with no feature attachment never appears in any `status` call, `health` call, or attention-item list.

**Recommendations:**

1. Extend the attention-item system to surface open `high` and `critical` bugs regardless of whether they are linked to a feature.
2. Review the P19 C4 acceptance criterion — this appears to be a wording oversight, not a deliberate design decision. Fix the spec and the implementation together.

---

## Tier 2 — Improve Soon

These issues cause recurring friction but are not immediately destructive. Left unaddressed, they compound.

---

### REC-05 · Workflow state commits are not happening; the checklist that prompts them is gated behind task-claiming

**Category:** workflow-friction  
**Source:** KE-01KN5ZXK486TM  
**Impact:** Every management session that does not claim a task

The `kanbanzai-agents` skill's commit checklist is only surfaced when an agent calls `next(id: "TASK-...")`. Sessions that do management work — doc registration, entity transitions, knowledge contributions, plan reviews — without ever claiming a task never see the checklist. `.kbz/state/` changes accumulate silently and are only caught at the start of the next session via the `git status` pre-task check.

**Recommendations:**

1. Add a standalone "commit workflow state" section to AGENTS.md that is visible to every agent on first read, not only within the task-claiming flow.
2. Add a rule to the `kanbanzai-getting-started` skill: "After any tool call that writes to `.kbz/state/` or `.kbz/index/`, commit before switching concerns."
3. Longer term: make the MCP server auto-commit `.kbz/` changes at transaction boundaries (this is the P18 automation pillar). Until then, the manual prompt must be prominent.

---

### REC-06 · `edit_file` writes to the main repo root, not to worktrees; agents are repeatedly caught by this

**Category:** tool-friction  
**Source:** KE-01KN5CXMBWSXE, KE-01KNBQ9KYG6XS  
**Impact:** Any agent working inside a worktree

`edit_file` and `create_directory` only operate on the registered project root. When an agent is working in a worktree (`.worktrees/FEAT-.../`), any `edit_file` call silently modifies the main repo instead of the worktree. Two separate knowledge entries were filed for this; it caught agents in multiple plans.

**Recommendations:**

1. Add a prominent rule to the `implement-task` skill and the worktree section of AGENTS.md: "`edit_file` does not work in worktrees. Use `terminal` with `python3 -c` or `python3` stdin piping for all file writes when working in a worktree."
2. Investigate whether the `edit_file` tool can be made worktree-aware — if the agent's active worktree path is known, file operations could be redirected automatically.
3. Document the reliable pattern that emerged: read file with `terminal cat`, construct replacement string in Python, write with `python3 -c "open(...).write(...)"`. This pattern should be the standard example in the worktree skill.

---

### REC-07 · Agents bypass the MCP tool system and query `.kbz/state/` via shell commands

**Category:** workflow-friction  
**Source:** KE-01KMT93QAEMDS, KE-01KMT9J3YKJCB  
**Impact:** Workflow integrity; retrospective and report quality

In P7, the orchestrating agent used `cat`, `grep`, and bash for-loops to read `.kbz/state/` YAML files instead of `entity`, `status`, and `doc` tools. The same pattern recurred in document authoring — agents wrote retrospectives from in-session memory rather than calling `retro synthesise` and `knowledge list`, missing cross-plan learning.

This is not a one-time failure; two separate knowledge entries were filed across different plans.

**Recommendations:**

1. Strengthen the anti-pattern language in AGENTS.md: "Do **not** read `.kbz/state/` or `.kbz/index/` files directly. Use the MCP tools. Shell reads of state files bypass lifecycle enforcement, miss derived state, and do not count usage."
2. Add a specific entry to the `kanbanzai-getting-started` skill anti-patterns: "Shell-querying state files — BECAUSE: the MCP tools provide derived and validated state that raw YAML does not. DETECT: any `cat .kbz/state/...` or `grep .kbz/` in a terminal command."
3. Add a specific anti-pattern to the `documenter` role and `update-docs` skill: "Writing reports or retrospectives from session memory alone — BECAUSE: the `retro` and `knowledge` tools contain accumulated project learning across sessions. RESOLVE: always call `retro synthesise` and `knowledge list` before composing any retrospective or review document."

---

### REC-08 · Flaky tests are being dismissed as "pre-existing" without filing bugs

**Category:** workflow-friction  
**Source:** KE-01KN87E3343KP, KE-01KN87ENDQRYX  
**Impact:** Test suite integrity; silent regression risk

The observed agent behaviour pattern is: observe a flaky or race-detected test failure → classify it as "pre-existing" → proceed as if tests are green. Two knowledge entries were filed explicitly naming this as an anti-pattern. A flaky test that was once passing is a real signal of a race condition or timing problem, and "pre-existing" is not a reason to ignore it — it makes it more urgent.

**Recommendations:**

1. Add to the `implement-task` skill checklist: "If any test fails intermittently or with a race detector: (a) do not mark the task done, (b) file a `BUG` entity with the failure details, (c) include the BUG ID in the task completion summary."
2. Add to AGENTS.md: "A test that fails intermittently is not passing. Do not use `--count=1` or `-timeout` flags to mask race conditions. Use `-race` to confirm."
3. Consider adding a CI policy: the race detector flag (`-race`) should be standard on all `go test ./...` runs in the project.

---

### REC-09 · GitHub PR creation is unnecessary overhead in agentic workflows

**Category:** workflow-friction  
**Source:** KE-01KN87VR66NDS  
**Impact:** Every merge cycle with AI-only reviewers

The `pr(action: "create")` step in the merge workflow was designed for human review via GitHub. In an agentic workflow where all reviewers are AI agents working directly on the worktree, creating a GitHub PR adds latency and ceremony with no benefit.

**Recommendations:**

1. Make the PR creation step explicitly optional in the `kanbanzai-workflow` skill: "Skip `pr(action: \"create\")` when all reviewers are AI agents working on the worktree."
2. Add a configuration flag (e.g. `require_github_pr: false` in `.kbz/local.yaml` or project config) to opt in or out of GitHub PR creation at the project level.
3. Document the two-track merge path: (a) AI-only review → skip PR, merge directly after review passes; (b) human review required → create PR, wait for approval.

---

## Tier 3 — Reinforce

These are not code or tooling changes but policy and documentation gaps. The right fix is clarifying the rules so agents (and humans) encounter them reliably.

---

### REC-10 · The `server_info` tool exists but agents don't know to use it when debugging tool failures

**Category:** tool-friction  
**Source:** KE-01KMS0EE96969, KE-01KMS0EE97M2P  

The stale-binary problem (where tool calls appear to fail because the running server was built before the latest code change) wasted significant time. The `server_info` tool was added to address this, but agents don't reflexively reach for it when tool behaviour is unexpected.

**Recommendation:** Add to AGENTS.md and the `kanbanzai-getting-started` skill: "If tool calls return unexpected errors or unknown states, run `server_info` first. A stale binary is the most common cause of mysterious tool failures."

---

### REC-11 · Partial-task-completion dependency patterns need design guidance

**Category:** decomposition-issue  
**Source:** KE-01KN5R87D2TCE  

A circular dependency arose in P9 because the plan assumed partial completion of one task before another could start — a pattern the entity model does not support. The entity model supports `depends_on` (full completion required) but not "depends on phase A of task X".

**Recommendation:** Add a note to the `decompose-feature` skill: "Do not create dependencies that assume partial task completion. If two tasks need to interleave, split the first task into two sequential tasks (phase A and phase B) and make the dependency explicit."

---

### REC-12 · Heredoc syntax fails in the terminal tool; agents keep trying it

**Category:** tool-friction  
**Source:** KE-01KN5TFZZB7B9  

**Recommendation:** Add to the terminal usage guidance in AGENTS.md: "Do not use heredoc (`<<EOF`) syntax in terminal commands — it fails consistently in the sh shell. Use `python3 -c` with escaped strings, or `echo` with single quotes for short content."

---

## Tier 4 — Keep Doing

These patterns emerged as reliably effective across multiple plans and should be preserved, promoted, and referenced in skill documentation.

---

### KEEP-01 · Sub-agent parallelism with explicit file ownership tables

Consistently produced zero merge conflicts across P9, P16, P17, and P22. The key is not just assigning disjoint files — it is writing an explicit ownership table in the dev plan so agents can verify before they start. This pattern should be the default recommendation in `decompose-feature`.

### KEEP-02 · Editorial pipeline for documentation (Write → Edit → Check → Style → Copyedit)

The five-stage pipeline used in P22 produced focused, non-overlapping changes at each stage. The Check stage in particular caught 3–5 genuine factual errors per document that earlier stages missed — errors that would have been published without it. The pipeline should be the default approach for any documentation feature.

### KEEP-03 · Dedicated fact-checking sub-agents for technical documents

Fact-checking sub-agents (Check stage) found real errors: wrong tool counts, incorrect struct fields, stale lifecycle descriptions, wrong conflict risk levels. This is not redundant review — earlier stages (Write, Edit) do not catch factual errors reliably. Always include a Check stage for technical documentation.

### KEEP-04 · `entity(advance: true)` for walking features through lifecycle stages

The `advance: true` flag handles prerequisite checking at each gate automatically. It is faster and safer than manually calling `entity(action: "transition")` at each stage. Should be the default recommendation in the workflow skill for any stage-gate progression.

### KEEP-05 · Committing document intelligence index state as part of feature branches

Several features benefited from the worktree carrying a committed `.kbz/index/` state. This makes the index reproducible and avoids "spec not indexed" errors during decomposition in a fresh session.

---

## Summary Table

| ID | Category | Effort | Priority |
|----|----------|--------|----------|
| REC-01 | Fix `docint` AC pattern recognition + `decompose` hard stop | Medium | 🔴 High |
| REC-02 | Remove stash instruction; add commit-before-spawn rule | Low | 🔴 High |
| REC-03 | `doc approve` patches Status in file | Low | 🔴 High |
| REC-04 | Surface standalone bugs in `status` health | Medium | 🔴 High |
| REC-05 | Workflow state commit prompt visible outside task-claiming | Low | 🟡 Medium |
| REC-06 | Document `edit_file` worktree limitation; add pattern to skill | Low | 🟡 Medium |
| REC-07 | Anti-pattern guidance for shell-querying state files | Low | 🟡 Medium |
| REC-08 | Flaky test policy in skill checklist + AGENTS.md | Low | 🟡 Medium |
| REC-09 | Make GitHub PR creation optional; add config flag | Medium | 🟡 Medium |
| REC-10 | Add `server_info` debugging prompt to AGENTS.md | Low | 🟢 Low |
| REC-11 | Partial-completion dependency guidance in decompose skill | Low | 🟢 Low |
| REC-12 | Heredoc warning in AGENTS.md terminal guidance | Low | 🟢 Low |

---

## Suggested Next Steps

The highest-leverage sequence is:

1. **REC-01** — Fix the AC recognition gap in `docint` and the `decompose` fallback. This is a code change that unblocks every future decomposition.
2. **REC-02 + REC-05** — Update AGENTS.md and the two skills (`kanbanzai-getting-started`, `kanbanzai-agents`) with the commit discipline rules. This is documentation-only and can be done in one session.
3. **REC-03** — Add file patching to `doc approve`. Small code change with immediate visible benefit.
4. **REC-04** — Extend the attention-item system for standalone bugs. Requires a spec review of P19 C4 first.
5. **REC-06 through REC-08** — Bundle the remaining documentation and policy changes into a single "workflow hygiene" plan.