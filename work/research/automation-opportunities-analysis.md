# Automation Opportunities Analysis

> "Knowledge without automation decays. Encode your principles into tools that enforce them automatically."
> — The Toolkit Principle

## Executive Summary

This report analyses the Kanbanzai workflow system to identify knowledge currently encoded as
documentation or convention that could be safely automated. Every recommendation targets one
of two properties: **rule enforcement** (making violations impossible rather than merely
discouraged) or **atomicity** (ensuring related state changes succeed or fail together).

The analysis is conservative. Recommendations are limited to automations that are:

- **Safe** — no risk of data loss, no ambiguous decision-making
- **Deterministic** — the correct action can be derived from system state alone
- **Autonomous** — no human or design-manager input required (agent input acceptable where noted)

Recommendations requiring product-owner judgement, scope decisions, or design approval are
explicitly excluded.

---

## Current Architecture: How State Flows Today

Understanding the automation gap requires understanding the current commit topology.

### The single commit point

Every MCP tool (`entity`, `doc`, `finish`, `decompose`, etc.) writes YAML files to
`.kbz/state/` but **none auto-commits**. The only production commit point is
`git.CommitStateIfDirty()`, called exclusively by the `handoff` tool before sub-agent
dispatch. This means:

1. State changes accumulate as dirty files across an entire orchestration session.
2. If the session ends without a `handoff` call (direct implementation, session crash,
   forgotten commit), **all state changes are orphaned**.
3. The `AGENTS.md` pre-task checklist tells agents to check for and commit orphaned
   `.kbz/state/` files — a manual recovery step for an automated system's gap.

### The document–state split

Document files (`work/**/*.md`) and their metadata records (`.kbz/state/documents/`) live in
different directory trees. Creating a document requires: write file → `doc(register)` → manual
`git add` + `git commit` of both paths. These three steps have no transactional guarantee.

### The cascade gap

`doc(action: approve)` triggers entity lifecycle cascades (e.g., approving a dev-plan
transitions the feature to `developing`). This writes **both** the document record and the
entity record — but commits neither. The cascade is invisible to git until something
eventually commits `.kbz/state/`.

---

## Recommendations

### 1. Auto-commit after terminal workflow operations

**Category:** Atomicity + Rule Enforcement
**Complexity:** Low
**Risk:** Very Low

#### Problem

Only `handoff` commits state. Many workflow paths never call `handoff`:

- Direct implementation (agent works without sub-agents)
- Task completion (`finish`)
- Decomposition (`decompose apply`)
- Document approval (`doc approve`)
- Merge completion (`merge execute`)

In all these cases, state files are written to disk but left uncommitted. An agent must
remember to commit manually — or the next `handoff` bundles them into an unrelated commit
with a generic message.

#### Recommendation

Add `CommitStateIfDirty()` calls at natural transaction boundaries — operations that
represent the completion of a logical unit of work:

| Tool | Action | Commit message |
|------|--------|---------------|
| `finish` | complete | `workflow(TASK-xxx): complete task` |
| `decompose` | apply | `workflow(FEAT-xxx): create N tasks from decomposition` |
| `doc` | approve | `workflow(DOC-xxx): approve {type} document` |
| `doc` | register | `workflow(DOC-xxx): register {type} document` |
| `merge` | execute | `workflow(FEAT-xxx): mark worktree merged` |
| `entity` | transition | `workflow(FEAT-xxx): transition {from} → {to}` |

Each commit should use a descriptive message (not the current generic `chore(kbz)` message)
so that `git log` becomes a readable workflow audit trail.

#### Safety argument

- `CommitStateIfDirty` only stages files under `.kbz/state/` — it cannot accidentally
  commit code, documents, or other work-in-progress.
- It is a no-op when the state directory is clean (REQ-05 already guarantees this).
- The existing `handoff` usage proves the pattern is safe in production.
- Making commits more frequent and more granular strictly improves recoverability.

#### Design consideration

The commit message should be parameterised. Extend `CommitStateIfDirty` (or add a sibling
function `CommitStateWithMessage(repoRoot, message string)`) that accepts a formatted message.
The existing `handoff` call site can continue using its fixed message.

---

### 2. Atomic document + state commits

