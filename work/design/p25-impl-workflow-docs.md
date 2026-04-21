# Design: Implementation Workflow Documentation Improvements

**Feature:** FEAT-01KPQ08YH16WZ  
**Plan:** P25 — Agent Tooling and Pipeline Quality  
**Author:** architect  
**Status:** draft

---

## Overview

Three documentation gaps in the agent skill files were identified through P24 post-mortems and sprint retrospective analysis. Each gap has a measurable impact on agentic pipeline quality:

**Gap 1 — Heredoc vs `python3 -c` for worktree file writes (`implement-task/SKILL.md`)**

The current skill documents `python3 -c` with triple-quoted strings as the primary pattern for writing files inside Git worktrees. This pattern breaks silently on Go source files containing embedded triple-single-quotes — which appear in doc comments, test string literals, and generated code. The failure mode is a Python syntax error mid-write, leaving a corrupt or zero-byte file with no clear diagnostic. Heredoc (`cat > file << 'GOEOF'`) avoids the Python escaping problem entirely and is the correct primary pattern for Go source files. This gap was directly observed in P24 implementation sessions.

**Gap 2 — Manual `entity` fallback when `decompose propose` fails (`decompose-feature/SKILL.md`)**

`decompose propose` was bypassed by two of five agents in the P24 sprint:
- One feature (hygiene docs) produced a 13-task proposal from 13 acceptance criteria, ignoring the dev plan's intended 4-task file-scoped grouping — the proposal was structurally wrong and could not be corrected by adding context.
- One feature (optional GitHub PR) returned a proposal with empty task names, causing `decompose apply` to raise an error.

Both agents fell back to creating tasks manually via `entity(action: create)` — a valid and correct approach that is entirely undocumented in the skill. Agents had to rediscover the required fields and dependency wiring format from the tool schema, adding friction and creating inconsistency in the resulting task records.

**Gap 3 — One-sub-agent-per-task sizing for large-file features (`orchestrate-development/SKILL.md`)**

When a feature has more than three tasks that involve reading or rewriting large source files (>300 lines), dispatching one sub-agent per feature (rather than one per task) causes rapid context window exhaustion. Full-file content is embedded in terminal tool calls; a single agent accumulates the content of multiple large files in its context before completing its first task. One agent in P24 hit context limits partly due to this pattern. The skill currently documents parallel dispatch and context compaction techniques but provides no sizing guidance for features where per-task isolation is necessary to avoid context saturation.

---

## Goals and Non-Goals

### Goals

- Update `implement-task/SKILL.md` to prefer heredoc over `python3 -c` for Go source file writes in worktrees, and document the known heredoc failure mode (delimiter collision).
- Update `decompose-feature/SKILL.md` to document the manual `entity(action: create)` fallback when `decompose propose` produces a structurally wrong or crashing proposal. Include required fields and a dependency wiring example.
- Update `orchestrate-development/SKILL.md` to add sizing guidance: for features with more than three tasks involving large source files, dispatch one sub-agent per task rather than one per feature.
- All changes document existing, proven workarounds. No new patterns are invented here.

### Non-Goals

- **No code changes.** This feature touches only skill documentation files under `.kbz/skills/`. No MCP server code, no tool signatures, no Go source changes.
- **No new tools.** This design does not propose creating a `write_file` tool or patching `decompose propose` — those are tracked separately (P1-alt and P3/P4 in the proposal).
- **No rewrites.** Each change is a targeted addition or substitution within an existing skill section. The overall structure of each skill file is preserved.
- **No cross-file consolidation.** The three changes address separate skills used at separate workflow stages. They are delivered as independent edits to three files.

---

## Design

### Change 1: `implement-task/SKILL.md` — Worktree File Editing section

**File:** `.kbz/skills/implement-task/SKILL.md`  
**Section:** `## Worktree File Editing`

**Current state:** The section presents `python3 -c` with triple-quoted string content as the primary pattern. Heredoc is mentioned second, as an option for "smaller targeted edits." The checklist item also says "use `terminal` + `python3 -c` for file writes."

**Required change:** Swap the primary and secondary recommendations. Heredoc becomes the primary pattern for Go source files. `python3 -c` is retained as a secondary pattern, scoped to Markdown and YAML files where Python string escaping is not a problem. A new known-failure note is added: if the file content contains a line that is exactly `GOEOF` (the delimiter), the heredoc will silently truncate at that line — the fix is to choose a different delimiter (`GOEOF2`, `ENDOFFILE`, etc.).

