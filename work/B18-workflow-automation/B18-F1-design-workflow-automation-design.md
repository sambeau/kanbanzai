# Workflow State Automation — Design

**Status:** Draft
**Feature:** FEAT-01KN73BFK4M4Z (auto-commit-and-doc-ops)
**Plan:** P18-workflow-automation
**Basis:** [Automation Opportunities Analysis](../reports/automation-opportunities-analysis.md)
**Date:** 2026-04-02

---

## Problem and Motivation

The Kanbanzai MCP server writes workflow state as YAML files under `.kbz/state/` but has
only one production commit point: `git.CommitStateIfDirty()`, called by the `handoff` tool
before sub-agent dispatch. Every other state-mutating tool — `entity`, `doc`, `finish`,
`decompose`, `merge` — writes to disk and leaves files uncommitted. This creates three
categories of problem:

### 1. Orphaned state

If a session ends without a `handoff` call (direct implementation, crash, session timeout,
forgotten commit), all state changes are lost to git. The `AGENTS.md` pre-task checklist
mitigates this by telling agents to check for dirty `.kbz/state/` files at session start —
a manual recovery step for an automated system's gap. This happens frequently enough to
warrant a dedicated checklist item, which means it is not an edge case.

### 2. Document–state inconsistency

Document files (`work/**/*.md`) and their metadata records (`.kbz/state/documents/`) live
in different directory trees. Creating, moving, or deleting a registered document requires
multiple manual steps with no transactional guarantee. A moved document can leave a ghost
record pointing at a deleted path. A deleted file can leave a dangling metadata record.
A registered file can exist in git without its record, or vice versa.

### 3. Conflated concerns in the approval cascade

Approving a dev-plan document automatically cascades the owning feature from `dev-planning`
to `developing`, which is the signal to begin parallel implementation. This conflates two
distinct decisions:

- **Plan quality** — an implementation judgement the agent can make. Agents author dev-plans
  and can assess structural quality, spec traceability, and dependency soundness.
- **Implementation trigger** — a scheduling decision. Humans may need to consider timing,
  resource availability, cross-feature priorities, or external dependencies before work
  begins.

The coupling means agents cannot approve their own plans without accidentally starting
implementation, and humans must review implementation details they don't own just to
control scheduling.

### What happens if nothing changes

- Agents continue to orphan state on every session that doesn't use sub-agents.
- Document file operations remain a multi-step manual process prone to inconsistency.
- The dev-plan approval cascade continues to conflate quality and scheduling decisions.
- The `AGENTS.md` orphaned-state checklist item remains necessary indefinitely.

---

## Design

The design has three pillars, each independent and incrementally deliverable.

### Pillar A: Parameterised state commit

#### Current state

`git.CommitStateIfDirty(repoRoot)` stages all files under `.kbz/state/` and commits with
a fixed message: `"chore(kbz): persist workflow state before sub-agent dispatch"`. It is
called from exactly one site: the `handoff` tool handler.

#### Proposed change

Add a sibling function:

```
CommitStateWithMessage(repoRoot, message string) (committed bool, err error)
```

This function behaves identically to `CommitStateIfDirty` except the commit message is
caller-supplied. The existing `CommitStateIfDirty` becomes a thin wrapper that calls
`CommitStateWithMessage` with the fixed message, preserving backward compatibility.

Each state-mutating MCP tool handler calls `CommitStateWithMessage` at the end of its
successful execution path with a descriptive message:

| Tool | Action | Commit message pattern |
|------|--------|----------------------|
| `finish` | complete | `workflow(TASK-xxx): complete – {summary}` |
| `decompose` | apply | `workflow(FEAT-xxx): decompose into N tasks` |
| `doc` | approve | `workflow(DOC-xxx): approve {type}` |
| `doc` | register | `workflow(DOC-xxx): register {type}` |
| `merge` | execute | `workflow(FEAT-xxx): mark worktree merged` |
| `entity` | transition | `workflow(ID): transition {from} → {to}` |
| `entity` | create | `workflow(ID): create {type}` |

#### Commit semantics

- **Best-effort.** If the commit fails (e.g., git lock contention), the tool logs a
  warning and returns the normal result. The tool operation itself has already succeeded.
  This matches the existing `handoff` behaviour.
- **No-op when clean.** If `.kbz/state/` has no dirty files, no commit is created.
- **State-only scope.** Only files under `.kbz/state/` are staged. Code, documents, and
  other working files are never touched.
- **Idempotent.** Multiple tools called in sequence each attempt to commit. If a prior
  tool already committed the state, the next tool's commit is a no-op.

#### Interaction with `handoff`

`handoff` continues to call `CommitStateIfDirty` before sub-agent dispatch. If all state
has already been committed by individual tool calls, this is a no-op. If any state was
written by a tool that doesn't auto-commit (future tools, or a commit failure), `handoff`
catches it. The safety net remains.

#### Failure mode

If `CommitStateWithMessage` fails, the state files remain on disk as dirty files. They
will be caught by: (a) the next tool call's commit, (b) `handoff`'s pre-dispatch commit,
or (c) the session-start orphaned-state check. No state is lost. The failure mode is
identical to the current system, just with more frequent commit attempts.

### Pillar B: Atomic document file operations

#### Current state

The `doc` tool supports `register`, `approve`, `refresh`, `supersede`, `get`, `list`,
`validate`, and several other read actions. It does not support `move` or `delete`. It
does not commit any state changes or document files.

#### Foundation: `CommitStateAndPaths`

Add a new function:

```
CommitStateAndPaths(repoRoot, message string, extraPaths ...string) (committed bool, err error)
```

This function:

1. Stages all files under `.kbz/state/` (same as `CommitStateIfDirty`)
2. Stages each path in `extraPaths` (using `git add -- <path>` for each)
3. Creates a single commit with the supplied message
4. Returns `(false, nil)` if nothing was dirty

The `extraPaths` are explicitly provided by the caller — never globbed or discovered.
This ensures the function cannot accidentally stage unrelated files.

`CommitStateWithMessage` from Pillar A becomes a thin wrapper:
`CommitStateAndPaths(repoRoot, message)` with no extra paths.

#### New action: `doc move`

Parameters: `id` (required), `new_path` (required).

Behaviour:

1. Load the document record by ID
2. Verify the file exists at the current `path`
3. Move the file on disk (rename)
4. Update the record's `path` field in-place
5. If the new path is in a different document-type directory (e.g., `work/research/` →
   `work/reports/`), update the `type` field
6. Recompute the content hash from the new path
7. Write the updated record via `DocumentStore.Write()`
8. Call `CommitStateAndPaths(repoRoot, message, oldPath, newPath)` to commit the file
   move and state record atomically

The document ID, approval status, owner, and all cross-references are preserved.

#### New action: `doc delete`

Parameters: `id` (required), `force` (optional, default `false`).

Behaviour:

1. Load the document record by ID
2. If status is `approved` and `force` is not `true`, return an error explaining that
   approved documents cannot be deleted without `force: true`
3. Remove the file from disk
4. If the document has an `owner` entity, clear the entity's document reference field
   (e.g., `design`, `spec`, or `dev_plan`) via the entity lifecycle hook
5. Remove the state record file from `.kbz/state/documents/`
6. Call `CommitStateAndPaths(repoRoot, message, filePath)` to commit the file deletion,
   state record removal, and any entity field update atomically

#### Enhanced `doc register`

When `doc register` is called, after writing the state record, it calls
`CommitStateAndPaths(repoRoot, message, documentFilePath)` to commit both the document
file and its metadata record in a single commit.

#### Auto-refresh on `doc approve`

Currently, `doc approve` loads the stored content hash, reads the file, computes the
current hash, and rejects approval if they differ. The agent must call `doc refresh`
first.

Change: `doc approve` computes the current hash directly from the file and updates the
stored hash as part of the approval. The separate `refresh` step before approval is
eliminated. The `refresh` action remains available for other use cases (e.g., detecting
drift on `doc get`) but is no longer a prerequisite for approval.

#### Auto-approve flag on `doc register`

Add an optional `auto_approve` parameter to `doc register`. When `true`:

1. Register the document (write state record with current content hash)
2. Set status to `approved`, set `approved_by` and `approved_at`
3. Fire the entity lifecycle cascade **only for document types where it is safe**
   (see Pillar C for the dev-plan exception)
4. Commit atomically via `CommitStateAndPaths`

The `auto_approve` flag is opt-in per call. It does not change the default registration
behaviour.

Safe document types for auto-approve: `dev-plan` (with cascade decoupled per Pillar C),
`research`, `report`. Unsafe: `design`, `specification`.

### Pillar C: Decouple dev-plan approval from implementation trigger

#### Current state

In `internal/service/documents.go`, the approval cascade includes:

```
case entityType == "feature" && doc.Type == model.DocumentTypeDevPlan:
    targetStatus = "developing"
```

This means approving a dev-plan document automatically transitions the feature to
`developing`.

#### Proposed change

Remove the `DocumentTypeDevPlan` case from the approval cascade. Dev-plan approval marks
the document as `approved` but does **not** transition the owning feature.

The transition from `dev-planning` → `developing` becomes an explicit operation requiring
`entity(action: transition, id: "FEAT-xxx", status: "developing")`. The existing stage
gate checks remain enforced: the transition requires an approved dev-plan and at least one
child task.

This means the workflow becomes:

1. Agent writes dev-plan → agent approves (or auto-approves) the dev-plan
2. Agent decomposes the spec into tasks (can happen during `dev-planning`)
3. Human reviews readiness and says "go"
4. Agent calls `entity(action: transition, status: "developing")`
5. Implementation begins

#### What cascades are preserved

- Design approval → feature transitions to `specifying` (unchanged)
- Spec approval → feature transitions to `dev-planning` (unchanged)
- Plan design approval → plan transitions to `active` (unchanged)

Only the dev-plan cascade is removed. The `human_gate: true` on `dev-planning` in
`stage-bindings.yaml` is now enforced at the transition, where it belongs, rather than
at the document approval.

#### Failure mode

If the cascade removal is deployed but the stage-bindings or skill documents still tell
agents to expect the automatic transition, agents will write and approve the plan, then
wait for a transition that never happens. The skill documents (`write-dev-plan`,
`decompose-feature`, `orchestrate-development`) must be updated to reflect the new
workflow: approve plan → decompose → human says "go" → transition.

### Pillar D: Document section validation

#### Current state

`stage-bindings.yaml` declares `document_template.required_sections` for each stage that
produces a document. These requirements are not checked by any tool.

#### Proposed change

On `doc register` and `doc approve`, read the document file and check for markdown
headings (level 2: `## Section Name`) matching the required sections declared in
`stage-bindings.yaml` for the document's type.

- On `register`: return a structured warning listing missing sections. Registration
  proceeds regardless — this is a lint, not a gate.
- On `approve`: return an error rejecting approval if required sections are missing.
  This makes section completeness a hard gate for approval.

The mapping from document type to required sections is read from `stage-bindings.yaml`
at server startup and cached. The check is a simple case-insensitive heading scan — no
NLP or fuzzy matching.

---

## Alternatives Considered

### A. Commit on every state write (inside the store layer)

**Approach.** Move the commit call into `EntityStore.Write()` and `DocumentStore.Write()`
so every state file write triggers a commit.

**Trade-offs.**

- Makes it impossible to forget a commit — every write is immediately persisted.
- Creates excessive git commits for operations that write multiple state files. `finish`
  writes task + knowledge + retro + unblocked dependencies; this would produce 4+ commits
  for a single logical operation.
- `decompose apply` writes N tasks in two passes — committing after each write would
  commit incomplete dependency graphs.
- Destroys the "logical unit of work" property. Git history becomes noise.

**Rejected because:** the commit granularity is wrong. Commits should represent completed
operations, not individual file writes.

### B. Deferred commit with a session-end hook

**Approach.** Instead of committing after each tool call, register a shutdown hook or
session-end callback that commits all dirty state when the MCP session terminates.

**Trade-offs.**

- Fewer commits (one per session).
- Session-end detection is unreliable: MCP sessions can be killed, crash, or disconnect
  without a clean shutdown signal.
- If the session lasts hours and crashes midway, all state from the session is lost.
- No intermediate commit points for recovery.

**Rejected because:** the failure mode is worse than the current system. At least today,
`handoff` provides some intermediate commits. A session-end-only approach removes even
that safety net for non-sub-agent workflows.

### C. Keep the dev-plan cascade and add a `--no-cascade` flag to `doc approve`

**Approach.** Instead of removing the cascade, add an opt-out flag:
`doc(action: approve, no_cascade: true)`.

**Trade-offs.**

- Backward compatible — existing workflows continue to work.
- Requires agents to remember to pass the flag when approving dev-plans.
- Default behaviour remains "approval triggers implementation," which is the conflated
  concern we're trying to fix.
- Creates two modes for the same operation, increasing cognitive load.

**Rejected because:** the default should be safe. Requiring an opt-out flag to prevent
unintended side effects is the wrong default. The cascade should be removed entirely for
dev-plans, and the transition should be explicit.

### D. File-system watcher for document drift detection

**Approach.** Run a background goroutine that watches `work/` for file changes and
automatically calls `refresh` on any registered document whose file is modified.

**Trade-offs.**

- Eliminates the stale-hash problem entirely — hashes are always current.
- Adds runtime complexity: file-system watchers are platform-dependent, can miss events,
  and consume resources.
- The MCP server is a request-response tool, not a daemon. Adding background processes
  changes its operational model.
- Solves a problem that auto-refresh-on-approve already solves for the most important
  path (approval).

**Rejected because:** the complexity is disproportionate to the problem. Auto-refresh on
approve (Pillar B) handles the critical path, and `doc validate` can surface drift for
informational queries.

---

## Decisions

### D1: Commit at the tool handler level, not the store level

**Decision:** Each MCP tool handler calls `CommitStateWithMessage` at the end of its
successful execution path. The store layer (`EntityStore`, `DocumentStore`) does not
commit.

**Context:** The system needs more commit points, but operations like `finish` and
`decompose apply` write multiple state files as a single logical unit.

**Rationale:** Committing at the handler level preserves logical atomicity. The handler
knows when all writes for an operation are complete. The store layer does not — it only
sees individual file writes. Handler-level commits produce one commit per logical
operation (e.g., "complete task" not "write task file, write knowledge file, write retro
file").

**Consequences:** Every state-mutating handler must add a commit call. If a new handler
is added without a commit call, its state changes fall through to the `handoff` safety
net. This is a safe failure mode (state is eventually committed) but suboptimal
(generic commit message, delayed persistence).

### D2: Best-effort commits that do not block tool results

**Decision:** If `CommitStateWithMessage` fails, the tool handler logs a warning and
returns the normal result. The commit failure does not propagate as a tool error.

**Context:** Git operations can fail transiently (lock contention, disk full, permission
issues). The tool's primary function (state mutation) has already succeeded.

**Rationale:** The `handoff` tool already uses this pattern successfully. Blocking the
tool result on a commit failure would degrade the user experience for a transient
infrastructure issue. The state files are on disk and will be committed by the next
successful commit attempt.

**Consequences:** In rare failure cases, a tool may report success but the state is not
yet committed. The `handoff` safety net and session-start orphaned-state check catch
these cases. Agents see a warning in the log but are not blocked.

### D3: Remove the dev-plan approval cascade unconditionally

**Decision:** The `DocumentTypeDevPlan` case is removed from the approval cascade in
`documents.go`. There is no flag to re-enable it.

**Context:** The cascade conflates plan quality (agent domain) with implementation
scheduling (human domain). A flag-based approach was considered and rejected.

**Rationale:** The safe default is "approval does not start implementation." If a team
wants immediate cascading, they can call `entity(action: transition)` immediately after
`doc(action: approve)` — two explicit calls that make the intent clear. A flag that
re-enables the cascade preserves the conflation as the default path, which is the problem
we are solving.

**Consequences:** Agents and skills that expect the automatic cascade will need updating.
The `write-dev-plan`, `decompose-feature`, and `orchestrate-development` skills must be
revised to reflect the new workflow. Agents that call `doc(approve)` on a dev-plan will
no longer see the feature transition as a side effect — they must explicitly transition.

### D4: `CommitStateAndPaths` uses explicit path list, never globs

**Decision:** The `CommitStateAndPaths` function accepts an explicit list of extra file
paths. It never discovers paths via glob, directory scan, or pattern matching.

**Context:** The function stages files outside `.kbz/state/` (document files in `work/`).
Accidental staging of unrelated files would be a serious problem.

**Rationale:** Explicit paths eliminate the risk of staging work-in-progress code,
unrelated documents, or editor temp files. Every path staged is deliberately chosen by
the calling tool handler. The document path is already known to each handler (it is a
required parameter or stored on the record).

**Consequences:** If a tool handler forgets to include a path, that file is not committed.
This is visible in `git status` and easily corrected. The failure mode (missing file in
commit) is far less severe than the alternative (accidentally committed file).

### D5: Section validation is a lint on register, a gate on approve

**Decision:** Missing required sections produce a warning on `doc register` and an error
on `doc approve`.

**Context:** `stage-bindings.yaml` declares required sections for each document type.
Today no tool checks them.

**Rationale:** On registration, the document may be a work-in-progress — blocking
registration would force authors to write complete documents before they can track them.
A warning gives immediate feedback without blocking. On approval, the document should be
complete — missing sections at this point are a defect. Making it a hard gate prevents
approval of incomplete documents without requiring agent discipline.

**Consequences:** Documents with missing sections can be registered (with a warning) but
not approved. This may surface issues with existing registered documents that were never
checked. A migration pass to identify non-compliant documents should be considered.

### D6: Auto-approve is opt-in per call, scoped by document type

**Decision:** The `auto_approve` parameter on `doc register` is opt-in. The server
enforces a type whitelist: `dev-plan`, `research`, `report`. Attempting to auto-approve
a `design` or `specification` returns an error.

**Context:** Some document types require human judgement for approval. Others are
agent-authored and do not.

**Rationale:** A server-enforced whitelist is safer than trusting agents to use the flag
appropriately. The whitelist is small and stable — it maps directly to the `human_gate`
semantics in `stage-bindings.yaml`. Adding a new auto-approvable type requires a code
change, which is intentional friction.

**Consequences:** Agents that attempt to auto-approve a design or specification receive
a clear error. The whitelist may need updating if new document types are added, but this
is a deliberate design decision — new types should be explicitly classified.