**Category:** Atomicity
**Complexity:** Medium
**Risk:** Low

#### Problem

When an agent creates a document, three things must happen:

1. Write the `.md` file to `work/{type}/`
2. Call `doc(action: register)` which writes `.kbz/state/documents/`
3. Commit both the file and the state record

Today, step 3 must be done manually, and the agent must remember to `git add` both paths.
If only the state record is committed (via `CommitStateIfDirty`), the document file is
missing from git. If only the file is committed, the registration is invisible.

#### Recommendation

Introduce a `CommitStateAndPaths(repoRoot, message string, extraPaths ...string)` function
that stages `.kbz/state/` **plus** specified additional paths in a single commit. The `doc
register` action should call this automatically, passing the document file path.

Similarly, `doc approve` should commit the state changes (document record + any cascaded
entity transitions) in a single atomic commit.

#### Safety argument

- The extra paths are explicitly provided by the tool, not discovered by glob — no risk
  of accidentally staging unrelated files.
- The document path is already known to the `register` action (it is a required parameter).
- If the commit fails, both the file and the record remain uncommitted — consistent state.

---

### 3. Auto-approve implementation documents (dev-plans)

**Category:** Rule Enforcement
**Complexity:** Low
**Risk:** Low

#### Problem

Dev-plans are implementation documents. The stage-bindings declare `human_gate: true` for
`dev-planning`, but the gate exists to review task decomposition — not the document format.
In practice, the agent writes the plan, registers it, then must remember to call
`doc(action: approve)` before decomposition can begin. The cascade from dev-plan approval to
entity transition (`dev-planning` → `developing`) is already automated server-side.

The manual approval step adds friction without adding safety: agents are the authors of
dev-plans, and the system already validates the transition prerequisites.

#### Recommendation

Add an `auto_approve` option to `doc(action: register)`. When `auto_approve: true`:

1. Register the document (write state record)
2. Immediately approve it (set status to `approved`, fire entity cascade)
3. Commit the combined state atomically

This should be **opt-in per call**, not a blanket policy. The agent (or orchestrator skill)
decides when auto-approval is appropriate. The stage-bindings `human_gate` flag remains
advisory for the overall stage — the automation applies only to the document approval step.

Scope this to document types where agent authorship is sufficient:

| Document Type | Auto-approve safe? | Rationale |
|---------------|-------------------|-----------|
| `dev-plan` | ✅ Yes | Implementation detail, agent-authored |
| `research` | ✅ Yes | Informational, no downstream gates depend on approval |
| `report` | ✅ Yes | Review output, agent-authored |
| `design` | ❌ No | Requires human architectural judgement |
| `specification` | ❌ No | Requires human product-owner approval |

#### Safety argument

- Dev-plan approval already cascades via `EntityLifecycleHook.TransitionStatus()`, which
  validates the transition against the lifecycle state machine. Invalid transitions are
  rejected.
- The existing content-hash verification in `approve` ensures the document hasn't been
  modified between registration and approval (trivially true when they happen in the same
  call).
- Agent-authored implementation documents are the one category where the author and
  approver can legitimately be the same entity.

---

### 4. Auto-refresh content hash on document approval

**Category:** Rule Enforcement
**Complexity:** Very Low
**Risk:** Very Low

#### Problem

If a registered document is edited on disk, its content hash drifts from the stored hash.
The `approve` action checks the hash and rejects approval if it doesn't match. The agent
must remember to call `doc(action: refresh)` before `doc(action: approve)`. If they forget,
approval fails with a confusing error.

Worse: if an *already-approved* document is edited, the approval status remains `approved`
with a stale hash. The system silently diverges from reality until someone explicitly
calls `refresh` (which then demotes the document back to `draft`).

#### Recommendation

Make `doc(action: approve)` automatically recompute the content hash before comparing.
If the hash has changed since registration, update it as part of the approval operation.
This eliminates the separate `refresh` step entirely for the approval path.

For the stale-approved-document problem: the `doc(action: get)` and `doc(action: validate)`
actions should compare the stored hash against the file on disk and include a `hash_current`
boolean in the response. This makes drift visible without requiring explicit refresh calls.

#### Safety argument

