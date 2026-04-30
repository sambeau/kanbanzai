# Retrospective Report: P38 Plans and Batches Implementation

| Field    | Value                           |
|----------|---------------------------------|
| Date     | 2026-04-28                      |
| Author   | AI Agent (Claude via Kanbanzai) |
| Scope    | P38 Plans and Batches (F1–F8)   |

---

## Summary

P38 was the most ambitious feature set I've worked on in Kanbanzai: introducing a
recursive strategic plan entity alongside renaming the existing plan to batch. It
spanned 8 features, involved ~30+ Go source files and ~20 documentation files, and
required deep coordination across model, validation, storage, service, MCP tool, and
documentation layers.

The overall experience was positive — the system held up well under sustained load,
and the tooling (entity, doc, decompose, status) provided reliable workflow support.
Key friction points clustered around document ownership, task lifecycle semantics,
and sub-agent worktree coordination.

---

## What Went Well

### 1. MCP Server Reliability

The Kanbanzai MCP server performed excellently throughout. All entity CRUD operations,
document registration, lifecycle transitions, and status queries worked consistently
across ~100+ tool calls. No server crashes, timeouts, or data corruption. The system
scaled cleanly from a single feature to 8 concurrent features.

### 2. Spec/Dev-Plan Pipeline

The `specifying → dev-planning → developing` pipeline provided a clear separation of
concerns. Writing specifications forced me to think through requirements before
writing code. The structure validation scripts (`.kbz/skills/write-spec/scripts/`)
caught formatting issues early. The doc approval mechanism provided clear gates.

### 3. Sub-Agent Parallelism

For the dev-planning and implementation phases, dispatching 6 parallel sub-agents
worked remarkably well. Each sub-agent operated independently on a feature's dev-plan
or code changes with no cross-contamination. The worktree system isolated changes
correctly. This was the single biggest productivity multiplier in the project.

### 4. Backward Compatibility Design

The decision to keep deprecated aliases (`EntityKindPlan = EntityKindBatch`,
`PlanStatus = BatchStatus`, `ComputePlanRollup` as wrapper) meant I could refactor
incrementally without breaking every caller simultaneously. This reduced risk
substantially.

### 5. Status Dashboard as Single Source of Truth

The `status()` tool was invaluable for tracking progress. At any point I could call
it and see exactly which features were in which state, how many tasks were done, and
whether the build was healthy. This replaced the mental tracking that typically
consumes significant orchestrator context.

---

## What Didn't Go Well

### 1. Document Ownership Confusion (Critical)

The single biggest recurring problem was documents registered under `PROJECT/` scope
vs feature-owned scope. When I registered specs in the P38 work folder using
`doc(action: "register", path: "...")`, they defaulted to `PROJECT/` ownership.
Later, when `decompose(action: "propose")` tried to find the spec by feature ID, the
lookup failed because the spec wasn't owned by the feature.

I re-registered the same documents multiple times as I discovered this mismatch.
Each re-registration created a new document record in a different ownership scope,
cluttering the document store.

**Impact:** 3+ re-registration cycles, blocked decompose proposals, confusion about
which document ID to reference.

**Suggestions:**
- When registering a document whose path is inside a feature's parent plan folder,
  auto-detect the owner from context rather than defaulting to PROJECT/.
- If a document at the same path is already registered under a different owner,
  surface a warning during register: "This path is already registered under a
  different owner. Did you mean to use owner: FEAT-xxx?"
- The decompose tool should fall back to searching PROJECT/ docs when feature-owned
  docs are not found, since both can satisfy the gate.

### 2. Decompose AC Format Requirements (Significant)

