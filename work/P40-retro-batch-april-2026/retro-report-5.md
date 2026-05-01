# Retro Report 5 — B38 Batch Conformance Review Resolution

**Session:** Resolving 4 blocking conformance gaps to close batch B38  
**Date:** 2026-04-30  
**Author:** Claude Sonnet 4.6

---

## What I was doing

Working through the B38 conformance review verdict: approve the review document, supersede B38-F8, close B38-F9 (already merged), rebase B38-F1 to resolve its merge conflict with main, then close the batch.

---

## What went well

**`merge(action: check)` is excellent.** It gave me a precise, named list of failing gates with actionable messages. I never had to guess why a merge was blocked — the output told me exactly what was missing and what to do. This is the best-designed tool in the workflow.

**`status(id: B38-...)` gave an immediate, accurate picture.** At a glance I could see 60/60 tasks, 100% progress, and the "ready to close" signal. This made the end-state validation instant.

**Entity transitions are fast and clear.** Superseding B38-F8 and closing B38-F9 with override were single tool calls. The override mechanism with `override_reason` feels like the right design — it forces you to articulate the justification without blocking legitimate cases.

**The rebase conflict was easy to reason about.** All conflicts were in `.kbz/state/` YAML files from a workflow commit (`transition dev-planning → developing`) that predated main's current state. The right resolution (take `--ours`) was immediately obvious because these files have clear semantics: main always has the authoritative current state.

**`cleanup(action: list)` and `cleanup(action: execute)` worked well together.** The list gave me a preview and execute cleaned up correctly once I force-deleted the squash-merged branches that git couldn't auto-detect as merged.

---

## What didn't go well

### 1. `review_report_exists` gate blocks without upfront guidance

When `merge(action: execute, override: true)` was called, the gate error said it "cannot be bypassed with override: true — a report must exist." That's a reasonable hard gate. But the first call to `merge(action: check)` only said the gate failed — it didn't indicate in advance which gates were override-bypassable and which weren't. I only discovered the hard block at execution time. The `check` output should mark gates as `hard` vs `soft` so I know what to prepare before attempting the merge.

### 2. Document path format requirements are opaque

When I tried to register the review report, I got: *"expected file to be in work/_project/ (type-only filename prefix requires _project folder)"*. There's no documentation of the path format in the error or in the tool description. I had to dig through `doc(action: get)` on existing review documents to reverse-engineer the convention (`work/{batch-slug}/{batch-prefix}-{display-id}-report-...md`). The `doc(action: register)` tool should either state the path constraints upfront or provide a `doc(action: validate_path)` helper.

### 3. MCP auto-commits and manual commits collide silently

When I called `doc(action: register, auto_approve: true)`, the MCP server automatically committed the file (`workflow(...): register report`). I then tried to amend my own commit to include the same file — which silently succeeded but added nothing, because the file was already committed. The result was a redundant `workflow(FEAT-01KQ7YQK6DDDA): register post-rebase conformance review report` commit that only contained index YAML, not the document itself. The MCP server's auto-commit behavior is useful but it needs to either: (a) make it obvious in the tool response that a commit was made, or (b) not auto-commit when the caller is managing commits manually.

### 4. Worktree tool has a 1:1 entity assumption that breaks with duplicates

`worktree(action: remove)` takes only `entity_id`. When there were two worktrees for the same entity (one created with the real ID, one accidentally created with the display-ID format), the tool had no way to target a specific one. I ended up using `git worktree remove --force` directly, which removed the physical directories but left the records as "active" in the state store. Those orphan records are invisible to cleanup and can't be removed through the workflow tools without knowing the worktree ID. The `worktree` tool should accept an optional `worktree_id` parameter to disambiguate.

### 5. Verification fields have no setter

The merge gate checks for `verification` and `verification_status` on features, but there's no tool call that sets these fields outside the `finish` task flow. For a feature that needed re-verification after a rebase (not a normal task completion), I had no choice but to override the gates. A lightweight `entity(action: update)` extension — or a dedicated `verify(entity_id, summary)` tool — would close this gap.

### 6. Batch close ceremony after a batch review feels redundant

After the batch conformance review was approved and all features were done, I still had to manually call `entity(action: transition, status: reviewing)` and then `entity(action: transition, status: done)`. The batch was already in `active` and had just completed a formal batch review cycle. Having to re-enter `reviewing` to reach `done` felt like process friction rather than meaningful gate. The `reviewing` state makes sense when transitioning from active development — it seems less necessary as a required hop after an already-completed batch review.

### 7. Ghost worktree from display-ID misuse persists

The worktree `WT-01KQEX7T3X5XD` was created with entity_id `"FEAT-01KQ7-JDT511BZ"` (the display-ID format with an embedded hyphen) rather than the canonical `"FEAT-01KQ7JDT511BZ"`. The `worktree(action: create)` tool accepted this without error. The resulting record points to a non-existent entity and can't be removed through normal workflow tools. Input validation on entity IDs at worktree creation would prevent this class of ghost record entirely.

---

## Suggestions

1. **`merge(action: check)` output** — add a `bypassable: bool` field to each gate result so the caller knows before attempting `execute` which gates are hard stops.

2. **`doc(action: register)` errors** — include the expected path pattern in the error message, or expose a `doc(action: infer_path, owner: ..., type: ...)` helper.

3. **MCP auto-commits** — either surface the commit SHA prominently in the tool response, or add a `no_commit: true` parameter for callers managing their own commit flow.

4. **`worktree(action: remove)`** — add an optional `worktree_id` parameter to target a specific record when multiple exist for the same entity.

5. **`worktree(action: create)` validation** — reject entity IDs that match the display-ID pattern (with embedded segment hyphen) rather than the canonical ULID form.

6. **Feature verification** — add a `verification` parameter to `entity(action: update)` so post-hoc verification (e.g. after a rebase) can be recorded without going through a full task completion flow.

7. **Batch close shortcut** — if a batch is in `active` and has an approved batch-level conformance review on record, allow a direct `active → done` transition (or at least make the `reviewing` hop a no-ceremony pass-through).
