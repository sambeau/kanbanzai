# Design: Agent Workflow Ergonomics (P34)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-25                    |
| Status | Draft                         |
| Author | Architect                     |
| Plan   | P34-agent-workflow-ergonomics |

---

## Related Work

**Corpus concepts searched:** task lifecycle, auto-promote, queued, idempotent claim,
plan prefix, decompose proposal, worktree context assembly.

**Prior decisions and designs found:**

| Document | Relevance |
|----------|-----------|
| P19 — Workflow lifecycle integrity | Introduced auto-advance for parent features when all child tasks reach terminal state. H-2 is the inverse: child tasks are not promoted when the parent feature advances. The same `OnStatusTransition` hook mechanism applies. |
| P21 — Codebase Memory Integration | Added `graph_project` to context assembly and the worktree record. `worktree_path` is already computed in `assembly.go` (`actx.worktreePath = wt.Path`) but never written to the response map. H-5 is a one-field omission. |
| P25 — Agent Tooling and Pipeline Quality | Addressed decompose reliability (empty names, dev-plan awareness). Did not change task granularity or the generic "Write tests" catch-all pattern. |
| P29 — State Store Read Path Performance | Fixed the O(n) file-scan root cause of `next()` timeouts. H-3 is the remaining UX issue: when a timeout still occurs, the tool returns an unrecoverable error rather than the context packet. |
| P7 — Developer Experience | Added `server_info`. Established the pattern of small quality-of-life additions to existing tools with no structural change. |

No directly related prior work was found on plan ID prefix shorthand resolution or
idempotent task claiming.

---

## Problem and Motivation

Six friction patterns were identified across P30–P33 agent session retrospectives,
each reported by at least two independent agents. At the current server maturity,
these issues are felt on every sprint rather than occasionally.

**H-1 — Shortened plan IDs rejected.** Humans and agents refer to plans as `P30`,
`P31`, etc. The tools `status()` and `entity()` require the full canonical form
(`P30-handoff-skill-assembly-prompt-hygiene`). `model.IsPlanID("P30")` returns
false because the validator requires a hyphen and slug. Short forms cause tool
errors or a manual lookup round-trip on every human-directed task.

**H-2 — Tasks start in `queued`, not `ready`.** When a feature transitions to
`developing`, its tasks remain in `queued` state. Tasks with no unmet dependencies
require a manual `entity(action: transition)` call before they can be claimed.
Three separate agents reported this as pure ceremony — the dependency
auto-promotion system (P19) handles parent-from-child advances but the reverse
(child promotion on parent advance) is not implemented.

**H-3 — Silent `next()` timeout leaves task unclearably active.** When
`next(id: TASK-...)` times out, the task is claimed server-side but the agent
receives no response. A retry fails with "already dispatched." P29 reduced timeout
frequency by eliminating O(n) file scans, but two agents hit the issue after P29
was deployed, confirming the UX problem is independent of performance. The
`active` status becomes a dead end with no recovery path other than manually
inspecting entity state.

**H-4 — Decompose proposals generate weak tasks.** `decompose(propose)` creates
one task per acceptance criterion (or a group of 2–4 ACs), then appends a single
generic "Write tests" task for the whole feature. This produces a lopsided task
set: multiple narrow implementation tasks sharing one vague test task with no
scope boundary. Agents bypass the tool entirely and use `entity(action: create)`
directly. The detour through propose → reformat → re-approve costs more than it
saves.

**H-5 — Worktree path not surfaced in context packet.** The `next()` claim
response includes `graph_project` from the active worktree record but not the
filesystem `path`. The path is already computed in `assembly.go`
(`actx.worktreePath = wt.Path`) but never written to the response. Agents must
call `worktree(action: get)` separately — a redundant round-trip for information
the server has already retrieved.

**M-1 — Multiple `decompose(apply)` runs accumulate stale tasks.** Each call
creates a new task set without cancelling the previous one. A feature that has
been decomposed three times carries three overlapping sets of `queued` tasks.
The dashboard shows all of them, making true progress invisible. In P32/FEAT-3,
12 stale tasks appeared alongside the real 13.

---

## Design

### §1 · Plan prefix shorthand resolution

`model.ParsePlanID` decomposes a full plan ID into `(prefix, number, slug)`.
A short reference like `P30` has a prefix and number but no slug, so
`IsPlanID("P30")` returns false.

**New predicate:** `model.ParseShortPlanRef(s string) (prefix, number string, ok bool)`
returns `ok = true` if `s` is exactly one uppercase letter followed by one or more
digits and nothing else (no hyphen, no trailing characters). This is a fast lexical
check with no I/O.

**New service method:** `EntityService.ResolvePlanByNumber(prefix, number string) (id, slug string, err error)`
calls the cache-backed `ListPlanIDs()` (O(1) after P29) and scans for the plan
whose `ParsePlanID` decomposition matches the given prefix and number. Sequential
plan numbers are unique; ambiguity cannot occur.

**Integration:** The `entity` MCP tool and the `status` MCP tool both resolve
input IDs before dispatching to the service layer. Both already call
`entitySvc.ResolvePrefix` for ULID prefixes. The same pre-resolution step is
extended: when `ParseShortPlanRef` succeeds on the input, call
`ResolvePlanByNumber` and substitute the full canonical ID before proceeding.

This is purely additive — all existing full-ID paths are unchanged.

---

### §2 · Task auto-promotion on `developing` transition

`EntityService.UpdateStatus` fires `s.statusHook.OnStatusTransition` after writing
the new status. The hook is already used for automatic worktree creation (added in
P19).

**New hook behaviour:** When a feature transitions to `developing`, call a new
service function `PromoteQueuedTasks(featureID string)`.

`PromoteQueuedTasks` algorithm:

1. List all tasks with `parent_feature == featureID` and `status == "queued"`.
2. For each task, inspect its `depends_on` list.
3. If `depends_on` is empty, **or** all entries are in a terminal status (`done`,
   `not-planned`, `cancelled`), transition the task `queued → ready` via
   `UpdateStatus`.
4. Skip tasks where any dependency is non-terminal; they will be promoted by the
   existing dependency auto-promotion logic when the dependency completes.

Individual transition failures are logged and do not abort the loop — partial
promotion is preferable to none.

**Idempotency:** `PromoteQueuedTasks` is safe to call multiple times. Tasks already
in `ready` or any other non-`queued` status are not touched.

---

### §3 · Idempotent task claim in `next()`

In `nextClaimMode`, the `status == "active"` branch currently returns a hard error.
The fix re-routes this branch to return the context packet instead.

**Behaviour change:** When `next(id: TASK-...)` is called for a task already in
`active` status:

1. Reload the task record (already done for the error path).
2. Call `assembleContext` with the task's state (identical to the normal claim
   path).
3. Return the standard claim response map with one additional boolean field:
   `"reclaimed": true`.
4. Do **not** re-fire the status transition hook or overwrite `claimed_at` /
   `dispatched_to` / `dispatched_by`. The existing dispatch metadata is surfaced
   unchanged in the response.

The error for a non-`ready`, non-`active` task (e.g. `queued`, `done`) is
unchanged — only the `active` case is affected.

---

### §4 · Worktree path in context packet

`assembleContext` already sets `actx.worktreePath = wt.Path` when a worktree
record exists for the parent feature. `nextContextToMap` does not include it.

**Change:** Add one conditional field to `nextContextToMap`:

```internal/mcp/next_tool.go
if actx.worktreePath != "" {
    out["worktree_path"] = actx.worktreePath
}
```

The field is omitted (not null, not empty string) when no worktree is active —
consistent with how `tool_hint` and other optional fields are handled in the same
function.

---

### §5 · Decompose proposal quality — paired test tasks

The current algorithm (lines 679–692, `decompose.go`) generates all implementation
tasks first, then appends a single catch-all "Write tests" task if none of the
generated tasks mentions testing. This produces poor task decomposition: one
unclaimed test obligation for an arbitrarily large scope.

**Change:** Replace the global test-task injection with per-unit paired test task
generation.

For each generated implementation task (whether a single-AC task or a 2–4 AC
grouped task), immediately append a sibling test task:

- **Slug:** `{impl-slug}-tests`
- **Name:** `Test {impl-name}`
- **Summary:** `Write tests covering: {impl-summary}`
- **DependsOn:** `[{impl-slug}]` (the test runs after its paired implementation)
- **Covers:** the same AC texts as the implementation task

The global "Write tests" catch-all injection is removed.

**Exception:** If an AC's text itself contains the word "test" (e.g. "Add a
regression test for …"), the criterion is already an explicit testing requirement.
The implementation task derived from it does not receive a paired test task — the
AC is itself the test work.

A feature with three implementation tasks will produce three paired test tasks
(total: six tasks) rather than four tasks (three implementation + one catch-all).

---

### §6 · Decompose apply — supersede prior queued task sets

When `decomposeApply` is called for a feature that already has tasks, it currently
adds the new task set to the existing one. The fix adds a supersession pass before
any task is created.

**Supersession algorithm** (inserted at the start of `decomposeApply`, before
Pass 1):

1. List all tasks for `featureID`.
2. Partition:
   - **Supersedable:** `status == "queued"` only.
   - **Protected:** all other statuses (`ready`, `active`, `needs-rework`,
     `needs-review`, `done`, `not-planned`, `cancelled`).
3. Transition all supersedable tasks to `not-planned` via `UpdateStatus`.
4. If any tasks are in `active` or `needs-rework` (in-progress work exists),
   add `"warning"` to the response: `"N task(s) in active/needs-rework status
   were preserved; verify they are still needed."` Do not block the new apply.
5. Proceed with Pass 1 (create new tasks) as before.
6. Add `"superseded_count": N` to the response map.

This makes the `decompose → apply` cycle idempotent: applying three times on the
same feature produces exactly one `queued` task set.

---

## Alternatives Considered

### H-1: Full ID required vs. prefix shorthand

**Alternative A (chosen) — Accept short form, resolve server-side.** `P30` is
resolved to the canonical full ID before the service call. Transparent to all
callers.
*Pros:* Natural syntax works everywhere without agent changes.
*Cons:* Adds a scan of the plan list at resolution time — O(1) after P29's cache.

**Alternative B — Add a `plan_lookup(short_ref)` tool.** Agents call
`plan_lookup("P30")` first, then use the result in subsequent calls.
*Rejected:* Forces an extra round-trip; agents will continue using `P30`
directly from human context and will keep hitting errors.

**Alternative C — Accept shorthand in `status` only, not `entity`.**
*Rejected:* Inconsistent rules across tools are harder to learn and cause
unexpected failures when agents switch tools.

---

### H-2: Task auto-promotion placement

**Alternative A (chosen) — Hook on `developing` feature transition.** Uses the
established `OnStatusTransition` hook; same code path as worktree auto-creation.
Promotes tasks at the earliest correct moment with no extra mechanism.

**Alternative B — Promote lazily at `next()` queue inspection time.** Delay until
an agent requests work.
*Rejected:* The agent's first `next()` call should return data immediately,
not trigger background work that may not complete before the response.

**Alternative C — Collapse `queued` and `ready` into one status.**
*Rejected:* `queued` (waiting for dependencies or enablement) is semantically
distinct from `ready` (claimable now). Merging them breaks the existing dependency
auto-promotion logic.

---

### H-3: Recovery from timeout claim

**Alternative A (chosen) — Return context packet for already-active tasks.** The
agent that timed out needs the context to proceed anyway. `reclaimed: true` is
sufficient disambiguation. One-branch change.

**Alternative B — Add `next(action: "reclaim", id: ...)`.** Explicit recovery
action on the tool.
*Rejected:* Extra API surface; the already-active case is exactly the state the
agent wants. Returning it unconditionally is simpler.

**Alternative C — Auto-revert active tasks to `ready` after N hours of
inactivity.** Stale task recovery.
*Rejected:* This is a separate concern (crash recovery, addressed by P13) and
should not be conflated with the timeout claim UX issue.

---

### H-4: Test task quality

**Alternative A (chosen) — Paired test task per implementation unit.** Each
implementation task gets a sibling with explicit dependency and matching scope.
Scope is unambiguous; tasks are parallelisable after the implementation lands.

**Alternative B — Keep the global task, improve its summary.** List covered ACs
in the "Write tests" summary.
*Rejected:* One task still cannot be parallelised and will be arbitrarily large
on a multi-AC feature. The scope problem is structural.

**Alternative C — Remove test task injection entirely.** Rely on ACs to drive
testing decisions.
*Rejected:* Evidence from prior sprints shows that without explicit test tasks,
agents omit test files from implementations. The injection rule exists for
demonstrated reasons.

---

### H-5: Worktree path omission

**Alternative A (chosen) — Add `worktree_path` to `nextContextToMap`.** One
conditional field; the value is already computed in `assembleContext`.

**Alternative B — Document that agents must call `worktree(action: get)`.** Add
guidance to SKILL.md.
*Rejected:* A redundant tool call for information already retrieved by the server
is avoidable waste.

---

### M-1: Decompose apply supersession

**Alternative A (chosen) — Auto-supersede `queued` tasks; protect all others.**
Zero friction for the common case (iterating on decomposition before work starts).
Active/done tasks are preserved with a warning.

**Alternative B — Require explicit `decompose(action: "reset", ...)` before
re-apply.**
*Rejected:* The common case should require no extra call. The protected/supersedable
partition already prevents the dangerous case (cancelling in-progress work).

**Alternative C — Tag tasks with a decompose revision; hide older revisions in
dashboards.**
*Rejected:* More complex implementation; transitioning superseded tasks to
`not-planned` achieves the same visual result with the existing status model.

---

## Decisions

**Decision 1 — `ParseShortPlanRef` is a separate predicate; resolution lives at
the MCP tool layer.**
*Context:* `ParsePlanID` is used throughout the codebase with an implicit
assumption that a valid plan ID always contains a slug. Extending it to handle
slug-less short refs would require all callers to handle the empty-slug case.
*Rationale:* A new, separate predicate leaves existing call sites unchanged. The
resolution step belongs at the MCP boundary, not in the model package.
*Consequences:* Two small functions instead of one; the boundary is clear and
the model package remains simple.

**Decision 2 — Task promotion runs in the existing `OnStatusTransition` hook,
synchronously.**
*Context:* The hook is the established extension point for lifecycle side effects.
Worktree auto-creation already uses it. With P29's SQLite cache, listing tasks for
a feature is O(1).
*Rationale:* Consistency with the existing hook pattern; no new mechanism
introduced. Synchronous execution is acceptable given cache-backed list performance.
*Consequences:* `PromoteQueuedTasks` runs in the same request as the feature
transition. On a feature with many tasks, this is still fast, but it is a real
side effect — any hook failure must be logged and not returned as a transition
error (best-effort, same as worktree creation).

**Decision 3 — `reclaimed: true` is a flag on the normal success response, not a
distinct error or response variant.**
*Context:* The claim response is a map consumed by multiple callers (agents,
tests). Adding a response variant would require callers to handle a new type.
*Rationale:* A boolean flag on the existing success shape is backward-compatible.
Callers that do not check `reclaimed` continue to work correctly.
*Consequences:* Callers cannot distinguish "first claim" from "re-claim" unless
they inspect the flag. This is intentional — from the agent's perspective,
both cases should result in the same action (proceed with the task).

**Decision 4 — Paired test tasks carry an explicit `DependsOn` on their
implementation sibling.**
*Context:* The proposal's dependency graph is used by `decomposeApply` to wire
`depends_on` fields on the created tasks. Explicit dependencies are more reliable
than implied ordering.
*Rationale:* An explicit dependency means the dependency graph is correct without
agent inference. Agents can parallelize unrelated implementation tasks while their
test tasks wait for the corresponding implementation.
*Consequences:* A feature with N implementation tasks produces 2N tasks in the
proposal. This better reflects actual work scope; the increase is not overhead but
previously hidden work.

**Decision 5 — Supersede only `queued` tasks; `ready` tasks are protected.**
*Context:* A task transitions from `queued` to `ready` either by explicit agent
action or by the new auto-promotion hook (§2). Either way, the transition
represents intent to work on the task.
*Rationale:* If an agent has promoted a task to `ready`, that is an explicit
act that should be respected. Superseding `ready` tasks would silently undo
deliberate agent state changes.
*Consequences:* An agent that manually promotes tasks and then re-decomposes
will see those tasks preserved. The warning message guides the agent to verify
whether they are still needed.