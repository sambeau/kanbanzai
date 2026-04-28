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

### 2. Atomic document file operations (Create, Move, Delete)

**Category:** Atomicity
**Complexity:** Medium
**Risk:** Low

Every document has two representations: a file on disk (`work/**/*.md`) and a metadata
record in `.kbz/state/documents/`. Today, every file operation that touches a registered
document requires multiple manual steps with no transactional guarantee. This
recommendation covers the full CRUD surface — Create, Move, Delete — as a unified
pattern. (Read is already safe; Update is addressed by Recommendation 4.)

#### Foundation: `CommitStateAndPaths`

Introduce a `CommitStateAndPaths(repoRoot, message string, extraPaths ...string)` function
that stages `.kbz/state/` **plus** specified additional paths in a single commit. All three
operations below use this primitive.

#### 2a. Atomic Create (`doc register`)

**Problem.** Creating a document requires: write file → `doc(register)` → manually `git add`
both paths → `git commit`. If only the state record is committed (via `CommitStateIfDirty`),
the document file is missing from git. If only the file is committed, the registration is
invisible.

**Recommendation.** `doc(action: register)` should call `CommitStateAndPaths` automatically,
passing the document file path. The commit message identifies the operation:

```
workflow(DOC-xxx): register {type} document
```

#### 2b. Atomic Move (`doc move`) — see also Recommendation 9

**Problem.** Moving a registered document requires four manual steps: move the file,
register a new record, handle the orphaned old record, stage and commit both paths. The old
draft record cannot be superseded (supersede requires `approved` status), so it lingers as a
ghost record.

**Recommendation.** Add `doc(action: move, id: "DOC-xxx", new_path: "work/reports/foo.md")`:

1. Move the file on disk
2. Update the existing record's `path` field in-place (preserving ID and approval status)
3. Optionally update `type` if the new path implies a different document type
4. Recompute the content hash
5. Commit via `CommitStateAndPaths` with the old path (deletion), new path, and state record

#### 2c. Atomic Delete (`doc delete`)

**Problem.** There is no `doc` action for removing a document. Today, deleting a registered
document requires: manually delete the file → manually retire the record (if it's even
possible — `retire` may not exist) → stage the deletion and state change → commit. If the
agent deletes the file but forgets the record, `health` reports a dangling reference. If the
record is removed but the file remains, the document becomes invisible to the workflow system
but still exists in the repository.

**Recommendation.** Add `doc(action: delete, id: "DOC-xxx")` that:

1. Verifies the document is not `approved` (or requires a `force: true` flag if it is, to
   prevent accidental deletion of documents that downstream entities depend on)
2. Removes the file from disk
3. Removes the state record from `.kbz/state/documents/`
4. Commits the file deletion and record removal atomically via `CommitStateAndPaths`

The commit message identifies the operation:

```
workflow(DOC-xxx): delete {type} document
```

If the document is owned by an entity, the entity's document reference field should be
cleared as part of the same transaction.

#### Safety argument (all three operations)

- The extra paths are explicitly provided by the tool, not discovered by glob — no risk
  of accidentally staging unrelated files.
- The document path is already known to each action (required parameter or stored on the
  record) — no path guessing.
- If the commit fails, all changes remain uncommitted but consistent on disk — the agent
  can retry or commit manually.
- Delete requires the document to be non-approved by default, preventing accidental removal
  of documents that gate downstream transitions.
- Move preserves document identity (ID, approval status, cross-references), avoiding the
  register-new / supersede-old dance.

---

### 3. Auto-approve agent-authored documents and decouple dev-plan approval from implementation trigger

**Category:** Rule Enforcement + Atomicity
**Complexity:** Low–Medium
**Risk:** Low

#### Problem: two concerns are conflated

Today, approving a dev-plan document does two things at once:

1. Marks the document as approved (quality judgement on the plan)
2. Cascades the feature from `dev-planning` → `developing` (triggers implementation)

These are separate concerns with different ownership:

- **Plan quality** — an implementation decision. Agents author dev-plans; agents can
  assess whether a plan is well-structured, traceable to the spec, and has a sound
  dependency graph. No human design or product manager input is needed.
- **Implementation trigger** — a scheduling decision. Humans may need to consider timing,
  resource availability, priorities across features, or dependencies on external work
  before parallel implementation begins.

The current coupling means either: (a) agents cannot approve their own plans without
accidentally starting implementation, or (b) humans must review implementation details
they don't need to own just to control scheduling.

#### Recommendation: decouple approval from transition

**3a. Allow agent approval of dev-plans without cascading.**

Remove the automatic `dev-planning → developing` cascade from `doc(action: approve)` when
the document type is `dev-plan`. Instead, dev-plan approval should:

1. Mark the document as `approved`
2. **Not** trigger the entity lifecycle cascade

This allows the agent to approve its own plan and proceed with decomposition (creating
tasks, wiring dependencies) while the feature remains in `dev-planning`.

The cascade currently lives in `documents.go` L424:

```
case entityType == "feature" && doc.Type == model.DocumentTypeDevPlan:
    targetStatus = "developing"
```

This line should be removed or gated behind a flag. The transition to `developing` becomes
an explicit human-triggered action (see 3b).

**3b. Require explicit human signal to begin implementation.**

The transition from `dev-planning` → `developing` should require an explicit
`entity(action: transition, id: "FEAT-xxx", status: "developing")` call, which is the
natural place for the human gate. The `developing` stage's prerequisites already require
an approved dev-plan and at least one child task — these checks remain enforced. The human
simply says "go" when scheduling permits, and the agent records the transition.

This preserves the `human_gate: true` intent from `stage-bindings.yaml` while moving the
gate to the right operation: the *transition*, not the *document approval*.

**3c. Auto-approve agent-authored documents via `auto_approve` flag.**

Add an `auto_approve` option to `doc(action: register)`. When `auto_approve: true`:

1. Register the document (write state record)
2. Immediately approve it (set status to `approved`)
3. Fire entity cascade only for document types where cascading is safe
4. Commit the combined state atomically

Scope:

| Document Type | Auto-approve safe? | Cascade on approval? | Rationale |
|---------------|-------------------|---------------------|-----------|
| `dev-plan` | ✅ Yes | ❌ No (decoupled per 3a) | Implementation detail, agent-authored; human gate moves to the transition |
| `research` | ✅ Yes | N/A (no cascade defined) | Informational; `researching` stage has `human_gate: false` |
| `report` | ✅ Yes | N/A (no cascade defined) | Review output, agent-authored |
| `design` | ❌ No | ✅ Yes | Requires human architectural judgement |
| `specification` | ❌ No | ✅ Yes | Requires human product-owner approval |

#### Safety argument

- **Decoupling is strictly safer than the status quo.** Today, an agent that calls
  `doc(approve)` on a dev-plan inadvertently triggers implementation. After decoupling,
  approving a plan is a low-stakes quality signal; starting implementation is a separate,
  intentional act.
- The `developing` stage prerequisites (approved dev-plan + child tasks) remain enforced
  on the transition — decoupling does not weaken the gate, it moves it.
- Decomposition can occur during `dev-planning` without requiring `developing` status.
  The `decompose` tool reads the feature's spec, not its lifecycle state. Tasks created
  during `dev-planning` are ready to execute the moment the feature transitions.
- The `auto_approve` flag is opt-in per call. Agents and orchestrator skills decide when
  to use it based on document type and stage context.
- Documents requiring human judgement (`design`, `specification`) remain excluded.

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

### 9. [Consolidated into Recommendation 2b — Atomic document move]

See **Recommendation 2b** above. Originally a standalone recommendation; now part of the
unified document file operations family.

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
| 6 | 3 | Decouple dev-plan approval + auto-approve agent docs | Low–Med | High |
| 7 | 2 | Atomic document file ops (create, move, delete) | Medium | High |
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
