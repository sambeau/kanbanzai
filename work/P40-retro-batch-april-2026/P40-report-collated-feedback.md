# Collated Feedback Report — April 2026 Retrospective Batch

**Generated:** 2026-04-30
**Sources:** 16 reports across the April 2026 retrospective batch
**Methodology:** Cross-report thematic synthesis — overlapping findings merged, severity calibrated by report count and impact evidence.

---

## 1. Executive Summary

The April 2026 retrospective batch surfaces a mature workflow system that delivers substantial value — parallel sub-agent dispatch, structured spec→dev-plan→implement pipelines, the `health()` diagnostic tool, and the stage-binding/role/skill scaffolding all received consistently positive reports. The system's core architecture is sound. Friction is concentrated not in the workflow model itself but in the **tooling layer**: discoverability gaps, format sensitivity, state-consistency edge cases, and cross-tool parameter inconsistencies.

The most severe recurring theme is the **worktree file-editing experience**. Five separate reports document that `edit_file` writes to the main repo root instead of worktrees, that `write_file` with `entity_id` is undiscoverable, and that the workaround patterns (heredocs, `python3 -c` with triple-escaping) are fragile and error-prone. This is the single largest source of wasted tool calls across the batch and has been a known issue across multiple plan cycles (P37, P38, B36, B38).

Tool reliability and state consistency form the second major cluster. `finish()` returning success while leaving state inconsistent, `edit_file` silently skipping non-matching edits in multi-edit calls, the `decompose` tool's narrow AC format recognition, and the `parent_feature` filter returning all 614 tasks instead of the scoped subset are correctness bugs that erode trust in workflow automation. Each costs only 2–4 extra tool calls per encounter, but the cumulative effect across a multi-feature batch is significant.

The batch also reveals a clear **session-continuity gap**. When a sub-agent's context window is exhausted mid-task, or when an orchestrator session ends without clean `finish()` calls, the system leaves no recovery trail. Uncommitted state, stuck task lifecycles, and orphaned `.kbz/` files accumulate silently and are only discovered at the start of the next session via the `git status` pre-task check. The system is good at managing *active* work but weak at preserving *interrupted* work.

Positively, the system's strengths are concentrated in exactly the areas that differentiate it from ad-hoc development: structured parallelism (zero file conflicts across dozens of parallel sub-agent dispatches), lifecycle gate enforcement (clear, actionable error messages that prevent premature advancement), and the `health()` diagnostic tool (which surfaced problems from branch drift to skill-file truncation that would otherwise have gone unnoticed). These are the system's durable competitive advantages and should be protected as the friction points are addressed.

---

## 2. Thematic Synthesis

### Theme 1: Worktree File Editing Is Broken and Undiscoverable

**Severity:** Critical
**Reports:** 5 (P40-report-b38-implementation-session, P40-report-retro-synthesis, P40-retro-p38-plans-and-batches-implementation, retro-report-1, retro-report-15)
**Signals:** KE-01KN5CXMBWSXE, KE-01KPW39640YG9, KE-01KQ7TKTJ7YVB, KE-01KPPYEPA1XHZ

**Synthesized Description:**

The `edit_file` tool silently writes to the main repository root, not to worktrees — even when development is happening in a feature's isolated worktree. This causes two failure modes: (a) agents unknowingly edit the wrong tree, and (b) agents who understand the limitation resort to fragile workarounds. The `write_file` MCP tool *does* support worktrees via its `entity_id` parameter, but this capability is undiscoverable — it's not documented in the `implement-task` skill, and the tool description doesn't clearly advertise it.

The workaround patterns — `python3 -c` with triple-escaping through shell/Python/Go layers, and heredoc syntax (`cat << 'EOF'`) — both fail unpredictably. Heredocs are consistently rejected by the terminal tool's `sh` shell. Python-based file writes require multiple debug iterations for code containing embedded quotes, braces, or YAML. The B38 implementation report documents resorting to `git checkout --` to recover from a truncated file write caused by shell escaping failures.

**Suggested Fixes:**

1. Make `edit_file` worktree-aware: resolve `entity_id` to the worktree path when one is active, or accept an explicit `entity_id` parameter like `write_file` does.
2. Document `write_file(entity_id: ...)` as the **primary and recommended** pattern for worktree file creation in the `implement-task` skill, replacing all references to `python3 -c` and heredoc workarounds.
3. If `edit_file` cannot be made worktree-aware, add a prominent warning in the tool description: "This tool writes to the main repo root. For worktree development, use `write_file` with `entity_id`."
4. Add `write_file` to the default tool subset for the `developing` stage so sub-agents can discover it without explicit instruction.
5. Fix heredoc support in the terminal tool's `sh` shell, or document that heredocs are unsupported and provide an alternative.

---

### Theme 2: decompose Tool Has Brittle AC Format Recognition

**Severity:** Significant
**Reports:** 4 (P40-report-b38-implementation-session, P40-retro-p38-plans-and-batches-implementation, retro-report-1, retro-report-2)

**Synthesized Description:**

The `decompose(action: propose)` tool only recognizes three acceptance-criteria formats: `**AC-NN.**` (bold with period), `- [ ] **AC-NNN:**` (checklist), and numbered lists. Specifications using heading-based ACs (`### AC-001`), bold-with-parenthetical-references (`**AC-001 (REQ-001):**`), or other common formats are silently rejected with "no acceptance criteria found." The error message gives no indication of *why* parsing failed or what format *is* expected.