**Before (high-level):**
```
Primary: python3 -c with triple-quoted string
Secondary: heredoc for "smaller targeted edits"
Checklist: use python3 -c for file writes
```

**After (high-level):**
```
Primary: heredoc (cat > file << 'GOEOF') for Go source files
  Known-failure note: delimiter collision — if content contains bare GOEOF
  line, use a different delimiter
Secondary: python3 -c for Markdown and YAML files (no Go triple-quote risk)
Checklist: use heredoc for Go source files; python3 -c for Markdown/YAML
```

The section heading and warning box remain unchanged. The change is confined to the two code block examples and the ordering of the recommendations within the section.

---

### Change 2: `decompose-feature/SKILL.md` — Manual fallback when `decompose propose` fails

**File:** `.kbz/skills/decompose-feature/SKILL.md`  
**Section:** `## Procedure` → Phase 2: Generate Initial Decomposition, and Phase 4: Apply the Decomposition

**Current state:** Phase 2 instructs the agent to call `decompose(action: "propose")` and aim for vertical slices. Phase 4 instructs the agent to call `decompose(action: "apply")` once all five validation checks pass. There is no guidance for what to do when the proposal is structurally wrong before validation, or when `apply` would raise an error.

**Required change:** Add a fallback note at the end of Phase 2 and a fallback block in Phase 4.

**Phase 2 addition:** After the five decompose-propose steps, add a note:

> If `decompose propose` returns a proposal with empty task names, wrong task count, or a structure that cannot be corrected by adjusting input context, do not call `decompose apply`. Proceed to the manual fallback in Phase 4.

**Phase 4 addition:** After the existing apply step, add a "Manual Fallback" subsection:

> **Manual fallback — when `decompose propose` is wrong or crashes:**  
> Use `entity(action: "create", type: "task")` directly. Required fields: `name`, `summary`, `parent_feature`. Optional but recommended: `depends_on` (array of TASK-... IDs for dependency wiring). Create tasks in dependency order so IDs are available for `depends_on` references before they are needed. Verify the created tasks with `status(id: "FEAT-xxx")` before proceeding.

A minimal wiring example is included to show the `depends_on` field format. The fallback is presented as the escape hatch, not the default — the existing Phase 2 procedure remains the primary path.

---

### Change 3: `orchestrate-development/SKILL.md` — One-sub-agent-per-task sizing guidance

**File:** `.kbz/skills/orchestrate-development/SKILL.md`  
**Section:** `## Procedure` → Phase 3: Dispatch Sub-Agents

**Current state:** Phase 3 describes how to generate sub-agent prompts via `handoff`, how to pass file scope boundaries, and how to dispatch in parallel. It does not address how to size task batches per sub-agent, or when dispatching one sub-agent per task (rather than per feature-batch) is preferable.

**Required change:** Add a sizing rule before the dispatch steps in Phase 3:

> **Sizing rule — one sub-agent per task for large-file features:**  
> For features with more than three tasks where tasks involve reading or rewriting source files longer than ~300 lines, dispatch one sub-agent per task rather than assigning multiple tasks to a single agent. Full-file rewrites embed entire file content in terminal tool calls; an agent assigned multiple large-file tasks will saturate its context window before completing the second task. Per-task isolation gives each agent a fresh context window sized for one file scope.  
> Features with small files or documentation-only tasks do not require per-task isolation — batch dispatch remains appropriate.

This addition sits between the graph-project verification step (step 0) and the `handoff` invocation step (step 1), as a decision checkpoint that the orchestrator applies before generating prompts. It is framed as a guideline, not an absolute rule, because the threshold (>3 tasks, >300 lines) requires judgment and will vary.

A corresponding anti-pattern entry may optionally be added in the Anti-Patterns section: "Assigning multiple large-file tasks to one sub-agent" — but the primary placement is in Phase 3 where the dispatch decision is made.

---

## Alternatives Considered

### Alternative 1: Consolidate all three changes into a single skill file

All three changes address implementation friction observed in P24. They could be consolidated into a single "worktree and task management" addendum document or aggregated into one skill file.