The `decompose(action: "propose")` tool requires acceptance criteria in a very specific
format (`**AC-NN.**` — bold identifier ending with period). My initial specs used
other formats (`**AC-NN (REQ-xxx):**`, which silently failed. The error message
("no acceptance criteria found") was unhelpful — it listed the sections it found but
gave no indication of *why* no criteria matched.

**Impact:** 3 failed decompose attempts across 6 features, requiring sed rewrites
of all 6 spec files and re-approval cycles.

**Suggestions:**
- Add a "nearest match" diagnostic: when no criteria are found, show the closest
  patterns in the document that might be acceptance criteria but don't match the
  expected format.
- Consider also matching simpler formats: numbered lines, checkbox items, lines
  containing "Given/When/Then".
- Document the expected AC format in the write-spec skill more prominently, perhaps
  with a "AC Format Reference" section at the top.

### 3. Task Lifecycle: queued vs ready vs active vs done (Moderate)

The task lifecycle requires `queued → ready → active → done` via `next()` to claim,
but I frequently called `finish()` directly on queued/ready tasks and got errors.
The sub-agents also skipped the `next()` claim step and used `finish()` directly,
which silently marked tasks as done without proper lifecycle tracking.

**Impact:** Wasted tool calls, inconsistent task state, unclear which tasks were
actually "claimed" by which agent.

**Suggestions:**
- Allow `finish()` to work from any non-terminal state with an implicit advance,
  recording the transition in the task's override history. The current requirement
  of `queued → ready → active → done` is rigid for single-agent workflows.
- Provide a `finish_batch` action that completes multiple tasks at once, handling
  all the intermediate transitions automatically.

### 4. Sub-Agent Worktree Coordination (Moderate)

When sub-agents modified files in their worktrees, the changes weren't automatically
visible on main. I had to manually `git add`, `git commit`, and `git merge` from each
worktree branch. One sub-agent's worktree changes also appeared to overwrite changes
from a prior sub-agent (F2's entity_tool.go changes were clobbered by F6's changes).

**Impact:** Manual merge resolution, extra tool calls to `git status`/`git merge`,
one instance of worktree file overwrite.

**Suggestions:**
- Provide a `consolidate` or `merge` tool action that merges a feature's worktree
  branch into main with conflict detection.
- The `conflict` tool should detect when two features share modified files and warn
  before dispatching parallel sub-agents.
- Auto-commit uncommitted changes in worktrees before merging — the current state
  leaves dirty worktrees that require manual cleanup.

### 5. Feature Display ID Confusion (Minor)

The feature display IDs shown in entity data use `P38-F2` etc. even after the B{n}
rename. This is because the entity records still reference their parent batch by the
old P{n} ID in some internal state that wasn't migrated. When I ran `status()`, the
display IDs showed `P38-F2` instead of `B38-F2`.

**Impact:** Visual inconsistency — the dashboard says "Batch B38" but features show
"P38-F2".

**Suggestions:**
- Rebuild the display ID cache after migration.
- The health check system should flag this inconsistency automatically.

---

## Opportunities for Improvement

1. **Document Owner Inference:** The doc tool should auto-detect the correct owner
   from file path context. A spec at `work/B38-plans-and-batches/B38-spec-f2.md`
   should automatically get `owner: FEAT-01KQ7YQKBDNAP` if that feature is a child
   of B38. Or at minimum, warn when owner doesn't match path conventions.

2. **Batch Finish for Tasks:** The orchestrator role (and single-agent workflows)
   need a way to mark multiple tasks done simultaneously. The current per-task
   `finish()` is too granular for features with 3-5 tasks.

3. **Decompose Feedback Quality:** When decompose fails to parse a spec, the error
   should suggest the fix, not just state the failure. "No acceptance criteria found.
   Expected format: `**AC-NN.** description`. Found: 14 sections including
   'Acceptance Criteria' but no bold-identifier lines."

4. **Retro Knowledge Integration:** I didn't use retro knowledge because there were
   no P38 signals, but the system should have asked me to record observations *as I
   encountered friction*. A post-task prompt like "You completed TASK-xxx. Any
   friction to record?" would capture signals at the source.

5. **Worktree Cleanup Automation:** After merging a feature branch, the worktree
   should be automatically scheduled for cleanup. The `cleanup(action: "list")`
   tool showed many stale worktrees but I didn't use it proactively.

---

## Conclusion

The Plans and Batches implementation validated the Kanbanzai workflow system at scale.
The core tools (entity, doc, decompose, status) worked reliably. The main friction
points were around document ownership semantics, decompose's AC format requirements,
and sub-agent worktree coordination. None were blockers — all were worked around —
but each added 2-3 wasted tool calls per feature.

The sub-agent parallelism model is the system's strongest feature for scale. With
better document ownership inference and worktree coordination, an 8-feature migration
like this could run with near-zero orchestrator intervention.