This caused 4 rounds of spec reformatting across 4 features in B36, 3 failed attempts across 6 features in P38, and blocked decomposition for Feature 9 in B38 — cumulatively consuming approximately 30% of planning-phase effort across the affected batches. The workaround (editing specs to checklist format, re-approving, and retrying) is mechanical but time-consuming and creates unnecessary doc re-approval churn.

**Suggested Fixes:**

1. Expand format recognition to include: heading-based ACs (`### AC-NNN`), bold-with-parenthetical (`**AC-NNN (REQ-NNN):**`), and Given/When/Then blocks.
2. When parsing fails, emit a diagnostic showing the closest matching patterns found in the document: "Found 14 sections including 'Acceptance Criteria' heading, but no lines matching expected format `**AC-NN.**`. Nearest candidates: lines 45–52 (`### AC-001` through `### AC-005`)."
3. Document the expected AC format prominently in the `write-spec` skill, ideally as an "AC Format Reference" section at the top.
4. Consider reading spec content directly from disk as a fallback when the `doc_intel` index is stale or missing content.

---

### Theme 3: State Consistency Bugs Between Tool Calls

**Severity:** Significant
**Reports:** 4 (retro-report-1, P40-report-retro-agent-session-experience, retro-report-5, retro-report-15)

**Synthesized Description:**

Multiple reports document cases where a tool call returns success but subsequent tool calls observe different state. Specific instances:

- **`finish()` returns success but gate checks see task as `active`** (retro-report-1): TASK-01KQFT5G5XMS7 was marked done via `finish()`, confirmed via `entity get`, but `entity list` and feature transition gates both saw it as `active`. Required two override transitions to work around.
- **Closed bugs still appear as attention items** (P40-report-retro-agent-session-experience): BUG-01KQB-54280HDP and BUG-01KQF-4CEB8Z90 both showed `status: "closed"` but the status dashboard still flagged them as "prioritise resolution." Attention recalculation lags behind entity state changes.
- **`entity list parent_feature` filter returns all 614 tasks** (retro-report-5): The `parent_feature` filter appears silently broken — queries return the entire project task list instead of the scoped subset. This is the most functionally impactful bug found.
- **T3 auto-promotion from `queued` to `ready` missed** (retro-report-15): After T2 completion, T3 should have auto-promoted but didn't. Required manual `entity transition` to unblock.

**Suggested Fixes:**

1. Ensure `finish()` uses the same write-through path for both the entity record and the cache/index — the inconsistency suggests two code paths that should be unified.
2. Recalculate attention items on every entity state mutation, or add a short TTL so stale warnings expire within one polling interval.
3. Fix the `parent_feature` filter in `entity(action: list)` — this is a correctness bug that affects every review and orchestration workflow.
4. Audit the auto-promotion hook for edge cases where a single-dependency task's dependency completes but the promotion signal is missed (race condition or batch-finish path gap).

---

### Theme 4: Document Ownership and Registration Friction

**Severity:** Significant
**Reports:** 4 (P40-retro-p38-plans-and-batches-implementation, retro-report-1, retro-report-5, P40-report-retro-report-4)

**Synthesized Description:**

Document ownership semantics create recurring confusion. Documents registered inside a plan/batch folder default to `PROJECT/` ownership rather than the containing entity. When `decompose` or feature-level tools search for documents by entity ID, they don't find `PROJECT/`-owned documents, causing feature-level operations to fail silently. In P38, this caused 3+ re-registration cycles across multiple features.

Related issues include: `doc refresh` reverting approval status on content hash change (requiring full re-approval for minor edits like AC format fixes); `record_false_positive` appearing to succeed but not actually bypassing the approval gate (two different code paths); document path format requirements being opaque (e.g., `review-` prefix requiring `work/_project/` folder with no documentation of the rule); the `id` vs `path` parameter distinction in the `doc` tool being unclear when entity records store doc references.

**Suggested Fixes:**

1. Auto-infer document owner from file path context: a spec at `work/B38-plans-and-batches/B38-spec-f2.md` should default to the feature that is a child of B38.
2. Warn when registering a document whose path is already registered under a different owner: "This path is already registered under PROJECT/. Did you mean owner: FEAT-xxx?"
3. `decompose` and feature-level tools should fall back to searching `PROJECT/`-scoped documents when feature-owned documents are not found.
4. `doc refresh` should preserve approval status for minor edits, or at minimum warn explicitly before resetting.
5. Unify the `record_false_positive` and `approve` code paths so the override consistently applies.
6. Surface expected path patterns in `doc register` error messages, or add a `doc(action: infer_path)` helper.

---

### Theme 5: Session Continuity and Context Window Gaps

**Severity:** Significant
**Reports:** 3 (retro-report-14, retro-report-15, P40-report-b38-implementation-session)

**Synthesized Description:**

When a sub-agent's context window is exhausted mid-task, or when an orchestrator session ends without clean `finish()` calls, the system leaves no recovery trail. In retro-report-15, the previous agent had uncommitted state (`internal/git/git.go` and `internal/service/documents.go` modified but not staged) with no continuation note. The resuming agent had to reconstruct intent via `git diff` and dev-plan cross-referencing — taking more tool calls than the actual implementation.

Compounding this: ~80 orphaned `.kbz/` files accumulated from uncommitted session state and had to be committed blind in a catch-all commit because the pre-task checklist requires a clean working tree. The `health` tool doesn't surface uncommitted `.kbz/` files as a warning. The `next` tool claims tasks on inspection — there's no read-only "peek" mode — making it harder to assess state without side effects.

The orchestrator context budget (30K bytes) is consistently at 99%+ utilization. Knowledge entries, vocabulary, anti-patterns, procedure steps, and examples all compete for space, and several knowledge entries are trimmed from context. This is particularly acute for the orchestrator role, which needs the full procedure and vocabulary to function correctly.