- Computing a file hash is a pure read operation — no side effects.
- The approval action already reads the file to verify the hash; recomputing instead of
  comparing against a stale stored value is strictly more correct.
- This removes a class of confusing errors without changing any approval semantics.

---

### 5. Post-finish state commit with structured message

**Category:** Atomicity + Rule Enforcement
**Complexity:** Low
**Risk:** Very Low

#### Problem

`finish` is the most prolific state writer — a single call can produce:

- Task entity record (status transition + completion metadata)
- N knowledge entry files
- N retrospective signal files
- M unblocked task entity files (dependency cascade)

All of these are written to `.kbz/state/` but none are committed. If the agent's session
ends after `finish` but before a manual commit, the task appears incomplete to the next
session despite all state files existing on disk.

#### Recommendation

`finish` should call `CommitStateIfDirty` (or its parameterised variant) after successfully
writing all state. The commit message should identify the completed task:

```
workflow(TASK-xxx): complete – {summary truncated to 50 chars}
```

This is the single highest-value automation in this report because:

1. `finish` is called at the end of every task — it is the most frequent terminal operation.
2. The state it writes is complex (task + knowledge + retro + unblocked dependencies).
3. Forgetting to commit after `finish` is the most common orphaned-state scenario.

#### Safety argument

- `finish` already validates all inputs and writes all state before returning.
  Adding a commit at the end is a pure persistence operation.
- If the commit fails, the state files still exist on disk — the agent can retry or
  commit manually. No data is lost.
- The `handoff` tool already proves that best-effort post-operation commits are safe
  (it logs warnings on failure but does not block).

---

### 6. Post-decompose-apply state commit

**Category:** Atomicity
**Complexity:** Very Low
**Risk:** Very Low

#### Problem

`decompose apply` creates N task entity files in two passes (creation, then dependency
wiring). If only some files are committed (e.g., via an intervening `handoff`), the task
graph is inconsistent — tasks exist but their dependency links are missing.

#### Recommendation

`decompose apply` should commit all created task files atomically after both passes complete:

```
workflow(FEAT-xxx): decompose into N tasks
```

#### Safety argument

- `decompose apply` is already atomic at the service level — both passes complete or the
  operation fails.
- Committing after completion preserves this atomicity in git history.
- The two-pass write pattern (create, then link dependencies) makes it especially important
  that partial state is never committed between passes.

---

### 7. Post-merge worktree state commit

**Category:** Atomicity
**Complexity:** Very Low
**Risk:** Very Low

#### Problem

`merge execute` creates a merge commit for the feature code (via squash/merge/rebase), then
updates the worktree record to `merged` status. The code merge is committed, but the
worktree state update is not. If the agent's session ends here, the worktree record says
`active` while the branch has been merged.

#### Recommendation

After a successful merge, `merge execute` should commit the updated worktree state:

```
workflow(FEAT-xxx): mark worktree merged
```

#### Safety argument

- The merge has already succeeded at this point — the worktree state update is a bookkeeping
  operation.
- The `postMergeInstall` hook already runs after merge as a best-effort side effect.
  A state commit fits the same pattern.

---

### 8. Validate document required sections on registration

**Category:** Rule Enforcement
**Complexity:** Medium
**Risk:** Low

#### Problem

`stage-bindings.yaml` declares required sections for each document type (e.g., "Overview",
"Goals and Non-Goals", "Design", "Alternatives Considered", "Dependencies" for design docs).
These requirements are purely documentary — no tool checks whether a registered document
actually contains these sections. An agent can register and approve an empty file.

#### Recommendation

Add a validation step to `doc(action: register)` that checks whether the document contains
the required sections declared in `stage-bindings.yaml` for its document type. The check
should look for markdown headings (`## Section Name`) matching the required section names.

On failure, return a structured warning (not an error) listing missing sections. The
registration still proceeds — this is a *lint*, not a gate — but the warning gives the
agent immediate feedback to fix the document before requesting approval.

Optionally, `doc(action: approve)` could enforce the check as a hard gate: reject approval
if required sections are missing.

#### Safety argument

- Heading detection is a simple string match against a known list — no ambiguity.
- Warnings on registration (not blocking) preserve flexibility for edge cases while
  surfacing problems early.
