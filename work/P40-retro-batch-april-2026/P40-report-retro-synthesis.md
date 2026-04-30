# Retrospective: Project-Wide Synthesis — April 2026

| Field | Value |
|-------|-------|
| Scope | project |
| Total Signals | 25 |
| Period | 2026-03-28T01:20:27Z to 2026-04-27T15:58:32Z |

---

## Themes

### 1. tool-friction: edit_file and create_directory tools write to the main repo ...

**Category:** tool-friction | **Signals:** 1 | **Severity Score:** 5

> edit_file and create_directory tools write to the main repo root, not to the worktree. All file operations for worktree-based development must go through terminal commands (python3 -c) instead.

**Suggestion:** Document that worktree development requires terminal-based file writes; edit_file only works with the registered project root

Signals: KE-01KN5CXMBWSXE

### 2. workflow-friction: Commit discipline is not being followed for workflow/managem...

**Category:** workflow-friction | **Signals:** 1 | **Severity Score:** 5

> Commit discipline is not being followed for workflow/management actions (doc registration, feature transitions, state file changes). The kanbanzai-agents skill's commit checklist is only visible when a task is claimed via next(id:...) — sessions that do management work without claiming a task never see it. State changes accumulate silently across a session and are only caught at the start of the next session via the git status pre-task check.

**Suggestion:** Add an explicit 'commit workflow state changes' step to the kanbanzai-agents SKILL.md checklist that covers non-code changes (doc registration, entity transitions, knowledge contributions). Consider adding a note that any tool call writing to .kbz/state/ or .kbz/index/ should be followed by a commit before moving to a different concern.

Signals: KE-01KN5ZXK486TM

### 3. tool-friction: The stale MCP binary issue wasted significant verification t...

**Category:** tool-friction | **Signals:** 1 | **Severity Score:** 3

> The stale MCP binary issue wasted significant verification time — 3 of 9 ACs appeared to fail when the code was correct. There's no built-in way to query the running server's build timestamp or source version.

**Suggestion:** Add a server_info or version MCP tool that reports build timestamp, git commit SHA, and binary path. This would make stale-binary issues immediately diagnosable.

Signals: KE-01KMS0EE97M2P

### 4. tool-friction: Terminal heredoc failed for multi-line Go code with embedded...

**Category:** tool-friction | **Signals:** 1 | **Severity Score:** 3

> Terminal heredoc failed for multi-line Go code with embedded quotes; write_file with entity_id was the correct tool

**Suggestion:** Update task instructions to allow write_file for worktree file creation

Signals: KE-01KPW39640YG9

### 5. tool-friction: Writing Go source files from a worktree sub-agent is signifi...

**Category:** tool-friction | **Signals:** 1 | **Severity Score:** 3

> Writing Go source files from a worktree sub-agent is significantly complicated by shell/Python/Go triple-escaping. b-string and python3 -c approaches both required multiple debug iterations.

**Suggestion:** Use write_file MCP tool as the primary recommendation in implement-task SKILL.md for worktree file writing.

Signals: KE-01KQ7TKTJ7YVB

### 6. workflow-friction: Shell string escaping in python3 -c with embedded YAML/Go co...

**Category:** workflow-friction | **Signals:** 1 | **Severity Score:** 3

> Shell string escaping in python3 -c with embedded YAML/Go code required multiple workaround attempts; heredoc (cat << EOF) was more reliable

**Suggestion:** Use cat with heredoc for writing complete Go source files rather than python3 -c with embedded string literals

Signals: KE-01KPPYEPA1XHZ

### 7. workflow-friction: Working directly on main instead of feature worktrees. Workt...

**Category:** workflow-friction | **Signals:** 1 | **Severity Score:** 3

> Working directly on main instead of feature worktrees. Worktrees were set up but not used because the workflow had already drifted. Implementation is clean but the feature branches now need to be reconciled or abandoned.

**Suggestion:** Add a check in the pre-task checklist: if a worktree exists for the active feature, verify you are inside it before making any code changes.

Signals: KE-01KPYJ4TZ88AE

### 8. decomposition-issue: Task 4 (P3 descriptions) and Task 6 (P2+P3 testing) had a ci...

**Category:** decomposition-issue | **Signals:** 1 | **Severity Score:** 1

> Task 4 (P3 descriptions) and Task 6 (P2+P3 testing) had a circular dependency that required manual intervention. The plan's dependency graph assumed partial task completion (P2 testing before P3 descriptions), which the entity model doesn't support.

Signals: KE-01KN5R87D2TCE

### 9. spec-ambiguity: AC-07 describes 'needs-review task blocks auto-advance' but ...

**Category:** spec-ambiguity | **Signals:** 1 | **Severity Score:** 1

> AC-07 describes 'needs-review task blocks auto-advance' but needs-review is non-terminal so it blocks developing→reviewing before the auto-advance check runs. Integration test reflects actual behavior; unit test covers the checkAllTasksHaveVerification code path directly.