**Suggested Fixes:**

1. When a session ends without `finish()`, auto-reset the active task to `ready` with a timestamped note, and auto-commit any pending `.kbz/state/` changes.
2. `health` should report uncommitted `.kbz/` files as a warning, distinguishing between safe-to-commit (`knowledge/`, `index/`) and suspicious (`tasks/`, `features/`) directories.
3. Add a `peek: true` parameter to `next` for read-only context inspection without claiming the task.
4. Increase the orchestrator context budget to 40–45K bytes, or implement more aggressive knowledge deduplication (plan-scoped knowledge preferred over project-wide for feature-level work).
5. Add a sub-agent prompt budget guideline to the agents skill: "keep sub-agent prompts under ~800 words of instructions."
6. Auto-save a continuation note on the active task when a `finish` call is not the last action before session end.

---

### Theme 6: Worktree Lifecycle and Cleanup Automation Gaps

**Severity:** Moderate
**Reports:** 5 (P40-report-b38-implementation-session, P40-report-retro-report-4, retro-report-5, retro-report-12, retro-report-13)

**Synthesized Description:**

Worktrees accumulate as development proceeds and cleanup is not automated. Specific patterns:

- **Orphaned worktree records**: 60+ health errors about worktrees whose directories were manually deleted but whose `.kbz/state/worktrees/` YAML records persist. `worktree(action: remove)` fails with a git-level "not a working tree" error for these.
- **Squash-merged branches appear as critically drifted**: After a squash merge, the local branch is 321 commits behind main, triggering drift alerts even though a `merged_at` timestamp exists. The branch health check doesn't account for squash-merge semantics.
- **Duplicate worktrees for same entity**: `worktree(action: remove)` takes only `entity_id` and can't disambiguate when two worktrees exist for the same entity. Input validation accepts display-ID format (`FEAT-01KQ7-JDT511BZ`) creating ghost records pointing to non-existent entities.
- **Remote branches of merged features survive**: Six stale remote branches from features merged weeks ago. The `cleanup` tool only handles local worktrees.
- **Worktrees auto-created on bug transitions**: Transitioning a bug to `in-progress` auto-creates a worktree even when the fix is already on main.
- **Worktrees for shared-file edits**: When a feature only touches shared repo-root files (`.agents/skills/`, `internal/kbzinit/`), the worktree provides no isolation value.

**Suggested Fixes:**

1. Add `worktree(action: gc)` to detect and remove orphaned records where the git worktree directory no longer exists.
2. When a branch has `merged_at` in the worktree record, skip drift alerts — the branch is intentionally stale.
3. Add optional `worktree_id` parameter to `worktree(action: remove)` for disambiguation.
4. Validate entity IDs at worktree creation, rejecting display-ID format (with embedded segment hyphen).
5. Auto-schedule worktree cleanup on merge, and auto-delete remote branches by default.
6. Warn (or skip auto-creation) when creating a worktree for a bug, or when the entity's work only touches non-`work/` files.
7. Delete local tracking branches on squash merge so drift checks don't trigger.

---

### Theme 7: Merge and Lifecycle Gate Design Issues

**Severity:** Moderate
**Reports:** 4 (retro-report-5, retro-report-12, retro-report-13, P40-report-b38-implementation-session)

**Synthesized Description:**

The merge gate system is well-designed in principle but has specific design issues:

- **EntityDoneGate ordering**: The merge tool checks pre-merge gates (including EntityDoneGate requiring `done` status) before the merge and before the auto-advance hook runs. Features in `developing` with all tasks complete get caught in a catch-22.
- **Missing verification setter**: Merge gates require `verification_exists` and `verification_passed` on features, but `entity update` has no way to set these fields outside the `finish` task flow.
- **`merge check` doesn't distinguish hard vs soft gates**: `review_report_exists` is a hard gate that cannot be bypassed, but this is only discovered at execution time.
- **Batch close ceremony**: After a batch conformance review is approved, reaching `done` still requires `active → reviewing → done` — the `reviewing` hop feels redundant.
- **No PR workflow support**: The merge gate assumes a GitHub PR workflow, but the project uses a single-user agentic workflow. PR gates can't be made optional.
- **Feature 8 can't close without tasks**: Features with 0 tasks that were implemented externally can't be advanced past `reviewing`.

**Suggested Fixes:**

1. Reorder merge tool: merge → auto-advance lifecycle → check post-merge gates.
2. Add `verification` parameter to `entity(action: update)` for post-hoc verification recording.
3. Add `bypassable: bool` field to each gate result in `merge(action: check)` output.
4. Allow direct `active → done` batch transition when an approved batch conformance review is on record.
5. Make PR gates optional or configurable for single-user workflows.
6. Support an "implementation-not-required" or "verified-complete" transition path for features with 0 tasks.

---

### Theme 8: Bug Lifecycle Is Friction-Heavy

**Severity:** Moderate
**Reports:** 2 (P40-report-retro-report-4, retro-report-8)

**Synthesized Description:**

Closing a bug requires 7 sequential `entity(action: transition)` calls (reported → triaged → planned → in-progress → needs-review → verified → closed), each a round-trip. The `advance` parameter exists for features but not for bugs. For bugs that are already fixed on main, this feels like ceremony without value.

Additionally, bugs fixed directly on main (not on bug branches) don't auto-advance — the commit is the real closure signal, but the entity state is an afterthought. The `finish` tool doesn't apply to bugs, and the error message is misleading ("task not found" instead of "bugs use entity transition, not finish").

**Suggested Fixes:**

