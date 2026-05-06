# Review Report: Commit Discipline Prompts (F3) — Cycle 2

| Field | Value |
|-------|-------|
| Feature | FEAT-01KQTNYN01ZF8 |
| Reviewer | reviewer-conformance |
| Date | 2026-05-06 |
| Verdict | approved |

## Findings Resolution

### BF-4: getting-started mirror
**Status: resolved**

Both `.agents/skills/kanbanzai-getting-started/SKILL.md` and `internal/kbzinit/skills/getting-started/SKILL.md` are identical in all substantive content. The only differences are two comment markers at the top of the kbzinit copy (`# kanbanzai-managed: true` and `# kanbanzai-version: dev`), which are kbzinit-specific metadata not present in the agent-facing copy.

The strengthened git status check is present and identical in both files:
- **Clean slate** section: requires `git status`, commits coherent changes, forbids stashing
- **Commit workflow state** section: explicitly calls out `.kbz/state/`, `.kbz/index/`, and `.kbz/context/` as versioned project state; mandates `git add .kbz/ && git commit` before new work; explicitly forbids stashing, discarding, or `.gitignore`-ing these files
- **Store Neglect** anti-pattern: reinforces the "commit immediately, do not stash or discard" rule
- **Resuming an in-flight batch** section: first step requires committing orphaned `.kbz/` changes

## Remaining Findings

None.

All five spec requirements verified:

| Requirement | Status | Evidence |
|-------------|--------|----------|
| REQ-001 (`state_modified: true` on mutation) | PASS | `SignalStateModified(ctx)` called in all mutation handlers; `buildResult` injects `"state_modified": true` into JSON responses |
| REQ-002 (absent on read-only) | PASS | Test `TestStateModified_ReadOnlyHandlerOmitsFlag` confirms flag is absent; `TestStateModified_ReadOnlyErrorOmitsFlag` confirms absent on read-only errors |
| REQ-003 (high-mutation tools) | PASS | `entity` (create, update, transition, bootstrap, close-out), `doc` (register, approve, supersede, move, delete, record_false_positive, refresh, import, evaluate), `knowledge` (contribute, confirm, retire, flag, update, promote, resolve), `finish` — all call `SignalStateModified(ctx)`. Coverage exceeds the minimum specified set. |
| REQ-004 (kanbanzai-agents rule) | PASS | Rule present in task lifecycle checklist: "If `state_modified: true` was in the last tool response: `git add .kbz/ && git commit -m "chore(state): commit workflow state changes"` before proceeding to the next task" |
| REQ-005 (strengthened git status check) | PASS | Both getting-started copies contain the "Commit workflow state" section with explicit `.kbz/state/`, `.kbz/index/`, `.kbz/context/` directives and the "do not stash, discard, or .gitignore" prohibition |
| REQ-006 (dual-write) | PASS | Both kanbanzai-getting-started and kanbanzai-agents skill files mirrored in `internal/kbzinit/skills/` |

### Minor observation (non-blocking)

`knowledge` `compact` and `prune` actions call `SignalMutation(ctx)` but not `SignalStateModified(ctx)` when operating in non-dry-run mode. These actions are outside REQ-003's explicit scope (which lists only contribute, confirm, retire for knowledge), so this is not a finding against the spec. It may be worth addressing in a follow-up pass.

## Verdict

**approved** — The sole blocking finding (BF-4) from cycle 1 is resolved: both getting-started SKILL.md files contain the identical strengthened `git status` check with explicit `.kbz/state/`, `.kbz/index/`, `.kbz/context/` coverage and the "no stash, no discard" prohibition. The `state_modified: true` flag is correctly set in all four high-mutation tools (`entity`, `doc`, `knowledge`, `finish`), with comprehensive unit test coverage for mutation, read-only, nil-result, and error-response cases. The kanbanzai-agents skill contains the required `state_modified` commit-discipline rule. All acceptance criteria (AC-001 through AC-006) are satisfied.