**Suggestion:** Clarify in specs whether AC-07 covers the unit function or end-to-end advance behavior.

Signals: KE-01KPQJ0W5EMBP

### 10. spec-ambiguity: Spec FR-002 listed paired test task fields but omitted Ratio...

**Category:** spec-ambiguity | **Signals:** 1 | **Severity Score:** 1

> Spec FR-002 listed paired test task fields but omitted Rationale, yet the existing invariant test required all tasks to have non-empty Rationale.

**Suggestion:** Spec should include Rationale in the field list for paired test tasks.

Signals: KE-01KQ2JDTJR0PQ

### 11. tool-friction: Heredoc/HEREDOC syntax consistently fails in the terminal to...

**Category:** tool-friction | **Signals:** 1 | **Severity Score:** 1

> Heredoc/HEREDOC syntax consistently fails in the terminal tool's sh shell. Workaround: use python3 or edit_file for file creation, or pipe to python3 stdin.

Signals: KE-01KN5TFZZB7B9

## What Worked Well

### worked-well: The advance: true flag on entity transitions worked perfectl...

**Signals:** 1

> The advance: true flag on entity transitions worked perfectly for walking Feature E through specifying → dev-planning → developing → reviewing, checking document prerequisites at each gate. The spec approval side-effect auto-advanced the first gate.

### worked-well: Using a cross-plan feature (Skills Content from P3) as a sec...

**Signals:** 1

> Using a cross-plan feature (Skills Content from P3) as a second review target worked well — it provided a genuine review with real blocking findings (fabricated lifecycles, stale 1.0 tool names) rather than a synthetic scenario. The missing-spec edge case also created a natural ambiguous-finding candidate.

### worked-well: T1-T4 parallelism plan held exactly — zero file conflicts....

**Signals:** 1

> T1-T4 parallelism plan held exactly — zero file conflicts. The file ownership table in the dev plan made this unambiguous.

### worked-well: Version-aware conflict logic pattern from mcp_config.go tran...

**Signals:** 1

> Version-aware conflict logic pattern from mcp_config.go translated cleanly to Markdown managed-marker pattern. The const/var embedding approach kept the template content visible and auditable in code review.

### worked-well: Sub-agent parallelism for disjoint file edits (description r...

**Signals:** 1

> Sub-agent parallelism for disjoint file edits (description rewrites and error message audits) was extremely effective — 4 sub-agents editing different tool files simultaneously with zero conflicts. The key enabler was clear file ownership boundaries in the task delegation.

### worked-well: Sub-agent parallelism with disjoint file scopes worked perfe...

**Signals:** 1

> Sub-agent parallelism with disjoint file scopes worked perfectly for this feature — 7 tasks with zero merge conflicts. Tasks 4+6 and 3+5 ran in parallel with clean boundaries.

### worked-well: Python-based file editing via terminal was reliable for work...

**Signals:** 1

> Python-based file editing via terminal was reliable for worktree modifications across 8 files. The pattern of reading, replacing, and writing was more predictable than sed for multi-line changes.

### worked-well: Sub-agent delegation per editorial pipeline stage was effect...

**Signals:** 1

> Sub-agent delegation per editorial pipeline stage was effective — each stage produced focused, non-overlapping changes. The Write stage did not need rework from later stages.

### worked-well: Fact-checking sub-agent found 3 genuine inaccuracies (tool c...

**Signals:** 1

> Fact-checking sub-agent found 3 genuine inaccuracies (tool count, bug lifecycle, incident RCA enforcement) that would have been published uncorrected without the Check stage.

### worked-well: The fact-checking sub-agent found 5 genuine errors in the or...

**Signals:** 1

> The fact-checking sub-agent found 5 genuine errors in the orchestration doc including incorrect conflict risk levels and missing struct fields — more errors than the Workflow Overview check stage.

### worked-well: Retrospectives doc required only 1 factual correction (scope...

**Signals:** 1

> Retrospectives doc required only 1 factual correction (scope hardcoded vs inherited) — the source material gathering was thorough enough that the Write stage produced highly accurate content.

### worked-well: Narrowing exclusion checks to Type == 'open_critical_bug' pr...

**Signals:** 1

> Narrowing exclusion checks to Type == 'open_critical_bug' prevented false failures from health_error items that the health-check block legitimately adds for bugs with non-model status values.

### worked-well: Sub-agent parallelism for disjoint file sets (gate.go vs sta...

**Signals:** 1

> Sub-agent parallelism for disjoint file sets (gate.go vs status_tool.go) worked well — no conflicts, both features implemented independently without coordination overhead.

### worked-well: Parallel dispatch for Tasks 1 and 3 (disjoint files) worked ...

**Signals:** 1

> Parallel dispatch for Tasks 1 and 3 (disjoint files) worked perfectly — zero conflicts, both completed in the same round-trip.