1. Add `advance` parameter support for bugs, or add a `close` action that chains through intermediate states in a single call.
2. Consider a "bug resolved by commit" heuristic: if a commit message references a bug ID and the bug is in a non-terminal state, surface it as a health warning.
3. Improve `finish` error message for non-task entities: "finish is for tasks; use entity(action: transition) for bugs/features."
4. Skip worktree auto-creation when transitioning a bug to `in-progress` if the fix is already on main.

---

### Theme 9: Tool Discoverability and Cross-Tool Consistency

**Severity:** Moderate
**Reports:** 3 (retro-report-5, retro-report-1, P40-report-retro-agent-session-experience)

**Synthesized Description:**

The tool ecosystem has internal consistency (each tool's schema is logical) but lacks cross-tool consistency. Specific gaps:

- **Parameter naming**: `entity` uses `parent: "B38-plans-and-batches"` while `doc` uses `owner: "B38-plans-and-batches"` — same value, different parameter name.
- **`id` vs `path` ambiguity**: Entity records store a `spec` field that looks like a path but is actually a doc record ID. `doc(action: get, path: ...)` fails silently.
- **Short-form ID resolution**: `B38` doesn't resolve to `B38-plans-and-batches` — it returns silent empty results instead of "did you mean?"
- **`status` doesn't support batch IDs**: `status(id: "B38-plans-and-batches")` returns a parse error — it only recognizes `P` prefix for legacy plan IDs.
- **`retro` scope for batch IDs returns 0 signals silently**: It's unclear whether this is "no signals exist" or "batch scoping isn't supported."
- **Document audit skips files silently**: `doc(action: import)` skips files that can't be auto-typed without suggesting `default_type` as a remedy.
- **No credential-gap early warning**: The absence of a GitHub token is only discovered at `pr(action: create)` time — at the end of a multi-step workflow.

**Suggested Fixes:**

1. Standardize parameter naming: use `owner` consistently across `entity`, `doc`, `status`, and `retro` for entity/scope references.
2. Accept short-form IDs (`B38`) across all tools with a "did you mean B38-plans-and-batches?" error on ambiguity.
3. Accept doc IDs via the `path` parameter as a fallback (try ID lookup when path lookup fails).
4. Add batch ID support to `status()`.
5. Make `retro` return explicit "0 signals found for scope X — try scope: project" rather than silent empty results.
6. Surface credential gaps at workflow start (worktree creation or feature transition to `developing`).
7. Improve doc import UX to report skipped-file count and suggest `default_type`.

---

### Theme 10: edit_file Reliability Issues

**Severity:** Moderate
**Reports:** 2 (retro-report-1, retro-report-8)

**Synthesized Description:**

The `edit_file` tool has reliability issues beyond the worktree problem:

- **Silent partial application of multi-edit calls**: When some `old_text` patterns in a multi-edit call don't match, the matching edits are applied and non-matching ones are silently skipped. In one case, this caused deletions without replacements, corrupting a dev-plan file.
- **"old_text did not match" on verified-correct text**: After three attempts with manual verification between each, `edit_file` repeatedly failed to match text that was confirmed present. Suggests a stale-buffer issue.
- **`write_file` directory-not-found errors**: `write_file` failed with a directory error, possibly misinterpreting the project root path.

**Suggested Fixes:**

1. Make multi-edit calls atomic — either apply all edits or none, with clear error on which pattern failed to match.
2. When `old_text` doesn't match, offer to re-read the file and show a diff of what changed since the last read.
3. Investigate the stale-buffer hypothesis — `edit_file` may not be reading the latest on-disk contents before attempting the match.

---

### Theme 11: Spec/Skill Validation Mismatches

**Severity:** Minor
**Reports:** 2 (P40-report-retro-report-4, retro-report-9)

**Synthesized Description:**

The `write-spec` skill describes heading structures that don't match the validation scripts. The skill says `## Problem Statement` and `## Requirements > ### Functional Requirements`, but the `validate-spec-structure.sh` script requires `## Overview`, `## Scope`, `## Functional Requirements` (top-level). Agents discover the mismatch by trial and error after validation fails.

Similarly, skill files have no integrity checks. A commit truncated `write-spec/SKILL.md` mid-sentence (310 deletions, 2 insertions), and no pre-commit or CI hook caught the structural break.

**Suggested Fixes:**

1. Align `write-spec` skill heading names with validation script requirements (or vice versa).
2. Add a `kbz validate --skills` command or pre-commit hook that verifies every `SKILL.md` has all required sections.
3. Add a diff/stat ratio check: if deletions > N× insertions in a skill file, warn before commit.

---

### Theme 12: Knowledge and Context Assembly Signal-to-Noise

**Severity:** Minor
**Reports:** 2 (P40-report-b38-implementation-session, retro-report-12)

**Synthesized Description:**

The context assembly from `next()` includes ~70 knowledge entries, many of which are retrospective signals from unrelated plans. For feature-level work, plan-scoped knowledge would be more relevant than project-wide knowledge.

The codebase knowledge graph (`codebase-memory-mcp`) was not indexed for this project, so `search_graph`, `trace_path`, and `query_graph` all returned "project not found." Agents fell back to `grep`, which worked but was slower. The `AGENTS.md` instructions assume graph tools are available, but they weren't.

**Suggested Fixes:**

1. Filter knowledge entries in context assembly to prefer plan-scoped entries over project-wide entries for feature-level work.
2. Ensure the knowledge graph is indexed — either rebuild automatically or update `AGENTS.md` to acknowledge the fallback path.
3. Improve the `grep`/`codebase_memory_mcp_search_code` tool failure messages — "tool input was not fully received" is opaque.

---

### Theme 13: CLI and MCP Health-Check Divergence

**Severity:** Minor
**Reports:** 1 (retro-report-12)

**Synthesized Description:**

`kanbanzai status` (CLI) reported 158 "non-existent batch" errors while the MCP `status` and `health` tools reported zero. The discrepancy made the user experience alarming when the underlying state was clean. The `P{n}` ID aliasing (`EntityKindPlan` silently aliases to `EntityKindBatch`) was a contributing factor.

**Suggested Fixes:**

1. Unify CLI and MCP health-check code paths so they produce identical results.
2. Write dual entity state during migration: when a plan is created in `.kbz/state/plans/`, also write a batch record in `.kbz/state/batches/` until migration completes.
3. Make the `P{n}` → `B{n}` alias explicit in health-check queries rather than relying on silent model-layer aliasing.

---

### Theme 14: Cross-Feature Dependency Visibility

**Severity:** Minor
**Reports:** 1 (retro-report-15)

**Synthesized Description:**

When implementing `kbz move` (P37-F3), the sub-agent needed functions that existed in a feature that was `done` but not yet merged to main. The `next`/`handoff` context assembly had no way to flag this gap. The agent discovered it by searching for function names and finding them in a different worktree, requiring inline re-implementation.

**Suggested Fixes:**

1. When a task's spec or dev-plan references functions from a feature that is `done` but unmerged, surface a warning in the `handoff` context: "FEAT-X is done but not yet merged to main — these symbols are not available."

---

### Theme 15: Test Infrastructure Friction

**Severity:** Minor
**Reports:** 2 (retro-report-14, retro-report-1)

**Synthesized Description:**

- `TestDisplayID_AC017_ResolutionPerformance` flakes under parallel `go test ./...` load (216ms vs 100ms bound) but passes in isolation (~33ms). Produces noise in full-suite runs.
- Tests sometimes assert against implementation-specific strings rather than spec-required output, making them fragile.

**Suggested Fixes:**

1. Raise the performance test bound to 500ms or skip it under parallel load.
2. Document in the `implement-task` skill that tests should assert against spec requirements, not implementation details.

---

### Theme 16: Stale Binary Diagnostics

**Severity:** Minor
**Reports:** 1 (P40-report-retro-synthesis)

**Synthesized Description:**

A stale MCP binary caused 3 of 9 acceptance criteria to appear to fail when the code was correct, wasting significant verification time. The `server_info` tool was added as a fix, but discoverability of stale-binary-as-root-cause remains a gap.

**Suggested Fixes:**

1. Add a startup self-check that compares the running binary's build timestamp against the source tree's latest commit and warns on mismatch.
2. Document the `server_info` tool in the getting-started checklist as a first diagnostic step for unexpected test failures.

---

## 3. What Consistently Works Well

### Sub-Agent Parallelism with Disjoint File Scopes

**Reports:** 7 (P40-report-retro-synthesis, P40-report-b38-implementation-session, P40-retro-p38-plans-and-batches-implementation, retro-report-1, retro-report-2, retro-report-12, retro-report-15)

The parallel dispatch model is the system's strongest scaling feature. Zero file conflicts were reported across dozens of parallel sub-agent dispatches. Key enablers: clear file ownership boundaries in task delegation, the conflict detection via `next(conflict_check: true)`, and the worktree isolation system. Disjoint file edits (different tool files, different renderer packages) consistently completed in parallel without coordination overhead.

### The `health()` Diagnostic Tool

**Reports:** 5 (P40-report-retro-report-4, retro-report-5, retro-report-12, retro-report-8, retro-report-4)

Consistently described as the highest-signal single tool call in any session. Returns structured, actionable output: merge conflicts, branch drift, stale worktrees, TTL-expired knowledge, doc-currency gaps, and feature-child-consistency warnings — all in one call. Converts vague "something is wrong" into a triaged work list with entity IDs and severity levels.

### Stage Binding → Role → Skill Chain

**Reports:** 4 (retro-report-5, P40-report-retro-agent-session-experience, retro-report-8, P40-report-b38-implementation-session)

Following the chain (stage-bindings.yaml → role YAML → skill SKILL.md) provides a complete picture of what to do, what vocabulary to use, what anti-patterns to avoid, and what output format to produce. The checklist structure means agents never have to guess the next step. This scaffolding is consistently praised as "genuinely useful" and "well-structured."

### Spec → Dev-Plan → Implement Traceability

**Reports:** 4 (P40-retro-p38-plans-and-batches-implementation, retro-report-2, retro-report-8, retro-report-14)

The pipeline from specification to dev-plan to implementation held up under sustained load. Acceptance criteria mapped cleanly to tasks, specs served as reliable contracts during implementation, and the design→spec→implement cycle caught gaps before code was written. The cache infrastructure (P29 design) was specifically praised: write-through consistency, fallback-on-miss discipline, and correct-by-construction staleness handling.

### `merge(action: check)` Output Quality

**Reports:** 2 (retro-report-5, retro-report-12)

Precise, named gate failures with actionable messages. The tool tells you exactly what's missing and what to do, eliminating guesswork about why a merge is blocked. The override mechanism with required `override_reason` forces articulation of justification without blocking legitimate cases.

### Handoff Context Assembly

**Reports:** 2 (retro-report-14, retro-report-15)

`handoff`-generated context packets contain the right things: spec sections, knowledge entries, role identity, and codebase references. Sub-agents receiving handoff prompts completed tasks correctly on the first attempt. The handoff document pattern is effective at spanning context-window boundaries.

### Status Dashboard as Single Source of Truth

**Reports:** 3 (P40-retro-p38-plans-and-batches-implementation, retro-report-12, retro-report-15)

`status()` provides an instant, accurate picture of plan/feature/task state. Replaces the mental tracking that typically consumes significant orchestrator context. The attention item grouping by severity and entity type effectively triages work order.

### Entity Lifecycle Gate Enforcement

**Reports:** 2 (P40-report-b38-implementation-session, P40-report-retro-report-4)

The requirement to decompose before implementing, and to have approved specs before decomposing, prevented at least one premature implementation attempt. Error messages were clear about what was blocking and how to resolve it.

### Document Pipeline and Sub-Agent Editorial Workflow

**Reports:** 2 (P40-report-retro-synthesis, P40-report-retro-report-4)

The document pipeline (write → edit → check → style → copyedit) produced consistent, accurate output. Fact-checking sub-agents found 8 genuine inaccuracies across two doc-publishing sessions that would have been published uncorrected. The per-stage sub-agent delegation produced focused, non-overlapping changes.

### Display ID System

**Reports:** 1 (retro-report-15)

The `FEAT-01KQ7-JDT11MH6` display ID format is significantly more human-readable than raw TSIDs. Marked as a "genuine UX win."

---

## 4. Quick Wins

Prioritized by ROI: impact relative to implementation scope.

| # | Fix | What It Fixes | Scope | Supporting Reports |
|---|-----|--------------|-------|-------------------|
| 1 | **Document `write_file(entity_id: ...)` as primary worktree pattern** in `implement-task` skill; add it to developing-stage tool subset | Eliminates triple-escaping workarounds; reduces wasted tool calls by 3–5 per sub-agent session | Small | P40-report-b38, P40-report-retro-synthesis, P40-retro-p38, retro-report-15 |
| 2 | **Fix `entity list parent_feature` filter** — currently returns all 614 tasks instead of scoped subset | Restores fundamental task-per-feature query; affects every review and orchestration workflow | Small | retro-report-5 |
| 3 | **Expand decompose AC format recognition** — add heading-based ACs, bold-with-parenthetical, and Given/When/Then | Eliminates 30% of planning-phase format-fix churn; unblocks decomposition without spec rewrites | Small | P40-report-b38, P40-retro-p38, retro-report-1, retro-report-2 |
| 4 | **Add `bypassable: bool` to `merge(action: check)` gate results** | One field addition; prevents execution-time surprise on hard vs soft gates | Small | retro-report-5 |
| 5 | **Add short-form ID resolution** (`B38` → `B38-plans-and-batches`) with "did you mean?" error | Eliminates silent empty-result confusion across all tools | Small | retro-report-5 |
| 6 | **Unify `finish()` state propagation** — ensure write-through to both entity record and cache/index | Fixes the most impactful state consistency bug (done tasks seen as active by gates) | Small | retro-report-1 |
| 7 | **Make multi-edit calls atomic** — all-or-nothing application with clear error on which pattern failed | Prevents silent file corruption from partial edit application | Small | retro-report-1 |
| 8 | **Add `verification` parameter to `entity(action: update)`** | Closes the post-rebase re-verification gap without gate overrides | Small | retro-report-5, retro-report-13 |
| 9 | **Add batch ID support to `status()`** | Primary dashboard tool works for the current entity type | Small | retro-report-5 |
| 10 | **Auto-infer document owner from path context** during `doc register` | Eliminates 3+ re-registration cycles per batch; prevents decompose lookup failures | Medium | P40-retro-p38 |
| 11 | **Surface credential gaps at worktree creation** not at `pr create` time | One early warning vs late-stage workflow failure | Small | retro-report-14 |
| 12 | **Add `worktree(action: gc)` for orphaned records** | Eliminates 60+ stale health warnings and manual state-file cleanup | Medium | retro-report-12 |

---

## 5. Appendix: Finding Inventory

| # | Report Source | Finding Summary | Category | Severity |
|---|--------------|----------------|----------|----------|
| 1 | P40-report-retro-synthesis | `edit_file` writes to main repo root, not worktrees | tool-friction | Critical |
| 2 | P40-report-b38-implementation-session | Worktree file editing via python3 -c/heredocs is fragile; `write_file(entity_id)` exists but is undiscoverable | tool-friction | Critical |
| 3 | P40-report-retro-synthesis | Heredoc syntax consistently fails in terminal tool's sh shell | tool-friction | Moderate |
| 4 | P40-report-retro-synthesis | Shell string escaping in python3 -c with embedded YAML/Go requires multiple attempts | workflow-friction | Moderate |
| 5 | retro-report-15 | `write_file` with `entity_id` is significantly better than heredocs — should be primary pattern | tool-friction | Moderate |
| 6 | P40-report-b38-implementation-session | decompose can't parse `**AC-001 (REQ-001):**` format; only recognizes three formats | tool-friction | Significant |
| 7 | P40-retro-p38-plans-and-batches-implementation | decompose AC format requirements caused 3 failed attempts across 6 features; error message unhelpful | tool-friction | Significant |
| 8 | retro-report-2 | decompose format mismatch consumed ~30% of B36 planning-phase effort across 4 features | tool-friction | Significant |
| 9 | retro-report-1 | Four rounds of fix-and-retry for AC formatting before decomposition succeeded | workflow-friction | Significant |
| 10 | retro-report-1 | `finish()` returned success but task was still seen as `active` by gates; state inconsistency | tool-friction | Significant |
| 11 | retro-report-1 | `edit_file` silently skipped non-matching edits in multi-edit call, corrupting dev-plan | tool-friction | Significant |
| 12 | P40-report-retro-agent-session-experience | Closed bugs still appear as attention items; attention recalculation lags | tool-friction | Moderate |
| 13 | retro-report-5 | `entity list parent_feature` filter returns all 614 tasks — filter is silently broken | tool-friction | Significant |
| 14 | retro-report-15 | T3 auto-promotion from queued to ready didn't fire after T2 completion | tool-friction | Moderate |
| 15 | P40-retro-p38-plans-and-batches-implementation | Documents default to PROJECT/ ownership, not feature ownership; causes decompose lookup failures | tool-friction | Significant |
| 16 | retro-report-2 | doc refresh reverts approval status on content hash change; requires full re-approval for minor edits | workflow-friction | Moderate |
| 17 | retro-report-2 | Approval gate ignores `record_false_positive` entries; approve and validate use different code paths | tool-friction | Significant |
| 18 | retro-report-5 | Document path format requirements opaque; `review-` prefix requires `work/_project/` with no documentation | tool-friction | Moderate |
| 19 | retro-report-5 | doc `id` vs `path` distinction unclear; entity spec field looks like a path but is actually an ID | tool-friction | Moderate |
| 20 | retro-report-13 | Document registration rejected review file at `work/reviews/`; required path restructuring | workflow-friction | Moderate |
| 21 | retro-report-14 | Sub-agent exhausted context window mid-task; no recovery trail left | workflow-friction | Significant |
| 22 | retro-report-15 | ~80 orphaned .kbz/ files committed blind; no way to inspect safety before committing | workflow-friction | Significant |
| 23 | retro-report-15 | Session interruption leaves no continuation note; recovery requires git diff archaeology | workflow-friction | Significant |
| 24 | P40-report-b38-implementation-session | Context budget (30K) at 99%+ utilization; knowledge entries trimmed; insufficient for orchestrator role | workflow-friction | Moderate |
| 25 | retro-report-12 | 60+ health errors about orphaned worktrees with missing paths/branches | workflow-friction | Moderate |
| 26 | retro-report-12 | Squash-merged branches appear as critically drifted (321 commits behind) despite merged_at timestamp | tool-friction | Moderate |
| 27 | retro-report-5 | Duplicate worktrees for same entity; `worktree remove` can't disambiguate without worktree_id | tool-friction | Moderate |
| 28 | retro-report-5 | Ghost worktree from display-ID format misuse; input validation doesn't reject malformed entity IDs | tool-friction | Moderate |
| 29 | retro-report-12 | Six stale remote branches survive after merge; cleanup tool only handles local worktrees | workflow-friction | Minor |
| 30 | retro-report-12 | `worktree(action: remove)` fails for orphaned records with git-level "not a working tree" error | tool-friction | Moderate |
| 31 | P40-report-retro-report-4 | Worktree auto-creation on bug transition creates waste for already-fixed bugs | workflow-friction | Minor |
| 32 | retro-report-8 | Worktree for shared-file edits provides no isolation value but adds cleanup overhead | workflow-friction | Minor |
| 33 | retro-report-12 | EntityDoneGate ordering: merge checks pre-merge gates before auto-advance, creating catch-22 | workflow-friction | Moderate |
| 34 | retro-report-5 | Merge gate `review_report_exists` is hard-blocking but `check` doesn't distinguish hard vs soft gates | tool-friction | Moderate |
| 35 | retro-report-5 | No `verification` setter on entity update; post-rebase re-verification requires gate override | tool-friction | Moderate |
| 36 | retro-report-5 | Batch close requires `active → reviewing → done`; reviewing hop redundant after formal review | workflow-friction | Minor |
| 37 | retro-report-13 | Merge gate assumes GitHub PR workflow; no way to make PR gates optional for single-user workflow | workflow-friction | Moderate |
| 38 | P40-report-b38-implementation-session | Feature 8 with 0 tasks can't advance past reviewing; no "implementation-not-required" path | workflow-friction | Moderate |
| 39 | P40-report-retro-report-4 | Bug closure requires 7 sequential transition calls; no `advance` support for bugs | workflow-friction | Significant |
| 40 | P40-report-retro-report-4 | Bugs fixed on main don't auto-advance; commit is closure signal but entity state lags | workflow-friction | Moderate |
| 41 | retro-report-8 | `finish` on bug ID returns misleading "task not found" instead of "use entity transition" | tool-friction | Minor |
| 42 | retro-report-5 | `entity` uses `parent` parameter while `doc` uses `owner` — same value, different names | tool-friction | Minor |
| 43 | retro-report-5 | Short ID `B38` doesn't resolve to `B38-plans-and-batches`; silent empty result instead of suggestion | tool-friction | Moderate |
| 44 | retro-report-5 | `status()` doesn't support batch IDs — only legacy P-prefix plan IDs | tool-friction | Moderate |
| 45 | retro-report-5 | `retro(scope: batch)` returns 0 signals silently; unclear if batch scoping is supported | tool-friction | Minor |
| 46 | P40-report-retro-agent-session-experience | Design document under `work/_project/` not discoverable from batch without grep | workflow-friction | Minor |
| 47 | P40-report-retro-agent-session-experience | doc import skips untyped files without suggesting default_type remedy | tool-friction | Minor |
| 48 | retro-report-14 | No credential-gap warning at workflow start; GitHub token absence only discovered at `pr create` | workflow-friction | Moderate |
| 49 | retro-report-8 | `edit_file` repeatedly failed "old_text did not match" on verified-correct text; stale-buffer hypothesis | tool-friction | Moderate |
| 50 | retro-report-1 | `write_file` failed with directory-not-found error; possible project root path misinterpretation | tool-friction | Minor |
| 51 | P40-report-retro-report-4 | write-spec skill heading names don't match validate-spec-structure.sh requirements | workflow-friction | Minor |
| 52 | retro-report-9 | Skill file truncated mid-sentence (310 deletions in commit); no integrity check caught it | workflow-friction | Minor |
| 53 | P40-report-b38-implementation-session | ~70 knowledge entries in context assembly; many from unrelated plans dilute signal | workflow-friction | Minor |
| 54 | P40-report-retro-report-4 | Codebase knowledge graph not indexed; graph tools unavailable despite AGENTS.md instructions | tool-friction | Minor |
| 55 | retro-report-12 | CLI `kanbanzai status` reports 158 errors while MCP tools report zero; divergent code paths | tool-friction | Moderate |
| 56 | retro-report-12 | P{n} ID aliasing silently delegates to batch lookup; plans in plans/ dir not found in batches/ dir | workflow-friction | Minor |
| 57 | retro-report-15 | Cross-feature dependency not surfaced: done-but-unmerged feature's symbols invisible to other worktrees | workflow-friction | Minor |
| 58 | retro-report-15 | `next` claims task on inspection; no `peek` mode for read-only context inspection | tool-friction | Minor |
| 59 | retro-report-14 | Performance test flakes under parallel load (216ms vs 100ms bound); passes in isolation at 33ms | tool-friction | Minor |
| 60 | retro-report-1 | Tests assert against implementation strings rather than spec requirements; fragile and miss violations | workflow-friction | Minor |
| 61 | P40-report-retro-synthesis | Stale MCP binary caused 3/9 ACs to appear to fail; no built-in version check at startup | tool-friction | Minor |
| 62 | P40-report-retro-synthesis | Commit discipline not followed for management actions; checklist only visible when task claimed | workflow-friction | Moderate |
| 63 | P40-report-retro-synthesis | Circular dependency between tasks 4 and 6 required manual intervention; entity model doesn't support partial completion | decomposition-issue | Minor |
| 64 | P40-report-retro-synthesis | Spec FR-002 omitted Rationale from paired test task fields but invariant test required it | spec-ambiguity | Minor |
| 65 | P40-report-retro-synthesis | AC-07 spec ambiguity: describes "needs-review blocks auto-advance" but needs-review is non-terminal | spec-ambiguity | Minor |
| 66 | retro-report-2 | dev-plan described 4 tasks but only 2 created as entities; missing task validation on approval | workflow-friction | Moderate |
| 67 | retro-report-2 | Tasks referenced non-existent dependency IDs; dev-plan approval should validate dependency references | workflow-friction | Minor |
| 68 | retro-report-5 | MCP auto-commits and manual commits collide silently; commit SHA not surfaced in tool response | tool-friction | Minor |
| 69 | P40-report-retro-report-4 | Agent jumped to code before analysis on a diagnostic question; bias toward action-completion | workflow-friction | Moderate |
| 70 | retro-report-13 | Feature re-parenting confused status cache; cache clear/rebuild didn't fix stale plan membership display | tool-friction | Moderate |
| 71 | retro-report-13 | Plan lifecycle requires reviewing intermediate state; advance parameter doesn't work for plans | workflow-friction | Minor |
| 72 | retro-report-14 | `pr` tool fails on missing credentials; `create_pull_request` fallback also fails (401); only git push + manual URL works | tool-friction | Moderate |
| 73 | retro-report-14 | No gate or prompt on `transition → reviewing` without a PR; post-hoc warning is too late | workflow-friction | Minor |
| 74 | retro-report-15 | `finish` batch error unhelpful: "task in status queued (expected ready/active)" with no corrective action suggestion | tool-friction | Minor |
| 75 | retro-report-15 | T4 introduced `--force` regression: cross-cutting flag parsing concern not surfaced in task description | decomposition-issue | Minor |
| 76 | retro-report-8 | kbzinit dual-write enforced by convention only; no CI/health check for drift between `.agents/` and `internal/kbzinit/` | workflow-friction | Minor |
| 77 | retro-report-8 | No standalone retro signal collection during bug work; `finish` is the only signal recording point | tool-friction | Minor |
| 78 | retro-report-12 | Merge tool's auto-advance attempt failed silently (developing→done rejected); required second commit to fix | workflow-friction | Minor |
| 79 | P40-report-retro-agent-session-experience | Batch list interleaves active/proposed with done chronologically; grouping active at top would help | workflow-friction | Minor |
| 80 | P40-report-retro-agent-session-experience | Design-to-batch connection requires grep; `doc_intel find-by-entity` shortcut would be more natural | tool-friction | Minor |
| 81 | P40-report-b38-implementation-session | Feature display IDs show P38-F2 format after B38 rename; display ID cache not rebuilt post-migration | tool-friction | Minor |
| 82 | P40-retro-p38-plans-and-batches-implementation | Sub-agent worktree changes overwrote prior sub-agent's changes (F2 entity_tool.go clobbered by F6) | workflow-friction | Moderate |
| 83 | retro-report-1 | Orchestrator direct implementation has no handoff-assembled context packet; lightweight context mode needed | workflow-friction | Minor |
| 84 | retro-report-5 | `status(id: "B38-plans-and-batches")` not supported — resolution error on B prefix | tool-friction | Moderate |
| 85 | P40-report-b38-implementation-session | Feature 8 stuck in limbo — developing status, 0 tasks, draft spec, work already done externally | workflow-friction | Moderate |

---

*85 distinct findings synthesized from 16 source reports. Findings numbered 1–85. Severity distribution: 2 Critical, 18 Significant, 36 Moderate, 29 Minor.*