**Trade-offs:**
- Makes the three changes easier to review as a unit
- However, the three skills are used at different lifecycle stages by different agents: `decompose-feature` is used in `dev-planning`, `orchestrate-development` in the orchestration phase, and `implement-task` inside sub-agents during `developing`. Consolidating them into one file would require agents at each stage to load context that is irrelevant to their current work.
- Cross-skill consolidation also breaks the stage-binding model: each stage binding points to a specific skill file. A new composite file would need new stage bindings or would not be surfaced by the existing routing.

**Verdict:** Rejected. Each change belongs in the skill file that governs the stage where the failure occurs. Targeted edits to three files is lower coupling than a new composite document.

---

### Alternative 2: Fix the underlying tools instead of updating documentation

The root causes of all three gaps are tool limitations:
- Gap 1: `edit_file` is not worktree-aware. Fix: make it worktree-aware (P1) or add `write_file` (P1-alt).
- Gap 2: `decompose propose` produces wrong proposals. Fix: dev-plan-aware grouping (P4) and empty-name bug fix (P3).
- Gap 3: No built-in context-window sizing enforcement in the dispatch toolchain.

If the tools were fixed, the documentation patches would be unnecessary or would become legacy guidance.

**Trade-offs:**
- Tool fixes eliminate the category of failure entirely, for all current and future agents.
- However, P1, P1-alt, P3, and P4 are medium-to-high effort code changes. P1-alt (`write_file` tool) is the most likely to ship first, but it requires MCP server changes, schema updates, and client-side documentation — a non-trivial scope.
- In the interim (which may span multiple sprints), agents using the current skill documentation will continue to hit the same failures with no documented workaround.

**Verdict:** Both approaches are necessary and complementary. The documentation patches are low-effort, zero-risk changes that improve the current sprint's pipeline quality immediately. The tool fixes (tracked separately) eliminate the need for the workarounds long-term. This is not an either/or decision.

---

## Dependencies

### FEAT-01KPQ08Y47522 — `write_file` MCP tool

The heredoc guidance added by Change 1 is a workaround for the absence of a worktree-safe file writing tool. Once `write_file` (P1-alt) ships and agents can write files via a JSON-parameter tool call without any shell escaping, the heredoc pattern becomes unnecessary.

**When FEAT-01KPQ08Y47522 is merged**, the `## Worktree File Editing` section in `implement-task/SKILL.md` should be updated or removed:
- The `write_file` tool becomes the primary recommendation.
- The heredoc and `python3 -c` patterns become legacy fallbacks or are removed entirely.
- The checklist item should be updated to reference `write_file` as the default.

This dependency is directional: Change 1 in this feature does not block FEAT-01KPQ08Y47522, and FEAT-01KPQ08Y47522 does not block this feature. They can be implemented independently. The lifecycle relationship is: this feature ships workaround documentation now; FEAT-01KPQ08Y47522 supersedes that guidance when it lands.

The spec for FEAT-01KPQ08Y47522 should include an explicit follow-on task: update `implement-task/SKILL.md` to replace heredoc guidance with `write_file` usage.

### No other blocking dependencies

Changes 2 and 3 (decompose fallback, per-task sizing) are self-contained documentation additions. They do not depend on any in-flight code changes and do not become legacy when any currently-planned tool fix ships.

---

## Open Questions

1. **Delimiter naming convention for heredocs:** Should the skill standardise on `GOEOF` as the default delimiter for Go files specifically, or use a generic `'EOF'` with a note to change it on collision? A Go-specific delimiter makes the collision case rarer (a file with a bare `GOEOF` line is unusual) and distinguishes the pattern visually. The implementation stage should decide and document consistently.

2. **Anti-pattern entry for over-sized sub-agent dispatch:** Change 3 proposes adding sizing guidance in Phase 3. It is an open question whether a corresponding Anti-Patterns entry should also be added to `orchestrate-development/SKILL.md` for discoverability. This can be decided at the spec stage without changing the design intent.

3. **Dual-write requirement for `.kbz/skills/` changes:** Changes to skill files under `.kbz/skills/` may need to be mirrored in `internal/kbzinit/skills/` (the embedded copy distributed to other projects). This is a separate concern (P9 in the proposal) and is out of scope for this feature. However, whoever implements the three skill edits should be aware of the dual-write requirement and check whether corresponding files exist in `internal/kbzinit/skills/` before closing their tasks.
```

Now I'll register the document: