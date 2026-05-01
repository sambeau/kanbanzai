# Design: Retrospective Batch Improvements — April 2026

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Plan   | P40-retro-batch-april-2026    |

---

## Overview

This design addresses 14 improvement items across four independent workstreams,
drawn from the April 2026 retrospective batch's collated feedback report (85 findings
from 16 agent reports). Each item is a targeted fix to an existing MCP tool, skill
file, or lifecycle hook — no new architectural components are introduced. The
workstream structure (A: worktree experience, B: tool correctness, C: document
ownership, D: cleanup automation) enables parallel implementation within a batch
and staggered delivery by priority.

All changes are backward-compatible: new parameters are optional, existing
behaviour is preserved when new parameters are omitted, and no existing tool
signatures are removed.

---

## Goals and Non-Goals

**Goals**

- Make worktree file editing discoverable and reliable through the existing
  `write_file` and `edit_file` tools, eliminating the triple-escaping workaround
  patterns documented across five retro reports.
- Fix four correctness bugs in the tool surface: the broken `parent_feature`
  filter, inconsistent `finish()` state propagation, non-atomic multi-edit
  application, and brittle decompose AC format recognition.
- Auto-infer document ownership from file path context during registration,
  eliminating the PROJECT/-owner default that causes 3+ re-registration cycles
  per batch.
- Automate worktree cleanup on merge and add garbage collection for orphaned
  records, reducing the 60+ stale-worktree health warnings to zero.
- Improve merge gate UX with bypassable signalling and a verification setter.

**Non-Goals**

- This design does not introduce new MCP tools (except possibly `worktree(action: gc)`).
  All changes extend existing tools' parameter surfaces.
- This design does not change the knowledge graph, context assembly budget, or
  knowledge deduplication logic.
- This design does not address bug lifecycle fast-forward, CLI/MCP health-check
  unification, cross-feature dependency visibility, or session continuity auto-commit.
  These are documented as out-of-scope follow-up work.
- This design does not change entity lifecycle states, stage gate definitions, or
  the plan/batch entity model.

---

## Dependencies

| Dependency | Type | Notes |
|------------|------|-------|
| B38 — Plans and Batches (migration complete) | Runtime prerequisite | Document owner inference (C1) must correctly distinguish plan-owned from batch-owned paths. The P38 migration's `kbz migrate` must have completed before C1 can resolve owners from the new path conventions. |
| P34 — Agent Workflow Ergonomics | Design precedent | A1–A3, B1–B4, C1–C4, and D1–D4 follow the same extension pattern established in P34: add optional parameters to existing tools, preserve backward compatibility, and ship independently. |
| `write_file` `entity_id` parameter (P21) | Code dependency | A1 documents an existing capability; A2 mirrors the same worktree resolution logic in `edit_file`. Both depend on the worktree path resolution added in P21. |
| `AutoDeleteRemoteBranch` config option | Config dependency | D3's auto-delete behaviour depends on this config flag existing and defaulting to `true`. If the flag defaults to `false`, D3 should change the default. |
| `OnStatusTransition` hook (P19) | Code dependency | B2's unified state propagation may need to ensure the hook fires synchronously before the finish response returns. The hook contract (best-effort, must not block) must be preserved. |

---

## Problem and Motivation

The April 2026 retrospective batch synthesised 85 distinct findings from 16 agent
feedback reports across B34–B38 implementation sessions, bug fixes, batch reviews,
and CLI investigations. While the Kanbanzai workflow system's core architecture is
sound — sub-agent parallelism, lifecycle gate enforcement, the `health()` diagnostic
tool, and the spec→dev-plan→implement pipeline all received consistently positive
reports — friction is concentrated in the **tooling layer**.

Twelve of sixteen thematic clusters represent unfixed, recurring friction that
affects every batch going forward. The two critical-severity themes alone — worktree
file editing and tool reliability/state consistency — account for cumulative
wasted tool calls measured in dozens per session.

This design proposes targeted fixes across four subsystems — the MCP tool surface,
the document registration flow, worktree lifecycle management, and context
assembly — prioritised by impact-to-effort ratio. Each change is independently
deliverable as a feature within a batch; none requires architectural restructuring.

Doing nothing means the same friction patterns will recur in every future batch,
eroding agent productivity by an estimated 15–30% through wasted tool calls,
format-fix cycles, and state-repair operations.

---

## Related Work

**Corpus concepts searched:** worktree file editing, decompose AC format, document
ownership, state consistency, session continuity, merge gate design.

**Prior decisions and designs found:**

| Document | Relevance |
|----------|-----------|
| P34 — Agent Workflow Ergonomics | Addressed six friction points including short-form ID resolution (H-1), task auto-promotion (H-2), idempotent claim (H-3), worktree path in context (H-4), paired test tasks (H-5), and decompose apply supersession (H-6). All six are implemented. This design extends the same pattern to the next wave of friction. |
| P37 — File Names and Actions | Established canonical filename templates, plan-scoped feature display IDs, and document type enforcement. The document ownership and registration friction in this design is a direct consequence of P37's path conventions being enforced without owner inference. |
| P38 — Plans and Batches | Introduced recursive plan entities and renamed plans to batches. Several findings (CLI health-check divergence, display ID staleness, retro scope for batch IDs) are transitional artefacts that will self-resolve as migration completes. |
| Retro Report 12 (branch audit) | Applied direct fixes for merge auto-advance ordering and EntityDoneGate relaxation. These fixes are on main but were applied outside a formal batch — they are not yet tracked as features with verification records. |
| Collated Feedback Report (P40) | The primary evidence base. 85 findings synthesised into 16 themes with severity ratings, report counts, and merged suggested fixes. |

No prior design addresses the worktree file-editing experience, the decompose AC
format brittleness, document owner inference, or worktree cleanup automation.
These are new design territory.

---

## Design

The design is organised into four independent workstreams, each of which can be
delivered as a batch of features. Within each workstream, individual features are
ordered by dependency and can be implemented in parallel where file scopes are
disjoint.

### Workstream A: Fix the Worktree Development Experience

**Components affected:** `edit_file` tool, `write_file` tool, `implement-task` skill,
`kanbanzai-agents` skill, terminal tool shell configuration.

#### A1 — Document `write_file(entity_id)` as primary worktree pattern

**What:** Update `implement-task/SKILL.md` to make `write_file(entity_id: ...)` the
recommended primary pattern for creating and modifying files in worktrees. Remove
all references to `python3 -c` and heredoc workarounds as the recommended approach.
Add `write_file` to the default tool subset for the `developing` stage so sub-agents
discover it without explicit instruction from the orchestrator.

**Why:** `write_file` already works correctly with worktrees via its `entity_id`
parameter. The capability exists and is tested. The gap is purely discoverability —
every agent that discovers it independently reports it as "significantly better than
heredocs" (retro-report-15, B38 implementation report). The fix is a single skill
file edit with outsized workflow impact.

**Interface contract:** No code change. This is a documentation-only fix. The
`write_file` tool's `entity_id` parameter and behaviour are already correct.

#### A2 — Make `edit_file` worktree-aware

**What:** Add an optional `entity_id` parameter to `edit_file`. When provided, the
tool resolves the entity's worktree path (as `write_file` already does) and applies
edits relative to that worktree rather than the main repo root. When omitted,
behaviour is unchanged — writes go to the main repo root.

**Why:** `edit_file` has capabilities `write_file` does not: granular old_text/new_text
edits, multi-edit calls, and fuzzy matching. Agents naturally reach for `edit_file`
for modifications and `write_file` for creation. Currently `edit_file` silently
writes to the wrong location when development is in a worktree, which is worse than
an outright error. Making it worktree-aware eliminates the silent-failure mode and
preserves the tool's editing capabilities for worktree workflows.

**Interface contract:**
- Input: `entity_id` (optional string) — resolves to worktree path when present
- Input: `path` (required) — resolved relative to worktree root when `entity_id` set
- Behaviour: identical to current `edit_file` but operating on the resolved worktree path
- Error: if `entity_id` is provided but no active worktree exists, return a clear error

#### A3 — Fix heredoc support in terminal tool

**What:** Configure the terminal tool's shell to support heredoc syntax (`<< 'EOF'`
and `<< EOF`). If the `sh` shell cannot support heredocs, switch to `bash` or
document the limitation explicitly in the `implement-task` skill.

**Why:** Five separate reports document heredoc failure in the terminal tool. Even
though A1 makes `write_file` the primary pattern, heredocs remain useful for ad-hoc
multi-line operations (YAML patches, test data generation, inline scripts). The
terminal tool is the fallback when structured tools fail — it should support basic
POSIX shell features.

**Interface contract:** No code change if the fix is switching the shell binary. If
heredoc support cannot be added, the terminal tool description should state the
limitation.

### Workstream B: Fix Tool Correctness and Reliability

**Components affected:** `entity` tool (list filter), `finish` tool (state propagation),
`edit_file` tool (multi-edit atomicity), `decompose` tool (AC format recognition).

#### B1 — Fix `entity list parent_feature` filter

**What:** The `parent_feature` filter on `entity(action: list, type: task)` currently
returns all tasks in the project (614 total) regardless of the filter value. Fix the
filter to return only tasks whose parent feature matches the provided ID.

**Why:** This is the most functionally impactful correctness bug found in the retro
batch. Every review and orchestration workflow queries tasks by parent feature.
Silently returning all tasks produces results that look plausible (large numbers)
but are completely wrong. Retro-report-5 documents the impact: the reviewer had to
fall back to grepping `.kbz/state/tasks/` by slug convention.

**Interface contract:**
- Input: `parent_feature` (string) — filters to tasks owned by this feature
- Output: only tasks with `parent_feature` matching the provided ID
- Behaviour: unchanged for other filter combinations

#### B2 — Unify `finish()` state propagation

**What:** Ensure `finish()` uses the same write-through path for both the entity
record and the cache/index. The inconsistency — `entity get` returns `done` but
`entity list` and gate checks see `active` — suggests two code paths that should
be unified.

**Why:** Retro-report-1 documents a case where `finish()` returned success, `entity
get` confirmed `done`, but the feature transition gate saw the task as `active`.
This required two override transitions to work around. A `finish()` call that
returns success must guarantee the task is visible as terminal to all subsequent
operations.

**Interface contract:**
- After `finish(task_id, status: "done")` returns success, all subsequent reads
  (`entity get`, `entity list`, gate checks, `status`) must observe the task as
  `done` within the same request context.
- Write-through to both the entity YAML store and the SQLite cache must happen
  synchronously before the response is returned.

#### B3 — Make multi-edit calls atomic

**What:** When a multi-edit `edit_file` call contains multiple `old_text`/`new_text`
pairs, apply all edits or none. If any `old_text` pattern fails to match, fail the
entire call with a clear error identifying which pattern(s) failed.

**Why:** Retro-report-1 documents silent partial application: matching edits were
applied while non-matching ones were skipped, including deletions without
replacements. This corrupted a dev-plan file requiring `git checkout` to restore.

**Interface contract:**
- Pre-flight: attempt to match all `old_text` patterns before applying any edits
- On success: apply all edits sequentially
- On failure: apply none; error message lists which patterns failed to match

#### B4 — Expand decompose AC format recognition

**What:** Extend `decompose(action: propose)` to recognise acceptance criteria in
these additional formats:
- Heading-based ACs: `### AC-NNN` or `### AC-NNN: description`
- Bold with parenthetical: `**AC-NNN (REQ-NNN):** description`
- Given/When/Then blocks: lines beginning with `**Given**`, `**When**`, `**Then**`
- Numbered lines under an "Acceptance Criteria" heading

When parsing fails, emit a diagnostic showing the closest matching patterns found
in the document, with a before/after example of the expected format.

**Why:** Four reports (B38, P38, B36 retro-report-1, retro-report-2) document
cumulative planning-phase effort of ~30% consumed by AC format reformatting.
The workaround (edit specs to checklist format, re-approve, retry) is mechanical
but time-consuming. The tool should accept the formats spec authors naturally use.

**Interface contract:**
- Input: unchanged (spec document ID)
- Output: task proposal generated from ACs in any recognised format
- Error on no match: diagnostic listing closest candidates and expected format

### Workstream C: Fix Document Ownership and Lifecycle

**Components affected:** `doc` tool (register, refresh, approve), `merge` tool
(check, execute).

#### C1 — Auto-infer document owner from path context

**What:** When registering a document with `doc(action: register, path: ...)`,
infer the owner from the file path. A document at
`work/B38-plans-and-batches/B38-spec-f2.md` should default to the feature that
is a child of B38. If a document at the same path is already registered under a
different owner, warn: "This path is already registered under PROJECT/. Did you
mean owner: FEAT-xxx?"

**Why:** P38 retro report documents 3+ re-registration cycles per batch caused by
documents defaulting to `PROJECT/` ownership when registered inside a plan folder.
This blocks `decompose` and feature-level document lookups.

**Interface contract:**
- Path analysis: extract batch/plan slug and feature slug from path components
- Owner resolution: look up the matching entity; if found, use it as owner
- Fallback: if no entity matches, default to `PROJECT/` (current behaviour)
- Warning: if path is already registered under a different owner, surface the conflict

#### C2 — Preserve approval status on minor doc edits

**What:** `doc(action: refresh)` should not reset approval status from `approved`
to `draft` when only minor edits are detected. At minimum, warn explicitly before
resetting: "Refreshing will reset approval status from approved to draft. Continue?"

**Why:** The format-fixing cycle during AC format remediation (edit spec → refresh
→ re-approve → retry decompose) amplified planning friction. The human already
approved the document content; minor formatting edits don't invalidate that approval.

**Interface contract:**
- Detect edit scope: if only formatting/whitespace changed, preserve approval
- If substantive changes detected, warn before resetting
- `auto_approve: true` on register should be sufficient for minor edits

#### C3 — Add `bypassable` field to merge gate results

**What:** Add a `bypassable: bool` field to each gate result in
`merge(action: check)` output. Hard gates (e.g., `review_report_exists`) cannot be
overridden; soft gates can. This lets the caller know before attempting `execute`
which gates require preparation vs. which can be overridden.

**Why:** Retro-report-5 documents discovering the hard/soft distinction only at
execution time: `merge check` said the gate failed, but only `merge execute` revealed
it couldn't be bypassed with `override: true`.

**Interface contract:**
- Output: each gate result gains `bypassable: bool`
- Hard gates: cannot be bypassed with `override: true`
- Behaviour: unchanged for `merge(action: execute)`

#### C4 — Add verification parameter to entity update

**What:** Add `verification` and `verification_status` parameters to
`entity(action: update)`. Merge gates require these fields but they are only
settable through the `finish` task flow. Features that need re-verification after
a rebase (not a normal task completion) have no way to record it.

**Why:** Retro-report-5 and retro-report-13 both document being forced to override
merge gates because verification fields couldn't be set outside task completion.

**Interface contract:**
- New optional parameters on `entity(action: update)`: `verification` (string),
  `verification_status` (string: "passed" | "failed")
- When provided, set directly on the entity record
- No side effects (no lifecycle transition triggered)

### Workstream D: Worktree and Cleanup Automation

**Components affected:** `worktree` tool, `cleanup` tool, `merge` tool.

#### D1 — Add `worktree(action: gc)` for orphaned records

**What:** When `.kbz/state/worktrees/WT-XXX.yaml` exists but the git worktree
directory and `.git/worktrees/WT-XXX/` have both been removed, the record is
orphaned. A `worktree(action: gc)` detects and removes these in bulk, with a
`dry_run` preview mode.

**Why:** Retro-report-12 documents 60+ health errors from orphaned worktree records
created by manual directory deletion. `worktree(action: remove)` fails with a
git-level "not a working tree" error. The cleanup tool only handles merged/abandoned
worktrees with intact directories.

**Interface contract:**
- `dry_run: true` — list orphaned records without removing them
- `dry_run: false` (default) — remove orphaned records from the state store
- Detection: state file exists AND git worktree directory does not exist

#### D2 — Skip drift alerts for merged branches

**What:** When a worktree record has a `merged_at` timestamp, skip branch drift
alerts. The branch is intentionally stale — it's a completed feature whose squash
merge left a local tracking branch behind.

**Why:** Retro-report-12 documents squash-merged branches showing as "321 commits
behind main" with critical drift alerts, even though their worktree records
correctly show `merged_at`. The branch health check doesn't account for
squash-merge semantics.

**Interface contract:**
- Check: if `worktree.merged_at` is non-nil, suppress drift alert
- Behaviour: unchanged for unmerged branches

#### D3 — Auto-schedule cleanup on merge

**What:** `merge(action: execute)` should automatically schedule a `cleanup`
operation for the merged worktree rather than leaving it for the next human to
notice 60+ stale records. Also auto-delete the remote tracking branch when the
merge succeeds.

**Why:** The branch audit retro-report found six stale remote branches from features
merged weeks ago, and the `AutoDeleteRemoteBranch` config option exists but may
not default to true. Making cleanup automatic closes the loop on merge.

**Interface contract:**
- On merge success: call `cleanup(action: execute, worktree_id: ...)` for the
  merged worktree
- On merge success: delete the remote tracking branch if `AutoDeleteRemoteBranch`
  is true (default to true)
- On squash merge: also delete the local tracking branch to prevent drift alerts

#### D4 — Validate entity IDs at worktree creation

**What:** `worktree(action: create)` should reject entity IDs that match the
display-ID format (with embedded segment hyphen, e.g., `FEAT-01KQ7-JDT511BZ`)
rather than the canonical ULID form (`FEAT-01KQ7JDT511BZ`).

**Why:** Retro-report-5 documents a ghost worktree (WT-01KQEX7T3X5XD) created with
a display-format ID. The resulting record points to a non-existent entity and can't
be removed through normal workflow tools.

**Interface contract:**
- Validation: entity ID must match canonical format (no embedded hyphen after the
  type prefix)
- Error: "entity_id 'FEAT-01KQ7-JDT511BZ' appears to be a display ID. Use the
  canonical form FEAT-01KQ7JDT511BZ instead."

---

## Alternatives Considered

### Alternative 1: One monolithic batch for all fixes

Package all 14 features into a single large batch.

**Trade-offs:**
- Easier: one design, one spec cycle, one batch review
- Harder: serialises work that could be parallel; large blast radius if scope
  grows; 14 features would require 50+ tasks in a single batch

**Rejected** because the workstreams are independently deliverable and have
different urgency profiles. Workstream A (worktree editing) is critical and
should ship first; Workstream D (cleanup automation) is lower priority.

### Alternative 2: Fix only the critical-severity items

Limit scope to Themes 1 (worktree editing) and 3 (state consistency bugs).

**Trade-offs:**
- Easier: smallest possible batch, fastest to deliver
- Harder: leaves significant friction (decompose format, document ownership,
  worktree cleanup) unaddressed; guarantees another retro cycle produces the
  same findings

**Rejected** because the moderate-severity items (decompose format, document
ownership) have high impact-to-effort ratios and fixing them alongside the
critical items amortises the design/spec overhead.

### Alternative 3: Status quo — fix nothing, let agents adapt

Accept the friction as inherent to the system and rely on agent adaptation.

**Trade-offs:**
- Easier: zero implementation effort
- Harder: every future batch pays the same friction tax; agent adaptation means
  working around tool limitations rather than using tools as designed; the
  accumulated cost across 10+ future batches far exceeds the one-time fix cost

**Rejected** because the collated feedback report provides clear evidence that
these are systematic tooling gaps, not one-time adaptation costs. The fixes are
well-understood and low-risk.

---

## Decisions

**Decision 1: Organise fixes into four independent workstreams**

**Context:** 16 themes with 85 findings. Some fixes are documentation-only (A1);
others require code changes (B1–B4, C1–C4, D1–D4). Some are critical (A1–A3);
others are moderate (C3, D2).

**Rationale:** Independent workstreams enable parallel implementation and
staggered delivery. Workstream A (worktree experience) is the highest-ROI and
should ship first. Workstream B (tool correctness) contains the most impactful
bug fixes. Workstream C (document ownership) resolves the second most common
friction cluster. Workstream D (cleanup automation) is the lowest urgency.

**Consequences:** Each workstream becomes its own batch within P40. Features
within a workstream can be implemented in parallel where file scopes are disjoint.
The workstream structure also maps naturally to the reporting: each workstream
has clear success metrics from the retro findings it addresses.

**Decision 2: Prioritise documentation fixes over code changes for worktree editing**

**Context:** A1 (document `write_file(entity_id)`) is a skill-file edit. A2
(make `edit_file` worktree-aware) is a code change in the MCP server.

**Rationale:** A1 fixes 80% of the problem with 5% of the effort. Agents who
know about `write_file(entity_id)` consistently report it works well. The
remaining 20% (granular edits in worktrees) is addressed by A2. Shipping A1
first and A2 second follows the "simplest thing that could possibly work"
principle.

**Consequences:** A1 delivers immediate value before A2 is complete. The risk
is that A1 alone isn't sufficient — some editing operations genuinely need the
granularity of `edit_file`. A2 closes that gap.

**Decision 3: Do not add new tools — extend existing tools**

**Context:** Several suggested fixes could be implemented as new tools
(`doc(action: infer_path)`, `bug(action: close)`, `worktree(action: gc)`).

**Rationale:** New tools increase the tool count (already at 50+), add
discoverability burden, and require new documentation. Extending existing tools'
parameter surfaces preserves the mental model agents already have. The exception
is `worktree(action: gc)` — garbage collection is a genuinely new operation with
no natural home in an existing action.

**Consequences:** The `entity`, `doc`, `edit_file`, `merge`, and `worktree` tools
gain new parameters and actions. The tool count increases by at most 1 (if
`worktree(action: gc)` is added). Existing documentation is amended rather than
new documentation written.

**Decision 4: Defer knowledge graph and context assembly improvements**

**Context:** Theme 12 (knowledge signal-to-noise) and the context budget issue
(30K bytes at 99% utilisation) are real friction points but require deeper
architectural changes to the context assembly pipeline.

**Rationale:** These are design questions (how should knowledge be scoped? what
should context budgets be per role?) that need their own design cycle. The
quick wins in this design (A1–A3, B1–B4, C1–C4, D1–D4) can ship without
answering those questions. Including them would expand scope beyond what can be
designed and implemented in a focused batch.

**Consequences:** Theme 12 remains as open follow-up work. The context budget
and knowledge deduplication questions should be addressed in a separate design
cycle, possibly as a P41 plan.

---

## Scope Boundaries

**In scope:**
- Worktree file-editing discoverability and tool support
- Tool correctness bugs (parent_feature filter, finish state propagation,
  edit_file atomicity, decompose AC format)
- Document ownership inference and lifecycle ergonomics
- Worktree cleanup automation
- Merge gate UX improvements
- Verification field settability

**Out of scope:**
- Knowledge graph indexing and context assembly budget changes (needs separate design)
- Bug lifecycle fast-forward (`advance` for bugs) — documented as follow-up
- CLI/MCP health-check unification — transitional, self-resolving post-migration
- Cross-feature dependency visibility in handoff context — needs research
- Test infrastructure (flake fixes, spec-assertion guidance)
- Stale binary startup self-check (server_info tool already addresses the diagnostic gap)
- Session continuity auto-commit (needs research on safe auto-commit boundaries)
- New entity statuses or lifecycle states
- Batch `active → done` shortcut (needs separate discussion on gate semantics)