- The required sections are already declared in `stage-bindings.yaml` — no new
  configuration is needed.

---

## Recommendations NOT Made

The following automations were considered and rejected:

### Auto-approve designs or specifications

Designs encode architectural decisions. Specifications encode product requirements. Both
require human judgement about correctness, completeness, and alignment with project goals.
Automating their approval would bypass the only meaningful quality gate in the workflow.

### Auto-transition features through multiple stages

The `entity(action: transition, advance: true)` flag already exists for multi-step
advancement, but it checks document prerequisites at each gate. Making this fully
automatic (trigger on document creation) would remove the human's ability to pause
and redirect between stages.

### Enforce commit message format via pre-commit hook

While the commit message format (`type(scope): description`) is unenforced, adding a
pre-commit hook would affect all git operations in the repository — including manual human
commits. This is a repository configuration decision, not a tool automation.

### Auto-detect document edits and trigger refresh

File-system watchers are unreliable across platforms and add complexity. The
recommendation to recompute hashes at approval time (Recommendation 4) solves the
approval-path problem without runtime monitoring.

### Enforce effort budgets

Stage-binding effort budgets ("5-15 tool calls") are advisory heuristics. Enforcing them
would require counting tool calls per stage per agent — complex instrumentation for a
guideline that should remain flexible.

### Auto-assign tasks from the work queue

Task claiming via `next(id)` is intentionally explicit. Auto-assignment would remove the
orchestrator's ability to sequence work based on runtime context (branch state, dependency
readiness, available agent capacity).

---

## Implementation Priority

Ordered by value-to-effort ratio:

| Priority | Rec | Description | Effort | Value |
|----------|-----|-------------|--------|-------|
| 1 | 5 | Post-`finish` state commit | Very Low | Very High |
| 2 | 4 | Auto-refresh hash on approval | Very Low | High |
| 3 | 7 | Post-merge worktree state commit | Very Low | Medium |
| 4 | 6 | Post-decompose state commit | Very Low | Medium |
| 5 | 1 | Auto-commit at all terminal operations | Low | High |
| 6 | 3 | Auto-approve implementation documents | Low | Medium |
| 7 | 2 | Atomic document + state commits | Medium | High |
| 8 | 8 | Validate document required sections | Medium | Medium |

Recommendations 1–4 share a common implementation pattern (call `CommitStateIfDirty` or
its parameterised variant at the end of a handler). They could be implemented together as
a single change to `internal/git/commit.go` plus call-site additions in the relevant MCP
tool handlers.

---

## Appendix: Current Commit Topology

```
Agent session
│
├─ entity(create)          → writes .kbz/state/features/FEAT-xxx.yaml  [uncommitted]
├─ doc(register)           → writes .kbz/state/documents/DOC-xxx.yaml  [uncommitted]
├─ doc(approve)            → writes DOC + FEAT records                  [uncommitted]
├─ decompose(apply)        → writes N task records                      [uncommitted]
│
├─ handoff(task_id)  ←── ONLY COMMIT POINT ──→  "chore(kbz): persist workflow state"
│   └─ sub-agent
│       ├─ implement code
│       ├─ git commit (code)
│       └─ finish(task_id)  → writes task + knowledge + retro           [uncommitted]
│
├─ ... more orchestration ...
│
└─ session ends            → orphaned .kbz/state/ files if no handoff fired
```

### Proposed commit topology

```
Agent session
│
├─ entity(create)          → writes + COMMITS: "workflow(FEAT-xxx): create feature"
├─ doc(register)           → writes + COMMITS: "workflow(DOC-xxx): register design"
├─ doc(approve)            → writes + COMMITS: "workflow(DOC-xxx): approve design"
├─ decompose(apply)        → writes + COMMITS: "workflow(FEAT-xxx): decompose into 6 tasks"
│
├─ handoff(task_id)        → commits if dirty (unchanged behaviour)
│   └─ sub-agent
│       ├─ implement code
│       ├─ git commit (code)
│       └─ finish(task_id)  → writes + COMMITS: "workflow(TASK-xxx): complete"
│
├─ merge(execute)          → code merge commit + state COMMIT
│
└─ session ends            → no orphaned state (already committed at each boundary)
```
